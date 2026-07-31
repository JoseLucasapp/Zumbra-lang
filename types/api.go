package types

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"zumbra/ast"
	"zumbra/lexer"
	"zumbra/parser"
)

func Analyze(program *ast.Program) []error {
	c := NewChecker()
	return c.Check(program)
}

func AnalyzeWithChecker(c *Checker, program *ast.Program) []error {
	if c == nil {
		c = NewChecker()
	}
	return c.Check(program)
}

func AnalyzeModule(entryFile string, program *ast.Program) []error {
	if program == nil {
		return nil
	}

	checker := NewChecker()
	errorList := []error{}

	seen := map[string]bool{}
	loading := map[string]bool{}

	entryAbs := entryFile
	if entryAbs == "" {
		entryAbs = "."
	}
	if abs, err := filepath.Abs(entryAbs); err == nil {
		entryAbs = abs
	}

	analyzeModuleRecursiveTypes(
		checker,
		entryAbs,
		program,
		seen,
		loading,
		&errorList,
	)

	return errorList
}

func AnalyzeModuleFromDir(baseDir string, program *ast.Program) []error {
	entry := filepath.Join(baseDir, "__repl__.zum")
	return AnalyzeModule(entry, program)
}

func analyzeModuleRecursiveTypes(
	checker *Checker,
	currentFile string,
	program *ast.Program,
	seen map[string]bool,
	loading map[string]bool,
	errorList *[]error,
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

		analyzeModuleRecursiveTypes(
			checker,
			resolvedPath,
			importedProgram,
			seen,
			loading,
			errorList,
		)
	}

	errs := AnalyzeWithChecker(checker, program)
	*errorList = append(*errorList, errs...)

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

// AnalyzeWithInfo checks a program and returns reusable type information.
func AnalyzeWithInfo(program *ast.Program) (*Analysis, []error) {
	checker := NewChecker()
	errs := checker.Check(program)
	return checker.snapshotAnalysis(), errs
}

// AnalyzeModuleWithInfo performs normal module validation and returns type
// information for the entry program. Imported modules remain validated by the
// existing module walker; the returned node map belongs to the entry AST.
func AnalyzeModuleWithInfo(entryFile string, program *ast.Program) (*Analysis, []error) {
	errs := AnalyzeModule(entryFile, program)
	analysis, localErrs := AnalyzeWithInfo(program)
	if len(localErrs) > 0 && len(errs) == 0 {
		errs = append(errs, localErrs...)
	}
	return analysis, errs
}
