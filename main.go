package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

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

	if len(os.Args) > 2 && os.Args[1] == "build" {
		err := buildZumbra(os.Args[2])
		if err != nil {
			fmt.Printf("Error when trying to build the file: %s\n", err)
		}
		return
	}

	if len(os.Args) > 1 {
		runFile(os.Args[1])
		return
	}

	version := "0.0.9"

	fmt.Printf("\nHello %s!\n", user.Username)
	fmt.Printf("This is the ZUMBRA programming language, version: %s!\n", version)
	fmt.Printf("Feel free to type in commands\n")
	repl.Start(os.Stdin, os.Stdout)
}

func runFile(filename string) {
	data, err := ioutil.ReadFile(filename)
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

	comp := compiler.NewWithStateAndDir(symbolTable, constants, dir) // AQUI
	err = comp.Compile(program)
	if err != nil {
		fmt.Printf("Compilation error: %s\n", err)
		return
	}

	code := comp.Bytecode()
	constants = code.Constants

	machine := vm.NewWithGlobalsStore(code, globals)
	err = machine.Run()
	if err != nil {
		fmt.Printf("Error on VM execution: %s\n", err)
		return
	}

	machine.LastPoppedStackElem()
}

func buildZumbra(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("Error when trying to read the file: %s\n", err)
	}

	source := string(data)
	goCode, err := ZumbraTranspiler(source)
	if err != nil {
		return fmt.Errorf("erro ao transpilar: %w", err)
	}

	if _, err := os.Stat("build"); err == nil {
		err := os.RemoveAll("build")
		if err != nil {
			return fmt.Errorf("Error when trying to remove the file: %w", err)
		}
	}

	err = os.MkdirAll("build", 0755)
	if err != nil {
		return fmt.Errorf("Error when trying to create the file: %w", err)
	}

	err = os.WriteFile("build/main.go", []byte(goCode), 0644)
	if err != nil {
		return fmt.Errorf("Error when trying to write the file: %w", err)
	}

	cmd := exec.Command("go", "build", "-o", "build/zumbra-app", "build/main.go")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func ZumbraTranspiler(zum string) (string, error) {
	lines := strings.Split(zum, "\n")
	var goBody []string

	for _, line := range lines {
		if idx := strings.Index(line, "//"); idx != -1 {
			line = line[:idx]
		}
		line = strings.TrimSpace(line)
		line = strings.TrimSuffix(line, ";")
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "show(") {
			content := strings.TrimPrefix(line, "show(")

			content = strings.TrimSuffix(content, ")")

			args := splitArgs(content)

			if len(args) == 0 {
				goBody = append(goBody, `    fmt.Println()`)
				continue
			}

			if len(args) == 1 && !strings.HasPrefix(args[0], `"`) {
				goBody = append(goBody, fmt.Sprintf(`    fmt.Println(%s)`, args[0]))
				continue
			}

			format := args[0]
			placeholders := strings.Count(format, "{}")
			formatGo := strings.ReplaceAll(format, "{}", "%v")

			if placeholders > 0 && len(args)-1 < placeholders {
				goBody = append(goBody, fmt.Sprintf(`    fmt.Println(%q)`, format))
				continue
			}

			line := fmt.Sprintf(`    fmt.Println(fmt.Sprintf(%s`, formatGo)

			if len(args) > 1 {
				line += ", " + strings.Join(args[1:], ", ")
			}
			line += "))"
			goBody = append(goBody, line)
		}

		if strings.HasPrefix(line, "var ") {
			line = strings.ReplaceAll(line, "<<", "=")
			goBody = append(goBody, "    "+line)
		}

		if strings.HasPrefix(line, "//") {
			goBody = append(goBody, "    "+line)
		}

	}

	return fmt.Sprintf(
		`package main
		import "fmt"

		func main() {
			%s
		}
	`, strings.Join(goBody, "\n")), nil
}

func splitArgs(input string) []string {
	var args []string
	var curr strings.Builder
	inStr := false

	for i := 0; i < len(input); i++ {
		ch := input[i]
		if ch == '"' {
			inStr = !inStr
		}

		if ch == ',' && !inStr {
			args = append(args, strings.TrimSpace(curr.String()))
			curr.Reset()
		} else {
			curr.WriteByte(ch)
		}
	}
	if curr.Len() > 0 {
		args = append(args, strings.TrimSpace(curr.String()))
	}
	return args
}
