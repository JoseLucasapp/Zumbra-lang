package types

import (
	"strings"
	"testing"
)

func TestCheckIndexAssignment(t *testing.T) {
	valid := checkInput(t, `
		var memory << [0u8, 0u8];
		memory[1] << 0xA9u8;
		var player << {"x": 10};
		player["x"] << 20;
	`)
	if len(valid) != 0 {
		t.Fatalf("expected no errors, got %v", valid)
	}
}

func TestCheckInvalidIndexAssignment(t *testing.T) {
	tests := []struct {
		input   string
		message string
	}{
		{`var xs << [1, 2]; xs["zero"] << 3;`, "array index must be int"},
		{`var xs << [1u8, 2u8]; xs[0] << 300;`, "array element expects u8, got int"},
		{`var data << {"x": 1}; data[1] << 2;`, "dict key expects string, got int"},
		{`var text << "abc"; text[0] << "A";`, "index assignment not supported for string"},
	}

	for _, test := range tests {
		errs := checkInput(t, test.input)
		if len(errs) != 1 {
			t.Fatalf("expected one error for %q, got %v", test.input, errs)
		}
		if !strings.Contains(errs[0].Error(), test.message) {
			t.Fatalf("unexpected error for %q: %v", test.input, errs[0])
		}
	}
}
