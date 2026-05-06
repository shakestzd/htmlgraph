# Git-Based Continuity Spine Architecture

## Overview

Wipnote uses Git as a universal continuity spine that enables agent-agnostic session tracking and event logging. This architecture eliminates dependence on platform-specific hooks (like Claude Code plugin hooks) and makes Wipnote work with ANY coding agent.

**Core Principle**: Git commits are universal continuity points that work regardless of which agent wrote the code.

## Architecture Components

### 1. Git Hooks as Continuity Anchors

Git provides universal hooks that work regardless of which agent/tool is writing code. Wipnote leverages these to create a continuity spine:

```
┌─────────────────────────────────────────────────────────────────┐
│                       GIT COMMIT GRAPH                           │
│  (Universal source of truth - works with all agents)            │
└────────────────┬────────────────────────────────────────────────┘
                 │
    ┌────────────┼────────────┬────────────────────────┐
    │            │            │                        │
┌───▼────┐  ┌───▼────┐  ┌───▼────┐              ┌───▼────┐
│ Commit │  │ Commit │  │ Commit │     ...      │ Commit │
│  abc1  │──│  abc2  │──│  abc3  │──────────────│  abc4  │
│        │  │        │  │        │              │        │
│session │  │session │  │session │              │session │
│  S1    │  │  S1    │  │  S2    │              │  S2    │
└────────┘  └────────┘  └────────┘              └────────┘
     │           │           │                        │
     └───────────┴───────────┴────────────────────────┘
                         │
                ┌────────▼─────────┐
                │  Event Log       │
                │  (.wipnote/    │
                │   events/)       │
                └──────────────────┘
```

#### Hook Types and Responsibilities

**`post-commit`** - Primary continuity anchor
- Logs commit hash, branch, author, message
- Records files changed (insertions/deletions)
- Links to active features via:
  - Explicit references in commit message (`feature-xyz`)
  - Active features from session state
  - File pattern matching to feature file patterns
- Creates GitCommit events in `.wipnote/events/`

**`post-checkout`** - Branch continuity
- Tracks branch switches (`main` → `feature/auth`)
- Detects context switches between work items
- Enables session reconstruction across branches
- Links work that spans multiple branches

**`post-merge`** - Integration events
- Logs successful merges
- Tracks source and target branches
- Records integration milestones
- Detects merge commits (multiple parents)

**`pre-push`** - Team boundaries
- Logs what's being pushed before remote update
- Marks "going public" events
- Tracks when local work becomes shared
- Can trigger team notifications

**`pre-commit`** - Quality gates (blocking)
- Enforces SDK usage (blocks direct `.wipnote/` edits)
- Runs code quality checks (ruff, mypy)
- Can be bypassed with `git commit --no-verify`

### 2. Event Log Schema

All Git hooks write to the same append-only JSONL event stream:

```python
@dataclass(frozen=True)
class EventRecord:
    event_id: str              # Unique ID (e.g., "git-commit-abc123-feature-xyz")
    timestamp: datetime        # When the event occurred
    session_id: str            # Session this event belongs to
    agent: str                 # Agent name (claude, codex, git, etc.)
    tool: str                  # Tool name (GitCommit, GitCheckout, etc.)
    summary: str               # Human-readable summary
    success: bool              # Whether operation succeeded
    feature_id: str | None     # Feature this event relates to
    start_commit: str | None   # Session's starting commit
    continued_from: str | None # Previous session ID (for continuity)
    work_type: str | None      # Type of work (feature, bug, spike, etc.)
    session_status: str | None # Session status (active, ended, etc.)
    file_paths: list[str]      # Files affected
    payload: dict              # Event-specific data
```

#### Event Types

**GitCommit**
```json
{
  "type": "GitCommit",
  "commit_hash": "abc123def456...",
  "commit_hash_short": "abc123",
  "parents": ["parent1", "parent2"],
  "is_merge": false,
  "branch": "main",
  "author_name": "Alice Developer",
  "author_email": "alice@example.com",
  "commit_message": "feat: add user authentication",
  "subject": "feat: add user authentication",
  "files_changed": ["src/auth/login.py", "tests/test_auth.py"],
  "insertions": 145,
  "deletions": 23,
  "features": ["feature-20251220-auth"]
}
```

