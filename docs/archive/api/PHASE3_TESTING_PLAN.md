# Phase 3: Comprehensive Testing & Validation

## Overview

Phase 3 validates that the API refactor (Phase 2) meets all acceptance criteria:
- Repositories and services properly encapsulate data access
- FastSQLA is used consistently throughout
- fastapi-cache2 replaces QueryCache
- N+1 queries eliminated (bulk fetching)
- WebSocket performance improved (session pooling)
- All endpoints maintain backward compatibility

## Acceptance Criteria Checklist

### Criteria 1: Code Organization
- [ ] `src/python/wipnote/api/main.py` reduced to <500 lines (from 2761)
- [ ] All database queries moved to repositories
- [ ] All business logic moved to services
- [ ] All models defined in schemas module
- [ ] app.py, db.py, cache.py properly separated

### Criteria 2: API Compatibility
- [ ] All endpoints return identical response shapes
- [ ] Status codes unchanged
- [ ] Response times within baseline ±10%
- [ ] No breaking changes to REST API
- [ ] WebSocket messages unchanged

### Criteria 3: Database Architecture
- [ ] FastSQLA used for all DB sessions
- [ ] No raw aiosqlite connections outside of repositories
- [ ] Connection pooling working correctly
- [ ] Busy timeout properly configured (5000ms)
- [ ] Transactions properly scoped

### Criteria 4: Caching Strategy
- [ ] fastapi-cache2 integrated for HTTP caching
- [ ] Cache TTL configurable (currently 1.0s)
- [ ] Cache invalidation on data changes
- [ ] Cache hit rate tracked and logged
- [ ] QueryCache removed (or only used internally)

### Criteria 5: Query Performance
- [ ] Activity feed uses single bulk query (not N+1)
- [ ] Orchestration summaries batch-load dependencies
- [ ] Analytics aggregations bulk-query metrics
- [ ] WebSocket polling uses cached/pooled connections
- [ ] Pagination uses efficient LIMIT/OFFSET

### Criteria 6: WebSocket Performance
- [ ] Single DB connection per WebSocket lifecycle
- [ ] No new connections per cycle
- [ ] Connection pooling reduces latency
- [ ] Graceful reconnection on errors
- [ ] Memory usage stable over time

## Testing Strategy

### Unit Tests (Repository & Service Layer)

**Location**: `tests/unit/api/`

```
tests/unit/api/
├── test_repositories/
│   ├── test_events_repository.py
│   ├── test_features_repository.py
│   ├── test_sessions_repository.py
│   └── test_base_repository.py
├── test_services/
│   ├── test_activity_service.py
│   ├── test_orchestration_service.py
│   ├── test_analytics_service.py
│   └── test_base_service.py
└── test_models/
    ├── test_schemas.py
    └── test_pagination.py
```

**Test Coverage Targets**:
- Repositories: 85% (all query methods, error handling)
- Services: 90% (business logic, caching, aggregation)
- Models: 95% (validation, serialization)

**Key Test Areas**:

1. **Repository Tests**
   - `test_find_by_id` - Single record retrieval
   - `test_find_all` - Bulk retrieval
   - `test_find_with_filters` - Filtering accuracy
   - `test_pagination` - Offset/limit correctness
   - `test_bulk_operations` - Batch insert/update
   - `test_transaction_handling` - ACID properties
   - `test_error_handling` - Exception paths

2. **Service Tests**
   - `test_get_grouped_events` - Activity aggregation
   - `test_cache_hit_rate` - Cache effectiveness
   - `test_cache_invalidation` - Cache coherence
   - `test_bulk_fetching` - N+1 prevention
   - `test_circular_dependency_detection` - Orchestration logic
   - `test_cost_aggregation` - Analytics correctness
   - `test_dependency_injection` - Service initialization

3. **Schema Tests**
   - `test_event_model_validation` - Pydantic validation
   - `test_feature_model_serialization` - JSON serialization
   - `test_session_model_defaults` - Default values
   - `test_pagination_model` - Limit/offset validation

### Integration Tests (Endpoint Layer)

**Location**: `tests/integration/api/`

