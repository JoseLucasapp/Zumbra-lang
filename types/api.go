package types

import "zumbra/ast"

func Analyze(program *ast.Program) []error {
	c := NewChecker()
	return c.Check(program)
}

func AnalyzeWithChecker(c *Checker, program *ast.Program) []error {
	if c == nil {
		c = NewChecker()
	}
	return c.Check(program)
}
