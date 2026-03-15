"""
Critical Path Analysis for HtmlGraph.

Analyzes feature dependency graphs to find critical paths and bottlenecks,
using the `graph_edges` and `features` SQLite tables.

Functions:
- find_critical_path(track_id, db_path) -> list[str]
- find_bottlenecks(track_id, db_path) -> list[dict]
- get_dependency_graph(track_id, db_path) -> dict
"""

from __future__ import annotations

import logging
import sqlite3
from collections import defaultdict, deque
from pathlib import Path
from typing import Any

logger = logging.getLogger(__name__)

# Relationship types that indicate "A blocks B" (edge from A -> B means A must be done first)
_BLOCKS_RELATIONSHIPS = {"blocks", "blocked_by"}


def _get_default_db_path() -> str:
    """Return the default database path for the current project."""
    import os
    import subprocess

    env_dir = os.environ.get("HTMLGRAPH_PROJECT_DIR") or os.environ.get(
        "CLAUDE_PROJECT_DIR"
    )
    if env_dir:
        return str(Path(env_dir) / ".htmlgraph" / "htmlgraph.db")

    try:
        result = subprocess.run(
            ["git", "rev-parse", "--show-toplevel"],
            capture_output=True,
            text=True,
            timeout=5,
        )
        if result.returncode == 0:
            project_root = Path(result.stdout.strip())
            return str(project_root / ".htmlgraph" / "htmlgraph.db")
    except Exception:
        pass

    return str(Path.home() / ".htmlgraph" / "htmlgraph.db")


def _connect(db_path: str) -> sqlite3.Connection:
    """Open a read-friendly SQLite connection."""
    conn = sqlite3.connect(db_path)
    conn.row_factory = sqlite3.Row
    conn.execute("PRAGMA journal_mode=WAL")
    conn.execute("PRAGMA foreign_keys=ON")
    return conn


def _load_features(
    cursor: sqlite3.Cursor, track_id: str | None
) -> dict[str, dict[str, Any]]:
    """
    Load features from the database.

    Returns a mapping of feature_id -> feature dict with keys:
    id, title, status, type.
    """
    if track_id is not None:
        cursor.execute(
            "SELECT id, title, status, type FROM features WHERE track_id = ?",
            (track_id,),
        )
    else:
        cursor.execute("SELECT id, title, status, type FROM features")

    return {row["id"]: dict(row) for row in cursor.fetchall()}


def _load_edges(
    cursor: sqlite3.Cursor, feature_ids: set[str]
) -> list[tuple[str, str, str]]:
    """
    Load dependency edges from graph_edges for the given set of feature IDs.

    Returns list of (from_node_id, to_node_id, relationship_type) tuples.

    Edge semantics:
    - relationship_type == "blocks":    from_node blocks to_node (from must finish first)
    - relationship_type == "blocked_by": from_node is blocked_by to_node (to must finish first)
    """
    if not feature_ids:
        return []

    placeholders = ",".join("?" for _ in feature_ids)
    cursor.execute(
        f"""
        SELECT from_node_id, to_node_id, relationship_type
        FROM graph_edges
        WHERE relationship_type IN ('blocks', 'blocked_by')
          AND from_node_id IN ({placeholders})
          AND to_node_id IN ({placeholders})
        """,
        list(feature_ids) + list(feature_ids),
    )
    return [
        (row["from_node_id"], row["to_node_id"], row["relationship_type"])
        for row in cursor.fetchall()
    ]


def _build_adjacency(
    feature_ids: set[str],
    edges: list[tuple[str, str, str]],
) -> tuple[dict[str, list[str]], dict[str, list[str]]]:
    """
    Build adjacency lists from raw edges.

    Normalises both "blocks" and "blocked_by" into a single directed graph
    where an edge A -> B means "A must be completed before B".

    Returns:
        successors:  {node_id: [nodes that depend on this node]}
        predecessors: {node_id: [nodes this node depends on]}
    """
    successors: dict[str, list[str]] = defaultdict(list)
    predecessors: dict[str, list[str]] = defaultdict(list)

    for fid in feature_ids:
        successors.setdefault(fid, [])
        predecessors.setdefault(fid, [])

    for from_id, to_id, rel in edges:
        if from_id not in feature_ids or to_id not in feature_ids:
            continue

        if rel == "blocks":
            # from_id blocks to_id => from_id -> to_id
            if to_id not in successors[from_id]:
                successors[from_id].append(to_id)
            if from_id not in predecessors[to_id]:
                predecessors[to_id].append(from_id)
        elif rel == "blocked_by":
            # from_id is blocked by to_id => to_id -> from_id
            if from_id not in successors[to_id]:
                successors[to_id].append(from_id)
            if to_id not in predecessors[from_id]:
                predecessors[from_id].append(to_id)

    return dict(successors), dict(predecessors)


