# Subagent Event Attribution Bug - Complete Investigation Index

## Overview

This directory contains a complete investigation of the subagent event attribution bug where subagent tool calls (Read, Grep, Edit, etc.) are incorrectly recorded to the parent orchestrator's session instead of a separate subagent session.

**Status**: ROOT CAUSE IDENTIFIED | FIX DESIGNED | READY FOR IMPLEMENTATION

---

## Documents in This Investigation

### 1. **Executive Summary** (Start Here)
📄 **File**: `SUBAGENT_ATTRIBUTION_BUG_SUMMARY.md`

**Best for**: Quick understanding of the problem, root cause, and where to fix

**Contains**:
- One-sentence problem statement
- Three missing pieces explanation
- Code locations with snippets
- Impact matrix
- Why this happened
- Before/after comparison

**Read time**: 5 minutes

---

### 2. **Complete Investigation Report** (Deep Dive)
📄 **File**: `SUBAGENT_ATTRIBUTION_BUG_INVESTIGATION.md`

**Best for**: Complete technical understanding, implementation planning, testing

**Contains**:
- Detailed problem statement with examples
- Root cause analysis (5 sections)
- Data flow analysis showing current vs expected
- Solution architecture with 3 fixes
- Implementation checklist
- Testing strategy with code examples
- Deployment plan
- Success criteria
- Risk assessment
- File modifications list

**Read time**: 30 minutes

**Key Sections**:
- Root Cause Analysis (most important)
- Solution Architecture (implementation guide)
- Testing Strategy (verification approach)

---

### 3. **Flow Diagrams** (Visual Understanding)
📄 **File**: `SUBAGENT_ATTRIBUTION_BUG_FLOW_DIAGRAM.md`

**Best for**: Visual learners, understanding the data flow, presentation

**Contains**:
- ASCII flow diagrams showing current (broken) state
- ASCII flow diagrams showing fixed state
- Session lifecycle timeline comparison
- Environment variable flow comparison
- Database state before/after
- Dashboard UI appearance before/after

**Read time**: 15 minutes

**Diagrams**:
- Current (Broken) Flow
- Fixed Flow (After Applying Fixes)
- Session Lifecycle Timeline
- Environment Variable Flow

---

### 4. **Exact Code Locations** (Implementation Guide)
📄 **File**: `SUBAGENT_ATTRIBUTION_BUG_CODE_LOCATIONS.md`

**Best for**: Developers implementing the fix, precise line numbers and code

**Contains**:
- Quick reference table
- Detailed code analysis with line numbers
- Current code snippets
- What's missing code snippets
- Exact code to add
- Testing templates
- Database schema verification
- Migration scripts
- Environment variable reference
- Quick debug commands

**Read time**: 20 minutes

**Key Sections**:
- Issue #1: PreToolUse Hook (Lines 412-430)
- Issue #2: Track Event Hook (Lines 710-723)
- Testing Locations
- Debug Commands

---

## Quick Navigation

### I want to understand the problem
1. Read: `SUBAGENT_ATTRIBUTION_BUG_SUMMARY.md` (5 min)
2. Look at: `SUBAGENT_ATTRIBUTION_BUG_FLOW_DIAGRAM.md` (15 min)

### I want to implement the fix
1. Read: `SUBAGENT_ATTRIBUTION_BUG_SUMMARY.md` (5 min)
2. Read: `SUBAGENT_ATTRIBUTION_BUG_CODE_LOCATIONS.md` (20 min)
3. Follow Implementation Checklist in `SUBAGENT_ATTRIBUTION_BUG_INVESTIGATION.md`

### I want to present this to others
1. Use `SUBAGENT_ATTRIBUTION_BUG_SUMMARY.md` for intro (2 min)
2. Show `SUBAGENT_ATTRIBUTION_BUG_FLOW_DIAGRAM.md` for visuals (5 min)
3. Reference `SUBAGENT_ATTRIBUTION_BUG_INVESTIGATION.md` for details (Q&A)

### I want to write tests
1. Read: `SUBAGENT_ATTRIBUTION_BUG_INVESTIGATION.md` → Testing Strategy section
2. Use templates in `SUBAGENT_ATTRIBUTION_BUG_CODE_LOCATIONS.md`

### I need to debug
1. Use: `SUBAGENT_ATTRIBUTION_BUG_CODE_LOCATIONS.md` → Quick Debug Commands
2. Reference: `SUBAGENT_ATTRIBUTION_BUG_FLOW_DIAGRAM.md` → Session Lifecycle

---

## The Bug at a Glance

