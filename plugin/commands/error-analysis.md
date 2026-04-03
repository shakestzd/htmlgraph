---
name: error-analysis
description: Systematically capture, analyze, and track errors with HtmlGraph spike-based investigation workflow
args:
  - name: error_context
    description: Brief description of the error or error message
    required: false
---

# /htmlgraph:error-analysis

Systematically capture, analyze, and track errors with HtmlGraph spike-based investigation workflow.

## Usage

```
/htmlgraph:error-analysis [error_context]
```

## Parameters

- `error_context` (optional): Brief description of the error or error message


## Examples

```bash
/htmlgraph:error-analysis "PreToolUse hook failing with 'No such file'"
```
Capture and analyze a hook error with HtmlGraph tracking

```bash
/htmlgraph:error-analysis
```
Interactive error capture workflow


## Instructions for Claude

**CRITICAL: This command implements systematic error investigation using HtmlGraph spikes.**

This command follows the research-first debugging methodology from `.claude/rules/debugging.md`. It ensures errors are properly documented, investigated systematically, and tracked in HtmlGraph.

### Implementation:

```
**DO THIS:**

1. **Capture error details:**
   - If error_context provided, use it as starting point
   - Otherwise, use AskUserQuestion to gather:
     - Exact error message
     - When did it occur (what operation)
     - What changed recently (code, config, plugins)
     - Can it be reproduced consistently?
     - Expected vs actual behavior

2. **Categorize the error:**
   Identify error type:
   - **Hook Error** - PreToolUse, PostToolUse, SessionStart failures
   - **API Error** - Network, authentication, rate limits
   - **Build Error** - Compilation, linting, type checking
   - **Runtime Error** - Exceptions, crashes, unexpected behavior
   - **Configuration Error** - Plugin, settings, environment issues
   - **Integration Error** - External services, databases, APIs

3. **Gather relevant context:**
   Collect diagnostic information based on error type:

   For Hook Errors:
   ```bash
   /hooks                    # List all active hooks
   /hooks PreToolUse        # Show specific hook type
   claude --debug <command> # Verbose output
   ```

   For Build Errors:
   ```bash
   go build ./...           # Build errors
   go vet ./...             # Vet warnings
   go test ./...            # Test failures
   ```

   For Runtime Errors:
   - Recent file changes (git diff)
   - Environment variables
   - Dependency versions
   - Stack traces

4. **Create HtmlGraph spike for investigation:**
   ```bash
   # Check for similar past spikes
   htmlgraph spike list

   # Create spike for this investigation
   htmlgraph spike create "Error Investigation: {error_category} - {brief_description}"
   ```

   Review the spike list output to identify similar past issues by title. Note any relevant past spikes and their resolutions before proceeding.

5. **Provide systematic investigation prompts:**
   Based on error category, guide investigation:

   **Hook Errors:**
   - [ ] Research hook documentation (use /htmlgraph:research or claude-code-guide)
   - [ ] Check for duplicate hooks across sources
   - [ ] Verify hook file paths and permissions
   - [ ] Test with --debug flag for verbose output
   - [ ] Check plugin versions and compatibility

   **API Errors:**
   - [ ] Verify authentication credentials
   - [ ] Check rate limits and quotas
   - [ ] Test with curl/requests directly
   - [ ] Review API documentation for changes
   - [ ] Check network connectivity

   **Build Errors:**
   - [ ] Build check: `go build ./...`
   - [ ] Run vet: `go vet ./...`
   - [ ] Run tests: `go test ./...`
   - [ ] Review recent code changes
   - [ ] Check dependency versions

   **Runtime Errors:**
   - [ ] Reproduce error with minimal test case
   - [ ] Check stack trace for root cause
   - [ ] Verify input data and edge cases
   - [ ] Review recent code changes
   - [ ] Test in isolation (unit test)

   **Configuration Errors:**
   - [ ] Validate configuration files (JSON, YAML)
   - [ ] Check environment variables
   - [ ] Verify file paths and permissions
   - [ ] Review plugin settings
   - [ ] Compare with working configuration

6. **Offer debugging agent integration:**
   Based on investigation needs:

   ```
   ## Debugging Resources Available

   **DELEGATION**: Use `Task(subagent_type="htmlgraph:researcher")` for researching documentation and prior art.
   Use `Task(subagent_type="htmlgraph:debugger")` for systematic error investigation.
   Use `Task(subagent_type="htmlgraph:test-runner")` to validate fixes.

   ### Researcher Agent (htmlgraph:researcher)
   Use when you need to understand unfamiliar concepts or APIs:
   - Research Claude Code hook behavior
   - Look up library documentation
   - Find best practices for error handling

   ### Debugger Agent (htmlgraph:debugger)
   Use for systematic error analysis:
   - Reproduce errors consistently
   - Isolate root causes
   - Test hypotheses systematically

   ### Test Runner Agent (htmlgraph:test-runner)
   Use to validate fixes:
   - Run quality gates (lint, type, test)
   - Verify error is resolved
   - Prevent regression
   ```

7. **Document investigation workflow:**

   After creating the spike, record progress as you investigate:
   ```bash
   htmlgraph spike show <spike-id>
   ```

   Follow these investigation steps:
   - Gather diagnostic information
   - Research root cause (if unfamiliar)
   - Form hypothesis about cause
   - Test hypothesis systematically
   - Implement minimal fix
   - Validate fix resolves error
   - Document learning

8. **Output structured investigation plan:**
   Show spike details and next steps
```

