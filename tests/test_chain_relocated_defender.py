"""RG2 must restore its roles and prove an actual hit after defender relocation."""
import copy
import json
from pathlib import Path
import re
import subprocess
import tempfile
import unittest

from relocated_defender import attack, fixture, state

ROOT = Path(__file__).parent
SOURCE = (ROOT / 'test_chain.sh').read_text()
HASH = 'A' * 64


def structure(sid='5-2', owner='1-6', kind='14', location='2-6', health='2'):
    return {'Struct': {'id': sid, 'index': sid.split('-')[1], 'owner': owner, 'type': kind,
                       'locationId': location, 'operatingAmbit': 'land'},
            'structAttributes': {'health': health, 'isBuilt': True, 'isOnline': True, 'isDestroyed': False},
            'structDefenders': ['5-3'] if sid == '5-2' else []}


def receipt():
    shot = {'targetStructId': '5-2', 'targetStructTypeId': '14', 'targetStructOperatingAmbit': 'land',
            'targetHealthBefore': '2', 'targetHealthAfter': '0', 'targetDestroyed': True,
            'damageDealt': '2', 'damageReduction': '0', 'damage': '2', 'blocked': False,
            'blockerDestroyed': False, 'evaded': False, 'evadedByPlanetaryDefenses': False,
            'targetCountered': False, 'postDestructionDamageToAttacker': False,
            'eventAttackDefenderCounterDetail': []}
    row = {'attackerStructId': '5-1', 'attackerStructTypeId': '8', 'weaponSystem': 'primaryWeapon',
           'attackerHealthBefore': '3', 'attackerHealthAfter': '3',
           'planetaryDefenseCannonDamageToAttacker': False, 'recoilDamageToAttacker': False,
           'eventAttackShotDetail': [shot]}
    return {'txhash': HASH, 'height': '12', 'code': 0, 'events': [{'type': 'structs.structs.EventAttack',
            'attributes': [{'key': 'eventAttackDetail', 'value': json.dumps(row)}]}]}


def function(name):
    return re.search(r'(?ms)^' + name + r'\(\) \{.*?^\}', SOURCE).group()


