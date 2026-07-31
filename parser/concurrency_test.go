package parser

import (
	"testing"
	"zumbra/ast"
	"zumbra/lexer"
)

func TestParseSpawnCall(t *testing.T) {
	p := New(lexer.New(`var task << spawn work(21);`))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	statement := program.Statements[0].(*ast.VarStatement)
	spawn, ok := statement.Value.(*ast.SpawnExpression)
	if !ok {
		t.Fatalf("expected SpawnExpression, got %T", statement.Value)
	}
	call, ok := spawn.Value.(*ast.CallExpression)
	if !ok || len(call.Arguments) != 1 {
		t.Fatalf("expected one-argument call, got %T", spawn.Value)
	}
}

func TestSpawnRejectsNonCall(t *testing.T) {
	p := New(lexer.New(`var task << spawn work;`))
	p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected spawn diagnostic")
	}
}

func TestParseAsyncStructMethod(t *testing.T) {
	p := New(lexer.New(`
        struct Worker {
            value: int;
            async fct calculate(amount) { self.value + amount; }
        }
    `))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	stmt := program.Statements[0].(*ast.StructStatement)
	if len(stmt.Methods) != 1 || stmt.Methods[0].Function == nil || !stmt.Methods[0].Function.Async {
		t.Fatalf("async method was not preserved: %#v", stmt.Methods)
	}
}
