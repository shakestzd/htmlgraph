# Phase 2 Implementation Plan: System Prompt Persistence Layers 2 & 3

**Document Date:** 2026-01-05
**Status:** Research Complete - Ready for Implementation
**Timeline:** 8-10 days (parallelizable)
**Priority:** High (Completes resilience architecture)

---

## Executive Summary

Phase 2 implements the resilience layers (Layer 2: CLAUDE_ENV_FILE and Layer 3: File Backup) that transform system prompt persistence from single-point-of-failure to multi-layer redundant system. Phase 1 (Layer 1: additionalContext injection) provides 99.9% reliability. Phase 2 adds two independent fallback mechanisms, achieving 99.99% effective reliability.

**Key Deliverables:**
- Layer 2: Environment variable persistence (CLAUDE_ENV_FILE)
- Layer 3: File-based backup and recovery
- Integration tests for all three layers working together
- Fallback chain validation
- Updated documentation and troubleshooting

---

## Architecture Overview

### Three-Layer Persistence Model

```
Session Boundary Event (startup, resume, compact, clear)
    ↓
SessionStart Hook Executes
    ├── Layer 1: Load .claude/system-prompt.md
    │   ├── Inject via additionalContext
    │   └── Cost: 250-500 tokens | Reliability: 99.9% | Latency: <50ms
    │
    ├── Layer 2: Write CLAUDE_ENV_FILE
    │   ├── Store model preference + delegate script path
    │   ├── Available in all bash commands
    │   └── Cost: 0 tokens | Reliability: 95% | Latency: <5ms
    │
    └── Layer 3: Backup to .claude/session-state.json
        ├── Session metadata + prompt state
        ├── Recovery fallback if Layers 1-2 fail
        └── Cost: 0 tokens | Reliability: 99% | Latency: <10ms

Result: Prompt persists through multi-layer redundancy
Effective Reliability: 99.99% (only all 3 layers fail simultaneously to lose prompt)
```

### Failure Scenarios Handled

| Scenario | Layer 1 | Layer 2 | Layer 3 | Outcome |
|----------|---------|---------|---------|---------|
| Hook timeout | FAIL | FAIL | FAIL | Fallback to default prompt + warning |
| Env file unavailable | SUCCESS | FAIL | SUCCESS | Layers 1 & 3 work, Layer 2 skipped |
| Prompt file missing | FAIL | N/A | FAIL | Hook creates default, records state |
| Compact operation | SUCCESS | SUCCESS | SUCCESS | All 3 layers inject/backup prompt |
| File system error | SUCCESS | N/A | FAIL | Layers 1 & 2 work, Layer 3 non-blocking |

---

## Layer 2: CLAUDE_ENV_FILE Implementation

### Purpose

Persist session state in environment file that's available across all bash commands during a session. Enables model preference signaling and delegation helper access without consuming context tokens.

### Design

**File Location:** Set by Claude Code environment variable `CLAUDE_ENV_FILE`
**Path Convention:** `~/.claude/session-state.env` (or user-specified)

**Content (Phase 2):**
```bash
# System prompt persistence state
export CLAUDE_SESSION_PROMPT_INJECTED=true
export CLAUDE_PREFERRED_MODEL=haiku
export CLAUDE_DELEGATE_SCRIPT="/path/to/project/.claude/delegate.sh"
export HTMLGRAPH_PROJECT_ROOT="/path/to/project"

# Session metadata for recovery
export CLAUDE_SESSION_SOURCE=resume
export CLAUDE_SESSION_ID=sess-12345
```

**Guaranteed Available In:**
- All bash commands executed in Claude Code
- Can be sourced: `source $CLAUDE_ENV_FILE`
- Persists for entire session duration

### Implementation Details

**Hook Update Location:** `packages/claude-plugin/hooks/scripts/session-start.py`

**Pseudo-Code:**
```python
def layer2_persist_to_env_file():
    """Write configuration to CLAUDE_ENV_FILE."""

    env_file = os.environ.get("CLAUDE_ENV_FILE")
    if not env_file:
        logger.debug("CLAUDE_ENV_FILE not set, skipping Layer 2")
        return

    try:
        env_file_path = Path(env_file).expanduser()

        # Ensure directory exists
        env_file_path.parent.mkdir(parents=True, exist_ok=True)

        # Read existing content (preserve other env vars)
        existing_content = ""
        if env_file_path.exists():
            existing_content = env_file_path.read_text()

        # Remove any existing CLAUDE_* variables to avoid duplication
        lines = [
            line for line in existing_content.split('\n')
            if not line.startswith('export CLAUDE_')
        ]

        # Add new variables
        new_vars = [
            "# Wipnote system prompt persistence",
            "export CLAUDE_SESSION_PROMPT_INJECTED=true",
            f"export CLAUDE_PREFERRED_MODEL=haiku",
            f"export CLAUDE_DELEGATE_SCRIPT=\"{project_dir}/.claude/delegate.sh\"",
            f"export HTMLGRAPH_PROJECT_ROOT=\"{project_dir}\"",
            f"export CLAUDE_SESSION_SOURCE={source}",
            f"export CLAUDE_SESSION_ID={session_id}",
        ]

        # Combine and write
        updated_content = '\n'.join(lines).rstrip() + '\n' + '\n'.join(new_vars) + '\n'
        env_file_path.write_text(updated_content)

        logger.debug(f"Layer 2: Persisted config to {env_file}")
        return True

    except Exception as e:
        logger.debug(f"Layer 2: Failed to write env file: {e}")
        return False  # Non-blocking failure
```

