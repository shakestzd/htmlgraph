from __future__ import annotations

"""
Feature collection for managing feature work items.

Extends BaseCollection with feature-specific builder support.
"""


from typing import TYPE_CHECKING, Any

if TYPE_CHECKING:
    from htmlgraph.models import Node
    from htmlgraph.sdk import SDK

from htmlgraph.collections.base import BaseCollection


class FeatureCollection(BaseCollection["FeatureCollection"]):
    """
    Collection interface for features with builder support.

    Provides all base collection methods plus a fluent builder
    interface for creating new features.

    Example:
        >>> sdk = SDK(agent="claude")
        >>> feature = sdk.features.create("User Authentication") \\
        ...     .set_priority("high") \\
        ...     .add_steps(["Design schema", "Implement API", "Add tests"]) \\
        ...     .save()
        >>>
        >>> # Query features
        >>> high_priority = sdk.features.where(status="todo", priority="high")
        >>> all_features = sdk.features.all()
    """

    _collection_name = "features"
    _node_type = "feature"

    def __init__(self, sdk: SDK):
        """
        Initialize feature collection.

        Args:
            sdk: Parent SDK instance
        """
        super().__init__(sdk, "features", "feature")
        self._sdk = sdk

        # Set builder class for create() method
        from htmlgraph.builders import FeatureBuilder

        self._builder_class = FeatureBuilder

    def edges(self, node_id: str, rel_type: str | None = None) -> list[dict[str, Any]]:
        """
        Query graph_edges for a given node, optionally filtered by rel_type.

        Queries the SQLite graph_edges table for edges where this node is either
        the source (from_node_id) or the target (to_node_id).

        Args:
            node_id: Node ID to query edges for
            rel_type: Optional relationship type filter (e.g., 'blocked_by', 'depends_on')

        Returns:
            List of edge dictionaries with keys: edge_id, from_node_id, from_node_type,
            to_node_id, to_node_type, relationship_type, weight, created_at, metadata

        Example:
            >>> all_edges = sdk.features.edges("feat-001")
            >>> blocked_by = sdk.features.edges("feat-001", rel_type="blocked_by")
        """
        import sqlite3

        db_path = self._sdk._db.db_path
        conn = sqlite3.connect(str(db_path), timeout=2.0, check_same_thread=False)
        conn.row_factory = sqlite3.Row
        try:
            if rel_type is not None:
                cursor = conn.execute(
                    """
                    SELECT * FROM graph_edges
                    WHERE (from_node_id = ? OR to_node_id = ?)
                      AND relationship_type = ?
                    ORDER BY created_at DESC
                    """,
                    (node_id, node_id, rel_type),
                )
            else:
                cursor = conn.execute(
                    """
                    SELECT * FROM graph_edges
                    WHERE from_node_id = ? OR to_node_id = ?
                    ORDER BY created_at DESC
                    """,
                    (node_id, node_id),
                )
            return [dict(row) for row in cursor.fetchall()]
        finally:
            conn.close()

    def set_primary(self, node_id: str) -> Node | None:
        """
        Set a feature as the primary focus.

        Delegates to SessionManager.

        Args:
            node_id: Node ID to set as primary

        Returns:
            Updated Node
        """
        if hasattr(self._sdk, "session_manager"):
            return self._sdk.session_manager.set_primary_feature(
                feature_id=node_id,
                collection=self._collection_name,
                agent=self._sdk.agent,
                log_activity=True,
            )
        return None
