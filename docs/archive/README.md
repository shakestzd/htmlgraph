# Wipnote Documentation

Welcome to Wipnote - HTML is All You Need. A lightweight graph database built on web standards for AI agent coordination and human observability.

## Quick Start

**New to Wipnote?** Start here:

1. **[Installation](getting-started/installation.md)** - Install and setup
2. **[Quick Start](getting-started/quick-start.md)** - Your first 5 minutes
3. **[Core Concepts](getting-started/concepts.md)** - Understand the basics

## User Paths

### I Want to...

**Build with Wipnote**
- → [SDK Reference](api/sdk.md) - Complete API documentation
- → [Basic Examples](examples/basic.md) - Simple code examples
- → [Features & Tracks Guide](guide/features-tracks.md) - Core concepts

**Coordinate Multiple AI Agents**
- → [Delegation Guide](guide/delegation.md) - How to delegate work
- → [Orchestration Guide](guide/orchestration.md) - Coordinate agents
- → [Multi-Agent Examples](examples/multi-agent-workflow.md) - Real-world example
- → [Session Hierarchies](guide/session-hierarchies.md) - Track parent-child relationships

**Track Development Work**
- → [Track Builder Guide](guide/track-builder.md) - Plan features
- → [Session Management](guide/sessions.md) - Monitor progress
- → [Dashboard Guide](guide/dashboard.md) - Visualize your work

**Deploy & Integrate**
- → [CLI Reference](api/cli.md) - Command-line tools
- → [Hooks Architecture](api/hooks.md) - Event-driven automation
- → [Server Guide](api/server.md) - Run as a service

**Understand the Design**
- → [Architecture Guide](architecture/design.md) - System design
- → [Parent-Child Event Linking](architecture/parent-child-events.md) - How orchestration works
- → [Database Schema](architecture/sqlite-schema.md) - Storage design
- → [Hash-Based IDs](architecture/hash-based-ids.md) - Identifier design

## Documentation Structure

```
docs/
├── getting-started/      [START HERE for new users]
│   ├── installation.md   - Setup and configuration
│   ├── quick-start.md    - Your first 5 minutes
│   └── concepts.md       - Core concepts and terminology
│
├── guide/                [USER WORKFLOWS & PATTERNS]
│   ├── features-tracks.md     - Create and manage features
│   ├── sessions.md            - Monitor sessions and events
│   ├── orchestration.md       - Coordinate multiple agents
│   ├── delegation.md          - Delegate work to subagents
│   ├── session-hierarchies.md - Parent-child relationships
│   ├── track-builder.md       - Plan with TrackBuilder
│   ├── dashboard.md           - Visualize your work
│   ├── queries.md             - Query the graph
│   ├── agents.md              - Agent-specific patterns
│   ├── sqlite-index.md        - Database queries
│   └── migration.md           - Upgrade guides
│
├── api/                  [TECHNICAL REFERENCE]
│   ├── sdk.md                 - SDK API reference
│   ├── cli.md                 - Command-line tools
│   ├── hooks.md               - Hook system API
│   ├── server.md              - Server configuration
│   ├── models.md              - Data models
│   ├── planning.md            - Planning API
│   ├── track-builder.md       - TrackBuilder API
│   ├── agents.md              - Agent API
│   ├── graph.md               - Graph operations
│   └── ids.md                 - ID generation
│
├── examples/             [RUNNABLE CODE & PATTERNS]
│   ├── basic.md               - Simple feature creation
│   ├── agents.md              - Agent patterns
│   ├── tracks.md              - Track examples
│   ├── agent-coordination.md  - Two-phase explorer+coder
│   ├── multi-agent-workflow.md - Parallel subagent delegation
│   └── delegation.md          - Delegation examples
│
├── architecture/         [DEEP TECHNICAL DIVES]
│   ├── design.md              - System architecture
│   ├── parent-child-events.md - Event hierarchy
│   ├── event-architecture.md  - Event system design
│   ├── sqlite-schema.md       - Database schema
│   ├── hash-based-ids.md      - ID generation
│   ├── orchestration.md       - Orchestrator design
│   └── git-continuity.md      - Git integration
│
├── contributing/         [DEVELOPER GUIDES]
│   ├── development.md    - Setup development environment
│   ├── publishing.md     - Release and publishing
│   └── index.md          - Contributing overview
│
└── archive/              [DEPRECATED DOCUMENTATION]
    └── [Old versions and superseded docs]
```

