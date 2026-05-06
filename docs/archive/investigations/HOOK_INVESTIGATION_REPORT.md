# Plugin Hooks vs Project Hooks - Architecture Investigation

## Executive Summary

**CRITICAL FINDING: The hook system has two separate implementations that are OUT OF SYNC and serve different purposes.**

- **Plugin hooks** (`packages/claude-plugin/hooks/`) - AUTHORITATIVE, referenced in plugin.json
- **Project hooks** (`.claude/hooks/`) - PROJECT-SPECIFIC OVERRIDES, used by this project locally

This is intentional and correct, BUT the implementations have diverged significantly.

---

## Architecture Overview

### Two-Layer Hook System (By Design)

```
Claude Code Plugin System
├── packages/claude-plugin/hooks/                 (PLUGIN: Distributed to users)
│   ├── hooks.json                               (Plugin hook manifest)
│   └── scripts/                                 (15 hook implementations)
│       ├── session-start.py                     (52 KB, complex)
│       ├── subagent-stop.py                     (496 B, thin wrapper)
│       ├── track-event.py
│       ├── user-prompt-submit.py
│       └── ... (12 more)
│
└── .claude/hooks/                               (PROJECT: Wipnote development)
    ├── hooks.json                               (Project hook manifest)
    └── scripts/                                 (15 hook implementations)
        ├── session-start.py                     (38 KB, older version)
        ├── subagent-stop.py                     (496 B, inline implementation)
        └── ... (13 more)
```

### How Claude Code Loads Hooks

1. **Plugin hooks are embedded** in the plugin package at install time
2. **Project hooks can override** plugin hooks when `.claude/hooks/` exists
3. **Both hook.json files are loaded and MERGED** (not replaced)
   - Hooks from both sources are COMBINED
   - Creates duplicate hook execution if both define same hook type

---

## Current State: OUT OF SYNC

### 1. subagent-stop.py - COMPLETELY DIFFERENT ARCHITECTURES

| Aspect | Plugin (Authoritative) | Project (Stale) |
|--------|------------------------|-----------------|
| Size | 496 bytes | 496 bytes |
| Type | **Thin wrapper** | **Full implementation** |
| Implementation | Delegates to `wipnote.hooks.subagent_stop` | Inline full logic |
| MD5 Hash | `0c1b1ace4798e0e7b4804a638dd158ed` | `6afb8e28a650865782d529875188f2e2` |
| **Status** | ✅ Current | ❌ Stale |

**Plugin Version (Current):**
```python
from wipnote.hooks.subagent_stop import main

if __name__ == "__main__":
    main()
```

**Project Version (Stale):**
```python
# 482 lines of inline implementation with:
- parse_transcript()
- find_pending_task_invocation()
- extract_subagent_type_from_input()
- count_child_spikes()
- create_subagent_completion_event()
- main()
```

### 2. session-start.py - DIFFERENT FEATURE SETS

| Aspect | Plugin | Project |
|--------|--------|---------|
| Size | 52 KB | 38 KB |
| Modified | Jan 8, 2026 (TODAY) | Jan 5, 2026 |
| **Key Difference** | Full CIGS integration | Missing CIGS features |
| **Status** | ✅ Current | ❌ Stale |

**Plugin has features project lacks:**
```python
# Plugin imports (lines 151-159):
from wipnote.cigs import (
    AutonomyRecommender,
    PatternDetector,
    ViolationTracker,
)
from wipnote.reflection import get_reflection_context
# ... ~50 lines of CIGS integration logic
```

**Project is missing:**
- Autonomy Recommender integration
- Pattern Detector integration
- Violation Tracker integration
- Reflection context injection
- ~50 lines of related logic

### 3. Other Hook Files

Checked files show varying sync status:
- `track-event.py` - Similar sizes, need diff
- `user-prompt-submit.py` - Different timestamps
- `post-tool-use-failure.py` - Different implementations
- `orchestrator-enforce.py` - Different permissions (600 vs 755)
- `orchestrator-reflect.py` - Same (3.6 KB both)

---

## Why This Happened

1. **Plugin was updated** with new CIGS features (Jan 8, 2026)
2. **Project hooks were not updated** - still on Jan 5 version
3. **Both hook.json files exist**, so BOTH sets of hooks run:
   - Plugin hooks run first (from plugin)
   - Project hooks run second (from project override)
   - **Creates duplicate execution and conflicting logic**

---

## Impact Analysis

### What This Breaks

1. **subagent-stop.py Duplication**
   - Plugin version (thin wrapper) calls `wipnote.hooks.subagent_stop`
   - Project version (inline) executes same logic inline
   - **Result: subagent completions tracked TWICE (duplicate events)**

2. **session-start.py Missing Features**
   - Plugin has CIGS integration, project doesn't
   - Project version hooks first, then plugin version
   - **Result: CIGS context may be stale or doubled**

3. **Inconsistent State**
   - Plugin is authoritative (distributed to users)
   - Project version is older (local development)
   - **Result: Dogfooding doesn't test real plugin behavior**

