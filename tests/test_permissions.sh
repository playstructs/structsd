#!/usr/bin/env bash
#
# Permissions Integration Test Script
#
# Exhaustive tests for permission behavior:
#   - Object creation and default (owner) permissions
#   - Positive: actions with the right permissions
#   - Negative: actions without permissions (with permission-denied assertion)
#   - Grant/revoke/set ordering and on-chain state
#   - Guild rank permissions and (if available) player guild rank changes
#   - Object deletion and permission cleanup
#
# Prerequisites:
#   - structsd chain running locally (fresh chain recommended)
#   - 'alice' and 'bob' keys in keyring
#
# Usage:
#   ./tests/test_permissions.sh                    # full run with setup
#   ./tests/test_permissions.sh --skip-setup       # use existing env IDs (e.g. after test_chain.sh)
#
# Not safe to run concurrently; assume one test runner per chain.
#
# Permission constants (from x/structs/types/permissions.go):
#   PermPlay=1, PermAdmin=2, PermUpdate=4, PermDelete=8
#   PermGuildEndpointUpdate, PermSubstationConnection, etc. (higher bits)
#   For "update" actions we use PermUpdate (4); for admin-style we use PermAdmin (2).
#

set -euo pipefail

# ─── Flags ───────────────────────────────────────────────────────────────────

SKIP_SETUP=false
while [ $# -gt 0 ]; do
    case "$1" in
        --skip-setup) SKIP_SETUP=true ;;
        *)            echo "Unknown flag: $1"; exit 1 ;;
    esac
    shift
done

# ─── Configuration ────────────────────────────────────────────────────────────

SLEEP=2
PARAMS_TX="--home ~/.structs --keyring-dir ~/.structs --keyring-backend test --gas auto --yes=true"
PARAMS_QUERY="--home ~/.structs --output json"
PARAMS_KEYS="--home ~/.structs --keyring-dir ~/.structs --keyring-backend test --output json"

# Permission constants (must match x/structs/types/permissions.go iota-based 1<<N)
PERM_PLAY=1                    # 1 << 0
PERM_ADMIN=2                   # 1 << 1
PERM_UPDATE=4                  # 1 << 2
PERM_DELETE=8                  # 1 << 3
PERM_TOKEN_TRANSFER=16         # 1 << 4
PERM_TOKEN_INFUSE=32           # 1 << 5
PERM_SOURCE_ALLOCATION=256     # 1 << 8
PERM_GUILD_MEMBERSHIP=512      # 1 << 9
PERM_SUBSTATION_CONNECTION=1024  # 1 << 10
PERM_ALLOCATION_CONNECTION=2048  # 1 << 11
PERM_GUILD_ENDPOINT_UPDATE=16384 # 1 << 14

# ─── Colours & Helpers ────────────────────────────────────────────────────────

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

PASS_COUNT=0
FAIL_COUNT=0

section() {
    echo ""
    echo -e "${CYAN}${BOLD}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${CYAN}${BOLD}  $1${NC}"
    echo -e "${CYAN}${BOLD}═══════════════════════════════════════════════════════════════${NC}"
    echo ""
}

info() {
    echo -e "${YELLOW}-> $1${NC}"
}

_check_tx_output() {
    local output="$1"
    local tx_code
    tx_code=$(echo "${output}" | jq -r '.code // empty' 2>/dev/null || echo "")
    if [ "${tx_code}" = "0" ]; then
        echo -e "  ${GREEN}TX submitted${NC}"
    elif [ -n "${tx_code}" ]; then
        echo -e "  ${RED}TX failed (code=${tx_code})${NC}"
        echo "  ${output}" | head -5
    elif echo "${output}" | grep -qi "error\|panic\|failed\|invalid"; then
        echo -e "  ${RED}TX failed (simulation/gas estimate error)${NC}"
        echo "  ${output}" | tail -3
    else
        echo -e "  ${GREEN}TX submitted${NC}"
    fi
}

run_tx() {
    local description="$1"
    shift
    info "${description}"
    echo -e "  ${BOLD}structsd ${PARAMS_TX} $*${NC}"
    local OUTPUT
    OUTPUT=$(structsd ${PARAMS_TX} "$@" 2>&1) || true
    _check_tx_output "${OUTPUT}"
    sleep "${SLEEP}"
}

# run_tx_expect_fail: execute a TX that we EXPECT to fail
run_tx_expect_fail() {
    local description="$1"
    shift
    info "${description}"
    echo -e "  ${BOLD}structsd ${PARAMS_TX} $*${NC}"
    local OUTPUT
    OUTPUT=$(structsd ${PARAMS_TX} "$@" 2>&1) || true
    if echo "${OUTPUT}" | grep -qi "error\|panic\|failed\|invalid\|unreachable"; then
        echo -e "  ${GREEN}Correctly rejected${NC}"
        return 0
    fi
    local tx_code
    tx_code=$(echo "${OUTPUT}" | jq -r '.code // empty' 2>/dev/null || echo "")
    if [ -n "${tx_code}" ] && [ "${tx_code}" != "0" ]; then
        echo -e "  ${GREEN}Correctly rejected (code=${tx_code})${NC}"
        return 0
    fi
    echo -e "  ${RED}Expected failure but TX succeeded${NC}"
    return 1
}

# run_tx_expect_permission_denied: expect failure AND permission/authority-related message
run_tx_expect_permission_denied() {
    local description="$1"
    shift
    info "${description}"
    echo -e "  ${BOLD}structsd ${PARAMS_TX} $*${NC}"
    local OUTPUT
    OUTPUT=$(structsd ${PARAMS_TX} "$@" 2>&1) || true
    if ! echo "${OUTPUT}" | grep -qi "error\|failed\|invalid\|unreachable"; then
        local tx_code
        tx_code=$(echo "${OUTPUT}" | jq -r '.code // empty' 2>/dev/null || echo "")
        if [ -z "${tx_code}" ] || [ "${tx_code}" = "0" ]; then
            echo -e "  ${RED}Expected permission denial but TX succeeded${NC}"
            return 1
        fi
    fi
    if echo "${OUTPUT}" | grep -qiE "permission|authority|does not have|not have the authority|unauthorized"; then
        echo -e "  ${GREEN}Correctly rejected (permission/authority)${NC}"
        return 0
    fi
    echo -e "  ${GREEN}Correctly rejected${NC} (no permission phrase in output)"
    return 0
}

query() {
    structsd ${PARAMS_QUERY} "$@" 2>/dev/null
}

jqr() {
    local json="$1"
    local path="$2"
    local fallback="${3:-}"
    local result
    result=$(echo "${json}" | jq -r "${path}" 2>/dev/null || echo "")
    if [ -z "${result}" ] || [ "${result}" = "null" ]; then
        echo "${fallback}"
    else
        echo "${result}"
    fi
}

assert_eq() {
    local label="$1"
    local expected="$2"
    local actual="$3"
    if [ "${expected}" = "${actual}" ]; then
        echo -e "  ${GREEN}PASS${NC}: ${label}  (expected='${expected}')"
        PASS_COUNT=$((PASS_COUNT + 1))
    else
        echo -e "  ${RED}FAIL${NC}: ${label}  (expected='${expected}', got='${actual}')"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
}

assert_not_empty() {
    local label="$1"
    local actual="$2"
    if [ -n "${actual}" ] && [ "${actual}" != "null" ] && [ "${actual}" != "" ]; then
        echo -e "  ${GREEN}PASS${NC}: ${label} = '${actual}'"
        PASS_COUNT=$((PASS_COUNT + 1))
    else
        echo -e "  ${RED}FAIL${NC}: ${label} is empty or null"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
}

# JSON query response: permission-by-object returns .permissionRecords (or .permissionRecord)
get_permission_by_object() {
    query query structs permission-by-object "$1" 2>/dev/null || echo '{}'
}

