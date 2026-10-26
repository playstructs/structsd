"""Phase 13b must exercise real state transitions even on direct resume."""
from pathlib import Path
import subprocess
import unittest

SOURCE = (Path(__file__).parent / 'test_chain.sh').read_text()
PHASE = SOURCE.split('section "PHASE 13b: Defense Clear & Stealth Systems"', 1)[1].split('fi # phase 13b', 1)[0]

HARNESS = r'''set -euo pipefail
fault="$1" HIDDEN="$2" DEFENDING=false STAGE=initial SETS=0
PLAYER_3_ID=1-3 PLAYER_3_FLEET_ID=9-3 PLAYER_3_PLANET_ID=2-3
CHARGE_DEFEND=1 CHARGE_ACTIVATE=1
# No prior command, SAM or Bomber IDs: direct resume must restore every role.
ar_require_command() { COMMAND_SHIP_ID=5-3; echo command-restored; }
combat_move_fleet() { [ "$*" = '3 2-3' ]; }
combat_require_struct() {
 case "$1" in
 SAM_STRUCT_ID) SAM_STRUCT_ID=5-10;;
 STEALTH_BOMBER_ID) [ "$fault" != missing_bomber ] || return 1; STEALTH_BOMBER_ID=5-6;;
 *) return 9;;
 esac
 echo "restored:$1"
}
wait_for_charge() { :; }
assert_eq() { [ "$2" = "$3" ]; }
run_tx() {
 local description="$1"; shift
 case "$3" in
 struct-defense-set)
  SETS=$((SETS+1)); STAGE="set$SETS"
  [ "$fault" = "$STAGE" ] || DEFENDING=true;;
 struct-defense-clear) STAGE=clear; [ "$fault" = clear ] || DEFENDING=false;;
 struct-stealth-activate) STAGE=hide; [ "$fault" = hide ] || HIDDEN=true;;
 struct-stealth-deactivate)
  STAGE=show
  if [[ "$description" = *resumed* ]];then STAGE=reveal;fi
  [ "$fault" = "$STAGE" ] || HIDDEN=false;;
 *) return 9;;
 esac
 echo "tx:$STAGE"
}
query() {
 if [ "$*" = 'query structs block-height' ];then echo '{"blockHeight":"123"}';return;fi
 local id="$4" defenders='[]' protected=0 hidden="$HIDDEN" rowid="$4" owner=1-3 status=7
 if [ "$DEFENDING" = true ];then defenders='["5-10"]';protected=3;fi
 [ "$fault:$STAGE" != duplicate:set1 ] || defenders='["5-10","5-10"]'
 [ "$fault:$STAGE" != reverse:set1 ] || protected=0
 [ "$fault:$STAGE" != stale_reverse:clear ] || protected=3
 [ "$fault:$STAGE" != stale_forward:clear ] || defenders='["5-10"]'
 [ "$fault:$STAGE" != wrong_owner:clear ] || owner=1-9
 [ "$fault:$STAGE" != wrong_id:show ] || rowid=5-999
 if [ "$fault:$STAGE" = empty:clear ];then echo '{}';return;fi
 if [ "$3" = struct-attribute ];then
  [ "$*" = 'query structs struct-attribute 5-6 status --height 123' ] || return 9
  [ "$fault" != attributes ] || status=23
  printf '{"attribute":"%s"}' "$status"
  [ "$fault" != attribute_query ];return
 fi
 case "$id" in
 5-3) printf '{"Struct":{"id":"%s","owner":"%s"},"structAttributes":{"isBuilt":true},"structDefenders":%s}' "$rowid" "$owner" "$defenders";;
 5-10) printf '{"Struct":{"id":"%s","owner":"%s"},"structAttributes":{"isBuilt":true,"protectedStructIndex":"%s"}}' "$rowid" "$owner" "$protected";;
 5-6) if [ "$HIDDEN" = true ];then status=23;fi
      printf '{"Struct":{"id":"%s","owner":"%s","type":"6"},"structAttributes":{"status":"%s","isBuilt":true,"isOnline":true,"isHidden":%s}}' "$rowid" "$owner" "$status" "$hidden";;
 *) return 9;;
 esac
 # Even valid-looking JSON from a failed RPC must not authorize a pass.
 [ "$fault:$STAGE" != query:clear ]
}
'''


class DefenseStealthTests(unittest.TestCase):
    def run_phase(self, fault='none', hidden='false'):
        return subprocess.run(['bash', '-c', HARNESS + PHASE, 'test', fault, hidden],
                              capture_output=True, text=True)

    def test_direct_resume_restores_roles_and_exercises_every_transition(self):
        for hidden in ('true', 'false'):
            with self.subTest(hidden=hidden):
                result = self.run_phase(hidden=hidden)
                self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                for stage in ('command-restored', 'restored:SAM_STRUCT_ID', 'restored:STEALTH_BOMBER_ID',
                              'tx:set1', 'tx:clear', 'tx:set2', 'tx:hide', 'tx:show'):
                    self.assertIn(stage, result.stdout)
                self.assertEqual('tx:reveal' in result.stdout, hidden == 'true')
                self.assertNotIn('SKIP', result.stdout)

    def test_missing_roles_ignored_transactions_and_false_query_evidence_fail(self):
        for fault in ('missing_bomber', 'set1', 'clear', 'set2', 'hide', 'show', 'reveal',
                      'duplicate', 'reverse', 'stale_reverse', 'stale_forward', 'wrong_owner',
                      'wrong_id', 'empty', 'query', 'attributes', 'attribute_query'):
            with self.subTest(fault=fault):
                result = self.run_phase(fault, hidden='true')
                self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)


if __name__ == '__main__':
    unittest.main()
