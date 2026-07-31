package pipeline_test

import (
	"strings"
	"testing"
	"zumbra/pipeline"
)

const structuredSource = `
type Energy << u8;
const START << 10u8;
struct Player {
    x: int;
    energy: Energy;
    fct move(dx) { self.x << self.x + dx; }
}
enum Direction { Left; Right; }
var player << Player(2, START);
player.move(3);
var direction << Direction.Right;
var label << match(direction) {
    case Direction.Left { "left"; }
    case Direction.Right { "right"; }
    else { "other"; }
};
var folded << 2 + 3 * 4;
`

func TestBuildCreatesCanonicalFrontendArtifacts(t *testing.T) {
	result, diagnostics := pipeline.Build("test.zum", structuredSource, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics: %v", diagnostics)
	}
	if result.Program == nil || result.Types == nil || result.HIR == nil || result.MIR == nil {
		t.Fatal("pipeline returned incomplete result")
	}
	if !result.MIR.Optimized {
		t.Fatal("expected optimized MIR")
	}
	if !strings.Contains(result.DumpHIR(), `struct name="Player"`) {
		t.Fatalf("missing struct in HIR:\n%s", result.DumpHIR())
	}
	if !strings.Contains(result.DumpMIR(), `const value="14" : int`) {
		t.Fatalf("missing folded value in MIR:\n%s", result.DumpMIR())
	}
}

func TestBuildStopsAtParserErrors(t *testing.T) {
	result, diagnostics := pipeline.Build("bad.zum", `var value << ;`, pipeline.Options{})
	if result == nil || result.Program == nil {
		t.Fatal("expected partial result")
	}
	if len(diagnostics) == 0 || diagnostics[0].Stage != pipeline.StageParser {
		t.Fatalf("expected parser diagnostic, got %v", diagnostics)
	}
}
