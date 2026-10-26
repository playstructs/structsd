# Required attack-run fixtures and receipt checks, sourced by test_chain.sh.

ar_require_command() {
    local player="$1" fleet owner height payload id ambit stage output="AR_P${1}_CS_ID"
    eval "fleet=\${PLAYER_${player}_FLEET_ID}; owner=\${PLAYER_${player}_ID}"
    # The destruction sweep clears the fleet slot and the struct in one block, so
    # both reads must share a height or a sweep between them reads as an error.
    height=$(query query structs block-height | jq -er '.blockHeight | select(type == "string" and test("^[1-9][0-9]*$"))') || return 1
    payload=$(query query structs fleet "${fleet}" --height "${height}") || return 1
    id=$(printf '%s' "${payload}" | jq -er --arg id "${fleet}" --arg owner "${owner}" \
        '.Fleet | select(.id == $id and .owner == $owner) | .commandStruct // ""') || return 1
    if [ -n "${id}" ]; then
        payload=$(query query structs struct "${id}" --height "${height}") || return 1
        ambit=$(printf '%s' "${payload}" | jq -er '.Struct.operatingAmbit') || return 1
        stage=$(printf '%s' "${payload}" | python3 "${GENERATOR_TEST_DIR}/combat_fixture.py" fixture "${id}" "${owner}" 1 "${fleet}" "${ambit}") || return 1
        if [ "${stage}" != destroyed ]; then
            printf -v "${output}" '%s' "${id}"
            combat_require_struct "${output}" "${player}" 1 fleet "${fleet}" "${ambit}" "P${player} command" || return 1
            if [ "${ambit}" != space ]; then
                wait_for_charge "${owner}" "${CHARGE_MOVE}" || return 1
                run_tx "AR: move P${player} command to space" tx structs struct-move "${id}" fleet space --from "player_${player}" || return 1
            fi
        fi
    fi
    combat_require_struct "${output}" "${player}" 1 fleet "${fleet}" space "P${player} command" || return 1
    eval "id=\${${output}}"
    case "${player}" in
        2) PLAYER_2_CMD_SHIP_ID="${id}" ;;
        3) PLAYER_3_CMD_SHIP_ID="${id}"; COMMAND_SHIP_ID="${id}" ;;
        6) P6_COMMAND_SHIP_ID="${id}" ;;
    esac
}

ar_require_fighter() {
    local output="$1" player="$2" minimum="${3:-1}" fleet role number id excluded="" attempt payload hp
    case "${minimum}" in 1|2|3) ;; *) return 1 ;; esac
    eval "fleet=\${PLAYER_${player}_FLEET_ID}"
    # Preserve each named fighter across cases; replacements may not take the
    # identity of another fighter role, including a survivor with lower health.
    for number in 1 2 3; do
        role="AR_P${player}_SF${number}_ID"
        if [ "${role}" != "${output}" ]; then
            eval "id=\${${role}:-}"
            excluded="${excluded} ${id}"
        fi
    done
    # At most four pre-existing fleet slots can contain a damaged candidate.
    # Retire one through verified combat before rebuilding, so replacing a live
    # weak role never fills the fleet with abandoned hulls.
    for attempt in 1 2 3 4 5; do
        combat_require_struct "${output}" "${player}" 3 fleet "${fleet}" space "${output}" "${excluded}" || return 1
        eval "id=\${${output}}"
        hp=3
        if [ "${minimum}" -gt 1 ]; then
            payload=$(query query structs struct "${id}") || return 1
            hp=$(printf '%s' "${payload}" | jq -er --arg id "${id}" \
                'select(.Struct.id == $id) | .structAttributes.health | select(. == "1" or . == "2" or . == "3")') || return 1
        fi
        if [ "${hp}" -ge "${minimum}" ]; then
            if [ "${player}" = 6 ]; then AR_P6_SF_ID="${id}"; fi
            return 0
        fi
        ar_retire_fighter "${player}" "${id}" || return 1
    done
    echo "FAIL: ${output} could not acquire the required health" >&2
    return 1
}

