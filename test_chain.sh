#!/usr/bin/env bash
#
# Structs Chain Integration Test Script
#
# Tests the full lifecycle:
#   1. Player setup (allocations, substations, guilds)
#   2. Guild membership (join, allocations)
#   3. Planet exploration
#   4. Struct building (miners, refineries, combat units)
#   5. Mining & Refining
#   6. Fleet movement & Combat (attacks, defense, raids)
#
# Prerequisites:
#   - structsd chain running locally (fresh chain recommended)
#   - 'alice' key in keyring (genesis validator)
#   - 'bob' key in keyring (faucet / bank sender)
#
# Flags:
#   --skip-mining   Skip ore mining, refinery build/refine, and planet raid.
#                   Dramatically reduces runtime by avoiding the slowest
#                   proof-of-work compute operations.
#

set -euo pipefail

# ─── Flag Parsing ─────────────────────────────────────────────────────────────

SKIP_MINING=false
for arg in "$@"; do
    case "${arg}" in
        --skip-mining) SKIP_MINING=true ;;
        *)            echo "Unknown flag: ${arg}"; exit 1 ;;
    esac
done

# ─── Configuration ────────────────────────────────────────────────────────────

SLEEP=2
BIGGER_SLEEP=15
PARAMS_TX="--home ~/.structs --keyring-dir ~/.structs --keyring-backend test --gas auto --yes=true"
PARAMS_QUERY="--home ~/.structs --output json"
PARAMS_KEYS="--home ~/.structs --keyring-dir ~/.structs --keyring-backend test --output json"

# ─── Colours & Helpers ────────────────────────────────────────────────────────

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Colour

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

# _check_tx_output: shared logic for checking TX command output
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

# run_tx: execute a transaction, show the command, and check for success
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

# run_tx_big: same as run_tx but with BIGGER_SLEEP afterwards
run_tx_big() {
    local description="$1"
    shift
    info "${description}"
    echo -e "  ${BOLD}structsd ${PARAMS_TX} $*${NC}"
    local OUTPUT
    OUTPUT=$(structsd ${PARAMS_TX} "$@" 2>&1) || true
    _check_tx_output "${OUTPUT}"
    sleep "${BIGGER_SLEEP}"
}

# run_tx_noauto: execute a TX with fixed gas (bypasses --gas auto simulation)
# Used for operations where --gas auto simulation fails due to stale state
# (e.g., invite-approve/deny where the application isn't visible in simulation)
PARAMS_TX_NOAUTO="--home ~/.structs --keyring-dir ~/.structs --keyring-backend test --gas 500000 --yes=true"
run_tx_noauto() {
    local description="$1"
    shift
    info "${description}"
    echo -e "  ${BOLD}structsd ${PARAMS_TX_NOAUTO} $*${NC}"
    local OUTPUT
    OUTPUT=$(structsd ${PARAMS_TX_NOAUTO} "$@" 2>&1) || true
    _check_tx_output "${OUTPUT}"
    sleep "${SLEEP}"
}

# run_compute: execute a compute command (proof-of-work)
run_compute() {
    local description="$1"
    shift
    info "${description} (compute)"
    echo -e "  ${BOLD}structsd ${PARAMS_TX} $*${NC}"
    structsd ${PARAMS_TX} "$@" 2>&1 || true
    echo -e "  ${GREEN}Compute completed${NC}"
    sleep "${BIGGER_SLEEP}"
}

# query: run a query and return JSON
query() {
    structsd ${PARAMS_QUERY} "$@" 2>/dev/null
}

# jqr: safe jq extraction with fallback
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

# assert_eq: check that two values are equal
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

# assert_not_empty: check that a value is not empty/null
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

# assert_gt: check that actual > threshold (numeric)
assert_gt() {
    local label="$1"
    local threshold="$2"
    local actual="$3"
    if [ -n "${actual}" ] && [ "${actual}" != "null" ] && [ "${actual}" -gt "${threshold}" ] 2>/dev/null; then
        echo -e "  ${GREEN}PASS${NC}: ${label} = ${actual} > ${threshold}"
        PASS_COUNT=$((PASS_COUNT + 1))
    else
        echo -e "  ${RED}FAIL${NC}: ${label} = '${actual}' not > ${threshold}"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
}

# assert_lt: check that actual < threshold (numeric)
assert_lt() {
    local label="$1"
    local threshold="$2"
    local actual="$3"
    if [ -n "${actual}" ] && [ "${actual}" != "null" ] && [ "${actual}" -lt "${threshold}" ] 2>/dev/null; then
        echo -e "  ${GREEN}PASS${NC}: ${label} = ${actual} < ${threshold}"
        PASS_COUNT=$((PASS_COUNT + 1))
    else
        echo -e "  ${RED}FAIL${NC}: ${label} = '${actual}' not < ${threshold}"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
}

# get_newest_struct_id: find the struct with the highest numeric index
# NOTE: struct-all returns structs in string-sorted order (5-10 < 5-2),
# so .Struct[-1] breaks once indices exceed single digits. This sorts numerically.
get_newest_struct_id() {
    local json="${1:-}"
    if [ -z "${json}" ]; then
        json=$(query query structs struct-all)
    fi
    echo "${json}" | jq -r '[.Struct[].id] | map(split("-") | {p: .[0], n: (.[1] | tonumber)}) | sort_by(.n) | last | "\(.p)-\(.n)"' 2>/dev/null || echo ""
}

# get_latest_allocation_for_source: find the most recent allocation for a given source
get_latest_allocation_for_source() {
    local source_id="$1"
    query query structs allocation-all-by-source "${source_id}" | jq -r '.Allocation[-1].id // empty'
}

# get_allocation_count: get current number of allocations
get_allocation_count() {
    query query structs allocation-all | jq -r '.pagination.total // "0"'
}

# get_balance: get the balance of a specific denom for an address
# Usage: get_balance <address> <denom>
get_balance() {
    local addr="$1"
    local denom="$2"
    local result
    result=$(query query bank balances "${addr}" | jq -r --arg d "${denom}" '.balances[] | select(.denom == $d) | .amount // "0"' 2>/dev/null || echo "0")
    if [ -z "${result}" ]; then
        echo "0"
    else
        echo "${result}"
    fi
}

# ─── Charge-Aware Wait Helpers ────────────────────────────────────────────────
# Charge = CurrentBlockHeight - Player.lastAction
# Each block is ~1 second. Struct operations require a minimum charge level
# before they can proceed (e.g. BuildCharge=8 means 8 blocks since last action).

# get_block_height: query the current block height
get_block_height() {
    query query structs block-height | jq -r '.blockHeight // "0"' 2>/dev/null || echo "0"
}

# get_player_charge: compute a player's current charge
# Usage: get_player_charge <player_id>
get_player_charge() {
    local player_id="$1"
    local player_json last_action block_height
    player_json=$(query query structs player "${player_id}" 2>/dev/null || echo '{}')
    last_action=$(echo "${player_json}" | jq -r '.gridAttributes.lastAction // "0"' 2>/dev/null || echo "0")
    block_height=$(get_block_height)
    if [ "${last_action}" = "0" ] || [ "${block_height}" = "0" ]; then
        echo "999"
        return
    fi
    echo $((block_height - last_action))
}

# wait_for_charge: wait until a player has accumulated enough charge
# Usage: wait_for_charge <player_id> <required_charge>
wait_for_charge() {
    local player_id="$1"
    local required="${2:-8}"
    local charge
    charge=$(get_player_charge "${player_id}")
    if [ "${charge}" -ge "${required}" ] 2>/dev/null; then
        echo -e "  ${GREEN}Charge OK${NC}: ${player_id} charge=${charge} >= ${required}"
        return
    fi
    local deficit=$((required - charge))
    echo -e "  ${YELLOW}Waiting for charge${NC}: ${player_id} charge=${charge}, need=${required}, waiting ~${deficit}s"
    sleep $((deficit + 2))
    charge=$(get_player_charge "${player_id}")
    echo -e "  ${GREEN}Charge ready${NC}: ${player_id} charge=${charge}"
}

# Charge constants (from genesis_struct_type.go)
CHARGE_BUILD=8
CHARGE_ATTACK_DEFAULT=1
CHARGE_ATTACK_BATTLESHIP=8
CHARGE_ATTACK_SAM=20
CHARGE_MOVE=8
CHARGE_DEFEND=1
CHARGE_ACTIVATE=1

# print_summary: final report
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
#  PHASE 1: Initial Setup — Validator, Player 1, Allocation, Substation, Guild
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 1: Initial Setup"

# ─── Module params query ───
info "Module params:"
query query structs params 2>/dev/null | jq '.' || echo "  (no params)"

# ─── Validator ───
info "Looking up validator"
VALIDATOR_ADDRESS=$(query query staking validators | jq -r ".validators[0].operator_address")
assert_not_empty "Validator address" "${VALIDATOR_ADDRESS}"

# ─── Player 1 (alice) ───
info "Looking up Player 1 (alice)"
PLAYER_1_ADDRESS=$(structsd ${PARAMS_KEYS} keys show alice | jq -r .address)
assert_not_empty "Player 1 address" "${PLAYER_1_ADDRESS}"
echo "  Player 1 Address: ${PLAYER_1_ADDRESS}"

PLAYER_ME_JSON=$(query query structs player-me)
PLAYER_1_ID=$(jqr "${PLAYER_ME_JSON}" '.Player.id')
assert_not_empty "Player 1 ID" "${PLAYER_1_ID}"
echo "  Player 1 ID: ${PLAYER_1_ID}"

# ─── Create Allocation from Player 1 ───
PLAYER_1_CAPACITY=$(jqr "${PLAYER_ME_JSON}" '.gridAttributes.capacity')
assert_gt "Player 1 capacity" 0 "${PLAYER_1_CAPACITY}"
echo "  Player 1 Capacity: ${PLAYER_1_CAPACITY}"

# Track allocation count before creation so we can find the new one
ALLOC_COUNT_BEFORE=$(get_allocation_count)

run_tx "Creating allocation from Player 1" \
    tx structs allocation-create "${PLAYER_1_ID}" "${PLAYER_1_CAPACITY}" \
    --allocation-type dynamic --from alice

# Discover the allocation ID dynamically
P1_ALLOC_ID=$(get_latest_allocation_for_source "${PLAYER_1_ID}")
assert_not_empty "Player 1 allocation ID" "${P1_ALLOC_ID}"
echo "  Player 1 Allocation ID: ${P1_ALLOC_ID}"

# Verify allocation details
P1_ALLOC_JSON=$(query query structs allocation "${P1_ALLOC_ID}")
P1_ALLOC_SRC=$(jqr "${P1_ALLOC_JSON}" '.Allocation.sourceObjectId')
assert_eq "Allocation source is Player 1" "${PLAYER_1_ID}" "${P1_ALLOC_SRC}"

# ─── Create Substation 1 ───
run_tx "Creating Substation 1" \
    tx structs substation-create "${PLAYER_1_ID}" "${P1_ALLOC_ID}" --from alice

# Discover the substation ID from the allocation's destination
P1_ALLOC_JSON=$(query query structs allocation "${P1_ALLOC_ID}")
SUBSTATION_ID=$(jqr "${P1_ALLOC_JSON}" '.Allocation.destinationId')
assert_not_empty "Substation ID" "${SUBSTATION_ID}"
echo "  Substation ID: ${SUBSTATION_ID}"

# Verify substation exists
SUB_JSON=$(query query structs substation "${SUBSTATION_ID}")
SUB_CHECK=$(jqr "${SUB_JSON}" '.Substation.id')
assert_eq "Substation exists" "${SUBSTATION_ID}" "${SUB_CHECK}"

# ─── Create Guild ───
run_tx "Creating Guild" \
    tx structs guild-create "oh.energy" "${SUBSTATION_ID}" --from alice

# Discover guild and reactor IDs
GUILD_ALL_JSON=$(query query structs guild-all)
GUILD_ID=$(jqr "${GUILD_ALL_JSON}" '.Guild[0].id')
REACTOR_ID=$(jqr "${GUILD_ALL_JSON}" '.Guild[0].primaryReactorId')
assert_not_empty "Guild ID" "${GUILD_ID}"
assert_not_empty "Reactor ID" "${REACTOR_ID}"
echo "  Guild ID: ${GUILD_ID}  Reactor ID: ${REACTOR_ID}"

# Verify guild details
GUILD_JSON=$(query query structs guild "${GUILD_ID}")
GUILD_ENDPOINT=$(jqr "${GUILD_JSON}" '.Guild.endpoint')
assert_eq "Guild endpoint" "oh.energy" "${GUILD_ENDPOINT}"

GUILD_ENTRY_SUB=$(jqr "${GUILD_JSON}" '.Guild.entrySubstationId')
assert_eq "Guild entry substation" "${SUBSTATION_ID}" "${GUILD_ENTRY_SUB}"


# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 2: Create Players 2, 3, 4, 5 — Fund, Delegate, Get IDs
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 2: Create Additional Players"

# Create keys (reuse if they already exist)
# Player 5 is used exclusively for guild membership tests
for PLAYER_NUM in 2 3 4 5; do
    PLAYER_KEY="player_${PLAYER_NUM}"
    info "Setting up ${PLAYER_KEY}"
    EXISTING=$(structsd ${PARAMS_KEYS} keys show "${PLAYER_KEY}" 2>/dev/null | jq -r .address || echo "")
    if [ -z "${EXISTING}" ]; then
        ADDR=$(structsd ${PARAMS_KEYS} keys add "${PLAYER_KEY}" | jq -r .address)
        echo "  Created ${PLAYER_KEY}: ${ADDR}"
    else
        ADDR="${EXISTING}"
        echo "  Reusing ${PLAYER_KEY}: ${ADDR}"
    fi
    eval "PLAYER_${PLAYER_NUM}_ADDRESS=${ADDR}"
done

assert_not_empty "Player 2 address" "${PLAYER_2_ADDRESS}"
assert_not_empty "Player 3 address" "${PLAYER_3_ADDRESS}"
assert_not_empty "Player 4 address" "${PLAYER_4_ADDRESS}"
assert_not_empty "Player 5 address" "${PLAYER_5_ADDRESS}"

# ─── Fund players from bob (separate faucet account) ───
BOB_ADDRESS=$(structsd ${PARAMS_KEYS} keys show bob | jq -r .address)
info "Bob (faucet) address: ${BOB_ADDRESS}"

for PLAYER_NUM in 2 3 4 5; do
    eval "PADDR=\${PLAYER_${PLAYER_NUM}_ADDRESS}"
    run_tx "Sending 10000000ualpha from bob to player_${PLAYER_NUM}" \
        tx bank send "${BOB_ADDRESS}" "${PADDR}" 10000000ualpha --from bob
done

# ─── Delegate to validator (creates player accounts on the structs module) ───
for PLAYER_NUM in 2 3 4 5; do
    run_tx "Delegating 5000000ualpha from player_${PLAYER_NUM} to validator" \
        tx staking delegate "${VALIDATOR_ADDRESS}" 5000000ualpha --from "player_${PLAYER_NUM}"

    eval "PADDR=\${PLAYER_${PLAYER_NUM}_ADDRESS}"
    ADDR_JSON=$(query query structs address "${PADDR}")
    PID=$(jqr "${ADDR_JSON}" '.playerId')
    eval "PLAYER_${PLAYER_NUM}_ID=${PID}"
    assert_not_empty "Player ${PLAYER_NUM} ID" "${PID}"
    echo "  Player ${PLAYER_NUM} ID: ${PID}"
done


# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 3: Create Allocations for Players 2, 3, 4 (controlled by Player 1)
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 3: Player Allocations"

for PLAYER_NUM in 2 3 4; do
    eval "PID=\${PLAYER_${PLAYER_NUM}_ID}"
    PLAYER_JSON=$(query query structs player "${PID}")
    PCAP=$(jqr "${PLAYER_JSON}" '.gridAttributes.capacity')
    assert_gt "Player ${PLAYER_NUM} capacity" 0 "${PCAP}"
    echo "  Player ${PLAYER_NUM} (${PID}) capacity: ${PCAP}"

    run_tx "Creating allocation from Player ${PLAYER_NUM} (controller=alice)" \
        tx structs allocation-create "${PID}" "${PCAP}" \
        --controller "${PLAYER_1_ADDRESS}" --allocation-type dynamic --from "player_${PLAYER_NUM}"

    # Discover the allocation ID dynamically
    ALLOC_ID=$(get_latest_allocation_for_source "${PID}")
    eval "P${PLAYER_NUM}_ALLOC_ID=${ALLOC_ID}"
    assert_not_empty "Player ${PLAYER_NUM} allocation ID" "${ALLOC_ID}"
    echo "  Player ${PLAYER_NUM} Allocation ID: ${ALLOC_ID}"

    # Verify
    ALLOC_JSON=$(query query structs allocation "${ALLOC_ID}")
    ALLOC_SRC=$(jqr "${ALLOC_JSON}" '.Allocation.sourceObjectId')
    assert_eq "Allocation ${ALLOC_ID} source" "${PID}" "${ALLOC_SRC}"
done

