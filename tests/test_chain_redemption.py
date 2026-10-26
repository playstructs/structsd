"""Exercise the actual redemption assertions against independent fake ledger changes."""
from pathlib import Path
import re
import subprocess
import unittest

SOURCE = (Path(__file__).parent / "test_chain.sh").read_text()
ASSERT = re.search(r"(?ms)^assert_eq\(\) \{.*?^\}", SOURCE).group()
INITIAL = SOURCE.split("REDEEM_AMOUNT=50000", 1)[1].split("# ─── Confiscate and burn", 1)[0]
INITIAL = "REDEEM_AMOUNT=50000" + INITIAL
FEE = SOURCE.split("FEE_REDEEM_AMOUNT=40000", 1)[1].split("# ─── Cross-guild convert:", 1)[0]
FEE = "FEE_REDEEM_AMOUNT=40000" + FEE


class RedemptionPhaseTests(unittest.TestCase):
    def test_both_redemptions_require_all_ledger_changes(self):
        for phase, amount, payout in ((INITIAL, 50000, 66666), (FEE, 40000, 47999)):
            for fault in ("none", "credit", "collateral", "tokens", "supply"):
                with self.subTest(amount=amount, fault=fault):
                    # Supply includes 1M founding units plus the 500k mint;
                    # collateral includes the founding 1M plus the 1M deposit.
                    script = r'''
set -euo pipefail
GREEN='' RED='' NC=''
PASS_COUNT=0 FAIL_COUNT=0 CALLS=0
PLAYER_2_VAULT=vault2 COLLATERAL_ADDR=pool GUILD_TOKEN_DENOM=uguild.0-1
ALPHA=200000 TOKENS=200000 POOL=2000000 SUPPLY=1500000
P2_TOKEN_AFTER_TRANSFER=200000 COLLATERAL_ALPHA_AFTER_MINT=2000000
fault="$1" amount="$2" payout="$3"
info() { :; }
guild_supply() { [ "$1" = uguild.0-1 ]; echo "$SUPPLY"; }
get_balance() {
    case "$*" in
        'vault2 ualpha') echo "$ALPHA" ;;
        'vault2 uguild.0-1') echo "$TOKENS" ;;
        'pool ualpha') echo "$POOL" ;;
        *) return 9 ;;
    esac
}
run_tx() {
    shift
    [ "$*" = "tx structs guild-bank-redeem ${amount}uguild.0-1 1 --from player_2" ]
    CALLS=$((CALLS + 1))
    # A one-unit discrepancy still satisfies the old directional assertions.
    ALPHA=$((ALPHA + payout)); POOL=$((POOL - payout))
    TOKENS=$((TOKENS - amount)); SUPPLY=$((SUPPLY - amount))
    case "$fault" in
        credit) ALPHA=$((ALPHA - 1)) ;;
        collateral) POOL=$((POOL - 1)) ;;
        tokens) TOKENS=$((TOKENS + 1)) ;;
        supply) SUPPLY=$((SUPPLY + 1)) ;;
    esac
}
'''
                    script += ASSERT + "\n" + phase
                    script += '\n[ "$CALLS" -eq 1 ]\n[ "$((PASS_COUNT + FAIL_COUNT))" -eq 4 ]\n'
                    script += '[ "$FAIL_COUNT" -eq "$4" ]\n'
                    result = subprocess.run(["bash", "-c", script, "test", fault, str(amount), str(payout),
                        "0" if fault == "none" else "1"], capture_output=True, text=True)
                    self.assertEqual(result.returncode, 0, result.stdout + result.stderr)


if __name__ == "__main__":
    unittest.main()
