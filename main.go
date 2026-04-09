package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"

	"zumbra/compiler"
	"zumbra/lexer"
	"zumbra/object"
	"zumbra/object/builtins"
	"zumbra/parser"
	"zumbra/repl"
	"zumbra/semantic"
	"zumbra/transpiler"
	"zumbra/types"
	"zumbra/vm"
)

const version = "0.1.4"

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
	fmt.Println("  zumbra version")
}

func runFile(filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error when trying to read the file: %s\n", err)
		os.Exit(1)
	}

	source := string(data)
	constants := []object.Object{}
	globals := make([]object.Object, vm.GlobalSize)
	symbolTable := compiler.NewSymbolTable()

	for i, v := range builtins.Builtins {
		symbolTable.DefineBuiltin(i, v.Name)
	}

	builtins.SetRouteInvoker(func(handler object.Object, args ...object.Object) (object.Object, error) {
		return vm.InvokeFunction(handler, args, constants, globals)
	})

	l := lexer.New(source)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		fmt.Printf("Parsing errors in %s:\n", filename)
		for _, msg := range p.Errors() {
			fmt.Println("\t" + msg)
		}
		return
	}

	semResult, semErrs := semantic.AnalyzeModule(filename, program)
	if len(semErrs) != 0 {
		fmt.Printf("Semantic errors in %s:\n", filename)
		for _, err := range semErrs {
			fmt.Println("\t" + err.Error())
		}
		return
	}

	if semResult != nil && len(semResult.Warnings) > 0 {
		fmt.Printf("Semantic warnings in %s:\n", filename)
		for _, w := range semResult.Warnings {
			fmt.Println("\t" + w.Message)
		}
	}

	typeErrs := types.AnalyzeModule(filename, program)
	if len(typeErrs) != 0 {
		fmt.Printf("Type errors in %s:\n", filename)
		for _, err := range typeErrs {
			fmt.Println("\t" + err.Error())
		}
		return
	}

	absPath, err := filepath.Abs(filename)
	if err != nil {
		fmt.Printf("Path error: %s\n", err)
		return
	}

	dir := filepath.Dir(absPath)

	comp := compiler.NewWithStateAndDir(symbolTable, constants, dir)

	if diagProvider, ok := any(comp).(interface {
		Diagnostics() []error
	}); ok {
		diags := diagProvider.Diagnostics()
		if len(diags) > 0 {
			fmt.Printf("Compiler diagnostics in %s:\n", filename)
			for _, d := range diags {
				fmt.Println("\t" + d.Error())
			}
		}
	}

	if err := comp.Compile(program); err != nil {
		fmt.Printf("Compilation error: %s\n", err)
		return
	}

	code := comp.Bytecode()
	constants = code.Constants

	machine := vm.NewWithGlobalsStore(code, globals)
	if err := machine.Run(); err != nil {
		fmt.Printf("Error on VM execution: %s\n", err)
		return
	}
}

func buildZumbra(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("error when trying to read the file: %w", err)
	}

	source := string(data)
	goCode, err := transpiler.ZumbraTranspiler(source)
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
