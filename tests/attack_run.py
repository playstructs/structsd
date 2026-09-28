"""Verify the required attack-run matrix from catalog, receipts and pinned state."""
import argparse
import json
import sys

from combat_fixture import fields, fixture, state
from economic_assertions import confirmed, records


CASES = {
    'A1': (2, 1, 6, '1', []), 'A2': (3, 1, 6, '1', []), 'A3': (6, 1, 2, '1', []),
    'B1': (2, 2, 6, '1', ['2']), 'B2': (2, 3, 6, '1', ['8']), 'B3': (3, 2, 6, '1', ['12']),
    'C1': (2, 1, 6, '1', ['13']), 'C2': (6, 1, 2, '1', ['10']),
    'D1': (2, 2, 6, '1', ['2', '13']), 'D2': (2, 3, 6, '1', ['10', '13']),
    'D3': (3, 1, 6, '1', ['2', '10']), 'D4': (2, 1, 6, '1', ['2', '10', '13']),
    'E1': (2, 2, 6, '1', []), 'E2': (2, 3, 6, '3', []), 'E3': (6, 1, 2, '3', []),
}
AMBITS = {'1': 'space', '2': 'space', '3': 'space', '8': 'land', '10': 'land', '12': 'water', '13': 'water'}
# Primary/secondary reach, counter damage, same-ambit damage, maximum health.
TYPES = {'1': ('32', '0', '2', '2', '6'), '2': ('6', '16', '1', '1', '3'),
         '3': ('16', '16', '1', '1', '3'), '8': ('6', '0', '0', '0', '3'),
         '10': ('24', '0', '1', '1', '3'), '12': ('10', '0', '1', '2', '3'),
         '13': ('18', '0', '1', '1', '3')}


def profile(case):
    if case not in CASES:
        raise ValueError('unknown attack-run case')
    ap, number, tp, target, defenders = CASES[case]
    minimum = 3 if case.startswith('D') else 2 if case in ('B1', 'C1', 'C2') else 1
    return {'attackerPlayer': ap, 'attackerRole': f'AR_P{ap}_SF{number}_ID', 'attackerMinimumHealth': minimum, 'targetPlayer': tp,
            'targetRole': f'AR_P{tp}_CS_ID' if target == '1' else f'AR_P{tp}_SF1_ID',
            'targetType': target, 'visitor': tp if ap == 6 else ap,
            'defenders': [{'type': t, 'ambit': AMBITS[t], 'role': f'AR_P{tp}_DEF{t}_ID'} for t in defenders]}


def catalog(document, case):
    plan = profile(case)
    expected = ['3', plan['targetType']] + [d['type'] for d in plan['defenders']]
    rows = [document['attacker'], document['target']] + document['defenders']
    if len(rows) != len(expected):
        raise ValueError('catalog is missing a required role')
    for wrapped, kind in zip(rows, expected):
        row = wrapped['StructType']
        primary, secondary, counter, same, health = TYPES[kind]
        fields(row, {'id': kind, 'category': 'fleet', 'maxHealth': health, 'canDefend': True})
        for key, value in [('primaryWeaponAmbits', primary), ('secondaryWeaponAmbits', secondary),
                           ('counterAttack', counter), ('counterAttackSameAmbit', same),
                           ('attackReduction', '0'), ('postDestructionDamage', '0'),
                           ('unguidedDefensiveSuccessRateNumerator', '0')]:
            if row.get(key, '0') != value:
                raise ValueError('attack-run catalog reach, defense or damage differs')
    fields(rows[0]['StructType'], {'attackCounterable': True, 'secondaryWeapon': 'attackRun',
           'secondaryWeaponControl': 'unguided', 'secondaryWeaponTargets': '1', 'secondaryWeaponShots': '3',
           'secondaryWeaponDamage': '1', 'secondaryWeaponGuaranteedShots': '1',
           'secondaryWeaponShotSuccessRateNumerator': '1', 'secondaryWeaponShotSuccessRateDenominator': '3',
           'secondaryWeaponBlockable': True, 'secondaryWeaponCounterable': True})
    attacker = rows[0]['StructType']
    if attacker.get('secondaryWeaponArmourPiercing', False) is not False or attacker.get('secondaryWeaponRecoilDamage', '0') != '0':
        raise ValueError('unexpected attack-run piercing or recoil')
    return {'verified': True}


