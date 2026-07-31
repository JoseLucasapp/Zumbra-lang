package transpiler

import (
	"strings"
	"testing"
)

func TestZumbraTranspilerNormalizesCRLF(t *testing.T) {
	input := "var x << 10;\r\nshow(x);\r\n"

	got, err := ZumbraTranspiler(input)
	if err != nil {
		t.Fatalf("transpiler returned error: %s", err)
	}

	if !strings.Contains(got, "var x = 10") {
		t.Fatalf("transpiled output missing var assignment. got:\n%s", got)
	}

	if !strings.Contains(got, "fmt.Println(x)") {
		t.Fatalf("transpiled output missing show translation. got:\n%s", got)
	}
}

func TestZumbraTranspilerRejectsUnterminatedRestHandler(t *testing.T) {
	input := `
restGet("/health", fct(req, res) {
	show("ok");
`

	_, err := ZumbraTranspiler(input)
	if err == nil {
		t.Fatalf("expected unterminated REST handler error, got nil")
	}

	if err.Error() != "unterminated REST handler block" {
		t.Fatalf("wrong error. got=%q", err.Error())
	}
}
