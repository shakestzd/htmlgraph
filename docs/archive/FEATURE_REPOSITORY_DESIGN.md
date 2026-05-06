# FeatureRepository Interface Design

## Overview

The FeatureRepository interface unifies all data access patterns for Features in Wipnote. Currently, Features are accessed through three separate code paths:

1. **SDK Collections** (`sdk/collections/features.py`)
2. **CLI Work Commands** (`cli/work/snapshot.py`)
3. **Collections Base** (`collections/base.py`)

Each implements similar logic independently:
- Filtering (status, priority, track_id, agent_assigned)
- Querying (get, list, where)
- Caching behavior
- Batch operations

This design document describes the unified interface that will eliminate duplication.

## Design Goals

1. **Single Source of Truth**: One interface, multiple implementations
2. **Backward Compatible**: Existing SDK/CLI code continues working
3. **Testable**: Compliance tests ensure all implementations work correctly
4. **Extensible**: Easy to add new storage backends (currently HTML + SQLite)
5. **Performance**: Efficient caching, batch operations, early termination

## Interface Architecture

```
FeatureRepository (Abstract Interface)
├── Read Operations
│   ├── get(id)              - Single feature by ID
│   ├── list(filters)        - All features with optional filters
│   ├── where(**kwargs)      - Query builder with chaining
│   ├── by_track(id)         - Features in track
│   ├── by_status(status)    - Filter by status
│   ├── by_priority(priority)- Filter by priority
│   ├── by_assigned_to(agent)- Filter by agent
│   ├── batch_get(ids)       - Bulk retrieve
│   └── filter(predicate)    - Custom filter function
│
├── Write Operations
│   ├── create(title, **kwargs)   - Create new feature
│   ├── save(feature)             - Save/update feature
│   ├── batch_update(ids, updates)- Bulk update
│   ├── delete(id)                - Delete feature
│   └── batch_delete(ids)         - Bulk delete
│
├── Advanced Queries
│   ├── find_dependencies(id) - Transitive dependencies
│   ├── find_blocking(id)     - Features blocked by this one
│   └── filter(predicate)     - Custom predicates
│
└── Cache Management
    ├── invalidate_cache(id)  - Clear cache
    ├── reload()              - Force reload from storage
    └── auto_load property    - Control auto-loading
```

## Key Design Decisions

### 1. Identity Caching (Object Instance Reuse)

**Decision**: `get(id)` returns the same Python object instance for the same feature across multiple calls.

```python
f1 = repo.get("feat-001")
f2 = repo.get("feat-001")
assert f1 is f2  # Same instance (identity), not just equality
```

**Rationale**:
- Matches current SDK behavior (single graph instance)
- Prevents stale copies in memory
- Simplifies consistency (one object = one source of truth)
- Enables weak references for memory efficiency

**Implementation Pattern**:
```python
def get(self, feature_id: str):
    if feature_id in self._cache:
        return self._cache[feature_id]
    feature = self._load_from_storage(feature_id)
    if feature:
        self._cache[feature_id] = feature
    return feature
```

### 2. List Returns Empty List, Never None

**Decision**: `list()` and `list(filters)` always return a list, never None.

```python
result = repo.list()  # Returns [], not None
assert isinstance(result, list)
```

**Rationale**:
- Safer API (no need to check for None)
- Consistent with Python conventions (dict.values(), etc.)
- Allows chaining: `for item in repo.list(): ...`

### 3. Where/Query Builder Pattern

**Decision**: `where()` returns a RepositoryQuery object supporting method chaining.

```python
repo.where(status="todo").where(priority="high").execute()
```

**Rationale**:
- Composable queries without complex DSL
- Lazy evaluation (filters build up before execute)
- Extensible for future optimizations
- Matches modern Python patterns (SQLAlchemy-like)

### 4. Batch Operations for Efficiency

**Decision**: Separate batch methods for vectorized operations.

```python
# More efficient than N individual saves
repo.batch_update(["f1", "f2", "f3"], {"status": "done"})
```

**Rationale**:
- Fewer function call overhead
- Database can optimize (single transaction)
- Explicit intent (not hidden optimization)
- Returns count for progress tracking

### 5. Exception Types for Error Handling

**Decision**: Defined exception hierarchy for different error scenarios.

