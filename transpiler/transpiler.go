package transpiler

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"zumbra/ast"
	"zumbra/lexer"
	"zumbra/parser"
	"zumbra/runtime"
)

func ZumbraTranspiler(zum string) (string, error) {
	prelude, lowered, err := lowerZ5Program(zum)
	if err != nil {
		return "", err
	}
	zum = lowered
	zum = strings.ReplaceAll(zum, "\r\n", "\n")
	zum = strings.ReplaceAll(zum, "\r", "\n")
	lines := strings.Split(zum, "\n")
	var goBody []string
	var blockStack []string

	inRestHandler := false
	restDepth := 0
	restMethod := ""
	restPath := ""
	restReq := ""
	restRes := ""
	var restBody []string
	var restBlockStack []string

	for _, rawLine := range lines {
		line := sanitizeLine(rawLine)
		if line == "" {
			continue
		}

		if inRestHandler {
			trimmed := strings.TrimSpace(line)

			restDepth += strings.Count(trimmed, "{")
			restDepth -= strings.Count(trimmed, "}")

			if restDepth == 0 {
				handlerBody := translateProgramLines(restBody, &restBlockStack)
				goBody = append(goBody, fmt.Sprintf(
					`%s(%s, func(%s *ZRequest, %s *ZResponse) {
%s
})`,
					restMethod,
					restPath,
					restReq,
					restRes,
					indentLines(strings.Join(handlerBody, "\n"), 1),
				))

				inRestHandler = false
				restDepth = 0
				restMethod = ""
				restPath = ""
				restReq = ""
				restRes = ""
				restBody = nil
				restBlockStack = nil
				continue
			}

			restBody = append(restBody, line)
			continue
		}

		if method, path, reqName, resName, ok := parseRestStart(line); ok {
			inRestHandler = true
			restDepth = 1
			restMethod = method
			restPath = path
			restReq = reqName
			restRes = resName
			restBody = []string{}
			restBlockStack = []string{}
			continue
		}

		goBody = append(goBody, translateProgramLines([]string{line}, &blockStack)...)
	}

	if inRestHandler {
		return "", fmt.Errorf("unterminated REST handler block")
	}

	return fmt.Sprintf(`package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"math/rand"
	"net"
	"net/http"
	neturl "net/url"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

%s

%s

func main() {
%s
}
`, runtime.Runtime(), prelude, indentLines(strings.Join(goBody, "\n"), 1)), nil
}

