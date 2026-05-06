# Wipnote Orchestrator - Complexity Assessment Visual Guide

## Decision Flow Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                     TASK ARRIVES AT ORCHESTRATOR                        │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    ASSESS: What type of task is this?                   │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────┐  ┌─────────────┐  ┌──────────────┐  ┌──────────┐    │
│  │ Exploration │  │  Debugging  │  │Implementation│  │  Quality │    │
│  │  (research) │  │   (errors)  │  │   (coding)   │  │ (linting)│    │
│  └──────┬──────┘  └──────┬──────┘  └──────┬───────┘  └─────┬────┘    │
│         │                │                 │                 │          │
└─────────┼────────────────┼─────────────────┼─────────────────┼──────────┘
          │                │                 │                 │
          └────────────────┴─────────────────┴─────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│              APPLY 4-FACTOR COMPLEXITY ASSESSMENT                       │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  Factor 1: FILES AFFECTED                                               │
│  ┌───────────┬─────────────┬────────────────┐                         │
│  │  1-2      │   3-8       │     10+        │                         │
│  │  files    │   files     │    files       │                         │
│  └───────────┴─────────────┴────────────────┘                         │
│      ↓             ↓              ↓                                    │
│   HAIKU        SONNET          OPUS                                    │
│                                                                         │
│  Factor 2: REQUIREMENTS CLARITY                                         │
│  ┌───────────┬─────────────┬────────────────┐                         │
│  │  100%     │  70-90%     │    <70%        │                         │
│  │  clear    │  clear      │   unclear      │                         │
│  └───────────┴─────────────┴────────────────┘                         │
│      ↓             ↓              ↓                                    │
│   HAIKU        SONNET          OPUS                                    │
│                                                                         │
│  Factor 3: COGNITIVE LOAD                                               │
│  ┌───────────┬─────────────┬────────────────┐                         │
│  │   Low     │   Medium    │     High       │                         │
│  │  (simple) │ (features)  │ (architecture) │                         │
│  └───────────┴─────────────┴────────────────┘                         │
│      ↓             ↓              ↓                                    │
│   HAIKU        SONNET          OPUS                                    │
│                                                                         │
│  Factor 4: RISK LEVEL                                                   │
│  ┌───────────┬─────────────┬────────────────┐                         │
│  │   Low     │   Medium    │     High       │                         │
│  │ (docs)    │ (business)  │  (security)    │                         │
│  └───────────┴─────────────┴────────────────┘                         │
│      ↓             ↓              ↓                                    │
│   HAIKU        SONNET          OPUS                                    │
│                                                                         │
│  AGGREGATE → COMPLEXITY LEVEL: LOW | MEDIUM | HIGH                     │
│                                                                         │
└─────────────────────────────────┬───────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    CONSIDER BUDGET CONSTRAINTS                          │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐           │
│  │  FREE          │  │  BALANCED      │  │  QUALITY       │           │
│  │  (cost first)  │  │  (default)     │  │  (best model)  │           │
│  └────────┬───────┘  └────────┬───────┘  └────────┬───────┘           │
│           │                   │                    │                   │
│      Prefer Haiku         Use Sonnet          Use Opus                 │
│      or Gemini            most times          most times               │
│                                                                         │
└─────────────────────────────────┬───────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                  LOOKUP IN DECISION MATRIX (75 combos)                  │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  KEY: (TaskType, ComplexityLevel, BudgetMode)                           │
│  VALUE: Model Name                                                      │
│                                                                         │
│  Example lookups:                                                       │
│  ├─ (QUALITY, LOW, BALANCED) → "claude-haiku"                           │
│  ├─ (IMPLEMENTATION, MEDIUM, BALANCED) → "codex"                        │
│  └─ (IMPLEMENTATION, HIGH, BALANCED) → "claude-opus"                    │
│                                                                         │
└─────────────────────────────────┬───────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                        SELECT MODEL + FALLBACKS                         │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  Primary Model: claude-sonnet                                           │
│  Fallbacks: [claude-opus, claude-haiku]                                │
│                                                                         │
│  If primary unavailable → try fallback[0]                               │
│  If fallback[0] unavailable → try fallback[1]                           │
│  If all fail → return "claude-sonnet" (safe default)                    │
│                                                                         │
└─────────────────────────────────┬───────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         DELEGATE TO MODEL                               │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  Task(model="claude-sonnet", prompt="...")                              │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Example: Real Task Flow

### Example 1: Simple Typo Fix

```
USER REQUEST: "Fix typo in README.md line 42: 'recieve' → 'receive'"

┌─ TASK TYPE: Quality (documentation fix)
├─ FACTOR 1 (Files): 1 file → LOW
├─ FACTOR 2 (Clarity): 100% clear → LOW
├─ FACTOR 3 (Cognitive): Simple text replacement → LOW
├─ FACTOR 4 (Risk): Documentation only → LOW
└─ COMPLEXITY: LOW

BUDGET: balanced (default)

DECISION MATRIX LOOKUP:
  (TaskType.QUALITY, ComplexityLevel.LOW, BudgetMode.BALANCED)
  → "claude-haiku"

FALLBACK CHAIN:
  haiku → sonnet → opus

✅ SELECTED: claude-haiku ($0.80/1M tokens)
```

