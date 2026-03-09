# Implementation Plan: Multi-Agent Coordination Enhancements

**Type:** Plan
**Status:** Ready
**Created:** 20251223-010000

---

## Overview

Enhance HtmlGraph's multi-agent coordination capabilities by implementing handoff context management, packaging deployment automation for general use, and adding agent routing intelligence. Leverages existing SessionManager, SDK, and deployment infrastructure with minimal new code.

---

## Plan Structure

```yaml
metadata:
  name: "Multi-Agent Coordination Enhancements"
  created: "20251223-010000"
  status: "ready"

# Research-grounded approach
overview: |
  Extend HtmlGraph's multi-agent coordination with three parallel improvements:
  1. Handoff Context & Notes - Enable seamless agent-to-agent task transitions
  2. Package Deployment Scripts - Make deployment automation reusable for all Python projects
  3. Agent Capabilities & Smart Routing - Intelligent task assignment based on agent skills
  
  All features leverage existing infrastructure (SessionManager, SDK, deploy-all.sh) with
  minimal new code (~20% new, 80% reuse/extend).

# Research synthesis
research:
  approach: "Hybrid artifact + structured handoff using HTML as the artifact store"
  libraries:
    - name: "invoke"
      version: ">=2.2"
      reason: "Optional Python-native alternative to shell scripts for CI/CD automation"
      optional: true
    - name: "tomllib"
      version: "Python 3.11+"
      reason: "Built-in TOML parser for deploy script metadata updates"
      optional: false
  
  patterns:
    - file: "src/python/htmlgraph/session_manager.py:1-100"
      description: "SessionManager lifecycle - reuse claim/release for handoff coordination"
    - file: "src/python/htmlgraph/agents.py:1-100"
      description: "AgentInterface pattern - extend for capability-based routing"
    - file: "src/python/htmlgraph/sdk.py:1-100"
      description: "Fluent builder pattern - add .complete_and_handoff() method"
    - file: "scripts/deploy-all.sh:1-50"
      description: "Deployment workflow - generalize for all Python projects"
  
  specifications:
    - requirement: "Never edit HTML directly - always use SDK/API/CLI"
      status: "must_follow"
    - requirement: "All data through Pydantic models (validation before HTML)"
      status: "must_follow"
    - requirement: "ISO 8601 timestamps, semantic HTML format"
      status: "must_follow"
    - requirement: "Agent attribution on all activities"
      status: "must_follow"
    - requirement: "Append-only event logging in JSONL"
      status: "must_follow"
  
  dependencies:
    existing:
      - "justhtml>=0.6.0"
      - "pydantic>=2.0.0"
      - "watchdog>=3.0.0"
      - "rich>=13.0.0"
    new:
      - "invoke>=2.2 (optional dev dependency)"

# Feature IDs for tracking
features:
  - "feature-20251221-211345"  # Handoff Context & Notes
  - "feat-130780b2"            # Package Deployment Script
  - "feature-20251221-211346"  # Agent Capabilities & Smart Routing

# Task index (TOC)
tasks:
  - id: "task-0"
    name: "Handoff Context System"
    file: "tasks/task-0.md"
    priority: "high"
    dependencies: []
    estimated_tokens: 3000

  - id: "task-1"
    name: "Deployment Script Generalization"
    file: "tasks/task-1.md"
    priority: "high"
    dependencies: []
    estimated_tokens: 2500

  - id: "task-2"
    name: "Agent Routing & Capabilities"
    file: "tasks/task-2.md"
    priority: "high"
    dependencies: []
    estimated_tokens: 3500

  - id: "task-3"
    name: "Integration Tests & Documentation"
    file: "tasks/task-3.md"
    priority: "medium"
    dependencies: ["task-0", "task-1", "task-2"]
    estimated_tokens: 2000

# Shared resources
shared_resources:
  files:
    - path: "src/python/htmlgraph/models.py"
      reason: "Task 0 adds handoff fields to Node model; Task 2 adds capability fields"
      mitigation: "Task 0 creates handoff fields first (lines 150-160), Task 2 adds capability fields after (lines 200-210)"
    
    - path: "src/python/htmlgraph/sdk.py"
      reason: "Task 0 adds .complete_and_handoff(); Task 2 adds .assign_by_capability()"
      mitigation: "Different method names, no conflicts. Both extend FeatureBuilder class."
    
    - path: "pyproject.toml"
      reason: "Task 1 adds deployment entry points; Task 2 might add optional deps"
      mitigation: "Task 1 adds [project.scripts], Task 2 adds [project.optional-dependencies]"

  databases:
    - name: ".htmlgraph/events/*.jsonl"
      concern: "Concurrent writes to event log during parallel development"
      mitigation: "Append-only file writes are atomic on POSIX systems; watchdog handles race conditions"

# Testing strategy
testing:
  unit:
    - "Each task writes tests in tests/python/test_{feature}.py"
    - "Must achieve >90% coverage for new code"
    - "All tests must pass before merge: uv run pytest"
  
  integration:
    - "Task 3 creates E2E tests for multi-agent handoff scenarios"
    - "Test cross-feature interactions (handoff + routing)"
    - "Verify deployment script works on clean install"
  
  isolation:
    - "Each worktree runs tests independently"
    - "No shared test state or fixtures"
    - "Mock file I/O where needed to prevent conflicts"

# Success criteria
success_criteria:
  - "All 4 tasks complete with tests passing"
  - "Zero merge conflicts (tasks are truly independent)"
  - "Documentation updated (AGENTS.md, README.md)"
  - "Examples added for each feature"
  - "Performance: handoff overhead <50ms, routing <100ms"
  - "Code review approved (PR created for each task)"

# Notes
notes: |
  **Parallel Execution Strategy:**
  
  Tasks 0, 1, 2 are fully independent:
  - Task 0 (Handoff) touches: models.py (lines 150-160), session_manager.py, sdk.py
  - Task 1 (Deployment) touches: scripts/, pyproject.toml ([project.scripts])
  - Task 2 (Routing) touches: models.py (lines 200-210), agents.py, sdk.py
  
  Conflict mitigation:
  - models.py: Task 0 uses lines 150-160, Task 2 uses lines 200-210 (non-overlapping)
  - sdk.py: Different method names (.complete_and_handoff() vs .assign_by_capability())
  - pyproject.toml: Different sections ([project.scripts] vs [project.optional-dependencies])
  
  Task 3 (Integration) runs sequentially after merge to main.
  
  **Execution Timeline:**
  - Parallel: Tasks 0, 1, 2 (3 agents, est. 60-90 min wall time)
  - Sequential: Task 3 after merge (1 agent, est. 30-45 min)
  - Total: ~2 hours wall time vs ~5 hours sequential (2.5x speedup)

# Changelog
changelog:
  - timestamp: "20251223-010000"
    event: "Plan created via /ctx:plan with parallel research (5 agents)"
  - timestamp: "20251223-010000"
    event: "Research synthesis completed - identified 80% code reuse opportunity"
```

