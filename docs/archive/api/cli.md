# Command Line Interface

Wipnote provides command-line tools for managing sessions, features, and the dashboard.

## Quick Reference

### Serve Dashboard
```bash
uv run wipnote serve
# Visit http://localhost:8000
```

### Check Status
```bash
uv run wipnote status
```

### List Features
```bash
uv run wipnote feature list
```

### Get Feature Details
```bash
uv run wipnote feature show <feature-id>
```

### List Sessions
```bash
uv run wipnote session list
```

### View Session Details
```bash
uv run wipnote session show <session-id>
```

## Full Reference

For complete CLI documentation, see:
- [API Reference](reference.md) - Full API overview
- [SDK Reference](sdk.md) - Python SDK documentation
- [Guide: Dashboard](../guide/dashboard.md) - Using the dashboard

## Environment Variables

Set these to control Wipnote behavior:

```bash
# Data storage location
export HTMLGRAPH_HOME=~/.wipnote

# Server configuration
export HTMLGRAPH_HOST=localhost
export HTMLGRAPH_PORT=8000

# Logging
export HTMLGRAPH_LOG_LEVEL=INFO
```
