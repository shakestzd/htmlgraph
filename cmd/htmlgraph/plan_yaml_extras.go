package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	dbpkg "github.com/shakestzd/htmlgraph/internal/db"
	"github.com/shakestzd/htmlgraph/internal/planyaml"
	"github.com/spf13/cobra"
)

func planAddQuestionYAMLCmd() *cobra.Command {
	var description, recommended, options string
	cmd := &cobra.Command{
		Use:   "add-question-yaml <plan-id> <question-text>",
		Short: "Add a question with description and recommended option to a YAML plan",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			return runAddQuestionYAML(args[0], args[1], description, recommended, options)
		},
	}
	cmd.Flags().StringVar(&description, "description", "", "context paragraph (required)")
	cmd.Flags().StringVar(&recommended, "recommended", "", "recommended option key")
	cmd.Flags().StringVar(&options, "options", "", "comma-separated key:label pairs (min 2)")
	return cmd
}

func runAddQuestionYAML(planID, text, description, recommended, optionsStr string) error {
	if description == "" {
		return fmt.Errorf("--description is required")
	}
	opts := parseQuestionOptions(optionsStr)
	if len(opts) < 2 {
		return fmt.Errorf("--options must have at least 2 entries (got %d)", len(opts))
	}
	if recommended != "" {
		found := false
		for _, o := range opts {
			if o.Key == recommended {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("--recommended %q not found in options", recommended)
		}
	}
	htmlgraphDir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}
	planPath := filepath.Join(htmlgraphDir, "plans", planID+".yaml")
	plan, err := planyaml.Load(planPath)
	if err != nil {
		return fmt.Errorf("load plan: %w", err)
	}
	qid := "q-" + kebabCase(text, 40)
	plan.Questions = append(plan.Questions, planyaml.PlanQuestion{
		ID: qid, Text: text, Description: description,
		Recommended: recommended, Options: opts, Answer: nil,
	})
	if err := planyaml.Save(planPath, plan); err != nil {
		return fmt.Errorf("save plan: %w", err)
	}
	fmt.Printf("Added question: %s (%d options)\n", qid, len(opts))
	return nil
}

func parseQuestionOptions(s string) []planyaml.QuestionOption {
	if s == "" {
		return nil
	}
	var opts []planyaml.QuestionOption
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		idx := strings.Index(part, ":")
		if idx < 0 {
			continue
		}
		opts = append(opts, planyaml.QuestionOption{
			Key: strings.TrimSpace(part[:idx]), Label: strings.TrimSpace(part[idx+1:]),
		})
	}
	return opts
}

func kebabCase(s string, maxLen int) string {
	s = strings.ToLower(s)
	s = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		return '-'
	}, s)
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	s = strings.Trim(s, "-")
	if len(s) > maxLen {
		s = s[:maxLen]
		s = strings.TrimRight(s, "-")
	}
	return s
}

func planSetCritiqueYAMLCmd() *cobra.Command {
	var data string
	cmd := &cobra.Command{
		Use:   "set-critique-yaml <plan-id>",
		Short: "Write AI critique data to a YAML plan (from --data or stdin)",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runSetCritiqueYAML(args[0], data)
		},
	}
	cmd.Flags().StringVar(&data, "data", "", "critique JSON (reads stdin if empty)")
	return cmd
}

func runSetCritiqueYAML(planID, dataStr string) error {
	htmlgraphDir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}
	planPath := filepath.Join(htmlgraphDir, "plans", planID+".yaml")
	plan, err := planyaml.Load(planPath)
	if err != nil {
		return fmt.Errorf("load plan: %w", err)
	}
	var jsonBytes []byte
	if dataStr != "" {
		jsonBytes = []byte(dataStr)
	} else {
		jsonBytes, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}
	}
	var critique planyaml.PlanCritique
	if err := json.Unmarshal(jsonBytes, &critique); err != nil {
		return fmt.Errorf("parse critique JSON: %w", err)
	}
	plan.Critique = &critique
	if err := planyaml.Save(planPath, plan); err != nil {
		return fmt.Errorf("save plan: %w", err)
	}
	fmt.Printf("Critique set for %s: %d assumptions, %d risks\n",
		planID, len(critique.Assumptions), len(critique.Risks))
	return nil
}

func planValidateYAMLCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate-yaml <plan-id>",
		Short: "Validate a YAML plan's schema",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runValidateYAML(args[0])
		},
	}
}

