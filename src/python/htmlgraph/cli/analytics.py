from __future__ import annotations

"""HtmlGraph CLI - Analytics and reporting commands.

Commands for analytics and reporting:
- analytics: Project-wide analytics
- cigs: Cost dashboard and attribution
- transcripts: Transcript management
- sync-docs: Documentation synchronization
"""


import argparse
import json
import webbrowser
from datetime import datetime
from pathlib import Path
from typing import TYPE_CHECKING

from pydantic import BaseModel, Field
from rich import box
from rich.console import Console
from rich.panel import Panel
from rich.table import Table

from htmlgraph.cli.base import BaseCommand, CommandError, CommandResult
from htmlgraph.cli.constants import DEFAULT_GRAPH_DIR

if TYPE_CHECKING:
    from argparse import _SubParsersAction

console = Console()


# ============================================================================
# Command Registration
# ============================================================================


def register_commands(subparsers: _SubParsersAction) -> None:
    """Register analytics and reporting commands with the argument parser.

    Args:
        subparsers: Subparser action from ArgumentParser.add_subparsers()
    """
    # Analytics command
    analytics_parser = subparsers.add_parser(
        "analytics", help="Project-wide analytics and insights"
    )
    analytics_parser.add_argument(
        "--graph-dir", "-g", default=DEFAULT_GRAPH_DIR, help="Graph directory"
    )
    analytics_parser.add_argument("--session-id", help="Analyze specific session")
    analytics_parser.add_argument(
        "--recent", type=int, metavar="N", help="Analyze recent N sessions"
    )
    analytics_parser.add_argument(
        "--agent", default="cli", help="Agent name for SDK initialization"
    )
    analytics_parser.add_argument(
        "--quiet", "-q", action="store_true", help="Suppress progress indicators"
    )
    analytics_parser.set_defaults(func=AnalyticsCommand.from_args)

    # CIGS commands
    _register_cigs_commands(subparsers)

    # Transcript commands
    _register_transcript_commands(subparsers)

    # Sync docs command
    _register_sync_docs_command(subparsers)

    # Costs command
    _register_costs_command(subparsers)

    # Search command
    _register_search_command(subparsers)


def _register_cigs_commands(subparsers: _SubParsersAction) -> None:
    """Register CIGS (Cost Intelligence & Governance System) commands."""
    cigs_parser = subparsers.add_parser("cigs", help="Cost dashboard and attribution")
    cigs_subparsers = cigs_parser.add_subparsers(
        dest="cigs_command", help="CIGS command"
    )

    # cigs cost-dashboard
    cost_dashboard = cigs_subparsers.add_parser(
        "cost-dashboard", help="Display cost summary dashboard"
    )
    cost_dashboard.add_argument(
        "--graph-dir", "-g", default=DEFAULT_GRAPH_DIR, help="Graph directory"
    )
    cost_dashboard.add_argument(
        "--save", action="store_true", help="Save to .htmlgraph/cost-dashboard.html"
    )
    cost_dashboard.add_argument(
        "--open", action="store_true", help="Open in browser after generation"
    )
    cost_dashboard.add_argument(
        "--json", action="store_true", help="Output JSON instead of HTML"
    )
    cost_dashboard.add_argument("--output", help="Custom output path")
    cost_dashboard.set_defaults(func=CostDashboardCommand.from_args)

    # cigs roi-analysis (Phase 1 OTEL ROI)
    roi_analysis = cigs_subparsers.add_parser(
        "roi-analysis", help="OTEL ROI analysis - cost attribution of Task delegations"
    )
    roi_analysis.add_argument(
        "--graph-dir", "-g", default=DEFAULT_GRAPH_DIR, help="Graph directory"
    )
    roi_analysis.add_argument(
        "--save", action="store_true", help="Save to .htmlgraph/cost-analysis.html"
    )
    roi_analysis.add_argument(
        "--open", action="store_true", help="Open in browser after generation"
    )
    roi_analysis.add_argument(
        "--json", action="store_true", help="Output JSON instead of HTML"
    )
    roi_analysis.add_argument("--output", help="Custom output path")
    # roi_analysis.set_defaults(func=OTELROIAnalysisCommand.from_args)  # TODO: Implement OTELROIAnalysisCommand

    # cigs status
    cigs_status = cigs_subparsers.add_parser("status", help="Show CIGS status")
    cigs_status.add_argument(
        "--graph-dir", "-g", default=DEFAULT_GRAPH_DIR, help="Graph directory"
    )
    cigs_status.set_defaults(func=CigsStatusCommand.from_args)

    # cigs summary
    cigs_summary = cigs_subparsers.add_parser("summary", help="Show cost summary")
    cigs_summary.add_argument(
        "--graph-dir", "-g", default=DEFAULT_GRAPH_DIR, help="Graph directory"
    )
    cigs_summary.add_argument("--session-id", help="Specific session ID")
    cigs_summary.set_defaults(func=CigsSummaryCommand.from_args)


def _register_transcript_commands(subparsers: _SubParsersAction) -> None:
    """Register transcript management commands."""
    transcript_parser = subparsers.add_parser(
        "transcript", help="Transcript management"
    )
    transcript_subparsers = transcript_parser.add_subparsers(
        dest="transcript_command", help="Transcript command"
    )

    # transcript list
    transcript_list = transcript_subparsers.add_parser("list", help="List transcripts")
    transcript_list.add_argument(
        "--graph-dir", "-g", default=DEFAULT_GRAPH_DIR, help="Graph directory"
    )
    transcript_list.add_argument("--format", choices=["text", "json"], default="text")
    transcript_list.add_argument("--limit", type=int, default=20)
    transcript_list.add_argument("--project", help="Filter by project path")
    transcript_list.set_defaults(func=TranscriptListCommand.from_args)

    # transcript import
    transcript_import = transcript_subparsers.add_parser(
        "import", help="Import transcript"
    )
    transcript_import.add_argument("session_id", help="Transcript session ID to import")
    transcript_import.add_argument(
        "--graph-dir", "-g", default=DEFAULT_GRAPH_DIR, help="Graph directory"
    )
    transcript_import.add_argument("--to-session", help="Target HtmlGraph session ID")
    transcript_import.add_argument("--agent", default="claude-code", help="Agent name")
    transcript_import.add_argument(
        "--overwrite", action="store_true", help="Overwrite existing events"
    )
    transcript_import.add_argument("--link-feature", help="Link to feature ID")
    transcript_import.add_argument("--format", choices=["text", "json"], default="text")
    transcript_import.set_defaults(func=TranscriptImportCommand.from_args)


