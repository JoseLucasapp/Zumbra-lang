package nativec_test

import (
	"os/exec"
	"strings"
	"testing"

	"zumbra/nativec"
	"zumbra/pipeline"
)

func TestZ9ConcurrencyBuildsAndRunsNatively(t *testing.T) {
	output := buildAndRunZ8(t, "code_examples/core/concurrency.zum")
	if output != "36\n7\n8\n2000\n" {
		t.Fatalf("unexpected concurrency output %q", output)
	}
}

func TestZ9NativeGeneratorUsesPthreadsAndSpawn(t *testing.T) {
	result, diagnostics := pipeline.Build("concurrency.zum", `
        fct answer() { 42; }
        var task << spawn answer();
        show(await task);
    `, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("pipeline diagnostics: %s", pipeline.FormatDiagnostics(diagnostics))
	}
	sources, nativeDiagnostics := nativec.Generate(result.MIR)
	if len(nativeDiagnostics) != 0 {
		t.Fatalf("native diagnostics: %#v", nativeDiagnostics)
	}
	program := string(sources.Program)
	runtimeSource := string(sources.Runtime)
	if !strings.Contains(program, "z_spawn") || !strings.Contains(program, "z_task_await") {
		t.Fatalf("spawn/await were not emitted:\n%s", program)
	}
	if !strings.Contains(runtimeSource, "pthread_create") || !strings.Contains(runtimeSource, "pthread_cond_wait") {
		t.Fatal("native concurrency runtime does not use pthread primitives")
	}
}

func TestZ9NativeTimeoutAndCancellation(t *testing.T) {
	output := buildAndRunSource(t, `
        fct slow() { sleepMs(30); 42; }
        var first << spawn slow();
        var timed << joinTimeout(first, 1);
        show(timed[1]);
        show(await first);

        var second << spawn slow();
        show(cancel(second));
        show(taskCancelled(second));
        show(await second);

        var messages << channel(1);
        var receiveResult << receiveTimeout(messages, 1);
        show(receiveResult[2]);
        closeChannel(messages);
        var closedResult << receiveOk(messages);
        show(closedResult[1]);
    `)
	if output != "false\n42\ntrue\ntrue\nnull\nfalse\nfalse\n" {
		t.Fatalf("unexpected timeout/cancellation output %q", output)
	}
}

func buildAndRunSource(t *testing.T, source string) string {
	t.Helper()
	result, diagnostics := pipeline.Build("z9-native-test.zum", source, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("pipeline diagnostics: %s", pipeline.FormatDiagnostics(diagnostics))
	}
	compiler, err := nativec.DetectCompiler("auto")
	if err != nil {
		t.Skip(err)
	}
	directory := t.TempDir()
	output := directory + "/program"
	built, nativeDiagnostics, err := nativec.Build(result.MIR, nativec.BuildOptions{Release: true, Compiler: compiler, Output: output, BuildDir: directory + "/sources"})
	if err != nil || len(nativeDiagnostics) != 0 {
		t.Fatalf("native build failed: err=%v diagnostics=%#v", err, nativeDiagnostics)
	}
	data, err := exec.Command(built.Output).CombinedOutput()
	if err != nil {
		t.Fatalf("native execution failed: %v\n%s", err, data)
	}
	return strings.ReplaceAll(string(data), "\r\n", "\n")
}

func TestZ9AsyncMethodBuildsAndRunsNatively(t *testing.T) {
	output := buildAndRunSource(t, `
        struct Worker {
            value: int;
            async fct calculate(amount) { self.value + amount; }
        }
        var worker << Worker(20);
        show(await worker.calculate(22));
    `)
	if output != "42\n" {
		t.Fatalf("unexpected async method output %q", output)
	}
}

func TestZ9NativeTaskFailurePropagatesWithoutDeadlock(t *testing.T) {
	result, diagnostics := pipeline.Build("task-error.zum", `
        fct fail() {
            var messages << channel(1);
            closeChannel(messages);
            send(messages, 1);
            return;
        }
        var task << spawn fail();
        await task;
    `, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("pipeline diagnostics: %s", pipeline.FormatDiagnostics(diagnostics))
	}
	compiler, err := nativec.DetectCompiler("auto")
	if err != nil {
		t.Skip(err)
	}
	directory := t.TempDir()
	built, nativeDiagnostics, err := nativec.Build(result.MIR, nativec.BuildOptions{
		Release: true, Compiler: compiler, Output: directory + "/program", BuildDir: directory + "/sources",
	})
	if err != nil || len(nativeDiagnostics) != 0 {
		t.Fatalf("native build failed: err=%v diagnostics=%#v", err, nativeDiagnostics)
	}
	command := exec.Command(built.Output)
	data, runErr := command.CombinedOutput()
	if runErr == nil {
		t.Fatalf("expected native task failure, got successful output %q", data)
	}
	if !strings.Contains(string(data), "task failed: cannot send to closed channel") {
		t.Fatalf("unexpected task failure output %q", data)
	}
}
