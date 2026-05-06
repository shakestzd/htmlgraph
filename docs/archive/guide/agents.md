# Agents

Wipnote provides seamless integration with AI agents through SDKs, CLI tools, and browser extensions.

## Supported Agents

### Claude Code

Official plugin for Claude Code CLI.

**Installation:**

```bash
claude plugin install wipnote
```

**Features:**

- Automatic session management via hooks
- Activity tracking for all tool calls
- Drift detection and warnings
- Feature creation decision framework
- Session continuity across conversations

**Documentation:** See the `wipnote` skill in Claude Code.

### Gemini CLI

Extension for Gemini CLI.

**Installation:**

```bash
gemini extension install wipnote
```

**Features:**

- Session tracking
- Feature management
- TrackBuilder integration
- Activity logging

**Documentation:** Included in the extension's `GEMINI.md` file.

### Codex CLI

Skill for Codex CLI.

**Installation:**

```bash
codex skill install wipnote
```

**Features:**

- Feature tracking
- Session management
- Spec and plan creation
- CLI integration

**Documentation:** Included in the skill's `SKILL.md` file.

## Agent Workflow

### 1. Session Start

When an agent begins working:

```bash
# Check current status
wipnote status
```

### 2. Create or Select Feature

```bash
# Create a new feature
wipnote feature create "Add user profile page" --priority high

# Start working on it
wipnote feature start feat-a1b2c3d4
```

### 3. Work and Track Progress

```bash
# Document decisions as spikes
wipnote spike create "Chose React Router over Reach Router (better TypeScript support)"
```

### 4. Complete Feature

```bash
# Mark feature as done
wipnote feature complete feat-a1b2c3d4
```

## Multi-Agent Collaboration

Multiple agents can work together on the same graph:

### Agent Assignment

Agents claim features by starting them:

```bash
# Agent claims a feature
wipnote feature start feature-001
```

### Handoff Notes

When passing work between agents, document the handoff as a spike:

```bash
# Agent 1 documents phase 1 completion
wipnote spike create "OAuth setup complete. OAuth provider configured, redirect endpoints created. Blocked on Database schema (feature-005). Next: JWT signing once DB ready, add token refresh logic."

# Agent 2 checks status before picking up
wipnote feature show feature-001
wipnote snapshot --summary
```

## CLI Integration Patterns

### Simple Task

```bash
wipnote feature create "Quick task" --priority medium
wipnote feature start feat-a1b2c3d4
# Do work...
wipnote feature complete feat-a1b2c3d4
```

### Complex Multi-Step Work

```bash
# Create a track for multi-phase work
wipnote track new "Complex Agent Task" --priority high
# Note track ID (e.g. trk-a1b2c3d4)

# Create features per phase
wipnote feature create "Phase 1: Setup" --priority high --track trk-a1b2c3d4
wipnote feature create "Phase 2: Implementation" --priority high --track trk-a1b2c3d4

# Work through each feature
wipnote feature start feat-phase1-id
# ... do work ...
wipnote feature complete feat-phase1-id

wipnote feature start feat-phase2-id
# ... do work ...
wipnote feature complete feat-phase2-id
```

## Hooks

Wipnote uses hooks to automatically track agent activity.

### Available Hooks

- **SessionStart**: Creates session, provides context
- **PostToolUse**: Logs every tool call
- **UserPromptSubmit**: Logs user queries
- **SessionEnd**: Finalizes session, generates summary

### Hook Configuration

**Claude Code:**

Hooks are configured automatically via the Claude Code plugin. The plugin bundles a `hooks.json` that registers all event handlers. Hook scripts live in `packages/claude-plugin/hooks/scripts/` in the plugin source. Install the plugin to activate all hooks:

```bash
claude plugin install wipnote
```

### Custom Hooks

Hooks are configured in `packages/claude-plugin/hooks/scripts/`. They run automatically via the plugin and use the `wipnote` CLI for tracking operations.

## Best Practices

### 1. Feature Creation Decision Framework

Use this framework to decide when to create a feature:

**Create a feature if:**

- Estimated >30 minutes of work
- Involves 3+ files
- Requires new tests
- Affects multiple components
- Hard to revert (schema changes, API changes)
- Needs documentation

**Implement directly if:**

- Single file, obvious change
- <30 minutes work
- No cross-system impact
- Easy to revert
- No tests needed

### 2. Use Tracks for Complex Work

For multi-feature projects, create a track first:

```bash
wipnote track new "Multi-phase project" --priority high
# Create features linked to the track
wipnote feature create "Phase 1" --priority high --track trk-a1b2c3d4
wipnote feature create "Phase 2" --priority high --track trk-a1b2c3d4
```

### 3. Document Decisions

Always record important decisions:

```bash
wipnote spike create "Chose PostgreSQL over MongoDB: better transactions, team familiarity"
```

### 4. One Feature at a Time

Focus on single features for clear attribution:

```bash
wipnote feature start feature-001
# Complete all work
wipnote feature complete feature-001
```

## Next Steps

- [Sessions Guide](sessions.md) - Understanding session tracking
- [Features & Tracks Guide](features-tracks.md) - Creating and managing work
- [API Reference](../api/agents.md) - Complete agent API documentation
- [Examples](../examples/agents.md) - Real-world agent examples
