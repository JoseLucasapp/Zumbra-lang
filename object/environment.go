package object

import (
	"fmt"
	"sync"
)

func NewEnvironment() *Environment {
	return &Environment{
		store:         make(map[string]Object),
		constants:     make(map[string]bool),
		outer:         nil,
		importedFiles: make(map[string]bool),
	}
}

type Environment struct {
	mu            sync.RWMutex
	store         map[string]Object
	constants     map[string]bool
	outer         *Environment
	importedFiles map[string]bool
}

func (e *Environment) Get(name string) (Object, bool) {
	e.mu.RLock()
	obj, ok := e.store[name]
	e.mu.RUnlock()
	if !ok && e.outer != nil {
		obj, ok = e.outer.Get(name)
	}
	return obj, ok
}

func (e *Environment) Set(name string, val Object) Object {
	e.mu.Lock()
	e.store[name] = val
	e.mu.Unlock()
	return val
}

func (e *Environment) DefineConst(name string, val Object) Object {
	e.mu.Lock()
	e.store[name] = val
	e.constants[name] = true
	e.mu.Unlock()
	return val
}

func (e *Environment) IsConst(name string) bool {
	e.mu.RLock()
	isConst := e.constants[name]
	_, local := e.store[name]
	e.mu.RUnlock()
	if isConst {
		return true
	}
	if local {
		return false
	}
	if e.outer != nil {
		return e.outer.IsConst(name)
	}
	return false
}

func (e *Environment) Assign(name string, val Object) (Object, error) {
	e.mu.Lock()
	if _, ok := e.store[name]; ok {
		if e.constants[name] {
			e.mu.Unlock()
			return nil, fmt.Errorf("cannot assign to constant %s", name)
		}
		e.store[name] = val
		e.mu.Unlock()
		return val, nil
	}
	e.mu.Unlock()
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
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.importedFiles == nil {
		return false
	}
	return e.importedFiles[path]
}

func (e *Environment) MarkImported(path string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.importedFiles == nil {
		e.importedFiles = make(map[string]bool)
	}
	e.importedFiles[path] = true
}