def before(document, case, attacker, target, attacker_owner, target_owner, attacker_fleet, target_fleet):
    plan = profile(case)
    if attacker == target or attacker_owner == target_owner or attacker_fleet == target_fleet:
        raise ValueError('attack-run requires opposing fleets')
    rows = [document['attacker'], document['target']] + document['defenders']
    if len(document['defenders']) != len(plan['defenders']):
        raise ValueError('required defender is absent')
    identities = [(attacker, attacker_owner, '3', attacker_fleet, 'space'),
                  (target, target_owner, plan['targetType'], target_fleet, 'space')]
    identities += [(row['Struct']['id'], target_owner, role['type'], target_fleet, role['ambit'])
                   for row, role in zip(document['defenders'], plan['defenders'])]
    if len({args[0] for args in identities}) != len(identities):
        raise ValueError('combat roles must be distinct')
    for row, args in zip(rows, identities):
        if fixture(row, *args) != 'ready' or int(row['structAttributes']['health']) > int(TYPES[args[2]][4]):
            raise ValueError('attack-run role is unavailable or has impossible health')
    if int(document['attacker']['structAttributes']['health']) < plan['attackerMinimumHealth']:
        raise ValueError('attacker lacks the health needed to exercise the required counter/block path')
    defender_ids = [row['Struct']['id'] for row in document['defenders']]
    if document['target'].get('structDefenders', []) != sorted(defender_ids):
        raise ValueError('target registrations differ from the complete required defense')
    for row in document['defenders']:
        if row['structAttributes'].get('protectedStructIndex') != target.split('-')[1]:
            raise ValueError('defender is registered to a different target')
    return {'verified': True}


