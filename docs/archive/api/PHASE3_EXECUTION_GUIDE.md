# Phase 3: Execution Guide - Tests & Acceptance Criteria

## Overview

Phase 3 validates the API refactor completed in Phase 2 through comprehensive testing and manual verification. This guide provides step-by-step instructions for executing Phase 3.

**Timeline**: 5-7 days
**Status**: Ready to Start (Phase 2 must be complete first)
**Owner**: QA Team / Test Automation Engineer

---

## Pre-Execution Checklist

Before starting Phase 3, verify Phase 2 is complete:

```bash
# 1. Verify repositories exist
ls -1 src/python/wipnote/api/repositories/*.py
# Expected: base_repository.py, events_repository.py, features_repository.py, sessions_repository.py

# 2. Verify services exist
ls -1 src/python/wipnote/api/services/*.py
# Expected: base_service.py, activity_service.py, orchestration_service.py, analytics_service.py

# 3. Run existing tests to establish baseline
uv run pytest -xvs --tb=short
# Expected: All existing tests should pass
```

---

## Phase 3a: Unit Test Implementation (Days 1-2)

### Objective
Write comprehensive unit tests for repositories and services, achieving 85%+ coverage.

### Tasks

#### Task 3a.1: Create Test Infrastructure
**Duration**: 2-4 hours

```bash
# 1. Create test directories
mkdir -p tests/unit/api/
mkdir -p tests/integration/api/

# 2. Create conftest.py with fixtures (ALREADY DONE)
cp docs/api/test_fixtures.py tests/unit/api/conftest.py

# 3. Verify fixtures work
uv run pytest tests/unit/api/conftest.py -v
# Expected: 0 errors (conftest doesn't have tests, just fixtures)
```

#### Task 3a.2: Implement Repository Tests
**Duration**: 4-6 hours

```bash
# 1. Create repository test file (ALREADY DONE)
cp docs/api/test_repositories.py tests/unit/api/

# 2. Run repository tests
uv run pytest tests/unit/api/test_repositories.py -v

# 3. Check coverage
uv run pytest tests/unit/api/test_repositories.py --cov=src/python/wipnote/api/repositories

# Expected coverage targets:
# - EventsRepository: 85%+
# - FeaturesRepository: 85%+
# - SessionsRepository: 85%+
```

**Sample Output**:
```
tests/unit/api/test_repositories.py::TestEventsRepository::test_create_event PASSED
tests/unit/api/test_repositories.py::TestEventsRepository::test_find_event_by_id PASSED
tests/unit/api/test_repositories.py::TestEventsRepository::test_find_all_events_with_pagination PASSED
...
===== 20 passed in 2.34s =====
```

#### Task 3a.3: Implement Service Tests
**Duration**: 4-6 hours

```bash
# 1. Create service test file (ALREADY DONE)
cp docs/api/test_services.py tests/unit/api/

# 2. Run service tests
uv run pytest tests/unit/api/test_services.py -v

# 3. Check coverage
uv run pytest tests/unit/api/test_services.py --cov=src/python/wipnote/api/services

# Expected coverage targets:
# - ActivityService: 90%+
# - OrchestrationService: 90%+
# - AnalyticsService: 90%+
```

#### Task 3a.4: Run Full Unit Test Suite
**Duration**: 1 hour

```bash
# 1. Run all unit tests
uv run pytest tests/unit/api/ -v

# 2. Generate coverage report
uv run pytest tests/unit/api/ --cov=src/python/wipnote/api \
  --cov-report=html --cov-report=term-missing

# 3. Review coverage report
# Expected: >85% coverage on repositories/ and services/
open htmlcov/index.html
```

**Success Criteria**:
- [ ] All unit tests passing (>30 tests)
- [ ] >85% coverage on repositories
- [ ] >90% coverage on services
- [ ] No flaky tests (run twice, should pass both times)

---

## Phase 3b: Integration Test Implementation (Days 2-3)

### Objective
Write integration tests validating endpoints and WebSocket functionality.

### Tasks

#### Task 3b.1: Create Integration Test Infrastructure
**Duration**: 2 hours

