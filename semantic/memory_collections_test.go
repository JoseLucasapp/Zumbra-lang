package semantic

import "testing"

func TestResolveMemoryCollectionBuiltins(t *testing.T) {
	errors := resolveInput(`
        var memory << bytes(16);
        var registers << arrayOf("u16", 8);
        var view << slice(memory, 0, 4);
        fill(view, 0u8);
        memory[0] << 1u8;
    `)
	if len(errors) != 0 {
		t.Fatalf("expected no resolver errors, got %v", errors)
	}
}
