"""Check transaction-free native genesis restart against its exported input.

Both preserved and zero initial heights are accepted. Zero-height preparation
already happened in the input export; this compares import/continuation, not
the correctness of that preparation or all gameplay/staking state.
"""
import argparse
from collections import defaultdict
import hashlib
import json
from pathlib import Path

from upgrade_inventory import base_account
from verify_upgrade_bank import amount, attributes, balances, coins


def supply(document):
    result = {}
    for coin in document["app_state"]["bank"]["supply"]:
        denom = coin["denom"]
        if denom in result:
            raise ValueError("duplicate supply denomination")
        result[denom] = amount(coin["amount"])
    return result


def identities(document):
    result = {}
    for account in document["app_state"]["auth"]["accounts"]:
        base = base_account(account)
        address = base["address"]
        if address in result:
            raise ValueError("duplicate auth identity")
        result[address] = {"type": account["@type"], "base": base}
    return result


def players(document):
    result = {}
    for player in document["app_state"]["structs"]["playerList"]:
        if player["id"] in result:
            raise ValueError("duplicate player")
        result[player["id"]] = {name: player[name] for name in (
            "index", "primaryAddress", "guildId", "guildRank")}
    return result


def verify_restart(before, after, blocks):
    start = max(1, int(before["initial_height"]))
    end = int(after["initial_height"])
    if int(before["initial_height"]) < 0 or end <= start or [int(b["height"]) for b in blocks] != list(range(start, end)):
        raise ValueError("restart events must cover every exported boundary height exactly once")
    if before["chain_id"] != after["chain_id"]:
        raise ValueError("restart exports refer to different chain IDs")
    original, final = balances(before), balances(after)
    replay = defaultdict(int, original)
    expected_supply = defaultdict(int, supply(before))
    count = 0
    for block in blocks:
        if block.get("txs_results"):
            raise ValueError("restart comparison requires a transaction-free window")
        if "finalize_block_events" not in block:
            raise ValueError("missing block event envelope")
        for event in block["finalize_block_events"] or []:
            kind = event["type"]
            if kind not in ("coin_spent", "coin_received", "coinbase", "burn"):
                continue  # Transfer summaries overlap the debit/credit stream.
            data = attributes(event)
            values = coins(data["amount"])
            if kind in ("coinbase", "burn"):
                for denom, value in values.items():
                    expected_supply[denom] += value if kind == "coinbase" else -value
            else:
                address = data["spender" if kind == "coin_spent" else "receiver"]
                if not isinstance(address, str) or not address:
                    raise ValueError("missing bank event address")
                for denom, value in values.items():
                    replay[address, denom] += value if kind == "coin_received" else -value
            count += 1
        if any(value < 0 for value in replay.values()) or any(value < 0 for value in expected_supply.values()):
            raise ValueError("restart event replay overdraws a balance or supply")
    differences = []
    for key in sorted(replay.keys() | final.keys()):
        if replay.get(key, 0) != final.get(key, 0):
            differences.append(f"bank balance differs from events: {key}")
    actual_supply = supply(after)
    for denom in sorted(expected_supply.keys() | actual_supply.keys()):
        if expected_supply.get(denom, 0) != actual_supply.get(denom, 0):
            differences.append(f"supply differs from mint/burn events: {denom}")
    for document in (before, after):
        totals = defaultdict(int)
        for (_, denom), value in balances(document).items():
            totals[denom] += value
        exported = supply(document)
        if any(totals.get(denom, 0) != exported.get(denom, 0) for denom in totals.keys() | exported.keys()):
            differences.append("exported supply disagrees with sum of bank balances")
    if identities(before) != identities(after):
        differences.append("auth account type, identity, public key, account number or sequence changed")
    if players(before) != players(after):
        differences.append("player identity, signing address, membership or rank changed")
    old, new = before["app_state"]["structs"], after["app_state"]["structs"]
    # These records have no autonomous time-based transition. Lists exported in
    # deterministic store order must survive an idle genesis restart exactly.
    for name in ("vaultOwnerList", "guildList", "addressList", "addressNonceList"):
        if old[name] != new[name]:
            differences.append(f"{name} changed across restart")
    old_cap, new_cap = old.get("reactorStakeCapActive", False), new.get("reactorStakeCapActive", False)
    if type(old_cap) is not bool or type(new_cap) is not bool:
        raise ValueError("invalid stake cap activation flag")
    if old_cap != new_cap:
        differences.append("stake cap activation changed across restart")
    if old.get("params") != new.get("params"):
        differences.append("module parameters changed across restart")
    if amount(new["stakingUnbondingId"]) < amount(old["stakingUnbondingId"]):
        differences.append("staking operation ID counter regressed")
    return {"differences": differences, "blocks": len(blocks), "bank_events": count,
            "accounts": len(identities(after)), "vault_owners": len(new["vaultOwnerList"]),
            "scope": "All-denomination bank/supply event reconciliation, auth identities, player membership/ranks, guild settings/owners, vault-owner list, signing-address associations/nonces, stake cap activation, parameters and staking ID counter; not all gameplay state or staking/vesting lifecycle correctness."}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ("before", "after", "blocks"):
        parser.add_argument(name, type=Path)
    args = parser.parse_args()
    raw = {name: getattr(args, name).read_bytes() for name in ("before", "after", "blocks")}
    result = verify_restart(*(json.loads(raw[name]) for name in ("before", "after", "blocks")))
    result["input_sha256"] = {name: hashlib.sha256(value).hexdigest() for name, value in raw.items()}
    print(json.dumps(result, indent=2))
    return int(bool(result["differences"]))


if __name__ == "__main__":
    raise SystemExit(main())
