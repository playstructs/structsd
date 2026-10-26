"""Require RG1's defender counter to kill the attacker before any volley lands."""
import argparse
import json

from agreement_terms import identity
from economic_assertions import confirmed, records
from struct_inventory import positive_integer


def verify(tx, txhash, attacker, target, defender, target_health):
    height = confirmed(tx, txhash)
    for value in (attacker, target, defender):
        identity(value, '5')
    if len({attacker, target, defender}) != 3:
        raise ValueError('counter fixture requires three distinct structs')
    positive_integer(target_health, 2**64-1, 'target health')
    attacks = records(tx, 'EventAttack', 'eventAttackDetail')
    if len(attacks) != 1:
        raise ValueError('one confirmed attack is required')
    attack = attacks[0]
    expected = {'attackerStructId': attacker, 'attackerStructTypeId': '3', 'weaponSystem': 'primaryWeapon',
                'attackerHealthBefore': '1', 'attackerHealthAfter': '0',
                'planetaryDefenseCannonDamageToAttacker': False, 'recoilDamageToAttacker': False}
    require_fields(attack, expected)
    shots = attack.get('eventAttackShotDetail')
    if not isinstance(shots, list) or len(shots) != 1 or not isinstance(shots[0], dict):
        raise ValueError('one counter-killed target result is required')
    shot = shots[0]
    require_fields(shot, {'targetStructId': target, 'targetStructTypeId': '3', 'targetStructOperatingAmbit': 'space',
        'targetHealthBefore': target_health, 'targetHealthAfter': target_health,
        'targetDestroyed': False, 'damage': '0', 'damageDealt': '0', 'blocked': False,
        'blockerDestroyed': False, 'evaded': False, 'evadedByPlanetaryDefenses': False, 'targetCountered': False,
        'postDestructionDamageToAttacker': False})
    counters = shot.get('eventAttackDefenderCounterDetail')
    if not isinstance(counters, list) or len(counters) != 1 or not isinstance(counters[0], dict):
        raise ValueError('one identified defender counter is required')
    require_fields(counters[0], {'counterByStructId': defender, 'counterByStructOperatingAmbit': 'space',
        'counterDestroyedAttacker': True})
    positive_integer(counters[0].get('counterDamage'), 2**64-1, 'defender counter damage')
    return {'verified': True, 'height': str(height)}


def require_fields(actual, expected):
    if any(type(actual.get(k)) is not type(v) or actual[k] != v for k, v in expected.items()):
        raise ValueError('counter outcome differs from the regression fixture')


if __name__ == '__main__':
    import sys
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('txhash', 'attacker', 'target', 'defender', 'target_health'):
        parser.add_argument(name)
    args = parser.parse_args()
    try:
        print(json.dumps(verify(json.load(sys.stdin), args.txhash, args.attacker, args.target, args.defender, args.target_health)))
    except (ValueError, KeyError, TypeError) as error:
        parser.exit(1, f'Invalid defender counter kill: {error}\n')