---

## Task Details

### Task 0: Handoff Context System

---
id: task-0
priority: high
status: pending
dependencies: []
labels:
  - parallel-execution
  - auto-created
  - priority-high
  - feature-20251221-211345
---

# Handoff Context System

## 🎯 Objective

Enable seamless agent-to-agent task transitions by implementing structured handoff context that preserves decisions, blockers, and next steps. Agents can pass lightweight HTML hyperlink references to handoff artifacts instead of full context, preventing token bloat while maintaining audit trail.

## 🛠️ Implementation Approach

**Hybrid Artifact + Structured Schema Approach:**
- Extend `Node` model with handoff-specific fields (`previous_agent`, `handoff_reason`, `handoff_notes`)
- Add `create_handoff()` method to `SessionManager` (reuses existing claim/release logic)
- Implement `complete_and_handoff()` convenience method on `SDK.FeatureBuilder`
- Store handoff context as HTML `<section data-handoff>` with hyperlink references to previous sessions

**Libraries:**
- `pydantic>=2.0.0` - Schema validation for handoff context
- `justhtml>=0.6.0` - HTML serialization of handoff sections
- `rich>=13.0.0` - CLI output for handoff status

**Pattern to follow:**
- **File:** `src/python/htmlgraph/session_manager.py:1-100`
- **Description:** Reuse `claim_feature()` / `release_feature()` pattern. Add `create_handoff()` that:
  1. Validates current agent owns the feature
  2. Serializes handoff context to Pydantic model
  3. Updates Node with handoff metadata
  4. Releases feature for next agent to claim