def _detect_cycles(
    feature_ids: set[str],
    successors: dict[str, list[str]],
) -> list[list[str]]:
    """
    Detect cycles using DFS coloring (white/gray/black).

    Returns list of cycles found (each cycle is a list of node IDs).
    """
    white, gray, black = 0, 1, 2
    color: dict[str, int] = {fid: white for fid in feature_ids}
    cycles: list[list[str]] = []

    def dfs(node: str, path: list[str]) -> None:
        color[node] = gray
        path.append(node)
        for neighbor in successors.get(node, []):
            if color[neighbor] == gray:
                # Found a cycle — record it
                cycle_start = path.index(neighbor)
                cycles.append(path[cycle_start:] + [neighbor])
            elif color[neighbor] == white:
                dfs(neighbor, path)
        path.pop()
        color[node] = black

    for fid in feature_ids:
        if color[fid] == white:
            dfs(fid, [])

    return cycles


def _topological_sort_dag(
    feature_ids: set[str],
    successors: dict[str, list[str]],
    predecessors: dict[str, list[str]],
) -> list[str] | None:
    """
    Kahn's algorithm for topological sort.

    Returns ordered list, or None if a cycle prevents full ordering.
    """
    in_degree = {fid: len(predecessors.get(fid, [])) for fid in feature_ids}
    queue: deque[str] = deque(n for n in feature_ids if in_degree[n] == 0)
    order: list[str] = []

    while queue:
        node = queue.popleft()
        order.append(node)
        for succ in successors.get(node, []):
            in_degree[succ] -= 1
            if in_degree[succ] == 0:
                queue.append(succ)

    if len(order) != len(feature_ids):
        return None  # cycle detected

    return order


def _longest_path(
    topo_order: list[str],
    successors: dict[str, list[str]],
) -> list[str]:
    """
    Compute the longest path in a DAG using dynamic programming over topo order.

    Returns the sequence of node IDs forming the critical path.
    """
    dist: dict[str, int] = {node: 1 for node in topo_order}
    prev: dict[str, str | None] = {node: None for node in topo_order}

    for node in topo_order:
        for succ in successors.get(node, []):
            if dist[node] + 1 > dist[succ]:
                dist[succ] = dist[node] + 1
                prev[succ] = node

    if not dist:
        return []

    # Find the end of the longest path
    end_node = max(dist, key=lambda n: dist[n])

    # Reconstruct path by walking back through `prev`
    path: list[str] = []
    current: str | None = end_node
    while current is not None:
        path.append(current)
        current = prev[current]

    path.reverse()
    return path


def find_critical_path(
    track_id: str | None = None,
    db_path: str | None = None,
) -> list[str]:
    """
    Find the critical path through feature dependencies for a track.

    The critical path is the longest chain of dependent features — completing
    it determines the minimum time to finish the track.

    Algorithm:
    1. Load all features in the track from the `features` table.
    2. Load dependency edges from the `graph_edges` table.
    3. Detect cycles; if found, log a warning and proceed with a DAG approximation.
    4. Run topological sort + DP longest-path on the resulting DAG.

    Args:
        track_id: Track ID to scope the analysis (None = all features).
        db_path:  Path to the SQLite database. Defaults to project database.

    Returns:
        Ordered list of feature_ids on the critical path (first = no deps,
        last = terminal). Returns [] when there are no features or no edges.
    """
    resolved_db = db_path or _get_default_db_path()

    try:
        conn = _connect(resolved_db)
    except sqlite3.Error as exc:
        logger.error("Cannot open database %s: %s", resolved_db, exc)
        return []

    try:
        cursor = conn.cursor()
        features = _load_features(cursor, track_id)

        if not features:
            return []

        feature_ids = set(features)
        raw_edges = _load_edges(cursor, feature_ids)
    except sqlite3.Error as exc:
        logger.error("Database error loading features/edges: %s", exc)
        return []
    finally:
        conn.close()

    if not raw_edges:
        # No dependency edges — each feature is independent, critical path is any single node
        return []

    successors, predecessors = _build_adjacency(feature_ids, raw_edges)

    # Detect and report cycles before attempting topo sort
    cycles = _detect_cycles(feature_ids, successors)
    if cycles:
        cycle_strs = [" -> ".join(c) for c in cycles]
        logger.warning(
            "Cycle(s) detected in dependency graph: %s. "
            "Cycle members will be excluded from critical path.",
            cycle_strs,
        )
        # Remove cyclic nodes from consideration
        cyclic_nodes: set[str] = set()
        for cycle in cycles:
            cyclic_nodes.update(cycle)

        feature_ids -= cyclic_nodes
        if not feature_ids:
            return []

        # Rebuild adjacency without cyclic nodes
        successors, predecessors = _build_adjacency(feature_ids, raw_edges)

    topo_order = _topological_sort_dag(feature_ids, successors, predecessors)
    if topo_order is None:
        logger.error("Topological sort failed despite cycle removal; returning []")
        return []

    return _longest_path(topo_order, successors)