```bash
# 1. Create integration test fixtures
mkdir -p tests/integration/api/
touch tests/integration/api/__init__.py
touch tests/integration/api/conftest.py

# 2. Copy test fixtures
# (Would include FastAPI TestClient setup)
```

#### Task 3b.2: Implement Endpoint Tests
**Duration**: 6-8 hours

```bash
# 1. Create endpoint test file
touch tests/integration/api/test_endpoints.py

# 2. Tests to implement:
# - Activity endpoints (GET /api/events, GET /views/activity)
# - Orchestration endpoints (GET /api/orchestration)
# - Analytics endpoints (GET /api/analytics)
# - Features endpoints (GET /api/features)
# - Sessions endpoints (GET /api/sessions)

# 3. Run endpoint tests
uv run pytest tests/integration/api/test_endpoints.py -v

# Expected: >15 test cases
```

#### Task 3b.3: Implement Cache Behavior Tests
**Duration**: 4 hours

```bash
# 1. Create cache test file
touch tests/integration/api/test_cache_behavior.py

# 2. Tests to implement:
# - Cache hit tracking
# - Cache invalidation on data changes
# - TTL expiration
# - Cache performance impact

# 3. Run cache tests
uv run pytest tests/integration/api/test_cache_behavior.py -v
```

#### Task 3b.4: Verify Backward Compatibility
**Duration**: 2 hours

```bash
# 1. Record baseline responses
python tests/api/record_baseline.py

# 2. Compare current responses
python tests/api/compare_responses.py

# 3. Verify no breaking changes
# Expected: 100% response shape compatibility
```

**Success Criteria**:
- [ ] All integration tests passing (>20 tests)
- [ ] 100% endpoint backward compatibility
- [ ] Cache hit rates >80%
- [ ] No response shape changes

---

## Phase 3c: Manual Testing & Performance Validation (Days 4-5)

### Objective
Manually verify dashboard functionality and measure performance improvements.

### Tasks

#### Task 3c.1: UI Testing
**Duration**: 3-4 hours

**Manual Testing Checklist**:

1. **Dashboard Loading**
   - [ ] Dashboard loads without errors
   - [ ] All tabs render correctly (Activity, Agents, Features, Analytics)
   - [ ] No console errors (open DevTools, check console)
   - [ ] Page loads in <2 seconds

2. **Activity Feed**
   - [ ] Events display in correct order (newest first)
   - [ ] Pagination works (next/prev buttons)
   - [ ] Event details display correctly
   - [ ] Search/filter functionality works
   - [ ] Real-time updates appear (if WebSocket enabled)

3. **Data Display**
   - [ ] Agent names correct and match database
   - [ ] Event timestamps accurate
   - [ ] Status badges display correctly
   - [ ] Cost calculations accurate (if visible)

4. **Responsiveness**
   - [ ] Dashboard responsive on mobile
   - [ ] Tabs switch without lag
   - [ ] Scrolling smooth
   - [ ] No layout shifts

#### Task 3c.2: Performance Baseline Measurement
**Duration**: 4-6 hours

```bash
# 1. Start dashboard
uv run wipnote serve

# 2. Measure endpoint latencies (before refactor baseline)
python tests/benchmarks/measure_baseline.py

# Expected output:
# Activity feed (100 events): 200ms
# Agent stats: 150ms
# Orchestration chain: 250ms
# WebSocket cycle: 50ms

# 3. Record query counts (to detect N+1)
SQLALCHEMY_ECHO=true uv run wipnote serve 2>&1 | tee query_log.txt

# 4. Measure cache hit rates
curl -H "X-Cache-Metrics: true" http://localhost:8000/api/events
# Expected: X-Cache-Hit: true, X-Cache-Hit-Rate: >80%
```

**Performance Targets (After Refactor)**:
- Activity feed: <150ms (25% improvement)
- Agent stats: <120ms (20% improvement)
- Orchestration: <180ms (28% improvement)
- WebSocket: <30ms (40% improvement)

#### Task 3c.3: Database Query Analysis
**Duration**: 2-3 hours