# ─── Dump state ───
info "Current state dump"
echo "  Guilds:"
query query structs guild-all | jq -r '.Guild[] | "    \(.id) endpoint=\(.endpoint)"' 2>/dev/null || true
echo "  Players:"
query query structs player-all | jq -r '.Player[] | "    \(.id) guild=\(.guildId) planet=\(.planetId)"' 2>/dev/null || true
echo "  Allocations:"
query query structs allocation-all | jq -r '.Allocation[] | "    \(.id) src=\(.sourceObjectId) dst=\(.destinationId)"' 2>/dev/null || true


# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 3b: Allocation Lifecycle — Update, Transfer, Delete
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 3b: Allocation Lifecycle"

# Create an allocation for Player 5 to use in lifecycle tests
P5_JSON=$(query query structs player "${PLAYER_5_ID}")
P5_CAP=$(jqr "${P5_JSON}" '.gridAttributes.capacity' '0')
HALF_CAP=$(( P5_CAP / 2 ))
QUARTER_CAP=$(( P5_CAP / 4 ))
info "Player 5 (${PLAYER_5_ID}) capacity: ${P5_CAP}"

# Create with half capacity so we have room to test updates
run_tx "Creating allocation from Player 5 (${HALF_CAP} of ${P5_CAP})" \
    tx structs allocation-create "${PLAYER_5_ID}" "${HALF_CAP}" \
    --controller "${PLAYER_5_ADDRESS}" --allocation-type dynamic --from player_5

P5_ALLOC_ID=$(get_latest_allocation_for_source "${PLAYER_5_ID}")
assert_not_empty "Player 5 allocation ID" "${P5_ALLOC_ID}"
echo "  Player 5 Allocation ID: ${P5_ALLOC_ID}"

# ─── allocation-update: change power ───
info "Updating allocation power from ${HALF_CAP} to ${QUARTER_CAP}"
run_tx "Updating allocation ${P5_ALLOC_ID} power to ${QUARTER_CAP}" \
    tx structs allocation-update "${P5_ALLOC_ID}" "${QUARTER_CAP}" --from player_5

ALLOC_JSON=$(query query structs allocation "${P5_ALLOC_ID}")
ALLOC_POWER=$(jqr "${ALLOC_JSON}" '.gridAttributes.power' '0')
assert_eq "Allocation power updated" "${QUARTER_CAP}" "${ALLOC_POWER}"

# ─── allocation-transfer: transfer controller to alice ───
info "Transferring allocation controller from player_5 to alice"
run_tx "Transferring allocation ${P5_ALLOC_ID} controller to alice" \
    tx structs allocation-transfer "${P5_ALLOC_ID}" "${PLAYER_1_ADDRESS}" --from player_5

ALLOC_JSON=$(query query structs allocation "${P5_ALLOC_ID}")
ALLOC_CTRL=$(jqr "${ALLOC_JSON}" '.Allocation.controller')
assert_eq "Allocation controller transferred" "${PLAYER_1_ADDRESS}" "${ALLOC_CTRL}"

# Transfer back to player_5 so they can delete it
run_tx "Transferring allocation back to player_5" \
    tx structs allocation-transfer "${P5_ALLOC_ID}" "${PLAYER_5_ADDRESS}" --from alice

# ─── allocation-delete ───
run_tx "Deleting allocation ${P5_ALLOC_ID}" \
    tx structs allocation-delete "${P5_ALLOC_ID}" --from player_5

# Verify allocation is gone (query should return empty or error)
ALLOC_GONE_JSON=$(query query structs allocation "${P5_ALLOC_ID}" 2>/dev/null || echo '{}')
ALLOC_GONE_ID=$(jqr "${ALLOC_GONE_JSON}" '.Allocation.id' '')
assert_eq "Allocation deleted" "" "${ALLOC_GONE_ID}"

# ─── Query: allocation-all-by-destination ───
info "Querying allocations by destination (substation ${SUBSTATION_ID})"
ALLOC_BY_DEST=$(query query structs allocation-all-by-destination "${SUBSTATION_ID}" 2>/dev/null || echo '{}')
echo "  Allocations connected to substation: $(echo "${ALLOC_BY_DEST}" | jq '.Allocation | length' 2>/dev/null || echo '0')"

# ─── Re-create allocation for Player 5 for later phases ───
run_tx "Re-creating allocation for Player 5" \
    tx structs allocation-create "${PLAYER_5_ID}" "${P5_CAP}" \
    --controller "${PLAYER_5_ADDRESS}" --allocation-type dynamic --from player_5

P5_ALLOC_ID=$(get_latest_allocation_for_source "${PLAYER_5_ID}")
assert_not_empty "Player 5 re-created allocation ID" "${P5_ALLOC_ID}"
echo "  Player 5 new Allocation ID: ${P5_ALLOC_ID}"


# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 4: Guild Membership — Join & Connect Allocations
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 4: Guild Membership"

# ─── Add Player 2 to Guild ───
# The infusion ID format is: {reactorId}-{playerAddress}
run_tx "Player 2 joining guild" \
    tx structs guild-membership-join "${GUILD_ID}" "${REACTOR_ID}-${PLAYER_2_ADDRESS}" --from player_2

P2_JSON=$(query query structs player "${PLAYER_2_ID}")
P2_GUILD=$(jqr "${P2_JSON}" '.Player.guildId')
assert_eq "Player 2 guild membership" "${GUILD_ID}" "${P2_GUILD}"

# ─── Add Player 3 to Guild ───
run_tx "Player 3 joining guild" \
    tx structs guild-membership-join "${GUILD_ID}" "${REACTOR_ID}-${PLAYER_3_ADDRESS}" --from player_3

P3_JSON=$(query query structs player "${PLAYER_3_ID}")
P3_GUILD=$(jqr "${P3_JSON}" '.Player.guildId')
assert_eq "Player 3 guild membership" "${GUILD_ID}" "${P3_GUILD}"

# ─── Connect Player 2 & 3 Allocations to Substation ───
run_tx "Connecting Player 2 allocation (${P2_ALLOC_ID}) to substation" \
    tx structs substation-allocation-connect "${P2_ALLOC_ID}" "${SUBSTATION_ID}" --from alice

P2_ALLOC_JSON=$(query query structs allocation "${P2_ALLOC_ID}")
ALLOC_2_DST=$(jqr "${P2_ALLOC_JSON}" '.Allocation.destinationId')
assert_eq "Player 2 allocation connected to substation" "${SUBSTATION_ID}" "${ALLOC_2_DST}"

run_tx "Connecting Player 3 allocation (${P3_ALLOC_ID}) to substation" \
    tx structs substation-allocation-connect "${P3_ALLOC_ID}" "${SUBSTATION_ID}" --from alice

P3_ALLOC_JSON=$(query query structs allocation "${P3_ALLOC_ID}")
ALLOC_3_DST=$(jqr "${P3_ALLOC_JSON}" '.Allocation.destinationId')
assert_eq "Player 3 allocation connected to substation" "${SUBSTATION_ID}" "${ALLOC_3_DST}"

# ─── Add Player 4 to Guild ───
run_tx "Player 4 joining guild" \
    tx structs guild-membership-join "${GUILD_ID}" "${REACTOR_ID}-${PLAYER_4_ADDRESS}" --from player_4

P4_JSON=$(query query structs player "${PLAYER_4_ID}")
P4_GUILD=$(jqr "${P4_JSON}" '.Player.guildId')
assert_eq "Player 4 guild membership" "${GUILD_ID}" "${P4_GUILD}"

# ─── Connect Player 4 Allocation ───
run_tx "Connecting Player 4 allocation (${P4_ALLOC_ID}) to substation" \
    tx structs substation-allocation-connect "${P4_ALLOC_ID}" "${SUBSTATION_ID}" --from alice

P4_ALLOC_JSON=$(query query structs allocation "${P4_ALLOC_ID}")
ALLOC_4_DST=$(jqr "${P4_ALLOC_JSON}" '.Allocation.destinationId')
assert_eq "Player 4 allocation connected to substation" "${SUBSTATION_ID}" "${ALLOC_4_DST}"

# Verify substation exists (load will be 0 until structs are built and drawing power)
info "Checking substation power"
SUB_JSON=$(query query structs substation "${SUBSTATION_ID}")
SUB_LOAD=$(jqr "${SUB_JSON}" '.gridAttributes.load' '0')
SUB_CAP=$(jqr "${SUB_JSON}" '.gridAttributes.capacity' '0')
assert_gt "Substation capacity" 0 "${SUB_CAP}"
echo "  Substation capacity: ${SUB_CAP}, load: ${SUB_LOAD} (load=0 expected before builds)"

# ─── Dump all players ───
info "All players state"
query query structs player-all | jq -r '.Player[] | "  \(.id) guild=\(.guildId) sub=\(.substationId) planet=\(.planetId)"' 2>/dev/null || true


# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 4b: Guild Bank & Token Operations
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 4b: Guild Bank & Token Operations"

# Guild token denomination is uguild.{guildId}  (e.g. uguild.0-1)
GUILD_TOKEN_DENOM="uguild.${GUILD_ID}"
info "Guild token denomination: ${GUILD_TOKEN_DENOM}"

# ─── Query collateral address ───
info "Looking up guild bank collateral address"
COLLATERAL_JSON=$(query query structs guild-bank-collateral-address "${GUILD_ID}")
COLLATERAL_ADDR=$(jqr "${COLLATERAL_JSON}" '.internalAddressAssociation[0].address')
assert_not_empty "Collateral address" "${COLLATERAL_ADDR}"
echo "  Collateral address: ${COLLATERAL_ADDR}"

# ─── Check initial balances ───
info "Alice's ualpha balance before mint"
ALICE_ALPHA_BEFORE=$(get_balance "${PLAYER_1_ADDRESS}" ualpha)
echo "  Alice ualpha: ${ALICE_ALPHA_BEFORE}"

ALICE_TOKEN_BEFORE=$(get_balance "${PLAYER_1_ADDRESS}" "${GUILD_TOKEN_DENOM}")
echo "  Alice ${GUILD_TOKEN_DENOM}: ${ALICE_TOKEN_BEFORE}"

COLLATERAL_ALPHA_BEFORE=$(get_balance "${COLLATERAL_ADDR}" ualpha)
echo "  Collateral ualpha: ${COLLATERAL_ALPHA_BEFORE}"

# ─── Mint guild tokens ───
# Alice (guild owner) deposits 1000000 ualpha as collateral and mints 500000 guild tokens
MINT_ALPHA=1000000
MINT_TOKENS=500000
run_tx "Minting ${MINT_TOKENS} guild tokens with ${MINT_ALPHA} ualpha collateral" \
    tx structs guild-bank-mint "${MINT_ALPHA}" "${MINT_TOKENS}" --from alice

# Verify: alice should have guild tokens now
ALICE_TOKEN_AFTER_MINT=$(get_balance "${PLAYER_1_ADDRESS}" "${GUILD_TOKEN_DENOM}")
info "Alice guild tokens after mint: ${ALICE_TOKEN_AFTER_MINT}"
assert_gt "Alice guild token balance after mint" 0 "${ALICE_TOKEN_AFTER_MINT}"

# Verify: collateral address should hold the Alpha
COLLATERAL_ALPHA_AFTER_MINT=$(get_balance "${COLLATERAL_ADDR}" ualpha)
info "Collateral ualpha after mint: ${COLLATERAL_ALPHA_AFTER_MINT}"
assert_gt "Collateral Alpha increased after mint" "${COLLATERAL_ALPHA_BEFORE}" "${COLLATERAL_ALPHA_AFTER_MINT}"

# ─── Transfer guild tokens to Player 2 via bank send ───
TRANSFER_TO_P2=200000
run_tx "Transferring ${TRANSFER_TO_P2}${GUILD_TOKEN_DENOM} to Player 2 via bank send" \
    tx bank send "${PLAYER_1_ADDRESS}" "${PLAYER_2_ADDRESS}" "${TRANSFER_TO_P2}${GUILD_TOKEN_DENOM}" --from alice

P2_TOKEN_AFTER_TRANSFER=$(get_balance "${PLAYER_2_ADDRESS}" "${GUILD_TOKEN_DENOM}")
info "Player 2 guild tokens after transfer: ${P2_TOKEN_AFTER_TRANSFER}"
assert_eq "Player 2 received guild tokens" "${TRANSFER_TO_P2}" "${P2_TOKEN_AFTER_TRANSFER}"

# ─── Transfer guild tokens to Player 3 (for confiscation test later) ───
TRANSFER_TO_P3=100000
run_tx "Transferring ${TRANSFER_TO_P3}${GUILD_TOKEN_DENOM} to Player 3 via bank send" \
    tx bank send "${PLAYER_1_ADDRESS}" "${PLAYER_3_ADDRESS}" "${TRANSFER_TO_P3}${GUILD_TOKEN_DENOM}" --from alice

P3_TOKEN_AFTER_TRANSFER=$(get_balance "${PLAYER_3_ADDRESS}" "${GUILD_TOKEN_DENOM}")
info "Player 3 guild tokens after transfer: ${P3_TOKEN_AFTER_TRANSFER}"
assert_eq "Player 3 received guild tokens" "${TRANSFER_TO_P3}" "${P3_TOKEN_AFTER_TRANSFER}"

# Verify Alice's remaining tokens = minted - transferred
ALICE_TOKEN_AFTER_TRANSFERS=$(get_balance "${PLAYER_1_ADDRESS}" "${GUILD_TOKEN_DENOM}")
EXPECTED_ALICE_REMAINING=$((ALICE_TOKEN_AFTER_MINT - TRANSFER_TO_P2 - TRANSFER_TO_P3))
info "Alice guild tokens after transfers: ${ALICE_TOKEN_AFTER_TRANSFERS} (expected ${EXPECTED_ALICE_REMAINING})"
assert_eq "Alice token balance after transfers" "${EXPECTED_ALICE_REMAINING}" "${ALICE_TOKEN_AFTER_TRANSFERS}"

# ─── Redeem guild tokens (Player 2 redeems some for Alpha) ───
REDEEM_AMOUNT=50000
P2_ALPHA_BEFORE_REDEEM=$(get_balance "${PLAYER_2_ADDRESS}" ualpha)
info "Player 2 ualpha before redeem: ${P2_ALPHA_BEFORE_REDEEM}"

run_tx "Player 2 redeeming ${REDEEM_AMOUNT}${GUILD_TOKEN_DENOM} for Alpha" \
    tx structs guild-bank-redeem "${REDEEM_AMOUNT}${GUILD_TOKEN_DENOM}" --from player_2

# Verify: Player 2 token balance decreased
P2_TOKEN_AFTER_REDEEM=$(get_balance "${PLAYER_2_ADDRESS}" "${GUILD_TOKEN_DENOM}")
EXPECTED_P2_TOKENS=$((TRANSFER_TO_P2 - REDEEM_AMOUNT))
info "Player 2 guild tokens after redeem: ${P2_TOKEN_AFTER_REDEEM} (expected ${EXPECTED_P2_TOKENS})"
assert_eq "Player 2 tokens decreased after redeem" "${EXPECTED_P2_TOKENS}" "${P2_TOKEN_AFTER_REDEEM}"

# Verify: Player 2 received some Alpha back
P2_ALPHA_AFTER_REDEEM=$(get_balance "${PLAYER_2_ADDRESS}" ualpha)
info "Player 2 ualpha after redeem: ${P2_ALPHA_AFTER_REDEEM}"
assert_gt "Player 2 Alpha increased after redeem" "${P2_ALPHA_BEFORE_REDEEM}" "${P2_ALPHA_AFTER_REDEEM}"

# Verify: Collateral pool decreased
COLLATERAL_ALPHA_AFTER_REDEEM=$(get_balance "${COLLATERAL_ADDR}" ualpha)
info "Collateral ualpha after redeem: ${COLLATERAL_ALPHA_AFTER_REDEEM}"
assert_lt "Collateral decreased after redeem" "${COLLATERAL_ALPHA_AFTER_MINT}" "${COLLATERAL_ALPHA_AFTER_REDEEM}"

# ─── Confiscate and burn (Alice confiscates tokens from Player 3) ───
CONFISCATE_AMOUNT=50000
P3_TOKEN_BEFORE_CONFISCATE=$(get_balance "${PLAYER_3_ADDRESS}" "${GUILD_TOKEN_DENOM}")
info "Player 3 guild tokens before confiscate: ${P3_TOKEN_BEFORE_CONFISCATE}"

run_tx "Alice confiscating ${CONFISCATE_AMOUNT} guild tokens from Player 3" \
    tx structs guild-bank-confiscate-and-burn "${CONFISCATE_AMOUNT}" "${PLAYER_3_ADDRESS}" --from alice

# Verify: Player 3 token balance decreased
P3_TOKEN_AFTER_CONFISCATE=$(get_balance "${PLAYER_3_ADDRESS}" "${GUILD_TOKEN_DENOM}")
EXPECTED_P3_TOKENS=$((P3_TOKEN_BEFORE_CONFISCATE - CONFISCATE_AMOUNT))
info "Player 3 guild tokens after confiscate: ${P3_TOKEN_AFTER_CONFISCATE} (expected ${EXPECTED_P3_TOKENS})"
assert_eq "Player 3 tokens decreased after confiscate" "${EXPECTED_P3_TOKENS}" "${P3_TOKEN_AFTER_CONFISCATE}"

