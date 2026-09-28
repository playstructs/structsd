"""Continuation controls are offline; native execution is separate evidence."""
from copy import deepcopy
from datetime import datetime, timezone
from io import BytesIO
import json
from pathlib import Path
import tempfile
import time
import unittest
from unittest.mock import Mock, patch

from continue_upgrade_vesting import Continuation, PASSED, REPORTS, account_defaults, pending_entries, spendable_map, time_ns, verify_maturity_events, verify_observation, verify_transfer
from upgrade_checkpoint import file_hash


def coin(value):
    return [{"denom": "ualpha", "amount": str(value)}] if value else []


def utc(seconds):
    return datetime.fromtimestamp(seconds, timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def page(field, rows):
    return {field: rows, "pagination": {"next_key": None}}


def fixture():
    schedules = []
    for kind, delegated in ((1, 800000), (2, 1000000), (3, 600000), (4, 500000)):
        schedules.append({"kind": kind, "source_address": str(kind), "start_time": "1000" if kind in (1, 3) else "0",
            "end_time": "1400" if kind != 4 else "0", "original_vesting": coin(2000000), "delegated_vesting": coin(delegated),
            "delegated_free": [], "periods": [{"length": "100", "amount": coin(750000)}, {"length": "300", "amount": coin(1250000)}] if kind == 3 else []})
    vault = {"@type": "/structs.structs.VaultVestingAccount", "base_account": {"address": "vault", "sequence": "0"},
        "owner_id": "1-1", "schedules": schedules, "liquid_delegated": coin(800000)}
    control = {"@type": "/cosmos.vesting.v1beta1.DelayedVestingAccount", "base_vesting_account": {
        "base_account": {"address": "structs1xxshfg0neejpykgm4x9hl2pnl8p57kvxghj0u4"},
        "original_vesting": coin(2000000), "delegated_free": [], "delegated_vesting": [], "end_time": "1400"}}
    entries = [{"unbonding_id": str(i), "creation_height": str(i + 10), "completion_time": utc(at).replace("Z", fraction + "Z"),
        "balance": "100000", "initial_balance": "100000", "unbonding_on_hold_ref_count": "0"}
        for i, at, fraction in ((1, 1150, ".100000001"), (2, 1250, ".000000002"))]
    return {"address": "vault", "vault": vault, "unbonded_pool": "pool",
        "pending": pending_entries([{"delegator_address": "vault", "validator_address": "validator", "entries": entries}], "vault"),
        "balances": {"ualpha": 6000000}, "recipient": "recipient", "recipient_balances": {"ualpha": 2000000},
        "unassociated": control, "delegations": [{"delegator_address": "vault", "validator_address": "validator", "shares": "3500000.000000000000000000"}]}


def snapshot(baseline, at=1000, completed=0, locked=5100000, sent=0):
    vault = deepcopy(baseline["vault"])
    vault["liquid_delegated"] = coin(800000 - completed * 100000)
    entries = [{k: v for k, v in row.items() if k != "validator_address"} for key, row in baseline["pending"].items() if int(key) > completed]
    balance = 6000000 + completed * 100000 - sent
    return {"height": "42", "block_time": utc(at), "account": {"account": vault},
        "balances": page("balances", coin(balance)), "spendable": page("balances", coin(balance - locked)),
        "delegations": page("delegation_responses", [{"delegation": row} for row in baseline["delegations"]]),
        "unbondings": page("unbonding_responses", [{"delegator_address": "vault", "validator_address": "validator", "entries": entries}] if entries else None),
        "recipient": page("balances", coin(2000000 + sent)), "control_account": {"account": deepcopy(baseline["unassociated"])},
        "control_balances": page("balances", coin(2000000)), "control_spendable": page("balances", coin(2000000 if at >= 1400 else 0))}


def delivered(allowed):
    return {"txhash": "ABC", "height": "42", "code": 0 if allowed else 5,
        "raw_log": "" if allowed else "spendable balance 1ualpha is smaller than 5ualpha: insufficient funds",
        "events": [{"type": kind, "attributes": [{"key": key, "value": address}, {"key": "amount", "value": "1ualpha"}]} for kind, key, address in (
            ("coin_spent", "spender", "vault"), ("coin_received", "receiver", "recipient"))] if allowed else []}


def maturity_block(height="80"):
    def event(kind, **fields):
        return {"type": kind, "attributes": [{"key": key, "value": value} for key, value in fields.items()]}
    return {"height": height, "txs_results": None, "finalize_block_events": [
        event("coin_spent", spender="pool", amount="100000ualpha"),
        event("coin_received", receiver="vault", amount="100000ualpha"),
        event("complete_unbonding", delegator="vault", validator="validator", amount="100000ualpha")]}


class ContinuationTests(unittest.TestCase):
    def test_native_omitted_defaults_preserve_exact_account_and_queue_meaning(self):
        baseline = fixture()
        expected = account_defaults(baseline["vault"])
        observed = deepcopy(baseline["vault"])
        observed["base_account"].pop("sequence")
        for schedule in observed["schedules"]:
            schedule.pop("delegated_free")
            for name in ("periods", "start_time", "end_time"):
                if schedule[name] in ([], "0"):
                    schedule.pop(name)
        self.assertEqual(account_defaults(observed), expected)
        observed["base_account"]["sequence"] = None
        self.assertNotEqual(account_defaults(observed), expected)
        rows = snapshot(baseline)["unbondings"]["unbonding_responses"]
        for entry in rows[0]["entries"]:
            entry.pop("unbonding_on_hold_ref_count")
        self.assertEqual(pending_entries(rows, "vault"), baseline["pending"])
        rows[0]["entries"][0]["unbonding_on_hold_ref_count"] = None
        with self.assertRaises(ValueError):
            pending_entries(rows, "vault")

    def test_spendability_zero_coins_are_not_persisted_vesting_coin_defaults(self):
        self.assertEqual(spendable_map([{"denom": "ualpha", "amount": "0"}]), {})
        self.assertEqual(spendable_map(coin(9)), {"ualpha": 9})
        for rows in ([{"denom": "ualpha", "amount": "0"}] * 2,
                [{"denom": "ualpha", "amount": "-1"}], [{"denom": "ualpha", "amount": 0}],
                [{"denom": "?", "amount": "0"}], [None]):
            with self.assertRaises(ValueError):
                spendable_map(rows)

    def test_maturity_requires_exact_public_completion_and_bank_events(self):
        previous = {"height": "79", "chain_id": "fixture", "time": utc(1150)}
        current = {"height": "80", "chain_id": "fixture", "time": utc(1151)}
        entry = fixture()["pending"]["1"]
        verified = verify_maturity_events(previous, current, maturity_block(), entry, "vault", "pool")
        self.assertEqual(verified["refund"], {"ualpha": 100000})
        for index in range(3):
            for action in ("missing", "duplicate", "amount", "address"):
                results = maturity_block()
                event = results["finalize_block_events"][index]
                if action == "missing":
                    results["finalize_block_events"].pop(index)
                elif action == "duplicate":
                    results["finalize_block_events"].append(deepcopy(event))
                else:
                    key = "amount" if action == "amount" else ("spender", "receiver", "delegator")[index]
                    next(a for a in event["attributes"] if a["key"] == key)["value"] = "99999ualpha" if action == "amount" else "old source"
                with self.subTest(index=index, action=action), self.assertRaises(ValueError):
                    verify_maturity_events(previous, current, results, entry, "vault", "pool")
        wrong_validator = maturity_block()
        wrong_validator["finalize_block_events"][2]["attributes"][1]["value"] = "wrong validator"
        with self.assertRaises(ValueError):
            verify_maturity_events(previous, current, wrong_validator, entry, "vault", "pool")

    def test_maturity_rejects_wrong_height_time_chain_and_transaction_envelope(self):
        previous = {"height": "79", "chain_id": "fixture", "time": utc(1150)}
        current = {"height": "80", "chain_id": "fixture", "time": utc(1151)}
        entry = fixture()["pending"]["1"]
        for before, after, results in (
                ({**previous, "height": "78"}, current, maturity_block()),
                ({**previous, "time": entry["completion_time"]}, current, maturity_block()),
                (previous, {**current, "time": utc(1150)}, maturity_block()),
                (previous, {**current, "chain_id": "another"}, maturity_block()),
                (previous, current, maturity_block("81")),
                (previous, current, {**maturity_block(), "txs_results": [{"code": 0}]})):
            with self.assertRaises(ValueError):
                verify_maturity_events(before, after, results, entry, "vault", "pool")

    @patch("continue_upgrade_vesting.urlopen")
    def test_maturity_capture_locates_first_header_and_preserves_failure(self, urlopen):
        with tempfile.TemporaryDirectory() as temporary:
            runner = self.runner(Path(temporary))
            runner.output.mkdir()
            (runner.directory / "after.json").write_text(json.dumps({"initial_height": "44"}))
            runner.live_status = Mock()
            def response(url, **kwargs):
                height = int(url.split("=")[-1])
                value = maturity_block(str(height)) if "/block_results?" in url else {"block": {"header": {
                    "chain_id": "fixture", "height": str(height), "time": utc(1000 + height * 2)}}}
                return BytesIO(json.dumps({"result": value}).encode())
            urlopen.side_effect = response
            runner.maturity_events("unbonding-1", {"height": "100"})
            report = json.loads((runner.output / "unbonding-1-maturity-verification.json").read_text())
            self.assertEqual(report["height"], "76")
            self.assertLessEqual(urlopen.call_count, 10)
            self.assertTrue((runner.output / "unbonding-1-maturity-queries.json").exists())
            urlopen.side_effect = lambda *a, **k: BytesIO(b'{"error": {"message": "unavailable"}}')
            with self.assertRaisesRegex(ValueError, "RPC failed"):
                runner.maturity_events("unbonding-2", {"height": "150"})
            self.assertIn("block-43", json.loads((runner.output / "unbonding-2-maturity-queries.json").read_text()))
            self.assertFalse((runner.output / "unbonding-2-maturity-verification.json").exists())

    def test_nanosecond_boundaries_never_use_float_or_microsecond_rounding(self):
        self.assertEqual(time_ns("2026-09-15T00:00:00.123456789Z") - time_ns("2026-09-15T00:00:00.123456788Z"), 1)
        self.assertEqual(time_ns("1970-01-01T00:00:00Z"), 0)
        for invalid in ("2026-09-15T00:00:00+00:00", "2026-09-15T00:00:00.1234567890Z", "2026-02-30T00:00:00Z", "", None):
            with self.subTest(invalid=invalid), self.assertRaises(ValueError):
                time_ns(invalid)

    def test_individual_refunds_unlocks_and_permanent_lock(self):
        baseline = fixture()
        for at, completed, locked, sent in ((1000, 0, 5100000, 0), (1100, 0, 3850000, 1),
                (1151, 1, 3595000, 1), (1251, 2, 3150000, 1), (1400, 2, 1500000, 2)):
            with self.subTest(at=at):
                observation = snapshot(baseline, at, completed, locked, sent)
                result = verify_observation(observation, baseline, sent)
                self.assertEqual(result["refunded_ualpha"], completed * 100000)
                self.assertEqual(result["locked"], {"ualpha": locked})

    def test_queue_does_not_complete_a_nanosecond_early(self):
        baseline = fixture()
        observation = snapshot(baseline, 1150, 0, 3600000)
        observation["block_time"] = utc(1150).replace("Z", ".100000000Z")
        verify_observation(observation, baseline, 0)
        observation["block_time"] = utc(1150).replace("Z", ".100000001Z")
        with self.assertRaisesRegex(ValueError, "individual completion"):
            verify_observation(observation, baseline, 0)

    def test_query_and_ledger_tampering_cannot_pass(self):
        baseline = fixture()
        mutations = [
            lambda s: s["balances"]["balances"][0].update(amount="6000001"),
            lambda s: s["spendable"]["balances"][0].update(amount="900001"),
            lambda s: s["recipient"]["balances"][0].update(amount="2000001"),
            lambda s: s["account"]["account"].update(owner_id="1-2"),
            lambda s: s["account"]["account"]["schedules"][0].update(kind=True),
            lambda s: s["account"]["account"].update(liquid_delegated=coin(700000)),
            lambda s: s["unbondings"].update(unbonding_responses=None),
            lambda s: s["delegations"].update(delegation_responses=None),
            lambda s: s["balances"].pop("pagination"),
            lambda s: s["unbondings"]["pagination"].update(next_key="YQ=="),
            lambda s: s["control_spendable"].update(balances=coin(1)),
        ]
        for mutate in mutations:
            observation = snapshot(baseline)
            mutate(observation)
            with self.subTest(mutation=mutate), self.assertRaises(ValueError):
                verify_observation(observation, baseline, 0)

    def test_pending_ids_are_unique_and_holds_are_not_ordinary_maturities(self):
        baseline = fixture()
        rows = snapshot(baseline)["unbondings"]["unbonding_responses"]
        for field, value in (("unbonding_id", "1"), ("unbonding_on_hold_ref_count", "1"), ("balance", "-1")):
            altered = deepcopy(rows)
            altered[0]["entries"][1][field] = value
            with self.assertRaises(ValueError):
                pending_entries(altered, "vault")
        with self.assertRaises(ValueError):
            pending_entries(rows, "wrong owner")

    def test_transfers_require_exact_delivery_bank_movements_and_refusal(self):
        for allowed in (True, False):
            verify_transfer(delivered(allowed), "ABC", "vault", "recipient", 1, allowed)
        for change in ({"height": "0"}, {"txhash": "OTHER"}, {"code": True}, {"events": []}):
            with self.assertRaises(ValueError):
                verify_transfer({**delivered(True), **change}, "ABC", "vault", "recipient", 1, True)
        for change in ({"code": 0}, {"raw_log": "insufficient fee"}, {"raw_log": "permission denied"}, {"events": delivered(True)["events"]}):
            with self.assertRaises(ValueError):
                verify_transfer({**delivered(False), **change}, "ABC", "vault", "recipient", 1, False)
        # Zero-fee fixture transactions must not move other accounts' funds,
        # even when the intended vault/recipient pair happens to be correct.
        for allowed in (True, False):
            for kind, key in (("coin_spent", "spender"), ("coin_received", "receiver")):
                transaction = delivered(allowed)
                transaction["events"].append({"type": kind, "attributes": [
                    {"key": key, "value": "another account"}, {"key": "amount", "value": "1ualpha"}]})
                with self.assertRaises(ValueError):
                    verify_transfer(transaction, "ABC", "vault", "recipient", 1, allowed)

    def runner(self, directory):
        runner = object.__new__(Continuation)
        runner.directory, runner.home = directory, directory / "chain"
        runner.output = directory / "vesting-continuation"
        runner.baseline = fixture()
        runner.new = Path("/unused/candidate")
        runner.manifest = {"chain_id": "fixture"}
        runner.sent, runner.sequence = 0, 0
        runner.deadline = time.monotonic() + 30
        runner.node = Mock()
        runner.node.poll.return_value = None
        return runner

    def test_all_queries_share_one_height_and_partial_evidence_survives(self):
        with tempfile.TemporaryDirectory() as temporary:
            runner = self.runner(Path(temporary))
            runner.output.mkdir()
            observation = snapshot(runner.baseline)
            runner.live_status = Mock(return_value={"latest_block_height": "42", "latest_block_time": utc(1000)})
            fields = ("account", "balances", "spendable", "delegations", "unbondings", "recipient", "control_account", "control_balances", "control_spendable")
            runner.cli = Mock(side_effect=[observation[field] for field in fields])
            runner.observe("initial", 1100 * 10**9)
            self.assertEqual(runner.cli.call_count, 9)
            for invocation in runner.cli.call_args_list:
                self.assertEqual(invocation.args[-2:], ("--height", "42"))
            runner.cli = Mock(side_effect=[observation["account"], RuntimeError("query failed")])
            with self.assertRaises(RuntimeError):
                runner.observe("partial")
            partial = json.loads((runner.output / "02-partial-queries.json").read_text())
            self.assertIn("account", partial)
            self.assertNotIn("balances", partial)
            self.assertFalse((runner.output / "02-partial-verification.json").exists())
            with self.assertRaisesRegex(ValueError, "missed"):
                runner.observe("late", 1000 * 10**9)

    def test_send_uses_candidate_fixed_gas_and_waits_for_delivered_failure(self):
        with tempfile.TemporaryDirectory() as temporary:
            runner = self.runner(Path(temporary))
            runner.output.mkdir()
            runner.cli = Mock(side_effect=[{"code": 0, "txhash": "ABC"}, delivered(False)])
            runner.send("refused", 6000000, False)
            args = runner.cli.call_args_list[0].args
            self.assertEqual(args[:4], (runner.new, "tx", "structs", "player-send"))
            self.assertEqual(args[args.index("--gas") + 1], "2000000")
            self.assertEqual(runner.sent, 0)
            self.assertTrue((runner.output / "refused-delivered.json").exists())
            runner.cli = Mock(return_value={"code": 5, "txhash": "ABC"})
            with self.assertRaisesRegex(ValueError, "before delivery"):
                runner.send("checktx-refused", 1, False)
            self.assertEqual(runner.cli.call_count, 1)

    def test_failure_cleans_up_owned_nodes_and_prevents_overwrite(self):
        with tempfile.TemporaryDirectory() as temporary:
            runner = self.runner(Path(temporary))
            runner.preflight_continuation = Mock(return_value={})
            runner.start = Mock(side_effect=RuntimeError("start failed"))
            runner.cleanup = Mock()
            with self.assertRaises(RuntimeError):
                runner.run_continuation()
            runner.cleanup.assert_called_once()
            self.assertTrue((runner.output / "failure.json").exists())
            with self.assertRaises(FileExistsError):
                runner.record("failure.json", {})
            with self.assertRaisesRegex(ValueError, "unused result directory"):
                Continuation.preflight_continuation(runner)

    @patch("continue_upgrade_vesting.socket.socket")
    def test_preflight_requires_matching_successful_and_unchanged_replay(self, socket):
        socket.return_value.__enter__.return_value.connect_ex.return_value = 1
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary).resolve()
            replay = directory / "replay"
            replay.mkdir()
            checkpoint = directory / "upgrade-checkpoint"
            checkpoint.mkdir()
            (checkpoint / "checkpoint.json").write_text("{}")
            binary = directory / "candidate"
            binary.write_text("never executed")
            baseline = fixture()
            after = {"initial_height": "43", "app_state": {
                "auth": {"accounts": [baseline["vault"], baseline["unassociated"], {
                    "@type": "/cosmos.auth.v1beta1.ModuleAccount", "name": "not_bonded_tokens_pool", "base_account": {"address": "pool"}}]},
                "staking": {"params": {"unbonding_time": "600s"}, "delegations": baseline["delegations"],
                    "unbonding_delegations": snapshot(baseline)["unbondings"]["unbonding_responses"]},
                "bank": {"balances": [{"address": "vault", "coins": coin(6000000)}, {"address": "recipient", "coins": coin(2000000)}]}}}
            def save(path, value):
                path.write_text(json.dumps(value))
            manifest = {"chain_home": str(directory / "chain"), "binary": str(binary), "status": PASSED,
                "chain_id": "fixture", "vesting_fixture_schedule_seconds": 1800, "new_binary_sha256": file_hash(binary)}
            save(directory / "manifest.json", manifest)
            replay_manifest = {**manifest, "checkpoint": str(checkpoint), "checkpoint_sha256": file_hash(checkpoint / "checkpoint.json")}
            save(replay / "manifest.json", replay_manifest)
            for target in (directory, replay):
                for name, value in (("before.json", {"initial_height": "40"}), ("after.json", after), ("upgrade-block-results.json", []), ("vault-map.json", {"1-1": "vault"})):
                    save(target / name, value)
                hashes = {name: file_hash(target / name) for name in ("before.json", "after.json", "upgrade-block-results.json", "vault-map.json")}
                for name, fields in REPORTS.items():
                    save(target / name, {**{field: [] for field in fields}, "input_sha256": hashes})
            save(directory / "vesting-seed.json", {"status": "native vesting seed verified; upgrade migration pending", "schedule_seconds": 1800,
                "transactions": [{}] * 18, "player_id": "1-1", "end_time": "1400", "sources": [{"account_type": name} for name in (
                    "DelayedVestingAccount", "ContinuousVestingAccount", "DelayedVestingAccount", "PeriodicVestingAccount", "PermanentLockedAccount", "BaseAccount")]})
            runner = Continuation(directory, replay, binary)
            for output in ("../vesting-continuation", "vesting-continuation/child", "anything", "vesting-continuation-"):
                with self.assertRaisesRegex(ValueError, "output name"):
                    Continuation(directory, replay, binary, output)
            separate = Continuation(directory, replay, binary, "vesting-continuation-review")
            self.assertEqual(separate.output, directory / "vesting-continuation-review")
            runner.cli = Mock(return_value={"address": "recipient"})
            report = runner.preflight_continuation()
            self.assertEqual(len(report["boundaries_ns"]), 4)
            self.assertFalse(runner.output.exists())
            for path, value, reason in (
                (replay / "manifest.json", {**replay_manifest, "status": "failed"}, "must both pass"),
                (replay / "manifest.json", {**replay_manifest, "checkpoint": "/another/checkpoint"}, "belong"),
                (directory / "bank-verification.json", {"balance_differences": [], "migration_differences": []}, "verification report"),
                (replay / "after.json", {"tampered": True}, "verification report")):
                original = path.read_bytes()
                save(path, value)
                with self.subTest(reason=reason), self.assertRaisesRegex(ValueError, reason):
                    runner.preflight_continuation()
                path.write_bytes(original)
            socket.return_value.__enter__.return_value.connect_ex.return_value = 0
            with self.assertRaisesRegex(ValueError, "occupied"):
                runner.preflight_continuation()
            binary.write_text("modified")
            with self.assertRaisesRegex(ValueError, "candidate changed"):
                runner.preflight_continuation()
            self.assertFalse(runner.output.exists())


if __name__ == "__main__":
    unittest.main()
