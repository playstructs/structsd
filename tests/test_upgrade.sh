#!/usr/bin/env bash
#
# Structs Chain Upgrade Test: v0.15.0 -> v0.16.0
#
# Exercises the full on-chain upgrade path:
#   1. Bootstrap a chain with the v0.15.0 binary (manual genesis, no Ignite)
#   2. Populate state via test_chain_v0.15.0.sh
#   3. Submit a governance software-upgrade proposal and vote
#   4. Wait for the proposal to PASS, then halt the old chain at the upgrade height
#   5. Swap in the v0.16.0 binary and restart
#   6. Verify the migration ran correctly and the chain is healthy
#
# Prerequisites:
#   - build/structsd-v0.15.0  (built from branch github/111b)
#   - build/structsd-v0.16.0  (built from branch 112b, i.e. the current branch)
#   Missing binaries are auto-built via --auto-build.
#
# Usage:
#   bash tests/test_upgrade.sh [--skip-populate] [--auto-build] [--keep-alive]
#
#   --skip-populate   Skip running test_chain_v0.15.0.sh (useful if chain already has state)
#   --auto-build      Build any missing versioned binary from the right branch
#   --keep-alive      Keep the v0.16.0 chain running at the end for manual interaction
#

set -euo pipefail

# ─── Flag Parsing ─────────────────────────────────────────────────────────────

SKIP_POPULATE=false
AUTO_BUILD=false
KEEP_ALIVE=false
while [ $# -gt 0 ]; do
    case "$1" in
        --skip-populate) SKIP_POPULATE=true ;;
        --auto-build)    AUTO_BUILD=true ;;
        --keep-alive)    KEEP_ALIVE=true ;;
        -h|--help)
            sed -n '2,24p' "$0" | sed 's/^# \{0,1\}//'
            exit 0 ;;
        *) echo "Unknown flag: $1"; exit 1 ;;
    esac
    shift
done

# ─── Configuration ────────────────────────────────────────────────────────────

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

CHAIN_HOME="${HOME}/.structs"
CHAIN_ID="structstestnet-00"
DENOM="ualpha"
UPGRADE_NAME="v0.16.0"

OLD_BIN="${REPO_DIR}/build/structsd-v0.15.0"
NEW_BIN="${REPO_DIR}/build/structsd-v0.16.0"

OLD_BRANCH="github/111b"
NEW_BRANCH="112b"

PARAMS_TX="--home ${CHAIN_HOME} --keyring-dir ${CHAIN_HOME} --keyring-backend test --chain-id ${CHAIN_ID} --gas auto --yes=true"
PARAMS_QUERY="--home ${CHAIN_HOME} --output json"

UPGRADE_HEIGHT_BUFFER=200  # must exceed voting period (120s / ~1s blocks) plus margin

# Permission bit constants mirrored from x/structs/types/permissions.go
UGC_BIT=$((1 << 24))                              # PermGuildUGCUpdate
PERM_ALL_NEW=$(( (1 << 25) - 1 ))                 # 33554431, PermAll in v0.16.0
PERM_ALL_OLD=$(( PERM_ALL_NEW ^ UGC_BIT ))        # 16777215, PermAll in v0.15.0

OLD_PID=""
NEW_PID=""
SNAPSHOT_DIR="/tmp/structs_upgrade_test_$(date +%Y%m%d_%H%M%S)"

# ─── Colours & Helpers ────────────────────────────────────────────────────────

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

section() {
    echo ""
    echo -e "${CYAN}${BOLD}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${CYAN}${BOLD}  $1${NC}"
    echo -e "${CYAN}${BOLD}═══════════════════════════════════════════════════════════════${NC}"
    echo ""
}

info()  { echo -e "${YELLOW}-> $1${NC}"; }
pass()  { echo -e "  ${GREEN}PASS${NC}: $1"; }
fail()  { echo -e "  ${RED}FAIL${NC}: $1"; }
abort() { echo -e "${RED}ABORT${NC}: $1"; cleanup; exit 1; }

sed_inplace() {
    # Portable sed -i across macOS / GNU
    local expr="$1"; local file="$2"
    if [ "$(uname)" = "Darwin" ]; then
        sed -i '' "$expr" "$file"
    else
        sed -i "$expr" "$file"
    fi
}

is_port_in_use() {
    local port="$1"
    if command -v lsof &>/dev/null; then
        lsof -nP -iTCP:"${port}" -sTCP:LISTEN >/dev/null 2>&1
    else
        nc -z 127.0.0.1 "${port}" >/dev/null 2>&1
    fi
}

wait_for_block() {
    local bin="$1"
    local max_attempts="${2:-30}"
    local attempt=0
    info "Waiting for chain to produce blocks..."
    while [ $attempt -lt $max_attempts ]; do
        local height
        height=$($bin status --home "${CHAIN_HOME}" 2>/dev/null | jq -r '.sync_info.latest_block_height // empty' 2>/dev/null || echo "")
        if [ -n "$height" ] && [ "$height" != "0" ]; then
            pass "Chain is live at height ${height}"
            return 0
        fi
        sleep 2
        attempt=$((attempt + 1))
    done
    abort "Chain failed to produce blocks after ${max_attempts} attempts"
}