def _register_sync_docs_command(subparsers: _SubParsersAction) -> None:
    """Register documentation synchronization command."""
    sync_docs = subparsers.add_parser(
        "sync-docs", help="Synchronize AI agent memory files"
    )
    sync_docs.add_argument(
        "--project-root", help="Project root directory (default: current directory)"
    )
    sync_docs.add_argument(
        "--check", action="store_true", help="Check synchronization status"
    )
    sync_docs.add_argument(
        "--generate",
        choices=["claude", "gemini"],
        help="Generate specific platform file",
    )
    sync_docs.add_argument(
        "--force", action="store_true", help="Force overwrite existing files"
    )
    sync_docs.set_defaults(func=SyncDocsCommand.from_args)


def _register_costs_command(subparsers: _SubParsersAction) -> None:
    """Register cost visibility and analysis command."""
    costs_parser = subparsers.add_parser(
        "costs",
        help="View token cost breakdown and analytics",
    )
    costs_parser.add_argument(
        "--graph-dir", "-g", default=DEFAULT_GRAPH_DIR, help="Graph directory"
    )
    costs_parser.add_argument(
        "--period",
        choices=["today", "day", "week", "month", "all"],
        default="week",
        help="Time period to analyze (default: week)",
    )
    costs_parser.add_argument(
        "--by",
        choices=["session", "feature", "tool", "agent"],
        default="session",
        help="Group costs by (default: session)",
    )
    costs_parser.add_argument(
        "--format",
        choices=["terminal", "csv"],
        default="terminal",
        help="Output format (default: terminal)",
    )
    costs_parser.add_argument(
        "--model",
        choices=["opus", "sonnet", "haiku", "auto"],
        default="auto",
        help="Claude model to assume for pricing (default: auto-detect)",
    )
    costs_parser.add_argument(
        "--limit",
        type=int,
        default=10,
        help="Maximum number of rows to display (default: 10)",
    )
    costs_parser.set_defaults(func=CostsCommand.from_args)


# ============================================================================
# Pydantic Models for Cost Analytics
# ============================================================================


class ToolCostData(BaseModel):
    """Cost data for a specific tool."""

    count: int = Field(ge=0)
    total_tokens: int = Field(ge=0)


class CategoryCostData(BaseModel):
    """Cost data for a category (delegation/direct)."""

    count: int = Field(ge=0)
    total_tokens: int = Field(ge=0)


class CostSummary(BaseModel):
    """Complete cost analysis summary."""

    total_cost_tokens: int = Field(ge=0)
    total_events: int = Field(ge=0)
    tool_costs: dict[str, ToolCostData] = Field(default_factory=dict)
    session_costs: dict[str, ToolCostData] = Field(default_factory=dict)
    delegation_count: int = Field(ge=0)
    direct_execution_count: int = Field(ge=0)
    cost_by_category: dict[str, CategoryCostData] = Field(default_factory=dict)

    @property
    def avg_cost_per_event(self) -> float:
        """Average token cost per event."""
        return (
            self.total_cost_tokens / self.total_events if self.total_events > 0 else 0
        )

    @property
    def delegation_percentage(self) -> float:
        """Percentage of events that were delegated."""
        return (
            self.delegation_count / self.total_events * 100
            if self.total_events > 0
            else 0
        )

    @property
    def estimated_cost_usd(self) -> float:
        """Estimated cost in USD (rough approximation)."""
        return self.total_cost_tokens / 1_000_000 * 5


# ============================================================================
# Command Implementations
# ============================================================================


class AnalyticsCommand(BaseCommand):
    """Project-wide analytics and insights."""

    def __init__(
        self, *, session_id: str | None, recent: int | None, agent: str, quiet: bool
    ) -> None:
        super().__init__()
        self.session_id = session_id
        self.recent = recent
        self.agent = agent
        self.quiet = quiet

    @classmethod
    def from_args(cls, args: argparse.Namespace) -> AnalyticsCommand:
        return cls(
            session_id=getattr(args, "session_id", None),
            recent=getattr(args, "recent", None),
            agent=getattr(args, "agent", "cli"),
            quiet=getattr(args, "quiet", False),
        )

    def execute(self) -> CommandResult:
        """Execute analytics analysis using analytics/cli.py implementation."""
        from htmlgraph.analytics.cli import cmd_analytics

        args = argparse.Namespace(
            graph_dir=self.graph_dir,
            session_id=self.session_id,
            recent=self.recent,
            agent=self.agent,
            quiet=self.quiet,
        )
        exit_code = cmd_analytics(args)
        if exit_code != 0:
            raise CommandError("Analytics command failed", exit_code=exit_code)
        return CommandResult(text="Analytics complete")


