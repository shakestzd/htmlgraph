"""
Graph algorithms for HtmlGraph.

Provides pure graph algorithm implementations:
- Shortest path (BFS)
- Transitive dependencies and dependents
- Cycle detection
- Topological sorting
- Ancestor/descendant traversal
- Connected components
- Subgraph extraction
"""

from __future__ import annotations

from collections import defaultdict, deque
from typing import TYPE_CHECKING, Any

if TYPE_CHECKING:
    from .core import HtmlGraph


def shortest_path(
    graph: HtmlGraph, from_id: str, to_id: str, relationship: str | None = None
) -> list[str] | None:
    """
    Find shortest path between two nodes using BFS.

    Args:
        graph: HtmlGraph instance
        from_id: Starting node ID
        to_id: Target node ID
        relationship: Optional filter to specific edge type

    Returns:
        List of node IDs representing path, or None if no path exists
    """
    if from_id not in graph._nodes or to_id not in graph._nodes:
        return None

    if from_id == to_id:
        return [from_id]

    adj = _build_adjacency(graph, relationship)

    # BFS
    queue = deque([(from_id, [from_id])])
    visited = {from_id}

    while queue:
        current, path = queue.popleft()

        for neighbor in adj.get(current, []):
            if neighbor == to_id:
                return path + [neighbor]

            if neighbor not in visited and neighbor in graph._nodes:
                visited.add(neighbor)
                queue.append((neighbor, path + [neighbor]))

    return None


def transitive_deps(
    graph: HtmlGraph, node_id: str, relationship: str = "blocked_by"
) -> set[str]:
    """
    Get all transitive dependencies of a node.

    Follows edges recursively to find all nodes that must be
    completed before this one.

    Args:
        graph: HtmlGraph instance
        node_id: Starting node ID
        relationship: Edge type to follow (default: blocked_by)

    Returns:
        Set of all dependency node IDs
    """
    if node_id not in graph._nodes:
        return set()

    deps: set[str] = set()
    queue = deque([node_id])

    while queue:
        current = queue.popleft()
        node = graph._nodes.get(current)
        if not node:
            continue

        for edge in node.edges.get(relationship, []):
            if edge.target_id not in deps:
                deps.add(edge.target_id)
                if edge.target_id in graph._nodes:
                    queue.append(edge.target_id)

    return deps


def get_dependencies(
    graph: HtmlGraph, node_id: str, rel_type: str = "blocked_by"
) -> set[str]:
    """
    Get all transitive dependencies of a node, filtered by relationship type.

    Alias for transitive_deps() with a named rel_type parameter for clarity.
    When rel_type is specified, only edges of that type are traversed.

    Args:
        graph: HtmlGraph instance
        node_id: Starting node ID
        rel_type: Edge relationship type to follow (default: blocked_by)

    Returns:
        Set of all dependency node IDs reachable via edges of rel_type

    Example:
        # Only follow blocked_by edges
        deps = get_dependencies(graph, "feat-001", rel_type="blocked_by")
        # Only follow depends_on edges
        deps = get_dependencies(graph, "feat-001", rel_type="depends_on")
    """
    return transitive_deps(graph, node_id, relationship=rel_type)


def dependents(
    graph: HtmlGraph, node_id: str, relationship: str = "blocked_by"
) -> set[str]:
    """
    Find all nodes that depend on this node (O(1) lookup).

    Uses the edge index for efficient reverse lookups.

    Args:
        graph: HtmlGraph instance
        node_id: Node to find dependents for
        relationship: Edge type indicating dependency

    Returns:
        Set of node IDs that depend on this node
    """
    # O(1) lookup using edge index instead of O(V×E) scan
    incoming = graph._edge_index.get_incoming(node_id, relationship)
    return {ref.source_id for ref in incoming}


