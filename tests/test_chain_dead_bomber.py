"""Dead-bomber coverage requires a lethal counter after an earlier sorted blocker."""
import copy
import json
from pathlib import Path
import re
import subprocess
import unittest

from dead_bomber import after, attack, before, catalog, ordered, range_before, range_refusal
from test_chain_evasion import structure

ROOT=Path(__file__).parent.resolve()
SHELL=(ROOT/'dead_bomber.sh').read_text()
SOURCE=(ROOT/'test_chain.sh').read_text()
HASH='A'*64


def function(name):return re.search(r'(?ms)^'+name+r'\(\) \{.*?^\}',SHELL).group()


def initial(mode,ah=3,th=3):
    if mode=='regression':
        d={'bomber':structure('5-1','6','air',1,'1-3','9-3'),
           'target':structure('5-2','8','land',th,'1-6','9-6'),
           'tank':structure('5-3','9','land',3,'1-6','9-6'),
           'hai':structure('5-4','7','air',3,'1-6','9-6')}
        d['target']['structDefenders']=['5-3','5-4']
        for r in ('tank','hai'):d[r]['structAttributes']['protectedStructIndex']='2'
        return d
    ak,aa,tk,ta=('7','air','6','air') if mode=='bomber' else ('1','land','9','land') if mode=='tank' else ('9','land','9','land') if mode=='armour' else ('1','air','7','air')
    if mode=='sustained':ak,aa,tk,ta='2','space','3','space'
    if mode=='command':ak,aa,tk,ta='1','space','3','space'
    return {'attacker':structure('5-1',ak,aa,ah,'1-6' if mode=='bomber' else '1-3','9-6' if mode=='bomber' else '9-3'),
            'target':structure('5-2',tk,ta,th,'1-3' if mode=='bomber' else '1-6','9-3' if mode=='bomber' else '9-6')}


def types(mode):
    d=initial(mode);out={}
    for role,entry in d.items():
        k=entry['Struct']['type'];row={'id':k,'category':'fleet','maxHealth':'6' if k=='1' else '3',
             'canDefend':True,'primaryWeaponAmbits':{'1':'32','2':'6','3':'16','6':'6','7':'24','8':'6','9':'4'}[k],
             'attackCounterable':True,'primaryWeaponControl':'unguided' if k=='9' else 'guided','primaryWeaponTargets':'1','primaryWeaponShots':'1',
             'primaryWeaponDamage':'2','primaryWeaponBlockable':True,'primaryWeaponCounterable':True,
             'primaryWeaponCharge':'3' if k in ('1','9') else '5','primaryWeaponShotSuccessRateNumerator':'1',
             'primaryWeaponShotSuccessRateDenominator':'1','counterAttack':'0' if k=='8' else '2' if k=='1' else '1',
             'counterAttackSameAmbit':'0' if k=='8' else '2' if k=='1' else '1','attackReduction':'1' if k=='9' else '0'}
        if k=='9':row['unitDefenses']='armour'
        if k in ('2','3'):row['secondaryWeaponAmbits']='16'
        if k=='2':
            row['guidedDefensiveSuccessRateNumerator']='2'
            for key in ('Control','Targets','Shots','Damage','Blockable','Counterable','Charge','ShotSuccessRateNumerator','ShotSuccessRateDenominator'):
                row['secondaryWeapon'+key]=row['primaryWeapon'+key]
        out[role]={'StructType':row}
    return out


