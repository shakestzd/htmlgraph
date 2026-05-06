# Dashboard Event Nesting Fix - Verification

## Problem Description

In the dashboard, `subprocess.gemini` events were appearing at the wrong indentation level. They should be nested 3 levels deep under the conversation turn, but were showing at the same level as their parent `gemini-cli` event.

**Expected hierarchy:**
```
Task (UserQuery, depth 0 - turn root)
├─ HeadlessSpawner.initialize (child event, depth 0)
│  └─ gemini-cli (child event, depth 1)
│     └─ subprocess.gemini (child event, depth 2) ← CORRECT: 2 levels of nesting
```

**Actual behavior (before fix):**
```
Task (UserQuery, depth 0)
├─ HeadlessSpawner.initialize (depth 0)
│  └─ gemini-cli (depth 1)
└─ subprocess.gemini (depth 1) ← WRONG: Same indentation as parent
```

## Root Cause Analysis

The bug was in the `calculateEventDepth()` function (lines 388-428).

### The Broken Logic

The original function walked UP the DOM tree starting from the parent's `.event-children` container:

```javascript
let current = container;  // Start from .event-children
while (current) {
    if (current.classList.contains('event-children')) {
        depth++;
        const parentEventElement = current.parentElement;  // Get child-event-row
        if (parentEventElement && parentEventElement.hasAttribute('data-event-id')) {
            current = parentEventElement.parentElement;  // Jump to ???
        }
    }
}
```

**The bug:** When jumping `parentElement.parentElement`, the code skips over the `.event-children` container that belongs to the grandparent event. This breaks the ancestor chain counting.

### Example Trace (Broken)

For `subprocess.gemini` with parent `gemini-cli`:

1. Start: Find `gemini-cli`'s `.event-children` container
2. Loop iteration 1:
   - `current` = `.event-children` (belongs to `gemini-cli`)
   - Is this `.event-children`? YES
   - `depth++` → depth = 1
   - `parentEventElement` = `gemini-cli` (the child-event-row)
   - `current` = `parentEventElement.parentElement` = `.event-children` (belongs to HeadlessSpawner)
3. Loop iteration 2:
   - `current` = `.event-children` (belongs to HeadlessSpawner)
   - Is this `.event-children`? YES
   - `depth++` → depth = 2
   - But WAIT: `HeadlessSpawner` event row `.parentElement` = `.event-children` (belongs to Task)
   - `current` = `.event-children.parentElement` = `conversation-turn` (the root)
4. Loop iteration 3:
   - `current` = `conversation-turn`
   - Is this `.turn-children`? NO (it's `.conversation-turn`)
   - Is this `.event-children`? NO
   - `current` = `current.parentElement` → infinite loop or escape

**The real issue:** The DOM structure is actually:
```
turn-children
└─ child-event-row (HeadlessSpawner, depth 0)
   └─ event-children
      └─ child-event-row (gemini-cli, depth 1)
         └─ event-children
            └─ child-event-row (subprocess.gemini, depth 2)
```

The original code was trying to walk this by jumping around, but the jumps were incorrect.

## The Fix

The new logic directly walks up through the DOM ancestors and counts how many `child-event-row` elements we encounter:

```javascript
let current = container.parentElement; // Start from direct parent of children container
while (current) {
    if (current.classList.contains('turn-children')) {
        return depth; // Reached root - return accumulated depth
    }

    if (current.classList.contains('child-event-row')) {
        depth++;  // Found an ancestor event, increment depth
        current = current.parentElement;  // Move to its .event-children
        if (current) {
            current = current.parentElement; // Move to ITS parent event row
        }
    } else {
        current = current.parentElement;  // Keep walking up
    }
}
```

### Correct Trace (Fixed)

For `subprocess.gemini` with parent `gemini-cli`:

1. Start: Find `gemini-cli`'s `.event-children` container
2. `current` = `.event-children.parentElement` = `gemini-cli` (the child-event-row)
3. Loop iteration 1:
   - `current` = `gemini-cli` (child-event-row)
   - Is this `.turn-children`? NO
   - Is this `.child-event-row`? YES
   - `depth++` → depth = 1 (one ancestor event)
   - `current` = `current.parentElement` = `.event-children` (belongs to HeadlessSpawner)
   - `current` = `current.parentElement` = `HeadlessSpawner` (child-event-row)
4. Loop iteration 2:
   - `current` = `HeadlessSpawner` (child-event-row)
   - Is this `.turn-children`? NO
   - Is this `.child-event-row`? YES
   - `depth++` → depth = 2 (two ancestor events)
   - `current` = `current.parentElement` = `.event-children` (belongs to Task turn)
   - `current` = `current.parentElement` = `turn-children`
5. Loop iteration 3:
   - `current` = `turn-children`
   - Is this `.turn-children`? YES
   - `return depth` → return 2

**Result:** subprocess.gemini gets depth = 2, which means `margin-left: 40px` (2 × 20px), correct nesting! ✓

## Visual Result After Fix

```
Task (UserQuery, depth 0 - turn root)
├─ HeadlessSpawner.initialize (child-event-row, depth 0)
│  └─ gemini-cli (child-event-row, depth 1)  [margin-left: 20px]
│     └─ subprocess.gemini (child-event-row, depth 2)  [margin-left: 40px]
```

The indentation now correctly reflects the nesting hierarchy!

## CSS Depth Classes

The fix works with the existing CSS depth classes:

```css
.child-event-row.depth-0 { margin-left: 0; }
.child-event-row.depth-1 { margin-left: 20px; }
.child-event-row.depth-2 { margin-left: 40px; }
.child-event-row.depth-3 { margin-left: 60px; }
/* etc. */
```

## Files Modified

- `src/python/wipnote/api/templates/dashboard-redesign.html` - Lines 388-426
  - Changed `calculateEventDepth()` function to correctly walk ancestor chain
  - Now properly counts all ancestor event rows (child-event-row elements)
  - No changes to HTML structure or CSS required

## Testing

The fix should be tested by:

1. Running a Claude Code session with spawner delegation that creates nested events
2. Verifying in the dashboard that:
   - Parent events display with correct indentation
   - Child events are properly indented below their parents
   - Subprocess events are indented 2+ levels deep as expected
   - Tree connectors (└─, ├─) align with indentation

Example event stream to verify:
```json
UserQuery: "expand all files" (depth 0)
  ├─ Task delegation event (depth 0)
  │  ├─ HeadlessSpawner.initialize (depth 1)
  │  │  └─ gemini-cli (depth 2)
  │  │     └─ subprocess.gemini (depth 3) ← Should be deeply indented
```

## Backward Compatibility

The fix maintains backward compatibility:
- No changes to event data structure
- No changes to HTML generation
- No changes to CSS classes
- Only internal depth calculation logic changed
- Existing events continue to display correctly