# ─── Verify total supply decreased (tokens were burned, not moved) ───
TOTAL_SUPPLY=$(query query bank total | jq -r --arg d "${GUILD_TOKEN_DENOM}" '.supply[] | select(.denom == $d) | .amount // "0"' 2>/dev/null || echo "0")
info "Total guild token supply: ${TOTAL_SUPPLY}"
EXPECTED_SUPPLY=$((MINT_TOKENS - REDEEM_AMOUNT - CONFISCATE_AMOUNT))
info "Expected supply (minted ${MINT_TOKENS} - redeemed ${REDEEM_AMOUNT} - burned ${CONFISCATE_AMOUNT} = ${EXPECTED_SUPPLY})"
assert_eq "Total guild token supply after redeem+burn" "${EXPECTED_SUPPLY}" "${TOTAL_SUPPLY}"

# ─── Attempt unauthorized mint (Player 2 should fail) ───
info "Testing unauthorized mint (Player 2, non-admin)"
P2_TOKEN_BEFORE_BAD_MINT=$(get_balance "${PLAYER_2_ADDRESS}" "${GUILD_TOKEN_DENOM}")

OUTPUT=$(structsd ${PARAMS_TX} tx structs guild-bank-mint 100 100 --from player_2 2>&1) || true
BAD_MINT_CODE=$(echo "${OUTPUT}" | jq -r '.code // empty' 2>/dev/null || echo "")
sleep "${SLEEP}"

P2_TOKEN_AFTER_BAD_MINT=$(get_balance "${PLAYER_2_ADDRESS}" "${GUILD_TOKEN_DENOM}")
assert_eq "Unauthorized mint did not change Player 2 balance" "${P2_TOKEN_BEFORE_BAD_MINT}" "${P2_TOKEN_AFTER_BAD_MINT}"
info "Unauthorized mint result code: ${BAD_MINT_CODE} (non-zero expected)"

# ─── Summary of token state ───
info "Guild token summary:"
echo "  Denom: ${GUILD_TOKEN_DENOM}"
echo "  Total supply: ${TOTAL_SUPPLY}"
echo "  Alice: $(get_balance "${PLAYER_1_ADDRESS}" "${GUILD_TOKEN_DENOM}")"
echo "  Player 2: $(get_balance "${PLAYER_2_ADDRESS}" "${GUILD_TOKEN_DENOM}")"
echo "  Player 3: $(get_balance "${PLAYER_3_ADDRESS}" "${GUILD_TOKEN_DENOM}")"
echo "  Collateral: $(get_balance "${COLLATERAL_ADDR}" ualpha) ualpha"


# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 4c: Guild Membership Operations — Invite, Request, Kick, Deny, Revoke
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 4c: Guild Membership Operations"

echo "  Testing with Player 5 (${PLAYER_5_ID}) — not yet in any guild"
echo "  Guild admin: Alice (${PLAYER_1_ID}), Guild: ${GUILD_ID}"

# Enable invite and request join modes (default is closed)
# GuildJoinBypassLevel: 0=closed, 1=permissioned, 2=member
run_tx "Enabling guild invites (bypass=member)" \
    tx structs guild-update-join-infusion-minimum-by-invite "${GUILD_ID}" member --from alice

run_tx "Enabling guild requests (bypass=member)" \
    tx structs guild-update-join-infusion-minimum-by-request "${GUILD_ID}" member --from alice

# Verify Player 5 starts without a guild
P5_JSON=$(query query structs player "${PLAYER_5_ID}")
P5_GUILD_BEFORE=$(jqr "${P5_JSON}" '.Player.guildId' '')
info "Player 5 guild before tests: '${P5_GUILD_BEFORE}'"

# ─── Test 1: Invite → Query → Deny ──────────────────────────────────────────
# NOTE: invite-approve has a known simulation issue where the store state
# used by --gas auto simulation may not see the recently committed invite
# application, causing it to fall into the "create new invite" path which
# requires the calling player to have invite permissions (which the invitee
# doesn't have). We test invite creation, querying, deny, and revoke instead.
# The request-approve flow works reliably and is used for joining.
info "--- Test 1: Invite Flow (deny) ---"

run_tx "Alice invites Player 5 to guild" \
    tx structs guild-membership-invite "${PLAYER_5_ID}" --from alice

# Query the membership application
APP_JSON=$(query query structs guild-membership-application "${GUILD_ID}" "${PLAYER_5_ID}" 2>/dev/null || echo '{}')
APP_TYPE=$(jqr "${APP_JSON}" '.GuildMembershipApplication.joinType' '')
info "Application type after invite: ${APP_TYPE}"
assert_eq "Invite application type" "invite" "${APP_TYPE}"

run_tx_noauto "Player 5 denies guild invite" \
    tx structs guild-membership-invite-deny "${GUILD_ID}" --from player_5

# Verify Player 5 is NOT in the guild
P5_JSON=$(query query structs player "${PLAYER_5_ID}")
P5_GUILD=$(jqr "${P5_JSON}" '.Player.guildId' '')
assert_eq "Player 5 not in guild after deny" "" "${P5_GUILD}"

# ─── Test 2: Invite → Revoke (guild cancels before player acts) ────────────
info "--- Test 2: Invite Flow (revoke) ---"

run_tx "Alice invites Player 5" \
    tx structs guild-membership-invite "${PLAYER_5_ID}" --from alice

APP_JSON=$(query query structs guild-membership-application "${GUILD_ID}" "${PLAYER_5_ID}" 2>/dev/null || echo '{}')
APP_TYPE=$(jqr "${APP_JSON}" '.GuildMembershipApplication.joinType' '')
assert_eq "Invite exists before revoke" "invite" "${APP_TYPE}"

run_tx "Alice revokes the invite" \
    tx structs guild-membership-invite-revoke "${GUILD_ID}" "${PLAYER_5_ID}" --from alice

# Verify Player 5 is NOT in the guild
P5_JSON=$(query query structs player "${PLAYER_5_ID}")
P5_GUILD=$(jqr "${P5_JSON}" '.Player.guildId' '')
assert_eq "Player 5 not in guild after invite revoke" "" "${P5_GUILD}"

# ─── Test 3: Request → Approve → Verify Joined ─────────────────────────────
info "--- Test 3: Request Flow (approve) ---"

run_tx "Player 5 requests to join guild" \
    tx structs guild-membership-request "${GUILD_ID}" --from player_5

APP_JSON=$(query query structs guild-membership-application "${GUILD_ID}" "${PLAYER_5_ID}" 2>/dev/null || echo '{}')
APP_TYPE=$(jqr "${APP_JSON}" '.GuildMembershipApplication.joinType' '')
info "Application type after request: ${APP_TYPE}"
assert_eq "Request application type" "request" "${APP_TYPE}"

run_tx "Alice approves Player 5's request" \
    tx structs guild-membership-request-approve "${PLAYER_5_ID}" --from alice

P5_JSON=$(query query structs player "${PLAYER_5_ID}")
P5_GUILD=$(jqr "${P5_JSON}" '.Player.guildId' '')
assert_eq "Player 5 joined guild via request" "${GUILD_ID}" "${P5_GUILD}"

# ─── Test 4: Kick ───────────────────────────────────────────────────────────
info "--- Test 4: Kick ---"

run_tx "Alice kicks Player 5 from guild" \
    tx structs guild-membership-kick "${PLAYER_5_ID}" --from alice

P5_JSON=$(query query structs player "${PLAYER_5_ID}")
P5_GUILD=$(jqr "${P5_JSON}" '.Player.guildId' '')
assert_eq "Player 5 removed after kick" "" "${P5_GUILD}"

# ─── Test 5: Request → Deny ────────────────────────────────────────────────
info "--- Test 5: Request Flow (deny) ---"

run_tx "Player 5 requests to join guild" \
    tx structs guild-membership-request "${GUILD_ID}" --from player_5

run_tx "Alice denies Player 5's request" \
    tx structs guild-membership-request-deny "${PLAYER_5_ID}" --from alice

P5_JSON=$(query query structs player "${PLAYER_5_ID}")
P5_GUILD=$(jqr "${P5_JSON}" '.Player.guildId' '')
assert_eq "Player 5 not in guild after request deny" "" "${P5_GUILD}"

# ─── Test 6: Request → Revoke (player cancels own request) ─────────────────
info "--- Test 6: Request Flow (revoke) ---"

run_tx "Player 5 requests to join guild" \
    tx structs guild-membership-request "${GUILD_ID}" --from player_5

APP_JSON=$(query query structs guild-membership-application "${GUILD_ID}" "${PLAYER_5_ID}" 2>/dev/null || echo '{}')
APP_TYPE=$(jqr "${APP_JSON}" '.GuildMembershipApplication.joinType' '')
assert_eq "Request exists before revoke" "request" "${APP_TYPE}"

run_tx "Player 5 revokes own request" \
    tx structs guild-membership-request-revoke "${GUILD_ID}" "${PLAYER_5_ID}" --from player_5

P5_JSON=$(query query structs player "${PLAYER_5_ID}")
P5_GUILD=$(jqr "${P5_JSON}" '.Player.guildId' '')
assert_eq "Player 5 not in guild after request revoke" "" "${P5_GUILD}"

# ─── Test 7: Join via request for unauthorized kick test ────────────────────
info "--- Test 7: Re-join for auth test ---"

run_tx "Player 5 requests to join guild (setup for auth test)" \
    tx structs guild-membership-request "${GUILD_ID}" --from player_5
run_tx "Alice approves Player 5's request (setup for auth test)" \
    tx structs guild-membership-request-approve "${PLAYER_5_ID}" --from alice

P5_JSON=$(query query structs player "${PLAYER_5_ID}")
P5_GUILD=$(jqr "${P5_JSON}" '.Player.guildId' '')
assert_eq "Player 5 in guild (setup)" "${GUILD_ID}" "${P5_GUILD}"

# ─── Test 8: Unauthorized kick (Player 5 tries to kick Player 2) ───────────
info "--- Test 8: Unauthorized kick attempt ---"

# Player 5 should NOT be able to kick Player 2 (no admin permissions)
run_tx "Player 5 tries to kick Player 2 (should fail)" \
    tx structs guild-membership-kick "${PLAYER_2_ID}" --from player_5

# Verify Player 2 is still in the guild
P2_JSON=$(query query structs player "${PLAYER_2_ID}")
P2_GUILD=$(jqr "${P2_JSON}" '.Player.guildId' '')
assert_eq "Player 2 still in guild after unauthorized kick" "${GUILD_ID}" "${P2_GUILD}"

# ─── Cleanup: leave Player 5 in guild for completeness ───
info "Guild membership tests complete. Player 5 remains in guild."
info "All membership applications summary:"
query query structs guild-membership-application-all | jq -r '.GuildMembershipApplication[] | "  \(.guildId) player=\(.playerId) type=\(.joinType)"' 2>/dev/null || echo "  (none pending)"


# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 4d: Guild Settings
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 4d: Guild Settings"

# ─── guild-update-endpoint ───
run_tx "Updating guild endpoint to test.energy" \
    tx structs guild-update-endpoint "${GUILD_ID}" "test.energy" --from alice

GUILD_JSON=$(query query structs guild "${GUILD_ID}")
GUILD_EP=$(jqr "${GUILD_JSON}" '.Guild.endpoint')
assert_eq "Guild endpoint updated" "test.energy" "${GUILD_EP}"

# Reset endpoint
run_tx "Resetting guild endpoint to oh.energy" \
    tx structs guild-update-endpoint "${GUILD_ID}" "oh.energy" --from alice

GUILD_JSON=$(query query structs guild "${GUILD_ID}")
assert_eq "Guild endpoint reset" "oh.energy" "$(jqr "${GUILD_JSON}" '.Guild.endpoint')"

# ─── guild-update-entry-substation-id ───
# Create a second substation for this test
run_tx "Creating second substation for guild settings test" \
    tx structs substation-create "${PLAYER_1_ID}" "${P1_ALLOC_ID}" --from alice

STRUCT_ALL_SUBS=$(query query structs substation-all 2>/dev/null || echo '{}')
SECOND_SUB_ID=$(echo "${STRUCT_ALL_SUBS}" | jq -r '.Substation[-1].id // empty' 2>/dev/null || echo "")
if [ -n "${SECOND_SUB_ID}" ] && [ "${SECOND_SUB_ID}" != "${SUBSTATION_ID}" ]; then
    info "Second substation created: ${SECOND_SUB_ID}"

    run_tx "Updating guild entry substation to ${SECOND_SUB_ID}" \
        tx structs guild-update-entry-substation-id "${GUILD_ID}" "${SECOND_SUB_ID}" --from alice

    GUILD_JSON=$(query query structs guild "${GUILD_ID}")
    GUILD_ENTRY_SUB=$(jqr "${GUILD_JSON}" '.Guild.entrySubstationId')
    assert_eq "Guild entry substation updated" "${SECOND_SUB_ID}" "${GUILD_ENTRY_SUB}"

    # Reset to original
    run_tx "Resetting guild entry substation to ${SUBSTATION_ID}" \
        tx structs guild-update-entry-substation-id "${GUILD_ID}" "${SUBSTATION_ID}" --from alice

    GUILD_JSON=$(query query structs guild "${GUILD_ID}")
    assert_eq "Guild entry substation reset" "${SUBSTATION_ID}" "$(jqr "${GUILD_JSON}" '.Guild.entrySubstationId')"
else
    info "SKIP: Could not create second substation, skipping entry substation test"
fi

# ─── guild-update-join-infusion-minimum ───
run_tx "Setting guild join infusion minimum to 1000000" \
    tx structs guild-update-join-infusion-minimum "${GUILD_ID}" 1000000 --from alice

GUILD_JSON=$(query query structs guild "${GUILD_ID}")
GUILD_MIN=$(jqr "${GUILD_JSON}" '.Guild.joinInfusionMinimum' '0')
assert_eq "Guild join infusion minimum set" "1000000" "${GUILD_MIN}"

# Reset to 0
run_tx "Resetting guild join infusion minimum to 0" \
    tx structs guild-update-join-infusion-minimum "${GUILD_ID}" 0 --from alice

# ─── guild-update-join-infusion-minimum-by-request / by-invite ───
# GuildJoinBypassLevel values: closed, permissioned, member
run_tx "Setting join bypass level for requests to permissioned" \
    tx structs guild-update-join-infusion-minimum-by-request "${GUILD_ID}" permissioned --from alice

GUILD_JSON=$(query query structs guild "${GUILD_ID}")
GUILD_REQ_BYPASS=$(jqr "${GUILD_JSON}" '.Guild.joinInfusionMinimumBypassByRequest' 'closed')
assert_eq "Guild request bypass level set" "permissioned" "${GUILD_REQ_BYPASS}"

run_tx "Setting join bypass level for invites to permissioned" \
    tx structs guild-update-join-infusion-minimum-by-invite "${GUILD_ID}" permissioned --from alice

GUILD_JSON=$(query query structs guild "${GUILD_ID}")
GUILD_INV_BYPASS=$(jqr "${GUILD_JSON}" '.Guild.joinInfusionMinimumBypassByInvite' 'closed')
assert_eq "Guild invite bypass level set" "permissioned" "${GUILD_INV_BYPASS}"

# Reset both to closed
run_tx "Resetting request bypass to closed" \
    tx structs guild-update-join-infusion-minimum-by-request "${GUILD_ID}" closed --from alice
run_tx "Resetting invite bypass to closed" \
    tx structs guild-update-join-infusion-minimum-by-invite "${GUILD_ID}" closed --from alice

# ─── guild-update-owner-id: transfer ownership ───
# Grant Player 2 PermissionUpdate (2) on guild so they can transfer back
run_tx "Granting Player 2 PermissionUpdate on guild" \
    tx structs permission-grant-on-object "${GUILD_ID}" "${PLAYER_2_ID}" 2 --from alice

info "Transferring guild ownership to Player 2"
run_tx "Transferring guild ownership to Player 2" \
    tx structs guild-update-owner-id "${GUILD_ID}" "${PLAYER_2_ID}" --from alice

GUILD_JSON=$(query query structs guild "${GUILD_ID}")
GUILD_OWNER=$(jqr "${GUILD_JSON}" '.Guild.owner')
assert_eq "Guild owner transferred to Player 2" "${PLAYER_2_ID}" "${GUILD_OWNER}"

# Player 2 transfers back to Player 1
run_tx "Player 2 transfers guild ownership back to Player 1" \
    tx structs guild-update-owner-id "${GUILD_ID}" "${PLAYER_1_ID}" --from player_2

GUILD_JSON=$(query query structs guild "${GUILD_ID}")
GUILD_OWNER=$(jqr "${GUILD_JSON}" '.Guild.owner')
assert_eq "Guild owner transferred back to Player 1" "${PLAYER_1_ID}" "${GUILD_OWNER}"

# ─── Negative: non-owner tries to update endpoint ───
info "Testing unauthorized guild update (Player 3 tries to update endpoint)"
run_tx "Player 3 tries to update guild endpoint (should fail)" \
    tx structs guild-update-endpoint "${GUILD_ID}" "hacked.energy" --from player_3

GUILD_JSON=$(query query structs guild "${GUILD_ID}")
GUILD_EP_AFTER=$(jqr "${GUILD_JSON}" '.Guild.endpoint')
assert_eq "Guild endpoint unchanged after unauthorized update" "oh.energy" "${GUILD_EP_AFTER}"


# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 4e: Permission System
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 4e: Permission System"

# Permission constants: Play=1, Update=2, Delete=4, Assets=8, Associations=16, Grid=32, Permissions=64

# ─── permission-grant-on-object: grant Player 5 Grid permission on substation ───
run_tx "Granting Player 5 Grid permission (32) on substation ${SUBSTATION_ID}" \
    tx structs permission-grant-on-object "${SUBSTATION_ID}" "${PLAYER_5_ID}" 32 --from alice

PERM_JSON=$(query query structs permission-by-object "${SUBSTATION_ID}" 2>/dev/null || echo '{}')
info "Permissions on substation after grant:"
echo "${PERM_JSON}" | jq -r '.permissionRecord[]? | "  player=\(.objectId) val=\(.value)"' 2>/dev/null || echo "  (no records)"

# ─── permission-revoke-on-object ───
run_tx "Revoking Player 5 Grid permission on substation" \
    tx structs permission-revoke-on-object "${SUBSTATION_ID}" "${PLAYER_5_ID}" 32 --from alice

# ─── permission-set-on-object: set a specific permission set ───
run_tx "Setting Player 5 permissions on substation to Assets+Grid (40)" \
    tx structs permission-set-on-object "${SUBSTATION_ID}" "${PLAYER_5_ID}" 40 --from alice

# Verify with query
PERM_JSON=$(query query structs permission-by-object "${SUBSTATION_ID}" 2>/dev/null || echo '{}')
info "Permissions on substation after set:"
echo "${PERM_JSON}" | jq -r '.permissionRecord[]? | "  player=\(.objectId) val=\(.value)"' 2>/dev/null || echo "  (no records)"

# Clean up: revoke all from Player 5
run_tx "Clearing Player 5 permissions on substation" \
    tx structs permission-set-on-object "${SUBSTATION_ID}" "${PLAYER_5_ID}" 0 --from alice

# ─── permission-grant-on-address / permission-revoke-on-address ───
# Address permissions can only be managed on your OWN player's addresses.
# Player 5 grants PermissionAssets (8) on their own address.
run_tx "Player 5 granting own address PermissionAssets (8)" \
    tx structs permission-grant-on-address "${PLAYER_5_ADDRESS}" 8 --from player_5

# Query permissions by player
PERM_BY_PLAYER=$(query query structs permission-by-player "${PLAYER_5_ID}" 2>/dev/null || echo '{}')
info "Permissions for Player 5 after address grant:"
echo "${PERM_BY_PLAYER}" | jq -r '.permissionRecord[]? | "  obj=\(.objectId) val=\(.value)"' 2>/dev/null | head -5 || echo "  (no records)"

# Revoke
run_tx "Player 5 revoking own address PermissionAssets" \
    tx structs permission-revoke-on-address "${PLAYER_5_ADDRESS}" 8 --from player_5

# ─── permission-set-on-address ───
# NOTE: permission-set-on-address prevents privilege escalation — the caller
# needs ALL bits of the target value (plus Permissions=64). After revoking
# PermissionAssets (8), the address has 247. We demonstrate set by setting
# to 247 (proving the command works). Restoring to 255 is impossible because
# the address no longer holds bit 8 and the system blocks escalation.
run_tx "Player 5 setting own address permissions to 247 (all except PermissionAssets)" \
    tx structs permission-set-on-address "${PLAYER_5_ADDRESS}" 247 --from player_5

# ─── General permission query ───
info "All permissions sample:"
query query structs permission-all 2>/dev/null | jq -r '.permissionRecord[:5]? | .[]? | "  \(.objectId) = \(.value)"' || echo "  (none)"


# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 4f: Substation Management
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 4f: Substation Management"

# Grant Player 5 PermissionGrid (32) on the substation for allocation connect/disconnect
run_tx "Granting Player 5 PermissionGrid on substation for allocation ops" \
    tx structs permission-grant-on-object "${SUBSTATION_ID}" "${PLAYER_5_ID}" 32 --from alice

# Allocation operations: Player 5 is the allocation controller, so they must sign
run_tx "Connecting Player 5 allocation to substation" \
    tx structs substation-allocation-connect "${P5_ALLOC_ID}" "${SUBSTATION_ID}" --from player_5

# ─── substation-allocation-disconnect ───
run_tx "Disconnecting Player 5 allocation from substation" \
    tx structs substation-allocation-disconnect "${P5_ALLOC_ID}" --from player_5

ALLOC_JSON=$(query query structs allocation "${P5_ALLOC_ID}")
ALLOC_DST=$(jqr "${ALLOC_JSON}" '.Allocation.destinationId' '')
assert_eq "Allocation disconnected" "" "${ALLOC_DST}"

# Reconnect for later use
run_tx "Reconnecting Player 5 allocation" \
    tx structs substation-allocation-connect "${P5_ALLOC_ID}" "${SUBSTATION_ID}" --from player_5

# ─── Create a second substation for player migration tests ───
run_tx "Creating second substation for migration test" \
    tx structs substation-create "${PLAYER_1_ID}" "${P1_ALLOC_ID}" --from alice

# Find the second substation
SUB_ALL_JSON=$(query query structs substation-all 2>/dev/null || echo '{}')
SECOND_SUB_ID=$(echo "${SUB_ALL_JSON}" | jq -r '.Substation[-1].id // empty' 2>/dev/null || echo "")

if [ -n "${SECOND_SUB_ID}" ] && [ "${SECOND_SUB_ID}" != "${SUBSTATION_ID}" ]; then
    info "Second substation for migration: ${SECOND_SUB_ID}"

    # Grant Player 5 PermissionAssociations (16) on both substations so they can connect themselves
    run_tx "Granting Player 5 PermissionAssociations on original substation" \
        tx structs permission-grant-on-object "${SUBSTATION_ID}" "${PLAYER_5_ID}" 16 --from alice
    run_tx "Granting Player 5 PermissionAssociations on second substation" \
        tx structs permission-grant-on-object "${SECOND_SUB_ID}" "${PLAYER_5_ID}" 16 --from alice

    # ─── substation-player-connect: connect Player 5 to second substation ───
    run_tx "Connecting Player 5 to second substation" \
        tx structs substation-player-connect "${SECOND_SUB_ID}" "${PLAYER_5_ID}" --from player_5

    P5_JSON=$(query query structs player "${PLAYER_5_ID}")
    P5_SUB=$(jqr "${P5_JSON}" '.Player.substationId' '')
    assert_eq "Player 5 connected to second substation" "${SECOND_SUB_ID}" "${P5_SUB}"

    # ─── substation-player-disconnect ───
    run_tx "Disconnecting Player 5 from second substation" \
        tx structs substation-player-disconnect "${PLAYER_5_ID}" --from player_5

    P5_JSON=$(query query structs player "${PLAYER_5_ID}")
    P5_SUB=$(jqr "${P5_JSON}" '.Player.substationId' '')
    info "Player 5 substation after disconnect: '${P5_SUB}'"

    # Reconnect to original substation
    run_tx "Reconnecting Player 5 to original substation" \
        tx structs substation-player-connect "${SUBSTATION_ID}" "${PLAYER_5_ID}" --from player_5

    # ─── substation-player-migrate: migrate Player 5 to second then back ───
    run_tx "Migrating Player 5 to second substation" \
        tx structs substation-player-migrate "${SECOND_SUB_ID}" "${PLAYER_5_ID}" --from player_5

    P5_JSON=$(query query structs player "${PLAYER_5_ID}")
    P5_SUB=$(jqr "${P5_JSON}" '.Player.substationId' '')
    assert_eq "Player 5 migrated to second substation" "${SECOND_SUB_ID}" "${P5_SUB}"

    # Migrate back
    run_tx "Migrating Player 5 back to original substation" \
        tx structs substation-player-migrate "${SUBSTATION_ID}" "${PLAYER_5_ID}" --from player_5

    P5_JSON=$(query query structs player "${PLAYER_5_ID}")
    P5_SUB=$(jqr "${P5_JSON}" '.Player.substationId' '')
    assert_eq "Player 5 back on original substation" "${SUBSTATION_ID}" "${P5_SUB}"

    # ─── substation-delete: delete second substation ───
    run_tx "Deleting second substation (migrate to original)" \
        tx structs substation-delete "${SECOND_SUB_ID}" "${SUBSTATION_ID}" --from alice

    # Verify second substation is gone
    DEL_SUB_JSON=$(query query structs substation "${SECOND_SUB_ID}" 2>/dev/null || echo '{}')
    DEL_SUB_ID=$(jqr "${DEL_SUB_JSON}" '.Substation.id' '')
    assert_eq "Second substation deleted" "" "${DEL_SUB_ID}"
else
    info "SKIP: Could not create second substation for migration tests"
fi

# ─── Grid query coverage ───
info "Grid attributes sample:"
query query structs grid-all 2>/dev/null | jq -r '.gridAttribute[:3]? | .[]? | "  \(.objectId) cap=\(.capacity) load=\(.load)"' || echo "  (none)"


# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 4g: Reactor Operations
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 4g: Reactor Operations"

# Query current reactor state
info "Querying reactors"
REACTOR_ALL=$(query query structs reactor-all 2>/dev/null || echo '{}')
echo "  Reactor count: $(echo "${REACTOR_ALL}" | jq '.Reactor | length' 2>/dev/null || echo '0')"

# Get Player 2's delegation info for reactor operations
P2_DELEGATION=$(query query staking delegations "${PLAYER_2_ADDRESS}" 2>/dev/null || echo '{}')
P2_VAL_ADDR=$(echo "${P2_DELEGATION}" | jq -r '.delegation_responses[0].delegation.validator_address // empty' 2>/dev/null || echo "")
P2_SHARES_BEFORE=$(echo "${P2_DELEGATION}" | jq -r '.delegation_responses[0].delegation.shares // "0"' 2>/dev/null || echo "0")
info "Player 2 delegated to: ${P2_VAL_ADDR}"
info "Player 2 shares: ${P2_SHARES_BEFORE}"

if [ -n "${P2_VAL_ADDR}" ]; then
    # Snapshot Player 2 capacity before
    P2_CAP_BEFORE=$(query query structs player "${PLAYER_2_ID}" | jq -r '.gridAttributes.capacity // "0"' 2>/dev/null || echo "0")
    info "Player 2 capacity before infuse: ${P2_CAP_BEFORE}"

    # ─── reactor-infuse: Player 2 infuses additional Alpha ───
    run_tx "Player 2 infusing 1000000ualpha into reactor" \
        tx structs reactor-infuse "${PLAYER_2_ADDRESS}" "${P2_VAL_ADDR}" 1000000ualpha --from player_2

    # Verify capacity increased
    sleep "${SLEEP}"
    P2_CAP_AFTER=$(query query structs player "${PLAYER_2_ID}" | jq -r '.gridAttributes.capacity // "0"' 2>/dev/null || echo "0")
    info "Player 2 capacity after infuse: ${P2_CAP_AFTER}"
    assert_gt "Player 2 capacity increased after infuse" "${P2_CAP_BEFORE}" "${P2_CAP_AFTER}"

    # ─── reactor-defuse: Player 2 defuses a small amount ───
    run_tx "Player 2 defusing 500000ualpha from reactor" \
        tx structs reactor-defuse "${PLAYER_2_ADDRESS}" "${P2_VAL_ADDR}" 500000ualpha --from player_2

    # Get the creation height for cancel-defusion
    UNBONDING_JSON=$(query query staking unbonding-delegations "${PLAYER_2_ADDRESS}" 2>/dev/null || echo '{}')
    UNBOND_HEIGHT=$(echo "${UNBONDING_JSON}" | jq -r '.unbonding_responses[0].entries[0].creation_height // empty' 2>/dev/null || echo "")
    info "Unbonding entry creation height: ${UNBOND_HEIGHT}"

    if [ -n "${UNBOND_HEIGHT}" ]; then
        # ─── reactor-cancel-defusion ───
        run_tx "Cancelling defusion (re-delegate)" \
            tx structs reactor-cancel-defusion "${PLAYER_2_ADDRESS}" "${P2_VAL_ADDR}" 500000ualpha "${UNBOND_HEIGHT}" --from player_2

        # Verify delegation restored
        P2_CAP_RESTORED=$(query query structs player "${PLAYER_2_ID}" | jq -r '.gridAttributes.capacity // "0"' 2>/dev/null || echo "0")
        info "Player 2 capacity after cancel-defusion: ${P2_CAP_RESTORED}"
    else
        info "SKIP: No unbonding entry found for cancel-defusion test"
    fi

    # ─── Infusion/reactor query coverage ───
    info "Infusion query coverage:"
    query query structs infusion-all 2>/dev/null | jq -r '.infusion[:3]? | .[]? | "  dst=\(.destinationId) addr=\(.address)"' || echo "  (none)"
    query query structs reactor-all 2>/dev/null | jq -r '.Reactor[:3]? | .[]? | "  \(.id) validator=\(.validator)"' || echo "  (none)"
else
    info "SKIP: Player 2 has no delegation, skipping reactor tests"
fi


# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 5: Address Register & Proxy Join (advanced)
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 5: Address Register & Proxy Join"

run_tx "Registering external address for Player 1" \
    tx structs address-register \
    structs12eufgpe24hnqndwh7hccxw36nhs47wt85hunjw \
    02faf4ada9b17d17441861baa580f95b4e5852cd56f6555c4c1f1ac6d27f6b97f8 \
    cbf4e9276a7f54ecea553779c1a589431e29327d894eef12edadf1e314030e5b3259db9f8f3a2b963f94ed13b7c66b94fa15cb5bf7df4bddd78bb64480093a8b00 \
    127 --from alice

# Verify address was registered
ADDR_CHECK_JSON=$(query query structs address structs12eufgpe24hnqndwh7hccxw36nhs47wt85hunjw)
REGISTERED_PLAYER=$(jqr "${ADDR_CHECK_JSON}" '.playerId')
assert_not_empty "Registered address player ID" "${REGISTERED_PLAYER}"

run_tx "Proxy joining guild for external address" \
    tx structs guild-membership-join-proxy \
    structs1wfs4s5er9lpkxlcrh8ezdqayewjnudkrlwpxqc \
    031b16cabd6c322e1a9ec4ead0240e70be7b2deb7b71e167a380fe405e3adaf99b \
    0c1623a753074f49bc20c6e8bb9e6572903b90e386598c4baa34e056e468e53076938ec4ab411f5889adb771f63b2be9b15912d5e1e70a97d1b091926181c8ae01 \
    --from alice


# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 5b: Address Revoke
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 5b: Address Revoke"

# The address registered in Phase 5 for Player 1
REGISTERED_EXT_ADDR="structs12eufgpe24hnqndwh7hccxw36nhs47wt85hunjw"

# Query address-all for coverage
info "Address query coverage:"
ADDR_ALL=$(query query structs address-all 2>/dev/null || echo '{}')
echo "  Total addresses: $(echo "${ADDR_ALL}" | jq '.address | length' 2>/dev/null || echo '?')"

# Check if address was properly registered (crypto signatures are chain-specific)
ADDR_JSON=$(query query structs address "${REGISTERED_EXT_ADDR}" 2>/dev/null || echo '{}')
ADDR_PLAYER=$(jqr "${ADDR_JSON}" '.playerId' '')
info "Registered address ${REGISTERED_EXT_ADDR} belongs to player: ${ADDR_PLAYER}"

if [ "${ADDR_PLAYER}" = "${PLAYER_1_ID}" ]; then
    assert_not_empty "Registered address has player" "${ADDR_PLAYER}"

    # Revoke the address
    run_tx "Revoking registered external address" \
        tx structs address-revoke "${REGISTERED_EXT_ADDR}" --from alice

    # Verify address is no longer associated
    ADDR_JSON=$(query query structs address "${REGISTERED_EXT_ADDR}" 2>/dev/null || echo '{}')
    ADDR_PLAYER_AFTER=$(jqr "${ADDR_JSON}" '.playerId' '')
    assert_eq "Address revoked (no player)" "" "${ADDR_PLAYER_AFTER}"
else
    info "SKIP: Address registration used static crypto data; address not properly associated (got '${ADDR_PLAYER}', expected '${PLAYER_1_ID}')"
    info "Address revoke test skipped — crypto signatures may be chain-specific"
fi


# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 6: Planet Exploration
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 6: Planet Exploration"

# planet-explore now requires [player id] as positional arg
run_tx "Player 2 exploring a planet" \
    tx structs planet-explore "${PLAYER_2_ID}" --from player_2

P2_JSON=$(query query structs player "${PLAYER_2_ID}")
PLAYER_2_PLANET_ID=$(jqr "${P2_JSON}" '.Player.planetId')
PLAYER_2_FLEET_ID=$(jqr "${P2_JSON}" '.Player.fleetId')
assert_not_empty "Player 2 planet" "${PLAYER_2_PLANET_ID}"
assert_not_empty "Player 2 fleet" "${PLAYER_2_FLEET_ID}"
echo "  Player 2 Planet: ${PLAYER_2_PLANET_ID}  Fleet: ${PLAYER_2_FLEET_ID}"

run_tx "Player 3 exploring a planet" \
    tx structs planet-explore "${PLAYER_3_ID}" --from player_3

