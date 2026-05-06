# Model Selection Guide

This guide helps you choose the right Claude model for each task based on complexity, novelty, cost, and execution characteristics.

## Quick Decision Matrix

| Task Type | Haiku | Sonnet | Opus |
|-----------|:-----:|:------:|:----:|
| Quick fix/bug | ✅ Best | ⚠️ Overkill | ❌ No |
| Feature (simple) | ✅ Best | ⚠️ OK | ❌ No |
| Feature (complex) | ⚠️ Limited | ✅ Best | ⚠️ Good |
| Architecture design | ⚠️ Limited | ✅ Best | ⚠️ Good |
| Novel problem | ❌ No | ⚠️ OK | ✅ Best |
| Research/investigation | ⚠️ Limited | ⚠️ OK | ✅ Best |
| Refactoring | ✅ Best | ⚠️ Overkill | ❌ No |
| Running tests | ✅ Best | ❌ No | ❌ No |
| Code review | ⚠️ OK | ✅ Best | ⚠️ Good |

---

## Model Profiles

### Haiku - The Delegator

**Best For:**
- Orchestration and delegation
- Quick fixes and refactoring
- Following established patterns and instructions
- Running tests and quality gates
- File operations and searches
- Simple implementations (<30 minutes)

**Strengths:**
- Excellent at following detailed instructions
- Very responsive to patterns and templates
- Fast execution (40-80ms typical)
- Lower cost per token
- Highly reliable for straightforward tasks
- Good at recognizing when to defer to larger models

**Limitations:**
- May miss subtle design trade-offs
- Limited context for novel problems
- Not ideal for deep architectural decisions
- Can miss edge cases in complex scenarios

**Speed**: ~40-80ms execution time
**Cost**: Lowest (baseline)
**Token rate**: ~50 tokens/second

**When to Use:**
```
✅ Task: "Refactor this function to improve readability"
   Why: Clear, bounded task following a pattern

✅ Task: "Fix the bug in login.py where tokens expire immediately"
   Why: Concrete problem with clear solution path

✅ Task: "Run all tests and report results"
   Why: Straightforward execution task

❌ Task: "Design a real-time collaboration system from scratch"
   Why: Novel, requires deep reasoning about trade-offs

❌ Task: "Choose between Redis vs in-memory caching"
   Why: Complex architectural decision needed
```

**Example Delegation:**
```python
# Perfect use of Haiku—clear, bounded task
Task(
    prompt="""
    Fix the authentication bug in src/auth/jwt.py.

    Problem: Tokens expire immediately after login.
    Location: validate_token() function (line 42-50)

    Steps:
    1. Review the token expiration logic
    2. Fix the time calculation
    3. Run tests: pytest tests/auth/test_jwt.py
    4. Verify no regressions

    Report results by creating a spike:
    wipnote spike create "Auth Fix: [RESULTS]"
    """,
    subagent_type="general-purpose"  # Uses Haiku
)
```

---

### Sonnet - The Architect

**Best For:**
- Complex reasoning and multi-step logic
- Architecture and design decisions
- Performance optimization
- Security analysis
- Code review and quality assessment
- Trade-off analysis

**Strengths:**
- Strong reasoning about complex systems
- Good understanding of trade-offs
- Balanced speed and capability
- Excellent for architecture decisions
- Good at recognizing dependencies and impacts
- Reliable for most non-novel problems

**Limitations:**
- Can over-engineer simple tasks
- Slower than Haiku for straightforward work
- May not tackle completely novel problems as well as Opus
- Still not ideal for deep research

**Speed**: ~100-200ms execution time
**Cost**: 2-3x Haiku
**Token rate**: ~30 tokens/second

**When to Use:**
```
✅ Task: "Design a caching strategy for our API"
   Why: Requires trade-off analysis between approaches

✅ Task: "Review this code for security issues"
   Why: Needs careful analysis and cross-domain knowledge

✅ Task: "Optimize this algorithm that's causing slowdowns"
   Why: Requires reasoning about performance implications

❌ Task: "Fix the CSS alignment issue"
   Why: Too simple, use Haiku instead

❌ Task: "Design a quantum computing simulator"
   Why: Too novel, use Opus instead
```

