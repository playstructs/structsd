"""Shared collateral coverage must not overflow or silently expire the fixture."""
from pathlib import Path
import re
import subprocess
import unittest

SOURCE = (Path(__file__).parent / 'test_chain.sh').read_text()
HELPER = re.search(r'(?ms)^assert_eq\(\) \{.*?^\}', SOURCE).group()
BLOCK = 'SETTLE_HEIGHT=' + SOURCE.split('SETTLE_HEIGHT=', 1)[1].split('SETTLE_LOAD_AFTER_A=', 1)[0]


class SettlementCoverageTests(unittest.TestCase):
    def invoke(self, end, height, capacity, pool, fault=''):
        script = r'''
set -euo pipefail
GREEN='' RED='' NC='' PASS_COUNT=0 FAIL_COUNT=0
SETTLE_B_END="$1" HEIGHT="$2" SETTLE_CAP_B="$3" POOL="$4" fault="$5" SETTLE_COLL_ADDR=collateral
info() { :; }
get_block_height() { [ "$fault" != height_rpc ] || return 1; echo "$HEIGHT"; }
get_balance_at_height() { [ "$*" = "collateral ualpha $HEIGHT" ]; [ "$fault" != pool_rpc ] || return 1; echo "$POOL"; }
'''
        script += HELPER + '\n' + BLOCK + '\n[ "$FAIL_COUNT" -eq 0 ]\nprintf "%s\\n" "$SETTLE_B_UNEARNED"\n'
        return subprocess.run(['bash', '-c', script, 'test', str(end), str(height), str(capacity), str(pool), fault], capture_output=True, text=True)

    def test_exact_floor_and_one_coin_shortfall(self):
        for end, height, capacity in ((220, 120, 3000), (2**53+121, 2**53+1, 3000),
                                      (2**63-1, 2**63-121, 2**64-1)):
            expected = (end-height)*capacity
            for delta in (-1, 0, 1):
                with self.subTest(end=end, capacity=capacity, delta=delta):
                    result = self.invoke(end, height, capacity, expected+delta)
                    if delta < 0:
                        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
                    else:
                        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                        self.assertEqual(result.stdout.splitlines()[-1], str(expected))

    def test_invalid_inputs_and_expired_fixture_fail(self):
        for index in range(4):
            for bad in ('', '-1', '00', '1.5', 'false', ' 1'):
                values = ['220', '120', '3000', '300000']
                values[index] = bad
                with self.subTest(index=index, bad=bad):
                    self.assertNotEqual(self.invoke(*values).returncode, 0)
        for values in ((220, 0, 3000, 300000), (220, 220, 3000, 0), (220, 221, 3000, 0),
                       (2**64, 120, 3000, 300000), (2**64-1, 2**63, 3000, 300000),
                       (220, 120, 0, 300000), (220, 120, 2**64, 300000)):
            self.assertNotEqual(self.invoke(*values).returncode, 0)
        for fault in ('height_rpc', 'pool_rpc'):
            self.assertNotEqual(self.invoke(220, 120, 3000, 300000, fault).returncode, 0)


if __name__ == '__main__':
    unittest.main()
