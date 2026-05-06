# Hook Triplication Investigation - Complete Index

## Quick Reference

**Problem:** "UserPromptSubmit hook success: Success" appears 3 times
**Root Cause:** Claude Code merges hooks from 3 different locations
**Solution:** Delete `/Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/hooks/hooks.json`

---

## Investigation Documents

### 1. Main Report: HOOK_TRIPLICATION_ROOT_CAUSE.md
**Location:** `/Users/shakes/DevProjects/htmlgraph/HOOK_TRIPLICATION_ROOT_CAUSE.md`

**Contains:**
- Complete problem statement
- Root cause analysis (3 simultaneous hook registrations)
- All hook sources with exact file paths and line numbers
- How Claude Code loads and merges plugins
- Hook execution flow diagram
- 3-step solution
- Before/after verification procedures
- Related issues discovered
- Why this happened (history)
- Best practices for plugin structure

**Read this for:** Full technical understanding of the issue

---

## Hook Sources Identified

### Source 1: PRIMARY PLUGIN (CORRECT) ✅
```
/Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/.claude-plugin/hooks/hooks.json
```
- Lines 3-13: UserPromptSubmit (1 hook)
- Lines 48-57: PreToolUse (1 hook)
- Lines 60-68: PostToolUse (1 hook)
- **Action:** KEEP - This is the source of truth

### Source 2: DUPLICATE (DELETE) ❌
```
/Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/hooks/hooks.json
```
- Lines 3-13: UserPromptSubmit (1 hook - DUPLICATE)
- Lines 48-57: PreToolUse (1 hook - DUPLICATE)
- Lines 60-68: PostToolUse (1 hook - DUPLICATE)
- **Action:** DELETE - Causes the triplication

### Source 3: MARKETPLACE CACHE (AUTO-UPDATE) ⚠️
```
/Users/shakes/.claude/plugins/marketplaces/wipnote/packages/claude-plugin/.claude-plugin/hooks/hooks.json
```
- Lines 3-17: UserPromptSubmit (2 hooks - outdated)
- Lines 53-68: PreToolUse (2 hooks - outdated with extra spawner-router.py)
- Lines 70-84: PostToolUse (2 hooks - outdated)
- **Action:** AUTO-FIXES on next deployment

---

## Exact Files to Delete

### File to Delete
```bash
/Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/hooks/hooks.json
```

### Verification Command
```bash
ls -la /Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/ | grep hooks.json
# Should show only:
# .claude-plugin/hooks/
# Should NOT show:
# hooks/hooks.json
```

---

## How the Triplication Happens

```
User submits prompt
        ↓
  UserPromptSubmit event fires
        ↓
  Claude Code Hooks Merger loads from 3 sources:
        ├─ Source 1: packages/claude-plugin/.claude-plugin/hooks/hooks.json
        ├─ Source 2: packages/claude-plugin/hooks/hooks.json (DUPLICATE)
        └─ Source 3: ~/.claude/plugins/marketplaces/wipnote/.../hooks.json
        ↓
  Each source registers the same UserPromptSubmit hook
        ↓
  Hook runs 3 times independently:
        ├─ Execution 1: user-prompt-submit.py from Source 1
        ├─ Execution 2: user-prompt-submit.py from Source 2
        └─ Execution 3: user-prompt-submit.py from Source 3
        ↓
  Result: "UserPromptSubmit hook success: Success" appears 3 times
```

---

## Solution Steps

### Step 1: Delete Duplicate File (1 min)
```bash
rm /Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/hooks/hooks.json
```

### Step 2: Test (2 min)
1. Submit a prompt in Claude Code
2. Check system reminders
3. Should see only 1 "UserPromptSubmit hook success" (not 3)

### Step 3: Deploy (5 min)
```bash
uv run pytest
./scripts/deploy-all.sh 0.26.1 --no-confirm
```

---

## Related Issues Discovered

### PreToolUse Duplication in Marketplace Cache
**File:** `/Users/shakes/.claude/plugins/marketplaces/wipnote/packages/claude-plugin/.claude-plugin/hooks/hooks.json`

**Issue:** Lines 53-68 have 2 PreToolUse hooks:
- pretooluse-integrator.py
- pretooluse-spawner-router.py (EXTRA - not in current source)

**Current source** only has `pretooluse-integrator.py`

**Fix:** Will auto-update on next deployment

---

## Project Structure Best Practice

### Correct Plugin Structure
```
packages/claude-plugin/
├── .claude-plugin/                    ← OFFICIAL PLUGIN SOURCE
│   ├── hooks/
│   │   ├── hooks.json                ← Single source of truth
│   │   └── scripts/
│   │       ├── user-prompt-submit.py
│   │       ├── track-event.py
│   │       └── ...
│   ├── agents/
│   ├── skills/
│   └── plugin.json
├── README.md
└── (no duplicate hooks/ directory!)
```

