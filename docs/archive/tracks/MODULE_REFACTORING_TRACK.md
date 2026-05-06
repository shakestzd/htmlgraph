# Track: Module Refactoring & Code Standards Enforcement

**Track ID**: module-refactoring-2026Q1
**Created**: 2026-03-15
**Status**: Planning
**Priority**: High
**Estimated Phases**: 5 parallel execution lanes

---

## Executive Summary

Wipnote has 132,291 lines across 308 Python files. **15 modules exceed 1,000 lines** (the top 3 exceed 2,000 lines), violating industry standards of 300-500 lines per module. Additionally, **5 utility functions are duplicated 2-3x** across the codebase, and several custom implementations can be replaced by existing dependencies or standard library features.

This track addresses three goals:
1. **Refactor oversized modules** into focused, single-responsibility files
2. **Eliminate code duplication** by consolidating shared utilities
3. **Enforce standards going forward** via tooling, agent instructions, and pre-commit hooks

---

## Industry Standards Reference

| Metric | Standard | Current State |
|--------|----------|---------------|
| Module size | 200-500 lines | 15 modules >1,000 lines |
| Function length | 10-20 lines (max 50) | Not yet measured |
| Class length | 100-200 lines | Not yet measured |
| Cyclomatic complexity | <10 per function | Not yet measured |
| Responsibilities per module | 1 (SRP) | Many modules have 5-10+ |

---

## Phase 1: Shared Utilities Consolidation (Parallel Lane A)

**Goal**: Eliminate duplicated code by creating canonical shared modules.

### 1A. Formatting Utilities → `src/python/wipnote/utils/formatting.py`

**Problem**: `format_number`, `format_duration`, `format_bytes`, `truncate_text`, `format_timestamp` are implemented **3 times identically** in:
- `api/templates.py` (lines 44-184)
- `api/filters.py` (lines 10-58)
- `api/main.py` (lines 246-281)

Plus similar `_format_duration` in:
- `cli/analytics.py:1394`
- `transcript_analytics.py:149`

**Action**:
1. Create `src/python/wipnote/utils/formatting.py` with canonical implementations
2. Replace all 5 locations with imports from the shared module
3. Add unit tests for formatting functions

**Existing dependency opportunity**: `humanize` package provides `naturalsize()`, `naturaldelta()`, `intcomma()` which could replace custom `format_bytes`, `format_duration`, `format_number`. However, adding a new dependency for formatting is marginal — the custom code is simple and well-understood. **Recommendation: Keep custom code but consolidate to one location.**

### 1B. Truncation Utilities → consolidate into `utils/formatting.py`

**Problem**: 5 different truncation implementations:
- `api/templates.py:86` — `truncate_text`
- `api/filters.py:31` — `truncate_text`
- `error_handler.py:122` — `truncate_if_needed`
- `http_hook.py:163` — `_truncate`
- `ingest/claude_code.py:473` — `_truncate_tool_input`

**Action**:
1. Create `truncate(text, max_len, suffix="...")` in shared module
2. Create `truncate_recursive(obj, max_len, max_depth)` for nested structures
3. Replace all 5 implementations

### 1C. JSON Utilities → consolidate to `utils/json.py`

**Problem**: Two JSON utility modules:
- `json_utils.py` (root, simpler)
- `api/json_utils.py` (comprehensive `JSONHandler` class)

**Action**:
1. Move comprehensive version to `utils/json.py`
2. Re-export from `api/json_utils.py` for backward compatibility
3. Remove root `json_utils.py`, update imports

**Existing dependency**: `orjson` is already a dependency and handles fast JSON. The custom code adds validation and subsetting — **keep custom code, just consolidate location.**

### 1D. Cache Consolidation

**Problem**: 3 cache implementations:
- `api/cache.py:41` — `QueryCache` (basic TTL)
- `repositories/shared_cache_memory.py:21` — `MemorySharedCache` (LRU + TTL + thread-safe)
- `api/main.py:33` — `QueryCache` (duplicate of cache.py)

**Action**:
1. Keep `MemorySharedCache` as the canonical implementation (most robust)
2. Make `QueryCache` a thin wrapper or alias
3. Remove duplicate in `api/main.py`

**Existing dependency**: `fastapi-cache2` is already a dependency for HTTP-level caching. The in-memory caches serve a different purpose (query result caching). **Keep custom cache but eliminate duplicates.**

---

## Phase 2: Critical Module Splits (Parallel Lane B)

**Goal**: Split the 3 largest modules (>2,000 lines) into focused sub-modules.

### 2A. `session_manager.py` (2,918 lines → 4-5 modules)

