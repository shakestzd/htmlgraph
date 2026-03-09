# Phase 2 Feature 1: Smart Delegation Suggestions - Implementation Checklist

**Created:** 2026-01-13
**Target Completion:** 2 weeks from start
**Status:** 🟡 Not Started

---

## Pre-Implementation

- [ ] Review specification with team
- [ ] Confirm architecture decisions
- [ ] Create feature branch: `feature/smart-delegation-suggestions`
- [ ] Set up test database fixtures

---

## Phase 1: Pattern Detection (Days 1-3)

### Files to Create
- [ ] `src/python/htmlgraph/orchestration/pattern_detector.py`
- [ ] `tests/python/test_pattern_detector.py`

### Implementation Tasks
- [ ] Implement `Pattern` dataclass
- [ ] Implement `PatternDetector.__init__()` with database connection
- [ ] Implement `PatternDetector.detect_pattern()` main entry point
- [ ] Implement `PatternDetector._load_tool_history()` from database
- [ ] Implement `PatternDetector._extract_file_paths()` helper

#### Pattern Type Implementations
- [ ] Implement `_detect_exploration()` (Read/Grep/Glob sequences)
  - [ ] High confidence (5+ calls in last 7)
  - [ ] Medium confidence (3-4 calls in last 5)
  - [ ] Low confidence (2 calls in last 3)

- [ ] Implement `_detect_implementation()` (Edit/Write sequences)
  - [ ] High confidence (3+ edits, 2+ files)
  - [ ] Medium confidence (2+ edits)
  - [ ] Low confidence (1 edit after reads)

- [ ] Implement `_detect_debugging()` (test failures + fixes)
  - [ ] High confidence (failed test + 2+ ops)
  - [ ] Medium confidence (failed test + 1 op)
  - [ ] Low confidence (single failed test)

- [ ] Implement `_detect_refactoring()` (multi-file edits)
  - [ ] High confidence (3+ edits, same files)
  - [ ] Medium confidence (2+ edits, related files)
  - [ ] Detection of related files via path similarity

### Unit Tests
- [ ] Test exploration pattern detection (high/medium/low confidence)
- [ ] Test implementation pattern detection (multiple files)
- [ ] Test debugging pattern detection (test failures)
- [ ] Test refactoring pattern detection (same file edits)
- [ ] Test no pattern detected below threshold
- [ ] Test pattern confidence scores are accurate
- [ ] Test file path extraction from context
- [ ] Test tool history loading from database
- [ ] Test edge cases (empty history, single tool call)
- [ ] Achieve >90% test coverage

### Validation
- [ ] All tests pass
- [ ] Pattern detection completes in <100ms
- [ ] Code passes ruff, mypy checks
- [ ] Manual testing with sample tool sequences

---

## Phase 2: Suggestion Engine (Days 4-6)

### Files to Create
- [ ] `src/python/htmlgraph/orchestration/suggestion_engine.py`
- [ ] `tests/python/test_suggestion_engine.py`

### Implementation Tasks
- [ ] Implement `Suggestion` dataclass
- [ ] Implement `SuggestionEngine.__init__()`
- [ ] Implement `SuggestionEngine.generate_suggestion()` dispatcher

#### Pattern-Specific Suggestions
- [ ] Implement `_suggest_exploration_delegation()`
  - [ ] Extract files explored from history
  - [ ] Infer exploration goal (module/directory)
  - [ ] Generate contextual Task() prompt
  - [ ] Add HtmlGraph spike reporting pattern

- [ ] Implement `_suggest_implementation_delegation()`
  - [ ] Extract files to edit
  - [ ] Infer implementation goal
  - [ ] Generate Task() with file list
  - [ ] Add test verification step

- [ ] Implement `_suggest_debugging_delegation()`
  - [ ] Extract failed test command
  - [ ] Generate debugging Task() prompt
  - [ ] Include test run + fix + verify steps

- [ ] Implement `_suggest_refactoring_delegation()`
  - [ ] Extract files to refactor
  - [ ] Generate refactoring Task() prompt
  - [ ] Include test verification

- [ ] Implement `_suggest_generic_delegation()` fallback

