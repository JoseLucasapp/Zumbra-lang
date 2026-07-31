package nativec

import (
	"strings"
	"testing"
)

func TestNativeUIControlLocalClipsRespectScrollViewport(t *testing.T) {
	data, err := runtimeFiles.ReadFile("runtime/zumbra_ui.inc")
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	for _, token := range []string{
		`static bool z_ui_set_local_clip`,
		`has_parent&&!z_ui_rect_intersection(parent,local,&effective)`,
		`z_ui_set_local_clip(r,has_clip,clip,local_clip)`,
	} {
		if !strings.Contains(source, token) {
			t.Fatalf("native UI runtime missing %q", token)
		}
	}
	if strings.Contains(source, `z_ui_set_clip(r,true,(ZUIRectNative){b.x+1,b.y+1`) {
		t.Fatal("button renderer still replaces the inherited scroll clip")
	}
}
