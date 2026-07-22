package nativec_test

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"zumbra/nativec"
	"zumbra/pipeline"
)

const nativeStructuredSource = `
const START << 5;
struct Counter {
    value: int;
    fct add(amount) { self.value << self.value + amount; }
}
enum Mode { Idle; Running; }
var folded << 2 + 3 * 4;
var counter << Counter(START);
counter.add(folded);
var mode << Mode.Running;
var label << match(mode) {
    case Mode.Idle { "idle"; }
    case Mode.Running { "running"; }
    else { "unknown"; }
};
show(folded);
show(counter.value);
show(label);
`

func buildMIR(t *testing.T, source string) *pipeline.Result {
	t.Helper()
	result, diagnostics := pipeline.Build("native_test.zum", source, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("pipeline failed:\n%s", pipeline.FormatDiagnostics(diagnostics))
	}
	return result
}

func TestGenerateConsumesOptimizedMIR(t *testing.T) {
	result := buildMIR(t, nativeStructuredSource)
	sources, diagnostics := nativec.Generate(result.MIR)
	if len(diagnostics) != 0 {
		t.Fatalf("native diagnostics: %v", diagnostics)
	}
	program := string(sources.Program)
	for _, expected := range []string{
		"Generated from Zumbra MIR",
		"z_construct_struct",
		"z_dispatch_function",
		"z_int(INT64_C(14))",
	} {
		if !strings.Contains(program, expected) {
			t.Fatalf("generated C does not contain %q:\n%s", expected, program)
		}
	}
	if len(sources.Runtime) == 0 || len(sources.Header) == 0 {
		t.Fatal("embedded native runtime was not returned")
	}
}

func TestBuildAndRunNativeStructuredProgram(t *testing.T) {
	compiler, err := nativec.DetectCompiler("auto")
	if err != nil {
		t.Skip(err)
	}
	result := buildMIR(t, nativeStructuredSource)
	directory := t.TempDir()
	output := filepath.Join(directory, "native-test")
	build, diagnostics, err := nativec.Build(result.MIR, nativec.BuildOptions{
		Release:  true,
		Compiler: compiler,
		Output:   output,
		BuildDir: filepath.Join(directory, "src"),
	})
	if err != nil {
		t.Fatalf("native build failed: %v", err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("native diagnostics: %v", diagnostics)
	}
	if build == nil || build.Output != output {
		t.Fatalf("unexpected build result: %#v", build)
	}
	command := exec.Command(output)
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("native executable failed: %v\n%s", err, data)
	}
	if got, want := string(data), "14\n19\nrunning\n"; got != want {
		t.Fatalf("unexpected native output:\nwant: %q\n got: %q", want, got)
	}
}

func TestNativeRuntimeSupportsCompactMemoryAndHashing(t *testing.T) {
	compiler, err := nativec.DetectCompiler("auto")
	if err != nil {
		t.Skip(err)
	}
	source := `
var memory << bytes(8);
memory[2] << 0xA9u8;
var page << slice(memory, 2, 6);
page[1] << 0x42u8;
show(memory[2]);
show(memory[3]);
show(sizeOf(page));
show(sha256(memory));
`
	result := buildMIR(t, source)
	directory := t.TempDir()
	output := filepath.Join(directory, "memory-test")
	_, diagnostics, err := nativec.Build(result.MIR, nativec.BuildOptions{
		Compiler: compiler,
		Output:   output,
		BuildDir: filepath.Join(directory, "src"),
	})
	if err != nil || len(diagnostics) != 0 {
		t.Fatalf("native build failed: err=%v diagnostics=%v", err, diagnostics)
	}
	data, err := exec.Command(output).CombinedOutput()
	if err != nil {
		t.Fatalf("native executable failed: %v\n%s", err, data)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 4 || lines[0] != "169" || lines[1] != "66" || lines[2] != "4" || len(lines[3]) != 64 {
		t.Fatalf("unexpected compact-memory output: %q", data)
	}
}

func TestUnsupportedNativeBuiltinHasExplicitDiagnostic(t *testing.T) {
	result := buildMIR(t, `show(randomInteger(1, 2));`)
	_, diagnostics := nativec.Generate(result.MIR)
	if len(diagnostics) == 0 {
		t.Fatal("expected unsupported builtin diagnostic")
	}
	if !strings.Contains(diagnostics[0].Message, `builtin "randomInteger" is not available`) {
		t.Fatalf("unexpected diagnostic: %v", diagnostics)
	}
}

func TestNativeRuntimeSupportsControlFlowFunctionsAndFixedIntegers(t *testing.T) {
	compiler, err := nativec.DetectCompiler("auto")
	if err != nil {
		t.Skip(err)
	}
	source := `
var sumTo << fct(limit) {
    var total << 0;
    var current << 1;
    while (current <= limit) {
        total << total + current;
        current << current + 1;
    }
    total;
}
var values << [1, 2, 3, 4];
var filtered << 0;
for value in values where value % 2 == 0 {
    filtered << filtered + value;
}
show(sumTo(5));
show(filtered);
show(255u8 + 1);
show(satAdd(255u8, 1u8));
`
	result := buildMIR(t, source)
	directory := t.TempDir()
	output := filepath.Join(directory, "control-test")
	_, diagnostics, err := nativec.Build(result.MIR, nativec.BuildOptions{
		Release:  true,
		Compiler: compiler,
		Output:   output,
		BuildDir: filepath.Join(directory, "src"),
	})
	if err != nil || len(diagnostics) != 0 {
		t.Fatalf("native build failed: err=%v diagnostics=%v", err, diagnostics)
	}
	data, err := exec.Command(output).CombinedOutput()
	if err != nil {
		t.Fatalf("native executable failed: %v\n%s", err, data)
	}
	if got, want := string(data), "15\n6\n0\n255\n"; got != want {
		t.Fatalf("unexpected control-flow output:\nwant: %q\n got: %q", want, got)
	}
}

func TestNativeRuntimeSupportsBinaryRoundTripAndEndianness(t *testing.T) {
	compiler, err := nativec.DetectCompiler("auto")
	if err != nil {
		t.Skip(err)
	}
	source := `
var data << bytes(16);
writeU16LE(data, 0, 0x1234u16);
writeU32BE(data, 2, 0x89ABCDEFu32);
writeU64LE(data, 6, 0x0102030405060708u64);
var clone << bytes(16);
copyBytes(clone, 0, data, 0, sizeOf(data));
show(readU16LE(clone, 0));
show(readU32BE(clone, 2));
show(readU64LE(clone, 6));
show(bytesEqual(data, clone));
show(writeBytes("roundtrip.bin", clone));
show(bytesEqual(clone, readBytes("roundtrip.bin")));
`
	result := buildMIR(t, source)
	directory := t.TempDir()
	output := filepath.Join(directory, "binary-test")
	_, diagnostics, err := nativec.Build(result.MIR, nativec.BuildOptions{
		Release:  true,
		Compiler: compiler,
		Output:   output,
		BuildDir: filepath.Join(directory, "src"),
	})
	if err != nil || len(diagnostics) != 0 {
		t.Fatalf("native build failed: err=%v diagnostics=%v", err, diagnostics)
	}
	command := exec.Command(output)
	command.Dir = directory
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("native executable failed: %v\n%s", err, data)
	}
	if got, want := string(data), "4660\n2309737967\n72623859790382856\ntrue\n16\ntrue\n"; got != want {
		t.Fatalf("unexpected binary output:\nwant: %q\n got: %q", want, got)
	}
}
