"""Offline checks for the live cap scenario's state and delivered-event assertions."""
import json
from pathlib import Path
import subprocess
import unittest

HELPER = Path(__file__).with_name("substation_capacity_maximum.sh")
HASH = "A" * 64


def invoke(function, document, *arguments):
    script = '''set -euo pipefail
source "$1"
CAP_MAX_STATION=4-99
LAST_TX_OUTPUT="$2"
response="$3"
query() { printf '%s' "$response"; }
_tx_json() { printf '%s' "$1"; }
assert_eq() { [ "$2" = "$3" ] || { echo "$1: expected $2, got $3" >&2; return 1; }; }
function_name="$4"
shift 4
"$function_name" fixture "$@"
'''
    return subprocess.run(["bash", "-c", script, "test", str(HELPER),
        json.dumps({"txhash": HASH}), json.dumps(document), function,
        *map(str, arguments)], capture_output=True, text=True)


def grid_event(attribute):
    return {"type": "structs.structs.EventGrid", "attributes": [
        {"key": "gridRecord", "value": json.dumps({"attributeId": attribute, "value": "200"})}]}


class SubstationCapacityMaximumTests(unittest.TestCase):
    def test_delivered_writes_and_no_write_transaction(self):
        for events, counts in (([], (0, 0)),
                               ([grid_event("6-4-99"), grid_event("15-4-99")], (1, 1)),
                               ([grid_event("6-4-100")], (0, 0))):
            result = invoke("capacity_maximum_assert_events", {
                "txhash": HASH, "height": "123", "code": 0, "events": events}, *counts)
            self.assertEqual(result.returncode, 0, result.stderr)

    def test_missing_failed_or_wrong_transaction_cannot_pass_as_no_writes(self):
        valid = {"txhash": HASH, "height": "123", "code": 0, "events": []}
        for document in ({}, {**valid, "txhash": "B" * 64}, {**valid, "code": 1},
                         {**valid, "height": "0"}, {**valid, "events": None},
                         {**valid, "events": [grid_event("6-4-99")]},
                         {**valid, "events": [grid_event("15-4-99")]}):
            result = invoke("capacity_maximum_assert_events", document, 0, 0)
            self.assertNotEqual(result.returncode, 0, document)

    def test_state_checks_exact_uint64_values_and_identity(self):
        maximum = str(2**64 - 1)
        document = {"Substation": {"id": "4-99"}, "gridAttributes": {
            "capacity": maximum, "load": "1", "connectionCount": "2",
            "connectionCapacityMaximum": maximum, "connectionCapacity": "100"}}
        result = invoke("capacity_maximum_assert_state", document, maximum, 1, 2, maximum, 100)
        self.assertEqual(result.returncode, 0, result.stderr)
        for field in document["gridAttributes"]:
            wrong = {**document, "gridAttributes": {**document["gridAttributes"], field: "3"}}
            result = invoke("capacity_maximum_assert_state", wrong, maximum, 1, 2, maximum, 100)
            self.assertNotEqual(result.returncode, 0, field)
        document["Substation"]["id"] = "4-100"
        self.assertNotEqual(invoke("capacity_maximum_assert_state", document, maximum, 1, 2, maximum, 100).returncode, 0)

    def test_scenario_is_called_from_test_chain(self):
        source = HELPER.with_name("test_chain.sh").read_text()
        self.assertIn('source "${GENERATOR_TEST_DIR}/substation_capacity_maximum.sh"', source)
        self.assertIn('\nrun_substation_capacity_maximum_tests\n', source)


if __name__ == "__main__":
    unittest.main()
