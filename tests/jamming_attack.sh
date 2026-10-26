# Required satellite bypass against a planet target; never fall back to a fleet unit.
js_collect() {
    local height="$1" role sid row result='{}'
    for role in attacker target satellite; do
        case "${role}" in attacker) sid="${JS_ATTACKER_ID}";; target) sid="${JS_TARGET_ID}";; satellite) sid="${JS_SAT_ID}";; esac
        row=$(query query structs struct "${sid}" --height "${height}") || return 1
        result=$(jq -n --argjson r "${result}" --arg role "${role}" --argjson row "${row}" '$r + {($role):$row}') || return 1
    done
    row=$(query query structs planet "${PLAYER_6_PLANET_ID}" --height "${height}") || return 1
    jq -n --argjson r "${result}" --argjson row "${row}" '$r + {planet:$row}'
}

js_execute() {
    local before_height before after role kind row catalog='{}' txhash receipt height outcome actual
    combat_clear_target_defenders 6 "${JS_TARGET_ID}" || return 1
    ar_park_as_head 3 || return 1
    wait_for_charge "${PLAYER_3_ID}" 5 || return 1
    before_height=$(query query structs block-height | jq -er '.blockHeight') || return 1
    ar_verify_head 3 "${before_height}" || return 1
    actual=$(combat_fleet_location 6 "${before_height}") || return 1
    [ "${actual}" = "${PLAYER_6_PLANET_ID}" ] || return 1
    before=$(js_collect "${before_height}") || return 1
    printf '%s' "${before}" | python3 "${GENERATOR_TEST_DIR}/jamming_attack.py" before \
        "${JS_ATTACKER_ID}" "${JS_TARGET_ID}" "${JS_SAT_ID}" "${PLAYER_3_ID}" "${PLAYER_6_ID}" \
        "${PLAYER_3_FLEET_ID}" "${PLAYER_6_PLANET_ID}" >/dev/null || return 1
    for role in attacker target satellite; do
        kind=$(printf '%s' "${before}" | jq -er --arg role "${role}" '.[$role].Struct.type') || return 1
        row=$(query query structs struct-type "${kind}" --height "${before_height}") || return 1
        catalog=$(jq -n --argjson c "${catalog}" --arg role "${role}" --argjson row "${row}" '$c + {($role):$row}') || return 1
    done
    printf '%s' "${catalog}" | python3 "${GENERATOR_TEST_DIR}/jamming_attack.py" catalog >/dev/null || return 1
    run_tx "JS1: required primary volley" tx structs struct-attack "${JS_ATTACKER_ID}" "${JS_TARGET_ID}" primaryWeapon --from player_3 || return 1
    txhash=$(_tx_json "${LAST_TX_OUTPUT}" | jq -er '.txhash | select(type == "string" and test("^[0-9A-Fa-f]{64}$"))') || return 1
    receipt=$(query query tx "${txhash}") || return 1
    outcome=$(jq -n --argjson before "${before}" --argjson tx "${receipt}" '{before:$before,tx:$tx}' \
        | python3 "${GENERATOR_TEST_DIR}/jamming_attack.py" attack "${txhash}") || return 1
    height=$(printf '%s' "${outcome}" | jq -er '.height') || return 1
    [ "${height}" -gt "${before_height}" ] || return 1
    after=$(js_collect "${height}") || return 1
    jq -n --argjson before "${before}" --argjson after "${after}" --argjson outcome "${outcome}" '{before:$before,after:$after,outcome:$outcome}' \
        | python3 "${GENERATOR_TEST_DIR}/jamming_attack.py" after >/dev/null || return 1
    ar_verify_head 3 "${height}" || return 1
    actual=$(combat_fleet_location 6 "${height}") || return 1
    [ "${actual}" = "${PLAYER_6_PLANET_ID}" ] || return 1
    JS_LAST_OUTCOME="${outcome}"
    EB_ATTACKS=$((${EB_ATTACKS:-0} + 1))
    EB_DESTROYED=$((${EB_DESTROYED:-0} + $(printf '%s' "${outcome}" | jq '[.health | .attacker,.target | select(. == "0")] | length')))
    assert_eq "JS1: receipt, canonical health and active satellite network verified" true "$(printf '%s' "${outcome}" | jq -r '.verified')"
    printf 'JS outcome: %s\n' "$(printf '%s' "${outcome}" | jq -c --arg beforeHeight "${before_height}" '. + {beforeHeight:$beforeHeight}')"
}

js_run() {
    local round
    ar_require_command 2 || return 1
    ar_require_command 3 || return 1
    ar_require_command 6 || return 1
    combat_require_struct JS_SAT_ID 6 17 planet "${PLAYER_6_PLANET_ID}" space "JS1 active Jamming Satellite" || return 1
    JS_SHOTS=0
    for round in 1 2 3; do
        combat_require_struct JS_ATTACKER_ID 3 2 fleet "${PLAYER_3_FLEET_ID}" space "JS1 unguided Battleship" || return 1
        combat_require_struct JS_TARGET_ID 6 14 planet "${PLAYER_6_PLANET_ID}" land "JS1 planet Ore Extractor" || return 1
        js_execute || return 1
        JS_SHOTS=$((JS_SHOTS + 1))
    done
    [ "${JS_SHOTS}" -eq 3 ] || return 1
}
