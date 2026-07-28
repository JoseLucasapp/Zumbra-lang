package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"zumbra/appmanifest"
	"zumbra/nativec"
	"zumbra/object/builtins"
	"zumbra/pipeline"
)

type appCLIOptions struct {
	Manifest string
	Compiler string
	Output   string
	Release  *bool
}

func handleAppCommand(arguments []string) error {
	if len(arguments) == 0 {
		return fmt.Errorf("missing app command; use inspect, run or build")
	}
	command := arguments[0]
	options, err := parseAppOptions(arguments[1:])
	if err != nil {
		return err
	}
	manifest, err := appmanifest.Load(options.Manifest)
	if err != nil {
		return err
	}
	switch command {
	case "inspect":
		return inspectApp(manifest)
	case "run":
		return runApp(manifest)
	case "build":
		return buildApp(manifest, options)
	default:
		return fmt.Errorf("unknown app command %q; use inspect, run or build", command)
	}
}

func parseAppOptions(arguments []string) (appCLIOptions, error) {
	options := appCLIOptions{Manifest: appmanifest.DefaultManifestName}
	for index := 0; index < len(arguments); index++ {
		switch argument := arguments[index]; argument {
		case "--manifest", "-m":
			index++
			if index >= len(arguments) {
				return options, fmt.Errorf("%s requires a path", argument)
			}
			options.Manifest = arguments[index]
		case "--compiler":
			index++
			if index >= len(arguments) {
				return options, fmt.Errorf("--compiler requires clang, gcc, cc or auto")
			}
			options.Compiler = arguments[index]
		case "--release":
			value := true
			options.Release = &value
		case "--debug":
			value := false
			options.Release = &value
		case "-o", "--output":
			index++
			if index >= len(arguments) {
				return options, fmt.Errorf("%s requires a path", argument)
			}
			options.Output = arguments[index]
		default:
			return options, fmt.Errorf("unknown app option %s", argument)
		}
	}
	return options, nil
}

func collectAppAssets(manifest *appmanifest.Manifest) ([]appmanifest.EmbeddedAsset, error) {
	assets, err := manifest.CollectAssets()
	if err != nil {
		return nil, err
	}
	metadata, err := manifest.Metadata(version, assets)
	if err != nil {
		return nil, fmt.Errorf("encode application metadata: %w", err)
	}
	sum := sha256.Sum256(metadata)
	assets = append(assets, appmanifest.EmbeddedAsset{Name: "__zumbra__/manifest.json", Data: metadata, Size: int64(len(metadata)), SHA256: hex.EncodeToString(sum[:])})
	return assets, nil
}

func configureRuntimeAssets(assets []appmanifest.EmbeddedAsset) error {
	values := make([]builtins.EmbeddedAsset, len(assets))
	for index, asset := range assets {
		values[index] = builtins.EmbeddedAsset{Name: asset.Name, Data: asset.Data}
	}
	return builtins.ConfigureEmbeddedAssets(values)
}

func inspectApp(manifest *appmanifest.Manifest) error {
	assets, err := collectAppAssets(manifest)
	if err != nil {
		return err
	}
	type assetInfo struct {
		Name   string `json:"name"`
		Size   int64  `json:"size"`
		SHA256 string `json:"sha256,omitempty"`
	}
	listed := make([]assetInfo, len(assets))
	for index, asset := range assets {
		listed[index] = assetInfo{Name: asset.Name, Size: asset.Size, SHA256: asset.SHA256}
	}
	payload := struct {
		Manifest *appmanifest.Manifest `json:"manifest"`
		Assets   []assetInfo           `json:"embedded_assets"`
		Output   string                `json:"output"`
	}{Manifest: manifest, Assets: listed, Output: manifest.OutputPath()}
	encoded, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(encoded))
	return nil
}

func runApp(manifest *appmanifest.Manifest) error {
	assets, err := collectAppAssets(manifest)
	if err != nil {
		return err
	}
	if err := configureRuntimeAssets(assets); err != nil {
		return err
	}
	defer builtins.ResetEmbeddedAssets()
	current, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("read current directory: %w", err)
	}
	if err := os.Chdir(manifest.Root); err != nil {
		return fmt.Errorf("enter application root: %w", err)
	}
	defer func() { _ = os.Chdir(current) }()
	runFile(manifest.EntryPath())
	return nil
}

func buildApp(manifest *appmanifest.Manifest, cli appCLIOptions) error {
	assets, err := collectAppAssets(manifest)
	if err != nil {
		return err
	}
	result, diagnostics := pipeline.BuildFile(manifest.EntryPath(), pipeline.Options{Optimize: true})
	if len(diagnostics) > 0 {
		return fmt.Errorf("pipeline failed:\n%s", pipeline.FormatDiagnostics(diagnostics))
	}

	output := manifest.OutputPath()
	if cli.Output != "" {
		if filepath.IsAbs(cli.Output) {
			output = cli.Output
		} else {
			output = filepath.Join(manifest.Root, cli.Output)
		}
	}
	release := manifest.Build.Release
	if cli.Release != nil {
		release = *cli.Release
	}
	compiler := strings.TrimSpace(cli.Compiler)
	if compiler == "" {
		compiler = manifest.Build.Compiler
	}
	if compiler == "" {
		compiler = "auto"
	}

	nativeAssets := make([]nativec.EmbeddedAsset, len(assets))
	for index, asset := range assets {
		nativeAssets[index] = nativec.EmbeddedAsset{Name: asset.Name, Data: asset.Data}
	}
	baseName := strings.TrimSuffix(filepath.Base(manifest.App.Entry), filepath.Ext(manifest.App.Entry))
	buildResult, nativeDiagnostics, err := nativec.Build(result.MIR, nativec.BuildOptions{
		Release: release, Compiler: compiler, Output: output,
		BuildDir: filepath.Join(manifest.Root, "build", "native", baseName), EmbeddedAssets: nativeAssets,
	})
	if err != nil {
		return err
	}
	if len(nativeDiagnostics) != 0 {
		messages := make([]string, len(nativeDiagnostics))
		for index, diagnostic := range nativeDiagnostics {
			messages[index] = diagnostic.Error()
		}
		return fmt.Errorf("native backend rejected the application:\n\t%s", strings.Join(messages, "\n\t"))
	}
	if buildResult == nil {
		return fmt.Errorf("native backend returned no build result")
	}
	metadata, err := manifest.Metadata(version, assets[:len(assets)-1])
	if err != nil {
		return err
	}
	metadataPath := output + ".manifest.json"
	if err := os.MkdirAll(filepath.Dir(metadataPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(metadataPath, append(metadata, '\n'), 0o644); err != nil {
		return fmt.Errorf("write build metadata: %w", err)
	}
	mode := "debug"
	if release {
		mode = "release"
	}
	fmt.Printf("Built %s desktop application: %s\n", mode, buildResult.Output)
	fmt.Printf("Embedded assets: %d\n", len(assets))
	fmt.Printf("Application metadata: %s\n", metadataPath)
	fmt.Printf("C compiler: %s\n", buildResult.Compiler)
	return nil
}
