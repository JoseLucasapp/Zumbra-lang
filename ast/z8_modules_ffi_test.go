package ast_test

import (
	"strings"
	"testing"
	"zumbra/lexer"
	"zumbra/parser"
)

func TestZ8ASTStringPreservesVisibilityAliasAndCName(t *testing.T) {
	source := `
        import "native.zum" as native;
        pub fct wrapper(value) { value; }
        pub extern "C" from "native.c" {
            fct add(left: i32, right: i32) -> i32 as "project_add";
        }
    `
	p := parser.New(lexer.New(source))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	text := program.String()
	for _, expected := range []string{
		`import "native.zum" as native;`,
		`pub var wrapper << fct(value)`,
		`pub extern "C" from "native.c"`,
		`as "project_add"`,
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("missing %q in %s", expected, text)
		}
	}
}
