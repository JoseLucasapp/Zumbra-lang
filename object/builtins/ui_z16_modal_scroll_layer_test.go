package builtins

import (
	"testing"
	"zumbra/object"
)

func TestZ16ModalSeparatesBackgroundScrollbarLayer(t *testing.T) {
	items := []object.UIRenderItem{
		{ID: "root", Kind: "container"},
		{ID: "sidebar", Kind: "column", ScrollContentHeight: 900, ContentBounds: object.UIRect{Height: 600}},
		{ID: "dialog", Kind: "modal"},
		{ID: "dialog-content", Kind: "column", ScrollContentHeight: 420, ContentBounds: object.UIRect{Height: 300}},
	}
	if got := sdlUIFirstModalIndex(items); got != 2 {
		t.Fatalf("expected modal layer to start at index 2, got %d", got)
	}
	if got := sdlUIFirstModalIndex(items[:2]); got != 2 {
		t.Fatalf("expected frames without a modal to use the full item range, got %d", got)
	}
}
