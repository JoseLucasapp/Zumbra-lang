package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"zumbra/compiler"
	"zumbra/nativec"
	"zumbra/object"
	"zumbra/object/builtins"
	"zumbra/pipeline"
)

func TestCoreSyntaxSnippetsParseAndCompile(t *testing.T) {
	snippets := []string{
		`var x << 10; x << x + 1;`,
		`var sum << fct(a, b) { a + b; }; sum(1, 2);`,
		`if (true) { 1; } else { 2; }`,
		`while (false) { 1; }`,
		`for i in 1..3 { i; }`,
		`var arr << [1, 2, 3]; arr[0];`,
		`var arr << [1, 2, 3]; arr[0] << 9; arr[0];`,
		`var dict << {"name": "z"}; dict["name"];`,
		`var dict << {"score": 1}; dict["score"] << 2; dict["score"];`,
		`var memory << bytes(16); memory[0] << 0xA9u8; memory[0];`,
		`var values << arrayOf("u16", 4); values[1] << 0x1234u16; values[1];`,
		`var memory << bytes(8); var view << slice(memory, 2, 6); fill(view, 0u8);`,
		`var data << bytes(16); writeU16LE(data, 0, 0x1234u16); readU16LE(data, 0);`,
		`var source << bytes(4); var target << bytes(4); copyBytes(target, 0, source, 0, 4); bytesEqual(source, target); sha256(target);`,
		`const Max << 3; Max;`,
		`type Byte << u8; struct Cpu { opcode: Byte; pc: u16; } var cpu << Cpu(0xA9u8, 0x8000u16); cpu.pc;`,
		`struct Point { x: int; y: int; fct move(dx, dy) { self.x << self.x + dx; self.y << self.y + dy; } } var p << Point(1, 2); p.move(3, 4); p.x;`,
		`enum Direction { Up; Down; } match(Direction.Up) { case Direction.Up { 1; } else { 0; } };`,
		`var task << async fct() { 10; }; await task();`,
		`fct answer() { 42; } var task << spawn answer(); await task;`,
		`var messages << channel(2); send(messages, "ready"); var message << receive(messages); closeChannel(messages); message;`,
		`var counter << atomicInt(0); atomicAdd(counter, 1); atomicLoad(counter);`,
		`var identity << fct(value) { value; }; var result << identity(42); result;`,
		`fct publish(output, value) { send(output, value); return; } var messages << channel(1); var task << spawn publish(messages, 7); await task;`,
		`var listener << tcpListen("127.0.0.1", 0); var port << listenerPort(listener); listenerClose(listener); port;`,
		`var socket << udpBind("127.0.0.1", 0); udpClose(socket);`,
		`var run << fct() { 1; }; try run() or err { err; };`,
	}

	for i, source := range snippets {
		t.Run(strings.Join([]string{"snippet", string(rune('0' + i))}, "_"), func(t *testing.T) {
			result, diagnostics := pipeline.Build("docs-snippet.zum", source, pipeline.Options{Optimize: true})
			if len(diagnostics) > 0 {
				t.Fatalf("pipeline errors:\n\t%s", pipeline.FormatDiagnostics(diagnostics))
			}

			symbolTable := compiler.NewSymbolTable()
			for i, v := range builtins.Builtins {
				symbolTable.DefineBuiltin(i, v.Name)
			}

			comp := compiler.NewWithStateAndDir(symbolTable, []object.Object{}, ".")
			if err := comp.CompilePipeline(result); err != nil {
				t.Fatalf("compiler error: %s", err)
			}
		})
	}
}

func TestNativeDocumentationSnippetGeneratesC(t *testing.T) {
	source := `
struct Counter { value: int; fct add(amount) { self.value << self.value + amount; } }
var counter << Counter(5);
counter.add(14);
var memory << bytes(4);
memory[0] << 0xA9u8;
show(counter.value);
show(memory[0]);
`
	result, diagnostics := pipeline.Build("native-docs.zum", source, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("pipeline failed: %s", pipeline.FormatDiagnostics(diagnostics))
	}
	sources, nativeDiagnostics := nativec.Generate(result.MIR)
	if len(nativeDiagnostics) != 0 {
		t.Fatalf("native generation failed: %v", nativeDiagnostics)
	}
	if !strings.Contains(string(sources.Program), "Generated from Zumbra MIR") {
		t.Fatal("native C marker not found")
	}
}

