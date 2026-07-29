package appdist

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func (c *packageContext) packageLinux() error {
	base := c.baseName()
	stage := filepath.Join(c.outDir, base+"-bundle")
	_ = os.RemoveAll(stage)
	if err := c.populateLinuxTree(stage, false); err != nil {
		return err
	}
	if c.wants("bundle") {
		output := filepath.Join(c.outDir, base+".tar.gz")
		if err := writeTarGz(stage, output, base, c.epoch); err != nil {
			return err
		}
		if err := c.addArtifact("linux-bundle", output); err != nil {
			return err
		}
	}
	if c.wants("deb") {
		output := filepath.Join(c.outDir, debFileName(c.options.Manifest.Slug(), c.options.Manifest.App.Version, c.options.Arch))
		if err := c.writeDeb(output); err != nil {
			return err
		}
		if err := c.addArtifact("deb", output); err != nil {
			return err
		}
	}
	if c.wants("appdir") || c.wants("appimage") {
		appDir := filepath.Join(c.outDir, base+".AppDir")
		_ = os.RemoveAll(appDir)
		if err := c.populateAppDir(appDir); err != nil {
			return err
		}
		if err := c.addArtifact("appdir", appDir); err != nil {
			return err
		}
		if c.wants("appimage") {
			tool, findErr := FindAppImageTool(c.options.AppImageTool, c.options.Manifest.Root, c.options.Arch)
			if findErr != nil {
				return fmt.Errorf("AppImage requested but appimagetool is unavailable: %s", AppImageInstallHint(c.options.Arch))
			}
			output := filepath.Join(c.outDir, base+".AppImage")
			cmd := exec.Command(tool, appDir, output)
			cmd.Env = append(os.Environ(), "ARCH="+appImageArch(c.options.Arch), fmt.Sprintf("SOURCE_DATE_EPOCH=%d", c.epoch.Unix()))
			cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("appimagetool failed: %w", err)
			}
			if err := os.Chtimes(output, c.epoch, c.epoch); err != nil {
				return err
			}
			if err := c.addArtifact("appimage", output); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *packageContext) populateLinuxTree(root string, debLayout bool) error {
	slug := c.options.Manifest.Slug()
	prefix := root
	if debLayout {
		prefix = filepath.Join(root, "usr")
	}
	binDir := filepath.Join(prefix, "bin")
	shareDir := filepath.Join(prefix, "share")
	if !debLayout {
		binDir = filepath.Join(root, "bin")
		shareDir = filepath.Join(root, "share")
	}
	if err := copyFile(c.options.Binary, filepath.Join(binDir, slug), 0o755, c.epoch); err != nil {
		return err
	}
	desktop := c.linuxDesktopEntry()
	if err := writeFile(filepath.Join(shareDir, "applications", slug+".desktop"), []byte(desktop), 0o644, c.epoch); err != nil {
		return err
	}
	if icon := c.options.Manifest.IconPathForTarget("linux"); icon != "" {
		ext := strings.ToLower(filepath.Ext(icon))
		if ext == "" {
			ext = ".png"
		}
		if err := copyFile(icon, filepath.Join(shareDir, "icons", "hicolor", "256x256", "apps", slug+ext), 0o644, c.epoch); err != nil {
			return err
		}
	}
	metaDir := filepath.Join(shareDir, "zumbra", slug)
	if err := c.writeMetadata(filepath.Join(metaDir, "package.json")); err != nil {
		return err
	}
	c.auditDependencies(c.options.Binary, filepath.Join(metaDir, "dependencies.txt"))
	if license := strings.TrimSpace(c.options.Manifest.Package.License); license != "" {
		_ = writeFile(filepath.Join(shareDir, "doc", slug, "license.txt"), []byte(license+"\n"), 0o644, c.epoch)
	}
	return nil
}

func (c *packageContext) linuxDesktopEntry() string {
	m := c.options.Manifest
	category := strings.TrimSpace(m.Package.Category)
	if category == "" {
		category = "Utility"
	}
	return fmt.Sprintf("[Desktop Entry]\nType=Application\nName=%s\nComment=%s\nExec=%s\nIcon=%s\nTerminal=false\nCategories=%s;\nStartupNotify=true\n", escapeDesktop(m.App.Name), escapeDesktop(m.Package.Description), m.Slug(), m.Slug(), category)
}

func escapeDesktop(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\n", "\\n")
	return value
}

func (c *packageContext) populateAppDir(root string) error {
	if err := c.populateLinuxTree(root, true); err != nil {
		return err
	}
	slug := c.options.Manifest.Slug()
	appRun := "#!/bin/sh\nset -eu\nHERE=$(CDPATH= cd -- \"$(dirname -- \"$0\")\" && pwd)\nexec \"$HERE/usr/bin/" + slug + "\" \"$@\"\n"
	if err := writeFile(filepath.Join(root, "AppRun"), []byte(appRun), 0o755, c.epoch); err != nil {
		return err
	}
	if err := writeFile(filepath.Join(root, slug+".desktop"), []byte(c.linuxDesktopEntry()), 0o644, c.epoch); err != nil {
		return err
	}
	if icon := c.options.Manifest.IconPathForTarget("linux"); icon != "" {
		ext := strings.ToLower(filepath.Ext(icon))
		if ext == "" {
			ext = ".png"
		}
		if err := copyFile(icon, filepath.Join(root, slug+ext), 0o644, c.epoch); err != nil {
			return err
		}
	}
	return nil
}

func (c *packageContext) writeDeb(output string) error {
	root, err := os.MkdirTemp("", "zumbra-deb-data-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(root)
	if err := c.populateLinuxTree(root, true); err != nil {
		return err
	}
	controlRoot, err := os.MkdirTemp("", "zumbra-deb-control-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(controlRoot)
	m := c.options.Manifest
	maintainer := strings.TrimSpace(m.Package.Publisher)
	if maintainer == "" {
		maintainer = m.App.Name
	}
	depends := append([]string{}, m.Linux.Dependencies...)
	if len(depends) == 0 {
		depends = []string{"libc6"}
	}
	var control bytes.Buffer
	fmt.Fprintf(&control, "Package: %s\nVersion: %s\nArchitecture: %s\nMaintainer: %s\nDescription: %s\nSection: utils\nPriority: optional\nDepends: %s\n", m.Slug(), m.App.Version, debArch(c.options.Arch), maintainer, firstNonEmpty(m.Package.Description, m.App.Name), strings.Join(depends, ", "))
	if len(m.Linux.Recommends) > 0 {
		fmt.Fprintf(&control, "Recommends: %s\n", strings.Join(m.Linux.Recommends, ", "))
	}
	if m.Package.Homepage != "" {
		fmt.Fprintf(&control, "Homepage: %s\n", m.Package.Homepage)
	}
	if err := writeFile(filepath.Join(controlRoot, "control"), control.Bytes(), 0o644, c.epoch); err != nil {
		return err
	}
	controlTar := filepath.Join(filepath.Dir(output), ".control.tar.gz")
	dataTar := filepath.Join(filepath.Dir(output), ".data.tar.gz")
	defer os.Remove(controlTar)
	defer os.Remove(dataTar)
	if err := writeTarGz(controlRoot, controlTar, "", c.epoch); err != nil {
		return err
	}
	if err := writeTarGz(root, dataTar, "", c.epoch); err != nil {
		return err
	}
	return writeDebArchive(output, controlTar, dataTar, c.epoch)
}

func debFileName(slug, version, arch string) string {
	return fmt.Sprintf("%s_%s_%s.deb", slug, version, debArch(arch))
}
func debArch(arch string) string {
	if arch == "arm64" {
		return "arm64"
	}
	return "amd64"
}
func appImageArch(arch string) string {
	if arch == "arm64" {
		return "aarch64"
	}
	return "x86_64"
}
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return "Zumbra application"
}