```
tests/integration/api/
├── test_endpoints/
│   ├── test_activity_endpoints.py
│   ├── test_orchestration_endpoints.py
│   ├── test_analytics_endpoints.py
│   ├── test_features_endpoints.py
│   └── test_sessions_endpoints.py
├── test_websockets/
│   ├── test_broadcast_websocket.py
│   └── test_cost_alerts_websocket.py
└── test_cache/
    ├── test_cache_invalidation.py
    └── test_cache_performance.py
```

**Test Coverage Targets**:
- Integration tests: 80% (happy paths + error cases)

**Key Test Areas**:

1. **Endpoint Tests**
   - Request/response shape validation
   - HTTP status codes
   - Error handling (404, 500, etc.)
   - Pagination parameters
   - Query parameters validation
   - Authentication/authorization (if applicable)

2. **WebSocket Tests**
   - Connection establishment
   - Message reception
   - Reconnection handling
   - Graceful closure
   - Error recovery

3. **Cache Tests**
   - Cache hit/miss rates
   - TTL expiration
   - Invalidation triggers
   - Performance under load

### End-to-End Tests (Full Stack)

**Location**: `tests/e2e/`

```
tests/e2e/
├── test_dashboard_flows.py
├── test_data_consistency.py
└── test_performance_baseline.py
```

**Key Test Scenarios**:

1. **Dashboard Loading**
   - Load main page
   - All tabs render correctly
   - Data displays accurately
   - No console errors

2. **Data Consistency**
   - Create feature, verify in API
   - Create event, verify in feed
   - Update session, verify in sessions list
   - Delete event, verify cache invalidation

3. **Performance Baseline**
   - Activity feed load time <500ms
   - Agent stats load time <300ms
   - WebSocket latency <100ms
   - Cache hit rate >80%

## Manual Testing Checklist

### UI Testing
- [ ] Dashboard loads without errors
- [ ] All tabs are clickable and responsive
- [ ] Activity feed displays events correctly
- [ ] Pagination works (next/prev buttons)
- [ ] Search/filter functionality works
- [ ] WebSocket updates appear in real-time

### Data Verification
- [ ] Events display with correct timestamps
- [ ] Agent names are correct
- [ ] Cost calculations are accurate
- [ ] Feature status transitions work
- [ ] Session duration calculations correct

### Performance Verification
- [ ] Initial page load <2s
- [ ] Activity feed renders <500ms
- [ ] WebSocket updates <200ms
- [ ] Dashboard stable with 1000+ events
- [ ] No memory leaks over 1 hour

### Error Handling
- [ ] Database connection errors handled gracefully
- [ ] Network timeouts don't crash dashboard
- [ ] Invalid query parameters return 400 errors
- [ ] Missing resources return 404 errors
- [ ] Server errors return 500 with details

## Performance Benchmarks

### Query Performance Targets

**Before Refactor** (Baseline):
- Activity feed (100 events): ~200ms
- Agent stats: ~150ms
- Orchestration chain: ~250ms
- WebSocket cycle: ~50ms

**After Refactor** (Target):
- Activity feed (100 events): <150ms (25% improvement)
- Agent stats: <120ms (20% improvement)
- Orchestration chain: <180ms (28% improvement)
- WebSocket cycle: <30ms (40% improvement)

### Cache Performance Targets

- Cache hit rate: >80% (measured over 1 hour)
- Cache miss recovery time: <50ms
- Cache invalidation latency: <10ms
- Memory overhead: <50MB (for entire cache)

### Database Connection Targets

- Connection pool utilization: 60-80%
- Connection wait time: <5ms (p99)
- Busy timeout triggers: <1% of queries
- Connection leak detection: 0 leaks over 24 hours

## Test Execution Plan

### Phase 3a: Unit Test Implementation
**Timeline**: 2-3 days
**Tasks**:
1. Create repository test fixtures
2. Write repository tests (EventsRepository, FeaturesRepository, SessionsRepository)
3. Write service tests (ActivityService, OrchestrationService, AnalyticsService)
4. Write schema/model tests
5. Achieve 85%+ coverage on repositories, 90%+ on services

**Success Criteria**:
- All unit tests passing
- Coverage reports generated
- No flaky tests

