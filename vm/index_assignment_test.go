package vm

import (
	"strings"
	"testing"
	"zumbra/compiler"
)

func TestVMIndexAssignment(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
				var memory << [0, 0, 0];
				memory[2] << 169;
				memory[2];
			`,
			expected: 169,
		},
		{
			input: `
				var player << {"x": 10};
				player["x"] << 30;
				player["score"] << 12;
				player["x"] + player["score"];
			`,
			expected: 42,
		},
		{
			input: `
				var matrix << [[0, 0], [0, 0]];
				matrix[1][0] << 7;
				matrix[1][0];
			`,
			expected: 7,
		},
	}
	runVmTests(t, tests)
}

func TestVMIndexAssignmentOutOfBounds(t *testing.T) {
	program := parse(`var xs << [1]; xs[5] << 2;`)
	compiled := compiler.New()
	if err := compiled.Compile(program); err != nil {
		t.Fatalf("compiler error: %v", err)
	}
	machine := New(compiled.Bytecode())
	err := machine.Run()
	if err == nil || !strings.Contains(err.Error(), "array index out of bounds") {
		t.Fatalf("expected bounds error, got %v", err)
	}
}
