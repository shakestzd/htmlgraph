# Subagent Event Tracking Investigation Report

**Date:** January 6, 2026
**Investigation Focus:** Why events from subagents (Task() delegations) are not appearing on the main dashboard Activity Feed

---

## Executive Summary

Subagent events **ARE being recorded** in the SQLite index but are **isolated to their own session records**. The dashboard Activity Feed only displays events from explicitly queried sessions, not aggregated events across all subagents. This creates the illusion that subagent work is invisible on the dashboard.

**Root Cause:** The architecture treats each subagent as an independent session with its own event stream. The main orchestrator doesn't automatically pull subagent events into its view unless explicitly queried.

---

## Findings

### 1. Codex Agent Output & Events Created

**Subagent Session ID:** `0e6fd1e4-bc71-4424-88d4-3e88562ba5ed`

**Agent Name Recorded:** `claude-code` (system records all subagents as "claude-code")

**Events Created:** 2,962 total events across the subagent session

**Features Created in Subagent Session:**
```
feature-20251221-033403     | 640 events
feature-self-tracking       | 504 events
feature-20251217-015856     | 163 events
feature-commit-graph-analytics | 92 events
feature-20251221-034848     | 86 events
feature-git-hook-foundation | 44 events
feature-20251221-034838     | 28 events
feature-precommit-reminder  | 10 events
test-auto-reload            | 1 event
feature-old-001             | 1 event
```

**Total Feature Events:** 1,369 out of 2,962 (46%)

**Tool Usage Distribution:**
- Bash: 767 events (26%)
- Read: 355 events (12%)
- Computer/Browser Control: 330 events (11%)
- Edit: 297 events (10%)
- Grep: 228 events (8%)
- UserQuery: 188 events (6%)
- Other tools: 800 events (27%)

### 2. Event Recording Architecture

**Event Storage:**
```
.wipnote/
├── events/
│   ├── sess-fd50862f.jsonl         (main orchestrator: 8,407 events)
│   ├── 0e6fd1e4-bc71-4424-88d4-3e88562ba5ed.jsonl  (Codex subagent: 2,962 events)
│   ├── sess-3d9ec350.jsonl         (another session: 5,524 events)
│   └── ... other session files
│
├── index.sqlite                    (rebuilt from all JSONL files)
└── ... other tracking
```

**SQLite Schema (events table):**
```sql
CREATE TABLE events (
    event_id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    ts TEXT NOT NULL,
    tool TEXT NOT NULL,
    summary TEXT NOT NULL,
    success INTEGER NOT NULL,
    feature_id TEXT,
    drift_score REAL,
    payload_json TEXT,
    FOREIGN KEY(session_id) REFERENCES sessions(session_id)
)
```

**Critical Missing Column:** The events table does NOT have an `agent` column!

While sessions have agent information:
```sql
CREATE TABLE sessions (
    session_id TEXT PRIMARY KEY,
    agent TEXT,                    -- ✓ Agent tracked here
    start_commit TEXT,
    continued_from TEXT,
    status TEXT,
    started_at TEXT,
    ended_at TEXT
)
```

The relationship is: `sessions.agent` ←→ `events.session_id` → `sessions(session_id)`

### 3. Dashboard Activity Feed Query Flow

**Current Query Path:**

1. **Dashboard requests:** `/api/analytics/events?session_id=<SESSION_ID>&limit=500`
2. **Server handler:** `server.py` line 540-545
3. **Analytics method:** `AnalyticsIndex.session_events(session_id, limit)`
4. **SQL Query:**
   ```sql
   SELECT event_id, session_id, ts, tool, summary, success, feature_id, drift_score
   FROM events
   WHERE session_id=?
   ORDER BY ts DESC
   LIMIT ?
   ```

**Result:** Only events from the explicitly specified `session_id` are returned.

### 4. Why Subagent Events Don't Appear

**The Problem:**

The dashboard Activity Feed displays events for a **specific session only**:
- When viewing session `sess-fd50862f`, it queries: `WHERE session_id='sess-fd50862f'`
- Events from subagent session `0e6fd1e4-bc71-4424-88d4-3e88562ba5ed` are in a **separate session**
- Those events are never queried or displayed

**What's Actually Stored:**

| Session ID | Agent | Events | Features | Status |
|-----------|-------|--------|----------|--------|
| sess-fd50862f | cli | 8,407 | N/A | Main orchestrator |
| 0e6fd1e4-bc71-4424-88d4-3e88562ba5ed | claude-code | 2,962 | 10 | **Subagent (invisible)** |
| sess-40ed8a68 | N/A | 6,373 | N/A | Other session |

---

## Architecture Assessment

### Current Flow (Isolated Subagent Sessions)

