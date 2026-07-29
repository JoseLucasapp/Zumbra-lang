package nativec

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"zumbra/mir"
)

type BuildOptions struct {
	Release               bool
	EmitCOnly             bool
	Compiler              string
	Output                string
	BuildDir              string
	Links                 []string
	IncludeDirs           []string
	LibraryDirs           []string
	Libraries             []string
	EmbeddedAssets        []EmbeddedAsset
	TargetOS              string
	TargetArch            string
	Reproducible          bool
	ProjectRoot           string
	SourceDateEpoch       int64
	ApplicationName       string
	ApplicationVersion    string
	ApplicationIdentifier string
	WindowsIcon           string
	WindowsConsole        bool
}

type BuildResult struct {
	Compiler       string
	Output         string
	SourceDir      string
	ProgramSource  string
	RuntimeSource  string
	RuntimeHeader  string
	AssetsSource   string
	Command        []string
	Links          []string
	TargetOS       string
	TargetArch     string
	ResourceSource string
	ResourceObject string
}

func DetectCompiler(preferred string) (string, error) {
	if preferred != "" && preferred != "auto" {
		path, err := exec.LookPath(preferred)
		if err != nil {
			return "", fmt.Errorf("C compiler %q was not found in PATH", preferred)
		}
		return path, nil
	}
	if cc := os.Getenv("CC"); cc != "" {
		if path, err := exec.LookPath(cc); err == nil {
			return path, nil
		}
	}
	for _, candidate := range []string{"clang", "gcc", "cc"} {
		if path, err := exec.LookPath(candidate); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no C compiler found; install Clang or GCC, or set CC")
}

func DetectCompilerForTarget(preferred, targetOS, targetArch string) (string, error) {
	targetOS = strings.ToLower(strings.TrimSpace(targetOS))
	targetArch = strings.ToLower(strings.TrimSpace(targetArch))
	if targetOS == "" {
		targetOS = runtime.GOOS
	}
	if targetArch == "" {
		targetArch = runtime.GOARCH
	}
	if preferred != "" && preferred != "auto" {
		path, err := exec.LookPath(preferred)
		if err != nil {
			return "", fmt.Errorf("C compiler %q was not found in PATH", preferred)
		}
		return path, nil
	}
	key := "ZUMBRA_CC_" + strings.ToUpper(strings.ReplaceAll(targetOS+"_"+targetArch, "-", "_"))
	if configured := strings.TrimSpace(os.Getenv(key)); configured != "" {
		if path, err := exec.LookPath(configured); err == nil {
			return path, nil
		}
		return "", fmt.Errorf("compiler configured by %s was not found: %s", key, configured)
	}
	hostOS := runtime.GOOS
	if hostOS == "darwin" {
		hostOS = "macos"
	}
	if targetOS == hostOS && targetArch == runtime.GOARCH {
		return DetectCompiler("auto")
	}
	candidates := []string{}
	switch targetOS + "/" + targetArch {
	case "windows/amd64":
		candidates = []string{"x86_64-w64-mingw32-gcc", "x86_64-w64-mingw32-clang"}
	case "windows/arm64":
		candidates = []string{"aarch64-w64-mingw32-gcc", "aarch64-w64-mingw32-clang"}
	case "linux/amd64":
		candidates = []string{"x86_64-linux-gnu-gcc"}
	case "linux/arm64":
		candidates = []string{"aarch64-linux-gnu-gcc"}
	case "macos/amd64":
		candidates = []string{"o64-clang", "x86_64-apple-darwin-clang"}
	case "macos/arm64":
		candidates = []string{"oa64-clang", "arm64-apple-darwin-clang", "aarch64-apple-darwin-clang"}
	}
	for _, candidate := range candidates {
		if path, err := exec.LookPath(candidate); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no compiler found for %s/%s; %s", targetOS, targetArch, CompilerInstallHint(targetOS, targetArch))
}

// CompilerInstallHint returns the concise setup action used by CLI diagnostics.
func CompilerInstallHint(targetOS, targetArch string) string {
	key := "ZUMBRA_CC_" + strings.ToUpper(strings.ReplaceAll(targetOS+"_"+targetArch, "-", "_"))
	switch targetOS {
	case "windows":
		if runtime.GOOS == "linux" {
			return "install MinGW-w64 (Debian/Ubuntu: sudo apt install gcc-mingw-w64-x86-64 binutils-mingw-w64-x86-64 nsis) or set " + key
		}
		return "install a MinGW-w64 cross compiler or set " + key
	case "macos":
		return "build on macOS or configure an osxcross compiler and SDK through " + key
	case "linux":
		return "install the target GNU cross compiler or set " + key
	default:
		return "install a cross compiler or set " + key
	}
}

func Build(module *mir.Module, options BuildOptions) (*BuildResult, []Diagnostic, error) {
	targetOS := strings.ToLower(strings.TrimSpace(options.TargetOS))
	targetArch := strings.ToLower(strings.TrimSpace(options.TargetArch))
	if targetOS == "" || targetOS == "host" {
		targetOS = runtime.GOOS
	}
	if targetOS == "darwin" {
		targetOS = "macos"
	}
	if targetArch == "" || targetArch == "host" {
		targetArch = runtime.GOARCH
	}
	sources, diagnostics := Generate(module)
	if len(diagnostics) != 0 {
		return nil, diagnostics, nil
	}
	buildDir := options.BuildDir
	if buildDir == "" {
		buildDir = filepath.Join("build", "native")
	}
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		return nil, nil, fmt.Errorf("create native build directory: %w", err)
	}
	programPath := filepath.Join(buildDir, "main.c")
	runtimePath := filepath.Join(buildDir, "zumbra_runtime.c")
	headerPath := filepath.Join(buildDir, "zumbra_runtime.h")
	assetsPath := filepath.Join(buildDir, "zumbra_assets.c")
	assetSource := generateEmbeddedAssetsSource(options.EmbeddedAssets)
	for path, content := range map[string][]byte{
		programPath: sources.Program,
		runtimePath: sources.Runtime,
		headerPath:  sources.Header,
		assetsPath:  assetSource,
	} {
		if err := os.WriteFile(path, content, 0o644); err != nil {
			return nil, nil, fmt.Errorf("write %s: %w", path, err)
		}
	}

	result := &BuildResult{
		SourceDir:     buildDir,
		TargetOS:      targetOS,
		TargetArch:    targetArch,
		ProgramSource: programPath,
		RuntimeSource: runtimePath,
		RuntimeHeader: headerPath,
		AssetsSource:  assetsPath,
	}
	if options.EmitCOnly {
		return result, nil, nil
	}
	compiler, err := DetectCompilerForTarget(options.Compiler, targetOS, targetArch)
	if err != nil {
		return nil, nil, err
	}
	output := options.Output
	if output == "" {
		output = filepath.Join("build", executableNameForTarget(module.Filename, targetOS))
	}
	if parent := filepath.Dir(output); parent != "." {
		if err := os.MkdirAll(parent, 0o755); err != nil {
			return nil, nil, fmt.Errorf("create output directory: %w", err)
		}
	}
	args := []string{"-std=c11", "-pthread", "-Wall", "-Wextra", "-Werror", "-Wno-unused-variable", "-Wno-unused-parameter", "-I", buildDir}
	for _, includeDir := range options.IncludeDirs {
		args = append(args, "-I", includeDir)
	}
	for _, libraryDir := range options.LibraryDirs {
		args = append(args, "-L", libraryDir)
	}
	targetKey := strings.ToUpper(strings.ReplaceAll(targetOS+"_"+targetArch, "-", "_"))
	if sysroot := strings.TrimSpace(os.Getenv("ZUMBRA_SYSROOT_" + targetKey)); sysroot != "" {
		args = append(args, "--sysroot="+sysroot)
	}
	if flags := strings.TrimSpace(os.Getenv("ZUMBRA_CFLAGS_" + targetKey)); flags != "" {
		args = append(args, strings.Fields(flags)...)
	}
	if options.Reproducible {
		root := strings.TrimSpace(options.ProjectRoot)
		if root != "" {
			if absolute, absErr := filepath.Abs(root); absErr == nil {
				args = append(args, "-ffile-prefix-map="+absolute+"=.", "-fdebug-prefix-map="+absolute+"=.")
			}
		}
		if targetOS == "linux" {
			args = append(args, "-Wl,--build-id=sha1")
		}
		if targetOS == "windows" {
			args = append(args, "-Wl,--no-insert-timestamp")
		}
	}
	if options.Release {
		args = append(args, "-O3", "-DNDEBUG")
	} else {
		args = append(args, "-O0", "-g3")
	}
	links := append([]string{}, NativeLinks(module)...)
	links = append(links, options.Links...)
	links = uniqueStrings(links)
	for _, link := range links {
		if _, statErr := os.Stat(link); statErr != nil {
			return nil, nil, fmt.Errorf("native link input %s: %w", link, statErr)
		}
	}
	resourceObject := ""
	if targetOS == "windows" && (strings.TrimSpace(options.WindowsIcon) != "" || strings.TrimSpace(options.ApplicationName) != "" || strings.TrimSpace(options.ApplicationVersion) != "") {
		resourcePath := filepath.Join(buildDir, "zumbra_app.rc")
		resourceObject = filepath.Join(buildDir, "zumbra_app_res.o")
		resource, resourceErr := generateWindowsResource(options)
		if resourceErr != nil {
			return nil, nil, resourceErr
		}
		if writeErr := os.WriteFile(resourcePath, []byte(resource), 0o644); writeErr != nil {
			return nil, nil, fmt.Errorf("write Windows resource: %w", writeErr)
		}
		windres, detectErr := detectWindres(targetArch)
		if detectErr != nil {
			return nil, nil, detectErr
		}
		command := exec.Command(windres, resourcePath, "-O", "coff", "-o", resourceObject)
		command.Stdout, command.Stderr = os.Stdout, os.Stderr
		if runErr := command.Run(); runErr != nil {
			return nil, nil, fmt.Errorf("Windows resource compilation failed: %w", runErr)
		}
		result.ResourceSource = resourcePath
		result.ResourceObject = resourceObject
	}
	args = append(args, programPath, runtimePath)
	if resourceObject != "" {
		args = append(args, resourceObject)
	}
	if targetOS == "windows" && !options.WindowsConsole {
		args = append(args, "-mwindows")
	}
	if UsesAssets(module) || len(options.EmbeddedAssets) > 0 {
		args = append(args, assetsPath)
	}
	args = append(args, links...)
	for _, library := range options.Libraries {
		args = append(args, "-l"+library)
	}
	if UsesDesktop(module) && targetOS == "linux" {
		args = append(args, "-ldl")
	}
	if UsesDesktop(module) && targetOS == "windows" {
		args = append(args, "-lcomdlg32", "-lshell32", "-lole32", "-luser32")
	}
	if UsesTLS(module) || UsesHTTP(module) {
		args = append(args, "-lssl", "-lcrypto")
	}
	if UsesHTTP(module) {
		args = append(args, "-lz")
	}
	if UsesSQLite(module) {
		args = append(args, "-lsqlite3")
	}
	if UsesPostgres(module) {
		args = append(args, "-lpq")
	}
	if UsesRedis(module) {
		args = append(args, "-lhiredis")
	}
	if flags := strings.TrimSpace(os.Getenv("ZUMBRA_LDFLAGS_" + targetKey)); flags != "" {
		args = append(args, strings.Fields(flags)...)
	}
	args = append(args, "-lm", "-pthread", "-o", output)
	command := exec.Command(compiler, args...)
	command.Env = os.Environ()
	if options.Reproducible {
		epoch := options.SourceDateEpoch
		if epoch <= 0 {
			epoch = 946684800
		}
		command.Env = append(command.Env, fmt.Sprintf("SOURCE_DATE_EPOCH=%d", epoch), "LC_ALL=C", "TZ=UTC")
	}
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return nil, nil, fmt.Errorf("native C compilation failed: %w", err)
	}
	result.Compiler = compiler
	result.Output = output
	result.Command = append([]string{compiler}, args...)
	result.Links = links
	return result, nil, nil
}

