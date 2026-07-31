package builtins

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"zumbra/collections"
	"zumbra/object"
)

// goValueToObject converts values returned by JSON decoders and database
// drivers into regular Zumbra runtime objects. The conversion is recursive so
// platform integrations can return nested arrays and dictionaries without
// depending on the HTTP implementation's private helpers.
func goValueToObject(value any) object.Object {
	switch current := value.(type) {
	case nil:
		return &object.Null{}
	case object.Object:
		return current
	case string:
		return &object.String{Value: current}
	case []byte:
		// database/sql commonly returns textual columns as []byte.
		return &object.String{Value: string(current)}
	case json.RawMessage:
		var decoded any
		if err := json.Unmarshal(current, &decoded); err != nil {
			return &object.String{Value: string(current)}
		}
		return goValueToObject(decoded)
	case bool:
		return &object.Boolean{Value: current}
	case int:
		return &object.Integer{Value: int64(current)}
	case int8:
		return &object.Integer{Value: int64(current)}
	case int16:
		return &object.Integer{Value: int64(current)}
	case int32:
		return &object.Integer{Value: int64(current)}
	case int64:
		return &object.Integer{Value: current}
	case uint:
		return unsignedGoValueToObject(uint64(current))
	case uint8:
		return unsignedGoValueToObject(uint64(current))
	case uint16:
		return unsignedGoValueToObject(uint64(current))
	case uint32:
		return unsignedGoValueToObject(uint64(current))
	case uint64:
		return unsignedGoValueToObject(current)
	case float32:
		return floatGoValueToObject(float64(current))
	case float64:
		return floatGoValueToObject(current)
	case json.Number:
		if integer, err := current.Int64(); err == nil {
			return &object.Integer{Value: integer}
		}
		if floating, err := current.Float64(); err == nil {
			return &object.Float{Value: floating}
		}
		return &object.String{Value: current.String()}
	case time.Time:
		return &object.Date{
			FullDate: current,
			Hour:     current.Hour(),
			Minute:   current.Minute(),
			Second:   current.Second(),
			Day:      current.Day(),
			Month:    current.Month(),
			Year:     current.Year(),
		}
	case []any:
		elements := make([]object.Object, 0, len(current))
		for _, element := range current {
			elements = append(elements, goValueToObject(element))
		}
		return &object.Array{Elements: elements}
	case map[string]any:
		pairs := make(map[object.DictKey]object.DictPair, len(current))
		for name, element := range current {
			key := &object.String{Value: name}
			pairs[key.DictKey()] = object.DictPair{Key: key, Value: goValueToObject(element)}
		}
		return &object.Dict{Pairs: pairs}
	default:
		return &object.String{Value: fmt.Sprint(current)}
	}
}

func unsignedGoValueToObject(value uint64) object.Object {
	if value <= math.MaxInt64 {
		return &object.Integer{Value: int64(value)}
	}
	return object.NewFixedIntegerRaw(object.FixedU64, value)
}

func floatGoValueToObject(value float64) object.Object {
	if !math.IsNaN(value) && !math.IsInf(value, 0) && value >= math.MinInt64 && value <= math.MaxInt64 {
		integer := int64(value)
		if float64(integer) == value {
			return &object.Integer{Value: integer}
		}
	}
	return &object.Float{Value: value}
}

// objectToGoValue converts Zumbra runtime objects into values understood by
// encoding/json and database drivers. Unknown runtime-only values fall back to
// Inspect so integrations fail visibly instead of silently dropping data.
func objectToGoValue(value object.Object) any {
	switch current := value.(type) {
	case nil, *object.Null:
		return nil
	case *object.String:
		return current.Value
	case *object.Boolean:
		return current.Value
	case *object.Integer:
		return current.Value
	case *object.Float:
		return current.Value
	case *object.FixedInteger:
		if current.Kind.Signed() {
			return current.SignedValue()
		}
		return current.UnsignedValue()
	case *object.Array:
		result := make([]any, 0, len(current.Elements))
		for _, element := range current.Elements {
			result = append(result, objectToGoValue(element))
		}
		return result
	case *object.Dict:
		result := make(map[string]any, len(current.Pairs))
		for _, pair := range current.Pairs {
			result[pair.Key.Inspect()] = objectToGoValue(pair.Value)
		}
		return result
	case *object.ByteArray:
		result := make([]byte, len(current.Data))
		copy(result, current.Data)
		return result
	case *object.TypedArray:
		result := make([]any, 0, current.Length)
		for index := 0; index < current.Length; index++ {
			result = append(result, objectToGoValue(current.Read(index)))
		}
		return result
	case *object.Slice:
		result := make([]any, 0, current.Length)
		for index := 0; index < current.Length; index++ {
			element, handled, err := collections.Get(current, &object.Integer{Value: int64(index)})
			if err != nil || !handled {
				return current.Inspect()
			}
			result = append(result, objectToGoValue(element))
		}
		return result
	case *object.Date:
		if current.FullDate.IsZero() {
			return fmt.Sprintf("%04d-%02d-%02dT%02d:%02d:%02dZ", current.Year, current.Month, current.Day, current.Hour, current.Minute, current.Second)
		}
		return current.FullDate.Format(time.RFC3339Nano)
	case *object.Record:
		result := make(map[string]any, len(current.Fields))
		for name, element := range current.Fields {
			switch typed := element.(type) {
			case object.Object:
				result[name] = objectToGoValue(typed)
			default:
				result[name] = typed
			}
		}
		return result
	case *object.StructInstance:
		result := make(map[string]any, len(current.Fields))
		for name, element := range current.Fields {
			result[name] = objectToGoValue(element)
		}
		return result
	case *object.EnumValue:
		return current.Inspect()
	default:
		return current.Inspect()
	}
}
