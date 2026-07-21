package main

import (
	"flag"
	"fmt"
	"time"
	"zumbra/compiler"
	"zumbra/evaluator"
	"zumbra/object"
	"zumbra/pipeline"
	"zumbra/vm"
)

var engine = flag.String("engine", "vm", "use 'vm' or 'eval'")

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
	} else {

		env := object.NewEnvironment()
		start := time.Now()
		result = evaluator.EvalPipeline(frontEnd, env)
		duration = time.Since(start)
	}
	fmt.Printf(
		"engine=%s, result=%s, duration=%s\n",
		*engine,
		result.Inspect(),
		duration)

}
