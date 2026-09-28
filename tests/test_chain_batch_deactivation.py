"""Batch/deferred-build coverage may not pass on stale roles or unrelated failures."""
import copy
import json
import subprocess
import tempfile
import unittest
from pathlib import Path

from batch_deactivation import before, verify, MISSING, STAGES
from test_chain_evasion import HASH, SOURCE, ROOT, function, structure


def evidence(mode):
    initial = {}
    for key, sid, stage in zip(('first','second'), ('5-1','5-2'), STAGES[mode]):
        initial[key] = structure(sid, '2', 'space', 3, '1-3', '9-3')
        initial[key]['structAttributes'].update(status='7' if stage=='ready' else '3', isOnline=stage=='ready')
    current = copy.deepcopy(initial)
    changed = ('first','second') if mode=='offline' else (mode,) if mode in ('first','second') else ()
    events = []
    for key in changed:
        current[key]['structAttributes'].update(status='3' if mode=='offline' else '7', isOnline=mode!='offline')
        events.append({'type':'structs.structs.EventStructAttribute','attributes':[{'key':'structAttributeRecord',
            'value':json.dumps({'attributeId':'1-'+current[key]['Struct']['id'], 'value':current[key]['structAttributes']['status']})}]})
    tx = {'txhash':HASH,'height':'12','code':1 if mode=='reject' else 0,'codespace':'undefined' if mode=='reject' else '',
          'raw_log':f'failed to execute message; message index: 0: struct ({MISSING}) not found' if mode=='reject' else '', 'events':events}
    return {'before':initial,'after':current,'tx':tx}


