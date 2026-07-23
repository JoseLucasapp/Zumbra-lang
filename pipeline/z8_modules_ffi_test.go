package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"zumbra/mir"
)

func TestPipelineBuildsAliasedModuleGraphAndNativeLinks(t *testing.T) {
	dir := t.TempDir()
	module := filepath.Join(dir, "math.zum")
	entry := filepath.Join(dir, "app.zum")
	native := filepath.Join(dir, "math.c")
	if err := os.WriteFile(native, []byte("int z8_answer(void) { return 42; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(module, []byte(`
        pub const BASE << 40;
        pub extern "C" from "math.c" { fct answer() -> i32 as "z8_answer"; }
        const PRIVATE << 1;
    `), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(entry, []byte(`
        import "math.zum" as math;
        unsafe { show(math.answer()); }
        show(math.BASE);
    `), 0o644); err != nil {
		t.Fatal(err)
	}
	result, diagnostics := BuildFile(entry, Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("pipeline diagnostics: %s", FormatDiagnostics(diagnostics))
	}
	if result.Modules == nil || len(result.Modules.Units) != 2 {
		t.Fatalf("unexpected module graph: %#v", result.Modules)
	}
	if len(result.Modules.Links) != 1 || result.Modules.Links[0] != native {
		t.Fatalf("native link was not resolved: %#v", result.Modules.Links)
	}
	foundExtern := false
	for _, declaration := range result.MIR.Declarations {
		if declaration.Op == mir.OpExtern {
			foundExtern = true
			if declaration.Meta["c_name"] != "z8_answer" || declaration.Meta["link"] != native {
				t.Fatalf("extern metadata not preserved: %#v", declaration.Meta)
			}
		}
	}
	if !foundExtern {
		t.Fatalf("extern declaration missing from MIR:\n%s", result.DumpMIR())
	}
	if strings.Contains(result.Program.String(), "PRIVATE") && strings.Contains(result.Program.String(), "math.PRIVATE") {
		t.Fatal("private member leaked through module alias")
	}
}
