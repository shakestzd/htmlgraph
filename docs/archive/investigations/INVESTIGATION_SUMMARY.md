# Subagent Event Tracking Investigation - Executive Summary

**Date:** January 6, 2026
**Status:** Complete ✓
**Deliverables:** 3 detailed documents + investigation findings

---

## The Problem

**Dashboard Activity Feed doesn't show events from Task() delegations (subagents)**

When you delegate work to a subagent (e.g., Codex), their events are invisible on the main dashboard Activity Feed.

**Example:**
- Main session (sess-fd50862f): 8,407 events visible ✓
- Codex subagent work (0e6fd1e4-bc...): 2,962 events invisible ✗
- Users see only 74% of the actual work

---

## The Finding

**Events ARE being recorded correctly. The problem is architectural, not a bug.**

```
┌──────────────────────────────────────────────────┐
│ REALITY                                          │
├──────────────────────────────────────────────────┤
│ Codex Agent Created:                            │
│ • 2,962 total events                            │
│ • 10 new features                               │
│ • 1,369 feature-creation events (46% of work)   │
│ • Complete work history in SQLite               │
└──────────────────────────────────────────────────┘
                        ↓
┌──────────────────────────────────────────────────┐
│ BUT...                                           │
├──────────────────────────────────────────────────┤
│ • Events stored in separate session file        │
│ • Dashboard only queries main session           │
│ • No link between Task() and subagent session   │
│ • Dashboard never learns about subagent work    │
└──────────────────────────────────────────────────┘
```

---

## What's Actually Happening

### Data Flow

1. **Subagent spawned** via `Task(prompt="...", subagent_type="codex")`
2. **Runs in isolated process** with unique session_id: `0e6fd1e4-bc71-4424-88d4-3e88562ba5ed`
3. **Creates 2,962 events** - all recorded to `.wipnote/events/0e6fd1e4-bc...jsonl`
4. **Creates 10 features** - visible in `.wipnote/features/`
5. **Subagent completes** and returns to main orchestrator
6. **Main session logs** only "Task completed successfully"
7. **Dashboard queries** main session only: `WHERE session_id='sess-fd50862f'`
8. **Subagent events never fetched** ✗

### Database Reality

**SQLite index.sqlite contains EVERYTHING:**

```sql
-- Query 1: Main session events (VISIBLE)
SELECT COUNT(*) FROM events
WHERE session_id='sess-fd50862f'
→ 8,407 events ✓ (shown on dashboard)

-- Query 2: Subagent events (INVISIBLE)
SELECT COUNT(*) FROM events
WHERE session_id='0e6fd1e4-bc71-4424-88d4-3e88562ba5ed'
→ 2,962 events ✗ (never queried by dashboard)

-- Query 3: Total work
SELECT COUNT(*) FROM events
→ 11,369 events (database knows about all work)
```

**The data is there. It's just not being queried.**

---

## Codex's Actual Work Output

### Features Created
```
feature-20251221-033403 ........... 640 events
feature-self-tracking ............ 504 events
feature-20251217-015856 .......... 163 events
feature-commit-graph-analytics ... 92 events
feature-20251221-034848 .......... 86 events
feature-git-hook-foundation ...... 44 events
feature-20251221-034838 .......... 28 events
feature-precommit-reminder ....... 10 events
test-auto-reload ................. 1 event
feature-old-001 .................. 1 event
                              ──────────
                              1,369 events = 46% of Codex work
```

### Tool Usage Distribution
- Bash: 767 events (26%)
- Read: 355 events (12%)
- Browser/Computer Control: 330 events (11%)
- Edit: 297 events (10%)
- Grep: 228 events (8%)
- UserQuery: 188 events (6%)
- Other: 797 events (27%)

### Session Details
```
Session ID: 0e6fd1e4-bc71-4424-88d4-3e88562ba5ed
Agent: claude-code
Total Events: 2,962
Duration: 2025-12-16 to 2025-12-22 (6 days)
Status: Active (visible in database)
Visibility: Hidden from dashboard ✗
```

---

## Root Cause

**One-sentence summary:**
The dashboard architecture assumes a single session per view, so when a subagent creates a separate session, that session and its 2,962 events are never queried.

**Technical detail:**
```python
# Current dashboard query
def session_events(self, session_id: str):
    return db.execute("""
        SELECT * FROM events
        WHERE session_id = ?  ← Only this session!
        ORDER BY ts DESC
    """, (session_id,))

# No mechanism to:
# 1. Discover related subagent sessions
# 2. Link Task() events to their subagent sessions
# 3. Aggregate events across related sessions
```

---

## Why This Matters

### Lost Visibility
- 26% of work (2,962 events) invisible
- Feature creation process hidden
- Can't see what tools were used
- Can't track feature development

### Broken Analytics
- "Total events" undercount (8,407 vs 11,369)
- Feature continuity incomplete
- Tool transition analysis broken
- Workflow patterns incomplete

### Cost Implications
- Can't calculate true cost of Task() delegations
- Token usage in subagents uncounted
- Performance metrics incomplete

### User Experience
- "Task completed" looks like minimal work
- Actually created 10 features
- Invisible work seems wasteful

---

## What's NOT Broken

✓ **Event Recording Works Perfectly**
- All 2,962 subagent events recorded
- Full JSONL files maintained
- SQLite index complete
- No data loss

✓ **Features Work**
- 10 features created successfully
- All visible in `.wipnote/features/`
- Data is there
- Just not connected to creation events

✓ **Database Has Everything**
- Query the subagent session directly and you'll see all work
- SQLite rebuild works correctly
- Event log files are pristine

