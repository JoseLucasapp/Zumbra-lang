package compiler

import (
	"strings"
	"testing"
	"zumbra/lexer"
	"zumbra/parser"
)

func TestZ5ArityValidatesStructConstructors(t *testing.T) {
	p := parser.New(lexer.New(`struct Point { x: int; y: int; } var p << Point(1);`))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatal(p.Errors())
	}
	diagnostics := ValidateProgramArity(program)
	if len(diagnostics) == 0 || !strings.Contains(diagnostics[0].Message, "Point") {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}
}

func TestZ5AnalysisVisitsMethodAndMatchBodies(t *testing.T) {
	p := parser.New(lexer.New(`
        const Max << 3;
        struct Worker {
            value: int;
            fct run() {
                var sum << fct(a) { a; };
                sum();
            }
        }
        enum State { Ready; Done; }
        match(State.Ready) { case State.Ready { Max; } else { 0; } };
    `))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatal(p.Errors())
	}
	diagnostics := ValidateProgramArity(program)
	found := false
	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Message, "sum") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected method body arity diagnostic, got %v", diagnostics)
	}
	_ = AnalyzeProgram(program)
}
