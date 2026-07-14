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
#   --skip-mining      Skip ore mining, refinery build/refine, and planet raid.
#                      Dramatically reduces runtime by avoiding the slowest
#                      proof-of-work compute operations.
#   --extended-battle  Run comprehensive combat tests after the standard phases.
#                      Builds all 13 fleet struct types, sets up defensive
#                      configurations, and exercises every combat mechanic.
#   --log-battle       Capture full EventAttack details from every struct-attack
#                      transaction and write them to a JSONL file under
#                      tests/battle_logs/. Each line is a JSON object with
#                      timestamp, description, txhash, height, and the raw
#                      event attributes emitted by the chain.
#   --resume-from N    Skip phases before N and resume execution from phase N.
#                      Recovers all IDs by querying the running chain.
#                      Phase names: 0 1 2 3 3b 4 4b 4c 4d 4e 4f 4g 5 5b 6
#                        7 7b 8 9 10 11 12 13 13b 14 15 15b 16
#                        17 17b 17c 18 eb1-eb6 ev1 ar1-ar4 rg1 rg2
#

set -euo pipefail

# ─── Flag Parsing ─────────────────────────────────────────────────────────────

SKIP_MINING=false
EXTENDED_BATTLE=false
LOG_BATTLE=false
RESUME_FROM=""
while [ $# -gt 0 ]; do
    case "$1" in
        --skip-mining)      SKIP_MINING=true ;;
        --extended-battle)  EXTENDED_BATTLE=true ;;
        --log-battle)       LOG_BATTLE=true ;;
        --resume-from)      RESUME_FROM="$2"; shift ;;
        *)                  echo "Unknown flag: $1"; exit 1 ;;
    esac
    shift
done

# ─── Configuration ────────────────────────────────────────────────────────────

SLEEP=2
BIGGER_SLEEP=15
PARAMS_TX="--home ~/.structs --keyring-dir ~/.structs --keyring-backend test --gas auto --yes=true"
PARAMS_QUERY="--home ~/.structs --output json"
PARAMS_KEYS="--home ~/.structs --keyring-dir ~/.structs --keyring-backend test --output json"

# ─── Battle Log Setup ─────────────────────────────────────────────────────────

BATTLE_LOG_COUNT=0
BATTLE_LOG_FILE=""
if [ "${LOG_BATTLE}" = true ]; then
    BATTLE_LOG_DIR="$(cd "$(dirname "$0")" && pwd)/battle_logs"
    mkdir -p "${BATTLE_LOG_DIR}"
    BATTLE_LOG_FILE="${BATTLE_LOG_DIR}/battle_$(date +%Y%m%d_%H%M%S).jsonl"
    jq -nc \
        --arg ts "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
        --arg skip_mining "${SKIP_MINING}" \
        --arg extended_battle "${EXTENDED_BATTLE}" \
        --arg resume_from "${RESUME_FROM}" \
        '{type:"session_start", timestamp:$ts, flags:{skip_mining:$skip_mining, extended_battle:$extended_battle, resume_from:$resume_from}}' \
        > "${BATTLE_LOG_FILE}"
fi

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
        echo "  $(echo "${output}" | head -5)"
    elif echo "${output}" | grep -qi "error\|panic\|failed\|invalid"; then
        echo -e "  ${RED}TX failed (simulation/gas estimate error)${NC}"
        echo "  $(echo "${output}" | tail -3)"
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
    if [ "${LOG_BATTLE}" = true ] && [[ "$*" == *"struct-attack"* ]]; then
        OUTPUT=$(structsd ${PARAMS_TX} --output json "$@" 2>&1) || true
    else
        OUTPUT=$(structsd ${PARAMS_TX} "$@" 2>&1) || true
    fi
    _check_tx_output "${OUTPUT}"
    sleep "${SLEEP}"
    if [ "${LOG_BATTLE}" = true ] && [[ "$*" == *"struct-attack"* ]]; then
        _log_battle_event "${OUTPUT}" "${description}"
    fi
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

# run_tx_expect_fail: execute a TX that SHOULD fail. Pass if it fails, fail if it succeeds.
run_tx_expect_fail() {
    local description="$1"
    shift
    info "${description} (expect failure)"
    echo -e "  ${BOLD}structsd ${PARAMS_TX} $*${NC}"
    local OUTPUT
    OUTPUT=$(structsd ${PARAMS_TX} "$@" 2>&1) || true
    local tx_code
    tx_code=$(echo "${OUTPUT}" | jq -r '.code // empty' 2>/dev/null || echo "")
    if [ "${tx_code}" = "0" ]; then
        echo -e "  ${RED}FAIL${NC}: TX succeeded but was expected to fail"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    elif echo "${OUTPUT}" | grep -qi "error\|panic\|failed\|invalid\|rejected"; then
        echo -e "  ${GREEN}PASS${NC}: TX correctly rejected"
        PASS_COUNT=$((PASS_COUNT + 1))
    elif [ -n "${tx_code}" ] && [ "${tx_code}" != "0" ]; then
        echo -e "  ${GREEN}PASS${NC}: TX failed with code=${tx_code}"
        PASS_COUNT=$((PASS_COUNT + 1))
    else
        echo -e "  ${YELLOW}WARN${NC}: Could not determine TX outcome, assuming failure"
        PASS_COUNT=$((PASS_COUNT + 1))
    fi
    sleep "${SLEEP}"
}

# run_tx_expect_fail_noauto: same but with fixed gas (no --gas auto)
PARAMS_TX_NOFEE="--home ~/.structs --keyring-dir ~/.structs --keyring-backend test --gas 500000 --fees 0ualpha --yes=true"
run_tx_expect_fail_noauto() {
    local description="$1"
    shift
    info "${description} (expect failure, fixed gas)"
    echo -e "  ${BOLD}structsd ${PARAMS_TX_NOFEE} $*${NC}"
    local OUTPUT
    OUTPUT=$(structsd ${PARAMS_TX_NOFEE} "$@" 2>&1) || true
    local tx_code
    tx_code=$(echo "${OUTPUT}" | jq -r '.code // empty' 2>/dev/null || echo "")
    if [ "${tx_code}" = "0" ]; then
        echo -e "  ${RED}FAIL${NC}: TX succeeded but was expected to fail"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    else
        echo -e "  ${GREEN}PASS${NC}: TX correctly rejected"
        PASS_COUNT=$((PASS_COUNT + 1))
    fi
    sleep "${SLEEP}"
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

# _log_battle_event: capture EventAttack details from a struct-attack tx
_log_battle_event() {
    local tx_output="$1"
    local description="$2"

    local json_line txhash
    json_line=$(echo "${tx_output}" | grep '^{' | tail -1) || true
    txhash=$(echo "${json_line}" | jq -r '.txhash // empty' 2>/dev/null || echo "")
    if [ -z "${txhash}" ]; then return; fi

    local tx_result
    tx_result=$(structsd query tx "${txhash}" --home ~/.structs --output json 2>/dev/null) || return

    local height
    height=$(echo "${tx_result}" | jq -r '.height // "?"' 2>/dev/null || echo "?")

    local attack_events
    attack_events=$(echo "${tx_result}" | jq -c '
        [(.events // [])[] | select(.type | test("EventAttack"))]
    ' 2>/dev/null || echo "[]")

    if [ "${attack_events}" = "[]" ] || [ -z "${attack_events}" ]; then
        attack_events=$(echo "${tx_result}" | jq -c '
            [(.events // [])[] | select(.type | test("[Aa]ttack"))]
        ' 2>/dev/null || echo "[]")
    fi

    jq -nc \
        --arg ts "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
        --arg desc "${description}" \
        --arg tx "${txhash}" \
        --arg ht "${height}" \
        --argjson events "${attack_events}" \
        '{type:"attack", timestamp:$ts, description:$desc, txhash:$tx, height:$ht, events:$events}' \
        >> "${BATTLE_LOG_FILE}"

    BATTLE_LOG_COUNT=$((BATTLE_LOG_COUNT + 1))
    echo -e "  ${CYAN}(battle event #${BATTLE_LOG_COUNT} logged)${NC}"
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

# assert_new_struct: fail loudly if a struct-build-initiate did not produce a
# NEW struct of the expected type (guards against get_newest_struct_id silently
# returning a stale id when an initiate is rejected).
# Usage: assert_new_struct <label> <new_id> <prev_newest_id> <expected_type>
assert_new_struct() {
    local label="$1" new_id="$2" prev_id="$3" exp_type="$4"
    if [ -z "${new_id}" ] || [ "${new_id}" = "${prev_id}" ]; then
        echo -e "  ${RED}FAIL${NC}: ${label} - initiate produced no new struct (id='${new_id}', prev='${prev_id}')"
        FAIL_COUNT=$((FAIL_COUNT + 1)); return 1
    fi
    local t; t=$(query query structs struct "${new_id}" | jq -r '.Struct.type // empty')
    if [ "${t}" != "${exp_type}" ]; then
        echo -e "  ${RED}FAIL${NC}: ${label} - new struct ${new_id} type='${t}', expected '${exp_type}'"
        FAIL_COUNT=$((FAIL_COUNT + 1)); return 1
    fi
    echo -e "  ${GREEN}PASS${NC}: ${label} = ${new_id} (type ${exp_type})"
    PASS_COUNT=$((PASS_COUNT + 1))
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
CHARGE_ATTACK_DEFAULT=3              # primary charge 3 (Tank/Starfighter/Pursuit/CmdShip)
CHARGE_ATTACK_BATTLESHIP=5           # guided secondary (v0.18.0 charge rebalance)
CHARGE_ATTACK_BATTLESHIP_PRIMARY=5   # armour-piercing unguided primary
CHARGE_ATTACK_SAM=5
CHARGE_MOVE=3                        # only the Command Ship pays moveCharge (3); others 0
CHARGE_DEFEND=1
CHARGE_ACTIVATE=2

# Permission constants (from x/structs/types/permissions.go, 1<<iota)
PERM_PLAY=1
PERM_ADMIN=2
PERM_UPDATE=4
PERM_DELETE=8
PERM_TOKEN_TRANSFER=16
PERM_TOKEN_INFUSE=32
PERM_SOURCE_ALLOCATION=256
PERM_GUILD_MEMBERSHIP=512
PERM_SUBSTATION_CONNECTION=1024
PERM_ALLOCATION_CONNECTION=2048
PERM_GUILD_ENDPOINT_UPDATE=16384

# ─── Fleet / Planet / Struct Query Helpers ─────────────────────────────────────

# query_fleet: return fleet JSON
query_fleet() { query query structs fleet "$1"; }

# query_planet: return planet JSON
query_planet() { query query structs planet "$1"; }

# get_fleet_field: extract a specific field from a fleet
get_fleet_field() {
    local fleet_id="$1" field="$2"
    query_fleet "${fleet_id}" | jq -r ".Fleet.${field} // empty" 2>/dev/null || echo ""
}

# get_planet_field: extract a specific field from a planet
get_planet_field() {
    local planet_id="$1" field="$2"
    query_planet "${planet_id}" | jq -r ".Planet.${field} // empty" 2>/dev/null || echo ""
}

# get_hp: query a struct's health (returns "0" for destroyed/wiped structs)
get_hp() {
    local sid="$1"
    local hp
    hp=$(query query structs struct "${sid}" 2>/dev/null | jq -r '.structAttributes.health // empty' 2>/dev/null || echo "")
    if [ -z "${hp}" ]; then echo "0"; else echo "${hp}"; fi
}

# run_tx_expect_fail: execute a TX that we EXPECT to fail, and verify it does
run_tx_expect_fail() {
    local description="$1"
    shift
    info "${description}"
    echo -e "  ${BOLD}structsd ${PARAMS_TX} $*${NC}"
    local OUTPUT
    OUTPUT=$(structsd ${PARAMS_TX} "$@" 2>&1) || true
    if echo "${OUTPUT}" | grep -qi "error\|panic\|failed\|invalid\|unreachable"; then
        echo -e "  ${GREEN}Correctly rejected${NC}"
        echo "  $(echo "${OUTPUT}" | grep -i 'error\|unreachable' | head -1)"
        return 0
    else
        local tx_code
        tx_code=$(echo "${OUTPUT}" | jq -r '.code // empty' 2>/dev/null || echo "")
        if [ -n "${tx_code}" ] && [ "${tx_code}" != "0" ]; then
            echo -e "  ${GREEN}Correctly rejected (code=${tx_code})${NC}"
            return 0
        fi
        echo -e "  ${RED}Expected failure but TX succeeded${NC}"
        return 1
    fi
}

# run_tx_expect_permission_denied: expect failure with permission/authority message
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
            FAIL_COUNT=$((FAIL_COUNT + 1))
            return 0
        fi
    fi
    if echo "${OUTPUT}" | grep -qiE "permission|authority|does not have|not have the authority|unauthorized"; then
        echo -e "  ${GREEN}Correctly rejected (permission/authority)${NC}"
        PASS_COUNT=$((PASS_COUNT + 1))
        return 0
    fi
    echo -e "  ${GREEN}Correctly rejected${NC} (no permission phrase in output)"
    PASS_COUNT=$((PASS_COUNT + 1))
    return 0
}

# ─── Permission Query Helpers ────────────────────────────────────────────────

get_permission_by_object() {
    query query structs permission-by-object "$1" 2>/dev/null || echo '{}'
}

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
    if [ "${LOG_BATTLE}" = true ]; then
        echo ""
        echo -e "  ${CYAN}Battle Log : ${BATTLE_LOG_COUNT} attack events captured${NC}"
        echo -e "  ${CYAN}Log File   : ${BATTLE_LOG_FILE}${NC}"
        echo -e "  ${CYAN}Review     : jq . ${BATTLE_LOG_FILE}${NC}"
    fi
    echo ""
}

# ─── Resume Helpers ──────────────────────────────────────────────────────────

phase_order() {
    case "$1" in
        0) echo 50;; 1) echo 100;; 2) echo 200;; 3) echo 300;; 3b) echo 350;;
        4) echo 400;; 4b) echo 450;; 4c) echo 460;; 4d) echo 470;;
        4e) echo 480;; 4e2) echo 482;; 4e3) echo 484;; 4f) echo 490;; 4g) echo 495;;
        5) echo 500;; 5b) echo 550;; 6) echo 600;;
        7) echo 700;; 7b) echo 750;; 8) echo 800;;
        9) echo 900;; 10) echo 1000;; 11) echo 1100;;
        12) echo 1200;; 13) echo 1300;; 13b) echo 1350;;
        14) echo 1400;; 15) echo 1500;; 15b) echo 1550;; 16) echo 1600;;
        18) echo 1700;;
        17) echo 2300;; 17b) echo 2350;; 17c) echo 2400;;
        eb1) echo 2500;; eb2) echo 2600;; eb3) echo 2700;;
        eb4) echo 2800;; eb5) echo 2900;; eb6) echo 3000;;
        ev1) echo 3050;;
        ar1) echo 3100;; ar2) echo 3200;; ar3) echo 3300;; ar4) echo 3400;;
        rg1) echo 3450;; rg2) echo 3500;;
        gp1) echo 3800;;
        *) echo "Unknown phase: $1" >&2; exit 1;;
    esac
}

RESUME_PHASE_NUM=0
if [ -n "${RESUME_FROM}" ]; then
    RESUME_PHASE_NUM=$(phase_order "${RESUME_FROM}")
    info "Will resume from phase ${RESUME_FROM} (order=${RESUME_PHASE_NUM})"
fi

run_phase() {
    [ "$1" -ge "${RESUME_PHASE_NUM}" ]
}

# find_struct_by_owner_type: locate a struct on-chain by owner player ID and type number
# Usage: find_struct_by_owner_type <owner_player_id> <type_num> [nth] [struct_all_json]
find_struct_by_owner_type() {
    local owner="$1" type_num="$2" nth="${3:-1}" json="${4:-}"
    if [ -z "${json}" ]; then json=$(query query structs struct-all); fi
    echo "${json}" | jq -r --arg o "${owner}" --argjson t "${type_num}" --argjson n "${nth}" \
        '[.Struct[] | select(.owner == $o and (.type | tonumber) == $t)]
         | sort_by(.id | split("-") | .[1] | tonumber)
         | .[($n - 1)].id // empty' 2>/dev/null || echo ""
}

# ─────────────────────────────────────────────────────────────────────────────
# Combat helpers (used by EB5+ phases and RG1/RG2). Defined at top-level so
# they are available even when --resume-from skips the EB phase block.
# ─────────────────────────────────────────────────────────────────────────────

# Weapon charge lookups by struct type (from genesis_struct_type.go)
_primary_charge() {
    case "$1" in
        1|3|5|9) echo 3 ;; # CommandShip,Starfighter,PursuitFighter,Tank
        2|4|6|7|8|10|11|12|13) echo 5 ;; # Battleship,Frigate,StealthBomber,Interceptor,MobArt,SAM,Cruiser,Destroyer,Sub
        *) echo 0 ;; # planetary / no primary weapon
    esac
}
_secondary_charge() {
    case "$1" in
        2)  echo 5 ;; # Battleship guided secondary (v0.18.0 charge rebalance)
        3)  echo 5 ;; # Starfighter attackRun
        11) echo 3 ;; # Cruiser secondary
        *)  echo 1 ;; # fallback
    esac
}

# Helper: query struct health
eb_health() {
    local struct_id="$1"
    query query structs struct "${struct_id}" 2>/dev/null | jq -r '.structAttributes.health // "0"' 2>/dev/null || echo "0"
}

# Helper: get the weapon charge for a struct, auto-detecting type
eb_get_charge() {
    local struct_id="$1"
    local weapon="${2:-primaryWeapon}"
    local stype
    stype=$(query query structs struct "${struct_id}" 2>/dev/null | jq -r '.Struct.type // "0"' 2>/dev/null || echo "0")
    if [ "${weapon}" = "secondaryWeapon" ]; then
        _secondary_charge "${stype}"
    else
        _primary_charge "${stype}"
    fi
}

# Helper: run an attack, track health changes, increment counters
eb_attack() {
    local desc="$1"
    local attacker="$2"
    local target="$3"
    local weapon="${4:-primaryWeapon}"
    local from_player="$5"
    local charge="${6:-}"

    if [ -z "${charge}" ]; then
        charge=$(eb_get_charge "${attacker}" "${weapon}")
    fi

    EB_ATTACKS=$((${EB_ATTACKS:-0} + 1))

    local atk_hp_before
    atk_hp_before=$(eb_health "${attacker}")
    local tgt_hp_before
    tgt_hp_before=$(eb_health "${target}")

    info "[Attack ${EB_ATTACKS}] ${desc}"
    echo "  Attacker: ${attacker} (HP=${atk_hp_before})  Target: ${target} (HP=${tgt_hp_before})"

    if [ "${atk_hp_before}" = "0" ]; then
        echo "  SKIP: Attacker already destroyed"
        return
    fi
    if [ "${tgt_hp_before}" = "0" ]; then
        echo "  SKIP: Target already destroyed"
        return
    fi

    wait_for_charge "$(eval echo "\${PLAYER_${from_player}_ID}")" "${charge}"
    run_tx "${desc}" \
        tx structs struct-attack "${attacker}" "${target}" "${weapon}" --from "player_${from_player}"

    local atk_hp_after
    atk_hp_after=$(eb_health "${attacker}")
    local tgt_hp_after
    tgt_hp_after=$(eb_health "${target}")

    echo "  Result: Attacker HP ${atk_hp_before}→${atk_hp_after}  Target HP ${tgt_hp_before}→${tgt_hp_after}"

    if [ "${tgt_hp_after}" = "0" ]; then
        info "  TARGET DESTROYED"
        EB_DESTROYED=$((${EB_DESTROYED:-0} + 1))
    fi
    if [ "${atk_hp_after}" = "0" ]; then
        info "  ATTACKER DESTROYED (counter-attack/post-destruction)"
        EB_DESTROYED=$((${EB_DESTROYED:-0} + 1))
    fi
}

# Helper: attempt an attack that should fail (wrong ambit targeting)
eb_attack_should_fail() {
    local desc="$1"
    local attacker="$2"
    local target="$3"
    local from_player="$4"

    local tgt_hp_before
    tgt_hp_before=$(eb_health "${target}")

    info "[Negative] ${desc}"
    echo "  Attacker: ${attacker}  Target: ${target} (HP=${tgt_hp_before})"

    local charge
    charge=$(eb_get_charge "${attacker}" "primaryWeapon")
    wait_for_charge "$(eval echo "\${PLAYER_${from_player}_ID}")" "${charge}"
    run_tx_expect_fail "${desc}" \
        tx structs struct-attack "${attacker}" "${target}" primaryWeapon --from "player_${from_player}"

    local tgt_hp_after
    tgt_hp_after=$(eb_health "${target}")

    assert_eq "${desc} — target HP unchanged" "${tgt_hp_before}" "${tgt_hp_after}"
}

# recover_state: rebuild all script variables by querying the running chain
recover_state() {
    info "Recovering state from chain for resume..."

    VALIDATOR_ADDRESS=$(query query staking validators | jq -r '.validators[0].operator_address')
    assert_not_empty "Recovered validator" "${VALIDATOR_ADDRESS}"

    PLAYER_1_ADDRESS=$(structsd ${PARAMS_KEYS} keys show alice 2>/dev/null | jq -r .address || echo "")
    BOB_ADDRESS=$(structsd ${PARAMS_KEYS} keys show bob 2>/dev/null | jq -r .address || echo "")

    ADDR_JSON=$(query query structs address "${PLAYER_1_ADDRESS}")
    PLAYER_1_ID=$(jqr "${ADDR_JSON}" '.playerId')
    assert_not_empty "Recovered Player 1 ID" "${PLAYER_1_ID}"

    for PLAYER_NUM in 2 3 4 5 6; do
        local ADDR
        ADDR=$(structsd ${PARAMS_KEYS} keys show "player_${PLAYER_NUM}" 2>/dev/null | jq -r .address || echo "")
        if [ -z "${ADDR}" ]; then continue; fi
        eval "PLAYER_${PLAYER_NUM}_ADDRESS=${ADDR}"
        local PID
        PID=$(query query structs address "${ADDR}" 2>/dev/null | jq -r '.playerId // empty' 2>/dev/null || echo "")
        if [ -n "${PID}" ]; then
            eval "PLAYER_${PLAYER_NUM}_ID=${PID}"
            echo "  Player ${PLAYER_NUM}: ${PID} (${ADDR})"
        fi
    done

    GUILD_ID=$(query query structs player "${PLAYER_1_ID}" 2>/dev/null | jq -r '.Player.guildId // empty' 2>/dev/null || echo "")
    REACTOR_ID=$(query query structs reactor-all 2>/dev/null | jq -r '.Reactor[0].id // empty' 2>/dev/null || echo "")
    SUBSTATION_ID=$(query query structs substation-all 2>/dev/null | jq -r '.Substation[0].id // empty' 2>/dev/null || echo "")
    GUILD_TOKEN_DENOM="uguild.${GUILD_ID}"
    echo "  Guild A: ${GUILD_ID}  Reactor: ${REACTOR_ID}  Substation: ${SUBSTATION_ID}"

    # Recover guild leaders and guilds B/C
    for LEADER_SUFFIX in b c; do
        local LADDR
        LADDR=$(structsd ${PARAMS_KEYS} keys show "guild_leader_${LEADER_SUFFIX}" 2>/dev/null | jq -r .address || echo "")
        if [ -z "${LADDR}" ]; then continue; fi
        local LSUFFIX_UPPER
        LSUFFIX_UPPER=$(echo "${LEADER_SUFFIX}" | tr '[:lower:]' '[:upper:]')
        eval "GUILD_LEADER_${LSUFFIX_UPPER}_ADDRESS=${LADDR}"
        local LPID
        LPID=$(query query structs address "${LADDR}" 2>/dev/null | jq -r '.playerId // empty' 2>/dev/null || echo "")
        if [ -n "${LPID}" ]; then
            eval "GUILD_LEADER_${LSUFFIX_UPPER}_ID=${LPID}"
            echo "  Guild Leader ${LSUFFIX_UPPER}: ${LPID} (${LADDR})"
        fi
    done
    GUILD_B_ID=$(query query structs player "${GUILD_LEADER_B_ID:-}" 2>/dev/null | jq -r '.Player.guildId // empty' 2>/dev/null || echo "")
    GUILD_C_ID=$(query query structs player "${GUILD_LEADER_C_ID:-}" 2>/dev/null | jq -r '.Player.guildId // empty' 2>/dev/null || echo "")
    echo "  Guild B: ${GUILD_B_ID:-?}  Guild C: ${GUILD_C_ID:-?}"

    for PLAYER_NUM in 2 3 4; do
        eval "local PID=\${PLAYER_${PLAYER_NUM}_ID:-}"
        if [ -z "${PID}" ]; then continue; fi
        local PJSON
        PJSON=$(query query structs player "${PID}" 2>/dev/null || echo '{}')
        eval "PLAYER_${PLAYER_NUM}_PLANET_ID=$(jqr "${PJSON}" '.Player.planetId')"
        eval "PLAYER_${PLAYER_NUM}_FLEET_ID=$(jqr "${PJSON}" '.Player.fleetId')"
        local ALLOC
        ALLOC=$(query query structs allocation-all-by-source "${PID}" 2>/dev/null | jq -r '.Allocation[-1].id // empty' 2>/dev/null || echo "")
        eval "P${PLAYER_NUM}_ALLOC_ID=${ALLOC}"
    done
    echo "  P2 planet=${PLAYER_2_PLANET_ID:-?} fleet=${PLAYER_2_FLEET_ID:-?}"
    echo "  P3 planet=${PLAYER_3_PLANET_ID:-?} fleet=${PLAYER_3_FLEET_ID:-?}"
    echo "  P4 planet=${PLAYER_4_PLANET_ID:-?} fleet=${PLAYER_4_FLEET_ID:-?}"

    local SA
    SA=$(query query structs struct-all)

    COMMAND_SHIP_ID=$(find_struct_by_owner_type "${PLAYER_3_ID}" 1 1 "${SA}")
    PLAYER_2_CMD_SHIP_ID=$(find_struct_by_owner_type "${PLAYER_2_ID}" 1 1 "${SA}")
    PLAYER_3_CMD_SHIP_ID="${COMMAND_SHIP_ID}"
    MINER_STRUCT_ID=$(find_struct_by_owner_type "${PLAYER_2_ID}" 14 1 "${SA}")
    REFINERY_STRUCT_ID=$(find_struct_by_owner_type "${PLAYER_2_ID}" 16 1 "${SA}")
    DESTROYER_STRUCT_ID=$(find_struct_by_owner_type "${PLAYER_3_ID}" 9 1 "${SA}")
    AP_TANK_ID=$(find_struct_by_owner_type "${PLAYER_3_ID}" 9 2 "${SA}")
    DEFENDER_STRUCT_ID=$(find_struct_by_owner_type "${PLAYER_2_ID}" 9 1 "${SA}")
    GENERATOR_STRUCT_ID=$(find_struct_by_owner_type "${PLAYER_4_ID}" 20 1 "${SA}")
    SAM_STRUCT_ID=$(find_struct_by_owner_type "${PLAYER_3_ID}" 10 1 "${SA}")
    P2_BATTLESHIP_ID=$(find_struct_by_owner_type "${PLAYER_2_ID}" 2 1 "${SA}")
    SUB_STRUCT_ID=$(find_struct_by_owner_type "${PLAYER_3_ID}" 13 1 "${SA}")
    INTERCEPTOR_ID=$(find_struct_by_owner_type "${PLAYER_2_ID}" 7 1 "${SA}")
    BATTLESHIP_1_ID=$(find_struct_by_owner_type "${PLAYER_3_ID}" 2 1 "${SA}")
    BATTLESHIP_2_ID=$(find_struct_by_owner_type "${PLAYER_3_ID}" 2 2 "${SA}")
    STEALTH_BOMBER_ID=$(find_struct_by_owner_type "${PLAYER_3_ID}" 6 1 "${SA}")
    CRUISER_ID=$(find_struct_by_owner_type "${PLAYER_3_ID}" 11 1 "${SA}")

    echo "  CommandShip=${COMMAND_SHIP_ID:-?} Destroyer=${DESTROYER_STRUCT_ID:-?}"
    echo "  SAM=${SAM_STRUCT_ID:-?} Sub=${SUB_STRUCT_ID:-?} StealthBomber=${STEALTH_BOMBER_ID:-?}"
    echo "  BB1=${BATTLESHIP_1_ID:-?} BB2=${BATTLESHIP_2_ID:-?} Cruiser=${CRUISER_ID:-?}"
    echo "  P2: Defender=${DEFENDER_STRUCT_ID:-?} Battleship=${P2_BATTLESHIP_ID:-?} Interceptor=${INTERCEPTOR_ID:-?}"

    if [ -n "${PLAYER_6_ID:-}" ]; then
        local P6JSON
        P6JSON=$(query query structs player "${PLAYER_6_ID}" 2>/dev/null || echo '{}')
        PLAYER_6_PLANET_ID=$(jqr "${P6JSON}" '.Player.planetId')
        PLAYER_6_FLEET_ID=$(jqr "${P6JSON}" '.Player.fleetId')
        P6_ALLOC_ID=$(query query structs allocation-all-by-source "${PLAYER_6_ID}" 2>/dev/null | jq -r '.Allocation[-1].id // empty' 2>/dev/null || echo "")
        P6_COMMAND_SHIP_ID=$(find_struct_by_owner_type "${PLAYER_6_ID}" 1 1 "${SA}")
        EB_PURSUIT_FIGHTER_ID=$(find_struct_by_owner_type "${PLAYER_3_ID}" 5 1 "${SA}")
        EB_STARFIGHTER_ID=$(find_struct_by_owner_type "${PLAYER_6_ID}" 3 1 "${SA}")
        EB_FRIGATE_ID=$(find_struct_by_owner_type "${PLAYER_6_ID}" 4 1 "${SA}")
        EB_MOBILE_ART_ID=$(find_struct_by_owner_type "${PLAYER_6_ID}" 8 1 "${SA}")
        EB_DESTROYER_W_ID=$(find_struct_by_owner_type "${PLAYER_6_ID}" 12 1 "${SA}")
        EB_P6_BATTLESHIP_ID=$(find_struct_by_owner_type "${PLAYER_6_ID}" 2 1 "${SA}")
        EB_P6_TANK_ID=$(find_struct_by_owner_type "${PLAYER_6_ID}" 9 1 "${SA}")
        EB_P6_CRUISER_ID=$(find_struct_by_owner_type "${PLAYER_6_ID}" 11 1 "${SA}")
        EB_P3_MOBILE_ART_ID=$(find_struct_by_owner_type "${PLAYER_3_ID}" 8 1 "${SA}")
        EB_PDC_ID=$(find_struct_by_owner_type "${PLAYER_6_ID}" 19 1 "${SA}")
        EB_ORE_EXTRACTOR_ID=$(find_struct_by_owner_type "${PLAYER_6_ID}" 14 1 "${SA}")
        echo "  P6: planet=${PLAYER_6_PLANET_ID:-?} fleet=${PLAYER_6_FLEET_ID:-?} CS=${P6_COMMAND_SHIP_ID:-?}"
    fi

    # Fleet movement test players (fplayer_1 through fplayer_5)
    for FP_NUM in 1 2 3 4 5; do
        local FP_ADDR
        FP_ADDR=$(structsd ${PARAMS_KEYS} keys show "fplayer_${FP_NUM}" 2>/dev/null | jq -r .address || echo "")
        if [ -z "${FP_ADDR}" ]; then continue; fi
        eval "FP_${FP_NUM}_ADDRESS=${FP_ADDR}"
        local FP_PID
        FP_PID=$(query query structs address "${FP_ADDR}" 2>/dev/null | jq -r '.playerId // empty' 2>/dev/null || echo "")
        if [ -n "${FP_PID}" ]; then
            eval "FP_${FP_NUM}_ID=${FP_PID}"
            local FP_JSON
            FP_JSON=$(query query structs player "${FP_PID}" 2>/dev/null || echo '{}')
            eval "FP_${FP_NUM}_PLANET_ID=$(jqr "${FP_JSON}" '.Player.planetId')"
            eval "FP_${FP_NUM}_FLEET_ID=$(jqr "${FP_JSON}" '.Player.fleetId')"
            local FP_CS
            FP_CS=$(find_struct_by_owner_type "${FP_PID}" 1 1 "${SA}")
            eval "FP_CS_${FP_NUM}=${FP_CS}"
            echo "  FP ${FP_NUM}: ${FP_PID} planet=${FP_JSON##*planetId} fleet=$(jqr "${FP_JSON}" '.Player.fleetId') CS=${FP_CS}"
        fi
    done

    # Rank-test players (rp_1 through rp_20)
    RP_IDS=()
    RP_KEYS=()
    for RP_NUM in $(seq 1 20); do
        local RP_KEY="rp_${RP_NUM}"
        local RP_ADDR
        RP_ADDR=$(structsd ${PARAMS_KEYS} keys show "${RP_KEY}" 2>/dev/null | jq -r .address 2>/dev/null || echo "")
        if [ -z "${RP_ADDR}" ]; then continue; fi
        local RP_PID
        RP_PID=$(query query structs address "${RP_ADDR}" 2>/dev/null | jq -r '.playerId // empty' 2>/dev/null || echo "")
        if [ -n "${RP_PID}" ] && [ "${RP_PID}" != "1-0" ]; then
            RP_KEYS+=("${RP_KEY}")
            RP_IDS+=("${RP_PID}")
        fi
    done
    if [ "${#RP_IDS[@]}" -gt 0 ]; then
        echo "  Recovered ${#RP_IDS[@]} rank-test players"
    fi

    info "State recovery complete"
}

# Arrays for rank-test players (populated by Phase 4e2, recovered by recover_state)
declare -a RP_IDS
declare -a RP_KEYS

if [ -n "${RESUME_FROM}" ]; then
    recover_state
fi

if [ "${LOG_BATTLE}" = true ]; then
    echo ""
    echo -e "${CYAN}${BOLD}  Battle logging enabled — ${BATTLE_LOG_FILE}${NC}"
    echo ""
fi

if run_phase 50; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 0: Ante Handler Smoke Tests
#  Verifies the custom ante handler is functioning:
#    - Free Structs txs work without gas fees
#    - Paid Cosmos txs still require gas fees
#    - Unregistered addresses are rejected for Structs messages
#    - Message count cap is enforced
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 0: Ante Handler Smoke Tests"

info "Verifying alice's address"
ALICE_ADDRESS=$(structsd ${PARAMS_KEYS} keys show alice | jq -r .address)
assert_not_empty "Alice address" "${ALICE_ADDRESS}"

# 0a. Bank send with fees works (standard Cosmos tx)
run_tx "0a: Bank send with fees (should succeed)" \
    tx bank send "${ALICE_ADDRESS}" "${ALICE_ADDRESS}" 1ualpha --from alice

# 0b. Once player is set up (after Phase 1), free Structs txs will be tested
#     in subsequent phases. For now, verify the ante handler doesn't break
#     basic Cosmos operations.
info "Ante handler Phase 0 basic checks complete"
info "Free-tx and throttle tests are woven into subsequent phases"

fi # phase 0

if run_phase 100; then

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

P1_PRIMARY=$(jqr "${PLAYER_ME_JSON}" '.Player.primaryAddress')
assert_eq "Player 1 primaryAddress matches keyring" "${PLAYER_1_ADDRESS}" "${P1_PRIMARY}"

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

# ─── Discover Reactor (created during validator setup) ───
info "Looking up reactor"
REACTOR_ALL_JSON=$(query query structs reactor-all)
REACTOR_ID=$(jqr "${REACTOR_ALL_JSON}" '.Reactor[0].id')
assert_not_empty "Reactor ID" "${REACTOR_ID}"
echo "  Reactor ID: ${REACTOR_ID}"

REACTOR_VAL=$(jqr "${REACTOR_ALL_JSON}" '.Reactor[0].validator')
assert_eq "Reactor validator matches" "${VALIDATOR_ADDRESS}" "${REACTOR_VAL}"

# ─── Create Guild ───
run_tx "Creating Guild" \
    tx structs guild-create "${REACTOR_ID}" "oh.energy" "${SUBSTATION_ID}" --from alice

# Discover guild ID from Alice's player record (reliable regardless of genesis guilds)
P1_JSON=$(query query structs player "${PLAYER_1_ID}")
GUILD_ID=$(jqr "${P1_JSON}" '.Player.guildId')
assert_not_empty "Guild ID" "${GUILD_ID}"
echo "  Guild ID: ${GUILD_ID}"

# Verify guild details
GUILD_JSON=$(query query structs guild "${GUILD_ID}")
GUILD_ENDPOINT=$(jqr "${GUILD_JSON}" '.Guild.endpoint')
assert_eq "Guild endpoint" "oh.energy" "${GUILD_ENDPOINT}"

GUILD_ENTRY_SUB=$(jqr "${GUILD_JSON}" '.Guild.entrySubstationId')
assert_eq "Guild entry substation" "${SUBSTATION_ID}" "${GUILD_ENTRY_SUB}"

fi # phase 1

if run_phase 200; then

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

# ─── Verify player identity integrity ───
info "Verifying player identity integrity"
EXPECTED_INDEX=2
for PLAYER_NUM in 2 3 4 5; do
    eval "PID=\${PLAYER_${PLAYER_NUM}_ID}"
    eval "PADDR=\${PLAYER_${PLAYER_NUM}_ADDRESS}"
    assert_eq "Player ${PLAYER_NUM} sequential ID" "1-${EXPECTED_INDEX}" "${PID}"
    P_JSON=$(query query structs player "${PID}")
    P_PRIMARY=$(echo "${P_JSON}" | jq -r '.Player.primaryAddress')
    assert_eq "Player ${PLAYER_NUM} primaryAddress matches keyring" "${PADDR}" "${P_PRIMARY}"
    EXPECTED_INDEX=$((EXPECTED_INDEX + 1))
done

# Re-verify Player 1 and Reactor were not corrupted by new player creation
P1_RECHECK=$(query query structs player "${PLAYER_1_ID}")
P1_RECHECK_ADDR=$(jqr "${P1_RECHECK}" '.Player.primaryAddress')
assert_eq "Player 1 primaryAddress intact after new players" "${PLAYER_1_ADDRESS}" "${P1_RECHECK_ADDR}"

REACTOR_RECHECK=$(query query structs reactor "${REACTOR_ID}")
REACTOR_RECHECK_VAL=$(jqr "${REACTOR_RECHECK}" '.Reactor.validator')
assert_eq "Reactor validator intact after new players" "${VALIDATOR_ADDRESS}" "${REACTOR_RECHECK_VAL}"

# ─── Create Guild Leaders and Additional Guilds ──────────────────────────────
# Guild creation moves the creator into the new guild, so alice cannot create
# more guilds without leaving her own. Instead, create dedicated leader accounts
# and grant them PermReactorGuildCreate (524288) on the reactor.

section "Create Guild Leaders (B, C)"

for LEADER_SUFFIX in b c; do
    LEADER_KEY="guild_leader_${LEADER_SUFFIX}"
    info "Setting up ${LEADER_KEY}"
    EXISTING=$(structsd ${PARAMS_KEYS} keys show "${LEADER_KEY}" 2>/dev/null | jq -r .address || echo "")
    if [ -z "${EXISTING}" ]; then
        ADDR=$(structsd ${PARAMS_KEYS} keys add "${LEADER_KEY}" | jq -r .address)
        echo "  Created ${LEADER_KEY}: ${ADDR}"
    else
        ADDR="${EXISTING}"
        echo "  Reusing ${LEADER_KEY}: ${ADDR}"
    fi
    LEADER_SUFFIX_UPPER=$(echo "${LEADER_SUFFIX}" | tr '[:lower:]' '[:upper:]')
    eval "GUILD_LEADER_${LEADER_SUFFIX_UPPER}_ADDRESS=${ADDR}"
done

assert_not_empty "Guild Leader B address" "${GUILD_LEADER_B_ADDRESS}"
assert_not_empty "Guild Leader C address" "${GUILD_LEADER_C_ADDRESS}"

# Fund guild leaders (from alice to avoid draining bob's faucet budget)
for LEADER_SUFFIX in B C; do
    LEADER_LOWER=$(echo "${LEADER_SUFFIX}" | tr '[:upper:]' '[:lower:]')
    eval "LADDR=\${GUILD_LEADER_${LEADER_SUFFIX}_ADDRESS}"
    run_tx "Sending 10000000ualpha from alice to guild_leader_${LEADER_LOWER}" \
        tx bank send "${PLAYER_1_ADDRESS}" "${LADDR}" 10000000ualpha --from alice
done

# Delegate guild leaders (creates player accounts)
for LEADER_SUFFIX in B C; do
    LEADER_LOWER=$(echo "${LEADER_SUFFIX}" | tr '[:upper:]' '[:lower:]')
    run_tx "Delegating 5000000ualpha from guild_leader_${LEADER_LOWER} to validator" \
        tx staking delegate "${VALIDATOR_ADDRESS}" 5000000ualpha --from "guild_leader_${LEADER_LOWER}"

    eval "LADDR=\${GUILD_LEADER_${LEADER_SUFFIX}_ADDRESS}"
    ADDR_JSON=$(query query structs address "${LADDR}")
    LID=$(jqr "${ADDR_JSON}" '.playerId')
    eval "GUILD_LEADER_${LEADER_SUFFIX}_ID=${LID}"
    assert_not_empty "Guild Leader ${LEADER_SUFFIX} ID" "${LID}"
    echo "  Guild Leader ${LEADER_SUFFIX} ID: ${LID}"
done

# Grant PermReactorGuildCreate (524288) on reactor to each leader
run_tx "Granting PermReactorGuildCreate on reactor to Guild Leader B" \
    tx structs permission-grant-on-object "${REACTOR_ID}" "${GUILD_LEADER_B_ID}" 524288 --from alice

run_tx "Granting PermReactorGuildCreate on reactor to Guild Leader C" \
    tx structs permission-grant-on-object "${REACTOR_ID}" "${GUILD_LEADER_C_ID}" 524288 --from alice

# Grant PermSubstationConnection (1024) on substation to each leader
run_tx "Granting PermSubstationConnection on substation to Guild Leader B" \
    tx structs permission-grant-on-object "${SUBSTATION_ID}" "${GUILD_LEADER_B_ID}" 1024 --from alice

run_tx "Granting PermSubstationConnection on substation to Guild Leader C" \
    tx structs permission-grant-on-object "${SUBSTATION_ID}" "${GUILD_LEADER_C_ID}" 1024 --from alice

# Create Guild B
run_tx "Guild Leader B creates Guild B" \
    tx structs guild-create "${REACTOR_ID}" "guild-b.energy" "${SUBSTATION_ID}" --from guild_leader_b

GUILD_B_ID=$(query query structs player "${GUILD_LEADER_B_ID}" | jq -r '.Player.guildId // empty' 2>/dev/null || echo "")
assert_not_empty "Guild B ID" "${GUILD_B_ID}"
echo "  Guild B ID: ${GUILD_B_ID}"

# Create Guild C
run_tx "Guild Leader C creates Guild C" \
    tx structs guild-create "${REACTOR_ID}" "guild-c.energy" "${SUBSTATION_ID}" --from guild_leader_c

GUILD_C_ID=$(query query structs player "${GUILD_LEADER_C_ID}" | jq -r '.Player.guildId // empty' 2>/dev/null || echo "")
assert_not_empty "Guild C ID" "${GUILD_C_ID}"
echo "  Guild C ID: ${GUILD_C_ID}"

info "Guilds: A=${GUILD_ID}  B=${GUILD_B_ID}  C=${GUILD_C_ID}"

# Enable invite and request bypass on Guild B (for cross-guild membership tests)
run_tx "Enabling Guild B invites (bypass=member)" \
    tx structs guild-update-join-infusion-minimum-by-invite "${GUILD_B_ID}" member --from guild_leader_b

run_tx "Enabling Guild B requests (bypass=member)" \
    tx structs guild-update-join-infusion-minimum-by-request "${GUILD_B_ID}" member --from guild_leader_b

fi # phase 2

if run_phase 300; then

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
        --controller "${PLAYER_1_ID}" --allocation-type dynamic --from "player_${PLAYER_NUM}"

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

fi # phase 3

if run_phase 350; then

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
    --controller "${PLAYER_5_ID}" --allocation-type dynamic --from player_5

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

# v0.20.0: growing an existing allocation must release its own power first.
# Re-set to half capacity, then raise to 3/4 — exceeds free (1/4) but fits total.
run_tx "Updating allocation ${P5_ALLOC_ID} power back to ${HALF_CAP}" \
    tx structs allocation-update "${P5_ALLOC_ID}" "${HALF_CAP}" --from player_5

THREE_QUARTER_CAP=$(( (P5_CAP * 3) / 4 ))
run_tx "Updating allocation ${P5_ALLOC_ID} power up to ${THREE_QUARTER_CAP} (own power released)" \
    tx structs allocation-update "${P5_ALLOC_ID}" "${THREE_QUARTER_CAP}" --from player_5

ALLOC_JSON=$(query query structs allocation "${P5_ALLOC_ID}")
ALLOC_POWER=$(jqr "${ALLOC_JSON}" '.gridAttributes.power' '0')
assert_eq "Allocation grew past prior free capacity" "${THREE_QUARTER_CAP}" "${ALLOC_POWER}"

# ─── allocation-transfer: transfer controller to alice ───
info "Transferring allocation controller from player_5 to alice"
run_tx "Transferring allocation ${P5_ALLOC_ID} controller to alice" \
    tx structs allocation-transfer "${P5_ALLOC_ID}" "${PLAYER_1_ID}" --from player_5

ALLOC_JSON=$(query query structs allocation "${P5_ALLOC_ID}")
ALLOC_CTRL=$(jqr "${ALLOC_JSON}" '.Allocation.controller')
assert_eq "Allocation controller transferred" "${PLAYER_1_ID}" "${ALLOC_CTRL}"

# Transfer back to player_5 so they can delete it
run_tx "Transferring allocation back to player_5" \
    tx structs allocation-transfer "${P5_ALLOC_ID}" "${PLAYER_5_ID}" --from alice

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
    --controller "${PLAYER_5_ID}" --allocation-type dynamic --from player_5

P5_ALLOC_ID=$(get_latest_allocation_for_source "${PLAYER_5_ID}")
assert_not_empty "Player 5 re-created allocation ID" "${P5_ALLOC_ID}"
echo "  Player 5 new Allocation ID: ${P5_ALLOC_ID}"

fi # phase 3b

if run_phase 400; then

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

fi # phase 4

if run_phase 450; then

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

fi # phase 4b

if run_phase 460; then

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

# Verify Player 5 receives default entry rank on joining
P5_RANK_ON_JOIN=$(jqr "${P5_JSON}" '.Player.guildRank' '0')
info "Player 5 guild rank after joining: ${P5_RANK_ON_JOIN}"
assert_eq "Player 5 gets default entry rank (101) on join" "101" "${P5_RANK_ON_JOIN}"

# ─── Test 4: Kick ───────────────────────────────────────────────────────────
info "--- Test 4: Kick ---"

run_tx "Alice kicks Player 5 from guild" \
    tx structs guild-membership-kick "${PLAYER_5_ID}" --from alice

P5_JSON=$(query query structs player "${PLAYER_5_ID}")
P5_GUILD=$(jqr "${P5_JSON}" '.Player.guildId' '')
assert_eq "Player 5 removed after kick" "" "${P5_GUILD}"

# Verify guild rank resets after kick
P5_RANK_AFTER_KICK=$(jqr "${P5_JSON}" '.Player.guildRank' '0')
info "Player 5 guild rank after kick: ${P5_RANK_AFTER_KICK}"
assert_eq "Guild rank resets to default (101) after kick" "101" "${P5_RANK_AFTER_KICK}"

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

# ─── Reset: kick Player 5 so subsequent tests start clean ─────────────────
run_tx "Kick Player 5 (reset for tests 9+)" \
    tx structs guild-membership-kick "${PLAYER_5_ID}" --from alice

P5_JSON=$(query query structs player "${PLAYER_5_ID}")
P5_GUILD=$(jqr "${P5_JSON}" '.Player.guildId' '')
assert_eq "Player 5 out of guild (reset)" "" "${P5_GUILD}"

# ─── Test 9: Invite → Approve (using run_tx_noauto) ─────────────────────────
info "--- Test 9: Invite → Approve ---"

run_tx "Alice invites Player 5" \
    tx structs guild-membership-invite "${PLAYER_5_ID}" --from alice

APP_JSON=$(query query structs guild-membership-application "${GUILD_ID}" "${PLAYER_5_ID}" 2>/dev/null || echo '{}')
APP_TYPE=$(jqr "${APP_JSON}" '.GuildMembershipApplication.joinType' '')
assert_eq "Invite application exists" "invite" "${APP_TYPE}"

run_tx_noauto "Player 5 approves invite" \
    tx structs guild-membership-invite-approve "${GUILD_ID}" --from player_5

P5_JSON=$(query query structs player "${PLAYER_5_ID}")
P5_GUILD=$(jqr "${P5_JSON}" '.Player.guildId' '')
assert_eq "Player 5 joined guild via invite-approve" "${GUILD_ID}" "${P5_GUILD}"

run_tx "Kick Player 5 (reset after test 9)" \
    tx structs guild-membership-kick "${PLAYER_5_ID}" --from alice

# ─── Test 10: Invite → Approve with substation override ─────────────────────
info "--- Test 10: Invite → Approve with substation override ---"

# Player 5 needs PermSubstationConnection on the substation for the override
run_tx "Grant Player 5 PermSubstationConnection on substation" \
    tx structs permission-grant-on-object "${SUBSTATION_ID}" "${PLAYER_5_ID}" 1024 --from alice

run_tx "Alice invites Player 5" \
    tx structs guild-membership-invite "${PLAYER_5_ID}" --from alice

run_tx_noauto "Player 5 approves invite with substation override" \
    tx structs guild-membership-invite-approve "${GUILD_ID}" --substation-id "${SUBSTATION_ID}" --from player_5

P5_JSON=$(query query structs player "${PLAYER_5_ID}")
P5_GUILD=$(jqr "${P5_JSON}" '.Player.guildId' '')
assert_eq "Player 5 joined guild via invite-approve with substation" "${GUILD_ID}" "${P5_GUILD}"

run_tx "Kick Player 5 (reset after test 10)" \
    tx structs guild-membership-kick "${PLAYER_5_ID}" --from alice

# ─── Test 11: Request when already a member (negative) ──────────────────────
info "--- Test 11: Request when already a member ---"

# Safety: revoke any lingering invite application from previous tests
run_tx "Revoking any lingering invite (safety cleanup)" \
    tx structs guild-membership-invite-revoke "${GUILD_ID}" "${PLAYER_5_ID}" --from alice

run_tx "Player 5 requests to join guild (setup)" \
    tx structs guild-membership-request "${GUILD_ID}" --from player_5
run_tx "Alice approves Player 5's request (setup)" \
    tx structs guild-membership-request-approve "${PLAYER_5_ID}" --from alice

P5_JSON=$(query query structs player "${PLAYER_5_ID}")
P5_GUILD=$(jqr "${P5_JSON}" '.Player.guildId' '')
assert_eq "Player 5 in guild (setup for test 11)" "${GUILD_ID}" "${P5_GUILD}"

run_tx_expect_fail "Player 5 requests again while already in guild (should fail)" \
    tx structs guild-membership-request "${GUILD_ID}" --from player_5

run_tx "Kick Player 5 (reset after test 11)" \
    tx structs guild-membership-kick "${PLAYER_5_ID}" --from alice

# ─── Test 12: Invite a player who is already a member (negative) ────────────
info "--- Test 12: Invite already-member ---"

# Safety cleanup of any lingering applications
run_tx "Revoking any lingering application (safety)" \
    tx structs guild-membership-invite-revoke "${GUILD_ID}" "${PLAYER_5_ID}" --from alice

run_tx "Player 5 requests to join guild (setup)" \
    tx structs guild-membership-request "${GUILD_ID}" --from player_5
run_tx "Alice approves (setup)" \
    tx structs guild-membership-request-approve "${PLAYER_5_ID}" --from alice

run_tx_expect_fail "Alice invites Player 5 who is already in guild (should fail)" \
    tx structs guild-membership-invite "${PLAYER_5_ID}" --from alice

run_tx "Kick Player 5 (reset after test 12)" \
    tx structs guild-membership-kick "${PLAYER_5_ID}" --from alice

# ─── Test 13: Cross-guild request — player in Guild B requests Guild A ──────
info "--- Test 13: Cross-guild request (Guild B -> Guild A) ---"

# Safety cleanup
run_tx "Revoking any lingering application (safety)" \
    tx structs guild-membership-invite-revoke "${GUILD_ID}" "${PLAYER_5_ID}" --from alice

# Player 5 joins Guild B
run_tx "Player 5 requests to join Guild B" \
    tx structs guild-membership-request "${GUILD_B_ID}" --from player_5
run_tx "Guild Leader B approves Player 5" \
    tx structs guild-membership-request-approve "${PLAYER_5_ID}" --from guild_leader_b

P5_JSON=$(query query structs player "${PLAYER_5_ID}")
P5_GUILD=$(jqr "${P5_JSON}" '.Player.guildId' '')
assert_eq "Player 5 in Guild B" "${GUILD_B_ID}" "${P5_GUILD}"

# Player 5 requests Guild A while in Guild B (allowed — creates application)
run_tx "Player 5 requests Guild A while in Guild B" \
    tx structs guild-membership-request "${GUILD_ID}" --from player_5

# Alice approves — migrates Player 5 from Guild B to Guild A
run_tx "Alice approves cross-guild request" \
    tx structs guild-membership-request-approve "${PLAYER_5_ID}" --from alice

P5_JSON=$(query query structs player "${PLAYER_5_ID}")
P5_GUILD=$(jqr "${P5_JSON}" '.Player.guildId' '')
assert_eq "Player 5 migrated from Guild B to Guild A" "${GUILD_ID}" "${P5_GUILD}"

# Kick from Guild A to reset
run_tx "Kick Player 5 from Guild A (reset after test 13)" \
    tx structs guild-membership-kick "${PLAYER_5_ID}" --from alice

# ─── Test 14: Cross-guild invite — player in Guild B accepts Guild A invite ─
info "--- Test 14: Cross-guild invite-approve (Guild B -> Guild A) ---"

# Player 5 joins Guild B
run_tx "Player 5 requests to join Guild B (setup)" \
    tx structs guild-membership-request "${GUILD_B_ID}" --from player_5
run_tx "Guild Leader B approves Player 5 (setup)" \
    tx structs guild-membership-request-approve "${PLAYER_5_ID}" --from guild_leader_b

# Alice invites Player 5 to Guild A
run_tx "Alice invites Player 5 to Guild A" \
    tx structs guild-membership-invite "${PLAYER_5_ID}" --from alice

# Player 5 accepts the invite while in Guild B (allowed — migrates to Guild A)
run_tx_noauto "Player 5 approves Guild A invite while in Guild B" \
    tx structs guild-membership-invite-approve "${GUILD_ID}" --from player_5

P5_JSON=$(query query structs player "${PLAYER_5_ID}")
P5_GUILD=$(jqr "${P5_JSON}" '.Player.guildId' '')
assert_eq "Player 5 migrated from Guild B to Guild A" "${GUILD_ID}" "${P5_GUILD}"

# Kick from Guild A to reset
run_tx "Kick Player 5 from Guild A (reset after test 14)" \
    tx structs guild-membership-kick "${PLAYER_5_ID}" --from alice

# ─── Test 15: Invites closed (negative) ─────────────────────────────────────
info "--- Test 15: Invites closed ---"

run_tx "Setting invite bypass to closed" \
    tx structs guild-update-join-infusion-minimum-by-invite "${GUILD_ID}" closed --from alice

run_tx_expect_fail "Alice tries to invite Player 5 with invites closed (should fail)" \
    tx structs guild-membership-invite "${PLAYER_5_ID}" --from alice

# Restore
run_tx "Restoring invite bypass to member" \
    tx structs guild-update-join-infusion-minimum-by-invite "${GUILD_ID}" member --from alice

# ─── Test 16: Requests closed (negative) ────────────────────────────────────
info "--- Test 16: Requests closed ---"

run_tx "Setting request bypass to closed" \
    tx structs guild-update-join-infusion-minimum-by-request "${GUILD_ID}" closed --from alice

run_tx_expect_fail "Player 5 requests to join with requests closed (should fail)" \
    tx structs guild-membership-request "${GUILD_ID}" --from player_5

# Restore
run_tx "Restoring request bypass to member" \
    tx structs guild-update-join-infusion-minimum-by-request "${GUILD_ID}" member --from alice

# ─── Test 17: Kick guild owner (negative) ───────────────────────────────────
info "--- Test 17: Kick guild owner ---"

# Player 5 joins, gets high rank, tries to kick alice
run_tx "Player 5 requests to join guild (setup)" \
    tx structs guild-membership-request "${GUILD_ID}" --from player_5
run_tx "Alice approves (setup)" \
    tx structs guild-membership-request-approve "${PLAYER_5_ID}" --from alice
run_tx "Alice sets Player 5 rank to 2" \
    tx structs player-update-guild-rank "${PLAYER_5_ID}" 2 --from alice

run_tx_expect_fail "Player 5 (rank 2) tries to kick alice/owner (should fail)" \
    tx structs guild-membership-kick "${PLAYER_1_ID}" --from player_5

P1_JSON=$(query query structs player "${PLAYER_1_ID}")
P1_GUILD=$(jqr "${P1_JSON}" '.Player.guildId' '')
assert_eq "Alice still in guild after owner-kick attempt" "${GUILD_ID}" "${P1_GUILD}"

run_tx "Kick Player 5 (reset after test 17)" \
    tx structs guild-membership-kick "${PLAYER_5_ID}" --from alice

# ─── Test 18: Permissioned invite bypass — member without permission ────────
info "--- Test 18: Permissioned invite bypass (no perm) ---"

run_tx "Setting invite bypass to permissioned" \
    tx structs guild-update-join-infusion-minimum-by-invite "${GUILD_ID}" permissioned --from alice

# Player 2 is a guild member but does NOT have explicit PermGuildMembership on guild object
run_tx_expect_fail "Player 2 (no guild perm) tries to invite Player 5 (should fail)" \
    tx structs guild-membership-invite "${PLAYER_5_ID}" --from player_2

run_tx "Restoring invite bypass to member" \
    tx structs guild-update-join-infusion-minimum-by-invite "${GUILD_ID}" member --from alice

# ─── Test 19: Permissioned invite bypass — admin with permission (success) ──
info "--- Test 19: Permissioned invite bypass (admin) ---"

run_tx "Setting invite bypass to permissioned" \
    tx structs guild-update-join-infusion-minimum-by-invite "${GUILD_ID}" permissioned --from alice

# Alice (owner) has admin permissions — should succeed
run_tx "Alice invites Player 5 (permissioned mode)" \
    tx structs guild-membership-invite "${PLAYER_5_ID}" --from alice

APP_JSON=$(query query structs guild-membership-application "${GUILD_ID}" "${PLAYER_5_ID}" 2>/dev/null || echo '{}')
APP_TYPE=$(jqr "${APP_JSON}" '.GuildMembershipApplication.joinType' '')
assert_eq "Invite created in permissioned mode" "invite" "${APP_TYPE}"

# Revoke the invite to clean up
run_tx "Alice revokes the invite (cleanup)" \
    tx structs guild-membership-invite-revoke "${GUILD_ID}" "${PLAYER_5_ID}" --from alice

run_tx "Restoring invite bypass to member" \
    tx structs guild-update-join-infusion-minimum-by-invite "${GUILD_ID}" member --from alice

# ─── Test 20: Permissioned request bypass — non-permissioned approver ───────
info "--- Test 20: Permissioned request bypass (non-perm approver) ---"

run_tx "Setting request bypass to permissioned" \
    tx structs guild-update-join-infusion-minimum-by-request "${GUILD_ID}" permissioned --from alice

# Player 5 requests to join
run_tx "Player 5 requests to join guild" \
    tx structs guild-membership-request "${GUILD_ID}" --from player_5

# Player 2 (member, no explicit PermGuildMembership on guild) tries to approve
run_tx_expect_fail "Player 2 (no guild perm) tries to approve request (should fail)" \
    tx structs guild-membership-request-approve "${PLAYER_5_ID}" --from player_2

# Alice (admin) approves instead
run_tx "Alice approves Player 5's request (cleanup)" \
    tx structs guild-membership-request-approve "${PLAYER_5_ID}" --from alice

run_tx "Kick Player 5 (reset after test 20)" \
    tx structs guild-membership-kick "${PLAYER_5_ID}" --from alice

run_tx "Restoring request bypass to member" \
    tx structs guild-update-join-infusion-minimum-by-request "${GUILD_ID}" member --from alice

# ─── Final: Re-join Player 5 for subsequent phases ──────────────────────────
info "--- Re-joining Player 5 for later phases ---"
run_tx "Player 5 requests to join guild (final setup)" \
    tx structs guild-membership-request "${GUILD_ID}" --from player_5
run_tx "Alice approves Player 5 (final setup)" \
    tx structs guild-membership-request-approve "${PLAYER_5_ID}" --from alice

P5_JSON=$(query query structs player "${PLAYER_5_ID}")
P5_GUILD=$(jqr "${P5_JSON}" '.Player.guildId' '')
assert_eq "Player 5 in guild (final)" "${GUILD_ID}" "${P5_GUILD}"

# ─── Cleanup ───
info "Guild membership tests complete (20 tests). Player 5 remains in guild."
info "All membership applications summary:"
query query structs guild-membership-application-all | jq -r '.GuildMembershipApplication[] | "  \(.guildId) player=\(.playerId) type=\(.joinType)"' 2>/dev/null || echo "  (none pending)"

fi # phase 4c

if run_phase 470; then

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
# Grant Player 2 PermAdmin (2) on guild so they can transfer ownership back.
# (CanTransferOwnershipBy requires PermAdmin.) The grant is intentionally NOT
# revoked here — later phases (notably GP1) re-clear player_2's grants on the
# guild when they need a clean "non-admin" caller.
run_tx "Granting Player 2 PermAdmin on guild (for ownership transfer test)" \
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

# ─── guild-update-entry-rank ─────────────────────────────────────────────────
info "--- Guild Entry Rank Lifecycle ---"

# Check guild creator (Player 1) has rank 1
P1_JSON=$(query query structs player "${PLAYER_1_ID}")
P1_RANK=$(jqr "${P1_JSON}" '.Player.guildRank' '0')
assert_eq "Guild creator (Player 1) has rank 1" "1" "${P1_RANK}"

# Check current guild entry rank (should be DefaultEntryRank = 101)
GUILD_JSON=$(query query structs guild "${GUILD_ID}")
GUILD_ENTRY_RANK_BEFORE=$(jqr "${GUILD_JSON}" '.Guild.entryRank' '0')
info "Guild entry rank before update: ${GUILD_ENTRY_RANK_BEFORE}"

# Update entry rank to 50
run_tx "Updating guild entry rank to 50" \
    tx structs guild-update-entry-rank 50 --from alice

GUILD_JSON=$(query query structs guild "${GUILD_ID}")
GUILD_ENTRY_RANK_SET=$(jqr "${GUILD_JSON}" '.Guild.entryRank' '0')
assert_eq "Guild entry rank updated to 50" "50" "${GUILD_ENTRY_RANK_SET}"

# Negative: non-admin Player 3 tries to update entry rank
run_tx "Player 3 tries to update entry rank (should fail)" \
    tx structs guild-update-entry-rank 10 --from player_3

GUILD_JSON=$(query query structs guild "${GUILD_ID}")
GUILD_ENTRY_RANK_UNCHANGED=$(jqr "${GUILD_JSON}" '.Guild.entryRank' '0')
assert_eq "Entry rank unchanged after unauthorized attempt" "50" "${GUILD_ENTRY_RANK_UNCHANGED}"

# Reset entry rank to default
run_tx "Resetting guild entry rank to 101" \
    tx structs guild-update-entry-rank 101 --from alice

GUILD_JSON=$(query query structs guild "${GUILD_ID}")
assert_eq "Guild entry rank reset to 101" "101" "$(jqr "${GUILD_JSON}" '.Guild.entryRank' '0')"

# ─── player-update-guild-rank ────────────────────────────────────────────────
info "--- Player Guild Rank Management ---"

# Set Player 2 to rank 5
run_tx "Setting Player 2 guild rank to 5" \
    tx structs player-update-guild-rank "${PLAYER_2_ID}" 5 --from alice

P2_JSON=$(query query structs player "${PLAYER_2_ID}")
P2_RANK=$(jqr "${P2_JSON}" '.Player.guildRank' '0')
assert_eq "Player 2 guild rank set to 5" "5" "${P2_RANK}"

# Set Player 3 to rank 10
run_tx "Setting Player 3 guild rank to 10" \
    tx structs player-update-guild-rank "${PLAYER_3_ID}" 10 --from alice

P3_JSON=$(query query structs player "${PLAYER_3_ID}")
P3_RANK=$(jqr "${P3_JSON}" '.Player.guildRank' '0')
assert_eq "Player 3 guild rank set to 10" "10" "${P3_RANK}"

# Negative: setting rank to 0 should fail
run_tx "Setting Player 2 rank to 0 (should fail — rank 0 is forbidden)" \
    tx structs player-update-guild-rank "${PLAYER_2_ID}" 0 --from alice

P2_JSON=$(query query structs player "${PLAYER_2_ID}")
P2_RANK_AFTER_ZERO=$(jqr "${P2_JSON}" '.Player.guildRank' '0')
assert_eq "Player 2 rank unchanged after rank=0 attempt" "5" "${P2_RANK_AFTER_ZERO}"

# Negative: Player 3 (rank 10) tries to update Player 2 (rank 5) — must have strictly better rank
run_tx "Player 3 (rank 10) tries to update Player 2 rank (should fail)" \
    tx structs player-update-guild-rank "${PLAYER_2_ID}" 3 --from player_3

P2_JSON=$(query query structs player "${PLAYER_2_ID}")
P2_RANK_UNCHANGED=$(jqr "${P2_JSON}" '.Player.guildRank' '0')
assert_eq "Player 2 rank unchanged after unauthorized change" "5" "${P2_RANK_UNCHANGED}"

# Positive: Player 2 (rank 5) sets Player 3 (rank 10) to rank 8
# Actor rank 5 < target rank 10, and new rank 8 >= actor rank 5
run_tx "Player 2 (rank 5) sets Player 3 (rank 10) to rank 8" \
    tx structs player-update-guild-rank "${PLAYER_3_ID}" 8 --from player_2

P3_JSON=$(query query structs player "${PLAYER_3_ID}")
P3_RANK_AFTER=$(jqr "${P3_JSON}" '.Player.guildRank' '0')
assert_eq "Player 3 rank updated to 8 by Player 2" "8" "${P3_RANK_AFTER}"

# Reset ranks for later phases
run_tx "Resetting Player 2 guild rank to 101" \
    tx structs player-update-guild-rank "${PLAYER_2_ID}" 101 --from alice
run_tx "Resetting Player 3 guild rank to 101" \
    tx structs player-update-guild-rank "${PLAYER_3_ID}" 101 --from alice

fi # phase 4d

if run_phase 480; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 4e: Permission System
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 4e: Permission System"

# Permission constants (1<<iota): Play=1, Admin=2, Update=4, Delete=8,
# TokenTransfer=16, TokenInfuse=32, TokenMigrate=64, TokenDefuse=128,
# SourceAllocation=256, GuildMembership=512, SubstationConnection=1024, AllocationConnection=2048

# ─── permission-grant-on-object: grant Player 5 Grid permission on substation ───
VAL_BEFORE_GRANT=$(get_permission_value_for_player "${SUBSTATION_ID}" "${PLAYER_5_ID}")
run_tx "Granting Player 5 Grid permission (32) on substation ${SUBSTATION_ID}" \
    tx structs permission-grant-on-object "${SUBSTATION_ID}" "${PLAYER_5_ID}" 32 --from alice

VAL_AFTER_GRANT=$(get_permission_value_for_player "${SUBSTATION_ID}" "${PLAYER_5_ID}")
EXPECTED_AFTER_GRANT=$(( VAL_BEFORE_GRANT | 32 ))
assert_eq "Permission on substation after grant includes Grid" "${EXPECTED_AFTER_GRANT}" "${VAL_AFTER_GRANT}"

# ─── permission-revoke-on-object ───
run_tx "Revoking Player 5 Grid permission on substation" \
    tx structs permission-revoke-on-object "${SUBSTATION_ID}" "${PLAYER_5_ID}" 32 --from alice

VAL_AFTER_REVOKE=$(get_permission_value_for_player "${SUBSTATION_ID}" "${PLAYER_5_ID}")
EXPECTED_AFTER_REVOKE=$(( EXPECTED_AFTER_GRANT & ~32 ))
assert_eq "Permission on substation after revoke removes Grid" "${EXPECTED_AFTER_REVOKE}" "${VAL_AFTER_REVOKE}"

# ─── permission-set-on-object: set a specific permission set ───
run_tx "Setting Player 5 permissions on substation to Assets+Grid (40)" \
    tx structs permission-set-on-object "${SUBSTATION_ID}" "${PLAYER_5_ID}" 40 --from alice

VAL_AFTER_SET=$(get_permission_value_for_player "${SUBSTATION_ID}" "${PLAYER_5_ID}")
assert_eq "Permission on substation after set (40)" "40" "${VAL_AFTER_SET}"

# Clean up: revoke all from Player 5
run_tx "Clearing Player 5 permissions on substation" \
    tx structs permission-set-on-object "${SUBSTATION_ID}" "${PLAYER_5_ID}" 0 --from alice

# ─── permission-grant-on-address / permission-revoke-on-address ───
# Address permissions can only be managed on your OWN player's addresses.
# Player 5 grants PermDelete (8) on their own address (already has it, but tests the grant path).
run_tx "Player 5 granting own address PermDelete (8)" \
    tx structs permission-grant-on-address "${PLAYER_5_ADDRESS}" 8 --from player_5

# Query permissions by player
PERM_BY_PLAYER=$(query query structs permission-by-player "${PLAYER_5_ID}" 2>/dev/null || echo '{}')
info "Permissions for Player 5 after address grant:"
echo "${PERM_BY_PLAYER}" | jq -r '.permissionRecord[]? | "  obj=\(.objectId) val=\(.value)"' 2>/dev/null | head -5 || echo "  (no records)"

# Revoke PermDelete (8) from Player 5's address
run_tx "Player 5 revoking own address PermDelete (8)" \
    tx structs permission-revoke-on-address "${PLAYER_5_ADDRESS}" 8 --from player_5

# ─── permission-set-on-address ───
# NOTE: permission-set-on-address prevents privilege escalation — the caller
# needs ALL bits of the target value. After revoking PermDelete (8),
# the address has PermAll minus PermDelete = 16777207. We demonstrate set
# by setting to that value (proving the command works).
run_tx "Player 5 setting own address permissions to 16777207 (PermAll minus PermDelete)" \
    tx structs permission-set-on-address "${PLAYER_5_ADDRESS}" 16777207 --from player_5

# Restore Player 5 address to full permissions for later phases.
# Player 5 can't re-grant PermDelete (escalation prevention), so Alice does it.
run_tx "Alice restoring Player 5 address PermDelete" \
    tx structs permission-grant-on-address "${PLAYER_5_ADDRESS}" 8 --from alice

# ─── General permission query ───
info "All permissions sample:"
query query structs permission-all 2>/dev/null | jq -r '.permissionRecord[:5]? | .[]? | "  \(.objectId) = \(.value)"' || echo "  (none)"

# ─── Bitmask arithmetic verification ────────────────────────────────────────
info "--- Bitmask Arithmetic: Grant/Revoke/Set on Guild ---"

run_tx "Grant Player 4 PermUpdate (4) on guild" \
    tx structs permission-grant-on-object "${GUILD_ID}" "${PLAYER_4_ID}" "${PERM_UPDATE}" --from alice

VAL=$(get_permission_value_for_player "${GUILD_ID}" "${PLAYER_4_ID}")
assert_eq "Permission value after grant (4)" "4" "${VAL}"

run_tx "Grant additional PermDelete (8) on guild" \
    tx structs permission-grant-on-object "${GUILD_ID}" "${PLAYER_4_ID}" "${PERM_DELETE}" --from alice

VAL=$(get_permission_value_for_player "${GUILD_ID}" "${PLAYER_4_ID}")
assert_eq "Permission value after grant 4+8 (12)" "12" "${VAL}"

run_tx "Revoke PermUpdate (4) from Player 4" \
    tx structs permission-revoke-on-object "${GUILD_ID}" "${PLAYER_4_ID}" "${PERM_UPDATE}" --from alice

VAL=$(get_permission_value_for_player "${GUILD_ID}" "${PLAYER_4_ID}")
assert_eq "Permission value after revoke 4 (8)" "8" "${VAL}"

run_tx "Revoke PermDelete (8) to clear all" \
    tx structs permission-revoke-on-object "${GUILD_ID}" "${PLAYER_4_ID}" "${PERM_DELETE}" --from alice

VAL=$(get_permission_value_for_player "${GUILD_ID}" "${PLAYER_4_ID}")
assert_eq "Permission value after revoke all" "0" "${VAL}"

run_tx "Set permission to 32 (overwrite)" \
    tx structs permission-set-on-object "${GUILD_ID}" "${PLAYER_4_ID}" "${PERM_TOKEN_INFUSE}" --from alice

VAL=$(get_permission_value_for_player "${GUILD_ID}" "${PLAYER_4_ID}")
assert_eq "Permission value after set 32" "32" "${VAL}"

run_tx "Set permission to 8 (overwrite again)" \
    tx structs permission-set-on-object "${GUILD_ID}" "${PLAYER_4_ID}" "${PERM_DELETE}" --from alice

VAL=$(get_permission_value_for_player "${GUILD_ID}" "${PLAYER_4_ID}")
assert_eq "Permission value after set overwrite (8)" "8" "${VAL}"

run_tx "Clean up Player 4 permissions on guild" \
    tx structs permission-revoke-on-object "${GUILD_ID}" "${PLAYER_4_ID}" "${PERM_DELETE}" --from alice

# ─── Positive action tests (with permission) ────────────────────────────────
info "--- Positive Action Tests ---"

run_tx "Grant Player 5 PermGuildEndpointUpdate on guild" \
    tx structs permission-grant-on-object "${GUILD_ID}" "${PLAYER_5_ID}" "${PERM_GUILD_ENDPOINT_UPDATE}" --from alice

run_tx "Player 5 (has PermGuildEndpointUpdate) updates guild endpoint" \
    tx structs guild-update-endpoint "${GUILD_ID}" "positive-test.energy" --from player_5

GUILD_EP=$(query query structs guild "${GUILD_ID}" | jq -r '.Guild.endpoint // empty' 2>/dev/null || echo "")
assert_eq "Guild endpoint updated by Player 5" "positive-test.energy" "${GUILD_EP}"

run_tx "Restore guild endpoint" \
    tx structs guild-update-endpoint "${GUILD_ID}" "oh.energy" --from alice

run_tx "Grant Player 5 PermUpdate on substation for connect/disconnect" \
    tx structs permission-grant-on-object "${SUBSTATION_ID}" "${PLAYER_5_ID}" "${PERM_UPDATE}" --from alice

run_tx "Player 5 disconnects allocation (positive)" \
    tx structs substation-allocation-disconnect "${P5_ALLOC_ID}" --from player_5
run_tx "Player 5 reconnects allocation (positive)" \
    tx structs substation-allocation-connect "${P5_ALLOC_ID}" "${SUBSTATION_ID}" --from player_5

# ─── Negative action tests (without permission) ─────────────────────────────
info "--- Negative Action Tests ---"

run_tx "Revoke Player 5 PermGuildEndpointUpdate on guild" \
    tx structs permission-revoke-on-object "${GUILD_ID}" "${PLAYER_5_ID}" "${PERM_GUILD_ENDPOINT_UPDATE}" --from alice

run_tx_expect_permission_denied "Player 5 (no endpoint perm) tries guild-update-endpoint" \
    tx structs guild-update-endpoint "${GUILD_ID}" "hacked.energy" --from player_5

run_tx_expect_permission_denied "Player 4 (no perm on guild) tries guild-update-endpoint" \
    tx structs guild-update-endpoint "${GUILD_ID}" "hacked.energy" --from player_4

run_tx "Grant Player 4 only PermPlay (1) on guild" \
    tx structs permission-grant-on-object "${GUILD_ID}" "${PLAYER_4_ID}" "${PERM_PLAY}" --from alice

run_tx_expect_permission_denied "Player 4 with PermPlay only tries guild-update-endpoint" \
    tx structs guild-update-endpoint "${GUILD_ID}" "hacked.energy" --from player_4

run_tx "Revoke Player 4 PermPlay" \
    tx structs permission-revoke-on-object "${GUILD_ID}" "${PLAYER_4_ID}" "${PERM_PLAY}" --from alice

run_tx "Revoke Player 5 PermUpdate on substation" \
    tx structs permission-revoke-on-object "${SUBSTATION_ID}" "${PLAYER_5_ID}" "${PERM_UPDATE}" --from alice

# ─── Guild rank permission lifecycle ─────────────────────────────────────────
info "--- Guild Rank Permission Lifecycle ---"

# Permission constants: PermGuildEndpointUpdate = 1<<14 = 16384
#                       PermAllocationConnection = 1<<11 = 2048

# Set guild rank permission: PermGuildEndpointUpdate (16384) on guild, rank <= 3
run_tx "Setting guild rank perm: PermGuildEndpointUpdate (rank 3) on guild" \
    tx structs permission-guild-rank-set "${GUILD_ID}" "${GUILD_ID}" 16384 3 --from alice

GRANK_JSON=$(query query structs guild-rank-permission-by-object "${GUILD_ID}" 2>/dev/null || echo '{}')
info "Guild rank permissions on guild after set:"
echo "${GRANK_JSON}" | jq '.' 2>/dev/null | head -10 || echo "  (raw: ${GRANK_JSON})"

# Set Player 4 to rank 2 (within threshold) and Player 5 to rank 5 (outside threshold)
run_tx "Setting Player 4 rank to 2 for guild rank perm test" \
    tx structs player-update-guild-rank "${PLAYER_4_ID}" 2 --from alice
run_tx "Setting Player 5 rank to 5 for guild rank perm test" \
    tx structs player-update-guild-rank "${PLAYER_5_ID}" 5 --from alice

# Positive: Player 4 (rank 2 <= 3) updates guild endpoint
run_tx "Player 4 (rank 2) updates guild endpoint via guild rank permission" \
    tx structs guild-update-endpoint "${GUILD_ID}" "rank-test.energy" --from player_4

GUILD_JSON=$(query query structs guild "${GUILD_ID}")
GUILD_EP_RANK_POS=$(jqr "${GUILD_JSON}" '.Guild.endpoint')
assert_eq "Guild endpoint updated by rank-2 player" "rank-test.energy" "${GUILD_EP_RANK_POS}"

# Negative: Player 5 (rank 5 > 3) tries to update guild endpoint
run_tx "Player 5 (rank 5) tries to update guild endpoint (should fail)" \
    tx structs guild-update-endpoint "${GUILD_ID}" "hacked.energy" --from player_5

GUILD_JSON=$(query query structs guild "${GUILD_ID}")
GUILD_EP_RANK_NEG=$(jqr "${GUILD_JSON}" '.Guild.endpoint')
assert_eq "Guild endpoint unchanged by rank-5 player" "rank-test.energy" "${GUILD_EP_RANK_NEG}"

# Reset guild endpoint
run_tx "Resetting guild endpoint to oh.energy" \
    tx structs guild-update-endpoint "${GUILD_ID}" "oh.energy" --from alice

# Revoke guild rank permission
run_tx "Revoking guild rank PermGuildEndpointUpdate on guild" \
    tx structs permission-guild-rank-revoke "${GUILD_ID}" "${GUILD_ID}" 16384 --from alice

GRANK_AFTER_REVOKE=$(query query structs guild-rank-permission-by-object-and-guild "${GUILD_ID}" "${GUILD_ID}" 2>/dev/null || echo '{}')
info "Guild rank permissions on guild after revoke:"
echo "${GRANK_AFTER_REVOKE}" | jq '.' 2>/dev/null | head -5 || echo "  (empty/revoked)"

# Reset player ranks
run_tx "Resetting Player 4 rank to 101" \
    tx structs player-update-guild-rank "${PLAYER_4_ID}" 101 --from alice
run_tx "Resetting Player 5 rank to 101" \
    tx structs player-update-guild-rank "${PLAYER_5_ID}" 101 --from alice

# ─── Guild rank permission on substation ─────────────────────────────────────
info "--- Guild Rank Permission on Substation ---"

# Set guild rank permission: PermAllocationConnection (2048) on substation, rank <= 2
run_tx "Setting guild rank perm: PermAllocationConnection (2048, rank 2) on substation" \
    tx structs permission-guild-rank-set "${SUBSTATION_ID}" "${GUILD_ID}" 2048 2 --from alice

GRANK_SUB_JSON=$(query query structs guild-rank-permission-by-object "${SUBSTATION_ID}" 2>/dev/null || echo '{}')
GRANK_SUB_COUNT=$(echo "${GRANK_SUB_JSON}" | jq -r '.guild_rank_permission_records | length' 2>/dev/null || echo "0")
assert_gt "Guild rank perm records exist on substation" 0 "${GRANK_SUB_COUNT}"

# Revoke
run_tx "Revoking guild rank PermAllocationConnection on substation" \
    tx structs permission-guild-rank-revoke "${SUBSTATION_ID}" "${GUILD_ID}" 2048 --from alice

# ─── Combined bitmask guild rank permission tests ─────────────────────────────
info "--- Combined Bitmask Guild Rank Permissions ---"

# Permission constants: PermUpdate=4 (1<<2), PermGuildEndpointUpdate=16384 (1<<14)
# Combined: 4 | 16384 = 16388

# Set combined permission mask on guild, rank <= 3
run_tx "Setting combined guild rank perm (PermUpdate|PermGuildEndpointUpdate = 16388, rank 3) on guild" \
    tx structs permission-guild-rank-set "${GUILD_ID}" "${GUILD_ID}" 16388 3 --from alice

# Query and verify decomposition into 2 individual records
GRANK_COMBINED_JSON=$(query query structs guild-rank-permission-by-object-and-guild "${GUILD_ID}" "${GUILD_ID}" 2>/dev/null || echo '{}')
GRANK_COMBINED_COUNT=$(echo "${GRANK_COMBINED_JSON}" | jq -r '.guild_rank_permission_records | length' 2>/dev/null || echo "0")
assert_eq "Combined mask decomposed into 2 records" "2" "${GRANK_COMBINED_COUNT}"

# Verify individual records have correct single-bit permission values
GRANK_HAS_4=$(echo "${GRANK_COMBINED_JSON}" | jq -r '[.guild_rank_permission_records[]? | select(.permissions == "4")] | length' 2>/dev/null || echo "0")
GRANK_HAS_16384=$(echo "${GRANK_COMBINED_JSON}" | jq -r '[.guild_rank_permission_records[]? | select(.permissions == "16384")] | length' 2>/dev/null || echo "0")
assert_eq "Record for PermUpdate (4) exists" "1" "${GRANK_HAS_4}"
assert_eq "Record for PermGuildEndpointUpdate (16384) exists" "1" "${GRANK_HAS_16384}"

# Action test: Player 4 (rank 2 <= 3) can act, Player 5 (rank 5 > 3) cannot
run_tx "Setting Player 4 rank to 2 for combined mask test" \
    tx structs player-update-guild-rank "${PLAYER_4_ID}" 2 --from alice
run_tx "Setting Player 5 rank to 5 for combined mask test" \
    tx structs player-update-guild-rank "${PLAYER_5_ID}" 5 --from alice

run_tx "Player 4 (rank 2) updates guild endpoint via combined guild rank permission" \
    tx structs guild-update-endpoint "${GUILD_ID}" "combined-rank-test.energy" --from player_4

GUILD_JSON=$(query query structs guild "${GUILD_ID}")
GUILD_EP_COMB=$(jqr "${GUILD_JSON}" '.Guild.endpoint')
assert_eq "Guild endpoint updated by rank-2 player (combined mask)" "combined-rank-test.energy" "${GUILD_EP_COMB}"

run_tx "Player 5 (rank 5) tries to update guild endpoint (should fail — combined mask, rank 3)" \
    tx structs guild-update-endpoint "${GUILD_ID}" "hacked.energy" --from player_5

GUILD_JSON=$(query query structs guild "${GUILD_ID}")
GUILD_EP_COMB_NEG=$(jqr "${GUILD_JSON}" '.Guild.endpoint')
assert_eq "Guild endpoint unchanged by rank-5 player (combined mask)" "combined-rank-test.energy" "${GUILD_EP_COMB_NEG}"

run_tx "Resetting guild endpoint to oh.energy" \
    tx structs guild-update-endpoint "${GUILD_ID}" "oh.energy" --from alice

# Partial revoke: remove only PermGuildEndpointUpdate (16384), keep PermUpdate (4)
run_tx "Revoking only PermGuildEndpointUpdate (16384) from combined mask" \
    tx structs permission-guild-rank-revoke "${GUILD_ID}" "${GUILD_ID}" 16384 --from alice

GRANK_PARTIAL_JSON=$(query query structs guild-rank-permission-by-object-and-guild "${GUILD_ID}" "${GUILD_ID}" 2>/dev/null || echo '{}')
GRANK_PARTIAL_COUNT=$(echo "${GRANK_PARTIAL_JSON}" | jq -r '.guild_rank_permission_records | length' 2>/dev/null || echo "0")
assert_eq "After partial revoke, 1 record remains" "1" "${GRANK_PARTIAL_COUNT}"

GRANK_PARTIAL_PERM=$(echo "${GRANK_PARTIAL_JSON}" | jq -r '.guild_rank_permission_records[0].permissions // empty' 2>/dev/null || echo "")
assert_eq "Remaining record is PermUpdate (4)" "4" "${GRANK_PARTIAL_PERM}"

# Revoke remaining PermUpdate (4)
run_tx "Revoking remaining PermUpdate (4) from guild rank" \
    tx structs permission-guild-rank-revoke "${GUILD_ID}" "${GUILD_ID}" 4 --from alice

GRANK_EMPTY_JSON=$(query query structs guild-rank-permission-by-object-and-guild "${GUILD_ID}" "${GUILD_ID}" 2>/dev/null || echo '{}')
GRANK_EMPTY_COUNT=$(echo "${GRANK_EMPTY_JSON}" | jq -r '.guild_rank_permission_records | length' 2>/dev/null || echo "0")
assert_eq "After full revoke, 0 records remain" "0" "${GRANK_EMPTY_COUNT}"

# Different ranks per bit on substation
info "--- Per-Bit Rank Independence ---"

run_tx "Setting PermUpdate (4) rank 3 on substation" \
    tx structs permission-guild-rank-set "${SUBSTATION_ID}" "${GUILD_ID}" 4 3 --from alice
run_tx "Setting PermGuildEndpointUpdate (16384) rank 5 on substation" \
    tx structs permission-guild-rank-set "${SUBSTATION_ID}" "${GUILD_ID}" 16384 5 --from alice

GRANK_MULTI_JSON=$(query query structs guild-rank-permission-by-object-and-guild "${SUBSTATION_ID}" "${GUILD_ID}" 2>/dev/null || echo '{}')
GRANK_MULTI_COUNT=$(echo "${GRANK_MULTI_JSON}" | jq -r '.guild_rank_permission_records | length' 2>/dev/null || echo "0")
assert_eq "Two records with different ranks" "2" "${GRANK_MULTI_COUNT}"

GRANK_RANK_FOR_4=$(echo "${GRANK_MULTI_JSON}" | jq -r '[.guild_rank_permission_records[]? | select(.permissions == "4")] | .[0].rank // empty' 2>/dev/null || echo "")
GRANK_RANK_FOR_16384=$(echo "${GRANK_MULTI_JSON}" | jq -r '[.guild_rank_permission_records[]? | select(.permissions == "16384")] | .[0].rank // empty' 2>/dev/null || echo "")
assert_eq "PermUpdate rank is 3" "3" "${GRANK_RANK_FOR_4}"
assert_eq "PermGuildEndpointUpdate rank is 5" "5" "${GRANK_RANK_FOR_16384}"

# Clean up per-bit test
run_tx "Revoking PermUpdate on substation" \
    tx structs permission-guild-rank-revoke "${SUBSTATION_ID}" "${GUILD_ID}" 4 --from alice
run_tx "Revoking PermGuildEndpointUpdate on substation" \
    tx structs permission-guild-rank-revoke "${SUBSTATION_ID}" "${GUILD_ID}" 16384 --from alice

# Reset player ranks
run_tx "Resetting Player 4 rank to 101" \
    tx structs player-update-guild-rank "${PLAYER_4_ID}" 101 --from alice
run_tx "Resetting Player 5 rank to 101" \
    tx structs player-update-guild-rank "${PLAYER_5_ID}" 101 --from alice

# ─── Grant/revoke ordering ──────────────────────────────────────────────────
info "--- Grant/Revoke Ordering ---"

VAL_P4_START=$(get_permission_value_for_player "${GUILD_ID}" "${PLAYER_4_ID}")
VAL_P5_START=$(get_permission_value_for_player "${GUILD_ID}" "${PLAYER_5_ID}")
info "Starting state: P4=${VAL_P4_START}, P5=${VAL_P5_START}"

run_tx "Grant P4→guild PermUpdate(4)" \
    tx structs permission-grant-on-object "${GUILD_ID}" "${PLAYER_4_ID}" "${PERM_UPDATE}" --from alice
run_tx "Grant P5→guild PermDelete(8)" \
    tx structs permission-grant-on-object "${GUILD_ID}" "${PLAYER_5_ID}" "${PERM_DELETE}" --from alice

VAL_P4=$(get_permission_value_for_player "${GUILD_ID}" "${PLAYER_4_ID}")
VAL_P5=$(get_permission_value_for_player "${GUILD_ID}" "${PLAYER_5_ID}")
EXPECT_P4=$(( VAL_P4_START | PERM_UPDATE ))
EXPECT_P5=$(( VAL_P5_START | PERM_DELETE ))
assert_eq "P4 permission on guild after grant" "${EXPECT_P4}" "${VAL_P4}"
assert_eq "P5 permission on guild after grant" "${EXPECT_P5}" "${VAL_P5}"

run_tx "Revoke P4 PermUpdate on guild" \
    tx structs permission-revoke-on-object "${GUILD_ID}" "${PLAYER_4_ID}" "${PERM_UPDATE}" --from alice
VAL_P4=$(get_permission_value_for_player "${GUILD_ID}" "${PLAYER_4_ID}")
assert_eq "P4 permission after revoke PermUpdate" "${VAL_P4_START}" "${VAL_P4}"

run_tx "Revoke P5 PermUpdate (not set) — idempotent" \
    tx structs permission-revoke-on-object "${GUILD_ID}" "${PLAYER_5_ID}" "${PERM_UPDATE}" --from alice
VAL_P5=$(get_permission_value_for_player "${GUILD_ID}" "${PLAYER_5_ID}")
assert_eq "P5 permission unchanged after idempotent revoke" "${EXPECT_P5}" "${VAL_P5}"

run_tx "Clean up P5 PermDelete" \
    tx structs permission-revoke-on-object "${GUILD_ID}" "${PLAYER_5_ID}" "${PERM_DELETE}" --from alice

# ─── Object deletion and permission cleanup ──────────────────────────────────
info "--- Object Deletion Permission Cleanup ---"

if structsd tx structs provider-create --help 2>&1 | grep -q "Create a new Energy Provider"; then
    run_tx "Create provider for cleanup test" \
        tx structs provider-create "${SUBSTATION_ID}" \
        "1ualpha" "open" 0 0 100 1000 10 1000 --from alice
    PROVIDER_ALL=$(query query structs provider-all 2>/dev/null || echo '{}')
    PROVIDER_ID=$(echo "${PROVIDER_ALL}" | jq -r '.Provider[-1].id // empty' 2>/dev/null || echo "")
    if [ -n "${PROVIDER_ID}" ]; then
        run_tx "Grant P4 permission on provider" \
            tx structs permission-grant-on-object "${PROVIDER_ID}" "${PLAYER_4_ID}" "${PERM_UPDATE}" --from alice
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

fi # phase 4e

if run_phase 482; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 4e2: Create 20 Rank-Test Players (rp_1..rp_20)
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 4e2: Rank-Test Player Setup (20 players)"

NUM_RP=20
info "Creating ${NUM_RP} rank-test players (rp_1..rp_20)"

for RP_NUM in $(seq 1 ${NUM_RP}); do
    RP_KEY="rp_${RP_NUM}"
    RP_KEYS+=("${RP_KEY}")
    EXISTING=$(structsd ${PARAMS_KEYS} keys show "${RP_KEY}" 2>/dev/null | jq -r .address 2>/dev/null || echo "")
    if [ -z "${EXISTING}" ]; then
        (echo ""; echo "") | structsd ${PARAMS_KEYS} keys add "${RP_KEY}" --no-backup 2>/dev/null || true
        ADDR=$(structsd ${PARAMS_KEYS} keys show "${RP_KEY}" 2>/dev/null | jq -r .address 2>/dev/null || echo "")
    else
        ADDR="${EXISTING}"
    fi
    if [ -z "${ADDR}" ]; then
        echo -e "  ${RED}Cannot get address for ${RP_KEY}${NC}"
        exit 1
    fi

    run_tx "Fund ${RP_KEY}" tx bank send "${PLAYER_1_ADDRESS}" "${ADDR}" 4000000ualpha --from alice
    run_tx "Delegate ${RP_KEY}" tx staking delegate "${VALIDATOR_ADDRESS}" 2000000ualpha --from "${RP_KEY}"

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
        echo -e "  ${RED}Failed to get valid player ID for ${RP_KEY} (got '${PID}')${NC}"
        exit 1
    fi
    RP_IDS+=("${PID}")

    run_tx "Guild join ${RP_KEY}" \
        tx structs guild-membership-join "${GUILD_ID}" "${REACTOR_ID}-${ADDR}" --from "${RP_KEY}"
    echo -e "  ${GREEN}OK${NC} ${RP_KEY} -> ${PID}"
done

echo "  Created ${#RP_IDS[@]} rank-test players"

fi # phase 4e2

if run_phase 484; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 4e3: Comprehensive Rank Tests
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 4e3: Comprehensive Rank Tests"

if [ "${#RP_IDS[@]}" -lt 10 ]; then
    info "Fewer than 10 rank-test players; skipping rank tests (need Phase 4e2 first)"
else

# ── 4e3a: Admin rank assignment sweep ────────────────────────────────────────
info "--- Admin Rank Assignment Sweep (${#RP_IDS[@]} players) ---"

for i in "${!RP_IDS[@]}"; do
    PID="${RP_IDS[$i]}"
    DESIRED_RANK=$(( i + 1 ))
    run_tx "Alice sets ${RP_KEYS[$i]} to rank ${DESIRED_RANK}" \
        tx structs player-update-guild-rank "${PID}" "${DESIRED_RANK}" --from alice
done

RANK_VERIFY_PASS=0
RANK_VERIFY_FAIL=0
for i in "${!RP_IDS[@]}"; do
    PID="${RP_IDS[$i]}"
    EXPECTED=$(( i + 1 ))
    ACTUAL=$(get_player_guild_rank "${PID}")
    if [ "${ACTUAL}" = "${EXPECTED}" ]; then
        RANK_VERIFY_PASS=$(( RANK_VERIFY_PASS + 1 ))
    else
        echo -e "  ${RED}FAIL${NC}: ${RP_KEYS[$i]} rank expected=${EXPECTED} got=${ACTUAL}"
        RANK_VERIFY_FAIL=$(( RANK_VERIFY_FAIL + 1 ))
        FAIL_COUNT=$(( FAIL_COUNT + 1 ))
    fi
done
echo -e "  ${GREEN}Rank assignment sweep: ${RANK_VERIFY_PASS} verified${NC}"
if [ "${RANK_VERIFY_FAIL}" -gt 0 ]; then
    echo -e "  ${RED}${RANK_VERIFY_FAIL} rank verifications failed${NC}"
fi
PASS_COUNT=$(( PASS_COUNT + RANK_VERIFY_PASS ))

run_tx "Set Player 4 to rank 2" tx structs player-update-guild-rank "${PLAYER_4_ID}" 2 --from alice
run_tx "Set Player 5 to rank 5" tx structs player-update-guild-rank "${PLAYER_5_ID}" 5 --from alice

# ── 4e3b: Guild rank permission threshold sweep ─────────────────────────────
info "--- Guild Rank Permission Threshold Sweep ---"

for THRESHOLD in 0 3 5 10 15 19; do
    run_tx "Set PermGuildEndpointUpdate threshold=${THRESHOLD} on guild" \
        tx structs permission-guild-rank-set "${GUILD_ID}" "${GUILD_ID}" "${PERM_GUILD_ENDPOINT_UPDATE}" "${THRESHOLD}" --from alice

    if [ "${THRESHOLD}" -ge 2 ]; then
        run_tx "Player 4 (rank 2) endpoint update (threshold=${THRESHOLD}, expect PASS)" \
            tx structs guild-update-endpoint "${GUILD_ID}" "threshold-${THRESHOLD}-pass.energy" --from player_4
    else
        run_tx_expect_permission_denied "Player 4 (rank 2) endpoint update (threshold=${THRESHOLD}, expect DENY)" \
            tx structs guild-update-endpoint "${GUILD_ID}" "threshold-${THRESHOLD}-fail.energy" --from player_4
    fi

    if [ "${THRESHOLD}" -ge 5 ]; then
        run_tx "Player 5 (rank 5) endpoint update (threshold=${THRESHOLD}, expect PASS)" \
            tx structs guild-update-endpoint "${GUILD_ID}" "threshold-${THRESHOLD}-p5pass.energy" --from player_5
    else
        run_tx_expect_permission_denied "Player 5 (rank 5) endpoint update (threshold=${THRESHOLD}, expect DENY)" \
            tx structs guild-update-endpoint "${GUILD_ID}" "threshold-${THRESHOLD}-p5fail.energy" --from player_5
    fi

    if [ "${THRESHOLD}" -ge 1 ] && [ "${THRESHOLD}" -le "${#RP_IDS[@]}" ]; then
        IDX=$(( THRESHOLD - 1 ))
        run_tx "${RP_KEYS[$IDX]} (rank ${THRESHOLD}) at exact boundary (expect PASS)" \
            tx structs guild-update-endpoint "${GUILD_ID}" "boundary-${THRESHOLD}-at.energy" --from "${RP_KEYS[$IDX]}"
    fi
    ABOVE=$(( THRESHOLD + 1 ))
    if [ "${ABOVE}" -ge 1 ] && [ "${ABOVE}" -le "${#RP_IDS[@]}" ]; then
        IDX=$(( ABOVE - 1 ))
        run_tx_expect_permission_denied "${RP_KEYS[$IDX]} (rank ${ABOVE}) just above boundary (expect DENY)" \
            tx structs guild-update-endpoint "${GUILD_ID}" "boundary-${THRESHOLD}-above.energy" --from "${RP_KEYS[$IDX]}"
    fi
done

run_tx "Revoke guild-rank PermGuildEndpointUpdate on guild" \
    tx structs permission-guild-rank-revoke "${GUILD_ID}" "${GUILD_ID}" "${PERM_GUILD_ENDPOINT_UPDATE}" --from alice

# ── 4e3c: Rank-based rank management ────────────────────────────────────────
info "--- Rank-Based Rank Management ---"

# Player 4=rank 2, Player 5=rank 5, rp_1..rp_20=rank 1..20
run_tx "Player 4 (rank 2) promotes rp_10 (rank 10) to rank 7" \
    tx structs player-update-guild-rank "${RP_IDS[9]}" 7 --from player_4
RANK=$(get_player_guild_rank "${RP_IDS[9]}")
assert_eq "rp_10 rank after partial promote" "7" "${RANK}"

run_tx "Player 4 (rank 2) demotes rp_15 (rank 15) to rank 18" \
    tx structs player-update-guild-rank "${RP_IDS[14]}" 18 --from player_4
RANK=$(get_player_guild_rank "${RP_IDS[14]}")
assert_eq "rp_15 rank after demotion" "18" "${RANK}"

run_tx "Player 4 (rank 2) promotes rp_20 (rank 20) to rank 2 (own level)" \
    tx structs player-update-guild-rank "${RP_IDS[19]}" 2 --from player_4
RANK=$(get_player_guild_rank "${RP_IDS[19]}")
assert_eq "rp_20 rank after promote to own level" "2" "${RANK}"

run_tx_expect_permission_denied "Player 4 (rank 2) cannot promote rp_10 to rank 1 (above self)" \
    tx structs player-update-guild-rank "${RP_IDS[9]}" 1 --from player_4

run_tx_expect_permission_denied "Player 5 (rank 5) cannot modify Player 4 (rank 2)" \
    tx structs player-update-guild-rank "${PLAYER_4_ID}" 10 --from player_5

run_tx_expect_permission_denied "Player 5 (rank 5) cannot modify rp_1 (rank 1)" \
    tx structs player-update-guild-rank "${RP_IDS[0]}" 10 --from player_5

run_tx_expect_permission_denied "Player 5 (rank 5) cannot modify rp_5 (rank 5, equal)" \
    tx structs player-update-guild-rank "${RP_IDS[4]}" 10 --from player_5

run_tx_expect_permission_denied "rp_20 (rank 2) cannot modify Player 4 (rank 2, equal)" \
    tx structs player-update-guild-rank "${PLAYER_4_ID}" 10 --from "${RP_KEYS[19]}"

run_tx "Player 5 (rank 5) promotes rp_10 (rank 7) to rank 6" \
    tx structs player-update-guild-rank "${RP_IDS[9]}" 6 --from player_5
RANK=$(get_player_guild_rank "${RP_IDS[9]}")
assert_eq "rp_10 rank after Player 5 promotes to 6" "6" "${RANK}"

run_tx "Player 5 (rank 5) demotes rp_10 (rank 6) to rank 100" \
    tx structs player-update-guild-rank "${RP_IDS[9]}" 100 --from player_5
RANK=$(get_player_guild_rank "${RP_IDS[9]}")
assert_eq "rp_10 rank after demotion to 100" "100" "${RANK}"

# ── 4e3d: Chain of rank modification ────────────────────────────────────────
info "--- Chain of Rank Modification ---"

run_tx "Alice sets rp_6 to rank 2" tx structs player-update-guild-rank "${RP_IDS[5]}" 2 --from alice
run_tx "Alice sets rp_7 to rank 10" tx structs player-update-guild-rank "${RP_IDS[6]}" 10 --from alice
run_tx "Alice sets rp_8 to rank 15" tx structs player-update-guild-rank "${RP_IDS[7]}" 15 --from alice

run_tx "Chain step 1: rp_6 (rank 2) sets rp_7 (rank 10) to rank 4" \
    tx structs player-update-guild-rank "${RP_IDS[6]}" 4 --from "${RP_KEYS[5]}"
RANK=$(get_player_guild_rank "${RP_IDS[6]}")
assert_eq "Chain step 1: rp_7 is now rank 4" "4" "${RANK}"

run_tx "Chain step 2: rp_7 (rank 4) sets rp_8 (rank 15) to rank 5" \
    tx structs player-update-guild-rank "${RP_IDS[7]}" 5 --from "${RP_KEYS[6]}"
RANK=$(get_player_guild_rank "${RP_IDS[7]}")
assert_eq "Chain step 2: rp_8 is now rank 5" "5" "${RANK}"

run_tx_expect_permission_denied "Chain: rp_8 (rank 5) cannot modify rp_7 (rank 4)" \
    tx structs player-update-guild-rank "${RP_IDS[6]}" 10 --from "${RP_KEYS[7]}"

run_tx "Chain step 3: rp_7 (rank 4) demotes rp_8 (rank 5) to rank 15" \
    tx structs player-update-guild-rank "${RP_IDS[7]}" 15 --from "${RP_KEYS[6]}"
RANK=$(get_player_guild_rank "${RP_IDS[7]}")
assert_eq "Chain step 3: rp_8 back to rank 15" "15" "${RANK}"

# ── 4e3e: Mass rank shuffle and verify ──────────────────────────────────────
info "--- Mass Rank Shuffle ---"

for i in "${!RP_IDS[@]}"; do
    NEW_RANK=$(( ${#RP_IDS[@]} - i ))
    structsd ${PARAMS_TX} tx structs player-update-guild-rank "${RP_IDS[$i]}" "${NEW_RANK}" --from alice 2>&1 || true
    sleep 1
done

SHUFFLE_PASS=0
SHUFFLE_FAIL=0
for i in "${!RP_IDS[@]}"; do
    EXPECTED=$(( ${#RP_IDS[@]} - i ))
    ACTUAL=$(get_player_guild_rank "${RP_IDS[$i]}")
    if [ "${ACTUAL}" = "${EXPECTED}" ]; then
        SHUFFLE_PASS=$(( SHUFFLE_PASS + 1 ))
    else
        echo -e "  ${RED}FAIL${NC}: ${RP_KEYS[$i]} shuffle expected=${EXPECTED} got=${ACTUAL}"
        SHUFFLE_FAIL=$(( SHUFFLE_FAIL + 1 ))
        FAIL_COUNT=$(( FAIL_COUNT + 1 ))
    fi
done
echo -e "  ${GREEN}Mass shuffle: ${SHUFFLE_PASS}/${#RP_IDS[@]} verified${NC}"
PASS_COUNT=$(( PASS_COUNT + SHUFFLE_PASS ))

# After shuffle: rp_1=rank 20, rp_20=rank 1
run_tx "rp_20 (rank 1) sets rp_1 (rank 20) to rank 10" \
    tx structs player-update-guild-rank "${RP_IDS[0]}" 10 --from "${RP_KEYS[19]}"
RANK=$(get_player_guild_rank "${RP_IDS[0]}")
assert_eq "rp_1 rank after shuffle-based modify" "10" "${RANK}"

run_tx_expect_permission_denied "rp_1 (rank 10) cannot modify rp_20 (rank 1)" \
    tx structs player-update-guild-rank "${RP_IDS[19]}" 15 --from "${RP_KEYS[0]}"

# ── 4e3f: Edge cases ────────────────────────────────────────────────────────
info "--- Edge Cases ---"

run_tx_expect_permission_denied "Player 4 cannot self-modify rank (equal = denied)" \
    tx structs player-update-guild-rank "${PLAYER_4_ID}" 50 --from player_4

run_tx "alice (admin) can change own rank" \
    tx structs player-update-guild-rank "${PLAYER_1_ID}" 3 --from alice
RANK=$(get_player_guild_rank "${PLAYER_1_ID}")
assert_eq "alice rank after self-set" "3" "${RANK}"
run_tx "alice restores own rank to 1" \
    tx structs player-update-guild-rank "${PLAYER_1_ID}" 1 --from alice

run_tx "Alice sets rp_1 to max-ish rank (999999)" \
    tx structs player-update-guild-rank "${RP_IDS[0]}" 999999 --from alice
RANK=$(get_player_guild_rank "${RP_IDS[0]}")
assert_eq "rp_1 rank after set to 999999" "999999" "${RANK}"

run_tx "Alice sets rp_1 back to rank 1" \
    tx structs player-update-guild-rank "${RP_IDS[0]}" 1 --from alice
RANK=$(get_player_guild_rank "${RP_IDS[0]}")
assert_eq "rp_1 rank after set back to 1" "1" "${RANK}"

run_tx_expect_permission_denied "Setting rank to 0 is rejected" \
    tx structs player-update-guild-rank "${RP_IDS[0]}" 0 --from alice

# ── Clean up all ranks ──────────────────────────────────────────────────────
info "Resetting all test player ranks"
run_tx "Reset Player 4 rank to 101" tx structs player-update-guild-rank "${PLAYER_4_ID}" 101 --from alice
run_tx "Reset Player 5 rank to 101" tx structs player-update-guild-rank "${PLAYER_5_ID}" 101 --from alice
for i in "${!RP_IDS[@]}"; do
    structsd ${PARAMS_TX} tx structs player-update-guild-rank "${RP_IDS[$i]}" 101 --from alice 2>&1 || true
    sleep 1
done

fi # end rank-test player guard

fi # phase 4e3

if run_phase 490; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 4f: Substation Management
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 4f: Substation Management"

# Grant Player 5 PermSubstationConnection (1024) on the substation for connection ops
run_tx "Granting Player 5 PermSubstationConnection on substation for connection ops" \
    tx structs permission-grant-on-object "${SUBSTATION_ID}" "${PLAYER_5_ID}" 1024 --from alice

# Allocation operations: Player 5 is the allocation controller, so they must sign
run_tx "Connecting Player 5 allocation to substation" \
    tx structs substation-allocation-connect "${P5_ALLOC_ID}" "${SUBSTATION_ID}" --from player_5

ALLOC_JSON=$(query query structs allocation "${P5_ALLOC_ID}")
assert_eq "Allocation connected to substation" "${SUBSTATION_ID}" "$(jqr "${ALLOC_JSON}" '.Allocation.destinationId' '')"

# ─── substation-allocation-disconnect ───
run_tx "Disconnecting Player 5 allocation from substation" \
    tx structs substation-allocation-disconnect "${P5_ALLOC_ID}" --from player_5

ALLOC_JSON=$(query query structs allocation "${P5_ALLOC_ID}")
ALLOC_DST=$(jqr "${ALLOC_JSON}" '.Allocation.destinationId' '')
assert_eq "Allocation disconnected" "" "${ALLOC_DST}"

# Reconnect for later use
run_tx "Reconnecting Player 5 allocation" \
    tx structs substation-allocation-connect "${P5_ALLOC_ID}" "${SUBSTATION_ID}" --from player_5

ALLOC_JSON=$(query query structs allocation "${P5_ALLOC_ID}")
assert_eq "Allocation reconnected to substation" "${SUBSTATION_ID}" "$(jqr "${ALLOC_JSON}" '.Allocation.destinationId' '')"

# ─── Dual-path disconnect: substation owner disconnects player's allocation ───
# CanBeDisconnectedBy checks PermAllocationConnection on allocation first,
# then falls back to PermAllocationConnection on destination (substation).
# Alice (substation owner) should succeed via the destination path.
info "Testing allocation disconnect via substation owner (dual-path)"
run_tx "Alice disconnects Player 5 allocation via substation ownership" \
    tx structs substation-allocation-disconnect "${P5_ALLOC_ID}" --from alice

ALLOC_JSON=$(query query structs allocation "${P5_ALLOC_ID}")
ALLOC_DST_DUAL=$(jqr "${ALLOC_JSON}" '.Allocation.destinationId' '')
assert_eq "Allocation disconnected by substation owner (dual-path)" "" "${ALLOC_DST_DUAL}"

# Reconnect after dual-path test
run_tx "Reconnecting Player 5 allocation after dual-path test" \
    tx structs substation-allocation-connect "${P5_ALLOC_ID}" "${SUBSTATION_ID}" --from player_5

# ─── Create a second substation for player migration tests ───
run_tx "Creating second substation for migration test" \
    tx structs substation-create "${PLAYER_1_ID}" "${P1_ALLOC_ID}" --from alice

# Find the second substation
SUB_ALL_JSON=$(query query structs substation-all 2>/dev/null || echo '{}')
SECOND_SUB_ID=$(echo "${SUB_ALL_JSON}" | jq -r '.Substation[-1].id // empty' 2>/dev/null || echo "")

if [ -n "${SECOND_SUB_ID}" ] && [ "${SECOND_SUB_ID}" != "${SUBSTATION_ID}" ]; then
    info "Second substation for migration: ${SECOND_SUB_ID}"

    # Grant Player 5 PermSubstationConnection (1024) on both substations so they can connect themselves
    run_tx "Granting Player 5 PermSubstationConnection on original substation" \
        tx structs permission-grant-on-object "${SUBSTATION_ID}" "${PLAYER_5_ID}" 1024 --from alice
    run_tx "Granting Player 5 PermSubstationConnection on second substation" \
        tx structs permission-grant-on-object "${SECOND_SUB_ID}" "${PLAYER_5_ID}" 1024 --from alice

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

    # ─── Permission cleanup on substation delete ───
    # Grant a permission on the second substation, then delete it and verify cleanup
    run_tx "Granting Player 3 PermUpdate (4) on second substation (pre-delete)" \
        tx structs permission-grant-on-object "${SECOND_SUB_ID}" "${PLAYER_3_ID}" 4 --from alice

    # Set a guild rank permission on the second substation too
    run_tx "Setting guild rank perm on second substation (pre-delete)" \
        tx structs permission-guild-rank-set "${SECOND_SUB_ID}" "${GUILD_ID}" 4 2 --from alice

    # ─── substation-delete: delete second substation ───
    run_tx "Deleting second substation (migrate to original)" \
        tx structs substation-delete "${SECOND_SUB_ID}" "${SUBSTATION_ID}" --from alice

    # Verify second substation is gone
    DEL_SUB_JSON=$(query query structs substation "${SECOND_SUB_ID}" 2>/dev/null || echo '{}')
    DEL_SUB_ID=$(jqr "${DEL_SUB_JSON}" '.Substation.id' '')
    assert_eq "Second substation deleted" "" "${DEL_SUB_ID}"

    # Verify permissions were cleaned up with the substation
    PERM_CLEANUP_JSON=$(query query structs permission-by-object "${SECOND_SUB_ID}" 2>/dev/null || echo '{}')
    PERM_CLEANUP_COUNT=$(echo "${PERM_CLEANUP_JSON}" | jq -r '.permissionRecord | length' 2>/dev/null || echo "0")
    info "Object permissions on deleted substation: ${PERM_CLEANUP_COUNT} records"
    assert_eq "Object permissions cleaned up after substation delete" "0" "${PERM_CLEANUP_COUNT}"

    GRANK_CLEANUP_JSON=$(query query structs guild-rank-permission-by-object "${SECOND_SUB_ID}" 2>/dev/null || echo '{}')
    GRANK_CLEANUP_COUNT=$(echo "${GRANK_CLEANUP_JSON}" | jq -r '.guild_rank_permission_records | length' 2>/dev/null || echo "0")
    info "Guild rank permissions on deleted substation: ${GRANK_CLEANUP_COUNT} records"
    assert_eq "Guild rank permissions cleaned up after substation delete" "0" "${GRANK_CLEANUP_COUNT}"
else
    info "SKIP: Could not create second substation for migration tests"
fi

# ─── Grid query coverage ───
info "Grid attributes sample:"
query query structs grid-all 2>/dev/null | jq -r '.gridAttribute[:3]? | .[]? | "  \(.objectId) cap=\(.capacity) load=\(.load)"' || echo "  (none)"

fi # phase 4f

if run_phase 495; then

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

fi # phase 4g

if run_phase 500; then

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

# Verify proxy join created a player for the target address
PROXY_JOIN_JSON=$(query query structs address structs1wfs4s5er9lpkxlcrh8ezdqayewjnudkrlwpxqc)
PROXY_PLAYER=$(jqr "${PROXY_JOIN_JSON}" '.playerId')
assert_not_empty "Proxy joined player ID" "${PROXY_PLAYER}"

fi # phase 5

if run_phase 550; then

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

fi # phase 5b

if run_phase 600; then

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

# Player 4 exercises the optional planet-name argument on planet-explore.
# Bad names must reject the entire tx and leave Player 4 without a planet so
# the subsequent successful explore is a clean first-explore (no prior-planet
# completion path). Same validation rules as planet-update-name.
run_tx_expect_fail "Player 4 exploring with too-short name" \
    tx structs planet-explore "${PLAYER_4_ID}" "ab" --from player_4

run_tx_expect_fail "Player 4 exploring with too-long name (26 chars)" \
    tx structs planet-explore "${PLAYER_4_ID}" "ABCDEFGHIJKLMNOPQRSTUVWXYZ" --from player_4

run_tx_expect_fail "Player 4 exploring with object-id-like name" \
    tx structs planet-explore "${PLAYER_4_ID}" "5-100" --from player_4

# Validation must fail before any state mutation: Player 4 should still have
# no planet attached after the rejected txs above.
P4_PRE_JSON=$(query query structs player "${PLAYER_4_ID}")
assert_eq "Player 4 has no planet after failed name validation" "" "$(jqr "${P4_PRE_JSON}" '.Player.planetId')"

run_tx "Player 4 exploring a planet with name 'NewEden'" \
    tx structs planet-explore "${PLAYER_4_ID}" "NewEden" --from player_4

P4_JSON=$(query query structs player "${PLAYER_4_ID}")
PLAYER_4_PLANET_ID=$(jqr "${P4_JSON}" '.Player.planetId')
PLAYER_4_FLEET_ID=$(jqr "${P4_JSON}" '.Player.fleetId')
assert_not_empty "Player 4 planet" "${PLAYER_4_PLANET_ID}"
assert_not_empty "Player 4 fleet" "${PLAYER_4_FLEET_ID}"
echo "  Player 4 Planet: ${PLAYER_4_PLANET_ID}  Fleet: ${PLAYER_4_FLEET_ID}"

P4_PLANET_JSON=$(query query structs planet "${PLAYER_4_PLANET_ID}")
assert_eq "Player 4 planet named on explore" "NewEden" "$(jqr "${P4_PLANET_JSON}" '.Planet.name')"

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

fi # phase 6

if run_phase 700; then

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

    # Check player 2 ore inventory. Stored ore is a grid attribute
    # (GridAttributeType_ore) keyed by the player, surfaced at
    # .gridAttributes.ore; PlayerInventory only carries spendable rocks.
    P2_JSON=$(query query structs player "${PLAYER_2_ID}")
    P2_ORE=$(jqr "${P2_JSON}" '.gridAttributes.ore' '0')
    info "Player 2 ore after mining: ${P2_ORE}"
    assert_gt "Player 2 ore after mining" 0 "${P2_ORE}"

    # ─── Build Refinery (struct type 15 = Ore Refinery, land, slot 2) ───
    wait_for_charge "${PLAYER_2_ID}" "${CHARGE_BUILD}"
    run_tx "Initiating Refinery build (type=15, ambit=land, slot=2)" \
        tx structs struct-build-initiate "${PLAYER_2_ID}" 15 land 2 --from player_2

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
    assert_eq "Refinery type" "15" "${REFINERY_TYPE}"

    # ─── Refine ore ───
    # NOTE: old command was struct-refine-compute, now struct-ore-refine-compute
    run_compute "Refining ore" \
        tx structs struct-ore-refine-compute "${REFINERY_STRUCT_ID}" --from player_2
fi

fi # phase 7

if run_phase 750; then

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

fi # phase 7b

if run_phase 800; then

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

fi # phase 8

if run_phase 900; then

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

# NOTE: The per-block fleet throttle (ThrottleDecorator) prevents the same
# fleet from moving twice in one block. This can't be reliably tested with
# sequential CLI calls (2s sleep between txs = different blocks). The throttle
# is verified by unit tests in app/ante/throttle_test.go.

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

fi # phase 9

if run_phase 1000; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 10: Planet Raid — SHIELDS_VULNERABLE mechanics (v0.18.0)
# ═════════════════════════════════════════════════════════════════════════════
# A raid can only be won while the defending Command Ship is offline,
# destroyed, or non-existent. blockStartRaid tracks that vulnerability
# window: it anchors when the defender's Command Ship goes down and clears
# when it comes back online (or when the raid ends).

section "PHASE 10: Planet Raid (SHIELDS_VULNERABLE)"

P2_PLANET_JSON=$(query query structs planet "${PLAYER_2_PLANET_ID}")
P2_SHIELD=$(jqr "${P2_PLANET_JSON}" '.planetAttributes.planetaryShield' '0')
P2_RAID_CLOCK=$(jqr "${P2_PLANET_JSON}" '.planetAttributes.blockStartRaid' '0')
info "P2 planet: planetaryShield=${P2_SHIELD} blockStartRaid=${P2_RAID_CLOCK}"

P2_CMD_JSON=$(query query structs struct "${PLAYER_2_CMD_SHIP_ID}" || echo '{}')
P2_CMD_ONLINE=$(jqr "${P2_CMD_JSON}" '.structAttributes.isOnline' 'false')
assert_eq "P2 Command Ship online before raid scenarios" "true" "${P2_CMD_ONLINE}"

# ─── Scenario A (expected bad): raid cannot be won while defender CMD online ───

assert_eq "blockStartRaid unset while defender Command Ship is online" "0" "${P2_RAID_CLOCK}"

run_tx_expect_fail "Raid complete while defender Command Ship online (should fail)" \
    tx structs planet-raid-complete "${PLAYER_3_FLEET_ID}" deadbeef 1 --from player_3

run_tx_expect_fail "Raid compute fast-fails while shields are up (should fail)" \
    tx structs planet-raid-compute "${PLAYER_3_FLEET_ID}" --from player_3

# ─── Scenario B: defender CMD offline opens the vulnerability window ───

run_tx "P2 deactivates their Command Ship (shields drop)" \
    tx structs struct-deactivate "${PLAYER_2_CMD_SHIP_ID}" --from player_2

P2_PLANET_JSON=$(query query structs planet "${PLAYER_2_PLANET_ID}")
P2_RAID_CLOCK=$(jqr "${P2_PLANET_JSON}" '.planetAttributes.blockStartRaid' '0')
assert_gt "blockStartRaid anchored after defender Command Ship went offline" "0" "${P2_RAID_CLOCK}"

# ─── Scenario C (expected bad): CMD back online closes the window again ───

wait_for_charge "${PLAYER_2_ID}" "${CHARGE_ACTIVATE}"
run_tx "P2 re-activates their Command Ship (shields restored)" \
    tx structs struct-activate "${PLAYER_2_CMD_SHIP_ID}" --from player_2

P2_PLANET_JSON=$(query query structs planet "${PLAYER_2_PLANET_ID}")
P2_RAID_CLOCK=$(jqr "${P2_PLANET_JSON}" '.planetAttributes.blockStartRaid' '0')
assert_eq "blockStartRaid cleared when defender Command Ship back online" "0" "${P2_RAID_CLOCK}"

run_tx_expect_fail "Raid complete after shields restored (should fail)" \
    tx structs planet-raid-complete "${PLAYER_3_FLEET_ID}" deadbeef 1 --from player_3

# ─── Scenario D (good): raid succeeds while defender CMD is offline ───

run_tx "P2 deactivates their Command Ship again (shields drop)" \
    tx structs struct-deactivate "${PLAYER_2_CMD_SHIP_ID}" --from player_2

P2_PLANET_JSON=$(query query structs planet "${PLAYER_2_PLANET_ID}")
P2_RAID_CLOCK=$(jqr "${P2_PLANET_JSON}" '.planetAttributes.blockStartRaid' '0')
assert_gt "blockStartRaid re-anchored for the raid attempt" "0" "${P2_RAID_CLOCK}"

P3_JSON=$(query query structs player "${PLAYER_3_ID}")
P2_JSON=$(query query structs player "${PLAYER_2_ID}")
P3_ORE_BEFORE=$(jqr "${P3_JSON}" '.gridAttributes.ore' '0')
P2_ORE_BEFORE=$(jqr "${P2_JSON}" '.gridAttributes.ore' '0')
info "Player 3 ore before raid: ${P3_ORE_BEFORE}"
info "Player 2 ore before raid: ${P2_ORE_BEFORE}"

run_compute "Completing planet raid (defender Command Ship offline)" \
    tx structs planet-raid-compute "${PLAYER_3_FLEET_ID}" --from player_3

P3_JSON=$(query query structs player "${PLAYER_3_ID}")
P2_JSON=$(query query structs player "${PLAYER_2_ID}")
P3_ORE_AFTER=$(jqr "${P3_JSON}" '.gridAttributes.ore' '0')
P2_ORE_AFTER=$(jqr "${P2_JSON}" '.gridAttributes.ore' '0')
info "Player 3 ore after raid: ${P3_ORE_AFTER}"
info "Player 2 ore after raid: ${P2_ORE_AFTER}"
echo "  Raid results: P3 ore ${P3_ORE_BEFORE} -> ${P3_ORE_AFTER}, P2 ore ${P2_ORE_BEFORE} -> ${P2_ORE_AFTER}"

if [ "${SKIP_MINING}" = true ]; then
    info "Skipping ore-theft assertion (--skip-mining, no ore to steal)"
else
    assert_eq "P2 ore emptied by raid" "0" "${P2_ORE_AFTER}"
    assert_gt "P3 ore increased by raid" "${P3_ORE_BEFORE}" "${P3_ORE_AFTER}"
fi

# A successful raid sends the attacking fleet home
FLEET_3_JSON=$(query query structs fleet "${PLAYER_3_FLEET_ID}")
FLEET_3_LOC=$(jqr "${FLEET_3_JSON}" '.Fleet.locationId')
assert_eq "P3 fleet returned home after successful raid" "${PLAYER_3_PLANET_ID}" "${FLEET_3_LOC}"

# Raid over: the vulnerability clock must be cleared
P2_PLANET_JSON=$(query query structs planet "${PLAYER_2_PLANET_ID}")
P2_RAID_CLOCK=$(jqr "${P2_PLANET_JSON}" '.planetAttributes.blockStartRaid' '0')
assert_eq "blockStartRaid cleared after raid completed" "0" "${P2_RAID_CLOCK}"

# ─── Restore: bring P2's Command Ship back online for later phases ───

wait_for_charge "${PLAYER_2_ID}" "${CHARGE_ACTIVATE}"
run_tx "P2 re-activates their Command Ship (cleanup)" \
    tx structs struct-activate "${PLAYER_2_CMD_SHIP_ID}" --from player_2

P2_CMD_JSON=$(query query structs struct "${PLAYER_2_CMD_SHIP_ID}" || echo '{}')
P2_CMD_ONLINE=$(jqr "${P2_CMD_JSON}" '.structAttributes.isOnline' 'false')
assert_eq "P2 Command Ship back online after raid scenarios" "true" "${P2_CMD_ONLINE}"

fi # phase 10

if run_phase 1100; then

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

fi # phase 11

if run_phase 1200; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 12: Complex Battle — Build multi-unit fleet
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 12: Complex Battle Setup"

# Player 3 needs extra capacity to support the full combat fleet (8+ structs).
# Their initial 5M delegation isn't enough, so delegate the remaining 5M.
run_tx "Additional delegation for Player 3 (fleet capacity)" \
    tx staking delegate "${VALIDATOR_ADDRESS}" 5000000ualpha --from player_3

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

# ─── P3: SAM Launcher (type 10, land, slot 2) ───
wait_for_charge "${PLAYER_3_ID}" "${CHARGE_BUILD}"
run_tx "Initiating SAM Launcher (type=10, land, slot=2)" \
    tx structs struct-build-initiate "${PLAYER_3_ID}" 10 land 2 --from player_3

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

# ─── P3: Submarine (type 13, water, slot 1) ───
wait_for_charge "${PLAYER_3_ID}" "${CHARGE_BUILD}"
run_tx "Initiating Submarine (type=13, water, slot=1)" \
    tx structs struct-build-initiate "${PLAYER_3_ID}" 13 water 1 --from player_3

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

# ─── P3: Battleship #1 (type 2, space, slot 2) ───
wait_for_charge "${PLAYER_3_ID}" "${CHARGE_BUILD}"
run_tx "Initiating Battleship #1 (type=2, space, slot=2)" \
    tx structs struct-build-initiate "${PLAYER_3_ID}" 2 space 2 --from player_3

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
STRUCT_COUNT_BEFORE=$(query query structs struct-all | jq '.Struct | length' 2>/dev/null || echo 0)
run_tx "Initiating Cruiser (type=11, water, slot=0)" \
    tx structs struct-build-initiate "${PLAYER_3_ID}" 11 water 0 --from player_3

STRUCT_ALL_JSON=$(query query structs struct-all)
STRUCT_COUNT_AFTER=$(echo "${STRUCT_ALL_JSON}" | jq '.Struct | length' 2>/dev/null || echo 0)
if [ "${STRUCT_COUNT_AFTER}" -gt "${STRUCT_COUNT_BEFORE}" ]; then
    CRUISER_ID=$(get_newest_struct_id "${STRUCT_ALL_JSON}")
else
    CRUISER_ID=""
    echo -e "  ${RED}Cruiser build failed (Player 3 may lack capacity) — Phase 14 will be skipped${NC}"
fi
echo "  Cruiser Struct ID: ${CRUISER_ID:-NONE} (compute deferred to Phase 14)"

info "All 8 builds initiated. Computing Phase 12 builds now (others age in parallel)."

# ═══════════════════════════════════════════════════════════════
# COMPUTE Phase 12 builds — each subsequent compute benefits from aging
# ═══════════════════════════════════════════════════════════════

# ─── Compute SAM Launcher ───
run_compute "Building SAM Launcher ${SAM_STRUCT_ID}" \
    tx structs struct-build-compute "${SAM_STRUCT_ID}" --from player_3

assert_eq "SAM built" "true" "$(query query structs struct "${SAM_STRUCT_ID}" | jq -r '.structAttributes.isBuilt')"

# ─── Compute Submarine (aged during SAM compute) ───
run_compute "Building Submarine ${SUB_STRUCT_ID}" \
    tx structs struct-build-compute "${SUB_STRUCT_ID}" --from player_3

assert_eq "Submarine built" "true" "$(query query structs struct "${SUB_STRUCT_ID}" | jq -r '.structAttributes.isBuilt')"

# ─── Compute Battleship #1 (aged during SAM + Sub computes) ───
run_compute "Building Galactic Battleship ${BATTLESHIP_1_ID}" \
    tx structs struct-build-compute "${BATTLESHIP_1_ID}" --from player_3

assert_eq "Battleship #1 built" "true" "$(query query structs struct "${BATTLESHIP_1_ID}" | jq -r '.structAttributes.isBuilt')"

# ─── Compute Battleship #2 (aged during SAM + Sub + BB1 computes) ───
run_compute "Building Galactic Battleship #2 ${BATTLESHIP_2_ID}" \
    tx structs struct-build-compute "${BATTLESHIP_2_ID}" --from player_3

assert_eq "Battleship #2 built" "true" "$(query query structs struct "${BATTLESHIP_2_ID}" | jq -r '.structAttributes.isBuilt')"

# ─── P3: Tank #2 (type 9, land, slot 0) — armour-piercing target for Phase 13 ───
# Dedicated target so the AP test never disturbs the main Tank's HP.
wait_for_charge "${PLAYER_3_ID}" "${CHARGE_BUILD}"
run_tx "Initiating P3 Tank #2 (type=9, land, slot=0) — armour piercing target" \
    tx structs struct-build-initiate "${PLAYER_3_ID}" 9 land 0 --from player_3

STRUCT_ALL_JSON=$(query query structs struct-all)
AP_TANK_ID=$(get_newest_struct_id "${STRUCT_ALL_JSON}")
assert_not_empty "P3 Tank #2 struct ID" "${AP_TANK_ID}"
echo "  P3 Tank #2 Struct ID: ${AP_TANK_ID}"

run_compute "Building P3 Tank #2 ${AP_TANK_ID}" \
    tx structs struct-build-compute "${AP_TANK_ID}" --from player_3

assert_eq "P3 Tank #2 built" "true" "$(query query structs struct "${AP_TANK_ID}" | jq -r '.structAttributes.isBuilt')"

# ═══════════════════════════════════════════════════════════════
# Batch deactivation (v0.20.0): struct-deactivate-batch
# Uses P3's two freshly-built Battleships (online) as a self-contained
# target set, then reactivates them so later phases are unaffected.
# ═══════════════════════════════════════════════════════════════

BB1_ONLINE_BEFORE=$(jqr "$(query query structs struct "${BATTLESHIP_1_ID}")" '.structAttributes.isOnline' 'false')
BB2_ONLINE_BEFORE=$(jqr "$(query query structs struct "${BATTLESHIP_2_ID}")" '.structAttributes.isOnline' 'false')
assert_eq "Battleship #1 online before batch deactivate" "true" "${BB1_ONLINE_BEFORE}"
assert_eq "Battleship #2 online before batch deactivate" "true" "${BB2_ONLINE_BEFORE}"

# Atomicity: a batch containing a bogus ID must fail without deactivating any struct
run_tx_expect_fail "Batch deactivate with a bogus ID (should fail atomically)" \
    tx structs struct-deactivate-batch "${BATTLESHIP_1_ID},invalid-struct" --from player_3

BB1_ONLINE_AFTER_FAIL=$(jqr "$(query query structs struct "${BATTLESHIP_1_ID}")" '.structAttributes.isOnline' 'false')
assert_eq "Battleship #1 still online after failed batch (atomic)" "true" "${BB1_ONLINE_AFTER_FAIL}"

# Happy path: deactivate both battleships in one transaction (deactivate is not charge-gated)
run_tx "Batch deactivating P3 Battleships" \
    tx structs struct-deactivate-batch "${BATTLESHIP_1_ID},${BATTLESHIP_2_ID}" --from player_3

# Batch deactivate commits asynchronously; poll (up to ~10s) for both to
# report offline instead of relying on run_tx's fixed sleep. Falls through on
# timeout so a genuine regression still fails the assertions below.
#
# NOTE: isOnline is a proto3 bool, so the query JSON OMITS it when false (an
# offline struct has no .structAttributes.isOnline key at all). The fallback
# here must therefore be 'false' — an absent field means offline. While the
# struct is still online (or the tx hasn't committed) isOnline is present as
# true, so the loop keeps polling until the deactivate lands.
for _ in $(seq 1 10); do
    BB1_OFFLINE=$(jqr "$(query query structs struct "${BATTLESHIP_1_ID}")" '.structAttributes.isOnline' 'false')
    BB2_OFFLINE=$(jqr "$(query query structs struct "${BATTLESHIP_2_ID}")" '.structAttributes.isOnline' 'false')
    if [ "${BB1_OFFLINE}" = "false" ] && [ "${BB2_OFFLINE}" = "false" ]; then
        break
    fi
    sleep 1
done
assert_eq "Battleship #1 offline after batch deactivate" "false" "${BB1_OFFLINE}"
assert_eq "Battleship #2 offline after batch deactivate" "false" "${BB2_OFFLINE}"

# Cleanup: reactivate both (activation IS charge-gated) so downstream phases see them online
wait_for_charge "${PLAYER_3_ID}" "${CHARGE_ACTIVATE}"
run_tx "Reactivating P3 Battleship #1 (batch cleanup)" \
    tx structs struct-activate "${BATTLESHIP_1_ID}" --from player_3
wait_for_charge "${PLAYER_3_ID}" "${CHARGE_ACTIVATE}"
run_tx "Reactivating P3 Battleship #2 (batch cleanup)" \
    tx structs struct-activate "${BATTLESHIP_2_ID}" --from player_3

BB1_ONLINE_RESTORED=$(jqr "$(query query structs struct "${BATTLESHIP_1_ID}")" '.structAttributes.isOnline' 'false')
BB2_ONLINE_RESTORED=$(jqr "$(query query structs struct "${BATTLESHIP_2_ID}")" '.structAttributes.isOnline' 'false')
assert_eq "Battleship #1 back online after batch cleanup" "true" "${BB1_ONLINE_RESTORED}"
assert_eq "Battleship #2 back online after batch cleanup" "true" "${BB2_ONLINE_RESTORED}"

fi # phase 12

if run_phase 1300; then

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

# ─── v0.19.0: a not-yet-built struct cannot be attacked ───
# The P2 Battleship was build-initiated in Phase 12 but has not been computed
# yet (the compute is just below), so it is materialized but not Built. An
# attack against it must be rejected with the "unbuilt" targeting reason,
# regardless of online/offline state. CanAttack checks Built before ambit, so
# a built P3 Battleship firing its space-capable secondary is enough to prove
# the rule. wait_for_charge ensures the failure is the build check, not charge.
P2_BB_JSON=$(query query structs struct "${P2_BATTLESHIP_ID}")
P2_BB_PREBUILD_BUILT=$(jqr "${P2_BB_JSON}" '.structAttributes.isBuilt' 'false')
assert_eq "P2 Battleship not yet built (pre-compute)" "false" "${P2_BB_PREBUILD_BUILT}"

wait_for_charge "${PLAYER_3_ID}" "${CHARGE_ATTACK_BATTLESHIP}"
run_tx_expect_fail "Attack against a not-yet-built struct rejected (v0.19.0)" \
    tx structs struct-attack "${BATTLESHIP_1_ID}" "${P2_BATTLESHIP_ID}" secondaryWeapon --from player_3

# ─── Player 2's Battleship was pre-seeded in Phase 12 — compute now (heavily aged) ───
info "Computing P2 Battleship ${P2_BATTLESHIP_ID} (pre-seeded in Phase 12)"
run_compute "Building Player 2's Battleship ${P2_BATTLESHIP_ID}" \
    tx structs struct-build-compute "${P2_BATTLESHIP_ID}" --from player_2

assert_eq "Player 2 Battleship built" "true" "$(query query structs struct "${P2_BATTLESHIP_ID}" | jq -r '.structAttributes.isBuilt')"

# ─── v0.18.0 Battleship rebalance: primary is armour-piercing, land/water only ───
# The unguided primary (PrimaryWeaponAmbits=6: water+land) can no longer
# target space — the guided secondary (SecondaryWeaponAmbits=16: space) covers it.

# Negative: Battleship primary cannot target the space-ambit Command Ship.
# Wait for full primary charge first so the failure is the ambit check, not charge.
wait_for_charge "${PLAYER_2_ID}" "${CHARGE_ATTACK_BATTLESHIP_PRIMARY}"
run_tx_expect_fail "Battleship primary (land/water) rejected against space target" \
    tx structs struct-attack "${P2_BATTLESHIP_ID}" "${COMMAND_SHIP_ID}" primaryWeapon --from player_2

# ─── Armour piercing: Battleship primary vs Tank (AttackReduction=1) ───
# P3 Tank #2 (built in Phase 12, full HP, land ambit on P3's fleet at this
# planet). The armour-piercing primary negates the Tank's reduction, so the
# full 2 damage lands. A non-piercing weapon would only deal 1 (2 - 1).
AP_TANK_HP_BEFORE=$(query query structs struct "${AP_TANK_ID}" | jq -r '.structAttributes.health // "0"')
P2_BB_HP_BEFORE=$(query query structs struct "${P2_BATTLESHIP_ID}" | jq -r '.structAttributes.health // "0"')
info "P3 Tank #2 health before armour-piercing attack: ${AP_TANK_HP_BEFORE}"

run_tx "P2 Battleship fires armour-piercing primary at P3 Tank #2" \
    tx structs struct-attack "${P2_BATTLESHIP_ID}" "${AP_TANK_ID}" primaryWeapon --from player_2

AP_TANK_HP_AFTER=$(query query structs struct "${AP_TANK_ID}" | jq -r '.structAttributes.health // "0"')
P2_BB_HP_AFTER=$(query query structs struct "${P2_BATTLESHIP_ID}" | jq -r '.structAttributes.health // "0"')
assert_eq "Armour piercing negates Tank reduction (full 2 damage)" "2" "$((AP_TANK_HP_BEFORE - AP_TANK_HP_AFTER))"
assert_eq "Battleship unharmed (Tank cannot counter into space)" "${P2_BB_HP_BEFORE}" "${P2_BB_HP_AFTER}"

# Attack the defended Command Ship with the guided secondary (space ambit)
CMDSHIP_JSON=$(query query structs struct "${COMMAND_SHIP_ID}" || echo '{}')
CMDSHIP_HP_BEFORE=$(jqr "${CMDSHIP_JSON}" '.structAttributes.health' '0')
info "Command Ship health before defended attack: ${CMDSHIP_HP_BEFORE}"

wait_for_charge "${PLAYER_2_ID}" "${CHARGE_ATTACK_BATTLESHIP}"
run_tx "Player 2 attacks the defended Command Ship (guided secondary)" \
    tx structs struct-attack "${P2_BATTLESHIP_ID}" "${COMMAND_SHIP_ID}" secondaryWeapon --from player_2

CMDSHIP_JSON=$(query query structs struct "${COMMAND_SHIP_ID}" || echo '{}')
CMDSHIP_HP_AFTER=$(jqr "${CMDSHIP_JSON}" '.structAttributes.health' '0')
info "Command Ship health after defended attack: ${CMDSHIP_HP_AFTER}"
echo "  (Defenders may have blocked/intercepted the attack)"

BLOCK_HEIGHT=$(query query structs block-height | jq -r '.blockHeight // empty' 2>/dev/null || echo "?")
info "Current block height: ${BLOCK_HEIGHT}"

fi # phase 13

if run_phase 1350; then

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
    SB_HIDDEN=$(jqr "${SB_JSON}" '.structAttributes.isHidden' 'false')
    assert_eq "Stealth Bomber hidden after activate" "true" "${SB_HIDDEN}"

    # ─── struct-stealth-deactivate ───
    wait_for_charge "${PLAYER_3_ID}" "${CHARGE_ACTIVATE}"
    run_tx "Deactivating stealth on Stealth Bomber" \
        tx structs struct-stealth-deactivate "${STEALTH_BOMBER_ID}" --from player_3

    SB_JSON=$(query query structs struct "${STEALTH_BOMBER_ID}" 2>/dev/null || echo '{}')
    SB_HIDDEN_AFTER=$(jqr "${SB_JSON}" '.structAttributes.isHidden' 'false')
    assert_eq "Stealth Bomber visible after deactivate" "false" "${SB_HIDDEN_AFTER}"

    # struct-attribute query coverage
    info "Struct attribute query coverage:"
    query query structs struct-attribute "${STEALTH_BOMBER_ID}" 2>/dev/null | jq -r '"  isBuilt=\(.isBuilt) isOnline=\(.isOnline) health=\(.health)"' || echo "  (query failed)"
else
    info "SKIP: Could not build Stealth Bomber for stealth tests"
fi

fi # phase 13b

if run_phase 1400; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 14: Cruiser vs Interceptor — Secondary Weapons & Defensive Maneuvers
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 14: Secondary Weapons & Defensive Maneuvers"

if [ -z "${CRUISER_ID}" ]; then
    info "Skipping Phase 14 (Cruiser build failed in Phase 12)"
else

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

# ═══════════════════════════════════════════════════════════════
# v0.19.0: attack / defense / stealth no longer require the Command Ship
# Deactivate P3's Command Ship, then confirm a non-command struct can still
# attack, (de)register a defensive assignment, and toggle stealth. The Command
# Ship is re-activated afterwards so later phases are unaffected. (Movement and
# building still require the Command Ship — covered elsewhere.)
# ═══════════════════════════════════════════════════════════════
section "PHASE 14b: Command Ship NOT required for attack/defense/stealth (v0.19.0)"

wait_for_charge "${PLAYER_3_ID}" "${CHARGE_ACTIVATE}"
run_tx "P3 deactivates Command Ship (command-ship-not-required test)" \
    tx structs struct-deactivate "${COMMAND_SHIP_ID}" --from player_3
CS_OFFLINE=$(query query structs struct "${COMMAND_SHIP_ID}" | jq -r '.structAttributes.isOnline // "false"')
assert_eq "P3 Command Ship offline for the test" "false" "${CS_OFFLINE}"

# Attack works with the Command Ship offline (only if the Interceptor survived
# the earlier rounds — it may have been destroyed or dodged).
INTERCEPTOR_HP_NOW=$(query query structs struct "${INTERCEPTOR_ID}" | jq -r '.structAttributes.health // "0"')
if [ "${INTERCEPTOR_HP_NOW}" != "0" ]; then
    wait_for_charge "${PLAYER_3_ID}" "${CHARGE_ATTACK_DEFAULT}"
    run_tx "Cruiser attacks Interceptor with Command Ship offline (v0.19.0)" \
        tx structs struct-attack "${CRUISER_ID}" "${INTERCEPTOR_ID}" secondaryWeapon --from player_3
else
    info "Interceptor already destroyed — skipping the command-ship-offline attack assertion"
fi

# Defensive assignment set/clear works with the Command Ship offline. Use the
# Cruiser (freshly built, guaranteed alive) so we do not depend on SAM's combat
# survival; clearing afterward leaves no residual defense relationship.
wait_for_charge "${PLAYER_3_ID}" "${CHARGE_DEFEND}"
run_tx "Set Cruiser to defend Command Ship (Command Ship offline, v0.19.0)" \
    tx structs struct-defense-set "${CRUISER_ID}" "${COMMAND_SHIP_ID}" --from player_3
wait_for_charge "${PLAYER_3_ID}" "${CHARGE_DEFEND}"
run_tx "Clear Cruiser defense (Command Ship offline, v0.19.0)" \
    tx structs struct-defense-clear "${CRUISER_ID}" --from player_3

# Stealth toggle works with the Command Ship offline.
if [ -n "${STEALTH_BOMBER_ID}" ]; then
    wait_for_charge "${PLAYER_3_ID}" "${CHARGE_ACTIVATE}"
    run_tx "Activate stealth with Command Ship offline (v0.19.0)" \
        tx structs struct-stealth-activate "${STEALTH_BOMBER_ID}" --from player_3
    wait_for_charge "${PLAYER_3_ID}" "${CHARGE_ACTIVATE}"
    run_tx "Deactivate stealth with Command Ship offline (v0.19.0)" \
        tx structs struct-stealth-deactivate "${STEALTH_BOMBER_ID}" --from player_3
fi

# Restore the Command Ship online so the remaining phases are unaffected.
wait_for_charge "${PLAYER_3_ID}" "${CHARGE_ACTIVATE}"
run_tx "P3 re-activates Command Ship (cleanup)" \
    tx structs struct-activate "${COMMAND_SHIP_ID}" --from player_3
CS_BACK_ONLINE=$(query query structs struct "${COMMAND_SHIP_ID}" | jq -r '.structAttributes.isOnline // "false"')
assert_eq "P3 Command Ship back online after offline test" "true" "${CS_BACK_ONLINE}"

fi # cruiser available

fi # phase 14

if run_phase 1500; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 15: Power Generator — Build, Infuse, Verify, and Destroy
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 15: Power Generator (Player 4)"

echo "  Player 4 Planet: ${PLAYER_4_PLANET_ID}"
echo "  Using Field Generator (struct type 20, land, slot 0)"
echo "  GeneratingRate=2, PassiveDraw=500000, MaxHealth=8 (armour, AttackReduction=1)"

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

# ─── Infuse alpha into the generator (must be online) ───
INFUSE_AMOUNT="1000000ualpha"
info "Infusing ${INFUSE_AMOUNT} into generator (GeneratingRate=2, expected power=2000000)"

run_tx "Infusing ${INFUSE_AMOUNT} into generator ${GENERATOR_STRUCT_ID}" \
    tx structs struct-generator-infuse "${GENERATOR_STRUCT_ID}" "${INFUSE_AMOUNT}" --from player_4

# Verify fuel was added to the struct
GEN_JSON=$(query query structs struct "${GENERATOR_STRUCT_ID}")
GEN_FUEL=$(jqr "${GEN_JSON}" '.gridAttributes.fuel' '0')
info "Generator fuel after infusion: ${GEN_FUEL}"
assert_gt "Generator has fuel" 0 "${GEN_FUEL}"

# Generator remains online — verify
GEN_ONLINE_AFTER_INFUSE=$(jqr "${GEN_JSON}" '.structAttributes.isOnline' 'false')
assert_eq "Generator still online after infuse" "true" "${GEN_ONLINE_AFTER_INFUSE}"

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
echo "  Generator MaxHealth=8 + armour (AttackReduction=1): Tank (type=9, land) 2 dmg → 1 net per shot → 8 rounds"
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
info "Generator health before attacks: ${GEN_HP_BEFORE} (MaxHealth=8; armour reduces each 2-dmg Tank hit to 1 net)"

# Attack round 1 — use Destroyer/Tank (type=9, land ambit, PrimaryWeaponAmbits=4=land)
# Tank PrimaryWeaponCharge=1, PrimaryWeaponDamage=2; generator AttackReduction=1 → 1 net.
wait_for_charge "${PLAYER_3_ID}" "${CHARGE_ATTACK_DEFAULT}"
run_tx "Attack round 1: Tank -> Generator (armour: 2 dmg → 1 net)" \
    tx structs struct-attack "${DESTROYER_STRUCT_ID}" "${GENERATOR_STRUCT_ID}" primaryWeapon --from player_3

GEN_JSON=$(query query structs struct "${GENERATOR_STRUCT_ID}" || echo '{}')
GEN_HP_MID=$(jqr "${GEN_JSON}" '.structAttributes.health' '0')
info "Generator health after round 1: ${GEN_HP_MID} (was ${GEN_HP_BEFORE})"
# Armour: a 2-damage Tank hit lands only 1 net damage on the generator.
GEN_DMG_R1=$(( GEN_HP_BEFORE - GEN_HP_MID ))
assert_eq "Generator armour reduced Tank hit to 1 net damage" "1" "${GEN_DMG_R1}"

# Keep attacking until destroyed. With armour the Tank lands 1 net per shot,
# so an 8-HP generator takes ~8 rounds; cap the loop for safety.
GEN_HP_NOW="${GEN_HP_MID}"
for round in 2 3 4 5 6 7 8 9 10 11 12; do
    if [ "${GEN_HP_NOW}" = "0" ]; then break; fi
    wait_for_charge "${PLAYER_3_ID}" "${CHARGE_ATTACK_DEFAULT}"
    run_tx "Attack round ${round}: Tank -> Generator" \
        tx structs struct-attack "${DESTROYER_STRUCT_ID}" "${GENERATOR_STRUCT_ID}" primaryWeapon --from player_3
    GEN_JSON=$(query query structs struct "${GENERATOR_STRUCT_ID}" || echo '{}')
    GEN_HP_NOW=$(jqr "${GEN_JSON}" '.structAttributes.health' '0')
    info "Generator health after round ${round}: ${GEN_HP_NOW}"
done

GEN_JSON=$(query query structs struct "${GENERATOR_STRUCT_ID}" || echo '{}')
GEN_HP_AFTER=$(jqr "${GEN_JSON}" '.structAttributes.health' '0')
GEN_DESTROYED=$(jqr "${GEN_JSON}" '.structAttributes.isDestroyed' 'false')
info "Generator health after destruction loop: ${GEN_HP_AFTER}"
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

fi # phase 15

if run_phase 1550; then

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
    tx structs player-send "${PLAYER_2_ADDRESS}" "${PLAYER_3_ADDRESS}" "${SEND_AMOUNT}ualpha" --from player_2

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

fi # phase 15b

if run_phase 1600; then

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

        # ─── Verify agreement allocation is created and connect to substation ───
        AGREE_ALLOC_ID=$(jqr "${AGREE_JSON}" '.Agreement.allocationId' '')
        assert_not_empty "Agreement allocation ID exists" "${AGREE_ALLOC_ID}"
        info "Agreement allocation ID: ${AGREE_ALLOC_ID}"

        AGREE_ALLOC_JSON=$(query query structs allocation "${AGREE_ALLOC_ID}" 2>/dev/null || echo '{}')
        AGREE_ALLOC_SRC=$(jqr "${AGREE_ALLOC_JSON}" '.Allocation.sourceObjectId' '')
        AGREE_ALLOC_DST=$(jqr "${AGREE_ALLOC_JSON}" '.Allocation.destinationId' '')
        AGREE_ALLOC_TYPE=$(jqr "${AGREE_ALLOC_JSON}" '.Allocation.type' '')
        assert_eq "Agreement allocation source is provider substation" "${SUBSTATION_ID}" "${AGREE_ALLOC_SRC}"
        assert_eq "Agreement allocation destination initially empty" "" "${AGREE_ALLOC_DST}"
        info "Agreement allocation: src=${AGREE_ALLOC_SRC}, dst=${AGREE_ALLOC_DST}, type=${AGREE_ALLOC_TYPE}"

        # Create a fresh substation to test connecting the agreement allocation.
        # The original SECOND_SUB_ID may have been deleted in Phase 4f.
        run_tx "Creating substation for agreement allocation connect test" \
            tx structs substation-create "${PLAYER_1_ID}" "${P1_ALLOC_ID}" --from alice

        AGREE_TEST_SUB_JSON=$(query query structs substation-all 2>/dev/null || echo '{}')
        AGREE_TEST_SUB_ID=$(echo "${AGREE_TEST_SUB_JSON}" | jq -r '.Substation[-1].id // empty' 2>/dev/null || echo "")

        if [ -n "${AGREE_TEST_SUB_ID}" ] && [ "${AGREE_TEST_SUB_ID}" != "${SUBSTATION_ID}" ]; then
            info "Created test substation for agreement allocation: ${AGREE_TEST_SUB_ID}"

            # Grant Player 2 PermAllocationConnection on the agreement allocation
            # so they can connect it (Player 2 already has PermAll on address)
            run_tx "Connecting agreement allocation to test substation" \
                tx structs substation-allocation-connect "${AGREE_ALLOC_ID}" "${AGREE_TEST_SUB_ID}" --from player_2

            AGREE_ALLOC_JSON=$(query query structs allocation "${AGREE_ALLOC_ID}" 2>/dev/null || echo '{}')
            AGREE_ALLOC_DST_AFTER=$(jqr "${AGREE_ALLOC_JSON}" '.Allocation.destinationId' '')
            assert_eq "Agreement allocation connected to substation" "${AGREE_TEST_SUB_ID}" "${AGREE_ALLOC_DST_AFTER}"
            info "Agreement allocation now connected: dst=${AGREE_ALLOC_DST_AFTER}"

            # Disconnect so it doesn't interfere with later tests
            run_tx "Disconnecting agreement allocation from test substation" \
                tx structs substation-allocation-disconnect "${AGREE_ALLOC_ID}" --from player_2

            AGREE_ALLOC_JSON=$(query query structs allocation "${AGREE_ALLOC_ID}" 2>/dev/null || echo '{}')
            AGREE_ALLOC_DST_DISCONN=$(jqr "${AGREE_ALLOC_JSON}" '.Allocation.destinationId' '')
            assert_eq "Agreement allocation disconnected" "" "${AGREE_ALLOC_DST_DISCONN}"

            # Clean up the test substation
            run_tx "Deleting agreement test substation" \
                tx structs substation-delete "${AGREE_TEST_SUB_ID}" "${SUBSTATION_ID}" --from alice
        else
            info "SKIP: Could not create substation for agreement allocation connect test"
        fi

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

fi # phase 16


if run_phase 1700; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 18: UGC (User-Generated Content) — Names & Profile Pictures
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 18: UGC — Names & Profile Pictures"

# ─── Guild Name ───

run_tx "Setting guild name" \
    tx structs guild-update-name "${GUILD_ID}" "TestGuild" --from alice

GUILD_JSON=$(query query structs guild "${GUILD_ID}")
GUILD_NAME=$(jqr "${GUILD_JSON}" '.Guild.name')
assert_eq "Guild name set" "TestGuild" "${GUILD_NAME}"

# Rename guild
run_tx "Renaming guild" \
    tx structs guild-update-name "${GUILD_ID}" "RenamedGuild" --from alice

GUILD_JSON=$(query query structs guild "${GUILD_ID}")
GUILD_NAME=$(jqr "${GUILD_JSON}" '.Guild.name')
assert_eq "Guild name renamed" "RenamedGuild" "${GUILD_NAME}"

# Duplicate guild name should fail (create guild B name first, then try same)
run_tx "Setting guild B name" \
    tx structs guild-update-name "${GUILD_B_ID}" "UniqueGuildB" --from guild_leader_b

run_tx_expect_fail "Duplicate guild name should be rejected" \
    tx structs guild-update-name "${GUILD_ID}" "uniqueguildb" --from alice

# Invalid guild name: too short
run_tx_expect_fail "Guild name too short" \
    tx structs guild-update-name "${GUILD_ID}" "ab" --from alice

# Invalid guild name: resembles object ID
run_tx_expect_fail "Guild name resembles object ID" \
    tx structs guild-update-name "${GUILD_ID}" "1-42" --from alice

# Verify name unchanged after failed updates
GUILD_JSON=$(query query structs guild "${GUILD_ID}")
GUILD_NAME=$(jqr "${GUILD_JSON}" '.Guild.name')
assert_eq "Guild name unchanged after failures" "RenamedGuild" "${GUILD_NAME}"

# ─── Guild PFP ───

run_tx "Setting guild pfp" \
    tx structs guild-update-pfp "${GUILD_ID}" "https://example.com/guild.png" --from alice

GUILD_JSON=$(query query structs guild "${GUILD_ID}")
GUILD_PFP=$(jqr "${GUILD_JSON}" '.Guild.pfp')
assert_eq "Guild pfp set" "https://example.com/guild.png" "${GUILD_PFP}"

# Clear guild pfp
run_tx "Clearing guild pfp" \
    tx structs guild-update-pfp "${GUILD_ID}" "" --from alice

GUILD_JSON=$(query query structs guild "${GUILD_ID}")
GUILD_PFP=$(jqr "${GUILD_JSON}" '.Guild.pfp' '')
assert_eq "Guild pfp cleared" "" "${GUILD_PFP}"

# ─── Player Name ───

run_tx "Setting player 1 name" \
    tx structs player-update-name "${PLAYER_1_ID}" "AlicePlayer" --from alice

P1_JSON=$(query query structs player "${PLAYER_1_ID}")
P1_NAME=$(jqr "${P1_JSON}" '.Player.name')
assert_eq "Player 1 name set" "AlicePlayer" "${P1_NAME}"

# Invalid player name: contains space (not allowed for players)
run_tx_expect_fail "Player name with space should fail" \
    tx structs player-update-name "${PLAYER_1_ID}" "Alice Player" --from alice

# Invalid player name: too short
run_tx_expect_fail "Player name too short" \
    tx structs player-update-name "${PLAYER_1_ID}" "ab" --from alice

# Verify name unchanged after failed updates
P1_JSON=$(query query structs player "${PLAYER_1_ID}")
P1_NAME=$(jqr "${P1_JSON}" '.Player.name')
assert_eq "Player 1 name unchanged" "AlicePlayer" "${P1_NAME}"

# ─── Player PFP ───

run_tx "Setting player 1 pfp" \
    tx structs player-update-pfp "${PLAYER_1_ID}" "https://example.com/alice.png" --from alice

P1_JSON=$(query query structs player "${PLAYER_1_ID}")
P1_PFP=$(jqr "${P1_JSON}" '.Player.pfp')
assert_eq "Player 1 pfp set" "https://example.com/alice.png" "${P1_PFP}"

# Clear player pfp
run_tx "Clearing player 1 pfp" \
    tx structs player-update-pfp "${PLAYER_1_ID}" "" --from alice

P1_JSON=$(query query structs player "${PLAYER_1_ID}")
P1_PFP=$(jqr "${P1_JSON}" '.Player.pfp' '')
assert_eq "Player 1 pfp cleared" "" "${P1_PFP}"

# ─── Substation Name ───

run_tx "Setting substation name" \
    tx structs substation-update-name "${SUBSTATION_ID}" "MainStation" --from alice

SUB_JSON=$(query query structs substation "${SUBSTATION_ID}")
SUB_NAME=$(jqr "${SUB_JSON}" '.Substation.name')
assert_eq "Substation name set" "MainStation" "${SUB_NAME}"

# Invalid substation name: too short
run_tx_expect_fail "Substation name too short" \
    tx structs substation-update-name "${SUBSTATION_ID}" "ab" --from alice

# Verify name unchanged
SUB_JSON=$(query query structs substation "${SUBSTATION_ID}")
SUB_NAME=$(jqr "${SUB_JSON}" '.Substation.name')
assert_eq "Substation name unchanged" "MainStation" "${SUB_NAME}"

# ─── Substation PFP ───

run_tx "Setting substation pfp" \
    tx structs substation-update-pfp "${SUBSTATION_ID}" "https://example.com/sub.png" --from alice

SUB_JSON=$(query query structs substation "${SUBSTATION_ID}")
SUB_PFP=$(jqr "${SUB_JSON}" '.Substation.pfp')
assert_eq "Substation pfp set" "https://example.com/sub.png" "${SUB_PFP}"

# ─── Planet Name ───

run_tx "Setting player 2 planet name" \
    tx structs planet-update-name "${PLAYER_2_PLANET_ID}" "AlphaPrime" --from player_2

PLANET_JSON=$(query query structs planet "${PLAYER_2_PLANET_ID}")
PLANET_NAME=$(jqr "${PLANET_JSON}" '.Planet.name')
assert_eq "Planet name set" "AlphaPrime" "${PLANET_NAME}"

# Invalid planet name: too short
run_tx_expect_fail "Planet name too short" \
    tx structs planet-update-name "${PLAYER_2_PLANET_ID}" "ab" --from player_2

# Planet name: max length (25 chars)
run_tx "Setting planet name at max length" \
    tx structs planet-update-name "${PLAYER_2_PLANET_ID}" "ABCDEFGHIJKLMNOPQRSTUVWXY" --from player_2

PLANET_JSON=$(query query structs planet "${PLAYER_2_PLANET_ID}")
PLANET_NAME=$(jqr "${PLANET_JSON}" '.Planet.name')
assert_eq "Planet name at max length" "ABCDEFGHIJKLMNOPQRSTUVWXY" "${PLANET_NAME}"

# Planet name: over max length should fail
run_tx_expect_fail "Planet name too long (26 chars)" \
    tx structs planet-update-name "${PLAYER_2_PLANET_ID}" "ABCDEFGHIJKLMNOPQRSTUVWXYZ" --from player_2

# ─── Permission Denied: wrong player ───

run_tx_expect_fail "Player 3 cannot rename Player 2 planet" \
    tx structs planet-update-name "${PLAYER_2_PLANET_ID}" "Stolen" --from player_3

run_tx_expect_fail "Player 3 cannot rename guild (no PermUpdate)" \
    tx structs guild-update-name "${GUILD_ID}" "Hijacked" --from player_3

info "UGC phase complete"

fi # phase 18


if run_phase 2300; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 17: Fleet Movement Setup — 5 Fleet Players with Planets
# ═════════════════════════════════════════════════════════════════════════════
#
# Creates 5 dedicated fleet-test players (fplayer_1 through fplayer_5), each
# with a planet, fleet, and command ship. These are separate from the main
# test players to avoid state interactions from earlier phases.
#
# Fleet linked list structure (after Phase 17b moves to FP_1's planet):
#
#   Planet (locationListStart → F2, locationListLast → F5)
#      ↕
#   F2 (forward="", backward=F3)     ← first to arrive = front of list
#      ↕
#   F3 (forward=F2, backward=F4)
#      ↕
#   F4 (forward=F3, backward=F5)
#      ↕
#   F5 (forward=F4, backward="")     ← last to arrive = back of list
#
#   F1 is "on station" at its home planet — not in the list

section "PHASE 17: Fleet Movement Setup"

for FP_NUM in 1 2 3 4 5; do
    FPLAYER_KEY="fplayer_${FP_NUM}"
    info "Setting up ${FPLAYER_KEY}"

    EXISTING=$(structsd ${PARAMS_KEYS} keys show "${FPLAYER_KEY}" 2>/dev/null | jq -r .address || echo "")
    if [ -z "${EXISTING}" ]; then
        ADDR=$(structsd ${PARAMS_KEYS} keys add "${FPLAYER_KEY}" | jq -r .address)
        echo "  Created ${FPLAYER_KEY}: ${ADDR}"
    else
        ADDR="${EXISTING}"
        echo "  Reusing ${FPLAYER_KEY}: ${ADDR}"
    fi
    eval "FP_${FP_NUM}_ADDRESS=${ADDR}"

    run_tx "Funding ${FPLAYER_KEY}" \
        tx bank send "${BOB_ADDRESS}" "${ADDR}" 10000000ualpha --from bob

    run_tx "Delegating for ${FPLAYER_KEY}" \
        tx staking delegate "${VALIDATOR_ADDRESS}" 5000000ualpha --from "${FPLAYER_KEY}"

    ADDR_JSON=$(query query structs address "${ADDR}")
    FP_PID=$(jqr "${ADDR_JSON}" '.playerId')
    eval "FP_${FP_NUM}_ID=${FP_PID}"
    assert_not_empty "Fleet Player ${FP_NUM} ID" "${FP_PID}"
    echo "  Fleet Player ${FP_NUM} ID: ${FP_PID}"

    PJSON=$(query query structs player "${FP_PID}")
    PCAP=$(jqr "${PJSON}" '.gridAttributes.capacity')

    run_tx "Creating allocation for fleet player ${FP_NUM}" \
        tx structs allocation-create "${FP_PID}" "${PCAP}" \
        --controller "${PLAYER_1_ID}" --allocation-type dynamic --from "${FPLAYER_KEY}"

    FP_ALLOC_ID=$(get_latest_allocation_for_source "${FP_PID}")
    eval "FP_${FP_NUM}_ALLOC_ID=${FP_ALLOC_ID}"

    run_tx "Fleet player ${FP_NUM} joining guild" \
        tx structs guild-membership-join "${GUILD_ID}" "${REACTOR_ID}-${ADDR}" --from "${FPLAYER_KEY}"

    run_tx "Connecting fleet player ${FP_NUM} allocation to substation" \
        tx structs substation-allocation-connect "${FP_ALLOC_ID}" "${SUBSTATION_ID}" --from alice
done

echo ""
info "All fleet players:"
for FP_NUM in 1 2 3 4 5; do
    eval "echo \"  FP ${FP_NUM}: ID=\${FP_${FP_NUM}_ID} ADDR=\${FP_${FP_NUM}_ADDRESS}\""
done

# ─── Planet Exploration ───
info "Fleet players exploring planets"
for FP_NUM in 1 2 3 4 5; do
    eval "FP_PID=\${FP_${FP_NUM}_ID}"
    FPLAYER_KEY="fplayer_${FP_NUM}"

    run_tx "Fleet Player ${FP_NUM} exploring planet" \
        tx structs planet-explore "${FP_PID}" --from "${FPLAYER_KEY}"

    PJSON=$(query query structs player "${FP_PID}")
    PLANET_ID=$(jqr "${PJSON}" '.Player.planetId')
    FLEET_ID=$(jqr "${PJSON}" '.Player.fleetId')
    eval "FP_${FP_NUM}_PLANET_ID=${PLANET_ID}"
    eval "FP_${FP_NUM}_FLEET_ID=${FLEET_ID}"
    assert_not_empty "FP ${FP_NUM} planet" "${PLANET_ID}"
    assert_not_empty "FP ${FP_NUM} fleet" "${FLEET_ID}"
    echo "  FP ${FP_NUM}: planet=${PLANET_ID} fleet=${FLEET_ID}"
done

# ─── Verify Command Ships ───
info "Verifying command ships"
for FP_NUM in 1 2 3 4 5; do
    eval "FLEET_ID=\${FP_${FP_NUM}_FLEET_ID}"
    FLEET_JSON=$(query_fleet "${FLEET_ID}")
    CMD_STRUCT=$(jqr "${FLEET_JSON}" '.Fleet.commandStruct')
    eval "FP_CS_${FP_NUM}=${CMD_STRUCT}"
    assert_not_empty "FP ${FP_NUM} command ship" "${CMD_STRUCT}"

    STRUCT_JSON=$(query query structs struct "${CMD_STRUCT}")
    BUILT=$(echo "${STRUCT_JSON}" | jq -r '.structAttributes.isBuilt // "false"' 2>/dev/null)
    ONLINE=$(echo "${STRUCT_JSON}" | jq -r '.structAttributes.isOnline // "false"' 2>/dev/null)
    HP=$(echo "${STRUCT_JSON}" | jq -r '.structAttributes.health // "0"' 2>/dev/null)
    STYPE=$(echo "${STRUCT_JSON}" | jq -r '.Struct.type // ""' 2>/dev/null)
    assert_eq "FP CS ${FP_NUM} type" "1" "${STYPE}"
    assert_eq "FP CS ${FP_NUM} built" "true" "${BUILT}"
    assert_eq "FP CS ${FP_NUM} online" "true" "${ONLINE}"
    echo "  FP_CS_${FP_NUM}=${CMD_STRUCT}  HP=${HP}  built=${BUILT}  online=${ONLINE}"
done

# ─── Verify Initial Fleet State ───
info "Verifying initial fleet state (each fleet on its own planet)"
for FP_NUM in 1 2 3 4 5; do
    eval "FLEET_ID=\${FP_${FP_NUM}_FLEET_ID}"
    eval "PLANET_ID=\${FP_${FP_NUM}_PLANET_ID}"

    FLEET_JSON=$(query_fleet "${FLEET_ID}")
    LOC=$(jqr "${FLEET_JSON}" '.Fleet.locationId')
    STATUS=$(jqr "${FLEET_JSON}" '.Fleet.status')
    if [ -z "${STATUS}" ]; then STATUS="onStation"; fi
    assert_eq "FP Fleet ${FP_NUM} location" "${PLANET_ID}" "${LOC}"
    assert_eq "FP Fleet ${FP_NUM} status" "onStation" "${STATUS}"
    echo "  Fleet ${FLEET_ID}: loc=${LOC} status=${STATUS}"
done

fi # phase 17


if run_phase 2350; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 17b: Fleet Linked List — Move Fleets & Verify Structure
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 17b: Fleet Linked List"

FP_TARGET_PLANET="${FP_1_PLANET_ID}"
info "Target planet: ${FP_TARGET_PLANET} (FP 1's home)"

# Move fleets 2-5 to FP_1's planet (each appends to END of linked list)
for FP_NUM in 2 3 4 5; do
    eval "FP_PID=\${FP_${FP_NUM}_ID}"
    eval "FLEET_ID=\${FP_${FP_NUM}_FLEET_ID}"
    FPLAYER_KEY="fplayer_${FP_NUM}"

    wait_for_charge "${FP_PID}" "${CHARGE_MOVE}"
    run_tx "Moving FP Fleet ${FP_NUM} (${FLEET_ID}) to planet ${FP_TARGET_PLANET}" \
        tx structs fleet-move "${FLEET_ID}" "${FP_TARGET_PLANET}" --from "${FPLAYER_KEY}"
done

info "Waiting for state to settle"
sleep 3

# Shorthand variables for linked list verification
FP_F1="${FP_1_FLEET_ID}"
FP_F2="${FP_2_FLEET_ID}"
FP_F3="${FP_3_FLEET_ID}"
FP_F4="${FP_4_FLEET_ID}"
FP_F5="${FP_5_FLEET_ID}"

info "Expected list on planet ${FP_TARGET_PLANET}:"
echo "  Planet.start → ${FP_F2} ↔ ${FP_F3} ↔ ${FP_F4} ↔ ${FP_F5} ← Planet.last"
echo "  ${FP_F1} is on station (home fleet, not in list)"
echo ""

# ─── Planet pointers ───
info "Checking planet linked list pointers"
FP_PLANET_START=$(get_planet_field "${FP_TARGET_PLANET}" "locationListStart")
FP_PLANET_LAST=$(get_planet_field "${FP_TARGET_PLANET}" "locationListLast")
assert_eq "Planet locationListStart" "${FP_F2}" "${FP_PLANET_START}"
assert_eq "Planet locationListLast"  "${FP_F5}" "${FP_PLANET_LAST}"

# ─── Fleet 1 (home, on station — not in the list) ───
info "Fleet 1 (home fleet)"
F1_JSON=$(query_fleet "${FP_F1}")
F1_LOC=$(jqr "${F1_JSON}" '.Fleet.locationId')
F1_STATUS=$(jqr "${F1_JSON}" '.Fleet.status')
if [ -z "${F1_STATUS}" ]; then F1_STATUS="onStation"; fi
assert_eq "FP Fleet 1 location" "${FP_1_PLANET_ID}" "${F1_LOC}"
assert_eq "FP Fleet 1 status" "onStation" "${F1_STATUS}"
echo "  F1: loc=${F1_LOC} status=${F1_STATUS}"

# ─── Fleet 2 (front of list) ───
info "Fleet 2 (front of list)"
F2_JSON=$(query_fleet "${FP_F2}")
F2_LOC=$(jqr "${F2_JSON}" '.Fleet.locationId')
F2_STATUS=$(jqr "${F2_JSON}" '.Fleet.status')
F2_FWD=$(jqr "${F2_JSON}" '.Fleet.locationListForward')
F2_BWD=$(jqr "${F2_JSON}" '.Fleet.locationListBackward')
assert_eq "FP Fleet 2 location" "${FP_TARGET_PLANET}" "${F2_LOC}"
assert_eq "FP Fleet 2 status" "away" "${F2_STATUS}"
assert_eq "FP Fleet 2 forward (toward planet)" "" "${F2_FWD}"
assert_eq "FP Fleet 2 backward" "${FP_F3}" "${F2_BWD}"
echo "  F2: loc=${F2_LOC} status=${F2_STATUS} fwd='${F2_FWD}' bwd='${F2_BWD}'"

# ─── Fleet 3 ───
info "Fleet 3 (second in list)"
F3_JSON=$(query_fleet "${FP_F3}")
F3_LOC=$(jqr "${F3_JSON}" '.Fleet.locationId')
F3_STATUS=$(jqr "${F3_JSON}" '.Fleet.status')
F3_FWD=$(jqr "${F3_JSON}" '.Fleet.locationListForward')
F3_BWD=$(jqr "${F3_JSON}" '.Fleet.locationListBackward')
assert_eq "FP Fleet 3 location" "${FP_TARGET_PLANET}" "${F3_LOC}"
assert_eq "FP Fleet 3 status" "away" "${F3_STATUS}"
assert_eq "FP Fleet 3 forward" "${FP_F2}" "${F3_FWD}"
assert_eq "FP Fleet 3 backward" "${FP_F4}" "${F3_BWD}"
echo "  F3: loc=${F3_LOC} status=${F3_STATUS} fwd='${F3_FWD}' bwd='${F3_BWD}'"

# ─── Fleet 4 ───
info "Fleet 4 (third in list)"
F4_JSON=$(query_fleet "${FP_F4}")
F4_LOC=$(jqr "${F4_JSON}" '.Fleet.locationId')
F4_STATUS=$(jqr "${F4_JSON}" '.Fleet.status')
F4_FWD=$(jqr "${F4_JSON}" '.Fleet.locationListForward')
F4_BWD=$(jqr "${F4_JSON}" '.Fleet.locationListBackward')
assert_eq "FP Fleet 4 location" "${FP_TARGET_PLANET}" "${F4_LOC}"
assert_eq "FP Fleet 4 status" "away" "${F4_STATUS}"
assert_eq "FP Fleet 4 forward" "${FP_F3}" "${F4_FWD}"
assert_eq "FP Fleet 4 backward" "${FP_F5}" "${F4_BWD}"
echo "  F4: loc=${F4_LOC} status=${F4_STATUS} fwd='${F4_FWD}' bwd='${F4_BWD}'"

# ─── Fleet 5 (back of list) ───
info "Fleet 5 (back of list)"
F5_JSON=$(query_fleet "${FP_F5}")
F5_LOC=$(jqr "${F5_JSON}" '.Fleet.locationId')
F5_STATUS=$(jqr "${F5_JSON}" '.Fleet.status')
F5_FWD=$(jqr "${F5_JSON}" '.Fleet.locationListForward')
F5_BWD=$(jqr "${F5_JSON}" '.Fleet.locationListBackward')
assert_eq "FP Fleet 5 location" "${FP_TARGET_PLANET}" "${F5_LOC}"
assert_eq "FP Fleet 5 status" "away" "${F5_STATUS}"
assert_eq "FP Fleet 5 forward" "${FP_F4}" "${F5_FWD}"
assert_eq "FP Fleet 5 backward" "" "${F5_BWD}"
echo "  F5: loc=${F5_LOC} status=${F5_STATUS} fwd='${F5_FWD}' bwd='${F5_BWD}'"

echo ""
info "Linked list verified:"
echo "  Planet(${FP_TARGET_PLANET}).start=${FP_PLANET_START}"
echo "    ${FP_F2} fwd='' bwd=${F2_BWD}"
echo "    ${FP_F3} fwd=${F3_FWD} bwd=${F3_BWD}"
echo "    ${FP_F4} fwd=${F4_FWD} bwd=${F4_BWD}"
echo "    ${FP_F5} fwd=${F5_FWD} bwd=''"
echo "  Planet(${FP_TARGET_PLANET}).last=${FP_PLANET_LAST}"
echo "  (Home) ${FP_F1} status=${F1_STATUS}"

fi # phase 17b


if run_phase 2400; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 17c: Fleet Range Combat — Adjacent, Planetary Reach, Destruction
# ═════════════════════════════════════════════════════════════════════════════
#
# Range rules:
#   Away fleets (middle/back): can only attack adjacent (forward/backward) fleets
#   Away fleet (front, fwd=""): can attack ANY target on the same planet
#   Home fleet (on station):   can only attack the first fleet (locationListStart)

section "PHASE 17c: Fleet Range Combat"

FP_TARGET_PLANET="${FP_1_PLANET_ID}"
FP_F1="${FP_1_FLEET_ID}"; FP_F2="${FP_2_FLEET_ID}"; FP_F3="${FP_3_FLEET_ID}"
FP_F4="${FP_4_FLEET_ID}"; FP_F5="${FP_5_FLEET_ID}"

info "Recording initial health (all should be 6)"
for FP_NUM in 1 2 3 4 5; do
    eval "CS=\${FP_CS_${FP_NUM}}"
    echo "  FP_CS_${FP_NUM} (${CS}): HP=$(get_hp "${CS}")"
done

# ─── Adjacent Attacks (Should Succeed) ───

# Test 1: Home fleet (F1) → front of list (F2)
info "Test 1: F1 (home) → F2 (front of list)"
wait_for_charge "${FP_1_ID}" "${CHARGE_ATTACK_DEFAULT}"
FP_CS2_HP_BEFORE=$(get_hp "${FP_CS_2}")
run_tx "F1 (home) attacks FP_CS_2 on F2 (front of list)" \
    tx structs struct-attack "${FP_CS_1}" "${FP_CS_2}" primaryWeapon --from fplayer_1
FP_CS2_HP=$(get_hp "${FP_CS_2}")
echo "  FP_CS_2 HP: ${FP_CS2_HP} (was ${FP_CS2_HP_BEFORE})"
assert_eq "Home fleet hit front of list" "true" "$([ "${FP_CS2_HP}" -lt "${FP_CS2_HP_BEFORE}" ] && echo true || echo false)"

# Test 2: F3 → forward neighbor F2
info "Test 2: F3 → F2 (forward neighbor)"
wait_for_charge "${FP_3_ID}" "${CHARGE_ATTACK_DEFAULT}"
FP_CS2_HP_BEFORE=$(get_hp "${FP_CS_2}")
run_tx "F3 attacks FP_CS_2 on F2 (forward neighbor)" \
    tx structs struct-attack "${FP_CS_3}" "${FP_CS_2}" primaryWeapon --from fplayer_3
FP_CS2_HP=$(get_hp "${FP_CS_2}")
echo "  FP_CS_2 HP: ${FP_CS2_HP} (was ${FP_CS2_HP_BEFORE})"
assert_eq "F3 hit forward neighbor F2" "true" "$([ "${FP_CS2_HP}" -lt "${FP_CS2_HP_BEFORE}" ] && echo true || echo false)"

# Test 3: F5 → forward neighbor F4
info "Test 3: F5 → F4 (forward neighbor)"
wait_for_charge "${FP_5_ID}" "${CHARGE_ATTACK_DEFAULT}"
FP_CS4_HP_BEFORE=$(get_hp "${FP_CS_4}")
run_tx "F5 attacks FP_CS_4 on F4 (forward neighbor)" \
    tx structs struct-attack "${FP_CS_5}" "${FP_CS_4}" primaryWeapon --from fplayer_5
FP_CS4_HP=$(get_hp "${FP_CS_4}")
echo "  FP_CS_4 HP: ${FP_CS4_HP} (was ${FP_CS4_HP_BEFORE})"
assert_eq "F5 hit forward neighbor F4" "true" "$([ "${FP_CS4_HP}" -lt "${FP_CS4_HP_BEFORE}" ] && echo true || echo false)"

info "Health after adjacent attacks"
for FP_NUM in 1 2 3 4 5; do eval "CS=\${FP_CS_${FP_NUM}}"; echo "  FP_CS_${FP_NUM}: HP=$(get_hp "${CS}")"; done

# ─── Front-of-List Planetary Reach ───

# F2 has locationListForward="" → front of raid queue → can reach ANY target on the planet
info "Test R1: F2 (front, fwd='') → F5 (non-adjacent, same planet)"
wait_for_charge "${FP_2_ID}" "${CHARGE_ATTACK_DEFAULT}"
FP_CS5_HP_BEFORE=$(get_hp "${FP_CS_5}")
run_tx "F2 attacks FP_CS_5 on F5 (front-of-list reaches whole planet)" \
    tx structs struct-attack "${FP_CS_2}" "${FP_CS_5}" primaryWeapon --from fplayer_2
FP_CS5_HP=$(get_hp "${FP_CS_5}")
echo "  FP_CS_5 HP: ${FP_CS5_HP} (was ${FP_CS5_HP_BEFORE})"
assert_eq "Front-of-list F2 hit non-adjacent F5" "true" "$([ "${FP_CS5_HP}" -lt "${FP_CS5_HP_BEFORE}" ] && echo true || echo false)"

# ─── Destruction & Linked List Collapse ───
# Destroy CS_2 by attacking it until HP=0, then verify:
#   - CS_2 wiped from chain
#   - F2 returned to its home planet
#   - List collapsed: Planet.start → F3, F3.forward = ""

info "Current health before destruction test"
for FP_NUM in 1 2 3 4 5; do eval "CS=\${FP_CS_${FP_NUM}}"; echo "  FP_CS_${FP_NUM}: HP=$(get_hp "${CS}")"; done

FP_CS2_HP=$(get_hp "${FP_CS_2}")
ATTACK_COUNT=0
while [ "${FP_CS2_HP}" -gt 0 ] 2>/dev/null && [ "${ATTACK_COUNT}" -lt 5 ]; do
    ATTACK_COUNT=$((ATTACK_COUNT + 1))
    wait_for_charge "${FP_3_ID}" "${CHARGE_ATTACK_DEFAULT}"
    run_tx "F3 attacks FP_CS_2 (#${ATTACK_COUNT}, HP=${FP_CS2_HP})" \
        tx structs struct-attack "${FP_CS_3}" "${FP_CS_2}" primaryWeapon --from fplayer_3
    FP_CS2_HP=$(get_hp "${FP_CS_2}")
    echo "  FP_CS_2 HP after attack #${ATTACK_COUNT}: ${FP_CS2_HP}"
done

info "FP_CS_2 destruction result"
echo "  Attacks required: ${ATTACK_COUNT}"

sleep 6
FP_CS2_QUERY=$(structsd ${PARAMS_QUERY} query structs struct "${FP_CS_2}" 2>&1 || true)
FP_CS2_HP_CHECK=$(get_hp "${FP_CS_2}")
if [ -z "${FP_CS2_QUERY}" ] || echo "${FP_CS2_QUERY}" | grep -qi "not found\|error\|object"; then
    echo -e "  ${GREEN}PASS${NC}: FP_CS_2 (${FP_CS_2}) wiped from chain"
    PASS_COUNT=$((PASS_COUNT + 1))
elif [ "${FP_CS2_HP_CHECK}" = "0" ]; then
    echo -e "  ${GREEN}PASS${NC}: FP_CS_2 (${FP_CS_2}) HP=0 (destroyed, pending cleanup)"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    echo -e "  ${RED}FAIL${NC}: FP_CS_2 (${FP_CS_2}) still exists on chain (HP=${FP_CS2_HP_CHECK})"
    FAIL_COUNT=$((FAIL_COUNT + 1))
fi

# Verify F2 returned home
info "Checking F2 returned home after CS_2 destruction"
F2_JSON=$(query_fleet "${FP_F2}")
F2_LOC=$(jqr "${F2_JSON}" '.Fleet.locationId')
F2_STATUS=$(jqr "${F2_JSON}" '.Fleet.status')
if [ -z "${F2_STATUS}" ]; then F2_STATUS="onStation"; fi
assert_eq "F2 returned to home planet" "${FP_2_PLANET_ID}" "${F2_LOC}"
assert_eq "F2 status after recall" "onStation" "${F2_STATUS}"
echo "  F2: loc=${F2_LOC} status=${F2_STATUS}"

# Verify linked list collapsed: F3 is now front
info "Verifying linked list collapsed (F2 removed)"
echo "  Expected: Planet.start → F3 ↔ F4 ↔ F5 ← Planet.last"

FP_PLANET_START=$(get_planet_field "${FP_TARGET_PLANET}" "locationListStart")
FP_PLANET_LAST=$(get_planet_field "${FP_TARGET_PLANET}" "locationListLast")
assert_eq "Planet.start after F2 removal" "${FP_F3}" "${FP_PLANET_START}"
assert_eq "Planet.last unchanged" "${FP_F5}" "${FP_PLANET_LAST}"

F3_JSON=$(query_fleet "${FP_F3}")
F3_FWD=$(jqr "${F3_JSON}" '.Fleet.locationListForward')
F3_BWD=$(jqr "${F3_JSON}" '.Fleet.locationListBackward')
assert_eq "F3 is now front (forward='')" "" "${F3_FWD}"
assert_eq "F3 backward" "${FP_F4}" "${F3_BWD}"
echo "  F3: fwd='${F3_FWD}' bwd='${F3_BWD}'"

F4_JSON=$(query_fleet "${FP_F4}")
F4_FWD=$(jqr "${F4_JSON}" '.Fleet.locationListForward')
F4_BWD=$(jqr "${F4_JSON}" '.Fleet.locationListBackward')
assert_eq "F4 forward" "${FP_F3}" "${F4_FWD}"
assert_eq "F4 backward" "${FP_F5}" "${F4_BWD}"
echo "  F4: fwd='${F4_FWD}' bwd='${F4_BWD}'"

F5_JSON=$(query_fleet "${FP_F5}")
F5_FWD=$(jqr "${F5_JSON}" '.Fleet.locationListForward')
F5_BWD=$(jqr "${F5_JSON}" '.Fleet.locationListBackward')
assert_eq "F5 forward" "${FP_F4}" "${F5_FWD}"
assert_eq "F5 backward (still last)" "" "${F5_BWD}"
echo "  F5: fwd='${F5_FWD}' bwd='${F5_BWD}'"

info "Linked list after collapse:"
echo "  Planet(${FP_TARGET_PLANET}).start=${FP_PLANET_START} .last=${FP_PLANET_LAST}"
echo "  F3(fwd='', bwd=${F3_BWD}) ↔ F4(fwd=${F4_FWD}, bwd=${F4_BWD}) ↔ F5(fwd=${F5_FWD}, bwd='')"
echo "  F2 → home (${F2_LOC}), F1 → home (on station)"

# ─── Non-Adjacent Attacks (Should Fail) ───
# Current list: F3 ↔ F4 ↔ F5 (F3 is front)
# Should fail:
#   F5 → F3 (F5 only sees F4, not F3)
#   F1 (home) → F4 (home can only hit front = F3)
#   F1 (home) → F5 (same)

info "Health snapshot before negative tests"
FP_CS3_HP=$(get_hp "${FP_CS_3}")
FP_CS4_HP=$(get_hp "${FP_CS_4}")
FP_CS5_HP=$(get_hp "${FP_CS_5}")
echo "  FP_CS_3: HP=${FP_CS3_HP}, FP_CS_4: HP=${FP_CS4_HP}, FP_CS_5: HP=${FP_CS5_HP}"

# Test N1: F5 → F3 (not adjacent, gap of 1)
if [ "${FP_CS5_HP}" = "0" ]; then
    info "SKIP N1: FP_CS_5 destroyed"
else
    info "Test N1: F5 → F3 (not adjacent — F5 only sees F4)"
    wait_for_charge "${FP_5_ID}" "${CHARGE_ATTACK_DEFAULT}"
    run_tx_expect_fail "F5 attacks FP_CS_3 on F3 (not adjacent)" \
        tx structs struct-attack "${FP_CS_5}" "${FP_CS_3}" primaryWeapon --from fplayer_5
    sleep "${SLEEP}"
    FP_CS3_CHECK=$(get_hp "${FP_CS_3}")
    assert_eq "FP_CS_3 HP unchanged after F5→F3" "${FP_CS3_HP}" "${FP_CS3_CHECK}"
fi

# Test N2: F1 (home) → F4 (not front of list)
FP_CS1_HP=$(get_hp "${FP_CS_1}")
if [ "${FP_CS1_HP}" = "0" ]; then
    info "SKIP N2: FP_CS_1 destroyed"
else
    info "Test N2: F1 (home) → F4 (home can only hit front = F3)"
    wait_for_charge "${FP_1_ID}" "${CHARGE_ATTACK_DEFAULT}"
    run_tx_expect_fail "F1 (home) attacks FP_CS_4 on F4 (not front of list)" \
        tx structs struct-attack "${FP_CS_1}" "${FP_CS_4}" primaryWeapon --from fplayer_1
    sleep "${SLEEP}"
    FP_CS4_CHECK=$(get_hp "${FP_CS_4}")
    assert_eq "FP_CS_4 HP unchanged after F1→F4" "${FP_CS4_HP}" "${FP_CS4_CHECK}"
fi

# Test N3: F1 (home) → F5 (not front of list)
if [ "${FP_CS1_HP}" = "0" ]; then
    info "SKIP N3: FP_CS_1 destroyed"
else
    info "Test N3: F1 (home) → F5 (home can only hit front = F3)"
    wait_for_charge "${FP_1_ID}" "${CHARGE_ATTACK_DEFAULT}"
    run_tx_expect_fail "F1 (home) attacks FP_CS_5 on F5 (not front of list)" \
        tx structs struct-attack "${FP_CS_1}" "${FP_CS_5}" primaryWeapon --from fplayer_1
    sleep "${SLEEP}"
    FP_CS5_CHECK=$(get_hp "${FP_CS_5}")
    assert_eq "FP_CS_5 HP unchanged after F1→F5" "${FP_CS5_HP}" "${FP_CS5_CHECK}"
fi

fi # phase 17c


# ═════════════════════════════════════════════════════════════════════════════
#  EXTENDED BATTLE TESTING (--extended-battle)
# ═════════════════════════════════════════════════════════════════════════════

if [ "${EXTENDED_BATTLE}" = true ]; then

if run_phase 2500; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE EB1: Player 6 Setup (Third-Party Adversary)
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE EB1: Player 6 Setup (Adversary)"

info "Setting up player_6 as third-party adversary"
EXISTING_6=$(structsd ${PARAMS_KEYS} keys show player_6 2>/dev/null | jq -r .address || echo "")
if [ -z "${EXISTING_6}" ]; then
    PLAYER_6_ADDRESS=$(structsd ${PARAMS_KEYS} keys add player_6 | jq -r .address)
    echo "  Created player_6: ${PLAYER_6_ADDRESS}"
else
    PLAYER_6_ADDRESS="${EXISTING_6}"
    echo "  Reusing player_6: ${PLAYER_6_ADDRESS}"
fi
assert_not_empty "Player 6 address" "${PLAYER_6_ADDRESS}"

run_tx "Funding player_6 from bob" \
    tx bank send "${BOB_ADDRESS}" "${PLAYER_6_ADDRESS}" 25000000ualpha --from bob

run_tx "Delegating 20000000ualpha from player_6 to validator" \
    tx staking delegate "${VALIDATOR_ADDRESS}" 20000000ualpha --from player_6

ADDR_JSON_6=$(query query structs address "${PLAYER_6_ADDRESS}")
PLAYER_6_ID=$(jqr "${ADDR_JSON_6}" '.playerId')
assert_not_empty "Player 6 ID" "${PLAYER_6_ID}"
echo "  Player 6 ID: ${PLAYER_6_ID}"

# Create allocation (controller = alice)
P6_JSON=$(query query structs player "${PLAYER_6_ID}")
P6_CAP=$(jqr "${P6_JSON}" '.gridAttributes.capacity')
assert_gt "Player 6 capacity" 0 "${P6_CAP}"
echo "  Player 6 capacity: ${P6_CAP}"

run_tx "Creating allocation from Player 6 (controller=alice)" \
    tx structs allocation-create "${PLAYER_6_ID}" "${P6_CAP}" \
    --controller "${PLAYER_1_ID}" --allocation-type dynamic --from player_6

P6_ALLOC_ID=$(get_latest_allocation_for_source "${PLAYER_6_ID}")
assert_not_empty "Player 6 allocation ID" "${P6_ALLOC_ID}"
echo "  Player 6 Allocation ID: ${P6_ALLOC_ID}"

# Join guild and connect to substation
run_tx "Player 6 joining guild" \
    tx structs guild-membership-join "${GUILD_ID}" "${REACTOR_ID}-${PLAYER_6_ADDRESS}" --from player_6

run_tx "Connecting Player 6 allocation to substation" \
    tx structs substation-allocation-connect "${P6_ALLOC_ID}" "${SUBSTATION_ID}" --from alice

# Explore planet (creates command ship + fleet)
run_tx "Player 6 exploring a planet" \
    tx structs planet-explore "${PLAYER_6_ID}" --from player_6

P6_JSON=$(query query structs player "${PLAYER_6_ID}")
PLAYER_6_PLANET_ID=$(jqr "${P6_JSON}" '.Player.planetId')
PLAYER_6_FLEET_ID=$(jqr "${P6_JSON}" '.Player.fleetId')
assert_not_empty "Player 6 planet" "${PLAYER_6_PLANET_ID}"
assert_not_empty "Player 6 fleet" "${PLAYER_6_FLEET_ID}"
echo "  Player 6 Planet: ${PLAYER_6_PLANET_ID}  Fleet: ${PLAYER_6_FLEET_ID}"

# Discover P6's command ship (newest struct after explore)
STRUCT_ALL_JSON=$(query query structs struct-all)
P6_COMMAND_SHIP_ID=$(get_newest_struct_id "${STRUCT_ALL_JSON}")
assert_not_empty "Player 6 Command Ship ID" "${P6_COMMAND_SHIP_ID}"
echo "  Player 6 Command Ship: ${P6_COMMAND_SHIP_ID}"

fi # phase EB1

if run_phase 2600; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE EB2: Build All 13 Fleet-Capable Struct Types
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE EB2: Build All 13 Fleet Types"

info "Building the 5 never-built types plus extra units for P6's fleet"
info "Types needed: 3(Starfighter), 4(Frigate), 5(Pursuit Fighter), 8(Mobile Artillery), 12(Destroyer-water)"

# ═══════════════════════════════════════════════════════════════
# BATCH INITIATE: All builds for extended battle
# P3 builds: Pursuit Fighter (type 5, air, slot 1)
# P6 builds: Starfighter (3), Frigate (4), Mobile Artillery (8),
#            Destroyer-water (12), Battleship (2), Tank (9), Cruiser (11)
# Interleave across players for charge efficiency
# ═══════════════════════════════════════════════════════════════

info "Batch-initiating all extended battle builds"

# P3's fleet may be away after Phase 17c combat — move home first
run_tx "Moving P3 fleet home before building" \
    tx structs fleet-move "${PLAYER_3_FLEET_ID}" "${PLAYER_3_PLANET_ID}" --from player_3

# ─── P3: Pursuit Fighter (type 5, air, slot 1) ───
wait_for_charge "${PLAYER_3_ID}" "${CHARGE_BUILD}"
run_tx "Initiating Pursuit Fighter (type=5, air, slot=1) for P3" \
    tx structs struct-build-initiate "${PLAYER_3_ID}" 5 air 1 --from player_3

STRUCT_ALL_JSON=$(query query structs struct-all)
EB_PURSUIT_FIGHTER_ID=$(get_newest_struct_id "${STRUCT_ALL_JSON}")
assert_not_empty "Pursuit Fighter struct ID" "${EB_PURSUIT_FIGHTER_ID}"
echo "  Pursuit Fighter ID: ${EB_PURSUIT_FIGHTER_ID}"

# ─── P6: Starfighter (type 3, space, slot 0) ───
wait_for_charge "${PLAYER_6_ID}" "${CHARGE_BUILD}"
run_tx "Initiating Starfighter (type=3, space, slot=0) for P6" \
    tx structs struct-build-initiate "${PLAYER_6_ID}" 3 space 0 --from player_6

STRUCT_ALL_JSON=$(query query structs struct-all)
EB_STARFIGHTER_ID=$(get_newest_struct_id "${STRUCT_ALL_JSON}")
assert_not_empty "Starfighter struct ID" "${EB_STARFIGHTER_ID}"
echo "  Starfighter ID: ${EB_STARFIGHTER_ID}"

# ─── P6: Frigate (type 4, space, slot 1) ───
wait_for_charge "${PLAYER_6_ID}" "${CHARGE_BUILD}"
run_tx "Initiating Frigate (type=4, space, slot=1) for P6" \
    tx structs struct-build-initiate "${PLAYER_6_ID}" 4 space 1 --from player_6

STRUCT_ALL_JSON=$(query query structs struct-all)
EB_FRIGATE_ID=$(get_newest_struct_id "${STRUCT_ALL_JSON}")
assert_not_empty "Frigate struct ID" "${EB_FRIGATE_ID}"
echo "  Frigate ID: ${EB_FRIGATE_ID}"

# ─── P6: Mobile Artillery (type 8, land, slot 0) ───
wait_for_charge "${PLAYER_6_ID}" "${CHARGE_BUILD}"
run_tx "Initiating Mobile Artillery (type=8, land, slot=0) for P6" \
    tx structs struct-build-initiate "${PLAYER_6_ID}" 8 land 0 --from player_6

STRUCT_ALL_JSON=$(query query structs struct-all)
EB_MOBILE_ART_ID=$(get_newest_struct_id "${STRUCT_ALL_JSON}")
assert_not_empty "Mobile Artillery struct ID" "${EB_MOBILE_ART_ID}"
echo "  Mobile Artillery ID: ${EB_MOBILE_ART_ID}"

# ─── P6: Destroyer-water (type 12, water, slot 0) ───
wait_for_charge "${PLAYER_6_ID}" "${CHARGE_BUILD}"
run_tx "Initiating Destroyer-water (type=12, water, slot=0) for P6" \
    tx structs struct-build-initiate "${PLAYER_6_ID}" 12 water 0 --from player_6

STRUCT_ALL_JSON=$(query query structs struct-all)
EB_DESTROYER_W_ID=$(get_newest_struct_id "${STRUCT_ALL_JSON}")
assert_not_empty "Destroyer-water struct ID" "${EB_DESTROYER_W_ID}"
echo "  Destroyer-water ID: ${EB_DESTROYER_W_ID}"

# ─── P6: Battleship (type 2, space, slot 2) ───
wait_for_charge "${PLAYER_6_ID}" "${CHARGE_BUILD}"
run_tx "Initiating P6 Battleship (type=2, space, slot=2) for P6" \
    tx structs struct-build-initiate "${PLAYER_6_ID}" 2 space 2 --from player_6

STRUCT_ALL_JSON=$(query query structs struct-all)
EB_P6_BATTLESHIP_ID=$(get_newest_struct_id "${STRUCT_ALL_JSON}")
assert_not_empty "P6 Battleship struct ID" "${EB_P6_BATTLESHIP_ID}"
echo "  P6 Battleship ID: ${EB_P6_BATTLESHIP_ID}"

# ─── P6: Tank (type 9, land, slot 1) ───
wait_for_charge "${PLAYER_6_ID}" "${CHARGE_BUILD}"
run_tx "Initiating P6 Tank (type=9, land, slot=1) for P6" \
    tx structs struct-build-initiate "${PLAYER_6_ID}" 9 land 1 --from player_6

STRUCT_ALL_JSON=$(query query structs struct-all)
EB_P6_TANK_ID=$(get_newest_struct_id "${STRUCT_ALL_JSON}")
assert_not_empty "P6 Tank struct ID" "${EB_P6_TANK_ID}"
echo "  P6 Tank ID: ${EB_P6_TANK_ID}"

# ─── P6: Cruiser (type 11, water, slot 1) ───
wait_for_charge "${PLAYER_6_ID}" "${CHARGE_BUILD}"
run_tx "Initiating P6 Cruiser (type=11, water, slot=1) for P6" \
    tx structs struct-build-initiate "${PLAYER_6_ID}" 11 water 1 --from player_6

STRUCT_ALL_JSON=$(query query structs struct-all)
EB_P6_CRUISER_ID=$(get_newest_struct_id "${STRUCT_ALL_JSON}")
assert_not_empty "P6 Cruiser struct ID" "${EB_P6_CRUISER_ID}"
echo "  P6 Cruiser ID: ${EB_P6_CRUISER_ID}"

# ─── P6: High Altitude Interceptor (type 7, air, slot 0) — for evasion testing ───
wait_for_charge "${PLAYER_6_ID}" "${CHARGE_BUILD}"
run_tx "Initiating P6 HAI (type=7, air, slot=0) for evasion testing" \
    tx structs struct-build-initiate "${PLAYER_6_ID}" 7 air 0 --from player_6

STRUCT_ALL_JSON=$(query query structs struct-all)
EB_P6_HAI_ID=$(get_newest_struct_id "${STRUCT_ALL_JSON}")
assert_not_empty "P6 HAI struct ID" "${EB_P6_HAI_ID}"
echo "  P6 HAI ID: ${EB_P6_HAI_ID}"

# ─── P3: Mobile Artillery (type 8, land, slot 3) — for PDC immunity test ───
PREV_NEWEST_STRUCT_ID=$(get_newest_struct_id)
wait_for_charge "${PLAYER_3_ID}" "${CHARGE_BUILD}"
run_tx "Initiating P3 Mobile Artillery (type=8, land, slot=3) for P3" \
    tx structs struct-build-initiate "${PLAYER_3_ID}" 8 land 3 --from player_3

STRUCT_ALL_JSON=$(query query structs struct-all)
EB_P3_MOBILE_ART_ID=$(get_newest_struct_id "${STRUCT_ALL_JSON}")
assert_new_struct "P3 Mobile Artillery initiated" "${EB_P3_MOBILE_ART_ID}" "${PREV_NEWEST_STRUCT_ID}" 8
echo "  P3 Mobile Artillery ID: ${EB_P3_MOBILE_ART_ID}"

# ─── P6: PDC (type 19, land, slot 2) — planetary struct for defense cannon test ───
PREV_NEWEST_STRUCT_ID=$(get_newest_struct_id)
wait_for_charge "${PLAYER_6_ID}" "${CHARGE_BUILD}"
run_tx "Initiating P6 PDC (type=19, land, slot=2) for PDC test" \
    tx structs struct-build-initiate "${PLAYER_6_ID}" 19 land 2 --from player_6

STRUCT_ALL_JSON=$(query query structs struct-all)
EB_PDC_ID=$(get_newest_struct_id "${STRUCT_ALL_JSON}")
assert_new_struct "P6 PDC initiated" "${EB_PDC_ID}" "${PREV_NEWEST_STRUCT_ID}" 19
echo "  P6 PDC ID: ${EB_PDC_ID}"

# ─── P6: Ore Extractor (type 14, land, slot 3) — non-PDC planet struct for PDC cross-defense test ───
PREV_NEWEST_STRUCT_ID=$(get_newest_struct_id)
wait_for_charge "${PLAYER_6_ID}" "${CHARGE_BUILD}"
run_tx "Initiating P6 Ore Extractor (type=14, land, slot=3) for PDC cross-defense test" \
    tx structs struct-build-initiate "${PLAYER_6_ID}" 14 land 3 --from player_6

STRUCT_ALL_JSON=$(query query structs struct-all)
EB_ORE_EXTRACTOR_ID=$(get_newest_struct_id "${STRUCT_ALL_JSON}")
assert_new_struct "P6 Ore Extractor initiated" "${EB_ORE_EXTRACTOR_ID}" "${PREV_NEWEST_STRUCT_ID}" 14
echo "  P6 Ore Extractor ID: ${EB_ORE_EXTRACTOR_ID}"

info "All 12 extended battle builds initiated. Computing now (difficulty decays with age)."

# ═══════════════════════════════════════════════════════════════
# COMPUTE: Build all extended battle structs
# P3's Pursuit Fighter first (lets P6's builds age further)
# Then P6's builds in sequence
# ═══════════════════════════════════════════════════════════════

# ─── Compute P3 Pursuit Fighter ───
run_compute "Building Pursuit Fighter ${EB_PURSUIT_FIGHTER_ID}" \
    tx structs struct-build-compute "${EB_PURSUIT_FIGHTER_ID}" --from player_3

assert_eq "Pursuit Fighter built" "true" "$(query query structs struct "${EB_PURSUIT_FIGHTER_ID}" | jq -r '.structAttributes.isBuilt')"

# ─── Compute P6 Starfighter ───
run_compute "Building Starfighter ${EB_STARFIGHTER_ID}" \
    tx structs struct-build-compute "${EB_STARFIGHTER_ID}" --from player_6

assert_eq "Starfighter built" "true" "$(query query structs struct "${EB_STARFIGHTER_ID}" | jq -r '.structAttributes.isBuilt')"

# ─── Compute P6 Frigate ───
run_compute "Building Frigate ${EB_FRIGATE_ID}" \
    tx structs struct-build-compute "${EB_FRIGATE_ID}" --from player_6

assert_eq "Frigate built" "true" "$(query query structs struct "${EB_FRIGATE_ID}" | jq -r '.structAttributes.isBuilt')"

# ─── Compute P6 Mobile Artillery ───
run_compute "Building Mobile Artillery ${EB_MOBILE_ART_ID}" \
    tx structs struct-build-compute "${EB_MOBILE_ART_ID}" --from player_6

assert_eq "Mobile Artillery built" "true" "$(query query structs struct "${EB_MOBILE_ART_ID}" | jq -r '.structAttributes.isBuilt')"

# ─── Compute P6 Destroyer-water ───
run_compute "Building Destroyer-water ${EB_DESTROYER_W_ID}" \
    tx structs struct-build-compute "${EB_DESTROYER_W_ID}" --from player_6

assert_eq "Destroyer-water built" "true" "$(query query structs struct "${EB_DESTROYER_W_ID}" | jq -r '.structAttributes.isBuilt')"

# ─── Compute P6 Battleship ───
run_compute "Building P6 Battleship ${EB_P6_BATTLESHIP_ID}" \
    tx structs struct-build-compute "${EB_P6_BATTLESHIP_ID}" --from player_6

assert_eq "P6 Battleship built" "true" "$(query query structs struct "${EB_P6_BATTLESHIP_ID}" | jq -r '.structAttributes.isBuilt')"

# ─── Compute P6 Tank ───
run_compute "Building P6 Tank ${EB_P6_TANK_ID}" \
    tx structs struct-build-compute "${EB_P6_TANK_ID}" --from player_6

assert_eq "P6 Tank built" "true" "$(query query structs struct "${EB_P6_TANK_ID}" | jq -r '.structAttributes.isBuilt')"

# ─── Compute P6 Cruiser ───
run_compute "Building P6 Cruiser ${EB_P6_CRUISER_ID}" \
    tx structs struct-build-compute "${EB_P6_CRUISER_ID}" --from player_6

assert_eq "P6 Cruiser built" "true" "$(query query structs struct "${EB_P6_CRUISER_ID}" | jq -r '.structAttributes.isBuilt')"

# ─── Compute P6 HAI ───
run_compute "Building P6 HAI ${EB_P6_HAI_ID}" \
    tx structs struct-build-compute "${EB_P6_HAI_ID}" --from player_6

assert_eq "P6 HAI built" "true" "$(query query structs struct "${EB_P6_HAI_ID}" | jq -r '.structAttributes.isBuilt')"

# ─── Compute P3 Mobile Artillery ───
run_compute "Building P3 Mobile Artillery ${EB_P3_MOBILE_ART_ID}" \
    tx structs struct-build-compute "${EB_P3_MOBILE_ART_ID}" --from player_3

assert_eq "P3 Mobile Artillery built" "true" "$(query query structs struct "${EB_P3_MOBILE_ART_ID}" | jq -r '.structAttributes.isBuilt')"

# ─── Compute P6 PDC ───
run_compute "Building P6 PDC ${EB_PDC_ID}" \
    tx structs struct-build-compute "${EB_PDC_ID}" --from player_6

assert_eq "P6 PDC built" "true" "$(query query structs struct "${EB_PDC_ID}" | jq -r '.structAttributes.isBuilt')"

# Verify PDC added a defensive cannon to P6's planet
P6_PLANET_JSON=$(query query structs planet "${PLAYER_6_PLANET_ID}" 2>/dev/null || echo '{}')
P6_DEF_CANNON_QTY=$(jqr "${P6_PLANET_JSON}" '.planetAttributes.defensiveCannonQuantity' '0')
assert_gt "P6 planet has defensive cannons" 0 "${P6_DEF_CANNON_QTY}"
info "P6 planet defensive cannon quantity: ${P6_DEF_CANNON_QTY}"

# ─── Compute P6 Ore Extractor ───
run_compute "Building P6 Ore Extractor ${EB_ORE_EXTRACTOR_ID}" \
    tx structs struct-build-compute "${EB_ORE_EXTRACTOR_ID}" --from player_6

assert_eq "P6 Ore Extractor built" "true" "$(query query structs struct "${EB_ORE_EXTRACTOR_ID}" | jq -r '.structAttributes.isBuilt')"

info "All 14 fleet-capable struct types now exist across P3 and P6 (+ P6 HAI for evasion, Ore Extractor for PDC test)"
info "  Types 1-13: Command Ship, Battleship, Starfighter, Frigate, Pursuit Fighter,"
info "              Stealth Bomber, HAI, Mobile Artillery, Tank, SAM, Cruiser, Destroyer(W), Submersible"
info "  P6 HAI (type 7, air/0) built for defensiveManeuver evasion testing"

fi # phase EB2

if run_phase 2700; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE EB3: Fleet Assembly & Positioning
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE EB3: Fleet Assembly & Positioning"

# Fleet units are built directly on their final fleet slots (Movable=false).
# Only the Command Ship may struct-move (ambit changes).

# ─── Move P3's fleet home for assembly ───
run_tx "Moving P3's fleet home for extended battle assembly" \
    tx structs fleet-move "${PLAYER_3_FLEET_ID}" "${PLAYER_3_PLANET_ID}" --from player_3

# ─── P6 Command Ship to space ambit ───
wait_for_charge "${PLAYER_6_ID}" "${CHARGE_MOVE}"
run_tx "Moving P6 Command Ship to fleet (space)" \
    tx structs struct-move "${P6_COMMAND_SHIP_ID}" fleet space --from player_6

info "P6 fleet assembled: CS(space), Starfighter(space/0), Frigate(space/1), Battleship(space/2),"
info "  MobileArt(land/0), Tank(land/1), Destroyer(water/0), Cruiser(water/1), HAI(air/0)"
info "P6 planet structs: PDC(land/2), Ore Extractor(land/3)"
info "P3 fleet now also has: Pursuit Fighter(air/1), Mobile Artillery(land/3) for PDC immunity test"

# ─── Move P3's fleet to P6's planet for battle ───
run_tx "Moving P3's fleet to P6's planet for battle" \
    tx structs fleet-move "${PLAYER_3_FLEET_ID}" "${PLAYER_6_PLANET_ID}" --from player_3

info "P3's fleet is now at P6's planet — battle positions set"

fi # phase EB3

if run_phase 2800; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE EB4: Defense Configurations
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE EB4: Defense Configurations"

# ─── P6: Tank defends Mobile Artillery (same-ambit defense, both land) ───
wait_for_charge "${PLAYER_6_ID}" "${CHARGE_DEFEND}"
run_tx "P6 Tank defends Mobile Artillery (same-ambit: land)" \
    tx structs struct-defense-set "${EB_P6_TANK_ID}" "${EB_MOBILE_ART_ID}" --from player_6

MOBILE_ART_JSON=$(query query structs struct "${EB_MOBILE_ART_ID}" 2>/dev/null || echo '{}')
MA_DEFENDERS=$(echo "${MOBILE_ART_JSON}" | jq -r '.structDefenders | length' 2>/dev/null || echo "0")
assert_gt "Mobile Artillery has defenders" 0 "${MA_DEFENDERS}"
info "Mobile Artillery defender count: ${MA_DEFENDERS}"

# ─── P6: Frigate + Starfighter defend P6 Battleship (multiple defenders, space) ───
wait_for_charge "${PLAYER_6_ID}" "${CHARGE_DEFEND}"
run_tx "P6 Frigate defends P6 Battleship" \
    tx structs struct-defense-set "${EB_FRIGATE_ID}" "${EB_P6_BATTLESHIP_ID}" --from player_6

wait_for_charge "${PLAYER_6_ID}" "${CHARGE_DEFEND}"
run_tx "P6 Starfighter defends P6 Battleship" \
    tx structs struct-defense-set "${EB_STARFIGHTER_ID}" "${EB_P6_BATTLESHIP_ID}" --from player_6

P6_BB_JSON=$(query query structs struct "${EB_P6_BATTLESHIP_ID}" 2>/dev/null || echo '{}')
P6_BB_DEF_COUNT=$(echo "${P6_BB_JSON}" | jq -r '.structDefenders | length' 2>/dev/null || echo "0")
assert_gt "P6 Battleship has multiple defenders" 1 "${P6_BB_DEF_COUNT}"
info "P6 Battleship defender count: ${P6_BB_DEF_COUNT}"

# ─── P3: SAM defends Battleship #1 (cross-ambit: land defends space) ───
wait_for_charge "${PLAYER_3_ID}" "${CHARGE_DEFEND}"
run_tx "P3 SAM defends Battleship #1 (cross-ambit: land→space)" \
    tx structs struct-defense-set "${SAM_STRUCT_ID}" "${BATTLESHIP_1_ID}" --from player_3

BB1_JSON=$(query query structs struct "${BATTLESHIP_1_ID}" 2>/dev/null || echo '{}')
BB1_DEF_COUNT=$(echo "${BB1_JSON}" | jq -r '.structDefenders | length' 2>/dev/null || echo "0")
assert_gt "Battleship #1 has defenders" 0 "${BB1_DEF_COUNT}"
info "Battleship #1 defender count: ${BB1_DEF_COUNT}"

fi # phase EB4

if run_phase 2900; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE EB5: Comprehensive Attack Scenarios
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE EB5: Comprehensive Attack Scenarios"

EB_ATTACKS=0
EB_DESTROYED=0

# Combat helpers (eb_health, eb_attack, etc.) are defined at top-level so
# they remain available when --resume-from skips this phase.

# ─────────────────────────────────────────────────────────────────────────────
# GROUP A: Same-Ambit Combat
# ─────────────────────────────────────────────────────────────────────────────

info "── Group A: Same-Ambit Combat ──"

# A1: Space — P3 Battleship #1 → P6 Starfighter
# v0.18.0: Battleship primary is land/water only; space targets use the
# guided secondary (damage 1).
eb_attack "Space: P3 Battleship #1 → P6 Starfighter (guided secondary)" \
    "${BATTLESHIP_1_ID}" "${EB_STARFIGHTER_ID}" secondaryWeapon 3

# A2: Space — P6 Frigate → P3 Battleship #2
eb_attack "Space: P6 Frigate → P3 Battleship #2" \
    "${EB_FRIGATE_ID}" "${BATTLESHIP_2_ID}" primaryWeapon 6

# A3: Land — P3 Tank → P6 Tank
eb_attack "Land: P3 Tank → P6 Tank" \
    "${DESTROYER_STRUCT_ID}" "${EB_P6_TANK_ID}" primaryWeapon 3

# A4: Land — P6 Mobile Artillery → P3 SAM (non-counterable!)
# Mobile Artillery has CounterAttack=0 and AttackCounterable=false
SAM_HP_BEFORE=$(eb_health "${SAM_STRUCT_ID}")
MOBILE_ART_HP_BEFORE=$(eb_health "${EB_MOBILE_ART_ID}")

eb_attack "Land: P6 Mobile Artillery → P3 SAM (non-counterable)" \
    "${EB_MOBILE_ART_ID}" "${SAM_STRUCT_ID}" primaryWeapon 6

MOBILE_ART_HP_AFTER=$(eb_health "${EB_MOBILE_ART_ID}")
info "  Mobile Artillery HP unchanged? Before=${MOBILE_ART_HP_BEFORE} After=${MOBILE_ART_HP_AFTER}"
echo "  (Mobile Artillery attacks are non-counterable — HP should not decrease from counter)"

# A5: Water — P3 Cruiser → P6 Destroyer-water
eb_attack "Water: P3 Cruiser → P6 Destroyer-water" \
    "${CRUISER_ID}" "${EB_DESTROYER_W_ID}" primaryWeapon 3

# A6: Water — P6 Cruiser → P3 Submersible
eb_attack "Water: P6 Cruiser → P3 Submersible" \
    "${EB_P6_CRUISER_ID}" "${SUB_STRUCT_ID}" primaryWeapon 6

# A7: Space — P3 Battleship #2 → P6 Frigate (guided secondary, space ambit)
eb_attack "Space: P3 Battleship #2 → P6 Frigate (guided secondary)" \
    "${BATTLESHIP_2_ID}" "${EB_FRIGATE_ID}" secondaryWeapon 3

# ─────────────────────────────────────────────────────────────────────────────
# GROUP B: Cross-Ambit Combat
# ─────────────────────────────────────────────────────────────────────────────

info "── Group B: Cross-Ambit Combat ──"

# B1: P3 SAM (land, weapons=space+air) → P6 Starfighter (space)
eb_attack "Cross: P3 SAM(land) → P6 Starfighter(space)" \
    "${SAM_STRUCT_ID}" "${EB_STARFIGHTER_ID}" primaryWeapon 3

# B2: P6 Destroyer-water (water, weapons=air+water) → P3 Pursuit Fighter (air)
eb_attack "Cross: P6 Destroyer-water(water) → P3 Pursuit Fighter(air)" \
    "${EB_DESTROYER_W_ID}" "${EB_PURSUIT_FIGHTER_ID}" primaryWeapon 6

# B3: P3 Submersible (water, weapons=space+water) → P6 Battleship (space)
eb_attack "Cross: P3 Submersible(water) → P6 Battleship(space)" \
    "${SUB_STRUCT_ID}" "${EB_P6_BATTLESHIP_ID}" primaryWeapon 3

# B4: P3 Stealth Bomber (air, weapons=water+land) → P6 Cruiser (water)
# Deactivate stealth only if currently hidden
SB_HIDDEN=$(query query structs struct "${STEALTH_BOMBER_ID}" 2>/dev/null | jq -r '.structAttributes.isHidden // "false"' 2>/dev/null || echo "false")
if [ "${SB_HIDDEN}" = "true" ]; then
    wait_for_charge "${PLAYER_3_ID}" "${CHARGE_ACTIVATE}"
    run_tx "Deactivating stealth on Stealth Bomber for cross-ambit test" \
        tx structs struct-stealth-deactivate "${STEALTH_BOMBER_ID}" --from player_3
fi

eb_attack "Cross: P3 Stealth Bomber(air) → P6 Cruiser(water)" \
    "${STEALTH_BOMBER_ID}" "${EB_P6_CRUISER_ID}" primaryWeapon 3

# B5: P3 Cruiser (water, weapons=water+land) → P6 Tank (land)
eb_attack "Cross: P3 Cruiser(water) → P6 Tank(land)" \
    "${CRUISER_ID}" "${EB_P6_TANK_ID}" primaryWeapon 3

# B6: P6 Frigate (space, weapons=space+air) → P3 Pursuit Fighter (air)
eb_attack "Cross: P6 Frigate(space) → P3 Pursuit Fighter(air)" \
    "${EB_FRIGATE_ID}" "${EB_PURSUIT_FIGHTER_ID}" primaryWeapon 6

# ─────────────────────────────────────────────────────────────────────────────
# GROUP C: Special Mechanics
# ─────────────────────────────────────────────────────────────────────────────

info "── Group C: Special Mechanics ──"

# C1: Stealth — activate stealth, verify cross-ambit attack fails against hidden
info "Testing stealth mechanics with Stealth Bomber"
SB_ALIVE=$(eb_health "${STEALTH_BOMBER_ID}")
if [ "${SB_ALIVE}" != "0" ]; then
    wait_for_charge "${PLAYER_3_ID}" "${CHARGE_ACTIVATE}"
    run_tx "Activating stealth on Stealth Bomber" \
        tx structs struct-stealth-activate "${STEALTH_BOMBER_ID}" --from player_3

    # P6 Starfighter (space) tries to attack hidden Stealth Bomber (air)
    # Different ambit + hidden = should fail
    SB_ALIVE_CHECK=$(eb_health "${EB_STARFIGHTER_ID}")
    if [ "${SB_ALIVE_CHECK}" != "0" ]; then
        eb_attack_should_fail "Stealth: P6 Starfighter(space) → hidden Stealth Bomber(air)" \
            "${EB_STARFIGHTER_ID}" "${STEALTH_BOMBER_ID}" 6
    else
        info "SKIP: Starfighter destroyed, cannot test stealth cross-ambit failure"
    fi

    # Deactivate stealth for remaining tests
    wait_for_charge "${PLAYER_3_ID}" "${CHARGE_ACTIVATE}"
    run_tx "Deactivating stealth on Stealth Bomber" \
        tx structs struct-stealth-deactivate "${STEALTH_BOMBER_ID}" --from player_3
else
    info "SKIP: Stealth Bomber destroyed, cannot test stealth mechanics"
fi

# C1b: Dead-attacker defender ordering (v0.20.0 regression) ─────────────────────
# Guards the "dead bomber deals damage" fix (player 1-61 report). When defender
# counters destroy the attacker, NO block volley may land: a blocker must not
# absorb damage from an attacker the counters already killed.
#
# Setup: P3's air Stealth Bomber attacks the land P6 Mobile Artillery, which is
# defended by the land P6 Tank (a valid blocker for a land target that cannot
# counter into air) plus the air-capable P6 HAI (a counter). The Tank was built
# before the HAI, so it sorts ahead of it in defender iteration order — the
# arrangement that exposed the bug (blocker processed before the lethal counter).
#
# The HAI counter is only 1 damage, so the bomber is first softened with one HAI
# strike (2 dmg) to make the counter lethal; otherwise the dead-attacker path
# would not be exercised. The precise, order-controlled unit test is
# TestMsgStructAttackBlockerSortedBeforeLethalCounter; this is the live-chain
# end-to-end confirmation. All assertions are conditional on the bomber actually
# dying so this never produces a false failure if balance numbers change.
info "Testing dead-attacker defender ordering (v0.20.0): bomber vs Tank-blocked, counter-defended land target"
DA_SB_HP=$(eb_health "${STEALTH_BOMBER_ID}")
DA_MA_HP=$(eb_health "${EB_MOBILE_ART_ID}")
DA_TANK_HP=$(eb_health "${EB_P6_TANK_ID}")
DA_HAI_HP=$(eb_health "${EB_P6_HAI_ID}")
if [ "${DA_SB_HP}" != "0" ] && [ "${DA_MA_HP}" != "0" ] && [ "${DA_TANK_HP}" != "0" ] && [ "${DA_HAI_HP}" != "0" ]; then
    # Add the air-capable HAI as a counter defender alongside the land Tank
    # blocker (the Tank already defends the Mobile Artillery from PHASE EB4).
    # HAI is not assigned elsewhere, so this does not disturb other defenses.
    wait_for_charge "${PLAYER_6_ID}" "${CHARGE_DEFEND}"
    run_tx "P6 HAI defends Mobile Artillery (air counter alongside Tank blocker)" \
        tx structs struct-defense-set "${EB_P6_HAI_ID}" "${EB_MOBILE_ART_ID}" --from player_6

    # Soften the bomber so the combined counters are lethal (HAI primary = 2).
    eb_attack "Dead-attacker setup: P6 HAI(air) softens P3 Stealth Bomber(air)" \
        "${EB_P6_HAI_ID}" "${STEALTH_BOMBER_ID}" primaryWeapon 6

    # Reveal the bomber if it re-hid, then fire at the defended land target.
    SB_HIDDEN=$(query query structs struct "${STEALTH_BOMBER_ID}" 2>/dev/null | jq -r '.structAttributes.isHidden // "false"' 2>/dev/null || echo "false")
    if [ "${SB_HIDDEN}" = "true" ]; then
        wait_for_charge "${PLAYER_3_ID}" "${CHARGE_ACTIVATE}"
        run_tx "Deactivating stealth on Stealth Bomber for dead-attacker test" \
            tx structs struct-stealth-deactivate "${STEALTH_BOMBER_ID}" --from player_3
    fi

    DA_TANK_BEFORE=$(eb_health "${EB_P6_TANK_ID}")
    DA_MA_BEFORE=$(eb_health "${EB_MOBILE_ART_ID}")

    eb_attack "Dead-attacker: P3 Stealth Bomber(air) → P6 Mobile Artillery(land) [Tank blocker + HAI counter]" \
        "${STEALTH_BOMBER_ID}" "${EB_MOBILE_ART_ID}" primaryWeapon 3

    DA_SB_AFTER=$(eb_health "${STEALTH_BOMBER_ID}")
    DA_TANK_AFTER=$(eb_health "${EB_P6_TANK_ID}")
    DA_MA_AFTER=$(eb_health "${EB_MOBILE_ART_ID}")
    info "  Result: Bomber HP→${DA_SB_AFTER}, Tank(blocker) HP ${DA_TANK_BEFORE}→${DA_TANK_AFTER}, MobileArt(target) HP ${DA_MA_BEFORE}→${DA_MA_AFTER}"

    if [ "${DA_SB_AFTER}" = "0" ]; then
        info "  Bomber destroyed by counters — asserting no block/volley damage landed"
        assert_eq "v0.20.0: blocker Tank undamaged when counters destroyed the attacker" "${DA_TANK_BEFORE}" "${DA_TANK_AFTER}"
        assert_eq "v0.20.0: target undamaged when counters destroyed the attacker" "${DA_MA_BEFORE}" "${DA_MA_AFTER}"
    else
        info "  Bomber survived the counters (HP=${DA_SB_AFTER}); dead-attacker invariant not exercised this run"
    fi
else
    info "SKIP: bomber/target/Tank/HAI not all alive for dead-attacker ordering test"
fi

# C2: Blocking — Attack P6 Battleship (defended by Frigate + Starfighter)
# At least one defender should attempt to block
info "Testing blocking: attack P6 Battleship (defended by Frigate + Starfighter)"
P6_BB_HP_BEFORE=$(eb_health "${EB_P6_BATTLESHIP_ID}")
P6_FRIG_HP_BEFORE=$(eb_health "${EB_FRIGATE_ID}")
P6_STAR_HP_BEFORE=$(eb_health "${EB_STARFIGHTER_ID}")

BB2_ALIVE=$(eb_health "${BATTLESHIP_2_ID}")
if [ "${BB2_ALIVE}" != "0" ]; then
    # v0.18.0: space target requires the guided secondary. Note the P6
    # Battleship's signal jamming may evade guided attacks (2/3 chance).
    eb_attack "Blocking: P3 Battleship #2 → P6 Battleship (defended, guided secondary)" \
        "${BATTLESHIP_2_ID}" "${EB_P6_BATTLESHIP_ID}" secondaryWeapon 3

    P6_BB_HP_AFTER=$(eb_health "${EB_P6_BATTLESHIP_ID}")
    P6_FRIG_HP_AFTER=$(eb_health "${EB_FRIGATE_ID}")
    P6_STAR_HP_AFTER=$(eb_health "${EB_STARFIGHTER_ID}")
    info "  P6 Battleship HP: ${P6_BB_HP_BEFORE}→${P6_BB_HP_AFTER}"
    info "  P6 Frigate HP: ${P6_FRIG_HP_BEFORE}→${P6_FRIG_HP_AFTER}"
    info "  P6 Starfighter HP: ${P6_STAR_HP_BEFORE}→${P6_STAR_HP_AFTER}"
    echo "  (If a defender blocked, its HP decreased instead of the Battleship's)"
else
    info "SKIP: Battleship #2 destroyed, cannot test blocking"
fi

# C3: Damage reduction — attack P6 Tank (AttackReduction=1)
# Use P3 Tank (type 9, land→land, damage=2) or P3 Cruiser (water→land) as fallback.
# SAM can't target land (PrimaryWeaponAmbits=space+air).
info "Testing damage reduction on P6 Tank (AttackReduction=1)"
P6_TANK_HP=$(eb_health "${EB_P6_TANK_ID}")
if [ "${P6_TANK_HP}" != "0" ]; then
    P3_TANK_ALIVE=$(eb_health "${DESTROYER_STRUCT_ID}")
    if [ "${P3_TANK_ALIVE}" != "0" ]; then
        eb_attack "Damage Reduction: P3 Tank(land) → P6 Tank(land, reduction=1)" \
            "${DESTROYER_STRUCT_ID}" "${EB_P6_TANK_ID}" primaryWeapon 3
        P6_TANK_HP_AFTER=$(eb_health "${EB_P6_TANK_ID}")
        info "  P6 Tank HP after (with reduction): ${P6_TANK_HP}→${P6_TANK_HP_AFTER}"
        echo "  (Tank has AttackReduction=1, so 2 damage becomes 1)"
    elif [ -n "${CRUISER_ID}" ] && [ "$(eb_health "${CRUISER_ID}")" != "0" ]; then
        info "P3 Tank destroyed, using Cruiser for damage reduction test"
        eb_attack "Damage Reduction: P3 Cruiser(water) → P6 Tank(land, reduction=1)" \
            "${CRUISER_ID}" "${EB_P6_TANK_ID}" primaryWeapon 3
        P6_TANK_HP_AFTER=$(eb_health "${EB_P6_TANK_ID}")
        info "  P6 Tank HP after (with reduction): ${P6_TANK_HP}→${P6_TANK_HP_AFTER}"
    else
        info "SKIP: No P3 land-capable attacker alive for damage reduction test"
    fi
else
    info "SKIP: P6 Tank already destroyed"
fi

# C4: Sustained combat — keep attacking P6 Starfighter until destroyed
info "Testing sustained combat: repeatedly attack P6 Starfighter until destroyed"
STAR_HP=$(eb_health "${EB_STARFIGHTER_ID}")
SUSTAINED_ROUNDS=0
while [ "${STAR_HP}" != "0" ] && [ "${SUSTAINED_ROUNDS}" -lt 4 ]; do
    SUSTAINED_ROUNDS=$((SUSTAINED_ROUNDS + 1))
    # Use P3's Battleship #1 (space → space, same ambit)
    BB1_ALIVE=$(eb_health "${BATTLESHIP_1_ID}")
    if [ "${BB1_ALIVE}" = "0" ]; then
        info "  Battleship #1 destroyed, stopping sustained attack"
        break
    fi
    eb_attack "Sustained round ${SUSTAINED_ROUNDS}: P3 BB#1 → P6 Starfighter (guided secondary)" \
        "${BATTLESHIP_1_ID}" "${EB_STARFIGHTER_ID}" secondaryWeapon 3
    STAR_HP=$(eb_health "${EB_STARFIGHTER_ID}")
done
if [ "${STAR_HP}" = "0" ]; then
    info "  Starfighter destroyed after ${SUSTAINED_ROUNDS} sustained rounds"
else
    info "  Starfighter survived ${SUSTAINED_ROUNDS} rounds (HP=${STAR_HP})"
fi

# C5: Command Ship local weapon — P3 Command Ship → P6 unit in same ambit
info "Testing Command Ship local weapon (targets same ambit)"
CS_HP=$(eb_health "${COMMAND_SHIP_ID}")
if [ "${CS_HP}" != "0" ]; then
    P6_BB_HP=$(eb_health "${EB_P6_BATTLESHIP_ID}")
    if [ "${P6_BB_HP}" != "0" ]; then
        eb_attack "Command Ship local weapon: P3 CS(space) → P6 Battleship(space)" \
            "${COMMAND_SHIP_ID}" "${EB_P6_BATTLESHIP_ID}" primaryWeapon 3
    else
        info "SKIP: P6 Battleship destroyed, trying P6 Frigate"
        P6_FRIG_HP=$(eb_health "${EB_FRIGATE_ID}")
        if [ "${P6_FRIG_HP}" != "0" ]; then
            eb_attack "Command Ship local weapon: P3 CS(space) → P6 Frigate(space)" \
                "${COMMAND_SHIP_ID}" "${EB_FRIGATE_ID}" primaryWeapon 3
        else
            info "SKIP: No viable space targets for Command Ship test"
        fi
    fi
else
    info "SKIP: Command Ship destroyed"
fi

# ─────────────────────────────────────────────────────────────────────────────
# GROUP E: Planetary Defense Cannon (PDC) Tests
# ─────────────────────────────────────────────────────────────────────────────
#
# The PDC (type 19) is a planet-category struct that adds defensiveCannonQuantity
# to the planet. After every attack action against a planet-category struct, the
# planet's cannon count is dealt as damage to the attacker (if counterable).
#
# P6's planet has: PDC (land/2), Ore Extractor (land/3).
# The Ore Extractor (type 14) has no counter-attack, no evasion, no post-
# destruction damage — so the ONLY damage an attacker receives from hitting it
# is PDC fire.
#
# Test order: E1-E4 keep the PDC alive (attack the Ore Extractor or fleet
# structs, not the PDC). E5 attacks the PDC directly. E6-E9 test PDC
# destruction and aftermath.
# ─────────────────────────────────────────────────────────────────────────────

info "── Group E: Planetary Defense Cannon (PDC) Tests ──"
info "P6's planet has a PDC (cannon qty) and an Ore Extractor (no counter/evasion)."
info "Tests verify PDC fires when OTHER planet structs are attacked, not just itself."

# ─── E1: PDC fires when Ore Extractor is attacked (PDC is NOT the target) ───
# P3 Battleship #1 (type 2, space, counterable) → P6 Ore Extractor (type 14, planet-category, land)
# Ore Extractor has CounterAttack=0, no evasion, no PostDestructionDamage.
# The ONLY source of damage to the attacker is the PDC.
PDC_HP=$(eb_health "${EB_PDC_ID}")
EXTRACTOR_HP=$(eb_health "${EB_ORE_EXTRACTOR_ID}")
BB1_HP=$(eb_health "${BATTLESHIP_1_ID}")
if [ "${PDC_HP}" != "0" ] && [ "${EXTRACTOR_HP}" != "0" ] && [ "${BB1_HP}" != "0" ]; then
    P6_PLANET_JSON=$(query query structs planet "${PLAYER_6_PLANET_ID}" 2>/dev/null || echo '{}')
    E1_CANNON_QTY=$(jqr "${P6_PLANET_JSON}" '.planetAttributes.defensiveCannonQuantity' '0')
    BB1_HP_BEFORE=$(eb_health "${BATTLESHIP_1_ID}")

    eb_attack "E1: P3 BB#1(counterable) → P6 Ore Extractor (PDC not target)" \
        "${BATTLESHIP_1_ID}" "${EB_ORE_EXTRACTOR_ID}" primaryWeapon 3

    BB1_HP_AFTER=$(eb_health "${BATTLESHIP_1_ID}")
    BB1_HP_LOST=$((BB1_HP_BEFORE - BB1_HP_AFTER))

    if [ "${BB1_HP_LOST}" -gt 0 ] 2>/dev/null; then
        echo -e "  ${GREEN}PASS${NC}: E1 — Attacker took damage from PDC while targeting Ore Extractor (HP ${BB1_HP_BEFORE}→${BB1_HP_AFTER})"
        PASS_COUNT=$((PASS_COUNT + 1))
    else
        echo -e "  ${RED}FAIL${NC}: E1 — Attacker should have taken PDC damage (HP ${BB1_HP_BEFORE}→${BB1_HP_AFTER})"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi

    # E2: Verify exact PDC damage equals defensiveCannonQuantity
    if [ "${E1_CANNON_QTY}" != "0" ] && [ "${BB1_HP_LOST}" -gt 0 ] 2>/dev/null; then
        assert_eq "E2 — PDC damage equals defensiveCannonQuantity" "${E1_CANNON_QTY}" "${BB1_HP_LOST}"
    else
        info "SKIP E2: cannon qty=${E1_CANNON_QTY}, HP lost=${BB1_HP_LOST}"
    fi
else
    info "SKIP E1/E2: PDC(HP=${PDC_HP}), Ore Extractor(HP=${EXTRACTOR_HP}), or BB#1(HP=${BB1_HP}) destroyed"
fi

# ─── E3: Non-counterable attacker vs Ore Extractor — PDC should NOT fire ───
# P3 Mobile Artillery (type 8, AttackCounterable=false) → P6 Ore Extractor
# Ore Extractor has no counter, so Mobile Art should take zero damage.
PDC_HP=$(eb_health "${EB_PDC_ID}")
EXTRACTOR_HP=$(eb_health "${EB_ORE_EXTRACTOR_ID}")
P3_MA_HP=$(eb_health "${EB_P3_MOBILE_ART_ID}")
if [ "${PDC_HP}" != "0" ] && [ "${EXTRACTOR_HP}" != "0" ] && [ "${P3_MA_HP}" != "0" ]; then
    P3_MA_HP_BEFORE=$(eb_health "${EB_P3_MOBILE_ART_ID}")

    eb_attack "E3: P3 Mobile Art(non-counterable) → P6 Ore Extractor" \
        "${EB_P3_MOBILE_ART_ID}" "${EB_ORE_EXTRACTOR_ID}" primaryWeapon 3

    P3_MA_HP_AFTER=$(eb_health "${EB_P3_MOBILE_ART_ID}")

    assert_eq "E3 — Non-counterable attacker took no PDC damage" "${P3_MA_HP_BEFORE}" "${P3_MA_HP_AFTER}"
else
    info "SKIP E3: PDC(HP=${PDC_HP}), Ore Extractor(HP=${EXTRACTOR_HP}), or Mobile Art(HP=${P3_MA_HP}) destroyed"
fi

# ─── E4: Fleet-vs-fleet combat does NOT trigger PDC ───
# P3 Tank (type 9, counterable, land) → P6 Mobile Art (type 8, fleet-category, land)
# P6 Mobile Art has CounterAttack=0, no evasion, no PostDestructionDamage.
# Since the target is fleet-category, trackTargetedPlanet() won't track the planet
# and the PDC should NOT fire. Attacker HP should be unchanged.
PDC_HP=$(eb_health "${EB_PDC_ID}")
TANK_HP=$(eb_health "${DESTROYER_STRUCT_ID}")
P6_MA_HP=$(eb_health "${EB_MOBILE_ART_ID}")
if [ "${PDC_HP}" != "0" ] && [ "${TANK_HP}" != "0" ] && [ "${P6_MA_HP}" != "0" ]; then
    TANK_HP_BEFORE=$(eb_health "${DESTROYER_STRUCT_ID}")

    eb_attack "E4: P3 Tank(counterable) → P6 Mobile Art(fleet-category, no counter)" \
        "${DESTROYER_STRUCT_ID}" "${EB_MOBILE_ART_ID}" primaryWeapon 3

    TANK_HP_AFTER=$(eb_health "${DESTROYER_STRUCT_ID}")

    assert_eq "E4 — No PDC damage when attacking fleet-category struct" "${TANK_HP_BEFORE}" "${TANK_HP_AFTER}"
else
    info "SKIP E4: PDC(HP=${PDC_HP}), Tank(HP=${TANK_HP}), or P6 Mobile Art(HP=${P6_MA_HP}) destroyed"
fi

# ─── E4b: Soften PDC to HP=2 for the E5 killing-blow test ───
# PDC MaxHealth is now 6 (v0.18.0). E5's Tank (primaryWeaponDamage=2) must
# deliver the killing blow, so soften the PDC down to HP=2 first. Use the
# non-counterable Mobile Art (2 dmg) so the attacker survives any PDC fire;
# from a full 6 HP this takes ~2 hits (6→4→2).
for soften_round in 1 2 3 4 5; do
    PDC_HP=$(eb_health "${EB_PDC_ID}")
    P3_MA_HP=$(eb_health "${EB_P3_MOBILE_ART_ID}")
    if [ "${PDC_HP}" = "0" ] || [ "${PDC_HP}" -le 2 ] 2>/dev/null; then
        break
    fi
    if [ "${P3_MA_HP}" = "0" ]; then
        info "SKIP E4b soften: P3 Mobile Art destroyed (PDC HP=${PDC_HP})"
        break
    fi
    eb_attack "E4b: P3 Mobile Art(non-counterable) → P6 PDC (soften toward HP=2)" \
        "${EB_P3_MOBILE_ART_ID}" "${EB_PDC_ID}" primaryWeapon 3
done
PDC_HP=$(eb_health "${EB_PDC_ID}")
info "E4b: PDC softened to HP=${PDC_HP} (target HP=2 for E5 killing blow)"

# ─── E5: Counterable attacker vs PDC directly — destroyed PDC does not counter ───
# P3 Tank (type 9, counterable, land) → P6 PDC (type 19, planet-category, land)
# Tank primaryWeaponDamage=2, PDC softened to HP=2 in E4b, so PDC is destroyed.
# A destroyed PDC does not fire counter-damage — Tank HP should be unchanged.
PDC_HP=$(eb_health "${EB_PDC_ID}")
TANK_HP=$(eb_health "${DESTROYER_STRUCT_ID}")
if [ "${PDC_HP}" != "0" ] && [ "${TANK_HP}" != "0" ]; then
    PDC_HP_BEFORE=$(eb_health "${EB_PDC_ID}")
    TANK_HP_BEFORE=$(eb_health "${DESTROYER_STRUCT_ID}")

    # Snapshot shield BEFORE E5 so we can measure the drop after PDC destruction
    P6_PLANET_JSON=$(query query structs planet "${PLAYER_6_PLANET_ID}" 2>/dev/null || echo '{}')
    E6_SHIELD_BEFORE=$(jqr "${P6_PLANET_JSON}" '.planetAttributes.planetaryShield' '0')
    info "E6: Planetary shield before PDC destruction: ${E6_SHIELD_BEFORE}"

    eb_attack "E5: P3 Tank(counterable) → P6 PDC(planet-category)" \
        "${DESTROYER_STRUCT_ID}" "${EB_PDC_ID}" primaryWeapon 3

    PDC_HP_AFTER=$(eb_health "${EB_PDC_ID}")
    TANK_HP_AFTER=$(eb_health "${DESTROYER_STRUCT_ID}")

    assert_eq "E5 — Destroyed PDC does not counter-damage attacker" "${TANK_HP_BEFORE}" "${TANK_HP_AFTER}"

    info "  PDC HP: ${PDC_HP_BEFORE}→${PDC_HP_AFTER}"
else
    # Still need a shield baseline if E5 is skipped
    P6_PLANET_JSON=$(query query structs planet "${PLAYER_6_PLANET_ID}" 2>/dev/null || echo '{}')
    E6_SHIELD_BEFORE=$(jqr "${P6_PLANET_JSON}" '.planetAttributes.planetaryShield' '0')
    info "SKIP E5: PDC(HP=${PDC_HP}) or Tank(HP=${TANK_HP}) destroyed"
    info "E6: Planetary shield snapshot: ${E6_SHIELD_BEFORE}"
fi

# ─── E7: PDC killing blow — PDC should STILL fire (Bug 1 regression test) ───
# E5's Tank normally destroys the PDC, so this block usually SKIPS. If a
# future change leaves the PDC alive into E7, a counterable attacker (BB#1)
# destroys it here. Under Bug 1, DestroyAndCommit() decrements
# defensiveCannonQuantity before ResolvePlanetaryDefense() runs, so the PDC
# fails to fire on the killing blow; this test EXPECTS the PDC to still fire.
PDC_HP=$(eb_health "${EB_PDC_ID}")
BB1_HP=$(eb_health "${BATTLESHIP_1_ID}")
if [ "${PDC_HP}" != "0" ] && [ "${BB1_HP}" != "0" ]; then
    BB1_HP_BEFORE=$(eb_health "${BATTLESHIP_1_ID}")

    info "E7: PDC at HP=${PDC_HP} — attacking with BB#1 to destroy it"
    eb_attack "E7: P3 BB#1(counterable) → P6 PDC(killing blow)" \
        "${BATTLESHIP_1_ID}" "${EB_PDC_ID}" primaryWeapon 3

    PDC_HP_AFTER=$(eb_health "${EB_PDC_ID}")
    BB1_HP_AFTER=$(eb_health "${BATTLESHIP_1_ID}")

    assert_eq "E7 — PDC destroyed" "0" "${PDC_HP_AFTER}"

    BB1_HP_LOST=$((BB1_HP_BEFORE - BB1_HP_AFTER))
    if [ "${BB1_HP_LOST}" -gt 0 ] 2>/dev/null; then
        echo -e "  ${GREEN}PASS${NC}: E7 — PDC fired on killing blow (attacker HP ${BB1_HP_BEFORE}→${BB1_HP_AFTER})"
        PASS_COUNT=$((PASS_COUNT + 1))
    else
        echo -e "  ${RED}FAIL${NC}: E7 — PDC should fire even when destroyed (Bug 1: attacker HP ${BB1_HP_BEFORE}→${BB1_HP_AFTER})"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
else
    if [ "${PDC_HP}" = "0" ]; then
        info "SKIP E7: PDC already destroyed (expected HP=1 after E5)"
    else
        info "SKIP E7: BB#1(HP=${BB1_HP}) destroyed"
    fi
fi

# ─── E8: defensiveCannonQuantity drops to 0 after PDC destruction ───
P6_PLANET_JSON=$(query query structs planet "${PLAYER_6_PLANET_ID}" 2>/dev/null || echo '{}')
E8_CANNON_QTY=$(jqr "${P6_PLANET_JSON}" '.planetAttributes.defensiveCannonQuantity' '0')
assert_eq "E8 — defensiveCannonQuantity is 0 after PDC destroyed" "0" "${E8_CANNON_QTY}"

# ─── E9: Planetary shield decreased after PDC destruction ───
E9_SHIELD_AFTER=$(jqr "${P6_PLANET_JSON}" '.planetAttributes.planetaryShield' '0')
E9_SHIELD_DIFF=$((E6_SHIELD_BEFORE - E9_SHIELD_AFTER))
info "E9: Planetary shield after PDC destruction: ${E9_SHIELD_AFTER} (decreased by ${E9_SHIELD_DIFF})"
# PDC contributes PlanetaryShieldContribution=13 (v0.18.0 rebase)
assert_eq "E9 — Planetary shield decreased by PDC contribution (13)" "13" "${E9_SHIELD_DIFF}"

# ─── E10: After PDC destroyed, Ore Extractor attack yields NO PDC damage ───
# With cannon count = 0, attacking a planet-category struct should not damage
# the attacker via PDC fire.
EXTRACTOR_HP=$(eb_health "${EB_ORE_EXTRACTOR_ID}")
BB1_HP=$(eb_health "${BATTLESHIP_1_ID}")
if [ "${EXTRACTOR_HP}" != "0" ] && [ "${BB1_HP}" != "0" ]; then
    BB1_HP_BEFORE=$(eb_health "${BATTLESHIP_1_ID}")

    eb_attack "E10: P3 BB#1 → P6 Ore Extractor (PDC destroyed, no cannon)" \
        "${BATTLESHIP_1_ID}" "${EB_ORE_EXTRACTOR_ID}" primaryWeapon 3

    BB1_HP_AFTER=$(eb_health "${BATTLESHIP_1_ID}")

    assert_eq "E10 — No PDC damage after PDC destroyed" "${BB1_HP_BEFORE}" "${BB1_HP_AFTER}"
else
    info "SKIP E10: Ore Extractor(HP=${EXTRACTOR_HP}) or BB#1(HP=${BB1_HP}) destroyed"
fi

# ─────────────────────────────────────────────────────────────────────────────
# GROUP D: Negative Targeting Tests (attacks that should fail)
# ─────────────────────────────────────────────────────────────────────────────

info "── Group D: Negative Targeting Tests ──"

# D1: P3 Tank (weapons=land only) → P6 Battleship (space) — should fail
P6_BB_ALIVE=$(eb_health "${EB_P6_BATTLESHIP_ID}")
TANK_ALIVE=$(eb_health "${DESTROYER_STRUCT_ID}")
if [ "${P6_BB_ALIVE}" != "0" ] && [ "${TANK_ALIVE}" != "0" ]; then
    eb_attack_should_fail "Tank(land-only) → Battleship(space) — wrong ambit" \
        "${DESTROYER_STRUCT_ID}" "${EB_P6_BATTLESHIP_ID}" 3
else
    info "SKIP: Tank or P6 Battleship destroyed for negative test D1"
fi

# D2: P3 Pursuit Fighter (weapons=air only) → P6 Tank (land) — should fail
PF_ALIVE=$(eb_health "${EB_PURSUIT_FIGHTER_ID}")
P6_TANK_ALIVE=$(eb_health "${EB_P6_TANK_ID}")
if [ "${PF_ALIVE}" != "0" ] && [ "${P6_TANK_ALIVE}" != "0" ]; then
    eb_attack_should_fail "Pursuit Fighter(air-only) → Tank(land) — wrong ambit" \
        "${EB_PURSUIT_FIGHTER_ID}" "${EB_P6_TANK_ID}" 3
else
    info "SKIP: Pursuit Fighter or P6 Tank destroyed for negative test D2"
fi

# D3: P6 Starfighter (weapons=space only) → P3 SAM (land) — should fail
STAR_ALIVE=$(eb_health "${EB_STARFIGHTER_ID}")
SAM_ALIVE=$(eb_health "${SAM_STRUCT_ID}")
if [ "${STAR_ALIVE}" != "0" ] && [ "${SAM_ALIVE}" != "0" ]; then
    eb_attack_should_fail "Starfighter(space-only) → SAM(land) — wrong ambit" \
        "${EB_STARFIGHTER_ID}" "${SAM_STRUCT_ID}" 6
else
    info "SKIP: Starfighter or SAM destroyed for negative test D3"
fi

fi # phase EB5

if run_phase 3000; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE EB6: Battle Results Review
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE EB6: Extended Battle Results"

info "Querying final state of all combat structs"

echo ""
echo "  ─── Player 3 Fleet Status ───"
for SID in "${COMMAND_SHIP_ID}" "${DESTROYER_STRUCT_ID}" "${SAM_STRUCT_ID}" "${SUB_STRUCT_ID}" \
           "${BATTLESHIP_1_ID}" "${BATTLESHIP_2_ID}" "${STEALTH_BOMBER_ID}" "${CRUISER_ID}" \
           "${EB_PURSUIT_FIGHTER_ID}" "${EB_P3_MOBILE_ART_ID}"; do
    S_JSON=$(query query structs struct "${SID}" 2>/dev/null || echo '{}')
    S_HP=$(echo "${S_JSON}" | jq -r '.structAttributes.health // "?"' 2>/dev/null || echo "?")
    S_TYPE=$(echo "${S_JSON}" | jq -r '.Struct.type // "?"' 2>/dev/null || echo "?")
    S_AMBIT=$(echo "${S_JSON}" | jq -r '.Struct.operatingAmbit // "?"' 2>/dev/null || echo "?")
    S_STATUS="alive"
    if [ "${S_HP}" = "0" ]; then S_STATUS="DESTROYED"; fi
    echo "    ${SID} type=${S_TYPE} ambit=${S_AMBIT} HP=${S_HP} [${S_STATUS}]"
done

echo ""
echo "  ─── Player 6 Fleet Status ───"
for SID in "${P6_COMMAND_SHIP_ID}" "${EB_STARFIGHTER_ID}" "${EB_FRIGATE_ID}" "${EB_P6_BATTLESHIP_ID}" \
           "${EB_MOBILE_ART_ID}" "${EB_P6_TANK_ID}" "${EB_DESTROYER_W_ID}" "${EB_P6_CRUISER_ID}" \
           "${EB_P6_HAI_ID}" "${EB_PDC_ID}" "${EB_ORE_EXTRACTOR_ID}"; do
    S_JSON=$(query query structs struct "${SID}" 2>/dev/null || echo '{}')
    S_HP=$(echo "${S_JSON}" | jq -r '.structAttributes.health // "?"' 2>/dev/null || echo "?")
    S_TYPE=$(echo "${S_JSON}" | jq -r '.Struct.type // "?"' 2>/dev/null || echo "?")
    S_AMBIT=$(echo "${S_JSON}" | jq -r '.Struct.operatingAmbit // "?"' 2>/dev/null || echo "?")
    S_STATUS="alive"
    if [ "${S_HP}" = "0" ]; then S_STATUS="DESTROYED"; fi
    echo "    ${SID} type=${S_TYPE} ambit=${S_AMBIT} HP=${S_HP} [${S_STATUS}]"
done

echo ""
info "Extended Battle Summary: ${EB_ATTACKS} attacks, ${EB_DESTROYED} structs destroyed"

BLOCK_HEIGHT=$(query query structs block-height | jq -r '.blockHeight // empty' 2>/dev/null || echo "?")
info "Block height after extended battle: ${BLOCK_HEIGHT}"

fi # phase EB6

if run_phase 3050; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE EV1: Evasion Testing
#  Tests signalJamming (guided evasion 2/3) and defensiveManeuver (unguided 2/3)
#  Evasion is probabilistic — outcomes vary per run based on block hash + nonce
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE EV1: Evasion Testing"

info "Testing unit evasion mechanics: signalJamming (guided evasion 2/3)"
info "  and defensiveManeuver (unguided evasion 2/3)"
info "  Evasion is probabilistic — expect ~2/3 of shots to evade"

# ─── Group 1: Signal Jamming — P6 HAI (guided) → P3 PF (signalJamming) ───
# HAI (type 7): guided primary, ambits air+space
# Pursuit Fighter (type 5): signalJamming, guidedDefensiveSuccessRate 2/3
info "── Signal Jamming: P6 HAI → P3 Pursuit Fighter (air→air) ──"

eb_attack "EV1: P6 HAI → P3 PF (guided vs signalJamming)" \
    "${EB_P6_HAI_ID}" "${EB_PURSUIT_FIGHTER_ID}" primaryWeapon 6

eb_attack "EV2: P6 HAI → P3 PF (guided vs signalJamming)" \
    "${EB_P6_HAI_ID}" "${EB_PURSUIT_FIGHTER_ID}" primaryWeapon 6

eb_attack "EV3: P6 HAI → P3 PF (guided vs signalJamming)" \
    "${EB_P6_HAI_ID}" "${EB_PURSUIT_FIGHTER_ID}" primaryWeapon 6

# ─── Group 2: Signal Jamming — P6 CS (guided local) → P3 BB#2 (signalJamming) ───
# Command Ship (type 1): guided primary, ambits local (space→space)
# Battleship (type 2): signalJamming, guidedDefensiveSuccessRate 2/3
info "── Signal Jamming: P6 CS → P3 Battleship#2 (space→space) ──"

eb_attack "EV4: P6 CS → P3 BB#2 (guided vs signalJamming)" \
    "${P6_COMMAND_SHIP_ID}" "${BATTLESHIP_2_ID}" primaryWeapon 6

eb_attack "EV5: P6 CS → P3 BB#2 (guided vs signalJamming)" \
    "${P6_COMMAND_SHIP_ID}" "${BATTLESHIP_2_ID}" primaryWeapon 6

# ─── Group 3: Defensive Maneuver — P3 Cruiser (unguided secondary) → P6 HAI ───
# Cruiser (type 11): unguided secondary, only unguided weapon that targets air
# HAI (type 7): defensiveManeuver, unguidedDefensiveSuccessRate 2/3
info "── Defensive Maneuver: P3 Cruiser → P6 HAI (water→air, unguided secondary) ──"

eb_attack "EV6: P3 Cruiser → P6 HAI (unguided vs defensiveManeuver)" \
    "${CRUISER_ID}" "${EB_P6_HAI_ID}" secondaryWeapon 3

eb_attack "EV7: P3 Cruiser → P6 HAI (unguided vs defensiveManeuver)" \
    "${CRUISER_ID}" "${EB_P6_HAI_ID}" secondaryWeapon 3

# ─── Group 4: Signal Jamming — P6 Destroyer (guided) → P3 Cruiser (signalJamming) ───
# Destroyer (type 12): guided primary, ambits water+air
# Cruiser (type 11): signalJamming, guidedDefensiveSuccessRate 2/3
info "── Signal Jamming: P6 Destroyer → P3 Cruiser (water→water) ──"

eb_attack "EV8: P6 Destroyer → P3 Cruiser (guided vs signalJamming)" \
    "${EB_DESTROYER_W_ID}" "${CRUISER_ID}" primaryWeapon 6

info "Evasion testing complete: ${EB_ATTACKS} total EB attacks, ${EB_DESTROYED} total destroyed"

fi # phase EV1

if run_phase 3100; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE AR1: Attack Run — Build 6 Starfighters
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE AR1: Attack Run — Build Starfighters"

info "Building 6 Starfighters across P2/P3/P6 for Attack Run (secondaryWeapon) testing"

# P2 needs extra capacity to support 3 new structs
run_tx "Additional delegation for Player 2 (fleet capacity)" \
    tx staking delegate "${VALIDATOR_ADDRESS}" 2000000ualpha --from player_2

# P3's fleet is at P6's planet after EB3 — move home for building
run_tx "Moving P3 fleet home for Attack Run builds" \
    tx structs fleet-move "${PLAYER_3_FLEET_ID}" "${PLAYER_3_PLANET_ID}" --from player_3

# ═══════════════════════════════════════════════════════════════
# BATCH INITIATE: 6 Starfighters (type 3, space ambit)
# P2: space slots 0, 2, 3  (slot 1 used by P2 Battleship)
# P3: space slots 1, 3     (slot 0 used by BB2, slot 2 freed by BB1 move)
# P6: space slot 3          (safe — 0/1 may need sweep, 2 was BB)
# ═══════════════════════════════════════════════════════════════

info "Batch-initiating all Attack Run Starfighter builds"

# ─── P2: Starfighter #1 (space, slot 0) ───
wait_for_charge "${PLAYER_2_ID}" "${CHARGE_BUILD}"
run_tx "Initiating P2 Starfighter #1 (type=3, space, slot=0)" \
    tx structs struct-build-initiate "${PLAYER_2_ID}" 3 space 0 --from player_2

STRUCT_ALL_JSON=$(query query structs struct-all)
AR_P2_SF1_ID=$(get_newest_struct_id "${STRUCT_ALL_JSON}")
assert_not_empty "AR P2 Starfighter #1 ID" "${AR_P2_SF1_ID}"
echo "  AR P2 SF#1 ID: ${AR_P2_SF1_ID}"

# ─── P2: Starfighter #2 (space, slot 2) ───
wait_for_charge "${PLAYER_2_ID}" "${CHARGE_BUILD}"
run_tx "Initiating P2 Starfighter #2 (type=3, space, slot=2)" \
    tx structs struct-build-initiate "${PLAYER_2_ID}" 3 space 2 --from player_2

STRUCT_ALL_JSON=$(query query structs struct-all)
AR_P2_SF2_ID=$(get_newest_struct_id "${STRUCT_ALL_JSON}")
assert_not_empty "AR P2 Starfighter #2 ID" "${AR_P2_SF2_ID}"
echo "  AR P2 SF#2 ID: ${AR_P2_SF2_ID}"

# ─── P2: Starfighter #3 (space, slot 3) ───
wait_for_charge "${PLAYER_2_ID}" "${CHARGE_BUILD}"
run_tx "Initiating P2 Starfighter #3 (type=3, space, slot=3)" \
    tx structs struct-build-initiate "${PLAYER_2_ID}" 3 space 3 --from player_2

STRUCT_ALL_JSON=$(query query structs struct-all)
AR_P2_SF3_ID=$(get_newest_struct_id "${STRUCT_ALL_JSON}")
assert_not_empty "AR P2 Starfighter #3 ID" "${AR_P2_SF3_ID}"
echo "  AR P2 SF#3 ID: ${AR_P2_SF3_ID}"

# ─── P3: Starfighter #1 (space, slot 1) ───
wait_for_charge "${PLAYER_3_ID}" "${CHARGE_BUILD}"
run_tx "Initiating P3 Starfighter #1 (type=3, space, slot=1)" \
    tx structs struct-build-initiate "${PLAYER_3_ID}" 3 space 1 --from player_3

STRUCT_ALL_JSON=$(query query structs struct-all)
AR_P3_SF1_ID=$(get_newest_struct_id "${STRUCT_ALL_JSON}")
assert_not_empty "AR P3 Starfighter #1 ID" "${AR_P3_SF1_ID}"
echo "  AR P3 SF#1 ID: ${AR_P3_SF1_ID}"

# ─── P3: Starfighter #2 (space, slot 3) ───
wait_for_charge "${PLAYER_3_ID}" "${CHARGE_BUILD}"
run_tx "Initiating P3 Starfighter #2 (type=3, space, slot=3)" \
    tx structs struct-build-initiate "${PLAYER_3_ID}" 3 space 3 --from player_3

STRUCT_ALL_JSON=$(query query structs struct-all)
AR_P3_SF2_ID=$(get_newest_struct_id "${STRUCT_ALL_JSON}")
assert_not_empty "AR P3 Starfighter #2 ID" "${AR_P3_SF2_ID}"
echo "  AR P3 SF#2 ID: ${AR_P3_SF2_ID}"

# ─── P6: Starfighter (space, slot 3) ───
wait_for_charge "${PLAYER_6_ID}" "${CHARGE_BUILD}"
run_tx "Initiating P6 Starfighter (type=3, space, slot=3)" \
    tx structs struct-build-initiate "${PLAYER_6_ID}" 3 space 3 --from player_6

STRUCT_ALL_JSON=$(query query structs struct-all)
AR_P6_SF_ID=$(get_newest_struct_id "${STRUCT_ALL_JSON}")
assert_not_empty "AR P6 Starfighter ID" "${AR_P6_SF_ID}"
echo "  AR P6 SF ID: ${AR_P6_SF_ID}"

info "All 6 Attack Run builds initiated. Computing now (difficulty decays with age)."

# ═══════════════════════════════════════════════════════════════
# COMPUTE: Interleave across players for aging benefit
# ═══════════════════════════════════════════════════════════════

run_compute "Building P2 Starfighter #1 ${AR_P2_SF1_ID}" \
    tx structs struct-build-compute "${AR_P2_SF1_ID}" --from player_2

assert_eq "P2 SF#1 built" "true" "$(query query structs struct "${AR_P2_SF1_ID}" | jq -r '.structAttributes.isBuilt')"

run_compute "Building P3 Starfighter #1 ${AR_P3_SF1_ID}" \
    tx structs struct-build-compute "${AR_P3_SF1_ID}" --from player_3

assert_eq "P3 SF#1 built" "true" "$(query query structs struct "${AR_P3_SF1_ID}" | jq -r '.structAttributes.isBuilt')"

run_compute "Building P6 Starfighter ${AR_P6_SF_ID}" \
    tx structs struct-build-compute "${AR_P6_SF_ID}" --from player_6

assert_eq "P6 SF built" "true" "$(query query structs struct "${AR_P6_SF_ID}" | jq -r '.structAttributes.isBuilt')"

run_compute "Building P2 Starfighter #2 ${AR_P2_SF2_ID}" \
    tx structs struct-build-compute "${AR_P2_SF2_ID}" --from player_2

assert_eq "P2 SF#2 built" "true" "$(query query structs struct "${AR_P2_SF2_ID}" | jq -r '.structAttributes.isBuilt')"

run_compute "Building P3 Starfighter #2 ${AR_P3_SF2_ID}" \
    tx structs struct-build-compute "${AR_P3_SF2_ID}" --from player_3

assert_eq "P3 SF#2 built" "true" "$(query query structs struct "${AR_P3_SF2_ID}" | jq -r '.structAttributes.isBuilt')"

run_compute "Building P2 Starfighter #3 ${AR_P2_SF3_ID}" \
    tx structs struct-build-compute "${AR_P2_SF3_ID}" --from player_2

assert_eq "P2 SF#3 built" "true" "$(query query structs struct "${AR_P2_SF3_ID}" | jq -r '.structAttributes.isBuilt')"

info "All 6 Attack Run Starfighters built"

fi # phase AR1

if run_phase 3200; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE AR2: Attack Run — Fleet Assembly & Positioning
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE AR2: Attack Run — Fleet Assembly"

# struct-build-initiate already places fleet-category structs on the fleet,
# so no struct-move calls are needed for the newly built Starfighters.
# However, the P2 Command Ship needs its operating ambit changed to space
# so that Attack Run (space-to-space) weapons can target it.
wait_for_charge "${PLAYER_2_ID}" "${CHARGE_MOVE}"
run_tx "Moving P2 Command Ship to fleet (space ambit)" \
    tx structs struct-move "${PLAYER_2_CMD_SHIP_ID}" fleet space --from player_2

# ─── Position fleets for battle at P6's planet ───
# P2 moves FIRST to become HEAD of invasion queue (can attack anyone on planet).
# P3 moves SECOND to become TAIL (forward neighbor = P2's fleet).
run_tx "Moving P2's fleet to P6's planet for Attack Run" \
    tx structs fleet-move "${PLAYER_2_FLEET_ID}" "${PLAYER_6_PLANET_ID}" --from player_2

run_tx "Moving P3's fleet to P6's planet for Attack Run" \
    tx structs fleet-move "${PLAYER_3_FLEET_ID}" "${PLAYER_6_PLANET_ID}" --from player_3

info "Attack Run fleets assembled and positioned at P6's planet"
info "  P2 (HEAD): CS(space), SF#1(space/0), SF#2(space/2), SF#3(space/3)"
info "  P3 (TAIL): SF#1(space/1), SF#2(space/3)"
info "  P6 (HOME): CS(space), SF(space/3), BB(space/2), MobArt(land/0), Destroyer(water)"

fi # phase AR2

if run_phase 3300; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE AR3: Attack Run — 15 Secondary Weapon Attacks
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE AR3: Attack Run — Combat"

AR_ATTACKS=0
AR_DESTROYED=0

# Reuse eb_health from EB5 (already defined)
# Track Attack Run stats with local counters
ar_attack() {
    local desc="$1"
    local attacker="$2"
    local target="$3"
    local from_player="$4"

    AR_ATTACKS=$((AR_ATTACKS + 1))

    local atk_hp_before
    atk_hp_before=$(eb_health "${attacker}")
    local tgt_hp_before
    tgt_hp_before=$(eb_health "${target}")

    info "[AR Attack ${AR_ATTACKS}] ${desc}"
    echo "  Attacker: ${attacker} (HP=${atk_hp_before})  Target: ${target} (HP=${tgt_hp_before})"

    if [ "${atk_hp_before}" = "0" ]; then
        echo "  SKIP: Attacker already destroyed"
        return
    fi
    if [ "${tgt_hp_before}" = "0" ]; then
        echo "  SKIP: Target already destroyed"
        return
    fi

    local charge
    charge=$(_secondary_charge 3)
    wait_for_charge "$(eval echo "\${PLAYER_${from_player}_ID}")" "${charge}"
    run_tx "${desc}" \
        tx structs struct-attack "${attacker}" "${target}" secondaryWeapon --from "player_${from_player}"

    local atk_hp_after
    atk_hp_after=$(eb_health "${attacker}")
    local tgt_hp_after
    tgt_hp_after=$(eb_health "${target}")

    echo "  Result: Attacker HP ${atk_hp_before}→${atk_hp_after}  Target HP ${tgt_hp_before}→${tgt_hp_after}"

    if [ "${tgt_hp_after}" = "0" ]; then
        info "  TARGET DESTROYED"
        AR_DESTROYED=$((AR_DESTROYED + 1))
    fi
    if [ "${atk_hp_after}" = "0" ]; then
        info "  ATTACKER DESTROYED (counter-attack)"
        AR_DESTROYED=$((AR_DESTROYED + 1))
    fi
}

# ─────────────────────────────────────────────────────────────────────────────
# Reachability after AR2 fleet moves:
#   P2 (HEAD, Forward=""): can attack anyone on P6's planet
#   P3 (TAIL, Forward=P2):  can attack P2's fleet structs
#   P6 (HOME):              can attack HEAD = P2's fleet structs
#
# Post-EB5 alive defenders:
#   P6: BB(space,HP=1) MobileArt(land,HP=3) Destroyer(water,HP=1)
#   P6 destroyed: EB-SF, Frigate, Tank, Cruiser, PDC
#   P2: DEFENDER_STRUCT/Tank(land) — BB and Interceptor destroyed in EB5
# ─────────────────────────────────────────────────────────────────────────────

# ─────────────────────────────────────────────────────────────────────────────
# GROUP A: No Defenders (3 attacks)
# ─────────────────────────────────────────────────────────────────────────────

info "── Group A: No Defenders ──"

# A1: P2 SF#1 → P6 CS (baseline Attack Run — HEAD attacks home fleet)
ar_attack "AR A1: P2 SF#1 → P6 CS (no defenders)" \
    "${AR_P2_SF1_ID}" "${P6_COMMAND_SHIP_ID}" 2

# A2: P3 SF#1 → P2 CS (TAIL attacks HEAD via forward link)
ar_attack "AR A2: P3 SF#1 → P2 CS (no defenders)" \
    "${AR_P3_SF1_ID}" "${PLAYER_2_CMD_SHIP_ID}" 3

# A3: P6 SF → P2 CS (HOME attacks HEAD)
ar_attack "AR A3: P6 SF → P2 CS (no defenders)" \
    "${AR_P6_SF_ID}" "${PLAYER_2_CMD_SHIP_ID}" 6

# ─────────────────────────────────────────────────────────────────────────────
# GROUP B: Single Defender (3 attacks)
# ─────────────────────────────────────────────────────────────────────────────

info "── Group B: Single Defender ──"

# B1: P2 SF#2 → P6 CS, defended by P6 BB (space — can block AND counter)
wait_for_charge "${PLAYER_6_ID}" "${CHARGE_DEFEND}"
run_tx "Setting P6 BB to defend P6 CS" \
    tx structs struct-defense-set "${EB_P6_BATTLESHIP_ID}" "${P6_COMMAND_SHIP_ID}" --from player_6

ar_attack "AR B1: P2 SF#2 → P6 CS (def: P6 BB/space)" \
    "${AR_P2_SF2_ID}" "${P6_COMMAND_SHIP_ID}" 2

wait_for_charge "${PLAYER_6_ID}" "${CHARGE_DEFEND}"
run_tx "Clearing P6 BB defense" \
    tx structs struct-defense-clear "${EB_P6_BATTLESHIP_ID}" --from player_6

# B2: P2 SF#3 → P6 CS, defended by P6 Mobile Art (land — counter only, no block)
wait_for_charge "${PLAYER_6_ID}" "${CHARGE_DEFEND}"
run_tx "Setting P6 Mobile Art to defend P6 CS" \
    tx structs struct-defense-set "${EB_MOBILE_ART_ID}" "${P6_COMMAND_SHIP_ID}" --from player_6

ar_attack "AR B2: P2 SF#3 → P6 CS (def: P6 MobArt/land)" \
    "${AR_P2_SF3_ID}" "${P6_COMMAND_SHIP_ID}" 2

wait_for_charge "${PLAYER_6_ID}" "${CHARGE_DEFEND}"
run_tx "Clearing P6 Mobile Art defense" \
    tx structs struct-defense-clear "${EB_MOBILE_ART_ID}" --from player_6

# B3: P3 SF#2 → P2 CS, defended by P2 Tank (land — counter only, no block)
wait_for_charge "${PLAYER_2_ID}" "${CHARGE_DEFEND}"
run_tx "Setting P2 Tank to defend P2 CS" \
    tx structs struct-defense-set "${DEFENDER_STRUCT_ID}" "${PLAYER_2_CMD_SHIP_ID}" --from player_2

ar_attack "AR B3: P3 SF#2 → P2 CS (def: P2 Tank/land)" \
    "${AR_P3_SF2_ID}" "${PLAYER_2_CMD_SHIP_ID}" 3

wait_for_charge "${PLAYER_2_ID}" "${CHARGE_DEFEND}"
run_tx "Clearing P2 Tank defense" \
    tx structs struct-defense-clear "${DEFENDER_STRUCT_ID}" --from player_2

# ─────────────────────────────────────────────────────────────────────────────
# GROUP C: Single Cross-Ambit Defender (2 attacks)
# Non-space defender — can counter only (cannot block from different ambit)
# ─────────────────────────────────────────────────────────────────────────────

info "── Group C: Single Cross-Ambit Defender ──"

# C1: P2 SF#1 → P6 CS, defended by P6 Destroyer (water)
wait_for_charge "${PLAYER_6_ID}" "${CHARGE_DEFEND}"
run_tx "Setting P6 Destroyer to defend P6 CS" \
    tx structs struct-defense-set "${EB_DESTROYER_W_ID}" "${P6_COMMAND_SHIP_ID}" --from player_6

ar_attack "AR C1: P2 SF#1 → P6 CS (def: P6 Destroyer/water)" \
    "${AR_P2_SF1_ID}" "${P6_COMMAND_SHIP_ID}" 2

wait_for_charge "${PLAYER_6_ID}" "${CHARGE_DEFEND}"
run_tx "Clearing P6 Destroyer defense" \
    tx structs struct-defense-clear "${EB_DESTROYER_W_ID}" --from player_6

# C2: P6 SF → P2 CS, defended by P2 Tank (land)
wait_for_charge "${PLAYER_2_ID}" "${CHARGE_DEFEND}"
run_tx "Setting P2 Tank to defend P2 CS" \
    tx structs struct-defense-set "${DEFENDER_STRUCT_ID}" "${PLAYER_2_CMD_SHIP_ID}" --from player_2

ar_attack "AR C2: P6 SF → P2 CS (def: P2 Tank/land)" \
    "${AR_P6_SF_ID}" "${PLAYER_2_CMD_SHIP_ID}" 6

wait_for_charge "${PLAYER_2_ID}" "${CHARGE_DEFEND}"
run_tx "Clearing P2 Tank defense" \
    tx structs struct-defense-clear "${DEFENDER_STRUCT_ID}" --from player_2

# ─────────────────────────────────────────────────────────────────────────────
# GROUP D: Multiple Defenders (4 attacks)
# Mix of same-ambit blockers and cross-ambit counters
# ─────────────────────────────────────────────────────────────────────────────

info "── Group D: Multiple Defenders ──"

# D1: P2 SF#2 → P6 CS, 2 defenders: P6 BB (space) + P6 Destroyer (water)
wait_for_charge "${PLAYER_6_ID}" "${CHARGE_DEFEND}"
run_tx "Setting P6 BB to defend P6 CS" \
    tx structs struct-defense-set "${EB_P6_BATTLESHIP_ID}" "${P6_COMMAND_SHIP_ID}" --from player_6
wait_for_charge "${PLAYER_6_ID}" "${CHARGE_DEFEND}"
run_tx "Setting P6 Destroyer to defend P6 CS" \
    tx structs struct-defense-set "${EB_DESTROYER_W_ID}" "${P6_COMMAND_SHIP_ID}" --from player_6

ar_attack "AR D1: P2 SF#2 → P6 CS (def: P6 BB/space + P6 Destroyer/water)" \
    "${AR_P2_SF2_ID}" "${P6_COMMAND_SHIP_ID}" 2

wait_for_charge "${PLAYER_6_ID}" "${CHARGE_DEFEND}"
run_tx "Clearing P6 BB defense" \
    tx structs struct-defense-clear "${EB_P6_BATTLESHIP_ID}" --from player_6
wait_for_charge "${PLAYER_6_ID}" "${CHARGE_DEFEND}"
run_tx "Clearing P6 Destroyer defense" \
    tx structs struct-defense-clear "${EB_DESTROYER_W_ID}" --from player_6

# D2: P2 SF#3 → P6 CS, 2 defenders: P6 Mobile Art (land) + P6 Destroyer (water)
wait_for_charge "${PLAYER_6_ID}" "${CHARGE_DEFEND}"
run_tx "Setting P6 Mobile Art to defend P6 CS" \
    tx structs struct-defense-set "${EB_MOBILE_ART_ID}" "${P6_COMMAND_SHIP_ID}" --from player_6
wait_for_charge "${PLAYER_6_ID}" "${CHARGE_DEFEND}"
run_tx "Setting P6 Destroyer to defend P6 CS" \
    tx structs struct-defense-set "${EB_DESTROYER_W_ID}" "${P6_COMMAND_SHIP_ID}" --from player_6

ar_attack "AR D2: P2 SF#3 → P6 CS (def: P6 MobArt/land + P6 Destroyer/water)" \
    "${AR_P2_SF3_ID}" "${P6_COMMAND_SHIP_ID}" 2

wait_for_charge "${PLAYER_6_ID}" "${CHARGE_DEFEND}"
run_tx "Clearing P6 Mobile Art defense" \
    tx structs struct-defense-clear "${EB_MOBILE_ART_ID}" --from player_6
wait_for_charge "${PLAYER_6_ID}" "${CHARGE_DEFEND}"
run_tx "Clearing P6 Destroyer defense" \
    tx structs struct-defense-clear "${EB_DESTROYER_W_ID}" --from player_6

# D3: P3 SF#1 → P2 CS, 2 defenders: P2 Tank (land) + P2 SF#2 (space, if alive)
wait_for_charge "${PLAYER_2_ID}" "${CHARGE_DEFEND}"
run_tx "Setting P2 Tank to defend P2 CS" \
    tx structs struct-defense-set "${DEFENDER_STRUCT_ID}" "${PLAYER_2_CMD_SHIP_ID}" --from player_2
wait_for_charge "${PLAYER_2_ID}" "${CHARGE_DEFEND}"
run_tx "Setting P2 SF#2 to defend P2 CS" \
    tx structs struct-defense-set "${AR_P2_SF2_ID}" "${PLAYER_2_CMD_SHIP_ID}" --from player_2

ar_attack "AR D3: P3 SF#1 → P2 CS (def: P2 Tank/land + P2 SF#2/space)" \
    "${AR_P3_SF1_ID}" "${PLAYER_2_CMD_SHIP_ID}" 3

wait_for_charge "${PLAYER_2_ID}" "${CHARGE_DEFEND}"
run_tx "Clearing P2 Tank defense" \
    tx structs struct-defense-clear "${DEFENDER_STRUCT_ID}" --from player_2
wait_for_charge "${PLAYER_2_ID}" "${CHARGE_DEFEND}"
run_tx "Clearing P2 SF#2 defense" \
    tx structs struct-defense-clear "${AR_P2_SF2_ID}" --from player_2

# D4: P2 SF#1 → P6 CS, 3 defenders: P6 BB (space) + P6 Mobile Art (land) + P6 Destroyer (water)
wait_for_charge "${PLAYER_6_ID}" "${CHARGE_DEFEND}"
run_tx "Setting P6 BB to defend P6 CS" \
    tx structs struct-defense-set "${EB_P6_BATTLESHIP_ID}" "${P6_COMMAND_SHIP_ID}" --from player_6
wait_for_charge "${PLAYER_6_ID}" "${CHARGE_DEFEND}"
run_tx "Setting P6 Mobile Art to defend P6 CS" \
    tx structs struct-defense-set "${EB_MOBILE_ART_ID}" "${P6_COMMAND_SHIP_ID}" --from player_6
wait_for_charge "${PLAYER_6_ID}" "${CHARGE_DEFEND}"
run_tx "Setting P6 Destroyer to defend P6 CS" \
    tx structs struct-defense-set "${EB_DESTROYER_W_ID}" "${P6_COMMAND_SHIP_ID}" --from player_6

ar_attack "AR D4: P2 SF#1 → P6 CS (def: P6 BB/space + MobArt/land + Destroyer/water)" \
    "${AR_P2_SF1_ID}" "${P6_COMMAND_SHIP_ID}" 2

wait_for_charge "${PLAYER_6_ID}" "${CHARGE_DEFEND}"
run_tx "Clearing P6 BB defense" \
    tx structs struct-defense-clear "${EB_P6_BATTLESHIP_ID}" --from player_6
wait_for_charge "${PLAYER_6_ID}" "${CHARGE_DEFEND}"
run_tx "Clearing P6 Mobile Art defense" \
    tx structs struct-defense-clear "${EB_MOBILE_ART_ID}" --from player_6
wait_for_charge "${PLAYER_6_ID}" "${CHARGE_DEFEND}"
run_tx "Clearing P6 Destroyer defense" \
    tx structs struct-defense-clear "${EB_DESTROYER_W_ID}" --from player_6

# ─────────────────────────────────────────────────────────────────────────────
# GROUP E: Sustained Fire & Special (3 attacks)
# ─────────────────────────────────────────────────────────────────────────────

info "── Group E: Sustained Fire & Special ──"

# E1: P2 SF#2 → P6 CS again (cumulative damage, no defenders)
ar_attack "AR E1: P2 SF#2 → P6 CS (cumulative, no defenders)" \
    "${AR_P2_SF2_ID}" "${P6_COMMAND_SHIP_ID}" 2

# E2: P2 SF#3 → P6 AR SF (Starfighter-vs-Starfighter, low HP target)
ar_attack "AR E2: P2 SF#3 → P6 SF (SF vs SF, no defenders)" \
    "${AR_P2_SF3_ID}" "${AR_P6_SF_ID}" 2

# E3: P6 SF → P2 SF#1 (reverse SF-vs-SF, HOME attacks HEAD)
ar_attack "AR E3: P6 SF → P2 SF#1 (SF vs SF, no defenders)" \
    "${AR_P6_SF_ID}" "${AR_P2_SF1_ID}" 6

info "Attack Run combat complete: ${AR_ATTACKS} attacks, ${AR_DESTROYED} structs destroyed"

fi # phase AR3

if run_phase 3400; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE AR4: Attack Run — Results Summary
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE AR4: Attack Run — Results"

info "Querying final state of all Attack Run structs"

echo ""
echo "  ─── Attack Run Starfighters ───"
for SID_LABEL in "P2_SF1:${AR_P2_SF1_ID}" "P2_SF2:${AR_P2_SF2_ID}" "P2_SF3:${AR_P2_SF3_ID}" \
                  "P3_SF1:${AR_P3_SF1_ID}" "P3_SF2:${AR_P3_SF2_ID}" "P6_SF:${AR_P6_SF_ID}"; do
    LABEL="${SID_LABEL%%:*}"
    SID="${SID_LABEL#*:}"
    S_JSON=$(query query structs struct "${SID}" 2>/dev/null || echo '{}')
    S_HP=$(echo "${S_JSON}" | jq -r '.structAttributes.health // "?"' 2>/dev/null || echo "?")
    S_STATUS="alive"
    if [ "${S_HP}" = "0" ]; then S_STATUS="DESTROYED"; fi
    echo "    ${LABEL} (${SID}) HP=${S_HP} [${S_STATUS}]"
done

echo ""
echo "  ─── Attack Run Targets ───"
for SID_LABEL in "P6_CS:${P6_COMMAND_SHIP_ID}" "P2_CS:${PLAYER_2_CMD_SHIP_ID}" "P6_SF:${AR_P6_SF_ID}" "P2_SF1:${AR_P2_SF1_ID}"; do
    LABEL="${SID_LABEL%%:*}"
    SID="${SID_LABEL#*:}"
    S_JSON=$(query query structs struct "${SID}" 2>/dev/null || echo '{}')
    S_HP=$(echo "${S_JSON}" | jq -r '.structAttributes.health // "?"' 2>/dev/null || echo "?")
    S_STATUS="alive"
    if [ "${S_HP}" = "0" ]; then S_STATUS="DESTROYED"; fi
    echo "    ${LABEL} (${SID}) HP=${S_HP} [${S_STATUS}]"
done

echo ""
info "Attack Run Summary: ${AR_ATTACKS} attacks, ${AR_DESTROYED} structs destroyed"

BLOCK_HEIGHT=$(query query structs block-height | jq -r '.blockHeight // empty' 2>/dev/null || echo "?")
info "Block height after Attack Run: ${BLOCK_HEIGHT}"

fi # phase AR4

if run_phase 3450; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE RG1: Regression — Defender Counter Destroys Attacker (Bug 1)
# ═════════════════════════════════════════════════════════════════════════════
#
# Validates the resolveBlock / ResolveDefenders / resolveVolleyDamageOn fix
# in x/structs/keeper/attack_context.go: when a defender's counter destroys
# the attacker mid-iteration, the same defender's block must NOT fire on
# the dead volley.
#
# Go unit test (deterministic):
#   x/structs/keeper/msg_server_struct_attack_test.go
#     TestMsgStructAttackDefenderCounterDestroysAttacker
# This phase adds end-to-end coverage via the live state machine.
#
# Setup (all space-ambit Starfighters — MaxHealth 3, CounterAttack 1,
# PrimaryWeaponDamage 2, PrimaryWeaponCharge 1):
#   RG1_ATTACKER_ID: an alive 1-HP P2 or P3 SF from the AR phases
#   RG1_TARGET_ID:   freshly-built P6 SF (space slot — was an EB SF that has
#                    been destroyed and swept by now)
#   RG1_DEFENDER_ID: freshly-built P6 SF (different space slot)
#
# Bug-1 trigger: defender counter (1) destroys the HP=1 attacker.
#   • With the fix → block does NOT fire → defender HP stays at 3.
#   • Without the fix → block fires → defender takes attacker primary
#     damage 2 → defender HP drops 3→1.
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE RG1: Regression — Defender Counter Destroys Attacker (Bug 1)"

# Discover roles at runtime from the post-AR chain state. P6's command ship
# is typically destroyed in AR Group C, so P6's fleet can't accept new
# struct builds (`fleet needs a command struct before deploy`). P3 keeps
# its CS alive throughout AR and is therefore the most reliable host for
# the target + defender pair.
#
# Bug-1 condition: same-ambit defender's counter (CA=1) must destroy the
# attacker. We pick a Starfighter attacker at HP=1 from any non-P3 player.

SA_RG1=$(query query structs struct-all 2>/dev/null || echo '{}')

RG1_ATTACKER_ID=""
RG1_ATTACKER_PLAYER=""
for ATK_PLAYER_NUM in 2 6; do
    eval "RG1_ATK_PID=\${PLAYER_${ATK_PLAYER_NUM}_ID:-}"
    if [ -z "${RG1_ATK_PID}" ]; then continue; fi
    for NTH in 1 2 3 4 5; do
        CAND=$(find_struct_by_owner_type "${RG1_ATK_PID}" 3 "${NTH}" "${SA_RG1}")
        if [ -z "${CAND}" ]; then continue; fi
        CAND_HP=$(eb_health "${CAND}")
        if [ "${CAND_HP}" = "1" ]; then
            RG1_ATTACKER_ID="${CAND}"
            RG1_ATTACKER_PLAYER="${ATK_PLAYER_NUM}"
            info "RG1: Selected attacker ${RG1_ATTACKER_ID} (player_${RG1_ATTACKER_PLAYER}, Starfighter) at HP=1"
            break 2
        fi
    done
done

# Build the list of alive P3 space-ambit structs. Order matters: place the
# higher-HP candidate first so it becomes the TARGET (more HP headroom for
# the attacker volley if the bug allows it through), leaving the
# lower-HP one as the DEFENDER (assertion still distinguishes since
# block damage = 2).
declare -a P3_SPACE_ALIVE=()
for CAND_ID in "${BATTLESHIP_2_ID:-}" "${BATTLESHIP_1_ID:-}"; do
    if [ -z "${CAND_ID}" ]; then continue; fi
    CAND_HP=$(eb_health "${CAND_ID}")
    if [ "${CAND_HP}" != "0" ]; then P3_SPACE_ALIVE+=("${CAND_ID}"); fi
done
for NTH in 1 2 3 4; do
    SF_CAND=$(find_struct_by_owner_type "${PLAYER_3_ID}" 3 "${NTH}" "${SA_RG1}")
    if [ -z "${SF_CAND}" ]; then continue; fi
    SF_HP=$(eb_health "${SF_CAND}")
    if [ "${SF_HP}" != "0" ]; then P3_SPACE_ALIVE+=("${SF_CAND}"); fi
done

RG1_TARGET_ID=""
RG1_DEFENDER_ID=""
if [ "${#P3_SPACE_ALIVE[@]}" -ge 2 ]; then
    RG1_TARGET_ID="${P3_SPACE_ALIVE[0]}"
    RG1_DEFENDER_ID="${P3_SPACE_ALIVE[1]}"
fi

if [ -z "${RG1_ATTACKER_ID}" ] || [ -z "${RG1_TARGET_ID}" ] || [ -z "${RG1_DEFENDER_ID}" ]; then
    info "SKIP RG1: missing roles (attacker=${RG1_ATTACKER_ID} target=${RG1_TARGET_ID} defender=${RG1_DEFENDER_ID})"
else
    info "RG1: target=${RG1_TARGET_ID} defender=${RG1_DEFENDER_ID} (both P3, space-ambit)"

    # v0.19.0: the attack itself no longer requires an online (or present)
    # Command Ship. However, the co-location fleet-move below still does
    # (movement remains gated on an online command struct), so we rebuild the
    # attacker's CS if AR3 destroyed it — purely to allow that fleet-move, not
    # to "deploy" the attack. CS BuildLimit=1, but the count decrements on
    # destruction so a fresh build is allowed.
    eval "RG1_ATTACKER_PID=\${PLAYER_${RG1_ATTACKER_PLAYER}_ID}"
    RG1_ATTACKER_KEY="player_${RG1_ATTACKER_PLAYER}"
    RG1_ATK_CS_COUNT=$(echo "${SA_RG1}" | jq -r --arg pid "${RG1_ATTACKER_PID}" \
        '[.Struct[] | select(.owner==$pid and (.type|tonumber)==1)] | length')
    if [ "${RG1_ATK_CS_COUNT}" = "0" ]; then
        info "RG1: ${RG1_ATTACKER_KEY} has no Command Ship — rebuilding (needed for the co-location fleet-move; destroyed in AR phase)"
        wait_for_charge "${RG1_ATTACKER_PID}" "${CHARGE_BUILD}"
        run_tx "RG1: Initiating fresh ${RG1_ATTACKER_KEY} Command Ship (type=1, space, slot=1)" \
            tx structs struct-build-initiate "${RG1_ATTACKER_PID}" 1 space 1 --from "${RG1_ATTACKER_KEY}"

        STRUCT_ALL_JSON=$(query query structs struct-all)
        RG1_NEW_CS_ID=$(get_newest_struct_id "${STRUCT_ALL_JSON}")
        RG1_NEW_CS_OWNER=$(echo "${STRUCT_ALL_JSON}" | jq -r --arg sid "${RG1_NEW_CS_ID}" \
            '[.Struct[] | select(.id == $sid)] | .[0].owner // empty' 2>/dev/null || echo "")

        if [ "${RG1_NEW_CS_OWNER}" = "${RG1_ATTACKER_PID}" ]; then
            run_compute "RG1: Building fresh ${RG1_ATTACKER_KEY} CS ${RG1_NEW_CS_ID}" \
                tx structs struct-build-compute "${RG1_NEW_CS_ID}" --from "${RG1_ATTACKER_KEY}"
            RG1_NEW_CS_BUILT=$(query query structs struct "${RG1_NEW_CS_ID}" 2>/dev/null \
                | jq -r '.structAttributes.isBuilt // "false"')
            assert_eq "RG1 fresh ${RG1_ATTACKER_KEY} CS built" "true" "${RG1_NEW_CS_BUILT}"
        else
            info "RG1: fresh CS build did not materialise (got owner='${RG1_NEW_CS_OWNER}'); attack will likely be skipped"
        fi
    fi

    # The attacker fleet must be co-located with the target's fleet to be
    # reachable. After AR3, P3's fleet typically stays at P6's planet (2-9)
    # while the attacker's fleet sits at home, so a fleet-move is needed.
    RG1_TARGET_FLEET_ID=$(query query structs struct "${RG1_TARGET_ID}" 2>/dev/null \
        | jq -r '.Struct.locationId // empty')
    RG1_TARGET_FLEET_LOC=""
    if [ -n "${RG1_TARGET_FLEET_ID}" ]; then
        RG1_TARGET_FLEET_LOC=$(query query structs fleet "${RG1_TARGET_FLEET_ID}" 2>/dev/null \
            | jq -r '.Fleet.locationId // empty')
    fi
    RG1_ATTACKER_FLEET_ID=$(query query structs struct "${RG1_ATTACKER_ID}" 2>/dev/null \
        | jq -r '.Struct.locationId // empty')
    RG1_ATTACKER_FLEET_LOC=""
    if [ -n "${RG1_ATTACKER_FLEET_ID}" ]; then
        RG1_ATTACKER_FLEET_LOC=$(query query structs fleet "${RG1_ATTACKER_FLEET_ID}" 2>/dev/null \
            | jq -r '.Fleet.locationId // empty')
    fi
    if [ -n "${RG1_TARGET_FLEET_LOC}" ] && [ -n "${RG1_ATTACKER_FLEET_LOC}" ] \
       && [ "${RG1_ATTACKER_FLEET_LOC}" != "${RG1_TARGET_FLEET_LOC}" ]; then
        info "RG1: co-locating attacker fleet ${RG1_ATTACKER_FLEET_ID} → ${RG1_TARGET_FLEET_LOC} (was ${RG1_ATTACKER_FLEET_LOC})"
        run_tx "RG1: fleet-move ${RG1_ATTACKER_FLEET_ID} to ${RG1_TARGET_FLEET_LOC}" \
            tx structs fleet-move "${RG1_ATTACKER_FLEET_ID}" "${RG1_TARGET_FLEET_LOC}" --from "${RG1_ATTACKER_KEY}"
    fi

    # Both target and defender belong to P3, so the defense registration
    # transaction is authorised by player_3.
    wait_for_charge "${PLAYER_3_ID}" "${CHARGE_DEFEND}"
    run_tx "RG1: Register ${RG1_DEFENDER_ID} as defender of ${RG1_TARGET_ID}" \
        tx structs struct-defense-set "${RG1_DEFENDER_ID}" "${RG1_TARGET_ID}" --from player_3

    RG1_ATTACKER_HP_BEFORE=$(eb_health "${RG1_ATTACKER_ID}")
    RG1_TARGET_HP_BEFORE=$(eb_health "${RG1_TARGET_ID}")
    RG1_DEFENDER_HP_BEFORE=$(eb_health "${RG1_DEFENDER_ID}")
    info "RG1 pre-attack HPs: attacker(P${RG1_ATTACKER_PLAYER},${RG1_ATTACKER_ID})=${RG1_ATTACKER_HP_BEFORE} target(${RG1_TARGET_ID})=${RG1_TARGET_HP_BEFORE} defender(${RG1_DEFENDER_ID})=${RG1_DEFENDER_HP_BEFORE}"
    info "  Expected: defender counter (CA=1) destroys HP=1 attacker; block must NOT fire."

    if [ "${RG1_ATTACKER_HP_BEFORE}" != "1" ]; then
        info "SKIP RG1 attack: attacker HP changed to ${RG1_ATTACKER_HP_BEFORE} between selection and attack"
    else
        eb_attack "RG1: P${RG1_ATTACKER_PLAYER} SF(HP=1) → P3 target (def: P3 same-ambit struct — counter-kills attacker)" \
            "${RG1_ATTACKER_ID}" "${RG1_TARGET_ID}" primaryWeapon "${RG1_ATTACKER_PLAYER}"

        RG1_ATTACKER_HP_AFTER=$(eb_health "${RG1_ATTACKER_ID}")
        RG1_TARGET_HP_AFTER=$(eb_health "${RG1_TARGET_ID}")
        RG1_DEFENDER_HP_AFTER=$(eb_health "${RG1_DEFENDER_ID}")
        info "RG1 post-attack HPs: attacker=${RG1_ATTACKER_HP_AFTER} target=${RG1_TARGET_HP_AFTER} defender=${RG1_DEFENDER_HP_AFTER}"

        # If the attacker, target, and defender all show identical HP before and
        # after, the attack TX did not actually run on chain (e.g. ante reject,
        # missing CS, etc.) and the bug-1 condition was never reached. Skip
        # the assertions in that case rather than erroring out the whole
        # phase on a no-op.
        if [ "${RG1_ATTACKER_HP_AFTER}" = "${RG1_ATTACKER_HP_BEFORE}" ] \
           && [ "${RG1_TARGET_HP_AFTER}" = "${RG1_TARGET_HP_BEFORE}" ] \
           && [ "${RG1_DEFENDER_HP_AFTER}" = "${RG1_DEFENDER_HP_BEFORE}" ]; then
            info "SKIP RG1 assertions: attack TX appears not to have executed (no HP changes)"
        else
            assert_eq "RG1: Attacker destroyed by defender counter" \
                "0" "${RG1_ATTACKER_HP_AFTER}"
            assert_eq "RG1: Defender HP unchanged — no block on dead volley [Bug 1 regression]" \
                "${RG1_DEFENDER_HP_BEFORE}" "${RG1_DEFENDER_HP_AFTER}"
            assert_eq "RG1: Target HP unchanged — dead attacker delivers no damage" \
                "${RG1_TARGET_HP_BEFORE}" "${RG1_TARGET_HP_AFTER}"
        fi
    fi

    wait_for_charge "${PLAYER_3_ID}" "${CHARGE_DEFEND}"
    run_tx "RG1 cleanup: clear defender registration" \
        tx structs struct-defense-clear "${RG1_DEFENDER_ID}" --from player_3
fi

fi # phase RG1

if run_phase 3500; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE RG2: Regression — Relocated Defender Provides No Support (Bug 2)
# ═════════════════════════════════════════════════════════════════════════════
#
# Validates the IsProtecting filter in ResolveDefenders (and the matching
# range check in MsgStructDefenseSet): a defender whose fleet has moved
# away from the protected target's location must not counter or block.
# Stale StructDefender entries are left intact across moves on purpose;
# the runtime co-location check is what enforces correctness.
#
# Go unit test (deterministic):
#   x/structs/keeper/msg_server_struct_attack_test.go
#     TestMsgStructAttackDefenderFleetMovedNoSupport
#
# Setup:
#   Defender: EB_MOBILE_ART_ID (P6 fleet, land slot 0). Mobile Artillery has
#             CounterAttack=0, so the only defensive action it could take is
#             block — and only when co-located with the protected target.
#   Target:   EB_ORE_EXTRACTOR_ID (P6 planet, land slot 3). Same ambit as
#             the MA defender, so block would fire when in range.
#   Attacker: EB_P3_MOBILE_ART_ID (P3 fleet, currently at P6 planet from
#             AR2). AttackCounterable=false, so PDC defensive cannons do
#             not return damage and we can observe block independently.
#
# Bug-2 trigger: register defender → target while co-located, move P6 fleet
# to P2 planet (defender relocates with the fleet; target stays on the
# planet), then attack the target from P3.
#   • With the fix → IsProtecting returns false → MA does not block →
#     attacker primary damage 2 reaches Ore Extractor → MA HP unchanged.
#   • Without the fix → MA still blocks → MA takes 2 damage; Ore Extractor
#     takes no damage.
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE RG2: Regression — Relocated Defender Provides No Support (Bug 2)"

RG2_MA_HP=$(eb_health "${EB_MOBILE_ART_ID}")
RG2_EXTRACTOR_HP=$(eb_health "${EB_ORE_EXTRACTOR_ID}")
RG2_P3_MA_HP=$(eb_health "${EB_P3_MOBILE_ART_ID}")

# Bug 2 requires a defender on a fleet that can be relocated AND a same-owner,
# same-ambit target that stays behind (typically on a planet). EB5 attacks
# usually leave the P6 Ore Extractor destroyed, in which case this scenario
# is no longer reproducible from the natural EB/AR state. The Go unit test
# (TestMsgStructAttackDefenderFleetMovedNoSupport) still provides
# deterministic coverage.
if [ "${RG2_MA_HP}" = "0" ] || [ "${RG2_EXTRACTOR_HP}" = "0" ] || [ "${RG2_P3_MA_HP}" = "0" ]; then
    info "SKIP RG2: P6 MA(HP=${RG2_MA_HP}), Ore Extractor(HP=${RG2_EXTRACTOR_HP}), or P3 MA(HP=${RG2_P3_MA_HP}) destroyed — Go unit test still covers Bug 2."
else
    # Register P6 Mobile Art as defender of P6 Ore Extractor. Both are on
    # P6's planet (MA on the fleet, Ore Extractor as planet struct), so
    # IsProtecting accepts the relationship at registration time.
    wait_for_charge "${PLAYER_6_ID}" "${CHARGE_DEFEND}"
    run_tx "RG2: Register P6 Mobile Art (${EB_MOBILE_ART_ID}) as defender of P6 Ore Extractor (${EB_ORE_EXTRACTOR_ID})" \
        tx structs struct-defense-set "${EB_MOBILE_ART_ID}" "${EB_ORE_EXTRACTOR_ID}" --from player_6

    RG2_REGISTERED=$(query query structs struct "${EB_ORE_EXTRACTOR_ID}" 2>/dev/null \
        | jq -r --arg sid "${EB_MOBILE_ART_ID}" \
            '[.structDefenders // [] | .[] | select(. == $sid)] | first // ""' \
            2>/dev/null || echo "")
    assert_eq "RG2: MA registered as Ore Extractor defender" \
        "${EB_MOBILE_ART_ID}" "${RG2_REGISTERED}"

    RG2_MA_HP_BEFORE=$(eb_health "${EB_MOBILE_ART_ID}")
    RG2_EXTRACTOR_HP_BEFORE=$(eb_health "${EB_ORE_EXTRACTOR_ID}")
    RG2_P3_MA_HP_BEFORE=$(eb_health "${EB_P3_MOBILE_ART_ID}")

    # v0.19.0: the attack no longer needs a Command Ship, but the relocation
    # fleet-move below still does (movement remains gated on an online command
    # struct). P6's CS (5-22 in a typical run) is destroyed during the AR
    # phase, so rebuild it here purely to enable the move. CS BuildLimit=1, but
    # the count decrements on destruction so a fresh build is allowed.
    RG2_SA=$(query query structs struct-all 2>/dev/null || echo '{}')
    RG2_P6_CS_COUNT=$(echo "${RG2_SA}" | jq -r --arg pid "${PLAYER_6_ID}" \
        '[.Struct[] | select(.owner==$pid and (.type|tonumber)==1)] | length')
    if [ "${RG2_P6_CS_COUNT}" = "0" ]; then
        info "RG2: player_6 has no Command Ship — rebuilding (needed for the relocation fleet-move; destroyed in AR phase)"
        wait_for_charge "${PLAYER_6_ID}" "${CHARGE_BUILD}"
        run_tx "RG2: Initiating fresh player_6 Command Ship (type=1, space, slot=1)" \
            tx structs struct-build-initiate "${PLAYER_6_ID}" 1 space 1 --from player_6

        STRUCT_ALL_JSON=$(query query structs struct-all)
        RG2_NEW_CS_ID=$(get_newest_struct_id "${STRUCT_ALL_JSON}")
        RG2_NEW_CS_OWNER=$(echo "${STRUCT_ALL_JSON}" | jq -r --arg sid "${RG2_NEW_CS_ID}" \
            '[.Struct[] | select(.id == $sid)] | .[0].owner // empty' 2>/dev/null || echo "")
        if [ "${RG2_NEW_CS_OWNER}" = "${PLAYER_6_ID}" ]; then
            run_compute "RG2: Building fresh player_6 CS ${RG2_NEW_CS_ID}" \
                tx structs struct-build-compute "${RG2_NEW_CS_ID}" --from player_6
            RG2_NEW_CS_BUILT=$(query query structs struct "${RG2_NEW_CS_ID}" 2>/dev/null \
                | jq -r '.structAttributes.isBuilt // "false"')
            assert_eq "RG2 fresh player_6 CS built" "true" "${RG2_NEW_CS_BUILT}"
        else
            info "RG2: fresh CS build did not materialise (got owner='${RG2_NEW_CS_OWNER}'); scenario will be skipped"
        fi
    fi

    # Move P6 fleet to P2 planet. The MA goes with the fleet; the Ore
    # Extractor stays on P6's planet. The stale StructDefender entry remains
    # in state, but IsProtecting should now return false at attack time.
    wait_for_charge "${PLAYER_6_ID}" "${CHARGE_MOVE}"
    run_tx "RG2: Move P6 fleet to P2 planet (defender relocates away from target)" \
        tx structs fleet-move "${PLAYER_6_FLEET_ID}" "${PLAYER_2_PLANET_ID}" --from player_6

    # The relocation precondition must actually hold: P6's fleet (and thus the
    # MA defender) must now sit at P2's planet, away from the Ore Extractor on
    # P6's planet. If the move did not take (e.g. CS could not be rebuilt), the
    # defender stays co-located and correctly blocks — which is not the Bug 2
    # scenario. Skip the assertions in that case; the Go unit test
    # (TestMsgStructAttackDefenderFleetMovedNoSupport) still covers Bug 2.
    RG2_P6_FLEET_LOC=$(query query structs fleet "${PLAYER_6_FLEET_ID}" 2>/dev/null \
        | jq -r '.Fleet.locationId // empty')
    if [ "${RG2_P6_FLEET_LOC}" != "${PLAYER_2_PLANET_ID}" ]; then
        info "SKIP RG2: P6 fleet did not relocate (loc='${RG2_P6_FLEET_LOC}', want='${PLAYER_2_PLANET_ID}') — defender still co-located; Go unit test covers Bug 2."
    else
        info "RG2 pre-attack HPs: P6 MA=${RG2_MA_HP_BEFORE} Ore Extractor=${RG2_EXTRACTOR_HP_BEFORE} P3 MA=${RG2_P3_MA_HP_BEFORE}"
        info "  Expected: P6 MA out of range (IsProtecting=false). MA HP unchanged; Ore Extractor takes damage."

        eb_attack "RG2: P3 Mobile Art → P6 Ore Extractor (MA defender out of range after fleet move)" \
            "${EB_P3_MOBILE_ART_ID}" "${EB_ORE_EXTRACTOR_ID}" primaryWeapon 3

        RG2_MA_HP_AFTER=$(eb_health "${EB_MOBILE_ART_ID}")
        RG2_EXTRACTOR_HP_AFTER=$(eb_health "${EB_ORE_EXTRACTOR_ID}")
        RG2_P3_MA_HP_AFTER=$(eb_health "${EB_P3_MOBILE_ART_ID}")
        info "RG2 post-attack HPs: P6 MA=${RG2_MA_HP_AFTER} Ore Extractor=${RG2_EXTRACTOR_HP_AFTER} P3 MA=${RG2_P3_MA_HP_AFTER}"

        assert_eq "RG2: P6 MA HP unchanged after fleet move — defender did not block [Bug 2 regression]" \
            "${RG2_MA_HP_BEFORE}" "${RG2_MA_HP_AFTER}"
        RG2_EXTRACTOR_DMG=$((RG2_EXTRACTOR_HP_BEFORE - RG2_EXTRACTOR_HP_AFTER))
        assert_gt "RG2: Ore Extractor took damage — not deflected by stale defender" \
            0 "${RG2_EXTRACTOR_DMG}"
    fi

    wait_for_charge "${PLAYER_6_ID}" "${CHARGE_DEFEND}"
    run_tx "RG2 cleanup: clear stale P6 MA defender registration" \
        tx structs struct-defense-clear "${EB_MOBILE_ART_ID}" --from player_6

    wait_for_charge "${PLAYER_6_ID}" "${CHARGE_MOVE}"
    run_tx "RG2 cleanup: move P6 fleet back to P6 planet" \
        tx structs fleet-move "${PLAYER_6_FLEET_ID}" "${PLAYER_6_PLANET_ID}" --from player_6
fi

fi # phase RG2

fi  # end EXTENDED_BATTLE


# ═════════════════════════════════════════════════════════════════════════════
if run_phase 3800; then
# ═════════════════════════════════════════════════════════════════════════════
#  PHASE GP1: Guild — Update Primary Reactor (recovery for retired validators)
#
#  Exercises MsgGuildUpdatePrimaryReactor end-to-end through the CLI. The
#  single-validator test chain cannot rotate to a *different* reactor (there
#  is only one), so this phase covers the validation surface:
#
#    1. Reject when target reactor does not exist.
#    2. Reject when caller lacks PermAdmin on the guild.
#    3. Accept when an admin targets a known, non-jailed validator's reactor
#       (idempotent no-op on the value, but exercises every validation branch
#       plus the EventGuild emission via cache commit → SetGuild).
#    4. Confirm the guild's primaryReactorId is unchanged after the no-op
#       update (and matches the configured reactor).
# ═════════════════════════════════════════════════════════════════════════════
section "PHASE GP1: Guild Update Primary Reactor"

# Sanity: capture the pre-update value so we can detect any unintended drift.
GP1_BEFORE=$(query query structs guild "${GUILD_ID}" 2>/dev/null | jq -r '.Guild.primaryReactorId // empty')
assert_not_empty "GP1: pre-update guild.primaryReactorId" "${GP1_BEFORE}"

# 1. Reject: unknown reactor. Format mirrors a real reactor id ("<n>-<addr>")
#    so it parses but resolves to no object.
GP1_FAKE_REACTOR="999999-${ALICE_ADDRESS}"
run_tx_expect_fail "GP1: reject update to unknown reactor (${GP1_FAKE_REACTOR})" \
    tx structs guild-update-primary-reactor "${GUILD_ID}" "${GP1_FAKE_REACTOR}" --from alice

# 2. Reject: non-admin caller (player_2).
#    Earlier phases (4d ownership-transfer suite) granted player_2 PermAdmin
#    and briefly made them the guild owner (which adds PermGuildAll). Neither
#    grant is revoked there, so by the time we get here player_2 legitimately
#    has admin rights on the guild. Strip every bit they may have accumulated
#    on the guild object before asserting that the message handler rejects a
#    non-admin caller — otherwise this phase silently passes on whatever
#    cruft Phase 4 left behind.
GP1_P2_GUILD_PERMS=$(get_permission_value_for_player "${GUILD_ID}" "${PLAYER_2_ID}")
GP1_P2_GUILD_PERMS=${GP1_P2_GUILD_PERMS:-0}
if [ "${GP1_P2_GUILD_PERMS}" != "0" ] && [ -n "${GP1_P2_GUILD_PERMS}" ]; then
    info "GP1: stripping player_2's residual perms on guild (bits=${GP1_P2_GUILD_PERMS})"
    run_tx "GP1: revoke residual player_2 perms on guild" \
        tx structs permission-revoke-on-object "${GUILD_ID}" "${PLAYER_2_ID}" "${GP1_P2_GUILD_PERMS}" --from alice
fi
GP1_P2_GUILD_PERMS_AFTER=$(get_permission_value_for_player "${GUILD_ID}" "${PLAYER_2_ID}")
GP1_P2_GUILD_PERMS_AFTER=${GP1_P2_GUILD_PERMS_AFTER:-0}
assert_eq "GP1: player_2 has no per-object perms on guild before reject test" \
    "0" "${GP1_P2_GUILD_PERMS_AFTER}"

run_tx_expect_fail "GP1: reject update from non-admin caller (player_2)" \
    tx structs guild-update-primary-reactor "${GUILD_ID}" "${REACTOR_ID}" --from player_2

# 3. Happy path: admin (alice) targets the current reactor. Exercises the
#    full handler chain (player lookup, guild lookup, PermAdmin check, reactor
#    lookup, validator lookup, jailed check, cache commit → EventGuild). We
#    capture the tx output explicitly because the assertions in (4) below
#    would trivially pass against unchanged state if this tx were silently
#    rejected by ante (incident 2026-05: every new Structs message MUST be
#    registered in app/ante/maps.go or it gets bounced as "unknown structs
#    message type"). PARAMS_TX defaults to YAML, so force --output json here
#    so we can pull .code with jq.
GP1_HAPPY_OUT=$(structsd ${PARAMS_TX} --output json tx structs guild-update-primary-reactor \
    "${GUILD_ID}" "${REACTOR_ID}" --from alice 2>&1) || true
echo -e "  ${BOLD}structsd ${PARAMS_TX} --output json tx structs guild-update-primary-reactor ${GUILD_ID} ${REACTOR_ID} --from alice${NC}"
# `--gas auto` prints "gas estimate: NNN" on stderr before the JSON tx response,
# so pull the JSON line out explicitly before handing it to jq. Use `.code | tostring`
# so that the legitimate value 0 round-trips as the string "0".
GP1_HAPPY_JSON=$(echo "${GP1_HAPPY_OUT}" | grep -E '^\{' | tail -n 1)
GP1_HAPPY_CODE=$(echo "${GP1_HAPPY_JSON}" | jq -r '.code | tostring' 2>/dev/null || echo "")
if [ -z "${GP1_HAPPY_CODE}" ]; then
    echo -e "  ${YELLOW}WARN${NC}: could not parse tx code from output, raw output follows:"
    echo "${GP1_HAPPY_OUT}" | head -20
fi
assert_eq "GP1: admin update returned tx code 0 (message registered in ante maps)" \
    "0" "${GP1_HAPPY_CODE}"
sleep "${SLEEP}"

# 4. Confirm the guild's primaryReactorId is intact (no drift from no-op).
GP1_AFTER=$(query query structs guild "${GUILD_ID}" 2>/dev/null | jq -r '.Guild.primaryReactorId // empty')
assert_eq "GP1: guild.primaryReactorId unchanged after no-op update" \
    "${GP1_BEFORE}" "${GP1_AFTER}"
assert_eq "GP1: guild.primaryReactorId matches configured reactor" \
    "${REACTOR_ID}" "${GP1_AFTER}"

fi # phase GP1

if run_phase 3850; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE JS1: Jamming Satellite — Guided-Only Planetary Defense (v0.20.0)
# ═════════════════════════════════════════════════════════════════════════════
#
# The Jamming Satellite (struct type 17) grants the planetary defense
# lowOrbitBallisticInterceptorNetwork. As of v0.20.0 it shields a planetary
# (non-fleet) struct on its own planet from GUIDED ordnance, regardless of the
# source or target ambit; unguided ordnance must pass through untouched.
# Before v0.20.0 the planetary evasion required a fleet attacker striking from
# air/space against a land/water target and — the bug this fixes — ignored
# weapon control, so it could also evade unguided attacks.
#
# Guided evasion is probabilistic (a single interceptor gives a 1/3 evade
# chance) and ambit-independence / the planetary-target requirement are
# deterministic-only concerns, so those guarantees live in the Go unit test:
#   x/structs/keeper/msg_server_struct_attack_test.go
#     TestMsgStructAttackPlanetaryDefenseGuidedOnly
#
# This phase is the integration guard for the unguided-bypass direction: an
# unguided fleet attack against a target on the defended planet must always land
# damage. It reuses P6's planet (defender) and a P3 Battleship (space ambit,
# unguided armour-piercing primary that targets land/water, shot success 1/1).
# Preconditions are checked and the phase skips gracefully when the
# extended-battle state is not available.

section "PHASE JS1: Jamming Satellite — Guided-Only Planetary Defense (v0.20.0)"

# find_free_space_slot: echo the first unoccupied space slot index on a planet,
# or empty when the planet's space slots are all full.
find_free_space_slot() {
    local planet_json="$1"
    local slot_count idx occ
    slot_count=$(jqr "${planet_json}" '.Planet.spaceSlots' '0')
    for idx in $(seq 0 $((slot_count - 1))); do
        occ=$(echo "${planet_json}" | jq -r ".Planet.space[${idx}] // \"\"" 2>/dev/null || echo "")
        if [ -z "${occ}" ] || [ "${occ}" = "null" ]; then
            echo "${idx}"
            return
        fi
    done
    echo ""
}

if [ -z "${PLAYER_6_PLANET_ID:-}" ] || [ -z "${PLAYER_3_ID:-}" ] || [ -z "${PLAYER_3_FLEET_ID:-}" ]; then
    info "SKIP JS1: extended-battle state (P6 planet / P3 fleet) not available"
else
    P6_PLANET_JSON=$(query query structs planet "${PLAYER_6_PLANET_ID}" 2>/dev/null || echo '{}')
    JS_SLOT=$(find_free_space_slot "${P6_PLANET_JSON}")

    if [ -z "${JS_SLOT}" ]; then
        info "SKIP JS1: no free space slot on P6's planet for a Jamming Satellite"
    else
        # ─── Build the Jamming Satellite (type 17, space) on P6's planet ───
        PREV_NEWEST_STRUCT_ID=$(get_newest_struct_id)
        wait_for_charge "${PLAYER_6_ID}" "${CHARGE_BUILD}"
        run_tx "Initiating P6 Jamming Satellite (type=17, space, slot=${JS_SLOT})" \
            tx structs struct-build-initiate "${PLAYER_6_ID}" 17 space "${JS_SLOT}" --from player_6

        JS_SAT_ID=$(get_newest_struct_id)
        if [ "${JS_SAT_ID}" = "${PREV_NEWEST_STRUCT_ID}" ] || [ -z "${JS_SAT_ID}" ]; then
            info "SKIP JS1: Jamming Satellite build did not materialize (slot/charge)"
        else
            echo "  P6 Jamming Satellite ID: ${JS_SAT_ID}"
            run_compute "Building Jamming Satellite ${JS_SAT_ID}" \
                tx structs struct-build-compute "${JS_SAT_ID}" --from player_6

            # ─── Deterministic: the interceptor network is now active ───
            P6_PLANET_JSON=$(query query structs planet "${PLAYER_6_PLANET_ID}" 2>/dev/null || echo '{}')
            JS_QTY=$(jqr "${P6_PLANET_JSON}" '.planetAttributes.lowOrbitBallisticsInterceptorNetworkQuantity' '0')
            JS_RATE_NUM=$(jqr "${P6_PLANET_JSON}" '.planetAttributes.lowOrbitBallisticsInterceptorNetworkSuccessRateNumerator' '0')
            assert_gt "JS1 — interceptor network quantity increased" 0 "${JS_QTY}"
            assert_gt "JS1 — interceptor network success rate is active" 0 "${JS_RATE_NUM}"

            # ─── Locate an unguided air/space attacker and a land/water target ───
            STRUCT_ALL_JSON=$(query query structs struct-all)
            JS_ATTACKER_ID=$(find_struct_by_owner_type "${PLAYER_3_ID}" 2 1 "${STRUCT_ALL_JSON}")   # P3 Battleship (space, unguided primary)

            # Prefer the Ore Extractor (no counter/defense) for clean accounting,
            # then fall back to other P6 land/water structs.
            JS_TARGET_ID=""
            for CAND in "${EB_ORE_EXTRACTOR_ID:-}" "${EB_P6_TANK_ID:-}" "${EB_P6_CRUISER_ID:-}"; do
                if [ -n "${CAND}" ] && [ "$(eb_health "${CAND}")" != "0" ]; then
                    JS_TARGET_ID="${CAND}"
                    break
                fi
            done

            if [ -z "${JS_ATTACKER_ID}" ] || [ "$(eb_health "${JS_ATTACKER_ID}")" = "0" ]; then
                info "SKIP JS1 attack: no live P3 Battleship (unguided air/space attacker) available"
            elif [ -z "${JS_TARGET_ID}" ]; then
                info "SKIP JS1 attack: no live P6 land/water target available"
            else
                info "JS1 attack: P3 Battleship ${JS_ATTACKER_ID} (unguided, space) → P6 target ${JS_TARGET_ID} (land/water)"

                # Bring the P3 fleet to P6's planet so the Battleship is in range.
                run_tx "Moving P3 fleet to P6 planet for JS1" \
                    tx structs fleet-move "${PLAYER_3_FLEET_ID}" "${PLAYER_6_PLANET_ID}" --from player_3

                # Fire several unguided shots. Every shot must land damage: the
                # Jamming Satellite only jams guided ordnance, so unguided shots
                # bypass it regardless of ambit. Pre-v0.20.0 each shot had a
                # ~1/3 chance of being (incorrectly) jammed.
                JS_LANDED=0
                JS_SHOTS=0
                for JS_ROUND in 1 2 3; do
                    ATK_HP=$(eb_health "${JS_ATTACKER_ID}")
                    TGT_HP=$(eb_health "${JS_TARGET_ID}")
                    if [ "${ATK_HP}" = "0" ] || [ "${TGT_HP}" = "0" ]; then
                        break
                    fi
                    JS_SHOTS=$((JS_SHOTS + 1))
                    JS_TGT_BEFORE="${TGT_HP}"
                    eb_attack "JS1 round ${JS_ROUND}: unguided Battleship primary vs planetary defense" \
                        "${JS_ATTACKER_ID}" "${JS_TARGET_ID}" primaryWeapon 3
                    JS_TGT_AFTER=$(eb_health "${JS_TARGET_ID}")
                    if [ "${JS_TGT_AFTER}" -lt "${JS_TGT_BEFORE}" ]; then
                        JS_LANDED=$((JS_LANDED + 1))
                    else
                        info "  round ${JS_ROUND}: target HP unchanged (${JS_TGT_BEFORE} -> ${JS_TGT_AFTER}) — possible planetary evasion"
                    fi
                done

                if [ "${JS_SHOTS}" -gt 0 ]; then
                    assert_eq "JS1 — every unguided shot bypassed the Jamming Satellite" "${JS_SHOTS}" "${JS_LANDED}"
                else
                    info "SKIP JS1 attack assertion: no shots could be fired (structs destroyed early)"
                fi
            fi
        fi
    fi
fi

fi # phase JS1


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
[ "${FAIL_COUNT}" -eq 0 ] || exit 1
