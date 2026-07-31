package main

import (
	"path/filepath"
	"testing"

	"zumbra/compiler"
	"zumbra/object"
	"zumbra/object/builtins"
	"zumbra/pipeline"
	"zumbra/vm"
)

func TestZ8ModuleExampleCompilesAndRunsOnVM(t *testing.T) {
	filename := filepath.Join("code_examples", "core", "modules.zum")
	result, diagnostics := pipeline.BuildFile(filename, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("pipeline diagnostics: %s", pipeline.FormatDiagnostics(diagnostics))
	}
	symbolTable := compiler.NewSymbolTable()
	for index, builtin := range builtins.Builtins {
		symbolTable.DefineBuiltin(index, builtin.Name)
	}
	comp := compiler.NewWithStateAndDir(symbolTable, []object.Object{}, filepath.Dir(filename))
	if err := comp.CompilePipeline(result); err != nil {
		t.Fatal(err)
	}
	machine := vm.New(comp.Bytecode())
	if err := machine.Run(); err != nil {
		t.Fatal(err)
	}
}
