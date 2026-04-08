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
	c := &Checker{
		global: global,
		scope:  global,
		errors: []error{},
	}
	c.installBuiltins()
	return c
}

func (c *Checker) installBuiltins() {
	names := []string{
		"show",
		"input",
		"sizeOf",
		"toInt",
		"toFloat",
		"toString",
		"toBool",
		"first",
		"last",
	}

	for _, name := range names {
		if t, ok := builtinType(name); ok {
			c.global.define(name, t)
		}
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

func isIdentifierUnknownParam(expr ast.Expression) (*ast.Identifier, bool) {
	id, ok := expr.(*ast.Identifier)
	if !ok || id == nil {
		return nil, false
	}
	return id, true
}

func mergeTypes(current *Type, wanted *Type) *Type {
	if current == nil || current.Kind == Unknown {
		return wanted
	}
	if wanted == nil || wanted.Kind == Unknown {
		return current
	}
	if Same(current, wanted) {
		return current
	}
	return Simple(Unknown)
}

func (c *Checker) constrainIdentifier(expr ast.Expression, wanted *Type) {
	id, ok := isIdentifierUnknownParam(expr)
	if !ok || wanted == nil {
		return
	}

	current, exists := c.scope.resolve(id.Value)
	if !exists || current == nil {
		return
	}

	if current.Kind == Unknown {
		_ = c.scope.assign(id.Value, wanted)
		return
	}

	merged := mergeTypes(current, wanted)
	_ = c.scope.assign(id.Value, merged)
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

	case *ast.WhileStatement:
		c.checkWhileStatement(s)
	}
}

func (c *Checker) checkWhileStatement(stmt *ast.WhileStatement) {
	if stmt == nil {
		return
	}

	if stmt.Condition != nil {
		condType := c.inferExpression(stmt.Condition)
		if condType.Kind != Unknown && condType.Kind != Bool {
			c.addError(fmt.Errorf("while condition must be bool, got %s", condType.Kind))
		}
	}

	if stmt.Body != nil {
		c.pushScope()
		for _, inner := range stmt.Body.Statements {
			c.checkStatement(inner)
		}
		c.popScope()
	}
}

func (c *Checker) inferBlockReturnType(block *ast.BlockStatement) *Type {
	if block == nil {
		return Simple(Unknown)
	}

	var inferred *Type
	for _, stmt := range block.Statements {
		switch s := stmt.(type) {
		case *ast.ReturnStatement:
			current := Simple(Null)
			if s.ReturnValue != nil {
				current = c.inferExpression(s.ReturnValue)
			}

			if inferred == nil {
				inferred = current
				continue
			}

			if inferred.Kind == Unknown || current.Kind == Unknown {
				inferred = Simple(Unknown)
				continue
			}

			if !Same(inferred, current) {
				c.addError(fmt.Errorf("function has conflicting return types: %s and %s", inferred.Kind, current.Kind))
				inferred = Simple(Unknown)
			}

		default:
			c.checkStatement(stmt)
		}
	}

	if inferred == nil {
		return Simple(Null)
	}

	return inferred
}

func (c *Checker) checkBuiltinCall(name string, args []ast.Expression) *Type {
	switch name {
	case "show":
		if len(args) != 1 {
			c.addError(fmt.Errorf("show expects 1 argument, got %d", len(args)))
			return Simple(Null)
		}
		c.inferExpression(args[0])
		return Simple(Null)

	case "input":
		if len(args) > 1 {
			c.addError(fmt.Errorf("input expects 0 or 1 arguments, got %d", len(args)))
			return Simple(String)
		}
		if len(args) == 1 {
			c.inferExpression(args[0])
		}
		return Simple(String)

	case "sizeOf":
		if len(args) != 1 {
			c.addError(fmt.Errorf("sizeOf expects 1 argument, got %d", len(args)))
			return Simple(Int)
		}
		argType := c.inferExpression(args[0])
		if argType.Kind != Unknown && argType.Kind != Array && argType.Kind != String {
			c.addError(fmt.Errorf("sizeOf expects array or string, got %s", argType.Kind))
		}
		return Simple(Int)

	case "toInt":
		if len(args) != 1 {
			c.addError(fmt.Errorf("toInt expects 1 argument, got %d", len(args)))
			return Simple(Int)
		}
		c.inferExpression(args[0])
		return Simple(Int)

	case "toFloat":
		if len(args) != 1 {
			c.addError(fmt.Errorf("toFloat expects 1 argument, got %d", len(args)))
			return Simple(Float)
		}
		c.inferExpression(args[0])
		return Simple(Float)

	case "toString":
		if len(args) != 1 {
			c.addError(fmt.Errorf("toString expects 1 argument, got %d", len(args)))
			return Simple(String)
		}
		c.inferExpression(args[0])
		return Simple(String)

	case "toBool":
		if len(args) != 1 {
			c.addError(fmt.Errorf("toBool expects 1 argument, got %d", len(args)))
			return Simple(Bool)
		}
		c.inferExpression(args[0])
		return Simple(Bool)

	case "first", "last":
		if len(args) != 1 {
			c.addError(fmt.Errorf("%s expects 1 argument, got %d", name, len(args)))
			return Simple(Unknown)
		}
		argType := c.inferExpression(args[0])
		if argType.Kind == Array {
			if argType.Elem != nil {
				return argType.Elem
			}
			return Simple(Unknown)
		}
		if argType.Kind != Unknown {
			c.addError(fmt.Errorf("%s expects array, got %s", name, argType.Kind))
		}
		return Simple(Unknown)
	}

	for _, arg := range args {
		c.inferExpression(arg)
	}
	return Simple(Unknown)
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
		if t, ok := builtinType(e.Value); ok {
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
			// tenta restringir parâmetros unknown com base no contexto do operador
			if e.Operator == "+" {
				if left.Kind == String && right.Kind == Unknown {
					c.constrainIdentifier(e.Right, Simple(String))
					right = c.inferExpression(e.Right)
				}
				if right.Kind == String && left.Kind == Unknown {
					c.constrainIdentifier(e.Left, Simple(String))
					left = c.inferExpression(e.Left)
				}
			}

			if IsNumeric(left) && right.Kind == Unknown {
				c.constrainIdentifier(e.Right, left)
				right = c.inferExpression(e.Right)
			}
			if IsNumeric(right) && left.Kind == Unknown {
				c.constrainIdentifier(e.Left, right)
				left = c.inferExpression(e.Left)
			}

			// se ambos ainda forem unknown, não erra cedo
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
			if left.Kind == Unknown && right.Kind != Unknown {
				c.constrainIdentifier(e.Left, right)
				left = c.inferExpression(e.Left)
			}
			if right.Kind == Unknown && left.Kind != Unknown {
				c.constrainIdentifier(e.Right, left)
				right = c.inferExpression(e.Right)
			}

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

		ret := Simple(Null)
		if e.Body != nil {
			ret = c.inferBlockReturnType(e.Body)
		}

		// captura tipos possivelmente refinados dos parâmetros após analisar o corpo
		refinedParams := make([]*Type, 0, len(e.Parameters))
		for _, p := range e.Parameters {
			if p == nil {
				continue
			}
			if pt, ok := c.scope.resolve(p.Value); ok {
				refinedParams = append(refinedParams, pt)
			} else {
				refinedParams = append(refinedParams, Simple(Unknown))
			}
		}

		c.popScope()
		return FuncOf(refinedParams, ret)

	case *ast.CallExpression:
		if ident, ok := e.Function.(*ast.Identifier); ok {
			if _, builtin := builtinType(ident.Value); builtin {
				return c.checkBuiltinCall(ident.Value, e.Arguments)
			}
		}

		fnType := Simple(Unknown)
		if e.Function != nil {
			fnType = c.inferExpression(e.Function)
		}

		argTypes := make([]*Type, 0, len(e.Arguments))
		for _, arg := range e.Arguments {
			argTypes = append(argTypes, c.inferExpression(arg))
		}

		if fnType.Kind == Func {
			if len(fnType.Params) != len(e.Arguments) {
				c.addError(fmt.Errorf("function expects %d arguments, got %d", len(fnType.Params), len(e.Arguments)))
			} else {
				for i := range fnType.Params {
					paramType := fnType.Params[i]
					argType := argTypes[i]

					if paramType == nil || argType == nil {
						continue
					}
					if paramType.Kind == Unknown || argType.Kind == Unknown {
						continue
					}
					if !Same(paramType, argType) {
						c.addError(fmt.Errorf("argument %d expects %s, got %s", i+1, paramType.Kind, argType.Kind))
					}
				}
			}

			if fnType.Return != nil {
				return fnType.Return
			}
			return Simple(Unknown)
		}

		return Simple(Unknown)

	case *ast.IndexExpression:
		left := c.inferExpression(e.Left)
		index := c.inferExpression(e.Index)

		if index.Kind != Unknown && index.Kind != Int {
			c.addError(fmt.Errorf("array index must be int, got %s", index.Kind))
		}

		if left.Kind == Array && left.Elem != nil {
			return left.Elem
		}
		return Simple(Unknown)

	case *ast.IfExpression:
		if e.Condition != nil {
			condType := c.inferExpression(e.Condition)
			if condType.Kind != Unknown && condType.Kind != Bool {
				c.addError(fmt.Errorf("if condition must be bool, got %s", condType.Kind))
			}
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
