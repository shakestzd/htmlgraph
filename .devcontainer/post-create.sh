#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "$0")/.."

echo "Installing AI agent CLIs..."
npm install -g --no-fund --no-audit @anthropic-ai/claude-code @google/gemini-cli @openai/codex

echo "Syncing HtmlGraph dependencies..."
uv sync --frozen --all-extras --all-groups

echo
echo "Installed tool versions:"
python3 --version
uv --version
go version
node --version
npm --version
claude --version || true
codex --version || true
gemini --version || true

cat <<'EOF'

Devcontainer bootstrap complete.

Next steps:
- Run `claude`, `codex`, and `gemini` once each to complete interactive login if you are not passing API keys in from the host.
- Optional browser setup for Playwright tests: `uv run playwright install --with-deps chromium`
- Start HtmlGraph locally with: `uv run htmlgraph serve`

Persistent auth/config is stored in named Docker volumes mounted at:
- ~/.claude
- ~/.codex
- ~/.gemini
EOF
