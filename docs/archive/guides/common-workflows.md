# Common Workflows

Practical patterns for using Wipnote SDK in AI agent sessions.

## Session Start

```python
from wipnote import SDK

sdk = SDK(agent="claude")
info = sdk.get_session_start_info()

# Check for active work
if info.get("active_work"):
    print(f"Continue: {info['active_work']['title']}")
else:
    # Get recommendations
    recs = info.get("analytics", {}).get("recommendations", [])
    if recs:
        print(f"Recommended: {recs[0]['title']}")
```

**What this does:**
- Loads session context automatically
- Identifies in-progress work to continue
- Provides strategic recommendations if starting fresh
- Returns analytics about project health

## Creating Work Items

### Features (Implementation Work)

```python
# Simple feature creation
feature = sdk.features.create("Add user authentication")

# With full configuration
feature = sdk.features.create("Add user authentication") \
    .set_priority("high") \
    .set_status("in-progress") \
    .add_steps([
        "Create auth routes",
        "Add middleware",
        "Write tests"
    ]) \
    .add_metadata({
        "estimated_hours": 4,
        "requires_review": True
    }) \
    .save()

print(f"Created feature: {feature.id}")
```

**When to use:**
- New features or enhancements
- Multi-step implementation work
- Work that adds new capabilities

### Bugs (Fixes)

```python
# Simple bug report
bug = sdk.bugs.create("Login fails on timeout")

# With details
bug = sdk.bugs.create("Login fails on timeout") \
    .set_severity("high") \
    .set_priority("urgent") \
    .add_steps([
        "Reproduce timeout scenario",
        "Add timeout handling",
        "Test edge cases"
    ]) \
    .add_metadata({
        "error_message": "ConnectionTimeout in auth.py:45",
        "affects_version": "0.9.3"
    }) \
    .save()
```

**When to use:**
- Fixing broken functionality
- Resolving errors or exceptions
- Addressing production issues

### Spikes (Research)

```python
# Research spike with timebox
spike = sdk.spikes.create("Evaluate caching options") \
    .set_timebox_hours(2) \
    .set_priority("medium") \
    .add_steps([
        "Review Redis vs Memcached",
        "Test performance",
        "Document findings"
    ]) \
    .save()
```

**When to use:**
- Investigating unknowns
- Proof-of-concept work
- Evaluating alternatives
- Time-boxed research

## Orchestration Pattern

### Explore Then Implement

```python
# 1. Spawn explorer for discovery
explorer = sdk.spawn_explorer(
    task="Understand authentication system",
    scope="src/auth/"
)

# Use with Task tool (pseudo-code for illustration)
# Task(prompt=explorer["prompt"], subagent_type=explorer["subagent_type"])

# 2. After exploration, spawn coder with context
coder = sdk.spawn_coder(
    feature_id=feature.id,
    context=explorer_results,  # Pass findings from explorer
    test_command="uv run pytest tests/auth/"
)

# Task(prompt=coder["prompt"], subagent_type=coder["subagent_type"])
```

**Pattern benefits:**
- Separates discovery from implementation
- Reduces context switching
- Improves code quality through focused exploration

### Full Orchestration

```python
# Automatic exploration → implementation flow
prompts = sdk.orchestrate(
    feature_id="feat-123",
    exploration_scope="src/",
    test_command="uv run pytest"
)

# Returns both explorer and coder prompts
# prompts = {
#     "explorer": {"prompt": "...", "subagent_type": "explorer"},
#     "coder": {"prompt": "...", "subagent_type": "coder"}
# }

# Execute in sequence:
# 1. Task(prompt=prompts["explorer"]["prompt"], ...)
# 2. After explorer completes, use results in coder context
# 3. Task(prompt=prompts["coder"]["prompt"], ...)
```

**When to use:**
- Complex features requiring research
- Unfamiliar codebases
- When exploration findings inform implementation

## Parallel Execution

```python
# Check if work can be parallelized
parallel = sdk.get_parallel_work(max_agents=3)

if parallel["can_parallelize"]:
    print(f"Found {len(parallel['prompts'])} parallelizable tasks")

    # Spawn multiple subagents in one message
    for p in parallel["prompts"]:
        # Task(prompt=p["prompt"], subagent_type=p["subagent_type"])
        print(f"- {p['work_item']['title']}")
else:
    print("No parallelizable work found")
    print(f"Reason: {parallel.get('reason', 'Unknown')}")
```

**Parallelization criteria:**
- Work items have no blocking dependencies
- Different codebases/modules
- Independent test suites
- No shared state mutations

**Example parallel scenarios:**
- Bug fixes in different modules
- Independent feature implementations
- Documentation updates
- Separate test suites

## Progress Tracking

### Starting Work

```python
# Mark feature as in-progress
sdk.features.start(feature.id)

# Automatically logs:
# - Start timestamp
# - Agent assignment
# - Session linkage
```

### Completing Steps

```python
# Complete individual steps
with sdk.features.edit(feature.id) as f:
    f.complete_step(0)  # "Create auth routes" ✓
    f.complete_step(1)  # "Add middleware" ✓

# Or complete with notes
with sdk.features.edit(feature.id) as f:
    f.complete_step(2, notes="All tests passing")
```

