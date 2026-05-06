# Tool History Conflict - Fix Required

**Issue:** Two hooks write to the same file with incompatible formats
**Severity:** MEDIUM
**Priority:** LOW (doesn't block production use)

---

## Problem Description

Two git hooks maintain tool usage history in `/tmp/wipnote-tool-history.json`:

1. **orchestrator-enforce.py** - Tracks tool sequences for orchestrator enforcement
2. **validate-work.py** - Tracks tool patterns for anti-pattern detection

They use incompatible schemas, causing data corruption when both run.

---

## Current Behavior

**orchestrator-enforce.py format:**
```json
{
  "history": [
    {"tool": "Read", "timestamp": "2025-12-31T00:16:36+00:00"}
  ]
}
```

**validate-work.py format:**
```json
[
  {"tool": "Read", "ts": 1767140177.975007}
]
```

**Problem:**
When validate-work.py runs after orchestrator-enforce.py:
- Overwrites the file with its format
- Orchestrator loses sequence tracking data
- Second Read calls aren't blocked (should be blocked in strict mode)
- Tool history shows `{"tool": "", "ts": ...}` (empty tool name)

---

## Recommended Fix: Option A (Unified Format)

### Implementation

**1. Create shared utility module:**
```python
# src/python/wipnote/tool_history.py

from datetime import datetime, timezone
from pathlib import Path
import json

TOOL_HISTORY_FILE = Path("/tmp/wipnote-tool-history.json")
MAX_HISTORY_SIZE = 50

class ToolHistory:
    """Shared tool history tracker for git hooks."""

    @staticmethod
    def load() -> list[dict]:
        """Load tool history from disk."""
        if not TOOL_HISTORY_FILE.exists():
            return []

        try:
            data = json.loads(TOOL_HISTORY_FILE.read_text())
            # Support both formats during migration
            if isinstance(data, list):
                # Old format: [{"tool": "X", "ts": 123}]
                return [
                    {
                        "tool": item.get("tool", ""),
                        "timestamp": datetime.fromtimestamp(
                            item["ts"], tz=timezone.utc
                        ).isoformat()
                    }
                    for item in data
                ]
            elif isinstance(data, dict) and "history" in data:
                # New format: {"history": [...]}
                return data["history"]
            return []
        except Exception:
            return []

    @staticmethod
    def save(history: list[dict]) -> None:
        """Save tool history to disk."""
        try:
            recent = history[-MAX_HISTORY_SIZE:] if len(history) > MAX_HISTORY_SIZE else history
            TOOL_HISTORY_FILE.write_text(
                json.dumps({"history": recent}, indent=2)
            )
        except Exception:
            pass  # Fail silently

    @staticmethod
    def add(tool: str) -> None:
        """Add a tool to history."""
        history = ToolHistory.load()
        history.append({
            "tool": tool,
            "timestamp": datetime.now(timezone.utc).isoformat()
        })
        ToolHistory.save(history)

    @staticmethod
    def get_recent(tool: str, limit: int = 3) -> int:
        """Count recent uses of a specific tool."""
        history = ToolHistory.load()
        recent = history[-limit:] if len(history) > limit else history
        return sum(1 for h in recent if h.get("tool") == tool)
```

**2. Update orchestrator-enforce.py:**
```python
# Replace lines 55-106 with:
try:
    from wipnote.tool_history import ToolHistory
except Exception:
    # Fallback if module not available
    class ToolHistory:
        @staticmethod
        def load(): return []
        @staticmethod
        def save(h): pass
        @staticmethod
        def add(t): pass
        @staticmethod
        def get_recent(t, l=3): return 0

# Replace lines 144-159 with:
if tool in ["Read", "Grep", "Glob"]:
    recent_same_tool = ToolHistory.get_recent(tool, limit=3)

    if recent_same_tool == 0:  # First use
        return True, "Single lookup allowed", "single-lookup"
    else:
        return (
            False,
            f"Multiple {tool} calls detected. This is exploration work.\n\n"
            f"Delegate to Explorer subagent using Task tool.",
            "multi-lookup-blocked"
        )

# Replace line 308 with:
ToolHistory.add(tool)
```

**3. Update validate-work.py:**
```python
# Replace lines 71-106 with:
try:
    from wipnote.tool_history import ToolHistory
except Exception:
    # Fallback if module not available
    class ToolHistory:
        @staticmethod
        def load(): return []
        @staticmethod
        def add(t): pass

# Replace record_tool function with:
def record_tool(tool: str) -> None:
    """Record a tool use in history."""
    ToolHistory.add(tool)

# Update detect_anti_pattern to use ToolHistory.load()
```

**4. Add tests:**
```python
# tests/python/test_tool_history.py

def test_tool_history_unified_format():
    """Test tool history uses unified format."""
    from wipnote.tool_history import ToolHistory

    # Clear history
    TOOL_HISTORY_FILE = Path("/tmp/wipnote-tool-history.json")
    if TOOL_HISTORY_FILE.exists():
        TOOL_HISTORY_FILE.unlink()

    # Add tool
    ToolHistory.add("Read")

    # Check format
    data = json.loads(TOOL_HISTORY_FILE.read_text())
    assert "history" in data
    assert len(data["history"]) == 1
    assert data["history"][0]["tool"] == "Read"
    assert "timestamp" in data["history"][0]

def test_tool_history_migration():
    """Test migration from old format to new format."""
    from wipnote.tool_history import ToolHistory

    # Write old format
    old_data = [{"tool": "Read", "ts": 1767140177.975007}]
    TOOL_HISTORY_FILE.write_text(json.dumps(old_data))

    # Load should migrate
    history = ToolHistory.load()
    assert len(history) == 1
    assert history[0]["tool"] == "Read"
    assert "timestamp" in history[0]
```

---

## Alternative Fixes

### Option B: Separate History Files

**Pros:**
- No migration needed
- Each hook independent
- Quick to implement

**Cons:**
- Duplication of tool tracking
- More files to manage
- Harder to correlate events

**Implementation:**
```python
# orchestrator-enforce.py
TOOL_HISTORY_FILE = Path("/tmp/wipnote-orchestrator-history.json")

# validate-work.py
TOOL_HISTORY_FILE = Path("/tmp/wipnote-validate-history.json")
```

### Option C: Deprecate validate-work.py

**Pros:**
- Removes conflict completely
- Simplifies codebase
- Orchestrator hook is more comprehensive

**Cons:**
- Loses anti-pattern detection
- May need to port features to orchestrator hook

**Decision Required:**
- Is validate-work.py still needed?
- Can orchestrator-enforce.py handle all validation?

---

## Testing Plan

1. **Unit Tests:**
   - Test ToolHistory.load() with both formats
   - Test ToolHistory.save() produces correct format
   - Test ToolHistory.add() appends correctly
   - Test ToolHistory.get_recent() counts correctly

2. **Integration Tests:**
   - Run orchestrator-enforce.py, verify history format
   - Run validate-work.py, verify history format
   - Run both hooks sequentially, verify no corruption
   - Test multiple Read calls are correctly blocked

3. **Migration Tests:**
   - Start with old format file
   - Load with new ToolHistory
   - Verify migration to new format
   - Verify data preserved

---

## Migration Strategy

1. **Phase 1:** Create ToolHistory utility module
2. **Phase 2:** Update orchestrator-enforce.py to use it
3. **Phase 3:** Update validate-work.py to use it
4. **Phase 4:** Add migration tests
5. **Phase 5:** Deploy and monitor

**Timeline:** 2-4 hours
**Risk:** LOW (fallback to old behavior if module unavailable)

---

## Acceptance Criteria

- ✅ Both hooks use unified format
- ✅ No data corruption when both hooks run
- ✅ Multiple Read calls correctly blocked in strict mode
- ✅ Old format files automatically migrated
- ✅ All existing tests still pass
- ✅ New tests cover tool history scenarios

---

## Status

- **Documented:** ✅ 2025-12-30
- **Implemented:** ⏳ Pending
- **Tested:** ⏳ Pending
- **Deployed:** ⏳ Pending

**Owner:** TBD
**Priority:** LOW (doesn't block production use)
**Effort:** 2-4 hours
