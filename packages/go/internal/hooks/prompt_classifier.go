package hooks

import (
	"fmt"
	"strings"
)

// PromptIntent captures the classification of a user prompt.
type PromptIntent struct {
	// Primary intent flags (from classify_prompt)
	IsImplementation bool
	IsInvestigation  bool
	IsBugReport      bool
	IsContinuation   bool

	// CIGS delegation flags (from classify_cigs_intent)
	InvolvesExploration  bool
	InvolvesCodeChanges  bool
	InvolvesGit          bool

	// Confidence score (0.0–1.0) for the strongest matched category.
	Confidence float64
}

// ---------- keyword lists (mirrors Python prompt_analyzer.py) ----------

// explorationKeywords signal search / read / review activity.
var explorationKeywords = []string{
	"search", "find", "what files", "which files", "where is",
	"locate", "analyze", "examine", "inspect", "review",
	"check", "look at", "show me", "list", "grep",
	"read", "scan", "explore",
}

// codeChangeKeywords signal implementation / modification activity.
var codeChangeKeywords = []string{
	"implement", "fix", "update", "refactor", "change",
	"modify", "edit", "write", "create file", "add code",
	"remove code", "replace", "rewrite", "patch", "add",
}

// gitKeywords signal git operations.
var gitKeywords = []string{
	"commit", "push", "pull", "merge", "branch", "checkout",
	"git add", "git commit", "git push", "git status", "git diff",
	"rebase", "cherry-pick", "stash",
}

// bugKeywords signal bug / error reports.
var bugKeywords = []string{
	"bug", "issue", "error", "problem", "broken",
	"not working", "fails", "crash", "something wrong",
	"doesn't work", "isn't working",
}

// implementationKeywords signal implementation requests.
var implementationKeywords = []string{
	"implement", "create", "build", "develop", "make",
	"add feature", "add function", "add method", "add endpoint",
	"write code", "fix bug", "resolve issue", "patch",
}

// investigationKeywords signal research / exploration intent.
var investigationKeywords = []string{
	"investigate", "research", "explore", "analyze",
	"understand", "find out", "look into",
	"why", "how come", "what causes",
}

// continuationKeywords signal "keep going" type prompts.
var continuationKeywords = []string{
	"continue", "resume", "proceed", "go on", "keep going",
	"next", "where we left off", "from before", "last time",
	"ok", "okay", "yes", "sure", "do it", "go ahead",
}

// ClassifyPrompt analyses a user prompt and returns a PromptIntent
// describing the user's likely intent. Uses fast keyword matching
// (no regex) for hook-level performance.
func ClassifyPrompt(prompt string) PromptIntent {
	lower := strings.ToLower(strings.TrimSpace(prompt))
	intent := PromptIntent{}

	// Short prompts that are pure continuation signals.
	if matchesContinuation(lower) {
		intent.IsContinuation = true
		intent.Confidence = 0.9
		return intent
	}

	// Primary intent classification.
	if countKeywordHits(lower, implementationKeywords) > 0 {
		intent.IsImplementation = true
		intent.Confidence = max64(intent.Confidence, 0.8)
	}
	if countKeywordHits(lower, investigationKeywords) > 0 {
		intent.IsInvestigation = true
		intent.Confidence = max64(intent.Confidence, 0.7)
	}
	if countKeywordHits(lower, bugKeywords) > 0 {
		intent.IsBugReport = true
		intent.Confidence = max64(intent.Confidence, 0.75)
	}

	// CIGS delegation flags.
	if n := countKeywordHits(lower, explorationKeywords); n > 0 {
		intent.InvolvesExploration = true
		intent.Confidence = max64(intent.Confidence, min64(1.0, float64(n)*0.3))
	}
	if n := countKeywordHits(lower, codeChangeKeywords); n > 0 {
		intent.InvolvesCodeChanges = true
		intent.Confidence = max64(intent.Confidence, min64(1.0, float64(n)*0.35))
	}
	if n := countKeywordHits(lower, gitKeywords); n > 0 {
		intent.InvolvesGit = true
		intent.Confidence = max64(intent.Confidence, min64(1.0, float64(n)*0.4))
	}

	return intent
}

// ---------- guidance generators ----------