### Example 2: Moderate Implementation

```
USER REQUEST: "Implement CLI command for listing sessions with pagination"

┌─ TASK TYPE: Implementation (coding)
├─ FACTOR 1 (Files): 5 files (cli.py, handlers, tests) → MEDIUM
├─ FACTOR 2 (Clarity): 80% clear (spec provided) → MEDIUM
├─ FACTOR 3 (Cognitive): Integration + testing → MEDIUM
├─ FACTOR 4 (Risk): Business logic → MEDIUM
└─ COMPLEXITY: MEDIUM

BUDGET: balanced (default)

DECISION MATRIX LOOKUP:
  (TaskType.IMPLEMENTATION, ComplexityLevel.MEDIUM, BudgetMode.BALANCED)
  → "codex"

FALLBACK CHAIN:
  codex → sonnet → opus

✅ SELECTED: codex (specialized for coding)
   Fallback: claude-sonnet if codex unavailable
```

### Example 3: Complex Architecture

```
USER REQUEST: "Design distributed event processing architecture"

┌─ TASK TYPE: Implementation (system design)
├─ FACTOR 1 (Files): 12+ files (system-wide) → HIGH
├─ FACTOR 2 (Clarity): 50% clear (needs exploration) → HIGH
├─ FACTOR 3 (Cognitive): Architectural decisions → HIGH
├─ FACTOR 4 (Risk): Affects entire system → HIGH
└─ COMPLEXITY: HIGH

BUDGET: balanced (default)

DECISION MATRIX LOOKUP:
  (TaskType.IMPLEMENTATION, ComplexityLevel.HIGH, BudgetMode.BALANCED)
  → "claude-opus"

FALLBACK CHAIN:
  opus → sonnet → haiku

✅ SELECTED: claude-opus ($15/1M tokens)
   Worth the cost for complex architecture
```

---

## Model Selection Matrix Heatmap

```
                    FREE BUDGET       BALANCED BUDGET      QUALITY BUDGET
                   ─────────────────────────────────────────────────────────
EXPLORATION
  Low              gemini            gemini               sonnet
  Medium           gemini            gemini               sonnet
  High             gemini            sonnet               opus

DEBUGGING
  Low              haiku             sonnet               opus
  Medium           haiku             sonnet               opus
  High             haiku             opus                 opus

IMPLEMENTATION
  Low              haiku             codex                opus
  Medium           haiku             codex                opus
  High             haiku             opus                 opus

QUALITY
  Low              haiku             haiku                sonnet
  Medium           haiku             sonnet               opus
  High             haiku             sonnet               opus

GENERAL
  Low              haiku             sonnet               opus
  Medium           haiku             sonnet               opus
  High             haiku             opus                 opus

Legend:
  gemini → FREE tier (cost: $0)
  haiku  → Fast, cheap ($0.80/1M tokens)
  codex  → Specialized for code
  sonnet → Default, balanced ($3/1M tokens)
  opus   → Best quality ($15/1M tokens)
```

---

## Cost Optimization Strategy

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         COST OPTIMIZATION                               │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  BASELINE: Using Opus for all tasks                                    │
│  ├─ 100 tasks/day × 5000 tokens/task = 500K tokens/day                 │
│  └─ Cost: 500K × $15/1M = $7.50/day = $225/month                       │
│                                                                         │
│  OPTIMIZED: Intelligent model selection                                │
│  ├─ 20% Haiku (100K tokens): 100K × $0.80/1M = $0.08                   │
│  ├─ 70% Sonnet (350K tokens): 350K × $3/1M = $1.05                     │
│  └─ 10% Opus (50K tokens): 50K × $15/1M = $0.75                        │
│                                                                         │
│  TOTAL OPTIMIZED: $1.88/day = $56.40/month                             │
│                                                                         │
│  💰 SAVINGS: $168.60/month (75% reduction)                              │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Fallback Chain Visualization

```
┌────────────────────────────────────────────────────────────────────────┐
│                        MODEL FALLBACK CHAINS                           │
├────────────────────────────────────────────────────────────────────────┤
│                                                                        │
│  gemini ──→ haiku ──→ sonnet ──→ opus                                 │
│             (free)    (cheap)     (best)                               │
│                                                                        │
│  codex ──→ sonnet ──→ opus                                             │
│           (balanced)  (complex)                                        │
│                                                                        │
│  copilot ──→ sonnet ──→ opus                                           │
│              (balanced) (complex)                                      │
│                                                                        │
│  haiku ──→ sonnet ──→ opus                                             │
│            (upgrade)   (highest)                                       │
│                                                                        │
│  sonnet ──→ opus ──→ haiku                                             │
│             (upgrade) (downgrade if needed)                            │
│                                                                        │
│  opus ──→ sonnet ──→ haiku                                             │
│           (downgrade) (last resort)                                    │
│                                                                        │
└────────────────────────────────────────────────────────────────────────┘

Design Philosophy:
  • Start with optimal model for task
  • Fallback to more capable model if primary fails
  • Last resort: Fallback to cheaper model (better than no model)
```

