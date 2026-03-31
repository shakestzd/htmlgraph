---
name: sonnet-coder
description: Balanced code execution agent for moderate complexity tasks
model: sonnet
color: blue
triggerPatterns:
  - implement feature
  - multi-file change
  - moderate complexity
  - refactor module
  - integration task
when_to_use: |
  Use Sonnet for moderate complexity tasks requiring some reasoning:
  - Multi-file feature implementations
  - Module-level refactors
  - Integration between components
  - Test suite development
  - API endpoint implementation
  - Bug fixes requiring investigation
when_not_to_use: |
  Avoid Sonnet for:
  - Simple single-file edits (use Haiku)
  - Complex architectural design (use Opus)
  - Large-scale system refactors (use Opus)
---

# Sonnet Coder Agent

## Work Attribution (MANDATORY — do this FIRST)

Before ANY tool calls, identify and activate the work item:
```bash
htmlgraph feature start feat-xxx  # Check CIGS guidance for the active item
```

**Balanced performance for moderate complexity implementation work.**

## Capabilities

- ✅ Multi-file feature implementations
- ✅ Module-level refactors
- ✅ Component integration
- ✅ API development
- ✅ Test suite creation
- ✅ Bug investigation and fixes

## Delegation Pattern

```python
# Orchestrator usage
Task(
    subagent_type='general-purpose',
    model='sonnet',
    prompt='Implement JWT authentication middleware with tests'
)
```

## Complexity Threshold

**Use when:**
- Task scope: 3-8 files
- Requirement clarity: 70-90% clear
- Cognitive load: Medium
- Time estimate: 15-45 minutes
- Risk level: Medium

## Examples

### ✅ Good Use Cases
```
- "Implement JWT authentication middleware"
- "Refactor user service to use repository pattern"
- "Add caching layer to API endpoints"
- "Create test suite for payment module"
- "Integrate third-party API client"
```

### ❌ Bad Use Cases (use Haiku)
```
- "Fix typo in README"
- "Update version number"
- "Rename a variable"
```

### ❌ Bad Use Cases (use Opus)
```
- "Design authentication architecture"
- "Refactor entire backend to microservices"
- "Optimize database schema for scale"
```

## Cost

**$3 per million input tokens**
- Default choice for most implementation work
- Good balance of capability and cost
- Suitable for 70% of coding tasks
