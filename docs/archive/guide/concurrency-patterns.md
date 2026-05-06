# Concurrency Patterns for Wipnote

## Overview

Wipnote provides transaction and snapshot patterns for safe concurrent access to graph data. These patterns enable multiple agents to read and write the graph without data corruption or race conditions.

## GraphSnapshot: Immutable State Views

A `GraphSnapshot` is a frozen, read-only copy of the graph at a specific point in time. It remains unchanged even if the original graph is modified.

### Features

- **Immutable**: Once created, the snapshot never changes
- **Deep Copy**: All nodes are deep-copied to prevent mutation
- **Full Query Support**: Supports all read operations (get, query, filter, iteration)
- **Lightweight**: Only copies node data, not file I/O structures

### Usage

```python
from wipnote import Wipnote

graph = Wipnote("features/")

# Create snapshot
snapshot = graph.snapshot()

# Read from snapshot (safe even while graph is being modified)
node = snapshot.get("feature-001")
results = snapshot.query("[data-status='blocked']")
todos = snapshot.filter(lambda n: n.status == "todo")

# Iterate over snapshot
for node in snapshot:
    print(node.title)

# Check membership
if "feature-001" in snapshot:
    print("Found!")
```

### Use Cases

#### 1. Concurrent Read Operations

Multiple agents can safely read from snapshots while other agents modify the graph:

```python
# Agent 1: Take snapshot for analysis
snapshot = graph.snapshot()
bottlenecks = analyze_bottlenecks(snapshot)  # Long-running operation

# Agent 2: Modify graph (won't affect Agent 1's snapshot)
node = graph.get("feature-001")
node.status = "done"
graph.update(node)
```

#### 2. Before/After Comparison

Compare graph state at different points in time:

```python
before = graph.snapshot()

# Make changes
with graph.transaction() as tx:
    for node_id in ["feat-001", "feat-002", "feat-003"]:
        node = graph.get(node_id)
        node.status = "done"
        tx.update(node)

after = graph.snapshot()

# Compare
before_done = len(before.filter(lambda n: n.status == "done"))
after_done = len(after.filter(lambda n: n.status == "done"))
print(f"Completed {after_done - before_done} features")
```

#### 3. Safe Reporting

Generate reports without worrying about concurrent modifications:

```python
def generate_report(graph):
    # Take snapshot for consistent view
    snapshot = graph.snapshot()

    # Generate report (may take several seconds)
    stats = calculate_statistics(snapshot)
    chart = create_visualization(snapshot)

    # Report reflects consistent state, even if graph was modified during generation
    return {"stats": stats, "chart": chart}
```

## Transactions: Atomic Multi-Operation Changes

A `transaction()` context manager batches multiple operations and applies them atomically. If any operation fails, all changes are rolled back.

### Features

- **Atomic**: All operations succeed or all fail
- **Automatic Rollback**: On exception, graph state is restored
- **Chainable**: Operations can be chained for fluent API
- **Supports All CRUD**: Add, update, delete operations

### Usage

```python
from wipnote import Wipnote
from wipnote.models import Node

graph = Wipnote("features/")

# Basic transaction
with graph.transaction() as tx:
    # Add new node
    node1 = Node(id="feat-001", title="New Feature", status="todo")
    tx.add(node1)

    # Update existing node
    node2 = graph.get("feat-002")
    node2.status = "done"
    tx.update(node2)

    # Delete node
    tx.delete("feat-003")

# All changes are committed atomically
```

### Rollback on Error

If any exception occurs within the transaction, no changes are persisted:

```python
try:
    with graph.transaction() as tx:
        # This will succeed
        tx.add(Node(id="feat-001", title="Feature 1", status="todo"))

        # This will fail (duplicate ID)
        tx.add(Node(id="feat-001", title="Duplicate", status="todo"))
except ValueError:
    pass

# No changes were applied - graph is unchanged
assert "feat-001" not in graph
```

### Chainable Operations

Operations can be chained for concise code:

```python
with graph.transaction() as tx:
    tx.add(node1).add(node2).update(node3).delete("feat-004")
```

### Use Cases

#### 1. Batch Updates

Update multiple related nodes atomically:

```python
def complete_feature_and_dependencies(graph, feature_id):
    """Complete a feature and all its dependencies."""
    feature = graph.get(feature_id)
    deps = graph.transitive_deps(feature_id)

    with graph.transaction() as tx:
        # Mark feature as done
        feature.status = "done"
        tx.update(feature)

        # Mark all dependencies as done
        for dep_id in deps:
            dep = graph.get(dep_id)
            dep.status = "done"
            tx.update(dep)
```

#### 2. Complex State Transitions

Ensure complex state changes are atomic:

```python
def start_work_on_feature(graph, feature_id, agent_id):
    """Start work on a feature (atomic state change)."""
    feature = graph.get(feature_id)

    # Check prerequisites
    deps = graph.transitive_deps(feature_id)
    if any(graph.get(dep_id).status != "done" for dep_id in deps):
        raise ValueError("Cannot start - dependencies not complete")

    with graph.transaction() as tx:
        # Update feature
        feature.status = "in-progress"
        feature.assigned_to = agent_id
        tx.update(feature)

        # Create session node
        session = Node(
            id=f"session-{feature_id}",
            type="session",
            title=f"Session for {feature.title}",
            status="active"
        )
        tx.add(session)

    # Both changes applied atomically, or neither if anything fails
```

