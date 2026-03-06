"""HtmlGraph CLI - WIP (Work In Progress) limit management commands.

Commands for managing WIP limits:
- wip: Show current WIP status and items
- wip reset: Reset an item from in-progress back to todo
"""

from __future__ import annotations

import argparse
from datetime import datetime, timezone
from typing import TYPE_CHECKING

from rich.console import Console

from htmlgraph.cli.base import BaseCommand, CommandError, CommandResult
from htmlgraph.cli.constants import DEFAULT_GRAPH_DIR

if TYPE_CHECKING:
    from argparse import _SubParsersAction


def register_wip_commands(subparsers: _SubParsersAction) -> None:
    """Register WIP management commands with the argument parser.

    Args:
        subparsers: Subparser action from ArgumentParser.add_subparsers()
    """
    # wip (main command with subcommands)
    wip_parser = subparsers.add_parser(
        "wip",
        help="Show and manage WIP (Work In Progress) limit status",
    )
    wip_subparsers = wip_parser.add_subparsers(
        dest="wip_command",
        help="WIP command",
    )

    # wip show (default - no subcommand needed)
    wip_show = wip_subparsers.add_parser(
        "show",
        help="Show WIP status (default action)",
    )
    wip_show.add_argument(
        "--graph-dir",
        "-g",
        default=DEFAULT_GRAPH_DIR,
        help="Graph directory",
    )
    wip_show.set_defaults(func=WipShowCommand.from_args)

    # wip reset - reset item from in-progress to todo
    wip_reset = wip_subparsers.add_parser(
        "reset",
        help="Reset an item from in-progress back to todo",
    )
    wip_reset.add_argument(
        "item_id",
        help="Item ID to reset (e.g., feat-12345 or spk-12345)",
    )
    wip_reset.add_argument(
        "--graph-dir",
        "-g",
        default=DEFAULT_GRAPH_DIR,
        help="Graph directory",
    )
    wip_reset.set_defaults(func=WipResetCommand.from_args)

    # Handle case where no subcommand is provided (show wip by default)
    wip_parser.set_defaults(func=WipShowCommand.from_args)