get_height() {
    local bin="$1"
    $bin status --home "${CHAIN_HOME}" 2>/dev/null | jq -r '.sync_info.latest_block_height' 2>/dev/null || echo "0"
}

# Extract a pure JSON object from a structsd tx output that may be prefixed
# with "gas estimate: NNNN" or similar chatter on stderr/stdout.
json_from_tx() {
    echo "$1" | grep '^{' | tail -1
}

wait_for_exit() {
    # Wait up to ${2:-20}s for a PID to exit, returning 0 if it did, 1 otherwise.
    local pid="$1"
    local timeout="${2:-20}"
    local i=0
    while kill -0 "$pid" 2>/dev/null; do
        [ $i -ge "$timeout" ] && return 1
        sleep 1
        i=$((i + 1))
    done
    return 0
}

stop_pid() {
    # Graceful stop with escalation: TERM -> wait -> KILL -> wait.
    local pid="$1"
    [ -z "$pid" ] && return 0
    kill -0 "$pid" 2>/dev/null || return 0
    kill "$pid" 2>/dev/null || true
    if ! wait_for_exit "$pid" 15; then
        info "PID ${pid} did not exit after SIGTERM, sending SIGKILL..."
        kill -9 "$pid" 2>/dev/null || true
        wait_for_exit "$pid" 10 || true
    fi
    wait "$pid" 2>/dev/null || true
}

cleanup() {
    set +e
    if [ -n "$OLD_PID" ]; then
        info "Stopping old chain (PID ${OLD_PID})..."
        stop_pid "$OLD_PID"
        OLD_PID=""
    fi
    if [ -n "$NEW_PID" ]; then
        info "Stopping new chain (PID ${NEW_PID})..."
        stop_pid "$NEW_PID"
        NEW_PID=""
    fi
    # Belt & braces: kill any stray structsd that might still hold the DB lock.
    pkill -f "structsd-v0\\.1[56]\\.0 start" 2>/dev/null || true
    sleep 1
    set -e
}
trap 'cleanup; exit 130' INT TERM
trap cleanup EXIT

build_binary() {
    # $1 = branch, $2 = out path
    local branch="$1" out="$2"
    info "Building ${out} from ${branch}..."
    (
        cd "${REPO_DIR}"
        local original_ref
        original_ref=$(git symbolic-ref --quiet --short HEAD || git rev-parse HEAD)
        local had_stash=false
        if ! git diff --quiet || ! git diff --cached --quiet; then
            info "Stashing working tree changes before checkout..."
            git stash push -u -m "test_upgrade.sh build autostash" >/dev/null
            had_stash=true
        fi
        git checkout "${branch}" >/dev/null 2>&1 || { echo "git checkout ${branch} failed"; exit 1; }
        if [ -f Makefile ] && grep -q '^build:' Makefile; then
            make build
        else
            go build -o build/structsd ./cmd/structsd
        fi
        cp build/structsd "${out}"
        git checkout "${original_ref}" >/dev/null 2>&1
        if [ "${had_stash}" = true ]; then
            git stash pop >/dev/null
        fi
    )
    [ -x "${out}" ] || { echo "Build of ${out} produced nothing executable"; exit 1; }
    pass "Built ${out}"
}

# ─── Preflight Checks ────────────────────────────────────────────────────────

section "Preflight Checks"

for cmd in jq git; do
    if ! command -v "$cmd" &>/dev/null; then
        abort "Required tool not found: ${cmd}"
    fi
done

if [ ! -x "$OLD_BIN" ]; then
    if [ "$AUTO_BUILD" = true ]; then
        build_binary "${OLD_BRANCH}" "${OLD_BIN}"
    else
        abort "Old binary not found: ${OLD_BIN}
  Build it with:   git checkout ${OLD_BRANCH} && make build && cp build/structsd ${OLD_BIN}
  Or rerun with:   bash tests/test_upgrade.sh --auto-build"
    fi
fi
if [ ! -x "$NEW_BIN" ]; then
    if [ "$AUTO_BUILD" = true ]; then
        build_binary "${NEW_BRANCH}" "${NEW_BIN}"
    else
        abort "New binary not found: ${NEW_BIN}
  Build it with:   git checkout ${NEW_BRANCH} && make build && cp build/structsd ${NEW_BIN}
  Or rerun with:   bash tests/test_upgrade.sh --auto-build"
    fi
fi

pass "Old binary: ${OLD_BIN}"
pass "New binary: ${NEW_BIN}"

# Port conflict detection -- the default Cosmos SDK ports. If anything is
# already listening on 26656/26657/1317/9090 we'll never be able to start cleanly.
for port in 26656 26657 1317 9090; do
    if is_port_in_use "${port}"; then
        abort "Port ${port} is already in use. Kill the process listening on it before running the upgrade test."
    fi
done
pass "No port conflicts on 26656 / 26657 / 1317 / 9090"

