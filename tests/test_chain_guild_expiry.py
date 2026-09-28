"""Exercise guild expiry accounting at balances beyond shell signed integers."""
from pathlib import Path
import re
import subprocess
import unittest

ROOT = Path(__file__).parent
SOURCE = (ROOT / "test_chain.sh").read_text()
HELPERS = "\n".join(re.search(r"(?ms)^" + name + r"\(\) \{.*?^\}", SOURCE).group()
                    for name in ("assert_eq", "assert_not_empty", "assert_object_missing", "get_provider_load", "jqr"))
BLOCK = SOURCE.split('# ─── Guild-owned consumer agreement ───', 1)[1].split('# ─── Clean up ───', 1)[0]


class GuildExpiryTests(unittest.TestCase):
    def test_complete_guild_expiry_checks(self):
        for balance in (40000, 2**63+1, 2**64-1):
            for fault in ("none", "early_refund", "late_refund", "owner_debit", "owner_expiry", "earnings", "collateral", "load", "rpc", "timeout", "wrong_owner", "allocation_retained", "allocation_rpc", "missing_allocation", "wrong_allocation"):
                with self.subTest(balance=balance, fault=fault):
                    script = r'''
set -euo pipefail
GREEN='' RED='' NC='' PASS_COUNT=0 FAIL_COUNT=0 PARAMS_QUERY='' CALLS=0
GENERATOR_TEST_DIR="$1" VAULT="$2" EARNINGS="$2" fault="$3"
GUILD_ID=0-1 SETTLE_GUILD_VAULT=vault SETTLE_OWNER_VAULT=owner
SETTLE_PROV_ID=10-1 SETTLE_COLL_ADDR=collateral SETTLE_EARN_ADDR=earnings
POOL=17 OWNER=23
info() { :; }
fund_vault_alpha() { [ "$*" = '0-1 40000' ]; }
get_balance() {
    [ "$2" = ualpha ]
    case "$1" in vault) echo "$VAULT" ;; owner) echo "$OWNER" ;; collateral) echo "$POOL" ;; earnings) echo "$EARNINGS" ;; *) return 9 ;; esac
}
run_tx() {
    shift
    [ "$*" = 'tx structs agreement-open 10-1 20 2000 --owner-id 0-1 --from alice' ]
    CALLS=$((CALLS + 1))
    VAULT=$(python3 -c 'import sys; print(int(sys.argv[1])-40000)' "$VAULT")
    EARNINGS=$(python3 -c 'import sys; print(int(sys.argv[1])+40000)' "$EARNINGS")
    case "$fault" in
        early_refund) VAULT=$(python3 -c 'import sys; print(int(sys.argv[1])+1)' "$VAULT") ;;
        owner_debit) OWNER=22 ;;
    esac
}
get_opened_agreement_record() {
    [ "$*" = '10-1 0-1 2000' ]
    case "$fault" in
        missing_allocation) echo '{"id":"11-4","endBlock":"120","owner":"0-1"}'; return ;;
        wrong_allocation) echo '{"id":"11-4","endBlock":"120","owner":"0-1","allocationId":"10-4"}'; return ;;
    esac
    if [ "$fault" = wrong_owner ]; then echo '{"id":"11-4","endBlock":"120","allocationId":"6-4","owner":"1-1"}'; else echo '{"id":"11-4","endBlock":"120","allocationId":"6-4","owner":"0-1"}'; fi
}
wait_for_block() {
    [ "$*" = '122 120' ]
    case "$fault" in
        late_refund) VAULT=$(python3 -c 'import sys; print(int(sys.argv[1])+1)' "$VAULT") ;;
        owner_expiry) OWNER=24 ;;
        earnings) EARNINGS=$(python3 -c 'import sys; print(int(sys.argv[1])-1)' "$EARNINGS") ;;
        collateral) POOL=16 ;;
        timeout) return 1 ;;
    esac
}
structsd() {
    case "$*" in
        'query structs agreement 11-4') ;;
        'query structs allocation 6-4')
            case "$fault" in
                allocation_retained) echo '{"Allocation":{"id":"6-4"}}'; return ;;
                allocation_rpc) echo 'connection refused' >&2; return 1 ;;
            esac ;;
        *) return 9 ;;
    esac
    echo 'Error: object not found' >&2; return 1
}
query() {
    [ "$*" = 'query structs provider 10-1' ]
    case "$fault" in
        rpc) return 1 ;;
        load) echo '{"Provider":{"id":"10-1"},"gridAttributes":{"load":"2000"}}' ;;
        *) echo '{"Provider":{"id":"10-1"},"gridAttributes":{}}' ;;
    esac
}
'''
                    script += HELPERS + '\n' + BLOCK + '\n[ "$CALLS" -eq 1 ]\n[ "$FAIL_COUNT" -eq 0 ]\n'
                    result = subprocess.run(["bash", "-c", script, "test", str(ROOT.resolve()), str(balance), fault], capture_output=True, text=True)
                    if fault == "none":
                        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                    else:
                        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)


if __name__ == "__main__":
    unittest.main()
