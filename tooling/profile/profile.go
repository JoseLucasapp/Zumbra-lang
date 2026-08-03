// Package profile measures the canonical compiler pipeline and emits pprof data.
package profile

import (
	"fmt"
	"math"
	"os"
	"runtime"
	"runtime/pprof"
	"sort"
	"time"

	"zumbra/pipeline"
)

type Options struct {
	Runs        int
	Warmup      int
	Optimize    bool
	CPUProfile  string
	HeapProfile string
}

type Stage struct {
	Name    string        `json:"name"`
	Total   time.Duration `json:"total"`
	Average time.Duration `json:"average"`
	Percent float64       `json:"percent"`
}

type Report struct {
	File              string                `json:"file"`
	Runs              int                   `json:"runs"`
	Warmup            int                   `json:"warmup"`
	Optimize          bool                  `json:"optimize"`
	Total             time.Duration         `json:"total"`
	Average           time.Duration         `json:"average"`
	Minimum           time.Duration         `json:"minimum"`
	Maximum           time.Duration         `json:"maximum"`
	Median            time.Duration         `json:"median"`
	P95               time.Duration         `json:"p95"`
	AllocatedBytes    uint64                `json:"allocated_bytes"`
	Allocations       uint64                `json:"allocations"`
	BytesPerRun       uint64                `json:"bytes_per_run"`
	AllocationsPerRun uint64                `json:"allocations_per_run"`
	Stages            []Stage               `json:"stages"`
	Diagnostics       []pipeline.Diagnostic `json:"diagnostics,omitempty"`
}

func Run(filename string, options Options) (*Report, error) {
	if options.Runs <= 0 {
		options.Runs = 10
	}
	if options.Warmup < 0 {
		options.Warmup = 0
	}
	for index := 0; index < options.Warmup; index++ {
		if _, diagnostics := pipeline.BuildFile(filename, pipeline.Options{Optimize: options.Optimize}); len(diagnostics) > 0 {
			return &Report{File: filename, Diagnostics: diagnostics}, fmt.Errorf("profile input does not compile")
		}
	}

	var cpuFile *os.File
	if options.CPUProfile != "" {
		file, err := os.Create(options.CPUProfile)
		if err != nil {
			return nil, fmt.Errorf("create CPU profile: %w", err)
		}
		cpuFile = file
		if err := pprof.StartCPUProfile(cpuFile); err != nil {
			_ = cpuFile.Close()
			return nil, fmt.Errorf("start CPU profile: %w", err)
		}
		defer func() {
			pprof.StopCPUProfile()
			_ = cpuFile.Close()
		}()
	}

	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	durations := make([]time.Duration, 0, options.Runs)
	stageTotals := map[pipeline.Stage]time.Duration{}
	report := &Report{File: filename, Runs: options.Runs, Warmup: options.Warmup, Optimize: options.Optimize}

	for index := 0; index < options.Runs; index++ {
		started := time.Now()
		result, diagnostics := pipeline.BuildFile(filename, pipeline.Options{Optimize: options.Optimize})
		elapsed := time.Since(started)
		durations = append(durations, elapsed)
		if len(diagnostics) > 0 {
			report.Diagnostics = diagnostics
			return report, fmt.Errorf("profile input failed on run %d", index+1)
		}
		if result != nil {
			for stage, duration := range result.Timings {
				stageTotals[stage] += duration
			}
		}
	}

	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	report.AllocatedBytes = after.TotalAlloc - before.TotalAlloc
	report.Allocations = after.Mallocs - before.Mallocs
	report.BytesPerRun = report.AllocatedBytes / uint64(options.Runs)
	report.AllocationsPerRun = report.Allocations / uint64(options.Runs)

	sortedDurations := append([]time.Duration(nil), durations...)
	sort.Slice(sortedDurations, func(i, j int) bool { return sortedDurations[i] < sortedDurations[j] })
	for _, duration := range durations {
		report.Total += duration
	}
	report.Average = report.Total / time.Duration(options.Runs)
	report.Minimum = sortedDurations[0]
	report.Maximum = sortedDurations[len(sortedDurations)-1]
	report.Median = percentile(sortedDurations, 0.50)
	report.P95 = percentile(sortedDurations, 0.95)

	stageNames := make([]string, 0, len(stageTotals))
	byName := map[string]time.Duration{}
	for stage, duration := range stageTotals {
		name := string(stage)
		stageNames = append(stageNames, name)
		byName[name] = duration
	}
	sort.Strings(stageNames)
	for _, name := range stageNames {
		total := byName[name]
		percent := 0.0
		if report.Total > 0 {
			percent = float64(total) / float64(report.Total) * 100
		}
		report.Stages = append(report.Stages, Stage{Name: name, Total: total, Average: total / time.Duration(options.Runs), Percent: percent})
	}
	sort.SliceStable(report.Stages, func(i, j int) bool { return report.Stages[i].Total > report.Stages[j].Total })

	if options.HeapProfile != "" {
		file, err := os.Create(options.HeapProfile)
		if err != nil {
			return nil, fmt.Errorf("create heap profile: %w", err)
		}
		runtime.GC()
		if err := pprof.WriteHeapProfile(file); err != nil {
			_ = file.Close()
			return nil, fmt.Errorf("write heap profile: %w", err)
		}
		if err := file.Close(); err != nil {
			return nil, err
		}
	}
	return report, nil
}

func percentile(sorted []time.Duration, ratio float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	index := int(math.Ceil(float64(len(sorted))*ratio)) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}
