package compiler

type SymbolScope string

const (
	LocalScope    SymbolScope = "LOCAL"
	GlobalScope   SymbolScope = "GLOBAL"
	BuiltinScope  SymbolScope = "BUILTIN"
	FreeScope     SymbolScope = "FREE"
	FunctionScope SymbolScope = "FUNCTION"
)

type Symbol struct {
	Name  string
	Scope SymbolScope
	Index int
}

type SymbolTable struct {
	Outer *SymbolTable

	store           map[string]Symbol
	immutable       map[string]bool
	numDefinitions  int
	FreeSymbols     []Symbol
	isFunctionScope bool
}

func NewEnclosedSymbolTable(outer *SymbolTable) *SymbolTable {
	return &SymbolTable{store: make(map[string]Symbol), immutable: make(map[string]bool), Outer: outer, FreeSymbols: []Symbol{}, isFunctionScope: true}
}

func NewBlockSymbolTable(outer *SymbolTable) *SymbolTable {
	numDefinitions := 0
	if outer != nil {
		numDefinitions = outer.numDefinitions
	}
	return &SymbolTable{store: make(map[string]Symbol), immutable: make(map[string]bool), Outer: outer, numDefinitions: numDefinitions, FreeSymbols: []Symbol{}, isFunctionScope: false}
}

func NewSymbolTable() *SymbolTable {
	return &SymbolTable{store: make(map[string]Symbol), immutable: make(map[string]bool), FreeSymbols: []Symbol{}, isFunctionScope: false}
}

func (s *SymbolTable) Define(name string) Symbol {
	symbol := Symbol{Name: name, Index: s.numDefinitions}
	if s.Outer == nil {
		symbol.Scope = GlobalScope
	} else {
		symbol.Scope = LocalScope
	}
	s.store[name] = symbol
	s.immutable[name] = false
	s.numDefinitions++
	return symbol
}

func (s *SymbolTable) DefineConst(name string) Symbol {
	symbol := s.Define(name)
	s.immutable[name] = true
	return symbol
}

func (s *SymbolTable) Resolve(name string) (Symbol, bool) {
	obj, ok := s.store[name]
	if ok {
		return obj, true
	}
	if s.Outer == nil {
		return obj, false
	}
	obj, ok = s.Outer.Resolve(name)
	if !ok {
		return obj, false
	}
	if obj.Scope == GlobalScope || obj.Scope == BuiltinScope {
		return obj, true
	}
	if s.isFunctionScope {
		return s.DefineFree(obj), true
	}
	return obj, true
}

func (s *SymbolTable) IsMutable(name string) bool {
	if _, ok := s.store[name]; ok {
		return !s.immutable[name]
	}
	if s.Outer != nil {
		return s.Outer.IsMutable(name)
	}
	return false
}

func (s *SymbolTable) DefineBuiltin(index int, name string) Symbol {
	symbol := Symbol{Name: name, Index: index, Scope: BuiltinScope}
	s.store[name] = symbol
	s.immutable[name] = true
	return symbol
}

func (s *SymbolTable) DefineFree(original Symbol) Symbol {
	s.FreeSymbols = append(s.FreeSymbols, original)
	symbol := Symbol{Name: original.Name, Index: len(s.FreeSymbols) - 1, Scope: FreeScope}
	s.store[original.Name] = symbol
	return symbol
}

func (s *SymbolTable) DefineFunctionName(name string) Symbol {
	symbol := Symbol{Name: name, Scope: FunctionScope, Index: 0}
	s.store[name] = symbol
	s.immutable[name] = true
	return symbol
}
