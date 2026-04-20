package main

import (
	"strings"
	"testing"
)

// assertEnvContains asserts the env slice contains "key=want". Returns the
// actual value found for key, or "<unset>" when missing.
func assertEnvContains(t *testing.T, env []string, key, want string) {
	t.Helper()
	prefix := key + "="
	for _, kv := range env {
		if strings.HasPrefix(kv, prefix) {
			got := strings.TrimPrefix(kv, prefix)
			if got != want {
				t.Errorf("%s = %q, want %q", key, got, want)
			}
			return
		}
	}
	t.Errorf("%s not set; want %q", key, want)
}

func assertEnvNotSet(t *testing.T, env []string, key string) {
	t.Helper()
	prefix := key + "="
	for _, kv := range env {
		if strings.HasPrefix(kv, prefix) {
			t.Errorf("%s should be unset, got %q", key, strings.TrimPrefix(kv, prefix))
			return
		}
	}
}

func TestBuildClaudeLaunchEnv_OptOutByDefault(t *testing.T) {
	// Explicitly clear any gate + OTEL_* parent env so the test is
	// hermetic regardless of the shell it runs in. (The launcher is
	// expected to pass through any non-empty user OTEL_* values, so a
	// shell that already exports them would otherwise leak into the
	// assertion.)
	t.Setenv("HTMLGRAPH_OTEL_ENABLED", "")
	t.Setenv("CLAUDE_CODE_ENABLE_TELEMETRY", "")
	t.Setenv("CLAUDE_CODE_ENHANCED_TELEMETRY_BETA", "")
	t.Setenv("OTEL_METRICS_EXPORTER", "")
	t.Setenv("OTEL_LOGS_EXPORTER", "")
	t.Setenv("OTEL_TRACES_EXPORTER", "")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")

	env := buildClaudeLaunchEnv("")
	// Gate off: every OTel var is either unset or preserves the empty
	// value from the parent env. Neither state enables telemetry — Claude
	// Code treats empty CLAUDE_CODE_ENABLE_TELEMETRY as disabled.
	for _, key := range []string{
		"CLAUDE_CODE_ENABLE_TELEMETRY",
		"OTEL_METRICS_EXPORTER",
		"OTEL_EXPORTER_OTLP_ENDPOINT",
	} {
		assertEnvEmptyOrUnset(t, env, key)
	}
}

// assertEnvEmptyOrUnset accepts either "missing from env slice" or
// "present with empty value" — both satisfy Claude Code's "telemetry
// disabled" contract, and t.Setenv("KEY", "") produces the latter.
func assertEnvEmptyOrUnset(t *testing.T, env []string, key string) {
	t.Helper()
	prefix := key + "="
	for _, kv := range env {
		if strings.HasPrefix(kv, prefix) {
			if got := strings.TrimPrefix(kv, prefix); got != "" {
				t.Errorf("%s = %q, want empty or unset", key, got)
			}
			return
		}
	}
	// Not in env slice at all — also fine.
}

func TestBuildClaudeLaunchEnv_InjectsWhenEnabled(t *testing.T) {
	t.Setenv("HTMLGRAPH_OTEL_ENABLED", "1")
	// Clear any parent values so we test our defaults.
	t.Setenv("OTEL_METRICS_EXPORTER", "")
	t.Setenv("OTEL_LOGS_EXPORTER", "")
	t.Setenv("OTEL_TRACES_EXPORTER", "")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("OTEL_LOG_TOOL_DETAILS", "")
	t.Setenv("CLAUDE_CODE_ENABLE_TELEMETRY", "")
	t.Setenv("CLAUDE_CODE_ENHANCED_TELEMETRY_BETA", "")

	env := buildClaudeLaunchEnv("")

	assertEnvContains(t, env, "CLAUDE_CODE_ENABLE_TELEMETRY", "1")
	assertEnvContains(t, env, "CLAUDE_CODE_ENHANCED_TELEMETRY_BETA", "1")
	assertEnvContains(t, env, "OTEL_METRICS_EXPORTER", "otlp")
	assertEnvContains(t, env, "OTEL_LOGS_EXPORTER", "otlp")
	assertEnvContains(t, env, "OTEL_TRACES_EXPORTER", "otlp")
	assertEnvContains(t, env, "OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf")
	assertEnvContains(t, env, "OTEL_EXPORTER_OTLP_ENDPOINT", "http://127.0.0.1:4318")
	assertEnvContains(t, env, "OTEL_LOG_TOOL_DETAILS", "1")
}

func TestBuildClaudeLaunchEnv_RespectsUserOverrides(t *testing.T) {
	// If the user already set OTEL_EXPORTER_OTLP_ENDPOINT or changed the
	// exporter, we must not clobber those choices.
	t.Setenv("HTMLGRAPH_OTEL_ENABLED", "1")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "https://custom.example.com:4318")
	t.Setenv("OTEL_METRICS_EXPORTER", "console")
	t.Setenv("OTEL_LOG_TOOL_DETAILS", "0")

	env := buildClaudeLaunchEnv("")

	assertEnvContains(t, env, "OTEL_EXPORTER_OTLP_ENDPOINT", "https://custom.example.com:4318")
	assertEnvContains(t, env, "OTEL_METRICS_EXPORTER", "console")
	assertEnvContains(t, env, "OTEL_LOG_TOOL_DETAILS", "0")
	// But flags that control telemetry activation can be added by us if
	// unset — user is allowed to rely on the launcher to turn things on.
	assertEnvContains(t, env, "CLAUDE_CODE_ENABLE_TELEMETRY", "1")
}

func TestBuildClaudeLaunchEnv_WorktreeProjectDir(t *testing.T) {
	t.Setenv("HTMLGRAPH_OTEL_ENABLED", "")
	t.Setenv("HTMLGRAPH_PROJECT_DIR", "/old/value")
	env := buildClaudeLaunchEnv("/worktree/main/.htmlgraph")
	assertEnvContains(t, env, "HTMLGRAPH_PROJECT_DIR", "/worktree/main/.htmlgraph")
}

func TestBuildClaudeLaunchEnv_EndpointFromCustomHostPort(t *testing.T) {
	t.Setenv("HTMLGRAPH_OTEL_ENABLED", "1")
	t.Setenv("HTMLGRAPH_OTEL_BIND", "0.0.0.0")
	t.Setenv("HTMLGRAPH_OTEL_HTTP_PORT", "14318")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	env := buildClaudeLaunchEnv("")
	// 0.0.0.0 bind host maps to 127.0.0.1 for the outbound direction —
	// child processes should never send to 0.0.0.0.
	assertEnvContains(t, env, "OTEL_EXPORTER_OTLP_ENDPOINT", "http://127.0.0.1:14318")
}

func TestIsTruthy(t *testing.T) {
	for _, s := range []string{"1", "true", "TRUE", "yes", "on"} {
		if !isTruthy(s) {
			t.Errorf("isTruthy(%q) = false, want true", s)
		}
	}
	for _, s := range []string{"", "0", "false", "no", "off", "maybe"} {
		if isTruthy(s) {
			t.Errorf("isTruthy(%q) = true, want false", s)
		}
	}
}
