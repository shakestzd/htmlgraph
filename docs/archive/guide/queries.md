# Query Cookbook

Wipnote provides multiple ways to query your graph data. This cookbook covers all query methods with practical examples.

## Query Methods Overview

| Method | Use Case | Example |
|--------|----------|---------|
| `query()` | CSS selector queries | `graph.query('[data-status="blocked"]')` |
| `query_builder()` | Complex conditions with AND/OR/NOT | `graph.query_builder().where("status", "blocked").execute()` |
| `find()` | Single node lookup | `graph.find(type="feature", status="done")` |
| `find_all()` | Multiple nodes with filters | `graph.find_all(priority="high")` |

## CSS Selector Queries

The simplest way to query - uses familiar CSS selector syntax.

```bash
# Use CLI to query by status
wipnote find features --status blocked
wipnote find features --status done
```

Or using the Wipnote Python library directly:

```python
from wipnote import Wipnote

graph = Wipnote(".wipnote")

# Status-based queries
blocked = graph.query('[data-status="blocked"]')
done = graph.query('[data-status="done"]')

# Priority queries
high_priority = graph.query('[data-priority="high"]')
critical = graph.query('[data-priority="critical"]')

# Type queries
features = graph.query('[data-type="feature"]')
sessions = graph.query('[data-type="session"]')

# Combined selectors
urgent = graph.query('[data-status="blocked"][data-priority="high"]')
```

**Limitations**: CSS selectors cannot express OR logic, NOT conditions, numeric comparisons, or text search.

## Fluent Query Builder

For complex queries that CSS selectors can't handle.

### Basic Conditions

```python
# Start a query
qb = graph.query_builder()

# Simple equality
features = qb.where("type", "feature").execute()

# With status
blocked = qb.where("status", "blocked").execute()
```

### Chaining Conditions

```python
# AND conditions (implicit)
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
```

### Comparison Operators

```python
# Greater than
large_effort = (graph.query_builder()
    .where("properties.effort").gt(8)
    .execute())

# Less than
quick_tasks = (graph.query_builder()
    .where("properties.effort").lt(4)
    .execute())

# Greater than or equal
medium_plus = (graph.query_builder()
    .where("properties.effort").gte(5)
    .execute())

# Less than or equal
small_tasks = (graph.query_builder()
    .where("properties.effort").lte(2)
    .execute())

# Between (inclusive)
medium_tasks = (graph.query_builder()
    .where("properties.effort").between(3, 6)
    .execute())
```

### Text Search

```python
# Contains substring
auth_features = (graph.query_builder()
    .where("title").contains("auth")
    .execute())

# Case-insensitive contains
auth_any_case = (graph.query_builder()
    .where("title").icontains("AUTH")
    .execute())

# Regex matching
api_features = (graph.query_builder()
    .where("title").matches(r"API|REST|GraphQL")
    .execute())
```

### List Operations

```python
# In list
high_priorities = (graph.query_builder()
    .where("priority").in_(["high", "critical"])
    .execute())

# Not in list
not_done_or_blocked = (graph.query_builder()
    .where("status").not_in(["done", "blocked"])
    .execute())
```

### Nested Attributes

```python
# Access nested properties
high_effort = (graph.query_builder()
    .where("properties.effort").gt(10)
    .execute())

# Multiple levels deep
specific_config = (graph.query_builder()
    .where("properties.config.enabled", True)
    .execute())
```

### Result Methods

```python
qb = graph.query_builder().where("status", "blocked")

# Get all results
all_blocked = qb.execute()

# Get first match
first_blocked = qb.first()

# Get count only
blocked_count = qb.count()
```

## Find API (BeautifulSoup-style)

Simple, intuitive queries inspired by BeautifulSoup.

### Basic Find

```python
# Find first match
feature = graph.find(type="feature")
blocked = graph.find(status="blocked")

# Find with multiple criteria
urgent = graph.find(type="feature", status="blocked", priority="high")
```

### Find All

```python
# Find all matches
all_features = graph.find_all(type="feature")
all_blocked = graph.find_all(status="blocked")

# With limit
top_5 = graph.find_all(type="feature", limit=5)
```

### Django-style Lookup Suffixes

