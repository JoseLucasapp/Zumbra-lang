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

func TestMemoryCollectionsMatchEvaluatorAndVM(t *testing.T) {
	sources := []string{
		`var memory << bytes(8); memory[3] << 0xA9u8; memory[3];`,
		`var values << arrayOf("i16", 4); values[1] << -20i16; values[1];`,
		`var memory << bytes(8); var view << slice(memory, 2, 6); view[0] << 0x42u8; memory[2];`,
	}
	for _, source := range sources {
		p := parser.New(lexer.New(source))
		program := p.ParseProgram()
		if len(p.Errors()) != 0 {
			t.Fatalf("parser errors: %v", p.Errors())
		}
		evalResult := evaluator.Eval(program, object.NewEnvironment())
		compiled := compiler.New()
		if err := compiled.Compile(program); err != nil {
			t.Fatal(err)
		}
		machine := vm.New(compiled.Bytecode())
		if err := machine.Run(); err != nil {
			t.Fatal(err)
		}
		vmResult := machine.LastPoppedStackElem()
		if evalResult.Type() != vmResult.Type() || evalResult.Inspect() != vmResult.Inspect() {
			t.Fatalf("mismatch for %q: evaluator=%s/%s vm=%s/%s", source, evalResult.Type(), evalResult.Inspect(), vmResult.Type(), vmResult.Inspect())
		}
	}
}
