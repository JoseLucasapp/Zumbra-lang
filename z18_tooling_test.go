package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"zumbra/tooling/docgen"
	"zumbra/tooling/formatter"
	"zumbra/tooling/lint"
	"zumbra/tooling/project"
)

func TestZ18VersionAndIntegratedTooling(t *testing.T) {
	if version != "0.14.5" {
		t.Fatalf("current release line must expose version 0.14.5, got %s", version)
	}
	source := "/// Main entry.\npub fct main(){show(1);}\nmain();"
	formatted, err := formatter.Format("main.zum", source, formatter.Options{})
	if err != nil {
		t.Fatal(err)
	}
	lintResult := lint.Source("main.zum", formatted.Source, lint.Options{RequirePublicDocs: true})
	if lintResult.Errors != 0 || lintResult.Warnings != 0 {
		t.Fatalf("unexpected lint diagnostics: %#v", lintResult.Diagnostics)
	}
	symbols, err := docgen.Extract("main.zum", formatted.Source, docgen.Options{})
	if err != nil || len(symbols) != 1 || symbols[0].Name != "main" {
		t.Fatalf("unexpected documentation: %#v %v", symbols, err)
	}
}

func TestZ18ProjectScaffold(t *testing.T) {
	root := filepath.Join(t.TempDir(), "sample")
	manifest, created, err := project.Init(project.InitOptions{Directory: root, Name: "Sample", Kind: project.KindCLI})
	if err != nil {
		t.Fatal(err)
	}
	if len(created) == 0 || manifest.EntryPath() != filepath.Join(root, "src", "main.zum") {
		t.Fatalf("unexpected scaffold: %#v %#v", manifest, created)
	}
	if _, err := os.Stat(manifest.EntryPath()); err != nil {
		t.Fatal(err)
	}
}

func TestZ18ToolingHelpCommands(t *testing.T) {
	tests := []struct {
		name string
		run  func() error
	}{
		{name: "check", run: func() error { return handleCheckCommand([]string{"--help"}) }},
		{name: "fmt", run: func() error { return handleFormatCommand([]string{"--help"}) }},
		{name: "lint", run: func() error { return handleLintCommand([]string{"--help"}) }},
		{name: "doc", run: func() error { return handleDocCommand([]string{"--help"}) }},
		{name: "project", run: func() error { return handleProjectCommand([]string{"--help"}) }},
		{name: "project init", run: func() error { return handleProjectCommand([]string{"init", "--help"}) }},
		{name: "project build", run: func() error { return handleProjectCommand([]string{"build", "--help"}) }},
		{name: "profile", run: func() error { return handleProfileCommand([]string{"--help"}) }},
		{name: "lsp", run: func() error { return handleLSPCommand([]string{"--help"}) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.run(); err != nil {
				t.Fatalf("help returned an error: %v", err)
			}
		})
	}
}

func TestZ18ReleaseDocumentation(t *testing.T) {
	read := func(path string) string {
		t.Helper()
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("required Z18 documentation %s is unavailable: %v", path, err)
		}
		return string(content)
	}

	readme := read("README.MD")
	if !strings.Contains(readme, "docs/releases/0.14.0.md") {
		t.Fatal("README.MD must link to the Zumbra 0.14.0 release notes")
	}

	releaseNotes := read(filepath.Join("docs", "releases", "0.14.0.md"))
	for _, required := range []string{
		"# Zumbra 0.14.0",
		"Z18",
		"scripts/test-z18-tooling.sh",
		"scripts/check-repository-hygiene.sh",
	} {
		if !strings.Contains(releaseNotes, required) {
			t.Fatalf("release notes must contain %q", required)
		}
	}

	toolingGuide := read(filepath.Join("docs", "pt-BR", "tooling-z18.md"))
	if !strings.Contains(toolingGuide, "0.14.0") || !strings.Contains(toolingGuide, "Z18") {
		t.Fatal("Z18 tooling guide must identify the 0.14.0 milestone")
	}
}

func TestZumbra0141ReleaseDocumentation(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("docs", "releases", "0.14.1.md"))
	if err != nil {
		t.Fatalf("required 0.14.1 release notes are unavailable: %v", err)
	}
	notes := string(content)
	for _, required := range []string{
		"# Zumbra 0.14.1",
		"Boolean.Value",
		"scripts/test-0.14.1-fixed-bool.sh",
		"EXPECTED_ZUMBRA_VERSION=0.14.1",
	} {
		if !strings.Contains(notes, required) {
			t.Fatalf("0.14.1 release notes must contain %q", required)
		}
	}

	readme, err := os.ReadFile("README.MD")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(readme), "docs/releases/0.14.1.md") {
		t.Fatal("README.MD must link to the Zumbra 0.14.1 release notes")
	}
}

func TestZumbra0142ReleaseDocumentation(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("docs", "releases", "0.14.2.md"))
	if err != nil {
		t.Fatalf("required 0.14.2 release notes are unavailable: %v", err)
	}
	notes := string(content)
	for _, required := range []string{
		"# Zumbra 0.14.2",
		"panic",
		"nativec",
		"scripts/test-0.14.2-native-panic.sh",
		"EXPECTED_ZUMBRA_VERSION=0.14.2",
	} {
		if !strings.Contains(notes, required) {
			t.Fatalf("0.14.2 release notes must contain %q", required)
		}
	}

	readme, err := os.ReadFile("README.MD")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(readme), "docs/releases/0.14.2.md") {
		t.Fatal("README.MD must link to the Zumbra 0.14.2 release notes")
	}
}

func TestZumbra0145ReleaseDocumentation(t *testing.T) {
	content, err := os.ReadFile("docs/releases/0.14.5.md")
	if err != nil {
		t.Fatalf("0.14.5 release notes must exist: %v", err)
	}

	if !strings.Contains(string(content), "# Zumbra 0.14.5") {
		t.Fatalf(`0.14.5 release notes must contain "# Zumbra 0.14.5"`)
	}
}
