// Package lint implements the official Zumbra static-analysis rules.
package lint

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"unicode"

	"zumbra/ast"
	"zumbra/diagnostics"
	"zumbra/lexer"
	"zumbra/parser"
	"zumbra/pipeline"
	"zumbra/tooling/formatter"
)

type Options struct {
	CheckPipeline     bool
	RequirePublicDocs bool
	MaxLineLength     int
}

type Result struct {
	Diagnostics []diagnostics.Diagnostic `json:"diagnostics"`
	Errors      int                      `json:"errors"`
	Warnings    int                      `json:"warnings"`
	Infos       int                      `json:"infos"`
}

func (r Result) Failed(denyWarnings bool) bool {
	return r.Errors > 0 || (denyWarnings && r.Warnings > 0)
}

func File(filename string, options Options) (Result, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return Result{}, fmt.Errorf("read %s: %w", filename, err)
	}
	return Source(filename, string(data), options), nil
}

func Files(paths []string, options Options) (Result, error) {
	files, err := formatter.Discover(paths)
	if err != nil {
		return Result{}, err
	}
	combined := Result{}
	for _, filename := range files {
		result, err := File(filename, options)
		if err != nil {
			return Result{}, err
		}
		combined.Diagnostics = append(combined.Diagnostics, result.Diagnostics...)
	}
	finalize(&combined)
	return combined, nil
}

func Source(filename, source string, options Options) Result {
	if options.MaxLineLength <= 0 {
		options.MaxLineLength = 120
	}
	result := Result{}
	addTextRules(&result, filename, source, options)

	parserInstance := parser.New(lexer.New(source))
	program := parserInstance.ParseProgram()
	if parseErrors := parserInstance.Errors(); len(parseErrors) > 0 {
		for _, message := range parseErrors {
			result.Diagnostics = append(result.Diagnostics, diagnostics.New(filename, "ZL0001", "zumbra-lint", message, diagnostics.SeverityError))
		}
		finalize(&result)
		return result
	}

	if options.CheckPipeline {
		built, pipelineDiagnostics := pipeline.Build(filename, source, pipeline.Options{Optimize: false})
		for _, item := range pipelineDiagnostics {
			diagnostic := item.Structured()
			diagnostic.Source = "zumbra-lint/" + diagnostic.Source
			result.Diagnostics = append(result.Diagnostics, diagnostic)
		}
		if built != nil {
			for _, item := range built.Warnings {
				diagnostic := item.Structured()
				diagnostic.Source = "zumbra-lint/" + diagnostic.Source
				result.Diagnostics = append(result.Diagnostics, diagnostic)
			}
		}
	}

	addASTRules(&result, filename, source, program, options)
	finalize(&result)
	return result
}