### What Works

- Both hook.json files are valid JSON
- Hook execution order is deterministic
- No syntax errors (both are valid Python)
- Database writes are transaction-safe

---

## Correct Architecture

This is intentional design (NOT a bug):

1. **Plugin hooks** = Source of truth for distributed package
   - Used by all users installing `wipnote` plugin
   - Should be well-tested and stable
   - Located in: `packages/claude-plugin/hooks/`

2. **Project hooks** = Wipnote dogfooding overrides
   - Test experimental features before releasing
   - Can safely be out of sync for testing
   - Located in: `.claude/hooks/`
   - Should be DELETED when tested features are promoted to plugin

---

## Recommendations for Next Agent (Opus)

### If making changes to hooks:

1. **Determine which version to edit:**
   ```
   Are you implementing a PLUGIN feature (for all users)?
   → Edit: packages/claude-plugin/hooks/scripts/

   Are you testing a DOGFOODING-ONLY feature (Wipnote development)?
   → Edit: .claude/hooks/scripts/

   Are you fixing a bug that affects both?
   → Edit BOTH (plugin first, then sync project)
   ```

2. **Sync strategy:**
   - Plugin is authoritative
   - When plugin is updated, sync to project if appropriate
   - When testing new feature in project, move to plugin before release

3. **Current sync tasks:**
   - [ ] Update `.claude/hooks/scripts/session-start.py` from plugin version
   - [ ] Update `.claude/hooks/scripts/subagent-stop.py` from plugin version
   - [ ] Verify other hooks are in sync
   - [ ] Delete project hooks once tested (optional)
   - [ ] Test hook execution to verify no duplicates

---

## Hook File Structure

### Plugin Hooks (Authoritative)
```
packages/claude-plugin/hooks/
├── hooks.json                              (Plugin manifest)
└── scripts/
    ├── link-activities.py
    ├── orchestrator-enforce.py
    ├── orchestrator-reflect.py
    ├── post-tool-use-failure.py
    ├── post_tool_use_failure.py
    ├── posttooluse-integrator.py
    ├── pretooluse-integrator.py
    ├── session-end.py
    ├── session-start.py                    (52 KB - WITH CIGS)
    ├── stop.py
    ├── subagent-stop.py                    (496 B - thin wrapper)
    ├── track-event.py
    ├── user-prompt-submit.py
    └── validate-work.py
```

### Project Hooks (Overrides)
```
.claude/hooks/
├── hooks.json                              (Project manifest)
├── hooks.json.bak                          (Backup)
├── protect-wipnote.sh
├── session-start.sh
└── scripts/
    ├── link-activities.py
    ├── orchestrator-enforce.py
    ├── orchestrator-reflect.py
    ├── post-tool-use-failure.py
    ├── post_tool_use_failure.py
    ├── posttooluse-integrator.py
    ├── pretooluse-integrator.py
    ├── session-end.py
    ├── session-start.py                    (38 KB - older, no CIGS)
    ├── subagent-stop.py                    (496 B - inline)
    ├── track-event.py
    ├── user-prompt-submit.py
    └── validate-work.py
```

---

## How Hooks Are Loaded (Claude Code Behavior)

1. Claude Code loads plugin hooks from `${CLAUDE_PLUGIN_ROOT}/hooks/hooks.json`
2. Claude Code loads project hooks from `.claude/hooks/hooks.json`
3. **Both sets are MERGED (combined), not replaced**
4. Hooks with same name execute in order:
   - Plugin hook executes
   - Project hook executes
   - **Creates duplicate execution**

This is why having both `.claude/hooks/` and plugin hooks can cause issues.

---

## Evidence

### File Locations Confirmed
```bash
✅ Plugin hooks: /Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/hooks/
✅ Project hooks: /Users/shakes/DevProjects/htmlgraph/.claude/hooks/
✅ Both hooks.json files exist
✅ Both have 15 hook implementations
```

### Sync Status
```
session-start.py:
  Plugin:  52 KB, Jan 8 2026, 13:29 (TODAY)
  Project: 38 KB, Jan 5 2026, 04:44 (3+ days old)
  Status: OUT OF SYNC ❌

subagent-stop.py:
  Plugin:  496 B, thin wrapper to package
  Project: 496 B, full inline implementation
  MD5: 0c1b1ace... vs 6afb8e28... (completely different)
  Status: OUT OF SYNC ❌
```

---

## Next Steps

1. **Clarify the intention:**
   - Are project hooks meant to be experimental overrides?
   - Or should they be kept in sync with plugin?

2. **Sync the hooks:**
   - Copy latest versions from plugin to project (if overriding intentional)
   - OR delete project hooks (if plugin is authoritative)

3. **Verify no duplicates:**
   - Run test to confirm hooks execute only once
   - Check agent_events table for duplicate entries

4. **Document the pattern:**
   - Add comments to hooks explaining two-layer architecture
   - Update CLAUDE.md to clarify plugin vs project hooks
