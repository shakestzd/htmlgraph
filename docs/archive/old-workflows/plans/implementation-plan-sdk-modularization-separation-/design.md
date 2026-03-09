Excellent! Research complete. Now I'll create the comprehensive parallel development plan based on these findings.

# Implementation Plan: SDK Modularization & Separation of Concerns

**Type:** Plan
**Status:** Ready
**Created:** 20251223-100000

---

## Overview

Complete the SDK modularization by extracting Collection classes to `collections/` module, updating SDK to use new modular structure, ensuring zero breaking changes to public API, and comprehensive testing with all 371+ tests passing.

---

## Plan Structure

```yaml
metadata:
  name: "SDK Modularization & Separation of Concerns"
  created: "20251223-100000"
  status: "ready"

overview: |
  Refactor sdk.py (1000+ lines) into modular structure with builders/ and collections/
  modules. Achieve ~60% code reduction through BaseBuilder/BaseCollection reuse while
  maintaining 100% backward compatibility with public API.

research:
  approach: "Layered + Fluent Hybrid (Azure SDK model)"
  libraries:
    - name: "No new dependencies needed"
      reason: "Using stdlib typing.Generic for type-safe collections"
  patterns:
    - file: "builders/base.py:22"
      description: "Generic BaseBuilder[BuilderT] with TypeVar for fluent chaining"
    - file: "sdk.py:232"
      description: "Lazy-load graph pattern in Collection._ensure_graph()"
    - file: "src/python/htmlgraph/__init__.py:35"
      description: "Public API exports via __all__ list"
  specifications:
    - requirement: "All __all__ exports must remain importable from root"
      status: "must_follow"
    - requirement: "SDK class signature unchanged"
      status: "must_follow"
    - requirement: "Fluent builder API preserved"
      status: "must_follow"
  dependencies:
    existing:
      - "typing.Generic, TypeVar (stdlib)"
      - "pydantic >= 2.0.0"
      - "justhtml >= 0.6.0"
    new: []

features:
  - "BaseCollection with Generic[T] support"
  - "FeatureCollection with create() method"
  - "SpikeCollection with create() method"
  - "Updated SDK using modular imports"
  - "Backward compatibility layer"

tasks:
  - id: "task-0"
    name: "Extract Collection to collections/base.py"
    file: "tasks/task-0.md"
    priority: "high"
    dependencies: []

  - id: "task-1"
    name: "Create FeatureCollection and SpikeCollection"
    file: "tasks/task-1.md"
    priority: "high"
    dependencies: []

  - id: "task-2"
    name: "Update SDK to use modular imports"
    file: "tasks/task-2.md"
    priority: "high"
    dependencies: ["task-0", "task-1"]

  - id: "task-3"
    name: "Update __init__.py exports and run tests"
    file: "tasks/task-3.md"
    priority: "blocker"
    dependencies: ["task-0", "task-1", "task-2"]

shared_resources:
  files:
    - path: "src/python/htmlgraph/sdk.py"
      reason: "Task 2 refactors this after tasks 0-1 create modules"
      mitigation: "Tasks 0-1 run in parallel, task 2 runs after both complete"
    - path: "src/python/htmlgraph/__init__.py"
      reason: "Task 3 updates exports"
      mitigation: "Task 3 runs last sequentially"

testing:
  unit:
    - "Each task validates its own changes"
    - "Task 3 runs full test suite (371+ tests)"
  integration:
    - "Verify SDK fluent API still works"
    - "Verify Collection.create() methods"
    - "Verify backward compatibility"
  isolation:
    - "Tasks 0-1 are independent (different files)"
    - "Task 2 depends on 0-1 completion"
    - "Task 3 validates everything"

success_criteria:
  - "All 371+ tests passing"
  - "Zero breaking changes to public API"
  - "BaseCollection eliminates ~60% duplication"
  - "SpikeCollection.create() method works"
  - "SDK imports from new modules"
  - "Documentation updated"

notes: |
  Research findings show this aligns with Azure SDK architecture (proven at scale).
  No circular import risks detected. Using stdlib typing.Generic for type safety.
  Backward compatibility critical - all __all__ exports must work.

changelog:
  - timestamp: "20251223-100000"
    event: "Plan created with parallel research"
```

---

## Task Details

### Task 0: Extract Collection to collections/base.py

```yaml
---
id: task-0
priority: high
status: pending
dependencies: []
labels:
  - parallel-execution
  - auto-created
  - priority-high
---
```

# Extract Collection to collections/base.py

## 🎯 Objective