### Phase 3b: Integration Test Implementation
**Timeline**: 2-3 days
**Tasks**:
1. Create test client fixtures (FastAPI TestClient)
2. Write endpoint tests (activity, orchestration, analytics, features, sessions)
3. Write WebSocket tests
4. Write cache behavior tests
5. Achieve 80%+ coverage on integration layer

**Success Criteria**:
- All integration tests passing
- No endpoint behavior changes detected
- Cache hit rates measured and logged

### Phase 3c: Manual Testing & Performance Validation
**Timeline**: 1-2 days
**Tasks**:
1. Manual UI testing using test checklist
2. Performance profiling with load testing
3. Compare before/after query times
4. Verify cache hit rates in production
5. Document performance improvements

**Success Criteria**:
- All manual test checklist items pass
- Performance improvements documented
- No regressions detected

### Phase 3d: Code Quality Gates
**Timeline**: 1 day
**Tasks**:
1. Run full test suite: `uv run pytest`
2. Type checking: `uv run mypy src/`
3. Linting: `uv run ruff check --fix && uv run ruff format`
4. Coverage analysis: `uv run pytest --cov`
5. Document any remaining tech debt

**Success Criteria**:
- 100% pass rate on all tests
- 0 type errors
- 0 lint warnings
- Coverage >85% overall

## Test Data & Fixtures

### Fixture Strategy

Use factory pattern for test data:

```python
# tests/fixtures.py
@pytest.fixture
def sample_event():
    """Create a sample event for testing."""
    return EventFactory.create(
        agent_id="test-agent",
        event_type="UserQuery",
        status="completed"
    )

@pytest.fixture
async def db_with_events(db):
    """Create database with sample events."""
    events = [EventFactory.create() for _ in range(100)]
    await db.insert_many(events)
    return db
```

### Test Data Requirements

- Minimum 100 sample events for testing
- 10 sample features with various statuses
- 5 sample sessions with different durations
- Mock WebSocket clients for connection testing
- Synthetic metrics for analytics testing

## Documentation Requirements

### Test Documentation
- [ ] Test plan document (this file)
- [ ] Repository test guide
- [ ] Service test guide
- [ ] Integration test guide
- [ ] Performance benchmarking guide

### Code Documentation
- [ ] Docstrings updated in repositories
- [ ] Docstrings updated in services
- [ ] API endpoint documentation
- [ ] Cache behavior documentation
- [ ] WebSocket protocol documentation

## Deployment Readiness Checklist

Before deploying to PyPI:

- [ ] All unit tests passing (2665+ tests)
- [ ] All integration tests passing
- [ ] Code coverage >85%
- [ ] Type checking 100% pass
- [ ] Linting 100% pass
- [ ] Performance benchmarks met
- [ ] Manual testing complete
- [ ] Documentation updated
- [ ] Changelog entry added
- [ ] Version number bumped (0.29.0)
- [ ] GitHub release created

## Success Metrics

**Quantitative**:
- Test coverage: >85% overall
- Performance improvement: >25% on activity feed
- Cache hit rate: >80%
- All tests passing: 100%
- Type errors: 0
- Lint warnings: 0

**Qualitative**:
- Code is more maintainable
- API behavior unchanged
- Performance improved
- Architecture is cleaner
- Documentation is comprehensive

## Risk Mitigation

**Risk**: Tests reveal breaking changes
**Mitigation**: Catch early, revert and re-implement Phase 2

**Risk**: Performance regressions
**Mitigation**: Benchmark before/after, optimize queries

**Risk**: Cache invalidation bugs
**Mitigation**: Comprehensive cache testing, manual verification

**Risk**: WebSocket connection leaks
**Mitigation**: Connection pool monitoring, graceful error handling

## References

- Repository pattern: `src/python/wipnote/api/repositories/`
- Service layer: `src/python/wipnote/api/services/`
- Cache implementation: `src/python/wipnote/api/cache.py`
- Database layer: `src/python/wipnote/api/db.py`
- WebSocket handlers: `src/python/wipnote/api/broadcast_websocket.py`

---

**Last Updated**: 2026-02-03
**Status**: In Progress (Phase 2 → Phase 3)
**Owner**: Claude (Refactoring Team)
