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

func TestBinaryOperationsMatchEvaluatorAndVM(t *testing.T) {
	sources := []string{
		`var data << bytes(8); writeU16LE(data, 0, 0x1234u16); readU16LE(data, 0);`,
		`var source << bytes(4); source[0] << 1u8; var target << bytes(4); copyBytes(target, 0, source, 0, 4); bytesEqual(source, target);`,
		`var data << bytes(3); data[1] << 1u8; data[2] << 2u8; sha256(data);`,
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
