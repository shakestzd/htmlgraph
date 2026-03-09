# Implementation Plan: Pre-Work Validation Hook with Auto-Spike Integration

**Type:** Plan
**Status:** Ready
**Created:** 20251225-050700

---

## Overview

Implement a PreToolUse hook that enforces the HtmlGraph workflow by validating that code-modifying tools (Write, Edit, Delete) are only used when an active work item exists. The system integrates seamlessly with auto-spikes (session-init, transition) and respects the decision framework for when work items are mandatory vs optional.

---

## Research Synthesis

### Best Approach
PreToolUse hook with JSON-based permission decisions (deny/allow/ask). Uses structured validation instead of stderr blocking for better Claude integration. Hook validates before tool execution, checking for active work items (including auto-spikes) and applying the decision framework to determine if work item creation is mandatory.

### Libraries/Tools
- **Python stdlib only**: json, sys, os, pathlib, subprocess
- **Existing HtmlGraph SDK**: SessionManager, Node models
- **Reuse patterns from**: track-event.py (hook structure, bootstrapping)

### Existing Code to Reuse
- **File:** `packages/claude-plugin/hooks/scripts/track-event.py:23-71`
  - **Pattern:** `_resolve_project_dir()` and `_bootstrap_pythonpath()` functions
- **File:** `packages/claude-plugin/hooks/scripts/track-event.py:88-122`
  - **Pattern:** `load_drift_config()` configuration loading pattern
- **File:** `src/python/htmlgraph/session_manager.py:1176-1190`
  - **Pattern:** `get_active_features()` method for status-based filtering

### Specification Compliance
- **Requirement:** Must NOT block auto-spike creation (session-init, transition)
- **Status:** must_follow
- **Requirement:** Apply decision framework (3+ files = mandatory work item)
- **Status:** must_follow
- **Requirement:** Use JSON permission decisions, not stderr blocking
- **Status:** must_follow
- **Requirement:** Gracefully degrade if dependencies missing
- **Status:** should_follow

### Dependencies

**Existing:**
- htmlgraph package (SessionManager, Node, SDK)
- Python 3.10+ stdlib (json, sys, pathlib, subprocess)

**New:**
- None - all infrastructure exists

---

## Plan Structure

