# Required planetary cannon sequence. Every attack checks its delivery-height state.
pd_collect() {
    local mode="$1" attacker="$2" target="$3" height="$4" role sid row result='{}'
    for role in attacker target cannon; do
        if [ "${role}" = cannon ] && [ "${mode}" = E10 ]; then continue; fi
        case "${role}" in attacker) sid="${attacker}";; target) sid="${target}";; cannon) sid="${EB_PDC_ID}";; esac
        row=$(query query structs struct "${sid}" --height "${height}") || return 1
        result=$(jq -n --argjson r "${result}" --arg role "${role}" --argjson row "${row}" '$r + {($role):$row}') || return 1
    done
    row=$(query query structs planet "${PLAYER_6_PLANET_ID}" --height "${height}") || return 1
    jq -n --argjson r "${result}" --argjson row "${row}" '$r + {planet:$row}'
}

pd_execute() {
    local mode="$1" attacker="$2" target="$3" before_height before after role kind row catalog='{}' txhash receipt height outcome actual
    combat_clear_target_defenders 6 "${target}" || return 1
    ar_park_as_head 3 || return 1
    # All three attack types need at most five charge; use the ceiling before
    # taking the snapshot so neither charging nor a fleet move splits it.
    wait_for_charge "${PLAYER_3_ID}" 5 || return 1
    before_height=$(query query structs block-height | jq -er '.blockHeight') || return 1
    ar_verify_head 3 "${before_height}" || return 1
    actual=$(combat_fleet_location 6 "${before_height}") || return 1
    [ "${actual}" = "${PLAYER_6_PLANET_ID}" ] || return 1
    before=$(pd_collect "${mode}" "${attacker}" "${target}" "${before_height}") || return 1
    printf '%s' "${before}" | python3 "${GENERATOR_TEST_DIR}/pdc_attack.py" before "${mode}" \
        "${attacker}" "${target}" "${EB_PDC_ID}" "${PLAYER_3_ID}" "${PLAYER_6_ID}" \
        "${PLAYER_3_FLEET_ID}" "${PLAYER_6_FLEET_ID}" "${PLAYER_6_PLANET_ID}" >/dev/null || return 1
    for role in attacker target cannon; do
        if [ "${role}" = cannon ] && [ "${mode}" = E10 ]; then continue; fi
        kind=$(printf '%s' "${before}" | jq -er --arg role "${role}" '.[$role].Struct.type') || return 1
        row=$(query query structs struct-type "${kind}" --height "${before_height}") || return 1
        catalog=$(jq -n --argjson c "${catalog}" --arg role "${role}" --argjson row "${row}" '$c + {($role):$row}') || return 1
    done
    printf '%s' "${catalog}" | python3 "${GENERATOR_TEST_DIR}/pdc_attack.py" catalog "${mode}" >/dev/null || return 1
    run_tx "PDC ${mode}: required primary volley" tx structs struct-attack "${attacker}" "${target}" primaryWeapon --from player_3 || return 1
    txhash=$(_tx_json "${LAST_TX_OUTPUT}" | jq -er '.txhash | select(type == "string" and test("^[0-9A-Fa-f]{64}$"))') || return 1
    receipt=$(query query tx "${txhash}") || return 1
    outcome=$(jq -n --argjson before "${before}" --argjson tx "${receipt}" '{before:$before,tx:$tx}' \
        | python3 "${GENERATOR_TEST_DIR}/pdc_attack.py" attack "${mode}" "${txhash}") || return 1
    height=$(printf '%s' "${outcome}" | jq -er '.height') || return 1
    [ "${height}" -gt "${before_height}" ] || return 1
    after=$(pd_collect "${mode}" "${attacker}" "${target}" "${height}") || return 1
    jq -n --argjson before "${before}" --argjson after "${after}" --argjson outcome "${outcome}" '{before:$before,after:$after,outcome:$outcome}' \
        | python3 "${GENERATOR_TEST_DIR}/pdc_attack.py" after >/dev/null || return 1
    ar_verify_head 3 "${height}" || return 1
    actual=$(combat_fleet_location 6 "${height}") || return 1
    [ "${actual}" = "${PLAYER_6_PLANET_ID}" ] || return 1
    PD_LAST_OUTCOME="${outcome}"
    EB_ATTACKS=$((${EB_ATTACKS:-0} + 1))
    EB_DESTROYED=$((${EB_DESTROYED:-0} + $(printf '%s' "${outcome}" | jq '[.health | .attacker,.target | select(. == "0")] | length')))
    assert_eq "PDC ${mode}: receipt, canonical health, cannon count and shield verified" true "$(printf '%s' "${outcome}" | jq -r '.verified')"
    printf 'PD outcome: %s\n' "$(printf '%s' "${outcome}" | jq -c --arg beforeHeight "${before_height}" '. + {beforeHeight:$beforeHeight}')"
}