func detectWindres(targetArch string) (string, error) {
	key := "ZUMBRA_WINDRES_WINDOWS_" + strings.ToUpper(targetArch)
	if configured := strings.TrimSpace(os.Getenv(key)); configured != "" {
		if path, err := exec.LookPath(configured); err == nil {
			return path, nil
		}
		return "", fmt.Errorf("resource compiler configured by %s was not found: %s", key, configured)
	}
	candidates := []string{"windres"}
	if targetArch == "arm64" {
		candidates = append([]string{"aarch64-w64-mingw32-windres"}, candidates...)
	} else {
		candidates = append([]string{"x86_64-w64-mingw32-windres"}, candidates...)
	}
	for _, candidate := range candidates {
		if path, err := exec.LookPath(candidate); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("Windows resource compiler was not found; install MinGW windres or set %s", key)
}

func generateWindowsResource(options BuildOptions) (string, error) {
	version := strings.TrimSpace(options.ApplicationVersion)
	if version == "" {
		version = "0.0.0"
	}
	parts := strings.SplitN(version, "+", 2)
	parts = strings.SplitN(parts[0], "-", 2)
	numbers := strings.Split(parts[0], ".")
	if len(numbers) != 3 {
		return "", fmt.Errorf("Windows resource version must be semantic version, got %q", version)
	}
	for len(numbers) < 4 {
		numbers = append(numbers, "0")
	}
	name := windowsRCString(options.ApplicationName)
	identifier := windowsRCString(options.ApplicationIdentifier)
	icon := ""
	if strings.TrimSpace(options.WindowsIcon) != "" {
		if !strings.EqualFold(filepath.Ext(options.WindowsIcon), ".ico") {
			return "", fmt.Errorf("Windows executable icon must be an .ico file")
		}
		absolute, err := filepath.Abs(options.WindowsIcon)
		if err != nil {
			return "", err
		}
		icon = fmt.Sprintf("1 ICON %q\n", filepath.ToSlash(absolute))
	}
	return fmt.Sprintf(`%s1 VERSIONINFO
FILEVERSION %s,%s,%s,%s
PRODUCTVERSION %s,%s,%s,%s
FILEFLAGSMASK 0x3fL
FILEFLAGS 0x0L
FILEOS 0x40004L
FILETYPE 0x1L
FILESUBTYPE 0x0L
BEGIN
  BLOCK "StringFileInfo"
  BEGIN
    BLOCK "040904b0"
    BEGIN
      VALUE "CompanyName", "%s\\0"
      VALUE "FileDescription", "%s\\0"
      VALUE "FileVersion", "%s\\0"
      VALUE "InternalName", "%s\\0"
      VALUE "OriginalFilename", "%s.exe\\0"
      VALUE "ProductName", "%s\\0"
      VALUE "ProductVersion", "%s\\0"
    END
  END
  BLOCK "VarFileInfo"
  BEGIN
    VALUE "Translation", 0x409, 1200
  END
END
`, icon, numbers[0], numbers[1], numbers[2], numbers[3], numbers[0], numbers[1], numbers[2], numbers[3], name, name, windowsRCString(version), identifier, identifier, name, windowsRCString(version)), nil
}

func windowsRCString(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\"", "\\\"")
	value = strings.ReplaceAll(value, "\r", "")
	return strings.ReplaceAll(value, "\n", " ")
}

func executableName(filename string) string { return executableNameForTarget(filename, runtime.GOOS) }
func executableNameForTarget(filename, targetOS string) string {
	name := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	if name == "" || name == "." {
		name = "zumbra-app"
	}
	if targetOS == "windows" {
		name += ".exe"
	}
	return name
}

// NativeLinks returns source, object or library files declared by extern blocks.
func NativeLinks(module *mir.Module) []string {
	if module == nil {
		return nil
	}
	result := []string{}
	for _, declaration := range module.Declarations {
		if declaration != nil && declaration.Op == mir.OpExtern && declaration.Meta["link"] != "" {
			result = append(result, declaration.Meta["link"])
		}
	}
	return uniqueStrings(result)
}

func uniqueStrings(values []string) []string {
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

// DetectWindres reports the Windows resource compiler selected for an architecture.
func DetectWindres(targetArch string) (string, error) { return detectWindres(targetArch) }
