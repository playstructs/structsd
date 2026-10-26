#!/usr/bin/env python3
"""Historical composite-vault fixture continuation (pre-September-28 design).

The current upgrade retains native accounts and cannot use this fixture's
composite-vault preflight. Retained-account continuation is covered by real-app
upgrade/restart, held-maturity and voluntary-transition tests; a fresh native
rehearsal remains required. Keep this reader for archived rehearsal evidence.

Observe the historical fixture through native queue completions.

Requires a successful original upgrade and cold checkpoint replay. Runs only
the original private chain, preserves the boundary evidence, and records every
query at one explicit height. This is fixture evidence, not release approval.
"""
import argparse
from copy import deepcopy
from datetime import datetime, timezone
import json
from pathlib import Path
import re
import socket
import time
from urllib.request import urlopen

from run_upgrade_rehearsal import Rehearsal
from seed_upgrade_vesting import account_value, complete_rows
from upgrade_checkpoint import file_hash
from verify_upgrade_bank import amount, attributes, coins
from verify_upgrade_vesting import accounts, original_schedule, VAULT
from vesting_schedule_balances import coin_map, schedule_balances, vault_balances


PASSED = "rehearsal checks passed; release approval remains outstanding"
REPORTS = {
    "staking-verification.json": ("differences",),
    "bank-verification.json": ("balance_differences", "migration_differences"),
    "vesting-verification.json": ("differences",),
    "grid-infusion-verification.json": ("differences",),
    "infusion-backing-verification.json": ("differences",),
    "activation-verification.json": ("differences",),
}


def read(path):
    return json.loads(path.read_text())


def time_ns(value):
    match = re.fullmatch(r"(\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d)(?:\.(\d{1,9}))?Z", value or "")
    if not match:
        raise ValueError("invalid native UTC timestamp")
    date = datetime.strptime(match[1], "%Y-%m-%dT%H:%M:%S").replace(tzinfo=timezone.utc)
    delta = date - datetime(1970, 1, 1, tzinfo=timezone.utc)
    return (delta.days * 86400 + delta.seconds) * 10**9 + int((match[2] or "").ljust(9, "0"))


def exact(left, right, label):
    if json.dumps(left, sort_keys=True) != json.dumps(right, sort_keys=True):
        raise ValueError(label)


def account_defaults(account):
    """Expand only known omitted protobuf defaults; preserve explicit bad values."""
    value = deepcopy(account)
    value.pop("@type", None)
    base = value.get("base_vesting_account", value)
    auth = base["base_account"]
    for key, default in (("pub_key", None), ("account_number", "0"), ("sequence", "0")):
        auth.setdefault(key, default)
    schedules = value.get("schedules", [base] if "base_vesting_account" in value else [])
    for schedule in schedules:
        for key in ("delegated_free", "delegated_vesting"):
            schedule.setdefault(key, [])
        if "kind" in schedule:
            schedule.setdefault("start_time", "0")
            schedule.setdefault("end_time", "0")
            schedule.setdefault("periods", [])
    if "schedules" in value:
        value.setdefault("liquid_delegated", [])
    return value


def spendable_map(rows):
    # Bank spendability queries can return an explicit zero coin. Persisted
    # vesting coin sets still use the stricter positive-only validator.
    result = {}
    for row in rows:
        if not isinstance(row, dict):
            raise ValueError("invalid spendable coin")
        denom = next(iter(coin_map([{**row, "amount": "1"}])))
        if denom in result:
            raise ValueError("duplicate spendable denomination")
        result[denom] = amount(row.get("amount"))
    return {denom: value for denom, value in result.items() if value}


def pending_entries(rows, address):
    result, validators = {}, set()
    for row in rows:
        validator = row.get("validator_address")
        if row.get("delegator_address") != address or not validator or validator in validators:
            raise ValueError("invalid or duplicate pending validator")
        validators.add(validator)
        if not isinstance(row.get("entries"), list) or not row["entries"]:
            raise ValueError("pending validator has no entries")
        for entry in row["entries"]:
            identity = entry.get("unbonding_id")
            if not isinstance(identity, str) or not re.fullmatch(r"[1-9][0-9]*", identity) or identity in result:
                raise ValueError("invalid or duplicate pending unbonding id")
            if entry.get("unbonding_on_hold_ref_count", "0") != "0":
                raise ValueError("fixture unbonding is on hold")
            time_ns(entry.get("completion_time"))
            coin_map([{"denom": "ualpha", "amount": entry.get("balance")}])
            result[identity] = {"validator_address": validator, "unbonding_on_hold_ref_count": "0", **entry}
    return result


