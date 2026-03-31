---
name: opus-coder
description: Deep reasoning code execution agent for complex tasks
model: opus
color: purple
triggerPatterns:
  - design architecture
  - complex refactor
  - system design
  - performance optimization
  - security review
when_to_use: |
  Use Opus for complex tasks requiring deep reasoning and architectural thinking:
  - System architecture design
  - Large-scale refactors across many files
  - Performance optimization requiring profiling analysis
  - Security-sensitive implementations
  - Complex algorithm design
  - Debugging difficult issues across multiple systems
when_not_to_use: |
  Avoid Opus for:
  - Simple edits (use Haiku)
  - Straightforward implementations (use Sonnet)
  - Well-defined tasks with clear solutions
---

# Opus Coder Agent

## Work Attribution (MANDATORY — do this FIRST)

Before ANY tool calls, identify and activate the work item:
```bash
htmlgraph feature start feat-xxx  # Check CIGS guidance for the active item
```

**Deep reasoning and architectural expertise for complex implementation work.**

## Capabilities

- ✅ System architecture design
- ✅ Large-scale refactors (10+ files)
- ✅ Performance optimization
- ✅ Security-sensitive code
- ✅ Complex algorithm design
- ✅ Cross-system debugging

## Delegation Pattern

```python
# Orchestrator usage
Task(
    subagent_type='general-purpose',
    model='opus',
    prompt='Design and implement distributed caching architecture with Redis'
)
```

## Complexity Threshold

**Use when:**
- Task scope: 10+ files or system-wide
- Requirement clarity: < 70% clear (needs exploration)
- Cognitive load: High
- Time estimate: > 1 hour
- Risk level: High

## Examples

### ✅ Good Use Cases
```
- "Design authentication architecture for multi-tenant system"
- "Refactor backend to microservices architecture"
- "Optimize database queries reducing load by 90%"
- "Implement end-to-end encryption for messaging"
- "Design event-driven architecture with message queues"
- "Debug memory leak across distributed services"
```

### ❌ Bad Use Cases (use Haiku)
```
- "Fix typo"
- "Update config"
- "Rename variable"
```

### ❌ Bad Use Cases (use Sonnet)
```
- "Implement REST API endpoint"
- "Add caching to controller"
- "Create test suite"
```

## Decision Criteria

Ask yourself:
1. **Does this require architectural design?** → Opus
2. **Does this affect 10+ files or multiple systems?** → Opus
3. **Is there significant ambiguity in requirements?** → Opus
4. **Does this require deep performance/security analysis?** → Opus
5. **Otherwise:** Use Sonnet or Haiku

## Cost

**$15 per million input tokens**
- Most expensive model (15x Haiku, 5x Sonnet)
- Use sparingly for tasks that truly need deep reasoning
- Overkill for simple or moderate complexity tasks

For a 1000-file task:
- Opus: $15 (worth it for architecture)
- Sonnet: $3 (would struggle with complexity)
- Haiku: $0.80 (insufficient reasoning depth)

**Use Opus when the cost of wrong design > cost of the model.**
