package nativec

import (
	"strings"
	"testing"
)

func TestNativeUIUsesPersistentRenderCaches(t *testing.T) {
	data, err := runtimeFiles.ReadFile("runtime/zumbra_ui.inc")
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	for _, token := range []string{
		`ZUITextTextureCacheNative`,
		`ZUIImageTextureCacheNative`,
		`ZUIBackdropCacheNative`,
		`z_ui_text_cache_find`,
		`z_ui_image_cache_find`,
		`z_ui_backdrop_cache_find`,
		`z_ui_release_renderer_resources`,
		`z_ui_cache_clear_renderer`,
	} {
		if !strings.Contains(source, token) {
			t.Fatalf("native UI runtime missing %q", token)
		}
	}
	if strings.Contains(source, `z_desktop_sdl.DestroyTexture(texture);\n                return;`) {
		t.Fatal("native text renderer still destroys every generated texture immediately")
	}
}

func TestNativeUIMouseMotionDoesNotRepaintSameHoverTarget(t *testing.T) {
	data, err := runtimeFiles.ReadFile("runtime/zumbra_ui.inc")
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	for _, token := range []string{
		`static bool z_ui_set_hover`,
		`bool hover_changed=z_ui_set_hover(ctx,target->id)`,
		`if(hover_changed)z_ui_mark_dirty(ctx);return;`,
	} {
		if !strings.Contains(source, token) {
			t.Fatalf("native UI runtime missing %q", token)
		}
	}
	if strings.Contains(source, `ctx->hover_id=z_strdup(target->id);`) {
		t.Fatal("native UI still reallocates hover identity for every mouse-motion event")
	}
}

func TestNativeUIBackdropBlurIsCachedPerModal(t *testing.T) {
	data, err := runtimeFiles.ReadFile("runtime/zumbra_ui.inc")
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	for _, token := range []string{
		`z_ui_blur_backdrop(SDL_Renderer*r,int width,int height,int radius,const void*owner)`,
		`z_ui_backdrop_cache_find(r,owner,width,height,radius)`,
		`z_ui_blur_backdrop(r,(int)viewport.width,(int)viewport.height,blur,n)`,
	} {
		if !strings.Contains(source, token) {
			t.Fatalf("native UI backdrop cache missing %q", token)
		}
	}
}
