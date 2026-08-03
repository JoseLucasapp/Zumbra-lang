package main

import (
	"os"
	"strings"
	"testing"
)

func TestMultiplatformReleaseWorkflowContract(t *testing.T) {
	workflowBytes, err := os.ReadFile(".github/workflows/release.yml")
	if err != nil {
		t.Fatalf("read release workflow: %v", err)
	}
	workflow := string(workflowBytes)

	for _, required := range []string{
		"linux-amd64:",
		"windows-amd64:",
		"macos:",
		"publish:",
		"scripts/test-release-platform.sh",
		"libhiredis-dev",
		"libpq-dev",
		"libsqlite3-dev",
		"libssl-dev",
		"zlib1g-dev",
		"mingw-w64-ucrt-x86_64-sqlite",
		"actions/upload-artifact@v7",
		"actions/download-artifact@v8",
		"SHA256SUMS",
	} {
		if !strings.Contains(workflow, required) {
			t.Fatalf("release workflow is missing %q", required)
		}
	}
}

func TestReleasePlatformGateSeparatesCanonicalAndPortableSuites(t *testing.T) {
	scriptBytes, err := os.ReadFile("scripts/test-release-platform.sh")
	if err != nil {
		t.Fatalf("read release test script: %v", err)
	}
	script := string(scriptBytes)

	for _, required := range []string{
		"Linux)",
		"Windows|macOS)",
		"go test ./...",
		"portable_packages",
		"EXPECTED_VERSION",
		"Release host validation passed",
	} {
		if !strings.Contains(script, required) {
			t.Fatalf("release platform gate is missing %q", required)
		}
	}
}

func TestWindowsSQLiteLinkerContract(t *testing.T) {
	sourceBytes, err := os.ReadFile("object/builtins/sqlite_builtins.go")
	if err != nil {
		t.Fatalf("read SQLite builtins: %v", err)
	}
	if !strings.Contains(string(sourceBytes), "#cgo windows LDFLAGS: -lsqlite3") {
		t.Fatal("Windows CGO builds must link sqlite3")
	}
}