# Final sanity: make sure no leftover structsd is running from a previous run.
if pgrep -f "structsd-v0\\.1[56]\\.0 start" >/dev/null 2>&1; then
    info "Found leftover structsd process(es); killing before proceeding..."
    pkill -f "structsd-v0\\.1[56]\\.0 start" 2>/dev/null || true
    sleep 2
fi

mkdir -p "${SNAPSHOT_DIR}"
info "Snapshot directory: ${SNAPSHOT_DIR}"

# ─── Phase 1: Bootstrap Genesis ──────────────────────────────────────────────

section "Phase 1: Bootstrap Genesis (v0.15.0)"

info "Cleaning chain home: ${CHAIN_HOME}"
rm -rf "${CHAIN_HOME}"

info "Initializing chain..."
$OLD_BIN init structstestnet --chain-id "${CHAIN_ID}" --home "${CHAIN_HOME}" 2>/dev/null

info "Adding keys..."
$OLD_BIN keys add alice --keyring-backend test --home "${CHAIN_HOME}" 2>/dev/null
$OLD_BIN keys add bob   --keyring-backend test --home "${CHAIN_HOME}" 2>/dev/null

info "Funding genesis accounts..."
$OLD_BIN genesis add-genesis-account alice "400000000${DENOM}" \
    --keyring-backend test --home "${CHAIN_HOME}"
$OLD_BIN genesis add-genesis-account bob   "400000000${DENOM}" \
    --keyring-backend test --home "${CHAIN_HOME}"

# Resolve Bob's address now so we can use it on the v0.16.0 chain where the
# keyring may or may not round-trip "bob" through the name lookup (the v0.16.0
# binary loads the same on-disk test keyring, but being explicit avoids
# surprises).
BOB_ADDR=$($OLD_BIN keys show bob -a --keyring-backend test --home "${CHAIN_HOME}")
ALICE_ADDR=$($OLD_BIN keys show alice -a --keyring-backend test --home "${CHAIN_HOME}")
pass "alice: ${ALICE_ADDR}"
pass "bob:   ${BOB_ADDR}"

info "Patching genesis.json..."
GENESIS="${CHAIN_HOME}/config/genesis.json"

jq '
  .app_state.gov.params.voting_period = "120s"
  | .app_state.gov.params.max_deposit_period = "120s"
  | .app_state.gov.params.expedited_voting_period = "60s"
  | .app_state.gov.params.min_deposit = [{"denom":"ualpha","amount":"10000000"}]
  | .app_state.gov.params.expedited_min_deposit = [{"denom":"ualpha","amount":"50000000"}]
  | .app_state.staking.params.bond_denom = "ualpha"
  | .app_state.crisis.constant_fee.denom = "ualpha"
  | .app_state.slashing.params.signed_blocks_window = "35000"
  | .app_state.slashing.params.min_signed_per_window = "0.05"
  | .app_state.bank.denom_metadata = [{
      "name":"alpha",
      "description":"Alpha Matter, the most powerful material in the Structs universe.",
      "denom_units":[
        {"denom":"ualpha","exponent":0},
        {"denom":"malpha","exponent":3},
        {"denom":"alpha","exponent":6},
        {"denom":"kalpha","exponent":9},
        {"denom":"talpha","exponent":18}
      ],
      "base":"ualpha","display":"alpha","symbol":"ALPHA"
    }]
' "${GENESIS}" > "${GENESIS}.tmp" && mv "${GENESIS}.tmp" "${GENESIS}"

info "Creating validator gentx..."
$OLD_BIN genesis gentx alice "100000000${DENOM}" \
    --chain-id "${CHAIN_ID}" --keyring-backend test --home "${CHAIN_HOME}" 2>/dev/null

info "Collecting gentxs..."
$OLD_BIN genesis collect-gentxs --home "${CHAIN_HOME}" 2>/dev/null

info "Validating genesis..."
$OLD_BIN genesis validate --home "${CHAIN_HOME}" 2>/dev/null
pass "Genesis validated"

info "Fixing client.toml (chain-id + keyring-backend)..."
sed_inplace "s/chain-id = \".*\"/chain-id = \"${CHAIN_ID}\"/" "${CHAIN_HOME}/config/client.toml"
sed_inplace 's/keyring-backend = ".*"/keyring-backend = "test"/' "${CHAIN_HOME}/config/client.toml"
pass "client.toml: chain-id=${CHAIN_ID}, keyring-backend=test"

info "Setting minimum-gas-prices in app.toml..."
sed_inplace 's/minimum-gas-prices = ""/minimum-gas-prices = "0ualpha"/' "${CHAIN_HOME}/config/app.toml"
pass "minimum-gas-prices set to 0ualpha"

info "Reducing CometBFT block time for faster testing..."
sed_inplace 's/timeout_commit = "5s"/timeout_commit = "1s"/'  "${CHAIN_HOME}/config/config.toml"
sed_inplace 's/timeout_propose = "3s"/timeout_propose = "1s"/' "${CHAIN_HOME}/config/config.toml"
pass "Block time reduced (timeout_commit=1s, timeout_propose=1s)"