// GenerateGuidance produces the additionalContext string for CIGS injection.
// It combines work-item attribution (already handled by buildAttributionGuidance)
// with intent-specific orchestrator directives.
//
// Parameters:
//   - intent: classification result from ClassifyPrompt
//   - activeFeatureID: currently active work item (may be "")
//   - activeWorkType: type of the active work item ("feature", "spike", "bug", or "")
//   - attributionBlock: pre-built attribution guidance from buildAttributionGuidance
//
// Returns the combined guidance string (may be empty).
func GenerateGuidance(intent PromptIntent, activeFeatureID, activeWorkType, attributionBlock string) string {
	var parts []string

	directive := intentDirective(intent, activeFeatureID, activeWorkType)
	if directive != "" {
		parts = append(parts, directive)
	}

	cigsBlock := cigsImperatives(intent)
	if cigsBlock != "" {
		parts = append(parts, cigsBlock)
	}

	if attributionBlock != "" {
		parts = append(parts, attributionBlock)
	}

	return strings.Join(parts, "\n\n")
}

// intentDirective returns orchestrator workflow directives based on the prompt
// intent and the currently active work item type. Mirrors the Python
// generate_guidance() function's branching logic.
func intentDirective(intent PromptIntent, activeFeatureID, activeWorkType string) string {
	// Continuation with active work — no extra directive needed.
	if intent.IsContinuation && activeFeatureID != "" {
		return ""
	}

	hasActive := activeFeatureID != ""

	// Implementation during a spike — warn to transition to a feature.
	if intent.IsImplementation && hasActive && activeWorkType == "spike" {
		return fmt.Sprintf(
			"ORCHESTRATOR DIRECTIVE: Implementation requested during spike.\n"+
				"Active work: %s — Type: spike\n\n"+
				"Spikes are for investigation, NOT implementation.\n"+
				"REQUIRED: Complete or pause the spike, then create a feature for implementation.\n"+
				"Delegate to a coder subagent — orchestrators coordinate, subagents implement.",
			activeFeatureID,
		)
	}

	// Implementation with a feature active — remind to delegate.
	if intent.IsImplementation && hasActive && activeWorkType == "feature" {
		return fmt.Sprintf(
			"ORCHESTRATOR DIRECTIVE: Implementation work detected.\n"+
				"Active work: %s — Type: feature\n\n"+
				"REQUIRED: Delegate to a coder subagent.\n"+
				"DO NOT execute code directly in orchestrator context.",
			activeFeatureID,
		)
	}

	// Bug report when feature is active — suggest creating a bug.
	if intent.IsBugReport && hasActive && activeWorkType == "feature" {
		return fmt.Sprintf(
			"WORKFLOW GUIDANCE: Bug report detected.\n"+
				"Active work: %s — Type: feature\n\n"+
				"If this bug is part of the current feature, continue.\n"+
				"If separate, create a bug: sdk.bugs.create('Title').save()",
			activeFeatureID,
		)
	}

	// No active work item — nudge toward creating one.
	if !hasActive {
		if intent.IsImplementation {
			return "ORCHESTRATOR DIRECTIVE: Implementation work detected but no active work item.\n" +
				"REQUIRED: Create a feature, start it, then delegate to a coder subagent."
		}
		if intent.IsBugReport {
			return "WORKFLOW GUIDANCE: Bug report detected but no active work item.\n" +
				"Create a bug: sdk.bugs.create('Title').save() then sdk.bugs.start(id)"
		}
		if intent.IsInvestigation {
			return "WORKFLOW GUIDANCE: Investigation detected but no active work item.\n" +
				"Create a spike: sdk.spikes.create('Title').save() then sdk.spikes.start(id)"
		}
	}

	return ""
}

// cigsImperatives returns delegation imperative lines for exploration,
// code changes, or git operations. Mirrors generate_cigs_guidance() in Python.
func cigsImperatives(intent PromptIntent) string {
	var lines []string

	if intent.InvolvesExploration {
		lines = append(lines,
			"[CIGS] Exploration detected — consider delegating to researcher subagent.")
	}
	if intent.InvolvesCodeChanges {
		lines = append(lines,
			"[CIGS] Code changes detected — consider delegating to coder subagent.")
	}
	if intent.InvolvesGit {
		lines = append(lines,
			"[CIGS] Git operations detected — consider delegating to copilot subagent.")
	}

	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n")
}

// ---------- helpers ----------

// countKeywordHits returns how many keywords from the list appear in text.
func countKeywordHits(text string, keywords []string) int {
	n := 0
	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			n++
		}
	}
	return n
}

// matchesContinuation checks whether the prompt is a short continuation signal.
// We only match when the keyword appears at or near the start of the prompt.
func matchesContinuation(lower string) bool {
	for _, kw := range continuationKeywords {
		if strings.HasPrefix(lower, kw) {
			return true
		}
		// Also match if the entire prompt equals the keyword.
		if lower == kw {
			return true
		}
	}
	return false
}

func max64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func min64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
