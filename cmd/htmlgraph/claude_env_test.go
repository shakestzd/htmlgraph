package main

import (
	"os"
	"strconv"
	"strings"
	"testing"

	otelreceiver "github.com/shakestzd/htmlgraph/internal/otel/receiver"
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
	env := buildClaudeLaunchEnv("", nil)
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
		env = buildClaudeLaunchEnv("", nil)
		assertEnvEmptyOrUnset(t, env, "CLAUDE_CODE_ENABLE_TELEMETRY")
	}
}

func TestBuildClaudeLaunchEnv_DefaultOn(t *testing.T) {
	// An unset or empty HTMLGRAPH_OTEL_ENABLED should enable OTel (default-on).
	clearOtelEnv(t)
	t.Setenv("HTMLGRAPH_OTEL_ENABLED", "") // explicitly unset

	env := buildClaudeLaunchEnv("", nil)
	assertEnvContains(t, env, "CLAUDE_CODE_ENABLE_TELEMETRY", "1")
	assertEnvContains(t, env, "OTEL_TRACES_EXPORTER", "otlp")
	// The endpoint should be derived from cwd (since no explicit projectDir or env vars are set).
	// Verify it's set and contains the port for the current working directory.
	expectedPort := otelreceiver.PortForProject(effectiveProjectDir(""))
	expectedEndpoint := "http://127.0.0.1:" + strconv.Itoa(expectedPort)
	assertEnvContains(t, env, "OTEL_EXPORTER_OTLP_ENDPOINT", expectedEndpoint)
}

// clearOtelEnv clears all OTel-related environment variables for test isolation.
func clearOtelEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"HTMLGRAPH_OTEL_ENABLED",
		"HTMLGRAPH_OTEL_HTTP_PORT",
		"HTMLGRAPH_OTEL_BIND",
		"HTMLGRAPH_PROJECT_DIR",
		"CLAUDE_PROJECT_DIR",
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
	t.Setenv("HTMLGRAPH_OTEL_HTTP_PORT", "")
	t.Setenv("HTMLGRAPH_PROJECT_DIR", "")
	t.Setenv("CLAUDE_PROJECT_DIR", "")
	t.Setenv("OTEL_METRICS_EXPORTER", "")
	t.Setenv("OTEL_LOGS_EXPORTER", "")
	t.Setenv("OTEL_TRACES_EXPORTER", "")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("OTEL_LOG_TOOL_DETAILS", "")
	t.Setenv("CLAUDE_CODE_ENABLE_TELEMETRY", "")
	t.Setenv("CLAUDE_CODE_ENHANCED_TELEMETRY_BETA", "")

	env := buildClaudeLaunchEnv("", nil)

	assertEnvContains(t, env, "CLAUDE_CODE_ENABLE_TELEMETRY", "1")
	assertEnvContains(t, env, "CLAUDE_CODE_ENHANCED_TELEMETRY_BETA", "1")
	assertEnvContains(t, env, "OTEL_METRICS_EXPORTER", "otlp")
	assertEnvContains(t, env, "OTEL_LOGS_EXPORTER", "otlp")
	assertEnvContains(t, env, "OTEL_TRACES_EXPORTER", "otlp")
	assertEnvContains(t, env, "OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf")
	// Endpoint is derived from cwd fallback (since no explicit projectDir or env vars are set).
	expectedPort := otelreceiver.PortForProject(effectiveProjectDir(""))
	expectedEndpoint := "http://127.0.0.1:" + strconv.Itoa(expectedPort)
	assertEnvContains(t, env, "OTEL_EXPORTER_OTLP_ENDPOINT", expectedEndpoint)
	assertEnvContains(t, env, "OTEL_LOG_TOOL_DETAILS", "1")
}

