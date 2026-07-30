package builtins

import (
	"os"
	"path/filepath"
	"testing"

	"zumbra/object"
)

func requireZ12DataValue(t *testing.T, value object.Object) object.Object {
	t.Helper()
	if errObj, ok := value.(*object.Error); ok {
		t.Fatalf("Z12 data builtin failed: %s", errObj.Message)
	}
	return value
}

func TestZ12BinarySerializationPreservesNestedValuesAndBytes(t *testing.T) {
	payload := z11TestDict(map[string]object.Object{
		"name":   NewString("Lucas"),
		"score":  NewInteger(42),
		"active": NewBoolean(true),
		"raw":    &object.ByteArray{Data: []byte{0xab, 0xcd, 0x00}},
		"values": &object.Array{Elements: []object.Object{NewInteger(1), NewString("two"), &object.Null{}}},
	})
	encoded := requireZ12DataValue(t, BinaryEncodeBuiltin().Fn(payload)).(*object.ByteArray)
	if len(encoded.Data) <= len(zumbraBinaryEnvelope) || string(encoded.Data[:len(zumbraBinaryEnvelope)]) != zumbraBinaryEnvelope {
		t.Fatalf("invalid binary envelope: %q", encoded.Data)
	}
	decoded := requireZ12DataValue(t, BinaryDecodeBuiltin().Fn(encoded)).(*object.Dict)
	if got := sqliteTestDictValue(t, decoded, "name").(*object.String).Value; got != "Lucas" {
		t.Fatalf("name=%q", got)
	}
	if got := sqliteTestDictValue(t, decoded, "score").(*object.Integer).Value; got != 42 {
		t.Fatalf("score=%d", got)
	}
	if got := sqliteTestDictValue(t, decoded, "raw").(*object.ByteArray).Data; string(got) != string([]byte{0xab, 0xcd, 0x00}) {
		t.Fatalf("raw=%v", got)
	}
	values := sqliteTestDictValue(t, decoded, "values").(*object.Array)
	if len(values.Elements) != 3 || values.Elements[1].Inspect() != "two" || values.Elements[2].Type() != object.NULL_OBJ {
		t.Fatalf("values=%s", values.Inspect())
	}
}

func TestZ12JSONAndBinaryFilesRoundTripAtomically(t *testing.T) {
	directory := t.TempDir()
	jsonPath := filepath.Join(directory, "nested", "data.json")
	binaryPath := filepath.Join(directory, "nested", "data.zb")
	payload := z11TestDict(map[string]object.Object{"ok": NewBoolean(true), "count": NewInteger(2)})

	requireZ12DataValue(t, JSONWriteFileBuiltin().Fn(NewString(jsonPath), payload, NewBoolean(true)))
	jsonValue := requireZ12DataValue(t, JSONReadFileBuiltin().Fn(NewString(jsonPath))).(*object.Dict)
	if !sqliteTestDictValue(t, jsonValue, "ok").(*object.Boolean).Value {
		t.Fatal("JSON bool was not preserved")
	}
	requireZ12DataValue(t, BinaryWriteFileBuiltin().Fn(NewString(binaryPath), payload))
	binaryValue := requireZ12DataValue(t, BinaryReadFileBuiltin().Fn(NewString(binaryPath))).(*object.Dict)
	if got := sqliteTestDictValue(t, binaryValue, "count").(*object.Integer).Value; got != 2 {
		t.Fatalf("binary count=%d", got)
	}
	info, err := os.Stat(binaryPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("binary mode=%#o", info.Mode().Perm())
	}
}

func TestZ12BinarySerializationRejectsUnknownEnvelope(t *testing.T) {
	value := BinaryDecodeBuiltin().Fn(&object.ByteArray{Data: []byte("invalid")})
	if _, ok := value.(*object.Error); !ok {
		t.Fatalf("expected corrupt-envelope error, got %T", value)
	}
}

func TestZ16RecoverableJSONAndCSVFileOperations(t *testing.T) {
	directory := t.TempDir()
	missing := filepath.Join(directory, "missing.json")
	if FileExistsBuiltin().Fn(NewString(missing)).(*object.Boolean).Value {
		t.Fatal("missing file reported as existing")
	}
	missingResult := JSONReadResultBuiltin().Fn(NewString(missing)).(*object.Dict)
	if sqliteTestDictValue(t, missingResult, "ok").(*object.Boolean).Value {
		t.Fatal("missing JSON read should be recoverable failure")
	}
	if sqliteTestDictValue(t, missingResult, "error").(*object.String).Value == "" {
		t.Fatal("missing JSON read did not report an error")
	}

	jsonPath := filepath.Join(directory, "export.json")
	payload := z11TestDict(map[string]object.Object{"name": NewString("Final Fantasy IX"), "year": NewInteger(2000)})
	writeResult := JSONWriteResultBuiltin().Fn(NewString(jsonPath), payload, NewBoolean(true)).(*object.Dict)
	if !sqliteTestDictValue(t, writeResult, "ok").(*object.Boolean).Value {
		t.Fatalf("JSON write failed: %s", sqliteTestDictValue(t, writeResult, "error").Inspect())
	}
	if !FileExistsBuiltin().Fn(NewString(jsonPath)).(*object.Boolean).Value {
		t.Fatal("written JSON file was not found")
	}
	readResult := JSONReadResultBuiltin().Fn(NewString(jsonPath)).(*object.Dict)
	if !sqliteTestDictValue(t, readResult, "ok").(*object.Boolean).Value {
		t.Fatalf("JSON read failed: %s", sqliteTestDictValue(t, readResult, "error").Inspect())
	}

	csvPath := filepath.Join(directory, "collection.csv")
	rows := &object.Array{Elements: []object.Object{
		&object.Array{Elements: []object.Object{NewString("name"), NewString("platform"), NewString("notes")}},
		&object.Array{Elements: []object.Object{NewString("Final Fantasy IX"), NewString("PlayStation"), NewString("comma, quote \" and newline\nkept")}},
	}}
	csvWrite := CSVWriteResultBuiltin().Fn(NewString(csvPath), rows).(*object.Dict)
	if !sqliteTestDictValue(t, csvWrite, "ok").(*object.Boolean).Value {
		t.Fatalf("CSV write failed: %s", sqliteTestDictValue(t, csvWrite, "error").Inspect())
	}
	csvRead := CSVReadResultBuiltin().Fn(NewString(csvPath)).(*object.Dict)
	if !sqliteTestDictValue(t, csvRead, "ok").(*object.Boolean).Value {
		t.Fatalf("CSV read failed: %s", sqliteTestDictValue(t, csvRead, "error").Inspect())
	}
	decodedRows := sqliteTestDictValue(t, csvRead, "value").(*object.Array)
	if len(decodedRows.Elements) != 2 {
		t.Fatalf("CSV rows=%d", len(decodedRows.Elements))
	}
	decodedSecond := decodedRows.Elements[1].(*object.Array)
	if got := decodedSecond.Elements[2].(*object.String).Value; got != "comma, quote \" and newline\nkept" {
		t.Fatalf("CSV quoted value=%q", got)
	}
}
