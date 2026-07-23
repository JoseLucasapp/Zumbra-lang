package types

type Kind string

const (
	Unknown    Kind = "unknown"
	Int        Kind = "int"
	U8         Kind = "u8"
	U16        Kind = "u16"
	U32        Kind = "u32"
	U64        Kind = "u64"
	I8         Kind = "i8"
	I16        Kind = "i16"
	I32        Kind = "i32"
	I64        Kind = "i64"
	Float      Kind = "float"
	Bool       Kind = "bool"
	String     Kind = "string"
	Null       Kind = "null"
	Array      Kind = "array"
	ByteArray  Kind = "byte_array"
	TypedArray Kind = "typed_array"
	Slice      Kind = "slice"
	Dict       Kind = "dict"
	Func       Kind = "function"
	Struct     Kind = "struct"
	Enum       Kind = "enum"
	Pointer    Kind = "ptr"
	Task       Kind = "task"
	Channel    Kind = "channel"
	Mutex      Kind = "mutex"
	RWMutex    Kind = "rw_mutex"
	WaitGroup  Kind = "wait_group"
	Semaphore  Kind = "semaphore"
	AtomicInt  Kind = "atomic_int"
)

type Type struct {
	Kind    Kind
	Name    string
	Async   bool
	Elem    *Type
	Key     *Type
	Value   *Type
	Params  []*Type
	Return  *Type
	Fields  map[string]*Type
	Methods map[string]*Type
	Members map[string]bool
}

func Simple(kind Kind) *Type {
	return &Type{Kind: kind}
}

func ArrayOf(elem *Type) *Type {
	return &Type{Kind: Array, Elem: elem}
}

func ByteArrayOf() *Type {
	return &Type{Kind: ByteArray, Elem: Simple(U8)}
}

func TypedArrayOf(elem *Type) *Type {
	return &Type{Kind: TypedArray, Elem: elem}
}

func SliceOf(elem *Type) *Type {
	return &Type{Kind: Slice, Elem: elem}
}

func TaskOf(result *Type) *Type  { return &Type{Kind: Task, Elem: result} }
func ChannelOf(elem *Type) *Type { return &Type{Kind: Channel, Elem: elem} }

func DictOf(key *Type, value *Type) *Type {
	return &Type{
		Kind:  Dict,
		Key:   key,
		Value: value,
	}
}

func FuncOf(params []*Type, ret *Type) *Type {
	return &Type{
		Kind:   Func,
		Params: params,
		Return: ret,
	}
}

func Same(a, b *Type) bool {
	if a == nil || b == nil {
		return false
	}

	if a.Kind != b.Kind {
		return false
	}

	switch a.Kind {
	case Array, ByteArray, TypedArray, Slice, Task, Channel:
		if a.Elem == nil || b.Elem == nil {
			return a.Elem == b.Elem
		}
		return Same(a.Elem, b.Elem)

	case Dict:
		if a.Key == nil || b.Key == nil || a.Value == nil || b.Value == nil {
			return a.Key == b.Key && a.Value == b.Value
		}
		return Same(a.Key, b.Key) && Same(a.Value, b.Value)

	case Struct, Enum:
		return a.Name == b.Name

	case Func:
		if a.Async != b.Async {
			return false
		}
		if len(a.Params) != len(b.Params) {
			return false
		}
		for i := range a.Params {
			if !Same(a.Params[i], b.Params[i]) {
				return false
			}
		}
		if a.Return == nil || b.Return == nil {
			return a.Return == b.Return
		}
		return Same(a.Return, b.Return)
	}

	return true
}

// Compatible accepts partially inferred types while preserving concrete ABI and
// structured type requirements. It is used for function and callback calls.
func Compatible(expected, actual *Type) bool {
	if expected == nil || actual == nil {
		return false
	}
	if expected.Kind == Unknown || actual.Kind == Unknown {
		return true
	}
	if expected.Kind != actual.Kind {
		return false
	}
	switch expected.Kind {
	case Array, ByteArray, TypedArray, Slice, Task, Channel:
		if expected.Elem == nil || actual.Elem == nil {
			return true
		}
		return Compatible(expected.Elem, actual.Elem)
	case Dict:
		return Compatible(expected.Key, actual.Key) && Compatible(expected.Value, actual.Value)
	case Struct, Enum:
		return expected.Name == actual.Name
	case Func:
		if expected.Async != actual.Async {
			return false
		}
		if len(expected.Params) != len(actual.Params) {
			return false
		}
		for index := range expected.Params {
			if !Compatible(expected.Params[index], actual.Params[index]) {
				return false
			}
		}
		if expected.Return == nil || actual.Return == nil {
			return true
		}
		return Compatible(expected.Return, actual.Return)
	default:
		return true
	}
}

func IsNumeric(t *Type) bool {
	if t == nil {
		return false
	}
	return IsInteger(t) || t.Kind == Float
}

func IsInteger(t *Type) bool {
	if t == nil {
		return false
	}
	switch t.Kind {
	case Int, U8, U16, U32, U64, I8, I16, I32, I64:
		return true
	default:
		return false
	}
}

func IsFixedInteger(t *Type) bool {
	return IsInteger(t) && t.Kind != Int
}

func FixedIntegerKind(name string) (Kind, bool) {
	switch Kind(name) {
	case U8, U16, U32, U64, I8, I16, I32, I64:
		return Kind(name), true
	default:
		return Unknown, false
	}
}

func StructOf(name string, fields map[string]*Type, methods map[string]*Type) *Type {
	return &Type{Kind: Struct, Name: name, Fields: fields, Methods: methods}
}

func EnumOf(name string, members []string) *Type {
	set := make(map[string]bool, len(members))
	for _, member := range members {
		set[member] = true
	}
	return &Type{Kind: Enum, Name: name, Members: set}
}

func (t *Type) String() string {
	if t == nil {
		return string(Unknown)
	}
	switch t.Kind {
	case Array, ByteArray, TypedArray, Slice, Task, Channel:
		if t.Elem == nil {
			return string(t.Kind)
		}
		return string(t.Kind) + "<" + t.Elem.String() + ">"
	case Dict:
		key, value := string(Unknown), string(Unknown)
		if t.Key != nil {
			key = t.Key.String()
		}
		if t.Value != nil {
			value = t.Value.String()
		}
		return "dict<" + key + "," + value + ">"
	case Func:
		text := "fct("
		if t.Async {
			text = "async " + text
		}
		for i, param := range t.Params {
			if i > 0 {
				text += ","
			}
			text += param.String()
		}
		text += ")"
		if t.Return != nil {
			text += " -> " + t.Return.String()
		}
		return text
	case Struct, Enum:
		if t.Name != "" {
			return t.Name
		}
	}
	return string(t.Kind)
}
