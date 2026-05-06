# Phase 3: Quick Reference Guide

## Quick Start

When Phase 2 is complete, execute Phase 3 with:

```bash
# 1. Verify Phase 2 complete
ls -1 src/python/wipnote/api/repositories/*.py | wc -l  # Expected: 5
ls -1 src/python/wipnote/api/services/*.py | wc -l      # Expected: 4

# 2. Run Phase 3a: Unit Tests
cd /Users/shakes/DevProjects/htmlgraph
uv run pytest tests/unit/api/ -v --tb=short

# 3. Check coverage
uv run pytest tests/unit/api/ --cov=src/python/wipnote/api --cov-report=term-missing

# 4. Run full test suite
uv run pytest --tb=short
```

---

## Key Documents (Read in Order)

1. **PHASE3_READINESS_SUMMARY.md** (Start here - 5 min read)
   - Overview of what's prepared
   - Next steps
   - Timeline

2. **PHASE3_TESTING_PLAN.md** (10 min read)
   - Testing strategy
   - Test types and coverage
   - Performance benchmarks

3. **PHASE3_ACCEPTANCE_CRITERIA.md** (15 min read)
   - 6 measurable criteria
   - Verification steps
   - Success metrics

4. **PHASE3_EXECUTION_GUIDE.md** (Detailed reference)
   - Step-by-step tasks
   - Bash commands
   - Troubleshooting

---

## Test Files Location

```
tests/unit/api/
├── conftest.py              ← Test fixtures & factories
├── test_repositories.py     ← Repository tests (~24 tests)
└── test_services.py         ← Service tests (~19 tests)
```

Run with:
```bash
uv run pytest tests/unit/api/ -v
```

---

## Acceptance Criteria at a Glance

| # | Criterion | Target | How to Verify |
|---|-----------|--------|---------------|
| 1 | Code Organization | main.py <500 lines | `wc -l src/python/wipnote/api/main.py` |
| 2 | API Compatibility | 100% shape match | Integration tests |
| 3 | Database Arch | 100% FastSQLA | `grep -r "aiosqlite" src/` (only in repos) |
| 4 | Caching | >80% hit rate | Performance tests |
| 5 | Query Performance | 25%+ improvement | Benchmark tests |
| 6 | WebSocket | <30ms latency | Performance tests |

---

## Success Criteria (All Must Pass)

```
✅ All 2665+ tests passing
✅ Code coverage >85%
✅ 0 type errors (mypy)
✅ 0 lint warnings (ruff)
✅ Performance targets met (25%+ faster)
✅ 100% backward compatibility
✅ All manual tests complete
✅ QA sign-off obtained
```

Then: Ready for v0.29.0 deployment

---

## Phase Breakdown

### Phase 3a: Unit Tests (2-3 days)
- Run conftest.py to verify fixtures
- Run test_repositories.py
- Run test_services.py
- Target: 85%+ coverage

**Command**: `uv run pytest tests/unit/api/ -v`

### Phase 3b: Integration Tests (2-3 days)
- Implement endpoint tests
- Implement cache tests
- Verify backward compatibility

**Command**: `uv run pytest tests/integration/api/ -v`

### Phase 3c: Manual Testing (2-3 days)
- Dashboard UI testing
- Performance measurement
- Query analysis
- WebSocket testing

**Command**: See PHASE3_EXECUTION_GUIDE.md

### Phase 3d: Code Quality (1 day)
- Full test suite
- Type checking
- Linting
- Coverage analysis

**Commands**:
```bash
uv run pytest
uv run mypy src/
uv run ruff check --fix && uv run ruff format
```

### Phase 3e: Documentation (1 day)
- Document results
- Create deployment checklist
- Release notes
- QA sign-off

---

## Performance Targets

**Before → After (Expected Improvement)**

| Operation | Before | Target | Improvement |
|-----------|--------|--------|-------------|
| Activity feed (100 events) | 200ms | <150ms | -25% |
| Agent stats | 150ms | <120ms | -20% |
| Orchestration chain | 250ms | <180ms | -28% |
| WebSocket message | 50ms | <30ms | -40% |
| Cache hit rate | N/A | >80% | - |

---

## Troubleshooting Quick Guide

### Tests Failing
1. Check Phase 2 complete: `ls src/python/wipnote/api/repositories/`
2. Check DB schema: `sqlite3 .wipnote/wipnote.db ".schema agent_events"`
3. Check fixtures work: `uv run pytest tests/unit/api/conftest.py -v`

### Coverage Below 85%
1. Generate report: `uv run pytest --cov=src/ --cov-report=html`
2. Review report: `open htmlcov/index.html`
3. Add tests for uncovered code

