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

func TestZ14PointOneThemeSwitchReRendersDarkPalette(t *testing.T) {
	app := DesktopAppBuiltin().Fn(desktopTestDict(map[string]object.Object{"backend": NewString("headless")})).(*object.DesktopApp)
	window := DesktopWindowBuiltin().Fn(app, desktopTestDict(map[string]object.Object{"title": NewString("Theme"), "width": NewInteger(360), "height": NewInteger(220)})).(*object.DesktopWindow)
	label := uiTestNode("text", map[string]object.Object{"text": NewString("Ação e café")})
	root := uiTestNode("column", map[string]object.Object{"padding": NewInteger(12)}, label)
	ctx := UIMountBuiltin().Fn(app, window, root, desktopTestDict(map[string]object.Object{"theme": UIThemeBuiltin().Fn(NewString("light"))})).(*object.UIContext)
	if got := ctx.LastFrame.Background; got != "#f5f7fb" {
		t.Fatalf("light background=%q", got)
	}
	result := UISetThemeBuiltin().Fn(ctx, UIThemeBuiltin().Fn(NewString("dark")))
	if result.Type() == object.ERROR_OBJ {
		t.Fatal(result.Inspect())
	}
	if got := ctx.LastFrame.Background; got != "#141822" {
		t.Fatalf("dark background=%q", got)
	}
	if len(ctx.LastFrame.Items) == 0 {
		t.Fatal("missing rendered items")
	}
	if got := uiString(ctx.LastFrame.Items[0].Props, "textColor", ""); got != "#f2f5fa" {
		t.Fatalf("dark text color=%q", got)
	}
}

func TestZ14PointOneUTF8TextMeasurementUsesRunesAndFontSize(t *testing.T) {
	style := object.UITextStyle{FontFamily: "sans", FontSize: 18, LineHeight: 1.25}
	small := approximateUITextMetrics("ação", style)
	if small.Width <= 0 || small.Height < 18 {
		t.Fatalf("small metrics=%+v", small)
	}
	large := approximateUITextMetrics("ação", object.UITextStyle{FontFamily: "sans", FontSize: 28, LineHeight: 1.25})
	if large.Width <= small.Width || large.Height <= small.Height {
		t.Fatalf("font size did not affect metrics: small=%+v large=%+v", small, large)
	}
	ascii := approximateUITextMetrics("acao", style)
	if small.Width != ascii.Width {
		t.Fatalf("UTF-8 code points were measured as bytes: utf8=%+v ascii=%+v", small, ascii)
	}
}

func TestZ16UIEventIncludesTargetIdentity(t *testing.T) {
	app := DesktopAppBuiltin().Fn(desktopTestDict(map[string]object.Object{"backend": NewString("headless")})).(*object.DesktopApp)
	window := DesktopWindowBuiltin().Fn(app, desktopTestDict(map[string]object.Object{"title": NewString("UI"), "width": NewInteger(320), "height": NewInteger(200)})).(*object.DesktopWindow)
	var targetID, targetKind string
	handler := &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("handler expects one event")
		}
		event, ok := args[0].(*object.Dict)
		if !ok {
			return NewError("handler expects event dictionary")
		}
		targetID = objectDictValue(event, "targetId").Inspect()
		targetKind = objectDictValue(event, "targetKind").Inspect()
		return &object.Null{}
	}}
	button := uiTestNode("button", map[string]object.Object{"id": NewString("edit-42"), "text": NewString("Edit"), "onClick": handler})
	root := uiTestNode("column", map[string]object.Object{"padding": NewInteger(8)}, button)
	ctx := UIMountBuiltin().Fn(app, window, root, desktopTestDict(map[string]object.Object{})).(*object.UIContext)
	UIDispatchBuiltin().Fn(ctx, desktopTestDict(map[string]object.Object{"type": NewString("mouse_down"), "x": NewInteger(12), "y": NewInteger(12)}))
	UIDispatchBuiltin().Fn(ctx, desktopTestDict(map[string]object.Object{"type": NewString("mouse_up"), "x": NewInteger(12), "y": NewInteger(12)}))
	if targetID != "edit-42" || targetKind != "button" {
		t.Fatalf("target identity=%q/%q", targetID, targetKind)
	}
}

