package evaluator

import (
	"strings"
	"testing"
	"zumbra/object"
)

func TestEvaluatorRegistersExternButRequiresNativeBuild(t *testing.T) {
	result := testEval(`extern "C" { fct answer() -> i32; } unsafe { answer(); }`)
	errorObject, ok := result.(*object.Error)
	if !ok || !strings.Contains(errorObject.Message, "requires `zumbra build`") {
		t.Fatalf("expected native-build error, got %T %#v", result, result)
	}
}