func translateProgramLines(lines []string, blockStack *[]string) []string {
	var out []string

	inResponseJSONMap := false
	responseJSONTarget := ""
	var responseJSONLines []string

	flushResponseJSON := func() {
		if !inResponseJSONMap {
			return
		}

		out = append(out, fmt.Sprintf("%s = responseJSON(%s, map[string]interface{}{", responseJSONTarget, responseJSONTarget))
		for _, l := range responseJSONLines {
			line := sanitizeLine(l)
			if line == "" {
				continue
			}

			line = translateRequestFields(line)

			if line != "}" && line != "})" && !strings.HasSuffix(line, ",") {
				line += ","
			}

			out = append(out, line)
		}
		out = append(out, "})")

		inResponseJSONMap = false
		responseJSONTarget = ""
		responseJSONLines = nil
	}

	for _, raw := range lines {
		line := sanitizeLine(raw)
		if line == "" {
			continue
		}

		if inResponseJSONMap {
			if line == "})" || line == "}" {
				flushResponseJSON()
				continue
			}

			responseJSONLines = append(responseJSONLines, line)
			continue
		}

		line = translateDictLiteral(line)
		line = translateRequestFields(line)
		line = translateResponseCalls(line)
		line = translateAsyncErrorSyntax(line)
		line = translateFixedIntegerSyntax(line)
		line = translateMemoryCalls(line)
		// Field assignment must be recognized before attribute reads are lowered.
		if strings.Contains(line, "<<") {
			if target, property, value, ok := splitAttributeAssignment(line); ok {
				value = translateZ5Inline(translateLogicalExpression(value))
				out = append(out, fmt.Sprintf("zSetAttr(%s, %q, %s)", target, property, value))
				continue
			}
		}
		line = translateZ5Inline(line)

		if strings.HasSuffix(line, ".json({") || strings.HasSuffix(line, ".json(map[string]interface{}{") {
			dot := strings.Index(line, ".")
			responseJSONTarget = strings.TrimSpace(line[:dot])
			inResponseJSONMap = true
			responseJSONLines = []string{}
			continue
		}

		if strings.HasPrefix(line, "if (") {
			condition := strings.TrimPrefix(line, "if (")
			condition = strings.TrimSuffix(condition, "){")
			condition = strings.TrimSuffix(condition, ") {")
			condition = strings.TrimSpace(condition)
			condition = translateLogicalExpression(condition)

			out = append(out, fmt.Sprintf("if isTruthy(%s) {", condition))
			*blockStack = append(*blockStack, "if")
			continue
		}

		if line == "else {" || line == "} else {" || strings.HasPrefix(line, "else") {
			if len(*blockStack) > 0 && ((*blockStack)[len(*blockStack)-1] == "if" || (*blockStack)[len(*blockStack)-1] == "if-else") {
				out = append(out, "} else {")
				(*blockStack)[len(*blockStack)-1] = "if-else"
				continue
			}
		}

		if strings.HasPrefix(line, "while (") {
			condition := strings.TrimPrefix(line, "while (")
			condition = strings.TrimSuffix(condition, "){")
			condition = strings.TrimSuffix(condition, ") {")
			condition = strings.TrimSpace(condition)
			condition = translateLogicalExpression(condition)

			out = append(out, fmt.Sprintf("for isTruthy(%s) {", condition))
			*blockStack = append(*blockStack, "while")
			continue
		}

		if line == "}" {
			out = append(out, "}")
			if len(*blockStack) > 0 {
				*blockStack = (*blockStack)[:len(*blockStack)-1]
			}
			continue
		}

		if strings.HasPrefix(line, "show(") {
			content := strings.TrimPrefix(line, "show(")
			content = strings.TrimSuffix(content, ")")
			content = translateIndexReads(content)
			args := splitArgs(content)

			if len(args) == 0 {
				out = append(out, `fmt.Println()`)
				continue
			}

			if len(args) == 1 {
				out = append(out, fmt.Sprintf("fmt.Println(%s)", strings.TrimSpace(args[0])))
				continue
			}

			format := args[0]
			if strings.HasPrefix(format, `"`) && strings.HasSuffix(format, `"`) {
				format = format[1 : len(format)-1]
			}

			formatGo := strings.ReplaceAll(format, "{}", "%v")
			stmt := fmt.Sprintf(`fmt.Printf("%s\n"`, formatGo)
			if len(args) > 1 {
				stmt += ", " + strings.Join(args[1:], ", ")
			}
			stmt += ")"
			out = append(out, stmt)
			continue
		}

		if strings.HasPrefix(line, "const ") {
			line = "var " + strings.TrimPrefix(line, "const ")
		}

		if strings.HasPrefix(line, "var ") {
			line = strings.Replace(line, "<<", "=", 1)
			line = translateLogicalExpression(line)
			line = translateRequestFields(line)
			line = translateResponseCalls(line)
			line = translateInlineDictAssignment(line)
			line = translateIndexReads(line)

			if strings.Contains(line, "jsonParse(") {
				parts := strings.SplitN(line, "=", 2)
				varName := strings.TrimSpace(parts[0])
				rightSide := strings.TrimSpace(parts[1])
				out = append(out, fmt.Sprintf("%s map[string]interface{} = %s", varName, rightSide))
				continue
			}

			if strings.Contains(line, "jwtCreateToken(") {
				line = strings.ReplaceAll(line, "=", ", _ =")
				out = append(out, line)
				continue
			}

			out = append(out, line)
			continue
		}

		if strings.Contains(line, "<<") {
			if target, property, value, ok := splitAttributeAssignment(line); ok {
				value = translateZ5Inline(translateLogicalExpression(value))
				out = append(out, fmt.Sprintf("zSetAttr(%s, %q, %s)", target, property, value))
				continue
			}
			if target, index, value, ok := splitIndexAssignment(line); ok {
				target = translateIndexReads(target)
				index = translateIndexReads(translateLogicalExpression(index))
				value = translateIndexReads(translateLogicalExpression(value))
				out = append(out, fmt.Sprintf("zSet(%s, %s, %s)", target, index, value))
				continue
			}
			line = strings.Replace(line, "<<", "=", 1)
			line = translateLogicalExpression(line)
			line = translateRequestFields(line)
			line = translateResponseCalls(line)
			line = translateIndexReads(line)
			out = append(out, line)
			continue
		}

		if strings.HasPrefix(line, "//") {
			out = append(out, line)
			continue
		}

		if strings.Contains(line, "(") && strings.Contains(line, ")") {
			if strings.HasPrefix(line, "addToArrayStart(") || strings.HasPrefix(line, "addToArrayEnd(") {
				parts := strings.SplitN(line, "(", 2)
				args := parts[1][:len(parts[1])-1]
				funcName := parts[0]
				targetVar := strings.Split(args, ",")[0]
				out = append(out, fmt.Sprintf("%s = %s(%s)", strings.TrimSpace(targetVar), funcName, args))
				continue
			}

			line = translateIndexReads(line)
			out = append(out, line)
			continue
		}
	}

	flushResponseJSON()
	return out
}

