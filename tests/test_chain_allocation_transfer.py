"""Run the actual transfer round trip and inject independent state faults."""
from pathlib import Path
import re
import subprocess
import unittest

SOURCE = (Path(__file__).parent / "test_chain.sh").read_text()
HELPERS = "\n".join(re.search(r"(?ms)^" + name + r"\(\) \{.*?^\}", SOURCE).group()
                    for name in ("assert_eq", "jqr"))
BLOCK = SOURCE.split("# ─── allocation-transfer: transfer controller to alice ───", 1)[1].split(
    "# ─── allocation-transfer to a player that does not exist ───", 1)[0]


class AllocationTransferPhaseTests(unittest.TestCase):
    def test_round_trip_preserves_permissions_and_allocation_state(self):
        for stage in (1, 2):
            for fault in ("none", "controller", "outgoing", "incoming", "source", "destination", "power"):
                with self.subTest(stage=stage, fault=fault):
                    script = r'''
set -euo pipefail
GREEN='' RED='' NC='' PASS_COUNT=0 FAIL_COUNT=0 CALLS=0
P5_ALLOC_ID=6-1 PLAYER_5_ID=1-5 PLAYER_1_ID=1-1
PERM_UPDATE=4 PERM_DELETE=8 PERM_ALLOCATION_CONNECTION=2048
P5_PERMS=2062 ALICE_PERMS=4 CONTROLLER=1-5 SOURCE=1-5 DESTINATION=4-1 POWER=123
stage="$1" fault="$2"
info() { :; }
get_permission_value_for_player() {
    [ "$1" = 6-1 ]
    case "$2" in 1-5) echo "$P5_PERMS" ;; 1-1) echo "$ALICE_PERMS" ;; *) return 9 ;; esac
}
query() {
    [ "$*" = 'query structs allocation 6-1' ]
    printf '{"Allocation":{"id":"6-1","controller":"%s","sourceObjectId":"%s","destinationId":"%s"},"gridAttributes":{"power":"%s"}}' "$CONTROLLER" "$SOURCE" "$DESTINATION" "$POWER"
}
run_tx() {
    shift
    CALLS=$((CALLS + 1))
    if [ "$CALLS" -eq 1 ]; then
        [ "$*" = 'tx structs allocation-transfer 6-1 1-1 --from player_5' ]
        CONTROLLER=1-1 P5_PERMS=14 ALICE_PERMS=2052
    else
        [ "$*" = 'tx structs allocation-transfer 6-1 1-5 --from alice' ]
        CONTROLLER=1-5 P5_PERMS=2062 ALICE_PERMS=4
    fi
    if [ "$CALLS" -eq "$stage" ]; then
        case "$fault" in
            controller) CONTROLLER=1-99 ;;
            outgoing)
                if [ "$stage" -eq 1 ]; then P5_PERMS=0; else ALICE_PERMS=0; fi ;;
            incoming)
                if [ "$stage" -eq 1 ]; then ALICE_PERMS=2048; else P5_PERMS=2048; fi ;;
            source) SOURCE=1-99 ;;
            destination) DESTINATION=4-99 ;;
            power) POWER=124 ;;
        esac
    fi
}
'''
                    script += HELPERS + "\n" + BLOCK
                    script += '\n[ "$CALLS" -eq 2 ]\n[ "$((PASS_COUNT + FAIL_COUNT))" -eq 9 ]\n'
                    script += '[ "$FAIL_COUNT" -eq 0 ]\n' if fault == "none" else '[ "$FAIL_COUNT" -gt 0 ]\n'
                    result = subprocess.run(["bash", "-c", script, "test", str(stage), fault],
                                            capture_output=True, text=True)
                    self.assertEqual(result.returncode, 0, result.stdout + result.stderr)


if __name__ == "__main__":
    unittest.main()
