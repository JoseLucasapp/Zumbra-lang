// Package docgen extracts API documentation from Zumbra source files.
package docgen

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"zumbra/ast"
	"zumbra/lexer"
	"zumbra/parser"
)

type Options struct {
	IncludePrivate bool
	Title          string
}

type Parameter struct {
	Name string `json:"name"`
}

type Member struct {
	Name       string      `json:"name"`
	Kind       string      `json:"kind"`
	Type       string      `json:"type,omitempty"`
	Signature  string      `json:"signature,omitempty"`
	Parameters []Parameter `json:"parameters,omitempty"`
}

type Symbol struct {
	Name        string   `json:"name"`
	Kind        string   `json:"kind"`
	Public      bool     `json:"public"`
	File        string   `json:"file"`
	Line        int      `json:"line"`
	Signature   string   `json:"signature"`
	Description string   `json:"description,omitempty"`
	Members     []Member `json:"members,omitempty"`
}

type Document struct {
	SchemaVersion int      `json:"schema_version"`
	Title         string   `json:"title"`
	GeneratedBy   string   `json:"generated_by"`
	Symbols       []Symbol `json:"symbols"`
}

func Generate(files []string, options Options) (*Document, error) {
	if options.Title == "" {
		options.Title = "Zumbra API"
	}
	document := &Document{SchemaVersion: 1, Title: options.Title, GeneratedBy: "zumbra doc"}
	for _, filename := range files {
		data, err := os.ReadFile(filename)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", filename, err)
		}
		symbols, err := Extract(filename, string(data), options)
		if err != nil {
			return nil, err
		}
		document.Symbols = append(document.Symbols, symbols...)
	}
	sort.Slice(document.Symbols, func(i, j int) bool {
		if document.Symbols[i].File == document.Symbols[j].File {
			if document.Symbols[i].Line == document.Symbols[j].Line {
				return document.Symbols[i].Name < document.Symbols[j].Name
			}
			return document.Symbols[i].Line < document.Symbols[j].Line
		}
		return document.Symbols[i].File < document.Symbols[j].File
	})
	return document, nil
}

func Extract(filename, source string, options Options) ([]Symbol, error) {
	p := parser.New(lexer.New(source))
	program := p.ParseProgram()
	if errors := p.Errors(); len(errors) > 0 {
		return nil, fmt.Errorf("parse %s: %s", filename, strings.Join(errors, "; "))
	}
	docs := documentationByLine(source)
	symbols := []Symbol{}
	for _, statement := range program.Statements {
		symbol, ok := symbolFromStatement(filename, statement, docs)
		if !ok || (!symbol.Public && !options.IncludePrivate) {
			continue
		}
		symbols = append(symbols, symbol)
	}
	return symbols, nil
}

