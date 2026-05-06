# User Guide

Welcome to the Wipnote user guide. This section provides in-depth documentation for all Wipnote features.

## What You'll Learn

This guide covers:

- **[Features & Tracks](features-tracks.md)** - Creating and managing features and tracks
- **[TrackBuilder](track-builder.md)** - Mastering the TrackBuilder fluent API
- **[Delegation](delegation.md)** - Distributing work to parallel subagents with Task()
- **[Skills](skills.md)** - Specialized guides for orchestration, deployment, and debugging
- **[Session Hierarchies](session-hierarchies.md)** - Understanding parent-child session relationships
- **[Sessions](sessions.md)** - Understanding session tracking and attribution
- **[Agents](agents.md)** - Integrating Wipnote with AI agents
- **[Dashboard](dashboard.md)** - Using the interactive dashboard

## Quick Navigation

### For Beginners

Start with these guides in order:

1. [Features & Tracks](features-tracks.md) - Learn the basics
2. [Dashboard](dashboard.md) - Visualize your work
3. [Sessions](sessions.md) - Understand activity tracking

### For Agent Developers

Jump to these sections:

1. [Delegation](delegation.md) - Task distribution and orchestration patterns
2. [Skills](skills.md) - Specialized guides for complex workflows
3. [Session Hierarchies](session-hierarchies.md) - Parent-child session relationships
4. [Agents](agents.md) - Agent integration patterns
5. [TrackBuilder](track-builder.md) - Deterministic track creation
6. [Sessions](sessions.md) - Session management and attribution

### For Power Users

Advanced topics:

1. [Skills](skills.md) - Master all specialized guides and decision frameworks
2. [Session Hierarchies](session-hierarchies.md) - Advanced session querying and analysis
3. [Delegation](delegation.md) - Complex delegation patterns and cost optimization
4. [TrackBuilder](track-builder.md) - Complex track workflows
5. [Sessions](sessions.md) - Custom session handling
6. [API Reference](../api/index.md) - Complete API documentation

## Common Workflows

### Creating a Simple Feature

```bash
wipnote feature create "Add login page" --priority high
```

[Learn more →](features-tracks.md#creating-features)

### Creating a Complex Track

```bash
wipnote track new "User Authentication" --priority high
# Note the track ID (e.g. trk-a1b2c3d4)
```

[Learn more →](track-builder.md)

### Linking Features to Tracks

```bash
# Create features linked to the track
wipnote feature create "OAuth Setup" --priority high --track trk-a1b2c3d4
wipnote feature create "JWT Middleware" --priority high --track trk-a1b2c3d4
```

[Learn more →](features-tracks.md#linking-features-to-tracks)

### Starting a Session

Sessions are automatically managed by Wipnote hooks:

```bash
# Session starts automatically when you begin working
wipnote feature start feature-001

# View session status
wipnote status

# Session ends automatically when you complete the feature
wipnote feature complete feature-001
```

[Learn more →](sessions.md)

## Need Help?

- Check the [API Reference](../api/index.md) for detailed SDK documentation
- Browse [Examples](../examples/index.md) for real-world use cases
- Read the [Philosophy](../philosophy/why-html.md) to understand design decisions
- Visit [GitHub Discussions](https://github.com/shakestzd/wipnote/discussions) for community support
