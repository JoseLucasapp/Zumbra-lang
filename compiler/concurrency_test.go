package compiler

import (
	"testing"

	"zumbra/code"
	"zumbra/object"
)

func TestCompilerEmitsSpawnAndAwaitOpcodes(t *testing.T) {
	program := parse(`fct answer(){42;} var task << spawn answer(); await task;`)
	compiler := New()
	if err := compiler.Compile(program); err != nil {
		t.Fatal(err)
	}
	instructions := compiler.Bytecode().Instructions
	foundSpawn := false
	foundAwait := false
	for offset := 0; offset < len(instructions); {
		definition, err := code.Lookup(instructions[offset])
		if err != nil {
			t.Fatal(err)
		}
		if instructions[offset] == byte(code.OpSpawn) {
			foundSpawn = true
		}
		if instructions[offset] == byte(code.OpAwait) {
			foundAwait = true
		}
		_, read := code.ReadOperands(definition, instructions[offset+1:])
		offset += 1 + read
	}
	if !foundSpawn || !foundAwait {
		t.Fatalf("expected OpSpawn and OpAwait, got %s", instructions.String())
	}
}

func TestCompilerMarksAsyncFunction(t *testing.T) {
	program := parse(`var answer << async fct(){42;}; answer();`)
	compiler := New()
	if err := compiler.Compile(program); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, constant := range compiler.Bytecode().Constants {
		if function, ok := constant.(*object.CompiledFunction); ok && function.Async {
			found = true
		}
	}
	if !found {
		t.Fatal("compiled async function was not marked async")
	}
}
