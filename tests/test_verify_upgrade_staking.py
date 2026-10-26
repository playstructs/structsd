import copy
import unittest

from verify_upgrade_staking import shares, verify
from sum_staking_shares import sum_shares


class StakingRelocationTest(unittest.TestCase):
    def fixture(self):
        sources = ["structs1sourceaabcdef", "structs1sourcebabcdef"]
        vault = "structs1vaultabcdef"
        validator = "structsvaloper1validatorabcdef"
        account = lambda address: {"@type": "/cosmos.auth.v1beta1.BaseAccount", "address": address}
        entries = [{"creation_height": "7", "completion_time": "2026-10-01T00:00:00Z",
                    "initial_balance": "19", "balance": "17", "unbonding_id": str(i), "unbonding_on_hold_ref_count": "2"} for i in (9, 10)]
        before = {"structs": {"playerList": [{"id": "1-1"}], "addressList": [{"address": s, "playerIndex": "1"} for s in sources]},
                  "auth": {"accounts": [account(s) for s in [*sources, vault]]}, "bank": {"balances": []},
                  "staking": {"delegations": [{"delegator_address": s, "validator_address": validator, "shares": n} for s, n in zip([*sources, vault], ["9007199254740993.000000000000000001", "0.999999999999999999", "2.0"])],
                              "unbonding_delegations": [{"delegator_address": s, "validator_address": validator, "entries": [e]} for s, e in zip(sources, entries)],
                              "redelegations": [{"delegator_address": sources[0], "validator_src_address": validator, "validator_dst_address": "structsvaloper1otherabcdef", "entries": [{"creation_height": "8", "completion_time": "2026-10-02T00:00:00Z", "initial_balance": "13", "shares_dst": "12.125", "unbonding_id": "11", "unbonding_on_hold_ref_count": "1"}]}]}}
        after = copy.deepcopy(before)
        after["structs"]["vaultOwnerList"] = ["1-1"]
        after["staking"] = copy.deepcopy(before["staking"])
        return before, after, {"1-1": vault}

    def test_retained_records_with_existing_vault_stake(self):
        self.assertEqual([], verify(*self.fixture())["differences"])

    def test_detects_lost_fraction_hold_and_source_not_removed(self):
        for change in ("fraction", "hold", "source", "redelegation", "duplicate_entry"):
            with self.subTest(change=change):
                before, after, vaults = self.fixture()
                stake = after["staking"]
                if change == "fraction":
                    stake["delegations"][0]["shares"] = "9007199254740995.999999999999999999"
                elif change == "hold":
                    stake["unbonding_delegations"][0]["entries"][0]["unbonding_on_hold_ref_count"] = "0"
                elif change == "source":
                    stake["delegations"][0]["delegator_address"] = "structs1wrongabcdef"
                elif change == "redelegation":
                    stake["redelegations"][0]["entries"][0]["shares_dst"] = "12.124"
                else:
                    stake["unbonding_delegations"][0]["entries"] *= 2
                self.assertTrue(verify(before, after, vaults)["differences"])

    def test_rejects_partial_exports_missing_mapping_and_duplicate_records(self):
        before, after, vaults = self.fixture()
        with self.assertRaisesRegex(ValueError, "missing vault"):
            verify(before, after, {})
        after["staking"]["delegations"] *= 2
        with self.assertRaisesRegex(ValueError, "duplicate delegations"):
            verify(before, after, vaults)
        del after["staking"]["redelegations"]
        with self.assertRaisesRegex(ValueError, "export list"):
            verify(before, after, vaults)

    def test_rejects_relocation_of_any_staking_record(self):
        for field in ("delegations", "unbonding_delegations", "redelegations"):
            with self.subTest(field=field):
                before, after, vaults = self.fixture()
                after["staking"][field][0]["delegator_address"] = "structs1wrongabcdef"
                self.assertTrue(verify(before, after, vaults)["differences"])

    def test_vesting_without_self_stake_remains_native(self):
        before, after, vaults = self.fixture()
        source = before["structs"]["addressList"][0]["address"]
        for state in (before, after):
            state["auth"]["accounts"][0] = {"@type": "/cosmos.vesting.v1beta1.DelayedVestingAccount",
                "base_vesting_account": {"base_account": {"address": source}, "original_vesting": []}}
        result = verify(before, after, vaults)
        self.assertEqual([], result["differences"])
        self.assertEqual([source], result["retained_vesting_addresses"])

    def test_live_assertion_share_sum_is_exact(self):
        self.assertEqual("9007199254740994.000000000000000000", sum_shares(["9007199254740993.000000000000000001", "0.999999999999999999"]))
        for values in ([], ["1e18"], ["-1"]):
            with self.assertRaises(ValueError):
                sum_shares(values)

    def test_shares_never_use_float_or_decimal_context_rounding(self):
        self.assertEqual(10**50 + 1, shares(str(10**32) + ".000000000000000001"))
        for invalid in (1.0, "1e10", "-1", "0.0000000000000000001"):
            with self.assertRaises(ValueError):
                shares(invalid)


if __name__ == "__main__":
    unittest.main()
