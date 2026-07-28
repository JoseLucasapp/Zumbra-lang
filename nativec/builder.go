package nativec

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"zumbra/mir"
)

type BuildOptions struct {
	Release     bool
	EmitCOnly   bool
	Compiler    string
	Output      string
	BuildDir    string
	Links       []string
	IncludeDirs []string
	LibraryDirs []string
	Libraries   []string
}

type BuildResult struct {
	Compiler      string
	Output        string
	SourceDir     string
	ProgramSource string
	RuntimeSource string
	RuntimeHeader string
	Command       []string
	Links         []string
}

func DetectCompiler(preferred string) (string, error) {
	if preferred != "" && preferred != "auto" {
		path, err := exec.LookPath(preferred)
		if err != nil {
			return "", fmt.Errorf("C compiler %q was not found in PATH", preferred)
		}
		return path, nil
	}
	if cc := os.Getenv("CC"); cc != "" {
		if path, err := exec.LookPath(cc); err == nil {
			return path, nil
		}
	}
	for _, candidate := range []string{"clang", "gcc", "cc"} {
		if path, err := exec.LookPath(candidate); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no C compiler found; install Clang or GCC, or set CC")
}

func Build(module *mir.Module, options BuildOptions) (*BuildResult, []Diagnostic, error) {
	sources, diagnostics := Generate(module)
	if len(diagnostics) != 0 {
		return nil, diagnostics, nil
	}
	buildDir := options.BuildDir
	if buildDir == "" {
		buildDir = filepath.Join("build", "native")
	}
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		return nil, nil, fmt.Errorf("create native build directory: %w", err)
	}
	programPath := filepath.Join(buildDir, "main.c")
	runtimePath := filepath.Join(buildDir, "zumbra_runtime.c")
	headerPath := filepath.Join(buildDir, "zumbra_runtime.h")
	for path, content := range map[string][]byte{
		programPath: sources.Program,
		runtimePath: sources.Runtime,
		headerPath:  sources.Header,
	} {
		if err := os.WriteFile(path, content, 0o644); err != nil {
			return nil, nil, fmt.Errorf("write %s: %w", path, err)
		}
	}

	result := &BuildResult{
		SourceDir:     buildDir,
		ProgramSource: programPath,
		RuntimeSource: runtimePath,
		RuntimeHeader: headerPath,
	}
	if options.EmitCOnly {
		return result, nil, nil
	}
	compiler, err := DetectCompiler(options.Compiler)
	if err != nil {
		return nil, nil, err
	}
	output := options.Output
	if output == "" {
		output = filepath.Join("build", executableName(module.Filename))
	}
	if parent := filepath.Dir(output); parent != "." {
		if err := os.MkdirAll(parent, 0o755); err != nil {
			return nil, nil, fmt.Errorf("create output directory: %w", err)
		}
	}
	args := []string{"-std=c11", "-pthread", "-Wall", "-Wextra", "-Werror", "-Wno-unused-variable", "-Wno-unused-parameter", "-I", buildDir}
	for _, includeDir := range options.IncludeDirs {
		args = append(args, "-I", includeDir)
	}
	for _, libraryDir := range options.LibraryDirs {
		args = append(args, "-L", libraryDir)
	}
	if options.Release {
		args = append(args, "-O3", "-DNDEBUG")
	} else {
		args = append(args, "-O0", "-g3")
	}
	links := append([]string{}, NativeLinks(module)...)
	links = append(links, options.Links...)
	links = uniqueStrings(links)
	for _, link := range links {
		if _, statErr := os.Stat(link); statErr != nil {
			return nil, nil, fmt.Errorf("native link input %s: %w", link, statErr)
		}
	}
	args = append(args, programPath, runtimePath)
	args = append(args, links...)
	for _, library := range options.Libraries {
		args = append(args, "-l"+library)
	}
	if UsesDesktop(module) && runtime.GOOS == "linux" {
		args = append(args, "-ldl")
	}
	if UsesTLS(module) || UsesHTTP(module) {
		args = append(args, "-lssl", "-lcrypto")
	}
	if UsesHTTP(module) {
		args = append(args, "-lz")
	}
	if UsesSQLite(module) {
		args = append(args, "-lsqlite3")
	}
	if UsesPostgres(module) {
		args = append(args, "-lpq")
	}
	if UsesRedis(module) {
		args = append(args, "-lhiredis")
	}
	args = append(args, "-lm", "-pthread", "-o", output)
	command := exec.Command(compiler, args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return nil, nil, fmt.Errorf("native C compilation failed: %w", err)
	}
	result.Compiler = compiler
	result.Output = output
	result.Command = append([]string{compiler}, args...)
	result.Links = links
	return result, nil, nil
}

func executableName(filename string) string {
	name := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	if name == "" || name == "." {
		name = "zumbra-app"
	}
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

// NativeLinks returns source, object or library files declared by extern blocks.
func NativeLinks(module *mir.Module) []string {
	if module == nil {
		return nil
	}
	result := []string{}
	for _, declaration := range module.Declarations {
		if declaration != nil && declaration.Op == mir.OpExtern && declaration.Meta["link"] != "" {
			result = append(result, declaration.Meta["link"])
		}
	}
	return uniqueStrings(result)
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}