**Example Usage:**
```python
# Good use of Sonnet—complex decision needed
Task(
    prompt="""
    Design a caching strategy for our REST API.

    Context:
    - Currently: No caching, 100 requests/sec
    - Problem: Database is maxing out at peak hours
    - Requirements: <5s cache expiration, 90% hit rate target

    Analyze:
    1. Redis vs in-memory vs hybrid approach
    2. Cache invalidation strategy
    3. Fallback for cache misses
    4. Cost vs performance implications

    Deliverable: Design document with:
    - Approach chosen and why
    - Implementation plan
    - Estimated cost and performance impact
    - Risks and mitigations

    Report findings by creating a spike:
    wipnote spike create "Caching Strategy: [DESIGN]"
    """,
    subagent_type="sonnet"  # Request Sonnet explicitly
)
```

---

### Opus - The Researcher

**Best For:**
- Novel problems and exploration
- Deep research and investigation
- Multi-step reasoning with unknowns
- Completely new feature design
- When previous attempts failed
- Complex investigation tasks

**Strengths:**
- Strongest reasoning capabilities
- Excellent handling of ambiguity and unknowns
- Good at exploration and discovery
- Handles novel problem domains well
- Can tackle multiple perspectives simultaneously
- Best for "completely new" problems

**Limitations:**
- Slowest execution (150-300ms)
- Highest cost (3-5x Haiku)
- Overkill for straightforward tasks
- May over-think simple problems

**Speed**: ~150-300ms execution time
**Cost**: Highest (3-5x Haiku baseline)
**Token rate**: ~20 tokens/second

**When to Use:**
```
✅ Task: "Research: design a real-time collaboration system from first principles"
   Why: Novel domain, needs exploration and discovery

✅ Task: "Investigate why our system has intermittent failures"
   Why: Root cause unknown, needs investigation

✅ Task: "Previous attempts to optimize this failed. Design a new approach."
   Why: Haiku and Sonnet already tried; need stronger reasoning

❌ Task: "Add a new button to the UI"
   Why: Too simple, use Haiku

❌ Task: "Optimize database query performance"
   Why: Use Sonnet for this, not Opus
```

**Example Usage:**
```python
# Good use of Opus—completely novel problem
Task(
    prompt="""
    Research: Design a real-time collaborative editing system.

    Start from first principles:
    1. What are the key technical challenges?
    2. How would you approach the architecture?
    3. What are the trade-offs?
    4. Compare to existing solutions (Google Docs, Figma, etc)

    Investigate:
    - Operational transformation vs CRDT for conflict resolution
    - Client-server vs peer-to-peer models
    - Latency and consistency requirements
    - Offline support and syncing

    Deliverable: Research report including:
    - Architecture overview
    - Key design decisions and rationale
    - Comparison to existing approaches
    - Implementation challenges
    - Recommended path forward

    Report findings by creating a spike:
    wipnote spike create "Collab System Design: [RESEARCH]"
    """,
    subagent_type="opus"  # Request Opus explicitly
)
```

---

## Decision Framework

### Step 1: Characterize the Task

Ask yourself:

**Complexity:**
- Low: Single file, <100 lines, straightforward logic
- Medium: 2-3 files, 200-500 lines, some logic complexity
- High: 4+ files, 500+ lines, architectural implications

**Novelty:**
- Familiar: Done this task type before, similar patterns exist
- Somewhat: Similar to past work but with new aspects
- Novel: Completely new problem domain or approach

**Familiarity:**
- Well-known pattern: Established solution path exists
- Some uncertainty: Know general direction, some unknowns
- High uncertainty: Many unknowns, exploration needed

**Time estimate:**
- Quick: <30 minutes
- Medium: 30-120 minutes
- Long: >120 minutes

### Step 2: Map to Model

Use this decision tree:

```
Is task well-known pattern?
├─ YES
│  └─ Will take <30 min?
│     ├─ YES → HAIKU (quick fix)
│     └─ NO → Is it architectural?
│        ├─ YES → SONNET (complex but known)
│        └─ NO → HAIKU (straightforward)
└─ NO
   └─ Have you done similar before?
      ├─ YES → SONNET (novel variation on known work)
      └─ NO → OPUS (completely new territory)
```

### Step 3: Verify with Checklist

**For Haiku:**
- [ ] Task is <30 minutes
- [ ] Clear solution path exists
- [ ] Pattern has been done before
- [ ] No major architectural decisions
- [ ] Easy to verify correctness
- [ ] Straightforward to revert if needed

**For Sonnet:**
- [ ] Task requires multi-step reasoning
- [ ] Trade-offs between approaches exist
- [ ] Previous attempts didn't work
- [ ] Architectural or design decision
- [ ] Some novelty but grounded in known work
- [ ] Code review, optimization, or security needed

**For Opus:**
- [ ] Problem domain is novel/unfamiliar
- [ ] Significant unknowns about approach
- [ ] Requires deep research or exploration
- [ ] Can't be solved with pattern matching
- [ ] Previous attempts by other models failed
- [ ] Worth the higher cost due to importance

---

## Cost Optimization

### Cost Comparison

| Scenario | Model | Speed | Cost | Score |
|----------|-------|-------|------|-------|
| Simple bug fix | Haiku | 50ms | $0.01 | ✅ Perfect |
| Simple bug fix | Sonnet | 150ms | $0.03 | ❌ Wrong choice |
| Feature implementation | Haiku | 100ms | $0.02 | ✅ Best |
| Feature implementation | Sonnet | 200ms | $0.06 | ⚠️ Overkill |
| Architecture design | Haiku | 150ms | $0.03 | ❌ Limited |
| Architecture design | Sonnet | 200ms | $0.06 | ✅ Right choice |
| Novel problem | Sonnet | 200ms | $0.06 | ⚠️ May fail |
| Novel problem | Opus | 250ms | $0.15 | ✅ Best chance |

### ROI Analysis

**Haiku:**
- Cost: Baseline (1x)
- Speed: Fastest
- Reliability: High for known tasks
- ROI: Best for delegation and quick tasks

**Sonnet:**
- Cost: 2-3x Haiku
- Speed: Medium
- Reliability: High for complex known problems
- ROI: Good for architectural decisions and complex reasoning

**Opus:**
- Cost: 3-5x Haiku
- Speed: Slowest
- Reliability: Best for completely novel problems
- ROI: Only choose when other models will likely fail

### Guidelines

**Always use Haiku for:**
- Straightforward tasks (<30 minutes)
- Established patterns
- Delegation (following instructions)
- Running tests and quality gates
- File operations and searches

**Use Sonnet for:**
- Tasks Haiku struggles with
- Design and architecture
- Performance or security analysis
- Code review
- Trade-off analysis

**Only use Opus for:**
- Tasks Sonnet can't handle
- Novel problem domains
- Deep research and exploration
- Strategic decisions with high impact

---

## Common Mistakes

### Mistake 1: Always Using Haiku

**Problem**: Using Haiku for everything because it's cheap.

**Result**: Incomplete solutions, failed attempts, wasted time debugging.

**Solution**: Match model to task. Sometimes paying 3x for Sonnet saves 10x in time/rework.

### Mistake 2: Always Using Opus

**Problem**: Using Opus for every task because "it's the best."

**Result**: Slow, expensive, overkill solutions for simple tasks.

**Solution**: Reserve Opus for truly novel problems. Use Haiku/Sonnet for 95% of work.

### Mistake 3: Haiku for Architecture

**Problem**: Using Haiku to design complex system.

**Result**: Limited analysis, missed trade-offs, incomplete design.

**Solution**: Use Sonnet for architecture decisions. Worth the extra cost.

### Mistake 4: Not Retrying with Stronger Model

**Problem**: Haiku fails at complex task, you accept the failure.

