"""GP1 must establish delivery and the intended refusals, even for a no-op."""
from pathlib import Path
import re
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parent
SOURCE = (ROOT / 'test_chain.sh').read_text()
PHASE = SOURCE.split('section "PHASE GP1: Guild Update Primary Reactor"', 1)[1].split('fi # phase GP1', 1)[0]
NAMES = ('_tx_json', '_tx_error_line', '_tx_delivered', '_tx_outcome',
         '_tx_outcome_known', '_check_tx_output', 'run_tx', 'run_tx_expect_fail',
         'run_tx_expect_reject_matching', 'run_tx_expect_permission_denied',
         'assert_eq', 'assert_not_empty', 'get_permission_value_for_player')
HELPERS = '\n'.join(re.search(rf'(?ms)^{name}\(\) \{{.*?^\}}', SOURCE).group() for name in NAMES)


class PrimaryReactorPhaseTests(unittest.TestCase):
    def run_phase(self, fault):
        script = r'''set -euo pipefail
fault="$1" work="$2"
RED='' GREEN='' YELLOW='' BOLD='' NC='' PARAMS_TX=''
PASS_COUNT=0 FAIL_COUNT=0 SLEEP=0 LOG_BATTLE=false TX_CONFIRM_TIMEOUT=1
GUILD_ID=0-1 REACTOR_ID=3-1 PLAYER_2_ID=1-2 ALICE_ADDRESS=alice
info() { :; }
sleep() { :; }
structsd() {
    echo "$*" >> "$work/commands"
    if [[ "$*" == *999999-alice* ]]; then
        echo unknown > "$work/stage"
        local reason='reactor (999999-alice) not found'
        case "$fault" in
            unknown_gas) reason='out of gas';;
            unknown_wrong_object) reason='guild (0-1) not found';;
            unknown_panic) reason='panic: reactor (999999-alice) not found';;
        esac
        jq -nc --arg reason "$reason" '{code:5,raw_log:$reason}'
    elif [[ "$*" == *'--from player_2'* ]]; then
        echo member > "$work/stage"
        local reason='calling player (1-2) has no admin permission on object (0-1)'
        case "$fault" in member_gas) reason='out of gas';; member_missing) reason='reactor (3-1) not found';; esac
        jq -nc --arg reason "$reason" '{code:5,raw_log:$reason}'
    else
        echo owner > "$work/stage"
        case "$fault" in
            owner_missing_hash) echo '{"code":0}';;
            owner_checktx) echo '{"code":5,"raw_log":"unauthorized"}';;
            *) echo '{"code":0,"txhash":"ABC"}';;
        esac
    fi
}
query() {
    echo "$*" >> "$work/queries"
    local stage; stage=$(cat "$work/stage")
    case "$2" in
        tx)
            case "$fault" in
                owner_delivery) echo '{"txhash":"ABC","height":"10","code":5,"raw_log":"out of gas"}';;
                owner_timeout) return 1;;
                owner_wrong_hash) echo '{"txhash":"OTHER","height":"10","code":0}';;
                owner_uncommitted) echo '{"txhash":"ABC","height":"0","code":0}';;
                *) echo '{"txhash":"ABC","height":"10","code":0}';;
            esac;;
        structs)
            if [ "$3" = permission ]; then
                echo '{"permissionRecord":{"permissionId":"0-1@1-2","value":"0"}}'
            else
                local id=0-1 reactor=3-1
                case "$fault:$stage" in
                    before_query:initial|after_query:owner) return 1;;
                    before_wrong_id:initial|after_wrong_id:owner) id=0-2;;
                    unknown_mutation:unknown|member_mutation:member|after_mutation:owner) reactor=3-9;;
                esac
                jq -nc --arg id "$id" --arg reactor "$reactor" '{Guild:{id:$id,primaryReactorId:$reactor}}'
            fi;;
        *) return 9;;
    esac
}
'''
        with tempfile.TemporaryDirectory(prefix='structs-gp1-test-') as work:
            Path(work, 'stage').write_text('initial')
            result = subprocess.run(['bash', '-c', script + HELPERS + '\n' + PHASE + '\n[ "$FAIL_COUNT" -eq 0 ]\n',
                                     'test', fault, work], capture_output=True, text=True)
            commands = Path(work, 'commands')
            return result, commands.read_text() if commands.exists() else ''

    def test_committed_owner_noop_passes(self):
        result, commands = self.run_phase('none')
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertEqual(len(commands.splitlines()), 3)

    def test_broadcast_acceptance_cannot_stand_in_for_delivery(self):
        for fault in ('owner_delivery', 'owner_timeout', 'owner_wrong_hash',
                      'owner_uncommitted', 'owner_missing_hash', 'owner_checktx'):
            with self.subTest(fault=fault):
                result, _ = self.run_phase(fault)
                self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_refusals_require_the_intended_reason(self):
        for fault in ('unknown_gas', 'unknown_wrong_object', 'unknown_panic', 'member_gas', 'member_missing'):
            with self.subTest(fault=fault):
                result, _ = self.run_phase(fault)
                self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_queries_bind_identity_and_rejections_preserve_state(self):
        for fault in ('before_query', 'after_query', 'before_wrong_id', 'after_wrong_id',
                      'unknown_mutation', 'member_mutation', 'after_mutation'):
            with self.subTest(fault=fault):
                result, commands = self.run_phase(fault)
                self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
                if fault.startswith('before_'):
                    self.assertEqual(commands, '')


if __name__ == '__main__':
    unittest.main()
