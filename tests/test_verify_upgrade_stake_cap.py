import copy
import json
import unittest
from verify_upgrade_stake_cap import EVENT, verify_stake_cap


def fixture(active=True):
    before = {"app_state": {"structs": {}}}
    after = {"app_state": {"structs": {"reactorStakeCapActive": active,
        "params": {"reactorStakeCapBps": "3000", "reactorStakeCapMinValidators": "10"}},
        "staking": {"validators": [{"status": "BOND_STATUS_BONDED"}] * (10 if active else 1)}}}
    event = {"type": EVENT, "attributes": [{"key": key, "value": json.dumps(value)} for key, value in
        (("active", active), ("capBps", "3000"), ("minValidators", "10"))]}
    return before, after, [{"height": "10", "finalize_block_events": [event]}]


class StakeCapUpgradeTests(unittest.TestCase):
    def test_active_and_bootstrap_snapshot(self):
        for active in (False, True):
            self.assertEqual([], verify_stake_cap(*fixture(active), 10)["differences"])

    def test_missing_wrong_height_and_duplicate_snapshot(self):
        for edit in (lambda e: [], lambda e: e * 2):
            old, new, blocks = fixture()
            blocks[0]["finalize_block_events"] = edit(blocks[0]["finalize_block_events"])
            self.assertTrue(verify_stake_cap(old, new, blocks, 10)["differences"])
        self.assertTrue(verify_stake_cap(*fixture(), 11)["differences"])

    def test_sticky_activation_and_threshold(self):
        old, new, blocks = fixture(False)
        old["app_state"]["structs"]["reactorStakeCapActive"] = True
        self.assertTrue(verify_stake_cap(old, new, blocks, 10)["differences"])
        old, new, blocks = fixture(False)
        new["app_state"]["staking"]["validators"] *= 10
        self.assertTrue(verify_stake_cap(old, new, blocks, 10)["differences"])

    def test_threshold_crossing_after_upgrade_snapshot_in_same_block(self):
        old, new, blocks = fixture(True)
        initial = copy.deepcopy(blocks[0]["finalize_block_events"][0])
        initial["attributes"][0]["value"] = "false"
        blocks[0]["finalize_block_events"].insert(0, initial)
        self.assertEqual([], verify_stake_cap(old, new, blocks, 10)["differences"])

    def test_malformed_event_and_export(self):
        for malformed in (1, "true", None):
            old, new, blocks = fixture()
            blocks[0]["finalize_block_events"][0]["attributes"][0]["value"] = json.dumps(malformed)
            with self.assertRaises(ValueError): verify_stake_cap(old, new, blocks, 10)
        old, new, blocks = fixture()
        event = blocks[0]["finalize_block_events"][0]
        event["attributes"].append(copy.deepcopy(event["attributes"][0]))
        with self.assertRaises(ValueError): verify_stake_cap(old, new, blocks, 10)

if __name__ == "__main__": unittest.main()