#### Helper Methods
- [ ] Implement `_infer_exploration_goal()` (from file paths)
- [ ] Implement `_infer_implementation_goal()` (from file names)
- [ ] Implement `_format_file_list()` (for Task() prompts)
- [ ] Implement `_estimate_token_savings()` (heuristic)

### Unit Tests
- [ ] Test exploration suggestion generation
- [ ] Test implementation suggestion generation
- [ ] Test debugging suggestion generation
- [ ] Test refactoring suggestion generation
- [ ] Test generic fallback suggestion
- [ ] Test Task() code is syntactically valid (compile check)
- [ ] Test prompts are contextual (contain file paths)
- [ ] Test subagent type selection is correct
- [ ] Test token savings estimation
- [ ] Test edge cases (empty history, no files)
- [ ] Achieve >85% test coverage

### Validation
- [ ] All tests pass
- [ ] Generated Task() code compiles without errors
- [ ] Prompts are clear and actionable
- [ ] Token savings estimates are reasonable
- [ ] Code passes ruff, mypy checks

---

## Phase 3: Preference Management (Days 7-8)

### Files to Create/Modify
- [ ] `src/python/htmlgraph/orchestration/preference_manager.py` (NEW)
- [ ] `src/python/htmlgraph/db/schema.py` (MODIFY - add tables)
- [ ] `tests/python/test_preference_manager.py` (NEW)

### Database Schema
- [ ] Add `delegation_preferences` table
  - [ ] Columns: id, session_id, pattern_type, action, confidence, timestamp
  - [ ] Foreign key to sessions table
  - [ ] Check constraint on action (accepted/rejected/always/never)
  - [ ] Check constraint on pattern_type

- [ ] Add `delegation_suggestions` table
  - [ ] Columns: id, session_id, pattern_type, confidence, suggestion_text, user_action, timestamp
  - [ ] Foreign key to sessions table
  - [ ] Track all suggestions shown

- [ ] Add indexes
  - [ ] `idx_delegation_preferences_session`
  - [ ] `idx_delegation_suggestions_session`

- [ ] Test database migration (existing DB → new schema)

### Implementation Tasks
- [ ] Implement `PreferenceAction` enum
- [ ] Implement `PreferenceManager.__init__()` with database
- [ ] Implement `PreferenceManager._ensure_preferences_table()`
- [ ] Implement `PreferenceManager.should_suggest()` (check "never")
- [ ] Implement `PreferenceManager.should_auto_delegate()` (check "always")
- [ ] Implement `PreferenceManager.record_action()` (store preference)
- [ ] Implement `PreferenceManager.get_acceptance_rate()` (calculate rate)

### Unit Tests
- [ ] Test preference storage and retrieval
- [ ] Test "never" preference blocks suggestions
- [ ] Test "always" preference enables auto-delegation
- [ ] Test acceptance rate calculation (0%, 50%, 100%)
- [ ] Test preferences persist across sessions
- [ ] Test database foreign key constraints work
- [ ] Test edge cases (no preferences, multiple sessions)
- [ ] Achieve >90% test coverage

### Validation
- [ ] All tests pass
- [ ] Database migrations work cleanly
- [ ] Preferences persist after database close/reopen
- [ ] Code passes ruff, mypy checks
- [ ] Foreign key constraints enforced

---

## Phase 4: Hook Integration (Days 9-10)

### Files to Modify
- [ ] `src/python/htmlgraph/hooks/orchestrator.py` (integrate suggestion engine)
- [ ] `src/python/htmlgraph/hooks/pretooluse.py` (add suggestion check)

### Files to Create
- [ ] `tests/python/test_suggestion_integration.py` (end-to-end tests)

### Implementation Tasks

#### Orchestrator.py Integration
- [ ] Add `from orchestration.pattern_detector import PatternDetector`
- [ ] Add `from orchestration.suggestion_engine import SuggestionEngine`
- [ ] Add `from orchestration.preference_manager import PreferenceManager`
- [ ] Modify `enforce_orchestrator_mode()` to check for patterns
- [ ] Add pattern detection before existing violation checks
- [ ] Add preference check (auto-delegate if "always" set)
- [ ] Add suggestion display if pattern detected
- [ ] Add user response handling (Y/N/A/?)

