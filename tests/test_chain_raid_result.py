"""Raid acceptance needs exact receipts and canonical state, including zero loot."""
import copy
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile
import unittest

from raid_result import after, before, receipt

ROOT = Path(__file__).resolve().parent
SOURCE = (ROOT / 'test_chain.sh').read_text()
HASH = 'A' * 64


def function(name):
    return re.search(rf'(?ms)^{name}\(\) \{{.*?^\}}', SOURCE).group()


def fixture(ore='7', require_ore=True):
    start = {
        'height': '120', 'victim_id': '1-2', 'raider_id': '1-3', 'planet_id': '2-2',
        'home_id': '2-3', 'fleet_id': '9-3', 'require_ore': require_ore,
        'victim': {'Player': {'id': '1-2', 'planetId': '2-2', 'primaryAddress': 'victim'},
                   'gridAttributes': {'ore': ore}},
        'raider': {'Player': {'id': '1-3', 'planetId': '2-3', 'fleetId': '9-3', 'primaryAddress': 'raider'},
                   'gridAttributes': {'ore': '11'}},
        'planet': {'Planet': {'id': '2-2', 'owner': '1-2', 'status': 'active',
                    'locationListStart': '9-3', 'locationListLast': '9-3', 'locationListCount': '1'},
                   'gridAttributes': {'ore': '50'}, 'planetAttributes': {
                       'blockStartRaid': '115', 'blockRaiderArrived': '100',
                       'blockStartOreMine': '90', 'blockStartOreRefine': '110',
                       'oreMiningActiveQuantity': '1', 'oreRefiningActiveQuantity': '1', 'planetaryShield': '25'}},
        'fleet': {'Fleet': {'id': '9-3', 'owner': '1-3', 'locationType': 'planet',
                            'locationId': '2-2', 'status': 'away'}},
    }
    tx = {'txhash': HASH, 'height': '150', 'code': 0, 'events': []}
    for kind, field, value in (
        ('EventOreTheft', 'eventOreTheftDetail', {'victimPlayerId': '1-2', 'victimPrimaryAddress': 'victim',
            'thiefPlayerId': '1-3', 'thiefPrimaryAddress': 'raider', 'amount': ore}),
        ('EventRaid', 'eventRaidDetail', {'fleetId': '9-3', 'planetId': '2-2', 'status': 'raidSuccessful', 'seized_ore': ore}),
        ('EventHashSuccess', 'eventHashSuccessDetail', {'callerAddress': 'raider', 'category': 'raid',
            'objectId': '9-3', 'planetId': '2-2', 'difficulty': '2'}),
    ):
        tx['events'].append({'type': 'structs.structs.' + kind,
                             'attributes': [{'key': field, 'value': json.dumps(value)}]})
    end = copy.deepcopy(start)
    end['height'] = '150'
    end['victim']['gridAttributes']['ore'] = '0'
    end['raider']['gridAttributes']['ore'] = str(11 + int(ore))
    end['fleet']['Fleet'].update(locationId='2-3', status='onStation')
    end['planet']['Planet'].update(locationListStart='', locationListLast='', locationListCount='0')
    end['planet']['planetAttributes'].update(blockStartRaid='0', blockRaiderArrived='0',
                                            blockStartOreMine='140', blockStartOreRefine='150')
    return start, tx, end


