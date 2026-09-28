import unittest

from upgrade_inventory import inventory


class InventoryTest(unittest.TestCase):
    def fixture(self):
        # Payload matching assumes a validated export; these are synthetic
        # address labels with differing account/operator prefixes/checksums.
        source = "structs1payloadabcdef"
        validator = "structsvaloper1payloadghjkmn"
        state = {
            "structs": {"playerList": [{"id": "1-1"}], "addressList": [{"address": source, "playerIndex": "1"}]},
            "auth": {"accounts": [{"@type": "/cosmos.vesting.v1beta1.PeriodicVestingAccount",
                "base_vesting_account": {"base_account": {"address": source}, "original_vesting": [{"denom": "ualpha", "amount": "10"}]},
                "vesting_periods": [{"length": "10"}, {"length": "20"}]}]},
            "bank": {"balances": [{"address": source, "coins": [{"denom": "ualpha", "amount": "9007199254740993"}]},
                {"address": "unassociated", "coins": [{"denom": "other", "amount": "7"}]}]},
            "staking": {"delegations": [{"delegator_address": source, "validator_address": validator, "shares": "12.3"}],
                "unbonding_delegations": [{"delegator_address": source, "entries": [{}, {}]}], "redelegations": []},
        }
        return state

    def test_exact_amounts_vesting_and_self_delegation(self):
        result = inventory({"app_state": self.fixture()})
        owner = result["owners"]["1-1"]
        self.assertEqual("9007199254740993", owner["bank_balances"]["ualpha"])
        self.assertEqual(1, owner["vesting_accounts"])
        self.assertEqual(2, owner["vesting_periods"])
        self.assertEqual(2, owner["unbonding_delegations_entries"])
        self.assertEqual(1, len(result["associated_self_delegations"]))
        self.assertEqual({"other": "7"}, result["unassociated_bank_balances"])
        self.assertEqual([], result["problems"])

    def test_missing_modules_and_orphan_indices_fail(self):
        with self.assertRaisesRegex(ValueError, "missing module"):
            inventory({})
        state = self.fixture()
        state["structs"]["playerList"] = []
        with self.assertRaisesRegex(ValueError, "missing player"):
            inventory(state)

    def test_missing_auth_is_reported_not_silently_omitted(self):
        state = self.fixture()
        state["auth"]["accounts"] = []
        self.assertEqual(1, len(inventory(state)["problems"]))

    def test_partial_export_cannot_report_empty_success(self):
        state = self.fixture()
        del state["structs"]["addressList"]
        with self.assertRaisesRegex(ValueError, "structs.addressList"):
            inventory(state)
        state = self.fixture()
        state["staking"]["unbonding_delegations"] = {}
        with self.assertRaisesRegex(ValueError, "staking.unbonding_delegations"):
            inventory(state)


if __name__ == "__main__":
    unittest.main()
