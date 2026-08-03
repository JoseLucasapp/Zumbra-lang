// Package lsp provides Zumbra's built-in Language Server Protocol implementation.
package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode/utf16"
	"unicode/utf8"

	"zumbra/builtinspec"
	"zumbra/diagnostics"
	"zumbra/tooling/docgen"
	"zumbra/tooling/formatter"
	"zumbra/tooling/lint"
)

const ServerVersion = "0.14.1"

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *rpcError   `json:"error,omitempty"`
}

type rpcError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type document struct {
	URI     string
	Version int
	Text    string
}

type Server struct {
	reader    *bufio.Reader
	writer    io.Writer
	writeMu   sync.Mutex
	documents map[string]document
	shutdown  bool
}

func New(in io.Reader, out io.Writer) *Server {
	return &Server{reader: bufio.NewReader(in), writer: out, documents: map[string]document{}}
}

func Run(in io.Reader, out io.Writer) error {
	return New(in, out).Serve()
}

func (s *Server) Serve() error {
	for {
		payload, err := readMessage(s.reader)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		var message request
		if err := json.Unmarshal(payload, &message); err != nil {
			_ = s.send(response{JSONRPC: "2.0", ID: nil, Error: &rpcError{Code: -32700, Message: "parse error", Data: err.Error()}})
			continue
		}
		exit, err := s.handle(message)
		if err != nil {
			return err
		}
		if exit {
			return nil
		}
	}
}