func sanitizeLine(raw string) string {
	line := strings.ReplaceAll(raw, "\r", "")

	if idx := strings.Index(line, " //"); idx != -1 {
		line = line[:idx]
	}

	line = strings.TrimSpace(line)
	line = strings.TrimSuffix(line, ";")
	line = strings.TrimSpace(line)

	return line
}
func splitArgs(input string) []string {
	var args []string
	var curr strings.Builder
	inStr := false
	parens := 0
	brackets := 0
	braces := 0

	for i := 0; i < len(input); i++ {
		ch := input[i]

		if ch == '"' {
			inStr = !inStr
		}

		if !inStr {
			switch ch {
			case '(':
				parens++
			case ')':
				parens--
			case '[':
				brackets++
			case ']':
				brackets--
			case '{':
				braces++
			case '}':
				braces--
			}
		}

		if ch == ',' && !inStr && parens == 0 && brackets == 0 && braces == 0 {
			args = append(args, strings.TrimSpace(curr.String()))
			curr.Reset()
		} else {
			curr.WriteByte(ch)
		}
	}

	if curr.Len() > 0 {
		args = append(args, strings.TrimSpace(curr.String()))
	}

	return args
}

func translateAsyncErrorSyntax(line string) string {
	line = strings.TrimSpace(line)

	if line == "return" || line == "return;" {
		return "return"
	}

	if strings.HasPrefix(line, "await ") {
		line = strings.TrimPrefix(line, "await ")
	}

	if strings.HasPrefix(line, "try await ") {
		line = strings.TrimPrefix(line, "try await ")
	} else if strings.HasPrefix(line, "try ") {
		line = strings.TrimPrefix(line, "try ")
	}

	line = strings.ReplaceAll(line, " await ", " ")
	line = strings.ReplaceAll(line, " try ", " ")

	return line
}

func translateLogicalExpression(expr string) string {
	expr = strings.TrimSpace(expr)
	expr = translateFixedIntegerSyntax(expr)

	if expr == "" {
		return expr
	}

	if strings.Contains(expr, " or {") {
		return expr
	}

	if idx := findTopLevelLogical(expr, " or "); idx != -1 {
		left := strings.TrimSpace(expr[:idx])
		right := strings.TrimSpace(expr[idx+4:])
		return fmt.Sprintf("zOr(%s, %s)", translateLogicalExpression(left), translateLogicalExpression(right))
	}

	if idx := findTopLevelLogical(expr, " and "); idx != -1 {
		left := strings.TrimSpace(expr[:idx])
		right := strings.TrimSpace(expr[idx+5:])
		return fmt.Sprintf("zAnd(%s, %s)", translateLogicalExpression(left), translateLogicalExpression(right))
	}

	return expr
}

func findTopLevelLogical(input string, op string) int {
	inStr := false
	parens := 0
	brackets := 0
	braces := 0

	for i := 0; i <= len(input)-len(op); i++ {
		ch := input[i]

		if ch == '"' {
			inStr = !inStr
		}

		if !inStr {
			switch ch {
			case '(':
				parens++
			case ')':
				parens--
			case '[':
				brackets++
			case ']':
				brackets--
			case '{':
				braces++
			case '}':
				braces--
			}
		}

		if !inStr && parens == 0 && brackets == 0 && braces == 0 {
			if strings.HasPrefix(input[i:], op) {
				return i
			}
		}
	}

	return -1
}

var fixedIntegerLiteralPattern = regexp.MustCompile(`(?i)\b(0x[0-9a-f_]+|0b[01_]+|0o[0-7_]+|[0-9][0-9_]*)(u8|u16|u32|u64|i8|i16|i32|i64)\b`)

