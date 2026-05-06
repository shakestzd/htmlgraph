# Installation

## Requirements

Wipnote requires Python 3.10 or higher.

## Install from PyPI

The easiest way to install Wipnote is via pip:

```bash
pip install wipnote
```

Or using uv (recommended):

```bash
uv pip install wipnote
```

> **Note:** Use `uvx wipnote` (not `uv run wipnote`) to run CLI commands after installation.
> `uv run` uses your project's lockfile and may run a cached/locked version instead of the latest.
> `uvx` always runs the installed package directly, ensuring you get the correct version.

## Install from Source

Clone the repository and install in development mode:

```bash
git clone https://github.com/shakestzd/wipnote.git
cd wipnote
uv pip install -e .
```

## Verify Installation

Check that Wipnote is installed correctly:

```bash
uv run python -c "import wipnote; print(wipnote.__version__)"
```

Or using the CLI:

```bash
wipnote --version
```

## Optional Dependencies

For development and testing:

```bash
# Install development dependencies
uv pip install -e ".[dev]"

# Install testing dependencies
uv pip install -e ".[test]"

# Install documentation dependencies
uv pip install -e ".[docs]"
```

## Agent Integration

### Claude Code Plugin

```bash
# Install the Wipnote plugin for Claude Code
claude plugin install wipnote

# Or from local marketplace
claude plugin marketplace add local-marketplace
claude plugin install wipnote
```

### Gemini CLI Extension

```bash
# Install the Wipnote extension for Gemini CLI
gemini extension install wipnote
```

### Codex CLI Skill

```bash
# Install the Wipnote skill for Codex CLI
codex skill install wipnote
```

## Next Steps

- [Quick Start Guide](quick-start.md) - Get started with your first graph
- [Core Concepts](concepts.md) - Understand Wipnote fundamentals
