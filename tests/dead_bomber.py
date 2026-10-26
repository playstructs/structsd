"""Certify dead-bomber ordering and the ordinary combat used to prepare its roles."""
import argparse
import json
import sys

from agreement_terms import identity
from combat_fixture import fields, fixture, state
from economic_assertions import confirmed, records
from struct_inventory import positive_integer

# Attacker type/ambit, target type/ambit, armor reduction, reachable target counter.
DUELS={'bomber':('7','air','6','air',0,False),
       'tank':('1','land','9','land',1,True),'hai':('1','air','7','air',0,True),
       'armour':('9','land','9','land',1,True),
       'sustained':('2','space','3','space',0,True),
       'command':('1','space','3','space',0,True)}
KINDS={'bomber':('6','air'),'target':('8','land'),'tank':('9','land'),'hai':('7','air')}
REACH={'1':'32','2':'6','3':'16','6':'6','7':'24','8':'6','9':'4'}


def ordered(tank,hai):
    identity(tank,'5');identity(hai,'5')
    for sid in (tank,hai):positive_integer(sid.split('-')[1],2**64-1,'defender index')
    return tank.encode('ascii') < hai.encode('ascii')


def type_check(row,kind):
    fields(row,{'id':kind,'category':'fleet','maxHealth':'6' if kind=='1' else '3','canDefend':True,
                'primaryWeaponAmbits':REACH[kind]})
    if kind=='9':fields(row,{'unitDefenses':'armour'})
    for key,value in [('secondaryWeaponAmbits','16' if kind in ('2','3') else '0'),('attackReduction','1' if kind=='9' else '0'),
            ('postDestructionDamage','0'),('guidedDefensiveSuccessRateNumerator','2' if kind=='2' else '0'),
            ('counterAttack','0' if kind=='8' else '2' if kind=='1' else '1'),
            ('counterAttackSameAmbit','0' if kind=='8' else '2' if kind=='1' else '1')]:
        if row.get(key,'0')!=value:raise ValueError('unexpected dead-bomber catalog defense or counter')


def weapon_check(row):
    weapon='secondaryWeapon' if row['id']=='2' else 'primaryWeapon'
    fields(row,{'attackCounterable':True,weapon+'Control':'unguided' if row['id']=='9' else 'guided',weapon+'Targets':'1',
        weapon+'Shots':'1',weapon+'Damage':'2',weapon+'Blockable':True,weapon+'Counterable':True,
        weapon+'Charge':'3' if row['id'] in ('1','9') else '5',
        weapon+'ShotSuccessRateNumerator':'1',weapon+'ShotSuccessRateDenominator':'1'})
    if row.get(weapon+'RecoilDamage','0')!='0' or row.get(weapon+'ArmourPiercing',False) is not False:
        raise ValueError('unexpected recoil or armor piercing')


def catalog(document,mode):
    if mode=='regression':
        for role,(kind,_) in KINDS.items():type_check(document[role]['StructType'],kind)
        weapon_check(document['bomber']['StructType'])
    else:
        ak,_,tk,_,_,_=DUELS[mode]
        type_check(document['attacker']['StructType'],ak);type_check(document['target']['StructType'],tk)
        weapon_check(document['attacker']['StructType'])
    return {'verified':True}


def ready(document,sid,owner,kind,fleet,ambit):
    if fixture(document,sid,owner,kind,fleet,ambit)!='ready':raise ValueError('dead-bomber role is unavailable')
    hp=document['structAttributes'].get('health','0')
    if hp not in tuple(str(i) for i in range(1,7 if kind=='1' else 4)):raise ValueError('invalid fixture health')
    hidden=document['structAttributes'].get('isHidden',False)
    if type(hidden) is not bool or hidden:raise ValueError('fixture must be revealed')