class BatchDeactivationTests(unittest.TestCase):
    def test_all_four_steps_and_proto_omitted_false(self):
        for mode in STAGES:
            doc=evidence(mode)
            before(doc['before'],mode,'5-1','5-2','1-3','9-3')
            self.assertEqual(verify(doc,mode,HASH)['mode'],mode)
            for key in ('first','second'):
                if doc['after'][key]['structAttributes']['isOnline'] is False:
                    del doc['after'][key]['structAttributes']['isOnline']
            self.assertTrue(verify(doc,mode,HASH)['verified'])

    def test_wrong_failure_partial_commit_wrong_status_event_and_identity_refused(self):
        for mode in STAGES:
            for fault in ('hash','height','code','health','identity','online','status','events'):
                doc=evidence(mode)
                if fault=='hash':doc['tx']['txhash']='B'*64
                elif fault=='height':doc['tx']['height']='0'
                elif fault=='code':doc['tx']['code']=0 if mode=='reject' else 7
                elif fault=='health':doc['after']['first']['structAttributes']['health']='2'
                elif fault=='identity':doc['before']['first']['Struct']['owner']='1-4'
                elif fault=='online':doc['after']['second']['structAttributes']['isOnline']=None
                elif fault=='status':doc['after']['first']['structAttributes']['status']='0'
                elif fault=='events':
                    if mode=='reject':doc['tx']['events']=evidence('offline')['tx']['events']
                    else:doc['tx']['events']=[]
                with self.subTest(mode=mode,fault=fault),self.assertRaises(ValueError):verify(doc,mode,HASH)
        for reason in ('out of gas','permission denied',f'panic: struct ({MISSING}) not found','struct (5-42) not found'):
            doc=evidence('reject');doc['tx']['raw_log']=reason
            with self.assertRaises(ValueError):verify(doc,'reject',HASH)
        doc=evidence('offline');doc['after']['second']=copy.deepcopy(doc['before']['second'])
        with self.assertRaises(ValueError):verify(doc,'offline',HASH)

    def test_actual_shell_steps_propagate_rpc_and_transaction_failures(self):
        for mode in STAGES:
            for fault in ('none','before','after','tx','receipt'):
                with self.subTest(mode=mode,fault=fault),tempfile.TemporaryDirectory() as directory:
                    root=Path(directory);doc=evidence(mode)
                    for stage in ('before','after'):
                        for key in ('first','second'):(root/f'{stage}-{key}.json').write_text(json.dumps(doc[stage][key]))
                    (root/'tx.json').write_text(json.dumps(doc['tx']))
                    script=r'''set -euo pipefail
GENERATOR_TEST_DIR="$1" fixture_dir="$2" fault="$3" HASH="$4"
BATTLESHIP_1_ID=5-1 BATTLESHIP_2_ID=5-2 PLAYER_3_ID=1-3 PLAYER_3_FLEET_ID=9-3 CHARGE_ACTIVATE=2
wait_for_charge() { :; }
combat_state_at_height() {
 local stage=after role=second
 [ "$2" != 10 ] || stage=before
 [ "$1" != 5-1 ] || role=first
 cat "$fixture_dir/$stage-$role.json"
 [ "$fault" != "$stage" ]
}
query() {
 if [ "$*" = 'query structs block-height' ];then echo '{"blockHeight":"10"}'
 else cat "$fixture_dir/tx.json";[ "$fault" != receipt ];fi
}
run_tx() { touch "$fixture_dir/sent";LAST_TX_OUTPUT="{\"txhash\":\"$HASH\"}";[ "$fault" != tx ]; }
run_tx_expect_fail_noauto() { run_tx "$@"; }
_tx_json() { printf '%s' "$1"; }
assert_eq() { [ "$2" = "$3" ]; }
'''
                    result=subprocess.run(['bash','-c',script+function('p12_batch_step')+f'\np12_batch_step {mode}',
                        'test',str(ROOT),directory,fault,HASH],capture_output=True,text=True,timeout=10)
                    self.assertEqual(result.returncode==0,fault=='none',result.stdout+result.stderr)
                    if fault=='before':self.assertFalse((root/'sent').exists())
                    if fault=='none':self.assertIn('P12 outcome:',result.stdout)

    def test_required_helper_prepare_pending_and_ready_modes(self):
        for mode in ('prepare','pending','ready','invalid'):
            for initial in ('build','activate','ready'):
                with self.subTest(mode=mode,initial=initial),tempfile.TemporaryDirectory() as directory:
                    script=r'''set -euo pipefail
mode="$1" initial="$2" fixture_dir="$3"
PLAYER_3_ID=1-3 PLAYER_3_PLANET_ID=2-3 CHARGE_BUILD=8 CHARGE_ACTIVATE=2 FAIL_COUNT=0
get_all_structs() { echo '{"Struct":[{"id":"5-1","owner":"1-3","type":"2","locationId":"9-3","operatingAmbit":"space"}],"queryHeight":"10"}'; }
combat_fixture_state() {
 if [ "$1" = 5-2 ];then echo build
 elif [ -f "$fixture_dir/ready" ];then echo ready
 else echo "$initial";fi
}
combat_fleet_location() { echo 2-3; }
combat_move_fleet() { :; }
wait_for_free_slot() { echo 2; }
wait_for_charge() { :; }
run_tx() {
 echo "$*" >> "$fixture_dir/actions"
 if [[ "$*" = *struct-activate* ]];then touch "$fixture_dir/ready";fi
}
get_created_struct_id() { echo 5-2; }
assert_new_struct() { :; }
run_compute() { echo compute >> "$fixture_dir/actions";touch "$fixture_dir/ready"; }
'''
                    result=subprocess.run(['bash','-c',script+function('combat_require_struct')+'\ncombat_require_struct OUT 3 2 fleet 9-3 space Fixture "" player_3 "$mode"\necho "OUT:$OUT"',
                        'test',mode,initial,directory],capture_output=True,text=True,timeout=10)
                    self.assertEqual(result.returncode==0,mode!='invalid',result.stdout+result.stderr)
                    actions=(Path(directory)/'actions').read_text() if (Path(directory)/'actions').exists() else ''
                    if mode=='pending' and initial!='build':
                        self.assertIn('struct-build-initiate',actions);self.assertIn('OUT:5-2',result.stdout)
                    elif mode!='invalid':self.assertIn('OUT:5-1',result.stdout)
                    if mode in ('prepare','pending'):self.assertNotIn('compute',actions);self.assertNotIn('struct-activate',actions)
                    if mode=='ready' and initial=='build':self.assertIn('compute',actions)
                    if mode=='ready' and initial=='activate':self.assertIn('struct-activate',actions)

    def test_phase_defers_roles_keeps_battleships_distinct_and_executes_all_batch_steps(self):
        phase=SOURCE.split('section "PHASE 12: Complex Battle Setup"',1)[1].split('fi # phase 12',1)[0]
        script=r'''set -euo pipefail
ap_health="$1" query_failure="$2"
PLAYER_2_ID=1-2 PLAYER_3_ID=1-3 PLAYER_2_FLEET_ID=9-2 PLAYER_3_FLEET_ID=9-3
PLAYER_2_PLANET_ID=2-2 PLAYER_3_PLANET_ID=2-3 VALIDATOR_ADDRESS=validator CHARGE_BUILD=8
fund_vault_alpha() { :; }
run_tx() { if [[ "$*" = *struct-trash* ]];then echo replace-armor-target; ap_health=3; fi; }
wait_for_charge() { :; }
combat_require_command() { printf -v "$1" '%s' 5-99; }
combat_move_fleet() { echo "fleet $*"; }
combat_require_struct() {
 echo "role $*"
 if [ "$1" = BATTLESHIP_1_ID ];then printf -v "$1" '%s' 5-1
 elif [ "$1" = BATTLESHIP_2_ID ];then [ "$8" = 5-1 ];printf -v "$1" '%s' 5-2
 elif [ "$1" = AP_TANK_ID ];then [ "$8" = 5-3 ];printf -v "$1" '%s' 5-4
 else printf -v "$1" '%s' 5-3;fi
}
query() { printf '{"structAttributes":{"health":"%s"}}\n' "$ap_health";[ "$query_failure" != yes ]; }
p12_batch_step() { echo "batch $1"; }
'''
        for health, failed in [('1','no'),('2','no'),('3','no'),('0','no'),('3','yes')]:
            result=subprocess.run(['bash','-c',script+phase,'test',health,failed],capture_output=True,text=True,timeout=10)
            succeeds = health != '0' and failed == 'no'
            self.assertEqual(result.returncode == 0,succeeds,result.stderr)
            if not succeeds:
                self.assertNotIn('batch reject',result.stdout)
                continue
            self.assertIn('P13 unbuilt Battleship  player_2 pending',result.stdout)
            self.assertEqual(result.stdout.count(' player_3 prepare'),6)
            self.assertEqual('replace-armor-target' in result.stdout, health in ('1','2'))
            self.assertTrue(result.stdout.endswith('batch reject\nbatch offline\nbatch first\nbatch second\n'))
        self.assertNotIn('struct-build-initiate',phase)


if __name__ == '__main__':unittest.main()
