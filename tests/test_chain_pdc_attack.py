"""Every PDC case needs the intended volley and matching stored planet state."""
import copy
import json
from pathlib import Path
import re
import subprocess
import unittest

from pdc_attack import PROFILES, after, attack, before, catalog
from test_chain_evasion import structure
from test_chain_dead_bomber import receipt as duel_receipt, HASH

ROOT = Path(__file__).parent.resolve()
SHELL = (ROOT/'pdc_attack.sh').read_text()


def function(name): return re.search(r'(?ms)^'+name+r'\(\) \{.*?^\}', SHELL).group()


def initial(mode, ah=3, th=None, ch=6):
    ak, tk, health = PROFILES[mode]
    if th is None: th = health[0]
    d = {'attacker': structure('5-1', ak, 'space' if ak == '2' else 'land', ah, '1-3', '9-3'),
         'target': structure('5-2', tk, 'land', th, '1-6', '9-6' if tk == '8' else '2-6'),
         'planet': {'Planet': {'id': '2-6', 'owner': '1-6'},
                    'planetAttributes': {'defensiveCannonQuantity': '0' if mode == 'E10' else '1', 'planetaryShield': '25' if mode == 'E10' else '38'}}}
    for role in ('attacker', 'target'): d[role]['Struct']['locationType'] = 'fleet' if role == 'attacker' or tk == '8' else 'planet'
    if mode != 'E10':
        d['cannon'] = copy.deepcopy(d['target']) if tk == '19' else structure('5-3', '19', 'land', ch, '1-6', '2-6')
        d['cannon']['Struct']['locationType'] = 'planet'
    return d


def types(mode):
    out = {}
    for role, unit in initial(mode).items():
        if role == 'planet': continue
        k = unit['Struct']['type']
        row = {'id': k, 'category': unit['Struct']['locationType'], 'maxHealth': '6' if k in ('14', '19') else '3'}
        if role == 'attacker':
            row.update(attackCounterable=k != '8', primaryWeaponControl='unguided', primaryWeaponTargets='1', primaryWeaponShots='1',
                primaryWeaponDamage='2', primaryWeaponBlockable=True, primaryWeaponCounterable=True, primaryWeaponCharge='3' if k == '9' else '5',
                primaryWeaponAmbits='4' if k == '9' else '6', primaryWeaponShotSuccessRateNumerator='1', primaryWeaponShotSuccessRateDenominator='1',
                primaryWeaponArmourPiercing=k == '2')
        if k in ('14', '19'): row.update(buildLimit='1', planetaryShieldContribution='13' if k == '19' else '0')
        out[role] = {'StructType': row}
    return out


def receipt(mode, start):
    ah = int(start['attacker']['structAttributes']['health']); th = int(start['target']['structAttributes']['health'])
    damage = int(mode == 'E1')
    tx = duel_receipt('armour', start, ah-damage, max(0, th-2), 2, 0, False)
    row = json.loads(tx['events'][0]['attributes'][0]['value']); shot = row['eventAttackShotDetail'][0]
    row.update(planetaryDefenseCannonDamageToAttacker=bool(damage), planetaryDefenseCannonDamage=str(damage),
               planetaryDefenseCannonDamageDestroyedAttacker=ah == damage)
    shot['targetStructLocationType'] = start['target']['Struct']['locationType']
    tx['events'][0]['attributes'][0]['value'] = json.dumps(row)
    return tx


def ending(start, o):
    end = copy.deepcopy(start)
    for role, hp in o['health'].items(): end[role]['structAttributes'].update(health=hp, isDestroyed=hp == '0', isOnline=hp != '0')
    end['planet']['planetAttributes'].update(defensiveCannonQuantity=o['cannonCount'], planetaryShield=o['shield'])
    return end