func TestZ8ModuleDocumentationExample(t *testing.T) {
	dir := t.TempDir()
	module := filepath.Join(dir, "math.zum")
	entry := filepath.Join(dir, "app.zum")
	if err := os.WriteFile(module, []byte(`
        pub const BASE << 40;
        const SECRET << 99;
        pub fct add(left, right) { left + right; }
    `), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(entry, []byte(`
        import "math.zum" as math;
        show(math.add(math.BASE, 2));
    `), 0o644); err != nil {
		t.Fatal(err)
	}
	result, diagnostics := pipeline.BuildFile(entry, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("module documentation failed: %s", pipeline.FormatDiagnostics(diagnostics))
	}
	if result.Modules == nil || len(result.Modules.Units) != 2 {
		t.Fatalf("module graph missing: %#v", result.Modules)
	}
}

func TestZ8FFIDocumentationExampleGeneratesC(t *testing.T) {
	source := `
        extern "C" {
            fct add(left: i32, right: i32) -> i32 as "native_add";
            fct apply(value: i32, cb: callback(i32) -> i32) -> i32;
        }
        var double << fct(value) { value * 2i32; };
        unsafe { show(add(20i32, 22i32)); show(apply(21i32, double)); }
    `
	result, diagnostics := pipeline.Build("ffi-docs.zum", source, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("FFI documentation pipeline failed: %s", pipeline.FormatDiagnostics(diagnostics))
	}
	sources, nativeDiagnostics := nativec.Generate(result.MIR)
	if len(nativeDiagnostics) != 0 {
		t.Fatalf("FFI documentation generation failed: %v", nativeDiagnostics)
	}
	generated := string(sources.Program)
	if !strings.Contains(generated, "extern int32_t native_add") || !strings.Contains(generated, "zffi_trampoline") {
		t.Fatalf("FFI declarations were not generated:\n%s", generated)
	}
	if strings.Contains(result.DumpHIR(), "fct(unknown) -> unknown") || strings.Contains(result.DumpMIR(), "fct(unknown) -> unknown") {
		t.Fatalf("contextual callback type was not propagated:\nHIR:\n%s\nMIR:\n%s", result.DumpHIR(), result.DumpMIR())
	}
	if !strings.Contains(result.DumpHIR(), "fct(i32) -> i32") || !strings.Contains(result.DumpMIR(), "fct(i32) -> i32") {
		t.Fatalf("contextual callback type is missing:\nHIR:\n%s\nMIR:\n%s", result.DumpHIR(), result.DumpMIR())
	}
}

func TestZ9ConcurrencyDocumentationGeneratesNativeC(t *testing.T) {
	source := `
        fct answer() { 42; }
        var task << spawn answer();
        var result << await task;
        var messages << channel(1);
        send(messages, result);
        show(receive(messages));
    `
	result, diagnostics := pipeline.Build("concurrency-docs.zum", source, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("concurrency documentation pipeline failed: %s", pipeline.FormatDiagnostics(diagnostics))
	}
	sources, nativeDiagnostics := nativec.Generate(result.MIR)
	if len(nativeDiagnostics) != 0 {
		t.Fatalf("concurrency documentation native generation failed: %v", nativeDiagnostics)
	}
	generated := string(sources.Program)
	if !strings.Contains(generated, "z_spawn") || !strings.Contains(generated, "z_task_await") {
		t.Fatalf("concurrency operations were not generated:\n%s", generated)
	}
}

func TestZ91CallInferenceDocumentationIsConcrete(t *testing.T) {
	source := `
        fct square(value) { value * value; }
        fct publish(messages, value) { send(messages, value); return; }
        var task << spawn square(6);
        var messages << channel(1);
        var producer << spawn publish(messages, 7);
        await producer;
        await task;
    `
	result, diagnostics := pipeline.Build("call-inference-docs.zum", source, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("call inference documentation pipeline failed: %s", pipeline.FormatDiagnostics(diagnostics))
	}
	for _, dump := range []string{result.DumpHIR(), result.DumpMIR()} {
		for _, expected := range []string{"fct(int) -> int", "channel<int>", "task<int>"} {
			if !strings.Contains(dump, expected) {
				t.Fatalf("missing %q from typed dump:\n%s", expected, dump)
			}
		}
		for _, forbidden := range []string{"function square(value) -> unknown", `name="square" : fct(unknown) -> unknown`} {
			if strings.Contains(dump, forbidden) {
				t.Fatalf("call inference documentation retained %q:\n%s", forbidden, dump)
			}
		}
	}
}

func TestZ10NetworkDocumentationGeneratesConditionalNativeRuntime(t *testing.T) {
	result, diagnostics := pipeline.Build("network-docs.zum", `
        var listener << tcpListen("127.0.0.1", 0);
        var port << listenerPort(listener);
        listenerClose(listener);
        show(port > 0);
    `, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("network documentation pipeline failed: %s", pipeline.FormatDiagnostics(diagnostics))
	}
	sources, nativeDiagnostics := nativec.Generate(result.MIR)
	if len(nativeDiagnostics) != 0 {
		t.Fatalf("network documentation native generation failed: %v", nativeDiagnostics)
	}
	if !strings.Contains(string(sources.Runtime), "#define ZUMBRA_ENABLE_NETWORK 1") {
		t.Fatal("network documentation did not enable the native network runtime")
	}
	if strings.Contains(string(sources.Runtime), "#define ZUMBRA_ENABLE_TLS 1") {
		t.Fatal("TCP-only documentation unexpectedly enabled TLS")
	}
}
