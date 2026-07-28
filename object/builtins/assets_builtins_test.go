package builtins

import (
	"testing"

	"zumbra/object"
)

func TestEmbeddedAssetsBuiltins(t *testing.T) {
	defer ResetEmbeddedAssets()
	if err := ConfigureEmbeddedAssets([]EmbeddedAsset{{Name: "assets/hello.txt", Data: []byte("olá")}, {Name: "assets/raw.bin", Data: []byte{0xff, 0x00}}}); err != nil {
		t.Fatal(err)
	}
	if got := AssetExistsBuiltin().Fn(NewString("./assets/hello.txt")); got.Inspect() != "true" {
		t.Fatalf("expected true, got %s", got.Inspect())
	}
	if got := AssetTextBuiltin().Fn(NewString("assets/hello.txt")); got.Inspect() != "olá" {
		t.Fatalf("unexpected text: %s", got.Inspect())
	}
	bytes, ok := AssetBytesBuiltin().Fn(NewString("assets/raw.bin")).(*object.ByteArray)
	if !ok || len(bytes.Data) != 2 || bytes.Data[0] != 0xff {
		t.Fatalf("unexpected bytes: %#v", bytes)
	}
	list, ok := AssetListBuiltin().Fn().(*object.Array)
	if !ok || len(list.Elements) != 2 || list.Elements[0].Inspect() != "assets/hello.txt" {
		t.Fatalf("unexpected list: %#v", list)
	}
	if _, ok := AssetTextBuiltin().Fn(NewString("assets/raw.bin")).(*object.Error); !ok {
		t.Fatal("expected invalid UTF-8 error")
	}
}

func TestConfigureEmbeddedAssetsRejectsDuplicates(t *testing.T) {
	defer ResetEmbeddedAssets()
	if err := ConfigureEmbeddedAssets([]EmbeddedAsset{{Name: "a.txt"}, {Name: "./a.txt"}}); err == nil {
		t.Fatal("expected duplicate error")
	}
}
