<!-- Efficiency: SDK calls: 1, Bash calls: 0, Context: ~3% -->

# /htmlgraph:feature-add

Add a new feature to the backlog

## Usage

```
/htmlgraph:feature-add [title]
```

## Parameters

- `title` (optional): The feature title. If not provided, ask the user.


## Examples

```bash
/htmlgraph:feature-add User Authentication
```
Add a new feature with the title "User Authentication"

```bash
/htmlgraph:feature-add
```
Prompt the user for a feature title



## Instructions for Claude

This command uses the CLI's `feature create` command.

### Implementation:

**DO THIS:**

1. **Check if title is provided:**
   - If title argument provided → proceed to step 2
   - If no title → ask the user: "What feature would you like to add?"

2. **Create the feature using CLI:**
   ```bash
   htmlgraph feature create "title"
   ```

3. **Present confirmation** using the output template below with the feature ID and title shown in the CLI output.

4. **Suggest next steps:**
   - Show command to start working: `/htmlgraph:feature-start {feature_id}`
   - Optionally suggest `/htmlgraph:plan` to plan the feature

### Output Format:

## Feature Added

**ID:** {feature_id}
**Title:** {title}
**Status:** todo

Start working on it with:
```bash
htmlgraph feature start {feature_id}
```
