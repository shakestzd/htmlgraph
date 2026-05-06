# Event Capture System - Next Steps

**Status:** System is healthy. Ready for real-world usage.

---

## Immediate Actions (Do Now)

### 1. Verify Everything Works
```bash
curl http://localhost:9999/api/events | jq '.[] | {event_id, tool_name, timestamp}' | head -50
```

Expected output: 3 events from January 8, 2026

### 2. Review Diagnostic Report
```bash
cat EVENT_CAPTURE_DIAGNOSTIC_REPORT.md
```

This document explains the full system status.

---

## Generate Real Events (Testing)

### Method A: Task() Delegation (Preferred)

**What it does:** Creates a real event that mimics actual multi-agent work

**How to do it:**
1. In Claude Code with orchestrator mode enabled:
```python
Task(
    prompt="Analyze the event capture system and verify it's working",
    subagent_type="haiku"
)
```

2. Watch the database update:
```bash
# Before
sqlite3 .wipnote/index.sqlite "SELECT COUNT(*) FROM agent_events"
# → 3

# Wait 5 seconds...

# After
sqlite3 .wipnote/index.sqlite "SELECT COUNT(*) FROM agent_events"
# → 4 (or higher)
```

3. Check the dashboard:
```bash
curl http://localhost:9999/api/events | jq '.[-1]'
```

**Result:** New event appears in database and dashboard within 5 seconds

**Timeline:**
- T0: Task() called → PreToolUse hook executes
- T0+1s: Subagent receives task and starts work
- T0+2-5s: Subagent completes → SubagentStop hook executes
- T0+5.5s: Dashboard updated with new event

### Method B: Run Tests

**What it does:** Creates test events to verify system functionality

**How to do it:**
```bash
uv run pytest tests/hooks/test_hybrid_event_capture.py -v
```

**Result:**
- 8 tests execute and pass
- Test events created in database
- No real delegation, but shows system working

### Method C: Manual Event Creation

**What it does:** Creates a single event programmatically

**How to do it:**
```python
from wipnote.hooks.event_tracker import track_tool_execution

track_tool_execution(
    tool_name="ManualTest",
    input_summary='{"test": "manual"}',
    result="Success",
    error=None
)
```

**Result:** Single event in database (limited functionality)

---

## Dashboard Walkthrough

### 1. Start Dashboard
```bash
# Already running on port 9999
# If not, start it:
uv run wipnote serve --port 9999
```

### 2. Access Dashboard
Open: http://localhost:9999

### 3. What to Look For

**Current state (with old test data):**
- 3 events displayed
- All from 2026-01-08 (2+ days ago)
- Shows test data from system verification

**After creating new events:**
- New events appear in list
- Most recent events show at top
- Timestamps update to current time
- Parent-child relationships visible

### 4. API Endpoint
```bash
curl http://localhost:9999/api/events | jq .
```

Returns all events as JSON array.

---

## Integrating Into Development Workflow

### Current State
- Most work: Direct agent execution (no events captured)
- Limitation: No event history for this work
- Result: Dashboard only shows old test data

### To Build Event History
1. **Use Task() for multi-agent work:**
   ```python
   Task(
       prompt="Delegate research to Gemini",
       subagent_type="gemini-spawner"
   )
   ```

2. **Events automatically captured:**
   - Triggered by Task() call
   - Recorded by hooks
   - Stored in database
   - Displayed in dashboard

3. **Dashboard becomes useful:**
   - Shows recent delegation history
   - Tracks subagent interactions
   - Provides audit trail of work
   - Enables analytics on patterns

### Best Practices
- Use Task() when delegating to subagents
- Let hooks capture events automatically
- Dashboard will show delegation chain
- Events accumulate over time for historical view

---

## Understanding What You're Seeing

### Old Test Data
The 3 events in the database are from when the event capture system was being tested and verified:

```
Event 1: Bash tool call test
Event 2: Task tool call test
Event 3: Task delegation test (status="started")
```