func translateFixedIntegerSyntax(input string) string {
	input = fixedIntegerLiteralPattern.ReplaceAllStringFunc(input, func(literal string) string {
		lower := strings.ToLower(literal)
		for _, suffix := range []string{"u16", "u32", "u64", "i16", "i32", "i64", "u8", "i8"} {
			if !strings.HasSuffix(lower, suffix) {
				continue
			}
			number := literal[:len(literal)-len(suffix)]
			if suffix == "u64" {
				return fmt.Sprintf("zU64(uint64(%s))", number)
			}
			return fmt.Sprintf("%s(%s)", goFixedIntegerType(suffix), number)
		}
		return literal
	})

	for zumbraType, goType := range map[string]string{
		"u8": "zU8", "u16": "zU16", "u32": "zU32", "u64": "zU64",
		"i8": "zI8", "i16": "zI16", "i32": "zI32", "i64": "zI64",
	} {
		input = regexp.MustCompile(`\b`+zumbraType+`\s*\(`).ReplaceAllString(input, goType+"(")
	}

	replacements := []struct{ old, new string }{
		{" band ", " & "},
		{" bor ", " | "},
		{" bxor ", " ^ "},
		{" shl ", " << "},
		{" shr ", " >> "},
		{"bnot ", "^"},
	}
	for _, replacement := range replacements {
		input = strings.ReplaceAll(input, replacement.old, replacement.new)
	}
	return input
}

func goFixedIntegerType(kind string) string {
	switch kind {
	case "u8":
		return "zU8"
	case "u16":
		return "zU16"
	case "u32":
		return "zU32"
	case "u64":
		return "zU64"
	case "i8":
		return "zI8"
	case "i16":
		return "zI16"
	case "i32":
		return "zI32"
	case "i64":
		return "zI64"
	default:
		return kind
	}
}

func translateMemoryCalls(input string) string {
	for name, replacement := range map[string]string{
		"bytes":      "zBytes",
		"arrayOf":    "zArrayOf",
		"slice":      "zSlice",
		"fill":       "zFill",
		"readBytes":  "zReadBytes",
		"writeBytes": "zWriteBytes",
		"readU16LE":  "zReadU16LE",
		"readU16BE":  "zReadU16BE",
		"readU32LE":  "zReadU32LE",
		"readU32BE":  "zReadU32BE",
		"readU64LE":  "zReadU64LE",
		"readU64BE":  "zReadU64BE",
		"writeU16LE": "zWriteU16LE",
		"writeU16BE": "zWriteU16BE",
		"writeU32LE": "zWriteU32LE",
		"writeU32BE": "zWriteU32BE",
		"writeU64LE": "zWriteU64LE",
		"writeU64BE": "zWriteU64BE",
		"copyBytes":  "zCopyBytes",
		"bytesEqual": "zBytesEqual",
		"sha256":     "zSHA256",
	} {
		input = regexp.MustCompile(`\b`+name+`\s*\(`).ReplaceAllString(input, replacement+"(")
	}
	return input
}

var simpleIndexPattern = regexp.MustCompile(`(\bzGet\([^()]*\)|\b[A-Za-z_][A-Za-z0-9_]*)\s*\[([^\[\]]+)\]`)

func translateIndexReads(input string) string {
	for {
		updated := simpleIndexPattern.ReplaceAllString(input, `zGet($1, $2)`)
		if updated == input {
			return input
		}
		input = updated
	}
}

func splitIndexAssignment(line string) (string, string, string, bool) {
	assignment := findTopLevelLogical(line, " << ")
	separatorLength := 4
	if assignment == -1 {
		assignment = strings.Index(line, "<<")
		separatorLength = 2
	}
	if assignment == -1 {
		return "", "", "", false
	}
	left := strings.TrimSpace(line[:assignment])
	value := strings.TrimSpace(line[assignment+separatorLength:])
	if !strings.HasSuffix(left, "]") {
		return "", "", "", false
	}
	depth := 0
	open := -1
	for i := len(left) - 1; i >= 0; i-- {
		switch left[i] {
		case ']':
			depth++
		case '[':
			depth--
			if depth == 0 {
				open = i
				i = -1
			}
		}
	}
	if open <= 0 {
		return "", "", "", false
	}
	return strings.TrimSpace(left[:open]), strings.TrimSpace(left[open+1 : len(left)-1]), value, true
}