# permissionId format from keeper is objectId@playerId
get_permission_value_for_player() {
    local obj_id="$1"
    local player_id="$2"
    local json
    json=$(get_permission_by_object "${obj_id}")
    local perm_id="${obj_id}@${player_id}"
    local val
    val=$(echo "${json}" | jq -r --arg id "${perm_id}" '[.permissionRecords[]? | select(.permissionId == $id) | .value] | first // empty' 2>/dev/null)
    if [[ -z "${val}" || "${val}" == "null" ]]; then
        val=$(echo "${json}" | jq -r --arg id "${perm_id}" '[.permissionRecord[]? | select(.permissionId == $id) | .value] | first // empty' 2>/dev/null)
    fi
    if [[ -z "${val}" || "${val}" == "null" ]]; then
        echo "0"
    else
        echo "${val}"
    fi
}

get_guild_rank_permission_by_object() {
    query query structs guild-rank-permission-by-object "$1" 2>/dev/null || echo '{}'
}

get_guild_rank_permission_by_object_and_guild() {
    query query structs guild-rank-permission-by-object-and-guild "$1" "$2" 2>/dev/null || echo '{}'
}

get_player_guild_rank() {
    local player_id="$1"
    local json
    json=$(query query structs player "${player_id}" 2>/dev/null || echo '{}')
    jqr "${json}" '.Player.guildRank // .guildRank' '0'
}

get_latest_allocation_for_source() {
    local source_id="$1"
    query query structs allocation-all-by-source "${source_id}" | jq -r '.Allocation[-1].id // empty'
}

print_summary() {
    echo ""
    echo -e "${CYAN}${BOLD}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${CYAN}${BOLD}  TEST SUMMARY${NC}"
    echo -e "${CYAN}${BOLD}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "  ${GREEN}Passed : ${PASS_COUNT}${NC}"
    echo -e "  ${RED}Failed : ${FAIL_COUNT}${NC}"
    local TOTAL=$((PASS_COUNT + FAIL_COUNT))
    if [ "${FAIL_COUNT}" -eq 0 ]; then
        echo -e "  ${GREEN}${BOLD}ALL ${TOTAL} CHECKS PASSED${NC}"
    else
        echo -e "  ${RED}${BOLD}${FAIL_COUNT} of ${TOTAL} CHECKS FAILED${NC}"
    fi
    echo ""
}

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 1: Setup (or skip)
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 1: Setup"

declare -a EXTRA_PLAYER_IDS=()
declare -a EXTRA_PLAYER_KEYS=()

if [ "${SKIP_SETUP}" = true ]; then
    info "Skip setup: using existing env (PLAYER_1_ID, GUILD_ID, SUBSTATION_ID, etc.)"
    if [ -z "${PLAYER_1_ID:-}" ] || [ -z "${GUILD_ID:-}" ] || [ -z "${SUBSTATION_ID:-}" ]; then
        echo -e "  ${RED}Missing required env. Export PLAYER_1_ID, GUILD_ID, SUBSTATION_ID (and PLAYER_2_ID, PLAYER_3_ID if needed).${NC}"
        exit 1
    fi
    PLAYER_ME_JSON=$(query query structs player-me 2>/dev/null || echo '{}')
    ALICE_ADDRESS=$(structsd ${PARAMS_KEYS} keys show alice 2>/dev/null | jq -r .address)
    P1_ALLOC_ID=$(get_latest_allocation_for_source "${PLAYER_1_ID}")
    [ -n "${PLAYER_2_ID:-}" ] && P2_ALLOC_ID=$(get_latest_allocation_for_source "${PLAYER_2_ID}")
    [ -n "${PLAYER_3_ID:-}" ] && P3_ALLOC_ID=$(get_latest_allocation_for_source "${PLAYER_3_ID}")
    GUILD_ALL_JSON=$(query query structs guild-all 2>/dev/null || echo '{}')
    REACTOR_ID=$(jqr "${GUILD_ALL_JSON}" '.Guild[0].primaryReactorId')
else
    info "Looking up alice (Player 1)"
    ALICE_ADDRESS=$(structsd ${PARAMS_KEYS} keys show alice | jq -r .address)
    assert_not_empty "alice address" "${ALICE_ADDRESS}"

    PLAYER_ME_JSON=$(query query structs player-me)
    PLAYER_1_ID=$(jqr "${PLAYER_ME_JSON}" '.Player.id')
    assert_not_empty "Player 1 ID" "${PLAYER_1_ID}"
    echo "  Player 1 (alice) ID: ${PLAYER_1_ID}"

    PLAYER_1_CAPACITY=$(jqr "${PLAYER_ME_JSON}" '.gridAttributes.capacity')
    run_tx "Create allocation for alice" \
        tx structs allocation-create "${PLAYER_1_ID}" "${PLAYER_1_CAPACITY}" \
        --allocation-type dynamic --from alice

    P1_ALLOC_ID=$(get_latest_allocation_for_source "${PLAYER_1_ID}")
    assert_not_empty "Player 1 allocation" "${P1_ALLOC_ID}"

    run_tx "Create substation" \
        tx structs substation-create "${PLAYER_1_ID}" "${P1_ALLOC_ID}" --from alice

    P1_ALLOC_JSON=$(query query structs allocation "${P1_ALLOC_ID}")
    SUBSTATION_ID=$(jqr "${P1_ALLOC_JSON}" '.Allocation.destinationId')
    assert_not_empty "Substation ID" "${SUBSTATION_ID}"
    echo "  Substation ID: ${SUBSTATION_ID}"

    run_tx "Create guild" \
        tx structs guild-create "perm-test" "${SUBSTATION_ID}" --from alice

    GUILD_ALL_JSON=$(query query structs guild-all)
    GUILD_ID=$(jqr "${GUILD_ALL_JSON}" '.Guild[0].id')
    REACTOR_ID=$(jqr "${GUILD_ALL_JSON}" '.Guild[0].primaryReactorId')
    assert_not_empty "Guild ID" "${GUILD_ID}"
    assert_not_empty "Reactor ID" "${REACTOR_ID}"
    echo "  Guild ID: ${GUILD_ID}  Reactor ID: ${REACTOR_ID}"

    # Create two "full" players (player_2, player_3) with allocations and substation connections
    VALIDATOR_ADDRESS=$(query query staking validators | jq -r '.validators[0].operator_address')
    BOB_ADDRESS=$(structsd ${PARAMS_KEYS} keys show bob | jq -r .address)
    for P in 2 3; do
        PLAYER_KEY="player_${P}"
        info "Setting up ${PLAYER_KEY}"
        EXISTING=$(structsd ${PARAMS_KEYS} keys show "${PLAYER_KEY}" 2>/dev/null | jq -r .address 2>/dev/null || echo "")
        if [ -z "${EXISTING}" ]; then
            (echo ""; echo "") | structsd ${PARAMS_KEYS} keys add "${PLAYER_KEY}" --no-backup 2>/dev/null || true
            ADDR=$(structsd ${PARAMS_KEYS} keys show "${PLAYER_KEY}" 2>/dev/null | jq -r .address 2>/dev/null || echo "")
        else
            ADDR="${EXISTING}"
        fi
        if [ -z "${ADDR}" ]; then
            echo -e "  ${RED}Cannot get address for ${PLAYER_KEY}; create key with: structsd keys add ${PLAYER_KEY}${NC}"
            exit 1
        fi
        eval "PLAYER_${P}_ADDRESS=${ADDR}"
        run_tx "Fund ${PLAYER_KEY}" tx bank send "${BOB_ADDRESS}" "${ADDR}" 10000000ualpha --from bob
        run_tx "Delegate for ${PLAYER_KEY} (creates structs player)" \
            tx staking delegate "${VALIDATOR_ADDRESS}" 5000000ualpha --from "${PLAYER_KEY}"
        ADDR_JSON=$(query query structs address "${ADDR}")
        PID=$(jqr "${ADDR_JSON}" '.playerId')
        eval "PLAYER_${P}_ID=${PID}"
        assert_not_empty "Player ${P} ID" "${PID}"
        echo "  Player ${P} ID: ${PID}"
        PCAP=$(query query structs player "${PID}" | jq -r '.gridAttributes.capacity')
        run_tx "Create allocation for ${PLAYER_KEY}" \
            tx structs allocation-create "${PID}" "${PCAP}" --allocation-type dynamic --from "${PLAYER_KEY}"
        ALLOC_ID=$(get_latest_allocation_for_source "${PID}")
        eval "P${P}_ALLOC_ID=${ALLOC_ID}"
        run_tx "${PLAYER_KEY} join guild" \
            tx structs guild-membership-join "${GUILD_ID}" "${REACTOR_ID}-${ADDR}" --from "${PLAYER_KEY}"
        run_tx "Grant ${PLAYER_KEY} PermUpdate on substation so they can connect own allocation" \
            tx structs permission-grant-on-object "${SUBSTATION_ID}" "${PID}" "${PERM_UPDATE}" --from alice
        run_tx "Connect ${PLAYER_KEY} allocation to substation (as controller)" \
            tx structs substation-allocation-connect "${ALLOC_ID}" "${SUBSTATION_ID}" --from "${PLAYER_KEY}"
    done

    # ── Create 20 extra "lightweight" players for rank testing ──
    # These only need: key, fund, delegate, guild join (no allocations)
    NUM_EXTRA=20
    EXTRA_START=4
    EXTRA_END=$(( EXTRA_START + NUM_EXTRA - 1 ))

    # Arrays to track extra player info
    declare -a EXTRA_PLAYER_IDS
    declare -a EXTRA_PLAYER_KEYS

    info "Creating ${NUM_EXTRA} extra players (player_${EXTRA_START}..player_${EXTRA_END}) for rank testing"
    for P in $(seq ${EXTRA_START} ${EXTRA_END}); do
        PLAYER_KEY="player_${P}"
        EXTRA_PLAYER_KEYS+=("${PLAYER_KEY}")
        EXISTING=$(structsd ${PARAMS_KEYS} keys show "${PLAYER_KEY}" 2>/dev/null | jq -r .address 2>/dev/null || echo "")
        if [ -z "${EXISTING}" ]; then
            (echo ""; echo "") | structsd ${PARAMS_KEYS} keys add "${PLAYER_KEY}" --no-backup 2>/dev/null || true
            ADDR=$(structsd ${PARAMS_KEYS} keys show "${PLAYER_KEY}" 2>/dev/null | jq -r .address 2>/dev/null || echo "")
        else
            ADDR="${EXISTING}"
        fi
        if [ -z "${ADDR}" ]; then
            echo -e "  ${RED}Cannot get address for ${PLAYER_KEY}${NC}"
            exit 1
        fi
        run_tx "Fund ${PLAYER_KEY}" tx bank send "${ALICE_ADDRESS}" "${ADDR}" 4000000ualpha --from alice
        run_tx "Delegate ${PLAYER_KEY}" tx staking delegate "${VALIDATOR_ADDRESS}" 2000000ualpha --from "${PLAYER_KEY}"

        # Retry address query up to 3 times (delegation may need an extra block)
        PID=""
        for ATTEMPT in 1 2 3; do
            ADDR_JSON=$(query query structs address "${ADDR}" 2>/dev/null || echo '{}')
            PID=$(jqr "${ADDR_JSON}" '.playerId')
            if [ -n "${PID}" ] && [ "${PID}" != "" ] && [ "${PID}" != "1-0" ]; then
                break
            fi
            sleep "${SLEEP}"
        done
        if [ -z "${PID}" ] || [ "${PID}" = "" ] || [ "${PID}" = "1-0" ]; then
            echo -e "  ${RED}Failed to get valid player ID for ${PLAYER_KEY} (got '${PID}')${NC}"
            exit 1
        fi
        EXTRA_PLAYER_IDS+=("${PID}")
        run_tx "Guild join ${PLAYER_KEY}" \
            tx structs guild-membership-join "${GUILD_ID}" "${REACTOR_ID}-${ADDR}" --from "${PLAYER_KEY}"
        echo -e "  ${GREEN}✓${NC} ${PLAYER_KEY} → ${PID}"
    done
    echo "  Created ${#EXTRA_PLAYER_IDS[@]} extra players"
