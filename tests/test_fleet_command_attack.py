"""Fleet combat resumes must prove damage and lifecycle state, including last-hit recall."""
import copy
import json
import unittest
from fleet_command_attack import after, attack, before, catalog
from test_chain_evasion import HASH
from test_chain_retire_fighter import fixture


def command_case(ah, th):
    start, tx, end = fixture(ah, min(th, 2))
    start['target']['Struct']['type'] = '1'
    start['target']['structAttributes']['health'] = str(th)
    end = copy.deepcopy(start)
    remaining = max(th - 2, 0)
    end['target']['structAttributes'].update(health=str(remaining), isDestroyed=remaining == 0, isOnline=remaining != 0)
    attacker_remaining = max(ah - (2 if remaining else 0), 0)
    end['attacker']['structAttributes'].update(health=str(attacker_remaining), isDestroyed=attacker_remaining == 0, isOnline=attacker_remaining != 0)
    row = json.loads(tx['events'][0]['attributes'][0]['value'])
    row['eventAttackShotDetail'][0].update(targetStructTypeId='1', targetHealthBefore=str(th),
        targetHealthAfter=str(remaining), targetDestroyed=remaining == 0)
    row['attackerHealthAfter'] = str(attacker_remaining)
    row['eventAttackShotDetail'][0].update(targetCountered=remaining != 0, targetCounteredDamage='2' if remaining else '0',
        targetCounterDestroyedAttacker=attacker_remaining == 0)
    tx['events'][0]['attributes'][0]['value'] = json.dumps(row)
    return start, tx, end


class FleetCommandTests(unittest.TestCase):
    def test_resumed_hulls_and_last_hit_have_exact_receipt_and_historical_health(self):
        for ah in range(1, 7):
            for th in range(1, 7):
                start, tx, end = command_case(ah, th)
                outcome = attack({'before': start, 'tx': tx}, HASH)
                self.assertEqual(outcome['targetHealth'], str(max(th - 2, 0)))
                self.assertEqual(after({'before': start, 'after': end, 'outcome': outcome}), {'verified': True})
                end['target']['structAttributes']['health'] = str(th)
                with self.assertRaises(ValueError):
                    after({'before': start, 'after': end, 'outcome': outcome})

    def test_invalid_receipts_and_unavailable_commands_fail(self):
        for key, value in [('targetHealthAfter', '1'), ('damage', '1'), ('damageDealt', '1'),
                           ('targetStructId', '5-99'), ('targetDestroyed', False), ('evaded', True),
                           ('targetCountered', True), ('eventAttackDefenderCounterDetail', [{}])]:
            start, tx, _ = command_case(1, 2)
            row = json.loads(tx['events'][0]['attributes'][0]['value'])
            row['eventAttackShotDetail'][0][key] = value
            tx['events'][0]['attributes'][0]['value'] = json.dumps(row)
            with self.subTest(key=key), self.assertRaises(ValueError):
                attack({'before': start, 'tx': tx}, HASH)
        for key, value in [('isOnline', False), ('isDestroyed', True), ('health', '0'), ('health', '7')]:
            start, _, _ = command_case(1, 2)
            start['target']['structAttributes'][key] = value
            with self.subTest(key=key), self.assertRaises(ValueError):
                before(start, '5-1', '5-2', '1-6', '1-3', '9-6', '9-3')
        for key, value in [('code', 1), ('txhash', '0' * 64)]:
            start, tx, _ = command_case(1, 2); tx[key] = value
            with self.assertRaises(ValueError): attack({'before': start, 'tx': tx}, HASH)

    def test_catalog_rejects_changed_damage_or_defense(self):
        doc = {'StructType': {'id': '1', 'maxHealth': '6', 'primaryWeaponControl': 'guided', 'primaryWeaponCharge': '3',
            'primaryWeaponAmbits': '32', 'primaryWeaponTargets': '1', 'primaryWeaponShots': '1',
            'primaryWeaponDamage': '2', 'primaryWeaponShotSuccessRateNumerator': '1', 'primaryWeaponShotSuccessRateDenominator': '1',
            'attackCounterable': True, 'primaryWeaponCounterable': True, 'counterAttack': '2', 'counterAttackSameAmbit': '2',
            'passiveWeaponry': 'strongCounterAttack'}}
        self.assertEqual(catalog(doc), {'verified': True})
        for key in ('primaryWeaponDamage', 'primaryWeaponShotSuccessRateDenominator', 'attackReduction',
                    'guidedDefensiveSuccessRateNumerator', 'postDestructionDamage', 'counterAttack', 'counterAttackSameAmbit'):
            bad = copy.deepcopy(doc); bad['StructType'][key] = '9'
            with self.assertRaises(ValueError): catalog(bad)
