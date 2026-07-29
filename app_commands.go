package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"zumbra/appdist"
	"zumbra/appmanifest"
	"zumbra/nativec"
	"zumbra/object/builtins"
	"zumbra/pipeline"
)

type appUsageError struct{ message string }

func (e appUsageError) Error() string { return e.message }
func appUsageErrorf(format string, args ...any) error {
	return appUsageError{message: fmt.Sprintf(format, args...)}
}
func isAppUsageError(err error) bool { _, ok := err.(appUsageError); return ok }

type appCLIOptions struct {
	Manifest        string
	Compiler        string
	Output          string
	OutputDir       string
	Release         *bool
	Target          string
	Arch            string
	Format          string
	Binary          string
	AppImageTool    string
	AppImageRuntime string
	NSISTool        string
	SignIdentity    string
	Symbols         bool
	JSON            bool
	SourceDateEpoch int64
}

func handleAppCommand(arguments []string) error {
	if len(arguments) == 0 {
		return appUsageErrorf("missing app command; use inspect, run, build, package or doctor")
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
	case "package":
		return packageApp(manifest, options)
	case "doctor":
		return doctorApp(manifest, options)
	default:
		return appUsageErrorf("unknown app command %q; use inspect, run, build, package or doctor", command)
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
		case "--output-dir":
			index++
			if index >= len(arguments) {
				return options, fmt.Errorf("--output-dir requires a path")
			}
			options.OutputDir = arguments[index]
		case "--target":
			index++
			if index >= len(arguments) {
				return options, fmt.Errorf("--target requires linux, windows, macos or host")
			}
			options.Target = arguments[index]
		case "--arch":
			index++
			if index >= len(arguments) {
				return options, fmt.Errorf("--arch requires amd64, arm64 or host")
			}
			options.Arch = arguments[index]
		case "--format":
			index++
			if index >= len(arguments) {
				return options, fmt.Errorf("--format requires a package format")
			}
			options.Format = arguments[index]
		case "--binary":
			index++
			if index >= len(arguments) {
				return options, fmt.Errorf("--binary requires a path")
			}
			options.Binary = arguments[index]
		case "--appimagetool":
			index++
			if index >= len(arguments) {
				return options, appUsageErrorf("--appimagetool requires a path")
			}
			options.AppImageTool = arguments[index]
		case "--appimage-runtime":
			index++
			if index >= len(arguments) {
				return options, appUsageErrorf("--appimage-runtime requires a path")
			}
			options.AppImageRuntime = arguments[index]
		case "--makensis":
			index++
			if index >= len(arguments) {
				return options, fmt.Errorf("--makensis requires a path")
			}
			options.NSISTool = arguments[index]
		case "--sign":
			index++
			if index >= len(arguments) {
				return options, fmt.Errorf("--sign requires an identity or key ID")
			}
			options.SignIdentity = arguments[index]
		case "--symbols":
			options.Symbols = true
		case "--json":
			options.JSON = true
		case "--source-date-epoch":
			index++
			if index >= len(arguments) {
				return options, fmt.Errorf("--source-date-epoch requires an integer")
			}
			value, err := strconv.ParseInt(arguments[index], 10, 64)
			if err != nil || value < 0 {
				return options, fmt.Errorf("--source-date-epoch must be a non-negative integer")
			}
			options.SourceDateEpoch = value
		default:
			return options, appUsageErrorf("unknown app option %s", argument)
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
	output, buildResult, assetCount, metadataPath, mode, err := buildAppBinary(manifest, cli)
	if err != nil {
		return err
	}
	fmt.Printf("Built %s desktop application: %s\n", mode, output)
	fmt.Printf("Embedded assets: %d\n", assetCount)
	fmt.Printf("Application metadata: %s\n", metadataPath)
	fmt.Printf("Target: %s/%s\n", buildResult.TargetOS, buildResult.TargetArch)
	fmt.Printf("C compiler: %s\n", buildResult.Compiler)
	return nil
}

func buildAppBinary(manifest *appmanifest.Manifest, cli appCLIOptions) (string, *nativec.BuildResult, int, string, string, error) {
	assets, err := collectAppAssets(manifest)
	if err != nil {
		return "", nil, 0, "", "", err
	}
	result, diagnostics := pipeline.BuildFile(manifest.EntryPath(), pipeline.Options{Optimize: true})
	if len(diagnostics) > 0 {
		return "", nil, 0, "", "", fmt.Errorf("pipeline failed:\n%s", pipeline.FormatDiagnostics(diagnostics))
	}
	output := manifest.OutputPath()
	if cli.Output != "" {
		if filepath.IsAbs(cli.Output) {
			output = cli.Output
		} else {
			output = filepath.Join(manifest.Root, cli.Output)
		}
	}
	target := strings.ToLower(strings.TrimSpace(cli.Target))
	if target == "windows" && !strings.HasSuffix(strings.ToLower(output), ".exe") {
		output += ".exe"
	}
	release := manifest.Build.Release
	if cli.Release != nil {
		release = *cli.Release
	}
	compilerName := strings.TrimSpace(cli.Compiler)
	if compilerName == "" {
		compilerName = manifest.Build.Compiler
	}
	if compilerName == "" {
		compilerName = "auto"
	}
	nativeAssets := make([]nativec.EmbeddedAsset, len(assets))
	for index, asset := range assets {
		nativeAssets[index] = nativec.EmbeddedAsset{Name: asset.Name, Data: asset.Data}
	}
	baseName := strings.TrimSuffix(filepath.Base(manifest.App.Entry), filepath.Ext(manifest.App.Entry))
	buildTarget := strings.ToLower(strings.TrimSpace(cli.Target))
	if buildTarget == "" || buildTarget == "host" {
		buildTarget = runtime.GOOS
	}
	if buildTarget == "darwin" {
		buildTarget = "macos"
	}
	buildArch := strings.ToLower(strings.TrimSpace(cli.Arch))
	if buildArch == "" || buildArch == "host" {
		buildArch = runtime.GOARCH
	}
	buildResult, nativeDiagnostics, err := nativec.Build(result.MIR, nativec.BuildOptions{
		Release: release, Compiler: compilerName, Output: output,
		BuildDir: filepath.Join(manifest.Root, "build", "native", baseName, buildTarget, buildArch), EmbeddedAssets: nativeAssets,
		TargetOS: cli.Target, TargetArch: cli.Arch, Reproducible: true, ProjectRoot: manifest.Root, SourceDateEpoch: cli.SourceDateEpoch,
		ApplicationName: manifest.App.Name, ApplicationVersion: manifest.App.Version, ApplicationIdentifier: manifest.App.Identifier,
		WindowsIcon: manifest.IconPathForTarget("windows"), WindowsConsole: manifest.Windows.Console,
	})
	if err != nil {
		return "", nil, 0, "", "", err
	}
	if len(nativeDiagnostics) != 0 {
		messages := make([]string, len(nativeDiagnostics))
		for index, diagnostic := range nativeDiagnostics {
			messages[index] = diagnostic.Error()
		}
		return "", nil, 0, "", "", fmt.Errorf("native backend rejected the application:\n\t%s", strings.Join(messages, "\n\t"))
	}
	if buildResult == nil {
		return "", nil, 0, "", "", fmt.Errorf("native backend returned no build result")
	}
	metadata, err := manifest.Metadata(version, assets[:len(assets)-1])
	if err != nil {
		return "", nil, 0, "", "", err
	}
	metadataPath := output + ".manifest.json"
	if err := os.MkdirAll(filepath.Dir(metadataPath), 0o755); err != nil {
		return "", nil, 0, "", "", err
	}
	if err := os.WriteFile(metadataPath, append(metadata, '\n'), 0o644); err != nil {
		return "", nil, 0, "", "", fmt.Errorf("write build metadata: %w", err)
	}
	mode := "debug"
	if release {
		mode = "release"
	}
	return buildResult.Output, buildResult, len(assets), metadataPath, mode, nil
}

func packageApp(manifest *appmanifest.Manifest, cli appCLIOptions) error {
	binary := strings.TrimSpace(cli.Binary)
	if binary == "" {
		target := strings.ToLower(strings.TrimSpace(cli.Target))
		if target == "" || target == "host" {
			target = runtimeGOOS()
		}
		arch := strings.ToLower(strings.TrimSpace(cli.Arch))
		if arch == "" || arch == "host" {
			arch = runtimeGOARCH()
		}
		name := manifest.Slug()
		if target == "windows" {
			name += ".exe"
		}
		buildOutput := filepath.Join(manifest.Root, "build", "package", target, arch, name)
		buildCLI := cli
		buildCLI.Output = buildOutput
		built, _, _, _, _, err := buildAppBinary(manifest, buildCLI)
		if err != nil {
			return err
		}
		binary = built
	} else if !filepath.IsAbs(binary) {
		binary = filepath.Join(manifest.Root, binary)
	}
	outputDir := cli.OutputDir
	if outputDir == "" && cli.Output != "" {
		outputDir = cli.Output
	}
	result, err := appdist.Package(appdist.Options{Manifest: manifest, ZumbraVersion: version, Binary: binary, Target: cli.Target, Arch: cli.Arch, Format: cli.Format, OutputDir: outputDir, AppImageTool: cli.AppImageTool, AppImageRuntime: cli.AppImageRuntime, NSISTool: cli.NSISTool, SignIdentity: cli.SignIdentity, Symbols: cli.Symbols, SourceDateEpoch: cli.SourceDateEpoch})
	if err != nil {
		return err
	}
	for _, artifact := range result.Artifacts {
		fmt.Printf("Packaged %-20s %s\n", artifact.Kind+":", artifact.Path)
	}
	for _, warning := range result.Warnings {
		fmt.Printf("Package warning: %s\n", warning)
	}
	fmt.Printf("Package report: %s\n", result.ReportPath)
	return nil
}

func runtimeGOOS() string   { return runtime.GOOS }
func runtimeGOARCH() string { return runtime.GOARCH }

type appDoctorCheck struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Required bool   `json:"required"`
	Detail   string `json:"detail"`
}

