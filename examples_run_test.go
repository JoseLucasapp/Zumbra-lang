package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"zumbra/compiler"
	"zumbra/object"
	"zumbra/object/builtins"
	"zumbra/pipeline"
	"zumbra/vm"
)

func TestRunnableCodeExamplesParseCompileAndRun(t *testing.T) {
	root := "code_examples"

	allowedRunnableDirs := map[string]bool{
		"arrays":     true,
		"async":      true,
		"comparison": true,
		"dicts":      true,
		"errors":     true,
		"imports":    true,
		"math":       true,
		"strings":    true,
	}

	allowedRunnableFiles := map[string]bool{
		"attribute_access.zum": true,
		"date.zum":             true,
		"for.zum":              true,
		"functions.zum":        true,
		"hello_world.zum":      true,
		"parse.zum":            true,
		"show.zum":             true,
		"switch_case.zum":      true,
		"structured_types.zum": true,
		"typed_ir.zum":         true,
		"types.zum":            true,
		"var_names.zum":        true,
		"vars.zum":             true,
		"while.zum":            true,
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

		shouldRun := false

		if len(parts) == 1 {
			shouldRun = allowedRunnableFiles[parts[0]]
		} else if len(parts) > 1 {
			shouldRun = allowedRunnableDirs[parts[0]]
		}

		if !shouldRun {
			return nil
		}

		t.Run(normalizeExampleTestName(path), func(t *testing.T) {
			sourceBytes, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read example %s: %s", path, err)
			}

			result, diagnostics := pipeline.Build(path, string(sourceBytes), pipeline.Options{Optimize: true})
			if len(diagnostics) > 0 {
				t.Fatalf("pipeline errors in %s:\n\t%s", path, pipeline.FormatDiagnostics(diagnostics))
			}

			symbolTable := compiler.NewSymbolTable()
			for i, v := range builtins.Builtins {
				symbolTable.DefineBuiltin(i, v.Name)
			}

			comp := compiler.NewWithStateAndDir(symbolTable, []object.Object{}, filepath.Dir(path))
			if err := comp.CompilePipeline(result); err != nil {
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
