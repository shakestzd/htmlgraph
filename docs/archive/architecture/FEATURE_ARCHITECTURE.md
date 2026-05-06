# Wipnote Feature Architecture & Exposure Review

**Document Version:** 1.0
**Date:** 2026-01-05
**Focus:** How features are exposed through SDK, Plugin, and CLI; state management; customization points

---

## Executive Summary

Wipnote exposes work tracking features through three integrated layers:

1. **Python SDK** - Fluent API for programmatic access (primary interface)
2. **Claude Code Plugin** - CLI commands for interactive use (secondary interface)
3. **File-Based State** - HTML and JSONL as single source of truth (storage layer)

The architecture separates **exposure mechanisms** (SDK/Plugin/CLI) from **state storage** (HTML files + JSONL event log), enabling flexible consumption while maintaining a unified data model.

---

## 1. Feature Exposure Architecture

### 1.1 Exposure Layers

```
┌─────────────────────────────────────────────────────────┐
│  User Interaction Layer                                 │
├─────────────────────────────────────────────────────────┤
│  CLI Commands (Plugin)  │  Python SDK  │  Direct HTML  │
├─────────────────────────────────────────────────────────┤
│  Data Access Layer (Collections + Builders)            │
├─────────────────────────────────────────────────────────┤
│  Storage Layer                                          │
│  - HTML files (.wipnote/features/*.html)             │
│  - JSONL events (.wipnote/events/*.jsonl)            │
│  - SQLite index (.wipnote/index.sqlite - cached)     │
└─────────────────────────────────────────────────────────┘
```

### 1.2 Exposure Method Comparison

| Exposure Method | Primary Use | State Source | Default Behavior | Customization |
|---|---|---|---|---|
| **SDK (Python)** | Automation, scripts, hooks | Direct file I/O | Auto-discovers `.wipnote` | Full control via builders |
| **Plugin (CLI)** | Interactive session work | SDK → File I/O | Project-scoped defaults | Command arguments + config |
| **Hook Scripts** | Automatic tracking | Event log writer | Real-time attribution | Event matcher + config files |
| **HTML Direct** | Manual inspection, dashboards | Read-only parsing | Rendered in `index.html` | None (immutable on disk) |

---

## 2. Feature Collection & Builder Pattern

### 2.1 Collection Architecture

**BaseCollection** (generic base class)
```python
class BaseCollection(Generic[CollectionT]):
    """Generic collection for any node type"""

    # Shared methods across all collections
    get(node_id) -> Node | None
    where(status, priority, **filters) -> list[Node]
    all() -> list[Node]
    create(title) -> Builder  # Returns typed builder
    edit(node_id) -> context manager  # Auto-saves
    delete(node_id) -> bool
    mark_done(node_ids: list) -> None  # Batch
    assign(node_id, agent) -> None
    release(node_id) -> None
```

**Specialized Collections** (per work type)
- `FeatureCollection` - Fluent feature creation with builder
- `BugCollection` - Bug tracking
- `SpikeCollection` - Research/planning spikes
- `ChoreCollection` - Maintenance tasks
- `EpicCollection` - Large initiatives
- `PhaseCollection` - Project phases
- `TrackCollection` - Multi-feature work tracks

Each collection lazy-loads its graph from:
```
.wipnote/
├── features/          # Feature nodes (HTML files)
├── bugs/              # Bug nodes
├── spikes/            # Spike nodes
├── chores/            # Chore nodes
├── epics/             # Epic nodes
├── phases/            # Phase nodes
├── tracks/            # Track nodes (files or directories)
└── ...
```

### 2.2 Builder Pattern (Fluent API)

**BaseBuilder** - Shared builder methods
```python
class BaseBuilder(Generic[BuilderT]):
    node_type: str  # Override in subclasses

    # Common methods
    set_priority(priority) -> Self
    set_status(status) -> Self
    add_steps(descriptions: list) -> Self
    complete_step(index) -> Self
    set_description(desc) -> Self
    add_edge(edge_type, target_id) -> Self
    save() -> Node
```

