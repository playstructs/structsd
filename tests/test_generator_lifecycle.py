"""Resuming a funded generator must not treat its old power as independent supply."""
import copy
import json
from pathlib import Path
import re
import subprocess
import unittest

from generator_lifecycle import MAX, verify

SOURCE = (Path(__file__).parent / 'test_chain.sh').read_text()


def fixture(fuel=0, disabled=False):
    power = 0 if disabled else fuel * 2
    return {
        'generator': {'Struct': {'id': '5-4', 'owner': '1-4', 'type': '20', 'locationId': '2-4',
                                'locationType': 'planet', 'operatingAmbit': 'land'},
                      'structAttributes': {'isBuilt': True, 'isOnline': True, 'health': '8'},
                      'gridAttributes': {'fuel': str(fuel)}},
        'player': {'Player': {'id': '1-4', 'planetId': '2-4'}, 'gridAttributes': {'capacity': str(4800000 + power)}},
        'catalog': {'StructType': {'id': '20', 'generatingRate': '2', 'maxHealth': '8', 'attackReduction': '1'}},
        'positions': {'Infusion': [] if not fuel else [{
            'destinationId': '5-4', 'destinationType': 'struct', 'address': 'vault4', 'playerId': '1-4',
            'fuel': str(fuel), 'ratio': '2', 'commission': '0.000000000000000000',
            'power': str(power), 'generatorDisabled': disabled}], 'pagination': {}},
    }


def check(document):
    return verify(document, '5-4', '1-4', 'vault4', '2-4')


class GeneratorLifecycleTests(unittest.TestCase):
    def test_fresh_resumed_and_pending_restoration_accounting(self):
        for fuel in (0, 53, 1000000):
            for disabled in (False, True) if fuel else (False,):
                with self.subTest(fuel=fuel, disabled=disabled):
                    result = check(fixture(fuel, disabled))
                    self.assertEqual(result['ready'], not disabled)
                    self.assertEqual(result['capacityAfterDestruction'], '4800000')
                    self.assertEqual(result['capacityAfterInfusion'], str(6800000 + (0 if disabled else fuel * 2)))
                    self.assertEqual(result['fuelAfterInfusion'], str(fuel + 1000000))
        document = fixture(53)
        document['positions']['Infusion'][0]['ownerId'] = '1-4'
        self.assertTrue(check(document)['ready'])

    def test_inconsistent_or_incomplete_evidence_refuses_a_quote(self):
        edits = [
            (('generator', 'Struct', 'id'), '5-9'),
            (('generator', 'Struct', 'owner'), '1-9'),
            (('generator', 'Struct', 'locationType'), 'fleet'),
            (('generator', 'structAttributes', 'isBuilt'), False),
            (('generator', 'structAttributes', 'isOnline'), False),
            (('generator', 'structAttributes', 'isDestroyed'), True),
            (('generator', 'structAttributes', 'health'), '0'),
            (('generator', 'gridAttributes', 'fuel'), '54'),
            (('player', 'Player', 'id'), '1-9'),
            (('player', 'gridAttributes', 'capacity'), '105'),
            (('player', 'gridAttributes', 'capacity'), str(MAX)),
            (('catalog', 'StructType', 'generatingRate'), '1'),
            (('catalog', 'StructType', 'attackReduction'), '0'),
            (('positions', 'pagination', 'next_key'), 'unread'),
            (('positions', 'pagination', 'nextKey'), 'unread'),
            (('positions', 'pagination'), None),
            (('positions', 'pagination'), {'next_key': '', 'nextKey': ''}),
            (('positions', 'Infusion'), []),
        ]
        for path, value in edits:
            with self.subTest(path=path, value=value):
                document = fixture(53)
                section = document
                for name in path[:-1]:
                    section = section[name]
                section[path[-1]] = value
                with self.assertRaises(ValueError):
                    check(document)
        for key, value in (('ownerId', '0-1'), ('address', 'key4'), ('power', '105'), ('fuel', '54'),
                           ('ratio', '1'), ('commission', '0.01'), ('generatorDisabled', 'false'),
                           ('defusing', '1')):
            with self.subTest(key=key, value=value):
                document = fixture(53)
                document['positions']['Infusion'][0][key] = value
                with self.assertRaises(ValueError):
                    check(document)
        document = fixture(53)
        document['positions']['Infusion'].append(copy.deepcopy(document['positions']['Infusion'][0]))
        with self.assertRaises(ValueError):
            check(document)
        document = fixture()
        del document['positions']['pagination']
        with self.assertRaises(ValueError):
            check(document)

    def test_shell_pins_every_input_and_propagates_query_failure(self):
        function = re.search(r'(?ms)^generator_fixture_snapshot\(\) \{.*?^\}', SOURCE).group()
        script = r'''set -euo pipefail
GENERATOR_STRUCT_ID=5-4 PLAYER_4_ID=1-4 fault="$1" pinned="${2:-}"
query() {
 case "$*" in
 'query structs block-height') [ -z "$pinned" ] || return 9; echo '{"blockHeight":"123"}';;
 'query structs struct 5-4 --height 123') echo '{"Struct":{}}';[ "$fault" != generator ];;
 'query structs player 1-4 --height 123') echo '{"Player":{}}';[ "$fault" != player ];;
 'query structs infusion-all-by-destination 5-4 --page-limit 2 --height 123') echo '{"Infusion":[]}';[ "$fault" != positions ];;
 'query structs struct-type 20 --height 123') echo '{"StructType":{}}';[ "$fault" != catalog ];;
 *) return 9;;
 esac
}
'''
        for pinned in ('', '123', '0', '01', '-1', str(2**63)):
            for fault in ('none', 'generator', 'player', 'positions', 'catalog'):
                with self.subTest(fault=fault, pinned=pinned):
                    result = subprocess.run(['bash', '-c', script + function + '\ngenerator_fixture_snapshot "$pinned"', 'test', fault, pinned],
                                            capture_output=True, text=True)
                    success = fault == 'none' and pinned in ('', '123')
                    self.assertEqual(result.returncode == 0, success, result.stdout + result.stderr)
                    if success:
                        self.assertEqual(json.loads(result.stdout)['height'], '123')


if __name__ == '__main__':
    unittest.main()
