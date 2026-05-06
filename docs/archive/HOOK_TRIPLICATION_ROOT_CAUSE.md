# HOOK TRIPLICATION ROOT CAUSE - Complete Investigation

## Problem Statement
"UserPromptSubmit hook success: Success" appears **3 times** in system reminders instead of once, indicating the hook is executing 3 times per user prompt.

---

## Root Cause: THREE SIMULTANEOUS HOOK REGISTRATIONS

Claude Code loads and **MERGES** hooks from **3 different locations**:

| # | Location | Status | Content |
|---|----------|--------|---------|
| 1 | `/Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/.claude-plugin/hooks/hooks.json` | ✅ CORRECT SOURCE | Single UserPromptSubmit hook |
| 2 | `/Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/hooks/hooks.json` | ❌ DUPLICATE FILE | Identical copy of Source 1 |
| 3 | `/Users/shakes/.claude/plugins/marketplaces/wipnote/packages/claude-plugin/.claude-plugin/hooks/hooks.json` | ⚠️ CACHED/OUTDATED | Includes extra track-event.py hook |

When Claude Code encounters the same hook type from multiple sources, it **merges the hooks** instead of deduplicating them. Each UserPromptSubmit registration executes independently, resulting in 3 executions.

---

## ALL HOOK SOURCES DISCOVERED

### Hook Source #1: PRIMARY PLUGIN SOURCE (DEVELOPMENT)
**Path:** `/Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/.claude-plugin/hooks/hooks.json`

**Content:**
```json
{
  "hooks": {
    "UserPromptSubmit": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "comment": "CIGS analysis and workflow guidance",
            "command": "uv run \"${CLAUDE_PLUGIN_ROOT}/hooks/scripts/user-prompt-submit.py\""
          }
        ]
      }
    ],
    "PreToolUse": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "comment": "Route Task() calls to spawner agents (gemini, codex, copilot)",
            "command": "uv run \"${CLAUDE_PLUGIN_ROOT}/hooks/scripts/pretooluse-integrator.py\""
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "uv run \"${CLAUDE_PLUGIN_ROOT}/hooks/scripts/posttooluse-integrator.py\""
          }
        ]
      }
    ],
    ...
  }
}
```

**Status:** ✅ CORRECT - This is the source of truth

---

### Hook Source #2: DUPLICATE IN ALTERNATE DIRECTORY (DELETE THIS)
**Path:** `/Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/hooks/hooks.json`

**Problem:** IDENTICAL to Source #1 - causes hook duplication

**Lines 3-13:** UserPromptSubmit - DUPLICATE
**Lines 48-57:** PreToolUse - DUPLICATE
**Lines 60-68:** PostToolUse - DUPLICATE

**Status:** ❌ DELETE - This file should not exist

**Why it exists:** At some point both `packages/claude-plugin/hooks/hooks.json` and `packages/claude-plugin/.claude-plugin/hooks/hooks.json` were created and both committed to git. Claude Code's plugin loader scans for both patterns and **merges** them.

---

### Hook Source #3: GLOBAL MARKETPLACE CACHE (OUTDATED)
**Path:** `/Users/shakes/.claude/plugins/marketplaces/wipnote/packages/claude-plugin/.claude-plugin/hooks/hooks.json`

**Problem:** Contains DIFFERENT hooks than current source

**UserPromptSubmit (Lines 3-17):**
```json
"UserPromptSubmit": [
  {
    "matcher": "",
    "hooks": [
      {
        "type": "command",
        "comment": "Record UserQuery event to SQLite",
        "command": "HTMLGRAPH_HOOK_TYPE=UserPromptSubmit uv run \"${CLAUDE_PLUGIN_ROOT}/hooks/scripts/track-event.py\""
      },
      {
        "type": "command",
        "comment": "CIGS analysis and workflow guidance",
        "command": "uv run \"${CLAUDE_PLUGIN_ROOT}/hooks/scripts/user-prompt-submit.py\""
      }
    ]
  }
]
```

**Current Source only has:** Single hook (user-prompt-submit.py)

