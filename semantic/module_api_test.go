package semantic

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"zumbra/lexer"
	"zumbra/parser"
)

func parseModuleProgram(t *testing.T, input string) *parser.Parser {
	t.Helper()
	l := lexer.New(input)
	return parser.New(l)
}

func parseModuleOrFatal(t *testing.T, input string) *parser.Parser {
	t.Helper()
	p := parseModuleProgram(t, input)
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	return p
}

func TestAnalyzeModuleLoadsImportedGlobals(t *testing.T) {
	dir := t.TempDir()

	err := os.WriteFile(filepath.Join(dir, "module.zum"), []byte(`var importedValue << 10;`), 0o644)
	if err != nil {
		t.Fatalf("failed to write module: %s", err)
	}

	p := parseModuleOrFatal(t, `import "module.zum"; importedValue;`)
	program := p.ParseProgram()

	result, errs := AnalyzeModule(filepath.Join(dir, "main.zum"), program)
	if len(errs) != 0 {
		t.Fatalf("expected 0 errors, got %d: %v", len(errs), errs)
	}
	if result == nil {
		t.Fatalf("expected non-nil result")
	}
}

func TestAnalyzeModuleDetectsCycle(t *testing.T) {
	dir := t.TempDir()

	err := os.WriteFile(filepath.Join(dir, "a.zum"), []byte(`import "b.zum"; var aValue << 1;`), 0o644)
	if err != nil {
		t.Fatalf("write a.zum failed: %s", err)
	}

	err = os.WriteFile(filepath.Join(dir, "b.zum"), []byte(`import "a.zum"; var bValue << 2;`), 0o644)
	if err != nil {
		t.Fatalf("write b.zum failed: %s", err)
	}

	p := parseModuleOrFatal(t, `import "a.zum";`)
	program := p.ParseProgram()

	_, errs := AnalyzeModule(filepath.Join(dir, "main.zum"), program)
	if len(errs) == 0 {
		t.Fatalf("expected cycle error, got none")
	}

	msg := errs[0].Error()
	if !strings.Contains(msg, "cyclic import") {
		t.Fatalf("unexpected cycle error: %s", msg)
	}
}
