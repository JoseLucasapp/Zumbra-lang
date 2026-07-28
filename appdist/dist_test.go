package appdist

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"zumbra/appmanifest"
)

func TestLinuxPackagesAreReproducible(t *testing.T) {
	manifest, binary := testFixture(t)
	first := filepath.Join(t.TempDir(), "one")
	second := filepath.Join(t.TempDir(), "two")
	options := Options{Manifest: manifest, Binary: binary, Target: "linux", Arch: "amd64", Format: "bundle,deb,appdir", OutputDir: first, SourceDateEpoch: 1700000000}
	one, err := Package(options)
	if err != nil {
		t.Fatal(err)
	}
	options.OutputDir = second
	two, err := Package(options)
	if err != nil {
		t.Fatal(err)
	}
	if one.BuildID != two.BuildID {
		t.Fatalf("build IDs differ: %s %s", one.BuildID, two.BuildID)
	}
	oneHashes, twoHashes := artifactHashes(one), artifactHashes(two)
	for kind, hash := range oneHashes {
		if kind == "checksums" {
			continue
		}
		if twoHashes[kind] != hash {
			t.Fatalf("%s is not reproducible: %s != %s", kind, hash, twoHashes[kind])
		}
	}
	deb := findArtifact(t, one, "deb")
	data, err := os.ReadFile(deb)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte("!<arch>\n")) || !bytes.Contains(data, []byte("control.tar.gz/")) || !bytes.Contains(data, []byte("data.tar.gz/")) {
		t.Fatalf("invalid deb archive")
	}
}

func TestWindowsPortableAndInstaller(t *testing.T) {
	manifest, binary := testFixture(t)
	tool := filepath.Join(t.TempDir(), "makensis")
	script := `#!/bin/sh
set -eu
out=$(sed -n 's/^OutFile "\(.*\)"/\1/p' "$1")
printf 'MZfake installer' > "$out"
`
	if err := os.WriteFile(tool, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	result, err := Package(Options{Manifest: manifest, Binary: binary, Target: "windows", Arch: "amd64", Format: "all", OutputDir: t.TempDir(), NSISTool: tool, SourceDateEpoch: 1700000000})
	if err != nil {
		t.Fatal(err)
	}
	portable := findArtifact(t, result, "windows-portable")
	installer := findArtifact(t, result, "windows-installer")
	if _, err := os.Stat(installer); err != nil {
		t.Fatal(err)
	}
	reader, err := zip.OpenReader(portable)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	found := false
	for _, file := range reader.File {
		if strings.HasSuffix(file.Name, "/zumbra-test-app.exe") {
			found = true
		}
	}
	if !found {
		t.Fatal("portable ZIP does not contain executable")
	}
}

func TestMacOSApplicationBundleAndZip(t *testing.T) {
	manifest, binary := testFixture(t)
	result, err := Package(Options{Manifest: manifest, Binary: binary, Target: "macos", Arch: "arm64", Format: "all", OutputDir: t.TempDir(), SourceDateEpoch: 1700000000})
	if err != nil {
		t.Fatal(err)
	}
	app := findArtifact(t, result, "macos-app")
	if _, err := os.Stat(filepath.Join(app, "Contents", "Info.plist")); err != nil {
		t.Fatal(err)
	}
	archive := findArtifact(t, result, "macos-zip")
	reader, err := zip.OpenReader(archive)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	found := false
	for _, file := range reader.File {
		if strings.HasSuffix(file.Name, ".app/Contents/Info.plist") {
			found = true
		}
	}
	if !found {
		t.Fatal("macOS ZIP does not contain Info.plist")
	}
}

func TestAppImageToolIsInvoked(t *testing.T) {
	manifest, binary := testFixture(t)
	tool := filepath.Join(t.TempDir(), "appimagetool")
	script := `#!/bin/sh
set -eu
printf 'APPIMAGE' > "$2"
chmod +x "$2"
`
	if err := os.WriteFile(tool, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	result, err := Package(Options{Manifest: manifest, Binary: binary, Target: "linux", Arch: "amd64", Format: "appimage", OutputDir: t.TempDir(), AppImageTool: tool, SourceDateEpoch: 1700000000})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(findArtifact(t, result, "appimage")); err != nil {
		t.Fatal(err)
	}
}

func testFixture(t *testing.T) (*appmanifest.Manifest, string) {
	t.Helper()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "src", "main.zum"), "show(1);")
	mustWrite(t, filepath.Join(root, "assets", "message.txt"), "hello")
	mustWrite(t, filepath.Join(root, "zumbra.toml"), `[app]
name = "Zumbra Test App"
version = "1.2.3"
identifier = "dev.zumbra.test"
entry = "src/main.zum"

[package]
description = "Test application"
publisher = "Zumbra"
homepage = "https://zumbra.dev"
license = "Apache-2.0"
category = "Development"

[linux]
dependencies = ["libc6"]
recommends = ["libsdl3-0"]

[windows]
console = false
installer = "nsis"

[macos]
minimum_version = "12.0"
category = "public.app-category.developer-tools"

[updates]
url = "https://updates.zumbra.dev/test"
channel = "stable"

[assets]
include = ["assets/**"]
`)
	manifest, err := appmanifest.Load(filepath.Join(root, "zumbra.toml"))
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(root, "build", "app")
	mustWrite(t, binary, "#!/bin/sh\necho test\n")
	if err := os.Chmod(binary, 0o755); err != nil {
		t.Fatal(err)
	}
	return manifest, binary
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
func artifactHashes(result *Result) map[string]string {
	out := map[string]string{}
	for _, a := range result.Artifacts {
		if a.SHA256 != "" {
			out[a.Kind] = a.SHA256
		}
	}
	return out
}
func findArtifact(t *testing.T, result *Result, kind string) string {
	t.Helper()
	for _, a := range result.Artifacts {
		if a.Kind == kind {
			return a.Path
		}
	}
	t.Fatalf("artifact %s not found in %#v", kind, result.Artifacts)
	return ""
}