**GitCheckout**
```json
{
  "type": "GitCheckout",
  "old_head": "abc123...",
  "new_head": "def456...",
  "flag": 1,
  "reflog_action": "checkout: moving from main to feature/auth",
  "from_branch": "main",
  "to_branch": "feature/auth"
}
```

**GitMerge**
```json
{
  "type": "GitMerge",
  "squash": false,
  "orig_head": "abc123...",
  "new_head": "def456...",
  "reflog_action": "merge feature/auth: Fast-forward"
}
```

**GitPush**
```json
{
  "type": "GitPush",
  "remote_name": "origin",
  "remote_url": "git@github.com:user/repo.git",
  "updates": [
    {
      "local_ref": "refs/heads/main",
      "local_sha": "abc123...",
      "remote_ref": "refs/heads/main",
      "remote_sha": "def456..."
    }
  ]
}
```

### 3. Session Continuity via Git Commits

Sessions are linked through Git commits, creating a continuity spine that survives session boundaries:

```
Session S1 (Claude)          Session S2 (Codex)         Session S3 (Claude)
─────────────────────       ─────────────────────      ─────────────────────
start_commit: abc1          start_commit: abc3         start_commit: abc5
continued_from: None        continued_from: S1         continued_from: S2

Events:                     Events:                    Events:
  - Edit file               - Edit file                - Edit file
  - GitCommit abc1          - GitCommit abc3           - GitCommit abc5
  - Edit file               - GitCommit abc4           - GitCommit abc6
  - GitCommit abc2

end_commit: abc2            end_commit: abc4           end_commit: abc6
───────────────────────────────────────────────────────────────────────────
                                Git Commit Graph:
                                abc1 → abc2 → abc3 → abc4 → abc5 → abc6
                                 │             │             │
                                S1            S2            S3
```

**Key Continuity Mechanisms**:

1. **start_commit** - Git commit hash when session started
2. **continued_from** - Previous session ID (when continuing work)
3. **Commit graph analysis** - Walk commit history to find related sessions
4. **Feature attribution** - Link sessions via shared features

### 4. Agent-Agnostic Design

The architecture works with ANY agent because it relies on universal primitives:

| Primitive | Why Universal | Example |
|-----------|---------------|---------|
| Git commits | Every agent that saves code creates commits | Claude, Codex, Cursor, vim + git |
| File changes | All agents modify files | Any text editor |
| Commit messages | Standard Git feature | Any Git client |
| Branch operations | Core Git workflow | Any Git tool |

**Agent Compatibility Matrix**:

| Agent | Git Hooks | File Watching | Session Tracking | Notes |
|-------|-----------|---------------|------------------|-------|
| Claude Code | ✅ | ✅ | ✅ | Full integration via plugin |
| GitHub Codex | ✅ | ✅ | ✅ | Git hooks + filesystem watcher |
| Google Gemini | ✅ | ✅ | ✅ | Git hooks + filesystem watcher |
| Cursor | ✅ | ✅ | ✅ | Git hooks + filesystem watcher |
| vim/emacs + git | ✅ | ✅ | ⚠️ | Requires manual session start |
| Any CLI tool | ✅ | ❌ | ❌ | Commits tracked, but no fine-grained events |

## Continuity Reconstruction

### How Sessions Link Across Agents

Wipnote reconstructs session continuity using multiple signals:

**1. Explicit Continuation**
```python
# Session S2 explicitly continues S1
session_s2 = manager.start_session(
    agent="codex",
    continued_from="session-s1"
)
# S2.continued_from = "session-s1"
# S2.start_commit = get_current_commit()
```

