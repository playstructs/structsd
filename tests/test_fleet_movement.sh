#!/usr/bin/env bash
#
# Fleet Movement & Range-Based Combat Test Script
#
# Tests the fleet linked-list structure on planets and attack range rules:
#   1. Creates 5 players with planets and fleets
#   2. Moves all 5 fleets to the same target planet
#   3. Verifies the doubly-linked list pointers (forward/backward) and
#      planet's locationListStart/locationListLast
#   4. Tests range-based combat: command ships can hit adjacent fleets
#      but NOT fleets further away in the list
#   5. Tests the home fleet special rule: the planet owner's fleet
#      (on station) can only target the first fleet in the list
#
# Prerequisites:
#   - structsd chain running locally (fresh chain recommended)
#   - 'alice' key in keyring (genesis validator)
#   - 'bob' key in keyring (faucet / bank sender)
#
# Fleet linked list structure (after all moves to Player 1's planet):
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
#
# Range rules:
#   Away fleets (middle/back): can only attack adjacent (forward/backward) fleets
#   Away fleet (front, fwd=""): can attack ANY target on the same planet
#   Home fleet (on station):   can only attack the first fleet (locationListStart)
#   Destroyed structs are wiped from chain (query returns "not found")
#

set -euo pipefail

# ─── Configuration ────────────────────────────────────────────────────────────

SLEEP=2
PARAMS_TX="--home ~/.structs --keyring-dir ~/.structs --keyring-backend test --gas auto --yes=true"
PARAMS_QUERY="--home ~/.structs --output json"
PARAMS_KEYS="--home ~/.structs --keyring-dir ~/.structs --keyring-backend test --output json"

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

get_block_height() {
    query query structs block-height | jq -r '.blockHeight // "0"' 2>/dev/null || echo "0"
}

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

get_latest_allocation_for_source() {
    local source_id="$1"
    query query structs allocation-all-by-source "${source_id}" | jq -r '.Allocation[-1].id // empty'
}

# query_fleet: return fleet JSON
query_fleet() {
    query query structs fleet "$1"
}

# query_planet: return planet JSON
query_planet() {
    query query structs planet "$1"
}

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

CHARGE_ATTACK=1
CHARGE_MOVE=8

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
#  PHASE 1: Initial Setup — Validator, Player 1, Guild, Substation
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 1: Initial Setup"

info "Looking up validator"
VALIDATOR_ADDRESS=$(query query staking validators | jq -r ".validators[0].operator_address")
assert_not_empty "Validator address" "${VALIDATOR_ADDRESS}"

info "Looking up Player 1 (alice)"
PLAYER_1_ADDRESS=$(structsd ${PARAMS_KEYS} keys show alice | jq -r .address)
assert_not_empty "Player 1 address" "${PLAYER_1_ADDRESS}"

PLAYER_ME_JSON=$(query query structs player-me)
PLAYER_1_ID=$(jqr "${PLAYER_ME_JSON}" '.Player.id')
assert_not_empty "Player 1 ID" "${PLAYER_1_ID}"
echo "  Player 1 ID: ${PLAYER_1_ID}"

ALICE_ADDRESS="${PLAYER_1_ADDRESS}"
PLAYER_1_CAPACITY=$(jqr "${PLAYER_ME_JSON}" '.gridAttributes.capacity')
echo "  Player 1 Capacity: ${PLAYER_1_CAPACITY}"

run_tx "Creating allocation from Player 1" \
    tx structs allocation-create "${PLAYER_1_ID}" "${PLAYER_1_CAPACITY}" \
    --allocation-type dynamic --from alice

P1_ALLOC_ID=$(get_latest_allocation_for_source "${PLAYER_1_ID}")
assert_not_empty "Player 1 allocation" "${P1_ALLOC_ID}"

run_tx "Creating Substation" \
    tx structs substation-create "${PLAYER_1_ID}" "${P1_ALLOC_ID}" --from alice

P1_ALLOC_JSON=$(query query structs allocation "${P1_ALLOC_ID}")
SUBSTATION_ID=$(jqr "${P1_ALLOC_JSON}" '.Allocation.destinationId')
assert_not_empty "Substation ID" "${SUBSTATION_ID}"
echo "  Substation ID: ${SUBSTATION_ID}"

run_tx "Creating Guild" \
    tx structs guild-create "fleet-test" "${SUBSTATION_ID}" --from alice

