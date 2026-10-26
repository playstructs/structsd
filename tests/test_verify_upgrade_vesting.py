import copy
import json
import unittest

from verify_upgrade_vesting import BASE, VAULT, source_address_bytes, verify_vesting


SOURCE = "structs1xxshfg0neejpykgm4x9hl2pnl8p57kvxghj0u4"


class VestingReplayTests(unittest.TestCase):
    def fixture(self):
        identity = {"address": SOURCE, "pub_key": None, "account_number": "9007199254740993", "sequence": "2"}
        source = {"@type": "/cosmos.vesting.v1beta1.DelayedVestingAccount",
            "base_vesting_account": {"base_account": identity, "end_time": "100", "original_vesting": [{"denom": "ualpha", "amount": "10"}]}}
        vault = {"base_account": {**identity, "address": "vault", "sequence": "0"}, "owner_id": "1-1",
            "schedules": [{"source_address": "MaF0ofPOZBJZG6mLf6gz+cNPWYY=", "kind": 2,
                "start_time": "0", "end_time": "100", "original_vesting": [{"denom": "ualpha", "amount": "10"}],
                "delegated_free": [], "delegated_vesting": [], "periods": []}], "liquid_delegated": []}
        before = {"initial_height": "7", "app_state": {"auth": {"accounts": [source]},
            "staking": {"params": {"bond_denom": "ualpha"}, "validators": [], "delegations": [], "unbonding_delegations": [], "redelegations": []},
            "structs": {"addressList": [{"address": SOURCE, "playerIndex": "1"}]}}}
        after = copy.deepcopy(before)
        after["initial_height"] = "8"
        after["app_state"]["auth"]["accounts"] = [{"@type": BASE, **identity}, {"@type": VAULT, **copy.deepcopy(vault)}]
        event = {"type": "structs.structs.EventVaultVestingMigration", "attributes": [
            {"key": "vault", "value": json.dumps(vault)}, {"key": "source_addresses", "value": json.dumps([SOURCE])}]}
        after["app_state"]["auth"]["accounts"] = [copy.deepcopy(source), {"@type": BASE, "address": "vault", "account_number": "9", "sequence": "0"}]
        return before, after, [{"height": "7", "finalize_block_events": []}], {"1-1": "vault"}

    def test_all_native_vesting_types_remain_exactly_unchanged(self):
        for name in ("ContinuousVestingAccount", "DelayedVestingAccount", "PeriodicVestingAccount", "PermanentLockedAccount"):
            before, after, blocks, vaults = self.fixture()
            for document in (before, after):
                document["app_state"]["auth"]["accounts"][0]["@type"] = "/cosmos.vesting.v1beta1." + name
            result = verify_vesting(before, after, blocks, vaults)
            self.assertEqual([], result["differences"])
            self.assertEqual(0, result["migrated_source_count"])
            self.assertEqual([SOURCE], result["retained_vesting_addresses"])

    def test_counter_schedule_identity_and_account_removal_detected(self):
        for change in ("counter", "schedule", "identity", "missing", "flattened"):
            before, after, blocks, vaults = self.fixture()
            account = after["app_state"]["auth"]["accounts"][0]
            base = account["base_vesting_account"]
            if change == "counter": base["delegated_vesting"] = [{"denom": "ualpha", "amount": "99"}]
            elif change == "schedule": base["end_time"] = "99"
            elif change == "identity": base["base_account"]["sequence"] = "99"
            elif change == "missing": after["app_state"]["auth"]["accounts"].pop(0)
            else: after["app_state"]["auth"]["accounts"][0] = {"@type": BASE, **base["base_account"]}
            self.assertTrue(verify_vesting(before, after, blocks, vaults)["differences"], change)

    def test_relocation_event_and_partial_window_rejected(self):
        before, after, blocks, vaults = self.fixture()
        with self.assertRaisesRegex(ValueError, "boundary"):
            verify_vesting(before, after, [], vaults)
        blocks[0]["finalize_block_events"] = [{"type": "structs.structs.EventVaultVestingMigration"}]
        with self.assertRaisesRegex(ValueError, "unexpected"):
            verify_vesting(before, after, blocks, vaults)

    def test_unassociated_vesting_is_also_preserved(self):
        before, after, blocks, vaults = self.fixture()
        before["app_state"]["structs"]["addressList"] = []
        after["app_state"]["auth"]["accounts"][0]["base_vesting_account"]["end_time"] = "99"
        self.assertTrue(verify_vesting(before, after, blocks, vaults)["differences"])

    def test_source_decoder_checks_native_vector_and_malformed_addresses(self):
        self.assertEqual(source_address_bytes(SOURCE).hex(), "31a174a1f3ce6412591ba98b7fa833f9c34f5986")
        for address in (SOURCE.upper(), SOURCE[:-1] + "q", SOURCE + "q", "source", "structs1!", None):
            with self.subTest(address=address), self.assertRaises(ValueError):
                source_address_bytes(address)


if __name__ == "__main__":
    unittest.main()
