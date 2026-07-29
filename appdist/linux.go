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
			c.result.Tools["appimagetool"] = tool
			output := filepath.Join(c.outDir, base+".AppImage")
			args := []string{}
			runtimePath := strings.TrimSpace(c.options.AppImageRuntime)
			if runtimePath != "" {
				args = append(args, "--runtime-file", runtimePath)
				c.result.Tools["appimage_runtime"] = runtimePath
			} else {
				c.result.Warnings = append(c.result.Warnings, "no pinned AppImage runtime was found; appimagetool may use network once and Zumbra will cache the generated runtime")
			}
			args = append(args, appDir, output)
			cmd := exec.Command(tool, args...)
			cmd.Env = append(os.Environ(), "ARCH="+appImageArch(c.options.Arch), fmt.Sprintf("SOURCE_DATE_EPOCH=%d", c.epoch.Unix()))
			cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("appimagetool failed: %w", err)
			}
			if runtimePath == "" {
				if cached, cacheErr := cacheAppImageRuntime(output, c.options.Arch); cacheErr == nil {
					c.result.Tools["appimage_runtime"] = cached
					c.result.Warnings = append(c.result.Warnings, "cached AppImage runtime for future offline builds at "+cached)
				} else {
					c.result.Warnings = append(c.result.Warnings, "could not cache generated AppImage runtime: "+cacheErr.Error())
				}
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
	runtimeFiles, err := c.runtimeFiles("linux")
	if err != nil {
		return err
	}
	installedBinary := filepath.Join(binDir, slug)
	if len(runtimeFiles) == 0 {
		if err := copyFile(c.options.Binary, installedBinary, 0o755, c.epoch); err != nil {
			return err
		}
	} else {
		libDir := filepath.Join(prefix, "lib", slug)
		installedBinary = filepath.Join(libDir, slug+".bin")
		if err := copyFile(c.options.Binary, installedBinary, 0o755, c.epoch); err != nil {
			return err
		}
		if err := c.copyRuntimeFiles("linux", libDir); err != nil {
			return err
		}
		launcher := "#!/bin/sh\nset -eu\nBIN_DIR=$(CDPATH= cd -- \"$(dirname -- \"$0\")\" && pwd)\nPREFIX=$(CDPATH= cd -- \"$BIN_DIR/..\" && pwd)\nAPP_LIB=\"$PREFIX/lib/" + slug + "\"\nexport LD_LIBRARY_PATH=\"$APP_LIB${LD_LIBRARY_PATH:+:$LD_LIBRARY_PATH}\"\nexec \"$APP_LIB/" + slug + ".bin\" \"$@\"\n"
		if err := writeFile(filepath.Join(binDir, slug), []byte(launcher), 0o755, c.epoch); err != nil {
			return err
		}
	}
	desktop := c.linuxDesktopEntry()
	if err := writeFile(filepath.Join(shareDir, "applications", c.linuxDesktopFileName()), []byte(desktop), 0o644, c.epoch); err != nil {
		return err
	}
	if err := writeFile(filepath.Join(shareDir, "metainfo", c.appStreamFileName()), []byte(c.linuxAppStream()), 0o644, c.epoch); err != nil {
		return err
	}
	if icon := c.options.Manifest.IconPathForTarget("linux"); icon != "" {
		ext := strings.ToLower(filepath.Ext(icon))
		if ext == "" {
			ext = ".png"
		}
		if err := copyFile(icon, filepath.Join(shareDir, "icons", "hicolor", "256x256", "apps", c.linuxIconName()+ext), 0o644, c.epoch); err != nil {
			return err
		}
	}
	metaDir := filepath.Join(shareDir, "zumbra", slug)
	if err := c.writeMetadata(filepath.Join(metaDir, "package.json")); err != nil {
		return err
	}
	c.auditDependencies(installedBinary, filepath.Join(metaDir, "dependencies.txt"))
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
	return fmt.Sprintf("[Desktop Entry]\nType=Application\nName=%s\nComment=%s\nExec=%s\nIcon=%s\nTerminal=false\nCategories=%s;\nStartupNotify=true\n", escapeDesktop(m.App.Name), escapeDesktop(m.Package.Description), m.Slug(), c.linuxIconName(), category)
}

