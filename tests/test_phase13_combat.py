"""Phase 13 must prove intended refusals and stop damage after lethal counters."""
import copy
import json
import subprocess
import tempfile
import unittest
from pathlib import Path

from phase13_combat import PROFILES, REACH, before, catalog, verify
from test_chain_evasion import HASH, ROOT, SOURCE, structure


def evidence(mode):
    kind,ambit,weapon,control=PROFILES[mode]
    start={'attacker':structure('5-1','2','space',3,'1-2','9-2'),
           'target':structure('5-2',kind,ambit,6 if kind=='1' else 3,'1-3','9-3'),'defenders':[]}
    if mode=='unbuilt':start['target']['structAttributes'].update(isBuilt=False,isOnline=False)
    if mode=='defended':
        for sid,k,a in [('5-3','10','land'),('5-20','13','water'),('5-9','2','space'),('5-11','2','space')]:
            d=structure(sid,k,a,3,'1-3','9-3');d['structAttributes']['protectedStructIndex']='2';start['defenders'].append(d)
        start['target']['structDefenders']=sorted(d['Struct']['id'] for d in start['defenders'])
    current=copy.deepcopy(start)
    tx={'txhash':HASH,'height':'12','code':0,'codespace':'','raw_log':'','events':[]}
    if mode in ('unbuilt','range'):
        reason='unbuilt' if mode=='unbuilt' else 'out_of_range'
        tx.update(code=1,codespace='undefined',raw_log=f'target struct (5-2) is {reason} from attacker struct (5-1)')
    else:
        defended=mode=='defended';ah='0' if defended else '3';th='6' if defended else '1'
        current['attacker']['structAttributes'].update(health=ah,isDestroyed=defended,isOnline=not defended)
        current['target']['structAttributes']['health']=th
        shot={'targetStructId':'5-2','targetStructTypeId':kind,'targetPlayerId':'1-3','targetStructLocationType':'fleet',
              'targetStructLocationId':'9-3','targetStructOperatingAmbit':ambit,'targetHealthBefore':'6' if defended else '3',
              'targetHealthAfter':th,'targetDestroyed':False,'damage':'0' if defended else '2','damageDealt':'0' if defended else '2',
              'damageReduction':'0','armourPiercing':not defended,'evaded':False,'evadedCause':'noUnitDefenses',
              'evadedByPlanetaryDefenses':False,'blocked':False,'blockerDestroyed':False,'targetCountered':False,
              'targetCounteredDamage':'0','targetCounterDestroyedAttacker':False,'postDestructionDamageToAttacker':False,
              'eventAttackDefenderCounterDetail':[]}
        if defended:
            for i,d in enumerate(sorted(start['defenders'],key=lambda d:d['Struct']['id'])[:3]):
                s=d['Struct'];shot['eventAttackDefenderCounterDetail'].append({'counterByStructId':s['id'],
                    'counterByStructTypeId':s['type'],'counterByStructLocationType':'fleet','counterByStructLocationId':'9-3',
                    'counterByStructOperatingAmbit':s['operatingAmbit'],'counterDamage':'1','counterDestroyedAttacker':i==2,
                    'counterByStructWeaponSystem':'secondaryWeapon' if s['type']=='2' else 'primaryWeapon','counterByStructWeaponControl':'guided'})
        detail={'attackerStructId':'5-1','attackerStructTypeId':'2','attackerPlayerId':'1-2','attackerStructLocationType':'fleet',
                'attackerStructLocationId':'9-2','attackerStructOperatingAmbit':'space','attackerHealthBefore':'3','attackerHealthAfter':ah,
                'weaponSystem':weapon,'weaponControl':control,'recoilDamageToAttacker':False,'planetaryDefenseCannonDamageToAttacker':False,
                'eventAttackShotDetail':[shot]}
        tx['events']=[{'type':'structs.structs.EventAttack','attributes':[{'key':'eventAttackDetail','value':json.dumps(detail)}]}]
    return {'before':start,'after':current,'tx':tx}


def types(mode):
    doc=evidence(mode)['before'];out={}
    for key in ('attacker','target','defenders'):
        rows=doc[key] if key=='defenders' else [doc[key]];result=[]
        for d in rows:
            k=d['Struct']['type'];row={'id':k,'category':'fleet','canDefend':True,'maxHealth':'6' if k=='1' else '3',
                'primaryWeaponAmbits':REACH[k][0],'secondaryWeaponAmbits':REACH[k][1],'counterAttack':'2' if k=='1' else '1',
                'counterAttackSameAmbit':'2' if k=='1' else '1','attackReduction':'1' if k=='9' else '0'}
            result.append({'StructType':row})
        out[key]=result if key=='defenders' else result[0]
    _,_,weapon,control=PROFILES[mode]
    out['attacker']['StructType'].update({'attackCounterable':True,weapon+'Control':control,weapon+'Charge':'5',weapon+'Targets':'1',
        weapon+'Shots':'1',weapon+'Damage':'2',weapon+'Blockable':True,weapon+'Counterable':True,
        weapon+'ShotSuccessRateNumerator':'1',weapon+'ShotSuccessRateDenominator':'1',weapon+'ArmourPiercing':weapon=='primaryWeapon'})
    return out


