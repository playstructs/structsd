"""Reject skipped or misattributed RG1 outcomes through its real shell checks."""
import json
from pathlib import Path
import re
import subprocess
import unittest

from defender_counter_kill import verify

ROOT = Path(__file__).parent
SOURCE = (ROOT / 'test_chain.sh').read_text()
HASH = 'A' * 64


def delivered():
    counter = {'counterByStructId': '5-3', 'counterByStructOperatingAmbit': 'space',
               'counterDestroyedAttacker': True, 'counterDamage': '1'}
    shot = {'targetStructId': '5-2', 'targetStructTypeId': '3', 'targetStructOperatingAmbit': 'space', 'targetHealthBefore': '3',
            'targetHealthAfter': '3', 'targetDestroyed': False, 'damage': '0', 'damageDealt': '0', 'blocked': False,
            'blockerDestroyed': False, 'evaded': False, 'evadedByPlanetaryDefenses': False,
            'targetCountered': False, 'postDestructionDamageToAttacker': False,
            'eventAttackDefenderCounterDetail': [counter]}
    attack = {'attackerStructId': '5-1', 'attackerStructTypeId': '3', 'weaponSystem': 'primaryWeapon',
              'attackerHealthBefore': '1', 'attackerHealthAfter': '0',
              'planetaryDefenseCannonDamageToAttacker': False, 'recoilDamageToAttacker': False,
              'eventAttackShotDetail': [shot]}
    return {'txhash': HASH, 'height': '42', 'code': 0, 'events': [{'type': 'structs.structs.EventAttack',
            'attributes': [{'key': 'eventAttackDetail', 'value': json.dumps(attack)}]}]}


class DefenderCounterKillTests(unittest.TestCase):
    def test_requires_identified_counter_death_without_block_or_target_damage(self):
        self.assertEqual(verify(delivered(), HASH, '5-1', '5-2', '5-3', '3'), {'verified': True, 'height': '42'})
        for level, key, value in [('attack', 'attackerHealthAfter', '1'), ('attack', 'attackerHealthBefore', '2'),
                ('attack', 'attackerStructTypeId', '2'), ('attack', 'recoilDamageToAttacker', True),
                ('attack', 'planetaryDefenseCannonDamageToAttacker', True), ('shot', 'blocked', True),
                ('shot', 'damage', '1'), ('shot', 'damageDealt', '2'), ('shot', 'blockerDestroyed', True),
                ('shot', 'evaded', True), ('shot', 'targetHealthAfter', '2'), ('shot', 'targetCountered', True),
                ('shot', 'targetStructOperatingAmbit', 'land'), ('shot', 'targetStructTypeId', '2'), ('counter', 'counterByStructId', '5-4'),
                ('counter', 'counterDestroyedAttacker', False), ('counter', 'counterDestroyedAttacker', 1),
                ('counter', 'counterDamage', '0'), ('counter', 'counterByStructOperatingAmbit', 'air')]:
            tx = delivered()
            attack = json.loads(tx['events'][0]['attributes'][0]['value'])
            shot = attack['eventAttackShotDetail'][0]
            {'attack': attack, 'shot': shot, 'counter': shot['eventAttackDefenderCounterDetail'][0]}[level][key] = value
            tx['events'][0]['attributes'][0]['value'] = json.dumps(attack)
            with self.subTest(level=level, key=key), self.assertRaises(ValueError):
                verify(tx, HASH, '5-1', '5-2', '5-3', '3')
        for tx in ({}, dict(delivered(), code=1), dict(delivered(), height='0'), dict(delivered(), txhash='B'*64),
                dict(delivered(), events=[]), dict(delivered(), events=delivered()['events']*2)):
            with self.subTest(tx=tx), self.assertRaises(ValueError):
                verify(tx, HASH, '5-1', '5-2', '5-3', '3')



if __name__ == '__main__':
    unittest.main()
