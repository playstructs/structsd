"""Verify cap configuration, sticky activation and upgrade event delivery."""
import json
from verify_upgrade_bank import amount

EVENT = "structs.structs.EventReactorStakeCap"


def verify_stake_cap(before, after, blocks, upgrade_height):
    differences = []
    old = before["app_state"]["structs"].get("reactorStakeCapActive", False)
    state = after["app_state"]["structs"]
    active = state.get("reactorStakeCapActive", False)
    params = state["params"]
    cap, minimum = amount(params["reactorStakeCapBps"]), amount(params["reactorStakeCapMinValidators"])
    if type(old) is not bool or type(active) is not bool or not 1 <= cap <= 10000 or minimum == 0:
        raise ValueError("invalid exported stake cap state")
    if old and not active:
        differences.append("stake cap activation was lost")
    bonded = sum(row["status"] == "BOND_STATUS_BONDED" for row in after["app_state"]["staking"]["validators"])
    if bonded >= minimum and not active:
        differences.append("stake cap did not activate at the validator threshold")
    snapshots = []
    replay = old
    for block in blocks:
        for event in block.get("finalize_block_events") or []:
            if event["type"] != EVENT:
                continue
            data = {}
            for item in event["attributes"]:
                key = item["key"]
                if key == "mode":
                    continue
                if key in data:
                    raise ValueError("duplicate stake cap event attribute")
                data[key] = json.loads(item["value"])
            value = data.get("active", False)
            if type(value) is not bool or amount(data["capBps"]) != cap or amount(data["minValidators"]) != minimum:
                raise ValueError("invalid stake cap event configuration")
            if snapshots and replay == value:
                differences.append("redundant stake cap snapshot in migration window")
            if replay and not value:
                differences.append("stake cap event deactivated a sticky cap")
            replay = value
            snapshots.append((int(block["height"]), value))
        for tx in block.get("txs_results") or []:
            if any(event["type"] == EVENT for event in tx.get("events") or []):
                differences.append("upgrade cap snapshot must not be a transaction event")
    if not snapshots or snapshots[0][0] != upgrade_height:
        differences.append("expected initial stake cap snapshot at the upgrade height")
    if replay != active:
        differences.append("stake cap event replay differs from export")
    return {"differences": differences, "snapshots": snapshots, "cap_bps": cap,
        "activation_threshold": minimum, "active": active,
        "scope": "cap parameters, required upgrade snapshot, sticky activation and final validator threshold; not transaction admission or intermediate validator-set reconstruction"}
