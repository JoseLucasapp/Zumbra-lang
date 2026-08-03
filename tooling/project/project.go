// Package project manages reproducible Zumbra project layouts and manifests.
package project

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"zumbra/pipeline"
	"zumbra/tooling/formatter"
)

const ManifestName = "zumbra.toml"

type Kind string

const (
	KindCLI     Kind = "cli"
	KindLibrary Kind = "library"
	KindDesktop Kind = "desktop"
)

type Manifest struct {
	Path       string   `json:"path"`
	Root       string   `json:"root"`
	Name       string   `json:"name"`
	Version    string   `json:"version"`
	Kind       Kind     `json:"kind"`
	Entry      string   `json:"entry"`
	SourceDirs []string `json:"source_dirs"`
	TestDirs   []string `json:"test_dirs"`
	DocsOutput string   `json:"docs_output"`
}

type InitOptions struct {
	Directory  string
	Name       string
	Kind       Kind
	Identifier string
	Force      bool
}

type CheckResult struct {
	Files       int                   `json:"files"`
	Diagnostics []pipeline.Diagnostic `json:"diagnostics"`
}

func Init(options InitOptions) (*Manifest, []string, error) {
	name := strings.TrimSpace(options.Name)
	if name == "" {
		return nil, nil, fmt.Errorf("project name is required")
	}
	projectSlug := slug(name)
	if projectSlug == "" {
		return nil, nil, fmt.Errorf("project name must contain at least one letter or number")
	}
	kind := options.Kind
	if kind == "" {
		kind = KindCLI
	}
	if kind != KindCLI && kind != KindLibrary && kind != KindDesktop {
		return nil, nil, fmt.Errorf("unknown project kind %q; use cli, library or desktop", kind)
	}
	directory := options.Directory
	if directory == "" {
		directory = projectSlug
	}
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return nil, nil, err
	}
	if entries, readErr := os.ReadDir(absolute); readErr == nil && len(entries) > 0 && !options.Force {
		return nil, nil, fmt.Errorf("directory %s is not empty; use --force to add missing project files", absolute)
	} else if readErr != nil && !os.IsNotExist(readErr) {
		return nil, nil, readErr
	}
	if err := os.MkdirAll(absolute, 0o755); err != nil {
		return nil, nil, err
	}

	entry := "src/main.zum"
	if kind == KindLibrary {
		entry = "src/lib.zum"
	}
	identifier := strings.TrimSpace(options.Identifier)
	if identifier == "" {
		identifier = "dev.zumbra." + projectSlug
	}
	if kind == KindDesktop && !validIdentifier(identifier) {
		return nil, nil, fmt.Errorf("desktop identifier %q must be a reverse-DNS name such as dev.example.app", identifier)
	}
	manifestText := projectManifest(name, kind, entry)
	if kind == KindDesktop {
		manifestText = desktopManifest(name, identifier, entry)
	}
	created := []string{}
	files := map[string]string{
		ManifestName: manifestText,
		".gitignore": "build/\ndist/\n.zumbra/\n*.prof\n",
		"README.md":  "# " + name + "\n\nCreated with Zumbra.\n",
	}
	if kind == KindLibrary {
		files[entry] = "/// Returns the library name.\npub fct name() {\n    return \"" + escapeString(name) + "\";\n}\n"
		files["tests/library_test.zum"] = "import \"../src/lib.zum\" as library;\nshow(library.name());\n"
	} else {
		files[entry] = "fct main() {\n    show(\"Hello from " + escapeString(name) + "!\");\n}\n\nmain();\n"
		files["tests/smoke_test.zum"] = "show(\"smoke ok\");\n"
	}
	for relative, content := range files {
		target := filepath.Join(absolute, filepath.FromSlash(relative))
		if _, statErr := os.Stat(target); statErr == nil && !options.Force {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return nil, created, err
		}
		if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
			return nil, created, err
		}
		created = append(created, filepath.ToSlash(target))
	}
	sort.Strings(created)
	manifest, err := Load(filepath.Join(absolute, ManifestName))
	return manifest, created, err
}

func Find(start string) (string, error) {
	if start == "" {
		start = "."
	}
	absolute, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(absolute)
	if err == nil && !info.IsDir() {
		absolute = filepath.Dir(absolute)
	}
	for {
		candidate := filepath.Join(absolute, ManifestName)
		if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(absolute)
		if parent == absolute {
			return "", fmt.Errorf("%s not found from %s", ManifestName, start)
		}
		absolute = parent
	}
}

