package mir_test

import (
	"strings"
	"testing"
)

func TestZ17OptimizeFoldsNativeSizeAndAlignment(t *testing.T) {
	module := lower(t, `
        var size << sizeOfType("i32");
        var alignment << alignOfType("u64");
        show(size);
        show(alignment);
    `, true)
	dump := module.Dump()
	if !strings.Contains(dump, `const value="4" : int`) {
		t.Fatalf("expected folded i32 size:\n%s", dump)
	}
	if !strings.Contains(dump, `const value="8" : int`) {
		t.Fatalf("expected folded u64 alignment:\n%s", dump)
	}
	if strings.Contains(dump, `load name="sizeOfType"`) || strings.Contains(dump, `load name="alignOfType"`) {
		t.Fatalf("native layout calls were not eliminated:\n%s", dump)
	}
}
