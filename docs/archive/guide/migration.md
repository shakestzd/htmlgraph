# Migration Guide: CSS Selectors to Advanced Queries

This guide helps you migrate from CSS-only queries to Wipnote's advanced query APIs.

## Why Migrate?

CSS selectors are great for simple queries but have limitations:

| Feature | CSS Selectors | QueryBuilder/Find |
|---------|--------------|-------------------|
| AND conditions | `[a="x"][b="y"]` | `where("a", "x").and_("b", "y")` |
| OR conditions | Not possible | `where("a", "x").or_("a", "y")` |
| NOT conditions | Not possible | `not_("status").eq("done")` |
| Numeric comparisons | Not possible | `where("effort").gt(8)` |
| Text search | Not possible | `where("title").contains("auth")` |
| Regex matching | Not possible | `where("title").matches(r"v\d+")` |
| Nested attributes | Limited | `where("properties.effort").gt(8)` |

## Quick Migration Reference

### Simple Equality

```python
# Before (CSS)
graph.query('[data-status="blocked"]')

# After (QueryBuilder)
graph.query_builder().where("status", "blocked").execute()

# After (Find)
graph.find_all(status="blocked")
```

### Multiple Conditions (AND)

```python
# Before (CSS)
graph.query('[data-status="blocked"][data-priority="high"]')

# After (QueryBuilder)
graph.query_builder() \
    .where("status", "blocked") \
    .and_("priority", "high") \
    .execute()

# After (Find)
graph.find_all(status="blocked", priority="high")
```

### Type + Status

```python
# Before (CSS)
graph.query('[data-type="feature"][data-status="todo"]')

# After (QueryBuilder)
graph.query_builder() \
    .of_type("feature") \
    .where("status", "todo") \
    .execute()

# After (Find)
graph.find_all(type="feature", status="todo")
```

## New Capabilities

### OR Conditions

```python
# Not possible with CSS selectors

# QueryBuilder
graph.query_builder() \
    .where("priority", "high") \
    .or_("priority", "critical") \
    .execute()

# Find (use multiple calls or query builder)
high = graph.find_all(priority="high")
critical = graph.find_all(priority="critical")
combined = high + critical
```

### NOT Conditions

```python
# Not possible with CSS selectors

# QueryBuilder
graph.query_builder() \
    .where("type", "feature") \
    .not_("status").eq("done") \
    .execute()

# Find (filter after)
[n for n in graph.find_all(type="feature") if n.status != "done"]
```

### Numeric Comparisons

```python
# Not possible with CSS selectors

# QueryBuilder
graph.query_builder() \
    .where("properties.effort").gt(8) \
    .execute()

# Find with lookup suffix
graph.find_all(properties__effort__gt=8)
```

### Text Search

```python
# Not possible with CSS selectors

# QueryBuilder
graph.query_builder() \
    .where("title").contains("authentication") \
    .execute()

# Find with lookup suffix
graph.find_all(title__contains="authentication")
```

### Regex Matching

```python
# Not possible with CSS selectors

# QueryBuilder
graph.query_builder() \
    .where("title").matches(r"API|REST|GraphQL") \
    .execute()

# Find with lookup suffix
graph.find_all(title__regex=r"API|REST|GraphQL")
```

## Common Migration Patterns

### Pattern 1: Status-based Filtering

```python
# Old approach
blocked = graph.query('[data-status="blocked"]')
todo = graph.query('[data-status="todo"]')

# New approach (more readable)
blocked = graph.find_all(status="blocked")
todo = graph.find_all(status="todo")

# New approach (with additional filtering)
blocked_high = graph.find_all(status="blocked", priority="high")
```

### Pattern 2: Type Filtering

```python
# Old approach
features = graph.query('[data-type="feature"]')
sessions = graph.query('[data-type="session"]')

# New approach
features = graph.find_all(type="feature")
sessions = graph.find_all(type="session")
```

### Pattern 3: Complex Business Logic

```python
# Old approach (required post-processing)
all_features = graph.query('[data-type="feature"]')
urgent = [f for f in all_features
          if f.status == "blocked" and f.priority in ["high", "critical"]]

# New approach (single query)
urgent = graph.query_builder() \
    .of_type("feature") \
    .where("status", "blocked") \
    .and_("priority").in_(["high", "critical"]) \
    .execute()
```

### Pattern 4: Relationship-based Queries

```python
# Old approach (manual traversal)
node = graph.get_node("feature-001")
blocked_by_ids = [e.target_id for e in node.edges.get("blocked_by", [])]
blockers = [graph.get_node(id) for id in blocked_by_ids]

# New approach
blockers = graph.find_blocked_by("feature-001")

# Or with find_related
blockers = graph.find_related("feature-001", relationship="blocked_by")
```

### Pattern 5: Reverse Edge Lookups

```python
# Old approach (O(V*E) scan)
def find_dependents(graph, node_id):
    dependents = []
    for node in graph.get_nodes():
        for edge in node.edges.get("blocked_by", []):
            if edge.target_id == node_id:
                dependents.append(node)
    return dependents

# New approach (O(1) with EdgeIndex)
dependents = graph.descendants("feature-001", relationship="blocked_by")

# Or using edge index directly
incoming = graph.get_incoming_edges("feature-001", relationship="blocked_by")
```

## Backward Compatibility

The `query()` method with CSS selectors still works and will continue to work:

```python
# This will always work
graph.query('[data-status="blocked"]')
```

You can mix both approaches in your codebase:

```python
# Simple queries - use CSS selectors
blocked = graph.query('[data-status="blocked"]')

# Complex queries - use QueryBuilder
complex_result = graph.query_builder() \
    .where("status", "blocked") \
    .and_("priority").in_(["high", "critical"]) \
    .and_("properties.effort").lt(8) \
    .execute()
```

## Performance Considerations

### When to Use Each Method

| Method | Best For | Performance |
|--------|----------|-------------|
| `query()` | Simple attribute matches | Fast (native CSS) |
| `query_builder()` | Complex conditions, aggregations | Fast (optimized filtering) |
| `find()` | Single result lookups | Fast (early termination) |
| `find_all()` | Multiple results with filters | Fast (direct filtering) |

### EdgeIndex Benefits

The new EdgeIndex provides O(1) reverse edge lookups:

```python
# Before: O(V*E) - scanning all nodes and edges
def old_get_dependents(graph, node_id):
    result = []
    for node in graph.get_nodes():
        for edge in node.edges.get("blocked_by", []):
            if edge.target_id == node_id:
                result.append(node)
    return result

# After: O(1) - direct index lookup
dependents = graph.get_incoming_edges(node_id, "blocked_by")
```

## Summary

1. **Keep using CSS selectors** for simple attribute queries
2. **Use QueryBuilder** when you need OR, NOT, numeric comparisons, or text search
3. **Use Find API** for readable, Django-style queries
4. **Use EdgeIndex** for efficient reverse edge lookups
5. **Use graph traversal methods** for ancestors, descendants, and path finding

All methods are interoperable - use whichever fits your use case best.
