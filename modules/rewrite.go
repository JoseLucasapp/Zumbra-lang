package modules

import (
	"fmt"
	"strings"

	"zumbra/ast"
)

type rewriter struct {
	unit        *unitState
	diagnostics *[]Diagnostic
	scopes      []map[string]bool
}

func newRewriter(unit *unitState, diagnostics *[]Diagnostic) *rewriter {
	return &rewriter{unit: unit, diagnostics: diagnostics, scopes: []map[string]bool{{}}}
}

func (r *rewriter) push() { r.scopes = append(r.scopes, map[string]bool{}) }
func (r *rewriter) pop()  { r.scopes = r.scopes[:len(r.scopes)-1] }
func (r *rewriter) local(name string) {
	if name != "" {
		r.scopes[len(r.scopes)-1][name] = true
	}
}
func (r *rewriter) isLocal(name string) bool {
	for i := len(r.scopes) - 1; i >= 0; i-- {
		if r.scopes[i][name] {
			return true
		}
	}
	return false
}

func (r *rewriter) mapped(name string) string {
	if r.isLocal(name) {
		return name
	}
	if internal, ok := r.unit.Symbols[name]; ok {
		return internal
	}
	if internal, ok := r.unit.legacyNames[name]; ok {
		return internal
	}
	return name
}

func renameIdentifier(identifier *ast.Identifier, value string) {
	if identifier == nil {
		return
	}
	identifier.Value = value
	identifier.Token.Literal = value
}

func (r *rewriter) statement(statement ast.Statement, topLevel bool) ast.Statement {
	switch s := statement.(type) {
	case *ast.VarStatement:
		original := ""
		if s.Name != nil {
			original = s.Name.Value
		}
		s.Value = r.expression(s.Value)
		if topLevel {
			renameIdentifier(s.Name, r.mapped(original))
		} else {
			r.local(original)
		}
		if fn, ok := s.Value.(*ast.FunctionLiteral); ok && s.Name != nil {
			fn.Name = s.Name.Value
		}
		return s
	case *ast.ConstStatement:
		original := ""
		if s.Name != nil {
			original = s.Name.Value
		}
		s.Value = r.expression(s.Value)
		if topLevel {
			renameIdentifier(s.Name, r.mapped(original))
		} else {
			r.local(original)
		}
		return s
	case *ast.AssignStatement:
		if s.Name != nil {
			renameIdentifier(s.Name, r.mapped(s.Name.Value))
		}
		s.Value = r.expression(s.Value)
		return s
	case *ast.AttributeAssignStatement:
		if s.Target != nil {
			if alias, ok := s.Target.Object.(*ast.Identifier); ok && r.unit.importAliases[alias.Value] != nil {
				r.error(fmt.Sprintf("cannot assign to imported module member %s.%s", alias.Value, s.Target.Property.Value))
			}
			s.Target.Object = r.expression(s.Target.Object)
		}
		s.Value = r.expression(s.Value)
		return s
	case *ast.IndexAssignStatement:
		if s.Target != nil {
			s.Target.Left = r.expression(s.Target.Left)
			s.Target.Index = r.expression(s.Target.Index)
		}
		s.Value = r.expression(s.Value)
		return s
	case *ast.ExpressionStatement:
		s.Expression = r.expression(s.Expression)
		return s
	case *ast.ReturnStatement:
		s.ReturnValue = r.expression(s.ReturnValue)
		return s
	case *ast.WhileStatement:
		s.Condition = r.expression(s.Condition)
		s.Body = r.block(s.Body)
		return s
	case *ast.TypeAliasStatement:
		original := s.Name.Value
		renameIdentifier(s.Name, r.mapped(original))
		if s.Target != nil {
			renameIdentifier(s.Target, r.typeName(s.Target.Value))
		}
		return s
	case *ast.StructStatement:
		original := s.Name.Value
		renameIdentifier(s.Name, r.mapped(original))
		for _, field := range s.Fields {
			field.TypeName = r.typeName(field.TypeName)
		}
		for _, method := range s.Methods {
			if method == nil || method.Function == nil {
				continue
			}
			method.Function = r.function(method.Function)
		}
		return s
	case *ast.EnumStatement:
		original := s.Name.Value
		renameIdentifier(s.Name, r.mapped(original))
		return s
	case *ast.ExternBlockStatement:
		for _, fn := range s.Functions {
			if fn == nil || fn.Name == nil {
				continue
			}
			original := fn.Name.Value
			renameIdentifier(fn.Name, r.mapped(original))
		}
		return s
	case *ast.UnsafeStatement:
		s.Body = r.block(s.Body)
		return s
	default:
		return statement
	}
}

func (r *rewriter) block(block *ast.BlockStatement) *ast.BlockStatement {
	if block == nil {
		return nil
	}
	r.push()
	for index, statement := range block.Statements {
		block.Statements[index] = r.statement(statement, false)
	}
	r.pop()
	return block
}

func (r *rewriter) function(fn *ast.FunctionLiteral) *ast.FunctionLiteral {
	if fn == nil {
		return nil
	}
	r.push()
	for _, parameter := range fn.Parameters {
		if parameter != nil {
			r.local(parameter.Value)
		}
	}
	if fn.Body != nil {
		for index, statement := range fn.Body.Statements {
			fn.Body.Statements[index] = r.statement(statement, false)
		}
	}
	r.pop()
	return fn
}

