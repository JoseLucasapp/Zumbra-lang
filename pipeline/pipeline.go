// Package pipeline owns Zumbra's canonical front-end sequence. File execution,
// compiler adapters, evaluator adapters and future native backends consume the
// same analyzed Result instead of independently parsing and checking source.
package pipeline

import (
	"fmt"
	"os"
	"strings"

	"zumbra/ast"
	"zumbra/hir"
	"zumbra/lexer"
	"zumbra/mir"
	"zumbra/parser"
	"zumbra/semantic"
	"zumbra/types"
)

type Stage string

const (
	StageParser   Stage = "parser"
	StageSemantic Stage = "semantic"
	StageTypes    Stage = "types"
	StageHIR      Stage = "hir"
	StageMIR      Stage = "mir"
)

type Diagnostic struct {
	Stage   Stage
	Message string
	Warning bool
}

func (d Diagnostic) Error() string { return fmt.Sprintf("%s: %s", d.Stage, d.Message) }

type Options struct {
	Optimize bool
}

type Result struct {
	Filename string
	Source   string
	Program  *ast.Program
	Semantic *semantic.Result
	Types    *types.Analysis
	HIR      *hir.Module
	MIR      *mir.Module
	Warnings []Diagnostic
}

func Build(filename, source string, options Options) (*Result, []Diagnostic) {
	result := &Result{Filename: filename, Source: source}
	l := lexer.New(source)
	p := parser.New(l)
	program := p.ParseProgram()
	result.Program = program
	if len(p.Errors()) > 0 {
		diagnostics := make([]Diagnostic, 0, len(p.Errors()))
		for _, message := range p.Errors() {
			diagnostics = append(diagnostics, Diagnostic{Stage: StageParser, Message: message})
		}
		return result, diagnostics
	}

	semResult, semErrs := semantic.AnalyzeModule(filename, program)
	result.Semantic = semResult
	if len(semErrs) > 0 {
		return result, errorDiagnostics(StageSemantic, semErrs)
	}
	if semResult != nil {
		for _, warning := range semResult.Warnings {
			result.Warnings = append(result.Warnings, Diagnostic{Stage: StageSemantic, Message: warning.Message, Warning: true})
		}
	}

	typeInfo, typeErrs := types.AnalyzeModuleWithInfo(filename, program)
	result.Types = typeInfo
	if len(typeErrs) > 0 {
		return result, errorDiagnostics(StageTypes, typeErrs)
	}

	hirModule, err := hir.Lower(filename, program, typeInfo)
	if err != nil {
		return result, []Diagnostic{{Stage: StageHIR, Message: err.Error()}}
	}
	result.HIR = hirModule

	mirModule, err := mir.Lower(hirModule)
	if err != nil {
		return result, []Diagnostic{{Stage: StageMIR, Message: err.Error()}}
	}
	if options.Optimize {
		if err := mir.Optimize(mirModule); err != nil {
			return result, []Diagnostic{{Stage: StageMIR, Message: err.Error()}}
		}
	}
	result.MIR = mirModule
	return result, nil
}

func BuildFile(filename string, options Options) (*Result, []Diagnostic) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, []Diagnostic{{Stage: StageParser, Message: fmt.Sprintf("could not read %s: %v", filename, err)}}
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

func errorDiagnostics(stage Stage, errs []error) []Diagnostic {
	result := make([]Diagnostic, 0, len(errs))
	for _, err := range errs {
		result = append(result, Diagnostic{Stage: stage, Message: err.Error()})
	}
	return result
}
