# Cross-Environment Absolute-Path Drift Audit

## Context

HtmlGraph runs in multiple environments for the same repo clone: macOS host, Linux
devcontainer, Codespaces, and CI. Artifacts that bake in a host-specific absolute path
(`/Users/<name>/…`, `/home/<name>/…`, `/workspaces/<name>/…`, `/private/var/folders/…`)
work on one host and break on another. Three P1/P2 bugs in track `trk-787f57d3` were all
expressions of this class. This audit enumerates every artifact that can carry such a
path, classifies it, and records remediation state.

## Classification Key

- **ephemeral** — regenerated on every session start; drift self-heals.
- **relative-rewriteable** — can store a placeholder or repo-relative path; fix forward.
- **must-stay-absolute** — value is absolute by nature; must be detected and repaired on
  session entry.

## Audit Matrix

| # | Artifact | Classification | Remediation | Status |
|---|----------|---------------|-------------|--------|
| 1 | `.htmlgraph/plans/*.yaml` body text | relative-rewriteable | Use placeholders; pre-commit guardrail | Done (bug-f4760452, bug-4b6d8369) |
| 2 | `.htmlgraph/bugs/*.html` body text | relative-rewriteable | Use placeholders; pre-commit guardrail | Done (bug-f4760452, bug-4b6d8369) |
| 3 | `.htmlgraph/sessions/*.html` `cwd` / `project_dir` meta | ephemeral | Emitted fresh each session; no on-disk persistence across hosts | Accepted |
| 4 | SQLite `sessions.project_dir` | ephemeral | Per-session row; read-only across session starts on other hosts | Accepted |
| 5 | SQLite `sessions.transcript_path` | ephemeral | Mirrors Claude Code's local transcript; consumed only when present | Accepted |
| 6 | `.claude/worktrees/<trk>/.git` gitdir line | must-stay-absolute | Detect + rewrite on SessionStart | Done (bug-e1c968fe) |
| 7 | `.claude/settings.local.json` | must-stay-absolute | Documented cross-machine drift in `memory/settings_local_cross_machine_drift.md`; should be `.gitignore`d | Open (follow-up 1) |
| 8 | `~/.claude/projects/<slug>/memory/*.md` body prose | relative-rewriteable | Placeholders in examples; user-authored content | Accepted |
| 9 | `plugin/hooks/bin/htmlgraph` (compiled binary) | ephemeral | Rebuilt on `htmlgraph build`; no runtime path references persisted | Accepted |
| 10 | `plugin/skills/**/*.md` reference examples | relative-rewriteable | Skill text uses `<repo-root>` placeholder or `$HTMLGRAPH_PROJECT_DIR`; audit skill corpus quarterly | Open (follow-up 2) |
| 11 | `.env`, `.env.local` | relative-rewriteable | None present today; recommend adding `.env*` to `.gitignore` | Accepted (no artifact) |

**Summary:** 5 ephemeral · 4 relative-rewriteable · 2 must-stay-absolute.
**Remediated:** 4 (bugs f4760452, 4b6d8369, e1c968fe, plus in-memory acknowledgement for 3–5).
**Open:** 2 follow-ups listed below.

## Deep-Dive Per Artifact

### `.htmlgraph/plans/*.yaml` body text (relative-rewriteable)

Plan YAMLs store free-form text in `done_when`, `evidence`, and `rationale` fields.
Human-authored text occasionally embeds absolute paths (e.g. Copilot session-state file
locations, codex config paths).

- **Who writes**: agents during `htmlgraph plan` interactive flows and manual edits.
- **Remediation**: two known hits in `plan-c248b73f.yaml` scrubbed in bug-f4760452;
  the `htmlgraph check host-paths` guardrail from bug-4b6d8369 flags future drift at
  commit time.

### `.htmlgraph/bugs/*.html` body text (relative-rewriteable)

Bug HTMLs carry descriptions written by agents and humans. Same failure mode as plans:
free-text embeds paths to files that only exist on one host.

- **Who writes**: `htmlgraph bug create/update`, direct file edits during investigations.
- **Remediation**: hit in `bug-71fc095f.html` scrubbed in bug-f4760452; guardrail covers
  forward.

### `.htmlgraph/sessions/*.html` and SQLite `sessions` rows (ephemeral)

