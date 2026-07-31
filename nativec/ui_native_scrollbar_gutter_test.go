package nativec

import (
	"strings"
	"testing"
)

func TestNativeUIScrollbarUsesInheritedClipForReservedGutter(t *testing.T) {
	data, err := runtimeFiles.ReadFile("runtime/zumbra_ui.inc")
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	comment := "A scrollbar with scrollbarAvoidContent=true lives in the reserved gutter"
	start := strings.Index(source, comment)
	if start < 0 {
		t.Fatal("native reserved-gutter scrollbar fix was not found")
	}
	section := source[start:]
	end := strings.Index(section, "static ZUINodeNative*z_ui_topmost_modal")
	if end >= 0 {
		section = section[:end]
	}
	clipIndex := strings.Index(section, "z_ui_set_clip(r,has_clip,clip);")
	scrollbarIndex := strings.Index(section, "if(z_ui_scrollable_y(n)")
	if clipIndex < 0 || scrollbarIndex < 0 || clipIndex > scrollbarIndex {
		t.Fatal("native renderer must restore the inherited clip before painting the scrollbar gutter")
	}
	if strings.Contains(section[:scrollbarIndex], "z_ui_set_clip(r,child_has_clip,child_clip);") {
		t.Fatal("reserved-gutter scrollbar is still painted with the child content clip")
	}
	for _, token := range []string{
		`z_ui_prop_bool(n->props,"scrollbarAvoidContent",false)`,
		`track_x=n->content_bounds.x+n->content_bounds.width+fmax(2,gutter)`,
		`z_ui_prop_string(n->props,"scrollbarThumb","#9aa7ba")`,
	} {
		if !strings.Contains(section, token) {
			t.Fatalf("native scrollbar gutter fix missing %q", token)
		}
	}
}
