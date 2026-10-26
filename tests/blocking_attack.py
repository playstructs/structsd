"""Certify C2's guided Battleship volley, two defenders and actual blocking."""
import argparse
import json
import sys

from combat_fixture import fields, state
from dead_bomber import ready, weapon_check
from economic_assertions import confirmed, records

KINDS={'attacker':'2','target':'2','frigate':'4','fighter':'3'}


def catalog(document):
    for role,kind in KINDS.items():
        row=document[role]['StructType']
        fields(row,{'id':kind,'category':'fleet','maxHealth':'3','canDefend':True,
                    'primaryWeaponAmbits':{'2':'6','3':'16','4':'24'}[kind]})
        for key,value in [('secondaryWeaponAmbits','0' if kind=='4' else '16'),
                ('counterAttack','1'),('counterAttackSameAmbit','1'),('attackReduction','0'),
                ('postDestructionDamage','0'),('unguidedDefensiveSuccessRateNumerator','0'),
                ('guidedDefensiveSuccessRateNumerator','2' if kind=='2' else '0')]:
            if row.get(key,'0')!=value:raise ValueError('unexpected blocking catalog')
        if kind=='2':fields(row,{'guidedDefensiveSuccessRateDenominator':'3','unitDefenses':'signalJamming'})
    weapon_check(document['attacker']['StructType'])
    return {'verified':True}


def before(document,attacker,target,frigate,fighter,ao,to,af,tf):
    ids=dict(zip(KINDS,(attacker,target,frigate,fighter)))
    if len(set(ids.values()))!=4 or ao==to or af==tf:raise ValueError('four opposing blocking roles required')
    for role,sid in ids.items():ready(document[role],sid,ao if role=='attacker' else to,KINDS[role],af if role=='attacker' else tf,'space')
    if document['target'].get('structDefenders',[])!=sorted([frigate,fighter]):raise ValueError('exact two defenders required')
    for role in ('frigate','fighter'):
        if document[role]['structAttributes'].get('protectedStructIndex')!=target.split('-')[1]:raise ValueError('wrong defender association')
    return {'verified':True}


def validate_before(document):
    a,t=(document[r]['Struct'] for r in ('attacker','target'))
    return before(document,*[document[r]['Struct']['id'] for r in KINDS],a['owner'],t['owner'],a['locationId'],t['locationId'])


def attack(document,txhash):
    start=document['before'];validate_before(start)
    height=confirmed(document['tx'],txhash);rows=records(document['tx'],'EventAttack','eventAttackDetail')
    if len(rows)!=1:raise ValueError('one blocking attack required')
    row=rows[0];shots=row.get('eventAttackShotDetail')
    if not isinstance(shots,list) or len(shots)!=1:raise ValueError('one guided projectile required')
    shot=shots[0];evaded=shot.get('evaded')
    if type(evaded) is not bool:raise ValueError('invalid evasion flag')
    ah=int(start['attacker']['structAttributes']['health']);remaining=ah;counters=[]
    defenders=sorted(('frigate','fighter'),key=lambda r:start[r]['Struct']['id'])
    for role in defenders:
        if remaining==0:break
        d=start[role]['Struct'];remaining-=1
        counters.append({'counterByStructId':d['id'],'counterByStructTypeId':d['type'],
            'counterByStructLocationType':'fleet','counterByStructLocationId':d['locationId'],
            'counterByStructOperatingAmbit':'space','counterByStructWeaponSystem':'primaryWeapon',
            'counterByStructWeaponControl':'guided','counterByStructActiveWeaponry':'guidedWeaponry',
            'counterDamage':'1','counterDestroyedAttacker':remaining==0})
    blocker=defenders[0] if remaining and not evaded else None
    counter=remaining>0;final=remaining-int(counter)
    a,t=(start[r]['Struct'] for r in ('attacker','target'))
    hp={r:start[r]['structAttributes']['health'] for r in KINDS};hp['attacker']=str(final)
    if blocker:hp[blocker]=str(max(0,int(hp[blocker])-2))
    fields(row,{'attackerStructId':a['id'],'attackerStructTypeId':'2','attackerPlayerId':a['owner'],
        'attackerStructLocationType':'fleet','attackerStructLocationId':a['locationId'],'attackerStructOperatingAmbit':'space',
        'attackerHealthBefore':str(ah),'attackerHealthAfter':str(final),'weaponSystem':'secondaryWeapon','weaponControl':'guided',
        'planetaryDefenseCannonDamageToAttacker':False,'recoilDamageToAttacker':False})
    fields(shot,{'targetStructId':t['id'],'targetStructTypeId':'2','targetPlayerId':t['owner'],
        'targetStructLocationType':'fleet','targetStructLocationId':t['locationId'],'targetStructOperatingAmbit':'space',
        'targetHealthBefore':hp['target'],'targetHealthAfter':hp['target'],'targetDestroyed':False,
        'damage':'2' if blocker else '0','damageDealt':'2' if blocker else '0','damageReduction':'0','armourPiercing':False,
        'blocked':blocker is not None,'blockerDestroyed':blocker is not None and hp[blocker]=='0',
        'evadedByPlanetaryDefenses':False,'evadedCause':'signalJamming' if evaded else 'noUnitDefenses',
        'targetCountered':counter,'targetCounteredDamage':'1' if counter else '0',
        'targetCounterDestroyedAttacker':counter,'postDestructionDamageToAttacker':False})
    actual=shot.get('eventAttackDefenderCounterDetail') or []
    if not isinstance(actual,list) or len(actual)!=len(counters):raise ValueError('wrong defender counter inventory')
    for found,wanted in zip(actual,counters):fields(found,wanted)
    if counter:fields(shot,{'targetCounterWeaponSystem':'secondaryWeapon','targetCounterWeaponControl':'guided','targetCounterActiveWeaponry':'guidedWeaponry'})
    if blocker:
        b=start[blocker]['Struct']
        fields(shot,{'blockedByStructId':b['id'],'blockedByStructTypeId':b['type'],
            'blockedByStructLocationType':'fleet','blockedByStructLocationId':b['locationId'],
            'blockedByStructOperatingAmbit':'space','blockerHealthBefore':start[blocker]['structAttributes']['health'],
            'blockerHealthAfter':hp[blocker]})
    elif shot.get('blockedByStructId','')!='':raise ValueError('unexpected blocker')
    return {'verified':True,'height':str(height),'txhash':txhash,'blocked':blocker is not None,'evaded':evaded,
            'health':hp,'roles':{r:start[r]['Struct']['id'] for r in KINDS}}


def after(document):
    o=document['outcome'];start=document['before'];validate_before(start)
    if o.get('verified') is not True:raise ValueError('missing verified blocking outcome')
    for role,hp in o['health'].items():
        old=start[role]['Struct']
        if o['roles'][role]!=old['id']:raise ValueError('blocking role changed')
        state(document['after'][role],old['id'],old['owner'],old['type'],old['locationId'],'space',hp,hp=='0')
    return {'verified':True}


if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command',choices=('catalog','before','attack','after'));parser.add_argument('arguments',nargs='*');args=parser.parse_args()
    try:print(json.dumps(globals()[args.command](json.load(sys.stdin),*args.arguments)))
    except (ValueError,KeyError,TypeError,AttributeError) as error:parser.exit(1,f'Invalid blocking evidence: {error}\n')
