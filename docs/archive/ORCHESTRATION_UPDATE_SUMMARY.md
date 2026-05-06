# Orchestration Rules Update - Multi-AI Delegation Made Imperative

## Summary

Updated `/Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/rules/orchestration.md` to make multi-AI delegation IMPERATIVE with cost-first priority.

## Key Changes

### 1. Added Cost-First Delegation Priority (NEW SECTION at top)

**Before**: File started with generic delegation philosophy
**After**: File now starts with imperative decision tree:

```
1. Exploration/Research → spawn_gemini() (FREE)
2. Code Implementation → spawn_codex() ($, specialized)
3. Git/GitHub Operations → spawn_copilot() ($, GitHub integration)
4. Deep Reasoning/Architecture → Claude Opus ($$$$, via Task)
5. Multi-Agent Coordination → Claude Sonnet ($$$, via Task)
6. FALLBACK ONLY → Task() with Haiku ($$, when above unavailable)
```

### 2. Language Changes: Permissive → Imperative

**Before**:
- "should use"
- "can delegate"
- "prefer"
- "consider using"

**After**:
- "MUST use"
- "ALWAYS use"
- "REQUIRED"
- "NEVER use Task() for..."

### 3. Added HeadlessSpawner Examples Throughout

**Before**: Only showed Task() delegation patterns
**After**: Shows HeadlessSpawner as PRIMARY, Task() as fallback

**Example (Git Operations)**:

**BEFORE**:
```python
Task(
    prompt="commit changes...",
    subagent_type="general-purpose"
)
```

**AFTER**:
```python
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

### 4. Updated All 7 Operation Categories

#### Git Operations (Section 1)
- **Before**: "ALWAYS DELEGATE" (vague)
- **After**: "ALWAYS use Copilot" (specific)
- Added cost comparison: Task($5-10) vs Copilot($2-3) = 60% savings

#### Code Changes (Section 2)
- **Before**: "DELEGATE Unless Trivial"
- **After**: "ALWAYS use Codex" with imperative pattern
- Added cost comparison: Task($10) vs Codex($3) = 70% savings

#### Research & Exploration (Section 3)
- **Before**: "ALWAYS DELEGATE" (vague)
- **After**: "ALWAYS use Gemini (FREE!)"
- Added cost comparison: Task($15-25) vs Gemini(FREE) = 100% savings

#### Testing & Validation (Section 4)
- **Before**: Generic delegation
- **After**: Prefer Codex for specialized testing

#### Build & Deployment (Section 5)
- **Before**: Generic delegation
- **After**: MUST delegate (strengthened language)

#### File Operations (Section 6)
- **Before**: "DELEGATE Complex Operations"
- **After**: "MUST delegate" (stronger language)

#### Analysis & Computation (Section 7)
- **Before**: "DELEGATE Heavy Work"
- **After**: "ALWAYS use Gemini (FREE!)"

### 5. Updated "Why Strict Delegation Matters"

**Added NEW #1 Priority**: Cost Optimization
- Gemini is FREE for exploration (vs $15-25 with Task)
- Codex is 70% cheaper for code (vs Task)
- Copilot is 60% cheaper for git (vs Task)
- Choosing the right model saves 60-100% per operation

**Reordered priorities**:
1. Cost Optimization (NEW)
2. Context Preservation
3. Parallel Efficiency
4. Error Isolation
5. Cognitive Clarity

### 6. Updated Decision Framework

**Before** (4 questions):
1. Will this likely be one tool call?
2. Does this require error handling?
3. Could this cascade?
4. Strategic vs tactical?

**After** (5 questions IN ORDER):
1. Is this exploration/research? → MUST use spawn_gemini() (FREE)
2. Is this code implementation? → MUST use spawn_codex() (cheaper)
3. Is this git/GitHub operation? → MUST use spawn_copilot() (cheaper)
4. Is this strategic coordination? → MAY use Task() with Opus/Sonnet
5. Is this trivial single tool call? → MAY do directly IF certain

### 7. Updated Orchestrator Reflection System

**Before**:
```
Ask yourself:
- Could this have been delegated to a subagent?
- Would parallel Task() calls have been faster?
```

**After**:
```
Ask yourself:
- Could this have been delegated to Gemini (FREE)?
- Could this have been delegated to Codex (70% cheaper)?
- Could this have been delegated to Copilot (60% cheaper)?
- Would parallel HeadlessSpawner calls have been faster?
```

### 8. Updated SDK Integration Example

**Before**: Used Task() for all delegation
**After**: Uses HeadlessSpawner with appropriate models:

```python
# Research (FREE!)
research_result = spawner.spawn_gemini(...)

