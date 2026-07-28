package appdist

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (c *packageContext) packageMacOS() error {
	base := c.baseName()
	appPath := filepath.Join(c.outDir, c.options.Manifest.App.Name+".app")
	_ = os.RemoveAll(appPath)
	if err := c.populateMacApp(appPath); err != nil {
		return err
	}
	if strings.TrimSpace(c.options.SignIdentity) != "" {
		if err := c.signMacApp(appPath); err != nil {
			return err
		}
	}
	if c.wants("app") {
		if err := c.addArtifact("macos-app", appPath); err != nil {
			return err
		}
	}
	if c.wants("zip") {
		output := filepath.Join(c.outDir, base+".zip")
		if err := writeZip(appPath, output, filepath.Base(appPath), c.epoch); err != nil {
			return err
		}
		if err := c.addArtifact("macos-zip", output); err != nil {
			return err
		}
	}
	return nil
}

func (c *packageContext) populateMacApp(appPath string) error {
	m := c.options.Manifest
	slug := m.Slug()
	contents := filepath.Join(appPath, "Contents")
	macOSDir := filepath.Join(contents, "MacOS")
	resources := filepath.Join(contents, "Resources")
	if err := copyFile(c.options.Binary, filepath.Join(macOSDir, slug), 0o755, c.epoch); err != nil {
		return err
	}
	iconName := ""
	if icon := m.IconPathForTarget("macos"); icon != "" {
		ext := filepath.Ext(icon)
		if ext == "" {
			ext = ".icns"
		}
		iconName = slug + ext
		if err := copyFile(icon, filepath.Join(resources, iconName), 0o644, c.epoch); err != nil {
			return err
		}
	}
	if err := c.writeMetadata(filepath.Join(resources, "package.json")); err != nil {
		return err
	}
	c.auditDependencies(c.options.Binary, filepath.Join(resources, "dependencies.txt"))
	plist := c.infoPlist(iconName)
	if err := writeFile(filepath.Join(contents, "Info.plist"), []byte(plist), 0o644, c.epoch); err != nil {
		return err
	}
	return writeFile(filepath.Join(contents, "PkgInfo"), []byte("APPL????"), 0o644, c.epoch)
}

func (c *packageContext) infoPlist(iconName string) string {
	m := c.options.Manifest
	category := firstNonEmpty(m.MacOS.Category, "public.app-category.utilities")
	icon := ""
	if iconName != "" {
		icon = "\n\t<key>CFBundleIconFile</key>\n\t<string>" + xmlEscape(iconName) + "</string>"
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleDisplayName</key><string>%s</string>
	<key>CFBundleExecutable</key><string>%s</string>
	<key>CFBundleIdentifier</key><string>%s</string>
	<key>CFBundleName</key><string>%s</string>
	<key>CFBundlePackageType</key><string>APPL</string>
	<key>CFBundleShortVersionString</key><string>%s</string>
	<key>CFBundleVersion</key><string>%s</string>
	<key>LSMinimumSystemVersion</key><string>%s</string>
	<key>LSApplicationCategoryType</key><string>%s</string>
	<key>NSHighResolutionCapable</key><true/>%s
</dict>
</plist>
`, xmlEscape(m.App.Name), m.Slug(), xmlEscape(m.App.Identifier), xmlEscape(m.App.Name), xmlEscape(m.App.Version), xmlEscape(m.App.Version), xmlEscape(m.MacOS.MinimumVersion), xmlEscape(category), icon)
}

func xmlEscape(value string) string {
	var out strings.Builder
	_ = xml.EscapeText(&out, []byte(value))
	return out.String()
}
