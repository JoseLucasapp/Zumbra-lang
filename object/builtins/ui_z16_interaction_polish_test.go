package builtins

import (
	"testing"
	"zumbra/object"
)

func TestZ16OverlayScrollbarPreservesSymmetricContentWidth(t *testing.T) {
	theme := &object.UITheme{Name: "light", Values: defaultUITheme("light")}
	children := []*object.UINode{
		uiTestNode("container", map[string]object.Object{"height": NewInteger(80)}),
		uiTestNode("container", map[string]object.Object{"height": NewInteger(80)}),
	}
	overlay := uiTestNode("column", map[string]object.Object{
		"overflowY": NewString("auto"), "scrollbarOverlay": NewBoolean(true),
		"scrollbarWidth": NewInteger(8), "scrollbarGutter": NewInteger(4),
	}, children...)
	layoutUINode(overlay, object.UIRect{Width: 240, Height: 100}, theme, nil)
	if overlay.ContentBounds.Width != 240 {
		t.Fatalf("overlay scrollbar changed content width: %v", overlay.ContentBounds.Width)
	}

	reserved := uiTestNode("column", map[string]object.Object{
		"overflowY": NewString("auto"), "scrollbarOverlay": NewBoolean(false),
		"scrollbarWidth": NewInteger(8), "scrollbarGutter": NewInteger(4),
	},
		uiTestNode("container", map[string]object.Object{"height": NewInteger(80)}),
		uiTestNode("container", map[string]object.Object{"height": NewInteger(80)}),
	)
	layoutUINode(reserved, object.UIRect{Width: 240, Height: 100}, theme, nil)
	if reserved.ContentBounds.Width != 228 {
		t.Fatalf("reserved scrollbar width=%v, want 228", reserved.ContentBounds.Width)
	}
}

func TestZ16HiddenNavigationReflowsSibling(t *testing.T) {
	theme := &object.UITheme{Name: "light", Values: defaultUITheme("light")}
	sidebar := uiTestNode("menu", map[string]object.Object{
		"placement": NewString("left"), "expandedSize": NewInteger(350),
	}, uiTestNode("text", map[string]object.Object{"text": NewString("Menu")}))
	content := uiTestNode("container", map[string]object.Object{"grow": NewInteger(1)})
	root := uiTestNode("row", map[string]object.Object{"gap": NewInteger(18), "grow": NewInteger(1)}, sidebar, content)
	layoutUINode(root, object.UIRect{Width: 1000, Height: 600}, theme, nil)
	if content.Bounds.X != 368 || content.Bounds.Width != 632 {
		t.Fatalf("expanded layout content=%+v", content.Bounds)
	}

	sidebar.Mu.Lock()
	sidebar.Visible = false
	sidebar.Props["visible"] = NewBoolean(false)
	sidebar.Mu.Unlock()
	root.Mu.Lock()
	root.Props["gap"] = NewInteger(0)
	root.Mu.Unlock()
	layoutUINode(root, object.UIRect{Width: 1000, Height: 600}, theme, nil)
	if sidebar.Bounds.Width != 0 || sidebar.Bounds.Height != 0 {
		t.Fatalf("hidden navigation retained bounds: %+v", sidebar.Bounds)
	}
	if content.Bounds.X != 0 || content.Bounds.Width != 1000 {
		t.Fatalf("hidden navigation did not reflow content: %+v", content.Bounds)
	}
}

func TestZ16InteractiveCursorAndCaretDefaults(t *testing.T) {
	theme := &object.UITheme{Name: "dark", Values: defaultUITheme("dark")}
	button := map[string]object.Object{}
	applyUIStyleDefaults("button", button, theme)
	if got := optionString(button, "cursor", ""); got != "pointer" {
		t.Fatalf("button cursor=%q", got)
	}
	input := map[string]object.Object{}
	applyUIStyleDefaults("input", input, theme)
	if got := optionString(input, "cursor", ""); got != "text" {
		t.Fatalf("input cursor=%q", got)
	}
	if optionString(input, "caretColor", "") == "" {
		t.Fatal("input caret color was not resolved from the theme")
	}
	if optionString(input, "focusBackground", "") == "" {
		t.Fatal("input focus background was not resolved from the theme")
	}
}
