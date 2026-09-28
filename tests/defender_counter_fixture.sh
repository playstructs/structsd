# Required RG1 fixture; shared movement/build helpers are in attack_run.sh.
rg1_collect_state() {
    local height="$1" payload
    payload=$(ar_collect_state "${RG1_ATTACKER_ID}" "${RG1_TARGET_ID}" "${RG1_DEFENDER_ID}" "${height}") || return 1
    printf '%s' "${payload}" | jq -e 'select(.defenders | length == 1) | . + {defender:.defenders[0]} | del(.defenders)'
}

rg1_attack() {
    local mode="$1" shooter="${RG1_ATTACKER_ID}" receiver="${RG1_TARGET_ID}" player=6 owner="${PLAYER_6_ID}"
    local before_height before catalog sf bb receipt txhash outcome height after actual
    case "${mode}" in
        prepare) shooter="${RG1_TARGET_ID}"; receiver="${RG1_ATTACKER_ID}"; player=3; owner="${PLAYER_3_ID}" ;;
        regression) ;;
        *) return 1 ;;
    esac
    wait_for_charge "${owner}" "$(_primary_charge 3)" || return 1
    before_height=$(query query structs block-height | jq -er '.blockHeight | select(type == "string" and test("^[1-9][0-9]*$"))') || return 1
    ar_verify_head 3 "${before_height}" || return 1
    actual=$(combat_fleet_location 6 "${before_height}") || return 1
    [ "${actual}" = "${PLAYER_6_PLANET_ID}" ] || return 1
    before=$(rg1_collect_state "${before_height}") || return 1
    printf '%s' "${before}" | python3 "${GENERATOR_TEST_DIR}/defender_counter_fixture.py" before "${mode}" \
        "${RG1_ATTACKER_ID}" "${RG1_TARGET_ID}" "${RG1_DEFENDER_ID}" "${PLAYER_6_ID}" "${PLAYER_3_ID}" \
        "${PLAYER_6_FLEET_ID}" "${PLAYER_3_FLEET_ID}" >/dev/null || return 1
    sf=$(query query structs struct-type 3 --height "${before_height}") || return 1
    bb=$(query query structs struct-type 2 --height "${before_height}") || return 1
    catalog=$(jq -n --argjson sf "${sf}" --argjson bb "${bb}" '{fighter:$sf,defender:$bb}') || return 1
    printf '%s' "${catalog}" | python3 "${GENERATOR_TEST_DIR}/defender_counter_fixture.py" catalog >/dev/null || return 1
    run_tx "RG1 ${mode}: ${shooter} attacks ${receiver}" tx structs struct-attack \
        "${shooter}" "${receiver}" primaryWeapon --from "player_${player}" || return 1
    txhash=$(_tx_json "${LAST_TX_OUTPUT}" | jq -er '.txhash | select(type == "string" and test("^[0-9A-Fa-f]{64}$"))') || return 1
    receipt=$(query query tx "${txhash}") || return 1
    outcome=$(jq -n --argjson before "${before}" --argjson tx "${receipt}" '{before:$before,tx:$tx}' \
        | python3 "${GENERATOR_TEST_DIR}/defender_counter_fixture.py" attack "${mode}" "${txhash}") || return 1
    height=$(printf '%s' "${outcome}" | jq -er '.height') || return 1
    [ "${height}" -gt "${before_height}" ] || return 1
    after=$(rg1_collect_state "${height}") || return 1
    jq -n --argjson before "${before}" --argjson after "${after}" --argjson outcome "${outcome}" '{before:$before,after:$after,outcome:$outcome}' \
        | python3 "${GENERATOR_TEST_DIR}/defender_counter_fixture.py" after >/dev/null || return 1
    ar_verify_head 3 "${height}" || return 1
    actual=$(combat_fleet_location 6 "${height}") || return 1
    [ "${actual}" = "${PLAYER_6_PLANET_ID}" ] || return 1
    assert_eq "RG1 ${mode}: exact receipt and canonical health verified" true "$(printf '%s' "${outcome}" | jq -r '.verified')"
    printf 'RG1 outcome: %s\n' "$(printf '%s' "${outcome}" | jq -c --arg beforeHeight "${before_height}" '. + {beforeHeight:$beforeHeight}')"
}

rg1_run() {
    local payload hp
    ar_require_command 2 || return 1
    ar_require_command 3 || return 1
    ar_require_command 6 || return 1
    ar_require_fighter AR_P6_SF1_ID 6 || return 1
    RG1_ATTACKER_ID="${AR_P6_SF1_ID}"
    payload=$(query query structs struct "${RG1_ATTACKER_ID}") || return 1
    hp=$(printf '%s' "${payload}" | jq -er --arg id "${RG1_ATTACKER_ID}" \
        'select(.Struct.id == $id) | .structAttributes.health | select(. == "1" or . == "2" or . == "3")') || return 1
    # Two health cannot be reduced to one by this guided weapon. Replace that
    # fighter using the bounded, verified retirement path before preparing it.
    if [ "${hp}" = 2 ]; then
        ar_require_fighter AR_P6_SF1_ID 6 3 || return 1
        RG1_ATTACKER_ID="${AR_P6_SF1_ID}"
        hp=3
    fi
    if [ "${hp}" = 3 ]; then
        ar_require_fighter AR_P3_SF1_ID 3 3 || return 1
    else
        ar_require_fighter AR_P3_SF1_ID 3 || return 1
    fi
    RG1_TARGET_ID="${AR_P3_SF1_ID}"
    combat_require_struct RG1_DEFENDER_ID 3 2 fleet "${PLAYER_3_FLEET_ID}" space "RG1 defender" || return 1
    combat_clear_target_defenders 6 "${RG1_ATTACKER_ID}" || return 1
    combat_clear_target_defenders 3 "${RG1_TARGET_ID}" || return 1
    ar_park_as_head 3 || return 1
    if [ "${hp}" = 3 ]; then rg1_attack prepare || return 1; fi
    ar_defense_set 3 "${RG1_DEFENDER_ID}" "${RG1_TARGET_ID}" || return 1
    rg1_attack regression || return 1
    combat_clear_target_defenders 3 "${RG1_TARGET_ID}" || return 1
}
