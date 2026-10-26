#!/usr/bin/env bash
# v0.21.0 -> v0.22.0 rehearsal. Starts nodes and broadcasts transactions.
# Prepare a fresh artifact with prepare_upgrade_rehearsal.py --bootstrap first.
# See --help and docs/upgrades/v0.22.0-test-chain-audit.md for current coverage.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
exec python3 -B "${SCRIPT_DIR}/run_upgrade_rehearsal.py" "$@"