# Code implementation (70% cheaper)
code_result = spawner.spawn_codex(...)

# Git operations (60% cheaper)
git_result = spawner.spawn_copilot(...)
```

### 9. Updated Model Selection Reference

Added links to:
- `/multi-ai-orchestration` skill (comprehensive guide)
- `src/python/wipnote/orchestration/model_selection.py` (decision matrix)
- `src/python/wipnote/orchestration/headless_spawner.py` (implementation)

## Behavior Change Example

### Scenario: Implement Authentication Feature

**BEFORE (Old Behavior)**:
```python
# Agent would default to Task() for everything
Task(prompt="Analyze auth patterns...", subagent_type="explorer")  # $15-25
Task(prompt="Implement OAuth...", subagent_type="general-purpose")  # $10
Task(prompt="Commit changes...", subagent_type="general-purpose")   # $5
# Total: $30-40
```

**AFTER (New Imperative Behavior)**:
```python
from wipnote.orchestration import HeadlessSpawner
spawner = HeadlessSpawner()

# Exploration (FREE!)
result = spawner.spawn_gemini(
    prompt="Analyze auth patterns...",
    model="gemini-2.0-flash-exp"
)  # FREE

# Code implementation (70% cheaper)
result = spawner.spawn_codex(
    prompt="Implement OAuth...",
    model="gpt-4"
)  # $3

# Git operations (60% cheaper)
result = spawner.spawn_copilot(
    prompt="Commit changes...",
    allow_all_tools=True
)  # $2

# Total: $5 (83% savings!)
```

## Cost Impact

For a typical development workflow with:
- 10 exploration tasks
- 5 code implementations
- 10 git operations

**Before**: 10×$20 + 5×$10 + 10×$5 = $300
**After**: 10×$0 + 5×$3 + 10×$2 = $35

**Total savings: $265 (88% reduction)**

## Files Modified

1. `/Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/rules/orchestration.md`
   - This is the CANONICAL version (plugin source of truth)
   - No .claude/rules version exists (intentionally - plugin is source)

## Quality Checks

- ✅ Language changed to imperative ("MUST", "ALWAYS", "NEVER")
- ✅ Cost comparisons added to all sections
- ✅ HeadlessSpawner examples added before Task() examples
- ✅ Decision tree prioritizes cost-first
- ✅ Visual markers (✅/❌) for clarity
- ✅ References to multi-ai-orchestration skill added
- ✅ Markdown formatting preserved

## Next Steps

1. **Sync to .claude**: Optionally copy to `.claude/rules/orchestration.md` if local override needed
2. **Update hooks**: Modify orchestrator hooks to reference HeadlessSpawner
3. **Test**: Verify agents now use cost-first delegation by default
4. **Document**: Add this pattern to multi-ai-orchestration skill
5. **Deploy**: Include in next package release

## Verification

To verify the changes are working:

1. Enable orchestrator mode: `orchestrator mode strict`
2. Ask agent to explore codebase
3. Check that it uses `spawn_gemini()` instead of Task()
4. Ask agent to implement code
5. Check that it uses `spawn_codex()` instead of Task()
6. Ask agent to commit changes
7. Check that it uses `spawn_copilot()` instead of Task()

## References

- Original file: `/Users/shakes/DevProjects/htmlgraph/packages/claude-plugin/rules/orchestration.md`
- Multi-AI skill: `packages/claude-plugin/skills/multi-ai-orchestration-skill/SKILL.md`
- HeadlessSpawner: `src/python/wipnote/orchestration/headless_spawner.py`
- Model selection: `src/python/wipnote/orchestration/model_selection.py`
