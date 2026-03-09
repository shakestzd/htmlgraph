# Phase 2 Feature 1: Smart Delegation Suggestions - Executive Summary

**Status:** Ready for Implementation
**Effort:** 1-2 weeks
**Priority:** High (drives delegation adoption)

---

## Problem Statement

**Current State:**
- Orchestrator mode blocks operations AFTER users have already done the work
- Users hit violations and feel punished for doing direct work
- No proactive guidance on when/how to delegate

**Impact:**
- Low delegation adoption rate
- Frustrating user experience
- Users disable orchestrator mode to avoid violations

---

## Proposed Solution

### Proactive Delegation Suggestions

Show smart suggestions BEFORE patterns become violations:

```
⚠️ ORCHESTRATOR: You've read 5 files exploring the auth system.

SUGGESTED DELEGATION:
Task(
    prompt="""
    Analyze the authentication system and document:
    1. All authentication endpoints
    2. Token flow and refresh logic
    3. Integration points with other modules

    Return a structured summary.
    """,
    subagent_type="general-purpose"
)

Delegation saves ~3-5k tokens of context.

[Y]es, run  [N]o, continue  [A]lways delegate exploration  [?]Learn more
```

### Key Features

1. **Pattern Detection** - Identifies delegation-worthy patterns:
   - Exploration (Read/Grep sequences)
   - Implementation (Edit/Write sequences)
   - Debugging (test failures + fixes)
   - Refactoring (multiple file edits)

2. **Contextual Suggestions** - Generates Task() calls with:
   - Specific prompts based on user's work
   - File paths and context from history
   - Appropriate subagent type
   - Estimated context savings

3. **User Preferences** - Learns and respects user choices:
   - "Always delegate exploration" → auto-delegates
   - "Never suggest debugging" → stops suggesting
   - Tracks acceptance rate for learning

4. **Analytics** - Measures effectiveness:
   - Suggestion acceptance rate
   - Context savings per session
   - Pattern frequency
   - User adoption trends

---

## Architecture

### Components

```
PreToolUse Hook
├── PatternDetector       # Detects delegation patterns
├── SuggestionEngine      # Generates Task() suggestions
├── PreferenceManager     # Stores/retrieves user preferences
└── ResponseFormatter     # Shows interactive prompts
```

### Data Flow

```
1. User executes tool call (e.g., Read)
   ↓
2. PatternDetector analyzes tool history
   ↓
3. Pattern detected? (exploration, implementation, etc.)
   ↓
4. Check PreferenceManager
   ├─ "Always" set? → Auto-delegate
   ├─ "Never" set? → Skip suggestion
   └─ No preference? → Show suggestion
   ↓
5. SuggestionEngine generates Task() code
   ↓
6. Show interactive prompt [Y/N/A/?]
   ↓
7. Record user response in database
   ↓
8. Continue or execute Task()
```

---

## Implementation Plan

### Phase 1: Pattern Detection (Days 1-3)
- Implement PatternDetector class
- Add exploration, implementation, debugging, refactoring patterns
- Write comprehensive unit tests (>90% coverage)

### Phase 2: Suggestion Engine (Days 4-6)
- Implement SuggestionEngine class
- Generate contextual Task() calls for each pattern
- Estimate token savings

### Phase 3: Preference Management (Days 7-8)
- Implement PreferenceManager class
- Add database schema for preferences
- Track acceptance rate

### Phase 4: Hook Integration (Days 9-10)
- Integrate into PreToolUse hook
- Add interactive prompts
- Wire up auto-delegation

### Phase 5: Response Formatting (Days 11-12)
- Rich formatting for suggestions
- "Learn more" documentation
- Interactive prompt handling

### Phase 6: Testing & Refinement (Days 13-14)
- Full test suite (>90% coverage)
- Edge case handling
- Documentation

---

## Success Metrics

### Functional
- ✅ All 4 pattern types detected (exploration, implementation, debugging, refactoring)
- ✅ Suggestion acceptance rate >40%
- ✅ Violation reduction by 30%
- ✅ User preferences persist correctly

### Technical
- ✅ Pattern detection <100ms
- ✅ Test coverage >90%
- ✅ No regressions in existing tests
- ✅ Database migrations work cleanly

### User Experience
- ✅ Suggestions are proactive (before violations)
- ✅ Task() code is contextual and actionable
- ✅ Interactive prompts are intuitive
- ✅ "Always/Never" preferences respected

---

## Example Use Cases

### Use Case 1: Exploration

**User Activity:**
```
Read auth/routes.py
Read auth/middleware.py
Read auth/models.py
Read auth/utils.py
Read auth/tokens.py
```

