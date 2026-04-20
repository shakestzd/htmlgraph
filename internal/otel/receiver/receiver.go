package receiver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/shakestzd/htmlgraph/internal/otel/adapter"
)

// Config controls the embedded OTLP receiver that ships inside
// `htmlgraph serve`. Defaults match the Phase 1 posture: opt-in,
// loopback-only, HTTP-only (no gRPC).
type Config struct {
	// Enabled turns the receiver on. When false, Start is a no-op.
	// Default: false (v1 ships opt-in).
	Enabled bool

	// BindHost is the listen address. Default: 127.0.0.1. Loopback
	// prevents exposing raw session signals on the LAN by accident.
	BindHost string

	// HTTPPort is the OTLP/HTTP port. Default: 4318 (per OTel spec).
	// Set to 0 to disable the HTTP listener entirely.
	HTTPPort int

	// DBPath is the SQLite file path for persistence. If empty, the
	// receiver assumes it's been initialized inline — callers pass
	// this when embedding inside `htmlgraph serve`.
	DBPath string
}

// LoadConfigFromEnv reads HTMLGRAPH_OTEL_* env vars and returns a
// Config with sensible defaults. Calling with no env set yields a
// disabled receiver.
//
// Recognized vars:
//   HTMLGRAPH_OTEL_ENABLED    (0/1 or true/false; default 0)
//   HTMLGRAPH_OTEL_BIND       (default 127.0.0.1)
//   HTMLGRAPH_OTEL_HTTP_PORT  (default 4318; set 0 to disable)
func LoadConfigFromEnv(dbPath string) Config {
	c := Config{
		Enabled:  parseBool(os.Getenv("HTMLGRAPH_OTEL_ENABLED")),
		BindHost: envOr("HTMLGRAPH_OTEL_BIND", "127.0.0.1"),
		HTTPPort: parseIntDefault(os.Getenv("HTMLGRAPH_OTEL_HTTP_PORT"), 4318),
		DBPath:   dbPath,
	}
	return c
}

// Receiver wires the HTTP handler, writer, and adapter registry into
// a lifecycle object that `htmlgraph serve` can Start/Stop.
//
// Typical usage:
//
//	r, err := receiver.New(cfg)
//	if err != nil { ... }
//	if err := r.Start(ctx); err != nil { ... }
//	defer r.Stop(ctx)
type Receiver struct {
	cfg      Config
	writer   *Writer
	registry *adapter.Registry
	handler  *HTTPHandler
	srv      *http.Server

	mu      sync.Mutex
	started bool
}

// New constructs a Receiver with the default adapter set. Returns an
// unconfigured Receiver when cfg.Enabled is false — Start will no-op.
func New(cfg Config) (*Receiver, error) {
	r := &Receiver{cfg: cfg, registry: adapter.NewRegistry()}
	r.registry.Register(adapter.NewClaudeAdapter())
	// Codex and Gemini adapters register in later phases.

	if !cfg.Enabled {
		return r, nil
	}
	if cfg.DBPath == "" {
		return nil, errors.New("DBPath required when Enabled")
	}
	w, err := NewWriter(cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("otel writer: %w", err)
	}
	r.writer = w
	r.handler = NewHTTPHandler(r.registry, w)
	return r, nil
}

// Registry exposes the adapter registry so tests can register fakes
// without reconstructing the receiver.
func (r *Receiver) Registry() *adapter.Registry { return r.registry }

// Writer exposes the writer for integration tests and diagnostics.
func (r *Receiver) Writer() *Writer { return r.writer }

// Handler exposes the HTTP handler so it can be mounted on an existing
// mux (preferred) instead of a standalone server (fallback).
func (r *Receiver) Handler() *HTTPHandler { return r.handler }

// Start launches the OTLP HTTP listener. No-op if Enabled is false.
// Start is idempotent; concurrent calls return the same running state.
//
// When HTTPPort is 0 the listener is skipped — useful when callers
// mount the handler on their own mux (e.g. inside htmlgraph serve,
// which already runs an HTTP server).
func (r *Receiver) Start(ctx context.Context) error {
	if !r.cfg.Enabled {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.started {
		return nil
	}
	if r.cfg.HTTPPort > 0 {
		mux := http.NewServeMux()
		r.handler.Register(mux)
		addr := net.JoinHostPort(r.cfg.BindHost, strconv.Itoa(r.cfg.HTTPPort))
		r.srv = &http.Server{
			Addr:              addr,
			Handler:           mux,
			ReadHeaderTimeout: 10 * time.Second,
		}
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			return fmt.Errorf("otel listen %s: %w", addr, err)
		}
		go func() {
			if err := r.srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
				// Receiver failures are non-fatal to the rest of serve —
				// log and carry on so the dashboard stays up.
				fmt.Fprintf(os.Stderr, "otel receiver stopped: %v\n", err)
			}
		}()
	}
	r.started = true
	return nil
}

// Stop gracefully shuts down the HTTP listener and writer. Safe to
// call multiple times. Blocks up to 10 seconds for in-flight requests
// to complete.
func (r *Receiver) Stop(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.started {
		if r.writer != nil {
			return r.writer.Close()
		}
		return nil
	}
	var firstErr error
	if r.srv != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if err := r.srv.Shutdown(shutdownCtx); err != nil {
			firstErr = err
		}
	}
	if r.writer != nil {
		if err := r.writer.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	r.started = false
	return firstErr
}

func parseBool(s string) bool {
	switch s {
	case "1", "true", "TRUE", "yes", "on":
		return true
	default:
		return false
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