---

## Token Estimation Scale

```
┌────────────────────────────────────────────────────────────────────────┐
│                      TOKEN ESTIMATION GUIDE                            │
├────────────────────────────────────────────────────────────────────────┤
│                                                                        │
│  LOW COMPLEXITY                                                        │
│  ├─ Description: "Fix typo in README"                                 │
│  ├─ Estimated: ~500-1000 tokens                                       │
│  └─ Actual: 300-2000 tokens (depends on context)                      │
│                                                                        │
│  MEDIUM COMPLEXITY                                                     │
│  ├─ Description: "Implement JWT authentication middleware"            │
│  ├─ Estimated: ~1000-5000 tokens                                      │
│  └─ Actual: 2000-10000 tokens (depends on iterations)                 │
│                                                                        │
│  HIGH COMPLEXITY                                                       │
│  ├─ Description: "Design distributed caching architecture"            │
│  ├─ Estimated: ~5000-20000 tokens                                     │
│  └─ Actual: 10000-50000+ tokens (multiple iterations, context)        │
│                                                                        │
│  ⚠️  ESTIMATES ARE GUIDELINES ONLY                                     │
│  • Actual usage varies based on:                                      │
│    - Context size (codebase, files included)                          │
│    - Tool calls (Read, Edit, Bash, etc.)                              │
│    - Iterations (trial and error, refinement)                         │
│    - Conversation length (back-and-forth)                             │
│                                                                        │
└────────────────────────────────────────────────────────────────────────┘
```

---

## Testing Coverage Map

```
┌────────────────────────────────────────────────────────────────────────┐
│                      TEST COVERAGE SUMMARY                             │
├────────────────────────────────────────────────────────────────────────┤
│                                                                        │
│  ✅ Basic Functionality (4 tests)                                      │
│     ├─ Default model selection                                        │
│     ├─ ComplexityLevel enum                                           │
│     ├─ TaskType enum                                                  │
│     └─ BudgetMode enum                                                │
│                                                                        │
│  ✅ Core Complexity Tests (6 tests)                                    │
│     ├─ Simple task → Haiku (2 tests)                                  │
│     ├─ Moderate task → Sonnet/Codex (2 tests)                         │
│     └─ Complex task → Opus (2 tests)                                  │
│                                                                        │
│  ✅ Budget Mode Tests (2 tests)                                        │
│     ├─ FREE budget → Cheapest models                                  │
│     └─ QUALITY budget → Best models                                   │
│                                                                        │
│  ✅ Fallback Chain Tests (1 test)                                      │
│     └─ 6 models with correct fallback chains                          │
│                                                                        │
│  ✅ Token Estimation (1 test)                                          │
│     └─ Scales correctly with complexity                               │
│                                                                        │
│  ✅ Edge Cases (1 test)                                                │
│     └─ Invalid input handling                                         │
│                                                                        │
│  TOTAL: 15 tests, 436 lines, 100% passing                             │
│                                                                        │
└────────────────────────────────────────────────────────────────────────┘
```

---

## Usage Examples in Code

### Example 1: Direct Model Selection

```python
from wipnote.orchestration import select_model

# Simple task
model = select_model(
    task_type="quality",
    complexity="low",
    budget="balanced"
)
# Returns: "claude-haiku"

# Moderate implementation
model = select_model(
    task_type="implementation",
    complexity="medium",
    budget="balanced"
)
# Returns: "codex"

# Complex architecture
model = select_model(
    task_type="implementation",
    complexity="high",
    budget="balanced"
)
# Returns: "claude-opus"
```

### Example 2: With Fallback Chain

```python
from wipnote.orchestration import select_model, get_fallback_chain

primary_model = select_model("implementation", "high", "balanced")
# Returns: "claude-opus"

fallbacks = get_fallback_chain(primary_model)
# Returns: ["claude-sonnet", "claude-haiku"]

# Try primary, then fallbacks
for model in [primary_model] + fallbacks:
    try:
        result = delegate_task(model, task)
        break  # Success!
    except ModelUnavailable:
        continue  # Try next fallback
```

### Example 3: Token Estimation

```python
from wipnote.orchestration import ModelSelection

task_desc = "Implement user authentication with JWT tokens"

low_estimate = ModelSelection.estimate_tokens(task_desc, "low")
medium_estimate = ModelSelection.estimate_tokens(task_desc, "medium")
high_estimate = ModelSelection.estimate_tokens(task_desc, "high")

print(f"Low: {low_estimate} tokens")
print(f"Medium: {medium_estimate} tokens")
print(f"High: {high_estimate} tokens")
```

---

**Visual Guide Version**: 1.0
**Last Updated**: 2026-01-12
**Related Documents**: COMPLEXITY_ASSESSMENT_REPORT.md
