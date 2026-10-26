"""Verify Phase 13 targeting refusals, armor piercing and the four-unit defense."""
import argparse
import json
import sys

from combat_fixture import fields, fixture, state
from dead_bomber import ready
from economic_assertions import confirmed, records

PROFILES = {'unbuilt': ('2','space','secondaryWeapon','guided'),
            'range': ('1','space','primaryWeapon','unguided'),
            'armour': ('9','land','primaryWeapon','unguided'),
            'defended': ('1','space','secondaryWeapon','guided')}
REACH = {'1':('32','0'), '2':('6','16'), '9':('4','0'), '10':('24','0'), '13':('18','0')}


def catalog(doc, mode):
    kind, _, weapon, control = PROFILES[mode]
    rows = [(doc['attacker']['StructType'],'2'),(doc['target']['StructType'],kind)]
    if mode == 'defended':
        if sorted(r['StructType']['id'] for r in doc['defenders']) != ['10','13','2','2']:
            raise ValueError('wrong defense catalog')
        rows += [(r['StructType'],r['StructType']['id']) for r in doc['defenders']]
    for row, expected in rows:
        fields(row, {'id':expected,'category':'fleet','canDefend':True,'maxHealth':'6' if expected=='1' else '3',
                     'primaryWeaponAmbits':REACH[expected][0], 'counterAttack':'2' if expected=='1' else '1',
                     'counterAttackSameAmbit':'2' if expected=='1' else '1'})
        for key, value in [('secondaryWeaponAmbits',REACH[expected][1]),('postDestructionDamage','0'),
                           ('attackReduction','1' if expected=='9' else '0')]:
            if row.get(key,'0') != value: raise ValueError('unexpected defense or reach')
    target = doc['target']['StructType']
    if mode != 'unbuilt' and target.get(control+'DefensiveSuccessRateNumerator','0') != '0':
        raise ValueError('fixture target must not evade')
    a = doc['attacker']['StructType']
    fields(a, {'attackCounterable':True,weapon+'Control':control,weapon+'Charge':'5',weapon+'Targets':'1',
               weapon+'Shots':'1',weapon+'Damage':'2',weapon+'Blockable':True,weapon+'Counterable':True,
               weapon+'ShotSuccessRateNumerator':'1',weapon+'ShotSuccessRateDenominator':'1'})
    if a.get(weapon+'RecoilDamage','0') != '0' or a.get(weapon+'ArmourPiercing',False) is not (weapon=='primaryWeapon'):
        raise ValueError('unexpected piercing or recoil')
    return {'verified':True}


def before(doc, mode, attacker, target, ao, to, af, tf):
    kind, ambit, _, _ = PROFILES[mode]
    if attacker == target or ao == to or af == tf: raise ValueError('opposing combat roles required')
    ready(doc['attacker'],attacker,ao,'2',af,'space')
    if mode == 'unbuilt':
        if fixture(doc['target'],target,to,kind,tf,ambit) != 'build': raise ValueError('target must still be unbuilt')
    else: ready(doc['target'],target,to,kind,tf,ambit)
    defenders = doc.get('defenders',[])
    if mode == 'defended':
        if len(defenders)!=4 or sorted(d['Struct']['type'] for d in defenders)!=['10','13','2','2']:
            raise ValueError('four exact defender roles required')
        if doc['attacker']['structAttributes']['health'] != '3': raise ValueError('full-health attacker must exercise three defender counters')
        ids = [d['Struct']['id'] for d in defenders]
        if len(set(ids+[attacker,target])) != 6 or doc['target'].get('structDefenders',[]) != sorted(ids):
            raise ValueError('defense registry differs from required roles')
        for d in defenders:
            role=d['Struct']; da={'10':'land','13':'water','2':'space'}[role['type']]
            if fixture(d,role['id'],to,role['type'],tf,da) != 'ready' or d['structAttributes']['health'] not in ('1','2','3'):
                raise ValueError('defender must be built, alive and online')
            if d['structAttributes'].get('protectedStructIndex') != target.split('-')[1]: raise ValueError('wrong defender association')
    elif defenders or doc['target'].get('structDefenders',[]) != []:
        raise ValueError('isolated targeting/piercing test may not have defenders')
    if mode == 'armour' and doc['target']['structAttributes']['health'] != '3':
        raise ValueError('armor target must have full health')
    return {'verified':True}


def validate(doc, mode):
    a,t=(doc[k]['Struct'] for k in ('attacker','target'))
    return before(doc,mode,a['id'],t['id'],a['owner'],t['owner'],a['locationId'],t['locationId'])


