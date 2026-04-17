# plugin-core — DRY source of truth for HtmlGraph plugin ports

All HtmlGraph plugin ports (Claude Code, Codex CLI, …) are generated from the
files in this directory so we never edit the same logic twice.

## Source of truth

- **`manifest.json`** — plugin metadata, per-target output paths, hook event
  matrix. Both `plugin/.claude-plugin/plugin.json` and
  `packages/codex-plugin/.codex-plugin/plugin.json` are generated from it.
- **Assets** (commands, agents, skills, templates, static, config) live in
  `plugin/…/` and are copied verbatim into each target. The markdown formats
  (SKILL.md, agent `.md`, slash-command `.md`) are compatible with Claude Code
  and Codex CLI, so no per-target translation is needed.
- **Generated trees** — `plugin/` (Claude) and `packages/codex-plugin/` (Codex)
  are output directories. Treat them as build artifacts: do not hand-edit
  anything under `plugin/.claude-plugin/`, `plugin/hooks/hooks.json`, or
  `packages/codex-plugin/`. Regenerate instead.

## Build

    htmlgraph plugin build-ports              # regenerate all targets
    htmlgraph plugin build-ports --target codex
    htmlgraph plugin build-ports --target claude

The command writes each target's tree under the `outDir` declared in
`manifest.json → targets.<name>`.

## Hooks — thin wrappers

Every hook resolves to `htmlgraph hook <handler>`. Business logic lives in the
Go CLI (`internal/hooks/`); the plugin manifests only declare which events route
to which handler and on which target. Events whose `targets` list omits a given
target are not emitted to that target's hooks file.

### Hook event matrix

Derived from `manifest.json → hooks.events`. Update this table whenever you
edit the manifest.

| Event | Handler | Claude | Codex | Gemini | Notes |
|-------|---------|:---:|:---:|:---:|-------|
| `SessionStart` | `session-start` | x | x | x | |
| `SessionStart` | `session-resume` | x | | | matcher: `resume` |
| `SessionEnd` | `session-end` | x | | | |
| `UserPromptSubmit` | `user-prompt` | x | x | x | |
| `UserPromptSubmit` | `timestamp` | x | | | shell `command:` only — injects local timestamp |
| `PreToolUse` | `pretooluse` | x | x | x | |
| `PostToolUse` | `posttooluse` | x | x | x | |
| `PostToolUse` | `exit-plan-mode` | x | | | matcher: `ExitPlanMode` |
| `PostToolUseFailure` | `posttooluse-failure` | x | | | |
| `SubagentStart` | `subagent-start` | x | | | |
| `SubagentStop` | `subagent-stop` | x | | | |
| `Stop` | `stop` | x | | x | |
| `PreCompact` | `pre-compact` | x | | | |
| `PostCompact` | `post-compact` | x | | | |
| `TeammateIdle` | `teammate-idle` | x | | | |
| `TaskCompleted` | `task-completed` | x | | | |
| `TaskCreated` | `task-created` | x | | | |
| `InstructionsLoaded` | `instructions-loaded` | x | | | |
| `WorktreeCreate` | `worktree-create` | x | | | |
| `WorktreeRemove` | `worktree-remove` | x | | | |
| `PermissionRequest` | `permission-request` | x | | | |
| `ConfigChange` | `config-change` | x | | | |
| `TaskStarted` | `task-started` | | x | | Codex-specific |
| `TaskComplete` | `stop` | | x | | Codex-specific — reuses `stop` handler |
| `TurnAborted` | `task-aborted` | | x | | Codex-specific |

## Recipes

### Add a new slash command / agent / skill

Drop the markdown file into the matching `plugin/` subtree and regenerate:

```bash
# examples
$EDITOR plugin/commands/mycmd.md
$EDITOR plugin/agents/my-agent.md
$EDITOR plugin/skills/my-skill/SKILL.md

htmlgraph plugin build-ports
```

Every target picks the new asset up automatically — no manifest edit needed,
because `manifest.json → assetSources` already points at `plugin/{commands,agents,skills,…}`.

### Add a new hook event

Three places, always in this order:

1. **Manifest** (`packages/plugin-core/manifest.json`) — add one entry to
   `hooks.events`, listing the event name, handler, and targets:

   ```json
   { "name": "MyNewEvent", "handler": "my-new-event", "targets": ["claude", "codex"] }
   ```

   Optional keys: `matcher` (e.g. `"ExitPlanMode"`), `timeout` (seconds),
   `command` (escape hatch for shell-only hooks; bypasses `handler`).

2. **Go handler** (`internal/hooks/my_new_event.go`) — implement the handler with
   the signature matching the wiring you'll use:

   ```go
   package hooks

   func MyNewEvent(event *CloudEvent, db *sql.DB) (*HookResult, error) {
       // business logic
       return &HookResult{}, nil
   }
   ```

3. **Route** (`cmd/htmlgraph/hook.go`) — register the CLI subcommand so
   `htmlgraph hook my-new-event` resolves to the Go handler:

   ```go
   hookSubcmd("my-new-event", "Handle MyNewEvent event", emptyResult, hooks.MyNewEvent),
   ```

   Use `hookSubcmdWithProject(...)` instead when the handler needs the project
   dir passed through (see `session-start` for the pattern).

Then run `htmlgraph plugin build-ports && htmlgraph build` and update the
**Hook event matrix** table above.

### Add a new target (e.g. Gemini)

1. **Manifest** — add a `targets.<name>` entry:

   ```json
   "gemini": {
     "outDir": "packages/gemini-plugin",
     "manifestPath": ".gemini-plugin/plugin.json",
     "hooksPath": "hooks.json"
   }
   ```

   `mcpPath` is optional (see Codex for an example). Then tag each applicable
   hook event in `hooks.events` with `"gemini"` in its `targets` list.

2. **Adapter** — implement the `Adapter` interface in a new file under
   `internal/pluginbuild/` (model it on `claude.go` / `codex.go`):

   ```go
   package pluginbuild

   type geminiAdapter struct{}

   func init() { Register(geminiAdapter{}) }

   func (geminiAdapter) Name() string { return "gemini" }

   func (geminiAdapter) Emit(m *Manifest, repoRoot, outDir string) error {
       // 1. write the target-specific plugin.json from m (use writeJSON)
       // 2. write the target-specific hooks.json from m.Hooks.Events
       //    (filter with HookEvent.AppliesTo("gemini"))
       // 3. copy assets with copyAssetTree(...) using m.AssetSources
       return nil
   }
   ```

   The `Adapter` interface is defined in `internal/pluginbuild/adapter.go`:

   - `Name() string` — must match the manifest `targets.<name>` key.
   - `Emit(m *Manifest, repoRoot, outDir string) error` — write the full tree
     rooted at `outDir`.

   `init()` must call `Register(...)` so the target is discoverable by
   `htmlgraph plugin build-ports --target <name>`. Duplicate registrations panic.

3. **Regenerate and verify**:

   ```bash
   htmlgraph build
   htmlgraph plugin build-ports --target gemini
   ```
