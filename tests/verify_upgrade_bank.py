#!/usr/bin/env python3
"""Replay SDK bank events and check vault routing across height-bound exports."""
import argparse
from collections import defaultdict
import hashlib
import json
from pathlib import Path
import re

from upgrade_inventory import retained_vesting_addresses


def amount(value):
    if not isinstance(value, str) or not re.fullmatch(r"[0-9]{1,78}", value):
        raise ValueError(f"invalid integer coin amount: {value!r}")
    number = int(value)
    if number >= 2**256:
        raise ValueError("coin amount exceeds SDK integer range")
    return number


def coins(value):
    if not isinstance(value, str):
        raise ValueError("event coin amount must be a string")
    result = {}
    if not value:
        return result  # SDK can emit an empty coin set for a no-op bank call.
    for coin in value.split(","):
        match = re.fullmatch(r"([0-9]+)([a-zA-Z][a-zA-Z0-9/:._-]{2,127})", coin)
        if not match or match[2] in result:
            raise ValueError(f"invalid or duplicate event coin: {coin!r}")
        result[match[2]] = amount(match[1])
    return result


def balances(document):
    rows = document["app_state"]["bank"]["balances"]
    if not isinstance(rows, list):
        raise ValueError("export must include bank balance rows")
    result, addresses = {}, set()
    for row in rows:
        address = row["address"]
        if not isinstance(address, str) or not address or address in addresses:
            raise ValueError("invalid or duplicate bank address")
        addresses.add(address)
        for coin in row["coins"]:
            key = (address, coin["denom"])
            if key in result:
                raise ValueError("duplicate bank denomination")
            result[key] = amount(coin["amount"])
    return result


def attributes(event):
    result = {}
    for item in event["attributes"]:
        if item["key"] in result:
            raise ValueError("duplicate bank event attribute")
        result[item["key"]] = item["value"]
    return result


def verify_bank(before, after, blocks, vaults):
    start, end = int(before["initial_height"]), int(after["initial_height"])
    if start <= 0 or end <= start or [int(block["height"]) for block in blocks] != list(range(start, end)):
        raise ValueError("block events must cover every exported boundary height exactly once")
    original, final = balances(before), balances(after)
    replay = defaultdict(int, original)
    sources = {}
    for row in before["app_state"]["structs"]["addressList"]:
        if int(row["playerIndex"]) > 0:
            if row["address"] in sources:
                raise ValueError("duplicate source address association")
            sources[row["address"]] = f"1-{int(row['playerIndex'])}"
    if not isinstance(vaults, dict) or any(not isinstance(value, str) or not value for value in vaults.values()):
        raise ValueError("invalid vault map")
    if len(set(vaults.values())) != len(vaults) or set(sources) & set(vaults.values()):
        raise ValueError("vaults must be distinct non-signing addresses")
    if set(sources.values()) - vaults.keys():
        raise ValueError("missing vault mapping")
    retained = retained_vesting_addresses(before)
    sources = {source: owner for source, owner in sources.items() if source not in retained}
    moved = defaultdict(int)
    routing = []
    bank_events = 0
    for block in blocks:
        # FinalizeBlock combines pre/begin/end events; transaction events are
        # separate. Sum each block before checking balances, since serialized
        # stream order is not a total execution order across those envelopes.
        events = list(block["finalize_block_events"] or [])
        for transaction in block.get("txs_results") or []:
            # Failed transactions can still commit ante fee movements. Do not
            # discard their public events based on the transaction result code.
            events.extend(transaction.get("events") or [])
        for event in events:
            kind = event["type"]
            if kind not in ("coin_spent", "coin_received", "transfer"):
                continue
            data = attributes(event)
            event_coins = coins(data["amount"])
            if kind == "transfer":
                source, destination = data["sender"], data["recipient"]
                if source in retained and any(event_coins.values()):
                    routing.append({"source": source, "retained_vesting_account_debited": True, "actual_recipient": destination})
                if source in sources:
                    expected = vaults[sources[source]]
                    if destination != expected and any(event_coins.values()):
                        routing.append({"source": source, "expected_vault": expected, "actual_recipient": destination})
                    if destination == expected:
                        for denom, value in event_coins.items():
                            moved[source, denom] += value
                continue  # Never apply the transfer a second time to balances.
            address = data["spender" if kind == "coin_spent" else "receiver"]
            if not isinstance(address, str) or not address:
                raise ValueError("missing bank event address")
            sign = -1 if kind == "coin_spent" else 1
            for denom, value in event_coins.items():
                replay[address, denom] += sign * value
            bank_events += 1
        if any(value < 0 for value in replay.values()):
            raise ValueError(f"bank event replay overdraws an account at height {block['height']}")
    differences = []
    for key in sorted(replay.keys() | final.keys()):
        if replay.get(key, 0) != final.get(key, 0):
            differences.append({"address": key[0], "denom": key[1],
                "event_balance": str(replay.get(key, 0)), "export_balance": str(final.get(key, 0))})
    for (source, denom), value in original.items():
        if source in sources and moved[source, denom] < value:
            routing.append({"source": source, "denom": denom, "required": str(value), "transferred_to_vault": str(moved[source, denom])})
    for (source, denom), value in final.items():
        if source in sources and value:
            routing.append({"source": source, "denom": denom, "remaining_signing_address_balance": str(value)})
    return {"balance_differences": differences, "migration_differences": routing, "bank_event_count": bank_events,
        "retained_vesting_addresses": sorted(retained),
        "scope": "All bank balances and associated-source vault routing only; excludes vesting locks, staking and grid state. Ordinary source spending in this window requires separate attribution."}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ("before", "after", "blocks", "vault_map"):
        parser.add_argument(name, type=Path)
    args = parser.parse_args()
    raw = {name: getattr(args, name).read_bytes() for name in ("before", "after", "blocks", "vault_map")}
    try:
        result = verify_bank(*(json.loads(raw[name]) for name in ("before", "after", "blocks", "vault_map")))
    except (ValueError, KeyError, TypeError) as error:
        parser.error(str(error))
    result["input_sha256"] = {name: hashlib.sha256(value).hexdigest() for name, value in raw.items()}
    print(json.dumps(result, indent=2))
    return 1 if result["balance_differences"] or result["migration_differences"] else 0


if __name__ == "__main__":
    raise SystemExit(main())
