"""
Mixin classes for HtmlGraph functionality.

Extracts transaction support, memory-efficient loading, and metrics/stats
from core.py to keep the main class focused on CRUD operations.

These mixins are combined into HtmlGraph via multiple inheritance.
Attribute access (self._nodes, self._metrics, etc.) is resolved at
runtime through the concrete HtmlGraph class.
"""

from __future__ import annotations

from collections.abc import Callable, Iterator
from contextlib import contextmanager
from pathlib import Path
from typing import Any, cast

from htmlgraph.models import Node

from .snapshot import GraphSnapshot


class TransactionMixin:
    """Transaction and snapshot support for HtmlGraph."""

    # Declare attributes that exist on the concrete class for type checkers.
    # These are set by HtmlGraph.__init__ and used here.
    _nodes: dict[str, Node]
    _file_hashes: dict[str, str]
    directory: Path

    def snapshot(self) -> GraphSnapshot:
        """
        Create an immutable snapshot of the current graph state.

        The snapshot is a frozen copy that won't be affected by subsequent
        changes to the graph. Useful for:
        - Concurrent read operations
        - Comparing graph state before/after changes
        - Safe multi-agent scenarios

        Returns:
            GraphSnapshot: Immutable view of current graph state

        Example:
            # Agent 1 takes snapshot
            snapshot = graph.snapshot()

            # Agent 2 modifies graph
            graph.update(node)

            # Agent 1's snapshot is unchanged
            old_node = snapshot.get("feature-001")
        """
        self._ensure_loaded()  # type: ignore[attr-defined]
        return GraphSnapshot(self._nodes, self.directory)

    @contextmanager
    def transaction(self) -> Iterator[Any]:
        """
        Context manager for atomic multi-operation transactions.

        Operations performed within the transaction are batched and applied
        atomically. If any exception occurs, no changes are persisted.

        Yields:
            TransactionContext: Context for collecting operations

        Raises:
            Exception: Any exception from operations causes rollback

        Example:
            # All-or-nothing batch update
            with graph.transaction() as tx:
                tx.add(node1)
                tx.update(node2)
                tx.delete("feature-003")
            # All changes persisted atomically

            # Failed transaction (rollback)
            try:
                with graph.transaction() as tx:
                    tx.add(node1)
                    tx.update(invalid_node)  # Raises error
            except Exception:
                pass  # No changes persisted
        """
        # Create snapshot before transaction
        snapshot_nodes = {
            node_id: node.model_copy(deep=True) for node_id, node in self._nodes.items()
        }
        snapshot_file_hashes = self._file_hashes.copy()

        # Use 'graph' to reference self with full type for inner class
        graph = self

        # Transaction context for collecting operations
        class TransactionContext:
            def __init__(self) -> None:
                self._operations: list[Callable[[], Any]] = []

            def add(self, node: Node, overwrite: bool = False) -> TransactionContext:
                """Queue an add operation."""
                self._operations.append(
                    lambda: graph.add(node, overwrite=overwrite)  # type: ignore[attr-defined]
                )
                return self

            def update(self, node: Node) -> TransactionContext:
                """Queue an update operation."""
                self._operations.append(lambda: graph.update(node))  # type: ignore[attr-defined]
                return self

            def delete(self, node_id: str) -> TransactionContext:
                """Queue a delete operation."""
                self._operations.append(lambda: graph.delete(node_id))  # type: ignore[attr-defined]
                return self

            def remove(self, node_id: str) -> TransactionContext:
                """Queue a remove operation (alias for delete)."""
                return self.delete(node_id)

            def _commit(self) -> None:
                """Execute all queued operations."""
                for operation in self._operations:
                    operation()

        tx = TransactionContext()

        try:
            yield tx
            # Commit all operations if no exceptions
            tx._commit()
        except Exception:
            # Rollback: restore snapshot state
            self._nodes = snapshot_nodes
            self._file_hashes = snapshot_file_hashes
            self._invalidate_cache()  # type: ignore[attr-defined]

            # Rebuild indexes from restored state
            self._edge_index.rebuild(self._nodes)  # type: ignore[attr-defined]
            self._attr_index.rebuild(self._nodes)  # type: ignore[attr-defined]

            # Re-raise exception
            raise


