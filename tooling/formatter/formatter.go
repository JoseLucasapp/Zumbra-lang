// Package formatter implements the canonical, comment-preserving Zumbra source formatter.
package formatter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"zumbra/lexer"
	"zumbra/parser"
)

type Options struct {
	IndentWidth int
}

type Result struct {
	Source  string
	Changed bool
}

type lexemeKind uint8

const (
	lexemeWord lexemeKind = iota
	lexemeString
	lexemeOperator
	lexemePunctuation
	lexemeComment
)

type lexeme struct {
	kind            lexemeKind
	text            string
	line            int
	leadingNewlines int
}

func Format(filename, source string, options Options) (Result, error) {
	if options.IndentWidth <= 0 {
		options.IndentWidth = 4
	}
	if err := validate(source); err != nil {
		return Result{}, fmt.Errorf("cannot format invalid source %s: %w", filename, err)
	}
	tokens, err := scan(source)
	if err != nil {
		return Result{}, fmt.Errorf("scan %s: %w", filename, err)
	}
	output := render(tokens, options)
	if err := validate(output); err != nil {
		return Result{}, fmt.Errorf("formatter produced invalid source for %s: %w", filename, err)
	}
	canonicalInput := strings.ReplaceAll(source, "\r\n", "\n")
	return Result{Source: output, Changed: canonicalInput != output}, nil
}

func File(path string, write bool, options Options) (Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Result{}, fmt.Errorf("read %s: %w", path, err)
	}
	result, err := Format(path, string(data), options)
	if err != nil {
		return Result{}, err
	}
	if write && result.Changed {
		info, statErr := os.Stat(path)
		mode := os.FileMode(0o644)
		if statErr == nil {
			mode = info.Mode().Perm()
		}
		temporary := path + ".zumbra-fmt"
		if err := os.WriteFile(temporary, []byte(result.Source), mode); err != nil {
			return Result{}, fmt.Errorf("write temporary formatted file: %w", err)
		}
		if err := os.Rename(temporary, path); err != nil {
			_ = os.Remove(temporary)
			return Result{}, fmt.Errorf("replace %s: %w", path, err)
		}
	}
	return result, nil
}

