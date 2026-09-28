"""Certify an ordinary command-ship attack used to replace a damaged test fighter."""
import argparse
import json
import sys

from combat_fixture import fields, fixture, state
from economic_assertions import confirmed, records


def before(document, commander, fighter, commander_owner, fighter_owner, commander_fleet, fighter_fleet):
    if commander_owner == fighter_owner or commander_fleet == fighter_fleet or commander == fighter:
        raise ValueError('fighter retirement needs opposing fleets')
    for role, args in [('attacker',(commander,commander_owner,'1',commander_fleet,'space')),
                       ('target',(fighter,fighter_owner,'3',fighter_fleet,'space'))]:
        if fixture(document[role],*args) != 'ready':
            raise ValueError('retirement role is unavailable')
    if not 1 <= int(document['attacker']['structAttributes']['health']) <= 6:
        raise ValueError('command ship health is invalid')
    if document['target']['structAttributes']['health'] not in ('1','2') or document['target'].get('structDefenders',[]) != []:
        raise ValueError('only an undefended damaged fighter can be retired')
    return {'verified':True}


def catalog(document):
    a,t = document['attacker']['StructType'],document['target']['StructType']
    fields(a,{'id':'1','primaryWeaponControl':'guided','primaryWeaponCharge':'3','primaryWeaponAmbits':'32',
              'primaryWeaponTargets':'1','primaryWeaponShots':'1','primaryWeaponDamage':'2',
              'primaryWeaponShotSuccessRateNumerator':'1','primaryWeaponShotSuccessRateDenominator':'1'})
    fields(t,{'id':'3','maxHealth':'3'})
    for row,key in [(a,'primaryWeaponRecoilDamage'),(a,'postDestructionDamage'),
                    (t,'postDestructionDamage'),(t,'attackReduction'),(t,'guidedDefensiveSuccessRateNumerator')]:
        if row.get(key,'0') != '0':raise ValueError('unexpected retirement damage or defense')
    return {'verified':True}


def attack(document, txhash):
    start=document['before'];a=start['attacker']['Struct'];t=start['target']['Struct']
    before(start,a['id'],t['id'],a['owner'],t['owner'],a['locationId'],t['locationId'])
    height=confirmed(document['tx'],txhash)
    rows=records(document['tx'],'EventAttack','eventAttackDetail')
    if len(rows)!=1:raise ValueError('one retirement attack is required')
    row=rows[0];shots=row.get('eventAttackShotDetail')
    if not isinstance(shots,list) or len(shots)!=1:raise ValueError('one retirement projectile is required')
    fields(row,{'attackerStructId':a['id'],'attackerStructTypeId':'1','attackerPlayerId':a['owner'],
        'attackerStructLocationType':'fleet','attackerStructLocationId':a['locationId'],'attackerStructOperatingAmbit':'space',
        'attackerHealthBefore':start['attacker']['structAttributes']['health'],
        'attackerHealthAfter':start['attacker']['structAttributes']['health'],
        'weaponSystem':'primaryWeapon','weaponControl':'guided','recoilDamageToAttacker':False,
        'planetaryDefenseCannonDamageToAttacker':False})
    shot=shots[0]
    fields(shot,{'targetStructId':t['id'],'targetStructTypeId':'3','targetPlayerId':t['owner'],
        'targetStructLocationType':'fleet','targetStructLocationId':t['locationId'],'targetStructOperatingAmbit':'space',
        'targetHealthBefore':start['target']['structAttributes']['health'],'targetHealthAfter':'0','targetDestroyed':True,
        'damage':'2','damageDealt':'2','damageReduction':'0','blocked':False,'evaded':False,
        'evadedByPlanetaryDefenses':False,'targetCountered':False,'targetCounteredDamage':'0',
        'targetCounterDestroyedAttacker':False,'postDestructionDamageToAttacker':False})
    if shot.get('eventAttackDefenderCounterDetail') not in (None,[]):raise ValueError('unexpected retirement defender')
    return {'verified':True,'height':str(height),'txhash':txhash,'commander':a['id'],'fighter':t['id'],
            'commanderHealth':start['attacker']['structAttributes']['health']}


def after(document):
    outcome=document['outcome']
    if outcome.get('verified') is not True:raise ValueError('missing retirement outcome')
    for role,hp,destroyed in [('attacker',outcome['commanderHealth'],False),('target','0',True)]:
        old=document['before'][role]['Struct']
        expected=outcome['commander'] if role=='attacker' else outcome['fighter']
        if old['id']!=expected:raise ValueError('retirement identity changed')
        state(document['after'][role],old['id'],old['owner'],old['type'],old['locationId'],old['operatingAmbit'],hp,destroyed)
    return {'verified':True}


if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command',choices=('before','catalog','attack','after'))
    parser.add_argument('arguments',nargs='*')
    args=parser.parse_args()
    try:print(json.dumps(globals()[args.command](json.load(sys.stdin),*args.arguments)))
    except (ValueError,TypeError,KeyError,AttributeError) as error:parser.exit(1,f'Invalid fighter retirement evidence: {error}\n')
