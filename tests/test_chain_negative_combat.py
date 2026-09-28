"""Negative battle coverage must reach the intended targeting guard."""
from pathlib import Path
import re
import subprocess
import tempfile
import unittest

SOURCE = (Path(__file__).parent / 'test_chain.sh').read_text()

def function(name):
    return re.search(r'(?ms)^' + name + r'\(\) \{.*?^\}', SOURCE).group()

HELPERS = '\n'.join(function(name) for name in ('assert_eq', 'run_tx_expect_reject_matching', 'eb_attack_should_fail'))
BLOCK = SOURCE.split('# ─── Required negative targeting before destructive combat ───', 1)[1].split('# ─── Required negative targeting complete ───', 1)[0]


class NegativeCombatTests(unittest.TestCase):
    def test_intended_reason_and_both_health_invariants(self):
        for expected in ('hidden', 'out_of_range'):
            for actual in (expected, 'permission denied', 'out of gas', 'panic ' + expected,
                           'destroyed', 'unreachable', 'out_of_range' if expected == 'hidden' else 'hidden'):
                for mutation in ('none', 'attacker', 'target') if actual == expected else ('none',):
                    with tempfile.TemporaryDirectory() as directory:
                        script = '''set -euo pipefail
GREEN='' RED='' NC='' BOLD='' PASS_COUNT=0 FAIL_COUNT=0 PARAMS_TX=''
PLAYER_3_ID=1-3
expected="$1" actual="$2" mutation="$3" marker="$4"
info() { :; }
_tx_outcome() { printf '%s' "$1"; }
_tx_outcome_known() { return 0; }
structsd() {
  [ "$*" = 'tx structs struct-attack 5-1 5-2 primaryWeapon --from player_3' ] || exit 9
  touch "$marker"
  printf 'fail|target struct (5-2) is %s from attacker struct (5-1)' "$actual"
}
eb_health() {
  if [ -f "$marker" ] && { [ "$mutation:$1" = attacker:5-1 ] || [ "$mutation:$1" = target:5-2 ]; }; then echo 2; else echo 3; fi
}
eb_get_charge() { [ "$*" = '5-1 primaryWeapon' ] || exit 9; echo 5; }
wait_for_charge() { [ "$*" = '1-3 5' ]; }
'''
                        result = subprocess.run(['bash', '-c', script + HELPERS + '\neb_attack_should_fail check 5-1 5-2 3 "$expected"\n[ "$FAIL_COUNT" -eq 0 ]',
                            'test', expected, actual, mutation, str(Path(directory)/'sent')], capture_output=True, text=True)
                        with self.subTest(expected=expected, actual=actual, mutation=mutation):
                            self.assertEqual(result.returncode == 0, expected == actual and mutation == 'none', result.stdout + result.stderr)

    def test_missing_fixtures_refuse_before_transaction(self):
        for missing, health in ((which, value) for which in ('5-1', '5-2') for value in ('0', '', '0\n0', 'null', '-1')):
            script = '''set -euo pipefail
FAIL_COUNT=0
missing="$1" health="$2"
eb_health() { if [ "$1" = "$missing" ]; then printf '%s\n' "$health"; else echo 3; fi; }
eb_get_charge() { echo unexpected >&2; exit 9; }
'''
            result = subprocess.run(['bash', '-c', script + function('eb_attack_should_fail') + '\neb_attack_should_fail check 5-1 5-2 3\n', 'test', missing, health], capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('Required negative attack fixture', result.stderr)
            self.assertNotIn('unexpected', result.stderr)

    def test_required_checks_precede_destructive_combat_and_use_air_capable_attacker(self):
        self.assertLess(SOURCE.index('# ─── Required negative targeting complete ───'), SOURCE.index('# GROUP A: Same-Ambit Combat'))
        self.assertNotIn('SKIP', BLOCK)
        for hidden in ('omitted', 'false', 'true', 'null', 'string', 'rpc'):
            payload = {'omitted': '{"Struct":{"id":"5-8"},"structAttributes":{}}',
                'false': '{"Struct":{"id":"5-8"},"structAttributes":{"isHidden":false}}',
                'true': '{"Struct":{"id":"5-8"},"structAttributes":{"isHidden":true}}',
                'null': '{"Struct":{"id":"5-8"},"structAttributes":{"isHidden":null}}',
                'string': '{"Struct":{"id":"5-8"},"structAttributes":{"isHidden":"false"}}',
                'rpc': '{}'}[hidden]
            script = '''set -euo pipefail
DESTROYER_STRUCT_ID=5-1 EB_P6_BATTLESHIP_ID=5-2 EB_PURSUIT_FIGHTER_ID=5-3 EB_P6_TANK_ID=5-4
EB_STARFIGHTER_ID=5-5 SAM_STRUCT_ID=5-6 EB_FRIGATE_ID=5-7 STEALTH_BOMBER_ID=5-8
PLAYER_3_ID=1-3 CHARGE_ACTIVATE=1
payload="$1" mode="$2"
info() { :; }
eb_attack_should_fail() { shift; echo "CHECK:$*"; }
query() { [ "$*" = 'query structs struct 5-8' ] || return 9; echo "$payload"; [ "$mode" != rpc ]; }
wait_for_charge() { [ "$*" = '1-3 1' ]; }
run_tx() { shift; echo "TX:$*"; }
'''
            result = subprocess.run(['bash', '-c', script + BLOCK, 'test', payload, hidden], capture_output=True, text=True)
            with self.subTest(hidden=hidden):
                if hidden in ('omitted', 'false', 'true'):
                    self.assertEqual(result.returncode, 0, result.stderr)
                    checks = [line for line in result.stdout.splitlines() if line.startswith('CHECK:')]
                    self.assertEqual(checks, ['CHECK:5-1 5-2 3', 'CHECK:5-3 5-4 3', 'CHECK:5-5 5-6 6', 'CHECK:5-7 5-8 6 hidden'])
                    self.assertEqual('TX:tx structs struct-stealth-activate' in result.stdout, hidden != 'true')
                    self.assertIn('TX:tx structs struct-stealth-deactivate', result.stdout)
                else:
                    self.assertNotEqual(result.returncode, 0)
                    self.assertNotIn('CHECK:5-7', result.stdout)
                    self.assertNotIn('TX:', result.stdout)


if __name__ == '__main__':
    unittest.main()
