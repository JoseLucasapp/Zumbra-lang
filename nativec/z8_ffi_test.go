package nativec_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"zumbra/nativec"
	"zumbra/pipeline"
)

func buildAndRunZ8(t *testing.T, relative string) string {
	t.Helper()
	if _, err := nativec.DetectCompiler("auto"); err != nil {
		t.Skip(err)
	}
	input := filepath.Join("..", relative)
	result, diagnostics := pipeline.BuildFile(input, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("pipeline diagnostics: %s", pipeline.FormatDiagnostics(diagnostics))
	}
	output := filepath.Join(t.TempDir(), "program")
	built, nativeDiagnostics, err := nativec.Build(result.MIR, nativec.BuildOptions{Release: true, Output: output, BuildDir: filepath.Join(t.TempDir(), "sources")})
	if err != nil {
		t.Fatal(err)
	}
	if len(nativeDiagnostics) != 0 {
		t.Fatalf("native diagnostics: %#v", nativeDiagnostics)
	}
	command := exec.Command(built.Output)
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("native execution: %v\n%s", err, data)
	}
	return strings.ReplaceAll(string(data), "\r\n", "\n")
}

func TestZ8AliasedModulesBuildNatively(t *testing.T) {
	output := buildAndRunZ8(t, "code_examples/core/modules.zum")
	if output != "42\n15\n" {
		t.Fatalf("unexpected module output %q", output)
	}
}

func TestZ8CFFIAndCallbackBuildNatively(t *testing.T) {
	output := buildAndRunZ8(t, "code_examples/core/ffi.zum")
	if output != "42\n42\nzumbra\ntrue\n" {
		t.Fatalf("unexpected FFI output %q", output)
	}
}

