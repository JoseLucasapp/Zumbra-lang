package types

import (
	"strings"
	"testing"
	"zumbra/lexer"
	"zumbra/parser"
)

func checkZ5Source(t *testing.T, source string) []error {
	t.Helper()
	p := parser.New(lexer.New(source))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	return NewChecker().Check(program)
}

func TestTypeCheckZ5Program(t *testing.T) {
	errors := checkZ5Source(t, `
        type Byte << u8;
        struct Cpu { opcode: Byte; pc: u16; fct advance(amount) { self.pc << self.pc + amount; } }
        enum State { Running; Stopped; }
        var cpu << Cpu(0xA9u8, 0x8000u16);
        cpu.advance(1u16);
        var name << match(State.Running) { case State.Running { "run"; } else { "stop"; } };
    `)
	if len(errors) != 0 {
		t.Fatalf("type errors: %v", errors)
	}
}

func TestTypeCheckRejectsWrongField(t *testing.T) {
	errors := checkZ5Source(t, `struct Point { x: int; } var p << Point(1); p.x << "bad";`)
	if len(errors) == 0 || !strings.Contains(errors[0].Error(), "field x expects") {
		t.Fatalf("unexpected errors: %v", errors)
	}
}

func TestTypeCheckRejectsDuplicateStructField(t *testing.T) {
	errors := checkZ5Source(t, `struct Point { x: int; x: int; }`)
	if len(errors) == 0 || !strings.Contains(errors[0].Error(), "duplicate field") {
		t.Fatalf("unexpected errors: %v", errors)
	}
}

func TestTypeCheckRejectsDifferentEnumsInMatch(t *testing.T) {
	errors := checkZ5Source(t, `enum A { Ready; } enum B { Ready; } match(A.Ready) { case B.Ready { 1; } else { 0; } };`)
	if len(errors) == 0 || !strings.Contains(errors[0].Error(), "match compares") {
		t.Fatalf("unexpected errors: %v", errors)
	}
}
