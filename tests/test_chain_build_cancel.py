"""Exercise the selected cancellation phase's completion and state assertions."""
from pathlib import Path
import re
import subprocess
import unittest

SOURCE = (Path(__file__).parent / "test_chain.sh").read_text()
HELPERS = "\n".join(re.search(r"(?ms)^" + name + r"\(\) \{.*?^\}", SOURCE).group()
                    for name in ("assert_eq", "assert_gt", "assert_not_empty", "jqr", "combat_state_at_height", "struct_state_after_last_tx"))
MARKER = 'assert_not_empty "Build-cancel fixture must be created"'
BLOCK = MARKER + SOURCE.split(MARKER, 1)[1].split("# struct-type-all query coverage", 1)[0]


class BuildCancelTests(unittest.TestCase):
    def test_missing_fixture_and_incorrect_cancellation_cannot_pass(self):
        for fault in ("none", "omitted_false", "missing", "already_built", "no_reservation", "excess_reservation", "not_destroyed", "load_leak",
                "query_error", "no_attributes", "not_materialized", "wrong_struct", "wrong_owner", "wrong_type", "null_flag", "string_flag"):
            with self.subTest(fault=fault):
                script = r'''
set -euo pipefail
GREEN='' RED='' NC='' PASS_COUNT=0 FAIL_COUNT=0
PLAYER_2_ID=1-2 CANCEL_STRUCT_ID=5-7 P2_LOAD_BEFORE_CANCEL=100
CANCELLED=0 CALLS=0 CANCEL_BUILD_DRAW=750 fault="$1"
[ "$fault" != missing ] || CANCEL_STRUCT_ID=''
info() { :; }
LAST_TX_OUTPUT='{"txhash":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}'
_tx_json() { echo "$1"; }
query() {
    if [ "$*" = 'query tx AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA' ]; then
        printf '%s' '{"txhash":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA","height":"42","code":0}'
    elif [ "$*" = 'query structs struct 5-7' ] || [ "$*" = 'query structs struct 5-7 --height 42' ]; then
        if [ "$CANCELLED" -eq 1 ]; then [ "$*" = 'query structs struct 5-7 --height 42' ]; fi
        built=false destroyed=false
        [ "$fault" != already_built ] || built=true
        if [ "$CANCELLED" -eq 1 ] && [ "$fault" != not_destroyed ]; then destroyed=true; fi
        value=$(printf '{"Struct":{"id":"5-7","owner":"1-2","type":"18"},"structAttributes":{"isMaterialized":true,"isBuilt":%s,"isDestroyed":%s}}' "$built" "$destroyed")
        if [ "$CANCELLED" -eq 0 ]; then
            case "$fault" in
                query_error) return 1 ;;
                omitted_false) value=$(echo "$value" | jq 'del(.structAttributes.isBuilt)') ;;
                no_attributes) value=$(echo "$value" | jq 'del(.structAttributes)') ;;
                not_materialized) value=$(echo "$value" | jq '.structAttributes.isMaterialized=false') ;;
                wrong_struct) value=$(echo "$value" | jq '.Struct.id="5-8"') ;;
                wrong_owner) value=$(echo "$value" | jq '.Struct.owner="1-3"') ;;
                wrong_type) value=$(echo "$value" | jq '.Struct.type="15"') ;;
                null_flag) value=$(echo "$value" | jq '.structAttributes.isBuilt=null') ;;
                string_flag) value=$(echo "$value" | jq '.structAttributes.isBuilt="false"') ;;
            esac
        fi
        echo "$value"
    else
        [ "$*" = 'query structs player 1-2' ]
        load=850
        [ "$fault" != no_reservation ] || load=100
        [ "$fault" != excess_reservation ] || load=851
        if [ "$CANCELLED" -eq 1 ]; then
            load=100
            [ "$fault" != load_leak ] || load=101
        fi
        printf '{"gridAttributes":{"structsLoad":"%s"}}' "$load"
    fi
}
run_tx() {
    shift
    [ "$*" = 'tx structs struct-build-cancel 5-7 --from player_2' ]
    CANCELLED=1 CALLS=$((CALLS + 1))
}
'''
                script += HELPERS + "\n" + BLOCK
                script += '\n[ "$FAIL_COUNT" -eq "$2" ]\n[ "$CALLS" -eq "$3" ]\n'
                result = subprocess.run(
                    ["bash", "-c", script, "test", fault, "0" if fault in ("none", "omitted_false") else "1", "0" if fault == "missing" else "1"],
                    capture_output=True, text=True,
                )
                self.assertEqual(result.returncode, 0, result.stdout + result.stderr)


if __name__ == "__main__":
    unittest.main()
