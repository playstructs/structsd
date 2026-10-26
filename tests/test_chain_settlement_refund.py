"""Price consumer refunds at the actual close height with SDK penalty rounding."""
import json
from pathlib import Path
import re
import subprocess
import unittest

SOURCE = (Path(__file__).parent / "test_chain.sh").read_text()
HELPER = re.search(r"(?ms)^expected_settlement_consumer_vault\(\) \{.*?^\}", SOURCE).group()
HASH = "A" * 64


class SettlementRefundTests(unittest.TestCase):
    def invoke(self, before, end, capacity, response, status=0):
        script = '''set -euo pipefail
before="$1" end="$2" capacity="$3" payload="$4" status="$5"
LAST_TX_OUTPUT='{ "txhash": "''' + HASH + '''" }'
_tx_json() { printf '%s' "$1"; }
query() { [ "$*" = 'query tx ''' + HASH + '''' ]; printf '%s' "$payload"; return "$status"; }
'''
        script += HELPER + '\nexpected_settlement_consumer_vault "$before" "$end" "$capacity"\n'
        return subprocess.run(["bash", "-c", script, "test", str(before), str(end), str(capacity),
                               json.dumps(response), str(status)], capture_output=True, text=True)

    def test_exact_refund_and_penalty_truncation(self):
        for base in (100, 2**53 + 1, 2**64 - 11):
            for capacity, blocks, refund in ((2000, 7, 10500), (3000, 7, 15750),
                                             (7, 1, 6), (3, 1, 3), (2000, 0, 0)):
                with self.subTest(base=base, capacity=capacity, blocks=blocks):
                    result = self.invoke(2**64, base+blocks, capacity, {"code": 0, "txhash": HASH, "height": str(base)})
                    self.assertEqual(result.returncode, 0, result.stderr)
                    self.assertEqual(result.stdout.strip(), str(2**64+refund))

    def test_invalid_delivery_or_impossible_window_has_no_refund(self):
        valid = {"code": 0, "txhash": HASH, "height": "100"}
        for response in ({}, dict(valid, code=4), dict(valid, code=False), dict(valid, txhash="B"*64),
                         dict(valid, height="101"), dict(valid, height="-1"), dict(valid, height=100)):
            result = self.invoke(10, 100, 7, response)
            self.assertNotEqual(result.returncode, 0)
            self.assertEqual(result.stdout, "")
        result = self.invoke(10, 100, 7, valid, status=1)
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(result.stdout, "")


if __name__ == "__main__":
    unittest.main()