func Discover(paths []string) ([]string, error) {
	if len(paths) == 0 {
		paths = []string{"."}
	}
	seen := map[string]bool{}
	files := []string{}
	for _, input := range paths {
		info, err := os.Stat(input)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			if strings.EqualFold(filepath.Ext(input), ".zum") {
				absolute, _ := filepath.Abs(input)
				if !seen[absolute] {
					seen[absolute] = true
					files = append(files, input)
				}
			}
			continue
		}
		err = filepath.WalkDir(input, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				name := entry.Name()
				if path != input && (strings.HasPrefix(name, ".") || name == "build" || name == "dist" || name == "vendor" || name == "node_modules") {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.EqualFold(filepath.Ext(path), ".zum") {
				return nil
			}
			absolute, _ := filepath.Abs(path)
			if !seen[absolute] {
				seen[absolute] = true
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return files, nil
}

func validate(source string) error {
	p := parser.New(lexer.New(source))
	_ = p.ParseProgram()
	if errors := p.Errors(); len(errors) > 0 {
		limit := len(errors)
		if limit > 3 {
			limit = 3
		}
		return fmt.Errorf("%s", strings.Join(errors[:limit], "; "))
	}
	return nil
}

func scan(source string) ([]lexeme, error) {
	result := []lexeme{}
	line := 1
	pendingNewlines := 0
	for index := 0; index < len(source); {
		ch := source[index]
		if ch == ' ' || ch == '\t' || ch == '\r' {
			index++
			continue
		}
		if ch == '\n' {
			pendingNewlines++
			line++
			index++
			continue
		}
		startLine := line
		appendToken := func(kind lexemeKind, text string) {
			result = append(result, lexeme{kind: kind, text: text, line: startLine, leadingNewlines: pendingNewlines})
			pendingNewlines = 0
		}
		if ch == '/' && index+1 < len(source) && source[index+1] == '/' {
			end := index + 2
			for end < len(source) && source[end] != '\n' {
				end++
			}
			appendToken(lexemeComment, strings.TrimRight(source[index:end], " \t\r"))
			index = end
			continue
		}
		if ch == '"' {
			end := index + 1
			escaped := false
			for end < len(source) {
				current := source[end]
				if current == '\n' {
					line++
				}
				if current == '"' && !escaped {
					end++
					break
				}
				if current == '\\' && !escaped {
					escaped = true
				} else {
					escaped = false
				}
				end++
			}
			if end > len(source) || source[end-1] != '"' {
				return nil, fmt.Errorf("unterminated string at line %d", startLine)
			}
			appendToken(lexemeString, source[index:end])
			index = end
			continue
		}
		if isIdentifierStart(rune(ch)) {
			end := index + 1
			for end < len(source) && isIdentifierPart(rune(source[end])) {
				end++
			}
			appendToken(lexemeWord, source[index:end])
			index = end
			continue
		}
		if ch >= '0' && ch <= '9' {
			end := index + 1
			seenDecimal := false
			for end < len(source) {
				current := source[end]
				if isIdentifierPart(rune(current)) {
					end++
					continue
				}
				if current == '.' && !seenDecimal && end+1 < len(source) && source[end+1] != '.' && source[end+1] >= '0' && source[end+1] <= '9' {
					seenDecimal = true
					end++
					continue
				}
				break
			}
			appendToken(lexemeWord, source[index:end])
			index = end
			continue
		}
		matched := ""
		for _, operator := range []string{"<<", "==", "!=", "<=", ">=", "**", "++", "--", "..", "->"} {
			if strings.HasPrefix(source[index:], operator) {
				matched = operator
				break
			}
		}
		if matched != "" {
			appendToken(lexemeOperator, matched)
			index += len(matched)
			continue
		}
		text := string(ch)
		kind := lexemeOperator
		if strings.ContainsRune("(){}[],:;.", rune(ch)) {
			kind = lexemePunctuation
		}
		appendToken(kind, text)
		index++
	}
	return result, nil
}

func isIdentifierStart(ch rune) bool { return ch == '_' || unicode.IsLetter(ch) }
func isIdentifierPart(ch rune) bool  { return isIdentifierStart(ch) || unicode.IsDigit(ch) }

type writer struct {
	out         strings.Builder
	indent      int
	indentWidth int
	lineStart   bool
	last        string
	lineWords   []string
}

func newWriter(indentWidth int) *writer {
	return &writer{indentWidth: indentWidth, lineStart: true}
}

func (w *writer) write(text string, spaceBefore bool) {
	if text == "" {
		return
	}
	if w.lineStart {
		w.out.WriteString(strings.Repeat(" ", w.indent*w.indentWidth))
		w.lineStart = false
	} else if spaceBefore && !strings.HasSuffix(w.out.String(), " ") && !strings.HasSuffix(w.out.String(), "\n") {
		w.out.WriteByte(' ')
	}
	w.out.WriteString(text)
	w.last = text
	if isWordText(text) {
		w.lineWords = append(w.lineWords, text)
	}
}

func (w *writer) trimSpaces() {
	text := w.out.String()
	trimmed := strings.TrimRight(text, " \t")
	if len(trimmed) == len(text) {
		return
	}
	w.out.Reset()
	w.out.WriteString(trimmed)
}

func (w *writer) newline() {
	w.trimSpaces()
	text := w.out.String()
	if text == "" || strings.HasSuffix(text, "\n") {
		w.lineStart = true
		w.last = ""
		w.lineWords = nil
		return
	}
	w.out.WriteByte('\n')
	w.lineStart = true
	w.last = ""
	w.lineWords = nil
}

func (w *writer) blankLine() {
	w.newline()
	if !strings.HasSuffix(w.out.String(), "\n\n") {
		w.out.WriteByte('\n')
	}
	w.lineStart = true
}

func (w *writer) result() string {
	w.trimSpaces()
	return strings.TrimRight(w.out.String(), "\n") + "\n"
}

func render(tokens []lexeme, options Options) string {
	w := newWriter(options.IndentWidth)
	parenDepth := 0
	bracketDepth := 0
	prefixExpected := true
	noSpaceBeforeNext := false

	for index, item := range tokens {
		next := lexeme{}
		if index+1 < len(tokens) {
			next = tokens[index+1]
		}
		if item.leadingNewlines > 1 && parenDepth == 0 && bracketDepth == 0 && !w.lineStart {
			w.blankLine()
		}
		text := item.text
		if item.kind == lexemeComment {
			if !w.lineStart {
				w.write(text, true)
			} else {
				w.write(text, false)
			}
			w.newline()
			prefixExpected = true
			continue
		}

		switch text {
		case "{":
			w.write("{", !w.lineStart && w.last != "(" && w.last != "[" && w.last != ".")
			w.newline()
			w.indent++
			prefixExpected = true
		case "}":
			if !w.lineStart {
				w.newline()
			}
			if w.indent > 0 {
				w.indent--
			}
			w.write("}", false)
			if next.text == "else" {
				// The following token supplies the separating space.
			} else if next.text != ";" && next.text != "," && next.text != ")" && next.text != "]" && next.text != "." && next.text != "(" && next.kind != lexemeComment {
				w.newline()
			}
			prefixExpected = false
		case ";":
			w.write(";", false)
			trailingComment := next.kind == lexemeComment && next.leadingNewlines == 0
			if parenDepth == 0 && bracketDepth == 0 && next.text != "}" && !trailingComment {
				w.newline()
			}
			prefixExpected = true
		case ",":
			w.write(",", false)
			if next.text != "}" && next.text != ")" && next.text != "]" {
				w.write("", true)
			}
			prefixExpected = true
		case ":":
			w.write(":", false)
			prefixExpected = true
		case ".":
			w.write(".", false)
			prefixExpected = true
		case "(":
			keywordSpace := isControlKeyword(w.last)
			w.write("(", keywordSpace)
			parenDepth++
			prefixExpected = true
		case ")":
			w.write(")", false)
			if parenDepth > 0 {
				parenDepth--
			}
			prefixExpected = false
		case "[":
			w.write("[", false)
			bracketDepth++
			prefixExpected = true
		case "]":
			w.write("]", false)
			if bracketDepth > 0 {
				bracketDepth--
			}
			prefixExpected = false
		default:
			if isOperator(text) {
				if (text == "!" || text == "bnot" || text == "-" || text == "+") && prefixExpected {
					w.write(text, !w.lineStart && text == "bnot")
					noSpaceBeforeNext = true
				} else if text == "++" || text == "--" {
					w.write(text, false)
				} else {
					w.write(text, !w.lineStart)
				}
				prefixExpected = true
				continue
			}
			space := needsSpaceBefore(w.last, text)
			if noSpaceBeforeNext {
				space = false
				noSpaceBeforeNext = false
			}
			w.write(text, space)
			prefixExpected = text == "return" || text == "case" || text == "<<" || text == "in" || text == "where"
		}
	}
	return w.result()
}

func needsSpaceBefore(previous, current string) bool {
	if previous == "" {
		return false
	}
	if previous == "." || previous == "(" || previous == "[" || previous == "bnot" || previous == "!" {
		return false
	}
	if current == "else" || current == "case" {
		return true
	}
	return true
}

func isControlKeyword(value string) bool {
	switch value {
	case "if", "while", "for", "match", "where":
		return true
	default:
		return false
	}
}

func isOperator(value string) bool {
	switch value {
	case "<<", "==", "!=", "+", "-", "!", "*", "/", "%", "<", ">", "<=", ">=", "**", "++", "--", "->", "..", "and", "or", "band", "bor", "bxor", "bnot", "shl", "shr":
		return true
	default:
		return false
	}
}

func isWordText(value string) bool {
	if value == "" {
		return false
	}
	first := rune(value[0])
	return isIdentifierStart(first) || unicode.IsDigit(first) || value[0] == '"'
}
