package compiler

import (
	"fmt"
	"strings"
	"zumbra/ast"
)

type DiagnosticSeverity string

const (
	DiagnosticWarning DiagnosticSeverity = "warning"
	DiagnosticError   DiagnosticSeverity = "error"
)

type Diagnostic struct {
	Severity DiagnosticSeverity
	Message  string
}

type UsageInfo struct {
	Writes      int
	Reads       int
	IsParam     bool
	IsSynthetic bool
}

type Analyzer struct {
	usages     map[string]*UsageInfo
	imports    []string
	warnings   []Diagnostic
	scopeStack []map[string]bool
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		usages:     map[string]*UsageInfo{},
		imports:    []string{},
		warnings:   []Diagnostic{},
		scopeStack: []map[string]bool{{}},
	}
}

func (a *Analyzer) Warnings() []Diagnostic {
	return a.warnings
}

func AnalyzeProgram(program *ast.Program) []Diagnostic {
	analyzer := NewAnalyzer()
	analyzer.visitProgram(program)
	analyzer.emitUnusedDiagnostics()
	return analyzer.Warnings()
}

func (a *Analyzer) visitProgram(program *ast.Program) {
	if program == nil {
		return
	}
	a.visitStatementList(program.Statements)
}

func (a *Analyzer) visitStatementList(stmts []ast.Statement) {
	terminated := false

	for _, stmt := range stmts {
		if terminated {
			a.warnings = append(a.warnings, Diagnostic{
				Severity: DiagnosticWarning,
				Message:  "unreachable code detected",
			})
			break
		}

		a.visitStatement(stmt)

		switch node := stmt.(type) {
		case *ast.ReturnStatement:
			terminated = true
		case *ast.ExpressionStatement:
			switch node.Expression.(type) {
			case *ast.BreakExpression, *ast.ContinueExpression:
				terminated = true
			}
		}
	}
}

func (a *Analyzer) pushScope() {
	a.scopeStack = append(a.scopeStack, map[string]bool{})
}

func (a *Analyzer) popScope() {
	if len(a.scopeStack) > 1 {
		a.scopeStack = a.scopeStack[:len(a.scopeStack)-1]
	}
}

func isSyntheticName(name string) bool {
	return strings.HasPrefix(name, "__z_") || strings.HasPrefix(name, "__zm_")
}

func (a *Analyzer) markDeclared(name string) {
	scope := a.scopeStack[len(a.scopeStack)-1]
	scope[name] = true

	if _, ok := a.usages[name]; !ok {
		a.usages[name] = &UsageInfo{}
	}
	a.usages[name].Writes++
	if isSyntheticName(name) {
		a.usages[name].IsSynthetic = true
	}
}

func (a *Analyzer) markDeclaredParam(name string) {
	scope := a.scopeStack[len(a.scopeStack)-1]
	scope[name] = true

	if _, ok := a.usages[name]; !ok {
		a.usages[name] = &UsageInfo{}
	}
	a.usages[name].Writes++
	a.usages[name].IsParam = true
	if isSyntheticName(name) {
		a.usages[name].IsSynthetic = true
	}
}

func (a *Analyzer) markRead(name string) {
	if _, ok := a.usages[name]; !ok {
		a.usages[name] = &UsageInfo{}
	}
	a.usages[name].Reads++
}

func (a *Analyzer) emitUnusedDiagnostics() {
	for name, usage := range a.usages {
		if usage.IsSynthetic {
			continue
		}

		if usage.Writes > 0 && usage.Reads == 0 {
			msg := fmt.Sprintf("variable declared but never used: %s", name)
			if usage.IsParam {
				msg = fmt.Sprintf("parameter declared but never used: %s", name)
			}

			a.warnings = append(a.warnings, Diagnostic{
				Severity: DiagnosticWarning,
				Message:  msg,
			})
		}
	}

	for _, imp := range a.imports {
		a.warnings = append(a.warnings, Diagnostic{
			Severity: DiagnosticWarning,
			Message:  fmt.Sprintf("import may be unused: %s", imp),
		})
	}
}

func (a *Analyzer) visitStatement(stmt ast.Statement) {
	switch node := stmt.(type) {
	case *ast.VarStatement:
		if node.Name != nil {
			a.markDeclared(node.Name.Value)
		}
		if node.Value != nil {
			a.visitExpression(node.Value)
		}

	case *ast.AssignStatement:
		if node.Name != nil {
			a.markRead(node.Name.Value)
		}
		if node.Value != nil {
			a.visitExpression(node.Value)
		}

	case *ast.ConstStatement:
		if node.Name != nil {
			a.markDeclared(node.Name.Value)
		}
		if node.Value != nil {
			a.visitExpression(node.Value)
		}

	case *ast.AttributeAssignStatement:
		if node.Target != nil {
			a.visitExpression(node.Target)
		}
		if node.Value != nil {
			a.visitExpression(node.Value)
		}

	case *ast.IndexAssignStatement:
		if node.Target != nil {
			a.visitExpression(node.Target)
		}
		if node.Value != nil {
			a.visitExpression(node.Value)
		}

	case *ast.ReturnStatement:
		if node.ReturnValue != nil {
			a.visitExpression(node.ReturnValue)
		}

	case *ast.ExpressionStatement:
		if node.Expression != nil {
			a.visitExpression(node.Expression)
		}

	case *ast.WhileStatement:
		if node.Condition != nil {
			a.visitExpression(node.Condition)
		}
		a.pushScope()
		if node.Body != nil {
			a.visitStatementList(node.Body.Statements)
		}
		a.popScope()

	case *ast.StructStatement:
		if node.Name != nil {
			a.markDeclared(node.Name.Value)
		}
		for _, method := range node.Methods {
			if method == nil || method.Function == nil {
				continue
			}
			a.visitExpression(method.Function)
		}

	case *ast.EnumStatement:
		if node.Name != nil {
			a.markDeclared(node.Name.Value)
		}

	case *ast.TypeAliasStatement:
		return

	case *ast.ExternBlockStatement:
		for _, function := range node.Functions {
			if function != nil && function.Name != nil {
				a.markDeclared(function.Name.Value)
			}
		}

	case *ast.UnsafeStatement:
		a.pushScope()
		if node.Body != nil {
			a.visitStatementList(node.Body.Statements)
		}
		a.popScope()

	case *ast.ImportStatement:
		if node.Path != nil {
			a.imports = append(a.imports, node.Path.Value)
		}
	}
}