## 📁 Files to Touch

**Modify:**
- `src/python/htmlgraph/models.py` (lines 150-160)
  - Add `handoff_required: bool = False`
  - Add `previous_agent: Optional[str] = None`
  - Add `handoff_reason: Optional[str] = None`
  - Add `handoff_notes: Optional[str] = None`
  - Add `handoff_timestamp: Optional[datetime] = None`

- `src/python/htmlgraph/session_manager.py`
  - Add `create_handoff(feature_id, reason, notes, next_agent=None)` method
  - Update `release_feature()` to check for pending handoffs

- `src/python/htmlgraph/sdk.py`
  - Add `complete_and_handoff(reason, notes, next_agent=None)` to `FeatureBuilder`
  - Update `to_context()` to include handoff metadata

**Create:**
- `tests/python/test_handoff.py` - Unit tests for handoff lifecycle
- `examples/handoff-demo.py` - Example multi-agent handoff workflow

## 🧪 Tests Required

**Unit:**
- [ ] Test handoff creation with valid agent ownership
- [ ] Test handoff rejection if agent doesn't own feature
- [ ] Test handoff context serialization to HTML
- [ ] Test handoff metadata in `to_context()` output
- [ ] Test edge case: handoff to self (should warn but allow)
- [ ] Test edge case: handoff with no notes (should require reason minimum)

**Integration:**
- [ ] Test full handoff workflow: Agent A claims → works → hands off → Agent B claims
- [ ] Test handoff visibility in dashboard HTML
- [ ] Test handoff context lightweight (<200 tokens for LLM)

## ✅ Acceptance Criteria

- [ ] All unit tests pass (`uv run pytest tests/python/test_handoff.py`)
- [ ] Handoff creates `<section data-handoff>` in feature HTML
- [ ] Handoff preserves hyperlink references to previous session
- [ ] `to_context()` includes handoff metadata (previous agent, reason, notes)
- [ ] Handoff overhead <50ms (benchmarked in tests)
- [ ] Code follows project conventions (Pydantic models, semantic HTML)
- [ ] Example added to `examples/handoff-demo.py`

## ⚠️ Potential Conflicts

**Files:**
- `src/python/htmlgraph/models.py` - Task 2 also modifies (adds capability fields)
  - **Mitigation:** Task 0 uses lines 150-160 (handoff fields), Task 2 uses lines 200-210 (capability fields)

## 📝 Notes

**Design Decision:** Use HTML hyperlinks (`<a href="session-xyz.html">`) instead of embedding full context. This:
- Prevents token bloat (reference = ~10 tokens vs full context = ~500 tokens)
- Maintains git-friendly audit trail
- Aligns with HtmlGraph philosophy (HTML hyperlinks ARE graph edges)

**Future Enhancement:** Add handoff priority queue (agents query for available handoffs first before new work).

---

**Worktree:** `worktrees/task-0-handoff`
**Branch:** `feature/handoff-context`

🤖 Auto-created via Contextune parallel execution

---

### Task 1: Deployment Script Generalization

---
id: task-1
priority: high
status: pending
dependencies: []
labels:
  - parallel-execution
  - auto-created
  - priority-high
  - feat-130780b2
---

# Deployment Script Generalization

## 🎯 Objective

Package HtmlGraph's deployment automation (`deploy-all.sh`) as a reusable pattern for all Python projects. Provide both shell-based interface (primary) and optional Python entry points via Invoke for CI/CD automation.

## 🛠️ Implementation Approach

**Dual Interface Strategy (Shell + Python):**
- Keep `deploy-all.sh` as primary interface (portable, familiar to users)
- Add **Invoke** tasks as optional Python-native alternative
- Package both via `pyproject.toml` entry points
- Create template for users to adapt to their projects

**Libraries:**
- `invoke>=2.2` - Task automation framework (optional dev dependency)
- `tomllib` (Python 3.11+) - Built-in TOML parser for metadata updates

**Pattern to follow:**
- **File:** `scripts/deploy-all.sh:1-50`
- **Description:** Current 7-step workflow (git push → build → publish → install → update plugins). Generalize by:
  1. Extracting project-specific vars to config section
  2. Making PyPI/plugin steps optional (flags: `--skip-pypi`, `--skip-plugins`)
  3. Adding template generation: `htmlgraph init-deploy` creates deploy script for user's project