def verify(doc, mode, txhash):
    start=doc['before']; validate(start,mode); tx=doc['tx']
    a,t=(start[k]['Struct'] for k in ('attacker','target'))
    if mode in ('unbuilt','range'):
        fields(tx,{'code':1,'codespace':'undefined'})
        height=confirmed({**tx,'code':0},txhash)
        reason='unbuilt' if mode=='unbuilt' else 'out_of_range'
        expected=f"target struct ({t['id']}) is {reason} from attacker struct ({a['id']})"
        if not isinstance(tx.get('raw_log'),str) or expected not in tx['raw_log'] or 'panic' in tx['raw_log'].lower():
            raise ValueError('wrong targeting refusal')
        if records(tx,'EventAttack','eventAttackDetail') or doc['after'] != start: raise ValueError('rejected attack changed combat state')
        return {'verified':True,'mode':mode,'height':str(height),'txhash':txhash,'attacker':a['id'],'target':t['id']}
    height=confirmed(tx,txhash); rows=records(tx,'EventAttack','eventAttackDetail')
    if len(rows)!=1 or len(rows[0].get('eventAttackShotDetail',[]))!=1: raise ValueError('one exact volley required')
    row=rows[0]; shot=row['eventAttackShotDetail'][0]; _,_,weapon,control=PROFILES[mode]
    ah=start['attacker']['structAttributes']['health']; th=start['target']['structAttributes']['health']
    defended=mode=='defended'; hp={a['id']:'0' if defended else ah,t['id']:th if defended else '1'}
    fields(row,{'attackerStructId':a['id'],'attackerStructTypeId':'2','attackerPlayerId':a['owner'],
        'attackerStructLocationType':'fleet','attackerStructLocationId':a['locationId'],'attackerStructOperatingAmbit':'space',
        'attackerHealthBefore':ah,'attackerHealthAfter':hp[a['id']],'weaponSystem':weapon,'weaponControl':control,
        'recoilDamageToAttacker':False,'planetaryDefenseCannonDamageToAttacker':False})
    fields(shot,{'targetStructId':t['id'],'targetStructTypeId':t['type'],'targetPlayerId':t['owner'],
        'targetStructLocationType':'fleet','targetStructLocationId':t['locationId'],'targetStructOperatingAmbit':t['operatingAmbit'],
        'targetHealthBefore':th,'targetHealthAfter':hp[t['id']],'targetDestroyed':False,
        'damage':'0' if defended else '2','damageDealt':'0' if defended else '2','damageReduction':'0','armourPiercing':not defended,
        'evaded':False,'evadedCause':'noUnitDefenses','evadedByPlanetaryDefenses':False,'blocked':False,'blockerDestroyed':False,
        'targetCountered':False,'targetCounteredDamage':'0','targetCounterDestroyedAttacker':False,
        'postDestructionDamageToAttacker':False})
    counters=shot.get('eventAttackDefenderCounterDetail') or []
    if defended:
        eligible=sorted(start['defenders'],key=lambda d:d['Struct']['id'])[:3]
        if len(counters)!=3: raise ValueError('three ordered defender counters must kill the attacker')
        for i,(d,c) in enumerate(zip(eligible,counters)):
            s=d['Struct'];fields(c,{'counterByStructId':s['id'],'counterByStructTypeId':s['type'],'counterByStructLocationType':'fleet',
                'counterByStructLocationId':s['locationId'],'counterByStructOperatingAmbit':s['operatingAmbit'],'counterDamage':'1',
                'counterDestroyedAttacker':i==2,'counterByStructWeaponSystem':'secondaryWeapon' if s['type']=='2' else 'primaryWeapon',
                'counterByStructWeaponControl':'guided'})
        if shot.get('blockedByStructId','')!='': raise ValueError('dead attacker may not damage a blocker')
        for d in start['defenders']: hp[d['Struct']['id']]=d['structAttributes']['health']
    elif counters or shot.get('blockedByStructId','')!='' or shot.get('blockerDestroyed') is not False:
        raise ValueError('isolated armor hit had a defender or counter')
    for key in ('attacker','target'):
        old=start[key]['Struct']; state(doc['after'][key],old['id'],old['owner'],old['type'],old['locationId'],old['operatingAmbit'],hp[old['id']],hp[old['id']]=='0')
    if len(doc['after'].get('defenders',[]))!=len(start.get('defenders',[])): raise ValueError('missing post-state defender')
    for old,new in zip(start.get('defenders',[]),doc['after'].get('defenders',[])):
        s=old['Struct']; state(new,s['id'],s['owner'],s['type'],s['locationId'],s['operatingAmbit'],hp[s['id']],hp[s['id']]=='0')
    return {'verified':True,'mode':mode,'height':str(height),'txhash':txhash,'attacker':a['id'],'target':t['id'],'health':hp,'defenders':[d['Struct']['id'] for d in start.get('defenders',[])]}


if __name__=='__main__':
    p=argparse.ArgumentParser(description=__doc__);p.add_argument('command',choices=('catalog','before','verify'));p.add_argument('mode',choices=tuple(PROFILES));p.add_argument('arguments',nargs='*');args=p.parse_args()
    try: print(json.dumps(globals()[args.command](json.load(sys.stdin),args.mode,*args.arguments)))
    except (ValueError,TypeError,KeyError,AttributeError) as error:p.exit(1,f'Invalid Phase 13 evidence: {error}\n')
