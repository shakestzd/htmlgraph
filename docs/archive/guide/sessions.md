# Sessions

Sessions automatically track all activity during an agent's work, providing complete attribution and continuity across work periods.

## What is a Session?

A **session** is a period of continuous work by an agent. Wipnote automatically:

- Creates a session when work begins
- Logs all tool calls and interactions
- Attributes activities to features
- Generates a session summary when work ends
- Preserves full context for future sessions

Each session is stored as an HTML file in `.wipnote/sessions/`.

## Session Lifecycle

### 1. Session Start

Sessions start automatically when:

- An agent begins working (via hooks)
- You run `wipnote feature start <id>`

```bash
# Start working on a feature (creates a session)
wipnote feature start feat-a1b2c3d4

# View current session status
wipnote status
```

### 2. Activity Logging

During the session, Wipnote logs:

- **Tool calls**: Every Read, Write, Edit, Bash command
- **User prompts**: Questions and requests
- **Feature updates**: Status changes, step completions
- **Decisions**: Notes and tracking via `sdk.track()`

All activities are attributed to the current feature.

### 3. Session End

Sessions end automatically when:

- Work is completed
- Feature is marked as done
- Agent exits

The session end hook generates a summary of all work performed.

## Viewing Sessions

### CLI

```bash
# List all sessions
wipnote session list

# Show specific session
wipnote session show session-abc-123

# View current session status
wipnote status
```

### Browser

Open `.wipnote/sessions/session-abc-123.html` in any browser to view:

- Session timeline
- Activity log with timestamps
- Features worked on
- Agent attribution
- Full event history

## Activity Attribution

Activities are attributed to features based on:

1. **Current feature**: Set via `wipnote feature start <id>`
2. **Primary feature**: If multiple features are active
3. **Drift detection**: Alerts if activity doesn't match the current feature

### Example

```bash
# Create and start a feature (attributes all subsequent activity to it)
wipnote feature create "Add login page" --priority high
wipnote feature start feat-a1b2c3d4

# Record decisions and notes as spikes
wipnote spike create "Implementing OAuth flow: chose Passport.js for simpler API"
```

## Drift Detection

Wipnote monitors activity to detect "drift" - when work diverges from the assigned feature.

### How It Works

The drift detector analyzes:

- Files modified
- Keywords in commands and prompts
- Feature descriptions and steps

If drift is detected (score > 0.7), you'll see a warning:

```
⚠️  Drift detected (0.85): Activity may not align with feature-auth-001
   Consider switching features or updating the feature scope
```

### Responding to Drift

When you see drift warnings:

1. **Expected drift**: Work naturally spans features (ignore)
2. **Switch features**: Use `wipnote feature primary <id>`
3. **Wrong feature**: Update the feature's scope or file patterns

## Session Continuity

### Across Sessions

Wipnote preserves context between sessions:

```bash
# End of Session 1
wipnote feature start feature-001
# Work on feature...
# Session ends

# Start of Session 2
wipnote status
# Shows: Previous session worked on feature-001
#        Feature is 60% complete
#        Last activity: "Implemented OAuth callback"
```

### Session Summaries

At session end, Wipnote generates a summary:

```markdown
Session: session-abc-123
Duration: 2h 34m
Agent: claude
Features: feature-001 (80% complete)

Activities:
- Implemented OAuth callback
- Added JWT middleware
- Wrote integration tests
- Fixed redirect bug

Next Steps:
- Complete user profile endpoint
- Add error handling
- Deploy to staging
```

## Manual Session Management

### Track Custom Activities

```bash
# Record decisions as spikes linked to the current feature
wipnote spike create "Decided to use Passport.js instead of Auth0 (simpler API)"
```

### Set Primary Feature

When multiple features are active:

```bash
wipnote feature primary feature-001
```

## Session Files

Each session creates an HTML file with:

```html
<article id="session-abc-123"
         data-type="session"
         data-agent="claude"
         data-start="2024-12-16T10:30:00Z"
         data-end="2024-12-16T13:04:00Z">

    <h1>Session ABC-123</h1>

    <section data-features>
        <h3>Features Worked On:</h3>
        <ul>
            <li><a href="../features/feature-001.html">User Authentication (80%)</a></li>
        </ul>
    </section>

    <section data-activity-log>
        <h3>Activity Log:</h3>
        <ol reversed>
            <li data-timestamp="2024-12-16T13:04:00Z">
                Completed integration tests
            </li>
            <li data-timestamp="2024-12-16T12:30:00Z">
                Implemented OAuth callback
            </li>
            <!-- ... more activities ... -->
        </ol>
    </section>
</article>
```

## Best Practices

### 1. One Feature at a Time

Focus on a single feature per session for clear attribution:

```bash
wipnote feature start feature-001
# Work only on this feature
wipnote feature complete feature-001
```

### 2. Document Decisions

Record important decisions as you make them:

```bash
wipnote spike create "Chose PostgreSQL over MongoDB for better transaction support"
```

### 3. Complete Sessions

Mark features as complete to trigger session summaries:

```bash
wipnote feature complete feature-001
```

### 4. Review Previous Sessions

Before starting new work, review what happened last:

```bash
wipnote status
wipnote session list --recent 5
```

## Integration with Claude Code

When using the Claude Code plugin, sessions are managed automatically via hooks bundled in the plugin. Install the plugin with:

```bash
claude plugin install wipnote
```

The plugin registers hooks (configured in `hooks.json` inside the plugin) that:

- Start sessions when Claude begins working
- Log all tool calls automatically
- Detect drift and warn you
- Generate summaries when sessions end
- Provide context at the start of new sessions

## Next Steps

- [Features & Tracks Guide](features-tracks.md) - Creating and managing work
- [Agents Guide](agents.md) - Agent integration patterns
- [Dashboard Guide](dashboard.md) - Visualizing session history
- [API Reference](../api/sdk.md) - Complete SDK documentation
