# Warp Integration Plan for HtmlGraph

This document captures the integration strategy for the newly open-sourced Warp
terminal (`github.com/warpdotdev/warp`, AGPL-3.0 + MIT for `warpui_*`, ~98% Rust).
Unlike the Claude Code, Codex CLI, and Gemini CLI ports — which all share a
hook-event lifecycle — Warp does **not** ship a Claude-Code-style hook lattice.
That single fact reshapes the entire plan: HtmlGraph cannot simply add a fourth
adapter to `htmlgraph plugin build-ports` and call it done. The HtmlGraph
philosophy still applies — it just has to express itself through the surfaces
Warp actually offers.

---

## Philosophy Mapping

HtmlGraph's claims are: **local-first**, **HTML as canonical store**,
**observe-don't-steer**, **work-item attribution**, **harness-agnostic**. Each
needs a surface in Warp.

| HtmlGraph principle           | Today (Claude/Codex/Gemini)         | Warp equivalent                                                  |
|-------------------------------|-------------------------------------|------------------------------------------------------------------|
| Local-first storage           | `.htmlgraph/*.html` + SQLite index  | unchanged — Warp does not require any cloud round-trip           |
| Capture every prompt          | `UserPromptSubmit` hook             | poll Warp's local SQLite (`crates/persistence`) for new messages |
| Capture every tool call       | `PreToolUse` / `PostToolUse` hooks  | subscribe to `agent_events` driver, or tail conversation rows    |
| Block unsafe operations       | hook returns `{"decision":"block"}` | not directly possible — Warp has no exit-code interception       |
| Inject system prompt / skills | `--append-system-prompt`, plugin    | `.agents/skills/htmlgraph/SKILL.md` (identical format)           |
| Provide CLI tools to agent    | slash commands + agent skills       | **MCP server** (cleanest surface) + skills                       |
| Wrap the harness binary       | `htmlgraph claude --dev`            | `htmlgraph warp` wrapping `warp agent run`                       |
| Same source-of-truth          | `packages/plugin-core/manifest.json`| add `warp` target — but with `hooks: []`                         |

The two rows that *cannot* map cleanly are the gating ones (`PreToolUse` block
decisions). Warp does not expose a synchronous interception point. HtmlGraph's
`yolo_guard` / `task_completion_gate` style of *prevention* therefore degrades
to *post-hoc detection* on the Warp side. This is acceptable: the same gates
already run in CI and pre-commit hooks, which Warp respects via shell-out.

---

## Warp's Actual Integration Seams

From `warpdotdev/warp` HEAD:

1. **MCP** — fully integrated via `rmcp` (the Rust MCP SDK). Manager at
   `app/src/ai/mcp/{file_based_manager.rs, file_mcp_watcher.rs}` watches
   `.mcp.json` for hot reload. Templatable per-server installs. **This is the
   primary integration surface.**
2. **Skills** — `app/src/ai/skills/skill_manager.rs` loads markdown bundles from
   `.agents/skills/<name>/SKILL.md`. Format is identical to Claude Code skills.
   `git`-based skill resolution exists (`resolve_skill_spec`).
3. **Agent SDK CLI** — `warp agent run`, `warp run message {send,watch,list}`,
   `warp mcp list`, `warp task get`. The `--harness` flag accepts strings
   (`oz`, `claude`, third-party). Telemetry events are typed
   (`CliTelemetryEvent::AgentRun{...}` in `agent_sdk/telemetry.rs`).
4. **Local persistence** — Diesel + SQLite at `app/src/persistence/schema.rs`.
   Conversations (`AIConversationId`), tasks (`AmbientAgentTaskId`), messages
   are durable, hand-readable, and the natural read-side for an external
   observer.
5. **Plugin host** (heavy) — `app/src/plugin/{app/, host/native/, host/wasm/,
   service/}` spawns plugins as subprocesses, talks JSON-RPC over IPC, env var
   `WARP_PLUGIN_HOST_ADDRESS`. Out of scope for v1.
