package compiler

import (
	"strings"
	"testing"
)

func TestBytecodeCompilerRejectsExternWithNativeGuidance(t *testing.T) {
	program := parse(`extern "C" { fct answer() -> i32; } unsafe { answer(); }`)
	compiler := New()
	err := compiler.Compile(program)
	if err == nil || !strings.Contains(err.Error(), "zumbra build") {
		t.Fatalf("expected native-build diagnostic, got %v", err)
	}
}