func indentLines(input string, level int) string {
	prefix := strings.Repeat("\t", level)
	lines := strings.Split(input, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}

func parseRestStart(line string) (string, string, string, string, bool) {
	prefixes := []string{"restGet", "restPost", "restPut", "restDelete", "restPatch"}

	for _, prefix := range prefixes {
		start := prefix + "("
		if !strings.HasPrefix(line, start) {
			continue
		}

		fctIdx := strings.Index(line, "fct(")
		if fctIdx == -1 {
			return "", "", "", "", false
		}

		beforeFct := strings.TrimSpace(line[len(start):fctIdx])
		beforeFct = strings.TrimSuffix(beforeFct, ",")
		beforeFct = strings.TrimSpace(beforeFct)

		if strings.HasSuffix(beforeFct, "async") {
			beforeFct = strings.TrimSpace(strings.TrimSuffix(beforeFct, "async"))
			beforeFct = strings.TrimSuffix(beforeFct, ",")
			beforeFct = strings.TrimSpace(beforeFct)
		}

		pathArg := beforeFct

		afterFct := line[fctIdx+len("fct("):]
		paramsEnd := strings.Index(afterFct, "){")
		if paramsEnd == -1 {
			paramsEnd = strings.Index(afterFct, ") {")
		}
		if paramsEnd == -1 {
			return "", "", "", "", false
		}

		params := strings.TrimSpace(afterFct[:paramsEnd])
		paramList := splitArgs(params)
		if len(paramList) != 2 {
			return "", "", "", "", false
		}

		return prefix, pathArg, strings.TrimSpace(paramList[0]), strings.TrimSpace(paramList[1]), true
	}

	return "", "", "", "", false
}

func translateResponseCalls(line string) string {
	line = strings.TrimSpace(line)

	type responseCall struct {
		method string
		helper string
	}

	calls := []responseCall{
		{method: ".json(", helper: "responseJSON"},
		{method: ".send(", helper: "responseSend"},
		{method: ".html(", helper: "responseHTML"},
		{method: ".status(", helper: "responseStatus"},
		{method: ".header(", helper: "responseHeader"},
	}

	for _, call := range calls {
		if strings.Contains(line, call.method) {
			dot := strings.Index(line, ".")
			obj := strings.TrimSpace(line[:dot])

			openIdx := strings.Index(line, call.method)
			argsStart := openIdx + len(call.method)
			argsEnd := strings.LastIndex(line, ")")
			if argsEnd == -1 || argsEnd < argsStart {
				return line
			}

			args := strings.TrimSpace(line[argsStart:argsEnd])
			return fmt.Sprintf("%s = %s(%s, %s)", obj, call.helper, obj, args)
		}
	}

	return line
}

func translateRequestFields(line string) string {
	replacements := map[string]string{
		".method":  ".Method",
		".path":    ".Path",
		".params":  ".Params",
		".query":   ".Query",
		".headers": ".Headers",
		".body":    ".Body",
		".rawBody": ".RawBody",
	}

	for old, newVal := range replacements {
		line = strings.ReplaceAll(line, old, newVal)
	}

	return line
}

func translateDictLiteral(line string) string {
	line = strings.TrimSpace(line)

	if line == "{" {
		return "map[string]interface{}{"
	}

	if strings.HasSuffix(line, "{") {
		prefix := strings.TrimSpace(strings.TrimSuffix(line, "{"))
		if strings.HasSuffix(prefix, "(") || strings.HasSuffix(prefix, ",") {
			return line[:len(line)-1] + "map[string]interface{}{"
		}
	}

	return line
}

func translateInlineDictAssignment(line string) string {
	if strings.Contains(line, "= {") {
		return strings.Replace(line, "= {", "= map[string]interface{}{", 1)
	}
	return line
}

func lowerZ5Program(source string) (string, string, error) {
	if !regexp.MustCompile(`\b(const|struct|enum|type|match)\b`).MatchString(source) {
		return "", source, nil
	}
	p := parser.New(lexer.New(source))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		return "", source, fmt.Errorf("z5 lowering parser errors: %s", strings.Join(p.Errors(), "; "))
	}
	replacements := []z5Replacement{}
	prelude := []string{}
	for _, statement := range program.Statements {
		switch node := statement.(type) {
		case *ast.StructStatement:
			prelude = append(prelude, z5StructPrelude(node))
			replacements = append(replacements, z5Replacement{node.Token.Pos.Offset, extendZ5Span(source, node.RBraceToken.Pos.Offset+1), ""})
		case *ast.EnumStatement:
			prelude = append(prelude, z5EnumPrelude(node))
			replacements = append(replacements, z5Replacement{node.Token.Pos.Offset, extendZ5Span(source, node.RBraceToken.Pos.Offset+1), ""})
		case *ast.TypeAliasStatement:
			end := node.Target.Token.Pos.Offset + len(node.Target.Token.Literal)
			replacements = append(replacements, z5Replacement{node.Token.Pos.Offset, extendZ5Span(source, end), ""})
		default:
			collectZ5MatchReplacements(statement, source, &replacements)
		}
	}
	for i := 0; i < len(replacements); i++ {
		for j := i + 1; j < len(replacements); j++ {
			if replacements[j].start > replacements[i].start {
				replacements[i], replacements[j] = replacements[j], replacements[i]
			}
		}
	}
	lowered := source
	for _, item := range replacements {
		if item.start < 0 || item.end > len(lowered) || item.start > item.end {
			continue
		}
		lowered = lowered[:item.start] + item.text + lowered[item.end:]
	}
	return strings.Join(prelude, "\n\n"), lowered, nil
}