class WipShowCommand(BaseCommand):
    """Show current WIP (Work In Progress) status."""

    @classmethod
    def from_args(cls, args: argparse.Namespace) -> WipShowCommand:
        return cls()

    def execute(self) -> CommandResult:
        """Display WIP status with active items grouped by type."""
        from pathlib import Path

        console = Console()

        graph_dir = Path(self.graph_dir or DEFAULT_GRAPH_DIR)
        if not graph_dir.exists():
            raise CommandError(f"Graph directory not found: {graph_dir}")

        # Initialize SDK to get session manager
        sdk = self.get_sdk()
        session_manager = sdk.session_manager

        # Get active features
        active_items = session_manager.get_active_features()
        wip_limit = session_manager.wip_limit
        wip_count = len(active_items)

        console.print()

        # WIP status header
        wip_color = "red" if wip_count >= wip_limit else "green"
        console.print(
            f"[bold]WIP Limit:[/bold] [{wip_color}]{wip_count}/{wip_limit}[/{wip_color}]"
        )
        console.print()

        if not active_items:
            console.print("[dim]No items in progress[/dim]")
            console.print()
            return CommandResult(
                text="No items in progress",
                data={
                    "wip_count": wip_count,
                    "wip_limit": wip_limit,
                    "items": [],
                },
            )

        # Group items by type (features vs spikes)
        features = []
        spikes = []

        for item in active_items:
            item_id = getattr(item, "id", "?")
            title = getattr(item, "title", "Untitled")
            updated_at = getattr(item, "updated_at", None)
            created_at = getattr(item, "created_at", None)

            # Determine item type from ID prefix
            item_type = "feature" if item_id.startswith("feat-") else "spike"

            # Calculate age
            timestamp = updated_at or created_at
            if timestamp:
                try:
                    if isinstance(timestamp, str):
                        # Parse ISO format timestamp
                        item_datetime = datetime.fromisoformat(
                            timestamp.replace("Z", "+00:00")
                        )
                    else:
                        item_datetime = timestamp

                    now = datetime.now(timezone.utc)
                    age_delta = now - (
                        item_datetime
                        if item_datetime.tzinfo
                        else item_datetime.replace(tzinfo=timezone.utc)
                    )
                    age_seconds = age_delta.total_seconds()

                    # Format age
                    if age_seconds < 60:
                        age_str = f"{int(age_seconds)}s"
                    elif age_seconds < 3600:
                        age_str = f"{int(age_seconds // 60)}m"
                    elif age_seconds < 86400:
                        age_str = f"{int(age_seconds // 3600)}h"
                    else:
                        age_str = f"{int(age_seconds // 86400)}d"

                    # Flag stale items (older than 24h)
                    stale = " ← stale?" if age_seconds > 86400 else ""
                except (ValueError, TypeError):
                    age_str = "?"
                    stale = ""
            else:
                age_str = "?"
                stale = ""

            item_info = {
                "id": item_id,
                "title": title,
                "type": item_type,
                "age": age_str,
                "stale": bool(stale),
            }

            if item_type == "feature":
                features.append(item_info)
            else:
                spikes.append(item_info)

        # Display features section
        if features:
            console.print("[bold cyan]In Progress:[/bold cyan]")
            for entry in features:
                stale_marker = " [yellow]← stale?[/yellow]" if entry["stale"] else ""
                console.print(
                    f"  {entry['id']}  {entry['title']}  [dim][{entry['type']}][/dim]  "
                    f"started {entry['age']} ago{stale_marker}"
                )

        # Display spikes section
        if spikes:
            if features:
                console.print()
            console.print("[bold cyan]In Progress:[/bold cyan]")
            for entry in spikes:
                stale_marker = " [yellow]← stale?[/yellow]" if entry["stale"] else ""
                console.print(
                    f"  {entry['id']}  {entry['title']}  [dim][{entry['type']}][/dim]  "
                    f"started {entry['age']} ago{stale_marker}"
                )

        # Show reset command suggestion
        console.print()
        console.print("[dim]To reset a stale item:[/dim]")
        console.print("  [dim]uv run htmlgraph wip reset <id>[/dim]")
        console.print()

        # Prepare data for JSON output
        all_items = features + spikes
        items_data = [
            {
                "id": item["id"],
                "title": item["title"],
                "type": item["type"],
                "age": item["age"],
                "stale": item["stale"],
            }
            for item in all_items
        ]

        return CommandResult(
            text=f"WIP Status: {wip_count}/{wip_limit} items in progress",
            data={
                "wip_count": wip_count,
                "wip_limit": wip_limit,
                "items": items_data,
            },
        )


class WipResetCommand(BaseCommand):
    """Reset an item from in-progress back to todo."""

    def __init__(self, *, item_id: str) -> None:
        super().__init__()
        self.item_id = item_id

    @classmethod
    def from_args(cls, args: argparse.Namespace) -> WipResetCommand:
        return cls(item_id=args.item_id)

    def execute(self) -> CommandResult:
        """Reset item status from in-progress to todo."""
        from pathlib import Path

        from htmlgraph.graph import HtmlGraph

        console = Console()

        graph_dir = Path(self.graph_dir or DEFAULT_GRAPH_DIR)
        if not graph_dir.exists():
            raise CommandError(f"Graph directory not found: {graph_dir}")

        # Determine collection from ID prefix
        if self.item_id.startswith("feat-"):
            collection_name = "features"
        elif self.item_id.startswith("spk-"):
            collection_name = "spikes"
        elif self.item_id.startswith("bug-"):
            collection_name = "bugs"
        else:
            raise CommandError(
                f"Invalid item ID format: {self.item_id}. "
                "Expected format: feat-XXXXX, spk-XXXXX, or bug-XXXXX"
            )

        # Load the collection
        collection_dir = graph_dir / collection_name
        if not collection_dir.exists():
            raise CommandError(f"Collection not found: {collection_name}")

        graph = HtmlGraph(collection_dir, auto_load=True)

        # Find the item
        item = graph.get(self.item_id)
        if not item:
            raise CommandError(f"Item not found: {self.item_id}")

        # Check current status
        current_status = getattr(item, "status", None)
        if current_status != "in_progress":
            raise CommandError(
                f"Item {self.item_id} is not in progress (current status: {current_status})"
            )

        # Reset status to todo
        item.status = "todo"
        graph.update(item)

        console.print()
        console.print(f"[green]✓[/green] Reset {self.item_id} to todo")
        console.print()

        return CommandResult(
            text=f"Item {self.item_id} reset to todo",
            data={
                "item_id": self.item_id,
                "previous_status": current_status,
                "new_status": "todo",
            },
        )
