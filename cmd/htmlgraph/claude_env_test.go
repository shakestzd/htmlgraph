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

func TestBuildClaudeLaunchEnv_ExplicitOptOut(t *testing.T) {
	// Clear parent OTel vars so the test is hermetic.
	clearOtelEnv(t)

	// "0" explicitly disables — no OTel vars should be injected.
	t.Setenv("HTMLGRAPH_OTEL_ENABLED", "0")
	env := buildClaudeLaunchEnv("")
	for _, key := range []string{
		"CLAUDE_CODE_ENABLE_TELEMETRY",
		"OTEL_METRICS_EXPORTER",
		"OTEL_EXPORTER_OTLP_ENDPOINT",
	} {
		assertEnvEmptyOrUnset(t, env, key)
	}

	// Also test other opt-out values.
	for _, val := range []string{"false", "no", "off"} {
		t.Setenv("HTMLGRAPH_OTEL_ENABLED", val)
		env = buildClaudeLaunchEnv("")
		assertEnvEmptyOrUnset(t, env, "CLAUDE_CODE_ENABLE_TELEMETRY")
	}
}

func TestBuildClaudeLaunchEnv_DefaultOn(t *testing.T) {
	// An unset or empty HTMLGRAPH_OTEL_ENABLED should enable OTel (default-on).
	clearOtelEnv(t)
	t.Setenv("HTMLGRAPH_OTEL_ENABLED", "") // explicitly unset

	env := buildClaudeLaunchEnv("")
	assertEnvContains(t, env, "CLAUDE_CODE_ENABLE_TELEMETRY", "1")
	assertEnvContains(t, env, "OTEL_TRACES_EXPORTER", "otlp")
	assertEnvContains(t, env, "OTEL_EXPORTER_OTLP_ENDPOINT", "http://127.0.0.1:4318")
}

// clearOtelEnv clears all OTel-related environment variables for test isolation.
func clearOtelEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"HTMLGRAPH_OTEL_ENABLED",
		"CLAUDE_CODE_ENABLE_TELEMETRY",
		"CLAUDE_CODE_ENHANCED_TELEMETRY_BETA",
		"OTEL_METRICS_EXPORTER",
		"OTEL_LOGS_EXPORTER",
		"OTEL_TRACES_EXPORTER",
		"OTEL_EXPORTER_OTLP_PROTOCOL",
		"OTEL_EXPORTER_OTLP_ENDPOINT",
		"OTEL_LOG_TOOL_DETAILS",
	} {
		t.Setenv(key, "")
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
	clearOtelEnv(t)
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

func TestIsExplicitlyDisabled(t *testing.T) {
	// Values that should return true (opt-out).
	for _, s := range []string{"0", "false", "FALSE", "no", "off"} {
		if !isExplicitlyDisabled(s) {
			t.Errorf("isExplicitlyDisabled(%q) = false, want true", s)
		}
	}
	// Values that should return false (not opted out — default-on applies).
	for _, s := range []string{"", "1", "true", "yes", "random"} {
		if isExplicitlyDisabled(s) {
			t.Errorf("isExplicitlyDisabled(%q) = true, want false", s)
		}
	}
	// Whitespace variants of opt-out values should also be recognized.
	for _, s := range []string{" 0", "false ", "  no  ", "\toff\t"} {
		if !isExplicitlyDisabled(s) {
			t.Errorf("isExplicitlyDisabled(%q) = false, want true (whitespace should be trimmed)", s)
		}
	}
}
