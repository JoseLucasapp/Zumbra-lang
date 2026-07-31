package nativec

import (
	"strings"
	"testing"
)

func TestNativeDesktopPollSentinelEndsDrainCycle(t *testing.T) {
	data, err := runtimeFiles.ReadFile("runtime/zumbra_desktop.inc")
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	required := []string{
		`#define Z_DESKTOP_POLL_SENTINEL UINT32_C(0x7F00)`,
		`common->type==Z_DESKTOP_POLL_SENTINEL`,
		`type="__zumbra_poll_sentinel"`,
		`strcmp(event->type,"__zumbra_poll_sentinel")==0`,
		`if(timeout_ms==0)return NULL`,
		`drained<256`,
		`ZDesktopEventNative*pending=z_desktop_dequeue(app)`,
	}
	for _, token := range required {
		if !strings.Contains(source, token) {
			t.Fatalf("native desktop runtime missing %q", token)
		}
	}
	if strings.Contains(source, `if(timeout_ms==0)continue;`) {
		t.Fatal("non-blocking native event drain can still spin after an internal event")
	}
}

func TestNativeDesktopSentinelIsNotDispatchedPublicly(t *testing.T) {
	data, err := runtimeFiles.ReadFile("runtime/zumbra_desktop.inc")
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	sentinelBranch := `if(strcmp(event->type,"__zumbra_poll_sentinel")==0){if(timeout_ms==0)return NULL;z_desktop_sleep_ms(1);continue;}`
	wakeBranch := `if(strcmp(event->type,"__zumbra_wake")!=0)return event;`
	if !strings.Contains(source, sentinelBranch) || !strings.Contains(source, wakeBranch) {
		t.Fatal("internal SDL events are not filtered before public event dispatch")
	}
	if strings.Index(source, sentinelBranch) > strings.Index(source, wakeBranch) {
		t.Fatal("poll sentinel must be filtered before the public-event return path")
	}
	if strings.Contains(source, `ZDesktopEventNative*pending=z_desktop_next_event(app,0)`) {
		t.Fatal("desktopRun must not drain SDL through repeated non-blocking polls")
	}
}
