"""RG1 must prepare its lethal counter path and refuse unavailable or false evidence."""
import copy
import json
from pathlib import Path
import re
import subprocess
import unittest

from defender_counter_fixture import after, attack, before, catalog
from test_chain_defender_counter_kill import delivered, HASH
from test_chain_evasion import structure

ROOT=Path(__file__).parent.resolve()
SOURCE=(ROOT/'test_chain.sh').read_text()
SHELL=(ROOT/'defender_counter_fixture.sh').read_text()


def function(name,source=SHELL):
    return re.search(r'(?ms)^'+name+r'\(\) \{.*?^\}',source).group()


def initial(mode='regression'):
    d={'attacker':structure('5-1','3','space',1 if mode=='regression' else 3,'1-6','9-6'),
       'target':structure('5-2','3','space',3,'1-3','9-3'),
       'defender':structure('5-3','2','space',3,'1-3','9-3')}
    if mode=='regression':
        d['target']['structDefenders']=['5-3'];d['defender']['structAttributes']['protectedStructIndex']='2'
    return d


def types():
    sf={'id':'3','category':'fleet','maxHealth':'3','canDefend':True,'attackCounterable':True,
        'primaryWeaponControl':'guided','primaryWeaponCharge':'3','primaryWeaponAmbits':'16',
        'primaryWeaponTargets':'1','primaryWeaponShots':'1','primaryWeaponDamage':'2',
        'primaryWeaponShotSuccessRateNumerator':'1','primaryWeaponShotSuccessRateDenominator':'1',
        'primaryWeaponBlockable':True,'primaryWeaponCounterable':True,'counterAttack':'1','counterAttackSameAmbit':'1'}
    bb={'id':'2','category':'fleet','maxHealth':'3','canDefend':True,'counterAttack':'1','counterAttackSameAmbit':'1'}
    return {'fighter':{'StructType':sf},'defender':{'StructType':bb}}


def receipt(mode='regression'):
    tx=delivered();row=json.loads(tx['events'][0]['attributes'][0]['value']);shot=row['eventAttackShotDetail'][0]
    row.update(attackerPlayerId='1-6',attackerStructLocationType='fleet',attackerStructLocationId='9-6',
               attackerStructOperatingAmbit='space',weaponControl='guided')
    shot.update(targetPlayerId='1-3',targetStructLocationType='fleet',targetStructLocationId='9-3')
    shot['eventAttackDefenderCounterDetail'][0].update(counterByStructTypeId='2',counterByStructLocationType='fleet',counterByStructLocationId='9-3')
    if mode=='prepare':
        row.update(attackerStructId='5-2',attackerPlayerId='1-3',attackerStructLocationId='9-3',attackerHealthBefore='3',attackerHealthAfter='2')
        shot.update(targetStructId='5-1',targetPlayerId='1-6',targetStructLocationId='9-6',targetHealthBefore='3',targetHealthAfter='1',
            damage='2',damageDealt='2',damageReduction='0',targetCountered=True,targetCounteredDamage='1',
            targetCounterDestroyedAttacker=False,eventAttackDefenderCounterDetail=[])
    tx['events'][0]['attributes'][0]['value']=json.dumps(row)
    return tx


def final(start,outcome):
    end=copy.deepcopy(start)
    for role,hp in outcome['health'].items():end[role]['structAttributes'].update(health=hp,isDestroyed=hp=='0',isOnline=hp!='0')
    return end


