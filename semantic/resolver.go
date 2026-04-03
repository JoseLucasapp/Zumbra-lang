package semantic

import (
	"zumbra/ast"
	objbuiltins "zumbra/object/builtins"
)

type functionContext struct {
	fn          *ast.FunctionLiteral
	rootScope   *Scope
	freeSymbols []Symbol
}

type Resolver struct {
	global    *Scope
	scope     *Scope
	errors    []error
	result    *Result
	functions []*functionContext
}

func NewResolver() *Resolver {
	global := NewGlobalScope()

	r := &Resolver{
		global:    global,
		scope:     global,
		errors:    []error{},
		result:    NewResult(),
		functions: []*functionContext{},
	}

	r.installBuiltins()
	return r
}

func NewResolverWithGlobalScope(global *Scope) *Resolver {
	if global == nil {
		global = NewGlobalScope()
	}

	r := &Resolver{
		global:    global,
		scope:     global,
		errors:    []error{},
		result:    NewResult(),
		functions: []*functionContext{},
	}

	if len(global.Symbols) == 0 {
		r.installBuiltins()
	}

	return r
}

func (r *Resolver) Result() *Result {
	return r.result
}

func (r *Resolver) GlobalScope() *Scope {
	return r.global
}

func (r *Resolver) ResetForNextRun() {
	r.scope = r.global
	r.errors = []error{}
	r.result = NewResult()
	r.functions = []*functionContext{}
}

func (r *Resolver) Resolve(program *ast.Program) []error {
	r.ResetForNextRun()
	r.resolveProgram(program)
	return r.errors
}

func (r *Resolver) installBuiltins() {
	for _, b := range objbuiltins.Builtins {
		_ = r.global.Define(Symbol{
			Name:        b.Name,
			Kind:        SymbolBuiltin,
			Depth:       r.global.Depth,
			Mutable:     false,
			IsFree:      false,
			OriginDepth: r.global.Depth,
		})
	}
}

func (r *Resolver) pushScope(kind ScopeKind) *Scope {
	r.scope = NewChildScope(r.scope, kind)
	return r.scope
}

func (r *Resolver) popScope() {
	if r.scope.Parent != nil {
		r.scope = r.scope.Parent
	}
}

func (r *Resolver) addError(err error) {
	if err != nil {
		r.errors = append(r.errors, err)
	}
}

func (r *Resolver) currentFunction() *functionContext {
	if len(r.functions) == 0 {
		return nil
	}
	return r.functions[len(r.functions)-1]
}

func (r *Resolver) pushFunction(fn *ast.FunctionLiteral, rootScope *Scope) {
	r.functions = append(r.functions, &functionContext{
		fn:          fn,
		rootScope:   rootScope,
		freeSymbols: []Symbol{},
	})
}

func (r *Resolver) popFunction() {
	if len(r.functions) == 0 {
		return
	}

	ctx := r.functions[len(r.functions)-1]
	r.functions = r.functions[:len(r.functions)-1]

	r.result.Functions[ctx.fn] = FunctionResolution{
		FreeSymbols: append([]Symbol(nil), ctx.freeSymbols...),
		ScopeDepth:  ctx.rootScope.Depth,
	}
}

func (r *Resolver) addFreeSymbol(sym Symbol) {
	ctx := r.currentFunction()
	if ctx == nil {
		return
	}

	for _, existing := range ctx.freeSymbols {
		if existing.Name == sym.Name && existing.OriginDepth == sym.OriginDepth {
			return
		}
	}

	ctx.freeSymbols = append(ctx.freeSymbols, sym)
}

func scopeBelongsToFunction(defScope *Scope, fnRoot *Scope) bool {
	for current := defScope; current != nil; current = current.Parent {
		if current == fnRoot {
			return true
		}
	}
	return false
}