**Result**: Accept suboptimal solution instead of trying Sonnet.

**Solution**: When Haiku struggles, retry with Sonnet. Higher success rate justifies cost.

### Mistake 5: Treating All Novelty the Same

**Problem**: Using Opus for "somewhat novel" tasks.

**Result**: Overspending on tasks Sonnet can handle.

**Solution**: Sonnet handles "novel variations on known work." Only use Opus for "completely new" problems.

---

## Real-World Examples

### Example 1: Bug Fix

**Task**: "Fix JWT token expiration bug"

**Analysis**:
- Complexity: Low (single file)
- Novelty: Familiar (auth bug, done before)
- Time: <30 minutes
- Decision: **Haiku**

**Outcome**: Fixed in 15 minutes, low cost

---

### Example 2: Feature Implementation

**Task**: "Add email notifications to user alerts"

**Analysis**:
- Complexity: Medium (3-4 files involved)
- Novelty: Somewhat (email is known, but new to this codebase)
- Time: 1-2 hours
- Has tests? Yes
- Decision: **Haiku** (with Sonnet backup if it struggles)

**Outcome**: Completed in 90 minutes, moderate cost

---

### Example 3: Architecture Decision

**Task**: "Design caching strategy for our API"

**Analysis**:
- Complexity: High (architectural)
- Novelty: Somewhat (caching is known, but this design is project-specific)
- Time: 2-3 hours
- Trade-offs: Multiple approaches, each with pros/cons
- Decision: **Sonnet**

**Outcome**: Thorough design with trade-off analysis, justified cost increase

---

### Example 4: Novel Research

**Task**: "Design real-time collaborative editing system"

**Analysis**:
- Complexity: Very high
- Novelty: Novel (new domain)
- Time: 4+ hours
- Unknowns: Many (CRDT vs OT, architecture choices, etc)
- Decision: **Opus**

**Outcome**: Comprehensive research report, novel insights, worth the cost

---

### Example 5: Code Review

**Task**: "Review security of authentication module"

**Analysis**:
- Complexity: High (cross-domain knowledge needed)
- Novelty: Familiar pattern (auth is well-known)
- Time: 1-2 hours
- Requires: Careful analysis, multiple perspectives
- Decision: **Sonnet**

**Outcome**: Thorough security review, caught subtle issues Haiku missed

---

## Integration with System Prompt

Your `.claude/system-prompt.md` should include model guidance:

```markdown
## Model Guidance

Use Haiku (Default) for:
- Orchestration and delegation
- Quick fixes (<30 minutes)
- Refactoring and cleanup
- Following patterns

Use Sonnet for:
- Complex reasoning
- Architecture decisions
- Performance optimization
- Code review

Use Opus for:
- Novel problems
- Research and exploration
- When previous attempts failed
```

This keeps model selection top-of-mind throughout your work.

---

## Related Resources

- **[System Prompt Architecture](SYSTEM_PROMPT_ARCHITECTURE.md)**: How system prompts persist across sessions
- **ORCHESTRATOR_MODE_GUIDE.md**: Delegation and orchestration patterns
- **delegate.sh**: Helper script for model selection decisions

---

## Checklist: Am I Using the Right Model?

Before delegating, ask:

- [ ] Is this task well-known pattern? (If NO, consider Opus)
- [ ] Will this take <30 minutes? (If YES, use Haiku)
- [ ] Does this need architectural thinking? (If YES, use Sonnet)
- [ ] Is this completely novel? (If YES, use Opus)
- [ ] Have I solved similar before? (If NO, use Sonnet or Opus)
- [ ] What's my cost vs. quality trade-off? (Balance with budget)
- [ ] If this model fails, can I retry with stronger one? (Plan for it)

---

## Summary

| When | Use | Why |
|------|-----|-----|
| Quick, known pattern | **Haiku** | Fast, cheap, reliable |
| Complex, known domain | **Sonnet** | Good reasoning, balanced |
| Novel, exploratory | **Opus** | Best reasoning, thorough |

Choose wisely, ship fast, iterate on success.
