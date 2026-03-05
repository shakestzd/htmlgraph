from __future__ import annotations

"""HtmlGraph CLI - Main entry point.

Entry point with argument parsing and command routing.
Keeps main() thin by delegating to command modules.
"""


import argparse
import sys

from rich.console import Console

from htmlgraph.cli.constants import (
    DEFAULT_GRAPH_DIR,
    DEFAULT_OUTPUT_FORMAT,
    OUTPUT_FORMATS,
)


def create_parser() -> argparse.ArgumentParser:
    """Create and configure the argument parser.

    Returns:
        Configured ArgumentParser with all subcommands
    """
    parser = argparse.ArgumentParser(
        description="HtmlGraph - HTML is All You Need",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Quick Start:
  htmlgraph bootstrap               # One-command setup (< 60 seconds)

Examples:
  htmlgraph init                    # Initialize .htmlgraph in current dir
  htmlgraph serve                   # Start server on port 8080
  htmlgraph status                  # Show graph status
  htmlgraph query "[data-status='todo']"  # Query nodes

Session Management:
  htmlgraph session start           # Start a new session (auto-ID)
  htmlgraph session end my-session  # End a session
  htmlgraph session list            # List all sessions

Feature Management:
  htmlgraph feature list            # List all features
  htmlgraph feature start feat-001  # Start working on a feature
  htmlgraph feature complete feat-001  # Mark feature as done

Track Management:
  htmlgraph track new "User Auth"  # Create a new track
  htmlgraph track list              # List all tracks

Planning:
  htmlgraph plan show              # Show current plan
  htmlgraph plan create plan.json  # Create plan from JSON
  htmlgraph worktree setup         # Set up worktrees for tasks
  htmlgraph worktree status        # Show worktree status
  htmlgraph worktree merge task-1  # Merge completed task
  htmlgraph worktree cleanup       # Clean up worktrees

Analytics:
  htmlgraph analytics               # Project-wide analytics
  htmlgraph analytics --recent 10   # Analyze last 10 sessions

For more help: https://github.com/Shakes-tzd/htmlgraph
""",
    )

    # Global output control flags
    parser.add_argument(
        "--format",
        choices=OUTPUT_FORMATS,
        default=DEFAULT_OUTPUT_FORMAT,
        help="Output format: text (default), json, or plain",
    )
    parser.add_argument(
        "--quiet",
        "-q",
        action="store_true",
        help="Suppress progress messages and non-essential output",
    )
    parser.add_argument(
        "--verbose",
        "-v",
        action="count",
        default=0,
        help="Increase verbosity (can be used multiple times: -v, -vv, -vvv)",
    )

    subparsers = parser.add_subparsers(dest="command", help="Command to run")

    # Import command registration functions
    from htmlgraph.cli import analytics, core, plan, work

    # Register commands from each module
    core.register_commands(subparsers)
    work.register_commands(subparsers)
    analytics.register_commands(subparsers)
    plan.register_commands(subparsers)

    return parser


def main() -> None:
    """Main entry point for the CLI."""
    parser = create_parser()
    args = parser.parse_args()

    # If no command specified, show help
    if not args.command:
        parser.print_help()
        sys.exit(0)

    # Get the command handler (set by register_commands via set_defaults)
    if not hasattr(args, "func"):
        parser.print_help()
        sys.exit(1)

    # Determine graph directory and agent
    graph_dir = getattr(args, "graph_dir", DEFAULT_GRAPH_DIR)
    agent = getattr(args, "agent", None)

    # Get output format from args
    output_format = args.format

    # Execute the command
    try:
        # Create command instance from args
        command = args.func(args)

        # Run command with context
        command.run(
            graph_dir=graph_dir,
            agent=agent,
            output_format=output_format,
        )
    except KeyboardInterrupt:
        err_console = Console(stderr=True)
        err_console.print("\n\n[yellow]Interrupted by user[/yellow]")
        sys.exit(130)
    except Exception as e:
        from htmlgraph.cli.base import save_traceback

        err_console = Console(stderr=True)
        log_file = save_traceback(
            e, context={"command": args.command, "cwd": graph_dir}
        )
        err_console.print(f"[red]Error:[/red] {e}")
        err_console.print(f"[dim]Full traceback saved to:[/dim] {log_file}")
        sys.exit(1)


if __name__ == "__main__":
    main()
