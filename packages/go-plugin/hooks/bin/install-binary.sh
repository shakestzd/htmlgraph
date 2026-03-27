#!/bin/sh
# install-binary.sh - Dev convenience: install the locally-built binary
# into the location expected by the bootstrap script.
#
# Usage (from repo root):
#   packages/go-plugin/hooks/bin/install-binary.sh
#
# This copies the locally-compiled "htmlgraph-hooks" binary to
# "htmlgraph-hooks-bin" and writes a .binary-version file so that
# the bootstrap script's version check passes immediately.

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SRC="${SCRIPT_DIR}/htmlgraph-hooks"

# Mirror bootstrap.sh: install into CLAUDE_PLUGIN_DATA so the binary persists
# across `claude plugin update`.  In dev mode CLAUDE_PLUGIN_DATA is unset, so
# fall back to the same predictable local path bootstrap.sh uses.
BINARY_DIR="${CLAUDE_PLUGIN_DATA:-${HOME}/.claude/plugins/data/htmlgraph}"
DST="${BINARY_DIR}/htmlgraph-hooks-bin"

if [ ! -f "${SRC}" ]; then
    echo "Error: ${SRC} not found. Run build.sh first." >&2
    exit 1
fi

mkdir -p "${BINARY_DIR}"
cp "${SRC}" "${DST}"
chmod +x "${DST}"

# Write version from the binary's own version command.
# Output format is "htmlgraph X.Y.Z (go)" — extract just the semver part.
RAW_VERSION="$("${DST}" version 2>/dev/null || echo 'dev')"
VERSION="$(echo "${RAW_VERSION}" | sed -n 's/.*htmlgraph[[:space:]]*\([0-9][^ ]*\).*/\1/p')"
VERSION="${VERSION:-dev}"
echo "${VERSION}" > "${BINARY_DIR}/.binary-version"

echo "Installed: ${DST}"
echo "Version:   ${VERSION}"
