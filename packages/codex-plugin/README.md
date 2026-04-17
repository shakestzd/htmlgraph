# codex-plugin — generated Codex CLI plugin tree

**This directory is generated. Do not edit files here directly — changes are
overwritten on the next port build.**

## Source of truth

- **Manifest, hook matrix, metadata:** [`packages/plugin-core/manifest.json`](../plugin-core/manifest.json)
- **Assets (commands, agents, skills, templates, static, config):** [`plugin/`](../../plugin/)

The generator that emits this tree is `internal/pluginbuild/codex.go`. The CLI
entry point is `htmlgraph plugin build-ports --target codex`.

## Regenerate

From the repo root:

    htmlgraph plugin build-ports --target codex

This rewrites:

- `.codex-plugin/plugin.json` — Codex manifest (name, version, author, `interface` block)
- `hooks.json` — Codex-applicable hook events (wrappers around `htmlgraph hook <handler>`)
- `.mcp.json` — stub MCP server map (HtmlGraph does not currently ship an MCP server)
- `commands/`, `agents/`, `skills/`, `templates/`, `static/`, `config/` — copied verbatim from `plugin/`

## Install locally for testing

The Codex CLI loads plugins from `~/.codex/plugins/<name>/`. Symlink (or copy)
this generated tree there and Codex will pick it up on next launch:

    mkdir -p ~/.codex/plugins
    ln -sf "$(pwd)/packages/codex-plugin" ~/.codex/plugins/htmlgraph

    # or, for a copy (stable across regenerations):
    rm -rf ~/.codex/plugins/htmlgraph
    cp -R packages/codex-plugin ~/.codex/plugins/htmlgraph

Verify the `htmlgraph` CLI is on your `PATH` (hooks shell out to it); otherwise
install it with `htmlgraph build` or your system package manager.

Launch Codex and confirm the plugin registers:

    codex plugins list    # htmlgraph should appear

## Hook events

Codex-supported events (per `packages/plugin-core/manifest.json`):

| Event              | Handler                      |
|--------------------|------------------------------|
| `SessionStart`     | `htmlgraph hook session-start` |
| `UserPromptSubmit` | `htmlgraph hook user-prompt`   |
| `PreToolUse`       | `htmlgraph hook pretooluse`    |
| `PostToolUse`      | `htmlgraph hook posttooluse`   |
| `TaskStarted`      | `htmlgraph hook task-started`  |
| `TaskComplete`     | `htmlgraph hook stop`          |
| `TurnAborted`      | `htmlgraph hook task-aborted`  |

Claude-only events (`SessionEnd`, `Stop`, `SubagentStart`/`Stop`, compaction,
teammate, worktree, permission, config) are omitted from this tree — they are
not part of the Codex event surface.

## Adding a surface

1. New command / agent / skill — drop the markdown under `plugin/` and rerun
   `htmlgraph plugin build-ports`.
2. New hook — add the entry to `packages/plugin-core/manifest.json` with the
   appropriate `targets` list, then rerun the generator.

See [`packages/plugin-core/README.md`](../plugin-core/README.md) for the full
porting contract.
