package sdk

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"time"
)

// prefixes maps node types to their short ID prefix.
// Matches Python htmlgraph.ids.PREFIXES.
var prefixes = map[string]string{
	"feature": "feat",
	"bug":     "bug",
	"chore":   "chr",
	"spike":   "spk",
	"epic":    "epc",
	"session": "sess",
	"track":   "trk",
	"phase":   "phs",
	"agent":   "agt",
	"spec":    "spec",
	"plan":    "plan",
	"event":   "evt",
}

// generateID creates a collision-resistant ID matching the Python format.
//
// Format: {prefix}-{hex8} (e.g., feat-a1b2c3d4)
//
// The hash combines: title + UTC timestamp (nanosecond) + 4 random bytes.
func generateID(nodeType, title string) string {
	prefix, ok := prefixes[nodeType]
	if !ok && len(nodeType) >= 4 {
		prefix = nodeType[:4]
	} else if !ok {
		prefix = nodeType
	}

	ts := time.Now().UTC().Format(time.RFC3339Nano)
	entropy := make([]byte, 4)
	_, _ = rand.Read(entropy) // crypto/rand never errors on supported platforms

	content := append([]byte(fmt.Sprintf("%s:%s", title, ts)), entropy...)
	hash := sha256.Sum256(content)

	return fmt.Sprintf("%s-%x", prefix, hash[:4])
}