def receipt(mode,start,ah_after,th_after,damage,reduction,counter):
    ar='bomber' if mode=='regression' else 'attacker';a=start[ar]['Struct'];t=start['target']['Struct']
    shot={'targetStructId':t['id'],'targetStructTypeId':t['type'],'targetPlayerId':t['owner'],
          'targetStructLocationType':'fleet','targetStructLocationId':t['locationId'],'targetStructOperatingAmbit':t['operatingAmbit'],
          'targetHealthBefore':start['target']['structAttributes']['health'],'targetHealthAfter':str(th_after),
          'targetDestroyed':th_after==0,'damage':str(damage),'damageDealt':'0' if mode=='regression' else '2',
          'damageReductionCause':'armour' if reduction else 'noUnitDefenses',
          'damageReduction':str(reduction),'armourPiercing':False,'blocked':False,'blockerDestroyed':False,'evaded':False,
          'evadedByPlanetaryDefenses':False,'targetCountered':counter,'targetCounteredDamage':'1' if counter else '0',
          'targetCounterDestroyedAttacker':counter and ah_after==0,'postDestructionDamageToAttacker':False,
          'eventAttackDefenderCounterDetail':[]}
    if mode=='regression':shot['eventAttackDefenderCounterDetail']=[{'counterByStructId':'5-4','counterByStructTypeId':'7',
          'counterByStructLocationType':'fleet','counterByStructLocationId':'9-6','counterByStructOperatingAmbit':'air',
          'counterDamage':'1','counterDestroyedAttacker':True}]
    row={'attackerStructId':a['id'],'attackerStructTypeId':a['type'],'attackerPlayerId':a['owner'],
         'attackerStructLocationType':'fleet','attackerStructLocationId':a['locationId'],'attackerStructOperatingAmbit':a['operatingAmbit'],
         'attackerHealthBefore':start[ar]['structAttributes']['health'],'attackerHealthAfter':str(ah_after),
         'weaponSystem':'secondaryWeapon' if mode=='sustained' else 'primaryWeapon','weaponControl':'unguided' if mode=='armour' else 'guided','recoilDamageToAttacker':False,
         'planetaryDefenseCannonDamageToAttacker':False,'eventAttackShotDetail':[shot]}
    return {'txhash':HASH,'height':'42','code':0,'events':[{'type':'structs.structs.EventAttack',
        'attributes':[{'key':'eventAttackDetail','value':json.dumps(row)}]}]}


def ending(start,outcome):
    end=copy.deepcopy(start)
    for role,hp in outcome['health'].items():end[role]['structAttributes'].update(health=hp,isDestroyed=hp=='0',isOnline=hp!='0')
    return end


