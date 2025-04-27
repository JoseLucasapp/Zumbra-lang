package object

func NewEnvironment() *Environment {
	s := make(map[string]Object)
	return &Environment{store: s, outer: nil, importedFiles: make(map[string]bool)}
}

type Environment struct {
	store         map[string]Object
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

func NewEnclosedEnvironment(outer *Environment) *Environment {
	env := NewEnvironment()
	env.outer = outer
	return env
}

func (e *Environment) IsImported(path string) bool {
	return e.importedFiles[path]
}

func (e *Environment) MarkImported(path string) {
	e.importedFiles[path] = true
}
