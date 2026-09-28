"""Run actual conversion assertion blocks against an independently updated ledger."""
from pathlib import Path
import re
import subprocess
import unittest

SOURCE = (Path(__file__).parent / "test_chain.sh").read_text()
ASSERT = re.search(r"(?ms)^assert_eq\(\) \{.*?^\}", SOURCE).group()
CONVERT = "CONVERT_ALPHA=100000" + SOURCE.split("CONVERT_ALPHA=100000", 1)[1].split("# ─── Set a 10% convert-out fee", 1)[0]
CROSS = "XCONVERT_TOKENS=20000" + SOURCE.split("XCONVERT_TOKENS=20000", 1)[1].split("# Same-guild convert-token is rejected.", 1)[0]
SETUP = r'''
set -euo pipefail
GREEN='' RED='' NC='' PASS_COUNT=0 FAIL_COUNT=0 CALLS=0
PLAYER_2_VAULT=vault COLLATERAL_ADDR=poolA B_COLLATERAL_ADDR=poolB
GUILD_ID=0-1 GUILD_B_ID=0-2 GUILD_TOKEN_DENOM=uguild.0-1 GUILD_B_TOKEN_DENOM=uguild.0-2
ALPHA=200000 TOKEN_A=200000 TOKEN_B=0 POOL_A=2000000 POOL_B=2000000 SUPPLY_A=1500000 SUPPLY_B=1500000
fault="$1"
info() { :; }
guild_supply() {
    case "$1" in uguild.0-1) echo "$SUPPLY_A" ;; uguild.0-2) echo "$SUPPLY_B" ;; *) return 9 ;; esac
}
get_balance() {
    case "$*" in
        'vault ualpha') echo "$ALPHA" ;;
        'vault uguild.0-1') echo "$TOKEN_A" ;;
        'vault uguild.0-2') echo "$TOKEN_B" ;;
        'poolA ualpha') echo "$POOL_A" ;;
        'poolB ualpha') echo "$POOL_B" ;;
        *) return 9 ;;
    esac
}
corrupt() {
    case "$fault" in
        "$1:alpha") ALPHA=$((ALPHA + 1)) ;;
        "$1:token_a") TOKEN_A=$((TOKEN_A + 1)) ;;
        "$1:token_b") TOKEN_B=$((TOKEN_B + 1)) ;;
        "$1:pool_a") POOL_A=$((POOL_A + 1)) ;;
        "$1:pool_b") POOL_B=$((POOL_B + 1)) ;;
        "$1:supply_a") SUPPLY_A=$((SUPPLY_A + 1)) ;;
        "$1:supply_b") SUPPLY_B=$((SUPPLY_B + 1)) ;;
    esac
}
'''


class ConversionPhaseTests(unittest.TestCase):
    def check_script(self, script, fault, calls, assertions):
        script += f'\n[ "$CALLS" -eq {calls} ]\n[ "$((PASS_COUNT + FAIL_COUNT))" -eq {assertions} ]\n'
        script += f'[ "$FAIL_COUNT" -eq {0 if fault == "none" else 1} ]\n'
        result = subprocess.run(["bash", "-c", script, "test", fault], capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_conversion_and_slippage_require_all_ledger_changes(self):
        for fault in ["none"] + [stage+":"+field for stage in ("convert", "reject")
                for field in ("alpha", "token_a", "pool_a", "supply_a")]:
            with self.subTest(fault=fault):
                operations = r'''
run_tx() {
    shift
    [ "$*" = 'tx structs guild-bank-convert 0-1 100000 1 --from player_2' ]
    CALLS=$((CALLS + 1))
    ALPHA=100000 TOKEN_A=267500 POOL_A=2100000 SUPPLY_A=1567500
    corrupt convert
}
run_tx_expect_reject_matching() {
    shift
    [ "$1" = slippage ]; shift
    [ "$*" = 'tx structs guild-bank-convert 0-1 100000 999999999 --from player_2' ]
    CALLS=$((CALLS + 1))
    corrupt reject
}
'''
                self.check_script(SETUP + operations + ASSERT + "\n" + CONVERT, fault, 2, 8)

    def test_cross_conversion_requires_both_pools_and_supplies(self):
        for fault in ["none"] + ["cross:"+field for field in
                ("alpha", "token_a", "token_b", "pool_a", "pool_b", "supply_a", "supply_b")]:
            with self.subTest(fault=fault):
                operations = r'''
run_tx() {
    shift
    [ "$*" = 'tx structs guild-bank-convert-token 20000uguild.0-1 0-2 1 --from player_2' ]
    CALLS=$((CALLS + 1))
    # floor(20k*2M/1.5M)=26666; ceil(10%)=2667; bridge=23999.
    # B fee=2400; floor(21599*1.5M/2M)=16199.
    TOKEN_A=180000 TOKEN_B=16199 POOL_A=1976001 POOL_B=2023999 SUPPLY_A=1480000 SUPPLY_B=1516199
    corrupt cross
}
'''
                self.check_script(SETUP + operations + ASSERT + "\n" + CROSS, fault, 1, 7)


if __name__ == "__main__":
    unittest.main()
