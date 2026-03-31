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

func TestRunnableCodeExamplesParseCompileAndRun(t *testing.T) {
	root := "code_examples"

	nonRunnableDirs := map[string]bool{
		"http":         true,
		"database":     true,
		"db":           true,
		"mysql":        true,
		"postgres":     true,
		"redis":        true,
		"supabase":     true,
		"rest":         true,
		"integrations": true,
		"experimental": true,
	}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if filepath.Ext(path) != ".zum" {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		parts := strings.Split(rel, string(filepath.Separator))
		if len(parts) > 0 && nonRunnableDirs[parts[0]] {
			return nil
		}

		t.Run(normalizeExampleTestName(path), func(t *testing.T) {
			sourceBytes, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read example %s: %s", path, err)
			}

			l := lexer.New(string(sourceBytes))
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors in %s:\n\t%s", path, strings.Join(p.Errors(), "\n\t"))
			}

			symbolTable := compiler.NewSymbolTable()
			for i, v := range builtins.Builtins {
				symbolTable.DefineBuiltin(i, v.Name)
			}

			comp := compiler.NewWithStateAndDir(symbolTable, []object.Object{}, filepath.Dir(path))
			if err := comp.Compile(program); err != nil {
				t.Fatalf("compiler error in %s: %s", path, err)
			}

			machine := vm.New(comp.Bytecode())
			if err := machine.Run(); err != nil {
				t.Fatalf("vm error in %s: %s", path, err)
			}
		})

		return nil
	})

	if err != nil {
		t.Fatalf("failed to walk code_examples: %s", err)
	}
}

func normalizeExampleTestName(path string) string {
	name := strings.ReplaceAll(path, string(filepath.Separator), "_")
	name = strings.TrimSuffix(name, filepath.Ext(name))
	return name
}