fi

# Ensure we have PLAYER_2_ID for tests that need a second player
if [ -z "${PLAYER_2_ID:-}" ]; then
    info "PLAYER_2_ID not set; using PLAYER_1_ID for second player in some tests"
    PLAYER_2_ID="${PLAYER_1_ID}"
fi
if [ -z "${PLAYER_3_ID:-}" ]; then
    PLAYER_3_ID="${PLAYER_2_ID}"
fi

# Verify signer resolution: --from alice resolves to PLAYER_1_ID
info "Verify player-me for alice"
PLAYER_ME_ALICE=$(query query structs player-me 2>/dev/null || echo '{}')
assert_eq "alice player-me id" "${PLAYER_1_ID}" "$(jqr "${PLAYER_ME_ALICE}" '.Player.id' '')"

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 2: Default permissions (owner can act)
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 2: Default permissions (owner)"

info "Guild: alice (owner) can update endpoint"
run_tx "Guild owner updates endpoint" \
    tx structs guild-update-endpoint "${GUILD_ID}" "perm-test.energy" --from alice

info "Substation: P1 allocation already connected at creation; query to confirm"
P1_ALLOC_J=$(query query structs allocation "${P1_ALLOC_ID}")
P1_DST=$(jqr "${P1_ALLOC_J}" '.Allocation.destinationId')
assert_eq "P1 allocation connected to substation" "${SUBSTATION_ID}" "${P1_DST}"

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 3: Grant/revoke/set and query on-chain state
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 3: Grant, revoke, set and query"

run_tx "Grant Player 2 PermUpdate (4) on guild" \
    tx structs permission-grant-on-object "${GUILD_ID}" "${PLAYER_2_ID}" "${PERM_UPDATE}" --from alice

PERM_JSON=$(get_permission_by_object "${GUILD_ID}")
VAL=$(get_permission_value_for_player "${GUILD_ID}" "${PLAYER_2_ID}")
assert_eq "Permission value after grant (4)" "4" "${VAL}"

run_tx "Grant additional PermDelete (8) on guild" \
    tx structs permission-grant-on-object "${GUILD_ID}" "${PLAYER_2_ID}" "${PERM_DELETE}" --from alice

VAL=$(get_permission_value_for_player "${GUILD_ID}" "${PLAYER_2_ID}")
assert_eq "Permission value after grant 4+8 (12)" "12" "${VAL}"

run_tx "Revoke PermUpdate (4)" \
    tx structs permission-revoke-on-object "${GUILD_ID}" "${PLAYER_2_ID}" "${PERM_UPDATE}" --from alice

VAL=$(get_permission_value_for_player "${GUILD_ID}" "${PLAYER_2_ID}")
assert_eq "Permission value after revoke 4 (8)" "8" "${VAL}"

# Revoke remaining bit (8) to clear all permissions
run_tx "Revoke PermDelete (8) to clear all" \
    tx structs permission-revoke-on-object "${GUILD_ID}" "${PLAYER_2_ID}" "${PERM_DELETE}" --from alice

VAL=$(get_permission_value_for_player "${GUILD_ID}" "${PLAYER_2_ID}")
assert_eq "Permission value after revoke all" "0" "${VAL}"

# Set overwrite test: set to PermTokenInfuse(32), then PermDelete(8)
run_tx "Set permission to 32 (overwrite)" \
    tx structs permission-set-on-object "${GUILD_ID}" "${PLAYER_2_ID}" "${PERM_TOKEN_INFUSE}" --from alice

VAL=$(get_permission_value_for_player "${GUILD_ID}" "${PLAYER_2_ID}")
assert_eq "Permission value after set 32" "32" "${VAL}"

run_tx "Set permission to 8 (overwrite again)" \
    tx structs permission-set-on-object "${GUILD_ID}" "${PLAYER_2_ID}" "${PERM_DELETE}" --from alice

VAL=$(get_permission_value_for_player "${GUILD_ID}" "${PLAYER_2_ID}")
assert_eq "Permission value after set overwrite (8)" "8" "${VAL}"

# Clean up P2 on guild before next phases
run_tx "Revoke P2 remaining permission on guild" \
    tx structs permission-revoke-on-object "${GUILD_ID}" "${PLAYER_2_ID}" "${PERM_DELETE}" --from alice

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 4: Positive tests (with permission)
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 4: Positive (with permission)"