Extract the generic `Collection` class from sdk.py (lines 212-462) to `collections/base.py`, rename to `BaseCollection`, and add `Generic[T]` support for type-safe collection methods.

## 🛠️ Implementation Approach

**Pattern from research:**
- Azure SDK layered architecture - resource classes as collections
- Use `typing.Generic[T]` with TypeVar for type safety (python-telegram-bot pattern)
- Lazy-load graph pattern (preserve existing sdk.py:232 approach)

**Libraries:**
- `typing.Generic, TypeVar` (stdlib) - Type-safe collection interface

**Pattern to follow:**
- **File:** `builders/base.py:22` - Generic BaseBuilder[BuilderT] pattern
- **Description:** TypeVar with Generic for fluent chaining, preserve in subclasses

## 📁 Files to Touch

**Create:**
- `src/python/htmlgraph/collections/__init__.py` - Exports BaseCollection
- `src/python/htmlgraph/collections/base.py` - Renamed Collection → BaseCollection with Generic[T]

**Read (for reference):**
- `src/python/htmlgraph/sdk.py` (lines 212-462) - Source code to extract

**No modifications yet** - This task only creates new modules

## 🧪 Tests Required

**Unit:**
- [ ] Test BaseCollection.get() returns correct type
- [ ] Test BaseCollection.where() filtering
- [ ] Test BaseCollection.all() returns typed list
- [ ] Test lazy-load graph initialization
- [ ] Test edit() context manager with auto-save

**Type Checking:**
- [ ] Run mypy on collections/base.py
- [ ] Verify Generic[T] type hints work

## ✅ Acceptance Criteria

- [ ] collections/base.py created with BaseCollection class
- [ ] Generic[T] typing support added
- [ ] All methods from original Collection preserved
- [ ] Lazy-load _ensure_graph() pattern maintained
- [ ] collections/__init__.py exports BaseCollection
- [ ] No mypy errors

## ⚠️ Potential Conflicts

**None** - This task creates new files only, no modifications to existing code.

## 📝 Notes

**Backward Compatibility:**
- Keep original Collection class in sdk.py temporarily (task 2 will replace)
- BaseCollection is internal API (not in __all__ exports)

**Type Safety Enhancement:**
```python
T = TypeVar('T', bound=Node)

class BaseCollection(Generic[T]):
    def all(self) -> list[T]:  # Type-safe!
        pass
```

---

**Worktree:** `worktrees/task-0`
**Branch:** `feature/task-0-base-collection`

🤖 Auto-created via Contextune parallel execution

---

### Task 1: Create FeatureCollection and SpikeCollection

```yaml
---
id: task-1
priority: high
status: pending
dependencies: []
labels:
  - parallel-execution
  - auto-created
  - priority-high
---
```

# Create FeatureCollection and SpikeCollection

## 🎯 Objective

Create specialized collection classes `FeatureCollection` and `SpikeCollection` that extend BaseCollection and add `create()` builder methods. This fixes the AttributeError bug where Collection.create() was missing.

## 🛠️ Implementation Approach

**Pattern from research:**
- Repository pattern with builder integration (cosmic-python model)
- Each collection returns its specific builder type (FeatureBuilder, SpikeBuilder)
- Follow existing FeatureCollection pattern from sdk.py:464-515

**Libraries:**
- No new dependencies - use existing builders from builders/ module

**Pattern to follow:**
- **File:** `sdk.py:471` - FeatureCollection.create() returns FeatureBuilder
- **Description:** Collection wraps builder, provides create() factory method

## 📁 Files to Touch

**Create:**
- `src/python/htmlgraph/collections/feature.py` - FeatureCollection with create()
- `src/python/htmlgraph/collections/spike.py` - SpikeCollection with create()

**Read (for reference):**
- `src/python/htmlgraph/sdk.py` (lines 464-515) - Existing FeatureCollection pattern
- `src/python/htmlgraph/builders/feature.py` - FeatureBuilder to instantiate
- `src/python/htmlgraph/builders/spike.py` - SpikeBuilder to instantiate

**Modify:**
- `src/python/htmlgraph/collections/__init__.py` - Add FeatureCollection, SpikeCollection exports

## 🧪 Tests Required

**Unit:**
- [ ] Test FeatureCollection.create() returns FeatureBuilder
- [ ] Test SpikeCollection.create() returns SpikeBuilder
- [ ] Test builder.save() adds to collection
- [ ] Test fluent chaining: `.create("Title").set_priority("high").save()`

**Integration:**
- [ ] Test SDK.features.create() workflow end-to-end
- [ ] Test SDK.spikes.create() workflow end-to-end

