"""Verify one confirmed Tank hit in the field-generator destruction fixture."""
import argparse
import json
import sys

from agreement_terms import identity
from economic_assertions import confirmed, records
from struct_inventory import positive_integer


def verify(tx, txhash, attacker, target, health):
    height = confirmed(tx, txhash)
    identity(attacker, '5')
    identity(target, '5')
    before = positive_integer(health, 8, 'generator health')
    details = records(tx, 'EventAttack', 'eventAttackDetail')
    if len(details) != 1:
        raise ValueError('one generator attack event is required')
    attack = details[0]
    expected = {'attackerStructId': attacker, 'attackerStructTypeId': '9', 'weaponSystem': 'primaryWeapon'}
    if any(type(attack.get(k)) is not type(v) or attack[k] != v for k, v in expected.items()):
        raise ValueError('generator attack identity differs from fixture')
    shots = attack.get('eventAttackShotDetail')
    if not isinstance(shots, list) or len(shots) != 1 or not isinstance(shots[0], dict):
        raise ValueError('one generator target is required')
    expected = {'targetStructId': target, 'targetStructTypeId': '20', 'targetHealthBefore': health,
                'targetHealthAfter': str(before - 1), 'targetDestroyed': before == 1,
                'damageDealt': '2', 'damageReduction': '1', 'damage': '1', 'blocked': False, 'evaded': False}
    if any(type(shots[0].get(k)) is not type(v) or shots[0][k] != v for k, v in expected.items()):
        raise ValueError('generator hit differs from the one-damage armour fixture')
    return {'height': str(height), 'health': str(before - 1), 'destroyed': before == 1}


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
        parser.exit(1, f'Invalid generator attack: {error}\n')