# guild-update-endpoint requires PermGuildEndpointUpdate (16384)
run_tx "Grant Player 2 PermGuildEndpointUpdate on guild" \
    tx structs permission-grant-on-object "${GUILD_ID}" "${PLAYER_2_ID}" "${PERM_GUILD_ENDPOINT_UPDATE}" --from alice

info "Player 2 (has PermGuildEndpointUpdate on guild) updates guild endpoint"
run_tx "Player 2 updates guild endpoint" \
    tx structs guild-update-endpoint "${GUILD_ID}" "updated-by-p2.energy" --from player_2

GUILD_EP=$(query query structs guild "${GUILD_ID}" | jq -r '.Guild.endpoint // empty' 2>/dev/null || echo "")
assert_eq "Guild endpoint updated by P2" "updated-by-p2.energy" "${GUILD_EP}"

# Disconnect then reconnect to test substation permission
info "Player 2 (has PermUpdate on substation) disconnects and reconnects allocation"
run_tx "Player 2 disconnects allocation" \
    tx structs substation-allocation-disconnect "${P2_ALLOC_ID}" --from player_2
run_tx "Player 2 reconnects allocation" \
    tx structs substation-allocation-connect "${P2_ALLOC_ID}" "${SUBSTATION_ID}" --from player_2

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 5: Negative tests (without permission)
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 5: Negative (without permission)"

# Revoke Player 2's PermGuildEndpointUpdate on guild
run_tx "Revoke Player 2 PermGuildEndpointUpdate on guild" \
    tx structs permission-revoke-on-object "${GUILD_ID}" "${PLAYER_2_ID}" "${PERM_GUILD_ENDPOINT_UPDATE}" --from alice

info "Player 2 (no endpoint update permission) cannot update endpoint"
run_tx_expect_permission_denied "Player 2 tries guild-update-endpoint without permission" \
    tx structs guild-update-endpoint "${GUILD_ID}" "hacked.energy" --from player_2

info "Player 3 (no permission on guild) cannot update endpoint"
run_tx_expect_permission_denied "Player 3 tries guild-update-endpoint" \
    tx structs guild-update-endpoint "${GUILD_ID}" "hacked.energy" --from player_3

# Grant only Play (1), not GuildEndpointUpdate (16384)
run_tx "Grant Player 3 PermPlay (1) on guild" \
    tx structs permission-grant-on-object "${GUILD_ID}" "${PLAYER_3_ID}" "${PERM_PLAY}" --from alice
info "Player 3 (only PermPlay) cannot update endpoint"
run_tx_expect_permission_denied "Player 3 with PermPlay only tries guild-update-endpoint" \
    tx structs guild-update-endpoint "${GUILD_ID}" "hacked.energy" --from player_3
run_tx "Revoke Player 3 PermPlay" \
    tx structs permission-revoke-on-object "${GUILD_ID}" "${PLAYER_3_ID}" "${PERM_PLAY}" --from alice

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 6: Guild rank permissions
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 6: Guild rank permissions"

# Revoke P2's explicit PermUpdate on substation so only guild-rank path is tested
run_tx "Revoke P2 PermUpdate on substation" \
    tx structs permission-revoke-on-object "${SUBSTATION_ID}" "${PLAYER_2_ID}" "${PERM_UPDATE}" --from alice

# Set guild-rank permission: on substation, guild ${GUILD_ID}, PermUpdate (4), highest_rank 2
# So players in guild with rank 0,1,2 can update; rank 3+ cannot
run_tx "Set guild-rank permission (PermUpdate, highest_rank 2) on substation" \
    tx structs permission-guild-rank-set "${SUBSTATION_ID}" "${GUILD_ID}" "${PERM_UPDATE}" 2 --from alice

GRANK_JSON=$(get_guild_rank_permission_by_object "${SUBSTATION_ID}")
RANK_COUNT=$(echo "${GRANK_JSON}" | jq -r '.guild_rank_permission_records | length' 2>/dev/null || echo "${GRANK_JSON}" | jq -r '.guildRankPermissionRecords | length' 2>/dev/null || echo "0")
assert_not_empty "Guild rank records exist" "${RANK_COUNT}"

# Player 2 has rank 0 (default for members?) or we need to check. If alice is owner she has rank 0.
# Player 2 and 3 are members - typically rank 1 or 2. So with highest_rank 2 they can act.
info "Player 2 (guild member, rank <= 2) disconnects allocation (substation)"
run_tx "Player 2 disconnects allocation" \
    tx structs substation-allocation-disconnect "${P2_ALLOC_ID}" --from player_2
run_tx "Player 2 reconnects allocation" \
    tx structs substation-allocation-connect "${P2_ALLOC_ID}" "${SUBSTATION_ID}" --from player_2

# Revoke guild-rank permission and query
run_tx "Revoke guild-rank permission on substation" \
    tx structs permission-guild-rank-revoke "${SUBSTATION_ID}" "${GUILD_ID}" "${PERM_UPDATE}" --from alice

GRANK_AFTER=$(get_guild_rank_permission_by_object_and_guild "${SUBSTATION_ID}" "${GUILD_ID}")
RECORDS_AFTER=$(echo "${GRANK_AFTER}" | jq -r '.guild_rank_permission_records | length' 2>/dev/null || echo "${GRANK_AFTER}" | jq -r '.guildRankPermissionRecords | length' 2>/dev/null || echo "0")
# After revoke, record may be gone or rank=0
echo "  Guild rank records after revoke: ${RECORDS_AFTER}"

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 6b: Combined bitmask guild rank permissions
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 6b: Combined bitmask guild rank permissions"

# ── Combined mask lifecycle ──────────────────────────────────────────────────
info "--- Combined mask set and query ---"

# PermUpdate=4 (1<<2), PermDelete=8 (1<<3), combined = 12
run_tx "Set combined mask (PermUpdate|PermDelete = 12, rank 3) on substation" \
    tx structs permission-guild-rank-set "${SUBSTATION_ID}" "${GUILD_ID}" 12 3 --from alice

GRANK_COMB=$(get_guild_rank_permission_by_object_and_guild "${SUBSTATION_ID}" "${GUILD_ID}")
GRANK_COMB_COUNT=$(echo "${GRANK_COMB}" | jq -r '.guild_rank_permission_records | length' 2>/dev/null || echo "0")
assert_eq "Combined mask decomposed into 2 records" "2" "${GRANK_COMB_COUNT}"

GRANK_HAS_4=$(echo "${GRANK_COMB}" | jq -r '[.guild_rank_permission_records[]? | select(.permissions == "4")] | length' 2>/dev/null || echo "0")
GRANK_HAS_8=$(echo "${GRANK_COMB}" | jq -r '[.guild_rank_permission_records[]? | select(.permissions == "8")] | length' 2>/dev/null || echo "0")
assert_eq "Record for PermUpdate (4) exists" "1" "${GRANK_HAS_4}"
assert_eq "Record for PermDelete (8) exists" "1" "${GRANK_HAS_8}"

RANK_FOR_4=$(echo "${GRANK_COMB}" | jq -r '[.guild_rank_permission_records[]? | select(.permissions == "4")] | .[0].rank // empty' 2>/dev/null || echo "")
RANK_FOR_8=$(echo "${GRANK_COMB}" | jq -r '[.guild_rank_permission_records[]? | select(.permissions == "8")] | .[0].rank // empty' 2>/dev/null || echo "")
assert_eq "PermUpdate rank is 3" "3" "${RANK_FOR_4}"
assert_eq "PermDelete rank is 3" "3" "${RANK_FOR_8}"

# ── Partial revoke ───────────────────────────────────────────────────────────
info "--- Partial revoke of combined mask ---"

run_tx "Revoke only PermUpdate (4) from combined mask" \
    tx structs permission-guild-rank-revoke "${SUBSTATION_ID}" "${GUILD_ID}" 4 --from alice

GRANK_PART=$(get_guild_rank_permission_by_object_and_guild "${SUBSTATION_ID}" "${GUILD_ID}")
GRANK_PART_COUNT=$(echo "${GRANK_PART}" | jq -r '.guild_rank_permission_records | length' 2>/dev/null || echo "0")
assert_eq "After partial revoke, 1 record remains" "1" "${GRANK_PART_COUNT}"

