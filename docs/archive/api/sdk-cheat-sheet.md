# Wipnote SDK Cheat Sheet

Quick reference for AI agents and developers.

## Initialization

```python
from wipnote import SDK
sdk = SDK(agent="claude")  # Auto-discovers .wipnote directory
```

## Work Items

| Operation | Features | Bugs | Spikes |
|-----------|----------|------|--------|
| Create | `sdk.features.create("Title").save()` | `sdk.bugs.create("Title").save()` | `sdk.spikes.create("Title").save()` |
| Get | `sdk.features.get(id)` | `sdk.bugs.get(id)` | `sdk.spikes.get(id)` |
| List | `sdk.features.all()` | `sdk.bugs.all()` | `sdk.spikes.all()` |
| Start | `sdk.features.start(id)` | `sdk.bugs.start(id)` | `sdk.spikes.start(id)` |
| Complete | `sdk.features.complete(id)` | `sdk.bugs.complete(id)` | `sdk.spikes.complete(id)` |
| Where | `sdk.features.where(status="todo")` | `sdk.bugs.where(priority="high")` | `sdk.spikes.where(status="done")` |

**Other Work Types:** `sdk.chores`, `sdk.epics`, `sdk.phases` (same operations)

## Fluent Builder

```python
# Features
feature = sdk.features.create("Add Authentication") \
    .set_priority("high")      # high, medium, low
    .set_status("in-progress") # todo, in-progress, done
    .add_steps(["Design schema", "Implement API", "Add tests"]) \
    .save()

# Bugs
bug = sdk.bugs.create("Login button broken") \
    .set_priority("critical") \
    .set_severity("high") \
    .add_steps(["Reproduce", "Fix", "Test"]) \
    .save()

# Spikes
spike = sdk.spikes.create("Research auth options") \
    .set_spike_type("architectural") \
    .set_timebox_hours(4) \
    .add_steps(["Research OAuth", "Compare providers"]) \
    .save()
```

## Orchestration

| Method | Purpose | Returns |
|--------|---------|---------|
| `sdk.spawn_explorer(task, scope)` | Research codebase | `{prompt, description, subagent_type}` |
| `sdk.spawn_coder(feature_id, context)` | Implement changes | `{prompt, description, subagent_type}` |
| `sdk.orchestrate(feature_id, scope)` | Full workflow | `{explorer: {...}, coder: {...}}` |

```python
# Spawn explorer for research
explorer = sdk.spawn_explorer(
    task="Find all database models",
    scope="src/models/",
    questions=["What ORM is used?"]
)
# Use with: Task(prompt=explorer["prompt"], description=explorer["description"])

# Spawn coder for implementation
coder = sdk.spawn_coder(
    feature_id="feat-add-auth",
    context="Explorer found 3 models using SQLAlchemy",
    test_command="uv run pytest tests/auth/"
)

# Full orchestration (2-phase: explore then code)
prompts = sdk.orchestrate(
    feature_id="feat-add-caching",
    exploration_scope="src/cache/",
    test_command="uv run pytest tests/cache/"
)
# Returns: {explorer: {...}, coder: {...}, workflow: [...]}
```

## Analytics

```python
# Find blocking tasks
bottlenecks = sdk.find_bottlenecks(top_n=5)
# Returns: [{"id": "feat-001", "title": "...", "blocking_count": 3}, ...]

# Get smart recommendations
recommendations = sdk.recommend_next_work(agent_count=1)
# Returns: [{"id": "feat-002", "title": "...", "reason": "unblocked"}, ...]

# Find parallelizable tasks
parallel = sdk.get_parallel_work(max_agents=3)
# Returns: {"prompts": [...], "count": 3, "tasks": [...]}
```

## Session Management

```python
# Get comprehensive session start info (1 call vs 6+)
info = sdk.get_session_start_info()
# Returns: {
#   "status": {...},           # Project status
#   "active_work": {...},      # Current WIP item
#   "features": [...],         # All features
#   "sessions": [...],         # Recent sessions
#   "git_log": [...],          # Recent commits
#   "analytics": {             # Strategic insights
#     "bottlenecks": [...],
#     "recommendations": [...],
#     "parallel_work": {...}
#   }
# }

# Check active work
if info.get("active_work"):
    work = info["active_work"]
    print(f"Working on: {work['title']}")

# End session with handoff
sdk.end_session(
    session_id="session-123",
    handoff_notes="Completed auth implementation, tests passing"
)
```

## Planning Workflow

