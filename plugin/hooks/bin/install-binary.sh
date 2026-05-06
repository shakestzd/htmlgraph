#!/bin/sh
# install-binary.sh - Dev convenience: install the locally-built binary
# into ~/.local/bin (the canonical install location shared by bootstrap,
# curl install script, Homebrew, and setup-cli).
#
# Usage (from repo root):
#   packages/go-plugin/hooks/bin/install-binary.sh
#
# Copies the locally-compiled binary to ~/.local/bin/wipnote and writes
# a version file so the bootstrap script's version check passes immediately.
# Also creates a `wn` symlink as the short alias.

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SRC="${SCRIPT_DIR}/wipnote"

INSTALL_DIR="${HOME}/.local/bin"
DST="${INSTALL_DIR}/wipnote"
ALIAS_DST="${INSTALL_DIR}/wn"
META_DIR="${HOME}/.local/share/wipnote"

if [ ! -f "${SRC}" ]; then
    echo "Error: ${SRC} not found. Run build.sh first." >&2
    exit 1
fi

mkdir -p "${INSTALL_DIR}"
mkdir -p "${META_DIR}"
cp "${SRC}" "${DST}"
chmod +x "${DST}"

# Short alias `wn` -> `wipnote`
ln -sfn "${DST}" "${ALIAS_DST}"

# Write version from the binary's own version command.
# Output format is "wipnote X.Y.Z (go)" — extract just the semver part.
RAW_VERSION="$("${DST}" version 2>/dev/null || echo 'dev')"
VERSION="$(echo "${RAW_VERSION}" | sed -n 's/.*wipnote[[:space:]]*\([0-9][^ ]*\).*/\1/p')"
VERSION="${VERSION:-dev}"
echo "${VERSION}" > "${META_DIR}/.binary-version"

echo "Installed: ${DST}"
echo "Alias:     ${ALIAS_DST} -> wipnote"
echo "Version:   ${VERSION}"

# Check if ~/.local/bin is in PATH
case ":${PATH}:" in
    *":${INSTALL_DIR}:"*) ;;
    *)
        echo ""
        echo "NOTE: ${INSTALL_DIR} is not in your PATH."
        echo "Add this to your shell profile (~/.zshrc or ~/.bashrc):"
        echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
        ;;
esac
