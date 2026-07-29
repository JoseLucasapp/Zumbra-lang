package appdist

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"zumbra/appmanifest"
)

const defaultEpoch int64 = 946684800 // 2000-01-01T00:00:00Z

type Options struct {
	Manifest        *appmanifest.Manifest
	ZumbraVersion   string
	Binary          string
	Target          string
	Arch            string
	Format          string
	OutputDir       string
	AppImageTool    string
	AppImageRuntime string
	NSISTool        string
	SignIdentity    string
	Symbols         bool
	SourceDateEpoch int64
}

type Artifact struct {
	Kind   string `json:"kind"`
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type Result struct {
	SchemaVersion int               `json:"schema_version"`
	App           string            `json:"app"`
	Version       string            `json:"version"`
	Target        string            `json:"target"`
	Arch          string            `json:"arch"`
	Binary        BinaryInfo        `json:"binary"`
	BuildID       string            `json:"build_id"`
	Artifacts     []Artifact        `json:"artifacts"`
	Warnings      []string          `json:"warnings,omitempty"`
	Tools         map[string]string `json:"tools,omitempty"`
	ReportPath    string            `json:"report_path"`
}

type packageContext struct {
	options Options
	result  Result
	outDir  string
	epoch   time.Time
	formats map[string]bool
}

func Package(options Options) (*Result, error) {
	if options.Manifest == nil {
		return nil, fmt.Errorf("package manifest is required")
	}
	if strings.TrimSpace(options.Binary) == "" {
		return nil, fmt.Errorf("package binary is required")
	}
	binary, err := filepath.Abs(options.Binary)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(binary)
	if err != nil {
		return nil, fmt.Errorf("read package binary: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("package binary cannot be a directory")
	}
	options.Binary = binary
	options.Target, err = normalizeTarget(options.Target)
	if err != nil {
		return nil, err
	}
	options.Arch, err = normalizeArch(options.Arch)
	if err != nil {
		return nil, err
	}
	formats, err := parseFormats(options.Target, options.Format)
	if err != nil {
		return nil, err
	}
	wants := func(name string) bool { return formats["all"] || formats[name] }
	if options.Target == "linux" && wants("appimage") {
		tool, findErr := FindAppImageTool(options.AppImageTool, options.Manifest.Root, options.Arch)
		if findErr != nil {
			return nil, fmt.Errorf("AppImage requested but appimagetool is unavailable: %s", AppImageInstallHint(options.Arch))
		}
		options.AppImageTool = tool
		if runtimePath, runtimeErr := FindAppImageRuntime(options.AppImageRuntime, options.Manifest.Root, options.Arch); runtimeErr == nil {
			options.AppImageRuntime = runtimePath
		}
	}
	if options.Target == "windows" && wants("installer") && options.Manifest.Windows.Installer != "none" {
		tool, findErr := FindNSISTool(options.NSISTool)
		if findErr != nil {
			return nil, fmt.Errorf("Windows installer requested but makensis is unavailable: %s", NSISInstallHint())
		}
		options.NSISTool = tool
	}
	if options.OutputDir == "" {
		options.OutputDir = filepath.Join(options.Manifest.Root, "dist", "packages")
	} else if !filepath.IsAbs(options.OutputDir) {
		options.OutputDir = filepath.Join(options.Manifest.Root, options.OutputDir)
	}
	if err := os.MkdirAll(options.OutputDir, 0o755); err != nil {
		return nil, fmt.Errorf("create package output directory: %w", err)
	}
	epoch := sourceDateEpoch(options.SourceDateEpoch)
	binaryInfo, err := validateBinaryForTarget(binary, options.Target, options.Arch)
	if err != nil {
		return nil, err
	}
	buildID, err := calculateBuildID(options.Manifest, binary, options.Target, options.Arch)
	if err != nil {
		return nil, err
	}
	ctx := &packageContext{
		options: options,
		result:  Result{SchemaVersion: 2, Tools: map[string]string{}, App: options.Manifest.App.Name, Version: options.Manifest.App.Version, Target: options.Target, Arch: options.Arch, Binary: binaryInfo, BuildID: buildID},
		outDir:  options.OutputDir,
		epoch:   epoch,
		formats: formats,
	}

	switch options.Target {
	case "linux":
		err = ctx.packageLinux()
	case "windows":
		err = ctx.packageWindows()
	case "macos":
		err = ctx.packageMacOS()
	default:
		err = fmt.Errorf("unsupported package target %q", options.Target)
	}
	if err != nil {
		return nil, err
	}
	if options.Symbols {
		if err := ctx.generateSymbols(); err != nil {
			return nil, err
		}
	}
	if err := ctx.signArtifacts(); err != nil {
		return nil, err
	}
	if err := ctx.refreshArtifacts(); err != nil {
		return nil, err
	}
	sort.Slice(ctx.result.Artifacts, func(i, j int) bool {
		if ctx.result.Artifacts[i].Kind == ctx.result.Artifacts[j].Kind {
			return ctx.result.Artifacts[i].Path < ctx.result.Artifacts[j].Path
		}
		return ctx.result.Artifacts[i].Kind < ctx.result.Artifacts[j].Kind
	})
	if err := ctx.writeUpdateDescriptor(); err != nil {
		return nil, err
	}
	if err := ctx.writeChecksums(); err != nil {
		return nil, err
	}
	if err := ctx.writeReport(); err != nil {
		return nil, err
	}
	return &ctx.result, nil
}

func normalizeTarget(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" || value == "host" {
		value = runtime.GOOS
	}
	if value == "darwin" || value == "mac" || value == "osx" {
		value = "macos"
	}
	if value != "linux" && value != "windows" && value != "macos" {
		return "", fmt.Errorf("target must be linux, windows, macos or host")
	}
	return value, nil
}

func normalizeArch(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" || value == "host" {
		value = runtime.GOARCH
	}
	switch value {
	case "x86_64", "x64":
		value = "amd64"
	case "aarch64":
		value = "arm64"
	}
	if value != "amd64" && value != "arm64" {
		return "", fmt.Errorf("architecture must be amd64, arm64 or host")
	}
	return value, nil
}

