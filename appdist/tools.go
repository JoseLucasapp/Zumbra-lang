package appdist

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// AppImageToolCandidates returns the ordered locations inspected for
// appimagetool. Both the manifest root and the caller's working directory are
// considered so monorepos and examples behave consistently.
func AppImageToolCandidates(explicit, projectRoot, arch string) []string {
	if value := strings.TrimSpace(explicit); value != "" {
		return []string{value}
	}
	if value := strings.TrimSpace(os.Getenv("APPIMAGETOOL")); value != "" {
		return []string{value}
	}
	appImageName := "appimagetool-" + appImageArch(arch) + ".AppImage"
	candidates := []string{}
	for _, root := range toolRoots(projectRoot) {
		candidates = append(candidates,
			filepath.Join(root, "tools", "appimagetool"),
			filepath.Join(root, "tools", appImageName),
		)
	}
	if cache, err := os.UserCacheDir(); err == nil && cache != "" {
		candidates = append(candidates,
			filepath.Join(cache, "zumbra", "tools", "appimagetool"),
			filepath.Join(cache, "zumbra", "tools", appImageName),
		)
	}
	candidates = append(candidates, "appimagetool", appImageName)
	return uniqueCandidates(candidates)
}

func FindAppImageTool(explicit, projectRoot, arch string) (string, error) {
	return findExecutable(AppImageToolCandidates(explicit, projectRoot, arch), "appimagetool")
}

// AppImageRuntimeCandidates returns deterministic local runtime locations. A
// runtime is optional on the first online build because recent appimagetool
// versions can fetch one, but once cached it is reused through --runtime-file.
func AppImageRuntimeCandidates(explicit, projectRoot, arch string) []string {
	if value := strings.TrimSpace(explicit); value != "" {
		return []string{value}
	}
	if value := strings.TrimSpace(os.Getenv("APPIMAGE_RUNTIME")); value != "" {
		return []string{value}
	}
	name := "runtime-" + appImageArch(arch)
	candidates := []string{}
	for _, root := range toolRoots(projectRoot) {
		candidates = append(candidates,
			filepath.Join(root, "tools", name),
			filepath.Join(root, "tools", name+".bin"),
		)
	}
	if cache, err := os.UserCacheDir(); err == nil && cache != "" {
		candidates = append(candidates,
			filepath.Join(cache, "zumbra", "tools", name),
			filepath.Join(cache, "zumbra", "tools", name+".bin"),
		)
	}
	return uniqueCandidates(candidates)
}

func FindAppImageRuntime(explicit, projectRoot, arch string) (string, error) {
	return findReadableFile(AppImageRuntimeCandidates(explicit, projectRoot, arch), "AppImage runtime")
}

func AppImageRuntimeCachePath(arch string) (string, error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cache, "zumbra", "tools", "runtime-"+appImageArch(arch)), nil
}

func FindNSISTool(explicit string) (string, error) {
	if value := strings.TrimSpace(explicit); value != "" {
		return findExecutable([]string{value}, "makensis")
	}
	if value := strings.TrimSpace(os.Getenv("MAKENSIS")); value != "" {
		return findExecutable([]string{value}, "makensis")
	}
	return findExecutable([]string{"makensis", "makensis.exe"}, "makensis")
}

func FindCodeSignTool() (string, error) {
	if value := strings.TrimSpace(os.Getenv("CODESIGN")); value != "" {
		return findExecutable([]string{value}, "codesign")
	}
	return findExecutable([]string{"codesign"}, "codesign")
}

func toolRoots(projectRoot string) []string {
	roots := []string{}
	if strings.TrimSpace(projectRoot) != "" {
		roots = append(roots, projectRoot)
	}
	if cwd, err := os.Getwd(); err == nil && cwd != "" {
		duplicate := false
		for _, root := range roots {
			if samePath(root, cwd) {
				duplicate = true
				break
			}
		}
		if !duplicate {
			roots = append(roots, cwd)
		}
	}
	return roots
}

func samePath(left, right string) bool {
	leftAbs, leftErr := filepath.Abs(left)
	rightAbs, rightErr := filepath.Abs(right)
	return leftErr == nil && rightErr == nil && filepath.Clean(leftAbs) == filepath.Clean(rightAbs)
}

func uniqueCandidates(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func findExecutable(candidates []string, label string) (string, error) {
	values := uniqueCandidates(candidates)
	for _, candidate := range values {
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
	if len(values) == 1 {
		candidate := values[0]
		if strings.ContainsRune(candidate, filepath.Separator) || filepath.IsAbs(candidate) {
			if absolute, err := filepath.Abs(candidate); err == nil {
				return "", fmt.Errorf("%s %q does not exist or is not executable", label, absolute)
			}
		}
	}
	return "", fmt.Errorf("%s was not found", label)
}

func findReadableFile(candidates []string, label string) (string, error) {
	values := uniqueCandidates(candidates)
	for _, candidate := range values {
		absolute, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		info, err := os.Stat(absolute)
		if err == nil && !info.IsDir() && info.Mode().IsRegular() {
			return absolute, nil
		}
	}
	if len(values) == 1 {
		if absolute, err := filepath.Abs(values[0]); err == nil {
			return "", fmt.Errorf("%s %q does not exist or is not a readable regular file", label, absolute)
		}
	}
	return "", fmt.Errorf("%s was not found", label)
}

func FormatToolSearch(candidates []string) string {
	values := uniqueCandidates(candidates)
	for index, value := range values {
		if strings.ContainsRune(value, filepath.Separator) || filepath.IsAbs(value) {
			if absolute, err := filepath.Abs(value); err == nil {
				values[index] = absolute
			}
		}
	}
	return strings.Join(values, ", ")
}

func AppImageInstallHint(arch string) string {
	appArch := appImageArch(arch)
	return fmt.Sprintf("install appimagetool in PATH, set APPIMAGETOOL, or place appimagetool-%s.AppImage in tools/ at the manifest root or current directory", appArch)
}

func AppImageRuntimeHint(arch string) string {
	return fmt.Sprintf("set APPIMAGE_RUNTIME, use --appimage-runtime, or place runtime-%s in tools/; without it appimagetool may require network on the first build", appImageArch(arch))
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