func TestBuildClaudeLaunchEnv_RespectsUserOverrides(t *testing.T) {
	// The launcher respects user overrides for OTEL_METRICS_EXPORTER and
	// OTEL_LOG_TOOL_DETAILS via addIfUnset. However, OTEL_EXPORTER_OTLP_ENDPOINT
	// is NOT user-overrideable — it's always set by the launcher to match the
	// receiver's per-project port. Users who need a custom receiver should
	// steer via HTMLGRAPH_OTEL_HTTP_PORT / HTMLGRAPH_OTEL_BIND.
	t.Setenv("HTMLGRAPH_OTEL_ENABLED", "1")
	t.Setenv("HTMLGRAPH_PROJECT_DIR", "")
	t.Setenv("CLAUDE_PROJECT_DIR", "")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "https://custom.example.com:4318")
	t.Setenv("OTEL_METRICS_EXPORTER", "console")
	t.Setenv("OTEL_LOG_TOOL_DETAILS", "0")

	env := buildClaudeLaunchEnv("", nil)

	// OTEL_EXPORTER_OTLP_ENDPOINT is overridden by the launcher — derived from cwd fallback.
	expectedPort := otelreceiver.PortForProject(effectiveProjectDir(""))
	expectedEndpoint := "http://127.0.0.1:" + strconv.Itoa(expectedPort)
	assertEnvContains(t, env, "OTEL_EXPORTER_OTLP_ENDPOINT", expectedEndpoint)
	// But other OTEL_* vars respect user overrides.
	assertEnvContains(t, env, "OTEL_METRICS_EXPORTER", "console")
	assertEnvContains(t, env, "OTEL_LOG_TOOL_DETAILS", "0")
	// Telemetry activation flags can be added by us if unset.
	assertEnvContains(t, env, "CLAUDE_CODE_ENABLE_TELEMETRY", "1")
}

func TestBuildClaudeLaunchEnv_WorktreeProjectDir(t *testing.T) {
	clearOtelEnv(t)
	t.Setenv("HTMLGRAPH_PROJECT_DIR", "/old/value")
	env := buildClaudeLaunchEnv("/worktree/main/.htmlgraph", nil)
	assertEnvContains(t, env, "HTMLGRAPH_PROJECT_DIR", "/worktree/main/.htmlgraph")
}

