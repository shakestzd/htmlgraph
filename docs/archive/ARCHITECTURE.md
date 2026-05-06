# Wipnote SDK Architecture

Design philosophy and architectural decisions behind the Wipnote SDK.

---

## Table of Contents

1. [Design Philosophy](#design-philosophy)
2. [Component Architecture](#component-architecture)
3. [Design Patterns](#design-patterns)
4. [Common Mistakes Explained](#common-mistakes-explained)
5. [Event-Driven State Management](#event-driven-state-management)
6. [Multi-Agent Coordination](#multi-agent-coordination)

---

## Design Philosophy

### Core Principles

Wipnote's SDK is built on four foundational principles:

1. **Immutability by Design** - Nodes are data models, not state machines
2. **Centralized State Management** - Collections manage state transitions
3. **Event-Driven Architecture** - All changes generate auditable events
4. **Separation of Concerns** - Clear boundaries between components

These principles emerged from real-world usage patterns with AI agents, where:
- Parallel execution is common (multiple agents working simultaneously)
- Observability is critical (what changed, who changed it, when)
- State consistency matters (no hidden mutations)
- Developer experience needs to be intuitive (familiar patterns from Django/Rails)

---

## Component Architecture

### Node - Pure Data Model

**Nodes** (Feature, Spike, Bug, Session, etc.) are **immutable data containers** based on Pydantic BaseModel.

#### Responsibilities

- Store node data (title, status, priority, steps, edges)
- Serialize to HTML and context formats
- Provide read-only properties (`completion_percentage`, `next_step`)
- **NO state mutations**

#### Why Immutable?

**Problem:** What if Node instances could mutate themselves?

```python
# ❌ ANTI-PATTERN - If nodes were mutable
feature = sdk.features.get('feat-123')
feature.status = "done"  # Silent change, no events, no audit trail
feature.save()           # When does this happen? Who knows?
```

**Issues with mutable nodes:**
- 🔴 **Hidden state changes** - No visibility into what changed
- 🔴 **No audit trails** - Can't track who made changes or when
- 🔴 **Thread-unsafe** - Race conditions with parallel agents
- 🔴 **Debugging nightmares** - "It changed but I don't know where"

**Solution:** Immutable nodes with centralized state management

```python
# ✅ CORRECT - Immutable nodes, state changes through Collections
feature = sdk.features.get('feat-123')

# Node is read-only - properties are transparent
print(feature.status)              # "in-progress"
print(feature.completion_percentage)  # 0.33

# State changes go through Collection (logged, auditable, atomic)
sdk.features.complete('feat-123')
# → Generates FeatureComplete event
# → Updates status, timestamp, completion
# → Logs who, when, which session
# → Thread-safe atomic operation
```

**Benefits:**
- ✅ **Predictable behavior** - Nodes never change unexpectedly
- ✅ **Thread-safe** - Safe for parallel agent access
- ✅ **Audit trails** - All mutations logged through Collections
- ✅ **Prevents bugs** - Can't accidentally modify without tracking
- ✅ **Clean separation** - Data model vs state machine

#### Example Usage

```python
from wipnote import SDK

sdk = SDK(agent='claude')

# Nodes are read-only data containers
feature = sdk.features.get('feat-123')
print(feature.id)        # feat-123
print(feature.status)    # in-progress
print(feature.priority)  # high

# ❌ WRONG - No instance methods for state changes
feature.complete()  # AttributeError: 'Node' has no attribute 'complete'
feature.start()     # AttributeError
feature.status = "done"  # Pydantic field is immutable

# ✅ CORRECT - Use Collection methods
sdk.features.start('feat-123')
sdk.features.complete('feat-123')
sdk.features.update('feat-123', priority='critical')
```

---

### Collection - State Management

**Collections** (FeatureCollection, SpikeCollection, etc.) manage state transitions and operations.

#### Responsibilities

- **CRUD operations** - create, get, update, delete
- **State transitions** - start, complete, claim, release
- **Event logging** - via SessionManager integration
- **Ownership tracking** - claims, assignments, WIP limits
- **Batch operations** - mark_done, assign, batch_update

#### Why Centralized?

**Problem:** What if state changes were decentralized?

```python
# ❌ ANTI-PATTERN - If state changes were scattered
feature.complete()          # Where's the event log?
feature.assign('agent-2')   # Where's the claiming logic?
feature.steps[0].complete() # Where's the validation?
```

**Issues with decentralized state:**
- 🔴 **No single source of truth** - Scattered state management
- 🔴 **Inconsistent events** - Some mutations logged, others not
- 🔴 **No validation** - Can't enforce WIP limits or claims
- 🔴 **Audit gaps** - Missing who/when/why information

**Solution:** Centralized state management through Collections

```python
# ✅ CORRECT - All state changes through Collection
sdk.features.complete('feat-123')
# Collection handles:
# 1. Load current node
# 2. Validate state transition (can it be completed?)
# 3. Update status + timestamp
# 4. Generate event (FeatureComplete)
# 5. Log to session (who, when, which session)
# 6. Save to HTML (atomic write)
# 7. Update SQLite index (searchable)
```

**Benefits:**
- ✅ **Single source of truth** - All state changes in one place
- ✅ **Consistent event logging** - Every mutation generates events
- ✅ **Validation and authorization** - Enforce rules (WIP limits, claims)
- ✅ **Audit trails** - Full traceability (who, when, why)
- ✅ **Parallel agent coordination** - Claims prevent conflicts

#### Example Usage

```python
from wipnote import SDK

sdk = SDK(agent='claude')

# State transitions
sdk.features.start('feat-123')        # todo → in-progress
sdk.features.complete('feat-123')     # in-progress → done
sdk.features.claim('feat-123')        # Claim ownership
sdk.features.release('feat-123')      # Release ownership

# Batch operations
sdk.features.mark_done(['feat-001', 'feat-002', 'feat-003'])
sdk.features.assign(['feat-004'], agent='agent-2')

# Query operations (lazy-loaded, efficient)
high_priority = sdk.features.where(status='todo', priority='high')
all_features = sdk.features.all()
in_progress = sdk.features.filter(lambda f: f.status == 'in-progress')
```

---

### Builder - Fluent Creation Interface

**Builders** (FeatureBuilder, SpikeBuilder, etc.) provide fluent APIs for creating nodes.

#### Responsibilities

- **Fluent method chaining** - `.set_priority().add_steps()`
- **Validation during construction** - Catch errors before save
- **Return immutable Node** - `.save()` returns read-only Node

#### Why Builders?

**Problem:** What if creation used constructors directly?

```python
# ❌ ANTI-PATTERN - Without builders
from wipnote.models import Node

feature = Node(
    id='feat-123',
    title='User Auth',
    type='feature',
    status='todo',
    priority='high',
    steps=[
        Step(description='Create endpoint', completed=False),
        Step(description='Add tests', completed=False),
    ],
    edges={'blocks': [Edge(target_id='feat-456', relationship='blocks')]},
    properties={},
    created=datetime.now(),
    updated=datetime.now(),
)
# Issues:
# - Verbose (15+ lines for simple feature)
# - Error-prone (manual ID generation)
# - No validation until save
# - No integration with SessionManager
```

**Solution:** Fluent Builder pattern

```python
# ✅ CORRECT - Fluent builder
feature = sdk.features.create("User Authentication") \
    .set_priority("high") \
    .add_steps([
        "Create login endpoint",
        "Add JWT middleware",
        "Write integration tests"
    ]) \
    .blocks("feat-456") \
    .save()

# Benefits:
# - Concise (6 lines vs 20+)
# - Readable (clear intent)
# - Auto-generates ID
# - Validates before save
# - Integrates with SessionManager
# - Returns immutable Node
```

**Benefits:**
- ✅ **Method chaining** - Fluent, readable code
- ✅ **Validation during construction** - Fail fast
- ✅ **Clear separation** - Builder vs data model
- ✅ **Integration** - Auto-connects to SessionManager events

#### Example Usage

```python
from wipnote import SDK

sdk = SDK(agent='claude')

# Create with builder
feature = sdk.features.create("User Authentication") \
    .set_priority("high") \
    .set_description("Implement OAuth 2.0 login") \
    .add_steps([
        "Create login endpoint",
        "Add JWT middleware",
        "Write integration tests"
    ]) \
    .blocks("feat-456") \
    .blocked_by("spike-789") \
    .save()  # Returns immutable Node

print(feature.id)        # feat-abc123 (auto-generated)
print(feature.status)    # todo
print(feature.priority)  # high

# Builder validates before save
try:
    bad_feature = sdk.features.create("") \
        .set_priority("invalid") \
        .save()
except Exception as e:
    print(f"Validation failed: {e}")
```

---

## Design Patterns

### Django ORM Influence

Wipnote's Collections follow **Django ORM naming conventions** for familiarity.

#### Why `.all()` not `.list()`?

**Common confusion:**

```python
# Many developers expect:
features = sdk.features.list()  # ❌ AttributeError

# But the API is:
features = sdk.features.all()   # ✅ Correct
```

**Why `.all()` instead of `.list()`?**

1. **Semantic clarity** - "Get ALL items" vs type-based "list"
   - `.all()` = "give me all features" (intent)
   - `.list()` = "return a list type" (implementation detail)

2. **Django/Rails familiarity** - Developers recognize pattern
   ```python
   # Django QuerySet API
   features = Feature.objects.all()
   high_priority = Feature.objects.filter(priority='high')

   # Wipnote Collection API (same pattern!)
   features = sdk.features.all()
   high_priority = sdk.features.where(priority='high')
   ```

3. **Pairs naturally** - Works with `.where()` and `.filter()`
   ```python
   sdk.features.all()                      # All features
   sdk.features.where(status='todo')       # Filtered subset
   sdk.features.filter(lambda f: ...)      # Custom filter
   ```

4. **Lazy evaluation future** - `.all()` can be lazy (Django-style)
   ```python
   # Future: Could return lazy QuerySet-like object
   features = sdk.features.all()  # Doesn't load yet
   count = features.count()        # Loads only count
   for f in features:              # Iterates without loading all
       print(f.title)
   ```

#### Pattern Comparison

```python
# Django ORM
from django.db import models

class Feature(models.Model):
    title = models.CharField(max_length=200)
    status = models.CharField(max_length=50)

# Query
features = Feature.objects.all()
todos = Feature.objects.filter(status='todo')
high_priority = Feature.objects.filter(priority='high')

# ─────────────────────────────────────────────────

# Wipnote SDK (same pattern!)
from wipnote import SDK

sdk = SDK(agent='claude')

# Query (same API!)
features = sdk.features.all()
todos = sdk.features.where(status='todo')
high_priority = sdk.features.where(priority='high')
```

---

### Event-Driven State Management

All state changes generate **events** for observability and coordination.

#### Event Flow

```
User Code                 Collection              SessionManager         Storage
─────────                 ──────────              ──────────────         ───────

sdk.features.complete()
    │
    ├─→ Load node         (from HTML + SQLite)
    │
    ├─→ Validate          (can it be completed?)
    │
    ├─→ Update state      status = "done"
    │                     timestamp = now()
    │
    ├─→ Generate event ───→ FeatureComplete
    │                          │
    │                          ├─→ Log event
    │                          │   (who, when, session)
    │                          │
    │                          └─→ Update session
    │                              (track completion)
    │
    └─→ Save to disk ──────────────────────────→ HTML file
                                                 SQLite index
```

#### Event Types

```python
from wipnote import SDK

sdk = SDK(agent='claude')

# FeatureCreate event
feature = sdk.features.create("User Auth").save()
# Event: {"type": "FeatureCreate", "agent": "claude", "feature_id": "feat-123"}

# FeatureStart event
sdk.features.start('feat-123')
# Event: {"type": "FeatureStart", "agent": "claude", "feature_id": "feat-123"}

# FeatureComplete event
sdk.features.complete('feat-123')
# Event: {"type": "FeatureComplete", "agent": "claude", "feature_id": "feat-123"}

# All events include:
# - timestamp (when)
# - agent (who)
# - session_id (which session)
# - transcript_id (which conversation)
```

#### Benefits

- ✅ **Full audit trail** - Who did what, when, and why
- ✅ **Multi-agent coordination** - Agents can see each other's work
- ✅ **Observability dashboards** - Visualize progress, bottlenecks
- ✅ **Debugging** - Replay events to understand state changes
- ✅ **Analytics** - Time-series data, velocity metrics

---

## Common Mistakes Explained

### Mistake 1: Why doesn't `Node.complete()` exist?

**The Error:**

```python
from wipnote import SDK

sdk = SDK(agent='claude')
feature = sdk.features.get('feat-123')

# ❌ This fails
feature.complete()
# AttributeError: 'Node' object has no attribute 'complete'
```

**Why this design?**

**Architecture Decision:** State mutations through Collections enforce:

1. **Event logging** - For audit trails
2. **Ownership validation** - Respect claims
3. **Parallel agent coordination** - Prevent conflicts
4. **SessionManager integration** - Track activity

**Correct pattern:**

```python
# ✅ CORRECT - State changes through Collection
sdk.features.complete('feat-123')

# Collection handles:
# - Validates node exists
# - Checks if claimed by another agent
# - Updates status + timestamp
# - Generates FeatureComplete event
# - Logs to active session
# - Saves atomically
```

**Reference:** See [bug-3a2bf73c](../.wipnote/bugs/bug-3a2bf73c.html) - Node objects have no `.complete()` instance method

---

### Mistake 2: Why `.all()` instead of `.list()`?

**The Error:**

```python
from wipnote import SDK

sdk = SDK(agent='claude')

# ❌ This fails
features = sdk.features.list()
# AttributeError: 'FeatureCollection' object has no attribute 'list'
```

**Why this design?**

**Design Choice:** Django ORM naming for:

1. **Semantic clarity** - "Get all" vs "return list type"
2. **Developer familiarity** - Recognized pattern
3. **Consistency** - Pairs with `.where()` / `.filter()`
4. **Future lazy evaluation** - Can optimize without breaking API

**Correct pattern:**

```python
# ✅ CORRECT - Use .all() (Django-style)
features = sdk.features.all()

# Also available:
todos = sdk.features.where(status='todo')
custom = sdk.features.filter(lambda f: f.priority == 'high')
```

**Reference:** See [bug-8b6e9736](../.wipnote/bugs/bug-8b6e9736.html) - Collection objects use `.all()` not `.list()`

---

### Mistake 3: Why can't I edit Node properties directly?

**The Error:**

```python
from wipnote import SDK

sdk = SDK(agent='claude')
feature = sdk.features.get('feat-123')

# ❌ This fails
feature.status = "done"
# pydantic_core._pydantic_core.ValidationError: Instance is frozen
```

**Why this design?**

**Immutability Principle:** Nodes are **frozen Pydantic models** to prevent:
- Silent state changes
- Missing audit trails
- Race conditions

**Correct pattern:**

```python
# ✅ CORRECT - Use edit() context manager
with sdk.features.edit('feat-123') as feature:
    feature.status = "done"
    feature.priority = "high"
# Auto-saves on exit, generates events

# Or direct update
sdk.features.update('feat-123', status='done', priority='high')
```

---

### Mistake 4: Why do builders return `Node`, not `Builder`?

**The Confusion:**

```python
from wipnote import SDK

sdk = SDK(agent='claude')

# .save() returns Node, not Builder
feature = sdk.features.create("User Auth").save()

# ❌ Can't chain after .save()
feature.set_priority("high")  # AttributeError
```

**Why this design?**

**Separation of Concerns:**
- **Builder** = Construction phase (mutable, chainable)
- **Node** = Data model (immutable, read-only)

`.save()` is the **transition point** from construction to data:

```python
# Construction phase (Builder)
builder = sdk.features.create("User Auth") \
    .set_priority("high") \
    .add_steps(["Create endpoint", "Add tests"])

# Transition: Builder → Node
feature = builder.save()

# Data phase (Node)
print(feature.id)        # Read-only access
print(feature.status)    # Read-only access
```

**Correct pattern:**

```python
# ✅ Build completely, then save
feature = sdk.features.create("User Auth") \
    .set_priority("high") \
    .set_description("OAuth 2.0") \
    .add_steps(["Create endpoint", "Add tests"]) \
    .save()  # Returns immutable Node

# Use Collection methods for state changes
sdk.features.start(feature.id)
```

---

## Event-Driven State Management

### Event Schema

All Wipnote events follow a consistent schema:

```python
{
    "type": "FeatureComplete",      # Event type
    "timestamp": "2025-01-03T...",  # ISO 8601 timestamp
    "agent": "claude",              # Agent who triggered event
    "session_id": "sess-abc123",    # Session ID
    "transcript_id": "tx-xyz789",   # Conversation/transcript ID
    "feature_id": "feat-456",       # Entity ID
    "data": {                       # Event-specific data
        "previous_status": "in-progress",
        "new_status": "done"
    }
}
```

### Event Storage

Events are stored in multiple locations for different use cases:

1. **JSONL event log** - `.wipnote/events/*.jsonl`
   - Append-only log
   - Time-ordered
   - Fast sequential reads

2. **SQLite database** - `.wipnote/graph.db`
   - Indexed for queries
   - Fast lookups
   - Aggregations

3. **Session HTML** - `.wipnote/sessions/*.html`
   - Human-readable
   - Visualizable in dashboard
   - Shareable

### Event Query API

```python
from wipnote import SDK

sdk = SDK(agent='claude')

# Get events for a feature
events = sdk.analytics.get_events(
    entity_type='feature',
    entity_id='feat-123'
)

for event in events:
    print(f"{event.timestamp}: {event.type} by {event.agent}")

# Cross-session analytics
velocity = sdk.cross_session.get_velocity_metrics(days=7)
print(f"Features completed: {velocity['features_completed']}")
```

---

## Multi-Agent Coordination

### Claims and Ownership

Wipnote supports **multi-agent coordination** through claims:

```python
from wipnote import SDK

sdk = SDK(agent='claude')

# Agent 1 claims a feature
sdk.features.claim('feat-123')
# → Generates FeatureClaim event
# → Sets agent_assigned = "claude"
# → Prevents other agents from claiming

# Agent 2 tries to claim (fails)
sdk2 = SDK(agent='copilot')
try:
    sdk2.features.claim('feat-123')
except ClaimConflictError:
    print("Already claimed by claude")

# Agent 1 completes and releases
sdk.features.complete('feat-123')
sdk.features.release('feat-123')
# → Now available for other agents
```

### WIP Limits

SessionManager enforces **work-in-progress limits** to prevent overload:

```python
from wipnote import SDK

sdk = SDK(agent='claude')

# Default WIP limit = 3 features
sdk.features.start('feat-001')  # ✅ OK (1 in progress)
sdk.features.start('feat-002')  # ✅ OK (2 in progress)
sdk.features.start('feat-003')  # ✅ OK (3 in progress)
sdk.features.start('feat-004')  # ❌ Error: WIP limit exceeded

# Complete one to free up slot
sdk.features.complete('feat-001')
sdk.features.start('feat-004')  # ✅ OK now (3 in progress)
```

### Parallel Work Recommendations

Wipnote can **recommend parallelizable work** for multi-agent teams:

```python
from wipnote import SDK

sdk = SDK(agent='orchestrator')

# Find work for 3 agents
recommendations = sdk.dep_analytics.recommend_next_tasks(agent_count=3)

for rec in recommendations:
    print(f"Agent {rec['agent_slot']}: {rec['task_id']}")
    print(f"  Priority: {rec['priority']}")
    print(f"  Estimated effort: {rec['estimated_hours']}h")
    print(f"  Blockers: {rec['blocker_count']}")

# Output:
# Agent 1: feat-123
#   Priority: critical
#   Estimated effort: 2h
#   Blockers: 0
# Agent 2: feat-456
#   Priority: high
#   Estimated effort: 4h
#   Blockers: 0
# Agent 3: spike-789
#   Priority: medium
#   Estimated effort: 1h
#   Blockers: 0
```

---

## Architecture Diagrams

### Component Relationships

```
┌─────────────────────────────────────────────────────────┐
│                         SDK                             │
│  (Main entry point for AI agents)                       │
└─────────────────────────────────────────────────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        │                   │                   │
        ▼                   ▼                   ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│ Collections  │    │   Builders   │    │  Analytics   │
├──────────────┤    ├──────────────┤    ├──────────────┤
│ - Features   │    │ - Feature    │    │ - Dependency │
│ - Spikes     │    │ - Spike      │    │ - Velocity   │
│ - Bugs       │    │ - Bug        │    │ - Context    │
│ - Sessions   │    │ - Track      │    │ - Cross-     │
│ - Tracks     │    │              │    │   Session    │
└──────────────┘    └──────────────┘    └──────────────┘
        │                   │                   │
        │                   │                   │
        └───────────────────┼───────────────────┘
                            │
                            ▼
                ┌──────────────────────┐
                │   SessionManager     │
                │  (Event logging,     │
                │   state tracking)    │
                └──────────────────────┘
                            │
                ┌───────────┴───────────┐
                │                       │
                ▼                       ▼
        ┌──────────────┐        ┌──────────────┐
        │  Wipnote   │        │  EventLog    │
        │  (Storage)   │        │  (JSONL)     │
        └──────────────┘        └──────────────┘
```

### State Transition Flow

```
Builder Phase              Node Phase              Collection Phase
─────────────              ──────────              ────────────────

create("title")
   │
   ├─→ set_priority()
   │
   ├─→ add_steps()
   │
   └─→ save() ──────────→ Node (immutable) ───────→ start(id)
                             │                         │
                             │                         ├─→ Load
                             │                         │
                             │                         ├─→ Validate
                             │                         │
                             ▼                         ├─→ Update
                          Read-only                    │
                          Properties                   └─→ Event ──→ Log
                             │
                             │
                             └─────────────────────→ complete(id)
                                                        │
                                                        └─→ Event ──→ Log
```

---

## Summary

Wipnote's architecture follows three key principles:

1. **Immutable Nodes** - Data models are read-only, preventing silent mutations
2. **Centralized State** - Collections manage all state transitions with events
3. **Event-Driven** - Every change is logged for observability and coordination

This design enables:
- ✅ **Multi-agent coordination** - Claims, WIP limits, parallel work
- ✅ **Full observability** - Complete audit trails, analytics
- ✅ **Thread-safe operations** - No race conditions
- ✅ **Familiar patterns** - Django-style ORM API
- ✅ **Fluent builders** - Readable, chainable construction

**For more details:**
- [API Reference](./API_REFERENCE.md) - Complete API documentation
- [SDK Cheat Sheet](./sdk-cheat-sheet.md) - Quick reference
- [Planning Workflow](./PLANNING_WORKFLOW.md) - Real-world usage patterns
- [AGENTS.md](../AGENTS.md) - AI agent integration guide