**Specialized Builders** (inherit from BaseBuilder)
- `FeatureBuilder` - Features with `set_required_capabilities()`
- `BugBuilder` - Bugs with severity/reproduction steps
- `SpikeBuilder` - Spikes with timebox/investigation type
- `TrackBuilder` - Tracks with fluent phase management

**Example Usage:**
```python
# SDK + Builder pattern
sdk = SDK(agent="claude")

feature = sdk.features.create("User Authentication") \
    .set_priority("high") \
    .add_steps([
        "Design schema",
        "Implement API",
        "Add tests"
    ]) \
    .set_required_capabilities(["python", "security"]) \
    .save()

# Auto-saves on exit
with sdk.features.edit("feat-001") as feature:
    feature.status = "in-progress"
    feature.complete_step(0)
    # Auto-saves when exiting context
```

---

## 3. State Management & Storage

### 3.1 Primary Storage: HTML Files

**Location:** `.wipnote/{type}/{id}.html`

**Example Feature Node:**
```html
<article id="feat-64467b2c"
         data-type="feature"
         data-status="todo"
         data-priority="medium"
         data-created="2025-12-31T06:43:21.034248"
         data-updated="2025-12-31T06:43:21.038300"
         data-track-id="trk-46d512ab">

    <header>
        <h1>Convert list commands to Rich tables</h1>
        <div class="metadata">
            <span class="badge status-todo">Todo</span>
            <span class="badge priority-medium">Medium Priority</span>
        </div>
    </header>

    <nav data-graph-edges>
        <section data-edge-type="implemented-in">
            <h3>Implemented-In:</h3>
            <ul>
                <li><a href="sess-3d9ec350.html"
                       data-relationship="implemented-in"
                       data-since="2025-12-31T06:43:21.038295">
                    sess-3d9ec350
                </a></li>
            </ul>
        </section>
    </nav>

    <section data-steps>
        <h3>Implementation Steps</h3>
        <ol>
            <li data-completed="false">⏳ Define Rich table schema</li>
            <li data-completed="false">⏳ Implement table formatter</li>
        </ol>
    </section>

    <section data-properties>
        <h3>Properties</h3>
        <dl>
            <dt>Track Id</dt>
            <dd data-key="track_id" data-value="trk-46d512ab">
                trk-46d512ab
            </dd>
        </dl>
    </section>
</article>
```

**Key Design Points:**
- HTML is the **authoritative source** (not database)
- All metadata in `data-*` attributes (CSS-queryable)
- Git-friendly format (human-readable diffs)
- Hyperlinks represent edges in the graph
- Properties stored in `<dl>` elements
- Queryable via CSS selectors

### 3.2 Event Log: JSONL (Append-Only)

**Location:** `.wipnote/events/{date}.jsonl`

**Design:** Git-friendly event tracking
- Append-only (one event per line)
- Chronological (rebuild-able)
- Minimal I/O contention
- Supports high-frequency activity

**EventRecord Schema:**
```python
@dataclass
class EventRecord:
    # Core tracking
    event_id: str
    timestamp: datetime
    session_id: str
    agent: str

    # Activity details
    tool: str                    # "Edit", "Read", "Bash", etc.
    summary: str                # Human-readable summary
    success: bool

    # Work attribution
    feature_id: str | None      # Which feature is this for?
    work_type: str | None       # WorkType enum: feature, spike, bug, chore

    # Multi-AI delegation (Phase 1)
    delegated_to_ai: str | None # "gemini", "codex", "copilot", "claude"
    task_id: str | None
    task_status: str | None     # "pending", "running", "completed", "failed"
    model_selected: str | None  # "gemini-2.0-flash", etc.
    complexity_level: str | None
    execution_duration_seconds: float | None
    tokens_estimated: int | None
    tokens_actual: int | None
    cost_usd: float | None
```

### 3.3 Cached Index: SQLite

**Location:** `.wipnote/index.sqlite` (git-ignored, rebuildable)

**Purpose:** Fast analytics queries without parsing HTML each time

**Rebuild Command:**
```bash
uv run wipnote reindex  # Rebuilds from HTML + JSONL
```

---

## 4. Plugin Hook Integration

### 4.1 Hook Points & Tracking