func (r *rewriter) expression(expression ast.Expression) ast.Expression {
	if expression == nil {
		return nil
	}
	switch e := expression.(type) {
	case *ast.Identifier:
		if _, alias := r.unit.importAliases[e.Value]; alias {
			r.error(fmt.Sprintf("module alias %q must be followed by a public member", e.Value))
			return e
		}
		renameIdentifier(e, r.mapped(e.Value))
		return e
	case *ast.FunctionLiteral:
		return r.function(e)
	case *ast.BlockStatement:
		return r.block(e)
	case *ast.AttributeAccess:
		if alias, ok := e.Object.(*ast.Identifier); ok {
			if imported := r.unit.importAliases[alias.Value]; imported != nil {
				internal, exported := imported.Exports[e.Property.Value]
				if !exported {
					if _, exists := imported.Symbols[e.Property.Value]; exists {
						r.error(fmt.Sprintf("%s.%s is private", alias.Value, e.Property.Value))
					} else {
						r.error(fmt.Sprintf("module %s has no public member %q", alias.Value, e.Property.Value))
					}
					return e
				}
				replacement := &ast.Identifier{Token: e.Property.Token, Value: internal}
				replacement.Token.Literal = internal
				return replacement
			}
		}
		e.Object = r.expression(e.Object)
		return e
	case *ast.DotExpression:
		if alias, ok := e.Left.(*ast.Identifier); ok {
			if imported := r.unit.importAliases[alias.Value]; imported != nil {
				internal, exported := imported.Exports[e.Right.Value]
				if !exported {
					r.error(fmt.Sprintf("%s.%s is not public", alias.Value, e.Right.Value))
					return e
				}
				replacement := &ast.Identifier{Token: e.Right.Token, Value: internal}
				replacement.Token.Literal = internal
				return replacement
			}
		}
		e.Left = r.expression(e.Left)
		return e
	case *ast.CallExpression:
		e.Function = r.expression(e.Function)
		for i, arg := range e.Arguments {
			e.Arguments[i] = r.expression(arg)
		}
		return e
	case *ast.ArrayLiteral:
		for i, item := range e.Elements {
			e.Elements[i] = r.expression(item)
		}
		return e
	case *ast.DictLiteral:
		pairs := make(map[ast.Expression]ast.Expression, len(e.Pairs))
		for key, value := range e.Pairs {
			pairs[r.expression(key)] = r.expression(value)
		}
		e.Pairs = pairs
		return e
	case *ast.IndexExpression:
		e.Left = r.expression(e.Left)
		e.Index = r.expression(e.Index)
		return e
	case *ast.InfixExpression:
		e.Left = r.expression(e.Left)
		e.Right = r.expression(e.Right)
		return e
	case *ast.PrefixExpression:
		e.Right = r.expression(e.Right)
		return e
	case *ast.IfExpression:
		e.Condition = r.expression(e.Condition)
		e.Consequence = r.block(e.Consequence)
		e.Alternative = r.block(e.Alternative)
		return e
	case *ast.MatchExpression:
		e.Value = r.expression(e.Value)
		for _, candidate := range e.Cases {
			candidate.Pattern = r.expression(candidate.Pattern)
			candidate.Body = r.block(candidate.Body)
		}
		e.Default = r.block(e.Default)
		return e
	case *ast.AwaitExpression:
		e.Value = r.expression(e.Value)
		return e
	case *ast.TryExpression:
		e.Value = r.expression(e.Value)
		return e
	case *ast.ErrorHandlerExpression:
		e.Left = r.expression(e.Left)
		r.push()
		if e.ErrorIdent != nil {
			r.local(e.ErrorIdent.Value)
		}
		if e.Handler != nil {
			for i, st := range e.Handler.Statements {
				e.Handler.Statements[i] = r.statement(st, false)
			}
		}
		r.pop()
		return e
	case *ast.ForLoop:
		r.push()
		e.Init = r.expression(e.Init)
		e.Cond = r.expression(e.Cond)
		e.Update = r.expression(e.Update)
		e.Block = r.block(e.Block)
		r.pop()
		return e
	case *ast.ForEachArrayLoop:
		e.Value = r.expression(e.Value)
		r.push()
		r.local(e.Var)
		e.Cond = r.expression(e.Cond)
		e.Block = r.block(e.Block)
		r.pop()
		return e
	case *ast.ForEachMapLoop:
		e.X = r.expression(e.X)
		r.push()
		r.local(e.Key)
		r.local(e.Value)
		e.Cond = r.expression(e.Cond)
		e.Block = r.block(e.Block)
		r.pop()
		return e
	case *ast.ForEachDotRange:
		e.StartIdx = r.expression(e.StartIdx)
		e.EndIdx = r.expression(e.EndIdx)
		r.push()
		r.local(e.Var)
		e.Cond = r.expression(e.Cond)
		e.Block = r.block(e.Block)
		r.pop()
		return e
	case *ast.ForEverLoop:
		e.Block = r.block(e.Block)
		return e
	default:
		return expression
	}
}

func (r *rewriter) typeName(name string) string {
	if name == "" {
		return name
	}
	if dot := strings.IndexByte(name, '.'); dot > 0 {
		alias, member := name[:dot], name[dot+1:]
		if imported := r.unit.importAliases[alias]; imported != nil {
			if internal, ok := imported.Exports[member]; ok {
				return internal
			}
			r.error(fmt.Sprintf("type %s.%s is not public", alias, member))
			return name
		}
	}
	if internal, ok := r.unit.Symbols[name]; ok {
		return internal
	}
	if internal, ok := r.unit.legacyNames[name]; ok {
		return internal
	}
	return name
}

func (r *rewriter) error(message string) {
	*r.diagnostics = append(*r.diagnostics, Diagnostic{File: r.unit.Path, Message: message})
}
