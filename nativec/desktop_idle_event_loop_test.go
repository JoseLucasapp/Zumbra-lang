package nativec

import (
	"strings"
	"testing"
)

func TestNativeDesktopLoopBlocksWhileIdle(t *testing.T) {
	data, err := runtimeFiles.ReadFile("runtime/zumbra_desktop.inc")
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	required := []string{
		`ZD_SDL_PushEvent`,
		`SDL_PushEvent`,
		`Z_DESKTOP_WAKE_EVENT`,
		`pthread_equal(pthread_self(), app->run_thread)`,
		`int64_t wait_ms=app->headless?app->poll_interval_ms:-1`,
		`while(app->running&&!app->closed){if(z_ui_any_dirty(app))`,
		`ZDesktopEventNative*pending=z_desktop_dequeue(app)`,
	}
	for _, token := range required {
		if !strings.Contains(source, token) {
			t.Fatalf("native desktop runtime missing %q", token)
		}
	}
	if strings.Contains(source, `z_desktop_next_event(app,app->poll_interval_ms)`) {
		t.Fatal("native desktop loop still polls at pollIntervalMs while idle")
	}
	if strings.Contains(source, `ZDesktopEventNative*pending=z_desktop_next_event(app,0)`) {
		t.Fatal("native desktop loop must not drain SDL through repeated non-blocking polls")
	}
}

func TestNativeUIDirtyStateCanWakeBlockedLoop(t *testing.T) {
	data, err := runtimeFiles.ReadFile("runtime/zumbra_ui.inc")
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	for _, token := range []string{
		`static bool z_ui_any_dirty`,
		`static void z_ui_mark_dirty`,
		`z_desktop_wake_app(ctx->app)`,
	} {
		if !strings.Contains(source, token) {
			t.Fatalf("native UI runtime missing %q", token)
		}
	}
	if strings.Contains(source, `bool was_dirty=ctx->dirty;z_ui_mark_dirty(ctx);`) {
		t.Fatal("z_ui_mark_dirty recursively calls itself")
	}
}
