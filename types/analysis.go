package types

import "zumbra/ast"

// Analysis is the reusable typed result of a successful front-end pass.
// It intentionally stores copies so later compiler stages cannot mutate the
// checker's internal scopes.
type Analysis struct {
	NodeTypes  map[ast.Node]*Type
	Globals    map[string]*Type
	NamedTypes map[string]*Type
}

func newAnalysis() *Analysis {
	return &Analysis{
		NodeTypes:  map[ast.Node]*Type{},
		Globals:    map[string]*Type{},
		NamedTypes: map[string]*Type{},
	}
}

// TypeOf returns the inferred type for an AST node.
func (a *Analysis) TypeOf(node ast.Node) *Type {
	if a == nil || node == nil {
		return Simple(Unknown)
	}
	if value, ok := a.NodeTypes[node]; ok {
		return Clone(value)
	}
	return Simple(Unknown)
}

// Global returns a top-level symbol type.
func (a *Analysis) Global(name string) (*Type, bool) {
	if a == nil {
		return nil, false
	}
	value, ok := a.Globals[name]
	return Clone(value), ok
}

// Named returns a resolved alias, struct or enum type.
func (a *Analysis) Named(name string) (*Type, bool) {
	if a == nil {
		return nil, false
	}
	value, ok := a.NamedTypes[name]
	return Clone(value), ok
}

// Clone copies a type graph. Recursive language types are not introduced yet,
// so a straightforward deep copy is sufficient and predictable.
func Clone(value *Type) *Type {
	if value == nil {
		return nil
	}
	copyValue := &Type{Kind: value.Kind, Name: value.Name}
	copyValue.Elem = Clone(value.Elem)
	copyValue.Key = Clone(value.Key)
	copyValue.Value = Clone(value.Value)
	copyValue.Return = Clone(value.Return)
	if value.Params != nil {
		copyValue.Params = make([]*Type, len(value.Params))
		for i, item := range value.Params {
			copyValue.Params[i] = Clone(item)
		}
	}
	if value.Fields != nil {
		copyValue.Fields = make(map[string]*Type, len(value.Fields))
		for name, item := range value.Fields {
			copyValue.Fields[name] = Clone(item)
		}
	}
	if value.Methods != nil {
		copyValue.Methods = make(map[string]*Type, len(value.Methods))
		for name, item := range value.Methods {
			copyValue.Methods[name] = Clone(item)
		}
	}
	if value.Members != nil {
		copyValue.Members = make(map[string]bool, len(value.Members))
		for name, item := range value.Members {
			copyValue.Members[name] = item
		}
	}
	return copyValue
}

func (c *Checker) snapshotAnalysis() *Analysis {
	result := newAnalysis()
	for node, value := range c.nodeTypes {
		result.NodeTypes[node] = Clone(value)
	}
	for name, value := range c.global.values {
		result.Globals[name] = Clone(value)
	}
	for name, value := range c.aliases {
		result.NamedTypes[name] = Clone(value)
	}
	for name, value := range c.structs {
		result.NamedTypes[name] = Clone(value)
	}
	for name, value := range c.enums {
		result.NamedTypes[name] = Clone(value)
	}
	return result
}