func runValidateYAML(planID string) error {
	htmlgraphDir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}
	planPath := filepath.Join(htmlgraphDir, "plans", planID+".yaml")
	plan, err := planyaml.Load(planPath)
	if err != nil {
		return fmt.Errorf("load plan: %w", err)
	}
	errors := planyaml.Validate(plan)
	if len(errors) > 0 {
		for _, e := range errors {
			fmt.Fprintf(os.Stderr, "  - %s\n", e)
		}
		return fmt.Errorf("%d validation errors", len(errors))
	}
	fmt.Printf("Plan valid: %d slices, %d questions\n", len(plan.Slices), len(plan.Questions))
	return nil
}

// planReviewCmd launches a marimo notebook for interactive plan review.
func planReviewCmd() *cobra.Command {
	var port int
	var wait bool

	cmd := &cobra.Command{
		Use:   "review <plan-id>",
		Short: "Open a YAML plan in the marimo review notebook",
		Long: `Launch marimo to interactively review a YAML plan.

The notebook reads plan content from the YAML file and persists
human approvals to the SQLite plan_feedback table on every click.

Example:
  htmlgraph plan review plan-a1b2c3d4
  htmlgraph plan review plan-a1b2c3d4 --port 3001
  htmlgraph plan review plan-a1b2c3d4 --wait`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runPlanReview(args[0], port, wait)
		},
	}
	cmd.Flags().IntVar(&port, "port", 3001, "marimo server port")
	cmd.Flags().BoolVar(&wait, "wait", false, "block until plan is finalized")
	return cmd
}

func runPlanReview(planID string, port int, wait bool) error {
	htmlgraphDir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}

	// Verify plan YAML exists.
	planPath := filepath.Join(htmlgraphDir, "plans", planID+".yaml")
	if _, err := os.Stat(planPath); err != nil {
		return fmt.Errorf("plan YAML not found: %s", planPath)
	}

	// Find the notebook template. Check common locations.
	notebookPath := findNotebookTemplate(htmlgraphDir)
	if notebookPath == "" {
		return fmt.Errorf("marimo notebook template not found. Expected at prototypes/plan_notebook.py or plugin/templates/plan_notebook.py")
	}

	// Check marimo is installed.
	marimoPath, err := exec.LookPath("marimo")
	if err != nil {
		return fmt.Errorf("marimo not found in PATH. Install with: uv tool install marimo --with anywidget --with traitlets --with pyyaml")
	}

	fmt.Printf("Plan:     %s\n", planPath)
	fmt.Printf("Notebook: %s\n", notebookPath)
	fmt.Printf("URL:      http://localhost:%d\n", port)
	fmt.Println()

	// Launch marimo. Run from the notebook's directory so imports resolve.
	notebookDir := filepath.Dir(notebookPath)
	args := []string{
		"edit", filepath.Base(notebookPath),
		"--port", fmt.Sprintf("%d", port),
		"--headless", "--no-token",
	}

	// Set PLAN_YAML_PATH env var so the notebook knows which plan to load.
	env := append(os.Environ(), "PLAN_YAML_PATH="+planPath)

	marimoCmd := exec.Command(marimoPath, args...)
	marimoCmd.Dir = notebookDir
	marimoCmd.Env = env
	marimoCmd.Stdout = os.Stdout
	marimoCmd.Stderr = os.Stderr

	if !wait {
		// Start in background, print URL, return.
		if err := marimoCmd.Start(); err != nil {
			return fmt.Errorf("start marimo: %w", err)
		}
		fmt.Printf("Marimo running (PID %d). Open http://localhost:%d to review.\n", marimoCmd.Process.Pid, port)
		fmt.Println("Run 'htmlgraph plan review " + planID + " --wait' to block until finalized.")
		return nil
	}

	// Foreground mode: run marimo and block.
	fmt.Println("Marimo running. Waiting for plan finalization...")
	fmt.Println("Open http://localhost:" + fmt.Sprintf("%d", port) + " to review.")
	return marimoCmd.Run()
}

func findNotebookTemplate(htmlgraphDir string) string {
	// 1. Environment variable override
	if p := os.Getenv("PLAN_NOTEBOOK_PATH"); p != "" {
		if _, err := os.Stat(p); err == nil {
			abs, _ := filepath.Abs(p)
			return abs
		}
	}

	// 2. Relative to project
	candidates := []string{
		filepath.Join(filepath.Dir(htmlgraphDir), "prototypes", "plan_notebook.py"),
		filepath.Join(htmlgraphDir, "..", "prototypes", "plan_notebook.py"),
		filepath.Join(htmlgraphDir, "..", "plugin", "templates", "plan_notebook.py"),
	}

	// 3. User install location
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".local", "share", "htmlgraph", "plan_notebook.py"))
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			abs, _ := filepath.Abs(p)
			return abs
		}
	}
	return ""
}

