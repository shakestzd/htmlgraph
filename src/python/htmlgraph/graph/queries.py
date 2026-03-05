"""
Query and filter operations for HtmlGraph.

Provides:
- CSS selector queries with caching
- Query compilation and reuse
- Predicate-based filtering
- Attribute-based lookups (status, type, priority)
- BeautifulSoup-style find API
"""

from __future__ import annotations

import time
from collections.abc import Callable
from dataclasses import dataclass, field
from datetime import datetime
from typing import TYPE_CHECKING, Any, cast

from htmlgraph.parser import HtmlParser

if TYPE_CHECKING:
    from htmlgraph.edge_index import EdgeRef
    from htmlgraph.models import Node

    from .core import HtmlGraph


@dataclass
class CompiledQuery:
    """
    Pre-compiled CSS selector query for efficient reuse.

    While justhtml doesn't support native selector pre-compilation,
    this class provides:
    - Cached selector string to avoid string manipulation overhead
    - Reusable query execution with metrics tracking
    - Integration with query cache for performance

    Example:
        >>> graph = HtmlGraph("features/")
        >>> compiled = graph.compile_query("[data-status='blocked']")
        >>> results = graph.query_compiled(compiled)  # Fast on reuse
        >>> results2 = graph.query_compiled(compiled)  # Uses cache
    """

    selector: str
    _compiled_at: datetime = field(default_factory=datetime.now)
    _use_count: int = field(default=0, init=False)

    def matches(self, node: Node) -> bool:
        """
        Check if a node matches this compiled query.

        Args:
            node: Node to check

        Returns:
            True if node matches selector
        """
        try:
            # Convert node to HTML in-memory
            html_content = node.to_html()

            # Parse the HTML string
            parser = HtmlParser.from_string(html_content)

            # Check if selector matches
            return bool(parser.query(f"article{self.selector}"))
        except Exception:
            return False

    def execute(self, nodes: dict[str, Node]) -> list[Node]:
        """
        Execute this compiled query on a set of nodes.

        Args:
            nodes: Dict of nodes to query

        Returns:
            List of matching nodes
        """
        self._use_count += 1
        return [node for node in nodes.values() if self.matches(node)]


def query(graph: HtmlGraph, selector: str) -> list[Node]:
    """
    Query nodes using CSS selector with caching and metrics.

    Selector is applied to article element of each node.
    Uses cached nodes instead of re-parsing from disk for better performance.

    Args:
        graph: HtmlGraph instance
        selector: CSS selector string

    Returns:
        List of matching nodes

    Example:
        graph.query("[data-status='blocked']")
        graph.query("[data-priority='high'][data-type='feature']")
    """
    graph._ensure_loaded()
    query_count: int = int(graph._metrics.get("query_count", 0))  # type: ignore[call-overload]
    graph._metrics["query_count"] = query_count + 1

    # Check cache first
    if graph._cache_enabled and selector in graph._query_cache:
        cache_hits: int = int(graph._metrics.get("cache_hits", 0))  # type: ignore[call-overload]
        graph._metrics["cache_hits"] = cache_hits + 1
        return graph._query_cache[selector].copy()  # Return copy to prevent mutation

    cache_misses: int = int(graph._metrics.get("cache_misses", 0))  # type: ignore[call-overload]
    graph._metrics["cache_misses"] = cache_misses + 1

    # Time the query
    start = time.perf_counter()

    # Perform query using cached nodes instead of disk I/O
    matching = []

    for node in graph._nodes.values():
        try:
            # Convert node to HTML in-memory
            html_content = node.to_html()

            # Parse the HTML string
            parser = HtmlParser.from_string(html_content)

            # Check if selector matches
            if parser.query(f"article{selector}"):
                matching.append(node)
        except Exception:
            # Skip nodes that fail to parse
            continue

    # Track timing
    elapsed_ms = (time.perf_counter() - start) * 1000
    total_time: float = cast(float, graph._metrics.get("total_query_time_ms", 0.0))
    graph._metrics["total_query_time_ms"] = total_time + elapsed_ms

    slowest: float = cast(float, graph._metrics.get("slowest_query_ms", 0.0))
    if elapsed_ms > slowest:
        graph._metrics["slowest_query_ms"] = elapsed_ms
        graph._metrics["slowest_query_selector"] = selector

    # Cache result
    if graph._cache_enabled:
        graph._query_cache[selector] = matching.copy()

    return matching


