package main

import (
	"database/sql"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/shakestzd/htmlgraph/internal/db"
	"github.com/shakestzd/htmlgraph/internal/hooks"
)

// hookCmd returns the "htmlgraph hook" parent command with all subcommands.
func hookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "hook",
		Short:         "Claude Code hook handlers (replaces Python hook scripts)",
		SilenceErrors: true,
		SilenceUsage:  true,
		Long: `Hook subcommands read a CloudEvent JSON payload from stdin and write a
JSON result to stdout. They replace the Python hook scripts, eliminating the
~500ms uv cold-start cost per hook invocation.

Usage in hooks.json:
  "command": "htmlgraph hook session-start"
  "command": "htmlgraph hook pretooluse"
  etc.`,
	}

	// Shared fallback results used across commands.
	continueResult := &hooks.HookResult{Continue: true}
	allowResult := &hooks.HookResult{} // Empty object = allow (avoids Claude Code "hook error" label)
	emptyResult := &hooks.HookResult{}

	cmd.AddCommand(
		// Session lifecycle — need projectDir passed to the handler.
		hookSubcmdWithProject("session-start", "Handle SessionStart event", emptyResult,
			func(event *hooks.CloudEvent, database *sql.DB, projectDir string) (*hooks.HookResult, error) {
				hooks.ApplyTraceparent()
				return hooks.SessionStart(event, database, projectDir)
			}),
		hookSubcmdWithProject("session-end", "Handle SessionEnd event", continueResult, hooks.SessionEnd),
		hookSubcmdWithProject("session-resume", "Handle SessionResume event", continueResult, hooks.SessionResume),

		// Standard two-arg handlers (event + db only).
		hookSubcmd("user-prompt", "Handle UserPromptSubmit event", emptyResult, hooks.UserPrompt),
		hookSubcmd("pretooluse", "Handle PreToolUse event", allowResult, hooks.PreToolUse),
		hookSubcmd("posttooluse", "Handle PostToolUse event", continueResult, hooks.PostToolUse),
		hookSubcmd("subagent-start", "Handle SubagentStart event", continueResult, hooks.SubagentStart),
		hookSubcmd("subagent-stop", "Handle SubagentStop event", continueResult, hooks.SubagentStop),
		hookSubcmd("stop", "Handle Stop event", continueResult, hooks.Stop),
		hookSubcmd("posttooluse-failure", "Handle PostToolUseFailure event", continueResult, hooks.PostToolUseFailure),
		hookSubcmd("pre-compact", "Handle PreCompact event", continueResult, hooks.PreCompact),
		hookSubcmd("post-compact", "Handle PostCompact event", continueResult, hooks.PostCompact),
		hookSubcmd("worktree-create", "Handle WorktreeCreate event", continueResult, hooks.WorktreeCreate),
		hookSubcmd("worktree-remove", "Handle WorktreeRemove event", continueResult, hooks.WorktreeRemove),
		hookSubcmd("teammate-idle", "Handle TeammateIdle event", continueResult, hooks.TeammateIdle),
		hookSubcmd("task-completed", "Handle TaskCompleted event", continueResult, hooks.TaskCompleted),
		hookSubcmd("instructions-loaded", "Handle InstructionsLoaded event", continueResult, hooks.InstructionsLoaded),
		hookSubcmd("permission-request", "Handle PermissionRequest event", continueResult, hooks.PermissionRequest),

		// track-event accepts an optional tool-name argument.
		hookTrackEventCmd(continueResult),
	)
	return cmd
}

// hookSubcmd creates a hook subcommand that resolves the project dir and opens
// the DB before calling handler. fallback is returned when the project is not
// an HtmlGraph project or when the DB cannot be opened.
func hookSubcmd(
	use, short string,
	fallback *hooks.HookResult,
	handler func(*hooks.CloudEvent, *sql.DB) (*hooks.HookResult, error),
) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runHook(func(event *hooks.CloudEvent) (*hooks.HookResult, error) {
				projectDir := hooks.ResolveProjectDir(event.CWD)
				if !hooks.IsHtmlGraphProject(projectDir) {
					return fallback, nil
				}
				database, err := db.Open(hooks.DBPath(projectDir))
				if err != nil {
					return fallback, nil
				}
				defer database.Close()
				return handler(event, database)
			})
		},
	}
}

// hookSubcmdWithProject is like hookSubcmd but also passes projectDir to the
// handler (needed by session-start, session-end, session-resume).
func hookSubcmdWithProject(
	use, short string,
	fallback *hooks.HookResult,
	handler func(*hooks.CloudEvent, *sql.DB, string) (*hooks.HookResult, error),
) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runHook(func(event *hooks.CloudEvent) (*hooks.HookResult, error) {
				projectDir := hooks.ResolveProjectDir(event.CWD)
				if !hooks.IsHtmlGraphProject(projectDir) {
					return fallback, nil
				}
				database, err := db.Open(hooks.DBPath(projectDir))
				if err != nil {
					return fallback, nil
				}
				defer database.Close()
				return handler(event, database, projectDir)
			})
		},
	}
}

// hookTrackEventCmd returns the track-event subcommand, which accepts an
// optional tool-name CLI argument.
func hookTrackEventCmd(fallback *hooks.HookResult) *cobra.Command {
	return &cobra.Command{
		Use:   "track-event [tool-name]",
		Short: "Record a generic hook event",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			toolName := "GenericEvent"
			if len(args) == 1 {
				toolName = args[0]
			}
			return runHook(func(event *hooks.CloudEvent) (*hooks.HookResult, error) {
				projectDir := hooks.ResolveProjectDir(event.CWD)
				if !hooks.IsHtmlGraphProject(projectDir) {
					return fallback, nil
				}
				database, err := db.Open(hooks.DBPath(projectDir))
				if err != nil {
					return fallback, nil
				}
				defer database.Close()
				return hooks.TrackEvent(toolName, event, database)
			})
		},
	}
}

// runHook is the common wrapper: read stdin, call the handler, write stdout.
// On any error it logs to debug.log and falls back to writing an empty JSON
// object so Claude is never blocked by a hook failure.
func runHook(handler func(*hooks.CloudEvent) (*hooks.HookResult, error)) error {
	event, err := hooks.ReadInput()
	if err != nil {
		hooks.LogError("runHook", "", fmt.Sprintf("read input: %v", err))
		// Always return a valid decision so Claude Code doesn't show "hook error"
		return hooks.Allow()
	}

	result, err := handler(event)
	if err != nil {
		hooks.LogError("runHook", event.SessionID, fmt.Sprintf("handler error: %v", err))
		return hooks.Allow()
	}
	if result == nil {
		hooks.LogError("runHook", event.SessionID, "handler returned nil result")
		return hooks.Allow()
	}
	return hooks.WriteResult(result)
}
