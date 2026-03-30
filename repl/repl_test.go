package repl

import (
	"bytes"
	"strings"
	"testing"
)

func TestStartPrintsPromptAndResult(t *testing.T) {
	in := strings.NewReader("1;\n")
	var out bytes.Buffer

	Start(in, &out)

	got := out.String()

	if !strings.Contains(got, ">> ") {
		t.Fatalf("repl output missing primary prompt. got=%q", got)
	}

	if !strings.Contains(got, "1") {
		t.Fatalf("repl output missing evaluation result. got=%q", got)
	}
}

func TestStartSupportsMultilineBlocks(t *testing.T) {
	in := strings.NewReader("var add << fct(a, b) {\na + b;\n};\nadd(1, 2);\n")
	var out bytes.Buffer

	Start(in, &out)

	got := out.String()

	if !strings.Contains(got, ".. ") {
		t.Fatalf("repl output missing multiline continuation prompt. got=%q", got)
	}

	if !strings.Contains(got, "3") {
		t.Fatalf("repl output missing final result. got=%q", got)
	}
}
