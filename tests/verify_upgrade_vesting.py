"""Verify native vesting retention against boundary auth exports.

This rejects relocation events and checks exact account and schedule
preservation. SDK spendability requires separate real-app upgrade tests.
"""
import base64
import json

from upgrade_inventory import base_account, retained_vesting_addresses


BASE = "/cosmos.auth.v1beta1.BaseAccount"
VAULT = "/structs.structs.VaultVestingAccount"
VESTING = {"/cosmos.vesting.v1beta1." + name for name in (
    "ContinuousVestingAccount", "DelayedVestingAccount", "PeriodicVestingAccount", "PermanentLockedAccount")}


def source_address_bytes(address):
    """Decode canonical Structs Bech32 without using the migration encoder."""
    alphabet = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"
    if not isinstance(address, str) or not address.startswith("structs1") or address != address.lower():
        raise ValueError("invalid vesting source address")
    try:
        values = [alphabet.index(char) for char in address[8:]]
    except ValueError as error:
        raise ValueError("invalid vesting source address alphabet") from error
    if len(values) < 7:
        raise ValueError("invalid vesting source address length")
    hrp = "structs"
    checksum = 1
    for value in [ord(char) >> 5 for char in hrp] + [0] + [ord(char) & 31 for char in hrp] + values:
        top = checksum >> 25
        checksum = ((checksum & 0x1ffffff) << 5) ^ value
        for bit, generator in enumerate((0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3)):
            if (top >> bit) & 1:
                checksum ^= generator
    if checksum != 1:
        raise ValueError("invalid vesting source address checksum")
    result, accumulator, bits = bytearray(), 0, 0
    for value in values[:-6]:
        accumulator = (accumulator << 5) | value
        bits += 5
        if bits >= 8:
            bits -= 8
            result.append((accumulator >> bits) & 255)
        accumulator &= (1 << bits) - 1
    if bits >= 5 or accumulator or not 0 < len(result) <= 255:
        raise ValueError("invalid vesting source address padding or payload")
    return bytes(result)


def original_schedule(account):
    """Build the expected persisted schedule from the OLD account only.

    Keep original per-source counters, including unbondings and historical
    slashes; deriving these from current stake would conceal lost obligations.
    """
    names = ("ContinuousVestingAccount", "DelayedVestingAccount", "PeriodicVestingAccount", "PermanentLockedAccount")
    kind = names.index(account["@type"].removeprefix("/cosmos.vesting.v1beta1.")) + 1
    base = account["base_vesting_account"]
    return {
        "source_address": base64.b64encode(source_address_bytes(base["base_account"]["address"])).decode("ascii"),
        "kind": kind,
        "start_time": account.get("start_time", "0") if kind in (1, 3) else "0",
        "end_time": base.get("end_time", "0") if kind != 4 else "0",
        "original_vesting": base.get("original_vesting") or [],
        "delegated_free": base.get("delegated_free") or [],
        "delegated_vesting": base.get("delegated_vesting") or [],
        "periods": (account.get("vesting_periods") or []) if kind == 3 else [],
    }


def accounts(document):
    result = {}
    for account in document["app_state"]["auth"]["accounts"]:
        address = base_account(account)["address"]
        if address in result:
            raise ValueError("duplicate auth account")
        result[address] = account
    return result


def verify_vesting(before, after, blocks, vaults):
    start, end = int(before["initial_height"]), int(after["initial_height"])
    if start <= 0 or end <= start or [int(block["height"]) for block in blocks] != list(range(start, end)):
        raise ValueError("block events must cover every exported boundary height exactly once")
    old, new = accounts(before), accounts(after)
    owners = {}
    for row in before["app_state"]["structs"]["addressList"]:
        if int(row["playerIndex"]) > 0:
            if row["address"] in owners:
                raise ValueError("duplicate source address association")
            owners[row["address"]] = f"1-{int(row['playerIndex'])}"
    retained = retained_vesting_addresses(before)
    differences = []
    for block in blocks:
        if block.get("txs_results"):
            raise ValueError("vesting replay requires a transaction-free migration window")
        for event in block["finalize_block_events"] or []:
            if event["type"] == "structs.structs.EventVaultVestingMigration":
                raise ValueError("unexpected vesting migration event: native accounts must remain intact")
    for address, account in old.items():
        if account["@type"] in VESTING or account["@type"] == VAULT:
            if json.dumps(new.get(address), sort_keys=True) != json.dumps(account, sort_keys=True):
                differences.append(f"retained vesting account changed: {address}")
    for address, account in new.items():
        if account["@type"] == VAULT and address not in old:
            differences.append(f"unexpected composite vesting vault: {address}")
    return {"differences": differences, "migrated_source_count": 0,
        "retained_vesting_addresses": sorted(retained), "vault_event_count": 0,
        "scope": "Native vesting accounts and any pre-existing composite accounts remain byte-equivalent in JSON; no schedule relocation events. Excludes live spendability and queue maturity."}