class CostDashboardCommand(BaseCommand):
    """Display cost summary dashboard."""

    def __init__(
        self,
        *,
        save: bool,
        open_browser: bool,
        json_output: bool,
        output_path: str | None,
    ) -> None:
        super().__init__()
        self.save = save
        self.open_browser = open_browser
        self.json_output = json_output
        self.output_path = output_path

    @classmethod
    def from_args(cls, args: argparse.Namespace) -> CostDashboardCommand:
        return cls(
            save=args.save,
            open_browser=getattr(args, "open", False),
            json_output=getattr(args, "json", False),
            output_path=getattr(args, "output", None),
        )

    def execute(self) -> CommandResult:
        """Generate and display cost dashboard."""
        if not self.graph_dir:
            raise CommandError("Graph directory not specified")
        graph_dir = Path(self.graph_dir)

        # Get events from database
        with console.status(
            "[blue]Analyzing HtmlGraph events...[/blue]", spinner="dots"
        ):
            try:
                from htmlgraph.operations.events import query_events

                result = query_events(graph_dir=graph_dir, limit=None)
                events = result.events if hasattr(result, "events") else []

                if not events:
                    console.print(
                        "[yellow]No events found. Run some work to generate analytics![/yellow]"
                    )
                    return CommandResult(text="No events to analyze")

                # Calculate costs
                cost_summary = self._analyze_event_costs(events)

            except Exception as e:
                console.print(f"[red]Error analyzing events: {e}[/red]")
                raise CommandError(f"Failed to analyze events: {e}")

        # Generate output
        if self.json_output:
            self._output_json(cost_summary)
        else:
            if self.save or self.output_path:
                html_file = self._save_html_dashboard(cost_summary, graph_dir)
                if self.open_browser:
                    webbrowser.open(f"file://{html_file.absolute()}")
                    console.print("[blue]Opening dashboard in browser...[/blue]")
            else:
                self._display_console_summary(cost_summary)

        # Print recommendations
        self._print_recommendations(cost_summary)

        return CommandResult(text="Cost dashboard generated")

    def _analyze_event_costs(self, events: list[dict]) -> CostSummary:
        """Analyze events and calculate cost attribution."""
        summary = CostSummary(
            total_events=len(events),
            total_cost_tokens=0,
            delegation_count=0,
            direct_execution_count=0,
        )

        for event in events:
            try:
                tool = event.get("tool", "unknown")
                session_id = event.get("session_id", "unknown")
                cost = (
                    event.get("predicted_tokens", 0)
                    or event.get("actual_tokens", 0)
                    or 2000
                )

                # Track by tool
                if tool not in summary.tool_costs:
                    summary.tool_costs[tool] = ToolCostData(count=0, total_tokens=0)
                summary.tool_costs[tool].count += 1
                summary.tool_costs[tool].total_tokens += cost

                # Track by session
                if session_id not in summary.session_costs:
                    summary.session_costs[session_id] = ToolCostData(
                        count=0, total_tokens=0
                    )
                summary.session_costs[session_id].count += 1
                summary.session_costs[session_id].total_tokens += cost

                # Track delegation vs direct
                delegation_tools = [
                    "Task",
                    "spawn_gemini",
                    "spawn_codex",
                    "spawn_copilot",
                ]
                if tool in delegation_tools:
                    summary.delegation_count += 1
                    category = "delegation"
                else:
                    summary.direct_execution_count += 1
                    category = "direct"

                if category not in summary.cost_by_category:
                    summary.cost_by_category[category] = CategoryCostData(
                        count=0, total_tokens=0
                    )
                summary.cost_by_category[category].count += 1
                summary.cost_by_category[category].total_tokens += cost

                summary.total_cost_tokens += cost

            except Exception:
                continue

        return summary

    def _output_json(self, summary: CostSummary) -> None:
        """Output cost data as JSON."""
        output_file = (
            Path(self.output_path) if self.output_path else Path("cost-summary.json")
        )
        output_file.write_text(summary.model_dump_json(indent=2))
        console.print(f"[green]✓ JSON output saved to: {output_file}[/green]")

    def _save_html_dashboard(self, summary: CostSummary, graph_dir: Path) -> Path:
        """Save HTML dashboard to file."""
        from htmlgraph.cli.templates.cost_dashboard import generate_html

        html_content = generate_html(summary)
        output_file = (
            Path(self.output_path)
            if self.output_path
            else graph_dir / "cost-dashboard.html"
        )
        output_file.write_text(html_content)
        console.print(f"[green]✓ Dashboard saved to: {output_file}[/green]")
        return output_file

    def _display_console_summary(self, summary: CostSummary) -> None:
        """Display cost summary in console."""
        from htmlgraph.cli.base import TableBuilder

        console.print("\n[bold cyan]Cost Dashboard Summary[/bold cyan]\n")

        # Summary table
        summary_builder = TableBuilder.create_list_table(title=None)
        summary_builder.add_column("Metric", style="cyan")
        summary_builder.add_column("Value", style="green")

        summary_builder.add_row("Total Events", str(summary.total_events))
        summary_builder.add_row("Total Cost", f"{summary.total_cost_tokens:,} tokens")
        summary_builder.add_row(
            "Average Cost", f"{summary.avg_cost_per_event:,.0f} tokens/event"
        )
        summary_builder.add_row("Estimated USD", f"${summary.estimated_cost_usd:.2f}")
        summary_builder.add_row("Delegation Count", str(summary.delegation_count))
        summary_builder.add_row(
            "Delegation Rate", f"{summary.delegation_percentage:.1f}%"
        )
        summary_builder.add_row(
            "Direct Executions", str(summary.direct_execution_count)
        )

        console.print(summary_builder.table)

        # Top tools table
        if summary.tool_costs:
            console.print("\n[bold cyan]Top Cost Drivers (by Tool)[/bold cyan]\n")
            tools_builder = TableBuilder.create_list_table(title=None)
            tools_builder.add_column("Tool", style="cyan")
            tools_builder.add_numeric_column("Count", style="green")
            tools_builder.add_numeric_column("Tokens", style="yellow")
            tools_builder.add_numeric_column("% Total", style="magenta")

            sorted_tools = sorted(
                summary.tool_costs.items(),
                key=lambda x: x[1].total_tokens,
                reverse=True,
            )
            for tool, data in sorted_tools[:10]:
                pct = data.total_tokens / summary.total_cost_tokens * 100
                tools_builder.add_row(
                    tool, str(data.count), f"{data.total_tokens:,}", f"{pct:.1f}%"
                )

            console.print(tools_builder.table)

    def _print_recommendations(self, summary: CostSummary) -> None:
        """Print cost optimization recommendations."""
        console.print("\n[bold cyan]Recommendations[/bold cyan]\n")

        recommendations = []

        if summary.delegation_percentage < 50:
            recommendations.append(
                "[yellow]→ Increase delegation usage[/yellow] - Consider using Task() and spawn_* for more operations"
            )

        if summary.tool_costs:
            top_tool, top_data = max(
                summary.tool_costs.items(), key=lambda x: x[1].total_tokens
            )
            top_pct = top_data.total_tokens / summary.total_cost_tokens * 100
            if top_pct > 40:
                recommendations.append(
                    f"[yellow]→ Review {top_tool} usage[/yellow] - It accounts for {top_pct:.1f}% of total cost"
                )

        if summary.total_events > 100:
            recommendations.append(
                "[green]✓ Good event volume[/green] - Sufficient data for optimization analysis"
            )

        recommendations.append(
            "[blue]💡 Tip: Use parallel Task() calls to reduce execution time by ~40%[/blue]"
        )

        for rec in recommendations:
            console.print(f"  {rec}")

        console.print()


