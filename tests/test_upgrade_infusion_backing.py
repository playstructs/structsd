import copy
import unittest

from verify_upgrade_infusion_backing import verify_infusion_backing


class InfusionBackingTests(unittest.TestCase):
    def setUp(self):
        self.vaults = {"1-1": "vault"}
        self.row = {"destinationId": "3-1", "destinationType": "reactor",
            "address": "vault", "playerId": "1-1", "ownerId": "",
            "fuel": "7", "defusing": "2",
            "power": "7", "ratio": "1", "commission": "0.04"}
        self.validator = {"operator_address": "validator", "tokens": "10",
            "delegator_shares": "4", "jailed": False, "status": "BOND_STATUS_BONDED"}
        self.state = {"app_state": {"structs": {
            "playerList": [{"id": "1-1"}], "guildList": [],
            "addressList": [{"address": "key", "playerIndex": "1"}],
            "vaultOwnerList": ["1-1"], "reactorList": [{"id": "3-1", "validator": "validator"}],
            "infusionList": [self.row]}, "staking": {"validators": [self.validator],
            "delegations": [{"delegator_address": "vault", "validator_address": "validator", "shares": "3"}],
            "unbonding_delegations": [{"delegator_address": "vault", "validator_address": "validator",
                "entries": [{"balance": "2"}]}]}}}

    def test_fractional_staking_backing_and_unbonding(self):
        self.assertEqual(verify_infusion_backing(self.state, self.vaults)["differences"], [])

    def test_invalid_economics_cannot_pass_by_matching_exported_events(self):
        for field, value in (("fuel", "8"), ("defusing", "0"),
                ("power", "6"), ("ratio", "0"), ("playerId", "1-2"), ("commission", "1.01")):
            with self.subTest(field=field):
                state = copy.deepcopy(self.state)
                state["app_state"]["structs"]["infusionList"][0][field] = value
                self.assertTrue(verify_infusion_backing(state, self.vaults)["differences"])

    def test_ineligible_validator_retains_fuel_but_has_no_power(self):
        for jailed, status in ((True, "BOND_STATUS_BONDED"), (False, "BOND_STATUS_UNBONDED"),
                (False, "BOND_STATUS_UNBONDING")):
            with self.subTest(jailed=jailed, status=status):
                self.validator.update(jailed=jailed, status=status)
                self.row.update(ratio="1", power="5")
                self.assertTrue(verify_infusion_backing(self.state, self.vaults)["differences"])
                self.row.update(ratio="0", power="0")
                self.assertEqual(verify_infusion_backing(self.state, self.vaults)["differences"], [])

    def test_missing_infusion_is_detected(self):
        self.state["app_state"]["structs"]["infusionList"] = []
        self.assertTrue(verify_infusion_backing(self.state, self.vaults)["differences"])

    def test_missing_and_duplicate_associations_refused(self):
        with self.assertRaises(ValueError):
            verify_infusion_backing(self.state, {})
        self.state["app_state"]["structs"]["vaultOwnerList"].append("1-1")
        with self.assertRaises(ValueError):
            verify_infusion_backing(self.state, self.vaults)

    def test_duplicate_position_refused(self):
        self.state["app_state"]["structs"]["infusionList"].append(copy.deepcopy(self.row))
        with self.assertRaises(ValueError):
            verify_infusion_backing(self.state, self.vaults)


if __name__ == "__main__":
    unittest.main()
