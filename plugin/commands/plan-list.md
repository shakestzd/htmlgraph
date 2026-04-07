# /htmlgraph:plan-list

List all YAML plans in the project with their status, slice counts, and creation dates.

## Usage

```
/htmlgraph:plan-list
```

## Examples

```bash
/htmlgraph:plan-list
```
Show all plans with status

## Instructions for Claude

Run `htmlgraph plan list` and present the output. Then scan for YAML plans and show additional detail:

```bash
htmlgraph plan list
```

For each YAML plan, show a summary table:

```bash
for f in .htmlgraph/plans/plan-*.yaml; do
  if [ -f "$f" ]; then
    id=$(basename "$f" .yaml)
    title=$(grep 'title:' "$f" | head -1 | sed 's/.*title: *//')
    status=$(grep 'status:' "$f" | head -1 | sed 's/.*status: *//')
    slices=$(grep -c '  - id:' "$f" 2>/dev/null || echo 0)
    echo "$id | $status | $slices slices | $title"
  fi
done
```

Present as a formatted table:

```
| Plan ID | Status | Slices | Title |
|---------|--------|--------|-------|
| plan-31cd5de1 | finalized | 6 | Codex Interoperability |
| plan-3a88d8a9 | draft | 4 | Agent-Agnostic Lazy Session Registration |
```

If the user asks about a specific plan, suggest: `/htmlgraph:plan-review <plan-id>`