```yaml
metadata:
  name: "Pre-Work Validation Hook with Auto-Spike Integration"
  created: "20251225-050700"
  status: "ready"

overview: |
  Implement PreToolUse hook that enforces HtmlGraph workflow by requiring
  active work items for code changes. Integrates with auto-spike system
  and applies decision framework to determine when work items are mandatory.

research:
  approach: "PreToolUse hook with JSON permission decisions"
  libraries:
    - name: "Python stdlib"
      reason: "No external dependencies needed"
  patterns:
    - file: "track-event.py:23-71"
      description: "Project dir resolution and pythonpath bootstrapping"
    - file: "track-event.py:88-122"
      description: "Configuration loading pattern"
    - file: "session_manager.py:1176-1190"
      description: "Active feature detection via status filtering"
  specifications:
    - requirement: "Must not block auto-spike creation"
      status: "must_follow"
    - requirement: "Apply decision framework for mandatory work items"
      status: "must_follow"
    - requirement: "Use JSON permission decisions"
      status: "must_follow"
  dependencies:
    existing:
      - "htmlgraph (SessionManager, SDK)"
      - "Python 3.10+ stdlib"
    new: []

features:
  - "bug-60baa800"

tasks:
  - id: "task-0"
    name: "Create validation configuration schema"
    file: "tasks/task-0.md"
    priority: "blocker"
    dependencies: []

  - id: "task-1"
    name: "Implement get_active_work_item() method"
    file: "tasks/task-1.md"
    priority: "blocker"
    dependencies: []

  - id: "task-2"
    name: "Create PreToolUse validation hook script"
    file: "tasks/task-2.md"
    priority: "high"
    dependencies: ["task-0", "task-1"]

  - id: "task-3"
    name: "Integrate with auto-spike detection"
    file: "tasks/task-3.md"
    priority: "high"
    dependencies: ["task-2"]

  - id: "task-4"
    name: "Update htmlgraph-tracker skill documentation"
    file: "tasks/task-4.md"
    priority: "medium"
    dependencies: ["task-2", "task-3"]

  - id: "task-5"
    name: "Add validation to session-start-info"
    file: "tasks/task-5.md"
    priority: "medium"
    dependencies: ["task-1"]

  - id: "task-6"
    name: "Create integration tests"
    file: "tasks/task-6.md"
    priority: "high"
    dependencies: ["task-2", "task-3"]

shared_resources:
  files:
    - path: "src/python/htmlgraph/session_manager.py"
      reason: "Task 1 adds method, other tasks call it"
      mitigation: "Task 1 creates method first, others use it"
    - path: "packages/claude-plugin/skills/htmlgraph-tracker/SKILL.md"
      reason: "Task 4 updates documentation"
      mitigation: "Single task owns this file"

testing:
  unit:
    - "Each task writes unit tests for new methods"
    - "Test auto-spike detection logic"
    - "Test decision framework thresholds"
  integration:
    - "Test full validation flow with real sessions"
    - "Test auto-spike creation bypass"
    - "Test multi-file change blocking"
  isolation:
    - "Mock SessionManager for hook tests"
    - "Use test fixtures for config files"

success_criteria:
  - "Hook blocks Write/Edit/Delete when no active work item and 3+ files"
  - "Hook allows auto-spike creation (session-init, transition)"
  - "Hook warns but allows single-file changes without work item"
  - "get_active_work_item() returns correct Node or None"
  - "All tests passing"
  - "Documentation updated with workflow guidance"

notes: |
  The key challenge is detecting auto-spike file writes to allow them
  while blocking other code changes. We detect via data-auto-generated
  attribute in file content and spike-subtype matching.
  
  Decision framework thresholds (3+ files, >30min) are conservative
  to prioritize attribution over efficiency.

changelog:
  - timestamp: "20251225-050700"
    event: "Plan created with parallel research synthesis"
```

---

## Task 0: Create validation configuration schema

---
id: task-0
priority: blocker
status: pending
dependencies: []
labels:
  - parallel-execution
  - auto-created
  - priority-blocker
---

# Create Validation Configuration Schema

## 🎯 Objective

Create the validation configuration file that defines rules for when work items are required, blocking vs warning behavior, and auto-spike detection settings.

## 🛠️ Implementation Approach

Follow the drift-config.json pattern from track-event.py. Create a validation-config.json with decision framework thresholds, auto-spike detection rules, and hook behavior settings.

**Libraries:**
- Python stdlib (json)

**Pattern to follow:**
- **File:** `packages/claude-plugin/config/drift-config.json:1-20`
- **Description:** Nested configuration structure with defaults

## 📁 Files to Touch

**Create:**
- `packages/claude-plugin/config/validation-config.json`

## 🧪 Tests Required

**Unit:**
- [ ] Test config loading with defaults
- [ ] Test config validation (invalid values rejected)
- [ ] Test config override via environment variable

## ✅ Acceptance Criteria

- [ ] Configuration file exists with all required fields
- [ ] Decision framework thresholds configurable (file_count, duration)
- [ ] Auto-spike detection rules defined
- [ ] Block vs warn behavior configurable
- [ ] Defaults match decision framework (3 files, 30 min)
- [ ] JSON validates (no syntax errors)

## ⚠️ Potential Conflicts

None - new file creation only

## 📝 Notes

Default configuration should match current decision framework:
- file_threshold: 3
- duration_threshold_minutes: 30
- auto_spike_detection: true
- block_on_violation: true (can be set to false for warn-only mode)
- allow_direct_implementation: true (for single-file changes)

---

**Worktree:** `worktrees/task-0`
**Branch:** `feature/task-0`

🤖 Auto-created via Contextune parallel execution

---

## Task 1: Implement get_active_work_item() method

---
id: task-1
priority: blocker
status: pending
dependencies: []
labels:
  - parallel-execution
  - auto-created
  - priority-blocker
---

# Implement get_active_work_item() Method

## 🎯 Objective

Add `get_active_work_item(session_id)` method to SessionManager that returns the currently active work item (feature, bug, spike, chore) for a given session, including auto-spikes.

