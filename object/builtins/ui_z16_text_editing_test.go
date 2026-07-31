package builtins

import (
	"testing"
	"zumbra/object"
)

func TestZ16InputCaretSelectionClipboardAndReplacement(t *testing.T) {
	backend := newHeadlessDesktopBackend()
	app := &object.DesktopApp{Backend: backend, Windows: map[int64]*object.DesktopWindow{}, Handlers: map[string][]object.Object{}, Shortcuts: map[string]object.Object{}, UIContexts: map[int64]*object.UIContext{}}
	input := uiTestNode("input", map[string]object.Object{"id": NewString("editor"), "value": NewString("alpha beta"), "width": NewInteger(260), "height": NewInteger(40)})
	root := uiTestNode("container", map[string]object.Object{"padding": NewInteger(10)}, input)
	ctx := &object.UIContext{App: app, Root: root, Theme: &object.UITheme{Name: "light", Values: defaultUITheme("light")}, Nodes: map[string]*object.UINode{"editor": input}, FocusID: "editor"}
	send := func(kind, key, shortcut, text string) {
		data := map[string]object.Object{}
		if key != "" {
			data["key"] = NewString(key)
		}
		if shortcut != "" {
			data["shortcut"] = NewString(shortcut)
		}
		if text != "" {
			data["text"] = NewString(text)
		}
		if e := dispatchUIEvent(ctx, &object.DesktopEvent{Type: kind, Data: data}); e != nil {
			t.Fatal(e)
		}
	}
	send("key_down", "A", "ctrl+a", "")
	send("key_down", "C", "ctrl+c", "")
	clip, _ := backend.Clipboard()
	if clip != "alpha beta" {
		t.Fatalf("clipboard=%q", clip)
	}
	send("text_input", "", "", "gamma")
	if got := optionString(input.Props, "value", ""); got != "gamma" {
		t.Fatalf("value=%q", got)
	}
	send("key_down", "Left", "left", "")
	send("key_down", "Left", "shift+left", "")
	send("text_input", "", "", "X")
	if got := optionString(input.Props, "value", ""); got != "gamXa" {
		t.Fatalf("value after selection=%q", got)
	}
	send("key_down", "Home", "home", "")
	send("key_down", "Delete", "delete", "")
	if got := optionString(input.Props, "value", ""); got != "amXa" {
		t.Fatalf("delete value=%q", got)
	}
}

func TestZ16MouseCaretAndScrollbarGutter(t *testing.T) {
	theme := &object.UITheme{Name: "light", Values: defaultUITheme("light")}
	input := uiTestNode("input", map[string]object.Object{"id": NewString("mouse-editor"), "value": NewString("alpha beta"), "width": NewInteger(220), "height": NewInteger(40)})
	scroll := uiTestNode("column", map[string]object.Object{
		"width": NewInteger(240), "height": NewInteger(80), "overflowY": NewString("auto"),
		"scrollbarOverlay": NewBoolean(true), "scrollbarAvoidContent": NewBoolean(true),
		"scrollbarWidth": NewInteger(8), "scrollbarGutter": NewInteger(4),
	}, input, uiTestNode("spacer", map[string]object.Object{"size": NewInteger(100)}))
	layoutUINode(scroll, object.UIRect{Width: 240, Height: 80}, theme, nil)
	if scroll.ContentBounds.Width != 228 {
		t.Fatalf("reserved overlay content width=%v, want 228", scroll.ContentBounds.Width)
	}
	app := &object.DesktopApp{Backend: newHeadlessDesktopBackend(), Windows: map[int64]*object.DesktopWindow{}, Handlers: map[string][]object.Object{}, Shortcuts: map[string]object.Object{}, UIContexts: map[int64]*object.UIContext{}}
	ctx := &object.UIContext{App: app, Root: scroll, Theme: theme, Nodes: map[string]*object.UINode{"mouse-editor": input}, FocusID: "mouse-editor"}
	x := input.Bounds.X + 9
	y := input.Bounds.Y + input.Bounds.Height/2
	if e := dispatchUIEvent(ctx, &object.DesktopEvent{Type: "mouse_down", Data: map[string]object.Object{"x": NewFloat(x), "y": NewFloat(y)}}); e != nil {
		t.Fatal(e)
	}
	if got := optionInt(input.Props, "caretIndex", -1); got > 1 {
		t.Fatalf("mouse caret=%d, want beginning of text", got)
	}
}
