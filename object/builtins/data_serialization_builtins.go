package builtins

import (
	"bytes"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"zumbra/collections"
	"zumbra/object"
)

const zumbraBinaryEnvelope = "ZB1\n"

func dataResult(ok bool, value object.Object, errText string) object.Object {
	if value == nil {
		value = &object.Null{}
	}
	return objectMapDict(map[string]object.Object{
		"ok":    NewBoolean(ok),
		"value": value,
		"error": NewString(errText),
	})
}

func dataResultFromObject(value object.Object) object.Object {
	if errObj, ok := value.(*object.Error); ok {
		return dataResult(false, &object.Null{}, errObj.Message)
	}
	return dataResult(true, value, "")
}

// FileExistsBuiltin checks whether a regular file or directory exists without
// raising a runtime error when it does not.
func FileExistsBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("fileExists expects 1 argument, got %d", len(args))
		}
		path, errObj := httpString(args[0], "fileExists path")
		if errObj != nil {
			return errObj
		}
		_, err := os.Stat(path)
		if err == nil {
			return NewBoolean(true)
		}
		if os.IsNotExist(err) {
			return NewBoolean(false)
		}
		return NewError("fileExists: %s", err)
	}}
}

// JSONReadResultBuiltin is the recoverable form of jsonReadFile. It returns
// {ok, value, error} and never converts an ordinary I/O or parse failure into a
// fatal application error.
func JSONReadResultBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("jsonReadResult expects 1 argument, got %d", len(args))
		}
		return dataResultFromObject(JSONReadFileBuiltin().Fn(args...))
	}}
}

// JSONWriteResultBuiltin is the recoverable form of jsonWriteFile.
func JSONWriteResultBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) < 2 || len(args) > 3 {
			return NewError("jsonWriteResult expects path, value and optional pretty flag")
		}
		return dataResultFromObject(JSONWriteFileBuiltin().Fn(args...))
	}}
}

func csvCellText(value object.Object) (string, error) {
	switch current := value.(type) {
	case nil, *object.Null:
		return "", nil
	case *object.String:
		return current.Value, nil
	case *object.Integer, *object.Float, *object.Boolean, *object.FixedInteger:
		return current.Inspect(), nil
	default:
		return "", fmt.Errorf("unsupported CSV cell type %s", value.Type())
	}
}

// CSVReadFileBuiltin parses RFC 4180-style CSV records while preserving every
// cell as a string. Variable-width rows are accepted for import diagnostics.
func CSVReadFileBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("csvReadFile expects path")
		}
		path, errObj := httpString(args[0], "csvReadFile path")
		if errObj != nil {
			return errObj
		}
		file, err := os.Open(path)
		if err != nil {
			return NewError("csvReadFile: %s", err)
		}
		defer file.Close()
		reader := csv.NewReader(file)
		reader.FieldsPerRecord = -1
		records, err := reader.ReadAll()
		if err != nil {
			return NewError("csvReadFile: %s", err)
		}
		rows := make([]object.Object, 0, len(records))
		for _, record := range records {
			cells := make([]object.Object, 0, len(record))
			for _, cell := range record {
				cells = append(cells, NewString(cell))
			}
			rows = append(rows, &object.Array{Elements: cells})
		}
		return &object.Array{Elements: rows}
	}}
}

