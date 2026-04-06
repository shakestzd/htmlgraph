#!/usr/bin/env bash
# build.sh - Build the htmlgraph Go binary for the plugin.
#
# Usage:
#   ./build.sh          # Dev mode: binary at hooks/bin/htmlgraph
#   ./build.sh --dist   # Dist mode: binary at hooks/bin/htmlgraph-bin,
#                        #            bootstrap script at hooks/bin/htmlgraph

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
GO_DIR="${SCRIPT_DIR}/.."
BIN_DIR="${SCRIPT_DIR}/hooks/bin"
DIST_MODE=false

for arg in "$@"; do
    case "${arg}" in
        --dist) DIST_MODE=true ;;
        *)      echo "Unknown flag: ${arg}" >&2; exit 1 ;;
    esac
done

cd "${GO_DIR}"

# Copy notebook files for embedding (source of truth: prototypes/)
echo "  Copying notebook files for embedding..."
mkdir -p internal/notebook/files
cp prototypes/plan_notebook.py internal/notebook/files/
cp prototypes/plan_ui.py internal/notebook/files/
cp prototypes/plan_persistence.py internal/notebook/files/
cp prototypes/critique_renderer.py internal/notebook/files/
cp prototypes/dagre_widget.py internal/notebook/files/
cp prototypes/chat_widget.py internal/notebook/files/
cp prototypes/claude_chat.py internal/notebook/files/
cp prototypes/amendment_parser.py internal/notebook/files/

VERSION_RAW=$(git describe --tags --always 2>/dev/null || echo "dev")
# Strip leading 'v' for consistent version strings (goreleaser, plugin.json)
VERSION="${VERSION_RAW#v}"

if [ "${DIST_MODE}" = true ]; then
    echo "Building htmlgraph (dist mode, version: ${VERSION})..."
    go build -ldflags "-s -w -X main.version=${VERSION}" \
        -o "${BIN_DIR}/htmlgraph-bin" ./cmd/htmlgraph/
    chmod +x "${BIN_DIR}/htmlgraph-bin"

    # Copy bootstrap script as the entry point
    cp "${BIN_DIR}/bootstrap.sh" "${BIN_DIR}/htmlgraph"
    chmod +x "${BIN_DIR}/htmlgraph"

    # Install to ~/.local/bin so bootstrap skips download
    INSTALL_DIR="${HOME}/.local/bin"
    META_DIR="${HOME}/.local/share/htmlgraph"
    mkdir -p "${INSTALL_DIR}" "${META_DIR}"
    rm -f "${INSTALL_DIR}/htmlgraph"  # Fresh inode avoids macOS signature cache
    cp "${BIN_DIR}/htmlgraph-bin" "${INSTALL_DIR}/htmlgraph"
    chmod +x "${INSTALL_DIR}/htmlgraph"
    echo "${VERSION}" > "${META_DIR}/.binary-version"

    echo "Dist build complete:"
    echo "  Entry point: plugin/hooks/bin/htmlgraph (bootstrap)"
    echo "  Binary:      plugin/hooks/bin/htmlgraph-bin"
    echo "  Installed:   ${INSTALL_DIR}/htmlgraph (v${VERSION})"
else
    echo "Building htmlgraph (dev mode, version: ${VERSION})..."
    go build -ldflags "-s -w -X main.version=${VERSION}" \
        -o "${BIN_DIR}/htmlgraph" ./cmd/htmlgraph/
    chmod +x "${BIN_DIR}/htmlgraph"
    echo "Built: plugin/hooks/bin/htmlgraph"

    # Install to ~/.local/bin so the binary is on PATH.
    # Always copy (not symlink) so other projects on this machine get a
    # stable, intentionally-built binary rather than a live pointer into
    # the dev tree.
    INSTALL_DIR="${HOME}/.local/bin"
    META_DIR="${HOME}/.local/share/htmlgraph"
    mkdir -p "${INSTALL_DIR}" "${META_DIR}"
    rm -f "${INSTALL_DIR}/htmlgraph"  # Fresh inode avoids macOS signature cache
    cp "${BIN_DIR}/htmlgraph" "${INSTALL_DIR}/htmlgraph"
    chmod +x "${INSTALL_DIR}/htmlgraph"
    echo "${VERSION}" > "${META_DIR}/.binary-version"
    echo "Installed: ${INSTALL_DIR}/htmlgraph (v${VERSION})"
fi