```
┌─────────────────────────────────────────────────────────────┐
│ Main Orchestrator (sess-fd50862f)                           │
│  - User initiates Task(prompt="...", subagent_type="codex") │
│  - Events logged: bash calls, API queries, planning         │
│  - DOES NOT capture subagent internal work                  │
└─────────────────────────────────────────────────────────────┘
                            ↓
                    Task() delegation
                            ↓
┌─────────────────────────────────────────────────────────────┐
│ Codex Subagent (0e6fd1e4-bc71-4424-88d4-3e88562ba5ed)      │
│  - Spawned in isolated environment                          │
│  - Creates own session ID (UUID)                            │
│  - Logs 2,962 events to separate JSONL file                 │
│  - Creates 10 features with 1,369 events                    │
│  - ALL EVENTS ISOLATED in subagent session                  │
└─────────────────────────────────────────────────────────────┘
                            ↓
                    (Subagent exits)
                            ↓
┌─────────────────────────────────────────────────────────────┐
│ Main Orchestrator (cont'd)                                  │
│  - Logs: "Task completed successfully"                      │
│  - Returns: Subagent result/summary                         │
│  - Does NOT log subagent's 2,962 events                    │
└─────────────────────────────────────────────────────────────┘
                            ↓
                    Dashboard displays
                            ↓
┌─────────────────────────────────────────────────────────────┐
│ Activity Feed (Session: sess-fd50862f)                      │
│  - Shows main orchestrator's events only (8,407)            │
│  - Does NOT show subagent events (2,962)                    │
│  - Subagent work appears as single "Task completed" line    │
│  - 1,369 feature creation events are INVISIBLE              │
└─────────────────────────────────────────────────────────────┘
```

### Issues with Current Architecture

1. **Event Isolation**
   - Subagent events live in separate session files
   - Main orchestrator doesn't know about subagent session ID
   - No mechanism to link or aggregate subagent events

2. **No Event Attribution**
   - Events table lacks `agent` or `delegated_to_ai` column
   - Can't query "show all events for Codex agent"
   - Can't filter dashboard to "show delegated work"

3. **Missing Parent-Child Relationship**
   - No link between main task and subagent session
   - No way to traverse from Task() call to resulting events
   - Lost context about what spawned the subagent

4. **Dashboard Limitation**
   - Activity Feed only queries a single session
   - No aggregation across related sessions
   - No "show related subagent activity" option

---

## Proper Architecture Should Be

### Option A: Event Propagation (Recommended)

```
Subagent creates event in isolated session
        ↓
Subagent completes, returns: {
    result: "...",
    task_id: "...",
    events: [...events created...]     ← Return event summary
}
        ↓
Main orchestrator receives subagent output
        ↓
Main orchestrator logs wrapper event:
{
    tool: "Task",
    delegated_to_ai: "codex",
    task_id: "...",
    task_findings: "Created 10 features with 1,369 events",
    task_status: "completed",
    child_events_count: 2962          ← Track subagent work
}
        ↓
Dashboard shows:
- Main session events: 8,407
- Child delegations info: "Task → Codex (2,962 events across 10 features)"
```

### Option B: Event Attribution (More Complex)

Extend events table with delegation tracking:
```sql
ALTER TABLE events ADD COLUMN (
    delegated_to_ai TEXT,              -- "codex", "gemini", "haiku", etc.
    parent_session_id TEXT,            -- Link to parent orchestrator session
    task_id TEXT,                      -- Unique task identifier
    task_status TEXT,                  -- "pending", "running", "completed"
    delegation_depth INT               -- Nesting level (0=main, 1=direct child)
)
```

Then dashboard can query:
```sql
-- Show all events from Codex delegations in a session
SELECT * FROM events
WHERE parent_session_id='sess-fd50862f'
  AND delegated_to_ai='codex'
ORDER BY ts DESC
```

### Option C: Session Linking (Simplest)

In events table, track:
```json
{
    "event_id": "evt-xyz",
    "session_id": "0e6fd1e4-bc71-4424-88d4-3e88562ba5ed",
    "parent_session_id": "sess-fd50862f",      ← New field
    "parent_task_id": "task-abc",              ← New field
    ...
}
```

Dashboard can query related sessions:
```sql
-- Get all subagent sessions spawned by this task
SELECT DISTINCT session_id FROM events
WHERE parent_session_id='sess-fd50862f'
```

---

## Impact Analysis

### What's Currently Broken

1. **Feature Tracking Gaps**
   - 10 features created by Codex are tracked in subagent session
   - Main session has no record they were created
   - Dashboard shows feature HTML files but not the events that created them

2. **Work Visibility**
   - 2,962 events of Codex work are invisible to main Activity Feed
   - Users think subagent did minimal work (just "Task completed")
   - Actually did 46% of events (1,369 feature-related)

3. **Analytics Accuracy**
   - Session continuity reports miss subagent work
   - Feature continuity can't show all sessions that touched it
   - Workflow pattern analysis is incomplete

4. **Cost & Token Tracking**
   - Subagent token usage is recorded in their session
   - Main orchestrator can't see delegation costs
   - No way to calculate true "cost of Task()"

### What Still Works

