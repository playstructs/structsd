#!/usr/bin/env python3
"""Read-only v0.21 export inventory; does not validate or migrate genesis."""
import argparse
from collections import Counter, defaultdict
import hashlib
import json
from pathlib import Path


VESTING = {
    "/cosmos.vesting.v1beta1.ContinuousVestingAccount",
    "/cosmos.vesting.v1beta1.DelayedVestingAccount",
    "/cosmos.vesting.v1beta1.PeriodicVestingAccount",
    "/cosmos.vesting.v1beta1.PermanentLockedAccount",
}


def base_account(account):
    if "address" in account:
        return account
    for field in ("base_account", "base_vesting_account"):
        if field in account:
            return base_account(account[field])
    raise ValueError(f"cannot resolve auth account address: {account.get('@type')}")


def payload(address):
    """Compare account/operator payloads in an already validated SDK export.

    Prefix and its six-character checksum differ for account and validator
    addresses. This deliberately is not a Bech32/genesis validator.
    """
    prefix, separator, data = address.rpartition("1")
    if not prefix or not separator or len(data) <= 6:
        raise ValueError(f"malformed address: {address}")
    return data[:-6]


def retained_vesting_addresses(document):
    """All associated native vesting accounts remain whole at upgrade."""
    state = document.get("app_state", document)
    associated = {row["address"] for row in state["structs"]["addressList"] if int(row["playerIndex"]) > 0}
    return {base_account(row)["address"] for row in state.get("auth", {}).get("accounts", [])
            if row.get("@type") in VESTING} & associated


def inventory(document):
    state = document.get("app_state", document)
    for module in ("structs", "auth", "bank", "staking"):
        if not isinstance(state.get(module), dict):
            raise ValueError(f"missing module state: {module}")
    # SDK exports include these lists even when empty. Missing/query-shaped
    # input must not look like a valid export with no migration obligations.
    for module, fields in {
        "structs": ("playerList", "addressList"),
        "auth": ("accounts",),
        "bank": ("balances",),
        "staking": ("delegations", "unbonding_delegations", "redelegations"),
    }.items():
        for field in fields:
            if not isinstance(state[module].get(field), list):
                raise ValueError(f"missing or invalid export list: {module}.{field}")
    structs = state["structs"]
    players = {row["id"] for row in structs.get("playerList", [])}
    owners = {}
    for row in structs.get("addressList", []):
        index = int(row["playerIndex"])
        if index == 0:
            continue
        owner = f"1-{index}"
        if owner not in players:
            raise ValueError(f"address references missing player: {owner}")
        address = row["address"]
        if address in owners:
            raise ValueError(f"duplicate indexed address: {address}")
        owners[address] = owner
    accounts = {}
    for row in state["auth"].get("accounts", []):
        address = base_account(row)["address"]
        if address in accounts:
            raise ValueError(f"duplicate auth account: {address}")
        accounts[address] = row
    totals = defaultdict(Counter)
    untouched = Counter()
    for row in state["bank"].get("balances", []):
        address = row["address"]
        target = totals[owners[address]] if address in owners else untouched
        for coin in row["coins"]:
            amount = int(coin["amount"])
            if amount < 0:
                raise ValueError("negative balance")
            target[coin["denom"]] += amount
    records = {}
    problems = []
    for address, owner in sorted(owners.items()):
        record = records.setdefault(owner, {"addresses": [], "vesting_accounts": 0,
            "vesting_periods": 0, "vesting_denominations": [], "account_types": {}})
        record["addresses"].append(address)
        account = accounts.get(address)
        if account is None:
            problems.append(f"indexed address lacks auth account: {address}")
            continue
        kind = account.get("@type", "")
        record["account_types"][kind] = record["account_types"].get(kind, 0) + 1
        if kind in VESTING:
            record["vesting_accounts"] += 1
            record["vesting_periods"] += len(account.get("vesting_periods", []))
            denominations = set(record["vesting_denominations"])
            denominations.update(c["denom"] for c in account["base_vesting_account"]["original_vesting"])
            record["vesting_denominations"] = sorted(denominations)
        elif kind != "/cosmos.auth.v1beta1.BaseAccount":
            problems.append(f"unsupported associated account type {kind}: {address}")
    self_delegations = []
    staking = state["staking"]
    for delegation in staking.get("delegations", []):
        address = delegation["delegator_address"]
        if address in owners and payload(address) == payload(delegation["validator_address"]):
            self_delegations.append({"owner_id": owners[address], **delegation})
    for field in ("delegations", "unbonding_delegations", "redelegations"):
        counts, entries = Counter(), Counter()
        for row in staking.get(field, []):
            owner = owners.get(row["delegator_address"])
            if owner is not None:
                counts[owner] += 1
                if field != "delegations":
                    entries[owner] += len(row["entries"])
        for owner, record in records.items():
            record[field] = counts[owner]
            if field != "delegations":
                record[field + "_entries"] = entries[owner]
    for owner, record in records.items():
        record["bank_balances"] = {d: str(n) for d, n in sorted(totals[owner].items())}
    return {"owners": records, "associated_addresses": len(owners),
        "max_vesting_accounts_per_owner": max((r["vesting_accounts"] for r in records.values()), default=0),
        "associated_self_delegations": sorted(self_delegations, key=lambda r: r["delegator_address"]),
        "unassociated_bank_balances": {d: str(n) for d, n in sorted(untouched.items())},
        "problems": problems,
        "scope": "Inventory only; bank totals exclude unsettled rewards and staking tokens. Not migration approval."}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("export", type=Path)
    args = parser.parse_args()
    raw = args.export.read_bytes()
    try:
        result = inventory(json.loads(raw))
    except (ValueError, KeyError, TypeError) as error:
        parser.error(str(error))
    result["source_sha256"] = hashlib.sha256(raw).hexdigest()
    print(json.dumps(result, indent=2, sort_keys=True))
    return 1 if result["problems"] else 0


if __name__ == "__main__":
    raise SystemExit(main())