**2. Commit Graph Analysis**
```python
# Find sessions between two commits
def find_sessions_between(commit_a: str, commit_b: str) -> list[Session]:
    # Walk git log from commit_a to commit_b
    commits = git_log_between(commit_a, commit_b)

    # Find events with these commits
    events = event_log.query(
        tool="GitCommit",
        commit_hash__in=commits
    )

    # Extract unique session IDs
    session_ids = {event.session_id for event in events}
    return [get_session(sid) for sid in session_ids]
```

**3. Feature-Based Linking**
```python
# Find all sessions that worked on a feature
def find_feature_sessions(feature_id: str) -> list[Session]:
    events = event_log.query(feature_id=feature_id)
    session_ids = {event.session_id for event in events}
    return sorted([get_session(sid) for sid in session_ids],
                  key=lambda s: s.created)
```

**4. Time-Based Proximity**
```python
# Find sessions within time window
def find_proximate_sessions(
    reference_time: datetime,
    window_minutes: int = 60
) -> list[Session]:
    start = reference_time - timedelta(minutes=window_minutes)
    end = reference_time + timedelta(minutes=window_minutes)
    return session_converter.query(
        created__gte=start,
        created__lte=end
    )
```

### Example: Cross-Agent Continuity

**Scenario**: Work starts in Claude, continues in Codex, finishes in Claude

```
Day 1, 10am (Claude):
  session-s1-abc = manager.start_session(agent="claude")
  # User works...
  git commit -m "feat: start auth (feature-auth-001)"  → abc123
  manager.end_session("session-s1-abc")

Day 1, 2pm (Codex):
  session-s2-def = manager.start_session(
      agent="codex",
      continued_from="session-s1-abc"  # Optional but helpful
  )
  # User works...
  git commit -m "feat: continue auth (feature-auth-001)"  → def456
  manager.end_session("session-s2-def")

Day 2, 9am (Claude):
  session-s3-ghi = manager.start_session(agent="claude")
  # Wipnote automatically detects continuation via:
  #   1. Same feature (feature-auth-001)
  #   2. Commit graph (abc123 → def456 → current)
  #   3. Session summary handoff notes
  git commit -m "feat: finish auth (feature-auth-001)"  → ghi789
```

**Query for full history**:
```python
# Get all sessions for feature-auth-001
sessions = sdk.get_feature_sessions("feature-auth-001")

# Result:
# [
#   Session(id="session-s1-abc", agent="claude", ...),
#   Session(id="session-s2-def", agent="codex", ...),
#   Session(id="session-s3-ghi", agent="claude", ...)
# ]

# Get commit history
commits = git_log("feature-auth-001")
# [abc123, def456, ghi789]

# Link sessions via commits
for session in sessions:
    commits_in_session = event_log.query(
        session_id=session.id,
        tool="GitCommit"
    )
    print(f"{session.agent}: {[e.payload['commit_hash_short']
                               for e in commits_in_session]}")

# Output:
# claude: ['abc123']
# codex: ['def456']
# claude: ['ghi789']
```

## Event Attribution

### How Events Link to Features

Wipnote uses multiple strategies to attribute events to features:

**1. Active Features (Session State)**
```python
# Features marked as in-progress
active_features = manager.get_active_features()
# → ['feature-auth-001', 'feature-db-002']
```

**2. Commit Message Parsing**
```python
def parse_feature_refs(message: str) -> list[str]:
    """
    Extract feature references from commit message.

    Patterns:
    - Implements: feature-xyz
    - Fixes: bug-abc
    - feature-xyz (anywhere in message)
    """
    features = []

    # Explicit references
    pattern1 = r"(?:Implements|Fixes|Closes|Refs):\s*(feature-[\w-]+|bug-[\w-]+)"
    features.extend(re.findall(pattern1, message, re.IGNORECASE))

    # Mentions anywhere
    pattern2 = r"\b(feature-[\w-]+|bug-[\w-]+)\b"
    features.extend(re.findall(pattern2, message, re.IGNORECASE))

    return list(set(features))  # Remove duplicates

# Example:
message = "feat: add login endpoint (feature-auth-001)"
features = parse_feature_refs(message)
# → ['feature-auth-001']
```

