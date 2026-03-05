# File Read Pattern Analysis - February 12, 2026

## Executive Summary

Analysis of HtmlGraph database reveals significant file read patterns that present opportunities for caching and optimization.

**Key Findings:**
- 1,378 total Read events across 54 unique sessions
- 594 reads had empty/null file paths (43% of all reads)
- Top 19 files read across 5+ different sessions (strong cache candidates)
- Average 25.5 reads per session
- Significant within-session re-reading (up to 22 reads of same file in one session)

---

## Database Overview

**Schema:** 20 tables including:
- `agent_events` - Tool execution tracking (6,459 total events)
- `sessions` - Session management (237 active sessions)
- `features`, `tracks` - Work tracking
- `cost_events` - Token/cost tracking (currently empty - not yet deployed)
- `delegation_patterns`, `delegation_suggestions` - AI learning system
- `tool_traces` - Detailed tool execution traces

**Date Range:** 2026-01-13 to 2026-02-13 (31 days)

**Activity Summary:**
- 237 total sessions (all active)
- 6,459 total events across all tools
- Average 42.6 events per session
- Peak activity: Feb 2 (1,220 events), Feb 10 (1,040 events), Feb 12 (1,033 events)

---

## Tool Usage Distribution

```
Tool Name                Count    % of Total
=========================================
Bash                     2,871    44.5%
Read                     1,378    21.3%
Edit                       580     9.0%
Grep                       336     5.2%
Task                       315     4.9%
Glob                       163     2.5%
Stop                       135     2.1%
UserQuery                  100     1.5%
TaskUpdate                  92     1.4%
Write                       72     1.1%
TaskOutput                  68     1.1%
Chrome DevTools            152     2.4%
Playwright                  40     0.6%
WebSearch/WebFetch          18     0.3%
Other                      139     2.2%
```

**Key Insight:** Read is the 2nd most used tool (21.3% of all events), making it a high-impact optimization target.

---

## File Read Patterns - Top Candidates for Caching

### Tier 1: Read Across 10+ Sessions (Critical Cache Priority)

| File Path | Sessions | Total Reads | Avg Reads/Session |
|-----------|----------|-------------|-------------------|
| `api/services.py` | 14 | 48 | 3.4 |
| `hooks/event_tracker.py` | 13 | 53 | 4.1 |
| `hooks/pretooluse.py` | 12 | 50 | 4.2 |
| `api/templates/partials/activity-feed.html` | 12 | 37 | 3.1 |
| `api/routes/dashboard.py` | 10 | 27 | 2.7 |
| `packages/claude-plugin/hooks/hooks.json` | 10 | 16 | 1.6 |

### Tier 2: Read Across 5-9 Sessions (High Cache Priority)

| File Path | Sessions | Total Reads |
|-----------|----------|-------------|
| `packages/claude-plugin/hooks/scripts/pretooluse-integrator.py` | 9 | 15 |
| `db/schema.py` | 8 | 14 |
| `packages/claude-plugin/hooks/scripts/track-event.py` | 8 | 13 |
| `api/templates/dashboard.html` | 7 | 10 |
| `api/main.py` | 7 | 75 |
| `tests/python/test_pretooluse_event_hierarchy.py` | 5 | 18 |
| `dashboard.html` | 5 | 10 |
| `cli/core.py` | 5 | 6 |
| `api/routes/orchestration.py` | 5 | 10 |
| `packages/claude-plugin/hooks/scripts/user-prompt-submit.py` | 5 | 13 |
| `packages/claude-plugin/hooks/scripts/posttooluse-integrator.py` | 5 | 6 |
| `packages/claude-plugin/agents/test-runner.md` | 5 | 6 |
| `packages/claude-plugin/agents/researcher.md` | 5 | 6 |

**Cache Impact Estimate:**
- Tier 1 files: 231 reads (16.8% of all reads)
- Tier 2 files: 116 reads (8.4% of all reads)
- **Total cache potential: 347 reads (25.2% of all Read events)**

---

## Within-Session Re-Reading Patterns

