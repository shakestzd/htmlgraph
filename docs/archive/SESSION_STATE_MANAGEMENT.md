# Session State Management - Automatic Environment Variable Setup

## Overview

Wipnote SDK now provides automatic session state detection and environment variable management through the `SessionStateManager` class. This enables the SessionStart hook to automatically detect post-compact sessions, determine delegation status, and set up environment variables without manual configuration.

## Key Features

### 1. Automatic Session State Detection

The SDK automatically detects:

- **Session Source**: `startup`, `resume`, `compact`, or `clear`
- **Post-Compact Status**: Whether this is a session after context compaction
- **Delegation Status**: Whether orchestrator delegation should be active
- **Previous Session ID**: For tracking continuity across sessions
- **Session Validity**: Whether the session is valid for tracking

### 2. Environment Variable Management

Automatically sets environment variables that persist across context boundaries:

- `CLAUDE_SESSION_ID` - Current session identifier
- `CLAUDE_SESSION_SOURCE` - Session type (startup|resume|compact|clear)
- `CLAUDE_SESSION_COMPACTED` - True if post-compact
- `CLAUDE_DELEGATION_ENABLED` - True if delegation enabled
- `CLAUDE_PREVIOUS_SESSION_ID` - Previous session ID (if available)
- `CLAUDE_ORCHESTRATOR_ACTIVE` - True if orchestrator should be active
- `CLAUDE_PROMPT_PERSISTENCE_VERSION` - Version (1.0)

### 3. Session Metadata Recording

Stores session state metadata for future reference:

```json
{
  "session_id": "sess-abc123",
  "source": "compact",
  "is_post_compact": true,
  "delegation_enabled": true,
  "timestamp": "2026-01-05T10:00:00+00:00",
  "environment_vars": {
    "CLAUDE_SESSION_ID": "sess-abc123",
    "CLAUDE_DELEGATION_ENABLED": "true",
    ...
  }
}
```

## API Reference

### Using SessionStateManager Directly

```python
from wipnote.session_state import SessionStateManager

manager = SessionStateManager(".wipnote")

# Get current session state (auto-detects everything)
state = manager.get_current_state()

# Access state information
print(f"Session ID: {state['session_id']}")
print(f"Post-compact: {state['is_post_compact']}")
print(f"Delegation enabled: {state['delegation_enabled']}")

# Set up environment variables (automatically sets all vars)
env_vars = manager.setup_environment_variables(state)

# Check if this is post-compact
is_compact = manager.detect_compact_automatically()
```

### Using SDK Sessions Collection

```python
from wipnote import SDK

sdk = SDK(agent="claude-code")

# Get current session state
state = sdk.sessions.get_current_state()

# Set up environment variables
env_vars = sdk.sessions.setup_environment_variables(state)

# Record state for future reference
sdk.sessions.record_state(
    session_id=state['session_id'],
    source=state['session_source'],
    is_post_compact=state['is_post_compact'],
    delegation_enabled=state['delegation_enabled'],
    environment_vars=env_vars
)
```

### Integration with SessionStart Hook

```python
#!/usr/bin/env -S uv run
# .claude/hooks/scripts/session-start.py

from wipnote import SDK

def setup_session_state_and_environment():
    """Automatically set up session state and environment variables."""
    try:
        sdk = SDK(agent="claude-code")

        # Get current session state (auto-detects everything)
        state = sdk.sessions.get_current_state()

        # Automatically set up environment variables
        env_vars = sdk.sessions.setup_environment_variables(state)

        # All environment variables now set - no manual management needed!
        return env_vars
    except Exception as e:
        print(f"Warning: Could not set up session state: {e}")
        return None

# Call at session start
env_vars = setup_session_state_and_environment()
```

## SessionState Structure

The `SessionState` TypedDict contains:

```python
{
    "session_id": str,           # Current session ID
    "session_source": str,       # "startup", "resume", "compact", "clear"
    "is_post_compact": bool,     # True if post-compact
    "previous_session_id": str | None,  # Previous session ID
    "delegation_enabled": bool,  # Should delegation be active
    "prompt_injected": bool,     # Was orchestrator prompt injected
    "session_valid": bool,       # Is session valid for tracking
    "timestamp": str,            # ISO timestamp
    "compact_metadata": dict[str, Any],  # Compact detection details
}
```

## How Compaction Detection Works

The system uses several heuristics to detect post-compact sessions:

1. **Session ID Comparison**
   - If `CLAUDE_SESSION_ID` differs from previous session: likely post-compact
   - If same as previous session: likely resume (context switch)

2. **Session End Detection**
   - If previous session marked as `is_ended`: confirms post-compact
   - Checks `.wipnote/sessions/session_state.json`

3. **Compact Marker File**
   - If `.wipnote/sessions/.compacted` exists: post-compact
   - Marker can be created by `/clear` command or manual compaction

4. **Delegation Auto-Enable**
   - On post-compact: automatically enable delegation (need context carry-over)
   - Can be disabled via `HTMLGRAPH_DELEGATION_DISABLE=1`

## Use Cases

### 1. Fresh Session (Startup)