// CSVWriteFileBuiltin writes CSV atomically and returns the number of bytes
// persisted.
func CSVWriteFileBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("csvWriteFile expects path and rows")
		}
		path, errObj := httpString(args[0], "csvWriteFile path")
		if errObj != nil {
			return errObj
		}
		rows, ok := args[1].(*object.Array)
		if !ok {
			return NewError("csvWriteFile rows must be an array")
		}
		var buffer bytes.Buffer
		writer := csv.NewWriter(&buffer)
		for rowIndex, rowValue := range rows.Elements {
			row, ok := rowValue.(*object.Array)
			if !ok {
				return NewError("csvWriteFile row %d must be an array", rowIndex+1)
			}
			record := make([]string, 0, len(row.Elements))
			for columnIndex, cell := range row.Elements {
				text, err := csvCellText(cell)
				if err != nil {
					return NewError("csvWriteFile row %d column %d: %s", rowIndex+1, columnIndex+1, err)
				}
				record = append(record, text)
			}
			if err := writer.Write(record); err != nil {
				return NewError("csvWriteFile: %s", err)
			}
		}
		writer.Flush()
		if err := writer.Error(); err != nil {
			return NewError("csvWriteFile: %s", err)
		}
		if err := atomicWriteFile(path, buffer.Bytes(), 0o644); err != nil {
			return NewError("csvWriteFile: %s", err)
		}
		return &object.Integer{Value: int64(buffer.Len())}
	}}
}

func CSVReadResultBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("csvReadResult expects path")
		}
		return dataResultFromObject(CSVReadFileBuiltin().Fn(args...))
	}}
}

func CSVWriteResultBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("csvWriteResult expects path and rows")
		}
		return dataResultFromObject(CSVWriteFileBuiltin().Fn(args...))
	}}
}

// JSONReadFileBuiltin reads a standard JSON document using json.Number so
// integers do not silently become floats.
func JSONReadFileBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("jsonReadFile expects 1 argument, got %d", len(args))
		}
		path, errObj := httpString(args[0], "jsonReadFile path")
		if errObj != nil {
			return errObj
		}
		file, err := os.Open(path)
		if err != nil {
			return NewError("jsonReadFile: %s", err)
		}
		defer file.Close()

		decoder := json.NewDecoder(file)
		decoder.UseNumber()
		var decoded any
		if err := decoder.Decode(&decoded); err != nil {
			return NewError("jsonReadFile: %s", err)
		}
		// Reject trailing JSON values and non-whitespace garbage.
		var trailing any
		if err := decoder.Decode(&trailing); err == nil {
			return NewError("jsonReadFile: multiple JSON values are not allowed")
		}
		return goValueToObject(decoded)
	}}
}

// JSONWriteFileBuiltin writes JSON atomically. ByteArray values use the JSON
// standard base64 representation because encoding/json treats []byte that way.
func JSONWriteFileBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) < 2 || len(args) > 3 {
			return NewError("jsonWriteFile expects path, value and optional pretty flag")
		}
		path, errObj := httpString(args[0], "jsonWriteFile path")
		if errObj != nil {
			return errObj
		}
		pretty := false
		if len(args) == 3 {
			value, ok := args[2].(*object.Boolean)
			if !ok {
				return NewError("jsonWriteFile pretty flag must be bool")
			}
			pretty = value.Value
		}
		var data []byte
		var err error
		if pretty {
			data, err = json.MarshalIndent(objectToGoValue(args[1]), "", "  ")
		} else {
			data, err = json.Marshal(objectToGoValue(args[1]))
		}
		if err != nil {
			return NewError("jsonWriteFile: %s", err)
		}
		data = append(data, '\n')
		if err := atomicWriteFile(path, data, 0o644); err != nil {
			return NewError("jsonWriteFile: %s", err)
		}
		return &object.Integer{Value: int64(len(data))}
	}}
}

func BinaryEncodeBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("binaryEncode expects 1 argument, got %d", len(args))
		}
		portable, err := portableEncodeValue(args[0])
		if err != nil {
			return NewError("binaryEncode: %s", err)
		}
		payload, err := json.Marshal(portable)
		if err != nil {
			return NewError("binaryEncode: %s", err)
		}
		encoded := append([]byte(zumbraBinaryEnvelope), payload...)
		return &object.ByteArray{Data: encoded}
	}}
}

func BinaryDecodeBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("binaryDecode expects 1 argument, got %d", len(args))
		}
		data, err := binaryObjectBytes(args[0])
		if err != nil {
			return NewError("binaryDecode: %s", err)
		}
		if !strings.HasPrefix(string(data), zumbraBinaryEnvelope) {
			return NewError("binaryDecode: unsupported or corrupt envelope")
		}
		decoder := json.NewDecoder(strings.NewReader(string(data[len(zumbraBinaryEnvelope):])))
		decoder.UseNumber()
		var portable any
		if err := decoder.Decode(&portable); err != nil {
			return NewError("binaryDecode: %s", err)
		}
		decoded, err := portableDecodeValue(portable)
		if err != nil {
			return NewError("binaryDecode: %s", err)
		}
		return decoded
	}}
}

func BinaryWriteFileBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("binaryWriteFile expects path and value")
		}
		path, errObj := httpString(args[0], "binaryWriteFile path")
		if errObj != nil {
			return errObj
		}
		encoded := BinaryEncodeBuiltin().Fn(args[1])
		if errObj, ok := encoded.(*object.Error); ok {
			return errObj
		}
		data := encoded.(*object.ByteArray).Data
		if err := atomicWriteFile(path, data, 0o600); err != nil {
			return NewError("binaryWriteFile: %s", err)
		}
		return &object.Integer{Value: int64(len(data))}
	}}
}

func BinaryReadFileBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("binaryReadFile expects path")
		}
		path, errObj := httpString(args[0], "binaryReadFile path")
		if errObj != nil {
			return errObj
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return NewError("binaryReadFile: %s", err)
		}
		return BinaryDecodeBuiltin().Fn(&object.ByteArray{Data: data})
	}}
}

