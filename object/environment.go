package object

import "fmt"

func NewEnvironment() *Environment {
	return &Environment{
		store:         make(map[string]Object),
		constants:     make(map[string]bool),
		outer:         nil,
		importedFiles: make(map[string]bool),
	}
}

type Environment struct {
	store         map[string]Object
	constants     map[string]bool
	outer         *Environment
	importedFiles map[string]bool
}

func (e *Environment) Get(name string) (Object, bool) {
	obj, ok := e.store[name]
	if !ok && e.outer != nil {
		obj, ok = e.outer.Get(name)
	}
	return obj, ok
}

func (e *Environment) Set(name string, val Object) Object {
	e.store[name] = val
	return val
}

func (e *Environment) DefineConst(name string, val Object) Object {
	e.store[name] = val
	e.constants[name] = true
	return val
}

func (e *Environment) IsConst(name string) bool {
	if e.constants[name] {
		return true
	}
	if _, ok := e.store[name]; ok {
		return false
	}
	if e.outer != nil {
		return e.outer.IsConst(name)
	}
	return false
}

func (e *Environment) Assign(name string, val Object) (Object, error) {
	if _, ok := e.store[name]; ok {
		if e.constants[name] {
			return nil, fmt.Errorf("cannot assign to constant %s", name)
		}
		e.store[name] = val
		return val, nil
	}
	if e.outer != nil {
		return e.outer.Assign(name, val)
	}
	return nil, fmt.Errorf("identifier not found: %s", name)
}

func NewEnclosedEnvironment(outer *Environment) *Environment {
	env := &Environment{store: make(map[string]Object), constants: make(map[string]bool), outer: outer}
	if outer != nil && outer.importedFiles != nil {
		env.importedFiles = outer.importedFiles
	} else {
		env.importedFiles = make(map[string]bool)
	}
	return env
}

func (e *Environment) IsImported(path string) bool {
	if e.importedFiles == nil {
		return false
	}
	return e.importedFiles[path]
}

func (e *Environment) MarkImported(path string) {
	if e.importedFiles == nil {
		e.importedFiles = make(map[string]bool)
	}
	e.importedFiles[path] = true
}
