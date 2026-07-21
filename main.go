package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"

	"zumbra/compiler"
	"zumbra/object"
	"zumbra/object/builtins"
	"zumbra/pipeline"
	"zumbra/repl"
	"zumbra/transpiler"
	"zumbra/vm"
)

const version = "0.1.6"

func main() {
	currentUser, err := user.Current()
	if err != nil {
		panic(err)
	}

	args := os.Args[1:]

	if len(args) > 0 {
		switch args[0] {
		case "build":
			if len(args) < 2 {
				printUsage()
				os.Exit(1)
			}

			if err := buildZumbra(args[1]); err != nil {
				fmt.Printf("Error when trying to build the file: %s\n", err)
				os.Exit(1)
			}
			return

		case "run":
			if len(args) < 2 {
				printUsage()
				os.Exit(1)
			}
			runFile(args[1])
			return

		case "ir":
			if len(args) < 2 {
				printUsage()
				os.Exit(1)
			}
			mode := "optimized"
			if len(args) > 2 {
				mode = args[2]
			}
			dumpIR(args[1], mode)
			return

		case "check":
			if len(args) < 2 {
				printUsage()
				os.Exit(1)
			}
			checkFile(args[1])
			return

		case "version", "--version", "-v":
			fmt.Println(version)
			return

		case "help", "--help", "-h":
			printUsage()
			return

		default:
			runFile(args[0])
			return
		}
	}

	fmt.Printf("\nHello %s!\n", currentUser.Username)
	fmt.Printf("This is the ZUMBRA programming language, version: %s!\n", version)
	fmt.Printf("Feel free to type in commands\n")
	repl.Start(os.Stdin, os.Stdout)
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  zumbra <file.zum>")
	fmt.Println("  zumbra run <file.zum>")
	fmt.Println("  zumbra build <file.zum>")
	fmt.Println("  zumbra check <file.zum>")
	fmt.Println("  zumbra ir <file.zum> [hir|mir|optimized]")
	fmt.Println("  zumbra version")
}

func runFile(filename string) {
	result, diagnostics := pipeline.BuildFile(filename, pipeline.Options{Optimize: true})
	if len(diagnostics) > 0 {
		printPipelineDiagnostics(filename, diagnostics)
		return
	}
	for _, warning := range result.Warnings {
		fmt.Printf("%s warning in %s: %s\n", warning.Stage, filename, warning.Message)
	}

	constants := []object.Object{}
	globals := make([]object.Object, vm.GlobalSize)
	symbolTable := compiler.NewSymbolTable()
	for i, v := range builtins.Builtins {
		symbolTable.DefineBuiltin(i, v.Name)
	}
	builtins.SetRouteInvoker(func(handler object.Object, args ...object.Object) (object.Object, error) {
		return vm.InvokeFunction(handler, args, constants, globals)
	})

	absPath, err := filepath.Abs(filename)
	if err != nil {
		fmt.Printf("Path error: %s\n", err)
		return
	}
	comp := compiler.NewWithStateAndDir(symbolTable, constants, filepath.Dir(absPath))
	if err := comp.CompilePipeline(result); err != nil {
		fmt.Printf("Compilation error: %s\n", err)
		return
	}
	if diags := comp.Warnings(); len(diags) > 0 {
		fmt.Printf("Compiler diagnostics in %s:\n", filename)
		for _, d := range diags {
			fmt.Println("\t" + d.Message)
		}
	}

	code := comp.Bytecode()
	machine := vm.NewWithGlobalsStore(code, globals)
	if err := machine.Run(); err != nil {
		fmt.Printf("Error on VM execution: %s\n", err)
	}
}

func checkFile(filename string) {
	result, diagnostics := pipeline.BuildFile(filename, pipeline.Options{Optimize: true})
	if len(diagnostics) > 0 {
		printPipelineDiagnostics(filename, diagnostics)
		return
	}
	for _, warning := range result.Warnings {
		fmt.Printf("%s warning: %s\n", warning.Stage, warning.Message)
	}
	fmt.Printf("OK: %s\n", filename)
}

func dumpIR(filename, mode string) {
	optimize := mode == "optimized"
	result, diagnostics := pipeline.BuildFile(filename, pipeline.Options{Optimize: optimize})
	if len(diagnostics) > 0 {
		printPipelineDiagnostics(filename, diagnostics)
		return
	}
	switch mode {
	case "hir":
		fmt.Print(result.DumpHIR())
	case "mir", "optimized":
		fmt.Print(result.DumpMIR())
	default:
		fmt.Printf("Unknown IR mode %q. Use hir, mir or optimized.\n", mode)
	}
}

func printPipelineDiagnostics(filename string, diagnostics []pipeline.Diagnostic) {
	fmt.Printf("Pipeline errors in %s:\n", filename)
	for _, diagnostic := range diagnostics {
		fmt.Println("\t" + diagnostic.Error())
	}
}

func buildZumbra(filename string) error {
	result, diagnostics := pipeline.BuildFile(filename, pipeline.Options{Optimize: true})
	if len(diagnostics) > 0 {
		return fmt.Errorf("pipeline failed:\n%s", pipeline.FormatDiagnostics(diagnostics))
	}
	goCode, err := transpiler.ZumbraTranspilerPipeline(result)
	if err != nil {
		return fmt.Errorf("error when transpiling: %w", err)
	}

	if _, err := os.Stat("build"); err == nil {
		if err := os.RemoveAll("build"); err != nil {
			return fmt.Errorf("error when trying to remove build: %w", err)
		}
	}

	if err := os.MkdirAll("build", 0o755); err != nil {
		return fmt.Errorf("error when trying to create build dir: %w", err)
	}

	if err := os.WriteFile("build/main.go", []byte(goCode), 0o644); err != nil {
		return fmt.Errorf("error when trying to write main.go: %w", err)
	}

	goModContent := `
module zumbra-generated

go 1.21

require (
	github.com/go-sql-driver/mysql v1.9.2
	github.com/golang-jwt/jwt/v5 v5.2.2
	github.com/lib/pq v1.10.9
	github.com/redis/go-redis/v9 v9.6.1
)
`
	if err := os.WriteFile("build/go.mod", []byte(goModContent), 0o644); err != nil {
		return fmt.Errorf("error when trying to write go.mod: %w", err)
	}

	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = "build"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running go mod tidy: %w", err)
	}

	cmd = exec.Command("go", "build", "-o", "zumbra-app", "main.go")
	cmd.Dir = "build"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running go build: %w", err)
	}

	return nil
}
