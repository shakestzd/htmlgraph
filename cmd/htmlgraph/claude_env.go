package main

import (
	"os"
	"strconv"
	"strings"

	otelreceiver "github.com/shakestzd/htmlgraph/internal/otel/receiver"
)

// buildClaudeLaunchEnv returns the environment vector for a spawned
// `claude` process. It always starts from os.Environ() (so the child
// inherits the user's shell env) and layers HtmlGraph-specific overrides
// on top:
//
//   1. HTMLGRAPH_PROJECT_DIR — set when the launcher runs inside a
//      worktree, so hooks resolve to the main .htmlgraph/ directory.
//   2. OTel exporter vars — when HTMLGRAPH_OTEL_ENABLED=1 in the parent
//      env, we wire Claude's OTLP exporter at our receiver so every turn
//      captures spans/logs/metrics into otel_signals. User-set OTel vars
//      win: we never clobber an explicit OTEL_* choice.
//
// htmlgraphProjectDir is the empty string when no override is needed
// (not in a worktree). Pass it explicitly rather than deriving it from
// opts so the helper stays easy to unit-test.
func buildClaudeLaunchEnv(htmlgraphProjectDir string) []string {
	env := os.Environ()

	if htmlgraphProjectDir != "" {
		env = setOrReplaceEnv(env, "HTMLGRAPH_PROJECT_DIR", htmlgraphProjectDir)
	}

	// Gate OTel injection on the same env var that controls the receiver
	// in `htmlgraph serve` (Phase 1). Keeping one toggle avoids split-brain
	// where the receiver is running but the launcher doesn't point Claude
	// at it, or vice versa.
	if !isTruthy(os.Getenv("HTMLGRAPH_OTEL_ENABLED")) {
		return env
	}

	endpoint := otelEndpointFromEnv()
	// User-set values always win — only add our default if missing.
	env = addIfUnset(env, "CLAUDE_CODE_ENABLE_TELEMETRY", "1")
	env = addIfUnset(env, "CLAUDE_CODE_ENHANCED_TELEMETRY_BETA", "1")
	env = addIfUnset(env, "OTEL_METRICS_EXPORTER", "otlp")
	env = addIfUnset(env, "OTEL_LOGS_EXPORTER", "otlp")
	env = addIfUnset(env, "OTEL_TRACES_EXPORTER", "otlp")
	env = addIfUnset(env, "OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf")
	env = addIfUnset(env, "OTEL_EXPORTER_OTLP_ENDPOINT", endpoint)
	// Tool details include bash commands, skill names, MCP tool names —
	// non-sensitive by default. Turn off by setting to "0" before launch.
	env = addIfUnset(env, "OTEL_LOG_TOOL_DETAILS", "1")
	return env
}

// otelEndpointFromEnv derives the OTLP HTTP endpoint to point Claude at,
// honoring the same HTMLGRAPH_OTEL_BIND and HTMLGRAPH_OTEL_HTTP_PORT vars
// that the receiver's LoadConfigFromEnv reads. Keeps launcher and receiver
// symmetric without needing to duplicate the defaults.
func otelEndpointFromEnv() string {
	cfg := otelreceiver.LoadConfigFromEnv("")
	host := cfg.BindHost
	if host == "" || host == "0.0.0.0" {
		// "Listen on all interfaces" maps to "export to loopback" for the
		// outbound direction — a child on the same host should never send
		// to 0.0.0.0.
		host = "127.0.0.1"
	}
	port := cfg.HTTPPort
	if port == 0 {
		port = 4318
	}
	return "http://" + host + ":" + strconv.Itoa(port)
}

// addIfUnset appends key=value to env only when key is not already set
// to a non-empty value. This keeps non-empty user overrides authoritative
// while filling gaps with our defaults. An empty string is treated as
// "unset" because Claude Code itself sets several OTEL_* vars to empty
// when spawning subprocesses (observed empirically in the TRACEPARENT
// validation run) — if we respected those as authoritative choices, we'd
// never enable telemetry in a nested launcher.
func addIfUnset(env []string, key, value string) []string {
	prefix := key + "="
	for i, kv := range env {
		if strings.HasPrefix(kv, prefix) {
			if len(kv) > len(prefix) {
				return env // non-empty user value wins
			}
			// Empty value — treat as unset and overwrite in place.
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

// setOrReplaceEnv replaces the value of key if present, appending
// otherwise. Different from addIfUnset: used for vars where the launcher's
// authoritative intent should override any inherited value (e.g. worktree
// project dir override).
func setOrReplaceEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i, kv := range env {
		if strings.HasPrefix(kv, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

// isTruthy matches the parsing used by receiver.LoadConfigFromEnv.
// Kept local here to avoid exporting a helper from the receiver package
// for one env-var check.
func isTruthy(s string) bool {
	switch s {
	case "1", "true", "TRUE", "yes", "on":
		return true
	}
	return false
}
