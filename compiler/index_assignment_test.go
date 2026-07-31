package compiler

import (
	"testing"
	"zumbra/code"
)

func TestCompileIndexAssignment(t *testing.T) {
	tests := []compilerTestCase{
		{
			input: `
				var memory << [0, 0];
				memory[1] << 42;
				memory[1];
			`,
			expectedConstants: []interface{}{0, 0, 1, 42, 1},
			expectedInstructions: []code.Instructions{
				code.Make(code.OpConstant, 0),
				code.Make(code.OpConstant, 1),
				code.Make(code.OpArray, 2),
				code.Make(code.OpSetGlobal, 0),
				code.Make(code.OpGetGlobal, 0),
				code.Make(code.OpConstant, 2),
				code.Make(code.OpConstant, 3),
				code.Make(code.OpSetIndex),
				code.Make(code.OpGetGlobal, 0),
				code.Make(code.OpConstant, 4),
				code.Make(code.OpIndex),
				code.Make(code.OpPop),
			},
		},
	}

	runCompilerTests(t, tests)
}
