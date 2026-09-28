"""An erroneous early expiry refund must not enter the no-refund baseline."""
from pathlib import Path
import re
import subprocess
import unittest

ROOT = Path(__file__).parent
SOURCE = (ROOT / "test_chain.sh").read_text()
HELPERS = "\n".join(re.search(r"(?ms)^" + name + r"\(\) \{.*?^\}", SOURCE).group()
                    for name in ("assert_eq", "assert_not_empty", "assert_object_missing"))
BLOCK = 'SETTLE_COLL_C=$((SETTLE_DUR_C * SETTLE_CAP_C))' + SOURCE.split(
    'SETTLE_COLL_C=$((SETTLE_DUR_C * SETTLE_CAP_C))', 1)[1].split('# ─── Guild-owned consumer agreement ───', 1)[0]


class PlayerExpiryTests(unittest.TestCase):
    def test_refund_baseline_precedes_any_possible_expiry(self):
        for vault in (10000, 100000, 2**64-1):
            for fault in ("none", "early_refund", "late_refund", "earnings", "collateral", "load", "timeout", "allocation_retained", "allocation_rpc", "missing_allocation", "wrong_allocation"):
                with self.subTest(vault=vault, fault=fault):
                    script = r'''
set -euo pipefail
GREEN='' RED='' NC='' PASS_COUNT=0 FAIL_COUNT=0 PARAMS_QUERY='' CALLS=0
GENERATOR_TEST_DIR="$1" VAULT="$2" fault="$3"
SETTLE_DUR_C=5 SETTLE_CAP_C=2000 PLAYER_2_ID=1-2 PLAYER_2_VAULT=vault
SETTLE_PROV_ID=10-1 SETTLE_COLL_ADDR=collateral SETTLE_EARN_ADDR=earnings
POOL=17 EARNINGS="$2"
info() { :; }
fund_vault_alpha() { [ "$*" = '1-2 10000' ]; }
get_balance() {
    [ "$2" = ualpha ]
    case "$1" in vault) echo "$VAULT" ;; collateral) echo "$POOL" ;; earnings) echo "$EARNINGS" ;; *) return 9 ;; esac
}
run_tx() {
    shift
    [ "$*" = 'tx structs agreement-open 10-1 5 2000 --from player_2' ]
    CALLS=$((CALLS + 1))
    VAULT=$(python3 -c 'import sys; print(int(sys.argv[1])-10000)' "$VAULT")
    EARNINGS=$(python3 -c 'import sys; print(int(sys.argv[1])+10000)' "$EARNINGS")
    if [ "$fault" = early_refund ]; then VAULT=$(python3 -c 'import sys; print(int(sys.argv[1])+1)' "$VAULT"); fi
}
get_opened_agreement_record() {
    [ "$*" = '10-1 1-2 2000' ]
    case "$fault" in
        missing_allocation) echo '{"id":"11-3","endBlock":"105"}'; return ;;
        wrong_allocation) echo '{"id":"11-3","endBlock":"105","allocationId":"10-3"}'; return ;;
    esac
    echo '{"id":"11-3","endBlock":"105","allocationId":"6-3"}'
}
wait_for_block() {
    [ "$*" = '107 120' ]
    case "$fault" in
        late_refund) VAULT=$(python3 -c 'import sys; print(int(sys.argv[1])+1)' "$VAULT") ;;
        earnings) EARNINGS=$(python3 -c 'import sys; print(int(sys.argv[1])-1)' "$EARNINGS") ;;
        collateral) POOL=16 ;;
        timeout) return 1 ;;
    esac
}
structsd() {
    case "$*" in
        'query structs agreement 11-3') ;;
        'query structs allocation 6-3')
            case "$fault" in
                allocation_retained) echo '{"Allocation":{"id":"6-3"}}'; return ;;
                allocation_rpc) echo 'connection refused' >&2; return 1 ;;
            esac ;;
        *) return 9 ;;
    esac
    echo 'Error: object not found' >&2; return 1
}
get_provider_load() { [ "$1" = 10-1 ]; if [ "$fault" = load ]; then echo 2000; else echo 0; fi; }
'''
                    script += HELPERS + '\n' + BLOCK + '\n[ "$CALLS" -eq 1 ]\n[ "$FAIL_COUNT" -eq 0 ]\n'
                    result = subprocess.run(["bash", "-c", script, "test", str(ROOT.resolve()), str(vault), fault], capture_output=True, text=True)
                    if fault == "none":
                        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                    else:
                        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)


if __name__ == "__main__":
    unittest.main()
