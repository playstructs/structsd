"""Verify a confirmed Tank killing blow against the sole planetary cannon."""
import argparse
import json
import sys

from economic_assertions import confirmed, records
from agreement_terms import identity
from struct_inventory import positive_integer


def verify(tx, txhash, attacker, target, health):
    height = confirmed(tx, txhash)
    identity(attacker, '5')
    identity(target, '5')
    positive_integer(health, 2**64 - 1, 'attacker health')
    details = records(tx, 'EventAttack', 'eventAttackDetail')
    if len(details) != 1:
        raise ValueError('one attack event is required')
    attack = details[0]
    expected = {'attackerStructId': attacker, 'attackerStructTypeId': '9', 'weaponSystem': 'primaryWeapon',
                'attackerHealthBefore': health, 'attackerHealthAfter': health,
                'planetaryDefenseCannonDamageToAttacker': False, 'planetaryDefenseCannonDamage': '0',
                'planetaryDefenseCannonDamageDestroyedAttacker': False}
    for key, value in expected.items():
        if type(attack.get(key)) is not type(value) or attack[key] != value:
            raise ValueError('attacker or cannon outcome differs from killing-blow fixture')
    shots = attack.get('eventAttackShotDetail')
    if not isinstance(shots, list) or len(shots) != 1 or not isinstance(shots[0], dict):
        raise ValueError('one target result is required')
    expected = {'targetStructId': target, 'targetStructTypeId': '19', 'targetHealthBefore': '2',
                'targetHealthAfter': '0', 'targetDestroyed': True, 'damageDealt': '2',
                'blocked': False, 'evaded': False}
    for key, value in expected.items():
        if type(shots[0].get(key)) is not type(value) or shots[0][key] != value:
            raise ValueError('attack must destroy the two-health cannon directly')
    return {'verified': True, 'height': str(height), 'txhash': tx['txhash'], 'attacker': attacker, 'target': target}


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('txhash')
    parser.add_argument('attacker')
    parser.add_argument('target')
    parser.add_argument('health')
    args = parser.parse_args()
    try:
        print(json.dumps(verify(json.load(sys.stdin), args.txhash, args.attacker, args.target, args.health)))
    except (ValueError, KeyError, TypeError) as error:
        parser.exit(1, f'Invalid cannon killing blow: {error}\n')
