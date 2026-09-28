# Required blocker-before-lethal-counter regression and bounded fixture recovery.
da_collect() {
    local mode="$1" attacker="$2" target="$3" height="$4" payload defenders=""
    if [ "${mode}" = regression ]; then defenders="${EB_P6_TANK_ID} ${EB_P6_HAI_ID}"; fi
    payload=$(ar_collect_state "${attacker}" "${target}" "${defenders}" "${height}") || return 1
    if [ "${mode}" = regression ]; then
        printf '%s' "${payload}" | jq -e 'select(.defenders | length == 2) | {bomber:.attacker,target:.target,tank:.defenders[0],hai:.defenders[1]}'
    else
        printf '%s' "${payload}" | jq 'del(.defenders)'
    fi
}

da_execute() {
    local mode="$1" attacker="$2" target="$3" ap="$4" tp="$5" ao to af tf kind role roles row
    local before_height before after catalog='{}' receipt txhash height outcome actual weapon=primaryWeapon charge
    eval "ao=\${PLAYER_${ap}_ID}; af=\${PLAYER_${ap}_FLEET_ID}; to=\${PLAYER_${tp}_ID}; tf=\${PLAYER_${tp}_FLEET_ID}"
    case "${mode}" in
        regression) kind=6; roles="bomber target tank hai" ;;
        bomber) kind=7; roles="attacker target" ;;
        tank|hai|command) kind=1; roles="attacker target" ;;
        armour) kind=9; roles="attacker target" ;;
        sustained) kind=2; roles="attacker target"; weapon=secondaryWeapon ;;
        *) return 1 ;;
    esac
    if [ "${weapon}" = primaryWeapon ]; then charge=$(_primary_charge "${kind}"); else charge=$(_secondary_charge "${kind}"); fi
    wait_for_charge "${ao}" "${charge}" || return 1
    before_height=$(query query structs block-height | jq -er '.blockHeight | select(type == "string" and test("^[1-9][0-9]*$"))') || return 1
    ar_verify_head 3 "${before_height}" || return 1
    actual=$(combat_fleet_location 6 "${before_height}") || return 1
    [ "${actual}" = "${PLAYER_6_PLANET_ID}" ] || return 1
    before=$(da_collect "${mode}" "${attacker}" "${target}" "${before_height}") || return 1
    if [ "${mode}" = regression ]; then
        set -- "${attacker}" "${target}" "${EB_P6_TANK_ID}" "${EB_P6_HAI_ID}" "${ao}" "${to}" "${af}" "${tf}"
    else
        set -- "${attacker}" "${target}" "${ao}" "${to}" "${af}" "${tf}"
    fi
    printf '%s' "${before}" | python3 "${GENERATOR_TEST_DIR}/dead_bomber.py" before "${mode}" "$@" >/dev/null || return 1
    for role in ${roles}; do
        kind=$(printf '%s' "${before}" | jq -er --arg role "${role}" '.[$role].Struct.type') || return 1
        row=$(query query structs struct-type "${kind}" --height "${before_height}") || return 1
        catalog=$(jq -n --argjson c "${catalog}" --arg role "${role}" --argjson row "${row}" '$c + {($role):$row}') || return 1
    done
    printf '%s' "${catalog}" | python3 "${GENERATOR_TEST_DIR}/dead_bomber.py" catalog "${mode}" >/dev/null || return 1
    run_tx "Dead-bomber ${mode}: ${attacker} attacks ${target}" tx structs struct-attack \
        "${attacker}" "${target}" "${weapon}" --from "player_${ap}" || return 1
    txhash=$(_tx_json "${LAST_TX_OUTPUT}" | jq -er '.txhash | select(type == "string" and test("^[0-9A-Fa-f]{64}$"))') || return 1
    receipt=$(query query tx "${txhash}") || return 1
    outcome=$(jq -n --argjson before "${before}" --argjson tx "${receipt}" '{before:$before,tx:$tx}' \
        | python3 "${GENERATOR_TEST_DIR}/dead_bomber.py" attack "${mode}" "${txhash}") || return 1
    height=$(printf '%s' "${outcome}" | jq -er '.height') || return 1
    [ "${height}" -gt "${before_height}" ] || return 1
    after=$(da_collect "${mode}" "${attacker}" "${target}" "${height}") || return 1
    jq -n --argjson before "${before}" --argjson after "${after}" --argjson outcome "${outcome}" '{before:$before,after:$after,outcome:$outcome}' \
        | python3 "${GENERATOR_TEST_DIR}/dead_bomber.py" after >/dev/null || return 1
    # Retiring a defender can counter-kill a weakened visiting command. Its
    # defeat must return the fleet home and clear the planet's visitor list.
    if { [ "${mode}" = tank ] || [ "${mode}" = hai ] || [ "${mode}" = command ]; } && [ "$(printf '%s' "${outcome}" | jq -r '.health.attacker')" = 0 ]; then
        actual=$(combat_fleet_location 3 "${height}") || return 1
        [ "${actual}" = "${PLAYER_3_PLANET_ID}" ] || return 1
        row=$(query query structs planet "${PLAYER_6_PLANET_ID}" --height "${height}") || return 1
        printf '%s' "${row}" | jq -e --arg id "${PLAYER_6_PLANET_ID}" '.Planet | .id == $id
            and (.locationListCount // "0") == "0" and (.locationListExtra // "0") == "0"
            and (.locationListStart // "") == "" and (.locationListLast // "") == ""' >/dev/null || return 1
    else
        ar_verify_head 3 "${height}" || return 1
    fi
    actual=$(combat_fleet_location 6 "${height}") || return 1
    [ "${actual}" = "${PLAYER_6_PLANET_ID}" ] || return 1
    DA_LAST_OUTCOME="${outcome}"
    if [ "${mode}" = regression ] || [ "${mode}" = armour ] || [ "${mode}" = sustained ] || [ "${mode}" = command ]; then
        EB_ATTACKS=$((${EB_ATTACKS:-0} + 1))
        EB_DESTROYED=$((${EB_DESTROYED:-0} + $(printf '%s' "${outcome}" | jq '[.health[] | select(. == "0")] | length')))
    fi
    assert_eq "Dead-bomber ${mode}: receipt and canonical outcome verified" true "$(printf '%s' "${outcome}" | jq -r '.verified')"
    printf 'DA outcome: %s\n' "$(printf '%s' "${outcome}" | jq -c --arg beforeHeight "${before_height}" '. + {beforeHeight:$beforeHeight}')"
}

# Submit the off-ambit attempt with fixed gas so the handler refusal is included
# in a block. Require its exact code/reason and unchanged pinned combat state.
da_command_range() {
    local attacker="$1" target="$2" before_height before after ac tc catalog output outcome txhash receipt height
    wait_for_charge "${PLAYER_3_ID}" "${CHARGE_MOVE}" || return 1
    run_tx "C5: move command to land for range refusal" tx structs struct-move "${attacker}" fleet land --from player_3 || return 1
    wait_for_charge "${PLAYER_3_ID}" "$(_primary_charge 1)" || return 1
    before_height=$(query query structs block-height | jq -er '.blockHeight') || return 1
    ar_verify_head 3 "${before_height}" || return 1
    before=$(da_collect command "${attacker}" "${target}" "${before_height}") || return 1
    printf '%s' "${before}" | python3 "${GENERATOR_TEST_DIR}/dead_bomber.py" range_before \
        "${attacker}" "${target}" "${PLAYER_3_ID}" "${PLAYER_6_ID}" "${PLAYER_3_FLEET_ID}" "${PLAYER_6_FLEET_ID}" >/dev/null || return 1
    ac=$(query query structs struct-type 1 --height "${before_height}") || return 1
    tc=$(query query structs struct-type 3 --height "${before_height}") || return 1
    catalog=$(jq -n --argjson a "${ac}" --argjson t "${tc}" '{attacker:$a,target:$t}') || return 1
    printf '%s' "${catalog}" | python3 "${GENERATOR_TEST_DIR}/dead_bomber.py" catalog command >/dev/null || return 1
    output=$(structsd ${PARAMS_TX_NOAUTO} tx structs struct-attack "${attacker}" "${target}" primaryWeapon --from player_3 2>&1) || true
    outcome=$(_tx_outcome "${output}") || return 1
    [ "${outcome%%|*}" = rejected ] || return 1
    txhash=$(_tx_json "${output}" | jq -er '.txhash | select(type == "string" and test("^[0-9A-Fa-f]{64}$"))') || return 1
    receipt=$(query query tx "${txhash}") || return 1
    height=$(printf '%s' "${receipt}" | jq -er '.height | select(type == "string" and test("^[1-9][0-9]*$"))') || return 1
    [ "${height}" -gt "${before_height}" ] || return 1
    after=$(da_collect command "${attacker}" "${target}" "${height}") || return 1
    outcome=$(jq -n --argjson before "${before}" --argjson after "${after}" --argjson tx "${receipt}" '{before:$before,after:$after,tx:$tx}' \
        | python3 "${GENERATOR_TEST_DIR}/dead_bomber.py" range_refusal "${txhash}") || return 1
    ar_verify_head 3 "${height}" || return 1
    assert_eq "C5: delivered range refusal preserves combat state" true "$(printf '%s' "${outcome}" | jq -r '.verified')"
    printf 'DA range: %s\n' "$(printf '%s' "${outcome}" | jq -c --arg beforeHeight "${before_height}" '. + {beforeHeight:$beforeHeight}')"
}

da_command() {
    ar_require_command 2 || return 1
    ar_require_command 3 || return 1
    ar_require_command 6 || return 1
    ar_require_fighter EB_STARFIGHTER_ID 6 3 || return 1
    combat_clear_target_defenders 6 "${EB_STARFIGHTER_ID}" || return 1
    ar_park_as_head 3 || return 1
    da_command_range "${PLAYER_3_CMD_SHIP_ID}" "${EB_STARFIGHTER_ID}" || return 1
    # Return to the target's ambit, then require an actual hit and counter.
    ar_require_command 3 || return 1
    da_execute command "${PLAYER_3_CMD_SHIP_ID}" "${EB_STARFIGHTER_ID}" 3 6 || return 1
    printf '%s' "${DA_LAST_OUTCOME}" | jq -e --arg id "${EB_STARFIGHTER_ID}" \
        '.verified == true and .mode == "command" and .roles.target == $id and .health.target == "1"' >/dev/null || return 1
    # A lethal counter defeats the visiting fleet. Restore its command and
    # position for the following planetary-defense tests after verifying defeat.
    ar_require_command 3 || return 1
    ar_park_as_head 3 || return 1
}

# Required C4: start with three health and confirm both hits against one target.
# A counter-killed attacker is replaced before the second shot, never a reason
# to accept an incomplete battle. The second hit must destroy the same fighter.
da_sustained() {
    local expected
    ar_require_command 2 || return 1
    ar_require_command 3 || return 1
    ar_require_command 6 || return 1
    ar_require_fighter EB_STARFIGHTER_ID 6 3 || return 1
    combat_clear_target_defenders 6 "${EB_STARFIGHTER_ID}" || return 1
    for expected in 1 0; do
        combat_require_struct BATTLESHIP_1_ID 3 2 fleet "${PLAYER_3_FLEET_ID}" space "C4 attacking Battleship" || return 1
        ar_park_as_head 3 || return 1
        da_execute sustained "${BATTLESHIP_1_ID}" "${EB_STARFIGHTER_ID}" 3 6 || return 1
        printf '%s' "${DA_LAST_OUTCOME}" | jq -e --arg id "${EB_STARFIGHTER_ID}" --arg hp "${expected}" \
            '.verified == true and .mode == "sustained" and .roles.target == $id and .health.target == $hp' >/dev/null || return 1
    done
}

# Required C3: armor reduces the Tank's two-damage shot to exactly one.
# Existing hulls may be wounded, so both lethal and surviving results are
# checked rather than silently skipping or requiring a spare fleet slot.
da_armour() {
    ar_require_command 2 || return 1
    ar_require_command 3 || return 1
    ar_require_command 6 || return 1
    combat_require_struct DESTROYER_STRUCT_ID 3 9 fleet "${PLAYER_3_FLEET_ID}" land "C3 attacking Tank" || return 1
    combat_require_struct EB_P6_TANK_ID 6 9 fleet "${PLAYER_6_FLEET_ID}" land "C3 defending Tank" || return 1
    combat_clear_target_defenders 6 "${EB_P6_TANK_ID}" || return 1
    ar_park_as_head 3 || return 1
    da_execute armour "${DESTROYER_STRUCT_ID}" "${EB_P6_TANK_ID}" 3 6 || return 1
}

da_retire() {
    local mode="$1" victim="$2" ambit round
    case "${mode}" in tank) ambit=land ;; hai) ambit=air ;; *) return 1 ;; esac
    combat_clear_target_defenders 6 "${victim}" || return 1
    # Tank armor permits one damage per guided hit; every other defender takes
    # two. Three rounds suffice even if a counter kills the command between hits.
    for round in 1 2 3; do
        ar_require_command 3 || return 1
        wait_for_charge "${PLAYER_3_ID}" "${CHARGE_MOVE}" || return 1
        run_tx "Dead-bomber preparation: command to ${ambit}" tx structs struct-move "${PLAYER_3_CMD_SHIP_ID}" fleet "${ambit}" --from player_3 || return 1
        ar_park_as_head 3 || return 1
        da_execute "${mode}" "${PLAYER_3_CMD_SHIP_ID}" "${victim}" 3 6 || return 1
        if [ "$(printf '%s' "${DA_LAST_OUTCOME}" | jq -r '.health.target')" = 0 ]; then
            ar_require_command 3 || return 1
            return 0
        fi
    done
    echo "FAIL: Dead-bomber defender retirement exceeded three attacks" >&2
    return 1
}