## 🛠️ Implementation Approach

Use session-aware detection with priority fallback chain: check primary feature → session's in-progress items → global in-progress. Return Node or None.

**Libraries:**
- htmlgraph SDK (Node, Session models)

**Pattern to follow:**
- **File:** `src/python/htmlgraph/session_manager.py:1263-1276` 
- **Description:** `get_primary_feature()` method shows the pattern for priority-aware feature detection

## 📁 Files to Touch

**Modify:**
- `src/python/htmlgraph/session_manager.py` (add method after get_primary_feature)

## 🧪 Tests Required

**Unit:**
- [ ] Test returns None when no session
- [ ] Test returns None when no active features
- [ ] Test returns primary feature when set
- [ ] Test returns first in-progress from session's worked_on
- [ ] Test includes auto-spikes (status="in-progress" + auto_generated=True)
- [ ] Test handles multiple in-progress features (returns primary or first)

## ✅ Acceptance Criteria

- [ ] Method signature: `get_active_work_item(self, session_id: str) -> Node | None`
- [ ] Returns Node with status="in-progress" or None
- [ ] Respects primary feature flag (is_primary property)
- [ ] Includes auto-spikes in detection
- [ ] All unit tests pass
- [ ] Docstring documents behavior and return values

## ⚠️ Potential Conflicts

**Files:**
- `session_manager.py` - Task 2 imports this method

## 📝 Notes

Method should check collections in order: features, bugs, spikes, chores. Auto-spikes are just spikes with auto_generated=True, so no special handling needed - they're detected automatically via status check.

Consider adding optional `include_auto_spikes: bool = True` parameter for flexibility, but default should include them.

---

**Worktree:** `worktrees/task-1`
**Branch:** `feature/task-1`

🤖 Auto-created via Contextune parallel execution

---

## Task 2: Create PreToolUse validation hook script

---
id: task-2
priority: high
status: pending
dependencies: ["task-0", "task-1"]
labels:
  - parallel-execution
  - auto-created
  - priority-high
---

# Create PreToolUse Validation Hook Script

## 🎯 Objective

Create the core validation hook script that intercepts Write/Edit/Delete tool calls and validates whether an active work item exists, blocking or warning based on decision framework rules.

## 🛠️ Implementation Approach

Create `validate-work.py` following track-event.py structure. Use JSON permission decisions ("deny"/"allow"/"ask") instead of stderr blocking. Load validation config, check active work item via get_active_work_item(), apply decision framework thresholds.

**Libraries:**
- Python stdlib (json, sys, pathlib)
- htmlgraph SDK (SessionManager)

**Pattern to follow:**
- **File:** `packages/claude-plugin/hooks/scripts/track-event.py:23-71`
- **Description:** Project bootstrapping and pythonpath setup
- **File:** `packages/claude-plugin/hooks/scripts/track-event.py:357-367`
- **Description:** JSON output format for hook responses

## 📁 Files to Touch

**Create:**
- `packages/claude-plugin/hooks/scripts/validate-work.py`
- `packages/claude-plugin/hooks/hook-config.json` (add PreToolUse entry)

## 🧪 Tests Required

**Unit:**
- [ ] Test allows tool when active work item exists
- [ ] Test blocks Write when no work item and 3+ files
- [ ] Test warns but allows single-file Write without work item
- [ ] Test gracefully degrades when SessionManager unavailable
- [ ] Test JSON output format (permissionDecision, permissionDecisionReason)

**Integration:**
- [ ] Test with real SessionManager and .htmlgraph directory
- [ ] Test with multiple file changes (should block)
- [ ] Test with single file change (should warn but allow)

## ✅ Acceptance Criteria

- [ ] Hook script executes without errors
- [ ] Returns valid JSON with exit code 0
- [ ] Blocks Write/Edit/Delete when no active work item and 3+ files
- [ ] Provides clear error messages with work item creation instructions
- [ ] Warns but allows single-file changes
- [ ] Loads validation-config.json with fallback defaults
- [ ] All tests passing
- [ ] Hook registered in hook-config.json

