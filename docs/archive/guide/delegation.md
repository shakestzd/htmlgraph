# Delegating Work with Task()

## What is Delegation?

Delegation is Wipnote's orchestrator pattern for distributing work to specialized subagents. Instead of a single agent handling multiple sequential operations (which fills context with intermediate results), you spawn parallel subagents to work on focused tasks and receive summaries only.

**Key insight:** Parallel delegation is faster AND preserves orchestrator context for high-level decisions.

Example: Running 3 test suites takes the same time whether sequential or parallel, but delegation preserves your context for coordinating next steps.

## When Should You Delegate?

Use this decision framework:

**Delegate if your task:**
- ✅ Has multiple independent subtasks (can run in parallel)
- ✅ Requires many tool calls (5+ Bash, Grep, Edit, or Glob calls)
- ✅ Uses exploratory tools (Grep, Glob, Read) extensively
- ✅ Makes changes across many files (3+ file edits)
- ✅ Runs multiple tests (unit, integration, e2e)
- ✅ Explores unfamiliar codebases (needs lots of searching)

**Don't delegate if:**
- ❌ Task requires deep context from previous steps
- ❌ Work is sequential (step 2 depends on step 1 output)
- ❌ Single focused task (one file, one feature)
- ❌ Quick prototyping or experimenting

## Writing Effective Delegation Prompts

Clear delegation prompts lead to successful subagent execution. Follow these 5 guidelines:

### 1. Be Specific About the Goal
State exactly what output you need, not how to do it.

❌ Bad:
```python
Task(prompt="Explore the codebase")
```

✅ Good:
```python
Task(prompt="Find all API endpoints in src/api/ - list endpoint paths, HTTP methods, and file locations")
```

### 2. Include Success Criteria
Define what "done" looks like.

❌ Bad:
```python
Task(prompt="Run the tests")
```

✅ Good:
```python
Task(prompt="Run pytest on tests/unit/ and tests/integration/. Report: (1) total tests, (2) pass/fail count, (3) failed test names only")
```

### 3. Constrain the Scope
Tell the subagent where to focus.

❌ Bad:
```python
Task(prompt="Review the code")
```

✅ Good:
```python
Task(prompt="Review src/auth/*.py for security issues - check for SQL injection, hardcoded secrets, input validation")
```

### 4. Request Structured Output
Ask for organized, scannable results.

❌ Bad:
```python
Task(prompt="Search for database migrations")
```

✅ Good:
```python
Task(prompt="Find all database migration files in src/migrations/. Return a table with: filename, migration_type (up/down), date created")
```

### 5. Set Time/Resource Boundaries
Help the subagent know when to stop.

❌ Bad:
```python
Task(prompt="Find all TODO comments in the codebase")
```

✅ Good:
```python
Task(prompt="Find all TODO comments in src/ (exclude tests/). Report top 10 by priority. Stop after 5 minutes of searching.")
```

## Example 1: Running Tests in Parallel

**Scenario:** You need to validate a refactoring across unit, integration, and e2e tests before proceeding.

**Direct approach (sequential, fills context):**
```python
result1 = bash("uv run pytest tests/unit/ -v")      # Output: 50+ lines
result2 = bash("uv run pytest tests/integration/")   # Output: 30+ lines
result3 = bash("uv run pytest tests/e2e/")          # Output: 40+ lines
# Total context used: 120+ lines in orchestrator
```

**Delegated approach (parallel, preserves context):**
```python
Task(subagent_type="general-purpose",
     prompt="""Run pytest on tests/unit/ and report:
     1. Total test count
     2. Pass/fail count
     3. Names of failed tests (if any)
     Stop if tests take > 5 minutes""")

Task(subagent_type="general-purpose",
     prompt="""Run pytest on tests/integration/ and report:
     1. Total test count
     2. Pass/fail count
     3. Names of failed tests (if any)
     Stop if tests take > 5 minutes""")

Task(subagent_type="general-purpose",
     prompt="""Run pytest on tests/e2e/ and report:
     1. Total test count
     2. Pass/fail count
     3. Names of failed tests (if any)
     Stop if tests take > 5 minutes""")

# Orchestrator context: 3 Task() calls + summaries from subagents
# Total context used: ~20 lines in orchestrator
```

**Benefits:**
- 3x faster (parallel vs sequential)
- 6x less context used (~20 lines vs ~120 lines)
- Orchestrator can now coordinate next steps (e.g., triage failures)

## Example 2: Code Refactoring Across Many Files

**Scenario:** Rename a function across 15 files, update imports, and run tests.

**Direct approach:**
```python
# Read files to find all usages
for file in get_all_python_files():
    Grep(pattern=r"def old_function\(|old_function\(", path=file)
    # 15+ Grep calls

# Edit files
for file in files_to_update:
    Edit(file, ...)  # Replace old_function with new_function
    # 15+ Edit calls

# Run tests
bash("uv run pytest")

# Total: 30+ tool calls in orchestrator context
```

