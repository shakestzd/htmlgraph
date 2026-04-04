package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/shakestzd/htmlgraph/internal/workitem"
	"github.com/spf13/cobra"
)

//go:embed templates/plan-template.html
var planTemplateFS embed.FS

// planCmdWithExtras builds the standard workitem commands for plans,
// then adds CRISPI-specific subcommands: generate, open, wait, read-feedback.
func planCmdWithExtras() *cobra.Command {
	cmd := workitemCmd("plan", "plans")
	cmd.AddCommand(planGenerateCmd())
	cmd.AddCommand(planOpenCmd())
	cmd.AddCommand(planWaitCmd())
	cmd.AddCommand(planReadFeedbackCmd())
	cmd.AddCommand(planAddQuestionCmd())
	cmd.AddCommand(planSetSectionCmd())
	cmd.AddCommand(planSetSliceCmd())
	return cmd
}

// planGenerateCmd scaffolds a plan HTML file from a feature or track ID.
func planGenerateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "generate <feature-or-track-id>",
		Short: "Scaffold a plan HTML file from a feature or track",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runPlanGenerate(args[0])
		},
	}
}

func runPlanGenerate(sourceID string) error {
	htmlgraphDir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}

	resolved, err := resolveID(htmlgraphDir, sourceID)
	if err != nil {
		return fmt.Errorf("resolve %s: %w", sourceID, err)
	}
	nodePath := resolveNodePath(htmlgraphDir, resolved)
	if nodePath == "" {
		return fmt.Errorf("work item %q not found", resolved)
	}

	info, err := parseNodeForPlan(nodePath)
	if err != nil {
		return fmt.Errorf("parse work item: %w", err)
	}

	planID := workitem.GenerateID("plan", info.title)
	plansDir := filepath.Join(htmlgraphDir, "plans")
	if err := os.MkdirAll(plansDir, 0o755); err != nil {
		return fmt.Errorf("create plans dir: %w", err)
	}
	outPath := filepath.Join(plansDir, planID+".html")

	tmplData, err := planTemplateFS.ReadFile("templates/plan-template.html")
	if err != nil {
		return fmt.Errorf("read plan template: %w", err)
	}

	graphNodes, sliceCards, sectionsJSON, totalSections := buildPlanSections(nodePath, htmlgraphDir)

	// Generate design discussion from the track description and feature summary.
	designContent := buildDesignContent(info, nodePath, htmlgraphDir)
	outlineContent := buildOutlineContent(nodePath, htmlgraphDir)

	content := applyPlanTemplateVars(string(tmplData), planTemplateVars{
		PlanID:         planID,
		FeatureID:      resolved,
		Title:          info.title,
		Description:    info.description,
		Date:           time.Now().UTC().Format("2006-01-02"),
		GraphNodes:     graphNodes,
		SliceCards:     sliceCards,
		SectionsJSON:   sectionsJSON,
		TotalSections:  totalSections,
		DesignContent:  designContent,
		OutlineContent: outlineContent,
	})

	if err := os.WriteFile(outPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write plan: %w", err)
	}

	fmt.Println(outPath)
	return nil
}

// planOpenCmd opens a plan in the browser.
func planOpenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "open <plan-id>",
		Short: "Open a plan in the browser",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runPlanOpen(args[0])
		},
	}
}

func runPlanOpen(planID string) error {
	htmlgraphDir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}

	planPath := filepath.Join(htmlgraphDir, "plans", planID+".html")
	if _, err := os.Stat(planPath); err != nil {
		return fmt.Errorf("plan %q not found at %s", planID, planPath)
	}

	if !isServerRunning("http://localhost:8080") {
		// Auto-start server so plan feedback API works.
		cmd := exec.Command(os.Args[0], "serve", "-p", "8080")
		cmd.Stdout = nil
		cmd.Stderr = nil
		_ = cmd.Start()
		time.Sleep(500 * time.Millisecond)
	}

	url := "http://localhost:8080/plans/" + planID + ".html"
	return openBrowser(url)
}

// planWaitCmd blocks until a plan is finalized.
func planWaitCmd() *cobra.Command {
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:   "wait <plan-id>",
		Short: "Block until a plan is finalized",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runPlanWait(args[0], timeout)
		},
	}
	cmd.Flags().DurationVar(&timeout, "timeout", time.Hour, "Maximum wait time (e.g. 30m, 1h)")
	return cmd
}

func runPlanWait(planID string, timeout time.Duration) error {
	htmlgraphDir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	fmt.Printf("Waiting for plan %s to be finalized", planID)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println()
			return fmt.Errorf("timeout: plan %s was not finalized within %s", planID, timeout)
		case <-ticker.C:
			finalized, err := checkPlanFinalized(htmlgraphDir, planID)
			if err != nil {
				fmt.Print(".")
				continue
			}
			if finalized {
				fmt.Println("\nPlan finalized.")
				return nil
			}
			fmt.Print(".")
		}
	}
}

// checkPlanFinalized returns true when the plan's status is "finalized".
// Prefers the live API; falls back to reading the HTML file directly.
func checkPlanFinalized(htmlgraphDir, planID string) (bool, error) {
	if isServerRunning("http://localhost:8080") {
		status, err := fetchPlanStatusFromAPI(planID)
		if err == nil {
			return status == "finalized", nil
		}
	}

	planPath := filepath.Join(htmlgraphDir, "plans", planID+".html")
	status, err := parsePlanHTMLStatus(planPath)
	if err != nil {
		return false, err
	}
	return status == "finalized", nil
}

// fetchPlanStatusFromAPI calls GET /api/plans/{id}/status and returns the status.
func fetchPlanStatusFromAPI(planID string) (string, error) {
	url := "http://localhost:8080/api/plans/" + planID + "/status"
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url) //nolint:gosec,noctx
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned %d", resp.StatusCode)
	}
	var result struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.Status, nil
}

// ---- browser / server helpers -----------------------------------------------

// isServerRunning returns true when a GET to baseURL succeeds within 500ms.
func isServerRunning(baseURL string) bool {
	client := &http.Client{Timeout: 500 * time.Millisecond}
	resp, err := client.Get(baseURL) //nolint:gosec,noctx
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode < 500
}

// openBrowser opens the given URL or file path in the default OS browser.
func openBrowser(target string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", target)
	case "linux":
		cmd = exec.Command("xdg-open", target)
	default:
		fmt.Println(target)
		return nil
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open browser: %w", err)
	}
	return nil
}
