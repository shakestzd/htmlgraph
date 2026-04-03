# HtmlGraph

Local-first observability and coordination platform for AI-assisted development.

## Architecture

| Layer | Role |
|-------|------|
| `.htmlgraph/*.html` | Canonical store — single source of truth |
| SQLite (`.htmlgraph/htmlgraph.db`) | Read index for queries and dashboard |
| Go binary (`htmlgraph`) | CLI + hook handler |

## For AI Agents

All CLI usage, safety rules, and best practices are delivered by the HtmlGraph plugin.
Run `htmlgraph help --compact` for the CLI reference.

## Dogfooding

This project uses HtmlGraph to develop itself. `.htmlgraph/` contains real work items — not demos.