func (c *packageContext) linuxDesktopFileName() string {
	return c.options.Manifest.App.Identifier + ".desktop"
}

func (c *packageContext) linuxIconName() string {
	return c.options.Manifest.App.Identifier
}

func (c *packageContext) appStreamFileName() string {
	return c.options.Manifest.App.Identifier + ".appdata.xml"
}

func (c *packageContext) linuxAppStream() string {
	m := c.options.Manifest
	description := firstNonEmpty(m.Package.Description, m.App.Name)
	paragraph := appStreamDescription(m.App.Name, description)
	summary := strings.TrimSpace(description)
	summary = strings.TrimRight(summary, ".!?")
	if summary == "" {
		summary = m.App.Name
	}
	homepage := ""
	if strings.TrimSpace(m.Package.Homepage) != "" {
		homepage = fmt.Sprintf("  <url type=\"homepage\">%s</url>\n", xmlEscape(m.Package.Homepage))
	}
	developer := ""
	if publisher := strings.TrimSpace(m.Package.Publisher); publisher != "" {
		developer = fmt.Sprintf("  <developer id=\"%s\"><name>%s</name></developer>\n", xmlEscape(m.App.Identifier), xmlEscape(publisher))
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<component type="desktop-application">
  <id>%s</id>
  <name>%s</name>
  <summary>%s</summary>
  <metadata_license>CC0-1.0</metadata_license>
  <project_license>%s</project_license>
  <description>
    <p>%s</p>
  </description>
  <launchable type="desktop-id">%s</launchable>
%s%s  <content_rating type="oars-1.1"/>
  <releases><release version="%s" date="%s"/></releases>
</component>
`, xmlEscape(m.App.Identifier), xmlEscape(m.App.Name), xmlEscape(summary), xmlEscape(firstNonEmpty(m.Package.License, "LicenseRef-proprietary")), xmlEscape(paragraph), c.linuxDesktopFileName(), developer, homepage, xmlEscape(m.App.Version), c.epoch.Format("2006-01-02"))
}

func appStreamDescription(appName, description string) string {
	description = strings.TrimSpace(description)
	if description == "" {
		description = appName + " is a desktop application built with Zumbra."
	}
	paragraph := fmt.Sprintf("%s provides a native desktop experience packaged with the metadata, application assets, and runtime integration required for installation and portable distribution.", description)
	if len([]rune(paragraph)) < 100 {
		paragraph += " It is designed to run consistently across supported desktop environments."
	}
	return paragraph
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
	appRun := "#!/bin/sh\nset -eu\nHERE=$(CDPATH= cd -- \"$(dirname -- \"$0\")\" && pwd)\nexport LD_LIBRARY_PATH=\"$HERE/usr/lib/" + slug + "${LD_LIBRARY_PATH:+:$LD_LIBRARY_PATH}\"\nexec \"$HERE/usr/bin/" + slug + "\" \"$@\"\n"
	if err := writeFile(filepath.Join(root, "AppRun"), []byte(appRun), 0o755, c.epoch); err != nil {
		return err
	}
	if err := writeFile(filepath.Join(root, c.linuxDesktopFileName()), []byte(c.linuxDesktopEntry()), 0o644, c.epoch); err != nil {
		return err
	}
	if icon := c.options.Manifest.IconPathForTarget("linux"); icon != "" {
		ext := strings.ToLower(filepath.Ext(icon))
		if ext == "" {
			ext = ".png"
		}
		if err := copyFile(icon, filepath.Join(root, c.linuxIconName()+ext), 0o644, c.epoch); err != nil {
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
