"""Required evasion cases must restore roles and certify real, isolated outcomes."""
import copy
import json
from pathlib import Path
import re
import subprocess
import tempfile
import unittest

from evasion_attack import after, attack, before, catalog, profile

ROOT = Path(__file__).parent.resolve()
SOURCE = (ROOT/'test_chain.sh').read_text()
HASH = 'A'*64


def function(name):
    return re.search(r'(?ms)^'+name+r'\(\) \{.*?^\}', SOURCE).group()


def structure(sid, kind, ambit, health, owner, fleet):
    return {'Struct': {'id': sid, 'index': sid.split('-')[1], 'type': kind, 'operatingAmbit': ambit,
                       'owner': owner, 'locationId': fleet},
            'structAttributes': {'health': str(health), 'isBuilt': True, 'isOnline': True}}


def start(case, ah=3, th=3):
    atk, aa, tgt, ta, *_ = profile(case)
    return {'attacker': structure('5-1', atk, aa, ah, '1-6', '9-6'),
            'target': structure('5-2', tgt, ta, th, '1-3', '9-3')}


def types(case):
    atk, _, tgt, _, weapon, control, cause, _ = profile(case)
    primary, secondary = {'1': ('32','0'), '5': ('8','0'), '2': ('6','16'), '7': ('24','0'), '11': ('6','8')}[tgt]
    result = {'attacker': {'StructType': {'id': atk, 'category': 'fleet', 'attackCounterable': True,
             weapon+'Control': control, weapon+'Targets': '1', weapon+'Shots': '1', weapon+'Damage': '2',
             weapon+'Counterable': True, weapon+'ShotSuccessRateNumerator': '1', weapon+'ShotSuccessRateDenominator': '1'}},
            'target': {'StructType': {'id': tgt, 'category': 'fleet', 'unitDefenses': cause,
             control+'DefensiveSuccessRateNumerator': '2', control+'DefensiveSuccessRateDenominator': '3',
             'counterAttack': '1', 'counterAttackSameAmbit': '1', 'primaryWeaponAmbits': primary, 'secondaryWeaponAmbits': secondary}}}

    if case == 'P11':
        result['attacker']['StructType'].update(primaryWeaponAmbits='4', primaryWeaponCharge='3',
            maxHealth='3', attackReduction='1', primaryWeaponBlockable=True)
        result['target']['StructType'].update(maxHealth='6',counterAttack='2',counterAttackSameAmbit='2',
            unguidedDefensiveSuccessRateNumerator='0')
    if case == 'EB_B6':
        result['attacker']['StructType'].update(primaryWeaponAmbits='24',primaryWeaponCharge='5',maxHealth='3')
    return result


def receipt(case, evaded, ah=3, th=3):
    atk, _, tgt, ta, weapon, control, cause, counterable = profile(case)
    damage = 0 if evaded else 2
    target_after = max(0, th-damage)
    counter = counterable and target_after > 0
    counter_damage = (2 if case == 'P11' else 1) if counter else 0
    attacker_after = max(0, ah-counter_damage)
    shot = {'targetStructId': '5-2', 'targetStructTypeId': tgt, 'targetStructOperatingAmbit': ta,
            'targetPlayerId': '1-3', 'targetStructLocationType': 'fleet', 'targetStructLocationId': '9-3',
            'targetHealthBefore': str(th), 'targetHealthAfter': str(target_after), 'targetDestroyed': target_after == 0,
            'damage': str(damage), 'damageDealt': str(damage), 'damageReduction': '0', 'blocked': False, 'blockerDestroyed': False,
            'evadedByPlanetaryDefenses': False, 'evaded': evaded, 'evadedCause': cause if evaded else 'noUnitDefenses',
            'targetCountered': counter, 'targetCounteredDamage': str(counter_damage),
            'targetCounterDestroyedAttacker': counter and attacker_after == 0, 'postDestructionDamageToAttacker': False,
            'eventAttackDefenderCounterDetail': []}
    detail = {'attackerStructId': '5-1', 'attackerStructTypeId': atk, 'attackerHealthBefore': str(ah),
              'attackerPlayerId': '1-6', 'attackerStructLocationType': 'fleet', 'attackerStructLocationId': '9-6',
              'attackerStructOperatingAmbit': profile(case)[1],
              'attackerHealthAfter': str(attacker_after), 'weaponSystem': weapon, 'weaponControl': control,
              'planetaryDefenseCannonDamageToAttacker': False, 'recoilDamageToAttacker': False,
              'eventAttackShotDetail': [shot]}
    return {'txhash': HASH, 'height': '12', 'code': 0, 'events': [{'type': 'structs.structs.EventAttack',
            'attributes': [{'key': 'eventAttackDetail', 'value': json.dumps(detail)}]}]}