## ⚠️ Potential Conflicts

**Files:**
- `hook-config.json` - Multiple tasks may register hooks

## 📝 Notes

File count estimation logic:
- Check tool_input for multiple file_path entries
- For Edit: single file always (old_string/new_string)
- For Write: single file (file_path parameter)
- For batch operations: count distinct paths in content

Permission decision format:
```json
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "deny",
    "permissionDecisionReason": "Multi-file change requires work item..."
  }
}
```

---

**Worktree:** `worktrees/task-2`
**Branch:** `feature/task-2`

🤖 Auto-created via Contextune parallel execution

---

## Task 3: Integrate with auto-spike detection

---
id: task-3
priority: high
status: pending
dependencies: ["task-2"]
labels:
  - parallel-execution
  - auto-created
  - priority-high
---

# Integrate with Auto-Spike Detection

## 🎯 Objective

Add logic to validate-work.py to detect and allow auto-spike file creation (session-init, transition spikes) while still blocking other code changes when no work item exists.

## 🛠️ Implementation Approach

Check file content for auto-spike markers before validation. If file_path contains ".htmlgraph/spikes/" AND content contains 'data-auto-generated="true"' AND 'data-spike-subtype' in ("session-init", "transition"), allow the write without validation.

**Libraries:**
- Python stdlib (re for content parsing)

**Pattern to follow:**
- **File:** `src/python/htmlgraph/session_manager.py:837-855`
- **Description:** `_get_active_auto_spike()` shows auto-spike detection via attributes

## 📁 Files to Touch

**Modify:**
- `packages/claude-plugin/hooks/scripts/validate-work.py` (add auto-spike detection before validation)

## 🧪 Tests Required

**Unit:**
- [ ] Test allows session-init spike creation (no active work item needed)
- [ ] Test allows transition spike creation
- [ ] Test blocks regular spike creation without work item
- [ ] Test auto-spike detection via file path and content attributes
- [ ] Test edge case: manual spike with same path (blocked if no work item)

**Integration:**
- [ ] Test auto-spike creation in real session flow
- [ ] Test that auto-spikes count as active work items for subsequent writes

## ✅ Acceptance Criteria

- [ ] Auto-spike creation bypasses validation
- [ ] Detection checks both file path AND content attributes
- [ ] Session-init and transition spikes both allowed
- [ ] Manual spikes still require work item validation
- [ ] All tests passing
- [ ] No false positives (regular files not detected as auto-spikes)

## ⚠️ Potential Conflicts

**Files:**
- `validate-work.py` - Task 2 creates base, this task extends

## 📝 Notes

Auto-spike detection order:
1. Check tool_name == "Write" (auto-spikes only created via Write)
2. Check file_path contains ".htmlgraph/spikes/"
3. Parse content for 'data-auto-generated="true"'
4. Parse content for 'data-spike-subtype="session-init"' OR "transition"
5. If all match: return {"continue": True} immediately (skip validation)

Content parsing can use simple string search - no need for full HTML parsing:
```python
content = tool_input.get("content", "")
is_auto_spike = (
    ".htmlgraph/spikes/" in file_path
    and 'data-auto-generated="true"' in content
    and ('data-spike-subtype="session-init"' in content 
         or 'data-spike-subtype="transition"' in content)
)
```

---

**Worktree:** `worktrees/task-3`
**Branch:** `feature/task-3`

🤖 Auto-created via Contextune parallel execution

---

## Task 4: Update htmlgraph-tracker skill documentation

---
id: task-4
priority: medium
status: pending
dependencies: ["task-2", "task-3"]
labels:
  - parallel-execution
  - auto-created
  - priority-medium
---

# Update htmlgraph-tracker Skill Documentation

## 🎯 Objective

Update the htmlgraph-tracker skill documentation to reflect the new PreToolUse validation hook, explain when work items are enforced vs optional, and document auto-spike integration.

## 🛠️ Implementation Approach

Add new section "Pre-Work Validation" to SKILL.md explaining hook behavior, decision framework enforcement, and auto-spike integration. Update workflow checklist to reference validation.