func (s *Server) handle(message request) (bool, error) {
	if s.shutdown && message.Method != "exit" {
		if len(message.ID) == 0 {
			return false, nil
		}
		return false, s.replyError(message.ID, -32600, "server has shut down", nil)
	}
	switch message.Method {
	case "initialize":
		return false, s.reply(message.ID, map[string]interface{}{
			"serverInfo": map[string]string{"name": "zumbra-lsp", "version": ServerVersion},
			"capabilities": map[string]interface{}{
				"textDocumentSync":           2,
				"documentFormattingProvider": true,
				"documentSymbolProvider":     true,
				"hoverProvider":              true,
				"completionProvider":         map[string]interface{}{"resolveProvider": false, "triggerCharacters": []string{"."}},
			},
		})
	case "initialized":
		return false, nil
	case "shutdown":
		s.shutdown = true
		return false, s.reply(message.ID, nil)
	case "exit":
		return true, nil
	case "textDocument/didOpen":
		var params didOpenParams
		if err := json.Unmarshal(message.Params, &params); err != nil {
			return false, nil
		}
		s.documents[params.TextDocument.URI] = document{URI: params.TextDocument.URI, Version: params.TextDocument.Version, Text: params.TextDocument.Text}
		return false, s.publishDiagnostics(params.TextDocument.URI)
	case "textDocument/didChange":
		var params didChangeParams
		if err := json.Unmarshal(message.Params, &params); err != nil {
			return false, nil
		}
		current := s.documents[params.TextDocument.URI]
		current.URI = params.TextDocument.URI
		current.Version = params.TextDocument.Version
		if len(params.ContentChanges) > 0 {
			// Zumbra advertises full document synchronization.
			current.Text = params.ContentChanges[len(params.ContentChanges)-1].Text
		}
		s.documents[params.TextDocument.URI] = current
		return false, s.publishDiagnostics(params.TextDocument.URI)
	case "textDocument/didSave":
		var params textDocumentParams
		if err := json.Unmarshal(message.Params, &params); err != nil {
			return false, nil
		}
		return false, s.publishDiagnostics(params.TextDocument.URI)
	case "textDocument/didClose":
		var params textDocumentParams
		if err := json.Unmarshal(message.Params, &params); err == nil {
			delete(s.documents, params.TextDocument.URI)
			return false, s.notify("textDocument/publishDiagnostics", map[string]interface{}{"uri": params.TextDocument.URI, "diagnostics": []interface{}{}})
		}
		return false, nil
	case "textDocument/formatting":
		var params formattingParams
		if err := json.Unmarshal(message.Params, &params); err != nil {
			return false, s.replyError(message.ID, -32602, "invalid formatting parameters", err.Error())
		}
		doc, ok := s.documents[params.TextDocument.URI]
		if !ok {
			return false, s.reply(message.ID, []interface{}{})
		}
		width := params.Options.TabSize
		if width <= 0 {
			width = 4
		}
		formatted, err := formatter.Format(uriPath(doc.URI), doc.Text, formatter.Options{IndentWidth: width})
		if err != nil {
			return false, s.replyError(message.ID, -32603, "formatting failed", err.Error())
		}
		if !formatted.Changed {
			return false, s.reply(message.ID, []interface{}{})
		}
		edit := textEdit{
			Range:   lspRange{Start: lspPosition{Line: 0, Character: 0}, End: endPosition(doc.Text)},
			NewText: formatted.Source,
		}
		return false, s.reply(message.ID, []textEdit{edit})
	case "textDocument/documentSymbol":
		var params textDocumentParams
		if err := json.Unmarshal(message.Params, &params); err != nil {
			return false, s.replyError(message.ID, -32602, "invalid document symbol parameters", err.Error())
		}
		doc, ok := s.documents[params.TextDocument.URI]
		if !ok {
			return false, s.reply(message.ID, []interface{}{})
		}
		symbols, err := docgen.Extract(uriPath(doc.URI), doc.Text, docgen.Options{IncludePrivate: true})
		if err != nil {
			return false, s.reply(message.ID, []interface{}{})
		}
		result := make([]documentSymbol, 0, len(symbols))
		for _, symbol := range symbols {
			line := max(0, symbol.Line-1)
			result = append(result, documentSymbol{
				Name:           symbol.Name,
				Detail:         symbol.Signature,
				Kind:           symbolKind(symbol.Kind),
				Range:          lspRange{Start: lspPosition{Line: line, Character: 0}, End: lspPosition{Line: line, Character: utf16Length(symbol.Signature)}},
				SelectionRange: lspRange{Start: lspPosition{Line: line, Character: 0}, End: lspPosition{Line: line, Character: utf16Length(symbol.Name)}},
			})
		}
		return false, s.reply(message.ID, result)
	case "textDocument/hover":
		var params positionParams
		if err := json.Unmarshal(message.Params, &params); err != nil {
			return false, s.replyError(message.ID, -32602, "invalid hover parameters", err.Error())
		}
		doc, ok := s.documents[params.TextDocument.URI]
		if !ok {
			return false, s.reply(message.ID, nil)
		}
		word := wordAt(doc.Text, params.Position)
		if word == "" {
			return false, s.reply(message.ID, nil)
		}
		symbols, _ := docgen.Extract(uriPath(doc.URI), doc.Text, docgen.Options{IncludePrivate: true})
		for _, symbol := range symbols {
			if symbol.Name != word {
				continue
			}
			value := "```zumbra\n" + symbol.Signature + "\n```"
			if symbol.Description != "" {
				value += "\n\n" + symbol.Description
			}
			return false, s.reply(message.ID, map[string]interface{}{"contents": map[string]string{"kind": "markdown", "value": value}})
		}
		if isBuiltin(word) {
			value := "```zumbra\n" + word + "(...)\n```\n\nZumbra standard builtin."
			return false, s.reply(message.ID, map[string]interface{}{"contents": map[string]string{"kind": "markdown", "value": value}})
		}
		return false, s.reply(message.ID, nil)
	case "textDocument/completion":
		items := completionItems()
		return false, s.reply(message.ID, map[string]interface{}{"isIncomplete": false, "items": items})
	default:
		if len(message.ID) == 0 {
			return false, nil
		}
		return false, s.replyError(message.ID, -32601, "method not found", message.Method)
	}
}

func (s *Server) publishDiagnostics(uri string) error {
	doc, ok := s.documents[uri]
	if !ok {
		return nil
	}
	result := lint.Source(uriPath(uri), doc.Text, lint.Options{CheckPipeline: true, RequirePublicDocs: false, MaxLineLength: 120})
	items := make([]lspDiagnostic, 0, len(result.Diagnostics))
	for _, item := range result.Diagnostics {
		items = append(items, toLSPDiagnostic(item, doc.Text))
	}
	return s.notify("textDocument/publishDiagnostics", map[string]interface{}{"uri": uri, "version": doc.Version, "diagnostics": items})
}

func (s *Server) reply(id json.RawMessage, result interface{}) error {
	if result == nil {
		result = json.RawMessage("null")
	}
	return s.send(response{JSONRPC: "2.0", ID: decodeID(id), Result: result})
}

