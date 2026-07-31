package semantic

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"zumbra/ast"
	"zumbra/lexer"
	"zumbra/parser"
)

func Analyze(program *ast.Program) (*Result, []error) {
	r := NewResolver()
	errs := r.Resolve(program)
	return r.Result(), errs
}

func AnalyzeWithResolver(r *Resolver, program *ast.Program) (*Result, []error) {
	if r == nil {
		r = NewResolver()
	}
	errs := r.Resolve(program)
	return r.Result(), errs
}

func AnalyzeModule(entryFile string, program *ast.Program) (*Result, []error) {
	if program == nil {
		return NewResult(), nil
	}

	resolver := NewResolver()
	aggregate := NewResult()
	errorList := []error{}
	warningSet := map[string]struct{}{}

	seen := map[string]bool{}
	loading := map[string]bool{}

	entryAbs := entryFile
	if entryAbs == "" {
		entryAbs = "."
	}
	if abs, err := filepath.Abs(entryAbs); err == nil {
		entryAbs = abs
	}

	analyzeModuleRecursiveSemantic(
		resolver,
		entryAbs,
		program,
		seen,
		loading,
		aggregate,
		&errorList,
		warningSet,
	)

	return aggregate, errorList
}

func AnalyzeModuleFromDir(baseDir string, program *ast.Program) (*Result, []error) {
	entry := filepath.Join(baseDir, "__repl__.zum")
	return AnalyzeModule(entry, program)
}

func analyzeModuleRecursiveSemantic(
	resolver *Resolver,
	currentFile string,
	program *ast.Program,
	seen map[string]bool,
	loading map[string]bool,
	aggregate *Result,
	errorList *[]error,
	warningSet map[string]struct{},
) {
	if program == nil {
		return
	}

	if seen[currentFile] {
		return
	}

	if loading[currentFile] {
		*errorList = append(*errorList, fmt.Errorf("cyclic import detected: %s", currentFile))
		return
	}

	loading[currentFile] = true
	defer delete(loading, currentFile)

	imports := collectImportPaths(program)
	currentDir := filepath.Dir(currentFile)

	for _, importPath := range imports {
		resolvedPath := importPath
		if !filepath.IsAbs(resolvedPath) {
			resolvedPath = filepath.Join(currentDir, resolvedPath)
		}
		resolvedPath = filepath.Clean(resolvedPath)

		if abs, err := filepath.Abs(resolvedPath); err == nil {
			resolvedPath = abs
		}

		if loading[resolvedPath] {
			*errorList = append(*errorList, fmt.Errorf("cyclic import detected: %s", importPath))
			continue
		}

		if seen[resolvedPath] {
			continue
		}

		content, err := os.ReadFile(resolvedPath)
		if err != nil {
			*errorList = append(*errorList, fmt.Errorf("could not read imported file %s: %w", importPath, err))
			continue
		}

		l := lexer.New(string(content))
		p := parser.New(l)
		importedProgram := p.ParseProgram()

		if len(p.Errors()) > 0 {
			*errorList = append(*errorList, fmt.Errorf(
				"could not parse imported file %s:\n\t%s",
				importPath,
				strings.Join(p.Errors(), "\n\t"),
			))
			continue
		}

		analyzeModuleRecursiveSemantic(
			resolver,
			resolvedPath,
			importedProgram,
			seen,
			loading,
			aggregate,
			errorList,
			warningSet,
		)
	}

	result, errs := AnalyzeWithResolver(resolver, program)
	*errorList = append(*errorList, errs...)

	mergeSemanticResults(aggregate, result, warningSet)

	seen[currentFile] = true
}

func collectImportPaths(program *ast.Program) []string {
	if program == nil {
		return nil
	}

	paths := []string{}
	for _, stmt := range program.Statements {
		importStmt, ok := stmt.(*ast.ImportStatement)
		if !ok || importStmt.Path == nil {
			continue
		}
		if importStmt.Path.Value != "" {
			paths = append(paths, importStmt.Path.Value)
		}
	}
	return paths
}

func mergeSemanticResults(dst *Result, src *Result, warningSet map[string]struct{}) {
	if dst == nil || src == nil {
		return
	}

	for fn, fr := range src.Functions {
		dst.Functions[fn] = fr
	}

	for _, w := range src.Warnings {
		if _, exists := warningSet[w.Message]; exists {
			continue
		}
		warningSet[w.Message] = struct{}{}
		dst.Warnings = append(dst.Warnings, w)
	}
}
