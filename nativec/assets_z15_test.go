package nativec_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"zumbra/nativec"
	"zumbra/pipeline"
)

func TestZ15AssetRuntimeIsConditionallyEnabled(t *testing.T) {
	result, diagnostics := pipeline.Build("assets.zum", `assetText("message.txt");`, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("pipeline: %s", pipeline.FormatDiagnostics(diagnostics))
	}
	sources, nativeDiagnostics := nativec.Generate(result.MIR)
	if len(nativeDiagnostics) != 0 {
		t.Fatalf("native: %#v", nativeDiagnostics)
	}
	runtime := string(sources.Runtime)
	for _, expected := range []string{"#define ZUMBRA_ENABLE_ASSETS 1", "z_embedded_asset_find", "assetText", "z_asset_valid_utf8"} {
		if !strings.Contains(runtime, expected) {
			t.Fatalf("missing %q", expected)
		}
	}
}

func TestZ15BuildWritesEmbeddedAssetSource(t *testing.T) {
	result, diagnostics := pipeline.Build("assets.zum", `show(assetText("assets/message.txt"));`, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("pipeline: %s", pipeline.FormatDiagnostics(diagnostics))
	}
	buildDir := t.TempDir()
	built, nativeDiagnostics, err := nativec.Build(result.MIR, nativec.BuildOptions{
		EmitCOnly:      true,
		BuildDir:       buildDir,
		EmbeddedAssets: []nativec.EmbeddedAsset{{Name: "assets/message.txt", Data: []byte("hello")}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(nativeDiagnostics) != 0 {
		t.Fatalf("native: %#v", nativeDiagnostics)
	}
	data, err := os.ReadFile(filepath.Join(buildDir, "zumbra_assets.c"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "assets/message.txt") || !strings.Contains(string(data), "0x68,0x65,0x6c,0x6c,0x6f") {
		t.Fatalf("unexpected generated source: %s", data)
	}
	if built.AssetsSource == "" {
		t.Fatal("expected asset source path")
	}
}

func TestZ15EmbeddedAssetRunsNatively(t *testing.T) {
	result, diagnostics := pipeline.Build("assets-run.zum", `show(assetText("assets/message.txt")); show(assetExists("assets/message.txt")); show(sizeOf(assetList()));`, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("pipeline: %s", pipeline.FormatDiagnostics(diagnostics))
	}
	root := t.TempDir()
	output := filepath.Join(root, "asset-app")
	built, nativeDiagnostics, err := nativec.Build(result.MIR, nativec.BuildOptions{
		Release:        true,
		Compiler:       "auto",
		Output:         output,
		BuildDir:       filepath.Join(root, "native"),
		EmbeddedAssets: []nativec.EmbeddedAsset{{Name: "assets/message.txt", Data: []byte("olá")}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(nativeDiagnostics) != 0 {
		t.Fatalf("native: %#v", nativeDiagnostics)
	}
	if built == nil {
		t.Fatal("expected build result")
	}
	data, err := exec.Command(output).CombinedOutput()
	if err != nil {
		t.Fatalf("run: %v: %s", err, data)
	}
	if string(data) != "olá\ntrue\n1\n" {
		t.Fatalf("output=%q", data)
	}
}
