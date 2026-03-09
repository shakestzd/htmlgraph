---
id: task-3
priority: medium
status: completed
dependencies: ["task-0", "task-1", "task-2"]
labels:
  - sequential-execution
  - auto-created
  - priority-medium
---

# Integration Tests & Documentation

## 🎯 Objective

Create end-to-end integration tests that verify all three features work together correctly. Update documentation (AGENTS.md, README.md) with examples and API references for handoff, deployment, and routing features.

## 🛠️ Implementation Approach

**E2E Test Scenarios:**
1. **Handoff + Routing**: Agent A (Python capabilities) starts task → hands off → Agent B (Testing capabilities) claims and completes
2. **Deployment**: Run full deploy workflow on clean virtualenv, verify all steps execute correctly
3. **Multi-agent coordination**: 3 agents work on independent tasks, verify no conflicts in event log

**Libraries:**
- `pytest>=7.0.0` - Test framework
- `pytest-playwright>=0.4.0` - E2E testing (if dashboard testing needed)

**Pattern to follow:**
- **File:** `tests/python/test_claiming.py`
- **Description:** Integration test pattern. Create realistic multi-agent scenarios with proper setup/teardown.

## 📁 Files to Touch

**Create:**
- `tests/integration/test_handoff_routing.py` - E2E handoff + routing tests
- `tests/integration/test_deployment_workflow.py` - Full deploy cycle tests
- `tests/integration/test_multi_agent_coordination.py` - 3+ agent scenarios

**Modify:**
- `AGENTS.md`
  - Add "Handoff Context" section with API examples
  - Add "Agent Routing" section with capability matching examples
  - Add "Deployment Automation" section

- `README.md`
  - Update "Features" list with handoff, routing
  - Add deployment automation quick start

- `docs/guide/sessions.md`
  - Add handoff lifecycle documentation
  - Add handoff context best practices

## 🧪 Tests Required

**Integration:**
- [ ] Test handoff + routing: Agent A → handoff → Agent B (capability match)
- [ ] Test deployment: `./scripts/deploy-all.sh --dry-run` completes successfully
- [ ] Test multi-agent: 3 agents work simultaneously, no event log conflicts
- [ ] Test dashboard: Handoff context visible in session HTML
- [ ] Test CLI: `htmlgraph status` shows handoff metadata

**Documentation:**
- [ ] Verify all code examples in docs are executable
- [ ] Verify all API references are accurate (function signatures, return types)
- [ ] Verify links between docs are valid

## ✅ Acceptance Criteria

- [ ] All integration tests pass (`uv run pytest tests/integration/`)
- [ ] Documentation updated with examples for all 3 features
- [ ] Code examples in docs are copy-pasteable and work
- [ ] Performance benchmarks documented (handoff <50ms, routing <100ms)
- [ ] Changelog updated with new features
- [ ] PR created with all changes for code review

## ⚠️ Potential Conflicts

**None** - This task runs sequentially after tasks 0, 1, 2 are merged to main.

## 📝 Notes

**Test Data Strategy:** Use fixtures to create realistic feature graphs (10+ features, 3+ agents, handoff chains). Ensure tests are reproducible (deterministic IDs, mocked timestamps).

**Documentation Structure:**
- AGENTS.md: API reference for developers
- README.md: Quick start for users
- docs/guide/sessions.md: Deep dive into handoff lifecycle

---

**Worktree:** `worktrees/task-3-integration`
**Branch:** `feature/task-3`

🤖 Auto-created via Contextune parallel execution