```
CLAUDE_SESSION_ID=sess-abc123 (new)
Previous state: None

Result:
- session_source: "startup"
- is_post_compact: false
- delegation_enabled: false (no previous context)
```

### 2. Context Switch (Resume)

```
CLAUDE_SESSION_ID=sess-abc123 (same)
Previous session ID: sess-abc123
Previous session NOT ended

Result:
- session_source: "resume"
- is_post_compact: false
- delegation_enabled: unchanged
```

### 3. Post-Compact (Resume after Compaction)

```
CLAUDE_SESSION_ID=sess-new456 (different)
Previous session ID: sess-abc123
Previous session marked as ended

Result:
- session_source: "compact"
- is_post_compact: true
- delegation_enabled: true (auto-enable)
- previous_session_id: "sess-abc123"
```

### 4. Clear Command

```
CLAUDE_SESSION_ID=sess-new789 (new)
Clear marker file exists: .wipnote/sessions/.compacted

Result:
- session_source: "clear"
- is_post_compact: false
- delegation_enabled: false (fresh start)
```

## Benefits

1. **No Manual Configuration**
   - User doesn't manage environment variables
   - SessionStart hook just calls: `sdk.sessions.setup_environment_variables()`

2. **Automatic Post-Compact Detection**
   - No need to manually track session IDs
   - Wipnote detects compaction automatically

3. **Intelligent Delegation Status**
   - Automatically enable delegation when needed (post-compact)
   - Can be overridden via environment variable

4. **Persistent State**
   - Session metadata stored in `.wipnote/sessions/session_state.json`
   - Available for future reference and debugging

5. **Zero User Friction**
   - SessionStart hook setup is trivial
   - Works automatically without user intervention

## Testing

Run the test suite:

```bash
uv run pytest tests/python/test_session_state_manager.py -v

# All 15 tests pass:
# - Fresh session detection
# - Post-compact detection
# - Environment variable setup
# - Delegation status determination
# - Session validity checking
# - Metadata recording
# - Auto-compact detection
```

## Configuration

### Override Delegation

Disable delegation if needed:

```bash
export HTMLGRAPH_DELEGATION_DISABLE=1
```

### Check Session State

```bash
# View current session state
cat .wipnote/sessions/session_state.json

# Check environment variables
echo $CLAUDE_SESSION_ID
echo $CLAUDE_DELEGATION_ENABLED
echo $CLAUDE_SESSION_COMPACTED
```

## Migration Guide

### For SessionStart Hook Writers

**Before (manual):**

```python
# Had to manually track session IDs
# Had to set environment variables manually
# Had to detect post-compact manually
```

**After (automatic):**

```python
from wipnote import SDK

sdk = SDK(agent="claude-code")
state = sdk.sessions.get_current_state()
env_vars = sdk.sessions.setup_environment_variables(state)
# Done! All environment variables set automatically
```

### For SDK Users

No changes needed! The SDK automatically initializes `SessionStateManager` on first access to `sdk.sessions`:

```python
from wipnote import SDK

sdk = SDK(agent="claude-code")

# SessionStateManager is available automatically
state = sdk.sessions.get_current_state()
```

## Implementation Details

### SessionStateManager (`src/python/wipnote/session_state.py`)

- **Core class** for session state detection and management
- Detects session source by comparing session IDs and metadata
- Manages environment variables (set, retrieve, record)
- Stores session metadata in JSON file

### SessionCollection (`src/python/wipnote/collections/session.py`)

- **SDK integration layer** for session state operations
- Wraps SessionStateManager with SDK context
- Available via `sdk.sessions` in all SDK code
- Extends BaseCollection with session-specific methods

### Integration Points

1. **SessionStart Hook**: Calls `setup_session_state_and_environment()`
2. **SDK Initialization**: Auto-initializes SessionCollection
3. **Session Manager**: Uses SessionStateManager for state tracking

## Troubleshooting

### Session state not being saved

Check file permissions:

```bash
ls -la .wipnote/sessions/session_state.json
```

### Delegation not being enabled after compact

Verify post-compact detection:

```bash
python -c "from wipnote import SDK; sdk = SDK(); state = sdk.sessions.get_current_state(); print(f\"Post-compact: {state['is_post_compact']}\")"
```

### Environment variables not set

Check SessionStart hook is running:

```bash
echo $CLAUDE_SESSION_ID  # Should be non-empty
```

## Performance Notes

- Session state detection is O(1) - reads single JSON file
- Environment variable setup is O(1) - sets 7 environment variables
- No graph loading required - works independently of feature/bug graphs
- Thread-safe for concurrent sessions

## Future Enhancements

1. **Delegation Hints**: Return hint about whether delegation should be active
2. **Analytics Integration**: Record session transitions for analytics
3. **Cloud Sync**: Sync session state to cloud for multi-device support
4. **Metrics**: Track session continuity metrics over time
5. **Recovery**: Auto-detect and recover from crashed sessions

## See Also

- `AGENTS.md` - Main SDK documentation
- `test_session_state_manager.py` - Test examples
- `.claude/hooks/scripts/session-start.py` - Hook integration