def find_bottlenecks(
    track_id: str | None = None,
    db_path: str | None = None,
) -> list[dict[str, Any]]:
    """
    Identify features that block the most other features.

    Ranks features by transitive dependent count — features that directly or
    indirectly block many others surface first.

    Args:
        track_id: Track ID to scope the analysis (None = all features).
        db_path:  Path to the SQLite database. Defaults to project database.

    Returns:
        List of dicts sorted descending by transitive_dependents::

            [
                {
                    "feature_id": "feat-abc",
                    "title": "Auth system",
                    "blocks_count": 3,
                    "transitive_dependents": 7,
                },
                ...
            ]
    """
    resolved_db = db_path or _get_default_db_path()

    try:
        conn = _connect(resolved_db)
    except sqlite3.Error as exc:
        logger.error("Cannot open database %s: %s", resolved_db, exc)
        return []

    try:
        cursor = conn.cursor()
        features = _load_features(cursor, track_id)

        if not features:
            return []

        feature_ids = set(features)
        raw_edges = _load_edges(cursor, feature_ids)
    except sqlite3.Error as exc:
        logger.error("Database error: %s", exc)
        return []
    finally:
        conn.close()

    successors, _predecessors = _build_adjacency(feature_ids, raw_edges)

    def _count_transitive(start: str) -> int:
        """BFS count of all nodes reachable from start via successors."""
        visited: set[str] = set()
        queue: deque[str] = deque(successors.get(start, []))
        while queue:
            node = queue.popleft()
            if node in visited:
                continue
            visited.add(node)
            queue.extend(successors.get(node, []))
        return len(visited)

    results = []
    for fid, feat in features.items():
        direct_blocked = successors.get(fid, [])
        if not direct_blocked:
            continue  # No direct dependents — not a bottleneck

        transitive = _count_transitive(fid)
        results.append(
            {
                "feature_id": fid,
                "title": feat["title"],
                "blocks_count": len(direct_blocked),
                "transitive_dependents": transitive,
            }
        )

    results.sort(key=lambda r: r["transitive_dependents"], reverse=True)
    return results


def get_dependency_graph(
    track_id: str | None = None,
    db_path: str | None = None,
) -> dict[str, Any]:
    """
    Return a dependency graph structure suitable for visualization.

    Args:
        track_id: Track ID to scope the analysis (None = all features).
        db_path:  Path to the SQLite database. Defaults to project database.

    Returns:
        Dictionary with ``nodes`` and ``edges`` keys::

            {
                "nodes": [
                    {"id": "feat-abc", "title": "...", "status": "todo", "type": "feature"},
                    ...
                ],
                "edges": [
                    {"from": "feat-abc", "to": "feat-def", "relationship": "blocks"},
                    ...
                ],
            }
    """
    resolved_db = db_path or _get_default_db_path()

    try:
        conn = _connect(resolved_db)
    except sqlite3.Error as exc:
        logger.error("Cannot open database %s: %s", resolved_db, exc)
        return {"nodes": [], "edges": []}

    try:
        cursor = conn.cursor()
        features = _load_features(cursor, track_id)

        if not features:
            return {"nodes": [], "edges": []}

        feature_ids = set(features)
        raw_edges = _load_edges(cursor, feature_ids)
    except sqlite3.Error as exc:
        logger.error("Database error: %s", exc)
        return {"nodes": [], "edges": []}
    finally:
        conn.close()

    nodes = [
        {
            "id": fid,
            "title": feat["title"],
            "status": feat["status"],
            "type": feat["type"],
        }
        for fid, feat in features.items()
    ]

    # Normalise edges: "blocks" A->B means from=A, to=B
    #                  "blocked_by" A<-B (B blocks A) means from=B, to=A
    seen_edges: set[tuple[str, str]] = set()
    edges = []
    for from_id, to_id, rel in raw_edges:
        if rel == "blocks":
            key = (from_id, to_id)
            if key not in seen_edges:
                seen_edges.add(key)
                edges.append({"from": from_id, "to": to_id, "relationship": "blocks"})
        elif rel == "blocked_by":
            # from_id is blocked_by to_id => to_id blocks from_id
            key = (to_id, from_id)
            if key not in seen_edges:
                seen_edges.add(key)
                edges.append({"from": to_id, "to": from_id, "relationship": "blocks"})

    return {"nodes": nodes, "edges": edges}
