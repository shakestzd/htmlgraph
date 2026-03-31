---
name: ui-review
description: Run visual QA on a web application — screenshots pages and reports layout, readability, and correctness issues
user_invocable: true
---

# /htmlgraph:ui-review

Run a visual quality review of a web application.

## Usage
```
/htmlgraph:ui-review [url] [--pages /path1,/path2,...]
```

## Parameters
- `url` (optional): Base URL to review (e.g., http://localhost:3000). Auto-detects if not provided.
- `--pages` (optional): Comma-separated paths to review. Discovers from navigation if not provided.

## Examples

```bash
/htmlgraph:ui-review
```
Auto-detect dev server and review all discoverable pages.

```bash
/htmlgraph:ui-review http://localhost:5173
```
Review a Vite app.

```bash
/htmlgraph:ui-review http://localhost:4000 --pages /,/kanban,/costs
```
Review specific pages.

## Instructions for Claude

### Step 1: Determine target URL

If URL provided, use it. Otherwise auto-detect:

```bash
for port in 5173 3000 4000 8080 8000 3001 4200; do
  code=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:$port" 2>/dev/null)
  if [ "$code" = "200" ]; then
    echo "http://localhost:$port"
    break
  fi
done
```

If no server found, tell the user to start their dev server.

### Step 2: Create output directory

```bash
mkdir -p ui-review/
```

### Step 3: Delegate to ui-reviewer agent

**DELEGATION REQUIRED**: Always use `Agent(subagent_type="htmlgraph:ui-reviewer")`.

Prompt:
```
Review the web application at {url}.
{If --pages: "Review these pages: {pages}"}
{If no --pages: "Discover pages from navigation and review all of them."}
Save screenshots to ui-review/.
```

### Step 4: Present results

Show the agent's findings: per-page severity, issues list, screenshot paths, summary table.

## When to Use

Run after any UI change:
- Template/component changes
- CSS/styling updates
- Data query changes affecting display
- Layout or navigation changes

This is a quality gate — do not mark UI work as done without passing this review.
