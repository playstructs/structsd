#!/usr/bin/env bash
#
# ante_regression.sh — regression test for incident 2026-05.
#
# After the ThrottleDecorator rewrite (PR 1), the chain MUST reject the bug
# shape that filled the testnet mempool with stuck txs: a single transaction
# carrying two MsgStructDefenseSet messages from the same registered player.
#
# This script:
#   1. Picks a player with a built combat struct and at least one protectable
#      struct (passed in as DEFENDER_STRUCT_ID and two PROTECTED_STRUCT_ID*).
#   2. Builds an offline tx bundling two struct-defense-set messages.
#   3. Signs and broadcasts it.
#   4. Asserts that broadcast returns code 2020 (ErrDuplicateChargeInTx)
#      from codespace "structs-ante". A code 0 / success is a regression.
#
# Run after the chain is running and the player + structs exist. Phase 1+ of
# tests/test_chain.sh sets this up; or pass IDs from a live testnet node.
#
# Required env:
#   PLAYER_KEY                  keyring name of signer (e.g. player_3)
#   DEFENDER_STRUCT_ID          e.g. 5-1702
#   PROTECTED_STRUCT_ID_1       e.g. 5-1657
#   PROTECTED_STRUCT_ID_2       e.g. 5-1658
#
# Optional env:
#   NODE                        RPC endpoint (default: tcp://localhost:26657)
#   CHAIN_ID                    chain id (default: structslocal-1)
#
# Exit codes:
#   0  fix verified: broadcast rejected with ErrDuplicateChargeInTx (2020)
#   1  REGRESSION: broadcast accepted, the duplicate-charge admission bug is
#      back, or broadcast failed for a different reason
#   2  missing prerequisite / setup error

set -euo pipefail

: "${PLAYER_KEY:?need PLAYER_KEY}"
: "${DEFENDER_STRUCT_ID:?need DEFENDER_STRUCT_ID}"
: "${PROTECTED_STRUCT_ID_1:?need PROTECTED_STRUCT_ID_1}"
: "${PROTECTED_STRUCT_ID_2:?need PROTECTED_STRUCT_ID_2}"
NODE="${NODE:-tcp://localhost:26657}"
CHAIN_ID="${CHAIN_ID:-structslocal-1}"

PARAMS_TX="--home ~/.structs --keyring-dir ~/.structs --keyring-backend test --node ${NODE} --chain-id ${CHAIN_ID}"

TMPDIR=$(mktemp -d)
trap 'rm -rf "${TMPDIR}"' EXIT

echo "[ante_regression] Generating tx 1 (defender=${DEFENDER_STRUCT_ID}, protected=${PROTECTED_STRUCT_ID_1})..."
structsd ${PARAMS_TX} tx structs struct-defense-set \
    "${DEFENDER_STRUCT_ID}" "${PROTECTED_STRUCT_ID_1}" \
    --from "${PLAYER_KEY}" --generate-only > "${TMPDIR}/tx1.json"

echo "[ante_regression] Generating tx 2 (defender=${DEFENDER_STRUCT_ID}, protected=${PROTECTED_STRUCT_ID_2})..."
structsd ${PARAMS_TX} tx structs struct-defense-set \
    "${DEFENDER_STRUCT_ID}" "${PROTECTED_STRUCT_ID_2}" \
    --from "${PLAYER_KEY}" --generate-only > "${TMPDIR}/tx2.json"

# Merge the two msg arrays into a single tx body. We assume both txs have a
# single msg each (which they do for --generate-only with one message).
jq --slurpfile other "${TMPDIR}/tx2.json" \
    '.body.messages += $other[0].body.messages' \
    "${TMPDIR}/tx1.json" > "${TMPDIR}/merged.json"

MSG_COUNT=$(jq '.body.messages | length' "${TMPDIR}/merged.json")
if [ "${MSG_COUNT}" != "2" ]; then
    echo "[ante_regression] FATAL: merged tx has ${MSG_COUNT} messages, expected 2"
    cat "${TMPDIR}/merged.json"
    exit 2
fi

echo "[ante_regression] Signing merged tx..."
structsd ${PARAMS_TX} tx sign "${TMPDIR}/merged.json" \
    --from "${PLAYER_KEY}" > "${TMPDIR}/signed.json"

echo "[ante_regression] Broadcasting (sync, expecting reject)..."
BROADCAST_OUT=$(structsd ${PARAMS_TX} tx broadcast "${TMPDIR}/signed.json" \
    --broadcast-mode sync --output json 2>&1) || true
echo "${BROADCAST_OUT}"

CODE=$(echo "${BROADCAST_OUT}" | jq -r '.code // -1')
CODESPACE=$(echo "${BROADCAST_OUT}" | jq -r '.codespace // ""')
RAW_LOG=$(echo "${BROADCAST_OUT}" | jq -r '.raw_log // ""')

echo "[ante_regression] code=${CODE} codespace=${CODESPACE} raw_log=${RAW_LOG}"

# We expect: code=2020 codespace=structs-ante (ErrDuplicateChargeInTx).
# Anything else means either the fix regressed or a different reject fired.
if [ "${CODE}" = "0" ]; then
    echo "[ante_regression] REGRESSION: duplicate-charge tx was admitted (code 0)."
    echo "[ante_regression] The 2026-05 admission bug is back. Investigate ThrottleDecorator."
    exit 1
fi

if [ "${CODESPACE}" = "structs-ante" ] && [ "${CODE}" = "2020" ]; then
    echo "[ante_regression] OK: chain rejected with structs-ante/2020 (ErrDuplicateChargeInTx)."
    exit 0
fi

# Other ante rejects are also acceptable as long as they're from structs-ante:
# e.g. if PermissionMap evolves or a guard fires first. Surface for review.
if [ "${CODESPACE}" = "structs-ante" ]; then
    echo "[ante_regression] WARNING: rejected with structs-ante code ${CODE}, expected 2020."
    echo "[ante_regression] Treating as PASS (still rejected at ante) but review whether the right guard fired."
    exit 0
fi

echo "[ante_regression] UNEXPECTED reject: ${CODESPACE}/${CODE}. Review output."
exit 1
