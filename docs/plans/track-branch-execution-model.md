# Track Branch Execution Model

## Summary

Tracks execute on dedicated git branches. Features within a track are expressed
as commits (sequential) or ephemeral agent branches (parallel), not as durable
feature branches. One track = one branch = one PR = one merge to main.

## Context

htmlgraph currently creates a branch per feature in YOLO mode
(`yolo.go:createFeatureWorktree`). This produces many small branches and PRs,
each representing a fragment of a larger initiative. Reviewers see pieces, not
the whole. Merge conflicts between features in the same track get resolved on
main rather than within the track.

The insight: **features are a planning concept (what to build), tracks are an
execution concept (where to build it).** Branches should mirror execution.

---

## The Model

```
main ─────────────────────────────────────────────── main
       \                                           /
        ── trk-3030989f ──────────────────────────
           │                                     │
           │  feat-001: commits a1, a2, a3       │
           │  feat-002: commits b1, b2           │
           │  feat-003: commits c1, c2, c3, c4   │
           │                                     │
           └─────────────────────────────────────┘
                        ONE PR
```

### Branch Hierarchy

```
main                          durable, protected
└── trk-{track-id}            durable, one per track, merges to main
    ├── trk-{id}/agent-{n}    ephemeral, parallel execution only
    └── trk-{id}/agent-{n}    ephemeral, deleted after merge to track branch
```

### Naming Convention

| Branch | Purpose | Lifetime |
|--------|---------|----------|
| `trk-{track-id}` | Track execution space | Created at track start, deleted after merge to main |
| `trk-{track-id}/agent-{task-id}` | Parallel agent worktree | Created at dispatch, deleted after merge to track branch |

### Feature Boundaries

Features are NOT branches. They are visible through:

1. **Commit message prefixes:** `feat-15c458aa: add auth handler`
2. **Git tags on the track branch:** `git tag feat-15c458aa/done <commit>`
3. **Commit ranges:** first and last commit with a given prefix

This means `git log --grep='feat-15c458aa' trk-3030989f` shows all work for
that feature — no branch needed.

---

## Execution Modes

### Sequential Execution (Single Agent)

One agent works through features in order, all on the track branch.

```
trk-3030989f
    │
    ├── a1 feat-001: add auth types
    ├── a2 feat-001: add auth handler
    ├── a3 feat-001: add auth tests
    ├── b1 feat-002: add user model
    ├── b2 feat-002: add user API
    ├── c1 feat-003: add dashboard layout
    ├── c2 feat-003: add dashboard charts
    └── c3 feat-003: add dashboard tests
```

No feature branches. No merges within the track. Linear history.

### Parallel Execution (Multiple Agents)

Multiple agents work simultaneously. Each gets an ephemeral branch + worktree
forked from the track branch. After completion, merge back to the track branch.
Delete the ephemeral branch.

```
trk-3030989f ──────────┬────────────────── merge ← agent-1 ── merge ← agent-2 ──
                        │                    ↑                   ↑
                        ├── agent-feat-001 ──┘                   │
                        └── agent-feat-002 ──────────────────────┘
                            (ephemeral)          (ephemeral)
```

The dispatch loop from `execute/SKILL.md`:

```
LOOP:
  1. Query unblocked tasks
  2. For each: create ephemeral branch + worktree from track branch
  3. Dispatch agents in parallel
  4. Wait for completion
  5. Merge each ephemeral branch → track branch
  6. Delete ephemeral branches
  7. Run quality gates on track branch
  8. Any newly unblocked tasks? → LOOP
  9. Done? → PR from track branch to main
```

### Conflict Resolution

Conflicts between features are resolved **within the track**, not on main:

- Agent-1 and Agent-2 both modify `auth.go`
- Agent-1 finishes first, merges to `trk-3030989f`
- Agent-2 finishes, merge to `trk-3030989f` has conflicts
- Resolve on the track branch (or rebase agent-2's work)
- Main never sees the conflict

This is strictly better than the current model where both merge to main
independently and conflicts surface there.

---

## Implementation

### Phase 1: Track Worktree Creation

**Modified file:** `packages/go/cmd/htmlgraph/yolo.go`

Replace `createFeatureWorktree` with `createTrackWorktree`:

```go
// createTrackWorktree creates a git worktree for track execution.
// Branch: trk-{trackID}, Path: .claude/worktrees/{trackID}
func createTrackWorktree(trackID, projectRoot string) (string, func(), error) {
    worktreePath := filepath.Join(projectRoot, ".claude", "worktrees", trackID)
    branchName := "trk-" + trackID
    noop := func() {}

    // Reuse existing worktree
    if _, err := os.Stat(worktreePath); err == nil {
        fmt.Printf("  Worktree: %s (reusing existing)\n", worktreePath)
        return worktreePath, noop, nil
    }

    if err := os.MkdirAll(filepath.Dir(worktreePath), 0755); err != nil {
        return "", noop, fmt.Errorf("create worktrees dir: %w", err)
    }

    cmd := exec.Command("git", "-C", projectRoot, "worktree", "add",
        worktreePath, "-b", branchName)
    if out, err := cmd.CombinedOutput(); err != nil {
        return "", noop, fmt.Errorf("git worktree add: %w\n%s", err, out)
    }

    // Store track metadata in branch config
    exec.Command("git", "-C", worktreePath, "config",
        "branch."+branchName+".htmlgraph-track", trackID).Run()
    exec.Command("git", "-C", worktreePath, "config",
        "branch."+branchName+".htmlgraph-started",
        time.Now().UTC().Format(time.RFC3339)).Run()

    fmt.Printf("  Worktree: %s (branch: %s)\n", worktreePath, branchName)

    cleanup := func() {
        exec.Command("git", "-C", projectRoot, "worktree", "remove",
            "--force", worktreePath).Run()
    }
    return worktreePath, cleanup, nil
}
```

**Modified file:** `packages/go/cmd/htmlgraph/yolo.go`

Update `launchYoloDefault` and `launchYoloDev`: when `--track` is provided,
create a track worktree. When `--feature` is provided, check if the feature
belongs to a track — if so, use the track's worktree.

```go
// In launchYoloDefault:
switch {
case trackID != "":
    // Track mode: create/reuse track worktree
    workDir, cleanup, err = createTrackWorktree(trackID, projectRoot)
case featureID != "":
    // Feature mode: find parent track, use its worktree
    parentTrack := findTrackForFeature(featureID, projectRoot)
    if parentTrack != "" {
        workDir, cleanup, err = createTrackWorktree(parentTrack, projectRoot)
    } else {
        // Standalone feature (no track): create feature worktree (existing behavior)
        workDir, cleanup, err = createFeatureWorktree(featureID, projectRoot)
    }
}
```

### Phase 2: Parallel Agent Dispatch

**Modified skill:** `packages/go-plugin/skills/execute/SKILL.md`

Update the agent dispatch pattern to use ephemeral branches off the track:

```markdown
## Dispatch Pattern

Each agent gets a worktree branched from the TRACK branch, not from main:

    Agent(
        description="feat-001: Add check command",
        subagent_type="htmlgraph:sonnet-coder",
        isolation="worktree",
        prompt="
            You are working on track trk-3030989f.
            Your task: feat-001 — Add check command.
            
            IMPORTANT: Your branch was created from the track branch.
            Commit with prefix: feat-001:
            Example: git commit -m 'feat-001: add check command handler'
        "
    )
```

**New function in:** `packages/go/cmd/htmlgraph/yolo.go`

```go
// createAgentWorktree creates an ephemeral worktree for a parallel agent.
// Branches from the track branch, not from main.
func createAgentWorktree(trackID, taskID, projectRoot string) (string, func(), error) {
    trackBranch := "trk-" + trackID
    agentBranch := trackBranch + "/agent-" + taskID
    worktreePath := filepath.Join(projectRoot, ".claude", "worktrees",
        trackID, "agent-"+taskID)

    if err := os.MkdirAll(filepath.Dir(worktreePath), 0755); err != nil {
        return "", nil, err
    }

    cmd := exec.Command("git", "-C", projectRoot, "worktree", "add",
        worktreePath, "-b", agentBranch, trackBranch)
    if out, err := cmd.CombinedOutput(); err != nil {
        return "", nil, fmt.Errorf("git worktree add: %w\n%s", err, out)
    }

    cleanup := func() {
        // Merge to track branch, then remove
        exec.Command("git", "-C", projectRoot, "merge", agentBranch).Run()
        exec.Command("git", "-C", projectRoot, "worktree", "remove",
            "--force", worktreePath).Run()
        exec.Command("git", "-C", projectRoot, "branch", "-d", agentBranch).Run()
    }
    return worktreePath, cleanup, nil
}
```

### Phase 3: Track Status from Git

**New command:** `htmlgraph track status <track-id>`

Derives execution status entirely from git:

```go
func trackStatusCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "track-status [track-id]",
        Short: "Show track execution status derived from git",
        RunE: func(cmd *cobra.Command, args []string) error {
            trackID := args[0]
            branch := "trk-" + trackID
            projectRoot := mustFindProjectRoot()

            // Check if branch exists
            if !branchExists(branch, projectRoot) {
                fmt.Printf("Track %s: not started (no branch)\n", trackID)
                return nil
            }

            // Merged to main?
            if isBranchMerged(branch, "main", projectRoot) {
                mergeDate := mergeCommitDate(branch, projectRoot)
                fmt.Printf("Track %s: completed (merged %s)\n", trackID, mergeDate)
                return nil
            }

            // Active — show feature breakdown
            commits := gitLog(projectRoot, "main", branch)
            features := groupByPrefix(commits) // group by "feat-xxx:" prefix

            fmt.Printf("Track %s: in-progress (%d commits, %d features)\n",
                trackID, len(commits), len(features))
            fmt.Println()

            for prefix, featureCommits := range features {
                files := filesInCommits(projectRoot, featureCommits)
                fmt.Printf("  %s: %d commits, %d files\n",
                    prefix, len(featureCommits), len(files))
            }

            // Show diff stat vs main
            fmt.Println()
            stat := diffStat(projectRoot, "main", branch)
            fmt.Printf("Total: %s\n", stat)

            return nil
        },
    }
}
```

### Phase 4: Yolo Guard Updates

**Modified file:** `packages/go/internal/hooks/yolo_guard.go`

Update `checkYoloWorktreeGuard` to understand track branches:

```go
func checkYoloWorktreeGuard(toolName, branch string, yolo bool) string {
    if !yolo {
        return ""
    }
    switch toolName {
    case "Write", "Edit", "MultiEdit":
    default:
        return ""
    }
    // Allow: track branches, feature branches, agent branches
    if strings.HasPrefix(branch, "trk-") ||
        strings.HasPrefix(branch, "yolo-") {
        return ""
    }
    if branch == "main" || branch == "master" {
        return "YOLO mode requires a track branch or worktree. " +
            "Launch with: htmlgraph yolo --track <track-id>"
    }
    return ""
}
```

### Phase 5: PR Workflow

When a track is complete, create one PR:

```bash
# All track work in one PR
gh pr create \
  --base main \
  --head trk-3030989f \
  --title "Track: Planning Workflow" \
  --body "## Features
  - feat-001: Add check command
  - feat-002: Add budget command  
  - feat-003: Add dashboard layout
  
  ## Stats
  $(git diff --stat main...trk-3030989f)"
```

One PR per track means:
- Reviewers see the complete initiative
- CI runs on the integrated result
- One merge commit on main per initiative
- Clean main branch history

---

## Migration from Feature Branches

### Backwards Compatibility

- `createFeatureWorktree` remains for standalone features (no parent track)
- Existing `yolo-feat-*` branches continue to work
- `checkYoloWorktreeGuard` accepts both `trk-*` and `yolo-*` prefixes
- No existing data or workflows break

### Deprecation Path

1. **Now:** Add track worktree support alongside feature worktrees
2. **Next:** Default `htmlgraph yolo --feature` to use parent track's branch
   when one exists
3. **Later:** Remove `createFeatureWorktree`, require all YOLO work to have a
   track. Standalone features get auto-wrapped in a single-feature track.

---

## What Git Tells You (No SQLite Needed)

| Question | Git Command |
|----------|-------------|
| Is track started? | `git branch --list 'trk-{id}'` |
| Is track complete? | `git branch --merged main \| grep 'trk-{id}'` |
| How many commits? | `git rev-list --count main..trk-{id}` |
| Which features? | `git log --format=%s main..trk-{id} \| grep -oP '^feat-\w+' \| sort -u` |
| Files changed? | `git diff --stat main...trk-{id}` |
| When did it start? | `git log trk-{id} --reverse --format=%ci \| head -1` |
| When was it merged? | `git log main --merges --grep='trk-{id}' --format=%ci` |
| Active agent branches? | `git branch --list 'trk-{id}/agent-*'` |
| Which agents are done? | `git branch --merged trk-{id} --list 'trk-{id}/agent-*'` |
| Track metadata? | `git config --get branch.trk-{id}.htmlgraph-track` |

---

## Directory Layout

```
.claude/worktrees/
├── trk-3030989f/                   # Track worktree (persistent during execution)
│   ├── .git                        # Worktree git link
│   ├── packages/                   # Full repo checkout on track branch
│   └── .htmlgraph/                 # Shared work item store
│
│   # During parallel execution only:
│   ├── agent-feat-001/             # Ephemeral agent worktree
│   └── agent-feat-002/             # Ephemeral agent worktree
│
└── trk-8a2b1c3d/                   # Another track
```

---

## Relationship to Git Integration Plan

This model builds on changes 2 and 4 from the Git Integration Plan:

- **Change 2 (incremental reindex):** Track branches accumulate commits.
  `git diff` between last-indexed and current HEAD on the track branch drives
  incremental reindexing within the track's worktree.

- **Change 4 (derive feature_files):** With commit message prefixes encoding
  feature IDs, `git log --grep='feat-xxx' --format=%H trk-{id}` gives you the
  commits, and `git diff-tree` gives you the files. The feature→file mapping
  is fully derivable from git history on the track branch.

The track branch model makes both changes more natural because the branch IS
the execution boundary — everything needed to understand the track's work lives
in its git history.