**3. File Pattern Matching**
```python
# Features can specify file patterns
feature.file_patterns = [
    "src/auth/**/*.py",
    "tests/auth/**/*.py"
]

# When commit changes files, match against patterns
changed_files = ["src/auth/login.py", "tests/auth/test_login.py"]
matched_features = []
for feature in all_features:
    if any(fnmatch.fnmatch(f, pattern)
           for f in changed_files
           for pattern in feature.file_patterns):
        matched_features.append(feature.id)

# → ['feature-auth-001']
```

**4. Combined Attribution**
```python
def attribute_commit(commit_hash: str) -> list[str]:
    """Combine all attribution strategies."""
    commit_info = git_show(commit_hash)

    features = []

    # 1. Active features at commit time
    features.extend(get_active_features_at_commit(commit_hash))

    # 2. Mentioned in commit message
    features.extend(parse_feature_refs(commit_info.message))

    # 3. File pattern matching
    features.extend(match_files_to_features(commit_info.files))

    # Remove duplicates, preserve order
    return list(dict.fromkeys(features))
```

## Implementation Details

### Git Hook Installation

**Location**: `.wipnote/hooks/` (versioned) → `.git/hooks/` (symlinked)

```bash
# Install hooks
wipnote install-hooks

# Creates:
.git/hooks/post-commit    → .wipnote/hooks/post-commit.sh
.git/hooks/post-checkout  → .wipnote/hooks/post-checkout.sh
.git/hooks/post-merge     → .wipnote/hooks/post-merge.sh
.git/hooks/pre-push       → .wipnote/hooks/pre-push.sh
.git/hooks/pre-commit     → .wipnote/hooks/pre-commit.sh
```

**Hook Script Structure**:
```bash
#!/bin/bash
# .wipnote/hooks/post-commit.sh

# Exit early if disabled
if [ "$(git config --bool wipnote.hooks)" = "false" ]; then
    exit 0
fi

# Call Wipnote event logger
uv run python -m wipnote.git_events commit

# Always succeed (non-blocking)
exit 0
```

### Event Log Storage

**Directory Structure**:
```
.wipnote/
├── events/
│   ├── session-s1-abc.jsonl        # Events for session S1
│   ├── session-s2-def.jsonl        # Events for session S2
│   └── git.jsonl                   # Events from Git (no active session)
```

**Append-Only Design**:
- Each session gets its own JSONL file
- Events append to the file (never update existing lines)
- Deduplication: Check last 250 lines for duplicate event_id
- Git-friendly: Text diffs work well

**Querying Events**:
```python
from wipnote.event_log import JsonlEventLog

log = JsonlEventLog(".wipnote/events")

# Get all events for a session
events = log.get_session_events("session-s1-abc")

# Iterate all events across all sessions
for path, event in log.iter_events():
    if event["tool"] == "GitCommit":
        print(f"Commit: {event['payload']['commit_hash_short']}")
```

### Session Manager Integration

**Determining Context for Git Events**:
```python
def _determine_context(graph_dir: Path, commit_message: str | None = None) -> dict:
    """
    Determine best-effort session + feature context for Git events.

    Returns:
        session_id, agent, start_commit, continued_from, session_status,
        active_features, primary_feature_id, message_features, all_features
    """
    active_features = get_active_features(graph_dir)
    primary_feature_id = get_primary_feature_id(graph_dir)
    message_features = parse_feature_refs(commit_message or "")

    # Combine all feature references
    all_features = list(dict.fromkeys(
        active_features + message_features
    ))

    # Get active session if exists
    session = get_active_session(graph_dir)
    if session:
        return {
            "session_id": session.id,
            "agent": session.agent,
            "start_commit": session.start_commit,
            "continued_from": session.continued_from,
            "session_status": session.status,
            "active_features": active_features,
            "primary_feature_id": primary_feature_id,
            "message_features": message_features,
            "all_features": all_features,
        }

    # No active session: use stable pseudo-session "git"
    return {
        "session_id": "git",
        "agent": "git",
        "start_commit": None,
        "continued_from": None,
        "session_status": None,
        "active_features": active_features,
        "primary_feature_id": primary_feature_id,
        "message_features": message_features,
        "all_features": all_features,
    }
```