func extendZ5Span(source string, end int) int {
	for end < len(source) && (source[end] == ' ' || source[end] == '\t' || source[end] == '\r') {
		end++
	}
	if end < len(source) && source[end] == ';' {
		end++
	}
	return end
}

type z5Replacement struct {
	start, end int
	text       string
}

func collectZ5MatchReplacements(statement ast.Statement, source string, replacements *[]z5Replacement) {
	var visitExpression func(ast.Expression)
	var visitBlock func(*ast.BlockStatement)
	visitBlock = func(block *ast.BlockStatement) {
		if block == nil {
			return
		}
		for _, inner := range block.Statements {
			collectZ5MatchReplacements(inner, source, replacements)
		}
	}
	visitExpression = func(expression ast.Expression) {
		switch node := expression.(type) {
		case *ast.MatchExpression:
			*replacements = append(*replacements, z5Replacement{node.Token.Pos.Offset, extendZ5Span(source, node.RBraceToken.Pos.Offset+1), z5GoExpr(node)})
			return
		case *ast.InfixExpression:
			visitExpression(node.Left)
			visitExpression(node.Right)
		case *ast.PrefixExpression:
			visitExpression(node.Right)
		case *ast.CallExpression:
			visitExpression(node.Function)
			for _, arg := range node.Arguments {
				visitExpression(arg)
			}
		case *ast.AttributeAccess:
			visitExpression(node.Object)
		case *ast.IndexExpression:
			visitExpression(node.Left)
			visitExpression(node.Index)
		case *ast.ArrayLiteral:
			for _, item := range node.Elements {
				visitExpression(item)
			}
		case *ast.DictLiteral:
			for key, value := range node.Pairs {
				visitExpression(key)
				visitExpression(value)
			}
		case *ast.IfExpression:
			visitExpression(node.Condition)
			visitBlock(node.Consequence)
			visitBlock(node.Alternative)
		case *ast.FunctionLiteral:
			visitBlock(node.Body)
		}
	}
	switch node := statement.(type) {
	case *ast.ExpressionStatement:
		visitExpression(node.Expression)
	case *ast.VarStatement:
		visitExpression(node.Value)
	case *ast.ConstStatement:
		visitExpression(node.Value)
	case *ast.AssignStatement:
		visitExpression(node.Value)
	case *ast.AttributeAssignStatement:
		visitExpression(node.Target.Object)
		visitExpression(node.Value)
	case *ast.IndexAssignStatement:
		visitExpression(node.Target)
		visitExpression(node.Value)
	case *ast.ReturnStatement:
		visitExpression(node.ReturnValue)
	case *ast.WhileStatement:
		visitExpression(node.Condition)
		visitBlock(node.Body)
	}
}

