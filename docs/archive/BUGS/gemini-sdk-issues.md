# Gemini SDK Issues - December 2025

## Issue 1: Cross-Project Pollution (CRITICAL)

### Symptom
Gemini was working in the **contextune project** (`/Users/shakes/DevProjects/contextune`) but saw a feature `feat-4cec1d2e` titled "Cross-Session Continuity Enhancement" which is an **wipnote project** feature (about modifying `.wipnote/` schema and MCP server).

### Evidence
```
Current working directory: /Users/shakes/DevProjects/contextune

# Gemini ran wipnote status and saw:
feat-4cec1d2e: Cross-Session Continuity Enhancement
- Steps about modifying .wipnote/ schema
- Steps about updating MCP server
```

### Why This is Critical
- Projects should be isolated - contextune features should not see wipnote features
- This breaks the fundamental assumption that `.wipnote/` is project-local
- Agents will get confused about which project they're working on
- Attribution will be wrong (contextune work attributed to wipnote features)

### Possible Root Causes

1. **SDK initialization bug** - SDK(agent='gemini') might be looking in wrong directory
2. **Missing .wipnote/ in contextune** - If contextune doesn't have `.wipnote/`, SDK might fall back to parent or home directory
3. **Shared state** - Some global config or cache leaking between projects
4. **Virtual env confusion** - Gemini output showed:
   ```
   warning: `VIRTUAL_ENV=/Users/shakes/DevProjects/slashsense/.venv` does not match the project environment path `.venv`
   ```

### Investigation Needed

1. Check if `/Users/shakes/DevProjects/contextune/.wipnote/` exists
2. Check SDK.__init__() - does it properly detect current project directory?
3. Check if there's any global state or caching in SDK
4. Test: Create fresh project, run `wipnote status`, verify it doesn't see other projects' features

### Expected Behavior
- Each project has its own `.wipnote/` directory
- SDK should auto-detect the current project's `.wipnote/`
- If no `.wipnote/` exists, should error or offer to run `wipnote init`
- NEVER show features from other projects

---

## Issue 2: Missing Delete Functionality

### Symptom
Gemini wanted to delete `feat-4cec1d2e` (the misplaced feature) but found:
- ❌ No `wipnote feature delete` CLI command
- ❌ No obvious SDK method to delete features
- ❌ Manual `rm` is forbidden (bypasses index, event log, validation)

### Evidence

Gemini's investigation:
```bash
# Checked help
uv run wipnote feature --help
# Found: create, start, complete, primary, claim, release, auto-release, list, step-complete
# Missing: delete, remove, archive

# Considered Python SDK
sdk.features.delete('feat-4cec1d2e')  # Does this exist?
# Couldn't find documentation, gave up
```

### Current Workaround
None. Agents must:
1. Ignore the bad feature
2. Manually `rm` the HTML file (breaks rules, corrupts index)
3. Ask user to delete it manually

### Why Delete is Needed

**Common use cases:**
- Remove test/experimental features that were created by mistake
- Clean up abandoned features
- Remove duplicates
- Archive old features (move to different status)

**Why `rm` is not acceptable:**
- Bypasses Pydantic validation
- Breaks SQLite index sync
- Skips event logging
- Doesn't update related features (e.g., blocked_by links)
- Leaves orphaned references

### Required Implementation

1. **SDK Method:**
   ```python
   # BaseCollection.delete(id: str, confirm: bool = True)
   sdk.features.delete("feat-001", confirm=True)
   # Should:
   # - Remove HTML file
   # - Update index
   # - Log deletion event
   # - Ask for confirmation (destructive operation)
   # - Update related features (remove from blocked_by, etc.)
   ```

2. **CLI Command:**
   ```bash
   wipnote feature delete feat-001
   # Should prompt: "Delete feat-001? This cannot be undone. [y/N]"
   ```

3. **Safety Features:**
   - Require confirmation for delete (destructive)
   - Option to archive instead of delete (set status="archived")
   - Check for dependents (warn if other features link to this one)
   - Log deletion with reason

---

## Issue 3: SDK Method Discoverability

### Symptom
Gemini couldn't figure out if SDK has a `delete()` method because:
- No clear API reference at runtime
- Help text shows CLI commands but not SDK methods
- No examples of "how to explore SDK methods"

### Evidence

