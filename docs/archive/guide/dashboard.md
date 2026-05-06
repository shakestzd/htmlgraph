# Dashboard

The Wipnote dashboard provides real-time observability into agent activity, session history, and work item attribution.

## Accessing the Dashboard

Start the local server:

```bash
wipnote serve
```

Then open [http://localhost:4000](http://localhost:4000) in your browser.

## What the Dashboard Shows

### Activity Feed

The primary view is a real-time activity feed of agent tool calls and events. Each entry shows:

- The tool called (Read, Write, Bash, Task, etc.)
- Timestamps
- Which feature/work item the activity is attributed to
- Which agent performed the action

Events are nested hierarchically — tool calls made by a delegated subagent appear indented under the parent Task node.

### Session Tracking

The dashboard displays:

- **Active sessions**: Sessions currently in progress
- **Session history**: Past sessions with start/end times and duration
- Each session shows which agent ran it and what work it covered

### Feature Attribution

Each activity in the feed is linked to a work item (feature, bug, or spike). This lets you see at a glance which work item drove a given tool call or sequence of actions.

### Agent Activity Monitoring

The dashboard identifies which AI agents are active (e.g., `claude`, `haiku-coder`, `sonnet-coder`) and shows their ongoing and completed tool calls.

## Session Files

Individual session HTML files are stored at `.wipnote/sessions/` and can be opened directly in a browser for offline review.

## Next Steps

- [Features & Tracks Guide](features-tracks.md) - Creating and managing work
- [Sessions Guide](sessions.md) - Understanding session tracking
- [API Reference](../api/index.md) - Complete SDK documentation
