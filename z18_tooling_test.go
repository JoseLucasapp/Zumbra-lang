package main

import (
	"os"
	"path/filepath"
	"testing"

	"zumbra/tooling/docgen"
	"zumbra/tooling/formatter"
	"zumbra/tooling/lint"
	"zumbra/tooling/project"
)

func TestZ18VersionAndIntegratedTooling(t *testing.T) {
	if version != "0.14.0" {
		t.Fatalf("Z18 must expose version 0.14.0, got %s", version)
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
