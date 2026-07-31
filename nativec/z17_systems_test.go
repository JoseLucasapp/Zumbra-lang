package nativec_test

import (
	"strings"
	"testing"

	"zumbra/nativec"
	"zumbra/pipeline"
)

const z17SystemsExpected = "30\n4\n16\n20\ntrue\n80\ntrue\n7\ntrue\n9\ntrue\n7\n1\ntrue\ntrue\n16\n4\n8\n55\n56\ntrue\ntrue\ntrue\n"

func TestZ17SystemsProgrammingExampleBuildsAndRunsNatively(t *testing.T) {
	if output := buildAndRunZ8(t, "code_examples/core/systems_programming.zum"); output != z17SystemsExpected {
		t.Fatalf("unexpected native output %q", output)
	}
}

func TestZ17MappedFileExampleBuildsAndRunsNatively(t *testing.T) {
	const expected = "8\ntrue\ntrue\n42\n99\n"
	if output := buildAndRunZ8(t, "code_examples/core/systems_mapping.zum"); output != expected {
		t.Fatalf("unexpected native output %q", output)
	}
}

func TestZ17ProcessExampleBuildsAndRunsNatively(t *testing.T) {
	const expected = "true\n7\n"
	if output := buildAndRunZ8(t, "code_examples/core/systems_process.zum"); output != expected {
		t.Fatalf("unexpected native output %q", output)
	}
}

func TestZ17DynamicLibraryCallBuildsAndRunsNatively(t *testing.T) {
	const expected = "true\ntrue\n"
	if output := buildAndRunZ8(t, "code_examples/core/systems_dynamic.zum"); output != expected {
		t.Fatalf("unexpected native output %q", output)
	}
}

func TestZ17NativeRuntimeIsConditionallyEnabled(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		shouldUse bool
	}{
		{name: "plain", source: `show(42);`, shouldUse: false},
		{name: "pointer", source: `var p << alloc("u8", 1); p[0] << 7u8; show(p[0]); free(p);`, shouldUse: true},
		{name: "arena", source: `var a << arenaCreate(); show(arenaStats(a)["open"]); arenaFree(a);`, shouldUse: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, diagnostics := pipeline.Build(test.name+".zum", test.source, pipeline.Options{Optimize: true})
			if len(diagnostics) != 0 {
				t.Fatalf("pipeline diagnostics: %s", pipeline.FormatDiagnostics(diagnostics))
			}
			if nativec.UsesSystems(result.MIR) != test.shouldUse {
				t.Fatalf("UsesSystems=%t, want %t", nativec.UsesSystems(result.MIR), test.shouldUse)
			}
			sources, nativeDiagnostics := nativec.Generate(result.MIR)
			if len(nativeDiagnostics) != 0 {
				t.Fatalf("native diagnostics: %#v", nativeDiagnostics)
			}
			runtimeSource := string(sources.Runtime)
			enabled := strings.Contains(runtimeSource, "#define ZUMBRA_ENABLE_SYSTEMS 1")
			if enabled != test.shouldUse {
				t.Fatalf("systems runtime enabled=%t, want %t", enabled, test.shouldUse)
			}
		})
	}
}
