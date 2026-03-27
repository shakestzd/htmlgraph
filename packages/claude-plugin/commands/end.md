<!-- Efficiency: SDK calls: 2-3, Bash calls: 0, Context: ~5% -->

# /htmlgraph:end

End the current session and record work summary

## Usage

```
/htmlgraph:end
```

## Parameters



## Examples

```bash
/htmlgraph:end
```
Gracefully end the current session and show work summary



## Instructions for Claude

### Implementation:

**DO THIS:**

1. **Get current session:**
   ```bash
   htmlgraph session list
   ```

2. **Get current work item for handoff:**
   ```bash
   htmlgraph status
   ```
   Note the active feature title and description for handoff notes.

3. **End the session:**
   ```bash
   htmlgraph session end
   ```

4. **Extract session details:**
   - Session ID: `session.id`
   - Duration: Calculate from `session.created_at` to now
   - Event count: Query from database or session metadata
   - Features worked on: Get from session activities

5. **Present the session summary** using the output template below

6. **Include the summary of accomplishments:**
   - List features worked on during this session
   - Show any steps marked as complete
   - Acknowledge progress made

7. **Provide next-session guidance:**
   - Mention how to view dashboard: `htmlgraph serve`
   - Suggest next steps for the next session
   - Link to session record in `.htmlgraph/sessions/`
   - Show handoff notes if any

8. **CRITICAL CONSTRAINT:**
   - ONLY run `/htmlgraph:end` when the user explicitly requests it
   - Do NOT automatically end sessions
   - Wait for explicit user command
```

### Output Format:

## Session Ended

**Session ID:** {session_id}
**Duration:** {duration}
**Events:** {event_count}

### Work Summary
{features_worked_on_with_counts}

### Progress Made
- {accomplishment_summary}

---

Session recorded in `.htmlgraph/sessions/`
View dashboard: `htmlgraph serve`
