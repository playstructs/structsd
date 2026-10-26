"""Phase 11 must restore resumable roles and prove an actual Command Ship counter."""
import copy
import json
import subprocess
import unittest

from evasion_attack import after, attack, before, catalog
from test_chain_evasion import HASH, SOURCE, receipt, start, types

PHASE = SOURCE.split('section "PHASE 11: Counter-Attack"', 1)[1].split('fi # phase 11', 1)[0]


class CounterAttackTests(unittest.TestCase):
    def test_counter_bypasses_tank_armor_and_may_destroy_wounded_tank(self):
        catalog(types('P11'), 'P11')
        for ah in (1, 2, 3):
            for th in (3, 4, 5, 6):
                with self.subTest(attacker=ah, target=th):
                    initial = start('P11', ah, th)
                    outcome = attack({'before': initial, 'tx': receipt('P11', False, ah, th)}, 'P11', HASH)
                    self.assertEqual(outcome['attackerHealth'], str(max(0, ah-2)))
                    self.assertEqual(outcome['targetHealth'], str(th-2))
                    current = copy.deepcopy(initial)
                    for role in ('attacker', 'target'):
                        current[role]['structAttributes'].update(health=outcome[role+'Health'],
                            isDestroyed=outcome[role+'Destroyed'], isOnline=not outcome[role+'Destroyed'])
                    self.assertEqual(after({'before': initial, 'after': current, 'outcome': outcome}, 'P11'), {'verified': True})
                    current['target']['structAttributes']['health'] = str(th)
                    with self.assertRaises(ValueError):
                        after({'before': initial, 'after': current, 'outcome': outcome}, 'P11')

    def test_refuses_evasion_missing_counter_and_wrong_receipt(self):
        for key, value in [('evaded', True), ('targetCountered', False), ('targetCounteredDamage', '1'),
                           ('targetCounteredDamage', '0'), ('targetHealthAfter', '6'),
                           ('targetStructId', '5-9'), ('damageDealt', '1'), ('blocked', True)]:
            tx = receipt('P11', False, 3, 6)
            row = json.loads(tx['events'][0]['attributes'][0]['value'])
            row['eventAttackShotDetail'][0][key] = value
            tx['events'][0]['attributes'][0]['value'] = json.dumps(row)
            with self.subTest(field=key), self.assertRaises(ValueError):
                attack({'before': start('P11', 3, 6), 'tx': tx}, 'P11', HASH)
        for key, value in [('code', 7), ('txhash', 'B'*64), ('height', '0'), ('events', [])]:
            tx = receipt('P11', False, 3, 6); tx[key] = value
            with self.subTest(field=key), self.assertRaises(ValueError):
                attack({'before': start('P11', 3, 6), 'tx': tx}, 'P11', HASH)

    def test_refuses_wounded_command_and_catalog_drift(self):
        for health in (1, 2, 7):
            with self.subTest(health=health), self.assertRaises(ValueError):
                before(start('P11', 3, health), 'P11', '5-1', '5-2', '1-6', '1-3', '9-6', '9-3')
        for role, key, value in [('target','counterAttackSameAmbit','1'), ('target','counterAttack','1'),
                ('target','unguidedDefensiveSuccessRateNumerator','1'), ('target','attackReduction','1'),
                ('target','postDestructionDamage','1'), ('target','primaryWeaponAmbits','16'),
                ('attacker','primaryWeaponAmbits','16'), ('attacker','primaryWeaponDamage','1'),
                ('attacker','attackReduction','0'), ('attacker','primaryWeaponRecoilDamage','1')]:
            doc = types('P11'); doc[role]['StructType'][key] = value
            with self.subTest(field=key), self.assertRaises(ValueError): catalog(doc, 'P11')

    def run_phase(self, health='6', ambit='land', fault='none'):
        script = r'''set -euo pipefail
health="$1" ambit="$2" fault="$3"
PLAYER_2_ID=1-2 PLAYER_3_ID=1-3 PLAYER_2_PLANET_ID=2-1 CHARGE_BUILD=8 CHARGE_MOVE=3
combat_require_command() {
 echo "restore $*"
 [ "$fault" != restore ] || return 7
 printf -v "$1" '%s' 5-2
 if [[ "$*" = *replacement* ]]; then health=6; ambit=land; fi
}
query() {
 printf '{"Struct":{"operatingAmbit":"%s"},"structAttributes":{"health":"%s"}}\n' "$ambit" "$health"
 [ "$fault" != query ]
}
wait_for_charge() { echo "charge $*"; [ "$fault" != charge ]; }
run_tx() { echo "tx $*"; [ "$fault" != transaction ]; }
combat_move_fleet() { echo "fleet $*"; [ "$fault" != fleet ]; }
ev_attack() { echo "attack $*"; [ "$fault" != attack ]; }
'''
        return subprocess.run(['bash', '-c', script+PHASE, 'test', health, ambit, fault],
                              text=True, capture_output=True, timeout=10)

    def test_phase_restores_commands_and_moves_only_when_needed(self):
        for ambit in ('land', 'water', 'air', 'space'):
            result = self.run_phase(ambit=ambit)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual('struct-move' in result.stdout, ambit != 'land')
            self.assertNotIn('struct-trash', result.stdout)
            self.assertIn('restore PLAYER_2_CMD_SHIP_ID 2 P11 defender Command Ship', result.stdout)
            self.assertIn('restore COMMAND_SHIP_ID 3 P11 target Command Ship', result.stdout)
            self.assertTrue(result.stdout.endswith('fleet 2 2-1\nfleet 3 2-1\nattack P11 DEFENDER_STRUCT_ID 2 9 land COMMAND_SHIP_ID 3 1 land primaryWeapon 2-1\n'))
        self.assertNotIn('struct-build-compute', PHASE)
        self.assertNotIn('SKIP', PHASE)

    def test_phase_replaces_only_command_that_cannot_survive_volley(self):
        for health in ('1', '2', '3', '4', '5', '6'):
            result = self.run_phase(health=health, ambit='space')
            self.assertEqual(result.returncode, 0, result.stderr)
            replace = int(health) <= 2
            self.assertEqual('struct-trash' in result.stdout, replace)
            self.assertEqual('replacement Command Ship' in result.stdout, replace)
            self.assertEqual('struct-move' in result.stdout, not replace)

    def test_phase_propagates_preparation_and_receipt_failures(self):
        for fault in ('restore', 'query', 'charge', 'transaction', 'fleet', 'attack'):
            with self.subTest(fault=fault):
                result = self.run_phase(ambit='space', fault=fault)
                self.assertNotEqual(result.returncode, 0)
                if fault != 'attack': self.assertNotIn('attack P11', result.stdout)
        for health, ambit in [('0','land'), ('7','land'), ('01','land'), ('x','land'), ('6','unknown')]:
            result = self.run_phase(health=health, ambit=ambit)
            self.assertNotEqual(result.returncode, 0)
            self.assertNotIn('attack P11', result.stdout)


if __name__ == '__main__':
    unittest.main()
