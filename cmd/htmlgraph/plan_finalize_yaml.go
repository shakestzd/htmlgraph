package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	dbpkg "github.com/shakestzd/htmlgraph/internal/db"
	"github.com/shakestzd/htmlgraph/internal/models"
	"github.com/shakestzd/htmlgraph/internal/planyaml"
	"github.com/shakestzd/htmlgraph/internal/workitem"
	"github.com/spf13/cobra"
)

// planAmendment represents a structured amendment directive from the chat review system.
type planAmendment struct {
	SliceNum  int    `json:"slice_num"`
	Field     string `json:"field"`
	Operation string `json:"operation"`
	Content   string `json:"content"`
}

// planFinalizeYAMLCmd creates track + features from approved slices in a YAML plan.
func planFinalizeYAMLCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "finalize-yaml <plan-id>",
		Short: "Create track and features from approved YAML plan slices (dashboard flow)",
		Long: `Read a YAML plan + SQLite plan_feedback approvals, create a track and
features for approved slices, wire dependency edges. Updates YAML status to
finalized.

This is the dashboard-review workflow: only slices with explicit approve
actions in plan_feedback get promoted, and the track is created from scratch
when one does not yet exist.

For the simpler hierarchy-only flow that requires an existing track and
promotes every slice unconditionally, use 'plan finalize' instead.

Example:
  htmlgraph plan finalize-yaml plan-a1b2c3d4`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runFinalizeYAML(args[0])
		},
	}
}

func runFinalizeYAML(planID string) error {
	htmlgraphDir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}
	return finalizeYAML(htmlgraphDir, planID)
}

