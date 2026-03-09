# Plugin Synchronization

Documentation for synchronizing HtmlGraph plugin components between source and distribution.

## Overview

The plugin synchronization process ensures that all plugin components (hooks, skills, agents, commands) in the source directory are properly packaged and synced with the `.claude/` directory for distribution to Claude Code users.

## Plugin Architecture

HtmlGraph maintains two synchronized copies of plugin source:

1. **Plugin Source:** `packages/claude-plugin/.claude-plugin/`
   - Official source of truth for all plugin components
   - Where developers make changes
   - Committed to git repository

2. **Distribution Copy:** `.claude/`
   - Auto-synced from plugin source
   - What gets distributed to users
   - Should NOT be manually edited

## Sync Process

### Automatic Sync During Development

When running HtmlGraph in dev mode, plugin components are loaded directly from the source:

```bash
uv run htmlgraph claude --dev
```

This loads:
- Hooks from `packages/claude-plugin/.claude-plugin/hooks/`
- Skills from `packages/claude-plugin/.claude-plugin/skills/`
- Commands from `packages/claude-plugin/.claude-plugin/commands/`
- Agents from `packages/claude-plugin/.claude-plugin/agents/`

### Manual Sync for Distribution

Before deploying, ensure all components are synced:

```bash
# Check sync status
uv run htmlgraph sync-plugins --check

# Perform sync
uv run htmlgraph sync-plugins

# Verify results
ls -la .claude/hooks/
ls -la .claude/skills/
ls -la .claude/agents/
```

### Deployment Sync

The `deploy-all.sh` script automatically syncs plugins in pre-flight:

```bash
./scripts/deploy-all.sh 0.9.5 --no-confirm
# Pre-flight: Plugin Sync - Verify packages/claude-plugin and .claude are synced
```

## Component Structure

### Hooks

Location: `packages/claude-plugin/.claude-plugin/hooks/`

```
hooks/
├── hooks.json                 ← Event routing configuration
└── scripts/
    ├── session-start.py       ← Database initialization
    ├── user-prompt-submit.py  ← User query tracking
    ├── track-event.py         ← Tool event recording
    └── session-end.py         ← Session cleanup
```

### Skills

Location: `packages/claude-plugin/.claude-plugin/skills/`

```
skills/
├── orchestrator-directives/
│   └── SKILL.md
├── code-quality/
│   └── SKILL.md
└── deployment-automation/
    └── SKILL.md
```

### Commands

Location: `packages/claude-plugin/.claude-plugin/commands/`

```
commands/
├── deploy.md
├── init.md
├── plan.md
└── status.md
```

### Agents

Location: `packages/claude-plugin/.claude-plugin/agents/`

```
agents/
├── researcher.md
├── debugger.md
└── test-runner.md
```

## Troubleshooting

### Issue: Changes to hooks not visible

**Problem:** Modified hook in source but not appearing in Claude Code

**Solution:**
```bash
# 1. Verify source file exists
ls -la packages/claude-plugin/.claude-plugin/hooks/scripts/your-hook.py

# 2. Sync distribution
uv run htmlgraph sync-plugins

# 3. Restart Claude Code
claude --restart

# 4. Verify hook loaded
/hooks your-hook-type
```

### Issue: Sync reports out-of-sync files

**Problem:** `uv run htmlgraph sync-plugins --check` shows mismatches

**Solution:**
```bash
# 1. Run full sync
uv run htmlgraph sync-plugins

# 2. Commit synced changes
git add .claude/
git commit -m "chore: sync plugin components"

# 3. Verify no remaining diffs
git diff .claude/
```

## Best Practices

1. **Always edit plugin source, never .claude/**
   - Changes to `.claude/` are overwritten on next sync
   - Plugin source is the single source of truth

2. **Sync before deployment**
   - Run `./scripts/deploy-all.sh` which auto-syncs
   - Or manually sync: `uv run htmlgraph sync-plugins`

3. **Commit synced files**
   - Include `.claude/` changes in deployment commits
   - Ensures users get latest components

4. **Test in dev mode**
   - Use `uv run htmlgraph claude --dev` for testing
   - Loads from plugin source directly
   - No need to manually sync during development

## See Also

- [Plugin Development](./claude_code_plugins.md)
- [Deployment Guide](./.claude/rules/deployment.md)
- [Hook Development Guide](./.claude/rules/debugging.md)