## 📁 Files to Touch

**Modify:**
- `scripts/deploy-all.sh`
  - Add config section at top (project-specific variables)
  - Add usage documentation header
  - Improve error handling (fail fast on errors)

- `pyproject.toml`
  - Add `[project.scripts]` entry points:
    - `htmlgraph-deploy = "htmlgraph.scripts.deploy:main"`
  - Add `[project.optional-dependencies]` dev group:
    - `invoke>=2.2`

**Create:**
- `scripts/tasks.py` - Invoke task equivalents (deploy, build, publish)
- `src/python/htmlgraph/scripts/deploy.py` - Python entry point wrapping shell script
- `scripts/templates/deploy-template.sh` - User-customizable template
- `scripts/README.md` - Documentation for deployment automation
- `tests/python/test_deploy.py` - Unit tests for deploy script logic

## 🧪 Tests Required

**Unit:**
- [ ] Test version extraction from pyproject.toml
- [ ] Test dry-run mode (no actual publish)
- [ ] Test flag parsing (`--docs-only`, `--build-only`, `--skip-pypi`)
- [ ] Test error handling (invalid version, missing credentials)
- [ ] Test template generation (`htmlgraph init-deploy`)

**Integration:**
- [ ] Test full deploy workflow on clean virtualenv
- [ ] Test `invoke deploy --version=0.8.0` equivalence to shell script
- [ ] Verify packaged entry point works: `htmlgraph-deploy 0.8.0`

## ✅ Acceptance Criteria

- [ ] All tests pass (`uv run pytest tests/python/test_deploy.py`)
- [ ] Deploy script works on fresh clone (no hardcoded paths)
- [ ] Invoke tasks provide identical functionality to shell script
- [ ] Template generates customizable deploy script for other projects
- [ ] Documentation added to `scripts/README.md`
- [ ] Entry points registered in `pyproject.toml`
- [ ] No breaking changes to existing `./scripts/deploy-all.sh` usage

## ⚠️ Potential Conflicts

**Files:**
- `pyproject.toml` - Task 2 might add optional dependencies
  - **Mitigation:** Task 1 uses `[project.scripts]` section, Task 2 uses `[project.optional-dependencies]`. No overlap.

## 📝 Notes

