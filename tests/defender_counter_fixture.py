"""Required RG1 roles and deterministic fighter preparation through signed combat."""
import argparse
import json
import sys

from combat_fixture import fields, fixture, state
from defender_counter_kill import verify
from economic_assertions import confirmed, records


def catalog(document):
    sf,bb = document['fighter']['StructType'],document['defender']['StructType']
    fields(sf,{'id':'3','category':'fleet','maxHealth':'3','canDefend':True,
        'attackCounterable':True,'primaryWeaponControl':'guided','primaryWeaponCharge':'3',
        'primaryWeaponAmbits':'16','primaryWeaponTargets':'1','primaryWeaponShots':'1','primaryWeaponDamage':'2',
        'primaryWeaponShotSuccessRateNumerator':'1','primaryWeaponShotSuccessRateDenominator':'1',
        'primaryWeaponBlockable':True,'primaryWeaponCounterable':True,'counterAttack':'1','counterAttackSameAmbit':'1'})
    fields(bb,{'id':'2','category':'fleet','maxHealth':'3','canDefend':True,'counterAttack':'1','counterAttackSameAmbit':'1'})
    for row in (sf,bb):
        for key in ('attackReduction','postDestructionDamage'):
            if row.get(key,'0') != '0':raise ValueError('unexpected RG1 damage modifiers')
    for key in ('primaryWeaponRecoilDamage','guidedDefensiveSuccessRateNumerator'):
        if sf.get(key,'0') != '0':raise ValueError('unexpected fighter recoil or evasion')
    return {'verified':True}


def before(document, mode, attacker, target, defender, attacker_owner, target_owner, attacker_fleet, target_fleet):
    if mode not in ('prepare','regression'):raise ValueError('unknown RG1 mode')
    if len({attacker,target,defender}) != 3 or attacker_owner == target_owner or attacker_fleet == target_fleet:
        raise ValueError('RG1 needs three distinct roles on opposing fleets')
    for role,sid,owner,kind,fleet in [('attacker',attacker,attacker_owner,'3',attacker_fleet),
            ('target',target,target_owner,'3',target_fleet),('defender',defender,target_owner,'2',target_fleet)]:
        if fixture(document[role],sid,owner,kind,fleet,'space') != 'ready':raise ValueError('RG1 role is unavailable')
        if document[role]['structAttributes']['health'] not in ('1','2','3'):raise ValueError('invalid RG1 health')
    a,t,d = (document[r] for r in ('attacker','target','defender'))
    if mode == 'prepare':
        # The eventual target fires first to reduce the eventual attacker 3→1.
        if a['structAttributes']['health'] != '3' or t['structAttributes']['health'] != '3':
            raise ValueError('RG1 preparation needs two healthy fighters')
        if a.get('structDefenders',[]) != []:raise ValueError('preparation fighter is defended')
    else:
        if a['structAttributes']['health'] != '1':raise ValueError('regression attacker must have one health')
        if t.get('structDefenders',[]) != [defender] or d['structAttributes'].get('protectedStructIndex') != target.split('-')[1]:
            raise ValueError('RG1 requires exactly the registered defender')
    return {'verified':True}


def check_before(document, mode):
    a,t,d = (document[r]['Struct'] for r in ('attacker','target','defender'))
    return before(document,mode,a['id'],t['id'],d['id'],a['owner'],t['owner'],a['locationId'],t['locationId'])


def attack(document, mode, txhash):
    start=document['before'];check_before(start,mode)
    a,t,d = (start[r]['Struct'] for r in ('attacker','target','defender'))
    health={r:start[r]['structAttributes']['health'] for r in ('attacker','target','defender')}
    height=confirmed(document['tx'],txhash)
    rows=records(document['tx'],'EventAttack','eventAttackDetail')
    if len(rows)!=1:raise ValueError('one RG1 attack is required')
    row=rows[0]
    if mode=='regression':
        verify(document['tx'],txhash,a['id'],t['id'],d['id'],health['target'])
        health['attacker']='0'
        shooter,receiver=a,t
    else:
        shooter,receiver=t,a
        fields(row,{'attackerStructId':t['id'],'attackerStructTypeId':'3','weaponSystem':'primaryWeapon',
            'attackerHealthBefore':'3','attackerHealthAfter':'2',
            'recoilDamageToAttacker':False,'planetaryDefenseCannonDamageToAttacker':False})
        shots=row.get('eventAttackShotDetail')
        if not isinstance(shots,list) or len(shots)!=1:raise ValueError('one preparation projectile is required')
        fields(shots[0],{'targetStructId':a['id'],'targetStructTypeId':'3','targetHealthBefore':'3','targetHealthAfter':'1',
            'targetDestroyed':False,'damage':'2','damageDealt':'2','damageReduction':'0','blocked':False,
            'evaded':False,'evadedByPlanetaryDefenses':False,'targetCountered':True,'targetCounteredDamage':'1',
            'targetCounterDestroyedAttacker':False,'postDestructionDamageToAttacker':False})
        if shots[0].get('eventAttackDefenderCounterDetail') not in (None,[]):raise ValueError('unexpected preparation defender')
        health.update(attacker='1',target='2')
    fields(row,{'attackerPlayerId':shooter['owner'],'attackerStructLocationType':'fleet',
        'attackerStructLocationId':shooter['locationId'],'attackerStructOperatingAmbit':'space','weaponControl':'guided'})
    shot=row['eventAttackShotDetail'][0]
    fields(shot,{'targetPlayerId':receiver['owner'],'targetStructLocationType':'fleet',
        'targetStructLocationId':receiver['locationId'],'targetStructOperatingAmbit':'space'})
    if shot.get('blockedByStructId','')!='':raise ValueError('RG1 may not name a blocker')
    if mode=='regression':
        fields(shot['eventAttackDefenderCounterDetail'][0],{'counterByStructTypeId':'2','counterByStructLocationType':'fleet',
            'counterByStructLocationId':d['locationId'],'counterDamage':'1'})
    return {'verified':True,'mode':mode,'height':str(height),'txhash':txhash,'health':health,
            'attacker':a['id'],'target':t['id'],'defender':d['id']}


def after(document):
    outcome=document['outcome'];start=document['before']
    if outcome.get('verified') is not True:raise ValueError('missing RG1 outcome')
    check_before(start,outcome['mode'])
    for role in ('attacker','target','defender'):
        old=start[role]['Struct'];hp=outcome['health'][role]
        if old['id']!=outcome[role]:raise ValueError('RG1 identity changed')
        state(document['after'][role],old['id'],old['owner'],old['type'],old['locationId'],'space',hp,hp=='0')
    return {'verified':True}


if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command',choices=('catalog','before','attack','after'))
    parser.add_argument('arguments',nargs='*');args=parser.parse_args()
    try:print(json.dumps(globals()[args.command](json.load(sys.stdin),*args.arguments)))
    except (ValueError,KeyError,TypeError,AttributeError) as error:parser.exit(1,f'Invalid required RG1 evidence: {error}\n')