func (r *Resolver) defineSymbol(name string, kind SymbolKind, mutable bool, tok interface{ GetToken() }) {
	err := r.scope.Define(Symbol{
		Name:        name,
		Kind:        kind,
		Depth:       r.scope.Depth,
		Mutable:     mutable,
		IsFree:      false,
		OriginDepth: r.scope.Depth,
	})
	if err != nil {
		r.addError(ErrDuplicateSymbol(name))
	}
}

func (r *Resolver) resolveProgram(program *ast.Program) {
	if program == nil {
		return
	}

	for _, stmt := range program.Statements {
		r.resolveStatement(stmt)
	}
}

func (r *Resolver) resolveBlockStatement(block *ast.BlockStatement, createScope bool) {
	if block == nil {
		return
	}

	if createScope {
		r.pushScope(ScopeBlock)
		defer r.popScope()
	}

	for _, stmt := range block.Statements {
		r.resolveStatement(stmt)
	}
}

func (r *Resolver) resolveStatement(stmt ast.Statement) {
	switch s := stmt.(type) {
	case *ast.VarStatement:
		r.resolveVarStatement(s)

	case *ast.AssignStatement:
		r.resolveAssignStatement(s)

	case *ast.ReturnStatement:
		if s.ReturnValue != nil {
			r.resolveExpression(s.ReturnValue)
		}

	case *ast.ExpressionStatement:
		if s.Expression != nil {
			r.resolveExpression(s.Expression)
		}

	case *ast.WhileStatement:
		r.resolveWhileStatement(s)

	case *ast.ImportStatement:
		if s.Path != nil {
			r.resolveExpression(s.Path)
		}
	}
}

func (r *Resolver) resolveVarStatement(stmt *ast.VarStatement) {
	if stmt == nil || stmt.Name == nil {
		return
	}

	if stmt.Value != nil {
		r.resolveExpression(stmt.Value)
	}

	kind := SymbolVar
	if _, ok := stmt.Value.(*ast.FunctionLiteral); ok {
		kind = SymbolFunction
	}

	err := r.scope.Define(Symbol{
		Name:        stmt.Name.Value,
		Kind:        kind,
		Depth:       r.scope.Depth,
		Mutable:     true,
		IsFree:      false,
		OriginDepth: r.scope.Depth,
	})
	if err != nil {
		r.addError(ErrDuplicateSymbolAt(stmt.Name.Value, stmt.Token))
	}
}

func (r *Resolver) resolveAssignStatement(stmt *ast.AssignStatement) {
	if stmt == nil || stmt.Name == nil {
		return
	}

	if _, _, ok := r.scope.Resolve(stmt.Name.Value); !ok {
		r.addError(ErrAssignmentToUndefinedSymbolAt(stmt.Name.Value, stmt.Token))
	}

	if stmt.Value != nil {
		r.resolveExpression(stmt.Value)
	}
}

func (r *Resolver) resolveWhileStatement(stmt *ast.WhileStatement) {
	if stmt == nil {
		return
	}

	if stmt.Condition != nil {
		r.resolveExpression(stmt.Condition)
	}

	r.resolveBlockStatement(stmt.Body, true)
}

