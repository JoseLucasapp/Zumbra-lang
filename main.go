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
	"zumbra/transpiler"
	"zumbra/vm"
)

func main() {
	currentUser, err := user.Current()
	if err != nil {
		panic(err)
	}

	if len(os.Args) > 1 && os.Args[1] == "build" {
		if len(os.Args) < 3 {
			fmt.Println("usage: zumbra build <file.zum>")
			os.Exit(1)
		}

		if err := buildZumbra(os.Args[2]); err != nil {
			fmt.Printf("Error when trying to build the file: %s\n", err)
			os.Exit(1)
		}
		return
	}

	if len(os.Args) > 1 {
		runFile(os.Args[1])
		return
	}

	version := "0.1.0"

	fmt.Printf("\nHello %s!\n", currentUser.Username)
	fmt.Printf("This is the ZUMBRA programming language, version: %s!\n", version)
	fmt.Printf("Feel free to type in commands\n")
	repl.Start(os.Stdin, os.Stdout)
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
		fmt.Println("Parsing errors:")
		for _, msg := range p.Errors() {
			fmt.Println("\t" + msg)
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

	if err := os.MkdirAll("build", 0755); err != nil {
		return fmt.Errorf("error when trying to create build dir: %w", err)
	}

	if err := os.WriteFile("build/main.go", []byte(goCode), 0644); err != nil {
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
	if err := os.WriteFile("build/go.mod", []byte(goModContent), 0644); err != nil {
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