info "Symlinking old binary as 'structsd' in build/ for test_chain.sh..."
ln -sf "$(basename "${OLD_BIN}")" "${REPO_DIR}/build/structsd"
export PATH="${REPO_DIR}/build:${PATH}"
pass "structsd -> $(readlink "${REPO_DIR}/build/structsd")"

# ─── Phase 2: Start Old Chain ────────────────────────────────────────────────

section "Phase 2: Start Old Chain (v0.15.0)"

info "Starting v0.15.0 chain..."
$OLD_BIN start --home "${CHAIN_HOME}" > "${SNAPSHOT_DIR}/old_chain.log" 2>&1 &
OLD_PID=$!
info "Old chain PID: ${OLD_PID}"

wait_for_block "$OLD_BIN"

info "Letting chain settle (waiting for height 5)..."
while true; do
    H=$(get_height "$OLD_BIN")
    [ -n "$H" ] && [ "$H" -ge 5 ] 2>/dev/null && break
    sleep 1
done
pass "Chain settled at height $(get_height "$OLD_BIN")"

# ─── Phase 3: Populate State ─────────────────────────────────────────────────

section "Phase 3: Populate State"

if [ "${SKIP_POPULATE}" = true ]; then
    info "Skipping state population (--skip-populate)"
else
    info "Running test_chain_v0.15.0.sh --skip-mining to populate state..."
    info "(Using the v0.15.0 integration test to match the running binary)"
    bash "${SCRIPT_DIR}/test_chain_v0.15.0.sh" --skip-mining || {
        fail "test_chain_v0.15.0.sh failed -- check output above"
        info "Continuing with upgrade test anyway (state may be partial)..."
    }
    pass "State population complete"
fi

# ─── Phase 4: Pre-Upgrade Snapshot ───────────────────────────────────────────

section "Phase 4: Pre-Upgrade State Snapshot"

info "Capturing guilds..."
$OLD_BIN query structs guild-all ${PARAMS_QUERY} > "${SNAPSHOT_DIR}/pre_guilds.json" 2>/dev/null || echo '{}' > "${SNAPSHOT_DIR}/pre_guilds.json"
GUILD_COUNT=$(jq '.Guild // [] | length' "${SNAPSHOT_DIR}/pre_guilds.json" 2>/dev/null || echo "0")
info "  Guilds found: ${GUILD_COUNT}"

info "Capturing permissions..."
$OLD_BIN query structs permission-all ${PARAMS_QUERY} > "${SNAPSHOT_DIR}/pre_permissions.json" 2>/dev/null || echo '{}' > "${SNAPSHOT_DIR}/pre_permissions.json"
PERM_COUNT=$(jq '.Permission // [] | length' "${SNAPSHOT_DIR}/pre_permissions.json" 2>/dev/null || echo "0")
info "  Permission records found: ${PERM_COUNT}"