6. **ACP** — *planned, not shipped* (warpdotdev/warp#9233). Once landed, this
   becomes the canonical event-stream contract; the SQLite-tail approach in
   Phase 3 below is designed to be replaced by it without re-architecting.

What Warp does **not** ship: `UserPromptSubmit`/`PreToolUse`/`PostToolUse`
exit-code hooks, `hooks.json`, stdin-CloudEvent contracts. There is no
synchronous interception point on the agent loop.

---

## Phased Integration

### Phase 1 — Ship HtmlGraph as an MCP server (foundation)

Lowest friction, highest leverage. Once HtmlGraph speaks MCP, **every**
MCP-aware harness picks it up — Warp, Cursor, Codex (already), and any future
ACP-speaking client. The MCP path is the right primary integration even
ignoring Warp.

New code:

- `cmd/htmlgraph/mcp.go` — new subcommand `htmlgraph mcp serve` (stdio MCP
  transport).
- `internal/mcp/server.go` — server registration, lifecycle.
- `internal/mcp/tools/` — one file per tool. Initial set:
  - `current_session()` → returns active session metadata (id, work item,
    started_at, branch, cwd).
  - `list_work_items(status?, kind?, limit?)` → kanban data.
  - `get_work_item(id)` → full HTML record + provenance.
  - `record_attribution(work_item_id, evidence)` → append agent_event marking
    a tool result as causally tied to a work item.
  - `query_provenance(file_path | commit_sha)` → reverse-lookup: which work
    item / session / agent produced this artifact.
  - `snapshot_summary()` → equivalent of `htmlgraph snapshot --summary`.
- Wire registration into `cmd/htmlgraph/root.go`.

Tests live alongside (`internal/mcp/server_test.go`). The tools are thin
wrappers over existing Go APIs in `internal/storage` and `internal/sessions` —
no new business logic.

Distribution: a stanza users paste into their Warp `.mcp.json`:

```json
{
  "mcpServers": {
    "htmlgraph": {
      "command": "htmlgraph",
      "args": ["mcp", "serve"]
    }
  }
}
```

Acceptance: Warp's `app/src/ai/mcp/file_mcp_watcher.rs` picks the entry up
without restart; `warp mcp list` shows `htmlgraph` and its tools.

### Phase 2 — Skill drop-in via `build-ports` (alignment)

Warp's `.agents/skills/<name>/SKILL.md` accepts the same frontmatter and body
as Claude Code skills. The existing source tree at `plugin/skills/` already
produces the right artifact; we just need a `warp` adapter that copies it.

- Edit `packages/plugin-core/manifest.json`:
  ```jsonc
  "warp": {
    "outDir": "packages/warp-extension",
    "manifestPath": "warp-extension.json",   // hand-rolled manifest, see below
    "mcpPath": ".mcp.json",                  // pre-baked MCP entry users can copy
    "skillsDir": ".agents/skills",           // Warp's expected location
    "agentsDir": ".agents/agents",
    "commandNamespace": "htmlgraph"
  }
  ```
  The `hooks.events[].targets` array intentionally never includes `"warp"` —
  Warp is a *no-hooks target*.
- New `internal/pluginbuild/warp.go` implementing `Adapter`:
  ```go
  func (warpAdapter) Name() string { return "warp" }
  func (w warpAdapter) Emit(m *Manifest, repoRoot, outDir string) error {
      // 1. clean owned subtrees: skillsDir, agentsDir, ".mcp.json"
      // 2. write minimal warp-extension.json (name, version, description, mcpRef)
      // 3. copy plugin/skills → outDir/.agents/skills/htmlgraph/
      // 4. copy plugin/agents → outDir/.agents/agents/
      // 5. write outDir/.mcp.json with the htmlgraph MCP stanza
      // 6. write outDir/README.md (install steps)
  }
  func init() { Register(warpAdapter{}) }
  ```
- New tests in `internal/pluginbuild/warp_test.go` mirroring
  `gemini_assets_test.go` shape: snapshot the emitted tree, assert no hook
  artifacts leak in.

After this phase: `htmlgraph plugin build-ports --target warp` produces a
hand-installable extension tree. Users `cp -r packages/warp-extension/.agents
/path/to/their/repo/`, paste the `.mcp.json` stanza, restart Warp.

### Phase 3 — Local persistence observer (the philosophical adaptation)

This is where HtmlGraph evolves. The whole `internal/hooks/` tree was named
that because every existing harness pushed events via stdin hooks. Warp
doesn't push; it stores. The fix is to give the existing event router a
second source.

- New `internal/events/` package — pulls the harness-agnostic core out of
  `internal/hooks/` (session_start, posttooluse, etc. are *event handlers*,
  not *hook handlers*). This is a rename + interface extraction; behavior
  unchanged for Claude/Codex/Gemini.
- New `internal/events/warp_observer.go`:
  - Locates Warp's SQLite path (`$XDG_DATA_HOME/dev.warp.Warp/...` on Linux,
    `~/Library/Application Support/dev.warp.Warp/...` on macOS — discover via
    `warp environment paths` once that command exists, hard-code fallbacks
    until then).
  - Opens a read-only Diesel-compatible connection. Schema parsed from
    `app/src/persistence/schema.rs` — pin a version once, regenerate when
    Warp bumps it.
  - Polls (1–2 s interval) for new rows in `conversations`, `messages`,
    `tasks`. For each new row, synthesizes a CloudEvent and feeds it into the
    same router that hook stdin feeds. Mapping:
    - new `conversations` row → `SessionStart`
    - new `messages` row, `role=user` → `UserPromptSubmit`
    - new `messages` row, `role=assistant` with tool_calls → `PreToolUse`
      followed by `PostToolUse` once the tool result row lands
    - `conversations.ended_at` non-null → `SessionEnd`
  - Runs as `htmlgraph warp watch` (foreground) or as a daemon under
    `htmlgraph serve`'s lifecycle.