### Performance Not Improving
1. Enable query logging: `SQLALCHEMY_ECHO=true python -c "from wipnote.api.main import get_app; ..."`
2. Check for N+1 queries: `grep "SELECT" debug.log | wc -l` (should be small)
3. Verify cache working: Check logs for "Cache HIT"

### Type Errors
```bash
uv run mypy src/python/wipnote/api/
# Review and fix each error
# Add type hints where needed
```

### Lint Warnings
```bash
uv run ruff check src/
uv run ruff check --fix src/  # Auto-fix
```

---

## File Locations

**Test Files**:
- Fixtures: `tests/unit/api/conftest.py`
- Repository tests: `tests/unit/api/test_repositories.py`
- Service tests: `tests/unit/api/test_services.py`

**Documentation**:
- Readiness: `docs/api/PHASE3_READINESS_SUMMARY.md`
- Testing: `docs/api/PHASE3_TESTING_PLAN.md`
- Criteria: `docs/api/PHASE3_ACCEPTANCE_CRITERIA.md`
- Execution: `docs/api/PHASE3_EXECUTION_GUIDE.md`
- Reference: `docs/api/PHASE3_QUICK_REFERENCE.md` (this file)

**Production Code**:
- Repositories: `src/python/wipnote/api/repositories/`
- Services: `src/python/wipnote/api/services/`
- Main: `src/python/wipnote/api/main.py`

---

## Useful Commands

### Run All Tests
```bash
uv run pytest
```

### Run Only Unit Tests
```bash
uv run pytest tests/unit/api/ -v
```

### Generate Coverage Report
```bash
uv run pytest --cov=src/python/wipnote/api --cov-report=html
open htmlcov/index.html
```

### Type Check
```bash
uv run mypy src/
```

### Format Code
```bash
uv run ruff format src/
uv run ruff check --fix src/
```

### Monitor Tests During Development
```bash
uv run pytest tests/unit/api/ -v --tb=short --maxfail=1
```

### Run Specific Test
```bash
uv run pytest tests/unit/api/test_repositories.py::TestEventsRepository::test_create_event -v
```

---

## Expected Timeline

| Phase | Duration | Tasks | Status |
|-------|----------|-------|--------|
| 3a | 2-3 days | Unit tests | Ready |
| 3b | 2-3 days | Integration tests | Ready |
| 3c | 2-3 days | Manual testing | Ready |
| 3d | 1 day | Code quality | Ready |
| 3e | 1 day | Documentation | Ready |
| **Total** | **5-7 days** | **All** | **Ready** |

**Estimated Completion**: ~2026-02-10

---

## Deployment Checklist (Phase 3e)

```bash
# Before deploying, verify all:
[ ] All 2665+ tests passing: uv run pytest
[ ] Coverage >85%: uv run pytest --cov
[ ] Type errors 0: uv run mypy src/
[ ] Lint warnings 0: uv run ruff check src/
[ ] Performance targets met: See PHASE3_EXECUTION_GUIDE.md
[ ] Backward compatibility: Integration tests pass
[ ] Documentation complete: Review docs/api/
[ ] QA sign-off: Obtained
```

**Then**:
```bash
# Bump version
# Update CHANGELOG.md
# Create GitHub release
# Deploy to PyPI
```

---

## Key Files Created for Phase 3

✅ **Test Fixtures** (conftest.py)
- Database fixtures (sync/async)
- Repository fixtures
- Service fixtures
- Data factories (Event, Feature, Session)

✅ **Test Templates**
- Repository tests (~24 cases)
- Service tests (~19 cases)

✅ **Documentation**
- Testing Plan (strategy)
- Acceptance Criteria (6 criteria with targets)
- Execution Guide (step-by-step)
- Readiness Summary (overview)
- Quick Reference (this file)

---

## Support

For detailed information, see:
- Step-by-step tasks: `PHASE3_EXECUTION_GUIDE.md`
- Specific criteria: `PHASE3_ACCEPTANCE_CRITERIA.md`
- Testing approach: `PHASE3_TESTING_PLAN.md`
- Overall readiness: `PHASE3_READINESS_SUMMARY.md`

---

**Document Version**: 1.0
**Status**: Ready for Phase 3 Execution
**Next Milestone**: Phase 3 Start (After Phase 2 Complete)

---

## One-Line Summary

Phase 3 is fully prepared with test infrastructure, fixtures, templates, and execution guides. Ready to execute 5-7 day validation cycle upon Phase 2 completion. Expected to achieve 2665+ passing tests, 85%+ coverage, 25%+ performance improvement.