```bash
# 1. Enable query logging
SQLALCHEMY_ECHO=true uv run wipnote serve 2>&1 | tee queries.log

# 2. Analyze queries for N+1 patterns
grep "SELECT" queries.log | wc -l
# Expected: Should be small number (1-3 per operation)

# 3. Check for connection reuse
grep "connected" queries.log
# Expected: Should see connection pool messages, not new connections

# 4. Verify transaction boundaries
grep "BEGIN\|COMMIT\|ROLLBACK" queries.log
# Expected: Proper transaction nesting
```

#### Task 3c.4: WebSocket Performance Testing
**Duration**: 2-3 hours

```bash
# 1. Connect WebSocket and monitor
python tests/websockets/monitor_performance.py

# 2. Expected metrics:
# - Connection latency: <100ms
# - Message latency: <30ms
# - Memory stable: <10MB growth over 1 hour
# - Reconnection time: <5 seconds

# 3. Load test (multiple simultaneous WebSocket connections)
python tests/websockets/load_test.py --connections 10

# Expected: All connections maintained, no dropped messages
```

**Success Criteria**:
- [ ] All manual UI tests pass
- [ ] Performance improvements measured and documented
- [ ] No N+1 queries detected
- [ ] Cache hit rates >80%
- [ ] WebSocket stable and responsive

---

## Phase 3d: Code Quality Gates (Day 5-6)

### Objective
Ensure code quality meets standards before deployment.

### Tasks

#### Task 3d.1: Run Full Test Suite
**Duration**: 2-3 hours

```bash
# 1. Run all tests
uv run pytest

# Expected: 2665+ tests passing
# Time: ~5-10 minutes

# 2. Check for any failures or warnings
# Expected: 0 failures, 0 warnings

# 3. Generate coverage report
uv run pytest --cov=src/ --cov-report=html

# Expected: >85% overall coverage
```

#### Task 3d.2: Type Checking
**Duration**: 1 hour

```bash
# 1. Run mypy
uv run mypy src/

# Expected: 0 errors

# 2. If errors found, fix them:
# - Add type hints to repositories
# - Add type hints to services
# - Ensure async/await properly typed
```

#### Task 3d.3: Linting & Code Formatting
**Duration**: 1 hour

```bash
# 1. Check code style
uv run ruff check src/

# Expected: 0 errors

# 2. Auto-fix issues
uv run ruff check --fix src/

# 3. Format code
uv run ruff format src/

# 4. Verify no changes needed
uv run ruff check src/
# Expected: 0 errors
```

#### Task 3d.4: Code Coverage Analysis
**Duration**: 1-2 hours

```bash
# 1. Generate detailed coverage report
uv run pytest --cov=src/python/wipnote/api/ \
  --cov=src/python/wipnote/db/ \
  --cov-report=html --cov-report=term-missing:skip-covered

# 2. Review coverage by module:
# - api/main.py: >80%
# - api/repositories/: >85%
# - api/services/: >90%
# - api/cache.py: >85%
# - db/: >80%

# 3. Identify untested code
# - Review uncovered lines
# - Add tests for critical paths
# - Document intentional gaps
```

**Success Criteria**:
- [ ] All 2665+ tests passing
- [ ] 0 type errors
- [ ] 0 lint warnings
- [ ] >85% code coverage
- [ ] All critical paths tested

---

## Phase 3e: Final Documentation & Sign-Off (Day 6-7)

### Tasks

#### Task 3e.1: Document Results
**Duration**: 2 hours

Create summary document:

```markdown
# Phase 3 Test Results Summary

## Execution Timeline
- Started: 2026-02-XX
- Completed: 2026-02-XX
- Duration: X days

## Test Results
- Unit Tests: NNN/NNN passed
- Integration Tests: NNN/NNN passed
- Manual Tests: All passed
- Performance Tests: All targets met

## Coverage Metrics
- Overall Coverage: XX%
- Repositories: XX%
- Services: XX%
- API Layer: XX%

## Performance Improvements
- Activity Feed: XX% improvement
- Agent Stats: XX% improvement
- Orchestration: XX% improvement
- WebSocket: XX% improvement

## Code Quality
- Type Errors: 0
- Lint Warnings: 0
- Flaky Tests: 0

## Known Issues / Tech Debt
[List any remaining issues]

## Sign-Off
- QA Lead: ___________
- Date: ___________
- Deployment Approved: [ ] Yes [ ] No
```

