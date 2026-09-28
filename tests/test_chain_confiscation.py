"""Run the actual confiscation block with independently corrupted ledgers."""
from pathlib import Path
import re
import subprocess
import unittest

SOURCE = (Path(__file__).parent / "test_chain.sh").read_text()
ASSERT = re.search(r"(?ms)^assert_eq\(\) \{.*?^\}", SOURCE).group()
PHASE = "CONFISCATE_AMOUNT=50000" + SOURCE.split("CONFISCATE_AMOUNT=50000", 1)[1].split(
    "# ─── Verify total supply decreased", 1)[0]


class ConfiscationPhaseTests(unittest.TestCase):
    def test_confiscation_checks_debit_burn_and_untouched_assets(self):
        for fault in ("none", "tokens", "supply", "collateral", "alpha", "key_tokens", "key_alpha", "treasury"):
            with self.subTest(fault=fault):
                script = r'''
set -euo pipefail
GREEN='' RED='' NC=''
PASS_COUNT=0 FAIL_COUNT=0 CALLS=0
PLAYER_3_VAULT=vault3 PLAYER_3_ADDRESS=key3 GUILD_VAULT=treasury
COLLATERAL_ADDR=pool GUILD_TOKEN_DENOM=uguild.0-1
TOKENS=150000 SUPPLY=1450000 COLLATERAL=1933334 ALPHA=333
KEY_TOKENS=71 KEY_ALPHA=777 TREASURY=125000
fault="$1"
info() { :; }
guild_supply() { [ "$1" = uguild.0-1 ]; echo "$SUPPLY"; }
get_balance() {
    case "$*" in
        'vault3 uguild.0-1') echo "$TOKENS" ;;
        'vault3 ualpha') echo "$ALPHA" ;;
        'key3 uguild.0-1') echo "$KEY_TOKENS" ;;
        'key3 ualpha') echo "$KEY_ALPHA" ;;
        'treasury uguild.0-1') echo "$TREASURY" ;;
        'pool ualpha') echo "$COLLATERAL" ;;
        *) return 9 ;;
    esac
}
run_tx() {
    shift
    [ "$*" = 'tx structs guild-bank-confiscate-and-burn 50000 vault3 --from alice' ]
    CALLS=$((CALLS + 1))
    TOKENS=$((TOKENS - 50000)); SUPPLY=$((SUPPLY - 50000))
    case "$fault" in
        tokens) TOKENS=$((TOKENS + 1)) ;;
        supply) SUPPLY=$((SUPPLY + 1)) ;;
        collateral) COLLATERAL=$((COLLATERAL - 1)) ;;
        alpha) ALPHA=$((ALPHA + 1)) ;;
        key_tokens) KEY_TOKENS=$((KEY_TOKENS - 1)) ;;
        key_alpha) KEY_ALPHA=$((KEY_ALPHA - 1)) ;;
        treasury) TREASURY=$((TREASURY + 1)) ;;
    esac
}
'''
                script += ASSERT + "\n" + PHASE
                script += '\n[ "$CALLS" -eq 1 ]\n[ "$((PASS_COUNT + FAIL_COUNT))" -eq 7 ]\n'
                script += '[ "$FAIL_COUNT" -eq "$2" ]\n'
                result = subprocess.run(
                    ["bash", "-c", script, "test", fault, "0" if fault == "none" else "1"],
                    capture_output=True, text=True)
                self.assertEqual(result.returncode, 0, result.stdout + result.stderr)


if __name__ == "__main__":
    unittest.main()
