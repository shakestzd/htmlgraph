package hooks

import (
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/shakestzd/htmlgraph/internal/models"
)

// SessionEvent holds the data needed to write a <li> element to a session's
// HTML activity log. Kept minimal — only fields that appear in the HTML output.
type SessionEvent struct {
	Timestamp time.Time
	ToolName  string
	Success   bool
	EventID   string
	FeatureID string
	Summary   string
}

// CreateSessionHTML writes the initial session HTML file to
// .htmlgraph/sessions/{session-id}.html. It creates the sessions directory if
// needed. Errors are silently logged via debugLog — HTML is non-critical.
func CreateSessionHTML(projectDir string, s *models.Session) {
	sessDir := filepath.Join(projectDir, ".htmlgraph", "sessions")
	if err := os.MkdirAll(sessDir, 0o755); err != nil {
		debugLog(projectDir, "[session-html] mkdir sessions: %v", err)
		return
	}

	isSubagent := "false"
	if s.IsSubagent {
		isSubagent = "true"
	}

	startedAt := s.CreatedAt.UTC().Format(time.RFC3339)

	var b strings.Builder
	b.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="htmlgraph-version" content="1.0">
    <title>Session `)
	b.WriteString(html.EscapeString(startedAt))
	b.WriteString(`</title>
    <link rel="stylesheet" href="../styles.css">
</head>
<body>
    <article id="`)
	b.WriteString(html.EscapeString(s.SessionID))
	b.WriteString(`"
             data-type="session"
             data-status="active"
             data-agent="`)
	b.WriteString(html.EscapeString(s.AgentAssigned))
	b.WriteString(`"
             data-started-at="`)
	b.WriteString(html.EscapeString(startedAt))
	b.WriteString(`"
             data-event-count="0"
             data-is-subagent="`)
	b.WriteString(isSubagent)
	b.WriteString(`"
             data-start-commit="`)
	b.WriteString(html.EscapeString(s.StartCommit))
	b.WriteString(`">

        <header>
            <h1>Session `)
	b.WriteString(html.EscapeString(startedAt))
	b.WriteString(`</h1>
            <div class="metadata">
                <span class="badge status-active">Active</span>
                <span class="badge">`)
	b.WriteString(html.EscapeString(s.AgentAssigned))
	b.WriteString(`</span>
                <span class="badge">0 events</span>
            </div>
        </header>

        <nav data-graph-edges>
        </nav>

        <section data-activity-log>
            <ol reversed>
            </ol>
        </section>
    </article>
</body>
</html>
`)

	htmlPath := filepath.Join(sessDir, s.SessionID+".html")
	if err := os.WriteFile(htmlPath, []byte(b.String()), 0o644); err != nil {
		debugLog(projectDir, "[session-html] write %s: %v", htmlPath, err)
	}
}

