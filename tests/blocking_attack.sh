# Required original C2 matchup; evasion and lethal early counters are not blocks.
bc_collect() {
    local height="$1" payload
    payload=$(ar_collect_state "${BATTLESHIP_2_ID}" "${EB_P6_BATTLESHIP_ID}" "${EB_FRIGATE_ID} ${EB_STARFIGHTER_ID}" "${height}") || return 1
    printf '%s' "${payload}" | jq -e 'select(.defenders | length == 2) | . + {frigate:.defenders[0],fighter:.defenders[1]} | del(.defenders)'
}

bc_attack() {
    local before_height before after role kind row catalog='{}' txhash receipt height outcome actual
    wait_for_charge "${PLAYER_3_ID}" "$(_secondary_charge 2)" || return 1
    before_height=$(query query structs block-height | jq -er '.blockHeight') || return 1
    ar_verify_head 3 "${before_height}" || return 1
    actual=$(combat_fleet_location 6 "${before_height}") || return 1
    [ "${actual}" = "${PLAYER_6_PLANET_ID}" ] || return 1
    before=$(bc_collect "${before_height}") || return 1
    printf '%s' "${before}" | python3 "${GENERATOR_TEST_DIR}/blocking_attack.py" before \
        "${BATTLESHIP_2_ID}" "${EB_P6_BATTLESHIP_ID}" "${EB_FRIGATE_ID}" "${EB_STARFIGHTER_ID}" \
        "${PLAYER_3_ID}" "${PLAYER_6_ID}" "${PLAYER_3_FLEET_ID}" "${PLAYER_6_FLEET_ID}" >/dev/null || return 1
    for role in attacker target frigate fighter; do
        kind=$(printf '%s' "${before}" | jq -er --arg role "${role}" '.[$role].Struct.type') || return 1
        row=$(query query structs struct-type "${kind}" --height "${before_height}") || return 1
        catalog=$(jq -n --argjson c "${catalog}" --arg role "${role}" --argjson row "${row}" '$c + {($role):$row}') || return 1
    done
    printf '%s' "${catalog}" | python3 "${GENERATOR_TEST_DIR}/blocking_attack.py" catalog >/dev/null || return 1
    run_tx "C2: Battleship attacks Battleship defended by Frigate and Starfighter" tx structs struct-attack \
        "${BATTLESHIP_2_ID}" "${EB_P6_BATTLESHIP_ID}" secondaryWeapon --from player_3 || return 1
    txhash=$(_tx_json "${LAST_TX_OUTPUT}" | jq -er '.txhash | select(type == "string" and test("^[0-9A-Fa-f]{64}$"))') || return 1
    receipt=$(query query tx "${txhash}") || return 1
    outcome=$(jq -n --argjson before "${before}" --argjson tx "${receipt}" '{before:$before,tx:$tx}' \
        | python3 "${GENERATOR_TEST_DIR}/blocking_attack.py" attack "${txhash}") || return 1
    height=$(printf '%s' "${outcome}" | jq -er '.height') || return 1
    [ "${height}" -gt "${before_height}" ] || return 1
    after=$(bc_collect "${height}") || return 1
    jq -n --argjson before "${before}" --argjson after "${after}" --argjson outcome "${outcome}" '{before:$before,after:$after,outcome:$outcome}' \
        | python3 "${GENERATOR_TEST_DIR}/blocking_attack.py" after >/dev/null || return 1
    ar_verify_head 3 "${height}" || return 1
    actual=$(combat_fleet_location 6 "${height}") || return 1
    [ "${actual}" = "${PLAYER_6_PLANET_ID}" ] || return 1
    BC_LAST_OUTCOME="${outcome}"
    EB_ATTACKS=$((${EB_ATTACKS:-0} + 1))
    EB_DESTROYED=$((${EB_DESTROYED:-0} + $(printf '%s' "${outcome}" | jq '[.health[] | select(. == "0")] | length')))
    assert_eq "C2: receipt and canonical blocking outcome verified" true "$(printf '%s' "${outcome}" | jq -r '.verified')"
    printf 'BC outcome: %s\n' "$(printf '%s' "${outcome}" | jq -c --arg beforeHeight "${before_height}" '. + {beforeHeight:$beforeHeight}')"
}

bc_run() {
    local attempt
    ar_require_command 2 || return 1
    ar_require_command 3 || return 1
    ar_require_command 6 || return 1
    # Restoring after attrition is mandatory. Every attempt must be validated;
    # exhaustion is a failure, never evidence that a defender blocked.
    for attempt in {1..32}; do
        combat_require_struct BATTLESHIP_2_ID 3 2 fleet "${PLAYER_3_FLEET_ID}" space "C2 attacking Battleship" || return 1
        combat_require_struct EB_P6_BATTLESHIP_ID 6 2 fleet "${PLAYER_6_FLEET_ID}" space "C2 target Battleship" || return 1
        combat_require_struct EB_FRIGATE_ID 6 4 fleet "${PLAYER_6_FLEET_ID}" space "C2 Frigate defender" || return 1
        combat_require_struct EB_STARFIGHTER_ID 6 3 fleet "${PLAYER_6_FLEET_ID}" space "C2 Starfighter defender" || return 1
        combat_clear_target_defenders 6 "${EB_P6_BATTLESHIP_ID}" || return 1
        ar_defense_set 6 "${EB_FRIGATE_ID}" "${EB_P6_BATTLESHIP_ID}" || return 1
        ar_defense_set 6 "${EB_STARFIGHTER_ID}" "${EB_P6_BATTLESHIP_ID}" || return 1
        ar_park_as_head 3 || return 1
        bc_attack || return 1
        if printf '%s' "${BC_LAST_OUTCOME}" | jq -e '.verified == true and .blocked == true' >/dev/null; then
            combat_clear_target_defenders 6 "${EB_P6_BATTLESHIP_ID}" || return 1
            return 0
        fi
    done
    echo "FAIL: C2 did not observe a block in 32 verified attempts" >&2
    return 1
}
