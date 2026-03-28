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
