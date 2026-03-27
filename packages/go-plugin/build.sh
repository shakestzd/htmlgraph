#!/usr/bin/env bash
# build.sh - Build the htmlgraph-hooks Go binary for the go-plugin.
#
# Usage:
#   ./build.sh          # Dev mode: binary at hooks/bin/htmlgraph-hooks
#   ./build.sh --dist   # Dist mode: binary at hooks/bin/htmlgraph-hooks-bin,
#                        #            bootstrap script at hooks/bin/htmlgraph-hooks

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
GO_DIR="${SCRIPT_DIR}/../go"
BIN_DIR="${SCRIPT_DIR}/hooks/bin"
DIST_MODE=false

for arg in "$@"; do
    case "${arg}" in
        --dist) DIST_MODE=true ;;
        *)      echo "Unknown flag: ${arg}" >&2; exit 1 ;;
    esac
done

cd "${GO_DIR}"
VERSION_RAW=$(git describe --tags --always 2>/dev/null || echo "dev")
# Strip leading 'v' for consistent version strings (goreleaser, plugin.json)
VERSION="${VERSION_RAW#v}"

if [ "${DIST_MODE}" = true ]; then
    echo "Building htmlgraph-hooks (dist mode, version: ${VERSION})..."
    go build -ldflags "-s -w -X main.version=${VERSION}" \
        -o "${BIN_DIR}/htmlgraph-hooks-bin" ./cmd/htmlgraph/
    chmod +x "${BIN_DIR}/htmlgraph-hooks-bin"

    # Copy bootstrap script as the entry point
    cp "${BIN_DIR}/bootstrap.sh" "${BIN_DIR}/htmlgraph-hooks"
    chmod +x "${BIN_DIR}/htmlgraph-hooks"

    # Write version file so bootstrap skips download
    echo "${VERSION}" > "${BIN_DIR}/.binary-version"

    echo "Dist build complete:"
    echo "  Entry point: packages/go-plugin/hooks/bin/htmlgraph-hooks (bootstrap)"
    echo "  Binary:      packages/go-plugin/hooks/bin/htmlgraph-hooks-bin"
    echo "  Version:     ${VERSION}"
else
    echo "Building htmlgraph-hooks (dev mode, version: ${VERSION})..."
    go build -ldflags "-s -w -X main.version=${VERSION}" \
        -o "${BIN_DIR}/htmlgraph-hooks" ./cmd/htmlgraph/
    chmod +x "${BIN_DIR}/htmlgraph-hooks"
    echo "Built: packages/go-plugin/hooks/bin/htmlgraph-hooks"
    ls -la "${BIN_DIR}/htmlgraph-hooks"
fi
