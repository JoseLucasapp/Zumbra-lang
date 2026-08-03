package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitLoadAndDiscoverProject(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo")
	manifest, created, err := Init(InitOptions{Directory: root, Name: "Demo Project", Kind: KindLibrary})
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Kind != KindLibrary || len(created) < 5 {
		t.Fatalf("unexpected project: %#v %#v", manifest, created)
	}
	files, err := manifest.SourceFiles(true)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("expected source and test file, got %#v", files)
	}
	nested := filepath.Join(root, "src")
	found, err := Find(nested)
	if err != nil || found != filepath.Join(root, ManifestName) {
		t.Fatalf("find: %q %v", found, err)
	}
}

func TestInitDoesNotOverwriteNonEmptyDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "keep.txt"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Init(InitOptions{Directory: root, Name: "Demo"}); err == nil {
		t.Fatal("expected non-empty directory error")
	}
}

func TestDesktopIdentifierValidation(t *testing.T) {
	root := filepath.Join(t.TempDir(), "desktop")
	if _, _, err := Init(InitOptions{Directory: root, Name: "Desktop", Kind: KindDesktop, Identifier: "invalid"}); err == nil {
		t.Fatal("expected reverse-DNS identifier validation error")
	}
	manifest, _, err := Init(InitOptions{Directory: root, Name: "Desktop", Kind: KindDesktop, Identifier: "dev.example.desktop"})
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Kind != KindDesktop {
		t.Fatalf("unexpected kind: %s", manifest.Kind)
	}
}

func TestProjectNameRequiresSlugContent(t *testing.T) {
	if _, _, err := Init(InitOptions{Directory: filepath.Join(t.TempDir(), "invalid"), Name: "!!!"}); err == nil {
		t.Fatal("expected invalid project name error")
	}
}