def attack(document, case, txhash):
    start = document['before']
    a, t = start['attacker']['Struct'], start['target']['Struct']
    before(start, case, a['id'], t['id'], a['owner'], t['owner'], a['locationId'], t['locationId'])
    height = confirmed(document['tx'], txhash)
    rows = records(document['tx'], 'EventAttack', 'eventAttackDetail')
    if len(rows) != 1:
        raise ValueError('exactly one attack-run event is required')
    row = rows[0]
    ah, th = (int(start[role]['structAttributes']['health']) for role in ('attacker', 'target'))
    remaining = ah
    counters = []
    # The store iterates defender id bytes, not numeric struct indices.
    for defender in sorted(start['defenders'], key=lambda d: d['Struct']['id']):
        d = defender['Struct']
        if d['type'] in ('2', '10', '13') and remaining:
            remaining = max(0, remaining-1)
            counters.append({'counterByStructId': d['id'], 'counterByStructTypeId': d['type'],
                 'counterByStructLocationType': 'fleet', 'counterByStructLocationId': d['locationId'],
                 'counterByStructOperatingAmbit': d['operatingAmbit'], 'counterDamage': '1',
                 'counterDestroyedAttacker': remaining == 0})
    blocker = next((d for d in start['defenders'] if d['Struct']['type'] == '2'), None) if remaining else None
    shots = row.get('eventAttackShotDetail')
    if not isinstance(shots, list) or len(shots) != (3 if remaining else 1):
        raise ValueError('wrong projectile count or volley after lethal defender counters')
    damage = 0
    if remaining:
        for index, shot in enumerate(shots):
            if shot.get('damageDealt') not in ('0', '1') or (index == 0 and shot['damageDealt'] != '1'):
                raise ValueError('attack-run must have a guaranteed first hit and binary later hits')
            damage += int(shot['damageDealt'])
    target_after = th if blocker is not None else max(0, th-damage)
    target_counter = remaining > 0 and target_after > 0
    counter_damage = int(TYPES[t['type']][2]) if target_counter else 0
    attacker_after = max(0, remaining-counter_damage)
    fields(row, {'attackerStructId': a['id'], 'attackerStructTypeId': '3', 'attackerPlayerId': a['owner'],
                 'attackerStructLocationType': 'fleet', 'attackerStructLocationId': a['locationId'],
                 'attackerStructOperatingAmbit': 'space', 'attackerHealthBefore': str(ah),
                 'attackerHealthAfter': str(attacker_after), 'weaponSystem': 'secondaryWeapon', 'weaponControl': 'unguided',
                 'planetaryDefenseCannonDamageToAttacker': False, 'recoilDamageToAttacker': False})
    outcomes = {a['id']: {'health': str(attacker_after), 'destroyed': attacker_after == 0},
                t['id']: {'health': str(target_after), 'destroyed': target_after == 0}}
    for d in start['defenders']:
        hp = int(d['structAttributes']['health'])
        if d is blocker:
            hp = max(0, hp-damage)
        outcomes[d['Struct']['id']] = {'health': str(hp), 'destroyed': hp == 0}
    for index, shot in enumerate(shots):
        last = index == len(shots)-1
        fields(shot, {'targetStructId': t['id'], 'targetStructTypeId': t['type'], 'targetPlayerId': t['owner'],
            'targetStructLocationType': 'fleet', 'targetStructLocationId': t['locationId'], 'targetStructOperatingAmbit': 'space',
            'targetHealthBefore': str(th), 'targetHealthAfter': str(target_after if last else th),
            'targetDestroyed': last and target_after == 0, 'damage': str(damage if last else 0),
            'damageReduction': '0', 'armourPiercing': False, 'blocked': blocker is not None,
            'evaded': False, 'evadedByPlanetaryDefenses': False, 'evadedCause': 'noUnitDefenses',
            'targetCountered': last and target_counter, 'targetCounteredDamage': str(counter_damage if last else 0),
            'targetCounterDestroyedAttacker': last and target_counter and attacker_after == 0,
            'postDestructionDamageToAttacker': False,
            'blockerDestroyed': last and blocker is not None and outcomes[blocker['Struct']['id']]['destroyed']})
        if not remaining and shot.get('damageDealt') != '0':
            raise ValueError('lethal defender counter must prevent projectile damage')
        actual = shot.get('eventAttackDefenderCounterDetail')
        actual = [] if actual is None else actual
        expected = counters if last else []
        if not isinstance(actual, list) or len(actual) != len(expected):
            raise ValueError('defender counter inventory differs')
        for found, wanted in zip(actual, expected):
            fields(found, wanted)
        if blocker is not None:
            b = blocker['Struct']; bh = blocker['structAttributes']['health']
            fields(shot, {'blockedByStructId': b['id'], 'blockedByStructTypeId': b['type'],
                 'blockedByStructLocationType': 'fleet', 'blockedByStructLocationId': b['locationId'],
                 'blockedByStructOperatingAmbit': 'space', 'blockerHealthBefore': bh,
                 'blockerHealthAfter': outcomes[b['id']]['health'] if last else bh})
        elif shot.get('blockedByStructId', '') != '':
            raise ValueError('unblocked shot names a blocker')
    return {'verified': True, 'case': case, 'height': str(height), 'txhash': txhash,
            'attacker': a['id'], 'target': t['id'], 'defenders': [d['Struct']['id'] for d in start['defenders']],
            'damage': damage, 'blocked': blocker is not None, 'defenderCounters': len(counters), 'outcomes': outcomes}


def after(document, case):
    outcome = document['outcome']
    profile(case)
    if outcome.get('case') != case or outcome.get('verified') is not True:
        raise ValueError('missing verified attack-run outcome')
    previous = [document['before']['attacker'], document['before']['target']] + document['before']['defenders']
    current = [document['after']['attacker'], document['after']['target']] + document['after']['defenders']
    if len(current) != len(previous) or set(outcome['outcomes']) != {d['Struct']['id'] for d in previous}:
        raise ValueError('post-attack state inventory differs')
    for old, new in zip(previous, current):
        row = old['Struct']; expected = outcome['outcomes'][row['id']]
        state(new, row['id'], row['owner'], row['type'], row['locationId'], row['operatingAmbit'],
              expected['health'], expected['destroyed'])
    return {'verified': True}


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command', choices=('profile', 'catalog', 'before', 'attack', 'after'))
    parser.add_argument('case')
    parser.add_argument('arguments', nargs='*')
    args = parser.parse_args()
    try:
        result = profile(args.case) if args.command == 'profile' else globals()[args.command](json.load(sys.stdin), args.case, *args.arguments)
        print(json.dumps(result))
    except (ValueError, TypeError, KeyError, AttributeError) as error:
        parser.exit(1, f'Invalid attack-run evidence: {error}\n')