```
PROBLEM:
  Orchestrator (Sonnet) + Task() spawns subagent (Opus)
  ✓ Task() recorded correctly
  ✗ Subagent's Read/Grep/Edit ALSO recorded to parent session
  ✗ Cannot distinguish orchestrator from subagent work

ROOT CAUSE:
  Event tracking hook uses global session cache (.wipnote/session.json)
  instead of checking environment variables for subagent context.

THE FIX:
  1. PreToolUse: Set env vars HTMLGRAPH_SUBAGENT_TYPE, HTMLGRAPH_PARENT_SESSION
  2. TrackEvent: Check for subagent env vars BEFORE using global cache
  3. Create new subagent session with parent link if subagent detected

EFFORT:
  ~30 lines of code changes across 2 files
  ~200 lines of tests
  Low risk (infrastructure exists, just needs to be used)
```

---

## Files to Change

| File | Lines | Change | Complexity |
|------|-------|--------|------------|
| `packages/claude-plugin/.claude-plugin/hooks/scripts/pretooluse-spawner-router.py` | 412-430 | Add 20 lines to set environment variables | Low |
| `src/python/wipnote/hooks/event_tracker.py` | 710-723 | Replace 14 lines with 45 lines to detect and handle subagent | Medium |
| `src/python/wipnote/hooks/context.py` | ~105 | Add 15 line documentation comment | Low |

---

## Key Points

### Root Cause
Line 711 of `event_tracker.py` uses `manager.get_active_session()` which reads the global shared session cache, bypassing environment variable checks for subagent context.

### Why It Matters
- Subagent work is misattributed to parent
- Cannot distinguish which events are from orchestrator vs subagent
- Model tracking is confused (shows parent model for subagent events)
- Cost analysis is incorrect
- Dashboard is confusing

### Infrastructure Exists
- Database has `is_subagent`, `parent_session_id` columns ✓
- SessionManager supports `is_subagent` parameter ✓
- SpawnerEventTracker class exists for this ✓
- Just need to USE them! ✗

### Solution Is Simple
Check for subagent environment variables before using global cache. Create new session if subagent detected.

---

## Implementation Timeline

**Phase 1 (1 hour)**: Code changes
- Update PreToolUse hook to set env vars
- Update Track Event hook to detect subagent
- Add documentation comment

**Phase 2 (2 hours)**: Testing
- Write unit tests
- Write integration tests
- Run existing tests

**Phase 3 (1 hour)**: Verification
- Deploy to test environment
- Manual testing with actual spawner
- Verify database state

**Phase 4 (30 min)**: Monitoring
- Add logging
- Deploy to production
- Monitor for issues

**Total**: ~4.5 hours

---

## Success Criteria

After implementing the fix:

1. ✓ Separate subagent sessions created (`is_subagent=1`)
2. ✓ Subagent events in subagent session (not parent)
3. ✓ Parent-child session links established
4. ✓ Parent-child event links established
5. ✓ Correct model shown for each event
6. ✓ Dashboard shows clear separation
7. ✓ Cost analysis accurate per agent
8. ✓ All existing tests pass

---

## Questions?

### What if the parent session ID is not available?
**Answer**: Create standalone subagent session (fallback in code at lines 728-742 of event_tracker.py)

### Will this break existing orchestrator workflows?
**Answer**: No. Normal orchestrator flow is unchanged. Only subagents get new behavior.

### Do we need database migrations?
**Answer**: Check `SUBAGENT_ATTRIBUTION_BUG_CODE_LOCATIONS.md` → Database Schema Verification section. Likely no (columns probably exist).

### How do we verify the fix worked?
**Answer**: Use debug commands in `SUBAGENT_ATTRIBUTION_BUG_CODE_LOCATIONS.md` or see `SUBAGENT_ATTRIBUTION_BUG_INVESTIGATION.md` → Success Criteria section

---

## Related Issues

This bug explains existing but unused infrastructure:
- `HTMLGRAPH_PARENT_SESSION` env var (documented but never used)
- `is_subagent` database field (never set to True)
- `parent_session_id` database field (never populated)
- `SpawnerEventTracker` class (designed but not integrated)

The fix will enable this existing system to work as designed.

---

## References

- Session Manager: `src/python/wipnote/session_manager.py`
- Event Tracker: `src/python/wipnote/hooks/event_tracker.py`
- Hook Context: `src/python/wipnote/hooks/context.py`
- PreToolUse Hook: `packages/claude-plugin/.claude-plugin/hooks/scripts/pretooluse-spawner-router.py`
- Database Schema: `src/python/wipnote/db/schema.py`
- Tests: `tests/integration/test_orchestrator_spawner_delegation.py`

---

## Document Metadata

**Investigation Date**: 2026-01-11
**Status**: COMPLETE - Ready for implementation
**Confidence Level**: HIGH (all evidence collected and analyzed)
**Risk Level**: LOW (using existing infrastructure)

**Next Steps**:
1. Read `SUBAGENT_ATTRIBUTION_BUG_SUMMARY.md`
2. Review `SUBAGENT_ATTRIBUTION_BUG_CODE_LOCATIONS.md`
3. Start implementation following checklist in `SUBAGENT_ATTRIBUTION_BUG_INVESTIGATION.md`