- Result adapter: gating decisions (`{"decision":"block"}`) are dropped
  silently for the Warp source — there is no callback channel. They are
  still recorded so the dashboard surfaces "would-have-blocked" events.

### Phase 4 — `htmlgraph warp` launcher (parity)

Mirror `htmlgraph claude` / `htmlgraph yolo`. New `cmd/htmlgraph/warp.go`:

```
htmlgraph warp [args...]            # warp agent run with HtmlGraph context
htmlgraph warp --dev                # link packages/warp-extension/.agents
htmlgraph warp --resume <conv-id>   # warp run conversation get <id> + watch
htmlgraph warp --tmux               # tmux wrap for Codespaces, parity with claude
```

Responsibilities:

1. Set `HTMLGRAPH_PROJECT_DIR` (already required for worktree subagents).
2. Ensure `htmlgraph mcp serve` is reachable (start it if Warp can't spawn it
   via stdio — fallback for old Warp builds).
3. Start the Phase-3 observer as a child goroutine.
4. `exec` Warp with the appropriate args.
5. On exit, stop the observer cleanly (flush, close DB).

### Phase 5 — Switch to ACP when it lands

Tracked at `warpdotdev/warp#9233`. When ACP ships:

- Replace the SQLite poller in `internal/events/warp_observer.go` with an ACP
  subscriber. The `events.Source` interface stays unchanged; the underlying
  transport changes from "Diesel poll" to "ACP stream". This is the whole
  point of doing the interface extraction in Phase 3.
- Drop the schema pin once ACP is the supported channel.

### Phase 6 (deferred) — Native Warp plugin host

If a use case emerges that needs in-process integration (live UI overlays,
synchronous block decoration, custom command palette entries), implement a
JSON-RPC plugin against `app/src/plugin/service/`. Requires shipping a Rust
binary or subprocess that speaks Warp's plugin protocol. **No Phase-1–5
artifact needs to change.**

