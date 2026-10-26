"""Offline controls for transaction-only vesting fixture population."""
import copy
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import Mock, patch

from seed_upgrade_vesting import legacy_account, proof_tool_input, seed_vesting, source_bank_balance, verify_seed_snapshot
from test_upgrade_runner import rehearsal


def account(address, kind="DelayedVestingAccount"):
    return {"account": {"type": "/cosmos.vesting.v1beta1." + kind, "value": {
        "base_vesting_account": {"base_account": {"address": address},
            "original_vesting": [{"denom": "ualpha", "amount": "2000000"}],
            "delegated_vesting": [{"denom": "ualpha", "amount": "800000"}]}}}}


def proof(name, player):
    return {"key_name": name, "player_id": player, "address": "structs1" + name.replace("_", ""),
        "public_key": "02" + "11" * 32, "signature": "22" * 64}


class FakeRunner:
    def __init__(self, directory):
        self.directory = directory
        self.home = directory / "chain"
        self.old = directory / "old"
        self.calls, self.sent, self.accounts, self.owners = [], [], {}, {}
        self.pending = {}
        self.delegated = {}
        self.balances = {}
        self.fail_delegation = False

    def save(self, name, document):
        (self.directory / name).write_text(json.dumps(document))

    def height(self, binary):
        return 42

    def tx(self, *args, signer="alice"):
        self.sent.append((args, signer))
        events = []
        def movement(address, value):
            self.balances[address] = self.balances.get(address, 0) + value
            events.append({"type": "coin_received" if value >= 0 else "coin_spent", "attributes": [
                {"key": "receiver" if value >= 0 else "spender", "value": address},
                {"key": "amount", "value": str(abs(value)) + "ualpha"}]})
        if args[:2] == ("vesting", "create-vesting-account"):
            self.accounts[args[2]] = account(args[2], "DelayedVestingAccount" if "--delayed" in args else "ContinuousVestingAccount")
            base = self.accounts[args[2]]["account"]["value"]["base_vesting_account"]
            base["delegated_vesting"], base["end_time"] = [], args[4]
            movement(args[2], 2000000)
        elif args[:2] in (("vesting", "create-periodic-vesting-account"), ("vesting", "create-permanent-locked-account")):
            periodic = args[1] == "create-periodic-vesting-account"
            self.accounts[args[2]] = account(args[2], "PeriodicVestingAccount" if periodic else "PermanentLockedAccount")
            value = self.accounts[args[2]]["account"]["value"]
            value["base_vesting_account"]["delegated_vesting"] = []
            if periodic:
                data = json.loads(Path(args[3]).read_text())
                value["start_time"] = str(data["start_time"])
                value["vesting_periods"] = [{"length": str(period["length_seconds"]),
                    "amount": [{"denom": "ualpha", "amount": period["coins"][:-6]}]} for period in data["periods"]]
                value["base_vesting_account"]["end_time"] = str(data["start_time"] + sum(period["length_seconds"] for period in data["periods"]))
            else:
                value["base_vesting_account"]["end_time"] = "0"
            movement(args[2], 2000000)
        elif args[:2] == ("structs", "address-register"):
            self.owners[args[3]] = args[2]
            if args[3] not in self.accounts:
                self.accounts[args[3]] = {"account": {"type": "/cosmos.auth.v1beta1.BaseAccount", "value": {"address": args[3]}}}
            if "continuous" in args[3]:
                movement(args[3], -1)
        elif args[:2] == ("bank", "send"):
            if args[3] not in self.owners:
                raise AssertionError("ordinary source must register before funding")
            movement(args[3], int(args[4][:-6]))
        elif args[:2] == ("staking", "delegate"):
            if self.fail_delegation:
                raise RuntimeError("injected delegation refusal")
            address = proof(signer, "1-1")["address"]
            amount = args[3][:-6]
            self.delegated[address] = int(amount)
            base = self.accounts[address]["account"]["value"].get("base_vesting_account")
            if base is not None:
                base["delegated_vesting"] = [{"denom": "ualpha", "amount": amount}]
            movement(address, -int(amount))
        elif args[:2] == ("staking", "unbond"):
            address = proof(signer, "1-1")["address"]
            self.pending[address] = int(args[3][:-6])
            movement(address, 17)  # Distribution withdraws rewards before shares change.
        return {"height": "30", "txhash": "A" * 64, "code": 0, "events": events}

    def cli(self, binary, *args):
        self.calls.append(args)
        if args[:3] == ("keys", "show", "alice"):
            return {"address": "validator" if "--bech" in args else "structs1alice"}
        if args == ("status",):
            return {"sync_info": {"latest_block_time": "2026-09-14T12:00:00.123456789Z"}}
        if args[:3] == ("query", "staking", "validator"):
            return {"validator": {"operator_address": "validator"}}
        address = args[3]
        if args[:3] == ("query", "structs", "address"):
            return {"address": address, "playerId": "1-1" if address == "structs1alice" else self.owners.get(address, "")}
        if args[:3] == ("query", "auth", "account"):
            return self.accounts[address]
        if args[:3] == ("query", "staking", "unbonding-delegations"):
            rows = [] if address not in self.pending else [{"delegator_address": address, "validator_address": "validator",
                "entries": [{"creation_height": "30", "balance": str(self.pending[address]), "initial_balance": str(self.pending[address])}]}]
            return {"unbonding_responses": rows, "pagination": {}}
        if args[:3] == ("query", "staking", "delegations"):
            amount = self.delegated.get(address, 0)
            rows = [] if not amount else [{"delegation": {"delegator_address": address, "validator_address": "validator"},
                "balance": {"denom": "ualpha", "amount": str(amount - self.pending.get(address, 0))}}]
            return {"delegation_responses": rows, "pagination": {}}
        if args[:3] == ("query", "bank", "balances"):
            return {"balances": [{"denom": "ualpha", "amount": str(self.balances[address])}], "pagination": {}}
        raise AssertionError(args)