def before(document,mode,*args):
    if mode=='regression':
        bomber,target,tank,hai,ao,to,af,tf=args
        if len({bomber,target,tank,hai})!=4 or ao==to or af==tf:raise ValueError('four distinct opposing roles required')
        for role,sid in [('bomber',bomber),('target',target),('tank',tank),('hai',hai)]:
            kind,ambit=KINDS[role];ready(document[role],sid,ao if role=='bomber' else to,kind,af if role=='bomber' else tf,ambit)
        if document['bomber']['structAttributes']['health']!='1':raise ValueError('bomber must have exactly one health')
        if not ordered(tank,hai):raise ValueError('Tank must sort before the lethal counter')
        if document['target'].get('structDefenders',[])!=[tank,hai]:raise ValueError('exact ordered defender index required')
        for role in ('tank','hai'):
            if document[role]['structAttributes'].get('protectedStructIndex')!=target.split('-')[1]:raise ValueError('wrong defense association')
    else:
        a,t,ao,to,af,tf=args;ak,aa,tk,ta,_,_=DUELS[mode]
        if a==t or ao==to or af==tf:raise ValueError('opposing duel roles required')
        ready(document['attacker'],a,ao,ak,af,aa);ready(document['target'],t,to,tk,tf,ta)
        if document['target'].get('structDefenders',[])!=[]:raise ValueError('preparation target may not be defended')
    return {'verified':True}


def check_before(document,mode):
    if mode=='regression':
        rows=[document[r]['Struct'] for r in KINDS];a,t=rows[:2]
        return before(document,mode,*[r['id'] for r in rows],a['owner'],t['owner'],a['locationId'],t['locationId'])
    a,t=(document[r]['Struct'] for r in ('attacker','target'))
    return before(document,mode,a['id'],t['id'],a['owner'],t['owner'],a['locationId'],t['locationId'])


def range_before(document,a,t,ao,to,af,tf):
    if a==t or ao==to or af==tf:raise ValueError('opposing range-test roles required')
    ready(document['attacker'],a,ao,'1',af,'land')
    ready(document['target'],t,to,'3',tf,'space')
    if document['target'].get('structDefenders',[])!=[]:raise ValueError('range target may not be defended')
    return {'verified':True}


def range_refusal(document,txhash):
    start=document['before'];a,t=(start[r]['Struct'] for r in ('attacker','target'))
    range_before(start,a['id'],t['id'],a['owner'],t['owner'],a['locationId'],t['locationId'])
    # Current structured errors implement Code/Unwrap, while SDK ABCIInfo reads
    # ABCICode/Cause. The delivered envelope is therefore 1/undefined; the exact
    # targeting reason below is essential to distinguish this from other errors.
    tx=document['tx'];fields(tx,{'code':1,'codespace':'undefined'})
    # Reuse envelope validation while requiring the original failure code above.
    height=confirmed({**tx,'code':0},txhash)
    reason=f"target struct ({t['id']}) is out_of_range from attacker struct ({a['id']})"
    if not isinstance(tx.get('raw_log'),str) or reason not in tx['raw_log'] or 'panic' in tx['raw_log'].lower():
        raise ValueError('wrong targeting refusal')
    if records(tx,'EventAttack','eventAttackDetail'):raise ValueError('rejected attack emitted combat outcome')
    if document['after']!=start:raise ValueError('rejected attack changed combat state')
    return {'verified':True,'mode':'command_range','height':str(height),'txhash':txhash,
            'roles':{'attacker':a['id'],'target':t['id']}}


