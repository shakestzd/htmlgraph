# UI Review Skill

Run a visual quality review of the HtmlGraph Phoenix LiveView dashboard. Screenshots all pages using browser automation and reports layout, readability, and data correctness issues.

## Usage

```
/htmlgraph:ui-review [page]
```

## Parameters

- `page` (optional): Specific page to review. One of: `activity`, `graph`, `kanban`, `costs`. Default: all pages.

## What This Does

1. Ensures the Phoenix dashboard is running at `http://localhost:4000`
2. Creates the screenshot output directory: `ui-review/`
3. Delegates to the `ui-reviewer` agent which:
   - Navigates to each dashboard page via chrome-devtools MCP
   - Takes viewport and full-page screenshots
   - Analyzes each page for visual quality issues
   - Reports findings with CRITICAL/MAJOR/MINOR/OK severity
4. Presents findings with screenshot paths

## Instructions for Claude

### Step 1: Check dashboard is running

```bash
curl -s -o /dev/null -w "%{http_code}" http://localhost:4000
```

If the response is not `200`, inform the user:
> The Phoenix dashboard is not running at http://localhost:4000. Start it with `mix phx.server` in the `dashboard/` directory, then run `/htmlgraph:ui-review` again.

### Step 2: Create output directory

```bash
mkdir -p /Users/shakes/DevProjects/htmlgraph/ui-review/
```

### Step 3: Determine scope

- If `page` argument is provided, map it to the URL:
  - `activity` → `http://localhost:4000`
  - `graph` → `http://localhost:4000/graph`
  - `kanban` → `http://localhost:4000/kanban`
  - `costs` → `http://localhost:4000/costs`
- If no argument, review all four pages.

### Step 4: Delegate to ui-reviewer agent

**DELEGATION REQUIRED**: Always use `Agent(subagent_type="htmlgraph:ui-reviewer")` for the actual review work.

Prompt to pass to the agent:

```
Review the following dashboard page(s): [list pages to review].

For each page:
1. Navigate to the URL
2. Wait for the page to fully load
3. Take a viewport screenshot and a full-page screenshot
4. Save screenshots to /Users/shakes/DevProjects/htmlgraph/ui-review/
5. Analyze for layout, readability, data correctness, and visual hierarchy issues
6. Report findings using the CRITICAL/MAJOR/MINOR/OK severity scale

Start with the work item attribution step before taking any screenshots.
```

### Step 5: Present results

Show the agent's findings to the user, including:
- Per-page severity rating
- List of issues with severity
- Screenshot file paths
- Summary table across all reviewed pages

## Example Output

```
Running visual QA on 4 dashboard pages...

## Activity Feed — OK
Screenshot: ui-review/activity-20260322-230600.png
Looks Good: Session groups clean, feature badges visible, nesting correct.

## Graph — MAJOR
Screenshot: ui-review/graph-20260322-230612.png
Issues:
  1. [MAJOR] Stats bar shows 0 nodes despite 12 features in DB
  2. [MINOR] Node labels truncated at 15 chars — cut off for long feature titles

## Kanban — OK
Screenshot: ui-review/kanban-20260322-230624.png
Looks Good: All three columns rendered, cards readable.

## Costs — MINOR
Screenshot: ui-review/costs-20260322-230636.png
Issues:
  1. [MINOR] Totals row missing bottom border — blends into last data row

## Summary
| Page | Status | Issues |
|------|--------|--------|
| Activity Feed | OK | 0 |
| Graph | MAJOR | 2 |
| Kanban | OK | 0 |
| Costs | MINOR | 1 |
```

## When to Use

Run this skill after any change to:
- Phoenix LiveView templates (`dashboard/lib/**/*_live.ex`, `dashboard/lib/**/*.html.heex`)
- CSS/styling files
- Dashboard data queries that affect what is displayed
- Layout or component structure changes

This is a quality gate — do not mark UI work as done without passing this review.
