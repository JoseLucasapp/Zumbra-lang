package builtins

import (
	"testing"

	"zumbra/object"
)

func desktopTestDict(values map[string]object.Object) *object.Dict {
	return objectMapDict(values).(*object.Dict)
}

func TestDesktopHeadlessLifecycleAndWindow(t *testing.T) {
	appValue := DesktopAppBuiltin().Fn(desktopTestDict(map[string]object.Object{"backend": NewString("headless")}))
	app, ok := appValue.(*object.DesktopApp)
	if !ok {
		t.Fatalf("expected DesktopApp, got %T: %s", appValue, appValue.Inspect())
	}
	windowValue := DesktopWindowBuiltin().Fn(app, desktopTestDict(map[string]object.Object{
		"title": NewString("Test"), "width": NewInteger(640), "height": NewInteger(480), "scale": NewFloat(2),
	}))
	window, ok := windowValue.(*object.DesktopWindow)
	if !ok {
		t.Fatalf("expected DesktopWindow, got %T: %s", windowValue, windowValue.Inspect())
	}
	if got := DesktopBackendBuiltin().Fn(app).Inspect(); got != "headless" {
		t.Fatalf("backend=%s", got)
	}
	if got := DesktopWindowTitleBuiltin().Fn(window).Inspect(); got != "Test" {
		t.Fatalf("title=%s", got)
	}
	DesktopWindowSetSizeBuiltin().Fn(window, NewInteger(800), NewInteger(600))
	size := DesktopWindowSizeBuiltin().Fn(window).(*object.Dict)
	if objectDictValue(size, "width").Inspect() != "800" || objectDictValue(size, "height").Inspect() != "600" {
		t.Fatalf("unexpected size %s", size.Inspect())
	}
	pixels := DesktopWindowPixelSizeBuiltin().Fn(window).(*object.Dict)
	if objectDictValue(pixels, "width").Inspect() != "1600" {
		t.Fatalf("unexpected pixel size %s", pixels.Inspect())
	}
	if result := DesktopSetClipboardBuiltin().Fn(app, NewString("copied")); result.Type() == object.ERROR_OBJ {
		t.Fatal(result.Inspect())
	}
	if got := DesktopClipboardBuiltin().Fn(app).Inspect(); got != "copied" {
		t.Fatalf("clipboard=%s", got)
	}
	if result := DesktopCloseBuiltin().Fn(app); result.Inspect() != "true" {
		t.Fatalf("close=%s", result.Inspect())
	}
}

func TestDesktopHeadlessEventsAndShortcuts(t *testing.T) {
	app := DesktopAppBuiltin().Fn(desktopTestDict(map[string]object.Object{"backend": NewString("headless")})).(*object.DesktopApp)
	count := int64(0)
	handler := &object.Builtin{Fn: func(args ...object.Object) object.Object { count++; return &object.Null{} }}
	DesktopOnBuiltin().Fn(app, NewString("custom"), handler)
	DesktopShortcutBuiltin().Fn(app, NewString("Control + Shift + K"), handler)
	DesktopEmitBuiltin().Fn(app, desktopTestDict(map[string]object.Object{"type": NewString("custom")}))
	DesktopEmitBuiltin().Fn(app, desktopTestDict(map[string]object.Object{
		"type": NewString("key_down"), "data": desktopTestDict(map[string]object.Object{"shortcut": NewString("ctrl+shift+k")}),
	}))
	for i := 0; i < 5 && count < 2; i++ {
		DesktopPollBuiltin().Fn(app, NewInteger(0))
	}
	if count != 2 {
		t.Fatalf("expected 2 callbacks, got %d", count)
	}
}

func TestDesktopHeadlessPickerNotificationAndPaths(t *testing.T) {
	app := DesktopAppBuiltin().Fn(desktopTestDict(map[string]object.Object{"backend": NewString("headless")})).(*object.DesktopApp)
	picked := DesktopPickFileBuiltin().Fn(app, desktopTestDict(map[string]object.Object{"defaultPath": NewString("/tmp/example.txt")}))
	if got := picked.(*object.Array).Elements[0].Inspect(); got != "/tmp/example.txt" {
		t.Fatalf("picked=%s", got)
	}
	if result := DesktopNotifyBuiltin().Fn(app, desktopTestDict(map[string]object.Object{"title": NewString("Title"), "body": NewString("Body")})); result.Inspect() != "true" {
		t.Fatalf("notify=%s", result.Inspect())
	}
	paths := DesktopPathsBuiltin().Fn(app).(*object.Dict)
	for _, name := range []string{"home", "cache", "config", "data", "cwd", "temp"} {
		if objectDictValue(paths, name) == nil {
			t.Fatalf("missing path %s", name)
		}
	}
}

func TestDesktopProcessReturnsExitCode(t *testing.T) {
	options := desktopTestDict(map[string]object.Object{"args": &object.Array{Elements: []object.Object{NewString("-c"), NewString("exit 7")}}})
	processValue := DesktopSpawnBuiltin().Fn(NewString("/bin/sh"), options)
	process, ok := processValue.(*object.DesktopProcess)
	if !ok {
		t.Fatalf("expected process, got %T: %s", processValue, processValue.Inspect())
	}
	if got := DesktopProcessWaitBuiltin().Fn(process).Inspect(); got != "7" {
		t.Fatalf("exit=%s", got)
	}
}

func objectDictValue(dict *object.Dict, name string) object.Object {
	key := &object.String{Value: name}
	if pair, ok := dict.Pairs[key.DictKey()]; ok {
		return pair.Value
	}
	return nil
}
