package main

import "testing"

func TestParseNativeBuildArguments(t *testing.T) {
	filename, options, err := parseBuildArguments([]string{"--release", "--compiler", "clang", "-o", "dist/game", "game.zum"})
	if err != nil {
		t.Fatal(err)
	}
	if filename != "game.zum" || !options.Release || options.Compiler != "clang" || options.Output != "dist/game" {
		t.Fatalf("unexpected parsed arguments: %q %#v", filename, options)
	}
}

func TestParseNativeBuildArgumentsRejectsMultipleInputs(t *testing.T) {
	if _, _, err := parseBuildArguments([]string{"a.zum", "b.zum"}); err == nil {
		t.Fatal("expected multiple-input error")
	}
}

func TestParseNativeBuildArgumentsWithSanitizers(t *testing.T) {
	filename, options, err := parseBuildArguments([]string{"--sanitize", "address,undefined", "app.zum"})
	if err != nil {
		t.Fatal(err)
	}
	if filename != "app.zum" || len(options.Sanitizers) != 1 || options.Sanitizers[0] != "address,undefined" {
		t.Fatalf("unexpected sanitizer arguments: %q %#v", filename, options)
	}
}

func TestParseNativeBuildArgumentsRequiresSanitizerValue(t *testing.T) {
	if _, _, err := parseBuildArguments([]string{"--sanitize"}); err == nil {
		t.Fatal("expected missing sanitizer value error")
	}
}