### Updating Status

```python
# Manual status update
with sdk.features.edit(feature.id) as f:
    f.set_status("blocked")
    f.add_metadata({"blocked_by": "Waiting for API key"})

# Or use helper methods
sdk.features.block(feature.id, reason="Missing credentials")
```

### Marking Complete

```python
# Complete the feature
sdk.features.complete(feature.id)

# With completion notes
sdk.features.complete(
    feature.id,
    notes="All tests passing, ready for review"
)

# Automatically:
# - Sets status to "done"
# - Marks all steps complete
# - Records completion timestamp
# - Logs to session
```

## Session End

```python
# End session with handoff notes
sdk.end_session(
    session_id=session.id,
    handoff_notes="Completed auth feature, all tests passing",
    recommended_next="Implement rate limiting"
)

# Or use auto-generated summary
summary = sdk.get_session_summary()
sdk.end_session(
    session_id=session.id,
    handoff_notes=summary["summary"],
    recommended_next=summary["next_steps"]
)
```

**Session end checklist:**
- ✓ Commit all changes
- ✓ Run tests
- ✓ Document completion status
- ✓ Note any blockers
- ✓ Suggest next work

## Strategic Analytics

### Getting Recommendations

```python
# Get work recommendations
recs = sdk.analytics.recommend_next_work(max_items=5)

for rec in recs:
    print(f"{rec['priority']}: {rec['title']}")
    print(f"  Reason: {rec['reason']}")
    print(f"  Ready: {rec['is_ready']}")
```

**Recommendation criteria:**
- Priority and severity
- Unblocked dependencies
- Time sensitivity
- Resource availability

### Finding Bottlenecks

```python
# Identify blocking work items
bottlenecks = sdk.analytics.find_bottlenecks()

for item in bottlenecks:
    print(f"{item['title']} blocks {len(item['blocks'])} items")
    for blocked in item['blocks']:
        print(f"  - {blocked['title']}")
```

**Use bottleneck analysis to:**
- Prioritize unblocking work
- Identify dependency chains
- Plan parallel work streams

### Drift Detection

```python
# Check for stale work
drift = sdk.analytics.detect_drift()

if drift["stale_items"]:
    print(f"Found {len(drift['stale_items'])} stale items:")
    for item in drift["stale_items"]:
        print(f"- {item['title']} (idle {item['days_idle']} days)")
```

## Complete Example: Feature Implementation

```python
from wipnote import SDK

# Initialize
sdk = SDK(agent="claude")

# 1. Check session context
info = sdk.get_session_start_info()
if info.get("active_work"):
    feature = sdk.features.get(info["active_work"]["id"])
    print(f"Continuing: {feature.title}")
else:
    # 2. Create new feature
    feature = sdk.features.create("Add rate limiting") \
        .set_priority("high") \
        .add_steps([
            "Design rate limit strategy",
            "Implement middleware",
            "Add Redis integration",
            "Write tests",
            "Update documentation"
        ]) \
        .save()

    # 3. Start work
    sdk.features.start(feature.id)

# 4. Orchestrate exploration + implementation
prompts = sdk.orchestrate(
    feature_id=feature.id,
    exploration_scope="src/middleware/",
    test_command="uv run pytest tests/middleware/"
)

# Execute explorer first, then coder
# (Implementation depends on your agent framework)

# 5. Track progress
with sdk.features.edit(feature.id) as f:
    f.complete_step(0)  # Design done
    f.complete_step(1)  # Middleware done
    # ... continue as work progresses

# 6. Complete feature
sdk.features.complete(
    feature.id,
    notes="Rate limiting implemented, all tests passing"
)

# 7. End session
sdk.end_session(
    session_id=sdk.session.id,
    handoff_notes="Rate limiting complete, ready for review",
    recommended_next="Add monitoring dashboards"
)
```

## Best Practices

### Do's ✓
- Use `get_session_start_info()` at session start
- Create work items before starting implementation
- Track progress with step completion
- Use orchestration for complex features
- End sessions with handoff notes
- Check analytics for strategic guidance

### Don'ts ✗
- Don't create duplicate work items
- Don't skip session tracking
- Don't ignore bottleneck warnings
- Don't forget to mark work complete
- Don't bypass orchestration for complex work

## Troubleshooting

### "No session found"
```python
# Ensure SDK is initialized
sdk = SDK(agent="claude")  # Creates session automatically
```

### "Feature not found"
```python
# List all features
features = sdk.features.list()
print([f.id for f in features])
```

### "Cannot parallelize work"
```python
# Check dependency graph
bottlenecks = sdk.analytics.find_bottlenecks()
# Resolve blocking items first
```

## Next Steps

- **[SDK Reference](../api/sdk.md)** - Complete SDK documentation
- **[Orchestration Guide](orchestration.md)** - Deep dive into agent coordination
- **[Analytics Guide](analytics.md)** - Strategic planning with analytics
- **[Examples](../examples/)** - Real-world usage examples