func parseFormats(target, raw string) (map[string]bool, error) {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		raw = "all"
	}
	allowed := map[string]map[string]bool{
		"linux":   {"all": true, "bundle": true, "deb": true, "appimage": true, "appdir": true},
		"windows": {"all": true, "portable": true, "installer": true},
		"macos":   {"all": true, "app": true, "zip": true},
	}
	result := map[string]bool{}
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if !allowed[target][item] {
			return nil, fmt.Errorf("format %q is not supported for %s", item, target)
		}
		result[item] = true
	}
	return result, nil
}

func (c *packageContext) wants(format string) bool    { return c.formats["all"] || c.formats[format] }
func (c *packageContext) explicit(format string) bool { return c.formats[format] && !c.formats["all"] }

func sourceDateEpoch(value int64) time.Time {
	if value <= 0 {
		if raw := strings.TrimSpace(os.Getenv("SOURCE_DATE_EPOCH")); raw != "" {
			if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil && parsed >= 0 {
				value = parsed
			}
		}
	}
	if value <= 0 {
		value = defaultEpoch
	}
	return time.Unix(value, 0).UTC()
}

func calculateBuildID(manifest *appmanifest.Manifest, binary, target, arch string) (string, error) {
	hash := sha256.New()
	data, err := os.ReadFile(manifest.Path)
	if err != nil {
		return "", err
	}
	hash.Write(data)
	file, err := os.Open(binary)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(hash, file); err != nil {
		_ = file.Close()
		return "", err
	}
	_ = file.Close()
	_, _ = io.WriteString(hash, "\x00"+target+"\x00"+arch)
	return hex.EncodeToString(hash.Sum(nil))[:24], nil
}

func (c *packageContext) baseName() string {
	return fmt.Sprintf("%s-%s-%s-%s", c.options.Manifest.Slug(), c.options.Manifest.App.Version, c.options.Target, c.options.Arch)
}

func (c *packageContext) metadata() map[string]any {
	return map[string]any{
		"schema_version":    1,
		"zumbra_version":    firstNonEmpty(c.options.ZumbraVersion, "unknown"),
		"app":               c.options.Manifest.App,
		"package":           c.options.Manifest.Package,
		"updates":           c.options.Manifest.Updates,
		"target":            c.options.Target,
		"arch":              c.options.Arch,
		"build_id":          c.result.BuildID,
		"source_date_epoch": c.epoch.Unix(),
	}
}

func (c *packageContext) writeMetadata(path string) error {
	data, err := json.MarshalIndent(c.metadata(), "", "  ")
	if err != nil {
		return err
	}
	return writeFile(path, append(data, '\n'), 0o644, c.epoch)
}

func (c *packageContext) addArtifact(kind, path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		c.result.Artifacts = append(c.result.Artifacts, Artifact{Kind: kind, Path: path})
		return nil
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	h := sha256.New()
	_, copyErr := io.Copy(h, file)
	_ = file.Close()
	if copyErr != nil {
		return copyErr
	}
	c.result.Artifacts = append(c.result.Artifacts, Artifact{Kind: kind, Path: path, Size: info.Size(), SHA256: hex.EncodeToString(h.Sum(nil))})
	return nil
}

