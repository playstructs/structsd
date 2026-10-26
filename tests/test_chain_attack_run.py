"""Attack-run coverage requires complete defense, exact receipts and state."""
import copy
import json
from pathlib import Path
import re
import subprocess
import tempfile
import unittest

from attack_run import AMBITS, CASES, TYPES, after, attack, before, catalog, profile
from test_chain_evasion import structure

ROOT = Path(__file__).parent.resolve()
SOURCE = (ROOT/'test_chain.sh').read_text()
SHELL = (ROOT/'attack_run.sh').read_text()
HASH = 'A'*64


def function(name, source=SHELL):
    return re.search(r'(?ms)^'+name+r'\(\) \{.*?^\}', source).group()


def initial(case, ah=3, th=3, bh=3):
    plan = profile(case)
    a = structure('5-1', '3', 'space', ah, '1-2', '9-2')
    t = structure('5-9', plan['targetType'], 'space', th, '1-6', '9-6')
    defenders = []
    for role in plan['defenders']:
        sid = {'2':'5-2', '8':'5-8', '10':'5-10', '12':'5-12', '13':'5-30'}[role['type']]
        row = structure(sid, role['type'], role['ambit'], bh, '1-6', '9-6')
        row['structAttributes']['protectedStructIndex'] = '9'
        defenders.append(row)
    t['structDefenders'] = sorted(d['Struct']['id'] for d in defenders)
    return {'attacker':a,'target':t,'defenders':defenders}


def types(case):
    def row(kind):
        p,s,c,cs,h = TYPES[kind]
        return {'StructType': {'id':kind, 'category':'fleet', 'canDefend':True, 'maxHealth':h,
            'primaryWeaponAmbits':p, 'secondaryWeaponAmbits':s, 'counterAttack':c, 'counterAttackSameAmbit':cs}}
    plan = profile(case)
    result = {'attacker':row('3'), 'target':row(plan['targetType']), 'defenders':[row(d['type']) for d in plan['defenders']]}
    result['attacker']['StructType'].update(attackCounterable=True, secondaryWeapon='attackRun', secondaryWeaponControl='unguided',
        secondaryWeaponTargets='1',secondaryWeaponShots='3',secondaryWeaponDamage='1',secondaryWeaponGuaranteedShots='1',
        secondaryWeaponShotSuccessRateNumerator='1',secondaryWeaponShotSuccessRateDenominator='3',
        secondaryWeaponBlockable=True,secondaryWeaponCounterable=True)
    return result


# Explicit counter order for these fixture ids, including lexical 5-10 before 5-2.
COUNTERS = {'A1':[],'A2':[],'A3':[], 'B1':['5-2'],'B2':[],'B3':[], 'C1':['5-30'],'C2':['5-10'],
            'D1':['5-2','5-30'],'D2':['5-10','5-30'],'D3':['5-10','5-2'],'D4':['5-10','5-2','5-30'],
            'E1':[],'E2':[],'E3':[]}


