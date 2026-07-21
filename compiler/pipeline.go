package compiler

import (
	"fmt"
	"zumbra/pipeline"
)

// CompilePipeline consumes the canonical analyzed front-end result.
func (c *Compiler) CompilePipeline(result *pipeline.Result) error {
	if result == nil || result.Program == nil {
		return fmt.Errorf("compiler received an empty pipeline result")
	}
	if result.HIR == nil || result.MIR == nil {
		return fmt.Errorf("compiler requires HIR and MIR")
	}
	return c.Compile(result.Program)
}
