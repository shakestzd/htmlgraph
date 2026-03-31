<!-- Efficiency: SDK calls: 0, Bash calls: 0, Context: ~3% -->

# /htmlgraph:setup-statusline

Configure the HtmlGraph status line for Claude Code

## Usage

```
/htmlgraph:setup-statusline
```

## Parameters

None

## Examples

```bash
/htmlgraph:setup-statusline
```

Configure the Claude Code status line to show the active HtmlGraph work item.

## Instructions for Claude

This command configures `.claude/settings.json` to use the plugin-provided
`statusline.sh` script for the Claude Code status line.

The script uses `sqlite3` directly (~5ms) instead of `uv run python` (~1500ms),
making it fast enough for use on every prompt.

### Implementation:

```python
import json
from pathlib import Path

# Find the plugin statusline script via CLAUDE_PLUGIN_ROOT env var
import os
plugin_root = os.environ.get("CLAUDE_PLUGIN_ROOT", "")
if plugin_root:
    plugin_script = Path(plugin_root) / "scripts" / "statusline.sh"
else:
    # Fallback: search common locations
    candidates = [
        Path.home() / ".claude" / "plugins" / "htmlgraph" / "scripts" / "statusline.sh",
        Path("packages/claude-plugin/scripts/statusline.sh"),
    ]
    plugin_script = next((p for p in candidates if p.exists()), None)

if plugin_script is None or not plugin_script.exists():
    print("Error: statusline.sh not found. Is the htmlgraph plugin installed?")
    print("Run: claude plugin install htmlgraph")
else:
    # Ensure script is executable
    plugin_script.chmod(plugin_script.stat().st_mode | 0o111)

    # Read current settings
    settings_path = Path(".claude/settings.json")
    settings = json.loads(settings_path.read_text()) if settings_path.exists() else {}

    # Configure status line
    settings["statusLine"] = {
        "type": "command",
        "command": str(plugin_script.resolve()),
        "padding": 0
    }

    # Write settings
    settings_path.parent.mkdir(parents=True, exist_ok=True)
    settings_path.write_text(json.dumps(settings, indent=2) + "\n")

    print(f"Status line configured: {plugin_script.resolve()}")
    print("Restart Claude Code to see the status line.")
    print()
    print("Optional: Set HTMLGRAPH_OMP_CONFIG and HTMLGRAPH_OMP_BIN env vars")
    print("to use Oh My Posh for richer formatting.")
```
