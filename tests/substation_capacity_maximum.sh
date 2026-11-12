#!/usr/bin/env bash
# Sourced by test_chain.sh after phase 4's player/allocation setup.

capacity_maximum_assert_state() {
    local label="$1" capacity="$2" load="$3" count="$4" maximum="$5" share="$6" snapshot
    snapshot=$(query query structs substation "${CAP_MAX_STATION}")
    assert_eq "${label}: station identity" "${CAP_MAX_STATION}" "$(echo "${snapshot}" | jq -er '.Substation.id')"
    assert_eq "${label}: raw capacity" "${capacity}" "$(echo "${snapshot}" | jq -er '.gridAttributes.capacity // "0"')"
    assert_eq "${label}: outgoing load" "${load}" "$(echo "${snapshot}" | jq -er '.gridAttributes.load // "0"')"
    assert_eq "${label}: connection count" "${count}" "$(echo "${snapshot}" | jq -er '.gridAttributes.connectionCount // "0"')"
    assert_eq "${label}: configured maximum" "${maximum}" "$(echo "${snapshot}" | jq -er '.gridAttributes.connectionCapacityMaximum // "0"')"
    assert_eq "${label}: player share" "${share}" "$(echo "${snapshot}" | jq -er '.gridAttributes.connectionCapacity // "0"')"
}

