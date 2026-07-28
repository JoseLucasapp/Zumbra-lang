package builtins

import (
	"testing"

	"zumbra/object"
)

func uiTestNode(kind string, props map[string]object.Object, children ...*object.UINode) *object.UINode {
	values := make([]object.Object, len(children))
	for i, child := range children {
		values[i] = child
	}
	return uiNewNode(kind, props, children)
}

func TestZ14HeadlessLayoutStateCanvasAndAccessibility(t *testing.T) {
	app := DesktopAppBuiltin().Fn(desktopTestDict(map[string]object.Object{"backend": NewString("headless")})).(*object.DesktopApp)
	window := DesktopWindowBuiltin().Fn(app, desktopTestDict(map[string]object.Object{"title": NewString("UI"), "width": NewInteger(640), "height": NewInteger(480)})).(*object.DesktopWindow)
	label := uiTestNode("text", map[string]object.Object{"id": NewString("status")})
	button := uiTestNode("button", map[string]object.Object{"text": NewString("Run")})
	canvas := uiTestNode("canvas", map[string]object.Object{"height": NewInteger(80)})
	root := uiTestNode("column", map[string]object.Object{"padding": NewInteger(12), "gap": NewInteger(8)}, label, button, canvas)
	state := UIStateBuiltin().Fn(NewString("ready")).(*object.UIState)
	if result := UIBindBuiltin().Fn(label, NewString("text"), state); result.Type() == object.ERROR_OBJ {
		t.Fatal(result.Inspect())
	}
	contextValue := UIMountBuiltin().Fn(app, window, root, desktopTestDict(map[string]object.Object{"theme": UIThemeBuiltin().Fn(NewString("light"))}))
	context, ok := contextValue.(*object.UIContext)
	if !ok {
		t.Fatalf("expected UIContext, got %T: %s", contextValue, contextValue.Inspect())
	}
	UIStateSetBuiltin().Fn(state, NewString("updated"))
	if got := UIGetBuiltin().Fn(label, NewString("text")).Inspect(); got != "updated" {
		t.Fatalf("binding=%s", got)
	}
	UICanvasCommandBuiltin().Fn(canvas, NewString("fillRect"), desktopTestDict(map[string]object.Object{"x": NewInteger(2), "y": NewInteger(2), "width": NewInteger(20), "height": NewInteger(10)}))
	snapshot := UISnapshotBuiltin().Fn(context).(*object.Dict)
	items := objectDictValue(snapshot, "items").(*object.Array)
	if len(items.Elements) != 4 {
		t.Fatalf("items=%d", len(items.Elements))
	}
	accessibility := UIAccessibilityBuiltin().Fn(context).(*object.Array)
	if len(accessibility.Elements) != 4 {
		t.Fatalf("accessibility=%d", len(accessibility.Elements))
	}
	if renderer, ok := window.Runtime.(object.DesktopUIRenderer); !ok || renderer.LastUIFrame() == nil {
		t.Fatal("headless renderer did not retain a frame")
	}
}

func TestZ14HeadlessButtonDispatchAndFocus(t *testing.T) {
	app := DesktopAppBuiltin().Fn(desktopTestDict(map[string]object.Object{"backend": NewString("headless")})).(*object.DesktopApp)
	window := DesktopWindowBuiltin().Fn(app, desktopTestDict(map[string]object.Object{"title": NewString("UI"), "width": NewInteger(320), "height": NewInteger(200)})).(*object.DesktopWindow)
	clicked := false
	handler := &object.Builtin{Fn: func(args ...object.Object) object.Object { clicked = true; return &object.Null{} }}
	button := uiTestNode("button", map[string]object.Object{"id": NewString("run"), "text": NewString("Run"), "onClick": handler})
	root := uiTestNode("column", map[string]object.Object{"padding": NewInteger(8)}, button)
	ctx := UIMountBuiltin().Fn(app, window, root, desktopTestDict(map[string]object.Object{})).(*object.UIContext)
	UIDispatchBuiltin().Fn(ctx, desktopTestDict(map[string]object.Object{"type": NewString("mouse_down"), "x": NewInteger(12), "y": NewInteger(12)}))
	UIDispatchBuiltin().Fn(ctx, desktopTestDict(map[string]object.Object{"type": NewString("mouse_up"), "x": NewInteger(12), "y": NewInteger(12)}))
	if !clicked {
		t.Fatal("button callback was not invoked")
	}
	if ctx.FocusID != "run" {
		t.Fatalf("focus=%q", ctx.FocusID)
	}
}

func TestZ14BidirectionalBindingAndKeyboardEditing(t *testing.T) {
	app := DesktopAppBuiltin().Fn(desktopTestDict(map[string]object.Object{"backend": NewString("headless")})).(*object.DesktopApp)
	window := DesktopWindowBuiltin().Fn(app, desktopTestDict(map[string]object.Object{"title": NewString("UI"), "width": NewInteger(320), "height": NewInteger(180)})).(*object.DesktopWindow)
	state := UIStateBuiltin().Fn(NewString("A")).(*object.UIState)
	input := uiTestNode("input", map[string]object.Object{"id": NewString("name"), "value": NewString("")})
	root := uiTestNode("column", map[string]object.Object{"padding": NewInteger(8)}, input)
	UIBindBuiltin().Fn(input, NewString("value"), state)
	ctx := UIMountBuiltin().Fn(app, window, root, desktopTestDict(map[string]object.Object{})).(*object.UIContext)
	UIDispatchBuiltin().Fn(ctx, desktopTestDict(map[string]object.Object{"type": NewString("mouse_down"), "x": NewInteger(12), "y": NewInteger(12)}))
	UIDispatchBuiltin().Fn(ctx, desktopTestDict(map[string]object.Object{"type": NewString("text_input"), "text": NewString("B")}))
	if got := UIStateGetBuiltin().Fn(state).Inspect(); got != "AB" {
		t.Fatalf("two-way binding=%q", got)
	}
	UIDispatchBuiltin().Fn(ctx, desktopTestDict(map[string]object.Object{"type": NewString("key_down"), "key": NewString("Backspace")}))
	if got := UIStateGetBuiltin().Fn(state).Inspect(); got != "A" {
		t.Fatalf("backspace binding=%q", got)
	}
}

func TestZ14ModalUsesCenteredContentBounds(t *testing.T) {
	app := DesktopAppBuiltin().Fn(desktopTestDict(map[string]object.Object{"backend": NewString("headless")})).(*object.DesktopApp)
	window := DesktopWindowBuiltin().Fn(app, desktopTestDict(map[string]object.Object{"title": NewString("UI"), "width": NewInteger(800), "height": NewInteger(600)})).(*object.DesktopWindow)
	modal := uiTestNode("modal", map[string]object.Object{"contentWidth": NewInteger(400), "contentHeight": NewInteger(300)}, uiTestNode("text", map[string]object.Object{"text": NewString("Dialog")}))
	ctx := UIMountBuiltin().Fn(app, window, modal, desktopTestDict(map[string]object.Object{})).(*object.UIContext)
	frame := ctx.LastFrame
	if frame == nil || len(frame.Items) == 0 {
		t.Fatal("missing modal frame")
	}
	bounds := frame.Items[0].Bounds
	if bounds.X != 200 || bounds.Y != 150 || bounds.Width != 400 || bounds.Height != 300 {
		t.Fatalf("modal bounds=%+v", bounds)
	}
}