Gemini's thought process:
```
"I will check if delete is supported."
"Let's try to find a delete method or just set its status to 'abandoned' or something."
"I will assume for now I should just create the new feature the user wants."
```

Gemini eventually gave up on finding delete and just ignored the issue.

### Why This is a Problem

**AI agents need to discover capabilities at runtime:**
- Can't memorize all methods (context limits)
- Need to explore the API as they work
- Should be able to introspect: "What methods does sdk.features have?"

**Current discoverability is poor:**
- CLI help shows commands but not SDK methods
- AGENTS.md has examples but no complete API reference
- No runtime exploration guide

### Required Improvements

1. **Add to AGENTS.md:**
   ```markdown
   ## SDK Method Discovery

   ### Runtime Introspection
   ```python
   from wipnote import SDK
   sdk = SDK(agent="claude")

   # List all methods on a collection
   print(dir(sdk.features))
   # ['all', 'assign', 'batch_update', 'create', 'delete', 'edit', 'get', 'mark_done', 'where', ...]

   # Get method signature
   import inspect
   print(inspect.signature(sdk.features.create))
   # (title: str, **kwargs) -> FeatureBuilder

   # Get method docstring
   print(sdk.features.create.__doc__)
   ```

2. **Add API Reference Section:**
   - Complete list of all SDK methods
   - Parameters and return types
   - Usage examples for each
   - Organized by collection type

3. **Improve CLI Help:**
   ```bash
   wipnote feature --help
   # Should mention: "For programmatic access, use Python SDK: sdk.features.create(...)"
   # Should mention: "See docs/SDK_FOR_AI_AGENTS.md for complete API reference"
   ```

4. **Add Interactive Exploration Examples:**
   ```python
   # Discover what collections exist
   print([attr for attr in dir(sdk) if not attr.startswith('_')])
   # ['bugs', 'chores', 'dep_analytics', 'epics', 'features', 'phases', 'spikes', 'tracks']

   # Explore a collection
   from wipnote.collections import BaseCollection
   print(BaseCollection.__dict__.keys())
   ```

---

## Immediate Action Items

### Critical (Fix Now)
1. **Investigate cross-project pollution** - This breaks fundamental assumptions
   - Test: Does SDK properly isolate to current project?
   - Test: What happens if `.wipnote/` doesn't exist in current dir?
2. **Implement delete functionality** - Agents need this for cleanup
   - Add `sdk.collections.delete(id)`
   - Add `wipnote feature delete` CLI command

### Important (Next)
1. **Add SDK method discoverability to AGENTS.md**
   - Runtime introspection examples
   - Complete API reference
2. **Write tests for cross-project isolation**
3. **Write tests for delete functionality**

---

## Test Cases to Add

### Cross-Project Isolation
```python
def test_sdk_isolates_to_current_project():
    """SDK should only see features from current project's .wipnote/"""
    # Create two projects with .wipnote/
    # Create features in each
    # Initialize SDK in project A
    # Verify it ONLY sees project A features
    # Verify it does NOT see project B features
```

### Delete Functionality
```python
def test_delete_feature():
    """Delete should remove file, update index, log event"""
    # Create a feature
    # Delete it
    # Verify file removed
    # Verify index updated
    # Verify deletion logged in events

def test_delete_with_dependents():
    """Delete should warn if other features depend on this one"""
    # Create feat-A that blocks feat-B
    # Try to delete feat-A
    # Should warn: "feat-B depends on this feature"
```

### SDK Discoverability
```python
def test_runtime_introspection():
    """Agents should be able to discover methods via introspection"""
    sdk = SDK(agent="test")

    # Should be able to list methods
    methods = [m for m in dir(sdk.features) if not m.startswith('_')]
    assert 'create' in methods
    assert 'delete' in methods
    assert 'edit' in methods
```

---

## Related Issues

- **Issue #42**: SDK should error if no `.wipnote/` found (instead of falling back)
- **Issue #89**: Need archive functionality (soft delete with status="archived")
- **Feature Request**: Batch delete for cleanup operations

---

## User Impact

**Gemini's experience:**
1. Saw wrong features (confused about which project)
2. Couldn't delete the wrong feature (no tooling)
3. Couldn't discover if SDK had the capability (poor docs)
4. Gave up and asked user for help

**This is a poor UX for AI agents.** We need to fix this ASAP.