class RequiredCounterFixtureTests(unittest.TestCase):
    def test_preparation_and_regression_certify_each_identity_and_health(self):
        for mode in ('prepare','regression'):
            start=initial(mode);outcome=attack({'before':start,'tx':receipt(mode)},mode,HASH)
            self.assertEqual(after({'before':start,'after':final(start,outcome),'outcome':outcome}),{'verified':True})
            for role in ('attacker','target','defender'):
                end=final(start,outcome);end[role]['structAttributes']['health']='2'
                if end[role]['structAttributes']['health']==outcome['health'][role]:end[role]['structAttributes']['health']='3'
                with self.subTest(mode=mode,role=role),self.assertRaises(ValueError):after({'before':start,'after':end,'outcome':outcome})

    def test_catalog_and_roles_refuse_unreachable_or_nonlethal_setup(self):
        self.assertEqual(catalog(types()),{'verified':True})
        for role,key,value in [('fighter','primaryWeaponDamage','1'),('fighter','primaryWeaponCounterable',False),
                ('fighter','guidedDefensiveSuccessRateNumerator','1'),('fighter','primaryWeaponShotSuccessRateDenominator','3'),
                ('defender','counterAttackSameAmbit','0'),('defender','canDefend',False)]:
            d=types();d[role]['StructType'][key]=value
            with self.subTest(key=key),self.assertRaises(ValueError):catalog(d)
        for mode in ('prepare','regression'):
            for role,key,value in [('attacker','health','2'),('target','isOnline',False),('defender','isDestroyed',True),
                ('defender','health','4'),('target','health',3)]:
                d=initial(mode);d[role]['structAttributes'][key]=value
                with self.subTest(mode=mode,key=key),self.assertRaises(ValueError):
                    before(d,mode,'5-1','5-2','5-3','1-6','1-3','9-6','9-3')
        for value in ([],['5-3','5-4'],['5-4'],None):
            d=initial();d['target']['structDefenders']=value
            with self.assertRaises(ValueError):before(d,'regression','5-1','5-2','5-3','1-6','1-3','9-6','9-3')
        d=initial('prepare');d['attacker']['structDefenders']=['5-4']
        with self.assertRaises(ValueError):before(d,'prepare','5-1','5-2','5-3','1-6','1-3','9-6','9-3')

    def test_preparation_refuses_miss_block_missing_counter_wrong_recipient_or_failed_tx(self):
        for key,value in [('damage','0'),('blocked',True),('targetCountered',False),('targetCounteredDamage','0'),
                          ('targetHealthAfter','2'),('targetStructId','5-9'),('targetPlayerId','1-3')]:
            tx=receipt('prepare');row=json.loads(tx['events'][0]['attributes'][0]['value']);row['eventAttackShotDetail'][0][key]=value
            tx['events'][0]['attributes'][0]['value']=json.dumps(row)
            with self.subTest(key=key),self.assertRaises(ValueError):attack({'before':initial('prepare'),'tx':tx},'prepare',HASH)
        for tx in ({},dict(receipt(),code=1),dict(receipt(),events=[])):
            with self.assertRaises(ValueError):attack({'before':initial(),'tx':tx},'regression',HASH)

    def test_shell_requires_regression_for_all_initial_health_values_and_fails_setup_errors(self):
        setup='''set -euo pipefail
PLAYER_3_FLEET_ID=9-3 fixture_hp="$1" fail="$2"
ar_require_command() { [ "$fail" != command ]; }
ar_require_fighter() { [ "$fail" != fighter ] || return 1; printf -v "$1" '%s' "5-$2"; }
combat_require_struct() { [ "$fail" != defender ] || return 1; printf -v "$1" '%s' 5-9; }
query() { [ "$fail" != query ] || return 1; printf '{"Struct":{"id":"5-6"},"structAttributes":{"health":"%s"}}' "$fixture_hp"; }
combat_clear_target_defenders() { [ "$fail" != clear ]; }
ar_park_as_head() { [ "$fail" != queue ]; }
ar_defense_set() { [ "$fail" != register ]; }
rg1_attack() { [ "$fail" != "$1" ] || return 1; echo "$1"; }
'''
        for hp in ('1','2','3','0','bad'):
            for fail in ('none','command','fighter','defender','query','clear','queue','register','regression','prepare'):
                result=subprocess.run(['bash','-c',setup+function('rg1_run')+'\nrg1_run','test',hp,fail],text=True,capture_output=True,timeout=5)
                ok=hp in ('1','2','3') and (fail=='none' or (fail=='prepare' and hp=='1'))
                with self.subTest(hp=hp,fail=fail):
                    self.assertEqual(result.returncode==0,ok,result.stdout+result.stderr)
                    if ok:self.assertEqual(result.stdout.splitlines(),['regression'] if hp=='1' else ['prepare','regression'])
                    self.assertNotIn('SKIP',result.stdout)
        phase=SOURCE.split('if run_phase 3450; then',1)[1].split('fi # phase RG1',1)[0]
        self.assertIn('\nrg1_run\n',phase);self.assertNotIn('SKIP',phase)

    def test_actual_attack_shell_refuses_rpc_receipt_and_canonical_faults(self):
        setup='''set -euo pipefail
GENERATOR_TEST_DIR="$1" fault="$2" fixture_before="$3" fixture_after="$4" fixture_tx="$5" fixture_types="$6"
PLAYER_6_ID=1-6 PLAYER_3_ID=1-3 PLAYER_6_FLEET_ID=9-6 PLAYER_3_FLEET_ID=9-3 PLAYER_6_PLANET_ID=2-6
RG1_ATTACKER_ID=5-1 RG1_TARGET_ID=5-2 RG1_DEFENDER_ID=5-3
wait_for_charge() { :; }; _primary_charge() { echo 3; }; ar_verify_head() { :; }; combat_fleet_location() { echo 2-6; }
run_tx() { LAST_TX_OUTPUT="$fixture_tx"; }; _tx_json() { echo "$1"; }
assert_eq() { [ "$2" = "$3" ]; }
rg1_collect_state() { if [ "$1" = 40 ]; then echo "$fixture_before"; else [ "$fault" != state_rpc ] || return 1; echo "$fixture_after"; fi; }
query() {
 case "$2 $3" in
  'structs block-height') echo '{"blockHeight":"40"}' ;;
  'structs struct-type') [ "$fault" != catalog_rpc ] || return 1; echo "$fixture_types" | jq -c --arg t "$4" 'if $t == "3" then .fighter else .defender end' ;;
  *) [ "$fault" != tx_rpc ] || return 1; echo "$fixture_tx" ;;
 esac
}
'''
        for fault in ('none','tx_rpc','catalog_rpc','state_rpc','failed_tx','missing_event','wrong_id','null_health',
                      'numeric_health','false_flag','alive_attacker','damaged_defender','damaged_target'):
            start=initial();tx=receipt();outcome=attack({'before':start,'tx':tx},'regression',HASH);end=final(start,outcome)
            if fault=='failed_tx':tx['code']=1
            if fault=='missing_event':tx['events']=[]
            if fault=='wrong_id':end['attacker']['Struct']['id']='5-99'
            if fault=='null_health':end['attacker']['structAttributes']['health']=None
            if fault=='numeric_health':end['attacker']['structAttributes']['health']=0
            if fault=='false_flag':end['attacker']['structAttributes']['isDestroyed']=False
            if fault=='alive_attacker':end['attacker']['structAttributes']['health']='1'
            if fault=='damaged_defender':end['defender']['structAttributes']['health']='2'
            if fault=='damaged_target':end['target']['structAttributes']['health']='2'
            result=subprocess.run(['bash','-c',setup+function('rg1_attack')+'\nrg1_attack regression','test',str(ROOT),fault,
                json.dumps(start),json.dumps(end),json.dumps(tx),json.dumps(types())],text=True,capture_output=True,timeout=10)
            with self.subTest(fault=fault):self.assertEqual(result.returncode==0,fault=='none',result.stdout+result.stderr)


if __name__=='__main__':unittest.main()