// planSetDesignYAMLCmd sets the structured design subsections on a YAML plan.
func planSetDesignYAMLCmd() *cobra.Command {
	var problem, goals, constraints string

	cmd := &cobra.Command{
		Use:   "set-design-yaml <plan-id>",
		Short: "Set problem, goals, and constraints on a YAML plan",
		Long: `Set the structured design subsections on a YAML plan.

Example:
  htmlgraph plan set-design-yaml plan-a1b2c3d4 \
    --problem "The current system has X limitation..." \
    --goals "Goal 1,Goal 2,Goal 3" \
    --constraints "Must not break X,Must support Y"`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runSetDesignYAML(args[0], problem, goals, constraints)
		},
	}
	cmd.Flags().StringVar(&problem, "problem", "", "problem statement (what's wrong and why)")
	cmd.Flags().StringVar(&goals, "goals", "", "comma-separated measurable goals")
	cmd.Flags().StringVar(&constraints, "constraints", "", "comma-separated constraints")
	return cmd
}

func runSetDesignYAML(planID, problem, goals, constraints string) error {
	htmlgraphDir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}
	planPath := filepath.Join(htmlgraphDir, "plans", planID+".yaml")
	plan, err := planyaml.Load(planPath)
	if err != nil {
		return fmt.Errorf("load plan: %w", err)
	}
	if problem != "" {
		plan.Design.Problem = problem
	}
	if goals != "" {
		plan.Design.Goals = splitTrimmed(goals)
	}
	if constraints != "" {
		plan.Design.Constraints = splitTrimmed(constraints)
	}
	if err := planyaml.Save(planPath, plan); err != nil {
		return fmt.Errorf("save plan: %w", err)
	}
	fmt.Printf("Design updated for %s: problem=%v goals=%d constraints=%d\n",
		planID, problem != "", len(plan.Design.Goals), len(plan.Design.Constraints))
	return nil
}

func splitTrimmed(s string) []string {
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// planReadFeedbackYAMLCmd queries plan_feedback for a YAML plan and outputs JSON.
func planReadFeedbackYAMLCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "read-feedback-yaml <plan-id>",
		Short: "Read human feedback for a YAML plan from SQLite",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runReadFeedbackYAML(args[0])
		},
	}
}

func runReadFeedbackYAML(planID string) error {
	htmlgraphDir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}
	// Read YAML status.
	planPath := filepath.Join(htmlgraphDir, "plans", planID+".yaml")
	plan, err := planyaml.Load(planPath)
	if err != nil {
		return fmt.Errorf("load plan: %w", err)
	}
	// Query SQLite.
	dbPath := filepath.Join(htmlgraphDir, "htmlgraph.db")
	db, err := dbpkg.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()
	rows, err := db.Query("SELECT section, action, value, question_id FROM plan_feedback WHERE plan_id = ?", planID)
	if err != nil {
		return fmt.Errorf("query feedback: %w", err)
	}
	defer rows.Close()

	type feedbackResult struct {
		PlanID          string            `json:"plan_id"`
		Status          string            `json:"status"`
		DesignApproved  bool              `json:"design_approved"`
		DesignComment   string            `json:"design_comment,omitempty"`
		SliceApprovals  map[string]bool   `json:"slice_approvals"`
		QuestionAnswers map[string]string `json:"question_answers"`
		Comments        map[string]string `json:"comments"`
	}
	result := feedbackResult{
		PlanID:          planID,
		Status:          plan.Meta.Status,
		SliceApprovals:  make(map[string]bool),
		QuestionAnswers: make(map[string]string),
		Comments:        make(map[string]string),
	}
	for rows.Next() {
		var section, action, value, qid string
		if err := rows.Scan(&section, &action, &value, &qid); err != nil {
			return fmt.Errorf("scan feedback row: %w", err)
		}
		switch action {
		case "approve":
			if section == "design" {
				result.DesignApproved = strings.EqualFold(value, "true")
			} else {
				result.SliceApprovals[section] = strings.EqualFold(value, "true")
			}
		case "comment":
			if section == "design" {
				result.DesignComment = value
			} else {
				result.Comments[section] = value
			}
		case "answer":
			result.QuestionAnswers[qid] = value
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate feedback rows: %w", err)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}