```python
# Smart planning with automated research
plan = sdk.smart_plan(
    goal="Add user authentication",
    scope="src/",
    max_parallel_spikes=3
)
# Auto-creates: Planning spike, research spikes, track with phases

# Create track from plan
track = sdk.create_track_from_plan(
    plan_feature_id="feat-plan-auth",
    track_name="Authentication System"
)
# Converts planning steps into track phases

# Plan parallel work
parallel_plan = sdk.plan_parallel_work(
    feature_ids=["feat-001", "feat-002", "feat-003"],
    max_agents=3
)
# Returns prompts for parallel execution
```

## Help

```python
sdk.help()              # List all topics
sdk.help("features")    # Feature-specific help
sdk.help("orchestrate") # Orchestration help
sdk.help("analytics")   # Analytics help
sdk.help("planning")    # Planning workflow help
```

## Common Patterns

### Check Active Work
```python
info = sdk.get_session_start_info()
if info.get("active_work"):
    work = info["active_work"]
    print(f"Resume: {work['title']} ({work['status']})")
else:
    print("No active work, check recommendations")
    for rec in info["analytics"]["recommendations"]:
        print(f"  - {rec['title']} ({rec['reason']})")
```

### Create Before Code
```python
# Always create tracking before implementing
feature = sdk.features.create("Add authentication") \
    .set_priority("high") \
    .add_steps(["Design schema", "Implement API", "Add tests"]) \
    .save()

# Start work
sdk.features.start(feature.id)

# ... do implementation work ...

# Complete when done
sdk.features.complete(feature.id)
```

### Parallel Subagents
```python
# Get parallel work suggestions
parallel = sdk.get_parallel_work(max_agents=3)

# Spawn subagents for each task
for p in parallel["prompts"]:
    Task(
        prompt=p["prompt"],
        subagent_type=p["subagent_type"],
        description=p["description"]
    )
```

### Query and Filter
```python
# Get specific items
high_priority = sdk.features.where(status="todo", priority="high")
critical_bugs = sdk.bugs.where(priority="critical", status="todo")
active_spikes = sdk.spikes.where(status="in-progress")

# Get all items
all_features = sdk.features.all()
all_chores = sdk.chores.all()

# Iterate and process
for feature in high_priority:
    print(f"{feature.id}: {feature.title} - {feature.status}")
```

### Session Context
```python
# Start of session - get full context
info = sdk.get_session_start_info()

print(f"Project: {info['status']['total_nodes']} total nodes")
print(f"WIP: {info['status']['in_progress_count']} in progress")

# Check git activity
for commit in info['git_log'][:3]:
    print(f"  {commit}")

# Check bottlenecks
for bn in info['analytics']['bottlenecks']:
    print(f"⚠️  {bn['title']} (blocking {bn['blocking_count']} items)")
```

## Quick Tips

- **Always check return values**: SDK methods return `None` or empty results on errors
- **Use fluent builders**: Chain methods for cleaner code
- **Leverage analytics**: Let SDK recommend what to work on next
- **Track before implementing**: Create features/bugs before coding
- **Use session start info**: Single call for comprehensive context
- **Check active work first**: Resume WIP before starting new tasks
- **Orchestrate complex features**: Use spawn_explorer → spawn_coder workflow
- **Plan strategically**: Use smart_plan for large initiatives

## Error Handling

```python
# Always check before using
feature = sdk.features.get("nonexistent")
if feature:
    print(feature.title)
else:
    print("Feature not found")

# Safe iteration
features = sdk.features.where(status="todo")
for f in features:  # Empty list if none found
    print(f.title)
```

## Advanced Features

### Track Building
```python
from wipnote import TrackBuilder

track = TrackBuilder(sdk, "Authentication System") \
    .add_phase("Research", spike_ids=["spike-001"]) \
    .add_phase("Implementation", feature_ids=["feat-001", "feat-002"]) \
    .add_phase("Testing", feature_ids=["feat-003"]) \
    .save()
```

### Context Analytics
```python
# Track context usage
sdk.context.track_context_use(
    event_id="evt-123",
    method_name="get_session_start_info",
    token_savings=5000
)

# Get savings report
report = sdk.context.get_context_savings()
```

### Dependency Analysis
```python
# Advanced graph analytics
graph = sdk.dep_analytics

# Find circular dependencies
cycles = graph.find_cycles()

# Critical path analysis
critical = graph.find_critical_path("feat-001", "feat-020")

# Impact analysis
impact = graph.analyze_impact("feat-001")
```

## See Also

- **[AGENTS.md](./AGENTS.md)** - Complete SDK documentation
- **[PLANNING_WORKFLOW.md](./PLANNING_WORKFLOW.md)** - Strategic planning guide
- **[PARALLEL_WORKFLOW.md](./PARALLEL_WORKFLOW.md)** - Parallel execution patterns
- **[SDK_ANALYTICS.md](./SDK_ANALYTICS.md)** - Analytics deep-dive