**Installed Hooks** (from `packages/claude-plugin/hooks/hooks.json`)

| Hook Type | Script | Purpose | Tracking |
|---|---|---|---|
| `SessionStart` | `session-start.py` | Record session start, show context | Session creation |
| `SessionEnd` | `session-end.py` | Record session completion, analysis | Session closure + analysis |
| `UserPromptSubmit` | `user-prompt-submit.py` | Classify task type before execution | Task classification |
| `PreToolUse` | `pretooluse-integrator.py` | Validate work items, enforce orchestrator | Work validation |
| `PostToolUse` | `posttooluse-integrator.py` | Track activity, attribute to features | Activity logging |
| `PostToolUseFailure` | `post-tool-use-failure.py` | Capture error events | Error tracking |
| `Stop` | `track-event.py` | End-of-session summary | Session finalization |

### 4.2 Hook Script Architecture

**Entry Point:** `.claude/hooks/scripts/{hook-name}.py`
```python
#!/usr/bin/env -S uv run
# /// script
# requires-python = ">=3.10"
# dependencies = ["wipnote"]
# ///
"""
Hook integration with minimal logic.
Delegates to wipnote package modules.
"""

from wipnote.hooks.{module} import main

if __name__ == "__main__":
    main()
```

**Design:** Hooks are thin wrappers around package logic in `wipnote.hooks/`

**Key Modules:**
- `wipnote.hooks.event_tracker` - Core event logging
- `wipnote.hooks.pretooluse` - Work validation
- `wipnote.hooks.task_validator` - Task classification
- `wipnote.hooks.orchestrator_reflector` - Orchestrator mode support

### 4.3 Session Tracking Flow

```
Claude Code Session Start
    ↓
[SessionStart Hook]
    ├─ Load .wipnote/
    ├─ Create session record
    ├─ Check version (warn if outdated)
    └─ Return feature context to Claude
    ↓
[User Works]
    ↓
[PreToolUse Hook]
    ├─ Validate work items exist
    ├─ Enforce orchestrator mode rules
    └─ Provide feedback
    ↓
[PostToolUse Hook]
    ├─ Track activity (file edits, commands)
    ├─ Auto-attribute to active features
    ├─ Calculate drift score
    └─ Append to events.jsonl
    ↓
[SessionEnd Hook]
    ├─ Record session completion
    ├─ Run completion analysis
    ├─ Calculate efficiency score
    └─ Create session HTML summary
```

---

## 5. Plugin Commands (CLI Interface)

### 5.1 Available Commands

**Work Item Management:**
- `/wipnote:feature-add [title]` - Create feature
- `/wipnote:feature-start [id]` - Start working on feature
- `/wipnote:feature-complete [id]` - Mark feature done
- `/wipnote:feature-primary [id]` - Set primary focus
- `/wipnote:spike <title>` - Create research spike
- `/wipnote:init` - Initialize `.wipnote/` for project

**Workflow:**
- `/wipnote:start` - Begin session (manual)
- `/wipnote:end` - End session (manual)
- `/wipnote:plan [--track ID]` - Show next work plan
- `/wipnote:track <title>` - Create work track

**Analytics & Insights:**
- `/wipnote:recommend` - Smart recommendations (what to work on)
- `/wipnote:status` - Session/project status
- `/wipnote:research [query]` - Research a topic
- `/wipnote:deploy [version]` - Deployment workflow

**Infrastructure:**
- `/wipnote:serve` - Start dashboard (local HTTP)
- `/wipnote:help [topic]` - Get help

### 5.2 Command Structure

**YAML Frontmatter** (per command file)
```yaml
---
# /wipnote:feature-add

title: "Add a new feature to the backlog"
description: "Create a feature and optionally start working on it"
category: "work-management"
default_args: []  # Default if no args provided
examples:
  - "/wipnote:feature-add User Authentication"
  - "/wipnote:feature-add"  # Prompts for title
---

# Instructions for Claude
[Implementation in shell/Python]
```

**Implementation:** Each command is a markdown file with:
- YAML frontmatter for metadata
- Instructions for Claude to execute
- Bash/Python implementation details

