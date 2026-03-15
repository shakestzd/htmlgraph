"""
Core HtmlGraph class with CRUD operations.

This is the main entry point for the graph database.
Snapshot, transaction, chunked loading, and metrics logic
are in separate modules (snapshot.py, mixins.py).
"""

from __future__ import annotations

import hashlib
import os
import time
from collections.abc import Callable, Iterator
from pathlib import Path
from typing import Any

from htmlgraph.attribute_index import AttributeIndex
from htmlgraph.converter import NodeConverter
from htmlgraph.edge_index import EdgeIndex
from htmlgraph.exceptions import NodeNotFoundError
from htmlgraph.models import Node
from htmlgraph.query_builder import QueryBuilder

from . import queries
from .mixins import ChunkedLoadingMixin, MetricsMixin, TransactionMixin

# Re-export GraphSnapshot so existing `from .core import GraphSnapshot` works
from .snapshot import GraphSnapshot  # noqa: F401


class HtmlGraph(TransactionMixin, ChunkedLoadingMixin, MetricsMixin):
    """
    File-based graph database using HTML files.

    Each HTML file is a node, hyperlinks are edges.
    Queries use CSS selectors.

    Example:
        graph = HtmlGraph("features/")
        graph.add(node)
        blocked = graph.query("[data-status='blocked']")
        path = graph.shortest_path("feature-001", "feature-010")
    """

    def __init__(
        self,
        directory: Path | str,
        stylesheet_path: str = "../styles.css",
        auto_load: bool = False,
        pattern: str | list[str] = "*.html",
    ):
        """
        Initialize graph from a directory.

        Args:
            directory: Directory containing HTML node files
            stylesheet_path: Default stylesheet path for new files
            auto_load: Whether to load all nodes on init (default: False for lazy loading)
            pattern: Glob pattern(s) for node files. Can be a single pattern or list.
                     Examples: "*.html", ["*.html", "*/index.html"]
        """
        self.directory = Path(directory)
        self.directory.mkdir(parents=True, exist_ok=True)
        self.stylesheet_path = stylesheet_path
        self.pattern = pattern

        self._nodes: dict[str, Node] = {}
        self._converter = NodeConverter(directory, stylesheet_path)
        self._edge_index = EdgeIndex()
        self._attr_index = AttributeIndex()
        self._query_cache: dict[str, list[Node]] = {}
        self._adjacency_cache: dict[str, dict[str, list[str]]] | None = None
        self._cache_enabled: bool = True
        self._explicitly_loaded: bool = False
        self._file_hashes: dict[str, str] = {}  # Track file content hashes

        # Query compilation cache (LRU cache with max 100 compiled queries)
        self._compiled_queries: dict[str, queries.CompiledQuery] = {}
        self._compiled_query_max_size: int = 100

        # Performance metrics
        self._metrics = {
            "query_count": 0,
            "cache_hits": 0,
            "cache_misses": 0,
            "reload_count": 0,
            "single_reload_count": 0,
            "total_query_time_ms": 0.0,
            "slowest_query_ms": 0.0,
            "slowest_query_selector": "",
            "last_reload_time_ms": 0.0,
            "compiled_queries": 0,
            "compiled_query_hits": 0,
            "auto_compiled_count": 0,
        }

        # Check for env override (backwards compatibility)
        if os.environ.get("HTMLGRAPH_EAGER_LOAD") == "1":
            auto_load = True

        if auto_load:
            self.reload()

    # =========================================================================
    # Internal Helpers
    # =========================================================================

    def _invalidate_cache(self) -> None:
        """Clear query, adjacency, attribute, and compiled query caches. Called when graph is modified."""
        self._query_cache.clear()
        self._compiled_queries.clear()
        self._adjacency_cache = None
        self._attr_index.clear()

    def _compute_file_hash(self, filepath: Path) -> str:
        """Compute MD5 hash of file content."""
        try:
            content = filepath.read_bytes()
            return hashlib.md5(content).hexdigest()
        except Exception:
            return ""

    def has_file_changed(self, filepath: Path | str) -> bool:
        """
        Check if a file has changed since it was last loaded.

        Args:
            filepath: Path to file to check

        Returns:
            True if file changed or not yet loaded, False if unchanged
        """
        filepath = Path(filepath)
        if not filepath.exists():
            return True

        filepath_str = str(filepath)
        current_hash = self._compute_file_hash(filepath)
        stored_hash = self._file_hashes.get(filepath_str)

        return stored_hash is None or current_hash != stored_hash

    # =========================================================================
    # Loading
    # =========================================================================

    def reload(self) -> int:
        """
        Reload all nodes from disk.

        Returns:
            Number of nodes loaded
        """
        start = time.perf_counter()
        self._cache_enabled = False  # Disable during reload
        try:
            self._nodes.clear()
            self._file_hashes.clear()

            # Load all nodes and compute file hashes
            for node in self._converter.load_all(self.pattern):
                self._nodes[node.id] = node

                # Find and hash the node file
                filepath = self._find_node_file(node.id)
                if filepath:
                    file_hash = self._compute_file_hash(filepath)
                    self._file_hashes[str(filepath)] = file_hash

            # Rebuild edge index for O(1) reverse lookups
            # Rebuild attribute index for O(1) attribute lookups
            self._attr_index.rebuild(self._nodes)
            self._edge_index.rebuild(self._nodes)

            self._explicitly_loaded = True

            # Track metrics
            elapsed_ms = (time.perf_counter() - start) * 1000
            reload_count: int = int(self._metrics.get("reload_count", 0))  # type: ignore[call-overload]
            self._metrics["reload_count"] = reload_count + 1
            self._metrics["last_reload_time_ms"] = elapsed_ms

            return len(self._nodes)
        finally:
            self._cache_enabled = True
            self._invalidate_cache()

    def _ensure_loaded(self) -> None:
        """Ensure nodes are loaded. Called lazily on first access."""
        if not self._explicitly_loaded and not self._nodes:
            self.reload()

    def _get_node_files(self) -> list[Path]:
        """Get all node files matching the configured pattern(s)."""
        files: list[Path] = []
        patterns = [self.pattern] if isinstance(self.pattern, str) else self.pattern
        for pattern in patterns:
            files.extend(self.directory.glob(pattern))
        return files

    def _filepath_to_node_id(self, filepath: Path) -> str:
        """Extract node ID from a filepath."""
        if filepath.name == "index.html":
            return filepath.parent.name
        else:
            return filepath.stem

    def _find_node_file(self, node_id: str) -> Path | None:
        """
        Find the file path for a node by ID.

        Checks common naming patterns for node files.
        """
        # Try direct match patterns
        patterns = [
            f"{node_id}.html",
            f"{node_id}/index.html",
        ]

        for pattern in patterns:
            filepath = self.directory / pattern
            if filepath.exists():
                return filepath

        # Fall back to scanning (slower but thorough)
        for filepath in self.directory.glob("*.html"):
            try:
                content = filepath.read_text()
                if f'id="{node_id}"' in content or f"id='{node_id}'" in content:
                    return filepath
            except Exception:
                continue

        return None

    # =========================================================================
    # Collection Protocol (__len__, __contains__, __iter__)
    # =========================================================================

    @property
    def nodes(self) -> dict[str, Node]:
        """Get all nodes (read-only view)."""
        return self._nodes.copy()

    def __len__(self) -> int:
        """Get the number of nodes in the graph."""
        return len(self._nodes)

    def __contains__(self, node_id: str) -> bool:
        """Check if a node exists in the graph."""
        return node_id in self._nodes

    def __iter__(self) -> Iterator[Node]:
        """Iterate over all nodes in the graph."""
        self._ensure_loaded()
        return iter(self._nodes.values())

    # =========================================================================
    # CRUD Operations
    # =========================================================================

    def add(self, node: Node, overwrite: bool = False) -> Path:
        """
        Add a node to the graph (creates HTML file).

        Args:
            node: Node to add
            overwrite: Whether to overwrite existing node

        Returns:
            Path to created HTML file

        Raises:
            ValueError: If node exists and overwrite=False
        """
        if node.id in self._nodes and not overwrite:
            raise ValueError(f"Node already exists: {node.id}")

        # If overwriting, remove old node from indexes first
        if overwrite and node.id in self._nodes:
            old_node = self._nodes[node.id]
            self._edge_index.remove_node(node.id)
            self._attr_index.remove_node(node.id, old_node)

        filepath = self._converter.save(node)
        self._nodes[node.id] = node

        # Update file hash
        file_hash = self._compute_file_hash(filepath)
        self._file_hashes[str(filepath)] = file_hash

        # Add new edges to index
        for relationship, edges in node.edges.items():
            for edge in edges:
                self._edge_index.add(node.id, edge.target_id, edge.relationship)

        # Add node to attribute index
        self._attr_index.add_node(node.id, node)

        self._invalidate_cache()
        return filepath

    def save_node(self, node: Node) -> Path:
        """Save a node to the graph (add or update)."""
        return self.add(node, overwrite=True)

    def update(self, node: Node) -> Path:
        """
        Update an existing node.

        Args:
            node: Node with updated data

        Returns:
            Path to updated HTML file

        Raises:
            NodeNotFoundError: If node doesn't exist
        """
        if node.id not in self._nodes:
            raise NodeNotFoundError(node.type, node.id)

        # Get current outgoing edges from the edge index (source of truth)
        old_outgoing = self._edge_index.get_outgoing(node.id)

        # Remove all old OUTGOING edges (where this node is source)
        for edge_ref in old_outgoing:
            self._edge_index.remove(
                edge_ref.source_id, edge_ref.target_id, edge_ref.relationship
            )

        # Add new OUTGOING edges (where this node is source)
        for relationship, edges in node.edges.items():
            for edge in edges:
                self._edge_index.add(node.id, edge.target_id, edge.relationship)

        # Update attribute index
        old_node = self._nodes[node.id]
        self._attr_index.update_node(node.id, old_node, node)

        filepath = self._converter.save(node)
        self._nodes[node.id] = node

        # Update file hash
        file_hash = self._compute_file_hash(filepath)
        self._file_hashes[str(filepath)] = file_hash

        self._invalidate_cache()
        return filepath

    def get(self, node_id: str) -> Node | None:
        """Get a node by ID."""
        self._ensure_loaded()
        return self._nodes.get(node_id)

    def get_or_load(self, node_id: str) -> Node | None:
        """Get node from cache or load from disk."""
        if node_id in self._nodes:
            return self._nodes[node_id]

        node = self._converter.load(node_id)
        if node:
            self._nodes[node_id] = node
            reload_count: int = int(self._metrics.get("single_reload_count", 0))  # type: ignore[call-overload]
            self._metrics["single_reload_count"] = reload_count + 1
        return node

    def reload_node(self, node_id: str) -> Node | None:
        """
        Reload a single node from disk without full graph reload.

        Much faster than full reload() when only one node changed.
        Uses file hash to skip reload if content hasn't changed.

        Args:
            node_id: ID of the node to reload

        Returns:
            Updated node if found and loaded, None if not found
        """
        filepath = self._find_node_file(node_id)
        if not filepath:
            return None

        # Check if file has actually changed
        if not self.has_file_changed(filepath):
            return self._nodes.get(node_id)

        try:
            # Remove old node's edges from index if exists
            if node_id in self._nodes:
                old_node = self._nodes[node_id]
                self._edge_index.remove_node_edges(node_id, old_node)

            # Load updated node from disk
            updated_node = self._converter.load(node_id)
            if not updated_node:
                return None

            # Update cache
            self._nodes[node_id] = updated_node

            # Update file hash
            file_hash = self._compute_file_hash(filepath)
            self._file_hashes[str(filepath)] = file_hash

            # Add new edges to index
            self._edge_index.add_node_edges(node_id, updated_node)

            # Invalidate query cache
            self._invalidate_cache()

            # Track metric
            reload_count: int = int(self._metrics.get("single_reload_count", 0))  # type: ignore[call-overload]
            self._metrics["single_reload_count"] = reload_count + 1

            return updated_node
        except Exception:
            return None

    def remove(self, node_id: str) -> bool:
        """Remove a node from the graph."""
        if node_id in self._nodes:
            filepath = self._find_node_file(node_id)
            if filepath:
                self._file_hashes.pop(str(filepath), None)

            old_node = self._nodes[node_id]
            self._edge_index.remove_node(node_id)
            self._attr_index.remove_node(node_id, old_node)
            del self._nodes[node_id]
            result = self._converter.delete(node_id)
            self._invalidate_cache()
            return result
        return False

    def delete(self, node_id: str) -> bool:
        """Delete a node from the graph (alias for remove)."""
        return self.remove(node_id)

    def batch_delete(self, node_ids: list[str]) -> int:
        """Delete multiple nodes in batch."""
        count = 0
        for node_id in node_ids:
            if self.delete(node_id):
                count += 1
        return count

    # =========================================================================
    # Query Methods (delegates to queries module)
    # =========================================================================

    def query(self, selector: str) -> list[Node]:
        """Query nodes using CSS selector."""
        return queries.query(self, selector)

    def query_one(self, selector: str) -> Node | None:
        """Query for single node matching selector."""
        return queries.query_one(self, selector)

    def compile_query(self, selector: str) -> queries.CompiledQuery:
        """Pre-compile a CSS selector for reuse."""
        return queries.compile_query(self, selector)

    def query_compiled(self, compiled: queries.CompiledQuery) -> list[Node]:
        """Execute a pre-compiled query."""
        return queries.query_compiled(self, compiled)

    def filter(self, predicate: Callable[[Node], bool]) -> list[Node]:
        """Filter nodes using a Python predicate function."""
        return queries.filter_nodes(self, predicate)

    def by_status(self, status: str) -> list[Node]:
        """Get all nodes with given status (O(1) lookup via attribute index)."""
        return queries.by_status(self, status)

    def by_type(self, node_type: str) -> list[Node]:
        """Get all nodes with given type (O(1) lookup via attribute index)."""
        return queries.by_type(self, node_type)

    def by_priority(self, priority: str) -> list[Node]:
        """Get all nodes with given priority (O(1) lookup via attribute index)."""
        return queries.by_priority(self, priority)

    def get_by_status(self, status: str) -> list[Node]:
        """Get all nodes with given status (alias for by_status)."""
        return queries.get_by_status(self, status)

    def get_by_type(self, node_type: str) -> list[Node]:
        """Get all nodes with given type (alias for by_type)."""
        return queries.get_by_type(self, node_type)

    def get_by_priority(self, priority: str) -> list[Node]:
        """Get all nodes with given priority (alias for by_priority)."""
        return queries.get_by_priority(self, priority)

    def query_builder(self) -> QueryBuilder:
        """Create a fluent query builder for complex queries."""
        return QueryBuilder(_graph=self)

    def find(self, type: str | None = None, **kwargs: Any) -> Node | None:
        """Find the first node matching the given criteria."""
        return queries.find(self, type=type, **kwargs)

    def find_all(
        self, type: str | None = None, limit: int | None = None, **kwargs: Any
    ) -> list[Node]:
        """Find all nodes matching the given criteria."""
        return queries.find_all(self, type=type, limit=limit, **kwargs)

    def find_related(
        self, node_id: str, relationship: str | None = None, direction: str = "outgoing"
    ) -> list[Node]:
        """Find nodes related to a given node."""
        return queries.find_related(self, node_id, relationship, direction)

    # =========================================================================
    # Edge Index Operations
    # =========================================================================

    def get_incoming_edges(self, node_id: str, relationship: str | None = None) -> list:
        """Get all edges pointing TO a node (O(1) lookup)."""
        return queries.get_incoming_edges(self, node_id, relationship)

    def get_outgoing_edges(self, node_id: str, relationship: str | None = None) -> list:
        """Get all edges pointing FROM a node (O(1) lookup)."""
        return queries.get_outgoing_edges(self, node_id, relationship)

    def get_neighbors(
        self, node_id: str, relationship: str | None = None, direction: str = "both"
    ) -> set[str]:
        """Get all neighboring node IDs connected to a node (O(1) lookup)."""
        return queries.get_neighbors(self, node_id, relationship, direction)

    @property
    def edge_index(self) -> EdgeIndex:
        """Access the edge index for advanced queries."""
        return self._edge_index

    @property
    def attribute_index(self) -> AttributeIndex:
        """Access the attribute index for advanced queries."""
        self._ensure_loaded()
        self._attr_index.ensure_built(self._nodes)
        return self._attr_index

    # =========================================================================
    # Graph Algorithms (delegates to algorithms module)
    # =========================================================================

    def shortest_path(
        self, from_id: str, to_id: str, relationship: str | None = None
    ) -> list[str] | None:
        """Find shortest path between two nodes using BFS."""
        from . import algorithms

        return algorithms.shortest_path(self, from_id, to_id, relationship)

    def transitive_deps(
        self, node_id: str, relationship: str = "blocked_by"
    ) -> set[str]:
        """Get all transitive dependencies of a node."""
        from . import algorithms

        return algorithms.transitive_deps(self, node_id, relationship)

    def get_transitive_dependencies(
        self, node_id: str, rel_type: str = "blocked_by"
    ) -> set[str]:
        """
        Get all transitive dependencies of a node, filtered by relationship type.

        When rel_type is specified, only traverses edges of that type.
        This is the typed-relationship-aware version of transitive_deps().

        Args:
            node_id: Starting node ID
            rel_type: Relationship type to follow (default: blocked_by)

        Returns:
            Set of all dependency node IDs reachable via edges of rel_type

        Example:
            # Only traverse blocked_by edges
            deps = graph.get_transitive_dependencies("feat-001", rel_type="blocked_by")
            # Only traverse depends_on edges
            deps = graph.get_transitive_dependencies("feat-001", rel_type="depends_on")
        """
        from . import algorithms

        return algorithms.get_dependencies(self, node_id, rel_type=rel_type)

    def dependents(self, node_id: str, relationship: str = "blocked_by") -> set[str]:
        """Find all nodes that depend on this node (O(1) lookup)."""
        from . import algorithms

        return algorithms.dependents(self, node_id, relationship)

    def find_bottlenecks(
        self, relationship: str = "blocked_by", top_n: int = 5
    ) -> list[tuple[str, int]]:
        """Find nodes that block the most other nodes."""
        from . import algorithms

        return algorithms.find_bottlenecks(self, relationship, top_n)

    def find_cycles(self, relationship: str = "blocked_by") -> list[list[str]]:
        """Detect cycles in the graph."""
        from . import algorithms

        return algorithms.find_cycles(self, relationship)

    def topological_sort(self, relationship: str = "blocked_by") -> list[str] | None:
        """Return nodes in topological order (dependencies first)."""
        from . import algorithms

        return algorithms.topological_sort(self, relationship)

    def ancestors(
        self,
        node_id: str,
        relationship: str = "blocked_by",
        max_depth: int | None = None,
    ) -> list[str]:
        """Get all ancestor nodes (nodes that this node depends on)."""
        from . import algorithms

        return algorithms.ancestors(self, node_id, relationship, max_depth)

    def descendants(
        self,
        node_id: str,
        relationship: str = "blocked_by",
        max_depth: int | None = None,
    ) -> list[str]:
        """Get all descendant nodes (nodes that depend on this node)."""
        from . import algorithms

        return algorithms.descendants(self, node_id, relationship, max_depth)

    def subgraph(
        self, node_ids: list[str] | set[str], include_edges: bool = True
    ) -> HtmlGraph:
        """Extract a subgraph containing only the specified nodes."""
        from . import algorithms

        return algorithms.subgraph(self, node_ids, include_edges)

    def connected_component(
        self, node_id: str, relationship: str | None = None
    ) -> set[str]:
        """Get all nodes in the same connected component as the given node."""
        from . import algorithms

        return algorithms.connected_component(self, node_id, relationship)

    def all_paths(
        self,
        from_id: str,
        to_id: str,
        relationship: str | None = None,
        max_length: int | None = None,
        max_paths: int = 100,
        timeout_seconds: float = 5.0,
    ) -> list[list[str]]:
        """Find all paths between two nodes."""
        from . import algorithms

        return algorithms.all_paths(
            self, from_id, to_id, relationship, max_length, max_paths, timeout_seconds
        )

    # =========================================================================
    # Statistics & Export
    # =========================================================================

    def stats(self) -> dict[str, Any]:
        """Get graph statistics."""
        from . import algorithms

        return algorithms.stats(self)

    def to_context(self, max_nodes: int = 20) -> str:
        """Generate lightweight context for AI agents."""
        return queries.to_context(self, max_nodes)

    def to_json(self) -> list[dict[str, Any]]:
        """Export all nodes as JSON-serializable list."""
        return queries.to_json(self)

    def to_mermaid(self, relationship: str | None = None) -> str:
        """Export graph as Mermaid diagram."""
        from . import algorithms

        return algorithms.to_mermaid(self, relationship)

    # =========================================================================
    # Internal Helpers for Algorithms
    # =========================================================================

    def _get_adjacency_cache(self) -> dict[str, dict[str, list[str]]]:
        """
        Get or build the persistent adjacency cache.

        Returns:
            Dict mapping node_id to dict with "outgoing" and "incoming" neighbor lists
        """
        if self._adjacency_cache is None:
            self._adjacency_cache = {}
            for node_id in self._nodes:
                outgoing = self._edge_index.get_neighbors(
                    node_id, relationship=None, direction="outgoing"
                )
                incoming = self._edge_index.get_neighbors(
                    node_id, relationship=None, direction="incoming"
                )
                self._adjacency_cache[node_id] = {
                    "outgoing": list(outgoing),
                    "incoming": list(incoming),
                }
        return self._adjacency_cache