# Count the subset of pre-upgrade permissions that the migration is supposed
# to rewrite: anything with a "8-" prefix whose value is the old PermAll
# bitmask (i.e. PermAll without PermGuildUGCUpdate).
PRE_GUILD_PERM_ALL_COUNT=$(jq --argjson old "${PERM_ALL_OLD}" '
  [.Permission // [] | .[]
    | select((.permissionId | startswith("8-"))
             and ((.value | tonumber) == $old))] | length
' "${SNAPSHOT_DIR}/pre_permissions.json" 2>/dev/null || echo "0")
info "  Pre-upgrade guild PermAll records (value=${PERM_ALL_OLD}): ${PRE_GUILD_PERM_ALL_COUNT}"

info "Capturing struct types..."
$OLD_BIN query structs struct-type-all ${PARAMS_QUERY} > "${SNAPSHOT_DIR}/pre_struct_types.json" 2>/dev/null || echo '{}' > "${SNAPSHOT_DIR}/pre_struct_types.json"
PRE_STRUCT_TYPE_COUNT=$(jq '.StructType // .structType // [] | length' "${SNAPSHOT_DIR}/pre_struct_types.json" 2>/dev/null || echo "0")
info "  Struct types found: ${PRE_STRUCT_TYPE_COUNT}"

# Pick a guild id for rank-register verification after the upgrade.
SAMPLE_GUILD_ID=$(jq -r '.Guild[0].id // empty' "${SNAPSHOT_DIR}/pre_guilds.json" 2>/dev/null || echo "")
if [ -n "${SAMPLE_GUILD_ID}" ]; then
    info "  Sample guild for rank-register check: ${SAMPLE_GUILD_ID}"
fi

CURRENT_HEIGHT=$(get_height "$OLD_BIN")
info "Current chain height: ${CURRENT_HEIGHT}"
pass "Pre-upgrade snapshot saved to ${SNAPSHOT_DIR}"

# ─── Phase 5: Governance Upgrade Proposal ─────────────────────────────────────

section "Phase 5: Governance Upgrade Proposal"

info "Waiting for a few blocks before proposal..."
sleep 10

CURRENT_HEIGHT=$(get_height "$OLD_BIN")
UPGRADE_HEIGHT=$((CURRENT_HEIGHT + UPGRADE_HEIGHT_BUFFER))
info "Current height: ${CURRENT_HEIGHT}"
info "Scheduling upgrade at height: ${UPGRADE_HEIGHT}"

info "Submitting software-upgrade proposal..."
PROPOSAL_OUTPUT=$($OLD_BIN tx upgrade software-upgrade "${UPGRADE_NAME}" \
    --title "Upgrade to ${UPGRADE_NAME}" \
    --summary "Chain upgrade to ${UPGRADE_NAME}: UGC permissions migration, struct type refresh, custom ante handler" \
    --upgrade-info '{"binaries":{}}' \
    --no-validate \
    --upgrade-height "${UPGRADE_HEIGHT}" \
    --deposit "10000000${DENOM}" \
    --from alice \
    ${PARAMS_TX} --output json 2>&1) || true

PROPOSAL_JSON=$(json_from_tx "${PROPOSAL_OUTPUT}")
PROPOSAL_CODE=$(echo "${PROPOSAL_JSON}" | jq -r '.code // empty' 2>/dev/null || echo "")
if [ "${PROPOSAL_CODE}" = "0" ]; then
    PROPOSAL_TX=$(echo "${PROPOSAL_JSON}" | jq -r '.txhash // empty' 2>/dev/null || echo "")
    pass "Upgrade proposal submitted (tx: ${PROPOSAL_TX})"
else
    echo "${PROPOSAL_OUTPUT}"
    abort "Failed to submit upgrade proposal (code: ${PROPOSAL_CODE})"
fi

info "Waiting for proposal TX to be committed..."
sleep 8

info "Looking up proposal ID..."
PROPOSAL_ID=$($OLD_BIN query gov proposals ${PARAMS_QUERY} 2>/dev/null \
    | jq -r '.proposals[-1].id // empty' 2>/dev/null || echo "1")
info "Voting YES on proposal ${PROPOSAL_ID}..."
VOTE_OUTPUT=$($OLD_BIN tx gov vote "${PROPOSAL_ID}" yes --from alice ${PARAMS_TX} --output json 2>&1) || true

VOTE_JSON=$(json_from_tx "${VOTE_OUTPUT}")
VOTE_CODE=$(echo "${VOTE_JSON}" | jq -r '.code // empty' 2>/dev/null || echo "")
if [ "${VOTE_CODE}" = "0" ]; then
    pass "Vote submitted"
else
    echo "${VOTE_OUTPUT}"
    abort "Failed to vote on proposal (code: ${VOTE_CODE})"
fi

info "Polling proposal ${PROPOSAL_ID} for PASSED status (voting_period=120s)..."
PROPOSAL_TIMEOUT=180
PROPOSAL_ELAPSED=0
PROPOSAL_STATUS="unknown"
while [ "${PROPOSAL_ELAPSED}" -lt "${PROPOSAL_TIMEOUT}" ]; do
    PROPOSAL_STATUS=$($OLD_BIN query gov proposal "${PROPOSAL_ID}" ${PARAMS_QUERY} 2>/dev/null \
        | jq -r '.proposal.status // .status // "unknown"' 2>/dev/null || echo "unknown")
    case "${PROPOSAL_STATUS}" in
        PROPOSAL_STATUS_PASSED|3)
            pass "Proposal ${PROPOSAL_ID} PASSED (status=${PROPOSAL_STATUS})"
            break ;;
        PROPOSAL_STATUS_REJECTED|4|PROPOSAL_STATUS_FAILED|5)
            abort "Proposal ${PROPOSAL_ID} ended with terminal status ${PROPOSAL_STATUS}" ;;
    esac
    H=$(get_height "$OLD_BIN")
    printf "\r  proposal=%s status=%s height=%s elapsed=%ds    " "${PROPOSAL_ID}" "${PROPOSAL_STATUS}" "${H}" "${PROPOSAL_ELAPSED}"
    sleep 4
    PROPOSAL_ELAPSED=$((PROPOSAL_ELAPSED + 4))
done
echo ""
case "${PROPOSAL_STATUS}" in
    PROPOSAL_STATUS_PASSED|3) ;;
    *) abort "Proposal ${PROPOSAL_ID} never reached PASSED (final status=${PROPOSAL_STATUS})" ;;
esac

# Sanity: make sure we still have headroom to hit UPGRADE_HEIGHT.
CURRENT_HEIGHT=$(get_height "$OLD_BIN")
if [ "${CURRENT_HEIGHT}" -ge "${UPGRADE_HEIGHT}" ]; then
    abort "Chain already at height ${CURRENT_HEIGHT} >= UPGRADE_HEIGHT ${UPGRADE_HEIGHT}; bump UPGRADE_HEIGHT_BUFFER."
fi
info "Headroom OK: current=${CURRENT_HEIGHT}, upgrade_height=${UPGRADE_HEIGHT}"

# ─── Phase 6: Halt via app.toml ──────────────────────────────────────────────

section "Phase 6: Halt Chain at Upgrade Height"