class Phase13Tests(unittest.TestCase):
    def test_all_four_outcomes_and_lexical_counter_order(self):
        for mode in PROFILES:
            self.assertTrue(catalog(types(mode),mode)['verified'])
            self.assertTrue(verify(evidence(mode),mode,HASH)['verified'])
        result=verify(evidence('defended'),'defended',HASH)
        self.assertEqual(result['health'],{'5-1':'0','5-2':'6','5-3':'3','5-20':'3','5-9':'3','5-11':'3'})
        # The input list is deliberately not byte-sorted; 5-11,5-20,5-3 act, 5-9 cannot.
        doc=evidence('defended');row=json.loads(doc['tx']['events'][0]['attributes'][0]['value'])
        self.assertEqual([c['counterByStructId'] for c in row['eventAttackShotDetail'][0]['eventAttackDefenderCounterDetail']],['5-11','5-20','5-3'])

    def test_refusals_require_exact_reason_identity_delivered_height_and_unchanged_state(self):
        for mode in ('unbuilt','range'):
            for fault in ('code','hash','height','reason','identity','state','event','built'):
                doc=evidence(mode)
                if fault=='code':doc['tx']['code']=0
                elif fault=='hash':doc['tx']['txhash']='B'*64
                elif fault=='height':doc['tx']['height']='0'
                elif fault=='reason':doc['tx']['raw_log']='insufficient charge'
                elif fault=='identity':doc['tx']['raw_log']=doc['tx']['raw_log'].replace('5-2','5-99')
                elif fault=='state':doc['after']['attacker']['structAttributes']['health']='1'
                elif fault=='event':doc['tx']['events']=evidence('armour')['tx']['events']
                elif fault=='built':doc['before']['target']['structAttributes']['isBuilt']=mode=='unbuilt'
                with self.subTest(mode=mode,fault=fault),self.assertRaises(ValueError):verify(doc,mode,HASH)

    def test_piercing_and_dead_attacker_cannot_be_replaced_with_plausible_wrong_damage(self):
        for mode in ('armour','defended'):
            changes=[('damage','1'),('targetCountered',True),('targetCounteredDamage','1'),('blocked',True),('evaded',True),
                     ('targetHealthAfter','0'),('armourPiercing',mode=='defended'),('damageReduction','1')]
            for key,value in changes:
                doc=evidence(mode);row=json.loads(doc['tx']['events'][0]['attributes'][0]['value']);row['eventAttackShotDetail'][0][key]=value
                doc['tx']['events'][0]['attributes'][0]['value']=json.dumps(row)
                with self.subTest(mode=mode,key=key),self.assertRaises(ValueError):verify(doc,mode,HASH)
        for fault in ('fourth_counter','wrong_order','early_death','blocker_damage','target_damage'):
            doc=evidence('defended');row=json.loads(doc['tx']['events'][0]['attributes'][0]['value']);cs=row['eventAttackShotDetail'][0]['eventAttackDefenderCounterDetail']
            if fault=='fourth_counter':cs.append(copy.deepcopy(cs[-1]))
            elif fault=='wrong_order':cs.reverse()
            elif fault=='early_death':cs[0]['counterDestroyedAttacker']=True
            elif fault=='blocker_damage':doc['after']['defenders'][0]['structAttributes']['health']='1'
            else:doc['after']['target']['structAttributes']['health']='4'
            doc['tx']['events'][0]['attributes'][0]['value']=json.dumps(row)
            with self.subTest(fault=fault),self.assertRaises(ValueError):verify(doc,'defended',HASH)

    def test_catalog_and_fixture_require_all_four_live_registered_defenders(self):
        for fault in ('reach','missing','registration','duplicate','attacker_health','armor_health'):
            mode='armour' if fault=='armor_health' else 'defended';doc=evidence(mode)
            if fault=='reach':
                c=types(mode);c['defenders'][0]['StructType']['primaryWeaponAmbits']='8'
                with self.assertRaises(ValueError):catalog(c,mode)
                continue
            if fault=='missing':doc['before']['defenders'].pop()
            elif fault=='registration':doc['before']['defenders'][0]['structAttributes']['protectedStructIndex']='9'
            elif fault=='duplicate':doc['before']['defenders'][0]=copy.deepcopy(doc['before']['defenders'][1])
            elif fault=='attacker_health':doc['before']['attacker']['structAttributes']['health']='2'
            else:doc['before']['target']['structAttributes']['health']='1'
            with self.subTest(fault=fault),self.assertRaises(ValueError):verify(doc,mode,HASH)

    def test_shell_adapter_checks_all_receipts_and_propagates_failed_queries(self):
        helper=(ROOT/'phase13_combat.sh').read_text()
        for mode in PROFILES:
            for fault in ('none','before','after','catalog','receipt','fleet'):
                with self.subTest(mode=mode,fault=fault),tempfile.TemporaryDirectory() as directory:
                    root=Path(directory);doc=evidence(mode);cat=types(mode)
                    for name,data in [('before',doc['before']),('after',doc['after']),('tx',doc['tx']),('attacker',cat['attacker'])]:
                        (root/(name+'.json')).write_text(json.dumps(data))
                    for data in [cat['target'],*cat['defenders']]:
                        (root/(data['StructType']['id']+'.json')).write_text(json.dumps(data))
                    # Battleship target/defender and attacker share one actual catalog.
                    if (root/'2.json').exists():(root/'2.json').write_text(json.dumps(cat['attacker']))
                    script=r'''set -euo pipefail
GENERATOR_TEST_DIR="$1" fixture_dir="$2" fault="$3" HASH="$4"
PLAYER_2_ID=1-2 PLAYER_3_ID=1-3 PLAYER_2_FLEET_ID=9-2 PLAYER_3_FLEET_ID=9-3 PLAYER_2_PLANET_ID=2-1
wait_for_charge() { :; }
ar_collect_state() { local stage=after;[ "$4" != 10 ] || stage=before;cat "$fixture_dir/$stage.json";[ "$fault" != "$stage" ]; }
combat_fleet_location() { echo 2-1;[ "$fault" != fleet ]; }
query() {
 if [ "$*" = 'query structs block-height' ];then echo '{"blockHeight":"10"}'
 elif [ "$1 $2" = 'query tx' ];then cat "$fixture_dir/tx.json";[ "$fault" != receipt ]
 else
  if [ "$4" = 2 ] && [ ! -f "$fixture_dir/2.json" ];then cat "$fixture_dir/attacker.json";else cat "$fixture_dir/$4.json";fi
  [ "$fault" != catalog ]
 fi
}
run_tx() { touch "$fixture_dir/sent";LAST_TX_OUTPUT="{\"txhash\":\"$HASH\"}"; }
run_tx_expect_fail_noauto() { run_tx "$@"; }
_tx_json() { printf '%s' "$1"; }
assert_eq() { [ "$2" = "$3" ]; }
'''
                    weapon=PROFILES[mode][2]
                    result=subprocess.run(['bash','-c',script+helper+f'\np13_attack {mode} 5-1 5-2 2 3 {weapon}',
                        'test',str(ROOT),directory,fault,HASH],capture_output=True,text=True,timeout=10)
                    self.assertEqual(result.returncode==0,fault=='none',result.stdout+result.stderr)
                    if fault in ('before','catalog','fleet'):self.assertFalse((root/'sent').exists())

    def test_phase_restores_matchup_and_requires_all_four_cases(self):
        phase=SOURCE.split('section "PHASE 13: Defense Setup & Coordinated Attack"',1)[1].split('fi # phase 13',1)[0]
        setup=r'''set -euo pipefail
fault="$1"
PLAYER_2_ID=1-2 PLAYER_3_ID=1-3 PLAYER_2_FLEET_ID=9-2 PLAYER_3_FLEET_ID=9-3
PLAYER_2_PLANET_ID=2-1 PLAYER_3_PLANET_ID=2-2 CHARGE_BUILD=8
ar_require_command() { COMMAND_SHIP_ID=5-99; }
combat_move_fleet() { :; }
combat_require_struct() {
 echo "restore $*"
 [ "$fault" != setup ] || return 7
 case "$1" in
 P2_BATTLESHIP_ID) printf -v "$1" '%s' 5-1;;
 BATTLESHIP_1_ID) printf -v "$1" '%s' 5-11;;
 BATTLESHIP_2_ID) [ "$8" = 5-11 ];printf -v "$1" '%s' 5-12;;
 SAM_STRUCT_ID) printf -v "$1" '%s' 5-10;;
 SUB_STRUCT_ID) printf -v "$1" '%s' 5-13;;
 DESTROYER_STRUCT_ID) printf -v "$1" '%s' 5-9;;
 AP_TANK_ID) [ "$8" = 5-9 ];printf -v "$1" '%s' 5-20;;
 esac
}
query() { echo '{"structAttributes":{"health":"3"}}'; }
combat_clear_target_defenders() { :; }
p13_attack() { echo "attack $*";[ "$fault" != "$1" ]; }
ar_defense_set() { echo "defense $*"; }
'''
        for fault in ('none','setup','unbuilt','range','armour','defended'):
            result=subprocess.run(['bash','-c',setup+phase,'test',fault],capture_output=True,text=True,timeout=10)
            self.assertEqual(result.returncode==0,fault=='none',result.stderr)
            modes=[l.split()[1] for l in result.stdout.splitlines() if l.startswith('attack ')]
            expected=list(PROFILES) if fault in ('none','defended') else [] if fault=='setup' else list(PROFILES)[:list(PROFILES).index(fault)+1]
            self.assertEqual(modes,expected)
            if fault=='none':
                self.assertIn('player_2 pending',result.stdout)
                self.assertEqual([l for l in result.stdout.splitlines() if l.startswith('defense ')],
                    ['defense 3 5-10 5-99','defense 3 5-13 5-99','defense 3 5-11 5-99','defense 3 5-12 5-99'])
        self.assertNotIn('SKIP',phase)


if __name__=='__main__':unittest.main()
