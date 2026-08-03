package lint

import "testing"

func TestLintFindsStyleDocumentationAndControlFlowIssues(t *testing.T) {
	source := "pub struct point { value: i32; }\n" +
		"fct choose(flag) { return flag == true; show(1); }\n" +
		"import \"a.zum\";\nimport \"a.zum\";\n"
	result := Source("main.zum", source, Options{RequirePublicDocs: true, MaxLineLength: 120})
	codes := map[string]bool{}
	for _, diagnostic := range result.Diagnostics {
		codes[diagnostic.Code] = true
	}
	for _, expected := range []string{"ZL2001", "ZL2002", "ZL2003", "ZL2004", "ZL2005"} {
		if !codes[expected] {
			t.Fatalf("missing %s in %#v", expected, result.Diagnostics)
		}
	}
}

func TestLintReportsParserErrors(t *testing.T) {
	result := Source("broken.zum", "var value << ;", Options{})
	if result.Errors == 0 || result.Diagnostics[0].Code != "ZL0001" {
		t.Fatalf("expected parser error: %#v", result)
	}
}