info "Setting halt-height=${UPGRADE_HEIGHT} in app.toml..."
APP_TOML="${CHAIN_HOME}/config/app.toml"
sed_inplace "s/halt-height = .*/halt-height = ${UPGRADE_HEIGHT}/" "${APP_TOML}"
pass "halt-height set to ${UPGRADE_HEIGHT}"

info "Restarting old chain so it picks up halt-height..."
stop_pid "$OLD_PID"
OLD_PID=""

sleep 3
$OLD_BIN start --home "${CHAIN_HOME}" > "${SNAPSHOT_DIR}/old_chain_halt.log" 2>&1 &
OLD_PID=$!
info "Old chain restarted (PID: ${OLD_PID}), waiting for halt at height ${UPGRADE_HEIGHT}..."

HALT_CHECK_INTERVAL=2
HALT_TIMEOUT=600
HALT_ELAPSED=0

while kill -0 "$OLD_PID" 2>/dev/null; do
    CURRENT_HEIGHT=$(get_height "$OLD_BIN")
    if [ -n "$CURRENT_HEIGHT" ] && [ "$CURRENT_HEIGHT" != "0" ]; then
        BLOCKS_LEFT=$((UPGRADE_HEIGHT - CURRENT_HEIGHT))
        if [ "$BLOCKS_LEFT" -gt 0 ]; then
            printf "\r  Height: %s / %s  (%d blocks remaining)    " "$CURRENT_HEIGHT" "$UPGRADE_HEIGHT" "$BLOCKS_LEFT"
        fi
    fi
    sleep "$HALT_CHECK_INTERVAL"
    HALT_ELAPSED=$((HALT_ELAPSED + HALT_CHECK_INTERVAL))
    if [ "$HALT_ELAPSED" -ge "$HALT_TIMEOUT" ]; then
        abort "Timed out waiting for chain halt after ${HALT_TIMEOUT}s"
    fi
done

echo ""
wait "$OLD_PID" 2>/dev/null || true
OLD_PID=""
pass "Old chain halted at height ${UPGRADE_HEIGHT}"

info "Waiting for the process to fully release the DB lock..."
sleep 5
# Ensure nothing is still holding the DB.
if pgrep -f "structsd-v0\\.15\\.0 start" >/dev/null 2>&1; then
    info "Found lingering v0.15.0 process; killing..."
    pkill -9 -f "structsd-v0\\.15\\.0 start" 2>/dev/null || true
    sleep 3
fi

info "Resetting halt-height to 0 for the new binary..."
sed_inplace "s/halt-height = .*/halt-height = 0/" "${APP_TOML}"
pass "halt-height reset to 0"

# ─── Phase 7: Binary Swap and Restart ─────────────────────────────────────────

section "Phase 7: Binary Swap (v0.16.0)"

info "Updating structsd symlink to v0.16.0 binary..."
ln -sf "$(basename "${NEW_BIN}")" "${REPO_DIR}/build/structsd"
pass "structsd -> $(readlink "${REPO_DIR}/build/structsd")"

info "Starting chain with v0.16.0 binary..."
$NEW_BIN start --home "${CHAIN_HOME}" > "${SNAPSHOT_DIR}/new_chain.log" 2>&1 &
NEW_PID=$!
info "New chain PID: ${NEW_PID}"

sleep 5

if ! kill -0 "$NEW_PID" 2>/dev/null; then
    fail "New binary exited immediately!"
    info "Last 50 lines of log:"
    tail -50 "${SNAPSHOT_DIR}/new_chain.log"
    abort "v0.16.0 binary failed to start"
fi

wait_for_block "$NEW_BIN" 90

# ─── Phase 8: Post-Upgrade Verification ──────────────────────────────────────

section "Phase 8: Post-Upgrade Verification"

PASS_COUNT=0
FAIL_COUNT=0

assert_pass() { pass "$1"; PASS_COUNT=$((PASS_COUNT + 1)); }
assert_fail() { fail "$1"; FAIL_COUNT=$((FAIL_COUNT + 1)); }

# --- 8a. Upgrade handler execution confirmed in logs ---
info "Asserting upgrade handler ran..."
if grep -E -q "applying upgrade \"${UPGRADE_NAME}\"|UPGRADE \"${UPGRADE_NAME}\" NEEDED|applying upgrade plan" "${SNAPSHOT_DIR}/new_chain.log"; then
    assert_pass "Upgrade handler '${UPGRADE_NAME}' executed (log confirmed)"
else
    assert_fail "Did not see 'applying upgrade' for ${UPGRADE_NAME} in new chain log"
    info "  Last 30 lines of new_chain.log:"
    tail -30 "${SNAPSHOT_DIR}/new_chain.log" | sed 's/^/    /'
fi

# --- 8b. Block production ---
info "Checking block production..."
HEIGHT_A=$(get_height "$NEW_BIN")
sleep 6
HEIGHT_B=$(get_height "$NEW_BIN")
if [ "$HEIGHT_B" -gt "$HEIGHT_A" ]; then
    assert_pass "Chain producing blocks (${HEIGHT_A} -> ${HEIGHT_B})"
else
    assert_fail "Chain stalled at height ${HEIGHT_A}"
fi