These are leftover from verification runs, not from actual recent development work.

### Why They're Still There
- System was tested and verified
- Events persisted to database
- They're not harmful - just historical test data
- New events will be added as Task() is used

### Expected Behavior
Once you create real events using Method A above:
- New event appears in database
- Dashboard updates automatically
- Event shows with current timestamp
- Old test data remains (for history)

---

## Monitoring Events

### Real-time Monitoring

**Watch database for new events:**
```bash
# In one terminal:
watch -n 1 'sqlite3 .wipnote/index.sqlite "SELECT COUNT(*) FROM agent_events; SELECT MAX(timestamp) FROM agent_events"'
```

**Watch API for updates:**
```bash
# In another terminal:
watch -n 2 'curl -s http://localhost:9999/api/events | jq ".[-1] | {event_id, timestamp, status}"'
```

### Event Log File
```bash
tail -f .wipnote/hook-debug.jsonl
```

Shows raw hook execution logs.

---

## Troubleshooting Quick Reference

| Issue | Check | Solution |
|-------|-------|----------|
| No events in dashboard | Database: `sqlite3 .wipnote/index.sqlite "SELECT COUNT(*) FROM agent_events"` | Create events using Method A |
| Old events only | Database timestamps | Use Task() to create new events |
| Dashboard not responding | `curl http://localhost:9999/api/events` | Restart dashboard: `pkill wipnote serve` then `uv run wipnote serve --port 9999` |
| Can't import Task | Context is direct Python | Switch to Claude Code environment |
| Hooks not running | Check `.claude/hooks/` directory | Verify hooks.json exists and is valid |

---

## Long-term Usage Pattern

### Week 1: Verification Phase
- ✓ Verify system works (you are here)
- ✓ Understand event capture pipeline
- Create a few test events
- Review dashboard functionality

### Week 2: Integration Phase
- Use Task() for multi-agent work
- Watch events accumulate in database
- Dashboard shows delegation history
- Become familiar with patterns

### Ongoing: Production Usage
- Events captured automatically when Task() used
- Dashboard provides work history
- Analytics possible on event data
- Audit trail available for compliance

---

## Questions & Answers

### Q: Why do I need to create events manually?
**A:** You don't! Events are created automatically when you use Task() to delegate work. This guide shows how to manually verify it works.

### Q: What if I don't use Task() in my work?
**A:** That's fine. Events only appear for delegated work. Direct execution doesn't create events.

### Q: Can I clear old test events?
**A:** Yes, but not recommended. They serve as verification that the system works. New real events will naturally accumulate over time.

### Q: How often should I create events?
**A:** Whenever you delegate work to subagents using Task(). Events are created automatically - no manual steps needed.

### Q: Can I query events programmatically?
**A:** Yes! Use the `/api/events` endpoint or query the database directly.

### Q: Are events stored permanently?
**A:** Yes. They're in SQLite database at `.wipnote/index.sqlite`. They persist between sessions.

### Q: What if I restart the dashboard?
**A:** Events persist in the database. Dashboard just reads from the database. Restarting doesn't lose any data.

---

## Summary

**Current Status:**
- ✓ System fully functional
- ✓ All tests passing
- ✓ Dashboard running
- ✓ Database operational
- ⚠ Only old test data (expected)

**What to Do Next:**
1. Review the diagnostic report
2. Create a test event using Method A
3. Verify it appears in dashboard
4. Use Task() for multi-agent work going forward

**Expected Timeline:**
- Review: 5-10 minutes
- Create test event: 30-60 seconds
- See it in dashboard: 5-10 seconds
- Total: 15-20 minutes to full understanding

---

**Files Created:**
- `EVENT_CAPTURE_DIAGNOSTIC_REPORT.md` - Full technical report
- `NEXT_STEPS.md` - This file (actionable next steps)
- `.wipnote/EVENT_CAPTURE_DIAGNOSTIC.md` - Summary for reference

**Ready to proceed?** Follow Method A above to create your first real event!