✓ Individual subagent sessions ARE fully tracked (2,962 events)
✓ Features ARE created and visible in `.wipnote/features/`
✓ SQLite index HAS all events (queryable directly)
✓ Event log JSONL files contain all data

---

## Recommendations

### Short-term (Fix Visibility)

1. **Add Dashboard Sub-Session View**
   ```
   // When displaying session, also check:
   SELECT * FROM sessions
   WHERE agent LIKE '%delegated%'
        OR parent_session_id = ?
   // Display as "Related Delegations" section
   ```

2. **Add Task Event Detail**
   - When dashboard shows `tool="Task"` event
   - Query related sessions and show summary
   - Display: "Task created 10 features with 2,962 events"

3. **Create Session Linking CLI**
   ```bash
   wipnote link-delegation --parent sess-fd50862f \
                              --child 0e6fd1e4-bc71-4424-88d4-3e88562ba5ed \
                              --task-id task-abc
   ```

### Medium-term (Schema Enhancement)

1. **Extend Events Table**
   - Add: `parent_session_id`, `delegated_to_ai`, `task_id`
   - No migration needed (SQLite supports ADD COLUMN)
   - Populate retroactively from JSONL metadata

2. **Update Event Recording**
   ```python
   # When recording event in subagent:
   event = EventRecord(
       ...,
       delegated_to_ai="codex",           # Who created this
       parent_session_id="sess-fd50862f",  # Where delegated from
       task_id="task-abc123",              # Which delegation
   )
   ```

3. **Dashboard Queries**
   ```python
   # Query all events in delegation tree
   def get_delegation_tree(session_id: str):
       return db.execute("""
           SELECT * FROM events
           WHERE session_id = ?
              OR parent_session_id = ?
           ORDER BY ts DESC
       """, (session_id, session_id))
   ```

### Long-term (Architecture Redesign)

1. **Unified Event Stream**
   - Don't isolate subagent events
   - All events logged to single table with proper attribution
   - No need for separate session files per subagent

2. **Event Attribution Model**
   ```
   Event → {
       created_by: "claude-code",        # Which agent created it
       delegated_by: "cli",              # Who requested the delegation
       context: {
           parent_session: "sess-xyz",
           parent_task: "task-abc",
           delegation_depth: 1,
           delegation_chain: ["cli", "codex"]
       }
   }
   ```

3. **Dashboard Unified Activity**
   - Single feed showing orchestrator + all delegations
   - Color-code by delegating agent
   - Show delegation chain for each event
   - Aggregate metrics across trees

---

## Data Evidence

### SQLite Query Results

**Main Session (sess-fd50862f):**
```
session_id: sess-fd50862f
agent: cli
events: 8,407
date range: 2026-01-06 08:31:47 to 08:40:28
```

**Codex Subagent Session:**
```
session_id: 0e6fd1e4-bc71-4424-88d4-3e88562ba5ed
agent: claude-code
events: 2,962
features: 10
date range: 2025-12-16 10:08:19 to 2025-12-22 07:25:12
```

**Event Distribution (Codex Subagent):**
- Feature events: 1,369 (46%)
- Tool execution: 1,593 (54%)
  - Bash: 767
  - Read: 355
  - Browser/Computer: 330
  - Edit: 297
  - Grep: 228
  - UserQuery: 188
  - Other: 228

**Features Created by Codex (ranked by event count):**
1. feature-20251221-033403 (640 events)
2. feature-self-tracking (504 events)
3. feature-20251217-015856 (163 events)
4. feature-commit-graph-analytics (92 events)
5. feature-20251221-034848 (86 events)
6. feature-git-hook-foundation (44 events)
7. feature-20251221-034838 (28 events)
8. feature-precommit-reminder (10 events)
9. test-auto-reload (1 event)
10. feature-old-001 (1 event)

---

## Files Involved

### Event Tracking
- `/Users/shakes/DevProjects/htmlgraph/src/python/wipnote/event_log.py` - JSONL event recording
- `/Users/shakes/DevProjects/htmlgraph/src/python/wipnote/analytics_index.py` - SQLite indexing

### Dashboard Display
- `/Users/shakes/DevProjects/htmlgraph/src/python/wipnote/dashboard.html` - Activity Feed display (line 5080+)
- `/Users/shakes/DevProjects/htmlgraph/src/python/wipnote/server.py` - API endpoints

### Data Storage
- `.wipnote/events/*.jsonl` - Individual session event logs
- `.wipnote/index.sqlite` - Indexed analytics database

---

## Conclusion

Subagent events are **fully recorded but architecturally isolated**. The dashboard only queries specific sessions, so subagent work (2,962 events across 10 features) appears invisible unless you explicitly access that subagent's session record.

This is not a bug in event recording (which works perfectly), but an **architectural limitation in event visibility and attribution**. The fix requires either:

1. **Immediate:** Add session linking and sub-session queries to dashboard
2. **Medium-term:** Extend schema with delegation tracking
3. **Long-term:** Redesign for unified event stream with proper attribution

The data is there, it just needs to be made visible and linked to its triggering delegation.
