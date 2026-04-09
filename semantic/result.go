package semantic

import "zumbra/ast"

type FunctionResolution struct {
	FreeSymbols []Symbol
	ScopeDepth  int
}

type Warning struct {
	Message string
	Code    string
}

type Result struct {
	Functions map[*ast.FunctionLiteral]FunctionResolution
	Warnings  []Warning
}

func NewResult() *Result {
	return &Result{
		Functions: make(map[*ast.FunctionLiteral]FunctionResolution),
		Warnings:  []Warning{},
	}
}