func TestZ8StaticAndSharedLibraryLinking(t *testing.T) {
	compiler, err := nativec.DetectCompiler("auto")
	if err != nil {
		t.Skip(err)
	}
	ar, err := exec.LookPath("ar")
	if err != nil {
		t.Skip("ar is required for static-link validation")
	}
	dir := t.TempDir()
	cSource := filepath.Join(dir, "answer.c")
	if err := os.WriteFile(cSource, []byte("#include <stdint.h>\nint32_t z8_answer(void){return 42;}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	object := filepath.Join(dir, "answer.o")
	if output, err := exec.Command(compiler, "-c", "-fPIC", cSource, "-o", object).CombinedOutput(); err != nil {
		t.Fatalf("compile fixture: %v\n%s", err, output)
	}
	archive := filepath.Join(dir, "libanswer.a")
	if output, err := exec.Command(ar, "rcs", archive, object).CombinedOutput(); err != nil {
		t.Fatalf("archive fixture: %v\n%s", err, output)
	}

	zumbraSource := filepath.Join(dir, "app.zum")
	if err := os.WriteFile(zumbraSource, []byte(`extern "C" { fct answer() -> i32 as "z8_answer"; } unsafe { show(answer()); }`), 0o644); err != nil {
		t.Fatal(err)
	}
	pipelineResult, diagnostics := pipeline.BuildFile(zumbraSource, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("pipeline diagnostics: %s", pipeline.FormatDiagnostics(diagnostics))
	}

	t.Run("static archive", func(t *testing.T) {
		output := filepath.Join(dir, "static-program")
		built, nativeDiagnostics, buildErr := nativec.Build(pipelineResult.MIR, nativec.BuildOptions{
			Release: true, Output: output, BuildDir: filepath.Join(dir, "static-src"), Links: []string{archive},
		})
		if buildErr != nil || len(nativeDiagnostics) != 0 {
			t.Fatalf("static build: %v %#v", buildErr, nativeDiagnostics)
		}
		data, runErr := exec.Command(built.Output).CombinedOutput()
		if runErr != nil || strings.TrimSpace(string(data)) != "42" {
			t.Fatalf("static run: %v %q", runErr, data)
		}
	})

	if runtime.GOOS != "linux" {
		return
	}
	t.Run("shared library", func(t *testing.T) {
		shared := filepath.Join(dir, "libanswer.so")
		if output, err := exec.Command(compiler, "-shared", "-fPIC", cSource, "-o", shared).CombinedOutput(); err != nil {
			t.Fatalf("shared fixture: %v\n%s", err, output)
		}
		output := filepath.Join(dir, "shared-program")
		built, nativeDiagnostics, buildErr := nativec.Build(pipelineResult.MIR, nativec.BuildOptions{
			Release: true, Output: output, BuildDir: filepath.Join(dir, "shared-src"), Links: []string{shared},
		})
		if buildErr != nil || len(nativeDiagnostics) != 0 {
			t.Fatalf("shared build: %v %#v", buildErr, nativeDiagnostics)
		}
		command := exec.Command(built.Output)
		command.Env = append(os.Environ(), "LD_LIBRARY_PATH="+dir)
		data, runErr := command.CombinedOutput()
		if runErr != nil || strings.TrimSpace(string(data)) != "42" {
			t.Fatalf("shared run: %v %q", runErr, data)
		}
	})

	t.Run("library search flags", func(t *testing.T) {
		shared := filepath.Join(dir, "libanswer.so")
		if _, err := os.Stat(shared); err != nil {
			t.Fatal(err)
		}
		output := filepath.Join(dir, "library-flag-program")
		built, nativeDiagnostics, buildErr := nativec.Build(pipelineResult.MIR, nativec.BuildOptions{
			Release: true, Output: output, BuildDir: filepath.Join(dir, "library-flag-src"),
			LibraryDirs: []string{dir}, Libraries: []string{"answer"},
		})
		if buildErr != nil || len(nativeDiagnostics) != 0 {
			t.Fatalf("library-flag build: %v %#v", buildErr, nativeDiagnostics)
		}
		command := exec.Command(built.Output)
		command.Env = append(os.Environ(), "LD_LIBRARY_PATH="+dir)
		data, runErr := command.CombinedOutput()
		if runErr != nil || strings.TrimSpace(string(data)) != "42" {
			t.Fatalf("library-flag run: %v %q", runErr, data)
		}
	})
}

func TestZ8FFIUsesExactCPrototypeTypes(t *testing.T) {
	result, diagnostics := pipeline.Build("types.zum", `
        extern "C" {
            fct platform(value: int) -> int as "platform_int";
            fct text() -> cstring as "native_text";
            fct handle(value: ptr) -> ptr as "native_handle";
        }
    `, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("pipeline diagnostics: %s", pipeline.FormatDiagnostics(diagnostics))
	}
	sources, nativeDiagnostics := nativec.Generate(result.MIR)
	if len(nativeDiagnostics) != 0 {
		t.Fatalf("native diagnostics: %#v", nativeDiagnostics)
	}
	generated := string(sources.Program)
	for _, expected := range []string{
		"extern int platform_int(int value);",
		"extern const char * native_text(void);",
		"extern void * native_handle(void * value);",
	} {
		if !strings.Contains(generated, expected) {
			t.Fatalf("missing C prototype %q:\n%s", expected, generated)
		}
	}
}

func TestZ8IncludeDirectoryForNativeSource(t *testing.T) {
	if _, err := nativec.DetectCompiler("auto"); err != nil {
		t.Skip(err)
	}
	dir := t.TempDir()
	includeDir := filepath.Join(dir, "include")
	sourceDir := filepath.Join(dir, "source")
	if err := os.MkdirAll(includeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(includeDir, "answer.h"), []byte("#include <stdint.h>\nint32_t z8_answer(void);\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cSource := filepath.Join(sourceDir, "answer.c")
	if err := os.WriteFile(cSource, []byte("#include <answer.h>\nint32_t z8_answer(void){return 42;}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, diagnostics := pipeline.Build("include.zum", `extern "C" { fct answer() -> i32 as "z8_answer"; } unsafe { show(answer()); }`, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("pipeline diagnostics: %s", pipeline.FormatDiagnostics(diagnostics))
	}
	output := filepath.Join(dir, "program")
	built, nativeDiagnostics, err := nativec.Build(result.MIR, nativec.BuildOptions{
		Release: true, Output: output, BuildDir: filepath.Join(dir, "generated"),
		Links: []string{cSource}, IncludeDirs: []string{includeDir},
	})
	if err != nil || len(nativeDiagnostics) != 0 {
		t.Fatalf("native build: %v %#v", err, nativeDiagnostics)
	}
	data, runErr := exec.Command(built.Output).CombinedOutput()
	if runErr != nil || strings.TrimSpace(string(data)) != "42" {
		t.Fatalf("native run: %v %q", runErr, data)
	}
}