func atomicWriteFile(path string, data []byte, mode os.FileMode) error {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".zumbra-write-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(mode); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}

func portableEncodeValue(value object.Object) (any, error) {
	switch current := value.(type) {
	case nil, *object.Null:
		return nil, nil
	case *object.Boolean:
		return current.Value, nil
	case *object.Integer:
		return current.Value, nil
	case *object.Float:
		return current.Value, nil
	case *object.String:
		return current.Value, nil
	case *object.FixedInteger:
		if current.Kind.Signed() {
			return current.SignedValue(), nil
		}
		if current.UnsignedValue() <= uint64(^uint64(0)>>1) && current.UnsignedValue() <= uint64(1<<63-1) {
			return int64(current.UnsignedValue()), nil
		}
		return map[string]any{"$uint": strconv.FormatUint(current.UnsignedValue(), 10)}, nil
	case *object.ByteArray:
		return map[string]any{"$bytes": hex.EncodeToString(current.Data)}, nil
	case *object.TypedArray:
		return map[string]any{"$bytes": hex.EncodeToString(current.Data)}, nil
	case *object.Slice:
		values := make([]any, 0, current.Length)
		for index := 0; index < current.Length; index++ {
			element, handled, err := collections.Get(current, &object.Integer{Value: int64(index)})
			if err != nil || !handled {
				return nil, fmt.Errorf("cannot read slice index %d", index)
			}
			encoded, err := portableEncodeValue(element)
			if err != nil {
				return nil, err
			}
			values = append(values, encoded)
		}
		return values, nil
	case *object.Array:
		values := make([]any, 0, len(current.Elements))
		for _, element := range current.Elements {
			encoded, err := portableEncodeValue(element)
			if err != nil {
				return nil, err
			}
			values = append(values, encoded)
		}
		return values, nil
	case *object.Dict:
		values := make(map[string]any, len(current.Pairs))
		for _, pair := range current.Pairs {
			key, ok := pair.Key.(*object.String)
			if !ok {
				return nil, fmt.Errorf("dictionary keys must be strings for serialization")
			}
			encoded, err := portableEncodeValue(pair.Value)
			if err != nil {
				return nil, err
			}
			values[key.Value] = encoded
		}
		return values, nil
	case *object.Date:
		dateValue := current.FullDate
		if dateValue.IsZero() {
			dateValue = time.Date(current.Year, current.Month, current.Day, current.Hour, current.Minute, current.Second, 0, time.UTC)
		}
		return dateValue.Format(time.RFC3339Nano), nil
	case *object.Record:
		values := make(map[string]any, len(current.Fields))
		for key, raw := range current.Fields {
			runtimeValue, ok := raw.(object.Object)
			if !ok {
				runtimeValue = goValueToObject(raw)
			}
			encoded, err := portableEncodeValue(runtimeValue)
			if err != nil {
				return nil, err
			}
			values[key] = encoded
		}
		return values, nil
	case *object.StructInstance:
		values := make(map[string]any, len(current.Fields))
		for key, runtimeValue := range current.Fields {
			encoded, err := portableEncodeValue(runtimeValue)
			if err != nil {
				return nil, err
			}
			values[key] = encoded
		}
		return values, nil
	case *object.EnumValue:
		return current.Inspect(), nil
	default:
		return nil, fmt.Errorf("unsupported runtime type %s", value.Type())
	}
}

func portableDecodeValue(value any) (object.Object, error) {
	switch current := value.(type) {
	case nil:
		return &object.Null{}, nil
	case bool:
		return &object.Boolean{Value: current}, nil
	case string:
		return &object.String{Value: current}, nil
	case json.Number:
		if integer, err := current.Int64(); err == nil {
			return &object.Integer{Value: integer}, nil
		}
		floating, err := current.Float64()
		if err != nil {
			return nil, err
		}
		return &object.Float{Value: floating}, nil
	case float64:
		return floatGoValueToObject(current), nil
	case []any:
		elements := make([]object.Object, 0, len(current))
		for _, element := range current {
			decoded, err := portableDecodeValue(element)
			if err != nil {
				return nil, err
			}
			elements = append(elements, decoded)
		}
		return &object.Array{Elements: elements}, nil
	case map[string]any:
		if raw, ok := current["$bytes"].(string); ok && len(current) == 1 {
			data, err := hex.DecodeString(raw)
			if err != nil {
				return nil, err
			}
			return &object.ByteArray{Data: data}, nil
		}
		if raw, ok := current["$uint"].(string); ok && len(current) == 1 {
			value, err := strconv.ParseUint(raw, 10, 64)
			if err != nil {
				return nil, err
			}
			return object.NewFixedIntegerRaw(object.FixedU64, value), nil
		}
		pairs := make(map[object.DictKey]object.DictPair, len(current))
		for keyText, rawValue := range current {
			decoded, err := portableDecodeValue(rawValue)
			if err != nil {
				return nil, err
			}
			key := &object.String{Value: keyText}
			pairs[key.DictKey()] = object.DictPair{Key: key, Value: decoded}
		}
		return &object.Dict{Pairs: pairs}, nil
	default:
		return nil, fmt.Errorf("unsupported serialized value %T", value)
	}
}

func binaryObjectBytes(value object.Object) ([]byte, error) {
	switch current := value.(type) {
	case *object.ByteArray:
		result := make([]byte, len(current.Data))
		copy(result, current.Data)
		return result, nil
	case *object.TypedArray:
		if current.Kind != object.FixedU8 && current.Kind != object.FixedI8 {
			return nil, fmt.Errorf("typed array must use u8 or i8")
		}
		result := make([]byte, current.Length)
		copy(result, current.Data[:current.Length])
		return result, nil
	case *object.Slice:
		result := make([]byte, current.Length)
		for index := 0; index < current.Length; index++ {
			element, handled, err := collections.Get(current, &object.Integer{Value: int64(index)})
			if err != nil || !handled {
				return nil, fmt.Errorf("cannot read slice index %d", index)
			}
			fixed, ok := element.(*object.FixedInteger)
			if !ok || (fixed.Kind != object.FixedU8 && fixed.Kind != object.FixedI8) {
				return nil, fmt.Errorf("slice must contain u8 or i8 values")
			}
			result[index] = byte(fixed.UnsignedValue())
		}
		return result, nil
	default:
		return nil, fmt.Errorf("expected ByteArray, u8/i8 typed array or byte slice, got %s", value.Type())
	}
}
