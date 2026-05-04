package workitem

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"embed"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/shakestzd/htmlgraph/internal/models"
)

// featureWriteMu serialises concurrent WriteNodeHTML calls for the same feature
// ID in a single process. Keyed by node ID (string) → *sync.Mutex.
// This prevents lost-update races when two goroutines write the same HTML file
// concurrently (e.g. two compliance auto invocations in tests).
var featureWriteMu sync.Map

// atomicWriteCounter provides a unique sequence number per atomic write call,
// used to make temp filenames unique even when called from multiple goroutines
// in the same process (which all share the same PID).
var atomicWriteCounter atomic.Int64

// --- ID generation -----------------------------------------------------------

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

// GenerateID creates a collision-resistant ID matching the Python format.
//
// Format: {prefix}-{hex8} (e.g., feat-a1b2c3d4)
//
// The hash combines: title + UTC timestamp (nanosecond) + 4 random bytes.
func GenerateID(nodeType, title string) string {
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

// --- HTML writing ------------------------------------------------------------

//go:embed templates/node.gohtml
var templateFS embed.FS

var nodeTmpl = template.Must(
	template.ParseFS(templateFS, "templates/node.gohtml"),
)

// WriteNodeHTML serialises a Node to the canonical HtmlGraph HTML format and
// writes it to the collection directory.  The output MUST be parseable by
// htmlparse.ParseFile to ensure round-trip fidelity.
//
// Writes are atomic: the content is rendered to a temp file, fsynced, then
// renamed over the target path (POSIX rename is atomic). A per-node mutex
// serialises concurrent in-process writes for the same node ID to prevent
// lost-update races.
//
// Returns the absolute path of the written file.
func WriteNodeHTML(dir string, node *models.Node) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create dir %s: %w", dir, err)
	}

	// Acquire per-node mutex to serialize concurrent writes for the same node.
	muVal, _ := featureWriteMu.LoadOrStore(node.ID, &sync.Mutex{})
	mu := muVal.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()

	path := filepath.Join(dir, node.ID+".html")
	html, err := renderNodeHTML(node)
	if err != nil {
		return "", fmt.Errorf("render %s: %w", node.ID, err)
	}

	if err := atomicWriteFile(path, []byte(html), 0o644); err != nil {
		return "", fmt.Errorf("write %s: %w", path, err)
	}
	return path, nil
}

// atomicWriteFile writes data to path atomically: it writes to a temp file in
// the same directory, calls Sync to flush to storage, then renames the temp
// file over the target. POSIX rename is atomic within the same filesystem.
func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	seq := atomicWriteCounter.Add(1)
	tmp := fmt.Sprintf("%s.tmp.%d.%d", path, os.Getpid(), seq)

	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return fmt.Errorf("open temp file: %w", err)
	}

	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tmp)
		return fmt.Errorf("write temp file: %w", err)
	}

	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmp)
		return fmt.Errorf("sync temp file: %w", err)
	}

	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("close temp file: %w", err)
	}

	_ = dir // ensure dir is always used (for documentation clarity)
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("rename temp to target: %w", err)
	}

	return nil
}

