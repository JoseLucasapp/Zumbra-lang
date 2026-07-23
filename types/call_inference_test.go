package types

import (
	"strings"
	"testing"

	"zumbra/ast"
	"zumbra/lexer"
	"zumbra/parser"
)

func analyzeCallInference(t *testing.T, source string) (*ast.Program, *Analysis, []error) {
	t.Helper()
	p := parser.New(lexer.New(source))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	analysis, diagnostics := AnalyzeWithInfo(program)
	return program, analysis, diagnostics
}

func TestCallInferenceRefinesConcurrencyFunctions(t *testing.T) {
	program, analysis, diagnostics := analyzeCallInference(t, `
        fct square(value) {
            sleepMs(1);
            value * value;
        }

        fct produce(messages) {
            send(messages, 7);
            closeChannel(messages);
            return;
        }

        fct count(counter, amount) {
            var index << 0;
            while (index < amount) {
                atomicAdd(counter, 1);
                index << index + 1;
            }
            return;
        }

        var calculation << spawn square(6);
        var messages << channel(2);
        var producer << spawn produce(messages);
        var counter << atomicInt(0);
        var worker << spawn count(counter, 10);
    `)
	if len(diagnostics) != 0 {
		t.Fatalf("type diagnostics: %v", diagnostics)
	}

	expected := map[string]string{
		"square":      "fct(int) -> int",
		"produce":     "fct(channel<int>) -> null",
		"count":       "fct(atomic_int,int) -> null",
		"calculation": "task<int>",
		"messages":    "channel<int>",
		"producer":    "task<null>",
		"counter":     "atomic_int",
		"worker":      "task<null>",
	}
	for name, want := range expected {
		got, ok := analysis.Global(name)
		if !ok {
			t.Fatalf("missing global %s", name)
		}
		if got.String() != want {
			t.Fatalf("%s: expected %s, got %s", name, want, got.String())
		}
	}

	messagesStatement := program.Statements[4].(*ast.VarStatement)
	if got := analysis.TypeOf(messagesStatement.Value).String(); got != "channel<int>" {
		t.Fatalf("channel initializer was not refined: %s", got)
	}
}

func TestOrdinaryCallRefinesFunctionAndReturn(t *testing.T) {
	_, analysis, diagnostics := analyzeCallInference(t, `
        var identity << fct(value) { value; };
        var result << identity(42);
    `)
	if len(diagnostics) != 0 {
		t.Fatalf("type diagnostics: %v", diagnostics)
	}
	functionType, _ := analysis.Global("identity")
	resultType, _ := analysis.Global("result")
	if functionType.String() != "fct(int) -> int" {
		t.Fatalf("unexpected function type: %s", functionType)
	}
	if resultType.String() != "int" {
		t.Fatalf("unexpected result type: %s", resultType)
	}
}

func TestCallInferenceRejectsIncompatibleReuse(t *testing.T) {
	_, _, diagnostics := analyzeCallInference(t, `
        var identity << fct(value) { value; };
        identity(42);
        identity("zumbra");
    `)
	if len(diagnostics) == 0 {
		t.Fatal("expected incompatible monomorphic reuse to be rejected")
	}
	joined := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		joined = append(joined, diagnostic.Error())
	}
	if !strings.Contains(strings.Join(joined, "\n"), "argument 1 expects int, got string") {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}
}

func TestMethodCallRefinesMethodParameters(t *testing.T) {
	_, analysis, diagnostics := analyzeCallInference(t, `
        struct Box {
            value: int;
            fct echo(input) { input; }
        }
        var box << Box(1);
        var result << box.echo(9);
    `)
	if len(diagnostics) != 0 {
		t.Fatalf("type diagnostics: %v", diagnostics)
	}
	boxType, ok := analysis.Named("Box")
	if !ok {
		t.Fatal("missing Box type")
	}
	method := boxType.Methods["echo"]
	if method == nil || method.String() != "fct(int) -> int" {
		t.Fatalf("unexpected method type: %v", method)
	}
	result, _ := analysis.Global("result")
	if result.String() != "int" {
		t.Fatalf("unexpected method result: %s", result)
	}
}
