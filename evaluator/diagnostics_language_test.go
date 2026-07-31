package evaluator

import (
	"strings"
	"testing"
	"zumbra/object"
)

func TestRuntimeDiagnosticsDefaultToEnglish(t *testing.T) {
	evaluated := testEval("var value << 1; var value << 2;")
	err, ok := evaluated.(*object.Error)
	if !ok {
		t.Fatalf("expected evaluator error, got %T", evaluated)
	}
	if !strings.Contains(err.Message, "identifier 'value' already declared") {
		t.Fatalf("unexpected diagnostic: %q", err.Message)
	}
	if strings.Contains(err.Message, "variável") || strings.Contains(err.Message, "declarada") {
		t.Fatalf("diagnostic is not English by default: %q", err.Message)
	}
}
