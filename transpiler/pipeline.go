package transpiler

import (
	"fmt"
	"zumbra/mir"
	"zumbra/pipeline"
)

// ZumbraTranspilerPipeline ensures the Go backend receives an already parsed,
// semantically checked and typed source unit. Z7 will replace the textual
// backend internals with direct MIR consumption.
func ZumbraTranspilerPipeline(result *pipeline.Result) (string, error) {
	if result == nil || result.Program == nil || result.MIR == nil {
		return "", fmt.Errorf("transpiler received an incomplete pipeline result")
	}
	for _, declaration := range result.MIR.Declarations {
		if declaration.Op == mir.OpExtern {
			return "", fmt.Errorf("extern C is available only in the native backend")
		}
	}
	for _, function := range result.MIR.Functions {
		if function != nil && function.Async {
			return "", fmt.Errorf("Z9 concurrency is supported by the VM and native backend; the legacy Go transpiler does not schedule async functions")
		}
		if function != nil && mirRegionHasConcurrency(function.Body) {
			return "", fmt.Errorf("Z9 concurrency is supported by the VM and native backend; the legacy Go transpiler does not support spawn/await")
		}
	}
	if mirRegionHasConcurrency(result.MIR.Entry) {
		return "", fmt.Errorf("Z9 concurrency is supported by the VM and native backend; the legacy Go transpiler does not support spawn/await")
	}
	return ZumbraTranspiler(result.Program.String())
}

func mirRegionHasConcurrency(region *mir.Region) bool {
	if region == nil {
		return false
	}
	for _, instruction := range region.Instructions {
		if instruction == nil {
			continue
		}
		if instruction.Op == mir.OpSpawn || instruction.Op == mir.OpAwait {
			return true
		}
		for _, child := range instruction.Regions {
			if mirRegionHasConcurrency(child) {
				return true
			}
		}
	}
	return false
}
