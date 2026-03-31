# /htmlgraph:spike

Create a research/planning spike

## Usage

```
/htmlgraph:spike <title> [--context TEXT] [--timebox HOURS]
```

## Parameters

- `title` (required): Spike title (e.g., "Research authentication options")
- `--context` (optional) (default: ): Background information for the spike
- `--timebox` (optional) (default: 4.0): Time limit in hours


## Examples

```bash
/htmlgraph:spike "Research OAuth providers"
```
Create a 4-hour research spike

```bash
/htmlgraph:spike "Investigate caching strategies" --timebox 2
```
Create a 2-hour spike

```bash
/htmlgraph:spike "Plan data migration" --context "Moving from SQL to NoSQL"
```
Spike with background context



## Instructions for Claude

### Implementation:

```bash
htmlgraph spike create "{title}"
```

Present the spike ID and title from CLI output using the output template below.

### Output Format:

## Spike Created

**ID:** {id}
**Title:** {title}
**Status:** {status}
**Timebox:** {timebox_hours} hours

### Steps
{steps}

Spike is now active. Complete the steps to finish planning.
