package nativec_test

import (
	"path/filepath"
	"strings"
	"testing"
	"zumbra/nativec"
	"zumbra/pipeline"
)

func TestZ12SQLiteBuildsAndRunsNatively(t *testing.T) {
	output := buildAndRunZ8(t, "code_examples/core/sqlite.zum")
	expected := "1\nLucas\n42\n1\ntrue\ntrue\n"
	if output != expected {
		t.Fatalf("unexpected native SQLite output %q", output)
	}
}

func TestZ12SQLiteRuntimeIsConditionallyLinked(t *testing.T) {
	plain, diagnostics := pipeline.Build("plain.zum", `show(42);`, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatal(pipeline.FormatDiagnostics(diagnostics))
	}
	plainSources, nativeDiagnostics := nativec.Generate(plain.MIR)
	if len(nativeDiagnostics) != 0 {
		t.Fatalf("native diagnostics: %#v", nativeDiagnostics)
	}
	if strings.Contains(string(plainSources.Runtime), "#define ZUMBRA_ENABLE_SQLITE 1") {
		t.Fatal("plain program unexpectedly enabled SQLite")
	}

	program, diagnostics := pipeline.Build("sqlite.zum", `var db << sqliteMemory(); db.close();`, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatal(pipeline.FormatDiagnostics(diagnostics))
	}
	sources, nativeDiagnostics := nativec.Generate(program.MIR)
	if len(nativeDiagnostics) != 0 {
		t.Fatalf("native diagnostics: %#v", nativeDiagnostics)
	}
	if !strings.Contains(string(sources.Runtime), "#define ZUMBRA_ENABLE_SQLITE 1") {
		t.Fatal("SQLite program did not enable native SQLite runtime")
	}

	result, buildDiagnostics, err := nativec.Build(program.MIR, nativec.BuildOptions{EmitCOnly: true, BuildDir: filepath.Join(t.TempDir(), "sources")})
	if err != nil || len(buildDiagnostics) != 0 || result == nil {
		t.Fatalf("emit C failed: result=%v diagnostics=%v err=%v", result, buildDiagnostics, err)
	}
}
