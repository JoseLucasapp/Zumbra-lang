package builtins

import (
	"testing"
	"zumbra/object"
)

func TestZ16NavigationPlacementAndCollapsedSize(t *testing.T) {
	expanded := uiTestNode("column", map[string]object.Object{"visible": NewBoolean(false), "grow": NewInteger(1)}, uiTestNode("text", map[string]object.Object{"text": NewString("hidden")}))
	collapsed := uiTestNode("column", map[string]object.Object{"visible": NewBoolean(true), "grow": NewInteger(1)}, uiTestNode("button", map[string]object.Object{"text": NewString("open")}))
	navigation := uiTestNode("menu", map[string]object.Object{
		"placement": NewString("left"), "expandedSize": NewInteger(280), "collapsedSize": NewInteger(56), "collapsed": NewBoolean(true),
	}, expanded, collapsed)
	content := uiTestNode("container", map[string]object.Object{"grow": NewInteger(1)})
	root := uiTestNode("row", map[string]object.Object{"grow": NewInteger(1)}, navigation, content)
	layoutUINode(root, object.UIRect{Width: 900, Height: 600}, &object.UITheme{Name: "light", Values: defaultUITheme("light")}, nil)
	if navigation.Bounds.Width != 56 {
		t.Fatalf("collapsed navigation width=%v", navigation.Bounds.Width)
	}
	if collapsed.Bounds.Height <= 0 {
		t.Fatalf("visible collapsed content was not laid out: %+v", collapsed.Bounds)
	}
	if expanded.Bounds.Width != 0 || expanded.Bounds.Height != 0 {
		t.Fatalf("hidden navigation content received layout: %+v", expanded.Bounds)
	}
	navigation.Props["collapsed"] = NewBoolean(false)
	layoutUINode(root, object.UIRect{Width: 900, Height: 600}, &object.UITheme{Name: "light", Values: defaultUITheme("light")}, nil)
	if navigation.Bounds.Width != 280 {
		t.Fatalf("expanded navigation width=%v", navigation.Bounds.Width)
	}
}

func TestZ16ButtonAndSelectTextDefaults(t *testing.T) {
	theme := &object.UITheme{Name: "light", Values: defaultUITheme("light")}
	button := map[string]object.Object{}
	applyUIStyleDefaults("button", button, theme)
	if got := optionString(button, "textAlign", ""); got != "center" {
		t.Fatalf("button text alignment=%q", got)
	}
	if got := optionString(button, "textOverflow", ""); got != "ellipsis" {
		t.Fatalf("button overflow=%q", got)
	}
	selectProps := map[string]object.Object{}
	applyUIStyleDefaults("select", selectProps, theme)
	if got := optionString(selectProps, "textAlign", ""); got != "left" {
		t.Fatalf("select text alignment=%q", got)
	}
}

func TestZ16PortableChartCommands(t *testing.T) {
	chart := uiTestNode("canvas", map[string]object.Object{"id": NewString("chart")})
	for _, kind := range []string{"pieChart", "barChart", "lineChart"} {
		result := UICanvasCommandBuiltin().Fn(chart, NewString(kind), desktopTestDict(map[string]object.Object{"values": &object.Array{Elements: []object.Object{NewInteger(1), NewInteger(2)}}}))
		if result.Type() == object.ERROR_OBJ {
			t.Fatalf("%s command failed: %s", kind, result.Inspect())
		}
	}
	commands := parseCanvasCommands(chart.Props["commands"])
	if len(commands) != 3 || commands[0].Kind != "pieChart" || commands[2].Kind != "lineChart" {
		t.Fatalf("chart commands=%+v", commands)
	}
}
