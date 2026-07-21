package semantic

type SymbolKind string
type ScopeKind string

const (
	SymbolVar      SymbolKind = "VAR"
	SymbolFunction SymbolKind = "FUNCTION"
	SymbolParam    SymbolKind = "PARAM"
	SymbolBuiltin  SymbolKind = "BUILTIN"
	SymbolImport   SymbolKind = "IMPORT"
	SymbolConst    SymbolKind = "CONST"
	SymbolStruct   SymbolKind = "STRUCT"
	SymbolEnum     SymbolKind = "ENUM"
	SymbolType     SymbolKind = "TYPE"
)

const (
	ScopeGlobal   ScopeKind = "GLOBAL"
	ScopeFunction ScopeKind = "FUNCTION"
	ScopeBlock    ScopeKind = "BLOCK"
)

type Symbol struct {
	Name        string
	Kind        SymbolKind
	Depth       int
	Mutable     bool
	IsFree      bool
	OriginDepth int
	Used        bool
}

type Scope struct {
	Parent  *Scope
	Symbols map[string]Symbol
	Depth   int
	Kind    ScopeKind
}

func NewGlobalScope() *Scope {
	return &Scope{
		Parent:  nil,
		Symbols: make(map[string]Symbol),
		Depth:   0,
		Kind:    ScopeGlobal,
	}
}

func NewChildScope(parent *Scope, kind ScopeKind) *Scope {
	depth := 0
	if parent != nil {
		depth = parent.Depth + 1
	}

	return &Scope{
		Parent:  parent,
		Symbols: make(map[string]Symbol),
		Depth:   depth,
		Kind:    kind,
	}
}

func (s *Scope) Define(sym Symbol) error {
	if existing, exists := s.Symbols[sym.Name]; exists {
		// Builtins are predeclared conveniences, not reserved words. A local or
		// global declaration may intentionally shadow one, matching the compiler
		// symbol table and keeping common names such as sum available.
		if existing.Kind != SymbolBuiltin || sym.Kind == SymbolBuiltin {
			return ErrDuplicateSymbol(sym.Name)
		}
	}

	s.Symbols[sym.Name] = sym
	return nil
}

func (s *Scope) Resolve(name string) (Symbol, *Scope, bool) {
	for current := s; current != nil; current = current.Parent {
		if sym, ok := current.Symbols[name]; ok {
			return sym, current, true
		}
	}

	return Symbol{}, nil, false
}

func (s *Scope) MarkUsed(name string) bool {
	for current := s; current != nil; current = current.Parent {
		if sym, ok := current.Symbols[name]; ok {
			sym.Used = true
			current.Symbols[name] = sym
			return true
		}
	}
	return false
}
