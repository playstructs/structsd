"""Guild cleanup accepts omitted zero load without accepting missing objects."""
import json
from pathlib import Path
import re
import subprocess
import unittest

ROOT = Path(__file__).resolve().parent
SOURCE = (ROOT / "test_chain.sh").read_text()
ASSERT = re.search(r"(?ms)^assert_eq\(\) \{.*?^\}", SOURCE).group()
BEFORE = re.search(r"(?m)^GGRID_SOURCE_BEFORE=.*$", SOURCE).group()
AFTER = re.search(r"(?m)^\s*GGRID_SOURCE_AFTER=.*$", SOURCE).group().strip()
CLEANUP = re.search(r'(?m)^\s*assert_eq "Main substation load restored after guild grid cleanup".*$', SOURCE).group().strip()


class GuildGridTests(unittest.TestCase):
    def invoke(self, payload, assignment=AFTER, status=0, cleanup=None):
        script = '''set -euo pipefail
GGRID_TEST_DIR="$1" RESPONSE="$2" STATUS="$3" SUBSTATION_ID=4-1
GREEN='' RED='' NC='' PASS_COUNT=0 FAIL_COUNT=0
query() {
    [ "$*" = 'query structs substation 4-1' ] || return 9
    printf '%s' "$RESPONSE"
    return "$STATUS"
}
'''
        script += ASSERT + '\n' + assignment + '\n'
        if cleanup is not None:
            script += 'GGRID_LOAD_BEFORE="$4"\n' + CLEANUP + '\n[ "$FAIL_COUNT" -eq 0 ]\n'
        else:
            variable = assignment.split('=', 1)[0]
            script += 'printf "%s\\n" "${' + variable + '}"\n'
        return subprocess.run(["bash", "-c", script, "test", str(ROOT), payload, str(status), cleanup or ""],
                              capture_output=True, text=True)

    def test_actual_cleanup_accepts_omitted_zero_and_exact_uint64_load(self):
        for value in (None, "0", "1", str(2**53 + 1), str(2**64 - 1)):
            grid = {} if value is None else {"load": value}
            result = self.invoke(json.dumps({"Substation": {"id": "4-1"}, "gridAttributes": grid}), cleanup=value or "0")
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_actual_cleanup_detects_nonzero_residual_load(self):
        result = self.invoke(json.dumps({"Substation": {"id": "4-1"}, "gridAttributes": {"load": "1"}}), cleanup="0")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("FAIL", result.stdout)

    def test_before_snapshot_uses_exact_saturating_headroom(self):
        for capacity, load in ((1000, 0), (1000, 1), (0, 1), (2**64 - 1, 0), (2**64 - 1, 2**64 - 2)):
            payload = {"Substation": {"id": "4-1"}, "gridAttributes": {"capacity": str(capacity), "load": str(load)}}
            result = self.invoke(json.dumps(payload), BEFORE)
            self.assertEqual(result.returncode, 0, result.stderr)
            snapshot = json.loads(result.stdout)
            self.assertEqual(snapshot["load"], str(load))
            self.assertEqual(snapshot["allocatableCapacity"], str(max(capacity * 9 // 10 - load, 0)))
            self.assertEqual(snapshot["hasCapacity"], capacity * 9 // 10 - load >= 1000)

    def test_missing_wrong_malformed_and_failed_queries_never_become_zero(self):
        payloads = ["", "broken", "null", "{}", '{"Substation":{"id":"4-1"}}',
                    '{"Substation":{"id":"4-2"},"gridAttributes":{}}',
                    '{"Substation":{"id":"4-1"},"gridAttributes":null}']
        for field in ("load", "capacity"):
            for value in (None, False, 0, "", "00", "-1", "1.5", str(2**64)):
                payloads.append(json.dumps({"Substation": {"id": "4-1"}, "gridAttributes": {field: value}}))
        for assignment in (BEFORE, AFTER):
            for payload in payloads:
                result = self.invoke(payload, assignment)
                self.assertNotEqual(result.returncode, 0, payload)
                self.assertEqual(result.stdout, "")
            result = self.invoke('{"Substation":{"id":"4-1"},"gridAttributes":{}}', assignment, status=1)
            self.assertNotEqual(result.returncode, 0)
            self.assertEqual(result.stdout, "")


if __name__ == "__main__":
    unittest.main()
