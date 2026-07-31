package evaluator

import (
	"zumbra/object"
	"zumbra/pipeline"
)

// EvalPipeline executes the same program that produced the canonical HIR/MIR.
func EvalPipeline(result *pipeline.Result, env *object.Environment) object.Object {
	if result == nil || result.Program == nil {
		return NULL
	}
	return Eval(result.Program, env)
}