def find_bottlenecks(
    graph: HtmlGraph, relationship: str = "blocked_by", top_n: int = 5
) -> list[tuple[str, int]]:
    """
    Find nodes that block the most other nodes.

    Args:
        graph: HtmlGraph instance
        relationship: Edge type indicating blocking
        top_n: Number of top bottlenecks to return

    Returns:
        List of (node_id, blocked_count) tuples, sorted by count descending
    """
    blocked_count: dict[str, int] = defaultdict(int)

    for node in graph._nodes.values():
        for edge in node.edges.get(relationship, []):
            blocked_count[edge.target_id] += 1

    sorted_bottlenecks = sorted(blocked_count.items(), key=lambda x: x[1], reverse=True)

    return sorted_bottlenecks[:top_n]


def find_cycles(graph: HtmlGraph, relationship: str = "blocked_by") -> list[list[str]]:
    """
    Detect cycles in the graph.

    Args:
        graph: HtmlGraph instance
        relationship: Edge type to check for cycles

    Returns:
        List of cycles, each as a list of node IDs
    """
    adj = _build_adjacency(graph, relationship)
    cycles: list[list[str]] = []
    visited: set[str] = set()
    rec_stack: set[str] = set()

    def dfs(node: str, path: list[str]) -> None:
        visited.add(node)
        rec_stack.add(node)
        path.append(node)

        for neighbor in adj.get(node, []):
            if neighbor not in visited:
                dfs(neighbor, path)
            elif neighbor in rec_stack:
                # Found cycle
                cycle_start = path.index(neighbor)
                cycles.append(path[cycle_start:] + [neighbor])

        path.pop()
        rec_stack.remove(node)

    for node_id in graph._nodes:
        if node_id not in visited:
            dfs(node_id, [])

    return cycles


def topological_sort(
    graph: HtmlGraph, relationship: str = "blocked_by"
) -> list[str] | None:
    """
    Return nodes in topological order (dependencies first).

    Args:
        graph: HtmlGraph instance
        relationship: Edge type indicating dependency

    Returns:
        List of node IDs in dependency order, or None if cycles exist
    """
    # Build in-degree map
    in_degree: dict[str, int] = {node_id: 0 for node_id in graph._nodes}

    for node in graph._nodes.values():
        for edge in node.edges.get(relationship, []):
            if edge.target_id in in_degree:
                in_degree[node.id] = in_degree.get(node.id, 0) + 1

    # Start with nodes having no dependencies
    queue = deque([n for n, d in in_degree.items() if d == 0])
    result: list[str] = []

    while queue:
        node_id = queue.popleft()
        result.append(node_id)

        # Reduce in-degree of dependents
        for dependent in dependents(graph, node_id, relationship):
            in_degree[dependent] -= 1
            if in_degree[dependent] == 0:
                queue.append(dependent)

    # Check for cycles
    if len(result) != len(graph._nodes):
        return None

    return result


def ancestors(
    graph: HtmlGraph,
    node_id: str,
    relationship: str = "blocked_by",
    max_depth: int | None = None,
) -> list[str]:
    """
    Get all ancestor nodes (nodes that this node depends on).

    Traverses incoming edges recursively to find all predecessors.

    Args:
        graph: HtmlGraph instance
        node_id: Starting node ID
        relationship: Edge type to follow (default: blocked_by)
        max_depth: Maximum traversal depth (None = unlimited)

    Returns:
        List of ancestor node IDs in BFS order (nearest first)
    """
    if node_id not in graph._nodes:
        return []

    ancestors_list: list[str] = []
    visited: set[str] = set()
    queue = deque([(node_id, 0)])
    visited.add(node_id)

    while queue:
        current, depth = queue.popleft()

        # Skip if we've hit max depth
        if max_depth is not None and depth >= max_depth:
            continue

        # Get nodes this one depends on (outgoing blocked_by edges)
        node = graph._nodes.get(current)
        if not node:
            continue

        for edge in node.edges.get(relationship, []):
            if edge.target_id not in visited:
                visited.add(edge.target_id)
                ancestors_list.append(edge.target_id)
                if edge.target_id in graph._nodes:
                    queue.append((edge.target_id, depth + 1))

    return ancestors_list