---

## 6. Default Behavior vs. Customization

### 6.1 Default Behavior (No Configuration)

| Feature | Default | Source |
|---|---|---|
| `.wipnote` location | Current project directory | Auto-discovery |
| Work item types | feature, bug, spike, chore, epic, phase | SDK enums |
| Feature status values | todo, in-progress, blocked, done | Node model defaults |
| Session auto-tracking | Enabled (via hooks) | Plugin hooks active |
| Drift detection | Enabled | `drift-config.json` |
| Drift threshold | 0.7 (warning), 0.85 (auto-classify) | Config defaults |
| Feature attribution | Automatic (file patterns + keywords) | SessionManager weights |
| WIP limit | 3 features max | SessionManager.DEFAULT_WIP_LIMIT |

### 6.2 Customization Points

**1. Project-Level Config** (`.claude/config/`)
```json
// drift-config.json
{
  "drift_detection": {
    "enabled": true,
    "warning_threshold": 0.7,
    "auto_classify_threshold": 0.85,
    "min_activities_before_classify": 3,
    "cooldown_minutes": 10
  },
  "classification": {
    "enabled": true,
    "use_haiku_agent": true
  }
}
```

**2. Plugin Settings** (`.claude/plugin-name.local.md`)
```yaml
---
# .claude/wipnote.local.md
version: "0.24.1"
enabled: true
wip_limit: 5
session_timeout_minutes: 60
auto_track_sessions: true
---

Project-specific settings for Wipnote
```

**3. Programmatic Configuration** (SDK)
```python
sdk = SDK(
    agent="claude",
    parent_session="sess-001",  # Parent session context
    wip_limit=5,                # Custom WIP limit
    session_dedupe_window=120   # Deduplication window
)
```

**4. Hook Disabling**
```bash
# Disable all tracking
export HTMLGRAPH_DISABLE_TRACKING=1

# Or in Claude settings
# Set HTMLGRAPH_DISABLE_TRACKING env var
```

**5. Feature Builder Extensions**
```python
# Custom feature properties
feature = sdk.features.create("Auth System") \
    .set_required_capabilities(["python", "security"]) \
    .add_capability_tags(["backend", "api"]) \
    .set_priority("critical") \
    .save()

# Custom attributes
feature.custom_field = "custom_value"
feature.save()
```

---

## 7. File Locations & State Organization

### 7.1 State Files Map

```
.wipnote/
├── features/                      # Feature work items
│   ├── feat-001.html
│   ├── feat-002.html
│   └── ...
├── bugs/                          # Bug reports
│   ├── bug-001.html
│   └── ...
├── spikes/                        # Research spikes
│   ├── spk-001.html
│   └── ...
├── chores/                        # Maintenance
│   ├── chore-001.html
│   └── ...
├── epics/                         # Large initiatives
│   ├── epic-001.html
│   └── ...
├── phases/                        # Project phases
│   ├── phase-001.html
│   └── ...
├── tracks/                        # Work tracks
│   ├── track-001.html
│   ├── track-002/
│   │   └── index.html             # Directory-based tracks
│   └── ...
├── sessions/                      # Session summaries
│   ├── sess-001.html
│   ├── sess-002.html
│   └── ...
├── events/                        # Event log (Git-friendly)
│   ├── 2026-01-05.jsonl
│   ├── 2026-01-04.jsonl
│   └── ...
├── agents.json                    # Agent registry
├── drift-queue.json               # Pending drift classifications
├── parent-activity.json           # Active parent session tracking
├── index.sqlite                   # Cached SQLite index (git-ignored)
├── .wipnote/                    # Nested tracking
│   ├── sessions/*.html
│   └── events/*.jsonl
└── config/
    ├── drift-config.json          # Customizable drift behavior
    └── validation-config.json     # Work item validation rules
```

### 7.2 Project-Level Config

