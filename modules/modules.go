// Package modules resolves Zumbra source files into an isolated module graph.
// Aliased imports expose only pub declarations and are flattened into stable,
// collision-free internal names before semantic analysis, HIR and MIR.
package modules

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"zumbra/ast"
	"zumbra/lexer"
	"zumbra/parser"
)

type Diagnostic struct {
	File    string
	Message string
	Warning bool
}

func (d Diagnostic) Error() string {
	if d.File == "" {
		return d.Message
	}
	return d.File + ": " + d.Message
}

type Import struct {
	Path   string
	Alias  string
	Legacy bool
}

type Unit struct {
	Path      string
	Namespace string
	Entry     bool
	Imports   []Import
	Symbols   map[string]string
	Exports   map[string]string
}

type Graph struct {
	Entry   string
	Units   []*Unit
	Links   []string
	aliases map[string]*unitState
}

type unitState struct {
	Unit
	Program       *ast.Program
	importAliases map[string]*unitState
	legacyNames   map[string]string
}

type resolver struct {
	entryPath   string
	provided    *ast.Program
	units       map[string]*unitState
	loading     map[string]bool
	order       []*unitState
	diagnostics []Diagnostic
	linkSet     map[string]bool
}

// Resolve loads all imports and returns one flattened program. Dependencies are
// emitted before dependants and every aliased module is internally namespaced.
func Resolve(entryFile string, program *ast.Program) (*ast.Program, *Graph, []Diagnostic) {
	entryPath, err := canonicalPath(entryFile)
	if err != nil {
		return program, nil, []Diagnostic{{File: entryFile, Message: err.Error()}}
	}
	r := &resolver{
		entryPath: entryPath,
		provided:  program,
		units:     map[string]*unitState{},
		loading:   map[string]bool{},
		linkSet:   map[string]bool{},
	}
	entry := r.load(entryPath, true)
	if entry == nil || hasErrors(r.diagnostics) {
		return program, nil, r.diagnostics
	}
	flattened := &ast.Program{Statements: []ast.Statement{}}
	for _, state := range r.order {
		rw := newRewriter(state, &r.diagnostics)
		for _, statement := range state.Program.Statements {
			if _, imported := statement.(*ast.ImportStatement); imported {
				continue
			}
			rewritten := rw.statement(statement, true)
			if rewritten != nil {
				flattened.Statements = append(flattened.Statements, rewritten)
			}
		}
	}
	graph := &Graph{Entry: entryPath, aliases: map[string]*unitState{}}
	for _, state := range r.order {
		copyUnit := state.Unit
		graph.Units = append(graph.Units, &copyUnit)
	}
	for path := range r.linkSet {
		graph.Links = append(graph.Links, path)
	}
	sort.Strings(graph.Links)
	return flattened, graph, r.diagnostics
}

func hasErrors(items []Diagnostic) bool {
	for _, item := range items {
		if !item.Warning {
			return true
		}
	}
	return false
}

func canonicalPath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("module filename cannot be empty")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve module path: %w", err)
	}
	return filepath.Clean(absolute), nil
}