def descendants(
    graph: HtmlGraph,
    node_id: str,
    relationship: str = "blocked_by",
    max_depth: int | None = None,
) -> list[str]:
    """
    Get all descendant nodes (nodes that depend on this node).

    Traverses incoming edges (reverse direction) to find all successors.

    Args:
        graph: HtmlGraph instance
        node_id: Starting node ID
        relationship: Edge type to follow (default: blocked_by)
        max_depth: Maximum traversal depth (None = unlimited)

    Returns:
        List of descendant node IDs in BFS order (nearest first)
    """
    if node_id not in graph._nodes:
        return []

    descendants_list: list[str] = []
    visited: set[str] = set()
    queue = deque([(node_id, 0)])
    visited.add(node_id)

    while queue:
        current, depth = queue.popleft()

        if max_depth is not None and depth >= max_depth:
            continue

        # Get nodes that depend on this one (incoming edges)
        incoming = graph._edge_index.get_incoming(current, relationship)

        for ref in incoming:
            if ref.source_id not in visited:
                visited.add(ref.source_id)
                descendants_list.append(ref.source_id)
                queue.append((ref.source_id, depth + 1))

    return descendants_list


def subgraph(
    graph: HtmlGraph, node_ids: list[str] | set[str], include_edges: bool = True
) -> HtmlGraph:
    """
    Extract a subgraph containing only the specified nodes.

    Args:
        graph: HtmlGraph instance
        node_ids: Node IDs to include in subgraph
        include_edges: Whether to include edges between nodes (default: True)

    Returns:
        New HtmlGraph containing only specified nodes

    Example:
        # Get subgraph of a node and its dependencies
        deps = graph.transitive_deps("feature-001")
        deps.add("feature-001")
        sub = graph.subgraph(deps)
    """
    import tempfile

    from .core import HtmlGraph

    # Create new graph in temp directory
    temp_dir = tempfile.mkdtemp(prefix="htmlgraph_subgraph_")
    subgraph_obj = HtmlGraph(temp_dir, auto_load=False)

    node_ids_set = set(node_ids)

    for node_id in node_ids:
        node = graph._nodes.get(node_id)
        if not node:
            continue

        # Create copy of node
        if include_edges:
            # Filter edges to only include those pointing to nodes in subgraph
            filtered_edges = {}
            for rel_type, edges in node.edges.items():
                filtered = [e for e in edges if e.target_id in node_ids_set]
                if filtered:
                    filtered_edges[rel_type] = filtered
            node_copy = node.model_copy(update={"edges": filtered_edges})
        else:
            node_copy = node.model_copy(update={"edges": {}})

        subgraph_obj.add(node_copy)

    return subgraph_obj


def connected_component(
    graph: HtmlGraph, node_id: str, relationship: str | None = None
) -> set[str]:
    """
    Get all nodes in the same connected component as the given node.

    Treats edges as undirected (both directions).

    Args:
        graph: HtmlGraph instance
        node_id: Starting node ID
        relationship: Optional filter to specific edge type

    Returns:
        Set of node IDs in the connected component
    """
    if node_id not in graph._nodes:
        return set()

    component: set[str] = set()
    queue = deque([node_id])

    while queue:
        current = queue.popleft()
        if current in component:
            continue

        component.add(current)

        # Get all neighbors (both directions)
        neighbors = graph._edge_index.get_neighbors(current, relationship, "both")
        for neighbor in neighbors:
            if neighbor not in component and neighbor in graph._nodes:
                queue.append(neighbor)

    return component


