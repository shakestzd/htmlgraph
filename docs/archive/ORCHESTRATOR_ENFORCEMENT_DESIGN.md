# Orchestrator Enforcement System - Design Document

## Problem Statement

**Current Issue**: Even when the orchestrator skill is activated, Claude continues to execute implementation work directly (Read, Edit, Write, Bash) instead of delegating to subagents via the Task tool.

**Root Causes**:
1. SessionStart hook's orchestrator directives aren't being injected into Claude's system prompt (Claude Code bug)
2. Skill activation is passive - provides guidance but doesn't enforce behavior
3. No mechanism to distinguish orchestrator operations from implementation
4. Claude's natural tendency is to execute directly, not delegate

## Design Goals

1. **Enforce orchestrator delegation** - Block implementation when in orchestrator mode
2. **Allow lightweight operations** - Orchestrators can do quick lookups and work item management
3. **Clear activation** - Explicit way to enter/exit orchestrator mode
4. **Smart detection** - Distinguish between orchestrator ops and implementation
5. **Graceful fallback** - If hook fails, default to allowing operations

## Solution Architecture

### Component 1: Orchestrator Mode State Management

**File**: `.wipnote/orchestrator-mode.json`

```json
{
  "enabled": true,
  "activated_at": "2025-12-30T08:00:00Z",
  "session_id": "sess-abc123",
  "enforcement_level": "strict"  // or "guidance"
}
```

**Activation Methods**:
1. SessionStart hook sets `enabled: true` when features exist
2. Skill activation (`/wipnote:orchestrator`) sets `enabled: true`
3. User can disable: `uv run wipnote orchestrator disable`
4. Auto-disables when no features exist

### Component 2: Smart PreToolUse Hook

**File**: `packages/claude-plugin/hooks/scripts/orchestrator-enforce.py`

**Hook Logic**:

```python
def is_orchestrator_mode_active() -> bool:
    """Check if orchestrator mode is enabled."""
    mode_file = Path(".wipnote/orchestrator-mode.json")
    if mode_file.exists():
        data = json.loads(mode_file.read_text())
        return data.get("enabled", False)
    return False

def get_enforcement_level() -> str:
    """Get enforcement level: strict or guidance."""
    mode_file = Path(".wipnote/orchestrator-mode.json")
    if mode_file.exists():
        data = json.loads(mode_file.read_text())
        return data.get("enforcement_level", "guidance")
    return "guidance"

def is_allowed_orchestrator_operation(tool: str, params: dict) -> tuple[bool, str]:
    """
    Check if operation is allowed for orchestrators.

    Returns: (is_allowed, reason_if_not)
    """
    # Category 1: ALWAYS ALLOWED - Orchestrator core operations
    if tool in ["Task", "AskUserQuestion", "TodoWrite"]:
        return True, ""

    # Category 2: SDK Operations - Always allowed
    if tool == "Bash":
        command = params.get("command", "")
        if command.startswith("uv run wipnote ") or command.startswith("wipnote "):
            return True, ""
        # Allow git status/diff (read-only)
        if command.startswith("git status") or command.startswith("git diff"):
            return True, ""

    # Category 3: Quick Lookups - Single operations only
    if tool in ["Read", "Grep", "Glob"]:
        # Check tool history to see if this is a single lookup or part of a sequence
        history = load_tool_history()
        recent_same_tool = sum(1 for h in history[-3:] if h["tool"] == tool)

        if recent_same_tool == 0:  # First use
            return True, "Single lookup allowed"
        else:
            return False, f"Multiple {tool} calls detected. Delegate to Explorer subagent using Task tool."

    # Category 4: Work Item Creation - Allowed via Python inline
    if tool == "Bash" and "from wipnote import SDK" in params.get("command", ""):
        return True, "SDK work item creation"

    # Category 5: BLOCKED - Implementation tools
    if tool in ["Edit", "Write", "NotebookEdit", "Delete"]:
        return False, f"{tool} is implementation work. Delegate to Coder subagent using Task tool."

    # Category 6: BLOCKED - Multiple file operations
    if tool == "Bash":
        command = params.get("command", "")
        # Block compilation, testing, building (should be in subagent)
        blocked_patterns = [
            r"^npm (run|test|build)",
            r"^pytest",
            r"^python -m pytest",
            r"^cargo (build|test)",
            r"^mvn (compile|test|package)",
        ]
        for pattern in blocked_patterns:
            if re.match(pattern, command):
                return False, "Testing/building should be delegated to subagent"

    # Default: Allow with guidance
    return True, "Allowed but consider if delegation would be better"

def enforce_orchestrator_mode(tool: str, params: dict) -> dict:
    """
    Enforce orchestrator mode rules.

    Returns: Hook response dict
    """
    # Check if orchestrator mode is active
    if not is_orchestrator_mode_active():
        return {"decision": "allow"}  # Not in orchestrator mode

    # Get enforcement level
    level = get_enforcement_level()

    # Check if operation is allowed
    is_allowed, reason = is_allowed_orchestrator_operation(tool, params)

    if is_allowed:
        if reason and level == "strict":
            # Provide guidance even when allowing
            return {
                "decision": "allow",
                "guidance": f"✅ {reason}"
            }
        return {"decision": "allow"}

    # Operation not allowed
    if level == "strict":
        # BLOCK the operation
        return {
            "decision": "block",
            "reason": f"🎯 ORCHESTRATOR MODE: {reason}\n\n"
                     f"Use: Task(prompt='...', subagent_type='general-purpose')\n\n"
                     f"To disable orchestrator mode: uv run wipnote orchestrator disable",
            "suggestion": create_task_suggestion(tool, params)
        }
    else:
        # GUIDANCE mode - allow but warn
        return {
            "decision": "allow",
            "guidance": f"⚠️ ORCHESTRATOR: {reason}",
            "suggestion": create_task_suggestion(tool, params)
        }

def create_task_suggestion(tool: str, params: dict) -> str:
    """Create Task tool suggestion based on blocked operation."""
    if tool in ["Edit", "Write", "NotebookEdit"]:
        return (
            "Task(\n"
            "    prompt='Implement [describe changes]',\n"
            "    subagent_type='general-purpose'\n"
            ")"
        )
    elif tool in ["Read", "Grep", "Glob"]:
        return (
            "Task(\n"
            "    prompt='Explore [describe what to find]',\n"
            "    subagent_type='Explore'\n"
            ")"
        )
    elif tool == "Bash" and "test" in params.get("command", ""):
        return (
            "Task(\n"
            "    prompt='Run tests and report results',\n"
            "    subagent_type='general-purpose'\n"
            ")"
        )
    else:
        return "Use Task tool to delegate this operation to a subagent"
```

### Component 3: Orchestrator Mode CLI Commands

**Add to wipnote CLI**:

```bash
# Enable orchestrator mode (strict)
uv run wipnote orchestrator enable

# Enable orchestrator mode (guidance only)
uv run wipnote orchestrator enable --level=guidance

# Disable orchestrator mode
uv run wipnote orchestrator disable

# Check status
uv run wipnote orchestrator status
```

### Component 4: SessionStart Hook Integration

**Modify `session-start.py`**:

```python
def activate_orchestrator_mode(graph_dir: Path, features: list) -> None:
    """Activate orchestrator mode if features exist."""
    mode_file = graph_dir / "orchestrator-mode.json"

    # Activate if features exist and mode not explicitly disabled
    if features:
        mode_data = {
            "enabled": True,
            "activated_at": datetime.now().isoformat(),
            "session_id": os.environ.get("CLAUDE_SESSION_ID", "unknown"),
            "enforcement_level": "strict",  # Default to strict
            "auto_activated": True
        }
        mode_file.write_text(json.dumps(mode_data, indent=2))
    elif mode_file.exists():
        # Deactivate if no features
        data = json.loads(mode_file.read_text())
        if data.get("auto_activated"):  # Only auto-deactivate if auto-activated
            data["enabled"] = False
            mode_file.write_text(json.dumps(data, indent=2))

# Call in main()
activate_orchestrator_mode(graph_dir, features)
```

### Component 5: Skill Activation Integration

**Modify orchestrator skill to activate mode**:

Add to skill.md header:

```markdown
**IMPORTANT**: This skill automatically activates orchestrator enforcement mode.

When activated:
- ✅ Task delegation is REQUIRED for implementation
- ✅ Quick lookups (single Read/Grep) are allowed
- ✅ Work item management is allowed
- ❌ Direct Edit/Write operations are BLOCKED

To disable: `uv run wipnote orchestrator disable`
```

## Operation Classification Matrix

| Tool | Operation | Orchestrator Allowed | Reasoning |
|------|-----------|---------------------|-----------|
| Task | Any | ✅ ALWAYS | Core orchestration tool |
| AskUserQuestion | Any | ✅ ALWAYS | Decision making |
| TodoWrite | Any | ✅ ALWAYS | Progress tracking |
| Bash | `uv run wipnote ...` | ✅ ALWAYS | SDK operations |
| Bash | `git status/diff` | ✅ ALWAYS | Read-only status |
| Bash | `from wipnote ...` | ✅ ALWAYS | Work item creation |
| Read | First use | ✅ SINGLE | Quick lookup allowed |
| Read | 2+ in sequence | ❌ DELEGATE | Explorer subagent |
| Grep | First use | ✅ SINGLE | Quick search allowed |
| Grep | 2+ in sequence | ❌ DELEGATE | Explorer subagent |
| Glob | First use | ✅ SINGLE | Quick file find |
| Glob | 2+ in sequence | ❌ DELEGATE | Explorer subagent |
| Edit | Any | ❌ DELEGATE | Coder subagent |
| Write | Any | ❌ DELEGATE | Coder subagent |
| NotebookEdit | Any | ❌ DELEGATE | Coder subagent |
| Delete | Any | ❌ DELEGATE | Coder subagent |
| Bash | `npm test/build` | ❌ DELEGATE | Testing subagent |
| Bash | `pytest` | ❌ DELEGATE | Testing subagent |

## Enforcement Levels

### Strict Mode (Default)
- **BLOCKS** implementation operations
- Provides clear error message with Task suggestion
- Enforces delegation pattern

### Guidance Mode
- **ALLOWS** all operations
- Provides warnings and suggestions
- Relies on Claude following guidance

## Activation Flow

```
Session Start
    ↓
Check for features
    ↓
Features exist? → YES → Activate orchestrator mode (strict)
    ↓                    Write .wipnote/orchestrator-mode.json
    ↓
PreToolUse hook runs
    ↓
Check orchestrator-mode.json
    ↓
Mode active? → YES → Enforce rules
    ↓                  Block/Guide based on level
    ↓
Allow operation
```

## Escape Hatches

1. **Disable for emergency fixes**:
   ```bash
   uv run wipnote orchestrator disable
   ```

2. **Guidance mode for learning**:
   ```bash
   uv run wipnote orchestrator enable --level=guidance
   ```

3. **Per-session override**:
   ```bash
   HTMLGRAPH_ORCHESTRATOR_DISABLED=1 claude
   ```

## Testing Strategy

1. **Unit Tests** - Test each operation classification
2. **Integration Tests** - Test full enforcement flow
3. **Edge Cases** - Test mode transitions, hook failures
4. **User Experience** - Ensure clear error messages

## Migration Plan

1. **Phase 1**: Implement mode state management (orchestrator-mode.json)
2. **Phase 2**: Implement PreToolUse hook (orchestrator-enforce.py)
3. **Phase 3**: Add CLI commands
4. **Phase 4**: Integrate with SessionStart hook
5. **Phase 5**: Update orchestrator skill
6. **Phase 6**: Deploy with guidance mode default (soft launch)
7. **Phase 7**: Switch to strict mode default (after validation)

## Success Metrics

- ✅ Claude consistently delegates implementation work
- ✅ Quick lookups still work (single Read/Grep)
- ✅ Work item creation unblocked
- ✅ Clear error messages when blocking
- ✅ Users can disable when needed
- ✅ No false positives (blocking legitimate operations)

## Open Questions

1. Should orchestrator mode persist across sessions or reset each time?
   - **Decision**: Persist, but allow manual disable

2. How to handle mixed workflows (orchestrator + direct work)?
   - **Decision**: Provide guidance mode for flexibility

3. What if user explicitly requests direct implementation?
   - **Decision**: Provide clear disable command in error message

4. Should we track delegation compliance metrics?
   - **Decision**: Yes, add to analytics (future enhancement)
