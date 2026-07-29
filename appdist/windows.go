package appdist

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func (c *packageContext) packageWindows() error {
	base := c.baseName()
	portable := filepath.Join(c.outDir, base+"-portable")
	_ = os.RemoveAll(portable)
	if err := c.populateWindowsPortable(portable); err != nil {
		return err
	}
	if c.wants("portable") {
		output := filepath.Join(c.outDir, base+"-portable.zip")
		if err := writeZip(portable, output, base, c.epoch); err != nil {
			return err
		}
		if err := c.addArtifact("windows-portable", output); err != nil {
			return err
		}
	}
	if c.wants("installer") && c.options.Manifest.Windows.Installer != "none" {
		script := filepath.Join(c.outDir, base+"-installer.nsi")
		installer := filepath.Join(c.outDir, base+"-setup.exe")
		if err := c.writeNSISScript(script, portable, installer); err != nil {
			return err
		}
		if err := c.addArtifact("nsis-script", script); err != nil {
			return err
		}
		tool, findErr := FindNSISTool(c.options.NSISTool)
		if findErr != nil {
			return fmt.Errorf("Windows installer requested but makensis is unavailable: %s", NSISInstallHint())
		}
		cmd := exec.Command(tool, script)
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("makensis failed: %w", err)
		}
		if _, err := os.Stat(installer); err != nil {
			return fmt.Errorf("makensis did not create %s", installer)
		}
		if err := os.Chtimes(installer, c.epoch, c.epoch); err != nil {
			return err
		}
		if err := c.addArtifact("windows-installer", installer); err != nil {
			return err
		}
	}
	return nil
}

func (c *packageContext) populateWindowsPortable(root string) error {
	slug := c.options.Manifest.Slug()
	executable := slug + ".exe"
	if err := copyFile(c.options.Binary, filepath.Join(root, executable), 0o755, c.epoch); err != nil {
		return err
	}
	if err := c.writeMetadata(filepath.Join(root, "package.json")); err != nil {
		return err
	}
	c.auditDependencies(c.options.Binary, filepath.Join(root, "dependencies.txt"))
	if icon := c.options.Manifest.IconPathForTarget("windows"); icon != "" {
		ext := strings.ToLower(filepath.Ext(icon))
		if ext == "" {
			ext = ".ico"
		}
		if err := copyFile(icon, filepath.Join(root, slug+ext), 0o644, c.epoch); err != nil {
			return err
		}
	}
	launcher := "@echo off\r\nsetlocal\r\n\"%~dp0" + executable + "\" %*\r\n"
	return writeFile(filepath.Join(root, "run.cmd"), []byte(launcher), 0o644, c.epoch)
}

func (c *packageContext) writeNSISScript(path, portable, installer string) error {
	m := c.options.Manifest
	slug := m.Slug()
	exe := slug + ".exe"
	var iconLine string
	if icon := m.IconPathForTarget("windows"); icon != "" {
		if strings.EqualFold(filepath.Ext(icon), ".ico") {
			iconLine = fmt.Sprintf("Icon \"%s\"\nUninstallIcon \"%s\"\n", nsisPath(icon), nsisPath(icon))
		}
	}
	script := fmt.Sprintf(`Unicode true
Name "%s"
OutFile "%s"
InstallDir "$PROGRAMFILES64\%s"
RequestExecutionLevel admin
%s
Page directory
Page instfiles
UninstPage uninstConfirm
UninstPage instfiles

Section "Install"
  SetOutPath "$INSTDIR"
  File /r "%s\*"
  CreateDirectory "$SMPROGRAMS\%s"
  CreateShortcut "$SMPROGRAMS\%s\%s.lnk" "$INSTDIR\%s"
  CreateShortcut "$DESKTOP\%s.lnk" "$INSTDIR\%s"
  WriteUninstaller "$INSTDIR\uninstall.exe"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\%s" "DisplayName" "%s"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\%s" "DisplayVersion" "%s"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\%s" "Publisher" "%s"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\%s" "UninstallString" "$INSTDIR\uninstall.exe"
SectionEnd

Section "Uninstall"
  Delete "$DESKTOP\%s.lnk"
  RMDir /r "$SMPROGRAMS\%s"
  RMDir /r "$INSTDIR"
  DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\%s"
SectionEnd
`, nsisEscape(m.App.Name), filepath.ToSlash(installer), nsisEscape(m.App.Name), iconLine, filepath.ToSlash(portable), nsisEscape(m.App.Name), nsisEscape(m.App.Name), nsisEscape(m.App.Name), exe, nsisEscape(m.App.Name), exe, m.App.Identifier, nsisEscape(m.App.Name), m.App.Identifier, nsisEscape(m.App.Version), m.App.Identifier, nsisEscape(m.Package.Publisher), m.App.Identifier, nsisEscape(m.App.Name), nsisEscape(m.App.Name), m.App.Identifier)
	return writeFile(path, []byte(script), 0o644, c.epoch)
}

func nsisEscape(value string) string {
	value = strings.ReplaceAll(value, "$", "$$")
	value = strings.ReplaceAll(value, "\"", "$\\\"")
	value = strings.ReplaceAll(value, "\r", "")
	return strings.ReplaceAll(value, "\n", " ")
}
func nsisPath(value string) string { return strings.ReplaceAll(value, "/", "\\") }