class CigsStatusCommand(BaseCommand):
    """Show CIGS status."""

    @classmethod
    def from_args(cls, args: argparse.Namespace) -> CigsStatusCommand:
        return cls()

    def execute(self) -> CommandResult:
        """Show CIGS status."""
        from htmlgraph.cigs.autonomy import AutonomyRecommender
        from htmlgraph.cigs.pattern_storage import PatternStorage
        from htmlgraph.cigs.tracker import ViolationTracker

        if not self.graph_dir:
            raise CommandError("Graph directory not specified")
        graph_dir = Path(self.graph_dir)

        # Get violation tracker
        tracker = ViolationTracker(graph_dir)
        summary = tracker.get_session_violations()

        # Get pattern storage
        pattern_storage = PatternStorage(graph_dir)
        patterns = pattern_storage.get_anti_patterns()

        # Get autonomy recommendation
        recommender = AutonomyRecommender()
        autonomy = recommender.recommend(summary, patterns)

        # Display with Rich
        status_table = Table(title="CIGS Status", box=box.ROUNDED)
        status_table.add_column("Metric", style="cyan")
        status_table.add_column("Value", style="green")

        status_table.add_row("Session", summary.session_id)
        status_table.add_row("Violations", f"{summary.total_violations}/3")
        status_table.add_row("Compliance Rate", f"{summary.compliance_rate:.1%}")
        status_table.add_row("Total Waste", f"{summary.total_waste_tokens} tokens")
        status_table.add_row(
            "Circuit Breaker",
            "🚨 TRIGGERED" if summary.circuit_breaker_triggered else "Not triggered",
        )

        console.print(status_table)

        if summary.violations_by_type:
            console.print("\n[bold]Violation Breakdown:[/bold]")
            for vtype, count in summary.violations_by_type.items():
                console.print(f"  • {vtype}: {count}")

        console.print(f"\n[bold]Autonomy Level:[/bold] {autonomy.level.upper()}")
        console.print(
            f"[bold]Messaging Intensity:[/bold] {autonomy.messaging_intensity}"
        )
        console.print(f"[bold]Enforcement Mode:[/bold] {autonomy.enforcement_mode}")

        if patterns:
            console.print(f"\n[bold]Anti-Patterns Detected:[/bold] {len(patterns)}")
            for pattern in patterns[:3]:
                console.print(f"  • {pattern.name} ({pattern.occurrence_count}x)")

        return CommandResult(text="CIGS status displayed")


class CigsSummaryCommand(BaseCommand):
    """Show cost summary."""

    def __init__(self, *, session_id: str | None) -> None:
        super().__init__()
        self.session_id = session_id

    @classmethod
    def from_args(cls, args: argparse.Namespace) -> CigsSummaryCommand:
        return cls(session_id=getattr(args, "session_id", None))

    def execute(self) -> CommandResult:
        """Show cost summary."""
        from htmlgraph.cigs.tracker import ViolationTracker

        if not self.graph_dir:
            raise CommandError("Graph directory not specified")
        graph_dir = Path(self.graph_dir)
        tracker = ViolationTracker(graph_dir)

        # Get session ID
        session_id = self.session_id or tracker._session_id

        if not session_id:
            console.print(
                "[yellow]⚠️  No active session. Specify --session-id to view past sessions.[/yellow]"
            )
            return CommandResult(text="No active session")

        summary = tracker.get_session_violations(session_id)

        # Display summary
        panel = Panel(
            f"[cyan]Session ID:[/cyan] {summary.session_id}\n"
            f"[cyan]Total Violations:[/cyan] {summary.total_violations}\n"
            f"[cyan]Compliance Rate:[/cyan] {summary.compliance_rate:.1%}\n"
            f"[cyan]Total Waste:[/cyan] {summary.total_waste_tokens} tokens\n"
            f"[cyan]Circuit Breaker:[/cyan] {'🚨 TRIGGERED' if summary.circuit_breaker_triggered else 'Not triggered'}",
            title="CIGS Session Summary",
            border_style="cyan",
        )
        console.print(panel)

        if summary.violations_by_type:
            console.print("\n[bold]Violation Breakdown:[/bold]")
            for vtype, count in summary.violations_by_type.items():
                console.print(f"  • {vtype}: {count}")

        if summary.violations:
            console.print(
                f"\n[bold]Recent Violations ({len(summary.violations)}):[/bold]"
            )
            for v in summary.violations[-5:]:
                console.print(
                    f"  • {v.tool} - {v.violation_type} - {v.waste_tokens} tokens wasted"
                )
                console.print(f"    Should have: {v.should_have_delegated_to}")

        return CommandResult(text="Cost summary displayed")


class TranscriptListCommand(BaseCommand):
    """List transcripts."""

    def __init__(self, *, format: str, limit: int, project: str | None) -> None:
        super().__init__()
        self.format = format
        self.limit = limit
        self.project = project

    @classmethod
    def from_args(cls, args: argparse.Namespace) -> TranscriptListCommand:
        return cls(
            format=getattr(args, "format", "text"),
            limit=getattr(args, "limit", 20),
            project=getattr(args, "project", None),
        )

    def execute(self) -> CommandResult:
        """List all transcripts."""
        from htmlgraph.transcript import TranscriptReader

        reader = TranscriptReader()
        sessions = reader.list_sessions(project_path=self.project, limit=self.limit)

        if not sessions:
            if self.format == "json":
                console.print_json(json.dumps({"sessions": [], "count": 0}))
            else:
                console.print("[yellow]No Claude Code transcripts found.[/yellow]")
                console.print(f"[dim]Looked in: {reader.claude_dir}[/dim]")
            return CommandResult(text="No transcripts found")

        if self.format == "json":
            data = {
                "sessions": [
                    {
                        "session_id": s.session_id,
                        "path": str(s.path),
                        "cwd": s.cwd,
                        "git_branch": s.git_branch,
                        "started_at": s.started_at.isoformat()
                        if s.started_at
                        else None,
                        "user_messages": s.user_message_count,
                        "tool_calls": s.tool_call_count,
                        "duration_seconds": s.duration_seconds,
                    }
                    for s in sessions
                ],
                "count": len(sessions),
            }
            console.print_json(json.dumps(data))
        else:
            # Display with Rich table
            table = Table(
                title=f"Claude Code Transcripts ({len(sessions)} found)",
                box=box.ROUNDED,
            )
            table.add_column("Session ID", style="cyan", no_wrap=False, max_width=20)
            table.add_column("Started", style="dim")
            table.add_column("Duration", justify="right")
            table.add_column("Messages", justify="right")
            table.add_column("Branch", style="blue")

            for s in sessions:
                started = (
                    s.started_at.strftime("%Y-%m-%d %H:%M")
                    if s.started_at
                    else "unknown"
                )
                duration = (
                    f"{int(s.duration_seconds / 60)}m" if s.duration_seconds else "?"
                )
                branch = s.git_branch or "no branch"

                table.add_row(
                    s.session_id[:20] + "...",
                    started,
                    duration,
                    str(s.user_message_count),
                    branch,
                )

            console.print(table)

        return CommandResult(text=f"Listed {len(sessions)} transcripts")


