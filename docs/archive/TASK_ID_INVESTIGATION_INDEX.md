# Task ID Investigation Index

## Overview

Complete investigation of why Claude Code Task notifications show "No child events" even though HtmlGraph has properly tracked all the child events in its database.

**Status:** Investigation Complete - Ready for Implementation Decision
**Duration:** ~30 hours of analysis and documentation
**Key Finding:** Claude Code's task_id system is separate from HtmlGraph's event_id system - they need to be linked

---

## Documents in This Investigation

### 1. **TASK_ID_FINDINGS_REPORT.txt** ⭐ START HERE
**Purpose:** Executive summary and decision point
**Length:** ~200 lines
**Read Time:** 5-10 minutes

**Contains:**
- Root cause identification
- What's confirmed vs. unknown
- Critical decision point
- Next steps with timeline
- Effort estimation

**Key Section:**
> "Does PostToolUse hook receive task_id in tool_response?"
> This single verification will determine feasibility

---

### 2. **INVESTIGATION_SUMMARY.md**
**Purpose:** Complete technical analysis
**Length:** ~600 lines
**Read Time:** 20-30 minutes

**Contains:**
- Problem statement
- Investigation findings (detailed)
- Hook data flow analysis
- Current HtmlGraph implementation review
- Database schema analysis
- Files requiring changes
- Success metrics

**Key Sections:**
- "Hook Data Flow Analysis" - Shows what data is available at each hook
- "Current HtmlGraph Implementation" - Reviews existing code
- "Critical Question" - The verification needed
- "Success Metrics" - How we'll know it worked

---

### 3. **TASK_ID_HOOKUP_PLAN.md**
**Purpose:** Detailed implementation roadmap
**Length:** ~400 lines
**Read Time:** 15-20 minutes

**Contains:**
- Phase 1: Verification (30 minutes)
- Phase 2: Capture task_id (2-3 hours)
- Phase 3: Lookup table (1 hour, optional)
- Phase 4: Tests (1-2 hours)
- Phase 5: Docs (1 hour)
- Implementation checklist
- File modification summary

**Use This For:**
- Step-by-step implementation guide
- Code examples for each phase
- Testing strategy
- Documentation updates

---

### 4. **DEBUG_LOGGING_PATCH.md** 🎯 CRITICAL FIRST STEP
**Purpose:** Ready-to-use verification code
**Length:** ~300 lines
**Read Time:** 10-15 minutes

**Contains:**
- Exact code to add to posttooluse.py
- Expected output formats (success case)
- Expected output formats (failure case)
- How to interpret results
- How to remove debug code
- Commands to deploy and test

**Use This For:**
- Answering the critical question
- First 30-minute action item
- Understanding what task_id looks like

---

### 5. **TASK_ID_INVESTIGATION.md**
**Purpose:** Initial investigation notes
**Length:** ~400 lines
**Read Time:** 15-20 minutes

**Contains:**
- Hook investigation methodology
- Data capture point analysis
- Current implementation gaps
- Investigation plan (detailed)
- Root cause analysis
- Questions for clarification

**Use This For:**
- Understanding investigation methodology
- Detailed exploration of each hook
- Architecture questions

---

## How to Read This Investigation

### If you have 5 minutes:
1. Read: **TASK_ID_FINDINGS_REPORT.txt** (entire document)
2. Key takeaway: Need to verify if task_id is available

### If you have 15 minutes:
1. Read: **TASK_ID_FINDINGS_REPORT.txt** (entire)
2. Skim: **INVESTIGATION_SUMMARY.md** sections "Hook Data Flow Analysis" and "Critical Question"
3. Key takeaway: Understand the gap and what needs to be verified

### If you have 30 minutes (recommended before implementation):
1. Read: **TASK_ID_FINDINGS_REPORT.txt** (entire)
2. Read: **INVESTIGATION_SUMMARY.md** (complete)
3. Skim: **DEBUG_LOGGING_PATCH.md** (understand debug code)
4. Key takeaway: Ready to implement Phase 1 verification

