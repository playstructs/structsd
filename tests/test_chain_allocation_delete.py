"""Verify the live deletion assertions detect load and permission leaks."""
from pathlib import Path
import re
import subprocess
import unittest

SOURCE = (Path(__file__).parent / "test_chain.sh").read_text()
HELPERS = "\n".join(re.search(r"(?ms)^" + name + r"\(\) \{.*?^\}", SOURCE).group()
                    for name in ("assert_eq", "jqr"))
BLOCK = SOURCE.split("# ─── allocation-delete ───", 1)[1].split("# ─── Query: allocation-all-by-destination ───", 1)[0]


class AllocationDeletePhaseTests(unittest.TestCase):
    def test_deletion_releases_only_its_load_and_clears_grants(self):
        for load, power in ((150, 100), (100, 100), (2**64 - 1, 2**63 + 1)):
            for fault in ("none", "still_exists", "load_leak", "over_release", "creator_grant", "recipient_grant", "missing_grid"):
                with self.subTest(load=load, power=power, fault=fault):
                    script = r'''
set -euo pipefail
GREEN='' RED='' NC='' PASS_COUNT=0 FAIL_COUNT=0 CALLS=0
P5_ALLOC_ID=6-1 PLAYER_5_ID=1-5 PLAYER_1_ID=1-1
LOAD="$1" POWER="$2" fault="$3" DELETED=0
query() {
    case "$*" in
        'query structs allocation 6-1')
            if [ "$DELETED" -eq 0 ] || [ "$fault" = still_exists ]; then
                printf '{"Allocation":{"id":"6-1"},"gridAttributes":{"power":"%s"}}' "$POWER"
            else echo '{}'; fi ;;
        'query structs player 1-5')
            if [ "$DELETED" -eq 1 ] && [ "$fault" = missing_grid ]; then echo '{}'; return; fi
            printf '{"gridAttributes":{"load":"%s"}}' "$LOAD" ;;
        *) return 9 ;;
    esac
}
get_permission_value_for_player() {
    [ "$1" = 6-1 ]
    if { [ "$fault" = creator_grant ] && [ "$2" = 1-5 ]; } ||
       { [ "$fault" = recipient_grant ] && [ "$2" = 1-1 ]; }; then echo 4; else echo 0; fi
}
run_tx() {
    shift
    [ "$*" = 'tx structs allocation-delete 6-1 --from player_5' ]
    DELETED=1 CALLS=$((CALLS + 1))
    LOAD=$(python3 -c 'import sys; a,b=map(int,sys.argv[1:3]); print(a-b+{"load_leak":1,"over_release":-1}.get(sys.argv[3],0))' "$LOAD" "$POWER" "$fault")
}
'''
                    script += HELPERS + "\n" + BLOCK
                    script += '\n[ "$CALLS" -eq 1 ]\n[ "$((PASS_COUNT + FAIL_COUNT))" -eq 4 ]\n'
                    script += '[ "$FAIL_COUNT" -eq "$4" ]\n'
                    result = subprocess.run(["bash", "-c", script, "test", str(load), str(power), fault,
                                             "0" if fault == "none" else "1"], capture_output=True, text=True)
                    if fault == "missing_grid":
                        self.assertNotEqual(result.returncode, 0)
                    else:
                        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)


if __name__ == "__main__":
    unittest.main()