def verify_observation(snapshot, baseline, sent):
    """Check bank locks independently of auth counters and queue disappearance."""
    at = time_ns(snapshot["block_time"])
    address = baseline["address"]
    vault = account_defaults(account_value(snapshot["account"], VAULT))
    expected = account_defaults(baseline["vault"])
    pending = pending_entries(complete_rows(snapshot["unbondings"], "unbonding_responses"), address)
    remaining = {key: row for key, row in baseline["pending"].items() if time_ns(row["completion_time"]) > at}
    exact(pending, remaining, "unbonding entries differ from their individual completion times")
    refund = sum(int(row["balance"]) for key, row in baseline["pending"].items() if key not in remaining)
    liquid = coin_map(expected.get("liquid_delegated"))
    if set(liquid) != {"ualpha"} or liquid["ualpha"] < refund:
        raise ValueError("fixture requires ordinary stake to cover all refunds")
    expected["liquid_delegated"] = [{"denom": "ualpha", "amount": str(liquid["ualpha"] - refund)}] if liquid["ualpha"] > refund else []
    exact(vault, expected, "vault identity, schedules or refund accounting changed unexpectedly")
    delegations = [row["delegation"] for row in complete_rows(snapshot["delegations"], "delegation_responses")]
    exact(sorted(delegations, key=lambda row: row["validator_address"]), baseline["delegations"], "active delegations changed during continuation")
    balance = coin_map(complete_rows(snapshot["balances"], "balances"))
    spendable = spendable_map(complete_rows(snapshot["spendable"], "balances"))
    expected_balance = dict(baseline["balances"])
    expected_balance["ualpha"] = expected_balance.get("ualpha", 0) + refund - sent
    exact(balance, expected_balance, "vault balance does not equal original balance plus refunds minus sends")
    locked = vault_balances(vault, at // 10**9)["locked"]
    expected_spendable = {denom: value - locked.get(denom, 0) for denom, value in balance.items()}
    if any(value < 0 for value in expected_spendable.values()) or set(locked) - set(balance):
        raise ValueError("vault has insufficient backing for modeled locks")
    exact(spendable, {d: v for d, v in expected_spendable.items() if v}, "native bank spendability differs from independent vesting model")
    recipient = dict(baseline["recipient_balances"])
    recipient["ualpha"] = recipient.get("ualpha", 0) + sent
    exact(coin_map(complete_rows(snapshot["recipient"], "balances")), recipient, "recipient balance does not match confirmed sends")
    control = baseline["unassociated"]
    control_account = account_value(snapshot["control_account"], control["@type"])
    exact(account_defaults(control_account), account_defaults(control), "unassociated vesting account changed")
    exact(coin_map(complete_rows(snapshot["control_balances"], "balances")), {"ualpha": 2000000}, "unassociated balance changed")
    control_lock = schedule_balances(original_schedule(control), at // 10**9)["locked"].get("ualpha", 0)
    exact(spendable_map(complete_rows(snapshot["control_spendable"], "balances")), {"ualpha": 2000000 - control_lock} if control_lock < 2000000 else {}, "unassociated spendability differs")
    return {"height": snapshot["height"], "block_time": snapshot["block_time"], "refunded_ualpha": refund,
        "pending_ids": sorted(pending), "sent_ualpha": sent, "locked": locked, "spendable": spendable}


def verify_transfer(delivered, txhash, address, recipient, quantity, allowed):
    if delivered.get("txhash") != txhash or int(delivered.get("height", 0)) <= 0:
        raise ValueError("send did not resolve to the requested committed transaction")
    code = delivered.get("code")
    if type(code) is not int:
        raise ValueError("transaction has no explicit delivery code")
    movements = []
    for event in delivered.get("events", []):
        if event.get("type") in ("coin_spent", "coin_received"):
            data = attributes(event)
            subject = data["spender" if event["type"] == "coin_spent" else "receiver"]
            value = coins(data["amount"])
            # These fixture sends pay zero fees. Check all nonzero movement,
            # including any unexpected debit/credit involving another account.
            if any(value.values()):
                movements.append((event["type"], subject, value))
    if allowed:
        if code != 0 or movements != [("coin_spent", address, {"ualpha": quantity}), ("coin_received", recipient, {"ualpha": quantity})]:
            raise ValueError("allowed send did not deliver exactly the intended bank movements")
    else:
        log = delivered.get("raw_log", "")
        if code == 0 or "spendable balance" not in log or "is smaller than" not in log or "insufficient funds" not in log or movements:
            raise ValueError("locked send was not refused by the native spendable-balance check")


def verify_maturity_events(previous, current, results, entry, address, pool):
    height = int(current["height"])
    if int(previous["height"]) != height - 1 or int(results["height"]) != height:
        raise ValueError("maturity evidence must cover adjacent headers and matching block results")
    if previous["chain_id"] != current["chain_id"]:
        raise ValueError("maturity headers belong to different chains")
    if not time_ns(previous["time"]) < time_ns(entry["completion_time"]) <= time_ns(current["time"]):
        raise ValueError("maturity block does not straddle the exact completion time")
    if results.get("txs_results"):
        raise ValueError("fixture maturity block contains unexpected transactions")
    expected = {"ualpha": int(entry["balance"])}
    completions, credits, debits = [], [], []
    for event in results["finalize_block_events"] or []:
        kind = event["type"]
        if kind not in ("complete_unbonding", "coin_received", "coin_spent"):
            continue
        data = attributes(event)
        if kind == "complete_unbonding" and data["delegator"] == address:
            completions.append((data["validator"], coins(data["amount"])))
        if kind == "coin_received" and data["receiver"] == address:
            credits.append(coins(data["amount"]))
        if kind == "coin_spent" and data["spender"] == pool:
            debits.append(coins(data["amount"]))
    if completions != [(entry["validator_address"], expected)] or credits != [expected] or debits != [expected]:
        raise ValueError("maturity must emit one exact completion, vault credit and unbonded-pool debit")
    return {"height": str(height), "unbonding_id": entry["unbonding_id"],
        "validator": entry["validator_address"], "vault": address, "pool": pool, "refund": expected}


class Continuation(Rehearsal):
    def __init__(self, directory, replay, binary, output_name="vesting-continuation"):
        super().__init__(directory, binary)
        self.replay = replay.resolve(strict=True)
        if not re.fullmatch(r"vesting-continuation(?:-[a-z0-9]+)*", output_name):
            raise ValueError("invalid continuation output name")
        self.output = self.directory / output_name
        self.sent = 0
        self.sequence = 0

    def record(self, name, value):
        with (self.output / name).open("x") as stream:
            json.dump(value, stream, indent=2)
            stream.write("\n")

    def preflight_continuation(self):
        if self.output.exists() or self.home.is_symlink() or self.manifest.get("chain_home") != str(self.home):
            raise ValueError("continuation requires an unused result directory and the original private home")
        if self.manifest.get("vesting_fixture_schedule_seconds") != 1800:
            raise ValueError("continuation requires the isolated 1800-second fixture")
        if file_hash(self.new) != self.manifest.get("new_binary_sha256"):
            raise ValueError("continuation candidate changed")
        replay_manifest = read(self.replay / "manifest.json")
        checkpoint = self.directory / "upgrade-checkpoint"
        if replay_manifest.get("checkpoint") != str(checkpoint) or replay_manifest.get("checkpoint_sha256") != file_hash(checkpoint / "checkpoint.json"):
            raise ValueError("cold replay does not belong to this checkpoint")
        hashes = {}
        for directory, manifest in ((self.directory, self.manifest), (self.replay, replay_manifest)):
            if manifest.get("status") != PASSED or manifest.get("new_binary_sha256") != self.manifest["new_binary_sha256"] or manifest.get("chain_id") != self.manifest["chain_id"]:
                raise ValueError("original upgrade and matching cold replay must both pass")
            inputs = {name: file_hash(directory / name) for name in ("before.json", "after.json", "upgrade-block-results.json", "vault-map.json")}
            for name, fields in REPORTS.items():
                report = read(directory / name)
                if report.get("input_sha256") != inputs or any(report.get(field) != [] for field in fields):
                    raise ValueError("missing, stale or failed upgrade verification report")
                hashes[str(directory / name)] = file_hash(directory / name)
            hashes.update({str(directory / name): digest for name, digest in inputs.items()})
        if file_hash(self.directory / "before.json") != file_hash(self.replay / "before.json"):
            raise ValueError("cold replay used a different boundary")
        after = read(self.directory / "after.json")
        if after["app_state"]["staking"]["params"]["unbonding_time"] != "600s":
            raise ValueError("continuation requires isolated 600-second unbonding")
        seed = read(self.directory / "vesting-seed.json")
        if seed.get("status") != "native vesting seed verified; upgrade migration pending" or seed.get("schedule_seconds") != 1800 or len(seed.get("transactions", [])) != 18:
            raise ValueError("complete short native vesting seed is required")
        source_types = [row["account_type"] for row in seed["sources"]]
        if sorted(source_types) != sorted(["DelayedVestingAccount", "ContinuousVestingAccount", "DelayedVestingAccount", "PeriodicVestingAccount", "PermanentLockedAccount", "BaseAccount"]):
            raise ValueError("native seed is missing a required account type")
        owner = seed["player_id"]
        address = read(self.directory / "vault-map.json")[owner]
        exported_accounts = accounts(after)
        pools = [address for address, account in exported_accounts.items()
            if account.get("@type") == "/cosmos.auth.v1beta1.ModuleAccount" and account.get("name") == "not_bonded_tokens_pool"]
        if len(pools) != 1:
            raise ValueError("fixture must contain exactly one native unbonded pool account")
        vault = exported_accounts[address]
        if vault.get("@type") != VAULT or vault.get("owner_id") != owner or sorted(row["kind"] for row in vault["schedules"]) != [1, 2, 3, 4]:
            raise ValueError("fixture must migrate all four schedules into the owner's vault")
        pending = pending_entries([row for row in after["app_state"]["staking"]["unbonding_delegations"] if row["delegator_address"] == address], address)
        if len(pending) != 2 or any(row["balance"] != "100000" for row in pending.values()) or coin_map(vault.get("liquid_delegated")) != {"ualpha": 800000}:
            raise ValueError("fixture must contain two separate refunds and ordinary delegated backing")
        # keys show is a local, read-only keyring lookup; no private material is returned.
        recipient = self.cli(self.new, "keys", "show", "bob", "--keyring-backend", "test")["address"]
        balances = {row["address"]: coin_map(row["coins"]) for row in after["app_state"]["bank"]["balances"]}
        control_rows = [account for account in exported_accounts.values() if account.get("@type") == "/cosmos.vesting.v1beta1.DelayedVestingAccount"]
        if len(control_rows) != 1 or recipient == address:
            raise ValueError("fixture must retain one unassociated delayed control and a separate recipient")
        self.baseline = {"address": address, "vault": vault, "pending": pending, "balances": balances[address], "unbonded_pool": pools[0],
            "recipient": recipient, "recipient_balances": balances.get(recipient, {}), "unassociated": control_rows[0],
            "delegations": sorted([row for row in after["app_state"]["staking"]["delegations"] if row["delegator_address"] == address], key=lambda row: row["validator_address"])}
        periodic = next(row for row in vault["schedules"] if row["kind"] == 3)
        first_period = (int(periodic["start_time"]) + int(periodic["periods"][0]["length"])) * 10**9
        self.boundaries = [("periodic-first", first_period)] + [("unbonding-" + key, time_ns(row["completion_time"])) for key, row in pending.items()] + [("all-timed-schedules-ended", int(seed["end_time"]) * 10**9)]
        self.boundaries.sort(key=lambda item: item[1])
        if len({at for _, at in self.boundaries}) != len(self.boundaries):
            raise ValueError("fixture completion boundaries must be distinct")
        for port in (26656, 26657, 1317, 9090):
            with socket.socket() as connection:
                connection.settimeout(0.2)
                if connection.connect_ex(("127.0.0.1", port)) == 0:
                    raise ValueError(f"local port {port} is occupied; no process was stopped")
        for path in (self.new, self.directory / "vesting-seed.json", self.directory / "manifest.json",
                self.replay / "manifest.json", checkpoint / "checkpoint.json", *sorted(Path(__file__).parent.glob("*.py"))):
            hashes[str(path)] = file_hash(path)
        return {"input_sha256": hashes, "baseline": self.baseline, "boundaries_ns": self.boundaries,
            "scope": "native bank locks, allowed/refused sends, each pending completion and its public refund events, timed unlocks, permanent locks and database reopen"}

    def live_status(self):
        if time.monotonic() >= self.deadline or self.node.poll() is not None:
            raise RuntimeError("continuation timed out or its owned node exited")
        status = self.cli(self.new, "status")
        if status["node_info"]["network"] != self.manifest["chain_id"]:
            raise ValueError("status belongs to a different chain")
        return status["sync_info"]

    def maturity_events(self, label, snapshot):
        entry = self.baseline["pending"][label.removeprefix("unbonding-")]
        target = time_ns(entry["completion_time"])
        queries, headers = {}, {}

        def rpc(method, height):
            self.live_status()  # Enforce owned-node liveness and the overall deadline.
            with urlopen(f"http://127.0.0.1:26657/{method}?height={height}", timeout=30) as response:
                result = json.load(response)
            queries[f"{method}-{height}"] = result
            if "error" in result:
                raise ValueError("maturity RPC failed")
            return result["result"]

        def header(height):
            if height not in headers:
                value = rpc("block", height)["block"]["header"]
                if int(value["height"]) != height or value["chain_id"] != self.manifest["chain_id"]:
                    raise ValueError("maturity header does not match the requested chain and height")
                time_ns(value["time"])
                headers[height] = value
            return headers[height]

        try:
            low = int(read(self.directory / "after.json")["initial_height"]) - 1
            high = int(snapshot["height"])
            if low <= 0 or low >= high or not time_ns(header(low)["time"]) < target <= time_ns(header(high)["time"]):
                raise ValueError("maturity is outside the observed continuation range")
            # Find the first committed header at/after this entry's exact deadline.
            while high - low > 1:
                middle = (low + high) // 2
                if time_ns(header(middle)["time"]) < target:
                    low = middle
                else:
                    high = middle
            result = rpc("block_results", high)
            verification = verify_maturity_events(header(low), header(high), result, entry,
                self.baseline["address"], self.baseline["unbonded_pool"])
        finally:
            self.record(label + "-maturity-queries.json", queries)
        verification["query_sha256"] = file_hash(self.output / (label + "-maturity-queries.json"))
        self.record(label + "-maturity-verification.json", verification)

    def observe(self, label, before=None):
        status = self.live_status()
        height, block_time = status["latest_block_height"], status["latest_block_time"]
        if before is not None and time_ns(block_time) >= before:
            raise ValueError("missed a distinct fixture boundary; cannot claim its observation")
        address, recipient = self.baseline["address"], self.baseline["recipient"]
        control = self.baseline["unassociated"]["base_vesting_account"]["base_account"]["address"]
        snapshot = {"height": height, "block_time": block_time, "label": label}
        self.sequence += 1
        prefix = f"{self.sequence:02d}-{label}"
        try:
            for name, arguments in (
                ("account", ("auth", "account", address)),
                ("balances", ("bank", "balances", address, "--page-limit", "100")),
                ("spendable", ("bank", "spendable-balances", address, "--page-limit", "100")),
                ("delegations", ("staking", "delegations", address, "--page-limit", "100")),
                ("unbondings", ("staking", "unbonding-delegations", address, "--page-limit", "100")),
                ("recipient", ("bank", "balances", recipient, "--page-limit", "100")),
                ("control_account", ("auth", "account", control)),
                ("control_balances", ("bank", "balances", control, "--page-limit", "100")),
                ("control_spendable", ("bank", "spendable-balances", control, "--page-limit", "100"))):
                snapshot[name] = self.cli(self.new, "query", *arguments, "--height", height)
            verification = verify_observation(snapshot, self.baseline, self.sent)
        finally:
            self.record(prefix + "-queries.json", snapshot)
        self.record(prefix + "-verification.json", verification)
        print(f"{label}: height {height}, refunds {verification['refunded_ualpha']}, pending {verification['pending_ids']}, locks {verification['locked']}", flush=True)
        return snapshot

    def send(self, label, quantity, allowed):
        address, recipient = self.baseline["address"], self.baseline["recipient"]
        response = self.cli(self.new, "tx", "structs", "player-send", address, recipient, str(quantity) + "ualpha",
            "--from", "alice", "--keyring-backend", "test", "--chain-id", self.manifest["chain_id"],
            "--gas", "2000000", "--fees", "0ualpha", "--yes")
        self.record(label + "-broadcast.json", response)
        if int(response.get("code", 0)) != 0 or not response.get("txhash"):
            raise ValueError("send was refused before delivery; no native spendability proof")
        deadline = min(self.deadline, time.monotonic() + 30)
        while time.monotonic() < deadline:
            try:
                delivered = self.cli(self.new, "query", "tx", response["txhash"])
            except RuntimeError:
                if self.node.poll() is not None:
                    raise RuntimeError("owned node exited before send delivery")
                time.sleep(0.5)
                continue
            self.record(label + "-delivered.json", delivered)
            verify_transfer(delivered, response["txhash"], address, recipient, quantity, allowed)
            if allowed:
                self.sent += quantity
            return
        raise RuntimeError("send delivery was not confirmed")

    def transfer_checks(self, label, before=None):
        snapshot = self.observe(label, before)
        quantity = coin_map(complete_rows(snapshot["balances"], "balances"))["ualpha"]
        self.send(label + "-locked-send", quantity, False)
        self.observe(label + "-refused", before)
        self.send(label + "-allowed-send", 1, True)
        return self.observe(label + "-sent", before)

    def run_continuation(self):
        preflight = self.preflight_continuation()
        self.output.mkdir(mode=0o700)
        self.record("preflight.json", preflight)
        self.deadline = time.monotonic() + 2400
        try:
            self.node = self.start([str(self.new), "start", "--home", str(self.home), "--p2p.laddr", "tcp://127.0.0.1:26656"], self.output.name + "/node.log")
            self.wait_height(self.new, self.node, int(read(self.directory / "after.json")["initial_height"]), timeout=60)
            self.transfer_checks("initial", self.boundaries[0][1])
            for index, (label, target) in enumerate(self.boundaries):
                while time_ns(self.live_status()["latest_block_time"]) < target:
                    time.sleep(0.5)
                next_target = self.boundaries[index + 1][1] if index + 1 < len(self.boundaries) else None
                snapshot = self.observe(label, next_target)
                if label.startswith("unbonding-"):
                    self.maturity_events(label, snapshot)
            final = self.transfer_checks("timed-unlocks-complete")
            height = int(final["height"])
            self.stop(self.node)
            self.export(self.new, height, self.output.name + "/before-reopen.json")
            self.node = self.start([str(self.new), "start", "--home", str(self.home), "--p2p.laddr", "tcp://127.0.0.1:26656"], self.output.name + "/reopened-node.log")
            self.wait_height(self.new, self.node, height + 3, timeout=60)
            final = self.transfer_checks("reopened")
            self.stop(self.node)
            self.export(self.new, int(final["height"]), self.output.name + "/after-reopen.json")
            self.record("result.json", {"status": "native continuation checks passed", "release_approved": False,
                "confirmed_sent_ualpha": self.sent, "final_height": final["height"], "boundaries_ns": self.boundaries})
        except BaseException as error:
            self.record("failure.json", {"status": "continuation incomplete", "error": str(error), "type": type(error).__name__, "release_approved": False})
            raise
        finally:
            self.cleanup()


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("directory", type=Path)
    parser.add_argument("--replay", type=Path, required=True)
    parser.add_argument("--new-binary", type=Path, required=True)
    parser.add_argument("--output-name", default="vesting-continuation", help="unused evidence-directory name for a separately recorded continuation attempt")
    args = parser.parse_args()
    Continuation(args.directory, args.replay, args.new_binary, args.output_name).run_continuation()