GUILD_ALL_JSON=$(query query structs guild-all)
GUILD_ID=$(jqr "${GUILD_ALL_JSON}" '.Guild[0].id')
REACTOR_ID=$(jqr "${GUILD_ALL_JSON}" '.Guild[0].primaryReactorId')
assert_not_empty "Guild ID" "${GUILD_ID}"
assert_not_empty "Reactor ID" "${REACTOR_ID}"
echo "  Guild: ${GUILD_ID}  Reactor: ${REACTOR_ID}"


# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 2: Create 5 Fleet Players — Fund, Delegate, Join Guild
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 2: Create Fleet Players (1-5)"

BOB_ADDRESS=$(structsd ${PARAMS_KEYS} keys show bob | jq -r .address)

for P in 1 2 3 4 5; do
    PLAYER_KEY="fplayer_${P}"
    info "Setting up ${PLAYER_KEY}"

    EXISTING=$(structsd ${PARAMS_KEYS} keys show "${PLAYER_KEY}" 2>/dev/null | jq -r .address || echo "")
    if [ -z "${EXISTING}" ]; then
        ADDR=$(structsd ${PARAMS_KEYS} keys add "${PLAYER_KEY}" | jq -r .address)
        echo "  Created ${PLAYER_KEY}: ${ADDR}"
    else
        ADDR="${EXISTING}"
        echo "  Reusing ${PLAYER_KEY}: ${ADDR}"
    fi
    eval "FP_${P}_ADDRESS=${ADDR}"

    run_tx "Funding ${PLAYER_KEY}" \
        tx bank send "${BOB_ADDRESS}" "${ADDR}" 10000000ualpha --from bob

    run_tx "Delegating for ${PLAYER_KEY}" \
        tx staking delegate "${VALIDATOR_ADDRESS}" 5000000ualpha --from "${PLAYER_KEY}"

    ADDR_JSON=$(query query structs address "${ADDR}")
    PID=$(jqr "${ADDR_JSON}" '.playerId')
    eval "FP_${P}_ID=${PID}"
    assert_not_empty "Fleet Player ${P} ID" "${PID}"
    echo "  Fleet Player ${P} ID: ${PID}"

    PJSON=$(query query structs player "${PID}")
    PCAP=$(jqr "${PJSON}" '.gridAttributes.capacity')

    run_tx "Creating allocation for fleet player ${P}" \
        tx structs allocation-create "${PID}" "${PCAP}" \
        --controller "${ALICE_ADDRESS}" --allocation-type dynamic --from "${PLAYER_KEY}"

    ALLOC_ID=$(get_latest_allocation_for_source "${PID}")
    eval "FP_${P}_ALLOC_ID=${ALLOC_ID}"

    run_tx "Fleet player ${P} joining guild" \
        tx structs guild-membership-join "${GUILD_ID}" "${REACTOR_ID}-${ADDR}" --from "${PLAYER_KEY}"

    run_tx "Connecting fleet player ${P} allocation to substation" \
        tx structs substation-allocation-connect "${ALLOC_ID}" "${SUBSTATION_ID}" --from alice
done

echo ""
info "All fleet players:"
for P in 1 2 3 4 5; do
    eval "echo \"  FP ${P}: ID=\${FP_${P}_ID} ADDR=\${FP_${P}_ADDRESS}\""
done


# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 3: Planet Exploration — Each Player Gets a Planet & Fleet
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 3: Planet Exploration"

for P in 1 2 3 4 5; do
    eval "PID=\${FP_${P}_ID}"
    PKEY="fplayer_${P}"

    run_tx "Fleet Player ${P} exploring planet" \
        tx structs planet-explore "${PID}" --from "${PKEY}"

    PJSON=$(query query structs player "${PID}")
    PLANET_ID=$(jqr "${PJSON}" '.Player.planetId')
    FLEET_ID=$(jqr "${PJSON}" '.Player.fleetId')
    eval "FP_${P}_PLANET_ID=${PLANET_ID}"
    eval "FP_${P}_FLEET_ID=${FLEET_ID}"
    assert_not_empty "FP ${P} planet" "${PLANET_ID}"
    assert_not_empty "FP ${P} fleet" "${FLEET_ID}"
    echo "  FP ${P}: planet=${PLANET_ID} fleet=${FLEET_ID}"
done


# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 4: Confirm Command Ships
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 4: Verify Command Ships"