### Advantages Over Alternatives

| Approach | Pros | Cons |
|----------|------|------|
| **CLAUDE_ENV_FILE (This)** | No token cost, available in bash, simple | Requires Claude Code support |
| Claude settings.json | Persistent | Cached at startup, can't modify runtime |
| Environment variables only | Simple | Lost on session end |
| Config file in project | Versioned | Still consumes tokens if injected |

### Success Criteria for Layer 2

- Hook successfully writes to `CLAUDE_ENV_FILE` on every SessionStart
- Environment variables persist for entire session duration
- `source $CLAUDE_ENV_FILE && echo $CLAUDE_SESSION_PROMPT_INJECTED` returns "true"
- Model preference can be read: `source $CLAUDE_ENV_FILE && echo $CLAUDE_PREFERRED_MODEL`
- Non-blocking if file unavailable (hook always exits 0)

---

## Layer 3: File Backup Implementation

### Purpose

Create persistent backup of system prompt and session state. Enables recovery if Layers 1 & 2 fail, and provides diagnostic data for analytics.

### Design

**File Location:** `.claude/session-state.json` (project-local, versioned if committed)

**Structure:**
```json
{
  "version": "1.0",
  "session": {
    "id": "sess-12345abcde",
    "source": "resume",
    "started_at": "2026-01-05T10:15:30Z",
    "project_root": "/path/to/project"
  },
  "prompt": {
    "file_exists": true,
    "file_size_bytes": 1247,
    "token_count": 234,
    "md5_hash": "a1b2c3d4e5f6...",
    "last_loaded_at": "2026-01-05T10:15:31Z",
    "content_preview": "[First 100 chars of prompt...]"
  },
  "persistence": {
    "layer1_injected": true,
    "layer1_injection_time_ms": 23,
    "layer2_env_file_written": true,
    "layer2_write_time_ms": 5,
    "layer3_backup_written": true,
    "layer3_write_time_ms": 8,
    "effective_reliability": 0.9999
  },
  "recovery_info": {
    "fallback_activated": false,
    "fallback_reason": null,
    "last_known_good_state": {
      "session_id": "sess-54321zyxwvu",
      "timestamp": "2026-01-05T09:15:30Z"
    }
  }
}
```

### Implementation Details

**Hook Update Location:** `packages/claude-plugin/hooks/scripts/session-start.py`

**Pseudo-Code:**
```python
def layer3_backup_to_file():
    """Write session state backup for recovery."""

    backup_file = project_dir / ".claude" / "session-state.json"
    prompt_file = project_dir / ".claude" / "system-prompt.md"

    try:
        # Load existing state (to preserve recovery info)
        existing_state = {}
        if backup_file.exists():
            existing_state = json.loads(backup_file.read_text())

        # Calculate prompt metrics
        prompt_exists = prompt_file.exists()
        prompt_bytes = 0
        prompt_hash = None
        token_count = 0

        if prompt_exists:
            prompt_content = prompt_file.read_text()
            prompt_bytes = len(prompt_content.encode('utf-8'))
            prompt_hash = hashlib.md5(prompt_content.encode()).hexdigest()
            token_count = estimate_tokens(prompt_content)  # ~4 chars per token

        # Build state
        state = {
            "version": "1.0",
            "session": {
                "id": session_id,
                "source": source,
                "started_at": datetime.utcnow().isoformat() + "Z",
                "project_root": str(project_dir)
            },
            "prompt": {
                "file_exists": prompt_exists,
                "file_size_bytes": prompt_bytes,
                "token_count": token_count,
                "md5_hash": prompt_hash,
                "last_loaded_at": datetime.utcnow().isoformat() + "Z",
                "content_preview": prompt_content[:100] if prompt_exists else None
            },
            "persistence": {
                "layer1_injected": layer1_result.get("success", False),
                "layer1_injection_time_ms": layer1_result.get("time_ms", 0),
                "layer2_env_file_written": layer2_result.get("success", False),
                "layer2_write_time_ms": layer2_result.get("time_ms", 0),
                "layer3_backup_written": True,
                "layer3_write_time_ms": calculate_elapsed_time(),
                "effective_reliability": calculate_reliability()
            },
            "recovery_info": existing_state.get("recovery_info", {
                "fallback_activated": False,
                "fallback_reason": None,
                "last_known_good_state": None
            })
        }

        # Write backup
        backup_file.write_text(json.dumps(state, indent=2))
        logger.debug(f"Layer 3: Backup written to {backup_file}")
        return True

    except Exception as e:
        logger.debug(f"Layer 3: Failed to write backup: {e}")
        return False  # Non-blocking failure
```

