package builtins

import (
	"os"
	"testing"

	"zumbra/object"
)

func TestDesktopMediaHeadlessContract(t *testing.T) {
	app := DesktopAppBuiltin().Fn(desktopTestDict(map[string]object.Object{"backend": NewString("headless")})).(*object.DesktopApp)
	window := DesktopWindowBuiltin().Fn(app, desktopTestDict(map[string]object.Object{"title": NewString("Media"), "width": NewInteger(4), "height": NewInteger(2)})).(*object.DesktopWindow)
	pixels := &object.ByteArray{Data: make([]byte, 4*2*4)}
	if !DesktopWindowPresentRGBABuiltin().Fn(window, pixels, NewInteger(4), NewInteger(2)).(*object.Boolean).Value {
		t.Fatal("headless RGBA presentation failed")
	}
	if got := DesktopWindowSetVSyncBuiltin().Fn(window, NewBoolean(true)); got != window {
		t.Fatalf("set VSync returned %T", got)
	}
	if DesktopKeyDownBuiltin().Fn(app, NewInteger(29)).(*object.Boolean).Value {
		t.Fatal("headless keyboard must be released")
	}
	if DesktopGamepadButtonBuiltin().Fn(app, NewInteger(1), NewInteger(0)).(*object.Boolean).Value {
		t.Fatal("headless gamepad must be released")
	}
	samples := &object.ByteArray{Data: []byte{0, 16, 32, 63}}
	if got := DesktopAudioQueueBuiltin().Fn(app, samples, NewInteger(80), NewBoolean(false)).(*object.Integer).Value; got != 4 {
		t.Fatalf("queued samples=%d", got)
	}
	if got := DesktopAudioQueuedBuiltin().Fn(app).(*object.Integer).Value; got != 8 {
		t.Fatalf("queued bytes=%d", got)
	}
}

func TestProcessArgsAndUnixTimeBuiltins(t *testing.T) {
	previous := os.Args
	os.Args = []string{"zumbra", "game.nes"}
	defer func() { os.Args = previous }()
	values := ProcessArgsBuiltin().Fn().(*object.Array)
	if len(values.Elements) != 2 || values.Elements[1].(*object.String).Value != "game.nes" {
		t.Fatalf("args=%s", values.Inspect())
	}
	timestamp := UnixTimeSecondsBuiltin().Fn().(*object.FixedInteger)
	if timestamp.Kind != object.FixedU64 || timestamp.Raw == 0 {
		t.Fatalf("timestamp=%s", timestamp.Inspect())
	}
}