Each session HTML and its SQLite row embeds `cwd` / `project_dir` / `transcript_path`
captured at SessionStart. When a session row generated on host A is read on host B, the
values are historical telemetry — they are not re-used to locate files. The dashboard
renders them as text.

- **Who writes**: `internal/hooks/session_start.go`, `internal/hooks/session_html.go`.
- **Accepted** because these are immutable per-session records, not live pointers.

### `.claude/worktrees/<trk>/.git` gitdir line (must-stay-absolute)

`git worktree add` writes a single-line `.git` file: `gitdir: /absolute/path/to/.git/worktrees/<name>`.
When the worktree is created on host A and opened on host B, the absolute path does not
resolve and every git command fails.

- **Who writes**: `git worktree add`, invoked by `htmlgraph worktree` helpers.
- **Remediation**: `internal/worktree/repair.go::RepairGitdirFromRepoRoot` detects a
  stale gitdir and rewrites it using the current repo root. Wired into `SessionStart` so
  every Claude Code session self-repairs before the first git command runs.
- **Landed in**: bug-e1c968fe.

### `.claude/settings.local.json` (must-stay-absolute, **open**)

User-specific Claude Code settings file. Records absolute paths (`statusLine.command`,
permissions allow/deny globs). The repo's shared `.claude/settings.json` points at
`/home/vscode/.claude/omp-claude-wrapper.sh` — a Linux-only path that breaks on macOS.

- **Who writes**: `claude` itself, plus manual edits.
- **Follow-up 1**: ensure the file is `.gitignore`d (it may already be — verify). If it
  must be committed, rewrite absolute paths to `${HOME}`-relative or platform-detected
  lookups. Alternatively, emit a session-start reconciliation that overwrites stale
  machine-local keys.

### `~/.claude/projects/<slug>/memory/*.md` (relative-rewriteable)

Auto-memory files under the user's home. Examples and references may embed
host-specific paths. These are user-authored; drift doesn't break anything, but
cross-machine users can get confused by paths that reference a directory layout they
don't have.

- **Accepted** with the convention that memory prose uses `<repo-root>` / `<user-home>`
  placeholders and `$HTMLGRAPH_PROJECT_DIR` where applicable.

### `plugin/hooks/bin/htmlgraph` (ephemeral)

Compiled Go binary. No runtime paths baked in; rebuilt on `htmlgraph build`. Cross-host
drift is zero because the binary is never committed.

### `plugin/skills/**/*.md` reference examples (relative-rewriteable, **open**)

Skills teach agents CLI commands. Some historically embedded absolute paths as examples.
The execute skill's `--format json` bug (bug-7ca3638b) and the same skill's redundant
git calls (bug-f07a612e) are cousins of this class — skills as code that needs review.

- **Follow-up 2**: quarterly skill-corpus grep for host-local paths. The skill-flag
  validator from bug-61dc9267 is the natural hook point — extend it to also flag
  literal `/Users/`, `/home/`, `/workspaces/<name>/` occurrences inside fenced code
  blocks.

### `.env`, `.env.local` (accepted, no artifact)

No `.env*` files present in the repo today. Recommend adding `.env*` to the top-level
`.gitignore` as a defensive measure even though nothing currently exists.

## Open Follow-Ups

1. **`.claude/settings.local.json` gitignore audit.** Verify the file is
   `.gitignore`d at the repo level (existing memory entry suggests users have been
   burned by stale commits). If committed state is required, document the exact
   subset of keys that must stay relative and add a session-start reconciler.
   *Priority: Medium.*

2. **Skill-corpus path lint.** Extend the `TestSkillFlagsIntegration` scanner from
   bug-61dc9267 to also match `/Users/[a-z]+/`, `/home/[a-z]+/`, and
   `/workspaces/[a-z]+/` inside fenced code blocks in `plugin/skills/**/*.md`,
   `plugin/commands/**/*.md`, and `plugin/agents/**/*.md`. Fail the build on any hit
   that is not in an explicit allowlist.
   *Priority: Low.*

## Related Bugs

- **bug-f4760452** — scrubbed two committed artifacts
- **bug-4b6d8369** — pre-commit guardrail (`htmlgraph check host-paths`)
- **bug-e1c968fe** — worktree `.git` auto-repair on SessionStart
- **bug-5150e42a** — this audit

All four together retire the immediate class and leave two documented follow-ups.
