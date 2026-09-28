"""Certify cannon damage, immunity and destruction against pinned chain state."""
import argparse
import json
import sys

from agreement_terms import identity
from combat_fixture import attributes, fields, fixture, state
from economic_assertions import confirmed, records
from pdc_killing_blow import verify as killing_blow

# Attacker, target, and admissible target health before each required attack.
PROFILES = {
    'E1': ('2', '14', (6,)), 'E3': ('8', '14', (4,)),
    'E4': ('9', '8', (1, 2, 3)), 'soften': ('8', '19', (4, 6)),
    'kill': ('9', '19', (2,)), 'E10': ('2', '14', (2,)),
    'prepare_ore': ('8', '14', (1, 2, 3, 4, 5)),
    'prepare_cannon': ('8', '19', (1, 3, 5)),
}


def number(row, key):
    value = row.get(key, '0')
    if not isinstance(value, str) or not value.isascii() or not value.isdecimal() or str(int(value)) != value or int(value) > 2**64-1:
        raise ValueError('invalid planet counter')
    return int(value)


def planet(document, pid, owner):
    identity(pid, '2'); identity(owner, '1')
    fields(document['Planet'], {'id': pid, 'owner': owner})
    row = document['planetAttributes']
    if not isinstance(row, dict):
        raise ValueError('missing planet attributes')
    return number(row, 'defensiveCannonQuantity'), number(row, 'planetaryShield')


def catalog(document, mode):
    ak, tk, _ = PROFILES[mode]
    kinds = {'attacker': ak, 'target': tk}
    if mode != 'E10': kinds['cannon'] = '19'
    for role, kind in kinds.items():
        row = document[role]['StructType']
        fields(row, {'id': kind, 'category': 'planet' if kind in ('14', '19') else 'fleet',
                     'maxHealth': '6' if kind in ('14', '19') else '3'})
        if role == 'attacker':
            fields(row, {'primaryWeaponControl': 'unguided', 'primaryWeaponTargets': '1',
                'primaryWeaponShots': '1', 'primaryWeaponDamage': '2', 'primaryWeaponBlockable': True,
                'primaryWeaponCounterable': True, 'primaryWeaponCharge': '3' if kind == '9' else '5',
                'primaryWeaponAmbits': '4' if kind == '9' else '6',
                'primaryWeaponShotSuccessRateNumerator': '1', 'primaryWeaponShotSuccessRateDenominator': '1'})
            if row.get('attackCounterable', False) is not (kind != '8') or row.get('primaryWeaponArmourPiercing', False) is not (kind == '2'):
                raise ValueError('unexpected attacker counterability or piercing')
            if row.get('primaryWeaponRecoilDamage', '0') != '0': raise ValueError('unexpected recoil')
        else:
            for key in ('counterAttack', 'counterAttackSameAmbit', 'attackReduction', 'postDestructionDamage',
                        'guidedDefensiveSuccessRateNumerator', 'unguidedDefensiveSuccessRateNumerator'):
                if row.get(key, '0') != '0': raise ValueError('target must have no damage or evasion of its own')
        if kind in ('14', '19'):
            fields(row, {'buildLimit': '1'})
            if row.get('planetaryShieldContribution', '0') != ('13' if kind == '19' else '0'):
                raise ValueError('unexpected shield contribution')
    return {'verified': True}


def ready(document, sid, owner, kind, location):
    ambit = 'space' if kind == '2' else 'land'
    fields(document['Struct'], {'locationType': 'planet' if kind in ('14', '19') else 'fleet'})
    if fixture(document, sid, owner, kind, location, ambit) != 'ready': raise ValueError('cannon fixture is unavailable')
    attrs, hp = attributes(document, sid, owner, kind, location, ambit)
    if int(hp) > (6 if kind in ('14', '19') else 3): raise ValueError('invalid fixture health')
    if attrs.get('isHidden', False) is not False: raise ValueError('fixture must be revealed')


def before(document, mode, attacker, target, cannon, ao, to, af, tf, pid):
    ak, tk, health = PROFILES[mode]
    if attacker == target or ao == to or (tk == '8' and af == tf): raise ValueError('opposing cannon roles required')
    ready(document['attacker'], attacker, ao, ak, af)
    ready(document['target'], target, to, tk, pid if tk in ('14', '19') else tf)
    if int(document['target']['structAttributes']['health']) not in health: raise ValueError('wrong target health for sequence')
    if document['target'].get('structDefenders', []) != []: raise ValueError('cannon target may not be defended')
    count, shield = planet(document['planet'], pid, to)
    if count != (0 if mode == 'E10' else 1): raise ValueError('wrong surviving cannon count')
    if mode != 'E10':
        ready(document['cannon'], cannon, to, '19', pid)
        if shield < 13 or cannon == attacker: raise ValueError('invalid active cannon')
        if tk == '19':
            if cannon != target or document['cannon'] != document['target']: raise ValueError('target must be this cannon')
        elif cannon == target: raise ValueError('cannon may not replace the target')
    return {'verified': True}