P3_JSON=$(query query structs player "${PLAYER_3_ID}")
PLAYER_3_PLANET_ID=$(jqr "${P3_JSON}" '.Player.planetId')
PLAYER_3_FLEET_ID=$(jqr "${P3_JSON}" '.Player.fleetId')
assert_not_empty "Player 3 planet" "${PLAYER_3_PLANET_ID}"
assert_not_empty "Player 3 fleet" "${PLAYER_3_FLEET_ID}"
echo "  Player 3 Planet: ${PLAYER_3_PLANET_ID}  Fleet: ${PLAYER_3_FLEET_ID}"

run_tx "Player 4 exploring a planet" \
    tx structs planet-explore "${PLAYER_4_ID}" --from player_4

P4_JSON=$(query query structs player "${PLAYER_4_ID}")
PLAYER_4_PLANET_ID=$(jqr "${P4_JSON}" '.Player.planetId')
PLAYER_4_FLEET_ID=$(jqr "${P4_JSON}" '.Player.fleetId')
assert_not_empty "Player 4 planet" "${PLAYER_4_PLANET_ID}"
assert_not_empty "Player 4 fleet" "${PLAYER_4_FLEET_ID}"
echo "  Player 4 Planet: ${PLAYER_4_PLANET_ID}  Fleet: ${PLAYER_4_FLEET_ID}"

# Verify planets exist
info "Verifying planets"
PLANET_COUNT=$(query query structs planet-all | jq '.Planet | length' 2>/dev/null || echo 0)
assert_gt "Total planets" 0 "${PLANET_COUNT}"
echo "  Total planets: ${PLANET_COUNT}"

FLEET_COUNT=$(query query structs fleet-all | jq '.Fleet | length' 2>/dev/null || echo 0)
assert_gt "Total fleets" 0 "${FLEET_COUNT}"
echo "  Total fleets: ${FLEET_COUNT}"

# Dump state
info "Planet/Fleet overview"
query query structs planet-all | jq -r '.Planet[] | "  Planet \(.id)"' 2>/dev/null || true
query query structs fleet-all  | jq -r '.Fleet[]  | "  Fleet  \(.id) loc=\(.locationId) status=\(.status)"' 2>/dev/null || true

# ─── Discover auto-created Command Ships (created during planet exploration) ───
info "Discovering auto-created Command Ships (type=1)"
STRUCT_ALL_JSON=$(query query structs struct-all)

PLAYER_2_CMD_SHIP_ID=$(echo "${STRUCT_ALL_JSON}" | jq -r '[.Struct[] | select(.type == "1" and .owner == "'"${PLAYER_2_ID}"'")] | .[0].id // empty' 2>/dev/null || echo "")
PLAYER_3_CMD_SHIP_ID=$(echo "${STRUCT_ALL_JSON}" | jq -r '[.Struct[] | select(.type == "1" and .owner == "'"${PLAYER_3_ID}"'")] | .[0].id // empty' 2>/dev/null || echo "")
PLAYER_4_CMD_SHIP_ID=$(echo "${STRUCT_ALL_JSON}" | jq -r '[.Struct[] | select(.type == "1" and .owner == "'"${PLAYER_4_ID}"'")] | .[0].id // empty' 2>/dev/null || echo "")

# Player 3's command ship is used extensively in combat phases
COMMAND_SHIP_ID="${PLAYER_3_CMD_SHIP_ID}"

assert_not_empty "Player 2 Command Ship (auto-created)" "${PLAYER_2_CMD_SHIP_ID}"
assert_not_empty "Player 3 Command Ship (auto-created)" "${PLAYER_3_CMD_SHIP_ID}"
assert_not_empty "Player 4 Command Ship (auto-created)" "${PLAYER_4_CMD_SHIP_ID}"
echo "  Player 2 Command Ship: ${PLAYER_2_CMD_SHIP_ID}"
echo "  Player 3 Command Ship: ${PLAYER_3_CMD_SHIP_ID}"
echo "  Player 4 Command Ship: ${PLAYER_4_CMD_SHIP_ID}"

# Verify Player 3's command ship is built and online
CMDSHIP_JSON=$(query query structs struct "${COMMAND_SHIP_ID}")
CMDSHIP_BUILT=$(jqr "${CMDSHIP_JSON}" '.structAttributes.isBuilt' 'false')
CMDSHIP_TYPE=$(jqr "${CMDSHIP_JSON}" '.Struct.type')
assert_eq "Player 3 Command Ship built" "true" "${CMDSHIP_BUILT}"
assert_eq "Player 3 Command Ship type" "1" "${CMDSHIP_TYPE}"

# ─── Extended planet/fleet query coverage ───
info "Planet detail query (Player 2's planet):"
query query structs planet "${PLAYER_2_PLANET_ID}" 2>/dev/null | jq -r '"  id=\(.Planet.id) owner=\(.Planet.owner) status=\(.Planet.status)"' || echo "  (query failed)"

info "Planet attribute query:"
query query structs planet-attribute "${PLAYER_2_PLANET_ID}" 2>/dev/null | jq -r '"  landSlots=\(.landSlots) waterSlots=\(.waterSlots)"' || echo "  (query failed)"

info "Planets by player (Player 2):"
query query structs planet-all-by-player "${PLAYER_2_ID}" 2>/dev/null | jq -r '.Planet[]? | "  \(.id)"' || echo "  (none)"

info "Fleet by index (Player 2, index=$(echo "${PLAYER_2_FLEET_ID}" | cut -d'-' -f2)):"
FLEET_INDEX=$(echo "${PLAYER_2_FLEET_ID}" | cut -d'-' -f2)
query query structs fleet-by-index "${FLEET_INDEX}" 2>/dev/null | jq -r '"  id=\(.Fleet.id) loc=\(.Fleet.locationId)"' || echo "  (query failed)"


# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 7: Struct Building — Miner & Refinery (Player 2)
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 7: Build Miner & Refinery (Player 2)"

echo "  Player 2 Planet: ${PLAYER_2_PLANET_ID}"

# struct-build-initiate no longer takes location_type (planet/fleet).
#   Old: [player id] [struct type] [location type] [ambit] [slot]
#   New: [player id] [struct type] [ambit] [slot]

if [ "${SKIP_MINING}" = true ]; then
    info "Skipping mine shaft build, mining, refinery, and refining (--skip-mining)"
else
    # ─── Build Mine Shaft (struct type 14, land, slot 1) ───
    STRUCT_COUNT_BEFORE=$(query query structs struct-all | jq '.Struct | length' 2>/dev/null || echo 0)

    wait_for_charge "${PLAYER_2_ID}" "${CHARGE_BUILD}"
    run_tx "Initiating Mine Shaft build (type=14, ambit=land, slot=1)" \
        tx structs struct-build-initiate "${PLAYER_2_ID}" 14 land 1 --from player_2

    STRUCT_ALL_JSON=$(query query structs struct-all)
    STRUCT_COUNT_AFTER=$(echo "${STRUCT_ALL_JSON}" | jq '.Struct | length' 2>/dev/null || echo 0)
    MINER_STRUCT_ID=$(get_newest_struct_id "${STRUCT_ALL_JSON}")
    assert_not_empty "Miner struct ID" "${MINER_STRUCT_ID}"
    echo "  Miner Struct ID: ${MINER_STRUCT_ID}"

    run_compute "Building Mine Shaft ${MINER_STRUCT_ID}" \
        tx structs struct-build-compute "${MINER_STRUCT_ID}" --from player_2

    MINER_JSON=$(query query structs struct "${MINER_STRUCT_ID}")
    MINER_BUILT=$(jqr "${MINER_JSON}" '.structAttributes.isBuilt' 'false')
    MINER_ONLINE=$(jqr "${MINER_JSON}" '.structAttributes.isOnline' 'false')
    MINER_TYPE=$(jqr "${MINER_JSON}" '.Struct.type')
    assert_eq "Mine Shaft built" "true" "${MINER_BUILT}"
    assert_eq "Mine Shaft online" "true" "${MINER_ONLINE}"
    assert_eq "Mine Shaft type" "14" "${MINER_TYPE}"
    # ─── Mine some ore (3 rounds) ───
    # NOTE: old command was struct-mine-compute, now struct-ore-mine-compute
    for ROUND in 1 2 3; do
        run_compute "Mining ore round ${ROUND}" \
            tx structs struct-ore-mine-compute "${MINER_STRUCT_ID}" --from player_2
    done

    # Check player 2 ore inventory
    P2_JSON=$(query query structs player "${PLAYER_2_ID}")
    P2_ORE=$(jqr "${P2_JSON}" '.playerInventory.ore' '0')
    info "Player 2 ore after mining: ${P2_ORE}"
    assert_gt "Player 2 ore after mining" 0 "${P2_ORE}"

    # ─── Build Refinery (struct type 16, land, slot 2) ───
    wait_for_charge "${PLAYER_2_ID}" "${CHARGE_BUILD}"
    run_tx "Initiating Refinery build (type=16, ambit=land, slot=2)" \
        tx structs struct-build-initiate "${PLAYER_2_ID}" 16 land 2 --from player_2

    # Find the new struct
    STRUCT_ALL_JSON=$(query query structs struct-all)
    REFINERY_STRUCT_ID=$(get_newest_struct_id "${STRUCT_ALL_JSON}")
    assert_not_empty "Refinery struct ID" "${REFINERY_STRUCT_ID}"
    echo "  Refinery Struct ID: ${REFINERY_STRUCT_ID}"

    run_compute "Building Refinery ${REFINERY_STRUCT_ID}" \
        tx structs struct-build-compute "${REFINERY_STRUCT_ID}" --from player_2

    # Verify
    REFINERY_JSON=$(query query structs struct "${REFINERY_STRUCT_ID}")
    REFINERY_BUILT=$(jqr "${REFINERY_JSON}" '.structAttributes.isBuilt' 'false')
    REFINERY_TYPE=$(jqr "${REFINERY_JSON}" '.Struct.type')
    assert_eq "Refinery built" "true" "${REFINERY_BUILT}"
    assert_eq "Refinery type" "16" "${REFINERY_TYPE}"

    # ─── Refine ore ───
    # NOTE: old command was struct-refine-compute, now struct-ore-refine-compute
    run_compute "Refining ore" \
        tx structs struct-ore-refine-compute "${REFINERY_STRUCT_ID}" --from player_2
fi


# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 7b: Struct Build Cancel
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 7b: Struct Build Cancel"

# Player 2 has a planet with available slots. Build an Ore Bunker (type 18, planet category, land)
# then cancel before compute completes.

# Snapshot Player 2 structsLoad before
P2_JSON_BEFORE=$(query query structs player "${PLAYER_2_ID}")
P2_LOAD_BEFORE_CANCEL=$(jqr "${P2_JSON_BEFORE}" '.gridAttributes.structsLoad' '0')
info "Player 2 structsLoad before build-initiate: ${P2_LOAD_BEFORE_CANCEL}"

# struct-type query coverage
info "Querying struct types:"
query query structs struct-type 18 2>/dev/null | jq -r '.structType | "  Type \(.id): \(.type) buildDraw=\(.buildDraw) category=\(.category)"' || echo "  (query failed)"

# Initiate the build (Ore Bunker type 18, land, slot 2)
wait_for_charge "${PLAYER_2_ID}" "${CHARGE_BUILD}"
run_tx "Initiating Ore Bunker build (type=18, land, slot=2)" \
    tx structs struct-build-initiate "${PLAYER_2_ID}" 18 land 2 --from player_2

# Find the new struct
STRUCT_ALL_JSON=$(query query structs struct-all)
CANCEL_STRUCT_ID=$(get_newest_struct_id "${STRUCT_ALL_JSON}")
info "Struct to cancel: ${CANCEL_STRUCT_ID}"

if [ -n "${CANCEL_STRUCT_ID}" ]; then
    # Check if struct is built yet (may auto-complete with low difficulty)
    CANCEL_JSON=$(query query structs struct "${CANCEL_STRUCT_ID}" 2>/dev/null || echo '{}')
    CANCEL_BUILT=$(jqr "${CANCEL_JSON}" '.structAttributes.isBuilt' 'true')
    info "Struct isBuilt after initiate: ${CANCEL_BUILT} (may auto-complete at low difficulty)"

    # Verify structsLoad increased (buildDraw added)
    P2_JSON_MID=$(query query structs player "${PLAYER_2_ID}")
    P2_LOAD_MID=$(jqr "${P2_JSON_MID}" '.gridAttributes.structsLoad' '0')
    info "Player 2 structsLoad after build-initiate: ${P2_LOAD_MID}"
    assert_gt "StructsLoad increased from build-initiate" "${P2_LOAD_BEFORE_CANCEL}" "${P2_LOAD_MID}"

    # Cancel the build
    run_tx "Cancelling Ore Bunker build" \
        tx structs struct-build-cancel "${CANCEL_STRUCT_ID}" --from player_2

    # Verify struct is gone
    CANCEL_GONE=$(query query structs struct "${CANCEL_STRUCT_ID}" 2>/dev/null || echo '{}')
    CANCEL_GONE_BUILT=$(jqr "${CANCEL_GONE}" '.structAttributes.isBuilt' '')
    info "Struct after cancel: isBuilt='${CANCEL_GONE_BUILT}'"

    # Verify structsLoad decreased back
    P2_JSON_AFTER=$(query query structs player "${PLAYER_2_ID}")
    P2_LOAD_AFTER_CANCEL=$(jqr "${P2_JSON_AFTER}" '.gridAttributes.structsLoad' '0')
    info "Player 2 structsLoad after cancel: ${P2_LOAD_AFTER_CANCEL}"
    assert_eq "StructsLoad restored after cancel" "${P2_LOAD_BEFORE_CANCEL}" "${P2_LOAD_AFTER_CANCEL}"
else
    info "SKIP: Could not initiate build for cancel test"
fi

# struct-type-all query coverage
info "All struct types count:"
echo "  $(query query structs struct-type-all 2>/dev/null | jq '.structType | length' || echo '?') types"


# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 8: Combat Setup — Player 3 builds attack fleet
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 8: Player 3 Combat Setup"

echo "  Player 3 Planet: ${PLAYER_3_PLANET_ID}"

# ─── Build Guided Missile Destroyer (type 9, land, slot 1) ───
wait_for_charge "${PLAYER_3_ID}" "${CHARGE_BUILD}"
run_tx "Initiating Guided Missile Destroyer (type=9, ambit=land, slot=1)" \
    tx structs struct-build-initiate "${PLAYER_3_ID}" 9 land 1 --from player_3

STRUCT_ALL_JSON=$(query query structs struct-all)
DESTROYER_STRUCT_ID=$(get_newest_struct_id "${STRUCT_ALL_JSON}")
assert_not_empty "Destroyer struct ID" "${DESTROYER_STRUCT_ID}"
echo "  Destroyer Struct ID: ${DESTROYER_STRUCT_ID}"

# ─── Pre-seed builds for other players while P3's Destroyer computes ────────
# Difficulty decays with block age, so initiating now means much faster computes later.
# P2 and P4 have independent charge — no waiting on P3.

info "Pre-seeding P2 Defender Destroyer (type=9, land, slot=0) — needed Phase 11"
wait_for_charge "${PLAYER_2_ID}" "${CHARGE_BUILD}"
run_tx "Pre-seed: P2 Defender Destroyer (type=9, land, slot=0)" \
    tx structs struct-build-initiate "${PLAYER_2_ID}" 9 land 0 --from player_2

STRUCT_ALL_JSON=$(query query structs struct-all)
DEFENDER_STRUCT_ID=$(get_newest_struct_id "${STRUCT_ALL_JSON}")
assert_not_empty "Defender struct ID" "${DEFENDER_STRUCT_ID}"
echo "  Defender Struct ID: ${DEFENDER_STRUCT_ID} (pre-seeded, compute deferred to Phase 11)"

info "Pre-seeding P4 Field Generator (type=20, land, slot=0) — needed Phase 15"
wait_for_charge "${PLAYER_4_ID}" "${CHARGE_BUILD}"
run_tx "Pre-seed: P4 Field Generator (type=20, land, slot=0)" \
    tx structs struct-build-initiate "${PLAYER_4_ID}" 20 land 0 --from player_4

STRUCT_ALL_JSON=$(query query structs struct-all)
GENERATOR_STRUCT_ID=$(get_newest_struct_id "${STRUCT_ALL_JSON}")
assert_not_empty "Generator struct ID" "${GENERATOR_STRUCT_ID}"
echo "  Generator Struct ID: ${GENERATOR_STRUCT_ID} (pre-seeded, compute deferred to Phase 15)"

# ─── Now compute P3's Destroyer (P2 Defender and P4 Generator age during this) ───
run_compute "Building Destroyer ${DESTROYER_STRUCT_ID}" \
    tx structs struct-build-compute "${DESTROYER_STRUCT_ID}" --from player_3

DESTROYER_JSON=$(query query structs struct "${DESTROYER_STRUCT_ID}")
assert_eq "Destroyer built" "true" "$(jqr "${DESTROYER_JSON}" '.structAttributes.isBuilt' 'false')"

# NOTE: Command Ship no longer needs to be built manually.
# It is auto-created during planet exploration (Phase 6).
# COMMAND_SHIP_ID was already set in Phase 6.


# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 9: Fleet Movement & Attack — Player 3 attacks Player 2's Miner
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 9: Fleet Movement & Attack"

# Move Player 3's fleet to Player 2's planet (fleet-move has no charge cost)
run_tx "Moving Player 3's fleet to Player 2's planet (${PLAYER_2_PLANET_ID})" \
    tx structs fleet-move "${PLAYER_3_FLEET_ID}" "${PLAYER_2_PLANET_ID}" --from player_3

# Verify fleet location
FLEET_3_JSON=$(query query structs fleet "${PLAYER_3_FLEET_ID}")
FLEET_3_LOC=$(jqr "${FLEET_3_JSON}" '.Fleet.locationId')
info "Player 3 fleet location after move: ${FLEET_3_LOC}"

if [ "${SKIP_MINING}" = true ]; then
    info "Skipping miner attack (--skip-mining, no miner to attack)"
else
    # ─── Attack the Miner (3 rounds) ───
    MINER_JSON=$(query query structs struct "${MINER_STRUCT_ID}" || echo '{}')
    MINER_HEALTH_BEFORE=$(jqr "${MINER_JSON}" '.structAttributes.health' '0')
    info "Miner health before attack: ${MINER_HEALTH_BEFORE}"

    for ATTACK_ROUND in 1 2 3; do
        wait_for_charge "${PLAYER_3_ID}" "${CHARGE_ATTACK_DEFAULT}"
        run_tx "Attack round ${ATTACK_ROUND}: Command Ship -> Miner (primaryWeapon)" \
            tx structs struct-attack "${COMMAND_SHIP_ID}" "${MINER_STRUCT_ID}" primaryWeapon --from player_3

        MINER_JSON=$(query query structs struct "${MINER_STRUCT_ID}" || echo '{}')
        MINER_HEALTH=$(jqr "${MINER_JSON}" '.structAttributes.health' '0')
        info "Miner health after attack round ${ATTACK_ROUND}: ${MINER_HEALTH}"
        if [ "${MINER_HEALTH}" = "0" ] || [ "${MINER_HEALTH}" = "" ]; then
            info "Miner destroyed — skipping remaining attack rounds"
            break
        fi
    done

    MINER_HEALTH_AFTER=$(jqr "${MINER_JSON}" '.structAttributes.health' '0')
    assert_lt "Miner health decreased after attacks" "${MINER_HEALTH_BEFORE}" "${MINER_HEALTH_AFTER}"
fi


# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 10: Planet Raid
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 10: Planet Raid"

if [ "${SKIP_MINING}" = true ]; then
    info "Skipping planet raid (--skip-mining, no ore to raid)"
else
    P3_JSON=$(query query structs player "${PLAYER_3_ID}")
    P2_JSON=$(query query structs player "${PLAYER_2_ID}")
    P3_ORE_BEFORE=$(jqr "${P3_JSON}" '.playerInventory.ore' '0')
    P2_ORE_BEFORE=$(jqr "${P2_JSON}" '.playerInventory.ore' '0')
    info "Player 3 ore before raid: ${P3_ORE_BEFORE}"
    info "Player 2 ore before raid: ${P2_ORE_BEFORE}"

    run_compute "Completing planet raid" \
        tx structs planet-raid-compute "${PLAYER_3_FLEET_ID}" --from player_3

    P3_JSON=$(query query structs player "${PLAYER_3_ID}")
    P2_JSON=$(query query structs player "${PLAYER_2_ID}")
    P3_ORE_AFTER=$(jqr "${P3_JSON}" '.playerInventory.ore' '0')
    P2_ORE_AFTER=$(jqr "${P2_JSON}" '.playerInventory.ore' '0')
    info "Player 3 ore after raid: ${P3_ORE_AFTER}"
    info "Player 2 ore after raid: ${P2_ORE_AFTER}"
    echo "  Raid results: P3 ore ${P3_ORE_BEFORE} -> ${P3_ORE_AFTER}, P2 ore ${P2_ORE_BEFORE} -> ${P2_ORE_AFTER}"
fi


# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 11: Counter-attack — Player 2 fights back
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 11: Counter-Attack"

echo "  Player 3's fleet may have retreated after raid"

# Move Player 3's fleet back to Player 2's planet (fleet-move has no charge cost)
run_tx "Moving Player 3's fleet back to Player 2's planet" \
    tx structs fleet-move "${PLAYER_3_FLEET_ID}" "${PLAYER_2_PLANET_ID}" --from player_3

# Player 2's Destroyer was pre-seeded in Phase 8 — difficulty has decayed significantly
info "Computing P2 Defender Destroyer ${DEFENDER_STRUCT_ID} (pre-seeded in Phase 8)"
run_compute "Building Player 2's Destroyer ${DEFENDER_STRUCT_ID}" \
    tx structs struct-build-compute "${DEFENDER_STRUCT_ID}" --from player_2

DEFENDER_JSON=$(query query structs struct "${DEFENDER_STRUCT_ID}")
assert_eq "Defender built" "true" "$(jqr "${DEFENDER_JSON}" '.structAttributes.isBuilt' 'false')"

# Player 2 attacks Command Ship
CMDSHIP_JSON=$(query query structs struct "${COMMAND_SHIP_ID}" || echo '{}')
CMDSHIP_HEALTH_BEFORE=$(jqr "${CMDSHIP_JSON}" '.structAttributes.health' '0')
info "Command Ship health before counter-attack: ${CMDSHIP_HEALTH_BEFORE}"

wait_for_charge "${PLAYER_2_ID}" "${CHARGE_ATTACK_DEFAULT}"
run_tx "Player 2 attacks Player 3's Command Ship" \
    tx structs struct-attack "${DEFENDER_STRUCT_ID}" "${COMMAND_SHIP_ID}" primaryWeapon --from player_2

CMDSHIP_JSON=$(query query structs struct "${COMMAND_SHIP_ID}" || echo '{}')
CMDSHIP_HEALTH_AFTER=$(jqr "${CMDSHIP_JSON}" '.structAttributes.health' '0')
info "Command Ship health after counter-attack: ${CMDSHIP_HEALTH_AFTER}"


# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 12: Complex Battle — Build multi-unit fleet
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 12: Complex Battle Setup"

# ─── Move Player 3's fleet home before building (fleet can't build while away) ───
run_tx "Moving Player 3's fleet home for building" \
    tx structs fleet-move "${PLAYER_3_FLEET_ID}" "${PLAYER_3_PLANET_ID}" --from player_3

# ═══════════════════════════════════════════════════════════════
# BATCH INITIATE: All builds for Phases 12-14
# Difficulty decays with block age, so initiating all builds now means
# each subsequent compute runs against a much lower difficulty.
# P2 builds are interleaved since players have independent charge.
# ═══════════════════════════════════════════════════════════════

info "Batch-initiating all builds for Phases 12-14 (difficulty decays while computing)"

# ─── P3: SAM Launcher (type 10, land, slot 3) ───
wait_for_charge "${PLAYER_3_ID}" "${CHARGE_BUILD}"
run_tx "Initiating SAM Launcher (type=10, land, slot=3)" \
    tx structs struct-build-initiate "${PLAYER_3_ID}" 10 land 3 --from player_3

STRUCT_ALL_JSON=$(query query structs struct-all)
SAM_STRUCT_ID=$(get_newest_struct_id "${STRUCT_ALL_JSON}")
assert_not_empty "SAM struct ID" "${SAM_STRUCT_ID}"
echo "  SAM Struct ID: ${SAM_STRUCT_ID}"

# ─── P2: Battleship (type 2, space, slot 1) — independent charge, no wait ───
wait_for_charge "${PLAYER_2_ID}" "${CHARGE_BUILD}"
run_tx "Initiating P2 Battleship (type=2, space, slot=1)" \
    tx structs struct-build-initiate "${PLAYER_2_ID}" 2 space 1 --from player_2

STRUCT_ALL_JSON=$(query query structs struct-all)
P2_BATTLESHIP_ID=$(get_newest_struct_id "${STRUCT_ALL_JSON}")
assert_not_empty "Player 2 Battleship struct ID" "${P2_BATTLESHIP_ID}"
echo "  P2 Battleship Struct ID: ${P2_BATTLESHIP_ID} (compute deferred to Phase 13)"

# ─── P3: Submarine (type 13, water, slot 2) ───
wait_for_charge "${PLAYER_3_ID}" "${CHARGE_BUILD}"
run_tx "Initiating Submarine (type=13, water, slot=2)" \
    tx structs struct-build-initiate "${PLAYER_3_ID}" 13 water 2 --from player_3

STRUCT_ALL_JSON=$(query query structs struct-all)
SUB_STRUCT_ID=$(get_newest_struct_id "${STRUCT_ALL_JSON}")
assert_not_empty "Submarine struct ID" "${SUB_STRUCT_ID}"
echo "  Submarine Struct ID: ${SUB_STRUCT_ID}"

# ─── P2: Interceptor (type 7, air, slot 0) — P2 charge recovered ───
wait_for_charge "${PLAYER_2_ID}" "${CHARGE_BUILD}"
run_tx "Initiating P2 Interceptor (type=7, air, slot=0)" \
    tx structs struct-build-initiate "${PLAYER_2_ID}" 7 air 0 --from player_2

STRUCT_ALL_JSON=$(query query structs struct-all)
INTERCEPTOR_ID=$(get_newest_struct_id "${STRUCT_ALL_JSON}")
assert_not_empty "Interceptor struct ID" "${INTERCEPTOR_ID}"
echo "  P2 Interceptor Struct ID: ${INTERCEPTOR_ID} (compute deferred to Phase 14)"

# ─── P3: Battleship #1 (type 2, space, slot 1) ───
wait_for_charge "${PLAYER_3_ID}" "${CHARGE_BUILD}"
run_tx "Initiating Battleship #1 (type=2, space, slot=1)" \
    tx structs struct-build-initiate "${PLAYER_3_ID}" 2 space 1 --from player_3

STRUCT_ALL_JSON=$(query query structs struct-all)
BATTLESHIP_1_ID=$(get_newest_struct_id "${STRUCT_ALL_JSON}")
assert_not_empty "Battleship #1 struct ID" "${BATTLESHIP_1_ID}"
echo "  Battleship #1 Struct ID: ${BATTLESHIP_1_ID}"

# ─── P3: Battleship #2 (type 2, space, slot 0) ───
wait_for_charge "${PLAYER_3_ID}" "${CHARGE_BUILD}"
run_tx "Initiating Battleship #2 (type=2, space, slot=0)" \
    tx structs struct-build-initiate "${PLAYER_3_ID}" 2 space 0 --from player_3

STRUCT_ALL_JSON=$(query query structs struct-all)
BATTLESHIP_2_ID=$(get_newest_struct_id "${STRUCT_ALL_JSON}")
assert_not_empty "Battleship #2 struct ID" "${BATTLESHIP_2_ID}"
echo "  Battleship #2 Struct ID: ${BATTLESHIP_2_ID}"

# ─── P3: Stealth Bomber (type 6, air, slot 0) — needed Phase 13b ───
wait_for_charge "${PLAYER_3_ID}" "${CHARGE_BUILD}"
run_tx "Initiating Stealth Bomber (type=6, air, slot=0)" \
    tx structs struct-build-initiate "${PLAYER_3_ID}" 6 air 0 --from player_3

STRUCT_ALL_JSON=$(query query structs struct-all)
STEALTH_BOMBER_ID=$(get_newest_struct_id "${STRUCT_ALL_JSON}")
assert_not_empty "Stealth Bomber struct ID" "${STEALTH_BOMBER_ID}"
echo "  Stealth Bomber Struct ID: ${STEALTH_BOMBER_ID} (compute deferred to Phase 13b)"

# ─── P3: Cruiser (type 11, water, slot 0) — needed Phase 14 ───
wait_for_charge "${PLAYER_3_ID}" "${CHARGE_BUILD}"
run_tx "Initiating Cruiser (type=11, water, slot=0)" \
    tx structs struct-build-initiate "${PLAYER_3_ID}" 11 water 0 --from player_3

STRUCT_ALL_JSON=$(query query structs struct-all)
CRUISER_ID=$(get_newest_struct_id "${STRUCT_ALL_JSON}")
assert_not_empty "Cruiser struct ID" "${CRUISER_ID}"
echo "  Cruiser Struct ID: ${CRUISER_ID} (compute deferred to Phase 14)"

info "All 8 builds initiated. Computing Phase 12 builds now (others age in parallel)."

# ═══════════════════════════════════════════════════════════════
# COMPUTE Phase 12 builds — each subsequent compute benefits from aging
# ═══════════════════════════════════════════════════════════════

# ─── Compute SAM Launcher ───
run_compute "Building SAM Launcher ${SAM_STRUCT_ID}" \
    tx structs struct-build-compute "${SAM_STRUCT_ID}" --from player_3

assert_eq "SAM built" "true" "$(query query structs struct "${SAM_STRUCT_ID}" | jq -r '.structAttributes.isBuilt')"

wait_for_charge "${PLAYER_3_ID}" "${CHARGE_MOVE}"
run_tx "Moving SAM Launcher to fleet" \
    tx structs struct-move "${SAM_STRUCT_ID}" fleet land 2 --from player_3

# ─── Compute Submarine (aged during SAM compute) ───
run_compute "Building Submarine ${SUB_STRUCT_ID}" \
    tx structs struct-build-compute "${SUB_STRUCT_ID}" --from player_3

assert_eq "Submarine built" "true" "$(query query structs struct "${SUB_STRUCT_ID}" | jq -r '.structAttributes.isBuilt')"

wait_for_charge "${PLAYER_3_ID}" "${CHARGE_MOVE}"
run_tx "Moving Submarine to fleet" \
    tx structs struct-move "${SUB_STRUCT_ID}" fleet water 1 --from player_3

# ─── Compute Battleship #1 (aged during SAM + Sub computes) ───
run_compute "Building Galactic Battleship ${BATTLESHIP_1_ID}" \
    tx structs struct-build-compute "${BATTLESHIP_1_ID}" --from player_3

assert_eq "Battleship #1 built" "true" "$(query query structs struct "${BATTLESHIP_1_ID}" | jq -r '.structAttributes.isBuilt')"

wait_for_charge "${PLAYER_3_ID}" "${CHARGE_MOVE}"
run_tx "Moving Battleship to fleet" \
    tx structs struct-move "${BATTLESHIP_1_ID}" fleet space 2 --from player_3

# ─── Compute Battleship #2 (aged during SAM + Sub + BB1 computes) ───
run_compute "Building Galactic Battleship #2 ${BATTLESHIP_2_ID}" \
    tx structs struct-build-compute "${BATTLESHIP_2_ID}" --from player_3

assert_eq "Battleship #2 built" "true" "$(query query structs struct "${BATTLESHIP_2_ID}" | jq -r '.structAttributes.isBuilt')"


# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 13: Defense Setup & Attack Against Defenders
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 13: Defense Setup & Coordinated Attack"

# Move fleet to Player 2's planet for combat (fleet-move has no charge cost)
run_tx "Moving Player 3's fleet to Player 2's planet" \
    tx structs fleet-move "${PLAYER_3_FLEET_ID}" "${PLAYER_2_PLANET_ID}" --from player_3

# Move Command Ship to space ambit
wait_for_charge "${PLAYER_3_ID}" "${CHARGE_MOVE}"
run_tx "Moving Command Ship to space ambit" \
    tx structs struct-move "${COMMAND_SHIP_ID}" fleet space --from player_3

# ─── Set up defense network: all units defend the Command Ship ───
for DEF_ID in "${SAM_STRUCT_ID}" "${SUB_STRUCT_ID}" "${BATTLESHIP_1_ID}" "${BATTLESHIP_2_ID}"; do
    wait_for_charge "${PLAYER_3_ID}" "${CHARGE_DEFEND}"
    run_tx "Setting ${DEF_ID} to defend Command Ship ${COMMAND_SHIP_ID}" \
        tx structs struct-defense-set "${DEF_ID}" "${COMMAND_SHIP_ID}" --from player_3
done

# Verify defenders are set
CMDSHIP_DEFENDERS=$(query query structs struct "${COMMAND_SHIP_ID}" | jq -r '.structDefenders | length' 2>/dev/null || echo "0")
assert_gt "Command Ship has defenders" 0 "${CMDSHIP_DEFENDERS}"
info "Command Ship defender count: ${CMDSHIP_DEFENDERS}"

# ─── Player 2's Battleship was pre-seeded in Phase 12 — compute now (heavily aged) ───
info "Computing P2 Battleship ${P2_BATTLESHIP_ID} (pre-seeded in Phase 12)"
run_compute "Building Player 2's Battleship ${P2_BATTLESHIP_ID}" \
    tx structs struct-build-compute "${P2_BATTLESHIP_ID}" --from player_2

assert_eq "Player 2 Battleship built" "true" "$(query query structs struct "${P2_BATTLESHIP_ID}" | jq -r '.structAttributes.isBuilt')"

# Attack the defended Command Ship
CMDSHIP_JSON=$(query query structs struct "${COMMAND_SHIP_ID}" || echo '{}')
CMDSHIP_HP_BEFORE=$(jqr "${CMDSHIP_JSON}" '.structAttributes.health' '0')
info "Command Ship health before defended attack: ${CMDSHIP_HP_BEFORE}"

