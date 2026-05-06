# Token-Based Planning for AI Agent Development

**CRITICAL DIRECTIVE: All work planning and estimation must be based on TOKEN BUDGET, not calendar time.**

## Foundational Principle

In AI agent development, the meaningful constraint is **token consumption**, not calendar time.

### Why Time-Based Planning Fails

1. **Parallelization destroys time estimates**
   - One agent sequentially: 5 weeks
   - Five agents in parallel: Same 5 calendar days
   - Token cost: 193,000 → 118,000 tokens (39% savings)
   - **Time is not predictive of work size**

2. **Agent execution time varies randomly**
   - Phase 2.1 could take 2 hours or 4 hours (doesn't matter)
   - Token cost is 15,000 tokens regardless
   - While agent works, other agents work on dependent phases
   - **Calendar time is decoupled from effort measurement**

3. **Scaling is non-linear with humans but linear with agents**
   - Adding 1 human: +5-10% overhead from coordination
   - Adding 1 agent: Pure additive benefit (independent work)
   - 5 agents ≠ 1/5 the time, but ≈ 1/3 the tokens
   - **Agent economics are fundamentally different**

## Token Budget as Primary Constraint

### Three Budget Types

**1. Phase Budget** (Multi-feature initiatives)
```
Phase 2: Repository Implementation
├─ Sequential cost: 193,000 tokens (one agent)
├─ Parallel cost: 118,000 tokens (5 agents at 61% efficiency)
├─ Budget ceiling: 130,000 tokens (12% safety margin)
└─ Decision: Parallelize (saves 39% tokens)
```

**2. Feature Budget** (Individual deliverables)
```
Feature: Analytics Double-Load Fix
├─ Token budget: 2,000 tokens
├─ Agents: 1 (no parallelization benefit)
└─ Priority: Fix immediately (high ROI)
```

**3. Task Budget** (Delegated work units)
```
Task: Implement FeatureRepository Interface
├─ Token budget: 15,000 tokens
├─ Part of: Phase 2.1 (parallelizable)
├─ Dependencies: None (can start immediately)
└─ Parallelizable with: TrackRepository, AnalyticsRepository
```

### Real-World Token Costs from Wipnote

**Phase 1 - Analysis (Complete)**
- 6 parallel agents: 45,000 tokens total
- Cost per agent: ~7,500 tokens
- Efficiency: 100% (parallel, no dependencies)

**Phase 2 - Repository Implementation (Planned)**
- Phase 2.1 (Interface design): 15,000 tokens
- Phase 2.2 (Concrete impl): 30,000 tokens
- Phase 2.3 (SDK migration): 35,000 tokens (sequential blocker)
- Phase 2.4 (CLI migration): 10,000 tokens (sequential blocker)
- Phase 2.5 (Cleanup): 20,000 tokens

Sequential total: 193,000 tokens
Parallel total: 118,000 tokens (5 agents, 61% efficiency)

## Parallelization Efficiency Model

Not all token work scales linearly with agents. Account for overhead:

### Efficiency by Agent Count

| Agents | Efficiency | Notes |
|--------|-----------|-------|
| 1 | 100% | Baseline (no parallelization) |
| 2 | 70-80% | Minimal coordination overhead |
| 3 | 65-75% | Task boundary costs increase |
| 4 | 60-70% | Dependency complexity grows |
| 5+ | 50-65% | Coordination dominates |

### Why Efficiency < 100%

1. **Task boundaries**: Passing context, writing spikes (3-5% overhead per agent)
2. **Sequential dependencies**: Some phases must finish before others (unavoidable delays)
3. **Integration testing**: Must run sequentially (can't parallelize verification)
4. **Merge consolidation**: Combining 5 agents' work into coherent whole (5-10% overhead)

### Example: Phase 2 Parallelization

**Sequential (1 agent):**
```
Phase 2.1: 20,000 tokens
Phase 2.2: 40,000 tokens (depends on 2.1)
Phase 2.3: 45,000 tokens (depends on 2.1, 2.2)
Phase 2.4: 15,000 tokens (depends on 2.3)
Phase 2.5: 25,000 tokens (depends on all)
Total: 193,000 tokens (20+40+45+15+25)
Duration: 5 weeks (wall-clock time irrelevant)
```

**Parallel (5 agents, 61% efficiency):**
```
Bucket 1 (simultaneous): Phase 2.1, 2.2.1, 2.2.2, 2.2.3
  Cost: 15,000 + 10,000 + 10,000 + 10,000 = 45,000 tokens

Bucket 2 (depends on 2.1, 2.2): Phase 2.3, 2.4.1, 2.4.2
  Cost: 35,000 + 5,000 + 5,000 = 45,000 tokens

Bucket 3 (depends on all): Phase 2.5
  Cost: 20,000 tokens

Coordination/integration: 8,000 tokens

Total: 118,000 tokens
Efficiency: 118,000 / (193,000 / 5) = 61%
Savings: 39% tokens vs sequential
```

## Decision Framework: When to Parallelize

### Parallelize When:
- ✅ Token budget > 50,000 (overhead worthwhile)
- ✅ Phases have independent deliverables
- ✅ Minimal sequential dependencies
- ✅ Integration cost < parallelization savings
- ✅ **Example: Phase 2 (193K tokens, 39% savings) → YES**

### Don't Parallelize When:
- ❌ Token budget < 20,000 (overhead dominates)
- ❌ Tight sequential dependencies (A must finish before B)
- ❌ Single complex problem (not decomposable)
- ❌ Heavy integration testing needed
- ❌ **Example: Analytics Double-Load Fix (2K tokens) → NO**

## Token-Based Metrics (Replace Time-Based Ones)

### What NOT to Track

❌ "Days until completion" — Meaningless with parallelization
❌ "Percentage of tasks done" — Misleading (tasks have different costs)
❌ "Burndown chart" — Doesn't apply to parallel agent work
❌ "Sprint velocity" — Replaced by token velocity

### What TO Track

**1. Token Velocity**
```
"Burning 18,000 tokens/day with 3 agents"
"Parallel efficiency: 62% (target: 61%, within bounds)"
"Current burn rate sustainable for remaining 25 days of runway"
```

**2. Budget Utilization**
```
"Phase 2 Budget: 118,000 tokens"
"Spent: 67,000 tokens (57%)"
"Remaining: 51,000 tokens (43%)"
"Buffer: 12,000 tokens remaining above ceiling"
```

**3. Cost Per Agent**
```
"Phase 2.1 cost: 15,000 tokens / 3 agents = 5,000 tokens/agent"
"Efficiency: 73% (15,000 / (20,000/3) = 2.25x parallelization)"
```

**4. Phase Progress**
```
"Phase 2.1: 8,000/15,000 tokens (53% complete)"
"Phase 2.2: Waiting on 2.1 completion"
"Phase 2.3: Queued, will start when 2.1+2.2 finish"
```

## How to Estimate Token Budgets

### Model 1: Historical Data

If you've completed similar work, use actual costs:
```
Previous Phase: 50,000 tokens
Similar Phase: Budget 48,000 tokens
Confidence: High
```

### Model 2: Complexity Breakdown

Decompose into subtasks, estimate each:
```
Feature: Repository Pattern Implementation
├─ Interface design: 3,000 tokens
├─ Implementation: 12,000 tokens
├─ Tests: 4,000 tokens
├─ Documentation: 1,000 tokens
└─ Integration: 2,000 tokens
Total: 22,000 tokens
Safety margin (15%): 25,300 tokens
```

### Model 3: Precedent + Adjustment

Use similar work, adjust for complexity:
```
Base precedent: 10,000 tokens (simple feature)
This feature complexity: 2x (more components, more tests)
Edge cases identified: +3,000 tokens
Estimated budget: (10,000 × 2) + 3,000 = 23,000 tokens
```

### Model 4: Rule of Thumb by Feature Type

- **Bug fix**: 1,000-5,000 tokens
- **Small feature**: 5,000-15,000 tokens
- **Medium feature**: 15,000-35,000 tokens
- **Large feature**: 35,000-75,000 tokens
- **System refactor**: 75,000-200,000 tokens

## Orchestrator Directives Update

### When Delegating Work

**OLD (WRONG):**
```python
Task(prompt="Complete Phase 2 repository implementation")
# Time estimate: 5 weeks
# User has no visibility into token cost
```

**NEW (CORRECT):**
```python
# Phase 2.1 - Can parallelize with 2.2
Task(
    prompt="Design repository interfaces: FeatureRepository, TrackRepository, AnalyticsRepository",
    description="Phase 2.1: Interface Design | Budget: 15,000 tokens | Can parallelize with Phase 2.2"
)

# Phase 2.2 - Can parallelize with 2.1
Task(
    prompt="Implement concrete repository classes with full test coverage",
    description="Phase 2.2: Concrete Implementation | Budget: 30,000 tokens | Depends on Phase 2.1"
)

# Phase 2.3 - Sequential dependency
Task(
    prompt="Migrate SDK to use repositories, maintain backward compatibility",
    description="Phase 2.3: SDK Migration | Budget: 35,000 tokens | Depends on Phase 2.1+2.2"
)
```

### At Phase Boundaries

Report actual vs. estimated tokens:
```
Phase 2.1 Complete
├─ Budget: 15,000 tokens
├─ Actual: 14,200 tokens (95% utilization)
├─ Overrun: None
└─ Status: On track for Phase 2 completion within 118,000 budget
```

## Critical Implications

### For Cost Control

- Parallelization saves 30-40% tokens on large projects
- Running single agents wastes parallelization opportunity
- Always parallelize when token budget > 50,000
- Calculate token savings before deciding to parallelize

### For Development Speed

- "Faster development" ≠ shorter calendar time (meaningless with parallelization)
- "Faster development" = lower token consumption (efficiency improvement)
- Example: 39% token savings = same calendar time, 39% less API cost

### For Resource Planning

- Don't ask "How many agents do we have?"
- Ask "How many agents can we parallelize?"
- Each additional agent reduces token cost by ~20% (due to efficiency drop-off)
- 5 agents typically optimal for large projects (diminishing returns beyond)

### For Quality Gates

- Don't block on "how long it takes"
- Block on "token budget exceeded"
- Phase 2 ceiling: 130,000 tokens (12% over 118,000 estimate)
- If approaching ceiling, escalate complexity issues for human review

## Template for Token-Based Planning

Use this structure for ALL future planning:

```
PROJECT: [Name]
PHASES: [List]

PHASE: [Phase Name]
├─ Budget (Sequential): [X] tokens (one agent)
├─ Budget (Parallel): [Y] tokens ([N] agents at [%] efficiency)
├─ Actual: [Z] tokens (when complete)
└─ Status: [In progress / Complete]

Dependencies:
├─ Phase X depends on: Phase Y
└─ Phase X is independent of: Phases A, B, C

Parallelization:
├─ Can run simultaneously with: [List phases]
├─ Sequential blocker for: [List phases]
└─ Efficiency expected: [%] (use 60% as default)

Risk & Contingency:
├─ Identified risks: [List]
├─ Contingency budget: [%] over base estimate
└─ Escalation criteria: Exceed ceiling by [X]%

Success Metrics:
├─ Budget utilization < 110% (acceptable)
├─ Efficiency within 5% of target
└─ All integration tests passing
```

## Example: Using Token-Based Planning for Wipnote Phase 2

```
PROJECT: Wipnote Single Source of Truth Refactoring
TOTAL BUDGET: 268,000 tokens (parallel across all phases)

PHASE 2: Repository Implementation
├─ Budget (Sequential): 193,000 tokens
├─ Budget (Parallel): 118,000 tokens (5 agents, 61% efficiency)
├─ Ceiling: 130,000 tokens (12% buffer)
└─ Status: Ready to start

Phase 2.1: Interface Design
├─ Budget: 15,000 tokens
├─ Agents: 3 (FeatureRepository, TrackRepository, AnalyticsRepository)
├─ Duration: ~1 day (wall-clock, irrelevant)
└─ Deliverable: 5 interface definitions + compliance tests

Phase 2.2: Concrete Implementation
├─ Budget: 30,000 tokens
├─ Agents: 3 (Feature, Track, Analytics implementations)
├─ Blocker: Requires Phase 2.1 complete
└─ Deliverable: 3 implementation classes + 100+ unit tests

Phase 2.3: SDK Migration
├─ Budget: 35,000 tokens
├─ Agents: 2 (sequential - high coupling)
├─ Blocker: Requires Phase 2.2 complete
└─ Deliverable: SDK using repositories + backward compatibility

Phase 2.4: CLI Migration
├─ Budget: 10,000 tokens
├─ Agents: 3 (32 commands grouped into 3 buckets)
├─ Blocker: Requires Phase 2.3 complete
└─ Deliverable: 32 commands using repositories

Phase 2.5: Cleanup
├─ Budget: 20,000 tokens
├─ Agents: 2 (Graph.py removal, cache consolidation)
├─ Blocker: All prior phases
└─ Deliverable: Old code removed, caches unified

DECISION MATRIX:
- Should we parallelize Phase 2? YES (saves 39% tokens)
- Should we use 5 agents? YES (61% efficiency is optimal)
- Should we add 6th agent? NO (efficiency would drop to 55%)

CHECKPOINTS:
1. After Phase 2.1 (15,000 tokens): Review interface design
2. After Phase 2.2 (45,000 tokens cumulative): Integration testing
3. After Phase 2.3 (80,000 tokens cumulative): SDK backward compatibility
4. After Phase 2.4 (90,000 tokens cumulative): CLI validation
5. After Phase 2.5 (110,000 tokens cumulative): Full system test

RISK MITIGATION:
- Identified risk: Phase 2.3 (SDK migration) is complex
- Contingency: 10,000 token buffer (already included in ceiling)
- Escalation: If Phase 2.3 exceeds 40,000 tokens, pause and review
```

---

**REMEMBER:** Token budget is the constraint, not calendar time. Plan accordingly.
