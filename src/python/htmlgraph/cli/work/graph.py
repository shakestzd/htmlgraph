from __future__ import annotations

"""HtmlGraph CLI - Graph neighborhood command.

Shows the edge neighborhood of a node: direct incoming and outgoing edges
with their relationship types.
"""

import argparse
import sqlite3
from typing import TYPE_CHECKING

from rich.console import Console
from rich.table import Table

from htmlgraph.cli.base import BaseCommand, CommandResult
from htmlgraph.cli.constants import DEFAULT_GRAPH_DIR

if TYPE_CHECKING:
    from argparse import _SubParsersAction

console = Console()


def register_graph_commands(subparsers: _SubParsersAction) -> None:
    """Register graph neighborhood commands."""
    graph_parser = subparsers.add_parser(
        "graph",
        help="Show graph neighborhood (direct edges) for a node",
    )
    graph_parser.add_argument("id", help="Node ID to inspect")
    graph_parser.add_argument(
        "--graph-dir", "-g", default=DEFAULT_GRAPH_DIR, help="Graph directory"
    )
    graph_parser.add_argument(
        "--format", choices=["json", "text"], default="text", help="Output format"
    )
    graph_parser.set_defaults(func=GraphNeighborhoodCommand.from_args)


class GraphNeighborhoodCommand(BaseCommand):
    """Show the graph neighborhood (direct edges) of a node."""

    def __init__(self, *, node_id: str) -> None:
        super().__init__()
        self.node_id = node_id

    @classmethod
    def from_args(cls, args: argparse.Namespace) -> GraphNeighborhoodCommand:
        return cls(node_id=args.id)

    def execute(self) -> CommandResult:
        """Show node neighborhood with typed edges."""
        sdk = self.get_sdk()
        db_path = sdk._db.db_path

        # Look up node title from in-memory graph (try all collections)
        node_title = self._find_node_title(sdk, self.node_id)

        # Query edges from SQLite graph_edges table
        outgoing: list[dict] = []
        incoming: list[dict] = []

        conn = sqlite3.connect(str(db_path), timeout=2.0, check_same_thread=False)
        conn.row_factory = sqlite3.Row
        try:
            cursor = conn.execute(
                """
                SELECT edge_id, from_node_id, from_node_type,
                       to_node_id, to_node_type, relationship_type, weight, created_at
                FROM graph_edges
                WHERE from_node_id = ?
                ORDER BY relationship_type, created_at DESC
                """,
                (self.node_id,),
            )
            outgoing = [dict(row) for row in cursor.fetchall()]

            cursor = conn.execute(
                """
                SELECT edge_id, from_node_id, from_node_type,
                       to_node_id, to_node_type, relationship_type, weight, created_at
                FROM graph_edges
                WHERE to_node_id = ?
                ORDER BY relationship_type, created_at DESC
                """,
                (self.node_id,),
            )
            incoming = [dict(row) for row in cursor.fetchall()]
        finally:
            conn.close()

        # Also collect edges from in-memory graph (HTML edges)
        html_outgoing, html_incoming = self._collect_html_edges(sdk, self.node_id)

        # Build output table
        header = f"[bold cyan]{self.node_id}[/bold cyan]"
        if node_title:
            header += f" [yellow]({node_title})[/yellow]"

        table = Table(show_header=True, header_style="bold magenta", show_lines=False)
        table.add_column("Direction", style="dim", width=4)
        table.add_column("Relationship", style="green")
        table.add_column("Node", style="cyan")

        # Outgoing edges from SQLite
        for edge in outgoing:
            target_id = edge["to_node_id"]
            target_title = self._find_node_title(sdk, target_id)
            target_label = (
                f"{target_id} ({target_title})" if target_title else target_id
            )
            table.add_row("→", edge["relationship_type"], target_label)

        # Outgoing edges from HTML graph (de-duplicated against SQLite)
        sqlite_out_pairs = {(e["to_node_id"], e["relationship_type"]) for e in outgoing}
        for target_id, rel_type in html_outgoing:
            if (target_id, rel_type) not in sqlite_out_pairs:
                target_title = self._find_node_title(sdk, target_id)
                target_label = (
                    f"{target_id} ({target_title})" if target_title else target_id
                )
                table.add_row("→", rel_type, target_label)

        # Incoming edges from SQLite
        for edge in incoming:
            source_id = edge["from_node_id"]
            source_title = self._find_node_title(sdk, source_id)
            source_label = (
                f"{source_id} ({source_title})" if source_title else source_id
            )
            table.add_row("←", edge["relationship_type"], source_label)

        # Incoming edges from HTML graph (de-duplicated)
        sqlite_in_pairs = {
            (e["from_node_id"], e["relationship_type"]) for e in incoming
        }
        for source_id, rel_type in html_incoming:
            if (source_id, rel_type) not in sqlite_in_pairs:
                source_title = self._find_node_title(sdk, source_id)
                source_label = (
                    f"{source_id} ({source_title})" if source_title else source_id
                )
                table.add_row("←", rel_type, source_label)

        total_edges = (
            len(outgoing) + len(html_outgoing) + len(incoming) + len(html_incoming)
        )

        if total_edges == 0:
            from htmlgraph.cli.base import TextOutputBuilder

            output = TextOutputBuilder()
            output.add_field("Node", header)
            output.add_warning("No edges found for this node.")
            return CommandResult(
                text=output.build(),
                json_data={
                    "node_id": self.node_id,
                    "title": node_title,
                    "outgoing": outgoing,
                    "incoming": incoming,
                },
            )

        console.print(f"\nNode: {header}")
        console.print("[dim]Edges:[/dim]")

        return CommandResult(
            data=table,
            json_data={
                "node_id": self.node_id,
                "title": node_title,
                "outgoing": outgoing
                + [
                    {
                        "from_node_id": self.node_id,
                        "to_node_id": t,
                        "relationship_type": r,
                    }
                    for t, r in html_outgoing
                ],
                "incoming": incoming
                + [
                    {
                        "from_node_id": s,
                        "to_node_id": self.node_id,
                        "relationship_type": r,
                    }
                    for s, r in html_incoming
                ],
            },
        )

    def _find_node_title(self, sdk: object, node_id: str) -> str | None:
        """Look up a node title across all collections."""
        from htmlgraph.sdk import SDK

        if not isinstance(sdk, SDK):
            return None

        for collection_name in ("features", "bugs", "spikes"):
            coll = getattr(sdk, collection_name, None)
            if coll is None:
                continue
            try:
                node = coll.get(node_id)
                if node is not None:
                    return str(node.title)
            except Exception:
                continue
        return None

    def _collect_html_edges(
        self, sdk: object, node_id: str
    ) -> tuple[list[tuple[str, str]], list[tuple[str, str]]]:
        """
        Collect edges from the in-memory HTML graph for the given node.

        Returns:
            Tuple of (outgoing, incoming) where each is a list of (other_id, rel_type)
        """
        from htmlgraph.sdk import SDK

        if not isinstance(sdk, SDK):
            return [], []

        outgoing: list[tuple[str, str]] = []
        incoming: list[tuple[str, str]] = []

        # Check outgoing edges from the node's own HTML data
        for collection_name in ("features", "bugs", "spikes"):
            coll = getattr(sdk, collection_name, None)
            if coll is None:
                continue
            try:
                node = coll.get(node_id)
                if node is not None:
                    for rel_type, edges in node.edges.items():
                        for edge in edges:
                            outgoing.append((edge.target_id, rel_type))
            except Exception:
                continue

        # Check incoming edges: scan all nodes for edges pointing TO node_id
        for collection_name in ("features", "bugs", "spikes"):
            coll = getattr(sdk, collection_name, None)
            if coll is None:
                continue
            try:
                graph = coll._ensure_graph()
                in_edges = graph.get_incoming_edges(node_id)
                for edge_ref in in_edges:
                    incoming.append((edge_ref.source_id, edge_ref.relationship))
            except Exception:
                continue

        return outgoing, incoming
