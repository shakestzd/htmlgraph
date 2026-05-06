# Before/After Comparison - Orchestration Rules Update

## File Structure Changes

### BEFORE
```markdown
# Orchestration Rules - Delegation Over Direct Execution

## Core Philosophy
(generic delegation discussion)

## Operations You MUST Delegate
(vague "delegate to subagent" without model specifics)

### 1. Git Operations - ALWAYS DELEGATE
- Task() examples only
- No cost information
```

### AFTER
```markdown
# Orchestration Rules - Cost-First Multi-AI Delegation

## Cost-First Delegation Priority (NEW!)
- Decision tree with 6 priorities
- HeadlessSpawner examples
- "Why Not Task()?" section
- Cost comparisons

## Model Selection Reference (NEW!)
- Links to multi-ai-orchestration skill
- Links to implementation files

## Core Philosophy
(same)

## Operations You MUST Delegate
- Specific model for each operation type
- Cost comparisons
- ✅/❌ visual markers
```

## Language Changes

### Git Operations

**BEFORE**:
```markdown
### 1. Git Operations - ALWAYS DELEGATE
- ❌ NEVER run git commands directly
- ✅ ALWAYS delegate to subagent with error handling

**Delegation pattern:**
Task(
    prompt="commit changes...",
    subagent_type="general-purpose"
)
```

**AFTER**:
```markdown
### 1. Git Operations - ALWAYS use Copilot

**REQUIRED: MUST use spawn_copilot() for all git/GitHub operations.**

- ❌ NEVER run git commands directly
- ❌ NEVER use Task() for git operations (expensive, not specialized)
- ✅ ALWAYS use spawn_copilot() (cheaper, GitHub-specialized)

**Cost comparison:**
Task() for git: $5-10 per workflow
Copilot for git: $2-3 per workflow (60% savings + better results)

**IMPERATIVE Delegation pattern:**
from wipnote.orchestration import HeadlessSpawner
spawner = HeadlessSpawner()

# ✅ CORRECT - Use Copilot for git
result = spawner.spawn_copilot(
    prompt="commit changes...",
    allow_all_tools=True
)

# ❌ INCORRECT - Don't use Task() for git
Task(prompt="commit changes...", subagent_type="general-purpose")
```

### Research & Exploration

**BEFORE**:
```markdown
### 3. Research & Exploration - ALWAYS DELEGATE
- ❌ Large codebase searches (multiple Grep/Glob calls)
- ❌ Understanding unfamiliar systems
- ❌ Documentation research
- ✅ Single file quick lookup (OK to do directly)
```

**AFTER**:
```markdown
### 3. Research & Exploration - ALWAYS use Gemini

**REQUIRED: MUST use spawn_gemini() for exploration (FREE!).**

- ❌ NEVER use Task() for exploration (expensive)
- ❌ Large codebase searches → MUST use Gemini (FREE)
- ❌ Understanding unfamiliar systems → MUST use Gemini (FREE)
- ❌ Documentation research → MUST use Gemini (FREE)
- ✅ Single file quick lookup (OK to do directly)

**IMPERATIVE Delegation pattern:**
from wipnote.orchestration import HeadlessSpawner
spawner = HeadlessSpawner()

# ✅ CORRECT - Use Gemini for exploration (FREE!)
result = spawner.spawn_gemini(
    prompt="Analyze all authentication patterns in codebase...",
    model="gemini-2.0-flash-exp"
)

# ❌ INCORRECT - Don't use Task() for exploration (costs $15-25)
Task(prompt="analyze codebase...", subagent_type="explorer")
```

### Code Implementation

**BEFORE**:
```markdown
### 2. Code Changes - DELEGATE Unless Trivial
- ❌ Multi-file edits
- ❌ Implementation requiring research
- ❌ Changes with testing requirements
- ✅ Single-line typo fixes (OK to do directly)
```

**AFTER**:
```markdown
### 2. Code Changes - ALWAYS use Codex

**REQUIRED: MUST use spawn_codex() for code implementation (unless trivial).**

- ❌ NEVER use Task() for code generation (expensive, not specialized)
- ❌ Multi-file edits → MUST use Codex
- ❌ Implementation requiring research → MUST use Codex
- ❌ Changes with testing requirements → MUST use Codex
- ✅ Single-line typo fixes (OK to do directly)

**IMPERATIVE Delegation pattern:**
from wipnote.orchestration import HeadlessSpawner
spawner = HeadlessSpawner()

# ✅ CORRECT - Use Codex for code
result = spawner.spawn_codex(
    prompt="Implement authentication middleware with JWT...",
    model="gpt-4"
)

# ❌ INCORRECT - Don't use Task() for code
Task(prompt="implement feature...", subagent_type="general-purpose")
```

