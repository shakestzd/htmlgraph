# Subagent Event Visibility - Action Items

**Issue:** Dashboard Activity Feed doesn't show events from Task() delegations (subagents)

**Impact:** 2,962 events (26% of work) from Codex subagent are invisible, including 1,369 feature creation events

**Status:** Events ARE recorded but architecturally isolated in separate sessions

---

## Quick Facts

| Item | Value |
|------|-------|
| Main Session | sess-fd50862f |
| Subagent Session | 0e6fd1e4-bc71-4424-88d4-3e88562ba5ed |
| Main Events | 8,407 |
| Subagent Events | 2,962 (invisible) |
| Features Created by Subagent | 10 |
| Feature Creation Events | 1,369 (46% of subagent work) |
| Database | .wipnote/index.sqlite |
| Event Logs | .wipnote/events/*.jsonl |

---

## Root Cause (One Sentence)

The dashboard queries a single session via `WHERE session_id=?`, so subagent events in a different session_id are never fetched.

---

## Short-term Fix (1-2 hours)

### Step 1: Add Task Event Detail to Dashboard

**File:** `src/python/wipnote/dashboard.html`

**What:** When displaying a `tool="Task"` event, show which session it delegated to

**How:**
```javascript
// In fetchActivityLog, when rendering Task events:
if (tool === "Task") {
    // Extract delegated_to_ai and task_id from payload
    const delegatedTo = item.dataset.delegatedTo || "unknown";
    const taskId = item.dataset.taskId;

    // Display: "Task → codex (task-abc)"
    html_content += `<span class="badge agent-${delegatedTo}">→ ${delegatedTo}</span>`;
}
```

**Effort:** 20 minutes

---

### Step 2: Create Session Lookup Helper

**File:** `src/python/wipnote/server.py`

**What:** Add API endpoint to find subagent session by parent task_id

**Endpoint:**
```python
GET /api/analytics/task/<task_id>/session
```

**Response:**
```json
{
    "task_id": "task-abc123",
    "parent_session": "sess-fd50862f",
    "subagent_session": "0e6fd1e4-bc71-4424-88d4-3e88562ba5ed",
    "delegated_to": "codex",
    "events_count": 2962,
    "features_created": 10
}
```

**Effort:** 30 minutes

---

### Step 3: Add "Related Sessions" Dashboard View

**File:** `src/python/wipnote/dashboard.html`

**What:** When viewing a session with Task events, show related subagent sessions

**Display:**
```
Activity Log (8,407 events)
├─ 2026-01-06 08:40:28  ✅ Bash
├─ 2026-01-06 08:32:19  ✅ Task → codex (task-abc)
│  └─ Related Session: 0e6fd1e4-bc71-4424-88d4-3e88562ba5ed
│     └─ Subagent Events: 2,962
│        └─ Features Created: 10
│        └─ View Details ↗
└─ ... more events ...
```

**Effort:** 1 hour

---

## Medium-term Fix (3-5 hours)

### Step 1: Extend Events Table Schema

**File:** `src/python/wipnote/analytics_index.py`

**What:** Add delegation tracking columns (no migration needed for SQLite)

**Schema Change:**
```python
# In ensure_schema(), after CREATE TABLE events:
conn.execute("ALTER TABLE events ADD COLUMN parent_session_id TEXT;")
conn.execute("ALTER TABLE events ADD COLUMN delegated_to_ai TEXT;")
conn.execute("ALTER TABLE events ADD COLUMN task_id TEXT;")
conn.execute("ALTER TABLE events ADD COLUMN task_status TEXT;")
```

**Notes:**
- SQLite allows ADD COLUMN with no downtime
- Existing events get NULL for new columns
- Fill in retroactively from JSONL metadata

**Effort:** 30 minutes

---

### Step 2: Update Event Recording

**File:** `src/python/wipnote/event_log.py`

**What:** Populate new delegation fields when recording events in subagent context

**Changes:**
```python
# In JsonlEventLog.append() or where EventRecord is created:
class EventRecord:
    # ... existing fields ...
    parent_session_id: str | None = None      # Link to parent
    delegated_to_ai: str | None = None        # Which agent was delegated
    task_id: str | None = None                # Which task spawned this
    task_status: str | None = None            # Task completion status
```

**Effort:** 20 minutes

---

### Step 3: Add Delegation Queries to Analytics

**File:** `src/python/wipnote/analytics_index.py`

**What:** New query methods for delegation trees

**Methods:**
```python
def get_delegations(self, session_id: str) -> list[dict]:
    """Get all delegations spawned by a session"""
    return conn.execute("""
        SELECT DISTINCT parent_session_id, delegated_to_ai, task_id
        FROM events
        WHERE parent_session_id = ?
    """, (session_id,)).fetchall()

def get_delegation_tree(self, session_id: str) -> dict:
    """Get session + all child delegations"""
    return {
        "session": self.get_session(session_id),
        "delegations": self.get_delegations(session_id),
        "total_events": self.count_events(session_id, include_children=True)
    }
```

**Effort:** 45 minutes

---

### Step 4: Update Dashboard to Show Delegation Tree

**File:** `src/python/wipnote/dashboard.html`

**What:** Display unified view of session + all subagent delegations

**Display:**
```
Session Tree: sess-fd50862f (cli)
├─ Main Session Events: 8,407
│  └─ [Activity Feed with Task events highlighted]
│
├─ Delegations:
│  ├─ Codex (task-abc123)
│  │  └─ Session: 0e6fd1e4-bc71-4424-88d4-3e88562ba5ed
│  │  └─ Events: 2,962
│  │  └─ Features: 10
│  │  └─ Tools: Bash, Read, Edit, Grep, ...
│  │  └─ View Subagent Activity ↗
│  │
│  └─ Gemini (task-def456)
│     └─ Session: 32300148-6c0c-4d8e-b034-97143689580a
│     └─ Events: 100
│     └─ View Subagent Activity ↗
│
└─ Total: 11,369 events across 3 sessions
```

**Effort:** 1.5 hours

---

## Long-term Fix (1-2 days)

### Complete Redesign: Unified Event Stream

**Concept:** Stop isolating subagent sessions; use proper parent-child attribution

**Architecture:**
```
Single events table with delegation tracking:

events {
    event_id ..................... PK
    session_id ................... Main session (where work originated)
    delegated_to_ai .............. "codex", "gemini", "haiku", null
    delegation_chain ............. ["cli", "codex"] (breadcrumb trail)
    parent_session_id ............ Link to spawning session
    parent_task_id ............... Link to Task() event that spawned this
    ...other fields...
}

Queries become simple:
- "Show all events in session" → WHERE session_id = ?
- "Show delegations" → WHERE parent_session_id = ?
- "Show codex work" → WHERE delegated_to_ai = 'codex'
- "Show delegation tree" → Recursive query with breadcrumbs
```

**Benefits:**
1. No separate session files needed
2. Single query can show delegation tree
3. Analytics can cross session boundaries
4. Cost tracking works automatically
5. Feature continuity shows all touches

**Implementation:**
1. Redesign event recording to populate delegation fields
2. Update analytics to use breadcrumb chains
3. Dashboard becomes much simpler
4. New reports become possible

**Effort:** 2-3 days

---

## Testing Strategy

### Unit Tests

**File:** `tests/python/test_delegation_tracking.py`

```python
def test_subagent_events_queryable():
    """Verify subagent events are in database"""
    db = AnalyticsIndex(":memory:")
    # ... create subagent event ...
    events = db.query_events(
        session_id="0e6fd1e4-bc71-4424-88d4-3e88562ba5ed"
    )
    assert len(events) > 0

def test_delegation_tree_query():
    """Verify parent-child queries work"""
    delegations = db.get_delegations(session_id="sess-fd50862f")
    assert any(d["delegated_to_ai"] == "codex" for d in delegations)

def test_dashboard_shows_delegations():
    """Verify dashboard API includes delegation info"""
    response = api.get("/api/analytics/sessions/sess-fd50862f")
    assert "delegations" in response
    assert response["delegations"][0]["events"] == 2962
```

---

### Integration Tests

**File:** `tests/python/test_subagent_visibility.py`

```python
def test_codex_features_visible_in_session_tree():
    """10 features created by Codex are visible in delegation tree"""
    tree = db.get_delegation_tree("sess-fd50862f")
    codex_features = [
        e for e in tree["delegations"]
        if e["delegated_to_ai"] == "codex"
    ]
    assert len(codex_features) == 10

def test_feature_events_visible_in_delegation():
    """Feature creation events appear in subagent session"""
    events = db.get_events(
        session_id="0e6fd1e4-bc71-4424-88d4-3e88562ba5ed",
        feature_id="feature-20251221-033403"
    )
    assert len(events) == 640
```

---

### Manual Testing

1. **Dashboard View:**
   - Open main session (sess-fd50862f)
   - Verify Activity Feed shows 8,407 events
   - Check if "Related Delegations" section appears
   - Click into Codex delegation
   - Verify 2,962 events shown for subagent

2. **Event Query:**
   ```bash
   # Query subagent events directly
   sqlite3 .wipnote/index.sqlite \
     "SELECT COUNT(*) FROM events \
      WHERE session_id='0e6fd1e4-bc71-4424-88d4-3e88562ba5ed';"
   # Should return: 2962
   ```

3. **Feature Visibility:**
   - Check `.wipnote/features/` directory
   - Verify 10 Codex-created features are there
   - Verify events show how they were created

---

## Priority Ranking

| Priority | Item | Effort | Impact |
|----------|------|--------|--------|
| 🔴 High | Add Task event detail to dashboard | 20 min | Shows delegation target |
| 🔴 High | Create session lookup API | 30 min | Enable subagent queries |
| 🟠 Medium | Extend events schema | 30 min | Foundation for better tracking |
| 🟠 Medium | Update event recording | 20 min | Populate delegation fields |
| 🟠 Medium | Add delegation queries | 45 min | Enable tree queries |
| 🟡 Low | Dashboard delegation tree UI | 1.5 hrs | Better visualization |
| ⚪ Future | Complete redesign | 2-3 days | Architectural improvement |

---

## Files to Modify

### Short-term
1. `src/python/wipnote/dashboard.html` - Add Task detail display
2. `src/python/wipnote/server.py` - Add lookup API endpoint

### Medium-term
1. `src/python/wipnote/analytics_index.py` - Add schema + queries
2. `src/python/wipnote/event_log.py` - Populate delegation fields
3. `src/python/wipnote/dashboard.html` - Show delegation tree

### Long-term
1. `src/python/wipnote/event_log.py` - Redesign recording
2. `src/python/wipnote/analytics_index.py` - Redesign schema
3. `src/python/wipnote/dashboard.html` - Simplify queries

---

## Success Criteria

✓ Dashboard shows when events are from delegated work
✓ Can view subagent session from main session
✓ Feature creation events are visible
✓ Delegation tree shows all levels
✓ Total event count accurate (11,369 not 8,407)
✓ Cost/token tracking includes delegations
✓ Analytics account for all work across sessions

---

## Next Steps

1. **Immediate:** Read `/Users/shakes/DevProjects/htmlgraph/SUBAGENT_EVENT_TRACKING_INVESTIGATION.md` for full context

2. **Week 1:** Implement short-term fixes (1.5 hours)
   - Add Task event detail
   - Create lookup API
   - Document changes

3. **Week 2:** Implement medium-term fixes (3-5 hours)
   - Extend schema
   - Add delegation queries
   - Update dashboard UI

4. **Week 3+:** Plan long-term redesign
   - Evaluate unified event stream architecture
   - Design new schema
   - Plan migration strategy

---

## Questions to Answer

1. Should subagent sessions be hidden or browsable?
   - Current: Hidden (invisible unless you know session_id)
   - Recommended: Visible but linked to parent

2. Should we backfill parent_session_id for old events?
   - Current JSONL has event data
   - Could infer relationships from timestamps
   - Or just populate for new events going forward

3. Should delegation be first-class in analytics?
   - Reports: "Show Codex work across all sessions"
   - Filtering: "Show only delegated features"
   - Cost: "Cost of Task() delegation"

4. Do we want nested delegations (delegations that spawn more delegations)?
   - Current: Only one level (main → subagent)
   - Future: Could have delegations that spawn more delegations

---

## Related Issues

- Feature visibility: 10 features exist but creation process invisible
- Cost tracking: Can't see cost of Task() delegation
- Analytics accuracy: Reports miss subagent work
- Session continuity: Incomplete picture of feature touches
- Tool transitions: Can't analyze full workflow including delegations

---

## References

- Investigation Report: `SUBAGENT_EVENT_TRACKING_INVESTIGATION.md`
- Diagram: `SUBAGENT_EVENT_VISIBILITY_DIAGRAM.txt`
- Event Log Code: `src/python/wipnote/event_log.py`
- Analytics Index: `src/python/wipnote/analytics_index.py`
- Dashboard: `src/python/wipnote/dashboard.html`
- Server API: `src/python/wipnote/server.py`