func symbolFromStatement(filename string, statement ast.Statement, docs map[int]string) (Symbol, bool) {
	switch node := statement.(type) {
	case *ast.VarStatement:
		if node.Name == nil {
			return Symbol{}, false
		}
		line := node.Token.Pos.Line
		kind := "variable"
		signature := "var " + node.Name.Value
		members := []Member{}
		if function, ok := node.Value.(*ast.FunctionLiteral); ok {
			kind = "function"
			signature = functionSignature(node.Name.Value, function)
			for _, parameter := range function.Parameters {
				members = append(members, Member{Name: parameter.Value, Kind: "parameter"})
			}
		}
		if node.Public {
			signature = "pub " + signature
		}
		return Symbol{Name: node.Name.Value, Kind: kind, Public: node.Public, File: filepath.ToSlash(filename), Line: line, Signature: signature, Description: docs[line], Members: members}, true
	case *ast.ConstStatement:
		if node.Name == nil {
			return Symbol{}, false
		}
		signature := "const " + node.Name.Value
		if node.Public {
			signature = "pub " + signature
		}
		return Symbol{Name: node.Name.Value, Kind: "constant", Public: node.Public, File: filepath.ToSlash(filename), Line: node.Token.Pos.Line, Signature: signature, Description: docs[node.Token.Pos.Line]}, true
	case *ast.StructStatement:
		if node.Name == nil {
			return Symbol{}, false
		}
		members := make([]Member, 0, len(node.Fields)+len(node.Methods))
		for _, field := range node.Fields {
			member := Member{Name: field.Name.Value, Kind: "field", Type: field.TypeName}
			members = append(members, member)
		}
		for _, method := range node.Methods {
			members = append(members, Member{Name: method.Name.Value, Kind: "method", Signature: functionSignature(method.Name.Value, method.Function)})
		}
		signature := "struct " + node.Name.Value
		if node.Public {
			signature = "pub " + signature
		}
		return Symbol{Name: node.Name.Value, Kind: "struct", Public: node.Public, File: filepath.ToSlash(filename), Line: node.Token.Pos.Line, Signature: signature, Description: docs[node.Token.Pos.Line], Members: members}, true
	case *ast.EnumStatement:
		if node.Name == nil {
			return Symbol{}, false
		}
		members := make([]Member, 0, len(node.Members))
		for _, member := range node.Members {
			members = append(members, Member{Name: member.Value, Kind: "enum-member"})
		}
		signature := "enum " + node.Name.Value
		if node.Public {
			signature = "pub " + signature
		}
		return Symbol{Name: node.Name.Value, Kind: "enum", Public: node.Public, File: filepath.ToSlash(filename), Line: node.Token.Pos.Line, Signature: signature, Description: docs[node.Token.Pos.Line], Members: members}, true
	case *ast.TypeAliasStatement:
		if node.Name == nil || node.Target == nil {
			return Symbol{}, false
		}
		signature := "type " + node.Name.Value + " << " + node.Target.Value
		if node.Public {
			signature = "pub " + signature
		}
		return Symbol{Name: node.Name.Value, Kind: "type", Public: node.Public, File: filepath.ToSlash(filename), Line: node.Token.Pos.Line, Signature: signature, Description: docs[node.Token.Pos.Line]}, true
	case *ast.ExternBlockStatement:
		name := "extern " + strconv.Quote(node.ABI)
		members := make([]Member, 0, len(node.Functions))
		for _, function := range node.Functions {
			parameters := make([]string, 0, len(function.Parameters))
			for _, parameter := range function.Parameters {
				parameters = append(parameters, parameter.Name.Value+": "+parameter.Type.String())
			}
			signature := "fct " + function.Name.Value + "(" + strings.Join(parameters, ", ") + ") -> " + function.ReturnType.String()
			members = append(members, Member{Name: function.Name.Value, Kind: "extern-function", Signature: signature})
		}
		signature := name
		if node.Link != "" {
			signature += " from " + strconv.Quote(node.Link)
		}
		if node.Public {
			signature = "pub " + signature
		}
		return Symbol{Name: name, Kind: "extern", Public: node.Public, File: filepath.ToSlash(filename), Line: node.Token.Pos.Line, Signature: signature, Description: docs[node.Token.Pos.Line], Members: members}, true
	default:
		return Symbol{}, false
	}
}

func functionSignature(name string, function *ast.FunctionLiteral) string {
	parameters := make([]string, 0, len(function.Parameters))
	for _, parameter := range function.Parameters {
		parameters = append(parameters, parameter.Value)
	}
	prefix := ""
	if function.Async {
		prefix = "async "
	}
	return prefix + "fct " + name + "(" + strings.Join(parameters, ", ") + ")"
}

func documentationByLine(source string) map[int]string {
	lines := strings.Split(strings.ReplaceAll(source, "\r\n", "\n"), "\n")
	result := map[int]string{}
	pending := []string{}
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "///") {
			pending = append(pending, strings.TrimSpace(strings.TrimPrefix(trimmed, "///")))
			continue
		}
		if trimmed == "" {
			if len(pending) > 0 {
				pending = nil
			}
			continue
		}
		if len(pending) > 0 {
			result[index+1] = strings.TrimSpace(strings.Join(pending, "\n"))
			pending = nil
		}
	}
	return result
}

func Markdown(document *Document) string {
	var output strings.Builder
	output.WriteString("# ")
	output.WriteString(document.Title)
	output.WriteString("\n\n")
	output.WriteString("Generated by `zumbra doc`.\n\n")
	if len(document.Symbols) == 0 {
		output.WriteString("No documented symbols.\n")
		return output.String()
	}
	currentFile := ""
	for _, symbol := range document.Symbols {
		if symbol.File != currentFile {
			currentFile = symbol.File
			output.WriteString("## `")
			output.WriteString(currentFile)
			output.WriteString("`\n\n")
		}
		output.WriteString("### ")
		output.WriteString(symbol.Name)
		output.WriteString("\n\n")
		output.WriteString("```zumbra\n")
		output.WriteString(symbol.Signature)
		output.WriteString("\n```\n\n")
		if symbol.Description != "" {
			output.WriteString(strings.ReplaceAll(symbol.Description, "\n", "  \n"))
			output.WriteString("\n\n")
		}
		if len(symbol.Members) > 0 {
			output.WriteString("| Member | Kind | Type / Signature |\n")
			output.WriteString("|---|---|---|\n")
			for _, member := range symbol.Members {
				detail := member.Type
				if detail == "" {
					detail = member.Signature
				}
				output.WriteString("| `" + member.Name + "` | " + member.Kind + " | `" + strings.ReplaceAll(detail, "|", "\\|") + "` |\n")
			}
			output.WriteString("\n")
		}
	}
	return output.String()
}

func JSON(document *Document) ([]byte, error) {
	return json.MarshalIndent(document, "", "  ")
}
