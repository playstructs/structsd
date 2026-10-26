# Height-pinned Phase 13 receipts; sourced by test_chain.sh.
p13_attack() {
    local mode="$1" attacker="$2" target="$3" player="$4" target_player="$5" weapon="$6" defenders="${7:-}"
    local owner target_owner fleet target_fleet before_height before after height txhash receipt outcome
    local attacker_location target_location
    local a t d ds='[]' catalog kind defender
    eval "owner=\${PLAYER_${player}_ID}; fleet=\${PLAYER_${player}_FLEET_ID}"
    eval "target_owner=\${PLAYER_${target_player}_ID}; target_fleet=\${PLAYER_${target_player}_FLEET_ID}"
    wait_for_charge "${owner}" 5 || return 1
    before_height=$(query query structs block-height | jq -er '.blockHeight | select(type == "string" and test("^[1-9][0-9]*$"))') || return 1
    before=$(ar_collect_state "${attacker}" "${target}" "${defenders}" "${before_height}") || return 1
    printf '%s' "${before}" | python3 "${GENERATOR_TEST_DIR}/phase13_combat.py" before "${mode}" \
        "${attacker}" "${target}" "${owner}" "${target_owner}" "${fleet}" "${target_fleet}" >/dev/null || return 1
    attacker_location=$(combat_fleet_location "${player}" "${before_height}") || return 1
    target_location=$(combat_fleet_location "${target_player}" "${before_height}") || return 1
    [ "${attacker_location}" = "${PLAYER_2_PLANET_ID}" ] && [ "${target_location}" = "${PLAYER_2_PLANET_ID}" ] || return 1
    a=$(query query structs struct-type 2 --height "${before_height}") || return 1
    kind=$(printf '%s' "${before}" | jq -er '.target.Struct.type') || return 1
    t=$(query query structs struct-type "${kind}" --height "${before_height}") || return 1
    for kind in $(printf '%s' "${before}" | jq -r '.defenders[].Struct.type'); do
        d=$(query query structs struct-type "${kind}" --height "${before_height}") || return 1
        ds=$(jq -n --argjson ds "${ds}" --argjson d "${d}" '$ds + [$d]') || return 1
    done
    catalog=$(jq -n --argjson attacker "${a}" --argjson target "${t}" --argjson defenders "${ds}" '{attacker:$attacker,target:$target,defenders:$defenders}') || return 1
    printf '%s' "${catalog}" | python3 "${GENERATOR_TEST_DIR}/phase13_combat.py" catalog "${mode}" >/dev/null || return 1
    case "${mode}" in
        unbuilt|range) run_tx_expect_fail_noauto "P13 ${mode}: required targeting refusal" \
            tx structs struct-attack "${attacker}" "${target}" "${weapon}" --from "player_${player}" || return 1 ;;
        armour|defended) run_tx "P13 ${mode}: required attack" \
            tx structs struct-attack "${attacker}" "${target}" "${weapon}" --from "player_${player}" || return 1 ;;
        *) return 1 ;;
    esac
    txhash=$(_tx_json "${LAST_TX_OUTPUT}" | jq -er '.txhash | select(type == "string" and test("^[0-9A-Fa-f]{64}$"))') || return 1
    receipt=$(query query tx "${txhash}") || return 1
    height=$(printf '%s' "${receipt}" | jq -er '.height | select(type == "string" and test("^[1-9][0-9]*$"))') || return 1
    [ "${height}" -gt "${before_height}" ] || return 1
    after=$(ar_collect_state "${attacker}" "${target}" "${defenders}" "${height}") || return 1
    outcome=$(jq -n --argjson before "${before}" --argjson after "${after}" --argjson tx "${receipt}" '{before:$before,after:$after,tx:$tx}' \
        | python3 "${GENERATOR_TEST_DIR}/phase13_combat.py" verify "${mode}" "${txhash}") || return 1
    assert_eq "P13 ${mode}: receipt and combat state verified" true "$(printf '%s' "${outcome}" | jq -r '.verified')"
    printf 'P13 outcome: %s\n' "$(printf '%s' "${outcome}" | jq -c --arg beforeHeight "${before_height}" '. + {beforeHeight:$beforeHeight}')"
}