func (r *Resolver) resolveExpression(exp ast.Expression) {
	switch e := exp.(type) {
	case *ast.Identifier:
		r.resolveIdentifier(e)

	case *ast.IntegerLiteral, *ast.FloatLiteral, *ast.StringLiteral, *ast.Boolean:
		return

	case *ast.PrefixExpression:
		if e.Right != nil {
			r.resolveExpression(e.Right)
		}

	case *ast.InfixExpression:
		if e.Left != nil {
			r.resolveExpression(e.Left)
		}
		if e.Right != nil {
			r.resolveExpression(e.Right)
		}

	case *ast.IfExpression:
		if e.Condition != nil {
			r.resolveExpression(e.Condition)
		}
		r.resolveBlockStatement(e.Consequence, true)
		if e.Alternative != nil {
			r.resolveBlockStatement(e.Alternative, true)
		}

	case *ast.FunctionLiteral:
		r.resolveFunctionLiteral(e)

	case *ast.CallExpression:
		if e.Function != nil {
			r.resolveExpression(e.Function)
		}
		for _, arg := range e.Arguments {
			r.resolveExpression(arg)
		}

	case *ast.ArrayLiteral:
		for _, el := range e.Elements {
			r.resolveExpression(el)
		}

	case *ast.DictLiteral:
		for key, value := range e.Pairs {
			if key != nil {
				r.resolveExpression(key)
			}
			if value != nil {
				r.resolveExpression(value)
			}
		}

	case *ast.IndexExpression:
		if e.Left != nil {
			r.resolveExpression(e.Left)
		}
		if e.Index != nil {
			r.resolveExpression(e.Index)
		}

	case *ast.AttributeAccess:
		if e.Object != nil {
			r.resolveExpression(e.Object)
		}

	case *ast.DotExpression:
		if e.Left != nil {
			r.resolveExpression(e.Left)
		}

	case *ast.AwaitExpression:
		if e.Value != nil {
			r.resolveExpression(e.Value)
		}

	case *ast.TryExpression:
		if e.Value != nil {
			r.resolveExpression(e.Value)
		}

	case *ast.ErrorHandlerExpression:
		if e.Left != nil {
			r.resolveExpression(e.Left)
		}

		r.pushScope(ScopeBlock)
		if e.ErrorIdent != nil {
			err := r.scope.Define(Symbol{
				Name:        e.ErrorIdent.Value,
				Kind:        SymbolVar,
				Depth:       r.scope.Depth,
				Mutable:     false,
				IsFree:      false,
				OriginDepth: r.scope.Depth,
			})
			if err != nil {
				r.addError(ErrDuplicateSymbolAt(e.ErrorIdent.Value, e.ErrorIdent.Token))
			}
		}
		r.resolveBlockStatement(e.Handler, false)
		r.popScope()

	case *ast.ForLoop:
		r.resolveForLoop(e)

	case *ast.ForEachArrayLoop:
		r.resolveForEachArrayLoop(e)

	case *ast.ForEachMapLoop:
		r.resolveForEachMapLoop(e)

	case *ast.ForEachDotRange:
		r.resolveForEachDotRange(e)

	case *ast.ForEverLoop:
		r.resolveForEverLoop(e)

	case *ast.BlockStatement:
		r.resolveBlockStatement(e, true)

	case *ast.BreakExpression, *ast.ContinueExpression:
		return
	}
}

func (r *Resolver) resolveFunctionLiteral(fn *ast.FunctionLiteral) {
	if fn == nil {
		return
	}

	functionScope := r.pushScope(ScopeFunction)
	r.pushFunction(fn, functionScope)

	if fn.Name != "" {
		err := r.scope.Define(Symbol{
			Name:        fn.Name,
			Kind:        SymbolFunction,
			Depth:       r.scope.Depth,
			Mutable:     false,
			IsFree:      false,
			OriginDepth: r.scope.Depth,
		})
		if err != nil {
			r.addError(ErrDuplicateSymbol(fn.Name))
		}
	}

	for _, param := range fn.Parameters {
		if param == nil {
			continue
		}

		err := r.scope.Define(Symbol{
			Name:        param.Value,
			Kind:        SymbolParam,
			Depth:       r.scope.Depth,
			Mutable:     false,
			IsFree:      false,
			OriginDepth: r.scope.Depth,
		})
		if err != nil {
			r.addError(ErrDuplicateSymbolAt(param.Value, param.Token))
		}
	}

	r.resolveBlockStatement(fn.Body, false)
	r.popFunction()
	r.popScope()
}

func (r *Resolver) resolveIdentifier(id *ast.Identifier) {
	if id == nil {
		return
	}

	sym, defScope, ok := r.scope.Resolve(id.Value)
	if !ok {
		r.addError(ErrUndefinedSymbolAt(id.Value, id.Token))
		return
	}

	ctx := r.currentFunction()
	if ctx == nil || defScope == nil {
		return
	}

	if sym.Kind == SymbolBuiltin {
		return
	}

	if scopeBelongsToFunction(defScope, ctx.rootScope) {
		return
	}

	r.addFreeSymbol(Symbol{
		Name:        sym.Name,
		Kind:        sym.Kind,
		Depth:       ctx.rootScope.Depth,
		Mutable:     sym.Mutable,
		IsFree:      true,
		OriginDepth: defScope.Depth,
	})
}