func TestBuildClaudeLaunchEnv_EndpointFromCustomHostPort(t *testing.T) {
	t.Setenv("HTMLGRAPH_OTEL_ENABLED", "1")
	t.Setenv("HTMLGRAPH_OTEL_BIND", "0.0.0.0")
	t.Setenv("HTMLGRAPH_OTEL_HTTP_PORT", "14318")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	env := buildClaudeLaunchEnv("", nil)
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

func TestBuildClaudeLaunchEnv_OverridesStaleOTELEndpoint(t *testing.T) {
	// When the parent env has OTEL_EXPORTER_OTLP_ENDPOINT from a prior
	// session (with a different port), the launcher's computed endpoint
	// should override it. This ensures spans aren't silently dropped.
	clearOtelEnv(t)
	t.Setenv("HTMLGRAPH_OTEL_ENABLED", "1")
	// Simulate a stale port from a prior session with a different hash.
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://127.0.0.1:9999")

	env := buildClaudeLaunchEnv("", nil)
	// The computed endpoint should override the inherited 9999. It's derived from cwd fallback.
	expectedPort := otelreceiver.PortForProject(effectiveProjectDir(""))
	expectedEndpoint := "http://127.0.0.1:" + strconv.Itoa(expectedPort)
	assertEnvContains(t, env, "OTEL_EXPORTER_OTLP_ENDPOINT", expectedEndpoint)
}

func TestBuildClaudeLaunchEnv_ResolvesFromCLAUDEProjectDir(t *testing.T) {
	// When htmlgraphProjectDir arg is empty (non-worktree case), the launcher
	// should resolve the effective projectDir from CLAUDE_PROJECT_DIR env var
	// and derive the OTLP port hash from it, not the base port 4318.
	// This is the key regression test for bug-e5c2df6d.
	clearOtelEnv(t)
	t.Setenv("HTMLGRAPH_OTEL_ENABLED", "1")
	t.Setenv("CLAUDE_PROJECT_DIR", "/workspaces/htmlgraph")

	env := buildClaudeLaunchEnv("", nil)

	// HTMLGRAPH_PROJECT_DIR should be set to the resolved projectDir.
	assertEnvContains(t, env, "HTMLGRAPH_PROJECT_DIR", "/workspaces/htmlgraph")

	// OTLP endpoint should use the hashed port, not the base port 4318.
	// We verify it's NOT 4318 (the bug symptom) and matches what the receiver
	// would compute for the same path.
	prefix := "OTEL_EXPORTER_OTLP_ENDPOINT="
	for _, kv := range env {
		if strings.HasPrefix(kv, prefix) {
			got := strings.TrimPrefix(kv, prefix)
			// The endpoint must not be the base port — that's the bug symptom.
			if got == "http://127.0.0.1:4318" {
				t.Errorf("OTEL_EXPORTER_OTLP_ENDPOINT = %q; bug-e5c2df6d: should use hashed port, not base 4318", got)
			}
			// Verify it contains the hashed port from the same projectDir.
			expectedPort := otelreceiver.PortForProject("/workspaces/htmlgraph")
			expectedEndpoint := "http://127.0.0.1:" + strconv.Itoa(expectedPort)
			if got != expectedEndpoint {
				t.Errorf("OTEL_EXPORTER_OTLP_ENDPOINT = %q, want %q", got, expectedEndpoint)
			}
			return
		}
	}
	t.Errorf("OTEL_EXPORTER_OTLP_ENDPOINT not set")
}

func TestBuildClaudeLaunchEnv_ResolvesFromHTMLGRAPHProjectDirEnv(t *testing.T) {
	// Test the second priority in the resolution chain: HTMLGRAPH_PROJECT_DIR env var.
	clearOtelEnv(t)
	t.Setenv("HTMLGRAPH_OTEL_ENABLED", "1")
	t.Setenv("CLAUDE_PROJECT_DIR", "") // empty first priority
	t.Setenv("HTMLGRAPH_PROJECT_DIR", "/workspaces/htmlgraph")

	env := buildClaudeLaunchEnv("", nil)

	// Should resolve from HTMLGRAPH_PROJECT_DIR and derive the hashed port.
	prefix := "OTEL_EXPORTER_OTLP_ENDPOINT="
	for _, kv := range env {
		if strings.HasPrefix(kv, prefix) {
			got := strings.TrimPrefix(kv, prefix)
			if got == "http://127.0.0.1:4318" {
				t.Errorf("OTEL_EXPORTER_OTLP_ENDPOINT = %q; should use hashed port from HTMLGRAPH_PROJECT_DIR env", got)
			}
			return
		}
	}
}

func TestEffectiveProjectDir(t *testing.T) {
	// Test the resolution priority chain directly.

	// Priority 1: explicit arg wins.
	if got := effectiveProjectDir("/explicit"); got != "/explicit" {
		t.Errorf("effectiveProjectDir(\"/explicit\") = %q, want /explicit", got)
	}

	// Priority 2: CLAUDE_PROJECT_DIR env when arg empty.
	oldClaude := os.Getenv("CLAUDE_PROJECT_DIR")
	oldHtmlgraph := os.Getenv("HTMLGRAPH_PROJECT_DIR")
	defer func() {
		if oldClaude != "" {
			os.Setenv("CLAUDE_PROJECT_DIR", oldClaude)
		} else {
			os.Unsetenv("CLAUDE_PROJECT_DIR")
		}
		if oldHtmlgraph != "" {
			os.Setenv("HTMLGRAPH_PROJECT_DIR", oldHtmlgraph)
		} else {
			os.Unsetenv("HTMLGRAPH_PROJECT_DIR")
		}
	}()

	os.Setenv("CLAUDE_PROJECT_DIR", "/from-claude")
	os.Setenv("HTMLGRAPH_PROJECT_DIR", "/from-htmlgraph") // also set, but should lose
	if got := effectiveProjectDir(""); got != "/from-claude" {
		t.Errorf("effectiveProjectDir(\"\") with CLAUDE_PROJECT_DIR set = %q, want /from-claude", got)
	}

	// Priority 3: HTMLGRAPH_PROJECT_DIR env when CLAUDE_PROJECT_DIR empty.
	os.Setenv("CLAUDE_PROJECT_DIR", "")
	if got := effectiveProjectDir(""); got != "/from-htmlgraph" {
		t.Errorf("effectiveProjectDir(\"\") with only HTMLGRAPH_PROJECT_DIR set = %q, want /from-htmlgraph", got)
	}

	// Priority 4: os.Getwd() fallback (can't easily stub, so just verify it returns non-empty).
	os.Unsetenv("CLAUDE_PROJECT_DIR")
	os.Unsetenv("HTMLGRAPH_PROJECT_DIR")
	if got := effectiveProjectDir(""); got == "" {
		t.Errorf("effectiveProjectDir(\"\") with no env vars should fallback to os.Getwd(), got empty string")
	}
}