```
.claude/
├── config/
│   ├── drift-config.json          # Drift detection thresholds
│   ├── validation-config.json     # Work item validation
│   └── classification-prompt.md   # Drift classification prompt
├── wipnote.local.md             # Plugin settings (user/project-specific)
├── hooks/
│   ├── hooks.json                 # Hook configuration
│   └── scripts/
│       ├── session-start.py
│       ├── session-end.py
│       ├── pretooluse-integrator.py
│       ├── posttooluse-integrator.py
│       ├── post-tool-use-failure.py
│       └── ...
└── settings.json                  # General Claude Code settings
```

---

## 8. Plugin Hooks & System Integration

### 8.1 Hook Registration Pattern

**Plugin declares hooks** in `plugin.json`:
```json
{
  "hooks": {
    "SessionStart": [{
      "matcher": "",
      "hooks": [{
        "type": "command",
        "command": "uv run \"${CLAUDE_PLUGIN_ROOT}/hooks/scripts/session-start.py\""
      }]
    }],
    ...
  }
}
```

**Hooks are merged** from multiple sources:
1. Plugin hooks (from plugin.json)
2. Project hooks (.claude/hooks/hooks.json)
3. User hooks (Claude Code settings)

**Critical:** Hooks MERGE, not replace. Duplicates can occur if not cleaned up.

### 8.2 Hook Execution Context

Each hook receives:
- `CLAUDE_PROJECT_DIR` - Project root
- `CLAUDE_PLUGIN_ROOT` - Plugin root
- `HTMLGRAPH_HOOK_TYPE` - Hook type being executed
- `HTMLGRAPH_PARENT_SESSION` - Parent session ID (for nested work)
- Standard input (JSON-encoded hook data)

### 8.3 Environment Variables for Control

| Variable | Effect |
|---|---|
| `HTMLGRAPH_DISABLE_TRACKING` | Disable all Wipnote tracking |
| `HTMLGRAPH_HOOK_TYPE` | Current hook being executed |
| `HTMLGRAPH_PARENT_SESSION` | Parent session ID for nested spawning |
| `HTMLGRAPH_PARENT_AGENT` | Parent agent name |
| `CLAUDE_PLUGIN_ROOT` | Plugin root directory |
| `CLAUDE_PROJECT_DIR` | Project root directory |

---

## 9. Multi-AI Orchestration Support

### 9.1 Event Tracking for Delegated Work

When work is delegated to another AI via `HeadlessSpawner`:

```python
# Phase 1: Track delegation
spawner = HeadlessSpawner()
result = spawner.spawn_gemini(prompt="Analyze code", ...)

# Events include delegation metadata
event = EventRecord(
    delegated_to_ai="gemini",
    task_id="task-001",
    task_status="completed",
    model_selected="gemini-2.0-flash",
    complexity_level="medium",
    execution_duration_seconds=15.3,
    tokens_estimated=2000,
    tokens_actual=1847,
    cost_usd=0.002,
    task_findings="Found 3 security issues"
)
```

### 9.2 Parent Session Context

Nested agents maintain context:
```python
# Parent agent creates spawner
sdk = SDK(agent="claude")
spawner = HeadlessSpawner()

# Child agent receives parent session
os.environ["HTMLGRAPH_PARENT_SESSION"] = sdk.session_id
os.environ["HTMLGRAPH_PARENT_AGENT"] = "claude"

result = spawner.spawn_gemini(...)
# Child automatically links events to parent session
```

---

## 10. System Prompt Integration Recommendations

### 10.1 Feature Exposure in System Prompt

**When to mention features:**
1. **SDK/Collections** - For AI agents doing multi-session work
2. **Builders** - For AI agents creating/modifying work items
3. **Analytics** - For decision support and recommendations
4. **Event tracking** - For background automatic tracking

**Sample System Prompt Sections:**

```markdown
## Wipnote Work Tracking

You have access to a work tracking system that organizes work by type:

### Creating Work
```python
from wipnote import SDK

sdk = SDK(agent="claude")

# Create a feature
feature = sdk.features.create("User Auth") \
    .set_priority("high") \
    .add_steps(["Design", "Implement", "Test"]) \
    .save()

# Create a spike for research
spike = sdk.spikes.create("Investigate OAuth options") \
    .set_timebox(4) \
    .save()
```

### Querying Work
```python
# Get high-priority todos
todos = sdk.features.where(status="todo", priority="high")

