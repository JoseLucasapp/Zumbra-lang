package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"

	"zumbra/compiler"
	"zumbra/lexer"
	"zumbra/object"
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

	for i, v := range object.Builtins {
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

	comp := compiler.NewWithState(symbolTable, constants)
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

	result := machine.LastPoppedStackElem()
	fmt.Println(result.Inspect())
}
