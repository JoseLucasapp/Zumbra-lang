package builtins

import (
	"testing"

	"zumbra/object"
)

func TestZ16FinalSelectOpensDropdownBeforeChangingValue(t *testing.T) {
	app := DesktopAppBuiltin().Fn(desktopTestDict(map[string]object.Object{"backend": NewString("headless")})).(*object.DesktopApp)
	window := DesktopWindowBuiltin().Fn(app, desktopTestDict(map[string]object.Object{"title": NewString("Select"), "width": NewInteger(360), "height": NewInteger(240)})).(*object.DesktopWindow)
	state := UIStateBuiltin().Fn(NewString("Primeira")).(*object.UIState)
	selectNode := uiTestNode("select", map[string]object.Object{
		"id": NewString("picker"), "value": NewString("Primeira"),
		"options": &object.Array{Elements: []object.Object{NewString("Primeira"), NewString("Segunda"), NewString("Terceira")}},
		"width":   NewInteger(220), "height": NewInteger(40), "optionHeight": NewInteger(36), "maxVisibleOptions": NewInteger(3),
	})
	UIBindBuiltin().Fn(selectNode, NewString("value"), state)
	root := uiTestNode("column", map[string]object.Object{"padding": NewInteger(12)}, selectNode)
	ctx := UIMountBuiltin().Fn(app, window, root, desktopTestDict(map[string]object.Object{})).(*object.UIContext)

	bounds := selectNode.Bounds
	x, y := bounds.X+10, bounds.Y+10
	UIDispatchBuiltin().Fn(ctx, desktopTestDict(map[string]object.Object{"type": NewString("mouse_down"), "x": NewFloat(x), "y": NewFloat(y)}))
	UIDispatchBuiltin().Fn(ctx, desktopTestDict(map[string]object.Object{"type": NewString("mouse_up"), "x": NewFloat(x), "y": NewFloat(y)}))
	if got := UIGetBuiltin().Fn(selectNode, NewString("open")).Inspect(); got != "true" {
		t.Fatalf("select should be open, got %s", got)
	}
	if got := UIStateGetBuiltin().Fn(state).Inspect(); got != "Primeira" {
		t.Fatalf("opening dropdown changed value to %q", got)
	}

	popup, rowHeight, _ := uiSelectPopupGeometry(selectNode, root.Bounds)
	optionY := popup.Y + rowHeight + 8
	UIDispatchBuiltin().Fn(ctx, desktopTestDict(map[string]object.Object{"type": NewString("mouse_down"), "x": NewFloat(popup.X + 10), "y": NewFloat(optionY)}))
	UIDispatchBuiltin().Fn(ctx, desktopTestDict(map[string]object.Object{"type": NewString("mouse_up"), "x": NewFloat(popup.X + 10), "y": NewFloat(optionY)}))
	if got := UIStateGetBuiltin().Fn(state).Inspect(); got != "Segunda" {
		t.Fatalf("selected value=%q", got)
	}
	if got := UIGetBuiltin().Fn(selectNode, NewString("open")).Inspect(); got != "false" {
		t.Fatalf("select should close after choosing an option, got %s", got)
	}
}

func TestZ16FinalSelectClosesWhenClickingOutside(t *testing.T) {
	app := DesktopAppBuiltin().Fn(desktopTestDict(map[string]object.Object{"backend": NewString("headless")})).(*object.DesktopApp)
	window := DesktopWindowBuiltin().Fn(app, desktopTestDict(map[string]object.Object{"title": NewString("Select"), "width": NewInteger(420), "height": NewInteger(240)})).(*object.DesktopWindow)
	selectNode := uiTestNode("select", map[string]object.Object{
		"id": NewString("picker"), "value": NewString("A"),
		"options": &object.Array{Elements: []object.Object{NewString("A"), NewString("B")}},
		"width":   NewInteger(180), "height": NewInteger(40),
	})
	outside := uiTestNode("button", map[string]object.Object{"id": NewString("outside"), "text": NewString("Outside"), "width": NewInteger(120), "height": NewInteger(40)})
	root := uiTestNode("row", map[string]object.Object{"padding": NewInteger(12), "gap": NewInteger(20)}, selectNode, outside)
	ctx := UIMountBuiltin().Fn(app, window, root, desktopTestDict(map[string]object.Object{})).(*object.UIContext)

	UIDispatchBuiltin().Fn(ctx, desktopTestDict(map[string]object.Object{"type": NewString("mouse_down"), "x": NewFloat(selectNode.Bounds.X + 10), "y": NewFloat(selectNode.Bounds.Y + 10)}))
	UIDispatchBuiltin().Fn(ctx, desktopTestDict(map[string]object.Object{"type": NewString("mouse_up"), "x": NewFloat(selectNode.Bounds.X + 10), "y": NewFloat(selectNode.Bounds.Y + 10)}))
	UIDispatchBuiltin().Fn(ctx, desktopTestDict(map[string]object.Object{"type": NewString("mouse_down"), "x": NewFloat(outside.Bounds.X + 10), "y": NewFloat(outside.Bounds.Y + 10)}))
	if got := UIGetBuiltin().Fn(selectNode, NewString("open")).Inspect(); got != "false" {
		t.Fatalf("outside click did not close select: %s", got)
	}
}

func TestZ16FinalModalWidthHeightAreCenteredAndChildrenAreClipped(t *testing.T) {
	app := DesktopAppBuiltin().Fn(desktopTestDict(map[string]object.Object{"backend": NewString("headless")})).(*object.DesktopApp)
	window := DesktopWindowBuiltin().Fn(app, desktopTestDict(map[string]object.Object{"title": NewString("Modal"), "width": NewInteger(1200), "height": NewInteger(800)})).(*object.DesktopWindow)
	child := uiTestNode("container", map[string]object.Object{"id": NewString("content"), "height": NewInteger(900)})
	modal := uiTestNode("modal", map[string]object.Object{
		"id": NewString("dialog"), "width": NewInteger(620), "height": NewInteger(520),
		"padding": NewInteger(20), "overflow": NewString("hidden"),
	}, child)
	root := uiTestNode("container", map[string]object.Object{"grow": NewInteger(1)}, modal)
	ctx := UIMountBuiltin().Fn(app, window, root, desktopTestDict(map[string]object.Object{})).(*object.UIContext)

	if modal.Bounds.X != 290 || modal.Bounds.Y != 140 || modal.Bounds.Width != 620 || modal.Bounds.Height != 520 {
		t.Fatalf("modal bounds=%+v", modal.Bounds)
	}
	var childItem *object.UIRenderItem
	for index := range ctx.LastFrame.Items {
		if ctx.LastFrame.Items[index].ID == "content" {
			childItem = &ctx.LastFrame.Items[index]
			break
		}
	}
	if childItem == nil || childItem.Clip == nil {
		t.Fatal("modal child should be clipped to the modal content bounds")
	}
	if *childItem.Clip != modal.ContentBounds {
		t.Fatalf("child clip=%+v content=%+v", *childItem.Clip, modal.ContentBounds)
	}
}