# Get feature details
feature = sdk.features.get("feat-001")
```

### Smart Recommendations
```python
# What should we work on next?
recommendations = sdk.dep_analytics.recommend_next_tasks(agent_count=1)
bottlenecks = sdk.dep_analytics.find_bottlenecks(top_n=5)
```

### Automatic Tracking
Work is automatically tracked:
- Session start/end (SessionStart/SessionEnd hooks)
- File changes (PostToolUse hook)
- Errors (PostToolUseFailure hook)
```

### 10.2 Key API Surface for System Prompts

**Collections API:**
```python
sdk.features          # Feature collection
sdk.bugs              # Bug collection
sdk.spikes            # Spike collection
sdk.chores            # Chore collection
sdk.epics             # Epic collection
sdk.phases            # Phase collection
sdk.tracks            # Track collection
sdk.todos             # Persistent task list
```

**Analytics API:**
```python
sdk.analytics         # Work type analysis
sdk.dep_analytics     # Dependency analysis (bottlenecks, recommendations)
sdk.context_analytics # Context efficiency tracking
```

**Builder Pattern:**
```python
collection.create(title)              # Returns builder
feature.set_priority("high")          # Chainable
.add_steps(["step 1", "step 2"])     # Chainable
.set_required_capabilities([...])     # Feature-specific
.save()                               # Persists to disk
```

---

## 11. Patterns & Best Practices

### 11.1 SDK Usage Patterns

**Pattern 1: Create & Save**
```python
sdk = SDK(agent="claude")
feature = sdk.features.create("New Feature") \
    .set_priority("high") \
    .add_steps(["Design", "Implement"]) \
    .save()
```

**Pattern 2: Edit with Context Manager**
```python
with sdk.features.edit("feat-001") as feature:
    feature.status = "in-progress"
    feature.complete_step(0)
    # Auto-saves on exit
```

**Pattern 3: Query & Filter**
```python
high_priority_todos = sdk.features.where(
    status="todo",
    priority="high"
)

for feature in high_priority_todos:
    print(feature.title, feature.priority)
```

**Pattern 4: Batch Operations**
```python
sdk.features.mark_done(["feat-001", "feat-002", "feat-003"])
sdk.features.assign("feat-004", agent="claude")
```

### 11.2 Plugin Command Patterns

**Pattern 1: Create from CLI**
```bash
/wipnote:feature-add User Authentication
```

**Pattern 2: Set Primary Focus**
```bash
/wipnote:feature-primary feat-001
/wipnote:feature-start feat-001
```

**Pattern 3: Get Recommendations**
```bash
/wipnote:recommend --count 5
/wipnote:plan --track trk-001
```

### 11.3 Hook Integration Patterns

**Pattern 1: Custom Event Classification**
Update `drift-config.json` to customize drift detection thresholds

**Pattern 2: Hook Disabling**
```bash
export HTMLGRAPH_DISABLE_TRACKING=1
```

**Pattern 3: Custom Session Tracking**
Hooks automatically track sessions; no configuration needed

---

## 12. Customization & Extension Points

### 12.1 User Customization Points (High-Level)

1. **WIP Limits** - SDK parameter or config
2. **Drift Thresholds** - `drift-config.json`
3. **Feature Properties** - Custom attributes via builder
4. **Work Types** - Pre-defined enums (extendable)
5. **Hook Behavior** - Disable via environment variable

### 12.2 Developer Extension Points (Low-Level)

1. **Custom Collections** - Subclass `BaseCollection`
2. **Custom Builders** - Subclass `BaseBuilder`
3. **Custom Analytics** - Subclass `Analytics`
4. **Custom Hooks** - Add to `.claude/hooks/hooks.json`
5. **Custom State Attributes** - Via `properties` dict

---

## 13. State Consistency & Transactions

### 13.1 Consistency Model

**Write Model:**
- Single-file writes (atomic at OS level)
- HTML files: read-modify-write
- Events: append-only (no conflicts)

**Read Model:**
- Lazy-loading (graph loaded on first access)
- Graph caching (within SDK lifetime)
- SQLite index (rebuildable from sources)

