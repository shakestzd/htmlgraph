# /htmlgraph:help

Display available HtmlGraph commands and usage

## Usage

```
/htmlgraph:help
```

## Parameters



## Examples

```bash
/htmlgraph:help
```
Show all available commands and their descriptions



## Instructions for Claude

### Implementation:

**DO THIS:**

1. **Retrieve help text:**
   ```bash
   htmlgraph --help
   ```

2. **Present the complete help message** using the output template above

3. **Organize by categories:**
   - Session Management - user-facing workflow commands
   - Feature Management - feature lifecycle commands
   - Utilities - setup, dashboard, and tracking
   - CLI Commands - direct CLI usage alternatives
   - Dashboard - browser-based viewing instructions

4. **Make it actionable:**
   - Each command includes a description of what it does
   - Include usage examples where applicable
   - Provide CLI equivalents for power users

5. **Highlight key information:**
   - Dashboard access: `htmlgraph serve` → http://localhost:8080
   - All commands start with `/htmlgraph:` for consistency
   - CLI is available as alternative interface
```

### Output Format:

## HtmlGraph Commands

### Session Management
- `/htmlgraph:start` - Start session, see status, choose what to work on
- `/htmlgraph:end` - End current session gracefully
- `/htmlgraph:status` - Quick status check

### Feature Management
- `/htmlgraph:feature-add [title]` - Add a new feature
- `/htmlgraph:feature-start <id>` - Start working on a feature
- `/htmlgraph:feature-complete [id]` - Mark feature as complete
- `/htmlgraph:feature-primary <id>` - Set primary feature for attribution

### Utilities
- `/htmlgraph:init` - Initialize HtmlGraph in project
- `/htmlgraph:serve [port]` - Start dashboard server
- `/htmlgraph:track <tool> <summary>` - Manually track activity

### CLI Commands
You can also use the CLI directly:
```bash
htmlgraph --help
htmlgraph status
htmlgraph feature list
htmlgraph session list
```

### Dashboard
View progress in browser:
```bash
htmlgraph serve
# Open http://localhost:8080
```
