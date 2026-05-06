# Agents API

Agent integration interface.

## AgentInterface

Simplified API for AI agents to interact with Wipnote.

```python
from wipnote.agents import AgentInterface

agent = AgentInterface(
    graph_dir=".wipnote",
    agent_id="claude"
)
```

## Getting Tasks

```python
# Get next available task
task = agent.get_next_task(
    filters={'priority': 'high', 'status': 'todo'}
)

# Claim the task
agent.claim_task(task.id, agent_id='claude')
```

## Context

Get lightweight context for LLM:

```python
# Get context for a feature
context = agent.get_context(feature_id="feature-001")

# Returns formatted string:
# """
# # feature-001: User Authentication
# Status: in-progress | Priority: high
# Assigned: claude
# Progress: 2/5 steps
# ⚠️  Blocked by: feature-005 (Database Schema)
#
# Next steps:
#   - Implement OAuth flow
#   - Add session management
# """
```

## Progress Updates

```python
# Complete a step
agent.complete_step(
    feature_id="feature-001",
    step_index=0,
    agent_id="claude"
)

# Mark feature complete
agent.complete_task(
    feature_id="feature-001",
    agent_id="claude"
)
```

## Complete API Reference

For detailed API documentation with method signatures, see the Python source code in `src/python/wipnote/agents.py`.
