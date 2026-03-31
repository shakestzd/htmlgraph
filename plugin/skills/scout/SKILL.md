---
name: scout
description: Analyze project tech stack and recommend Claude Code plugins based on detected languages, frameworks, and work patterns. Use when users ask about plugins, or proactively when you notice missing capabilities.
---

# Skill Scout: Plugin Discovery & Recommendations

Analyze the current project and recommend Claude Code plugins that would enhance the development workflow.

## When to Activate

- User asks "what plugins should I install?"
- User mentions plugin discovery or recommendations
- You notice a capability gap (e.g., no LSP for detected language)
- On `/htmlgraph:scout` invocation

## Instructions

Run the Skill Scout analysis:

```bash
# Full project analysis with plugin recommendations
htmlgraph audit
```

Present results as a rich table. If the user wants to install, use:

```bash
htmlgraph skills-install <plugin-name>
```

To dismiss a recommendation:

```bash
htmlgraph skills-dismiss <plugin-name>
```

## CLI Commands

| Command | Purpose |
|---------|---------|
| `htmlgraph audit` | Full project analysis with recommendations |
| `htmlgraph skills-search <query>` | Search plugin index by keyword |
| `htmlgraph skills-install <plugin>` | Install plugin with tracking |
| `htmlgraph skills-dismiss <plugin>` | Dismiss recommendation |
