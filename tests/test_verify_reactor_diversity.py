import copy
import json
import unittest

from verify_reactor_diversity import MAX_CAPACITY, owner_share, projection, verify_diversity


def infusion(reactor, power, address="a", owner="1-1", commission="0"):
    return {"destinationId": reactor, "destinationType": "reactor" if reactor.startswith("3-") else "struct",
            "address": address, "ownerId": owner, "playerId": owner if owner.startswith("1-") else "",
            "power": str(power), "commission": commission}


def event(row=None):
    return {"type": "structs.structs.EventOwnerReactorDiversity" + ("Reset" if row is None else ""),
            "attributes": ([] if row is None else [{"key": "diversity", "value": json.dumps(row)}]) +
                          [{"key": "mode", "value": "BeginBlock"}]}


def fixture(restart=False, zero=False):
    before = {"initial_height": "0" if zero else "7"}
    rows = [infusion("3-1", 70), infusion("3-1", 30, "b"), infusion("3-2", 100), infusion("5-1", 50)]
    after = {"initial_height": "2" if zero else "8", "app_state": {"structs": {
        "params": {"diversityMaxBonusBps": "5000", "diversityTargetExtraReactors": "9"},
        "playerList": [{"id": "1-1"}], "guildList": [], "infusionList": rows,
        "gridList": [{"attributeId": "2-1-1", "value": "261"}]}}}
    summary = {"ownerId": "1-1", "power": "200", "squaredPower": "20000", "appliedBonus": "11"}
    blocks = [{"height": "1" if zero else "7", "finalize_block_events":
               ([event()] if restart else []) + [event(summary)]}]
    return before, after, blocks


class DiversityVerificationTests(unittest.TestCase):
    def test_aggregation_generator_exclusion_and_upgrade_events(self):
        result = verify_diversity(*fixture(), upgrade_height=7)
        self.assertEqual([], result["differences"])
        self.assertEqual(1, result["owner_portfolios"])

    def test_restart_requires_reset_in_both_height_modes(self):
        for zero in (False, True):
            before, after, blocks = fixture(True, zero)
            self.assertEqual([], verify_diversity(before, after, blocks, restart=True)["differences"])
            blocks[0]["finalize_block_events"].pop(0)
            self.assertTrue(verify_diversity(before, after, blocks, restart=True)["differences"])

    def test_exact_commission_ties_to_even(self):
        self.assertEqual([1, 1, 3, 3], [owner_share(infusion("3-1", p, commission="0.5")) for p in (1, 3, 5, 7)])
        self.assertEqual(1, owner_share(infusion("3-1", 10**18 + 1, commission="0.999999999999999999")))

    def test_nonmonotonic_deposit_and_large_integers(self):
        _, document, _ = fixture()
        state = document["app_state"]["structs"]
        state["infusionList"] = [infusion("3-1", 600)] + [infusion(f"3-{i}", 70) for i in range(2, 22)]
        state["gridList"][0]["value"] = "2859"
        summaries, differences = projection(document)
        self.assertEqual([], differences)
        self.assertEqual("859", summaries["1-1"]["appliedBonus"])
        state["infusionList"][0]["power"] = "650"
        state["gridList"][0]["value"] = "2855"
        summaries, differences = projection(document)
        self.assertEqual([], differences)
        self.assertEqual("805", summaries["1-1"]["appliedBonus"])
        # Generator base constrains representable bonus but never its formula.
        state["infusionList"] = [infusion("3-1", 2**53 + 1), infusion("3-2", 2**53 + 1),
                                 infusion("5-1", MAX_CAPACITY - 2 * (2**53 + 1) - 7)]
        state["gridList"][0]["value"] = str(MAX_CAPACITY)
        summaries, differences = projection(document)
        self.assertEqual([], differences)
        self.assertEqual("7", summaries["1-1"]["appliedBonus"])

    def test_guild_and_member_portfolios_remain_separate(self):
        _, document, _ = fixture()
        state = document["app_state"]["structs"]
        state["guildList"] = [{"id": "0-1"}]
        state["infusionList"] += [infusion("3-3", 900, owner="0-1")]
        state["gridList"] += [{"attributeId": "2-0-1", "value": "900"}]
        summaries, differences = projection(document)
        self.assertEqual([], differences)
        self.assertEqual("0", summaries["0-1"]["appliedBonus"])
        self.assertEqual("11", summaries["1-1"]["appliedBonus"])

    def test_missing_wrong_and_phantom_summary_detected(self):
        for fault in ("missing", "power", "squares", "bonus", "phantom", "grid"):
            before, after, blocks = fixture()
            if fault == "missing": blocks[0]["finalize_block_events"].clear()
            elif fault == "grid": after["app_state"]["structs"]["gridList"][0]["value"] = "260"
            else:
                row = json.loads(blocks[0]["finalize_block_events"][0]["attributes"][0]["value"])
                name, value = {"power": ("power", "201"), "squares": ("squaredPower", "20100"),
                               "bonus": ("appliedBonus", "12"), "phantom": ("ownerId", "1-2")}[fault]
                row[name] = value
                blocks[0]["finalize_block_events"] = [event(row)]
            with self.subTest(fault=fault):
                self.assertTrue(verify_diversity(before, after, blocks, upgrade_height=7)["differences"])

    def test_zero_transition_removes_prior_summary(self):
        before, after, blocks = fixture()
        after["app_state"]["structs"]["infusionList"] = []
        after["app_state"]["structs"]["gridList"] = []
        blocks[0]["finalize_block_events"].append(event({"ownerId": "1-1", "power": "0", "squaredPower": "0", "appliedBonus": "0"}))
        self.assertEqual([], verify_diversity(before, after, blocks, upgrade_height=7)["differences"])
        blocks[0]["finalize_block_events"].pop()
        self.assertTrue(verify_diversity(before, after, blocks, upgrade_height=7)["differences"])

    def test_duplicate_or_late_reset_and_reset_after_summaries_detected(self):
        for fault in ("duplicate", "late", "order"):
            before, after, blocks = fixture(True)
            events = blocks[0]["finalize_block_events"]
            if fault == "duplicate": events.insert(0, event())
            elif fault == "order": events.reverse()
            else:
                events.pop(0)
                after["initial_height"] = "9"
                blocks.append({"height": "8", "finalize_block_events": [event()]})
            self.assertTrue(verify_diversity(before, after, blocks, restart=True)["differences"])

    def test_invalid_inputs_rejected(self):
        for fault in ("window", "tx", "duplicate", "commission", "params", "integer", "owner"):
            before, after, blocks = fixture()
            state = after["app_state"]["structs"]
            if fault == "window": blocks.clear()
            elif fault == "tx": blocks[0]["txs_results"] = [{"code": 1}]
            elif fault == "duplicate": state["infusionList"].append(copy.deepcopy(state["infusionList"][0]))
            elif fault == "commission": state["infusionList"][0]["commission"] = "1.01"
            elif fault == "params": state["params"]["diversityTargetExtraReactors"] = "0"
            elif fault == "owner": state["infusionList"][0]["ownerId"] = "1-2"
            else: blocks[0]["finalize_block_events"] = [event({"ownerId": "1-1", "power": "01", "squaredPower": "1", "appliedBonus": "0"})]
            with self.subTest(fault=fault), self.assertRaises(ValueError):
                verify_diversity(before, after, blocks, upgrade_height=7)


if __name__ == "__main__":
    unittest.main()