## ✅ Acceptance Criteria

- [ ] FeatureCollection.create() method implemented
- [ ] SpikeCollection.create() method implemented
- [ ] Both extend BaseCollection (from task-0)
- [ ] Builders properly instantiated and returned
- [ ] All builder fluent API methods work
- [ ] collections/__init__.py exports both classes

## ⚠️ Potential Conflicts

**Dependency on Task 0:**
- Needs BaseCollection from task-0 to extend
- **Mitigation:** Task-1 can develop independently, will merge after task-0

**Files:**
- `collections/__init__.py` - Both tasks may edit
- **Mitigation:** Task-0 creates base export, task-1 adds specialized exports (append-only, no conflicts)

## 📝 Notes

**This fixes the smart_plan() bug:**
```python
# Before (broken):
spike = self.spikes.create(title)  # AttributeError: Collection has no 'create'

# After (working):
spike = self.spikes.create(title)  # Returns SpikeBuilder ✅
```

**Implementation Pattern:**
```python
class SpikeCollection(BaseCollection[Spike]):
    def create(self, title: str, **kwargs) -> SpikeBuilder:
        from htmlgraph.builders.spike import SpikeBuilder
        return SpikeBuilder(self._sdk, title, **kwargs)
```

---

**Worktree:** `worktrees/task-1`
**Branch:** `feature/task-1-specialized-collections`

🤖 Auto-created via Contextune parallel execution

---

### Task 2: Update SDK to use modular imports

```yaml
---
id: task-2
priority: high
status: pending
dependencies: ["task-0", "task-1"]
labels:
  - sequential-execution
  - auto-created
  - priority-high
---
```

# Update SDK to use modular imports

## 🎯 Objective

Refactor sdk.py to replace inline FeatureBuilder and Collection classes with imports from builders/ and collections/ modules. Maintain 100% backward compatibility with public API.

## 🛠️ Implementation Approach

**Pattern from research:**
- Import from new modules instead of defining inline
- Remove duplicate FeatureBuilder (sdk.py:56-209) - use builders.feature instead
- Remove Collection (sdk.py:212-462) - use collections.base instead
- Update FeatureCollection instantiation to use collections.feature

**Libraries:**
- No new dependencies - importing from new modules

**Pattern to follow:**
- **File:** `src/python/htmlgraph/__init__.py:35` - Public API export pattern
- **Description:** Internal imports don't affect __all__ exports

## 📁 Files to Touch

**Modify:**
- `src/python/htmlgraph/sdk.py`
  - Remove lines 56-209 (FeatureBuilder - now in builders/)
  - Remove lines 212-462 (Collection - now in collections/)
  - Add imports from builders.feature, builders.spike
  - Add imports from collections.base, collections.feature, collections.spike
  - Update FeatureCollection(self, sdk) → use new module
  - Update self.spikes = Collection(...) → use SpikeCollection

**No new files created** - Only modifying sdk.py

## 🧪 Tests Required

**Unit:**
- [ ] Test SDK.features.create() still works
- [ ] Test SDK.spikes.create() still works (was broken, should work now!)
- [ ] Test SDK collection methods (where, all, get, edit)

**Regression:**
- [ ] Verify start_planning_spike() works (was broken in 0.7.4)
- [ ] Verify smart_plan() works (was broken in 0.7.4)

**Import Validation:**
- [ ] No circular imports (check with import sdk in fresh Python shell)
- [ ] mypy validates all type hints

## ✅ Acceptance Criteria

- [ ] sdk.py reduced by ~450 lines (removed duplicates)
- [ ] All imports from builders/ and collections/ modules
- [ ] SDK class functionality unchanged
- [ ] Public API preserved (SDK.features, SDK.spikes interfaces)
- [ ] No circular import errors
- [ ] mypy type checking passes

## ⚠️ Potential Conflicts

**Dependencies:**
- Requires task-0 (BaseCollection) and task-1 (SpikeCollection) to be merged first
- **Mitigation:** This task runs sequentially after both complete

**Files:**
- `sdk.py` - Heavily modified (450 lines removed, imports added)
- **Mitigation:** No other tasks touch sdk.py

## 📝 Notes

**Critical backward compatibility:**
- SDK public methods unchanged: `SDK(agent="claude")`, `sdk.features.create()`
- Internal implementation swapped (inline → imports) - users won't notice