**Current responsibilities** (God Object):
- Session lifecycle (start/end/resume/suspend)
- Smart attribution scoring
- Drift detection
- WIP limit enforcement
- Auto-completion checking
- Session deduplication
- Activity tracking/linking
- Spike auto-creation
- Error tracking
- HTML serialization

**Proposed split**:
```
src/python/wipnote/session/
├── __init__.py              # Re-exports for backward compat
├── manager.py               # Core lifecycle (~600 lines)
├── attribution.py           # Smart attribution scoring (~500 lines)
├── drift.py                 # Drift detection (~400 lines)
├── linking.py               # Activity tracking & linking (~400 lines)
├── wip.py                   # WIP limits & auto-completion (~300 lines)
└── serialization.py         # HTML serialization (~300 lines)
```

**Backward compatibility**: Keep `session_manager.py` as a re-export shim during transition, then deprecate.

### 2B. `models.py` (2,427 lines → 4 modules)

**Current state**: 18+ unrelated model classes in one file.

**Proposed split**:
```
src/python/wipnote/models/
├── __init__.py              # Re-exports ALL models (backward compat)
├── base.py                  # Enums (WorkType, SpikeType, etc.), Node, Edge, Step
├── work_items.py            # Spike, Chore, Todo, Graph
├── session.py               # Session, ActivityEntry, ErrorEntry, ContextSnapshot
└── analytics.py             # Pattern, SessionInsight, AggregatedMetric
```

**Note**: `models/session.py` already exists at 814 lines — review for overlap and merge.

### 2C. `graph.py` (2,082 lines → 3 modules)

**Current state**: Mixed I/O, algorithms, queries, indexing, transactions.

**Proposed split**:
```
src/python/wipnote/graph/
├── __init__.py              # Re-exports Graph class
├── core.py                  # Graph class: I/O, node/edge CRUD (~700 lines)
├── algorithms.py            # Already exists (597 lines) — move BFS/shortest-path here
├── queries.py               # Already exists (581 lines) — move CSS selector queries here
├── indexing.py              # Index management, caching (~300 lines)
└── transactions.py          # Snapshot/transaction support (~300 lines)
```

**Good news**: `graph/algorithms.py` and `graph/queries.py` already exist as companion modules. The refactoring is partially done — just need to move remaining logic out of `graph.py`.

---

## Phase 3: High-Priority Module Splits (Parallel Lane C)

**Goal**: Split 7 modules in the 1,000-1,800 line range.

### 3A. `hooks/event_tracker.py` (1,828 lines)

Split into:
- `hooks/event_recording.py` — Event persistence to SQLite
- `hooks/event_processor.py` — Event normalization and enrichment
- `hooks/model_detection.py` — AI model identification strategies

### 3B. `session_context.py` (1,646 lines)

Split into:
- `session/context_builder.py` — Context assembly for AI agents
- `session/version_check.py` — Installed vs PyPI version checking
- `session/environment.py` — Environment detection & git status

### 3C. `cli/analytics.py` (1,580 lines)

Split into separate command files:
```
cli/commands/
├── cost_analysis.py
├── cigs_status.py
├── transcript.py
├── sync_docs.py
└── search.py
```

### 3D. `api/services.py` (1,403 lines)

Split into:
- `api/services/activity.py` — ActivityService
- `api/services/orchestration.py` — OrchestrationService
- `api/services/analytics.py` — AnalyticsService

### 3E. `server.py` (1,434 lines)

Split into:
- `api/handlers.py` — HTTP request handling
- `api/server.py` — Server lifecycle, port management
- `api/static.py` — Static file and dashboard serving

### 3F. `cli/core.py` (1,371 lines)

Split 11+ commands into `cli/commands/` directory (one file per command group).

### 3G. `hooks/pretooluse.py` (1,313 lines)

Split into:
- `hooks/pretooluse.py` — Core PreToolUse event creation (~500 lines)
- `hooks/orchestration_validator.py` — CIGS enforcement, validation
- `hooks/task_resolution.py` — Parent resolution, subagent detection

---

## Phase 4: Dependency Optimization (Parallel Lane D)

**Goal**: Identify custom code that can be replaced by existing dependencies or well-established packages.

### 4A. Already Well-Utilized Dependencies (No Changes Needed)

| Dependency | Usage | Assessment |
|------------|-------|------------|
| **pydantic** | Data validation across project | Excellent usage |
| **tenacity** | Retry with backoff in `decorators.py` | Properly wraps tenacity |
| **orjson** | Fast JSON serialization | Used appropriately |
| **structlog** | Structured logging | Good integration |
| **rich** | CLI output formatting | Well-utilized |
| **collections** (stdlib) | 60 imports (Counter, defaultdict, deque) | Heavy, appropriate use |

