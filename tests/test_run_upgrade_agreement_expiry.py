from copy import deepcopy
import io
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import MagicMock, patch

import run_upgrade_agreement_expiry as driver
from test_verify_upgrade_agreement_expiry import expiry_fixture
from upgrade_checkpoint import file_hash


def seed():
    return {"status": "three live legacy agreements confirmed; upgrade verification pending",
            "agreements": [{"id": "11-" + str(i), "endBlock": str(end)} for i, end in enumerate((30, 35, 40), 1)]}


class ExpiryDriverTests(unittest.TestCase):
    def test_window_uses_post_transfer_consecutive_exports(self):
        self.assertEqual(driver.expiry_window(seed(), {"label": "initial-sent", "height": "8"},
                                             {"label": "timed-unlocks-complete", "height": "45"}),
                         {"before": 9, "migrated": 10, "after": 40})

    def test_window_refuses_ambiguous_or_overlapping_evidence(self):
        for fault in ("unconfirmed", "duplicate", "early-expiry", "late-expiry", "label", "height"):
            value = seed()
            initial = {"label": "initial-sent", "height": "8"}
            final = {"label": "timed-unlocks-complete", "height": "45"}
            if fault == "unconfirmed": value["status"] = "started"
            elif fault == "duplicate": value["agreements"][1]["id"] = "11-1"
            elif fault == "early-expiry": value["agreements"][0]["endBlock"] = "11"
            elif fault == "late-expiry": final["height"] = "40"
            elif fault == "label": initial["label"] = "initial-refused"
            else: initial["height"] = "08"
            with self.subTest(fault=fault), self.assertRaises(ValueError):
                driver.expiry_window(value, initial, final)

    def test_gap_cannot_hide_service_changes_or_earlier_payments(self):
        _, original, _, _, _, providers = expiry_fixture()
        seeded = {"agreements": original["app_state"]["structs"]["agreementList"]}
        driver.unchanged_service_baseline(original, deepcopy(original), seeded, providers)
        for fault in ("terms", "checkpoint", "load", "funds", "identity"):
            current = deepcopy(original)
            state = current["app_state"]["structs"]
            chosen_seed = deepcopy(seeded)
            if fault == "terms": state["agreementList"][0]["capacity"] = "99"
            elif fault == "checkpoint": state["gridList"].append({"attributeId": "14-10-1", "value": "999"})
            elif fault == "load": state["gridList"].append({"attributeId": "3-10-1", "value": "999"})
            elif fault == "funds": current["app_state"]["bank"]["balances"][0]["coins"][0]["amount"] = "999"
            else: chosen_seed["agreements"] = []
            with self.subTest(fault=fault), self.assertRaises(ValueError):
                driver.unchanged_service_baseline(original, current, chosen_seed, providers)

    def test_driver_stops_before_exports_and_preserves_unfiltered_rpc(self):
        for unexpected_transaction in (False, True):
            with self.subTest(transaction=unexpected_transaction), tempfile.TemporaryDirectory() as temporary:
                root = Path(temporary)
                continuation = root / "vesting-continuation"
                continuation.mkdir()
                binary = root / "structsd"
                binary.write_bytes(b"frozen candidate")
                before, migrated, after, _, _, providers = expiry_fixture()
                third = deepcopy(migrated["app_state"]["structs"]["agreementList"][0])
                third.update(id="11-3", endBlock="35")
                for document in (before, migrated):
                    document["app_state"]["structs"]["agreementList"].append(deepcopy(third))
                manifest = {"status": driver.PASSED, "chain_home": str(root / "chain"),
                            "chain_id": "private", "new_binary_sha256": file_hash(binary)}
                documents = {"manifest.json": manifest, "before.json": before, "after.json": migrated,
                             "upgrade-block-results.json": [], "vault-map.json": {}, "provider-accounts.json": providers,
                             "agreement-seed.json": seed(), "vesting-continuation/result.json": {"status": "native continuation checks passed"},
                             "vesting-continuation/03-initial-sent-queries.json": {"label": "initial-sent", "height": "8"},
                             "vesting-continuation/08-timed-unlocks-complete-queries.json": {"label": "timed-unlocks-complete", "height": "45"}}
                for name, value in documents.items(): (root / name).write_text(json.dumps(value))
                proof = {"differences": [], "input_sha256": {name: file_hash(root / name) for name in (
                    "before.json", "after.json", "upgrade-block-results.json", "vault-map.json", "provider-accounts.json")}}
                (root / "agreement-verification.json").write_text(json.dumps(proof))
                runner = MagicMock(directory=root, new=binary, home=root / "chain", manifest=manifest)
                runner.cli.return_value = {"node_info": {"network": "private"}}
                running = [True]
                runner.stop.side_effect = lambda node: running.__setitem__(0, False)

                def rpc(url, timeout):
                    height = url.split("=")[-1]
                    return io.BytesIO(json.dumps({"result": {"height": height, "finalize_block_events": [],
                        "txs_results": [{"code": 9}] if unexpected_transaction else None}}).encode())

                def export(binary, home, target, *, height):
                    self.assertFalse(running[0], "historical export must never race the node")
                    document = {9: before, 10: migrated, 40: after}[height]
                    target.write_text(json.dumps(document))
                    return document

                with patch.object(driver, "Rehearsal", return_value=runner), patch.object(driver, "assert_no_nodes"), \
                     patch.object(driver, "urlopen", side_effect=rpc), patch.object(driver, "export", side_effect=export) as exports, \
                     patch.object(driver, "verify_legacy_expiry", return_value={"differences": []}) as verify, patch("builtins.print"):
                    if unexpected_transaction:
                        with self.assertRaisesRegex(ValueError, "contains transactions"): driver.run(root, binary)
                        exports.assert_not_called()
                        verify.assert_not_called()
                        self.assertEqual(driver.read(root / "agreement-expiry/block-10.json")["result"]["txs_results"], [{"code": 9}])
                        self.assertTrue((root / "agreement-expiry/failure.json").exists())
                    else:
                        self.assertEqual(driver.run(root, binary)["differences"], [])
                        self.assertEqual(exports.call_count, 3)
                        args = verify.call_args.args
                        self.assertEqual([row["height"] for row in args[3]], ["10"])
                        self.assertEqual([row["height"] for row in args[4]], [str(h) for h in range(11, 41)])
                        self.assertIn("txs_results", args[4][0])
                    runner.cleanup.assert_called_once()


if __name__ == "__main__":
    unittest.main()