### Cross-Session Analytics

**Commit Graph Queries**:
```python
class CommitGraphAnalytics:
    """Analytics across sessions using Git commit graph."""

    def get_feature_timeline(self, feature_id: str) -> list[dict]:
        """Get chronological timeline of work on a feature."""
        events = event_log.query(feature_id=feature_id, tool="GitCommit")

        timeline = []
        for event in sorted(events, key=lambda e: e.timestamp):
            timeline.append({
                "timestamp": event.timestamp,
                "commit": event.payload["commit_hash_short"],
                "agent": event.agent,
                "session_id": event.session_id,
                "message": event.payload["subject"],
                "files_changed": len(event.file_paths),
                "insertions": event.payload["insertions"],
                "deletions": event.payload["deletions"],
            })

        return timeline

    def get_session_chain(self, session_id: str) -> list[Session]:
        """Get chain of sessions (backwards via continued_from)."""
        chain = []
        current = get_session(session_id)

        while current:
            chain.append(current)
            if current.continued_from:
                current = get_session(current.continued_from)
            else:
                break

        return list(reversed(chain))  # Oldest first

    def get_commit_attribution(self, commit_hash: str) -> dict:
        """Get full attribution for a commit."""
        events = event_log.query(
            tool="GitCommit",
            payload__commit_hash=commit_hash
        )

        if not events:
            return {"commit": commit_hash, "sessions": [], "features": []}

        return {
            "commit": commit_hash,
            "sessions": list(set(e.session_id for e in events)),
            "features": list(set(e.feature_id for e in events if e.feature_id)),
            "agents": list(set(e.agent for e in events)),
            "timestamp": events[0].timestamp,
        }
```

## Benefits of Git-Based Continuity

### 1. Agent Agnostic
- Works with ANY coding agent (Claude, Codex, Cursor, vim)
- No platform-specific integrations required
- Git is universal across all development tools

### 2. Survives Session Boundaries
- Commits link sessions even after process dies
- Work can be picked up by any agent
- Full history reconstruction possible

### 3. Team Collaboration
- Multiple developers/agents can work simultaneously
- Merge events track integration points
- Push events mark team boundaries

### 4. Offline-First
- All tracking works offline (Git is local)
- No network dependencies
- Sync naturally via git push/pull

### 5. Version Control Native
- Event log is Git-friendly (text files)
- Diffs show what changed
- Branches work naturally
- Merge conflicts are visible

### 6. Simple and Robust
- Minimal dependencies (just Git)
- Append-only design (no corruption)
- Fail-safe (hooks never block Git operations)
- Easy to debug (just read JSONL files)

## Performance Characteristics

### Git Hook Overhead

| Hook | Type | Typical Latency | Impact |
|------|------|-----------------|--------|
| post-commit | Non-blocking | <50ms | None (async) |
| post-checkout | Non-blocking | <30ms | None (async) |
| post-merge | Non-blocking | <30ms | None (async) |
| pre-push | Non-blocking | <100ms | None (async) |
| pre-commit | Blocking | <100ms | Minimal (fast checks) |

**Optimization Strategies**:
- Hooks run in background (daemon process)
- Deduplication prevents redundant events
- Batch writes to reduce I/O
- Tail-only deduplication (last 250 lines)

### Event Log Scalability

| Metric | Value | Notes |
|--------|-------|-------|
| Events per session | ~500-1000 | Typical development session |
| Session file size | ~500KB | Uncompressed JSONL |
| Query time (1000 events) | <10ms | Sequential scan |
| Deduplication check | <5ms | Tail-only (last 64KB) |