// AppendEventToSessionHTML appends a <li> element to the session's HTML
// activity log. It opens the file with an exclusive flock, reads, modifies,
// and rewrites — preventing lost updates from concurrent hook invocations.
// Errors are silently logged (non-critical path).
func AppendEventToSessionHTML(projectDir, sessionID string, ev SessionEvent) {
	htmlPath := filepath.Join(projectDir, ".htmlgraph", "sessions", sessionID+".html")

	f, err := os.OpenFile(htmlPath, os.O_RDWR, 0o644)
	if err != nil {
		// File doesn't exist — silently ignore (non-critical).
		return
	}
	defer f.Close()

	// Exclusive lock — blocks until no other process holds the lock.
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		debugLog(projectDir, "[session-html] flock %s: %v", htmlPath, err)
		return
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN) //nolint:errcheck

	data, err := io.ReadAll(f)
	if err != nil {
		debugLog(projectDir, "[session-html] read %s: %v", htmlPath, err)
		return
	}

	content := string(data)
	marker := "</ol>"
	idx := strings.LastIndex(content, marker)
	if idx == -1 {
		debugLog(projectDir, "[session-html] no </ol> marker in %s", htmlPath)
		return
	}

	successStr := "true"
	if !ev.Success {
		successStr = "false"
	}

	var li strings.Builder
	li.WriteString(`                <li data-ts="`)
	li.WriteString(ev.Timestamp.UTC().Format(time.RFC3339))
	li.WriteString(`" data-tool="`)
	li.WriteString(html.EscapeString(ev.ToolName))
	li.WriteString(`" data-success="`)
	li.WriteString(successStr)
	li.WriteString(`" data-event-id="`)
	li.WriteString(html.EscapeString(ev.EventID))
	li.WriteString(`"`)
	if ev.FeatureID != "" {
		li.WriteString(` data-feature="`)
		li.WriteString(html.EscapeString(ev.FeatureID))
		li.WriteString(`"`)
	}
	li.WriteString(`>`)
	li.WriteString(html.EscapeString(ev.Summary))
	li.WriteString("</li>\n")

	// Insert the <li> just before </ol>.
	newContent := content[:idx] + li.String() + "            " + content[idx:]

	// Truncate and rewrite in place (we already hold the lock).
	if err := f.Truncate(0); err != nil {
		debugLog(projectDir, "[session-html] truncate %s: %v", htmlPath, err)
		return
	}
	if _, err := f.Seek(0, 0); err != nil {
		debugLog(projectDir, "[session-html] seek %s: %v", htmlPath, err)
		return
	}
	if _, err := f.Write([]byte(newContent)); err != nil {
		debugLog(projectDir, "[session-html] write %s: %v", htmlPath, err)
	}
}

// articleAttrRe matches data attributes on the <article> tag for replacement.
var articleStatusRe = regexp.MustCompile(`data-status="[^"]*"`)
var articleEventCountRe = regexp.MustCompile(`data-event-count="[^"]*"`)
var badgeStatusRe = regexp.MustCompile(`<span class="badge status-[^"]*">[^<]*</span>`)
var badgeEventsRe = regexp.MustCompile(`<span class="badge">\d+ events?</span>`)

// FinalizeSessionHTML updates the session HTML file with completion data:
// sets data-status, adds data-ended-at, and updates data-event-count.
// Errors are silently logged.
func FinalizeSessionHTML(projectDir, sessionID, endedAt, status string, eventCount int) {
	htmlPath := filepath.Join(projectDir, ".htmlgraph", "sessions", sessionID+".html")
	data, err := os.ReadFile(htmlPath)
	if err != nil {
		// File doesn't exist — silently ignore.
		return
	}

	content := string(data)

	// Update data-status.
	content = articleStatusRe.ReplaceAllString(content,
		fmt.Sprintf(`data-status="%s"`, html.EscapeString(status)))

	// Add data-ended-at after data-status on the article tag.
	// Find the article's data-status and insert data-ended-at after it.
	endedAtAttr := fmt.Sprintf(`data-ended-at="%s"`, html.EscapeString(endedAt))
	if !strings.Contains(content, "data-ended-at=") {
		// Insert after data-status in the article tag.
		statusAttr := fmt.Sprintf(`data-status="%s"`, html.EscapeString(status))
		content = strings.Replace(content, statusAttr,
			statusAttr+"\n             "+endedAtAttr, 1)
	}

	// Update data-event-count.
	content = articleEventCountRe.ReplaceAllString(content,
		fmt.Sprintf(`data-event-count="%d"`, eventCount))

	// Update badge status text.
	statusTitle := strings.ToUpper(status[:1]) + status[1:]
	content = badgeStatusRe.ReplaceAllString(content,
		fmt.Sprintf(`<span class="badge status-%s">%s</span>`,
			html.EscapeString(status), html.EscapeString(statusTitle)))

	// Update badge event count text.
	evtWord := "events"
	if eventCount == 1 {
		evtWord = "event"
	}
	content = badgeEventsRe.ReplaceAllString(content,
		fmt.Sprintf(`<span class="badge">%d %s</span>`, eventCount, evtWord))

	if err := os.WriteFile(htmlPath, []byte(content), 0o644); err != nil {
		debugLog(projectDir, "[session-html] finalize %s: %v", htmlPath, err)
	}
}
