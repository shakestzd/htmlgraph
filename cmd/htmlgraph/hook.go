package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

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
		// Propagate the compiled version to the hooks package so session-start
		// can detect CLI/plugin version mismatches.
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			hooks.CLIVersion = version
		},
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
		hookSubcmd("task-created", "Handle TaskCreated event", continueResult, hooks.TaskCreated),
		hookSubcmd("instructions-loaded", "Handle InstructionsLoaded event", continueResult, hooks.InstructionsLoaded),
		hookSubcmd("permission-request", "Handle PermissionRequest event", continueResult, hooks.PermissionRequest),
		hookSubcmd("config-change", "Handle ConfigChange event — persist permission_mode to session metadata", continueResult, hooks.ConfigChange),
		hookSubcmdWithProject("exit-plan-mode", "Handle ExitPlanMode event — convert markdown plan to CRISPI YAML", continueResult, handleExitPlanMode),

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
				projectDir := hooks.ResolveProjectDir(event.CWD, event.SessionID)
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
				projectDir := hooks.ResolveProjectDir(event.CWD, event.SessionID)
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
				projectDir := hooks.ResolveProjectDir(event.CWD, event.SessionID)
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
// Timing is recorded via LogTimed so slow hooks are visible in debug.log.
func runHook(handler func(*hooks.CloudEvent) (*hooks.HookResult, error)) error {
	start := time.Now()

	event, err := hooks.ReadInput()
	if err != nil {
		hooks.LogError("runHook", "", fmt.Sprintf("read input: %v", err))
		// Always return a valid decision so Claude Code doesn't show "hook error"
		return hooks.Allow()
	}

	result, err := handler(event)
	if err != nil {
		// ErrBlockExit2 signals that the hook should exit with code 2 (block).
		// The handler already wrote its message to stderr; we just need to exit.
		var blockErr *hooks.BlockExit2Error
		if errors.As(err, &blockErr) {
			fmt.Fprintln(os.Stderr, blockErr.Message)
			os.Exit(2)
		}
		hooks.LogError("runHook", event.SessionID, fmt.Sprintf("handler error: %v", err))
		return hooks.Allow()
	}
	if result == nil {
		hooks.LogError("runHook", event.SessionID, "handler returned nil result")
		return hooks.Allow()
	}

	// Log timing for every hook invocation — helps identify slow handlers.
	// Use the cobra subcommand name (os.Args[2]) as the event label when available.
	projectDir := hooks.ResolveProjectDir(event.CWD, event.SessionID)
	hookName := ""
	if len(os.Args) >= 3 {
		hookName = os.Args[2]
	}
	hooks.LogTimed(projectDir, "runHook", map[string]string{
		"hook":    hookName,
		"session": event.SessionID[:hooks.MinSessionLen(event.SessionID)],
	}, start, "completed")

	return hooks.WriteResult(result)
}