for P in 1 2 3 4 5; do
    eval "FLEET_ID=\${FP_${P}_FLEET_ID}"
    FLEET_JSON=$(query_fleet "${FLEET_ID}")
    CMD_STRUCT=$(jqr "${FLEET_JSON}" '.Fleet.commandStruct')
    eval "CS_${P}=${CMD_STRUCT}"
    assert_not_empty "FP ${P} command ship" "${CMD_STRUCT}"

    STRUCT_JSON=$(query query structs struct "${CMD_STRUCT}")
    BUILT=$(echo "${STRUCT_JSON}" | jq -r '.structAttributes.isBuilt // "false"' 2>/dev/null)
    ONLINE=$(echo "${STRUCT_JSON}" | jq -r '.structAttributes.isOnline // "false"' 2>/dev/null)
    HP=$(echo "${STRUCT_JSON}" | jq -r '.structAttributes.health // "0"' 2>/dev/null)
    STYPE=$(echo "${STRUCT_JSON}" | jq -r '.Struct.type // ""' 2>/dev/null)
    assert_eq "CS ${P} type" "1" "${STYPE}"
    assert_eq "CS ${P} built" "true" "${BUILT}"
    assert_eq "CS ${P} online" "true" "${ONLINE}"
    echo "  CS_${P}=${CMD_STRUCT}  HP=${HP}  built=${BUILT}  online=${ONLINE}"
done


# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 5: Verify Initial Fleet State — Each Fleet on Its Own Planet
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 5: Initial Fleet State"

for P in 1 2 3 4 5; do
    eval "FLEET_ID=\${FP_${P}_FLEET_ID}"
    eval "PLANET_ID=\${FP_${P}_PLANET_ID}"

    FLEET_JSON=$(query_fleet "${FLEET_ID}")
    LOC=$(jqr "${FLEET_JSON}" '.Fleet.locationId')
    STATUS=$(jqr "${FLEET_JSON}" '.Fleet.status')
    FWD=$(jqr "${FLEET_JSON}" '.Fleet.locationListForward')
    BWD=$(jqr "${FLEET_JSON}" '.Fleet.locationListBackward')

    assert_eq "Fleet ${P} location" "${PLANET_ID}" "${LOC}"
    # proto3 omits zero-value enums: onStation=0 appears as empty string
    if [ -z "${STATUS}" ]; then STATUS="onStation"; fi
    assert_eq "Fleet ${P} status" "onStation" "${STATUS}"
    echo "  Fleet ${FLEET_ID}: loc=${LOC} status=${STATUS} fwd='${FWD}' bwd='${BWD}'"
done


# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 6: Move Fleets 2-5 to Player 1's Planet
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 6: Fleet Movement to Player 1's Planet"

TARGET_PLANET="${FP_1_PLANET_ID}"
info "Target planet: ${TARGET_PLANET} (FP 1's home)"

# Move fleets in order: 2, 3, 4, 5
# Each fleet appends to the END of the linked list
# Expected linked list after all moves:
#   Planet.locationListStart = F2
#   F2 (fwd="", bwd=F3)     ← front of list (first to arrive)
#   F3 (fwd=F2, bwd=F4)
#   F4 (fwd=F3, bwd=F5)
#   F5 (fwd=F4, bwd="")     ← back of list (last to arrive)
#   Planet.locationListLast = F5
#
#   F1 stays home (on station), NOT in the linked list

for P in 2 3 4 5; do
    eval "PID=\${FP_${P}_ID}"
    eval "FLEET_ID=\${FP_${P}_FLEET_ID}"
    PKEY="fplayer_${P}"

    wait_for_charge "${PID}" "${CHARGE_MOVE}"
    run_tx "Moving Fleet ${P} (${FLEET_ID}) to planet ${TARGET_PLANET}" \
        tx structs fleet-move "${FLEET_ID}" "${TARGET_PLANET}" --from "${PKEY}"
done

info "Waiting for state to settle"
sleep 3


# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 7: Verify Linked List Structure
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 7: Verify Fleet Linked List"

F1="${FP_1_FLEET_ID}"
F2="${FP_2_FLEET_ID}"
F3="${FP_3_FLEET_ID}"
F4="${FP_4_FLEET_ID}"
F5="${FP_5_FLEET_ID}"

info "Expected list on planet ${TARGET_PLANET}:"
echo "  Planet.start → ${F2} ↔ ${F3} ↔ ${F4} ↔ ${F5} ← Planet.last"
echo "  ${F1} is on station (home fleet, not in list)"
echo ""

