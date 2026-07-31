package builtins

import (
	"testing"
	"zumbra/object"
)

func TestZ16SideNavigationFillsAvailableHeight(t *testing.T) {
	theme := &object.UITheme{Name: "dark", Values: defaultUITheme("dark")}
	title := uiTestNode("text", map[string]object.Object{"text": NewString("Cadastrar jogo")})
	input := uiTestNode("input", map[string]object.Object{"height": NewInteger(40), "value": NewString("Final Fantasy IX")})
	content := uiTestNode("column", map[string]object.Object{"grow": NewInteger(1)}, title, input)
	sidebar := uiTestNode("menu", map[string]object.Object{
		"placement":    NewString("left"),
		"expandedSize": NewInteger(350),
		"padding":      NewInteger(18),
	}, content)

	layoutUINode(sidebar, object.UIRect{Width: 350, Height: 600}, theme, nil)
	if sidebar.Bounds.Height != 600 {
		t.Fatalf("sidebar height=%v, want 600", sidebar.Bounds.Height)
	}
	if content.Bounds.Height <= 0 || title.Bounds.Height <= 0 || input.Bounds.Height != 40 {
		t.Fatalf("sidebar content compressed: content=%+v title=%+v input=%+v", content.Bounds, title.Bounds, input.Bounds)
	}
}