```python
try:
    repo.find_dependencies("feat-invalid")
except FeatureNotFoundError as e:
    print(f"Feature not found: {e.feature_id}")
except FeatureValidationError as e:
    print(f"Invalid data: {e}")
```

**Exception Types**:
- `FeatureRepositoryException` - Base exception
- `FeatureNotFoundError` - Feature doesn't exist
- `FeatureValidationError` - Invalid data
- `FeatureConcurrencyError` - Concurrent modification

### 6. Lazy Loading with Auto-Load Control

**Decision**: Features loaded on-demand, with control over auto-loading.

```python
repo.auto_load = True   # Load on first access
repo.auto_load = False  # Require explicit reload()
```

**Rationale**:
- Faster startup (don't load all features at once)
- Control for memory-constrained environments
- Explicit reload for external changes

## Contract Invariants

Every FeatureRepository implementation MUST maintain:

### Invariant 1: Identity
```python
# Same object instance for same feature
f1 = repo.get("feat-001")
f2 = repo.get("feat-001")
assert f1 is f2  # Identity (is), not equality (==)
```

### Invariant 2: Consistency
```python
# Modified object stays in sync with storage
feature = repo.get("feat-001")
feature.status = "done"
repo.save(feature)
retrieved = repo.get("feat-001")
assert retrieved.status == "done"
assert retrieved is feature  # Still same instance
```

### Invariant 3: Atomicity
```python
# Updates are atomic (all or nothing)
# No partial updates on failure
repo.batch_update(["f1", "f2"], {"status": "done"})
# Either both updated or neither
```

### Invariant 4: Filtering Correctness
```python
# Filters applied with AND semantics
results = repo.list({"status": "todo", "priority": "high"})
assert all(f.status == "todo" and f.priority == "high" for f in results)
```

### Invariant 5: Cache Invalidation
```python
# Cache can be invalidated
repo.invalidate_cache("feat-001")  # Single feature
repo.invalidate_cache()  # All features
repo.reload()  # Immediate reload
```

## Performance Characteristics

| Operation | Time | Space | Notes |
|-----------|------|-------|-------|
| get(id) | O(1) | O(1) | Cached instance lookup |
| list() | O(n) | O(n) | Full scan, n = total features |
| where().execute() | O(n) | O(k) | n scanned, k results |
| by_track(id) | O(n) | O(k) | Early termination on match |
| by_status(s) | O(n) | O(k) | Full scan, k = results |
| batch_get(k) | O(k) | O(k) | Vectorized, k = batch size |
| batch_update(k, m) | O(k) | O(1) | k = batch size, m = update fields |
| batch_delete(k) | O(k) | O(1) | k = batch size |
| find_dependencies() | O(n) | O(d) | Graph traversal, d = depth |
| count(filters) | O(n) | O(1) | Full scan or optimized SQL |
| exists(id) | O(1) | O(1) | Index lookup |

## Compliance Tests

15+ compliance tests ensure all implementations behave correctly:

### Identity Tests
- `test_get_returns_same_instance` - Identity invariant
- `test_get_nonexistent_returns_none` - None for missing
- `test_get_with_invalid_id_format` - ValueError on bad ID

### List Tests
- `test_list_with_no_filters_returns_all` - No filters = all
- `test_list_returns_empty_list_not_none` - Never None
- `test_list_with_single_filter` - Single condition
- `test_list_with_multiple_filters` - Multiple conditions (AND)
- `test_list_preserves_object_identity` - Returns cached instances

### Query Builder Tests
- `test_where_returns_query_object` - Returns RepositoryQuery
- `test_where_chaining` - Supports method chaining
- `test_where_with_invalid_attribute` - Raises ValidationError

### Filtered Access Tests
- `test_by_track` - Filter by track
- `test_by_status` - Filter by status
- `test_by_priority` - Filter by priority
- `test_by_assigned_to` - Filter by agent

### Batch Operation Tests
- `test_batch_get` - Bulk retrieve
- `test_batch_update` - Bulk update
- `test_batch_delete` - Bulk delete
- `test_batch_get_invalid_input` - ValueError on bad input

### Write Operation Tests
- `test_create_generates_id` - Auto-generate ID
- `test_create_returns_saved_instance` - Immediately retrievable
- `test_save_updates_existing` - Update preserves state
- `test_save_preserves_identity` - Returns same instance

### Delete Tests
- `test_delete_removes_feature` - Feature gone after delete
- `test_delete_nonexistent_returns_false` - False if not found

### Advanced Query Tests
- `test_find_dependencies` - Transitive dependency resolution
- `test_find_blocking` - Reverse dependency resolution
- `test_filter_with_predicate` - Custom predicates

### Cache Management Tests
- `test_invalidate_single_feature_cache` - Single feature invalidation
- `test_invalidate_all_caches` - All features invalidation
- `test_reload` - Force reload
- `test_auto_load_property` - Control auto-loading

### Utility Tests
- `test_count` - Feature count
- `test_count_with_filter` - Filtered count
- `test_exists` - Existence check

### Error Handling Tests
- `test_validation_error_on_invalid_data` - Invalid data raises error
- `test_not_found_error_dependency_query` - Missing feature raises error

### Concurrency Tests
- `test_concurrent_safe_reads` - Multiple concurrent reads
- `test_concurrent_safe_writes` - Sequential write safety

## Implementation Roadmap

### Phase 1: Interface Definition (CURRENT)
- ✅ Abstract FeatureRepository interface
- ✅ 15+ compliance tests
- ✅ Exception types
- ✅ Documentation

### Phase 2: Current Implementation Adapters (NEXT)
Wrap existing implementations to implement interface:
- Adapt `collections/base.py` → FeatureRepositoryImpl
- Adapt `sdk/collections/features.py` → SDKFeatureRepositoryImpl
- Update CLI to use repository interface

### Phase 3: Unified Implementation (LATER)
Create optimized implementation combining HTML + SQLite:
- Single get() logic, both storage backends
- Unified caching strategy
- Unified query building

### Phase 4: Backward Compatibility (CONTINUOUS)
- SDK collections still work (delegate to repository)
- CLI still works (uses repository)
- All existing tests pass

## Example Usage

### Using the Repository (Once Implemented)

```python
from wipnote import SDK

sdk = SDK(agent="claude")

# repo implements FeatureRepository
repo = sdk.features

# Get single feature
feature = repo.get("feat-001")
if feature is None:
    print("Not found")

# List all
all_features = repo.list()

# List with filters
todo_features = repo.list({"status": "todo"})

# Complex query (method chaining)
critical_todo = repo.where(status="todo") \
    .where(priority="critical") \
    .execute()

# Batch operations
repo.batch_update(
    ["feat-1", "feat-2", "feat-3"],
    {"status": "done", "priority": "low"}
)

# Advanced queries
deps = repo.find_dependencies("feat-auth")
blocked = repo.find_blocking("feat-db-migration")

# Cache management
repo.invalidate_cache("feat-001")
repo.reload()

# Utility methods
total = repo.count()
todo_count = repo.count({"status": "todo"})
exists = repo.exists("feat-001")
```

### Implementing the Interface

```python
from wipnote.repositories import FeatureRepository

class GraphFeatureRepository(FeatureRepository):
    def __init__(self, sdk):
        self._sdk = sdk
        self._graph = None
        self._cache = {}
        self._auto_load = True

    def get(self, feature_id: str):
        if feature_id in self._cache:
            return self._cache[feature_id]
        graph = self._ensure_graph()
        feature = graph.get(feature_id)
        if feature:
            self._cache[feature_id] = feature
        return feature

    def list(self, filters=None):
        graph = self._ensure_graph()
        matches = []
        for node in graph:
            if node.type != "feature":
                continue
            if self._matches_filters(node, filters):
                matches.append(node)
                self._cache[node.id] = node  # Cache as we go
        return matches

    # ... implement all abstract methods ...
```

## Testing

```bash
# Run compliance tests
pytest tests/unit/repositories/test_feature_repository_compliance.py -v

# Test concrete implementation
pytest tests/unit/repositories/test_graph_feature_repository.py -v

# Full test suite
pytest tests/
```

## See Also

- `src/python/wipnote/repositories/feature_repository.py` - Interface
- `tests/unit/repositories/test_feature_repository_compliance.py` - Compliance tests
- `src/python/wipnote/collections/base.py` - Current implementation to adapt
- `AGENTS.md` - SDK usage documentation

## Questions?

Review the interface documentation in `feature_repository.py` for:
- Full method signatures with types
- Detailed docstrings with examples
- Performance characteristics
- Error conditions
- Contract invariants
