#!/usr/bin/env bash
# Build the v0.21.0 baseline and current v0.22.0 workspace for upgrade rehearsal.
# Does not start a node, alter chain data, or check out/stash the working tree.
set -euo pipefail

if [ "${1:-}" = "--help" ]; then
    echo "Usage: bash tests/build_upgrade_binaries.sh"
    echo "Outputs build/structsd-v0.21.0 and build/structsd-v0.22.0."
    echo "The new binary includes current workspace changes; it is not a release build."
    exit 0
fi
if [ "$#" -ne 0 ]; then
    echo "Unknown argument; use --help" >&2
    exit 1
fi

REPO_DIR="$(cd "$(dirname "$0")/.." && pwd)"
OLD_REF="v0.21.0"
OLD_COMMIT="$(git -C "${REPO_DIR}" rev-parse --verify "${OLD_REF}^{commit}")"
OUTPUT_DIR="${REPO_DIR}/build"
BUILD_TEMP="$(mktemp -d "${TMPDIR:-/tmp}/structs-upgrade-build.XXXXXX")"
WORKTREE_ADDED=false
cleanup() {
    if [ "${WORKTREE_ADDED}" = true ]; then
        # Only this script's detached, disposable checkout is removed.
        git -C "${REPO_DIR}" worktree remove --force "${BUILD_TEMP}/source" || return
    fi
    rm -rf -- "${BUILD_TEMP}"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
mkdir -p "${OUTPUT_DIR}"

git -C "${REPO_DIR}" worktree add --detach "${BUILD_TEMP}/source" "${OLD_COMMIT}"
WORKTREE_ADDED=true
# A fresh checkout can make go.mod appear newer than go.sum. Do not run the
# Makefile's tidy prerequisite: build uses -mod=readonly and must use the
# dependency files as supplied by the release/workspace.
make -C "${BUILD_TEMP}/source" -o go.sum build VERSION=v0.21.0 BUILDDIR="${BUILD_TEMP}/old"
make -C "${REPO_DIR}" -o go.sum build VERSION=v0.22.0 BUILDDIR="${BUILD_TEMP}/new"

# Publish only after both builds have succeeded. Keep versioned executables
# separate from build/structsd and from any globally installed binary.
cp "${BUILD_TEMP}/old/structsd" "${OUTPUT_DIR}/structsd-v0.21.0"
cp "${BUILD_TEMP}/new/structsd" "${OUTPUT_DIR}/structsd-v0.22.0"
{
    echo "old_ref=${OLD_REF}"
    echo "old_commit=${OLD_COMMIT}"
    echo "new_head=$(git -C "${REPO_DIR}" rev-parse HEAD)"
    echo "new_source=current workspace, including uncommitted files"
    shasum -a 256 "${OUTPUT_DIR}/structsd-v0.21.0" "${OUTPUT_DIR}/structsd-v0.22.0"
    shasum -a 256 "${REPO_DIR}/go.mod" "${REPO_DIR}/go.sum"
    go version -m "${OUTPUT_DIR}/structsd-v0.22.0"
    git -C "${REPO_DIR}" status --short
} > "${OUTPUT_DIR}/upgrade-binaries-provenance.txt"

"${OUTPUT_DIR}/structsd-v0.21.0" version
"${OUTPUT_DIR}/structsd-v0.22.0" version
echo "Built upgrade rehearsal binaries; no chain was started."