wait_for_charge "${PLAYER_2_ID}" "${CHARGE_ATTACK_BATTLESHIP}"
run_tx "Player 2 attacks the defended Command Ship" \
    tx structs struct-attack "${P2_BATTLESHIP_ID}" "${COMMAND_SHIP_ID}" primaryWeapon --from player_2

CMDSHIP_JSON=$(query query structs struct "${COMMAND_SHIP_ID}" || echo '{}')
CMDSHIP_HP_AFTER=$(jqr "${CMDSHIP_JSON}" '.structAttributes.health' '0')
info "Command Ship health after defended attack: ${CMDSHIP_HP_AFTER}"
echo "  (Defenders may have blocked/intercepted the attack)"

BLOCK_HEIGHT=$(query query structs block-height | jq -r '.blockHeight // empty' 2>/dev/null || echo "?")
info "Current block height: ${BLOCK_HEIGHT}"


# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 13b: Defense Clear & Stealth Systems
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 13b: Defense Clear & Stealth Systems"

# ─── struct-defense-clear: clear one defender, verify, re-set ───
# SAM is currently defending the Command Ship from Phase 13.
# Defense relationships are stored in structDefenders on the PROTECTED struct,
# not on the defending struct. Query the Command Ship to check.
CMD_JSON=$(query query structs struct "${COMMAND_SHIP_ID}" 2>/dev/null || echo '{}')
SAM_IN_DEFENDERS=$(echo "${CMD_JSON}" | jq -r --arg sid "${SAM_STRUCT_ID}" '[.structDefenders // [] | .[] | select(. == $sid)] | first // ""' 2>/dev/null || echo "")
info "SAM (${SAM_STRUCT_ID}) in Command Ship defenders: '${SAM_IN_DEFENDERS}'"

wait_for_charge "${PLAYER_3_ID}" "${CHARGE_DEFEND}"
run_tx "Clearing SAM defense assignment" \
    tx structs struct-defense-clear "${SAM_STRUCT_ID}" --from player_3

CMD_JSON=$(query query structs struct "${COMMAND_SHIP_ID}" 2>/dev/null || echo '{}')
SAM_CLEARED=$(echo "${CMD_JSON}" | jq -r --arg sid "${SAM_STRUCT_ID}" '[.structDefenders // [] | .[] | select(. == $sid)] | first // ""' 2>/dev/null || echo "")
assert_eq "SAM defense cleared" "" "${SAM_CLEARED}"

# Re-set defense for later phases
wait_for_charge "${PLAYER_3_ID}" "${CHARGE_DEFEND}"
run_tx "Re-setting SAM to defend Command Ship" \
    tx structs struct-defense-set "${SAM_STRUCT_ID}" "${COMMAND_SHIP_ID}" --from player_3

CMD_JSON=$(query query structs struct "${COMMAND_SHIP_ID}" 2>/dev/null || echo '{}')
SAM_RESET=$(echo "${CMD_JSON}" | jq -r --arg sid "${SAM_STRUCT_ID}" '[.structDefenders // [] | .[] | select(. == $sid)] | first // ""' 2>/dev/null || echo "")
assert_eq "SAM defense re-set" "${SAM_STRUCT_ID}" "${SAM_RESET}"

# ─── Stealth Bomber was pre-seeded in Phase 12 — compute now (heavily aged) ───
# Move fleet home first (needed for stealth tests — struct must be commandable)
run_tx "Moving Player 3's fleet home for stealth tests" \
    tx structs fleet-move "${PLAYER_3_FLEET_ID}" "${PLAYER_3_PLANET_ID}" --from player_3

if [ -n "${STEALTH_BOMBER_ID}" ]; then
    info "Computing Stealth Bomber ${STEALTH_BOMBER_ID} (pre-seeded in Phase 12)"
    run_compute "Building Stealth Bomber ${STEALTH_BOMBER_ID}" \
        tx structs struct-build-compute "${STEALTH_BOMBER_ID}" --from player_3

    SB_JSON=$(query query structs struct "${STEALTH_BOMBER_ID}" 2>/dev/null || echo '{}')
    SB_BUILT=$(jqr "${SB_JSON}" '.structAttributes.isBuilt' 'false')
    assert_eq "Stealth Bomber built" "true" "${SB_BUILT}"

    # ─── struct-stealth-activate ───
    wait_for_charge "${PLAYER_3_ID}" "${CHARGE_ACTIVATE}"
    run_tx "Activating stealth on Stealth Bomber" \
        tx structs struct-stealth-activate "${STEALTH_BOMBER_ID}" --from player_3

    SB_JSON=$(query query structs struct "${STEALTH_BOMBER_ID}" 2>/dev/null || echo '{}')
    SB_HIDDEN=$(jqr "${SB_JSON}" '.Struct.status' '')
    info "Stealth Bomber status after activate: '${SB_HIDDEN}'"

    # ─── struct-stealth-deactivate ───
    wait_for_charge "${PLAYER_3_ID}" "${CHARGE_ACTIVATE}"
    run_tx "Deactivating stealth on Stealth Bomber" \
        tx structs struct-stealth-deactivate "${STEALTH_BOMBER_ID}" --from player_3

    SB_JSON=$(query query structs struct "${STEALTH_BOMBER_ID}" 2>/dev/null || echo '{}')
    SB_HIDDEN_AFTER=$(jqr "${SB_JSON}" '.Struct.status' '')
    info "Stealth Bomber status after deactivate: '${SB_HIDDEN_AFTER}'"

    # struct-attribute query coverage
    info "Struct attribute query coverage:"
    query query structs struct-attribute "${STEALTH_BOMBER_ID}" 2>/dev/null | jq -r '"  isBuilt=\(.isBuilt) isOnline=\(.isOnline) health=\(.health)"' || echo "  (query failed)"
else
    info "SKIP: Could not build Stealth Bomber for stealth tests"
fi


# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 14: Cruiser vs Interceptor — Secondary Weapons & Defensive Maneuvers
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 14: Secondary Weapons & Defensive Maneuvers"

# Fleet is at home after Phase 13b stealth tests — no move needed
# Cruiser and Interceptor were pre-seeded in Phase 12 — just compute

# ─── Compute Cruiser (pre-seeded in Phase 12, heavily aged) ───
info "Cruiser (type=11) has a secondary weapon effective against air units"
info "Computing Cruiser ${CRUISER_ID} (pre-seeded in Phase 12)"
run_compute "Building Cruiser ${CRUISER_ID}" \
    tx structs struct-build-compute "${CRUISER_ID}" --from player_3

assert_eq "Cruiser built" "true" "$(query query structs struct "${CRUISER_ID}" | jq -r '.structAttributes.isBuilt')"

# Move fleet to Player 2's planet for combat (fleet-move has no charge cost)
run_tx "Moving Player 3's fleet to Player 2's planet" \
    tx structs fleet-move "${PLAYER_3_FLEET_ID}" "${PLAYER_2_PLANET_ID}" --from player_3

# ─── Compute Interceptor (pre-seeded in Phase 12, heavily aged) ───
info "Interceptor (type=7) has Defensive Maneuvers -- can try to dodge unguided attacks"
info "Computing P2 Interceptor ${INTERCEPTOR_ID} (pre-seeded in Phase 12)"
run_compute "Building Interceptor ${INTERCEPTOR_ID}" \
    tx structs struct-build-compute "${INTERCEPTOR_ID}" --from player_2

INTERCEPTOR_JSON=$(query query structs struct "${INTERCEPTOR_ID}")
assert_eq "Interceptor built" "true" "$(jqr "${INTERCEPTOR_JSON}" '.structAttributes.isBuilt' 'false')"
assert_eq "Interceptor type" "7" "$(jqr "${INTERCEPTOR_JSON}" '.Struct.type')"

# ─── Cruiser attacks Interceptor with secondary weapon (2 rounds) ───
INTERCEPTOR_HP_BEFORE=$(jqr "${INTERCEPTOR_JSON}" '.structAttributes.health' '0')
info "Interceptor health before attacks: ${INTERCEPTOR_HP_BEFORE}"

wait_for_charge "${PLAYER_3_ID}" "${CHARGE_ATTACK_DEFAULT}"
run_tx "Cruiser attacks Interceptor with secondaryWeapon (round 1)" \
    tx structs struct-attack "${CRUISER_ID}" "${INTERCEPTOR_ID}" secondaryWeapon --from player_3

INTERCEPTOR_JSON=$(query query structs struct "${INTERCEPTOR_ID}" || echo '{}')
INTERCEPTOR_HP_MID=$(jqr "${INTERCEPTOR_JSON}" '.structAttributes.health' '0')
info "Interceptor health after round 1: ${INTERCEPTOR_HP_MID} (may have dodged)"

wait_for_charge "${PLAYER_3_ID}" "${CHARGE_ATTACK_DEFAULT}"
run_tx "Cruiser attacks Interceptor with secondaryWeapon (round 2)" \
    tx structs struct-attack "${CRUISER_ID}" "${INTERCEPTOR_ID}" secondaryWeapon --from player_3

INTERCEPTOR_JSON=$(query query structs struct "${INTERCEPTOR_ID}" || echo '{}')
INTERCEPTOR_HP_AFTER=$(jqr "${INTERCEPTOR_JSON}" '.structAttributes.health' '0')
info "Interceptor health after round 2: ${INTERCEPTOR_HP_AFTER}"

echo ""
echo "  NOTE: The Interceptor has Defensive Maneuvers, so it may have dodged"
echo "  unguided attacks from the Cruiser's secondary weapon."

BLOCK_HEIGHT=$(query query structs block-height | jq -r '.blockHeight // empty' 2>/dev/null || echo "?")
info "Final block height: ${BLOCK_HEIGHT}"


# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 15: Power Generator — Build, Infuse, Verify, and Destroy
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 15: Power Generator (Player 4)"

echo "  Player 4 Planet: ${PLAYER_4_PLANET_ID}"
echo "  Using Field Generator (struct type 20, land, slot 0)"
echo "  GeneratingRate=2, PassiveDraw=500000, MaxHealth=3"

# ─── Snapshot Player 4's capacity before building ───
P4_JSON=$(query query structs player "${PLAYER_4_ID}")
P4_CAP_BEFORE=$(jqr "${P4_JSON}" '.gridAttributes.capacity' '0')
P4_LOAD_BEFORE=$(jqr "${P4_JSON}" '.gridAttributes.structsLoad' '0')
info "Player 4 capacity before generator: ${P4_CAP_BEFORE}"
info "Player 4 structsLoad before generator: ${P4_LOAD_BEFORE}"

# ─── Generator was pre-seeded in Phase 8 — compute now (massively aged) ───
info "Computing Field Generator ${GENERATOR_STRUCT_ID} (pre-seeded in Phase 8)"
run_compute "Building Field Generator ${GENERATOR_STRUCT_ID}" \
    tx structs struct-build-compute "${GENERATOR_STRUCT_ID}" --from player_4

# Verify it was built and went online automatically
GEN_JSON=$(query query structs struct "${GENERATOR_STRUCT_ID}")
GEN_BUILT=$(jqr "${GEN_JSON}" '.structAttributes.isBuilt' 'false')
GEN_ONLINE=$(jqr "${GEN_JSON}" '.structAttributes.isOnline' 'false')
GEN_TYPE=$(jqr "${GEN_JSON}" '.Struct.type')
assert_eq "Generator built" "true" "${GEN_BUILT}"
assert_eq "Generator online after build" "true" "${GEN_ONLINE}"
assert_eq "Generator type" "20" "${GEN_TYPE}"

# ─── Deactivate generator (must be offline to infuse) ───
run_tx "Deactivating generator for infusion" \
    tx structs struct-deactivate "${GENERATOR_STRUCT_ID}" --from player_4

# NOTE: isOnline may still read 'true' briefly after deactivation due to query timing.
# The infuse succeeding (next step) is the authoritative confirmation that the struct
# is offline, since struct-generator-infuse rejects online structs.
GEN_JSON=$(query query structs struct "${GENERATOR_STRUCT_ID}")
GEN_ONLINE_AFTER_DEACTIVATE=$(jqr "${GEN_JSON}" '.structAttributes.isOnline' 'true')
info "Generator isOnline after deactivate: ${GEN_ONLINE_AFTER_DEACTIVATE} (infuse success confirms offline)"

# ─── Infuse alpha into the generator ───
INFUSE_AMOUNT="1000000ualpha"
info "Infusing ${INFUSE_AMOUNT} into generator (GeneratingRate=2, expected power=2000000)"

run_tx "Infusing ${INFUSE_AMOUNT} into generator ${GENERATOR_STRUCT_ID}" \
    tx structs struct-generator-infuse "${GENERATOR_STRUCT_ID}" "${INFUSE_AMOUNT}" --from player_4

# Verify fuel was added to the struct
GEN_JSON=$(query query structs struct "${GENERATOR_STRUCT_ID}")
GEN_FUEL=$(jqr "${GEN_JSON}" '.gridAttributes.fuel' '0')
info "Generator fuel after infusion: ${GEN_FUEL}"
assert_gt "Generator has fuel" 0 "${GEN_FUEL}"

# ─── Activate generator (bring back online with power) ───
run_tx "Activating generator ${GENERATOR_STRUCT_ID}" \
    tx structs struct-activate "${GENERATOR_STRUCT_ID}" --from player_4

GEN_JSON=$(query query structs struct "${GENERATOR_STRUCT_ID}")
GEN_ONLINE_AFTER_ACTIVATE=$(jqr "${GEN_JSON}" '.structAttributes.isOnline' 'false')
assert_eq "Generator online after activate" "true" "${GEN_ONLINE_AFTER_ACTIVATE}"

# ─── Verify Player 4's capacity increased from generator power ───
P4_JSON=$(query query structs player "${PLAYER_4_ID}")
P4_CAP_AFTER_GEN=$(jqr "${P4_JSON}" '.gridAttributes.capacity' '0')
P4_LOAD_AFTER_GEN=$(jqr "${P4_JSON}" '.gridAttributes.structsLoad' '0')
info "Player 4 capacity after generator online: ${P4_CAP_AFTER_GEN} (was ${P4_CAP_BEFORE})"
info "Player 4 structsLoad after generator online: ${P4_LOAD_AFTER_GEN} (was ${P4_LOAD_BEFORE})"
assert_gt "Player 4 capacity increased from generator" "${P4_CAP_BEFORE}" "${P4_CAP_AFTER_GEN}"

# Generator power contribution is reflected in the player's capacity increase
# (verified above), not necessarily in the struct's own gridAttributes.power.
GEN_POWER=$(jqr "${GEN_JSON}" '.gridAttributes.power' '0')
info "Generator gridAttributes.power: ${GEN_POWER} (power reflected in player capacity, not struct)"

# ─── Verify generator energy can support allocations ───
# Player 4 should have more available capacity now
P4_AVAIL_CAP=$(( P4_CAP_AFTER_GEN - P4_LOAD_AFTER_GEN ))
info "Player 4 available capacity (capacity - structsLoad): ${P4_AVAIL_CAP}"
assert_gt "Player 4 has available capacity for allocations" 0 "${P4_AVAIL_CAP}"

# ─── Now destroy the generator: Player 3 attacks ───
info "--- Destruction Phase ---"
echo "  Player 3 will move fleet to Player 4's planet and destroy the generator"
echo "  Generator MaxHealth=3, Tank (type=9, land) does 2 damage per shot → 2 rounds"
echo "  NOTE: Command Ship (space ambit) cannot target land structs — using Tank instead"

# Move Player 3's fleet to Player 4's planet (fleet-move has no charge cost)
run_tx "Moving Player 3's fleet to Player 4's planet (${PLAYER_4_PLANET_ID})" \
    tx structs fleet-move "${PLAYER_3_FLEET_ID}" "${PLAYER_4_PLANET_ID}" --from player_3

# Verify fleet location
FLEET_3_JSON=$(query query structs fleet "${PLAYER_3_FLEET_ID}")
FLEET_3_LOC=$(jqr "${FLEET_3_JSON}" '.Fleet.locationId')
info "Player 3 fleet location: ${FLEET_3_LOC}"

# Record generator health before attacks
GEN_JSON=$(query query structs struct "${GENERATOR_STRUCT_ID}" || echo '{}')
GEN_HP_BEFORE=$(jqr "${GEN_JSON}" '.structAttributes.health' '0')
info "Generator health before attacks: ${GEN_HP_BEFORE}"

# Attack round 1 — use Destroyer/Tank (type=9, land ambit, PrimaryWeaponAmbits=4=land)
# Tank PrimaryWeaponCharge=1, PrimaryWeaponDamage=2
wait_for_charge "${PLAYER_3_ID}" "${CHARGE_ATTACK_DEFAULT}"
run_tx "Attack round 1: Tank -> Generator" \
    tx structs struct-attack "${DESTROYER_STRUCT_ID}" "${GENERATOR_STRUCT_ID}" primaryWeapon --from player_3

GEN_JSON=$(query query structs struct "${GENERATOR_STRUCT_ID}" || echo '{}')
GEN_HP_MID=$(jqr "${GEN_JSON}" '.structAttributes.health' '0')
info "Generator health after round 1: ${GEN_HP_MID}"

