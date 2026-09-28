"""A cannon killing-blow check cannot pass on a miss or a missing fixture."""
import json
import unittest

from pdc_killing_blow import verify

HASH = 'A' * 64


def delivered():
    detail = {'attackerStructId': '5-1', 'attackerStructTypeId': '9', 'weaponSystem': 'primaryWeapon',
        'attackerHealthBefore': '2', 'attackerHealthAfter': '2',
        'planetaryDefenseCannonDamageToAttacker': False, 'planetaryDefenseCannonDamage': '0',
        'planetaryDefenseCannonDamageDestroyedAttacker': False,
        'eventAttackShotDetail': [{'targetStructId': '5-2', 'targetStructTypeId': '19',
            'targetHealthBefore': '2', 'targetHealthAfter': '0', 'targetDestroyed': True,
            'damageDealt': '2', 'blocked': False, 'evaded': False}]}
    return {'code': 0, 'height': '500', 'txhash': HASH, 'events': [
        {'type': 'structs.structs.EventAttack', 'attributes': [{'key': 'eventAttackDetail', 'value': json.dumps(detail)}]}]}


class PdcKillingBlowTests(unittest.TestCase):
    def test_confirms_actual_kill_not_just_unchanged_attacker_health(self):
        self.assertTrue(verify(delivered(), HASH, '5-1', '5-2', '2')['verified'])
        for shot, field, value in ((False, 'attackerStructId', '5-3'), (False, 'attackerHealthAfter', '1'),
                (False, 'planetaryDefenseCannonDamageToAttacker', True), (False, 'planetaryDefenseCannonDamage', '1'),
                (False, 'planetaryDefenseCannonDamageToAttacker', 0),
                (True, 'targetHealthAfter', '1'), (True, 'targetHealthBefore', '0'),
                (True, 'targetDestroyed', False), (True, 'damageDealt', '0'),
                (True, 'targetStructId', '5-3'), (True, 'blocked', True), (True, 'evaded', True)):
            tx = delivered()
            attack = json.loads(tx['events'][0]['attributes'][0]['value'])
            (attack['eventAttackShotDetail'][0] if shot else attack)[field] = value
            tx['events'][0]['attributes'][0]['value'] = json.dumps(attack)
            with self.subTest(field=field, value=value), self.assertRaises(ValueError):
                verify(tx, HASH, '5-1', '5-2', '2')
        for tx in ({}, dict(delivered(), code=1), dict(delivered(), height='0'),
                dict(delivered(), txhash='B'*64), dict(delivered(), events=[]),
                dict(delivered(), events=delivered()['events']*2)):
            with self.subTest(tx=tx), self.assertRaises(ValueError):
                verify(tx, HASH, '5-1', '5-2', '2')



if __name__ == '__main__':
    unittest.main()