#### PreToolUse.py Integration
- [ ] Add suggestion check before tool execution
- [ ] Call `PatternDetector.detect_pattern()` for current tool
- [ ] If pattern detected, call `SuggestionEngine.generate_suggestion()`
- [ ] Check `PreferenceManager` for auto-delegation
- [ ] Display interactive prompt
- [ ] Record user response

#### Interactive Prompt Handling
- [ ] Implement [Y]es handler (execute Task())
- [ ] Implement [N]o handler (continue with tool)
- [ ] Implement [A]lways handler (set preference + execute Task())
- [ ] Implement [?] handler (show "Learn more" documentation)
- [ ] Handle invalid input gracefully

### Integration Tests
- [ ] Test end-to-end flow (pattern → suggestion → response)
- [ ] Test auto-delegation with "always" preference
- [ ] Test suggestion blocked by "never" preference
- [ ] Test acceptance rate tracking
- [ ] Test Task() execution on [Y]es response
- [ ] Test preference storage on [A]lways response
- [ ] Test multiple pattern types in same session
- [ ] Test cooldown prevents suggestion spam
- [ ] Achieve >85% test coverage

### Validation
- [ ] All tests pass
- [ ] Suggestions show before violations
- [ ] Interactive prompts work correctly
- [ ] Preferences are respected
- [ ] No regressions in existing orchestrator mode tests
- [ ] Code passes ruff, mypy checks

---

## Phase 5: Response Formatting (Days 11-12)

### Files to Create
- [ ] `src/python/htmlgraph/orchestration/formatters.py`
- [ ] `tests/python/test_formatters.py`

### Implementation Tasks
- [ ] Implement `format_suggestion()` (rich formatting)
- [ ] Implement `format_interactive_prompt()` ([Y/N/A/?])
- [ ] Implement `format_learn_more()` (documentation)
- [ ] Implement `format_token_savings()` (visual indicator)
- [ ] Add colors/styling using rich library
- [ ] Add visual hierarchy (warning icon, code blocks)
- [ ] Make Task() code easy to copy-paste

### Unit Tests
- [ ] Test suggestion formatting (visual output)
- [ ] Test interactive prompt formatting
- [ ] Test "Learn more" documentation content
- [ ] Test token savings display
- [ ] Test formatting with various suggestion types
- [ ] Test edge cases (very long prompts, missing data)
- [ ] Achieve >85% test coverage

### Validation
- [ ] All tests pass
- [ ] Suggestions are visually clear and readable
- [ ] Interactive prompts are intuitive
- [ ] "Learn more" is helpful and concise
- [ ] Code passes ruff, mypy checks
- [ ] Manual visual inspection looks good

---

## Phase 6: Testing & Refinement (Days 13-14)

### Full Test Suite
- [ ] Run all unit tests: `uv run pytest tests/python/test_pattern_detector.py`
- [ ] Run all unit tests: `uv run pytest tests/python/test_suggestion_engine.py`
- [ ] Run all unit tests: `uv run pytest tests/python/test_preference_manager.py`
- [ ] Run all integration tests: `uv run pytest tests/python/test_suggestion_integration.py`
- [ ] Run all formatting tests: `uv run pytest tests/python/test_formatters.py`
- [ ] Run full test suite: `uv run pytest`
- [ ] Check test coverage: `uv run pytest --cov=src/python/htmlgraph/orchestration --cov-report=term-missing`
- [ ] Achieve >90% overall coverage

### Code Quality
- [ ] Fix all ruff errors: `uv run ruff check --fix`
- [ ] Format code: `uv run ruff format`
- [ ] Fix all mypy errors: `uv run mypy src/python/htmlgraph/orchestration/`
- [ ] Review code for clarity and maintainability

### Edge Case Handling
- [ ] Test with empty tool history
- [ ] Test with single tool call (no pattern)
- [ ] Test with rapid-fire tool calls
- [ ] Test with database unavailable (graceful degradation)
- [ ] Test with corrupted preference data
- [ ] Test with very long file paths
- [ ] Test with Unicode in file names