# ─── Planet pointers ───
info "Checking planet linked list pointers"
PLANET_START=$(get_planet_field "${TARGET_PLANET}" "locationListStart")
PLANET_LAST=$(get_planet_field "${TARGET_PLANET}" "locationListLast")
assert_eq "Planet locationListStart" "${F2}" "${PLANET_START}"
assert_eq "Planet locationListLast"  "${F5}" "${PLANET_LAST}"

# ─── Fleet 1 (home, on station — not in the list) ───
info "Fleet 1 (home fleet)"
F1_JSON=$(query_fleet "${F1}")
F1_LOC=$(jqr "${F1_JSON}" '.Fleet.locationId')
F1_STATUS=$(jqr "${F1_JSON}" '.Fleet.status')
F1_FWD=$(jqr "${F1_JSON}" '.Fleet.locationListForward')
F1_BWD=$(jqr "${F1_JSON}" '.Fleet.locationListBackward')
assert_eq "Fleet 1 location" "${FP_1_PLANET_ID}" "${F1_LOC}"
if [ -z "${F1_STATUS}" ]; then F1_STATUS="onStation"; fi
assert_eq "Fleet 1 status" "onStation" "${F1_STATUS}"
echo "  F1: loc=${F1_LOC} status=${F1_STATUS} fwd='${F1_FWD}' bwd='${F1_BWD}'"

# ─── Fleet 2 (front of list) ───
info "Fleet 2 (front of list)"
F2_JSON=$(query_fleet "${F2}")
F2_LOC=$(jqr "${F2_JSON}" '.Fleet.locationId')
F2_STATUS=$(jqr "${F2_JSON}" '.Fleet.status')
F2_FWD=$(jqr "${F2_JSON}" '.Fleet.locationListForward')
F2_BWD=$(jqr "${F2_JSON}" '.Fleet.locationListBackward')
assert_eq "Fleet 2 location" "${TARGET_PLANET}" "${F2_LOC}"
assert_eq "Fleet 2 status" "away" "${F2_STATUS}"
assert_eq "Fleet 2 forward (toward planet)" "" "${F2_FWD}"
assert_eq "Fleet 2 backward" "${F3}" "${F2_BWD}"
echo "  F2: loc=${F2_LOC} status=${F2_STATUS} fwd='${F2_FWD}' bwd='${F2_BWD}'"

# ─── Fleet 3 ───
info "Fleet 3 (second in list)"
F3_JSON=$(query_fleet "${F3}")
F3_LOC=$(jqr "${F3_JSON}" '.Fleet.locationId')
F3_STATUS=$(jqr "${F3_JSON}" '.Fleet.status')
F3_FWD=$(jqr "${F3_JSON}" '.Fleet.locationListForward')
F3_BWD=$(jqr "${F3_JSON}" '.Fleet.locationListBackward')
assert_eq "Fleet 3 location" "${TARGET_PLANET}" "${F3_LOC}"
assert_eq "Fleet 3 status" "away" "${F3_STATUS}"
assert_eq "Fleet 3 forward" "${F2}" "${F3_FWD}"
assert_eq "Fleet 3 backward" "${F4}" "${F3_BWD}"
echo "  F3: loc=${F3_LOC} status=${F3_STATUS} fwd='${F3_FWD}' bwd='${F3_BWD}'"

# ─── Fleet 4 ───
info "Fleet 4 (third in list)"
F4_JSON=$(query_fleet "${F4}")
F4_LOC=$(jqr "${F4_JSON}" '.Fleet.locationId')
F4_STATUS=$(jqr "${F4_JSON}" '.Fleet.status')
F4_FWD=$(jqr "${F4_JSON}" '.Fleet.locationListForward')
F4_BWD=$(jqr "${F4_JSON}" '.Fleet.locationListBackward')
assert_eq "Fleet 4 location" "${TARGET_PLANET}" "${F4_LOC}"
assert_eq "Fleet 4 status" "away" "${F4_STATUS}"
assert_eq "Fleet 4 forward" "${F3}" "${F4_FWD}"
assert_eq "Fleet 4 backward" "${F5}" "${F4_BWD}"
echo "  F4: loc=${F4_LOC} status=${F4_STATUS} fwd='${F4_FWD}' bwd='${F4_BWD}'"

