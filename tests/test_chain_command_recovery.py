"""Required raid commands preserve their current ambit, including fresh land ships."""
import json
import os
from pathlib import Path
import re
import subprocess
import unittest

SOURCE = Path(__file__).with_name('test_chain.sh').read_text()
HELPER = re.search(r'(?ms)^combat_require_command\(\) \{.*?^\}', SOURCE).group()


def recover(ambit='land', fault='none'):
    fleet = {'Fleet': {'id': '9-2', 'owner': '1-2', 'commandStruct': '5-1'}}
    structure = {'Struct': {'id': '5-1', 'owner': '1-2', 'type': '1', 'locationId': '9-2',
                            'locationType': 'fleet', 'operatingAmbit': ambit}}
    if fault == 'missing': del fleet['Fleet']['commandStruct']
    if fault == 'empty': fleet['Fleet']['commandStruct'] = ''
    if fault == 'null_command': fleet['Fleet']['commandStruct'] = None
    if fault == 'wrong_fleet': fleet['Fleet']['id'] = '9-9'
    if fault == 'wrong_fleet_owner': fleet['Fleet']['owner'] = '1-9'
    if fault.startswith('struct_') and fault != 'struct_rpc':
        field = fault[7:]
        structure['Struct'][field] = 'wrong'
    script = '''set -euo pipefail
PLAYER_2_ID=1-2 PLAYER_2_FLEET_ID=9-2 RESULT=5-stale
query() {
    case "$*" in
        'query structs fleet 9-2') printf '%s' "$fixture_fleet_json"; [ "$fault" != fleet_rpc ];;
        'query structs struct 5-1') printf '%s' "$fixture_struct_json"; [ "$fault" != struct_rpc ];;
        *) return 9;;
    esac
}
combat_require_struct() {
    printf 'REQUIRE:%s\\n' "$*"
    [ "$fault" != restore ] || return 1
    printf -v "$1" '%s' 5-1
}
get_command_ship_id() {
    [ "$*" = '1-2 9-2' ] || return 9
    [ "$fault" != association_rpc ] || return 1
    if [ "$fault" = association_changed ]; then echo 5-2; else echo 5-1; fi
}
'''
    env = dict(os.environ, fault=fault, fixture_fleet_json=json.dumps(fleet), fixture_struct_json=json.dumps(structure))
    return subprocess.run(['bash', '-c', script + HELPER + '\ncombat_require_command RESULT 2 Command || exit 1\necho "RESULT:$RESULT"'],
        env=env, capture_output=True, text=True)


class CommandRecoveryTests(unittest.TestCase):
    def test_preserves_each_live_ambit_and_uses_land_for_a_missing_command(self):
        for ambit in ('land', 'water', 'air', 'space'):
            with self.subTest(ambit=ambit):
                result = recover(ambit)
                self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                self.assertIn(f'REQUIRE:RESULT 2 1 fleet 9-2 {ambit} Command', result.stdout)
                self.assertIn('RESULT:5-1', result.stdout)
        for fault in ('missing', 'empty'):
            result = recover('space', fault)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
            self.assertIn('REQUIRE:RESULT 2 1 fleet 9-2 land Command', result.stdout)

    def test_invalid_identity_or_failed_queries_cannot_trigger_duplicate_build(self):
        for fault in ('null_command', 'wrong_fleet', 'wrong_fleet_owner', 'fleet_rpc', 'struct_rpc',
                      'struct_id', 'struct_owner', 'struct_type', 'struct_locationId', 'struct_locationType'):
            with self.subTest(fault=fault):
                result = recover(fault=fault)
                self.assertNotEqual(result.returncode, 0)
                self.assertNotIn('REQUIRE:', result.stdout)
        for ambit in ('local', 'none', '', None, True, 2):
            result = recover(ambit)
            self.assertNotEqual(result.returncode, 0)
            self.assertNotIn('REQUIRE:', result.stdout)

    def test_failed_restoration_or_changed_association_refuses(self):
        for fault in ('restore', 'association_rpc', 'association_changed'):
            with self.subTest(fault=fault):
                result = recover(fault=fault)
                self.assertNotEqual(result.returncode, 0)
                self.assertNotIn('RESULT:5-1', result.stdout)

    def test_phase_preparation_requires_both_commands_before_fleet_movement(self):
        block = SOURCE.split('# ─── Required raid command and fleet preparation ───', 1)[1].split(
            '# ─── Required raid command and fleet preparation complete ───', 1)[0]
        for fault in ('none', '2', '3', 'move2', 'move3'):
            script = '''set -euo pipefail
PLAYER_2_PLANET_ID=2-2
combat_require_command() {
    echo "REQUIRE:$2"
    [ "$fault" != "$2" ] || return 1
    printf -v "$1" '%s' "5-$2"
}
combat_move_fleet() {
    echo "MOVE:$*"
    [ "$fault" != "move$1" ]
}
'''
            result = subprocess.run(['bash', '-c', script + block + '\n[ "$PLAYER_3_CMD_SHIP_ID" = 5-3 ]'],
                env=dict(os.environ, fault=fault), capture_output=True, text=True)
            self.assertEqual(result.returncode == 0, fault == 'none', result.stdout + result.stderr)
            expected = ['REQUIRE:2', 'REQUIRE:3', 'MOVE:2 2-2', 'MOVE:3 2-2']
            length = {'none': 4, '2': 1, '3': 2, 'move2': 3, 'move3': 4}[fault]
            self.assertEqual(result.stdout.splitlines(), expected[:length])


if __name__ == '__main__':
    unittest.main()
