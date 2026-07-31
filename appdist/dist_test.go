package appdist

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
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
	manifest, _ := testFixture(t)
	binary := filepath.Join(t.TempDir(), "app.exe")
	writePEFixture(t, binary, "amd64")
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
	manifest, _ := testFixture(t)
	binary := filepath.Join(t.TempDir(), "app")
	writeMachOFixture(t, binary, "arm64")
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
for last do :; done
printf 'APPIMAGE' > "$last"
chmod +x "$last"
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
	writeELFFixture(t, binary, "amd64")
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

func TestPackageRejectsForeignBinary(t *testing.T) {
	manifest, binary := testFixture(t)
	_, err := Package(Options{Manifest: manifest, Binary: binary, Target: "windows", Arch: "amd64", Format: "portable", OutputDir: t.TempDir(), SourceDateEpoch: 1700000000})
	if err == nil || !strings.Contains(err.Error(), "binary target mismatch") {
		t.Fatalf("expected target mismatch, got %v", err)
	}
}

func TestInspectNativeBinaryFormats(t *testing.T) {
	fixtures := []struct {
		name, osName, arch string
		write              func(*testing.T, string, string)
	}{
		{"elf", "linux", "amd64", writeELFFixture},
		{"pe", "windows", "amd64", writePEFixture},
		{"macho", "macos", "arm64", writeMachOFixture},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), fixture.name)
			fixture.write(t, path, fixture.arch)
			info, err := InspectBinary(path)
			if err != nil {
				t.Fatal(err)
			}
			if info.OS != fixture.osName || info.Arch != fixture.arch {
				t.Fatalf("unexpected binary info: %#v", info)
			}
		})
	}
}

func writeELFFixture(t *testing.T, path, arch string) {
	t.Helper()
	data := make([]byte, 64)
	copy(data[:4], []byte{0x7f, 'E', 'L', 'F'})
	data[4], data[5], data[6] = 2, 1, 1
	machine := uint16(0x3e)
	if arch == "arm64" {
		machine = 0xb7
	}
	data[18], data[19] = byte(machine), byte(machine>>8)
	writeBinaryFixture(t, path, data)
}

func writePEFixture(t *testing.T, path, arch string) {
	t.Helper()
	data := make([]byte, 256)
	data[0], data[1] = 'M', 'Z'
	data[0x3c] = 0x80
	copy(data[0x80:0x84], []byte{'P', 'E', 0, 0})
	machine := uint16(0x8664)
	if arch == "arm64" {
		machine = 0xaa64
	}
	data[0x84], data[0x85] = byte(machine), byte(machine>>8)
	writeBinaryFixture(t, path, data)
}

func writeMachOFixture(t *testing.T, path, arch string) {
	t.Helper()
	data := make([]byte, 64)
	copy(data[:4], []byte{0xcf, 0xfa, 0xed, 0xfe})
	cpu := uint32(0x01000007)
	if arch == "arm64" {
		cpu = 0x0100000c
	}
	data[4], data[5], data[6], data[7] = byte(cpu), byte(cpu>>8), byte(cpu>>16), byte(cpu>>24)
	writeBinaryFixture(t, path, data)
}

