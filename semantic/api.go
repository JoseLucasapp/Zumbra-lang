package semantic

import "zumbra/ast"

func Analyze(program *ast.Program) (*Result, []error) {
	r := NewResolver()
	errs := r.Resolve(program)
	return r.Result(), errs
}

func AnalyzeWithResolver(r *Resolver, program *ast.Program) (*Result, []error) {
	if r == nil {
		r = NewResolver()
	}
	errs := r.Resolve(program)
	return r.Result(), errs
}
