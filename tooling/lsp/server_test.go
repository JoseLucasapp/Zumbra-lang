package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
)

func frame(value interface{}) string {
	payload, _ := json.Marshal(value)
	return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(payload), payload)
}

func readFrames(t *testing.T, output string) []map[string]interface{} {
	t.Helper()
	reader := bufio.NewReader(strings.NewReader(output))
	result := []map[string]interface{}{}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		lengthText := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
		length, err := strconv.Atoi(lengthText)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := reader.ReadString('\n'); err != nil {
			t.Fatal(err)
		}
		payload := make([]byte, length)
		if _, err := reader.Read(payload); err != nil {
			t.Fatal(err)
		}
		var item map[string]interface{}
		if err := json.Unmarshal(payload, &item); err != nil {
			t.Fatal(err)
		}
		result = append(result, item)
	}
	return result
}

func TestInitializeOpenFormatAndShutdown(t *testing.T) {
	uri := "file:///tmp/main.zum"
	input := frame(map[string]interface{}{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]interface{}{}}) +
		frame(map[string]interface{}{"jsonrpc": "2.0", "method": "textDocument/didOpen", "params": map[string]interface{}{"textDocument": map[string]interface{}{"uri": uri, "version": 1, "text": "fct main(){show(1);}\nmain();"}}}) +
		frame(map[string]interface{}{"jsonrpc": "2.0", "id": 2, "method": "textDocument/formatting", "params": map[string]interface{}{"textDocument": map[string]interface{}{"uri": uri}, "options": map[string]interface{}{"tabSize": 4, "insertSpaces": true}}}) +
		frame(map[string]interface{}{"jsonrpc": "2.0", "id": 3, "method": "shutdown", "params": map[string]interface{}{}}) +
		frame(map[string]interface{}{"jsonrpc": "2.0", "method": "exit"})
	var output bytes.Buffer
	if err := Run(strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	frames := readFrames(t, output.String())
	if len(frames) < 4 {
		t.Fatalf("expected initialize, diagnostics, formatting and shutdown frames: %#v", frames)
	}
	foundFormatting := false
	for _, item := range frames {
		if item["id"] == float64(2) {
			foundFormatting = true
			edits, ok := item["result"].([]interface{})
			if !ok || len(edits) != 1 {
				t.Fatalf("unexpected formatting response: %#v", item)
			}
		}
	}
	if !foundFormatting {
		t.Fatal("missing formatting response")
	}
}

func TestUTF16Positions(t *testing.T) {
	text := "😀show(value);"
	if got := endPosition(text); got.Character != 14 {
		t.Fatalf("end position uses %d UTF-16 units, want 14", got.Character)
	}
	if got := wordAt(text, lspPosition{Line: 0, Character: 2}); got != "show" {
		t.Fatalf("wordAt after surrogate pair = %q, want show", got)
	}
	if got := lspCharacter("éx", 0, 3); got != 1 {
		t.Fatalf("UTF-8 byte column conversion = %d, want 1", got)
	}
}

func TestRejectRequestAfterShutdown(t *testing.T) {
	input := frame(map[string]interface{}{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]interface{}{}}) +
		frame(map[string]interface{}{"jsonrpc": "2.0", "id": 2, "method": "shutdown", "params": map[string]interface{}{}}) +
		frame(map[string]interface{}{"jsonrpc": "2.0", "id": 3, "method": "textDocument/completion", "params": map[string]interface{}{}}) +
		frame(map[string]interface{}{"jsonrpc": "2.0", "method": "exit"})
	var output bytes.Buffer
	if err := Run(strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	frames := readFrames(t, output.String())
	for _, item := range frames {
		if item["id"] == float64(3) {
			rpcError, ok := item["error"].(map[string]interface{})
			if !ok || rpcError["code"] != float64(-32600) {
				t.Fatalf("unexpected post-shutdown response: %#v", item)
			}
			return
		}
	}
	t.Fatal("missing post-shutdown error response")
}