func writeBinaryFixture(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestAllRequiresAppImageTool(t *testing.T) {
	manifest, binary := testFixture(t)
	t.Setenv("PATH", t.TempDir())
	t.Setenv("APPIMAGETOOL", "")
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	_, err := Package(Options{Manifest: manifest, Binary: binary, Target: "linux", Arch: "amd64", Format: "all", OutputDir: t.TempDir(), SourceDateEpoch: 1700000000})
	if err == nil || !strings.Contains(err.Error(), "appimagetool is unavailable") {
		t.Fatalf("expected appimagetool requirement, got %v", err)
	}
}

func TestAllRequiresNSIS(t *testing.T) {
	manifest, _ := testFixture(t)
	binary := filepath.Join(t.TempDir(), "app.exe")
	writePEFixture(t, binary, "amd64")
	t.Setenv("PATH", t.TempDir())
	t.Setenv("MAKENSIS", "")
	_, err := Package(Options{Manifest: manifest, Binary: binary, Target: "windows", Arch: "amd64", Format: "all", OutputDir: t.TempDir(), SourceDateEpoch: 1700000000})
	if err == nil || !strings.Contains(err.Error(), "makensis is unavailable") {
		t.Fatalf("expected makensis requirement, got %v", err)
	}
}

func TestProjectLocalAppImageTool(t *testing.T) {
	manifest, _ := testFixture(t)
	tool := filepath.Join(manifest.Root, "tools", "appimagetool-x86_64.AppImage")
	mustWrite(t, tool, "#!/bin/sh\nexit 0\n")
	if err := os.Chmod(tool, 0o755); err != nil {
		t.Fatal(err)
	}
	found, err := FindAppImageTool("", manifest.Root, "amd64")
	if err != nil {
		t.Fatal(err)
	}
	if found != tool {
		t.Fatalf("expected project-local tool %s, got %s", tool, found)
	}
}

func TestPackageRejectsArchitectureMismatch(t *testing.T) {
	manifest, _ := testFixture(t)
	binary := filepath.Join(t.TempDir(), "app")
	writeELFFixture(t, binary, "arm64")
	_, err := Package(Options{Manifest: manifest, Binary: binary, Target: "linux", Arch: "amd64", Format: "bundle", OutputDir: t.TempDir(), SourceDateEpoch: 1700000000})
	if err == nil || !strings.Contains(err.Error(), "binary architecture mismatch") {
		t.Fatalf("expected architecture mismatch, got %v", err)
	}
}

func TestCurrentWorkingDirectoryAppImageToolIsDiscovered(t *testing.T) {
	manifest, _ := testFixture(t)
	cwd := t.TempDir()
	tool := filepath.Join(cwd, "tools", "appimagetool-x86_64.AppImage")
	mustWrite(t, tool, "#!/bin/sh\nexit 0\n")
	if err := os.Chmod(tool, 0o755); err != nil {
		t.Fatal(err)
	}
	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(cwd); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(original) })
	found, err := FindAppImageTool("", manifest.Root, "amd64")
	if err != nil {
		t.Fatal(err)
	}
	if found != tool {
		t.Fatalf("expected cwd tool %s, got %s", tool, found)
	}
}

func TestPackagesPlatformRuntimeFiles(t *testing.T) {
	manifest, linuxBinary := testFixture(t)
	runtimeRoot := filepath.Join(manifest.Root, "runtime")
	mustWrite(t, filepath.Join(runtimeRoot, "libSDL3.so"), "linux runtime")
	mustWrite(t, filepath.Join(runtimeRoot, "SDL3.dll"), "windows runtime")
	mustWrite(t, filepath.Join(runtimeRoot, "libSDL3.dylib"), "mac runtime")
	manifest.Linux.RuntimeFiles = []string{"runtime/libSDL3.so"}
	manifest.Windows.RuntimeFiles = []string{"runtime/SDL3.dll"}
	manifest.MacOS.RuntimeFiles = []string{"runtime/libSDL3.dylib"}

	linux, err := Package(Options{Manifest: manifest, Binary: linuxBinary, Target: "linux", Arch: "amd64", Format: "appdir", OutputDir: t.TempDir(), SourceDateEpoch: 1700000000})
	if err != nil {
		t.Fatal(err)
	}
	appDir := findArtifact(t, linux, "appdir")
	for _, path := range []string{
		filepath.Join(appDir, "usr", "lib", manifest.Slug(), manifest.Slug()+".bin"),
		filepath.Join(appDir, "usr", "lib", manifest.Slug(), "libSDL3.so"),
		filepath.Join(appDir, "usr", "bin", manifest.Slug()),
		filepath.Join(appDir, "usr", "share", "metainfo", manifest.App.Identifier+".appdata.xml"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("missing Linux packaged runtime file %s: %v", path, err)
		}
	}

	windowsBinary := filepath.Join(t.TempDir(), "app.exe")
	writePEFixture(t, windowsBinary, "amd64")
	windows, err := Package(Options{Manifest: manifest, Binary: windowsBinary, Target: "windows", Arch: "amd64", Format: "portable", OutputDir: t.TempDir(), SourceDateEpoch: 1700000000})
	if err != nil {
		t.Fatal(err)
	}
	portable := findArtifact(t, windows, "windows-portable")
	zipReader, err := zip.OpenReader(portable)
	if err != nil {
		t.Fatal(err)
	}
	foundDLL := false
	for _, file := range zipReader.File {
		if strings.HasSuffix(file.Name, "/SDL3.dll") {
			foundDLL = true
		}
	}
	_ = zipReader.Close()
	if !foundDLL {
		t.Fatal("Windows portable package does not contain runtime DLL")
	}

	macBinary := filepath.Join(t.TempDir(), "app-macos")
	writeMachOFixture(t, macBinary, "arm64")
	mac, err := Package(Options{Manifest: manifest, Binary: macBinary, Target: "macos", Arch: "arm64", Format: "app", OutputDir: t.TempDir(), SourceDateEpoch: 1700000000})
	if err != nil {
		t.Fatal(err)
	}
	app := findArtifact(t, mac, "macos-app")
	if _, err := os.Stat(filepath.Join(app, "Contents", "Frameworks", "libSDL3.dylib")); err != nil {
		t.Fatalf("macOS bundle does not contain runtime dylib: %v", err)
	}
}

