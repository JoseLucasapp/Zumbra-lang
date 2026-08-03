package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"

	"zumbra/cbinding"
	"zumbra/compiler"
	"zumbra/modules"
	"zumbra/nativec"
	"zumbra/object"
	"zumbra/object/builtins"
	"zumbra/pipeline"
	"zumbra/repl"
	"zumbra/vm"
)

const version = "0.14.1"

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

		case "app":
			if err := handleAppCommand(args[1:]); err != nil {
				fmt.Printf("Application error: %s\n", err)
				if isAppUsageError(err) {
					printUsage()
				}
				os.Exit(1)
			}
			return

		case "run":
			if len(args) != 2 {
				printUsage()
				os.Exit(1)
			}
			if err := executeFile(args[1]); err != nil {
				fmt.Printf("Execution error: %s\n", err)
				os.Exit(1)
			}
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
			if err := handleCheckCommand(args[1:]); err != nil {
				fmt.Printf("Check error: %s\n", err)
				os.Exit(1)
			}
			return

		case "fmt":
			if err := handleFormatCommand(args[1:]); err != nil {
				fmt.Printf("Format error: %s\n", err)
				os.Exit(1)
			}
			return

		case "lint":
			if err := handleLintCommand(args[1:]); err != nil {
				fmt.Printf("Lint error: %s\n", err)
				os.Exit(1)
			}
			return

		case "doc":
			if err := handleDocCommand(args[1:]); err != nil {
				fmt.Printf("Documentation error: %s\n", err)
				os.Exit(1)
			}
			return

		case "project":
			if err := handleProjectCommand(args[1:]); err != nil {
				fmt.Printf("Project error: %s\n", err)
				os.Exit(1)
			}
			return

		case "profile":
			if err := handleProfileCommand(args[1:]); err != nil {
				fmt.Printf("Profile error: %s\n", err)
				os.Exit(1)
			}
			return

		case "lsp":
			if err := handleLSPCommand(args[1:]); err != nil {
				fmt.Fprintf(os.Stderr, "LSP error: %s\n", err)
				os.Exit(1)
			}
			return

		case "modules":
			if len(args) < 2 {
				printUsage()
				os.Exit(1)
			}
			dumpModules(args[1])
			return

		case "bind-c":
			header, output, options, err := parseBindCArguments(args[1:])
			if err != nil {
				fmt.Printf("Binding arguments error: %s\n", err)
				printUsage()
				os.Exit(1)
			}
			if err := generateCBinding(header, output, options); err != nil {
				fmt.Printf("C binding error: %s\n", err)
				os.Exit(1)
			}
			return

		case "version", "--version", "-v":
			fmt.Println(version)
			return

		case "help", "--help", "-h":
			printUsage()
			return

		default:
			if err := executeFile(args[0]); err != nil {
				fmt.Printf("Execution error: %s\n", err)
				os.Exit(1)
			}
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
	fmt.Println("  zumbra build [--release] [--emit-c] [--sanitize <name>] [--compiler <name>] [--link <file>] [--include <dir>] [--library-dir <dir>] [-l <name>] [-o <path>] <file.zum>")
	fmt.Println("  zumbra app inspect [--manifest <zumbra.toml>]")
	fmt.Println("  zumbra app run [--manifest <zumbra.toml>]")
	fmt.Println("  zumbra app build [--manifest <zumbra.toml>] [--target <os>] [--arch <arch>] [--release|--debug] [--compiler <name>] [-o <path>]")
	fmt.Println("  zumbra app package [--manifest <zumbra.toml>] [--target <linux|windows|macos>] [--arch <amd64|arm64>] [--format <format>] [--binary <path>] [--output-dir <dir>] [--appimagetool <path>] [--appimage-runtime <path>] [--makensis <path>] [--symbols] [--sign <identity>]")
	fmt.Println("  zumbra app doctor [--manifest <zumbra.toml>] [--target <os>] [--arch <arch>] [--format <format>] [--binary <path>] [--appimagetool <path>] [--appimage-runtime <path>] [--makensis <path>] [--json]")
	fmt.Println("  zumbra check [--json] <file.zum>")
	fmt.Println("  zumbra fmt [--check] [--stdout] [--indent <spaces>] [paths...]")
	fmt.Println("  zumbra lint [--json] [--deny-warnings] [--no-pipeline] [--no-public-docs] [paths...]")
	fmt.Println("  zumbra doc [--format <markdown|json>] [--private] [-o <path>] [paths...]")
	fmt.Println("  zumbra project <init|info|check|test|run|build|fmt|lint|doc|clean>")
	fmt.Println("  zumbra profile [--runs <n>] [--warmup <n>] [--json] [--cpu-profile <path>] [--heap-profile <path>] <file.zum>")
	fmt.Println("  zumbra lsp [--stdio]")
	fmt.Println("  zumbra modules <file.zum>")
	fmt.Println("  zumbra ir <file.zum> [hir|mir|optimized]")
	fmt.Println("  zumbra bind-c [--link <path>] [--pub] [-o <file.zum>] <header.h>")
	fmt.Println("  zumbra version")
}

