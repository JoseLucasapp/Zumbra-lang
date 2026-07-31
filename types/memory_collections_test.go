package types

import "testing"

func TestMemoryCollectionsAreTyped(t *testing.T) {
	errors := checkInput(t, `
        var memory << bytes(16);
        memory[0] << 255u8;
        var registers << arrayOf("u16", 8);
        registers[0] << 0xFFFFu16;
        var view << slice(registers, 0, 4);
        view[1] << 10u16;
        fill(memory, 0u8);
    `)
	if len(errors) != 0 {
		t.Fatalf("expected no errors, got %v", errors)
	}
}

func TestArrayOfRejectsUnknownElementType(t *testing.T) {
	errors := checkInput(t, `var values << arrayOf("word", 4);`)
	if len(errors) != 1 {
		t.Fatalf("expected one error, got %v", errors)
	}
}
