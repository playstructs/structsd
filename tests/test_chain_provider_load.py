"""A failed provider query must never prove that settlement released its load."""
import json
from pathlib import Path
import re
import subprocess
import unittest

SOURCE = (Path(__file__).parent / "test_chain.sh").read_text()
HELPER = re.search(r"(?ms)^get_provider_load\(\) \{.*?^\}", SOURCE).group()


class ProviderLoadTests(unittest.TestCase):
    def invoke(self, payload, status=0, command="get_provider_load 10-1"):
        script = '''set -euo pipefail
response="$1" status="$2" SETTLE_PROV_ID=10-1
query() { [ "$*" = 'query structs provider 10-1' ]; printf '%s' "$response"; return "$status"; }
'''
        script += HELPER + '\n' + command + '\n'
        return subprocess.run(["bash", "-c", script, "test", payload, str(status)], capture_output=True, text=True)

    def test_actual_settlement_load_reads_preserve_exact_values(self):
        assignments = re.findall(r"(?m)^\s*SETTLE_LOAD_[A-Z_]+=\$\(get_provider_load[^\n]+", SOURCE)
        self.assertEqual(len(assignments), 4)
        for assignment in assignments:
            variable = assignment.strip().split("=", 1)[0]
            for load in (None, "0", "5000", str(2**53+1), str(2**64-1)):
                grid = {} if load is None else {"load": load}
                response = {"Provider": {"id": "10-1"}, "gridAttributes": grid}
                result = self.invoke(json.dumps(response), command=assignment + '\nprintf "%s\\n" "${' + variable + '}"')
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertEqual(result.stdout.strip(), load or "0")

    def test_errors_and_malformed_loads_emit_nothing(self):
        responses = ["", "broken", "{}", '{"Provider":{"id":"10-1"}}',
                     '{"Provider":{"id":"10-2"},"gridAttributes":{}}']
        for load in (False, None, 0, "", "-1", "00", "1.0", str(2**64)):
            responses.append(json.dumps({"Provider": {"id": "10-1"}, "gridAttributes": {"load": load}}))
        for response in responses:
            result = self.invoke(response)
            self.assertNotEqual(result.returncode, 0)
            self.assertEqual(result.stdout, "")
        result = self.invoke('{"Provider":{"id":"10-1"},"gridAttributes":{}}', status=1)
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(result.stdout, "")


if __name__ == "__main__":
    unittest.main()
