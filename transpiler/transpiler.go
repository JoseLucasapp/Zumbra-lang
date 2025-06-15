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

	var inFunction bool
	var funcBuffer []string
	var funcName string
	var funcParams string

	for _, line := range lines {
		if idx := strings.Index(line, "//"); idx != -1 {
			line = line[:idx]
		}
		line = strings.TrimSpace(line)
		line = strings.TrimSuffix(line, ";")
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "var ") && strings.Contains(line, "fct") {
			inFunction = true
			line = strings.TrimPrefix(line, "var ")
			parts := strings.Split(line, "<<")
			funcName = strings.TrimSpace(parts[0])

			functionPart := strings.TrimSpace(parts[1])
			paramStart := strings.Index(functionPart, "(")
			paramEnd := strings.Index(functionPart, "){")
			funcParams = functionPart[paramStart+1 : paramEnd]
			continue
		}

		if inFunction {
			fmt.Println(line)
			if line == "}" {
				bodyLines := funcBuffer
				lastIndex := len(bodyLines) - 1
				lastLine := strings.TrimSpace(bodyLines[lastIndex])
				lastLine = strings.TrimSuffix(lastLine, ";")

				fmt.Println(lastLine)
				if !strings.HasPrefix(lastLine, "return") {
					bodyLines[lastIndex] = "return " + lastLine
				} else {
					bodyLines[lastIndex] = lastLine
				}

				body := strings.Join(bodyLines, " ")

				goBody = append(goBody,
					fmt.Sprintf("var %s = func(%s int, %s int) int { %s }",
						funcName, strings.Split(funcParams, ",")[0], strings.Split(funcParams, ",")[1], body))
				inFunction = false
				funcBuffer = nil
				funcName = ""
				funcParams = ""
				continue
			} else {
				funcBuffer = append(funcBuffer, strings.TrimSpace(line))
			}
			continue
		}

		if strings.HasPrefix(line, "if (") {
			condition := strings.TrimPrefix(line, "if (")
			condition = strings.TrimSuffix(condition, "){")
			condition = strings.TrimSpace(condition)

			goBody = append(goBody, fmt.Sprintf("    if %s {", condition))
			blockStack = append(blockStack, "if")
			continue
		}

		if strings.Contains(line, "else") {
			if len(blockStack) > 0 && blockStack[len(blockStack)-1] == "if" {
				goBody = append(goBody, "    } else {")
				blockStack[len(blockStack)-1] = "if-else"
			}

			continue
		}

		if strings.HasPrefix(line, "while (") {
			condition := strings.TrimPrefix(line, "while (")
			condition = strings.TrimSuffix(condition, ") {")
			condition = strings.TrimSpace(condition)

			goBody = append(goBody, fmt.Sprintf("for %s {", condition))
			blockStack = append(blockStack, "while")
			continue
		}

		if line == "}" {
			goBody = append(goBody, "    }")
			if len(blockStack) > 0 {
				blockStack = blockStack[:len(blockStack)-1]
			}
			continue
		}

		if strings.HasPrefix(line, "show(") {
			content := strings.TrimPrefix(line, "show(")
			content = strings.TrimSuffix(content, ")")
			args := splitArgs(content)

			if len(args) == 0 {
				goBody = append(goBody, `    fmt.Println()`)
				continue
			}

			if len(args) == 1 {
				arg := strings.TrimSpace(args[0])

				if strings.HasPrefix(arg, `"`) && strings.HasSuffix(arg, `"`) {
					if strings.Contains(arg, "{}") {
						goBody = append(goBody, fmt.Sprintf(`    fmt.Println(%s)`, arg))
					} else {
						goBody = append(goBody, fmt.Sprintf(`    fmt.Println(%s)`, arg))
					}
				} else {
					goBody = append(goBody, fmt.Sprintf(`    fmt.Println(%s)`, arg))
				}
				continue
			}

			format := args[0]
			if strings.HasPrefix(format, `"`) && strings.HasSuffix(format, `"`) {
				format = format[1 : len(format)-1]
			}

			placeholders := strings.Count(format, "{}")
			formatGo := strings.ReplaceAll(format, "{}", "%v")

			if placeholders > 0 && len(args)-1 < placeholders {
				goBody = append(goBody, fmt.Sprintf(`    fmt.Println("%s")`, format))
				continue
			}

			line := fmt.Sprintf(`    fmt.Printf("%s\n"`, formatGo)
			if len(args) > 1 {
				line += ", " + strings.Join(args[1:], ", ")
			}
			line += ")"
			goBody = append(goBody, line)
			continue
		}

		if strings.HasPrefix(line, "var ") {
			line = strings.ReplaceAll(line, "<<", "=")
			if strings.Contains(line, "[") && strings.Contains(line, "]") {
				parts := strings.SplitN(line, "=", 2)
				varName := strings.TrimSpace(parts[0])
				arrayPart := strings.TrimSpace(parts[1])
				arrayPart = strings.TrimPrefix(arrayPart, "[")
				arrayPart = strings.TrimSuffix(arrayPart, "]")
				arrayElements := strings.TrimSpace(arrayPart)

				goBody = append(goBody, fmt.Sprintf("    %s = []interface{}{%s}", varName, arrayElements))
			} else if strings.Contains(line, "{") && strings.Contains(line, "}") {
				parts := strings.SplitN(line, "=", 2)
				varName := strings.TrimSpace(parts[0])
				rightSide := strings.TrimSpace(parts[1])
				rightSide = strings.ReplaceAll(rightSide, "{", "map[string]interface{}{")
				line = varName + " = " + rightSide
				goBody = append(goBody, "    "+line)
			} else {
				goBody = append(goBody, "    "+line)
			}
			continue
		}

		if strings.Contains(line, "<<") {
			line = strings.ReplaceAll(line, "<<", "=")
			goBody = append(goBody, line)
			continue
		}

		if strings.HasPrefix(line, "//") {
			goBody = append(goBody, "    "+line)
		}

		if strings.Contains(line, "(") && strings.Contains(line, ")") {
			if strings.HasPrefix(line, "addToArrayStart") || strings.HasPrefix(line, "addToArrayEnd") {
				parts := strings.SplitN(line, "(", 2)
				args := parts[1][:len(parts[1])-1]
				funcName := parts[0]
				targetVar := strings.Split(args, ",")[0]
				goBody = append(goBody, fmt.Sprintf("    %s = %s(%s)", strings.TrimSpace(targetVar), funcName, args))
			} else {
				goBody = append(goBody, "    "+line)
			}
			continue
		}

	}

	return fmt.Sprintf(
		`package main

		import (
			"sort"
			"fmt"
			"time"
			"bufio"
			"os"
			"strings"
			"crypto/sha256"
			"math"
			"math/rand"
			"encoding/json"
			"strconv"
		)

		%s

		func main() {
			%s
		}
	`, runtime.Runtime(), strings.Join(goBody, "\n")), nil
}

func splitArgs(input string) []string {
	var args []string
	var curr strings.Builder
	inStr := false
	parens := 0

	for i := 0; i < len(input); i++ {
		ch := input[i]

		if ch == '"' {
			inStr = !inStr
		}

		if !inStr {
			if ch == '(' {
				parens++
			}
			if ch == ')' {
				parens--
			}
		}

		if ch == ',' && !inStr && parens == 0 {
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