### 13.2 Conflict Resolution

**Scenario 1: Concurrent Feature Edits**
- File-based storage prevents simultaneous writes
- Last write wins (typical for single-agent workflow)
- Events append for audit trail

**Scenario 2: Event Log Contention**
- Append-only JSONL is conflict-free
- Each event has unique ID
- Time-ordered for replay-ability

**Scenario 3: Index Rebuild**
```bash
uv run wipnote reindex  # Rebuilds from scratch
```

---

## 14. Migration & Upgrade Paths

### 14.1 Version Management

**Plugin Version Check:**
- At SessionStart, check installed vs. PyPI version
- Warn if outdated (non-blocking)
- Install command provided in warning

**SDK Version:**
```python
import wipnote
print(wipnote.__version__)
```

### 14.2 Backward Compatibility

**Breaking Changes:** Rare, semantic versioning used

**State Migration:** HTML format is extensible
- New attributes don't break old parsers
- Properties dict supports arbitrary fields

---

## 15. Dashboard & Visualization

### 15.1 Dashboard Types

**1. Feature Progress** (HTML visualization)
```
.wipnote/features/feat-001.html
→ Rendered in index.html
→ Viewable in browser via `uv run wipnote serve`
```

**2. Analytics Dashboard** (SQLite-backed)
```
Powered by index.sqlite
Aggregates work across sessions
Provides insights on velocity, drift, efficiency
```

**3. Session Summaries** (HTML)
```
.wipnote/sessions/sess-001.html
Shows work done in session
Links to features, bugs, spikes worked on
```

---

## 16. Critical Design Decisions

| Decision | Rationale | Impact |
|---|---|---|
| HTML as source of truth | Git-friendly, human-readable, no DB | Query via CSS selectors, manual inspection possible |
| JSONL events (append-only) | High-frequency activity, conflict-free | Rebuild-able analytics, chronological audit trail |
| Hook scripts vs. package logic | Thin wrapper pattern | Faster iteration, less duplication |
| Lazy-loading graphs | Memory efficiency, fast startup | Transparent to user, single API |
| Builder pattern | Fluent API, type-safe | More ergonomic than dict construction |
| Collection abstraction | Uniform interface across types | Feature discoverability, reduced API surface |
| Hook merging (not replace) | Support multiple sources | Can cause duplicates if not managed |

---

## 17. Known Limitations & Workarounds

| Limitation | Reason | Workaround |
|---|---|---|
| No database ACID guarantees | File-based storage | Use event log for audit trail, idempotent operations |
| CSS selector queries only | No SQL engine | Use Python graph API for complex queries |
| Hook execution overhead | Every tool invocation | Pre-flight checks could cache results |
| SQLite index rebuild needed | HTML changes don't update index | Run `uv run wipnote reindex` after major changes |
| Single-agent assumption | Current design | Parent session support for nested agents (Phase 1) |

---

## 18. Integration Examples

### 18.1 CLI → SDK Integration

Plugin command internally uses SDK:
```bash
# User runs
/wipnote:feature-add "New Feature"

# Plugin script executes
sdk = SDK(agent="claude")
feature = sdk.features.create("New Feature").save()
```

### 18.2 Hook → SDK Integration

Hook scripts use SDK for state management:
```python
# session-start.py
sdk = SDK(agent="claude")
session = sdk.sessions.create(...).save()
```

### 18.3 Multi-AI → SDK Integration

HeadlessSpawner tracks events:
```python
spawner = HeadlessSpawner()
result = spawner.spawn_gemini(...)
# Automatically creates EventRecord with delegated_to_ai="gemini"
```

---

## 19. Recommendations for System Prompt Integration

### 19.1 SDK Exposure

**Expose full SDK API** to AI agents for:
- ✅ Creating work items (features, bugs, spikes)
- ✅ Querying work (where, get, all)
- ✅ Builder pattern for fluent creation
- ✅ Analytics for decision support
- ✅ Session management (programmatic)

