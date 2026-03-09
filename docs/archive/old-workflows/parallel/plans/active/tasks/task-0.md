---
id: task-0
priority: high
status: completed
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
**Branch:** `feature/task-0`

🤖 Auto-created via Contextune parallel execution