---

## Solutions Available

### Short-term (1-2 hours)
1. Add Task event detail showing delegation target
2. Create API to find subagent session by task_id
3. Add "Related Sessions" section to dashboard
4. Cost: Low | Impact: Medium

### Medium-term (3-5 hours)
1. Extend events table with delegation fields
2. Update event recording to populate them
3. Add delegation queries to analytics
4. Update dashboard UI for delegation trees
5. Cost: Medium | Impact: High

### Long-term (1-2 days)
1. Redesign for unified event stream
2. Use parent-child relationships throughout
3. Eliminate session isolation
4. Proper breadcrumb trails for all work
5. Cost: High | Impact: Very High

---

## Key Statistics

| Metric | Value |
|--------|-------|
| Main Session Events | 8,407 |
| Subagent Events | 2,962 |
| Hidden Events | 2,962 (26%) |
| Total Actual Work | 11,369 |
| Dashboard Shows | 8,407 (74%) |
| Features Created | 10 |
| Feature Events | 1,369 |
| Sessions in Database | 15 |
| Sessions Visible on Dashboard | 1 |
| Hidden Sessions | 14 |
| Database Files | 4.8 MB |

---

## Files Involved

### Investigation Documents (New)
- `SUBAGENT_EVENT_TRACKING_INVESTIGATION.md` - Detailed technical analysis
- `SUBAGENT_EVENT_VISIBILITY_DIAGRAM.txt` - ASCII diagrams and flow charts
- `SUBAGENT_VISIBILITY_ACTION_ITEMS.md` - Actionable next steps
- `INVESTIGATION_SUMMARY.md` - This document

### Core System Files
- `src/python/wipnote/event_log.py` - Event recording
- `src/python/wipnote/analytics_index.py` - SQLite indexing
- `src/python/wipnote/dashboard.html` - Activity Feed display
- `src/python/wipnote/server.py` - API endpoints
- `.wipnote/index.sqlite` - Database with all events
- `.wipnote/events/*.jsonl` - Event log files

---

## Recommended Next Steps

### Immediate (Today)
1. ✓ Read investigation documents (complete)
2. ✓ Understand root cause (complete)
3. ✓ Assess impact (complete)

### This Week
1. Implement short-term fix (1-2 hours)
   - Task event detail in dashboard
   - Session lookup API
2. Test with Codex subagent session
3. Verify 2,962 events become visible

### Next Week
1. Implement medium-term fixes (3-5 hours)
2. Add schema extensions
3. Update analytics queries
4. Improve dashboard UI

### Planning
1. Design long-term architecture
2. Plan unified event stream migration
3. Consider impact on existing queries
4. Plan rollout strategy

---

## Success Criteria

After fixes are implemented:

✓ Dashboard shows subagent work in Activity Feed
✓ Can view Codex subagent session from main session
✓ 2,962 events become visible
✓ 1,369 feature creation events tracked
✓ Total event count shows 11,369 (not 8,407)
✓ Feature continuity includes all touches
✓ Analytics complete across sessions
✓ Users see true scope of work

---

## Questions Answered

### Q: Are events being lost?
**A:** No. All 2,962 subagent events are recorded and in the database. They're just invisible to the dashboard.

### Q: Can I access the subagent events?
**A:** Yes, via direct SQLite query or if you know the session_id. But the dashboard doesn't know about them.

### Q: Are features broken?
**A:** No. 10 features were created successfully. They're visible in `.wipnote/features/`. Only the event history of their creation is hidden.

### Q: Is this a data loss bug?
**A:** No. This is an architectural limitation, not a bug. The dashboard design assumes single-session views.

### Q: How do I see the hidden work?
**A:**
```bash
# Query the subagent session directly
sqlite3 .wipnote/index.sqlite \
  "SELECT COUNT(*) FROM events WHERE session_id='0e6fd1e4-bc71-4424-88d4-3e88562ba5ed';"
# Returns: 2962
```

### Q: What's the priority to fix?
**A:** Medium-High. The work is complete and tracked, just invisible. Fix dashboard visibility first, then address long-term architecture.

---

## References

For detailed information, see:

1. **Technical Deep Dive**
   → `SUBAGENT_EVENT_TRACKING_INVESTIGATION.md`
   - Full database queries
   - Architecture explanation
   - Feature listing
   - Event distribution

2. **Visual Overview**
   → `SUBAGENT_EVENT_VISIBILITY_DIAGRAM.txt`
   - ASCII diagrams
   - Flow charts
   - Comparison tables
   - Data structure visualization

3. **Implementation Plan**
   → `SUBAGENT_VISIBILITY_ACTION_ITEMS.md`
   - Short/medium/long-term fixes
   - Time estimates
   - Code changes needed
   - Testing strategy

---

## Conclusion

**Status:** Subagent event tracking works correctly. The problem is architectural visibility, not data loss.

**Impact:** 26% of work (2,962 events) invisible on dashboard

**Solution:** Link subagent sessions to parent tasks (1-5 hours implementation)

**Timeline:**
- Week 1: Make events visible (short-term)
- Week 2: Improve UI (medium-term)
- Week 3+: Long-term architecture (optional)

**Bottom Line:** The Codex agent created 10 features with 2,962 events. This work is complete and in the database. It just needs to be shown on the dashboard.

---

## Contact & Questions

For questions about this investigation:
1. Review the detailed documents listed above
2. Check the action items for next steps
3. Refer to database queries for verification

---

**Investigation Complete** ✓
**Ready for Implementation** ✓
