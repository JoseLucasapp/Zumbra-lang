package object

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"sort"
	"strings"
)

// StructFieldDefinition describes one field in declaration order.
type StructFieldDefinition struct {
	Name     string
	TypeName string
}

// StructDefinition is both metadata and a callable constructor.
type StructDefinition struct {
	Name    string
	Fields  []StructFieldDefinition
	Methods map[string]Object
}

func (s *StructDefinition) Type() ObjectType { return STRUCT_DEF_OBJ }
func (s *StructDefinition) Inspect() string  { return fmt.Sprintf("struct %s", s.Name) }

// StructInstance stores only the values of one declared struct.
type StructInstance struct {
	Definition *StructDefinition
	Fields     map[string]Object
}

func (s *StructInstance) Type() ObjectType { return STRUCT_INSTANCE_OBJ }
func (s *StructInstance) Inspect() string {
	var out bytes.Buffer
	name := "struct"
	if s.Definition != nil && s.Definition.Name != "" {
		name = s.Definition.Name
	}
	out.WriteString(name)
	out.WriteString("{")
	keys := make([]string, 0, len(s.Fields))
	for key := range s.Fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		value := s.Fields[key]
		text := "null"
		if value != nil {
			text = value.Inspect()
		}
		parts = append(parts, key+": "+text)
	}
	out.WriteString(strings.Join(parts, ", "))
	out.WriteString("}")
	return out.String()
}

// BoundMethod remembers the receiver while reusing a normal function/closure.
type BoundMethod struct {
	Receiver *StructInstance
	Function Object
}

func (b *BoundMethod) Type() ObjectType { return BOUND_METHOD_OBJ }
func (b *BoundMethod) Inspect() string  { return "bound method" }

// EnumDefinition stores the values declared for an enum.
type EnumDefinition struct {
	Name    string
	Members map[string]*EnumValue
}

func (e *EnumDefinition) Type() ObjectType { return ENUM_DEF_OBJ }
func (e *EnumDefinition) Inspect() string  { return fmt.Sprintf("enum %s", e.Name) }

// EnumValue is a stable, comparable enum member.
type EnumValue struct {
	EnumName string
	Name     string
	Ordinal  int
}

func (e *EnumValue) Type() ObjectType { return ENUM_VALUE_OBJ }
func (e *EnumValue) Inspect() string  { return e.EnumName + "." + e.Name }
func (e *EnumValue) DictKey() DictKey {
	h := fnv.New64a()
	_, _ = h.Write([]byte(e.EnumName + "\x00" + e.Name))
	return DictKey{Type: e.Type(), Value: h.Sum64()}
}
