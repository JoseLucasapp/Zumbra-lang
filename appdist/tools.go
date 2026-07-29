package appdist

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func FindAppImageTool(explicit, projectRoot, arch string) (string, error) {
	names := []string{}
	if value := strings.TrimSpace(explicit); value != "" {
		names = append(names, value)
	}
	if value := strings.TrimSpace(os.Getenv("APPIMAGETOOL")); value != "" {
		names = append(names, value)
	}
	appImageName := "appimagetool-" + appImageArch(arch) + ".AppImage"
	if strings.TrimSpace(projectRoot) != "" {
		names = append(names,
			filepath.Join(projectRoot, "tools", "appimagetool"),
			filepath.Join(projectRoot, "tools", appImageName),
		)
	}
	if cache, err := os.UserCacheDir(); err == nil && cache != "" {
		names = append(names,
			filepath.Join(cache, "zumbra", "tools", "appimagetool"),
			filepath.Join(cache, "zumbra", "tools", appImageName),
		)
	}
	names = append(names, "appimagetool", appImageName)
	return findExecutable(names, "appimagetool")
}

func FindNSISTool(explicit string) (string, error) {
	names := []string{}
	if value := strings.TrimSpace(explicit); value != "" {
		names = append(names, value)
	}
	if value := strings.TrimSpace(os.Getenv("MAKENSIS")); value != "" {
		names = append(names, value)
	}
	names = append(names, "makensis", "makensis.exe")
	return findExecutable(names, "makensis")
}

func FindCodeSignTool() (string, error) {
	if value := strings.TrimSpace(os.Getenv("CODESIGN")); value != "" {
		return findExecutable([]string{value}, "codesign")
	}
	return findExecutable([]string{"codesign"}, "codesign")
}

func findExecutable(candidates []string, label string) (string, error) {
	seen := map[string]bool{}
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" || seen[candidate] {
			continue
		}
		seen[candidate] = true
		if strings.ContainsRune(candidate, filepath.Separator) || filepath.IsAbs(candidate) {
			absolute, err := filepath.Abs(candidate)
			if err != nil {
				continue
			}
			info, err := os.Stat(absolute)
			if err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
				return absolute, nil
			}
			continue
		}
		if path, err := exec.LookPath(candidate); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("%s was not found", label)
}

func AppImageInstallHint(arch string) string {
	appArch := appImageArch(arch)
	return fmt.Sprintf("install appimagetool in PATH, set APPIMAGETOOL, or place appimagetool-%s.AppImage in tools/ or the Zumbra tool cache", appArch)
}

func NSISInstallHint() string {
	switch runtime.GOOS {
	case "linux":
		return "install NSIS (for Debian/Ubuntu: sudo apt install nsis) or set MAKENSIS"
	case "windows":
		return "install NSIS and add makensis.exe to PATH, or set MAKENSIS"
	default:
		return "install NSIS and add makensis to PATH, or set MAKENSIS"
	}
}