```python
# Contains (case-sensitive)
auth = graph.find_all(title__contains="auth")

# Case-insensitive contains
auth_any = graph.find_all(title__icontains="AUTH")

# Starts with
api_features = graph.find_all(title__startswith="API")

# Ends with
service_features = graph.find_all(title__endswith="Service")

# Regex
pattern_match = graph.find_all(title__regex=r"v\d+")

# Numeric comparisons
high_effort = graph.find_all(properties__effort__gt=8)
low_effort = graph.find_all(properties__effort__lt=4)
medium = graph.find_all(properties__effort__gte=4, properties__effort__lte=8)

# In list
priority_filter = graph.find_all(priority__in=["high", "critical"])

# Not in list
not_completed = graph.find_all(status__not_in=["done", "cancelled"])

# Is null / Is not null
no_assignee = graph.find_all(properties__assignee__isnull=True)
has_assignee = graph.find_all(properties__assignee__isnull=False)
```

### Relationship Queries

```python
# Find nodes related to a specific node
related = graph.find_related("feature-001")

# Find by specific relationship type
blockers = graph.find_related("feature-001", relationship="blocked_by")

# Convenience methods for common relationships
blocking = graph.find_blocking("feature-001")  # What this blocks
blocked_by = graph.find_blocked_by("feature-001")  # What blocks this
```

## Graph Traversal

Navigate the graph structure.

### Ancestors and Descendants

```python
# Get all ancestors (nodes this depends on)
ancestors = graph.ancestors("feature-001")

# With depth limit
immediate_deps = graph.ancestors("feature-001", max_depth=1)
two_levels = graph.ancestors("feature-001", max_depth=2)

# Get all descendants (nodes that depend on this)
descendants = graph.descendants("feature-001")

# With depth limit
immediate_dependents = graph.descendants("feature-001", max_depth=1)
```

### Path Finding

```python
# Find all paths between two nodes
paths = graph.all_paths("feature-001", "feature-010")

# With max length constraint
short_paths = graph.all_paths("feature-001", "feature-010", max_length=4)

# Existing shortest path
shortest = graph.shortest_path("feature-001", "feature-010")
```

### Subgraph Extraction

```python
# Extract a subgraph with specific nodes
subset = graph.subgraph(["feature-001", "feature-002", "feature-003"])

# Without internal edges
nodes_only = graph.subgraph(["feature-001", "feature-002"], include_edges=False)
```

### Connected Components

```python
# Get all nodes in the same connected component
component = graph.connected_component("feature-001")

# Filter by relationship type
blocking_component = graph.connected_component("feature-001", relationship="blocked_by")
```

## Edge Index (O(1) Lookups)

Efficient reverse edge lookups.

```python
# Get edges pointing TO a node
incoming = graph.get_incoming_edges("feature-001")

# Filter by relationship
blockers = graph.get_incoming_edges("feature-001", relationship="blocked_by")

# Get edges pointing FROM a node
outgoing = graph.get_outgoing_edges("feature-001")

# Get all connected neighbors
neighbors = graph.get_neighbors("feature-001")
```

## Common Patterns

### Finding Bottlenecks

```python
# Nodes that block the most others
def find_top_blockers(graph, limit=5):
    nodes = graph.find_all(type="feature")
    blockers = []
    for node in nodes:
        blocked_count = len(graph.descendants(node.id, relationship="blocked_by"))
        if blocked_count > 0:
            blockers.append((node, blocked_count))
    return sorted(blockers, key=lambda x: x[1], reverse=True)[:limit]
```

### Finding Leaf Nodes

```python
# Nodes with no dependencies
def find_leaf_nodes(graph):
    return graph.query_builder() \
        .where("type", "feature") \
        .execute()
    # Then filter for nodes where ancestors() returns empty
    return [n for n in graph.find_all(type="feature")
            if not graph.ancestors(n.id)]
```

### Finding Ready Tasks

```python
# Tasks where all dependencies are done
def find_ready_tasks(graph):
    ready = []
    for node in graph.find_all(type="feature", status="todo"):
        blockers = graph.find_blocked_by(node.id)
        if all(b.status == "done" for b in blockers):
            ready.append(node)
    return ready
```

### Dependency Chain Analysis

```python
# Find the longest dependency chain
def find_critical_path(graph):
    features = graph.find_all(type="feature")
    max_path = []
    for f1 in features:
        for f2 in features:
            if f1.id != f2.id:
                paths = graph.all_paths(f1.id, f2.id, relationship="blocked_by")
                for path in paths:
                    if len(path) > len(max_path):
                        max_path = path
    return max_path
```

## Performance Tips

1. **Use EdgeIndex for reverse lookups**: `get_incoming_edges()` is O(1) vs O(V*E) for scanning
2. **Limit traversal depth**: Use `max_depth` parameter when you don't need full transitive closure
3. **Use `first()` when you only need one result**: Avoids iterating entire graph
4. **Prefer `find_all()` with filters**: More efficient than filtering after `get_nodes()`
5. **Cache frequently accessed subgraphs**: Use `subgraph()` to create smaller working sets
