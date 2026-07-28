package appmanifest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	DefaultManifestName  = "zumbra.toml"
	DefaultMaxFileBytes  = int64(16 << 20)
	DefaultMaxTotalBytes = int64(64 << 20)
)

type App struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	Identifier string `json:"identifier"`
	Entry      string `json:"entry"`
	Icon       string `json:"icon,omitempty"`
}

type Build struct {
	Output   string `json:"output,omitempty"`
	Compiler string `json:"compiler,omitempty"`
	Release  bool   `json:"release"`
}

type Assets struct {
	Include       []string `json:"include,omitempty"`
	Exclude       []string `json:"exclude,omitempty"`
	MaxFileBytes  int64    `json:"max_file_bytes"`
	MaxTotalBytes int64    `json:"max_total_bytes"`
}

type Manifest struct {
	Path   string `json:"manifest_path"`
	Root   string `json:"project_root"`
	App    App    `json:"app"`
	Build  Build  `json:"build"`
	Assets Assets `json:"assets"`
}

type EmbeddedAsset struct {
	Name   string `json:"name"`
	Path   string `json:"path,omitempty"`
	Data   []byte `json:"-"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type BuildMetadata struct {
	SchemaVersion int             `json:"schema_version"`
	ZumbraVersion string          `json:"zumbra_version"`
	App           App             `json:"app"`
	Assets        []EmbeddedAsset `json:"assets"`
}

func Load(path string) (*Manifest, error) {
	if strings.TrimSpace(path) == "" {
		path = DefaultManifestName
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve manifest: %w", err)
	}
	data, err := os.ReadFile(absolute)
	if err != nil {
		return nil, fmt.Errorf("read manifest %s: %w", absolute, err)
	}
	manifest := &Manifest{
		Path:   absolute,
		Root:   filepath.Dir(absolute),
		Build:  Build{Compiler: "auto", Release: true},
		Assets: Assets{MaxFileBytes: DefaultMaxFileBytes, MaxTotalBytes: DefaultMaxTotalBytes},
	}
	if err := parse(data, manifest); err != nil {
		return nil, fmt.Errorf("parse %s: %w", absolute, err)
	}
	if err := manifest.Validate(); err != nil {
		return nil, err
	}
	return manifest, nil
}

func (m *Manifest) Validate() error {
	if strings.TrimSpace(m.App.Name) == "" {
		return fmt.Errorf("manifest [app].name is required")
	}
	if !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:[-+][0-9A-Za-z.-]+)?$`).MatchString(m.App.Version) {
		return fmt.Errorf("manifest [app].version must be semantic version, got %q", m.App.Version)
	}
	if !regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*(?:\.[A-Za-z0-9_-]+)+$`).MatchString(m.App.Identifier) {
		return fmt.Errorf("manifest [app].identifier must be reverse-DNS style, got %q", m.App.Identifier)
	}
	if strings.TrimSpace(m.App.Entry) == "" {
		return fmt.Errorf("manifest [app].entry is required")
	}
	if m.Build.Compiler != "" && m.Build.Compiler != "auto" && m.Build.Compiler != "clang" && m.Build.Compiler != "gcc" && m.Build.Compiler != "cc" {
		return fmt.Errorf("manifest [build].compiler must be auto, clang, gcc or cc")
	}
	if m.Build.Output != "" {
		if _, err := m.Resolve(m.Build.Output); err != nil {
			return fmt.Errorf("build output: %w", err)
		}
	}
	entry, err := m.Resolve(m.App.Entry)
	if err != nil {
		return fmt.Errorf("entry: %w", err)
	}
	info, err := os.Stat(entry)
	if err != nil {
		return fmt.Errorf("entry %s: %w", entry, err)
	}
	if info.IsDir() || strings.ToLower(filepath.Ext(entry)) != ".zum" {
		return fmt.Errorf("entry must be a .zum file")
	}
	if m.App.Icon != "" {
		icon, err := m.Resolve(m.App.Icon)
		if err != nil {
			return fmt.Errorf("icon: %w", err)
		}
		info, err := os.Stat(icon)
		if err != nil {
			return fmt.Errorf("icon %s: %w", icon, err)
		}
		if info.IsDir() {
			return fmt.Errorf("icon must be a file")
		}
	}
	if m.Assets.MaxFileBytes <= 0 || m.Assets.MaxTotalBytes <= 0 {
		return fmt.Errorf("asset size limits must be positive")
	}
	if m.Assets.MaxFileBytes > m.Assets.MaxTotalBytes {
		return fmt.Errorf("assets.max_file_bytes cannot exceed assets.max_total_bytes")
	}
	return nil
}

func (m *Manifest) Resolve(relative string) (string, error) {
	relative = strings.TrimSpace(relative)
	if relative == "" {
		return "", fmt.Errorf("path cannot be empty")
	}
	candidate := relative
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(m.Root, filepath.FromSlash(candidate))
	}
	candidate, err := filepath.Abs(candidate)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(m.Root, candidate)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes project root", relative)
	}
	return candidate, nil
}

func (m *Manifest) EntryPath() string { path, _ := m.Resolve(m.App.Entry); return path }

func (m *Manifest) OutputPath() string {
	if strings.TrimSpace(m.Build.Output) != "" {
		path, _ := m.Resolve(m.Build.Output)
		return path
	}
	return filepath.Join(m.Root, "dist", slug(m.App.Name))
}

func (m *Manifest) CollectAssets() ([]EmbeddedAsset, error) {
	includes := append([]string(nil), m.Assets.Include...)
	if m.App.Icon != "" {
		found := false
		for _, include := range includes {
			if filepath.ToSlash(strings.TrimSpace(include)) == filepath.ToSlash(m.App.Icon) {
				found = true
				break
			}
		}
		if !found {
			includes = append(includes, m.App.Icon)
		}
	}
	patterns, err := compilePatterns(includes)
	if err != nil {
		return nil, err
	}
	excludes, err := compilePatterns(m.Assets.Exclude)
	if err != nil {
		return nil, err
	}
	assets := []EmbeddedAsset{}
	var total int64
	err = filepath.WalkDir(m.Root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == m.Root {
			return nil
		}
		rel, err := filepath.Rel(m.Root, path)
		if err != nil {
			return err
		}
		logical := filepath.ToSlash(rel)
		if entry.IsDir() {
			if logical == ".git" || logical == "build" || logical == "dist" {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("asset %s is a symbolic link; symlinks are not packaged", logical)
		}
		if !matchesAny(patterns, logical) || matchesAny(excludes, logical) {
			return nil
		}
		if info.Size() > m.Assets.MaxFileBytes {
			return fmt.Errorf("asset %s exceeds max_file_bytes (%d > %d)", logical, info.Size(), m.Assets.MaxFileBytes)
		}
		total += info.Size()
		if total > m.Assets.MaxTotalBytes {
			return fmt.Errorf("asset bundle exceeds max_total_bytes (%d > %d)", total, m.Assets.MaxTotalBytes)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		assets = append(assets, EmbeddedAsset{Name: logical, Path: path, Data: data, Size: info.Size(), SHA256: hex.EncodeToString(sum[:])})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(assets, func(i, j int) bool { return assets[i].Name < assets[j].Name })
	return assets, nil
}

func (m *Manifest) Metadata(zumbraVersion string, assets []EmbeddedAsset) ([]byte, error) {
	public := make([]EmbeddedAsset, len(assets))
	copy(public, assets)
	for i := range public {
		public[i].Path = ""
		public[i].Data = nil
	}
	return json.MarshalIndent(BuildMetadata{SchemaVersion: 1, ZumbraVersion: zumbraVersion, App: m.App, Assets: public}, "", "  ")
}

func parse(data []byte, manifest *Manifest) error {
	section := ""
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	for index, raw := range lines {
		line := strings.TrimSpace(stripComment(raw))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			if section != "app" && section != "build" && section != "assets" {
				return fmt.Errorf("line %d: unsupported section [%s]", index+1, section)
			}
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("line %d: expected key = value", index+1)
		}
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if section == "" {
			return fmt.Errorf("line %d: property outside a section", index+1)
		}
		if err := assign(manifest, section, key, value); err != nil {
			return fmt.Errorf("line %d: %w", index+1, err)
		}
	}
	return nil
}

func assign(m *Manifest, section, key, raw string) error {
	switch section + "." + key {
	case "app.name":
		return setString(raw, &m.App.Name)
	case "app.version":
		return setString(raw, &m.App.Version)
	case "app.identifier":
		return setString(raw, &m.App.Identifier)
	case "app.entry":
		return setString(raw, &m.App.Entry)
	case "app.icon":
		return setString(raw, &m.App.Icon)
	case "build.output":
		return setString(raw, &m.Build.Output)
	case "build.compiler":
		return setString(raw, &m.Build.Compiler)
	case "build.release":
		return setBool(raw, &m.Build.Release)
	case "assets.include":
		values, err := parseStringArray(raw)
		m.Assets.Include = values
		return err
	case "assets.exclude":
		values, err := parseStringArray(raw)
		m.Assets.Exclude = values
		return err
	case "assets.max_file_bytes":
		return setInt(raw, &m.Assets.MaxFileBytes)
	case "assets.max_total_bytes":
		return setInt(raw, &m.Assets.MaxTotalBytes)
	default:
		return fmt.Errorf("unknown property %s.%s", section, key)
	}
}

func setString(raw string, target *string) error {
	value, err := strconv.Unquote(raw)
	if err != nil {
		return fmt.Errorf("expected quoted string")
	}
	*target = value
	return nil
}
func setBool(raw string, target *bool) error {
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fmt.Errorf("expected true or false")
	}
	*target = value
	return nil
}
func setInt(raw string, target *int64) error {
	value, err := strconv.ParseInt(strings.ReplaceAll(raw, "_", ""), 10, 64)
	if err != nil {
		return fmt.Errorf("expected integer")
	}
	*target = value
	return nil
}

func parseStringArray(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if len(raw) < 2 || raw[0] != '[' || raw[len(raw)-1] != ']' {
		return nil, fmt.Errorf("expected string array")
	}
	body := strings.TrimSpace(raw[1 : len(raw)-1])
	if body == "" {
		return []string{}, nil
	}
	result := []string{}
	for len(body) > 0 {
		if body[0] != '"' {
			return nil, fmt.Errorf("array values must be quoted strings")
		}
		end := 1
		escaped := false
		for ; end < len(body); end++ {
			if escaped {
				escaped = false
				continue
			}
			if body[end] == '\\' {
				escaped = true
				continue
			}
			if body[end] == '"' {
				break
			}
		}
		if end >= len(body) {
			return nil, fmt.Errorf("unterminated string")
		}
		value, err := strconv.Unquote(body[:end+1])
		if err != nil {
			return nil, err
		}
		result = append(result, value)
		body = strings.TrimSpace(body[end+1:])
		if body == "" {
			break
		}
		if body[0] != ',' {
			return nil, fmt.Errorf("expected comma between array values")
		}
		body = strings.TrimSpace(body[1:])
	}
	return result, nil
}

func stripComment(line string) string {
	quoted, escaped := false, false
	for index, char := range line {
		if escaped {
			escaped = false
			continue
		}
		if char == '\\' && quoted {
			escaped = true
			continue
		}
		if char == '"' {
			quoted = !quoted
			continue
		}
		if char == '#' && !quoted {
			return line[:index]
		}
	}
	return line
}

func compilePatterns(values []string) ([]*regexp.Regexp, error) {
	result := make([]*regexp.Regexp, 0, len(values))
	for _, value := range values {
		value = strings.TrimPrefix(filepath.ToSlash(strings.TrimSpace(value)), "./")
		if value == "" {
			continue
		}
		var pattern strings.Builder
		pattern.WriteString("^")
		for i := 0; i < len(value); i++ {
			switch value[i] {
			case '*':
				if i+1 < len(value) && value[i+1] == '*' {
					pattern.WriteString(".*")
					i++
				} else {
					pattern.WriteString("[^/]*")
				}
			case '?':
				pattern.WriteString("[^/]")
			default:
				pattern.WriteString(regexp.QuoteMeta(string(value[i])))
			}
		}
		if strings.HasSuffix(value, "/") {
			pattern.WriteString(".*")
		}
		pattern.WriteString("$")
		compiled, err := regexp.Compile(pattern.String())
		if err != nil {
			return nil, err
		}
		result = append(result, compiled)
	}
	return result, nil
}
func matchesAny(patterns []*regexp.Regexp, value string) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(value) {
			return true
		}
	}
	return false
}
func slug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	re := regexp.MustCompile(`[^a-z0-9]+`)
	value = strings.Trim(re.ReplaceAllString(value, "-"), "-")
	if value == "" {
		return "zumbra-app"
	}
	return value
}
