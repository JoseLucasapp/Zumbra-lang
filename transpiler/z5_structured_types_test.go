package transpiler

import (
	goParser "go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestTranspilerLowersZ5Features(t *testing.T) {
	got, err := ZumbraTranspiler(`
        const Start << 1;
        struct Point { x: int; y: int; fct move(dx, dy) { self.x << self.x + dx; self.y << self.y + dy; } }
        enum Direction { Up; Down; }
        var p << Point(1, 2);
        p.move(3, 4);
        p.x << 10;
        var label << match(Direction.Up) { case Direction.Up { "up"; } else { "other"; } };
        show(p.x);
    `)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"func Point(args ...interface{})", `var Direction = zEnum`, `zCallMethod(p, "move"`, `zSetAttr(p, "x", 10)`, `zMatch(`, `zGetAttr(p, "x")`} {
		if !strings.Contains(got, expected) {
			t.Fatalf("missing %q\n%s", expected, got)
		}
	}
	if _, err := goParser.ParseFile(token.NewFileSet(), "generated.go", got, goParser.AllErrors); err != nil {
		t.Fatalf("generated Go is invalid: %v", err)
	}
}
