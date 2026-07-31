package nativec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWindowsResourceContainsApplicationMetadata(t *testing.T) {
	icon := filepath.Join(t.TempDir(), "app.ico")
	if err := os.WriteFile(icon, []byte("ico"), 0o644); err != nil {
		t.Fatal(err)
	}
	resource, err := generateWindowsResource(BuildOptions{ApplicationName: "Zumbra App", ApplicationVersion: "1.2.3", ApplicationIdentifier: "dev.zumbra.app", WindowsIcon: icon})
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"1 ICON", "FILEVERSION 1,2,3,0", "Zumbra App", "dev.zumbra.app"} {
		if !strings.Contains(resource, expected) {
			t.Fatalf("resource does not contain %q:\n%s", expected, resource)
		}
	}
}

func TestExecutableNameUsesTarget(t *testing.T) {
	if got := executableNameForTarget("main.zum", "windows"); got != "main.exe" {
		t.Fatalf("unexpected Windows name %q", got)
	}
	if got := executableNameForTarget("main.zum", "linux"); got != "main" {
		t.Fatalf("unexpected Linux name %q", got)
	}
}

func TestDetectTargetCompilerFromEnvironment(t *testing.T) {
	tool := filepath.Join(t.TempDir(), "custom-cc")
	if err := os.WriteFile(tool, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Dir(tool)+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("ZUMBRA_CC_WINDOWS_AMD64", "custom-cc")
	found, err := DetectCompilerForTarget("auto", "windows", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	if found != tool {
		t.Fatalf("expected %s, got %s", tool, found)
	}
}

func TestDesktopRuntimeContainsWindowsAndMacOSBackends(t *testing.T) {
	desktop, err := os.ReadFile(filepath.Join("runtime", "zumbra_desktop.inc"))
	if err != nil {
		t.Fatal(err)
	}
	ui, err := os.ReadFile(filepath.Join("runtime", "zumbra_ui.inc"))
	if err != nil {
		t.Fatal(err)
	}
	joined := string(desktop) + "\n" + string(ui)
	for _, expected := range []string{
		"LoadLibraryA",
		"GetOpenFileNameA",
		"GetSaveFileNameA",
		"OFN_OVERWRITEPROMPT",
		"SHBrowseForFolderA",
		"CreateProcessA",
		"ShellExecuteA",
		"SDL3.dll",
		"SDL3_ttf.dll",
		"__APPLE__",
		"osascript",
		"choose file name",
		"--save --confirm-overwrite",
		"--getsavefilename",
		"_NSGetExecutablePath",
		"@executable_path/../Frameworks/libSDL3",
		"@executable_path/../Frameworks/libSDL3_ttf",
	} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("desktop runtime does not contain %q", expected)
		}
	}
	if strings.Contains(joined, "app->headless=true;\n#endif") {
		t.Fatal("desktop runtime still forces non-Linux targets into headless mode")
	}
}