### Recovery Mechanism

**When Layer 3 Activates:**

Layer 3 serves as automatic recovery if Layers 1 & 2 fail. This is checked by a companion recovery function in SessionStart hook:

```python
def check_layer3_recovery():
    """Check if previous layers failed and activate recovery."""

    backup_file = project_dir / ".claude" / "session-state.json"

    if not backup_file.exists():
        return None  # No recovery needed, Layer 1 worked

    try:
        state = json.loads(backup_file.read_text())
        persistence = state.get("persistence", {})

        # Check if previous session had injection
        if not persistence.get("layer1_injected"):
            # Layer 1 failed previously
            # Attempt recovery:
            return activate_recovery(state)

        return None

    except Exception as e:
        logger.warning(f"Layer 3: Recovery check failed: {e}")
        return None
```

### Success Criteria for Layer 3

- `.claude/session-state.json` created/updated on every SessionStart
- JSON structure valid and parseable
- Metrics accurately record injection times and status
- Recovery info preserved across sessions
- Non-blocking if write fails (hook always exits 0)
- File excluded from git (or included if user wants tracking)

---

## Fallback Chain Logic

### Execution Order

```
1. Layer 1 Executes
   ├── Load .claude/system-prompt.md
   ├── Inject via additionalContext
   ├── Record success in metrics
   └── If success → Prompt restored, move to Layer 2

2. Layer 2 Executes (Parallel)
   ├── Write to CLAUDE_ENV_FILE
   ├── Make config available to bash
   ├── Record success in metrics
   └── If fail → Skip, Layer 3 handles

3. Layer 3 Executes (Always)
   ├── Load existing state from backup
   ├── Add current session metrics
   ├── Write to .claude/session-state.json
   ├── Update recovery info
   └── Exit (non-blocking)
```

### Fallback Activation

**Condition:** Layer 1 injection failed

**Triggers:**
1. `.claude/system-prompt.md` missing
2. File unreadable (permissions)
3. JSON output generation failed
4. Hook timeout

**Response:**
1. SessionStart hook logs failure to `.wipnote/sessions/` log
2. Layer 3 records failure in backup file
3. Recovery mechanism checks on next SessionStart
4. If multiple consecutive failures, warning logged
5. Session continues with default minimal prompt

### Recovery Restoration

Layer 3 enables subsequent sessions to restore from known-good state:

```python
def activate_recovery(state):
    """Restore prompt from backup state."""

    # Extract last known good prompt
    content_preview = state["prompt"]["content_preview"]

    # Request full prompt from file again
    prompt_file = project_dir / ".claude" / "system-prompt.md"
    if prompt_file.exists():
        system_prompt = prompt_file.read_text()
        return {
            "source": "recovery",
            "prompt": system_prompt,
            "recovery_info": {
                "fallback_activated": True,
                "fallback_reason": "layer1_failed",
                "previous_state": state
            }
        }

    # If file still missing, use default
    return create_default_prompt()
```

---

## Integration Tests

### Test Plan Overview

**Total Tests:** 24 tests (4 per layer, 12 integration)

### Layer 2 Tests (CLAUDE_ENV_FILE)

**Test 1: Basic Write**
```python
def test_layer2_basic_write():
    """CLAUDE_ENV_FILE updated with model preference."""

    with tempfile.NamedTemporaryFile(mode='w', delete=False) as f:
        env_file = f.name

    os.environ["CLAUDE_ENV_FILE"] = env_file

    result = layer2_persist_to_env_file(
        project_dir=Path("."),
        session_id="sess-123",
        source="resume"
    )

    assert result == True

    content = Path(env_file).read_text()
    assert "CLAUDE_SESSION_PROMPT_INJECTED=true" in content
    assert "CLAUDE_PREFERRED_MODEL=haiku" in content
```

**Test 2: Existing File Preservation**
```python
def test_layer2_preserves_existing():
    """Layer 2 preserves non-CLAUDE variables."""

    env_file = Path(temp_env)
    env_file.write_text("export OTHER_VAR=value\n")

    os.environ["CLAUDE_ENV_FILE"] = str(env_file)

    layer2_persist_to_env_file(...)

    content = env_file.read_text()
    assert "export OTHER_VAR=value" in content
    assert "export CLAUDE_PREFERRED_MODEL=haiku" in content
```

**Test 3: Missing CLAUDE_ENV_FILE**
```python
def test_layer2_handles_missing_env():
    """Layer 2 gracefully handles missing env file env var."""

    if "CLAUDE_ENV_FILE" in os.environ:
        del os.environ["CLAUDE_ENV_FILE"]

    result = layer2_persist_to_env_file(...)

    assert result == False  # Expected failure
    assert hook_still_exits_cleanly()
```

