package conformance

import (
	"testing"
	"zumbra/compiler"
	"zumbra/evaluator"
	"zumbra/lexer"
	"zumbra/object"
	"zumbra/parser"
	"zumbra/vm"
)

func TestIndexAssignmentMatchesEvaluatorAndVM(t *testing.T) {
	sources := []string{
		`var xs << [1, 2, 3]; xs[1] << 42; xs[1];`,
		`var data << {"score": 10}; data["score"] << 99; data["score"];`,
		`var matrix << [[0, 0], [0, 0]]; matrix[1][1] << 7; matrix[1][1];`,
	}

	for _, source := range sources {
		evaluatorResult := indexAssignmentEvaluatorResult(t, source)
		vmResult := indexAssignmentVMResult(t, source)
		if evaluatorResult.Inspect() != vmResult.Inspect() || evaluatorResult.Type() != vmResult.Type() {
			t.Fatalf("runtime mismatch for %q: evaluator=%s/%s vm=%s/%s", source, evaluatorResult.Type(), evaluatorResult.Inspect(), vmResult.Type(), vmResult.Inspect())
		}
	}
}

func indexAssignmentEvaluatorResult(t *testing.T, source string) object.Object {
	t.Helper()
	p := parser.New(lexer.New(source))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	return evaluator.Eval(program, object.NewEnvironment())
}

func indexAssignmentVMResult(t *testing.T, source string) object.Object {
	t.Helper()
	p := parser.New(lexer.New(source))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	compiled := compiler.New()
	if err := compiled.Compile(program); err != nil {
		t.Fatalf("compiler error: %v", err)
	}
	machine := vm.New(compiled.Bytecode())
	if err := machine.Run(); err != nil {
		t.Fatalf("VM error: %v", err)
	}
	return machine.LastPoppedStackElem()
}
