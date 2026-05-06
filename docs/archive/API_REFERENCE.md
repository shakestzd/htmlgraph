# Wipnote SDK API Reference

Complete reference for the Wipnote Python SDK.

## Table of Contents

1. [SDK Initialization](#sdk-initialization)
2. [Collections API](#collections-api)
3. [Collection Methods](#collection-methods)
4. [Builder API](#builder-api)
5. [Analytics API](#analytics-api)
6. [Session Management](#session-management)
7. [Query Methods](#query-methods)
8. [Error Handling](#error-handling)
9. [Common Mistakes](#common-mistakes)

---

## SDK Initialization

### Basic Usage

```python
from wipnote import SDK

# Auto-discovers .wipnote directory
sdk = SDK(agent='my-agent')

# Explicit directory
sdk = SDK(directory='/path/to/.wipnote', agent='my-agent')

# Auto-detect agent name (Claude, Gemini, etc.)
sdk = SDK()  # Detects agent automatically
```

### Constructor Parameters

**Signature**: `SDK(directory=None, agent=None)`

**Parameters**:
- `directory` (optional): Path to .wipnote directory. Auto-discovers if not provided.
- `agent` (optional): Agent identifier. Auto-detects if not provided.

**Auto-Discovery**:
- Searches current directory and parents for `.wipnote/`
- Defaults to current directory if not found

---

## Collections API

### Available Collections

The SDK provides collections for different node types:

```python
sdk.features      # FeatureCollection - Feature work items
sdk.bugs          # BugCollection - Bug reports
sdk.chores        # ChoreCollection - Maintenance tasks
sdk.spikes        # SpikeCollection - Investigation spikes
sdk.epics         # EpicCollection - Large initiatives
sdk.phases        # PhaseCollection - Project phases
sdk.tracks        # TrackCollection - Work tracks (with builder)
sdk.sessions      # BaseCollection - Agent sessions
sdk.agents        # BaseCollection - Agent information
sdk.todos         # TodoCollection - Persistent tasks
sdk.patterns      # PatternCollection - Workflow patterns
sdk.insights      # InsightCollection - Session health insights
sdk.metrics       # MetricCollection - Time-series metrics
```

### Collection Characteristics

**With Builder Support** (fluent API):
- `sdk.features` - FeatureBuilder
- `sdk.bugs` - BugBuilder
- `sdk.chores` - ChoreBuilder
- `sdk.spikes` - SpikeBuilder
- `sdk.epics` - EpicBuilder
- `sdk.phases` - PhaseBuilder
- `sdk.tracks` - TrackBuilder
- `sdk.patterns` - PatternBuilder
- `sdk.insights` - InsightBuilder
- `sdk.metrics` - MetricBuilder

**Without Builder** (simple creation):
- `sdk.sessions`
- `sdk.agents`
- `sdk.todos`

---

## Collection Methods

All collections inherit from `BaseCollection` and provide these methods:

### Retrieval Methods

#### all()

Get all nodes in the collection.

**Signature**: `all() -> list[Node]`

**Returns**: List of all nodes of this type

**Example**:
```python
features = sdk.features.all()
bugs = sdk.bugs.all()
```

---

#### get()

Get a single node by ID.

**Signature**: `get(node_id: str) -> Node | None`

**Parameters**:
- `node_id` (required): Node ID to retrieve

**Returns**: Node if found, None otherwise

**Example**:
```python
feature = sdk.features.get('feat-abc123')
if feature:
    print(feature.title)
else:
    print("Not found")
```

---

#### where()

Query nodes with filters.

**Signature**: `where(status=None, priority=None, track=None, assigned_to=None, **extra_filters) -> list[Node]`

**Parameters**:
- `status` (optional): Filter by status ("todo", "in-progress", "blocked", "done")
- `priority` (optional): Filter by priority ("low", "medium", "high", "critical")
- `track` (optional): Filter by track_id
- `assigned_to` (optional): Filter by agent_assigned
- `**extra_filters`: Additional attribute filters

**Returns**: List of matching nodes (empty list if none match)

**Example**:
```python
# High priority todos
high_priority = sdk.features.where(priority='high', status='todo')

# Assigned to specific agent
my_work = sdk.features.where(assigned_to='claude')

# Multiple filters
urgent = sdk.bugs.where(priority='critical', status='todo', assigned_to='claude')

# Custom attribute filters
auth_features = sdk.features.where(track='auth', status='in-progress')
```

---

#### filter()

Filter nodes using a custom predicate function.

**Signature**: `filter(predicate: Callable[[Node], bool]) -> list[Node]`

**Parameters**:
- `predicate` (required): Function that takes a Node and returns True if it matches

**Returns**: List of matching nodes

**Example**:
```python
# Find features with "High" in title
high_priority = sdk.features.filter(lambda f: "High" in f.title)

# Find features created in the last week
from datetime import datetime, timedelta
recent = sdk.features.filter(
    lambda f: f.created > datetime.now() - timedelta(days=7)
)

# Complex multi-condition filter
urgent = sdk.features.filter(
    lambda f: f.priority == "high" and f.status == "todo" and len(f.steps) > 5
)
```

---

### Creation Methods

#### create()

Create a new node in this collection.

**Signature**: `create(title: str, priority='medium', status='todo', **kwargs) -> Builder | Node`

**Parameters**:
- `title` (required): Node title
- `priority` (optional): Priority level ("low", "medium", "high", "critical")
- `status` (optional): Status ("todo", "in-progress", "blocked", "done")
- `**kwargs`: Additional node properties

**Returns**:
- Builder instance if collection has builder support (e.g., FeatureBuilder)
- Node instance if collection doesn't have builder support

**Example**:
```python
# With builder (requires .save())
feature = sdk.features.create("User Authentication") \
    .set_priority("high") \
    .add_steps(["Create login endpoint", "Add JWT middleware"]) \
    .save()

# Without builder (auto-saved)
session = sdk.sessions.create("Session 1", status='active')
```

---

### Editing Methods

#### edit()

Context manager for editing a node. Auto-saves on exit.

**Signature**: `edit(node_id: str) -> Iterator[Node]`

**Parameters**:
- `node_id` (required): Node ID to edit

**Returns**: Context manager yielding the node

**Raises**: `NodeNotFoundError` if node not found

**Example**:
```python
# Edit a single node
with sdk.features.edit('feat-abc123') as feature:
    feature.status = 'in-progress'
    feature.priority = 'high'
    feature.steps.append(Step(description='New step'))
# Auto-saved on exit

# Handle missing nodes
from wipnote.exceptions import NodeNotFoundError
try:
    with sdk.features.edit('nonexistent') as feature:
        feature.status = 'done'
except NodeNotFoundError:
    print("Feature not found")
```

---

#### update()

Update a node directly.

**Signature**: `update(node: Node) -> Node`

**Parameters**:
- `node` (required): Node to update

**Returns**: Updated node

**Raises**: `NodeNotFoundError` if node doesn't exist

**Example**:
```python
feature = sdk.features.get('feat-abc123')
feature.status = 'done'
feature.priority = 'high'
sdk.features.update(feature)
```

---

#### batch_update()

Batch update multiple nodes with the same changes.

**Signature**: `batch_update(node_ids: list[str], updates: dict[str, Any]) -> int`

**Parameters**:
- `node_ids` (required): List of node IDs to update
- `updates` (required): Dictionary of attribute: value pairs

**Returns**: Number of nodes successfully updated

**Example**:
```python
# Update multiple nodes at once
count = sdk.features.batch_update(
    ['feat-1', 'feat-2', 'feat-3'],
    {'status': 'done', 'agent_assigned': 'claude'}
)
print(f"Updated {count} features")
```

---

### Work Management Methods

#### start()

Start working on a node.

**Signature**: `start(node_id: str, agent=None) -> Node | None`

**Parameters**:
- `node_id` (required): Node ID to start
- `agent` (optional): Agent ID (defaults to SDK agent)

**Returns**: Updated node, or None if not found

**Behavior**:
- Sets status to "in-progress"
- Auto-claims for agent
- Links to active session (if SessionManager available)
- Logs 'FeatureStart' event
- Checks WIP limits

**Raises**: `NodeNotFoundError` if node not found

**Example**:
```python
feature = sdk.features.start('feat-abc123')
# Or with explicit agent
feature = sdk.features.start('feat-abc123', agent='claude')
```

---

#### complete()

Mark a node as complete.

**Signature**: `complete(node_id: str, agent=None, transcript_id=None) -> Node | None`

**Parameters**:
- `node_id` (required): Node ID to complete
- `agent` (optional): Agent ID (defaults to SDK agent)
- `transcript_id` (optional): Transcript ID for parallel agent tracking

**Returns**: Updated node, or None if not found

**Behavior**:
- Sets status to "done"
- Logs 'FeatureComplete' event
- Optionally releases claim
- Links transcript if provided

**Raises**: `NodeNotFoundError` if node not found

**Example**:
```python
# Complete a feature
sdk.features.complete('feat-abc123')

# With explicit agent
sdk.features.complete('feat-abc123', agent='claude')

# With transcript tracking
sdk.features.complete('feat-abc123', transcript_id='sess-xyz')
```

---

#### claim()

Claim a node for an agent.

**Signature**: `claim(node_id: str, agent=None) -> Node | None`

**Parameters**:
- `node_id` (required): Node ID to claim
- `agent` (optional): Agent ID (defaults to SDK agent)

**Returns**: Claimed node, or None if not found

**Raises**:
- `ValueError` if agent not provided and SDK has no agent
- `NodeNotFoundError` if node not found
- `ClaimConflictError` if already claimed by different agent

**Example**:
```python
# Claim a feature
feature = sdk.features.claim('feat-abc123')

# Handle conflicts
from wipnote.exceptions import ClaimConflictError
try:
    feature = sdk.features.claim('feat-abc123')
except ClaimConflictError as e:
    print(f"Already claimed by {e.claimed_by}")
```

---

#### release()

Release a claimed node.

**Signature**: `release(node_id: str, agent=None) -> Node | None`

**Parameters**:
- `node_id` (required): Node ID to release
- `agent` (optional): Agent ID (defaults to SDK agent)

**Returns**: Released node, or None if not found

**Behavior**:
- Clears agent_assigned
- Clears claimed_at and claimed_by_session
- Sets status back to "todo"
- Logs 'FeatureRelease' event

**Raises**: `NodeNotFoundError` if node not found

**Example**:
```python
sdk.features.release('feat-abc123')
```

---

### Batch Operations

#### mark_done()

Batch mark nodes as done.

**Signature**: `mark_done(node_ids: list[str]) -> dict[str, Any]`

**Parameters**:
- `node_ids` (required): List of node IDs to mark as done

**Returns**: Dict with:
- `success_count`: Number of nodes successfully completed
- `failed_ids`: List of node IDs that failed
- `warnings`: List of warning messages

**Example**:
```python
result = sdk.features.mark_done(['feat-001', 'feat-002', 'feat-003'])
print(f"Completed {result['success_count']} of 3")

if result['failed_ids']:
    print(f"Failed: {result['failed_ids']}")
    print(f"Reasons: {result['warnings']}")
```

---

#### assign()

Batch assign nodes to an agent.

**Signature**: `assign(node_ids: list[str], agent: str) -> int`

**Parameters**:
- `node_ids` (required): List of node IDs to assign
- `agent` (required): Agent ID to assign to

**Returns**: Number of nodes assigned

**Example**:
```python
count = sdk.features.assign(['feat-001', 'feat-002'], 'claude')
print(f"Assigned {count} features to claude")
```

---

### Deletion Methods

#### delete()

Delete a single node.

**Signature**: `delete(node_id: str) -> bool`

**Parameters**:
- `node_id` (required): Node ID to delete

**Returns**: True if deleted, False if not found

**Example**:
```python
success = sdk.features.delete('feat-abc123')
if success:
    print("Deleted")
else:
    print("Not found")
```

---

#### batch_delete()

Delete multiple nodes in batch.

**Signature**: `batch_delete(node_ids: list[str]) -> int`

**Parameters**:
- `node_ids` (required): List of node IDs to delete

**Returns**: Number of nodes successfully deleted

**Example**:
```python
count = sdk.features.batch_delete(['feat-001', 'feat-002', 'feat-003'])
print(f"Deleted {count} features")
```

---

## Builder API

Collections with builder support return builder instances from `create()`.

### Fluent Methods

All builders provide method chaining:

```python
feature = sdk.features.create("User Auth") \
    .set_priority("high") \
    .set_status("in-progress") \
    .add_steps(["Login", "Logout", "JWT middleware"]) \
    .add_edge("blocks", "feat-other", "Needs auth") \
    .save()
```

### Common Builder Methods

#### set_priority()

**Signature**: `set_priority(priority: str) -> Self`

**Parameters**: Priority level ("low", "medium", "high", "critical")

**Returns**: Builder instance (for chaining)

---

#### set_status()

**Signature**: `set_status(status: str) -> Self`

**Parameters**: Status ("todo", "in-progress", "blocked", "done")

**Returns**: Builder instance (for chaining)

---

#### add_steps()

**Signature**: `add_steps(steps: list[str]) -> Self`

**Parameters**: List of step descriptions

**Returns**: Builder instance (for chaining)

---

#### add_edge()

**Signature**: `add_edge(relationship: str, target_id: str, title: str = None) -> Self`

**Parameters**:
- `relationship`: Relationship type ("blocks", "depends-on", "related", etc.)
- `target_id`: Target node ID
- `title` (optional): Link title

**Returns**: Builder instance (for chaining)

---

#### save()

**Signature**: `save() -> Node`

**Returns**: Created/updated node

**Note**: MUST be called to persist the node.

---

### Builder-Specific Methods

Different builders provide specialized methods:

#### FeatureBuilder

```python
feature = sdk.features.create("User Auth") \
    .set_priority("high") \
    .add_steps(["Login endpoint", "JWT middleware"]) \
    .add_acceptance_criteria("Must support OAuth") \
    .save()
```

#### BugBuilder

```python
bug = sdk.bugs.create("Login failure") \
    .set_priority("critical") \
    .set_severity("high") \
    .add_reproduction_steps(["Navigate to /login", "Enter credentials", "Click submit"]) \
    .save()
```

#### SpikeBuilder

```python
spike = sdk.spikes.create("Investigate auth libraries") \
    .set_spike_type("technical") \
    .add_questions(["Which library is best?", "Performance comparison?"]) \
    .save()
```

#### TrackBuilder

```python
track = sdk.tracks.create("Authentication Track") \
    .set_priority("high") \
    .add_features(["feat-001", "feat-002"]) \
    .save()
```

---

## Analytics API

### Dependency Analytics

Access via `sdk.dep_analytics`:

#### find_bottlenecks()

Find tasks blocking the most work.

**Signature**: `find_bottlenecks(top_n=5) -> list[BottleneckDict]`

**Example**:
```python
bottlenecks = sdk.dep_analytics.find_bottlenecks(top_n=3)
for task in bottlenecks:
    print(f"{task['node_id']}: blocks {task['blocks_count']} tasks")
```

---

#### get_parallel_work()

Find tasks that can be worked on in parallel.

**Signature**: `get_parallel_work(max_agents=5) -> list[ParallelWorkInfo]`

**Example**:
```python
parallel = sdk.dep_analytics.get_parallel_work(max_agents=3)
for work in parallel:
    print(f"{work['node_id']}: {work['title']}")
```

---

#### recommend_next_tasks()

Smart task recommendations based on dependencies.

**Signature**: `recommend_next_tasks(agent_count=1) -> list[WorkRecommendation]`

**Example**:
```python
recommendations = sdk.dep_analytics.recommend_next_tasks(agent_count=2)
for rec in recommendations:
    print(f"{rec['node_id']}: {rec['reason']}")
```

---

### Work Analytics

Access via `sdk.analytics`:

#### get_work_type_distribution()

Breakdown of work by type.

**Signature**: `get_work_type_distribution() -> dict[str, int]`

**Example**:
```python
distribution = sdk.analytics.get_work_type_distribution()
# {'feature': 10, 'bug': 5, 'spike': 3, 'chore': 2}
```

---

#### get_spike_to_feature_ratio()

Ratio of investigation to implementation work.

**Signature**: `get_spike_to_feature_ratio() -> float`

**Example**:
```python
ratio = sdk.analytics.get_spike_to_feature_ratio()
print(f"Investigation ratio: {ratio:.2f}")
```

---

### Context Analytics

Access via `sdk.context`:

#### get_context_usage()

Get context usage metrics for a session or feature.

**Signature**: `get_context_usage(session_id=None, feature_id=None) -> ContextUsage | None`

**Example**:
```python
usage = sdk.context.get_context_usage(session_id='sess-abc')
if usage:
    print(f"Tokens used: {usage.tokens_used}")
    print(f"Cost: ${usage.cost_usd:.4f}")
```

---

#### get_context_efficiency()

Calculate context efficiency score.

**Signature**: `get_context_efficiency() -> float`

**Example**:
```python
efficiency = sdk.context.get_context_efficiency()
print(f"Efficiency: {efficiency:.2%}")
```

---

## Session Management

### start_session()

Start a new session.

**Signature**: `start_session(session_id=None, title=None, agent=None) -> Session`

**Example**:
```python
session = sdk.start_session(title="Authentication work")
```

---

### end_session()

End a session.

**Signature**: `end_session(session_id: str, handoff_notes=None, recommended_next=None, blockers=None) -> Session`

**Example**:
```python
sdk.end_session(
    session_id='sess-abc',
    handoff_notes="Completed login, logout remaining",
    recommended_next="Implement logout endpoint",
    blockers=["Need JWT library decision"]
)
```

---

### set_session_handoff()

Set handoff context on active session.

**Signature**: `set_session_handoff(handoff_notes=None, recommended_next=None, blockers=None, session_id=None) -> Session | None`

**Example**:
```python
sdk.set_session_handoff(
    handoff_notes="Auth partially done",
    recommended_next="Complete JWT middleware"
)
```

---

## Query Methods

### Advanced Querying

Use `QueryBuilder` for complex queries:

```python
from wipnote import QueryBuilder

# Build complex query
query = QueryBuilder() \
    .where("status", "=", "todo") \
    .where("priority", "in", ["high", "critical"]) \
    .where("created", ">", datetime.now() - timedelta(days=7)) \
    .build()

# Execute
results = sdk.features._ensure_graph().query(query)
```

---

## Error Handling

### Error Handling Patterns

SDK methods follow consistent error handling by operation type:

| Operation Type | Error Behavior | Example Methods |
|---------------|----------------|-----------------|
| Lookup | Return None | `get(id)` |
| Query | Return [] | `where()`, `all()`, `filter()` |
| Edit | Raise Exception | `edit(id)` |
| Create | Raise on Invalid | `create(title)` |
| Batch | Return Results Dict | `mark_done([ids])` |
| Delete | Return Bool | `delete(id)` |

### Available Exceptions

```python
from wipnote.exceptions import (
    NodeNotFoundError,      # Node with ID not found
    ValidationError,        # Invalid input parameters
    ClaimConflictError,     # Node already claimed by another agent
)
```

### Error Handling Examples

#### Lookup Operations (Return None)

```python
feature = sdk.features.get("nonexistent")
if feature:
    print(feature.title)
else:
    print("Not found")
```

#### Query Operations (Return Empty List)

```python
results = sdk.features.where(status="impossible")
for r in results:  # Empty iteration is safe
    print(r.title)
```

#### Edit Operations (Raise Exception)

```python
from wipnote.exceptions import NodeNotFoundError
try:
    with sdk.features.edit("nonexistent") as f:
        f.status = "done"
except NodeNotFoundError:
    print("Feature not found")
```

#### Create Operations (Raise on Validation)

```python
from wipnote.exceptions import ValidationError
try:
    sdk.features.create("")  # Empty title
except ValidationError:
    print("Title required")
```

#### Batch Operations (Return Results Dict)

```python
result = sdk.features.mark_done(["feat-1", "missing", "feat-2"])
print(f"Completed {result['success_count']} of 3")
if result['failed_ids']:
    print(f"Failed: {result['failed_ids']}")
    print(f"Reasons: {result['warnings']}")
```

---

## Common Mistakes

### Using .list() instead of .all()

```python
# WRONG
features = sdk.features.list()  # AttributeError

# CORRECT
features = sdk.features.all()
```

---

### Calling complete() on Node instance

```python
# WRONG
feature = sdk.features.get('feat-123')
feature.complete()  # AttributeError

# CORRECT - Call on collection
sdk.features.complete('feat-123')
```

---

### Forgetting .save() on Builder

```python
# WRONG - Node not persisted
feature = sdk.features.create("User Auth") \
    .set_priority("high") \
    .add_steps(["Login", "Logout"])
# Feature not saved!

# CORRECT - Must call .save()
feature = sdk.features.create("User Auth") \
    .set_priority("high") \
    .add_steps(["Login", "Logout"]) \
    .save()
```

---

### Using to_dict() on older versions

```python
# WRONG (before v0.24)
data = node.to_dict()  # AttributeError

# CORRECT (all versions)
data = node.model_dump()

# CORRECT (v0.24+)
data = node.to_dict()  # Now available as alias
```

---

### Not checking return values

```python
# RISKY - node could be None
node = sdk.features.get('feat-123')
print(node.title)  # AttributeError if not found

# SAFE - Check first
node = sdk.features.get('feat-123')
if node:
    print(node.title)
else:
    print("Not found")
```

---

### Assuming batch operations succeed

```python
# RISKY - Some IDs might fail
sdk.features.mark_done(['feat-1', 'feat-2', 'feat-3'])

# SAFE - Check results
result = sdk.features.mark_done(['feat-1', 'feat-2', 'feat-3'])
if result['failed_ids']:
    print(f"Failed: {result['failed_ids']}")
    for warning in result['warnings']:
        print(f"  {warning}")
```

---

## Quick Reference Card

```python
# Initialize
sdk = SDK(agent='my-agent')

# Create with builder
feature = sdk.features.create("Title") \
    .set_priority("high") \
    .add_steps(["Step 1", "Step 2"]) \
    .save()

# Query
todos = sdk.features.where(status='todo', priority='high')
all_bugs = sdk.bugs.all()
recent = sdk.features.filter(lambda f: f.created > cutoff)

# Edit
with sdk.features.edit('feat-123') as f:
    f.status = 'done'

# Work management
sdk.features.start('feat-123')
sdk.features.complete('feat-123')
sdk.features.claim('feat-123')
sdk.features.release('feat-123')

# Batch operations
result = sdk.features.mark_done(['feat-1', 'feat-2'])
count = sdk.features.assign(['feat-1', 'feat-2'], 'claude')

# Analytics
bottlenecks = sdk.dep_analytics.find_bottlenecks()
parallel = sdk.dep_analytics.get_parallel_work()
recommendations = sdk.dep_analytics.recommend_next_tasks()

# Error handling
from wipnote.exceptions import NodeNotFoundError, ClaimConflictError
try:
    with sdk.features.edit('feat-123') as f:
        f.status = 'done'
except NodeNotFoundError:
    print("Not found")
```

---

## See Also

- [AGENTS.md](../AGENTS.md) - Complete SDK documentation with examples
- [README.md](../README.md) - Project overview
- [examples/](../examples/) - Code examples