**Design Decision:** Shell script remains primary interface (don't force users into Python workflows). Invoke tasks are opt-in enhancement for developers who prefer programmatic control.

**Distribution Strategy:**
```bash
# Default (shell script)
./scripts/deploy-all.sh 0.8.0

# Python package (after install)
htmlgraph-deploy 0.8.0
# or
invoke deploy --version=0.8.0
```

**Future Enhancement:** Add `htmlgraph init-deploy --template=pypi` to generate deployment scripts for different ecosystems (npm, cargo, etc.).

---

**Worktree:** `worktree/task-1-deploy`
**Branch:** `feature/package-deployment`

🤖 Auto-created via Contextune parallel execution

---

### Task 2: Agent Routing & Capabilities

---
id: task-2
priority: high
status: pending
dependencies: []
labels:
  - parallel-execution
  - auto-created
  - priority-high
  - feature-20251221-211346
---

# Agent Routing & Capabilities

## 🎯 Objective

Implement capability-based agent routing to intelligently assign tasks to agents based on their declared skills, availability, and workload. Enable multi-agent coordination where agents discover and claim tasks suited to their capabilities.

## 🛠️ Implementation Approach

**Capability Registry + Smart Routing:**
- Extend `Node` model with `required_capabilities: list[str]` field
- Add `AgentCapabilityRegistry` class to track agent skills
- Extend `AgentInterface` with `get_tasks_by_capability(agent_capabilities)`
- Add `assign_by_capability()` method to `SDK.FeatureBuilder`

**Libraries:**
- `pydantic>=2.0.0` - Schema validation for capability definitions
- `rich>=13.0.0` - CLI output for routing decisions

**Pattern to follow:**
- **File:** `src/python/htmlgraph/agents.py:1-100`
- **Description:** Extend `AgentInterface` pattern. Add capability matching that:
  1. Filters tasks by required capabilities
  2. Scores agent-task fit (exact match > partial match > no match)
  3. Considers agent workload (WIP limit from SessionManager)
  4. Returns prioritized task list

## 📁 Files to Touch

**Modify:**
- `src/python/htmlgraph/models.py` (lines 200-210)
  - Add `required_capabilities: list[str] = Field(default_factory=list)`
  - Add `capability_tags: list[str] = Field(default_factory=list)` (for flexible tagging)

- `src/python/htmlgraph/agents.py`
  - Add `AgentCapabilityRegistry` class
  - Extend `AgentInterface.get_available_tasks()` with capability filtering
  - Add `get_tasks_by_capability(agent_capabilities)` method

- `src/python/htmlgraph/sdk.py`
  - Add `set_required_capabilities(capabilities)` to `FeatureBuilder`
  - Add `assign_by_capability(agent_registry)` method

**Create:**
- `src/python/htmlgraph/routing.py` - Routing algorithms (capability matching, scoring)
- `tests/python/test_routing.py` - Unit tests for routing logic
- `examples/routing-demo.py` - Example multi-agent capability-based routing

## 🧪 Tests Required

**Unit:**
- [ ] Test capability matching (exact match, partial match, no match)
- [ ] Test agent-task fit scoring algorithm
- [ ] Test routing with workload balancing (prefer agents with lower WIP)
- [ ] Test routing with no capable agents (should return empty or log warning)
- [ ] Test edge case: task with no required capabilities (available to all agents)
- [ ] Test edge case: agent with no declared capabilities (gets untagged tasks only)

**Integration:**
- [ ] Test full routing workflow: Register agents → Create tasks → Route to best fit
- [ ] Test routing interacts correctly with SessionManager WIP limits
- [ ] Test routing + handoff: Agent A works → hands off → Agent B with right capabilities claims

## ✅ Acceptance Criteria

- [ ] All tests pass (`uv run pytest tests/python/test_routing.py`)
- [ ] Routing considers capabilities, workload, and availability
- [ ] Routing overhead <100ms for 100 tasks (benchmarked)
- [ ] Feature HTML includes `<ul data-required-capabilities>` section
- [ ] SDK fluent API: `sdk.features.create(...).set_required_capabilities(['python', 'testing']).save()`
- [ ] Code follows project conventions (Pydantic models, semantic HTML)
- [ ] Example added to `examples/routing-demo.py`

## ⚠️ Potential Conflicts

**Files:**
- `src/python/htmlgraph/models.py` - Task 0 also modifies (adds handoff fields)
  - **Mitigation:** Task 0 uses lines 150-160 (handoff fields), Task 2 uses lines 200-210 (capability fields). Non-overlapping.

- `src/python/htmlgraph/sdk.py` - Task 0 adds `.complete_and_handoff()`, Task 2 adds `.assign_by_capability()`
  - **Mitigation:** Different method names, both extend `FeatureBuilder`. No conflicts.

## 📝 Notes

**Design Decision:** Capability registry is in-memory (no persistence). Agents declare capabilities when initializing `AgentInterface`:
```python
agent = AgentInterface('.htmlgraph', agent_id='claude', capabilities=['python', 'documentation', 'testing'])
```

**Future Enhancement:** Persist agent capabilities to `.htmlgraph/agents/{agent-id}.html` for long-term tracking and analytics (which agents worked on what types of tasks).

**Capability Taxonomy:** Start with freeform strings. Later versions could add structured taxonomy (languages, frameworks, domains).

---

**Worktree:** `worktrees/task-2-routing`
**Branch:** `feature/agent-routing`

🤖 Auto-created via Contextune parallel execution

---

### Task 3: Integration Tests & Documentation

---
id: task-3
priority: medium
status: pending
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
**Branch:** `feature/integration-tests`

🤖 Auto-created via Contextune parallel execution

---

## References

- **Session Management**: [AGENTS.md](./AGENTS.md), [docs/guide/sessions.md](./docs/guide/sessions.md)
- **Existing Patterns**: `src/python/htmlgraph/session_manager.py`, `src/python/htmlgraph/agents.py`
- **Deployment Workflow**: `scripts/deploy-all.sh`, `scripts/README.md`
- **Research Sources**: See research agent outputs (Agents abd954c, a894127, a5c8e9a, afe0cd8, ad2c804)