func TestAppImageUsesPinnedRuntime(t *testing.T) {
	manifest, binary := testFixture(t)
	toolDir := t.TempDir()
	tool := filepath.Join(toolDir, "appimagetool")
	logPath := filepath.Join(toolDir, "args.txt")
	runtimePath := filepath.Join(toolDir, "runtime-x86_64")
	mustWrite(t, runtimePath, "runtime")
	script := `#!/bin/sh
set -eu
printf '%s\n' "$@" > "$ZUMBRA_TEST_ARGS"
for output do :; done
printf 'APPIMAGE' > "$output"
chmod +x "$output"
`
	if err := os.WriteFile(tool, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ZUMBRA_TEST_ARGS", logPath)
	result, err := Package(Options{Manifest: manifest, Binary: binary, Target: "linux", Arch: "amd64", Format: "appimage", OutputDir: t.TempDir(), AppImageTool: tool, AppImageRuntime: runtimePath, SourceDateEpoch: 1700000000})
	if err != nil {
		t.Fatal(err)
	}
	if result.Tools["appimage_runtime"] != runtimePath {
		t.Fatalf("runtime tool was not reported: %#v", result.Tools)
	}
	arguments, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(arguments), "--runtime-file\n"+runtimePath) {
		t.Fatalf("appimagetool did not receive pinned runtime: %s", arguments)
	}
}

func TestCacheAppImageRuntimeFromGeneratedELF(t *testing.T) {
	if _, err := os.Stat("/bin/true"); err != nil {
		t.Skip("/bin/true is unavailable")
	}
	original, err := os.ReadFile("/bin/true")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "test.AppImage")
	payload := append(append([]byte{}, original...), []byte("hsqsfixture")...)
	if err := os.WriteFile(path, payload, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	cached, err := cacheAppImageRuntime(path, "amd64")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(cached)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, original) {
		t.Fatalf("cached runtime has %d bytes, expected %d", len(data), len(original))
	}
}

func TestExplicitAppImageToolIsAuthoritative(t *testing.T) {
	manifest, _ := testFixture(t)
	fallbackRoot := t.TempDir()
	fallback := filepath.Join(fallbackRoot, "tools", "appimagetool-x86_64.AppImage")
	mustWrite(t, fallback, "#!/bin/sh\nexit 0\n")
	if err := os.Chmod(fallback, 0o755); err != nil {
		t.Fatal(err)
	}
	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(fallbackRoot); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(original) })

	missing := filepath.Join(t.TempDir(), "missing-appimagetool")
	if found, err := FindAppImageTool(missing, manifest.Root, "amd64"); err == nil {
		t.Fatalf("explicit missing appimagetool unexpectedly fell back to %s", found)
	}
	candidates := AppImageToolCandidates(missing, manifest.Root, "amd64")
	if len(candidates) != 1 || candidates[0] != missing {
		t.Fatalf("explicit tool candidates must contain only the explicit value: %#v", candidates)
	}
}