func addTextRules(result *Result, filename, source string, options Options) {
	normalized := strings.ReplaceAll(source, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	for index, line := range lines {
		lineNumber := index + 1
		trimmed := strings.TrimRight(line, " \t")
		if trimmed != line {
			result.Diagnostics = append(result.Diagnostics, at(filename, "ZL1001", "trailing whitespace", diagnostics.SeverityWarning, lineNumber, len([]rune(trimmed))+1, "remove whitespace at the end of the line"))
		}
		leading := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
		if strings.ContainsRune(leading, '\t') {
			result.Diagnostics = append(result.Diagnostics, at(filename, "ZL1002", "tab indentation is not canonical", diagnostics.SeverityWarning, lineNumber, 1, "use spaces; zumbra fmt fixes this automatically"))
		}
		length := len([]rune(line))
		if length > options.MaxLineLength {
			result.Diagnostics = append(result.Diagnostics, at(filename, "ZL1003", fmt.Sprintf("line has %d columns; limit is %d", length, options.MaxLineLength), diagnostics.SeverityWarning, lineNumber, options.MaxLineLength+1, "split the expression or declaration across lines"))
		}
	}
}

func addASTRules(result *Result, filename, source string, program *ast.Program, options Options) {
	documentation := documentedLines(source)
	imports := map[string]int{}
	for _, statement := range program.Statements {
		switch node := statement.(type) {
		case *ast.ImportStatement:
			if node.Path == nil {
				continue
			}
			if previous, exists := imports[node.Path.Value]; exists {
				result.Diagnostics = append(result.Diagnostics, at(filename, "ZL2001", fmt.Sprintf("duplicate import %q; first imported on line %d", node.Path.Value, previous), diagnostics.SeverityWarning, node.Token.Pos.Line, node.Token.Pos.Col, "remove the duplicate import"))
			} else {
				imports[node.Path.Value] = node.Token.Pos.Line
			}
		}
		declarationRules(result, filename, statement, documentation, options)
	}

	walk(program, func(node ast.Node) {
		switch current := node.(type) {
		case *ast.BlockStatement:
			unreachableRules(result, filename, current)
		case *ast.InfixExpression:
			if current.Operator == "==" || current.Operator == "!=" {
				if _, leftBoolean := current.Left.(*ast.Boolean); leftBoolean {
					result.Diagnostics = append(result.Diagnostics, at(filename, "ZL2004", "boolean comparison can be simplified", diagnostics.SeverityInfo, current.Token.Pos.Line, current.Token.Pos.Col, "use the boolean expression directly, with ! when needed"))
				} else if _, rightBoolean := current.Right.(*ast.Boolean); rightBoolean {
					result.Diagnostics = append(result.Diagnostics, at(filename, "ZL2004", "boolean comparison can be simplified", diagnostics.SeverityInfo, current.Token.Pos.Line, current.Token.Pos.Col, "use the boolean expression directly, with ! when needed"))
				}
			}
		}
	})
}

func declarationRules(result *Result, filename string, statement ast.Statement, docs map[int]bool, options Options) {
	public, name, kind, line, column := declarationInfo(statement)
	if name == "" {
		return
	}
	if options.RequirePublicDocs && public && !docs[line] {
		result.Diagnostics = append(result.Diagnostics, at(filename, "ZL2002", fmt.Sprintf("public %s %q has no /// documentation", kind, name), diagnostics.SeverityWarning, line, column, "add a contiguous /// comment immediately above the declaration"))
	}
	if (kind == "struct" || kind == "enum" || kind == "type") && !startsUpper(name) {
		result.Diagnostics = append(result.Diagnostics, at(filename, "ZL2003", fmt.Sprintf("%s name %q should start with an uppercase letter", kind, name), diagnostics.SeverityWarning, line, column, "use PascalCase for named types"))
	}
}

func declarationInfo(statement ast.Statement) (bool, string, string, int, int) {
	switch node := statement.(type) {
	case *ast.VarStatement:
		if node.Name == nil {
			return false, "", "", 0, 0
		}
		kind := "variable"
		if _, ok := node.Value.(*ast.FunctionLiteral); ok {
			kind = "function"
		}
		return node.Public, node.Name.Value, kind, node.Token.Pos.Line, node.Token.Pos.Col
	case *ast.ConstStatement:
		if node.Name == nil {
			return false, "", "", 0, 0
		}
		return node.Public, node.Name.Value, "constant", node.Token.Pos.Line, node.Token.Pos.Col
	case *ast.StructStatement:
		if node.Name == nil {
			return false, "", "", 0, 0
		}
		return node.Public, node.Name.Value, "struct", node.Token.Pos.Line, node.Token.Pos.Col
	case *ast.EnumStatement:
		if node.Name == nil {
			return false, "", "", 0, 0
		}
		return node.Public, node.Name.Value, "enum", node.Token.Pos.Line, node.Token.Pos.Col
	case *ast.TypeAliasStatement:
		if node.Name == nil {
			return false, "", "", 0, 0
		}
		return node.Public, node.Name.Value, "type", node.Token.Pos.Line, node.Token.Pos.Col
	case *ast.ExternBlockStatement:
		return node.Public, "extern " + node.ABI, "extern block", node.Token.Pos.Line, node.Token.Pos.Col
	default:
		return false, "", "", 0, 0
	}
}

func unreachableRules(result *Result, filename string, block *ast.BlockStatement) {
	terminated := false
	for _, statement := range block.Statements {
		if terminated {
			line, column := statementPosition(statement)
			result.Diagnostics = append(result.Diagnostics, at(filename, "ZL2005", "unreachable statement", diagnostics.SeverityWarning, line, column, "remove the statement or move it before the terminating control flow"))
			continue
		}
		terminated = terminates(statement)
	}
}

func terminates(statement ast.Statement) bool {
	switch node := statement.(type) {
	case *ast.ReturnStatement:
		return true
	case *ast.ExpressionStatement:
		switch node.Expression.(type) {
		case *ast.BreakExpression, *ast.ContinueExpression:
			return true
		}
	}
	return false
}

func statementPosition(statement ast.Statement) (int, int) {
	value := reflect.ValueOf(statement)
	if value.Kind() == reflect.Ptr && !value.IsNil() {
		tokenField := value.Elem().FieldByName("Token")
		if tokenField.IsValid() && tokenField.CanInterface() {
			if tokenValue, ok := tokenField.Interface().(interface{ GetPosition() }); ok {
				_ = tokenValue
			}
			position := tokenField.FieldByName("Pos")
			if position.IsValid() {
				line := int(position.FieldByName("Line").Int())
				column := int(position.FieldByName("Col").Int())
				return line, column
			}
		}
	}
	return 1, 1
}

func documentedLines(source string) map[int]bool {
	lines := strings.Split(strings.ReplaceAll(source, "\r\n", "\n"), "\n")
	documented := map[int]bool{}
	pending := false
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "///") {
			pending = true
			continue
		}
		if trimmed == "" {
			pending = false
			continue
		}
		if pending {
			documented[index+1] = true
		}
		pending = false
	}
	return documented
}

