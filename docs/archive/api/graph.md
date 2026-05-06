# Wipnote

Core graph operations, queries, and traversal algorithms.

## Overview

The `Wipnote` class provides:

- Node and edge management
- Multiple query methods (CSS selectors, QueryBuilder, Find API)
- Graph traversal algorithms
- O(1) edge index for reverse lookups

## Initialization

```python
from wipnote import Wipnote

# Initialize with graph directory
graph = Wipnote(graph_dir=".wipnote")
```

## Node Operations

### Adding Nodes

```python
from wipnote.models import Node

node = Node(
    id="feature-001",
    title="Add login",
    type="feature",
    status="todo",
    priority="high"
)

graph.add(node)
```

### Getting Nodes

```python
# Get single node by ID
node = graph.get("feature-001")

# Get all nodes
nodes = list(graph.nodes())

# Get nodes by type
features = [n for n in graph.nodes() if n.type == "feature"]
```

### Updating Nodes

```python
node = graph.get("feature-001")
node.status = "done"
graph.update(node)
```

### Removing Nodes

```python
graph.remove("feature-001")
```

## Query Methods

Wipnote provides four ways to query nodes:

### 1. CSS Selector Queries

```python
# Simple attribute queries
blocked = graph.query('[data-status="blocked"]')
high_priority = graph.query('[data-priority="high"]')

# Combined selectors (AND)
urgent = graph.query('[data-status="blocked"][data-priority="high"]')
```

### 2. QueryBuilder (Fluent API)

For complex queries with OR, NOT, numeric comparisons, and text search.

```python
from wipnote import QueryBuilder

# Simple equality
features = graph.query_builder().where("type", "feature").execute()

# Chained conditions
urgent = (graph.query_builder()
    .where("status", "blocked")
    .and_("priority", "high")
    .execute())

# OR conditions
high_or_critical = (graph.query_builder()
    .where("priority", "high")
    .or_("priority", "critical")
    .execute())

# NOT conditions
not_done = (graph.query_builder()
    .where("type", "feature")
    .not_("status").eq("done")
    .execute())

# Numeric comparisons
high_effort = (graph.query_builder()
    .where("properties.effort").gt(8)
    .execute())

# Text search
auth_features = (graph.query_builder()
    .where("title").contains("auth")
    .execute())

# Regex matching
api_features = (graph.query_builder()
    .where("title").matches(r"API|REST")
    .execute())

# List operations
priorities = (graph.query_builder()
    .where("priority").in_(["high", "critical"])
    .execute())

# Result methods
first_blocked = graph.query_builder().where("status", "blocked").first()
blocked_count = graph.query_builder().where("status", "blocked").count()
```

### 3. Find API (BeautifulSoup-style)

Simple, intuitive queries with Django-style lookup suffixes.

```python
# Find first match
feature = graph.find(type="feature")
blocked = graph.find(status="blocked", priority="high")

# Find all matches
all_features = graph.find_all(type="feature")
blocked_high = graph.find_all(status="blocked", priority="high")

# With limit
top_5 = graph.find_all(type="feature", limit=5)

# Django-style lookup suffixes
auth = graph.find_all(title__contains="auth")
high_effort = graph.find_all(properties__effort__gt=8)
priorities = graph.find_all(priority__in=["high", "critical"])
```

**Available lookup suffixes:**
- `__contains`, `__icontains` - substring search
- `__startswith`, `__endswith` - prefix/suffix match
- `__regex` - regular expression
- `__gt`, `__gte`, `__lt`, `__lte` - numeric comparisons
- `__in`, `__not_in` - list membership
- `__isnull` - null check

### 4. Relationship Queries

```python
# Find nodes related to a specific node
related = graph.find_related("feature-001")

# By relationship type
blockers = graph.find_related("feature-001", relationship="blocked_by")

# Convenience methods
blocking = graph.find_blocking("feature-001")  # What this blocks
blocked_by = graph.find_blocked_by("feature-001")  # What blocks this
```

## Edge Index

O(1) reverse edge lookups (vs O(V×E) linear scan).

