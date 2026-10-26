"""Run the rotation balance assertions with exact SDK-sized amounts."""
from pathlib import Path
import re
import subprocess
import unittest

SOURCE = (Path(__file__).parent / 'test_chain.sh').read_text()
ASSERTIONS = '\n'.join(re.search(rf'(?ms)^{name}\(\) \{{.*?^\}}', SOURCE).group()
                       for name in ('assert_eq', 'assert_ge'))
BLOCK = SOURCE.split('# ─── rotation exact balance accounting ───', 1)[1].split(
    '# ─── rotation exact balance accounting complete ───', 1)[0]


class RotationBalanceTests(unittest.TestCase):
    def invoke(self, vault, primary, after, old='0', fault=''):
        script = '''set -euo pipefail
GREEN='' RED='' NC='' PASS_COUNT=0 FAIL_COUNT=0
ROTATION_VAULT=vault ROTATION_ADDRESS=primary
ROTATION_VAULT_BALANCE="$1" ROTATION_PRIMARY_BALANCE="$2"
AFTER="$3" OLD="$4" FAULT="$5"
get_balance() {
    [ "$2" = ualpha ]
    case "$1" in
        vault) echo "$AFTER"; [ "$FAULT" != vault_rpc ] ;;
        primary) echo "$OLD"; [ "$FAULT" != primary_rpc ] ;;
        *) return 9 ;;
    esac
}
'''
        return subprocess.run(['bash', '-c', script + ASSERTIONS + '\n' + BLOCK +
                               '\n[ "$FAIL_COUNT" -eq 0 ]', 'test',
                               *map(str, (vault, primary, after, old, fault))],
                              capture_output=True, text=True)

    def test_rotation_preserves_exact_sum_and_allows_settled_rewards(self):
        for base in (500000, 2**53+1, 2**63-1, 2**64+7):
            for reward in (0, 1, 9007199254740993):
                with self.subTest(base=base, reward=reward):
                    result = self.invoke(base, base+1, base*2+1+reward)
                    self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
            for after, old in ((base*2, 0), (base*2+1, 1)):
                result = self.invoke(base, base+1, after, old)
                self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_rotation_rejects_invalid_amounts_and_failed_queries(self):
        for invalid in ('', 'null', '-1', '1.5', ' 1', '01'):
            for values in ((invalid, 10, 20), (10, invalid, 20), (10, 10, invalid)):
                result = self.invoke(*values)
                self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
        for fault in ('vault_rpc', 'primary_rpc'):
            result = self.invoke(10, 10, 20, fault=fault)
            self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_shared_comparison_preserves_negative_headroom_refusal(self):
        script = "set -euo pipefail\nGREEN='' RED='' NC='' PASS_COUNT=0 FAIL_COUNT=0\n"
        script += ASSERTIONS + '\nassert_ge headroom 1000 -1\n[ "$FAIL_COUNT" -eq 1 ]\n'
        result = subprocess.run(['bash', '-c', script], capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_revoked_spend_uses_its_own_snapshot_after_settled_rewards(self):
        block = SOURCE.split('# ─── rotation revoked spend accounting ───', 1)[1].split(
            '# ─── rotation revoked spend accounting complete ───', 1)[0]
        for base in (100, 2**64+7):
            for before, after, succeeds in ((base, base, True), (base+5, base+5, True),
                                             (base-1, base-1, False),
                                             (base+5, base, False), (base, base+1, False)):
                script = '''set -euo pipefail
GREEN='' RED='' NC='' PASS_COUNT=0 FAIL_COUNT=0
ROTATION_VAULT=vault ROTATION_PLAYER=1-3 PLAYER_2_ID=1-2 ROTATION_SECOND_KEY=revoked
ROTATION_VAULT_BALANCE="$1" BALANCE="$2" AFTER="$3"
get_balance() { [ "$*" = 'vault ualpha' ]; echo "$BALANCE"; }
run_tx_expect_fail_noauto() {
    shift
    [ "$*" = 'tx structs vault-send 1-3 1-2 1ualpha --from revoked' ]
    BALANCE="$AFTER"
}
'''
                result = subprocess.run(['bash', '-c', script + ASSERTIONS + '\n' + block +
                                         '\n[ "$FAIL_COUNT" -eq 0 ]', 'test',
                                         *map(str, (base, before, after))], capture_output=True, text=True)
                self.assertEqual(result.returncode == 0, succeeds, result.stdout + result.stderr)


if __name__ == '__main__':
    unittest.main()
