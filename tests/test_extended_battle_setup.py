"""Exercise resumed adversary setup against already allocated, populated state."""
import json
from pathlib import Path
import re
import subprocess
import tempfile
import unittest

from extended_battle_setup import feeder_plan

ROOT = Path(__file__).parent
SOURCE = (ROOT / 'test_chain.sh').read_text()
BLOCK = SOURCE.split('# ─── Extended feeder recovery ───', 1)[1].split('# ─── Extended feeder recovery complete ───', 1)[0]


def fixture(capacity=76800000, load=30400000, power=30400000):
    return {'player': {'Player': {'id': '1-6'}, 'gridAttributes': {
        'capacity': str(capacity), 'load': str(load), 'connectionCapacity': '999999999', 'structsLoad': '9000000'}},
        'allocation': {'Allocation': {'id': '6-6', 'sourceObjectId': '1-6', 'controller': '1-1',
                                      'type': 'dynamic', 'destinationId': '4-1'},
                       'gridAttributes': {'power': str(power)}}}


class ExtendedBattleSetupTests(unittest.TestCase):
    def test_replaces_own_power_and_preserves_other_load_and_build_reserve(self):
        for capacity, load, old, want in [(76800000, 30400000, 30400000, 68800000),
                                          (76800000, 40400000, 30400000, 58800000),
                                          (38400000, 30400000, 30400000, 30400000),
                                          (2**64-1, 2**53+1, 2**53, 2**64-8000002)]:
            with self.subTest(capacity=capacity, load=load, old=old):
                result = feeder_plan(fixture(capacity, load, old), '1-6', '1-1', '4-1', '8000000')
                self.assertEqual(result['power'], str(want))
                self.assertEqual(result['needsUpdate'], want != old)
        document = fixture(38400000, 0, 0)
        document['allocation'] = None
        self.assertEqual(feeder_plan(document, '1-6', '1-1', '4-1', '8000000')['power'], '30400000')

    def test_refuses_corrupt_or_unaffordable_state(self):
        mutations = [lambda d: d['player']['Player'].update(id='1-7'),
                     lambda d: d['allocation']['Allocation'].update(sourceObjectId='1-7'),
                     lambda d: d['allocation']['Allocation'].update(controller='1-7'),
                     lambda d: d['allocation']['Allocation'].update(destinationId='4-2'),
                     lambda d: d['allocation']['Allocation'].update(type='automated'),
                     lambda d: d['allocation']['gridAttributes'].update(power='30400001'),
                     lambda d: d['player']['gridAttributes'].update(capacity='8000000'),
                     lambda d: d['player']['gridAttributes'].update(load=str(2**64)),
                     lambda d: d['player']['gridAttributes'].update(capacity=True)]
        for mutate in mutations:
            document = fixture(); mutate(document)
            with self.assertRaises((ValueError, TypeError, KeyError)):
                feeder_plan(document, '1-6', '1-1', '4-1', '8000000')

    def invoke(self, state, lookup_code=0):
        with tempfile.TemporaryDirectory() as directory:
            log = Path(directory) / 'calls'
            script = '''set -euo pipefail
GENERATOR_TEST_DIR=$1 CALLS=$2 state=$3 LOOKUP_CODE=$4
PLAYER_6_ID=1-6 PLAYER_1_ID=1-1 SUBSTATION_ID=4-1 GUILD_ID=0-1 REACTOR_ID=3-1 PLAYER_6_ADDRESS=test
CURRENT_POWER=30400000 CURRENT_DEST=4-1 CURRENT_GUILD=0-1 CURRENT_PLANET=2-6 EXISTS=true CAPACITY=76800000
if [ "$state" = fresh ]; then
    CURRENT_POWER=0 CURRENT_DEST='' CURRENT_GUILD='' CURRENT_PLANET='' EXISTS=false CAPACITY=38400000
elif [ "$state" = interrupted ]; then CURRENT_DEST=''; fi
get_block_height() { echo 50; }
get_fixture_feeder() {
    [ "$LOOKUP_CODE" = 0 ] || return "$LOOKUP_CODE"
    if [ "$EXISTS" = true ] && [ "$2" = "$CURRENT_DEST" ]; then echo 6-6; fi
}
query() {
    if [ "$2" = structs ] && [ "$3" = player ]; then
        jq -nc --arg capacity "$CAPACITY" --arg load "$CURRENT_POWER" --arg guild "$CURRENT_GUILD" --arg planet "$CURRENT_PLANET" \
            '{Player:{id:"1-6",guildId:$guild,planetId:$planet,fleetId:"9-6"},gridAttributes:{capacity:$capacity,load:$load}}'
    elif [ "$2" = structs ] && [ "$3" = allocation ]; then
        jq -nc --arg power "$CURRENT_POWER" --arg destination "$CURRENT_DEST" \
            '{Allocation:{id:"6-6",sourceObjectId:"1-6",controller:"1-1",type:"dynamic",destinationId:$destination},gridAttributes:{power:$power}}'
    else return 23; fi
}
run_tx() {
    echo "$*" >> "$CALLS"
    case "$*" in
      *allocation-create*|*allocation-update*) CURRENT_POWER=$P6_ALLOC_POWER; EXISTS=true ;;
      *guild-membership-join*) CURRENT_GUILD=$GUILD_ID ;;
      *substation-allocation-connect*) CURRENT_DEST=$SUBSTATION_ID ;;
      *planet-explore*) CURRENT_PLANET=2-6 ;;
      *) return 24 ;;
    esac
}
get_created_grid_id() { echo 6-6; }
assert_eq() { [ "$2" = "$3" ]; }
assert_not_empty() { [ -n "$2" ]; }
jqr() { printf '%s' "$1" | jq -er "$2"; }
combat_require_command() { printf -v "$1" '%s' 5-6; echo 'command-restored' >> "$CALLS"; }
combat_move_fleet() { [ "$*" = '6 2-6 player_6' ]; echo 'fleet-home' >> "$CALLS"; }
''' + BLOCK
            result = subprocess.run(['bash', '-c', script, 'setup', str(ROOT.resolve()), str(log), state, str(lookup_code)], capture_output=True, text=True)
            return result, log.read_text().splitlines() if log.exists() else []

    def test_actual_setup_resizes_existing_feeder_without_rejoining_or_reexploring(self):
        result, calls = self.invoke('resumed')
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertEqual(len(calls), 3, calls)
        self.assertIn('allocation-update 6-6 68800000 --from player_6', calls[0])
        self.assertEqual(calls[1:], ['command-restored', 'fleet-home'])

    def test_actual_setup_handles_fresh_and_interrupted_connections(self):
        for state, operations in [('fresh', ['allocation-create', 'guild-membership-join', 'substation-allocation-connect', 'planet-explore']),
                                  ('interrupted', ['allocation-update', 'substation-allocation-connect'])]:
            result, calls = self.invoke(state)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
            self.assertEqual(len(calls), len(operations)+2, calls)
            for operation, call in zip(operations, calls): self.assertIn(operation, call)

    def test_failed_scoped_lookup_stops_before_any_mutation(self):
        result, calls = self.invoke('resumed', 9)
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(calls, [])

    def test_all_required_roles_prepare_before_any_completion(self):
        function = re.search(r'(?ms)^extended_battle_roles\(\) \{.*?^\}', SOURCE).group()
        script = '''set -euo pipefail
PLAYER_3_FLEET_ID=9-3 PLAYER_6_FLEET_ID=9-6 PLAYER_6_PLANET_ID=2-6
combat_require_struct() { printf '%s|%s|%s|%s|%s|%s\n' "$1" "$2" "$3" "$4" "$5" "${10}"; }
''' + function + '\nextended_battle_roles prepare\nextended_battle_roles ready\n'
        result = subprocess.run(['bash', '-c', script], capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stderr)
        rows = [line.split('|') for line in result.stdout.splitlines()]
        self.assertEqual(len(rows), 24)
        self.assertEqual([row[-1] for row in rows], ['prepare']*12 + ['ready']*12)
        self.assertEqual([row[:-1] for row in rows[:12]], [row[:-1] for row in rows[12:]])
        self.assertEqual(len({row[0] for row in rows[:12]}), 12)
        self.assertEqual({(row[1], row[2], row[3], row[4]) for row in rows[:12]},
                         {('3', '5', 'fleet', '9-3'), ('3', '8', 'fleet', '9-3'),
                          *{('6', kind, 'fleet', '9-6') for kind in ('3', '4', '8', '12', '2', '9', '11', '7')},
                          ('6', '19', 'planet', '2-6'), ('6', '14', 'planet', '2-6')})


if __name__ == '__main__':
    unittest.main()