func z5StructPrelude(node *ast.StructStatement) string {
	fields := make([]string, 0, len(node.Fields))
	for _, field := range node.Fields {
		fields = append(fields, strconv.Quote(field.Name.Value))
	}
	methods := make([]string, 0, len(node.Methods))
	for _, method := range node.Methods {
		params := method.Function.Parameters
		userParams := params
		if len(userParams) > 0 && userParams[0].Value == "self" {
			userParams = userParams[1:]
		}
		lines := []string{fmt.Sprintf("if len(args) != %d { panic(\"wrong number of method arguments\") }", len(userParams))}
		for index, param := range userParams {
			lines = append(lines, fmt.Sprintf("%s := args[%d]", param.Value, index))
		}
		body := z5GoBlock(method.Function.Body)
		if body != "" {
			lines = append(lines, body)
		} else {
			lines = append(lines, "return nil")
		}
		methods = append(methods, fmt.Sprintf("%q: func(self *zStructInstance, args ...interface{}) interface{} {\n%s\n}", method.Name.Value, indentLines(strings.Join(lines, "\n"), 1)))
	}
	definitionName := "__zumbra_struct_" + node.Name.Value
	methodBody := strings.Join(methods, ",\n")
	if methodBody != "" {
		methodBody += ","
	}
	return fmt.Sprintf("var %s = zStruct(%q, []string{%s}, map[string]zMethod{\n%s\n})\nfunc %s(args ...interface{}) *zStructInstance { return zConstruct(%s, args...) }", definitionName, node.Name.Value, strings.Join(fields, ", "), indentLines(methodBody, 1), node.Name.Value, definitionName)
}

func z5EnumPrelude(node *ast.EnumStatement) string {
	members := make([]string, 0, len(node.Members))
	for _, member := range node.Members {
		members = append(members, strconv.Quote(member.Value))
	}
	return fmt.Sprintf("var %s = zEnum(%q, []string{%s})", node.Name.Value, node.Name.Value, strings.Join(members, ", "))
}

func z5GoBlock(block *ast.BlockStatement) string {
	if block == nil {
		return "return nil"
	}
	lines := []string{}
	for index, statement := range block.Statements {
		last := index == len(block.Statements)-1
		switch node := statement.(type) {
		case *ast.AttributeAssignStatement:
			lines = append(lines, fmt.Sprintf("zSetAttr(%s, %q, %s)", z5GoExpr(node.Target.Object), node.Target.Property.Value, z5GoExpr(node.Value)))
		case *ast.IndexAssignStatement:
			lines = append(lines, fmt.Sprintf("zSet(%s, %s, %s)", z5GoExpr(node.Target.Left), z5GoExpr(node.Target.Index), z5GoExpr(node.Value)))
		case *ast.AssignStatement:
			lines = append(lines, fmt.Sprintf("%s = %s", node.Name.Value, z5GoExpr(node.Value)))
		case *ast.VarStatement:
			lines = append(lines, fmt.Sprintf("var %s = %s", node.Name.Value, z5GoExpr(node.Value)))
		case *ast.ConstStatement:
			lines = append(lines, fmt.Sprintf("var %s = %s", node.Name.Value, z5GoExpr(node.Value)))
		case *ast.ReturnStatement:
			if node.ReturnValue == nil {
				lines = append(lines, "return nil")
			} else {
				lines = append(lines, "return "+z5GoExpr(node.ReturnValue))
			}
		case *ast.ExpressionStatement:
			if last {
				lines = append(lines, "return "+z5GoExpr(node.Expression))
			} else {
				lines = append(lines, z5GoExpr(node.Expression))
			}
		}
	}
	if len(lines) == 0 || !strings.HasPrefix(strings.TrimSpace(lines[len(lines)-1]), "return") {
		lines = append(lines, "return nil")
	}
	return strings.Join(lines, "\n")
}

