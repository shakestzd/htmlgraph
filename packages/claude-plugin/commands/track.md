<!-- Efficiency: SDK calls: 2, Bash calls: 0, Context: ~5% -->

# /htmlgraph:track

Manually track an activity or note

## Usage

```
/htmlgraph:track <tool> <summary> [--files file1 file2]
```

## Parameters

- `tool` (required): The tool/action type (e.g., "Note", "Decision", "Research")
- `summary` (required): Description of the activity
- `files` (optional): Related files


## Examples

```bash
/htmlgraph:track "Decision" "Chose React over Vue for frontend" --files src/components/App.tsx
```
Track a decision with related files

```bash
/htmlgraph:track "Research" "Investigated auth options JWT vs sessions"
```
Track research activity

```bash
/htmlgraph:track "Note" "User prefers dark mode as default"
```
Track a general note



## Instructions for Claude

### Implementation:

**DO THIS:**

1. **Validate or suggest tool types:**
   Standard types: Decision, Research, Note, Context, Blocker, Insight, Refactor
   Accept any type, but note if non-standard.

2. **Track the activity via spike:**
   ```bash
   htmlgraph spike create "<tool>: <summary>"
   ```
   Or note it as a comment linked to the active work item.

3. **Get active feature attribution:**
   ```bash
   htmlgraph status
   ```

4. **Present summary** using the output template below

5. **Show attribution:**
   - Display which feature this activity is linked to
   - If no active feature, suggest starting one with `/htmlgraph:start`

### Output Format:

## Activity Tracked

**Type:** {tool}
**Summary:** {summary}
**Files:** {', '.join(files) if files else 'None'}
**Attributed to:** {feature_info}

Activity recorded in current session.

{suggestion_text if no active feature}
