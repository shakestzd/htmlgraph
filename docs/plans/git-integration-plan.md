# Git Integration Plan

## Summary

Selectively adopt git primitives where they provide concrete improvements over
htmlgraph's current architecture. This is NOT a replatforming — SQLite remains
the query layer, HTML files remain the canonical store, hooks remain the
real-time event path. These are five targeted changes that reduce custom code,
improve accuracy, and unlock ecosystem interoperability.

## Context

Research into the git-native tooling landscape (git-bug, Git AI, Agent Trace,
Entire CLI, Backlog.md) confirmed that every project attempting "git as
database" eventually adds a real query layer alongside it. htmlgraph's current
architecture — HTML as canonical store, SQLite as query index — is structurally
sound. These changes steal the good ideas without the architectural baggage.

---

## Change 1: GitHub Actions CI (Quality Gate Enforcement)

**Priority:** Highest — zero risk, additive only
**Effort:** Small

### Problem

Local quality gates (`quality_gate.go`, `pretooluse.go`) can be bypassed via
`--no-verify`, skipped on fresh clones, or absent for agents without hooks.
The `.github/BRANCH_PROTECTION.md` documents required status checks that don't
actually exist as workflows.

### Solution

Add `.github/workflows/ci.yml` running the same checks server-side on every PR.
Enable "Require status checks to pass before merging" on `main`.

### Implementation

**New file:** `.github/workflows/ci.yml`

```yaml
name: CI
on:
  pull_request:
    branches: [main]
  push:
    branches: [main]

jobs:
  quality-gate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: packages/go/go.mod
      - name: Build
        run: cd packages/go && go build ./...
      - name: Vet
        run: cd packages/go && go vet ./...
      - name: Test
        run: cd packages/go && go test ./...
```

**Existing files:** No changes. Local hooks remain for fast feedback — the
Action is the enforcement backstop. Defense in depth.

### What Changes

| Before | After |
|--------|-------|
| Quality gates are advisory (local only) | Quality gates are enforceable (server-side) |
| Branch protection docs reference phantom checks | Branch protection references real CI workflow |
| Agents can skip hooks | Merges blocked until CI passes |

---

## Change 2: Incremental Reindex via `git diff`

**Priority:** High — easy performance win
**Effort:** Small

### Problem

`reindex.go` globs and parses every HTML file on every run. As `.htmlgraph/`
grows, this gets linearly slower. With 500 work items, reindex parses 500
files even if only 3 changed.

### Solution

Track the last-indexed commit in a metadata table. On reindex, ask git which
files changed since then. Only parse those files.

### Implementation

**Modified file:** `packages/go/internal/db/schema.go`

Add a `metadata` key-value table:

```sql
CREATE TABLE IF NOT EXISTS metadata (
    key   TEXT PRIMARY KEY,
    value TEXT
)
```

Add `GetMetadata(db, key)` and `SetMetadata(db, key, value)` helpers.

**Modified file:** `packages/go/cmd/htmlgraph/reindex.go`

Before the two-pass reindex loop:

```go
func runReindex(database *sql.DB, htmlgraphDir string, fullReindex bool) error {
    lastCommit := db.GetMetadata(database, "last_indexed_commit")
    currentHEAD := headCommit(filepath.Dir(htmlgraphDir))

    var changedFiles []string
    if lastCommit != "" && !fullReindex {
        // Ask git what changed — O(changed) instead of O(all)
        cmd := exec.Command("git", "diff", "--name-only", lastCommit, currentHEAD,
            "--", ".htmlgraph/")
        out, err := cmd.Output()
        if err == nil && len(out) > 0 {
            changedFiles = strings.Split(strings.TrimSpace(string(out)), "\n")
        }
        // If git diff fails (shallow clone, force push), fall through to full
    }

    if changedFiles != nil {
        // Incremental: parse only changed files
        for _, f := range changedFiles {
            reindexSingleFile(database, filepath.Join(projectRoot, f))
        }
    } else {
        // Full: existing glob-and-parse behavior
        reindexFeatureDir(database, htmlgraphDir, "tracks")
        reindexFeatureDir(database, htmlgraphDir, "features")
        // ...
    }

    // Purge stale entries (unchanged from current behavior)
    purgeStaleEntries(database, htmlgraphDir)

    // Record indexed commit
    db.SetMetadata(database, "last_indexed_commit", currentHEAD)
    return nil
}
```

**CLI flag:** Add `--full` to force full reparse: `htmlgraph reindex --full`

### What Changes

