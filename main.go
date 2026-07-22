package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"zumbra/compiler"
	"zumbra/nativec"
	"zumbra/object"
	"zumbra/object/builtins"
	"zumbra/pipeline"
	"zumbra/repl"
	"zumbra/vm"
)

const version = "0.1.7"

func main() {
	currentUser, err := user.Current()
	if err != nil {
		panic(err)
	}

	args := os.Args[1:]

	if len(args) > 0 {
		switch args[0] {
		case "build":
			filename, buildOptions, err := parseBuildArguments(args[1:])
			if err != nil {
				fmt.Printf("Build arguments error: %s\n", err)
				printUsage()
				os.Exit(1)
			}
			if err := buildZumbra(filename, buildOptions); err != nil {
				fmt.Printf("Native build error: %s\n", err)
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
	fmt.Println("  zumbra build [--release] [--emit-c] [--compiler <name>] [-o <path>] <file.zum>")
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

type nativeBuildArguments struct {
	Release   bool
	EmitCOnly bool
	Compiler  string
	Output    string
}

func parseBuildArguments(arguments []string) (string, nativeBuildArguments, error) {
	options := nativeBuildArguments{Compiler: "auto"}
	filename := ""
	for index := 0; index < len(arguments); index++ {
		argument := arguments[index]
		switch argument {
		case "--release":
			options.Release = true
		case "--emit-c":
			options.EmitCOnly = true
		case "--compiler":
			index++
			if index >= len(arguments) {
				return "", options, fmt.Errorf("--compiler requires clang, gcc, cc or auto")
			}
			options.Compiler = arguments[index]
		case "-o", "--output":
			index++
			if index >= len(arguments) {
				return "", options, fmt.Errorf("%s requires an output path", argument)
			}
			options.Output = arguments[index]
		default:
			if strings.HasPrefix(argument, "-") {
				return "", options, fmt.Errorf("unknown build option %s", argument)
			}
			if filename != "" {
				return "", options, fmt.Errorf("multiple input files are not supported yet")
			}
			filename = argument
		}
	}
	if filename == "" {
		return "", options, fmt.Errorf("missing .zum input file")
	}
	return filename, options, nil
}

func buildZumbra(filename string, cli nativeBuildArguments) error {
	result, diagnostics := pipeline.BuildFile(filename, pipeline.Options{Optimize: true})
	if len(diagnostics) > 0 {
		return fmt.Errorf("pipeline failed:\n%s", pipeline.FormatDiagnostics(diagnostics))
	}
	baseName := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	buildDir := filepath.Join("build", "native", baseName)
	buildResult, nativeDiagnostics, err := nativec.Build(result.MIR, nativec.BuildOptions{
		Release:   cli.Release,
		EmitCOnly: cli.EmitCOnly,
		Compiler:  cli.Compiler,
		Output:    cli.Output,
		BuildDir:  buildDir,
	})
	if err != nil {
		return err
	}
	if len(nativeDiagnostics) != 0 {
		var messages strings.Builder
		for _, diagnostic := range nativeDiagnostics {
			messages.WriteString("\t")
			messages.WriteString(diagnostic.Error())
			messages.WriteByte('\n')
		}
		return fmt.Errorf("native backend rejected the program:\n%s", messages.String())
	}
	if buildResult == nil {
		return fmt.Errorf("native backend returned no build result")
	}
	if cli.EmitCOnly {
		fmt.Printf("Generated native C sources in %s\n", buildResult.SourceDir)
		return nil
	}
	mode := "debug"
	if cli.Release {
		mode = "release"
	}
	fmt.Printf("Built %s native executable: %s\n", mode, buildResult.Output)
	fmt.Printf("C compiler: %s\n", buildResult.Compiler)
	return nil
}