### Most Severe Cases (10+ reads same file, same session)

| Session ID | File | Reads |
|------------|------|-------|
| `6f0ea23f...` | (empty path) | 126 |
| `7d98e1b7...` | (empty path) | 101 |
| `7d98e1b7...-general-purpose` | (empty path) | 90 |
| `d62c4f67...` | (empty path) | 53 |
| `76ace32b...` | (empty path) | 52 |
| `ce37a5ff...` | `api/main.py` | 22 |
| `97e4f1d4...-htmlgraph:haiku-coder` | `api/main.py` | 17 |
| `6d85f8bb...` | `hooks/pretooluse.py` | 15 |
| `ce37a5ff...` | `api/templates/partials/activity-feed.html` | 15 |
| `6d85f8bb...` | `hooks/event_tracker.py` | 13 |

**Key Insights:**
- **Empty path reads:** 594 total (43% of all reads) - likely tool result files or temp files
- **api/main.py:** Read 22 times in one session, 17 times in another
- **Hook files:** Frequently re-read (pretooluse.py: 15x, event_tracker.py: 13x)

**Session-level caching opportunity:** Files read 3+ times in same session = significant waste

---

## Special Read Patterns (with "Read:" prefix)

261 reads use the format `Read: <file_path>` vs 1,119 using raw file path.

**Examples:**
- `Read: cli/work/snapshot.py` - 35 times total
- `Read: cli/base.py` - 27 times total
- `Read: api/main.py` - 16 times total

This prefix may indicate a different tool or display format.

---

## Subagent Analysis

### Subagent Activity Summary

- **26 subagent sessions** spawned (11% of all sessions)
- **Most active subagents:**
  - `7d98e1b7...-general-purpose`: 286 events
  - `d62c4f67...-general-purpose`: 269 events
  - `97e4f1d4...-htmlgraph:researcher`: 263 events
  - `97e4f1d4...-htmlgraph:sonnet-coder`: 242 events

### Files Most Read by Subagents

| File | Subagent Reads | % of Subagent Reads |
|------|----------------|---------------------|
| (empty path) | 199 | 29.6% |
| `api/main.py` | 50 | 7.4% |
| `api/services.py` | 28 | 4.2% |
| `hooks/event_tracker.py` | 25 | 3.7% |
| `hooks/pretooluse.py` | 20 | 3.0% |

**Total subagent reads: 672 (48.8% of all reads)**

**Duplication Issue:** Unable to find parent-child read duplication (parent_session_id is NULL for all subagents in current data). This suggests either:
1. Subagent tracking not fully implemented yet
2. Sessions marked as subagents but not properly linked to parents
3. Need to investigate session creation logic

---

## Other Tool Patterns

### Grep Patterns (Most Searched)

```
Pattern                                                     Count
================================================================
ORDER BY timestamp DESC                                      20
datetime\(REPLACE\(SUBSTR\(timestamp                          5
def track_activity                                            4
class InitCommand                                             4
class ActivityService                                         4
task.notification|task_notification|TaskNotification          3
```

Most grep searches are for database queries (ORDER BY, datetime) and class/function definitions.

### Glob Patterns (Most Used)

```
Pattern                                                     Count
================================================================
src/python/htmlgraph/api/services/**/*.py                     3
**/event_tracker.py                                           3
src/python/htmlgraph/cli/*.py                                 2
src/python/htmlgraph/api/services*.py                         2
src/python/htmlgraph/**/*.py                                  2
packages/claude-plugin/hooks/scripts/*.py                     2
packages/claude-plugin/**/*.md                                2
**/dashboard.html                                             2
**/activity-feed.html                                         2
```

Glob is used for multi-file discovery, particularly in API services and hook scripts.

---

## Model Distribution

```
Model          Events    % of Total
====================================
Opus 4.6       2,102     48.2%
Haiku 4.5      2,016     46.2%
Sonnet 4.5       250      5.7%
```

**Insight:** Nearly even split between Opus and Haiku, with Sonnet used sparingly (likely for specialized tasks).

---

## Cost Tracking Status

