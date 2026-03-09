# Archived Dashboard Templates

These templates were archived to reduce confusion during development.

## Files

### `dashboard.html.legacy-20260111` (284KB)
- **Original location**: `src/python/htmlgraph/dashboard.html`
- **Why archived**: Not used by the FastAPI server. The server uses Jinja2 partials in `src/python/htmlgraph/api/templates/partials/`
- **Archived on**: 2026-01-11
- **Reason**: Caused confusion during debugging - we were editing this file but the server was serving different templates

## Active Templates (DO NOT ARCHIVE)

The FastAPI dashboard uses these templates:
- **Main template**: `src/python/htmlgraph/api/templates/dashboard-redesign.html`
- **Partials**: `src/python/htmlgraph/api/templates/partials/*.html`
  - `partials/agents.html` - Agent Fleet Status + Complete Activity Feed
  - `partials/activity-feed.html` - Main activity feed
  - `partials/features.html` - Features Kanban
  - `partials/orchestration.html` - Orchestration view
  - `partials/metrics.html` - Metrics dashboard

## How to Verify Active Template

Check `src/python/htmlgraph/api/main.py`:
```python
@app.get("/", response_class=HTMLResponse)
async def dashboard(request: Request) -> HTMLResponse:
    return templates.TemplateResponse(
        "dashboard-redesign.html",  # ← This is the active template
        {"request": request}
    )
```

Views are loaded via HTMX from `/views/*` endpoints which render partials.
