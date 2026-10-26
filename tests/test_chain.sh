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
#   python3 (exact SDK delegation-share arithmetic)
#   - structsd chain running locally (fresh chain required unless resuming)
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
#                      event attributes emitted by the chain. Rejected and
#                      unverified attempts have separate record types; only
#                      confirmed attack events contribute to the event count.
#   --resume-from N    Skip phases before N and resume execution from phase N.
#                      Recovers all IDs by querying the running chain.
#                      Phase names: 0 1 2 3 3b 4 4b 4c 4d 4e 4f 4g 5 5b 6
#                        7 7b 7c 8 9 10 11 12 13 13b 14 15 15b 16
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

# Faster local blocks need shorter delivery sleeps and enough blocks for live
# agreement edits. These affect test timing/quotes, never chain PoW parameters.
configure_fixture_timing() {
    SLEEP="${STRUCTS_TEST_SETTLE_SECONDS:-2}"
    BIGGER_SLEEP="${STRUCTS_TEST_LONG_SETTLE_SECONDS:-15}"
    FIXTURE_DURATION_MULTIPLIER="${STRUCTS_TEST_DURATION_MULTIPLIER:-1}"
    python3 - "${SLEEP}" "${BIGGER_SLEEP}" "${FIXTURE_DURATION_MULTIPLIER}" <<'PY'
from decimal import Decimal
import re
import sys
for value in sys.argv[1:3]:
    if not re.fullmatch(r"(?:0|[1-9][0-9]?)(?:[.][0-9]{1,3})?", value) or not Decimal("0.01") <= Decimal(value) <= Decimal("60"):
        sys.exit("Fixture settle seconds must be a decimal between 0.01 and 60")
if not re.fullmatch(r"[1-9][0-9]{0,2}", sys.argv[3]) or int(sys.argv[3]) > 100:
    sys.exit("Fixture duration multiplier must be an integer between 1 and 100")
PY
}
configure_fixture_timing

# How long to follow a txhash into a block before giving up. Blocks are ~1s on a
# dev chain, so this is generous.
TX_CONFIRM_TIMEOUT=15

# --gas-adjustment matters for exactly one message today and is easy to leave off
# by accident, so: GasRouterDecorator replaces the gas meter with a 20M cap for
# any transaction made purely of free structs or staking messages, which discards
# whatever --gas auto estimated. MsgGuildCreate is in PricedStructsMessages and so
# keeps its estimate, and the CLI default adjustment of 1.0 is the estimate exactly
# — no headroom at all. Delivery routinely costs a percent more than simulation,
# which is how guild-create came to be accepted into the mempool and then die in
# the block with `out of gas`. Costs nothing: minimum-gas-prices is 0ualpha.
#
# --output json is here because the flag default is overridden by client.toml,
# which asks for text: every tx response this script has ever taken came out as
# YAML, so the jq that was meant to read `.code` found nothing and fell through to
# an unconditional "TX submitted". That is the other half of why an out-of-gas
# guild creation reported success.
PARAMS_TX="--home ~/.structs --keyring-dir ~/.structs --keyring-backend test --gas auto --gas-adjustment 1.5 --output json --yes=true"
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

# _tx_json: pick the tx response out of command output.
#
# A plain tx command prints only its response, but a compute command prints mining
# progress first and jq cannot be handed the whole blob. Falls back to the blob
# unchanged so that a command which failed before broadcasting anything still
# reaches the error branches below, which read it as text.
_tx_json() {
    local blob="$1"
    if echo "${blob}" | jq -e . >/dev/null 2>&1; then
        echo "${blob}"
        return 0
    fi

    # Take everything from the last line that opens a JSON object, so this works
    # whether the response came out compact or indented.
    local from tail_json
    from=$(echo "${blob}" | grep -n '^[[:space:]]*{' | tail -1 | cut -d: -f1)
    if [ -n "${from}" ]; then
        tail_json=$(echo "${blob}" | tail -n "+${from}")
        if echo "${tail_json}" | jq -e . >/dev/null 2>&1; then
            echo "${tail_json}"
            return 0
        fi
    fi

    echo "${blob}"
}

# _tx_delivered: follow a broadcast txhash into a block and report what the block
# actually did with it.
#
# A broadcast response carries the CheckTx code only, so `code: 0` there means
# "accepted into the mempool" and nothing more. Every message failure that --gas
# auto's simulation does not catch lands after that point: out of gas is the one
# that bit us, where guild-create reported success in green and then died at
# height 88, leaving a downstream assertion to notice hours later.
#
# Echoes "<code>|<first line of raw_log>", or nothing when there is no hash to
# follow — which is the honest answer for a simulation or cobra failure that never
# reached a block.
_tx_delivered() {
    local txhash result found height code elapsed=0
    txhash=$(echo "$1" | jq -r '.txhash // empty' 2>/dev/null || echo "")
    if [ -z "${txhash}" ]; then
        return 0
    fi

    while [ "${elapsed}" -lt "${TX_CONFIRM_TIMEOUT}" ]; do
        result=$(query query tx "${txhash}" 2>/dev/null || echo "")
        found=$(echo "${result}" | jq -r '.txhash // empty' 2>/dev/null || echo "")
        height=$(echo "${result}" | jq -r '.height // "0"' 2>/dev/null || echo "0")
        code=$(echo "${result}" | jq -r '.code // 0' 2>/dev/null || echo "")
        if [ "${found}" = "${txhash}" ] && [[ "${height}" =~ ^[0-9]+$ ]] && \
           [ "${height}" -gt 0 ] && [[ "${code}" =~ ^[0-9]+$ ]]; then
            # Presence is decided by txhash and the code by `.code // 0`, because a
            # zero-valued proto field may be omitted from the JSON entirely. A
            # missing code has to read as success, not as "not in a block yet".
            echo "${code}|$(echo "${result}" | jq -r '.raw_log // ""' 2>/dev/null | head -1)"
            return 0
        fi
        sleep 1
        elapsed=$((elapsed + 1))
    done

    echo "timeout|tx ${txhash} never landed in a block within ${TX_CONFIRM_TIMEOUT}s"
}

# _tx_error_line: the reason a command that never broadcast gives for stopping.
#
# structsd prints cobra's usage block and then the error, so the error is the last
# non-empty line and a wider window only picks up the flag list. That window has to
# be exactly one line, because --log_level's help text enumerates the logging levels
# and one of them is the word "panic" — read as a chain panic by anything scanning
# the reason for one, which is how a correct "is the owner of guild" refusal reported
# itself as a panic.
_tx_error_line() {
    printf '%s\n' "$1" | grep -v '^[[:space:]]*$' | tail -1 | tail -c 1000
}

# _tx_outcome: classify a transaction that was expected to fail.
#
# Echoes "ok|" on confirmed success, "rejected|<reason>" on a rejection, or
# "unknown|<reason>" when delivery cannot be established. A
# rejection can arrive from any of three places and all three count: the CLI before
# broadcast (a failed --gas auto simulation, or cobra refusing the invocation), the
# ante at CheckTx, or the handler in the block. Only the first two were ever looked
# at, so a message that passed the ante and was refused on delivery read as an
# unexpected success.
_tx_outcome() {
    local output tx_code delivered
    output=$(_tx_json "$1")
    tx_code=$(echo "${output}" | jq -r '.code // empty' 2>/dev/null || echo "")

    if [ -n "${tx_code}" ] && ! [[ "${tx_code}" =~ ^[0-9]+$ ]]; then
        echo "unknown|response has an invalid transaction code"
        return 0
    fi

    if [ -n "${tx_code}" ] && [ "${tx_code}" != "0" ]; then
        echo "rejected|$(echo "${output}" | jq -r '.raw_log // empty' 2>/dev/null | head -1)"
        return 0
    fi

    # A JSON response without a code is not itself a rejection. Some encoders
    # omit zero; follow its hash when present. Empty/malformed responses and
    # transport failures cannot prove a handler refused the message.
    if [ -z "${tx_code}" ]; then
        if [ -z "${output}" ] || echo "${output}" | jq -e 'type == "object"' >/dev/null 2>&1; then
            if [ -z "$(echo "${output}" | jq -r '.txhash // empty' 2>/dev/null || echo "")" ]; then
                echo "unknown|response has neither a rejection code nor a transaction hash"
                return 0
            fi
        elif echo "$1" | grep -qiE 'connection refused|dial tcp|context deadline exceeded|TLS handshake timeout|unexpected EOF'; then
            echo "unknown|$(_tx_error_line "$1")"
            return 0
        else
            echo "rejected|$(_tx_error_line "$1")"
            return 0
        fi
    fi

    delivered=$(_tx_delivered "${output}")
    if [ -z "${delivered}" ]; then
        echo "unknown|broadcast response has no transaction hash"
    elif [ "${delivered%%|*}" = "timeout" ]; then
        echo "unknown|${delivered#*|}"
    elif [ "${delivered%%|*}" = "0" ]; then
        echo "ok|"
    else
        echo "rejected|${delivered#*|}"
    fi
}

# Unknown delivery is never evidence of success or an intentional refusal.
# Call outside command substitution so the failed-check count survives.
_tx_outcome_known() {
    if [ "${1%%|*}" = "unknown" ]; then
        echo -e "  ${RED}FAIL${NC}: transaction outcome unverified: ${1#*|}"
        FAIL_COUNT=$((FAIL_COUNT + 1))
        return 1
    fi
    return 0
}

# _check_tx_output: shared logic for checking TX command output.
#
# Counts a failure, which it did not used to: every caller of this expects its
# transaction to succeed, and a red line nobody tallied is a red line that scrolls
# past. Because it now writes FAIL_COUNT, never call it inside a command
# substitution — a subshell would swallow the count. LAST_TX_DELIVERED is exposed
# so a caller can inspect the delivered reason without re-running anything.
_check_tx_output() {
    local output
    output=$(_tx_json "$1")
    LAST_TX_DELIVERED=""

    local tx_code
    tx_code=$(echo "${output}" | jq -r '.code // empty' 2>/dev/null || echo "")

    if [ -n "${tx_code}" ] && [ "${tx_code}" != "0" ]; then
        echo -e "  ${RED}TX rejected before delivery (code=${tx_code})${NC}"
        echo "  $(echo "${output}" | head -5)"
        FAIL_COUNT=$((FAIL_COUNT + 1))
        return 0
    fi

    if [ -z "${tx_code}" ]; then
        if echo "${output}" | grep -qi "error\|panic\|failed\|invalid"; then
            echo -e "  ${RED}TX failed before broadcast (simulation or client error)${NC}"
            echo "  $(_tx_error_line "${output}")"
            FAIL_COUNT=$((FAIL_COUNT + 1))
            return 0
        fi
        if echo "${output}" | grep -qiE "accepts [0-9]+ arg|unknown (command|flag|shorthand)|^Usage:"; then
            # cobra rejected the invocation, so nothing was ever broadcast. Its wording
            # carries none of the words above, which is how a malformed address-register
            # call read as "TX submitted" for as long as it did.
            echo -e "  ${RED}TX never broadcast (malformed command)${NC}"
            echo "  $(echo "${output}" | head -2)"
            FAIL_COUNT=$((FAIL_COUNT + 1))
            return 0
        fi
    fi

    # Accepted into the mempool. What matters now is what the block did with it.
    local delivered
    delivered=$(_tx_delivered "${output}")
    LAST_TX_DELIVERED="${delivered}"

    if [ -z "${delivered}" ]; then
        echo -e "  ${RED}FAIL${NC}: no confirmed transaction delivery or transaction hash"
        FAIL_COUNT=$((FAIL_COUNT + 1))
        return 0
    fi

    local delivered_code="${delivered%%|*}"
    if [ "${delivered_code}" = "0" ]; then
        echo -e "  ${GREEN}TX delivered${NC}"
    else
        echo -e "  ${RED}TX failed in block (code=${delivered_code})${NC}"
        echo "  ${delivered#*|}"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
}

# run_tx: execute a transaction, show the command, and check for success.
#
# The settle sleep comes before the check rather than after it, so that the wait
# the script already took for state to land is the same wait _check_tx_output
# needs to find the transaction in a block. Checking first would make every one of
# the 360 call sites poll for a block it was about to wait for anyway.
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
    # Exposed so callers can inspect a rejection reason without re-running the tx.
    LAST_TX_OUTPUT="${OUTPUT}"
    sleep "${SLEEP}"
    _check_tx_output "${OUTPUT}"
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
    LAST_TX_OUTPUT="${OUTPUT}"
    sleep "${BIGGER_SLEEP}"
    _check_tx_output "${OUTPUT}"
}

# run_tx_noauto: execute a TX with fixed gas (bypasses --gas auto simulation)
# Used for operations where --gas auto simulation fails due to stale state
# (e.g., invite-approve/deny where the application isn't visible in simulation)
PARAMS_TX_NOAUTO="--home ~/.structs --keyring-dir ~/.structs --keyring-backend test --gas 500000 --output json --yes=true"
run_tx_noauto() {
    local description="$1"
    shift
    info "${description}"
    echo -e "  ${BOLD}structsd ${PARAMS_TX_NOAUTO} $*${NC}"
    local OUTPUT
    OUTPUT=$(structsd ${PARAMS_TX_NOAUTO} "$@" 2>&1) || true
    LAST_TX_OUTPUT="${OUTPUT}"
    sleep "${SLEEP}"
    _check_tx_output "${OUTPUT}"
}

# run_tx_tolerate_reject: run_tx for the handful of places where a rejection is a
# documented reason to skip a phase rather than a defect — a build the miner has
# left the player too poor to afford, say. Reports the outcome and counts nothing,
# because the caller decides what the rejection means.
#
# Everything else goes through run_tx, which now counts a delivery failure. Keep
# this list short: it is the one way to make a real failure invisible again.
run_tx_tolerate_reject() {
    local description="$1"
    shift
    info "${description} (a rejection here is tolerated)"
    echo -e "  ${BOLD}structsd ${PARAMS_TX} $*${NC}"
    local OUTPUT OUTCOME
    OUTPUT=$(structsd ${PARAMS_TX} "$@" 2>&1) || true
    LAST_TX_OUTPUT="${OUTPUT}"
    sleep "${SLEEP}"
    OUTCOME=$(_tx_outcome "${OUTPUT}")
    _tx_outcome_known "${OUTCOME}" || return 0
    if [ "${OUTCOME%%|*}" = "ok" ]; then
        echo -e "  ${GREEN}TX delivered${NC}"
    else
        echo -e "  ${YELLOW}TX rejected (tolerated)${NC}"
        echo "  ${OUTCOME#*|}"
    fi
}

# run_compute: execute a compute command (proof-of-work)
#
# Used to print "Compute completed" in green unconditionally, having looked at
# nothing at all, so a compute that mined a valid proof and then lost the
# transaction reported success. Proof computation can also fail before broadcast.
run_compute() {
    local description="$1"
    shift
    info "${description} (compute)"
    echo -e "  ${BOLD}structsd ${PARAMS_TX} $*${NC}"
    local OUTPUT
    OUTPUT=$(structsd ${PARAMS_TX} "$@" 2>&1) || true
    LAST_TX_OUTPUT="${OUTPUT}"
    # The last few lines carry the solution and the tx response; the rest is
    # progress chatter.
    echo "${OUTPUT}" | tail -5 | sed 's/^/  /'
    sleep "${BIGGER_SLEEP}"
    _check_tx_output "${OUTPUT}"
}

# run_tx_expect_fail_noauto: same but with fixed gas (no --gas auto)
PARAMS_TX_NOFEE="--home ~/.structs --keyring-dir ~/.structs --keyring-backend test --gas 500000 --fees 0ualpha --output json --yes=true"
run_tx_expect_fail_noauto() {
    local description="$1"
    shift
    info "${description} (expect failure, fixed gas)"
    echo -e "  ${BOLD}structsd ${PARAMS_TX_NOFEE} $*${NC}"
    local OUTPUT OUTCOME
    OUTPUT=$(structsd ${PARAMS_TX_NOFEE} "$@" 2>&1) || true
    LAST_TX_OUTPUT="${OUTPUT}"
    sleep "${SLEEP}"
    OUTCOME=$(_tx_outcome "${OUTPUT}")
    _tx_outcome_known "${OUTCOME}" || return 0
    if [ "${OUTCOME%%|*}" = "ok" ]; then
        echo -e "  ${RED}FAIL${NC}: TX succeeded but was expected to fail"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    else
        echo -e "  ${GREEN}PASS${NC}: TX correctly rejected"
        echo "  ${OUTCOME#*|}"
        PASS_COUNT=$((PASS_COUNT + 1))
    fi
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

# _log_battle_event: only a confirmed typed event increments the capture count.
# Failed or unreadable attempts stay in the JSONL as distinct record types.
_log_battle_event() {
    local tx_output="$1" description="$2"
    local txhash='' tx_result='' record='' reason='' timestamp helper_dir
    timestamp=$(date -u +%Y-%m-%dT%H:%M:%SZ)
    helper_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd) || return 1
    if ! txhash=$(_tx_json "${tx_output}" | jq -er '.txhash | select(type == "string" and test("^[0-9A-Fa-f]{64}$"))'); then
        reason='Attack output has no valid transaction hash'
    elif ! tx_result=$(query query tx "${txhash}"); then
        reason='Delivered attack query failed'
    elif ! record=$(printf '%s' "${tx_result}" | python3 "${helper_dir}/battle_log.py" "${txhash}" "${description}" "${timestamp}"); then
        reason='Delivered attack receipt or typed event is invalid'
    fi
    if [ -n "${reason}" ]; then
        jq -nc --arg ts "${timestamp}" --arg desc "${description}" --arg tx "${txhash}" \
            --arg reason "${reason}" --arg output "${tx_output}" --arg receipt "${tx_result}" \
            '{type:"attack_unverified", timestamp:$ts, description:$desc, txhash:$tx, reason:$reason, tx_output:$output, query_output:$receipt}' \
            >> "${BATTLE_LOG_FILE}" || return 1
        echo -e "  ${RED}FAIL${NC}: ${reason}; unverified attempt preserved in battle log"
        FAIL_COUNT=$((FAIL_COUNT + 1))
        return 0
    fi
    printf '%s\n' "${record}" >> "${BATTLE_LOG_FILE}" || return 1
    if [ "$(printf '%s' "${record}" | jq -r '.type')" = attack ]; then
        BATTLE_LOG_COUNT=$((BATTLE_LOG_COUNT + 1))
        echo -e "  ${CYAN}(battle event #${BATTLE_LOG_COUNT} logged)${NC}"
    else
        echo -e "  ${CYAN}(rejected attack preserved; no attack event counted)${NC}"
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

# A transport failure or empty successful response does not prove deletion.
# Read stderr directly because query() deliberately suppresses it.
assert_object_missing() {
    local label="$1" kind="$2" id="$3" output missing=false
    if output=$(structsd ${PARAMS_QUERY} query structs "${kind}" "${id}" 2>&1); then
        missing=false
    elif [[ "${output}" == *"object not found"* ]]; then
        missing=true
    fi
    assert_eq "${label}" "true" "${missing}"
    if [ "${missing}" != true ]; then
        printf '  Unexpected query response: %s\n' "${output}"
    fi
}

# Parse addressproof output as data. Clear prior values before parsing so an
# incomplete proof cannot borrow signing material from another phase.
parse_address_proof() {
    ADDRESS='' PROOF_PUBKEY='' PROOF_SIGNATURE='' PRIVKEY=''
    local line value key seen='|'
    while IFS= read -r line; do
        value="${line#*=}"
        key="${line%%=*}"
        case "${key}" in ADDRESS|PROOF_PUBKEY|PROOF_SIGNATURE|PRIVKEY) ;; *) continue ;; esac
        case "${seen}" in *"|${key}|"*) return 1 ;; esac
        seen="${seen}${key}|"
        case "${line}" in
            ADDRESS=*) [ -z "${ADDRESS}" ] || return 1; ADDRESS="${value}" ;;
            PROOF_PUBKEY=*) [ -z "${PROOF_PUBKEY}" ] || return 1; PROOF_PUBKEY="${value}" ;;
            PROOF_SIGNATURE=*) [ -z "${PROOF_SIGNATURE}" ] || return 1; PROOF_SIGNATURE="${value}" ;;
            PRIVKEY=*) [ -z "${PRIVKEY}" ] || return 1; PRIVKEY="${value}" ;;
        esac
    done <<< "$1"
    [[ "${ADDRESS}" =~ ^structs1[0-9a-z]+$ ]] &&
    [[ "${PROOF_PUBKEY}" =~ ^[0-9a-f]{66}$ ]] &&
    [[ "${PROOF_SIGNATURE}" =~ ^[0-9a-f]{128}$ ]] &&
    [[ "${PRIVKEY}" =~ ^[0-9a-f]{64}$ ]]
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

# assert_ge: check that actual >= threshold (numeric)
# Used where truncation dust makes an exact figure the wrong assertion, most
# notably provider collateral solvency.
assert_ge() {
    local label="$1"
    local threshold="$2"
    local actual="$3"
    if python3 - "${threshold}" "${actual}" <<'PY_COMPARE'
import re
import sys
threshold, actual = sys.argv[1:]
if not all(re.fullmatch(r"-?(0|[1-9][0-9]*)", value) for value in (threshold, actual)):
    raise SystemExit(1)
raise SystemExit(0 if int(actual) >= int(threshold) else 1)
PY_COMPARE
    then
        echo -e "  ${GREEN}PASS${NC}: ${label} = ${actual} >= ${threshold}"
        PASS_COUNT=$((PASS_COUNT + 1))
    else
        echo -e "  ${RED}FAIL${NC}: ${label} = '${actual}' not >= ${threshold}"
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

# A successful build identifies its struct in its own confirmed event. A list
# page cannot establish which object this transaction created, regardless of size.
get_created_struct_id() {
    local owner="$1" type_id="$2" txhash delivered struct_id
    txhash=$(_tx_json "${LAST_TX_OUTPUT}" | jq -er '.txhash
        | select(type == "string" and test("^[0-9A-Fa-f]{64}$"))') || return 1
    delivered=$(query query tx "${txhash}") || return 1
    struct_id=$(echo "${delivered}" | jq -er --arg hash "${txhash}" \
        --arg owner "${owner}" --arg kind "${type_id}" '
        def positive_uint64:
            type == "string" and test("^[1-9][0-9]*$") and
            (length < 20 or (length == 20 and . <= "18446744073709551615"));
        select((if has("code") then (.code == 0 or .code == "0") else true end)
            and (.txhash | ascii_upcase) == ($hash | ascii_upcase))
        | select(.height | positive_uint64 and
            (length < 19 or (length == 19 and . <= "9223372036854775807")))
        | [.events[] | select(.type == "structs.structs.EventStruct")
            | [.attributes[] | select(.key == "structure")]
            | if length == 1 then .[0].value | fromjson else error("ambiguous struct event") end
            | select(.owner == $owner and .type == $kind)]
        | select(length == 1) | .[0]
        | select(.index | positive_uint64)
        | select(.id == ("5-" + .index)) | .id') || return 1
    printf '%s\n' "${struct_id}"
}

# Automatic command ships are named by the player's fleet, not by creation order.
get_command_ship_id() {
    local owner="$1" fleet="$2" fleet_json struct_id struct_json
    fleet_json=$(query query structs fleet "${fleet}") || return 1
    struct_id=$(echo "${fleet_json}" | jq -er --arg owner "${owner}" --arg fleet "${fleet}" '
        .Fleet | select(.id == $fleet and .owner == $owner) | .commandStruct
        | select(type == "string" and test("^5-[1-9][0-9]*$"))') || return 1
    struct_json=$(query query structs struct "${struct_id}") || return 1
    echo "${struct_json}" | jq -er --arg id "${struct_id}" --arg owner "${owner}" --arg fleet "${fleet}" '
        .Struct | select(.id == $id and .owner == $owner and .type == "1" and .locationId == $fleet) | .id'
}

# Verify the event-resolved object before computing against it.
# Returns non-zero, which under `set -e` aborts the run: every later assertion in
# the phase would be measuring the wrong struct, and the stale id may even belong
# to a struct that a subsequent sweep deletes, so stopping here is far cheaper
# than letting hours of proof-of-work run against corrupt bookkeeping.
# Usage: assert_new_struct <label> <new_id> <expected_owner> <expected_type>
assert_new_struct() {
    local label="$1" new_id="$2" owner="$3" exp_type="$4" record
    if ! record=$(query query structs struct "${new_id}" | jq -ce \
        --arg id "${new_id}" --arg owner "${owner}" --arg kind "${exp_type}" '
        .Struct | select(.id == $id and .owner == $owner and .type == $kind)'); then
        echo -e "  ${RED}FAIL${NC}: ${label} - created struct '${new_id}' does not match owner '${owner}' and type '${exp_type}'"
        FAIL_COUNT=$((FAIL_COUNT + 1)); return 1
    fi
    echo -e "  ${GREEN}PASS${NC}: ${label} = ${new_id} (type ${exp_type})"
    PASS_COUNT=$((PASS_COUNT + 1))
}

# first_free_slot: lowest unoccupied slot index for an ambit on a planet or fleet.
# Occupancy lives in the location record's per-ambit array (empty string = free),
# which is the same state the chain checks on build-initiate, so this stays
# correct as earlier phases consume slots. Echoes nothing when the ambit is full.
# Usage: first_free_slot <planet|fleet> <location id> <space|air|land|water>
first_free_slot() {
    local kind="$1" id="$2" ambit="$3"
    local root payload
    case "${kind}" in
        planet) root="Planet" ;;
        fleet)  root="Fleet" ;;
        *)      echo "Invalid slot location kind: ${kind}" >&2; return 1 ;;
    esac
    case "${ambit}" in
        space|air|land|water) ;;
        *) echo "Invalid slot ambit: ${ambit}" >&2; return 1 ;;
    esac
    # A failed RPC may still print JSON. Capture its status before parsing, and
    # distinguish a full location from missing or inconsistent occupancy data.
    payload=$(query query structs "${kind}" "${id}") || return 1
    printf '%s' "${payload}" | jq -r --arg root "${root}" --arg id "${id}" --arg ambit "${ambit}" '
        .[$root] as $loc
        | if ($loc | type) != "object" or $loc.id != $id then error("slot location identity differs") else . end
        | (if $loc | has($ambit + "Slots") then $loc[$ambit + "Slots"] else "0" end) as $count
        | (if $loc | has($ambit) then $loc[$ambit] else [] end) as $used
        | if ($used | type) != "array" then error("missing slot occupancy") else . end
        | if ($used | length | tostring) != $count then error("slot count and occupancy differ") else . end
        | if all($used[]; type == "string" and (. == "" or test("^5-[1-9][0-9]*$")))
          then ($used | index("") | if . == null then "" else tostring end)
          else error("invalid slot occupant") end
    '
}

# wait_for_free_slot: like first_free_slot, but tolerates the rubble window.
# A destroyed struct is not swept immediately: AppendStructDestructionQueue
# schedules it for blockHeight + StructSweepDelay (5 blocks, "Rubble Length"),
# and only then does StructSweepDestroyed clear planet.Land[slot] in the
# BeginBlocker. So for ~5 blocks after a kill the slot still reads occupied and
# build-initiate correctly rejects with "already has a struct on that slot".
# Poll instead of racing it. Echoes the slot index, or nothing on timeout.
# Usage: wait_for_free_slot <planet|fleet> <location id> <space|air|land|water> [timeout_seconds]
wait_for_free_slot() {
    local kind="$1" id="$2" ambit="$3" timeout="${4:-30}"
    local slot elapsed=0
    while [ "${elapsed}" -lt "${timeout}" ]; do
        slot=$(first_free_slot "${kind}" "${id}" "${ambit}") || return 1
        if [ -n "${slot}" ]; then
            # Progress goes to stderr so only the slot index lands on stdout.
            [ "${elapsed}" -gt 0 ] && echo -e "  ${GREEN}Slot free${NC}: ${kind} ${id} ${ambit} slot=${slot} (after ${elapsed}s)" >&2
            echo "${slot}"
            return 0
        fi
        [ "${elapsed}" -eq 0 ] && echo -e "  ${YELLOW}Waiting for free ${ambit} slot${NC}: ${kind} ${id} (rubble sweep takes ~5 blocks)" >&2
        sleep 3
        elapsed=$((elapsed + 3))
    done
    echo ""
}

# Resolve a newly created provider from its own confirmed transaction event.
get_created_provider_id() {
    local owner="$1" substation="$2" txhash delivered provider_id
    txhash=$(_tx_json "${LAST_TX_OUTPUT}" | jq -er '.txhash | select(type == "string" and test("^[0-9A-Fa-f]{64}$"))') || return 1
    delivered=$(query query tx "${txhash}") || return 1
    provider_id=$(echo "${delivered}" | jq -er \
        --arg hash "${txhash}" --arg owner "${owner}" --arg substation "${substation}" '
        select((.code == 0 or .code == "0") and (.txhash | ascii_upcase) == ($hash | ascii_upcase))
        | [.events[] | select(.type == "structs.structs.EventProvider")] | select(length == 1)
        | [.[0].attributes[] | select(.key == "provider")] | select(length == 1)
        | .[0].value | fromjson
        | select(.owner == $owner and .substationId == $substation)
        | .id | select(type == "string" and test("^10-[0-9]+$"))') || return 1
    printf '%s\n' "${provider_id}"
}

# Bind grid creation to its delivered transaction, including the feeder association.
# Usage: get_created_grid_id allocation <source> <controller>
#        get_created_grid_id substation <owner> <allocation>
get_created_grid_id() {
    local kind="$1" first="$2" second="$3" txhash delivered object_id
    case "${kind}" in allocation|substation) ;; *) return 1 ;; esac
    txhash=$(_tx_json "${LAST_TX_OUTPUT}" | jq -er '.txhash | select(type == "string" and test("^[0-9A-Fa-f]{64}$"))') || return 1
    delivered=$(query query tx "${txhash}") || return 1
    object_id=$(printf '%s' "${delivered}" | jq -er \
        --arg hash "${txhash}" --arg kind "${kind}" --arg first "${first}" --arg second "${second}" '
        def positive_uint64:
            type == "string" and test("^[1-9][0-9]*$") and
            (length < 20 or (length == 20 and . <= "18446744073709551615"));
        def records($name; $key):
            [.events[] | select(.type == ("structs.structs.Event" + $name))
                | [.attributes[] | select(.key == $key)]
                | if length == 1 then .[0].value | fromjson else error("ambiguous grid event") end];
        select((if has("code") then (.code == 0 or .code == "0") else true end)
            and (.txhash | ascii_upcase) == ($hash | ascii_upcase))
        | select(.height | positive_uint64 and
            (length < 19 or (length == 19 and . <= "9223372036854775807")))
        | if $kind == "allocation" then
            records("Allocation"; "allocation") | select(length == 1) | .[0]
            | select(.sourceObjectId == $first and .controller == $second and .type == "dynamic")
            | select((if has("destinationId") then .destinationId == "" else true end))
            | select(.index | positive_uint64)
            | select(.id == ("6-" + .index)) | .id
          else
            . as $tx | records("Substation"; "substation") | select(length == 1) | .[0]
            | select(.owner == $first)
            | select(.id | type == "string" and startswith("4-") and (.[2:] | positive_uint64))
            | .id as $id
            | $tx | records("Allocation"; "allocation")
            | map(select(.id == $second)) | select(length == 1) | .[0]
            | select(.index | positive_uint64)
            | select(.id == ("6-" + .index) and .destinationId == $id)
            | $id
          end') || return 1
    printf '%s\n' "${object_id}"
}

# Resolve the agreement from this transaction's typed event, never a list page.
get_opened_agreement_record() {
    local provider="$1" owner="$2" capacity="$3"
    local txhash delivered agreement_record
    txhash=$(_tx_json "${LAST_TX_OUTPUT}" | jq -er '.txhash | select(type == "string" and test("^[0-9A-Fa-f]{64}$"))') || return 1
    delivered=$(query query tx "${txhash}") || return 1
    agreement_record=$(echo "${delivered}" | jq -cer \
        --arg hash "${txhash}" --arg provider "${provider}" --arg owner "${owner}" --arg capacity "${capacity}" '
        select((.code == 0 or .code == "0") and (.txhash | ascii_upcase) == ($hash | ascii_upcase))
        | [.events[] | select(.type == "structs.structs.EventAgreement")] | select(length == 1)
        | [.[0].attributes[] | select(.key == "agreement")] | select(length == 1)
        | .[0].value | fromjson
        | select(.providerId == $provider and .owner == $owner and .capacity == $capacity)
        | select(.id | type == "string" and test("^11-[0-9]+$"))') || return 1
    printf '%s\n' "${agreement_record}"
}

get_opened_agreement_id() {
    local record
    record=$(get_opened_agreement_record "$1" "$2" "$3") || return 1
    echo "${record}" | jq -er '.id'
}

# Recover the fixture's own associations at one committed height.
get_fixture_grid() {
    local helper_dir
    helper_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
    python3 "${helper_dir}/fixture_grid_recovery.py" grid --player "$1" -- structsd ${PARAMS_QUERY}
}

# Match the source, destination and controller across the complete scoped index.
# Empty is allowed only when that complete inventory has no matching allocation.
get_fixture_feeder() {
    local helper_dir
    helper_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
    local arguments=(feeder --source "$1" --destination "$2" --controller "$3")
    if [ -n "${4:-}" ]; then arguments+=(--height "$4"); fi
    python3 "${helper_dir}/fixture_grid_recovery.py" "${arguments[@]}" -- structsd ${PARAMS_QUERY}
}

# get_balance: get the balance of a specific denom for an address
# Usage: get_balance <address> <denom>
get_balance() {
    local addr="$1"
    local denom="$2"
    local result
    result=$(query query bank balance "${addr}" "${denom}" | jq -er --arg d "${denom}" \
        '.balance | select(.denom == $d) | .amount | select(type == "string" and test("^[0-9]+$"))') || return 1
    printf '%s\n' "${result}"
}

# Read every leg of a changing balance comparison at one committed height.
# Pruned/unavailable historical state is a failure, never a retry at latest.
get_balance_at_height() {
    local address="$1" denom="$2" height="$3" response result
    python3 -c 'import re, sys
h = sys.argv[1]
assert re.fullmatch(r"[1-9][0-9]{0,18}", h) and int(h) <= 2**63-1, "invalid committed query height"' "${height}" || return 1
    response=$(query query bank balance "${address}" "${denom}" --height "${height}") || return 1
    result=$(printf '%s' "${response}" | jq -er --arg d "${denom}" \
        '.balance | select(.denom == $d) | .amount | select(type == "string" and test("^(0|[1-9][0-9]*)$"))') || return 1
    printf '%s\n' "${result}"
}

# Fresh single-validator fixtures must prove the entire delegation set, not
# select one matching row while ignoring duplicates, wrong owners or later pages.
get_fixture_delegation_shares() {
    local address="$1" validator="$2" presence="${3:-present}" output
    output=$(query query staking delegations "${address}" --page-limit 2) || return 1
    printf '%s\n' "${output}" | jq -er --arg address "${address}" --arg validator "${validator}" --arg presence "${presence}" '
        select(type == "object" and (keys - ["pagination", "delegation_responses"] | length == 0)) |
        select(.pagination | type == "object") |
        select(.pagination | (has("next_key") and has("nextKey")) | not) |
        select((.pagination.next_key == null or .pagination.next_key == "") and
               (.pagination.nextKey == null or .pagination.nextKey == "")) |
        .delegation_responses | if . == null then [] else . end | select(type == "array") |
        if $presence == "empty" then
            select(length == 0) | "0.000000000000000000"
        elif $presence == "present" then
            select(length == 1) | .[0].delegation |
            select(.delegator_address == $address and .validator_address == $validator) |
            .shares | select(type == "string" and test("^(0|[1-9][0-9]*)[.][0-9]{18}$") and test("[1-9]"))
        else empty end'
}

# Denomination-specific queries avoid pagination hiding a real asset or supply.
# Query errors are failures, not evidence that the amount is zero.
guild_supply() {
    local result
    result=$(query query bank total-supply-of "$1" | jq -er --arg d "$1" \
        '.amount | select(.denom == $d) | .amount | select(type == "string" and test("^[0-9]+$"))') || return 1
    printf '%s\n' "${result}"
}

# Resolve the protocol address from chain state; never infer a vault from a key.
# A missing owner/address must stop the phase rather than read as a zero balance.
get_vault_address() {
    local owner_id="$1"
    query query structs vault "${owner_id}" | jq -er --arg owner "${owner_id}" \
        'select(.ownerId == $owner) | .address | select(type == "string" and length > 0)'
}

# Fund only a shortfall from the faucet. Transaction fees stay on the signing key;
# economic collateral belongs in the vault. ualpha can fund an unprovisioned vault;
# the owning handler provisions its association before spending it.
fund_vault_alpha() {
    local owner_id="$1" minimum="$2" vault balance shortfall faucet
    vault=$(get_vault_address "${owner_id}") || return 1
    balance=$(get_balance "${vault}" ualpha) || return 1
    shortfall=$(python3 -c 'import re, sys
assert all(re.fullmatch(r"0|[1-9][0-9]*", value) for value in sys.argv[1:]), "invalid vault funding amount"
balance, minimum = map(int, sys.argv[1:])
print(max(minimum-balance, 0))' "${balance}" "${minimum}") || return 1
    if [ "${shortfall}" != "0" ]; then
        # Phase 1 funds guild creation before Phase 2 initializes BOB_ADDRESS.
        # Resolve the key here so early calls and resumed phases are independent.
        faucet=$(structsd ${PARAMS_KEYS} keys show bob | jq -er '.address | select(type == "string" and length > 0)') || return 1
        run_tx "Funding ${owner_id} vault with ${shortfall}ualpha" \
            tx bank send "${faucet}" "${vault}" "${shortfall}ualpha" --from bob
    fi
}

# The bank-address event is emitted once by AppendGuild. Guild snapshots may
# repeat as membership is committed, so validate every snapshot of that new ID.
get_created_guild_id() {
    local owner="$1" reactor="$2" endpoint="$3" station="$4" txhash delivered guild_id
    txhash=$(_tx_json "${LAST_TX_OUTPUT}" | jq -er '.txhash | select(type == "string" and test("^[0-9A-Fa-f]{64}$"))') || return 1
    delivered=$(query query tx "${txhash}") || return 1
    guild_id=$(echo "${delivered}" | jq -er --arg hash "${txhash}" --arg owner "${owner}" --arg reactor "${reactor}" --arg endpoint "${endpoint}" --arg station "${station}" '
        select((.code == 0 or .code == "0") and (.txhash | ascii_upcase) == ($hash | ascii_upcase))
        | .events as $events
        | [$events[] | select(.type == "structs.structs.EventGuildBankAddress")] | select(length == 1)
        | [.[0].attributes[] | select(.key == "eventGuildBankAddressDetail")] | select(length == 1)
        | .[0].value | fromjson | .guildId | select(type == "string" and test("^0-[0-9]+$")) as $id
        | [$events[] | select(.type == "structs.structs.EventGuild")
            | [.attributes[] | select(.key == "guild")]
            | if length == 1 then .[0].value | fromjson else error("invalid guild event attributes") end
            | select(.id == $id)]
        | select(length > 0 and all(.[]; .owner == $owner and .primaryReactorId == $reactor and .endpoint == $endpoint and (if has("entrySubstationId") then .entrySubstationId else "" end) == $station))
        | [$events[] | select(.type == "burn")] | select(length == 1)
        | [.[0].attributes[] | select(.key == "amount")] | select(length == 1 and .[0].value == "1000000ualpha")
        | [$events[] | select(.type == "coinbase")] | select(length == 1)
        | [.[0].attributes[] | select(.key == "amount")] | select(length == 1 and .[0].value == ("1000000uguild." + $id))
        | $id') || return 1
    printf '%s\n' "${guild_id}"
}

# The one founding path: pay from the founder vault and verify the new treasury.
create_paid_guild() {
    local key="$1" owner_id="$2" reactor_id="$3" endpoint="$4" station_id="$5"
    local founder_vault before expected_after guild_id guild_vault collateral vault_state guild_state founder_guild
    fund_vault_alpha "${owner_id}" 2000000 || return 1
    founder_vault=$(get_vault_address "${owner_id}") || return 1
    before=$(get_balance "${founder_vault}" ualpha) || return 1
    expected_after=$(python3 -c 'import re, sys; value = sys.argv[1]; assert re.fullmatch(r"0|[1-9][0-9]*", value); balance = int(value); assert balance >= 2000000; print(balance-2000000)' "${before}") || return 1
    run_tx "${key} creates a paid guild" \
        tx structs guild-create "${reactor_id}" "${endpoint}" "${station_id}" --from "${key}" || return 1
    guild_id=$(get_created_guild_id "${owner_id}" "${reactor_id}" "${endpoint}" "${station_id}") || return 1
    founder_guild=$(query query structs player "${owner_id}" | jq -er --arg owner "${owner_id}" '.Player | select(.id == $owner) | .guildId | select(type == "string" and test("^0-[0-9]+$"))') || return 1
    assert_eq "Founder joined the guild created by this transaction" "${guild_id}" "${founder_guild}"
    guild_vault=$(get_vault_address "${guild_id}") || return 1
    collateral=$(query query structs guild-bank-collateral-address "${guild_id}" | jq -er --arg guild "${guild_id}" '.internalAddressAssociation | select(type == "array" and length == 1) | .[0] | select(.objectId == $guild) | .address | select(type == "string" and length > 0)') || return 1
    vault_state=$(query query structs vault "${guild_id}") || return 1
    vault_state=$(echo "${vault_state}" | jq -ce --arg owner "${guild_id}" --arg address "${guild_vault}" 'select(.ownerId == $owner and .address == $address and (.provisioned | type == "boolean"))') || return 1
    guild_state=$(query query structs guild "${guild_id}" | jq -ce --arg id "${guild_id}" '.Guild | select(type == "object" and .id == $id)') || return 1
    assert_eq "Guild creation spends exactly two Alpha from founder vault" "${expected_after}" "$(get_balance "${founder_vault}" ualpha)"
    assert_eq "Initial guild token is in guild vault" "1000000" "$(get_balance "${guild_vault}" "uguild.${guild_id}")"
    assert_eq "Initial token has one Alpha collateral" "1000000" "$(get_balance "${collateral}" ualpha)"
    assert_eq "New guild treasury has no collateral Alpha" "0" "$(get_balance "${guild_vault}" ualpha)"
    assert_eq "Vault association was provisioned" "true" "$(echo "${vault_state}" | jq -r '.provisioned')"
    assert_eq "New guild belongs to the signing founder" "${owner_id}" "$(echo "${guild_state}" | jq -r '.owner')"
}

# ─── Charge-Aware Wait Helpers ────────────────────────────────────────────────
# Charge = CurrentBlockHeight - Player.lastAction
# Each block is ~1 second. Struct operations require a minimum charge level
# before they can proceed (e.g. BuildCharge=1 means 1 block since last action).

# get_block_height: query the current block height
get_block_height() {
    query query structs block-height | jq -r '.blockHeight // "0"' 2>/dev/null || echo "0"
}

# Capture every raid assertion at one committed height, including on resume.
raid_state_at_height() {
    local height="$1" victim raider planet fleet require_ore=true
    [[ "${height}" =~ ^[1-9][0-9]*$ ]] || return 1
    [ "${SKIP_MINING}" != true ] || require_ore=false
    victim=$(query query structs player "${PLAYER_2_ID}" --height "${height}") || return 1
    raider=$(query query structs player "${PLAYER_3_ID}" --height "${height}") || return 1
    planet=$(query query structs planet "${PLAYER_2_PLANET_ID}" --height "${height}") || return 1
    fleet=$(query query structs fleet "${PLAYER_3_FLEET_ID}" --height "${height}") || return 1
    jq -nc --arg height "${height}" --arg victim_id "${PLAYER_2_ID}" --arg raider_id "${PLAYER_3_ID}" \
        --arg planet_id "${PLAYER_2_PLANET_ID}" --arg home_id "${PLAYER_3_PLANET_ID}" --arg fleet_id "${PLAYER_3_FLEET_ID}" \
        --argjson require_ore "${require_ore}" --argjson victim "${victim}" --argjson raider "${raider}" \
        --argjson planet "${planet}" --argjson fleet "${fleet}" \
        '{height:$height,victim_id:$victim_id,raider_id:$raider_id,planet_id:$planet_id,home_id:$home_id,
          fleet_id:$fleet_id,require_ore:$require_ore,victim:$victim,raider:$raider,planet:$planet,fleet:$fleet}'
}

# wait_for_block: wait until the chain reaches a given height
# Used for state that only settles in the EndBlocker, such as agreement expiry.
# Usage: wait_for_block <target_height> [timeout_seconds]
wait_for_block() {
    local target="$1" timeout="${2:-90}"
    local height elapsed=0
    height=$(get_block_height)
    if [ "${height}" -ge "${target}" ] 2>/dev/null; then
        return 0
    fi
    echo -e "  ${YELLOW}Waiting for block${NC}: at ${height}, need ${target}"
    while [ "${elapsed}" -lt "${timeout}" ]; do
        sleep 2
        elapsed=$((elapsed + 2))
        height=$(get_block_height)
        if [ -n "${height}" ] && [ "${height}" -ge "${target}" ] 2>/dev/null; then
            echo -e "  ${GREEN}Reached block${NC}: ${height} >= ${target} (after ${elapsed}s)"
            return 0
        fi
    done
    echo -e "  ${RED}Timed out${NC} waiting for block ${target} (still at ${height})"
    return 1
}

# Record only the confirmed transaction's terms and transfers. The model prices
# each served window independently of the chain's settlement/checkpoint counters.
record_agreement_terms() {
    local operation="$1" amount="$2" state="$3" test_dir txhash delivered
    test_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd) || return 1
    txhash=$(_tx_json "${LAST_TX_OUTPUT}" | jq -er '.txhash | select(type == "string" and test("^[0-9A-Fa-f]{64}$"))') || return 1
    delivered=$(query query tx "${txhash}") || return 1
    printf '%s' "${delivered}" | python3 "${test_dir}/agreement_terms.py" step "${operation}" "${amount}" "${state}" "${txhash}" || return 1
}

# Preserve transaction-local evidence for the live-contract withdrawal/delete fixture.
record_provider_lifecycle_step() {
    local operation="$1" argument="$2" state="$3" test_dir txhash delivered
    test_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd) || return 1
    txhash=$(_tx_json "${LAST_TX_OUTPUT}" | jq -er '.txhash | select(type == "string" and test("^[0-9A-Fa-f]{64}$"))') || return 1
    delivered=$(query query tx "${txhash}") || return 1
    printf '%s' "${delivered}" | python3 "${test_dir}/provider_lifecycle_accounting.py" "${operation}" "${state}" "${argument}" "${txhash}" || return 1
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
CHARGE_BUILD=1
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
PERM_PROVIDER_OPEN=262144

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

# run_tx_expect_fail: execute a TX that we EXPECT to fail, and verify it does.
#
# This used to record nothing at all — no PASS_COUNT, no FAIL_COUNT, just a
# `return 1` into a script with no `set -e`, so all 40-odd negative tests were
# incapable of failing the suite. It was also defined twice, and the second
# definition silently won, which is why editing the first one changed nothing.
run_tx_expect_fail() {
    local description="$1"
    shift
    info "${description} (expect failure)"
    echo -e "  ${BOLD}structsd ${PARAMS_TX} $*${NC}"
    local OUTPUT OUTCOME
    OUTPUT=$(structsd ${PARAMS_TX} "$@" 2>&1) || true
    LAST_TX_OUTPUT="${OUTPUT}"
    OUTCOME=$(_tx_outcome "${OUTPUT}")
    _tx_outcome_known "${OUTCOME}" || return 0
    if [ "${OUTCOME%%|*}" = "ok" ]; then
        echo -e "  ${RED}FAIL${NC}: TX succeeded but was expected to fail"
        FAIL_COUNT=$((FAIL_COUNT + 1))
        return 0
    fi
    echo -e "  ${GREEN}PASS${NC}: TX correctly rejected"
    echo "  ${OUTCOME#*|}"
    PASS_COUNT=$((PASS_COUNT + 1))
    return 0
}

# run_tx_expect_permission_denied: expect failure with permission/authority message
run_tx_expect_permission_denied() {
    local description="$1"
    shift
    run_tx_expect_reject_matching "${description}" \
        'has no .*permission|permission denied|does not have .*permission|does not have .*authority|not have the authority|unauthorized|not authorized' "$@"
}

# run_tx_expect_reject_matching: expect failure AND pin the reason.
#
# run_tx_expect_fail passes on any failure, which cannot tell a deliberate
# refusal from a handler that happened to panic on the way to the same place, nor
# from a transaction that ran out of gas on its way to succeeding. That distinction
# is the whole point of some of these cases, so this variant takes a regex the
# rejection must match, and treats a panic as a failure even when the regex would
# otherwise match.
run_tx_expect_reject_matching() {
    local description="$1"
    local pattern="$2"
    shift 2
    info "${description}"
    echo -e "  ${BOLD}structsd ${PARAMS_TX} $*${NC}"
    local OUTPUT OUTCOME REASON
    OUTPUT=$(structsd ${PARAMS_TX} "$@" 2>&1) || true
    LAST_TX_OUTPUT="${OUTPUT}"
    OUTCOME=$(_tx_outcome "${OUTPUT}")
    _tx_outcome_known "${OUTCOME}" || return 0
    if [ "${OUTCOME%%|*}" = "ok" ]; then
        echo -e "  ${RED}FAIL${NC}: TX succeeded but was expected to fail"
        FAIL_COUNT=$((FAIL_COUNT + 1))
        return 0
    fi
    REASON="${OUTCOME#*|}"
    if echo "${REASON}" | grep -qi "panic"; then
        echo -e "  ${RED}FAIL${NC}: TX was rejected by a panic, not a check"
        echo "  ${REASON}"
        FAIL_COUNT=$((FAIL_COUNT + 1))
        return 0
    fi
    if echo "${REASON}" | grep -qiE "${pattern}"; then
        echo -e "  ${GREEN}PASS${NC}: TX rejected matching /${pattern}/"
        PASS_COUNT=$((PASS_COUNT + 1))
        return 0
    fi
    echo -e "  ${RED}FAIL${NC}: TX rejected, but not matching /${pattern}/"
    echo "  ${REASON}"
    FAIL_COUNT=$((FAIL_COUNT + 1))
    return 0
}

# ─── Permission Query Helpers ────────────────────────────────────────────────

get_permission_by_object() {
    local object_id="$1" page next='' seen='|' records='[]' page_records
    local pages=0
    while true; do
        pages=$((pages + 1))
        if [ "$pages" -gt 10000 ]; then
            echo "Permission pagination exceeded 10000 pages" >&2
            return 1
        fi
        if [ -z "$next" ]; then
            page=$(query query structs permission-by-object "$object_id" --page-limit 100) || return 1
        else
            page=$(query query structs permission-by-object "$object_id" --page-limit 100 --page-key "$next") || return 1
        fi
        page_records=$(echo "$page" | jq -ce --arg prefix "${object_id}@" '
            if type != "object" or (.pagination | type) != "object" then error("invalid permission page") else . end |
            (if has("permissionRecords") then .permissionRecords else [] end) |
            if type != "array" then error("invalid permission records") else . end |
            if all(.[]; type == "object" and (.permissionId | type == "string" and startswith($prefix)))
            then . else error("permission page contains another object") end') || return 1
        records=$(jq -cen --argjson old "$records" --argjson page "$page_records" '
            $old + $page | if (map(.permissionId) | unique | length) == length
            then . else error("duplicate permission row across pages") end') || return 1
        next=$(echo "$page" | jq -er '
            .pagination | (if has("next_key") and has("nextKey") then error("ambiguous permission cursor")
            elif has("next_key") then .next_key elif has("nextKey") then .nextKey else "" end) |
            (if . == null then "" else . end) |
            if type == "string" and test("^[A-Za-z0-9+/]*={0,2}$") then . else error("invalid permission cursor") end') || return 1
        if [ -z "$next" ]; then break; fi
        case "$seen" in
            *"|${next}|"*) echo "Permission pagination repeated a cursor" >&2; return 1 ;;
        esac
        seen="${seen}${next}|"
    done
    jq -cn --argjson rows "$records" '{permissionRecords: $rows}'
}

get_permission_value_for_player() {
    local perm_id="$1@$2"
    local json
    # Resolve the exact row. A filtered first page of permission-by-object can
    # omit this player, and a failed query is not evidence of zero permissions.
    json=$(query query structs permission "${perm_id}") || return 1
    echo "${json}" | jq -er --arg id "${perm_id}" '
        .permissionRecord |
        if type != "object" or .permissionId != $id then error("missing or mismatched permission record")
        else (if has("value") then .value else "0" end) end |
        if type != "string" then error("permission value must be a uint64 string")
        elif test("^(0|[1-9][0-9]*)$") | not then error("invalid permission value")
        elif length > 20 or (length == 20 and . > "18446744073709551615") then error("permission value exceeds uint64")
        else . end'
}

get_guild_rank_permission_by_object() {
    local object_id="$1" page next='' seen='|' records='[]' page_records
    local pages=0
    while true; do
        pages=$((pages + 1))
        if [ "$pages" -gt 10000 ]; then
            echo "Guild-rank permission pagination exceeded 10000 pages" >&2
            return 1
        fi
        if [ -z "$next" ]; then
            page=$(query query structs guild-rank-permission-by-object "$object_id" --page-limit 100) || return 1
        else
            page=$(query query structs guild-rank-permission-by-object "$object_id" --page-limit 100 --page-key "$next") || return 1
        fi
        page_records=$(echo "$page" | jq -ce --arg object "$object_id" '
            if type != "object" or (.pagination | type) != "object" then error("invalid permission page") else . end |
            (if has("guildRankPermissionRecords") then .guildRankPermissionRecords
             elif has("guild_rank_permission_records") then .guild_rank_permission_records else [] end) |
            if type != "array" then error("invalid permission records") else . end |
            if all(.[]; type == "object" and .objectId == $object and (.guildId | type == "string") and (.permissions | type == "string") and (.rank | type == "string"))
            then . else error("permission page contains another object") end') || return 1
        records=$(jq -cen --argjson old "$records" --argjson page "$page_records" '
            $old + $page | if (map([.guildId, .permissions]) | unique | length) == length
            then . else error("duplicate permission row across pages") end') || return 1
        next=$(echo "$page" | jq -er '
            .pagination | (if has("next_key") and has("nextKey") then error("ambiguous permission cursor")
            elif has("next_key") then .next_key elif has("nextKey") then .nextKey else "" end) |
            (if . == null then "" else . end) |
            if type == "string" and test("^[A-Za-z0-9+/]*={0,2}$") then . else error("invalid permission cursor") end') || return 1
        if [ -z "$next" ]; then break; fi
        case "$seen" in
            *"|${next}|"*) echo "Guild-rank permission pagination repeated a cursor" >&2; return 1 ;;
        esac
        seen="${seen}${next}|"
    done
    jq -cn --argjson rows "$records" '{guildRankPermissionRecords: $rows}'
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

# Decode the typed event from the last delivered rank transaction. Attribute
# values are JSON-encoded, including quoted uint64 values; compare as strings.
assert_rank_purchase_event() {
    local guild="$1" player="$2" previous="$3" rank="$4" price="$5" joined="$6"
    local txhash matched=false
    txhash=$(_tx_json "${LAST_TX_OUTPUT}" | jq -r '.txhash // empty')
    if [ -n "${txhash}" ]; then
        matched=$(query query tx "${txhash}" | jq -r \
            --arg guild "${guild}" --arg player "${player}" --arg previous "${previous}" \
            --arg rank "${rank}" --arg price "${price}" --argjson joined "${joined}" '
            [(.events // [])[] | select(.type == "structs.structs.EventGuildRankPurchase")
             | [.attributes[] | {key: .key, value: (.value | fromjson)}] | from_entries] as $events
            | ($events | length) == 1 and
              ($events[0] | .guildId == $guild and .playerId == $player and
               (.previousRank | tostring) == $previous and (.rank | tostring) == $rank and
               (.price | tostring) == $price and .denom == ("uguild." + $guild) and
               (.joined // false) == $joined)')
    fi
    assert_eq "Rank event identifies player, ranks, price, currency and admission" "true" "${matched}"
}

# Zero load is valid only inside a successful query for the requested provider.
get_provider_load() {
    local provider="$1" payload load
    payload=$(query query structs provider "${provider}") || return 1
    load=$(echo "${payload}" | jq -er --arg id "${provider}" '
        select(.Provider.id == $id) | .gridAttributes | select(type == "object")
        | if has("load") then .load else "0" end
        | select(type == "string" and test("^(0|[1-9][0-9]*)$")
          and (length < 20 or (length == 20 and . <= "18446744073709551615")))') || return 1
    printf '%s\n' "${load}"
}

# Phase 16b uses rate 1 ualpha and consumer penalty 0.25 (verified on creation).
# SDK settlement truncates the penalty, not the final refund.
expected_settlement_consumer_vault() {
    local before="$1" end="$2" capacity="$3" txhash delivered height
    txhash=$(_tx_json "${LAST_TX_OUTPUT}" | jq -er '.txhash | select(type == "string" and test("^[0-9A-Fa-f]{64}$"))') || return 1
    delivered=$(query query tx "${txhash}") || return 1
    height=$(echo "${delivered}" | jq -er --arg hash "${txhash}" 'select((.code == 0 or .code == "0") and (.txhash | ascii_upcase) == ($hash | ascii_upcase)) | .height | select(type == "string" and test("^[0-9]+$"))') || return 1
    python3 -c 'import sys
before, end, capacity, height = map(int, sys.argv[1:])
assert before >= 0 and 0 <= height <= end < 2**64 and 0 < capacity < 2**64
remaining = (end-height)*capacity
print(before+remaining-remaining//4)' "${before}" "${end}" "${capacity}" "${height}"
}

# Cancellation/trash can be swept before a latest-state query returns. Pin the
# assertion to the successful receipt, and fail if that historical state is missing.
struct_state_after_last_tx() {
    local id="$1" txhash receipt height
    txhash=$(_tx_json "${LAST_TX_OUTPUT}" | jq -er '.txhash | select(type == "string" and test("^[0-9A-Fa-f]{64}$"))') || return 1
    receipt=$(query query tx "${txhash}") || return 1
    height=$(printf '%s' "${receipt}" | jq -er --arg hash "${txhash}" '
        select((.code == 0 or .code == "0") and (.txhash | ascii_upcase) == ($hash | ascii_upcase))
        | .height | select(type == "string" and test("^[1-9][0-9]*$"))') || return 1
    combat_state_at_height "${id}" "${height}"
}

# A destroyed combatant remains queryable at its transaction height even after
# the live sweep. Accept omitted protobuf zeros, but never malformed values.
combat_state_at_height() {
    query query structs struct "$1" --height "$2" | jq -ce --arg id "$1" '
        select(.Struct.id == $id and (.structAttributes | type) == "object")
        | .structAttributes as $a
        | (if $a | has("health") then $a.health else "0" end) as $hp
        | select(($hp | type) == "string" and ($hp | test("^(0|[1-9][0-9]*)$")))
        | select(($hp | length) < 20 or (($hp | length) == 20 and $hp <= "18446744073709551615"))
        | select(if $a | has("isDestroyed") then ($a.isDestroyed | type) == "boolean" else true end)'
}

# Pin all generator baseline inputs to one committed height. Two rows suffice
# to reject a fixture with unexpected extra positions; pagination must be empty.
generator_fixture_snapshot() {
    local height="${1:-}" generator player positions catalog
    if [ -z "${height}" ]; then
        height=$(query query structs block-height | jq -er '.blockHeight | select(type == "string" and test("^[1-9][0-9]*$"))') || return 1
    fi
    python3 -c 'import re,sys; h=sys.argv[1]; assert re.fullmatch(r"[1-9][0-9]*", h) and int(h) <= 2**63-1' "${height}" || return 1
    generator=$(query query structs struct "${GENERATOR_STRUCT_ID}" --height "${height}") || return 1
    player=$(query query structs player "${PLAYER_4_ID}" --height "${height}") || return 1
    positions=$(query query structs infusion-all-by-destination "${GENERATOR_STRUCT_ID}" --page-limit 2 --height "${height}") || return 1
    catalog=$(query query structs struct-type 20 --height "${height}") || return 1
    jq -n --arg height "${height}" --argjson generator "${generator}" --argjson player "${player}" \
        --argjson positions "${positions}" --argjson catalog "${catalog}" \
        '{height:$height,generator:$generator,player:$player,positions:$positions,catalog:$catalog}'
}

# Read the generator at the confirmed attack height: a latest query can race
# its delayed sweep. Missing receipts/state must fail, never become zero health.
generator_state_after_attack() {
    local before="$1" txhash receipt outcome height state
    txhash=$(_tx_json "${LAST_TX_OUTPUT}" | jq -er '.txhash | select(type == "string" and test("^[0-9A-Fa-f]{64}$"))') || return 1
    receipt=$(query query tx "${txhash}") || return 1
    outcome=$(printf '%s' "${receipt}" | python3 "${GENERATOR_TEST_DIR}/generator_attack.py" "${txhash}" "${DESTROYER_STRUCT_ID}" "${GENERATOR_STRUCT_ID}" "${before}") || return 1
    height=$(printf '%s' "${outcome}" | jq -er '.height') || return 1
    state=$(query query structs struct "${GENERATOR_STRUCT_ID}" --height "${height}") || return 1
    printf '%s' "${state}" | jq -ce --arg id "${GENERATOR_STRUCT_ID}" --argjson expected "${outcome}" '
        select(.Struct.id == $id and (.structAttributes | type) == "object" and (.gridAttributes | type) == "object")
        | select((if .structAttributes | has("health") then .structAttributes.health else "0" end) == $expected.health)
        | select((if .structAttributes | has("isDestroyed") then .structAttributes.isDestroyed else false end) == $expected.destroyed)'
}

# Destruction retires the infusion in EndBlock. A transaction receipt alone
# cannot prove the deletion and capacity updates delivered to indexers.
generator_destruction_height() {
    local txhash receipt height block
    txhash=$(_tx_json "${LAST_TX_OUTPUT}" | jq -er '.txhash | select(type == "string" and test("^[0-9A-Fa-f]{64}$"))') || return 1
    receipt=$(query query tx "${txhash}") || return 1
    height=$(printf '%s' "${receipt}" | jq -er '.height | select(type == "string" and test("^[1-9][0-9]*$"))') || return 1
    block=$(query query block-results "${height}") || return 1
    jq -n --argjson receipt "${receipt}" --argjson block "${block}" '{receipt:$receipt,block:$block}' | \
        python3 "${GENERATOR_TEST_DIR}/generator_destruction.py" "${txhash}" "${DESTROYER_STRUCT_ID}" \
            "${GENERATOR_STRUCT_ID}" "${PLAYER_4_ID}" "${PLAYER_4_VAULT}" "${P4_CAP_EXPECTED_DESTROYED}"
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
        7) echo 700;; 7b) echo 750;; 7c) echo 760;; 8) echo 800;;
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

# A recovery/combat inventory genuinely needs every row. Pin all pages to one
# committed height so deferred sweeps cannot change the inventory between pages.
get_all_structs() {
    local helper_dir
    helper_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd) || return 1
    python3 "${helper_dir}/struct_inventory.py" collect -- structsd ${PARAMS_QUERY}
}

# find_struct_by_owner_type: select from a complete inventory, ordered by exact index.
# Usage: find_struct_by_owner_type <owner_player_id> <type_num> [nth] [struct_all_json]
find_struct_by_owner_type() {
    local owner="$1" type_num="$2" nth="${3:-1}" json="${4:-}"
    local helper_dir
    helper_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd) || return 1
    if [ -z "${json}" ]; then json=$(get_all_structs) || return 1; fi
    printf '%s\n' "${json}" | python3 "${helper_dir}/struct_inventory.py" select "${owner}" "${type_num}" "${nth}"
}

# find_last_struct_by_owner_type: the same lookup counted from the newest end, so
# nth=1 is the most recently built. A player can hold several structs of one type
# across phases and the later ones are the later phases' — AR1's defender pool
# rebuilds types the EB phases already used, and the EB one is usually the rubble.
find_last_struct_by_owner_type() {
    local owner="$1" type_num="$2" nth="${3:-1}" json="${4:-}"
    local helper_dir
    helper_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd) || return 1
    if [ -z "${json}" ]; then json=$(get_all_structs) || return 1; fi
    printf '%s\n' "${json}" | python3 "${helper_dir}/struct_inventory.py" select "${owner}" "${type_num}" "${nth}" --newest
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

# ar_live <id...>: echo the first id that still exists with health above 0, or
# nothing when every candidate is gone. eb_health answers "0" both for a destroyed
# struct and for one the sweep has already removed, which is exactly the
# distinction that does not matter to a caller looking for something to shoot at
# or defend with. Top-level because JS1 needs it as well as AR3.
ar_live() {
    local id
    for id in "$@"; do
        if [ -n "${id}" ] && [ "$(eb_health "${id}")" != "0" ]; then
            echo "${id}"
            return 0
        fi
    done
    echo ""
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

# Required combat roles are recovered by identity or rebuilt after combat attrition.
# Snapshot candidate queries at the inventory height so rubble sweeps cannot
# turn a known destroyed candidate into a misleading RPC failure mid-selection.
combat_fixture_state() {
    local id="$1" owner="$2" kind="$3" location="$4" ambit="$5" height="${6:-}" payload
    if [ -n "${height}" ]; then
        payload=$(query query structs struct "${id}" --height "${height}") || return 1
    else
        payload=$(query query structs struct "${id}") || return 1
    fi
    printf '%s' "${payload}" | python3 "${GENERATOR_TEST_DIR}/combat_fixture.py" fixture "${id}" "${owner}" "${kind}" "${location}" "${ambit}"
}

# Optional mode: ready completes/activates; prepare preserves the current build
# stage for parallel aging; pending requires an unbuilt role for refusal tests.
combat_require_struct() {
    local output="$1" player="$2" kind="$3" placement="$4" location="$5" ambit="$6" label="$7"
    local excluded="${8:-}" key="${9:-player_${2}}" preferred completion_mode="${10:-ready}"
    case "${completion_mode}" in ready|prepare|pending) ;; *) return 1 ;; esac
    local owner inventory height candidates candidate stage id="" slot home return_planet=""
    [[ "${output}" =~ ^[A-Z][A-Z0-9_]*$ && "${player}" =~ ^[1-9][0-9]*$ ]] || return 1
    eval "owner=\${PLAYER_${player}_ID}"
    eval "preferred=\${${output}:-}"
    inventory=$(get_all_structs) || return 1
    height=$(printf '%s' "${inventory}" | jq -er '.queryHeight | select(type == "string" and test("^[1-9][0-9]*$"))') || return 1
    candidates=$(printf '%s' "${inventory}" | jq -r --arg owner "${owner}" --arg kind "${kind}" --arg location "${location}" --arg ambit "${ambit}" \
        --arg preferred "${preferred}" --arg excluded "${excluded}" '
        ($excluded | split(" ") | map(select(. != ""))) as $excluded
        | .Struct | sort_by(.id != $preferred) | .[]
        | select(.owner == $owner and .type == $kind and .locationId == $location and .operatingAmbit == $ambit)
        | .id | select(. as $id | $excluded | index($id) | . == null)') || return 1
    for candidate in ${candidates}; do
        stage=$(combat_fixture_state "${candidate}" "${owner}" "${kind}" "${location}" "${ambit}" "${height}") || return 1
        if [ "${stage}" = destroyed ]; then continue; fi
        if [ "${completion_mode}" = pending ] && [ "${stage}" != build ]; then continue; fi
        id="${candidate}"; break
    done
    if [ -z "${id}" ]; then
        if [ "${placement}" = fleet ]; then
            eval "home=\${PLAYER_${player}_PLANET_ID}"
            return_planet=$(combat_fleet_location "${player}") || return 1
            combat_move_fleet "${player}" "${home}" "${key}" || return 1
        fi
        if [ "${kind}" = 1 ] && [ "${placement}" = fleet ]; then
            # Command ships use commandStruct, not an ordinary ambit slot.
            combat_wait_command_slot "${location}" || return 1
            slot=0
        else
            slot=$(wait_for_free_slot "${placement}" "${location}" "${ambit}") || return 1
        fi
        if ! [[ "${slot}" =~ ^(0|[1-9][0-9]*)$ ]]; then
            echo "FAIL: Combat fixture requires a free ${ambit} slot for ${label}" >&2
            FAIL_COUNT=$((FAIL_COUNT + 1)); return 1
        fi
        wait_for_charge "${owner}" "${CHARGE_BUILD}" || return 1
        run_tx "Combat fixture: Initiating ${label}" tx structs struct-build-initiate "${owner}" "${kind}" "${ambit}" "${slot}" --from "${key}" || return 1
        id=$(get_created_struct_id "${owner}" "${kind}") || return 1
        assert_new_struct "Combat fixture ${label}" "${id}" "${owner}" "${kind}" || return 1
    fi
    stage=$(combat_fixture_state "${id}" "${owner}" "${kind}" "${location}" "${ambit}") || return 1
    if [ "${completion_mode}" = ready ] && [ "${stage}" = "build" ]; then
        run_compute "Combat fixture: Building ${label} ${id}" tx structs struct-build-compute "${id}" --from "${key}" || return 1
        stage=$(combat_fixture_state "${id}" "${owner}" "${kind}" "${location}" "${ambit}") || return 1
    fi
    if [ "${completion_mode}" = ready ] && [ "${stage}" = "activate" ]; then
        wait_for_charge "${owner}" "${CHARGE_ACTIVATE}" || return 1
        run_tx "Combat fixture: Activating ${label} ${id}" tx structs struct-activate "${id}" --from "${key}" || return 1
        stage=$(combat_fixture_state "${id}" "${owner}" "${kind}" "${location}" "${ambit}") || return 1
    fi
    if { [ "${completion_mode}" = ready ] && [ "${stage}" != ready ]; } ||
       { [ "${completion_mode}" = pending ] && [ "${stage}" != build ]; } ||
       { [ "${stage}" != build ] && [ "${stage}" != activate ] && [ "${stage}" != ready ]; }; then
        echo "FAIL: Combat fixture ${label} ${id} does not satisfy ${completion_mode} (${stage})" >&2
        FAIL_COUNT=$((FAIL_COUNT + 1)); return 1
    fi
    if [ -n "${return_planet}" ]; then
        combat_move_fleet "${player}" "${return_planet}" "${key}" || return 1
    fi
    printf -v "${output}" '%s' "${id}"
}

# A Command Ship's ambit is movable; preserve it instead of treating a live
# command in another ambit as missing and attempting to build a duplicate.
combat_require_command() {
    local output="$1" player="$2" label="$3" owner fleet height payload command ambit=land resolved
    [[ "${output}" =~ ^[A-Z][A-Z0-9_]*$ && "${player}" =~ ^[1-9][0-9]*$ ]] || return 1
    eval "owner=\${PLAYER_${player}_ID}"
    eval "fleet=\${PLAYER_${player}_FLEET_ID}"
    # A destruction sweep between the fleet and struct reads would read as an error.
    height=$(query query structs block-height | jq -er '.blockHeight | select(type == "string" and test("^[1-9][0-9]*$"))') || return 1
    payload=$(query query structs fleet "${fleet}" --height "${height}") || return 1
    command=$(printf '%s' "${payload}" | jq -er --arg fleet "${fleet}" --arg owner "${owner}" '
        .Fleet | select(.id == $fleet and .owner == $owner)
        | (if has("commandStruct") then .commandStruct else "" end)
        | select(type == "string" and (. == "" or test("^5-[1-9][0-9]*$")))') || return 1
    if [ -n "${command}" ]; then
        payload=$(query query structs struct "${command}" --height "${height}") || return 1
        ambit=$(printf '%s' "${payload}" | jq -er --arg id "${command}" --arg owner "${owner}" --arg fleet "${fleet}" '
            .Struct | select(.id == $id and .owner == $owner and .type == "1" and .locationId == $fleet and .locationType == "fleet")
            | .operatingAmbit | select(. == "land" or . == "water" or . == "air" or . == "space")') || return 1
    fi
    combat_require_struct "${output}" "${player}" 1 fleet "${fleet}" "${ambit}" "${label}" || return 1
    resolved=$(get_command_ship_id "${owner}" "${fleet}") || return 1
    eval "command=\${${output}}"
    [ "${resolved}" = "${command}" ]
}

combat_wait_command_slot() {
    local fleet="$1" command elapsed=0
    while [ "${elapsed}" -lt 30 ]; do
        command=$(query query structs fleet "${fleet}" | jq -er --arg id "${fleet}" '.Fleet | select(.id == $id) | .commandStruct // ""') || return 1
        [ -n "${command}" ] || return 0
        sleep 3
        elapsed=$((elapsed + 3))
    done
    echo "FAIL: command ship slot did not clear for ${fleet}" >&2
    return 1
}

combat_fleet_location() {
    local player="$1" height="${2:-}" owner fleet payload
    eval "owner=\${PLAYER_${player}_ID}"
    eval "fleet=\${PLAYER_${player}_FLEET_ID}"
    if [ -n "${height}" ]; then
        payload=$(query query structs fleet "${fleet}" --height "${height}") || return 1
    else
        payload=$(query query structs fleet "${fleet}") || return 1
    fi
    printf '%s' "${payload}" | jq -er --arg id "${fleet}" --arg owner "${owner}" \
        '.Fleet | select(.id == $id and .owner == $owner) | .locationId | select(type == "string" and test("^2-[1-9][0-9]*$"))'
}

combat_move_fleet() {
    local player="$1" destination="$2" key="${3:-player_${1}}" owner fleet actual
    eval "owner=\${PLAYER_${player}_ID}"
    eval "fleet=\${PLAYER_${player}_FLEET_ID}"
    actual=$(combat_fleet_location "${player}") || return 1
    if [ "${actual}" != "${destination}" ]; then
        wait_for_charge "${owner}" "${CHARGE_MOVE}" || return 1
        run_tx "Combat fixture: Move ${fleet} to ${destination}" tx structs fleet-move "${fleet}" "${destination}" --from "${key}" || return 1
    fi
    actual=$(combat_fleet_location "${player}") || return 1
    assert_eq "Combat fixture fleet ${fleet} reached required planet" "${destination}" "${actual}"
    [ "${actual}" = "${destination}" ]
}

# Evasion isolates the unit defense from registered blockers and counters.
combat_clear_target_defenders() {
    local player="$1" target="$2" owner payload ids id kind location ambit stage
    eval "owner=\${PLAYER_${player}_ID}"
    payload=$(query query structs struct "${target}") || return 1
    ids=$(printf '%s' "${payload}" | jq -er --arg id "${target}" --arg owner "${owner}" '
        select(.Struct.id == $id and .Struct.owner == $owner) | (if has("structDefenders") then .structDefenders else [] end)
        | select(type == "array" and length == (unique | length))
        | if all(.[]; type == "string" and test("^5-[1-9][0-9]*$")) then join(" ") else error("invalid defender id") end') || return 1
    for id in ${ids}; do
        payload=$(query query structs struct "${id}") || return 1
        printf '%s' "${payload}" | jq -e --arg id "${id}" --arg owner "${owner}" --arg target "${target#5-}" \
            '.Struct.id == $id and .Struct.owner == $owner and .structAttributes.protectedStructIndex == $target' >/dev/null || return 1
        kind=$(printf '%s' "${payload}" | jq -er '.Struct.type') || return 1
        location=$(printf '%s' "${payload}" | jq -er '.Struct.locationId') || return 1
        ambit=$(printf '%s' "${payload}" | jq -er '.Struct.operatingAmbit') || return 1
        stage=$(printf '%s' "${payload}" | python3 "${GENERATOR_TEST_DIR}/combat_fixture.py" fixture "${id}" "${owner}" "${kind}" "${location}" "${ambit}") || return 1
        if [ "${stage}" = activate ]; then
            wait_for_charge "${owner}" "${CHARGE_ACTIVATE}" || return 1
            run_tx "Evasion: activate ${id} before clearing defense" tx structs struct-activate "${id}" --from "player_${player}" || return 1
        elif [ "${stage}" != ready ]; then
            echo "FAIL: evasion defender ${id} cannot be cleared (${stage})" >&2
            return 1
        fi
        wait_for_charge "${owner}" "${CHARGE_DEFEND}" || return 1
        run_tx "Evasion: clear ${id} from ${target}" tx structs struct-defense-clear "${id}" --from "player_${player}" || return 1
    done
    query query structs struct "${target}" | jq -e --arg id "${target}" --arg owner "${owner}" \
        '.Struct.id == $id and .Struct.owner == $owner and (if has("structDefenders") then .structDefenders == [] else true end)' >/dev/null
}

# ev_attack <case> <attacker-var/player/type/ambit> <target-var/player/type/ambit> <weapon> [planet] [offline-command]
# Every scheduled attempt restores its roles, verifies the catalog, then binds
# the receipt and canonical state to exact identities and committed heights.
ev_attack() {
    local case_id="$1" attacker_var="$2" attacker_player="$3" attacker_type="$4" attacker_ambit="$5"
    local target_var="$6" target_player="$7" target_type="$8" target_ambit="$9" weapon="${10}"
    local battlefield="${11:-${PLAYER_6_PLANET_ID:-}}" command="${12:-}" command_state
    local attacker_owner target_owner attacker_fleet target_fleet attacker target before_height before after catalog receipt txhash outcome height
    local attacker_state target_state attacker_catalog target_catalog attacker_location target_location charge
    [[ "${attacker_player}" =~ ^[1-9][0-9]*$ && "${target_player}" =~ ^[1-9][0-9]*$ && "${attacker_var}" =~ ^[A-Z][A-Z0-9_]*$ && "${target_var}" =~ ^[A-Z][A-Z0-9_]*$ ]] || return 1
    eval "attacker_owner=\${PLAYER_${attacker_player}_ID}; attacker_fleet=\${PLAYER_${attacker_player}_FLEET_ID}"
    eval "target_owner=\${PLAYER_${target_player}_ID}; target_fleet=\${PLAYER_${target_player}_FLEET_ID}"
    combat_require_struct "${attacker_var}" "${attacker_player}" "${attacker_type}" fleet "${attacker_fleet}" "${attacker_ambit}" "${case_id} attacker" || return 1
    combat_require_struct "${target_var}" "${target_player}" "${target_type}" fleet "${target_fleet}" "${target_ambit}" "${case_id} target" || return 1
    eval "attacker=\${${attacker_var}}; target=\${${target_var}}"
    combat_clear_target_defenders "${target_player}" "${target}" || return 1
    if [ "${weapon}" = primaryWeapon ]; then charge=$(_primary_charge "${attacker_type}"); else charge=$(_secondary_charge "${attacker_type}"); fi
    wait_for_charge "${attacker_owner}" "${charge}" || return 1
    before_height=$(query query structs block-height | jq -er '.blockHeight | select(type == "string" and test("^[1-9][0-9]*$"))') || return 1
    attacker_state=$(combat_state_at_height "${attacker}" "${before_height}") || return 1
    target_state=$(combat_state_at_height "${target}" "${before_height}") || return 1
    before=$(jq -n --argjson attacker "${attacker_state}" --argjson target "${target_state}" '{attacker:$attacker,target:$target}') || return 1
    if [ "${case_id}" = P14_OFFLINE ]; then
        command_state=$(combat_state_at_height "${command}" "${before_height}") || return 1
        before=$(jq -n --argjson before "${before}" --argjson command "${command_state}" '$before + {command:$command}') || return 1
    fi
    printf '%s' "${before}" | python3 "${GENERATOR_TEST_DIR}/evasion_attack.py" before "${case_id}" "${attacker}" "${target}" \
        "${attacker_owner}" "${target_owner}" "${attacker_fleet}" "${target_fleet}" "${command}" >/dev/null || return 1
    attacker_location=$(combat_fleet_location "${attacker_player}" "${before_height}") || return 1
    target_location=$(combat_fleet_location "${target_player}" "${before_height}") || return 1
    [ "${attacker_location}" = "${battlefield}" ] && [ "${target_location}" = "${battlefield}" ] || return 1
    attacker_catalog=$(query query structs struct-type "${attacker_type}" --height "${before_height}") || return 1
    target_catalog=$(query query structs struct-type "${target_type}" --height "${before_height}") || return 1
    catalog=$(jq -n --argjson attacker "${attacker_catalog}" --argjson target "${target_catalog}" '{attacker:$attacker,target:$target}') || return 1
    printf '%s' "${catalog}" | python3 "${GENERATOR_TEST_DIR}/evasion_attack.py" catalog "${case_id}" >/dev/null || return 1
    run_tx "${case_id}: Required evasion attempt ${attacker} -> ${target}" \
        tx structs struct-attack "${attacker}" "${target}" "${weapon}" --from "player_${attacker_player}" || return 1
    txhash=$(_tx_json "${LAST_TX_OUTPUT}" | jq -er '.txhash | select(type == "string" and test("^[0-9A-Fa-f]{64}$"))') || return 1
    receipt=$(query query tx "${txhash}") || return 1
    outcome=$(jq -n --argjson before "${before}" --argjson tx "${receipt}" '{before:$before,tx:$tx}' \
        | python3 "${GENERATOR_TEST_DIR}/evasion_attack.py" attack "${case_id}" "${txhash}") || return 1
    height=$(printf '%s' "${outcome}" | jq -er '.height') || return 1
    [ "${height}" -gt "${before_height}" ] || return 1
    attacker_state=$(combat_state_at_height "${attacker}" "${height}") || return 1
    target_state=$(combat_state_at_height "${target}" "${height}") || return 1
    after=$(jq -n --argjson attacker "${attacker_state}" --argjson target "${target_state}" '{attacker:$attacker,target:$target}') || return 1
    if [ "${case_id}" = P14_OFFLINE ]; then
        command_state=$(combat_state_at_height "${command}" "${height}") || return 1
        after=$(jq -n --argjson after "${after}" --argjson command "${command_state}" '$after + {command:$command}') || return 1
    fi
    jq -n --argjson before "${before}" --argjson after "${after}" --argjson outcome "${outcome}" '{before:$before,after:$after,outcome:$outcome}' \
        | python3 "${GENERATOR_TEST_DIR}/evasion_attack.py" after "${case_id}" >/dev/null || return 1
    attacker_location=$(combat_fleet_location "${attacker_player}" "${height}") || return 1
    target_location=$(combat_fleet_location "${target_player}" "${height}") || return 1
    [ "${attacker_location}" = "${battlefield}" ] && [ "${target_location}" = "${battlefield}" ] || return 1
    if [[ "${case_id}" = EV* ]]; then
        EV_CONFIRMED=$((EV_CONFIRMED + 1))
        EV_EVADED=$((EV_EVADED + $(printf '%s' "${outcome}" | jq 'if .evaded then 1 else 0 end')))
    fi
    EB_ATTACKS=$((${EB_ATTACKS:-0} + 1))
    EB_DESTROYED=$((${EB_DESTROYED:-0} + $(printf '%s' "${outcome}" | jq '[.attackerDestroyed,.targetDestroyed] | map(select(. == true)) | length')))
    assert_eq "${case_id}: receipt and canonical evasion outcome verified" true "$(printf '%s' "${outcome}" | jq -r '.verified')"
    if [ "${case_id}" = EB_B6 ]; then
        printf 'EB outcome: %s\n' "$(printf '%s' "${outcome}" | jq -c --arg beforeHeight "${before_height}" '. + {beforeHeight:$beforeHeight}')"
    elif [[ "${case_id}" = P14* || "${case_id}" = P11 ]]; then
        printf '%s outcome: %s\n' "${case_id%%_*}" "$(printf '%s' "${outcome}" | jq -c --arg beforeHeight "${before_height}" '. + {beforeHeight:$beforeHeight}')"
    else
        printf 'EV outcome: %s\n' "$(printf '%s' "${outcome}" | jq -c --arg beforeHeight "${before_height}" '. + {beforeHeight:$beforeHeight}')"
    fi
}

# Each batch transition reads both ships at a shared height and verifies the
# delivered receipt plus state at that receipt's height. Failed queries refuse.
p12_batch_step() {
    local mode="$1" before_height first second before txhash receipt height after outcome
    case "${mode}" in reject|offline|first|second) ;; *) return 1 ;; esac
    if [ "${mode}" = first ] || [ "${mode}" = second ]; then
        wait_for_charge "${PLAYER_3_ID}" "${CHARGE_ACTIVATE}" || return 1
    fi
    before_height=$(query query structs block-height | jq -er '.blockHeight | select(type == "string" and test("^[1-9][0-9]*$"))') || return 1
    first=$(combat_state_at_height "${BATTLESHIP_1_ID}" "${before_height}") || return 1
    second=$(combat_state_at_height "${BATTLESHIP_2_ID}" "${before_height}") || return 1
    before=$(jq -n --argjson first "${first}" --argjson second "${second}" '{first:$first,second:$second}') || return 1
    printf '%s' "${before}" | python3 "${GENERATOR_TEST_DIR}/batch_deactivation.py" before "${mode}" \
        "${BATTLESHIP_1_ID}" "${BATTLESHIP_2_ID}" "${PLAYER_3_ID}" "${PLAYER_3_FLEET_ID}" >/dev/null || return 1
    case "${mode}" in
        reject) run_tx_expect_fail_noauto "P12: Batch rejects missing ship atomically" \
            tx structs struct-deactivate-batch "${BATTLESHIP_1_ID},5-18446744073709551615" --from player_3 || return 1 ;;
        offline) run_tx "P12: Deactivate both Battleships" \
            tx structs struct-deactivate-batch "${BATTLESHIP_1_ID},${BATTLESHIP_2_ID}" --from player_3 || return 1 ;;
        first) run_tx "P12: Reactivate first Battleship" tx structs struct-activate "${BATTLESHIP_1_ID}" --from player_3 || return 1 ;;
        second) run_tx "P12: Reactivate second Battleship" tx structs struct-activate "${BATTLESHIP_2_ID}" --from player_3 || return 1 ;;
    esac
    txhash=$(_tx_json "${LAST_TX_OUTPUT}" | jq -er '.txhash | select(type == "string" and test("^[0-9A-Fa-f]{64}$"))') || return 1
    receipt=$(query query tx "${txhash}") || return 1
    height=$(printf '%s' "${receipt}" | jq -er '.height | select(type == "string" and test("^[1-9][0-9]*$"))') || return 1
    [ "${height}" -gt "${before_height}" ] || return 1
    first=$(combat_state_at_height "${BATTLESHIP_1_ID}" "${height}") || return 1
    second=$(combat_state_at_height "${BATTLESHIP_2_ID}" "${height}") || return 1
    after=$(jq -n --argjson first "${first}" --argjson second "${second}" '{first:$first,second:$second}') || return 1
    outcome=$(jq -n --argjson before "${before}" --argjson tx "${receipt}" --argjson after "${after}" '{before:$before,tx:$tx,after:$after}' \
        | python3 "${GENERATOR_TEST_DIR}/batch_deactivation.py" verify "${mode}" "${txhash}") || return 1
    assert_eq "P12 ${mode}: receipt and both ships verified" true "$(printf '%s' "${outcome}" | jq -r '.verified')"
    printf 'P12 outcome: %s\n' "$(printf '%s' "${outcome}" | jq -c --arg beforeHeight "${before_height}" '. + {beforeHeight:$beforeHeight}')"
}

# Helper: pin the intended targeting refusal; incidental errors are failures.
eb_attack_should_fail() {
    local desc="$1" attacker="$2" target="$3" from_player="$4" reason="${5:-out_of_range}"
    local atk_hp_before tgt_hp_before charge
    case "${reason}" in hidden|out_of_range) ;; *) echo "Invalid negative attack reason" >&2; return 1 ;; esac
    atk_hp_before=$(eb_health "${attacker}")
    tgt_hp_before=$(eb_health "${target}")
    if ! [[ "${atk_hp_before}" =~ ^[1-9][0-9]*$ && "${tgt_hp_before}" =~ ^[1-9][0-9]*$ ]]; then
        echo "FAIL: Required negative attack fixture is missing or destroyed: ${desc}" >&2
        FAIL_COUNT=$((FAIL_COUNT + 1))
        return 1
    fi
    info "[Negative] ${desc}"
    charge=$(eb_get_charge "${attacker}" "primaryWeapon")
    wait_for_charge "$(eval echo "\${PLAYER_${from_player}_ID}")" "${charge}"
    run_tx_expect_reject_matching "${desc}" \
        "target struct [(]${target}[)] is ${reason} from attacker struct [(]${attacker}[)]" \
        tx structs struct-attack "${attacker}" "${target}" primaryWeapon --from "player_${from_player}"
    assert_eq "${desc} — target HP unchanged" "${tgt_hp_before}" "$(eb_health "${target}")"
    assert_eq "${desc} — attacker HP unchanged" "${atk_hp_before}" "$(eb_health "${attacker}")"
}

# recover_state: rebuild all script variables by querying the running chain
recover_state() {
    info "Recovering state from chain for resume..."

    PLAYER_1_ADDRESS=$(structsd ${PARAMS_KEYS} keys show alice 2>/dev/null | jq -r .address || echo "")
    # Phase 0 / GP1 still reference ALICE_ADDRESS; keep both names in sync on resume.
    ALICE_ADDRESS="${PLAYER_1_ADDRESS}"
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

    local FIXTURE_GRID FIXTURE_GRID_HEIGHT
    FIXTURE_GRID=$(get_fixture_grid "${PLAYER_1_ID}") || return 1
    FIXTURE_GRID_HEIGHT=$(jqr "${FIXTURE_GRID}" '.queryHeight')
    GUILD_ID=$(jqr "${FIXTURE_GRID}" '.guildId')
    REACTOR_ID=$(jqr "${FIXTURE_GRID}" '.reactorId')
    SUBSTATION_ID=$(jqr "${FIXTURE_GRID}" '.substationId')
    P1_ALLOC_ID=$(jqr "${FIXTURE_GRID}" '.allocationId')
    VALIDATOR_ADDRESS=$(jqr "${FIXTURE_GRID}" '.validatorAddress')
    assert_not_empty "Recovered validator" "${VALIDATOR_ADDRESS}"
    GUILD_TOKEN_DENOM="uguild.${GUILD_ID}"
    echo "  Guild A: ${GUILD_ID}  Reactor: ${REACTOR_ID}  Substation: ${SUBSTATION_ID}  P1 alloc: ${P1_ALLOC_ID:-none}"

    # Recover guild leaders and guilds B/C/D
    for LEADER_SUFFIX in b c d; do
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
    GUILD_D_ID=$(query query structs player "${GUILD_LEADER_D_ID:-}" 2>/dev/null | jq -r '.Player.guildId // empty' 2>/dev/null || echo "")
    echo "  Guild B: ${GUILD_B_ID:-?}  Guild C: ${GUILD_C_ID:-?}  Guild D: ${GUILD_D_ID:-?}"

    for PLAYER_NUM in 2 3 4; do
        eval "local PID=\${PLAYER_${PLAYER_NUM}_ID:-}"
        if [ -z "${PID}" ]; then continue; fi
        local PJSON
        PJSON=$(query query structs player "${PID}" 2>/dev/null || echo '{}')
        eval "PLAYER_${PLAYER_NUM}_PLANET_ID=$(jqr "${PJSON}" '.Player.planetId')"
        eval "PLAYER_${PLAYER_NUM}_FLEET_ID=$(jqr "${PJSON}" '.Player.fleetId')"
        local ALLOC
        ALLOC=$(get_fixture_feeder "${PID}" "${SUBSTATION_ID}" "${PLAYER_1_ID}" "${FIXTURE_GRID_HEIGHT}") || return 1
        eval "P${PLAYER_NUM}_ALLOC_ID=${ALLOC}"
    done
    echo "  P2 planet=${PLAYER_2_PLANET_ID:-?} fleet=${PLAYER_2_FLEET_ID:-?}"
    echo "  P3 planet=${PLAYER_3_PLANET_ID:-?} fleet=${PLAYER_3_FLEET_ID:-?}"
    echo "  P4 planet=${PLAYER_4_PLANET_ID:-?} fleet=${PLAYER_4_FLEET_ID:-?}"

    local SA
    SA=$(get_all_structs)

    COMMAND_SHIP_ID=$(find_struct_by_owner_type "${PLAYER_3_ID}" 1 1 "${SA}")
    PLAYER_2_CMD_SHIP_ID=$(find_struct_by_owner_type "${PLAYER_2_ID}" 1 1 "${SA}")
    PLAYER_3_CMD_SHIP_ID="${COMMAND_SHIP_ID}"
    MINER_STRUCT_ID=$(find_struct_by_owner_type "${PLAYER_2_ID}" 14 1 "${SA}")
    # Ore Refinery is type 15 (type 16 is unused / different); a wrong lookup
    # silently blanks REFINERY_STRUCT_ID and skips refine-during-raid asserts.
    REFINERY_STRUCT_ID=$(find_struct_by_owner_type "${PLAYER_2_ID}" 15 1 "${SA}")
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
        P6_ALLOC_ID=$(get_fixture_feeder "${PLAYER_6_ID}" "${SUBSTATION_ID}" "${PLAYER_1_ID}" "${FIXTURE_GRID_HEIGHT}") || return 1
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

        # AR3's defender pool, built in AR1. Recovered from the end of each type's
        # list rather than the start, because the EB-era struct of the same type is
        # the earlier id and is usually the one that is already rubble. Whatever
        # cannot be found stays empty, which AR3's ar_live guards read as "skip".
        AR_P6_CS_ID=$(find_last_struct_by_owner_type "${PLAYER_6_ID}" 1 1 "${SA}")
        AR_P6_BB1_ID=$(find_last_struct_by_owner_type "${PLAYER_6_ID}" 2 1 "${SA}")
        AR_P6_BB2_ID=$(find_last_struct_by_owner_type "${PLAYER_6_ID}" 2 2 "${SA}")
        AR_P6_MA_ID=$(find_last_struct_by_owner_type "${PLAYER_6_ID}" 8 1 "${SA}")
        AR_P6_DD_ID=$(find_last_struct_by_owner_type "${PLAYER_6_ID}" 12 1 "${SA}")
        AR_P6_SPARE1_ID=$(find_last_struct_by_owner_type "${PLAYER_6_ID}" 4 1 "${SA}")
        AR_P6_SPARE2_ID=$(find_last_struct_by_owner_type "${PLAYER_6_ID}" 4 2 "${SA}")
        echo "  P6 AR pool: target=${AR_P6_CS_ID:-?} blockers=${AR_P6_BB1_ID:-?},${AR_P6_BB2_ID:-?}"
        echo "              land=${AR_P6_MA_ID:-?} water=${AR_P6_DD_ID:-?} spares=${AR_P6_SPARE1_ID:-?},${AR_P6_SPARE2_ID:-?}"
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

GENERATOR_TEST_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "${GENERATOR_TEST_DIR}/attack_run.sh"
source "${GENERATOR_TEST_DIR}/defender_counter_fixture.sh"
source "${GENERATOR_TEST_DIR}/dead_bomber.sh"
source "${GENERATOR_TEST_DIR}/phase13_combat.sh"
source "${GENERATOR_TEST_DIR}/blocking_attack.sh"
source "${GENERATOR_TEST_DIR}/pdc_attack.sh"
source "${GENERATOR_TEST_DIR}/jamming_attack.sh"

# Arrays for rank-test players (populated by Phase 4e2, recovered by recover_state)
declare -a RP_IDS
declare -a RP_KEYS

if [ -n "${RESUME_FROM}" ]; then
    recover_state
fi

# ─── Fresh-chain precondition ─────────────────────────────────────────────────
# Every phase assumes it is the only writer: player/planet/fleet ids, struct
# slots and charge budgets are all hardcoded against a genesis-only chain.
# Running against leftover state produces slot collisions dozens of phases in
# (and hours of proof-of-work later), so refuse up front instead. --resume-from
# deliberately targets an in-progress chain and is exempt.
assert_fresh_chain() {
    local helper_dir snapshot
    helper_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd) || return 1
    # Presence queries share one committed height and stop at the first excess
    # row. Transport errors or incomplete inventories must never authorize writes.
    snapshot=$(python3 "${helper_dir}/fresh_chain_inventory.py" -- structsd ${PARAMS_QUERY}) || return 1
    info "Fresh chain confirmed: ${snapshot}"
}

if [ -z "${RESUME_FROM}" ]; then
    assert_fresh_chain
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

run_tx "Creating allocation from Player 1" \
    tx structs allocation-create "${PLAYER_1_ID}" "${PLAYER_1_CAPACITY}" \
    --allocation-type dynamic --from alice

# Resolve the allocation from the confirmed creation transaction.
P1_ALLOC_ID=$(get_created_grid_id allocation "${PLAYER_1_ID}" "${PLAYER_1_ID}")
assert_not_empty "Player 1 allocation ID" "${P1_ALLOC_ID}"
echo "  Player 1 Allocation ID: ${P1_ALLOC_ID}"

# Verify allocation details
P1_ALLOC_JSON=$(query query structs allocation "${P1_ALLOC_ID}")
P1_ALLOC_SRC=$(jqr "${P1_ALLOC_JSON}" '.Allocation.sourceObjectId')
assert_eq "Allocation source is Player 1" "${PLAYER_1_ID}" "${P1_ALLOC_SRC}"

# ─── Create Substation 1 ───
run_tx "Creating Substation 1" \
    tx structs substation-create "${PLAYER_1_ID}" "${P1_ALLOC_ID}" --from alice

# Verify the substation event and its feeder update belong to this transaction.
SUBSTATION_ID=$(get_created_grid_id substation "${PLAYER_1_ID}" "${P1_ALLOC_ID}")
assert_not_empty "Substation ID" "${SUBSTATION_ID}"
echo "  Substation ID: ${SUBSTATION_ID}"

# Verify substation exists
SUB_JSON=$(query query structs substation "${SUBSTATION_ID}")
SUB_CHECK=$(jqr "${SUB_JSON}" '.Substation.id')
assert_eq "Substation exists" "${SUBSTATION_ID}" "${SUB_CHECK}"

# ─────────────────────────────────────────────────────────────────────────────
# Throwaway substations
#
# Three later phases need a second substation to test against. v0.22.0 refuses
# MsgSubstationCreate when the named allocation already feeds a substation
# ("already feeds substation … and cannot create another"), so these helpers
# mint a fresh unused allocation carved out of Alice's main feeder rather than
# reusing P1_ALLOC_ID.
#
# P1_ALLOC_ID stays pointed at the allocation that powers SUBSTATION_ID for the
# whole run. TEMP_ALLOC_ID is the throwaway; SubstationCache.Delete destroys
# allocations still pointing into the deleted substation, so the destroy path
# deliberately does not move TEMP_ALLOC_ID home first.
# ─────────────────────────────────────────────────────────────────────────────

TEMP_ALLOC_ID=""
TEMP_ALLOC_POWER=100000

# temp_substation_create <out_var> <label>
# Writes the new substation id to out_var. A setup failure is recorded before
# returning non-zero, so callers can omit dependent steps without passing them.
temp_substation_create() {
    local out_var="$1" label="$2"
    local main_power new_main after temp_dst main_after
    eval "${out_var}=''"
    TEMP_ALLOC_ID=""

    if [ -z "${P1_ALLOC_ID}" ]; then
        if ! P1_ALLOC_ID=$(get_fixture_feeder "${PLAYER_1_ID}" "${SUBSTATION_ID}" "${PLAYER_1_ID}"); then
            assert_eq "${label}: main feeder recovery succeeded" "true" "false"
            return 1
        fi
    fi
    if [ -z "${P1_ALLOC_ID}" ]; then
        assert_not_empty "${label}: required Player 1 allocation exists" "${P1_ALLOC_ID}"
        return 1
    fi

    main_power=$(jqr "$(query query structs allocation "${P1_ALLOC_ID}")" '.gridAttributes.power' '0')
    if [ -z "${main_power}" ] || [ "${main_power}" -le "${TEMP_ALLOC_POWER}" ] 2>/dev/null; then
        assert_gt "${label}: main allocation has fixture headroom" "${TEMP_ALLOC_POWER}" "${main_power}"
        return 1
    fi

    new_main=$((main_power - TEMP_ALLOC_POWER))
    run_tx "Freeing ${TEMP_ALLOC_POWER} from ${P1_ALLOC_ID} for ${label}" \
        tx structs allocation-update "${P1_ALLOC_ID}" "${new_main}" --from alice

    main_after=$(jqr "$(query query structs allocation "${P1_ALLOC_ID}")" '.gridAttributes.power' 'invalid')
    if [ "${main_after}" != "${new_main}" ]; then
        assert_eq "${label}: source power was reduced before fixture creation" "${new_main}" "${main_after}"
        return 1
    fi

    run_tx "Creating unused throwaway allocation for ${label}" \
        tx structs allocation-create "${PLAYER_1_ID}" "${TEMP_ALLOC_POWER}" \
        --controller "${PLAYER_1_ID}" --allocation-type dynamic --from alice

    TEMP_ALLOC_ID=$(get_created_grid_id allocation "${PLAYER_1_ID}" "${PLAYER_1_ID}") || TEMP_ALLOC_ID=""
    if [ -z "${TEMP_ALLOC_ID}" ] || [ "${TEMP_ALLOC_ID}" = "${P1_ALLOC_ID}" ]; then
        assert_eq "${label}: a new throwaway allocation appeared" "true" "false"
        TEMP_ALLOC_ID=""
        run_tx "Restoring ${P1_ALLOC_ID} power after failed throwaway mint (${label})" \
            tx structs allocation-update "${P1_ALLOC_ID}" "${main_power}" --from alice
        return 1
    fi

    temp_dst=$(jqr "$(query query structs allocation "${TEMP_ALLOC_ID}")" '.Allocation.destinationId' '')
    if [ -n "${temp_dst}" ]; then
        assert_eq "${label}: throwaway allocation is initially disconnected" "" "${temp_dst}"
        TEMP_ALLOC_ID=""
        return 1
    fi

    run_tx "Creating temporary substation for ${label}" \
        tx structs substation-create "${PLAYER_1_ID}" "${TEMP_ALLOC_ID}" --from alice

    after=$(get_created_grid_id substation "${PLAYER_1_ID}" "${TEMP_ALLOC_ID}") || after=""
    if [ -z "${after}" ] || [ "${after}" = "${SUBSTATION_ID}" ]; then
        assert_eq "${label}: a new temporary substation appeared" "true" "false"
        TEMP_ALLOC_ID=""
        return 1
    fi

    eval "${out_var}='${after}'"
    info "Temporary substation for ${label}: ${after} (alloc ${TEMP_ALLOC_ID})"
}

# temp_substation_return_allocation <label>
# Moves the throwaway allocation onto the main substation when a caller needs
# the allocation to survive a subsequent delete. Prefer temp_substation_destroy,
# which lets Delete consume the throwaway instead.
temp_substation_return_allocation() {
    local label="$1" dst
    [ -z "${TEMP_ALLOC_ID}" ] && return 0

    dst=$(query query structs allocation "${TEMP_ALLOC_ID}" 2>/dev/null | jq -r '.Allocation.destinationId // empty' 2>/dev/null || echo "")
    if [ "${dst}" = "${SUBSTATION_ID}" ]; then
        return 0
    fi
    if [ -z "${dst}" ] && [ -z "$(query query structs allocation "${TEMP_ALLOC_ID}" 2>/dev/null | jq -r '.Allocation.id // empty' 2>/dev/null || echo "")" ]; then
        TEMP_ALLOC_ID=""
        return 0
    fi

    run_tx "Returning throwaway allocation ${TEMP_ALLOC_ID} to ${SUBSTATION_ID} after ${label}" \
        tx structs substation-allocation-connect "${TEMP_ALLOC_ID}" "${SUBSTATION_ID}" --from alice
}

# temp_substation_destroy <sub_id> <label>
# Deletes the throwaway substation. The throwaway allocation still feeds it, so
# Delete destroys that allocation; P1_ALLOC_ID is untouched.
temp_substation_destroy() {
    local sub_id="$1" label="$2"
    [ -z "${sub_id}" ] && return 0
    run_tx "Deleting temporary substation ${sub_id} (${label})" \
        tx structs substation-delete "${sub_id}" "${SUBSTATION_ID}" --from alice
    TEMP_ALLOC_ID=""
}

# ─── Discover Reactor (created during validator setup) ───
info "Looking up reactor"
REACTOR_ALL_JSON=$(query query structs reactor-all)
REACTOR_ID=$(jqr "${REACTOR_ALL_JSON}" '.Reactor[0].id')
assert_not_empty "Reactor ID" "${REACTOR_ID}"
echo "  Reactor ID: ${REACTOR_ID}"

REACTOR_VAL=$(jqr "${REACTOR_ALL_JSON}" '.Reactor[0].validator')
assert_eq "Reactor validator matches" "${VALIDATOR_ADDRESS}" "${REACTOR_VAL}"

# ─── Create Guild ───
# Reactor choice is a preference, not a consumed founding entitlement.
create_paid_guild alice "${PLAYER_1_ID}" "${REACTOR_ID}" "oh.energy" "${SUBSTATION_ID}"

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
# Bootstrap stakes use twice the old fuel budget so a 50% entry rate supplies
# the prior pre-commission energy. Increase funding by the added stake only,
# preserving the 5M signing-key balance used by later address/asset fixtures.
BOB_ADDRESS=$(structsd ${PARAMS_KEYS} keys show bob | jq -r .address)
info "Bob (faucet) address: ${BOB_ADDRESS}"

for PLAYER_NUM in 2 3 4 5; do
    eval "PADDR=\${PLAYER_${PLAYER_NUM}_ADDRESS}"
    run_tx "Sending 15000000ualpha from bob to player_${PLAYER_NUM}" \
        tx bank send "${BOB_ADDRESS}" "${PADDR}" 15000000ualpha --from bob
done

# ─── Delegate to validator (creates player accounts on the structs module) ───
for PLAYER_NUM in 2 3 4 5; do
    run_tx "Delegating 10000000ualpha from player_${PLAYER_NUM} to validator" \
        tx staking delegate "${VALIDATOR_ADDRESS}" 10000000ualpha --from "player_${PLAYER_NUM}"

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
# Guild creation moves the founder into the new guild and a guild's owner may not
# found a second one, so alice — who owns Guild A — cannot create any more.
# Dedicated leader accounts instead.
#
# Reactor ownership grants are unnecessary for founding; B, C and D exercise
# creation without any such grant.

section "Create Guild Leaders (B, C, D)"

for LEADER_SUFFIX in b c d; do
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
assert_not_empty "Guild Leader D address" "${GUILD_LEADER_D_ADDRESS}"

# Fund guild leaders (from alice to avoid draining bob's faucet budget)
for LEADER_SUFFIX in B C D; do
    LEADER_LOWER=$(echo "${LEADER_SUFFIX}" | tr '[:upper:]' '[:lower:]')
    eval "LADDR=\${GUILD_LEADER_${LEADER_SUFFIX}_ADDRESS}"
    run_tx "Sending 15000000ualpha from alice to guild_leader_${LEADER_LOWER}" \
        tx bank send "${PLAYER_1_ADDRESS}" "${LADDR}" 15000000ualpha --from alice
done

# Delegate guild leaders (creates player accounts)
for LEADER_SUFFIX in B C D; do
    LEADER_LOWER=$(echo "${LEADER_SUFFIX}" | tr '[:upper:]' '[:lower:]')
    run_tx "Delegating 10000000ualpha from guild_leader_${LEADER_LOWER} to validator" \
        tx staking delegate "${VALIDATOR_ADDRESS}" 10000000ualpha --from "guild_leader_${LEADER_LOWER}"

    eval "LADDR=\${GUILD_LEADER_${LEADER_SUFFIX}_ADDRESS}"
    ADDR_JSON=$(query query structs address "${LADDR}")
    LID=$(jqr "${ADDR_JSON}" '.playerId')
    eval "GUILD_LEADER_${LEADER_SUFFIX}_ID=${LID}"
    assert_not_empty "Guild Leader ${LEADER_SUFFIX} ID" "${LID}"
    echo "  Guild Leader ${LEADER_SUFFIX} ID: ${LID}"
done

# Grant PermSubstationConnection (1024) on substation to each leader
run_tx "Granting PermSubstationConnection on substation to Guild Leader B" \
    tx structs permission-grant-on-object "${SUBSTATION_ID}" "${GUILD_LEADER_B_ID}" 1024 --from alice

run_tx "Granting PermSubstationConnection on substation to Guild Leader C" \
    tx structs permission-grant-on-object "${SUBSTATION_ID}" "${GUILD_LEADER_C_ID}" 1024 --from alice

# Each founder pays from their own vault. All guilds may choose the same reactor
# without owning it, waiting for an age threshold, or consuming an entitlement.
create_paid_guild guild_leader_b "${GUILD_LEADER_B_ID}" "${REACTOR_ID}" "guild-b.energy" "${SUBSTATION_ID}"
GUILD_B_ID=$(query query structs player "${GUILD_LEADER_B_ID}" | jq -er '.Player.guildId | select(length > 0)')

create_paid_guild guild_leader_c "${GUILD_LEADER_C_ID}" "${REACTOR_ID}" "guild-c.energy" "${SUBSTATION_ID}"
GUILD_C_ID=$(query query structs player "${GUILD_LEADER_C_ID}" | jq -er '.Player.guildId | select(length > 0)')

# Existing guild owners still cannot abandon their guild by founding another.
run_tx_expect_reject_matching "Guild owner founding a second guild rejected" \
    "is the owner of guild" \
    tx structs guild-create "${REACTOR_ID}" "guild-c2.energy" "${SUBSTATION_ID}" --from guild_leader_c
assert_eq "Rejected founding preserves Guild C membership" "${GUILD_C_ID}" \
    "$(query query structs player "${GUILD_LEADER_C_ID}" | jq -r '.Player.guildId')"

# Retained wire fields must not revive proof or third-party founding authority.
run_tx_expect_reject_matching "Legacy charter proof cannot authorize founding" \
    "legacy_charter_fields_not_supported" \
    tx structs guild-create "${REACTOR_ID}" "retired-proof.energy" "" --proof retired --from player_2
run_tx_expect_reject_matching "Third-party founding cannot name a different owner" \
    "legacy_charter_fields_not_supported" \
    tx structs guild-create "${REACTOR_ID}" "retired-consent.energy" "" \
    --founder-player-id "${GUILD_LEADER_D_ID}" --from guild_leader_b
assert_eq "Refused legacy founding leaves player_2 guildless" "" \
    "$(query query structs player "${PLAYER_2_ID}" | jq -r '.Player.guildId // ""')"

# Guild D needs no entry substation and no permission on the preferred reactor.
GUILD_D_ENDPOINT="guild-d.energy"
create_paid_guild guild_leader_d "${GUILD_LEADER_D_ID}" "${REACTOR_ID}" "${GUILD_D_ENDPOINT}" ""
GUILD_D_ID=$(query query structs player "${GUILD_LEADER_D_ID}" | jq -er '.Player.guildId | select(length > 0)')
assert_eq "Guild D endpoint" "${GUILD_D_ENDPOINT}" \
    "$(query query structs guild "${GUILD_D_ID}" | jq -r '.Guild.endpoint')"
assert_eq "Guild D founding leaves Guild B unchanged" "${GUILD_B_ID}" \
    "$(query query structs player "${GUILD_LEADER_B_ID}" | jq -r '.Player.guildId')"

info "Guilds: A=${GUILD_ID}  B=${GUILD_B_ID}  C=${GUILD_C_ID}  D=${GUILD_D_ID}"

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

    # Resolve this transaction's allocation, including its intended controller.
    ALLOC_ID=$(get_created_grid_id allocation "${PID}" "${PLAYER_1_ID}")
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
ALLOCATION_TEST_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
P5_LIFECYCLE_CAPACITY=$(echo "${P5_JSON}" | python3 "${ALLOCATION_TEST_DIR}/player_allocatable_capacity.py" --lifecycle)
P5_CAP=$(echo "${P5_LIFECYCLE_CAPACITY}" | jq -er '.allocatableCapacity')
HALF_CAP=$(echo "${P5_LIFECYCLE_CAPACITY}" | jq -er '.half')
QUARTER_CAP=$(echo "${P5_LIFECYCLE_CAPACITY}" | jq -er '.quarter')
THREE_QUARTER_CAP=$(echo "${P5_LIFECYCLE_CAPACITY}" | jq -er '.threeQuarter')
info "Player 5 (${PLAYER_5_ID}) available primary capacity: ${P5_CAP}"

# Create with half capacity so we have room to test updates
run_tx "Creating allocation from Player 5 (${HALF_CAP} of ${P5_CAP})" \
    tx structs allocation-create "${PLAYER_5_ID}" "${HALF_CAP}" \
    --controller "${PLAYER_5_ID}" --allocation-type dynamic --from player_5

P5_ALLOC_ID=$(get_created_grid_id allocation "${PLAYER_5_ID}" "${PLAYER_5_ID}")
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

run_tx "Updating allocation ${P5_ALLOC_ID} power up to ${THREE_QUARTER_CAP} (own power released)" \
    tx structs allocation-update "${P5_ALLOC_ID}" "${THREE_QUARTER_CAP}" --from player_5

ALLOC_JSON=$(query query structs allocation "${P5_ALLOC_ID}")
ALLOC_POWER=$(jqr "${ALLOC_JSON}" '.gridAttributes.power' '0')
assert_eq "Allocation grew past prior free capacity" "${THREE_QUARTER_CAP}" "${ALLOC_POWER}"

# ─── allocation-transfer: transfer controller to alice ───
P5_TRANSFER_PERMS=$(get_permission_value_for_player "${P5_ALLOC_ID}" "${PLAYER_5_ID}")
ALICE_TRANSFER_PERMS=$(get_permission_value_for_player "${P5_ALLOC_ID}" "${PLAYER_1_ID}")
TRANSFER_STATE_BEFORE=$(query query structs allocation "${P5_ALLOC_ID}" | jq -cS '{allocation: (.Allocation | del(.controller)), power: .gridAttributes.power}')
assert_eq "Creator retains Update/Delete before transfer" "$(( PERM_UPDATE | PERM_DELETE ))" "$(( P5_TRANSFER_PERMS & (PERM_UPDATE | PERM_DELETE) ))"
info "Transferring allocation controller from player_5 to alice"
run_tx "Transferring allocation ${P5_ALLOC_ID} controller to alice" \
    tx structs allocation-transfer "${P5_ALLOC_ID}" "${PLAYER_1_ID}" --from player_5

ALLOC_JSON=$(query query structs allocation "${P5_ALLOC_ID}")
ALLOC_CTRL=$(jqr "${ALLOC_JSON}" '.Allocation.controller')
assert_eq "Allocation controller transferred" "${PLAYER_1_ID}" "${ALLOC_CTRL}"
assert_eq "Outgoing controller retains unrelated permissions" "$(( P5_TRANSFER_PERMS & ~PERM_ALLOCATION_CONNECTION ))" "$(get_permission_value_for_player "${P5_ALLOC_ID}" "${PLAYER_5_ID}")"
assert_eq "Incoming controller retains existing permissions" "$(( ALICE_TRANSFER_PERMS | PERM_ALLOCATION_CONNECTION ))" "$(get_permission_value_for_player "${P5_ALLOC_ID}" "${PLAYER_1_ID}")"
assert_eq "Transfer preserves allocation connection and power" "${TRANSFER_STATE_BEFORE}" "$(echo "${ALLOC_JSON}" | jq -cS '{allocation: (.Allocation | del(.controller)), power: .gridAttributes.power}')"

# Transfer back to player_5 so they can delete it
run_tx "Transferring allocation back to player_5" \
    tx structs allocation-transfer "${P5_ALLOC_ID}" "${PLAYER_5_ID}" --from alice

ALLOC_JSON=$(query query structs allocation "${P5_ALLOC_ID}")
assert_eq "Allocation controller returned" "${PLAYER_5_ID}" "$(jqr "${ALLOC_JSON}" '.Allocation.controller')"
assert_eq "Returning controller keeps original rights" "$(( P5_TRANSFER_PERMS | PERM_ALLOCATION_CONNECTION ))" "$(get_permission_value_for_player "${P5_ALLOC_ID}" "${PLAYER_5_ID}")"
assert_eq "Alice retains only her independent rights" "$(( ALICE_TRANSFER_PERMS & ~PERM_ALLOCATION_CONNECTION ))" "$(get_permission_value_for_player "${P5_ALLOC_ID}" "${PLAYER_1_ID}")"
assert_eq "Return transfer preserves allocation connection and power" "${TRANSFER_STATE_BEFORE}" "$(echo "${ALLOC_JSON}" | jq -cS '{allocation: (.Allocation | del(.controller)), power: .gridAttributes.power}')"

# ─── allocation-transfer to a player that does not exist ───
#
# The controller was unchecked: cc.GetPlayer allocates a cache without reading
# state, so the handler's error guard could never fire and a transfer to any
# string committed. The allocation ended up controlled by nobody, since the
# write is keyed by allocation id and so nothing panicked to refuse it.
run_tx_expect_reject_matching "Transferring allocation ${P5_ALLOC_ID} to a nonexistent player" \
    "player \(1-99999\) not found" \
    tx structs allocation-transfer "${P5_ALLOC_ID}" "1-99999" --from player_5

ALLOC_JSON=$(query query structs allocation "${P5_ALLOC_ID}")
ALLOC_CTRL=$(jqr "${ALLOC_JSON}" '.Allocation.controller')
assert_eq "Allocation controller survives a refused transfer" "${PLAYER_5_ID}" "${ALLOC_CTRL}"

# ─── allocation-delete ───
DELETE_ALLOC_JSON=$(query query structs allocation "${P5_ALLOC_ID}")
DELETE_ALLOC_POWER=$(echo "${DELETE_ALLOC_JSON}" | jq -er '.gridAttributes.power')
DELETE_SOURCE_LOAD=$(query query structs player "${PLAYER_5_ID}" | jq -er '.gridAttributes.load')
DELETE_EXPECTED_LOAD=$(python3 -c 'import sys; load, power = map(int, sys.argv[1:]); assert 0 < power <= load < 2**64; print(load - power)' "${DELETE_SOURCE_LOAD}" "${DELETE_ALLOC_POWER}")
run_tx "Deleting allocation ${P5_ALLOC_ID}" \
    tx structs allocation-delete "${P5_ALLOC_ID}" --from player_5

# Verify allocation is gone (query should return empty or error)
ALLOC_GONE_JSON=$(query query structs allocation "${P5_ALLOC_ID}" 2>/dev/null || echo '{}')
ALLOC_GONE_ID=$(jqr "${ALLOC_GONE_JSON}" '.Allocation.id' '')
assert_eq "Allocation deleted" "" "${ALLOC_GONE_ID}"
DELETE_SOURCE_AFTER=$(query query structs player "${PLAYER_5_ID}" | jq -er '.gridAttributes | if type == "object" then (.load // "0") else error("missing player grid") end')
assert_eq "Allocation deletion releases exactly its source load" "${DELETE_EXPECTED_LOAD}" "${DELETE_SOURCE_AFTER}"
assert_eq "Deleted allocation clears creator permissions" "0" "$(get_permission_value_for_player "${P5_ALLOC_ID}" "${PLAYER_5_ID}")"
assert_eq "Deleted allocation clears recipient permissions" "0" "$(get_permission_value_for_player "${P5_ALLOC_ID}" "${PLAYER_1_ID}")"

# ─── Query: allocation-all-by-destination ───
info "Querying allocations by destination (substation ${SUBSTATION_ID})"
ALLOC_BY_DEST=$(query query structs allocation-all-by-destination "${SUBSTATION_ID}" 2>/dev/null || echo '{}')
echo "  Allocations connected to substation: $(echo "${ALLOC_BY_DEST}" | jq '.Allocation | length' 2>/dev/null || echo '0')"

# ─── Re-create allocation for Player 5 for later phases ───
# Earlier transactions may have changed power; use current unallocated capacity.
P5_JSON=$(query query structs player "${PLAYER_5_ID}")
P5_AVAILABLE=$(echo "${P5_JSON}" | python3 "${ALLOCATION_TEST_DIR}/player_allocatable_capacity.py")
assert_eq "Player 5 has power for replacement allocation" "true" "$(echo "${P5_AVAILABLE}" | jq -r '.hasCapacity')"
P5_CAP=$(echo "${P5_AVAILABLE}" | jq -er '.allocatableCapacity')
run_tx "Re-creating allocation for Player 5" \
    tx structs allocation-create "${PLAYER_5_ID}" "${P5_CAP}" \
    --controller "${PLAYER_5_ID}" --allocation-type dynamic --from player_5

P5_ALLOC_ID=$(get_created_grid_id allocation "${PLAYER_5_ID}" "${PLAYER_5_ID}")
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

# Verify substation input, outgoing load and shares after the fixed 10% operating loss.
info "Checking substation power"
SUB_JSON=$(query query structs substation "${SUBSTATION_ID}")
SUB_LOAD=$(jqr "${SUB_JSON}" '.gridAttributes.load' '0')
SUB_CAP=$(jqr "${SUB_JSON}" '.gridAttributes.capacity' '0')
assert_gt "Substation capacity" 0 "${SUB_CAP}"
echo "  Substation incoming capacity: ${SUB_CAP}, outgoing allocation load: ${SUB_LOAD}"
SUB_EXPECTED_SHARE=$(echo "${SUB_JSON}" | python3 -c 'import json,sys; g=json.load(sys.stdin)["gridAttributes"]; print(max(int(g.get("capacity","0"))*9//10-int(g.get("load","0")),0)//max(1,int(g.get("connectionCount","0"))))')
assert_eq "Substation player share includes 10% operating loss" "${SUB_EXPECTED_SHARE}" "$(jqr "${SUB_JSON}" '.gridAttributes.connectionCapacity // "0"')"

# ─── Dump all players ───
info "All players state"
query query structs player-all | jq -r '.Player[] | "  \(.id) guild=\(.guildId) sub=\(.substationId) planet=\(.planetId)"' 2>/dev/null || true

fi # phase 4

if run_phase 450; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 4b: Guild Bank & Token Operations
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 4b: Guild Bank & Token Operations"

GUILD_VAULT=$(get_vault_address "${GUILD_ID}")
PLAYER_2_VAULT=$(get_vault_address "${PLAYER_2_ID}")
PLAYER_3_VAULT=$(get_vault_address "${PLAYER_3_ID}")
fund_vault_alpha "${GUILD_ID}" 1000000
fund_vault_alpha "${PLAYER_2_ID}" 200000

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
info "Guild treasury ualpha balance before mint"
TREASURY_ALPHA_BEFORE=$(get_balance "${GUILD_VAULT}" ualpha)
echo "  Treasury ualpha: ${TREASURY_ALPHA_BEFORE}"

TREASURY_TOKEN_BEFORE=$(get_balance "${GUILD_VAULT}" "${GUILD_TOKEN_DENOM}")
echo "  Treasury ${GUILD_TOKEN_DENOM}: ${TREASURY_TOKEN_BEFORE}"

COLLATERAL_ALPHA_BEFORE=$(get_balance "${COLLATERAL_ADDR}" ualpha)
echo "  Collateral ualpha: ${COLLATERAL_ALPHA_BEFORE}"

# ─── Mint guild tokens ───
# Alice authorizes the guild vault to deposit 1000000 ualpha as collateral and mints 500000 guild tokens
MINT_ALPHA=1000000
MINT_TOKENS=500000
SUPPLY_BEFORE_MINT=$(guild_supply "${GUILD_TOKEN_DENOM}")
run_tx "Minting ${MINT_TOKENS} guild tokens with ${MINT_ALPHA} ualpha collateral" \
    tx structs guild-bank-mint "${MINT_ALPHA}" "${MINT_TOKENS}" --from alice

# Verify: newly minted tokens belong to the guild treasury
TREASURY_TOKEN_AFTER_MINT=$(get_balance "${GUILD_VAULT}" "${GUILD_TOKEN_DENOM}")
info "Guild treasury tokens after mint: ${TREASURY_TOKEN_AFTER_MINT}"
assert_eq "Guild treasury receives exactly the minted tokens" "$((TREASURY_TOKEN_BEFORE + MINT_TOKENS))" "${TREASURY_TOKEN_AFTER_MINT}"
assert_eq "Guild vault funds the mint collateral" "$((TREASURY_ALPHA_BEFORE - MINT_ALPHA))" "$(get_balance "${GUILD_VAULT}" ualpha)"

# Verify: collateral address should hold the Alpha
COLLATERAL_ALPHA_AFTER_MINT=$(get_balance "${COLLATERAL_ADDR}" ualpha)
info "Collateral ualpha after mint: ${COLLATERAL_ALPHA_AFTER_MINT}"
assert_eq "Collateral Alpha increased by the mint deposit" "$((COLLATERAL_ALPHA_BEFORE + MINT_ALPHA))" "${COLLATERAL_ALPHA_AFTER_MINT}"

# ─── Transfer guild tokens to Player 2 via vault-send ───
TRANSFER_TO_P2=200000
P2_TOKEN_BEFORE_TRANSFER=$(get_balance "${PLAYER_2_VAULT}" "${GUILD_TOKEN_DENOM}")
run_tx "Transferring ${TRANSFER_TO_P2}${GUILD_TOKEN_DENOM} to Player 2 via vault-send" \
    tx structs vault-send "${GUILD_ID}" "${PLAYER_2_ID}" "${TRANSFER_TO_P2}${GUILD_TOKEN_DENOM}" --from alice

P2_TOKEN_AFTER_TRANSFER=$(get_balance "${PLAYER_2_VAULT}" "${GUILD_TOKEN_DENOM}")
info "Player 2 guild tokens after transfer: ${P2_TOKEN_AFTER_TRANSFER}"
assert_eq "Player 2 received guild tokens" "$((P2_TOKEN_BEFORE_TRANSFER + TRANSFER_TO_P2))" "${P2_TOKEN_AFTER_TRANSFER}"

# ─── Transfer guild tokens to Player 3 (for confiscation test later) ───
TRANSFER_TO_P3=100000
P3_TOKEN_BEFORE_TRANSFER=$(get_balance "${PLAYER_3_VAULT}" "${GUILD_TOKEN_DENOM}")
run_tx "Transferring ${TRANSFER_TO_P3}${GUILD_TOKEN_DENOM} to Player 3 via vault-send" \
    tx structs vault-send "${GUILD_ID}" "${PLAYER_3_ID}" "${TRANSFER_TO_P3}${GUILD_TOKEN_DENOM}" --from alice

P3_TOKEN_AFTER_TRANSFER=$(get_balance "${PLAYER_3_VAULT}" "${GUILD_TOKEN_DENOM}")
info "Player 3 guild tokens after transfer: ${P3_TOKEN_AFTER_TRANSFER}"
assert_eq "Player 3 received guild tokens" "$((P3_TOKEN_BEFORE_TRANSFER + TRANSFER_TO_P3))" "${P3_TOKEN_AFTER_TRANSFER}"

# Verify treasury tokens, including the founding token, less distributions
TREASURY_TOKEN_AFTER_TRANSFERS=$(get_balance "${GUILD_VAULT}" "${GUILD_TOKEN_DENOM}")
EXPECTED_TREASURY_REMAINING=$((TREASURY_TOKEN_AFTER_MINT - TRANSFER_TO_P2 - TRANSFER_TO_P3))
info "Guild treasury tokens after transfers: ${TREASURY_TOKEN_AFTER_TRANSFERS} (expected ${EXPECTED_TREASURY_REMAINING})"
assert_eq "Guild treasury token balance after transfers" "${EXPECTED_TREASURY_REMAINING}" "${TREASURY_TOKEN_AFTER_TRANSFERS}"

# ─── Redeem guild tokens (Player 2 redeems some for Alpha) ───
REDEEM_AMOUNT=50000
P2_ALPHA_BEFORE_REDEEM=$(get_balance "${PLAYER_2_VAULT}" ualpha)
SUPPLY_BEFORE_REDEEM=$(guild_supply "${GUILD_TOKEN_DENOM}")
EXPECTED_REDEEM_ALPHA=$((REDEEM_AMOUNT * COLLATERAL_ALPHA_AFTER_MINT / SUPPLY_BEFORE_REDEEM))
info "Player 2 ualpha before redeem: ${P2_ALPHA_BEFORE_REDEEM}"

run_tx "Player 2 redeeming ${REDEEM_AMOUNT}${GUILD_TOKEN_DENOM} for Alpha" \
    tx structs guild-bank-redeem "${REDEEM_AMOUNT}${GUILD_TOKEN_DENOM}" 1 --from player_2

# Verify: Player 2 token balance decreased
P2_TOKEN_AFTER_REDEEM=$(get_balance "${PLAYER_2_VAULT}" "${GUILD_TOKEN_DENOM}")
EXPECTED_P2_TOKENS=$((P2_TOKEN_AFTER_TRANSFER - REDEEM_AMOUNT))
info "Player 2 guild tokens after redeem: ${P2_TOKEN_AFTER_REDEEM} (expected ${EXPECTED_P2_TOKENS})"
assert_eq "Player 2 tokens decreased after redeem" "${EXPECTED_P2_TOKENS}" "${P2_TOKEN_AFTER_REDEEM}"

# No convert-out fee has been configured yet. The founding token and collateral
# are included in the queried supply and pool; a positive credit alone is not proof.
P2_ALPHA_AFTER_REDEEM=$(get_balance "${PLAYER_2_VAULT}" ualpha)
info "Player 2 ualpha after redeem: ${P2_ALPHA_AFTER_REDEEM}"
assert_eq "Player 2 vault receives exact initial redemption" "$((P2_ALPHA_BEFORE_REDEEM + EXPECTED_REDEEM_ALPHA))" "${P2_ALPHA_AFTER_REDEEM}"

# Verify: Collateral pool decreased
COLLATERAL_ALPHA_AFTER_REDEEM=$(get_balance "${COLLATERAL_ADDR}" ualpha)
info "Collateral ualpha after redeem: ${COLLATERAL_ALPHA_AFTER_REDEEM}"
assert_eq "Collateral funds exact initial redemption" "$((COLLATERAL_ALPHA_AFTER_MINT - EXPECTED_REDEEM_ALPHA))" "${COLLATERAL_ALPHA_AFTER_REDEEM}"
assert_eq "Initial redemption burns exact token supply" "$((SUPPLY_BEFORE_REDEEM - REDEEM_AMOUNT))" "$(guild_supply "${GUILD_TOKEN_DENOM}")"

# ─── Confiscate and burn (Alice confiscates tokens from Player 3) ───
CONFISCATE_AMOUNT=50000
P3_TOKEN_BEFORE_CONFISCATE=$(get_balance "${PLAYER_3_VAULT}" "${GUILD_TOKEN_DENOM}")
P3_ALPHA_BEFORE_CONFISCATE=$(get_balance "${PLAYER_3_VAULT}" ualpha)
P3_KEY_TOKEN_BEFORE_CONFISCATE=$(get_balance "${PLAYER_3_ADDRESS}" "${GUILD_TOKEN_DENOM}")
P3_KEY_ALPHA_BEFORE_CONFISCATE=$(get_balance "${PLAYER_3_ADDRESS}" ualpha)
GUILD_TOKEN_BEFORE_CONFISCATE=$(get_balance "${GUILD_VAULT}" "${GUILD_TOKEN_DENOM}")
COLLATERAL_BEFORE_CONFISCATE=$(get_balance "${COLLATERAL_ADDR}" ualpha)
SUPPLY_BEFORE_CONFISCATE=$(guild_supply "${GUILD_TOKEN_DENOM}")
info "Player 3 guild tokens before confiscate: ${P3_TOKEN_BEFORE_CONFISCATE}"

run_tx "Alice confiscating ${CONFISCATE_AMOUNT} guild tokens from Player 3" \
    tx structs guild-bank-confiscate-and-burn "${CONFISCATE_AMOUNT}" "${PLAYER_3_VAULT}" --from alice

# Verify: Player 3 token balance decreased
P3_TOKEN_AFTER_CONFISCATE=$(get_balance "${PLAYER_3_VAULT}" "${GUILD_TOKEN_DENOM}")
EXPECTED_P3_TOKENS=$((P3_TOKEN_BEFORE_CONFISCATE - CONFISCATE_AMOUNT))
info "Player 3 guild tokens after confiscate: ${P3_TOKEN_AFTER_CONFISCATE} (expected ${EXPECTED_P3_TOKENS})"
assert_eq "Player 3 tokens decreased after confiscate" "${EXPECTED_P3_TOKENS}" "${P3_TOKEN_AFTER_CONFISCATE}"
assert_eq "Confiscation burns exact supply" "$((SUPPLY_BEFORE_CONFISCATE - CONFISCATE_AMOUNT))" "$(guild_supply "${GUILD_TOKEN_DENOM}")"
assert_eq "Confiscation does not redeem collateral" "${COLLATERAL_BEFORE_CONFISCATE}" "$(get_balance "${COLLATERAL_ADDR}" ualpha)"
assert_eq "Confiscation leaves holder vault Alpha unchanged" "${P3_ALPHA_BEFORE_CONFISCATE}" "$(get_balance "${PLAYER_3_VAULT}" ualpha)"
assert_eq "Confiscation leaves holder key tokens unchanged" "${P3_KEY_TOKEN_BEFORE_CONFISCATE}" "$(get_balance "${PLAYER_3_ADDRESS}" "${GUILD_TOKEN_DENOM}")"
assert_eq "Confiscation leaves holder key Alpha unchanged" "${P3_KEY_ALPHA_BEFORE_CONFISCATE}" "$(get_balance "${PLAYER_3_ADDRESS}" ualpha)"
assert_eq "Confiscation does not credit treasury tokens" "${GUILD_TOKEN_BEFORE_CONFISCATE}" "$(get_balance "${GUILD_VAULT}" "${GUILD_TOKEN_DENOM}")"

# ─── Verify total supply decreased (tokens were burned, not moved) ───
TOTAL_SUPPLY=$(guild_supply "${GUILD_TOKEN_DENOM}")
info "Total guild token supply: ${TOTAL_SUPPLY}"
EXPECTED_SUPPLY=$((SUPPLY_BEFORE_MINT + MINT_TOKENS - REDEEM_AMOUNT - CONFISCATE_AMOUNT))
info "Expected supply (prior ${SUPPLY_BEFORE_MINT} + minted ${MINT_TOKENS} - redeemed ${REDEEM_AMOUNT} - burned ${CONFISCATE_AMOUNT} = ${EXPECTED_SUPPLY})"
assert_eq "Total guild token supply after redeem+burn" "${EXPECTED_SUPPLY}" "${TOTAL_SUPPLY}"

# ─── Attempt unauthorized mint (Player 2 should fail) ───
info "Testing unauthorized mint (Player 2, non-admin)"
P2_TOKEN_BEFORE_BAD_MINT=$(get_balance "${PLAYER_2_VAULT}" "${GUILD_TOKEN_DENOM}")
P2_ALPHA_BEFORE_BAD_MINT=$(get_balance "${PLAYER_2_VAULT}" ualpha)
GUILD_TOKEN_BEFORE_BAD_MINT=$(get_balance "${GUILD_VAULT}" "${GUILD_TOKEN_DENOM}")
GUILD_ALPHA_BEFORE_BAD_MINT=$(get_balance "${GUILD_VAULT}" ualpha)
COLLATERAL_BEFORE_BAD_MINT=$(get_balance "${COLLATERAL_ADDR}" ualpha)
SUPPLY_BEFORE_BAD_MINT=$(guild_supply "${GUILD_TOKEN_DENOM}")

run_tx_expect_permission_denied "Non-owner Player 2 cannot mint treasury tokens" \
    tx structs guild-bank-mint 100 100 --from player_2

P2_TOKEN_AFTER_BAD_MINT=$(get_balance "${PLAYER_2_VAULT}" "${GUILD_TOKEN_DENOM}")
assert_eq "Unauthorized mint did not change Player 2 balance" "${P2_TOKEN_BEFORE_BAD_MINT}" "${P2_TOKEN_AFTER_BAD_MINT}"
assert_eq "Unauthorized mint preserves player vault Alpha" "${P2_ALPHA_BEFORE_BAD_MINT}" "$(get_balance "${PLAYER_2_VAULT}" ualpha)"
assert_eq "Unauthorized mint preserves treasury tokens" "${GUILD_TOKEN_BEFORE_BAD_MINT}" "$(get_balance "${GUILD_VAULT}" "${GUILD_TOKEN_DENOM}")"
assert_eq "Unauthorized mint preserves treasury Alpha" "${GUILD_ALPHA_BEFORE_BAD_MINT}" "$(get_balance "${GUILD_VAULT}" ualpha)"
assert_eq "Unauthorized mint preserves collateral" "${COLLATERAL_BEFORE_BAD_MINT}" "$(get_balance "${COLLATERAL_ADDR}" ualpha)"
assert_eq "Unauthorized mint preserves supply" "${SUPPLY_BEFORE_BAD_MINT}" "$(guild_supply "${GUILD_TOKEN_DENOM}")"


# ═════════════════════════════════════════════════════════════════════════════
#  Convert-in, convert fees, and cross-guild convert (v0.21.0)
# ═════════════════════════════════════════════════════════════════════════════
info "--- Guild Bank Convert & Fees (v0.21.0) ---"

# guild_supply <denom>: total supply of a guild token denom

# ─── Set a 10% convert-in fee and verify it is stored ───
run_tx "Setting guild convert-in fee to 0.1" \
    tx structs guild-update-bank-convert-in-fee "${GUILD_ID}" "0.1" --from alice

GUILD_JSON=$(query query structs guild "${GUILD_ID}")
assert_eq "Guild convert-in fee stored" "0.100000000000000000" "$(jqr "${GUILD_JSON}" '.Guild.bankConvertInFee' '0')"

# ─── Player 2 converts 100000 ualpha into guild tokens at the current ratio ───
# feeAlpha = ceil(0.1 * 100000) = 10000; netAlpha = 90000;
# tokensOut = floor(90000 * supply / collateral); full 100000 -> collateral.
CONVERT_ALPHA=100000
CONVERT_FEE=10000
CONVERT_NET=$((CONVERT_ALPHA - CONVERT_FEE))

SUPPLY_BEFORE_CONVERT=$(guild_supply "${GUILD_TOKEN_DENOM}")
COLLATERAL_BEFORE_CONVERT=$(get_balance "${COLLATERAL_ADDR}" ualpha)
P2_TOKEN_BEFORE_CONVERT=$(get_balance "${PLAYER_2_VAULT}" "${GUILD_TOKEN_DENOM}")
P2_ALPHA_BEFORE_CONVERT=$(get_balance "${PLAYER_2_VAULT}" ualpha)
EXPECTED_CONVERT_TOKENS=$((CONVERT_NET * SUPPLY_BEFORE_CONVERT / COLLATERAL_BEFORE_CONVERT))
info "Convert quote: ${CONVERT_ALPHA} ualpha (fee ${CONVERT_FEE}) -> ${EXPECTED_CONVERT_TOKENS} tokens (supply=${SUPPLY_BEFORE_CONVERT}, collateral=${COLLATERAL_BEFORE_CONVERT})"

run_tx "Player 2 converts ${CONVERT_ALPHA} ualpha into guild tokens" \
    tx structs guild-bank-convert "${GUILD_ID}" "${CONVERT_ALPHA}" 1 --from player_2

P2_TOKEN_AFTER_CONVERT=$(get_balance "${PLAYER_2_VAULT}" "${GUILD_TOKEN_DENOM}")
assert_eq "Player 2 received converted tokens" "$((P2_TOKEN_BEFORE_CONVERT + EXPECTED_CONVERT_TOKENS))" "${P2_TOKEN_AFTER_CONVERT}"

COLLATERAL_AFTER_CONVERT=$(get_balance "${COLLATERAL_ADDR}" ualpha)
assert_eq "Full convert alpha (incl fee) entered collateral" "$((COLLATERAL_BEFORE_CONVERT + CONVERT_ALPHA))" "${COLLATERAL_AFTER_CONVERT}"
assert_eq "Conversion debits exact player vault Alpha" "$((P2_ALPHA_BEFORE_CONVERT - CONVERT_ALPHA))" "$(get_balance "${PLAYER_2_VAULT}" ualpha)"
assert_eq "Conversion mints exact guild token supply" "$((SUPPLY_BEFORE_CONVERT + EXPECTED_CONVERT_TOKENS))" "$(guild_supply "${GUILD_TOKEN_DENOM}")"

# ─── Convert-in slippage guard: demand an impossibly high min output ───
info "Testing convert-in slippage guard (min-amount-token too high)"
P2_TOKEN_BEFORE_GUARD=$(get_balance "${PLAYER_2_VAULT}" "${GUILD_TOKEN_DENOM}")
P2_ALPHA_BEFORE_GUARD=$(get_balance "${PLAYER_2_VAULT}" ualpha)
COLLATERAL_BEFORE_GUARD=$(get_balance "${COLLATERAL_ADDR}" ualpha)
SUPPLY_BEFORE_GUARD=$(guild_supply "${GUILD_TOKEN_DENOM}")
run_tx_expect_reject_matching "Player 2 convert with min-amount-token 999999999 (should fail)" \
    "slippage" \
    tx structs guild-bank-convert "${GUILD_ID}" "${CONVERT_ALPHA}" 999999999 --from player_2
assert_eq "Slippage-guarded convert left Player 2 tokens unchanged" "${P2_TOKEN_BEFORE_GUARD}" "$(get_balance "${PLAYER_2_VAULT}" "${GUILD_TOKEN_DENOM}")"
assert_eq "Slippage-guarded convert preserves player vault Alpha" "${P2_ALPHA_BEFORE_GUARD}" "$(get_balance "${PLAYER_2_VAULT}" ualpha)"
assert_eq "Slippage-guarded convert preserves collateral" "${COLLATERAL_BEFORE_GUARD}" "$(get_balance "${COLLATERAL_ADDR}" ualpha)"
assert_eq "Slippage-guarded convert preserves supply" "${SUPPLY_BEFORE_GUARD}" "$(guild_supply "${GUILD_TOKEN_DENOM}")"

# ─── Set a 10% convert-out fee and redeem: fee stays in collateral ───
run_tx "Setting guild convert-out fee to 0.1" \
    tx structs guild-update-bank-convert-out-fee "${GUILD_ID}" "0.1" --from alice

GUILD_JSON=$(query query structs guild "${GUILD_ID}")
assert_eq "Guild convert-out fee stored" "0.100000000000000000" "$(jqr "${GUILD_JSON}" '.Guild.bankConvertOutFee' '0')"

FEE_REDEEM_AMOUNT=40000
SUPPLY_BEFORE_FEE_REDEEM=$(guild_supply "${GUILD_TOKEN_DENOM}")
COLLATERAL_BEFORE_FEE_REDEEM=$(get_balance "${COLLATERAL_ADDR}" ualpha)
P2_ALPHA_BEFORE_FEE_REDEEM=$(get_balance "${PLAYER_2_VAULT}" ualpha)
P2_TOKEN_BEFORE_FEE_REDEEM=$(get_balance "${PLAYER_2_VAULT}" "${GUILD_TOKEN_DENOM}")
# grossAlpha = floor(amount * collateral / supply); feeAlpha = ceil(0.1 * gross); net = gross - fee.
GROSS_ALPHA=$((FEE_REDEEM_AMOUNT * COLLATERAL_BEFORE_FEE_REDEEM / SUPPLY_BEFORE_FEE_REDEEM))
OUT_FEE=$(( (GROSS_ALPHA + 9) / 10 ))   # ceil(gross/10)
NET_ALPHA=$((GROSS_ALPHA - OUT_FEE))
info "Redeem-with-fee quote: ${FEE_REDEEM_AMOUNT} tokens -> gross ${GROSS_ALPHA}, fee ${OUT_FEE}, net ${NET_ALPHA}"

run_tx "Player 2 redeems ${FEE_REDEEM_AMOUNT}${GUILD_TOKEN_DENOM} with 10% out-fee" \
    tx structs guild-bank-redeem "${FEE_REDEEM_AMOUNT}${GUILD_TOKEN_DENOM}" 1 --from player_2

P2_ALPHA_AFTER_FEE_REDEEM=$(get_balance "${PLAYER_2_VAULT}" ualpha)
# Fees are paid by the signing key, so the vault credit must equal the quote.
P2_ALPHA_DELTA=$((P2_ALPHA_AFTER_FEE_REDEEM - P2_ALPHA_BEFORE_FEE_REDEEM))
assert_eq "Player 2 vault receives exact net redemption" "${NET_ALPHA}" "${P2_ALPHA_DELTA}"
COLLATERAL_AFTER_FEE_REDEEM=$(get_balance "${COLLATERAL_ADDR}" ualpha)
assert_eq "Collateral retained out-fee (dropped only net payout)" "$((COLLATERAL_BEFORE_FEE_REDEEM - NET_ALPHA))" "${COLLATERAL_AFTER_FEE_REDEEM}"
assert_eq "Fee redemption debits exact player tokens" "$((P2_TOKEN_BEFORE_FEE_REDEEM - FEE_REDEEM_AMOUNT))" "$(get_balance "${PLAYER_2_VAULT}" "${GUILD_TOKEN_DENOM}")"
assert_eq "Fee redemption burns exact token supply" "$((SUPPLY_BEFORE_FEE_REDEEM - FEE_REDEEM_AMOUNT))" "$(guild_supply "${GUILD_TOKEN_DENOM}")"

# ─── Cross-guild convert: Guild A token -> Guild B token in one tx ───
if [ -n "${GUILD_B_ID:-}" ]; then
    info "--- Cross-guild convert (A -> B) ---"
    GUILD_B_TOKEN_DENOM="uguild.${GUILD_B_ID}"

    # Add collateral and tokens to the bank already initialized by founding.
    fund_vault_alpha "${GUILD_B_ID}" 500000
    run_tx "Guild Leader B bootstraps Guild B bank (500000 ualpha -> 500000 token)" \
        tx structs guild-bank-mint 500000 500000 --from guild_leader_b

    B_COLLATERAL_JSON=$(query query structs guild-bank-collateral-address "${GUILD_B_ID}")
    B_COLLATERAL_ADDR=$(jqr "${B_COLLATERAL_JSON}" '.internalAddressAssociation[0].address')
    assert_not_empty "Guild B collateral address" "${B_COLLATERAL_ADDR}"

    # Guild B charges a 10% convert-in fee; Guild A already charges 10% out.
    run_tx "Setting Guild B convert-in fee to 0.1" \
        tx structs guild-update-bank-convert-in-fee "${GUILD_B_ID}" "0.1" --from guild_leader_b

    XCONVERT_TOKENS=20000
    A_SUPPLY_X=$(guild_supply "${GUILD_TOKEN_DENOM}")
    A_COLL_X=$(get_balance "${COLLATERAL_ADDR}" ualpha)
    B_SUPPLY_X=$(guild_supply "${GUILD_B_TOKEN_DENOM}")
    B_COLL_X=$(get_balance "${B_COLLATERAL_ADDR}" ualpha)

    # Leg 1 (redeem A): gross = floor(tokens * A_coll / A_supply); A out-fee = ceil(gross/10); bridge = gross - fee.
    X_GROSS=$((XCONVERT_TOKENS * A_COLL_X / A_SUPPLY_X))
    X_A_FEE=$(( (X_GROSS + 9) / 10 ))
    X_BRIDGE=$((X_GROSS - X_A_FEE))
    # Leg 2 (convert B): B in-fee = ceil(bridge/10); net = bridge - fee; out = floor(net * B_supply / B_coll).
    X_B_FEE=$(( (X_BRIDGE + 9) / 10 ))
    X_B_NET=$((X_BRIDGE - X_B_FEE))
    X_OUT=$((X_B_NET * B_SUPPLY_X / B_COLL_X))
    info "Cross quote: ${XCONVERT_TOKENS} A-tok -> gross ${X_GROSS} (A fee ${X_A_FEE}) -> bridge ${X_BRIDGE} -> (B fee ${X_B_FEE}) -> ${X_OUT} B-tok"

    P2_B_TOKEN_BEFORE_X=$(get_balance "${PLAYER_2_VAULT}" "${GUILD_B_TOKEN_DENOM}")
    P2_A_TOKEN_BEFORE_X=$(get_balance "${PLAYER_2_VAULT}" "${GUILD_TOKEN_DENOM}")
    P2_ALPHA_BEFORE_X=$(get_balance "${PLAYER_2_VAULT}" ualpha)
    run_tx "Player 2 converts ${XCONVERT_TOKENS}${GUILD_TOKEN_DENOM} into Guild B tokens" \
        tx structs guild-bank-convert-token "${XCONVERT_TOKENS}${GUILD_TOKEN_DENOM}" "${GUILD_B_ID}" 1 --from player_2

    P2_B_TOKEN_AFTER_X=$(get_balance "${PLAYER_2_VAULT}" "${GUILD_B_TOKEN_DENOM}")
    assert_eq "Player 2 received cross-converted Guild B tokens" "$((P2_B_TOKEN_BEFORE_X + X_OUT))" "${P2_B_TOKEN_AFTER_X}"
    assert_eq "Cross-conversion debits exact Guild A tokens" "$((P2_A_TOKEN_BEFORE_X - XCONVERT_TOKENS))" "$(get_balance "${PLAYER_2_VAULT}" "${GUILD_TOKEN_DENOM}")"
    assert_eq "Cross-conversion keeps bridge Alpha out of player vault" "${P2_ALPHA_BEFORE_X}" "$(get_balance "${PLAYER_2_VAULT}" ualpha)"
    assert_eq "Cross-conversion burns exact Guild A supply" "$((A_SUPPLY_X - XCONVERT_TOKENS))" "$(guild_supply "${GUILD_TOKEN_DENOM}")"
    assert_eq "Cross-conversion mints exact Guild B supply" "$((B_SUPPLY_X + X_OUT))" "$(guild_supply "${GUILD_B_TOKEN_DENOM}")"

    # Guild A collateral drops by the net bridge alpha (gross out, fee retained).
    assert_eq "Guild A collateral dropped by bridge alpha (fee retained)" "$((A_COLL_X - X_BRIDGE))" "$(get_balance "${COLLATERAL_ADDR}" ualpha)"
    # Guild B collateral gains the full bridge alpha (B in-fee stays in B's pool).
    assert_eq "Guild B collateral gained full bridge alpha" "$((B_COLL_X + X_BRIDGE))" "$(get_balance "${B_COLLATERAL_ADDR}" ualpha)"

    # Same-guild convert-token is rejected.
    info "Testing same-guild convert-token rejection"
    P2_A_TOKEN_BEFORE_SAME=$(get_balance "${PLAYER_2_VAULT}" "${GUILD_TOKEN_DENOM}")
    run_tx_expect_reject_matching "Player 2 same-guild convert-token A->A (should fail)" \
        "same_guild" \
        tx structs guild-bank-convert-token "1000${GUILD_TOKEN_DENOM}" "${GUILD_ID}" 1 --from player_2
    assert_eq "Same-guild convert left Player 2 A-token balance unchanged" "${P2_A_TOKEN_BEFORE_SAME}" "$(get_balance "${PLAYER_2_VAULT}" "${GUILD_TOKEN_DENOM}")"
else
    assert_not_empty "Guild B is required for cross-guild conversion coverage" "${GUILD_B_ID:-}"
fi

# ─── Reset Guild A convert fees to 0 so later phases are unaffected ───
run_tx "Resetting Guild A convert-in fee to 0" \
    tx structs guild-update-bank-convert-in-fee "${GUILD_ID}" "0" --from alice
run_tx "Resetting Guild A convert-out fee to 0" \
    tx structs guild-update-bank-convert-out-fee "${GUILD_ID}" "0" --from alice

# ─── Rank purchases: cumulative curve, quote guards and separate admission ───
# Entry 101 -> 100 costs 10; 100 -> 99 costs 15. Buying both at once costs 25.
run_tx "Configuring guild rank purchases with admission disabled" \
    tx structs guild-update-rank-purchase "${GUILD_ID}" 10 5 99 false --from alice
run_tx "Setting the member rank-purchase starting rank" \
    tx structs player-update-guild-rank "${PLAYER_2_ID}" 101 --from alice
run_tx "Funding member rank purchase in the guild's own token" \
    tx structs vault-send "${GUILD_ID}" "${PLAYER_2_ID}" "25${GUILD_TOKEN_DENOM}" --from alice
RANK_P2_BEFORE=$(get_balance "${PLAYER_2_VAULT}" "${GUILD_TOKEN_DENOM}")
RANK_GUILD_BEFORE=$(get_balance "${GUILD_VAULT}" "${GUILD_TOKEN_DENOM}")
RANK_SUPPLY_BEFORE=$(guild_supply "${GUILD_TOKEN_DENOM}")
run_tx_expect_reject_matching "Rank purchase rejects an insufficient quote ceiling" \
    "quoted_price_exceeded" \
    tx structs guild-rank-purchase "${GUILD_ID}" 100 9 101 --from player_2
assert_eq "Rejected quote moves no guild tokens" "${RANK_P2_BEFORE}" "$(get_balance "${PLAYER_2_VAULT}" "${GUILD_TOKEN_DENOM}")"
assert_eq "Rejected quote leaves treasury unchanged" "${RANK_GUILD_BEFORE}" "$(get_balance "${GUILD_VAULT}" "${GUILD_TOKEN_DENOM}")"
assert_eq "Rejected quote leaves supply unchanged" "${RANK_SUPPLY_BEFORE}" "$(guild_supply "${GUILD_TOKEN_DENOM}")"
assert_eq "Rejected quote preserves member rank" "101" "$(get_player_guild_rank "${PLAYER_2_ID}")"
run_tx "Purchasing first rank step" \
    tx structs guild-rank-purchase "${GUILD_ID}" 100 10 101 --from player_2
assert_rank_purchase_event "${GUILD_ID}" "${PLAYER_2_ID}" 101 100 10 false
assert_eq "First purchase rank" "100" "$(get_player_guild_rank "${PLAYER_2_ID}")"
assert_eq "First purchase debits 10 base tokens" "$((RANK_P2_BEFORE - 10))" "$(get_balance "${PLAYER_2_VAULT}" "${GUILD_TOKEN_DENOM}")"
run_tx_expect_reject_matching "Stale expected rank cannot buy again" \
    "rank_changed" \
    tx structs guild-rank-purchase "${GUILD_ID}" 99 25 101 --from player_2
assert_eq "Stale rank quote leaves player tokens unchanged" "$((RANK_P2_BEFORE - 10))" "$(get_balance "${PLAYER_2_VAULT}" "${GUILD_TOKEN_DENOM}")"
assert_eq "Stale rank quote leaves treasury unchanged" "$((RANK_GUILD_BEFORE + 10))" "$(get_balance "${GUILD_VAULT}" "${GUILD_TOKEN_DENOM}")"
assert_eq "Stale rank quote leaves supply unchanged" "${RANK_SUPPLY_BEFORE}" "$(guild_supply "${GUILD_TOKEN_DENOM}")"
assert_eq "Stale rank quote preserves purchased rank" "100" "$(get_player_guild_rank "${PLAYER_2_ID}")"
run_tx "Purchasing second rank step for the incremental price" \
    tx structs guild-rank-purchase "${GUILD_ID}" 99 15 100 --from player_2
assert_rank_purchase_event "${GUILD_ID}" "${PLAYER_2_ID}" 100 99 15 false
assert_eq "Second purchase rank" "99" "$(get_player_guild_rank "${PLAYER_2_ID}")"
assert_eq "Two upgrades charge only 25 base tokens in total" "$((RANK_P2_BEFORE - 25))" "$(get_balance "${PLAYER_2_VAULT}" "${GUILD_TOKEN_DENOM}")"
assert_eq "Rank payments reach the guild vault" "$((RANK_GUILD_BEFORE + 25))" "$(get_balance "${GUILD_VAULT}" "${GUILD_TOKEN_DENOM}")"
assert_eq "Rank purchases conserve guild token supply" "${RANK_SUPPLY_BEFORE}" "$(guild_supply "${GUILD_TOKEN_DENOM}")"

PLAYER_5_VAULT=$(get_vault_address "${PLAYER_5_ID}")
run_tx "Funding non-member purchase admission" \
    tx structs vault-send "${GUILD_ID}" "${PLAYER_5_ID}" "25${GUILD_TOKEN_DENOM}" --from alice
RANK_P5_BEFORE=$(get_balance "${PLAYER_5_VAULT}" "${GUILD_TOKEN_DENOM}")
RANK_GUILD_BEFORE_REJECT=$(get_balance "${GUILD_VAULT}" "${GUILD_TOKEN_DENOM}")
RANK_P5_BEFORE_REJECT=$(get_player_guild_rank "${PLAYER_5_ID}")
run_tx_expect_reject_matching "Normal rank purchase cannot admit a non-member" \
    "not a member of guild" \
    tx structs guild-rank-purchase "${GUILD_ID}" 99 25 101 --from player_5
assert_eq "Non-member purchase leaves player tokens unchanged" "${RANK_P5_BEFORE}" "$(get_balance "${PLAYER_5_VAULT}" "${GUILD_TOKEN_DENOM}")"
assert_eq "Non-member purchase leaves treasury unchanged" "${RANK_GUILD_BEFORE_REJECT}" "$(get_balance "${GUILD_VAULT}" "${GUILD_TOKEN_DENOM}")"
assert_eq "Non-member purchase leaves supply unchanged" "${RANK_SUPPLY_BEFORE}" "$(guild_supply "${GUILD_TOKEN_DENOM}")"
assert_eq "Non-member purchase preserves rank" "${RANK_P5_BEFORE_REJECT}" "$(get_player_guild_rank "${PLAYER_5_ID}")"
assert_eq "Non-member purchase leaves player outside guild" "" "$(query query structs player "${PLAYER_5_ID}" | jq -r '.Player.guildId // ""')"
run_tx_expect_reject_matching "Purchase admission requires its explicit guild setting" \
    "rank_purchase_admission_disabled" \
    tx structs guild-membership-join-by-rank-purchase "${GUILD_ID}" 99 25 --from player_5
assert_eq "Rejected admission moves no tokens" "${RANK_P5_BEFORE}" "$(get_balance "${PLAYER_5_VAULT}" "${GUILD_TOKEN_DENOM}")"
assert_eq "Rejected admission leaves treasury unchanged" "${RANK_GUILD_BEFORE_REJECT}" "$(get_balance "${GUILD_VAULT}" "${GUILD_TOKEN_DENOM}")"
assert_eq "Rejected admission leaves supply unchanged" "${RANK_SUPPLY_BEFORE}" "$(guild_supply "${GUILD_TOKEN_DENOM}")"
assert_eq "Rejected admission preserves rank" "${RANK_P5_BEFORE_REJECT}" "$(get_player_guild_rank "${PLAYER_5_ID}")"
assert_eq "Rejected admission leaves player outside guild" "" "$(query query structs player "${PLAYER_5_ID}" | jq -r '.Player.guildId // ""')"
run_tx "Enabling the separate purchase admission path" \
    tx structs guild-update-rank-purchase "${GUILD_ID}" 10 5 99 true --from alice
RANK_GUILD_BEFORE_JOIN=$(get_balance "${GUILD_VAULT}" "${GUILD_TOKEN_DENOM}")
RANK_P5_BEFORE_JOIN=$(get_player_guild_rank "${PLAYER_5_ID}")
run_tx "Player 5 joins by purchasing two rank steps" \
    tx structs guild-membership-join-by-rank-purchase "${GUILD_ID}" 99 25 --from player_5
assert_rank_purchase_event "${GUILD_ID}" "${PLAYER_5_ID}" "${RANK_P5_BEFORE_JOIN}" 99 25 true
assert_eq "Purchase admission joins guild" "${GUILD_ID}" "$(query query structs player "${PLAYER_5_ID}" | jq -r '.Player.guildId')"
assert_eq "Purchase admission sets bought rank" "99" "$(get_player_guild_rank "${PLAYER_5_ID}")"
assert_eq "Admission uses the same cumulative price" "$((RANK_P5_BEFORE - 25))" "$(get_balance "${PLAYER_5_VAULT}" "${GUILD_TOKEN_DENOM}")"
assert_eq "Admission payment reaches treasury" "$((RANK_GUILD_BEFORE_JOIN + 25))" "$(get_balance "${GUILD_VAULT}" "${GUILD_TOKEN_DENOM}")"
assert_eq "Admission conserves guild token supply" "${RANK_SUPPLY_BEFORE}" "$(guild_supply "${GUILD_TOKEN_DENOM}")"
run_tx "Disabling sales after purchase" \
    tx structs guild-update-rank-purchase "${GUILD_ID}" 0 0 0 false --from alice
assert_eq "Disabling sales preserves purchased member rank" "99" "$(get_player_guild_rank "${PLAYER_2_ID}")"
assert_eq "Disabling sales preserves purchased admission rank" "99" "$(get_player_guild_rank "${PLAYER_5_ID}")"
# Restore the membership fixtures for the later invite/request and rank suites.
run_tx "Returning Player 5 to the non-member fixture" \
    tx structs guild-membership-kick "${PLAYER_5_ID}" --from alice
run_tx "Restoring Player 2 entry rank for later tests" \
    tx structs player-update-guild-rank "${PLAYER_2_ID}" 101 --from alice

# ─── Summary of token state ───
info "Guild token summary:"
echo "  Denom: ${GUILD_TOKEN_DENOM}"
echo "  Total supply: $(guild_supply "${GUILD_TOKEN_DENOM}")"
echo "  Alice: $(get_balance "${GUILD_VAULT}" "${GUILD_TOKEN_DENOM}")"
echo "  Player 2: $(get_balance "${PLAYER_2_VAULT}" "${GUILD_TOKEN_DENOM}")"
echo "  Player 3: $(get_balance "${PLAYER_3_VAULT}" "${GUILD_TOKEN_DENOM}")"
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
run_tx_expect_permission_denied "Player 5 tries to kick Player 2 (should fail)" \
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

# Safety: revoke any lingering invite application from previous tests. Nothing on
# file is the ordinary case, and the handler says so rather than no-opping, so a
# rejection here is the cleanup succeeding at having nothing to do.
run_tx_tolerate_reject "Revoking any lingering invite (safety cleanup)" \
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

# Safety cleanup of any lingering applications (see test 11 — nothing on file is fine)
run_tx_tolerate_reject "Revoking any lingering application (safety)" \
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

# Safety cleanup (see test 11 — nothing on file is fine)
run_tx_tolerate_reject "Revoking any lingering application (safety)" \
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

# ─── Test 21: membership messages naming a player that does not exist ───────
#
# cc.GetPlayer allocates a cache without reading state, so the id a message
# carries was never resolved. An approver naming a phantom got a synthesized
# application and ApproveRequest mutated a zero-valued player; the transaction
# died only because the commit keyed on the empty player id and the KV store
# panics on an empty key. These must be refusals now, not panics.
info "--- Test 21: membership messages naming a nonexistent player ---"

run_tx_expect_reject_matching "Alice approves a request from a nonexistent player" \
    "player \(1-99999\) not found" \
    tx structs guild-membership-request-approve "1-99999" --from alice

run_tx_expect_reject_matching "Alice invites a nonexistent player" \
    "player \(1-99999\) not found" \
    tx structs guild-membership-invite "1-99999" --from alice

run_tx_expect_reject_matching "Alice kicks a nonexistent player" \
    "player \(1-99999\) not found" \
    tx structs guild-membership-kick "1-99999" --from alice

# The guild must be untouched by all three.
GUILD_JSON=$(query query structs guild "${GUILD_ID}")
assert_eq "Guild owner unchanged after phantom membership attempts" "${PLAYER_1_ID}" \
    "$(jqr "${GUILD_JSON}" '.Guild.owner' '')"

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
if temp_substation_create SECOND_SUB_ID "guild settings test"; then
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

    # This block used to end here, leaving the substation behind. The later
    # agreement test then picked the orphan up as its own and deleted it, which
    # is why that failure read as somebody else's bug.
    temp_substation_destroy "${SECOND_SUB_ID}" "guild settings test"
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

# ─── guild-update-bank-convert-in-fee / -out-fee (v0.21.0) ───
info "--- Guild Bank Convert Fee Settings ---"

run_tx "Setting guild convert-in fee to 0.05" \
    tx structs guild-update-bank-convert-in-fee "${GUILD_ID}" "0.05" --from alice
GUILD_JSON=$(query query structs guild "${GUILD_ID}")
assert_eq "Guild convert-in fee set to 0.05" "0.050000000000000000" "$(jqr "${GUILD_JSON}" '.Guild.bankConvertInFee' '0')"

run_tx "Setting guild convert-out fee to 0.25" \
    tx structs guild-update-bank-convert-out-fee "${GUILD_ID}" "0.25" --from alice
GUILD_JSON=$(query query structs guild "${GUILD_ID}")
assert_eq "Guild convert-out fee set to 0.25" "0.250000000000000000" "$(jqr "${GUILD_JSON}" '.Guild.bankConvertOutFee' '0')"

# Out-of-range fee (> 1.0) must be rejected and leave the stored rate unchanged.
info "Testing out-of-range fee rejection (1.5)"
run_tx_expect_reject_matching "Setting convert-in fee to 1.5 (should fail)" \
    "out_of_range" \
    tx structs guild-update-bank-convert-in-fee "${GUILD_ID}" "1.5" --from alice
GUILD_JSON=$(query query structs guild "${GUILD_ID}")
assert_eq "Out-of-range fee rejected (still 0.05)" "0.050000000000000000" "$(jqr "${GUILD_JSON}" '.Guild.bankConvertInFee' '0')"

# Non-admin (Player 3) cannot change bank fees.
info "Testing unauthorized bank fee update (Player 3)"
run_tx_expect_permission_denied "Player 3 tries to set convert-in fee (should fail)" \
    tx structs guild-update-bank-convert-in-fee "${GUILD_ID}" "0.9" --from player_3
GUILD_JSON=$(query query structs guild "${GUILD_ID}")
assert_eq "Unauthorized fee update did not change rate" "0.050000000000000000" "$(jqr "${GUILD_JSON}" '.Guild.bankConvertInFee' '0')"

# Reset fees to 0 so later phases (e.g. bank redeem/mint) are unaffected.
run_tx "Resetting convert-in fee to 0" \
    tx structs guild-update-bank-convert-in-fee "${GUILD_ID}" "0" --from alice
run_tx "Resetting convert-out fee to 0" \
    tx structs guild-update-bank-convert-out-fee "${GUILD_ID}" "0" --from alice

# ─── guild-update-owner-id: transfer ownership ───
# Seed an explicit treasury grant on the seller; the sale must clear it along
# with every other direct guild permission. The new owner can transfer by ownership.
run_tx "Grant seller explicit guild treasury permission before ownership transfer" \
    tx structs permission-grant-on-object "${GUILD_ID}" "${PLAYER_1_ID}" "${PERM_TOKEN_TRANSFER}" --from alice

info "Transferring guild ownership to Player 2"
run_tx "Transferring guild ownership to Player 2" \
    tx structs guild-update-owner-id "${GUILD_ID}" "${PLAYER_2_ID}" --from alice

GUILD_JSON=$(query query structs guild "${GUILD_ID}")
GUILD_OWNER=$(jqr "${GUILD_JSON}" '.Guild.owner')
assert_eq "Guild owner transferred to Player 2" "${PLAYER_2_ID}" "${GUILD_OWNER}"
assert_eq "Seller has no residual guild permissions" "0" "$(get_permission_value_for_player "${GUILD_ID}" "${PLAYER_1_ID}")"

# Player 2 transfers back to Player 1
run_tx "Player 2 transfers guild ownership back to Player 1" \
    tx structs guild-update-owner-id "${GUILD_ID}" "${PLAYER_1_ID}" --from player_2

GUILD_JSON=$(query query structs guild "${GUILD_ID}")
GUILD_OWNER=$(jqr "${GUILD_JSON}" '.Guild.owner')
assert_eq "Guild owner transferred back to Player 1" "${PLAYER_1_ID}" "${GUILD_OWNER}"
assert_eq "Returning seller has no residual guild permissions" "0" "$(get_permission_value_for_player "${GUILD_ID}" "${PLAYER_2_ID}")"

# ─── Negative: non-owner tries to update endpoint ───
info "Testing unauthorized guild update (Player 3 tries to update endpoint)"
run_tx_expect_permission_denied "Player 3 tries to update guild endpoint (should fail)" \
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
run_tx_expect_permission_denied "Player 3 tries to update entry rank (should fail)" \
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
run_tx_expect_reject_matching "Setting Player 2 rank to 0 (should fail — rank 0 is forbidden)" \
    "rank validation failed" \
    tx structs player-update-guild-rank "${PLAYER_2_ID}" 0 --from alice

P2_JSON=$(query query structs player "${PLAYER_2_ID}")
P2_RANK_AFTER_ZERO=$(jqr "${P2_JSON}" '.Player.guildRank' '0')
assert_eq "Player 2 rank unchanged after rank=0 attempt" "5" "${P2_RANK_AFTER_ZERO}"

# Negative: rank standing does not supply administrative authority.
run_tx_expect_permission_denied "Player 3 (rank 10) tries to update Player 2 rank (should fail)" \
    tx structs player-update-guild-rank "${PLAYER_2_ID}" 3 --from player_3

P2_JSON=$(query query structs player "${PLAYER_2_ID}")
P2_RANK_UNCHANGED=$(jqr "${P2_JSON}" '.Player.guildRank' '0')
assert_eq "Player 2 rank unchanged after unauthorized change" "5" "${P2_RANK_UNCHANGED}"

# A better rank still cannot promote another member without explicit Admin.
run_tx_expect_permission_denied "Player 2 rank alone cannot promote Player 3" \
    tx structs player-update-guild-rank "${PLAYER_3_ID}" 8 --from player_2

P3_JSON=$(query query structs player "${PLAYER_3_ID}")
P3_RANK_AFTER=$(jqr "${P3_JSON}" '.Player.guildRank' '0')
assert_eq "Player 3 rank unchanged after rank-only promotion attempt" "10" "${P3_RANK_AFTER}"

# Reset ranks for later phases
run_tx "Resetting Player 2 guild rank to 101" \
    tx structs player-update-guild-rank "${PLAYER_2_ID}" 101 --from alice
run_tx "Resetting Player 3 guild rank to 101" \
    tx structs player-update-guild-rank "${PLAYER_3_ID}" 101 --from alice

# ─── Owning a guild you are not a member of ──────────────────────────────────
# Guilds are property, and a sale moves the owner and the permission row without
# touching either player's guildId. So a buyer ends up administering a guild they
# are not in, which is why four handlers take an optional --guild-id instead of
# reading the signer's membership. Guild C is the subject because nothing else
# depends on it.
info "--- Guild Ownership Without Membership (Guild C) ---"

GUILD_C_JSON=$(query query structs guild "${GUILD_C_ID}")
assert_eq "Guild C starts owned by its founder" "${GUILD_LEADER_C_ID}" "$(jqr "${GUILD_C_JSON}" '.Guild.owner')"

GUILD_B_ENTRY_RANK_BEFORE=$(jqr "$(query query structs guild "${GUILD_B_ID}")" '.Guild.entryRank' '0')

run_tx "Guild Leader C sells Guild C to Guild Leader B" \
    tx structs guild-update-owner-id "${GUILD_C_ID}" "${GUILD_LEADER_B_ID}" --from guild_leader_c

GUILD_C_JSON=$(query query structs guild "${GUILD_C_ID}")
assert_eq "Guild C owner is now Guild Leader B" "${GUILD_LEADER_B_ID}" "$(jqr "${GUILD_C_JSON}" '.Guild.owner')"

# Buying a guild does not join it, and membership stays singular even though
# ownership does not: leader B holds two guilds and is a member of one.
LEADER_B_JSON=$(query query structs player "${GUILD_LEADER_B_ID}")
assert_eq "Buying a guild does not change the buyer's membership" "${GUILD_B_ID}" "$(jqr "${LEADER_B_JSON}" '.Player.guildId')"

# The seller kept membership and lost authority, which is what makes the sale a
# transfer rather than the addition of a second administrator.
run_tx_expect_fail "Seller can no longer set Guild C's entry rank" \
    tx structs guild-update-entry-rank 42 --from guild_leader_c

# ─── guild-update-entry-rank --guild-id ───
run_tx "Non-member owner sets Guild C entry rank via --guild-id" \
    tx structs guild-update-entry-rank 42 --guild-id "${GUILD_C_ID}" --from guild_leader_b

assert_eq "Guild C entry rank set by its non-member owner" "42" \
    "$(jqr "$(query query structs guild "${GUILD_C_ID}")" '.Guild.entryRank' '0')"
assert_eq "Guild B entry rank untouched by a named-guild update" "${GUILD_B_ENTRY_RANK_BEFORE}" \
    "$(jqr "$(query query structs guild "${GUILD_B_ID}")" '.Guild.entryRank' '0')"

# Omitting the flag still means the signer's own guild, which is what keeps
# existing clients working.
run_tx "Omitted --guild-id still falls back to the signer's guild" \
    tx structs guild-update-entry-rank 77 --from guild_leader_b

assert_eq "Membership fallback updated Guild B" "77" \
    "$(jqr "$(query query structs guild "${GUILD_B_ID}")" '.Guild.entryRank' '0')"
assert_eq "Membership fallback left Guild C alone" "42" \
    "$(jqr "$(query query structs guild "${GUILD_C_ID}")" '.Guild.entryRank' '0')"

# ─── player-update-guild-rank --guild-id ───
# The rule is that the target belongs to the named guild, which for a member
# caller is the same test as before. Leader C is still a member of what he sold.
run_tx "Non-member owner reranks a member of Guild C" \
    tx structs player-update-guild-rank "${GUILD_LEADER_C_ID}" 5 --guild-id "${GUILD_C_ID}" --from guild_leader_b

assert_eq "Guild C member reranked by the non-member owner" "5" \
    "$(jqr "$(query query structs player "${GUILD_LEADER_C_ID}")" '.Player.guildRank' '0')"

run_tx_expect_fail "Reranking a player outside the named guild rejected" \
    tx structs player-update-guild-rank "${PLAYER_3_ID}" 5 --guild-id "${GUILD_C_ID}" --from guild_leader_b

assert_eq "Player 3 rank untouched by an out-of-guild rerank" "101" \
    "$(jqr "$(query query structs player "${PLAYER_3_ID}")" '.Player.guildRank' '0')"

# ─── guild-bank-mint / guild-bank-confiscate-and-burn --guild-id ───
GUILD_C_DENOM="uguild.${GUILD_C_ID}"
GUILD_C_VAULT=$(get_vault_address "${GUILD_C_ID}")
fund_vault_alpha "${GUILD_C_ID}" 100
GUILD_C_TOKENS_BEFORE=$(get_balance "${GUILD_C_VAULT}" "${GUILD_C_DENOM}")
GUILD_C_COLLATERAL_ADDR=$(jqr "$(query query structs guild-bank-collateral-address "${GUILD_C_ID}")" '.internalAddressAssociation[0].address')
assert_not_empty "Guild C collateral address" "${GUILD_C_COLLATERAL_ADDR}"
LEADER_B_VAULT=$(get_vault_address "${GUILD_LEADER_B_ID}")
GUILD_B_VAULT=$(get_vault_address "${GUILD_B_ID}")
GUILD_C_ALPHA_BEFORE=$(get_balance "${GUILD_C_VAULT}" ualpha)
GUILD_C_COLLATERAL_BEFORE=$(get_balance "${GUILD_C_COLLATERAL_ADDR}" ualpha)
GUILD_C_SUPPLY_BEFORE=$(guild_supply "${GUILD_C_DENOM}")
LEADER_B_ALPHA_BEFORE_C_MINT=$(get_balance "${LEADER_B_VAULT}" ualpha)
GUILD_B_ALPHA_BEFORE_C_MINT=$(get_balance "${GUILD_B_VAULT}" ualpha)
run_tx "Non-member owner mints Guild C token" \
    tx structs guild-bank-mint 100 100 --guild-id "${GUILD_C_ID}" --from guild_leader_b

assert_eq "Non-member owner mints to Guild C vault" "$((GUILD_C_TOKENS_BEFORE + 100))" \
    "$(get_balance "${GUILD_C_VAULT}" "${GUILD_C_DENOM}")"
assert_eq "Named guild mint spends its own treasury Alpha" "$((GUILD_C_ALPHA_BEFORE - 100))" "$(get_balance "${GUILD_C_VAULT}" ualpha)"
assert_eq "Named guild mint funds its own collateral" "$((GUILD_C_COLLATERAL_BEFORE + 100))" "$(get_balance "${GUILD_C_COLLATERAL_ADDR}" ualpha)"
assert_eq "Named guild mint increases its own supply" "$((GUILD_C_SUPPLY_BEFORE + 100))" "$(guild_supply "${GUILD_C_DENOM}")"
assert_eq "Named guild mint preserves owner personal Alpha" "${LEADER_B_ALPHA_BEFORE_C_MINT}" "$(get_balance "${LEADER_B_VAULT}" ualpha)"
assert_eq "Named guild mint preserves membership guild Alpha" "${GUILD_B_ALPHA_BEFORE_C_MINT}" "$(get_balance "${GUILD_B_VAULT}" ualpha)"

run_tx "Non-member owner confiscates and burns Guild C token" \
    tx structs guild-bank-confiscate-and-burn 40 "${GUILD_C_VAULT}" --guild-id "${GUILD_C_ID}" --from guild_leader_b

assert_eq "Guild C token confiscated by its non-member owner" "$((GUILD_C_TOKENS_BEFORE + 60))" \
    "$(get_balance "${GUILD_C_VAULT}" "${GUILD_C_DENOM}")"
assert_eq "Named guild burn preserves treasury Alpha" "$((GUILD_C_ALPHA_BEFORE - 100))" "$(get_balance "${GUILD_C_VAULT}" ualpha)"
assert_eq "Named guild burn preserves collateral" "$((GUILD_C_COLLATERAL_BEFORE + 100))" "$(get_balance "${GUILD_C_COLLATERAL_ADDR}" ualpha)"
assert_eq "Named guild burn removes exact supply" "$((GUILD_C_SUPPLY_BEFORE + 60))" "$(guild_supply "${GUILD_C_DENOM}")"
assert_eq "Named guild burn preserves owner personal Alpha" "${LEADER_B_ALPHA_BEFORE_C_MINT}" "$(get_balance "${LEADER_B_VAULT}" ualpha)"
assert_eq "Named guild burn preserves membership guild Alpha" "${GUILD_B_ALPHA_BEFORE_C_MINT}" "$(get_balance "${GUILD_B_VAULT}" ualpha)"

# Naming a guild is not the same as holding rights on it.
run_tx_expect_fail "Stranger naming Guild C rejected" \
    tx structs guild-update-entry-rank 9 --guild-id "${GUILD_C_ID}" --from player_3

assert_eq "Guild C entry rank unchanged after the stranger's attempt" "42" \
    "$(jqr "$(query query structs guild "${GUILD_C_ID}")" '.Guild.entryRank' '0')"

# Hand Guild C back and restore what this block moved. The seller cannot take it
# back himself, having lost the permission row along with the ownership.
run_tx "Guild Leader B returns Guild C to Guild Leader C" \
    tx structs guild-update-owner-id "${GUILD_C_ID}" "${GUILD_LEADER_C_ID}" --from guild_leader_b

assert_eq "Guild C ownership restored" "${GUILD_LEADER_C_ID}" \
    "$(jqr "$(query query structs guild "${GUILD_C_ID}")" '.Guild.owner')"

run_tx "Restoring Guild Leader C rank to 1" \
    tx structs player-update-guild-rank "${GUILD_LEADER_C_ID}" 1 --from guild_leader_c
run_tx "Restoring Guild C entry rank to 101" \
    tx structs guild-update-entry-rank 101 --from guild_leader_c

# Rank 0 is not a settable value, so an unset reading is restored to the default
# rather than replayed.
if [ "${GUILD_B_ENTRY_RANK_BEFORE}" = "0" ]; then
    GUILD_B_ENTRY_RANK_BEFORE=101
fi
run_tx "Restoring Guild B entry rank to ${GUILD_B_ENTRY_RANK_BEFORE}" \
    tx structs guild-update-entry-rank "${GUILD_B_ENTRY_RANK_BEFORE}" --from guild_leader_b

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
# Player 5 grants PermDelete (8) on their own primary (already has it, but tests
# the grant path).
run_tx "Player 5 granting own address PermDelete (8)" \
    tx structs permission-grant-on-address "${PLAYER_5_ADDRESS}" 8 --from player_5

# Query permissions by player
PERM_BY_PLAYER=$(query query structs permission-by-player "${PLAYER_5_ID}" 2>/dev/null || echo '{}')
info "Permissions for Player 5 after address grant:"
echo "${PERM_BY_PLAYER}" | jq -r '.permissionRecord[]? | "  obj=\(.objectId) val=\(.value)"' 2>/dev/null | head -5 || echo "  (no records)"

# v0.22.0: a primary address must keep PermAll. Narrowing it in place is refused
# so a player cannot lock themselves out of every recovery route. Assert both
# revoke and set hit that gate; the supported way to hold a narrow key is to
# rotate the primary first, then narrow the old address.
run_tx_expect_fail "Player 5 cannot revoke PermDelete from primary address" \
    tx structs permission-revoke-on-address "${PLAYER_5_ADDRESS}" 8 --from player_5

# 33554423 = PermAll minus PermDelete = (2^25 - 1) ^ 8
run_tx_expect_fail "Player 5 cannot set primary address below PermAll" \
    tx structs permission-set-on-address "${PLAYER_5_ADDRESS}" 33554423 --from player_5

# With PermAll intact, a no-op primary self-update clears the gate.
run_tx "Player 5 can update primary address with PermAll" \
    tx structs player-update-primary-address "${PLAYER_5_ADDRESS}" --from player_5

# ─── PermAll gate on a secondary address ───
# Register a second key for Player 5, narrow it, and show that key cannot call
# player-update-primary-address. The primary stays whole throughout.
P5_SEC_KEY="player_5_sec"
P5_SEC_ADDR=""
ADDRESS_PROOF_REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CHAIN_ID_FOR_P5=$(query status | jq -er '.node_info.network | select(type == "string" and length > 0)' 2>/dev/null) || CHAIN_ID_FOR_P5=""
P5_SEC_PROOF=""
if [ -n "${CHAIN_ID_FOR_P5}" ] && [ "${CHAIN_ID_FOR_P5}" != "null" ]; then
    P5_SEC_PROOF=$(cd "${ADDRESS_PROOF_REPO}" && go run ./tests/tools/addressproof \
        -chain-id "${CHAIN_ID_FOR_P5}" -player "${PLAYER_5_ID}" -nonce 0 2>&1) || P5_SEC_PROOF=""
fi

if parse_address_proof "${P5_SEC_PROOF}"; then
    P5_SEC_ADDR="${ADDRESS}"
    assert_not_empty "Player 5 secondary address" "${P5_SEC_ADDR}"
    assert_eq "Player 5 secondary signing material generated" "true" "$([ -n "${PRIVKEY}" ] && echo true || echo false)"

    # Drop any leftover key from a prior run, then import the freshly minted one.
    structsd ${PARAMS_KEYS} keys delete "${P5_SEC_KEY}" --yes >/dev/null 2>&1 || true
    info "Importing Player 5 secondary key into keyring"
    IMPORT_OUT=$(structsd ${PARAMS_KEYS} keys import-hex "${P5_SEC_KEY}" "${PRIVKEY}" 2>&1) || true
    unset PRIVKEY P5_SEC_PROOF
    IMPORTED_ADDR=$(structsd ${PARAMS_KEYS} keys show "${P5_SEC_KEY}" 2>/dev/null | jq -er '.address | select(type == "string" and length > 0)' 2>/dev/null || true)
    if [ -z "${IMPORTED_ADDR}" ] || [ "${IMPORTED_ADDR}" = "null" ]; then
        IMPORTED_ADDR=$(structsd --home ~/.structs --keyring-dir ~/.structs --keyring-backend test keys show "${P5_SEC_KEY}" -a 2>/dev/null || true)
    fi
    if [ "${IMPORTED_ADDR}" != "${P5_SEC_ADDR}" ]; then
        echo -e "  ${RED}FAIL${NC}: imported key address '${IMPORTED_ADDR}' != proof address '${P5_SEC_ADDR}'"
        echo "  ${IMPORT_OUT}"
        FAIL_COUNT=$((FAIL_COUNT + 1))
        P5_SEC_ADDR=""
    else
        echo -e "  ${GREEN}Imported${NC}: ${P5_SEC_KEY} = ${IMPORTED_ADDR}"
        PASS_COUNT=$((PASS_COUNT + 1))
    fi

    if [ -n "${P5_SEC_ADDR}" ]; then
        # Register with PermAll so the subsequent narrow is the only thing stripping it.
        run_tx "Registering secondary address for Player 5 with PermAll" \
            tx structs address-register \
            "${PLAYER_5_ID}" \
            "${P5_SEC_ADDR}" \
            "${PROOF_PUBKEY}" \
            "${PROOF_SIGNATURE}" \
            33554431 --from player_5

        ADDR_CHECK_JSON=$(query query structs address "${P5_SEC_ADDR}")
        assert_eq "Player 5 secondary belongs to Player 5" "${PLAYER_5_ID}" "$(jqr "${ADDR_CHECK_JSON}" '.playerId')"

        run_tx "Narrowing Player 5 secondary to PermAll minus PermDelete" \
            tx structs permission-set-on-address "${P5_SEC_ADDR}" 33554423 --from player_5

        run_tx_expect_fail "Narrowed secondary cannot update primary address" \
            tx structs player-update-primary-address "${PLAYER_5_ADDRESS}" --from "${P5_SEC_KEY}"

        # Leave the secondary registered but restore PermAll so later phases that
        # touch Player 5's address set are not fighting a deliberately broken key.
        run_tx "Restoring Player 5 secondary to PermAll" \
            tx structs permission-set-on-address "${P5_SEC_ADDR}" 33554431 --from player_5
    fi
else
    echo -e "  ${RED}FAIL${NC}: could not generate Player 5 secondary address proof for the PermAll gate"
    FAIL_COUNT=$((FAIL_COUNT + 1))
    unset PRIVKEY P5_SEC_PROOF
fi

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
run_tx_expect_permission_denied "Player 5 (rank 5) tries to update guild endpoint (should fail)" \
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

run_tx_expect_permission_denied "Player 5 (rank 5) tries to update guild endpoint (should fail — combined mask, rank 3)" \
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

# Operational Update must not let a rank holder make free entry match a paid rank.
ENTRY_RANK_BEFORE_RANK_ATTEMPT=$(jqr "$(query query structs guild "${GUILD_ID}")" '.Guild.entryRank' '0')
run_tx_expect_permission_denied "Rank-derived Update cannot change free entry rank" \
    tx structs guild-update-entry-rank 2 --from player_4
assert_eq "Entry rank unchanged after rank-derived Update attempt" "${ENTRY_RANK_BEFORE_RANK_ATTEMPT}" \
    "$(jqr "$(query query structs guild "${GUILD_ID}")" '.Guild.entryRank' '0')"

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

# The access policy is an enum, and autocli takes the kebab-case rendering of
# the proto name -- openMarket is "open-market", not "open". A wrong value is
# refused by the client before it reaches the chain, which used to leave this
# block skipping its two cleanup assertions on the strength of an empty
# PROVIDER_ID rather than reporting anything.
run_tx "Create guild-owned provider for cleanup test" \
    tx structs provider-create "${SUBSTATION_ID}" \
    "1ualpha" "open-market" 0 0 100 1000 10 1000 --owner-id "${GUILD_ID}" --from alice
PROVIDER_ID=$(get_created_provider_id "${GUILD_ID}" "${SUBSTATION_ID}")
assert_not_empty "Guild-owned cleanup provider created" "${PROVIDER_ID}"
if [ -n "${PROVIDER_ID}" ]; then
    CLEANUP_PROVIDER_JSON=$(query query structs provider "${PROVIDER_ID}")
    assert_eq "Cleanup provider is owned by the guild" "${GUILD_ID}" "$(jqr "${CLEANUP_PROVIDER_JSON}" '.Provider.owner' '')"
    CLEANUP_PERMISSIONS=$(get_permission_by_object "${PROVIDER_ID}")
    CLEANUP_CREATOR_PERMISSION=$(echo "${CLEANUP_PERMISSIONS}" | jq -r --arg id "${PROVIDER_ID}@${PLAYER_1_ID}" '
        (.permissionRecords // .permissionRecord) | if type == "array" then
        ([.[] | select(.permissionId == $id) | .value] | first // 0) else "invalid response" end')
    assert_eq "Guild provider has no copied personal creator grant" "0" "${CLEANUP_CREATOR_PERMISSION}"
    CLEANUP_MEMBER_JSON=$(query query structs player "${PLAYER_4_ID}")
    assert_eq "Provider rank delegate belongs to the owning guild" "${GUILD_ID}" "$(jqr "${CLEANUP_MEMBER_JSON}" '.Player.guildId' '')"
    CLEANUP_MEMBER_RANK=$(jqr "${CLEANUP_MEMBER_JSON}" '.Player.guildRank' '0')
    assert_gt "Provider rank delegate has a valid rank" 0 "${CLEANUP_MEMBER_RANK}"
    run_tx_expect_permission_denied "Guild membership alone cannot manage a guild provider" \
        tx structs provider-update-access-policy "${PROVIDER_ID}" "guild-market" --from player_4
    run_tx "Grant current guild rank access to manage provider" \
        tx structs permission-guild-rank-set "${PROVIDER_ID}" "${GUILD_ID}" "${PERM_UPDATE}" "${CLEANUP_MEMBER_RANK}" --from alice
    run_tx "Rank delegate updates guild provider policy" \
        tx structs provider-update-access-policy "${PROVIDER_ID}" "guild-market" --from player_4
    assert_eq "Rank grant enables provider policy update" "guildMarket" "$(jqr "$(query query structs provider "${PROVIDER_ID}")" '.Provider.accessPolicy' '')"
    run_tx "Revoke rank access to manage provider" \
        tx structs permission-guild-rank-revoke "${PROVIDER_ID}" "${GUILD_ID}" "${PERM_UPDATE}" --from alice
    run_tx_expect_permission_denied "Revoked rank delegate cannot manage guild provider" \
        tx structs provider-update-access-policy "${PROVIDER_ID}" "open-market" --from player_4
    assert_eq "Rejected update preserves guild provider policy" "guildMarket" "$(jqr "$(query query structs provider "${PROVIDER_ID}")" '.Provider.accessPolicy' '')"
    CLEANUP_P2_GUILD_PERMS=$(get_permission_value_for_player "${GUILD_ID}" "${PLAYER_2_ID}")
    run_tx "Transfer guild ownership while its provider remains in place" \
        tx structs guild-update-owner-id "${GUILD_ID}" "${PLAYER_2_ID}" --from alice
    assert_eq "Guild ownership transferred for provider authority check" "${PLAYER_2_ID}" "$(jqr "$(query query structs guild "${GUILD_ID}")" '.Guild.owner' '')"
    run_tx "New guild owner manages existing provider" \
        tx structs provider-update-access-policy "${PROVIDER_ID}" "closed-market" --from player_2
    run_tx_expect_permission_denied "Former guild owner has no inherited provider authority" \
        tx structs provider-update-access-policy "${PROVIDER_ID}" "open-market" --from alice
    CLEANUP_PROVIDER_JSON=$(query query structs provider "${PROVIDER_ID}")
    assert_eq "Provider remains guild-owned across the handoff" "${GUILD_ID}" "$(jqr "${CLEANUP_PROVIDER_JSON}" '.Provider.owner' '')"
    assert_eq "Former owner's rejected update leaves policy unchanged" "closedMarket" "$(jqr "${CLEANUP_PROVIDER_JSON}" '.Provider.accessPolicy' '')"
    run_tx "Restore original guild owner after provider authority check" \
        tx structs guild-update-owner-id "${GUILD_ID}" "${PLAYER_1_ID}" --from player_2
    assert_eq "Original guild owner restored" "${PLAYER_1_ID}" "$(jqr "$(query query structs guild "${GUILD_ID}")" '.Guild.owner' '')"
    if [ "$(get_permission_value_for_player "${GUILD_ID}" "${PLAYER_2_ID}")" != "${CLEANUP_P2_GUILD_PERMS}" ]; then
        run_tx "Restore Player 2's pre-handoff guild grants" \
            tx structs permission-set-on-object "${GUILD_ID}" "${PLAYER_2_ID}" "${CLEANUP_P2_GUILD_PERMS}" --from alice
    fi
    run_tx "Restored guild owner manages provider" \
        tx structs provider-update-access-policy "${PROVIDER_ID}" "open-market" --from alice
    run_tx "Grant P4 permission on provider" \
        tx structs permission-grant-on-object "${PROVIDER_ID}" "${PLAYER_4_ID}" "${PERM_UPDATE}" --from alice
    run_tx "Set guild-rank on provider" \
        tx structs permission-guild-rank-set "${PROVIDER_ID}" "${GUILD_ID}" "${PERM_UPDATE}" 1 --from alice
    run_tx "Delete provider" \
        tx structs provider-delete "${PROVIDER_ID}" --from alice
    sleep "${SLEEP}"
    PERM_AFTER=$(get_permission_by_object "${PROVIDER_ID}")
    PERM_COUNT=$(echo "${PERM_AFTER}" | jq -r '(.permissionRecords // .permissionRecord) | if type == "array" then length else "invalid response" end')
    GRANK_AFTER=$(get_guild_rank_permission_by_object "${PROVIDER_ID}")
    GRANK_COUNT=$(echo "${GRANK_AFTER}" | jq -r '(.guild_rank_permission_records // .guildRankPermissionRecords) | if type == "array" then length else "invalid response" end')
    assert_eq "Permission records cleared after provider delete" "0" "${PERM_COUNT}"
    assert_eq "Guild rank records cleared after provider delete" "0" "${GRANK_COUNT}"
fi

# ─── Guild-controlled allocation and substation lifecycle ───
GGRID_TEST_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
GGRID_SOURCE_BEFORE=$(query query structs substation "${SUBSTATION_ID}" | python3 "${GGRID_TEST_DIR}/player_allocatable_capacity.py" --substation-id "${SUBSTATION_ID}" --required-capacity 1000)
GGRID_LOAD_BEFORE=$(echo "${GGRID_SOURCE_BEFORE}" | jq -er '.load')
assert_eq "Guild grid fixture has source headroom" "true" "$(echo "${GGRID_SOURCE_BEFORE}" | jq -r '.hasCapacity')"
run_tx "Create guild-controlled allocation from the main substation" \
    tx structs allocation-create "${SUBSTATION_ID}" 1000 --controller "${GUILD_ID}" --allocation-type dynamic --from alice
GGRID_ALLOC_ID=$(get_created_grid_id allocation "${SUBSTATION_ID}" "${GUILD_ID}") || GGRID_ALLOC_ID=""
assert_not_empty "Guild-controlled allocation created" "${GGRID_ALLOC_ID}"
if [ -n "${GGRID_ALLOC_ID}" ]; then
    assert_eq "Allocation controller is the guild" "${GUILD_ID}" "$(jqr "$(query query structs allocation "${GGRID_ALLOC_ID}")" '.Allocation.controller' '')"
    run_tx "Transfer allocation control from guild to player" \
        tx structs allocation-transfer "${GGRID_ALLOC_ID}" "${PLAYER_1_ID}" --from alice
    assert_eq "Allocation control transferred to player" "${PLAYER_1_ID}" "$(jqr "$(query query structs allocation "${GGRID_ALLOC_ID}")" '.Allocation.controller' '')"
    run_tx "Return allocation control to the guild" \
        tx structs allocation-transfer "${GGRID_ALLOC_ID}" "${GUILD_ID}" --from alice
    run_tx "Create guild-owned substation from guild allocation" \
        tx structs substation-create "${GUILD_ID}" "${GGRID_ALLOC_ID}" --from alice
    GGRID_STATION_ID=$(get_created_grid_id substation "${GUILD_ID}" "${GGRID_ALLOC_ID}") || GGRID_STATION_ID=""
    assert_not_empty "Guild substation created" "${GGRID_STATION_ID}"
    if [ -n "${GGRID_STATION_ID}" ] && [ "${GGRID_STATION_ID}" != "${SUBSTATION_ID}" ]; then
        assert_eq "Substation belongs to guild" "${GUILD_ID}" "$(jqr "$(query query structs substation "${GGRID_STATION_ID}")" '.Substation.owner' '')"
        run_tx "Guild owner updates guild substation name" \
            tx structs substation-update-name "${GGRID_STATION_ID}" "Guild Grid Test" --from alice
        assert_eq "Guild substation name updated" "Guild Grid Test" "$(jqr "$(query query structs substation "${GGRID_STATION_ID}")" '.Substation.name' '')"
        run_tx "Resize the guild-controlled allocation" \
            tx structs allocation-update "${GGRID_ALLOC_ID}" 750 --from alice
        assert_eq "Guild substation receives resized capacity" "750" "$(jqr "$(query query structs substation "${GGRID_STATION_ID}")" '.gridAttributes.capacity' 'invalid')"
        run_tx_expect_permission_denied "Membership alone cannot delete a guild substation" \
            tx structs substation-delete "${GGRID_STATION_ID}" "" --from player_4
        run_tx "Delete the temporary guild substation and its allocation" \
            tx structs substation-delete "${GGRID_STATION_ID}" "" --from alice
        assert_eq "Guild substation removed" "" "$(jqr "$(query query structs substation "${GGRID_STATION_ID}" 2>/dev/null || echo '{}')" '.Substation.id' '')"
        assert_eq "Incoming guild allocation removed with substation" "" "$(jqr "$(query query structs allocation "${GGRID_ALLOC_ID}" 2>/dev/null || echo '{}')" '.Allocation.id' '')"
        GGRID_SOURCE_AFTER=$(query query structs substation "${SUBSTATION_ID}" | python3 "${GGRID_TEST_DIR}/player_allocatable_capacity.py" --substation-id "${SUBSTATION_ID}")
        assert_eq "Main substation load restored after guild grid cleanup" "${GGRID_LOAD_BEFORE}" "$(echo "${GGRID_SOURCE_AFTER}" | jq -er '.load')"
    else
        assert_eq "Guild substation creation produced a new object" "true" "false"
        run_tx "Clean up guild allocation after failed substation creation" \
            tx structs allocation-delete "${GGRID_ALLOC_ID}" --from alice
    fi
else
    assert_eq "Guild allocation creation produced a new object" "true" "false"
fi

fi # phase 4e

if run_phase 482; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 4e2: Create 20 Rank-Test Players (rp_1..rp_20)
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 4e2: Rank-Test Player Setup (20 players)"

NUM_RP=20
info "Creating ${NUM_RP} rank-test players (rp_1..rp_20)"
# Resume may already have recovered these arrays. Rebuild them in key order,
# reusing accounts below, rather than appending a duplicate fixture set.
RP_IDS=()
RP_KEYS=()

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

    # 4M fuel preserves the previous 2M pre-commission energy at the minimum
    # entry rate; 6M funding leaves the same 2M signing-key balance.
    run_tx "Fund ${RP_KEY}" tx bank send "${PLAYER_1_ADDRESS}" "${ADDR}" 6000000ualpha --from alice
    run_tx "Delegate ${RP_KEY}" tx staking delegate "${VALIDATOR_ADDRESS}" 4000000ualpha --from "${RP_KEY}"

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

if [ "${#RP_IDS[@]}" -ne 20 ] || [ "${#RP_KEYS[@]}" -ne 20 ]; then
    assert_eq "Rank tests require all 20 player IDs" "20" "${#RP_IDS[@]}"
    assert_eq "Rank tests require all 20 signing keys" "20" "${#RP_KEYS[@]}"
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
    if [ "${THRESHOLD}" -eq 0 ]; then
        # Rank 0 is not a legal threshold: PermissionGuildRankSet rejects it before
        # touching state, so the record the DENY assertions below face is whatever
        # was there already. Revoke it so those assertions test the rule they read
        # as testing — that no rank record denies everyone — rather than a leftover.
        run_tx_expect_reject_matching "Set PermGuildEndpointUpdate threshold=0 on guild (should fail — rank 0 is forbidden)" \
            "cannot be lower than minimum rank" \
            tx structs permission-guild-rank-set "${GUILD_ID}" "${GUILD_ID}" "${PERM_GUILD_ENDPOINT_UPDATE}" "${THRESHOLD}" --from alice
        run_tx "Revoke any guild-rank PermGuildEndpointUpdate record (threshold=0 means no record)" \
            tx structs permission-guild-rank-revoke "${GUILD_ID}" "${GUILD_ID}" "${PERM_GUILD_ENDPOINT_UPDATE}" --from alice
    else
        run_tx "Set PermGuildEndpointUpdate threshold=${THRESHOLD} on guild" \
            tx structs permission-guild-rank-set "${GUILD_ID}" "${GUILD_ID}" "${PERM_GUILD_ENDPOINT_UPDATE}" "${THRESHOLD}" --from alice
    fi

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

# ── 4e3c: Explicit rank administration ──────────────────────────────────────
info "--- Explicit Rank Administration ---"

run_tx_expect_permission_denied "Rank 2 alone cannot promote a lower-ranked member" \
    tx structs player-update-guild-rank "${RP_IDS[9]}" 7 --from player_4
assert_eq "Rank-only promotion leaves rp_10 unchanged" "10" "$(get_player_guild_rank "${RP_IDS[9]}")"
run_tx_expect_permission_denied "Rank 2 alone cannot demote a paying member" \
    tx structs player-update-guild-rank "${RP_IDS[14]}" 18 --from player_4
assert_eq "Rank-only demotion leaves rp_15 unchanged" "15" "$(get_player_guild_rank "${RP_IDS[14]}")"

run_tx "Grant Player 4 explicit guild administration" \
    tx structs permission-grant-on-object "${GUILD_ID}" "${PLAYER_4_ID}" "${PERM_ADMIN}" --from alice
run_tx "Explicit administrator promotes rp_10 to rank 1" \
    tx structs player-update-guild-rank "${RP_IDS[9]}" 1 --from player_4
assert_eq "Administrator can assign rank 1" "1" "$(get_player_guild_rank "${RP_IDS[9]}")"
run_tx "Explicit administrator demotes rp_15" \
    tx structs player-update-guild-rank "${RP_IDS[14]}" 18 --from player_4
assert_eq "Administrator demotion applied" "18" "$(get_player_guild_rank "${RP_IDS[14]}")"
run_tx "Revoke Player 4 explicit guild administration" \
    tx structs permission-revoke-on-object "${GUILD_ID}" "${PLAYER_4_ID}" "${PERM_ADMIN}" --from alice
run_tx_expect_permission_denied "Revoked administrator cannot keep promoting via rank" \
    tx structs player-update-guild-rank "${RP_IDS[9]}" 7 --from player_4

# ── 4e3d: Rank authority cannot manufacture administration ───────────────────
info "--- Rank Authority Cannot Grant Permanent Permissions ---"
run_tx "Configure rank-derived admin to test the explicit-authority boundary" \
    tx structs permission-guild-rank-set "${GUILD_ID}" "${GUILD_ID}" "${PERM_ADMIN}" 5 --from alice
run_tx_expect_permission_denied "Rank-derived admin cannot promote members for free" \
    tx structs player-update-guild-rank "${RP_IDS[6]}" 4 --from player_4
run_tx_expect_permission_denied "Rank-derived admin cannot grant permanent admin" \
    tx structs permission-grant-on-object "${GUILD_ID}" "${PLAYER_4_ID}" "${PERM_ADMIN}" --from player_4
run_tx_expect_permission_denied "Rank-derived admin cannot transfer ownership" \
    tx structs guild-update-owner-id "${GUILD_ID}" "${PLAYER_4_ID}" --from player_4
run_tx "Remove rank-derived admin fixture" \
    tx structs permission-guild-rank-revoke "${GUILD_ID}" "${GUILD_ID}" "${PERM_ADMIN}" --from alice

# ── 4e3e: Mass rank shuffle and verify ──────────────────────────────────────
info "--- Mass Rank Shuffle ---"

# Twenty transactions from one key, back to back. This used to bypass run_tx for a
# bare `|| true` and a one-second sleep against SLEEP=2 everywhere else, and one of
# them went out on a stale sequence (expected 197, got 196) — silently dropped, so
# rp_15 kept rank 18 and the verification below reported it as a rank bug. run_tx
# waits the standard interval and confirms delivery, which is what serialises them.
for i in "${!RP_IDS[@]}"; do
    NEW_RANK=$(( ${#RP_IDS[@]} - i ))
    run_tx "Mass shuffle: ${RP_KEYS[$i]} to rank ${NEW_RANK}" \
        tx structs player-update-guild-rank "${RP_IDS[$i]}" "${NEW_RANK}" --from alice
done

# Delivery and resulting state are separate requirements: a successful message
# that writes the wrong rank must fail even when no transaction was rejected.
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
if [ "${SHUFFLE_FAIL}" -gt 0 ]; then
    echo -e "  ${RED}${SHUFFLE_FAIL} stored ranks did not match the requested shuffle${NC}"
fi
PASS_COUNT=$(( PASS_COUNT + SHUFFLE_PASS ))

# After shuffle: rp_1=rank 20, rp_20=rank 1
run_tx_expect_permission_denied "Rank 1 still cannot give another player a free promotion" \
    tx structs player-update-guild-rank "${RP_IDS[0]}" 10 --from "${RP_KEYS[19]}"
RANK=$(get_player_guild_rank "${RP_IDS[0]}")
assert_eq "rp_1 unchanged after rank-only promotion" "20" "${RANK}"

run_tx_expect_permission_denied "rp_1 without admin cannot modify rp_20" \
    tx structs player-update-guild-rank "${RP_IDS[19]}" 15 --from "${RP_KEYS[0]}"

# ── 4e3f: Edge cases ────────────────────────────────────────────────────────
info "--- Edge Cases ---"

run_tx_expect_permission_denied "Player 4 cannot self-modify rank without admin" \
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

run_tx_expect_reject_matching "Setting rank to 0 is rejected by parameter validation" 'parameter rank validation failed: zero' \
    tx structs player-update-guild-rank "${RP_IDS[0]}" 0 --from alice

# ── Clean up all ranks ──────────────────────────────────────────────────────
info "Resetting all test player ranks"
run_tx "Reset Player 4 rank to 101" tx structs player-update-guild-rank "${PLAYER_4_ID}" 101 --from alice
run_tx "Reset Player 5 rank to 101" tx structs player-update-guild-rank "${PLAYER_5_ID}" 101 --from alice
for i in "${!RP_IDS[@]}"; do
    run_tx "Reset ${RP_KEYS[$i]} rank to 101" \
        tx structs player-update-guild-rank "${RP_IDS[$i]}" 101 --from alice
    assert_eq "${RP_KEYS[$i]} rank restored for later tests" "101" "$(get_player_guild_rank "${RP_IDS[$i]}")"
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

# Allocation operations: Player 5 is the allocation controller, so they must sign.
# An earlier phase may already have connected this allocation here, in which case
# the handler refuses the no-op move; the assertion below is what matters.
run_tx_tolerate_reject "Connecting Player 5 allocation to substation" \
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
if temp_substation_create SECOND_SUB_ID "migration test"; then
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

    # The throwaway allocation still feeds SECOND_SUB_ID. Delete destroys it
    # with the substation; P1_ALLOC_ID never left the main feeder.
    run_tx "Deleting second substation (migrate to original)" \
        tx structs substation-delete "${SECOND_SUB_ID}" "${SUBSTATION_ID}" --from alice
    TEMP_ALLOC_ID=""

    # Verify second substation is gone
    DEL_SUB_JSON=$(query query structs substation "${SECOND_SUB_ID}" 2>/dev/null || echo '{}')
    DEL_SUB_ID=$(jqr "${DEL_SUB_JSON}" '.Substation.id' '')
    assert_eq "Second substation deleted" "" "${DEL_SUB_ID}"

    # Verify permissions were cleaned up with the substation
    PERM_CLEANUP_JSON=$(get_permission_by_object "${SECOND_SUB_ID}")
    PERM_CLEANUP_COUNT=$(echo "${PERM_CLEANUP_JSON}" | jq -er '.permissionRecords | length')
    info "Object permissions on deleted substation: ${PERM_CLEANUP_COUNT} records"
    assert_eq "Object permissions cleaned up after substation delete" "0" "${PERM_CLEANUP_COUNT}"

    GRANK_CLEANUP_JSON=$(get_guild_rank_permission_by_object "${SECOND_SUB_ID}")
    GRANK_CLEANUP_COUNT=$(echo "${GRANK_CLEANUP_JSON}" | jq -er '.guildRankPermissionRecords | length')
    info "Guild rank permissions on deleted substation: ${GRANK_CLEANUP_COUNT} records"
    assert_eq "Guild rank permissions cleaned up after substation delete" "0" "${GRANK_CLEANUP_COUNT}"

    # Main feeder never moved onto the throwaway, so it must still be powering
    # the main substation after the delete.
    P1_ALLOC_DST_AFTER=$(query query structs allocation "${P1_ALLOC_ID}" 2>/dev/null | jq -r '.Allocation.destinationId // empty' 2>/dev/null || echo "")
    assert_eq "Player 1 allocation survived the substation delete" "${SUBSTATION_ID}" "${P1_ALLOC_DST_AFTER}"
fi

# ─── Grid query coverage ───
info "Grid attributes sample:"
GRID_SAMPLE=$(query query structs grid-all --page-limit 3 | jq -ce '
    .gridRecords | select(type == "array" and length > 0)
    | select(all(.[]; (.attributeId | type == "string" and length > 0)))')
assert_gt "Grid query returns fixture attributes" 0 "$(echo "${GRID_SAMPLE}" | jq -r 'length')"
echo "${GRID_SAMPLE}" | jq -r '.[] | "  \(.attributeId)=\(.value // "0")"'

fi # phase 4f

if run_phase 495; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 4g: Reactor Operations
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 4g: Reactor Operations"

# Price the entire owner portfolio at one committed height. Deposits need not
# increase power under the diversity curve; correctness is exact accounting.
check_fixture_diversity() {
    local owner="$1" label="$2" proof
    if ! proof=$(python3 "${GENERATOR_TEST_DIR}/live_reactor_diversity.py" "${owner}" \
        -- structsd ${PARAMS_QUERY} 2>&1); then
        proof="verification failed: ${proof}"
    fi
    assert_eq "${label}: exact stored and usable diversity accounting" "verified" \
        "$(printf '%s' "${proof}" | jq -r '.status' 2>/dev/null || printf '%s' "${proof}")"
}
check_fixture_diversity "${PLAYER_2_ID}" "Before reactor operations"

# Query current reactor state
info "Querying reactors"
REACTOR_ALL=$(query query structs reactor-all 2>/dev/null || echo '{}')
echo "  Reactor count: $(echo "${REACTOR_ALL}" | jq '.Reactor | length' 2>/dev/null || echo '0')"

# Legacy stake remains at signing addresses after upgrade. Select the fixture
# reactor directly so the default vault deposit runs independently of legacy stake.
P2_REACTOR_JSON=$(query query structs reactor "${REACTOR_ID}" 2>/dev/null || echo '{}')
P2_VAL_ADDR=$(jqr "${P2_REACTOR_JSON}" '.Reactor.validator' '')
assert_not_empty "Reactor operation fixture has a validator" "${P2_VAL_ADDR}"
info "Player 2 vault reactor validator: ${P2_VAL_ADDR}"

if [ -n "${P2_VAL_ADDR}" ]; then
    PLAYER_2_VAULT=$(get_vault_address "${PLAYER_2_ID}")
    fund_vault_alpha "${PLAYER_2_ID}" 1000000
    P2_VAULT_BEFORE_INFUSE=$(get_balance "${PLAYER_2_VAULT}" ualpha)
    P2_SIGNING_BEFORE_INFUSE=$(get_balance "${PLAYER_2_ADDRESS}" ualpha)
    P2_LEGACY_STAKE_BEFORE=$(query query staking delegations "${PLAYER_2_ADDRESS}" --page-limit 100 | jq -ce '
        select((.pagination.next_key == null or .pagination.next_key == "") and
               (.pagination.nextKey == null or .pagination.nextKey == "")) |
        [.delegation_responses[]?.delegation] | sort_by(.validator_address)')
    # Snapshot Player 2 capacity before
    P2_CAP_BEFORE=$(query query structs player "${PLAYER_2_ID}" | jq -r '.gridAttributes.capacity // "0"' 2>/dev/null || echo "0")
    info "Player 2 capacity before infuse: ${P2_CAP_BEFORE}"

    # ─── reactor-infuse: Player 2 infuses additional Alpha ───
    run_tx "Player 2 infusing 1000000ualpha into reactor" \
        tx structs reactor-infuse "${P2_VAL_ADDR}" 1000000ualpha --from player_2
    assert_eq "Default reactor infusion debits player vault" \
        "$((P2_VAULT_BEFORE_INFUSE - 1000000))" "$(get_balance "${PLAYER_2_VAULT}" ualpha)"
    assert_eq "Default reactor infusion leaves signing-address funds unchanged" \
        "${P2_SIGNING_BEFORE_INFUSE}" "$(get_balance "${PLAYER_2_ADDRESS}" ualpha)"
    P2_VAULT_SHARES_INFUSED=$(query query staking delegations "${PLAYER_2_VAULT}" | jq -er \
        --arg validator "${P2_VAL_ADDR}" '.delegation_responses[] | select(.delegation.validator_address == $validator) | .delegation.shares')

    # This deposit can change diversity as well as base power.
    sleep "${SLEEP}"
    P2_CAP_AFTER=$(query query structs player "${PLAYER_2_ID}" | jq -r '.gridAttributes.capacity // "0"' 2>/dev/null || echo "0")
    info "Player 2 capacity after infuse: ${P2_CAP_AFTER}"
    check_fixture_diversity "${PLAYER_2_ID}" "After infusion"
    P2_INFUSION_INFUSED=$(query query structs infusion "${REACTOR_ID}" "${PLAYER_2_VAULT}")

    # ─── reactor-defuse: Player 2 defuses a small amount ───
    run_tx "Player 2 defusing 500000ualpha from reactor" \
        tx structs reactor-defuse "${P2_VAL_ADDR}" 500000ualpha --from player_2
    P2_INFUSION_DEFUSED=$(query query structs infusion "${REACTOR_ID}" "${PLAYER_2_VAULT}")
    P2_CAP_DEFUSED=$(query query structs player "${PLAYER_2_ID}" | jq -er '.gridAttributes.capacity')
    check_fixture_diversity "${PLAYER_2_ID}" "After defusion"

    # Get the creation height for cancel-defusion
    REACTOR_TEST_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
    UNBONDING_JSON=$(query query staking unbonding-delegations "${PLAYER_2_VAULT}" --page-limit 100)
    UNBOND_HEIGHT=$(echo "${UNBONDING_JSON}" | python3 "${REACTOR_TEST_DIR}/staking_unbonding_entries.py" "${PLAYER_2_VAULT}" "${P2_VAL_ADDR}")
    info "Unbonding entry creation height: ${UNBOND_HEIGHT}"

    if [ -n "${UNBOND_HEIGHT}" ]; then
        # ─── reactor-cancel-defusion ───
        run_tx "Cancelling defusion (re-delegate)" \
            tx structs reactor-cancel-defusion "${P2_VAL_ADDR}" 500000ualpha "${UNBOND_HEIGHT}" --from player_2
        P2_VAULT_SHARES_RESTORED=$(query query staking delegations "${PLAYER_2_VAULT}" | jq -er \
            --arg validator "${P2_VAL_ADDR}" '.delegation_responses[] | select(.delegation.validator_address == $validator) | .delegation.shares')
        assert_eq "Cancelling defusion restores vault delegation shares" "${P2_VAULT_SHARES_INFUSED}" "${P2_VAULT_SHARES_RESTORED}"

        # Verify delegation restored
        P2_CAP_RESTORED=$(query query structs player "${PLAYER_2_ID}" | jq -r '.gridAttributes.capacity // "0"' 2>/dev/null || echo "0")
        info "Player 2 capacity after cancel-defusion: ${P2_CAP_RESTORED}"
        assert_eq "Cancelling defusion restores the pre-defusion capacity" "${P2_CAP_AFTER}" "${P2_CAP_RESTORED}"
        check_fixture_diversity "${PLAYER_2_ID}" "After cancellation"
        # Cancellation restores flat backing; no historical pricing state is needed.
        P2_INFUSION_CANCELLED=$(query query structs infusion "${REACTOR_ID}" "${PLAYER_2_VAULT}")
        REACTOR_TEST_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
        if ! P2_CYCLE_PROOF=$(jq -n --argjson infused "${P2_INFUSION_INFUSED}" \
            --argjson defused "${P2_INFUSION_DEFUSED}" --argjson cancelled "${P2_INFUSION_CANCELLED}" \
            '{infused: $infused, defused: $defused, cancelled: $cancelled}' | \
            python3 "${REACTOR_TEST_DIR}/verify_reactor_defusion_cycle.py" \
                "${REACTOR_ID}" "${PLAYER_2_VAULT}" "${PLAYER_2_ID}" 500000 2>&1); then
            P2_CYCLE_PROOF="verification failed: ${P2_CYCLE_PROOF}"
        fi
        assert_eq "Defuse/cancel preserves flat backing" "verified" "${P2_CYCLE_PROOF}"
        P2_CANCELLED_ENTRIES=$(query query staking unbonding-delegations "${PLAYER_2_VAULT}" --page-limit 100 | python3 "${REACTOR_TEST_DIR}/staking_unbonding_entries.py" "${PLAYER_2_VAULT}" "${P2_VAL_ADDR}" --creation-height "${UNBOND_HEIGHT}")
        assert_eq "Full cancellation removes its vault unbonding entry" "0" "${P2_CANCELLED_ENTRIES}"
    else
        assert_eq "Vault defusion creates an unbonding entry" "present" "missing"
    fi

    P2_LEGACY_STAKE_AFTER=$(query query staking delegations "${PLAYER_2_ADDRESS}" --page-limit 100 | jq -ce '
        select((.pagination.next_key == null or .pagination.next_key == "") and
               (.pagination.nextKey == null or .pagination.nextKey == "")) |
        [.delegation_responses[]?.delegation] | sort_by(.validator_address)')
    assert_eq "Vault infuse/defuse/cancel leaves legacy delegation shares unchanged" \
        "${P2_LEGACY_STAKE_BEFORE}" "${P2_LEGACY_STAKE_AFTER}"

    # ─── Infusion/reactor query coverage ───
    info "Infusion query coverage:"
    INFUSION_SAMPLE=$(query query structs infusion-all --page-limit 3 | jq -ce '
        .Infusion | select(type == "array" and length > 0)
        | select(all(.[]; (.destinationId | type == "string" and length > 0)
            and (.address | type == "string" and length > 0)))')
    assert_gt "Infusion query returns fixture positions" 0 "$(echo "${INFUSION_SAMPLE}" | jq -r 'length')"
    echo "${INFUSION_SAMPLE}" | jq -r '.[] | "  dst=\(.destinationId) addr=\(.address)"'
    REACTOR_SAMPLE=$(query query structs reactor-all --page-limit 3 | jq -ce '
        .Reactor | select(type == "array" and length > 0)
        | select(all(.[]; (.id | type == "string" and length > 0)
            and (.validator | type == "string" and length > 0)))')
    assert_gt "Reactor query returns fixture reactors" 0 "$(echo "${REACTOR_SAMPLE}" | jq -r 'length')"
    echo "${REACTOR_SAMPLE}" | jq -r '.[] | "  \(.id) validator=\(.validator)"'
fi

fi # phase 4g

if run_phase 500; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 5: Address Register & Proxy Join (advanced)
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 5: Address Register & Proxy Join"

# The proof is generated here rather than hardcoded. It binds the chain id, the
# player id and the address's current registration nonce, so a literal captured
# from one run is worthless on the next chain - and a stale one does not fail
# loudly, it leaves this phase quietly registering nothing. tests/tools/addressproof
# signs with the same builder the chain verifies with, so the two cannot drift.
ADDRESS_PROOF_REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CHAIN_ID=$(query status | jq -er '.node_info.network | select(type == "string" and length > 0)' 2>/dev/null) || CHAIN_ID=""
REGISTERED_EXT_ADDR=""

if [ -z "${CHAIN_ID}" ] || [ "${CHAIN_ID}" = "null" ]; then
    ADDRESS_PROOF=""
    info "Could not read chain_id from the connected node"
else
    ADDRESS_PROOF=$(cd "${ADDRESS_PROOF_REPO}" && go run ./tests/tools/addressproof \
        -chain-id "${CHAIN_ID}" -player "${PLAYER_1_ID}" -nonce 0 2>&1) || ADDRESS_PROOF=""
fi

if parse_address_proof "${ADDRESS_PROOF}"; then
    assert_eq "Address register proof generated" "true" "true"
    unset PRIVKEY ADDRESS_PROOF
    REGISTERED_EXT_ADDR="${ADDRESS}"
    info "Generated address-register proof for ${REGISTERED_EXT_ADDR} on ${CHAIN_ID}"
fi

if [ -n "${REGISTERED_EXT_ADDR}" ]; then
    run_tx "Registering external address for Player 1" \
        tx structs address-register \
        "${PLAYER_1_ID}" \
        "${REGISTERED_EXT_ADDR}" \
        "${PROOF_PUBKEY}" \
        "${PROOF_SIGNATURE}" \
        127 --from alice

    # A hard assert: the proof was built for this chain and this player, so a
    # mismatch here is a real regression rather than stale test data.
    ADDR_CHECK_JSON=$(query query structs address "${REGISTERED_EXT_ADDR}")
    REGISTERED_PLAYER=$(jqr "${ADDR_CHECK_JSON}" '.playerId')
    assert_eq "Registered address belongs to Player 1" "${PLAYER_1_ID}" "${REGISTERED_PLAYER}"
else
    assert_eq "Address register proof generated" "true" "false"
    unset PRIVKEY ADDRESS_PROOF
fi

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

# The address registered in Phase 5 for Player 1. Resolved at use time rather
# than hardcoded, so this phase either tests the real registration or says why it
# cannot - it must not quietly pass against an address nobody registered.
if [ -z "${REGISTERED_EXT_ADDR:-}" ]; then
    info "SKIP: Phase 5 did not run, so there is no registered address to revoke"
    REGISTERED_EXT_ADDR=""
fi

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
elif [ -z "${REGISTERED_EXT_ADDR}" ]; then
    info "SKIP: no address from Phase 5 to revoke"
else
    fail "Registered address ${REGISTERED_EXT_ADDR} should belong to ${PLAYER_1_ID} but reports '${ADDR_PLAYER}'"
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
PLAYER_2_CMD_SHIP_ID=$(get_command_ship_id "${PLAYER_2_ID}" "${PLAYER_2_FLEET_ID}")
PLAYER_3_CMD_SHIP_ID=$(get_command_ship_id "${PLAYER_3_ID}" "${PLAYER_3_FLEET_ID}")
PLAYER_4_CMD_SHIP_ID=$(get_command_ship_id "${PLAYER_4_ID}" "${PLAYER_4_FLEET_ID}")

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
PLANET_QUERY_HEIGHT=$(query status | jq -er '.sync_info.latest_block_height
    | select(type == "string" and test("^[1-9][0-9]*$"))')
P2_PLANET_QUERY=$(query query structs planet "${PLAYER_2_PLANET_ID}" --height "${PLANET_QUERY_HEIGHT}")
P2_EXPECTED_SHIELD=$(echo "${P2_PLANET_QUERY}" | jq -er --arg id "${PLAYER_2_PLANET_ID}" '
    select(.Planet.id == $id) | .planetAttributes | select(type == "object")
    | if has("planetaryShield") then .planetaryShield else "0" end
    | select(type == "string" and test("^(0|[1-9][0-9]*)$"))
    | select(length < 20 or (length == 20 and . <= "18446744073709551615"))')
P2_QUERY_SHIELD=$(query query structs planet-attribute "${PLAYER_2_PLANET_ID}" planetaryShield \
    --height "${PLANET_QUERY_HEIGHT}" | jq -er '
    select(type == "object") | select((keys - ["attribute"] | length) == 0)
    | if has("attribute") then .attribute else "0" end
    | select(type == "string" and test("^(0|[1-9][0-9]*)$"))
    | select(length < 20 or (length == 20 and . <= "18446744073709551615"))')
assert_eq "Planet shield attribute matches its detail query at the same height" "${P2_EXPECTED_SHIELD}" "${P2_QUERY_SHIELD}"

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
    wait_for_charge "${PLAYER_2_ID}" "${CHARGE_BUILD}"
    run_tx "Initiating Mine Shaft build (type=14, ambit=land, slot=1)" \
        tx structs struct-build-initiate "${PLAYER_2_ID}" 14 land 1 --from player_2

    MINER_STRUCT_ID=$(get_created_struct_id "${PLAYER_2_ID}" 14)
    assert_new_struct "Miner struct ID" "${MINER_STRUCT_ID}" "${PLAYER_2_ID}" 14
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
    REFINERY_STRUCT_ID=$(get_created_struct_id "${PLAYER_2_ID}" 15)
    assert_new_struct "Refinery struct ID" "${REFINERY_STRUCT_ID}" "${PLAYER_2_ID}" 15
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
    # Resolve here as well as in banking phases so --resume does not depend on
    # an earlier phase having populated PLAYER_2_VAULT.
    PLAYER_2_VAULT=$(get_vault_address "${PLAYER_2_ID}")
    REFINE_VAULT_BEFORE=$(get_balance "${PLAYER_2_VAULT}" ualpha)
    REFINE_ORE_BEFORE=$(query query structs player "${PLAYER_2_ID}" | jq -er '.gridAttributes.ore')
    run_compute "Refining ore" \
        tx structs struct-ore-refine-compute "${REFINERY_STRUCT_ID}" --from player_2
    assert_eq "Refinement credits exactly one Alpha to player vault" \
        "$((REFINE_VAULT_BEFORE + 1000000))" "$(get_balance "${PLAYER_2_VAULT}" ualpha)"
    assert_eq "Refinement consumes exactly one stored ore" \
        "$((REFINE_ORE_BEFORE - 1))" "$(query query structs player "${PLAYER_2_ID}" | jq -er '.gridAttributes.ore')"
    # The event's legacy primaryAddress remains signing identity metadata.
    # Indexers must use vaultAddress as the credited account; amount is Alpha.
    REFINE_TX_HASH=$(_tx_json "${LAST_TX_OUTPUT}" | jq -r '.txhash // empty')
    REFINE_EVENT_MATCH=false
    if [ -n "${REFINE_TX_HASH}" ]; then
        REFINE_EVENT_MATCH=$(query query tx "${REFINE_TX_HASH}" | jq -r \
            --arg owner "${PLAYER_2_ID}" --arg vault "${PLAYER_2_VAULT}" \
            --arg primary "${PLAYER_2_ADDRESS}" '
            [(.events // [])[]
             | select(.type == "structs.structs.EventAlphaRefine")
             | .attributes[] | select(.key == "eventAlphaRefineDetail")
             | .value | fromjson] as $details
            | ($details | length) == 1 and
              ($details[0] | .playerId == $owner and .vaultAddress == $vault and
               .primaryAddress == $primary and (.amount | tostring) == "1")')
    fi
    assert_eq "Refinement event identifies the vault credit and player" "true" "${REFINE_EVENT_MATCH}"
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
CANCEL_TYPE_JSON=$(query query structs struct-type 18)
CANCEL_BUILD_DRAW=$(echo "${CANCEL_TYPE_JSON}" | jq -er '.StructType.buildDraw | select(type == "string" and test("^[0-9]+$"))')
assert_gt "Cancel fixture type has positive build draw" 0 "${CANCEL_BUILD_DRAW}"
echo "${CANCEL_TYPE_JSON}" | jq -r '.StructType | "  Type \(.id): \(.type) buildDraw=\(.buildDraw) category=\(.category)"'

# Initiate the build (Ore Bunker type 18, land). Both the slot and the build
# itself are conditional: with mining enabled Phase 7 has already taken land
# slots 1 and 2 for the Mine Shaft and Refinery, and their 500k+500k passive
# draw can leave Player 2 without the 750k of grid headroom an Ore Bunker needs.
# Missing fixture resources leave this selected phase incomplete. Record a
# failure instead of reporting a passing lifecycle without exercising cancel.
CANCEL_SLOT=$(first_free_slot planet "${PLAYER_2_PLANET_ID}" land)
CANCEL_STRUCT_ID=""
if [ -z "${CANCEL_SLOT}" ]; then
    info "Incomplete 7b: Player 2's planet has no free land slot for the cancel test"
else
    wait_for_charge "${PLAYER_2_ID}" "${CHARGE_BUILD}"
    run_tx "Initiating Ore Bunker build (type=18, land, slot=${CANCEL_SLOT})" \
        tx structs struct-build-initiate "${PLAYER_2_ID}" 18 land "${CANCEL_SLOT}" --from player_2

    CANCEL_STRUCT_ID=$(get_created_struct_id "${PLAYER_2_ID}" 18)
    if [ -z "${CANCEL_STRUCT_ID}" ]; then
        CANCEL_STRUCT_ID=""
        info "Incomplete 7b: build-initiate rejected — $(echo "${LAST_TX_OUTPUT}" | grep -o 'failed to execute message[^[]*' | head -1)"
    else
        assert_new_struct "Ore Bunker for cancel initiated" "${CANCEL_STRUCT_ID}" "${PLAYER_2_ID}" 18
        info "Struct to cancel: ${CANCEL_STRUCT_ID}"
    fi
fi

assert_not_empty "Build-cancel fixture must be created" "${CANCEL_STRUCT_ID}"
if [ -n "${CANCEL_STRUCT_ID}" ]; then
    # Build initiation reserves load; cancellation requires an unbuilt struct.
    CANCEL_JSON=$(query query structs struct "${CANCEL_STRUCT_ID}" 2>/dev/null || echo '{}')
    # Proto JSON omits false. Accept that only inside the expected object's
    # complete envelope; a failed query or malformed flag must still fail.
    CANCEL_BUILT=$(echo "${CANCEL_JSON}" | jq -er --arg id "${CANCEL_STRUCT_ID}" --arg owner "${PLAYER_2_ID}" '
        if (.Struct | type) == "object" and .Struct.id == $id and .Struct.owner == $owner
            and .Struct.type == "18" and (.structAttributes | type) == "object"
            and .structAttributes.isMaterialized == true then
            .structAttributes | (if has("isBuilt") then .isBuilt else false end)
            | if type == "boolean" then tostring else error("invalid built flag") end
        else error("invalid cancellation fixture") end' 2>/dev/null || echo 'invalid')
    assert_eq "Cancel fixture is still building" "false" "${CANCEL_BUILT}"

    # Verify structsLoad increased (buildDraw added)
    P2_JSON_MID=$(query query structs player "${PLAYER_2_ID}")
    P2_LOAD_MID=$(jqr "${P2_JSON_MID}" '.gridAttributes.structsLoad' '0')
    info "Player 2 structsLoad after build-initiate: ${P2_LOAD_MID}"
    assert_eq "Build initiation reserves the exact build draw" "$((P2_LOAD_BEFORE_CANCEL + CANCEL_BUILD_DRAW))" "${P2_LOAD_MID}"

    # Cancel the build
    run_tx "Cancelling Ore Bunker build" \
        tx structs struct-build-cancel "${CANCEL_STRUCT_ID}" --from player_2

    # Cancellation marks the struct destroyed; physical deletion happens later.
    CANCEL_GONE=$(struct_state_after_last_tx "${CANCEL_STRUCT_ID}")
    assert_eq "Cancelled struct is destroyed" "true" "$(jqr "${CANCEL_GONE}" '.structAttributes.isDestroyed' 'false')"

    # Verify structsLoad decreased back
    P2_JSON_AFTER=$(query query structs player "${PLAYER_2_ID}")
    P2_LOAD_AFTER_CANCEL=$(jqr "${P2_JSON_AFTER}" '.gridAttributes.structsLoad' '0')
    info "Player 2 structsLoad after cancel: ${P2_LOAD_AFTER_CANCEL}"
    assert_eq "StructsLoad restored after cancel" "${P2_LOAD_BEFORE_CANCEL}" "${P2_LOAD_AFTER_CANCEL}"
else
    info "Incomplete 7b: cancellation did not execute"
fi

# struct-type-all query coverage
info "All struct types count:"
echo "  $(query query structs struct-type-all 2>/dev/null | jq '.StructType | length' || echo '?') types"

fi # phase 7b

if run_phase 760; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 7c: Struct Trash
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 7c: Struct Trash"

# Unlike struct-build-cancel (which only removes an unfinished struct), struct-trash
# destroys any non-destroyed struct as long as the caller has play permission and the
# owner holds at least the struct type's build charge (which the action consumes). This
# integration phase exercises the on-chain trash path on a freshly-initiated (still
# building) struct; it deliberately skips a build-compute so it does not pay the
# peak-difficulty proof-of-work (this phase initiates and would compute immediately, so
# difficulty has not decayed). Trashing a fully BUILT struct is covered deterministically
# by the Go unit test TestMsgStructTrash. A missing fixture fails this selected phase.

P2_TRASH_LOAD_BASELINE=$(jqr "$(query query structs player "${PLAYER_2_ID}")" '.gridAttributes.structsLoad' '0')
info "Player 2 structsLoad before trash-target build: ${P2_TRASH_LOAD_BASELINE}"

# Same load squeeze as 7b: with mining enabled the extractor+refinery can leave
# Player 2 short of the Ore Bunker's build draw. The preceding cancellation
# also leaves rubble temporarily; wait for the existing bounded sweep window.
TRASH_TYPE_JSON=$(query query structs struct-type 18)
TRASH_BUILD_DRAW=$(echo "${TRASH_TYPE_JSON}" | jq -er '.StructType.buildDraw | select(type == "string" and test("^[0-9]+$"))')
TRASH_BUILD_CHARGE=$(echo "${TRASH_TYPE_JSON}" | jq -er '.StructType.buildCharge | select(type == "string" and test("^[0-9]+$"))')
assert_gt "Trash fixture type has positive build draw" 0 "${TRASH_BUILD_DRAW}"
TRASH_SLOT=$(wait_for_free_slot planet "${PLAYER_2_PLANET_ID}" land)
TRASH_STRUCT_ID=""
if [ -z "${TRASH_SLOT}" ]; then
    info "Incomplete 7c: Player 2's planet has no free land slot for the trash test"
else
    wait_for_charge "${PLAYER_2_ID}" "${TRASH_BUILD_CHARGE}"
    run_tx "Initiating Ore Bunker for trash (type=18, land, slot=${TRASH_SLOT})" \
        tx structs struct-build-initiate "${PLAYER_2_ID}" 18 land "${TRASH_SLOT}" --from player_2

    TRASH_STRUCT_ID=$(get_created_struct_id "${PLAYER_2_ID}" 18)
    if [ -z "${TRASH_STRUCT_ID}" ]; then
        TRASH_STRUCT_ID=""
        info "Incomplete 7c: build-initiate rejected — $(echo "${LAST_TX_OUTPUT}" | grep -o 'failed to execute message[^[]*' | head -1)"
    fi
fi

assert_not_empty "Trash fixture must be created" "${TRASH_STRUCT_ID}"
if [ -n "${TRASH_STRUCT_ID}" ]; then
    info "Trash target struct: ${TRASH_STRUCT_ID} (still building)"

    P2_TRASH_LOAD_MID=$(jqr "$(query query structs player "${PLAYER_2_ID}")" '.gridAttributes.structsLoad' '0')
    assert_eq "Trash fixture reserves the exact build draw" "$((P2_TRASH_LOAD_BASELINE + TRASH_BUILD_DRAW))" "${P2_TRASH_LOAD_MID}"

    # Trash the struct. This consumes the struct type's build charge.
    wait_for_charge "${PLAYER_2_ID}" "${TRASH_BUILD_CHARGE}"
    run_tx "Trashing Ore Bunker ${TRASH_STRUCT_ID}" \
        tx structs struct-trash "${TRASH_STRUCT_ID}" --from player_2

    # Authoritative check: the struct is now flagged destroyed.
    TRASH_GONE_JSON=$(struct_state_after_last_tx "${TRASH_STRUCT_ID}")
    assert_eq "7c — struct is destroyed after trash" "true" "$(jqr "${TRASH_GONE_JSON}" '.structAttributes.isDestroyed' 'false')"

    P2_TRASH_LOAD_AFTER=$(jqr "$(query query structs player "${PLAYER_2_ID}")" '.gridAttributes.structsLoad' '0')
    info "Player 2 structsLoad after trash: ${P2_TRASH_LOAD_AFTER}"
    assert_eq "7c — structsLoad released after trash" "${P2_TRASH_LOAD_BASELINE}" "${P2_TRASH_LOAD_AFTER}"
fi

fi # phase 7c

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

DESTROYER_STRUCT_ID=$(get_created_struct_id "${PLAYER_3_ID}" 9)
assert_new_struct "Destroyer struct ID" "${DESTROYER_STRUCT_ID}" "${PLAYER_3_ID}" 9
echo "  Destroyer Struct ID: ${DESTROYER_STRUCT_ID}"

# ─── Pre-seed builds for other players while P3's Destroyer computes ────────
# Difficulty decays with block age, so initiating now means much faster computes later.
# P2 and P4 have independent charge — no waiting on P3.

info "Pre-seeding P2 Defender Destroyer (type=9, land, slot=0) — needed Phase 11"
wait_for_charge "${PLAYER_2_ID}" "${CHARGE_BUILD}"
run_tx "Pre-seed: P2 Defender Destroyer (type=9, land, slot=0)" \
    tx structs struct-build-initiate "${PLAYER_2_ID}" 9 land 0 --from player_2

DEFENDER_STRUCT_ID=$(get_created_struct_id "${PLAYER_2_ID}" 9)
assert_new_struct "Defender struct ID" "${DEFENDER_STRUCT_ID}" "${PLAYER_2_ID}" 9
echo "  Defender Struct ID: ${DEFENDER_STRUCT_ID} (pre-seeded, compute deferred to Phase 11)"

info "Pre-seeding P4 Field Generator (type=20, land, slot=0) — needed Phase 15"
wait_for_charge "${PLAYER_4_ID}" "${CHARGE_BUILD}"
run_tx "Pre-seed: P4 Field Generator (type=20, land, slot=0)" \
    tx structs struct-build-initiate "${PLAYER_4_ID}" 20 land 0 --from player_4

GENERATOR_STRUCT_ID=$(get_created_struct_id "${PLAYER_4_ID}" 20)
assert_new_struct "Generator struct ID" "${GENERATOR_STRUCT_ID}" "${PLAYER_4_ID}" 20
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

# Verify fleet location and v0.21.0 queue occupancy (capacity 1 by default)
FLEET_3_JSON=$(query query structs fleet "${PLAYER_3_FLEET_ID}")
FLEET_3_LOC=$(jqr "${FLEET_3_JSON}" '.Fleet.locationId')
info "Player 3 fleet location after move: ${FLEET_3_LOC}"
assert_eq "P3 fleet at P2 planet" "${PLAYER_2_PLANET_ID}" "${FLEET_3_LOC}"
P2_QUEUE_COUNT=$(query query structs planet "${PLAYER_2_PLANET_ID}" | jq -r '.Planet.locationListCount // "0"')
assert_eq "P2 planet locationListCount after P3 arrive" "1" "${P2_QUEUE_COUNT}"

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

# ─── Required raid command and fleet preparation ───
combat_require_command PLAYER_2_CMD_SHIP_ID 2 "raid defender Command Ship" || exit 1
combat_require_command COMMAND_SHIP_ID 3 "raid attacker Command Ship" || exit 1
PLAYER_3_CMD_SHIP_ID="${COMMAND_SHIP_ID}"
combat_move_fleet 2 "${PLAYER_2_PLANET_ID}" || exit 1
combat_move_fleet 3 "${PLAYER_2_PLANET_ID}" || exit 1
# ─── Required raid command and fleet preparation complete ───

P2_PLANET_JSON=$(query query structs planet "${PLAYER_2_PLANET_ID}")
P2_SHIELD=$(jqr "${P2_PLANET_JSON}" '.planetAttributes.planetaryShield' '0')
P2_RAID_CLOCK=$(jqr "${P2_PLANET_JSON}" '.planetAttributes.blockStartRaid' '0')
info "P2 planet: planetaryShield=${P2_SHIELD} blockStartRaid=${P2_RAID_CLOCK}"

P2_CMD_JSON=$(query query structs struct "${PLAYER_2_CMD_SHIP_ID}" || echo '{}')
P2_CMD_ONLINE=$(jqr "${P2_CMD_JSON}" '.structAttributes.isOnline' 'false')
assert_eq "P2 Command Ship online before raid scenarios" "true" "${P2_CMD_ONLINE}"

# ─── Required mining/refining pause coverage ───
# Combat may have destroyed either ore system. Recover exact current identities,
# finish pending builds or rebuild/activate; unavailable required roles fail.
if [ "${SKIP_MINING}" = true ]; then
    info "Skipping mining/refining-pause assertions (--skip-mining)"
else
    combat_require_struct MINER_STRUCT_ID 2 14 planet "${PLAYER_2_PLANET_ID}" land "raid-pause Ore Extractor" || exit 1
    combat_require_struct REFINERY_STRUCT_ID 2 15 planet "${PLAYER_2_PLANET_ID}" land "raid-pause Ore Refinery" || exit 1
    P2_PLANET_JSON=$(query query structs planet "${PLAYER_2_PLANET_ID}") || exit 1
    printf '%s' "${P2_PLANET_JSON}" | jq -e -s --arg planet "${PLAYER_2_PLANET_ID}" --arg owner "${PLAYER_2_ID}" --arg raider "${PLAYER_3_FLEET_ID}" '
        def positive: type == "string" and test("^[1-9][0-9]*$")
            and (length < 20 or (length == 20 and . <= "18446744073709551615"));
        length == 1 and (.[0] |
        .Planet.id == $planet and .Planet.owner == $owner and .Planet.locationListStart == $raider
        and (.planetAttributes.blockRaiderArrived | positive)
        and (.planetAttributes.blockStartOreMine | positive)
        and (.planetAttributes.blockStartOreRefine | positive)
        and (.planetAttributes.oreMiningActiveQuantity | positive)
        and (.planetAttributes.oreRefiningActiveQuantity | positive))' >/dev/null || {
        echo "Required raid-pause planet, raider or active ore clocks unavailable" >&2
        exit 1
    }
    P2_RAIDER_ARRIVED=$(jqr "${P2_PLANET_JSON}" '.planetAttributes.blockRaiderArrived')
    P2_MINE_CLOCK_BEFORE=$(jqr "${P2_PLANET_JSON}" '.planetAttributes.blockStartOreMine')
    P2_REFINE_CLOCK_BEFORE=$(jqr "${P2_PLANET_JSON}" '.planetAttributes.blockStartOreRefine')
    info "Raid pause: required mining/refining systems online; arrival=${P2_RAIDER_ARRIVED}, mine=${P2_MINE_CLOCK_BEFORE}, refine=${P2_REFINE_CLOCK_BEFORE}"
    run_tx_expect_reject_matching "Ore mine compute refuses because of the raid" "is under raid: mining is paused until the raid ends" \
        tx structs struct-ore-mine-compute "${MINER_STRUCT_ID}" --from player_2
    run_tx_expect_reject_matching "Ore refine compute refuses because of the raid" "is under raid: refining is paused until the raid ends" \
        tx structs struct-ore-refine-compute "${REFINERY_STRUCT_ID}" --from player_2
fi
# ─── Required mining/refining pause coverage complete ───

# ─── Scenario A (expected bad): raid cannot be won while defender CMD online ───

assert_eq "blockStartRaid unset while defender Command Ship is online" "0" "${P2_RAID_CLOCK}"

run_tx_expect_reject_matching "Raid complete while defender Command Ship online" "shields_active" \
    tx structs planet-raid-complete "${PLAYER_3_FLEET_ID}" deadbeef 1 --from player_3

run_tx_expect_reject_matching "Raid compute fast-fails while shields are up" "shields are not vulnerable: no active raid window" \
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

run_tx_expect_reject_matching "Raid complete after shields restored" "shields_active" \
    tx structs planet-raid-complete "${PLAYER_3_FLEET_ID}" deadbeef 1 --from player_3

# ─── Scenario D (good): raid succeeds while defender CMD is offline ───

run_tx "P2 deactivates their Command Ship again (shields drop)" \
    tx structs struct-deactivate "${PLAYER_2_CMD_SHIP_ID}" --from player_2

P2_PLANET_JSON=$(query query structs planet "${PLAYER_2_PLANET_ID}")
P2_RAID_CLOCK=$(jqr "${P2_PLANET_JSON}" '.planetAttributes.blockStartRaid' '0')
assert_gt "blockStartRaid re-anchored for the raid attempt" "0" "${P2_RAID_CLOCK}"

# ─── Confirmed raid outcome and exact ore accounting ───
RAID_BEFORE=$(raid_state_at_height "$(get_block_height)") || exit 1
printf '%s' "${RAID_BEFORE}" | python3 "${GENERATOR_TEST_DIR}/raid_result.py" before >/dev/null || exit 1
run_compute "Completing planet raid (defender Command Ship offline)" \
    tx structs planet-raid-compute "${PLAYER_3_FLEET_ID}" --from player_3
RAID_TX_HASH=$(_tx_json "${LAST_TX_OUTPUT}" | jq -er '.txhash') || exit 1
RAID_RECEIPT=$(query query tx "${RAID_TX_HASH}") || exit 1
RAID_HEIGHT=$(printf '%s' "${RAID_RECEIPT}" | python3 "${GENERATOR_TEST_DIR}/raid_result.py" receipt \
    --before <(printf '%s' "${RAID_BEFORE}") --txhash "${RAID_TX_HASH}") || exit 1
RAID_AFTER=$(raid_state_at_height "${RAID_HEIGHT}") || exit 1
RAID_VERIFIED=$(printf '%s' "${RAID_AFTER}" | python3 "${GENERATOR_TEST_DIR}/raid_result.py" after \
    --before <(printf '%s' "${RAID_BEFORE}") --receipt <(printf '%s' "${RAID_RECEIPT}") --txhash "${RAID_TX_HASH}") || exit 1
assert_eq "Raid events, exact ore transfer, fleet recall and pause clocks verified" "${RAID_HEIGHT}" "$(jqr "${RAID_VERIFIED}" '.height')"
info "Raid seized $(jqr "${RAID_VERIFIED}" '.seized_ore') ore at height ${RAID_HEIGHT}"
P2_PLANET_JSON=$(jqr "${RAID_AFTER}" '.planet')
# ─── Confirmed raid outcome and exact ore accounting complete ───

# ─── Post-raid: ore clocks shifted, raider-arrived cleared, mining works again ───
if [ "${SKIP_MINING}" = true ]; then
    info "Skipping post-raid ore-clock assertions (--skip-mining)"
else
    P2_RAIDER_ARRIVED_AFTER=$(jqr "${P2_PLANET_JSON}" '.planetAttributes.blockRaiderArrived' '0')
    P2_MINE_CLOCK_AFTER=$(jqr "${P2_PLANET_JSON}" '.planetAttributes.blockStartOreMine' '0')
    P2_REFINE_CLOCK_AFTER=$(jqr "${P2_PLANET_JSON}" '.planetAttributes.blockStartOreRefine' '0')
    assert_eq "blockRaiderArrived cleared after raid ended" "0" "${P2_RAIDER_ARRIVED_AFTER}"
    assert_gt "mine clock shifted forward by raid pause" "${P2_MINE_CLOCK_BEFORE}" "${P2_MINE_CLOCK_AFTER}"
    assert_gt "refine clock shifted forward by raid pause" "${P2_REFINE_CLOCK_BEFORE}" "${P2_REFINE_CLOCK_AFTER}"
    info "Ore clocks after raid pause: mine ${P2_MINE_CLOCK_BEFORE} -> ${P2_MINE_CLOCK_AFTER}, refine ${P2_REFINE_CLOCK_BEFORE} -> ${P2_REFINE_CLOCK_AFTER}"

    run_compute "Mining ore after raid ends (planet productive again)" \
        tx structs struct-ore-mine-compute "${MINER_STRUCT_ID}" --from player_2
fi

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

# Recover commands in their current ambits and the pre-seeded Tank in any
# build state. A repeated run may have a wounded command: replace it only if
# it cannot survive the volley, so this phase always exercises its counter.
combat_require_command PLAYER_2_CMD_SHIP_ID 2 "P11 defender Command Ship" || exit 1
combat_require_command COMMAND_SHIP_ID 3 "P11 target Command Ship" || exit 1
P11_COMMAND_JSON=$(query query structs struct "${COMMAND_SHIP_ID}") || exit 1
P11_COMMAND_HEALTH=$(printf '%s' "${P11_COMMAND_JSON}" | jq -er '.structAttributes.health | select(type == "string" and test("^[1-6]$"))') || exit 1
if [ "${P11_COMMAND_HEALTH}" -le 2 ]; then
    wait_for_charge "${PLAYER_3_ID}" "${CHARGE_BUILD}" || exit 1
    run_tx "P11: Replace Command Ship too damaged to counter" \
        tx structs struct-trash "${COMMAND_SHIP_ID}" --from player_3 || exit 1
    combat_require_command COMMAND_SHIP_ID 3 "P11 replacement Command Ship" || exit 1
fi
P11_COMMAND_JSON=$(query query structs struct "${COMMAND_SHIP_ID}") || exit 1
P11_COMMAND_AMBIT=$(printf '%s' "${P11_COMMAND_JSON}" | jq -er '.Struct.operatingAmbit | select(. == "land" or . == "water" or . == "air" or . == "space")') || exit 1
if [ "${P11_COMMAND_AMBIT}" != land ]; then
    wait_for_charge "${PLAYER_3_ID}" "${CHARGE_MOVE}" || exit 1
    run_tx "P11: Move Command Ship into Tank weapon range" \
        tx structs struct-move "${COMMAND_SHIP_ID}" fleet land --from player_3 || exit 1
fi
PLAYER_3_CMD_SHIP_ID=${COMMAND_SHIP_ID}
combat_move_fleet 2 "${PLAYER_2_PLANET_ID}" || exit 1
combat_move_fleet 3 "${PLAYER_2_PLANET_ID}" || exit 1
# The shared oracle requires the Command Ship's two-damage counter. Counter
# damage bypasses Tank armor; a wounded Tank may legitimately be destroyed.
ev_attack P11 DEFENDER_STRUCT_ID 2 9 land COMMAND_SHIP_ID 3 1 land primaryWeapon "${PLAYER_2_PLANET_ID}" || exit 1

fi # phase 11

if run_phase 1200; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 12: Complex Battle — Build multi-unit fleet
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 12: Complex Battle Setup"

# Player 3 needs extra capacity for the full combat fleet. Supply enough new
# fuel for the former 5M increment even at the 50% minimum, before commission.
# Earlier address operations can leave no spendable coins on the signing key.
fund_vault_alpha "${PLAYER_3_ID}" 10000000
run_tx "Additional vault infusion for Player 3 (fleet capacity)" \
    tx structs reactor-infuse "${VALIDATOR_ADDRESS}" 10000000ualpha --from player_3

combat_require_command PLAYER_2_CMD_SHIP_ID 2 "P12 P2 Command Ship" || exit 1
combat_require_command COMMAND_SHIP_ID 3 "P12 P3 Command Ship" || exit 1
PLAYER_3_CMD_SHIP_ID=${COMMAND_SHIP_ID}
combat_move_fleet 2 "${PLAYER_2_PLANET_ID}" || exit 1
combat_move_fleet 3 "${PLAYER_3_PLANET_ID}" || exit 1

# Reserve missing roles first so their proof ages accumulate together. Existing
# live roles and pending builds are reused; slots come from the current fleet.
# The P2 Battleship must stay unbuilt for Phase 13's targeting refusal.
combat_require_struct SAM_STRUCT_ID 3 10 fleet "${PLAYER_3_FLEET_ID}" land "P12 SAM" "" player_3 prepare || exit 1
combat_require_struct P2_BATTLESHIP_ID 2 2 fleet "${PLAYER_2_FLEET_ID}" space "P13 unbuilt Battleship" "" player_2 pending || exit 1
combat_require_struct SUB_STRUCT_ID 3 13 fleet "${PLAYER_3_FLEET_ID}" water "P12 Submersible" "" player_3 prepare || exit 1
combat_require_struct INTERCEPTOR_ID 2 7 fleet "${PLAYER_2_FLEET_ID}" air "P14 Interceptor" "" player_2 prepare || exit 1
combat_require_struct BATTLESHIP_1_ID 3 2 fleet "${PLAYER_3_FLEET_ID}" space "P12 Battleship 1" "" player_3 prepare || exit 1
combat_require_struct BATTLESHIP_2_ID 3 2 fleet "${PLAYER_3_FLEET_ID}" space "P12 Battleship 2" "${BATTLESHIP_1_ID}" player_3 prepare || exit 1
combat_require_struct STEALTH_BOMBER_ID 3 6 fleet "${PLAYER_3_FLEET_ID}" air "P13b Stealth Bomber" "" player_3 prepare || exit 1
combat_require_struct CRUISER_ID 3 11 fleet "${PLAYER_3_FLEET_ID}" water "P14 Cruiser" "" player_3 prepare || exit 1

# Finish only the roles needed now; later phases retain their proof-aging head start.
combat_require_struct SAM_STRUCT_ID 3 10 fleet "${PLAYER_3_FLEET_ID}" land "P12 SAM" || exit 1
combat_require_struct SUB_STRUCT_ID 3 13 fleet "${PLAYER_3_FLEET_ID}" water "P12 Submersible" || exit 1
combat_require_struct BATTLESHIP_1_ID 3 2 fleet "${PLAYER_3_FLEET_ID}" space "P12 Battleship 1" "${BATTLESHIP_2_ID}" || exit 1
combat_require_struct BATTLESHIP_2_ID 3 2 fleet "${PLAYER_3_FLEET_ID}" space "P12 Battleship 2" "${BATTLESHIP_1_ID}" || exit 1

# Preserve a separate main Tank and a full-health armor-piercing target. A
# resumed wounded AP target must be replaced before Phase 13 measures two damage.
combat_require_struct DESTROYER_STRUCT_ID 3 9 fleet "${PLAYER_3_FLEET_ID}" land "P12 main Tank" || exit 1
combat_require_struct AP_TANK_ID 3 9 fleet "${PLAYER_3_FLEET_ID}" land "P13 armor target" "${DESTROYER_STRUCT_ID}" || exit 1
P12_AP_JSON=$(query query structs struct "${AP_TANK_ID}") || exit 1
P12_AP_HEALTH=$(printf '%s' "${P12_AP_JSON}" | jq -er '.structAttributes.health | select(type == "string" and test("^[1-3]$"))') || exit 1
if [ "${P12_AP_HEALTH}" -lt 3 ]; then
    wait_for_charge "${PLAYER_3_ID}" "${CHARGE_BUILD}" || exit 1
    run_tx "P12: Replace wounded armor-piercing target" tx structs struct-trash "${AP_TANK_ID}" --from player_3 || exit 1
    combat_require_struct AP_TANK_ID 3 9 fleet "${PLAYER_3_FLEET_ID}" land "P13 replacement armor target" "${DESTROYER_STRUCT_ID}" || exit 1
fi
P12_AP_JSON=$(query query structs struct "${AP_TANK_ID}") || exit 1
[ "$(printf '%s' "${P12_AP_JSON}" | jq -er '.structAttributes.health')" = 3 ] || exit 1

# Batch atomicity must fail for the intended missing object, then both status
# changes and both cleanup activations must match delivered events and state.
p12_batch_step reject || exit 1
p12_batch_step offline || exit 1
p12_batch_step first || exit 1
p12_batch_step second || exit 1

fi # phase 12

if run_phase 1300; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 13: Defense Setup & Attack Against Defenders
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 13: Defense Setup & Coordinated Attack"

# Restore the whole matchup on direct resume. The pending P2 ship preserves
# the intended unbuilt refusal; a previously completed one cannot stand in.
ar_require_command 2 || exit 1
ar_require_command 3 || exit 1
combat_move_fleet 2 "${PLAYER_2_PLANET_ID}" || exit 1
combat_move_fleet 3 "${PLAYER_3_PLANET_ID}" || exit 1
combat_require_struct P2_BATTLESHIP_ID 2 2 fleet "${PLAYER_2_FLEET_ID}" space "P13 unbuilt Battleship" "" player_2 pending || exit 1
combat_require_struct SAM_STRUCT_ID 3 10 fleet "${PLAYER_3_FLEET_ID}" land "P13 SAM" || exit 1
combat_require_struct SUB_STRUCT_ID 3 13 fleet "${PLAYER_3_FLEET_ID}" water "P13 Submersible" || exit 1
combat_require_struct BATTLESHIP_1_ID 3 2 fleet "${PLAYER_3_FLEET_ID}" space "P13 Battleship 1" "${BATTLESHIP_2_ID:-}" || exit 1
combat_require_struct BATTLESHIP_2_ID 3 2 fleet "${PLAYER_3_FLEET_ID}" space "P13 Battleship 2" "${BATTLESHIP_1_ID}" || exit 1
combat_require_struct DESTROYER_STRUCT_ID 3 9 fleet "${PLAYER_3_FLEET_ID}" land "P13 main Tank" || exit 1
combat_require_struct AP_TANK_ID 3 9 fleet "${PLAYER_3_FLEET_ID}" land "P13 armor target" "${DESTROYER_STRUCT_ID}" || exit 1
P13_AP_JSON=$(query query structs struct "${AP_TANK_ID}") || exit 1
P13_AP_HEALTH=$(printf '%s' "${P13_AP_JSON}" | jq -er '.structAttributes.health | select(type == "string" and test("^[1-3]$"))') || exit 1
if [ "${P13_AP_HEALTH}" -lt 3 ]; then
    wait_for_charge "${PLAYER_3_ID}" "${CHARGE_BUILD}" || exit 1
    run_tx "P13: Replace wounded armor target" tx structs struct-trash "${AP_TANK_ID}" --from player_3 || exit 1
    combat_require_struct AP_TANK_ID 3 9 fleet "${PLAYER_3_FLEET_ID}" land "P13 replacement armor target" "${DESTROYER_STRUCT_ID}" || exit 1
fi
combat_clear_target_defenders 2 "${P2_BATTLESHIP_ID}" || exit 1
combat_clear_target_defenders 3 "${COMMAND_SHIP_ID}" || exit 1
combat_clear_target_defenders 3 "${AP_TANK_ID}" || exit 1
combat_move_fleet 3 "${PLAYER_2_PLANET_ID}" || exit 1
p13_attack unbuilt "${BATTLESHIP_1_ID}" "${P2_BATTLESHIP_ID}" 3 2 secondaryWeapon || exit 1
combat_require_struct P2_BATTLESHIP_ID 2 2 fleet "${PLAYER_2_FLEET_ID}" space "P13 attacker Battleship" || exit 1
p13_attack range "${P2_BATTLESHIP_ID}" "${COMMAND_SHIP_ID}" 2 3 primaryWeapon || exit 1
p13_attack armour "${P2_BATTLESHIP_ID}" "${AP_TANK_ID}" 2 3 primaryWeapon || exit 1

# All four defenders can counter into space. The first three in store order
# destroy the attacker before its volley; the fourth and target cannot counter.
for DEF_ID in "${SAM_STRUCT_ID}" "${SUB_STRUCT_ID}" "${BATTLESHIP_1_ID}" "${BATTLESHIP_2_ID}"; do
    ar_defense_set 3 "${DEF_ID}" "${COMMAND_SHIP_ID}" || exit 1
done
p13_attack defended "${P2_BATTLESHIP_ID}" "${COMMAND_SHIP_ID}" 2 3 secondaryWeapon \
    "${SAM_STRUCT_ID} ${SUB_STRUCT_ID} ${BATTLESHIP_1_ID} ${BATTLESHIP_2_ID}" || exit 1

fi # phase 13

if run_phase 1350; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 13b: Defense Clear & Stealth Systems
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 13b: Defense Clear & Stealth Systems"

# Restore each required fixture even when entering directly on resume or when
# the earlier battle destroyed it. A missing role is a failure, never a skip.
ar_require_command 3
combat_move_fleet 3 "${PLAYER_3_PLANET_ID}"
combat_require_struct SAM_STRUCT_ID 3 10 fleet "${PLAYER_3_FLEET_ID}" land "P13b SAM"
combat_require_struct STEALTH_BOMBER_ID 3 6 fleet "${PLAYER_3_FLEET_ID}" air "P13b Stealth Bomber"

p13b_assert_defense() {
    local expected="$1" command defender
    command=$(query query structs struct "${COMMAND_SHIP_ID}") || return 1
    defender=$(query query structs struct "${SAM_STRUCT_ID}") || return 1
    assert_eq "P13b SAM defense relationship (${expected})" true "$(jq -n \
        --argjson command "${command}" --argjson defender "${defender}" \
        --arg commandId "${COMMAND_SHIP_ID}" --arg defenderId "${SAM_STRUCT_ID}" \
        --arg owner "${PLAYER_3_ID}" --argjson expected "${expected}" '
        $command.Struct.id == $commandId and $defender.Struct.id == $defenderId and
        $command.Struct.owner == $owner and $defender.Struct.owner == $owner and
        $command.structAttributes.isBuilt == true and $defender.structAttributes.isBuilt == true and
        (($command.structAttributes.isDestroyed // false) == false) and
        (($defender.structAttributes.isDestroyed // false) == false) and
        ([$command.structDefenders[]? | select(. == $defenderId)] | length) == (if $expected then 1 else 0 end) and
        ($defender.structAttributes.protectedStructIndex // "0") == (if $expected then ($commandId | ltrimstr("5-")) else "0" end)')"
}

# Establish the assignment explicitly: clearing an already-unassigned SAM would
# not exercise teardown. Both the protected list and reverse index must agree.
wait_for_charge "${PLAYER_3_ID}" "${CHARGE_DEFEND}"
run_tx "P13b: establish SAM defense assignment" \
    tx structs struct-defense-set "${SAM_STRUCT_ID}" "${COMMAND_SHIP_ID}" --from player_3
p13b_assert_defense true
wait_for_charge "${PLAYER_3_ID}" "${CHARGE_DEFEND}"
run_tx "Clearing SAM defense assignment" \
    tx structs struct-defense-clear "${SAM_STRUCT_ID}" --from player_3
p13b_assert_defense false
wait_for_charge "${PLAYER_3_ID}" "${CHARGE_DEFEND}"
run_tx "Re-setting SAM to defend Command Ship" \
    tx structs struct-defense-set "${SAM_STRUCT_ID}" "${COMMAND_SHIP_ID}" --from player_3
p13b_assert_defense true

p13b_assert_stealth() {
    local expected="$1" payload
    payload=$(query query structs struct "${STEALTH_BOMBER_ID}") || return 1
    assert_eq "P13b living Bomber hidden=${expected}" true "$(printf '%s' "${payload}" | jq -r \
        --arg id "${STEALTH_BOMBER_ID}" --arg owner "${PLAYER_3_ID}" --argjson expected "${expected}" '
        .Struct.id == $id and .Struct.owner == $owner and .Struct.type == "6" and
        .structAttributes.isBuilt == true and .structAttributes.isOnline == true and
        ((.structAttributes.isDestroyed // false) == false) and
        ((.structAttributes.isHidden // false) == $expected)')"
}

# A resumed Bomber may already be hidden. Normalize and prove visible before
# testing a new activation, so a successful no-op cannot satisfy the check.
SB_JSON=$(query query structs struct "${STEALTH_BOMBER_ID}")
SB_HIDDEN=$(printf '%s' "${SB_JSON}" | jq -r '.structAttributes.isHidden // false')
if [ "${SB_HIDDEN}" = true ]; then
    wait_for_charge "${PLAYER_3_ID}" "${CHARGE_ACTIVATE}"
    run_tx "P13b: reveal resumed Stealth Bomber" \
        tx structs struct-stealth-deactivate "${STEALTH_BOMBER_ID}" --from player_3
fi
p13b_assert_stealth false
wait_for_charge "${PLAYER_3_ID}" "${CHARGE_ACTIVATE}"
run_tx "Activating stealth on Stealth Bomber" \
    tx structs struct-stealth-activate "${STEALTH_BOMBER_ID}" --from player_3
p13b_assert_stealth true
wait_for_charge "${PLAYER_3_ID}" "${CHARGE_ACTIVATE}"
run_tx "Deactivating stealth on Stealth Bomber" \
    tx structs struct-stealth-deactivate "${STEALTH_BOMBER_ID}" --from player_3
p13b_assert_stealth false

# The dedicated attribute query must agree with the struct query too.
SB_ATTRIBUTE_HEIGHT=$(query query structs block-height | jq -er '.blockHeight | select(type == "string" and test("^[1-9][0-9]*$"))')
SB_JSON=$(query query structs struct "${STEALTH_BOMBER_ID}" --height "${SB_ATTRIBUTE_HEIGHT}")
SB_ATTRIBUTES=$(query query structs struct-attribute "${STEALTH_BOMBER_ID}" status --height "${SB_ATTRIBUTE_HEIGHT}")
assert_eq "P13b Bomber attribute query matches live visible state" true "$(jq -n \
    --argjson state "${SB_JSON}" --argjson attribute "${SB_ATTRIBUTES}" --arg id "${STEALTH_BOMBER_ID}" '
    $state.Struct.id == $id and $state.structAttributes.isBuilt == true and $state.structAttributes.isOnline == true and
    (($state.structAttributes.isHidden // false) == false) and (($state.structAttributes.isDestroyed // false) == false) and
    ($attribute.attribute | type) == "string" and ($attribute.attribute | test("^[1-9][0-9]*$")) and
    $state.structAttributes.status == $attribute.attribute')"

fi # phase 13b

if run_phase 1400; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 14: Cruiser vs Interceptor — Secondary Weapons & Defensive Maneuvers
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 14: Secondary Weapons & Defensive Maneuvers"

# Restore required roles before attacking, including on --resume-from 14.
# A dead or absent earlier fixture must be rebuilt, never counted as a skip.
ar_require_command 3
ar_require_command 2
combat_require_struct CRUISER_ID 3 11 fleet "${PLAYER_3_FLEET_ID}" water "P14 Cruiser"
combat_require_struct INTERCEPTOR_ID 2 7 fleet "${PLAYER_2_FLEET_ID}" air "P14 Interceptor"
combat_require_struct STEALTH_BOMBER_ID 3 6 fleet "${PLAYER_3_FLEET_ID}" air "P14 Stealth Bomber"
SB_JSON=$(query query structs struct "${STEALTH_BOMBER_ID}")
SB_HIDDEN=$(printf '%s' "${SB_JSON}" | jq -r --arg id "${STEALTH_BOMBER_ID}" 'select(.Struct.id == $id) | .structAttributes.isHidden // false')
if [ "${SB_HIDDEN}" = true ]; then
    wait_for_charge "${PLAYER_3_ID}" "${CHARGE_ACTIVATE}"
    run_tx "P14: reveal resumed Stealth Bomber" tx structs struct-stealth-deactivate "${STEALTH_BOMBER_ID}" --from player_3
else
    assert_eq "P14 Stealth Bomber starts visible" false "${SB_HIDDEN}"
fi
combat_move_fleet 2 "${PLAYER_2_PLANET_ID}"
combat_move_fleet 3 "${PLAYER_2_PLANET_ID}"

for P14_CASE in P14_1 P14_2; do
    ev_attack "${P14_CASE}" CRUISER_ID 3 11 water INTERCEPTOR_ID 2 7 air secondaryWeapon "${PLAYER_2_PLANET_ID}"
done

# Rebuild the target while commands are still online if either earlier hit
# destroyed it. No build or movement is needed during the offline assertion.
combat_require_struct INTERCEPTOR_ID 2 7 fleet "${PLAYER_2_FLEET_ID}" air "P14 offline target"

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

# A delivered attack and exact canonical outcome are required. The checker
# also binds the same live, offline Command Ship to both historical snapshots.
ev_attack P14_OFFLINE CRUISER_ID 3 11 water INTERCEPTOR_ID 2 7 air secondaryWeapon "${PLAYER_2_PLANET_ID}" "${COMMAND_SHIP_ID}"

# Defensive assignment set/clear works with the Command Ship offline. Use the
# Cruiser (freshly built, guaranteed alive) so we do not depend on SAM's combat
# survival; clearing afterward leaves no residual defense relationship.
wait_for_charge "${PLAYER_3_ID}" "${CHARGE_DEFEND}"
run_tx "Set Cruiser to defend Command Ship (Command Ship offline, v0.19.0)" \
    tx structs struct-defense-set "${CRUISER_ID}" "${COMMAND_SHIP_ID}" --from player_3
CMD_JSON=$(query query structs struct "${COMMAND_SHIP_ID}")
CRUISER_JSON=$(query query structs struct "${CRUISER_ID}")
assert_eq "P14 offline defense registered on both structs" true "$(jq -n --argjson command "${CMD_JSON}" --argjson cruiser "${CRUISER_JSON}" --arg id "${COMMAND_SHIP_ID}" --arg defender "${CRUISER_ID}" '
    $command.Struct.id == $id and $cruiser.Struct.id == $defender and
    (($command.structAttributes.isOnline // false) == false) and
    ([$command.structDefenders[] | select(. == $defender)] | length) == 1 and
    $cruiser.structAttributes.protectedStructIndex == ($id | ltrimstr("5-"))')"
wait_for_charge "${PLAYER_3_ID}" "${CHARGE_DEFEND}"
run_tx "Clear Cruiser defense (Command Ship offline, v0.19.0)" \
    tx structs struct-defense-clear "${CRUISER_ID}" --from player_3
CMD_JSON=$(query query structs struct "${COMMAND_SHIP_ID}")
CRUISER_JSON=$(query query structs struct "${CRUISER_ID}")
assert_eq "P14 offline defense cleared on both structs" true "$(jq -n --argjson command "${CMD_JSON}" --argjson cruiser "${CRUISER_JSON}" --arg id "${COMMAND_SHIP_ID}" --arg defender "${CRUISER_ID}" '
    $command.Struct.id == $id and $cruiser.Struct.id == $defender and
    (($command.structAttributes.isOnline // false) == false) and
    ([$command.structDefenders[]? | select(. == $defender)] | length) == 0 and
    ($cruiser.structAttributes.protectedStructIndex // "0") == "0"')"

# Stealth is required even when this phase is entered directly on resume.
wait_for_charge "${PLAYER_3_ID}" "${CHARGE_ACTIVATE}"
run_tx "Activate stealth with Command Ship offline (v0.19.0)" \
    tx structs struct-stealth-activate "${STEALTH_BOMBER_ID}" --from player_3
SB_JSON=$(query query structs struct "${STEALTH_BOMBER_ID}")
assert_eq "P14 Stealth Bomber hidden with Command Ship offline" true "$(printf '%s' "${SB_JSON}" | jq -er --arg id "${STEALTH_BOMBER_ID}" '.Struct.id == $id and .structAttributes.isBuilt == true and .structAttributes.isOnline == true and .structAttributes.isHidden == true')"
wait_for_charge "${PLAYER_3_ID}" "${CHARGE_ACTIVATE}"
run_tx "Deactivate stealth with Command Ship offline (v0.19.0)" \
    tx structs struct-stealth-deactivate "${STEALTH_BOMBER_ID}" --from player_3
SB_JSON=$(query query structs struct "${STEALTH_BOMBER_ID}")
assert_eq "P14 Stealth Bomber visible with Command Ship offline" true "$(printf '%s' "${SB_JSON}" | jq -er --arg id "${STEALTH_BOMBER_ID}" '.Struct.id == $id and .structAttributes.isBuilt == true and .structAttributes.isOnline == true and ((.structAttributes.isHidden // false) == false)')"
assert_eq "P14 Command Ship remains offline after defense and stealth" activate "$(combat_fixture_state "${COMMAND_SHIP_ID}" "${PLAYER_3_ID}" 1 "${PLAYER_3_FLEET_ID}" space)"

# Restore the Command Ship online so the remaining phases are unaffected.
wait_for_charge "${PLAYER_3_ID}" "${CHARGE_ACTIVATE}"
run_tx "P3 re-activates Command Ship (cleanup)" \
    tx structs struct-activate "${COMMAND_SHIP_ID}" --from player_3
CS_BACK_ONLINE=$(query query structs struct "${COMMAND_SHIP_ID}" | jq -r '.structAttributes.isOnline // "false"')
assert_eq "P3 Command Ship back online after offline test" "true" "${CS_BACK_ONLINE}"

fi # phase 14

if run_phase 1500; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 15: Power Generator — Build, Infuse, Verify, and Destroy
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 15: Power Generator (Player 4)"

echo "  Player 4 Planet: ${PLAYER_4_PLANET_ID}"
echo "  Using Field Generator (struct type 20, land)"

# Restore fixtures on fresh and resumed runs. Completing an existing build or
# reactivating a fueled generator must finish before capturing its baseline.
ar_require_command 4
ar_require_command 3
combat_move_fleet 4 "${PLAYER_4_PLANET_ID}"
combat_require_struct GENERATOR_STRUCT_ID 4 20 planet "${PLAYER_4_PLANET_ID}" land "P15 Field Generator"
combat_require_struct DESTROYER_STRUCT_ID 3 9 fleet "${PLAYER_3_FLEET_ID}" land "P15 Tank"
combat_clear_target_defenders 4 "${GENERATOR_STRUCT_ID}"
PLAYER_4_VAULT=$(get_vault_address "${PLAYER_4_ID}")
GEN_READY=false
for GEN_WAIT in $(seq 1 30); do
    GEN_FIXTURE=$(generator_fixture_snapshot)
    GEN_BASELINE=$(printf '%s' "${GEN_FIXTURE}" | python3 "${GENERATOR_TEST_DIR}/generator_lifecycle.py" \
        "${GENERATOR_STRUCT_ID}" "${PLAYER_4_ID}" "${PLAYER_4_VAULT}" "${PLAYER_4_PLANET_ID}")
    GEN_READY=$(printf '%s' "${GEN_BASELINE}" | jq -r '.ready')
    [ "${GEN_READY}" != true ] || break
    sleep "${SLEEP}"
done
if [ "${GEN_READY}" != true ]; then
    echo "FAIL: Generator restoration did not finish within the fixture allowance" >&2
    exit 1
fi
GEN_JSON=$(printf '%s' "${GEN_FIXTURE}" | jq -c '.generator')
P4_JSON=$(printf '%s' "${GEN_FIXTURE}" | jq -c '.player')
P4_CAP_BEFORE=$(printf '%s' "${GEN_BASELINE}" | jq -r '.capacityBefore')
P4_CAP_EXPECTED_INFUSED=$(printf '%s' "${GEN_BASELINE}" | jq -r '.capacityAfterInfusion')
P4_CAP_EXPECTED_DESTROYED=$(printf '%s' "${GEN_BASELINE}" | jq -r '.capacityAfterDestruction')
P4_LOAD_BEFORE=$(jqr "${P4_JSON}" '.gridAttributes.structsLoad' '0')
assert_eq "Generator ready with reconciled fuel and capacity" true "${GEN_READY}"
info "Player 4 capacity before top-up: ${P4_CAP_BEFORE}; after removal: ${P4_CAP_EXPECTED_DESTROYED}"

# ─── Infuse alpha into the generator (must be online) ───
# Generator fuel is burned from the player vault, independently of key funds.
PLAYER_4_VAULT=$(get_vault_address "${PLAYER_4_ID}")
fund_vault_alpha "${PLAYER_4_ID}" 1000000
# Funding can advance the chain: refresh the baseline before submitting the burn.
GEN_FIXTURE=$(generator_fixture_snapshot)
GEN_BASELINE=$(printf '%s' "${GEN_FIXTURE}" | python3 "${GENERATOR_TEST_DIR}/generator_lifecycle.py" \
    "${GENERATOR_STRUCT_ID}" "${PLAYER_4_ID}" "${PLAYER_4_VAULT}" "${PLAYER_4_PLANET_ID}")
printf '%s' "${GEN_BASELINE}" | jq -e '.ready == true' >/dev/null
GEN_BEFORE_HEIGHT=$(printf '%s' "${GEN_FIXTURE}" | jq -er '.height')
P4_JSON=$(printf '%s' "${GEN_FIXTURE}" | jq -c '.player')
P4_LOAD_BEFORE=$(jqr "${P4_JSON}" '.gridAttributes.structsLoad' '0')
P4_PRIMARY=$(printf '%s' "${GEN_FIXTURE}" | jq -er '.player.Player.primaryAddress | select(type == "string" and length > 0)')
P4_CAP_BEFORE=$(printf '%s' "${GEN_BASELINE}" | jq -er '.capacityBefore')
P4_CAP_EXPECTED_INFUSED=$(printf '%s' "${GEN_BASELINE}" | jq -er '.capacityAfterInfusion')
P4_CAP_EXPECTED_DESTROYED=$(printf '%s' "${GEN_BASELINE}" | jq -er '.capacityAfterDestruction')
GEN_FUEL_EXPECTED=$(printf '%s' "${GEN_BASELINE}" | jq -er '.fuelAfterInfusion')
P4_VAULT_BEFORE_GENERATOR=$(get_balance_at_height "${PLAYER_4_VAULT}" ualpha "${GEN_BEFORE_HEIGHT}")
P4_SIGNING_BEFORE_GENERATOR=$(get_balance_at_height "${PLAYER_4_ADDRESS}" ualpha "${GEN_BEFORE_HEIGHT}")
P4_VAULT_EXPECTED=$(python3 -c 'import sys; n=int(sys.argv[1]); assert n >= 1000000; print(n-1000000)' "${P4_VAULT_BEFORE_GENERATOR}")
GEN_MODULE=$(query query auth module-account structs --height "${GEN_BEFORE_HEIGHT}" | jq -er '
    .account | select(.type == "/cosmos.auth.v1beta1.ModuleAccount") | .value
    | select(.name == "structs") | .address | select(type == "string" and length > 0)')
INFUSE_AMOUNT="1000000ualpha"
info "Infusing ${INFUSE_AMOUNT} into generator (GeneratingRate=2, expected power=2000000)"

run_tx "Infusing ${INFUSE_AMOUNT} into generator ${GENERATOR_STRUCT_ID}" \
    tx structs struct-generator-infuse "${GENERATOR_STRUCT_ID}" "${INFUSE_AMOUNT}" --from player_4

GEN_TXHASH=$(_tx_json "${LAST_TX_OUTPUT}" | jq -er '.txhash | select(type == "string" and test("^[0-9A-Fa-f]{64}$"))')
GEN_RECEIPT=$(query query tx "${GEN_TXHASH}")
GEN_INFUSE_HEIGHT=$(printf '%s' "${GEN_RECEIPT}" | python3 "${GENERATOR_TEST_DIR}/generator_infuse.py" \
    "${GEN_TXHASH}" "${GENERATOR_STRUCT_ID}" "${PLAYER_4_ID}" "${PLAYER_4_VAULT}" "${P4_PRIMARY}" \
    "${PLAYER_4_ADDRESS}" "${GEN_MODULE}" "${GEN_FUEL_EXPECTED}" "${P4_CAP_EXPECTED_INFUSED}")
python3 -c 'import sys; assert int(sys.argv[1]) > int(sys.argv[2])' "${GEN_INFUSE_HEIGHT}" "${GEN_BEFORE_HEIGHT}"
GEN_JSON=$(query query structs struct "${GENERATOR_STRUCT_ID}" --height "${GEN_INFUSE_HEIGHT}")
GEN_FUEL=$(jqr "${GEN_JSON}" '.gridAttributes.fuel' '0')
info "Generator fuel after infusion: ${GEN_FUEL}"
assert_eq "Generator receives exactly the infused fuel" "${GEN_FUEL_EXPECTED}" "${GEN_FUEL}"
assert_eq "Generator infusion debits the player vault" "${P4_VAULT_EXPECTED}" "$(get_balance_at_height "${PLAYER_4_VAULT}" ualpha "${GEN_INFUSE_HEIGHT}")"
assert_eq "Generator infusion preserves signing-key funds" "${P4_SIGNING_BEFORE_GENERATOR}" "$(get_balance_at_height "${PLAYER_4_ADDRESS}" ualpha "${GEN_INFUSE_HEIGHT}")"

# Generator remains online — verify
GEN_ONLINE_AFTER_INFUSE=$(jqr "${GEN_JSON}" '.structAttributes.isOnline' 'false')
assert_eq "Generator still online after infuse" "true" "${GEN_ONLINE_AFTER_INFUSE}"

# ─── Verify Player 4's capacity increased from generator power ───
P4_JSON=$(query query structs player "${PLAYER_4_ID}" --height "${GEN_INFUSE_HEIGHT}")
P4_CAP_AFTER_GEN=$(jqr "${P4_JSON}" '.gridAttributes.capacity' '0')
P4_LOAD_AFTER_GEN=$(jqr "${P4_JSON}" '.gridAttributes.structsLoad' '0')
info "Player 4 capacity after generator online: ${P4_CAP_AFTER_GEN} (was ${P4_CAP_BEFORE})"
info "Player 4 structsLoad after generator online: ${P4_LOAD_AFTER_GEN} (was ${P4_LOAD_BEFORE})"
assert_eq "Player 4 receives exact generator capacity" "${P4_CAP_EXPECTED_INFUSED}" "${P4_CAP_AFTER_GEN}"

# This fixture burns one Alpha at its published generating rate of two. With
# zero commission its output is credited to the player, not the struct grid.
GEN_POWER=$((P4_CAP_AFTER_GEN - P4_CAP_BEFORE))
info "Generator contribution to player capacity: ${GEN_POWER}"

# ─── Verify generator energy can support allocations ───
# Outbound allocation uses primary capacity minus allocation load. Struct load
# and secondary capacity belong to the separate building/gameplay headroom check.
GENERATOR_TEST_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
P4_ALLOCATABLE=$(echo "${P4_JSON}" | python3 "${GENERATOR_TEST_DIR}/player_allocatable_capacity.py")
P4_AVAIL_CAP=$(echo "${P4_ALLOCATABLE}" | jq -er '.allocatableCapacity')
info "Player 4 allocatable primary capacity: ${P4_AVAIL_CAP}"
assert_eq "Player 4 has available capacity for allocations" "true" "$(echo "${P4_ALLOCATABLE}" | jq -r '.hasCapacity')"

# ─── Now destroy the generator: Player 3 attacks ───
info "--- Destruction Phase ---"
echo "  Player 3 will move fleet to Player 4's planet and destroy the generator"
echo "  Generator MaxHealth=8 + armour (AttackReduction=1): Tank (type=9, land) 2 dmg → 1 net per shot → 8 rounds"
echo "  NOTE: Command Ship (space ambit) cannot target land structs — using Tank instead"

combat_move_fleet 3 "${PLAYER_4_PLANET_ID}"

# Record generator health before attacks
GEN_JSON=$(query query structs struct "${GENERATOR_STRUCT_ID}")
GEN_HP_BEFORE=$(jqr "${GEN_JSON}" '.structAttributes.health' '0')
info "Generator health before attacks: ${GEN_HP_BEFORE} (MaxHealth=8; armour reduces each 2-dmg Tank hit to 1 net)"

# Attack round 1 — use Destroyer/Tank (type=9, land ambit, PrimaryWeaponAmbits=4=land)
# Tank PrimaryWeaponCharge=1, PrimaryWeaponDamage=2; generator AttackReduction=1 → 1 net.
wait_for_charge "${PLAYER_3_ID}" "${CHARGE_ATTACK_DEFAULT}"
run_tx "Attack round 1: Tank -> Generator (armour: 2 dmg → 1 net)" \
    tx structs struct-attack "${DESTROYER_STRUCT_ID}" "${GENERATOR_STRUCT_ID}" primaryWeapon --from player_3

GEN_JSON=$(generator_state_after_attack "${GEN_HP_BEFORE}")
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
    GEN_JSON=$(generator_state_after_attack "${GEN_HP_NOW}")
    GEN_HP_NOW=$(jqr "${GEN_JSON}" '.structAttributes.health' '0')
    info "Generator health after round ${round}: ${GEN_HP_NOW}"
done

# Keep the last confirmed attack-height state, even if the sweep has since run.
GEN_HP_AFTER=$(jqr "${GEN_JSON}" '.structAttributes.health' '0')
GEN_DESTROYED=$(jqr "${GEN_JSON}" '.structAttributes.isDestroyed' 'false')
info "Generator health after destruction loop: ${GEN_HP_AFTER}"
info "Generator isDestroyed: ${GEN_DESTROYED}"
assert_eq "Generator destroyed (health=0)" "0" "${GEN_HP_AFTER}"
assert_eq "Generator destroyed flag is set" "true" "${GEN_DESTROYED}"

# ─── Verify Player 4's capacity decreased — energy no longer generated ───
GEN_DESTROY_HEIGHT=$(generator_destruction_height)
P4_JSON=$(query query structs player "${PLAYER_4_ID}" --height "${GEN_DESTROY_HEIGHT}")
P4_CAP_AFTER_DESTROY=$(jqr "${P4_JSON}" '.gridAttributes.capacity' '0')
P4_LOAD_AFTER_DESTROY=$(jqr "${P4_JSON}" '.gridAttributes.structsLoad' '0')
info "Player 4 capacity after generator destroyed: ${P4_CAP_AFTER_DESTROY} (was ${P4_CAP_AFTER_GEN} with generator)"
info "Player 4 structsLoad after generator destroyed: ${P4_LOAD_AFTER_DESTROY} (was ${P4_LOAD_AFTER_GEN})"
assert_eq "Player 4 capacity removes all generator power including resumed fuel" "${P4_CAP_EXPECTED_DESTROYED}" "${P4_CAP_AFTER_DESTROY}"

# The one-position fixture's burned fuel is removed by bounded destruction.
# Exact removal of its capacity contribution was checked above.
GEN_FUEL_AFTER=$(jqr "${GEN_JSON}" '.gridAttributes.fuel' '0')
info "Generator fuel after destruction: ${GEN_FUEL_AFTER} (was ${GEN_FUEL})"
assert_eq "Generator fuel zeroed after destruction" "0" "${GEN_FUEL_AFTER}"
assert_eq "Generator destruction preserves already-debited vault funds" "${P4_VAULT_EXPECTED}" "$(get_balance_at_height "${PLAYER_4_VAULT}" ualpha "${GEN_DESTROY_HEIGHT}")"
assert_eq "Generator destruction preserves signing-key funds" "${P4_SIGNING_BEFORE_GENERATOR}" "$(get_balance_at_height "${PLAYER_4_ADDRESS}" ualpha "${GEN_DESTROY_HEIGHT}")"

echo ""
echo "  Summary: Generator built, infused (power=${GEN_POWER}), then destroyed."
echo "  Player 4 capacity: ${P4_CAP_BEFORE} → ${P4_CAP_AFTER_GEN} (with gen) → ${P4_CAP_AFTER_DESTROY} (destroyed)"

fi # phase 15

if run_phase 1550; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 15b: Player Operations
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 15b: Player Operations"

# ─── player-send: fund a signing key from its vault, then send to Player 3 ───
# NOTE: Player 5 lost PermissionAssets (8) in Phase 4e, so we use Player 2 instead.
# Earlier address operations can sweep all signing-address funds into the vault.
SEND_AMOUNT="100000"
PLAYER_2_VAULT=$(get_vault_address "${PLAYER_2_ID}")
fund_vault_alpha "${PLAYER_2_ID}" "${SEND_AMOUNT}"
P2_VAULT_BEFORE_SEND=$(get_balance "${PLAYER_2_VAULT}" ualpha)
P2_KEY_BEFORE_FUND=$(get_balance "${PLAYER_2_ADDRESS}" ualpha)
run_tx "Fund Player 2 signing key from its vault via player-send" \
    tx structs player-send "${PLAYER_2_VAULT}" "${PLAYER_2_ADDRESS}" "${SEND_AMOUNT}ualpha" --from player_2
assert_eq "Vault debited exactly for signing-key funding" \
    "$((P2_VAULT_BEFORE_SEND - SEND_AMOUNT))" "$(get_balance "${PLAYER_2_VAULT}" ualpha)"
assert_eq "Signing key credited exactly from vault" \
    "$((P2_KEY_BEFORE_FUND + SEND_AMOUNT))" "$(get_balance "${PLAYER_2_ADDRESS}" ualpha)"

P2_BALANCE_BEFORE=$(get_balance "${PLAYER_2_ADDRESS}" ualpha)
P3_BALANCE_BEFORE=$(get_balance "${PLAYER_3_ADDRESS}" ualpha)
run_tx "Player 2 sending ${SEND_AMOUNT}ualpha to Player 3 via player-send" \
    tx structs player-send "${PLAYER_2_ADDRESS}" "${PLAYER_3_ADDRESS}" "${SEND_AMOUNT}ualpha" --from player_2
P2_BALANCE_AFTER=$(get_balance "${PLAYER_2_ADDRESS}" ualpha)
P3_BALANCE_AFTER=$(get_balance "${PLAYER_3_ADDRESS}" ualpha)
assert_eq "Player 2 debited exactly by player-send" \
    "$((P2_BALANCE_BEFORE - SEND_AMOUNT))" "${P2_BALANCE_AFTER}"
assert_eq "Player 3 credited exactly by player-send, including an empty account" \
    "$((P3_BALANCE_BEFORE + SEND_AMOUNT))" "${P3_BALANCE_AFTER}"

# Use a fresh player: earlier guild joins can leave in-flight redelegations that
# correctly block primary-address movement. This fixture has no such operations.
ROTATION_KEY="vault_rotation_$(date +%s)_$$"
ROTATION_SECOND_KEY="${ROTATION_KEY}_secondary"
structsd ${PARAMS_KEYS} keys add "${ROTATION_KEY}" --no-backup >/dev/null
ROTATION_ADDRESS=$(structsd ${PARAMS_KEYS} keys show "${ROTATION_KEY}" | jq -er '.address | select(type == "string" and length > 0)')
ROTATION_VALIDATOR=$(jqr "$(query query structs reactor "${REACTOR_ID}")" '.Reactor.validator' '')
assert_not_empty "Rotation fixture validator exists" "${ROTATION_VALIDATOR}"
run_tx "Fund primary-rotation fixture" \
    tx bank send "${BOB_ADDRESS}" "${ROTATION_ADDRESS}" 4000000ualpha --from bob
run_tx "Create rotation player through initial delegation" \
    tx staking delegate "${ROTATION_VALIDATOR}" 2000000ualpha --from "${ROTATION_KEY}"
ROTATION_PLAYER=$(query query structs address "${ROTATION_ADDRESS}" | jq -er '.playerId | select(type == "string" and length > 0)')
ROTATION_VAULT=$(get_vault_address "${ROTATION_PLAYER}")
fund_vault_alpha "${ROTATION_PLAYER}" 1000000
run_tx "Create independent vault stake before primary rotation" \
    tx structs reactor-infuse "${ROTATION_VALIDATOR}" 500000ualpha --from "${ROTATION_KEY}"
ROTATION_VAULT_BALANCE=$(get_balance "${ROTATION_VAULT}" ualpha)
ROTATION_VAULT_SHARES=$(get_fixture_delegation_shares "${ROTATION_VAULT}" "${ROTATION_VALIDATOR}")

ROTATION_CHAIN_ID=$(query status | jq -er '.node_info.network | select(type == "string" and length > 0)')
ROTATION_REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ROTATION_PROOF=$(cd "${ROTATION_REPO}" && go run ./tests/tools/addressproof \
    -chain-id "${ROTATION_CHAIN_ID}" -player "${ROTATION_PLAYER}" -nonce 0) || ROTATION_PROOF=""
if parse_address_proof "${ROTATION_PROOF}" && \
    structsd ${PARAMS_KEYS} keys import-hex "${ROTATION_SECOND_KEY}" "${PRIVKEY}" >/dev/null 2>&1; then
ROTATION_SECOND_ADDRESS="${ADDRESS}"
ROTATION_PUBKEY="${PROOF_PUBKEY}"
ROTATION_SIGNATURE="${PROOF_SIGNATURE}"
unset PRIVKEY ROTATION_PROOF
assert_eq "Rotation proof matches imported key" "${ROTATION_SECOND_ADDRESS}" \
    "$(structsd ${PARAMS_KEYS} keys show "${ROTATION_SECOND_KEY}" | jq -er '.address | select(type == "string" and length > 0)')"
run_tx "Register rotation secondary with full permissions" \
    tx structs address-register "${ROTATION_PLAYER}" "${ROTATION_SECOND_ADDRESS}" \
    "${ROTATION_PUBKEY}" "${ROTATION_SIGNATURE}" 33554431 --from "${ROTATION_KEY}"
ROTATION_PRIMARY_BALANCE=$(get_balance "${ROTATION_ADDRESS}" ualpha)
ROTATION_PRIMARY_SHARES=$(get_fixture_delegation_shares "${ROTATION_ADDRESS}" "${ROTATION_VALIDATOR}")
ROTATION_EXPECTED_SHARES=$(python3 "${ROTATION_REPO}/tests/sum_staking_shares.py" "${ROTATION_VAULT_SHARES}" "${ROTATION_PRIMARY_SHARES}")
run_tx "Rotate primary to the new key" \
    tx structs player-update-primary-address "${ROTATION_SECOND_ADDRESS}" --from "${ROTATION_KEY}"
assert_eq "New primary address installed" "${ROTATION_SECOND_ADDRESS}" \
    "$(jqr "$(query query structs player "${ROTATION_PLAYER}")" '.Player.primaryAddress' '')"
assert_eq "Rotation preserves vault association" "${ROTATION_VAULT}" "$(get_vault_address "${ROTATION_PLAYER}")"
# ─── rotation exact balance accounting ───
ROTATION_MINIMUM_BALANCE=$(python3 - "${ROTATION_VAULT_BALANCE}" "${ROTATION_PRIMARY_BALANCE}" <<'PY_ROTATION'
import re
import sys
values = sys.argv[1:]
if not all(re.fullmatch(r"0|[1-9][0-9]*", value) for value in values):
    raise SystemExit("invalid rotation balance")
print(sum(map(int, values)))
PY_ROTATION
)
ROTATION_VAULT_BALANCE=$(get_balance "${ROTATION_VAULT}" ualpha)
ROTATION_OLD_PRIMARY_BALANCE=$(get_balance "${ROTATION_ADDRESS}" ualpha)
assert_ge "Rotation sweeps spendable balance and settled rewards to vault" \
    "${ROTATION_MINIMUM_BALANCE}" "${ROTATION_VAULT_BALANCE}"
assert_eq "Rotation clears old primary spendable funds" "0" "${ROTATION_OLD_PRIMARY_BALANCE}"
# ─── rotation exact balance accounting complete ───
assert_eq "Rotation merges primary stake with existing vault delegation" "${ROTATION_EXPECTED_SHARES}" \
    "$(get_fixture_delegation_shares "${ROTATION_VAULT}" "${ROTATION_VALIDATOR}")"
assert_eq "Rotation removes old primary delegation" "0.000000000000000000" \
    "$(get_fixture_delegation_shares "${ROTATION_ADDRESS}" "${ROTATION_VALIDATOR}" empty)"
run_tx "New primary signs restoration of original primary" \
    tx structs player-update-primary-address "${ROTATION_ADDRESS}" --from "${ROTATION_SECOND_KEY}"
assert_eq "Original primary restored" "${ROTATION_ADDRESS}" \
    "$(jqr "$(query query structs player "${ROTATION_PLAYER}")" '.Player.primaryAddress' '')"
run_tx "Revoke the rotation secondary" \
    tx structs address-revoke "${ROTATION_SECOND_ADDRESS}" --from "${ROTATION_KEY}"
# ─── rotation revoked spend accounting ───
ROTATION_BEFORE_REVOKED_SPEND=$(get_balance "${ROTATION_VAULT}" ualpha)
assert_ge "Restoration and revocation preserve vault funds including settled rewards" \
    "${ROTATION_VAULT_BALANCE}" "${ROTATION_BEFORE_REVOKED_SPEND}"
run_tx_expect_fail_noauto "Revoked rotation key cannot spend player vault funds" \
    tx structs vault-send "${ROTATION_PLAYER}" "${PLAYER_2_ID}" 1ualpha --from "${ROTATION_SECOND_KEY}"
ROTATION_AFTER_REVOKED_SPEND=$(get_balance "${ROTATION_VAULT}" ualpha)
assert_eq "Rejected revoked-key transfer preserves vault balance" "${ROTATION_BEFORE_REVOKED_SPEND}" "${ROTATION_AFTER_REVOKED_SPEND}"
# ─── rotation revoked spend accounting complete ───
assert_eq "Restoration leaves original primary without delegated stake" "0.000000000000000000" \
    "$(get_fixture_delegation_shares "${ROTATION_ADDRESS}" "${ROTATION_VALIDATOR}" empty)"
assert_eq "Revoked secondary retains no delegated stake" "0.000000000000000000" \
    "$(get_fixture_delegation_shares "${ROTATION_SECOND_ADDRESS}" "${ROTATION_VALIDATOR}" empty)"
assert_eq "Restoration and revocation preserve vault delegation" "${ROTATION_EXPECTED_SHARES}" \
    "$(get_fixture_delegation_shares "${ROTATION_VAULT}" "${ROTATION_VALIDATOR}")"
else
    echo -e "  ${RED}FAIL${NC}: rotation proof generation, validation or key import failed"
    FAIL_COUNT=$((FAIL_COUNT + 1))
    unset PRIVKEY ROTATION_PROOF
fi

fi # phase 15b

if run_phase 1600; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 16: Provider & Agreement System
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 16: Provider & Agreement System"

# Settlement also uses these Python helpers. Resume can skip the generator
# phase that first used them, so resolve the path again before any phase-16 work.
GENERATOR_TEST_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

PLAYER_1_VAULT=$(get_vault_address "${PLAYER_1_ID}")
PLAYER_2_VAULT=$(get_vault_address "${PLAYER_2_ID}")
PLAYER_3_VAULT=$(get_vault_address "${PLAYER_3_ID}")

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
# AutoCLI binds the rate positional to a Coin flag. Require its exact parse
# refusal below; an unrelated transport/permission failure is not this control.
run_tx_expect_fail "Alice attempting provider-create with bad rate (no denom)" \
    tx structs provider-create "${SUBSTATION_ID}" \
    "${BAD_RATE}" "${PROVIDER_ACCESS}" \
    "${PROVIDER_PROVIDER_PENALTY}" "${PROVIDER_CONSUMER_PENALTY}" \
    "${PROVIDER_CAP_MIN}" "${PROVIDER_CAP_MAX}" \
    "${PROVIDER_DUR_MIN}" "${PROVIDER_DUR_MAX}" \
    --from alice

# This is client validation. A global provider inventory cannot prove it: an
# existing provider is unrelated, and an unavailable query is not an empty list.
assert_eq "Bad provider rate refused by CLI coin parser" \
    'invalid argument "1" for "--1" flag: invalid input format' "${LAST_TX_OUTPUT}"

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
PROVIDER_ID=$(get_created_provider_id "${PLAYER_1_ID}" "${SUBSTATION_ID}")
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
    # Compare complete provider records after each update. Protobuf JSON may
    # omit the zero-valued openMarket enum; normalize only that valid default.
    PROV_RECORD_FILTER='.Provider | select(type == "object" and .id == $id) | if has("accessPolicy") then . else .accessPolicy = "openMarket" end'
    PROV_EXPECTED=$(query query structs provider "${PROVIDER_ID}" | jq -ceS --arg id "${PROVIDER_ID}" "${PROV_RECORD_FILTER}")
    run_tx "Updating provider access policy to 'open-market'" \
        tx structs provider-update-access-policy "${PROVIDER_ID}" "open-market" --from alice
    PROV_EXPECTED=$(echo "${PROV_EXPECTED}" | jq -ceS --arg value "openMarket" '.accessPolicy = $value')
    PROV_ACTUAL=$(query query structs provider "${PROVIDER_ID}" | jq -ceS --arg id "${PROVIDER_ID}" "${PROV_RECORD_FILTER}")
    assert_eq "Provider accessPolicy update preserves other terms" "${PROV_EXPECTED}" "${PROV_ACTUAL}"


    # Set back to guild
    run_tx "Setting provider access policy back to 'guild-market'" \
        tx structs provider-update-access-policy "${PROVIDER_ID}" "guild-market" --from alice
    PROV_EXPECTED=$(echo "${PROV_EXPECTED}" | jq -ceS --arg value "guildMarket" '.accessPolicy = $value')
    PROV_ACTUAL=$(query query structs provider "${PROVIDER_ID}" | jq -ceS --arg id "${PROVIDER_ID}" "${PROV_RECORD_FILTER}")
    assert_eq "Provider accessPolicy update preserves other terms" "${PROV_EXPECTED}" "${PROV_ACTUAL}"

    # ─── provider-update-capacity-minimum / maximum ───
    run_tx "Updating provider capacity minimum to 50000" \
        tx structs provider-update-capacity-minimum "${PROVIDER_ID}" 50000 --from alice
    PROV_EXPECTED=$(echo "${PROV_EXPECTED}" | jq -ceS --arg value "50000" '.capacityMinimum = $value')
    PROV_ACTUAL=$(query query structs provider "${PROVIDER_ID}" | jq -ceS --arg id "${PROVIDER_ID}" "${PROV_RECORD_FILTER}")
    assert_eq "Provider capacityMinimum update preserves other terms" "${PROV_EXPECTED}" "${PROV_ACTUAL}"

    run_tx "Updating provider capacity maximum to 10000000" \
        tx structs provider-update-capacity-maximum "${PROVIDER_ID}" 10000000 --from alice
    PROV_EXPECTED=$(echo "${PROV_EXPECTED}" | jq -ceS --arg value "10000000" '.capacityMaximum = $value')
    PROV_ACTUAL=$(query query structs provider "${PROVIDER_ID}" | jq -ceS --arg id "${PROVIDER_ID}" "${PROV_RECORD_FILTER}")
    assert_eq "Provider capacityMaximum update preserves other terms" "${PROV_EXPECTED}" "${PROV_ACTUAL}"

    # ─── provider-update-duration-minimum / maximum ───
    # Kept at 1 deliberately. A capacity change re-prices the unearned span and
    # now re-checks this minimum, so the capacity increase below (which computes
    # two thirds of the remaining duration) would need eight blocks still on the
    # clock against a minimum of 5. Blocks tick on wall time here, so that turns
    # the rest of the phase into a race whose failure reads as "duration below
    # minimum" rather than as the timeout it is. Nothing asserts a rejection
    # against this minimum, so lowering it costs no coverage; the alternative is
    # tripling the agreement collateral, which Player 2 needs for later phases.
    run_tx "Updating provider duration minimum to 1" \
        tx structs provider-update-duration-minimum "${PROVIDER_ID}" 1 --from alice
    PROV_EXPECTED=$(echo "${PROV_EXPECTED}" | jq -ceS --arg value "1" '.durationMinimum = $value')
    PROV_ACTUAL=$(query query structs provider "${PROVIDER_ID}" | jq -ceS --arg id "${PROVIDER_ID}" "${PROV_RECORD_FILTER}")
    assert_eq "Provider durationMinimum update preserves other terms" "${PROV_EXPECTED}" "${PROV_ACTUAL}"

    run_tx "Updating provider duration maximum to 50000" \
        tx structs provider-update-duration-maximum "${PROVIDER_ID}" 50000 --from alice
    PROV_EXPECTED=$(echo "${PROV_EXPECTED}" | jq -ceS --arg value "50000" '.durationMaximum = $value')
    PROV_ACTUAL=$(query query structs provider "${PROVIDER_ID}" | jq -ceS --arg id "${PROVIDER_ID}" "${PROV_RECORD_FILTER}")
    assert_eq "Provider durationMaximum update preserves other terms" "${PROV_EXPECTED}" "${PROV_ACTUAL}"


    # ─── agreement-open: Player 2 opens an agreement ───
    # Collateral = duration * capacity * rate
    AGREE_DURATION=$((40 * FIXTURE_DURATION_MULTIPLIER))
    AGREE_CAPACITY=50000

    fund_vault_alpha "${PLAYER_2_ID}" $((AGREE_DURATION * AGREE_CAPACITY + 20 * (AGREE_CAPACITY + 25000)))
    P2_ALPHA_BEFORE=$(get_balance "${PLAYER_2_VAULT}" "ualpha")
    info "Player 2 ualpha before agreement: ${P2_ALPHA_BEFORE}"

    # The policy above is guild-market, so CanOpenAgreement demands PermProviderOpen
    # on the provider itself — sharing Alice's guild is not standing on her provider.
    # Without this grant the open is refused and every assertion in the agreement
    # lifecycle below silently skips on an empty AGREE_ID.
    run_tx "Granting Player 2 PermProviderOpen on provider (guild-market access)" \
        tx structs permission-grant-on-object "${PROVIDER_ID}" "${PLAYER_2_ID}" "${PERM_PROVIDER_OPEN}" --from alice

    # ─── agreement-open collateral accounting ───
    # This is the fresh provider's first agreement: no prior load may be billed
    # by the opening checkpoint. Fees use the signing address, not this vault.
    PROV_OPEN_JSON=$(query query structs provider "${PROVIDER_ID}")
    PROV_OPEN_LOAD=$(echo "${PROV_OPEN_JSON}" | jq -er '.gridAttributes | select(type == "object") | if has("load") then .load else "0" end | select(type == "string")')
    assert_eq "Fresh provider has no prior agreement load" "0" "${PROV_OPEN_LOAD}"
    PROV_OPEN_COLL_ADDR=$(query query structs provider-collateral-address "${PROVIDER_ID}" | jq -er '.internalAddressAssociation[0].address | select(type == "string" and length > 0)')
    PROV_OPEN_COLL_BEFORE=$(get_balance "${PROV_OPEN_COLL_ADDR}" ualpha)
    P2_OPEN_VAULT_BEFORE=$(get_balance "${PLAYER_2_VAULT}" ualpha)
    PROV_OPEN_EARN_ADDR=$(query query structs provider-earnings-address "${PROVIDER_ID}" | jq -er --arg id "${PROVIDER_ID}" '.internalAddressAssociation | select(length == 1 and .[0].objectId == $id) | .[0].address | select(type == "string" and length > 0)')
    PROV_OPEN_EARN_BEFORE=$(get_balance "${PROV_OPEN_EARN_ADDR}" ualpha)
    AGREE_SERVICE_TERMS=$(jq -nc --arg provider "${PROVIDER_ID}" --arg owner "${PLAYER_2_ID}" --arg capacity "${AGREE_CAPACITY}" --arg vault "${PLAYER_2_VAULT}" --arg pool "${PROV_OPEN_COLL_ADDR}" '{provider_id:$provider, consumer_owner:$owner, capacity:$capacity, consumer_vault:$vault, collateral_address:$pool}')
    AGREE_TERMS_TEST_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
    # These fixture values are small, fixed integers (40 * 50000 * rate 1).
    AGREE_OPEN_COLLATERAL=$((AGREE_DURATION * AGREE_CAPACITY))
    run_tx "Player 2 opening agreement with provider (dur=${AGREE_DURATION}, cap=${AGREE_CAPACITY})" \
        tx structs agreement-open "${PROVIDER_ID}" "${AGREE_DURATION}" "${AGREE_CAPACITY}" --from player_2
    AGREE_SERVICE_TERMS=$(record_agreement_terms open "${AGREE_DURATION}" "${AGREE_SERVICE_TERMS}")
    assert_eq "Confirmed open funds exact collateral" "${AGREE_OPEN_COLLATERAL}" "$(printf '%s' "${AGREE_SERVICE_TERMS}" | jq -r '.principal')"
    PROV_OPEN_JSON=$(query query structs provider "${PROVIDER_ID}")
    PROV_OPEN_LOAD=$(echo "${PROV_OPEN_JSON}" | jq -er '.gridAttributes | select(type == "object") | if has("load") then .load else "0" end | select(type == "string")')
    assert_eq "Agreement open installs exact provider load" "${AGREE_CAPACITY}" "${PROV_OPEN_LOAD}"
    # ─── agreement-open collateral accounting complete ───

    # Identify this open from the confirmed transaction, not the newest list row.
    AGREE_ID=$(get_opened_agreement_id "${PROVIDER_ID}" "${PLAYER_2_ID}" "${AGREE_CAPACITY}")
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
        if temp_substation_create AGREE_TEST_SUB_ID "agreement allocation connect test"; then
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
            temp_substation_destroy "${AGREE_TEST_SUB_ID}" "agreement allocation connect test"
        fi

        # ─── agreement-capacity-increase ───
        run_tx "Increasing agreement capacity by 25000" \
            tx structs agreement-capacity-increase "${AGREE_ID}" 25000 --from player_2
        AGREE_SERVICE_TERMS=$(record_agreement_terms increase 25000 "${AGREE_SERVICE_TERMS}")

        AGREE_JSON=$(query query structs agreement "${AGREE_ID}")
        AGREE_CAP_AFTER=$(jqr "${AGREE_JSON}" '.Agreement.capacity' '0')
        info "Agreement capacity after increase: ${AGREE_CAP_AFTER}"
        AGREE_EXPECTED_CAPACITY=$((AGREE_CAPACITY + 25000))
        assert_eq "Agreement capacity increased by exact amount" "${AGREE_EXPECTED_CAPACITY}" "${AGREE_CAP_AFTER}"
        AGREE_RESIZE_POWER=$(query query structs allocation "${AGREE_ALLOC_ID}" | jq -er '.gridAttributes.power | select(type == "string")')
        AGREE_RESIZE_LOAD=$(query query structs provider "${PROVIDER_ID}" | jq -er '.gridAttributes.load | select(type == "string")')
        assert_eq "Resized agreement allocation power" "${AGREE_EXPECTED_CAPACITY}" "${AGREE_RESIZE_POWER}"
        assert_eq "Resized agreement provider load" "${AGREE_EXPECTED_CAPACITY}" "${AGREE_RESIZE_LOAD}"

        # ─── agreement-capacity-decrease ───
        run_tx "Decreasing agreement capacity by 10000" \
            tx structs agreement-capacity-decrease "${AGREE_ID}" 10000 --from player_2
        AGREE_SERVICE_TERMS=$(record_agreement_terms decrease 10000 "${AGREE_SERVICE_TERMS}")

        AGREE_JSON=$(query query structs agreement "${AGREE_ID}")
        AGREE_CAP_AFTER_DEC=$(jqr "${AGREE_JSON}" '.Agreement.capacity' '0')
        info "Agreement capacity after decrease: ${AGREE_CAP_AFTER_DEC}"
        AGREE_EXPECTED_CAPACITY=$((AGREE_CAPACITY + 25000 - 10000))
        assert_eq "Agreement capacity decreased by exact amount" "${AGREE_EXPECTED_CAPACITY}" "${AGREE_CAP_AFTER_DEC}"
        AGREE_RESIZE_POWER=$(query query structs allocation "${AGREE_ALLOC_ID}" | jq -er '.gridAttributes.power | select(type == "string")')
        AGREE_RESIZE_LOAD=$(query query structs provider "${PROVIDER_ID}" | jq -er '.gridAttributes.load | select(type == "string")')
        assert_eq "Resized agreement allocation power" "${AGREE_EXPECTED_CAPACITY}" "${AGREE_RESIZE_POWER}"
        assert_eq "Resized agreement provider load" "${AGREE_EXPECTED_CAPACITY}" "${AGREE_RESIZE_LOAD}"

        # ─── agreement-duration-increase ───
        AGREE_END_BEFORE_DUR=$(echo "${AGREE_JSON}" | jq -er '.Agreement.endBlock | select(type == "string" and test("^[0-9]+$"))')
        AGREE_EXPECTED_END=$(python3 -c 'import sys; end=int(sys.argv[1]); assert 0 <= end <= 2**64-21; print(end+20)' "${AGREE_END_BEFORE_DUR}")
        AGREE_BEFORE_EXTENSION=$(echo "${AGREE_JSON}" | jq -ceS '.Agreement | del(.endBlock)')
        AGREE_EXTENSION_COLLATERAL=$((20 * AGREE_EXPECTED_CAPACITY))
        run_tx "Increasing agreement duration by 20" \
            tx structs agreement-duration-increase "${AGREE_ID}" 20 --from player_2
        AGREE_SERVICE_TERMS=$(record_agreement_terms extend 20 "${AGREE_SERVICE_TERMS}")
        AGREE_TOTAL_PRINCIPAL=$(python3 -c 'import sys; print(sum(map(int, sys.argv[1:])))' "${AGREE_OPEN_COLLATERAL}" "${AGREE_EXTENSION_COLLATERAL}")
        assert_eq "Confirmed extension funds only the added service" "${AGREE_TOTAL_PRINCIPAL}" "$(printf '%s' "${AGREE_SERVICE_TERMS}" | jq -r '.principal')"

        AGREE_JSON=$(query query structs agreement "${AGREE_ID}")
        AGREE_END_AFTER_DUR=$(jqr "${AGREE_JSON}" '.Agreement.endBlock' '0')
        info "Agreement endBlock after duration increase: ${AGREE_END_AFTER_DUR} (was ${AGREE_END_BEFORE_DUR})"
        assert_eq "Agreement endBlock increased by exactly 20" "${AGREE_EXPECTED_END}" "${AGREE_END_AFTER_DUR}"
        assert_eq "Duration extension preserves other agreement fields" "${AGREE_BEFORE_EXTENSION}" \
            "$(echo "${AGREE_JSON}" | jq -ceS '.Agreement | del(.endBlock)')"
        assert_eq "Duration extension preserves allocation power" "${AGREE_EXPECTED_CAPACITY}" \
            "$(query query structs allocation "${AGREE_ALLOC_ID}" | jq -er '.gridAttributes.power | select(type == "string")')"
        assert_eq "Duration extension preserves provider load" "${AGREE_EXPECTED_CAPACITY}" \
            "$(query query structs provider "${PROVIDER_ID}" | jq -er '.gridAttributes.load | select(type == "string")')"

        # ─── agreement-close ───
        # This healthy fixture has rate 1 and zero cancellation penalties. The
        # transcript retains every capacity window and rounding credit; the
        # provider checkpoint is not a pricing clock for captured agreements.
        AGREE_CLOSE_PROVIDER=$(query query structs provider "${PROVIDER_ID}")
        echo "${AGREE_CLOSE_PROVIDER}" | jq -e '.Provider | .rate.denom == "ualpha" and .rate.amount == "1" and (.providerCancellationPenalty | test("^0(\\.0+)?$")) and (.consumerCancellationPenalty | test("^0(\\.0+)?$"))' >/dev/null
        run_tx "Closing agreement" \
            tx structs agreement-close "${AGREE_ID}" --from player_2
        AGREE_SERVICE_TERMS=$(record_agreement_terms close 0 "${AGREE_SERVICE_TERMS}")
        AGREE_CLOSE_EXPECTED=$(python3 "${AGREE_TERMS_TEST_DIR}/agreement_terms.py" balances "${AGREE_SERVICE_TERMS}" "${P2_OPEN_VAULT_BEFORE}" "${PROV_OPEN_COLL_BEFORE}" "${PROV_OPEN_EARN_BEFORE}")
        assert_eq "Close refunds exact unused collateral" "$(printf '%s' "${AGREE_CLOSE_EXPECTED}" | jq -r '.vault')" "$(get_balance "${PLAYER_2_VAULT}" ualpha)"
        assert_eq "Completed claim releases all its collateral" "$(printf '%s' "${AGREE_CLOSE_EXPECTED}" | jq -r '.collateral')" "$(get_balance "${PROV_OPEN_COLL_ADDR}" ualpha)"
        assert_eq "Provider receives exact elapsed-block earnings" "$(printf '%s' "${AGREE_CLOSE_EXPECTED}" | jq -r '.earnings')" "$(get_balance "${PROV_OPEN_EARN_ADDR}" ualpha)"
        # ─── agreement-close accounting complete ───

        # Verify teardown, including both temporary grants in this small fixture.
        assert_object_missing "Agreement closed (removed)" agreement "${AGREE_ID}"
        assert_object_missing "Closed agreement allocation removed" allocation "${AGREE_ALLOC_ID}"
        PROV_CLOSED_LOAD=$(query query structs provider "${PROVIDER_ID}" | jq -er '.gridAttributes | select(type == "object") | if has("load") then .load else "0" end | select(type == "string")')
        assert_eq "Agreement close releases provider load" "0" "${PROV_CLOSED_LOAD}"
        assert_eq "Agreement close clears owner agreement grant" "0" "$(get_permission_value_for_player "${AGREE_ID}" "${PLAYER_2_ID}")"
        assert_eq "Agreement close clears owner allocation grant" "0" "$(get_permission_value_for_player "${AGREE_ALLOC_ID}" "${PLAYER_2_ID}")"
    else
        assert_not_empty "Required agreement lifecycle fixture was created" "${AGREE_ID}"
    fi

    # ─── provider-withdraw-balance ───
    # Query provider collateral/earnings addresses
    PROV_COLLATERAL=$(query query structs provider-collateral-address "${PROVIDER_ID}")
    PROV_COLL_ADDR=$(echo "${PROV_COLLATERAL}" | jq -er '.internalAddressAssociation[0].address | select(type == "string" and length > 0)')
    info "Provider collateral address: ${PROV_COLL_ADDR}"

    PROV_EARN_ADDR=$(query query structs provider-earnings-address "${PROVIDER_ID}" | jq -er '.internalAddressAssociation[0].address | select(length > 0)')
    PROV_EARN_BEFORE=$(get_balance "${PROV_EARN_ADDR}" ualpha)
    ALICE_VAULT_BEFORE_WITHDRAW=$(get_balance "${PLAYER_1_VAULT}" ualpha)
    ALICE_VAULT_EXPECTED_WITHDRAW=$(python3 -c 'import sys; vault, earnings = map(int, sys.argv[1:]); assert min(vault, earnings) >= 0; print(vault+earnings)' "${ALICE_VAULT_BEFORE_WITHDRAW}" "${PROV_EARN_BEFORE}")
    run_tx "Withdrawing provider balance to Alice vault" \
        tx structs provider-withdraw-balance "${PROVIDER_ID}" --from alice
    assert_eq "Provider earnings reach Alice vault without transaction fees" \
        "${ALICE_VAULT_EXPECTED_WITHDRAW}" "$(get_balance "${PLAYER_1_VAULT}" ualpha)"
    assert_eq "Provider earnings pool emptied" "0" "$(get_balance "${PROV_EARN_ADDR}" ualpha)"

    # ─── provider-delete ───
    PROV_DELETE_LOAD=$(query query structs provider "${PROVIDER_ID}" | jq -er '.gridAttributes | select(type == "object") | if has("load") then .load else "0" end | select(type == "string")')
    assert_eq "Provider drain fixture has no active agreements" "0" "${PROV_DELETE_LOAD}"
    # Give both pools a positive balance even when prior settlement left no dust.
    run_tx "Seeding provider collateral drain with 1 ualpha" \
        tx bank send "${ALICE_ADDRESS}" "${PROV_COLL_ADDR}" 1ualpha --from alice
    run_tx "Seeding provider earnings drain with 1 ualpha" \
        tx bank send "${ALICE_ADDRESS}" "${PROV_EARN_ADDR}" 1ualpha --from alice
    PROV_DELETE_COLL_BEFORE=$(get_balance "${PROV_COLL_ADDR}" ualpha)
    PROV_DELETE_EARN_BEFORE=$(get_balance "${PROV_EARN_ADDR}" ualpha)
    PROV_DELETE_VAULT_BEFORE=$(get_balance "${PLAYER_1_VAULT}" ualpha)
    PROV_DELETE_EXPECTED_VAULT=$(python3 -c 'import sys
vault, collateral, earnings = map(int, sys.argv[1:])
assert vault >= 0 and collateral > 0 and earnings > 0
print(vault+collateral+earnings)' "${PROV_DELETE_VAULT_BEFORE}" "${PROV_DELETE_COLL_BEFORE}" "${PROV_DELETE_EARN_BEFORE}")
    run_tx "Deleting provider" \
        tx structs provider-delete "${PROVIDER_ID}" --from alice

    assert_object_missing "Provider deleted" provider "${PROVIDER_ID}"
    assert_eq "Provider deletion drains both pools to owner vault" "${PROV_DELETE_EXPECTED_VAULT}" "$(get_balance "${PLAYER_1_VAULT}" ualpha)"
    assert_eq "Deleted provider collateral pool is empty" "0" "$(get_balance "${PROV_COLL_ADDR}" ualpha)"
    assert_eq "Deleted provider earnings pool is empty" "0" "$(get_balance "${PROV_EARN_ADDR}" ualpha)"
    # ─── provider-delete accounting complete ───
else
    assert_not_empty "Required provider lifecycle fixture was created" "${PROVIDER_ID}"
fi

# ═════════════════════════════════════════════════════════════════════════════
#  Settlement accounting: an agreement settles exactly once, and one consumer's
#  collateral never funds another's payout.
#
#  Regression cover for the reciprocal allocation/agreement teardown, where
#  agreement teardown destroyed its allocation and allocation teardown settled
#  the agreement back, paying the same agreement twice out of a pool shared by
#  all of that provider's agreements.
#
#  The phase above cannot see any of that: it runs a single agreement with both
#  penalties at zero, so the duplicate payout had no second agreement's
#  collateral to take and simply failed for insufficient funds — an error that
#  used to be discarded. Three things are needed to make it observable, and all
#  three are set up below: two agreements sharing one pool, nonzero penalties,
#  and an agreement left to expire through the EndBlocker.
#
#  Assertions are on the collateral and earnings pools rather than on player
#  balances wherever an exact figure is claimed, because module accounts pay no
#  transaction fees.
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE 16b: Agreement Settlement Accounting"

SETTLE_GUILD_VAULT=$(get_vault_address "${GUILD_ID}")
SETTLE_OWNER_VAULT=$(get_vault_address "${PLAYER_1_ID}")
assert_not_empty "Settlement guild vault resolved" "${SETTLE_GUILD_VAULT}"
assert_not_empty "Settlement owner vault resolved" "${SETTLE_OWNER_VAULT}"

# Agreement A + B run concurrently; C is opened after both have closed.
SETTLE_CAP_A=2000
SETTLE_CAP_B=3000
SETTLE_CAP_C=2000
SETTLE_DUR_A=$((120 * FIXTURE_DURATION_MULTIPLIER))
SETTLE_DUR_B=$((120 * FIXTURE_DURATION_MULTIPLIER))
SETTLE_DUR_C=5     # must be >= the provider duration minimum below

# ─── settlement fixture headroom ───
# This shared substation may already be overloaded. Clamp free capacity to zero
# and compare uint64 values outside signed shell arithmetic.
SETTLE_SUB_JSON=$(query query structs substation "${SUBSTATION_ID}")
SETTLE_SUB_HEADROOM=$(echo "${SETTLE_SUB_JSON}" | jq -ce --arg id "${SUBSTATION_ID}" 'select(.Substation.id == $id)' | python3 "${GENERATOR_TEST_DIR}/player_allocatable_capacity.py" --required-capacity "$((SETTLE_CAP_A + SETTLE_CAP_B))")
SETTLE_SUB_FREE=$(echo "${SETTLE_SUB_HEADROOM}" | jq -er '.allocatableCapacity')
SETTLE_SUB_HAS_ROOM=$(echo "${SETTLE_SUB_HEADROOM}" | jq -r '.hasCapacity')
info "Substation ${SUBSTATION_ID} free capacity: ${SETTLE_SUB_FREE}"
assert_eq "Settlement fixture has capacity for both agreements" "true" "${SETTLE_SUB_HAS_ROOM}"
# ─── settlement fixture headroom complete ───

# Missing vaults or insufficient fixture capacity are failures, not successful
# coverage with the settlement cases skipped. Guard execution after recording
# those failures so empty-address balance queries cannot pass vacuously.
assert_not_empty "Settlement player 2 vault resolved" "${PLAYER_2_VAULT}"
assert_not_empty "Settlement player 3 vault resolved" "${PLAYER_3_VAULT}"
if [ -z "${PLAYER_2_VAULT}" ] || [ -z "${PLAYER_3_VAULT}" ]; then
    info "SKIP: player 2/3 addresses unresolved"
elif [ "${SETTLE_SUB_HAS_ROOM}" != true ]; then
    info "SKIP: substation has ${SETTLE_SUB_FREE} free capacity, need $((SETTLE_CAP_A + SETTLE_CAP_B))"
else

# ─── A provider with nonzero penalties, so the penalty payout paths execute ───
SETTLE_RATE="1ualpha"
SETTLE_PROVIDER_PENALTY="0.5"
SETTLE_CONSUMER_PENALTY="0.25"

run_tx "Creating guild-owned provider for settlement accounting (penalties ${SETTLE_PROVIDER_PENALTY}/${SETTLE_CONSUMER_PENALTY})" \
    tx structs provider-create "${SUBSTATION_ID}" \
    "${SETTLE_RATE}" "open-market" \
    "${SETTLE_PROVIDER_PENALTY}" "${SETTLE_CONSUMER_PENALTY}" \
    1000 100000 \
    5 100000 \
    --owner-id "${GUILD_ID}" --from alice

SETTLE_PROV_ID=$(get_created_provider_id "${GUILD_ID}" "${SUBSTATION_ID}")
assert_not_empty "Settlement provider created" "${SETTLE_PROV_ID}"

if [ -z "${SETTLE_PROV_ID}" ]; then
    info "SKIP: could not create settlement provider"
else

# ─── settlement provider terms and pools ───
SETTLE_PROV_JSON=$(query query structs provider "${SETTLE_PROV_ID}")
SETTLE_EXPECTED_TERMS=$(jq -cnS --arg id "${SETTLE_PROV_ID}" --arg owner "${GUILD_ID}" --arg station "${SUBSTATION_ID}" '{id:$id, owner:$owner, substationId:$station, rate:{denom:"ualpha",amount:"1"}, accessPolicy:"openMarket", capacityMinimum:"1000", capacityMaximum:"100000", durationMinimum:"5", durationMaximum:"100000", providerCancellationPenalty:"0.500000000000000000", consumerCancellationPenalty:"0.250000000000000000"}')
SETTLE_ACTUAL_TERMS=$(echo "${SETTLE_PROV_JSON}" | jq -ceS '.Provider | select(type == "object") | if has("accessPolicy") then . else .accessPolicy="openMarket" end | {id, owner, substationId, rate, accessPolicy, capacityMinimum, capacityMaximum, durationMinimum, durationMaximum, providerCancellationPenalty, consumerCancellationPenalty}')
assert_eq "Settlement provider terms match accounting assumptions" "${SETTLE_EXPECTED_TERMS}" "${SETTLE_ACTUAL_TERMS}"

SETTLE_COLL_ADDR=$(query query structs provider-collateral-address "${SETTLE_PROV_ID}" | jq -er --arg id "${SETTLE_PROV_ID}" '.internalAddressAssociation | select(length == 1 and .[0].objectId == $id) | .[0].address | select(type == "string" and length > 0)')
SETTLE_EARN_ADDR=$(query query structs provider-earnings-address "${SETTLE_PROV_ID}" | jq -er --arg id "${SETTLE_PROV_ID}" '.internalAddressAssociation | select(length == 1 and .[0].objectId == $id) | .[0].address | select(type == "string" and length > 0)')
assert_eq "Settlement collateral and earnings pools are distinct" "false" "$([ "${SETTLE_COLL_ADDR}" = "${SETTLE_EARN_ADDR}" ] && echo true || echo false)"
# ─── settlement provider terms and pools complete ───
assert_not_empty "Settlement collateral pool address" "${SETTLE_COLL_ADDR}"
assert_not_empty "Settlement earnings pool address" "${SETTLE_EARN_ADDR}"

if [ -z "${SETTLE_COLL_ADDR}" ] || [ -z "${SETTLE_EARN_ADDR}" ]; then
    info "SKIP: could not resolve settlement provider pool addresses"
else

# A brand new provider, so both pools start empty and every later figure is a
# delta from a known zero.
assert_eq "Fresh collateral pool is empty" "0" "$(get_balance "${SETTLE_COLL_ADDR}" ualpha)"
assert_eq "Fresh earnings pool is empty" "0" "$(get_balance "${SETTLE_EARN_ADDR}" ualpha)"

# ─── Agreement A (player_2) ───
# Collateral is duration * capacity * rate, taken from the message parameters
# alone, so it does not drift with block height.
SETTLE_COLL_A=$((SETTLE_DUR_A * SETTLE_CAP_A))
fund_vault_alpha "${PLAYER_2_ID}" "${SETTLE_COLL_A}"
run_tx "Player 2 opening settlement agreement A (dur=${SETTLE_DUR_A}, cap=${SETTLE_CAP_A})" \
    tx structs agreement-open "${SETTLE_PROV_ID}" "${SETTLE_DUR_A}" "${SETTLE_CAP_A}" --from player_2

SETTLE_A_RECORD=$(get_opened_agreement_record "${SETTLE_PROV_ID}" "${PLAYER_2_ID}" "${SETTLE_CAP_A}")
SETTLE_AGREE_A=$(echo "${SETTLE_A_RECORD}" | jq -er '.id')
SETTLE_ALLOC_A=$(echo "${SETTLE_A_RECORD}" | jq -er '.allocationId | select(type == "string" and test("^6-[0-9]+$"))')
assert_not_empty "Settlement agreement A opened" "${SETTLE_AGREE_A}"
# ─── opening A snapshot conservation ───
# EndBlock can already have moved earned revenue. Read both pools at the same
# committed height; the original deposit must still be present across them.
SETTLE_A_HEIGHT=$(get_block_height)
SETTLE_A_POOL=$(get_balance_at_height "${SETTLE_COLL_ADDR}" ualpha "${SETTLE_A_HEIGHT}")
SETTLE_A_EARN=$(get_balance_at_height "${SETTLE_EARN_ADDR}" ualpha "${SETTLE_A_HEIGHT}")
SETTLE_A_TERMS=$(printf '%s' "${SETTLE_A_RECORD}" | jq -ce --arg duration "${SETTLE_DUR_A}" --arg capacity "${SETTLE_CAP_A}" '[{start:.startBlock,duration:$duration,capacity:$capacity}]')
SETTLE_A_SNAPSHOT=$(python3 "${GENERATOR_TEST_DIR}/settlement_open_snapshot.py" "${SETTLE_A_HEIGHT}" "${SETTLE_A_POOL}" "${SETTLE_A_EARN}" "${SETTLE_A_TERMS}")
assert_eq "Agreement A deposit and unpaid holdback remain covered" true "$(printf '%s' "${SETTLE_A_SNAPSHOT}" | jq -r '.verified')"
# ─── opening A snapshot conservation complete ───

# ─── Agreement B (player_3), sharing the same pool ───
# Player 3's liquid balance is often near zero by this point (delegations and
# earlier spends). Fund the intended fixture before opening; agreement discovery
# below requires a matching event from this specific delivered transaction.
SETTLE_COLL_B=$((SETTLE_DUR_B * SETTLE_CAP_B))
SETTLE_P3_NEED=$((SETTLE_COLL_B + 100000))
fund_vault_alpha "${PLAYER_3_ID}" "${SETTLE_P3_NEED}"

run_tx "Player 3 opening settlement agreement B (dur=${SETTLE_DUR_B}, cap=${SETTLE_CAP_B})" \
    tx structs agreement-open "${SETTLE_PROV_ID}" "${SETTLE_DUR_B}" "${SETTLE_CAP_B}" --from player_3

SETTLE_B_RECORD=$(get_opened_agreement_record "${SETTLE_PROV_ID}" "${PLAYER_3_ID}" "${SETTLE_CAP_B}")
SETTLE_AGREE_B=$(echo "${SETTLE_B_RECORD}" | jq -er '.id')
SETTLE_ALLOC_B=$(echo "${SETTLE_B_RECORD}" | jq -er '.allocationId | select(type == "string" and test("^6-[0-9]+$"))')
assert_not_empty "Settlement agreement B opened" "${SETTLE_AGREE_B}"

if [ -z "${SETTLE_AGREE_B}" ]; then
    info "SKIP: settlement agreement B did not open; dual-agreement assertions skipped"
    # Close A so the expiry path below starts from a clean provider load.
    if [ -n "${SETTLE_AGREE_A}" ]; then
        run_tx "Closing settlement agreement A after B open failed" \
            tx structs agreement-close "${SETTLE_AGREE_A}" --from player_2
    fi
else

assert_eq "Agreement B is distinct from A" "false" "$([ "${SETTLE_AGREE_A}" = "${SETTLE_AGREE_B}" ] && echo true || echo false)"

# ─── opening B snapshot conservation ───
# Opening B checkpoints A's elapsed earnings. Pin both pool balances to one
# committed height so an expiry between RPCs cannot be counted twice or disappear.
# Their sum must still conserve both deposits.
SETTLE_BOTH_HEIGHT=$(get_block_height)
SETTLE_POOL_BOTH=$(get_balance_at_height "${SETTLE_COLL_ADDR}" ualpha "${SETTLE_BOTH_HEIGHT}")
SETTLE_EARN_BOTH=$(get_balance_at_height "${SETTLE_EARN_ADDR}" ualpha "${SETTLE_BOTH_HEIGHT}")
info "Pool with both agreements open: ${SETTLE_POOL_BOTH} + earnings ${SETTLE_EARN_BOTH} (A=${SETTLE_COLL_A}, B=${SETTLE_COLL_B})"
SETTLE_BOTH_TERMS=$(printf '%s' "${SETTLE_B_RECORD}" | jq -ce --argjson a "${SETTLE_A_TERMS}" --arg duration "${SETTLE_DUR_B}" --arg capacity "${SETTLE_CAP_B}" '$a + [{start:.startBlock,duration:$duration,capacity:$capacity}]')
SETTLE_BOTH_SNAPSHOT=$(python3 "${GENERATOR_TEST_DIR}/settlement_open_snapshot.py" "${SETTLE_BOTH_HEIGHT}" "${SETTLE_POOL_BOTH}" "${SETTLE_EARN_BOTH}" "${SETTLE_BOTH_TERMS}")
assert_eq "Both deposits and unpaid holdbacks remain covered" true "$(printf '%s' "${SETTLE_BOTH_SNAPSHOT}" | jq -r '.verified')"
# ─── opening B snapshot conservation complete ───

SETTLE_LOAD_BOTH=$(get_provider_load "${SETTLE_PROV_ID}")
assert_eq "Provider load is both capacities" "$((SETTLE_CAP_A + SETTLE_CAP_B))" "${SETTLE_LOAD_BOTH}"

# ─── Close A, and check it did not eat into B ───
# This is the assertion the whole section exists for. The double settlement paid
# A out twice, and the second payout could only come from B's collateral.
SETTLE_B_JSON=$(query query structs agreement "${SETTLE_AGREE_B}")
SETTLE_B_BEFORE_A=$(echo "${SETTLE_B_JSON}" | jq -ceS --arg id "${SETTLE_AGREE_B}" --arg allocation "${SETTLE_ALLOC_B}" '.Agreement | select(type == "object" and .id == $id and .allocationId == $allocation)')
SETTLE_B_ALLOC_BEFORE_A=$(query query structs allocation "${SETTLE_ALLOC_B}" | jq -ceS --arg id "${SETTLE_ALLOC_B}" 'select(.Allocation.id == $id and (.gridAttributes | type == "object")) | {Allocation, gridAttributes}')
SETTLE_B_END=$(echo "${SETTLE_B_JSON}" | jq -er '.Agreement.endBlock | select(type == "string")')

SETTLE_A_END=$(query query structs agreement "${SETTLE_AGREE_A}" | jq -er '.Agreement.endBlock | select(type == "string")')
P2_BEFORE_CLOSE=$(get_balance "${PLAYER_2_VAULT}" ualpha)

run_tx "Closing settlement agreement A" \
    tx structs agreement-close "${SETTLE_AGREE_A}" --from player_2

assert_object_missing "Settlement agreement A removed" agreement "${SETTLE_AGREE_A}"
assert_object_missing "Settlement agreement A allocation removed" allocation "${SETTLE_ALLOC_A}"
SETTLE_B_AFTER_A=$(query query structs agreement "${SETTLE_AGREE_B}" | jq -ceS --arg id "${SETTLE_AGREE_B}" '.Agreement | select(type == "object" and .id == $id)')
SETTLE_B_ALLOC_AFTER_A=$(query query structs allocation "${SETTLE_ALLOC_B}" | jq -ceS --arg id "${SETTLE_ALLOC_B}" 'select(.Allocation.id == $id and (.gridAttributes | type == "object")) | {Allocation, gridAttributes}')
assert_eq "Closing A preserves agreement B's complete record" "${SETTLE_B_BEFORE_A}" "${SETTLE_B_AFTER_A}"
assert_eq "Closing A preserves B's allocation and power" "${SETTLE_B_ALLOC_BEFORE_A}" "${SETTLE_B_ALLOC_AFTER_A}"

# The refund uses the delivered close block; the vault pays no transaction fees.
# Check the exact entitlement as well as the shared-pool coverage below.
P2_EXPECTED_AFTER_CLOSE=$(expected_settlement_consumer_vault "${P2_BEFORE_CLOSE}" "${SETTLE_A_END}" "${SETTLE_CAP_A}")
P2_AFTER_CLOSE=$(get_balance "${PLAYER_2_VAULT}" ualpha)
assert_eq "Player 2 receives exact refund after consumer penalty" "${P2_EXPECTED_AFTER_CLOSE}" "${P2_AFTER_CLOSE}"

# B is still owed unearned collateral plus the provider cancellation holdback.
# Unearned collateral alone is the conservative floor. The pool and the service
# height used to price that floor must refer to the same committed block.
SETTLE_HEIGHT=$(get_block_height)
SETTLE_POOL_AFTER_A=$(get_balance_at_height "${SETTLE_COLL_ADDR}" ualpha "${SETTLE_HEIGHT}")
SETTLE_B_COVERAGE=$(python3 -c 'import json, re, sys
assert all(re.fullmatch(r"0|[1-9][0-9]*", value) for value in sys.argv[1:]), "invalid settlement amount or height"
end, height, capacity, pool = map(int, sys.argv[1:])
assert 0 < height <= 2**63-1 and height < end < 2**64, "B must still be live for the early-close fixture"
assert 0 < capacity < 2**64
unearned = (end-height)*capacity
print(json.dumps({"unearned": str(unearned), "covered": pool >= unearned}))' "${SETTLE_B_END}" "${SETTLE_HEIGHT}" "${SETTLE_CAP_B}" "${SETTLE_POOL_AFTER_A}")
SETTLE_B_UNEARNED=$(echo "${SETTLE_B_COVERAGE}" | jq -er '.unearned')
info "Pool after closing A: ${SETTLE_POOL_AFTER_A}; B unearned collateral at block ${SETTLE_HEIGHT}: ${SETTLE_B_UNEARNED}"
assert_eq "Pool still covers agreement B after A settled" "true" "$(echo "${SETTLE_B_COVERAGE}" | jq -r '.covered')"

SETTLE_LOAD_AFTER_A=$(get_provider_load "${SETTLE_PROV_ID}")
assert_eq "Only agreement A's load was released" "${SETTLE_CAP_B}" "${SETTLE_LOAD_AFTER_A}"

# ─── Close B, and check its consumer was still paid ───
P3_BEFORE_CLOSE=$(get_balance "${PLAYER_3_VAULT}" ualpha)

run_tx "Closing settlement agreement B" \
    tx structs agreement-close "${SETTLE_AGREE_B}" --from player_3

assert_object_missing "Settlement agreement B removed" agreement "${SETTLE_AGREE_B}"
assert_object_missing "Settlement agreement B allocation removed" allocation "${SETTLE_ALLOC_B}"
P3_EXPECTED_AFTER_CLOSE=$(expected_settlement_consumer_vault "${P3_BEFORE_CLOSE}" "${SETTLE_B_END}" "${SETTLE_CAP_B}")
P3_AFTER_CLOSE=$(get_balance "${PLAYER_3_VAULT}" ualpha)
assert_eq "Player 3 receives exact refund without funding A's teardown" "${P3_EXPECTED_AFTER_CLOSE}" "${P3_AFTER_CLOSE}"

SETTLE_LOAD_EMPTY=$(get_provider_load "${SETTLE_PROV_ID}")
assert_eq "Provider load back to zero after both settled" "0" "${SETTLE_LOAD_EMPTY}"

# ─── dual-agreement exact conservation ───
# Both vaults pay no transaction fees. Count their actual refunds alongside both
# remaining pools, so a lost coin cannot hide under the old conservation ceiling.
SETTLE_POOL_END=$(get_balance "${SETTLE_COLL_ADDR}" ualpha)
SETTLE_EARN_END=$(get_balance "${SETTLE_EARN_ADDR}" ualpha)
SETTLE_CONSERVED=$(python3 -c 'import sys
p2_before, p2_after, p3_before, p3_after, collateral, earnings, deposit_a, deposit_b = map(int, sys.argv[1:])
assert min(p2_before, p3_before, collateral, earnings, deposit_a, deposit_b) >= 0
refund_a, refund_b = p2_after-p2_before, p3_after-p3_before
assert min(refund_a, refund_b) >= 0
print(str(refund_a+refund_b+collateral+earnings == deposit_a+deposit_b).lower())' "${P2_BEFORE_CLOSE}" "${P2_AFTER_CLOSE}" "${P3_BEFORE_CLOSE}" "${P3_AFTER_CLOSE}" "${SETTLE_POOL_END}" "${SETTLE_EARN_END}" "${SETTLE_COLL_A}" "${SETTLE_COLL_B}")
assert_eq "Both deposits exactly reconcile with refunds and provider pools" "true" "${SETTLE_CONSERVED}"
# ─── dual-agreement exact conservation complete ───

# ─── dual-agreement earned revenue ───
# Both capacities are divisible by four, so neither the half-rate checkpoint
# nor quarter-rate cancellation penalty leaves fractional-coin dust here.
SETTLE_EXPECTED_EARNINGS=$(python3 -c 'import sys
a, b, before_a, expected_a, before_b, expected_b = map(int, sys.argv[1:])
print(a+b-(expected_a-before_a)-(expected_b-before_b))' "${SETTLE_COLL_A}" "${SETTLE_COLL_B}" "${P2_BEFORE_CLOSE}" "${P2_EXPECTED_AFTER_CLOSE}" "${P3_BEFORE_CLOSE}" "${P3_EXPECTED_AFTER_CLOSE}")
assert_eq "Settled provider receives all earned service and cancellation revenue" "${SETTLE_EXPECTED_EARNINGS}" "${SETTLE_EARN_END}"
assert_eq "No earned provider holdback remains in collateral" "0" "${SETTLE_POOL_END}"
# ─── dual-agreement earned revenue complete ───

# Both agreements are closed and settled, so their load no longer accrues earnings. The default recipient must be the guild vault, even though the
# human owner signs and pays transaction fees from a separate address.
SETTLE_TREASURY_BEFORE_WITHDRAW=$(get_balance "${SETTLE_GUILD_VAULT}" ualpha)
SETTLE_OWNER_BEFORE_WITHDRAW=$(get_balance "${SETTLE_OWNER_VAULT}" ualpha)
SETTLE_MEMBER_BEFORE_WITHDRAW=$(get_balance "${PLAYER_2_VAULT}" ualpha)
SETTLE_TREASURY_EXPECTED_WITHDRAW=$(python3 -c 'import sys; vault, earnings = map(int, sys.argv[1:]); assert min(vault, earnings) >= 0; print(vault+earnings)' "${SETTLE_TREASURY_BEFORE_WITHDRAW}" "${SETTLE_EARN_END}")
run_tx_expect_permission_denied "Membership alone cannot withdraw guild provider earnings" \
    tx structs provider-withdraw-balance "${SETTLE_PROV_ID}" --from player_2
assert_eq "Rejected withdrawal preserves guild provider earnings" "${SETTLE_EARN_END}" "$(get_balance "${SETTLE_EARN_ADDR}" ualpha)"
assert_eq "Rejected withdrawal preserves guild collateral" "${SETTLE_POOL_END}" "$(get_balance "${SETTLE_COLL_ADDR}" ualpha)"
assert_eq "Rejected withdrawal preserves guild vault" "${SETTLE_TREASURY_BEFORE_WITHDRAW}" "$(get_balance "${SETTLE_GUILD_VAULT}" ualpha)"
assert_eq "Rejected withdrawal preserves owner vault" "${SETTLE_OWNER_BEFORE_WITHDRAW}" "$(get_balance "${SETTLE_OWNER_VAULT}" ualpha)"
assert_eq "Rejected withdrawal preserves member vault" "${SETTLE_MEMBER_BEFORE_WITHDRAW}" "$(get_balance "${PLAYER_2_VAULT}" ualpha)"
run_tx "Withdrawing guild provider earnings to its default vault" \
    tx structs provider-withdraw-balance "${SETTLE_PROV_ID}" --from alice
assert_eq "Guild vault receives exact provider earnings" "${SETTLE_TREASURY_EXPECTED_WITHDRAW}" "$(get_balance "${SETTLE_GUILD_VAULT}" ualpha)"
assert_eq "Provider withdrawal does not credit the owner's personal vault" "${SETTLE_OWNER_BEFORE_WITHDRAW}" "$(get_balance "${SETTLE_OWNER_VAULT}" ualpha)"
assert_eq "Guild provider earnings pool is drained" "0" "$(get_balance "${SETTLE_EARN_ADDR}" ualpha)"
assert_eq "Provider withdrawal preserves guild collateral" "${SETTLE_POOL_END}" "$(get_balance "${SETTLE_COLL_ADDR}" ualpha)"
assert_eq "Provider withdrawal preserves member vault" "${SETTLE_MEMBER_BEFORE_WITHDRAW}" "$(get_balance "${PLAYER_2_VAULT}" ualpha)"

fi # settlement agreement B opened

# ─── Expiry through the EndBlocker ───
# The worst case of the original bug. Over an agreement's life the checkpoints
# and the voided penalty payout together consume the whole deposit, and the
# re-entrant settlement then paid the consumer a provider-cancellation penalty
# out of a share that was already empty.
SETTLE_COLL_C=$((SETTLE_DUR_C * SETTLE_CAP_C))
fund_vault_alpha "${PLAYER_2_ID}" "${SETTLE_COLL_C}"
SETTLE_POOL_PRE_C=$(get_balance "${SETTLE_COLL_ADDR}" ualpha)
SETTLE_EARN_PRE_C=$(get_balance "${SETTLE_EARN_ADDR}" ualpha)
SETTLE_EARN_EXPECTED_C=$(python3 -c 'import sys; earnings, collateral = map(int, sys.argv[1:]); assert earnings >= 0 and collateral >= 0; print(earnings+collateral)' "${SETTLE_EARN_PRE_C}" "${SETTLE_COLL_C}")
# Establish the no-refund balance before delivery can reach the expiry block.
P2_PRE_C=$(get_balance "${PLAYER_2_VAULT}" ualpha)
P2_EXPECTED_POST_C=$(python3 -c 'import sys; balance, collateral = map(int, sys.argv[1:]); assert balance >= collateral >= 0; print(balance-collateral)' "${P2_PRE_C}" "${SETTLE_COLL_C}")

run_tx "Player 2 opening settlement agreement C to expire (dur=${SETTLE_DUR_C}, cap=${SETTLE_CAP_C})" \
    tx structs agreement-open "${SETTLE_PROV_ID}" "${SETTLE_DUR_C}" "${SETTLE_CAP_C}" --from player_2

SETTLE_C_RECORD=$(get_opened_agreement_record "${SETTLE_PROV_ID}" "${PLAYER_2_ID}" "${SETTLE_CAP_C}")
SETTLE_AGREE_C=$(echo "${SETTLE_C_RECORD}" | jq -er '.id')
SETTLE_ALLOC_C=$(echo "${SETTLE_C_RECORD}" | jq -er '.allocationId | select(type == "string" and test("^6-[0-9]+$"))')
assert_not_empty "Settlement agreement C opened" "${SETTLE_AGREE_C}"

SETTLE_C_EXPIRY=$(echo "${SETTLE_C_RECORD}" | jq -c '{Agreement: .}' | python3 "${GENERATOR_TEST_DIR}/agreement_expiry_height.py" "${SETTLE_AGREE_C}")
SETTLE_C_END=$(echo "${SETTLE_C_EXPIRY}" | jq -er '.endBlock')
SETTLE_C_WAIT=$(echo "${SETTLE_C_EXPIRY}" | jq -er '.waitHeight')
info "Agreement C endBlock: ${SETTLE_C_END}"

# Expiry fires in the EndBlocker of the end block itself; wait past it.
if wait_for_block "${SETTLE_C_WAIT}" 120; then
    assert_object_missing "Expired agreement removed by EndBlocker" agreement "${SETTLE_AGREE_C}"
    assert_object_missing "Expired agreement C allocation is removed" allocation "${SETTLE_ALLOC_C}"

    # An expired agreement owes the consumer nothing: they received the service
    # they paid for, and the provider keeps the cancellation penalty because they
    # did not cancel. Player 2 sends no transaction here, so this figure is exact.
    assert_eq "Expiry pays the consumer nothing" "${P2_EXPECTED_POST_C}" "$(get_balance "${PLAYER_2_VAULT}" ualpha)"

    SETTLE_EARN_POST_C=$(get_balance "${SETTLE_EARN_ADDR}" ualpha)
    SETTLE_POOL_POST_C=$(get_balance "${SETTLE_COLL_ADDR}" ualpha)
    info "After expiry: pool=${SETTLE_POOL_POST_C} (was ${SETTLE_POOL_PRE_C}), earnings=${SETTLE_EARN_POST_C} (was ${SETTLE_EARN_PRE_C}), C collateral=${SETTLE_COLL_C}"

    # The whole deposit becomes provider revenue, split between checkpointed
    # earnings and the penalty they keep.
    assert_eq "Expired collateral became provider earnings" "${SETTLE_EARN_EXPECTED_C}" "${SETTLE_EARN_POST_C}"
    assert_eq "Pool back to its pre-agreement dust after expiry" "${SETTLE_POOL_PRE_C}" "${SETTLE_POOL_POST_C}"

    SETTLE_LOAD_POST_C=$(get_provider_load "${SETTLE_PROV_ID}")
    assert_eq "Provider load released exactly once on expiry" "0" "${SETTLE_LOAD_POST_C}"
else
    assert_eq "Chain reaches settlement agreement C expiry height" "true" "false"
fi

# ─── Guild-owned consumer agreement ───
SETTLE_DUR_D=20
SETTLE_CAP_D=2000
SETTLE_COLL_D=$((SETTLE_DUR_D * SETTLE_CAP_D))
fund_vault_alpha "${GUILD_ID}" "${SETTLE_COLL_D}"
SETTLE_TREASURY_BEFORE_D=$(get_balance "${SETTLE_GUILD_VAULT}" ualpha)
SETTLE_OWNER_BEFORE_D=$(get_balance "${SETTLE_OWNER_VAULT}" ualpha)
SETTLE_EARN_BEFORE_D=$(get_balance "${SETTLE_EARN_ADDR}" ualpha)
SETTLE_POOL_BEFORE_D=$(get_balance "${SETTLE_COLL_ADDR}" ualpha)
SETTLE_TREASURY_EXPECTED_D=$(python3 -c 'import sys; balance, collateral = map(int, sys.argv[1:]); assert balance >= collateral >= 0; print(balance-collateral)' "${SETTLE_TREASURY_BEFORE_D}" "${SETTLE_COLL_D}")
SETTLE_EARN_EXPECTED_D=$(python3 -c 'import sys; earnings, collateral = map(int, sys.argv[1:]); assert earnings >= 0 and collateral >= 0; print(earnings+collateral)' "${SETTLE_EARN_BEFORE_D}" "${SETTLE_COLL_D}")
run_tx "Guild opens its own consumer agreement" \
    tx structs agreement-open "${SETTLE_PROV_ID}" "${SETTLE_DUR_D}" "${SETTLE_CAP_D}" --owner-id "${GUILD_ID}" --from alice
SETTLE_D_RECORD=$(get_opened_agreement_record "${SETTLE_PROV_ID}" "${GUILD_ID}" "${SETTLE_CAP_D}")
SETTLE_AGREE_D=$(echo "${SETTLE_D_RECORD}" | jq -er '.id')
SETTLE_ALLOC_D=$(echo "${SETTLE_D_RECORD}" | jq -er '.allocationId | select(type == "string" and test("^6-[0-9]+$"))')
assert_not_empty "Guild consumer agreement created" "${SETTLE_AGREE_D}"
SETTLE_D_JSON=$(echo "${SETTLE_D_RECORD}" | jq -c '{Agreement: .}')
assert_eq "Agreement consumer is the guild" "${GUILD_ID}" "$(jqr "${SETTLE_D_JSON}" '.Agreement.owner' '')"
assert_eq "Guild agreement collateral debits the guild vault" "${SETTLE_TREASURY_EXPECTED_D}" "$(get_balance "${SETTLE_GUILD_VAULT}" ualpha)"
assert_eq "Guild agreement leaves the signing owner's personal vault unchanged" "${SETTLE_OWNER_BEFORE_D}" "$(get_balance "${SETTLE_OWNER_VAULT}" ualpha)"
SETTLE_D_EXPIRY=$(echo "${SETTLE_D_JSON}" | python3 "${GENERATOR_TEST_DIR}/agreement_expiry_height.py" "${SETTLE_AGREE_D}")
SETTLE_D_END=$(echo "${SETTLE_D_EXPIRY}" | jq -er '.endBlock')
SETTLE_D_WAIT=$(echo "${SETTLE_D_EXPIRY}" | jq -er '.waitHeight')
if wait_for_block "${SETTLE_D_WAIT}" 120; then
    assert_object_missing "Expired guild agreement is removed" agreement "${SETTLE_AGREE_D}"
    assert_object_missing "Expired agreement D allocation is removed" allocation "${SETTLE_ALLOC_D}"
    assert_eq "Guild consumer receives no refund for fully served agreement" "${SETTLE_TREASURY_EXPECTED_D}" "$(get_balance "${SETTLE_GUILD_VAULT}" ualpha)"
    assert_eq "Guild agreement deposit becomes provider earnings" "${SETTLE_EARN_EXPECTED_D}" "$(get_balance "${SETTLE_EARN_ADDR}" ualpha)"
    assert_eq "Guild expiry leaves the signing owner's personal vault unchanged" "${SETTLE_OWNER_BEFORE_D}" "$(get_balance "${SETTLE_OWNER_VAULT}" ualpha)"
    assert_eq "Guild agreement releases its collateral pool balance" "${SETTLE_POOL_BEFORE_D}" "$(get_balance "${SETTLE_COLL_ADDR}" ualpha)"
    assert_eq "Guild agreement releases provider load" "0" "$(get_provider_load "${SETTLE_PROV_ID}")"
else
    assert_eq "Chain reaches guild agreement expiry height" "true" "false"
fi

# ─── Clean up ───
SETTLE_TREASURY_BEFORE_DELETE=$(get_balance "${SETTLE_GUILD_VAULT}" ualpha)
SETTLE_OWNER_BEFORE_DELETE=$(get_balance "${SETTLE_OWNER_VAULT}" ualpha)
SETTLE_COLL_BEFORE_DELETE=$(get_balance "${SETTLE_COLL_ADDR}" ualpha)
SETTLE_EARN_BEFORE_DELETE=$(get_balance "${SETTLE_EARN_ADDR}" ualpha)
SETTLE_TREASURY_EXPECTED_DELETE=$(python3 -c 'import sys; vault, collateral, earnings = map(int, sys.argv[1:]); assert min(vault, collateral, earnings) >= 0; print(vault+collateral+earnings)' "${SETTLE_TREASURY_BEFORE_DELETE}" "${SETTLE_COLL_BEFORE_DELETE}" "${SETTLE_EARN_BEFORE_DELETE}")
run_tx "Deleting settlement provider" \
    tx structs provider-delete "${SETTLE_PROV_ID}" --from alice

assert_object_missing "Settlement provider deleted" provider "${SETTLE_PROV_ID}"
assert_eq "Provider deletion sends all remaining pools to guild vault" "${SETTLE_TREASURY_EXPECTED_DELETE}" "$(get_balance "${SETTLE_GUILD_VAULT}" ualpha)"
assert_eq "Provider deletion leaves owner's personal vault unchanged" "${SETTLE_OWNER_BEFORE_DELETE}" "$(get_balance "${SETTLE_OWNER_VAULT}" ualpha)"
assert_eq "Deleted provider collateral pool is empty" "0" "$(get_balance "${SETTLE_COLL_ADDR}" ualpha)"
assert_eq "Deleted provider earnings pool is empty" "0" "$(get_balance "${SETTLE_EARN_ADDR}" ualpha)"

fi # settlement pool addresses
fi # settlement provider
fi # settlement substation capacity

# ─── pending provider lifecycle ───
section "PHASE 16c: Withdrawal and Deletion With Live Contracts"

PENDING_HEADROOM=$(query query structs substation "${SUBSTATION_ID}" | jq -ce --arg id "${SUBSTATION_ID}" 'select(.Substation.id == $id)' | python3 "${GENERATOR_TEST_DIR}/player_allocatable_capacity.py" --required-capacity 5000)
printf '%s' "${PENDING_HEADROOM}" | jq -e '.hasCapacity == true' >/dev/null
run_tx "Creating guild provider for pending-contract transitions" \
    tx structs provider-create "${SUBSTATION_ID}" 1ualpha open-market 0.5 0.25 1000 100000 5 100000 --owner-id "${GUILD_ID}" --from alice
PENDING_PROVIDER=$(get_created_provider_id "${GUILD_ID}" "${SUBSTATION_ID}")
PENDING_PROVIDER_JSON=$(query query structs provider "${PENDING_PROVIDER}")
printf '%s' "${PENDING_PROVIDER_JSON}" | jq -e --arg id "${PENDING_PROVIDER}" --arg owner "${GUILD_ID}" --arg station "${SUBSTATION_ID}" '.Provider | .id == $id and .owner == $owner and .substationId == $station and .rate == {denom:"ualpha",amount:"1"} and .providerCancellationPenalty == "0.500000000000000000" and .consumerCancellationPenalty == "0.250000000000000000"' >/dev/null
PENDING_COLLATERAL=$(query query structs provider-collateral-address "${PENDING_PROVIDER}" | jq -er --arg id "${PENDING_PROVIDER}" '.internalAddressAssociation | select(length == 1 and .[0].objectId == $id) | .[0].address | select(type == "string" and length > 0)')
PENDING_EARNINGS=$(query query structs provider-earnings-address "${PENDING_PROVIDER}" | jq -er --arg id "${PENDING_PROVIDER}" '.internalAddressAssociation | select(length == 1 and .[0].objectId == $id) | .[0].address | select(type == "string" and length > 0)')
PENDING_GUILD_VAULT=$(get_vault_address "${GUILD_ID}")
PENDING_PERSONAL_VAULT=$(get_vault_address "${PLAYER_1_ID}")
assert_eq "Fresh pending provider has no collateral" 0 "$(get_balance "${PENDING_COLLATERAL}" ualpha)"
assert_eq "Fresh pending provider has no earnings" 0 "$(get_balance "${PENDING_EARNINGS}" ualpha)"
PENDING_STATE=$(jq -cn --arg provider "${PENDING_PROVIDER}" --arg owner "${GUILD_ID}" --arg source "${SUBSTATION_ID}" --arg vault "${PENDING_GUILD_VAULT}" --arg personal "${PENDING_PERSONAL_VAULT}" --arg collateral "${PENDING_COLLATERAL}" --arg earnings "${PENDING_EARNINGS}" --arg balance "$(get_balance "${PENDING_GUILD_VAULT}" ualpha)" --arg personal_balance "$(get_balance "${PENDING_PERSONAL_VAULT}" ualpha)" '{provider_id:$provider,provider_owner:$owner,source_id:$source,owner_vault:$vault,personal_vault:$personal,collateral_address:$collateral,earnings_address:$earnings,initial_owner_balance:$balance,initial_personal_balance:$personal_balance,agreements:[]}')

# A long service window leaves time for receipt queries; the oracle still
# refuses a withdrawal/deletion that reaches expiry instead of this live path.
fund_vault_alpha "${PLAYER_2_ID}" 2000000
PENDING_OPEN=$(jq -cn --arg owner "${PLAYER_2_ID}" --arg vault "${PLAYER_2_VAULT}" --arg balance "$(get_balance "${PLAYER_2_VAULT}" ualpha)" '{consumer_owner:$owner,consumer_vault:$vault,initial_balance:$balance,capacity:"2000",duration:"1000"}')
run_tx "Opening first contract for pending provider transitions" \
    tx structs agreement-open "${PENDING_PROVIDER}" 1000 2000 --from player_2
PENDING_STATE=$(record_provider_lifecycle_step open "${PENDING_OPEN}" "${PENDING_STATE}")
fund_vault_alpha "${PLAYER_3_ID}" 3000000
PENDING_OPEN=$(jq -cn --arg owner "${PLAYER_3_ID}" --arg vault "${PLAYER_3_VAULT}" --arg balance "$(get_balance "${PLAYER_3_VAULT}" ualpha)" '{consumer_owner:$owner,consumer_vault:$vault,initial_balance:$balance,capacity:"3000",duration:"1000"}')
run_tx "Opening second contract for pending provider transitions" \
    tx structs agreement-open "${PENDING_PROVIDER}" 1000 3000 --from player_3
PENDING_STATE=$(record_provider_lifecycle_step open "${PENDING_OPEN}" "${PENDING_STATE}")
PENDING_A=$(printf '%s' "${PENDING_STATE}" | jq -er '.agreements[0].agreement_id')
PENDING_B=$(printf '%s' "${PENDING_STATE}" | jq -er '.agreements[1].agreement_id')

# The withdrawal checkpoints elapsed blocks itself. No background billing worker
# or historical claim query is required.
PENDING_NEXT_BLOCK=$(printf '%s' "${PENDING_STATE}" | python3 -c 'import json,sys; print(int(json.load(sys.stdin)["agreements"][1]["start"])+1)')
wait_for_block "${PENDING_NEXT_BLOCK}"
run_tx "Withdrawing earnings while both contracts remain live" \
    tx structs provider-withdraw-balance "${PENDING_PROVIDER}" --from alice
PENDING_STATE=$(record_provider_lifecycle_step withdraw '{}' "${PENDING_STATE}")

# Deletion settles both consumers before draining the remaining pools to the
# guild vault. Confirm exact transaction transfers and resulting balances.
run_tx "Deleting provider while both contracts remain live" \
    tx structs provider-delete "${PENDING_PROVIDER}" --from alice
PENDING_STATE=$(record_provider_lifecycle_step delete '{}' "${PENDING_STATE}")
assert_object_missing "Pending provider physically removed" provider "${PENDING_PROVIDER}"
assert_object_missing "First pending contract physically removed" agreement "${PENDING_A}"
assert_object_missing "Second pending contract physically removed" agreement "${PENDING_B}"
assert_object_missing "First pending allocation physically removed" allocation "$(printf '%s' "${PENDING_STATE}" | jq -er '.agreements[0].allocation_id')"
assert_object_missing "Second pending allocation physically removed" allocation "$(printf '%s' "${PENDING_STATE}" | jq -er '.agreements[1].allocation_id')"
PENDING_BALANCES=$(jq -cn --arg guild "${PENDING_GUILD_VAULT}" --arg personal "${PENDING_PERSONAL_VAULT}" --arg a "${PLAYER_2_VAULT}" --arg b "${PLAYER_3_VAULT}" --arg collateral "${PENDING_COLLATERAL}" --arg earnings "${PENDING_EARNINGS}" --arg gv "$(get_balance "${PENDING_GUILD_VAULT}" ualpha)" --arg pv "$(get_balance "${PENDING_PERSONAL_VAULT}" ualpha)" --arg av "$(get_balance "${PLAYER_2_VAULT}" ualpha)" --arg bv "$(get_balance "${PLAYER_3_VAULT}" ualpha)" --arg cv "$(get_balance "${PENDING_COLLATERAL}" ualpha)" --arg ev "$(get_balance "${PENDING_EARNINGS}" ualpha)" '{($guild):$gv,($personal):$pv,($a):$av,($b):$bv,($collateral):$cv,($earnings):$ev}')
PENDING_VERIFIED=$(printf '%s' "${PENDING_BALANCES}" | python3 "${GENERATOR_TEST_DIR}/provider_lifecycle_accounting.py" verify "${PENDING_STATE}")
assert_eq "Pending withdrawal/deletion pays exact consumer and guild totals" true "$(printf '%s' "${PENDING_VERIFIED}" | jq -r '.verified')"
# ─── pending provider lifecycle complete ───

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


source "${GENERATOR_TEST_DIR}/fleet_fixture.sh"

if run_phase 2300; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 17: Fleet Movement Setup — 5 Fleet Players with Planets
# ═════════════════════════════════════════════════════════════════════════════
#
# Creates 5 dedicated fleet-test players (fplayer_1 through fplayer_5), each
# with a planet, fleet, and command ship. These are separate from the main
# test players to avoid state interactions from earlier phases.
#
# v0.21.0 caps the raid queue at capacity = 1 + locationListExtra (default
# extra=0 => one visitor). Phase 17b exercises that limit and back-and-forth
# occupancy; Phase 17c covers single-visitor combat (home vs one raider).
# Mid-queue adjacency combat is deferred until something can raise
# locationListExtra.

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
        tx bank send "${BOB_ADDRESS}" "${ADDR}" 15000000ualpha --from bob

    run_tx "Delegating for ${FPLAYER_KEY}" \
        tx staking delegate "${VALIDATOR_ADDRESS}" 10000000ualpha --from "${FPLAYER_KEY}"

    ADDR_JSON=$(query query structs address "${ADDR}")
    FP_PID=$(jqr "${ADDR_JSON}" '.playerId')
    eval "FP_${FP_NUM}_ID=${FP_PID}"
    assert_not_empty "Fleet Player ${FP_NUM} ID" "${FP_PID}"
    echo "  Fleet Player ${FP_NUM} ID: ${FP_PID}"

    # Reuse the connected feeder, or finish one left between create/connect.
    FP_ALLOC_ID=$(get_fixture_feeder "${FP_PID}" "${SUBSTATION_ID}" "${PLAYER_1_ID}")
    if [ -z "${FP_ALLOC_ID}" ]; then
        FP_ALLOC_ID=$(get_fixture_feeder "${FP_PID}" "" "${PLAYER_1_ID}")
    fi
    if [ -z "${FP_ALLOC_ID}" ]; then
        PJSON=$(query query structs player "${FP_PID}")
        PCAP=$(printf '%s' "${PJSON}" | python3 "${ALLOCATION_TEST_DIR}/player_allocatable_capacity.py" | jq -er 'select(.hasCapacity == true) | .allocatableCapacity')
        run_tx "Creating allocation for fleet player ${FP_NUM}" \
            tx structs allocation-create "${FP_PID}" "${PCAP}" \
            --controller "${PLAYER_1_ID}" --allocation-type dynamic --from "${FPLAYER_KEY}"
        FP_ALLOC_ID=$(get_created_grid_id allocation "${FP_PID}" "${PLAYER_1_ID}")
    fi
    eval "FP_${FP_NUM}_ALLOC_ID=${FP_ALLOC_ID}"

    PJSON=$(query query structs player "${FP_PID}")
    FP_GUILD=$(printf '%s' "${PJSON}" | jq -er --arg id "${FP_PID}" '.Player | select(.id == $id) | .guildId // ""')
    if [ "${FP_GUILD}" != "${GUILD_ID}" ]; then
        run_tx "Fleet player ${FP_NUM} joining guild" \
            tx structs guild-membership-join "${GUILD_ID}" "${REACTOR_ID}-${ADDR}" --from "${FPLAYER_KEY}"
    fi
    FP_DEST=$(query query structs allocation "${FP_ALLOC_ID}" | jq -er --arg id "${FP_ALLOC_ID}" '.Allocation | select(.id == $id) | .destinationId // ""')
    if [ "${FP_DEST}" != "${SUBSTATION_ID}" ]; then
        run_tx "Connecting fleet player ${FP_NUM} allocation to substation" \
            tx structs substation-allocation-connect "${FP_ALLOC_ID}" "${SUBSTATION_ID}" --from alice
    fi
    assert_eq "FP ${FP_NUM} feeder association" "${FP_ALLOC_ID}" "$(get_fixture_feeder "${FP_PID}" "${SUBSTATION_ID}" "${PLAYER_1_ID}")"
    PJSON=$(query query structs player "${FP_PID}")
    assert_eq "FP ${FP_NUM} guild" "${GUILD_ID}" "$(jqr "${PJSON}" '.Player.guildId')"
    assert_eq "FP ${FP_NUM} substation" "${SUBSTATION_ID}" "$(jqr "${PJSON}" '.Player.substationId')"
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

    PJSON=$(query query structs player "${FP_PID}")
    PLANET_ID=$(printf '%s' "${PJSON}" | jq -er --arg id "${FP_PID}" '.Player | select(.id == $id) | .planetId // ""')
    if [ -z "${PLANET_ID}" ]; then
        run_tx "Fleet Player ${FP_NUM} exploring planet" \
            tx structs planet-explore "${FP_PID}" --from "${FPLAYER_KEY}"
    fi

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
    fp_require_command "${FP_NUM}"
    fp_move_home "${FP_NUM}"
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
#  PHASE 17b: Fleet Queue Limit — capacity 1 + back-and-forth (v0.21.0)
# ═════════════════════════════════════════════════════════════════════════════
# Default locationListExtra=0 => capacity 1 visitor. Home fleets do not occupy
# a queue slot. A second visitor is rejected until the first leaves.

section "PHASE 17b: Fleet Queue Limit"

for FP_NUM in 1 2 3 4 5; do
    fp_require_command "${FP_NUM}"
    fp_move_home "${FP_NUM}"
done

FP_TARGET_PLANET="${FP_1_PLANET_ID}"
FP_F1="${FP_1_FLEET_ID}"
FP_F2="${FP_2_FLEET_ID}"
FP_F3="${FP_3_FLEET_ID}"
info "Target planet: ${FP_TARGET_PLANET} (FP 1's home); visitors F2=${FP_F2} F3=${FP_F3}"

# Helper: read locationListCount (protobuf omitempty => missing means 0)
fp_queue_count() {
    query_planet "$1" | jq -er --arg id "$1" '.Planet | select(.id == $id) | .locationListCount // "0"'
}
fp_queue_extra() {
    query_planet "$1" | jq -er --arg id "$1" '.Planet | select(.id == $id) | .locationListExtra // "0"'
}

assert_eq "Target planet starts with locationListExtra 0" "0" "$(fp_queue_extra "${FP_TARGET_PLANET}")"
assert_eq "Target planet starts with locationListCount 0" "0" "$(fp_queue_count "${FP_TARGET_PLANET}")"

# ─── First visitor succeeds ───
wait_for_charge "${FP_2_ID}" "${CHARGE_MOVE}"
run_tx "Moving FP Fleet 2 (${FP_F2}) to planet ${FP_TARGET_PLANET}" \
    tx structs fleet-move "${FP_F2}" "${FP_TARGET_PLANET}" --from fplayer_2

F2_JSON=$(query_fleet "${FP_F2}")
assert_eq "FP Fleet 2 location after arrive" "${FP_TARGET_PLANET}" "$(jqr "${F2_JSON}" '.Fleet.locationId')"
assert_eq "FP Fleet 2 status away" "away" "$(jqr "${F2_JSON}" '.Fleet.status' 'onStation')"
assert_eq "Planet locationListStart is F2" "${FP_F2}" "$(get_planet_field "${FP_TARGET_PLANET}" "locationListStart")"
assert_eq "Planet locationListLast is F2" "${FP_F2}" "$(get_planet_field "${FP_TARGET_PLANET}" "locationListLast")"
assert_eq "Planet locationListCount is 1" "1" "$(fp_queue_count "${FP_TARGET_PLANET}")"

# Home fleet is not in the queue
F1_JSON=$(query_fleet "${FP_F1}")
assert_eq "FP Fleet 1 still on station at home" "${FP_1_PLANET_ID}" "$(jqr "${F1_JSON}" '.Fleet.locationId')"
F1_STATUS=$(jqr "${F1_JSON}" '.Fleet.status')
if [ -z "${F1_STATUS}" ]; then F1_STATUS="onStation"; fi
assert_eq "FP Fleet 1 status onStation" "onStation" "${F1_STATUS}"

# ─── Second visitor rejected (queue_full) ───
wait_for_charge "${FP_3_ID}" "${CHARGE_MOVE}"
run_tx_expect_reject_matching "FP Fleet 3 blocked while queue full" "queue_full" \
    tx structs fleet-move "${FP_F3}" "${FP_TARGET_PLANET}" --from fplayer_3

F3_JSON=$(query_fleet "${FP_F3}")
assert_eq "FP Fleet 3 still at home after reject" "${FP_3_PLANET_ID}" "$(jqr "${F3_JSON}" '.Fleet.locationId')"
assert_eq "Planet locationListCount unchanged after reject" "1" "$(fp_queue_count "${FP_TARGET_PLANET}")"

# ─── Head returns home: count drops, slot frees ───
wait_for_charge "${FP_2_ID}" "${CHARGE_MOVE}"
run_tx "FP Fleet 2 returns home (${FP_2_PLANET_ID})" \
    tx structs fleet-move "${FP_F2}" "${FP_2_PLANET_ID}" --from fplayer_2

F2_JSON=$(query_fleet "${FP_F2}")
assert_eq "FP Fleet 2 back home" "${FP_2_PLANET_ID}" "$(jqr "${F2_JSON}" '.Fleet.locationId')"
F2_STATUS=$(jqr "${F2_JSON}" '.Fleet.status')
if [ -z "${F2_STATUS}" ]; then F2_STATUS="onStation"; fi
assert_eq "FP Fleet 2 onStation after return" "onStation" "${F2_STATUS}"
assert_eq "Planet locationListCount 0 after F2 left" "0" "$(fp_queue_count "${FP_TARGET_PLANET}")"
assert_eq "Planet locationListStart cleared" "" "$(get_planet_field "${FP_TARGET_PLANET}" "locationListStart")"
assert_eq "Planet locationListLast cleared" "" "$(get_planet_field "${FP_TARGET_PLANET}" "locationListLast")"

# ─── Previously blocked visitor can now arrive ───
wait_for_charge "${FP_3_ID}" "${CHARGE_MOVE}"
run_tx "Moving FP Fleet 3 (${FP_F3}) to planet ${FP_TARGET_PLANET} after slot freed" \
    tx structs fleet-move "${FP_F3}" "${FP_TARGET_PLANET}" --from fplayer_3

F3_JSON=$(query_fleet "${FP_F3}")
assert_eq "FP Fleet 3 location after arrive" "${FP_TARGET_PLANET}" "$(jqr "${F3_JSON}" '.Fleet.locationId')"
assert_eq "Planet locationListCount is 1 with F3" "1" "$(fp_queue_count "${FP_TARGET_PLANET}")"
assert_eq "Planet locationListStart is F3" "${FP_F3}" "$(get_planet_field "${FP_TARGET_PLANET}" "locationListStart")"

# ─── Back-and-forth: F3 home, F2 visits again, F2 home ───
wait_for_charge "${FP_3_ID}" "${CHARGE_MOVE}"
run_tx "FP Fleet 3 returns home" \
    tx structs fleet-move "${FP_F3}" "${FP_3_PLANET_ID}" --from fplayer_3
assert_eq "Count 0 after F3 left" "0" "$(fp_queue_count "${FP_TARGET_PLANET}")"

wait_for_charge "${FP_2_ID}" "${CHARGE_MOVE}"
run_tx "FP Fleet 2 revisits target (back-and-forth)" \
    tx structs fleet-move "${FP_F2}" "${FP_TARGET_PLANET}" --from fplayer_2
assert_eq "Count 1 after F2 revisit" "1" "$(fp_queue_count "${FP_TARGET_PLANET}")"
assert_eq "Start is F2 after revisit" "${FP_F2}" "$(get_planet_field "${FP_TARGET_PLANET}" "locationListStart")"

wait_for_charge "${FP_3_ID}" "${CHARGE_MOVE}"
run_tx_expect_reject_matching "FP Fleet 3 still blocked during F2 revisit" "queue_full" \
    tx structs fleet-move "${FP_F3}" "${FP_TARGET_PLANET}" --from fplayer_3

# Leave F2 on the target for Phase 17c single-visitor combat
info "Queue limit verified; F2 remains on ${FP_TARGET_PLANET} for combat phase"
echo "  locationListExtra=$(fp_queue_extra "${FP_TARGET_PLANET}") locationListCount=$(fp_queue_count "${FP_TARGET_PLANET}")"
echo "  start=$(get_planet_field "${FP_TARGET_PLANET}" "locationListStart") last=$(get_planet_field "${FP_TARGET_PLANET}" "locationListLast")"

fi # phase 17b


if run_phase 2400; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE 17c: Single-Visitor Combat — home vs one raider (v0.21.0)
# ═════════════════════════════════════════════════════════════════════════════
# With capacity 1, only the head visitor is present. Range rules that still
# apply: home can hit the front visitor; the front visitor can hit the home
# fleet. Destroying the raider's Command Ship sends them home and clears the
# queue (count → 0).

section "PHASE 17c: Single-Visitor Combat"

FP_TARGET_PLANET="${FP_1_PLANET_ID}"
FP_F1="${FP_1_FLEET_ID}"
FP_F2="${FP_2_FLEET_ID}"

fp_queue_count() {
    query_planet "$1" | jq -er --arg id "$1" '.Planet | select(.id == $id) | .locationListCount // "0"'
}

# A direct phase-17c resume may follow a completed or interrupted battle.
for FP_NUM in 1 2 3 4 5; do
    fp_require_command "${FP_NUM}"
    fp_move_home "${FP_NUM}"
done
fp_move 2 "${FP_TARGET_PLANET}"
assert_eq "Combat setup: F2 on target" "${FP_TARGET_PLANET}" "$(get_fleet_field "${FP_F2}" "locationId")"
assert_eq "Combat setup: queue count 1" "1" "$(fp_queue_count "${FP_TARGET_PLANET}")"

# Preserve surviving hulls. Either opening attack may destroy a damaged command;
# restore it before the next required scenario instead of skipping that scenario.
fp_attack 1 2 "Home fleet hit sole visitor"
fp_require_command 1
fp_require_command 2
fp_move 2 "${FP_TARGET_PLANET}"
fp_attack 2 1 "Front visitor hit home fleet"
fp_require_command 1

info "Destroying FP_CS_2 to clear the queue"
FP_CS2_HP=$(query query structs struct "${FP_CS_2}" | jq -er --arg id "${FP_CS_2}" \
    'select(.Struct.id == $id) | .structAttributes.health | select(. == "1" or . == "2" or . == "3" or . == "4" or . == "5" or . == "6")')
ATTACK_COUNT=0
while [ "${FP_CS2_HP}" -gt 0 ] && [ "${ATTACK_COUNT}" -lt 3 ]; do
    ATTACK_COUNT=$((ATTACK_COUNT + 1))
    fp_require_command 1
    fp_attack 1 2 "F1 destroys visitor command (attack ${ATTACK_COUNT})"
    FP_CS2_HP="${FP_LAST_TARGET_HEALTH}"
done
assert_eq "FP_CS_2 destroyed in confirmed attack" "0" "${FP_CS2_HP}"
[ "${FP_CS2_HP}" = "0" ]

F2_JSON=$(query_fleet "${FP_F2}")
F2_LOC=$(jqr "${F2_JSON}" '.Fleet.locationId')
F2_STATUS=$(jqr "${F2_JSON}" '.Fleet.status')
if [ -z "${F2_STATUS}" ]; then F2_STATUS="onStation"; fi
assert_eq "F2 returned home after CS destruction" "${FP_2_PLANET_ID}" "${F2_LOC}"
assert_eq "F2 onStation after recall" "onStation" "${F2_STATUS}"
assert_eq "Queue count 0 after visitor recalled" "0" "$(fp_queue_count "${FP_TARGET_PLANET}")"
assert_eq "locationListStart cleared after recall" "" "$(get_planet_field "${FP_TARGET_PLANET}" "locationListStart")"
assert_eq "locationListLast cleared after recall" "" "$(get_planet_field "${FP_TARGET_PLANET}" "locationListLast")"

# Slot is free again for another visitor
wait_for_charge "${FP_3_ID}" "${CHARGE_MOVE}"
run_tx "FP Fleet 3 occupies freed queue slot after combat recall" \
    tx structs fleet-move "${FP_3_FLEET_ID}" "${FP_TARGET_PLANET}" --from fplayer_3
assert_eq "Count 1 after F3 occupies freed slot" "1" "$(fp_queue_count "${FP_TARGET_PLANET}")"
assert_eq "Start is F3" "${FP_3_FLEET_ID}" "$(get_planet_field "${FP_TARGET_PLANET}" "locationListStart")"

# Send F3 home so later phases are not left mid-raid
wait_for_charge "${FP_3_ID}" "${CHARGE_MOVE}"
run_tx "FP Fleet 3 returns home (cleanup)" \
    tx structs fleet-move "${FP_3_FLEET_ID}" "${FP_3_PLANET_ID}" --from fplayer_3
assert_eq "Queue empty after combat cleanup" "0" "$(fp_queue_count "${FP_TARGET_PLANET}")"

info "Single-visitor combat complete"

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
    tx bank send "${BOB_ADDRESS}" "${PLAYER_6_ADDRESS}" 85000000ualpha --from bob

run_tx "Delegating 80000000ualpha from player_6 to validator" \
    tx staking delegate "${VALIDATOR_ADDRESS}" 80000000ualpha --from player_6

ADDR_JSON_6=$(query query structs address "${PLAYER_6_ADDRESS}")
PLAYER_6_ID=$(jqr "${ADDR_JSON_6}" '.playerId')
assert_not_empty "Player 6 ID" "${PLAYER_6_ID}"
echo "  Player 6 ID: ${PLAYER_6_ID}"

# ─── Extended feeder recovery ───
# Resolve one connected (or interrupted, unconnected) feeder. Resizing releases
# its own load, while other allocations and the build reserve stay protected.
P6_SETUP_HEIGHT=$(get_block_height)
P6_ALLOC_ID=$(get_fixture_feeder "${PLAYER_6_ID}" "${SUBSTATION_ID}" "${PLAYER_1_ID}" "${P6_SETUP_HEIGHT}")
if [ -z "${P6_ALLOC_ID}" ]; then
    P6_ALLOC_ID=$(get_fixture_feeder "${PLAYER_6_ID}" "" "${PLAYER_1_ID}" "${P6_SETUP_HEIGHT}")
fi
P6_ALLOC_JSON=null
if [ -n "${P6_ALLOC_ID}" ]; then
    P6_ALLOC_JSON=$(query query structs allocation "${P6_ALLOC_ID}" --height "${P6_SETUP_HEIGHT}")
fi
P6_JSON=$(query query structs player "${PLAYER_6_ID}" --height "${P6_SETUP_HEIGHT}")
P6_SETUP=$(jq -nc --argjson player "${P6_JSON}" --argjson allocation "${P6_ALLOC_JSON}" \
    '{player:$player,allocation:$allocation}' | python3 "${GENERATOR_TEST_DIR}/extended_battle_setup.py" \
    "${PLAYER_6_ID}" "${PLAYER_1_ID}" "${SUBSTATION_ID}" 8000000)
P6_ALLOC_POWER=$(printf '%s' "${P6_SETUP}" | jq -er '.power')
if [ -z "${P6_ALLOC_ID}" ]; then
    run_tx "Creating allocation from Player 6 (controller=alice)" \
        tx structs allocation-create "${PLAYER_6_ID}" "${P6_ALLOC_POWER}" \
        --controller "${PLAYER_1_ID}" --allocation-type dynamic --from player_6
    P6_ALLOC_ID=$(get_created_grid_id allocation "${PLAYER_6_ID}" "${PLAYER_1_ID}")
elif [ "$(printf '%s' "${P6_SETUP}" | jq -r '.needsUpdate')" = true ]; then
    run_tx "Resizing Player 6 fixture allocation" \
        tx structs allocation-update "${P6_ALLOC_ID}" "${P6_ALLOC_POWER}" --from player_6
fi
assert_not_empty "Player 6 allocation ID" "${P6_ALLOC_ID}"
assert_eq "Player 6 feeder has exact reserved power" "${P6_ALLOC_POWER}" \
    "$(query query structs allocation "${P6_ALLOC_ID}" | jq -er '.gridAttributes.power')"

if [ "$(printf '%s' "${P6_JSON}" | jq -er '.Player.guildId // ""')" != "${GUILD_ID}" ]; then
    run_tx "Player 6 joining guild" \
        tx structs guild-membership-join "${GUILD_ID}" "${REACTOR_ID}-${PLAYER_6_ADDRESS}" --from player_6
fi
if [ "$(printf '%s' "${P6_SETUP}" | jq -r '.destinationId')" = "" ]; then
    run_tx "Connecting Player 6 allocation to substation" \
        tx structs substation-allocation-connect "${P6_ALLOC_ID}" "${SUBSTATION_ID}" --from alice
fi
assert_eq "Player 6 feeder destination" "${SUBSTATION_ID}" \
    "$(query query structs allocation "${P6_ALLOC_ID}" | jq -er '.Allocation.destinationId')"

P6_JSON=$(query query structs player "${PLAYER_6_ID}")
assert_eq "Player 6 guild membership" "${GUILD_ID}" "$(jqr "${P6_JSON}" '.Player.guildId')"
PLAYER_6_PLANET_ID=$(printf '%s' "${P6_JSON}" | jq -r '.Player.planetId // ""')
if [ -z "${PLAYER_6_PLANET_ID}" ]; then
    run_tx "Player 6 exploring a planet" \
        tx structs planet-explore "${PLAYER_6_ID}" --from player_6
    P6_JSON=$(query query structs player "${PLAYER_6_ID}")
    PLAYER_6_PLANET_ID=$(jqr "${P6_JSON}" '.Player.planetId')
fi
PLAYER_6_FLEET_ID=$(jqr "${P6_JSON}" '.Player.fleetId')
assert_not_empty "Player 6 planet" "${PLAYER_6_PLANET_ID}"
assert_not_empty "Player 6 fleet" "${PLAYER_6_FLEET_ID}"
combat_require_command P6_COMMAND_SHIP_ID 6 "extended battle command"
combat_move_fleet 6 "${PLAYER_6_PLANET_ID}" player_6
# ─── Extended feeder recovery complete ───

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

# Reuse living roles and finish interrupted builds; never build into a fixed
# occupied slot on resume. Prepare every role before computing so newly missing
# fixtures still age together, as they do in the fresh extended battle.
combat_move_fleet 3 "${PLAYER_3_PLANET_ID}" player_3
combat_move_fleet 6 "${PLAYER_6_PLANET_ID}" player_6
extended_battle_roles() {
    local mode="$1"
    combat_require_struct EB_PURSUIT_FIGHTER_ID 3 5 fleet "${PLAYER_3_FLEET_ID}" air "EB Pursuit Fighter" "" player_3 "${mode}"
    combat_require_struct EB_STARFIGHTER_ID 6 3 fleet "${PLAYER_6_FLEET_ID}" space "EB Starfighter" "" player_6 "${mode}"
    combat_require_struct EB_FRIGATE_ID 6 4 fleet "${PLAYER_6_FLEET_ID}" space "EB Frigate" "" player_6 "${mode}"
    combat_require_struct EB_MOBILE_ART_ID 6 8 fleet "${PLAYER_6_FLEET_ID}" land "EB Mobile Artillery" "" player_6 "${mode}"
    combat_require_struct EB_DESTROYER_W_ID 6 12 fleet "${PLAYER_6_FLEET_ID}" water "EB Destroyer" "" player_6 "${mode}"
    combat_require_struct EB_P6_BATTLESHIP_ID 6 2 fleet "${PLAYER_6_FLEET_ID}" space "EB Battleship" "" player_6 "${mode}"
    combat_require_struct EB_P6_TANK_ID 6 9 fleet "${PLAYER_6_FLEET_ID}" land "EB Tank" "" player_6 "${mode}"
    combat_require_struct EB_P6_CRUISER_ID 6 11 fleet "${PLAYER_6_FLEET_ID}" water "EB Cruiser" "" player_6 "${mode}"
    combat_require_struct EB_P6_HAI_ID 6 7 fleet "${PLAYER_6_FLEET_ID}" air "EB Interceptor" "" player_6 "${mode}"
    combat_require_struct EB_P3_MOBILE_ART_ID 3 8 fleet "${PLAYER_3_FLEET_ID}" land "EB P3 Mobile Artillery" "" player_3 "${mode}"
    combat_require_struct EB_PDC_ID 6 19 planet "${PLAYER_6_PLANET_ID}" land "EB PDC" "" player_6 "${mode}"
    combat_require_struct EB_ORE_EXTRACTOR_ID 6 14 planet "${PLAYER_6_PLANET_ID}" land "EB Ore Extractor" "" player_6 "${mode}"
}
extended_battle_roles prepare
extended_battle_roles ready

# Verify PDC added a defensive cannon to P6's planet
P6_PLANET_JSON=$(query query structs planet "${PLAYER_6_PLANET_ID}" 2>/dev/null || echo '{}')
P6_DEF_CANNON_QTY=$(jqr "${P6_PLANET_JSON}" '.planetAttributes.defensiveCannonQuantity' '0')
assert_gt "P6 planet has defensive cannons" 0 "${P6_DEF_CANNON_QTY}"
info "P6 planet defensive cannon quantity: ${P6_DEF_CANNON_QTY}"


info "Extended battle additions are ready; assembly restores the remaining fleet roles"
info "  Types 1-13: Command Ship, Battleship, Starfighter, Frigate, Pursuit Fighter,"
info "              Stealth Bomber, HAI, Mobile Artillery, Tank, SAM, Cruiser, Destroyer(W), Submersible"
info "  P6 HAI (type 7, air) ready for defensiveManeuver evasion testing"

fi # phase EB2

if run_phase 2700; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE EB3: Fleet Assembly & Positioning
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE EB3: Fleet Assembly & Positioning"

# Fleet units are built directly on their final fleet slots (Movable=false).
# Only the Command Ship may struct-move (ambit changes).

# Restore the ordinary-phase roles after earlier combat or a direct resume.
combat_move_fleet 3 "${PLAYER_3_PLANET_ID}" player_3
combat_move_fleet 6 "${PLAYER_6_PLANET_ID}" player_6
ar_require_command 3
ar_require_command 6
combat_require_struct DESTROYER_STRUCT_ID 3 9 fleet "${PLAYER_3_FLEET_ID}" land "EB P3 Tank"
combat_require_struct SAM_STRUCT_ID 3 10 fleet "${PLAYER_3_FLEET_ID}" land "EB P3 SAM"
combat_require_struct SUB_STRUCT_ID 3 13 fleet "${PLAYER_3_FLEET_ID}" water "EB P3 Submersible"
combat_require_struct BATTLESHIP_1_ID 3 2 fleet "${PLAYER_3_FLEET_ID}" space "EB P3 Battleship 1"
combat_require_struct BATTLESHIP_2_ID 3 2 fleet "${PLAYER_3_FLEET_ID}" space "EB P3 Battleship 2" "${BATTLESHIP_1_ID}"
combat_require_struct STEALTH_BOMBER_ID 3 6 fleet "${PLAYER_3_FLEET_ID}" air "EB P3 Stealth Bomber"
combat_require_struct CRUISER_ID 3 11 fleet "${PLAYER_3_FLEET_ID}" water "EB P3 Cruiser"

info "Extended battle roles are ready in their required ambits; existing slot positions are preserved"
combat_move_fleet 3 "${PLAYER_6_PLANET_ID}" player_3

info "P3's fleet is now at P6's planet — battle positions set"

fi # phase EB3

if run_phase 2800; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE EB4: Defense Configurations
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE EB4: Defense Configurations"

combat_clear_target_defenders 6 "${EB_MOBILE_ART_ID}"
combat_clear_target_defenders 6 "${EB_P6_BATTLESHIP_ID}"
combat_clear_target_defenders 3 "${BATTLESHIP_1_ID}"

# ─── P6: Tank defends Mobile Artillery (same-ambit defense, both land) ───
wait_for_charge "${PLAYER_6_ID}" "${CHARGE_DEFEND}"
run_tx "P6 Tank defends Mobile Artillery (same-ambit: land)" \
    tx structs struct-defense-set "${EB_P6_TANK_ID}" "${EB_MOBILE_ART_ID}" --from player_6

MOBILE_ART_JSON=$(query query structs struct "${EB_MOBILE_ART_ID}" 2>/dev/null || echo '{}')
MA_DEFENDERS=$(echo "${MOBILE_ART_JSON}" | jq -r '.structDefenders | length' 2>/dev/null || echo "0")
assert_gt "Mobile Artillery has defenders" 0 "${MA_DEFENDERS}"
info "Mobile Artillery defender count: ${MA_DEFENDERS}"

# ─── v0.21.0: planetary structs cannot defend (canDefend=false) ───
# The Ore Extractor (type 14, planet-category) is co-located with the docked
# Mobile Artillery, so the range check passes; the canDefend gate is the
# isolated failure. canDefend gates the DEFENDER, not the protected struct —
# a fleet struct defending the Ore Extractor is still allowed (see line 7186).
wait_for_charge "${PLAYER_6_ID}" "${CHARGE_DEFEND}"
run_tx_expect_fail "P6 Ore Extractor cannot defend (planetary struct, canDefend=false, v0.21.0)" \
    tx structs struct-defense-set "${EB_ORE_EXTRACTOR_ID}" "${EB_MOBILE_ART_ID}" --from player_6

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

# ─── Required negative targeting before destructive combat ───
# Rejections must precede the attacks that can destroy these same fixtures.
# A space-only Starfighter cannot isolate stealth on an air target: CanAttack
# checks range first. The Frigate primary supports both space and air.
info "── Required negative targeting and stealth ──"
eb_attack_should_fail "Tank(land-only) → Battleship(space) — wrong ambit" \
    "${DESTROYER_STRUCT_ID}" "${EB_P6_BATTLESHIP_ID}" 3
eb_attack_should_fail "Pursuit Fighter(air-only) → Tank(land) — wrong ambit" \
    "${EB_PURSUIT_FIGHTER_ID}" "${EB_P6_TANK_ID}" 3
eb_attack_should_fail "Starfighter(space-only) → SAM(land) — wrong ambit" \
    "${EB_STARFIGHTER_ID}" "${SAM_STRUCT_ID}" 6
SB_HIDDEN=$(query query structs struct "${STEALTH_BOMBER_ID}" | jq -er --arg id "${STEALTH_BOMBER_ID}" '
    select(.Struct.id == $id) | .structAttributes | select(type == "object") |
    (if has("isHidden") then .isHidden else false end) | select(type == "boolean") | tostring')
if [ "${SB_HIDDEN}" = "false" ]; then
    wait_for_charge "${PLAYER_3_ID}" "${CHARGE_ACTIVATE}"
    run_tx "Activating stealth on Stealth Bomber" \
        tx structs struct-stealth-activate "${STEALTH_BOMBER_ID}" --from player_3
fi
eb_attack_should_fail "Stealth: P6 Frigate(space) → hidden Stealth Bomber(air)" \
    "${EB_FRIGATE_ID}" "${STEALTH_BOMBER_ID}" 6 hidden
wait_for_charge "${PLAYER_3_ID}" "${CHARGE_ACTIVATE}"
run_tx "Deactivating stealth on Stealth Bomber" \
    tx structs struct-stealth-deactivate "${STEALTH_BOMBER_ID}" --from player_3
# ─── Required negative targeting complete ───

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
# Restore the exact roles after earlier attrition. Guided evasion is possible,
# but the air-only Pursuit Fighter must never counter the space Frigate.
ar_require_command 2
ar_require_command 3
ar_require_command 6
ar_park_as_head 3
ev_attack EB_B6 EB_FRIGATE_ID 6 4 space EB_PURSUIT_FIGHTER_ID 3 5 air primaryWeapon
# Required Frigate cross-ambit attack complete

# ─────────────────────────────────────────────────────────────────────────────
# GROUP C: Special Mechanics
# ─────────────────────────────────────────────────────────────────────────────

info "── Group C: Special Mechanics ──"

# C1b: A blocker sorting before a lethal counter may not absorb a dead volley.
# Restore all four roles, prepare exactly one bomber health, and require the
# Tank-before-HAI ordering plus exact receipt and canonical-state evidence.
da_run
# Required dead-bomber ordering complete

# C2: Required block with the original Battleship/Frigate/Starfighter matchup.
bc_run
# Required two-defender blocking complete

# C3: Required damage reduction with restored Tank roles and exact receipts.
da_armour
# Required Tank armour complete

# C4: Required sustained combat from full health through confirmed destruction.
da_sustained
# Required sustained combat complete

# C5: Required local-weapon range refusal and same-ambit attack.
da_command
# Required command local weapon complete

# ─────────────────────────────────────────────────────────────────────────────
# GROUP E: Planetary Defense Cannon (PDC) Tests
# Required live-cannon damage (E1/E2), non-counterable immunity (E3), fleet
# immunity (E4), softening (E4b), killing blow (E5), cannon/shield removal
# (E8/E9), and the same extractor attacked after cannon destruction (E10).
# Cannon damage uses the count surviving the volley, so a destroyed cannon
# does not fire. All cases check receipts and delivery-height stored state.
pd_run
# Required planetary cannon sequence complete

fi # phase EB5

if run_phase 3000; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE EB6: Battle Results Review
# ═════════════════════════════════════════════════════════════════════════════

section "PHASE EB6: Extended Battle Results"

info "Querying final state of all combat structs"

echo ""
echo "  ─── Player 3 Fleet Status ───"
# Health defaults to 0, not "?": once the rubble sweep removes a struct the field
# is simply absent, and defaulting to "?" made the status check below read it as
# alive — a results dump that reported destroyed structs as survivors. Type and
# ambit stay "?" because those genuinely are unknown for a struct that is gone.
for SID in "${COMMAND_SHIP_ID}" "${DESTROYER_STRUCT_ID}" "${SAM_STRUCT_ID}" "${SUB_STRUCT_ID}" \
           "${BATTLESHIP_1_ID}" "${BATTLESHIP_2_ID}" "${STEALTH_BOMBER_ID}" "${CRUISER_ID}" \
           "${EB_PURSUIT_FIGHTER_ID}" "${EB_P3_MOBILE_ART_ID}"; do
    S_JSON=$(query query structs struct "${SID}" 2>/dev/null || echo '{}')
    S_HP=$(echo "${S_JSON}" | jq -r '.structAttributes.health // "0"' 2>/dev/null || echo "0")
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
    S_HP=$(echo "${S_JSON}" | jq -r '.structAttributes.health // "0"' 2>/dev/null || echo "0")
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

info "Every evasion attempt is required; destroyed roles are rebuilt before use"
info "Each outcome is checked against the receipt and committed state; the random sample is not a probability test"

combat_require_struct P6_COMMAND_SHIP_ID 6 1 fleet "${PLAYER_6_FLEET_ID}" space "EV P6 Command Ship"
combat_require_struct COMMAND_SHIP_ID 3 1 fleet "${PLAYER_3_FLEET_ID}" space "EV P3 Command Ship"
PLAYER_3_CMD_SHIP_ID="${COMMAND_SHIP_ID}"
combat_move_fleet 6 "${PLAYER_6_PLANET_ID}"
combat_move_fleet 3 "${PLAYER_6_PLANET_ID}"
EV_CONFIRMED=0
EV_EVADED=0

# HAI guided primary versus Pursuit Fighter signal jamming (air -> air).
for EV_CASE in EV1 EV2 EV3; do
    ev_attack "${EV_CASE}" EB_P6_HAI_ID 6 7 air EB_PURSUIT_FIGHTER_ID 3 5 air primaryWeapon
done

# Command Ship guided primary versus Battleship signal jamming (space -> space).
for EV_CASE in EV4 EV5; do
    ev_attack "${EV_CASE}" P6_COMMAND_SHIP_ID 6 1 space BATTLESHIP_2_ID 3 2 space primaryWeapon
done

# Cruiser unguided secondary versus HAI defensive maneuver (water -> air).
for EV_CASE in EV6 EV7; do
    ev_attack "${EV_CASE}" CRUISER_ID 3 11 water EB_P6_HAI_ID 6 7 air secondaryWeapon
done

# Destroyer guided primary versus Cruiser signal jamming (water -> water).
ev_attack EV8 EB_DESTROYER_W_ID 6 12 water CRUISER_ID 3 11 water primaryWeapon

assert_eq "All eight scheduled evasion attacks were verified" 8 "${EV_CONFIRMED}"
info "Evasion outcomes: ${EV_EVADED} evaded, $((EV_CONFIRMED - EV_EVADED)) hit; no probability claim from eight attempts"

fi # phase EV1

if run_phase 3100; then
section "PHASE AR1: Attack Run — Required Fighters"

# Three distinct P2 fighters, two P3 fighters and one P6 fighter. Reuse living
# roles and rebuild losses using actual occupancy; command ships have separate slots.
fund_vault_alpha "${PLAYER_2_ID}" 4000000
run_tx "Additional vault infusion for Player 2 (fleet capacity)" \
    tx structs reactor-infuse "${VALIDATOR_ADDRESS}" 4000000ualpha --from player_2
for AR_PLAYER in 2 3 6; do ar_require_command "${AR_PLAYER}"; done
ar_require_fighter AR_P2_SF1_ID 2
ar_require_fighter AR_P2_SF2_ID 2
ar_require_fighter AR_P2_SF3_ID 2
ar_require_fighter AR_P3_SF1_ID 3
ar_require_fighter AR_P3_SF2_ID 3
ar_require_fighter AR_P6_SF1_ID 6
fi # phase AR1

if run_phase 3200; then
section "PHASE AR2: Attack Run — Fleet Assembly"
for AR_PLAYER in 2 3 6; do ar_require_command "${AR_PLAYER}"; done
ar_park_as_head 2
wait_for_charge "${PLAYER_3_ID}" "${CHARGE_MOVE}"
run_tx_expect_reject_matching "P3 cannot enter P6's full raid queue" "queue_full" \
    tx structs fleet-move "${PLAYER_3_FLEET_ID}" "${PLAYER_6_PLANET_ID}" --from player_3
ar_verify_head 2
AR_P3_LOCATION=$(combat_fleet_location 3)
assert_eq "Rejected second visitor stays home" "${PLAYER_3_PLANET_ID}" "${AR_P3_LOCATION}"
[ "${AR_P3_LOCATION}" = "${PLAYER_3_PLANET_ID}" ]
fi # phase AR2

if run_phase 3300; then
section "PHASE AR3: Attack Run — Combat"
AR_PLANNED=15
AR_ATTACKS=0
AR_EXECUTED=0
AR_DESTROYED=0

# Each case restores every role, verifies complete defender registration and
# checks its live catalog, confirmed receipt and canonical state at pinned heights.
# B2/B3 are nonparticipating cross-ambit defenders; C/D include real counters.
for AR_CASE in A1 A2 A3 B1 B2 B3 C1 C2 D1 D2 D3 D4 E1 E2 E3; do
    ar_attack "${AR_CASE}"
done
assert_eq "All fifteen attack-run cases were verified" "${AR_PLANNED}" "${AR_EXECUTED}"
[ "${AR_EXECUTED}" -eq "${AR_PLANNED}" ]
fi # phase AR3

if run_phase 3400; then
section "PHASE AR4: Attack Run — Results"
info "Attack Run Summary: ${AR_EXECUTED:-0} verified cases, ${AR_DESTROYED:-0} structs destroyed"
# Receipts above retain the identities of every replacement and each exact
# outcome. Latest-height health lookups cannot distinguish a sweep from RPC failure.
BLOCK_HEIGHT=$(query query structs block-height | jq -er '.blockHeight')
info "Block height after Attack Run: ${BLOCK_HEIGHT}"
fi # phase AR4

if run_phase 3450; then

# A same-ambit defender must stop after its counter kills the attacker:
# no block may run and no projectile may damage the target or defender.
# Restore exact roles and prepare one health through verified signed combat
# when earlier attacks did not leave that condition. Missing roles are failures.
section "PHASE RG1: Regression — Defender Counter Destroys Attacker (Bug 1)"
rg1_run

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
#   Defender: RG2_DEFENDER_ID (P6 fleet). Mobile Artillery has
#             CounterAttack=0, so the only defensive action it could take is
#             block — and only when co-located with the protected target.
#   Target:   RG2_TARGET_ID (P6 planet). Same ambit as
#             the MA defender, so block would fire when in range.
#   Attacker: RG2_ATTACKER_ID (P3 fleet, moved to P6 planet).
#             AttackCounterable=false, so PDC defensive cannons do
#             not return damage and we can observe block independently.
#   Required roles are recovered or rebuilt; missing fixtures fail the phase.
#   The stale defender index, receipt and resulting state are checked at
#   explicit heights so destruction sweeps cannot race the assertions.
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

# This regression is required. Earlier attacks may destroy any of its roles,
# so recover or rebuild them instead of interpreting attrition as coverage.
combat_require_struct P6_COMMAND_SHIP_ID 6 1 fleet "${PLAYER_6_FLEET_ID}" space "P6 Command Ship"
combat_require_struct COMMAND_SHIP_ID 3 1 fleet "${PLAYER_3_FLEET_ID}" space "P3 Command Ship"
PLAYER_3_CMD_SHIP_ID="${COMMAND_SHIP_ID}"
combat_move_fleet 6 "${PLAYER_6_PLANET_ID}"
combat_move_fleet 3 "${PLAYER_6_PLANET_ID}"
combat_require_struct RG2_DEFENDER_ID 6 8 fleet "${PLAYER_6_FLEET_ID}" land "P6 Mobile Artillery defender"
combat_require_struct RG2_TARGET_ID 6 14 planet "${PLAYER_6_PLANET_ID}" land "P6 Ore Extractor target"
combat_require_struct RG2_ATTACKER_ID 3 8 fleet "${PLAYER_3_FLEET_ID}" land "P3 Mobile Artillery attacker"

wait_for_charge "${PLAYER_6_ID}" "${CHARGE_DEFEND}"
run_tx "RG2: Register defender before relocation" \
    tx structs struct-defense-set "${RG2_DEFENDER_ID}" "${RG2_TARGET_ID}" --from player_6
combat_move_fleet 6 "${PLAYER_2_PLANET_ID}"
wait_for_charge "${PLAYER_3_ID}" "$(eb_get_charge "${RG2_ATTACKER_ID}" primaryWeapon)"

# Inspect all three roles and both fleet locations at the same committed height.
# The defender index must still exist: otherwise no stale relationship is tested.
RG2_BEFORE_HEIGHT=$(query query structs block-height | jq -er '.blockHeight | select(type == "string" and test("^[1-9][0-9]*$"))')
RG2_ATTACKER_BEFORE=$(combat_state_at_height "${RG2_ATTACKER_ID}" "${RG2_BEFORE_HEIGHT}")
RG2_TARGET_BEFORE=$(combat_state_at_height "${RG2_TARGET_ID}" "${RG2_BEFORE_HEIGHT}")
RG2_DEFENDER_BEFORE=$(combat_state_at_height "${RG2_DEFENDER_ID}" "${RG2_BEFORE_HEIGHT}")
for RG2_ROLE in ATTACKER TARGET DEFENDER; do
    case "${RG2_ROLE}" in
        ATTACKER) RG2_ROLE_ID="${RG2_ATTACKER_ID}"; RG2_ROLE_STATE="${RG2_ATTACKER_BEFORE}"; RG2_OWNER="${PLAYER_3_ID}"; RG2_KIND=8; RG2_LOCATION="${PLAYER_3_FLEET_ID}" ;;
        TARGET) RG2_ROLE_ID="${RG2_TARGET_ID}"; RG2_ROLE_STATE="${RG2_TARGET_BEFORE}"; RG2_OWNER="${PLAYER_6_ID}"; RG2_KIND=14; RG2_LOCATION="${PLAYER_6_PLANET_ID}" ;;
        DEFENDER) RG2_ROLE_ID="${RG2_DEFENDER_ID}"; RG2_ROLE_STATE="${RG2_DEFENDER_BEFORE}"; RG2_OWNER="${PLAYER_6_ID}"; RG2_KIND=8; RG2_LOCATION="${PLAYER_6_FLEET_ID}" ;;
    esac
    RG2_READY=$(printf '%s' "${RG2_ROLE_STATE}" | python3 "${GENERATOR_TEST_DIR}/relocated_defender.py" fixture "${RG2_ROLE_ID}" "${RG2_OWNER}" "${RG2_KIND}" "${RG2_LOCATION}" land)
    [ "${RG2_READY}" = ready ] || { echo "FAIL: RG2 role ${RG2_ROLE} became unavailable" >&2; exit 1; }
done
printf '%s' "${RG2_TARGET_BEFORE}" | jq -e --arg defender "${RG2_DEFENDER_ID}" '.structDefenders == [$defender]' >/dev/null
[ "$(combat_fleet_location 6 "${RG2_BEFORE_HEIGHT}")" = "${PLAYER_2_PLANET_ID}" ]
[ "$(combat_fleet_location 3 "${RG2_BEFORE_HEIGHT}")" = "${PLAYER_6_PLANET_ID}" ]
[ "${PLAYER_2_PLANET_ID}" != "${PLAYER_6_PLANET_ID}" ]
RG2_ATTACKER_HP=$(printf '%s' "${RG2_ATTACKER_BEFORE}" | jq -er '.structAttributes.health')
RG2_TARGET_HP=$(printf '%s' "${RG2_TARGET_BEFORE}" | jq -er '.structAttributes.health')
RG2_DEFENDER_HP=$(printf '%s' "${RG2_DEFENDER_BEFORE}" | jq -er '.structAttributes.health')

run_tx "RG2: Attack target after its defender relocated" \
    tx structs struct-attack "${RG2_ATTACKER_ID}" "${RG2_TARGET_ID}" primaryWeapon --from player_3
RG2_TX_HASH=$(_tx_json "${LAST_TX_OUTPUT}" | jq -er '.txhash | select(type == "string" and test("^[0-9A-Fa-f]{64}$"))')
RG2_OUTCOME=$(query query tx "${RG2_TX_HASH}" | python3 "${GENERATOR_TEST_DIR}/relocated_defender.py" attack "${RG2_TX_HASH}" "${RG2_ATTACKER_ID}" "${RG2_TARGET_ID}" "${RG2_DEFENDER_ID}" "${RG2_ATTACKER_HP}" "${RG2_TARGET_HP}" "${RG2_DEFENDER_HP}")
RG2_TX_HEIGHT=$(printf '%s' "${RG2_OUTCOME}" | jq -er '.height')
[ "${RG2_TX_HEIGHT}" -gt "${RG2_BEFORE_HEIGHT}" ]
RG2_TARGET_AFTER_HP=$(printf '%s' "${RG2_OUTCOME}" | jq -er '.health')
RG2_TARGET_DESTROYED=$(printf '%s' "${RG2_OUTCOME}" | jq -r '.destroyed')
combat_state_at_height "${RG2_ATTACKER_ID}" "${RG2_TX_HEIGHT}" | python3 "${GENERATOR_TEST_DIR}/relocated_defender.py" state "${RG2_ATTACKER_ID}" "${PLAYER_3_ID}" 8 "${PLAYER_3_FLEET_ID}" land "${RG2_ATTACKER_HP}" false
combat_state_at_height "${RG2_TARGET_ID}" "${RG2_TX_HEIGHT}" | python3 "${GENERATOR_TEST_DIR}/relocated_defender.py" state "${RG2_TARGET_ID}" "${PLAYER_6_ID}" 14 "${PLAYER_6_PLANET_ID}" land "${RG2_TARGET_AFTER_HP}" "${RG2_TARGET_DESTROYED}"
combat_state_at_height "${RG2_DEFENDER_ID}" "${RG2_TX_HEIGHT}" | python3 "${GENERATOR_TEST_DIR}/relocated_defender.py" state "${RG2_DEFENDER_ID}" "${PLAYER_6_ID}" 8 "${PLAYER_6_FLEET_ID}" land "${RG2_DEFENDER_HP}" false
[ "$(combat_fleet_location 6 "${RG2_TX_HEIGHT}")" = "${PLAYER_2_PLANET_ID}" ]
[ "$(combat_fleet_location 3 "${RG2_TX_HEIGHT}")" = "${PLAYER_6_PLANET_ID}" ]
assert_eq "RG2: Confirmed hit bypassed relocated defender; canonical state agrees" true "$(printf '%s' "${RG2_OUTCOME}" | jq -r '.verified')"

wait_for_charge "${PLAYER_6_ID}" "${CHARGE_DEFEND}"
run_tx "RG2 cleanup: clear stale defender registration" \
    tx structs struct-defense-clear "${RG2_DEFENDER_ID}" --from player_6
combat_move_fleet 6 "${PLAYER_6_PLANET_ID}"

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
#       (idempotent no-op on the value, with confirmed successful delivery).
#    4. Confirm the guild's primaryReactorId is unchanged after the no-op
#       update (and matches the configured reactor).
# ═════════════════════════════════════════════════════════════════════════════
section "PHASE GP1: Guild Update Primary Reactor"

# Bind snapshots to the requested guild and compare immediately after each
# attempt, before a later successful update could conceal a rejected mutation.
gp1_guild_snapshot() {
    local json
    json=$(query query structs guild "${GUILD_ID}") || return 1
    echo "${json}" | jq -ceS --arg id "${GUILD_ID}" '
        .Guild | if type != "object" or .id != $id then error("wrong guild")
        elif (.primaryReactorId | type) != "string" or .primaryReactorId == "" then error("missing primary reactor")
        else . end'
}
GP1_BEFORE=$(gp1_guild_snapshot)
assert_eq "GP1: pre-update guild.primaryReactorId" \
    "${REACTOR_ID}" "$(echo "${GP1_BEFORE}" | jq -r '.primaryReactorId')"

# 1. Reject: unknown reactor. Format mirrors a real reactor id ("<n>-<addr>")
#    so it parses but resolves to no object.
GP1_FAKE_REACTOR="999999-${ALICE_ADDRESS}"
run_tx_expect_reject_matching "GP1: reject update to unknown reactor (${GP1_FAKE_REACTOR})" \
    "reactor \\(${GP1_FAKE_REACTOR}\\) not found" \
    tx structs guild-update-primary-reactor "${GUILD_ID}" "${GP1_FAKE_REACTOR}" --from alice
GP1_AFTER=$(gp1_guild_snapshot)
assert_eq "GP1: unknown-reactor refusal preserves the guild" "${GP1_BEFORE}" "${GP1_AFTER}"

# 2. Reject: non-admin caller (player_2).
#    Other phases may grant player_2 explicit administrative permissions
#    and briefly made them the guild owner (which adds PermGuildAll). Neither
#    grant is revoked there, so by the time we get here player_2 legitimately
#    has admin rights on the guild. Strip every bit they may have accumulated
#    on the guild object before asserting that the message handler rejects a
#    non-admin caller — otherwise this phase silently passes on whatever
#    cruft Phase 4 left behind.
GP1_P2_GUILD_PERMS=$(get_permission_value_for_player "${GUILD_ID}" "${PLAYER_2_ID}")
if [ "${GP1_P2_GUILD_PERMS}" != "0" ]; then
    info "GP1: stripping player_2's residual perms on guild (bits=${GP1_P2_GUILD_PERMS})"
    run_tx "GP1: revoke residual player_2 perms on guild" \
        tx structs permission-revoke-on-object "${GUILD_ID}" "${PLAYER_2_ID}" "${GP1_P2_GUILD_PERMS}" --from alice
fi
GP1_P2_GUILD_PERMS_AFTER=$(get_permission_value_for_player "${GUILD_ID}" "${PLAYER_2_ID}")
assert_eq "GP1: player_2 has no per-object perms on guild before reject test" \
    "0" "${GP1_P2_GUILD_PERMS_AFTER}"

run_tx_expect_permission_denied "GP1: reject update from non-admin caller (player_2)" \
    tx structs guild-update-primary-reactor "${GUILD_ID}" "${REACTOR_ID}" --from player_2
GP1_AFTER=$(gp1_guild_snapshot)
assert_eq "GP1: non-admin refusal preserves the guild" "${GP1_BEFORE}" "${GP1_AFTER}"

# 3. Happy path: admin (alice) targets the current reactor. Exercises the
#    full handler chain (player lookup, guild lookup, PermAdmin check, reactor
#    lookup, validator lookup, jailed check). Unchanged state alone proves
#    nothing for a no-op: follow the broadcast hash into a committed block.
run_tx "GP1: admin updates the current primary reactor" \
    tx structs guild-update-primary-reactor "${GUILD_ID}" "${REACTOR_ID}" --from alice
assert_eq "GP1: admin update delivered successfully" "0" "${LAST_TX_DELIVERED%%|*}"

# 4. Confirm the guild's primaryReactorId is intact (no drift from no-op).
GP1_AFTER=$(gp1_guild_snapshot)
assert_eq "GP1: guild unchanged after no-op update" \
    "${GP1_BEFORE}" "${GP1_AFTER}"
assert_eq "GP1: guild.primaryReactorId matches configured reactor" \
    "${REACTOR_ID}" "$(echo "${GP1_AFTER}" | jq -r '.primaryReactorId')"

fi # phase GP1

if run_phase 3850; then

# ═════════════════════════════════════════════════════════════════════════════
#  PHASE JS1: Jamming Satellite — Guided-Only Planetary Defense (v0.20.0)
# ═════════════════════════════════════════════════════════════════════════════
#
# The satellite shields planet-category targets from guided ordnance. JS1
# requires three unguided Battleship primary hits against an Ore Extractor on
# the satellite owner's planet. Each hit pins the active interception network,
# exact receipt and committed health; attrition requires fixture restoration.
# Deterministic guided/jamming and fleet-exclusion coverage remains in
# TestMsgStructAttackPlanetaryDefenseGuidedOnly.
section "PHASE JS1: Jamming Satellite — Guided-Only Planetary Defense (v0.20.0)"
js_run
# Required satellite bypass complete

fi # phase JS1


# ═════════════════════════════════════════════════════════════════════════════
#  Final State Dump & Summary
# ═════════════════════════════════════════════════════════════════════════════

section "Final State Dump"

info "All Structs:"
get_all_structs | jq -r '.Struct[] | "  \(.id) type=\(.type) owner=\(.owner) ambit=\(.operatingAmbit) loc=\(.locationType)/\(.locationId)"'

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
