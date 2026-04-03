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

1. **Validate tool type:**
   Standard tool types: `Decision`, `Research`, `Note`, `Context`, `Blocker`, `Insight`, `Refactor`

   If no tool type given, show the standard options above and ask the user to specify one.
   If a non-standard tool type is given, accept it but note the standard types.

2. **Get active feature attribution:**
   ```bash
   htmlgraph status
   ```

3. **Persist the track** using the CLI:
   ```bash
   htmlgraph track create "{title}"
   ```
   Manual tracking is primarily a documentation step. Activity is tracked automatically by hooks — this command surfaces and persists the context.

4. **Present summary** using the output template below.

5. **Show attribution:**
   - Display which feature this activity links to (from `htmlgraph status`)
   - If no active feature, suggest starting one with `/htmlgraph:start`

### Output Format:

## Activity Tracked

**Type:** {tool}
**Summary:** {summary}
**Files:** {', '.join(files) if files else 'None'}
**Attributed to:** {feature_info}

Activity recorded in current session.

{suggestion_text if no active feature}