**Current State:** `cost_events` table exists but is empty (0 events).

**Implication:** Cost tracking hooks not yet deployed or not recording to this table. Session-level `total_tokens_used` is also 0/NULL for all sessions.

**Action Item:** Verify cost tracking implementation and deployment status.

---

## Sessions Overview

**Total Sessions:** 237 (all with status = 'active')

**Sessions with Most Events:**

| Session ID | Events |
|------------|--------|
| `ce37a5ff...` | 944 |
| `6f0ea23f...` | 738 |
| `7d98e1b7...` | 426 |
| `6d85f8bb...` | 354 |
| `d62c4f67...` | 354 |

**Activity Distribution:**
- Average: 42.6 events per session
- Range: ~10 to 944 events
- 54 sessions performed Read operations (22.8% of all sessions)

---

## Recommendations

### 1. Implement File Read Caching (High Priority)

**Target:** 19 files read across 5+ sessions (25.2% of all reads)

**Implementation Options:**

a) **Session-level cache** (simplest)
   - Cache file contents for duration of session
   - Clear on session end
   - Handles within-session re-reads (api/main.py: 22x → 1x)

b) **Global LRU cache** (more complex)
   - Share cache across all sessions
   - Invalidate on file modification (watch filesystem)
   - Handles cross-session reads (event_tracker.py: 13 sessions)

c) **Hybrid approach** (recommended)
   - Session-level cache for immediate re-reads
   - Global cache for frequently-accessed files (Tier 1)
   - File modification tracking for cache invalidation

**Estimated Impact:**
- Reduce 347 reads to ~19 cache misses (94% cache hit rate for Tier 1+2)
- Save ~5 Read operations per session on average
- Reduce token usage if Read results are sent to model

### 2. Investigate Empty File Path Reads

**Issue:** 594 reads (43%) have empty/null file paths

**Possible Causes:**
- Tool result files (temporary files)
- Missing hook data capture
- Placeholder reads for non-file resources

**Action:** Add debug logging to identify source of empty path reads

### 3. Investigate Subagent Parent Linking

**Issue:** All subagent sessions have `parent_session_id = NULL`

**Expected:** Subagents should link to parent session for traceability

**Action:** Review session creation logic in hooks (session-start.py, subagent-stop.py)

### 4. Enable Cost Tracking

**Issue:** `cost_events` table empty despite schema existing

**Action:** Verify cost tracking hooks are deployed and recording correctly

### 5. Monitor Re-Read Patterns

**Insight:** Some files read 15-22 times in single session

**Action:** 
- Add telemetry to track cache hit/miss rates
- Identify root cause of excessive re-reading
- Consider if file should be loaded once and held in memory

---

## Query Reference

For future analysis, here are the key queries used:

```sql
-- Top files by session count (cache candidates)
SELECT input_summary, COUNT(DISTINCT session_id) as session_count, COUNT(*) as total_reads
FROM agent_events 
WHERE tool_name = 'Read' AND input_summary IS NOT NULL AND input_summary NOT LIKE 'Read:%'
GROUP BY input_summary 
HAVING session_count >= 5
ORDER BY session_count DESC;

-- Within-session re-reads
SELECT session_id, input_summary, COUNT(*) as reads_in_session
FROM agent_events 
WHERE tool_name = 'Read' AND input_summary IS NOT NULL
GROUP BY session_id, input_summary 
HAVING reads_in_session > 2
ORDER BY reads_in_session DESC;

-- Tool usage distribution
SELECT tool_name, COUNT(*) as count 
FROM agent_events 
GROUP BY tool_name 
ORDER BY count DESC;

-- Daily activity
SELECT DATE(timestamp) as date, COUNT(*) as events 
FROM agent_events 
GROUP BY DATE(timestamp) 
ORDER BY date DESC;
```

---

**Generated:** 2026-02-12
**Analysis Period:** 2026-01-13 to 2026-02-13 (31 days)
**Data Source:** `/Users/shakes/DevProjects/htmlgraph/.htmlgraph/htmlgraph.db`