**PreToolUse (Lines 53-68):**
```json
"PreToolUse": [
  {
    "matcher": "",
    "hooks": [
      {
        "type": "command",
        "comment": "Route Task() calls to spawner agents (gemini, codex, copilot)",
        "command": "uv run \"${CLAUDE_PLUGIN_ROOT}/hooks/scripts/pretooluse-integrator.py\""
      },
      {
        "type": "command",
        "comment": "Route Task() calls to spawner agents (gemini, codex, copilot)",
        "command": "uv run \"${CLAUDE_PLUGIN_ROOT}/hooks/scripts/pretooluse-spawner-router.py\""
      }
    ]
  }
]
```

**Current Source only has:** Single hook (pretooluse-integrator.py)

**Status:** ⚠️ OUTDATED - Will auto-update on next deployment

---

## HOW CLAUDE CODE LOADS PLUGINS

```
1. Claude Code starts session
   ↓
2. Scans for local plugins in current directory
   └─ Finds: packages/claude-plugin/
   ↓
3. Checks for hook configurations (OLD & NEW patterns):
   ├─ packages/claude-plugin/hooks/hooks.json (OLD pattern) ← FOUND
   └─ packages/claude-plugin/.claude-plugin/hooks/hooks.json (NEW pattern) ← FOUND
   ↓
4. MERGES hooks from both files
   ├─ Hook 1: From packages/claude-plugin/hooks/hooks.json
   └─ Hook 2: From packages/claude-plugin/.claude-plugin/hooks/hooks.json
   ↓
5. Also loads from global marketplace cache:
   └─ ~/.claude/plugins/marketplaces/wipnote/...
   ↓
6. MERGES all three sources together
   ↓
7. When UserPromptSubmit event fires:
   ├─ Execution 1: Run hook from packages/claude-plugin/hooks/hooks.json
   ├─ Execution 2: Run hook from packages/claude-plugin/.claude-plugin/hooks/hooks.json
   └─ Execution 3: Run hook from ~/.claude/plugins/marketplaces/...
   ↓
Result: user-prompt-submit.py runs 3 TIMES
```

---

## EXACT FILES AND LINE NUMBERS

### Files Causing Triplication

| File | Hook Type | Lines | Status | Action |
|------|-----------|-------|--------|--------|
| `/Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/hooks/hooks.json` | UserPromptSubmit | 3-13 | ❌ DUPLICATE | DELETE |
| `/Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/hooks/hooks.json` | PreToolUse | 48-57 | ❌ DUPLICATE | DELETE |
| `/Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/hooks/hooks.json` | PostToolUse | 60-68 | ❌ DUPLICATE | DELETE |
| `/Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/.claude-plugin/hooks/hooks.json` | UserPromptSubmit | 3-13 | ✅ CORRECT | KEEP |
| `/Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/.claude-plugin/hooks/hooks.json` | PreToolUse | 48-57 | ✅ CORRECT | KEEP |
| `/Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/.claude-plugin/hooks/hooks.json` | PostToolUse | 60-68 | ✅ CORRECT | KEEP |
| `~/.claude/plugins/marketplaces/wipnote/packages/claude-plugin/.claude-plugin/hooks/hooks.json` | UserPromptSubmit | 3-17 | ⚠️ OUTDATED | AUTO-FIX |
| `~/.claude/plugins/marketplaces/wipnote/packages/claude-plugin/.claude-plugin/hooks/hooks.json` | PreToolUse | 53-68 | ⚠️ OUTDATED | AUTO-FIX |

---

## VERIFICATION: Before and After

### Before (Current - Broken)
```bash
# Count UserPromptSubmit registrations across all sources
find /Users/shakes -path "*wipnote*" -name "hooks.json" -type f -exec grep -l "UserPromptSubmit" {} \; 2>/dev/null | wc -l
# Output: 3 (three files have UserPromptSubmit)

# Test: Submit a prompt in Claude Code
# Observe: "UserPromptSubmit hook success: Success" appears 3 times
```

### After (Fixed)
```bash
# After deleting packages/claude-plugin/hooks/hooks.json
# After deploying to update marketplace cache
find /Users/shakes -path "*wipnote*" -name "hooks.json" -type f -exec grep -l "UserPromptSubmit" {} \; 2>/dev/null | wc -l
# Output: 2 (only two remain - source and marketplace cache)
# But since marketplace cache syncs from source, effectively 1

# Test: Submit a prompt in Claude Code
# Observe: "UserPromptSubmit hook success: Success" appears only 1 time ✓
```

