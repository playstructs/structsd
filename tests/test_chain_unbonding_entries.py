"""Exercise the live defuse/cancel readers against native empty-list encoding."""
import json
from pathlib import Path
import re
import subprocess
import unittest

ROOT = Path(__file__).resolve().parent
SOURCE = (ROOT / "test_chain.sh").read_text()
LATEST = re.search(r"(?m)^\s*UNBOND_HEIGHT=.*$", SOURCE).group().strip()
COUNT = re.search(r"(?m)^\s*P2_CANCELLED_ENTRIES=.*$", SOURCE).group().strip()


def response(*heights):
    return {"pagination": {}, "unbonding_responses": [{"delegator_address": "vault", "validator_address": "validator",
            "entries": [{"creation_height": str(value)} for value in heights]}]}


class UnbondingEntryTests(unittest.TestCase):
    def invoke(self, payload, assignment=COUNT, wanted="123", status=0):
        script = '''set -euo pipefail
REACTOR_TEST_DIR="$1" UNBONDING_JSON="$2" STATUS="$3" UNBOND_HEIGHT="$4"
PLAYER_2_VAULT=vault P2_VAL_ADDR=validator
query() {
    [ "$*" = 'query staking unbonding-delegations vault --page-limit 100' ] || return 9
    printf '%s' "$UNBONDING_JSON"
    return "$STATUS"
}
'''
        variable = assignment.split('=', 1)[0]
        script += assignment + '\nprintf "%s\\n" "${' + variable + '}"\n'
        return subprocess.run(["bash", "-c", script, "test", str(ROOT), payload, str(status), wanted],
                              capture_output=True, text=True)

    def test_empty_native_null_omitted_and_array_lists_are_zero(self):
        for payload in ({"pagination": {}}, {"pagination": {}, "unbonding_responses": None},
                        {"pagination": {"next_key": None}, "unbonding_responses": []}):
            result = self.invoke(json.dumps(payload))
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(result.stdout, "0\n")
            latest = self.invoke(json.dumps(payload), LATEST)
            self.assertEqual(latest.returncode, 0, latest.stderr)
            self.assertEqual(latest.stdout, "\n")

    def test_exact_latest_height_and_remaining_entry_count(self):
        large = 2**53 + 1
        payload = response(large, large - 1, 123, 123)
        payload["unbonding_responses"].append({"delegator_address": "vault", "validator_address": "other",
                                               "entries": [{"creation_height": str(2**63 - 1)}]})
        for assignment, wanted, expected in ((LATEST, "123", str(large)), (COUNT, "123", "2"),
                                             (COUNT, str(large), "1"), (COUNT, "124", "0")):
            result = self.invoke(json.dumps(payload), assignment, wanted)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(result.stdout.strip(), expected)

    def test_incomplete_ambiguous_and_failed_queries_cannot_prove_cancellation(self):
        for page in ({"next_key": "continuation"}, {"nextKey": "continuation"},
                     {"next_key": False}, {"next_key": "", "nextKey": "continuation"}):
            result = self.invoke(json.dumps({"pagination": page, "unbonding_responses": None}))
            self.assertNotEqual(result.returncode, 0)
            self.assertEqual(result.stdout, "")
        result = self.invoke('{"pagination":{},"unbonding_responses":null}', status=7)
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(result.stdout, "")

    def test_malformed_records_and_heights_refuse(self):
        malformed = ["", "broken", "{}", "null", '{"pagination":null}',
                     '{"pagination":{},"unbonding_responses":false}']
        for value in (None, False, 123, "-1", "00", "1.5", str(2**63)):
            payload = response(123)
            payload["unbonding_responses"][0]["entries"][0]["creation_height"] = value
            malformed.append(json.dumps(payload))
        wrong = response(123)
        wrong["unbonding_responses"][0]["delegator_address"] = "someone-else"
        malformed.append(json.dumps(wrong))
        duplicate = response(123)
        duplicate["unbonding_responses"] *= 2
        malformed.append(json.dumps(duplicate))
        for payload in malformed:
            for assignment in (LATEST, COUNT):
                result = self.invoke(payload, assignment)
                self.assertNotEqual(result.returncode, 0, payload)
                self.assertEqual(result.stdout, "")


if __name__ == "__main__":
    unittest.main()
