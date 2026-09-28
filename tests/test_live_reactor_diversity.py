import copy
import json
from pathlib import Path
import re
import subprocess
import unittest

from live_reactor_diversity import verify_live
from test_verify_reactor_diversity import fixture, infusion


class LiveDiversityTests(unittest.TestCase):
    def fixture(self):
        _, document, _ = fixture()
        state = document["app_state"]["structs"]
        result = {"stored": {"ownerId": "1-1", "power": "200", "squaredPower": "20000", "appliedBonus": "11"},
                  "liveBonus": "11", "usableBonus": "11", "maxBonusBps": "5000", "targetExtraReactors": "9",
                  "initialized": True, "livePortfolioComplete": True, "infusionInventoryComplete": True,
                  "storedCapacity": "261", "usablePrimaryCapacity": "261"}
        calls = []
        def query(*args):
            calls.append(args)
            if args == ("status",): return {"sync_info": {"latest_block_height": "25"}}
            self.assertEqual(args[-2:], ("--height", "25"))
            if args[2] == "player": return {"Player": {"id": "1-1"}, "gridAttributes": {"capacity": "261"}}
            if args[2] == "params": return {"params": state["params"]}
            if args[2] == "owner-reactor-diversity": return result
            self.assertEqual(args[2], "infusion-all")
            if "--page-key" in args:
                self.assertEqual(args[args.index("--page-key")+1], "bmV4dA==")
                return {"Infusion": state["infusionList"][2:] + [infusion("3-9", 9000, owner="0-7")],
                        "pagination": {"next_key": None}}
            return {"Infusion": state["infusionList"][:2], "pagination": {"next_key": "bmV4dA=="}}
        return state, result, calls, query

    def test_pins_every_page_and_excludes_other_owners(self):
        _, _, calls, query = self.fixture()
        proof = verify_live(query, "1-1")
        self.assertEqual("verified", proof["status"])
        self.assertEqual("25", proof["height"])
        self.assertEqual(4, proof["infusions"])
        self.assertEqual(1, sum(call == ("status",) for call in calls))

    def test_changed_totals_bonus_capacity_and_incomplete_proofs_rejected(self):
        for field, value in (("liveBonus", "12"), ("usableBonus", "10"), ("usablePrimaryCapacity", "260"),
                             ("storedCapacity", "260"), ("maxBonusBps", "4000"), ("targetExtraReactors", "8"),
                             ("initialized", False), ("livePortfolioComplete", False), ("infusionInventoryComplete", False)):
            with self.subTest(field=field):
                _, result, _, query = self.fixture()
                result[field] = value
                with self.assertRaises(ValueError): verify_live(query, "1-1")
        _, result, _, query = self.fixture()
        result["stored"]["power"] = "201"
        with self.assertRaisesRegex(ValueError, "complete infusion accounting"):
            verify_live(query, "1-1")

    def test_rpc_omitted_zero_power_preserves_accounting_and_input(self):
        state, _, _, query = self.fixture()
        inactive = infusion("3-4", 0)
        del inactive["power"]
        state["infusionList"].append(inactive)
        proof = verify_live(query, "1-1")
        self.assertEqual("verified", proof["status"])
        self.assertEqual(5, proof["infusions"])
        self.assertEqual("11", proof["summary"]["appliedBonus"])
        self.assertNotIn("power", inactive, "normalization must not rewrite captured RPC evidence")

    def test_omitting_nonzero_power_does_not_hide_an_accounting_mismatch(self):
        state, _, _, query = self.fixture()
        del state["infusionList"][0]["power"]
        with self.assertRaisesRegex(ValueError, "capacity differs"):
            verify_live(query, "1-1")

    def test_missing_or_duplicate_pages_cannot_appear_complete(self):
        for fault in ("missing", "duplicate", "cycle", "budget", "malformed"):
            state, _, _, original = self.fixture()
            def query(*args):
                data = original(*args)
                if "--page-key" in args:
                    if fault == "missing": data["Infusion"] = []
                    elif fault == "duplicate": data["Infusion"].append(copy.deepcopy(state["infusionList"][0]))
                    elif fault == "cycle": data["pagination"]["next_key"] = "bmV4dA=="
                    elif fault == "malformed": data["Infusion"] = {}
                return data
            with self.subTest(fault=fault), self.assertRaises(ValueError):
                verify_live(query, "1-1", max_pages=1 if fault == "budget" else 100)

    def test_shell_assertion_records_failed_proof(self):
        source = Path(__file__).with_name("test_chain.sh").read_text()
        helper = re.search(r"(?ms)^check_fixture_diversity\(\) \{.*?^\}", source).group()
        assertion = re.search(r"(?ms)^assert_eq\(\) \{.*?^\}", source).group()
        for good in (True, False):
            code = '''set -euo pipefail
GREEN='' RED='' NC='' GENERATOR_TEST_DIR=unused PARAMS_QUERY='--output json'
PASS_COUNT=0 FAIL_COUNT=0
python3() { printf '%s' "$1" >/dev/null; printf '%s' "$PROOF"; return "$CODE"; }
'''
            code += assertion + '\n' + helper + '\ncheck_fixture_diversity 1-1 test\n'
            code += '[ "$FAIL_COUNT" -eq "$EXPECTED" ]\n[ "$((PASS_COUNT + FAIL_COUNT))" -eq 1 ]\n'
            import os
            env = dict(os.environ, PROOF=json.dumps({"status": "verified"}) if good else "broken query",
                       CODE="0" if good else "1", EXPECTED="0" if good else "1")
            result = subprocess.run(["bash", "-c", code], env=env, capture_output=True, text=True)
            self.assertEqual(0, result.returncode, result.stdout + result.stderr)


if __name__ == "__main__":
    unittest.main()