class DeadBomberTests(unittest.TestCase):
    def test_delivered_command_range_refusal_requires_exact_reason_and_no_mutation(self):
        start=initial('command');start['attacker']['Struct']['operatingAmbit']='land'
        tx={'txhash':HASH,'height':'42','code':1,'codespace':'undefined','events':[],
            'raw_log':'target struct (5-2) is out_of_range from attacker struct (5-1)'}
        self.assertEqual(range_before(start,'5-1','5-2','1-3','1-6','9-3','9-6'),{'verified':True})
        self.assertTrue(range_refusal({'before':start,'after':copy.deepcopy(start),'tx':tx},HASH)['verified'])
        for fault in ('code','codespace','height','hash','reason','panic','event','health','ambit','defender','attacker'):
            d={'before':copy.deepcopy(start),'after':copy.deepcopy(start),'tx':copy.deepcopy(tx)}
            if fault=='code':d['tx']['code']=0
            if fault=='codespace':d['tx']['codespace']='sdk'
            if fault=='height':d['tx']['height']='0'
            if fault=='hash':d['tx']['txhash']='B'*64
            if fault=='reason':d['tx']['raw_log']='insufficient charge'
            if fault=='panic':d['tx']['raw_log']+=' panic'
            if fault=='event':d['tx']['events']=[{'type':'structs.structs.EventAttack','attributes':[{'key':'eventAttackDetail','value':'{}'}]}]
            if fault=='health':d['after']['target']['structAttributes']['health']='2'
            if fault=='ambit':d['before']['attacker']['Struct']['operatingAmbit']='space'
            if fault=='defender':d['before']['target']['structDefenders']=['5-3']
            if fault=='attacker':d['after']['attacker']['Struct']['id']='5-9'
            with self.subTest(fault=fault),self.assertRaises(ValueError):range_refusal(d,HASH)
        d=types('command');d['attacker']['StructType']['primaryWeaponAmbits']='16'
        with self.assertRaises(ValueError):catalog(d,'command')

    def test_command_range_executor_requires_delivered_refusal_and_canonical_state(self):
        start=initial('command');start['attacker']['Struct']['operatingAmbit']='land'
        tx={'txhash':HASH,'height':'42','code':1,'codespace':'undefined','events':[],
            'raw_log':'target struct (5-2) is out_of_range from attacker struct (5-1)'}
        setup='''set -euo pipefail
GENERATOR_TEST_DIR="$1" fault="$2" fixture_before="$3" fixture_after="$4" fixture_tx="$5" fixture_catalog="$6"
PLAYER_3_ID=1-3 PLAYER_6_ID=1-6 PLAYER_3_FLEET_ID=9-3 PLAYER_6_FLEET_ID=9-6 CHARGE_MOVE=3 PARAMS_TX_NOAUTO='--gas 500000'
wait_for_charge() { [ "$fault" != charge ]; }; _primary_charge() { echo 3; }
run_tx() { [ "$fault" != move ]; }
ar_verify_head() { [ "$fault" != queue ]; }
da_collect() { if [ "$4" = 40 ];then echo "$fixture_before";else echo "$fixture_after";[ "$fault" != state_rpc ] || return 1;fi; }
query() {
 case "$2 $3" in
 'structs block-height') echo '{"blockHeight":"40"}' ;;
 'structs struct-type') echo "$fixture_catalog" | jq -c --arg id "$4" '[.[] | select(.StructType.id == $id)][0]';[ "$fault" != catalog_rpc ] || return 1 ;;
 *) echo "$fixture_tx";[ "$fault" != tx_rpc ] || return 1 ;;
 esac
}
structsd() { [ "$1 $2" = '--gas 500000' ] || return 1;echo "$fixture_tx"; }
_tx_outcome() { if [ "$fault" = unknown ];then echo 'unknown|timeout';else echo 'rejected|range';fi; }
_tx_json() { echo "$1"; };assert_eq() { [ "$2" = "$3" ]; }
'''
        for fault in ('none','charge','move','queue','catalog_rpc','tx_rpc','state_rpc','unknown','bad_health','wrong_code','same_height'):
            end=copy.deepcopy(start);receipt=copy.deepcopy(tx)
            if fault=='bad_health':end['target']['structAttributes']['health']='2'
            if fault=='wrong_code':receipt['code']=3
            if fault=='same_height':receipt['height']='40'
            result=subprocess.run(['bash','-c',setup+function('da_command_range')+'\nda_command_range 5-1 5-2',
                'test',str(ROOT),fault,json.dumps(start),json.dumps(end),json.dumps(receipt),json.dumps(types('command'))],text=True,capture_output=True,timeout=10)
            with self.subTest(fault=fault):self.assertEqual(result.returncode==0,fault=='none',result.stdout+result.stderr)

    def test_command_phase_restores_roles_checks_range_and_recovers_defeated_fleet(self):
        setup='''set -euo pipefail
fault="$1" command_calls=0 park_calls=0 PLAYER_3_CMD_SHIP_ID=5-1 EB_STARFIGHTER_ID=5-2
step() { echo "$1"; [ "$fault" != "$1" ]; }
ar_require_command() { command_calls=$((command_calls+1)); step "command-$command_calls-$1"; }
ar_require_fighter() { [ "$*" = 'EB_STARFIGHTER_ID 6 3' ]; step fighter; }
combat_clear_target_defenders() { [ "$*" = '6 5-2' ]; step clear; }
ar_park_as_head() { park_calls=$((park_calls+1)); [ "$1" = 3 ]; step "park-$park_calls"; }
da_command_range() { [ "$*" = '5-1 5-2' ]; step range; }
da_execute() {
 [ "$*" = 'command 5-1 5-2 3 6' ]; step attack || return 1
 DA_LAST_OUTCOME='{"verified":true,"mode":"command","roles":{"target":"5-2"},"health":{"attacker":"0","target":"1"}}'
 if [ "$fault" = wrong_target ];then DA_LAST_OUTCOME='{}';fi
}
'''
        steps=['command-1-2','command-2-3','command-3-6','fighter','clear','park-1','range','command-4-3','attack','command-5-3','park-2']
        for fault in ['none',*steps,'wrong_target']:
            result=subprocess.run(['bash','-c',setup+function('da_command')+'\nda_command','test',fault],text=True,capture_output=True,timeout=5)
            with self.subTest(fault=fault):
                self.assertEqual(result.returncode==0,fault=='none',result.stdout+result.stderr)
                if fault=='none':self.assertEqual(result.stdout.splitlines(),steps)
                elif fault in steps:self.assertEqual(result.stdout.splitlines(),steps[:steps.index(fault)+1])
        phase=SOURCE.split('# C5: Required local-weapon',1)[1].split('# Required command local weapon complete',1)[0]
        self.assertIn('\nda_command\n',phase);self.assertNotIn('SKIP',phase)

    def test_sustained_requires_two_hits_and_refuses_damage_and_counter_mutations(self):
        for level,key,value in [('row','weaponSystem','primaryWeapon'),('shot','damage','1'),
                ('shot','targetHealthAfter','2'),('shot','targetCountered',False),
                ('shot','targetCounteredDamage','0'),('row','attackerHealthAfter','3'),
                ('shot','blocked',True),('shot','evaded',True)]:
            start=initial('sustained');tx=receipt('sustained',start,2,1,2,0,True)
            row=json.loads(tx['events'][0]['attributes'][0]['value'])
            (row if level=='row' else row['eventAttackShotDetail'][0])[key]=value
            tx['events'][0]['attributes'][0]['value']=json.dumps(row)
            with self.subTest(key=key),self.assertRaises(ValueError):attack({'before':start,'tx':tx},'sustained',HASH)
        for role,key,value in [('attacker','secondaryWeaponDamage','1'),('attacker','secondaryWeaponCharge','3'),
                ('attacker','secondaryWeaponAmbits','6'),('target','guidedDefensiveSuccessRateNumerator','1')]:
            d=types('sustained');d[role]['StructType'][key]=value
            with self.subTest(key=key),self.assertRaises(ValueError):catalog(d,'sustained')

    def test_sustained_setup_is_required_and_a_dead_attacker_does_not_end_the_case(self):
        setup='''set -euo pipefail
fault="$1" PLAYER_3_FLEET_ID=9-3 round=0
step() { echo "$1"; [ "$fault" != "$1" ]; }
ar_require_command() { step "command-$1"; }
ar_require_fighter() { [ "$*" = 'EB_STARFIGHTER_ID 6 3' ]; step fighter || return 1; EB_STARFIGHTER_ID=5-6; }
combat_clear_target_defenders() { [ "$*" = '6 5-6' ]; step clear; }
combat_require_struct() { round=$((round+1)); step "restore-$round" || return 1; BATTLESHIP_1_ID="5-$round"; }
ar_park_as_head() { [ "$1" = 3 ]; step "park-$round"; }
da_execute() {
 [ "$*" = "sustained 5-$round 5-6 3 6" ] || return 1
 step "attack-$round" || return 1
 hp=$((2-round)); target=5-6
 if [ "$fault:$round" = bad_first:1 ]; then hp=2; fi
 if [ "$fault:$round" = bad_second:2 ]; then hp=1; fi
 if [ "$fault:$round" = wrong_target:2 ]; then target=5-9; fi
 DA_LAST_OUTCOME="{\\"verified\\":true,\\"mode\\":\\"sustained\\",\\"roles\\":{\\"target\\":\\"$target\\"},\\"health\\":{\\"target\\":\\"$hp\\",\\"attacker\\":\\"0\\"}}"
}
'''
        steps=['command-2','command-3','command-6','fighter','clear','restore-1','park-1','attack-1','restore-2','park-2','attack-2']
        for fault in ['none',*steps,'bad_first','bad_second','wrong_target']:
            result=subprocess.run(['bash','-c',setup+function('da_sustained')+'\nda_sustained','test',fault],
                text=True,capture_output=True,timeout=5)
            with self.subTest(fault=fault):
                self.assertEqual(result.returncode==0,fault=='none',result.stdout+result.stderr)
                if fault=='none':self.assertEqual(result.stdout.splitlines(),steps)
                elif fault in steps:self.assertEqual(result.stdout.splitlines(),steps[:steps.index(fault)+1])
        phase=SOURCE.split('# C4: Required sustained combat',1)[1].split('# Required sustained combat complete',1)[0]
        self.assertIn('\nda_sustained\n',phase);self.assertNotIn('SKIP',phase)

    def test_armour_receipt_rejects_wrong_reduction_counter_and_defenses(self):
        faults=[('row','weaponControl','guided'),('shot','damage','2'),('shot','damageDealt','1'),
            ('shot','damageReduction','0'),('shot','damageReductionCause','noUnitDefenses'),
            ('shot','targetCountered',False),('shot','targetCounteredDamage','0'),
            ('shot','armourPiercing',True),('shot','evaded',True),('shot','blocked',True),
            ('shot','eventAttackDefenderCounterDetail',[{}])]
        for level,key,value in faults:
            start=initial('armour');tx=receipt('armour',start,2,2,1,1,True)
            row=json.loads(tx['events'][0]['attributes'][0]['value'])
            (row if level=='row' else row['eventAttackShotDetail'][0])[key]=value
            tx['events'][0]['attributes'][0]['value']=json.dumps(row)
            with self.subTest(key=key),self.assertRaises(ValueError):attack({'before':start,'tx':tx},'armour',HASH)
        for role,key,value in [('target','unitDefenses','noUnitDefenses'),('target','attackReduction','0'),
                ('attacker','primaryWeaponControl','guided'),('attacker','primaryWeaponCharge','5')]:
            d=types('armour');d[role]['StructType'][key]=value
            with self.subTest(role=role,key=key),self.assertRaises(ValueError):catalog(d,'armour')

    def test_required_armour_restores_roles_and_propagates_every_preparation_failure(self):
        setup='''set -euo pipefail
fault="$1" PLAYER_3_FLEET_ID=9-3 PLAYER_6_FLEET_ID=9-6
step() { echo "$1"; [ "$fault" != "$1" ]; }
ar_require_command() { step "command-$1"; }
combat_require_struct() { step "struct-$2" || return 1; printf -v "$1" '%s' "5-$2"; }
combat_clear_target_defenders() { [ "$*" = '6 5-6' ]; step clear; }
ar_park_as_head() { [ "$1" = 3 ]; step park; }
da_execute() { [ "$*" = 'armour 5-3 5-6 3 6' ]; step attack; }
'''
        steps=['command-2','command-3','command-6','struct-3','struct-6','clear','park','attack']
        for fault in ['none',*steps]:
            result=subprocess.run(['bash','-c',setup+function('da_armour')+'\nda_armour','test',fault],
                text=True,capture_output=True,timeout=5)
            with self.subTest(fault=fault):
                self.assertEqual(result.returncode==0,fault=='none',result.stdout+result.stderr)
                self.assertEqual(result.stdout.splitlines(),steps if fault=='none' else steps[:steps.index(fault)+1])
        phase=SOURCE.split('# C3: Required damage reduction',1)[1].split('# Required Tank armour complete',1)[0]
        self.assertIn('\nda_armour\n',phase);self.assertNotIn('SKIP',phase)

    def test_regression_and_preparations_account_for_armor_counter_and_command_death(self):
        cases=[('regression',1,3,0,3,0,0,False),('bomber',1,3,1,1,2,0,False),('bomber',3,2,3,0,2,0,False),
          ('bomber',2,1,2,0,2,0,False),('tank',6,3,5,2,1,1,True),('tank',1,3,0,2,1,1,True),
          ('tank',1,1,1,0,1,1,False),('hai',6,3,5,1,2,0,True),('hai',1,3,0,1,2,0,True),('hai',1,1,1,0,2,0,False)]
        cases += [('armour',ah,th,max(0,ah-int(th>1)),th-1,1,1,th>1) for ah in (1,2,3) for th in (1,2,3)]
        cases += [('sustained',ah,th,max(0,ah-int(th>2)),max(0,th-2),2,0,th>2) for ah in (1,2,3) for th in (1,2,3)]
        cases += [('command',ah,th,max(0,ah-int(th>2)),max(0,th-2),2,0,th>2) for ah in range(1,7) for th in (1,2,3)]
        for mode,ah,th,aa,ta,damage,reduction,counter in cases:
            with self.subTest(mode=mode,ah=ah,th=th):
                start=initial(mode,ah,th);tx=receipt(mode,start,aa,ta,damage,reduction,counter)
                self.assertEqual(catalog(types(mode),mode),{'verified':True})
                o=attack({'before':start,'tx':tx},mode,HASH)
                self.assertEqual(after({'before':start,'after':ending(start,o),'outcome':o}),{'verified':True})
                for role in o['roles']:
                    end=ending(start,o);end[role]['structAttributes']['health']='9'
                    with self.assertRaises(ValueError):after({'before':start,'after':end,'outcome':o})

    def test_order_is_lexical_and_exact_live_defender_associations_are_required(self):
        self.assertTrue(ordered('5-10','5-9'));self.assertFalse(ordered('5-9','5-10'))
        self.assertFalse(ordered('5-3','5-3'))
        for bad in ('','5-0','5-01','6-1','5-x'):
            with self.assertRaises(ValueError):ordered(bad,'5-9')
        for fault in ('health','order','extra','index','owner','hidden','offline','zero'):
            start=initial('regression')
            if fault=='health':start['bomber']['structAttributes']['health']='2'
            if fault=='order':start['target']['structDefenders'].reverse()
            if fault=='extra':start['target']['structDefenders'].append('5-9')
            if fault=='index':start['tank']['structAttributes']['protectedStructIndex']='9'
            if fault=='owner':start['hai']['Struct']['owner']='1-3'
            if fault=='hidden':start['bomber']['structAttributes']['isHidden']=True
            if fault=='offline':start['hai']['structAttributes']['isOnline']=False
            if fault=='zero':start['tank']['structAttributes']['health']='0'
            with self.subTest(fault=fault),self.assertRaises(ValueError):before(start,'regression','5-1','5-2','5-3','5-4','1-3','1-6','9-3','9-6')

    def test_wrong_reach_evasion_damage_and_early_block_are_rejected(self):
        for mode,role,key,value in [('regression','tank','primaryWeaponAmbits','12'),('regression','hai','counterAttackSameAmbit','0'),
                ('regression','target','guidedDefensiveSuccessRateNumerator','1'),('regression','bomber','primaryWeaponCharge','3'),
                ('tank','target','attackReduction','0'),('hai','attacker','primaryWeaponDamage','1')]:
            d=types(mode);d[role]['StructType'][key]=value
            with self.subTest(mode=mode,key=key),self.assertRaises(ValueError):catalog(d,mode)
        for level,key,value in [('row','attackerHealthAfter','1'),('shot','blocked',True),('shot','damage','1'),
                ('shot','damageDealt','2'),('shot','targetHealthAfter','2'),('shot','blockedByStructId','5-3'),
                ('shot','targetCountered',True),('counter','counterByStructId','5-3'),('counter','counterDamage','0'),
                ('counter','counterDestroyedAttacker',False),('counter','counterByStructTypeId','9')]:
            start=initial('regression');tx=receipt('regression',start,0,3,0,0,False);row=json.loads(tx['events'][0]['attributes'][0]['value'])
            shot=row['eventAttackShotDetail'][0];{'row':row,'shot':shot,'counter':shot['eventAttackDefenderCounterDetail'][0]}[level][key]=value
            tx['events'][0]['attributes'][0]['value']=json.dumps(row)
            with self.subTest(key=key),self.assertRaises(ValueError):attack({'before':start,'tx':tx},'regression',HASH)

    def test_shell_replaces_misordered_defenders_and_handles_decimal_boundary_with_bounded_work(self):
        setup='''set -euo pipefail
GENERATOR_TEST_DIR="$1" pairs=( $2 ) round=0 PLAYER_6_FLEET_ID=9-6
combat_require_struct() {
 if [ "$1" = EB_P6_TANK_ID ]; then pair="${pairs[$round]}";round=$((round+1));EB_P6_TANK_ID="${pair%:*}";
 else EB_P6_HAI_ID="${pair#*:}";fi
}
da_retire() { echo "retire $*"; }
'''
        for pairs,ok,expected in [('5-3:5-4',True,[]),('5-4:5-3 5-4:5-5',True,['retire hai 5-3']),
            ('5-99:5-9 5-99:5-100 5-101:5-102',True,['retire hai 5-9','retire tank 5-99','retire hai 5-100']),
            ('5-9:5-1 5-9:5-2 5-9:5-3',False,['retire hai 5-1','retire tank 5-9','retire hai 5-2'])]:
            result=subprocess.run(['bash','-c',setup+function('da_ordered_defenders')+'\nda_ordered_defenders','test',str(ROOT),pairs],text=True,capture_output=True,timeout=5)
            with self.subTest(pairs=pairs):
                self.assertEqual(result.returncode==0,ok,result.stdout+result.stderr);self.assertEqual(result.stdout.splitlines(),expected)
        phase=SOURCE.split('# C1b: A blocker sorting',1)[1].split('# Required dead-bomber ordering complete',1)[0]
        self.assertIn('\nda_run\n',phase);self.assertNotIn('SKIP',phase)

    def test_shell_bomber_preparation_is_required_for_each_health_and_refuses_query_failures(self):
        setup='''set -euo pipefail
GENERATOR_TEST_DIR="$1" fixture_hp="$2" fault="$3" PLAYER_3_ID=1-3 PLAYER_3_FLEET_ID=9-3 EB_P6_HAI_ID=5-4 CHARGE_ACTIVATE=2
combat_require_struct() { [ "$fault" != fixture ] || return 1; STEALTH_BOMBER_ID=5-1; }
query() { [ "$fault" != query ] || return 1; printf '{"Struct":{"id":"5-1"},"structAttributes":{"health":"%s","isHidden":false}}' "$fixture_hp"; }
combat_clear_target_defenders() { [ "$fault" != clear ]; }
ar_park_as_head() { [ "$fault" != queue ]; }
da_execute() {
 [ "$fault" != attack ] || return 1
 echo "attack $fixture_hp"
 if [ "$fixture_hp" = 2 ];then DA_LAST_OUTCOME='{"health":{"target":"0"}}';fixture_hp=3;
 else DA_LAST_OUTCOME='{"health":{"target":"1"}}';fi
}
'''
        for hp in ('1','2','3','bad'):
            for fault in ('none','fixture','query','clear','queue','attack'):
                result=subprocess.run(['bash','-c',setup+function('da_require_bomber')+'\nda_require_bomber','test',str(ROOT),hp,fault],text=True,capture_output=True,timeout=5)
                ok=hp in ('1','2','3') and (fault=='none' or (hp=='1' and fault=='attack'))
                with self.subTest(hp=hp,fault=fault):
                    self.assertEqual(result.returncode==0,ok,result.stdout+result.stderr)
                    if ok:self.assertEqual(result.stdout.splitlines(),[] if hp=='1' else ['attack 3'] if hp=='3' else ['attack 2','attack 3'])

    def test_actual_execution_refuses_rpc_and_canonical_faults_and_checks_command_defeat(self):
        setup='''set -euo pipefail
GENERATOR_TEST_DIR="$1" mode="$2" fault="$3" fixture_before="$4" fixture_after="$5" fixture_tx="$6" fixture_catalog="$7"
PLAYER_3_ID=1-3 PLAYER_6_ID=1-6 PLAYER_3_FLEET_ID=9-3 PLAYER_6_FLEET_ID=9-6 PLAYER_3_PLANET_ID=2-3 PLAYER_6_PLANET_ID=2-6
EB_P6_TANK_ID=5-3 EB_P6_HAI_ID=5-4
wait_for_charge() { :; }; _primary_charge() { echo 5; }; _secondary_charge() { echo 5; }
ar_verify_head() { [ "$fault" != queue ]; }
combat_fleet_location() { if [ "$1:$2:$mode" = 3:42:hai ] || [ "$1:$2:$mode" = 3:42:command ];then echo 2-3;else echo 2-6;fi; }
da_collect() { if [ "$4" = 40 ];then echo "$fixture_before";else echo "$fixture_after";[ "$fault" != state_rpc ] || return 1;fi; }
query() {
 case "$2 $3" in
 'structs block-height') echo '{"blockHeight":"40"}' ;;
 'structs struct-type') echo "$fixture_catalog" | jq -c --arg id "$4" '[.[] | select(.StructType.id == $id)][0]';[ "$fault" != catalog_rpc ] || return 1 ;;
 'structs planet') [ "$fault" != defeat_queue ] || return 1;echo '{"Planet":{"id":"2-6"}}' ;;
 *) echo "$fixture_tx";[ "$fault" != tx_rpc ] || return 1 ;;
 esac
}
run_tx() { LAST_TX_OUTPUT="$fixture_tx"; };_tx_json() { echo "$1"; };assert_eq() { [ "$2" = "$3" ]; }
'''
        for mode in ('regression','bomber','hai','armour','sustained','command'):
            start=initial(mode,1 if mode in ('hai','command') else 3,3)
            tx=receipt(mode,start,0 if mode in ('regression','hai') else 3,3 if mode=='regression' else 1,
                0 if mode=='regression' else 2,0,mode=='hai') if mode!='armour' else receipt(mode,start,2,2,1,1,True)
            if mode=='sustained':tx=receipt(mode,start,2,1,2,0,True)
            if mode=='command':tx=receipt(mode,start,0,1,2,0,True)
            o=attack({'before':start,'tx':tx},mode,HASH)
            for fault in ('none','queue','catalog_rpc','state_rpc','tx_rpc','bad_health','wrong_id','defeat_queue'):
                end=ending(start,o)
                if fault=='bad_health':end['target']['structAttributes']['health']='9'
                if fault=='wrong_id':end['target']['Struct']['id']='5-9'
                ap,tp=(6,3) if mode=='bomber' else (3,6)
                result=subprocess.run(['bash','-c',setup+function('da_execute')+f'\nda_execute {mode} 5-1 5-2 {ap} {tp}',
                    'test',str(ROOT),mode,fault,json.dumps(start),json.dumps(end),json.dumps(tx),json.dumps(types(mode))],text=True,capture_output=True,timeout=10)
                ok=fault=='none' or (fault=='defeat_queue' and mode not in ('hai','command'))
                with self.subTest(mode=mode,fault=fault):self.assertEqual(result.returncode==0,ok,result.stdout+result.stderr)


if __name__=='__main__':unittest.main()