def query_one(graph: HtmlGraph, selector: str) -> Node | None:
    """Query for single node matching selector."""
    results = query(graph, selector)
    return results[0] if results else None


def compile_query(graph: HtmlGraph, selector: str) -> CompiledQuery:
    """
    Pre-compile a CSS selector for reuse.

    Creates a CompiledQuery object that can be reused multiple times
    with query_compiled() for better performance when the same selector
    is used frequently.

    Args:
        graph: HtmlGraph instance
        selector: CSS selector string to compile

    Returns:
        CompiledQuery object that can be reused

    Example:
        >>> graph = HtmlGraph("features/")
        >>> compiled = graph.compile_query("[data-status='blocked']")
        >>> results1 = graph.query_compiled(compiled)
        >>> results2 = graph.query_compiled(compiled)  # Reuses compilation
    """
    # Check if already compiled
    if selector in graph._compiled_queries:
        hits: int = int(graph._metrics.get("compiled_query_hits", 0))  # type: ignore[call-overload]
        graph._metrics["compiled_query_hits"] = hits + 1
        return graph._compiled_queries[selector]

    # Create new compiled query
    compiled = CompiledQuery(selector=selector)
    compiled_count: int = int(graph._metrics.get("compiled_queries", 0))  # type: ignore[call-overload]
    graph._metrics["compiled_queries"] = compiled_count + 1

    # Add to cache (with LRU eviction if needed)
    if len(graph._compiled_queries) >= graph._compiled_query_max_size:
        # Evict least recently used (first item in dict)
        first_key = next(iter(graph._compiled_queries))
        del graph._compiled_queries[first_key]

    graph._compiled_queries[selector] = compiled
    return compiled


def query_compiled(graph: HtmlGraph, compiled: CompiledQuery) -> list[Node]:
    """
    Execute a pre-compiled query.

    Uses the regular query cache if available, otherwise executes
    the compiled query and caches the result.

    Args:
        graph: HtmlGraph instance
        compiled: CompiledQuery object from compile_query()

    Returns:
        List of matching nodes

    Example:
        >>> compiled = graph.compile_query("[data-priority='high']")
        >>> high_priority = graph.query_compiled(compiled)
    """
    graph._ensure_loaded()
    selector = compiled.selector
    query_count: int = int(graph._metrics.get("query_count", 0))  # type: ignore[call-overload]
    graph._metrics["query_count"] = query_count + 1

    # Check cache first (same cache as regular query())
    if graph._cache_enabled and selector in graph._query_cache:
        cache_hits: int = int(graph._metrics.get("cache_hits", 0))  # type: ignore[call-overload]
        graph._metrics["cache_hits"] = cache_hits + 1
        return graph._query_cache[selector].copy()

    cache_misses: int = int(graph._metrics.get("cache_misses", 0))  # type: ignore[call-overload]
    graph._metrics["cache_misses"] = cache_misses + 1

    # Time the query
    start = time.perf_counter()

    # Execute compiled query
    matching = compiled.execute(graph._nodes)

    # Track timing
    elapsed_ms = (time.perf_counter() - start) * 1000
    total_time: float = cast(float, graph._metrics.get("total_query_time_ms", 0.0))
    graph._metrics["total_query_time_ms"] = total_time + elapsed_ms

    slowest: float = cast(float, graph._metrics.get("slowest_query_ms", 0.0))
    if elapsed_ms > slowest:
        graph._metrics["slowest_query_ms"] = elapsed_ms
        graph._metrics["slowest_query_selector"] = selector

    # Cache result
    if graph._cache_enabled:
        graph._query_cache[selector] = matching.copy()

    return matching


def filter_nodes(graph: HtmlGraph, predicate: Callable[[Node], bool]) -> list[Node]:
    """
    Filter nodes using a Python predicate function.

    Args:
        graph: HtmlGraph instance
        predicate: Function that takes Node and returns bool

    Returns:
        List of nodes where predicate returns True

    Example:
        graph.filter(lambda n: n.status == "todo" and n.priority == "high")
    """
    graph._ensure_loaded()
    return [node for node in graph._nodes.values() if predicate(node)]


