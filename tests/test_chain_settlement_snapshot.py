"""Changing provider pools must be read at one explicit committed height."""
from pathlib import Path
import re
import subprocess
import tempfile
import unittest

from settlement_open_snapshot import opening_snapshot

ROOT = Path(__file__).resolve().parent
SOURCE = (ROOT/'test_chain.sh').read_text()
HELPER = re.search(r'(?ms)^get_balance_at_height\(\) \{.*?^\}', SOURCE).group()
A = SOURCE.split('# ─── opening A snapshot conservation ───', 1)[1].split('# ─── opening A snapshot conservation complete ───', 1)[0]
B = SOURCE.split('# ─── opening B snapshot conservation ───', 1)[1].split('# ─── opening B snapshot conservation complete ───', 1)[0]


class SettlementSnapshotTests(unittest.TestCase):
    def test_exact_deposits_and_remaining_half_holdback(self):
        for start in (100, 2**53+1, 2**63-200):
            terms = [{'start': str(start), 'duration': '120', 'capacity': '2000'},
                     {'start': str(start+11), 'duration': '120', 'capacity': '3000'}]
            # B has already earned revenue, so its full 360,000 deposit need
            # not remain in collateral. Both remaining obligations still must.
            result = opening_snapshot(str(start+110), '341500', '258500', terms)
            self.assertEqual(result, {'principal': '600000', 'minimum_collateral': '341500', 'verified': True})
            self.assertTrue(opening_snapshot(str(start+110), '600000', '0', terms)['verified'], 'proof lag legitimately leaves more collateral')
            for collateral, earnings in [('341499', '258501'), ('341500', '258499'), ('341501', '258500')]:
                with self.subTest(start=start, collateral=collateral, earnings=earnings), self.assertRaises(ValueError):
                    opening_snapshot(str(start+110), collateral, earnings, terms)
        odd = [{'start': '1', 'duration': '5', 'capacity': '3'}]
        self.assertEqual(opening_snapshot('2', '14', '1', odd)['minimum_collateral'], '14')
        with self.assertRaises(ValueError):
            opening_snapshot('2', '13', '2', odd)
        huge = [{'start': '1', 'duration': str(2**63), 'capacity': str(2**64-1)}]
        principal = 2**63*(2**64-1)
        self.assertTrue(opening_snapshot('2', str(principal-(2**64-1)//2), str((2**64-1)//2), huge)['verified'])

    def test_invalid_or_expired_snapshot_is_not_successful_coverage(self):
        terms = [{'start': '100', 'duration': '120', 'capacity': '2000'}]
        for height in ('0', '99', '220', '221', str(2**63)):
            with self.subTest(height=height), self.assertRaises(ValueError):
                opening_snapshot(height, '240000', '0', terms)
        for bad in ('', '00', '-1', '1.5', 1, False, str(2**256)):
            for i in range(3):
                values = ['110', '230000', '10000']
                values[i] = bad
                with self.subTest(bad=bad, i=i), self.assertRaises(ValueError):
                    opening_snapshot(*values, terms)
        for field, value in [('start', '0'), ('duration', '0'), ('capacity', '0'), ('capacity', str(2**64))]:
            row = dict(terms[0], **{field: value})
            with self.subTest(field=field, value=value), self.assertRaises(ValueError):
                opening_snapshot('110', '240000', '0', [row])
        for rows in (None, [], terms*3, [None], [{}]):
            with self.subTest(rows=rows), self.assertRaises((ValueError, KeyError, TypeError)):
                opening_snapshot('110', '240000', '0', rows)

    def test_actual_height_helper_has_no_latest_fallback_in_conditional_callers(self):
        for conditional in (False, True):
            for height, fault in [('110', 'none'), (str(2**53+1), 'none'), (str(2**63-1), 'none'),
                                  ('110', 'rpc'), ('110', 'missing'), ('110', 'wrong_denom'),
                                  ('110', 'leading_zero'), ('0', 'none'), ('01', 'none'),
                                  (str(2**63), 'none')]:
                script = '''set -euo pipefail
height="$1" fault="$2"
query() {
  [ "$*" = "query bank balance pool ualpha --height $height" ] || return 9
  case "$fault" in
    missing) echo '{}';;
    wrong_denom) echo '{"balance":{"denom":"other","amount":"0"}}';;
    leading_zero) echo '{"balance":{"denom":"ualpha","amount":"01"}}';;
    *) echo '{"balance":{"denom":"ualpha","amount":"18446744073709551616"}}';;
  esac
  [ "$fault" != rpc ]
}
'''+HELPER+'\n'
                call = 'get_balance_at_height pool ualpha "$height"'
                script += ('if output=$('+call+'); then printf "%s" "$output"; else exit 7; fi\n') if conditional else call+'\n'
                result = subprocess.run(['bash', '-c', script, 'fixture', height, fault], capture_output=True, text=True)
                with self.subTest(conditional=conditional, height=height, fault=fault):
                    if fault == 'none' and height in ('110', str(2**53+1), str(2**63-1)):
                        self.assertEqual(result.returncode, 0, result.stderr)
                        self.assertEqual(result.stdout.strip(), '18446744073709551616')
                    else:
                        self.assertNotEqual(result.returncode, 0)
                        self.assertEqual(result.stdout, '')

    def test_actual_opening_blocks_survive_payment_between_queries_but_reject_loss(self):
        for fault in ('none', 'drop_height', 'rpc', 'wrong_total', 'overpaid', 'expired'):
            with self.subTest(fault=fault), tempfile.TemporaryDirectory() as directory:
                directory = Path(directory)
                script = '''set -euo pipefail
cd "$1"
GENERATOR_TEST_DIR="$2" fault="$3"
SETTLE_COLL_ADDR=collateral SETTLE_EARN_ADDR=earnings
SETTLE_A_RECORD='{"startBlock":"100"}' SETTLE_B_RECORD='{"startBlock":"111"}'
SETTLE_DUR_A=120 SETTLE_DUR_B=120 SETTLE_CAP_A=2000 SETTLE_CAP_B=3000
SETTLE_COLL_A=240000 SETTLE_COLL_B=360000
info() { :; }
assert_eq() { [ "$2" = "$3" ]; }
get_block_height() {
  if [ ! -f height ]; then echo 110 > height; else echo 210 > height; fi
  if [ "$fault" = expired ]; then echo 220 > height; fi
  echo 0 > reads
  cat height
}
query() {
  [ "$1 $2 $3" = 'query bank balance' ] && [ "$5" = ualpha ] || return 9
  local base amount n
  base=$(cat height)
  if [ "$#" = 7 ]; then
    [ "$6" = --height ] && [ "$7" = "$base" ] || return 9
  elif [ "$#" = 5 ]; then
    # A simulated worker transfers another 1,000 between unpinned RPCs.
    n=$(cat reads); echo $((n+1)) > reads
  else return 9; fi
  if [ "$base" = 110 ]; then
    if [ "$4" = collateral ]; then amount=230000; else amount=10000; fi
  else
    if [ "$4" = collateral ]; then amount=341500; else amount=258500; fi
  fi
  if [ "$#" = 5 ] && [ "${n:-0}" -gt 0 ]; then amount=$((amount+1000)); fi
  if [ "$fault" = wrong_total ]; then amount=$((amount+1)); fi
  if [ "$fault" = overpaid ]; then
    if [ "$4" = collateral ]; then amount=$((amount-1)); else amount=$((amount+1)); fi
  fi
  printf '{"balance":{"denom":"ualpha","amount":"%s"}}' "$amount"
  [ "$fault" != rpc ]
}
'''
                helper = HELPER.replace(' --height "${height}"', '') if fault == 'drop_height' else HELPER
                script += helper+'\n'+A+'\n'+B+'\necho completed\n'
                result = subprocess.run(['bash', '-c', script, 'fixture', str(directory), str(ROOT), fault], capture_output=True, text=True)
                if fault == 'none':
                    self.assertEqual(result.returncode, 0, result.stderr)
                    self.assertIn('completed', result.stdout)
                else:
                    self.assertNotEqual(result.returncode, 0, result.stderr)
                    self.assertNotIn('completed', result.stdout)


if __name__ == '__main__':
    unittest.main()
