package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"

	"zumbra/compiler"
	"zumbra/lexer"
	"zumbra/object"
	"zumbra/object/builtins"
	"zumbra/parser"
	"zumbra/repl"
	"zumbra/vm"
)

func main() {
	user, err := user.Current()
	if err != nil {
		panic(err)
	}

	if len(os.Args) > 1 {
		runFile(os.Args[1])
		return
	}

	fmt.Printf("Hello %s! This is the ZUMBRA programming language!\n", user.Username)
	fmt.Printf("Feel free to type in commands\n")
	repl.Start(os.Stdin, os.Stdout)
}

func runFile(filename string) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Printf("Erro ao ler o arquivo: %s\n", err)
		os.Exit(1)
	}

	source := string(data)
	constants := []object.Object{}
	globals := make([]object.Object, vm.GlobalSize)
	symbolTable := compiler.NewSymbolTable()

	for i, v := range builtins.Builtins {
		symbolTable.DefineBuiltin(i, v.Name)
	}

	l := lexer.New(source)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		fmt.Println("Erros de parsing:")
		for _, msg := range p.Errors() {
			fmt.Println("\t" + msg)
		}
		return
	}

	// Novo trecho:
	absPath, err := filepath.Abs(filename)
	if err != nil {
		fmt.Printf("Erro ao resolver caminho absoluto: %s\n", err)
		return
	}
	dir := filepath.Dir(absPath)

	comp := compiler.NewWithStateAndDir(symbolTable, constants, dir) // AQUI
	err = comp.Compile(program)
	if err != nil {
		fmt.Printf("Erro na compilação: %s\n", err)
		return
	}

	code := comp.Bytecode()
	constants = code.Constants

	machine := vm.NewWithGlobalsStore(code, globals)
	err = machine.Run()
	if err != nil {
		fmt.Printf("Erro na execução da VM: %s\n", err)
		return
	}

	machine.LastPoppedStackElem()
}