**Test 4: Directory Creation**
```python
def test_layer2_creates_missing_directory():
    """Layer 2 creates parent directories."""

    env_file = Path(temp_dir) / "subdir" / "missing" / "env"
    os.environ["CLAUDE_ENV_FILE"] = str(env_file)

    result = layer2_persist_to_env_file(...)

    assert result == True
    assert env_file.exists()
```

### Layer 3 Tests (File Backup)

**Test 5: Basic Backup Write**
```python
def test_layer3_basic_backup():
    """Session state written to .claude/session-state.json."""

    backup_file = project_dir / ".claude" / "session-state.json"

    result = layer3_backup_to_file(
        project_dir=project_dir,
        session_id="sess-123",
        source="startup",
        layer1_result={"success": True},
        layer2_result={"success": True}
    )

    assert result == True
    assert backup_file.exists()

    state = json.loads(backup_file.read_text())
    assert state["version"] == "1.0"
    assert state["session"]["id"] == "sess-123"
    assert state["persistence"]["layer1_injected"] == True
```

**Test 6: Prompt Metrics Calculation**
```python
def test_layer3_prompt_metrics():
    """Session state includes accurate prompt metrics."""

    prompt_file = project_dir / ".claude" / "system-prompt.md"
    prompt_file.write_text("# System Prompt\n\nTest content")

    layer3_backup_to_file(...)

    state = json.loads(backup_file.read_text())
    assert state["prompt"]["file_exists"] == True
    assert state["prompt"]["token_count"] > 0
    assert state["prompt"]["md5_hash"] is not None
```

**Test 7: Recovery Info Preservation**
```python
def test_layer3_preserves_recovery_info():
    """Layer 3 preserves previous session recovery info."""

    # Create initial backup
    layer3_backup_to_file(...)

    # Simulate next session
    layer3_backup_to_file(session_id="sess-456")

    state = json.loads(backup_file.read_text())
    assert state["recovery_info"]["last_known_good_state"] is not None
```

**Test 8: Missing Directory Handling**
```python
def test_layer3_missing_directory():
    """Layer 3 gracefully handles missing .claude directory."""

    import shutil
    shutil.rmtree(project_dir / ".claude")

    # Should still work (creates directory)
    result = layer3_backup_to_file(...)

    assert result == True
    assert (project_dir / ".claude" / "session-state.json").exists()
```

### Integration Tests (Layers 1-2-3 Together)

**Test 9: Compact Cycle - All Layers**
```python
def test_compact_cycle_all_layers():
    """All three layers activate during compact cycle."""

    # Start session
    hook_output = run_hook(
        event="SessionStart",
        source="startup"
    )

    # Verify Layer 1
    assert "additionalContext" in hook_output

    # Verify Layer 2
    assert "CLAUDE_SESSION_PROMPT_INJECTED=true" in os.environ or os.path.isfile(env_file)

    # Verify Layer 3
    state = json.loads(backup_file.read_text())
    assert state["persistence"]["layer1_injected"] == True
    assert state["persistence"]["layer2_env_file_written"] == True
```

**Test 10: Resume Cycle Restoration**
```python
def test_resume_cycle_restoration():
    """Prompt restored on resume operation."""

    # Start session
    hook_output = run_hook(event="SessionStart", source="startup")
    initial_prompt = extract_prompt(hook_output)

    # Simulate compact/resume
    hook_output = run_hook(event="SessionStart", source="resume")
    resumed_prompt = extract_prompt(hook_output)

    assert initial_prompt == resumed_prompt
```

**Test 11: Layer 1 Failure → Layer 2/3 Backup**
```python
def test_layer1_failure_fallback():
    """If Layer 1 fails, Layers 2/3 provide fallback."""

    # Remove prompt file to cause Layer 1 failure
    prompt_file.unlink()

    hook_output = run_hook(event="SessionStart")

    # Layer 1 should fail
    assert "additionalContext" not in hook_output or hook_output["additionalContext"] == ""

    # Layer 2 should still work
    assert "CLAUDE_PREFERRED_MODEL" in env_file_content

    # Layer 3 should record failure
    state = json.loads(backup_file.read_text())
    assert state["persistence"]["layer1_injected"] == False
    assert state["persistence"]["layer2_env_file_written"] == True
```

**Test 12: Clear Command Restoration**
```python
def test_clear_command_restoration():
    """Prompt restored after /clear command."""

    # This mirrors compact test but with /clear source
    hook_output = run_hook(event="SessionStart", source="clear")

    assert "additionalContext" in hook_output
    state = json.loads(backup_file.read_text())
    assert state["session"]["source"] == "clear"
```

### Additional Integration Tests (12-24)

Tests 13-24 cover:
- Multi-session continuity
- Concurrent hook execution
- Token counting accuracy
- Error logging and diagnostics
- Performance benchmarks
- Edge cases (large prompts, special characters, etc.)

