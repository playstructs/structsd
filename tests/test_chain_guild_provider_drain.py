"""Guild provider deletion must account for both pools without shell overflow."""
from pathlib import Path
import re
import subprocess
import unittest

SOURCE = (Path(__file__).parent / "test_chain.sh").read_text()
HELPERS = "\n".join(re.search(r"(?ms)^" + name + r"\(\) \{.*?^\}", SOURCE).group()
                    for name in ("assert_eq", "assert_object_missing"))
BLOCK = 'SETTLE_TREASURY_BEFORE_DELETE=' + SOURCE.split('SETTLE_TREASURY_BEFORE_DELETE=', 1)[1].split('fi # settlement pool addresses', 1)[0]


class GuildProviderDrainTests(unittest.TestCase):
    def test_guild_receives_exact_pools_and_owner_receives_nothing(self):
        for vault, collateral, earnings in ((0, 0, 19), (100, 17, 19), (2**63+1, 17, 2**63+3), (2**64-1, 2**64-1, 2**64-1)):
            for fault in ("none", "vault", "owner", "collateral", "earnings", "retained_provider", "absence_rpc", "collateral_rpc", "earnings_rpc"):
                with self.subTest(vault=vault, collateral=collateral, earnings=earnings, fault=fault):
                    script = r'''
set -euo pipefail
GREEN='' RED='' NC='' PASS_COUNT=0 FAIL_COUNT=0 PARAMS_QUERY='' CALLS=0
SETTLE_PROV_ID=10-1 SETTLE_COLL_ADDR=collateral SETTLE_EARN_ADDR=earnings
SETTLE_GUILD_VAULT=vault SETTLE_OWNER_VAULT=owner OWNER=23
VAULT="$1" COLLATERAL="$2" EARNINGS="$3" fault="$4"
structsd() {
    [ "$*" = 'query structs provider 10-1' ]
    case "$fault" in
        retained_provider) echo '{"Provider":{"id":"10-1"}}'; return ;;
        absence_rpc) echo 'connection refused' >&2; return 1 ;;
        *) echo 'Error: object not found' >&2; return 1 ;;
    esac
}
get_balance() {
    [ "$2" = ualpha ]
    if [ "$CALLS" -eq 0 ] && { [ "$fault" = "${1}_rpc" ]; }; then return 1; fi
    case "$1" in vault) echo "$VAULT" ;; owner) echo "$OWNER" ;; collateral) echo "$COLLATERAL" ;; earnings) echo "$EARNINGS" ;; *) return 9 ;; esac
}
run_tx() {
    shift
    [ "$*" = 'tx structs provider-delete 10-1 --from alice' ]
    CALLS=$((CALLS + 1))
    echo 'DELETE_SENT'
    VAULT=$(python3 -c 'import sys; print(sum(map(int, sys.argv[1:])))' "$VAULT" "$COLLATERAL" "$EARNINGS")
    COLLATERAL=0 EARNINGS=0
    case "$fault" in
        vault) VAULT=$(python3 -c 'import sys; print(int(sys.argv[1])-1)' "$VAULT") ;;
        owner) OWNER=24 ;;
        collateral) COLLATERAL=1 ;;
        earnings) EARNINGS=1 ;;
    esac
}
'''
                    script += HELPERS + '\n' + BLOCK + '\n[ "$CALLS" -eq 1 ]\n[ "$PASS_COUNT" -eq 5 ]\n[ "$FAIL_COUNT" -eq 0 ]\n'
                    result = subprocess.run(["bash", "-c", script, "test", str(vault), str(collateral), str(earnings), fault], capture_output=True, text=True)
                    if fault == "none":
                        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                    else:
                        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
                    if fault in ("collateral_rpc", "earnings_rpc"):
                        self.assertNotIn("DELETE_SENT", result.stdout)


if __name__ == "__main__":
    unittest.main()