## Decision Framework Changes

### BEFORE (4 questions)
```markdown
Ask yourself:
1. Will this likely be one tool call?
   - If uncertain → DELEGATE
   - If certain → MAY do directly

2. Does this require error handling?
   - If yes → DELEGATE

3. Could this cascade into multiple operations?
   - If yes → DELEGATE

4. Is this strategic (decisions) or tactical (execution)?
   - Strategic → Do directly
   - Tactical → DELEGATE
```

### AFTER (5 questions IN ORDER - cost-first)
```markdown
Ask yourself IN ORDER:
1. Is this exploration/research?
   - If yes → MUST use spawn_gemini() (FREE)

2. Is this code implementation?
   - If yes → MUST use spawn_codex() (cheaper, specialized)

3. Is this git/GitHub operation?
   - If yes → MUST use spawn_copilot() (cheaper, specialized)

4. Is this strategic coordination?
   - If yes → MAY use Task() with Opus/Sonnet

5. Is this a trivial single tool call?
   - If yes AND certain → MAY do directly
   - If uncertain → MUST delegate to appropriate model
```

## Orchestrator Reflection Changes

### BEFORE
```markdown
ORCHESTRATOR REFLECTION: You executed code directly.

Ask yourself:
- Could this have been delegated to a subagent?
- Would parallel Task() calls have been faster?
- Is a work item tracking this effort?
- What if this operation fails - how many retries will consume context?
```

### AFTER
```markdown
ORCHESTRATOR REFLECTION: You executed code directly.

Ask yourself:
- Could this have been delegated to Gemini (FREE)?
- Could this have been delegated to Codex (70% cheaper)?
- Could this have been delegated to Copilot (60% cheaper)?
- What if this operation fails - how many retries will consume context?
- Would parallel HeadlessSpawner calls have been faster?
```

## "Why Strict Delegation Matters" Changes

### BEFORE (4 reasons)
```markdown
1. Context Preservation
2. Parallel Efficiency
3. Error Isolation
4. Cognitive Clarity
```

### AFTER (5 reasons, NEW #1)
```markdown
1. Cost Optimization (NEW - MOST IMPORTANT)
   - Gemini is FREE for exploration (vs $15-25 with Task)
   - Codex is 70% cheaper for code (vs Task)
   - Copilot is 60% cheaper for git (vs Task)
   - Choosing the right model saves 60-100% per operation

2. Context Preservation
3. Parallel Efficiency
4. Error Isolation
5. Cognitive Clarity
```

## Real-World Impact Example

### Scenario: Implement Authentication System

**BEFORE (using Task() for everything)**:
```python
# Step 1: Research patterns
Task(prompt="Analyze authentication patterns...", subagent_type="explorer")
# Cost: $15-25

# Step 2: Implement code
Task(prompt="Implement OAuth flow...", subagent_type="general-purpose")
# Cost: $10

# Step 3: Write tests
Task(prompt="Add test coverage...", subagent_type="general-purpose")
# Cost: $10

# Step 4: Commit changes
Task(prompt="Commit and push...", subagent_type="general-purpose")
# Cost: $5

# TOTAL: $40-50
```

**AFTER (using cost-first delegation)**:
```python
from wipnote.orchestration import HeadlessSpawner
spawner = HeadlessSpawner()

# Step 1: Research patterns (FREE!)
result = spawner.spawn_gemini(
    prompt="Analyze authentication patterns...",
    model="gemini-2.0-flash-exp"
)
# Cost: FREE

# Step 2: Implement code (70% cheaper)
result = spawner.spawn_codex(
    prompt="Implement OAuth flow...",
    model="gpt-4"
)
# Cost: $3

# Step 3: Write tests (70% cheaper)
result = spawner.spawn_codex(
    prompt="Add test coverage...",
    model="gpt-4"
)
# Cost: $3

# Step 4: Commit changes (60% cheaper)
result = spawner.spawn_copilot(
    prompt="Commit and push...",
    allow_all_tools=True
)
# Cost: $2

# TOTAL: $8 (80% savings!)
```

## Visual Markers Added

All sections now use visual markers for clarity:
- ✅ = CORRECT / DO THIS
- ❌ = INCORRECT / DON'T DO THIS

Example:
```markdown
- ❌ NEVER use Task() for exploration (expensive)
- ✅ ALWAYS use spawn_gemini() (FREE)
```

## Summary

The update transforms orchestration.md from:
- **Permissive** → **Imperative**
- **Generic** → **Specific**
- **Task()-first** → **Cost-first with HeadlessSpawner**
- **No cost awareness** → **Cost comparisons everywhere**
- **Vague delegation** → **Precise model selection**

**Result**: Agents will now default to cost-optimized multi-AI delegation instead of expensive Task() calls.
