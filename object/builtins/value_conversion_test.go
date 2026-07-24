package builtins

import (
	"math"
	"reflect"
	"testing"

	"zumbra/object"
)

func TestGoValueToObjectConvertsNestedJSONValues(t *testing.T) {
	converted := goValueToObject(map[string]any{
		"name":   "zumbra",
		"active": true,
		"count":  float64(3),
		"items":  []any{"a", float64(2)},
	})

	dict, ok := converted.(*object.Dict)
	if !ok {
		t.Fatalf("expected Dict, got %T", converted)
	}

	nameKey := (&object.String{Value: "name"}).DictKey()
	if dict.Pairs[nameKey].Value.Inspect() != "zumbra" {
		t.Fatalf("unexpected name: %s", dict.Pairs[nameKey].Value.Inspect())
	}

	countKey := (&object.String{Value: "count"}).DictKey()
	count, ok := dict.Pairs[countKey].Value.(*object.Integer)
	if !ok || count.Value != 3 {
		t.Fatalf("expected integer count 3, got %#v", dict.Pairs[countKey].Value)
	}
}

func TestGoValueToObjectHandlesDatabaseDriverValues(t *testing.T) {
	if got := goValueToObject([]byte("database text")); got.Inspect() != "database text" {
		t.Fatalf("unexpected byte conversion: %s", got.Inspect())
	}

	large := goValueToObject(uint64(math.MaxUint64))
	fixed, ok := large.(*object.FixedInteger)
	if !ok || fixed.Kind != object.FixedU64 || fixed.UnsignedValue() != math.MaxUint64 {
		t.Fatalf("expected u64 max, got %#v", large)
	}
}

func TestObjectToGoValueConvertsNestedRuntimeObjects(t *testing.T) {
	key := &object.String{Value: "items"}
	value := &object.Dict{Pairs: map[object.DictKey]object.DictPair{
		key.DictKey(): {
			Key: key,
			Value: &object.Array{Elements: []object.Object{
				&object.String{Value: "a"},
				&object.Integer{Value: 2},
			}},
		},
	}}

	got := objectToGoValue(value)
	want := map[string]any{"items": []any{"a", int64(2)}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected conversion:\nwant: %#v\n got: %#v", want, got)
	}
}

func TestObjectToGoValuePreservesByteArrayAndFixedIntegers(t *testing.T) {
	bytesValue := objectToGoValue(&object.ByteArray{Data: []byte{1, 2, 3}})
	if !reflect.DeepEqual(bytesValue, []byte{1, 2, 3}) {
		t.Fatalf("unexpected byte array conversion: %#v", bytesValue)
	}

	fixed := object.NewFixedIntegerRaw(object.FixedI16, uint64(uint16(0xfffe)))
	if got := objectToGoValue(fixed); got != int64(-2) {
		t.Fatalf("expected -2, got %#v", got)
	}
}
