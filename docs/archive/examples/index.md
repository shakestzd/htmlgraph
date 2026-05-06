# Examples

Real-world examples of using Wipnote.

## Examples by Type

### Getting Started
- **[Basic Usage](basic.md)** - Simple feature creation and management
- **[Track Creation](tracks.md)** - Creating tracks with specs and plans
- **[Agent Workflows](agents.md)** - Agent integration patterns

### Multi-Agent Orchestration
- **[Multi-Agent Workflow](multi-agent-workflow.md)** - Real-world: Parallel delegation of backend, frontend, and testing work (⭐ **Start here for orchestration**)
- **[Agent Coordination](agent-coordination.md)** - Two-phase explorer+coder pattern

## Use Cases

### Personal Knowledge Base

Use Wipnote to manage notes and ideas:

```python
from wipnote import SDK

sdk = SDK(agent="me")

# Create a note
note = sdk.features.create(
    title="Research: Graph Databases",
    properties={"type": "note", "category": "research"}
)

# Link related notes
sdk.features.add_edge(
    from_id=note.id,
    to_id="feature-another-note",
    relationship="related"
)
```

### Project Management

Track tasks and deliverables:

```python
# Create a project track
project = sdk.tracks.builder() \
    .title("Website Redesign") \
    .with_plan_phases([
        ("Phase 1: Design", [
            "Create mockups (8h)",
            "Review with stakeholders (2h)"
        ]),
        ("Phase 2: Development", [
            "Implement frontend (20h)",
            "Backend API (15h)"
        ])
    ]) \
    .create()

# Create features for each deliverable
for phase in project.plan.phases:
    for task in phase.tasks:
        sdk.features.create(
            title=task.description,
            track_id=project.track_id
        )
```

### Agent Coordination

Coordinate multiple AI agents:

```python
# Agent 1 creates feature
claude = SDK(agent="claude")
feature = claude.features.create("Implement auth")
feature.assigned_agent = "claude"
feature.save()

# Agent 1 completes part of the work
feature.steps[0].completed = True
feature.handoff_notes = "OAuth configured. JWT implementation next."
feature.assigned_agent = "gemini"
feature.save()

# Agent 2 picks up
gemini = SDK(agent="gemini")
feature = gemini.features.get(feature.id)
print(feature.handoff_notes)  # See what Agent 1 did
```

## Complete Examples

Browse the example implementations:

- [Todo List](https://github.com/shakestzd/wipnote/tree/main/examples/todo-list)
- [Agent Coordination](https://github.com/shakestzd/wipnote/tree/main/examples/agent-coordination)
- [Knowledge Base](https://github.com/shakestzd/wipnote/tree/main/examples/knowledge-base)

## Next Steps

- [Basic Usage Examples](basic.md) - Start here
- [Agent Workflows](agents.md) - Agent integration
- [Track Creation](tracks.md) - Complex track examples