func (r *resolver) load(path string, entry bool) *unitState {
	canonical, err := canonicalPath(path)
	if err != nil {
		r.diagnostics = append(r.diagnostics, Diagnostic{File: path, Message: err.Error()})
		return nil
	}
	if r.loading[canonical] {
		r.diagnostics = append(r.diagnostics, Diagnostic{File: canonical, Message: "cyclic module import detected"})
		return nil
	}
	if state := r.units[canonical]; state != nil {
		return state
	}
	r.loading[canonical] = true
	defer delete(r.loading, canonical)

	program := r.provided
	if !entry {
		content, readErr := os.ReadFile(canonical)
		if readErr != nil {
			r.diagnostics = append(r.diagnostics, Diagnostic{File: canonical, Message: fmt.Sprintf("could not read module: %v", readErr)})
			return nil
		}
		p := parser.New(lexer.New(string(content)))
		program = p.ParseProgram()
		for _, message := range p.Errors() {
			r.diagnostics = append(r.diagnostics, Diagnostic{File: canonical, Message: "parser: " + message})
		}
		if len(p.Errors()) != 0 {
			return nil
		}
	}
	state := &unitState{
		Unit: Unit{
			Path:      canonical,
			Namespace: r.namespaceFor(canonical, entry),
			Entry:     entry,
			Symbols:   map[string]string{},
			Exports:   map[string]string{},
		},
		Program:       program,
		importAliases: map[string]*unitState{},
		legacyNames:   map[string]string{},
	}
	r.units[canonical] = state
	r.collectSymbols(state)

	for _, statement := range program.Statements {
		imported, ok := statement.(*ast.ImportStatement)
		if !ok || imported.Path == nil {
			continue
		}
		resolved := imported.Path.Value
		if !filepath.IsAbs(resolved) {
			resolved = filepath.Join(filepath.Dir(canonical), resolved)
		}
		child := r.load(resolved, false)
		if child == nil {
			continue
		}
		alias := ""
		if imported.Alias != nil {
			alias = imported.Alias.Value
		}
		state.Imports = append(state.Imports, Import{Path: child.Path, Alias: alias, Legacy: alias == ""})
		if alias != "" {
			if _, exists := state.importAliases[alias]; exists {
				r.diagnostics = append(r.diagnostics, Diagnostic{File: canonical, Message: fmt.Sprintf("duplicate module alias %q", alias)})
				continue
			}
			if _, own := state.Symbols[alias]; own {
				r.diagnostics = append(r.diagnostics, Diagnostic{File: canonical, Message: fmt.Sprintf("module alias %q conflicts with a declaration", alias)})
				continue
			}
			state.importAliases[alias] = child
		} else {
			r.diagnostics = append(r.diagnostics, Diagnostic{File: canonical, Message: fmt.Sprintf("legacy import %q flattens all declarations; prefer `as <name>`", imported.Path.Value), Warning: true})
			for name, internal := range child.Symbols {
				if previous, exists := state.legacyNames[name]; exists && previous != internal {
					r.diagnostics = append(r.diagnostics, Diagnostic{File: canonical, Message: fmt.Sprintf("legacy imports define %q more than once", name)})
					continue
				}
				if _, own := state.Symbols[name]; own {
					r.diagnostics = append(r.diagnostics, Diagnostic{File: canonical, Message: fmt.Sprintf("legacy import declaration %q conflicts with local declaration", name)})
					continue
				}
				state.legacyNames[name] = internal
			}
		}
	}
	r.order = append(r.order, state)
	return state
}

func (r *resolver) namespaceFor(path string, entry bool) string {
	if entry {
		return ""
	}
	identity := filepath.ToSlash(filepath.Clean(path))
	if relative, err := filepath.Rel(filepath.Dir(r.entryPath), path); err == nil {
		identity = filepath.ToSlash(filepath.Clean(relative))
	}
	sum := sha256.Sum256([]byte(identity))
	return fmt.Sprintf("__zm_%x_", sum[:8])
}

func (r *resolver) collectSymbols(state *unitState) {
	for _, statement := range state.Program.Statements {
		names, public := declarationNames(statement)
		for _, name := range names {
			if name == "" {
				continue
			}
			if _, exists := state.Symbols[name]; exists {
				r.diagnostics = append(r.diagnostics, Diagnostic{File: state.Path, Message: fmt.Sprintf("duplicate top-level declaration %q", name)})
				continue
			}
			internal := name
			if !state.Entry {
				internal = state.Namespace + name
			}
			state.Symbols[name] = internal
			if public {
				state.Exports[name] = internal
			}
		}
		if external, ok := statement.(*ast.ExternBlockStatement); ok && external.Link != "" {
			link := external.Link
			if !filepath.IsAbs(link) {
				link = filepath.Join(filepath.Dir(state.Path), link)
			}
			if absolute, err := filepath.Abs(link); err == nil {
				link = absolute
			}
			external.Link = filepath.Clean(link)
			r.linkSet[external.Link] = true
		}
	}
}

func declarationNames(statement ast.Statement) ([]string, bool) {
	switch s := statement.(type) {
	case *ast.VarStatement:
		if s.Name == nil {
			return nil, s.Public
		}
		return []string{s.Name.Value}, s.Public
	case *ast.ConstStatement:
		if s.Name == nil {
			return nil, s.Public
		}
		return []string{s.Name.Value}, s.Public
	case *ast.StructStatement:
		if s.Name == nil {
			return nil, s.Public
		}
		return []string{s.Name.Value}, s.Public
	case *ast.EnumStatement:
		if s.Name == nil {
			return nil, s.Public
		}
		return []string{s.Name.Value}, s.Public
	case *ast.TypeAliasStatement:
		if s.Name == nil {
			return nil, s.Public
		}
		return []string{s.Name.Value}, s.Public
	case *ast.ExternBlockStatement:
		names := make([]string, 0, len(s.Functions))
		for _, fn := range s.Functions {
			if fn != nil && fn.Name != nil {
				names = append(names, fn.Name.Value)
			}
		}
		return names, s.Public
	default:
		return nil, false
	}
}

// Exported reports the internal symbol behind alias.name.
func (g *Graph) Exported(modulePath, name string) (string, bool) {
	if g == nil {
		return "", false
	}
	for _, unit := range g.Units {
		if unit.Path == modulePath {
			value, ok := unit.Exports[name]
			return value, ok
		}
	}
	return "", false
}