def check_before(document, mode):
    a, t = (document[r]['Struct'] for r in ('attacker', 'target'))
    return before(document, mode, a['id'], t['id'], '' if mode == 'E10' else document['cannon']['Struct']['id'],
        a['owner'], t['owner'], a['locationId'], t['locationId'] if t['locationType'] == 'fleet' else '',
        document['planet']['Planet']['id'])


def attack(document, mode, txhash):
    start = document['before']; check_before(start, mode)
    height = confirmed(document['tx'], txhash)
    rows = records(document['tx'], 'EventAttack', 'eventAttackDetail')
    if len(rows) != 1: raise ValueError('one cannon attack required')
    row = rows[0]; shots = row.get('eventAttackShotDetail')
    if not isinstance(shots, list) or len(shots) != 1: raise ValueError('one projectile required')
    a, t = (start[r]['Struct'] for r in ('attacker', 'target'))
    ah, th = (int(start[r]['structAttributes']['health']) for r in ('attacker', 'target'))
    damage = 1 if mode == 'E1' else 0
    ah_after, th_after = ah-damage, max(0, th-2)
    fields(row, {'attackerStructId': a['id'], 'attackerStructTypeId': a['type'], 'attackerPlayerId': a['owner'],
        'attackerStructLocationType': 'fleet', 'attackerStructLocationId': a['locationId'], 'attackerStructOperatingAmbit': a['operatingAmbit'],
        'attackerHealthBefore': str(ah), 'attackerHealthAfter': str(ah_after), 'weaponSystem': 'primaryWeapon', 'weaponControl': 'unguided',
        'planetaryDefenseCannonDamageToAttacker': damage != 0, 'planetaryDefenseCannonDamage': str(damage),
        'planetaryDefenseCannonDamageDestroyedAttacker': ah_after == 0, 'recoilDamageToAttacker': False})
    shot = shots[0]
    fields(shot, {'targetStructId': t['id'], 'targetStructTypeId': t['type'], 'targetPlayerId': t['owner'],
        'targetStructLocationType': t['locationType'], 'targetStructLocationId': t['locationId'], 'targetStructOperatingAmbit': 'land',
        'targetHealthBefore': str(th), 'targetHealthAfter': str(th_after), 'targetDestroyed': th_after == 0,
        'damage': '2', 'damageDealt': '2', 'damageReduction': '0', 'armourPiercing': False,
        'blocked': False, 'blockerDestroyed': False, 'evaded': False, 'evadedByPlanetaryDefenses': False,
        'targetCountered': False, 'targetCounteredDamage': '0', 'targetCounterDestroyedAttacker': False,
        'postDestructionDamageToAttacker': False})
    if shot.get('blockedByStructId', '') != '' or shot.get('eventAttackDefenderCounterDetail') not in (None, []):
        raise ValueError('unexpected defender')
    if mode == 'kill': killing_blow(document['tx'], txhash, a['id'], t['id'], str(ah))
    hp = {'attacker': str(ah_after), 'target': str(th_after)}
    count, shield = planet(start['planet'], start['planet']['Planet']['id'], t['owner'])
    if mode != 'E10': hp['cannon'] = str(th_after) if t['type'] == '19' else start['cannon']['structAttributes']['health']
    if t['type'] == '19' and th_after == 0: count -= 1; shield -= 13
    return {'verified': True, 'mode': mode, 'height': str(height), 'txhash': txhash, 'health': hp,
        'roles': {r: start[r]['Struct']['id'] for r in hp}, 'cannonCount': str(count), 'shield': str(shield)}


def after(document):
    o = document['outcome']; start = document['before']; end = document['after']
    if o.get('verified') is not True: raise ValueError('missing cannon outcome')
    check_before(start, o['mode'])
    expected_roles = {'attacker', 'target'} if o['mode'] == 'E10' else {'attacker', 'target', 'cannon'}
    if set(o['health']) != expected_roles or set(o['roles']) != expected_roles: raise ValueError('missing cannon role')
    for role, hp in o['health'].items():
        old = start[role]['Struct']
        if old['id'] != o['roles'][role]: raise ValueError('cannon role changed')
        fields(end[role]['Struct'], {'locationType': old['locationType']})
        state(end[role], old['id'], old['owner'], old['type'], old['locationId'], old['operatingAmbit'], hp, hp == '0')
        if hp == '0' and end[role]['structAttributes'].get('isOnline', False) is not False:
            raise ValueError('destroyed cannon fixture remains online')
    p = start['planet']['Planet']
    count, shield = planet(end['planet'], p['id'], p['owner'])
    if (str(count), str(shield)) != (o['cannonCount'], o['shield']): raise ValueError('planet disagrees with cannon outcome')
    return {'verified': True}


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command', choices=('catalog', 'before', 'attack', 'after'))
    parser.add_argument('arguments', nargs='*'); args = parser.parse_args()
    try: print(json.dumps(globals()[args.command](json.load(sys.stdin), *args.arguments)))
    except (ValueError, KeyError, TypeError, AttributeError) as error: parser.exit(1, f'Invalid cannon evidence: {error}\n')