func walk(root any, visit func(ast.Node)) {
	seen := map[uintptr]bool{}
	var recurse func(reflect.Value)
	recurse = func(value reflect.Value) {
		if !value.IsValid() {
			return
		}
		if value.Kind() == reflect.Interface {
			if value.IsNil() {
				return
			}
			recurse(value.Elem())
			return
		}
		if value.Kind() == reflect.Ptr {
			if value.IsNil() {
				return
			}
			pointer := value.Pointer()
			if pointer != 0 && seen[pointer] {
				return
			}
			if pointer != 0 {
				seen[pointer] = true
			}
			if value.CanInterface() {
				if node, ok := value.Interface().(ast.Node); ok {
					visit(node)
				}
			}
			recurse(value.Elem())
			return
		}
		switch value.Kind() {
		case reflect.Struct:
			valueType := value.Type()
			for index := 0; index < value.NumField(); index++ {
				field := valueType.Field(index)
				if field.Name == "Token" || field.Name == "RBraceToken" {
					continue
				}
				recurse(value.Field(index))
			}
		case reflect.Slice, reflect.Array:
			for index := 0; index < value.Len(); index++ {
				recurse(value.Index(index))
			}
		}
	}
	recurse(reflect.ValueOf(root))
}

func startsUpper(value string) bool {
	for _, character := range value {
		return unicode.IsUpper(character)
	}
	return false
}

func at(filename, code, message string, severity diagnostics.Severity, line, column int, help string) diagnostics.Diagnostic {
	if line < 1 {
		line = 1
	}
	if column < 1 {
		column = 1
	}
	item := diagnostics.New(filename, code, "zumbra-lint", message, severity)
	item.Range = diagnostics.Range{
		Start: diagnostics.Position{Line: line, Column: column},
		End:   diagnostics.Position{Line: line, Column: column + 1},
	}
	item.Help = help
	return item
}

func finalize(result *Result) {
	sort.SliceStable(result.Diagnostics, func(i, j int) bool {
		left, right := result.Diagnostics[i], result.Diagnostics[j]
		if left.File != right.File {
			return left.File < right.File
		}
		if left.Range.Start.Line != right.Range.Start.Line {
			return left.Range.Start.Line < right.Range.Start.Line
		}
		if left.Range.Start.Column != right.Range.Start.Column {
			return left.Range.Start.Column < right.Range.Start.Column
		}
		return left.Code < right.Code
	})
	result.Errors, result.Warnings, result.Infos = 0, 0, 0
	for _, item := range result.Diagnostics {
		switch item.Severity {
		case diagnostics.SeverityError:
			result.Errors++
		case diagnostics.SeverityWarning:
			result.Warnings++
		default:
			result.Infos++
		}
	}
}