class TranscriptImportCommand(BaseCommand):
    """Import transcript."""

    def __init__(
        self,
        *,
        session_id: str,
        to_session: str | None,
        agent: str,
        overwrite: bool,
        link_feature: str | None,
        format: str,
    ) -> None:
        super().__init__()
        self.session_id = session_id
        self.to_session = to_session
        self.agent = agent
        self.overwrite = overwrite
        self.link_feature = link_feature
        self.format = format

    @classmethod
    def from_args(cls, args: argparse.Namespace) -> TranscriptImportCommand:
        return cls(
            session_id=args.session_id,
            to_session=getattr(args, "to_session", None),
            agent=getattr(args, "agent", "claude-code"),
            overwrite=getattr(args, "overwrite", False),
            link_feature=getattr(args, "link_feature", None),
            format=getattr(args, "format", "text"),
        )

    def execute(self) -> CommandResult:
        """Import a transcript file."""
        from htmlgraph.session_manager import SessionManager
        from htmlgraph.transcript import TranscriptReader

        if not self.graph_dir:
            raise CommandError("Graph directory not specified")

        reader = TranscriptReader()
        manager = SessionManager(self.graph_dir)

        # Find the transcript
        transcript = reader.read_session(self.session_id)
        if not transcript:
            console.print(f"[red]Error: Transcript not found: {self.session_id}[/red]")
            return CommandResult(text="Transcript not found", exit_code=1)

        # Find or create HtmlGraph session
        htmlgraph_session_id = self.to_session
        if not htmlgraph_session_id:
            # Check if already linked
            existing = manager.find_session_by_transcript(self.session_id)
            if existing:
                htmlgraph_session_id = existing.id
                console.print(
                    f"[blue]Found existing linked session: {htmlgraph_session_id}[/blue]"
                )
            else:
                # Create new session
                new_session = manager.start_session(
                    agent=self.agent,
                    title=f"Imported: {transcript.session_id[:12]}",
                )
                htmlgraph_session_id = new_session.id
                console.print(
                    f"[green]Created new session: {htmlgraph_session_id}[/green]"
                )

        # Import events
        result = manager.import_transcript_events(
            session_id=htmlgraph_session_id,
            transcript_session=transcript,
            overwrite=self.overwrite,
        )

        # Link to feature if specified
        if self.link_feature:
            session = manager.get_session(htmlgraph_session_id)
            if session and self.link_feature not in session.worked_on:
                session.worked_on.append(self.link_feature)
                manager.session_converter.save(session)
                result["linked_feature"] = self.link_feature

        # Display results
        if self.format == "json":
            console.print_json(json.dumps(result))
        else:
            console.print(
                f"[green]✅ Imported transcript {self.session_id[:12]}:[/green]"
            )
            console.print(f"   → HtmlGraph session: {htmlgraph_session_id}")
            console.print(f"   → Events imported: {result.get('imported', 0)}")
            console.print(f"   → Events skipped: {result.get('skipped', 0)}")
            if result.get("linked_feature"):
                console.print(f"   → Linked to feature: {result['linked_feature']}")

        return CommandResult(text=f"Imported transcript: {self.session_id}")


class SyncDocsCommand(BaseCommand):
    """Synchronize AI agent memory files."""

    def __init__(
        self,
        *,
        project_root: str | None,
        check: bool,
        generate: str | None,
        force: bool,
    ) -> None:
        super().__init__()
        self.project_root = project_root
        self.check = check
        self.generate = generate
        self.force = force

    @classmethod
    def from_args(cls, args: argparse.Namespace) -> SyncDocsCommand:
        return cls(
            project_root=getattr(args, "project_root", None),
            check=getattr(args, "check", False),
            generate=getattr(args, "generate", None),
            force=getattr(args, "force", False),
        )

    def execute(self) -> CommandResult:
        """Synchronize AI agent memory files across platforms."""
        import os

        from htmlgraph.sync_docs import (
            PLATFORM_TEMPLATES,
            check_all_files,
            generate_platform_file,
            sync_all_files,
        )

        project_root = Path(self.project_root or os.getcwd()).resolve()

        if self.check:
            # Check mode
            console.print("[blue]🔍 Checking memory files...[/blue]")
            results = check_all_files(project_root)

            table = Table(title="Memory File Status", box=box.ROUNDED)
            table.add_column("File", style="cyan")
            table.add_column("Status", style="green")

            all_good = True
            for filename, status in results.items():
                if filename == "AGENTS.md":
                    if status:
                        table.add_row(filename, "✅ exists")
                    else:
                        table.add_row(filename, "❌ MISSING (required)")
                        all_good = False
                else:
                    if status:
                        table.add_row(filename, "✅ references AGENTS.md")
                    else:
                        table.add_row(filename, "⚠️  missing reference")
                        all_good = False

            console.print(table)

            if all_good:
                console.print(
                    "\n[green]✅ All files are properly synchronized![/green]"
                )
                return CommandResult(text="All files synchronized", exit_code=0)
            else:
                console.print("\n[yellow]⚠️  Some files need attention[/yellow]")
                return CommandResult(text="Files need attention", exit_code=1)

        elif self.generate:
            # Generate mode
            platform = self.generate.lower()
            console.print(
                f"[blue]📝 Generating {platform.upper()} memory file...[/blue]"
            )

            try:
                content = generate_platform_file(platform, project_root)
                template = PLATFORM_TEMPLATES[platform]
                filepath = project_root / template["filename"]

                if filepath.exists() and not self.force:
                    console.print(
                        f"[yellow]⚠️  {filepath.name} already exists. Use --force to overwrite.[/yellow]"
                    )
                    raise CommandError("File already exists")

                filepath.write_text(content)
                console.print(f"[green]✅ Created: {filepath}[/green]")
                console.print(
                    "\n[dim]The file references AGENTS.md for core documentation.[/dim]"
                )
                return CommandResult(text=f"Generated {platform} file")

            except ValueError as e:
                console.print(f"[red]❌ Error: {e}[/red]")
                return CommandResult(text=str(e), exit_code=1)

        else:
            # Sync mode (default)
            console.print("[blue]🔄 Synchronizing memory files...[/blue]")
            changes = sync_all_files(project_root)

            console.print("\n[bold]Results:[/bold]")
            for change in changes:
                console.print(f"  {change}")

            has_errors = any("⚠️" in c or "❌" in c for c in changes)
            return CommandResult(
                text="Synchronization complete",
                exit_code=1 if has_errors else 0,
            )