class ChunkedLoadingMixin:
    """Memory-efficient loading for large graphs (10K+ nodes)."""

    def load_chunked(self, chunk_size: int = 100) -> Iterator[list[Node]]:
        """
        Yield nodes in chunks for memory-efficient processing.

        Loads nodes in batches without loading the entire graph into memory.
        Useful for large graphs (10K+ nodes).

        Args:
            chunk_size: Number of nodes per chunk (default: 100)

        Yields:
            List of nodes (up to chunk_size per batch)

        Example:
            >>> graph = HtmlGraph("features/")
            >>> for chunk in graph.load_chunked(chunk_size=50):
            ...     # Process 50 nodes at a time
            ...     for node in chunk:
            ...         print(node.title)
        """
        files = self._get_node_files()  # type: ignore[attr-defined]

        # Yield nodes in chunks
        for i in range(0, len(files), chunk_size):
            chunk = []
            for filepath in files[i : i + chunk_size]:
                try:
                    node_id = self._filepath_to_node_id(filepath)  # type: ignore[attr-defined]
                    node = self._converter.load(node_id)  # type: ignore[attr-defined]
                    if node:
                        chunk.append(node)
                except Exception:
                    # Skip files that fail to parse
                    continue
            if chunk:
                yield chunk

    def iter_nodes(self) -> Iterator[Node]:
        """
        Iterate over all nodes without loading all into memory.

        Memory-efficient iteration for large graphs. Loads nodes one at a time
        instead of loading the entire graph.

        Yields:
            Node: Individual nodes from the graph

        Example:
            >>> graph = HtmlGraph("features/")
            >>> for node in graph.iter_nodes():
            ...     if node.status == "blocked":
            ...         print(f"Blocked: {node.title}")
        """
        for filepath in self._get_node_files():  # type: ignore[attr-defined]
            try:
                node_id = self._filepath_to_node_id(filepath)  # type: ignore[attr-defined]
                node = self._converter.load(node_id)  # type: ignore[attr-defined]
                if node:
                    yield node
            except Exception:
                # Skip files that fail to parse
                continue

    @property
    def node_count(self) -> int:
        """
        Count nodes without loading them.

        Efficient count by globbing files without parsing HTML.

        Returns:
            Number of nodes in the graph

        Example:
            >>> graph = HtmlGraph("features/")
            >>> print(f"Graph has {graph.node_count} nodes")
            Graph has 42 nodes
        """
        return len(self._get_node_files())  # type: ignore[attr-defined]


class MetricsMixin:
    """Performance metrics and statistics for HtmlGraph."""

    _metrics: dict[str, Any]
    _query_cache: dict[str, list[Node]]
    _cache_enabled: bool
    _compiled_queries: dict[str, Any]
    _nodes: dict[str, Node]

    @property
    def cache_stats(self) -> dict:
        """Get cache statistics."""
        return {
            "cached_queries": len(self._query_cache),
            "cache_enabled": self._cache_enabled,
        }

    @property
    def metrics(self) -> dict:
        """
        Get performance metrics.

        Returns:
            Dict with query counts, cache stats, timing info

        Example:
            >>> graph.metrics
            {
                'query_count': 42,
                'cache_hits': 38,
                'cache_hit_rate': '90.5%',
                'avg_query_time_ms': 12.3,
                ...
            }
        """
        m = self._metrics.copy()

        # Calculate derived metrics
        query_count = cast(int, m["query_count"])
        if query_count > 0:
            cache_hits = cast(int, m["cache_hits"])
            total_query_time_ms = cast(float, m["total_query_time_ms"])
            m["cache_hit_rate"] = f"{cache_hits / query_count * 100:.1f}%"
            m["avg_query_time_ms"] = total_query_time_ms / query_count
        else:
            m["cache_hit_rate"] = "N/A"
            m["avg_query_time_ms"] = 0.0

        # Add current state
        m["nodes_loaded"] = len(self._nodes)
        m["cached_queries"] = len(self._query_cache)
        m["compiled_queries_cached"] = len(self._compiled_queries)

        # Calculate compilation hit rate
        compiled_queries = cast(int, m["compiled_queries"])
        compiled_query_hits = cast(int, m["compiled_query_hits"])
        total_compilations = compiled_queries + compiled_query_hits
        if total_compilations > 0:
            m["compilation_hit_rate"] = (
                f"{compiled_query_hits / total_compilations * 100:.1f}%"
            )
        else:
            m["compilation_hit_rate"] = "N/A"

        return m

    def reset_metrics(self) -> None:
        """Reset all performance metrics to zero."""
        for key in self._metrics:
            if isinstance(self._metrics[key], (int, float)):
                self._metrics[key] = 0 if isinstance(self._metrics[key], int) else 0.0
            else:
                self._metrics[key] = ""