# ─── Fleet 5 (back of list) ───
info "Fleet 5 (back of list)"
F5_JSON=$(query_fleet "${F5}")
F5_LOC=$(jqr "${F5_JSON}" '.Fleet.locationId')
F5_STATUS=$(jqr "${F5_JSON}" '.Fleet.status')
F5_FWD=$(jqr "${F5_JSON}" '.Fleet.locationListForward')
F5_BWD=$(jqr "${F5_JSON}" '.Fleet.locationListBackward')
assert_eq "Fleet 5 location" "${TARGET_PLANET}" "${F5_LOC}"
assert_eq "Fleet 5 status" "away" "${F5_STATUS}"
assert_eq "Fleet 5 forward" "${F4}" "${F5_FWD}"
assert_eq "Fleet 5 backward" "" "${F5_BWD}"
echo "  F5: loc=${F5_LOC} status=${F5_STATUS} fwd='${F5_FWD}' bwd='${F5_BWD}'"

echo ""
info "Linked list diagram (verified):"
echo "  Planet(${TARGET_PLANET}).start=${PLANET_START}"
echo "    ${F2} fwd='' bwd=${F2_BWD}"
echo "    ${F3} fwd=${F3_FWD} bwd=${F3_BWD}"
echo "    ${F4} fwd=${F4_FWD} bwd=${F4_BWD}"
echo "    ${F5} fwd=${F5_FWD} bwd=''"
echo "  Planet(${TARGET_PLANET}).last=${PLANET_LAST}"
echo "  (Home) ${F1} status=${F1_STATUS}"


# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 8: Adjacent Attacks (Should Succeed)
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 8: Adjacent Fleet Attacks (Should Succeed)"

# List: F2 ↔ F3 ↔ F4 ↔ F5, F1 on station
# Command Ship: 6 HP, 2 dmg primary, 2 dmg strongCounterAttack
# We test 3 adjacency types: home→front, forward, backward
# Keeping attacks minimal so all ships survive for later phases.

# Helper: query HP (returns "0" for destroyed/wiped structs)
get_hp() {
    local sid="$1"
    local hp
    hp=$(query query structs struct "${sid}" 2>/dev/null | jq -r '.structAttributes.health // empty' 2>/dev/null || echo "")
    if [ -z "${hp}" ]; then echo "0"; else echo "${hp}"; fi
}

info "Recording initial health (all should be 6)"
for P in 1 2 3 4 5; do
    eval "CS=\${CS_${P}}"
    echo "  CS_${P} (${CS}): HP=$(get_hp "${CS}")"
done

# ─── Test 1: Home fleet (F1) → front of list (F2) ───
info "Test 1: F1 (home) → F2 (front of list)"
wait_for_charge "${FP_1_ID}" "${CHARGE_ATTACK}"
CS_2_HP_BEFORE=$(get_hp "${CS_2}")
run_tx "F1 (home) attacks CS_2 on F2 (front of list)" \
    tx structs struct-attack "${CS_1}" "${CS_2}" primaryWeapon --from fplayer_1
CS_2_HP=$(get_hp "${CS_2}")
echo "  CS_2 HP: ${CS_2_HP} (was ${CS_2_HP_BEFORE})"
assert_eq "Home fleet hit front of list" "true" "$([ "${CS_2_HP}" -lt "${CS_2_HP_BEFORE}" ] && echo true || echo false)"

# ─── Test 2: F3 → forward neighbor F2 ───
info "Test 2: F3 → F2 (forward neighbor)"
wait_for_charge "${FP_3_ID}" "${CHARGE_ATTACK}"
CS_2_HP_BEFORE=$(get_hp "${CS_2}")
run_tx "F3 attacks CS_2 on F2 (forward neighbor)" \
    tx structs struct-attack "${CS_3}" "${CS_2}" primaryWeapon --from fplayer_3
CS_2_HP=$(get_hp "${CS_2}")
echo "  CS_2 HP: ${CS_2_HP} (was ${CS_2_HP_BEFORE})"
assert_eq "F3 hit forward neighbor F2" "true" "$([ "${CS_2_HP}" -lt "${CS_2_HP_BEFORE}" ] && echo true || echo false)"

# ─── Test 3: F5 → forward neighbor F4 ───
info "Test 3: F5 → F4 (forward neighbor)"
wait_for_charge "${FP_5_ID}" "${CHARGE_ATTACK}"
CS_4_HP_BEFORE=$(get_hp "${CS_4}")
run_tx "F5 attacks CS_4 on F4 (forward neighbor)" \
    tx structs struct-attack "${CS_5}" "${CS_4}" primaryWeapon --from fplayer_5