func Load(path string) (*Manifest, error) {
	if path == "" {
		found, err := Find(".")
		if err != nil {
			return nil, err
		}
		path = found
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(absolute)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	manifest := &Manifest{
		Path:       absolute,
		Root:       filepath.Dir(absolute),
		Version:    "0.1.0",
		Kind:       KindCLI,
		Entry:      "src/main.zum",
		SourceDirs: []string{"src"},
		TestDirs:   []string{"tests"},
		DocsOutput: "docs/api.md",
	}
	section := ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(stripComment(scanner.Text()))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			continue
		}
		key, raw, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		raw = strings.TrimSpace(raw)
		switch section + "." + key {
		case "project.name", "app.name":
			manifest.Name = unquote(raw)
		case "project.version", "app.version":
			manifest.Version = unquote(raw)
		case "project.kind":
			manifest.Kind = Kind(unquote(raw))
		case "project.entry", "app.entry":
			manifest.Entry = unquote(raw)
		case "tooling.source_dirs":
			manifest.SourceDirs = parseArray(raw)
		case "tooling.test_dirs":
			manifest.TestDirs = parseArray(raw)
		case "tooling.docs_output":
			manifest.DocsOutput = unquote(raw)
		}
		if section == "app" {
			manifest.Kind = KindDesktop
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(manifest.Name) == "" {
		return nil, fmt.Errorf("manifest %s requires [project].name or [app].name", absolute)
	}
	if manifest.Kind != KindCLI && manifest.Kind != KindLibrary && manifest.Kind != KindDesktop {
		return nil, fmt.Errorf("manifest project kind %q is invalid", manifest.Kind)
	}
	if !regexp.MustCompile(`^\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?$`).MatchString(manifest.Version) {
		return nil, fmt.Errorf("manifest version %q is not semantic versioning", manifest.Version)
	}
	entryPath := filepath.Join(manifest.Root, filepath.FromSlash(manifest.Entry))
	if info, err := os.Stat(entryPath); err != nil || info.IsDir() {
		if err == nil {
			err = fmt.Errorf("entry is a directory")
		}
		return nil, fmt.Errorf("project entry %s: %w", entryPath, err)
	}
	return manifest, nil
}

func (m *Manifest) EntryPath() string {
	return filepath.Join(m.Root, filepath.FromSlash(m.Entry))
}

func (m *Manifest) TestFiles() ([]string, error) {
	paths := make([]string, 0, len(m.TestDirs))
	for _, root := range m.TestDirs {
		candidate := filepath.Join(m.Root, filepath.FromSlash(root))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			continue
		}
		paths = append(paths, candidate)
	}
	files, err := formatter.Discover(paths)
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func (m *Manifest) SourceFiles(includeTests bool) ([]string, error) {
	roots := append([]string{}, m.SourceDirs...)
	entryDir := filepath.Dir(m.Entry)
	if entryDir != "." && !contains(roots, entryDir) {
		roots = append(roots, entryDir)
	}
	if includeTests {
		roots = append(roots, m.TestDirs...)
	}
	paths := make([]string, 0, len(roots))
	for _, root := range roots {
		candidate := filepath.Join(m.Root, filepath.FromSlash(root))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			continue
		}
		paths = append(paths, candidate)
	}
	files, err := formatter.Discover(paths)
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func (m *Manifest) Check(includeTests bool) CheckResult {
	files, err := m.SourceFiles(includeTests)
	if err != nil {
		return CheckResult{Diagnostics: []pipeline.Diagnostic{{Stage: pipeline.StageParser, Message: err.Error(), Code: "ZP1000", File: m.Path}}}
	}
	result := CheckResult{Files: len(files)}
	for _, filename := range files {
		built, items := pipeline.BuildFile(filename, pipeline.Options{Optimize: true})
		result.Diagnostics = append(result.Diagnostics, items...)
		if built != nil {
			result.Diagnostics = append(result.Diagnostics, built.Warnings...)
		}
	}
	return result
}

func (m *Manifest) Clean() ([]string, error) {
	removed := []string{}
	for _, relative := range []string{"build", "dist", ".zumbra"} {
		target := filepath.Join(m.Root, relative)
		if _, err := os.Stat(target); os.IsNotExist(err) {
			continue
		}
		if err := os.RemoveAll(target); err != nil {
			return removed, err
		}
		removed = append(removed, filepath.ToSlash(target))
	}
	return removed, nil
}

func projectManifest(name string, kind Kind, entry string) string {
	return fmt.Sprintf(`[project]
name = %q
version = "0.1.0"
kind = %q
entry = %q

[tooling]
source_dirs = ["src"]
test_dirs = ["tests"]
docs_output = "docs/api.md"
`, name, string(kind), entry)
}

func desktopManifest(name, identifier, entry string) string {
	return fmt.Sprintf(`[app]
name = %q
version = "0.1.0"
identifier = %q
entry = %q

[build]
compiler = "auto"
release = true

[package]
description = %q
category = "Utility"
`, name, identifier, entry, name+" built with Zumbra")
}

func stripComment(line string) string {
	inString := false
	escaped := false
	for index := 0; index < len(line); index++ {
		switch line[index] {
		case '\\':
			escaped = !escaped
		case '"':
			if !escaped {
				inString = !inString
			}
			escaped = false
		case '#':
			if !inString {
				return line[:index]
			}
		default:
			escaped = false
		}
	}
	return line
}

func unquote(value string) string {
	value = strings.TrimSpace(value)
	if parsed, err := strconv.Unquote(value); err == nil {
		return parsed
	}
	return value
}

func parseArray(value string) []string {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "[") || !strings.HasSuffix(value, "]") {
		return nil
	}
	value = strings.TrimSpace(value[1 : len(value)-1])
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		item := unquote(strings.TrimSpace(part))
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

func slug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var output strings.Builder
	dash := false
	for _, character := range value {
		if (character >= 'a' && character <= 'z') || (character >= '0' && character <= '9') {
			output.WriteRune(character)
			dash = false
		} else if output.Len() > 0 && !dash {
			output.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(output.String(), "-")
}

func validIdentifier(value string) bool {
	return regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*(?:\.[A-Za-z][A-Za-z0-9_-]*)+$`).MatchString(value)
}

func escapeString(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "\\", "\\\\"), "\"", "\\\"")
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if filepath.Clean(value) == filepath.Clean(target) {
			return true
		}
	}
	return false
}