func z5GoExpr(expression ast.Expression) string {
	switch node := expression.(type) {
	case nil:
		return "nil"
	case *ast.Identifier:
		return node.Value
	case *ast.IntegerLiteral:
		return translateFixedIntegerSyntax(node.String())
	case *ast.FloatLiteral:
		return node.String()
	case *ast.StringLiteral:
		return strconv.Quote(node.Value)
	case *ast.Boolean:
		if node.Value {
			return "true"
		}
		return "false"
	case *ast.InfixExpression:
		return fmt.Sprintf("zBinary(%q, %s, %s)", node.Operator, z5GoExpr(node.Left), z5GoExpr(node.Right))
	case *ast.PrefixExpression:
		if node.Operator == "!" {
			return fmt.Sprintf("!isTruthy(%s)", z5GoExpr(node.Right))
		}
		if node.Operator == "-" {
			return fmt.Sprintf("zBinary(\"-\", int64(0), %s)", z5GoExpr(node.Right))
		}
		return fmt.Sprintf("%s%s", node.Operator, z5GoExpr(node.Right))
	case *ast.AttributeAccess:
		return fmt.Sprintf("zGetAttr(%s, %q)", z5GoExpr(node.Object), node.Property.Value)
	case *ast.CallExpression:
		args := make([]string, 0, len(node.Arguments))
		for _, arg := range node.Arguments {
			args = append(args, z5GoExpr(arg))
		}
		if attr, ok := node.Function.(*ast.AttributeAccess); ok {
			tail := ""
			if len(args) > 0 {
				tail = ", " + strings.Join(args, ", ")
			}
			return fmt.Sprintf("zCallMethod(%s, %q%s)", z5GoExpr(attr.Object), attr.Property.Value, tail)
		}
		return fmt.Sprintf("%s(%s)", z5GoExpr(node.Function), strings.Join(args, ", "))
	case *ast.IndexExpression:
		return fmt.Sprintf("zGet(%s, %s)", z5GoExpr(node.Left), z5GoExpr(node.Index))
	case *ast.ArrayLiteral:
		items := []string{}
		for _, item := range node.Elements {
			items = append(items, z5GoExpr(item))
		}
		return "[]interface{}{ " + strings.Join(items, ", ") + " }"
	case *ast.DictLiteral:
		items := []string{}
		for key, value := range node.Pairs {
			items = append(items, fmt.Sprintf("%s: %s", z5GoExpr(key), z5GoExpr(value)))
		}
		return "map[string]interface{}{ " + strings.Join(items, ", ") + " }"
	case *ast.MatchExpression:
		cases := []string{}
		for _, candidate := range node.Cases {
			cases = append(cases, fmt.Sprintf("{Pattern: %s, Body: func() interface{} { %s }}", z5GoExpr(candidate.Pattern), z5GoBlock(candidate.Body)))
		}
		fallback := "nil"
		if node.Default != nil {
			fallback = fmt.Sprintf("func() interface{} { %s }", z5GoBlock(node.Default))
		}
		return fmt.Sprintf("zMatch(%s, []zMatchCase{%s}, %s)", z5GoExpr(node.Value), strings.Join(cases, ", "), fallback)
	default:
		return "nil"
	}
}

var z5MethodCallPattern = regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_]*)\.([A-Za-z_][A-Za-z0-9_]*)\(([^()]*)\)`)
var z5AttributePattern = regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_]*)\.([A-Za-z_][A-Za-z0-9_]*)\b`)

func translateZ5Inline(input string) string {
	return transformZ5OutsideStrings(input, translateZ5CodeSegment)
}

func transformZ5OutsideStrings(input string, transform func(string) string) string {
	var out strings.Builder
	start := 0
	inString := false
	escaped := false
	for index := 0; index < len(input); index++ {
		ch := input[index]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				out.WriteString(input[start : index+1])
				start = index + 1
				inString = false
			}
			continue
		}
		if ch == '"' {
			out.WriteString(transform(input[start:index]))
			start = index
			inString = true
		}
	}
	if start < len(input) {
		if inString {
			out.WriteString(input[start:])
		} else {
			out.WriteString(transform(input[start:]))
		}
	}
	return out.String()
}

func translateZ5CodeSegment(input string) string {
	protected := map[string]bool{"method": true, "path": true, "params": true, "query": true, "headers": true, "body": true, "rawBody": true, "json": true, "send": true, "html": true, "status": true, "header": true}
	input = z5MethodCallPattern.ReplaceAllStringFunc(input, func(match string) string {
		parts := z5MethodCallPattern.FindStringSubmatch(match)
		if len(parts) != 4 || protected[parts[2]] {
			return match
		}
		tail := ""
		if strings.TrimSpace(parts[3]) != "" {
			tail = ", " + parts[3]
		}
		return fmt.Sprintf("zCallMethod(%s, %q%s)", parts[1], parts[2], tail)
	})
	input = z5AttributePattern.ReplaceAllStringFunc(input, func(match string) string {
		parts := z5AttributePattern.FindStringSubmatch(match)
		if len(parts) != 3 || protected[parts[2]] {
			return match
		}
		return fmt.Sprintf("zGetAttr(%s, %q)", parts[1], parts[2])
	})
	return input
}

func splitAttributeAssignment(line string) (string, string, string, bool) {
	assignment := strings.Index(line, "<<")
	if assignment == -1 {
		return "", "", "", false
	}
	left := strings.TrimSpace(line[:assignment])
	value := strings.TrimSpace(line[assignment+2:])
	parts := strings.Split(left, ".")
	if len(parts) != 2 {
		return "", "", "", false
	}
	objectName, property := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	valid := regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	if !valid.MatchString(objectName) || !valid.MatchString(property) {
		return "", "", "", false
	}
	return objectName, property, value, true
}
