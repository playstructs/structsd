"""C2 requires an actual block, not an evasion or counter-killed attacker."""
import copy
import json
from pathlib import Path
import re
import subprocess
import unittest

from blocking_attack import after, attack, before, catalog
from test_chain_dead_bomber import types as duel_types, receipt as duel_receipt, HASH
from test_chain_evasion import structure

ROOT=Path(__file__).parent.resolve()
SHELL=(ROOT/'blocking_attack.sh').read_text()


def function(name):return re.search(r'(?ms)^'+name+r'\(\) \{.*?^\}',SHELL).group()


def initial(hp=3,swap=False,bhp=3):
    d={'attacker':structure('5-1','2','space',hp,'1-3','9-3'),
       'target':structure('5-2','2','space',3,'1-6','9-6'),
       'frigate':structure('5-4' if swap else '5-3','4','space',bhp,'1-6','9-6'),
       'fighter':structure('5-3' if swap else '5-4','3','space',bhp,'1-6','9-6')}
    d['target']['structDefenders']=['5-3','5-4']
    for r in ('frigate','fighter'):d[r]['structAttributes']['protectedStructIndex']='2'
    return d


def types():
    original=duel_types('sustained');bb=original['attacker'];sf=original['target'];fg=copy.deepcopy(sf)
    bb['StructType'].update(guidedDefensiveSuccessRateDenominator='3',unitDefenses='signalJamming')
    fg['StructType'].update(id='4',primaryWeaponAmbits='24',secondaryWeaponAmbits='0')
    return {'attacker':bb,'target':copy.deepcopy(bb),'frigate':fg,'fighter':sf}


def receipt(start,evaded):
    hp=int(start['attacker']['structAttributes']['health']);blocked=hp==3 and not evaded
    tx=duel_receipt('sustained',{'attacker':start['attacker'],'target':start['target']},0,3,2 if blocked else 0,0,hp==3)
    row=json.loads(tx['events'][0]['attributes'][0]['value']);shot=row['eventAttackShotDetail'][0]
    shot.update(damageDealt='2' if blocked else '0',evaded=evaded,evadedCause='signalJamming' if evaded else 'noUnitDefenses',blocked=blocked)
    ds=sorted((start['frigate'],start['fighter']),key=lambda d:d['Struct']['id'])
    shot['eventAttackDefenderCounterDetail']=[]
    for i,d in enumerate(ds[:hp]):
        u=d['Struct'];shot['eventAttackDefenderCounterDetail'].append({'counterByStructId':u['id'],'counterByStructTypeId':u['type'],
            'counterByStructLocationType':'fleet','counterByStructLocationId':'9-6','counterByStructOperatingAmbit':'space',
            'counterByStructWeaponSystem':'primaryWeapon','counterByStructWeaponControl':'guided','counterByStructActiveWeaponry':'guidedWeaponry',
            'counterDamage':'1','counterDestroyedAttacker':i+1==hp})
    if hp==3:shot.update(targetCounterWeaponSystem='secondaryWeapon',targetCounterWeaponControl='guided',targetCounterActiveWeaponry='guidedWeaponry')
    if blocked:
        d=ds[0];u=d['Struct'];bhp=int(d['structAttributes']['health'])
        shot.update(blockedByStructId=u['id'],blockedByStructTypeId=u['type'],blockedByStructLocationType='fleet',blockedByStructLocationId='9-6',
            blockedByStructOperatingAmbit='space',blockerHealthBefore=str(bhp),blockerHealthAfter=str(max(0,bhp-2)),blockerDestroyed=bhp<=2)
    tx['events'][0]['attributes'][0]['value']=json.dumps(row)
    return tx


def ending(start,o):
    end=copy.deepcopy(start)
    for role,hp in o['health'].items():end[role]['structAttributes'].update(health=hp,isDestroyed=hp=='0',isOnline=hp!='0')
    return end