| Before | After |
|--------|-------|
| Reindex parses all files every time | Reindex parses only changed files |
| O(all files) | O(changed files) |
| ~500ms for 500 items | ~3ms for 3 changed items |
| No escape hatch | `--full` flag for when incremental misses something |

---

## Change 3: Derive Timestamps from Git History

**Priority:** Medium — data model simplification
**Effort:** Medium

### Problem

HTML work items carry `data-created` and `data-updated` attributes baked into
the file. These drift from reality — manual edits don't update `data-updated`,
rebases make `data-created` wrong, and the attributes duplicate what git
already knows immutably.

### Solution

Remove timestamp attributes from HTML. Derive them from git at index time.

### Implementation

**Modified file:** `packages/go/internal/workitem/templates/node.gohtml`

Remove lines:
```html
data-created="{{.CreatedAt}}"
data-updated="{{.UpdatedAt}}"
```

**Modified file:** `packages/go/internal/workitem/htmlwriter.go`

Remove `CreatedAt` and `UpdatedAt` from the template data struct and the
`fmtTime()` calls that populate them.

**Modified file:** `packages/go/internal/htmlparse/parser.go`

Stop parsing `data-created` and `data-updated` (they won't exist in new files).
For backwards compatibility during migration, still read them if present and
treat as fallback.

**Modified file:** `packages/go/cmd/htmlgraph/reindex.go`

At index time, derive timestamps from git:

```go
func gitTimestamps(projectRoot, filePath string) (created, updated time.Time) {
    // Single git command gets both in one call
    cmd := exec.Command("git", "log", "--format=%aI", "--follow", "--", filePath)
    cmd.Dir = projectRoot
    out, err := cmd.Output()
    if err != nil || len(out) == 0 {
        return time.Now(), time.Now() // fallback for untracked files
    }
    lines := strings.Split(strings.TrimSpace(string(out)), "\n")
    // First line = most recent commit (updated), last line = first commit (created)
    updated, _ = time.Parse(time.RFC3339, lines[0])
    created, _ = time.Parse(time.RFC3339, lines[len(lines)-1])
    return
}
```

For batch efficiency during full reindex, use a single git command:

```bash
git log --format='%aI %H' --name-only -- .htmlgraph/features/ .htmlgraph/bugs/
```

Parse the output once to build a map of `filepath → (created, updated)`.

### Migration

Existing HTML files with `data-created`/`data-updated` continue to work — the
parser reads them as fallback. New files omit them. Over time, as files are
re-rendered, the attributes disappear. No breaking change.

### What Changes

| Before | After |
|--------|-------|
| Timestamps stored in HTML and SQLite | Timestamps derived from git, cached in SQLite |
| Manual edits leave `data-updated` stale | Timestamps always reflect actual git history |
| HTML files carry redundant metadata | HTML files are simpler |
| Timestamps survive `git clone` in HTML | Timestamps survive `git clone` in git history (always did) |

---

## Change 4: Derive `feature_files` from Git History

**Priority:** Medium — accuracy improvement
**Effort:** Medium

### Problem

The `feature_files` table is populated by hooks (`pretooluse.go`) recording
which files each tool touches. This misses:
- Manual edits committed outside a Claude Code session
- Files touched by agents without htmlgraph hooks
- Historical work before htmlgraph was installed

### Solution

Derive file-to-feature mapping from the `git_commits` table (which already
links commits to features) plus `git diff-tree` (which shows files in each
commit). Rebuild during reindex instead of appending on every tool call.

### Implementation

**Modified file:** `packages/go/internal/hooks/pretooluse.go`

Remove `UpsertFeatureFile` calls from the hot path. Tool calls no longer write
to `feature_files` on every invocation.

**New function in:** `packages/go/cmd/htmlgraph/reindex.go`

```go
func reindexFeatureFiles(database *sql.DB, projectRoot string) error {
    rows, err := database.Query(`
        SELECT DISTINCT feature_id, commit_hash 
        FROM git_commits 
        WHERE feature_id IS NOT NULL`)
    if err != nil { return err }
    defer rows.Close()

    for rows.Next() {
        var featureID, hash string
        rows.Scan(&featureID, &hash)

        cmd := exec.Command("git", "diff-tree", "--no-commit-id", "-r", "--name-only", hash)
        cmd.Dir = projectRoot
        out, _ := cmd.Output()

        for _, filePath := range strings.Split(strings.TrimSpace(string(out)), "\n") {
            if filePath != "" {
                db.UpsertFeatureFile(database, featureID, filePath, "commit", "", time.Now())
            }
        }
    }
    return nil
}
```

Call `reindexFeatureFiles()` as a third pass in the reindex command, after
tracks and features.

**Modified file:** `packages/go/cmd/htmlgraph/backfill.go`

Simplify — backfill IS the reindex now. Remove duplicated logic.

### What Changes

| Before | After |
|--------|-------|
| `feature_files` populated by hooks (hot path write per tool call) | `feature_files` rebuilt from git during reindex |
| Misses manual commits, non-hooked agents | Captures all files from all commits linked to features |
| Hook overhead on every Write/Edit/Glob | Zero hook overhead for file tracking |
| Requires backfill command for historical data | Reindex handles historical data automatically |
| Data accuracy depends on hooks being installed | Data accuracy depends on git history (always available) |

---

## Change 5: Adopt Agent Trace Attribution Format

**Priority:** Medium — ecosystem play
**Effort:** Medium

### Problem

htmlgraph uses a custom `traceparentEntry` struct (`attribution.go:15-20`)
with a custom JSON format written to temp files. The Agent Trace RFC (backed by
Cursor, Cloudflare, Vercel, Google Jules, Git AI, and others) defines a common
format for AI code attribution. htmlgraph's custom format is an island.

### Solution

Align the attribution data format with Agent Trace. This makes htmlgraph's
attribution data readable by Git AI, Agent Blame (Mesa), and Cursor's tooling —
and vice versa.

### Implementation

**Modified file:** `packages/go/internal/hooks/attribution.go`

Align `traceparentEntry` with Agent Trace schema:

```go
// agentTraceRecord follows the Agent Trace RFC for interoperability
// with Git AI, Agent Blame, Cursor, and other tools.
type agentTraceRecord struct {
    Version       string  `json:"version"`        // "0.1.0" (Agent Trace version)
    ContributorID string  `json:"contributor_id"`  // agent identifier
    Tool          string  `json:"tool"`            // "claude-code", "cursor", etc.
    SessionID     string  `json:"session_id"`      // htmlgraph session ID
    ParentSession string  `json:"parent_session,omitempty"`
    Timestamp     string  `json:"timestamp"`       // RFC3339
    TraceID       string  `json:"trace_id"`        // correlation ID
    ParentSpanID  string  `json:"parent_span_id,omitempty"`
}
```

**Modified file:** `packages/go/internal/hooks/attribution.go`

Update `writeTraceparent()` to emit Agent Trace format. Update
`claimTraceparent()` to read both old and new formats (migration period).

**Modified file:** `packages/go/internal/hooks/subagent_start.go`

Include Agent Trace contributor records in delegation events.

### Migration

Version the format with a `version` field. During transition, read both
formats. Old temp files are cleaned up within 5 minutes (existing TTL in
`claimTraceparent`), so migration is automatic.

### What Changes

| Before | After |
|--------|-------|
| Custom traceparent format | Agent Trace RFC-compatible format |
| Only htmlgraph can read attribution data | Git AI, Agent Blame, Cursor can read it too |
| Attribution is an island | Attribution participates in the ecosystem |

### Risk

The Agent Trace RFC is still evolving. Pin to a specific version (`0.1.0`) and
gate on the `version` field. If the RFC changes, add a new version reader
without breaking the old one.

---

## Implementation Order

```
1. GitHub Actions CI          [Small effort, zero risk, immediate value]
2. Incremental reindex        [Small effort, big performance win]
3. Derive timestamps from git [Medium effort, data model cleanup]
4. Derive feature_files       [Medium effort, accuracy improvement]
5. Agent Trace format         [Medium effort, ecosystem interop]
```

Changes 1-2 are independent and can ship immediately.
Changes 3-4 both touch `reindex.go` and should be done together.
Change 5 is independent and can ship whenever.

---

## What We Explicitly Do NOT Change

| Keep as-is | Why |
|------------|-----|
| HTML files as canonical store | Human-readable, browser-viewable, git-diffable |
| SQLite as query layer | Sub-millisecond queries, offline, no sync conflicts |
| Hook-based session tracking | Sessions are not branches (many-to-many relationship) |
| `data-agent-assigned` attribute | Agent identity in HTML is useful for rendering without DB |
| `graph_edges` table | Semantic relationships (blocks, implements) have no git equivalent |
| `agent_events` table | Real-time telemetry needs sub-millisecond writes |
| `.active-session` fallback | Worktree subagent propagation has no git alternative |
