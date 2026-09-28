"""Offline tests for required fixture failure paths; no chain commands run."""
from pathlib import Path
import re
import subprocess
import unittest


SOURCE = (Path(__file__).parent / "test_chain.sh").read_text()
FUNCTIONS = "\n".join(
    re.search(rf"(?ms)^{name}\(\) \{{.*?^\}}", SOURCE).group()
    for name in ("assert_eq", "assert_gt", "assert_not_empty", "jqr", "temp_substation_create")
)


class SubstationFixtureTests(unittest.TestCase):
    def test_failed_main_feeder_recovery_stops_before_mutation(self):
        script = """
set -euo pipefail
RED='' GREEN='' NC=''
FAIL_COUNT=0 PASS_COUNT=0
PLAYER_1_ID=1-1 P1_ALLOC_ID='' SUBSTATION_ID=4-1
TEMP_ALLOC_POWER=100000 TEMP_ALLOC_ID=''
MUTATIONS=0
info() { :; }
get_fixture_feeder() { echo 'ambiguous fixture feeder' >&2; return 1; }
query() { echo 'unexpected query' >&2; exit 8; }
run_tx() { MUTATIONS=$((MUTATIONS + 1)); }
""" + FUNCTIONS + """
if temp_substation_create RESULT fixture; then exit 9; fi
[ "$FAIL_COUNT" -eq 1 ]
[ "$MUTATIONS" -eq 0 ]
[ -z "$TEMP_ALLOC_ID" ]
[ -z "$RESULT" ]
"""
        subprocess.run(["bash", "-c", script], check=True, capture_output=True, text=True)

    def test_failed_create_cannot_reuse_an_older_nonprimary_allocation(self):
        script = """
set -euo pipefail
RED='' GREEN='' NC=''
FAIL_COUNT=0 PASS_COUNT=0
PLAYER_1_ID=1-1 P1_ALLOC_ID=2-1 SUBSTATION_ID=4-1
TEMP_ALLOC_POWER=100000 TEMP_ALLOC_ID=''
MAIN_POWER=300000 SUBSTATION_CREATES=0
info() { :; }
get_fixture_feeder() { echo 6-9; }
query() { printf '{"gridAttributes":{"power":"%s"}}\\n' "$MAIN_POWER"; }
get_created_grid_id() { return 1; }
run_tx() {
    if [ "$4" = allocation-update ]; then MAIN_POWER="$6"; fi
    if [ "$4" = substation-create ]; then SUBSTATION_CREATES=$((SUBSTATION_CREATES + 1)); fi
}
""" + FUNCTIONS + """
if temp_substation_create RESULT fixture; then exit 9; fi
[ "$FAIL_COUNT" -eq 1 ]
[ "$MAIN_POWER" -eq 300000 ]
[ "$SUBSTATION_CREATES" -eq 0 ]
[ -z "$TEMP_ALLOC_ID" ]
[ -z "$RESULT" ]
"""
        subprocess.run(["bash", "-c", script], check=True, capture_output=True, text=True)


if __name__ == "__main__":
    unittest.main()
