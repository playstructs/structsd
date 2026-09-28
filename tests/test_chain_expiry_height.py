"""Missing or invalid expiry data must not become a successful wait at block 2."""
import json
from pathlib import Path
import subprocess
import unittest

HELPER = Path(__file__).with_name("agreement_expiry_height.py")


class ExpiryHeightTests(unittest.TestCase):
    def invoke(self, payload):
        return subprocess.run(["python3", "-B", str(HELPER), "11-1"], input=json.dumps(payload),
                              capture_output=True, text=True)

    def test_exact_post_expiry_height(self):
        for end in (1, 100, 2**53+1, 2**63-3):
            result = self.invoke({"Agreement": {"id": "11-1", "endBlock": str(end)}})
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(json.loads(result.stdout), {"endBlock": str(end), "waitHeight": str(end+2)})

    def test_missing_mismatched_and_unrepresentable_expiry_is_rejected(self):
        payloads = [{}, {"Agreement": None}, {"Agreement": {"id": "11-2", "endBlock": "100"}}]
        for end in (None, False, 100, "", "0", "01", "-1", "1.0", str(2**63-2), str(2**64-1)):
            payloads.append({"Agreement": {"id": "11-1", "endBlock": end}})
        for payload in payloads:
            result = self.invoke(payload)
            self.assertNotEqual(result.returncode, 0)
            self.assertEqual(result.stdout, "")


if __name__ == "__main__":
    unittest.main()
