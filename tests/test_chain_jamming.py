"""JS1 must reach planetary jamming, never certify a fleet-target fallback."""
import copy
import json
from pathlib import Path
import re
import subprocess
import unittest

from jamming_attack import NETWORK, after, attack, before, catalog
from test_chain_pdc_attack import initial as pdc_initial, types as pdc_types, receipt as pdc_receipt, HASH
from test_chain_evasion import structure

ROOT = Path(__file__).parent.resolve()
SHELL = (ROOT/'jamming_attack.sh').read_text()


def function(name): return re.search(r'(?ms)^'+name+r'\(\) \{.*?^\}', SHELL).group()


def initial(ah=3, th=6, cannon=0, sh=6):
    d = pdc_initial('E10', ah, th)
    d['planet']['planetAttributes'].update(defensiveCannonQuantity=str(cannon), planetaryShield=str(37+13*cannon),
        lowOrbitBallisticsInterceptorNetworkQuantity='1', lowOrbitBallisticsInterceptorNetworkSuccessRateNumerator='1',
        lowOrbitBallisticsInterceptorNetworkSuccessRateDenominator='3')
    d['satellite'] = structure('5-3', '17', 'space', sh, '1-6', '2-6')
    d['satellite']['Struct']['locationType'] = 'planet'
    return d


def types():
    d = pdc_types('E10');d['satellite'] = {'StructType': {'id': '17', 'category': 'planet', 'maxHealth': '6',
        'buildLimit': '1', 'possibleAmbit': '16', 'planetaryShieldContribution': '12', 'planetaryDefenses': 'lowOrbitBallisticInterceptorNetwork'}}
    return d


def receipt(start):
    tx = pdc_receipt('E1' if start['planet']['planetAttributes']['defensiveCannonQuantity'] == '1' else 'E10', start)
    row = json.loads(tx['events'][0]['attributes'][0]['value'])
    row['eventAttackShotDetail'][0].update(evadedCause='noUnitDefenses', evadedByPlanetaryDefensesCause='noPlanetaryDefense')
    tx['events'][0]['attributes'][0]['value'] = json.dumps(row)
    return tx


def ending(start, o):
    d = copy.deepcopy(start)
    for role, hp in o['health'].items(): d[role]['structAttributes'].update(health=hp, isDestroyed=hp == '0', isOnline=hp != '0')
    return d