---

## SOLUTION: 3-STEP FIX

### Step 1: Delete Duplicate Hook File (1 minute)
```bash
rm /Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/hooks/hooks.json
```

**Verification:**
```bash
ls -la /Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/ | grep hooks.json
# Should return nothing (or only .claude-plugin/hooks/)
```

### Step 2: Test Hook Execution (2 minutes)
1. Submit any prompt in Claude Code
2. Check system reminders
3. Should see only 1 "UserPromptSubmit hook success" message (not 3)

### Step 3: Deploy to Fix Marketplace Cache (5 minutes)
```bash
# Run tests first
uv run pytest

# Deploy to update marketplace version
./scripts/deploy-all.sh 0.26.1 --no-confirm
```

---

## Summary of All Hook Sources

```
Project Root: /Users/shakes/DevProjects/htmlgraph/

Hook Registration Sources:
├── ✅ packages/claude-plugin/.claude-plugin/hooks/hooks.json (SOURCE - KEEP)
├── ❌ packages/claude-plugin/hooks/hooks.json (DUPLICATE - DELETE)
└── ⚠️ ~/.claude/plugins/marketplaces/wipnote/packages/claude-plugin/.claude-plugin/hooks/hooks.json (CACHE - AUTO-FIXES)

Other Cache Locations (Safe to ignore, auto-cleaned):
├── ~/.claude/plugins/cache/wipnote/wipnote/0.24.1/hooks/hooks.json
├── ~/.claude/plugins/cache/wipnote/wipnote/0.9.6/hooks/hooks.json
└── ~/.claude/plugins/cache/wipnote/wipnote/0.25.0/hooks/hooks.json
```

---

## Why This Happened

1. **Initial design** used two hook file patterns (old and new)
2. **Both patterns** were implemented and committed to git
3. **Claude Code** scans for both patterns and merges them
4. **No cleanup** happened when switching from old pattern to new pattern
5. **Result:** Duplicate hooks execute whenever the event fires
6. **Marketplace cache** captured the duplicates when package was published
7. **Triplication** = source + duplicate + marketplace cache

---

## Key Files Referenced

| File | Purpose | Location |
|------|---------|----------|
| `.claude-plugin/hooks/hooks.json` | Source of truth for hooks | `/Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/.claude-plugin/hooks/hooks.json` |
| `hooks/hooks.json` | DUPLICATE - DELETE | `/Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/hooks/hooks.json` |
| Marketplace cache | Auto-synced from source | `~/.claude/plugins/marketplaces/wipnote/...` |
| Hook scripts | Actual executable scripts | `/Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/.claude-plugin/hooks/scripts/` |

---

## Related Issues Discovered

**Also found: PreToolUse duplication in marketplace cache**

The marketplace version has an extra `pretooluse-spawner-router.py` hook registered alongside `pretooluse-integrator.py`. Current source only has `pretooluse-integrator.py`. This will auto-fix on next deploy.

---

## Absolute File Paths (For Reference)

**DELETE:**
```
/Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/hooks/hooks.json
```

**KEEP (Source of Truth):**
```
/Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/.claude-plugin/hooks/hooks.json
```

**AUTO-UPDATES (Marketplace Cache):**
```
/Users/shakes/.claude/plugins/marketplaces/wipnote/packages/claude-plugin/.claude-plugin/hooks/hooks.json
/Users/shakes/.claude/plugins/marketplaces/wipnote/packages/claude-plugin/hooks/hooks.json
```

---

## Project Structure Best Practice

**Correct plugin structure:**
```
packages/claude-plugin/
├── .claude-plugin/          ← OFFICIAL PLUGIN SOURCE
│   ├── hooks/
│   │   ├── hooks.json       ← Single source of truth
│   │   └── scripts/
│   ├── agents/
│   ├── skills/
│   └── plugin.json
├── hooks/                   ← DEPRECATED (can be deleted)
└── README.md
```

The `.claude-plugin/` directory is the standard Claude Code plugin format. Alternate `hooks/` directory should be removed to avoid merging.
