package types

import (
	"strings"
	"testing"
)

func TestBitwiseExpressionsRequireIntegers(t *testing.T) {
	valid := checkInput(t, `
		var mask << 0xF0 band 0x0F;
		var shifted << mask shl 2;
		var inverted << bnot shifted;
	`)
	if len(valid) != 0 {
		t.Fatalf("expected no errors, got %v", valid)
	}

	invalid := checkInput(t, `var value << 1.5 band 1;`)
	if len(invalid) != 1 {
		t.Fatalf("expected one error, got %v", invalid)
	}
	if !strings.Contains(invalid[0].Error(), "band expects int operands") {
		t.Fatalf("unexpected error: %v", invalid[0])
	}
}

func TestBitNotRequiresInteger(t *testing.T) {
	errors := checkInput(t, `var value << bnot "text";`)
	if len(errors) != 1 {
		t.Fatalf("expected one error, got %v", errors)
	}
	if !strings.Contains(errors[0].Error(), "bnot expects int") {
		t.Fatalf("unexpected error: %v", errors[0])
	}
}
