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
	numDefinitions  int
	FreeSymbols     []Symbol
	isFunctionScope bool
}

func NewEnclosedSymbolTable(outer *SymbolTable) *SymbolTable {
	return &SymbolTable{
		store:           make(map[string]Symbol),
		Outer:           outer,
		numDefinitions:  0,
		FreeSymbols:     []Symbol{},
		isFunctionScope: true,
	}
}

func NewBlockSymbolTable(outer *SymbolTable) *SymbolTable {
	numDefinitions := 0
	if outer != nil {
		numDefinitions = outer.numDefinitions
	}

	return &SymbolTable{
		store:           make(map[string]Symbol),
		Outer:           outer,
		numDefinitions:  numDefinitions,
		FreeSymbols:     []Symbol{},
		isFunctionScope: false,
	}
}

func NewSymbolTable() *SymbolTable {
	s := make(map[string]Symbol)
	free := []Symbol{}
	return &SymbolTable{
		store:           s,
		FreeSymbols:     free,
		isFunctionScope: false,
	}
}

func (s *SymbolTable) Define(name string) Symbol {
	symbol := Symbol{Name: name, Index: s.numDefinitions}
	if s.Outer == nil {
		symbol.Scope = GlobalScope
	} else {
		symbol.Scope = LocalScope
	}

	s.store[name] = symbol
	s.numDefinitions++

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
		free := s.DefineFree(obj)
		return free, true
	}

	return obj, true
}

func (s *SymbolTable) DefineBuiltin(index int, name string) Symbol {
	symbol := Symbol{Name: name, Index: index, Scope: BuiltinScope}
	s.store[name] = symbol
	return symbol
}

func (s *SymbolTable) DefineFree(original Symbol) Symbol {
	s.FreeSymbols = append(s.FreeSymbols, original)

	symbol := Symbol{Name: original.Name, Index: len(s.FreeSymbols) - 1}
	symbol.Scope = FreeScope

	s.store[original.Name] = symbol
	return symbol
}

func (s *SymbolTable) DefineFunctionName(name string) Symbol {
	symbol := Symbol{Name: name, Scope: FunctionScope, Index: 0}
	s.store[name] = symbol
	return symbol
}
