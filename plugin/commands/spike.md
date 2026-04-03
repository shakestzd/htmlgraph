# /htmlgraph:spike

Create a research/planning spike

## Usage

```
/htmlgraph:spike <title> [--description TEXT] [--track ID]
```

## Parameters

- `title` (required): Spike title (e.g., "Research authentication options")
- `--description` (optional): Description text for the spike
- `--track` (optional): Track ID to link to


## Examples

```bash
/htmlgraph:spike "Research OAuth providers"
```
Create a spike

```bash
/htmlgraph:spike "Investigate caching strategies" --description "Focus on Redis vs Memcached"
```
Create a spike with description

```bash
/htmlgraph:spike "Plan data migration" --track trk-abc123
```
Spike linked to a track



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