## Popular Topics

### For AI Agents & Orchestrators
- [Delegation Guide](guide/delegation.md) - Write effective delegation prompts
- [Multi-Agent Workflow](examples/multi-agent-workflow.md) - Real parallel execution example
- [Orchestration Guide](guide/orchestration.md) - Coordinate agent work
- [Model Selection Guide](../docs/MODEL_SELECTION_GUIDE.md) - Choose the right model

### For SDK Users
- [SDK Reference](api/sdk.md) - Complete API
- [Basic Examples](examples/basic.md) - Getting started with code
- [Track Builder](guide/track-builder.md) - Plan features
- [Dashboard](guide/dashboard.md) - Visualize progress

### For System Design
- [Architecture](architecture/design.md) - System overview
- [Event Architecture](architecture/event-architecture.md) - How events flow
- [Database Schema](architecture/sqlite-schema.md) - Data storage
- [Hash-Based IDs](architecture/hash-based-ids.md) - ID design

## Core Features

**Features & Tracking**
- Create and manage features with properties
- Track progress and completion
- Link related features with edges
- Assign to agents and teams

**Session Management**
- Automatic session tracking
- Parent-child event hierarchies
- Event correlation and analysis
- Drift detection and alerting

**Orchestration**
- Delegate work to subagents
- Parallel task execution
- Context-preserving coordination
- Cost optimization (Opus orchestrator + Haiku subagents)

**Visualization**
- Interactive dashboard
- Session hierarchies
- Feature tracking
- Activity feeds
- Strategic analytics

## Quick Reference

### Installation
```bash
uv pip install wipnote
```

### Basic Usage
```python
from wipnote import SDK

sdk = SDK(agent="me")
feature = sdk.features.create("My Feature")
```

### Serve Dashboard
```bash
uv run wipnote serve
# Visit http://localhost:8000
```

### Check Status
```bash
uv run wipnote status
```

## Getting Help

- **[FAQ](../docs/FAQ.md)** (if exists) - Common questions
- **[System Prompt Troubleshooting](SYSTEM_PROMPT_ARCHITECTURE.md#troubleshooting-common-issues)** - Debug issues
- **[GitHub Issues](https://github.com/anthropics/wipnote/issues)** - Report bugs

## Advanced Topics

- [Hooks Architecture](api/hooks.md) - Event-driven automation
- [Event Tracing](../docs/EVENT_TRACING.md) - Deep dive into event system
- [SQLite Index](guide/sqlite-index.md) - Query optimization
- [System Prompt Persistence](../docs/SYSTEM_PROMPT_ARCHITECTURE.md) - Configuration
- [Git Continuity](architecture/git-continuity.md) - Git integration patterns
- [Concurrency Patterns](../docs/concurrency-patterns.md) - Async patterns
- [Quality Gates](../docs/QUALITY_GATES.md) - Testing and validation

## Project Status

Wipnote is actively developed. Current version: **0.9.6**

Latest changes:
- ✅ Parent-child event linking for orchestration
- ✅ Multi-agent coordination patterns
- ✅ Performance optimizations
- ✅ Enhanced dashboard

See [Changelog](changelog.md) for full history.

## License

Wipnote is open source. See LICENSE file for details.

---

**Ready to start?** → [Installation](getting-started/installation.md) | [Quick Start](getting-started/quick-start.md) | [Examples](examples/index.md)