# ============================================================================
# Cost Command Implementation
# ============================================================================


class CostsCommand(BaseCommand):
    """View token cost breakdown and analytics by session, feature, or tool."""

    def __init__(
        self,
        *,
        period: str,
        by: str,
        format: str,
        model: str,
        limit: int,
    ) -> None:
        super().__init__()
        self.period = period
        self.by = by
        self.format = format
        self.model = model
        self.limit = limit

    @classmethod
    def from_args(cls, args: argparse.Namespace) -> CostsCommand:
        return cls(
            period=getattr(args, "period", "week"),
            by=getattr(args, "by", "session"),
            format=getattr(args, "format", "terminal"),
            model=getattr(args, "model", "auto"),
            limit=getattr(args, "limit", 10),
        )

    def execute(self) -> CommandResult:
        """Execute cost analysis and display results."""

        if not self.graph_dir:
            raise CommandError("Graph directory not specified")

        graph_dir = Path(self.graph_dir)
        db_path = graph_dir / "htmlgraph.db"

        if not db_path.exists():
            console.print(
                "[yellow]No HtmlGraph database found. Run some work to generate cost data![/yellow]"
            )
            return CommandResult(text="No database", exit_code=1)

        # Query costs from database
        with console.status("[blue]Analyzing costs...[/blue]", spinner="dots"):
            try:
                cost_data = self._query_costs(db_path)
            except Exception as e:
                raise CommandError(f"Failed to query costs: {e}")

        if not cost_data:
            console.print(
                "[yellow]No cost data found for the specified period.[/yellow]"
            )
            return CommandResult(text="No cost data")

        # Calculate USD costs based on model pricing
        cost_data = self._add_usd_costs(cost_data)

        # Display results
        if self.format == "csv":
            self._display_csv(cost_data)
        else:
            self._display_terminal(cost_data)

        # Display insights
        self._display_insights(cost_data)

        return CommandResult(text="Cost analysis complete")

    def _query_costs(self, db_path: Path) -> list[dict]:
        """Query costs from the database based on period and grouping."""
        import sqlite3
        from datetime import datetime, timezone

        conn = sqlite3.connect(str(db_path))
        conn.row_factory = sqlite3.Row
        cursor = conn.cursor()

        # Calculate time filter
        now = datetime.now(timezone.utc)
        time_filter = self._get_time_filter(now)

        # Build the query based on grouping
        if self.by == "session":
            query = """
            SELECT
                session_id as group_id,
                session_id as name,
                'session' as type,
                COUNT(*) as event_count,
                SUM(cost_tokens) as total_tokens,
                MIN(timestamp) as start_time,
                MAX(timestamp) as end_time
            FROM agent_events
            WHERE event_type IN ('tool_call', 'tool_result')
            AND cost_tokens > 0
            AND timestamp >= ?
            GROUP BY session_id
            ORDER BY total_tokens DESC
            LIMIT ?
            """
            cursor.execute(query, (time_filter, self.limit))

        elif self.by == "feature":
            query = """
            SELECT
                feature_id as group_id,
                COALESCE(feature_id, 'unlinked') as name,
                'feature' as type,
                COUNT(*) as event_count,
                SUM(cost_tokens) as total_tokens,
                MIN(timestamp) as start_time,
                MAX(timestamp) as end_time
            FROM agent_events
            WHERE event_type IN ('tool_call', 'tool_result')
            AND cost_tokens > 0
            AND timestamp >= ?
            GROUP BY feature_id
            ORDER BY total_tokens DESC
            LIMIT ?
            """
            cursor.execute(query, (time_filter, self.limit))

        elif self.by == "tool":
            query = """
            SELECT
                tool_name as group_id,
                tool_name as name,
                'tool' as type,
                COUNT(*) as event_count,
                SUM(cost_tokens) as total_tokens,
                MIN(timestamp) as start_time,
                MAX(timestamp) as end_time
            FROM agent_events
            WHERE event_type IN ('tool_call', 'tool_result')
            AND cost_tokens > 0
            AND timestamp >= ?
            GROUP BY tool_name
            ORDER BY total_tokens DESC
            LIMIT ?
            """
            cursor.execute(query, (time_filter, self.limit))

        elif self.by == "agent":
            query = """
            SELECT
                agent as group_id,
                agent as name,
                'agent' as type,
                COUNT(*) as event_count,
                SUM(cost_tokens) as total_tokens,
                MIN(timestamp) as start_time,
                MAX(timestamp) as end_time
            FROM agent_events
            WHERE event_type IN ('tool_call', 'tool_result')
            AND cost_tokens > 0
            AND timestamp >= ?
            GROUP BY agent
            ORDER BY total_tokens DESC
            LIMIT ?
            """
            cursor.execute(query, (time_filter, self.limit))

        results = []
        for row in cursor.fetchall():
            results.append(dict(row))

        conn.close()
        return results

    def _get_time_filter(self, now: datetime) -> str:
        """Get ISO format timestamp for time filtering."""
        from datetime import timedelta

        if self.period == "today":
            delta = timedelta(hours=24)
        elif self.period == "day":
            delta = timedelta(days=1)
        elif self.period == "week":
            delta = timedelta(days=7)
        elif self.period == "month":
            delta = timedelta(days=30)
        else:  # "all"
            delta = timedelta(days=36500)  # ~100 years

        cutoff = now - delta
        return cutoff.isoformat()

    def _add_usd_costs(self, cost_data: list[dict]) -> list[dict]:
        """Add USD cost estimates to cost data."""
        for item in cost_data:
            item["cost_usd"] = self._calculate_usd(item["total_tokens"])
        return cost_data

    def _calculate_usd(self, tokens: int) -> float:
        """Calculate USD cost from tokens based on model pricing."""
        # Claude pricing (per 1M tokens):
        # Opus: $15 input, $45 output
        # Sonnet: $3 input, $15 output
        # Haiku: $0.80 input, $4 output

        # Assume ~90% input, 10% output ratio
        input_ratio = 0.9
        output_ratio = 0.1

        if self.model == "opus" or (self.model == "auto"):
            # Default to Opus for conservative estimate
            input_cost = 15 / 1_000_000
            output_cost = 45 / 1_000_000
        elif self.model == "sonnet":
            input_cost = 3 / 1_000_000
            output_cost = 15 / 1_000_000
        elif self.model == "haiku":
            input_cost = 0.80 / 1_000_000
            output_cost = 4 / 1_000_000
        else:
            # Fallback to Opus
            input_cost = 15 / 1_000_000
            output_cost = 45 / 1_000_000

        cost = (tokens * input_ratio * input_cost) + (
            tokens * output_ratio * output_cost
        )
        return cost

    def _display_terminal(self, cost_data: list[dict]) -> None:
        """Display costs in terminal with rich formatting."""
        from htmlgraph.cli.base import TableBuilder

        # Period label
        period_label = self.period.upper()
        if self.period == "today":
            period_label = "TODAY"
        elif self.period == "day":
            period_label = "LAST 24 HOURS"
        elif self.period == "week":
            period_label = "LAST 7 DAYS"
        elif self.period == "month":
            period_label = "LAST 30 DAYS"

        console.print(f"\n[bold cyan]{period_label} - COST SUMMARY[/bold cyan]")
        console.print("[dim]═" * 60 + "[/dim]\n")

        # Build table
        table_builder = TableBuilder.create_list_table(title=None)
        table_builder.add_column("Name", style="cyan")
        table_builder.add_numeric_column("Events", style="green")
        table_builder.add_numeric_column("Tokens", style="yellow")
        table_builder.add_numeric_column("Estimated Cost", style="magenta")

        total_tokens = 0
        total_usd = 0.0

        for item in cost_data:
            name = item["name"] or "(unlinked)"
            if len(name) > 30:
                name = name[:27] + "..."

            events = f"{item['event_count']:,}"
            tokens = f"{item['total_tokens']:,}"
            cost_str = f"${item['cost_usd']:.2f}"

            table_builder.add_row(name, events, tokens, cost_str)

            total_tokens += item["total_tokens"]
            total_usd += item["cost_usd"]

        console.print(table_builder.table)

        # Summary
        console.print("\n[dim]─" * 60 + "[/dim]")
        console.print(
            f"[bold]Total Tokens:[/bold] {total_tokens:,} [dim]({self._format_duration(cost_data)})[/dim]"
        )
        console.print(
            f"[bold]Estimated Cost:[/bold] ${total_usd:.2f} ({self.model.upper() if self.model != 'auto' else 'Opus'})"
        )

        # Insights
        if len(cost_data) > 0:
            top_item = cost_data[0]
            pct = (
                (top_item["total_tokens"] / total_tokens * 100)
                if total_tokens > 0
                else 0
            )
            console.print(
                f"\n[dim]Most expensive:[/dim] [yellow]{top_item['name']}[/yellow] "
                f"[dim]({pct:.0f}% of total)[/dim]"
            )

    def _display_csv(self, cost_data: list[dict]) -> None:
        """Display costs in CSV format for spreadsheet analysis."""
        import csv
        import io

        output = io.StringIO()
        writer = csv.writer(output)

        # Header
        if self.by == "session":
            writer.writerow(["Session ID", "Events", "Tokens", "Estimated Cost (USD)"])
        else:
            writer.writerow(
                [
                    self.by.capitalize(),
                    "Events",
                    "Tokens",
                    "Estimated Cost (USD)",
                ]
            )

        # Data rows
        for item in cost_data:
            writer.writerow(
                [
                    item["name"],
                    item["event_count"],
                    item["total_tokens"],
                    f"{item['cost_usd']:.2f}",
                ]
            )

        # Totals
        total_tokens = sum(item["total_tokens"] for item in cost_data)
        total_usd = sum(item["cost_usd"] for item in cost_data)
        writer.writerow(["TOTAL", "", total_tokens, f"{total_usd:.2f}"])

        csv_content = output.getvalue()
        console.print(csv_content)

    def _display_insights(self, cost_data: list[dict]) -> None:
        """Display cost optimization insights."""
        if not cost_data:
            return

        console.print("\n[bold cyan]Insights & Recommendations[/bold cyan]")
        console.print("[dim]─" * 60 + "[/dim]\n")

        total_tokens = sum(item["total_tokens"] for item in cost_data)

        # Insight 1: Top cost driver
        top_item = cost_data[0]
        top_pct = (
            (top_item["total_tokens"] / total_tokens * 100) if total_tokens > 0 else 0
        )
        console.print(
            f"[blue]→ Highest cost:[/blue] {top_item['name']} "
            f"[yellow]({top_pct:.0f}% of total)[/yellow]"
        )

        # Insight 2: Concentration
        if len(cost_data) > 1:
            top_3_pct = (
                sum(item["total_tokens"] for item in cost_data[:3])
                / (total_tokens if total_tokens > 0 else 1)
                * 100
            )
            console.print(
                f"[blue]→ Cost concentration:[/blue] Top 3 account for [yellow]{top_3_pct:.0f}%[/yellow]"
            )

        # Insight 3: Recommendations
        if self.by == "tool" and top_item["name"] in ["Read", "Bash", "Grep"]:
            console.print(
                f"[yellow]→ Tip:[/yellow] {top_item['name']} is expensive. Consider batching operations "
                "or using more efficient approaches."
            )
        elif self.by == "session" and len(cost_data) > 5:
            console.print(
                "[yellow]→ Tip:[/yellow] Many sessions with costs. Consider consolidating work "
                "to fewer, focused sessions."
            )

        console.print()

    def _format_duration(self, cost_data: list[dict]) -> str:
        """Format duration from start/end times."""
        if not cost_data or "start_time" not in cost_data[0]:
            return "unknown"

        try:
            from datetime import datetime

            start_times = [
                datetime.fromisoformat(item["start_time"])
                for item in cost_data
                if item.get("start_time")
            ]
            end_times = [
                datetime.fromisoformat(item["end_time"])
                for item in cost_data
                if item.get("end_time")
            ]

            if not start_times or not end_times:
                return "unknown"

            earliest = min(start_times)
            latest = max(end_times)
            duration = latest - earliest

            hours = duration.total_seconds() / 3600
            if hours > 1:
                return f"{hours:.1f}h"
            else:
                minutes = duration.total_seconds() / 60
                return f"{minutes:.0f}m"
        except Exception:
            return "unknown"