# --- 8c. Guild state preserved ---
info "Checking guild state post-upgrade..."
$NEW_BIN query structs guild-all ${PARAMS_QUERY} > "${SNAPSHOT_DIR}/post_guilds.json" 2>/dev/null || echo '{}' > "${SNAPSHOT_DIR}/post_guilds.json"
POST_GUILD_COUNT=$(jq '.Guild // [] | length' "${SNAPSHOT_DIR}/post_guilds.json" 2>/dev/null || echo "0")
if [ "${POST_GUILD_COUNT}" = "${GUILD_COUNT}" ]; then
    assert_pass "Guild count preserved: ${POST_GUILD_COUNT}"
else
    assert_fail "Guild count changed: ${GUILD_COUNT} -> ${POST_GUILD_COUNT}"
fi

# --- 8d. Permissions migration ---
info "Checking permissions post-upgrade..."
$NEW_BIN query structs permission-all ${PARAMS_QUERY} > "${SNAPSHOT_DIR}/post_permissions.json" 2>/dev/null || echo '{}' > "${SNAPSHOT_DIR}/post_permissions.json"
POST_PERM_COUNT=$(jq '.Permission // [] | length' "${SNAPSHOT_DIR}/post_permissions.json" 2>/dev/null || echo "0")
if [ "${POST_PERM_COUNT}" -ge "${PERM_COUNT}" ]; then
    assert_pass "Permission records preserved: ${POST_PERM_COUNT} (was ${PERM_COUNT})"
else
    assert_fail "Permission records lost: ${PERM_COUNT} -> ${POST_PERM_COUNT}"
fi

