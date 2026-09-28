"""Verify required combat receipts and committed health, including random evasion outcomes."""
import argparse
import json
import sys

from combat_fixture import fields, fixture, state
from economic_assertions import confirmed, records


def profile(case):
    if case == 'P11':
        return ('9', 'land', '1', 'land', 'primaryWeapon', 'unguided', 'noUnitDefenses', True)
    if case == 'EB_B6':
        return ('4', 'space', '5', 'air', 'primaryWeapon', 'guided', 'signalJamming', False)
    if case in ('EV1', 'EV2', 'EV3'):
        return ('7', 'air', '5', 'air', 'primaryWeapon', 'guided', 'signalJamming', True)
    if case in ('EV4', 'EV5'):
        return ('1', 'space', '2', 'space', 'primaryWeapon', 'guided', 'signalJamming', True)
    if case in ('EV6', 'EV7', 'P14_1', 'P14_2', 'P14_OFFLINE'):
        return ('11', 'water', '7', 'air', 'secondaryWeapon', 'unguided', 'defensiveManeuver', False)
    if case == 'EV8':
        return ('12', 'water', '11', 'water', 'primaryWeapon', 'guided', 'signalJamming', True)
    raise ValueError('unknown evasion case')


def catalog(document, case):
    atk, _, tgt, _, weapon, control, cause, _ = profile(case)
    attacker = document['attacker']['StructType']
    target = document['target']['StructType']
    fields(attacker, {'id': atk, 'category': 'fleet', 'attackCounterable': True,
                     weapon+'Control': control, weapon+'Targets': '1', weapon+'Shots': '1',
                     weapon+'Damage': '2', weapon+'Counterable': True,
                     weapon+'ShotSuccessRateNumerator': '1', weapon+'ShotSuccessRateDenominator': '1'})
    if case == 'P11':
        fields(attacker, {'primaryWeaponAmbits': '4', 'primaryWeaponCharge': '3',
                         'maxHealth': '3', 'attackReduction': '1', 'primaryWeaponBlockable': True})
        fields(target, {'id': '1', 'category': 'fleet', 'unitDefenses': 'noUnitDefenses',
                        'maxHealth': '6', 'counterAttack': '2', 'counterAttackSameAmbit': '2',
                        'primaryWeaponAmbits': '32'})
        for row, keys in ((attacker, ('primaryWeaponRecoilDamage', 'postDestructionDamage')),
                          (target, ('secondaryWeaponAmbits', 'postDestructionDamage', 'attackReduction',
                                    'unguidedDefensiveSuccessRateNumerator'))):
            if any(row.get(key, '0') != '0' for key in keys):
                raise ValueError('catalog differs from the required counter-attack fixture')
        return {'verified': True}
    if case == 'EB_B6':
        fields(attacker, {'primaryWeaponAmbits': '24', 'primaryWeaponCharge': '5', 'maxHealth': '3'})
    fields(target, {'id': tgt, 'category': 'fleet', 'unitDefenses': cause,
                   control+'DefensiveSuccessRateNumerator': '2', control+'DefensiveSuccessRateDenominator': '3',
                   'counterAttack': '1', 'counterAttackSameAmbit': '1'})
    primary, secondary = {'5': ('8', '0'), '2': ('6', '16'), '7': ('24', '0'), '11': ('6', '8')}[tgt]
    for row, key, expected in ((attacker, weapon+'RecoilDamage', '0'), (attacker, 'postDestructionDamage', '0'),
            (target, 'postDestructionDamage', '0'), (target, 'attackReduction', '0'),
            (target, 'primaryWeaponAmbits', primary), (target, 'secondaryWeaponAmbits', secondary)):
        if row.get(key, '0') != expected:
            raise ValueError('catalog differs from the required evasion fixture')
    return {'verified': True}


def offline_command(document, command, owner, fleet):
    if fixture(document, command, owner, '1', fleet, 'space') != 'activate':
        raise ValueError('required Command Ship must be built, alive and offline')


def before(document, case, attacker, target, attacker_owner, target_owner, attacker_fleet, target_fleet, command=None):
    atk, aa, tgt, ta, _, _, _, _ = profile(case)
    if attacker == target or attacker_owner == target_owner or attacker_fleet == target_fleet:
        raise ValueError('evasion requires opposing fleets and distinct roles')
    for role, args in [('attacker', (attacker, attacker_owner, atk, attacker_fleet, aa)),
                       ('target', (target, target_owner, tgt, target_fleet, ta))]:
        if fixture(document[role], *args) != 'ready':
            raise ValueError('evasion role is unavailable')
    if document['target'].get('structDefenders', []) != []:
        raise ValueError('evasion target must have no registered defenders')
    if case == 'P11' and not (1 <= int(document['attacker']['structAttributes']['health']) <= 3
                              and 3 <= int(document['target']['structAttributes']['health']) <= 6):
        raise ValueError('counter-attack requires a surviving Command Ship and valid Tank health')
    if case == 'P14_OFFLINE':
        offline_command(document['command'], command, attacker_owner, attacker_fleet)
    return {'verified': True}