**Example integration:**
```markdown
## Work Tracking (Wipnote SDK)

You can organize work using Wipnote:

from wipnote import SDK
sdk = SDK(agent="your-name")

# Create features
feature = sdk.features.create("Title").set_priority("high").save()

# Query work
todos = sdk.features.where(status="todo", priority="high")

# Get recommendations
next_tasks = sdk.dep_analytics.recommend_next_tasks(agent_count=1)
```

### 19.2 Analytics Exposure

**Expose analytics** for decision-making:
- Bottleneck detection
- Dependency analysis
- Work distribution
- Efficiency metrics

### 19.3 Automatic Tracking Transparency

**Document automatic tracking:**
- Sessions are auto-tracked via hooks
- File edits attributed to features
- Errors captured automatically
- No explicit tracking code needed

### 19.4 Extension Patterns

**Document extensibility:**
- Custom work types (builders)
- Custom properties (attributes)
- Custom analytics (override methods)

---

## Appendix A: File Format Specifications

### A.1 Feature Node HTML Schema

```html
<article id="{id}"
         data-type="feature"
         data-status="{status}"
         data-priority="{priority}"
         data-created="{iso_timestamp}"
         data-updated="{iso_timestamp}"
         data-track-id="{track_id}"
         data-agent-assigned="{agent_name}"
         data-claimed-at="{iso_timestamp}"
         data-claimed-by-session="{session_id}">

    <header>
        <h1>{title}</h1>
        <div class="metadata">
            <span class="badge status-{status}">{status}</span>
            <span class="badge priority-{priority}">{priority}</span>
        </div>
    </header>

    <nav data-graph-edges>
        <section data-edge-type="{type}">
            <h3>{type}:</h3>
            <ul>
                <li><a href="{target_id}.html"
                       data-relationship="{relationship}"
                       data-since="{iso_timestamp}">
                    {target_id}
                </a></li>
            </ul>
        </section>
    </nav>

    <section data-steps>
        <h3>Implementation Steps</h3>
        <ol>
            <li data-completed="{bool}" data-agent="{agent}">
                {status_emoji} {description}
            </li>
        </ol>
    </section>

    <section data-properties>
        <h3>Properties</h3>
        <dl>
            <dt>{key}</dt>
            <dd data-key="{key}" data-value="{value}">
                {value}
            </dd>
        </dl>
    </section>
</article>
```

### A.2 Event Log JSONL Schema

```jsonl
{"event_id": "evt-001", "timestamp": "2026-01-05T12:00:00Z", "session_id": "sess-001", "agent": "claude", "tool": "Edit", "summary": "Edit: src/auth/login.py:45-52", "success": true, "feature_id": "feat-001", "work_type": "feature-implementation", "drift_score": 0.3, "start_commit": "abc123", "file_paths": ["src/auth/login.py"], "delegated_to_ai": null, "task_id": null}
```

---

## Appendix B: Quick Reference

### B.1 SDK Quick Start

```python
from wipnote import SDK

# Initialize
sdk = SDK(agent="your-name")

# Create
feature = sdk.features.create("Title").set_priority("high").save()

# Read
todos = sdk.features.where(status="todo")
feature = sdk.features.get("feat-001")

# Update
with sdk.features.edit("feat-001") as f:
    f.status = "done"

# Analytics
recommendations = sdk.dep_analytics.recommend_next_tasks()
```

### B.2 Plugin Commands Quick Start

```bash
/wipnote:feature-add "New Feature"
/wipnote:feature-start feat-001
/wipnote:feature-primary feat-001
/wipnote:spike "Research topic"
/wipnote:recommend
/wipnote:status
/wipnote:serve
```

### B.3 State Locations Quick Reference

```
Python SDK      → .wipnote/
Plugin Commands → .wipnote/ (via SDK)
Hook Scripts    → .wipnote/ (via SDK)
Events          → .wipnote/events/*.jsonl
Features        → .wipnote/features/*.html
Sessions        → .wipnote/sessions/*.html
Config          → .claude/config/*.json
```

---

**Document Complete**

This architecture document represents Wipnote's feature exposure strategy as of v0.24.1. For the latest information, refer to:
- `src/python/wipnote/sdk.py` - SDK API
- `packages/claude-plugin/README.md` - Plugin installation
- Source code comments for implementation details