da_ordered_defenders() {
    local attempt sorted
    # Reuse existing roles first. If the HAI sorts too early, replace it. If
    # decimal ID width changed (5-99 versus 5-100), replace both in Tank/HAI order.
    for attempt in 1 2 3; do
        combat_require_struct EB_P6_TANK_ID 6 9 fleet "${PLAYER_6_FLEET_ID}" land "Dead-bomber Tank" || return 1
        combat_require_struct EB_P6_HAI_ID 6 7 fleet "${PLAYER_6_FLEET_ID}" air "Dead-bomber HAI" || return 1
        sorted=$(python3 "${GENERATOR_TEST_DIR}/dead_bomber.py" ordered "${EB_P6_TANK_ID}" "${EB_P6_HAI_ID}") || return 1
        [ "${sorted}" = true ] && return 0
        [ "${attempt}" -lt 3 ] || break
        if [ "${attempt}" = 2 ]; then da_retire tank "${EB_P6_TANK_ID}" || return 1; fi
        da_retire hai "${EB_P6_HAI_ID}" || return 1
    done
    echo "FAIL: Dead-bomber defenders could not be put in byte-sorted order" >&2
    return 1
}

da_require_bomber() {
    local attempt payload hidden hp
    # At most four pre-existing air slots plus a replacement. A two-health
    # bomber is retired; three health is softened to one; one is used directly.
    for attempt in 1 2 3 4 5; do
        combat_require_struct STEALTH_BOMBER_ID 3 6 fleet "${PLAYER_3_FLEET_ID}" air "Dead-bomber attacker" || return 1
        payload=$(query query structs struct "${STEALTH_BOMBER_ID}") || return 1
        hidden=$(printf '%s' "${payload}" | jq -er --arg id "${STEALTH_BOMBER_ID}" 'select(.Struct.id == $id) | .structAttributes
            | (if has("isHidden") then .isHidden else false end) | select(type == "boolean") | tostring') || return 1
        if [ "${hidden}" = true ]; then
            wait_for_charge "${PLAYER_3_ID}" "${CHARGE_ACTIVATE}" || return 1
            run_tx "Dead-bomber preparation: reveal bomber" tx structs struct-stealth-deactivate "${STEALTH_BOMBER_ID}" --from player_3 || return 1
        fi
        hp=$(printf '%s' "${payload}" | jq -er '.structAttributes.health | select(. == "1" or . == "2" or . == "3")') || return 1
        combat_clear_target_defenders 3 "${STEALTH_BOMBER_ID}" || return 1
        ar_park_as_head 3 || return 1
        if [ "${hp}" = 1 ]; then return 0; fi
        da_execute bomber "${EB_P6_HAI_ID}" "${STEALTH_BOMBER_ID}" 6 3 || return 1
        if [ "$(printf '%s' "${DA_LAST_OUTCOME}" | jq -r '.health.target')" = 1 ]; then return 0; fi
    done
    echo "FAIL: Could not prepare a one-health bomber" >&2
    return 1
}

da_run() {
    ar_require_command 2 || return 1
    ar_require_command 3 || return 1
    ar_require_command 6 || return 1
    combat_require_struct EB_MOBILE_ART_ID 6 8 fleet "${PLAYER_6_FLEET_ID}" land "Dead-bomber target" || return 1
    da_ordered_defenders || return 1
    da_require_bomber || return 1
    combat_clear_target_defenders 6 "${EB_MOBILE_ART_ID}" || return 1
    ar_defense_set 6 "${EB_P6_TANK_ID}" "${EB_MOBILE_ART_ID}" || return 1
    ar_defense_set 6 "${EB_P6_HAI_ID}" "${EB_MOBILE_ART_ID}" || return 1
    ar_park_as_head 3 || return 1
    da_execute regression "${STEALTH_BOMBER_ID}" "${EB_MOBILE_ART_ID}" 3 6 || return 1
    combat_clear_target_defenders 6 "${EB_MOBILE_ART_ID}" || return 1
}
