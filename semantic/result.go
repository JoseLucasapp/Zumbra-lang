package semantic

import "zumbra/ast"

type FunctionResolution struct {
	FreeSymbols []Symbol
	ScopeDepth  int
}

type Result struct {
	Functions map[*ast.FunctionLiteral]FunctionResolution
}

func NewResult() *Result {
	return &Result{
		Functions: make(map[*ast.FunctionLiteral]FunctionResolution),
	}
}
