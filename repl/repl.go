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
	"zumbra/semantic"
	"zumbra/types"
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

	semanticResolver := semantic.NewResolver()
	typeChecker := types.NewChecker()

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

		semResult, semErrs := semantic.AnalyzeWithResolver(semanticResolver, program)
		if len(semErrs) != 0 {
			printSemanticErrors(out, semErrs)
			continue
		}
		printSemanticWarnings(out, semResult)

		typeErrs := types.AnalyzeWithChecker(typeChecker, program)
		if len(typeErrs) != 0 {
			printTypeErrors(out, typeErrs)
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

func printSemanticErrors(out io.Writer, errors []error) {
	io.WriteString(out, beer)
	io.WriteString(out, "Woops! Semantic analysis found some issues here!\n")
	io.WriteString(out, "Semantic errors:\n")
	for _, err := range errors {
		io.WriteString(out, "\t"+err.Error()+"\n")
	}
}

func printSemanticWarnings(out io.Writer, result *semantic.Result) {
	if result == nil || len(result.Warnings) == 0 {
		return
	}

	io.WriteString(out, "Semantic warnings:\n")
	for _, w := range result.Warnings {
		io.WriteString(out, "\t"+w.Message+"\n")
	}
}

func printTypeErrors(out io.Writer, errors []error) {
	io.WriteString(out, beer)
	io.WriteString(out, "Woops! Type checking found some issues here!\n")
	io.WriteString(out, "Type errors:\n")
	for _, err := range errors {
		io.WriteString(out, "\t"+err.Error()+"\n")
	}
}

const beer = `
█▄▀▄▀▄█
█░▀░▀░█▄
█░▀░░░█─█
█░░░▀░█▄▀
▀▀▀▀▀▀▀
`