capacity_maximum_assert_events() {
    local label="$1" share_writes="$2" maximum_writes="$3" txhash delivered counts
    txhash=$(_tx_json "${LAST_TX_OUTPUT}" | jq -er '.txhash | select(type == "string" and length == 64)')
    delivered=$(query query tx "${txhash}")
    # Check the delivered transaction identity/status before counting its grid
    # events. Never let a missing response count as a successful no-write test.
    counts=$(echo "${delivered}" | jq -cer --arg hash "${txhash}" --arg id "${CAP_MAX_STATION}" '
        select(.txhash == $hash and (.code == 0 or .code == "0") and (.height | tonumber) > 0)
        | [.events[] | select(.type == "structs.structs.EventGrid")
            | .attributes[] | select(.key == "gridRecord") | .value | fromjson] as $rows
        | [([$rows[] | select(.attributeId == ("6-" + $id))] | length),
           ([$rows[] | select(.attributeId == ("15-" + $id))] | length)]')
    assert_eq "${label}: connection-capacity writes" "${share_writes}" "$(echo "${counts}" | jq -er '.[0]')"
    assert_eq "${label}: maximum writes" "${maximum_writes}" "$(echo "${counts}" | jq -er '.[1]')"
}

run_substation_capacity_maximum_tests() {
    local CAP_MAX_STATION="" feeder outgoing original_station original_main_power p5_station
    original_station=$(query query structs player "${PLAYER_1_ID}" | jq -er '.Player.substationId')
    p5_station=$(query query structs player "${PLAYER_5_ID}" | jq -er '.Player.substationId // ""')
    # Player 5's phase-4 allocation lifecycle leaves it unconnected. Refuse a
    # different fixture rather than moving an unrelated player's live grid.
    if [ -n "${p5_station}" ]; then
        assert_eq "Capacity maximum fixture: Player 5 is disconnected" "" "${p5_station}"
        return 1
    fi
    original_main_power=$(query query structs allocation "${P1_ALLOC_ID}" | jq -er '.gridAttributes.power')
    temp_substation_create CAP_MAX_STATION "connection capacity maximum" || return 1
    feeder="${TEMP_ALLOC_ID}"
    capacity_maximum_assert_state "Initial unlimited station" 100000 0 0 0 90000
    run_tx "Cap station player shares at 20000" \
        tx structs substation-update-connection-capacity-maximum "${CAP_MAX_STATION}" 20000 --from alice
    capacity_maximum_assert_events "Set maximum" 1 1
    capacity_maximum_assert_state "Capped station" 100000 0 0 20000 20000
    run_tx_expect_permission_denied "Unpermissioned player cannot change station maximum" \
        tx structs substation-update-connection-capacity-maximum "${CAP_MAX_STATION}" 1 --from player_5
    capacity_maximum_assert_state "Rejected maximum leaves state unchanged" 100000 0 0 20000 20000

    run_tx "Connect owner to capped station" \
        tx structs substation-player-connect "${CAP_MAX_STATION}" "${PLAYER_1_ID}" --from alice
    capacity_maximum_assert_events "First connection stays capped" 0 0
    run_tx "Grant player connection permission on fixture" \
        tx structs permission-grant-on-object "${CAP_MAX_STATION}" "${PLAYER_5_ID}" 1024 --from alice
    run_tx "Connect second player to capped station" \
        tx structs substation-player-connect "${CAP_MAX_STATION}" "${PLAYER_5_ID}" --from player_5
    capacity_maximum_assert_events "Second connection stays capped" 0 0
    capacity_maximum_assert_state "Two connected players" 100000 0 2 20000 20000
    assert_eq "Player 5 reads from capped station" "${CAP_MAX_STATION}" "$(query query structs player "${PLAYER_5_ID}" | jq -er '.Player.substationId')"

    run_tx "Lower input while retaining enough capacity for capped shares" \
        tx structs allocation-update "${feeder}" 80000 --from alice
    capacity_maximum_assert_events "Input change stays capped" 0 0
    capacity_maximum_assert_state "Lower capped input" 80000 0 2 20000 20000
    run_tx "Add excess input without increasing player shares" \
        tx structs allocation-update "${feeder}" 100000 --from alice
    capacity_maximum_assert_events "Additional input stays capped" 0 0
    run_tx "Reserve outgoing load from capped station" \
        tx structs allocation-create "${CAP_MAX_STATION}" 60000 --controller "${PLAYER_1_ID}" --allocation-type dynamic --from alice
    outgoing=$(get_created_grid_id allocation "${CAP_MAX_STATION}" "${PLAYER_1_ID}")
    capacity_maximum_assert_events "Outgoing load reduces shares below maximum" 1 0
    capacity_maximum_assert_state "Load-limited station" 100000 60000 2 20000 15000
    run_tx "Disconnect second player to restore capped share" \
        tx structs substation-player-disconnect "${PLAYER_5_ID}" --from player_5
    capacity_maximum_assert_state "Connection count decrease" 100000 60000 1 20000 20000
    run_tx "Lower maximum below natural share" \
        tx structs substation-update-connection-capacity-maximum "${CAP_MAX_STATION}" 10000 --from alice
    capacity_maximum_assert_state "Lower maximum" 100000 60000 1 10000 10000
    run_tx "Repeat the same maximum without rewriting shares or maximum" \
        tx structs substation-update-connection-capacity-maximum "${CAP_MAX_STATION}" 10000 --from alice
    capacity_maximum_assert_events "Identical setting" 0 0
    run_tx "Raise maximum toward natural share" \
        tx structs substation-update-connection-capacity-maximum "${CAP_MAX_STATION}" 20000 --from alice
    capacity_maximum_assert_state "Raised maximum" 100000 60000 1 20000 20000
    run_tx "Remove connection capacity maximum" \
        tx structs substation-update-connection-capacity-maximum "${CAP_MAX_STATION}" 0 --from alice
    capacity_maximum_assert_events "Remove maximum" 1 1
    capacity_maximum_assert_state "Unlimited again" 100000 60000 1 0 30000
    run_tx "Remove outgoing load from unlimited station" \
        tx structs allocation-delete "${outgoing}" --from alice
    capacity_maximum_assert_state "Outgoing load removed" 100000 0 1 0 90000

    run_tx "Restore owner's original station" \
        tx structs substation-player-connect "${original_station}" "${PLAYER_1_ID}" --from alice
    temp_substation_destroy "${CAP_MAX_STATION}" "connection capacity maximum"
    run_tx "Restore original main feeder power" \
        tx structs allocation-update "${P1_ALLOC_ID}" "${original_main_power}" --from alice
    assert_eq "Capacity maximum fixture restores owner connection" "${original_station}" "$(query query structs player "${PLAYER_1_ID}" | jq -er '.Player.substationId')"
    assert_eq "Capacity maximum fixture restores Player 5 connection" "" "$(query query structs player "${PLAYER_5_ID}" | jq -er '.Player.substationId // ""')"
}