CS_4_HP=$(get_hp "${CS_4}")
echo "  CS_4 HP: ${CS_4_HP} (was ${CS_4_HP_BEFORE})"
assert_eq "F5 hit forward neighbor F4" "true" "$([ "${CS_4_HP}" -lt "${CS_4_HP_BEFORE}" ] && echo true || echo false)"

info "Health after Phase 8"
for P in 1 2 3 4 5; do eval "CS=\${CS_${P}}"; echo "  CS_${P}: HP=$(get_hp "${CS}")"; done


# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 9: Front-of-List Planetary Reach
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 9: Front-of-List Planetary Reach"

# F2 has locationListForward="" → it's at the front of the raid queue.
# Rule: if forward=="" and same planet → can reach ANY target on the planet.
# F2 should be able to hit non-adjacent F4 and F5.

info "Test R1: F2 (front, fwd='') → F5 (non-adjacent, same planet)"
wait_for_charge "${FP_2_ID}" "${CHARGE_ATTACK}"
CS_5_HP_BEFORE=$(get_hp "${CS_5}")
run_tx "F2 attacks CS_5 on F5 (front-of-list reaches whole planet)" \
    tx structs struct-attack "${CS_2}" "${CS_5}" primaryWeapon --from fplayer_2
CS_5_HP=$(get_hp "${CS_5}")
echo "  CS_5 HP: ${CS_5_HP} (was ${CS_5_HP_BEFORE})"
assert_eq "Front-of-list F2 hit non-adjacent F5" "true" "$([ "${CS_5_HP}" -lt "${CS_5_HP_BEFORE}" ] && echo true || echo false)"


# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 10: Destruction, Fleet Recall & Linked List Collapse
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 10: Destruction & Linked List Collapse"

# When a command ship is destroyed:
#   1. The struct is wiped from chain (query returns "not found")
#   2. The fleet is sent home (status=onStation, location=home planet)
#   3. The fleet is removed from the linked list (neighbors relink)
#
# Current list: F2 ↔ F3 ↔ F4 ↔ F5
# We'll destroy CS_2 by attacking it until HP=0, then verify:
#   - CS_2 wiped from chain
#   - F2 returned to its home planet (FP_2_PLANET_ID)
#   - F2 status = onStation
#   - List collapsed: Planet.start → F3, F3.forward = ""

info "Current health before destruction test"
for P in 1 2 3 4 5; do eval "CS=\${CS_${P}}"; echo "  CS_${P}: HP=$(get_hp "${CS}")"; done

# Attack CS_2 repeatedly until it dies
# F3 is adjacent (forward neighbor of F2), so F3 can attack CS_2
CS_2_HP=$(get_hp "${CS_2}")
ATTACK_COUNT=0
while [ "${CS_2_HP}" -gt 0 ] 2>/dev/null && [ "${ATTACK_COUNT}" -lt 5 ]; do
    ATTACK_COUNT=$((ATTACK_COUNT + 1))
    wait_for_charge "${FP_3_ID}" "${CHARGE_ATTACK}"
    run_tx "F3 attacks CS_2 (#${ATTACK_COUNT}, CS_2 HP=${CS_2_HP})" \
        tx structs struct-attack "${CS_3}" "${CS_2}" primaryWeapon --from fplayer_3
    CS_2_HP=$(get_hp "${CS_2}")
    echo "  CS_2 HP after attack #${ATTACK_COUNT}: ${CS_2_HP}"
done

info "CS_2 destruction result"
echo "  Attacks required: ${ATTACK_COUNT}"

# Verify CS_2 is wiped from chain
# Struct may persist briefly at HP=0 before being cleaned up in the next block's end-blocker
sleep 6
CS_2_QUERY=$(structsd ${PARAMS_QUERY} query structs struct "${CS_2}" 2>&1 || true)
CS_2_HP_CHECK=$(get_hp "${CS_2}")
if [ -z "${CS_2_QUERY}" ] || echo "${CS_2_QUERY}" | grep -qi "not found\|error\|object"; then
    echo -e "  ${GREEN}PASS${NC}: CS_2 (${CS_2}) wiped from chain"
    PASS_COUNT=$((PASS_COUNT + 1))