class BlockingTests(unittest.TestCase):
    def test_both_defender_orders_evasion_and_lethal_counter_boundaries(self):
        self.assertTrue(catalog(types())['verified'])
        for hp in (1,2,3):
            for swap in (False,True):
                for evaded in (False,True):
                    for bhp in (1,2,3):
                        start=initial(hp,swap,bhp);o=attack({'before':start,'tx':receipt(start,evaded)},HASH)
                        with self.subTest(hp=hp,swap=swap,evaded=evaded,bhp=bhp):
                            self.assertEqual(o['blocked'],hp==3 and not evaded)
                            self.assertEqual(o['health']['attacker'],'0');self.assertEqual(o['health']['target'],'3')
                            self.assertTrue(after({'before':start,'after':ending(start,o),'outcome':o})['verified'])

    def test_wrong_catalog_associations_receipts_and_canonical_state_are_refused(self):
        for role,key,value in [('target','guidedDefensiveSuccessRateDenominator','2'),('attacker','secondaryWeaponDamage','1'),
                ('frigate','primaryWeaponAmbits','6'),('fighter','counterAttackSameAmbit','0')]:
            d=types();d[role]['StructType'][key]=value
            with self.subTest(key=key),self.assertRaises(ValueError):catalog(d)
        for fault in ('extra','order','index','owner','health','hidden'):
            d=initial()
            if fault=='extra':d['target']['structDefenders'].append('5-9')
            if fault=='order':d['target']['structDefenders'].reverse()
            if fault=='index':d['fighter']['structAttributes']['protectedStructIndex']='7'
            if fault=='owner':d['frigate']['Struct']['owner']='1-3'
            if fault=='health':d['attacker']['structAttributes']['health']='4'
            if fault=='hidden':d['target']['structAttributes']['isHidden']=True
            with self.subTest(fault=fault),self.assertRaises(ValueError):before(d,'5-1','5-2','5-3','5-4','1-3','1-6','9-3','9-6')
        start=initial()
        for level,key,value in [('row','weaponSystem','primaryWeapon'),('row','attackerHealthAfter','1'),('shot','blocked',False),
                ('shot','damage','1'),('shot','targetHealthAfter','1'),('shot','blockedByStructId','5-4'),('shot','blockerHealthAfter','3'),
                ('shot','eventAttackDefenderCounterDetail',[]),('shot','targetCountered',False),('shot','evadedCause','signalJamming')]:
            tx=receipt(start,False);row=json.loads(tx['events'][0]['attributes'][0]['value'])
            (row if level=='row' else row['eventAttackShotDetail'][0])[key]=value;tx['events'][0]['attributes'][0]['value']=json.dumps(row)
            with self.subTest(key=key),self.assertRaises(ValueError):attack({'before':start,'tx':tx},HASH)
        o=attack({'before':start,'tx':receipt(start,False)},HASH)
        for role in start:
            end=ending(start,o);end[role]['structAttributes']['health']='9'
            with self.subTest(role=role),self.assertRaises(ValueError):after({'before':start,'after':end,'outcome':o})

    def test_shell_requires_an_observed_block_and_fails_at_the_retry_bound(self):
        setup='''set -euo pipefail
fault="$1" stop_at="$2" count=0 PLAYER_3_FLEET_ID=9-3 PLAYER_6_FLEET_ID=9-6
ar_require_command() { [ "$fault" != command ]; }
combat_require_struct() { [ "$fault" != restore ] || return 1; printf -v "$1" '%s' "$3"; }
combat_clear_target_defenders() { [ "$fault" != clear ]; }
ar_defense_set() { [ "$fault" != defense ]; }
ar_park_as_head() { [ "$fault" != park ]; }
bc_attack() {
 count=$((count+1)); echo "attack $count"; [ "$fault" != attack ] || return 1
 if [ "$count" -eq "$stop_at" ];then BC_LAST_OUTCOME='{"verified":true,"blocked":true}';else BC_LAST_OUTCOME='{"verified":true,"blocked":false}';fi
}
'''
        for stop_at in (1,3,32,33):
            for fault in ('none','command','restore','clear','defense','park','attack'):
                r=subprocess.run(['bash','-c',setup+function('bc_run')+'\nbc_run','test',fault,str(stop_at)],text=True,capture_output=True,timeout=5)
                with self.subTest(stop_at=stop_at,fault=fault):
                    self.assertEqual(r.returncode==0,fault=='none' and stop_at<=32,r.stdout+r.stderr)
                    if fault=='none':self.assertEqual(len(r.stdout.splitlines()),min(stop_at,32))
        phase=(ROOT/'test_chain.sh').read_text().split('# C2: Required block',1)[1].split('# Required two-defender blocking complete',1)[0]
        self.assertIn('\nbc_run\n',phase);self.assertNotIn('SKIP',phase)

    def test_shell_execution_refuses_rpc_and_canonical_faults(self):
        setup='''set -euo pipefail
GENERATOR_TEST_DIR="$1" fault="$2" fixture_before="$3" fixture_after="$4" fixture_tx="$5" fixture_catalog="$6"
PLAYER_3_ID=1-3 PLAYER_6_ID=1-6 PLAYER_3_FLEET_ID=9-3 PLAYER_6_FLEET_ID=9-6 PLAYER_6_PLANET_ID=2-6
BATTLESHIP_2_ID=5-1 EB_P6_BATTLESHIP_ID=5-2 EB_FRIGATE_ID=5-3 EB_STARFIGHTER_ID=5-4
wait_for_charge() { [ "$fault" != charge ]; };_secondary_charge() { echo 5; }
ar_verify_head() { [ "$fault" != queue ]; };combat_fleet_location() { echo 2-6; }
bc_collect() { if [ "$1" = 40 ];then echo "$fixture_before";else echo "$fixture_after";[ "$fault" != state_rpc ] || return 1;fi; }
query() {
 case "$2 $3" in
 'structs block-height') echo '{"blockHeight":"40"}' ;;
 'structs struct-type') echo "$fixture_catalog" | jq -c --arg id "$4" '[.[] | select(.StructType.id == $id)][0]';[ "$fault" != catalog_rpc ] || return 1 ;;
 *) echo "$fixture_tx";[ "$fault" != tx_rpc ] || return 1 ;;
 esac
}
run_tx() { [ "$7" = secondaryWeapon ] || return 1;LAST_TX_OUTPUT="$fixture_tx";[ "$fault" != attack ]; }
_tx_json() { echo "$1"; };assert_eq() { [ "$2" = "$3" ]; }
'''
        for hp,evaded in ((1,False),(2,True),(3,True),(3,False)):
            start=initial(hp);tx=receipt(start,evaded);o=attack({'before':start,'tx':tx},HASH)
            for fault in ('none','charge','queue','catalog_rpc','state_rpc','tx_rpc','attack','bad_health','wrong_id'):
                end=ending(start,o)
                if fault=='bad_health':end['target']['structAttributes']['health']='2'
                if fault=='wrong_id':end['fighter']['Struct']['id']='5-8'
                r=subprocess.run(['bash','-c',setup+function('bc_attack')+'\nbc_attack','test',str(ROOT),fault,json.dumps(start),json.dumps(end),json.dumps(tx),json.dumps(types())],text=True,capture_output=True,timeout=10)
                with self.subTest(hp=hp,evaded=evaded,fault=fault):self.assertEqual(r.returncode==0,fault=='none',r.stdout+r.stderr)


if __name__=='__main__':unittest.main()