class EvasionTests(unittest.TestCase):
    def test_all_profiles_accept_both_rolls_and_check_counter_and_overkill_state(self):
        for case in ('EV1','EV2','EV3','EV4','EV5','EV6','EV7','EV8','EB_B6'):
            self.assertEqual(catalog(types(case), case), {'verified': True})
            for evade in (True, False):
                for ah, th in ((1,1),(1,3),(3,1),(3,3)):
                    with self.subTest(case=case, evaded=evade, ah=ah, th=th):
                        initial = start(case, ah, th)
                        outcome = attack({'before':initial,'tx':receipt(case,evade,ah,th)}, case, HASH)
                        self.assertEqual(outcome['evaded'], evade)
                        current = copy.deepcopy(initial)
                        for role in ('attacker','target'):
                            current[role]['structAttributes'].update(health=outcome[role+'Health'],
                                isDestroyed=outcome[role+'Destroyed'], isOnline=not outcome[role+'Destroyed'])
                        self.assertEqual(after({'before':initial,'after':current,'outcome':outcome},case), {'verified':True})

    def test_catalog_and_start_refuse_wrong_roles_rates_and_remaining_defenders(self):
        for role,key,value in [('attacker','id','8'),('attacker','primaryWeaponDamage','1'),
                ('attacker','primaryWeaponCounterable',False),('attacker','primaryWeaponRecoilDamage','1'),
                ('target','guidedDefensiveSuccessRateNumerator','1'),('target','unitDefenses','defensiveManeuver'),
                ('target','counterAttack','2'),('target','attackReduction','1'),('target','primaryWeaponAmbits','2')]:
            doc=types('EV1');doc[role]['StructType'][key]=value
            with self.subTest(key=key), self.assertRaises(ValueError): catalog(doc,'EV1')
        for mutate in ('defender','null_defenders','owner','type','offline','destroyed','zero_health'):
            doc=start('EV1')
            if mutate=='defender':doc['target']['structDefenders']=['5-3']
            elif mutate=='null_defenders':doc['target']['structDefenders']=None
            elif mutate=='owner':doc['target']['Struct']['owner']='1-6'
            elif mutate=='type':doc['target']['Struct']['type']='7'
            elif mutate=='offline':doc['target']['structAttributes']['isOnline']=False
            elif mutate=='destroyed':doc['target']['structAttributes']['isDestroyed']=True
            else:doc['target']['structAttributes']['health']='0'
            with self.subTest(mutate=mutate), self.assertRaises(ValueError):
                before(doc,'EV1','5-1','5-2','1-6','1-3','9-6','9-3')

    def test_event_and_canonical_mutations_cannot_count_as_evasion_coverage(self):
        for level,key,value in [('attack','attackerHealthAfter','3'),('attack','weaponControl','unguided'),
                ('attack','attackerPlayerId','1-3'),('shot','targetPlayerId','1-6'),
                ('attack','attackerStructId','5-9'),('attack','recoilDamageToAttacker',True),
                ('shot','targetHealthBefore','2'),('shot','targetHealthAfter','1'),('shot','targetDestroyed',True),
                ('shot','damage','2'),('shot','blocked',True),('shot','evadedByPlanetaryDefenses',True),
                ('shot','evadedCause','defensiveManeuver'),('shot','evaded',1),('shot','targetCountered',False),
                ('shot','targetCounteredDamage','0'),('shot','eventAttackDefenderCounterDetail',[{}])]:
            tx=receipt('EV1',True);row=json.loads(tx['events'][0]['attributes'][0]['value'])
            (row if level=='attack' else row['eventAttackShotDetail'][0])[key]=value
            tx['events'][0]['attributes'][0]['value']=json.dumps(row)
            with self.subTest(key=key), self.assertRaises(ValueError): attack({'before':start('EV1'),'tx':tx},'EV1',HASH)
        for tx in ({},dict(receipt('EV1',True),code=1),dict(receipt('EV1',True),height='0'),
                   dict(receipt('EV1',True),txhash='B'*64),dict(receipt('EV1',True),events=[])):
            with self.assertRaises((ValueError,KeyError)):attack({'before':start('EV1'),'tx':tx},'EV1',HASH)
        outcome=attack({'before':start('EV1'),'tx':receipt('EV1',True)},'EV1',HASH)
        with self.assertRaises(ValueError):after({'before':start('EV1'),'after':start('EV1'),'outcome':outcome},'EV1')

    def test_shell_attempt_requires_successful_queries_even_with_valid_error_payloads(self):
        for case, fault in [(case, fault) for case in ('EV1','EB_B6') for fault in ('none','fixture','clear','before_state','before_fleet','catalog','tx_rpc','after_state','after_fleet','wrong_state')]:
            with self.subTest(case=case,fault=fault),tempfile.TemporaryDirectory() as directory:
                root=Path(directory)
                initial=start(case);current=start(case,2 if case=='EV1' else 3,3)
                if fault=='wrong_state':current=start(case,3 if case=='EV1' else 2,3)
                for role in ('attacker','target'):
                    (root/f'{role}-10.json').write_text(json.dumps(initial[role]))
                    (root/f'{role}-12.json').write_text(json.dumps(current[role]))
                    (root/f'{role}-type.json').write_text(json.dumps(types(case)[role]))
                (root/'tx.json').write_text(json.dumps(receipt(case,True)))
                script=r'''set -euo pipefail
GENERATOR_TEST_DIR="$1" fixture_dir="$2" fault="$3" HASH="$4"
PLAYER_6_ID=1-6 PLAYER_3_ID=1-3 PLAYER_6_FLEET_ID=9-6 PLAYER_3_FLEET_ID=9-3 PLAYER_6_PLANET_ID=2-6
EV_CONFIRMED=0 EV_EVADED=0
combat_require_struct() { [ "$fault" != fixture ] || return 7; if [ "$1" = ATTACKER ];then ATTACKER=5-1;else TARGET=5-2;fi; }
combat_clear_target_defenders() { [ "$fault" != clear ]; }
wait_for_charge() { :; }
_primary_charge() { echo 5; }
combat_state_at_height() {
 local role=attacker
 if [ "$1" = 5-2 ];then role=target;fi
 cat "$fixture_dir/$role-$2.json"
 if [[ "$fault" = before_state && "$2" = 10 ]] || [[ "$fault" = after_state && "$2" = 12 ]];then return 7;fi
}
combat_fleet_location() {
 echo 2-6
 if [[ "$fault" = before_fleet && "$2" = 10 ]] || [[ "$fault" = after_fleet && "$2" = 12 ]];then return 7;fi
}
query() {
 case "$*" in
 'query structs block-height') echo '{"blockHeight":"10"}';;
 'query structs struct-type 7 --height 10'|'query structs struct-type 4 --height 10') cat "$fixture_dir/attacker-type.json";if [ "$fault" = catalog ];then return 7;fi;;
 'query structs struct-type 5 --height 10') cat "$fixture_dir/target-type.json";;
 "query tx $HASH") cat "$fixture_dir/tx.json";if [ "$fault" = tx_rpc ];then return 7;fi;;
 *) return 9;;
 esac
}
run_tx() { touch "$fixture_dir/sent"; LAST_TX_OUTPUT="{\"txhash\":\"$HASH\"}"; }
_tx_json() { printf '%s' "$1"; }
assert_eq() { [ "$2" = "$3" ]; }
'''
                kind,ambit,*_=profile(case)
                result=subprocess.run(['bash','-c',script+function('ev_attack')+f'\nev_attack {case} ATTACKER 6 {kind} {ambit} TARGET 3 5 air primaryWeapon\necho COUNT:$EV_CONFIRMED:$EB_ATTACKS:$EV_EVADED',
                    'test',str(ROOT),directory,fault,HASH],capture_output=True,text=True)
                self.assertEqual(result.returncode==0,fault=='none',result.stdout+result.stderr)
                if fault=='none':
                    self.assertIn('COUNT:1:1:1' if case=='EV1' else 'COUNT:0:1:0',result.stdout)
                    self.assertIn('EV outcome:' if case=='EV1' else 'EB outcome:',result.stdout)
                elif fault in ('fixture','clear','before_state','before_fleet','catalog'):
                    self.assertFalse((root/'sent').exists())

    def test_frigate_cross_ambit_requires_air_reach_and_forbids_pursuit_counter(self):
        for key,value in [('primaryWeaponAmbits','16'),('primaryWeaponCharge','1'),('maxHealth','6')]:
            doc=types('EB_B6');doc['attacker']['StructType'][key]=value
            with self.subTest(key=key),self.assertRaises(ValueError):catalog(doc,'EB_B6')
        for evade in (True,False):
            for key,value in [('targetCountered',True),('targetCounteredDamage','1'),('targetCounterDestroyedAttacker',True)]:
                tx=receipt('EB_B6',evade);row=json.loads(tx['events'][0]['attributes'][0]['value'])
                row['eventAttackShotDetail'][0][key]=value;tx['events'][0]['attributes'][0]['value']=json.dumps(row)
                with self.subTest(evade=evade,key=key),self.assertRaises(ValueError):attack({'before':start('EB_B6'),'tx':tx},'EB_B6',HASH)

    def test_extended_phase_requires_exact_frigate_and_pursuit_roles(self):
        phase=SOURCE.split('# B6: P6 Frigate',1)[1].split('\n',1)[1].split('# Required Frigate cross-ambit attack complete',1)[0]
        setup='''set -euo pipefail
ar_require_command() { echo "command $1"; }
ar_park_as_head() { echo "visitor $1"; }
ev_attack() { echo "$*"; }
'''
        result=subprocess.run(['bash','-c',setup+phase],text=True,capture_output=True,timeout=5)
        self.assertEqual(result.returncode,0,result.stderr)
        self.assertEqual(result.stdout.splitlines(),['command 2','command 3','command 6','visitor 3',
            'EB_B6 EB_FRIGATE_ID 6 4 space EB_PURSUIT_FIGHTER_ID 3 5 air primaryWeapon'])
        self.assertNotIn('SKIP',phase)

    def test_phase_schedules_all_eight_required_attempts(self):
        phase=SOURCE.split('section "PHASE EV1: Evasion Testing"',1)[1].split('fi # phase EV1',1)[0]
        script=r'''set -euo pipefail
PLAYER_3_FLEET_ID=9-3 PLAYER_6_FLEET_ID=9-6 PLAYER_6_PLANET_ID=2-6
info() { :; }
combat_require_struct() { printf -v "$1" '%s' 5-1; }
combat_move_fleet() { :; }
ev_attack() { echo "$*";EV_CONFIRMED=$((EV_CONFIRMED+1)); }
assert_eq() { [ "$2" = "$3" ]; }
'''
        result=subprocess.run(['bash','-c',script+phase],capture_output=True,text=True)
        self.assertEqual(result.returncode,0,result.stderr)
        self.assertEqual([line.split()[0] for line in result.stdout.splitlines()],['EV'+str(i) for i in range(1,9)])
        self.assertNotIn('eb_attack ',phase)

    def test_fleet_rebuild_visits_home_and_command_ship_uses_its_own_slot(self):
        for mode in ('fleet','command','fleet_key','move_failure','return_failure'):
            with self.subTest(mode=mode),tempfile.TemporaryDirectory() as directory:
                script=r'''set -euo pipefail
mode="$1" fixture_dir="$2" signer=player_6
if [ "$mode" = fleet_key ];then signer=fplayer_2;fi
PLAYER_6_ID=1-6 PLAYER_6_PLANET_ID=2-6 CHARGE_BUILD=8 FAIL_COUNT=0
get_all_structs() { echo '{"Struct":[],"queryHeight":"10"}'; }
combat_fleet_location() { echo 2-9; }
combat_move_fleet() {
 echo "move $*" >> "$fixture_dir/actions"
 if [[ "$mode" = move_failure && "$2" = 2-6 ]] || [[ "$mode" = return_failure && "$2" = 2-9 ]];then return 7;fi
}
combat_wait_command_slot() { echo command-slot >> "$fixture_dir/actions"; }
wait_for_free_slot() { [ "$mode" != command ] || return 9;echo normal-slot >> "$fixture_dir/actions";echo 2; }
wait_for_charge() { :; }
run_tx() { [ "${!#}" = "$signer" ];echo build >> "$fixture_dir/actions"; }
get_created_struct_id() { echo 5-7; }
assert_new_struct() { :; }
combat_fixture_state() { if [ -f "$fixture_dir/computed" ];then echo ready;else echo build;fi; }
run_compute() { [ "${!#}" = "$signer" ];touch "$fixture_dir/computed";echo compute >> "$fixture_dir/actions"; }
'''
                kind='1' if mode=='command' else '7'
                result=subprocess.run(['bash','-c',script+function('combat_require_struct')+f'\ncombat_require_struct RESULT 6 {kind} fleet 9-6 air Fixture "" "$signer"\necho "$RESULT"',
                                       'test',mode,directory],capture_output=True,text=True)
                self.assertEqual(result.returncode==0,mode in ('fleet','command','fleet_key'),result.stdout+result.stderr)
                actions=(Path(directory)/'actions').read_text().splitlines()
                signer = 'fplayer_2' if mode == 'fleet_key' else 'player_6'
                self.assertEqual(actions[0],f'move 6 2-6 {signer}')
                if mode=='move_failure':self.assertEqual(actions,[f'move 6 2-6 {signer}'])
                else:
                    self.assertEqual(actions,[f'move 6 2-6 {signer}','command-slot' if mode=='command' else 'normal-slot','build','compute',f'move 6 2-9 {signer}'])

    def test_defender_clear_requires_owner_index_and_delivery(self):
        for mode in ('ready','offline','owner','index','query_error','activation_error','clear_error'):
            with self.subTest(mode=mode),tempfile.TemporaryDirectory() as directory:
                root=Path(directory)
                target=structure('5-2','5','air',3,'1-3','9-3');target['structDefenders']=['5-3']
                defender=structure('5-3','7','air',3,'1-3','9-3');defender['structAttributes']['protectedStructIndex']='2'
                if mode=='owner':defender['Struct']['owner']='1-6'
                if mode=='index':defender['structAttributes']['protectedStructIndex']='4'
                if mode in ('offline','activation_error'):defender['structAttributes']['isOnline']=False
                (root/'target.json').write_text(json.dumps(target));(root/'defender.json').write_text(json.dumps(defender))
                target.pop('structDefenders');(root/'clear.json').write_text(json.dumps(target))
                script=r'''set -euo pipefail
GENERATOR_TEST_DIR="$1" fixture_dir="$2" mode="$3"
PLAYER_3_ID=1-3 CHARGE_ACTIVATE=2 CHARGE_DEFEND=1
query() {
 if [ "$*" = 'query structs struct 5-2' ];then
  if [ -f "$fixture_dir/cleared" ];then cat "$fixture_dir/clear.json";else cat "$fixture_dir/target.json";fi
 else cat "$fixture_dir/defender.json";fi
 [ "$mode" != query_error ]
}
wait_for_charge() { :; }
run_tx() {
 echo "$*" >> "$fixture_dir/actions"
 if [[ "$*" = *struct-activate* ]];then [ "$mode" != activation_error ] || return 7
 else [ "$mode" != clear_error ] || return 7;touch "$fixture_dir/cleared";fi
}
'''
                result=subprocess.run(['bash','-c',script+function('combat_clear_target_defenders')+'\ncombat_clear_target_defenders 3 5-2',
                    'test',str(ROOT),directory,mode],capture_output=True,text=True)
                self.assertEqual(result.returncode==0,mode in ('ready','offline'),result.stdout+result.stderr)
                if mode in ('owner','index','query_error'):self.assertFalse((root/'actions').exists())
                if mode=='activation_error':self.assertNotIn('struct-defense-clear',(root/'actions').read_text())

    def test_command_slot_wait_is_bounded_and_propagates_rpc_failure(self):
        for mode, succeeds, calls in [('empty',True,1),('swept',True,2),('occupied',False,10),('rpc',False,1)]:
            with self.subTest(mode=mode),tempfile.TemporaryDirectory() as directory:
                script=r'''set -euo pipefail
mode="$1" fixture_dir="$2"
query() {
 echo call >> "$fixture_dir/calls"
 if [ "$mode" = empty ] || { [ "$mode" = swept ] && [ -f "$fixture_dir/slept" ]; };then
  echo '{"Fleet":{"id":"9-6"}}'
 else echo '{"Fleet":{"id":"9-6","commandStruct":"5-9"}}';fi
 [ "$mode" != rpc ]
}
sleep() { touch "$fixture_dir/slept"; }
'''
                result=subprocess.run(['bash','-c',script+function('combat_wait_command_slot')+'\ncombat_wait_command_slot 9-6',
                    'test',mode,directory],capture_output=True,text=True)
                self.assertEqual(result.returncode==0,succeeds,result.stdout+result.stderr)
                self.assertEqual(len((Path(directory)/'calls').read_text().splitlines()),calls)


if __name__=='__main__':unittest.main()
