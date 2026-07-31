package object

import "testing"

func TestStructuredObjects(t *testing.T) {
	def := &StructDefinition{Name: "Point", Fields: []StructFieldDefinition{{Name: "x"}}, Methods: map[string]Object{}}
	instance := &StructInstance{Definition: def, Fields: map[string]Object{"x": &Integer{Value: 7}}}
	if instance.Type() != STRUCT_INSTANCE_OBJ || instance.Inspect() != "Point{x: 7}" {
		t.Fatalf("unexpected instance: %s", instance.Inspect())
	}
	value := &EnumValue{EnumName: "Direction", Name: "Up", Ordinal: 0}
	if value.Inspect() != "Direction.Up" {
		t.Fatalf("unexpected enum: %s", value.Inspect())
	}
}

func TestConstantInOuterEnvironmentCannotBeAssigned(t *testing.T) {
	outer := NewEnvironment()
	outer.DefineConst("Max", &Integer{Value: 3})
	inner := NewEnclosedEnvironment(outer)
	if _, err := inner.Assign("Max", &Integer{Value: 4}); err == nil {
		t.Fatal("expected constant assignment error")
	}
}
