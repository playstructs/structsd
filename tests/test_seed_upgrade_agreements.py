import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import Mock, patch

from run_upgrade_rehearsal import Rehearsal
from seed_upgrade_agreements import created_object, seed_agreements


def receipt(kind, row):
    return {"events": [{"type": "structs.structs.Event" + kind,
                        "attributes": [{"key": kind.lower(), "value": json.dumps(row)}]}]}


class LegacyAgreementSeedTests(unittest.TestCase):
    def test_creation_identity_comes_from_typed_receipt(self):
        value = receipt("Agreement", {"id": "11-7", "capacity": "2"})
        value["events"].insert(0, {"type": "transfer", "attributes": []})
        self.assertEqual(created_object(value, "Agreement", "agreement")["id"], "11-7")
        value["events"].append(receipt("Agreement", {"id": "11-7", "capacity": "3"})["events"][0])
        self.assertEqual(created_object(value, "Agreement", "agreement")["capacity"], "3")

    def test_missing_ambiguous_or_duplicate_attribute_refuses(self):
        value = receipt("Agreement", {"id": "11-7"})
        other = receipt("Agreement", {"id": "11-8"})
        duplicate = receipt("Agreement", {"id": "11-7"})
        duplicate["events"][0]["attributes"] *= 2
        for transaction in ({"events": []}, {"events": value["events"] + other["events"]}, duplicate):
            with self.subTest(transaction=transaction), self.assertRaises(ValueError):
                created_object(transaction, "Agreement", "agreement")

    def test_seed_requires_three_distinct_contracts_and_exact_terms(self):
        for scenario in ("valid", "wrong_provider", "wrong_duration", "duplicate_id"):
            with self.subTest(scenario=scenario), tempfile.TemporaryDirectory() as temporary:
                runner = Mock(directory=Path(temporary), old=Path("/fake/old"))
                def query(*args):
                    if "address" in args: return {"playerId": "1-1"}
                    if "player" in args: return {"Player": {"id": "1-1", "substationId": "4-99"}}
                    if "substation" in args: return {"Substation": {"id": "4-99", "owner": "1-1"}}
                    return {"address": "key-" + args[3]}
                runner.cli.side_effect = query
                contracts = [{"id": "11-" + str(i), "providerId": "10-1", "capacity": str(capacity),
                              "startBlock": str(100 + i), "endBlock": str(100 + i + duration)}
                             for i, (capacity, duration) in enumerate(((2, 800), (1, 810), (3, 820)), 1)]
                if scenario == "wrong_provider":
                    contracts[1]["providerId"] = "10-2"
                elif scenario == "wrong_duration":
                    contracts[1]["endBlock"] = "9000"
                elif scenario == "duplicate_id":
                    contracts[2]["id"] = contracts[0]["id"]
                runner.tx.side_effect = [receipt("Allocation", {"id": "6-1"}), receipt("Substation", {"id": "4-1"}),
                                         receipt("Provider", {"id": "10-1"}), {}, {},
                                         *[receipt("Agreement", row) for row in contracts]]
                if scenario == "valid":
                    seed_agreements(runner)
                    report = runner.save.call_args.args[1]
                    self.assertEqual(len(report["agreements"]), 3)
                    self.assertEqual(len(report["transactions"]), 8)
                    self.assertIn("confirmed", report["status"])
                    self.assertEqual(report["upstream_substation"]["id"], "4-99")
                    self.assertEqual(runner.tx.call_args_list[0].args[:4], ("structs", "allocation-create", "4-99", "1000"))
                else:
                    with self.assertRaises(ValueError):
                        seed_agreements(runner)
                    self.assertNotIn("confirmed", runner.save.call_args.args[1]["status"])

    def test_missing_or_foreign_source_refuses_before_transactions(self):
        for station, upstream in ((None, {}), ("4-99", {"id": "4-99", "owner": "1-2"}),
                                  ("4-99", {"id": "4-100", "owner": "1-1"})):
            with self.subTest(station=station, upstream=upstream), tempfile.TemporaryDirectory() as temporary:
                runner = Mock(directory=Path(temporary), old=Path("/fake/old"))
                runner.cli.side_effect = [{"address": "alice"}, {"playerId": "1-1"},
                                          {"Player": {"substationId": station}}, {"Substation": upstream}]
                with self.assertRaises(ValueError): seed_agreements(runner)
                runner.tx.assert_not_called()

    @patch("run_upgrade_rehearsal.seed_agreements", side_effect=ValueError("contract seed refused"))
    def test_failed_seed_prevents_governance(self, seed):
        runner = object.__new__(Rehearsal)
        runner.old, runner.home, runner.directory = Path("/old"), Path("/home"), Path("/fixture")
        runner.preflight, runner.wait_height, runner.tx = Mock(), Mock(), Mock()
        fixture = Mock()
        fixture.wait.return_value = 0
        runner.start = Mock(side_effect=[Mock(), fixture])
        with self.assertRaisesRegex(ValueError, "contract seed refused"):
            runner.run(1, live_agreements=True)
        seed.assert_called_once_with(runner)
        runner.tx.assert_not_called()


if __name__ == "__main__":
    unittest.main()