elif [ "${CS_2_HP_CHECK}" = "0" ]; then
    echo -e "  ${GREEN}PASS${NC}: CS_2 (${CS_2}) HP=0 (destroyed, pending cleanup)"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    echo -e "  ${RED}FAIL${NC}: CS_2 (${CS_2}) still exists on chain (HP=${CS_2_HP_CHECK})"
    FAIL_COUNT=$((FAIL_COUNT + 1))
fi

# Verify F2 returned home
info "Checking F2 returned home after CS_2 destruction"
F2_JSON=$(query_fleet "${F2}")
F2_LOC=$(jqr "${F2_JSON}" '.Fleet.locationId')
F2_STATUS=$(jqr "${F2_JSON}" '.Fleet.status')
if [ -z "${F2_STATUS}" ]; then F2_STATUS="onStation"; fi
assert_eq "F2 returned to home planet" "${FP_2_PLANET_ID}" "${F2_LOC}"
assert_eq "F2 status after recall" "onStation" "${F2_STATUS}"
echo "  F2: loc=${F2_LOC} status=${F2_STATUS}"

# Verify linked list collapsed: F3 is now front of list
info "Verifying linked list collapsed (F2 removed)"
echo "  Expected: Planet.start → F3 ↔ F4 ↔ F5 ← Planet.last"

PLANET_START=$(get_planet_field "${TARGET_PLANET}" "locationListStart")
PLANET_LAST=$(get_planet_field "${TARGET_PLANET}" "locationListLast")
assert_eq "Planet.start after F2 removal" "${F3}" "${PLANET_START}"
assert_eq "Planet.last unchanged" "${F5}" "${PLANET_LAST}"

F3_JSON=$(query_fleet "${F3}")
F3_FWD=$(jqr "${F3_JSON}" '.Fleet.locationListForward')
F3_BWD=$(jqr "${F3_JSON}" '.Fleet.locationListBackward')
assert_eq "F3 is now front (forward='')" "" "${F3_FWD}"
assert_eq "F3 backward" "${F4}" "${F3_BWD}"
echo "  F3: fwd='${F3_FWD}' bwd='${F3_BWD}'"

F4_JSON=$(query_fleet "${F4}")
F4_FWD=$(jqr "${F4_JSON}" '.Fleet.locationListForward')
F4_BWD=$(jqr "${F4_JSON}" '.Fleet.locationListBackward')
assert_eq "F4 forward" "${F3}" "${F4_FWD}"
assert_eq "F4 backward" "${F5}" "${F4_BWD}"
echo "  F4: fwd='${F4_FWD}' bwd='${F4_BWD}'"

F5_JSON=$(query_fleet "${F5}")
F5_FWD=$(jqr "${F5_JSON}" '.Fleet.locationListForward')
F5_BWD=$(jqr "${F5_JSON}" '.Fleet.locationListBackward')
assert_eq "F5 forward" "${F4}" "${F5_FWD}"
assert_eq "F5 backward (still last)" "" "${F5_BWD}"
echo "  F5: fwd='${F5_FWD}' bwd='${F5_BWD}'"

info "Linked list after collapse:"
echo "  Planet(${TARGET_PLANET}).start=${PLANET_START} .last=${PLANET_LAST}"
echo "  F3(fwd='', bwd=${F3_BWD}) ↔ F4(fwd=${F4_FWD}, bwd=${F4_BWD}) ↔ F5(fwd=${F5_FWD}, bwd='')"
echo "  F2 → home (${F2_LOC}), F1 → home (on station)"


# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 11: Non-Adjacent Attacks (Should Fail)
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 11: Non-Adjacent Attacks (Should Fail)"

# Current list: F3 ↔ F4 ↔ F5 (F3 is front)
# Middle/back fleets can only hit adjacent. F1 (home) can only hit front (F3).
#
# Should fail:
#   F4 → F3's neighbor check: F4 sees F3 and F5 — but NOT planet targets
#   F5 → F3 (F5 sees F4 only, not F3)
#   F1 (home) → F4 (home can only hit front = F3)
#   F1 (home) → F5 (same)

info "Health snapshot before negative tests"
CS_3_HP=$(get_hp "${CS_3}")
CS_4_HP=$(get_hp "${CS_4}")
CS_5_HP=$(get_hp "${CS_5}")
echo "  CS_3: HP=${CS_3_HP}, CS_4: HP=${CS_4_HP}, CS_5: HP=${CS_5_HP}"