def receipt(case, start, hits=(1,0,1)):
    a,t = start['attacker']['Struct'],start['target']['Struct']
    ah,th = int(start['attacker']['structAttributes']['health']),int(start['target']['structAttributes']['health'])
    by_id = {d['Struct']['id']:d for d in start['defenders']}
    counter_ids = COUNTERS[case][:ah]
    live = ah-len(counter_ids)
    block = live > 0 and case in ('B1','D1','D3','D4')
    damage = sum(hits) if live else 0
    target_hp = th if block else max(0,th-damage)
    target_counter = live > 0 and target_hp > 0
    target_damage = (1 if case in ('E2','E3') else 2) if target_counter else 0
    attacker_hp = max(0,live-target_damage)
    states = {a['id']:(attacker_hp,attacker_hp==0),t['id']:(target_hp,target_hp==0)}
    for sid,d in by_id.items():
        hp=int(d['structAttributes']['health'])
        if block and sid=='5-2':hp=max(0,hp-damage)
        states[sid]=(hp,hp==0)
    counters=[]
    for i,sid in enumerate(counter_ids):
        d=by_id[sid]['Struct']
        counters.append({'counterByStructId':sid,'counterByStructTypeId':d['type'],
            'counterByStructLocationType':'fleet','counterByStructLocationId':'9-6',
            'counterByStructOperatingAmbit':d['operatingAmbit'],'counterDamage':'1','counterDestroyedAttacker':i+1==ah})
    shots=[]
    for i,hit in enumerate(hits if live else (0,)):
        last=i==(2 if live else 0)
        shot={'targetStructId':t['id'],'targetStructTypeId':t['type'],'targetPlayerId':'1-6',
            'targetStructLocationType':'fleet','targetStructLocationId':'9-6','targetStructOperatingAmbit':'space',
            'targetHealthBefore':str(th),'targetHealthAfter':str(target_hp if last else th),
            'targetDestroyed':last and target_hp==0,'damage':str(damage if last else 0),'damageDealt':str(hit),
            'damageReduction':'0','armourPiercing':False,'blocked':block,'evaded':False,'evadedByPlanetaryDefenses':False,
            'evadedCause':'noUnitDefenses','targetCountered':last and target_counter,
            'targetCounteredDamage':str(target_damage if last else 0),
            'targetCounterDestroyedAttacker':last and target_counter and attacker_hp==0,
            'postDestructionDamageToAttacker':False,'blockerDestroyed':last and block and states['5-2'][1],
            'eventAttackDefenderCounterDetail':counters if last else []}
        if block:
            shot.update(blockedByStructId='5-2',blockedByStructTypeId='2',blockedByStructLocationType='fleet',
                blockedByStructLocationId='9-6',blockedByStructOperatingAmbit='space',
                blockerHealthBefore=by_id['5-2']['structAttributes']['health'],
                blockerHealthAfter=str(states['5-2'][0]) if last else by_id['5-2']['structAttributes']['health'])
        shots.append(shot)
    row={'attackerStructId':a['id'],'attackerStructTypeId':'3','attackerPlayerId':'1-2',
        'attackerStructLocationType':'fleet','attackerStructLocationId':'9-2','attackerStructOperatingAmbit':'space',
        'attackerHealthBefore':str(ah),'attackerHealthAfter':str(attacker_hp),'weaponSystem':'secondaryWeapon',
        'weaponControl':'unguided','planetaryDefenseCannonDamageToAttacker':False,'recoilDamageToAttacker':False,
        'eventAttackShotDetail':shots}
    tx={'txhash':HASH,'height':'12','code':0,'events':[{'type':'structs.structs.EventAttack',
        'attributes':[{'key':'eventAttackDetail','value':json.dumps(row)}]}]}
    end=copy.deepcopy(start)
    for d in [end['attacker'],end['target']]+end['defenders']:
        hp,dead=states[d['Struct']['id']]
        d['structAttributes'].update(health=str(hp),isDestroyed=dead,isOnline=not dead)
    return tx,end