func (a *Analyzer) visitExpression(expr ast.Expression) {
	switch node := expr.(type) {
	case *ast.Identifier:
		a.markRead(node.Value)

	case *ast.IntegerLiteral, *ast.FloatLiteral, *ast.Boolean, *ast.StringLiteral:
		return

	case *ast.PrefixExpression:
		if node.Right != nil {
			a.visitExpression(node.Right)
		}

	case *ast.InfixExpression:
		if node.Left != nil {
			a.visitExpression(node.Left)
		}
		if node.Right != nil {
			a.visitExpression(node.Right)
		}

	case *ast.IfExpression:
		if node.Condition != nil {
			a.visitExpression(node.Condition)
		}
		if node.Consequence != nil {
			a.pushScope()
			a.visitStatementList(node.Consequence.Statements)
			a.popScope()
		}
		if node.Alternative != nil {
			a.pushScope()
			a.visitStatementList(node.Alternative.Statements)
			a.popScope()
		}

	case *ast.FunctionLiteral:
		a.pushScope()
		for _, param := range node.Parameters {
			if param != nil {
				a.markDeclaredParam(param.Value)
			}
		}
		if node.Body != nil {
			a.visitStatementList(node.Body.Statements)
		}
		a.popScope()

	case *ast.CallExpression:
		if node.Function != nil {
			a.visitExpression(node.Function)
		}
		for _, arg := range node.Arguments {
			a.visitExpression(arg)
		}

	case *ast.ArrayLiteral:
		for _, el := range node.Elements {
			a.visitExpression(el)
		}

	case *ast.DictLiteral:
		for k, v := range node.Pairs {
			a.visitExpression(k)
			a.visitExpression(v)
		}

	case *ast.IndexExpression:
		if node.Left != nil {
			a.visitExpression(node.Left)
		}
		if node.Index != nil {
			a.visitExpression(node.Index)
		}

	case *ast.AttributeAccess:
		if node.Object != nil {
			a.visitExpression(node.Object)
		}

	case *ast.SpawnExpression:
		if node.Value != nil {
			a.visitExpression(node.Value)
		}

	case *ast.AwaitExpression:
		if node.Value != nil {
			a.visitExpression(node.Value)
		}

	case *ast.TryExpression:
		if node.Value != nil {
			a.visitExpression(node.Value)
		}

	case *ast.MatchExpression:
		if node.Value != nil {
			a.visitExpression(node.Value)
		}
		for _, candidate := range node.Cases {
			if candidate.Pattern != nil {
				a.visitExpression(candidate.Pattern)
			}
			a.pushScope()
			if candidate.Body != nil {
				a.visitStatementList(candidate.Body.Statements)
			}
			a.popScope()
		}
		if node.Default != nil {
			a.pushScope()
			a.visitStatementList(node.Default.Statements)
			a.popScope()
		}

	case *ast.ErrorHandlerExpression:
		if node.Left != nil {
			a.visitExpression(node.Left)
		}
		a.pushScope()
		if node.ErrorIdent != nil {
			a.markDeclared(node.ErrorIdent.Value)
		}
		if node.Handler != nil {
			a.visitStatementList(node.Handler.Statements)
		}
		a.popScope()

	case *ast.ForEverLoop:
		a.pushScope()
		if node.Block != nil {
			a.visitStatementList(node.Block.Statements)
		}
		a.popScope()

	case *ast.ForEachDotRange:
		a.pushScope()
		if node.Var != "" {
			a.markDeclared(node.Var)
		}
		if node.StartIdx != nil {
			a.visitExpression(node.StartIdx)
		}
		if node.EndIdx != nil {
			a.visitExpression(node.EndIdx)
		}
		if node.Cond != nil {
			a.visitExpression(node.Cond)
		}
		if node.Block != nil {
			a.visitStatementList(node.Block.Statements)
		}
		a.popScope()

	case *ast.ForEachArrayLoop:
		a.pushScope()
		if node.Var != "" {
			a.markDeclared(node.Var)
		}
		if node.Value != nil {
			a.visitExpression(node.Value)
		}
		if node.Cond != nil {
			a.visitExpression(node.Cond)
		}
		if node.Block != nil {
			a.visitStatementList(node.Block.Statements)
		}
		a.popScope()

	case *ast.ForEachMapLoop:
		a.pushScope()
		if node.Key != "" {
			a.markDeclared(node.Key)
		}
		if node.Value != "" {
			a.markDeclared(node.Value)
		}
		if node.X != nil {
			a.visitExpression(node.X)
		}
		if node.Cond != nil {
			a.visitExpression(node.Cond)
		}
		if node.Block != nil {
			a.visitStatementList(node.Block.Statements)
		}
		a.popScope()

	case *ast.BreakExpression, *ast.ContinueExpression:
		return
	}
}
