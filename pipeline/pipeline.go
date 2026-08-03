// Package pipeline owns Zumbra's canonical front-end sequence. File execution,
// compiler adapters, evaluator adapters and future native backends consume the
// same analyzed Result instead of independently parsing and checking source.
package pipeline

import (
	"fmt"
	"os"
	"strings"
	"time"

	"zumbra/ast"
	"zumbra/diagnostics"
	"zumbra/hir"
	"zumbra/lexer"
	"zumbra/mir"
	"zumbra/modules"
	"zumbra/parser"
	"zumbra/semantic"
	"zumbra/types"
)

type Stage string

const (
	StageParser   Stage = "parser"
	StageModules  Stage = "modules"
	StageSemantic Stage = "semantic"
	StageTypes    Stage = "types"
	StageHIR      Stage = "hir"
	StageMIR      Stage = "mir"
	StageOptimize Stage = "optimize"
)

type Diagnostic struct {
	Stage   Stage             `json:"stage"`
	Message string            `json:"message"`
	Warning bool              `json:"warning"`
	Code    string            `json:"code"`
	File    string            `json:"file,omitempty"`
	Range   diagnostics.Range `json:"range"`
}

func (d Diagnostic) Error() string { return fmt.Sprintf("%s: %s", d.Stage, d.Message) }

func (d Diagnostic) Structured() diagnostics.Diagnostic {
	severity := diagnostics.SeverityError
	if d.Warning {
		severity = diagnostics.SeverityWarning
	}
	item := diagnostics.New(d.File, d.Code, string(d.Stage), d.Message, severity)
	if d.Range.Start.Line > 0 {
		item.Range = d.Range
	}
	return item
}

func newDiagnostic(filename string, stage Stage, code, message string, warning bool) Diagnostic {
	item := diagnostics.New(filename, code, string(stage), message, diagnostics.SeverityError)
	if warning {
		item.Severity = diagnostics.SeverityWarning
	}
	return Diagnostic{Stage: stage, Message: message, Warning: warning, Code: code, File: filename, Range: item.Range}
}

type Options struct {
	Optimize bool
}

type Result struct {
	Filename string
	Source   string
	Program  *ast.Program
	Modules  *modules.Graph
	Semantic *semantic.Result
	Types    *types.Analysis
	HIR      *hir.Module
	MIR      *mir.Module
	Warnings []Diagnostic
	Timings  map[Stage]time.Duration `json:"timings"`
}

func Build(filename, source string, options Options) (*Result, []Diagnostic) {
	result := &Result{Filename: filename, Source: source, Timings: map[Stage]time.Duration{}}
	stageStarted := time.Now()
	l := lexer.New(source)
	p := parser.New(l)
	program := p.ParseProgram()
	result.Program = program
	result.Timings[StageParser] = time.Since(stageStarted)
	if len(p.Errors()) > 0 {
		diagnostics := make([]Diagnostic, 0, len(p.Errors()))
		for _, message := range p.Errors() {
			diagnostics = append(diagnostics, newDiagnostic(filename, StageParser, "ZP1001", message, false))
		}
		return result, diagnostics
	}

	stageStarted = time.Now()
	flattened, moduleGraph, moduleDiagnostics := modules.Resolve(filename, program)
	result.Program = flattened
	result.Modules = moduleGraph
	result.Timings[StageModules] = time.Since(stageStarted)
	if len(moduleDiagnostics) > 0 {
		hasModuleErrors := false
		for _, item := range moduleDiagnostics {
			diagnostic := newDiagnostic(filename, StageModules, "ZM1001", item.Error(), item.Warning)
			if item.Warning {
				result.Warnings = append(result.Warnings, diagnostic)
			} else {
				hasModuleErrors = true
			}
		}
		if hasModuleErrors {
			diagnostics := []Diagnostic{}
			for _, item := range moduleDiagnostics {
				if !item.Warning {
					diagnostics = append(diagnostics, newDiagnostic(filename, StageModules, "ZM1001", item.Error(), false))
				}
			}
			return result, diagnostics
		}
	}
	program = flattened

	stageStarted = time.Now()
	semResult, semErrs := semantic.Analyze(program)
	result.Semantic = semResult
	result.Timings[StageSemantic] = time.Since(stageStarted)
	if len(semErrs) > 0 {
		return result, errorDiagnostics(filename, StageSemantic, "ZS1001", semErrs)
	}
	if semResult != nil {
		for _, warning := range semResult.Warnings {
			result.Warnings = append(result.Warnings, newDiagnostic(filename, StageSemantic, "ZS2001", warning.Message, true))
		}
	}

	stageStarted = time.Now()
	typeInfo, typeErrs := types.AnalyzeWithInfo(program)
	result.Types = typeInfo
	result.Timings[StageTypes] = time.Since(stageStarted)
	if len(typeErrs) > 0 {
		return result, errorDiagnostics(filename, StageTypes, "ZT1001", typeErrs)
	}

	stageStarted = time.Now()
	hirModule, err := hir.Lower(filename, program, typeInfo)
	if err != nil {
		result.Timings[StageHIR] = time.Since(stageStarted)
		return result, []Diagnostic{newDiagnostic(filename, StageHIR, "ZH1001", err.Error(), false)}
	}
	result.HIR = hirModule
	result.Timings[StageHIR] = time.Since(stageStarted)

	stageStarted = time.Now()
	mirModule, err := mir.Lower(hirModule)
	if err != nil {
		result.Timings[StageMIR] = time.Since(stageStarted)
		return result, []Diagnostic{newDiagnostic(filename, StageMIR, "ZI1001", err.Error(), false)}
	}
	result.Timings[StageMIR] = time.Since(stageStarted)
	if options.Optimize {
		stageStarted = time.Now()
		if err := mir.Optimize(mirModule); err != nil {
			result.Timings[StageOptimize] = time.Since(stageStarted)
			return result, []Diagnostic{newDiagnostic(filename, StageOptimize, "ZO1001", err.Error(), false)}
		}
		result.Timings[StageOptimize] = time.Since(stageStarted)
	}
	result.MIR = mirModule
	return result, nil
}

func BuildFile(filename string, options Options) (*Result, []Diagnostic) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, []Diagnostic{newDiagnostic(filename, StageParser, "ZP1000", fmt.Sprintf("could not read %s: %v", filename, err), false)}
	}
	return Build(filename, string(content), options)
}

func (r *Result) DumpHIR() string {
	if r == nil || r.HIR == nil {
		return ""
	}
	return r.HIR.Dump()
}
func (r *Result) DumpMIR() string {
	if r == nil || r.MIR == nil {
		return ""
	}
	return r.MIR.Dump()
}

func FormatDiagnostics(diagnostics []Diagnostic) string {
	var out strings.Builder
	for _, item := range diagnostics {
		out.WriteString(item.Error())
		out.WriteByte('\n')
	}
	return out.String()
}

func errorDiagnostics(filename string, stage Stage, code string, errs []error) []Diagnostic {
	result := make([]Diagnostic, 0, len(errs))
	for _, err := range errs {
		result = append(result, newDiagnostic(filename, stage, code, err.Error(), false))
	}
	return result
}
