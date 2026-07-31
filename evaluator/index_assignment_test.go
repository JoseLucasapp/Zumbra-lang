package evaluator

import (
	"strings"
	"testing"
	"zumbra/object"
)

func TestEvaluateArrayIndexAssignment(t *testing.T) {
	result := testEval(`
		var memory << [0u8, 0u8, 0u8];
		memory[1] << 0xA9u8;
		memory[1];
	`)
	value, ok := result.(*object.FixedInteger)
	if !ok || value.Kind != object.FixedU8 || value.UnsignedValue() != 0xA9 {
		t.Fatalf("unexpected result: %T (%v)", result, result)
	}
}

func TestEvaluateDictIndexAssignment(t *testing.T) {
	result := testEval(`
		var player << {"x": 10};
		player["x"] << 30;
		player["score"] << 100;
		player["x"] + player["score"];
	`)
	testIntegerObject(t, result, 130)
}

func TestIndexAssignmentErrors(t *testing.T) {
	tests := []struct {
		input   string
		message string
	}{
		{`var xs << [1]; xs[2] << 3;`, "array index out of bounds"},
		{`var text << "zumbra"; text[0] << "Z";`, "index assignment not supported: STRING"},
		{`var xs << [1]; xs["zero"] << 3;`, "array index must be an integer"},
	}

	for _, test := range tests {
		result := testEval(test.input)
		err, ok := result.(*object.Error)
		if !ok {
			t.Fatalf("expected error for %q, got %T (%v)", test.input, result, result)
		}
		if !strings.Contains(err.Message, test.message) {
			t.Fatalf("unexpected error for %q: %s", test.input, err.Message)
		}
	}
}
