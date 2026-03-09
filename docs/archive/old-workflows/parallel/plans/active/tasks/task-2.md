---
id: task-2
priority: high
status: completed
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
**Branch:** `feature/task-2`

🤖 Auto-created via Contextune parallel execution