func runFile(filename string) {
	if err := executeFile(filename); err != nil {
		fmt.Printf("Execution error: %s\n", err)
	}
}

func executeFile(filename string) error {
	result, diagnostics := pipeline.BuildFile(filename, pipeline.Options{Optimize: true})
	if len(diagnostics) > 0 {
		return fmt.Errorf("pipeline failed:\n%s", pipeline.FormatDiagnostics(diagnostics))
	}
	if result == nil {
		return fmt.Errorf("pipeline returned no result")
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
	absPath, err := filepath.Abs(filename)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}
	comp := compiler.NewWithStateAndDir(symbolTable, constants, filepath.Dir(absPath))
	if err := comp.CompilePipeline(result); err != nil {
		return fmt.Errorf("compile: %w", err)
	}
	if diags := comp.Warnings(); len(diags) > 0 {
		fmt.Printf("Compiler diagnostics in %s:\n", filename)
		for _, d := range diags {
			fmt.Println("\t" + d.Message)
		}
	}

	code := comp.Bytecode()
	callbackInvoker := func(handler object.Object, args ...object.Object) (object.Object, error) {
		return vm.InvokeFunction(handler, args, code.Constants, globals)
	}
	builtins.SetRouteInvoker(callbackInvoker)
	builtins.SetDesktopInvoker(callbackInvoker)
	machine := vm.NewWithGlobalsStore(code, globals)
	if err := machine.Run(); err != nil {
		return fmt.Errorf("VM execution: %w", err)
	}
	return nil
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

func dumpModules(filename string) {
	result, diagnostics := pipeline.BuildFile(filename, pipeline.Options{Optimize: false})
	if len(diagnostics) > 0 {
		printPipelineDiagnostics(filename, diagnostics)
		return
	}
	if result == nil || result.Modules == nil {
		fmt.Printf("No module graph for %s\n", filename)
		return
	}
	base, _ := filepath.Abs(".")
	fmt.Printf("module graph %s\n", displayPath(result.Modules.Entry, base))
	for _, unit := range result.Modules.Units {
		kind := "module"
		if unit.Entry {
			kind = "entry"
		}
		fmt.Printf("  %s %s\n", kind, displayPath(unit.Path, base))
		imports := append([]modules.Import(nil), unit.Imports...)
		sort.Slice(imports, func(i, j int) bool { return imports[i].Path < imports[j].Path })
		for _, imported := range imports {
			label := imported.Alias
			if label == "" {
				label = "<legacy>"
			}
			fmt.Printf("    import %s as %s\n", displayPath(imported.Path, base), label)
		}
		exports := make([]string, 0, len(unit.Exports))
		for name := range unit.Exports {
			exports = append(exports, name)
		}
		sort.Strings(exports)
		if len(exports) > 0 {
			fmt.Printf("    exports %s\n", strings.Join(exports, ", "))
		}
	}
	for _, link := range result.Modules.Links {
		fmt.Printf("  link %s\n", displayPath(link, base))
	}
}

func displayPath(path, base string) string {
	if relative, err := filepath.Rel(base, path); err == nil && !strings.HasPrefix(relative, "..") {
		return filepath.ToSlash(relative)
	}
	return filepath.ToSlash(path)
}

type bindCArguments struct {
	Link   string
	Public bool
}

func parseBindCArguments(arguments []string) (string, string, bindCArguments, error) {
	options := bindCArguments{}
	header := ""
	output := ""
	for index := 0; index < len(arguments); index++ {
		switch argument := arguments[index]; argument {
		case "--link":
			index++
			if index >= len(arguments) {
				return "", "", options, fmt.Errorf("--link requires a source, object or library path")
			}
			options.Link = arguments[index]
		case "--pub":
			options.Public = true
		case "-o", "--output":
			index++
			if index >= len(arguments) {
				return "", "", options, fmt.Errorf("%s requires an output path", argument)
			}
			output = arguments[index]
		default:
			if strings.HasPrefix(argument, "-") {
				return "", "", options, fmt.Errorf("unknown bind-c option %s", argument)
			}
			if header != "" {
				return "", "", options, fmt.Errorf("multiple C headers are not supported in one invocation")
			}
			header = argument
		}
	}
	if header == "" {
		return "", "", options, fmt.Errorf("missing C header")
	}
	return header, output, options, nil
}

func generateCBinding(header, output string, cli bindCArguments) error {
	result, err := cbinding.GenerateFile(header, cbinding.Options{Link: cli.Link, Public: cli.Public})
	if err != nil {
		return err
	}
	for _, diagnostic := range result.Diagnostics {
		fmt.Printf("binding warning: %s\n", diagnostic.Error())
	}
	if len(result.Functions) == 0 {
		return fmt.Errorf("no supported C functions found in %s", header)
	}
	if output == "" {
		fmt.Print(result.Source)
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil && filepath.Dir(output) != "." {
		return fmt.Errorf("create binding output directory: %w", err)
	}
	if err := os.WriteFile(output, []byte(result.Source), 0o644); err != nil {
		return fmt.Errorf("write binding: %w", err)
	}
	fmt.Printf("Generated Zumbra C binding: %s\n", output)
	return nil
}

func printPipelineDiagnostics(filename string, diagnostics []pipeline.Diagnostic) {
	fmt.Printf("Pipeline errors in %s:\n", filename)
	for _, diagnostic := range diagnostics {
		fmt.Println("\t" + diagnostic.Error())
	}
}

type nativeBuildArguments struct {
	Release     bool
	EmitCOnly   bool
	Compiler    string
	Output      string
	Links       []string
	IncludeDirs []string
	LibraryDirs []string
	Libraries   []string
	Sanitizers  []string
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
		case "--sanitize":
			index++
			if index >= len(arguments) {
				return "", options, fmt.Errorf("--sanitize requires address, undefined, thread or leak")
			}
			options.Sanitizers = append(options.Sanitizers, arguments[index])
		case "--compiler":
			index++
			if index >= len(arguments) {
				return "", options, fmt.Errorf("--compiler requires clang, gcc, cc or auto")
			}
			options.Compiler = arguments[index]
		case "--link":
			index++
			if index >= len(arguments) {
				return "", options, fmt.Errorf("--link requires a source, object or library path")
			}
			options.Links = append(options.Links, arguments[index])
		case "--include":
			index++
			if index >= len(arguments) {
				return "", options, fmt.Errorf("--include requires a directory")
			}
			options.IncludeDirs = append(options.IncludeDirs, arguments[index])
		case "--library-dir":
			index++
			if index >= len(arguments) {
				return "", options, fmt.Errorf("--library-dir requires a directory")
			}
			options.LibraryDirs = append(options.LibraryDirs, arguments[index])
		case "-l", "--library":
			index++
			if index >= len(arguments) {
				return "", options, fmt.Errorf("%s requires a library name", argument)
			}
			options.Libraries = append(options.Libraries, arguments[index])
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
		Release:     cli.Release,
		EmitCOnly:   cli.EmitCOnly,
		Compiler:    cli.Compiler,
		Output:      cli.Output,
		BuildDir:    buildDir,
		Links:       cli.Links,
		IncludeDirs: cli.IncludeDirs,
		LibraryDirs: cli.LibraryDirs,
		Libraries:   cli.Libraries,
		Sanitizers:  cli.Sanitizers,
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