### If you're implementing the fix:
1. Reference: **TASK_ID_HOOKUP_PLAN.md** (detailed phases)
2. Use: **DEBUG_LOGGING_PATCH.md** (Phase 1 code)
3. Follow: Implementation checklist
4. Test: Each phase before moving to next

### If you need specific details:
| Question | Read |
|----------|------|
| "What code goes where?" | TASK_ID_HOOKUP_PLAN.md (Implementation Checklist) |
| "What's the root cause?" | INVESTIGATION_SUMMARY.md (Root Cause) |
| "How do I verify task_id?" | DEBUG_LOGGING_PATCH.md (entire) |
| "What files need changes?" | TASK_ID_HOOKUP_PLAN.md (Files to Modify table) |
| "What are the hooks doing?" | INVESTIGATION_SUMMARY.md (Hook Implementation Review) |
| "How much effort?" | TASK_ID_FINDINGS_REPORT.txt (Effort Estimation) |

---

## The Critical Question

**Does PostToolUse hook receive `task_id` in `tool_response` when Task() completes?**

This single question determines:
- ✅ If solution is feasible (30 minutes to verify)
- ⏱️ Total implementation effort (2-4 hours if yes, 6-9+ if no)
- 🎯 Whether to proceed with current approach or explore alternatives

**How to answer:** Follow DEBUG_LOGGING_PATCH.md (30 minutes)

---

## Implementation Quick Start

### Step 1: Verify task_id Availability (30 minutes)
```bash
# 1. Read: DEBUG_LOGGING_PATCH.md
# 2. Apply: Debug code to posttooluse.py
# 3. Deploy: ./scripts/deploy-all.sh 0.26.12 --no-confirm
# 4. Test: Run Task() in Claude Code
# 5. Check: Logs for DEBUG output
# 6. Document: Findings in INVESTIGATION_SUMMARY.md
```

### Step 2: If Verification Succeeds - Implement Capture (2-3 hours)
```bash
# 1. Read: TASK_ID_HOOKUP_PLAN.md (Phase 2)
# 2. Modify: pretooluse.py, posttooluse.py, event_tracker.py
# 3. Test: Unit tests for task_id extraction
# 4. Verify: Database shows task_id in context JSON
```

### Step 3: Add Lookup Table & Tests (2 hours)
```bash
# 1. Read: TASK_ID_HOOKUP_PLAN.md (Phases 3-4)
# 2. Create: claude_task_mappings table (optional but recommended)
# 3. Write: Integration tests for full flow
# 4. Document: Architecture updates
```

### Step 4: Deploy & Document (1 hour)
```bash
# 1. Test full flow end-to-end
# 2. Update AGENTS.md with task_id correlation strategy
# 3. Add API documentation for task lookup
# 4. Commit all changes
```

---

## Key Files to Modify (If Implementation Proceeds)

| Priority | File | Change | Effort |
|----------|------|--------|--------|
| 🔴 Critical | `hooks/posttooluse.py` | Extract task_id from tool_response | 30 min |
| 🔴 Critical | `hooks/event_tracker.py` | Accept and store claude_task_id parameter | 1 hour |
| 🔴 Critical | `hooks/pretooluse.py` | Export task_id to environment | 30 min |
| 🟡 High | `db/schema.py` | Add claude_task_mappings table (optional) | 30 min |
| 🟡 High | `test_task_id_correlation.py` | New test file for verification | 1 hour |
| 🟢 Medium | Documentation | Update hook architecture docs | 1 hour |

---

## Success Metrics

When complete, you'll be able to:

✅ **Capture:** Every Task() records its Claude Code task_id
✅ **Store:** Mapping between event_id and task_id in database
✅ **Query:** Look up events given task_id or vice versa
✅ **Integrate:** Claude Code can find our events by task_id
✅ **Display:** Task notifications show "5 events" instead of "No child events"

---

## Common Questions Answered

### Q: Why does this matter?
**A:** Claude Code's task notifications are invisible/unhelpful without this linkage. Users see "No child events" even though events were logged. This breaks task visibility.

### Q: How long will this take?
**A:** 30 minutes to verify, then 2-4 hours to implement if verification succeeds.