#### 3. Migration/Refactoring

Safely migrate graph structure:

```python
def migrate_priority_values(graph):
    """Migrate priority from numbers to labels."""
    mapping = {1: "low", 2: "medium", 3: "high", 4: "critical"}

    with graph.transaction() as tx:
        for node in graph:
            if isinstance(node.priority, int):
                node.priority = mapping.get(node.priority, "medium")
                tx.update(node)

    # All nodes migrated atomically
```

## Combining Snapshot and Transaction

Snapshots and transactions work well together:

```python
# Take snapshot before risky operation
before = graph.snapshot()

try:
    with graph.transaction() as tx:
        # Perform complex changes
        perform_risky_migration(tx, graph)
except Exception as e:
    # Transaction auto-rollbacks, but we also have snapshot for analysis
    print(f"Migration failed: {e}")
    print(f"Graph state preserved (had {len(before)} nodes)")
    raise
```

## Performance Considerations

### Snapshots

- **Memory**: Creates deep copy of all nodes (proportional to graph size)
- **Time**: O(N) where N is number of nodes
- **Best For**: Small to medium graphs (< 10K nodes), or infrequent snapshots

### Transactions

- **Memory**: Minimal overhead (stores snapshot only during transaction)
- **Time**: O(operations) for commit, O(N) for rollback
- **Best For**: Batch operations, ensuring atomicity

### Recommendations

1. **For large graphs (10K+ nodes)**: Use snapshots sparingly, prefer transaction for safety
2. **For many concurrent readers**: Create snapshots at intervals (e.g., every 5 minutes)
3. **For frequent writes**: Use transactions to batch operations and reduce I/O
4. **For critical operations**: Always use transactions to ensure atomicity

## Example: Multi-Agent Coordination

```python
from wipnote import Wipnote
from wipnote.models import Node

graph = Wipnote("features/")

# Agent 1: Analyze bottlenecks (uses snapshot for stable view)
def agent1_analyze():
    snapshot = graph.snapshot()

    # Long-running analysis
    bottlenecks = []
    for node in snapshot:
        if len(snapshot.filter(lambda n: f"blocked_by:{node.id}" in str(n.edges))) > 5:
            bottlenecks.append(node.id)

    return bottlenecks

# Agent 2: Complete work atomically
def agent2_complete_work(feature_id):
    with graph.transaction() as tx:
        # Mark feature done
        feature = graph.get(feature_id)
        feature.status = "done"
        tx.update(feature)

        # Unblock dependents
        for dependent_id in graph.dependents(feature_id):
            dependent = graph.get(dependent_id)
            dependent.edges["blocked_by"] = [
                e for e in dependent.edges.get("blocked_by", [])
                if e.target_id != feature_id
            ]
            tx.update(dependent)

# Run concurrently (safe!)
import concurrent.futures

with concurrent.futures.ThreadPoolExecutor(max_workers=2) as executor:
    future1 = executor.submit(agent1_analyze)
    future2 = executor.submit(agent2_complete_work, "feat-001")

    bottlenecks = future1.result()
    future2.result()

    print(f"Found {len(bottlenecks)} bottlenecks while completing work")
```

## API Reference

### GraphSnapshot

```python
class GraphSnapshot:
    """Immutable snapshot of graph state."""

    def get(self, node_id: str) -> Node | None:
        """Get node by ID (returns copy)."""

    def query(self, selector: str) -> list[Node]:
        """Query with CSS selector (returns copies)."""

    def filter(self, predicate: Callable[[Node], bool]) -> list[Node]:
        """Filter with predicate (returns copies)."""

    def __len__(self) -> int:
        """Number of nodes in snapshot."""

    def __contains__(self, node_id: str) -> bool:
        """Check if node exists."""

    def __iter__(self) -> Iterator[Node]:
        """Iterate over nodes (returns copies)."""

    @property
    def nodes(self) -> dict[str, Node]:
        """All nodes as dict (returns copies)."""
```

### Wipnote Transaction Methods

```python
class Wipnote:
    def snapshot(self) -> GraphSnapshot:
        """Create immutable snapshot of current state."""

    @contextmanager
    def transaction(self):
        """
        Context manager for atomic operations.

        Yields TransactionContext with:
        - add(node, overwrite=False)
        - update(node)
        - delete(node_id)
        - remove(node_id)  # alias for delete
        """
```

## Testing

Comprehensive tests are available in `tests/test_transaction_snapshot.py`:

```bash
# Run transaction/snapshot tests
uv run pytest tests/test_transaction_snapshot.py -v

# Run all tests including concurrency scenarios
uv run pytest tests/test_transaction_snapshot.py::TestConcurrencyScenarios -v
```

## Implementation Notes

- **Snapshot immutability**: Enforced through deep copying with `model_copy(deep=True)`
- **Transaction isolation**: Not true ACID transactions (no file locking), but ensures memory consistency
- **Rollback mechanism**: Restores node dict, file hashes, and rebuilds indexes
- **Thread safety**: Not thread-safe across processes; use file locking for true multi-process safety

## Future Enhancements

Potential future improvements:

1. **File-level locking**: Add fcntl-based locking for true multi-process safety
2. **Optimistic locking**: ETags or version numbers to detect concurrent modifications
3. **Incremental snapshots**: Only copy modified nodes since last snapshot
4. **Transaction log**: Persist operations for replay/audit trail
5. **Read-write locks**: Allow multiple concurrent readers with exclusive writers
