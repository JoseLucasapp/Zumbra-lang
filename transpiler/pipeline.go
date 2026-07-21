package transpiler

import (
	"fmt"
	"zumbra/pipeline"
)

// ZumbraTranspilerPipeline ensures the Go backend receives an already parsed,
// semantically checked and typed source unit. Z7 will replace the textual
// backend internals with direct MIR consumption.
func ZumbraTranspilerPipeline(result *pipeline.Result) (string, error) {
	if result == nil || result.Program == nil || result.MIR == nil {
		return "", fmt.Errorf("transpiler received an incomplete pipeline result")
	}
	return ZumbraTranspiler(result.Source)
}