pd_require_planet() {
    local output="$1" kind="$2" attempt hp sid round mode
    # BuildLimit=1. Retire an unsuitable live fixture through verified combat,
    # then rebuild once; missing state or unsuccessful attacks never mean skip.
    for attempt in 1 2; do
        combat_require_struct "${output}" 6 "${kind}" planet "${PLAYER_6_PLANET_ID}" land "PDC sequence planet fixture" || return 1
        sid="${!output}"
        hp=$(query query structs struct "${sid}" | jq -er '.structAttributes.health') || return 1
        if [ "${kind}" = 19 ]; then
            case "${hp}" in 2|4|6) return 0;; 1|3|5) mode=prepare_cannon;; *) return 1;; esac
        elif [ "${kind}" = 14 ]; then
            case "${hp}" in 6) return 0;; 1|2|3|4|5) mode=prepare_ore;; *) return 1;; esac
        else return 1; fi
        [ "${attempt}" = 1 ] || return 1
        for round in 1 2 3; do
            pd_execute "${mode}" "${EB_P3_MOBILE_ART_ID}" "${sid}" || return 1
            hp=$(printf '%s' "${PD_LAST_OUTCOME}" | jq -er '.health.target') || return 1
            if [ "${hp}" = 0 ]; then break; fi
        done
        [ "${hp}" = 0 ] || return 1
    done
    return 1
}

pd_run() {
    local round hp
    ar_require_command 2 || return 1
    ar_require_command 3 || return 1
    ar_require_command 6 || return 1
    combat_require_struct EB_P3_MOBILE_ART_ID 3 8 fleet "${PLAYER_3_FLEET_ID}" land "PDC immune Mobile Artillery" || return 1
    pd_require_planet EB_PDC_ID 19 || return 1
    pd_require_planet EB_ORE_EXTRACTOR_ID 14 || return 1
    combat_require_struct BATTLESHIP_1_ID 3 2 fleet "${PLAYER_3_FLEET_ID}" space "PDC counterable Battleship" || return 1
    combat_require_struct DESTROYER_STRUCT_ID 3 9 fleet "${PLAYER_3_FLEET_ID}" land "PDC killing Tank" || return 1
    combat_require_struct EB_MOBILE_ART_ID 6 8 fleet "${PLAYER_6_FLEET_ID}" land "PDC fleet target" || return 1
    # The same full-health extractor supplies the before/after-cannon contrast.
    pd_execute E1 "${BATTLESHIP_1_ID}" "${EB_ORE_EXTRACTOR_ID}" || return 1
    pd_execute E3 "${EB_P3_MOBILE_ART_ID}" "${EB_ORE_EXTRACTOR_ID}" || return 1
    pd_execute E4 "${DESTROYER_STRUCT_ID}" "${EB_MOBILE_ART_ID}" || return 1
    hp=$(printf '%s' "${PD_LAST_OUTCOME}" | jq -er '.health.cannon') || return 1
    for round in 1 2; do
        if [ "${hp}" = 2 ]; then break; fi
        pd_execute soften "${EB_P3_MOBILE_ART_ID}" "${EB_PDC_ID}" || return 1
        hp=$(printf '%s' "${PD_LAST_OUTCOME}" | jq -er '.health.cannon') || return 1
    done
    [ "${hp}" = 2 ] || return 1
    pd_execute kill "${DESTROYER_STRUCT_ID}" "${EB_PDC_ID}" || return 1
    # E1 may legitimately kill a one-health Battleship. Restore it for E10.
    combat_require_struct BATTLESHIP_1_ID 3 2 fleet "${PLAYER_3_FLEET_ID}" space "PDC aftermath Battleship" || return 1
    pd_execute E10 "${BATTLESHIP_1_ID}" "${EB_ORE_EXTRACTOR_ID}" || return 1
}
