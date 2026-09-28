"""Exercise the live query checks, including native field names and failures."""
import json
from pathlib import Path
import re
import subprocess
import unittest


SOURCE = Path(__file__).with_name("test_chain.sh").read_text()


def fragment(name):
    boundaries = {
        "grid": ('info "Grid attributes sample:"', "\nfi # phase 4f"),
        "infusion": ('    info "Infusion query coverage:"', "\nfi\n\nfi # phase 4g"),
        "planet": ('info "Planet attribute query:"', 'info "Planets by player'),
    }
    start, stop = boundaries[name]
    return SOURCE.split(start, 1)[1].split(stop, 1)[0]


def invoke(name, response, secondary=None, status=0):
    helpers = "\n".join(re.search(r"(?ms)^" + function + r"\(\) \{.*?^\}", SOURCE).group()
        for function in ("assert_eq", "assert_gt"))
    script = '''set -euo pipefail
GREEN='' RED='' NC=''
PASS_COUNT=0 FAIL_COUNT=0
PLAYER_2_PLANET_ID=2-1
payload="$1" secondary="$2" status="$3"
query() {
    case "$*" in
        'query structs grid-all --page-limit 3'|'query structs infusion-all --page-limit 3'|'query structs planet 2-1 --height 123')
            printf '%s' "$payload"; return "$status";;
        'query structs reactor-all --page-limit 3'|'query structs planet-attribute 2-1 planetaryShield --height 123')
            printf '%s' "$secondary"; return "$status";;
        'status') printf '%s' '{"sync_info":{"latest_block_height":"123"}}';;
        *) return 9;;
    esac
}
'''
    script += helpers + "\n" + fragment(name)
    expected = 2 if name == "infusion" else 1
    script += f'\n[ "$FAIL_COUNT" -eq 0 ]\n[ "$PASS_COUNT" -eq {expected} ]\n'
    return subprocess.run(["bash", "-c", script, "test", json.dumps(response),
        json.dumps(secondary), str(status)], capture_output=True, text=True)


class QueryCoverageTests(unittest.TestCase):
    def test_native_grid_infusion_and_reactor_samples(self):
        grid = {"gridRecords": [{"attributeId": "1-2@capacity", "value": "10000000"},
            {"attributeId": "1-2@load"}], "pagination": {"next_key": "YWJj"}}
        result = invoke("grid", grid)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn("1-2@capacity=10000000", result.stdout)
        self.assertIn("1-2@load=0", result.stdout)
        result = invoke("infusion", {"Infusion": [{"destinationId": "3-1", "address": "vault"}]},
            {"Reactor": [{"id": "3-1", "validator": "validator"}]})
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn("dst=3-1 addr=vault", result.stdout)
        self.assertIn("3-1 validator=validator", result.stdout)

    def test_samples_refuse_missing_records_wrong_fields_and_rpc_failure(self):
        valid_reactor = {"Reactor": [{"id": "3-1", "validator": "validator"}]}
        for name, field, row in (("grid", "gridRecords", {"attributeId": "capacity"}),
                ("infusion", "Infusion", {"destinationId": "3-1", "address": "vault"})):
            for response in ({}, {field: None}, {field: []}, {field: {}}, {field: [{}]},
                    {field: [None]}, {field: [row, {}]}, {field.lower(): [row]}):
                # The grid field's lowercase spelling already differs in Records.
                with self.subTest(name=name, response=response):
                    self.assertNotEqual(invoke(name, response, valid_reactor).returncode, 0)
            self.assertNotEqual(invoke(name, {field: [row]}, valid_reactor, status=1).returncode, 0)
        valid_infusion = {"Infusion": [{"destinationId": "3-1", "address": "vault"}]}
        for reactor in ({}, {"Reactor": []}, {"Reactor": [{"id": "3-1"}]}):
            self.assertNotEqual(invoke("infusion", valid_infusion, reactor).returncode, 0)

    def test_planet_attribute_matches_detail_at_one_height(self):
        for value in (None, "0", "100", str(2**64 - 1)):
            detail = {"Planet": {"id": "2-1"}, "planetAttributes": {}}
            attribute = {}
            if value is not None:
                detail["planetAttributes"]["planetaryShield"] = value
                attribute["attribute"] = value
            result = invoke("planet", detail, attribute)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_planet_refuses_wrong_identity_shape_value_and_rpc_failure(self):
        valid = {"Planet": {"id": "2-1"}, "planetAttributes": {"planetaryShield": "100"}}
        for detail, attribute in ((valid, {"attribute": "99"}),
                ({"Planet": {"id": "2-2"}, "planetAttributes": {}}, {}),
                ({"Planet": {"id": "2-1"}}, {}),
                ({}, {}), (valid, {"landSlots": "100"}), (valid, [])):
            self.assertNotEqual(invoke("planet", detail, attribute).returncode, 0)
        for value in (None, False, 100, "", "01", "-1", "1.0", str(2**64), "9" * 21):
            self.assertNotEqual(invoke("planet", valid, {"attribute": value}).returncode, 0)
            detail = {"Planet": {"id": "2-1"}, "planetAttributes": {"planetaryShield": value}}
            self.assertNotEqual(invoke("planet", detail, {"attribute": value}).returncode, 0)
        self.assertNotEqual(invoke("planet", valid, {"attribute": "100"}, status=1).returncode, 0)


if __name__ == "__main__":
    unittest.main()