class RelocatedDefenderTests(unittest.TestCase):
    def test_fixture_identity_lifecycle_and_canonical_receipt_state(self):
        args = ('5-2', '1-6', '14', '2-6', 'land')
        self.assertEqual(fixture(structure(), *args), 'ready')
        for flag, value, expected in [('isBuilt', False, 'build'), ('isOnline', False, 'activate'), ('isDestroyed', True, 'destroyed')]:
            doc = structure(); doc['structAttributes'][flag] = value
            self.assertEqual(fixture(doc, *args), expected)
        for group, key, value in [('Struct', 'owner', '1-3'), ('Struct', 'id', '5-9'), ('Struct', 'index', '9'),
                ('Struct', 'type', '8'), ('Struct', 'locationId', '9-6'), ('Struct', 'operatingAmbit', 'air'),
                ('structAttributes', 'health', 2), ('structAttributes', 'health', '0'),
                ('structAttributes', 'health', '-1'), ('structAttributes', 'health', '18446744073709551616'),
                ('structAttributes', 'isBuilt', 'true'), ('structAttributes', 'isOnline', None),
                ('structAttributes', 'isDestroyed', 0)]:
            doc = structure(); doc[group][key] = value
            with self.subTest(key=key, value=value), self.assertRaises(ValueError):
                fixture(doc, *args)
        doc = structure(health='0'); doc['structAttributes']['isDestroyed'] = True; doc['structAttributes']['isOnline'] = False
        self.assertEqual(state(doc, *args, '0', True), {'verified': True})
        for hp, destroyed in [('1', True), ('0', False)]:
            with self.assertRaises(ValueError): state(doc, *args, hp, destroyed)

    def test_attack_requires_exact_damage_and_no_defender_support(self):
        args = (HASH, '5-1', '5-2', '5-3', '3', '2', '3')
        self.assertEqual(attack(receipt(), *args), {'verified': True, 'height': '12', 'health': '0', 'destroyed': True})
        for health, remaining in [('6', '4'), ('1', '0')]:
            tx = receipt(); row = json.loads(tx['events'][0]['attributes'][0]['value'])
            row['eventAttackShotDetail'][0].update(targetHealthBefore=health, targetHealthAfter=remaining, targetDestroyed=remaining == '0')
            tx['events'][0]['attributes'][0]['value'] = json.dumps(row)
            self.assertEqual(attack(tx, HASH, '5-1', '5-2', '5-3', '3', health, '3'),
                             {'verified': True, 'height': '12', 'health': remaining, 'destroyed': remaining == '0'})
        for level, key, value in [('attack', 'attackerStructId', '5-9'), ('attack', 'attackerHealthAfter', '2'),
                ('attack', 'planetaryDefenseCannonDamageToAttacker', True), ('attack', 'recoilDamageToAttacker', True),
                ('shot', 'damage', '0'), ('shot', 'damageDealt', '0'), ('shot', 'targetHealthAfter', '2'),
                ('shot', 'targetStructTypeId', '8'), ('shot', 'targetDestroyed', False),
                ('shot', 'blocked', True), ('shot', 'evaded', True), ('shot', 'damageReduction', '1'),
                ('shot', 'targetCountered', True), ('shot', 'eventAttackDefenderCounterDetail', [{'counterByStructId': '5-3'}]),
                ('shot', 'blocked', 0)]:
            tx = receipt(); row = json.loads(tx['events'][0]['attributes'][0]['value'])
            (row if level == 'attack' else row['eventAttackShotDetail'][0])[key] = value
            tx['events'][0]['attributes'][0]['value'] = json.dumps(row)
            with self.subTest(key=key, value=value), self.assertRaises(ValueError): attack(tx, *args)
        for tx in ({}, dict(receipt(), code=1), dict(receipt(), height='0'), dict(receipt(), txhash='B'*64),
                   dict(receipt(), events=[]), dict(receipt(), events=receipt()['events']*2)):
            with self.assertRaises(ValueError): attack(tx, *args)

    def test_required_role_reuses_resumes_activates_or_rebuilds_without_skip(self):
        body = function('combat_require_struct')
        for mode in ('ready', 'dead', 'missing', 'pending', 'offline', 'no_slot', 'compute_error', 'activation_error', 'inventory_error', 'fixture_error'):
            with self.subTest(mode=mode), tempfile.TemporaryDirectory() as directory:
                script = r'''set -euo pipefail
PLAYER_6_ID=1-6 CHARGE_BUILD=8 CHARGE_ACTIVATE=1 FAIL_COUNT=0
mode="$1" marker="$2"
get_all_structs() {
 [ "$mode" != inventory_error ] || return 7
 if [[ "$mode" = missing || "$mode" = no_slot || "$mode" = compute_error ]]; then
  echo '{"Struct":[],"queryHeight":"10"}'
 else
  echo '{"Struct":[{"id":"5-4","owner":"1-6","type":"14","locationId":"2-6","operatingAmbit":"land"}],"queryHeight":"10"}'
 fi
}
combat_fixture_state() {
 [ "$2 $3 $4 $5" = '1-6 14 2-6 land' ] || return 9
 [ "$mode" != fixture_error ] || return 7
 if [ "$1" = 5-4 ]; then
  case "$mode" in
   dead) echo destroyed;;
   pending) if [ -f "$marker/compute" ];then echo ready;else echo build;fi;;
   offline|activation_error) if [ -f "$marker/activate" ];then echo ready;else echo activate;fi;;
   *) echo ready;;
  esac
 else
  [ "$1" = 5-7 ] || return 9
  if [ -f "$marker/compute" ];then echo ready;else echo build;fi
 fi
}
wait_for_free_slot() { [ "$*" = 'planet 2-6 land' ] || return 9; if [ "$mode" != no_slot ];then echo 0;fi; }
wait_for_charge() { :; }
run_tx() {
 echo "$*" >> "$marker/tx"
 if [[ "$*" = *struct-activate* ]] && [ "$mode" != activation_error ];then touch "$marker/activate";fi
}
get_created_struct_id() { [ "$*" = '1-6 14' ] || return 9;echo 5-7; }
assert_new_struct() { [ "$*" = 'Combat fixture Extractor 5-7 1-6 14' ]; }
run_compute() { [ "$mode" != compute_error ] || return 8; touch "$marker/compute"; }
'''
                result = subprocess.run(['bash', '-c', script + body + '\ncombat_require_struct RESULT 6 14 planet 2-6 land Extractor\necho "RESULT:$RESULT"',
                                         'test', mode, directory], text=True, capture_output=True)
                succeeds = mode in ('ready', 'dead', 'missing', 'pending', 'offline')
                self.assertEqual(result.returncode == 0, succeeds, result.stdout+result.stderr)
                self.assertNotIn('SKIP', result.stdout+result.stderr)
                if succeeds:
                    self.assertIn('RESULT:' + ('5-7' if mode in ('dead', 'missing') else '5-4'), result.stdout)
                if mode in ('ready', 'no_slot', 'inventory_error', 'fixture_error'):
                    self.assertFalse((Path(directory)/'tx').exists())

    def test_phase_requires_receipt_stale_index_and_height_pinned_canonical_states(self):
        phase = SOURCE.split('# Inspect all three roles and both fleet locations at the same committed height.', 1)[1].split('\nwait_for_charge "${PLAYER_6_ID}" "${CHARGE_DEFEND}"', 1)[0]
        self.assertNotIn('SKIP', phase)
        for fault in ('none', 'missing_index', 'wrong_defender_location', 'tx_rpc', 'failed_tx', 'no_damage', 'canonical_hp', 'canonical_identity', 'after_location'):
            with self.subTest(fault=fault), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                tx = receipt()
                if fault == 'failed_tx': tx['code'] = 1
                if fault == 'no_damage':
                    row = json.loads(tx['events'][0]['attributes'][0]['value']);row['eventAttackShotDetail'][0]['damage'] = '0'
                    tx['events'][0]['attributes'][0]['value'] = json.dumps(row)
                (root/'tx.json').write_text(json.dumps(tx))
                for sid, owner, kind, location, hp in [('5-1','1-3','8','9-3','3'), ('5-2','1-6','14','2-6','2'), ('5-3','1-6','8','9-6','3')]:
                    before = structure(sid, owner, kind, location, hp)
                    if sid == '5-2' and fault == 'missing_index': before['structDefenders'] = []
                    (root/f'{sid}-10.json').write_text(json.dumps(before))
                    after = copy.deepcopy(before)
                    if sid == '5-2': after['structAttributes'].update(health='0', isDestroyed=True, isOnline=False)
                    if sid == '5-3' and fault == 'canonical_hp': after['structAttributes']['health'] = '2'
                    if sid == '5-3' and fault == 'canonical_identity': after['Struct']['owner'] = '1-3'
                    (root/f'{sid}-12.json').write_text(json.dumps(after))
                script = r'''set -euo pipefail
GENERATOR_TEST_DIR="$1" fixture_dir="$2" fault="$3" HASH="$4"
PLAYER_3_ID=1-3 PLAYER_6_ID=1-6 PLAYER_3_FLEET_ID=9-3 PLAYER_6_FLEET_ID=9-6 PLAYER_2_PLANET_ID=2-2 PLAYER_6_PLANET_ID=2-6
RG2_ATTACKER_ID=5-1 RG2_TARGET_ID=5-2 RG2_DEFENDER_ID=5-3
query() {
 if [ "$*" = 'query structs block-height' ];then echo '{"blockHeight":"10"}';return;fi
 [ "$*" = "query tx $HASH" ] || return 9
 [ "$fault" != tx_rpc ] || return 7
 cat "$fixture_dir/tx.json"
}
combat_state_at_height() { cat "$fixture_dir/$1-$2.json"; }
combat_fleet_location() {
 if [ "$1" = 3 ];then echo 2-6
 elif [ "$fault" = wrong_defender_location ] || { [ "$fault" = after_location ] && [ "$2" = 12 ]; };then echo 2-6
 else echo 2-2;fi
}
run_tx() { touch "$fixture_dir/sent";LAST_TX_OUTPUT="{\"txhash\":\"$HASH\"}"; }
_tx_json() { printf '%s' "$1"; }
assert_eq() { [ "$2" = "$3" ]; }
'''
                result = subprocess.run(['bash','-c',script+phase,'test',str(ROOT),directory,fault,HASH], capture_output=True,text=True)
                self.assertEqual(result.returncode == 0, fault == 'none', result.stdout+result.stderr)
                if fault in ('missing_index', 'wrong_defender_location'):
                    self.assertFalse((root/'sent').exists())


if __name__ == '__main__':
    unittest.main()