class JammingTests(unittest.TestCase):
    def test_unguided_hits_all_health_boundaries_with_or_without_cannon(self):
        self.assertTrue(catalog(types())['verified'])
        for ah in (1, 2, 3):
            for th in range(1, 7):
                for count in (0, 1):
                    for sh in (1, 6):
                        d = initial(ah, th, count, sh)
                        with self.subTest(ah=ah, th=th, count=count, sh=sh):
                            o = attack({'before': d, 'tx': receipt(d)}, HASH)
                            self.assertEqual(o['health'], {'attacker': str(ah-count), 'target': str(max(0, th-2)), 'satellite': str(sh)})
                            self.assertTrue(after({'before': d, 'after': ending(d, o), 'outcome': o})['verified'])

    def test_fleet_fallback_inactive_satellite_wrong_owner_and_wrong_catalog_fail(self):
        for role, key, value in [('attacker', 'primaryWeaponControl', 'guided'), ('target', 'id', '9'),
                ('target', 'category', 'fleet'), ('target', 'unguidedDefensiveSuccessRateNumerator', '2'),
                ('satellite', 'planetaryDefenses', 'noPlanetaryDefense'), ('satellite', 'buildLimit', '0'),
                ('satellite', 'possibleAmbit', '4'), ('satellite', 'planetaryShieldContribution', '0')]:
            d = types();d[role]['StructType'][key] = value
            with self.subTest(role=role, key=key), self.assertRaises(ValueError): catalog(d)
        for fault in ('fleet', 'type', 'sat_owner', 'planet_owner', 'planet_location', 'offline', 'destroyed', 'hidden', 'too_much_health', 'defender', 'duplicate'):
            d = initial()
            if fault == 'fleet': d['target']['Struct'].update(locationType='fleet', locationId='9-6')
            if fault == 'type': d['target']['Struct']['type'] = '8'
            if fault == 'sat_owner': d['satellite']['Struct']['owner'] = '1-3'
            if fault == 'planet_owner': d['planet']['Planet']['owner'] = '1-3'
            if fault == 'planet_location': d['satellite']['Struct']['locationId'] = '2-7'
            if fault == 'offline': d['satellite']['structAttributes']['isOnline'] = False
            if fault == 'destroyed': d['satellite']['structAttributes']['isDestroyed'] = True
            if fault == 'hidden': d['satellite']['structAttributes']['isHidden'] = True
            if fault == 'too_much_health': d['satellite']['structAttributes']['health'] = '7'
            if fault == 'defender': d['target']['structDefenders'] = ['5-4']
            if fault == 'duplicate': d['satellite'] = copy.deepcopy(d['target'])
            with self.subTest(fault=fault), self.assertRaises(ValueError): before(d, '5-1', '5-2', '5-3', '1-3', '1-6', '9-3', '2-6')
        for key, value in [(NETWORK+'Quantity', '0'), (NETWORK+'Quantity', '2'), (NETWORK+'SuccessRateNumerator', '0'),
                (NETWORK+'SuccessRateDenominator', '2'), ('advancedLowOrbitBallisticsInterceptorNetworkQuantity', '1'),
                ('defensiveCannonQuantity', '2'), ('planetaryShield', '11'), (NETWORK+'Quantity', 1),
                (NETWORK+'Quantity', None), (NETWORK+'Quantity', '01')]:
            d = initial();d['planet']['planetAttributes'][key] = value
            with self.subTest(key=key, value=value), self.assertRaises(ValueError): before(d, '5-1', '5-2', '5-3', '1-3', '1-6', '9-3', '2-6')

    def test_misses_failed_receipts_and_inconsistent_committed_state_fail(self):
        for count in (0, 1):
            start = initial(1, 1, count);tx = receipt(start)
            for level, key, value in [('row', 'weaponControl', 'guided'), ('row', 'attackerHealthAfter', '9'),
                    ('row', 'planetaryDefenseCannonDamage', '2'), ('shot', 'targetStructLocationType', 'fleet'),
                    ('shot', 'targetHealthAfter', '1'), ('shot', 'targetDestroyed', False), ('shot', 'damageDealt', '0'),
                    ('shot', 'evaded', True), ('shot', 'evadedByPlanetaryDefenses', True),
                    ('shot', 'evadedByPlanetaryDefensesCause', 'lowOrbitBallisticInterceptorNetwork'),
                    ('shot', 'targetCountered', True), ('shot', 'eventAttackDefenderCounterDetail', [{}])]:
                bad = copy.deepcopy(tx);row = json.loads(bad['events'][0]['attributes'][0]['value'])
                (row if level == 'row' else row['eventAttackShotDetail'][0])[key] = value
                bad['events'][0]['attributes'][0]['value'] = json.dumps(row)
                with self.subTest(count=count, key=key), self.assertRaises(ValueError): attack({'before': start, 'tx': bad}, HASH)
            for bad in [dict(tx, code=1), dict(tx, height='0'), dict(tx, txhash='B'*64), dict(tx, events=[]), dict(tx, events=tx['events']*2)]:
                with self.assertRaises(ValueError): attack({'before': start, 'tx': bad}, HASH)
            o = attack({'before': start, 'tx': tx}, HASH)
            for fault in ('health', 'online_rubble', 'satellite', 'network', 'shield', 'count', 'omitted_role', 'wrong_role'):
                end = ending(start, o);outcome = copy.deepcopy(o)
                if fault == 'health': end['target']['structAttributes']['health'] = '1'
                if fault == 'online_rubble': end['target']['structAttributes']['isOnline'] = True
                if fault == 'satellite': end['satellite']['structAttributes']['isOnline'] = False
                if fault == 'network': end['planet']['planetAttributes'][NETWORK+'Quantity'] = '0'
                if fault == 'shield': end['planet']['planetAttributes']['planetaryShield'] = '99'
                if fault == 'count': end['planet']['planetAttributes']['defensiveCannonQuantity'] = str(1-count)
                if fault == 'omitted_role': outcome['health'].pop('satellite')
                if fault == 'wrong_role': outcome['roles']['satellite'] = '5-7'
                with self.subTest(count=count, fault=fault), self.assertRaises(ValueError): after({'before': start, 'after': end, 'outcome': outcome})

    def test_shell_execution_requires_exact_receipt_network_and_canonical_state(self):
        setup = '''set -euo pipefail
GENERATOR_TEST_DIR="$1" fault="$2" fixture_before="$3" fixture_after="$4" fixture_tx="$5" fixture_catalog="$6"
PLAYER_3_ID=1-3 PLAYER_6_ID=1-6 PLAYER_3_FLEET_ID=9-3 PLAYER_6_PLANET_ID=2-6
JS_ATTACKER_ID=5-1 JS_TARGET_ID=5-2 JS_SAT_ID=5-3
combat_clear_target_defenders() { [ "$fault" != clear ]; }
ar_park_as_head() { [ "$fault" != park ]; };wait_for_charge() { [ "$fault" != charge ]; }
ar_verify_head() { [ "$fault" != queue ]; };combat_fleet_location() { echo 2-6; }
js_collect() { if [ "$1" = 40 ];then echo "$fixture_before";[ "$fault" != before_rpc ];else echo "$fixture_after";[ "$fault" != after_rpc ];fi; }
query() {
 case "$2 $3" in
 'structs block-height') echo '{"blockHeight":"40"}' ;;
 'structs struct-type') echo "$fixture_catalog" | jq -c --arg id "$4" '[.[] | select(.StructType.id == $id)][0]';[ "$fault" != catalog_rpc ] ;;
 *) echo "$fixture_tx";[ "$fault" != tx_rpc ] ;;
 esac
}
run_tx() { [ "$5 $6 $7" = '5-1 5-2 primaryWeapon' ] || return 1;LAST_TX_OUTPUT="$fixture_tx";echo ATTEMPT;[ "$fault" != attack ]; }
_tx_json() { echo "$1"; };assert_eq() { [ "$2" = "$3" ]; }
'''
        for ah, th, count in ((3, 6, 0), (1, 1, 1), (3, 3, 1)):
            start = initial(ah, th, count);tx = receipt(start);o = attack({'before': start, 'tx': tx}, HASH)
            for fault in ('none', 'clear', 'park', 'charge', 'queue', 'before_rpc', 'after_rpc', 'catalog_rpc', 'tx_rpc', 'attack', 'fleet_fallback', 'missing_satellite', 'bad_health', 'bad_network'):
                pre = copy.deepcopy(start);end = ending(start, o)
                if fault == 'fleet_fallback': pre['target']['Struct'].update(locationType='fleet', locationId='9-6')
                if fault == 'missing_satellite': pre['satellite'] = {}
                if fault == 'bad_health': end['target']['structAttributes']['health'] = '9'
                if fault == 'bad_network': end['planet']['planetAttributes'][NETWORK+'SuccessRateNumerator'] = '0'
                r = subprocess.run(['bash', '-c', setup+function('js_execute')+'\njs_execute', 'test', str(ROOT), fault, json.dumps(pre), json.dumps(end), json.dumps(tx), json.dumps(types())], text=True, capture_output=True, timeout=10)
                with self.subTest(ah=ah, th=th, count=count, fault=fault):
                    self.assertEqual(r.returncode == 0, fault == 'none', r.stdout+r.stderr)
                    if fault in ('before_rpc', 'fleet_fallback', 'missing_satellite', 'catalog_rpc'): self.assertNotIn('ATTEMPT', r.stdout)

    def test_three_attacks_restore_roles_each_time_and_fail_instead_of_skip(self):
        setup = '''set -euo pipefail
fault="$1" fail_at="$2" PLAYER_3_FLEET_ID=9-3 PLAYER_6_PLANET_ID=2-6 calls=0 commands=0 restores=0
ar_require_command() { commands=$((commands+1));[ "$fault" != command ]; }
combat_require_struct() {
 [ "$fault" != restore ] || return 1
 case "$1 $3 $4 $5 $6" in
 'JS_SAT_ID 17 planet 2-6 space') JS_SAT_ID=5-3 ;;
 'JS_ATTACKER_ID 2 fleet 9-3 space') JS_ATTACKER_ID=5-1;restores=$((restores+1)) ;;
 'JS_TARGET_ID 14 planet 2-6 land') JS_TARGET_ID=5-2;restores=$((restores+1)) ;;
 *) return 1 ;;
 esac
}
js_execute() {
 calls=$((calls+1));[ "$restores" -eq $((calls*2)) ] || return 1
 [ "$JS_ATTACKER_ID $JS_TARGET_ID $JS_SAT_ID" = '5-1 5-2 5-3' ] || return 1
 JS_ATTACKER_ID=dead JS_TARGET_ID=dead
 echo ATTACK;[ "$fault" != attack ] || [ "$calls" -ne "$fail_at" ]
}
'''
        for fail_at in (1, 2, 3):
            for fault in ('none', 'command', 'restore', 'attack'):
                r = subprocess.run(['bash', '-c', setup+function('js_run')+'\njs_run\n[ "$commands" -eq 3 ] && [ "$JS_SHOTS" -eq 3 ]', 'test', fault, str(fail_at)], text=True, capture_output=True, timeout=5)
                with self.subTest(fault=fault, fail_at=fail_at):
                    self.assertEqual(r.returncode == 0, fault == 'none', r.stdout+r.stderr)
                    self.assertEqual(r.stdout.count('ATTACK'), 3 if fault == 'none' else fail_at if fault == 'attack' else 0)
        phase = (ROOT/'test_chain.sh').read_text().split('section "PHASE JS1:', 1)[1].split('fi # phase JS1', 1)[0]
        self.assertIn('\njs_run\n', phase);self.assertNotIn('SKIP', phase)


if __name__ == '__main__': unittest.main()