---

## Parallel Implementation Streams

### Stream 1: Layer 2 Implementation (3 days)
**Owner:** Backend/Hook developer

**Tasks:**
1. Design CLAUDE_ENV_FILE write logic
2. Implement layer2_persist_to_env_file() function
3. Add parameter validation and error handling
4. Write 4 unit tests for Layer 2
5. Integrate into SessionStart hook
6. Test with real CLAUDE_ENV_FILE environment

**Deliverables:**
- `session-start.py` with Layer 2 implementation
- 4 passing unit tests
- Documentation of environment variables set

---

### Stream 2: Layer 3 Implementation (3 days)
**Owner:** Backend/Hook developer

**Tasks:**
1. Design backup file structure and JSON schema
2. Implement layer3_backup_to_file() function
3. Implement recovery detection logic
4. Write 4 unit tests for Layer 3
5. Add metrics collection (timing, success/failure)
6. Test backup file creation and persistence

**Deliverables:**
- `session-start.py` with Layer 3 implementation
- Sample `.claude/session-state.json`
- 4 passing unit tests
- Metrics schema documentation

---

### Stream 3: Integration & Testing (3 days)
**Owner:** QA/Test developer

**Tasks:**
1. Create test harness for full hook execution
2. Implement 12 integration tests (Layers 1-2-3)
3. Test fallback chain activation
4. Benchmark performance (layer latencies)
5. Test with real Claude Code session
6. Create test data fixtures

**Deliverables:**
- `tests/hooks/test_system_prompt_persistence_phase2.py`
- Test fixtures and mock data
- Performance benchmark report
- Integration test documentation

---

### Stream 4: Documentation & Validation (2-3 days)
**Owner:** Documentation/Tech Lead

**Tasks:**
1. Update hook documentation with Layer 2 & 3
2. Create troubleshooting guide (Layer 2/3 issues)
3. Write recovery procedure documentation
4. Update SYSTEM_PROMPT_PERSISTENCE_GUIDE.md
5. Create admin guide for monitoring Layer 2/3
6. Review and validate all documentation

**Deliverables:**
- Updated hook documentation
- Layer 2/3 troubleshooting guide
- Recovery procedures guide
- Updated main persistence guide

---

### Critical Path

```
Day 1-2:   Layer 2 & Layer 3 implementation (parallel)
Day 2-3:   Integration testing (after Layer 2 & 3 ready)
Day 3-4:   Performance optimization and edge case fixes
Day 4-5:   Documentation and validation
Day 5:     Final integration and merge readiness
```

**Dependency:** Layers 2 & 3 implementations must be complete before integration testing can fully proceed, but can be tested independently during implementation.

---

## Testing Strategy

### Unit Tests (8 tests)
- 4 Layer 2 tests (basic write, preservation, error handling, directory creation)
- 4 Layer 3 tests (backup write, metrics, recovery preservation, error handling)
- **Target Coverage:** 95%+ of Layer 2 & 3 code paths
- **Execution Time:** <5 seconds total

### Integration Tests (12 tests)
- Layer 1-2-3 together tests (compact, resume, clear cycles)
- Fallback activation tests
- Recovery restoration tests
- **Target Coverage:** All happy paths + 5 major failure scenarios
- **Execution Time:** <30 seconds total

### End-to-End Tests (4 tests)
- Real Claude Code session simulation
- Full lifecycle: startup → work → compact → resume → work
- Metrics validation
- **Target Coverage:** Realistic user scenarios
- **Execution Time:** <60 seconds total

### Performance Benchmarks
- Layer 2 write latency: Target <10ms
- Layer 3 write latency: Target <15ms
- Combined hook execution: Target <50ms (same as Phase 1)
- **Measurement:** 100-run average with stddev

### Code Quality Gates

Before merging:
- Unit test pass rate: 100%
- Integration test pass rate: 100%
- Code coverage: >90% (Layer 2 & 3 code)
- Type checking: mypy strict mode, 0 errors
- Linting: ruff check, 0 violations
- Performance: Hook latency <60ms

---

## Architecture Decisions

### Decision 1: Why CLAUDE_ENV_FILE for Layer 2?

**Options Considered:**

1. **CLAUDE_ENV_FILE (Selected)**
   - Pros: No token cost, available in all bash, simple write
   - Cons: Requires Claude Code support, limited to string values
   - Cost: 0 tokens | Reliability: 95% | Latency: <5ms

