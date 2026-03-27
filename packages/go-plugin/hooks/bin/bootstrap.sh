#!/bin/sh
# bootstrap.sh - Lightweight bootstrap for htmlgraph-hooks Go binary.
#
# In the distributed plugin, this script IS named "htmlgraph-hooks".
# On first run it downloads the correct platform binary from GitHub Releases,
# then exec's into it.  Subsequent runs simply exec the cached binary after
# a fast (~1 ms) version check.
#
# Design constraints:
#   - POSIX sh (no bash-isms)
#   - No dependencies beyond curl/tar (standard on macOS + Linux)
#   - Never blocks Claude Code: on error, prints {} to stdout and exits 0
#   - Stdin passthrough via exec (CloudEvent JSON piped by hooks)

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Store the binary in CLAUDE_PLUGIN_DATA so it survives `claude plugin update`.
# CLAUDE_PLUGIN_ROOT is wiped on every update; CLAUDE_PLUGIN_DATA persists at
# ~/.claude/plugins/data/{plugin-id}/.  Fall back to a predictable local path
# when the env var is absent (dev mode / manual invocation).
BINARY_DIR="${CLAUDE_PLUGIN_DATA:-${HOME}/.claude/plugins/data/htmlgraph}"
BINARY="${BINARY_DIR}/htmlgraph-hooks-bin"
VERSION_FILE="${BINARY_DIR}/.binary-version"

# ---------------------------------------------------------------------------
# Resolve expected version from plugin.json
# ---------------------------------------------------------------------------
resolve_version() {
    plugin_json=""

    # CLAUDE_PLUGIN_ROOT is set by Claude Code at hook invocation time.
    if [ -n "${CLAUDE_PLUGIN_ROOT:-}" ]; then
        plugin_json="${CLAUDE_PLUGIN_ROOT}/.claude-plugin/plugin.json"
    fi

    # Fallback: walk up from script dir (hooks/bin -> hooks -> plugin root)
    if [ -z "${plugin_json}" ] || [ ! -f "${plugin_json}" ]; then
        plugin_json="${SCRIPT_DIR}/../../.claude-plugin/plugin.json"
    fi

    if [ ! -f "${plugin_json}" ]; then
        echo ""
        return
    fi

    # Extract "version": "X.Y.Z" without jq — portable sed.
    sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "${plugin_json}" | head -1
}

# ---------------------------------------------------------------------------
# Detect OS and architecture, map to goreleaser archive names
# ---------------------------------------------------------------------------
detect_platform() {
    _os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    _arch="$(uname -m)"

    case "${_os}" in
        darwin) PLATFORM_OS="darwin" ;;
        linux)  PLATFORM_OS="linux"  ;;
        *)
            log_err "Unsupported OS: ${_os}"
            bail
            ;;
    esac

    case "${_arch}" in
        x86_64|amd64)   PLATFORM_ARCH="amd64" ;;
        arm64|aarch64)  PLATFORM_ARCH="arm64" ;;
        *)
            log_err "Unsupported architecture: ${_arch}"
            bail
            ;;
    esac
}

# ---------------------------------------------------------------------------
# Download binary from GitHub Releases
# ---------------------------------------------------------------------------
download_binary() {
    _version="$1"
    _url="https://github.com/shakestzd/htmlgraph/releases/download/go/v${_version}/htmlgraph-hooks_${_version}_${PLATFORM_OS}_${PLATFORM_ARCH}.tar.gz"

    log_err "Downloading hooks binary v${_version} for ${PLATFORM_OS}/${PLATFORM_ARCH}..."

    mkdir -p "${BINARY_DIR}"
    _tmpdir="$(mktemp -d)"
    _tarball="${_tmpdir}/htmlgraph-hooks.tar.gz"

    # Try curl first (available on macOS + most Linux), fall back to wget.
    if command -v curl >/dev/null 2>&1; then
        if ! curl -fsSL -o "${_tarball}" "${_url}" 2>/dev/null; then
            rm -rf "${_tmpdir}"
            log_err "Download failed (curl): ${_url}"
            bail
        fi
    elif command -v wget >/dev/null 2>&1; then
        if ! wget -q -O "${_tarball}" "${_url}" 2>/dev/null; then
            rm -rf "${_tmpdir}"
            log_err "Download failed (wget): ${_url}"
            bail
        fi
    else
        rm -rf "${_tmpdir}"
        log_err "Neither curl nor wget found. Cannot download binary."
        bail
    fi

    # Extract — tar.gz contains the binary named "htmlgraph-hooks"
    if ! tar xzf "${_tarball}" -C "${_tmpdir}" 2>/dev/null; then
        rm -rf "${_tmpdir}"
        log_err "Failed to extract archive."
        bail
    fi

    # Move extracted binary into place
    if [ -f "${_tmpdir}/htmlgraph-hooks" ]; then
        mv "${_tmpdir}/htmlgraph-hooks" "${BINARY}"
    else
        rm -rf "${_tmpdir}"
        log_err "Binary not found in archive."
        bail
    fi

    chmod +x "${BINARY}"
    echo "${_version}" > "${VERSION_FILE}"

    rm -rf "${_tmpdir}"
    log_err "Installed hooks binary v${_version}."
}

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
log_err() {
    echo "[htmlgraph] $*" >&2
}

# bail outputs {} to stdout (so Claude Code sees valid JSON) and exits 0.
bail() {
    echo "{}"
    exit 0
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
EXPECTED_VERSION="$(resolve_version)"

if [ -z "${EXPECTED_VERSION}" ]; then
    log_err "Could not determine expected version from plugin.json."
    bail
fi

# Fast path: binary exists and version matches.
if [ -x "${BINARY}" ] && [ -f "${VERSION_FILE}" ]; then
    CACHED_VERSION="$(cat "${VERSION_FILE}" 2>/dev/null || echo "")"
    if [ "${CACHED_VERSION}" = "${EXPECTED_VERSION}" ]; then
        exec "${BINARY}" "$@"
    fi
fi

# Slow path: download or update.
detect_platform
download_binary "${EXPECTED_VERSION}"

# Now exec the freshly downloaded binary.
if [ -x "${BINARY}" ]; then
    exec "${BINARY}" "$@"
fi

# Should not reach here, but handle gracefully.
log_err "Binary not executable after download."
bail
