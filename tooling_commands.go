package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"zumbra/diagnostics"
	"zumbra/pipeline"
	"zumbra/tooling/docgen"
	"zumbra/tooling/formatter"
	"zumbra/tooling/lint"
	zumbraLSP "zumbra/tooling/lsp"
	zumbraProfile "zumbra/tooling/profile"
	"zumbra/tooling/project"
)

var errToolingUsage = errors.New("invalid tooling command usage")

func parseToolFlags(flags *flag.FlagSet, arguments []string) (bool, error) {
	if err := flags.Parse(arguments); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return true, nil
		}
		return false, errToolingUsage
	}
	return false, nil
}

func isHelpArgument(argument string) bool {
	return argument == "help" || argument == "--help" || argument == "-h"
}

func printProjectUsage(output io.Writer) {
	fmt.Fprintln(output, "Usage:")
	fmt.Fprintln(output, "  zumbra project init [--dir <path>] [--kind <cli|library|desktop>] [--identifier <reverse-DNS>] [--force] <name>")
	fmt.Fprintln(output, "  zumbra project info [--manifest <zumbra.toml>] [--json]")
	fmt.Fprintln(output, "  zumbra project check [--manifest <zumbra.toml>] [--json] [--tests]")
	fmt.Fprintln(output, "  zumbra project test [--manifest <zumbra.toml>] [--json] [--tests]")
	fmt.Fprintln(output, "  zumbra project run [--manifest <zumbra.toml>]")
	fmt.Fprintln(output, "  zumbra project build [--manifest <zumbra.toml>] [--debug] [build options]")
	fmt.Fprintln(output, "  zumbra project fmt [--manifest <zumbra.toml>]")
	fmt.Fprintln(output, "  zumbra project lint [--manifest <zumbra.toml>]")
	fmt.Fprintln(output, "  zumbra project doc [--manifest <zumbra.toml>]")
	fmt.Fprintln(output, "  zumbra project clean [--manifest <zumbra.toml>]")
}

func printProjectCommandUsage(output io.Writer, command string) error {
	usage := map[string]string{
		"init":  "zumbra project init [--dir <path>] [--kind <cli|library|desktop>] [--identifier <reverse-DNS>] [--force] <name>",
		"info":  "zumbra project info [--manifest <zumbra.toml>] [--json]",
		"check": "zumbra project check [--manifest <zumbra.toml>] [--json] [--tests]",
		"test":  "zumbra project test [--manifest <zumbra.toml>] [--json] [--tests]",
		"run":   "zumbra project run [--manifest <zumbra.toml>]",
		"build": "zumbra project build [--manifest <zumbra.toml>] [--debug] [build options]",
		"fmt":   "zumbra project fmt [--manifest <zumbra.toml>]",
		"lint":  "zumbra project lint [--manifest <zumbra.toml>]",
		"doc":   "zumbra project doc [--manifest <zumbra.toml>]",
		"clean": "zumbra project clean [--manifest <zumbra.toml>]",
	}
	line, ok := usage[command]
	if !ok {
		return fmt.Errorf("unknown project command %q", command)
	}
	fmt.Fprintf(output, "Usage:\n  %s\n", line)
	return nil
}

func handleFormatCommand(arguments []string) error {
	flags := flag.NewFlagSet("fmt", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	check := flags.Bool("check", false, "report files that are not canonically formatted")
	stdout := flags.Bool("stdout", false, "write one formatted file to stdout")
	indent := flags.Int("indent", 4, "indentation width")
	help, err := parseToolFlags(flags, arguments)
	if err != nil {
		return err
	}
	if help {
		return nil
	}
	files, err := formatter.Discover(flags.Args())
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no .zum files found")
	}
	if *stdout && len(files) != 1 {
		return fmt.Errorf("--stdout requires exactly one .zum file")
	}
	changed := []string{}
	for _, filename := range files {
		result, err := formatter.File(filename, !*check && !*stdout, formatter.Options{IndentWidth: *indent})
		if err != nil {
			return err
		}
		if *stdout {
			fmt.Print(result.Source)
			continue
		}
		if result.Changed {
			changed = append(changed, filename)
			if !*check {
				fmt.Printf("formatted %s\n", filename)
			}
		}
	}
	if *check && len(changed) > 0 {
		for _, filename := range changed {
			fmt.Printf("not formatted: %s\n", filename)
		}
		return fmt.Errorf("%d file(s) require zumbra fmt", len(changed))
	}
	if !*stdout {
		fmt.Printf("format: %d file(s), %d changed\n", len(files), len(changed))
	}
	return nil
}