# ─── Test N1: F5 → F3 (not adjacent, gap of 1) ───
if [ "${CS_5_HP}" = "0" ]; then
    info "SKIP N1: CS_5 destroyed"
else
    info "Test N1: F5 → F3 (not adjacent — F5 only sees F4)"
    wait_for_charge "${FP_5_ID}" "${CHARGE_ATTACK}"
    run_tx_expect_fail "F5 attacks CS_3 on F3 (not adjacent)" \
        tx structs struct-attack "${CS_5}" "${CS_3}" primaryWeapon --from fplayer_5
    sleep "${SLEEP}"
    CS_3_CHECK=$(get_hp "${CS_3}")
    assert_eq "CS_3 HP unchanged after F5→F3" "${CS_3_HP}" "${CS_3_CHECK}"
fi

# ─── Test N2: F1 (home) → F4 (not front of list) ───
CS_1_HP=$(get_hp "${CS_1}")
if [ "${CS_1_HP}" = "0" ]; then
    info "SKIP N2: CS_1 destroyed"
else
    info "Test N2: F1 (home) → F4 (home can only hit front = F3)"
    wait_for_charge "${FP_1_ID}" "${CHARGE_ATTACK}"
    run_tx_expect_fail "F1 (home) attacks CS_4 on F4 (not front of list)" \
        tx structs struct-attack "${CS_1}" "${CS_4}" primaryWeapon --from fplayer_1
    sleep "${SLEEP}"
    CS_4_CHECK=$(get_hp "${CS_4}")
    assert_eq "CS_4 HP unchanged after F1→F4" "${CS_4_HP}" "${CS_4_CHECK}"
fi

# ─── Test N3: F1 (home) → F5 (not front of list) ───
if [ "${CS_1_HP}" = "0" ]; then
    info "SKIP N3: CS_1 destroyed"
else
    info "Test N3: F1 (home) → F5 (home can only hit front = F3)"
    wait_for_charge "${FP_1_ID}" "${CHARGE_ATTACK}"
    run_tx_expect_fail "F1 (home) attacks CS_5 on F5 (not front of list)" \
        tx structs struct-attack "${CS_1}" "${CS_5}" primaryWeapon --from fplayer_1
    sleep "${SLEEP}"
    CS_5_CHECK=$(get_hp "${CS_5}")
    assert_eq "CS_5 HP unchanged after F1→F5" "${CS_5_HP}" "${CS_5_CHECK}"
fi


# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 12: Final State Dump
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 12: Final Status"

info "Command Ship Health"
for P in 1 2 3 4 5; do
    eval "CS=\${CS_${P}}"
    HP=$(get_hp "${CS}")
    if [ "${HP}" = "0" ]; then
        echo "  CS_${P} (${CS}): DESTROYED (wiped from chain)"
    else
        echo "  CS_${P} (${CS}): HP=${HP}"
    fi
done

echo ""
info "Fleet Positions"
for P in 1 2 3 4 5; do
    eval "FLEET_ID=\${FP_${P}_FLEET_ID}"
    eval "HOME_PLANET=\${FP_${P}_PLANET_ID}"
    FLEET_JSON=$(query_fleet "${FLEET_ID}" 2>/dev/null || echo '{}')
    LOC=$(jqr "${FLEET_JSON}" '.Fleet.locationId')
    STATUS=$(jqr "${FLEET_JSON}" '.Fleet.status')
    if [ -z "${STATUS}" ]; then STATUS="onStation"; fi
    FWD=$(jqr "${FLEET_JSON}" '.Fleet.locationListForward')
    BWD=$(jqr "${FLEET_JSON}" '.Fleet.locationListBackward')
    AT_HOME=""
    if [ "${LOC}" = "${HOME_PLANET}" ]; then AT_HOME=" (HOME)"; fi
    echo "  F${P} (${FLEET_ID}): loc=${LOC}${AT_HOME} status=${STATUS} fwd='${FWD}' bwd='${BWD}'"
done

echo ""
info "Planet ${TARGET_PLANET} Fleet List"
echo "  locationListStart: $(get_planet_field "${TARGET_PLANET}" "locationListStart")"
echo "  locationListLast:  $(get_planet_field "${TARGET_PLANET}" "locationListLast")"


# ═════════════════════════════════════════════════════════════════════════════
#  Summary
# ═════════════════════════════════════════════════════════════════════════════

print_summary
