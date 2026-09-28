"""Exercise the real fleet setup and signer adapter with reused and partial fixtures."""
from pathlib import Path
import subprocess
import unittest
from fixture_grid_recovery import feeder
from test_chain_grid_recovery import allocation, page

ROOT = Path(__file__).resolve().parent
SOURCE = (ROOT / 'test_chain.sh').read_text()


class FleetResumeTests(unittest.TestCase):
    def test_unconnected_feeder_recovery_preserves_identity_and_refuses_ambiguity(self):
        self.assertEqual(feeder(lambda *a: page([allocation(1), allocation(2, '')]), '1-7', '', '1-7', '123'), '6-2')
        self.assertEqual(feeder(lambda *a: page([allocation(1)]), '1-7', '', '1-7', '123'), '')
        with self.assertRaisesRegex(ValueError, 'ambiguous'):
            feeder(lambda *a: page([allocation(1, ''), allocation(2, '')]), '1-7', '', '1-7', '123')

    def test_setup_reuses_connected_partial_and_existing_planets_and_uses_free_capacity(self):
        phase = SOURCE.split('section "PHASE 17: Fleet Movement Setup"', 1)[1].split('# ─── Verify Command Ships', 1)[0]
        script = r'''set -euo pipefail
mode="$1" ALLOCATION_TEST_DIR="$2"
BOB_ADDRESS=bob PLAYER_1_ID=1-1 GUILD_ID=0-1 REACTOR_ID=3-1 VALIDATOR_ADDRESS=validator SUBSTATION_ID=4-1 PARAMS_KEYS=''
info() { :; }
assert_not_empty() { [ -n "$2" ]; }
assert_eq() { [ "$2" = "$3" ]; }
jqr() { printf '%s' "$1" | jq -r "$2"; }
structsd() { echo '{"address":"address"}'; }
get_fixture_feeder() {
 if [ "$mode" = connected ] && [ "$2" = 4-1 ];then echo 6-1
 elif [ "$mode" = partial ] && [ -z "$2" ];then echo 6-1
 elif [ "${connected:-false}" = true ] && [ "$2" = 4-1 ];then echo 6-1;fi
}
get_created_grid_id() { echo 6-1; }
query() {
 case "$3" in
 address) echo '{"playerId":"1-31"}';;
 player) echo '{"Player":{"id":"1-31","guildId":"0-1","substationId":"4-1","planetId":"2-31","fleetId":"9-31"},"gridAttributes":{"capacity":"9600000","load":"4800000"}}';;
 allocation) local dest='';if [ "$mode" = connected ] || [ "${connected:-false}" = true ];then dest=4-1;fi
   printf '{"Allocation":{"id":"6-1","destinationId":"%s"}}' "$dest";;
 *) return 9;; esac
}
run_tx() {
 shift
 case "$3" in
 send|delegate) :;;
 allocation-create) [ "$mode" = new ] && [ "$5" = 4800000 ];echo created;;
 substation-allocation-connect) [ "$4" = 6-1 ] && [ "$5" = 4-1 ];connected=true;echo connected;;
 *) echo "Unexpected mutation $*" >&2;return 9;; esac
}
'''
        for mode in ('connected', 'partial', 'new'):
            result = subprocess.run(['bash', '-c', script + phase, 'test', mode, str(ROOT)], capture_output=True, text=True)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
            self.assertEqual(result.stdout.count('\ncreated\n'), 1 if mode == 'new' else 0)
            self.assertEqual(result.stdout.count('\nconnected\n'), 0 if mode == 'connected' else 1)

    def test_fleet_adapter_preserves_main_player_and_uses_fleet_key(self):
        script = r'''set -euo pipefail
source "$1/fleet_fixture.sh"
GENERATOR_TEST_DIR="$1"
PLAYER_1_ID=1-1 PLAYER_1_FLEET_ID=9-1 PLAYER_1_PLANET_ID=2-1
FP_2_ID=1-32 FP_2_FLEET_ID=9-32 FP_2_PLANET_ID=2-32
query() { echo '{"Fleet":{"id":"9-32","owner":"1-32","commandStruct":""}}'; }
combat_require_struct() {
 [ "$PLAYER_1_ID" = 1-32 ] && [ "$PLAYER_1_FLEET_ID" = 9-32 ] && [ "$PLAYER_1_PLANET_ID" = 2-32 ]
 [ "$1" = FP_CS_2 ] && [ "$9" = fplayer_2 ]
 printf -v "$1" '%s' 5-32
}
combat_move_fleet() { [ "$PLAYER_1_ID" = 1-32 ] && [ "$2" = 2-32 ] && [ "$3" = fplayer_2 ]; }
fp_require_command 2
fp_move_home 2
[ "$FP_CS_2" = 5-32 ] && [ "$PLAYER_1_ID" = 1-1 ] && [ "$PLAYER_1_FLEET_ID" = 9-1 ] && [ "$PLAYER_1_PLANET_ID" = 2-1 ]
'''
        result = subprocess.run(['bash', '-c', script, 'test', str(ROOT)], capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
