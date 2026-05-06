# Wipnote SDK - Modular Architecture

**Comprehensive guide to the refactored SDK structure**

---

## Table of Contents

1. [Overview](#overview)
2. [Architecture Diagram](#architecture-diagram)
3. [Component Breakdown](#component-breakdown)
4. [Dependency Flow](#dependency-flow)
5. [Benefits](#benefits)
6. [Migration Guide](#migration-guide)
7. [Extension Guide](#extension-guide)

---

## Overview

The Wipnote SDK uses a **modular mixin-based architecture** that composes specialized functionality into a unified interface. This design emerged from refactoring a monolithic 2,492-line `sdk.py` file into focused, maintainable components.

### Key Principles

1. **Single Responsibility** - Each mixin handles one domain
2. **Composition over Inheritance** - SDK is composed from specialized mixins
3. **Lazy Loading** - Components initialized only when needed
4. **Backward Compatibility** - All existing SDK code continues to work
5. **Type Safety** - Full type hints throughout

### Before and After

**Before (Monolithic):**
```
src/python/wipnote/
└── sdk.py (2,492 lines - everything in one file)
```

**After (Modular):**
```
src/python/wipnote/sdk/
├── __init__.py           # SDK composition (80 lines)
├── base.py               # Core initialization (485 lines)
├── constants.py          # Configuration (217 lines)
├── discovery.py          # Auto-discovery (121 lines)
├── analytics/            # Analytics registry
│   ├── __init__.py
│   ├── registry.py       # Property access (110 lines)
│   ├── engine.py         # Lazy loading (150 lines)
│   └── helpers.py        # Utilities
├── session/              # Session management (4 mixins)
│   ├── __init__.py
│   ├── manager.py        # Lifecycle (200 lines)
│   ├── handoff.py        # Context transfer (150 lines)
│   ├── continuity.py     # Resume work (100 lines)
│   └── info.py           # Session info (180 lines)
├── planning/             # Strategic planning
│   ├── __init__.py
│   ├── mixin.py          # Delegation layer (212 lines)
│   ├── bottlenecks.py    # Bottleneck detection
│   ├── parallel.py       # Parallel work
│   ├── recommendations.py # Smart recommendations
│   ├── queue.py          # Work queue
│   └── smart_planning.py # Planning workflows
├── orchestration/        # Multi-agent coordination
│   ├── __init__.py
│   ├── coordinator.py    # Orchestration mixin
│   └── spawner.py        # Spawner utilities
├── operations/           # Infrastructure operations
│   ├── __init__.py
│   └── mixin.py          # Server, hooks, events (428 lines)
├── mixins/               # Core utilities
│   ├── __init__.py
│   ├── mixin.py          # Database, refs, utils (411 lines)
│   └── attribution.py    # Task attribution
└── help/                 # Help system
    ├── __init__.py
    └── mixin.py          # Interactive help
```

---

## Architecture Diagram

```
                              ┌──────────────┐
                              │     SDK      │
                              │   (Thin      │
                              │ Composition) │
                              └──────┬───────┘
                                     │
        ┌────────────────────────────┼────────────────────────────┐
        │                            │                            │
        │                            │                            │
┌───────▼────────┐         ┌────────▼─────────┐        ┌────────▼────────┐
│   Analytics    │         │    Session       │        │    Planning     │
│    Registry    │         │  Management      │        │     Mixin       │
├────────────────┤         ├──────────────────┤        ├─────────────────┤
│ • analytics    │         │ • start_session  │        │ • bottlenecks   │
│ • dep_analytics│         │ • end_session    │        │ • recommend     │
│ • context      │         │ • handoff        │        │ • parallel_work │
│ • pattern_     │         │ • continuity     │        │ • smart_plan    │
│   learning     │         │ • session_info   │        │ • work_queue    │
└────────────────┘         └──────────────────┘        └─────────────────┘
        │                            │                            │
        │                            │                            │
┌───────▼────────┐         ┌────────▼─────────┐        ┌────────▼────────┐
│ Orchestration  │         │   Operations     │        │      Core       │
│     Mixin      │         │     Mixin        │        │     Mixin       │
├────────────────┤         ├──────────────────┤        ├─────────────────┤
│ • spawn_*      │         │ • start_server   │        │ • db/query      │
│ • orchestrate  │         │ • install_hooks  │        │ • ref           │
│ • coordinator  │         │ • export_sessions│        │ • reload        │
│                │         │ • analytics_ops  │        │ • summary       │
└────────────────┘         └──────────────────┘        └─────────────────┘
        │                            │                            │
        │                            │                            │
┌───────▼────────┐         ┌────────▼─────────┐
│ TaskAttribution│         │      Help        │
│     Mixin      │         │     Mixin        │
├────────────────┤         ├──────────────────┤
│ • get_task_    │         │ • help()         │
│   attribution  │         │ • __dir__()      │
│ • get_subagent_│         │                  │
│   work         │         │                  │
└────────────────┘         └──────────────────┘
```

---

## Component Breakdown

### 1. AnalyticsRegistry

**Purpose**: Property-based access to analytics interfaces

**Location**: `src/python/wipnote/sdk/analytics/registry.py`

**Responsibilities**:
- Lazy-load analytics engines
- Provide property access (`sdk.analytics`, `sdk.dep_analytics`)
- Delegate to AnalyticsEngine for initialization

**Properties**:
```python
sdk.analytics                  # Work type analytics
sdk.dep_analytics              # Dependency graph analytics
sdk.cross_session_analytics    # Git commit-based analytics
sdk.context                    # Context usage tracking
sdk.pattern_learning           # Behavior pattern learning
```

**Implementation Pattern**:
```python
class AnalyticsRegistry:
    _analytics_engine: AnalyticsEngine

    @property
    def analytics(self) -> Analytics:
        """Lazy-loaded analytics interface."""
        return self._analytics_engine.analytics
```

**Dependencies**:
- `AnalyticsEngine` (lazy-loaded)
- Analytics modules (Analytics, DependencyAnalytics, etc.)

---

### 2. Session Management (4 Mixins)

**Location**: `src/python/wipnote/sdk/session/`

#### 2a. SessionManagerMixin

**File**: `session/manager.py`

**Purpose**: Session lifecycle operations

**Methods**:
```python
sdk.start_session(feature_id, agent, ...)
sdk.end_session(session_id)
sdk._ensure_session_exists(session_id)
```

**Responsibilities**:
- Start/end sessions
- Ensure session exists
- Create session records

#### 2b. SessionHandoffMixin

**File**: `session/handoff.py`

**Purpose**: Context-preserving task transfers between agents

**Methods**:
```python
sdk.set_session_handoff(session_id, handoff_to, reason, notes)
sdk.end_session_with_handoff(session_id, handoff_to, reason, notes)
```

**Responsibilities**:
- Set handoff metadata
- End session with handoff
- Preserve context for next agent

#### 2c. SessionContinuityMixin

**File**: `session/continuity.py`

**Purpose**: Resume work from previous sessions

**Methods**:
```python
sdk.continue_from_last(feature_id, agent)
```

**Responsibilities**:
- Continue from last session
- Maintain session chain

#### 2d. SessionInfoMixin

**File**: `session/info.py`

**Purpose**: Session context and work item tracking

**Methods**:
```python
sdk.get_session_start_info(session_id)
sdk.get_active_work_item(session_id)
sdk.track_activity(session_id, activity_type, ...)
```

**Responsibilities**:
- Get session start info
- Get active work item
- Track activity to session

---

### 3. PlanningMixin

**Location**: `src/python/wipnote/sdk/planning/mixin.py`

**Purpose**: Strategic planning and work recommendations

**Methods**:
```python
# Strategic analytics
sdk.find_bottlenecks(top_n=5)
sdk.get_parallel_work(max_agents=5)
sdk.recommend_next_work(agent_count=1)
sdk.assess_risks()
sdk.analyze_impact(node_id)

# Work queue
sdk.get_work_queue(agent_id, limit, min_score)
sdk.work_next(agent_id, auto_claim, min_score)

# Planning workflows
sdk.start_planning_spike(title, context, timebox_hours)
sdk.create_track_from_plan(title, description, spike_id, ...)
sdk.smart_plan(description, create_spike, ...)
sdk.plan_parallel_work(max_agents, shared_files)
sdk.aggregate_parallel_results(agent_ids)
```

**Delegation Pattern**: Each method delegates to a specialized module:
- `bottlenecks.py` - Bottleneck detection
- `parallel.py` - Parallel work coordination
- `recommendations.py` - Smart recommendations
- `queue.py` - Work queue management
- `smart_planning.py` - Planning workflows

**Example Delegation**:
```python
def find_bottlenecks(self, top_n: int = 5) -> list[BottleneckDict]:
    """Delegates to bottlenecks module."""
    from wipnote.sdk.planning.bottlenecks import find_bottlenecks
    return find_bottlenecks(self, top_n=top_n)
```

---

### 4. OrchestrationMixin

**Location**: `src/python/wipnote/sdk/orchestration/coordinator.py`

**Purpose**: Multi-agent coordination and spawning

**Methods**:
```python
sdk.spawn_explorer(feature_id, search_scope, ...)
sdk.spawn_coder(feature_id, implementation_plan, ...)
sdk.orchestrate(tasks, coordination_strategy)
sdk.orchestrator  # Property - lazy-loaded coordinator
```

**Responsibilities**:
- Spawn subagents (explorer, coder, etc.)
- Orchestrate parallel workflows
- Manage orchestrator instance

**Dependencies**:
- `spawner.py` - Spawner utilities
- Orchestrator class (lazy-loaded)

---

### 5. OperationsMixin

**Location**: `src/python/wipnote/sdk/operations/mixin.py`

**Purpose**: Infrastructure operations (server, hooks, events, analytics)

**Methods**:
```python
# Server (75 lines)
sdk.start_server(port, host, watch, auto_port)
sdk.stop_server(handle)
sdk.get_server_status(handle)

# Hooks (72 lines)
sdk.install_hooks(use_copy)
sdk.list_hooks()
sdk.validate_hook_config()

# Events (128 lines)
sdk.export_sessions(overwrite)
sdk.rebuild_event_index()
sdk.query_events(session_id, tool, feature_id, since, limit)
sdk.get_event_stats()

# Analytics (98 lines)
sdk.analyze_session(session_id)
sdk.analyze_project()
sdk.get_work_recommendations()
```

**Delegation**: All methods delegate to `wipnote.operations` module for shared backend.

**Example**:
```python
def start_server(self, port=8080, host="localhost", watch=True, auto_port=False):
    """Delegates to operations.server module."""
    from wipnote.operations import server
    return server.start_server(
        port=port,
        graph_dir=self._directory,
        static_dir=self._directory.parent,
        host=host,
        watch=watch,
        auto_port=auto_port,
    )
```

---

### 6. CoreMixin

**Location**: `src/python/wipnote/sdk/mixins/mixin.py`

**Purpose**: Essential SDK utilities

**Methods**:
```python
# Database (60 lines)
sdk.db()
sdk.query(sql, params)
sdk.execute_query_builder(sql, params)
sdk.export_to_html(output_dir, include_features, ...)

# Refs (54 lines)
sdk.ref(short_ref)  # "@f1" → Feature node

# Utilities (85 lines)
sdk.reload()
sdk.summary(max_items)
sdk.my_work()
sdk.next_task(priority, auto_claim)
sdk.get_status()
sdk.dedupe_sessions(max_events, move_dir_name, dry_run)

# Internal (70 lines)
sdk._log_event(event_type, tool_name, ...)
```

**Responsibilities**:
- Database access
- Ref resolution
- Export functionality
- Event logging
- Utility methods

---

### 7. TaskAttributionMixin

**Location**: `src/python/wipnote/sdk/mixins/attribution.py`

**Purpose**: Task and subagent work tracking

**Methods**:
```python
sdk.get_task_attribution(task_id)
sdk.get_subagent_work(agent_id)
```

---

### 8. HelpMixin

**Location**: `src/python/wipnote/sdk/help/mixin.py`

**Purpose**: Interactive help and introspection

**Methods**:
```python
sdk.help()           # General help
sdk.help("planning") # Planning-specific help
dir(sdk)             # List all available methods
```

---

## SDK Composition

The main SDK class is a **thin composition layer** that inherits from all mixins:

```python
# src/python/wipnote/sdk/__init__.py

class SDK(
    AnalyticsRegistry,
    SessionManagerMixin,
    SessionHandoffMixin,
    SessionContinuityMixin,
    SessionInfoMixin,
    PlanningMixin,
    OrchestrationMixin,
    OperationsMixin,
    CoreMixin,
    TaskAttributionMixin,
    HelpMixin,
):
    """Main SDK interface - composed from specialized mixins."""

    def __init__(self, directory=None, agent=None, ...):
        # Initialize shared resources
        self._directory = Path(directory or discover_wipnote_dir())
        self._agent_id = agent or auto_discover_agent()
        self._db = WipnoteDB(...)
        self._graph = Wipnote(...)

        # Initialize collections
        self.features = FeatureCollection(self)
        self.bugs = BugCollection(self)
        # ... etc

        # Initialize analytics engine (lazy-loaded components)
        self._analytics_engine = create_analytics_engine(...)

        # Session manager
        self.session_manager = SessionManager(...)
```

**Key Design Decision**: SDK initialization sets up shared resources (database, graphs, collections) that mixins can access via attributes. Mixins add domain-specific methods but don't duplicate initialization logic.

---

## Dependency Flow

```
User Code
   │
   ▼
SDK (thin composition layer)
   │
   ├──▶ AnalyticsRegistry ──▶ AnalyticsEngine ──▶ Analytics modules
   │
   ├──▶ SessionManagerMixin ──▶ SessionManager
   │
   ├──▶ PlanningMixin ──▶ Planning modules (bottlenecks, parallel, queue, etc.)
   │
   ├──▶ OrchestrationMixin ──▶ Orchestrator (lazy-loaded)
   │
   ├──▶ OperationsMixin ──▶ Operations modules (server, hooks, events, analytics)
   │
   └──▶ CoreMixin ──▶ WipnoteDB, Wipnote, RefManager
```

**Lazy Loading Strategy**:
- Analytics components loaded on first property access
- Orchestrator loaded on first orchestration call
- System prompts loaded on first access
- Collections loaded eagerly (lightweight)

---

## Benefits

### 1. Maintainability
- **Before**: 2,492-line monolithic file
- **After**: Largest mixin ~428 lines, most ~100-200 lines
- Easier to understand, modify, and test individual components

### 2. Separation of Concerns
- Each mixin has clear, focused responsibility
- No cross-cutting concerns (analytics doesn't know about orchestration)
- Easy to reason about component behavior

### 3. Extensibility
- Add new mixin for new functionality
- Extend existing mixin without touching others
- Plugin architecture possible (custom mixins)

### 4. Testability
- Test mixins independently
- Mock dependencies easily
- Smaller test surface area

### 5. Type Safety
- Full type hints throughout
- TYPE_CHECKING imports prevent circular dependencies
- MyPy validates mixin composition

### 6. Backward Compatibility
- All existing SDK code works unchanged
- API surface identical to monolithic version
- Migration path: internal refactor, external stability

---

## Migration Guide

### No Breaking Changes

The modular refactor was **100% backward compatible**:

```python
# Old code (pre-refactor)
from wipnote import SDK
sdk = SDK(agent="claude")
sdk.analytics.work_type_distribution()
sdk.find_bottlenecks(top_n=5)

# New code (post-refactor) - IDENTICAL
from wipnote import SDK
sdk = SDK(agent="claude")
sdk.analytics.work_type_distribution()
sdk.find_bottlenecks(top_n=5)
```

### Internal Changes Only

- File structure changed (`sdk.py` → `sdk/` package)
- Class hierarchy changed (monolithic → mixin composition)
- **Public API unchanged**

### Import Compatibility

Both old and new import paths work:

```python
# Old path (still works)
from wipnote import SDK

# New path (also works)
from wipnote.sdk import SDK

# Direct mixin access (for custom compositions)
from wipnote.sdk.planning import PlanningMixin
```

---

## Extension Guide

### Adding New Mixins

To add a new capability domain:

1. Create mixin file: `src/python/wipnote/sdk/new_domain/mixin.py`
2. Define mixin class with methods
3. Add to SDK composition in `sdk/__init__.py`
4. Add exports to `sdk/new_domain/__init__.py`

Example:

```python
# src/python/wipnote/sdk/testing/mixin.py
class TestingMixin:
    """Test execution and validation."""

    def run_tests(self, pattern: str) -> TestResult:
        from wipnote.sdk.testing.runner import run_tests
        return run_tests(self, pattern)

# src/python/wipnote/sdk/__init__.py
class SDK(
    # ... existing mixins
    TestingMixin,
):
    pass
```

### Custom SDK Compositions

Users can compose custom SDKs with only needed mixins:

```python
from wipnote.sdk.base import BaseSDK
from wipnote.sdk.planning import PlanningMixin
from wipnote.sdk.analytics import AnalyticsRegistry

class LightweightSDK(BaseSDK, PlanningMixin, AnalyticsRegistry):
    """Custom SDK with only planning and analytics."""
    pass

sdk = LightweightSDK(agent="claude")
```

---

## Common Patterns

### Pattern 1: Delegation to Specialized Modules

Mixins are **thin delegation layers**. They don't implement complex logic; they delegate to specialized modules:

```python
# In PlanningMixin
def find_bottlenecks(self, top_n: int = 5) -> list[BottleneckDict]:
    """Delegates to bottlenecks module."""
    from wipnote.sdk.planning.bottlenecks import find_bottlenecks
    return find_bottlenecks(self, top_n=top_n)
```

**Why?** Keeps mixins small, testable. Logic lives in pure functions, not methods.

### Pattern 2: Lazy Loading via Properties

Heavy components initialized only when needed:

```python
# In AnalyticsRegistry
@property
def analytics(self) -> Analytics:
    """Lazy-loaded analytics interface."""
    return self._analytics_engine.analytics
```

**Why?** Fast SDK initialization. Most users don't use all features.

### Pattern 3: TYPE_CHECKING for Circular Dependencies

Use TYPE_CHECKING to import types without runtime cost:

```python
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from wipnote.models import Node
    from wipnote.types import BottleneckDict
```

**Why?** Prevents circular imports, maintains type safety.

### Pattern 4: Shared Resources via Attributes

SDK initialization sets up shared resources (database, graphs) that mixins access via attributes:

```python
class CoreMixin:
    _directory: Path
    _db: WipnoteDB
    _agent_id: str | None
    # ... mixins declare what they need
```

**Why?** No duplication. SDK owns initialization, mixins consume.

---

## Performance Characteristics

### Initialization Time
- **Fast**: Lazy loading defers heavy components
- Collections loaded eagerly (lightweight HTML graphs)
- Analytics engine created but components not initialized

### Memory Footprint
- **Small**: Most components loaded on-demand
- Database connection reused across operations
- Graphs cached, not reloaded

### Method Call Overhead
- **Negligible**: Single delegation hop (mixin → module)
- No performance regression vs monolithic version

---

## Related Documentation

- [MODULARIZATION.md](./MODULARIZATION.md) - Refactoring journey and lessons learned
- [ARCHITECTURE.md](./architecture/design.md) - Design philosophy and patterns
- [API_REFERENCE.md](./api/reference.md) - Complete SDK API documentation
- [AGENTS.md](../AGENTS.md) - SDK usage guide for AI agents
- Source code: `src/python/wipnote/sdk/`

---

## Summary

The Wipnote SDK's modular architecture achieves:

✅ **Maintainability** - Small, focused components (100-400 lines each)
✅ **Extensibility** - Easy to add new capabilities via mixins
✅ **Testability** - Components tested independently
✅ **Performance** - Lazy loading, minimal overhead
✅ **Backward Compatibility** - Zero breaking changes
✅ **Type Safety** - Full type hints, MyPy validation

The architecture enables rapid development while maintaining code quality and developer experience.
