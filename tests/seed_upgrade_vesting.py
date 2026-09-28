"""Populate vesting migration cases through v0.21 transactions, after gameplay."""
from datetime import datetime
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess

from staking_unbonding_entries import creation_heights
from struct_inventory import next_cursor
from verify_upgrade_bank import attributes, coins


def proof_tool_input(path):
    path = Path(path).resolve(strict=True)
    if not path.is_file() or not os.access(path, os.X_OK):
        raise ValueError("vesting proof tool must be executable")
    return {"path": str(path), "sha256": hashlib.sha256(path.read_bytes()).hexdigest()}


def schedule_duration(value):
    if type(value) is not int or not 1200 <= value <= 86400 or value % 4:
        raise ValueError("vesting fixture schedule must be 1200..86400 seconds and divisible by four")
    return value


def account_value(document, expected_type):
    record = document.get("account") if isinstance(document, dict) else None
    if not isinstance(record, dict):
        raise ValueError("missing vesting account query")
    if "@type" in record:
        kind, value = record["@type"], record
        if "type" in record or "value" in record:
            raise ValueError("ambiguous vesting account encoding")
    else:
        kind, value = record.get("type"), record.get("value")
    if kind != expected_type or not isinstance(value, dict):
        raise ValueError("unexpected vesting account type")
    return value


def legacy_account(document, address, expected_type):
    value = account_value(document, "/cosmos.vesting.v1beta1." + expected_type)
    base = value.get("base_vesting_account")
    if not isinstance(base, dict) or not isinstance(base.get("base_account"), dict) or base["base_account"].get("address") != address:
        raise ValueError("vesting account address did not resolve")
    original = base.get("original_vesting")
    if not isinstance(original, list) or len(original) != 1 or original[0].get("denom") != "ualpha" or original[0].get("amount") != "2000000":
        raise ValueError("vesting account has unexpected original coins")
    return base


def alpha_coins(coins, label):
    if coins is None:
        coins = []
    if not isinstance(coins, list) or len(coins) > 1:
        raise ValueError(f"invalid {label} coin inventory")
    if not coins:
        return 0
    coin = coins[0]
    if not isinstance(coin, dict) or coin.get("denom") != "ualpha":
        raise ValueError(f"unexpected {label} denomination")
    amount = coin.get("amount")
    if not isinstance(amount, str) or len(amount) > 78 or not re.fullmatch(r"0|[1-9][0-9]*", amount):
        raise ValueError(f"invalid {label} amount")
    return int(amount)


def complete_rows(document, field):
    if next_cursor(document):
        raise ValueError(f"incomplete {field} fixture query")
    rows = document.get(field)
    if rows is None:
        rows = []
    if not isinstance(rows, list):
        raise ValueError(f"invalid {field} fixture query")
    return rows


def source_bank_balance(transactions, address):
    balance, matched = 0, False
    for delivered in transactions:
        for event in delivered["transaction"]["events"]:
            if event["type"] not in ("coin_spent", "coin_received"):
                continue
            data = attributes(event)
            subject = data["spender" if event["type"] == "coin_spent" else "receiver"]
            if subject != address:
                continue
            amounts = coins(data["amount"])
            if set(amounts) - {"ualpha"}:
                raise ValueError("vesting source received an unexpected denomination")
            balance += amounts.get("ualpha", 0) * (-1 if event["type"] == "coin_spent" else 1)
            matched = True
    if not matched or balance <= 0:
        raise ValueError("vesting source bank event history is missing or exhausted")
    return balance


