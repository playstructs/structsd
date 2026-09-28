"""Verify command-ship attacks in the fleet queue lifecycle, including resumed hulls."""
import argparse
import json
import sys

from combat_fixture import fields, fixture, state
from economic_assertions import confirmed, records


def before(document, attacker, target, owner, target_owner, fleet, target_fleet):
    if attacker == target or owner == target_owner or fleet == target_fleet:
        raise ValueError('fleet combat needs opposing commands')
    for role, sid, player, location in (('attacker', attacker, owner, fleet), ('target', target, target_owner, target_fleet)):
        row = document[role]
        ambit = row['Struct']['operatingAmbit']
        if ambit not in ('land', 'water', 'air', 'space') or fixture(row, sid, player, '1', location, ambit) != 'ready':
            raise ValueError('command is not ready')
        if not 1 <= int(row['structAttributes']['health']) <= 6 or row.get('structDefenders', []) != []:
            raise ValueError('command health or defense differs from the fleet fixture')
    if document['attacker']['Struct']['operatingAmbit'] != document['target']['Struct']['operatingAmbit']:
        raise ValueError('command weapons require the same ambit')
    return {'verified': True}


def catalog(document):
    row = document['StructType']
    fields(row, {'id': '1', 'maxHealth': '6', 'primaryWeaponControl': 'guided', 'primaryWeaponCharge': '3',
                 'primaryWeaponAmbits': '32', 'primaryWeaponTargets': '1', 'primaryWeaponShots': '1',
                 'primaryWeaponDamage': '2', 'primaryWeaponShotSuccessRateNumerator': '1',
                 'primaryWeaponShotSuccessRateDenominator': '1', 'attackCounterable': True,
                 'primaryWeaponCounterable': True, 'counterAttack': '2', 'counterAttackSameAmbit': '2',
                 'passiveWeaponry': 'strongCounterAttack'})
    for key in ('primaryWeaponRecoilDamage', 'postDestructionDamage', 'attackReduction',
                'guidedDefensiveSuccessRateNumerator'):
        if row.get(key, '0') != '0':
            raise ValueError('unexpected command defense or additional damage')
    return {'verified': True}


def attack(document, txhash):
    start = document['before']; a = start['attacker']['Struct']; t = start['target']['Struct']
    before(start, a['id'], t['id'], a['owner'], t['owner'], a['locationId'], t['locationId'])
    ah = start['attacker']['structAttributes']['health']; th = start['target']['structAttributes']['health']
    remaining = str(max(int(th) - 2, 0))
    countered = remaining != '0'
    attacker_remaining = str(max(int(ah) - (2 if countered else 0), 0))
    height = confirmed(document['tx'], txhash)
    rows = records(document['tx'], 'EventAttack', 'eventAttackDetail')
    if len(rows) != 1:
        raise ValueError('one command attack is required')
    row = rows[0]; shots = row.get('eventAttackShotDetail')
    if not isinstance(shots, list) or len(shots) != 1:
        raise ValueError('one command projectile is required')
    fields(row, {'attackerStructId': a['id'], 'attackerStructTypeId': '1', 'attackerPlayerId': a['owner'],
                 'attackerStructLocationType': 'fleet', 'attackerStructLocationId': a['locationId'],
                 'attackerStructOperatingAmbit': a['operatingAmbit'], 'attackerHealthBefore': ah, 'attackerHealthAfter': attacker_remaining,
                 'weaponSystem': 'primaryWeapon', 'weaponControl': 'guided', 'recoilDamageToAttacker': False,
                 'planetaryDefenseCannonDamageToAttacker': False})
    fields(shots[0], {'targetStructId': t['id'], 'targetStructTypeId': '1', 'targetPlayerId': t['owner'],
                     'targetStructLocationType': 'fleet', 'targetStructLocationId': t['locationId'],
                     'targetStructOperatingAmbit': t['operatingAmbit'], 'targetHealthBefore': th,
                     'targetHealthAfter': remaining, 'targetDestroyed': remaining == '0', 'damage': '2',
                     'damageDealt': '2', 'damageReduction': '0', 'blocked': False, 'evaded': False,
                     'evadedByPlanetaryDefenses': False, 'targetCountered': countered, 'targetCounteredDamage': '2' if countered else '0',
                     'targetCounterDestroyedAttacker': attacker_remaining == '0', 'postDestructionDamageToAttacker': False})
    if shots[0].get('eventAttackDefenderCounterDetail') not in (None, []):
        raise ValueError('unexpected command defender')
    return {'verified': True, 'height': str(height), 'txhash': txhash, 'attacker': a['id'], 'target': t['id'],
            'attackerHealth': attacker_remaining, 'attackerDestroyed': attacker_remaining == '0',
            'targetHealth': remaining, 'targetDestroyed': remaining == '0'}


def after(document):
    outcome = document['outcome']
    if outcome.get('verified') is not True:
        raise ValueError('missing command attack outcome')
    for role in ('attacker', 'target'):
        old = document['before'][role]['Struct']
        if old['id'] != outcome[role]:
            raise ValueError('command identity changed')
        state(document['after'][role], old['id'], old['owner'], old['type'], old['locationId'], old['operatingAmbit'],
              outcome[role + 'Health'], outcome[role + 'Destroyed'])
    return {'verified': True}


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command', choices=('before', 'catalog', 'attack', 'after'))
    parser.add_argument('arguments', nargs='*')
    args = parser.parse_args()
    try:
        print(json.dumps(globals()[args.command](json.load(sys.stdin), *args.arguments)))
    except (ValueError, TypeError, KeyError, AttributeError) as error:
        parser.exit(1, f'Invalid fleet combat evidence: {error}\n')