class PdcAttackTests(unittest.TestCase):
    def test_all_cases_health_boundaries_and_killing_blow(self):
        for mode, (_, tk, health) in PROFILES.items():
            self.assertTrue(catalog(types(mode), mode)['verified'])
            for ah in (1, 2, 3):
                for th in health:
                    for ch in (2, 4, 6):
                        start = initial(mode, ah, th, ch)
                        with self.subTest(mode=mode, ah=ah, th=th, ch=ch):
                            o = attack({'before': start, 'tx': receipt(mode, start)}, mode, HASH)
                            self.assertEqual(o['health']['attacker'], str(ah-int(mode == 'E1')))
                            self.assertEqual(o['health']['target'], str(max(0, th-2)))
                            self.assertEqual(o['cannonCount'], '0' if mode == 'E10' or tk == '19' and th <= 2 else '1')
                            self.assertTrue(after({'before': start, 'after': ending(start, o), 'outcome': o})['verified'])

    def test_catalog_and_before_refuse_wrong_fixture_and_unisolated_damage(self):
        for mode, role, key, value in [('E1', 'attacker', 'primaryWeaponArmourPiercing', False), ('E1', 'target', 'counterAttack', '1'),
                ('E3', 'attacker', 'attackCounterable', True), ('E4', 'target', 'unguidedDefensiveSuccessRateNumerator', '2'),
                ('kill', 'cannon', 'planetaryShieldContribution', '12'), ('E10', 'target', 'buildLimit', '2'),
                ('soften', 'attacker', 'primaryWeaponControl', 'guided')]:
            d = types(mode); d[role]['StructType'][key] = value
            with self.subTest(mode=mode, key=key), self.assertRaises(ValueError): catalog(d, mode)
        for fault in ('target_health', 'cannon_health', 'count', 'shield', 'owner', 'location', 'defenders', 'hidden', 'offline', 'duplicate', 'malformed'):
            d = initial('E1')
            if fault == 'target_health': d['target']['structAttributes']['health'] = '4'
            if fault == 'cannon_health': d['cannon']['structAttributes']['health'] = '7'
            if fault == 'count': d['planet']['planetAttributes']['defensiveCannonQuantity'] = '2'
            if fault == 'shield': d['planet']['planetAttributes']['planetaryShield'] = '12'
            if fault == 'owner': d['planet']['Planet']['owner'] = '1-3'
            if fault == 'location': d['target']['Struct']['locationType'] = 'fleet'
            if fault == 'defenders': d['target']['structDefenders'] = ['5-4']
            if fault == 'hidden': d['target']['structAttributes']['isHidden'] = True
            if fault == 'offline': d['cannon']['structAttributes']['isOnline'] = False
            if fault == 'duplicate': d['cannon']['Struct']['id'] = '5-2'
            if fault == 'malformed': d['planet']['planetAttributes']['defensiveCannonQuantity'] = 1
            with self.subTest(fault=fault), self.assertRaises(ValueError): before(d, 'E1', '5-1', '5-2', '5-3', '1-3', '1-6', '9-3', '9-6', '2-6')
        d = initial('kill'); d['cannon']['structAttributes']['health'] = '4'
        with self.assertRaises(ValueError): attack({'before': d, 'tx': receipt('kill', d)}, 'kill', HASH)

    def test_false_receipts_failed_deliveries_and_canonical_mismatches_fail(self):
        for mode in PROFILES:
            start = initial(mode); tx = receipt(mode, start)
            for level, key, value in [('row', 'attackerHealthAfter', '9'), ('row', 'planetaryDefenseCannonDamage', '2'),
                    ('row', 'planetaryDefenseCannonDamageDestroyedAttacker', True), ('shot', 'targetHealthAfter', '6'),
                    ('shot', 'targetDestroyed', mode != 'kill' and mode != 'E10'), ('shot', 'damage', '0'),
                    ('shot', 'targetCountered', True), ('shot', 'blocked', True), ('shot', 'evaded', True),
                    ('shot', 'eventAttackDefenderCounterDetail', [{}])]:
                bad = copy.deepcopy(tx); row = json.loads(bad['events'][0]['attributes'][0]['value'])
                target = row if level == 'row' else row['eventAttackShotDetail'][0]
                if target[key] == value: continue
                target[key] = value; bad['events'][0]['attributes'][0]['value'] = json.dumps(row)
                with self.subTest(mode=mode, key=key), self.assertRaises(ValueError): attack({'before': start, 'tx': bad}, mode, HASH)
            for bad in [dict(tx, code=1), dict(tx, height='0'), dict(tx, txhash='B'*64), dict(tx, events=[]), dict(tx, events=tx['events']*2)]:
                with self.assertRaises(ValueError): attack({'before': start, 'tx': bad}, mode, HASH)
            o = attack({'before': start, 'tx': tx}, mode, HASH)
            for fault in ('health', 'count', 'shield', 'owner', 'omitted_role', 'online_rubble'):
                end = ending(start, o); outcome = copy.deepcopy(o)
                if fault == 'health': end['attacker']['structAttributes']['health'] = '9'
                if fault == 'count': end['planet']['planetAttributes']['defensiveCannonQuantity'] = '7'
                if fault == 'shield': end['planet']['planetAttributes']['planetaryShield'] = '99'
                if fault == 'owner': end['target']['Struct']['owner'] = '1-3'
                if fault == 'omitted_role': outcome['health'].pop('target')
                if fault == 'online_rubble':
                    if o['health']['target'] != '0': continue
                    end['target']['structAttributes']['isOnline'] = True
                with self.subTest(mode=mode, fault=fault), self.assertRaises(ValueError): after({'before': start, 'after': end, 'outcome': outcome})
        # Proto JSON omits zero-valued attributes. Missing counters mean zero;
        # nulls, booleans, numbers, padded decimals and overflow are not zero.
        d = initial('E10'); d['planet']['planetAttributes'].pop('defensiveCannonQuantity')
        self.assertTrue(attack({'before': d, 'tx': receipt('E10', d)}, 'E10', HASH)['verified'])
        for value in (None, False, 0, '00', '-1', str(2**64), '١'):
            d['planet']['planetAttributes']['defensiveCannonQuantity'] = value
            with self.assertRaises(ValueError): attack({'before': d, 'tx': receipt('E10', d)}, 'E10', HASH)

    def test_shell_execution_refuses_missing_fixture_rpc_and_false_outcome(self):
        setup = '''set -euo pipefail
GENERATOR_TEST_DIR="$1" mode="$2" fault="$3" fixture_before="$4" fixture_after="$5" fixture_tx="$6" fixture_catalog="$7"
PLAYER_3_ID=1-3 PLAYER_6_ID=1-6 PLAYER_3_FLEET_ID=9-3 PLAYER_6_FLEET_ID=9-6 PLAYER_6_PLANET_ID=2-6
EB_PDC_ID=$(echo "$fixture_before" | jq -r '.cannon.Struct.id // "5-3"')
combat_clear_target_defenders() { [ "$fault" != clear ]; }
ar_park_as_head() { [ "$fault" != park ]; }
wait_for_charge() { [ "$fault" != charge ]; }
ar_verify_head() { [ "$fault" != queue ]; }
combat_fleet_location() { echo 2-6; }
pd_collect() { if [ "$4" = 40 ];then echo "$fixture_before";[ "$fault" != before_rpc ];else echo "$fixture_after";[ "$fault" != after_rpc ];fi; }
query() {
 case "$2 $3" in
 'structs block-height') echo '{"blockHeight":"40"}' ;;
 'structs struct-type') echo "$fixture_catalog" | jq -c --arg id "$4" '[.[] | select(.StructType.id == $id)][0]';[ "$fault" != catalog_rpc ] ;;
 *) echo "$fixture_tx";[ "$fault" != tx_rpc ] ;;
 esac
}
run_tx() { [ "$7" = primaryWeapon ] || return 1;LAST_TX_OUTPUT="$fixture_tx";echo ATTEMPT;[ "$fault" != attack ]; }
_tx_json() { echo "$1"; };assert_eq() { [ "$2" = "$3" ]; }
'''
        for mode in ('E1', 'E3', 'E4', 'soften', 'kill', 'E10'):
            start = initial(mode); tx = receipt(mode, start); o = attack({'before': start, 'tx': tx}, mode, HASH)
            for fault in ('none', 'clear', 'park', 'charge', 'queue', 'before_rpc', 'after_rpc', 'catalog_rpc', 'tx_rpc', 'attack', 'bad_health', 'bad_shield', 'missing_cannon', 'wrong_start'):
                pre = copy.deepcopy(start); end = ending(start, o)
                if fault == 'bad_health': end['target']['structAttributes']['health'] = '9'
                if fault == 'bad_shield': end['planet']['planetAttributes']['planetaryShield'] = '99'
                if fault == 'missing_cannon':
                    if mode == 'E10': continue
                    pre['cannon'] = {}
                if fault == 'wrong_start': pre['target']['structAttributes']['health'] = '0'
                r = subprocess.run(['bash', '-c', setup+function('pd_execute')+'\npd_execute "$mode" 5-1 5-2', 'test', str(ROOT), mode, fault,
                    json.dumps(pre), json.dumps(end), json.dumps(tx), json.dumps(types(mode))], text=True, capture_output=True, timeout=10)
                with self.subTest(mode=mode, fault=fault):
                    self.assertEqual(r.returncode == 0, fault == 'none', r.stdout+r.stderr)
                    if fault in ('before_rpc', 'missing_cannon', 'wrong_start', 'catalog_rpc'): self.assertNotIn('ATTEMPT', r.stdout)

    def test_sequence_cannot_skip_immunity_softening_kill_or_aftermath(self):
        setup = '''set -euo pipefail
fault="$1" fixture_hp="$2" PLAYER_3_FLEET_ID=9-3 PLAYER_6_FLEET_ID=9-6
ar_require_command() { [ "$fault" != command ]; }
combat_require_struct() { [ "$fault" != restore ] || return 1;printf -v "$1" '%s' "$3"; }
pd_require_planet() { [ "$fault" != planet ] || return 1;printf -v "$1" '%s' "$2"; }
pd_execute() {
 echo "$1 $2 $3";[ "$fault" != "$1" ] || return 1
 if [ "$1" = soften ];then fixture_hp=$((fixture_hp-2));fi
 PD_LAST_OUTCOME="{\\"health\\":{\\"cannon\\":\\"$fixture_hp\\"}}"
}
'''
        for hp in (2, 4, 6):
            for fault in ('none', 'command', 'restore', 'planet', 'E1', 'E3', 'E4', 'soften', 'kill', 'E10'):
                r = subprocess.run(['bash', '-c', setup+function('pd_run')+'\npd_run', 'test', fault, str(hp)], text=True, capture_output=True, timeout=5)
                ok = fault == 'none' or fault == 'soften' and hp == 2
                with self.subTest(hp=hp, fault=fault):
                    self.assertEqual(r.returncode == 0, ok, r.stdout+r.stderr)
                    if ok: self.assertEqual([l.split()[0] for l in r.stdout.splitlines()], ['E1', 'E3', 'E4']+['soften']*((hp-2)//2)+['kill', 'E10'])
        phase = (ROOT/'test_chain.sh').read_text().split('# GROUP E: Planetary Defense Cannon (PDC) Tests', 1)[1].split('fi # phase EB5', 1)[0]
        self.assertIn('\npd_run\n', phase); self.assertNotIn('SKIP', phase)

    def test_wounded_planet_recovery_is_verified_bounded_and_required(self):
        setup = '''set -euo pipefail
fault="$1" fixture_kind="$2" fixture_hp="$3" builds=0 PLAYER_6_PLANET_ID=2-6 EB_P3_MOBILE_ART_ID=5-1
combat_require_struct() {
 [ "$fault" != build ] || return 1
 builds=$((builds+1));printf -v "$1" '%s' '5-2';echo BUILD
 if [ "$builds" = 2 ] && [ "$fault" != rebuild ];then fixture_hp=6;fi
}
query() { printf '{"structAttributes":{"health":"%s"}}' "$fixture_hp";[ "$fault" != query ]; }
pd_execute() {
 echo "$1 $2 $3";[ "$fault" != attack ] || return 1
 if [ "$fault" != unchanged ];then fixture_hp=$((fixture_hp>2 ? fixture_hp-2 : 0));fi
 PD_LAST_OUTCOME=$(printf '{"health":{"target":"%s"}}' "$fixture_hp")
}
'''
        for kind in ('14', '19'):
            for hp in range(1, 7):
                needs_rebuild = hp != 6 if kind == '14' else hp % 2 == 1
                for fault in ('none', 'build', 'query', 'attack', 'unchanged', 'rebuild'):
                    r = subprocess.run(['bash', '-c', setup+function('pd_require_planet')+'\npd_require_planet EB_PDC_ID "$fixture_kind"',
                        'test', fault, kind, str(hp)], text=True, capture_output=True, timeout=5)
                    ok = fault == 'none' or not needs_rebuild and fault in ('attack', 'unchanged', 'rebuild')
                    with self.subTest(kind=kind, hp=hp, fault=fault):
                        self.assertEqual(r.returncode == 0, ok, r.stdout+r.stderr)
                        self.assertLessEqual(r.stdout.count('BUILD'), 2)
                        attacks = [line for line in r.stdout.splitlines() if line.startswith('prepare_')]
                        self.assertLessEqual(len(attacks), 3)
                        if ok and needs_rebuild:
                            self.assertEqual(len(attacks), (hp+1)//2)
                            self.assertTrue(all(line.startswith('prepare_ore' if kind == '14' else 'prepare_cannon') for line in attacks))


if __name__ == '__main__': unittest.main()