#### Task 3e.2: Create Deployment Checklist
**Duration**: 1 hour

```bash
# Create deployment_checklist.md with:
- All tests passing: [ ] Yes
- Coverage >85%: [ ] Yes
- Performance targets met: [ ] Yes
- No breaking changes: [ ] Yes
- Documentation updated: [ ] Yes
- Version bumped (0.29.0): [ ] Yes
- Changelog updated: [ ] Yes
- Ready for PyPI: [ ] Yes
```

#### Task 3e.3: Prepare Release Notes
**Duration**: 2 hours

```markdown
# Wipnote 0.29.0 Release Notes

## What's New
- API refactored with repository and service patterns
- Database layer now uses FastSQLA for better connection pooling
- HTTP caching improved with fastapi-cache2
- Performance improvements: 25-40% faster on complex queries

## Performance Improvements
- Activity feed: 200ms → 150ms (-25%)
- Agent stats: 150ms → 120ms (-20%)
- Orchestration: 250ms → 180ms (-28%)
- WebSocket: 50ms → 30ms (-40%)

## Breaking Changes
None - full backward compatibility maintained

## Testing
- 2665+ tests passing
- 85%+ code coverage
- All acceptance criteria met
```

---

## Execution Checklist

### Pre-Phase 3
- [ ] Phase 2 completely finished
- [ ] All Phase 2 tests passing
- [ ] Main branch clean (no uncommitted changes)
- [ ] Task tracking updated

### Phase 3a: Unit Tests
- [ ] Test infrastructure created
- [ ] Repository tests implemented
- [ ] Service tests implemented
- [ ] 85%+ coverage achieved
- [ ] All unit tests passing

### Phase 3b: Integration Tests
- [ ] Integration test fixtures created
- [ ] Endpoint tests implemented
- [ ] Cache behavior tests implemented
- [ ] 100% backward compatibility verified
- [ ] All integration tests passing

### Phase 3c: Manual Testing
- [ ] UI testing checklist completed
- [ ] Performance baseline measured
- [ ] Database queries analyzed
- [ ] WebSocket performance verified
- [ ] All performance targets met

### Phase 3d: Code Quality
- [ ] Full test suite passing
- [ ] Type checking passing (0 errors)
- [ ] Linting passing (0 warnings)
- [ ] Coverage >85% achieved
- [ ] All critical paths tested

### Phase 3e: Documentation & Sign-Off
- [ ] Results documented
- [ ] Deployment checklist created
- [ ] Release notes written
- [ ] QA sign-off obtained
- [ ] Ready for deployment

---

## Troubleshooting

### Tests Failing
1. Check Phase 2 is complete
2. Verify database schema exists
3. Check for connection issues
4. Review error logs

### Performance Not Meeting Targets
1. Enable query logging (SQLALCHEMY_ECHO=true)
2. Check for N+1 queries
3. Verify cache is working
4. Profile with py-spy

### Coverage Below 85%
1. Identify untested modules
2. Add tests for critical paths
3. Use coverage report to find gaps
4. Review intentional gaps

---

## Success Criteria Summary

**All of the following must be true**:
1. ✅ All 2665+ tests passing
2. ✅ Code coverage >85%
3. ✅ Type checking 100% pass
4. ✅ Linting 100% pass
5. ✅ Performance improvements verified
6. ✅ 100% backward compatibility
7. ✅ Manual testing complete
8. ✅ Documentation complete
9. ✅ QA sign-off obtained

**Then**: Phase 3 Complete → Ready for v0.29.0 deployment

---

**Document Version**: 1.0
**Last Updated**: 2026-02-03
**Owner**: QA Team / Test Automation Engineer
