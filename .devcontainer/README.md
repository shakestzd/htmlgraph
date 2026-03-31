# HtmlGraph Devcontainer

This devcontainer gives HtmlGraph a standalone development environment with the main agent CLIs and the repo toolchain preinstalled.

Included:

- Python 3.11
- `uv`
- Go 1.23
- Node.js 22 and `npm`
- GitHub CLI (`gh`)
- Claude Code CLI
- OpenAI Codex CLI
- Gemini CLI
- Common shell/build tools: `git`, `make`, `ripgrep`, `fd`, `jq`, `sqlite3`, `shellcheck`

## Start

1. Open the repository in VS Code.
2. Run `Dev Containers: Reopen in Container`.
3. Wait for `.devcontainer/post-create.sh` to finish.

## Authentication

The container supports two ways to authenticate the AI CLIs:

1. Pass host environment variables through automatically:
   - `ANTHROPIC_API_KEY`
   - `OPENAI_API_KEY`
   - `GEMINI_API_KEY`
   - `GOOGLE_API_KEY`
   - `GOOGLE_CLOUD_PROJECT`
   - `GOOGLE_CLOUD_LOCATION`
2. Log in interactively inside the container by running:
   - `claude`
   - `codex`
   - `gemini`

## Persistence

To keep the container isolated from your host setup while still surviving rebuilds, the devcontainer mounts named Docker volumes for:

- `/home/vscode/.claude`
- `/home/vscode/.codex`
- `/home/vscode/.gemini`
- `/home/vscode/.cache/uv`

That keeps agent state and auth local to the container instead of depending on your host home directory.

## Repo Bootstrap

The post-create script runs:

```bash
npm install -g @anthropic-ai/claude-code @google/gemini-cli @openai/codex
uv sync --frozen --all-extras --all-groups
```

If you want Playwright browser binaries in the container too:

```bash
uv run playwright install --with-deps chromium
```

## Why npm for the agent CLIs?

Anthropic's current docs prefer a native Claude Code install on normal developer machines, but the container uses npm for Claude, Codex, and Gemini so the setup stays consistent and aligned with the container's own Node runtime.
