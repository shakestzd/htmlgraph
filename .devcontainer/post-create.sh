#!/usr/bin/env bash

set -euo pipefail

# Fix ownership of named-volume mount points (Docker creates these as root:root)
sudo chown -R vscode:vscode /workspaces/htmlgraph/.htmlgraph /workspaces/htmlgraph/.claude 2>/dev/null || true

cd "$(dirname "$0")/.."

export PATH="${HOME}/.local/bin:${PATH}"

echo "==> Installing AI agent CLIs..."
npm install -g --no-fund --no-audit \
    @anthropic-ai/claude-code \
    @google/gemini-cli \
    @openai/codex

echo "==> Building htmlgraph from source..."
./plugin/build.sh

echo "==> Running quality gates..."
go build ./...
go vet ./...

echo "==> Fixing Claude Code plugin data directory permissions..."
sudo chown -R vscode:vscode ~/.claude 2>/dev/null || true
mkdir -p ~/.claude/plugins/data

echo "==> Installing uv..."
if ! command -v uv >/dev/null 2>&1; then
  curl -LsSf https://astral.sh/uv/install.sh | sh
fi

echo "==> Installing oh-my-zsh..."
if [ ! -d "$HOME/.oh-my-zsh" ]; then
  RUNZSH=no CHSH=no KEEP_ZSHRC=yes sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)"
fi

echo "==> Installing powerlevel10k..."
if [ ! -d "$HOME/powerlevel10k" ]; then
  git clone --depth=1 https://github.com/romkatv/powerlevel10k.git "$HOME/powerlevel10k"
fi

echo "==> Installing zsh plugins..."
ZSH_CUSTOM="${ZSH_CUSTOM:-$HOME/.oh-my-zsh/custom}"
[ -d "$ZSH_CUSTOM/plugins/zsh-syntax-highlighting" ] || \
  git clone --depth=1 https://github.com/zsh-users/zsh-syntax-highlighting.git "$ZSH_CUSTOM/plugins/zsh-syntax-highlighting"
[ -d "$ZSH_CUSTOM/plugins/zsh-autosuggestions" ] || \
  git clone --depth=1 https://github.com/zsh-users/zsh-autosuggestions.git "$ZSH_CUSTOM/plugins/zsh-autosuggestions"

echo "==> Copying dotfiles..."
cp "$(dirname "$0")/dotfiles/.zshrc" "$HOME/.zshrc"
cp "$(dirname "$0")/dotfiles/.p10k.zsh" "$HOME/.p10k.zsh"

echo "==> Setting default shell to zsh..."
sudo chsh -s /usr/bin/zsh vscode 2>/dev/null || chsh -s /usr/bin/zsh || true

echo
echo "==> Installed tool versions:"
go version
node --version
npm --version
claude --version || true
codex --version || true
gemini --version || true
htmlgraph --version || true

cat <<'EOF'

Devcontainer bootstrap complete.

This is a source-development environment — every change you make to
cmd/, internal/, or plugin/ can be rebuilt with `htmlgraph build`.

Next steps:
- Authenticate the CLIs once (stored in persistent volumes):
    claude           # OAuth browser login (or API key)
    codex
    gemini
- Launch Claude Code in dev mode so it loads the plugin from source:
    htmlgraph claude --dev
- Start the dashboard:
    htmlgraph serve
    # http://localhost:8080
- Run the full test suite on demand:
    bash scripts/devcontainer-verify.sh

Persistent volumes mounted:
  /home/vscode/.claude         — Claude Code credentials
  /home/vscode/.codex          — Codex credentials
  /home/vscode/.gemini         — Gemini credentials
  /home/vscode/.local          — htmlgraph binary + version metadata
  <workspace>/.htmlgraph       — devcontainer-only work item state
                                 (isolated from your host .htmlgraph/)
EOF
