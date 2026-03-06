from __future__ import annotations

"""HtmlGraph CLI - Ingest commands.

Commands for ingesting sessions from external AI tool formats:
- ingest claude-code: Import Claude Code native JSONL sessions
"""

import argparse
from pathlib import Path
from typing import TYPE_CHECKING

from rich import box
from rich.console import Console
from rich.table import Table

from htmlgraph.cli.base import BaseCommand, CommandError, CommandResult
from htmlgraph.cli.constants import DEFAULT_GRAPH_DIR

if TYPE_CHECKING:
    from argparse import _SubParsersAction

console = Console()


def register_ingest_commands(subparsers: _SubParsersAction) -> None:
    """Register ingest commands with the argument parser."""
    ingest_parser = subparsers.add_parser(
        "ingest",
        help="Ingest sessions from external AI tools (Claude Code, etc.)",
    )
    ingest_subparsers = ingest_parser.add_subparsers(
        dest="ingest_command",
        help="Ingest source",
    )

    # ingest claude-code
    cc_parser = ingest_subparsers.add_parser(
        "claude-code",
        help="Ingest Claude Code native JSONL session files",
    )
    cc_parser.add_argument(
        "--path",
        "-p",
        help=(
            "Path to a directory containing *.jsonl files, or a single *.jsonl file. "
            "Defaults to the Claude Code project directory for the current working "
            "directory (~/.claude/projects/[encoded-cwd]/)."
        ),
    )
    cc_parser.add_argument(
        "--project",
        help=(
            "Project directory to look up in ~/.claude/projects/. "
            "Defaults to the current working directory."
        ),
    )
    cc_parser.add_argument(
        "--agent",
        default="claude-code",
        help="Agent name to assign to ingested sessions (default: claude-code)",
    )
    cc_parser.add_argument(
        "--limit",
        type=int,
        default=None,
        help="Maximum number of sessions to ingest (newest first)",
    )
    cc_parser.add_argument(
        "--since",
        help="Only ingest sessions started after this date (ISO format, e.g. 2026-01-01)",
    )
    cc_parser.add_argument(
        "--overwrite",
        action="store_true",
        help="Re-ingest sessions that were already imported",
    )
    cc_parser.add_argument(
        "--graph-dir",
        "-g",
        default=DEFAULT_GRAPH_DIR,
        help="HtmlGraph directory (default: .htmlgraph)",
    )
    cc_parser.set_defaults(func=IngestClaudeCodeCommand.from_args)


# ============================================================================
# Command classes
# ============================================================================