**Suggestion Shown:**
```
⚠️ ORCHESTRATOR: You've read 5 files exploring the auth system.

SUGGESTED DELEGATION:
Task(
    prompt="""
    Analyze the authentication system and document:
    1. All authentication endpoints
    2. Token flow and refresh logic
    3. Integration points with other modules

    🔴 CRITICAL - Report Results:
    from htmlgraph import SDK
    sdk = SDK(agent='explorer')
    sdk.spikes.create('Auth Analysis') \
        .set_findings('Summary...') \
        .save()
    """,
    subagent_type="general-purpose"
)

[Y]es, run  [N]o, continue  [A]lways delegate exploration
```

**User Selects [A]lways:**
- Preference stored: "always delegate exploration"
- Future exploration patterns auto-delegate
- No more prompts for exploration work

### Use Case 2: Implementation

**User Activity:**
```
Edit src/api/routes.py
Edit src/models/user.py
Edit tests/test_api.py
```

**Suggestion Shown:**
```
⚠️ ORCHESTRATOR: You're implementing changes across 3 files.

SUGGESTED DELEGATION:
Task(
    prompt="""
    Implement the required changes to:
    - src/api/routes.py
    - src/models/user.py
    - tests/test_api.py

    Run tests to verify.
    """,
    subagent_type="general-purpose"
)

[Y]es, run  [N]o, continue  [A]lways delegate implementation
```

**User Selects [N]o:**
- Preference not stored (one-time rejection)
- User continues with direct work
- Suggestion may appear again in future

### Use Case 3: Debugging

**User Activity:**
```
Bash: pytest tests/test_auth.py  # Exit code 1
Read tests/test_auth.py
Edit src/auth/utils.py
```

**Suggestion Shown:**
```
⚠️ ORCHESTRATOR: Test failures detected. Debugging cycles waste context.

SUGGESTED DELEGATION:
Task(
    prompt="""
    Debug and fix test failures from: pytest tests/test_auth.py

    Tasks:
    1. Run tests and capture full error output
    2. Identify root cause
    3. Implement fix
    4. Verify all tests pass
    """,
    subagent_type="general-purpose"
)

[Y]es, run  [N]o, continue  [Never] suggest debugging
```

**User Selects [Never]:**
- Preference stored: "never suggest debugging"
- Future debugging patterns ignored
- User prefers to debug directly

---

## Benefits

### For Users
1. **Proactive Guidance** - Learn when to delegate before violations
2. **Context Awareness** - Suggestions reflect actual work being done
3. **Educational** - See proper delegation patterns in action
4. **Flexible** - Set preferences to match workflow
5. **Less Friction** - Avoid violation blocks with smart delegation

### For HtmlGraph
1. **Higher Adoption** - Users embrace delegation patterns
2. **Better Metrics** - Track delegation effectiveness
3. **Learning Data** - Understand which patterns work best
4. **Reduced Support** - Fewer complaints about orchestrator mode
5. **Competitive Edge** - Unique proactive suggestion system

---

## Risks & Mitigations

### Risk: Suggestion Fatigue
**Mitigation:**
- Add cooldown (max 1 suggestion per 5 minutes)
- Respect "Never" preferences immediately
- Only suggest at confidence >= 0.5

### Risk: Inaccurate Patterns
**Mitigation:**
- Start with conservative thresholds
- Track acceptance rate per pattern type
- Refine thresholds based on user feedback

### Risk: Performance Impact
**Mitigation:**
- Pattern detection must be <100ms
- Cache recent tool history
- Use efficient database queries

### Risk: User Ignores Suggestions
**Mitigation:**
- Still enforce violations if pattern continues
- Note in violation message: "Suggestion was provided earlier"
- Track ignore rate and adjust strategy

---

## Open Questions

1. **Should suggestions show in guidance mode?**
   - YES - Educational value even without enforcement

2. **How to handle rapid-fire patterns?**
   - Add cooldown to avoid spam
   - Collapse similar suggestions within 5 minutes

3. **What confidence threshold for suggestions?**
   - Start with 0.5 (medium confidence)
   - Adjust based on acceptance rate

4. **Should we track which suggestions are copy-pasted?**
   - YES - Measure if users actually use the Task() code
   - Track via clipboard or subsequent Task() calls

5. **How to measure "educational value"?**
   - Survey users after 2 weeks
   - Track delegation adoption rate over time
   - Monitor violation frequency trends

---

## Next Steps

1. **Review Specification** - Get team/stakeholder feedback
2. **Finalize Architecture** - Confirm component design
3. **Create Feature Branch** - `feature/smart-delegation-suggestions`
4. **Implement Phase 1** - Pattern detection + tests
5. **Iterate** - Get early feedback, adjust thresholds
6. **Deploy** - Roll out with feature flag
7. **Monitor** - Track acceptance rate and user feedback

---

## References

- [Full Implementation Spec](PHASE2_FEATURE1_IMPLEMENTATION_SPEC.md)
- [Orchestrator Mode Docs](.claude/rules/orchestration.md)
- [Database Schema](src/python/htmlgraph/db/schema.py)
- [Event Tracking](src/python/htmlgraph/hooks/event_tracker.py)