### Why This Structure
- `.claude-plugin/` is the standard Claude Code plugin format
- Single source of truth prevents duplication
- Alternate `hooks/` directory should not exist
- Prevents hook merging behavior

---

## All Hook Files in Project

| File | Status | Action |
|------|--------|--------|
| `/Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/.claude-plugin/hooks/hooks.json` | ✅ Correct | KEEP |
| `/Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/hooks/hooks.json` | ❌ Duplicate | DELETE |
| `~/.claude/plugins/marketplaces/wipnote/packages/claude-plugin/.claude-plugin/hooks/hooks.json` | ⚠️ Cache | AUTO-UPDATE |
| `~/.claude/plugins/marketplaces/wipnote/packages/claude-plugin/hooks/hooks.json` | ⚠️ Cache | AUTO-UPDATE |
| `~/.claude/plugins/cache/wipnote/wipnote/0.24.1/hooks/hooks.json` | 🗑️ Old cache | Ignore |
| `~/.claude/plugins/cache/wipnote/wipnote/0.9.6/hooks/hooks.json` | 🗑️ Old cache | Ignore |
| `~/.claude/plugins/cache/wipnote/wipnote/0.25.0/hooks/hooks.json` | 🗑️ Old cache | Ignore |

---

## How This Happened

1. Two hook file patterns were implemented (old: `hooks/`, new: `.claude-plugin/`)
2. Both were created and committed to git
3. Claude Code plugin loader scans for both patterns
4. When both exist, Claude Code **MERGES** them instead of choosing one
5. Each registration runs independently
6. Marketplace cache captured the merged state when package was published
7. Result: 3 separate hook registrations = 3x execution

---

## Verification: Before and After

### Before (Current - Broken)
```bash
# Count UserPromptSubmit across all sources
grep -r "UserPromptSubmit" /Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/hooks/hooks.json
# Output: Found

grep -r "UserPromptSubmit" /Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/.claude-plugin/hooks/hooks.json
# Output: Found

grep -r "UserPromptSubmit" ~/.claude/plugins/marketplaces/wipnote/packages/claude-plugin/.claude-plugin/hooks/hooks.json
# Output: Found

# Total: 3 sources = 3x execution
```

### After (Fixed)
```bash
# Count UserPromptSubmit across remaining sources
grep -r "UserPromptSubmit" /Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/hooks/hooks.json
# Output: File not found

grep -r "UserPromptSubmit" /Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/.claude-plugin/hooks/hooks.json
# Output: Found

# After redeploy, marketplace cache syncs from source

# Total: 1 source = 1x execution (correct)
```

---

## Testing Hook Execution

### Test 1: Manual Verification
1. Open Claude Code
2. Submit any prompt
3. Look at system reminders
4. **Before fix:** See "UserPromptSubmit hook success: Success" 3 times
5. **After fix:** See it only 1 time

### Test 2: Check Hook Output
```bash
# Run a specific prompt that triggers UserPromptSubmit
# Watch for hook execution count in logs
```

---

## Command Reference

### Delete duplicate
```bash
rm /Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/hooks/hooks.json
```

### Verify deletion
```bash
test ! -f /Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/hooks/hooks.json && echo "Deleted successfully" || echo "File still exists"
```

### Check hook files
```bash
find /Users/shakes/DevProjects/htmlgraph/packages/claude-plugin -name "hooks.json" -type f
```

### Redeploy
```bash
cd /Users/shakes/DevProjects/htmlgraph
uv run pytest
./scripts/deploy-all.sh 0.26.1 --no-confirm
```

---

## Summary

| Aspect | Details |
|--------|---------|
| **Problem** | UserPromptSubmit hook executes 3 times instead of 1 |
| **Root Cause** | 3 hook registration sources are loaded and merged |
| **Primary Issue** | `/packages/claude-plugin/hooks/hooks.json` is a duplicate |
| **Quick Fix** | Delete the duplicate file |
| **Verification** | Submit prompt, check system reminders (should show 1x not 3x) |
| **Full Fix** | Deploy to update marketplace cache |
| **Prevention** | Keep hooks only in `.claude-plugin/`, delete alternate locations |

---

## Files in This Investigation

1. **HOOK_TRIPLICATION_ROOT_CAUSE.md** - Complete technical analysis
2. **HOOK_TRIPLICATION_INVESTIGATION_INDEX.md** - This file (quick reference)

---

## Next Steps

1. Read: `/Users/shakes/DevProjects/htmlgraph/HOOK_TRIPLICATION_ROOT_CAUSE.md`
2. Execute: `rm /Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/hooks/hooks.json`
3. Test: Submit prompt, verify 1x execution
4. Deploy: `./scripts/deploy-all.sh 0.26.1 --no-confirm`

---

**Investigation Complete:** All hook sources identified, root cause determined, solution provided.
