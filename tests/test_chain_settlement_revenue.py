"""Conserved funds in the wrong pool must not pass settlement accounting."""
from pathlib import Path
import re
import subprocess
import unittest

SOURCE = (Path(__file__).parent / "test_chain.sh").read_text()
ASSERT = re.search(r"(?ms)^assert_eq\(\) \{.*?^\}", SOURCE).group()
BLOCK = SOURCE.split("# ─── dual-agreement earned revenue ───", 1)[1].split(
    "# ─── dual-agreement earned revenue complete ───", 1)[0]


class SettlementRevenueTests(unittest.TestCase):
    def test_stranded_revenue_fails_even_when_total_is_conserved(self):
        for base in (100, 2**64):
            for stranded in (0, 1, 30000):
                with self.subTest(base=base, stranded=stranded):
                    script = '''set -euo pipefail
GREEN='' RED='' NC='' PASS_COUNT=0 FAIL_COUNT=0
SETTLE_COLL_A=240000 SETTLE_COLL_B=360000 P2_BEFORE_CLOSE="$1" P3_BEFORE_CLOSE="$1"
P2_EXPECTED_AFTER_CLOSE="$2" P3_EXPECTED_AFTER_CLOSE="$3" SETTLE_EARN_END="$4" SETTLE_POOL_END="$5"
'''
                    script += ASSERT + '\n' + BLOCK + '\n[ "$FAIL_COUNT" -eq 0 ]\n'
                    result = subprocess.run(["bash", "-c", script, "test", str(base), str(base+180000),
                                             str(base+270000), str(150000-stranded), str(stranded)],
                                            capture_output=True, text=True)
                    if stranded == 0:
                        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                    else:
                        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)


if __name__ == "__main__":
    unittest.main()
