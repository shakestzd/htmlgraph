# CIGS SessionStart Hook Integration

**Status:** ✅ Complete
**Date:** 2026-01-04
**Component:** SessionStart Hook Enhancement

---

## Overview

The Computational Imperative Guidance System (CIGS) has been integrated into the SessionStart hook to provide personalized, adaptive delegation guidance based on historical behavior and detected patterns.

## What Was Implemented

### 1. Hook Integration (`packages/claude-plugin/hooks/scripts/session-start.py`)

**Added Components:**
- Import CIGS modules: `ViolationTracker`, `PatternDetector`, `AutonomyRecommender`
- New function: `get_cigs_context(graph_dir: Path, session_id: str) -> str`
- Context injection into SessionStart output

**Key Features:**
- Loads violation history from last 5 sessions
- Detects anti-patterns using PatternDetector
- Recommends autonomy level based on compliance
- Generates personalized delegation reminders
- Injects CIGS status into session context

### 2. CIGS Context Structure

The context includes:

```markdown
## 🧠 CIGS Status (Computational Imperative Guidance System)

**Autonomy Level:** {OBSERVER|CONSULTANT|COLLABORATOR|OPERATOR}
**Messaging Intensity:** {minimal|moderate|high|maximal}
**Enforcement Mode:** {guidance|strict}

**Reason:** {Why this level was chosen}

### Session Violations (if any)
- Total violations: X
- Compliance rate: Y%
- Wasted tokens: Z

### Detected Anti-Patterns (if any)
- **pattern_name**: description
  - Fix: delegation_suggestion

### Delegation Reminders
{Personalized reminders based on autonomy level}
```

### 3. Autonomy Levels

| Level | Compliance | Messaging | When Applied |
|-------|-----------|-----------|--------------|
| **Observer** | >90% | Minimal guidance | Excellent compliance, no anti-patterns |
| **Consultant** | 70-90% | Moderate | Good compliance, 1-2 anti-patterns |
| **Collaborator** | 50-70% | High guidance | Moderate compliance, 3+ anti-patterns |
| **Operator** | <50% | Maximal (strict) | Low compliance or circuit breaker |

### 4. Personalized Reminders

**Observer Mode:**
```
💡 MINIMAL GUIDANCE - You're doing well!
Continue delegating as appropriate. Guidance will escalate if patterns change.
```

**Consultant Mode:**
```
🔴 MODERATE GUIDANCE - Remember delegation patterns:
- Multi-file exploration → spawn_gemini()
- Code changes with tests → Task() or spawn_codex()
- Git operations → spawn_copilot()
```

**Collaborator Mode:**
```
⚠️ ACTIVE GUIDANCE - Focus on delegation:
- Exploration: Use spawn_gemini() (FREE)
- Code changes: Use spawn_codex() or Task()
- Git operations: Use spawn_copilot()

Direct tool use should be rare and well-justified.
```

**Operator Mode:**
```
🚨 STRICT MODE ACTIVE - You MUST delegate ALL operations except:
- Task() - Delegation itself
- AskUserQuestion() - User clarification
- TodoWrite() - Work tracking
- SDK operations - Feature/session management

ALL other operations MUST be delegated to subagents.
```

## How It Works

### Data Flow

1. **SessionStart Hook Fires**
   - SessionManager creates/resumes session
   - Hook calls `get_cigs_context(graph_dir, session_id)`

2. **CIGS Context Generation**
   ```python
   # Initialize tracker
   tracker = ViolationTracker(graph_dir)
   tracker.set_session_id(session_id)

   # Load recent violations (last 5 sessions)
   recent_violations = tracker.get_recent_violations(sessions=5)

   # Detect patterns
   detector = PatternDetector()
   patterns = detector.detect_all_patterns(history)

   # Recommend autonomy level
   recommender = AutonomyRecommender()
   autonomy = recommender.recommend(violations, patterns, compliance_history)

   # Build context
   context = build_cigs_status(autonomy, violations, patterns)
   ```

3. **Context Injection**
   - CIGS context inserted after "HTMLGRAPH PROCESS NOTICE"
   - Appears before "ORCHESTRATOR MODE" section
   - Visible to Claude at session start

### Storage

**Violation Records:** `.wipnote/cigs/violations.jsonl`
```jsonl
{"id":"viol-001","session_id":"sess-abc","tool":"Read","violation_type":"direct_exploration",...}
```