**Scaling Strategies**:
- Partition by session (natural sharding)
- Optional SQLite index for complex queries
- Compression for old events
- TTL/rotation policy (configurable)

## Security Considerations

### Sensitive Data in Commit Messages

**Risk**: Developers may include secrets in commit messages

**Mitigation**:
- Pre-commit hooks can scan for patterns (API keys, passwords)
- Event payloads stored locally only (not pushed by default)
- `.wipnote/events/` should be in `.gitignore`

### Multi-Tenancy

**Risk**: Multiple projects in one repository

**Solution**:
- Wipnote per project (`.wipnote/` at project root)
- Hooks respect working directory
- No cross-project leakage

### Git Hook Tampering

**Risk**: Malicious users disable hooks

**Detection**:
- Hook installation status command: `wipnote install-hooks --list`
- Periodic verification in CI/CD
- Team policy enforcement

## Comparison: Git Spine vs Plugin Hooks

| Aspect | Plugin Hooks (Old) | Git Spine (New) |
|--------|-------------------|-----------------|
| **Agent Support** | Claude Code only | Any agent |
| **Continuity** | Session-based only | Commit-based + session-based |
| **Offline** | Requires plugin runtime | Works offline |
| **Team Collab** | Single agent | Multi-agent |
| **Setup** | Install plugin | Install git hooks |
| **Maintenance** | Plugin updates | Git hook updates |
| **Robustness** | Plugin crashes = data loss | Git always works |
| **Debugging** | Plugin logs (opaque) | JSONL logs (transparent) |

**Migration Path**: Both systems can run in parallel. Git spine becomes primary, plugin hooks remain as fallback for rich context.

## Future Enhancements

### 1. Automatic Session Continuation
Detect likely continuation when starting a new session:
```python
def auto_detect_continuation() -> str | None:
    """Detect if current session is continuing previous work."""
    last_session = get_most_recent_session()
    if not last_session:
        return None

    # Check if within time window (e.g., 4 hours)
    if datetime.now() - last_session.ended > timedelta(hours=4):
        return None

    # Check if on same branch
    if git_current_branch() != last_session.branch:
        return None

    # Check if same feature is active
    active_features = get_active_features()
    if not any(f.id in active_features for f in last_session.worked_on):
        return None

    return last_session.id  # Continue from this session
```

### 2. Merge Conflict Resolution Tracking
Track when conflicts occur and how they're resolved:
```python
{
  "type": "GitMergeConflict",
  "files_conflicted": ["src/file.py"],
  "resolution_time_seconds": 180,
  "resolution_strategy": "manual"
}
```

### 3. Smart Feature Recommendation
Suggest which feature to work on based on commit history:
```python
def recommend_next_feature(agent: str) -> str | None:
    """Recommend next feature based on recent commits."""
    recent_commits = git_log(limit=10)
    feature_mentions = Counter()

    for commit in recent_commits:
        features = parse_feature_refs(commit.message)
        feature_mentions.update(features)

    # Return most frequently mentioned feature
    return feature_mentions.most_common(1)[0][0] if feature_mentions else None
```

### 4. Cross-Repository Analytics
Track work across multiple repositories:
```python
class MultiRepoAnalytics:
    def get_agent_activity_across_repos(self, agent: str, days: int = 7):
        """Get agent activity across all repositories."""
        repos = discover_wipnote_repos()

        all_activity = []
        for repo in repos:
            events = repo.event_log.query(
                agent=agent,
                timestamp__gte=datetime.now() - timedelta(days=days)
            )
            all_activity.extend(events)

        return sorted(all_activity, key=lambda e: e.timestamp)
```

## References

- [Git Hooks Documentation](./GIT_HOOKS.md) - Installation and configuration
- [Event Log Reference](./EVENT_LOG.md) - Event schema and querying
- [Session Management](./SESSION_MANAGEMENT.md) - Session lifecycle
- [Migration Guide](./MIGRATION_GUIDE.md) - Migrating from old tracking

---

*Last updated: 2025-01-02*