def verify_seed_snapshot(row, snapshot, player, validator, end_time, expected_balance):
    address = row["proof"]["address"]
    ordinary = row["account_type"] == "BaseAccount"
    if ordinary:
        value = account_value(snapshot["account"], "/cosmos.auth.v1beta1.BaseAccount")
        if value.get("address") != address or "base_vesting_account" in value:
            raise ValueError("ordinary fixture account did not resolve")
    else:
        base = legacy_account(snapshot["account"], address, row["account_type"])
        actual_end = base.get("end_time", "0" if row["account_type"] == "PermanentLockedAccount" else None)
        if actual_end != row.get("end_time", end_time):
            raise ValueError("vesting fixture end time differs from its transaction")
        if row["account_type"] == "PeriodicVestingAccount":
            value = account_value(snapshot["account"], "/cosmos.vesting.v1beta1.PeriodicVestingAccount")
            if value.get("start_time") != row["start_time"] or value.get("vesting_periods") != row["vesting_periods"]:
                raise ValueError("periodic fixture installments differ from its transaction")
    expected_owner = player if row["associated"] else ""
    association = snapshot["association"]
    if association.get("address") != address or association.get("playerId", "") != expected_owner:
        raise ValueError("vesting address has the wrong player association")
    expected_delegated = int(row["delegated_amount"])
    expected_pending = int(row["unbonding_amount"])
    if not ordinary:
        vested = alpha_coins(base.get("delegated_vesting"), "delegated vesting")
        free = alpha_coins(base.get("delegated_free"), "delegated free")
        if vested + free != expected_delegated or (row["associated"] and vested == 0):
            raise ValueError("vesting fixture delegation counters differ from funded principal")
    bank_balance = alpha_coins(complete_rows(snapshot["balances"], "balances"), "source balance")
    # Native undelegation can withdraw staking rewards into the source; the
    # continuous registration sweep can also remove newly vested coins. Replay
    # exact SDK bank movements instead of assuming original minus delegation.
    if bank_balance != expected_balance:
        raise ValueError("vesting source balance differs from its bank event history")
    if not row["associated"] and bank_balance != 2000000:
        raise ValueError("unassociated control principal changed")
    delegations = complete_rows(snapshot["delegations"], "delegation_responses")
    if expected_delegated == 0:
        if delegations:
            raise ValueError("unassociated control unexpectedly has a delegation")
    else:
        if len(delegations) != 1 or not isinstance(delegations[0], dict):
            raise ValueError("vesting fixture must retain exactly one active delegation")
        delegation = delegations[0].get("delegation")
        if not isinstance(delegation, dict) or delegation.get("delegator_address") != address or delegation.get("validator_address") != validator:
            raise ValueError("vesting fixture delegation has the wrong identity")
        balance = alpha_coins([delegations[0].get("balance")], "active delegation")
        if balance != expected_delegated - expected_pending:
            raise ValueError("vesting fixture active principal differs")
    pending_heights = creation_heights(snapshot["unbondings"], address, validator)
    pending = complete_rows(snapshot["unbondings"], "unbonding_responses")
    if expected_pending == 0:
        if pending:
            raise ValueError("vesting fixture unexpectedly has unbonding state")
    else:
        if len(pending) != 1 or pending[0].get("validator_address") != validator or pending_heights != [int(row["unbonding_height"])]:
            raise ValueError("vesting fixture lost its exact pending unbonding")
        entry = pending[0]["entries"][0]
        for field in ("balance", "initial_balance"):
            if alpha_coins([{"denom": "ualpha", "amount": entry.get(field)}], "unbonding " + field) != expected_pending:
                raise ValueError("vesting fixture pending principal differs")


