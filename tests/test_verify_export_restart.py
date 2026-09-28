import copy
import unittest

from verify_export_restart import verify_restart


def event(kind, **attrs):
    return {"type": kind, "attributes": [{"key": k, "value": v} for k, v in attrs.items()]}


def fixture(zero=False):
    def coin(n): return {"denom": "ibc/ASSET", "amount": str(n)}
    before = {"initial_height": "0" if zero else "41", "chain_id": "restart-test", "app_state": {
        "bank": {"supply": [coin(100)], "balances": [
            {"address": "pool", "coins": [coin(100)]}, {"address": "vault", "coins": []}]},
        "auth": {"accounts": [{"@type": "/cosmos.auth.v1beta1.BaseAccount", "address": "vault",
                               "account_number": "9007199254740993", "sequence": "2", "pub_key": None}]},
        "structs": {"playerList": [{"id": "1-1", "index": "1", "primaryAddress": "signer", "guildId": "0-1", "guildRank": "3"}],
                    "guildList": [{"id": "0-1", "owner": "1-1", "rankPurchaseBasePrice": "9"}],
                    "vaultOwnerList": ["1-1", "0-1"], "addressList": [{"address": "signer", "playerIndex": "1"}],
                    "addressNonceList": [{"address": "signer", "nonce": "7"}],
                    "reactorStakeCapActive": True, "params": {"reactorStakeCapBps": "3000"}, "stakingUnbondingId": "9007199254740993"}}}
    after = copy.deepcopy(before)
    after["initial_height"] = "2" if zero else "42"
    after["app_state"]["bank"]["balances"] = [{"address": "pool", "coins": [coin(82)]}, {"address": "vault", "coins": [coin(18)]}]
    blocks = [{"height": "1" if zero else "41", "txs_results": [], "finalize_block_events": [
        event("coin_spent", spender="pool", amount="18ibc/ASSET"), event("coin_received", receiver="vault", amount="18ibc/ASSET")]}]
    return before, after, blocks


class ExportRestartTests(unittest.TestCase):
    def test_held_refund_preserved_and_zero_height(self):
        for zero in (False, True):
            with self.subTest(zero=zero):
                result = verify_restart(*fixture(zero))
                self.assertEqual([], result["differences"])
                self.assertEqual(2, result["bank_events"])

    def test_overlapping_transfer_summary_is_not_another_payment(self):
        before, after, blocks = fixture()
        blocks[0]["finalize_block_events"].append(event("transfer", sender="pool", recipient="vault", amount="18ibc/ASSET"))
        self.assertEqual([], verify_restart(before, after, blocks)["differences"])

    def test_missing_or_duplicate_bank_event_is_detected(self):
        for duplicate in (False, True):
            before, after, blocks = fixture()
            events = blocks[0]["finalize_block_events"]
            if duplicate: events.append(copy.deepcopy(events[-1]))
            else: events.pop()
            self.assertTrue(verify_restart(before, after, blocks)["differences"])

    def test_supply_loss_even_when_balance_events_match_is_detected(self):
        before, after, blocks = fixture()
        after["app_state"]["bank"]["balances"][0]["coins"][0]["amount"] = "80"
        after["app_state"]["bank"]["supply"][0]["amount"] = "98"
        blocks[0]["finalize_block_events"].append(event("coin_spent", spender="pool", amount="2ibc/ASSET"))
        self.assertIn("supply differs from mint/burn events: ibc/ASSET", verify_restart(before, after, blocks)["differences"])
        blocks[0]["finalize_block_events"].append(event("burn", burner="pool", amount="2ibc/ASSET"))
        self.assertEqual([], verify_restart(before, after, blocks)["differences"])

    def test_identity_rank_owner_and_counter_corruption_are_detected(self):
        for fault in ("identity", "rank", "guild", "vault", "nonce", "counter", "activation", "params"):
            with self.subTest(fault=fault):
                before, after, blocks = fixture()
                state = after["app_state"]["structs"]
                if fault == "identity": after["app_state"]["auth"]["accounts"][0]["account_number"] = "9007199254740992"
                elif fault == "rank": state["playerList"][0]["guildRank"] = "5"
                elif fault == "guild": state["guildList"][0]["owner"] = "1-2"
                elif fault == "vault": state["vaultOwnerList"].pop()
                elif fault == "nonce": state["addressNonceList"][0]["nonce"] = "0"
                elif fault == "counter": state["stakingUnbondingId"] = "9007199254740992"
                elif fault == "activation": state["reactorStakeCapActive"] = False
                elif fault == "params": state["params"]["reactorStakeCapBps"] = "4000"
                self.assertTrue(verify_restart(before, after, blocks)["differences"])

    def test_missing_duplicate_and_wrong_height_windows_are_rejected(self):
        for mutation in (lambda rows: [], lambda rows: rows * 2, lambda rows: [{**rows[0], "height": "40"}]):
            before, after, blocks = fixture()
            with self.assertRaises(ValueError): verify_restart(before, after, mutation(blocks))

    def test_transactions_and_missing_event_envelopes_are_rejected(self):
        for fault in ("transactions", "envelope"):
            before, after, blocks = fixture()
            if fault == "transactions": blocks[0]["txs_results"] = [{"code": 0}]
            else: del blocks[0]["finalize_block_events"]
            with self.assertRaises(ValueError): verify_restart(before, after, blocks)


if __name__ == "__main__":
    unittest.main()
