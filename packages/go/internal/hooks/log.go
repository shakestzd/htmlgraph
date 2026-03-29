package hooks

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// debugLog writes a diagnostic message to .htmlgraph/debug.log if it can be resolved.
// Silently no-ops if the project dir can't be found or the file can't be opened.
func debugLog(projectDir, format string, args ...any) {
	if projectDir == "" {
		return
	}
	logPath := filepath.Join(projectDir, ".htmlgraph", "debug.log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(f, "%s %s\n", time.Now().Format("2006-01-02T15:04:05"), msg)
}

// LogError logs a handler error with structured context (handler name, session ID).
// It resolves the project dir from env/CWD so it can be called from cmd/htmlgraph
// where projectDir is not yet known. Silently no-ops if the project cannot be found.
func LogError(handler, sessionID, msg string) {
	projectDir := resolveLogDir()
	if projectDir == "" {
		return
	}
	var line string
	if sessionID != "" {
		line = fmt.Sprintf("[error] handler=%s session=%s: %s", handler, sessionID[:minSessionLen(sessionID)], msg)
	} else {
		line = fmt.Sprintf("[error] handler=%s: %s", handler, msg)
	}
	debugLog(projectDir, line)
}

// resolveLogDir finds the project directory for logging by checking env then CWD walk-up.
func resolveLogDir() string {
	cwd, _ := os.Getwd()
	return ResolveProjectDir(cwd)
}

// minSessionLen returns min(8, len(s)) for safe session ID truncation in log messages.
func minSessionLen(s string) int {
	if len(s) < 8 {
		return len(s)
	}
	return 8
}