---

## Manifest Sketch

Concrete proposed addition to `packages/plugin-core/manifest.json`:

```jsonc
"targets": {
  // ... existing claude / codex / gemini ...
  "warp": {
    "outDir": "packages/warp-extension",
    "manifestPath": "warp-extension.json",
    "mcpPath": ".mcp.json",
    "skillsDir": ".agents/skills",
    "agentsDir": ".agents/agents",
    "commandNamespace": "htmlgraph"
  }
},
"hooks": {
  "events": [
    // existing entries unchanged.
    // No event has "warp" in its targets array — by design.
  ]
}
```

The `Target` struct in `internal/pluginbuild/manifest.go` will need
`SkillsDir`, `AgentsDir`, `McpPath` fields if they aren't already present
(`McpPath` already exists from the Codex target).

---

## Testing Strategy

- **Phase 1 (MCP)**: spawn `htmlgraph mcp serve` in a test, drive it with the
  Go MCP client (`github.com/anthropics/anthropic-cookbook` style harness or
  the `mcp-go` test kit), assert tool list and round-trip a
  `current_session` call against a seeded `.htmlgraph/`.
- **Phase 2 (build-ports)**: golden-file the emitted `packages/warp-extension/`
  tree under `internal/pluginbuild/testdata/warp/`. Same pattern as Codex/
  Gemini parity tests.
- **Phase 3 (observer)**: vendor a sample Warp SQLite fixture under
  `internal/events/testdata/warp.db` — a few conversations and messages
  hand-crafted to match the schema version we pin. Drive the observer
  against it, assert it produces the expected CloudEvent sequence.
- **Phase 4 (launcher)**: integration test gated on `WARP_BIN` env var —
  skipped unless Warp is installed locally.

---

## Acceptance Checklist

- [ ] `htmlgraph mcp serve` starts and answers `current_session` from
      stdio MCP.
- [ ] Warp's `.mcp.json` watcher picks up the htmlgraph entry without
      restart.
- [ ] `htmlgraph plugin build-ports --target warp` emits
      `packages/warp-extension/` with skills, agents, and `.mcp.json`.
- [ ] `htmlgraph plugin build-ports` (no `--target`) regenerates all four
      targets without touching unrelated trees.
- [ ] No file under `packages/warp-extension/` mentions `hooks.json` or
      `PreToolUse` — Warp is a no-hooks target.
- [ ] `htmlgraph warp watch` against a sample Warp DB writes session and
      message events into `.htmlgraph/sessions/`.
- [ ] Existing `go build ./... && go vet ./... && go test ./...` stays
      green.
- [ ] `./scripts/deploy-all.sh X.Y.Z --no-confirm` succeeds.

---

## Out of Scope (Explicitly)

- Writing a Rust crate for the JSON-RPC plugin host (Phase 6).
- Submitting an upstream PR to Warp introducing a hook lattice — HtmlGraph
  should adapt to Warp's native idioms, not the reverse.
- Replacing the existing Claude/Codex/Gemini hook adapters with the new
  observer pattern. Hooks remain the right contract for harnesses that
  expose them — the observer is *additive* for harnesses that don't.

---

## Why This Order

Phase 1 unblocks every MCP-aware harness, not just Warp — it has the
broadest payoff per unit of work. Phase 2 is mechanical and small. Phase 3
is the load-bearing piece for the *philosophy* (HtmlGraph still observes
every session even when there are no hooks); doing it before Phase 4 means
the launcher has something real to hook into. Phase 5 is opportunistic.
Phase 6 is "only if a concrete need emerges."

The thread tying them together: HtmlGraph's contract with a harness was
historically *"send me hook events on stdin."* After Warp, the contract
becomes *"expose a session model HtmlGraph can subscribe to — by hook, by
MCP, by SQLite, or by ACP."* That is the philosophical generalization the
Warp port forces, and it is a strict improvement.