**Delegated approach:**
```python
refactoring_task = Task(
    subagent_type="general-purpose",
    prompt="""Rename function old_function to new_function across src/:
    1. Find all files that import or use old_function
    2. Update function definition in src/utils/core.py
    3. Update all import statements
    4. Update all function calls
    5. Run pytest and report pass/fail

    Report: Files changed, test results, any conflicts
    """
)

# Orchestrator context: 1 Task() call + summary
# Subagent handles all 30+ operations internally
```

**Benefits:**
- Cleaner orchestration logic
- Subagent can batch similar operations
- Errors isolated to subagent, doesn't interrupt orchestrator
- Results consolidated into single summary

## Example 3: Exploring an Unfamiliar Codebase

**Scenario:** You're new to a project and need to understand the API structure.

**Direct approach:**
```python
# Many searches to understand structure
Grep(pattern="def.*endpoint", path="src/api/")
Grep(pattern="@router\.|@app\.", path="src/")
Grep(pattern="class.*Router\|class.*API", path="src/")
Glob(pattern="src/api/**/*.py")
Read("src/api/README.md")
Read("docs/API.md")
# ... and many more

# Context fills with search results and file contents
```

**Delegated approach:**
```python
exploration_task = Task(
    subagent_type="general-purpose",
    prompt="""Analyze the API structure of this codebase and provide:
    1. List of all API endpoints (path, HTTP method, handler file)
    2. Main router/app files
    3. Authentication/middleware setup
    4. Database models and schema
    5. External dependencies

    Look in: src/api/, src/routes/, src/models/, docs/

    Output format: Organized markdown with sections
    """
)

# Orchestrator can now use the structured summary
# to make high-level decisions
```

**Benefits:**
- Subagent explores systematically
- Orchestrator gets organized knowledge base
- Faster onboarding to unfamiliar codebase
- Reusable summary for team documentation

## Handling Results

Wipnote automatically tracks parent-child session relationships when you delegate.

### Getting Results

After `Task()` completes, the subagent's output is available in the task result. Session tracking links all work automatically.

```bash
# View session tree for current session
wipnote session tree

# Find all sessions that worked on a feature
wipnote session find-feature feat-a1b2c3d4
```

### Session Linking

All work on a feature is automatically linked:

```bash
# Find all sessions that worked on a feature
wipnote session find-feature feature-auth-001

# Includes:
# - Initial orchestrator session
# - All delegated subagent sessions
# - Later continuation sessions
```

### Cost Tracking

Delegation can reduce costs by using cheaper models for subagents:

```python
# Expensive orchestrator (Opus) delegates to cheaper subagents (Haiku)
Task(
    subagent_type="general-purpose",  # Uses cheaper model
    prompt="Run tests and report failures"
)

# Orchestrator cost: ~1 Task() call (cheap)
# Subagent cost: ~Test execution (cheaper model = lower cost)
# vs Direct: Orchestrator handles all tests (expensive model)
```

## Cost Optimization

Delegation strategy affects both speed and cost:

| Approach | Speed | Context | Cost | Best For |
|----------|-------|---------|------|----------|
| Direct (sequential) | Slow | High | High | Single focused task |
| Delegated (parallel) | Fast | Low | Low | Multi-step complex work |
| Mixed | Medium | Medium | Medium | Hybrid workflows |

**Cost-optimal pattern:**
- Use expensive orchestrator (Opus) for coordination/decisions
- Use cheaper subagents (Haiku) for execution/exploration
- Delegate exploratory work (Grep, Read, Bash)
- Keep coordination in orchestrator (Task, Analysis)

## Debugging Failed Delegations

If a Task() fails, understand what went wrong:

### Problem: Subagent exceeded time limit

**What happened:** Subagent took too long and timed out.

**Solution:** Make prompt more specific with tighter boundaries.

```python
# ❌ Too vague - subagent searches everywhere
Task(prompt="Find the bug")

# ✅ More specific - bounded scope
Task(prompt="In src/auth/login.py, find where session tokens are validated. Check for expiration edge cases. Stop after 10 minutes.")
```

### Problem: Subagent returned incomplete results

**What happened:** Subagent stopped early or didn't understand requirements.

**Solution:** Add explicit success criteria.

```python
# ❌ Vague - subagent might return anything
Task(prompt="Review the code")

# ✅ Explicit criteria
Task(prompt="""Review src/api/auth.py for:
1. Password validation rules
2. Token expiration times
3. Hardcoded secrets

Return: Security issues found (or "None if secure"), specific line numbers""")
```

### Problem: Subagent explored wrong directory

**What happened:** Prompt was ambiguous about scope.

