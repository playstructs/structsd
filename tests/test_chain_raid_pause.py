"""Execute Phase 10's actual shell assertions with fixture/CLI transport doubles."""
import copy
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile
import unittest


SOURCE = (Path(__file__).parent / 'test_chain.sh').read_text()
HELPERS = '\n'.join(
    re.search(rf'(?ms)^{name}\(\) \{{.*?^\}}', SOURCE).group()
    for name in ('jqr', 'assert_eq', 'assert_gt', '_tx_json', '_tx_error_line',
                 '_tx_delivered', '_tx_outcome', '_tx_outcome_known',
                 'run_tx_expect_reject_matching')
)
BLOCK = SOURCE.split('# ─── Required mining/refining pause coverage ───', 1)[1].split(
    '# ─── Required mining/refining pause coverage complete ───', 1)[0]
AFTER = SOURCE.split('# ─── Post-raid: ore clocks shifted, raider-arrived cleared, mining works again ───', 1)[1].split(
    '# ─── Restore: bring P2\'s Command Ship back online for later phases ───', 1)[0]
PLANET = {
    'Planet': {'id': '3-2', 'owner': '1-2', 'locationListStart': '9-3'},
    'planetAttributes': {'blockRaiderArrived': '100', 'blockStartOreMine': '110',
                         'blockStartOreRefine': '90', 'oreMiningActiveQuantity': '1',
                         'oreRefiningActiveQuantity': '1'},
}


def run_block(planet=None, *, missing='', query_fails=False, bad_action='', reply='',
              skip=False, after=False):
    with tempfile.TemporaryDirectory() as directory:
        trace = Path(directory) / 'trace'
        trace.touch()
        environment = dict(os.environ, TRACE=str(trace), MISSING=missing,
                           QUERY_FAILS=str(query_fails), BAD_ACTION=bad_action, REPLY=reply,
                           SKIP_MINING=str(skip).lower(),
                           PLANET_JSON=planet if isinstance(planet, str) else json.dumps(PLANET if planet is None else planet))
        script = '''set -euo pipefail
RED='' GREEN='' YELLOW='' BOLD='' NC='' PARAMS_TX=''
PASS_COUNT=0 FAIL_COUNT=0 TX_CONFIRM_TIMEOUT=1
PLAYER_2_PLANET_ID=3-2 PLAYER_2_ID=1-2 PLAYER_3_FLEET_ID=9-3
MINER_STRUCT_ID='' REFINERY_STRUCT_ID=''
P2_MINE_CLOCK_BEFORE=110 P2_REFINE_CLOCK_BEFORE=90
P2_PLANET_JSON="$PLANET_JSON"
info() { :; }
sleep() { :; }
combat_require_struct() {
    printf 'FIXTURE:%s\\n' "$*" >> "$TRACE"
    [ "$1" != "$MISSING" ] || return 1
    case "$1" in
        MINER_STRUCT_ID) printf -v "$1" '%s' '5-14';;
        REFINERY_STRUCT_ID) printf -v "$1" '%s' '5-15';;
        *) return 9;;
    esac
}
query() {
    printf 'QUERY:%s\\n' "$*" >> "$TRACE"
    if [ "$*" = 'query tx ABC' ]; then
        echo '{"txhash":"ABC","height":"123","code":0}'
        return 0
    fi
    [ "$*" = 'query structs planet 3-2' ] || return 9
    printf '%s\\n' "$PLANET_JSON"
    [ "$QUERY_FAILS" = False ]
}
structsd() {
    printf 'TX:%s\\n' "$*" >> "$TRACE"
    if [ "$3" = "$BAD_ACTION" ]; then
        printf '%s\\n' "$REPLY"
        return 1
    fi
    case "$*" in
        'tx structs struct-ore-mine-compute 5-14 --from player_2') action=mining;;
        'tx structs struct-ore-refine-compute 5-15 --from player_2') action=refining;;
        *) echo 'wrong compute arguments'; return 9;;
    esac
    printf 'Error: planet (3-2) is under raid: %s is paused until the raid ends\\n' "$action"
    return 1
}
run_compute() { shift; printf 'COMPUTE:%s\\n' "$*" >> "$TRACE"; }
'''
        # Post-raid controls begin with the identities returned by restoration.
        if after:
            script += '\nMINER_STRUCT_ID=5-14 REFINERY_STRUCT_ID=5-15\n'
        script += HELPERS + '\n' + (AFTER if after else BLOCK)
        script += '\nprintf "COUNTS:%s:%s\\n" "$PASS_COUNT" "$FAIL_COUNT"\n[ "$FAIL_COUNT" -eq 0 ]\n'
        result = subprocess.run(['bash', '-c', script], env=environment,
                                capture_output=True, text=True)
        return result, trace.read_text().splitlines()


