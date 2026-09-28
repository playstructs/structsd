"""Validate RG2 fixtures and a confirmed hit without relocated-defender support."""
import argparse
import json
import sys

from combat_fixture import fields, fixture, state
from agreement_terms import identity
from economic_assertions import confirmed, records
from struct_inventory import positive_integer


def attack(tx, txhash, attacker, target, defender, attacker_health, target_health, defender_health):
    height = confirmed(tx, txhash)
    for sid in (attacker, target, defender):
        identity(sid, '5')
    if len({attacker, target, defender}) != 3:
        raise ValueError('three distinct roles are required')
    for health in (attacker_health, target_health, defender_health):
        positive_integer(health, 2**64-1, 'fixture health')
    rows = records(tx, 'EventAttack', 'eventAttackDetail')
    if len(rows) != 1:
        raise ValueError('exactly one confirmed attack is required')
    row = rows[0]
    fields(row, {'attackerStructId': attacker, 'attackerStructTypeId': '8', 'weaponSystem': 'primaryWeapon',
                 'attackerHealthBefore': attacker_health, 'attackerHealthAfter': attacker_health,
                 'planetaryDefenseCannonDamageToAttacker': False, 'recoilDamageToAttacker': False})
    shots = row.get('eventAttackShotDetail')
    if not isinstance(shots, list) or len(shots) != 1:
        raise ValueError('exactly one target result is required')
    remaining = str(max(0, int(target_health)-2))
    fields(shots[0], {'targetStructId': target, 'targetStructTypeId': '14', 'targetStructOperatingAmbit': 'land',
                     'targetHealthBefore': target_health, 'targetHealthAfter': remaining, 'targetDestroyed': remaining == '0',
                     'damageDealt': '2', 'damageReduction': '0', 'damage': '2', 'blocked': False,
                     'blockerDestroyed': False, 'evaded': False, 'evadedByPlanetaryDefenses': False,
                     'targetCountered': False, 'postDestructionDamageToAttacker': False})
    if shots[0].get('eventAttackDefenderCounterDetail') not in (None, []):
        raise ValueError('relocated defender must not counter')
    return {'verified': True, 'height': str(height), 'health': remaining, 'destroyed': remaining == '0'}


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest='command', required=True)
    for mode in ('fixture', 'state'):
        child = sub.add_parser(mode)
        for name in ('sid', 'owner', 'kind', 'location', 'ambit'):
            child.add_argument(name)
        if mode == 'state':
            child.add_argument('health')
            child.add_argument('destroyed', choices=('true', 'false'))
    child = sub.add_parser('attack')
    for name in ('txhash', 'attacker', 'target', 'defender', 'attacker_health', 'target_health', 'defender_health'):
        child.add_argument(name)
    args = vars(parser.parse_args())
    mode = args.pop('command')
    if mode == 'state':
        args['destroyed'] = args['destroyed'] == 'true'
    try:
        result = globals()[mode](json.load(sys.stdin), **args)
        print(result if isinstance(result, str) else json.dumps(result))
    except (ValueError, TypeError, KeyError, AttributeError) as error:
        parser.exit(1, f'Invalid relocated-defender evidence: {error}\n')