ar_retire_fighter() {
    local player="$1" fighter="$2" shooter=6 visitor="$1" commander co fo cf ff
    local before_height before after ac tc catalog txhash receipt outcome height actual
    if [ "${player}" = 6 ]; then shooter=3; visitor=3; fi
    ar_require_command "${shooter}" || return 1
    ar_require_command "${player}" || return 1
    eval "commander=\${AR_P${shooter}_CS_ID}; co=\${PLAYER_${shooter}_ID}; cf=\${PLAYER_${shooter}_FLEET_ID}"
    eval "fo=\${PLAYER_${player}_ID}; ff=\${PLAYER_${player}_FLEET_ID}"
    combat_clear_target_defenders "${player}" "${fighter}" || return 1
    ar_park_as_head "${visitor}" || return 1
    wait_for_charge "${co}" "$(_primary_charge 1)" || return 1
    before_height=$(query query structs block-height | jq -er '.blockHeight') || return 1
    ar_verify_head "${visitor}" "${before_height}" || return 1
    before=$(ar_collect_state "${commander}" "${fighter}" "" "${before_height}") || return 1
    printf '%s' "${before}" | python3 "${GENERATOR_TEST_DIR}/retire_fighter.py" before \
        "${commander}" "${fighter}" "${co}" "${fo}" "${cf}" "${ff}" >/dev/null || return 1
    ac=$(query query structs struct-type 1 --height "${before_height}") || return 1
    tc=$(query query structs struct-type 3 --height "${before_height}") || return 1
    catalog=$(jq -n --argjson a "${ac}" --argjson t "${tc}" '{attacker:$a,target:$t}') || return 1
    printf '%s' "${catalog}" | python3 "${GENERATOR_TEST_DIR}/retire_fighter.py" catalog >/dev/null || return 1
    run_tx "AR preparation: retire damaged fighter ${fighter} for replacement" tx structs struct-attack \
        "${commander}" "${fighter}" primaryWeapon --from "player_${shooter}" || return 1
    txhash=$(_tx_json "${LAST_TX_OUTPUT}" | jq -er '.txhash | select(type == "string" and test("^[0-9A-Fa-f]{64}$"))') || return 1
    receipt=$(query query tx "${txhash}") || return 1
    outcome=$(jq -n --argjson before "${before}" --argjson tx "${receipt}" '{before:$before,tx:$tx}' \
        | python3 "${GENERATOR_TEST_DIR}/retire_fighter.py" attack "${txhash}") || return 1
    height=$(printf '%s' "${outcome}" | jq -er '.height') || return 1
    [ "${height}" -gt "${before_height}" ] || return 1
    after=$(ar_collect_state "${commander}" "${fighter}" "" "${height}") || return 1
    jq -n --argjson before "${before}" --argjson after "${after}" --argjson outcome "${outcome}" '{before:$before,after:$after,outcome:$outcome}' \
        | python3 "${GENERATOR_TEST_DIR}/retire_fighter.py" after >/dev/null || return 1
    ar_verify_head "${visitor}" "${height}" || return 1
    actual=$(combat_fleet_location 6 "${height}") || return 1
    [ "${actual}" = "${PLAYER_6_PLANET_ID}" ] || return 1
    assert_eq "AR preparation: damaged fighter retired with no counter damage" true "$(printf '%s' "${outcome}" | jq -r '.verified')"
    printf 'AR preparation: %s\n' "$(printf '%s' "${outcome}" | jq -c --arg beforeHeight "${before_height}" '. + {beforeHeight:$beforeHeight}')"
}

ar_park_as_head() {
    local visitor="$1" other=3 fleet owner payload
    case "${visitor}" in 2) ;; 3) other=2 ;; *) return 1 ;; esac
    combat_move_fleet 6 "${PLAYER_6_PLANET_ID}" || return 1
    eval "payload=\${PLAYER_${other}_PLANET_ID}"
    combat_move_fleet "${other}" "${payload}" || return 1
    combat_move_fleet "${visitor}" "${PLAYER_6_PLANET_ID}" || return 1
    ar_verify_head "${visitor}"
}

