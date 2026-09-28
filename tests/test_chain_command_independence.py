"""Phase 14 must deliver its attacks even after target destruction or resume."""
import copy
import json
from pathlib import Path
import subprocess
import tempfile
import unittest

from evasion_attack import after, attack, before, catalog
from test_chain_evasion import HASH, ROOT, SOURCE, function, receipt, start, structure, types


def initial():
    result = start('P14_OFFLINE')
    result['command'] = structure('5-3', '1', 'space', 4, '1-6', '9-6')
    result['command']['structAttributes']['isOnline'] = False
    return result


class CommandIndependenceTests(unittest.TestCase):
    def test_all_phase14_attacks_require_exact_receipt_and_state_for_either_roll(self):
        for case in ('P14_1', 'P14_2', 'P14_OFFLINE'):
            self.assertEqual(catalog(types(case), case), {'verified': True})
            for evaded in (True, False):
                for hp in (1, 3):
                    with self.subTest(case=case, evaded=evaded, hp=hp):
                        doc = initial()
                        doc['target']['structAttributes']['health'] = str(hp)
                        result = attack({'before': doc, 'tx': receipt(case, evaded, th=hp)}, case, HASH)
                        current = copy.deepcopy(doc)
                        current['target']['structAttributes'].update(health=result['targetHealth'],
                            isDestroyed=result['targetDestroyed'], isOnline=not result['targetDestroyed'])
                        self.assertEqual(after({'before': doc, 'after': current, 'outcome': result}, case), {'verified': True})

    def test_offline_claim_rejects_unavailable_wrong_or_changed_command(self):
        for section, key, value in (
                ('Struct', 'id', '5-4'), ('Struct', 'owner', '1-3'), ('Struct', 'type', '11'),
                ('Struct', 'locationId', '9-3'), ('Struct', 'operatingAmbit', 'land'),
                ('structAttributes', 'isOnline', True), ('structAttributes', 'isOnline', 'false'),
                ('structAttributes', 'isDestroyed', True), ('structAttributes', 'isBuilt', False),
                ('structAttributes', 'health', '0')):
            doc = initial()
            doc['command'][section][key] = value
            with self.subTest(key=key, value=value), self.assertRaises(ValueError):
                before(doc, 'P14_OFFLINE', '5-1', '5-2', '1-6', '1-3', '9-6', '9-3', '5-3')
        original = initial()
        result = attack({'before': original, 'tx': receipt('P14_OFFLINE', True)}, 'P14_OFFLINE', HASH)
        for section, key, value in (('Struct', 'id', '5-4'), ('structAttributes', 'isOnline', True),
                                    ('structAttributes', 'health', '3')):
            current = copy.deepcopy(original)
            current['command'][section][key] = value
            with self.subTest(after=key), self.assertRaises(ValueError):
                after({'before': original, 'after': current, 'outcome': result}, 'P14_OFFLINE')
        del original['command']
        with self.assertRaises(KeyError):
            attack({'before': original, 'tx': receipt('P14_OFFLINE', True)}, 'P14_OFFLINE', HASH)

    def test_shell_uses_explicit_battlefield_and_requires_both_command_snapshots(self):
        for fault in ('none', 'before_query', 'after_query', 'before_online', 'after_online', 'wrong_planet'):
            with self.subTest(fault=fault), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                doc = initial()
                for role in ('attacker', 'target', 'command'):
                    for height in (10, 12):
                        row = copy.deepcopy(doc[role])
                        if role == 'command' and ((height == 10 and fault == 'before_online') or (height == 12 and fault == 'after_online')):
                            row['structAttributes']['isOnline'] = True
                        (root/f'{role}-{height}.json').write_text(json.dumps(row))
                for role in ('attacker', 'target'):
                    (root/f'{role}-type.json').write_text(json.dumps(types('P14_OFFLINE')[role]))
                (root/'tx.json').write_text(json.dumps(receipt('P14_OFFLINE', True)))
                script = r'''set -euo pipefail
GENERATOR_TEST_DIR="$1" fixture_dir="$2" fault="$3" HASH="$4"
PLAYER_6_ID=1-6 PLAYER_3_ID=1-3 PLAYER_6_FLEET_ID=9-6 PLAYER_3_FLEET_ID=9-3
# No PLAYER_6_PLANET_ID or EV counters: the ordinary lifecycle must stand alone.
combat_require_struct() { if [ "$1" = ATTACKER ];then ATTACKER=5-1;else TARGET=5-2;fi; }
combat_clear_target_defenders() { :; }
wait_for_charge() { :; }
_secondary_charge() { echo 5; }
combat_state_at_height() {
 local role=attacker
 case "$1" in 5-2) role=target;; 5-3) role=command;; esac
 cat "$fixture_dir/$role-$2.json"
 if [ "$role" = command ] && { [[ "$fault" = before_query && "$2" = 10 ]] || [[ "$fault" = after_query && "$2" = 12 ]]; };then return 7;fi
}
combat_fleet_location() { if [ "$fault" = wrong_planet ];then echo 2-6;else echo 2-77;fi; }
query() {
 case "$*" in
 'query structs block-height') echo '{"blockHeight":"10"}';;
 'query structs struct-type 11 --height 10') cat "$fixture_dir/attacker-type.json";;
 'query structs struct-type 7 --height 10') cat "$fixture_dir/target-type.json";;
 "query tx $HASH") cat "$fixture_dir/tx.json";;
 *) return 9;;
 esac
}
run_tx() { touch "$fixture_dir/sent"; LAST_TX_OUTPUT="{\"txhash\":\"$HASH\"}"; }
_tx_json() { printf '%s' "$1"; }
assert_eq() { [ "$2" = "$3" ]; }
'''
                result = subprocess.run(['bash', '-c', script + function('ev_attack') +
                    '\nev_attack P14_OFFLINE ATTACKER 6 11 water TARGET 3 7 air secondaryWeapon 2-77 5-3',
                    'test', str(ROOT), directory, fault, HASH], capture_output=True, text=True)
                self.assertEqual(result.returncode == 0, fault == 'none', result.stdout + result.stderr)
                if fault == 'none':
                    self.assertIn('P14 outcome:', result.stdout)
                elif fault in ('before_query', 'before_online', 'wrong_planet'):
                    self.assertFalse((root/'sent').exists())

    def test_phase_restores_destroyed_target_before_offline_and_checks_defense_and_stealth(self):
        phase = SOURCE.split('section "PHASE 14: Secondary Weapons & Defensive Maneuvers"', 1)[1].split('fi # phase 14', 1)[0]
        script = r'''set -euo pipefail
fault="$1"
PLAYER_3_ID=1-3 PLAYER_2_ID=1-2 PLAYER_3_FLEET_ID=9-3 PLAYER_2_FLEET_ID=9-2 PLAYER_2_PLANET_ID=2-2
CHARGE_ACTIVATE=1 CHARGE_DEFEND=1
COMMAND_SHIP_ID=5-3 ONLINE=true HIDDEN=true DEFENDING=false
info() { :; }
section() { :; }
ar_require_command() { echo "command $1"; }
combat_move_fleet() { :; }
combat_require_struct() {
 case "$1" in CRUISER_ID) CRUISER_ID=5-11;; INTERCEPTOR_ID) INTERCEPTOR_ID=5-7;; STEALTH_BOMBER_ID) STEALTH_BOMBER_ID=5-6;; esac
 echo "restore $1"
}
wait_for_charge() { :; }
ev_attack() {
 [ -n "$INTERCEPTOR_ID" ]
 echo "attack $1 $ONLINE"
 if [ "$1" = P14_OFFLINE ];then [ "$ONLINE" = false ];fi
 # The second online attack destroys the target. It must be restored again.
 if [ "$1" = P14_2 ];then INTERCEPTOR_ID='';fi
}
run_tx() {
 shift
 [ "$3" != "$fault" ] || return 0
 case "$3" in
 struct-deactivate) ONLINE=false;; struct-activate) ONLINE=true;;
 struct-stealth-activate) HIDDEN=true;; struct-stealth-deactivate) HIDDEN=false;;
 struct-defense-set) DEFENDING=true;; struct-defense-clear) DEFENDING=false;;
 *) return 9;; esac
}
query() {
 case "$4" in
 5-3) local defenders='[]';if [ "$DEFENDING" = true ];then defenders='["5-11"]';fi
      printf '{"Struct":{"id":"5-3"},"structAttributes":{"isOnline":%s},"structDefenders":%s}' "$ONLINE" "$defenders";;
 5-11) local index=0;if [ "$DEFENDING" = true ];then index=3;fi
       printf '{"Struct":{"id":"5-11"},"structAttributes":{"protectedStructIndex":"%s"}}' "$index";;
 5-6) printf '{"Struct":{"id":"5-6"},"structAttributes":{"isBuilt":true,"isOnline":true,"isHidden":%s}}' "$HIDDEN";;
 *) return 9;; esac
}
combat_fixture_state() { [ "$ONLINE" = false ];echo activate; }
assert_eq() { [ "$2" = "$3" ] || { echo "FAILED $1 expected $2 actual $3" >&2;return 1; }; }
'''
        for fault in ('none', 'struct-deactivate', 'struct-activate', 'struct-defense-set',
                      'struct-defense-clear', 'struct-stealth-activate', 'struct-stealth-deactivate'):
            with self.subTest(fault=fault):
                result = subprocess.run(['bash', '-c', script + phase + '\n[ "$ONLINE" = true ] && [ "$HIDDEN" = false ] && [ "$DEFENDING" = false ]',
                                        'test', fault], capture_output=True, text=True)
                self.assertEqual(result.returncode == 0, fault == 'none', result.stdout + result.stderr)
                if fault == 'none':
                    self.assertEqual([line for line in result.stdout.splitlines() if line.startswith('attack ')],
                                     ['attack P14_1 true', 'attack P14_2 true', 'attack P14_OFFLINE false'])
                    self.assertIn('attack P14_2 true\nrestore INTERCEPTOR_ID\nattack P14_OFFLINE false', result.stdout)
        self.assertNotIn('SKIP:', phase)