```python
# Get edges pointing TO a node (incoming)
incoming = graph.get_incoming_edges("feature-001")
blockers = graph.get_incoming_edges("feature-001", relationship="blocked_by")

# Get edges pointing FROM a node (outgoing)
outgoing = graph.get_outgoing_edges("feature-001")

# Get all connected neighbors
neighbors = graph.get_neighbors("feature-001")
```

## Graph Traversal

### Ancestors and Descendants

```python
# Get all ancestors (nodes this depends on)
ancestors = graph.ancestors("feature-001")

# With depth limit
immediate_deps = graph.ancestors("feature-001", max_depth=1)

# Get all descendants (nodes that depend on this)
descendants = graph.descendants("feature-001")

# With depth limit
immediate_dependents = graph.descendants("feature-001", max_depth=1)
```

### Transitive Dependencies

```python
# Get all transitive dependencies (follows blocked_by edges)
deps = graph.transitive_deps("feature-001")
```

### Path Finding

```python
# Find shortest path between nodes
shortest = graph.shortest_path("feature-001", "feature-045")

# Find all paths
all_paths = graph.all_paths("feature-001", "feature-045")

# With max length constraint
short_paths = graph.all_paths("feature-001", "feature-045", max_length=4)
```

### Subgraph Extraction

```python
# Extract subgraph with specific nodes
subset = graph.subgraph(["feature-001", "feature-002", "feature-003"])

# Without internal edges
nodes_only = graph.subgraph(["f-001", "f-002"], include_edges=False)
```

### Connected Components

```python
# Get all nodes in the same connected component
component = graph.connected_component("feature-001")

# Filter by relationship type
blocking_component = graph.connected_component("feature-001", relationship="blocked_by")
```

## Graph Algorithms

### Find Bottlenecks

```python
# Features blocking many others
bottlenecks = graph.find_bottlenecks()
```

### Critical Path

```python
# Longest dependency chain
critical_path = graph.find_critical_path()
```

### Dependents

```python
# Nodes that directly depend on this one (O(1) with EdgeIndex)
dependents = graph.dependents("feature-001")
```

## Complete Method Reference

### Node Management
| Method | Description |
|--------|-------------|
| `add(node, overwrite=False)` | Add a node to the graph |
| `get(node_id)` | Get node by ID |
| `update(node)` | Update an existing node |
| `remove(node_id)` | Remove a node |
| `nodes()` | Iterator over all nodes |

### Query Methods
| Method | Description |
|--------|-------------|
| `query(css_selector)` | CSS selector query |
| `query_builder()` | Start fluent query builder |
| `find(type=None, **kwargs)` | Find first matching node |
| `find_all(type=None, limit=None, **kwargs)` | Find all matching nodes |
| `find_related(node_id, relationship=None)` | Find related nodes |
| `find_blocking(node_id)` | Find nodes this blocks |
| `find_blocked_by(node_id)` | Find nodes blocking this |

### Edge Index
| Method | Description |
|--------|-------------|
| `get_incoming_edges(node_id, relationship=None)` | Edges pointing to node |
| `get_outgoing_edges(node_id, relationship=None)` | Edges from node |
| `get_neighbors(node_id)` | All connected node IDs |

### Traversal
| Method | Description |
|--------|-------------|
| `ancestors(node_id, relationship, max_depth)` | All ancestor nodes |
| `descendants(node_id, relationship, max_depth)` | All descendant nodes |
| `transitive_deps(node_id)` | All transitive dependencies |
| `shortest_path(from_id, to_id)` | Shortest path between nodes |
| `all_paths(from_id, to_id, relationship, max_length)` | All paths between nodes |
| `subgraph(node_ids, include_edges)` | Extract subgraph |
| `connected_component(node_id, relationship)` | Connected component |

### Analysis
| Method | Description |
|--------|-------------|
| `find_bottlenecks()` | Nodes blocking many others |
| `find_critical_path()` | Longest dependency chain |
| `dependents(node_id)` | Direct dependents |

## See Also

- [Query Cookbook](../guide/queries.md) - Comprehensive query examples
- [Migration Guide](../guide/migration.md) - Migrating from CSS-only queries