func (r *Resolver) resolveForLoop(loop *ast.ForLoop) {
	if loop == nil {
		return
	}

	r.pushScope(ScopeBlock)
	defer r.popScope()

	if loop.Init != nil {
		r.resolveExpression(loop.Init)
	}
	if loop.Cond != nil {
		r.resolveExpression(loop.Cond)
	}
	if loop.Update != nil {
		r.resolveExpression(loop.Update)
	}

	r.resolveBlockStatement(loop.Block, false)
}

func (r *Resolver) resolveForEachArrayLoop(loop *ast.ForEachArrayLoop) {
	if loop == nil {
		return
	}

	if loop.Value != nil {
		r.resolveExpression(loop.Value)
	}
	if loop.Cond != nil {
		r.resolveExpression(loop.Cond)
	}

	r.pushScope(ScopeBlock)
	defer r.popScope()

	if loop.Var != "" {
		err := r.scope.Define(Symbol{
			Name:        loop.Var,
			Kind:        SymbolVar,
			Depth:       r.scope.Depth,
			Mutable:     true,
			IsFree:      false,
			OriginDepth: r.scope.Depth,
		})
		if err != nil {
			r.addError(ErrDuplicateSymbol(loop.Var))
		}
	}

	r.resolveBlockStatement(loop.Block, false)
}

func (r *Resolver) resolveForEachMapLoop(loop *ast.ForEachMapLoop) {
	if loop == nil {
		return
	}

	if loop.X != nil {
		r.resolveExpression(loop.X)
	}
	if loop.Cond != nil {
		r.resolveExpression(loop.Cond)
	}

	r.pushScope(ScopeBlock)
	defer r.popScope()

	if loop.Key != "" {
		err := r.scope.Define(Symbol{
			Name:        loop.Key,
			Kind:        SymbolVar,
			Depth:       r.scope.Depth,
			Mutable:     true,
			IsFree:      false,
			OriginDepth: r.scope.Depth,
		})
		if err != nil {
			r.addError(ErrDuplicateSymbol(loop.Key))
		}
	}

	if loop.Value != "" {
		err := r.scope.Define(Symbol{
			Name:        loop.Value,
			Kind:        SymbolVar,
			Depth:       r.scope.Depth,
			Mutable:     true,
			IsFree:      false,
			OriginDepth: r.scope.Depth,
		})
		if err != nil {
			r.addError(ErrDuplicateSymbol(loop.Value))
		}
	}

	r.resolveBlockStatement(loop.Block, false)
}

func (r *Resolver) resolveForEachDotRange(loop *ast.ForEachDotRange) {
	if loop == nil {
		return
	}

	if loop.StartIdx != nil {
		r.resolveExpression(loop.StartIdx)
	}
	if loop.EndIdx != nil {
		r.resolveExpression(loop.EndIdx)
	}
	if loop.Cond != nil {
		r.resolveExpression(loop.Cond)
	}

	r.pushScope(ScopeBlock)
	defer r.popScope()

	if loop.Var != "" {
		err := r.scope.Define(Symbol{
			Name:        loop.Var,
			Kind:        SymbolVar,
			Depth:       r.scope.Depth,
			Mutable:     true,
			IsFree:      false,
			OriginDepth: r.scope.Depth,
		})
		if err != nil {
			r.addError(ErrDuplicateSymbol(loop.Var))
		}
	}

	r.resolveBlockStatement(loop.Block, false)
}

func (r *Resolver) resolveForEverLoop(loop *ast.ForEverLoop) {
	if loop == nil {
		return
	}

	r.pushScope(ScopeBlock)
	defer r.popScope()

	r.resolveBlockStatement(loop.Block, false)
}