def by_status(graph: HtmlGraph, status: str) -> list[Node]:
    """
    Get all nodes with given status (O(1) lookup via attribute index).

    Uses the attribute index for efficient lookups instead of
    filtering all nodes.

    Args:
        graph: HtmlGraph instance
        status: Status value to filter by

    Returns:
        List of nodes with the given status
    """
    graph._ensure_loaded()
    graph._attr_index.ensure_built(graph._nodes)
    node_ids = graph._attr_index.get_by_status(status)
    return [graph._nodes[node_id] for node_id in node_ids if node_id in graph._nodes]


def by_type(graph: HtmlGraph, node_type: str) -> list[Node]:
    """
    Get all nodes with given type (O(1) lookup via attribute index).

    Uses the attribute index for efficient lookups instead of
    filtering all nodes.

    Args:
        graph: HtmlGraph instance
        node_type: Node type to filter by

    Returns:
        List of nodes with the given type
    """
    graph._ensure_loaded()
    graph._attr_index.ensure_built(graph._nodes)
    node_ids = graph._attr_index.get_by_type(node_type)
    return [graph._nodes[node_id] for node_id in node_ids if node_id in graph._nodes]


def by_priority(graph: HtmlGraph, priority: str) -> list[Node]:
    """
    Get all nodes with given priority (O(1) lookup via attribute index).

    Uses the attribute index for efficient lookups instead of
    filtering all nodes.

    Args:
        graph: HtmlGraph instance
        priority: Priority value to filter by

    Returns:
        List of nodes with the given priority
    """
    graph._ensure_loaded()
    graph._attr_index.ensure_built(graph._nodes)
    node_ids = graph._attr_index.get_by_priority(priority)
    return [graph._nodes[node_id] for node_id in node_ids if node_id in graph._nodes]


def get_by_status(graph: HtmlGraph, status: str) -> list[Node]:
    """
    Get all nodes with given status (O(1) lookup via attribute index).

    Alias for by_status() with explicit name for clarity.

    Args:
        graph: HtmlGraph instance
        status: Status value to filter by

    Returns:
        List of nodes with the given status
    """
    return by_status(graph, status)


def get_by_type(graph: HtmlGraph, node_type: str) -> list[Node]:
    """
    Get all nodes with given type (O(1) lookup via attribute index).

    Alias for by_type() with explicit name for clarity.

    Args:
        graph: HtmlGraph instance
        node_type: Node type to filter by

    Returns:
        List of nodes with the given type
    """
    return by_type(graph, node_type)


def get_by_priority(graph: HtmlGraph, priority: str) -> list[Node]:
    """
    Get all nodes with given priority (O(1) lookup via attribute index).

    Alias for by_priority() with explicit name for clarity.

    Args:
        graph: HtmlGraph instance
        priority: Priority value to filter by

    Returns:
        List of nodes with the given priority
    """
    return by_priority(graph, priority)


def find(graph: HtmlGraph, type: str | None = None, **kwargs: Any) -> Node | None:
    """
    Find the first node matching the given criteria.

    BeautifulSoup-style find method with keyword argument filtering.
    Supports lookup suffixes like __contains, __gt, __in.

    Args:
        graph: HtmlGraph instance
        type: Node type filter (e.g., "feature", "bug")
        **kwargs: Attribute filters with optional lookup suffixes

    Returns:
        First matching Node or None

    Example:
        # Find first blocked feature
        node = graph.find(type="feature", status="blocked")

        # Find with text search
        node = graph.find(title__contains="auth")

        # Find with numeric comparison
        node = graph.find(properties__effort__gt=8)
    """
    from htmlgraph.find_api import FindAPI

    return FindAPI(graph).find(type=type, **kwargs)