# ============================================================================
# Search Command - FTS5 full-text search across sessions and events
# ============================================================================


def _register_search_command(subparsers: _SubParsersAction) -> None:
    """Register the 'search' command with the argument parser."""
    search_parser = subparsers.add_parser(
        "search",
        help="Full-text search across sessions and events (FTS5)",
    )
    search_parser.add_argument(
        "query",
        help="Search query (supports FTS5 syntax: phrases, boolean, column filters)",
    )
    search_parser.add_argument(
        "--limit",
        "-n",
        type=int,
        default=20,
        metavar="N",
        help="Maximum number of results to return (default: 20)",
    )
    search_parser.add_argument(
        "--events",
        action="store_true",
        help="Search agent events (tool calls) instead of sessions",
    )
    search_parser.add_argument(
        "--rebuild-index",
        action="store_true",
        help="Rebuild the FTS index before searching",
    )
    search_parser.add_argument(
        "--graph-dir", "-g", default=DEFAULT_GRAPH_DIR, help="Graph directory"
    )
    search_parser.set_defaults(func=SearchCommand.from_args)


class SearchCommand(BaseCommand):
    """Full-text search across HtmlGraph sessions and events."""

    def __init__(
        self,
        query: str,
        limit: int = 20,
        search_events: bool = False,
        rebuild_index: bool = False,
        graph_dir: str = DEFAULT_GRAPH_DIR,
    ) -> None:
        super().__init__()
        self.query = query
        self.limit = limit
        self.search_events = search_events
        self.rebuild_index = rebuild_index
        self.graph_dir = graph_dir

    @classmethod
    def from_args(cls, args: argparse.Namespace) -> SearchCommand:
        return cls(
            query=args.query,
            limit=args.limit,
            search_events=getattr(args, "events", False),
            rebuild_index=getattr(args, "rebuild_index", False),
            graph_dir=getattr(args, "graph_dir", DEFAULT_GRAPH_DIR),
        )

    def execute(self) -> CommandResult:
        from pathlib import Path

        from htmlgraph.search import build_fts_index, search_events, search_sessions

        # Resolve database path from graph_dir
        graph_dir = self.graph_dir or DEFAULT_GRAPH_DIR
        db_path = str(Path(graph_dir) / "htmlgraph.db")
        # Fall back to home default if not found in graph_dir
        if not Path(db_path).exists():
            db_path = str(Path.home() / ".htmlgraph" / "htmlgraph.db")

        if self.rebuild_index:
            total = build_fts_index(db_path)
            console.print(f"[green]FTS index rebuilt:[/green] {total} rows indexed")

        if self.search_events:
            results = search_events(self.query, db_path=db_path, limit=self.limit)
            return self._render_event_results(results)
        else:
            results = search_sessions(self.query, db_path=db_path, limit=self.limit)
            return self._render_session_results(results)

    def _render_session_results(self, results: list[dict]) -> CommandResult:
        if not results:
            console.print(f"[yellow]No sessions found matching:[/yellow] {self.query}")
            return CommandResult(data=[], json_data=[])

        table = Table(
            title=f"Sessions matching '{self.query}'",
            show_header=True,
            header_style="bold magenta",
        )
        table.add_column("Session ID", style="cyan", no_wrap=True)
        table.add_column("Agent", style="yellow", max_width=20)
        table.add_column("Status", style="green", width=12)
        table.add_column("Created", style="white", width=19)
        table.add_column("Snippet", style="dim", max_width=60)

        for row in results:
            snippet = row.get("snippet") or row.get("last_user_query") or ""
            created = (row.get("created_at") or "")[:19]
            table.add_row(
                row.get("session_id", ""),
                row.get("agent_assigned", ""),
                row.get("status", ""),
                created,
                snippet,
            )

        console.print(table)
        console.print(f"\n[dim]{len(results)} result(s)[/dim]")
        return CommandResult(data=table, json_data=results)

    def _render_event_results(self, results: list[dict]) -> CommandResult:
        if not results:
            console.print(f"[yellow]No events found matching:[/yellow] {self.query}")
            return CommandResult(data=[], json_data=[])

        table = Table(
            title=f"Events matching '{self.query}'",
            show_header=True,
            header_style="bold magenta",
        )
        table.add_column("Event ID", style="cyan", no_wrap=True, max_width=16)
        table.add_column("Session ID", style="blue", no_wrap=True, max_width=16)
        table.add_column("Tool", style="yellow", max_width=20)
        table.add_column("Timestamp", style="white", width=19)
        table.add_column("Snippet", style="dim", max_width=60)

        for row in results:
            snippet = row.get("snippet", "")
            ts = (row.get("timestamp") or "")[:19]
            table.add_row(
                row.get("event_id", ""),
                row.get("session_id", ""),
                row.get("tool_name", ""),
                ts,
                snippet,
            )

        console.print(table)
        console.print(f"\n[dim]{len(results)} result(s)[/dim]")
        return CommandResult(data=table, json_data=results)
