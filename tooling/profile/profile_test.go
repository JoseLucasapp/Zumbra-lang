package profile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunProducesPipelineReport(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "main.zum")
	if err := os.WriteFile(filename, []byte("var value << 1 + 2; show(value);"), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := Run(filename, Options{Runs: 2, Warmup: 1, Optimize: true})
	if err != nil {
		t.Fatal(err)
	}
	if report.Runs != 2 || report.Average <= 0 || len(report.Stages) == 0 {
		t.Fatalf("unexpected report: %#v", report)
	}
}
