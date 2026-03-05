"""
Immutable graph snapshot for concurrent read access.

Provides a frozen copy of graph state that won't be affected by subsequent
mutations. Safe for multi-agent and multi-thread scenarios.
"""

from __future__ import annotations

from collections.abc import Callable, Iterator
from pathlib import Path

from htmlgraph.models import Node
from htmlgraph.parser import HtmlParser


class GraphSnapshot:
    """
    Immutable snapshot of graph state at a point in time.

    Provides read-only access to graph data without affecting the original graph.
    Safe to use across multiple agents or threads.

    Example:
        snapshot = graph.snapshot()
        node = snapshot.get("feature-001")  # Read-only access
        results = snapshot.query("[data-status='blocked']")
    """

    def __init__(self, nodes: dict[str, Node], directory: Path):
        """
        Create a snapshot of graph nodes.

        Args:
            nodes: Dictionary of nodes to snapshot
            directory: Graph directory (for context)
        """
        # Deep copy to prevent external mutations
        self._nodes = {
            node_id: node.model_copy(deep=True) for node_id, node in nodes.items()
        }
        self._directory = directory

    def get(self, node_id: str) -> Node | None:
        """
        Get a node by ID from the snapshot.

        Args:
            node_id: Node identifier

        Returns:
            Node instance or None if not found
        """
        node = self._nodes.get(node_id)
        # Return a copy to prevent mutation of snapshot
        return node.model_copy(deep=True) if node else None

    def query(self, selector: str) -> list[Node]:
        """
        Query nodes using CSS selector.

        Args:
            selector: CSS selector string

        Returns:
            List of matching nodes (copies)
        """
        matching = []

        for node in self._nodes.values():
            try:
                # Convert node to HTML in-memory
                html_content = node.to_html()

                # Parse the HTML string
                parser = HtmlParser.from_string(html_content)

                # Check if selector matches
                if parser.query(f"article{selector}"):
                    # Return copy to prevent mutation
                    matching.append(node.model_copy(deep=True))
            except Exception:
                # Skip nodes that fail to parse
                continue

        return matching

    def filter(self, predicate: Callable[[Node], bool]) -> list[Node]:
        """
        Filter nodes using a predicate function.

        Args:
            predicate: Function that takes Node and returns bool

        Returns:
            List of matching nodes (copies)
        """
        return [
            node.model_copy(deep=True)
            for node in self._nodes.values()
            if predicate(node)
        ]

    def __len__(self) -> int:
        """Get number of nodes in snapshot."""
        return len(self._nodes)

    def __contains__(self, node_id: str) -> bool:
        """Check if node exists in snapshot."""
        return node_id in self._nodes

    def __iter__(self) -> Iterator[Node]:
        """Iterate over nodes in snapshot (returns copies)."""
        return iter(node.model_copy(deep=True) for node in self._nodes.values())

    @property
    def nodes(self) -> dict[str, Node]:
        """Get all nodes as a dict (returns copies)."""
        return {
            node_id: node.model_copy(deep=True) for node_id, node in self._nodes.items()
        }
