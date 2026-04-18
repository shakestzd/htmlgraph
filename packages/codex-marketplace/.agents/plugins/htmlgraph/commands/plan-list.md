# /htmlgraph:plan-list

List all YAML plans in the project with their status, slice counts, and creation dates. Sorted newest first.

## Usage

```
/htmlgraph:plan-list
```

## Instructions for Claude

Single command — no shell parsing needed:

```bash
htmlgraph plan list-yaml
```

Present the output as-is. If the user wants to review a specific plan, suggest:

```
/htmlgraph:plan-review <plan-id>
```
