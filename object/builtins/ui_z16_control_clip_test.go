package builtins

import (
	"testing"
	"zumbra/object"
)

func TestZ16ControlRenderClipIntersectsScrollViewport(t *testing.T) {
	parent := object.UIRect{X: 10, Y: 40, Width: 300, Height: 120}
	item := object.UIRenderItem{
		Kind:   "button",
		Bounds: object.UIRect{X: 220, Y: 20, Width: 80, Height: 40},
		Clip:   &parent,
	}
	clip, visible := sdlUIItemRenderClip(item)
	if !visible || clip == nil {
		t.Fatal("partially visible control must keep an effective clip")
	}
	if clip.X != 220 || clip.Y != 40 || clip.Width != 80 || clip.Height != 20 {
		t.Fatalf("clip=%+v", *clip)
	}

	item.Bounds = object.UIRect{X: 220, Y: 0, Width: 80, Height: 20}
	if _, visible = sdlUIItemRenderClip(item); visible {
		t.Fatal("fully hidden control must not render")
	}
}

func TestZ16InputsAlwaysUseTheirOwnBoundsAsRenderClip(t *testing.T) {
	item := object.UIRenderItem{Kind: "input", Bounds: object.UIRect{X: 20, Y: 30, Width: 140, Height: 40}}
	clip, visible := sdlUIItemRenderClip(item)
	if !visible || clip == nil || *clip != item.Bounds {
		t.Fatalf("input clip=%+v visible=%v", clip, visible)
	}
}