class RaidPauseTests(unittest.TestCase):
    def test_both_required_roles_and_specific_cli_refusals(self):
        result, trace = run_block()
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn('COUNTS:2:0', result.stdout)
        self.assertEqual(trace, [
            'FIXTURE:MINER_STRUCT_ID 2 14 planet 3-2 land raid-pause Ore Extractor',
            'FIXTURE:REFINERY_STRUCT_ID 2 15 planet 3-2 land raid-pause Ore Refinery',
            'QUERY:query structs planet 3-2',
            'TX:tx structs struct-ore-mine-compute 5-14 --from player_2',
            'TX:tx structs struct-ore-refine-compute 5-15 --from player_2',
        ])

    def test_unavailable_required_role_stops_before_query_or_compute(self):
        for role, calls in (('MINER_STRUCT_ID', 1), ('REFINERY_STRUCT_ID', 2)):
            with self.subTest(role=role):
                result, trace = run_block(missing=role)
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(len(trace), calls)
                self.assertTrue(all(line.startswith('FIXTURE:') for line in trace))

    def test_failed_query_with_valid_body_cannot_pass(self):
        result, trace = run_block(query_fails=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(any(line.startswith('TX:') for line in trace))

    def test_empty_malformed_or_multiple_planet_responses_refuse(self):
        for payload in ('', '{', '{}', 'null', '[]', '[{}]',
                        json.dumps(PLANET) + json.dumps(PLANET), '{}' + json.dumps(PLANET)):
            with self.subTest(payload=payload):
                result, trace = run_block(payload)
                self.assertNotEqual(result.returncode, 0)
                self.assertFalse(any(line.startswith('TX:') for line in trace))

    def test_wrong_planet_owner_raider_or_invalid_active_clock_refuses(self):
        for section, fields in PLANET.items():
            for field in fields:
                values = ('wrong', '', None, True, 1) if section == 'Planet' else (
                    '', None, True, 1, '0', '-1', '01', '1.5', '1e2',
                    '18446744073709551616', '100000000000000000000')
                for value in (*values, 'missing'):
                    with self.subTest(field=field, value=value):
                        planet = copy.deepcopy(PLANET)
                        if value == 'missing':
                            del planet[section][field]
                        else:
                            planet[section][field] = value
                        result, trace = run_block(planet)
                        self.assertNotEqual(result.returncode, 0)
                        self.assertFalse(any(line.startswith('TX:') for line in trace))

    def test_each_compute_requires_its_own_reason_and_known_failure(self):
        for action in ('mine', 'refine'):
            expected = 'mining' if action == 'mine' else 'refining'
            other = 'refining' if action == 'mine' else 'mining'
            reason = f'planet (3-2) is under raid: {expected} is paused until the raid ends'
            replies = [
                ('Error: ' + reason, True),
                (json.dumps({'code': 5, 'raw_log': reason}), True),
                ('Error: panic ' + reason, False),
                ('Error: ' + reason.replace(expected, other), False),
                ('Error: permission denied', False), ('Error: struct is destroyed', False),
                ('Error: out of gas', False), ('Error: dial tcp: connection refused', False),
                ('{}', False), ('', False),
                ('{"code":0,"txhash":"ABC"}', False),
            ]
            for reply, passes in replies:
                with self.subTest(action=action, reply=reply):
                    result, trace = run_block(bad_action=f'struct-ore-{action}-compute', reply=reply)
                    self.assertEqual(result.returncode == 0, passes, result.stdout + result.stderr)
                    self.assertEqual(sum(line.startswith('TX:') for line in trace), 2)
                    self.assertIn('COUNTS:2:0' if passes else 'COUNTS:1:1', result.stdout)

    def test_only_explicit_skip_mining_disables_coverage(self):
        for after in (False, True):
            with self.subTest(after=after):
                result, trace = run_block(skip=True, after=after, planet={}, missing='MINER_STRUCT_ID')
                self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                self.assertEqual(trace, [])
                self.assertIn('COUNTS:0:0', result.stdout)

    def test_both_clocks_must_advance_and_arrival_must_clear_after_raid(self):
        for field, bad_values in (
            ('blockStartOreMine', ('110', '109', '0')),
            ('blockStartOreRefine', ('90', '89', '0')),
            ('blockRaiderArrived', ('100', '1')),
            ('healthy', (None,)),
        ):
            for value in bad_values:
                with self.subTest(field=field, value=value):
                    planet = copy.deepcopy(PLANET)
                    planet['planetAttributes'].update(blockRaiderArrived='0',
                        blockStartOreMine='150', blockStartOreRefine='140')
                    if field != 'healthy':
                        planet['planetAttributes'][field] = value
                    result, trace = run_block(planet, after=True)
                    self.assertEqual(result.returncode == 0, field == 'healthy', result.stdout + result.stderr)
                    self.assertEqual(trace, ['COMPUTE:tx structs struct-ore-mine-compute 5-14 --from player_2'])


if __name__ == '__main__':
    unittest.main()