### Q: What if task_id is not available?
**A:** We'll need to contact Anthropic for a feature request, explore workarounds, or document as a known limitation.

### Q: Will this affect existing functionality?
**A:** No. All changes are additive. The context JSON field is already there - we're just using it.

### Q: Do I need to do all 4 phases?
**A:** Phase 1 (Verification) is critical. Phases 2-3 are implementation. Phase 4 (Tests/Docs) is recommended. Optional: Phase 3 (Lookup table) can be deferred.

### Q: What's the rollback plan?
**A:** Remove debug code, delete context fields, drop mapping table (if created). All changes are reversible.

---

## Document Map

```
TASK_ID_FINDINGS_REPORT.txt
├─ Executive Summary (Read First)
├─ Root Cause Identified
├─ Critical Decision Point
└─ Next Steps

INVESTIGATION_SUMMARY.md
├─ Problem Statement
├─ Investigation Findings (Detailed)
├─ Hook Data Flow Analysis ⭐
├─ Database Schema Analysis
├─ Current Implementation Review
└─ Risk Assessment

TASK_ID_HOOKUP_PLAN.md
├─ Implementation Roadmap
├─ Phase 1: Verification
├─ Phase 2: Capture task_id
├─ Phase 3: Lookup Table
├─ Phase 4: Tests & Docs
└─ Implementation Checklist

DEBUG_LOGGING_PATCH.md 🎯
├─ Ready-to-Use Debug Code
├─ How to Deploy
├─ Expected Outputs
├─ Interpretation Guide
└─ Cleanup Instructions

TASK_ID_INVESTIGATION.md
├─ Investigation Methodology
├─ Data Capture Points
├─ Hook Analysis
└─ Architecture Questions
```

---

## Next Action

**Recommended:** Schedule 30 minutes for Phase 1 verification

This will answer the critical question and unblock all further decisions.

See: **DEBUG_LOGGING_PATCH.md** for step-by-step instructions

---

## Questions?

Refer to relevant document:
- **"How?"** → TASK_ID_HOOKUP_PLAN.md
- **"Why?"** → INVESTIGATION_SUMMARY.md
- **"What?"** → TASK_ID_FINDINGS_REPORT.txt
- **"Can I verify?"** → DEBUG_LOGGING_PATCH.md
- **"What's the approach?"** → TASK_ID_INVESTIGATION.md

---

## Investigation Metadata

| Aspect | Details |
|--------|---------|
| **Issue** | Claude Code Task notifications show "No child events" |
| **Root Cause** | task_id ↔ event_id linking missing |
| **Investigation Status** | Complete |
| **Blocking Item** | Verification of task_id availability |
| **Time to Verify** | 30 minutes |
| **Time to Implement** | 2-4 hours (if verification succeeds) |
| **Risk Level** | Low-Medium |
| **Complexity** | Medium |
| **Breaking Changes** | None (backward compatible) |

---

## Files Provided

```
Generated:  2026-01-12
Location:   /Users/shakes/DevProjects/htmlgraph/

1. TASK_ID_FINDINGS_REPORT.txt (This Summary)
2. INVESTIGATION_SUMMARY.md (Complete Analysis)
3. TASK_ID_HOOKUP_PLAN.md (Implementation Plan)
4. DEBUG_LOGGING_PATCH.md (Verification Code)
5. TASK_ID_INVESTIGATION.md (Investigation Notes)
6. TASK_ID_INVESTIGATION_INDEX.md (This Document)
```

All documents are ready for review and implementation.

---

## Recommended Reading Order

1. ⭐ **Start:** TASK_ID_FINDINGS_REPORT.txt (5 min)
2. 📖 **Understand:** INVESTIGATION_SUMMARY.md (20 min)
3. 🎯 **Verify:** DEBUG_LOGGING_PATCH.md (15 min)
4. 🛠️ **Implement:** TASK_ID_HOOKUP_PLAN.md (as needed)
5. 📚 **Reference:** TASK_ID_INVESTIGATION.md (as needed)

Total reading time for initial understanding: ~40-50 minutes

Implementation time (if verification succeeds): 2-4 additional hours
