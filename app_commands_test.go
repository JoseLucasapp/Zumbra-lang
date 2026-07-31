package main

import (
	"fmt"
	"testing"
)

func TestParseAppOptions(t *testing.T) {
	release := false
	options, err := parseAppOptions([]string{"--manifest", "app.toml", "--compiler", "clang", "--debug", "-o", "dist/app", "--target", "windows", "--arch", "amd64", "--format", "portable", "--sign", "Example", "--appimagetool", "tools/appimagetool", "--appimage-runtime", "tools/runtime-x86_64", "--makensis", "tools/makensis", "--json"})
	if err != nil {
		t.Fatal(err)
	}
	if options.Manifest != "app.toml" || options.Compiler != "clang" || options.Output != "dist/app" || options.Release == nil || *options.Release != release || options.Target != "windows" || options.Arch != "amd64" || options.Format != "portable" || options.SignIdentity != "Example" || options.AppImageTool != "tools/appimagetool" || options.AppImageRuntime != "tools/runtime-x86_64" || options.NSISTool != "tools/makensis" || !options.JSON {
		t.Fatalf("unexpected options: %#v", options)
	}
}

func TestParseAppOptionsRejectsUnknown(t *testing.T) {
	if _, err := parseAppOptions([]string{"--wat"}); err == nil {
		t.Fatal("expected unknown option error")
	}
}

func TestAppErrorUsageClassification(t *testing.T) {
	usage := appUsageErrorf("unknown app option --wat")
	if !isAppUsageError(usage) {
		t.Fatal("invalid CLI input must request usage")
	}
	operational := fmt.Errorf("appimagetool was not found")
	if isAppUsageError(operational) {
		t.Fatal("operational errors must not print global usage")
	}
}
