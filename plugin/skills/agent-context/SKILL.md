---
name: agent-context
description: Shared agent context — work attribution, safety rules, and development principles. Loaded by all plugin agents via skills: frontmatter.
---

# Shared Agent Context

## Work Attribution

The orchestrator always provides the work item ID in your task prompt (e.g., "Feature: feat-580dc00b"). Use it:

```bash
htmlgraph feature start <id>   # or bug start / spike start
```

**Rules:**
1. Look for a feature/bug/spike ID in the task prompt first
2. If found, run `start` on it — do NOT create a new one
3. Only create a new work item if the prompt genuinely contains no ID
4. If htmlgraph is unavailable, proceed — attribution is not a blocker

## Safety Rules

**FORBIDDEN:** Never edit `.htmlgraph/` files directly. Use the CLI:
- `htmlgraph feature complete <id>` not `Edit(".htmlgraph/features/...")`
- `htmlgraph bug create "title"` not `Write(".htmlgraph/bugs/...")`

## Development Principles

- DRY — check for existing utilities before creating new ones
- SRP — one purpose per function/module
- KISS — simplest solution that satisfies requirements
- YAGNI — only implement what is needed now
- Module limits: functions <50 lines, files <500 lines
- Research existing libraries/packages before implementing from scratch
- Check project dependencies before adding new ones

These principles are language-neutral and apply to any codebase.
