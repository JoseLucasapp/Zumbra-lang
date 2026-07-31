package nativec_test

import "testing"

func TestZ91CallInferenceBuildsAndRunsNatively(t *testing.T) {
	output := buildAndRunZ8(t, "code_examples/core/call_inference.zum")
	if output != "42\n7\n2000\n9\n" {
		t.Fatalf("unexpected call inference output %q", output)
	}
}