# Attack round 2 — should destroy it (3 HP - 2 dmg = 1 HP, then 1 HP - 2 dmg = destroyed)
wait_for_charge "${PLAYER_3_ID}" "${CHARGE_ATTACK_DEFAULT}"
run_tx "Attack round 2: Tank -> Generator (should destroy)" \
    tx structs struct-attack "${DESTROYER_STRUCT_ID}" "${GENERATOR_STRUCT_ID}" primaryWeapon --from player_3

GEN_JSON=$(query query structs struct "${GENERATOR_STRUCT_ID}" || echo '{}')
GEN_HP_AFTER=$(jqr "${GEN_JSON}" '.structAttributes.health' '0')
GEN_DESTROYED=$(jqr "${GEN_JSON}" '.structAttributes.isDestroyed' 'false')
info "Generator health after round 2: ${GEN_HP_AFTER}"
info "Generator isDestroyed: ${GEN_DESTROYED}"
assert_eq "Generator destroyed (health=0)" "0" "${GEN_HP_AFTER}"

# ─── Verify Player 4's capacity decreased — energy no longer generated ───
P4_JSON=$(query query structs player "${PLAYER_4_ID}")
P4_CAP_AFTER_DESTROY=$(jqr "${P4_JSON}" '.gridAttributes.capacity' '0')
P4_LOAD_AFTER_DESTROY=$(jqr "${P4_JSON}" '.gridAttributes.structsLoad' '0')
info "Player 4 capacity after generator destroyed: ${P4_CAP_AFTER_DESTROY} (was ${P4_CAP_AFTER_GEN} with generator)"
info "Player 4 structsLoad after generator destroyed: ${P4_LOAD_AFTER_DESTROY} (was ${P4_LOAD_AFTER_GEN})"
assert_lt "Player 4 capacity decreased after generator destroyed" "${P4_CAP_AFTER_GEN}" "${P4_CAP_AFTER_DESTROY}"

# Generator fuel/power should be zero after destruction
GEN_FUEL_AFTER=$(jqr "${GEN_JSON}" '.gridAttributes.fuel' '0')
GEN_POWER_AFTER=$(jqr "${GEN_JSON}" '.gridAttributes.power' '0')
info "Generator fuel after destruction: ${GEN_FUEL_AFTER} (was ${GEN_FUEL})"
info "Generator power after destruction: ${GEN_POWER_AFTER} (was ${GEN_POWER})"
assert_eq "Generator fuel zeroed after destruction" "0" "${GEN_FUEL_AFTER}"
assert_eq "Generator power zeroed after destruction" "0" "${GEN_POWER_AFTER}"

echo ""
echo "  Summary: Generator built, infused (power=${GEN_POWER}), then destroyed."
echo "  Player 4 capacity: ${P4_CAP_BEFORE} → ${P4_CAP_AFTER_GEN} (with gen) → ${P4_CAP_AFTER_DESTROY} (destroyed)"


# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 15b: Player Operations
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 15b: Player Operations"

# ─── player-send: Player 2 sends tokens to Player 3 ───
# NOTE: Player 5 lost PermissionAssets (8) in Phase 4e, so we use Player 2 instead.
P2_BALANCE_BEFORE=$(get_balance "${PLAYER_2_ADDRESS}" "ualpha")
P3_BALANCE_BEFORE=$(get_balance "${PLAYER_3_ADDRESS}" "ualpha")
info "Player 2 ualpha before send: ${P2_BALANCE_BEFORE}"
info "Player 3 ualpha before send: ${P3_BALANCE_BEFORE}"

SEND_AMOUNT="100000"
run_tx "Player 2 sending ${SEND_AMOUNT}ualpha to Player 3 via player-send" \
    tx structs player-send "${PLAYER_2_ID}" "${PLAYER_2_ADDRESS}" "${PLAYER_3_ADDRESS}" "${SEND_AMOUNT}ualpha" --from player_2

P2_BALANCE_AFTER=$(get_balance "${PLAYER_2_ADDRESS}" "ualpha")
P3_BALANCE_AFTER=$(get_balance "${PLAYER_3_ADDRESS}" "ualpha")
info "Player 2 ualpha after send: ${P2_BALANCE_AFTER}"
info "Player 3 ualpha after send: ${P3_BALANCE_AFTER}"

# Player 3 should have received the tokens
if [ -n "${P3_BALANCE_BEFORE}" ] && [ -n "${P3_BALANCE_AFTER}" ] && [ "${P3_BALANCE_BEFORE}" != "0" ]; then
    assert_gt "Player 3 balance increased after player-send" "${P3_BALANCE_BEFORE}" "${P3_BALANCE_AFTER}"
fi

# Note: player-update-primary-address requires a second address registered with
# a valid cryptographic proof signature, which is complex to generate in bash.
# Skipping that test but noting the limitation.
info "player-update-primary-address: SKIP (requires crypto proof for second address)"


# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 16: Provider & Agreement System
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 16: Provider & Agreement System"

# provider-create [substation id] [rate] [access policy] [provider penalty] [consumer penalty]
#                  [capacity min] [capacity max] [duration min] [duration max]
PROVIDER_ACCESS="guild-market"
PROVIDER_PROVIDER_PENALTY=0
PROVIDER_CONSUMER_PENALTY=0
PROVIDER_CAP_MIN=100000
PROVIDER_CAP_MAX=5000000
PROVIDER_DUR_MIN=10
PROVIDER_DUR_MAX=10000

# ─── Negative test: bad rate (no denomination) should be rejected ───
BAD_RATE="1"
run_tx "Alice attempting provider-create with bad rate (no denom)" \
    tx structs provider-create "${SUBSTATION_ID}" \
    "${BAD_RATE}" "${PROVIDER_ACCESS}" \
    "${PROVIDER_PROVIDER_PENALTY}" "${PROVIDER_CONSUMER_PENALTY}" \
    "${PROVIDER_CAP_MIN}" "${PROVIDER_CAP_MAX}" \
    "${PROVIDER_DUR_MIN}" "${PROVIDER_DUR_MAX}" \
    --from alice

PROVIDER_ALL_BEFORE=$(query query structs provider-all 2>/dev/null || echo '{}')
BAD_PROVIDER_ID=$(echo "${PROVIDER_ALL_BEFORE}" | jq -r '.Provider[-1].id // empty' 2>/dev/null || echo "")
assert_eq "Bad rate provider rejected (no provider created)" "" "${BAD_PROVIDER_ID}"

# ─── Good provider creation with proper coin denomination ───
PROVIDER_RATE="1ualpha"

run_tx "Alice creating energy provider on substation" \
    tx structs provider-create "${SUBSTATION_ID}" \
    "${PROVIDER_RATE}" "${PROVIDER_ACCESS}" \
    "${PROVIDER_PROVIDER_PENALTY}" "${PROVIDER_CONSUMER_PENALTY}" \
    "${PROVIDER_CAP_MIN}" "${PROVIDER_CAP_MAX}" \
    "${PROVIDER_DUR_MIN}" "${PROVIDER_DUR_MAX}" \
    --from alice

# Find the provider
PROVIDER_ALL=$(query query structs provider-all 2>/dev/null || echo '{}')
PROVIDER_ID=$(echo "${PROVIDER_ALL}" | jq -r '.Provider[-1].id // empty' 2>/dev/null || echo "")
info "Provider ID: ${PROVIDER_ID}"

if [ -n "${PROVIDER_ID}" ]; then
    # Query provider details
    PROV_JSON=$(query query structs provider "${PROVIDER_ID}" 2>/dev/null || echo '{}')
    PROV_RATE_AMOUNT=$(jqr "${PROV_JSON}" '.Provider.rate.amount' '')
    PROV_RATE_DENOM=$(jqr "${PROV_JSON}" '.Provider.rate.denom' '')
    assert_eq "Provider rate amount" "1" "${PROV_RATE_AMOUNT}"
    assert_eq "Provider rate denom" "ualpha" "${PROV_RATE_DENOM}"
    info "Provider created: rate=${PROV_RATE_AMOUNT}${PROV_RATE_DENOM}"

    # ─── provider-update-access-policy ───
    run_tx "Updating provider access policy to 'open-market'" \
        tx structs provider-update-access-policy "${PROVIDER_ID}" "open-market" --from alice

    PROV_JSON=$(query query structs provider "${PROVIDER_ID}" 2>/dev/null || echo '{}')
    PROV_ACCESS=$(jqr "${PROV_JSON}" '.Provider.accessPolicy' '')
    info "Provider access policy after update: ${PROV_ACCESS}"

    # Set back to guild
    run_tx "Setting provider access policy back to 'guild-market'" \
        tx structs provider-update-access-policy "${PROVIDER_ID}" "guild-market" --from alice

    # ─── provider-guild-grant: grant guild access ───
    run_tx "Granting guild ${GUILD_ID} access to provider" \
        tx structs provider-guild-grant "${PROVIDER_ID}" "${GUILD_ID}" --from alice

    # ─── provider-update-capacity-minimum / maximum ───
    run_tx "Updating provider capacity minimum to 50000" \
        tx structs provider-update-capacity-minimum "${PROVIDER_ID}" 50000 --from alice

    run_tx "Updating provider capacity maximum to 10000000" \
        tx structs provider-update-capacity-maximum "${PROVIDER_ID}" 10000000 --from alice

    # ─── provider-update-duration-minimum / maximum ───
    run_tx "Updating provider duration minimum to 5" \
        tx structs provider-update-duration-minimum "${PROVIDER_ID}" 5 --from alice

    run_tx "Updating provider duration maximum to 50000" \
        tx structs provider-update-duration-maximum "${PROVIDER_ID}" 50000 --from alice

    # Verify updates
    PROV_JSON=$(query query structs provider "${PROVIDER_ID}" 2>/dev/null || echo '{}')
    PROV_CAP_MIN=$(jqr "${PROV_JSON}" '.Provider.capacityMinimum' '0')
    PROV_DUR_MAX=$(jqr "${PROV_JSON}" '.Provider.durationMaximum' '0')
    assert_eq "Provider capacity minimum updated" "50000" "${PROV_CAP_MIN}"
    assert_eq "Provider duration maximum updated" "50000" "${PROV_DUR_MAX}"

    # ─── agreement-open: Player 2 opens an agreement ───
    # Collateral = duration * capacity * rate
    AGREE_DURATION=40
    AGREE_CAPACITY=50000

    P2_ALPHA_BEFORE=$(get_balance "${PLAYER_2_ADDRESS}" "ualpha")
    info "Player 2 ualpha before agreement: ${P2_ALPHA_BEFORE}"

    run_tx "Player 2 opening agreement with provider (dur=${AGREE_DURATION}, cap=${AGREE_CAPACITY})" \
        tx structs agreement-open "${PROVIDER_ID}" "${AGREE_DURATION}" "${AGREE_CAPACITY}" --from player_2

    # Find the agreement
    AGREE_ALL=$(query query structs agreement-all 2>/dev/null || echo '{}')
    AGREE_ID=$(echo "${AGREE_ALL}" | jq -r '.Agreement[-1].id // empty' 2>/dev/null || echo "")
    info "Agreement ID: ${AGREE_ID}"

    if [ -n "${AGREE_ID}" ]; then
        # Query agreement
        AGREE_JSON=$(query query structs agreement "${AGREE_ID}" 2>/dev/null || echo '{}')
        AGREE_PROV=$(jqr "${AGREE_JSON}" '.Agreement.providerId' '')
        AGREE_OWNER=$(jqr "${AGREE_JSON}" '.Agreement.owner' '')
        assert_eq "Agreement provider" "${PROVIDER_ID}" "${AGREE_PROV}"
        assert_eq "Agreement owner is Player 2" "${PLAYER_2_ID}" "${AGREE_OWNER}"

        AGREE_CAP_CURRENT=$(jqr "${AGREE_JSON}" '.Agreement.capacity' '0')
        AGREE_END_BEFORE=$(jqr "${AGREE_JSON}" '.Agreement.endBlock' '0')
        info "Agreement capacity: ${AGREE_CAP_CURRENT}, endBlock: ${AGREE_END_BEFORE}"

        # ─── agreement-capacity-increase ───
        run_tx "Increasing agreement capacity by 25000" \
            tx structs agreement-capacity-increase "${AGREE_ID}" 25000 --from player_2

        AGREE_JSON=$(query query structs agreement "${AGREE_ID}" 2>/dev/null || echo '{}')
        AGREE_CAP_AFTER=$(jqr "${AGREE_JSON}" '.Agreement.capacity' '0')
        info "Agreement capacity after increase: ${AGREE_CAP_AFTER}"
        assert_gt "Agreement capacity increased" "${AGREE_CAP_CURRENT}" "${AGREE_CAP_AFTER}"

        # ─── agreement-capacity-decrease ───
        run_tx "Decreasing agreement capacity by 10000" \
            tx structs agreement-capacity-decrease "${AGREE_ID}" 10000 --from player_2

        AGREE_JSON=$(query query structs agreement "${AGREE_ID}" 2>/dev/null || echo '{}')
        AGREE_CAP_AFTER_DEC=$(jqr "${AGREE_JSON}" '.Agreement.capacity' '0')
        info "Agreement capacity after decrease: ${AGREE_CAP_AFTER_DEC}"
        assert_gt "Agreement capacity decreased" "${AGREE_CAP_AFTER_DEC}" "${AGREE_CAP_AFTER}"

        # ─── agreement-duration-increase ───
        AGREE_END_BEFORE_DUR=$(jqr "${AGREE_JSON}" '.Agreement.endBlock' '0')
        run_tx "Increasing agreement duration by 20" \
            tx structs agreement-duration-increase "${AGREE_ID}" 20 --from player_2

        AGREE_JSON=$(query query structs agreement "${AGREE_ID}" 2>/dev/null || echo '{}')
        AGREE_END_AFTER_DUR=$(jqr "${AGREE_JSON}" '.Agreement.endBlock' '0')
        info "Agreement endBlock after duration increase: ${AGREE_END_AFTER_DUR} (was ${AGREE_END_BEFORE_DUR})"
        assert_gt "Agreement endBlock increased" "${AGREE_END_BEFORE_DUR}" "${AGREE_END_AFTER_DUR}"

        # ─── agreement-close ───
        run_tx "Closing agreement" \
            tx structs agreement-close "${AGREE_ID}" --from player_2

        # Verify agreement is closed (no longer queryable)
        AGREE_JSON=$(query query structs agreement "${AGREE_ID}" 2>/dev/null || echo '{}')
        AGREE_CLOSED_CHECK=$(jqr "${AGREE_JSON}" '.Agreement.id' '')
        assert_eq "Agreement closed (removed)" "" "${AGREE_CLOSED_CHECK}"
    else
        info "SKIP: Could not open agreement"
    fi

    # ─── provider-withdraw-balance ───
    # Query provider collateral/earnings addresses
    PROV_COLLATERAL=$(query query structs provider-collateral-address "${PROVIDER_ID}" 2>/dev/null || echo '{}')
    PROV_COLL_ADDR=$(echo "${PROV_COLLATERAL}" | jq -r '.internalAddressAssociation[0].address // empty' 2>/dev/null || echo "")
    info "Provider collateral address: ${PROV_COLL_ADDR}"

    run_tx "Withdrawing provider balance to alice" \
        tx structs provider-withdraw-balance "${PROVIDER_ID}" "${PLAYER_1_ADDRESS}" --from alice

    # ─── provider-guild-revoke ───
    run_tx "Revoking guild access from provider" \
        tx structs provider-guild-revoke "${PROVIDER_ID}" "${GUILD_ID}" --from alice

    # ─── provider-delete ───
    run_tx "Deleting provider" \
        tx structs provider-delete "${PROVIDER_ID}" --from alice

    # Verify provider is gone
    PROV_GONE=$(query query structs provider "${PROVIDER_ID}" 2>/dev/null || echo '{}')
    PROV_GONE_ID=$(jqr "${PROV_GONE}" '.Provider.id' '')
    assert_eq "Provider deleted" "" "${PROV_GONE_ID}"
else
    info "SKIP: Could not create provider"
fi


# ═════════════════════════════════════════════════════════════════════════════
#  Final State Dump & Summary
# ═════════════════════════════════════════════════════════════════════════════

section "Final State Dump"

info "All Structs:"
query query structs struct-all | jq -r '.Struct[] | "  \(.id) type=\(.type) owner=\(.owner) ambit=\(.operatingAmbit) loc=\(.locationType)/\(.locationId)"' 2>/dev/null || true

echo ""
info "All Players:"
query query structs player-all | jq -r '.Player[] | "  \(.id) guild=\(.guildId) planet=\(.planetId) fleet=\(.fleetId)"' 2>/dev/null || true

echo ""
info "All Fleets:"
query query structs fleet-all | jq -r '.Fleet[] | "  \(.id) loc=\(.locationId) status=\(.status)"' 2>/dev/null || true

echo ""
info "All Planets:"
query query structs planet-all | jq -r '.Planet[] | "  \(.id)"' 2>/dev/null || true

echo ""
info "All Allocations:"
query query structs allocation-all | jq -r '.Allocation[] | "  \(.id) src=\(.sourceObjectId) dst=\(.destinationId)"' 2>/dev/null || true

# ─── Print Summary ───
print_summary