// finalizeYAML is the testable inner implementation of runFinalizeYAML.
// It takes an explicit htmlgraphDir rather than resolving it from the environment.
func finalizeYAML(htmlgraphDir, planID string) error {
	planPath := filepath.Join(htmlgraphDir, "plans", planID+".yaml")
	plan, err := planyaml.Load(planPath)
	if err != nil {
		return fmt.Errorf("load plan: %w", err)
	}

	// Read approvals from SQLite.
	dbPath := filepath.Join(htmlgraphDir, "htmlgraph.db")
	db, err := dbpkg.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	approvals := map[string]bool{}
	rows, err := db.Query(
		"SELECT section, value FROM plan_feedback WHERE plan_id = ? AND action = 'approve'",
		planID,
	)
	if err != nil {
		return fmt.Errorf("query approvals: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var section, value string
		rows.Scan(&section, &value)
		approvals[section] = strings.EqualFold(value, "true")
	}

	// Read question answers from SQLite.
	answers := map[string]string{}
	answerRows, err := db.Query(
		"SELECT question_id, value FROM plan_feedback WHERE plan_id = ? AND action = 'answer'",
		planID,
	)
	if err != nil {
		return fmt.Errorf("query answers: %w", err)
	}
	defer answerRows.Close()
	for answerRows.Next() {
		var qID, value string
		answerRows.Scan(&qID, &value)
		answers[qID] = value
	}

	// Read accepted amendments from SQLite.
	amendRows, err := db.Query(
		"SELECT value FROM plan_feedback WHERE plan_id = ? AND section = 'amendment' AND action = 'accepted'",
		planID,
	)
	if err != nil {
		return fmt.Errorf("query amendments: %w", err)
	}
	defer amendRows.Close()

	var amendments []planAmendment
	for amendRows.Next() {
		var raw string
		amendRows.Scan(&raw)
		var a planAmendment
		if json.Unmarshal([]byte(raw), &a) == nil {
			amendments = append(amendments, a)
		}
	}

	// Apply accepted amendments to plan slices in memory.
	applyAmendments(plan, amendments)
	if len(amendments) > 0 {
		fmt.Printf("Applied %d amendment(s) to plan\n", len(amendments))
	}

	// Open project for work item creation.
	p, err := workitem.Open(htmlgraphDir, agentForClaim())
	if err != nil {
		return fmt.Errorf("open project: %w", err)
	}
	defer p.Close()

	// Idempotent re-finalize: plan already finalized — look up existing track+features
	// via planned_in edges and emit the full dispatch summary without creating new nodes.
	if plan.Meta.Status == "finalized" {
		trackID := plan.Meta.TrackID
		if trackID == "" {
			trackID = findTrackForPlan(p, planID)
		}
		var existingTrack *models.Node
		if trackID != "" {
			existingTrack, _ = p.Tracks.Get(trackID)
		}
		// Use planned_in edges (feature → plan) to find existing features.
		// This works even when part_of/contains edges were not written on first run.
		featIDs := findFeaturesForPlan(db, planID)
		var rejectedOnRefinalize []string
		for _, s := range plan.Slices {
			if !s.Approved {
				rejectedOnRefinalize = append(rejectedOnRefinalize, s.Title)
			}
		}
		printFinalizeYAMLSummary(plan, existingTrack, answers, featIDs, rejectedOnRefinalize)
		return nil
	}

	// Reuse existing track when meta.track_id references a valid track;
	// otherwise create a new one from the plan title.
	var track *models.Node
	if plan.Meta.TrackID != "" {
		existing, getErr := p.Tracks.Get(plan.Meta.TrackID)
		if getErr == nil && existing != nil {
			track = existing
			fmt.Printf("Using existing track: %s  %s\n", track.ID, track.Title)
		}
	}
	if track == nil {
		track, err = p.Tracks.Create(plan.Meta.Title)
		if err != nil {
			return fmt.Errorf("create track: %w", err)
		}
		fmt.Printf("Created track: %s  %s\n", track.ID, track.Title)
	}

	// Create features for approved slices, embedding design decisions.
	type createdFeat struct {
		id    string
		title string
	}
	numToFeat := map[int]createdFeat{}
	var rejectedTitles []string
	for _, s := range plan.Slices {
		approved := approvals[fmt.Sprintf("slice-%d", s.Num)]
		if !approved {
			rejectedTitles = append(rejectedTitles, s.Title)
			continue
		}
		content := buildFeatureContent(s.What, plan.Questions, answers)
		feat, err := p.Features.Create(s.Title,
			workitem.FeatWithTrack(track.ID),
			workitem.FeatWithContent(content),
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Error creating feature for slice %d: %v\n", s.Num, err)
			continue
		}
		numToFeat[s.Num] = createdFeat{id: feat.ID, title: feat.Title}

		// Link feature back to source plan (planned_in).
		p.Features.AddEdge(feat.ID, models.Edge{
			TargetID:     planID,
			Relationship: models.RelPlannedIn,
			Title:        planID,
			Since:        time.Now().UTC(),
		})

		// Wire part_of (feature→track) and contains (track→feature) edges.
		wireTrackEdges(p, feat.ID, track.ID, feat.Title) //nolint:errcheck
	}

	// Link plan to track: plan implemented_in track.
	p.Plans.AddEdge(planID, models.Edge{
		TargetID:     track.ID,
		Relationship: models.RelImplementedIn,
		Title:        track.ID,
		Since:        time.Now().UTC(),
	})

	// Wire blocked_by edges from slice deps.
	for _, s := range plan.Slices {
		cf, ok := numToFeat[s.Num]
		if !ok {
			continue
		}
		for _, depNum := range s.Deps {
			depCF, ok := numToFeat[depNum]
			if !ok {
				continue
			}
			p.Features.AddEdge(cf.id, models.Edge{
				TargetID:     depCF.id,
				Relationship: "blocked_by",
			})
		}
	}

	// Update YAML status.
	plan.Meta.Status = "finalized"
	plan.Meta.TrackID = track.ID
	for i := range plan.Slices {
		plan.Slices[i].Approved = approvals[fmt.Sprintf("slice-%d", plan.Slices[i].Num)]
	}
	if err := planyaml.Save(planPath, plan); err != nil {
		return fmt.Errorf("save plan: %w", err)
	}

	if err := commitPlanChange(planPath, fmt.Sprintf("plan(%s): finalize — %d slices approved", planID, len(numToFeat))); err != nil {
		return fmt.Errorf("autocommit finalize-yaml: %w", err)
	}

	// Build feat IDs list in slice order for summary.
	var featIDs []string
	for _, s := range plan.Slices {
		if cf, ok := numToFeat[s.Num]; ok {
			featIDs = append(featIDs, cf.id)
		}
	}
	printFinalizeYAMLSummary(plan, track, answers, featIDs, rejectedTitles)
	return nil
}

// buildFeatureContent constructs a feature description from a slice's "what" field
// plus an "Accepted Design Decisions" section derived from answered questions.
func buildFeatureContent(what string, questions []planyaml.PlanQuestion, answers map[string]string) string {
	if len(questions) == 0 {
		return what
	}

	var sb strings.Builder
	sb.WriteString(what)

	hasDecisions := false
	for _, q := range questions {
		optionKey := answers[q.ID]
		if optionKey == "" && q.Recommended == "" {
			continue // nothing to embed
		}
		hasDecisions = true
		break
	}
	if !hasDecisions {
		return what
	}

	sb.WriteString("\n\n## Accepted Design Decisions\n")
	for _, q := range questions {
		optionKey := answers[q.ID]
		label := ""
		isUnanswered := false

		if optionKey == "" {
			// Fall back to recommended option.
			optionKey = q.Recommended
			isUnanswered = true
		}

		for _, opt := range q.Options {
			if opt.Key == optionKey {
				label = opt.Label
				break
			}
		}
		if label == "" {
			label = optionKey
		}

		suffix := ""
		if isUnanswered {
			suffix = " (unanswered, using recommended)"
		}
		fmt.Fprintf(&sb, "- **%s** → %s (Q: %s)%s\n", q.Text, label, q.ID, suffix)
	}

	return sb.String()
}

// printFinalizeYAMLSummary prints the structured dispatch summary to stdout.
// featIDs may reference existing features (re-finalize) or newly created ones.
// rejectedTitles lists slice titles that were not approved.
func printFinalizeYAMLSummary(
	plan *planyaml.PlanYAML,
	track *models.Node,
	answers map[string]string,
	featIDs []string,
	rejectedTitles []string,
) {
	trackID := ""
	trackTitle := ""
	if track != nil {
		trackID = track.ID
		trackTitle = track.Title
	} else if plan.Meta.TrackID != "" {
		trackID = plan.Meta.TrackID
	}

	totalSlices := len(plan.Slices)
	approvedCount := len(featIDs)
	rejectedCount := len(rejectedTitles)
	explicitAnswers := 0
	recommendedFallbacks := 0
	for _, q := range plan.Questions {
		if answers[q.ID] != "" {
			explicitAnswers++
		} else if q.Recommended != "" {
			recommendedFallbacks++
		}
	}

	fmt.Printf("\nPlan %s dispatched.\n", plan.Meta.ID)
	fmt.Println()
	if trackTitle != "" {
		fmt.Printf("Track:        %s (%s)\n", trackID, trackTitle)
	} else {
		fmt.Printf("Track:        %s\n", trackID)
	}
	fmt.Printf("Approved:     %d of %d slices\n", approvedCount, totalSlices)
	if rejectedCount > 0 {
		fmt.Printf("Rejected:     %d slices (excluded from dispatch)\n", rejectedCount)
	}

	if len(featIDs) > 0 {
		fmt.Println("\nFeatures created:")
		for i, fid := range featIDs {
			sliceTitle := ""
			if i < len(plan.Slices) {
				// Find matching approved slice title.
				sliceTitle = plan.Slices[i].Title
			}
			fmt.Printf("  %-20s  %s\n", fid, sliceTitle)
		}
	}

	if len(rejectedTitles) > 0 {
		fmt.Println("\nRejected (excluded):")
		for _, t := range rejectedTitles {
			fmt.Printf("  %s  (not approved — excluded)\n", t)
		}
	}

	// Design decisions section: resolved questions with explicit/fallback breakdown.
	type resolvedDecision struct {
		qID   string
		text  string
		label string
		isRec bool // true if resolved via recommended fallback
	}
	var resolvedDecisions []resolvedDecision
	for _, q := range plan.Questions {
		optKey := answers[q.ID]
		isRec := false
		if optKey == "" {
			optKey = q.Recommended
			isRec = true
		}
		if optKey == "" {
			continue
		}
		label := optKey
		for _, opt := range q.Options {
			if opt.Key == optKey {
				label = opt.Label
				break
			}
		}
		resolvedDecisions = append(resolvedDecisions, resolvedDecision{
			qID:   q.ID,
			text:  q.Text,
			label: label,
			isRec: isRec,
		})
	}
	if len(resolvedDecisions) > 0 {
		fmt.Printf("\nDesign decisions (%d, %d explicit / %d recommended defaults):\n",
			len(resolvedDecisions), explicitAnswers, recommendedFallbacks)
		for _, d := range resolvedDecisions {
			fmt.Printf("  %-30s  → %s\n", d.qID, d.label)
		}
	}

	fmt.Println("\nNext:")
	fmt.Printf("  /htmlgraph:execute %s   (in Claude — dispatches tasks)\n", plan.Meta.ID)
	fmt.Printf("  OR: htmlgraph yolo --track %s   (autonomous mode)\n", trackID)
}

// applyAmendments applies accepted amendment directives to plan slices in memory.
// Amendments are applied in order; later amendments for the same field win.
func applyAmendments(plan *planyaml.PlanYAML, amendments []planAmendment) {
	for _, a := range amendments {
		idx := -1
		for i, s := range plan.Slices {
			if s.Num == a.SliceNum {
				idx = i
				break
			}
		}
		if idx < 0 {
			fmt.Fprintf(os.Stderr, "  Amendment skipped: slice %d not found\n", a.SliceNum)
			continue
		}
		s := &plan.Slices[idx]

		switch a.Operation {
		case "add":
			switch a.Field {
			case "done_when":
				s.DoneWhen = append(s.DoneWhen, a.Content)
			case "files":
				s.Files = append(s.Files, a.Content)
			}
		case "remove":
			switch a.Field {
			case "done_when":
				s.DoneWhen = removeStr(s.DoneWhen, a.Content)
			case "files":
				s.Files = removeStr(s.Files, a.Content)
			}
		case "set":
			switch a.Field {
			case "title":
				s.Title = a.Content
			case "what":
				s.What = a.Content
			case "why":
				s.Why = a.Content
			case "effort":
				s.Effort = a.Content
			case "risk":
				s.Risk = a.Content
			}
		}
		fmt.Printf("  Applied amendment: slice-%d %s %s\n", a.SliceNum, a.Operation, a.Field)
	}
}

// removeStr returns a new slice with all occurrences of target removed.
func removeStr(slice []string, target string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != target {
			result = append(result, s)
		}
	}
	return result
}

// findFeaturesForPlan returns feature IDs that have a planned_in edge pointing
// to planID, queried directly from the graph_edges SQLite table. This is the
// correct lookup for the re-finalize path because the yaml finalize first-run
// only writes planned_in edges, not part_of/contains edges.
func findFeaturesForPlan(db *sql.DB, planID string) []string {
	if db == nil {
		return nil
	}
	rows, err := db.Query(
		"SELECT from_node_id FROM graph_edges WHERE to_node_id = ? AND relationship_type = 'planned_in' AND from_node_id LIKE 'feat-%'",
		planID,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	return ids
}
