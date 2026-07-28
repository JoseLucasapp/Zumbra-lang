package main

import "testing"

func TestParseAppOptions(t *testing.T) {
	release := false
	options, err := parseAppOptions([]string{"--manifest", "app.toml", "--compiler", "clang", "--debug", "-o", "dist/app"})
	if err != nil {
		t.Fatal(err)
	}
	if options.Manifest != "app.toml" || options.Compiler != "clang" || options.Output != "dist/app" || options.Release == nil || *options.Release != release {
		t.Fatalf("unexpected options: %#v", options)
	}
}

func TestParseAppOptionsRejectsUnknown(t *testing.T) {
	if _, err := parseAppOptions([]string{"--wat"}); err == nil {
		t.Fatal("expected unknown option error")
	}
}