def find_all(
    graph: HtmlGraph, type: str | None = None, limit: int | None = None, **kwargs: Any
) -> list[Node]:
    """
    Find all nodes matching the given criteria.

    BeautifulSoup-style find_all method with keyword argument filtering.

    Args:
        graph: HtmlGraph instance
        type: Node type filter
        limit: Maximum number of results
        **kwargs: Attribute filters with optional lookup suffixes

    Returns:
        List of matching Nodes

    Example:
        # Find all high-priority features
        nodes = graph.find_all(type="feature", priority="high")

        # Find with multiple conditions
        nodes = graph.find_all(
            status__in=["todo", "blocked"],
            priority__in=["high", "critical"],
            limit=10
        )

        # Find with nested attribute
        nodes = graph.find_all(properties__completion__lt=50)
    """
    from htmlgraph.find_api import FindAPI

    return FindAPI(graph).find_all(type=type, limit=limit, **kwargs)


def find_related(
    graph: HtmlGraph,
    node_id: str,
    relationship: str | None = None,
    direction: str = "outgoing",
) -> list[Node]:
    """
    Find nodes related to a given node.

    Args:
        graph: HtmlGraph instance
        node_id: Node ID to find relations for
        relationship: Optional filter by relationship type
        direction: "outgoing", "incoming", or "both"

    Returns:
        List of related nodes
    """
    from htmlgraph.find_api import FindAPI

    return FindAPI(graph).find_related(node_id, relationship, direction)


def get_incoming_edges(
    graph: HtmlGraph, node_id: str, relationship: str | None = None
) -> list[EdgeRef]:
    """
    Get all edges pointing TO a node (O(1) lookup).

    Uses the edge index for efficient reverse lookups instead of
    scanning all nodes in the graph.

    Args:
        graph: HtmlGraph instance
        node_id: Node ID to find incoming edges for
        relationship: Optional filter by relationship type

    Returns:
        List of EdgeRefs for incoming edges

    Example:
        # Find all nodes that block feature-001
        blockers = graph.get_incoming_edges("feature-001", "blocked_by")
        for ref in blockers:
            blocker_node = graph.get(ref.source_id)
            print(f"{blocker_node.title} blocks feature-001")
    """
    return graph._edge_index.get_incoming(node_id, relationship)


def get_outgoing_edges(
    graph: HtmlGraph, node_id: str, relationship: str | None = None
) -> list[EdgeRef]:
    """
    Get all edges pointing FROM a node (O(1) lookup).

    Args:
        graph: HtmlGraph instance
        node_id: Node ID to find outgoing edges for
        relationship: Optional filter by relationship type

    Returns:
        List of EdgeRefs for outgoing edges
    """
    return graph._edge_index.get_outgoing(node_id, relationship)


def get_neighbors(
    graph: HtmlGraph,
    node_id: str,
    relationship: str | None = None,
    direction: str = "both",
) -> set[str]:
    """
    Get all neighboring node IDs connected to a node (O(1) lookup).

    Args:
        graph: HtmlGraph instance
        node_id: Node ID to find neighbors for
        relationship: Optional filter by relationship type
        direction: "incoming", "outgoing", or "both"

    Returns:
        Set of neighboring node IDs
    """
    return graph._edge_index.get_neighbors(node_id, relationship, direction)


def to_context(graph: HtmlGraph, max_nodes: int = 20) -> str:
    """
    Generate lightweight context for AI agents.

    Args:
        graph: HtmlGraph instance
        max_nodes: Maximum nodes to include

    Returns:
        Compact string representation of graph state
    """
    from . import algorithms

    lines = ["# Graph Summary"]
    stats_dict = algorithms.stats(graph)
    lines.append(
        f"Total: {stats_dict['total']} nodes | Done: {stats_dict['completion_rate']}%"
    )

    # Status breakdown
    status_parts = [f"{s}: {c}" for s, c in stats_dict["by_status"].items()]
    lines.append(f"Status: {' | '.join(status_parts)}")

    lines.append("")

    # Top priority items
    high_priority = filter_nodes(
        graph, lambda n: n.priority in ("high", "critical") and n.status != "done"
    )[:max_nodes]

    if high_priority:
        lines.append("## High Priority Items")
        for node in high_priority:
            lines.append(f"- {node.id}: {node.title} [{node.status}]")

    return "\n".join(lines)


def to_json(graph: HtmlGraph) -> list[dict[str, Any]]:
    """Export all nodes as JSON-serializable list."""
    from htmlgraph.converter import node_to_dict

    return [node_to_dict(node) for node in graph._nodes.values()]
