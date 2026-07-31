package main

import "testing"

func TestParseNativeBuildArgumentsWithFFILinks(t *testing.T) {
	filename, options, err := parseBuildArguments([]string{
		"--release",
		"--link", "native/math.c",
		"--include", "native/include",
		"--library-dir", "native/lib",
		"-l", "m",
		"app.zum",
	})
	if err != nil {
		t.Fatal(err)
	}
	if filename != "app.zum" || len(options.Links) != 1 || options.Links[0] != "native/math.c" {
		t.Fatalf("unexpected build input: %q %#v", filename, options)
	}
	if len(options.IncludeDirs) != 1 || len(options.LibraryDirs) != 1 || len(options.Libraries) != 1 {
		t.Fatalf("missing native link flags: %#v", options)
	}
}

func TestParseBindCArguments(t *testing.T) {
	header, output, options, err := parseBindCArguments([]string{
		"--pub",
		"--link", "native/math.c",
		"-o", "bindings/math.zum",
		"native/math.h",
	})
	if err != nil {
		t.Fatal(err)
	}
	if header != "native/math.h" || output != "bindings/math.zum" || !options.Public || options.Link != "native/math.c" {
		t.Fatalf("unexpected bind-c arguments: %q %q %#v", header, output, options)
	}
}

func TestParseBindCArgumentsRejectsMultipleHeaders(t *testing.T) {
	if _, _, _, err := parseBindCArguments([]string{"one.h", "two.h"}); err == nil {
		t.Fatal("expected multiple-header error")
	}
}
