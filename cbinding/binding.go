// Package cbinding creates conservative Zumbra extern declarations from C
// headers. It intentionally handles portable scalar and pointer prototypes and
// reports unsupported declarations rather than guessing an ABI.
package cbinding

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type Options struct {
	Link   string
	Public bool
}

type Diagnostic struct {
	Declaration string
	Message     string
}

func (d Diagnostic) Error() string {
	if d.Declaration == "" {
		return d.Message
	}
	return d.Message + ": " + d.Declaration
}

type Function struct {
	Name       string
	Params     []Parameter
	ReturnType string
}

type Parameter struct {
	Name string
	Type string
}

type Result struct {
	Source      string
	Functions   []Function
	Diagnostics []Diagnostic
}

var functionPattern = regexp.MustCompile(`(?s)^\s*(.+?)\b([_A-Za-z]\w*)\s*\((.*)\)\s*$`)
var commentsPattern = regexp.MustCompile(`(?s)/\*.*?\*/|//[^\n]*`)
var preprocessorPattern = regexp.MustCompile(`(?m)^\s*#.*$`)

func Generate(header string, options Options) Result {
	cleaned := commentsPattern.ReplaceAllString(header, "")
	cleaned = preprocessorPattern.ReplaceAllString(cleaned, "")
	result := Result{}
	for _, raw := range strings.Split(cleaned, ";") {
		declaration := strings.TrimSpace(raw)
		if declaration == "" || strings.HasPrefix(declaration, "typedef ") {
			continue
		}
		match := functionPattern.FindStringSubmatch(declaration)
		if match == nil {
			if strings.Contains(declaration, "(") {
				result.Diagnostics = append(result.Diagnostics, Diagnostic{Declaration: declaration, Message: "unsupported C declaration"})
			}
			continue
		}
		returned, ok := mapCType(match[1])
		if !ok {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{Declaration: declaration, Message: "unsupported C return type"})
			continue
		}
		function := Function{Name: match[2], ReturnType: returned}
		parameters := strings.TrimSpace(match[3])
		if parameters != "" && parameters != "void" {
			parts, splitOK := splitParameters(parameters)
			if !splitOK {
				result.Diagnostics = append(result.Diagnostics, Diagnostic{Declaration: declaration, Message: "function-pointer parameters require a handwritten callback declaration"})
				continue
			}
			failed := false
			for index, part := range parts {
				parameter, parseOK := parseParameter(part, index)
				if !parseOK {
					result.Diagnostics = append(result.Diagnostics, Diagnostic{Declaration: declaration, Message: "unsupported C parameter"})
					failed = true
					break
				}
				function.Params = append(function.Params, parameter)
			}
			if failed {
				continue
			}
		}
		result.Functions = append(result.Functions, function)
	}
	result.Source = render(result.Functions, options)
	return result
}

func GenerateFile(path string, options Options) (Result, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Result{}, fmt.Errorf("read C header: %w", err)
	}
	return Generate(string(content), options), nil
}

func splitParameters(value string) ([]string, bool) {
	depth := 0
	start := 0
	parts := []string{}
	for index, r := range value {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
			if depth < 0 {
				return nil, false
			}
		case ',':
			if depth == 0 {
				parts = append(parts, strings.TrimSpace(value[start:index]))
				start = index + 1
			}
		}
	}
	if depth != 0 {
		return nil, false
	}
	parts = append(parts, strings.TrimSpace(value[start:]))
	for _, part := range parts {
		if strings.Contains(part, "(") {
			return nil, false
		}
	}
	return parts, true
}

func parseParameter(raw string, index int) (Parameter, bool) {
	value := strings.TrimSpace(raw)
	if mapped, ok := mapCType(value); ok {
		return Parameter{Name: fmt.Sprintf("arg%d", index), Type: mapped}, true
	}
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return Parameter{}, false
	}
	name := fmt.Sprintf("arg%d", index)
	typeText := value
	last := fields[len(fields)-1]
	if isIdentifier(strings.TrimLeft(last, "*")) {
		pointerPrefix := last[:len(last)-len(strings.TrimLeft(last, "*"))]
		name = strings.TrimLeft(last, "*")
		typeText = strings.TrimSpace(strings.TrimSuffix(value, last)) + pointerPrefix
	}
	mapped, ok := mapCType(typeText)
	if !ok {
		return Parameter{}, false
	}
	return Parameter{Name: name, Type: mapped}, true
}

func mapCType(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	value = strings.ReplaceAll(value, "\t", " ")
	value = strings.Join(strings.Fields(value), " ")
	value = strings.TrimPrefix(value, "extern ")
	value = strings.TrimPrefix(value, "static ")
	value = strings.TrimPrefix(value, "inline ")
	value = strings.ReplaceAll(value, " *", "*")
	value = strings.ReplaceAll(value, "* ", "*")
	value = strings.TrimSpace(value)
	switch value {
	case "void":
		return "void", true
	case "bool", "_Bool":
		return "bool", true
	case "char*", "const char*", "char const*":
		return "cstring", true
	case "void*", "const void*", "void const*":
		return "ptr", true
	case "int8_t", "signed char":
		return "i8", true
	case "uint8_t", "unsigned char":
		return "u8", true
	case "int16_t", "short", "short int", "signed short", "signed short int":
		return "i16", true
	case "uint16_t", "unsigned short", "unsigned short int":
		return "u16", true
	case "int32_t", "int", "signed", "signed int":
		return "i32", true
	case "uint32_t", "unsigned", "unsigned int":
		return "u32", true
	case "int64_t", "long long", "long long int", "signed long long", "signed long long int":
		return "i64", true
	case "uint64_t", "unsigned long long", "unsigned long long int":
		return "u64", true
	case "size_t":
		return "usize", true
	case "float", "double":
		return "float", true
	default:
		return "", false
	}
}

func render(functions []Function, options Options) string {
	var out strings.Builder
	if options.Public {
		out.WriteString("pub ")
	}
	out.WriteString("extern \"C\"")
	if options.Link != "" {
		out.WriteString(" from ")
		out.WriteString(strconv.Quote(options.Link))
	}
	out.WriteString(" {\n")
	for _, function := range functions {
		out.WriteString("    fct ")
		out.WriteString(function.Name)
		out.WriteString("(")
		for index, parameter := range function.Params {
			if index > 0 {
				out.WriteString(", ")
			}
			out.WriteString(parameter.Name)
			out.WriteString(": ")
			out.WriteString(parameter.Type)
		}
		out.WriteString(") -> ")
		out.WriteString(function.ReturnType)
		out.WriteString(";\n")
	}
	out.WriteString("}\n")
	return out.String()
}

func isIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for index, r := range value {
		if r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || index > 0 && r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return true
}
