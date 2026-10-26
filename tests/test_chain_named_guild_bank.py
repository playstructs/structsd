"""Exercise both named-guild bank transactions and their actual assertions."""
from pathlib import Path
import re
import subprocess
import unittest

SOURCE = (Path(__file__).parent / "test_chain.sh").read_text()
HELPERS = "\n".join(re.search(r"(?ms)^" + name + r"\(\) \{.*?^\}", SOURCE).group()
                    for name in ("assert_eq", "assert_not_empty"))
PHASE = SOURCE.split("# ─── guild-bank-mint / guild-bank-confiscate-and-burn --guild-id ───", 1)[1].split(
    "# Naming a guild", 1)[0]


class NamedGuildBankPhaseTests(unittest.TestCase):
    def test_named_guild_uses_only_its_treasury_and_collateral(self):
        for stage in ("mint", "burn"):
            for fault in ("none", "tokens", "alpha", "collateral", "supply", "personal", "membership"):
                with self.subTest(stage=stage, fault=fault):
                    script = r'''
set -euo pipefail
GREEN='' RED='' NC=''
PASS_COUNT=0 FAIL_COUNT=0 CALLS=0
GUILD_C_ID=0-3 GUILD_B_ID=0-2 GUILD_LEADER_B_ID=1-2
TOKENS=1000000 ALPHA=200 POOL=1000000 SUPPLY=1000000 PERSONAL=456 MEMBERSHIP=789
stage="$1" fault="$2"
info() { :; }
get_vault_address() {
    case "$1" in 0-3) echo vaultC ;; 0-2) echo vaultB ;; 1-2) echo personal ;; *) return 9 ;; esac
}
fund_vault_alpha() { [ "$*" = '0-3 100' ]; ALPHA=$((ALPHA + 100)); }
query() { [ "$*" = 'query structs guild-bank-collateral-address 0-3' ]; echo '{"internalAddressAssociation":[{"address":"pool"}]}'; }
jqr() { echo "$1" | jq -er "$2"; }
guild_supply() { [ "$1" = uguild.0-3 ]; echo "$SUPPLY"; }
get_balance() {
    case "$*" in
        'vaultC uguild.0-3') echo "$TOKENS" ;;
        'vaultC ualpha') echo "$ALPHA" ;;
        'pool ualpha') echo "$POOL" ;;
        'personal ualpha') echo "$PERSONAL" ;;
        'vaultB ualpha') echo "$MEMBERSHIP" ;;
        *) return 9 ;;
    esac
}
run_tx() {
    shift
    CALLS=$((CALLS + 1))
    local current
    if [ "$CALLS" -eq 1 ]; then
        [ "$*" = 'tx structs guild-bank-mint 100 100 --guild-id 0-3 --from guild_leader_b' ]
        current=mint
        TOKENS=$((TOKENS + 100)); ALPHA=$((ALPHA - 100)); POOL=$((POOL + 100)); SUPPLY=$((SUPPLY + 100))
    else
        [ "$*" = 'tx structs guild-bank-confiscate-and-burn 40 vaultC --guild-id 0-3 --from guild_leader_b' ]
        current=burn
        TOKENS=$((TOKENS - 40)); SUPPLY=$((SUPPLY - 40))
    fi
    if [ "$current" = "$stage" ]; then
        case "$fault" in
            tokens) TOKENS=$((TOKENS + 1)) ;;
            alpha) ALPHA=$((ALPHA + 1)) ;;
            collateral) POOL=$((POOL - 1)) ;;
            supply) SUPPLY=$((SUPPLY + 1)) ;;
            personal) PERSONAL=$((PERSONAL - 1)) ;;
            membership) MEMBERSHIP=$((MEMBERSHIP - 1)) ;;
        esac
    fi
}
'''
                    script += HELPERS + "\n" + PHASE
                    script += '\n[ "$CALLS" -eq 2 ]\n[ "$((PASS_COUNT + FAIL_COUNT))" -eq 13 ]\n'
                    script += '[ "$FAIL_COUNT" -eq "$3" ]\n'
                    failures = 0 if fault == "none" else (2 if stage == "mint" else 1)
                    result = subprocess.run(
                        ["bash", "-c", script, "test", stage, fault, str(failures)],
                        capture_output=True, text=True)
                    self.assertEqual(result.returncode, 0, result.stdout + result.stderr)


if __name__ == "__main__":
    unittest.main()
