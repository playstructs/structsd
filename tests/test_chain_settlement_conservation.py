"""Reconcile actual consumer refunds and both pools with the original deposits."""
from pathlib import Path
import re
import subprocess
import unittest

SOURCE = (Path(__file__).parent / "test_chain.sh").read_text()
ASSERT = re.search(r"(?ms)^assert_eq\(\) \{.*?^\}", SOURCE).group()
BLOCK = SOURCE.split("# ─── dual-agreement exact conservation ───", 1)[1].split(
    "# ─── dual-agreement exact conservation complete ───", 1)[0]


class SettlementConservationTests(unittest.TestCase):
    def test_exact_total_detects_missing_or_extra_coins(self):
        for base in (100, 2**53 + 1, 2**64 - 1):
            values = [base, base+180000, base, base+270000, 17, 149983, 240000, 360000]
            for field in (None, *range(len(values))):
                for delta in ((0,) if field is None else (-1, 1)):
                    with self.subTest(base=base, field=field, delta=delta):
                        actual = list(values)
                        if field is not None:
                            actual[field] += delta
                        script = '''set -euo pipefail
GREEN='' RED='' NC='' PASS_COUNT=0 FAIL_COUNT=0 SETTLE_COLL_ADDR=collateral SETTLE_EARN_ADDR=earnings
P2_BEFORE_CLOSE="$1" P2_AFTER_CLOSE="$2" P3_BEFORE_CLOSE="$3" P3_AFTER_CLOSE="$4"
COLLATERAL="$5" EARNINGS="$6" SETTLE_COLL_A="$7" SETTLE_COLL_B="$8"
get_balance() {
    [ "$2" = ualpha ]
    case "$1" in collateral) echo "$COLLATERAL" ;; earnings) echo "$EARNINGS" ;; *) return 9 ;; esac
}
'''
                        script += ASSERT + '\n' + BLOCK + '\n[ "$FAIL_COUNT" -eq 0 ]\n'
                        result = subprocess.run(["bash", "-c", script, "test", *map(str, actual)], capture_output=True, text=True)
                        if field is None:
                            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                        else:
                            self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)


if __name__ == "__main__":
    unittest.main()
