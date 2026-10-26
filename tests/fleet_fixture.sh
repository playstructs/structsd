# Fleet-only fixtures use their own keys without overwriting main player state.
fp_move() {
    local number="$1" destination="$2" PLAYER_1_ID PLAYER_1_FLEET_ID PLAYER_1_PLANET_ID
    [[ "${number}" =~ ^[1-5]$ ]] || return 1
    eval "PLAYER_1_ID=\${FP_${number}_ID}; PLAYER_1_FLEET_ID=\${FP_${number}_FLEET_ID}; PLAYER_1_PLANET_ID=\${FP_${number}_PLANET_ID}"
    combat_move_fleet 1 "${destination}" "fplayer_${number}"
}

fp_move_home() {
    local number="$1" home
    [[ "${number}" =~ ^[1-5]$ ]] || return 1
    eval "home=\${FP_${number}_PLANET_ID}"
    fp_move "${number}" "${home}"
}

fp_require_command() {
    local number="$1" PLAYER_1_ID PLAYER_1_FLEET_ID PLAYER_1_PLANET_ID
    local payload id ambit=land stage output="FP_CS_${1}"
    [[ "${number}" =~ ^[1-5]$ ]] || return 1
    eval "PLAYER_1_ID=\${FP_${number}_ID}; PLAYER_1_FLEET_ID=\${FP_${number}_FLEET_ID}; PLAYER_1_PLANET_ID=\${FP_${number}_PLANET_ID}"
    payload=$(query query structs fleet "${PLAYER_1_FLEET_ID}") || return 1
    id=$(printf '%s' "${payload}" | jq -er --arg id "${PLAYER_1_FLEET_ID}" --arg owner "${PLAYER_1_ID}" \
        '.Fleet | select(.id == $id and .owner == $owner) | .commandStruct // ""') || return 1
    if [ -n "${id}" ]; then
        payload=$(query query structs struct "${id}") || return 1
        ambit=$(printf '%s' "${payload}" | jq -er '.Struct.operatingAmbit') || return 1
        stage=$(printf '%s' "${payload}" | python3 "${GENERATOR_TEST_DIR}/combat_fixture.py" fixture \
            "${id}" "${PLAYER_1_ID}" 1 "${PLAYER_1_FLEET_ID}" "${ambit}") || return 1
        if [ "${stage}" != destroyed ]; then printf -v "${output}" '%s' "${id}"; fi
    fi
    combat_require_struct "${output}" 1 1 fleet "${PLAYER_1_FLEET_ID}" "${ambit}" \
        "FP ${number} command" "" "fplayer_${number}" || return 1
    if [ "${ambit}" != land ]; then
        eval "id=\${${output}}"
        wait_for_charge "${PLAYER_1_ID}" "${CHARGE_MOVE}" || return 1
        run_tx "Fleet fixture: move FP ${number} command to land" tx structs struct-move "${id}" fleet land --from "fplayer_${number}" || return 1
        combat_require_struct "${output}" 1 1 fleet "${PLAYER_1_FLEET_ID}" land \
            "FP ${number} command" "" "fplayer_${number}" || return 1
    fi
}

fp_command_snapshot() {
    local attacker="$1" target="$2" height="$3" a t
    a=$(query query structs struct "${attacker}" --height "${height}") || return 1
    t=$(query query structs struct "${target}" --height "${height}") || return 1
    jq -n --argjson a "${a}" --argjson t "${t}" '{attacker:$a,target:$t}'
}

# Verify the delivered event and the exact health at its committed height.
# Destroyed rows may be swept before a later latest-state query.
fp_attack() {
    local shooter="$1" target="$2" label="$3" attacker victim owner target_owner fleet target_fleet
    local height before catalog hash receipt outcome after actual
    [[ "${shooter}" =~ ^[12]$ && "${target}" =~ ^[12]$ && "${shooter}" != "${target}" ]] || return 1
    eval "attacker=\${FP_CS_${shooter}}; victim=\${FP_CS_${target}}; owner=\${FP_${shooter}_ID}; target_owner=\${FP_${target}_ID}"
    eval "fleet=\${FP_${shooter}_FLEET_ID}; target_fleet=\${FP_${target}_FLEET_ID}"
    wait_for_charge "${owner}" "$(_primary_charge 1)" || return 1
    height=$(query query structs block-height | jq -er '.blockHeight | select(type == "string" and test("^[1-9][0-9]*$"))') || return 1
    before=$(fp_command_snapshot "${attacker}" "${victim}" "${height}") || return 1
    printf '%s' "${before}" | python3 "${GENERATOR_TEST_DIR}/fleet_command_attack.py" before \
        "${attacker}" "${victim}" "${owner}" "${target_owner}" "${fleet}" "${target_fleet}" >/dev/null || return 1
    catalog=$(query query structs struct-type 1 --height "${height}") || return 1
    printf '%s' "${catalog}" | python3 "${GENERATOR_TEST_DIR}/fleet_command_attack.py" catalog >/dev/null || return 1
    run_tx "${label}" tx structs struct-attack "${attacker}" "${victim}" primaryWeapon --from "fplayer_${shooter}" || return 1
    hash=$(_tx_json "${LAST_TX_OUTPUT}" | jq -er '.txhash | select(type == "string" and test("^[0-9A-Fa-f]{64}$"))') || return 1
    receipt=$(query query tx "${hash}") || return 1
    outcome=$(jq -n --argjson before "${before}" --argjson tx "${receipt}" '{before:$before,tx:$tx}' \
        | python3 "${GENERATOR_TEST_DIR}/fleet_command_attack.py" attack "${hash}") || return 1
    actual=$(printf '%s' "${outcome}" | jq -er '.height') || return 1
    [ "${actual}" -gt "${height}" ] || return 1
    after=$(fp_command_snapshot "${attacker}" "${victim}" "${actual}") || return 1
    jq -n --argjson before "${before}" --argjson after "${after}" --argjson outcome "${outcome}" \
        '{before:$before,after:$after,outcome:$outcome}' | python3 "${GENERATOR_TEST_DIR}/fleet_command_attack.py" after >/dev/null || return 1
    FP_LAST_TARGET_HEALTH=$(printf '%s' "${outcome}" | jq -er '.targetHealth') || return 1
    assert_eq "${label}: receipt and committed health" true "$(printf '%s' "${outcome}" | jq -r '.verified')"
    printf 'FP outcome: %s\n' "$(printf '%s' "${outcome}" | jq -c --arg beforeHeight "${height}" '. + {beforeHeight:$beforeHeight}')"
}