### 4B. Standard Library Underutilization

| Module | Opportunity | Files Affected |
|--------|-------------|----------------|
| **functools.lru_cache** | Memoize expensive analytics computations | `analytics/strategic/pattern_detector.py`, `analytics_index.py` |
| **itertools.batched** (3.12+) / **itertools.islice** | Replace manual chunking loops in batch processing | `hooks/event_tracker.py`, `cli/work/ingest.py` |
| **textwrap.shorten** | Replace some custom truncation logic | `api/templates.py`, `api/filters.py` |
| **dataclasses.asdict** | Replace manual dict conversion in some models | Various model files |

**Recommendation**: Use `textwrap.shorten()` from stdlib as the base for `truncate_text()` — it handles word boundaries and ellipsis natively. Wrap it in a thin utility for consistent behavior.

### 4C. Potential New Dependencies — Analysis

#### **humanize** (for `format_duration`, `format_bytes`, `format_number`)
- **PyPI**: 350M+ downloads/month, actively maintained
- **What it provides**: `naturalsize("1000000")` → "1.0 MB", `naturaldelta(timedelta(hours=3))` → "3 hours"
- **Assessment**: Custom formatting is simple and well-understood (6 functions, ~80 lines total after consolidation). Adding `humanize` would save ~80 lines but add a dependency.
- **Recommendation**: **Keep custom code.** The functions are trivial, well-tested, and adding a dependency for 80 lines of simple formatting is not worth the maintenance burden.

#### **cachetools** (for cache implementations)
- **PyPI**: 100M+ downloads/month, part of Google's ecosystem
- **What it provides**: `TTLCache`, `LRUCache`, thread-safe decorators
- **Assessment**: `MemorySharedCache` (396 lines) reimplements TTL+LRU cache with thread safety. `cachetools.TTLCache` provides this in ~5 lines of configuration.
- **Recommendation**: **Consider adopting.** This would eliminate ~350 lines of custom cache code and provide battle-tested LRU+TTL eviction. However, the custom cache has Wipnote-specific features (metrics, namespace isolation). **Evaluate in a spike** — if >80% of features can use `cachetools`, adopt it.

#### **radon** (for complexity measurement — dev dependency only)
- **PyPI**: Well-established Python complexity analyzer
- **What it provides**: Cyclomatic complexity, maintainability index, raw metrics per function/class/module
- **Assessment**: Would enable automated complexity checking in CI.
- **Recommendation**: **Add as dev dependency.** Use in the enforcement script (Phase 5) to measure function complexity alongside module line counts.

#### **wily** (for complexity tracking over time — dev dependency only)
- **PyPI**: Tracks code complexity metrics across git history
- **What it provides**: Complexity trends, diff complexity reports
- **Assessment**: Useful for tracking whether refactoring is reducing complexity.
- **Recommendation**: **Optional.** Nice for dashboarding but not critical. Can add later.

### 4D. Custom Code to Keep

| Custom Code | Why Keep |
|-------------|----------|
| `ids.py` (ID generation) | Domain-specific ID format, intentional design |
| `query_builder.py` | Domain-specific CSS selector queries |
| `atomic_ops.py` | Uses stdlib correctly, well-designed |
| `decorators.py` (retry) | Thin wrapper over tenacity, adds domain context |
| `graph/algorithms.py` | Domain-specific graph algorithms |

---

## Phase 5: Standards Enforcement (Parallel Lane E)

**Goal**: Ensure all future development adheres to module size and quality standards.

### 5A. Enforcement Script: `scripts/check-module-size.py`

Checks:
- **Module line count**: Warn >300, fail >500, critical >1000
- **Function length**: Warn >30, fail >50
- **Class length**: Warn >200, fail >300
- **Cyclomatic complexity**: Warn >7, fail >10 (using radon)

Exit codes: 0 (pass), 1 (warnings), 2 (failures)

### 5B. Pre-commit Hook Integration

Add to `.pre-commit-config.yaml`:
```yaml
- repo: local
  hooks:
    - id: module-size
      name: check module sizes
      entry: uv run python scripts/check-module-size.py
      language: system
      pass_filenames: false
      files: ^src/python/wipnote/
      stages: [pre-commit]
```

### 5C. Agent Definition Updates

Add module size awareness to ALL agent definitions in `packages/claude-plugin/agents/`:

