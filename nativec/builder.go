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
	Release   bool
	EmitCOnly bool
	Compiler  string
	Output    string
	BuildDir  string
}

type BuildResult struct {
	Compiler      string
	Output        string
	SourceDir     string
	ProgramSource string
	RuntimeSource string
	RuntimeHeader string
	Command       []string
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
	args := []string{"-std=c11", "-Wall", "-Wextra", "-Werror", "-Wno-unused-variable", "-Wno-unused-parameter", "-I", buildDir}
	if options.Release {
		args = append(args, "-O3", "-DNDEBUG")
	} else {
		args = append(args, "-O0", "-g3")
	}
	args = append(args, programPath, runtimePath, "-lm", "-o", output)
	command := exec.Command(compiler, args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return nil, nil, fmt.Errorf("native C compilation failed: %w", err)
	}
	result.Compiler = compiler
	result.Output = output
	result.Command = append([]string{compiler}, args...)
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
