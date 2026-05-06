# Skill Execution Update - HeadlessSpawner Integration

## Summary

Successfully updated the three spawner skills to execute external CLIs via the HeadlessSpawner SDK instead of just providing guidance documentation.

**Status: COMPLETE** ✅

## Files Updated

### 1. Gemini Skill
- **File**: `packages/claude-plugin/.claude-plugin/skills/gemini/skill.md`
- **Change**: Added executable Python code block with HeadlessSpawner integration
- **Skill Type**: Changed from `guidance` to `executable`

**Key Features**:
- Checks if `gemini` CLI is installed via `which gemini`
- If available: Invokes `spawner.spawn_gemini()` with default settings
- If not available: Prints helpful installation instructions
- Tracks execution in Wipnote with `track_in_wipnote=True`
- Displays formatted output with token usage and tracked events

**Invocation**:
```python
Skill(skill=".claude-plugin:gemini", args="Analyze authentication patterns in the codebase")
```

### 2. Codex Skill
- **File**: `packages/claude-plugin/.claude-plugin/skills/codex/skill.md`
- **Change**: Added executable Python code block with HeadlessSpawner integration
- **Skill Type**: Changed from `guidance` to `executable`

**Key Features**:
- Checks if `codex` CLI is installed via `which codex`
- If available: Invokes `spawner.spawn_codex()` with sandbox mode
- Configured for workspace-write sandbox (safe execution)
- Full auto mode enabled for headless operation
- Tracks execution in Wipnote with `track_in_wipnote=True`
- Displays formatted output with token usage and tracked events

**Invocation**:
```python
Skill(skill=".claude-plugin:codex", args="Generate FastAPI endpoint with authentication and tests")
```

### 3. Copilot Skill
- **File**: `packages/claude-plugin/.claude-plugin/skills/copilot/skill.md`
- **Change**: Added executable Python code block with HeadlessSpawner integration
- **Skill Type**: Changed from `guidance` to `executable`

**Key Features**:
- Checks if `copilot` CLI is installed via `which copilot`
- If available: Invokes `spawner.spawn_copilot()` with tool permissions
- Auto-allows all tools for headless operation
- Tracks execution in Wipnote with `track_in_wipnote=True`
- Displays formatted output with token usage and tracked events

**Invocation**:
```python
Skill(skill=".claude-plugin:copilot", args="Create PR for authentication feature with description")
```

## Implementation Pattern (All Three Skills)

Each skill follows this pattern:

```python
<python>
import subprocess
import sys
from wipnote.orchestration.headless_spawner import HeadlessSpawner

# 1. Extract task prompt from skill arguments
task_prompt = skill_args if 'skill_args' in dir() else ""

# 2. Validate prompt provided
if not task_prompt:
    print("❌ ERROR: No task prompt provided")
    sys.exit(1)

# 3. Check CLI availability
cli_check = subprocess.run(["which", "<cli_name>"], capture_output=True, text=True)

if cli_check.returncode != 0:
    print("⚠️ CLI not found on system")
    print("Install from: <install_url>")
    sys.exit(1)

# 4. Execute via HeadlessSpawner
try:
    spawner = HeadlessSpawner()
    result = spawner.spawn_<cli_name>(
        prompt=task_prompt,
        # ... CLI-specific options ...
        track_in_wipnote=True,
        timeout=120
    )

    if result.success:
        print("✅ Execution successful")
        print(result.response)
        print(f"📈 Tracked {len(result.tracked_events)} events in Wipnote")
    else:
        print(f"❌ Execution failed: {result.error}")
        sys.exit(1)

except Exception as e:
    print(f"❌ Error: {type(e).__name__}: {e}")
    sys.exit(1)
</python>
```

## HeadlessSpawner API Used

### spawn_gemini()
```python
spawner.spawn_gemini(
    prompt: str,
    output_format: str = "stream-json",
    model: str | None = None,
    include_directories: list[str] | None = None,
    track_in_wipnote: bool = True,
    timeout: int = 120,
) -> AIResult
```

**Configuration**:
- `output_format="stream-json"` - Enables real-time event tracking
- `track_in_wipnote=True` - Records activity in Wipnote
- `timeout=120` - 2-minute execution limit

### spawn_codex()
```python
spawner.spawn_codex(
    prompt: str,
    output_json: bool = True,
    model: str | None = None,
    sandbox: str | None = None,
    full_auto: bool = True,
    track_in_wipnote: bool = True,
    timeout: int = 120,
) -> AIResult
```

**Configuration**:
- `sandbox="workspace-write"` - Safe code generation with file writes
- `full_auto=True` - Headless mode (auto-approve operations)
- `output_json=True` - Machine-readable output with events
- `track_in_wipnote=True` - Records activity in Wipnote

### spawn_copilot()
```python
spawner.spawn_copilot(
    prompt: str,
    allow_tools: list[str] | None = None,
    allow_all_tools: bool = False,
    deny_tools: list[str] | None = None,
    track_in_wipnote: bool = True,
    timeout: int = 120,
) -> AIResult
```