class VestingSeedTests(unittest.TestCase):
    def setup_runner(self, directory):
        tool = directory / "legacyvestingproof"
        tool.write_text("not invoked")
        tool.chmod(0o700)
        return FakeRunner(directory), proof_tool_input(tool)

    def test_sources_use_real_transaction_shapes_and_one_snapshot_height(self):
        with tempfile.TemporaryDirectory() as directory:
            runner, tool = self.setup_runner(Path(directory))
            result = seed_vesting(runner, tool, proof)
            self.assertEqual(result["status"], "native vesting seed verified; upgrade migration pending")
            self.assertEqual(len(runner.sent), 18)
            self.assertEqual([row["associated"] for row in result["sources"]], [True, True, False, True, True, True])
            for args, signer in runner.sent:
                if args[0] == "staking":
                    self.assertTrue(signer.startswith("upgrade_vesting_"))
                else:
                    self.assertEqual(signer, "alice")
            self.assertEqual(sum(args[:2] == ("structs", "address-register") for args, _ in runner.sent), 5)
            ordinary = result["sources"][-1]
            self.assertEqual(ordinary["account_type"], "BaseAccount")
            self.assertEqual(ordinary["delegated_amount"], "800000")
            self.assertEqual(ordinary["unbonding_amount"], "100000")
            self.assertEqual(ordinary["bank_event_balance"], "1200017")
            self.assertEqual([row["step"] for row in result["transactions"]][-4:],
                ["ordinary_register", "ordinary_fund", "ordinary_delegate", "ordinary_undelegate"])
            self.assertEqual(runner.sent[-3][0], ("bank", "send", "structs1alice", ordinary["proof"]["address"], "2000000ualpha"))
            snapshot_calls = [args for args in runner.calls if "--height" in args]
            self.assertEqual(len(snapshot_calls), 30)
            self.assertTrue(all(args[-2:] == ("--height", "42") for args in snapshot_calls))
            self.assertEqual(result["sources"][0]["unbonding_height"], "30")
            with self.assertRaisesRegex(ValueError, "already attempted"):
                seed_vesting(runner, tool, proof)
            self.assertEqual(len(runner.sent), 18)

    def test_tool_tamper_and_private_proof_fields_refuse_before_broadcast(self):
        with tempfile.TemporaryDirectory() as directory:
            runner, tool = self.setup_runner(Path(directory))
            Path(tool["path"]).write_text("changed")
            with self.assertRaisesRegex(ValueError, "changed"):
                seed_vesting(runner, tool, proof)
            self.assertFalse((runner.directory / "vesting-seed.json").exists())
            self.assertFalse(runner.sent)
            tool = proof_tool_input(tool["path"])
            def unsafe(name, player):
                return dict(proof(name, player), private_key="must-not-be-recorded")
            with self.assertRaisesRegex(ValueError, "unexpected legacy"):
                seed_vesting(runner, tool, unsafe)
            self.assertFalse(runner.sent)
            self.assertNotIn("must-not-be-recorded", (runner.directory / "vesting-seed.json").read_text())

    def test_failed_seed_keeps_partial_evidence_and_refuses_continuation(self):
        with tempfile.TemporaryDirectory() as directory:
            runner, tool = self.setup_runner(Path(directory))
            runner.fail_delegation = True
            with self.assertRaisesRegex(RuntimeError, "delegation refusal"):
                seed_vesting(runner, tool, proof)
            evidence = json.loads((runner.directory / "vesting-seed.json").read_text())
            self.assertEqual(evidence["status"], "seeding failed; no upgrade should be proposed")
            self.assertEqual(len(evidence["transactions"]), 2)
            self.assertEqual(len(runner.sent), 3)

    def test_account_observation_refuses_wrong_kind_identity_and_original_coins(self):
        value = account("structs1vesting")
        self.assertEqual(legacy_account(value, "structs1vesting", "DelayedVestingAccount")["base_account"]["address"], "structs1vesting")
        modern = {"account": dict(value["account"]["value"], **{"@type": value["account"]["type"]})}
        legacy_account(modern, "structs1vesting", "DelayedVestingAccount")
        wrong = copy.deepcopy(value)
        wrong["account"]["value"]["base_vesting_account"]["original_vesting"][0]["amount"] = "1999999"
        for malformed in ({}, {"account": None}, account("structs1other"), account("structs1vesting", "ContinuousVestingAccount"),
                wrong, {"account": dict(modern["account"], type="ambiguous")}):
            with self.assertRaises(ValueError):
                legacy_account(malformed, "structs1vesting", "DelayedVestingAccount")

    def test_snapshot_principal_and_complete_inventory_are_required(self):
        with tempfile.TemporaryDirectory() as directory:
            runner, tool = self.setup_runner(Path(directory))
            result = seed_vesting(runner, tool, proof)
            source = result["sources"][0]
            self.assertEqual(source["bank_event_balance"], "1000017", "staking rewards must be included")
            self.assertEqual(result["sources"][1]["bank_event_balance"], "1199999", "registration sweeps vested coins")
            original = source["snapshot"]
            changes = [
                (["balances", "balances", 0, "amount"], "1000000"),
                (["balances", "pagination", "next_key"], "cursor"),
                (["delegations", "delegation_responses"], []),
                (["delegations", "delegation_responses", 0, "balance", "amount"], "899999"),
                (["delegations", "delegation_responses", 0, "delegation", "delegator_address"], "wrong"),
                (["delegations", "pagination", "next_key"], "cursor"),
                (["unbondings", "unbonding_responses"], []),
                (["unbondings", "unbonding_responses", 0, "entries", 0, "balance"], "99999"),
                (["unbondings", "unbonding_responses", 0, "entries", 0, "initial_balance"], "99999"),
                (["unbondings", "unbonding_responses", 0, "entries", 0, "creation_height"], "31"),
                (["account", "account", "value", "base_vesting_account", "delegated_vesting", 0, "amount"], "900000"),
                (["account", "account", "value", "base_vesting_account", "end_time"], "0"),
                (["association", "address"], "wrong"),
            ]
            for path, value in changes:
                snapshot = copy.deepcopy(original)
                cursor = snapshot
                for key in path[:-1]:
                    cursor = cursor[key]
                cursor[path[-1]] = value
                with self.subTest(path=path), self.assertRaises(ValueError):
                    verify_seed_snapshot(source, snapshot, result["player_id"], result["validator"], result["end_time"], 1000017)
            control = result["sources"][2]
            snapshot = copy.deepcopy(control["snapshot"])
            snapshot["delegations"] = original["delegations"]
            with self.assertRaisesRegex(ValueError, "unexpectedly has a delegation"):
                verify_seed_snapshot(control, snapshot, result["player_id"], result["validator"], result["end_time"], 2000000)
            missing = copy.deepcopy(result["transactions"])
            for transaction in missing:
                transaction["transaction"]["events"] = []
            with self.assertRaisesRegex(ValueError, "missing or exhausted"):
                source_bank_balance(missing, source["proof"]["address"])

    def test_ordinary_source_requires_base_identity_active_principal_and_pending_refund(self):
        with tempfile.TemporaryDirectory() as directory:
            runner, tool = self.setup_runner(Path(directory))
            result = seed_vesting(runner, tool, proof)
            source = result["sources"][-1]
            original = source["snapshot"]
            changes = [
                (["account", "account", "type"], "/cosmos.vesting.v1beta1.DelayedVestingAccount"),
                (["account", "account", "value", "address"], "wrong"),
                (["association", "playerId"], "1-2"),
                (["balances", "balances", 0, "amount"], "1200000"),
                (["delegations", "delegation_responses", 0, "balance", "amount"], "800000"),
                (["delegations", "pagination", "next_key"], "YQ=="),
                (["unbondings", "unbonding_responses"], []),
                (["unbondings", "unbonding_responses", 0, "entries", 0, "balance"], "99999"),
            ]
            for path, value in changes:
                snapshot = copy.deepcopy(original)
                cursor = snapshot
                for key in path[:-1]:
                    cursor = cursor[key]
                cursor[path[-1]] = value
                with self.subTest(path=path), self.assertRaises(ValueError):
                    verify_seed_snapshot(source, snapshot, result["player_id"], result["validator"], result["end_time"], 1200017)

    def test_ordinary_funding_failure_preserves_prior_vesting_evidence(self):
        with tempfile.TemporaryDirectory() as directory:
            runner, tool = self.setup_runner(Path(directory))
            original = runner.tx
            def refuse(*arguments, **kwargs):
                if arguments[:2] == ("bank", "send"):
                    raise RuntimeError("ordinary funding refused")
                return original(*arguments, **kwargs)
            runner.tx = refuse
            with self.assertRaisesRegex(RuntimeError, "ordinary funding refused"):
                seed_vesting(runner, tool, proof)
            report = json.loads((runner.directory / "vesting-seed.json").read_text())
            self.assertEqual(report["status"], "seeding failed; no upgrade should be proposed")
            self.assertEqual(len(report["transactions"]), 15)
            self.assertEqual(report["transactions"][-1]["step"], "ordinary_register")
            self.assertEqual(len(report["sources"]), 5)
            with self.assertRaisesRegex(ValueError, "already attempted"):
                seed_vesting(runner, tool, proof)

    def test_periodic_and_permanent_schedules_match_exact_native_inputs(self):
        with tempfile.TemporaryDirectory() as directory:
            runner, tool = self.setup_runner(Path(directory))
            result = seed_vesting(runner, tool, proof)
            periodic, permanent = result["sources"][3:5]
            self.assertEqual(periodic["account_type"], "PeriodicVestingAccount")
            self.assertEqual(periodic["delegated_amount"], "600000")
            self.assertEqual(permanent["account_type"], "PermanentLockedAccount")
            self.assertEqual(permanent["end_time"], "0")
            self.assertEqual(permanent["delegated_amount"], "500000")
            data = json.loads((runner.directory / "vesting-periodic-input.json").read_text())
            self.assertEqual(data, {"start_time": int(result["end_time"]) - 86400,
                "periods": [{"coins": "750000ualpha", "length_seconds": 21600},
                    {"coins": "1250000ualpha", "length_seconds": 64800}]})
            for field in ("start", "length", "amount", "order"):
                snapshot = copy.deepcopy(periodic["snapshot"])
                value = snapshot["account"]["account"]["value"]
                if field == "start":
                    value["start_time"] = str(int(value["start_time"]) + 1)
                elif field == "length":
                    value["vesting_periods"][0]["length"] = "21599"
                elif field == "amount":
                    value["vesting_periods"][0]["amount"][0]["amount"] = "749999"
                else:
                    value["vesting_periods"].reverse()
                with self.subTest(field=field), self.assertRaisesRegex(ValueError, "installments"):
                    verify_seed_snapshot(periodic, snapshot, result["player_id"], result["validator"], result["end_time"], 1400000)
            snapshot = copy.deepcopy(permanent["snapshot"])
            snapshot["account"]["account"]["value"]["base_vesting_account"]["end_time"] = result["end_time"]
            with self.assertRaisesRegex(ValueError, "end time"):
                verify_seed_snapshot(permanent, snapshot, result["player_id"], result["validator"], result["end_time"], 1500000)
            del snapshot["account"]["account"]["value"]["base_vesting_account"]["end_time"]
            verify_seed_snapshot(permanent, snapshot, result["player_id"], result["validator"], result["end_time"], 1500000)

    def test_short_fixture_schedule_and_invalid_durations(self):
        with tempfile.TemporaryDirectory() as directory:
            runner, tool = self.setup_runner(Path(directory))
            for value in (True, "1800", 1800.0, 0, 1196, 1201, 86404):
                with self.subTest(value=value), self.assertRaisesRegex(ValueError, "fixture schedule"):
                    seed_vesting(runner, tool, proof, schedule_seconds=value)
                self.assertFalse(runner.sent)
                self.assertFalse((runner.directory / "vesting-seed.json").exists())
            result = seed_vesting(runner, tool, proof, schedule_seconds=1800)
            self.assertEqual(result["schedule_seconds"], 1800)
            periodic = result["sources"][3]
            self.assertEqual(int(result["end_time"]) - int(periodic["start_time"]), 1800)
            self.assertEqual([period["length"] for period in periodic["vesting_periods"]], ["450", "1350"])

    @patch("run_upgrade_rehearsal.proof_tool_input", return_value={"path": "/proof", "sha256": "digest"})
    @patch("run_upgrade_rehearsal.seed_vesting", side_effect=RuntimeError("seed refused"))
    def test_seed_failure_prevents_governance_in_both_population_modes(self, seed, validate):
        for focused in (False, True):
            runner = rehearsal()
            runner.save, runner.preflight, runner.wait_height, runner.tx = Mock(), Mock(), Mock(), Mock()
            old, fixture = Mock(), Mock()
            fixture.wait.return_value = 0
            runner.start = Mock(side_effect=[old] if focused else [old, fixture])
            with self.assertRaisesRegex(RuntimeError, "seed refused"):
                runner.run(1, Path("/proof"), vesting_only=focused)
            runner.tx.assert_not_called()
            seed.assert_called_with(runner, validate.return_value, schedule_seconds=86400)
            self.assertEqual(runner.start.call_count, 1 if focused else 2)
            self.assertEqual(runner.manifest["population_mode"], "vesting_only" if focused else "release_fixture_and_vesting")
        runner = rehearsal()
        runner.preflight = Mock()
        with self.assertRaisesRegex(ValueError, "requires"):
            runner.run(1, vesting_only=True)
        runner.preflight.assert_not_called()
        runner.manifest["vesting_fixture_schedule_seconds"] = 1201
        with self.assertRaisesRegex(ValueError, "fixture schedule"):
            runner.run(1, Path("/proof"), vesting_only=True)
        runner.preflight.assert_not_called()


if __name__ == "__main__":
    unittest.main()
