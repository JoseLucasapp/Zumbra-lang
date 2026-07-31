package modules_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"zumbra/lexer"
	"zumbra/modules"
	"zumbra/parser"
)

func parse(t *testing.T, source string) *parser.Parser {
	t.Helper()
	return parser.New(lexer.New(source))
}

func TestAliasedImportUsesOnlyPublicSymbols(t *testing.T) {
	dir := t.TempDir()
	dependency := filepath.Join(dir, "math.zum")
	if err := os.WriteFile(dependency, []byte(`pub const Answer << 42; const Secret << 7; pub fct add(a, b) { a + b; }`), 0o644); err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(dir, "main.zum")
	source := `import "math.zum" as math; show(math.add(math.Answer, 0));`
	p := parse(t, source)
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatal(p.Errors())
	}
	flattened, graph, diagnostics := modules.Resolve(entry, program)
	for _, diagnostic := range diagnostics {
		if !diagnostic.Warning {
			t.Fatal(diagnostic)
		}
	}
	if graph == nil || len(graph.Units) != 2 {
		t.Fatalf("expected two modules, got %#v", graph)
	}
	text := flattened.String()
	if strings.Contains(text, "math.add") || strings.Contains(text, "math.Answer") {
		t.Fatalf("alias was not lowered: %s", text)
	}
	if !strings.Contains(text, "__zm_") {
		t.Fatalf("dependency was not namespaced: %s", text)
	}
}

func TestAliasedImportRejectsPrivateSymbol(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "math.zum"), []byte(`const Secret << 7;`), 0o644); err != nil {
		t.Fatal(err)
	}
	p := parse(t, `import "math.zum" as math; show(math.Secret);`)
	program := p.ParseProgram()
	_, _, diagnostics := modules.Resolve(filepath.Join(dir, "main.zum"), program)
	found := false
	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Message, "private") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected private member diagnostic: %#v", diagnostics)
	}
}

func TestModuleCycleIsRejected(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.zum"), []byte(`import "b.zum" as b; pub const A << 1;`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.zum"), []byte(`import "a.zum" as a; pub const B << 2;`), 0o644); err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(filepath.Join(dir, "a.zum"))
	p := parse(t, string(content))
	program := p.ParseProgram()
	_, _, diagnostics := modules.Resolve(filepath.Join(dir, "a.zum"), program)
	found := false
	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Message, "cyclic") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected cyclic import diagnostic: %#v", diagnostics)
	}
}

func TestModuleNamespaceIsStableAcrossProjectLocations(t *testing.T) {
	resolveProject := func(root string) string {
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "math.zum"), []byte(`pub const Answer << 42;`), 0o644); err != nil {
			t.Fatal(err)
		}
		p := parse(t, `import "math.zum" as math; math.Answer;`)
		program := p.ParseProgram()
		flattened, _, diagnostics := modules.Resolve(filepath.Join(root, "app.zum"), program)
		for _, diagnostic := range diagnostics {
			if !diagnostic.Warning {
				t.Fatal(diagnostic)
			}
		}
		return flattened.String()
	}
	first := resolveProject(filepath.Join(t.TempDir(), "project"))
	second := resolveProject(filepath.Join(t.TempDir(), "project"))
	if first != second {
		t.Fatalf("module namespace depends on absolute checkout path:\n%s\n%s", first, second)
	}
}