**Pattern Analysis:** In-memory (future: persist to `.wipnote/cigs/patterns.json`)

**Session Summaries:** In-memory (future: persist to `.wipnote/cigs/session-summaries/`)

## Testing

### Test Coverage

**File:** `tests/python/test_session_start_cigs.py`

**Tests:**
1. ✅ ViolationTracker initialization
2. ✅ Record violation and retrieve summary
3. ✅ Pattern detection (exploration sequences)
4. ✅ Autonomy recommender (strict mode)
5. ✅ Autonomy recommender (observer mode)
6. ✅ CIGS context with no violations
7. ✅ CIGS context with violations and patterns
8. ✅ CIGS context format validation
9. ✅ Hook output format
10. ✅ Context includes autonomy level

**All 10 tests pass + 113 existing CIGS tests pass**

### Manual Verification

```bash
# Test hook execution
python3 packages/claude-plugin/hooks/scripts/session-start.py < /dev/null

# Output includes:
## 🧠 CIGS Status (Computational Imperative Guidance System)
**Autonomy Level:** OBSERVER
**Messaging Intensity:** minimal
...
```

## Integration Points

### With Existing Systems

1. **ViolationTracker**
   - Already implemented in `src/python/wipnote/cigs/tracker.py`
   - Stores violations in JSONL format
   - Thread-safe access

2. **PatternDetector**
   - Already implemented in `src/python/wipnote/cigs/patterns.py`
   - Detects 4 anti-patterns: exploration_sequence, edit_without_test, direct_git_commit, repeated_read_same_file

3. **AutonomyRecommender**
   - Already implemented in `src/python/wipnote/cigs/autonomy.py`
   - Implements 4-level decision matrix

### With Future Hooks

**Ready for:**
- PreToolUse hook: Can read autonomy level and violation count
- PostToolUse hook: Can record violations and update metrics
- Stop hook: Can generate session summary with CIGS analytics

## Configuration

### Environment Variables

- `HTMLGRAPH_DISABLE_TRACKING=1` - Disables entire hook (including CIGS)
- `HTMLGRAPH_SESSION_ID` - Overrides session ID detection

### Default Behavior

- **Enabled by default** when Wipnote plugin is installed
- **Observer mode** for new sessions (no violation history)
- **Graceful degradation** if CIGS modules unavailable (warning logged, hook continues)

## Future Enhancements

### Phase 2 (Next)
- PreToolUse hook integration
- PostToolUse hook integration
- Cost tracking and reporting

### Phase 3
- Pattern persistence to JSON
- Session summary persistence
- Cross-session analytics

### Phase 4
- Dashboard integration
- CLI commands for CIGS status
- User-configurable thresholds

## Usage Example

### New Session (No History)
```
## 🧠 CIGS Status
Autonomy Level: OBSERVER
Messaging Intensity: minimal
Enforcement Mode: guidance

Reason: Excellent compliance (100%). Minimal guidance needed.

### Delegation Reminders
💡 MINIMAL GUIDANCE - You're doing well!
```

### After 3 Violations
```
## 🧠 CIGS Status
Autonomy Level: COLLABORATOR
Messaging Intensity: high
Enforcement Mode: strict

Reason: Moderate compliance (60%), 2 anti-pattern(s). Active guidance needed.

### Session Violations
- Total violations: 3
- Compliance rate: 60%
- Wasted tokens: 12500
- ⚠️ Circuit breaker active (3+ violations)

### Detected Anti-Patterns
- exploration_sequence: Multiple exploration tools in sequence
  - Fix: spawn_gemini(prompt='Comprehensive search and analysis...')

### Delegation Reminders
⚠️ ACTIVE GUIDANCE - Focus on delegation:
- Exploration: Use spawn_gemini() (FREE)
- Code changes: Use spawn_codex() or Task()
```

## References

- **Design Document:** `.wipnote/spikes/computational-imperative-guidance-system-design.md`
- **CIGS Modules:** `src/python/wipnote/cigs/`
- **Hook Script:** `packages/claude-plugin/hooks/scripts/session-start.py`
- **Tests:** `tests/python/test_session_start_cigs.py`
- **User Guide:** `docs/CIGS_USER_GUIDE.md`

---

**Integration Status: Complete ✅**

Next: PreToolUse hook integration for real-time imperative guidance.
