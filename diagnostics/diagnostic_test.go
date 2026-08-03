package diagnostics

import "testing"

func TestExtractPosition(t *testing.T) {
	cases := []struct {
		message string
		line    int
		column  int
	}{
		{"expected token at line 7, col 11", 7, 11},
		{"Syntax Error: <main.zum:4:2> bad", 4, 2},
		{"Syntax Error: <8:5> bad", 8, 5},
		{"undefined symbol at line 3, col 9", 3, 9},
	}
	for _, item := range cases {
		line, column := ExtractPosition(item.message)
		if line != item.line || column != item.column {
			t.Fatalf("%q: got %d:%d", item.message, line, column)
		}
	}
}