class RaidResultTests(unittest.TestCase):
    def test_exact_receipt_recall_and_both_pause_anchors(self):
        for ore in ('0', '7', '9007199254740993'):
            start, tx, end = fixture(ore, require_ore=ore != '0')
            self.assertEqual(after(start, tx, HASH, end), {'height': '150', 'seized_ore': ore})
        # Omitted protobuf zero fields remain valid, including a zero-loot receipt.
        start, tx, end = fixture('0', False)
        del start['victim']['gridAttributes']['ore']
        del end['victim']['gridAttributes']['ore']
        del end['fleet']['Fleet']['status']
        del end['planet']['Planet']['locationListCount']
        for key in ('blockStartRaid', 'blockRaiderArrived'):
            del end['planet']['planetAttributes'][key]
        for index, field in ((0, 'amount'), (1, 'seized_ore')):
            detail = json.loads(tx['events'][index]['attributes'][0]['value'])
            del detail[field]
            tx['events'][index]['attributes'][0]['value'] = json.dumps(detail)
        self.assertEqual(after(start, tx, HASH, end)['seized_ore'], '0')

    def test_invalid_baseline_identity_clocks_queue_or_ore_refuses(self):
        changes = [
            (('height',), '0'), (('height',), '114'), (('require_ore',), 'true'),
            (('victim', 'Player', 'id'), '1-99'), (('victim', 'Player', 'planetId'), '2-99'),
            (('raider', 'Player', 'fleetId'), '9-99'), (('raider', 'Player', 'planetId'), '2-99'),
            (('planet', 'Planet', 'owner'), '1-99'), (('planet', 'Planet', 'status'), 'complete'),
            (('planet', 'Planet', 'locationListLast'), '9-99'), (('planet', 'Planet', 'locationListCount'), '2'),
            (('fleet', 'Fleet', 'locationId'), '2-3'), (('fleet', 'Fleet', 'status'), 'onStation'),
            (('fleet', 'Fleet', 'locationListForward'), '9-2'), (('fleet', 'Fleet', 'locationListBackward'), '9-4'),
            (('victim', 'gridAttributes', 'ore'), '0'),
        ]
        for name in ('blockStartRaid', 'blockRaiderArrived', 'blockStartOreMine', 'blockStartOreRefine'):
            changes += [(('planet', 'planetAttributes', name), value) for value in ('0', '121', '01', None, 1, True)]
        changes += [(('victim', 'gridAttributes', 'ore'), value) for value in ('-1', '01', str(2**64), 1, True, None)]
        for path, value in changes:
            with self.subTest(path=path, value=value):
                start, _, _ = fixture()
                node = start
                for key in path[:-1]:
                    node = node[key]
                node[path[-1]] = value
                with self.assertRaises((ValueError, KeyError, TypeError)):
                    before(start)
        start, _, _ = fixture(str(2**64 - 1))
        with self.assertRaises(ValueError):
            before(start)

    def test_receipt_identity_amount_status_and_proof_target_are_exact(self):
        start, tx, _ = fixture()
        for index in range(3):
            detail = json.loads(tx['events'][index]['attributes'][0]['value'])
            for field in detail:
                for value in (None, True, 'wrong', '0'):
                    with self.subTest(index=index, field=field, value=value):
                        changed = copy.deepcopy(tx)
                        changed_detail = dict(detail, **{field: value})
                        changed['events'][index]['attributes'][0]['value'] = json.dumps(changed_detail)
                        with self.assertRaises(ValueError):
                            receipt(start, changed, HASH)
        for index in range(3):
            for duplicate in (False, True):
                changed = copy.deepcopy(tx)
                if duplicate:
                    changed['events'].append(changed['events'][index])
                else:
                    del changed['events'][index]
                with self.assertRaises(ValueError):
                    receipt(start, changed, HASH)
        for field, value in (('txhash', 'B'*64), ('height', '0'), ('height', '120'),
                             ('height', '01'), ('code', 5), ('code', False)):
            with self.assertRaises(ValueError):
                receipt(start, dict(tx, **{field: value}), HASH)

    def test_post_state_cannot_invent_ore_or_hide_queue_and_clock_errors(self):
        start, tx, end = fixture()
        changes = [(('height',), '151'), (('victim', 'gridAttributes', 'ore'), '1'),
            (('raider', 'gridAttributes', 'ore'), '19'), (('raider', 'gridAttributes', 'ore'), '17'),
            (('planet', 'gridAttributes', 'ore'), '49'), (('fleet', 'Fleet', 'locationId'), '2-2'),
            (('fleet', 'Fleet', 'status'), 'away'), (('fleet', 'Fleet', 'locationListForward'), '9-9'),
            (('fleet', 'Fleet', 'locationListBackward'), '9-9'),
            (('planet', 'Planet', 'locationListStart'), '9-3'), (('planet', 'Planet', 'locationListLast'), '9-3'),
            (('planet', 'Planet', 'locationListCount'), '1'), (('victim', 'Player', 'primaryAddress'), 'other')]
        for field in end['planet']['planetAttributes']:
            changes += [(('planet', 'planetAttributes', field), '999')]
        for path, value in changes:
            with self.subTest(path=path, value=value):
                changed = copy.deepcopy(end)
                node = changed
                for key in path[:-1]:
                    node = node[key]
                node[path[-1]] = value
                with self.assertRaises(ValueError):
                    after(start, tx, HASH, changed)

    def test_inactive_systems_keep_their_old_clocks(self):
        start, tx, end = fixture('0', False)
        for doc in (start, end):
            doc['planet']['planetAttributes'].update(oreMiningActiveQuantity='0', oreRefiningActiveQuantity='0',
                blockStartOreMine='90', blockStartOreRefine='0')
        self.assertEqual(after(start, tx, HASH, end)['seized_ore'], '0')
        end['planet']['planetAttributes']['blockStartOreMine'] = '140'
        with self.assertRaises(ValueError):
            after(start, tx, HASH, end)

    def test_snapshot_helper_queries_one_height_and_propagates_each_failure(self):
        start, _, _ = fixture()
        for fault in ('none', 'victim', 'raider', 'planet', 'fleet'):
            with tempfile.TemporaryDirectory() as directory:
                for key in ('victim', 'raider', 'planet', 'fleet'):
                    Path(directory, key).write_text(json.dumps(start[key]))
                script = '''set -euo pipefail
PLAYER_2_ID=1-2 PLAYER_3_ID=1-3 PLAYER_2_PLANET_ID=2-2 PLAYER_3_PLANET_ID=2-3 PLAYER_3_FLEET_ID=9-3
SKIP_MINING=false
query() {
    case "$*" in
        'query structs player 1-2 --height 120') key=victim;;
        'query structs player 1-3 --height 120') key=raider;;
        'query structs planet 2-2 --height 120') key=planet;;
        'query structs fleet 9-3 --height 120') key=fleet;;
        *) echo unexpected >&2; return 9;;
    esac
    cat "$work/$key"
    [ "$fault" != "$key" ]
}
'''
                env = dict(os.environ, work=directory, fault=fault)
                result = subprocess.run(['bash', '-c', script + function('raid_state_at_height') + '\nraid_state_at_height 120'],
                    env=env, capture_output=True, text=True)
                self.assertEqual(result.returncode == 0, fault == 'none', result.stderr)
                if fault == 'none':
                    self.assertEqual(json.loads(result.stdout), start)

    def test_script_validates_baseline_before_compute_and_uses_receipt_height(self):
        block = SOURCE.split('# ─── Confirmed raid outcome and exact ore accounting ───', 1)[1].split(
            '# ─── Confirmed raid outcome and exact ore accounting complete ───', 1)[0]
        for fault in ('none', 'baseline', 'tx_rpc', 'failed_tx', 'event', 'after_rpc', 'after_height', 'after_ore'):
            start, tx, end = fixture()
            if fault == 'baseline': start['planet']['Planet']['owner'] = '1-99'
            if fault == 'failed_tx': tx['code'] = 5
            if fault == 'event': tx['events'] = []
            if fault == 'after_height': end['height'] = '151'
            if fault == 'after_ore': end['raider']['gridAttributes']['ore'] = '19'
            with self.subTest(fault=fault), tempfile.TemporaryDirectory() as directory:
                for name, value in (('before', start), ('receipt', tx), ('after', end)):
                    Path(directory, name).write_text(json.dumps(value))
                script = '''set -euo pipefail
GREEN='' RED='' NC='' PASS_COUNT=0 FAIL_COUNT=0 PLAYER_3_FLEET_ID=9-3
info() { :; }
get_block_height() { echo 120; }
raid_state_at_height() {
    case "$1" in
        120) cat "$work/before";;
        150) cat "$work/after"; [ "$fault" != after_rpc ];;
        *) echo 'unexpected height' >&2; return 9;;
    esac
}
run_compute() {
    shift; [ "$*" = 'tx structs planet-raid-compute 9-3 --from player_3' ] || return 9
    touch "$work/computed"
    LAST_TX_OUTPUT="{\\"txhash\\":\\"$hash\\"}"
}
_tx_json() { printf '%s' "$1"; }
query() { [ "$*" = "query tx $hash" ] || return 9; cat "$work/receipt"; [ "$fault" != tx_rpc ]; }
'''
                script += function('jqr') + '\n' + function('assert_eq') + '\n' + block + '\n[ "$FAIL_COUNT" -eq 0 ]\n'
                env = dict(os.environ, work=directory, fault=fault, hash=HASH, GENERATOR_TEST_DIR=str(ROOT))
                result = subprocess.run(['bash', '-c', script], env=env, capture_output=True, text=True)
                self.assertEqual(result.returncode == 0, fault == 'none', result.stdout + result.stderr)
                self.assertEqual(Path(directory, 'computed').exists(), fault != 'baseline')

    def test_shield_refusals_pin_the_reason_for_each_actual_call(self):
        phase = SOURCE.split('section "PHASE 10: Planet Raid', 1)[1].split('fi # phase 10', 1)[0]
        calls = re.findall(r'run_tx_expect_reject_matching "Raid [^\n]+\n[^\n]+', phase)
        self.assertEqual(len(calls), 3)
        helpers = '\n'.join(function(name) for name in ('_tx_outcome_known', 'run_tx_expect_reject_matching'))
        for call in calls:
            expected = 'shields are not vulnerable: no active raid window' if 'fast-fails' in call else 'shields_active'
            for reason in (expected, 'work failed', 'out of gas', 'offline', 'panic ' + expected, 'wrong fleet'):
                script = '''set -euo pipefail
RED='' GREEN='' NC='' BOLD='' PASS_COUNT=0 FAIL_COUNT=0 PARAMS_TX='' PLAYER_3_FLEET_ID=9-3
info() { :; }
structsd() { printf 'rejected|%s' "$reason"; }
_tx_outcome() { printf '%s' "$1"; }
'''
                result = subprocess.run(['bash', '-c', script + helpers + '\n' + call + '\n[ "$FAIL_COUNT" -eq 0 ]'],
                    env=dict(os.environ, reason=reason), capture_output=True, text=True)
                self.assertEqual(result.returncode == 0, reason == expected, result.stdout + result.stderr)


if __name__ == '__main__':
    unittest.main()