func TestZ16PointOneTextInputLifecycleFollowsFocus(t *testing.T) {
	app := DesktopAppBuiltin().Fn(desktopTestDict(map[string]object.Object{"backend": NewString("headless")})).(*object.DesktopApp)
	window := DesktopWindowBuiltin().Fn(app, desktopTestDict(map[string]object.Object{"title": NewString("UI"), "width": NewInteger(360), "height": NewInteger(220)})).(*object.DesktopWindow)
	input := uiTestNode("input", map[string]object.Object{"id": NewString("name"), "value": NewString("")})
	button := uiTestNode("button", map[string]object.Object{"id": NewString("save"), "text": NewString("Save")})
	root := uiTestNode("column", map[string]object.Object{"padding": NewInteger(8), "gap": NewInteger(8)}, input, button)
	ctx := UIMountBuiltin().Fn(app, window, root, desktopTestDict(map[string]object.Object{})).(*object.UIContext)
	runtime := window.Runtime.(*headlessWindow)

	if runtime.textInput {
		t.Fatal("text input started before an editable control received focus")
	}
	if result := UIFocusBuiltin().Fn(ctx, input); result.Type() == object.ERROR_OBJ {
		t.Fatal(result.Inspect())
	}
	if !runtime.textInput {
		t.Fatal("text input did not start when the input received focus")
	}
	if result := UIFocusBuiltin().Fn(ctx, button); result.Type() == object.ERROR_OBJ {
		t.Fatal(result.Inspect())
	}
	if runtime.textInput {
		t.Fatal("text input remained enabled after focus moved to a button")
	}
	if result := UIFocusBuiltin().Fn(ctx, input); result.Type() == object.ERROR_OBJ {
		t.Fatal(result.Inspect())
	}
	if result := UIUnmountBuiltin().Fn(ctx); result.Type() == object.ERROR_OBJ {
		t.Fatal(result.Inspect())
	}
	if runtime.textInput {
		t.Fatal("text input remained enabled after UI unmount")
	}
}

func TestZ16PointTwoRowsAndColumnsDoNotAddImplicitPadding(t *testing.T) {
	app := DesktopAppBuiltin().Fn(desktopTestDict(map[string]object.Object{"backend": NewString("headless")})).(*object.DesktopApp)
	window := DesktopWindowBuiltin().Fn(app, desktopTestDict(map[string]object.Object{"title": NewString("Layout"), "width": NewInteger(300), "height": NewInteger(160)})).(*object.DesktopWindow)
	input := uiTestNode("input", map[string]object.Object{"id": NewString("aligned"), "width": NewInteger(180), "height": NewInteger(40)})
	root := uiTestNode("column", map[string]object.Object{"gap": NewInteger(8)}, input)
	ctx := UIMountBuiltin().Fn(app, window, root, desktopTestDict(map[string]object.Object{})).(*object.UIContext)
	if got := ctx.LastFrame.Items[1].Bounds.X; got != 0 {
		t.Fatalf("implicit horizontal padding=%v", got)
	}
	if got := ctx.LastFrame.Items[1].Bounds.Y; got != 0 {
		t.Fatalf("implicit vertical padding=%v", got)
	}
}

func TestZ16PointTwoVerticalOverflowScrollsAndClipsChildren(t *testing.T) {
	app := DesktopAppBuiltin().Fn(desktopTestDict(map[string]object.Object{"backend": NewString("headless")})).(*object.DesktopApp)
	window := DesktopWindowBuiltin().Fn(app, desktopTestDict(map[string]object.Object{"title": NewString("Scroll"), "width": NewInteger(320), "height": NewInteger(180)})).(*object.DesktopWindow)
	first := uiTestNode("container", map[string]object.Object{"id": NewString("first"), "height": NewInteger(80), "minHeight": NewInteger(80)})
	second := uiTestNode("container", map[string]object.Object{"id": NewString("second"), "height": NewInteger(80), "minHeight": NewInteger(80)})
	third := uiTestNode("container", map[string]object.Object{"id": NewString("third"), "height": NewInteger(80), "minHeight": NewInteger(80)})
	root := uiTestNode("column", map[string]object.Object{
		"height": NewInteger(120), "maxHeight": NewInteger(120), "gap": NewInteger(8),
		"overflowY": NewString("auto"), "scrollStep": NewInteger(40),
	}, first, second, third)
	ctx := UIMountBuiltin().Fn(app, window, root, desktopTestDict(map[string]object.Object{})).(*object.UIContext)
	if root.ContentHeight != 256 {
		t.Fatalf("content height=%v", root.ContentHeight)
	}
	if root.ScrollOffsetY != 0 {
		t.Fatalf("initial scroll offset=%v", root.ScrollOffsetY)
	}
	UIDispatchBuiltin().Fn(ctx, desktopTestDict(map[string]object.Object{
		"type": NewString("mouse_wheel"), "x": NewInteger(20), "y": NewInteger(20), "dy": NewInteger(-1),
	}))
	if root.ScrollOffsetY != 40 {
		t.Fatalf("scroll offset=%v", root.ScrollOffsetY)
	}
	if got := first.Bounds.Y; got != -40 {
		t.Fatalf("first row y=%v", got)
	}
	if len(ctx.LastFrame.Items) < 2 || ctx.LastFrame.Items[1].Clip == nil {
		t.Fatal("scroll descendants were not clipped")
	}
	clip := *ctx.LastFrame.Items[1].Clip
	if clip.Y != 0 || clip.Height != 120 {
		t.Fatalf("clip=%+v", clip)
	}
	for i := 0; i < 10; i++ {
		UIDispatchBuiltin().Fn(ctx, desktopTestDict(map[string]object.Object{
			"type": NewString("mouse_wheel"), "x": NewInteger(20), "y": NewInteger(20), "dy": NewInteger(-1),
		}))
	}
	if root.ScrollOffsetY != 136 {
		t.Fatalf("clamped scroll offset=%v", root.ScrollOffsetY)
	}
}
