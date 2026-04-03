package compiler

import (
	"fmt"
	"zumbra/ast"
)

type ArityValidator struct {
	functionScopes []map[string]int
	errors         []Diagnostic
}

func NewArityValidator() *ArityValidator {
	return &ArityValidator{
		functionScopes: []map[string]int{{}},
		errors:         []Diagnostic{},
	}
}

func ValidateProgramArity(program *ast.Program) []Diagnostic {
	v := NewArityValidator()
	v.visitProgram(program)
	return v.errors
}

func (v *ArityValidator) visitProgram(program *ast.Program) {
	if program == nil {
		return
	}
	v.visitStatementList(program.Statements)
}

func (v *ArityValidator) pushScope() {
	v.functionScopes = append(v.functionScopes, map[string]int{})
}

func (v *ArityValidator) popScope() {
	if len(v.functionScopes) > 1 {
		v.functionScopes = v.functionScopes[:len(v.functionScopes)-1]
	}
}

func (v *ArityValidator) defineFunction(name string, arity int) {
	scope := v.functionScopes[len(v.functionScopes)-1]
	scope[name] = arity
}

func (v *ArityValidator) resolveFunction(name string) (int, bool) {
	for i := len(v.functionScopes) - 1; i >= 0; i-- {
		if arity, ok := v.functionScopes[i][name]; ok {
			return arity, true
		}
	}
	return 0, false
}

func (v *ArityValidator) visitStatementList(stmts []ast.Statement) {
	for _, stmt := range stmts {
		v.visitStatement(stmt)
	}
}

func (v *ArityValidator) visitStatement(stmt ast.Statement) {
	switch node := stmt.(type) {
	case *ast.VarStatement:
		if node.Name != nil {
			if fn, ok := node.Value.(*ast.FunctionLiteral); ok {
				v.defineFunction(node.Name.Value, len(fn.Parameters))
			}
		}
		if node.Value != nil {
			v.visitExpression(node.Value)
		}

	case *ast.AssignStatement:
		if node.Value != nil {
			v.visitExpression(node.Value)
		}

	case *ast.ReturnStatement:
		if node.ReturnValue != nil {
			v.visitExpression(node.ReturnValue)
		}

	case *ast.ExpressionStatement:
		if node.Expression != nil {
			v.visitExpression(node.Expression)
		}

	case *ast.WhileStatement:
		if node.Condition != nil {
			v.visitExpression(node.Condition)
		}
		if node.Body != nil {
			v.pushScope()
			v.visitStatementList(node.Body.Statements)
			v.popScope()
		}

	case *ast.ImportStatement:
		return
	}
}

func (v *ArityValidator) visitExpression(expr ast.Expression) {
	switch node := expr.(type) {
	case *ast.Identifier, *ast.IntegerLiteral, *ast.FloatLiteral, *ast.Boolean, *ast.StringLiteral:
		return

	case *ast.PrefixExpression:
		if node.Right != nil {
			v.visitExpression(node.Right)
		}

	case *ast.InfixExpression:
		if node.Left != nil {
			v.visitExpression(node.Left)
		}
		if node.Right != nil {
			v.visitExpression(node.Right)
		}

	case *ast.IfExpression:
		if node.Condition != nil {
			v.visitExpression(node.Condition)
		}
		if node.Consequence != nil {
			v.pushScope()
			v.visitStatementList(node.Consequence.Statements)
			v.popScope()
		}
		if node.Alternative != nil {
			v.pushScope()
			v.visitStatementList(node.Alternative.Statements)
			v.popScope()
		}

	case *ast.FunctionLiteral:
		v.pushScope()
		if node.Body != nil {
			v.visitStatementList(node.Body.Statements)
		}
		v.popScope()

	case *ast.CallExpression:
		switch fn := node.Function.(type) {
		case *ast.Identifier:
			if expected, ok := v.resolveFunction(fn.Value); ok {
				got := len(node.Arguments)
				if expected != got {
					v.errors = append(v.errors, Diagnostic{
						Severity: DiagnosticError,
						Message:  fmt.Sprintf("wrong number of arguments for %s: want=%d, got=%d", fn.Value, expected, got),
					})
				}
			}
		case *ast.FunctionLiteral:
			expected := len(fn.Parameters)
			got := len(node.Arguments)
			if expected != got {
				v.errors = append(v.errors, Diagnostic{
					Severity: DiagnosticError,
					Message:  fmt.Sprintf("wrong number of arguments for anonymous function: want=%d, got=%d", expected, got),
				})
			}
			v.visitExpression(fn)
		default:
			v.visitExpression(node.Function)
		}

		for _, arg := range node.Arguments {
			v.visitExpression(arg)
		}

	case *ast.ArrayLiteral:
		for _, el := range node.Elements {
			v.visitExpression(el)
		}

	case *ast.DictLiteral:
		for k, val := range node.Pairs {
			v.visitExpression(k)
			v.visitExpression(val)
		}

	case *ast.IndexExpression:
		if node.Left != nil {
			v.visitExpression(node.Left)
		}
		if node.Index != nil {
			v.visitExpression(node.Index)
		}

	case *ast.AttributeAccess:
		if node.Object != nil {
			v.visitExpression(node.Object)
		}

	case *ast.AwaitExpression:
		if node.Value != nil {
			v.visitExpression(node.Value)
		}

	case *ast.TryExpression:
		if node.Value != nil {
			v.visitExpression(node.Value)
		}

	case *ast.ErrorHandlerExpression:
		if node.Left != nil {
			v.visitExpression(node.Left)
		}
		if node.Handler != nil {
			v.pushScope()
			v.visitStatementList(node.Handler.Statements)
			v.popScope()
		}

	case *ast.ForEverLoop:
		if node.Block != nil {
			v.pushScope()
			v.visitStatementList(node.Block.Statements)
			v.popScope()
		}

	case *ast.ForEachDotRange:
		if node.StartIdx != nil {
			v.visitExpression(node.StartIdx)
		}
		if node.EndIdx != nil {
			v.visitExpression(node.EndIdx)
		}
		if node.Cond != nil {
			v.visitExpression(node.Cond)
		}
		if node.Block != nil {
			v.pushScope()
			v.visitStatementList(node.Block.Statements)
			v.popScope()
		}

	case *ast.ForEachArrayLoop:
		if node.Value != nil {
			v.visitExpression(node.Value)
		}
		if node.Cond != nil {
			v.visitExpression(node.Cond)
		}
		if node.Block != nil {
			v.pushScope()
			v.visitStatementList(node.Block.Statements)
			v.popScope()
		}

	case *ast.ForEachMapLoop:
		if node.X != nil {
			v.visitExpression(node.X)
		}
		if node.Cond != nil {
			v.visitExpression(node.Cond)
		}
		if node.Block != nil {
			v.pushScope()
			v.visitStatementList(node.Block.Statements)
			v.popScope()
		}

	case *ast.BreakExpression, *ast.ContinueExpression:
		return
	}
}