ar_verify_head() {
    local visitor="$1" height="${2:-}" fleet owner payload
    # Empty arrays expand as unbound under macOS Bash 3.2 with set -u; an
    # empty positional-argument list is safe on both supported Bash versions.
    if [ -n "${height}" ]; then set -- --height "${height}"; else set --; fi
    eval "fleet=\${PLAYER_${visitor}_FLEET_ID}; owner=\${PLAYER_${visitor}_ID}"
    payload=$(query query structs planet "${PLAYER_6_PLANET_ID}" "$@") || return 1
    printf '%s' "${payload}" | jq -e --arg planet "${PLAYER_6_PLANET_ID}" --arg fleet "${fleet}" '
        .Planet | .id == $planet and .locationListCount == "1" and (.locationListExtra // "0") == "0"
        and .locationListStart == $fleet and .locationListLast == $fleet' >/dev/null || return 1
    payload=$(query query structs fleet "${fleet}" "$@") || return 1
    printf '%s' "${payload}" | jq -e --arg fleet "${fleet}" --arg owner "${owner}" --arg planet "${PLAYER_6_PLANET_ID}" '
        .Fleet | .id == $fleet and .owner == $owner and .locationId == $planet
        and (.locationListForward // "") == "" and (.locationListBackward // "") == ""' >/dev/null
}

ar_defense_set() {
    local player="$1" defender="$2" target="$3" owner payload previous
    eval "owner=\${PLAYER_${player}_ID}"
    payload=$(query query structs struct "${defender}") || return 1
    previous=$(printf '%s' "${payload}" | jq -er --arg id "${defender}" --arg owner "${owner}" \
        'select(.Struct.id == $id and .Struct.owner == $owner) | .structAttributes.protectedStructIndex // "0"') || return 1
    if [ "${previous}" != 0 ] && [ "${previous}" != "${target#5-}" ]; then
        wait_for_charge "${owner}" "${CHARGE_DEFEND}" || return 1
        run_tx "AR: clear earlier defense by ${defender}" tx structs struct-defense-clear "${defender}" --from "player_${player}" || return 1
    fi
    if [ "${previous}" != "${target#5-}" ]; then
        wait_for_charge "${owner}" "${CHARGE_DEFEND}" || return 1
        run_tx "AR: ${defender} defends ${target}" tx structs struct-defense-set "${defender}" "${target}" --from "player_${player}" || return 1
    fi
    payload=$(query query structs struct "${defender}") || return 1
    printf '%s' "${payload}" | jq -e --arg id "${defender}" --arg owner "${owner}" --arg target "${target#5-}" \
        '.Struct.id == $id and .Struct.owner == $owner and .structAttributes.protectedStructIndex == $target' >/dev/null
}

# Collect every role explicitly so a query's nonzero status cannot be swallowed
# by a later JSON constructor. The ordered defender list follows the case profile.
ar_collect_state() {
    local attacker="$1" target="$2" defenders="$3" height="$4" a t d id ds='[]'
    a=$(combat_state_at_height "${attacker}" "${height}") || return 1
    t=$(combat_state_at_height "${target}" "${height}") || return 1
    for id in ${defenders}; do
        d=$(combat_state_at_height "${id}" "${height}") || return 1
        ds=$(jq -n --argjson ds "${ds}" --argjson d "${d}" '$ds + [$d]') || return 1
    done
    jq -n --argjson attacker "${a}" --argjson target "${t}" --argjson defenders "${ds}" \
        '{attacker:$attacker,target:$target,defenders:$defenders}'
}

ar_collect_catalog() {
    local plan="$1" height="$2" a t d kind kinds ds='[]'
    kind=$(printf '%s' "${plan}" | jq -er '.targetType') || return 1
    a=$(query query structs struct-type 3 --height "${height}") || return 1
    t=$(query query structs struct-type "${kind}" --height "${height}") || return 1
    kinds=$(printf '%s' "${plan}" | jq -r '.defenders[].type') || return 1
    for kind in ${kinds}; do
        d=$(query query structs struct-type "${kind}" --height "${height}") || return 1
        ds=$(jq -n --argjson ds "${ds}" --argjson d "${d}" '$ds + [$d]') || return 1
    done
    jq -n --argjson attacker "${a}" --argjson target "${t}" --argjson defenders "${ds}" \
        '{attacker:$attacker,target:$target,defenders:$defenders}'
}

ar_attack() {
    local case_id="$1" plan ap tp av tv kind visitor ao af to tf attacker target minimum
    local definitions role ambit defender defenders="" before_height before after catalog receipt outcome height txhash expected actual
    plan=$(python3 "${GENERATOR_TEST_DIR}/attack_run.py" profile "${case_id}") || return 1
    ap=$(printf '%s' "${plan}" | jq -er '.attackerPlayer') || return 1
    tp=$(printf '%s' "${plan}" | jq -er '.targetPlayer') || return 1
    av=$(printf '%s' "${plan}" | jq -er '.attackerRole') || return 1
    tv=$(printf '%s' "${plan}" | jq -er '.targetRole') || return 1
    visitor=$(printf '%s' "${plan}" | jq -er '.visitor') || return 1
    kind=$(printf '%s' "${plan}" | jq -er '.targetType') || return 1
    minimum=$(printf '%s' "${plan}" | jq -er '.attackerMinimumHealth') || return 1
    eval "ao=\${PLAYER_${ap}_ID}; af=\${PLAYER_${ap}_FLEET_ID}; to=\${PLAYER_${tp}_ID}; tf=\${PLAYER_${tp}_FLEET_ID}"
    ar_require_command "${ap}" || return 1
    ar_require_command "${tp}" || return 1
    ar_require_fighter "${av}" "${ap}" "${minimum}" || return 1
    if [ "${kind}" = 3 ]; then ar_require_fighter "${tv}" "${tp}" || return 1; fi
    eval "attacker=\${${av}}; target=\${${tv}}"
    combat_clear_target_defenders "${tp}" "${target}" || return 1
    definitions=$(printf '%s' "${plan}" | jq -r '.defenders[] | [.role,.type,.ambit] | join("|")') || return 1
    while IFS='|' read -r role kind ambit; do
        [ -n "${role}" ] || continue
        combat_require_struct "${role}" "${tp}" "${kind}" fleet "${tf}" "${ambit}" "AR ${case_id} defender" || return 1
        eval "defender=\${${role}}"
        ar_defense_set "${tp}" "${defender}" "${target}" || return 1
        defenders="${defenders} ${defender}"
    done <<EOF
${definitions}
EOF
    ar_park_as_head "${visitor}" || return 1
    wait_for_charge "${ao}" "$(_secondary_charge 3)" || return 1
    before_height=$(query query structs block-height | jq -er '.blockHeight | select(type == "string" and test("^[1-9][0-9]*$"))') || return 1
    ar_verify_head "${visitor}" "${before_height}" || return 1
    actual=$(combat_fleet_location 6 "${before_height}") || return 1
    [ "${actual}" = "${PLAYER_6_PLANET_ID}" ] || return 1
    before=$(ar_collect_state "${attacker}" "${target}" "${defenders}" "${before_height}") || return 1
    printf '%s' "${before}" | python3 "${GENERATOR_TEST_DIR}/attack_run.py" before "${case_id}" \
        "${attacker}" "${target}" "${ao}" "${to}" "${af}" "${tf}" >/dev/null || return 1
    catalog=$(ar_collect_catalog "${plan}" "${before_height}") || return 1
    printf '%s' "${catalog}" | python3 "${GENERATOR_TEST_DIR}/attack_run.py" catalog "${case_id}" >/dev/null || return 1
    AR_ATTACKS=$((AR_ATTACKS + 1))
    run_tx "AR ${case_id}: required attack run ${attacker} -> ${target}" tx structs struct-attack \
        "${attacker}" "${target}" secondaryWeapon --from "player_${ap}" || return 1
    txhash=$(_tx_json "${LAST_TX_OUTPUT}" | jq -er '.txhash | select(type == "string" and test("^[0-9A-Fa-f]{64}$"))') || return 1
    receipt=$(query query tx "${txhash}") || return 1
    outcome=$(jq -n --argjson before "${before}" --argjson tx "${receipt}" '{before:$before,tx:$tx}' \
        | python3 "${GENERATOR_TEST_DIR}/attack_run.py" attack "${case_id}" "${txhash}") || return 1
    height=$(printf '%s' "${outcome}" | jq -er '.height') || return 1
    [ "${height}" -gt "${before_height}" ] || return 1
    after=$(ar_collect_state "${attacker}" "${target}" "${defenders}" "${height}") || return 1
    jq -n --argjson before "${before}" --argjson after "${after}" --argjson outcome "${outcome}" '{before:$before,after:$after,outcome:$outcome}' \
        | python3 "${GENERATOR_TEST_DIR}/attack_run.py" after "${case_id}" >/dev/null || return 1
    # Destroying the visiting command ship causes defeat and a return home.
    expected="${PLAYER_6_PLANET_ID}"
    if [ "${tp}" = "${visitor}" ] && [ "$(printf '%s' "${plan}" | jq -r '.targetType')" = 1 ] \
        && [ "$(printf '%s' "${outcome}" | jq -r --arg id "${target}" '.outcomes[$id].destroyed')" = true ]; then
        eval "expected=\${PLAYER_${visitor}_PLANET_ID}"
    else
        ar_verify_head "${visitor}" "${height}" || return 1
    fi
    actual=$(combat_fleet_location "${visitor}" "${height}") || return 1
    [ "${actual}" = "${expected}" ] || return 1
    actual=$(combat_fleet_location 6 "${height}") || return 1
    [ "${actual}" = "${PLAYER_6_PLANET_ID}" ] || return 1
    AR_EXECUTED=$((AR_EXECUTED + 1))
    AR_DESTROYED=$((AR_DESTROYED + $(printf '%s' "${outcome}" | jq '[.outcomes[] | select(.destroyed)] | length')))
    assert_eq "AR ${case_id}: receipt and canonical combat outcome verified" true "$(printf '%s' "${outcome}" | jq -r '.verified')"
    printf 'AR outcome: %s\n' "$(printf '%s' "${outcome}" | jq -c --arg beforeHeight "${before_height}" '. + {beforeHeight:$beforeHeight}')"
}