// renderNodeHTML produces the full HTML document for a node using
// html/template with an embedded .gohtml template.
func renderNodeHTML(n *models.Node) (string, error) {
	data := newNodeTemplateData(n)
	var buf bytes.Buffer
	if err := nodeTmpl.ExecuteTemplate(&buf, "node.gohtml", data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// nodeTemplateData holds all pre-computed values for the node template.
// Fields that contain trusted HTML use template.HTML to bypass auto-escaping.
type nodeTemplateData struct {
	ID               string
	Title            string
	Type             string
	Status           string
	Priority         string
	CreatedAt        string
	UpdatedAt        string
	AgentAssigned    string
	TrackID          string
	SpikeSubtype     string
	ClaimedAt        string
	ClaimedBySession string

	StatusLabel   string
	PriorityLabel string

	HasEdges   bool
	EdgeGroups []edgeGroupData

	HasSteps bool
	Steps    []stepData

	HasContent     bool
	TrustedContent template.HTML
}

// edgeGroupData holds one group of edges (same relationship type).
type edgeGroupData struct {
	RelType  string
	RelLabel string
	Edges    []edgeData
}

// edgeData holds one edge link for the template.
type edgeData struct {
	TargetID     string
	Relationship string
	Label        string
	HasSince     bool
	Since        string
}

// stepData holds one implementation step for the template.
type stepData struct {
	CompletedStr string
	StepID       string
	Agent        string
	DependsOnStr string
	Icon         string
	Description  string
}

// newNodeTemplateData converts a models.Node into template-ready data.
func newNodeTemplateData(n *models.Node) *nodeTemplateData {
	d := &nodeTemplateData{
		ID:               n.ID,
		Title:            n.Title,
		Type:             n.Type,
		Status:           string(n.Status),
		Priority:         string(n.Priority),
		CreatedAt:        fmtTime(n.CreatedAt),
		UpdatedAt:        fmtTime(n.UpdatedAt),
		AgentAssigned:    n.AgentAssigned,
		TrackID:          n.TrackID,
		SpikeSubtype:     n.SpikeSubtype,
		ClaimedAt:        n.ClaimedAt,
		ClaimedBySession: n.ClaimedBySession,

		StatusLabel:   titleCase(strings.ReplaceAll(string(n.Status), "-", " ")),
		PriorityLabel: titleCase(string(n.Priority)),
	}

	d.EdgeGroups = buildEdgeGroups(n)
	d.HasEdges = len(d.EdgeGroups) > 0

	d.Steps = buildSteps(n.Steps)
	d.HasSteps = len(d.Steps) > 0

	if n.Content != "" {
		d.HasContent = true
		content := n.Content
		// Wrap plain text in <p> so it survives the HTML round-trip.
		// The parser reads element children only, not text nodes.
		if !strings.HasPrefix(strings.TrimSpace(content), "<") {
			content = "<p>" + content + "</p>"
		}
		d.TrustedContent = template.HTML(content) // #nosec: authored HTML
	}

	return d
}

// buildEdgeGroups converts a Node's edges map into template-ready groups.
func buildEdgeGroups(n *models.Node) []edgeGroupData {
	if len(n.Edges) == 0 {
		return nil
	}
	groups := make([]edgeGroupData, 0, len(n.Edges))
	for relType, edges := range n.Edges {
		if len(edges) == 0 {
			continue
		}
		g := edgeGroupData{
			RelType:  relType,
			RelLabel: titleCase(strings.ReplaceAll(relType, "_", " ")),
			Edges:    make([]edgeData, 0, len(edges)),
		}
		for _, e := range edges {
			label := e.Title
			if label == "" {
				label = e.TargetID
			}
			ed := edgeData{
				TargetID:     e.TargetID,
				Relationship: string(e.Relationship),
				Label:        label,
			}
			if !e.Since.IsZero() {
				ed.HasSince = true
				ed.Since = fmtTime(e.Since)
			}
			g.Edges = append(g.Edges, ed)
		}
		groups = append(groups, g)
	}
	return groups
}

// buildSteps converts a slice of model Steps into template-ready data.
func buildSteps(steps []models.Step) []stepData {
	if len(steps) == 0 {
		return nil
	}
	result := make([]stepData, 0, len(steps))
	for _, s := range steps {
		icon := "\u23F3" // hourglass
		completed := "false"
		if s.Completed {
			icon = "\u2705" // checkmark
			completed = "true"
		}
		sd := stepData{
			CompletedStr: completed,
			StepID:       s.StepID,
			Agent:        s.Agent,
			Icon:         icon,
			Description:  s.Description,
		}
		if len(s.DependsOn) > 0 {
			sd.DependsOnStr = strings.Join(s.DependsOn, ",")
		}
		result = append(result, sd)
	}
	return result
}

// fmtTime formats a time.Time in Python-compatible ISO-8601.
func fmtTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02T15:04:05.999999")
}

// titleCase capitalises the first letter of each word.
func titleCase(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
