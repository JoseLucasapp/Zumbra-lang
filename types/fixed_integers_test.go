package types

import (
	"strings"
	"testing"
)

func TestFixedIntegerExpressionsAreTyped(t *testing.T) {
	errors := checkInput(t, `
		var opcode << 0xA9u8;
		var next << opcode + 1;
		var address << u16(0x8000);
		var wrapped << wrapAdd(255u8, 1u8);
		var saturated << satAdd(255u8, 1u8);
	`)
	if len(errors) != 0 {
		t.Fatalf("expected no errors, got %v", errors)
	}
}

func TestFixedIntegerTypesMustMatch(t *testing.T) {
	errors := checkInput(t, `var value << 1u8 + 1u16;`)
	if len(errors) != 1 {
		t.Fatalf("expected one error, got %v", errors)
	}
	if !strings.Contains(errors[0].Error(), "fixed integer types must match") {
		t.Fatalf("unexpected error: %v", errors[0])
	}
}

func TestFixedIntegerConversionRejectsFloatType(t *testing.T) {
	errors := checkInput(t, `var value << u8(1.5);`)
	if len(errors) != 1 {
		t.Fatalf("expected one error, got %v", errors)
	}
	if !strings.Contains(errors[0].Error(), "u8 expects an integer") {
		t.Fatalf("unexpected error: %v", errors[0])
	}
}
