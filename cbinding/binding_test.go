package cbinding

import (
	"strings"
	"testing"

	"zumbra/lexer"
	"zumbra/parser"
)

func TestGeneratePortableBindings(t *testing.T) {
	result := Generate(`
        #include <stdint.h>
        int32_t add(int32_t left, int32_t right);
        const char *name(void);
        void consume(void *data, size_t size);
        int32_t negate(int32_t);
    `, Options{Link: "native.c", Public: true})
	if len(result.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %#v", result.Diagnostics)
	}
	for _, expected := range []string{
		`pub extern "C" from "native.c"`,
		`fct add(left: i32, right: i32) -> i32;`,
		`fct name() -> cstring;`,
		`fct consume(data: ptr, size: usize) -> void;`,
		`fct negate(arg0: i32) -> i32;`,
	} {
		if !strings.Contains(result.Source, expected) {
			t.Fatalf("missing %q in:\n%s", expected, result.Source)
		}
	}
	p := parser.New(lexer.New(result.Source))
	p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("generated binding does not parse: %v\n%s", p.Errors(), result.Source)
	}
}

func TestFunctionPointerNeedsManualBinding(t *testing.T) {
	result := Generate(`int apply(int value, int (*callback)(int));`, Options{})
	if len(result.Diagnostics) == 0 {
		t.Fatal("expected function pointer diagnostic")
	}
}
