"""Replay infusion records and effective grid values from public block events."""
import copy
import json

from verify_upgrade_bank import amount


def infusion_key(row):
    return row["destinationId"] + "-" + row["address"]


def infusion_record(row):
    # v0.21 exports lack fields added in v0.22. An unchanged generator need
    # not emit a replacement merely because the new encoder includes defaults.
    # Fill only documented protobuf defaults; never discard a non-default value.
    row = copy.deepcopy(row)
    for key, default in (("ownerId", ""), ("generatorDisabled", False)):
        row.setdefault(key, default)
    return row


def projection(document):
    state = document["app_state"]["structs"]
    infusions, grid, grid_keys = {}, {}, set()
    for row in state["infusionList"]:
        key = infusion_key(row)
        if key in infusions:
            raise ValueError("duplicate infusion record")
        infusions[key] = infusion_record(row)
    for row in state["gridList"]:
        key, value = row["attributeId"], amount(row["value"])
        if key in grid_keys or value >= 2**64:
            raise ValueError("duplicate or invalid grid record")
        grid_keys.add(key)
        if value:
            grid[key] = value
    return infusions, grid


def verify_grid(before, after, blocks):
    start, end = int(before["initial_height"]), int(after["initial_height"])
    if start <= 0 or end <= start or [int(block["height"]) for block in blocks] != list(range(start, end)):
        raise ValueError("block events must cover every exported boundary height exactly once")
    infusions, grid = projection(before)
    expected_infusions, expected_grid = projection(after)
    counts = {"infusion": 0, "grid": 0, "infusion_deletion": 0}
    for block in blocks:
        if block.get("txs_results"):
            raise ValueError("grid replay requires a transaction-free migration window")
        for event in block["finalize_block_events"] or []:
            kind = event["type"]
            if kind not in ("structs.structs.EventInfusion", "structs.structs.EventGrid", "structs.structs.EventDelete"):
                continue
            data, seen = {}, set()
            for item in event["attributes"]:
                key, value = item["key"], item["value"]
                if key in seen:
                    raise ValueError("duplicate grid/infusion event attribute")
                seen.add(key)
                # BaseApp's envelope mode is raw text, not protobuf JSON.
                if key == "mode":
                    if value not in ("BeginBlock", "EndBlock"):
                        raise ValueError("unknown block event mode")
                    continue
                data[key] = json.loads(value)
            if kind == "structs.structs.EventInfusion":
                row = data["infusion"]
                infusions[infusion_key(row)] = infusion_record(row)
                counts["infusion"] += 1
            elif kind == "structs.structs.EventGrid":
                row = data["gridRecord"]
                key, value = row["attributeId"], amount(row["value"])
                if value >= 2**64:
                    raise ValueError("grid event exceeds uint64")
                if value:
                    grid[key] = value
                else:
                    grid.pop(key, None)
                counts["grid"] += 1
            elif data["objectId"] in infusions:
                del infusions[data["objectId"]]
                counts["infusion_deletion"] += 1
    differences = []
    for label, replay, expected in (("infusion", infusions, expected_infusions), ("grid", grid, expected_grid)):
        for key in sorted(replay.keys() | expected.keys()):
            if replay.get(key) != expected.get(key):
                differences.append(f"{label} event replay differs: {key}")
    return {"differences": differences, "event_counts": counts,
        "scope": "complete infusion rows, effective grid values; zero grid equals absence; excludes queue storage and independent economic correctness"}