def seed_vesting(runner, tool, make_proof=None, schedule_seconds=86400):
    """No restart or raw state editing: every seeded fact is queried on-chain."""
    schedule_seconds = schedule_duration(schedule_seconds)
    if proof_tool_input(tool["path"]) != tool:
        raise ValueError("vesting proof tool changed after preparation")
    if (runner.directory / "vesting-seed.json").exists():
        raise ValueError("vesting fixture already attempted; preserve its evidence")
    report = {"status": "seeding started; not verified", "proof_tool": tool, "transactions": [], "sources": []}
    runner.save("vesting-seed.json", report)
    try:
        key = runner.cli(runner.old, "keys", "show", "alice", "--keyring-backend", "test")
        association = runner.cli(runner.old, "query", "structs", "address", key["address"])
        player = association.get("playerId")
        if not isinstance(player, str) or not re.fullmatch(r"1-[1-9][0-9]*", player):
            raise ValueError("fixture signer has no player")
        operator = runner.cli(runner.old, "keys", "show", "alice", "--bech", "val", "--keyring-backend", "test")
        validator = operator["address"]
        validator_record = runner.cli(runner.old, "query", "staking", "validator", validator)["validator"]
        if validator_record.get("operator_address") != validator:
            raise ValueError("fixture operator validator did not resolve")
        status = runner.cli(runner.old, "status")
        end_time = str(int(datetime.fromisoformat(status["sync_info"]["latest_block_time"].replace("Z", "+00:00")).timestamp()) + schedule_seconds)
        report.update(player_id=player, validator=validator, end_time=end_time, schedule_seconds=schedule_seconds)
        def send(label, *arguments, signer="alice"):
            delivered = runner.tx(*arguments, signer=signer)
            report["transactions"].append({"step": label, "transaction": delivered})
            runner.save("vesting-seed.json", report)
            return delivered
        for name, account_type, associated, delegated in (
                ("delayed", "DelayedVestingAccount", True, 1000000),
                ("continuous", "ContinuousVestingAccount", True, 800000),
                ("unassociated", "DelayedVestingAccount", False, 0),
                ("periodic", "PeriodicVestingAccount", True, 600000),
                ("permanent", "PermanentLockedAccount", True, 500000),
                ("ordinary", "BaseAccount", True, 800000)):
            key_name = "upgrade_vesting_" + name
            if make_proof is None:
                result = subprocess.run([tool["path"], "--home", str(runner.home), "--name", key_name, "--player", player],
                    capture_output=True, text=True, check=True, timeout=30)
                proof = json.loads(result.stdout)
            else:
                proof = make_proof(key_name, player)
            if set(proof) != {"key_name", "player_id", "address", "public_key", "signature"} or proof["key_name"] != key_name or proof["player_id"] != player:
                raise ValueError("unexpected legacy registration proof")
            if not re.fullmatch(r"structs1[0-9a-z]+", proof["address"]) or not re.fullmatch(r"[0-9a-f]{66}", proof["public_key"]) or not re.fullmatch(r"[0-9a-f]{128}", proof["signature"]):
                raise ValueError("malformed public legacy registration proof")
            ordinary = name == "ordinary"
            schedule_input = {}
            if ordinary:
                # Register an empty source first. Legacy registration sweeps
                # spendable funds, so funding it first would erase this fixture.
                send(name + "_register", "structs", "address-register", player, proof["address"], proof["public_key"], proof["signature"], "127")
                send(name + "_fund", "bank", "send", key["address"], proof["address"], "2000000ualpha")
            elif name == "periodic":
                start_time = int(end_time) - schedule_seconds
                periods = ((schedule_seconds // 4, 750000), (schedule_seconds * 3 // 4, 1250000))
                installments = [{"length": str(length), "amount": [{"denom": "ualpha", "amount": str(coins)}]} for length, coins in periods]
                path = runner.directory / "vesting-periodic-input.json"
                with path.open("x") as output:
                    json.dump({"start_time": start_time, "periods": [
                        {"coins": str(coins) + "ualpha", "length_seconds": length} for length, coins in periods]}, output)
                schedule_input = {"start_time": str(start_time), "vesting_periods": installments,
                    "period_file_sha256": hashlib.sha256(path.read_bytes()).hexdigest()}
                send(name + "_create", "vesting", "create-periodic-vesting-account", proof["address"], str(path))
            elif name == "permanent":
                schedule_input = {"end_time": "0"}
                send(name + "_create", "vesting", "create-permanent-locked-account", proof["address"], "2000000ualpha")
            else:
                args = ["vesting", "create-vesting-account", proof["address"], "2000000ualpha", end_time]
                if account_type == "DelayedVestingAccount":
                    args.append("--delayed")
                send(name + "_create", *args)
            row = {"name": name, "proof": proof, "associated": associated,
                "account_type": account_type,
                "delegated_amount": str(delegated), "unbonding_amount": "100000" if name in ("delayed", "ordinary") else "0", **schedule_input}
            report["sources"].append(row)
            if associated:
                if not ordinary:
                    send(name + "_register", "structs", "address-register", player, proof["address"], proof["public_key"], proof["signature"], "127")
                send(name + "_delegate", "staking", "delegate", validator, str(delegated) + "ualpha", signer=key_name)
            if name in ("delayed", "ordinary"):
                transaction = send(name + "_undelegate", "staking", "unbond", validator, "100000ualpha", signer=key_name)
                row["unbonding_height"] = transaction["height"]
        height = str(runner.height(runner.old))
        report["query_height"] = height
        for row in report["sources"]:
            address = row["proof"]["address"]
            snapshot = {name: runner.cli(runner.old, "query", *arguments, "--height", height) for name, arguments in (
                ("account", ("auth", "account", address)),
                ("association", ("structs", "address", address)),
                ("balances", ("bank", "balances", address, "--page-limit", "100")),
                ("delegations", ("staking", "delegations", address, "--page-limit", "100")),
                ("unbondings", ("staking", "unbonding-delegations", address, "--page-limit", "100")))}
            expected_balance = source_bank_balance(report["transactions"], address)
            verify_seed_snapshot(row, snapshot, player, validator, end_time, expected_balance)
            row["bank_event_balance"] = str(expected_balance)
            row["snapshot"] = snapshot
        report["status"] = "native vesting seed verified; upgrade migration pending"
        runner.save("vesting-seed.json", report)
        return report
    except BaseException as error:
        report.update(status="seeding failed; no upgrade should be proposed", error=str(error))
        runner.save("vesting-seed.json", report)
        raise