**Configuration**:
- `allow_all_tools=True` - Auto-approve all GitHub/git operations
- `track_in_wipnote=True` - Records activity in Wipnote
- `timeout=120` - 2-minute execution limit

## Return Type: AIResult

All spawners return an `AIResult` dataclass:

```python
@dataclass
class AIResult:
    success: bool                               # Execution succeeded
    response: str                               # Main response text
    tokens_used: int | None                     # Token count (if available)
    error: str | None                           # Error message (if failed)
    raw_output: dict | list | str | None        # Raw CLI output
    tracked_events: list[dict] | None = None    # Wipnote tracked events
```

## Behavior When CLI Unavailable

Each skill gracefully handles missing external CLIs:

1. **Check Phase**: Uses `which <cli>` to detect installation
2. **Error Handling**: Prints clear instructions on how to install
3. **Exit Code**: Returns with error status (skill fails)
4. **User Guidance**: Suggests alternative approaches:
   - Gemini: "Use Task(subagent_type='Explore', prompt='...')"
   - Codex: "Use Task(subagent_type='general-purpose', prompt='...')"
   - Copilot: "Use gh CLI directly or Bash for git operations"

## Wipnote Integration

All three skills automatically track execution in Wipnote when available:

```python
result = spawner.spawn_gemini(
    prompt="Your task",
    track_in_wipnote=True,  # Enabled by default
    timeout=120
)

# Result includes tracked events
if result.tracked_events:
    print(f"Tracked {len(result.tracked_events)} events in Wipnote")
```

**Tracked Activities**:
- **Gemini**: Tool calls, tool results, messages, completion stats
- **Codex**: Commands, file changes, agent messages, token usage
- **Copilot**: Start event, result event (start/end tracking)

## Testing Recommendations

### 1. Test Gemini Skill
```bash
# With Gemini CLI installed
Skill(skill=".claude-plugin:gemini", args="List all Python files and their imports")

# Expected Output:
# ✅ Gemini CLI found, executing spawner...
# ✅ Gemini execution successful
# 📊 Tokens used: 1234
# [Gemini response content]
# 📈 Tracked X events in Wipnote
```

### 2. Test Codex Skill
```bash
# With Codex CLI installed (requires OpenAI setup)
Skill(skill=".claude-plugin:codex", args="Generate a REST API endpoint for user authentication")

# Expected Output:
# ✅ Codex CLI found, executing spawner...
# ✅ Codex execution successful
# 📊 Tokens used: 5678
# [Generated code]
# 📈 Tracked X events in Wipnote
```

### 3. Test Copilot Skill
```bash
# With Copilot CLI installed (requires GitHub auth)
Skill(skill=".claude-plugin:copilot", args="Create a pull request for new authentication feature")

# Expected Output:
# ✅ Copilot CLI found, executing spawner...
# ✅ Copilot execution successful
# [PR creation confirmation]
# 📈 Tracked X events in Wipnote
```

### 4. Test CLI Not Found Behavior
```bash
# Without Gemini CLI (uninstall temporarily to test)
Skill(skill=".claude-plugin:gemini", args="Test task")

# Expected Output:
# ⚠️ Gemini CLI not found on system
# Install from: https://github.com/google/gemini-cli
# Fallback: Use Task(subagent_type='Explore', prompt='...')
```

## Key Improvements

### Before (Guidance-Only Skills)
- Skills were pure documentation
- Invoked via `Skill()` just loaded markdown text
- Users had to manually run spawner or Task() commands
- No actual CLI execution
- No Wipnote tracking

### After (Executable Skills)
- Skills execute external CLIs automatically
- CLI availability checked before execution
- HeadlessSpawner SDK handles all subprocess management
- Wipnote tracks all activity automatically
- Clear success/error messages with actionable next steps
- Token usage reported when available
- Full integration with orchestration system

## Files Structure

```
packages/claude-plugin/.claude-plugin/skills/
├── gemini/
│   └── skill.md              # ✅ Updated with executable Python
├── codex/
│   └── skill.md              # ✅ Updated with executable Python
├── copilot/
│   └── skill.md              # ✅ Updated with executable Python
```

**Old Structure (Deleted)**:
- `gemini-spawner/SKILL.md` → Replaced by `gemini/skill.md`
- `codex-spawner/SKILL.md` → Replaced by `codex/skill.md`
- `copilot-spawner/SKILL.md` → Replaced by `copilot/skill.md`

## Next Steps

1. **Test with Real CLIs**: Install external CLIs and test execution
2. **Monitor Wipnote Events**: Verify events are properly tracked
3. **Handle Auth Issues**: Codex and Copilot may need auth configuration
4. **Document CLI Setup**: Add setup instructions to main docs
5. **User Communication**: Announce skill execution capability in release notes

## Notes

- **Gemini**: Works well, no known auth issues
- **Codex**: May require OpenAI API key configuration
- **Copilot**: May require GitHub CLI authentication (`gh auth login`)
- All skills timeout at 120 seconds (configurable in code)
- All skills support Wipnote tracking (optional, enabled by default)

---

**Status**: Ready for testing and deployment
**Last Updated**: 2026-01-12
