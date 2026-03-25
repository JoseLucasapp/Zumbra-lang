package transpiler

import (
	"fmt"
	"strings"
	"zumbra/runtime"
)

func ZumbraTranspiler(zum string) (string, error) {
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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net"
	"net/http"
	neturl "net/url"
	"os"
	"path/filepath"
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

func main() {
%s
}
`, runtime.Runtime(), indentLines(strings.Join(goBody, "\n"), 1)), nil
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

		if strings.HasPrefix(line, "var ") {
			line = strings.ReplaceAll(line, "<<", "=")
			line = translateLogicalExpression(line)
			line = translateRequestFields(line)
			line = translateResponseCalls(line)
			line = translateInlineDictAssignment(line)

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
			line = strings.ReplaceAll(line, "<<", "=")
			line = translateLogicalExpression(line)
			line = translateRequestFields(line)
			line = translateResponseCalls(line)
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

			out = append(out, line)
			continue
		}
	}

	flushResponseJSON()
	return out
}

func sanitizeLine(raw string) string {
	line := raw
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

func translateLogicalExpression(expr string) string {
	expr = strings.TrimSpace(expr)

	if strings.Contains(expr, " or ") {
		parts := strings.SplitN(expr, " or ", 2)
		return fmt.Sprintf("zOr(%s, %s)", translateLogicalExpression(parts[0]), translateLogicalExpression(parts[1]))
	}

	if strings.Contains(expr, " and ") {
		parts := strings.SplitN(expr, " and ", 2)
		return fmt.Sprintf("zAnd(%s, %s)", translateLogicalExpression(parts[0]), translateLogicalExpression(parts[1]))
	}

	return expr
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
		pathArg := strings.TrimSpace(beforeFct)

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