### Output Format:

```
## Error Investigation Spike Created

**Spike ID:** {spike.id}
**Title:** {spike.title}
**Category:** {error_category}
**Timebox:** 2 hours

### Error Summary
{formatted_error_details}

### Investigation Checklist
{category_specific_checklist}

### Debugging Tools Available
- `/hooks` - List active hooks
- `claude --debug <command>` - Verbose output
- `/doctor` - System diagnostics
- Quality gates: `go build ./... && go vet ./... && go test ./...`

### Next Steps
1. Complete investigation checklist items
2. Document findings in spike: `/htmlgraph:spike {spike.id}`
3. Delegate to specialized agents:
   - `Task(subagent_type="htmlgraph:researcher")` for unfamiliar concepts
   - `Task(subagent_type="htmlgraph:debugger")` for systematic debugging
   - `Task(subagent_type="htmlgraph:test-runner")` to validate fixes

### Start Investigation
Use these commands to begin:
```bash
# View spike in dashboard
htmlgraph serve
# Open: http://localhost:8080

# Research if needed (unfamiliar error)
/htmlgraph:research "{error topic}"

# Document findings as you investigate
# Findings are auto-tracked in the spike
```

**Remember: Research first, implement second. Don't make trial-and-error attempts.**
```

### Error Category Mappings

Use these patterns to categorize errors:

```bash
# Error category keywords for classification:

# hook: PreToolUse PostToolUse SessionStart SessionEnd hook plugin marketplace
# api: API authentication rate-limit network timeout HTTP request-failed connection
# build: go-build go-vet go-test lint build-error vet-warning test-failed compilation syntax-error
# runtime: panic error crash failed unexpected assertion goroutine nil-pointer
# config: configuration settings environment missing invalid not-found .env credentials
```

### Integration with Debugging Workflow

This command implements the systematic debugging workflow:

```
1. /htmlgraph:error-analysis "error message"  → Capture & categorize
2. [Complete investigation checklist]          → Gather evidence
3. /htmlgraph:research "topic" (if needed)    → Research unfamiliar concepts
4. [Test hypothesis systematically]            → Debug root cause
5. [Implement minimal fix]                     → Fix the issue
6. [Run quality gates]                         → Validate fix
7. [Document learning in spike]                → Capture knowledge
```

### Quality Checklist

Before marking investigation complete, verify:

- [ ] **Root cause identified** - Not just symptoms
- [ ] **Fix tested** - Error no longer occurs
- [ ] **Learning documented** - Added to spike findings
- [ ] **Prevention considered** - How to avoid in future
- [ ] **Quality gates pass** - All tests/lints pass

### When to Use This Command

**ALWAYS use for:**
- ✅ Unfamiliar errors (first time seeing this error)
- ✅ Recurring errors (happens multiple times)
- ✅ Critical errors (blocks work or breaks functionality)
- ✅ Complex errors (multiple potential causes)
- ✅ Learning opportunities (want to understand deeply)

**SKIP for:**
- ❌ Known simple fixes (typos, obvious mistakes)
- ❌ Already understood errors (seen and fixed before)
- ❌ Trivial warnings (can ignore safely)

### Example Investigation Flow

**Scenario: Hook error "No such file"**

```bash
# Step 1: Capture error
/htmlgraph:error-analysis "PreToolUse hook failing with 'No such file'"

# Creates spike with:
# - Error category: hook
# - Investigation checklist
# - Debugging resources

# Step 2: Research (if unfamiliar with hooks)
/htmlgraph:research "Claude Code hook loading and file paths"

# Finds:
# - Hooks load from .claude/hooks/ and plugin directories
# - File paths must be absolute or relative to hook location
# - Common issue: incorrect ${CLAUDE_PLUGIN_ROOT} usage

# Step 3: Debug systematically
/hooks PreToolUse  # List all PreToolUse hooks
# Shows duplicate hooks from plugin and .claude/settings.json

# Step 4: Fix
# Remove duplicate hook definition

# Step 5: Validate
claude --debug <command>  # Test with verbose output
# Error resolved!

# Step 6: Document
# Add finding to spike: "Hook duplication caused conflict"
# Mark investigation complete
```

### Research-First Principle

**CRITICAL: Always research before implementing fixes.**

❌ **Wrong approach:**
1. Try fix A → Still broken
2. Try fix B → Still broken
3. Try fix C → Still broken
4. Finally research documentation
5. Find actual solution

✅ **Correct approach:**
1. Use `/htmlgraph:error-analysis` to capture error
2. Use `/htmlgraph:research` to understand root cause
3. Implement fix based on understanding
4. Validate fix works
5. Document learning

**Impact:**
- Fewer failed attempts (saves context/time)
- Deeper understanding (prevents recurrence)
- Better documentation (captures learning)
- More efficient debugging (systematic vs reactive)