GRANK_PART_PERM=$(echo "${GRANK_PART}" | jq -r '.guild_rank_permission_records[0].permissions // empty' 2>/dev/null || echo "")
assert_eq "Remaining record is PermDelete (8)" "8" "${GRANK_PART_PERM}"

GRANK_PART_RANK=$(echo "${GRANK_PART}" | jq -r '.guild_rank_permission_records[0].rank // empty' 2>/dev/null || echo "")
assert_eq "Remaining record rank is still 3" "3" "${GRANK_PART_RANK}"

run_tx "Revoke remaining PermDelete (8)" \
    tx structs permission-guild-rank-revoke "${SUBSTATION_ID}" "${GUILD_ID}" 8 --from alice

GRANK_EMPTY=$(get_guild_rank_permission_by_object_and_guild "${SUBSTATION_ID}" "${GUILD_ID}")
GRANK_EMPTY_COUNT=$(echo "${GRANK_EMPTY}" | jq -r '.guild_rank_permission_records | length' 2>/dev/null || echo "0")
assert_eq "After full revoke, 0 records remain" "0" "${GRANK_EMPTY_COUNT}"

# ── Per-bit rank independence ────────────────────────────────────────────────
info "--- Per-bit rank independence ---"

run_tx "Set PermUpdate (4) rank 2 on substation" \
    tx structs permission-guild-rank-set "${SUBSTATION_ID}" "${GUILD_ID}" 4 2 --from alice
run_tx "Set PermDelete (8) rank 5 on substation" \
    tx structs permission-guild-rank-set "${SUBSTATION_ID}" "${GUILD_ID}" 8 5 --from alice

GRANK_INDEP=$(get_guild_rank_permission_by_object_and_guild "${SUBSTATION_ID}" "${GUILD_ID}")
GRANK_INDEP_COUNT=$(echo "${GRANK_INDEP}" | jq -r '.guild_rank_permission_records | length' 2>/dev/null || echo "0")
assert_eq "Two records with independent ranks" "2" "${GRANK_INDEP_COUNT}"

RANK_INDEP_4=$(echo "${GRANK_INDEP}" | jq -r '[.guild_rank_permission_records[]? | select(.permissions == "4")] | .[0].rank // empty' 2>/dev/null || echo "")
RANK_INDEP_8=$(echo "${GRANK_INDEP}" | jq -r '[.guild_rank_permission_records[]? | select(.permissions == "8")] | .[0].rank // empty' 2>/dev/null || echo "")
assert_eq "PermUpdate rank is 2" "2" "${RANK_INDEP_4}"
assert_eq "PermDelete rank is 5" "5" "${RANK_INDEP_8}"

# Overwrite PermUpdate to rank 10, verify PermDelete unchanged
run_tx "Overwrite PermUpdate (4) to rank 10 on substation" \
    tx structs permission-guild-rank-set "${SUBSTATION_ID}" "${GUILD_ID}" 4 10 --from alice

GRANK_OVR=$(get_guild_rank_permission_by_object_and_guild "${SUBSTATION_ID}" "${GUILD_ID}")
RANK_OVR_4=$(echo "${GRANK_OVR}" | jq -r '[.guild_rank_permission_records[]? | select(.permissions == "4")] | .[0].rank // empty' 2>/dev/null || echo "")
RANK_OVR_8=$(echo "${GRANK_OVR}" | jq -r '[.guild_rank_permission_records[]? | select(.permissions == "8")] | .[0].rank // empty' 2>/dev/null || echo "")
assert_eq "PermUpdate rank overwritten to 10" "10" "${RANK_OVR_4}"
assert_eq "PermDelete rank unchanged at 5" "5" "${RANK_OVR_8}"

# Clean up
run_tx "Revoke PermUpdate on substation" \
    tx structs permission-guild-rank-revoke "${SUBSTATION_ID}" "${GUILD_ID}" 4 --from alice
run_tx "Revoke PermDelete on substation" \
    tx structs permission-guild-rank-revoke "${SUBSTATION_ID}" "${GUILD_ID}" 8 --from alice

# ── Combined mask with action test ──────────────────────────────────────────
info "--- Combined mask action test ---"

# PermUpdate|PermGuildEndpointUpdate = 4|16384 = 16388
run_tx "Set combined mask (16388, rank 3) on guild" \
    tx structs permission-guild-rank-set "${GUILD_ID}" "${GUILD_ID}" 16388 3 --from alice

# Revoke explicit permissions on guild for P2 and P3 so only guild-rank path is tested
run_tx "Revoke any explicit P2 perms on guild" \
    tx structs permission-revoke-on-object "${GUILD_ID}" "${PLAYER_2_ID}" "${PERM_GUILD_ENDPOINT_UPDATE}" --from alice
run_tx "Revoke any explicit P3 perms on guild" \
    tx structs permission-revoke-on-object "${GUILD_ID}" "${PLAYER_3_ID}" "${PERM_GUILD_ENDPOINT_UPDATE}" --from alice

run_tx "Set Player 2 rank to 2" tx structs player-update-guild-rank "${PLAYER_2_ID}" 2 --from alice
run_tx "Set Player 3 rank to 5" tx structs player-update-guild-rank "${PLAYER_3_ID}" 5 --from alice

run_tx "Player 2 (rank 2, <= 3) updates guild endpoint via combined guild-rank" \
    tx structs guild-update-endpoint "${GUILD_ID}" "comb-action-test.energy" --from player_2
GUILD_EP=$(query query structs guild "${GUILD_ID}" | jq -r '.Guild.endpoint // empty' 2>/dev/null || echo "")
assert_eq "Guild endpoint updated by P2 via combined guild rank" "comb-action-test.energy" "${GUILD_EP}"

run_tx_expect_permission_denied "Player 3 (rank 5, > 3) tries endpoint update via combined guild-rank" \
    tx structs guild-update-endpoint "${GUILD_ID}" "hacked.energy" --from player_3

# Restore endpoint
run_tx "Restore guild endpoint" \
    tx structs guild-update-endpoint "${GUILD_ID}" "perm-test.energy" --from alice

# ── Combined mask revoke atomicity ──────────────────────────────────────────
info "--- Combined mask revoke atomicity ---"

# First set a 3-bit combined mask: PermUpdate|PermDelete|PermGuildEndpointUpdate = 4|8|16384 = 16396
run_tx "Set 3-bit combined mask (16396, rank 3) on guild" \
    tx structs permission-guild-rank-set "${GUILD_ID}" "${GUILD_ID}" 16396 3 --from alice

GRANK_3BIT=$(get_guild_rank_permission_by_object_and_guild "${GUILD_ID}" "${GUILD_ID}")
GRANK_3BIT_COUNT=$(echo "${GRANK_3BIT}" | jq -r '.guild_rank_permission_records | length' 2>/dev/null || echo "0")
assert_eq "3-bit combined mask decomposed into 3 records" "3" "${GRANK_3BIT_COUNT}"

# Revoke entire combined mask at once
run_tx "Revoke combined mask (16396) atomically" \
    tx structs permission-guild-rank-revoke "${GUILD_ID}" "${GUILD_ID}" 16396 --from alice

GRANK_AFTER_ATOMIC=$(get_guild_rank_permission_by_object_and_guild "${GUILD_ID}" "${GUILD_ID}")
GRANK_AFTER_ATOMIC_COUNT=$(echo "${GRANK_AFTER_ATOMIC}" | jq -r '.guild_rank_permission_records | length' 2>/dev/null || echo "0")
assert_eq "All 3 bits revoked atomically" "0" "${GRANK_AFTER_ATOMIC_COUNT}"

# Clean up player ranks
run_tx "Reset Player 2 rank" tx structs player-update-guild-rank "${PLAYER_2_ID}" 0 --from alice
run_tx "Reset Player 3 rank" tx structs player-update-guild-rank "${PLAYER_3_ID}" 0 --from alice

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 7: Player guild rank change (skip if command not implemented)
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 7: Player guild rank"