def attack(document, case, txhash):
    atk, _, tgt, ta, weapon, control, cause, counterable = profile(case)
    start = document['before']
    attacker = start['attacker']['Struct']['id']
    target = start['target']['Struct']['id']
    args = (attacker, target, start['attacker']['Struct']['owner'], start['target']['Struct']['owner'],
            start['attacker']['Struct']['locationId'], start['target']['Struct']['locationId'])
    command = start['command']['Struct']['id'] if case == 'P14_OFFLINE' else None
    before(start, case, *args, command)
    height = confirmed(document['tx'], txhash)
    rows = records(document['tx'], 'EventAttack', 'eventAttackDetail')
    if len(rows) != 1:
        raise ValueError('exactly one evasion attack event is required')
    row = rows[0]
    shots = row.get('eventAttackShotDetail')
    if not isinstance(shots, list) or len(shots) != 1 or type(shots[0].get('evaded')) is not bool:
        raise ValueError('one explicit evasion result is required')
    shot = shots[0]
    evaded = shot['evaded']
    if case == 'P11' and evaded:
        raise ValueError('Command Ship cannot evade this required attack')
    ah = int(start['attacker']['structAttributes']['health'])
    th = int(start['target']['structAttributes']['health'])
    damage = 0 if evaded else 2
    target_after = max(0, th-damage)
    counter = counterable and target_after > 0
    counter_damage = (2 if case == 'P11' else 1) if counter else 0
    attacker_after = max(0, ah-counter_damage)
    fields(row, {'attackerStructId': attacker, 'attackerStructTypeId': atk,
                 'attackerPlayerId': args[2], 'attackerStructLocationType': 'fleet',
                 'attackerStructLocationId': args[4], 'attackerStructOperatingAmbit': start['attacker']['Struct']['operatingAmbit'],
                 'attackerHealthBefore': str(ah), 'attackerHealthAfter': str(attacker_after),
                 'weaponSystem': weapon, 'weaponControl': control,
                 'planetaryDefenseCannonDamageToAttacker': False, 'recoilDamageToAttacker': False})
    fields(shot, {'targetStructId': target, 'targetStructTypeId': tgt, 'targetStructOperatingAmbit': ta,
                  'targetPlayerId': args[3], 'targetStructLocationType': 'fleet', 'targetStructLocationId': args[5],
                  'targetHealthBefore': str(th), 'targetHealthAfter': str(target_after),
                  'targetDestroyed': target_after == 0, 'damage': str(damage), 'damageDealt': str(damage),
                  'damageReduction': '0', 'blocked': False, 'blockerDestroyed': False, 'evadedByPlanetaryDefenses': False,
                  'evadedCause': cause if evaded else 'noUnitDefenses',
                  'targetCountered': counter, 'targetCounteredDamage': str(counter_damage),
                  'targetCounterDestroyedAttacker': counter and attacker_after == 0,
                  'postDestructionDamageToAttacker': False})
    if shot.get('eventAttackDefenderCounterDetail') not in (None, []):
        raise ValueError('registered defenders must not affect the evasion test')
    return {'verified': True, 'case': case, 'height': str(height), 'txhash': txhash,
            'attacker': attacker, 'target': target, 'evaded': evaded,
            'attackerHealth': str(attacker_after), 'targetHealth': str(target_after),
            'attackerDestroyed': attacker_after == 0, 'targetDestroyed': target_after == 0}


def after(document, case):
    outcome = document['outcome']
    profile(case)
    if outcome.get('case') != case or outcome.get('verified') is not True:
        raise ValueError('missing verified evasion outcome')
    for role in ('attacker', 'target'):
        previous = document['before'][role]['Struct']
        state(document['after'][role], outcome[role], previous['owner'], previous['type'],
              previous['locationId'], previous['operatingAmbit'], outcome[role+'Health'], outcome[role+'Destroyed'])
    if case == 'P14_OFFLINE':
        previous = document['before']['command']
        command = previous['Struct']
        offline_command(document['after']['command'], command['id'], command['owner'], command['locationId'])
        if document['after']['command'] != previous:
            raise ValueError('Command Ship changed during the offline attack')
    return {'verified': True}


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command', choices=('catalog', 'before', 'attack', 'after'))
    parser.add_argument('case')
    parser.add_argument('arguments', nargs='*')
    args = parser.parse_args()
    try:
        print(json.dumps(globals()[args.command](json.load(sys.stdin), args.case, *args.arguments)))
    except (ValueError, TypeError, KeyError, AttributeError) as error:
        parser.exit(1, f'Invalid evasion evidence: {error}\n')
