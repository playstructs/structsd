import copy
import unittest

from verify_upgrade_bank import verify_bank


def event(kind, **fields):
    return {"type": kind, "attributes": [{"key": key, "value": value} for key, value in fields.items()]}


class BankReplayTests(unittest.TestCase):
    def fixture(self):
        large = "9007199254740993"
        before = {"initial_height": "7", "app_state": {"structs": {"addressList": [{"address": "source", "playerIndex": "1"}]},
            "bank": {"balances": [{"address": "source", "coins": [{"denom": "ualpha", "amount": "20"}, {"denom": "uguild.0-1", "amount": large}]}]}}}
        after = copy.deepcopy(before)
        after["initial_height"] = "8"
        after["app_state"]["bank"]["balances"][0]["address"] = "vault"
        # Two identical-looking transfers are two real operations, not duplicate
        # delivery. Do not deduplicate by sender/recipient/amount.
        events = []
        for _ in range(2):
            events.extend([event("coin_spent", spender="source", amount="10ualpha"),
                event("coin_received", receiver="vault", amount="10ualpha"),
                event("transfer", sender="source", recipient="vault", amount="10ualpha")])
        events.extend([event("coin_spent", spender="source", amount=large + "uguild.0-1"),
            event("coin_received", receiver="vault", amount=large + "uguild.0-1"),
            event("transfer", sender="source", recipient="vault", amount=large + "uguild.0-1")])
        return before, after, [{"height": "7", "finalize_block_events": events}], {"1-1": "vault"}

    def test_exact_all_denom_replay_without_double_counting_transfer_events(self):
        result = verify_bank(*self.fixture())
        self.assertEqual(result["balance_differences"], [])
        self.assertEqual(result["migration_differences"], [])
        self.assertEqual(result["bank_event_count"], 6)

    def test_retained_vesting_balance_is_not_required_to_move_and_cannot_be_swept(self):
        before, after, blocks, vaults = self.fixture()
        operator = "structs1operatorabcdef"
        account = {"@type": "/cosmos.vesting.v1beta1.DelayedVestingAccount",
            "base_vesting_account": {"base_account": {"address": operator}}}
        for document in (before, after):
            document["app_state"]["auth"] = {"accounts": [copy.deepcopy(account)]}
            document["app_state"]["structs"]["addressList"].append({"address": operator, "playerIndex": "1"})
            document["app_state"]["bank"]["balances"].append({"address": operator, "coins": [{"denom": "ualpha", "amount": "7"}]})
        before["app_state"]["staking"] = {"delegations": [], "unbonding_delegations": [],
            "redelegations": [{"delegator_address": operator, "validator_dst_address": "structsvaloper1operatorghjkmn"}]}
        result = verify_bank(before, after, blocks, vaults)
        self.assertEqual([], result["balance_differences"])
        self.assertEqual([], result["migration_differences"])
        self.assertEqual([operator], result["retained_vesting_addresses"])
        # Even a perfectly event-reconciled sweep violates the exception.
        after["app_state"]["bank"]["balances"][-1]["coins"][0]["amount"] = "0"
        after["app_state"]["bank"]["balances"][0]["coins"][0]["amount"] = "27"
        blocks[0]["finalize_block_events"].extend([
            event("coin_spent", spender=operator, amount="7ualpha"),
            event("coin_received", receiver="vault", amount="7ualpha"),
            event("transfer", sender=operator, recipient="vault", amount="7ualpha")])
        result = verify_bank(before, after, blocks, vaults)
        self.assertEqual([], result["balance_differences"])
        self.assertTrue(result["migration_differences"])

    def test_detects_missing_bank_event_or_wrong_vault_routing(self):
        before, after, blocks, vaults = self.fixture()
        blocks[0]["finalize_block_events"].pop(1)
        self.assertTrue(verify_bank(before, after, blocks, vaults)["balance_differences"])
        before, after, blocks, vaults = self.fixture()
        blocks[0]["finalize_block_events"][2] = event("transfer", sender="source", recipient="wrong", amount="10ualpha")
        self.assertTrue(verify_bank(before, after, blocks, vaults)["migration_differences"])
        before, after, blocks, vaults = self.fixture()
        after["app_state"]["bank"]["balances"] = copy.deepcopy(before["app_state"]["bank"]["balances"])
        blocks[0]["finalize_block_events"] = []
        result = verify_bank(before, after, blocks, vaults)
        self.assertEqual(result["balance_differences"], [])
        self.assertTrue(result["migration_differences"], "unchanged bank state must not prove migration")

    def test_mint_burn_and_failed_transaction_ante_events(self):
        before, after, blocks, vaults = self.fixture()
        before["app_state"]["bank"]["balances"].append({"address": "outside", "coins": [{"denom": "other", "amount": "3"}]})
        after["app_state"]["bank"]["balances"].extend([
            {"address": "outside", "coins": [{"denom": "other", "amount": "1"}]},
            {"address": "fees", "coins": [{"denom": "other", "amount": "1"}]},
            {"address": "mint", "coins": [{"denom": "other", "amount": "2"}]},
        ])
        blocks[0]["finalize_block_events"].extend([
            event("coin_received", receiver="mint", amount="2other"), event("coinbase", minter="mint", amount="2other"),
            event("coin_spent", spender="outside", amount="1other"), event("burn", burner="outside", amount="1other"),
        ])
        blocks[0]["txs_results"] = [{"code": 5, "events": [
            event("coin_spent", spender="outside", amount="1other"), event("coin_received", receiver="fees", amount="1other"),
        ]}]
        result = verify_bank(before, after, blocks, vaults)
        self.assertEqual(result["balance_differences"], [])
        self.assertEqual(result["migration_differences"], [])

    def test_unbonding_refunds_need_bank_events_not_transfer_or_completion(self):
        for held_release in (False, True):
            for fault in ("none", "missing_debit", "missing_credit", "duplicate_credit"):
                with self.subTest(held_release=held_release, fault=fault):
                    before, after, blocks, vaults = self.fixture()
                    before["app_state"]["bank"]["balances"].append({
                        "address": "not_bonded_pool", "coins": [{"denom": "ualpha", "amount": "100"}],
                    })
                    # The migration moved 20 ualpha; an independent staking
                    # refund adds 100. Both events describe one refund, and no
                    # transfer event exists on the native undelegation path.
                    after["app_state"]["bank"]["balances"][0]["coins"][0]["amount"] = "120"
                    debit = event("coin_spent", spender="not_bonded_pool", amount="100ualpha")
                    credit = event("coin_received", receiver="vault", amount="100ualpha")
                    events = blocks[0]["finalize_block_events"]
                    if fault != "missing_debit":
                        events.append(debit)
                    if fault != "missing_credit":
                        events.append(credit)
                    if fault == "duplicate_credit":
                        events.append(copy.deepcopy(credit))
                    if not held_release:
                        # Ordinary maturity also reports completion. Releasing
                        # the last hold through the SDK pays without this event.
                        events.append(event("complete_unbonding", delegator="vault",
                                            validator="validator", amount="100ualpha"))
                    result = verify_bank(before, after, blocks, vaults)
                    self.assertEqual(result["migration_differences"], [])
                    if fault == "none":
                        self.assertEqual(result["balance_differences"], [])
                        self.assertEqual(result["bank_event_count"], 8)
                    else:
                        self.assertTrue(result["balance_differences"])

    def test_rejects_incomplete_or_duplicate_heights_and_duplicate_attributes(self):
        before, after, blocks, vaults = self.fixture()
        for invalid in ([], blocks * 2, [{**blocks[0], "height": "8"}]):
            with self.assertRaisesRegex(ValueError, "boundary height"):
                verify_bank(before, after, invalid, vaults)
        blocks[0]["finalize_block_events"][0]["attributes"].append({"key": "amount", "value": "1ualpha"})
        with self.assertRaisesRegex(ValueError, "duplicate bank event attribute"):
            verify_bank(before, after, blocks, vaults)


if __name__ == "__main__":
    unittest.main()
