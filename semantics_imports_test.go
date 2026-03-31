package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"zumbra/compiler"
	"zumbra/lexer"
	"zumbra/object"
	"zumbra/object/builtins"
	"zumbra/parser"
	"zumbra/vm"
)

func TestImportSemantics_CurrentOfficialPath(t *testing.T) {
	t.Run("single import exposes imported symbol", func(t *testing.T) {
		dir := t.TempDir()

		writeFile(t, filepath.Join(dir, "mod.zum"), `
var x << 10;
`)

		result := compileAndRunInDir(t, dir, `
import "mod.zum";
x;
`)
		assertInspect(t, result, "10")
	})

	t.Run("nested import works", func(t *testing.T) {
		dir := t.TempDir()

		writeFile(t, filepath.Join(dir, "b.zum"), `
var valueB << 20;
`)

		writeFile(t, filepath.Join(dir, "a.zum"), `
import "b.zum";
var valueA << valueB + 1;
`)

		result := compileAndRunInDir(t, dir, `
import "a.zum";
valueA;
`)
		assertInspect(t, result, "21")
	})

	t.Run("relative nested import works", func(t *testing.T) {
		dir := t.TempDir()

		writeFile(t, filepath.Join(dir, "shared", "base.zum"), `
var baseValue << 7;
`)

		writeFile(t, filepath.Join(dir, "pkg", "feature.zum"), `
import "../shared/base.zum";
var featureValue << baseValue + 5;
`)

		result := compileAndRunInDir(t, dir, `
import "pkg/feature.zum";
featureValue;
`)
		assertInspect(t, result, "12")
	})

	t.Run("reimport same module in same compile state does not break symbols", func(t *testing.T) {
		dir := t.TempDir()

		writeFile(t, filepath.Join(dir, "mod.zum"), `
var x << 10;
`)

		result := compileAndRunInDir(t, dir, `
import "mod.zum";
import "mod.zum";
x;
`)
		assertInspect(t, result, "10")
	})

	t.Run("import parser error is surfaced", func(t *testing.T) {
		dir := t.TempDir()

		writeFile(t, filepath.Join(dir, "broken.zum"), `
var x << ;
`)

		_, err := compileOnlyInDir(dir, `
import "broken.zum";
`)

		if err == nil {
			t.Fatalf("expected compile error, got nil")
		}

		msg := err.Error()
		if !strings.Contains(msg, "could not parse imported file") {
			t.Fatalf("wrong compile error, got=%q", msg)
		}
	})

	t.Run("import cycle fails cleanly", func(t *testing.T) {
		dir := t.TempDir()

		writeFile(t, filepath.Join(dir, "a.zum"), `
import "b.zum";
var aValue << 1;
`)

		writeFile(t, filepath.Join(dir, "b.zum"), `
import "a.zum";
var bValue << 2;
`)

		_, err := compileOnlyInDir(dir, `
import "a.zum";
aValue;
`)

		if err == nil {
			t.Fatalf("expected compile error for cyclic import, got nil")
		}

		msg := err.Error()
		if !strings.Contains(msg, "cyclic import") && !strings.Contains(msg, "import cycle") {
			t.Fatalf("expected cyclic import style error, got=%q", msg)
		}
	})
}

func compileOnlyInDir(baseDir string, source string) (*compiler.Compiler, error) {
	l := lexer.New(source)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		return nil, parseErr(p.Errors())
	}

	symbolTable := compiler.NewSymbolTable()
	for i, v := range builtins.Builtins {
		symbolTable.DefineBuiltin(i, v.Name)
	}

	comp := compiler.NewWithStateAndDir(symbolTable, []object.Object{}, baseDir)
	if err := comp.Compile(program); err != nil {
		return nil, err
	}

	return comp, nil
}

func compileAndRunInDir(t *testing.T, baseDir string, source string) object.Object {
	t.Helper()

	comp, err := compileOnlyInDir(baseDir, source)
	if err != nil {
		t.Fatalf("compile error: %s", err)
	}

	machine := vm.New(comp.Bytecode())
	if err := machine.Run(); err != nil {
		t.Fatalf("vm error: %s", err)
	}

	result := machine.LastPoppedStackElem()
	if result == nil {
		t.Fatalf("last popped stack elem is nil")
	}

	return result
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir failed for %s: %s", path, err)
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file failed for %s: %s", path, err)
	}
}

func assertInspect(t *testing.T, obj object.Object, want string) {
	t.Helper()

	got := obj.Inspect()
	if got != want {
		t.Fatalf("wrong result. want=%q, got=%q", want, got)
	}
}

type parseErr []string

func (p parseErr) Error() string {
	return strings.Join(p, "\n")
}