class AttackRunTests(unittest.TestCase):
    def test_recorded_native_block_uses_the_public_blocked_by_fields(self):
        # Native B1 at height 55569: BB counter 1, blocked volley 2, command
        # counter 2. This pins the wire names independently of our fixture builder.
        tx=json.loads((ROOT/'fixtures/attack_run_blocked_receipt.json').read_text())
        start={'attacker':structure('5-54','3','space',3,'1-2','9-2'),
               'target':structure('5-57','1','space',6,'1-36','9-36'),
               'defenders':[structure('5-32','2','space',3,'1-36','9-36')]}
        start['target']['structDefenders']=['5-32']
        start['defenders'][0]['structAttributes']['protectedStructIndex']='57'
        outcome=attack({'before':start,'tx':tx},'B1',tx['txhash'])
        self.assertEqual(outcome['outcomes'],{'5-54':{'health':'0','destroyed':True},
            '5-57':{'health':'6','destroyed':False},'5-32':{'health':'1','destroyed':False}})
        row=json.loads(tx['events'][0]['attributes'][0]['value'])
        row['eventAttackShotDetail'][0]['blockedByStructId']='5-99'
        tx['events'][0]['attributes'][0]['value']=json.dumps(row)
        with self.assertRaises(ValueError):attack({'before':start,'tx':tx},'B1',tx['txhash'])

    def test_all_fifteen_cases_all_random_projectiles_and_counter_death_boundaries(self):
        for case in CASES:
            self.assertEqual(catalog(types(case),case),{'verified':True})
            for ah,th,bh in ((3,3,3),(2,1,1),(1,3,3)):
                for hits in ((1,0,0),(1,0,1),(1,1,0),(1,1,1)):
                    with self.subTest(case=case, health=(ah,th,bh), hits=hits):
                        start=initial(case,ah,th,bh);tx,end=receipt(case,start,hits)
                        if ah < profile(case)['attackerMinimumHealth']:
                            with self.assertRaises(ValueError):attack({'before':start,'tx':tx},case,HASH)
                            continue
                        outcome=attack({'before':start,'tx':tx},case,HASH)
                        self.assertEqual(outcome['defenderCounters'],min(ah,len(COUNTERS[case])))
                        self.assertEqual(after({'before':start,'after':end,'outcome':outcome},case),{'verified':True})

    def test_catalog_requires_positive_and_nonparticipating_counter_roles(self):
        for case,role,key,value in [('A1','attacker','secondaryWeaponGuaranteedShots','0'),
                ('A1','attacker','secondaryWeaponShots','2'),('A1','attacker','secondaryWeaponCounterable',False),
                ('A1','target','unguidedDefensiveSuccessRateNumerator','1'),('A1','target','counterAttack','1'),
                ('B2',0,'counterAttack','1'),('B3',0,'primaryWeaponAmbits','18'),
                ('C1',0,'primaryWeaponAmbits','10'),('C2',0,'primaryWeaponAmbits','4')]:
            doc=types(case)
            row=doc['defenders'][role] if type(role) is int else doc[role]
            row['StructType'][key]=value
            with self.subTest(case=case,key=key),self.assertRaises(ValueError):catalog(doc,case)

    def test_before_requires_distinct_ready_roles_and_complete_actual_registrations(self):
        valid=initial('D4')
        changes=[lambda d:d['defenders'].pop(),lambda d:d['target'].update(structDefenders=[]),
            lambda d:d['defenders'][0]['structAttributes'].update(protectedStructIndex='99'),
            lambda d:d['defenders'][1]['Struct'].update(owner='1-2'),
            lambda d:d['defenders'][0]['Struct'].update(id='5-9'),
            lambda d:d['attacker']['structAttributes'].update(isDestroyed=True),
            lambda d:d['target']['structAttributes'].update(health='7')]
        for change in changes:
            doc=copy.deepcopy(valid);change(doc)
            with self.assertRaises(ValueError):before(doc,'D4','5-1','5-9','1-2','1-6','9-2','9-6')

    def test_receipt_must_account_for_every_projectile_counter_and_health_transition(self):
        for case in ('A1','B1','B2','B3','C1','C2','D3','D4','E2'):
            start=initial(case);tx,_=receipt(case,start)
            original=json.loads(tx['events'][0]['attributes'][0]['value'])
            mutations=[lambda r:r.update(attackerHealthAfter='99'), lambda r:r['eventAttackShotDetail'].pop(),
                lambda r:r['eventAttackShotDetail'][0].update(damageDealt='0' if case!='D4' else '1'),
                lambda r:r['eventAttackShotDetail'][0].update(targetHealthBefore='0'),
                lambda r:r['eventAttackShotDetail'][-1].update(targetCountered=not r['eventAttackShotDetail'][-1]['targetCountered']),
                lambda r:r['eventAttackShotDetail'][-1].update(damage='99'),
                lambda r:r['eventAttackShotDetail'][-1].update(eventAttackDefenderCounterDetail=[{}]),
                lambda r:r['eventAttackShotDetail'][-1].update(targetHealthAfter='99')]
            for mutate in mutations:
                row=copy.deepcopy(original);mutate(row);bad=copy.deepcopy(tx)
                bad['events'][0]['attributes'][0]['value']=json.dumps(row)
                with self.subTest(case=case),self.assertRaises(ValueError):attack({'before':start,'tx':bad},case,HASH)
            for key,value in [('code',1),('height','0'),('txhash','B'*64),('events',[])]:
                bad=copy.deepcopy(tx);bad[key]=value
                with self.assertRaises(ValueError):attack({'before':start,'tx':bad},case,HASH)
            tx['events'].append(copy.deepcopy(tx['events'][0]))
            with self.assertRaises(ValueError):attack({'before':start,'tx':tx},case,HASH)

    def test_post_state_requires_every_defender_and_rejects_a_different_outcome(self):
        start=initial('D1');tx,end=receipt('D1',start)
        outcome=attack({'before':start,'tx':tx},'D1',HASH)
        for role in ('attacker','target','defenders'):
            doc=copy.deepcopy(end)
            row=doc[role][0] if role=='defenders' else doc[role]
            row['structAttributes']['health']='99'
            with self.assertRaises(ValueError):after({'before':start,'after':doc,'outcome':outcome},'D1')
        end['defenders'].pop()
        with self.assertRaises(ValueError):after({'before':start,'after':end,'outcome':outcome},'D1')

    def test_phase_requires_fifteen_successful_cases_and_stops_on_a_failed_case(self):
        phase=SOURCE.split('section "PHASE AR3: Attack Run — Combat"',1)[1].split('fi # phase AR3',1)[0]
        for fail in ('','C1'):
            script='''set -eu
failed=$1
assert_eq() { [ "$2" = "$3" ]; }
ar_attack() { echo "$1"; [ "$1" != "$failed" ] || return 9; AR_EXECUTED=$((AR_EXECUTED+1)); }
'''+phase
            result=subprocess.run(['bash','-c',script,'phase',fail],capture_output=True,text=True,timeout=5)
            if fail:
                self.assertNotEqual(result.returncode,0)
                self.assertEqual(result.stdout.splitlines(),list(CASES)[:7])
            else:
                self.assertEqual(result.returncode,0,result.stderr)
                self.assertEqual(result.stdout.splitlines(),list(CASES))

    def test_shell_collection_never_masks_a_failed_role_query(self):
        for name,arguments,payload in [('ar_collect_state',['5-1','5-9','5-2 5-30','10'],initial('D1')['attacker']),
                                      ('ar_collect_catalog',[json.dumps(profile('D1')),'10'],types('D1')['attacker'])]:
            for fail in range(1,5):
                with tempfile.TemporaryDirectory() as directory:
                    script='''set -eu
fixture_dir=$1;fail=$2;payload=$3;shift 3
query() {
    n=0; [ ! -f "$fixture_dir/count" ] || n=$(cat "$fixture_dir/count")
    n=$((n+1));echo "$n" > "$fixture_dir/count"
    echo "$payload"; [ "$n" != "$fail" ]
}
combat_state_at_height() { query "$@"; }
'''+function(name)+f'\n{name} "$@"'
                    result=subprocess.run(['bash','-c',script,'collect',directory,str(fail),json.dumps(payload),*arguments],
                                          capture_output=True,text=True,timeout=5)
                    self.assertNotEqual(result.returncode,0,result.stderr)
                    self.assertEqual(result.stdout,'')
                    self.assertEqual((Path(directory)/'count').read_text().strip(),str(fail))


    def test_fighter_recovery_preserves_identity_and_never_steals_another_role(self):
        common=function('combat_require_struct', SOURCE)
        for mode in ('existing','missing','destroyed'):
            script='''set -eu
mode=$1
PLAYER_2_ID=1-2;PLAYER_2_FLEET_ID=9-2;PLAYER_2_PLANET_ID=2-2
CHARGE_BUILD=8;CHARGE_ACTIVATE=2;FAIL_COUNT=0
AR_P2_SF1_ID=5-10;AR_P2_SF2_ID=5-11;AR_P2_SF3_ID=5-12
[ "$mode" != missing ] || AR_P2_SF2_ID=''
get_all_structs() { echo '{"queryHeight":"10","Struct":[
 {"id":"5-10","owner":"1-2","type":"3","locationId":"9-2","operatingAmbit":"space"},
 {"id":"5-11","owner":"1-2","type":"3","locationId":"9-2","operatingAmbit":"space"},
 {"id":"5-12","owner":"1-2","type":"3","locationId":"9-2","operatingAmbit":"space"}]}'; }
combat_fixture_state() {
    if [ "$mode" = destroyed ] && [ "$1" = 5-11 ]; then echo destroyed; else echo ready; fi
}
combat_fleet_location() { echo 2-6; }
combat_move_fleet() { echo "move $*"; }
wait_for_free_slot() { echo 1; }
wait_for_charge() { :; }
run_tx() { echo "build $*"; }
get_created_struct_id() { echo 5-13; }
assert_new_struct() { :; }
'''+common+'\n'+function('ar_require_fighter')+'''
ar_require_fighter AR_P2_SF2_ID 2
echo "roles $AR_P2_SF1_ID $AR_P2_SF2_ID $AR_P2_SF3_ID"
'''
            result=subprocess.run(['bash','-c',script,'fighter',mode],capture_output=True,text=True,timeout=5)
            self.assertEqual(result.returncode,0,result.stderr)
            if mode=='destroyed':
                self.assertIn('move 2 2-2',result.stdout)
                self.assertIn('move 2 2-6',result.stdout)
                self.assertTrue(result.stdout.endswith('roles 5-10 5-13 5-12\n'))
            else:
                self.assertEqual(result.stdout,'roles 5-10 5-11 5-12\n')

    def test_head_verification_supports_bash32_and_never_masks_query_failure(self):
        script='''set -eu
height=$1;fail=$2
PLAYER_6_PLANET_ID=2-6;PLAYER_2_FLEET_ID=9-2;PLAYER_2_ID=1-2
query() {
    if [ -n "$height" ]; then [ "${5:-}" = --height ] && [ "${6:-}" = "$height" ] || return 8;
    else [ "$#" = 4 ] || return 8;fi
    if [ "$3" = planet ]; then
        echo '{"Planet":{"id":"2-6","locationListCount":"1","locationListStart":"9-2","locationListLast":"9-2"}}'
    else echo '{"Fleet":{"id":"9-2","owner":"1-2","locationId":"2-6"}}';fi
    [ "$3" != "$fail" ]
}
'''+function('ar_verify_head')+'\nar_verify_head 2 "$height"\n'
        for bash in ('bash','/bin/bash'):
            for height in ('','12'):
                for fail in ('','planet','fleet'):
                    result=subprocess.run([bash,'-c',script,'head',height,fail],capture_output=True,text=True,timeout=5)
                    with self.subTest(bash=bash,height=height,fail=fail):
                        self.assertEqual(result.returncode==0,fail=='',result.stderr)

    def test_weak_fighter_must_be_retired_before_it_can_satisfy_a_strong_case(self):
        script='''set -eu
fixture_dir=$1;reject=$2;mode=$3
PLAYER_2_FLEET_ID=9-2;AR_P2_SF1_ID=5-1
combat_require_struct() {
    if [ -f "$fixture_dir/retired" ] && [ "$mode" != stuck ]; then printf -v "$1" '%s' 5-2;
    else printf -v "$1" '%s' 5-1;fi
}
query() {
    if [ "$4" = 5-2 ]; then echo '{"Struct":{"id":"5-2"},"structAttributes":{"health":"3"}}';
    else echo '{"Struct":{"id":"5-1"},"structAttributes":{"health":"1"}}';fi
}
ar_retire_fighter() { echo "$*" >> "$fixture_dir/retired"; [ "$reject" = false ]; }
'''+function('ar_require_fighter')+'''
ar_require_fighter AR_P2_SF1_ID 2 3
echo "$AR_P2_SF1_ID"
'''
        for reject,mode,count in (('false','normal',1),('true','normal',1),('false','stuck',5)):
            with tempfile.TemporaryDirectory() as directory:
                result=subprocess.run(['bash','-c',script,'strong',directory,reject,mode],capture_output=True,text=True,timeout=5)
                self.assertEqual(result.returncode==0,reject=='false' and mode=='normal',result.stderr)
                self.assertEqual(len((Path(directory)/'retired').read_text().splitlines()),count)
                if result.returncode==0:self.assertEqual(result.stdout,'5-2\n')

    def test_command_ambit_is_corrected_before_requesting_a_space_command(self):
        script='''set -eu
GENERATOR_TEST_DIR=$1
PLAYER_2_ID=1-2;PLAYER_2_FLEET_ID=9-2;CHARGE_MOVE=3
query() {
 if [ "$3" = block-height ]; then echo '{"blockHeight":"7"}'; return; fi
 [ "$5 $6" = "--height 7" ] || return 1
 if [ "$3" = fleet ]; then echo '{"Fleet":{"id":"9-2","owner":"1-2","commandStruct":"5-1"}}';
 else echo '{"Struct":{"id":"5-1","index":"1","type":"1","owner":"1-2","locationId":"9-2","operatingAmbit":"land"},"structAttributes":{"isBuilt":true,"isOnline":true,"health":"6"}}';fi
}
combat_require_struct() { echo "require $6";printf -v "$1" '%s' 5-1; }
wait_for_charge() { :; }
run_tx() { echo "move $*"; }
'''+function('ar_require_command')+'''
ar_require_command 2
echo "command $AR_P2_CS_ID $PLAYER_2_CMD_SHIP_ID"
'''
        for player in (2,3):
            current=script.replace('PLAYER_2','PLAYER_'+str(player)).replace('1-2','1-'+str(player)).replace('9-2','9-'+str(player)).replace('AR_P2','AR_P'+str(player)).replace('ar_require_command 2','ar_require_command '+str(player))
            if player==3:current+='\n[ "$COMMAND_SHIP_ID" = "$PLAYER_3_CMD_SHIP_ID" ]\n'
            result=subprocess.run(['bash','-c',current,'command',str(ROOT)],capture_output=True,text=True,timeout=5)
            self.assertEqual(result.returncode,0,result.stderr)
            lines=result.stdout.splitlines()
            self.assertEqual(lines[0],'require land')
            self.assertIn('tx structs struct-move 5-1 fleet space --from player_'+str(player),lines[1])
            self.assertEqual(lines[2:],['require space','command 5-1 5-1'])


if __name__ == '__main__':
    unittest.main()