func handleLintCommand(arguments []string) error {
	flags := flag.NewFlagSet("lint", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	jsonOutput := flags.Bool("json", false, "emit JSON diagnostics")
	denyWarnings := flags.Bool("deny-warnings", false, "return failure when warnings are present")
	noPipeline := flags.Bool("no-pipeline", false, "skip semantic and type pipeline checks")
	noPublicDocs := flags.Bool("no-public-docs", false, "disable public API documentation rule")
	maxLineLength := flags.Int("max-line-length", 120, "maximum source line length")
	help, err := parseToolFlags(flags, arguments)
	if err != nil {
		return err
	}
	if help {
		return nil
	}
	result, err := lint.Files(flags.Args(), lint.Options{
		CheckPipeline:     !*noPipeline,
		RequirePublicDocs: !*noPublicDocs,
		MaxLineLength:     *maxLineLength,
	})
	if err != nil {
		return err
	}
	if *jsonOutput {
		payload, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(payload))
	} else {
		for _, item := range result.Diagnostics {
			fmt.Println(item.String())
			if item.Help != "" {
				fmt.Printf("  help: %s\n", item.Help)
			}
		}
		fmt.Printf("lint: %d error(s), %d warning(s), %d info(s)\n", result.Errors, result.Warnings, result.Infos)
	}
	if result.Failed(*denyWarnings) {
		return fmt.Errorf("lint failed")
	}
	return nil
}

func handleDocCommand(arguments []string) error {
	flags := flag.NewFlagSet("doc", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	output := flags.String("output", "", "output file; stdout when omitted")
	flags.StringVar(output, "o", "", "output file")
	format := flags.String("format", "markdown", "markdown or json")
	includePrivate := flags.Bool("private", false, "include private declarations")
	title := flags.String("title", "Zumbra API", "documentation title")
	help, err := parseToolFlags(flags, arguments)
	if err != nil {
		return err
	}
	if help {
		return nil
	}
	files, err := formatter.Discover(flags.Args())
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no .zum files found")
	}
	document, err := docgen.Generate(files, docgen.Options{IncludePrivate: *includePrivate, Title: *title})
	if err != nil {
		return err
	}
	var content []byte
	switch strings.ToLower(*format) {
	case "markdown", "md":
		content = []byte(docgen.Markdown(document))
	case "json":
		content, err = docgen.JSON(document)
		if err == nil {
			content = append(content, '\n')
		}
	default:
		return fmt.Errorf("unknown documentation format %q; use markdown or json", *format)
	}
	if err != nil {
		return err
	}
	if *output == "" {
		_, err = os.Stdout.Write(content)
		return err
	}
	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil && filepath.Dir(*output) != "." {
		return err
	}
	if err := os.WriteFile(*output, content, 0o644); err != nil {
		return err
	}
	fmt.Printf("generated documentation: %s (%d symbols)\n", *output, len(document.Symbols))
	return nil
}