**Pattern to follow:**
- **File:** `packages/claude-plugin/skills/htmlgraph-tracker/SKILL.md:1-100`
- **Description:** Existing skill structure with sections and examples

## 📁 Files to Touch

**Modify:**
- `packages/claude-plugin/skills/htmlgraph-tracker/SKILL.md`

## 🧪 Tests Required

**Manual:**
- [ ] Review updated skill for clarity
- [ ] Verify examples are accurate
- [ ] Check markdown formatting renders correctly

## ✅ Acceptance Criteria

- [ ] New "Pre-Work Validation" section added
- [ ] Explains when validation blocks vs warns
- [ ] Documents auto-spike integration
- [ ] Includes examples of validation scenarios
- [ ] Updates Session Workflow Checklist with validation references
- [ ] Markdown renders correctly
- [ ] No broken links or formatting issues

## ⚠️ Potential Conflicts

None - single task owns this documentation

## 📝 Notes

Add these scenarios to documentation:

**Scenario 1: Auto-Spike Active**
```
User: "investigate drift bug"
→ session-init auto-spike active
→ Write allowed (auto-spike counts as active work)
→ When ready to fix: complete spike, create bug
```

**Scenario 2: No Active Work + Multi-File Change**
```
User: "fix drift cleanup in 3 files"
→ No active work item
→ Validation: 3+ files = BLOCKED
→ Must create bug first
```

**Scenario 3: Single-File Quick Fix**
```
User: "fix typo in README"
→ No active work item
→ Validation: 1 file = WARNED but ALLOWED
→ Attribution: session-level admin work
```

---

**Worktree:** `worktrees/task-4`
**Branch:** `feature/task-4`

🤖 Auto-created via Contextune parallel execution

---

## Task 5: Add validation to session-start-info

---
id: task-5
priority: medium
status: pending
dependencies: ["task-1"]
labels:
  - parallel-execution
  - auto-created
  - priority-medium
---

# Add Validation to Session-Start-Info

## 🎯 Objective

Enhance the session-start-info output to show active work item status and alert if no work item is active, helping users understand validation state at session start.

## 🛠️ Implementation Approach

Call get_active_work_item() in session start-info generation and display active work item info or warning if none exists.

**Libraries:**
- htmlgraph SDK (SessionManager, SDK)

**Pattern to follow:**
- **File:** `src/python/htmlgraph/cli.py:400-450` (if session-start-info is in CLI)
- **Description:** Existing session-start-info formatting

## 📁 Files to Touch

**Modify:**
- Location of session-start-info implementation (CLI or hook)

## 🧪 Tests Required

**Integration:**
- [ ] Test output when active work item exists
- [ ] Test output when no work item (warning shown)
- [ ] Test output when auto-spike active

## ✅ Acceptance Criteria

- [ ] Session-start-info shows active work item (if exists)
- [ ] Shows warning if no active work item
- [ ] Indicates if auto-spike is active
- [ ] Output is clear and actionable
- [ ] All tests passing

## ⚠️ Potential Conflicts

None - extends existing functionality

## 📝 Notes

Output format:
```
## Active Work Item

✅ bug-60baa800 (in-progress): Drift queue cleanup
   Files: 3 modified
   Steps: 5/7 complete

OR

⚠️  No active work item
   Multi-file changes will be blocked.
   
   Start existing: uv run htmlgraph feature start <id>
   Create new: uv run htmlgraph bug create "Title"

OR

🔍 Auto-spike active: session-init
   Exploratory work allowed
   Create work item when ready to implement
```

---

**Worktree:** `worktrees/task-5`
**Branch:** `feature/task-5`

🤖 Auto-created via Contextune parallel execution

---

## Task 6: Create integration tests

---
id: task-6
priority: high
status: pending
dependencies: ["task-2", "task-3"]
labels:
  - parallel-execution
  - auto-created
  - priority-high
---

# Create Integration Tests

## 🎯 Objective

Create comprehensive integration tests that validate the full pre-work validation flow including hook execution, auto-spike detection, and decision framework application.

## 🛠️ Implementation Approach

Create test suite using pytest that simulates hook execution with various scenarios: auto-spike creation, multi-file changes, single-file changes, active work items, etc.