type appDoctorReport struct {
	SchemaVersion int              `json:"schema_version"`
	Target        string           `json:"target"`
	Arch          string           `json:"arch"`
	Format        string           `json:"format"`
	Ready         bool             `json:"ready"`
	Checks        []appDoctorCheck `json:"checks"`
}

func doctorApp(manifest *appmanifest.Manifest, cli appCLIOptions) error {
	target, err := appdist.NormalizeTarget(cli.Target)
	if err != nil {
		return err
	}
	arch, err := appdist.NormalizeArch(cli.Arch)
	if err != nil {
		return err
	}
	formats, err := appdist.ParseFormats(target, cli.Format)
	if err != nil {
		return err
	}
	formatLabel := strings.TrimSpace(cli.Format)
	if formatLabel == "" {
		formatLabel = "all"
	}
	report := appDoctorReport{SchemaVersion: 1, Target: target, Arch: arch, Format: formatLabel, Ready: true}
	add := func(name string, required bool, checkErr error, detail string) {
		status := "ok"
		if checkErr != nil {
			status = "warning"
			if required {
				status = "missing"
				report.Ready = false
			}
			detail = checkErr.Error()
		}
		report.Checks = append(report.Checks, appDoctorCheck{Name: name, Status: status, Required: required, Detail: detail})
	}

	add("manifest", true, nil, manifest.Path)

	binary := strings.TrimSpace(cli.Binary)
	if binary != "" {
		if !filepath.IsAbs(binary) {
			binary = filepath.Join(manifest.Root, binary)
		}
		info, inspectErr := appdist.InspectBinary(binary)
		if inspectErr == nil {
			if info.OS != target || (info.Arch != arch && info.Arch != "universal") {
				inspectErr = fmt.Errorf("binary is %s/%s, expected %s/%s", info.OS, info.Arch, target, arch)
			}
		}
		add("package binary", true, inspectErr, binary)
	} else {
		result, diagnostics := pipeline.BuildFile(manifest.EntryPath(), pipeline.Options{Optimize: true})
		if len(diagnostics) > 0 {
			add("application pipeline", true, fmt.Errorf("pipeline failed:\n%s", pipeline.FormatDiagnostics(diagnostics)), "")
		} else {
			add("application pipeline", true, nil, manifest.EntryPath())
			if nativec.UsesDesktop(result.MIR) || nativec.UsesUI(result.MIR) {
				add("desktop runtime backend", true, nil, "SDL3 dynamic backend for "+target+"/"+arch)
			} else {
				add("desktop runtime backend", true, nil, "not required")
			}
		}
		compilerName := strings.TrimSpace(cli.Compiler)
		if compilerName == "" {
			compilerName = strings.TrimSpace(manifest.Build.Compiler)
		}
		compiler, compilerErr := nativec.DetectCompilerForTarget(compilerName, target, arch)
		add("C compiler", true, compilerErr, compiler)
		if target == "windows" {
			resource, resourceErr := nativec.DetectWindres(arch)
			add("Windows resource compiler", true, resourceErr, resource)
		}
	}

	wants := func(name string) bool { return formats["all"] || formats[name] }
	if target == "linux" && wants("appimage") {
		tool, toolErr := appdist.FindAppImageTool(cli.AppImageTool, manifest.Root, arch)
		if toolErr != nil {
			toolErr = fmt.Errorf("%s; searched: %s", appdist.AppImageInstallHint(arch), appdist.FormatToolSearch(appdist.AppImageToolCandidates(cli.AppImageTool, manifest.Root, arch)))
		}
		add("appimagetool", true, toolErr, tool)
		runtimePath, runtimeErr := appdist.FindAppImageRuntime(cli.AppImageRuntime, manifest.Root, arch)
		if runtimeErr != nil {
			runtimeErr = fmt.Errorf("%s; searched: %s", appdist.AppImageRuntimeHint(arch), appdist.FormatToolSearch(appdist.AppImageRuntimeCandidates(cli.AppImageRuntime, manifest.Root, arch)))
		}
		add("AppImage runtime", false, runtimeErr, runtimePath)
	}
	if target == "windows" && wants("installer") && manifest.Windows.Installer != "none" {
		tool, toolErr := appdist.FindNSISTool(cli.NSISTool)
		if toolErr != nil {
			toolErr = fmt.Errorf("%s", appdist.NSISInstallHint())
		}
		add("makensis", true, toolErr, tool)
	}
	if target == "macos" && strings.TrimSpace(cli.SignIdentity) != "" {
		tool, toolErr := appdist.FindCodeSignTool()
		add("codesign", true, toolErr, tool)
	}

	if cli.JSON {
		encoded, encodeErr := json.MarshalIndent(report, "", "  ")
		if encodeErr != nil {
			return encodeErr
		}
		fmt.Println(string(encoded))
	} else {
		fmt.Printf("Zumbra application doctor: %s/%s (%s)\n", target, arch, formatLabel)
		for _, check := range report.Checks {
			marker := check.Status
			fmt.Printf("[%s] %-28s %s\n", marker, check.Name, check.Detail)
		}
	}
	if !report.Ready {
		return fmt.Errorf("application doctor found blocking requirements for %s/%s", target, arch)
	}
	return nil
}
