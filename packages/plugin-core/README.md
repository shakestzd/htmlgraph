# plugin-core — DRY source of truth for HtmlGraph plugin ports

All HtmlGraph plugin ports (Claude Code, Codex CLI) are generated from the files
in this directory so we never edit the same logic twice.

## Source of truth

- **`manifest.json`** — plugin metadata, per-target output paths, hook event
  matrix. Both `plugin/.claude-plugin/plugin.json` and
  `packages/codex-plugin/.codex-plugin/plugin.json` are generated from it.
- **Assets** (commands, agents, skills, templates, static, config) continue to
  live in `plugin/` and are copied verbatim into each target. The markdown
  formats (SKILL.md, agent `.md`, slash-command `.md`) are compatible with
  Claude Code and Codex CLI, so no per-target translation is needed.

## Build

    htmlgraph plugin build-ports              # regenerate all targets
    htmlgraph plugin build-ports --target codex
    htmlgraph plugin build-ports --target claude

The command writes to the `outDir` declared under each target in
`manifest.json`.

## Hooks — thin wrappers

Every hook resolves to `htmlgraph hook <handler>`. Business logic lives in the
Go CLI (`internal/hooks/`); the plugin manifests only declare which events route
to which handler and on which target. Events marked `targets: ["claude"]` are
omitted from Codex output, and vice versa.

## Adding a new plugin surface

1. **New command / agent / skill:** drop the markdown file into `plugin/…` and
   rerun `htmlgraph plugin build-ports`. Both targets pick it up automatically.
2. **New hook event:** add an entry to `manifest.json` → `hooks.events`
   declaring the event name, handler (CLI subcommand), and the targets that
   support it. If the business logic is new, add a handler in
   `internal/hooks/` and route it from `cmd/htmlgraph/hook.go`.
3. **New target (e.g. Gemini):** add a `targets.<name>` entry with `outDir`,
   `manifestPath`, and `hooksPath`, then implement an `Adapter` in
   `internal/pluginbuild/`.
