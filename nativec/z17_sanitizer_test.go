package nativec

import (
	"reflect"
	"strings"
	"testing"
)

func TestZ17NormalizeSanitizers(t *testing.T) {
	got, err := normalizeSanitizers([]string{"address,undefined", "address"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"address", "undefined"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sanitizers=%v, want %v", got, want)
	}
}

func TestZ17RejectsIncompatibleSanitizers(t *testing.T) {
	_, err := normalizeSanitizers([]string{"thread,address"})
	if err == nil || !strings.Contains(err.Error(), "cannot be combined") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestZ17RejectsUnknownSanitizer(t *testing.T) {
	_, err := normalizeSanitizers([]string{"memory"})
	if err == nil || !strings.Contains(err.Error(), "unsupported sanitizer") {
		t.Fatalf("unexpected error: %v", err)
	}
}