### Threshold Tuning
- [ ] Review confidence thresholds (0.5/0.7/0.9)
- [ ] Adjust based on false positive rate
- [ ] Test with real-world tool sequences
- [ ] Get feedback from test users

### Performance Testing
- [ ] Benchmark pattern detection (<100ms target)
- [ ] Benchmark suggestion generation (<50ms target)
- [ ] Benchmark database queries (<20ms target)
- [ ] Optimize if needed

### Documentation
- [ ] Add docstrings to all public methods
- [ ] Add inline comments for complex logic
- [ ] Update AGENTS.md with suggestion feature
- [ ] Update CLAUDE.md with usage examples
- [ ] Create migration guide for existing users

### Final Validation
- [ ] Manual end-to-end testing
- [ ] Test in development mode: `uv run htmlgraph claude --dev`
- [ ] Verify suggestions show correctly
- [ ] Verify preferences persist
- [ ] Verify auto-delegation works
- [ ] Get feedback from early testers

---

## Deployment Preparation

### Pre-Deployment Checklist
- [ ] All tests pass with >90% coverage
- [ ] Code quality checks pass (ruff, mypy)
- [ ] Documentation complete
- [ ] Database migrations tested
- [ ] No regressions in existing functionality

### Version & Release
- [ ] Bump version in `pyproject.toml`
- [ ] Bump version in `src/python/htmlgraph/__init__.py`
- [ ] Bump version in `packages/claude-plugin/.claude-plugin/plugin.json`
- [ ] Update CHANGELOG.md
- [ ] Create git tag: `git tag v0.X.Y`

### Deployment
- [ ] Run deployment script: `./scripts/deploy-all.sh 0.X.Y --no-confirm`
- [ ] Verify PyPI publication
- [ ] Test fresh install: `pip install htmlgraph==0.X.Y`
- [ ] Update plugin: `claude plugin update htmlgraph`
- [ ] Smoke test in production environment

### Post-Deployment
- [ ] Monitor error logs for issues
- [ ] Track suggestion acceptance rate
- [ ] Gather user feedback
- [ ] Plan refinements for next iteration

---

## Metrics to Track

### Functional Metrics
- [ ] Suggestion acceptance rate (target: >40%)
- [ ] Violation reduction (target: 30% fewer)
- [ ] Auto-delegation usage (% of sessions)
- [ ] Pattern detection accuracy (vs. manual review)

### Performance Metrics
- [ ] Pattern detection time (target: <100ms)
- [ ] Suggestion generation time (target: <50ms)
- [ ] Database query time (target: <20ms)
- [ ] Memory usage (should not increase significantly)

### User Experience Metrics
- [ ] User feedback (qualitative)
- [ ] Feature adoption rate (% of users)
- [ ] Preference usage ("always" vs. "never")
- [ ] "Learn more" click-through rate

---

## Known Issues & TODOs

### Phase 3+ Enhancements (Future)
- [ ] ML-based pattern detection (replace rule-based)
- [ ] LLM-generated Task() prompts (more contextual)
- [ ] Cross-session learning (aggregate patterns)
- [ ] Dashboard analytics for delegation metrics
- [ ] Pattern frequency heatmaps
- [ ] User delegation adoption trends graph

### Open Questions
- [ ] Should suggestions show in guidance mode? (YES)
- [ ] What confidence threshold? (Start with 0.5)
- [ ] How to handle rapid suggestions? (Cooldown: 5 min)
- [ ] Track copy-paste of Task() code? (Future)
- [ ] Measure educational value? (Survey after 2 weeks)

---

## Sign-Off

### Implementation Complete
- [ ] All checklist items completed
- [ ] All tests pass (>90% coverage)
- [ ] Code quality checks pass
- [ ] Documentation complete
- [ ] Deployed to production

### Sign-Off
- [ ] Developer: _________________ Date: _______
- [ ] Reviewer: _________________ Date: _______
- [ ] Product Owner: _________________ Date: _______

---

## Notes

(Add notes, issues, decisions made during implementation here)

---

**Last Updated:** 2026-01-13
**Status:** 🟡 Not Started → 🟢 Complete