func (c *packageContext) refreshArtifacts() error {
	for index := range c.result.Artifacts {
		artifact := &c.result.Artifacts[index]
		info, err := os.Stat(artifact.Path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			artifact.Size = 0
			artifact.SHA256 = ""
			continue
		}
		file, err := os.Open(artifact.Path)
		if err != nil {
			return err
		}
		h := sha256.New()
		_, copyErr := io.Copy(h, file)
		_ = file.Close()
		if copyErr != nil {
			return copyErr
		}
		artifact.Size = info.Size()
		artifact.SHA256 = hex.EncodeToString(h.Sum(nil))
	}
	return nil
}

func (c *packageContext) writeChecksums() error {
	var lines []string
	for _, artifact := range c.result.Artifacts {
		if artifact.SHA256 == "" {
			continue
		}
		lines = append(lines, artifact.SHA256+"  "+filepath.Base(artifact.Path))
	}
	sort.Strings(lines)
	path := filepath.Join(c.outDir, c.baseName()+"-SHA256SUMS.txt")
	if err := writeFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644, c.epoch); err != nil {
		return err
	}
	return c.addArtifact("checksums", path)
}

func (c *packageContext) writeUpdateDescriptor() error {
	if strings.TrimSpace(c.options.Manifest.Updates.URL) == "" {
		return nil
	}
	type releaseArtifact struct {
		Kind   string `json:"kind"`
		URL    string `json:"url"`
		Size   int64  `json:"size"`
		SHA256 string `json:"sha256"`
	}
	base := strings.TrimRight(c.options.Manifest.Updates.URL, "/")
	items := []releaseArtifact{}
	for _, artifact := range c.result.Artifacts {
		if artifact.SHA256 == "" || artifact.Kind == "checksums" {
			continue
		}
		items = append(items, releaseArtifact{Kind: artifact.Kind, URL: base + "/" + filepath.Base(artifact.Path), Size: artifact.Size, SHA256: artifact.SHA256})
	}
	payload := struct {
		SchemaVersion int               `json:"schema_version"`
		Identifier    string            `json:"identifier"`
		Version       string            `json:"version"`
		Channel       string            `json:"channel"`
		Target        string            `json:"target"`
		Arch          string            `json:"arch"`
		BuildID       string            `json:"build_id"`
		Artifacts     []releaseArtifact `json:"artifacts"`
	}{1, c.options.Manifest.App.Identifier, c.options.Manifest.App.Version, c.options.Manifest.Updates.Channel, c.options.Target, c.options.Arch, c.result.BuildID, items}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(c.outDir, c.baseName()+"-update.json")
	if err := writeFile(path, append(data, '\n'), 0o644, c.epoch); err != nil {
		return err
	}
	return c.addArtifact("update-metadata", path)
}

func (c *packageContext) writeReport() error {
	path := filepath.Join(c.outDir, c.baseName()+"-package-report.json")
	c.result.ReportPath = path
	data, err := json.MarshalIndent(c.result, "", "  ")
	if err != nil {
		return err
	}
	return writeFile(path, append(data, '\n'), 0o644, c.epoch)
}

func (c *packageContext) auditDependencies(binary, destination string) {
	var command *exec.Cmd
	switch c.options.Target {
	case "linux":
		if tool, err := exec.LookPath("ldd"); err == nil {
			command = exec.Command(tool, binary)
		}
	case "macos":
		if tool, err := exec.LookPath("otool"); err == nil {
			command = exec.Command(tool, "-L", binary)
		}
	case "windows":
		for _, name := range []string{"objdump", "llvm-objdump"} {
			if tool, err := exec.LookPath(name); err == nil {
				command = exec.Command(tool, "-p", binary)
				break
			}
		}
	}
	if command == nil {
		c.result.Warnings = append(c.result.Warnings, "dependency audit tool was not found for "+c.options.Target)
		return
	}
	output, err := command.CombinedOutput()
	if err != nil {
		c.result.Warnings = append(c.result.Warnings, "dependency audit failed: "+err.Error())
		return
	}
	// Runtime loaders print randomized virtual addresses. Normalize them so
	// dependency reports remain reproducible across package runs.
	output = regexp.MustCompile(`0x[0-9a-fA-F]+`).ReplaceAll(output, []byte("0xADDR"))
	_ = writeFile(destination, output, 0o644, c.epoch)
}

// NormalizeTarget exposes the same target normalization used by the packager.
func NormalizeTarget(value string) (string, error) { return normalizeTarget(value) }

// NormalizeArch exposes the same architecture normalization used by the packager.
func NormalizeArch(value string) (string, error) { return normalizeArch(value) }

// ParseFormats validates a comma-separated package format list for a target.
func ParseFormats(target, value string) (map[string]bool, error) { return parseFormats(target, value) }