class IngestClaudeCodeCommand(BaseCommand):
    """Ingest Claude Code native JSONL session files into HtmlGraph."""

    def __init__(
        self,
        *,
        path: str | None,
        project: str | None,
        agent: str,
        limit: int | None,
        since: str | None,
        overwrite: bool,
    ) -> None:
        super().__init__()
        self.path = path
        self.project = project
        self.agent = agent
        self.limit = limit
        self.since_str = since
        self.overwrite = overwrite

    @classmethod
    def from_args(cls, args: argparse.Namespace) -> IngestClaudeCodeCommand:
        return cls(
            path=getattr(args, "path", None),
            project=getattr(args, "project", None),
            agent=args.agent,
            limit=getattr(args, "limit", None),
            overwrite=args.overwrite,
            since=getattr(args, "since", None),
        )

    def execute(self) -> CommandResult:
        """Execute the Claude Code ingest command."""
        from datetime import datetime, timezone

        from htmlgraph.cli.base import TextOutputBuilder
        from htmlgraph.ingest.claude_code import ClaudeCodeIngester

        if self.graph_dir is None:
            raise CommandError("Missing graph directory")

        graph_dir = Path(self.graph_dir)
        if not graph_dir.exists():
            raise CommandError(
                f"HtmlGraph directory not found: {graph_dir}. "
                "Run 'htmlgraph init' first."
            )

        # Parse --since date
        since: datetime | None = None
        if self.since_str:
            try:
                since = datetime.fromisoformat(self.since_str).replace(
                    tzinfo=timezone.utc
                )
            except ValueError:
                raise CommandError(
                    f"Invalid --since date: {self.since_str!r}. "
                    "Use ISO format, e.g. 2026-01-01 or 2026-01-01T10:00:00"
                )

        ingester = ClaudeCodeIngester(
            graph_dir=graph_dir,
            agent=self.agent or "claude-code",
            overwrite=self.overwrite,
        )

        with console.status("[blue]Ingesting Claude Code sessions...", spinner="dots"):
            if self.path:
                summary = ingester.ingest_from_path(
                    path=Path(self.path),
                    limit=self.limit,
                    since=since,
                )
            else:
                project_path = Path(self.project) if self.project else None
                summary = ingester.ingest_project(
                    project_path=project_path,
                    limit=self.limit,
                    since=since,
                )

        # Build text output
        output = TextOutputBuilder()

        if summary.sessions_processed == 0 and not summary.errors:
            output.add_warning(
                "No Claude Code sessions found to ingest. "
                "Check that ~/.claude/projects/ contains sessions for this project, "
                "or specify --path to a directory of JSONL files."
            )
            return CommandResult(
                text=output.build(),
                json_data=self._summary_to_dict(summary),
            )

        output.add_success(
            f"Ingested {summary.sessions_processed} session(s) from Claude Code"
        )
        output.add_field("Created", summary.sessions_created)
        output.add_field("Updated", summary.sessions_updated)
        output.add_field("Skipped", summary.sessions_skipped)
        output.add_field("Total events imported", summary.total_events_imported)

        if summary.errors:
            output.add_warning(f"{len(summary.errors)} error(s) encountered")
            for err in summary.errors[:5]:
                output.add_line(f"  - {err}")

        # Build results table
        if summary.results:
            table = Table(
                title="Ingested Sessions",
                show_header=True,
                header_style="bold magenta",
                box=box.ROUNDED,
            )
            table.add_column("Claude Code ID", style="cyan", max_width=20)
            table.add_column("HtmlGraph ID", style="green", width=18)
            table.add_column("Status", style="yellow", width=10)
            table.add_column("Events", justify="right", style="blue", width=8)
            table.add_column("Branch", style="white", max_width=20)

            for result in summary.results:
                status = "updated" if result.was_existing else "created"
                # Truncate session_id for display
                cc_id = (
                    result.session_id[:16] + "..."
                    if len(result.session_id) > 16
                    else result.session_id
                )

                # Get branch from the session file if possible
                branch = "-"
                if result.output_path and result.output_path.exists():
                    try:
                        from htmlgraph.converter import html_to_session

                        sess = html_to_session(result.output_path)
                        branch = sess.transcript_git_branch or "-"
                    except Exception:
                        pass

                table.add_row(
                    cc_id,
                    result.htmlgraph_session_id,
                    status,
                    str(result.imported),
                    branch,
                )

            return CommandResult(
                data=table,
                text=output.build(),
                json_data=self._summary_to_dict(summary),
            )

        return CommandResult(
            text=output.build(),
            json_data=self._summary_to_dict(summary),
        )

    @staticmethod
    def _summary_to_dict(summary: object) -> dict:
        """Convert IngestSummary to a plain dict for JSON output."""
        from htmlgraph.ingest.claude_code import IngestSummary

        if not isinstance(summary, IngestSummary):
            return {}

        return {
            "sessions_processed": summary.sessions_processed,
            "sessions_created": summary.sessions_created,
            "sessions_updated": summary.sessions_updated,
            "sessions_skipped": summary.sessions_skipped,
            "total_events_imported": summary.total_events_imported,
            "errors": summary.errors,
            "results": [
                {
                    "session_id": r.session_id,
                    "htmlgraph_session_id": r.htmlgraph_session_id,
                    "imported": r.imported,
                    "skipped": r.skipped,
                    "total_entries": r.total_entries,
                    "was_existing": r.was_existing,
                    "output_path": str(r.output_path) if r.output_path else None,
                    "errors": r.errors,
                }
                for r in summary.results
            ],
        }