def attack(document,mode,txhash):
    start=document['before'];check_before(start,mode)
    height=confirmed(document['tx'],txhash);rows=records(document['tx'],'EventAttack','eventAttackDetail')
    if len(rows)!=1:raise ValueError('one dead-bomber fixture attack required')
    row=rows[0];shots=row.get('eventAttackShotDetail')
    if not isinstance(shots,list) or len(shots)!=1:raise ValueError('one projectile result required')
    ar='bomber' if mode=='regression' else 'attacker'
    a,t=start[ar]['Struct'],start['target']['Struct']
    ah,th=int(start[ar]['structAttributes']['health']),int(start['target']['structAttributes']['health'])
    reduction=0 if mode=='regression' else DUELS[mode][4]
    damage=0 if mode=='regression' else 2-reduction
    th_after=max(0,th-damage)
    counter=False if mode=='regression' else DUELS[mode][5] and th_after>0
    ah_after=0 if mode=='regression' else max(0,ah-int(counter))
    fields(row,{'attackerStructId':a['id'],'attackerStructTypeId':a['type'],'attackerPlayerId':a['owner'],
        'attackerStructLocationType':'fleet','attackerStructLocationId':a['locationId'],
        'attackerStructOperatingAmbit':a['operatingAmbit'],'attackerHealthBefore':str(ah),'attackerHealthAfter':str(ah_after),
        'weaponSystem':'secondaryWeapon' if mode=='sustained' else 'primaryWeapon','weaponControl':'unguided' if mode=='armour' else 'guided','recoilDamageToAttacker':False,'planetaryDefenseCannonDamageToAttacker':False})
    shot=shots[0]
    if reduction:fields(shot,{'damageReductionCause':'armour'})
    fields(shot,{'targetStructId':t['id'],'targetStructTypeId':t['type'],'targetPlayerId':t['owner'],
        'targetStructLocationType':'fleet','targetStructLocationId':t['locationId'],'targetStructOperatingAmbit':t['operatingAmbit'],
        'targetHealthBefore':str(th),'targetHealthAfter':str(th_after),'targetDestroyed':th_after==0,
        'damage':str(damage),'damageDealt':'0' if mode=='regression' else '2','damageReduction':str(reduction),
        'armourPiercing':False,'blocked':False,'blockerDestroyed':False,'evaded':False,'evadedByPlanetaryDefenses':False,
        'targetCountered':counter,'targetCounteredDamage':'1' if counter else '0',
        'targetCounterDestroyedAttacker':counter and ah_after==0,'postDestructionDamageToAttacker':False})
    if shot.get('blockedByStructId','')!='':raise ValueError('dead attacker may not produce a blocker')
    health={ar:str(ah_after),'target':str(th_after)}
    if mode=='regression':
        ds=shot.get('eventAttackDefenderCounterDetail')
        if not isinstance(ds,list) or len(ds)!=1:raise ValueError('exactly one HAI counter required')
        d=start['hai']['Struct']
        fields(ds[0],{'counterByStructId':d['id'],'counterByStructTypeId':'7','counterByStructLocationType':'fleet',
            'counterByStructLocationId':d['locationId'],'counterByStructOperatingAmbit':'air','counterDamage':'1','counterDestroyedAttacker':True})
        for role in ('tank','hai'):health[role]=start[role]['structAttributes']['health']
    elif shot.get('eventAttackDefenderCounterDetail') not in (None,[]):raise ValueError('unexpected preparation defender')
    return {'verified':True,'mode':mode,'height':str(height),'txhash':txhash,'health':health,
            'roles':{role:start[role]['Struct']['id'] for role in health}}


def after(document):
    o=document['outcome'];start=document['before']
    if o.get('verified') is not True:raise ValueError('missing verified fixture outcome')
    check_before(start,o['mode'])
    for role,hp in o['health'].items():
        old=start[role]['Struct']
        if old['id']!=o['roles'][role]:raise ValueError('fixture identity changed')
        state(document['after'][role],old['id'],old['owner'],old['type'],old['locationId'],old['operatingAmbit'],hp,hp=='0')
    return {'verified':True}


if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command',choices=('catalog','before','attack','after','ordered','range_before','range_refusal'))
    parser.add_argument('arguments',nargs='*');args=parser.parse_args()
    try:
        result=ordered(*args.arguments) if args.command=='ordered' else globals()[args.command](json.load(sys.stdin),*args.arguments)
        print(json.dumps(result))
    except (ValueError,KeyError,TypeError,AttributeError) as error:parser.exit(1,f'Invalid dead-bomber evidence: {error}\n')
