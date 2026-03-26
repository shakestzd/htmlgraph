package templates

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/shakestzd/htmlgraph/internal/models"
)

// formatTime formats a time.Time to ISO-8601 without timezone for consistency
// with the Python output that uses isoformat().
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	// Python's datetime.isoformat() produces "2026-03-26T09:47:14.165536"
	// for naive datetimes. Match that format.
	return t.Format("2006-01-02T15:04:05.999999")
}

// titleCase converts "in-progress" to "In Progress", "medium" to "Medium".
func titleCase(s string) string {
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
	}
	return strings.Join(words, " ")
}

// statusDisplay converts status string to display format matching Python.
// "in-progress" -> "In Progress", "todo" -> "Todo".
func statusDisplay(status string) string {
	return titleCase(status)
}

// relationshipDisplay converts edge relationship to display form.
// "implemented-in" -> "Implemented-In", "blocked_by" -> "Blocked By".
func relationshipDisplay(rel string) string {
	// Python uses: rel_type.replace("_", " ").title()
	// But the HTML shows "Implemented-In:" with hyphens preserved in display
	s := strings.ReplaceAll(rel, "_", " ")
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// edgeHref builds the href attribute for an edge link.
func edgeHref(e models.Edge) string {
	targetID := e.TargetID
	// Edges reference files in sibling or same directory
	return targetID + ".html"
}

// edgeTitle returns the display text for an edge link.
func edgeTitle(e models.Edge) string {
	if e.Title != "" {
		return e.Title
	}
	return e.TargetID
}

// spikeTypeOrDefault returns the spike_type from the node or "general".
func spikeTypeOrDefault(node *models.Node) string {
	if node.SpikeSubtype != "" {
		return node.SpikeSubtype
	}
	// Check properties for spike_type
	if node.Properties != nil {
		if v, ok := node.Properties["spike_type"]; ok {
			return fmt.Sprintf("%v", v)
		}
	}
	return "general"
}

// parseJSONStringSlice attempts to decode a json.RawMessage as []string.
func parseJSONStringSlice(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var result []string
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil
	}
	return result
}
