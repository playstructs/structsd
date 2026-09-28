#!/usr/bin/env python3
"""Compare staking exports across the liquid-only vault upgrade, before further staking activity.

The vault map is an owner-ID/address JSON object captured from vault queries or
association events. This checks retained staking accounting, not complete upgrade
approval. Run at a migration boundary: maturity, slashing, or user transactions
between snapshots intentionally produce differences requiring investigation.
"""
import argparse
from collections import Counter
import hashlib
import json
from pathlib import Path
import re

from upgrade_inventory import base_account, inventory, retained_vesting_addresses


def shares(value):
    if not isinstance(value, str) or not re.fullmatch(r"[0-9]+(?:\.[0-9]{1,18})?", value):
        raise ValueError(f"invalid SDK share amount: {value!r}")
    whole, _, fraction = value.partition(".")
    return int(whole) * 10**18 + int(fraction.ljust(18, "0"))


def staking_projection(state):
    stake = state["staking"]
    result = {}
    for field, validator_fields in {
        "delegations": ("validator_address",),
        "unbonding_delegations": ("validator_address",),
        "redelegations": ("validator_src_address", "validator_dst_address"),
    }.items():
        rows, seen = Counter(), set()
        for row in stake[field]:
            source = row["delegator_address"]
            validators = tuple(row[key] for key in validator_fields)
            original = (source, *validators)
            if original in seen:
                raise ValueError(f"duplicate {field} record: {original}")
            seen.add(original)
            target = (source, *validators)
            if field == "delegations":
                amount = shares(row["shares"])
                if amount <= 0:
                    raise ValueError("delegation must have positive shares")
                rows[target] += amount
            else:
                if not isinstance(row.get("entries"), list) or not row["entries"]:
                    raise ValueError(f"missing pending entries: {original}")
                for entry in row["entries"]:
                    # Preserve the complete entry, including SDK operation IDs,
                    # initial/current balances, completion times and hold counts.
                    rows[(*target, json.dumps(entry, sort_keys=True, separators=(",", ":")))] += 1
        result[field] = rows
    return result


def verify(before, after, vaults):
    for document in (before, after):
        report = inventory(document)
        if report["problems"]:
            raise ValueError("; ".join(report["problems"]))
    before, after = before.get("app_state", before), after.get("app_state", after)
    if not isinstance(vaults, dict) or any(not isinstance(v, str) or not v for v in vaults.values()):
        raise ValueError("vault map must contain owner-ID/address strings")
    if len(set(vaults.values())) != len(vaults):
        raise ValueError("vault addresses must be unique")
    owners = {row["address"]: f"1-{int(row['playerIndex'])}" for row in before["structs"]["addressList"] if int(row["playerIndex"]) > 0}
    missing = set(owners.values()) - vaults.keys()
    if missing:
        raise ValueError(f"missing vault mappings: {sorted(missing)}")
    exported_owners = after["structs"].get("vaultOwnerList")
    if not isinstance(exported_owners, list) or not set(vaults).issubset(exported_owners):
        raise ValueError("vault map owners are absent from post-upgrade export")
    accounts = {base_account(row)["address"]: row for row in after["auth"]["accounts"]}
    for owner, address in vaults.items():
        if address in owners or address not in accounts:
            raise ValueError(f"vault is a source signing address or lacks auth account: {owner}")
    retained = retained_vesting_addresses(before)
    destinations = {address: vaults[owner] for address, owner in owners.items() if address not in retained}
    expected, actual = staking_projection(before), staking_projection(after)
    differences = []
    for field in expected:
        for key in sorted(expected[field].keys() | actual[field].keys()):
            wanted, found = expected[field][key], actual[field][key]
            if wanted != found:
                differences.append({"field": field, "key": list(key), "expected": str(wanted), "actual": str(found)})
    return {"differences": differences, "source_addresses": len(destinations),
        "retained_vesting_addresses": sorted(retained),
        "scope": "Exact delegation shares and complete pending entries, at original addresses, including self-stake and vesting accounts; not queues, rewards, balances, vault derivation, or full upgrade approval."}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("before", type=Path)
    parser.add_argument("after", type=Path)
    parser.add_argument("vault_map", type=Path)
    args = parser.parse_args()
    raw = {name: getattr(args, name).read_bytes() for name in ("before", "after", "vault_map")}
    try:
        result = verify(*(json.loads(raw[name]) for name in ("before", "after", "vault_map")))
    except (ValueError, KeyError, TypeError) as error:
        parser.error(str(error))
    result["input_sha256"] = {name: hashlib.sha256(data).hexdigest() for name, data in raw.items()}
    print(json.dumps(result, indent=2, sort_keys=True))
    return 1 if result["differences"] else 0


if __name__ == "__main__":
    raise SystemExit(main())