**Lines to remove from sdk.py:**
```python
# DELETE: Lines 56-209 (FeatureBuilder)
class FeatureBuilder:
    ...

# DELETE: Lines 212-462 (Collection)
class Collection:
    ...

# REPLACE with imports:
from htmlgraph.builders.feature import FeatureBuilder
from htmlgraph.builders.spike import SpikeBuilder
from htmlgraph.collections.base import BaseCollection
from htmlgraph.collections.feature import FeatureCollection
from htmlgraph.collections.spike import SpikeCollection
```

**Instantiation changes:**
```python
# Before:
self.features = FeatureCollection(self)
self.spikes = Collection(self, "spikes", "spike")  # Missing create()!

# After:
self.features = FeatureCollection(self)
self.spikes = SpikeCollection(self)  # Has create() method ✅
```

---

**Worktree:** `worktrees/task-2`
**Branch:** `feature/task-2-sdk-imports`

🤖 Auto-created via Contextune parallel execution

---

### Task 3: Update __init__.py exports and run tests

```yaml
---
id: task-3
priority: blocker
status: pending
dependencies: ["task-0", "task-1", "task-2"]
labels:
  - sequential-execution
  - testing
  - auto-created
  - priority-blocker
---
```

# Update __init__.py exports and run tests

## 🎯 Objective

Update `__init__.py` to export builders and collections if needed for public API, run full test suite (371+ tests), fix any issues, and validate zero breaking changes to public API.

## 🛠️ Implementation Approach

**Pattern from research:**
- Keep __all__ exports minimal (only public API)
- Builders and collections are internal unless documented
- Run pytest with coverage to catch regressions

**Testing Strategy:**
- Run full test suite: `uv run pytest tests/python/ -v`
- Check for import errors
- Verify backward compatibility

## 📁 Files to Touch

**Modify:**
- `src/python/htmlgraph/__init__.py`
  - Review __all__ exports (add builders/collections if public)
  - Verify all existing exports still work

**Test:**
- Run entire `tests/python/` directory
- May need to fix import paths in test files

**Documentation:**
- `AGENTS.md` - Update SDK section if needed
- `README.md` - Note modular structure

## 🧪 Tests Required

**Full Test Suite:**
- [ ] Run `uv run pytest tests/python/ -v`
- [ ] All 371+ tests must pass
- [ ] No new warnings or errors

**Import Validation:**
- [ ] Test `from htmlgraph import SDK` works
- [ ] Test `from htmlgraph import Node, Edge, Step` works
- [ ] Test internal imports: `from htmlgraph.builders import FeatureBuilder`

**Backward Compatibility:**
- [ ] Verify examples in AGENTS.md still work
- [ ] Test SDK fluent API: `sdk.features.create("X").set_priority("high").save()`
- [ ] Test collection methods: `sdk.features.where(status="todo")`

## ✅ Acceptance Criteria

- [ ] All 371+ tests passing
- [ ] Zero import errors
- [ ] __all__ exports validated
- [ ] Backward compatibility confirmed
- [ ] No mypy errors
- [ ] Documentation updated with new structure
- [ ] Feature feat-b2a4f00e marked as complete

## ⚠️ Potential Conflicts

**None** - This is the final validation task, runs after all others complete.

## 📝 Notes

**If tests fail:**
1. Check import paths in test files
2. Verify builders/collections are importable
3. Check for circular imports
4. Review type hints (mypy errors)

**Documentation updates:**
- Mention modular structure in README
- Update AGENTS.md SDK examples if imports change
- Note in CLAUDE.md that SDK is now modular

**Success metrics:**
- ~450 lines removed from sdk.py
- ~60% code duplication eliminated via BaseBuilder/BaseCollection
- Zero breaking changes to public API
- All tests passing

---

**Worktree:** `worktrees/task-3`
**Branch:** `feature/task-3-validation`

🤖 Auto-created via Contextune parallel execution

---

## References

- [Azure SDK Python Design Guidelines](https://azure.github.io/azure-sdk/python_design.html)
- [Python Typing Generics](https://typing.python.org/en/latest/reference/generics.html)
- [Cosmic Python Repository Pattern](https://www.cosmicpython.com/book/chapter_02_repository.html)
- [Feature feat-b2a4f00e: SDK Modularization](feat-b2a4f00e.html)

---

📋 **Plan created in extraction-optimized format!**

**Plan Summary:**
- 4 total tasks
- 2 can run in parallel (task-0, task-1)
- 2 have dependencies (task-2 → task-3 sequential)
- Conflict risk: **Low**

**Tasks by Priority:**
- Blocker: task-3
- High: task-0, task-1, task-2

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
│   └── task-3.md
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