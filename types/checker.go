package types

import (
	"fmt"
	"zumbra/ast"
)

type scope struct {
	parent *scope
	values map[string]*Type
}

func newScope(parent *scope) *scope {
	return &scope{
		parent: parent,
		values: make(map[string]*Type),
	}
}

func (s *scope) define(name string, t *Type) {
	s.values[name] = t
}

func (s *scope) assign(name string, t *Type) bool {
	for cur := s; cur != nil; cur = cur.parent {
		if _, ok := cur.values[name]; ok {
			cur.values[name] = t
			return true
		}
	}
	return false
}

func (s *scope) resolve(name string) (*Type, bool) {
	for cur := s; cur != nil; cur = cur.parent {
		if t, ok := cur.values[name]; ok {
			return t, true
		}
	}
	return nil, false
}

type Checker struct {
	global *scope
	scope  *scope
	errors []error
}

func NewChecker() *Checker {
	global := newScope(nil)
	return &Checker{
		global: global,
		scope:  global,
		errors: []error{},
	}
}

func (c *Checker) ResetForNextRun() {
	c.scope = c.global
	c.errors = nil
}

func (c *Checker) Check(program *ast.Program) []error {
	c.ResetForNextRun()

	for _, stmt := range program.Statements {
		c.checkStatement(stmt)
	}

	return c.errors
}

func (c *Checker) addError(err error) {
	if err != nil {
		c.errors = append(c.errors, err)
	}
}

func (c *Checker) pushScope() {
	c.scope = newScope(c.scope)
}

func (c *Checker) popScope() {
	if c.scope.parent != nil {
		c.scope = c.scope.parent
	}
}

func (c *Checker) checkStatement(stmt ast.Statement) {
	switch s := stmt.(type) {
	case *ast.VarStatement:
		var t *Type = Simple(Unknown)
		if s.Value != nil {
			t = c.inferExpression(s.Value)
		}
		if s.Name != nil {
			c.scope.define(s.Name.Value, t)
		}

	case *ast.AssignStatement:
		if s.Value != nil {
			valueType := c.inferExpression(s.Value)
			if s.Name != nil {
				if !c.scope.assign(s.Name.Value, valueType) {
					c.scope.define(s.Name.Value, valueType)
				}
			}
		}

	case *ast.ExpressionStatement:
		if s.Expression != nil {
			c.inferExpression(s.Expression)
		}

	case *ast.ReturnStatement:
		if s.ReturnValue != nil {
			c.inferExpression(s.ReturnValue)
		}
	}
}

func (c *Checker) inferExpression(exp ast.Expression) *Type {
	switch e := exp.(type) {
	case nil:
		return Simple(Unknown)

	case *ast.IntegerLiteral:
		return Simple(Int)

	case *ast.FloatLiteral:
		return Simple(Float)

	case *ast.StringLiteral:
		return Simple(String)

	case *ast.Boolean:
		return Simple(Bool)

	case *ast.Identifier:
		if t, ok := c.scope.resolve(e.Value); ok {
			return t
		}
		return Simple(Unknown)

	case *ast.ArrayLiteral:
		if len(e.Elements) == 0 {
			return ArrayOf(Simple(Unknown))
		}

		first := c.inferExpression(e.Elements[0])
		homogeneous := true

		for i := 1; i < len(e.Elements); i++ {
			next := c.inferExpression(e.Elements[i])

			if first.Kind == Unknown || next.Kind == Unknown {
				first = Simple(Unknown)
				continue
			}

			if !Same(first, next) {
				homogeneous = false
			}
		}

		if !homogeneous {
			c.addError(fmt.Errorf("array literal has mixed element types"))
			return ArrayOf(Simple(Unknown))
		}

		return ArrayOf(first)

	case *ast.PrefixExpression:
		if e.Right == nil {
			return Simple(Unknown)
		}
		return c.inferExpression(e.Right)

	case *ast.InfixExpression:
		left := c.inferExpression(e.Left)
		right := c.inferExpression(e.Right)

		switch e.Operator {
		case "+", "-", "*", "/":
			if left.Kind == Unknown || right.Kind == Unknown {
				return Simple(Unknown)
			}

			if IsNumeric(left) && IsNumeric(right) {
				if left.Kind == Float || right.Kind == Float {
					return Simple(Float)
				}
				return Simple(Int)
			}

			if e.Operator == "+" && left.Kind == String && right.Kind == String {
				return Simple(String)
			}

			c.addError(fmt.Errorf("invalid operands for %s: %s and %s", e.Operator, left.Kind, right.Kind))
			return Simple(Unknown)

		case "==", "!=", "<", ">", "<=", ">=":
			if left.Kind == Unknown || right.Kind == Unknown {
				return Simple(Bool)
			}

			if left.Kind == right.Kind {
				return Simple(Bool)
			}

			if IsNumeric(left) && IsNumeric(right) {
				return Simple(Bool)
			}

			c.addError(fmt.Errorf("cannot compare %s with %s", left.Kind, right.Kind))
			return Simple(Bool)
		}

		return Simple(Unknown)

	case *ast.FunctionLiteral:
		c.pushScope()

		params := make([]*Type, 0, len(e.Parameters))
		for _, p := range e.Parameters {
			if p == nil {
				continue
			}
			pt := Simple(Unknown)
			c.scope.define(p.Value, pt)
			params = append(params, pt)
		}

		if e.Body != nil {
			for _, stmt := range e.Body.Statements {
				c.checkStatement(stmt)
			}
		}

		c.popScope()
		return FuncOf(params, Simple(Unknown))

	case *ast.CallExpression:
		fnType := Simple(Unknown)
		if e.Function != nil {
			fnType = c.inferExpression(e.Function)
		}

		for _, arg := range e.Arguments {
			c.inferExpression(arg)
		}

		if fnType.Kind == Func {
			if fnType.Return != nil {
				return fnType.Return
			}
			return Simple(Unknown)
		}

		return Simple(Unknown)

	case *ast.IndexExpression:
		left := c.inferExpression(e.Left)
		if left.Kind == Array && left.Elem != nil {
			return left.Elem
		}
		return Simple(Unknown)

	case *ast.IfExpression:
		if e.Condition != nil {
			c.inferExpression(e.Condition)
		}

		c.pushScope()
		if e.Consequence != nil {
			for _, stmt := range e.Consequence.Statements {
				c.checkStatement(stmt)
			}
		}
		c.popScope()

		if e.Alternative != nil {
			c.pushScope()
			for _, stmt := range e.Alternative.Statements {
				c.checkStatement(stmt)
			}
			c.popScope()
		}

		return Simple(Unknown)

	case *ast.AwaitExpression:
		if e.Value != nil {
			return c.inferExpression(e.Value)
		}
		return Simple(Unknown)

	case *ast.TryExpression:
		if e.Value != nil {
			return c.inferExpression(e.Value)
		}
		return Simple(Unknown)

	case *ast.AttributeAccess:
		return Simple(Unknown)

	case *ast.DictLiteral:
		return Simple(Unknown)
	}

	return Simple(Unknown)
}