func (s *Server) replyError(id json.RawMessage, code int, message string, data interface{}) error {
	return s.send(response{JSONRPC: "2.0", ID: decodeID(id), Error: &rpcError{Code: code, Message: message, Data: data}})
}

func (s *Server) notify(method string, params interface{}) error {
	payload := map[string]interface{}{"jsonrpc": "2.0", "method": method, "params": params}
	return s.write(payload)
}

func (s *Server) send(value response) error { return s.write(value) }

func (s *Server) write(value interface{}) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if _, err := fmt.Fprintf(s.writer, "Content-Length: %d\r\n\r\n", len(payload)); err != nil {
		return err
	}
	_, err = s.writer.Write(payload)
	return err
}

func readMessage(reader *bufio.Reader) ([]byte, error) {
	contentLength := -1
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		key, value, ok := strings.Cut(line, ":")
		if ok && strings.EqualFold(strings.TrimSpace(key), "Content-Length") {
			parsed, parseErr := strconv.Atoi(strings.TrimSpace(value))
			if parseErr != nil {
				return nil, fmt.Errorf("invalid Content-Length: %w", parseErr)
			}
			contentLength = parsed
		}
	}
	if contentLength < 0 {
		return nil, fmt.Errorf("missing Content-Length")
	}
	payload := make([]byte, contentLength)
	_, err := io.ReadFull(reader, payload)
	return payload, err
}

func decodeID(raw json.RawMessage) interface{} {
	if len(raw) == 0 {
		return nil
	}
	var value interface{}
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil
	}
	return value
}

type textDocumentIdentifier struct {
	URI string `json:"uri"`
}
type textDocumentItem struct {
	URI     string `json:"uri"`
	Version int    `json:"version"`
	Text    string `json:"text"`
}
type versionedTextDocumentIdentifier struct {
	URI     string `json:"uri"`
	Version int    `json:"version"`
}
type didOpenParams struct {
	TextDocument textDocumentItem `json:"textDocument"`
}
type contentChange struct {
	Text string `json:"text"`
}
type didChangeParams struct {
	TextDocument   versionedTextDocumentIdentifier `json:"textDocument"`
	ContentChanges []contentChange                 `json:"contentChanges"`
}
type textDocumentParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
}
type formattingParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
	Options      struct {
		TabSize      int  `json:"tabSize"`
		InsertSpaces bool `json:"insertSpaces"`
	} `json:"options"`
}
type lspPosition struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}
type lspRange struct {
	Start lspPosition `json:"start"`
	End   lspPosition `json:"end"`
}
type textEdit struct {
	Range   lspRange `json:"range"`
	NewText string   `json:"newText"`
}
type positionParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
	Position     lspPosition            `json:"position"`
}
type lspDiagnostic struct {
	Range    lspRange          `json:"range"`
	Severity int               `json:"severity"`
	Code     string            `json:"code"`
	Source   string            `json:"source"`
	Message  string            `json:"message"`
	Tags     []int             `json:"tags,omitempty"`
	Data     map[string]string `json:"data,omitempty"`
}
type documentSymbol struct {
	Name           string   `json:"name"`
	Detail         string   `json:"detail,omitempty"`
	Kind           int      `json:"kind"`
	Range          lspRange `json:"range"`
	SelectionRange lspRange `json:"selectionRange"`
}
type completionItem struct {
	Label      string `json:"label"`
	Kind       int    `json:"kind"`
	Detail     string `json:"detail,omitempty"`
	InsertText string `json:"insertText,omitempty"`
}

func toLSPDiagnostic(item diagnostics.Diagnostic, text string) lspDiagnostic {
	severity := 3
	switch item.Severity {
	case diagnostics.SeverityError:
		severity = 1
	case diagnostics.SeverityWarning:
		severity = 2
	case diagnostics.SeverityHint:
		severity = 4
	}
	startLine := max(0, item.Range.Start.Line-1)
	endLine := max(startLine, item.Range.End.Line-1)
	return lspDiagnostic{
		Range: lspRange{
			Start: lspPosition{Line: startLine, Character: lspCharacter(text, startLine, item.Range.Start.Column)},
			End:   lspPosition{Line: endLine, Character: lspCharacter(text, endLine, item.Range.End.Column)},
		},
		Severity: severity,
		Code:     item.Code,
		Source:   item.Source,
		Message:  item.Message,
		Data:     item.Metadata,
	}
}

