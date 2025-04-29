package repl

import (
	"bufio"
	"fmt"
	"io"

	"zumbra/compiler"
	"zumbra/lexer"
	"zumbra/object"
	"zumbra/object/builtins"
	"zumbra/parser"
	"zumbra/vm"
)

const PROMPT = ">> "

func Start(in io.Reader, out io.Writer) {
	scanner := bufio.NewScanner(in)

	constants := []object.Object{}
	globals := make([]object.Object, vm.GlobalSize)
	symbolTable := compiler.NewSymbolTable()

	for i, v := range builtins.Builtins {
		symbolTable.DefineBuiltin(i, v.Name)
	}

	for {
		var lines string
		var openBraces int

		fmt.Printf(PROMPT)
		for scanner.Scan() {
			line := scanner.Text()
			lines += line + "\n"

			openBraces += countChar(line, '{')
			openBraces -= countChar(line, '}')

			if openBraces <= 0 {
				break
			}
			fmt.Printf(".. ")
		}

		l := lexer.New(lines)
		p := parser.New(l)

		program := p.ParseProgram()
		if len(p.Errors()) != 0 {
			printParserErrors(out, p.Errors())
			continue
		}

		comp := compiler.NewWithState(symbolTable, constants)
		err := comp.Compile(program)
		if err != nil {
			fmt.Fprintf(out, "compiler error: %s\n", err)
			continue
		}

		code := comp.Bytecode()
		constants = code.Constants

		machine := vm.NewWithGlobalsStore(code, globals)
		err = machine.Run()
		if err != nil {
			fmt.Fprintf(out, "vm error: %s\n", err)
			continue
		}

		lastPopped := machine.LastPoppedStackElem()
		if lastPopped.Type() != object.NULL_OBJ {
			io.WriteString(out, lastPopped.Inspect())
			io.WriteString(out, "\n")
		}

	}
}

func countChar(s string, ch rune) int {
	count := 0
	for _, c := range s {
		if c == ch {
			count++
		}
	}
	return count
}

func printParserErrors(out io.Writer, errors []string) {
	io.WriteString(out, beer)
	io.WriteString(out, "Woops! We ran into some 'I need a beer' business here!\n")
	io.WriteString(out, "Parser errors:\n")
	for _, msg := range errors {
		io.WriteString(out, "\t"+msg+"\n")
	}
}

const beer = `
█▄▀▄▀▄█
█░▀░▀░█▄
█░▀░░░█─█
█░░░▀░█▄▀
▀▀▀▀▀▀▀
`
