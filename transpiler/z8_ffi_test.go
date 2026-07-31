package transpiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"zumbra/pipeline"
)

func TestGoTranspilerRejectsNativeFFI(t *testing.T) {
	result, diagnostics := pipeline.Build("ffi.zum", `extern "C" { fct answer() -> i32; } unsafe { answer(); }`, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("pipeline diagnostics: %s", pipeline.FormatDiagnostics(diagnostics))
	}
	_, err := ZumbraTranspilerPipeline(result)
	if err == nil || !strings.Contains(err.Error(), "native backend") {
		t.Fatalf("expected native-only diagnostic, got %v", err)
	}
}

func TestGoTranspilerConsumesFlattenedAliasedModule(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "math.zum"), []byte(`pub const BASE << 40; pub fct add(left, right) { left + right; }`), 0o644); err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(dir, "app.zum")
	if err := os.WriteFile(entry, []byte(`import "math.zum" as math; var result << math.add(math.BASE, 2); show(result);`), 0o644); err != nil {
		t.Fatal(err)
	}
	result, diagnostics := pipeline.BuildFile(entry, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("pipeline diagnostics: %s", pipeline.FormatDiagnostics(diagnostics))
	}
	generated, err := ZumbraTranspilerPipeline(result)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(generated, "package main") || strings.Contains(generated, `import "math.zum"`) {
		t.Fatalf("module was not flattened for Go output:\n%s", generated)
	}
}
