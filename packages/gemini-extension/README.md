# gemini-extension — Generated Gemini CLI extension tree

**This tree is generated from `packages/plugin-core/`. Do not hand-edit.**

Regenerate with:

    htmlgraph plugin build-ports --target gemini

Any change here is a change to the build output and will be overwritten on the
next run. To add commands, agents, skills, or hooks, edit the shared source
under `plugin/` and `packages/plugin-core/manifest.json` — see
[`packages/plugin-core/README.md`](../plugin-core/README.md) for the per-task
recipes (new command, new agent, new skill, new hook).

## Install for local testing

Link this tree into your Gemini CLI and restart so the new extension is picked up:

    gemini extensions link $(pwd)/packages/gemini-extension

Then restart the Gemini CLI. Unlink with `gemini extensions unlink htmlgraph`
when you're done.

## Tree layout

    packages/gemini-extension/
    ├── gemini-extension.json     # extension manifest (name, version, contextFileName)
    ├── GEMINI.md                 # context file copied from the repo root
    ├── commands/<namespace>/     # TOML slash commands (translated from plugin/commands/*.md)
    ├── agents/                   # markdown agent definitions (copied verbatim)
    ├── skills/<name>/SKILL.md    # skill directories (copied verbatim)
    └── hooks/hooks.json          # hook event wiring for Gemini-targeted events

See `internal/pluginbuild/gemini.go` for the emitter and the sub-emitter files
(`gemini_*.go`) that populate each part of the tree.
