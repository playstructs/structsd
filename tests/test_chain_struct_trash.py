"""Validate the live trash phase's required coverage and exact load assertions."""
from pathlib import Path
import re
import subprocess
import unittest

SOURCE = (Path(__file__).parent / "test_chain.sh").read_text()
HELPERS = "\n".join(re.search(r"(?ms)^" + name + r"\(\) \{.*?^\}", SOURCE).group()
                    for name in ("assert_eq", "assert_not_empty", "jqr", "combat_state_at_height", "struct_state_after_last_tx"))
MARKER = 'assert_not_empty "Trash fixture must be created"'
BLOCK = MARKER + SOURCE.split(MARKER, 1)[1].split("fi # phase 7c", 1)[0]


class StructTrashTests(unittest.TestCase):
    def test_missing_fixture_and_incomplete_teardown_fail(self):
        for fault in ("none", "missing", "no_reservation", "excess_reservation", "not_destroyed", "load_leak"):
            with self.subTest(fault=fault):
                script = r'''
set -euo pipefail
GREEN='' RED='' NC='' PASS_COUNT=0 FAIL_COUNT=0
PLAYER_2_ID=1-2 TRASH_STRUCT_ID=5-7 P2_TRASH_LOAD_BASELINE=100
TRASH_BUILD_DRAW=750 TRASH_BUILD_CHARGE=13 TRASHED=0 CALLS=0 fault="$1"
[ "$fault" != missing ] || TRASH_STRUCT_ID=''
info() { :; }
LAST_TX_OUTPUT='{"txhash":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}'
_tx_json() { echo "$1"; }
wait_for_charge() { [ "$*" = '1-2 13' ]; }
query() {
    if [ "$*" = 'query tx AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA' ]; then
        printf '%s' '{"txhash":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA","height":"42","code":0}'
    elif [ "$*" = 'query structs struct 5-7' ] || [ "$*" = 'query structs struct 5-7 --height 42' ]; then
        [ "$*" = 'query structs struct 5-7 --height 42' ]
        destroyed=true
        [ "$fault" != not_destroyed ] || destroyed=false
        printf '{"Struct":{"id":"5-7"},"structAttributes":{"isDestroyed":%s}}' "$destroyed"
    else
        [ "$*" = 'query structs player 1-2' ]
        load=850
        [ "$fault" != no_reservation ] || load=100
        [ "$fault" != excess_reservation ] || load=851
        if [ "$TRASHED" -eq 1 ]; then
            load=100
            [ "$fault" != load_leak ] || load=101
        fi
        printf '{"gridAttributes":{"structsLoad":"%s"}}' "$load"
    fi
}
run_tx() {
    shift
    [ "$*" = 'tx structs struct-trash 5-7 --from player_2' ]
    TRASHED=1 CALLS=$((CALLS + 1))
}
'''
                script += HELPERS + "\n" + BLOCK
                script += '\n[ "$FAIL_COUNT" -eq "$2" ]\n[ "$CALLS" -eq "$3" ]\n'
                result = subprocess.run(
                    ["bash", "-c", script, "test", fault, "0" if fault == "none" else "1", "0" if fault == "missing" else "1"],
                    capture_output=True, text=True,
                )
                self.assertEqual(result.returncode, 0, result.stdout + result.stderr)


if __name__ == "__main__":
    unittest.main()
