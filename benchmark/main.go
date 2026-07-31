package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"zumbra/compiler"
	"zumbra/evaluator"
	"zumbra/nativec"
	"zumbra/object"
	"zumbra/pipeline"
	"zumbra/vm"
)

var engine = flag.String("engine", "vm", "use 'vm', 'eval' or 'native'")
var release = flag.Bool("release", true, "compile the native benchmark with release optimizations")

var input = `
var fibonacci << fct(x) {
	if (x == 0) {
		0
	} else {
		if (x == 1) {
			return 1;
		} else {
			fibonacci(x - 1) + fibonacci(x - 2);
		}
	}
};
fibonacci(35);
`

func main() {
	flag.Parse()

	var duration time.Duration
	var result object.Object

	frontEnd, diagnostics := pipeline.Build("benchmark.zum", input, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		fmt.Printf("pipeline error: %s", pipeline.FormatDiagnostics(diagnostics))
		return
	}

	if *engine == "vm" {
		comp := compiler.New()
		err := comp.CompilePipeline(frontEnd)
		if err != nil {
			fmt.Printf("compiler error: %s", err)
			return
		}

		machine := vm.New(comp.Bytecode())
		start := time.Now()

		err = machine.Run()
		if err != nil {
			fmt.Printf("vm error: %s", err)
			return
		}

		duration = time.Since(start)
		result = machine.LastPoppedStackElem()
	} else if *engine == "eval" {
		env := object.NewEnvironment()
		start := time.Now()
		result = evaluator.EvalPipeline(frontEnd, env)
		duration = time.Since(start)
	} else if *engine == "native" {
		nativeSource := strings.Replace(input, "fibonacci(35);", "show(fibonacci(35));", 1)
		nativeFrontEnd, diagnostics := pipeline.Build("benchmark-native.zum", nativeSource, pipeline.Options{Optimize: true})
		if len(diagnostics) != 0 {
			fmt.Printf("native pipeline error: %s", pipeline.FormatDiagnostics(diagnostics))
			return
		}
		directory, err := os.MkdirTemp("", "zumbra-native-benchmark-")
		if err != nil {
			fmt.Printf("temporary directory error: %s", err)
			return
		}
		defer os.RemoveAll(directory)
		output := filepath.Join(directory, "benchmark-native")
		compileStart := time.Now()
		_, nativeDiagnostics, err := nativec.Build(nativeFrontEnd.MIR, nativec.BuildOptions{
			Release:  *release,
			Output:   output,
			BuildDir: filepath.Join(directory, "src"),
		})
		compileDuration := time.Since(compileStart)
		if err != nil || len(nativeDiagnostics) != 0 {
			fmt.Printf("native build error: %v diagnostics=%v", err, nativeDiagnostics)
			return
		}
		start := time.Now()
		data, err := exec.Command(output).CombinedOutput()
		duration = time.Since(start)
		if err != nil {
			fmt.Printf("native execution error: %s: %s", err, data)
			return
		}
		fmt.Printf("engine=native, result=%s, compile_duration=%s, duration=%s\n", strings.TrimSpace(string(data)), compileDuration, duration)
		return
	} else {
		fmt.Printf("unknown engine %q; use vm, eval or native\n", *engine)
		return
	}
	fmt.Printf(
		"engine=%s, result=%s, duration=%s\n",
		*engine,
		result.Inspect(),
		duration)

}
