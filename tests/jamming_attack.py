"""Certify unguided bypass against an actual satellite-protected planet target."""
import argparse
import json
import sys

from combat_fixture import fields, fixture, state
from pdc_attack import catalog as planetary_catalog, number, planet, ready
from economic_assertions import confirmed, records

ROLES = ('attacker', 'target', 'satellite')
NETWORK = 'lowOrbitBallisticsInterceptorNetwork'


def catalog(document):
    # The same Battleship primary and undefended extractor as the cannon suite.
    planetary_catalog(document, 'E10')
    fields(document['satellite']['StructType'], {'id': '17', 'category': 'planet', 'maxHealth': '6',
        'buildLimit': '1', 'possibleAmbit': '16', 'planetaryShieldContribution': '12',
        'planetaryDefenses': 'lowOrbitBallisticInterceptorNetwork'})
    return {'verified': True}


def network(document, pid, owner):
    count, shield = planet(document, pid, owner)
    row = document['planetAttributes']
    if count not in (0, 1) or shield < 12+13*count:
        raise ValueError('invalid cannon count or shield for defended planet')
    if tuple(number(row, key) for key in (NETWORK+'Quantity', 'advancedLowOrbitBallisticsInterceptorNetworkQuantity',
            NETWORK+'SuccessRateNumerator', NETWORK+'SuccessRateDenominator')) != (1, 0, 1, 3):
        raise ValueError('one active satellite and exact interception rate required')
    return count, shield


def before(document, attacker, target, satellite, ao, to, af, pid):
    if len({attacker, target, satellite}) != 3 or ao == to:
        raise ValueError('distinct opposing jamming roles required')
    ready(document['attacker'], attacker, ao, '2', af)
    ready(document['target'], target, to, '14', pid)
    if document['target'].get('structDefenders', []) != []:
        raise ValueError('jamming target may not have defenders')
    sat = document['satellite']
    fields(sat['Struct'], {'locationType': 'planet'})
    if fixture(sat, satellite, to, '17', pid, 'space') != 'ready':
        raise ValueError('satellite must be active on the target owner planet')
    if int(sat['structAttributes']['health']) > 6 or sat['structAttributes'].get('isHidden', False) is not False:
        raise ValueError('invalid satellite health or visibility')
    network(document['planet'], pid, to)
    return {'verified': True}


def check_before(document):
    a, t, s = (document[r]['Struct'] for r in ROLES)
    return before(document, a['id'], t['id'], s['id'], a['owner'], t['owner'], a['locationId'], document['planet']['Planet']['id'])


def attack(document, txhash):
    start = document['before']; check_before(start)
    height = confirmed(document['tx'], txhash)
    rows = records(document['tx'], 'EventAttack', 'eventAttackDetail')
    if len(rows) != 1: raise ValueError('one jamming attack required')
    row = rows[0]; shots = row.get('eventAttackShotDetail')
    if not isinstance(shots, list) or len(shots) != 1: raise ValueError('one unguided projectile required')
    a, t = (start[r]['Struct'] for r in ('attacker', 'target'))
    ah, th = (int(start[r]['structAttributes']['health']) for r in ('attacker', 'target'))
    count, shield = network(start['planet'], t['locationId'], t['owner'])
    ah_after, th_after = ah-count, max(0, th-2)
    fields(row, {'attackerStructId': a['id'], 'attackerStructTypeId': '2', 'attackerPlayerId': a['owner'],
        'attackerStructLocationType': 'fleet', 'attackerStructLocationId': a['locationId'], 'attackerStructOperatingAmbit': 'space',
        'attackerHealthBefore': str(ah), 'attackerHealthAfter': str(ah_after), 'weaponSystem': 'primaryWeapon', 'weaponControl': 'unguided',
        'planetaryDefenseCannonDamageToAttacker': count != 0, 'planetaryDefenseCannonDamage': str(count),
        'planetaryDefenseCannonDamageDestroyedAttacker': ah_after == 0, 'recoilDamageToAttacker': False})
    shot = shots[0]
    fields(shot, {'targetStructId': t['id'], 'targetStructTypeId': '14', 'targetPlayerId': t['owner'],
        'targetStructLocationType': 'planet', 'targetStructLocationId': t['locationId'], 'targetStructOperatingAmbit': 'land',
        'targetHealthBefore': str(th), 'targetHealthAfter': str(th_after), 'targetDestroyed': th_after == 0,
        'damage': '2', 'damageDealt': '2', 'damageReduction': '0', 'armourPiercing': False,
        'blocked': False, 'blockerDestroyed': False, 'evaded': False, 'evadedCause': 'noUnitDefenses',
        'evadedByPlanetaryDefenses': False, 'evadedByPlanetaryDefensesCause': 'noPlanetaryDefense',
        'targetCountered': False, 'targetCounteredDamage': '0', 'targetCounterDestroyedAttacker': False,
        'postDestructionDamageToAttacker': False})
    if shot.get('blockedByStructId', '') != '' or shot.get('eventAttackDefenderCounterDetail') not in (None, []):
        raise ValueError('unexpected jamming defender')
    return {'verified': True, 'height': str(height), 'txhash': txhash, 'roles': {r: start[r]['Struct']['id'] for r in ROLES},
        'health': {'attacker': str(ah_after), 'target': str(th_after), 'satellite': start['satellite']['structAttributes']['health']},
        'cannonCount': str(count), 'shield': str(shield)}


def after(document):
    o = document['outcome']; start = document['before']; end = document['after']
    if o.get('verified') is not True: raise ValueError('missing verified jamming outcome')
    check_before(start)
    if set(o['roles']) != set(ROLES) or set(o['health']) != set(ROLES): raise ValueError('missing jamming role')
    for role in ROLES:
        old = start[role]['Struct']; hp = o['health'][role]
        if old['id'] != o['roles'][role]: raise ValueError('jamming role changed')
        fields(end[role]['Struct'], {'locationType': old['locationType']})
        state(end[role], old['id'], old['owner'], old['type'], old['locationId'], old['operatingAmbit'], hp, hp == '0')
        if hp == '0' and end[role]['structAttributes'].get('isOnline', False) is not False:
            raise ValueError('destroyed jamming fixture remains online')
    p = start['planet']['Planet']
    initial = network(start['planet'], p['id'], p['owner'])
    if network(end['planet'], p['id'], p['owner']) != initial or (o['cannonCount'], o['shield']) != tuple(map(str, initial)):
        raise ValueError('planet defenses changed during unguided volley')
    return {'verified': True}


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command', choices=('catalog', 'before', 'attack', 'after'))
    parser.add_argument('arguments', nargs='*'); args = parser.parse_args()
    try: print(json.dumps(globals()[args.command](json.load(sys.stdin), *args.arguments)))
    except (ValueError, KeyError, TypeError, AttributeError) as error: parser.exit(1, f'Invalid jamming evidence: {error}\n')
