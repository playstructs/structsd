"""Deletion must drain two nonempty pools into the provider owner's vault."""
from pathlib import Path
import re
import subprocess
import unittest

SOURCE = (Path(__file__).parent / "test_chain.sh").read_text()
HELPERS = "\n".join(re.search(r"(?ms)^" + name + r"\(\) \{.*?^\}", SOURCE).group()
                    for name in ("assert_eq", "assert_object_missing"))
BLOCK = SOURCE.split("# ─── provider-delete ───", 1)[1].split(
    "# ─── provider-delete accounting complete ───", 1)[0]


class ProviderDrainTests(unittest.TestCase):
    def test_both_pools_are_reachable_until_drained(self):
        for vault in (100, 2**53 + 1, 2**64 - 1):
            for fault in ("none", "vault", "collateral", "earnings", "retained_provider", "active_load", "seed_missing", "rpc"):
                with self.subTest(vault=vault, fault=fault):
                    script = r'''
set -euo pipefail
GREEN='' RED='' NC='' PASS_COUNT=0 FAIL_COUNT=0 PARAMS_QUERY='' CALLS=0
PROVIDER_ID=10-1 PROV_COLL_ADDR=collateral PROV_EARN_ADDR=earnings PLAYER_1_VAULT=vault ALICE_ADDRESS=alice
VAULT="$1" COLLATERAL=17 EARNINGS=0 fault="$2"
query() {
    [ "$*" = 'query structs provider 10-1' ]
    if [ "$fault" = active_load ]; then echo '{"gridAttributes":{"load":"1"}}'; else echo '{"gridAttributes":{}}'; fi
}
structsd() {
    [ "$*" = 'query structs provider 10-1' ]
    case "$fault" in
        retained_provider) echo '{"Provider":{"id":"10-1"}}'; return ;;
        rpc) echo 'connection refused' >&2; return 1 ;;
        *) echo 'Error: object not found' >&2; return 1 ;;
    esac
}
get_balance() {
    [ "$2" = ualpha ]
    case "$1" in vault) echo "$VAULT" ;; collateral) echo "$COLLATERAL" ;; earnings) echo "$EARNINGS" ;; *) return 9 ;; esac
}
run_tx() {
    shift
    CALLS=$((CALLS + 1))
    case "$CALLS" in
        1) [ "$*" = 'tx bank send alice collateral 1ualpha --from alice' ]; COLLATERAL=18 ;;
        2) [ "$*" = 'tx bank send alice earnings 1ualpha --from alice' ]; if [ "$fault" != seed_missing ]; then EARNINGS=1; fi ;;
        3)
            [ "$*" = 'tx structs provider-delete 10-1 --from alice' ]
            VAULT=$(python3 -c 'import sys; print(int(sys.argv[1])+19)' "$VAULT")
            COLLATERAL=0 EARNINGS=0
            case "$fault" in
                vault) VAULT=$(python3 -c 'import sys; print(int(sys.argv[1])-1)' "$VAULT") ;;
                collateral) COLLATERAL=1 ;;
                earnings) EARNINGS=1 ;;
            esac ;;
        *) return 9 ;;
    esac
}
'''
                    script += HELPERS + '\n' + BLOCK + '\n[ "$CALLS" -eq 3 ]\n[ "$PASS_COUNT" -eq 5 ]\n[ "$FAIL_COUNT" -eq 0 ]\n'
                    result = subprocess.run(["bash", "-c", script, "test", str(vault), fault], capture_output=True, text=True)
                    if fault == "none":
                        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                    else:
                        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)


if __name__ == "__main__":
    unittest.main()