func handleProfileCommand(arguments []string) error {
	flags := flag.NewFlagSet("profile", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	runs := flags.Int("runs", 10, "measured compiler pipeline runs")
	warmup := flags.Int("warmup", 2, "warmup runs")
	noOptimize := flags.Bool("no-optimize", false, "disable MIR optimization")
	jsonOutput := flags.Bool("json", false, "emit JSON report")
	cpuProfile := flags.String("cpu-profile", "", "write a Go CPU pprof file")
	heapProfile := flags.String("heap-profile", "", "write a Go heap pprof file")
	help, err := parseToolFlags(flags, arguments)
	if err != nil {
		return err
	}
	if help {
		return nil
	}
	if flags.NArg() != 1 {
		return fmt.Errorf("profile requires exactly one .zum input file")
	}
	report, err := zumbraProfile.Run(flags.Arg(0), zumbraProfile.Options{
		Runs: *runs, Warmup: *warmup, Optimize: !*noOptimize,
		CPUProfile: *cpuProfile, HeapProfile: *heapProfile,
	})
	if err != nil {
		if report != nil && len(report.Diagnostics) > 0 {
			printPipelineDiagnostics(flags.Arg(0), report.Diagnostics)
		}
		return err
	}
	if *jsonOutput {
		payload, marshalErr := json.MarshalIndent(report, "", "  ")
		if marshalErr != nil {
			return marshalErr
		}
		fmt.Println(string(payload))
		return nil
	}
	fmt.Printf("profile %s: %d runs, average %s, median %s, p95 %s, min %s, max %s\n", report.File, report.Runs, report.Average, report.Median, report.P95, report.Minimum, report.Maximum)
	fmt.Printf("allocations: %d bytes/run, %d allocations/run\n", report.BytesPerRun, report.AllocationsPerRun)
	for _, stage := range report.Stages {
		fmt.Printf("  %-10s %12s/run %6.2f%%\n", stage.Name, stage.Average, stage.Percent)
	}
	if *cpuProfile != "" {
		fmt.Printf("CPU profile: %s\n", *cpuProfile)
	}
	if *heapProfile != "" {
		fmt.Printf("heap profile: %s\n", *heapProfile)
	}
	return nil
}

func handleLSPCommand(arguments []string) error {
	if len(arguments) == 1 && isHelpArgument(arguments[0]) {
		fmt.Fprintln(os.Stdout, "Usage:\n  zumbra lsp [--stdio]")
		return nil
	}
	if len(arguments) > 1 || len(arguments) == 1 && arguments[0] != "--stdio" {
		return fmt.Errorf("lsp only accepts --stdio")
	}
	return zumbraLSP.Run(os.Stdin, os.Stdout)
}

func handleProjectCommand(arguments []string) error {
	if len(arguments) == 0 {
		printProjectUsage(os.Stderr)
		return fmt.Errorf("project requires init, info, check, test, run, build, fmt, lint, doc or clean")
	}
	if len(arguments) == 1 && isHelpArgument(arguments[0]) {
		printProjectUsage(os.Stdout)
		return nil
	}
	if len(arguments) == 2 && isHelpArgument(arguments[1]) {
		return printProjectCommandUsage(os.Stdout, arguments[0])
	}
	switch arguments[0] {
	case "init":
		return projectInit(arguments[1:])
	case "info":
		return projectInfo(arguments[1:])
	case "check", "test":
		return projectCheck(arguments[0], arguments[1:])
	case "run":
		manifest, err := loadProjectFromArguments(arguments[1:])
		if err != nil {
			return err
		}
		return inProjectRoot(manifest, func() error {
			return executeFile(manifest.EntryPath())
		})
	case "build":
		return projectBuild(arguments[1:])
	case "fmt":
		manifest, err := loadProjectFromArguments(arguments[1:])
		if err != nil {
			return err
		}
		files, err := manifest.SourceFiles(true)
		if err != nil {
			return err
		}
		return handleFormatCommand(files)
	case "lint":
		manifest, err := loadProjectFromArguments(arguments[1:])
		if err != nil {
			return err
		}
		files, err := manifest.SourceFiles(true)
		if err != nil {
			return err
		}
		return handleLintCommand(append([]string{"--no-public-docs"}, files...))
	case "doc":
		manifest, err := loadProjectFromArguments(arguments[1:])
		if err != nil {
			return err
		}
		files, err := manifest.SourceFiles(false)
		if err != nil {
			return err
		}
		output := filepath.Join(manifest.Root, filepath.FromSlash(manifest.DocsOutput))
		args := []string{"--output", output, "--title", manifest.Name + " API"}
		args = append(args, files...)
		return handleDocCommand(args)
	case "clean":
		manifest, err := loadProjectFromArguments(arguments[1:])
		if err != nil {
			return err
		}
		removed, err := manifest.Clean()
		if err != nil {
			return err
		}
		for _, path := range removed {
			fmt.Printf("removed %s\n", path)
		}
		fmt.Printf("clean: %d path(s) removed\n", len(removed))
		return nil
	default:
		return fmt.Errorf("unknown project command %q", arguments[0])
	}
}

func projectInit(arguments []string) error {
	flags := flag.NewFlagSet("project init", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	directory := flags.String("dir", "", "target directory")
	kind := flags.String("kind", "cli", "cli, library or desktop")
	identifier := flags.String("identifier", "", "desktop reverse-DNS identifier")
	force := flags.Bool("force", false, "add or replace scaffold files")
	help, err := parseToolFlags(flags, arguments)
	if err != nil {
		return err
	}
	if help {
		return nil
	}
	if flags.NArg() != 1 {
		return fmt.Errorf("project init requires a project name")
	}
	manifest, created, err := project.Init(project.InitOptions{
		Directory: *directory, Name: flags.Arg(0), Kind: project.Kind(*kind), Identifier: *identifier, Force: *force,
	})
	if err != nil {
		return err
	}
	for _, path := range created {
		fmt.Printf("created %s\n", path)
	}
	fmt.Printf("initialized %s project %q at %s\n", manifest.Kind, manifest.Name, manifest.Root)
	return nil
}

func projectInfo(arguments []string) error {
	flags := flag.NewFlagSet("project info", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	manifestPath := flags.String("manifest", "", "manifest path")
	jsonOutput := flags.Bool("json", false, "emit JSON")
	help, err := parseToolFlags(flags, arguments)
	if err != nil {
		return err
	}
	if help {
		return nil
	}
	manifest, err := loadProject(*manifestPath)
	if err != nil {
		return err
	}
	if *jsonOutput {
		payload, err := json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(payload))
		return nil
	}
	fmt.Printf("project: %s\nversion: %s\nkind: %s\nroot: %s\nentry: %s\n", manifest.Name, manifest.Version, manifest.Kind, manifest.Root, manifest.EntryPath())
	files, _ := manifest.SourceFiles(true)
	fmt.Printf("files: %d\n", len(files))
	return nil
}

func projectCheck(command string, arguments []string) error {
	flags := flag.NewFlagSet("project "+command, flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	manifestPath := flags.String("manifest", "", "manifest path")
	jsonOutput := flags.Bool("json", false, "emit JSON")
	includeTests := flags.Bool("tests", command == "test", "include test sources")
	help, err := parseToolFlags(flags, arguments)
	if err != nil {
		return err
	}
	if help {
		return nil
	}
	manifest, err := loadProject(*manifestPath)
	if err != nil {
		return err
	}
	result := manifest.Check(*includeTests)
	if *jsonOutput {
		payload, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(payload))
	} else {
		for _, item := range result.Diagnostics {
			fmt.Println(item.Structured().String())
		}
		fmt.Printf("project %s: %d file(s), %d diagnostic(s)\n", command, result.Files, len(result.Diagnostics))
	}
	for _, item := range result.Diagnostics {
		if !item.Warning {
			return fmt.Errorf("project %s failed", command)
		}
	}
	if command == "test" {
		tests, err := manifest.TestFiles()
		if err != nil {
			return err
		}
		if err := inProjectRoot(manifest, func() error {
			for _, filename := range tests {
				fmt.Printf("test %s\n", filename)
				if err := executeFile(filename); err != nil {
					return fmt.Errorf("test %s failed: %w", filename, err)
				}
			}
			return nil
		}); err != nil {
			return err
		}
		fmt.Printf("project test: %d test file(s) executed\n", len(tests))
	}
	return nil
}

func projectBuild(arguments []string) error {
	manifestPath := ""
	debug := false
	buildArguments := []string{}
	for index := 0; index < len(arguments); index++ {
		switch arguments[index] {
		case "--manifest":
			index++
			if index >= len(arguments) {
				return fmt.Errorf("--manifest requires a path")
			}
			manifestPath = arguments[index]
		case "--debug":
			debug = true
		default:
			buildArguments = append(buildArguments, arguments[index])
		}
	}
	manifest, err := loadProject(manifestPath)
	if err != nil {
		return err
	}
	if !debug {
		hasRelease := false
		for _, argument := range buildArguments {
			if argument == "--release" {
				hasRelease = true
				break
			}
		}
		if !hasRelease {
			buildArguments = append([]string{"--release"}, buildArguments...)
		}
	}
	buildArguments = append(buildArguments, manifest.EntryPath())
	filename, options, err := parseBuildArguments(buildArguments)
	if err != nil {
		return err
	}
	return inProjectRoot(manifest, func() error {
		return buildZumbra(filename, options)
	})
}

func inProjectRoot(manifest *project.Manifest, action func() error) error {
	previous, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := os.Chdir(manifest.Root); err != nil {
		return err
	}
	defer func() { _ = os.Chdir(previous) }()
	return action()
}

func loadProjectFromArguments(arguments []string) (*project.Manifest, error) {
	manifestPath := ""
	if len(arguments) == 2 && arguments[0] == "--manifest" {
		manifestPath = arguments[1]
	} else if len(arguments) != 0 {
		return nil, fmt.Errorf("expected optional --manifest <zumbra.toml>")
	}
	return loadProject(manifestPath)
}

func loadProject(path string) (*project.Manifest, error) {
	if path == "" {
		found, err := project.Find(".")
		if err != nil {
			return nil, err
		}
		path = found
	}
	return project.Load(path)
}

func handleCheckCommand(arguments []string) error {
	if len(arguments) == 1 && isHelpArgument(arguments[0]) {
		fmt.Fprintln(os.Stdout, "Usage:\n  zumbra check [--json] <file.zum>")
		return nil
	}
	jsonOutput := false
	filename := ""
	for _, argument := range arguments {
		switch argument {
		case "--json":
			jsonOutput = true
		default:
			if strings.HasPrefix(argument, "-") {
				return fmt.Errorf("unknown check option %s", argument)
			}
			if filename != "" {
				return fmt.Errorf("check accepts one .zum file")
			}
			filename = argument
		}
	}
	if filename == "" {
		return fmt.Errorf("check requires a .zum file")
	}
	result, items := pipeline.BuildFile(filename, pipeline.Options{Optimize: true})
	if result != nil {
		items = append(items, result.Warnings...)
	}
	if jsonOutput {
		structured := make([]diagnostics.Diagnostic, 0, len(items))
		for _, item := range items {
			structured = append(structured, item.Structured())
		}
		payload, err := json.MarshalIndent(structured, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(payload))
	} else if len(items) == 0 {
		fmt.Printf("OK: %s\n", filename)
	} else {
		for _, item := range items {
			fmt.Println(item.Structured().String())
		}
	}
	for _, item := range items {
		if !item.Warning {
			return fmt.Errorf("check failed")
		}
	}
	return nil
}