func uriPath(uri string) string {
	parsed, err := url.Parse(uri)
	if err != nil || parsed.Scheme == "" {
		return uri
	}
	if parsed.Scheme != "file" {
		return uri
	}
	path, err := url.PathUnescape(parsed.Path)
	if err != nil {
		path = parsed.Path
	}
	if parsed.Host != "" && parsed.Host != "localhost" {
		path = "//" + parsed.Host + path
	}
	if runtime.GOOS == "windows" && len(path) >= 3 && path[0] == '/' && path[2] == ':' {
		path = path[1:]
	}
	return filepath.FromSlash(path)
}

func endPosition(text string) lspPosition {
	lines := normalizedLines(text)
	if len(lines) == 0 {
		return lspPosition{}
	}
	return lspPosition{Line: len(lines) - 1, Character: utf16Length(lines[len(lines)-1])}
}

func wordAt(text string, position lspPosition) string {
	lines := normalizedLines(text)
	if position.Line < 0 || position.Line >= len(lines) || position.Character < 0 {
		return ""
	}
	runes := []rune(lines[position.Line])
	character := runeIndexFromUTF16(lines[position.Line], position.Character)
	if character >= len(runes) {
		character = len(runes) - 1
	}
	if character < 0 {
		return ""
	}
	isWord := func(value rune) bool {
		return value == '_' || value >= '0' && value <= '9' || value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z'
	}
	if !isWord(runes[character]) && character > 0 && isWord(runes[character-1]) {
		character--
	}
	if !isWord(runes[character]) {
		return ""
	}
	start, end := character, character+1
	for start > 0 && isWord(runes[start-1]) {
		start--
	}
	for end < len(runes) && isWord(runes[end]) {
		end++
	}
	return string(runes[start:end])
}

func normalizedLines(text string) []string {
	return strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
}

// lspCharacter converts the compiler's one-based UTF-8 byte column into the
// zero-based UTF-16 code-unit offset required by the Language Server Protocol.
func lspCharacter(text string, zeroBasedLine, oneBasedColumn int) int {
	lines := normalizedLines(text)
	if zeroBasedLine < 0 || zeroBasedLine >= len(lines) {
		return 0
	}
	line := lines[zeroBasedLine]
	byteIndex := oneBasedColumn - 1
	if byteIndex < 0 {
		byteIndex = 0
	}
	if byteIndex > len(line) {
		byteIndex = len(line)
	}
	for byteIndex > 0 && byteIndex < len(line) && !utf8.RuneStart(line[byteIndex]) {
		byteIndex--
	}
	return utf16Length(line[:byteIndex])
}

func utf16Length(value string) int {
	return len(utf16.Encode([]rune(value)))
}

func runeIndexFromUTF16(value string, units int) int {
	if units <= 0 {
		return 0
	}
	used := 0
	index := 0
	for _, value := range value {
		width := 1
		if value > 0xFFFF {
			width = 2
		}
		if used+width > units {
			return index
		}
		used += width
		index++
		if used == units {
			return index
		}
	}
	return index
}

func completionItems() []completionItem {
	labels := map[string]completionItem{}
	for _, keyword := range []string{"fct", "var", "const", "pub", "struct", "enum", "type", "if", "else", "while", "for", "in", "where", "return", "match", "case", "import", "as", "extern", "from", "unsafe", "async", "await", "spawn", "try", "true", "false"} {
		labels[keyword] = completionItem{Label: keyword, Kind: 14, Detail: "Zumbra keyword"}
	}
	for _, name := range builtinspec.Names {
		labels[name] = completionItem{Label: name, Kind: 3, Detail: "Zumbra builtin", InsertText: name}
	}
	result := make([]completionItem, 0, len(labels))
	for _, item := range labels {
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Label < result[j].Label })
	return result
}

func isBuiltin(name string) bool {
	return builtinspec.Contains(name)
}

func symbolKind(kind string) int {
	switch kind {
	case "function":
		return 12
	case "variable":
		return 13
	case "constant":
		return 14
	case "struct":
		return 23
	case "enum":
		return 10
	case "type":
		return 26
	case "extern":
		return 2
	default:
		return 13
	}
}

func max(left, right int) int {
	if left > right {
		return left
	}
	return right
}