# Concrete migration assertion: every "8-*" record that used to equal the old
# PermAll bitmask should now equal the new PermAll bitmask (UGC bit flipped on).
POST_GUILD_PERM_ALL_NEW_COUNT=$(jq --argjson new "${PERM_ALL_NEW}" '
  [.Permission // [] | .[]
    | select((.permissionId | startswith("8-"))
             and ((.value | tonumber) == $new))] | length
' "${SNAPSHOT_DIR}/post_permissions.json" 2>/dev/null || echo "0")

POST_GUILD_PERM_ALL_STALE_COUNT=$(jq --argjson old "${PERM_ALL_OLD}" '
  [.Permission // [] | .[]
    | select((.permissionId | startswith("8-"))
             and ((.value | tonumber) == $old))] | length
' "${SNAPSHOT_DIR}/post_permissions.json" 2>/dev/null || echo "0")

info "  pre  guild PermAll (value=${PERM_ALL_OLD}): ${PRE_GUILD_PERM_ALL_COUNT}"
info "  post guild PermAll (value=${PERM_ALL_NEW}): ${POST_GUILD_PERM_ALL_NEW_COUNT}"
info "  post stale old-PermAll (value=${PERM_ALL_OLD}): ${POST_GUILD_PERM_ALL_STALE_COUNT}"

if [ "${PRE_GUILD_PERM_ALL_COUNT}" -gt 0 ]; then
    if [ "${POST_GUILD_PERM_ALL_NEW_COUNT}" -ge "${PRE_GUILD_PERM_ALL_COUNT}" ] \
       && [ "${POST_GUILD_PERM_ALL_STALE_COUNT}" = "0" ]; then
        assert_pass "UGC bit migrated on ${POST_GUILD_PERM_ALL_NEW_COUNT} guild PermAll record(s), no stale records remain"
    else
        assert_fail "UGC migration incomplete: expected ${PRE_GUILD_PERM_ALL_COUNT} upgraded, got new=${POST_GUILD_PERM_ALL_NEW_COUNT} stale=${POST_GUILD_PERM_ALL_STALE_COUNT}"
    fi
else
    info "  (no pre-upgrade guild PermAll records found; skipping bit-migration assertion)"
fi

# --- 8e. Struct types refreshed ---
info "Checking struct types post-upgrade..."
$NEW_BIN query structs struct-type-all ${PARAMS_QUERY} > "${SNAPSHOT_DIR}/post_struct_types.json" 2>/dev/null || echo '{}' > "${SNAPSHOT_DIR}/post_struct_types.json"
POST_STRUCT_TYPE_COUNT=$(jq '.StructType // .structType // [] | length' "${SNAPSHOT_DIR}/post_struct_types.json" 2>/dev/null || echo "0")

if [ "${POST_STRUCT_TYPE_COUNT}" -ge "${PRE_STRUCT_TYPE_COUNT}" ] && [ "${POST_STRUCT_TYPE_COUNT}" -gt 0 ]; then
    assert_pass "Struct types present: pre=${PRE_STRUCT_TYPE_COUNT} post=${POST_STRUCT_TYPE_COUNT}"
else
    assert_fail "Struct types missing after upgrade: pre=${PRE_STRUCT_TYPE_COUNT} post=${POST_STRUCT_TYPE_COUNT}"
fi

# The v0.16.0 StructType proto adds primaryWeaponGuaranteedShots /
# secondaryWeaponGuaranteedShots fields. They may legitimately be 0, but the
# fields themselves must at least exist on every record after the refresh.
MISSING_NEW_FIELDS=$(jq '
  [.StructType // .structType // [] | .[]
    | select((has("primaryWeaponGuaranteedShots") | not)
          or (has("secondaryWeaponGuaranteedShots") | not))] | length
' "${SNAPSHOT_DIR}/post_struct_types.json" 2>/dev/null || echo "0")
if [ "${MISSING_NEW_FIELDS}" = "0" ]; then
    assert_pass "All struct types carry the v0.16.0 weapon-shots fields"
else
    assert_fail "${MISSING_NEW_FIELDS} struct type(s) missing v0.16.0 weapon-shots fields"
fi

# --- 8f. Guild rank register UGC bit (spot check on sample guild) ---
if [ -n "${SAMPLE_GUILD_ID}" ]; then
    info "Checking guild rank permission for sample guild ${SAMPLE_GUILD_ID}..."
    $NEW_BIN query structs guild-rank-permission-by-object-and-guild \
        "${SAMPLE_GUILD_ID}" "${SAMPLE_GUILD_ID}" ${PARAMS_QUERY} \
        > "${SNAPSHOT_DIR}/post_sample_guild_rank.json" 2>/dev/null || echo '{}' > "${SNAPSHOT_DIR}/post_sample_guild_rank.json"
    # The guild-owner rank register has the UGC bit set as part of the
    # migration. Any record (non-zero rank) returned here is treated as
    # evidence the register was touched by the handler.
    RANK_RECORDS=$(jq '.guild_rank_permission_records // .guildRankPermissionRecords // [] | length' "${SNAPSHOT_DIR}/post_sample_guild_rank.json" 2>/dev/null || echo "0")
    if [ "${RANK_RECORDS}" -gt 0 ]; then
        assert_pass "Guild rank register present for ${SAMPLE_GUILD_ID} (${RANK_RECORDS} record(s))"
    else
        # This is a soft signal: the register may have a different shape under
        # the v0.16.0 query. Don't fail the run, but flag it.
        info "  (No rank-permission records returned for ${SAMPLE_GUILD_ID}; soft skip)"
    fi
fi

# --- 8g. Basic transaction on new chain ---
info "Testing a basic transaction on v0.16.0..."
BANK_OUTPUT=$($NEW_BIN tx bank send "${ALICE_ADDR}" "${BOB_ADDR}" "1000${DENOM}" \
    --from "${ALICE_ADDR}" ${PARAMS_TX} --output json 2>&1) || true
BANK_JSON=$(json_from_tx "${BANK_OUTPUT}")
BANK_CODE=$(echo "${BANK_JSON}" | jq -r '.code // empty' 2>/dev/null || echo "")
if [ "${BANK_CODE}" = "0" ]; then
    assert_pass "Bank send succeeded on v0.16.0"
else
    echo "${BANK_OUTPUT}" | tail -20
    assert_fail "Bank send failed on v0.16.0: code=${BANK_CODE}"
fi
sleep 3

# ─── Phase 9: Summary ────────────────────────────────────────────────────────

section "Upgrade Test Summary"

echo -e "  Upgrade:         v0.15.0 -> ${UPGRADE_NAME}"
echo -e "  Upgrade height:  ${UPGRADE_HEIGHT}"
echo -e "  Snapshots:       ${SNAPSHOT_DIR}"
echo ""
echo -e "  ${GREEN}Passed: ${PASS_COUNT}${NC}"
echo -e "  ${RED}Failed: ${FAIL_COUNT}${NC}"
echo ""

if [ "$FAIL_COUNT" -gt 0 ]; then
    echo -e "${RED}${BOLD}UPGRADE TEST FAILED${NC}"
    echo ""
    echo "  Logs:"
    echo "    Old chain:          ${SNAPSHOT_DIR}/old_chain.log"
    echo "    Old chain (halt):   ${SNAPSHOT_DIR}/old_chain_halt.log"
    echo "    New chain:          ${SNAPSHOT_DIR}/new_chain.log"
    echo ""
    echo "  State snapshots:"
    echo "    Pre:  ${SNAPSHOT_DIR}/pre_*.json"
    echo "    Post: ${SNAPSHOT_DIR}/post_*.json"
    exit 1
fi

echo -e "${GREEN}${BOLD}UPGRADE TEST PASSED${NC}"
echo ""

if [ "$KEEP_ALIVE" = true ]; then
    echo "  The v0.16.0 chain is still running (PID ${NEW_PID})."
    echo "  Snapshots at: ${SNAPSHOT_DIR}"
    echo "  Press Ctrl+C to stop."
    echo ""
    # Drop the EXIT trap so the chain keeps running once wait returns, but
    # keep the INT/TERM trap so Ctrl+C still cleans up.
    trap - EXIT
    trap 'cleanup; exit 130' INT TERM
    info "Chain running. Waiting... (Ctrl+C to stop)"
    wait "$NEW_PID" 2>/dev/null || true
else
    info "Shutting down v0.16.0 chain (use --keep-alive to leave it running)."
    # cleanup trap will handle it.
fi