def all_paths(
    graph: HtmlGraph,
    from_id: str,
    to_id: str,
    relationship: str | None = None,
    max_length: int | None = None,
    max_paths: int = 100,
    timeout_seconds: float = 5.0,
) -> list[list[str]]:
    """
    Find all paths between two nodes.

    WARNING: This method has O(V!) worst-case complexity in dense graphs.
    Use max_paths and timeout_seconds parameters to limit execution.
    For most use cases, prefer shortest_path() instead.

    Args:
        graph: HtmlGraph instance
        from_id: Source node ID
        to_id: Target node ID
        relationship: Optional edge type filter
        max_length: Maximum path length
        max_paths: Maximum number of paths to return (default 100)
        timeout_seconds: Maximum execution time (default 5.0)

    Returns:
        List of paths (each path is list of node IDs)

    Raises:
        TimeoutError: If execution exceeds timeout_seconds
    """
    import time

    if from_id not in graph._nodes or to_id not in graph._nodes:
        return []

    if from_id == to_id:
        return [[from_id]]

    paths: list[list[str]] = []
    adj = _build_adjacency(graph, relationship)
    start_time = time.time()

    def dfs(current: str, target: str, path: list[str], visited: set[str]) -> None:
        # Check timeout periodically (every recursive call)
        if time.time() - start_time > timeout_seconds:
            raise TimeoutError(
                f"all_paths() exceeded timeout of {timeout_seconds}s "
                f"(found {len(paths)} paths so far)"
            )

        # Check if we've hit the max_paths limit
        if len(paths) >= max_paths:
            return

        if max_length and len(path) > max_length:
            return

        if current == target:
            paths.append(path.copy())
            return

        for neighbor in adj.get(current, []):
            if neighbor not in visited:
                visited.add(neighbor)
                path.append(neighbor)
                dfs(neighbor, target, path, visited)
                path.pop()
                visited.remove(neighbor)

    dfs(from_id, to_id, [from_id], {from_id})
    return paths


def stats(graph: HtmlGraph) -> dict[str, Any]:
    """
    Get graph statistics.

    Args:
        graph: HtmlGraph instance

    Returns:
        Dict with:
        - total: Total node count
        - by_status: Count per status
        - by_type: Count per type
        - by_priority: Count per priority
        - completion_rate: Overall completion percentage
        - edge_count: Total number of edges
    """
    by_status: defaultdict[str, int] = defaultdict(int)
    by_type: defaultdict[str, int] = defaultdict(int)
    by_priority: defaultdict[str, int] = defaultdict(int)
    edge_count = 0

    stats_dict: dict[str, Any] = {
        "total": len(graph._nodes),
        "by_status": by_status,
        "by_type": by_type,
        "by_priority": by_priority,
        "edge_count": edge_count,
    }

    done_count = 0
    for node in graph._nodes.values():
        by_status[node.status] += 1
        by_type[node.type] += 1
        by_priority[node.priority] += 1

        for edges in node.edges.values():
            edge_count += len(edges)

        if node.status == "done":
            done_count += 1

    stats_dict["edge_count"] = edge_count
    stats_dict["completion_rate"] = (
        round(done_count / len(graph._nodes) * 100, 1) if graph._nodes else 0
    )

    # Convert defaultdicts to regular dicts
    stats_dict["by_status"] = dict(by_status)
    stats_dict["by_type"] = dict(by_type)
    stats_dict["by_priority"] = dict(by_priority)

    return stats_dict


def to_mermaid(graph: HtmlGraph, relationship: str | None = None) -> str:
    """
    Export graph as Mermaid diagram.

    Args:
        graph: HtmlGraph instance
        relationship: Optional filter to specific edge type

    Returns:
        Mermaid diagram string
    """
    lines = ["graph TD"]

    for node in graph._nodes.values():
        # Node definition with status styling
        node_label = f"{node.id}[{node.title}]"
        lines.append(f"    {node_label}")

        # Edges
        for rel_type, edges in node.edges.items():
            if relationship and rel_type != relationship:
                continue
            for edge in edges:
                arrow = "-->" if rel_type != "blocked_by" else "-.->|blocked|"
                lines.append(f"    {node.id} {arrow} {edge.target_id}")

    return "\n".join(lines)


# Private helper functions


def _build_adjacency(
    graph: HtmlGraph, relationship: str | None = None
) -> dict[str, set[str]]:
    """
    Build adjacency list from edges.

    Args:
        graph: HtmlGraph instance
        relationship: Filter to specific relationship type, or None for all

    Returns:
        Dict mapping node_id to set of connected node_ids
    """
    adj: dict[str, set[str]] = defaultdict(set)

    for node in graph._nodes.values():
        for rel_type, edges in node.edges.items():
            if relationship and rel_type != relationship:
                continue
            for edge in edges:
                adj[node.id].add(edge.target_id)

    return adj