**Libraries:**
- pytest
- unittest.mock (for mocking SessionManager)

**Pattern to follow:**
- **File:** `tests/python/test_session_manager.py:1-50`
- **Description:** Existing test patterns for SessionManager

## 📁 Files to Touch

**Create:**
- `tests/python/test_pre_work_validation.py`

## 🧪 Tests Required

**Integration:**
- [ ] Test full validation flow with active work item (allows Write)
- [ ] Test validation blocks multi-file Write without work item
- [ ] Test validation warns but allows single-file Write
- [ ] Test auto-spike creation bypasses validation
- [ ] Test session-init spike counts as active work
- [ ] Test transition spike counts as active work
- [ ] Test validation with missing SessionManager (graceful degradation)
- [ ] Test configuration loading and defaults

## ✅ Acceptance Criteria

- [ ] All integration tests pass
- [ ] Tests cover all validation scenarios
- [ ] Tests verify auto-spike integration
- [ ] Tests check decision framework thresholds
- [ ] Test fixtures include sample sessions and work items
- [ ] Tests run in CI/CD pipeline
- [ ] Code coverage >80% for validation logic

## ⚠️ Potential Conflicts

None - new test file only

## 📝 Notes

Test scenarios to cover:

1. **Auto-Spike Scenarios:**
   - Session-init spike creation (bypass validation)
   - Transition spike creation (bypass validation)
   - Work with active session-init spike (allowed)

2. **Validation Scenarios:**
   - Write 1 file, no work item (warn, allow)
   - Write 3+ files, no work item (block)
   - Write with active feature (allow)
   - Edit with active bug (allow)

3. **Edge Cases:**
   - Missing .htmlgraph directory
   - Corrupted session data
   - Multiple in-progress features (uses primary)
   - Auto-spike + manual spike active (both valid)

Use test fixtures for consistent test data:
```python
@pytest.fixture
def sample_session():
    return Session(
        id="test-session",
        agent="claude-code",
        status="active",
        worked_on=["feature-001"]
    )
```

---

**Worktree:** `worktrees/task-6`
**Branch:** `feature/task-6`

🤖 Auto-created via Contextune parallel execution

---

## References

- Session Manager API: `src/python/htmlgraph/session_manager.py`
- Hook Patterns: `packages/claude-plugin/hooks/scripts/track-event.py`
- Decision Framework: `packages/claude-plugin/skills/htmlgraph-tracker/SKILL.md`
- Auto-Spike Implementation: `src/python/htmlgraph/session_manager.py:273-397`

---

📋 Plan created in extraction-optimized format!

**Plan Summary:**
- 7 total tasks
- 4 can run in parallel (tasks 0, 1, 4, 5)
- 3 have dependencies (tasks 2, 3, 6)
- Conflict risk: Low

**Tasks by Priority:**
- Blocker: task-0, task-1
- High: task-2, task-3, task-6
- Medium: task-4, task-5

**What Happens Next:**

The plan above will be automatically extracted to modular files when you:
1. Run `/ctx:execute` - Extracts and executes immediately
2. End this session - SessionEnd hook extracts automatically

**Extraction Output:**
```
.parallel/plans/
├── plan.yaml           (main plan with metadata)
├── tasks/
│   ├── task-0.md      (GitHub-ready task files)
│   ├── task-1.md
│   ├── task-2.md
│   ├── task-3.md
│   ├── task-4.md
│   ├── task-5.md
│   └── task-6.md
├── templates/
│   └── task-template.md
└── scripts/
    ├── add_task.sh
    └── generate_full.sh
```

**Key Benefits:**
✅ **Full visibility**: You see complete plan in conversation
✅ **Easy iteration**: Ask for changes before extraction
✅ **Zero manual work**: Extraction happens automatically
✅ **Modular files**: Edit individual tasks after extraction
✅ **Perfect DRY**: Plan exists once (conversation), extracted once (files)

**Next Steps:**
1. Review the plan above (scroll up if needed)
2. Request changes: "Change task 2 to use different approach"
3. When satisfied, run: `/ctx:execute`

Ready to execute? Run `/ctx:execute` to extract and start parallel development.