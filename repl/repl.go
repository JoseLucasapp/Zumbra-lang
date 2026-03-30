package repl

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

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
	importedFiles := map[string]bool{}

	baseDir, err := os.Getwd()
	if err != nil {
		baseDir = "."
	}

	for i, v := range builtins.Builtins {
		symbolTable.DefineBuiltin(i, v.Name)
	}

	builtins.SetRouteInvoker(func(handler object.Object, args ...object.Object) (object.Object, error) {
		return vm.InvokeFunction(handler, args, constants, globals)
	})

	for {
		var lines string
		var openBraces int

		_, _ = io.WriteString(out, PROMPT)

		for {
			if !scanner.Scan() {
				if err := scanner.Err(); err != nil {
					fmt.Fprintf(out, "repl error: %s\n", err)
				}
				return
			}

			line := scanner.Text()
			lines += line + "\n"

			openBraces += countChar(line, '{')
			openBraces -= countChar(line, '}')

			if openBraces <= 0 {
				break
			}

			_, _ = io.WriteString(out, ".. ")
		}

		if strings.TrimSpace(lines) == "" {
			continue
		}

		l := lexer.New(lines)
		p := parser.New(l)

		program := p.ParseProgram()
		if len(p.Errors()) != 0 {
			printParserErrors(out, p.Errors())
			continue
		}

		comp := compiler.NewWithStateAndDirAndImports(symbolTable, constants, baseDir, importedFiles)
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
		if lastPopped != nil && lastPopped.Type() != object.NULL_OBJ {
			_, _ = io.WriteString(out, lastPopped.Inspect())
			_, _ = io.WriteString(out, "\n")
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