**Solution:** Be explicit with paths.

```python
# ❌ Ambiguous
Task(prompt="Find API routes")

# ✅ Explicit paths
Task(prompt="Find all HTTP endpoints in src/api/routes/ and src/handlers/. Use only these directories.")
```

### Viewing Subagent Details

See what subagent actually executed:

```bash
# Show a specific session's details
wipnote session show <session-id>

# View session tree to see parent-child relationships
wipnote session tree
```

## Best Practices

### 1. Start with Orchestrator Mode (Guidance)

Learn delegation patterns before strict enforcement:

```bash
uv run wipnote orchestrator enable --mode guidance
```

Monitor warnings to understand when delegation helps.

### 2. Delegate Early

Don't wait until you've filled context - delegate before starting heavy work:

```python
# ✅ Good - delegate before extensive work
Task(prompt="Run full test suite")

# ❌ Bad - after already using lots of tools
bash("search 1")
bash("search 2")
bash("search 3")
# ... now realizing this could have been delegated
```

### 3. Use Consistent Prompt Structure

Develop a template for reliable delegations:

```python
DELEGATION_TEMPLATE = """
Task: [One-line summary]

Scope: [Where to work - specific paths]

Requirements:
1. [Requirement 1]
2. [Requirement 2]
3. [Requirement 3]

Success criteria: [What done looks like]

Time limit: [Max time to spend]

Output format: [Structured result format]
"""

Task(prompt=DELEGATION_TEMPLATE.format(...))
```

### 4. Review Patterns in Wipnote

See examples in `.wipnote/spikes/` of real delegation patterns from Wipnote development.

### 5. Combine with Analytics

Use orchestrator analytics to find optimization opportunities:

```bash
# Find bottlenecks before dispatching parallel work
wipnote analytics bottlenecks --top 5

# Get smart recommendations for what to work on
wipnote analytics recommend --agent-count 3
```

## Common Delegation Patterns

### Pattern 1: Parallel Exploration

```python
# Explore multiple areas simultaneously
Task(prompt="Analyze src/auth/ security")
Task(prompt="Analyze src/database/ schema")
Task(prompt="Analyze src/api/ endpoints")

# Orchestrator waits for all to complete
# Results provide comprehensive system overview
```

### Pattern 2: Sequential Handoff

```python
# First task explores and prepares
task1 = Task(prompt="Explore codebase and list all TODO items")

# Second task acts on findings from first
task2 = Task(prompt=f"Based on TODOs found: {task1.result}, prioritize by impact")

# Good for discovery → action workflows
```

### Pattern 3: Divide and Conquer

```python
# Split large work into focused tasks
files = ["auth.py", "api.py", "database.py"]
for file in files:
    Task(prompt=f"Refactor {file} to use new pattern")

# Each subagent focuses on one file
# No context pollution from other files
```

### Pattern 4: Quality Gates

```python
# Delegate testing and validation
test_task = Task(prompt="Run full test suite and report failures")
lint_task = Task(prompt="Run linters (ruff, mypy) and report errors")
type_task = Task(prompt="Run type checking and report errors")

# Orchestrator waits for all gates
# Proceeds only if all pass
if all([test_task.passed, lint_task.passed, type_task.passed]):
    print("Quality gates passed - ready to commit")
```

## FAQ

**Q: When should I use Task() vs direct tool calls?**

A: Use Task() when you have multiple independent subtasks or when exploring. Use direct calls for focused, sequential work.

**Q: Does delegation add latency?**

A: No - tasks run in parallel, reducing total time despite task scheduling overhead.

**Q: Can subagents delegate further?**

A: Yes! Subagents can spawn their own Task() calls, creating hierarchical delegation trees.

**Q: What's the deepest delegation tree I should use?**

A: Usually 2-3 levels. Beyond that, orchestration complexity outweighs benefits.

**Q: How do I pass data between delegated tasks?**

A: Results from Task() are available immediately. Use results in subsequent prompts.

**Q: What if a subagent makes a mistake?**

A: The mistake is isolated to that subagent's work. Create a new Task() to fix it without affecting orchestrator context.

---

## Related Reading

- [Skills Guide](./skills.md) - Decision tree for choosing orchestrator directives or other skills
- [Session Hierarchies Guide](./session-hierarchies.md) - Understanding parent-child session relationships
- [AGENTS.md - Quick Start](../AGENTS.md#quick-start-python-sdk) - Delegation example in Quick Start
- [AGENTS.md - Orchestrator Mode](../AGENTS.md#orchestrator-mode) - Complete orchestrator reference
- [README.md - Orchestrator Architecture](../../README.md#orchestrator-architecture-flexible-multi-agent-coordination) - Multi-model spawner selection
- [Examples](../../examples/) - Real-world delegation examples
