package builtins

import (
	"testing"
	"zumbra/object"
)

func TestMemoryBuiltins(t *testing.T) {
	memory := BytesBuiltin().Fn(&object.Integer{Value: 4})
	if memory.Type() != object.BYTE_ARRAY_OBJ {
		t.Fatalf("unexpected bytes type: %s", memory.Type())
	}
	values := ArrayOfBuiltin().Fn(&object.String{Value: "u16"}, &object.Integer{Value: 2})
	if values.Type() != object.TYPED_ARRAY_OBJ {
		t.Fatalf("unexpected typed array type: %s", values.Type())
	}
	view := SliceBuiltin().Fn(memory, &object.Integer{Value: 1}, &object.Integer{Value: 3})
	if view.Type() != object.SLICE_OBJ {
		t.Fatalf("unexpected slice type: %s", view.Type())
	}
}
