package nativec_test

import (
	"strings"
	"testing"

	"zumbra/nativec"
	"zumbra/pipeline"
)

func TestZ12PublicExamplesBuildAndRunNatively(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"code_examples/core/data_persistence.zum", "2\n2\nLucas\n42\nLucas\nZumbra\ntrue\n43\ntrue\n"},
		{"code_examples/core/config_observability.zum", "8080\ntrue\n[REDACTED]\n2\ntrue\nok\ntrue\ntrue\nfalse\nLucas\ntrue\nLucas\ntrue\n"},
		{"code_examples/core/data_serialization.zum", "true\nLucas\n43981\ntrue\n42\n"},
		{"code_examples/core/data_exchange.zum", "true\ntrue\nFinal Fantasy IX\nfalse\ntrue\ntrue\n2\ncomma, preserved\n"},
	}
	for _, test := range tests {
		t.Run(strings.TrimSuffix(strings.TrimPrefix(test.path, "code_examples/core/"), ".zum"), func(t *testing.T) {
			if output := buildAndRunZ8(t, test.path); output != test.expected {
				t.Fatalf("unexpected native output %q", output)
			}
		})
	}
}

func TestZ12NativeRuntimeFeaturesAreConditionallyEnabled(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		required  []string
		forbidden []string
	}{
		{"plain", `show(42);`, nil, []string{"#define ZUMBRA_ENABLE_Z12 1", "#define ZUMBRA_ENABLE_SQLITE 1", "#define ZUMBRA_ENABLE_POSTGRES 1", "#define ZUMBRA_ENABLE_REDIS 1"}},
		{"services", `var value << binaryEncode({"ok": true}); show(sizeOf(value));`, []string{"#define ZUMBRA_ENABLE_Z12 1"}, []string{"#define ZUMBRA_ENABLE_SQLITE 1", "#define ZUMBRA_ENABLE_POSTGRES 1", "#define ZUMBRA_ENABLE_REDIS 1"}},
		{"sqlite", `var db << sqliteMemory(); db.close();`, []string{"#define ZUMBRA_ENABLE_Z12 1", "#define ZUMBRA_ENABLE_SQLITE 1"}, []string{"#define ZUMBRA_ENABLE_POSTGRES 1", "#define ZUMBRA_ENABLE_REDIS 1"}},
		{"postgres", `var db << postgresOpen("postgres://localhost/test", {}); db.close();`, []string{"#define ZUMBRA_ENABLE_Z12 1", "#define ZUMBRA_ENABLE_POSTGRES 1"}, []string{"#define ZUMBRA_ENABLE_SQLITE 1", "#define ZUMBRA_ENABLE_REDIS 1"}},
		{"redis", `var client << redisOpen("127.0.0.1", 6379, "", 0, 1); client.close();`, []string{"#define ZUMBRA_ENABLE_Z12 1", "#define ZUMBRA_ENABLE_REDIS 1"}, []string{"#define ZUMBRA_ENABLE_SQLITE 1", "#define ZUMBRA_ENABLE_POSTGRES 1"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, diagnostics := pipeline.Build(test.name+".zum", test.source, pipeline.Options{Optimize: true})
			if len(diagnostics) != 0 {
				t.Fatalf("pipeline diagnostics: %s", pipeline.FormatDiagnostics(diagnostics))
			}
			sources, nativeDiagnostics := nativec.Generate(result.MIR)
			if len(nativeDiagnostics) != 0 {
				t.Fatalf("native diagnostics: %#v", nativeDiagnostics)
			}
			runtime := string(sources.Runtime)
			for _, expected := range test.required {
				if !strings.Contains(runtime, expected) {
					t.Fatalf("missing %q", expected)
				}
			}
			for _, unexpected := range test.forbidden {
				if strings.Contains(runtime, unexpected) {
					t.Fatalf("unexpected %q", unexpected)
				}
			}
		})
	}
}

func TestZ12PostgresAndRedisExampleGeneratesPortableNativeC(t *testing.T) {
	result, diagnostics := pipeline.BuildFile("../code_examples/core/postgres_redis.zum", pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("pipeline diagnostics: %s", pipeline.FormatDiagnostics(diagnostics))
	}
	sources, nativeDiagnostics := nativec.Generate(result.MIR)
	if len(nativeDiagnostics) != 0 {
		t.Fatalf("native diagnostics: %#v", nativeDiagnostics)
	}
	runtime := string(sources.Runtime)
	for _, expected := range []string{"#define ZUMBRA_ENABLE_POSTGRES 1", "#define ZUMBRA_ENABLE_REDIS 1", "libpq-fe.h", "hiredis/hiredis.h"} {
		if !strings.Contains(runtime, expected) {
			t.Fatalf("missing %q from generated native runtime", expected)
		}
	}
}
