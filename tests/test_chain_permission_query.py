"""Exact permission queries must not turn missing data or RPC errors into zero."""
import json
from pathlib import Path
import re
import subprocess
import unittest

SOURCE = (Path(__file__).parent / "test_chain.sh").read_text()
HELPER = re.search(r"(?ms)^get_permission_value_for_player\(\) \{.*?^\}", SOURCE).group()


class PermissionQueryTests(unittest.TestCase):
    def invoke(self, payload, status=0):
        script = '''set -euo pipefail
payload="$1" status="$2"
query() {
    [ "$*" = 'query structs permission 6-1@1-5' ] || return 9
    printf '%s' "$payload"
    return "$status"
}
'''
        script += HELPER + '\nget_permission_value_for_player 6-1 1-5\n'
        return subprocess.run(["bash", "-c", script, "test", payload, str(status)], capture_output=True, text=True)

    def test_exact_values_including_omitted_zero(self):
        for value in (None, "0", "2062", str(2**53 + 1), str(2**64 - 1)):
            record = {"permissionId": "6-1@1-5"}
            if value is not None:
                record["value"] = value
            result = self.invoke(json.dumps({"permissionRecord": record}))
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(result.stdout.strip(), value or "0")

    def test_errors_and_incomplete_responses_never_produce_zero(self):
        payloads = ["", "not-json", "{}", '{"permissionRecords":[]}',
                    json.dumps({"permissionRecord": {"permissionId": "6-1@1-99", "value": "0"}})]
        for value in (None, False, 0, -1, "", "01", "-1", "1.0", str(2**64)):
            payloads.append(json.dumps({"permissionRecord": {"permissionId": "6-1@1-5", "value": value}}))
        for payload in payloads:
            with self.subTest(payload=payload):
                result = self.invoke(payload)
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(result.stdout, "")
        result = self.invoke('{"permissionRecord":{"permissionId":"6-1@1-5","value":"0"}}', 1)
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(result.stdout, "")


if __name__ == "__main__":
    unittest.main()
