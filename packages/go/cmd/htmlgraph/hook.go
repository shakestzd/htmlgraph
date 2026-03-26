package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/shakestzd/htmlgraph/internal/db"
	"github.com/shakestzd/htmlgraph/internal/hooks"
)

// hookCmd returns the "htmlgraph hook" parent command with all subcommands.
func hookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hook",
		Short: "Claude Code hook handlers (replaces Python hook scripts)",
		Long: `Hook subcommands read a CloudEvent JSON payload from stdin and write a
JSON result to stdout. They replace the Python hook scripts, eliminating the
~500ms uv cold-start cost per hook invocation.

Usage in hooks.json:
  "command": "htmlgraph hook session-start"
  "command": "htmlgraph hook pretooluse"
  etc.`,
	}

	cmd.AddCommand(
		hookSessionStartCmd(),
		hookSessionEndCmd(),
		hookSessionResumeCmd(),
		hookUserPromptCmd(),
		hookPreToolUseCmd(),
		hookPostToolUseCmd(),
		hookSubagentStartCmd(),
		hookSubagentStopCmd(),
		hookTrackEventCmd(),
	)
	return cmd
}

// openDB resolves the project directory and opens the HtmlGraph SQLite database.
func openHookDB(event *hooks.CloudEvent) (*db_handle, error) {
	projectDir := hooks.ResolveProjectDir(event.CWD)
	if !hooks.IsHtmlGraphProject(projectDir) {
		return nil, nil // Not an HtmlGraph project — skip silently.
	}
	database, err := db.Open(hooks.DBPath(projectDir))
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	return &db_handle{db: database, projectDir: projectDir}, nil
}

// db_handle bundles the open database with its project directory.
type db_handle struct {
	db         interface{ Close() error }
	projectDir string
}

func hookSessionStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "session-start",
		Short: "Handle SessionStart event",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runHook(func(event *hooks.CloudEvent) (*hooks.HookResult, error) {
				projectDir := hooks.ResolveProjectDir(event.CWD)
				if !hooks.IsHtmlGraphProject(projectDir) {
					return nil, hooks.Empty()
				}
				database, err := db.Open(hooks.DBPath(projectDir))
				if err != nil {
					fmt.Fprintf(os.Stderr, "htmlgraph hook: %v\n", err)
					return nil, hooks.Empty()
				}
				defer database.Close()
				hooks.ApplyTraceparent()
				return hooks.SessionStart(event, database, projectDir)
			})
		},
	}
}

func hookSessionEndCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "session-end",
		Short: "Handle SessionEnd event",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runHook(func(event *hooks.CloudEvent) (*hooks.HookResult, error) {
				projectDir := hooks.ResolveProjectDir(event.CWD)
				if !hooks.IsHtmlGraphProject(projectDir) {
					return nil, hooks.Continue()
				}
				database, err := db.Open(hooks.DBPath(projectDir))
				if err != nil {
					fmt.Fprintf(os.Stderr, "htmlgraph hook: %v\n", err)
					return nil, hooks.Continue()
				}
				defer database.Close()
				return hooks.SessionEnd(event, database, projectDir)
			})
		},
	}
}

func hookSessionResumeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "session-resume",
		Short: "Handle SessionResume event",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runHook(func(event *hooks.CloudEvent) (*hooks.HookResult, error) {
				projectDir := hooks.ResolveProjectDir(event.CWD)
				if !hooks.IsHtmlGraphProject(projectDir) {
					return nil, hooks.Continue()
				}
				database, err := db.Open(hooks.DBPath(projectDir))
				if err != nil {
					fmt.Fprintf(os.Stderr, "htmlgraph hook: %v\n", err)
					return nil, hooks.Continue()
				}
				defer database.Close()
				return hooks.SessionResume(event, database, projectDir)
			})
		},
	}
}

func hookUserPromptCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "user-prompt",
		Short: "Handle UserPromptSubmit event",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runHook(func(event *hooks.CloudEvent) (*hooks.HookResult, error) {
				projectDir := hooks.ResolveProjectDir(event.CWD)
				if !hooks.IsHtmlGraphProject(projectDir) {
					return nil, hooks.Empty()
				}
				database, err := db.Open(hooks.DBPath(projectDir))
				if err != nil {
					fmt.Fprintf(os.Stderr, "htmlgraph hook: %v\n", err)
					return nil, hooks.Empty()
				}
				defer database.Close()
				return hooks.UserPrompt(event, database)
			})
		},
	}
}

func hookPreToolUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pretooluse",
		Short: "Handle PreToolUse event",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runHook(func(event *hooks.CloudEvent) (*hooks.HookResult, error) {
				projectDir := hooks.ResolveProjectDir(event.CWD)
				if !hooks.IsHtmlGraphProject(projectDir) {
					return nil, hooks.Allow()
				}
				database, err := db.Open(hooks.DBPath(projectDir))
				if err != nil {
					fmt.Fprintf(os.Stderr, "htmlgraph hook: %v\n", err)
					return nil, hooks.Allow()
				}
				defer database.Close()
				return hooks.PreToolUse(event, database)
			})
		},
	}
}

func hookPostToolUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "posttooluse",
		Short: "Handle PostToolUse event",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runHook(func(event *hooks.CloudEvent) (*hooks.HookResult, error) {
				projectDir := hooks.ResolveProjectDir(event.CWD)
				if !hooks.IsHtmlGraphProject(projectDir) {
					return nil, hooks.Continue()
				}
				database, err := db.Open(hooks.DBPath(projectDir))
				if err != nil {
					fmt.Fprintf(os.Stderr, "htmlgraph hook: %v\n", err)
					return nil, hooks.Continue()
				}
				defer database.Close()
				return hooks.PostToolUse(event, database)
			})
		},
	}
}

func hookSubagentStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "subagent-start",
		Short: "Handle SubagentStart event",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runHook(func(event *hooks.CloudEvent) (*hooks.HookResult, error) {
				projectDir := hooks.ResolveProjectDir(event.CWD)
				if !hooks.IsHtmlGraphProject(projectDir) {
					return nil, hooks.Continue()
				}
				database, err := db.Open(hooks.DBPath(projectDir))
				if err != nil {
					fmt.Fprintf(os.Stderr, "htmlgraph hook: %v\n", err)
					return nil, hooks.Continue()
				}
				defer database.Close()
				return hooks.SubagentStart(event, database)
			})
		},
	}
}

func hookSubagentStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "subagent-stop",
		Short: "Handle SubagentStop event",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runHook(func(event *hooks.CloudEvent) (*hooks.HookResult, error) {
				projectDir := hooks.ResolveProjectDir(event.CWD)
				if !hooks.IsHtmlGraphProject(projectDir) {
					return nil, hooks.Continue()
				}
				database, err := db.Open(hooks.DBPath(projectDir))
				if err != nil {
					fmt.Fprintf(os.Stderr, "htmlgraph hook: %v\n", err)
					return nil, hooks.Continue()
				}
				defer database.Close()
				return hooks.SubagentStop(event, database)
			})
		},
	}
}

func hookTrackEventCmd() *cobra.Command {
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
					return nil, hooks.Continue()
				}
				database, err := db.Open(hooks.DBPath(projectDir))
				if err != nil {
					fmt.Fprintf(os.Stderr, "htmlgraph hook: %v\n", err)
					return nil, hooks.Continue()
				}
				defer database.Close()
				return hooks.TrackEvent(toolName, event, database)
			})
		},
	}
}

// runHook is the common wrapper: read stdin, call the handler, write stdout.
// On any error it falls back to writing an empty JSON object so Claude is
// never blocked by a hook failure.
func runHook(handler func(*hooks.CloudEvent) (*hooks.HookResult, error)) error {
	event, err := hooks.ReadInput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "htmlgraph hook: read input: %v\n", err)
		return hooks.Empty()
	}

	result, err := handler(event)
	if err != nil {
		fmt.Fprintf(os.Stderr, "htmlgraph hook: handler error: %v\n", err)
		return hooks.Empty()
	}
	if result == nil {
		return nil // handler already wrote to stdout
	}
	return hooks.WriteResult(result)
}
