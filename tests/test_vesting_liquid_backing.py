"""Ordinary principal must not become phantom free vesting counters."""
import copy
import unittest

from vesting_liquid_backing import BASE, PRECISION, original_liquid_delegated, redemption_value
from verify_upgrade_staking import shares


class LiquidBackingTests(unittest.TestCase):
    def fixture(self):
        # Addresses here come from an already validated export; only their
        # Bech32 data payload, not their different operator checksum, is compared.
        alice, bob, source, outsider, vault = ["structs1" + value + "abcdef" for value in
            ("alice", "bob", "vesting", "outsider", "vault")]
        validator, other = "structsvaloper1aliceghjkmn", "structsvaloper1otherghjkmn"
        account_map = {address: {"@type": BASE} for address in (alice, bob, outsider, vault)}
        account_map[source] = {"@type": "/cosmos.vesting.v1beta1.DelayedVestingAccount"}
        owners = {alice: "1-1", bob: "1-1", source: "1-1", outsider: "1-2"}
        row = lambda address, target, held: {"delegator_address": address, "validator_address": target, "shares": held}
        unbonding = lambda address, target, initial, balance, identity: {
            "delegator_address": address, "validator_address": target,
            "entries": [{"initial_balance": initial, "balance": balance, "unbonding_id": identity}]}
        staking = {"params": {"bond_denom": "ualpha"},
            "validators": [{"operator_address": validator, "tokens": "9", "delegator_shares": "10"},
                {"operator_address": other, "tokens": "9", "delegator_shares": "10"}],
            "delegations": [row(alice, validator, "4"), row(source, validator, "3"),
                row(outsider, validator, "1"), row(bob, validator, "1"), row(vault, validator, "1"),
                row(alice, other, "1")],
            "unbonding_delegations": [unbonding(alice, validator, "101", "100", "1"),
                unbonding(source, validator, "99", "98", "2"),
                unbonding(outsider, other, "88", "87", "3"),
                unbonding(bob, validator, "3", "2", "4"),
                unbonding(vault, other, "8", "6", "5")],
            "redelegations": [{"delegator_address": alice, "validator_src_address": validator,
                "validator_dst_address": other, "entries": [{"initial_balance": "1", "shares_dst": "1"}]}]}
        return {"app_state": {"staking": staking}}, "1-1", vault, account_map, owners

    def test_merge_before_rounding_retains_self_stake_and_ignores_vesting_and_other_owners(self):
        inputs = self.fixture()
        # Bob + existing vault = 1.8 coins -> 1; Alice's outbound stake = .9 -> 0.
        # Pending current balances add 2 + 6. Retained self-stake and vesting
        # principal do not participate, and redelegation does not add another 1.
        self.assertEqual(original_liquid_delegated(*inputs), [{"denom": "ualpha", "amount": "9"}])
        before = copy.deepcopy(inputs)
        original_liquid_delegated(*inputs)
        self.assertEqual(inputs, before, "verification must never mutate evidence")

    def test_each_validator_rounds_independently_and_zero_current_refunds_contribute_zero(self):
        inputs = self.fixture()
        stake = inputs[0]["app_state"]["staking"]
        stake["delegations"][-1]["shares"] = "2"
        stake["unbonding_delegations"][-1]["entries"][0]["balance"] = "0"
        self.assertEqual(original_liquid_delegated(*inputs), [{"denom": "ualpha", "amount": "4"}])

    def test_sdk_18_place_rounding_is_applied_before_integer_truncation(self):
        vectors = [
            ("1", "3", "1", 0),
            ("2", "3", "3", 2),
            ("1.999999999999999998", "2", "1", 0),
            ("1.999999999999999999", "2", "1", 1),
            ("3.999999999999999999", "4", "2", 2),
            ("9007199254740993.000000000000000001", "9007199254740994", "9007199254740994", 9007199254740993),
        ]
        for held, total, tokens, expected in vectors:
            with self.subTest(held=held, total=total, tokens=tokens):
                self.assertEqual(redemption_value(shares(held), shares(total), int(tokens)), expected)

    def test_missing_and_duplicate_input_cannot_look_like_zero_counter(self):
        for kind in ("validators", "delegations", "unbonding_delegations", "missing_validator", "missing_entries",
                "duplicate_validator", "duplicate_delegation", "duplicate_unbonding", "duplicate_entry", "nonbase_vault"):
            with self.subTest(kind=kind):
                inputs = self.fixture()
                stake = inputs[0]["app_state"]["staking"]
                if kind in ("validators", "delegations", "unbonding_delegations"):
                    del stake[kind]
                elif kind == "missing_validator":
                    stake["validators"] = []
                elif kind == "missing_entries":
                    del stake["unbonding_delegations"][-1]["entries"]
                elif kind == "duplicate_validator":
                    stake["validators"] *= 2
                elif kind == "duplicate_delegation":
                    stake["delegations"] *= 2
                elif kind == "duplicate_unbonding":
                    stake["unbonding_delegations"] *= 2
                elif kind == "duplicate_entry":
                    stake["unbonding_delegations"][-1]["entries"] *= 2
                else:
                    inputs[3][inputs[2]]["@type"] = "/structs.structs.VaultVestingAccount"
                with self.assertRaises(ValueError):
                    original_liquid_delegated(*inputs)

    def test_invalid_shares_and_sdk_overflow_refuse(self):
        for held, total, tokens in ((0, 1, 1), (2, 1, 1), (1, 0, 1), (1, 1, -1),
                (2**256 * PRECISION, 2**256 * PRECISION, 1), (2 * PRECISION, 2 * PRECISION, 2**255)):
            with self.subTest(values=(held, total, tokens)), self.assertRaises(ValueError):
                redemption_value(held, total, tokens)

    def test_missing_preexisting_vault_has_no_invented_stake(self):
        inputs = self.fixture()
        del inputs[3][inputs[2]]
        stake = inputs[0]["app_state"]["staking"]
        for field in ("delegations", "unbonding_delegations"):
            stake[field] = [row for row in stake[field] if row["delegator_address"] != inputs[2]]
        self.assertEqual(original_liquid_delegated(*inputs), [{"denom": "ualpha", "amount": "2"}])


if __name__ == "__main__":
    unittest.main()
