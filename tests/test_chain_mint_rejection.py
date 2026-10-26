"""Exercise the rejected treasury-mint block against independent ledgers."""
from pathlib import Path
import re
import subprocess
import unittest

SOURCE = (Path(__file__).parent / "test_chain.sh").read_text()
ASSERT = re.search(r"(?ms)^assert_eq\(\) \{.*?^\}", SOURCE).group()
PERMISSION = re.search(r"(?ms)^run_tx_expect_permission_denied\(\) \{.*?^\}", SOURCE).group()
PHASE = SOURCE.split("# ─── Attempt unauthorized mint (Player 2 should fail) ───", 1)[1].split("# ═════════", 1)[0]


class MintRejectionPhaseTests(unittest.TestCase):
    def test_rejected_mint_requires_permission_reason_and_unchanged_ledgers(self):
        for fault in ("none", "player_tokens", "player_alpha", "guild_tokens", "guild_alpha", "collateral", "supply", "wrong_reason"):
            with self.subTest(fault=fault):
                script = r'''
set -euo pipefail
GREEN='' RED='' NC=''
PASS_COUNT=0 FAIL_COUNT=0 CALLS=0
PLAYER_2_VAULT=player GUILD_VAULT=guild COLLATERAL_ADDR=pool GUILD_TOKEN_DENOM=uguild.0-1
PLAYER_TOKENS=123 PLAYER_ALPHA=456 GUILD_TOKENS=789 GUILD_ALPHA=111
COLLATERAL=222 SUPPLY=912
fault="$1"
info() { :; }
guild_supply() { [ "$1" = uguild.0-1 ]; echo "$SUPPLY"; }
get_balance() {
    case "$*" in
        'player uguild.0-1') echo "$PLAYER_TOKENS" ;;
        'player ualpha') echo "$PLAYER_ALPHA" ;;
        'guild uguild.0-1') echo "$GUILD_TOKENS" ;;
        'guild ualpha') echo "$GUILD_ALPHA" ;;
        'pool ualpha') echo "$COLLATERAL" ;;
        *) return 9 ;;
    esac
}
run_tx_expect_reject_matching() {
    local pattern="$2"
    shift 2
    [ "$*" = 'tx structs guild-bank-mint 100 100 --from player_2' ]
    CALLS=$((CALLS + 1))
    local reason='calling player (1-2) has no mint permission on guild (0-1)'
    case "$fault" in
        player_tokens) PLAYER_TOKENS=$((PLAYER_TOKENS + 1)) ;;
        player_alpha) PLAYER_ALPHA=$((PLAYER_ALPHA - 1)) ;;
        guild_tokens) GUILD_TOKENS=$((GUILD_TOKENS + 1)) ;;
        guild_alpha) GUILD_ALPHA=$((GUILD_ALPHA - 1)) ;;
        collateral) COLLATERAL=$((COLLATERAL + 1)) ;;
        supply) SUPPLY=$((SUPPLY + 1)) ;;
        wrong_reason) reason='out of gas' ;;
    esac
    if [[ "$reason" =~ $pattern ]]; then
        PASS_COUNT=$((PASS_COUNT + 1))
    else
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
}
'''
                script += ASSERT + "\n" + PERMISSION + "\n" + PHASE
                script += '\n[ "$CALLS" -eq 1 ]\n[ "$((PASS_COUNT + FAIL_COUNT))" -eq 7 ]\n'
                script += '[ "$FAIL_COUNT" -eq "$2" ]\n'
                result = subprocess.run(
                    ["bash", "-c", script, "test", fault, "0" if fault == "none" else "1"],
                    capture_output=True, text=True)
                self.assertEqual(result.returncode, 0, result.stdout + result.stderr)


if __name__ == "__main__":
    unittest.main()