func TestExplicitAppImageRuntimeIsAuthoritative(t *testing.T) {
	manifest, _ := testFixture(t)
	fallbackRoot := t.TempDir()
	fallback := filepath.Join(fallbackRoot, "tools", "runtime-x86_64")
	mustWrite(t, fallback, "runtime")
	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(fallbackRoot); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(original) })

	missing := filepath.Join(t.TempDir(), "missing-runtime")
	if found, err := FindAppImageRuntime(missing, manifest.Root, "amd64"); err == nil {
		t.Fatalf("explicit missing runtime unexpectedly fell back to %s", found)
	}
}

func TestExplicitNSISToolIsAuthoritative(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	missing := filepath.Join(t.TempDir(), "missing-makensis")
	if found, err := FindNSISTool(missing); err == nil {
		t.Fatalf("explicit missing makensis unexpectedly resolved to %s", found)
	}
}

func TestLinuxAppStreamMetadataIsCompleteAndNamedByComponentID(t *testing.T) {
	manifest, binary := testFixture(t)
	manifest.Package.Description = "A complete test application."
	output := t.TempDir()
	result, err := Package(Options{
		Manifest: manifest, Binary: binary, Target: "linux", Arch: "amd64",
		Format: "appdir", OutputDir: output, SourceDateEpoch: 1700000000,
	})
	if err != nil {
		t.Fatal(err)
	}
	appDir := findArtifact(t, result, "appdir")
	path := filepath.Join(appDir, "usr", "share", "metainfo", manifest.App.Identifier+".appdata.xml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		ID            string `xml:"id"`
		Summary       string `xml:"summary"`
		Description   string `xml:"description>p"`
		Launchable    string `xml:"launchable"`
		DeveloperName string `xml:"developer>name"`
		Homepage      string `xml:"url"`
		ContentRating struct {
			Type string `xml:"type,attr"`
		} `xml:"content_rating"`
	}
	if err := xml.Unmarshal(data, &document); err != nil {
		t.Fatalf("invalid AppStream XML: %v\n%s", err, data)
	}
	if document.ID != manifest.App.Identifier {
		t.Fatalf("unexpected AppStream ID %q", document.ID)
	}
	if strings.HasSuffix(document.Summary, ".") {
		t.Fatalf("AppStream summary must not end in a period: %q", document.Summary)
	}
	if !strings.Contains(document.Description, manifest.Package.Description) || len([]rune(document.Description)) < 100 {
		t.Fatalf("AppStream description is incomplete or too short: %q", document.Description)
	}
	if document.Launchable != manifest.App.Identifier+".desktop" {
		t.Fatalf("unexpected launchable %q", document.Launchable)
	}
	if document.DeveloperName != manifest.Package.Publisher {
		t.Fatalf("missing developer name: %q", document.DeveloperName)
	}
	if document.Homepage != manifest.Package.Homepage {
		t.Fatalf("missing homepage: %q", document.Homepage)
	}
	if document.ContentRating.Type != "oars-1.1" {
		t.Fatalf("missing content rating: %#v", document.ContentRating)
	}
}

func TestAppImageRequiresHomepage(t *testing.T) {
	manifest, binary := testFixture(t)
	manifest.Package.Homepage = ""
	tool := filepath.Join(t.TempDir(), "appimagetool")
	mustWrite(t, tool, "#!/bin/sh\nexit 0\n")
	if err := os.Chmod(tool, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := Package(Options{Manifest: manifest, Binary: binary, Target: "linux", Arch: "amd64", Format: "appimage", OutputDir: t.TempDir(), AppImageTool: tool})
	if err == nil || !strings.Contains(err.Error(), "package.homepage") {
		t.Fatalf("expected homepage requirement, got %v", err)
	}
}

func TestExplicitMissingExecutableReportsPath(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing-tool")
	_, err := findExecutable([]string{missing}, "test tool")
	if err == nil || !strings.Contains(err.Error(), missing) || !strings.Contains(err.Error(), "not executable") {
		t.Fatalf("unexpected explicit tool error: %v", err)
	}
}
