package compiler

import (
	"testing"

	"zumbra/code"
	"zumbra/lexer"
	"zumbra/parser"
)

func TestZ17TopLevelUnsafeVariablesUseGlobalStorage(t *testing.T) {
	p := parser.New(lexer.New(`unsafe { var pointer << nullPointer("u8"); show(pointer); }`))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	compiler := New()
	if err := compiler.Compile(program); err != nil {
		t.Fatal(err)
	}
	instructions := compiler.Bytecode().Instructions
	setGlobal := false
	getGlobal := false
	for offset := 0; offset < len(instructions); {
		op := code.Opcode(instructions[offset])
		definition, err := code.Lookup(byte(op))
		if err != nil {
			t.Fatal(err)
		}
		if op == code.OpSetGlobal {
			setGlobal = true
		}
		if op == code.OpGetGlobal {
			getGlobal = true
		}
		_, read := code.ReadOperands(definition, instructions[offset+1:])
		offset += 1 + read
	}
	if !setGlobal || !getGlobal {
		t.Fatalf("top-level unsafe variable must use global storage: %s", instructions.String())
	}
}
