package formatter

import (
	"strings"
	"testing"
)

func TestFormatIsIdempotentAndPreservesComments(t *testing.T) {
	input := `/// Adds two values.
pub fct add(a,b){return a+b;}
var values<<[1,2,3];// values
if(true){show(add(values[0],values[1]));}else{show(0);}`
	first, err := Format("main.zum", input, Options{})
	if err != nil {
		t.Fatal(err)
	}
	second, err := Format("main.zum", first.Source, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if first.Source != second.Source {
		t.Fatalf("formatter is not idempotent:\n%s\n---\n%s", first.Source, second.Source)
	}
	if !strings.Contains(first.Source, "/// Adds two values.") || !strings.Contains(first.Source, "// values") {
		t.Fatalf("comments were lost:\n%s", first.Source)
	}
	if !strings.Contains(first.Source, "pub fct add(a, b) {") {
		t.Fatalf("unexpected function formatting:\n%s", first.Source)
	}
}

func TestFormatRejectsInvalidSource(t *testing.T) {
	if _, err := Format("broken.zum", "var value << ;", Options{}); err == nil {
		t.Fatal("expected invalid source error")
	}
}
