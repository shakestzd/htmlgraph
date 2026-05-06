# Agent Strategic Planning Guide

This guide shows AI agents how to use Wipnote's strategic planning and dependency analytics features to make smart decisions about what to work on.

## Quick Start

```bash
# Get smart recommendations
wipnote analytics recommend

# Find bottlenecks
wipnote analytics bottlenecks
```

## Available Features

### 1. Find Bottlenecks 🚧

**What it does**: Identifies tasks blocking the most downstream work

**When to use**:
- At the start of a work session
- When planning sprints
- When coordinating multiple agents

**Example**:
```bash
wipnote analytics bottlenecks --top 5
```

### 2. Get Parallel Work ⚡

**What it does**: Finds tasks that can be worked on simultaneously by multiple agents

**When to use**:
- When coordinating multiple agents
- When planning team assignments
- When looking for independent work streams

**Example**:
```bash
wipnote analytics recommend --agent-count 5
```

### 3. Recommend Next Work 💡

**What it does**: Provides smart recommendations on what to work on next, considering priority, dependencies, and impact

**When to use**:
- When deciding what task to pick up
- When you need to prioritize between multiple options
- When coordinating work across agents

**Example**:
```bash
wipnote analytics recommend --agent-count 3
```

### 4. Assess Risks ⚠️

**What it does**: Identifies dependency-related risks like single points of failure, circular dependencies, and orphaned tasks

**When to use**:
- During project health checks
- Before starting a sprint
- When dependencies feel complex

**Example**:
```bash
# View snapshot for an overview of project health
wipnote snapshot --summary
```

### 5. Analyze Impact 📊

**What it does**: Shows what downstream work will be unblocked by completing a specific task

**When to use**:
- Before committing to a large task
- When deciding between tasks with similar priority
- When communicating value of work

**Example**:
```bash
wipnote analytics bottlenecks
# Shows impact scores and blocks_count for each bottleneck
```

## Decision Flow for AI Agents

Here's a recommended decision flow for AI agents:

```bash
# 1. Check for bottlenecks
wipnote analytics bottlenecks --top 3

# 2. Get smart recommendations
wipnote analytics recommend --agent-count 1

# 3. Start work on the recommended task
wipnote feature start feat-<id>

# 4. Check for parallel work (if coordinating with other agents)
wipnote analytics recommend --agent-count 3

# 5. Periodic health check
wipnote snapshot --summary
```

## Best Practices

1. **Start with recommendations**: Use `recommend_next_work()` as your starting point
2. **Check bottlenecks regularly**: At least once per session or sprint
3. **Assess risks periodically**: Before major milestones
4. **Analyze impact for big decisions**: When choosing between high-effort tasks
5. **Use parallel work for coordination**: When multiple agents are available

## Performance Notes

- All analytics queries are O(N) or O(N log N) where N = number of nodes
- Results are computed on-demand (no caching)
- For large graphs (1000+ nodes), consider:
  - Limiting `top_n` parameters
  - Filtering by status before analysis
  - Using the lower-level API for fine-grained control

## Examples

See `demo_agent_planning.py` for a complete working example.

## See Also

- [SDK Documentation](./SDK_FOR_AI_AGENTS.md)
- [Agent Interface](./AGENTS.md)
- [Dependency Analytics API](./API_REFERENCE.md#dependency-analytics)