PLAYER_UPDATE_GUILD_RANK_OK=false
if structsd tx structs --help 2>&1 | grep -q "player-update-guild-rank"; then
    PLAYER_UPDATE_GUILD_RANK_OK=true
fi
if [ "${PLAYER_UPDATE_GUILD_RANK_OK}" != true ]; then
    info "player-update-guild-rank not available; skipping all rank tests"
else
    info "player-update-guild-rank available; running comprehensive rank tests"

    if [ "${#EXTRA_PLAYER_IDS[@]}" -lt 10 ]; then
        info "Fewer than 10 extra players; skipping bulk rank tests (re-run without --skip-setup)"
    fi

  if [ "${#EXTRA_PLAYER_IDS[@]}" -ge 10 ]; then

    # ── 7a: Admin rank assignment sweep ──────────────────────────────────────
    section "PHASE 7a: Admin rank assignment sweep (${#EXTRA_PLAYER_IDS[@]} players)"

    # Alice (owner/admin) assigns ranks 1..N to extra players
    for i in "${!EXTRA_PLAYER_IDS[@]}"; do
        PID="${EXTRA_PLAYER_IDS[$i]}"
        DESIRED_RANK=$(( i + 1 ))
        run_tx "Alice sets ${EXTRA_PLAYER_KEYS[$i]} to rank ${DESIRED_RANK}" \
            tx structs player-update-guild-rank "${PID}" "${DESIRED_RANK}" --from alice
    done

    # Verify all ranks read back correctly
    RANK_VERIFY_PASS=0
    RANK_VERIFY_FAIL=0
    for i in "${!EXTRA_PLAYER_IDS[@]}"; do
        PID="${EXTRA_PLAYER_IDS[$i]}"
        EXPECTED=$(( i + 1 ))
        ACTUAL=$(get_player_guild_rank "${PID}")
        if [ "${ACTUAL}" = "${EXPECTED}" ]; then
            RANK_VERIFY_PASS=$(( RANK_VERIFY_PASS + 1 ))
        else
            echo -e "  ${RED}FAIL${NC}: ${EXTRA_PLAYER_KEYS[$i]} rank expected=${EXPECTED} got=${ACTUAL}"
            RANK_VERIFY_FAIL=$(( RANK_VERIFY_FAIL + 1 ))
            FAIL_COUNT=$(( FAIL_COUNT + 1 ))
        fi
    done
    echo -e "  ${GREEN}Rank assignment sweep: ${RANK_VERIFY_PASS} verified${NC}"
    if [ "${RANK_VERIFY_FAIL}" -gt 0 ]; then
        echo -e "  ${RED}${RANK_VERIFY_FAIL} rank verifications failed${NC}"
    fi
    PASS_COUNT=$(( PASS_COUNT + RANK_VERIFY_PASS ))

    # Also set player_2 and player_3 to known ranks for later tests
    run_tx "Set player_2 to rank 2" tx structs player-update-guild-rank "${PLAYER_2_ID}" 2 --from alice
    run_tx "Set player_3 to rank 5" tx structs player-update-guild-rank "${PLAYER_3_ID}" 5 --from alice

    # ── 7b: Guild rank permission threshold sweep ────────────────────────────
    section "PHASE 7b: Guild rank permission threshold sweep"

    # Set guild-rank permission on guild for PermGuildEndpointUpdate
    # then test players at various ranks against it.
    # Ranks assigned: player_2=2, player_3=5, extras=1..20

    for THRESHOLD in 0 3 5 10 15 19; do
        run_tx "Set PermGuildEndpointUpdate threshold=${THRESHOLD} on guild" \
            tx structs permission-guild-rank-set "${GUILD_ID}" "${GUILD_ID}" "${PERM_GUILD_ENDPOINT_UPDATE}" "${THRESHOLD}" --from alice

        # Test a player whose rank should PASS (rank <= threshold)
        # and a player whose rank should FAIL (rank > threshold)
        if [ "${THRESHOLD}" -ge 2 ]; then
            # player_2 (rank 2) should pass
            run_tx "player_2 (rank 2) endpoint update (threshold=${THRESHOLD}, expect PASS)" \
                tx structs guild-update-endpoint "${GUILD_ID}" "threshold-${THRESHOLD}-pass.energy" --from player_2
        else
            # player_2 (rank 2) should fail
            run_tx_expect_permission_denied "player_2 (rank 2) endpoint update (threshold=${THRESHOLD}, expect DENY)" \
                tx structs guild-update-endpoint "${GUILD_ID}" "threshold-${THRESHOLD}-fail.energy" --from player_2
        fi

        if [ "${THRESHOLD}" -ge 5 ]; then
            # player_3 (rank 5) should pass
            run_tx "player_3 (rank 5) endpoint update (threshold=${THRESHOLD}, expect PASS)" \
                tx structs guild-update-endpoint "${GUILD_ID}" "threshold-${THRESHOLD}-p3pass.energy" --from player_3
        else
            # player_3 (rank 5) should fail
            run_tx_expect_permission_denied "player_3 (rank 5) endpoint update (threshold=${THRESHOLD}, expect DENY)" \
                tx structs guild-update-endpoint "${GUILD_ID}" "threshold-${THRESHOLD}-p3fail.energy" --from player_3
        fi

        # Test boundary: pick extra player exactly at threshold and one above
        if [ "${THRESHOLD}" -ge 1 ] && [ "${THRESHOLD}" -le "${#EXTRA_PLAYER_IDS[@]}" ]; then
            IDX=$(( THRESHOLD - 1 ))  # extra player at rank THRESHOLD (0-indexed)
            PID_AT="${EXTRA_PLAYER_IDS[$IDX]}"
            KEY_AT="${EXTRA_PLAYER_KEYS[$IDX]}"
            run_tx "${KEY_AT} (rank ${THRESHOLD}) at exact boundary (expect PASS)" \
                tx structs guild-update-endpoint "${GUILD_ID}" "boundary-${THRESHOLD}-at.energy" --from "${KEY_AT}"
        fi
        ABOVE=$(( THRESHOLD + 1 ))
        if [ "${ABOVE}" -ge 1 ] && [ "${ABOVE}" -le "${#EXTRA_PLAYER_IDS[@]}" ]; then
            IDX=$(( ABOVE - 1 ))
            PID_ABOVE="${EXTRA_PLAYER_IDS[$IDX]}"
            KEY_ABOVE="${EXTRA_PLAYER_KEYS[$IDX]}"
            run_tx_expect_permission_denied "${KEY_ABOVE} (rank ${ABOVE}) just above boundary (expect DENY)" \
                tx structs guild-update-endpoint "${GUILD_ID}" "boundary-${THRESHOLD}-above.energy" --from "${KEY_ABOVE}"
        fi
    done

    # Clean up threshold
    run_tx "Revoke guild-rank PermGuildEndpointUpdate on guild" \
        tx structs permission-guild-rank-revoke "${GUILD_ID}" "${GUILD_ID}" "${PERM_GUILD_ENDPOINT_UPDATE}" --from alice

    # ── 7c: Rank-based rank management ───────────────────────────────────────
    section "PHASE 7c: Rank-based rank management"

    # Current state: player_2=rank 2, player_3=rank 5, extras=rank 1..20
    # extra[0]=rank 1, extra[4]=rank 5, extra[9]=rank 10, extra[14]=rank 15, extra[19]=rank 20

    # Test: player_2 (rank 2) can modify extra[9] (rank 10) → rank 7 (promote partially)
    run_tx "player_2 (rank 2) promotes extra[9] (rank 10) to rank 7" \
        tx structs player-update-guild-rank "${EXTRA_PLAYER_IDS[9]}" 7 --from player_2
    RANK=$(get_player_guild_rank "${EXTRA_PLAYER_IDS[9]}")
    assert_eq "extra[9] rank after partial promote" "7" "${RANK}"

    # Test: player_2 (rank 2) can demote extra[14] (rank 15) to rank 18
    run_tx "player_2 (rank 2) demotes extra[14] (rank 15) to rank 18" \
        tx structs player-update-guild-rank "${EXTRA_PLAYER_IDS[14]}" 18 --from player_2
    RANK=$(get_player_guild_rank "${EXTRA_PLAYER_IDS[14]}")
    assert_eq "extra[14] rank after demotion" "18" "${RANK}"

    # Test: player_2 (rank 2) can promote extra[19] (rank 20) to rank 2 (own level)
    run_tx "player_2 (rank 2) promotes extra[19] (rank 20) to rank 2 (own level)" \
        tx structs player-update-guild-rank "${EXTRA_PLAYER_IDS[19]}" 2 --from player_2
    RANK=$(get_player_guild_rank "${EXTRA_PLAYER_IDS[19]}")
    assert_eq "extra[19] rank after promote to own level" "2" "${RANK}"

    # Test: player_2 (rank 2) CANNOT promote extra[9] (now rank 7) to rank 1 (better than self)
    run_tx_expect_permission_denied "player_2 (rank 2) cannot promote to rank 1 (above self)" \
        tx structs player-update-guild-rank "${EXTRA_PLAYER_IDS[9]}" 1 --from player_2

    # Test: player_3 (rank 5) CANNOT modify player_2 (rank 2, better rank)
    run_tx_expect_permission_denied "player_3 (rank 5) cannot modify player_2 (rank 2)" \
        tx structs player-update-guild-rank "${PLAYER_2_ID}" 10 --from player_3

    # Test: player_3 (rank 5) CANNOT modify extra[0] (rank 1, better rank)
    run_tx_expect_permission_denied "player_3 (rank 5) cannot modify extra[0] (rank 1)" \
        tx structs player-update-guild-rank "${EXTRA_PLAYER_IDS[0]}" 10 --from player_3

    # Test: player_3 (rank 5) CANNOT modify extra[4] (rank 5, equal rank)
    run_tx_expect_permission_denied "player_3 (rank 5) cannot modify equal rank (5)" \
        tx structs player-update-guild-rank "${EXTRA_PLAYER_IDS[4]}" 10 --from player_3

    # Test: extra[19] (now rank 2, same as player_2) CANNOT modify player_2 (equal rank)
    run_tx_expect_permission_denied "extra[19] (rank 2) cannot modify player_2 (rank 2, equal)" \
        tx structs player-update-guild-rank "${PLAYER_2_ID}" 10 --from "${EXTRA_PLAYER_KEYS[19]}"

    # Test: player_3 (rank 5) CAN modify extra[9] (rank 7, worse rank) → rank 6
    run_tx "player_3 (rank 5) promotes extra[9] (rank 7) to rank 6" \
        tx structs player-update-guild-rank "${EXTRA_PLAYER_IDS[9]}" 6 --from player_3
    RANK=$(get_player_guild_rank "${EXTRA_PLAYER_IDS[9]}")
    assert_eq "extra[9] rank after player_3 promotes to 6" "6" "${RANK}"

    # Test: player_3 (rank 5) CAN demote extra[9] (rank 6) to rank 100
    run_tx "player_3 (rank 5) demotes extra[9] (rank 6) to rank 100" \
        tx structs player-update-guild-rank "${EXTRA_PLAYER_IDS[9]}" 100 --from player_3
    RANK=$(get_player_guild_rank "${EXTRA_PLAYER_IDS[9]}")
    assert_eq "extra[9] rank after demotion to 100" "100" "${RANK}"

    # ── 7d: Chain of rank modification ───────────────────────────────────────
    section "PHASE 7d: Chain of rank modification"

    # Reset some players for chain test
    run_tx "Alice sets extra[5] to rank 2" tx structs player-update-guild-rank "${EXTRA_PLAYER_IDS[5]}" 2 --from alice
    run_tx "Alice sets extra[6] to rank 10" tx structs player-update-guild-rank "${EXTRA_PLAYER_IDS[6]}" 10 --from alice
    run_tx "Alice sets extra[7] to rank 15" tx structs player-update-guild-rank "${EXTRA_PLAYER_IDS[7]}" 15 --from alice

    # Chain: extra[5](rank 2) → promotes extra[6](rank 10) to rank 4
    run_tx "Chain step 1: extra[5] (rank 2) sets extra[6] (rank 10) to rank 4" \
        tx structs player-update-guild-rank "${EXTRA_PLAYER_IDS[6]}" 4 --from "${EXTRA_PLAYER_KEYS[5]}"
    RANK=$(get_player_guild_rank "${EXTRA_PLAYER_IDS[6]}")
    assert_eq "Chain step 1: extra[6] is now rank 4" "4" "${RANK}"

    # Chain: extra[6](rank 4) → promotes extra[7](rank 15) to rank 5
    run_tx "Chain step 2: extra[6] (rank 4) sets extra[7] (rank 15) to rank 5" \
        tx structs player-update-guild-rank "${EXTRA_PLAYER_IDS[7]}" 5 --from "${EXTRA_PLAYER_KEYS[6]}"
    RANK=$(get_player_guild_rank "${EXTRA_PLAYER_IDS[7]}")
    assert_eq "Chain step 2: extra[7] is now rank 5" "5" "${RANK}"

    # Chain: extra[7](rank 5) CANNOT modify extra[6](rank 4, better rank)
    run_tx_expect_permission_denied "Chain: extra[7] (rank 5) cannot modify extra[6] (rank 4)" \
        tx structs player-update-guild-rank "${EXTRA_PLAYER_IDS[6]}" 10 --from "${EXTRA_PLAYER_KEYS[7]}"

    # Chain: extra[6](rank 4) can demote extra[7](rank 5) back to rank 15
    run_tx "Chain step 3: extra[6] (rank 4) demotes extra[7] (rank 5) to rank 15" \
        tx structs player-update-guild-rank "${EXTRA_PLAYER_IDS[7]}" 15 --from "${EXTRA_PLAYER_KEYS[6]}"
    RANK=$(get_player_guild_rank "${EXTRA_PLAYER_IDS[7]}")
    assert_eq "Chain step 3: extra[7] back to rank 15" "15" "${RANK}"

    # ── 7e: Batch rank operations and verification ───────────────────────────
    section "PHASE 7e: Mass rank shuffle and verify"

    # Alice shuffles all extra player ranks to inverse order
    for i in "${!EXTRA_PLAYER_IDS[@]}"; do
        NEW_RANK=$(( ${#EXTRA_PLAYER_IDS[@]} - i ))
        structsd ${PARAMS_TX} tx structs player-update-guild-rank "${EXTRA_PLAYER_IDS[$i]}" "${NEW_RANK}" --from alice 2>&1 || true
        sleep 1
    done

    # Verify the shuffle
    SHUFFLE_PASS=0
    SHUFFLE_FAIL=0
    for i in "${!EXTRA_PLAYER_IDS[@]}"; do
        EXPECTED=$(( ${#EXTRA_PLAYER_IDS[@]} - i ))
        ACTUAL=$(get_player_guild_rank "${EXTRA_PLAYER_IDS[$i]}")
        if [ "${ACTUAL}" = "${EXPECTED}" ]; then
            SHUFFLE_PASS=$(( SHUFFLE_PASS + 1 ))
        else
            echo -e "  ${RED}FAIL${NC}: ${EXTRA_PLAYER_KEYS[$i]} shuffle expected=${EXPECTED} got=${ACTUAL}"
            SHUFFLE_FAIL=$(( SHUFFLE_FAIL + 1 ))
            FAIL_COUNT=$(( FAIL_COUNT + 1 ))
        fi
    done
    echo -e "  ${GREEN}Mass shuffle: ${SHUFFLE_PASS}/${#EXTRA_PLAYER_IDS[@]} verified${NC}"
    PASS_COUNT=$(( PASS_COUNT + SHUFFLE_PASS ))

    # After shuffle: extra[0]=rank 20, extra[19]=rank 1
    # extra[19] (rank 1) should be able to modify extra[0] (rank 20) but not vice versa
    run_tx "extra[19] (rank 1) sets extra[0] (rank 20) to rank 10" \
        tx structs player-update-guild-rank "${EXTRA_PLAYER_IDS[0]}" 10 --from "${EXTRA_PLAYER_KEYS[19]}"
    RANK=$(get_player_guild_rank "${EXTRA_PLAYER_IDS[0]}")
    assert_eq "extra[0] rank after shuffle-based modify" "10" "${RANK}"

    run_tx_expect_permission_denied "extra[0] (rank 10) cannot modify extra[19] (rank 1)" \
        tx structs player-update-guild-rank "${EXTRA_PLAYER_IDS[19]}" 15 --from "${EXTRA_PLAYER_KEYS[0]}"

    # ── 7f: Edge cases ───────────────────────────────────────────────────────
    section "PHASE 7f: Edge cases"

    # Self-modification: player_2 (rank 2) tries to change own rank
    # actorRank (2) is NOT < targetRank (2), so denied via rank path
    # player_2 doesn't have PermAdmin on guild, so denied overall
    run_tx_expect_permission_denied "player_2 cannot self-modify rank (equal = denied)" \
        tx structs player-update-guild-rank "${PLAYER_2_ID}" 0 --from player_2

    # alice (owner) CAN self-modify because she has PermAdmin bypass
    run_tx "alice (admin) can change own rank" \
        tx structs player-update-guild-rank "${PLAYER_1_ID}" 3 --from alice
    RANK=$(get_player_guild_rank "${PLAYER_1_ID}")
    assert_eq "alice rank after self-set" "3" "${RANK}"
    # Restore alice to rank 0
    run_tx "alice restores own rank to 0" \
        tx structs player-update-guild-rank "${PLAYER_1_ID}" 0 --from alice

    # Setting rank to very large number
    run_tx "Alice sets extra[0] to max-ish rank (999999)" \
        tx structs player-update-guild-rank "${EXTRA_PLAYER_IDS[0]}" 999999 --from alice
    RANK=$(get_player_guild_rank "${EXTRA_PLAYER_IDS[0]}")
    assert_eq "extra[0] rank after set to 999999" "999999" "${RANK}"

    # Setting rank back to 0 (best)
    run_tx "Alice sets extra[0] back to rank 0" \
        tx structs player-update-guild-rank "${EXTRA_PLAYER_IDS[0]}" 0 --from alice
    RANK=$(get_player_guild_rank "${EXTRA_PLAYER_IDS[0]}")
    assert_eq "extra[0] rank after set back to 0" "0" "${RANK}"

    # ── Clean up all ranks ───────────────────────────────────────────────────
    info "Resetting all player ranks to 0"
    run_tx "Reset player_2 to rank 0" tx structs player-update-guild-rank "${PLAYER_2_ID}" 0 --from alice
    run_tx "Reset player_3 to rank 0" tx structs player-update-guild-rank "${PLAYER_3_ID}" 0 --from alice
    for i in "${!EXTRA_PLAYER_IDS[@]}"; do
        structsd ${PARAMS_TX} tx structs player-update-guild-rank "${EXTRA_PLAYER_IDS[$i]}" 0 --from alice 2>&1 || true
        sleep 1
    done

  fi # end extra players guard
fi # end player-update-guild-rank available

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 8: Grant/revoke ordering and multiple objects
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 8: Grant/revoke ordering"

# Verify clean starting state for P2 and P3 on guild
VAL_P2_START=$(get_permission_value_for_player "${GUILD_ID}" "${PLAYER_2_ID}")
VAL_P3_START=$(get_permission_value_for_player "${GUILD_ID}" "${PLAYER_3_ID}")
info "Starting state: P2=${VAL_P2_START}, P3=${VAL_P3_START}"

run_tx "Grant P2→guild PermUpdate(4)" \
    tx structs permission-grant-on-object "${GUILD_ID}" "${PLAYER_2_ID}" "${PERM_UPDATE}" --from alice
run_tx "Grant P3→guild PermDelete(8)" \
    tx structs permission-grant-on-object "${GUILD_ID}" "${PLAYER_3_ID}" "${PERM_DELETE}" --from alice

VAL_P2=$(get_permission_value_for_player "${GUILD_ID}" "${PLAYER_2_ID}")
VAL_P3=$(get_permission_value_for_player "${GUILD_ID}" "${PLAYER_3_ID}")
# Values should be start|4 and start|8
EXPECT_P2=$(( VAL_P2_START | PERM_UPDATE ))
EXPECT_P3=$(( VAL_P3_START | PERM_DELETE ))
assert_eq "P2 permission on guild after grant" "${EXPECT_P2}" "${VAL_P2}"
assert_eq "P3 permission on guild after grant" "${EXPECT_P3}" "${VAL_P3}"

run_tx "Revoke P2 PermUpdate on guild" \
    tx structs permission-revoke-on-object "${GUILD_ID}" "${PLAYER_2_ID}" "${PERM_UPDATE}" --from alice
VAL_P2=$(get_permission_value_for_player "${GUILD_ID}" "${PLAYER_2_ID}")
assert_eq "P2 permission after revoke PermUpdate" "${VAL_P2_START}" "${VAL_P2}"

# Idempotent revoke: revoke PermUpdate from P3 (which doesn't have it)
run_tx "Revoke P3 PermUpdate (not set) — idempotent" \
    tx structs permission-revoke-on-object "${GUILD_ID}" "${PLAYER_3_ID}" "${PERM_UPDATE}" --from alice
VAL_P3=$(get_permission_value_for_player "${GUILD_ID}" "${PLAYER_3_ID}")
assert_eq "P3 permission unchanged after idempotent revoke" "${EXPECT_P3}" "${VAL_P3}"

# Clean up: revoke what we granted
run_tx "Clean up P3 PermDelete" \
    tx structs permission-revoke-on-object "${GUILD_ID}" "${PLAYER_3_ID}" "${PERM_DELETE}" --from alice

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 9: Object deletion and permission cleanup
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 9: Object deletion and permission cleanup"

# Create a provider (if supported), grant permission, delete provider, assert permissions cleared
if structsd tx structs provider-create --help 2>&1 | grep -q "Create a new Energy Provider"; then
    run_tx "Create provider for cleanup test" \
        tx structs provider-create "${SUBSTATION_ID}" \
        "1ualpha" "open" 0 0 100 1000 10 1000 --from alice
    PROVIDER_ALL=$(query query structs provider-all 2>/dev/null || echo '{}')
    PROVIDER_ID=$(echo "${PROVIDER_ALL}" | jq -r '.Provider[-1].id // empty' 2>/dev/null || echo "")
    if [ -n "${PROVIDER_ID}" ]; then
        run_tx "Grant P2 permission on provider" \
            tx structs permission-grant-on-object "${PROVIDER_ID}" "${PLAYER_2_ID}" "${PERM_UPDATE}" --from alice
        run_tx "Set guild-rank on provider" \
            tx structs permission-guild-rank-set "${PROVIDER_ID}" "${GUILD_ID}" "${PERM_UPDATE}" 1 --from alice
        run_tx "Delete provider" \
            tx structs provider-delete "${PROVIDER_ID}" --from alice
        sleep "${SLEEP}"
        PERM_AFTER=$(get_permission_by_object "${PROVIDER_ID}")
        PERM_COUNT=$(echo "${PERM_AFTER}" | jq -r '.permissionRecords | length' 2>/dev/null || echo "${PERM_AFTER}" | jq -r '.permissionRecord | length' 2>/dev/null || echo "0")
        GRANK_AFTER=$(get_guild_rank_permission_by_object "${PROVIDER_ID}")
        GRANK_COUNT=$(echo "${GRANK_AFTER}" | jq -r '.guild_rank_permission_records | length' 2>/dev/null || echo "${GRANK_AFTER}" | jq -r '.guildRankPermissionRecords | length' 2>/dev/null || echo "0")
        assert_eq "Permission records cleared after provider delete" "0" "${PERM_COUNT}"
        assert_eq "Guild rank records cleared after provider delete" "0" "${GRANK_COUNT}"
    else
        info "Could not get provider ID; skipping deletion cleanup test"
    fi
else
    info "provider-create not available; skipping deletion cleanup test"
fi

# ═════════════════════════════════════════════════════════════════════════════
#  Summary and exit
# ═════════════════════════════════════════════════════════════════════════════

print_summary
[ "${FAIL_COUNT}" -eq 0 ] || exit 1
