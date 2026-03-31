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
	"time"

	"github.com/shakestzd/htmlgraph/internal/models"
)

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
// Returns the absolute path of the written file.
func WriteNodeHTML(dir string, node *models.Node) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create dir %s: %w", dir, err)
	}

	path := filepath.Join(dir, node.ID+".html")
	html, err := renderNodeHTML(node)
	if err != nil {
		return "", fmt.Errorf("render %s: %w", node.ID, err)
	}

	if err := os.WriteFile(path, []byte(html), 0o644); err != nil {
		return "", fmt.Errorf("write %s: %w", path, err)
	}
	return path, nil
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
		// Content may contain trusted HTML (e.g. <p>, <ul> from AddNote).
		d.TrustedContent = template.HTML(n.Content) // #nosec: authored HTML
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
