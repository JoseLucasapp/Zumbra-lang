package appmanifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndCollectAssets(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "src", "main.zum"), []byte("show(1);"))
	mustWrite(t, filepath.Join(root, "assets", "hello.txt"), []byte("hello"))
	mustWrite(t, filepath.Join(root, "assets", "skip.tmp"), []byte("skip"))
	mustWrite(t, filepath.Join(root, DefaultManifestName), []byte(`
[app]
name = "Example App"
version = "1.2.3"
identifier = "dev.zumbra.example"
entry = "src/main.zum"

[build]
output = "dist/example"
compiler = "auto"
release = true

[assets]
include = ["assets/**"]
exclude = ["**/*.tmp"]
max_file_bytes = 1024
max_total_bytes = 4096
`))
	manifest, err := Load(filepath.Join(root, DefaultManifestName))
	if err != nil {
		t.Fatal(err)
	}
	assets, err := manifest.CollectAssets()
	if err != nil {
		t.Fatal(err)
	}
	if len(assets) != 1 || assets[0].Name != "assets/hello.txt" || string(assets[0].Data) != "hello" {
		t.Fatalf("unexpected assets: %#v", assets)
	}
	if manifest.OutputPath() != filepath.Join(root, "dist", "example") {
		t.Fatalf("unexpected output: %s", manifest.OutputPath())
	}
}

func TestRejectsEscapingEntry(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, DefaultManifestName), []byte(`[app]
name = "Bad"
version = "1.0.0"
identifier = "dev.zumbra.bad"
entry = "../main.zum"
`))
	if _, err := Load(filepath.Join(root, DefaultManifestName)); err == nil {
		t.Fatal("expected traversal error")
	}
}

func TestRejectsOversizedAsset(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "main.zum"), []byte("show(1);"))
	mustWrite(t, filepath.Join(root, "large.bin"), []byte("12345"))
	mustWrite(t, filepath.Join(root, DefaultManifestName), []byte(`[app]
name = "Limit"
version = "1.0.0"
identifier = "dev.zumbra.limit"
entry = "main.zum"
[assets]
include = ["*.bin"]
max_file_bytes = 4
max_total_bytes = 10
`))
	manifest, err := Load(filepath.Join(root, DefaultManifestName))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manifest.CollectAssets(); err == nil {
		t.Fatal("expected asset size error")
	}
}

func mustWrite(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
