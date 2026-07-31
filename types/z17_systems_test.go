package types

import (
	"strings"
	"testing"

	"zumbra/lexer"
	"zumbra/parser"
)

func checkZ17Source(t *testing.T, source string) []error {
	t.Helper()
	p := parser.New(lexer.New(source))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	return NewChecker().Check(program)
}

func TestZ17TypeChecksExplicitMemoryAndPointerIndexing(t *testing.T) {
	errors := checkZ17Source(t, `
        var pointer << alloc("i32", 4);
        pointer[0] << 10i32;
        var value << pointer[0];
        pointer << realloc(pointer, 8);
        var borrowed << borrowPointer(pointer);
        show(pointerRead(borrowed, 0));
        releaseBorrow(borrowed);
        free(pointer);
    `)
	if len(errors) != 0 {
		t.Fatalf("type errors: %v", errors)
	}
}

func TestZ17RejectsUnsafeRawPointerOutsideUnsafeBlock(t *testing.T) {
	errors := checkZ17Source(t, `var pointer << pointerFromAddress("u8", 1u64, 1, false);`)
	if len(errors) == 0 {
		t.Fatal("expected an unsafe diagnostic")
	}
	joined := make([]string, 0, len(errors))
	for _, err := range errors {
		joined = append(joined, err.Error())
	}
	if !strings.Contains(strings.Join(joined, "\n"), "pointerFromAddress requires an unsafe block") {
		t.Fatalf("unexpected errors: %v", errors)
	}
}

func TestZ17AllowsRawPointerInsideUnsafeBlock(t *testing.T) {
	errors := checkZ17Source(t, `
        var pointer << nullPointer("u8");
        unsafe {
            pointer << pointerFromAddress("u8", 1u64, 1, false);
        }
    `)
	if len(errors) != 0 {
		t.Fatalf("type errors: %v", errors)
	}
}

func TestZ17RejectsPointerWriteWithIncompatibleElement(t *testing.T) {
	errors := checkZ17Source(t, `
        var pointer << alloc("i32", 1);
        pointerWrite(pointer, 0, "wrong");
    `)
	if len(errors) == 0 {
		t.Fatal("expected an incompatible element diagnostic")
	}
	joined := make([]string, 0, len(errors))
	for _, err := range errors {
		joined = append(joined, err.Error())
	}
	if !strings.Contains(strings.Join(joined, "\n"), "pointerWrite expects i32") {
		t.Fatalf("unexpected errors: %v", errors)
	}
}