**All agents** get this standard block:
```markdown
## Module Size Standards
- Target: 200-500 lines per module
- Hard limit: 500 lines for new modules
- If your changes would push a module >500 lines, split it first
- Functions: max 50 lines, target 10-20
- Classes: max 300 lines, target 100-200
- One responsibility per module (Single Responsibility Principle)
```

**Agent-specific additions**:
- **opus-coder.md**: "When assigned refactoring work, use the split patterns documented in MODULE_REFACTORING_TRACK.md"
- **sonnet-coder.md**: "Before adding to a module >400 lines, evaluate if it should be split first"
- **haiku-coder.md**: "Decline work that would push a module >500 lines — escalate to Sonnet/Opus"
- **test-runner.md**: "After tests pass, run `scripts/check-module-size.py` on changed files"
- **researcher.md**: "When researching a module, note its size and recommend refactoring if >500 lines"
- **debugger.md**: "If a bug is in a module >1000 lines, recommend refactoring as part of the fix"

### 5D. System Prompt Updates

Add to `packages/claude-plugin/.claude-plugin/system-prompt-default.md`:
```markdown
## Module Size Standards (Enforced)
- New modules: max 500 lines
- Existing modules: reduce toward 300-500 lines during any modification
- Never add code to a module >1000 lines without splitting first
- Run `scripts/check-module-size.py` before committing
```

### 5E. Code Hygiene Rules Update

Add to `.claude/rules/code-hygiene.md`:
```markdown
## Module Size & Complexity Standards

### Line Count Limits
| Metric | Target | Warning | Fail |
|--------|--------|---------|------|
| Module | 200-500 | >300 | >500 (new) |
| Function | 10-20 | >30 | >50 |
| Class | 100-200 | >200 | >300 |

### Enforcement
- `scripts/check-module-size.py` runs in pre-commit
- Existing large modules are grandfathered but tracked for refactoring
- Any modification to a grandfathered module must not increase its size
```

---

## Parallel Execution Plan

All 5 phases can execute concurrently across separate branches:

```
Week 1-2:
├── Lane A: Utilities consolidation (Phase 1)     ← Independent
├── Lane D: Dependency analysis spike (Phase 4)    ← Independent
└── Lane E: Enforcement scripts & docs (Phase 5)   ← Independent

Week 3-4:
├── Lane B: Critical splits (Phase 2A-2C)          ← After Lane A (shared utils exist)
└── Lane E: Agent/prompt updates (Phase 5C-5E)     ← After scripts created

Week 5-6:
└── Lane C: High-priority splits (Phase 3A-3G)     ← After Lane B patterns established

Ongoing:
└── All lanes: Enforce standards on new code        ← After Lane E complete
```

### Dependencies Between Lanes

```
Lane A (Utils) ──────────────┐
                             ├──→ Lane B (Critical Splits)
Lane D (Deps Analysis) ─────┘         │
                                      ├──→ Lane C (High-Priority Splits)
Lane E (Enforcement) ────────────────┘
```

---

## Success Metrics

| Metric | Current | Target | Measurement |
|--------|---------|--------|-------------|
| Modules >1000 lines | 15 | 0 | `scripts/check-module-size.py` |
| Modules >500 lines | ~30 | <5 (grandfathered) | Same script |
| Duplicated utility code | 5 instances | 0 | Manual audit |
| New dependencies added | 0 | 1-2 (radon, possibly cachetools) | pyproject.toml |
| Agent definitions with size guidance | 0/6 | 6/6 | Manual check |

---

## Risk Mitigation

1. **Import breakage**: Every split module provides backward-compatible re-exports via `__init__.py`
2. **Test failures**: Run full test suite after each module split; never split without tests passing
3. **Merge conflicts**: Each lane works on different files; coordinate if touching shared imports
4. **Over-engineering**: Don't split modules below 200 lines; don't add abstractions for single-use code
5. **Dependency bloat**: Only add dependencies that replace >200 lines of custom code OR provide critical correctness guarantees

---

## Files Modified by This Track

### New Files
- `scripts/check-module-size.py` — Enforcement script
- `docs/tracks/MODULE_REFACTORING_TRACK.md` — This document
- `src/python/wipnote/utils/formatting.py` — Consolidated formatting
- Multiple new split modules (per Phase 2 & 3)

### Modified Files
- `.pre-commit-config.yaml` — Add module-size hook
- `.claude/rules/code-hygiene.md` — Add module standards
- `packages/claude-plugin/agents/*.md` — Add size guidance (6 files)
- `packages/claude-plugin/.claude-plugin/system-prompt-default.md` — Add standards
- `AGENTS.md` — Add module organization section
- `pyproject.toml` — Add radon dev dependency