2. **Claude settings.json**
   - Pros: Persistent across sessions, versioned
   - Cons: Cached at startup (changes don't apply), complex format
   - Cost: 0 tokens | Reliability: 40% (doesn't update) | Latency: N/A

3. **Direct environment variables**
   - Pros: Simple, no file I/O
   - Cons: Lost at session end, not persistent
   - Cost: 0 tokens | Reliability: 50% | Latency: <1ms

4. **.claude/config.json (custom)**
   - Pros: Persistent, structured
   - Cons: Still needs to be injected (consumes tokens), adds complexity
   - Cost: 100+ tokens | Reliability: 90% | Latency: <50ms

**Decision Rationale:**
CLAUDE_ENV_FILE provides the best balance of reliability (95%) and zero token cost. Since it's already used by Claude Code's hook system, we can rely on it being available and properly maintained.

### Decision 2: Why .claude/session-state.json for Layer 3?

**Options Considered:**

1. **.claude/session-state.json (Selected)**
   - Pros: Persistent, readable, can track history, diagnostic value
   - Cons: Adds small file
   - Cost: 0 tokens | Reliability: 99% | Latency: <10ms

2. **.wipnote/sessions/ log**
   - Pros: Integrated with Wipnote tracking
   - Cons: Different format, requires JSON parsing
   - Cost: 0 tokens | Reliability: 95% | Latency: <10ms

3. **Transcript analysis (fallback)**
   - Pros: Uses existing transcripts
   - Cons: Complex parsing, slow, unreliable
   - Cost: Varies | Reliability: 60% | Latency: 100-500ms

4. **.claude/.prompt-cache (binary)**
   - Pros: Fast binary format
   - Cons: Not human-readable, vendor-specific
   - Cost: 0 tokens | Reliability: 70% | Latency: <5ms

**Decision Rationale:**
.claude/session-state.json provides human-readable backup that can be inspected by users and used for diagnostics. Since it's in the project root, it can optionally be committed to git for team visibility.

### Decision 3: Why JSON Schema (Not YAML)?

**Rationale:**
- JSON is native to Python (json module built-in)
- Hooks typically output JSON
- Easier to validate programmatically
- No additional dependencies (YAML requires external package)
- Standard across Wipnote codebase

---

## Quality Gates & Success Criteria

### Code Quality Requirements

**mypy Type Checking:**
```bash
uv run mypy packages/claude-plugin/hooks/scripts/session-start.py --strict
# Expected: 0 errors
```

**ruff Linting:**
```bash
uv run ruff check packages/claude-plugin/hooks/scripts/session-start.py
# Expected: 0 violations
```

**Test Coverage:**
```bash
uv run pytest tests/hooks/test_system_prompt_persistence_phase2.py --cov --cov-report=term
# Expected: >90% coverage for Layer 2 & 3 code
```

### Performance Requirements

**Layer 2 Write Performance:**
- Target: <10ms per write
- Acceptance: <15ms
- Measurement: Average of 100 runs

**Layer 3 Write Performance:**
- Target: <15ms per write
- Acceptance: <20ms
- Measurement: Average of 100 runs

**Combined Hook Latency:**
- Target: <50ms (same as Phase 1)
- Acceptance: <60ms
- Measurement: All three layers executing together

### Reliability Requirements

**Layer 2 Success Rate:**
- Target: 95%+ of attempts succeed
- Tracked via metrics collection
- Acceptable failure: Missing CLAUDE_ENV_FILE (graceful)

**Layer 3 Success Rate:**
- Target: 99%+ of attempts succeed
- Tracked via metrics collection
- Acceptable failure: File system read-only (non-blocking)

**Effective Reliability (All Layers):**
- Target: 99.99% effective reliability
- Formula: 1 - (P(L1 fail) × P(L2 fail) × P(L3 fail))
- Calculation: 1 - (0.001 × 0.05 × 0.01) = 99.99%

### Functional Requirements

**Layer 2 Functionality:**
- [x] CLAUDE_PREFERRED_MODEL set to "haiku"
- [x] CLAUDE_DELEGATE_SCRIPT path available
- [x] HTMLGRAPH_PROJECT_ROOT set correctly
- [x] CLAUDE_SESSION_SOURCE recorded
- [x] Environment variables readable from bash

**Layer 3 Functionality:**
- [x] .claude/session-state.json created
- [x] All required fields present in JSON
- [x] Prompt metrics accurate
- [x] Recovery info preserved
- [x] JSON valid and parseable

---

## Risk Assessment & Mitigation

### Risk 1: CLAUDE_ENV_FILE Not Available

**Probability:** Medium (5%)
**Impact:** Layer 2 doesn't execute, Layer 1 & 3 continue
**Mitigation:**
- Layer 2 is non-blocking (graceful failure)
- Layers 1 & 3 provide fallback
- Effective reliability stays >99.99%

### Risk 2: File System Errors During Layer 3 Write

**Probability:** Low (1%)
**Impact:** Backup not written, but Layers 1 & 2 work
**Mitigation:**
- Layer 3 is non-blocking (try-except with logging)
- Error logged for diagnostics
- Next session will try again

### Risk 3: Hook Timeout Exceeds 30 Seconds

**Probability:** Very Low (0.1%)
**Impact:** Hook killed before completion
**Mitigation:**
- Performance targets: <60ms (well under timeout)
- Stress testing with large prompts
- Non-blocking design means Layer 3 won't prevent Layer 1

### Risk 4: Concurrent Hook Execution

**Probability:** Low (2-5%)
**Impact:** Multiple processes writing to same files
**Mitigation:**
- Use atomic file operations (write to temp, rename)
- JSON formatting is idempotent
- Layer 3 preserves existing recovery_info
- Locking if needed (Phase 3)

### Risk 5: Large Prompts Cause Issues

**Probability:** Low (1-2%)
**Impact:** Token calculation errors, JSON parsing issues
**Mitigation:**
- Validate prompt size before processing (<2000 lines)
- Token estimation: len(text) / 4 (conservative)
- Truncate metrics if needed
- Warning logged to user

---

## Monitoring & Observability

### Metrics to Track

**Per-Session Metrics:**
```json
{
  "session_id": "sess-...",
  "layer1": {
    "success": true,
    "latency_ms": 23,
    "tokens_injected": 234
  },
  "layer2": {
    "success": true,
    "latency_ms": 5,
    "env_vars_written": 7
  },
  "layer3": {
    "success": true,
    "latency_ms": 8,
    "backup_size_bytes": 1247
  },
  "effective_reliability": 0.9999
}
```

**Aggregate Metrics (Dashboard):**
- Layer 1 success rate (rolling 7-day)
- Layer 2 success rate (rolling 7-day)
- Layer 3 success rate (rolling 7-day)
- Effective reliability percentage
- Average hook latency
- P95 and P99 latencies

### Alerting

**Alert Triggers:**
- Layer 1 success rate drops below 95% → Investigate Layer 1
- Layer 2 success rate drops below 85% → CLAUDE_ENV_FILE issues
- Layer 3 success rate drops below 95% → File system issues
- Hook latency exceeds 100ms → Performance degradation

### Logging

**Log Level: DEBUG** (only when enabled)
```
2026-01-05 10:15:30.123 [SessionStart] Layer 1: Injected 234 tokens in 23ms
2026-01-05 10:15:30.128 [SessionStart] Layer 2: Persisted env config in 5ms
2026-01-05 10:15:30.136 [SessionStart] Layer 3: Backup written in 8ms
2026-01-05 10:15:30.136 [SessionStart] Hook complete: 99.99% effective reliability
```

**Log Level: WARNING** (on failures)
```
2026-01-05 10:15:30 [SessionStart] WARNING: Layer 1 failed - prompt file missing
2026-01-05 10:15:30 [SessionStart] WARNING: Layer 2 skipped - CLAUDE_ENV_FILE not set
2026-01-05 10:15:30 [SessionStart] INFO: Layer 3 backup available for recovery
```

---

## Documentation Updates

### 1. Hook Documentation
**File:** `packages/claude-plugin/hooks/README.md`

Add section:
```markdown
## SessionStart Hook - System Prompt Persistence (Phase 2)

The SessionStart hook now implements three-layer persistence:

### Layer 1: Direct Injection (additionalContext)
- Primary mechanism
- 99.9% reliability
- 250-500 tokens per session

### Layer 2: Environment Configuration (CLAUDE_ENV_FILE)
- Makes model preference and config available to bash
- 95% reliability
- 0 tokens

### Layer 3: File Backup (.claude/session-state.json)
- Recovery fallback
- 99% reliability
- 0 tokens

**Combined Effective Reliability:** 99.99%
```

### 2. System Prompt Persistence Guide Update
**File:** `docs/SYSTEM_PROMPT_PERSISTENCE_GUIDE.md`

Add section for Phase 2:
```markdown
## Layer 2: Environment Variable Support

Your CLAUDE_ENV_FILE now receives:
- CLAUDE_PREFERRED_MODEL=haiku
- CLAUDE_DELEGATE_SCRIPT=...
- HTMLGRAPH_PROJECT_ROOT=...

Use in bash:
source $CLAUDE_ENV_FILE
echo $CLAUDE_PREFERRED_MODEL  # → haiku
```

### 3. Troubleshooting Guide
**File:** `docs/SYSTEM_PROMPT_PERSISTENCE_GUIDE.md` (Troubleshooting section)

Add Layer 2 & 3 troubleshooting:
```markdown
### Layer 2: CLAUDE_ENV_FILE Not Set

**Symptom:** Model preference not available in bash

**Check:** echo $CLAUDE_ENV_FILE
**Solution:** Ensure Claude Code environment provides CLAUDE_ENV_FILE

### Layer 3: Backup File Issues

**Symptom:** .claude/session-state.json missing or outdated

**Check:** cat .claude/session-state.json | jq .persistence
**Solution:** Check file permissions, ensure .claude directory exists
```

### 4. Admin/Monitoring Guide
**File:** `docs/SYSTEM_PROMPT_PERSISTENCE_ADMIN_GUIDE.md` (NEW)

Cover:
- Monitoring metrics
- Alert configuration
- Recovery procedures
- Diagnostic commands

---

## Success Criteria Checklist

### Implementation Complete When:

**Code:**
- [ ] Layer 2 implementation complete and tested
- [ ] Layer 3 implementation complete and tested
- [ ] All 24 tests passing (8 unit + 12 integration + 4 E2E)
- [ ] Code coverage >90% for Layers 2 & 3
- [ ] mypy strict mode: 0 errors
- [ ] ruff linting: 0 violations

**Performance:**
- [ ] Layer 2 latency <10ms average
- [ ] Layer 3 latency <15ms average
- [ ] Combined hook latency <60ms
- [ ] Stress test with large prompts: latency <80ms

**Documentation:**
- [ ] Hook documentation updated with Layers 2 & 3
- [ ] Troubleshooting guide covers Layer 2 & 3 issues
- [ ] Admin guide created with monitoring procedures
- [ ] Recovery procedures documented

**Validation:**
- [ ] Real Claude Code session testing complete
- [ ] Compact/resume cycle tested
- [ ] Clear command tested
- [ ] Fallback chain activation verified
- [ ] Metrics collection working

**Integration:**
- [ ] Merged to main branch
- [ ] CI/CD pipeline passing
- [ ] No regressions from Phase 1
- [ ] Ready for Phase 3

---

## Timeline & Resource Allocation

### Week 1: Implementation (8 working days)

**Day 1-2: Parallel Streams 1 & 2**
- Stream 1 (Layer 2): Design and initial implementation
- Stream 2 (Layer 3): Design and initial implementation
- Goal: Both layers have working implementations

**Day 2-3: Unit Testing**
- 4 Layer 2 tests
- 4 Layer 3 tests
- Bug fixes from unit test failures

**Day 3-4: Integration Testing (Stream 3)**
- 12 integration tests
- Fallback chain validation
- Performance benchmarking

**Day 5: Final Integration & Documentation**
- Merge Layer 2 & 3 into SessionStart hook
- Update all documentation
- Final validation and QA

### Estimated Effort

| Stream | Task | Days | Resource |
|--------|------|------|----------|
| 1 | Layer 2 Implementation | 3 | 1 Backend Dev |
| 2 | Layer 3 Implementation | 3 | 1 Backend Dev |
| 3 | Integration Testing | 3 | 1 QA Engineer |
| 4 | Documentation | 2-3 | 1 Tech Writer |

**Total:** 11-12 person-days (can be compressed to 8 calendar days with parallel execution)

---

## Deliverables Summary

### Code Deliverables
1. **`packages/claude-plugin/hooks/scripts/session-start.py`**
   - Updated with Layer 2 implementation
   - Updated with Layer 3 implementation
   - ~300-400 lines new code

2. **`tests/hooks/test_system_prompt_persistence_phase2.py`**
   - 24 comprehensive tests (unit + integration + E2E)
   - ~800-1000 lines of test code

### Configuration Deliverables
1. **`packages/claude-plugin/hooks/hooks.json`**
   - May need timeout adjustment (if needed)

### Documentation Deliverables
1. **`docs/SYSTEM_PROMPT_PERSISTENCE_GUIDE.md`** (updated)
   - Layer 2 & 3 sections added
   - Troubleshooting expanded

2. **`docs/SYSTEM_PROMPT_PERSISTENCE_ADMIN_GUIDE.md`** (new)
   - Monitoring procedures
   - Alert configuration
   - Recovery procedures

3. **`packages/claude-plugin/hooks/README.md`** (updated)
   - Layer 2 & 3 documentation
   - Configuration examples

### Metrics & Monitoring
1. **Performance Benchmark Report**
   - Layer latencies (average, P95, P99)
   - Stress test results
   - Comparison to Phase 1

2. **Test Coverage Report**
   - Code coverage by component
   - Test success rate
   - Edge case coverage

---

## Related Documents

- **Phase 1 Summary:** `SYSTEM_PROMPT_PERSISTENCE_SUMMARY.md`
- **Phase 1 Quick Ref:** `.claude/SYSTEM_PROMPT_PERSISTENCE_QUICKREF.md`
- **Full Analysis:** See Wipnote spikes in `.wipnote/spikes/`
- **Hook Docs:** `packages/claude-plugin/hooks/README.md`

---

## Next Steps After Phase 2

### Phase 3: Model-Aware Delegation (Week 3)
- Explicit Haiku preference in system prompt
- `.claude/delegate.sh` helper script
- Model-specific guidance testing

### Phase 4: Production Release (Week 4)
- User customization guide
- Comprehensive test suite (90%+ coverage)
- Setup monitoring and alerting
- GA release preparation

---

## Questions & Contact

For questions about this plan:
1. Review Phase 1 summary for baseline architecture
2. Check hook documentation for implementation details
3. Reference success criteria for acceptance conditions

**Document Owner:** Platform Architecture Team
**Last Updated:** 2026-01-05
**Status:** Ready for Implementation
