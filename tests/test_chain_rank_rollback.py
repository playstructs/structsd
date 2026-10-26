"""Exercise the live script's rejection assertions without starting a chain."""
from pathlib import Path
import re
import subprocess
import unittest

SOURCE = (Path(__file__).parent / "test_chain.sh").read_text()
ASSERT = re.search(r"(?ms)^assert_eq\(\) \{.*?^\}", SOURCE).group()
SETUP = r'''
set -euo pipefail
GREEN='' RED='' NC='' PASS_COUNT=0 FAIL_COUNT=0 CALLS=0
GUILD_ID=0-1 GUILD_TOKEN_DENOM=uguild.0-1
PLAYER_2_ID=1-2 PLAYER_5_ID=1-5
PLAYER_2_VAULT=member PLAYER_5_VAULT=outsider GUILD_VAULT=treasury
RANK_P2_BEFORE=125 RANK_GUILD_BEFORE=1000 RANK_SUPPLY_BEFORE=5000
RANK_P5_BEFORE=75 RANK_GUILD_BEFORE_REJECT=1100 RANK_P5_BEFORE_REJECT=101
TOKEN="$2" TREASURY="$3" RANK="$4" SUPPLY=5000 MEMBERSHIP=''
fault="$1"
get_balance() {
    case "$1" in member|outsider) echo "$TOKEN" ;; treasury) echo "$TREASURY" ;; *) return 9 ;; esac
}
guild_supply() { echo "$SUPPLY"; }
get_player_guild_rank() { echo "$RANK"; }
query() { printf '{"Player":{"guildId":"%s"}}\n' "$MEMBERSHIP"; }
run_tx_expect_reject_matching() {
    CALLS=$((CALLS + 1))
    case "$fault" in
        token) TOKEN=$((TOKEN + 1)) ;;
        treasury) TREASURY=$((TREASURY + 1)) ;;
        supply) SUPPLY=$((SUPPLY + 1)) ;;
        rank) RANK=$((RANK - 1)) ;;
        membership) MEMBERSHIP=0-1 ;;
        none) : ;;
        *) return 9 ;;
    esac
}
'''


class RankRollbackTests(unittest.TestCase):
    def test_each_rejection_checks_state_before_the_next_transaction(self):
        cases = [
            ("Rank purchase rejects an insufficient quote ceiling", 125, 1000, 101, False),
            ("Stale expected rank cannot buy again", 115, 1010, 100, False),
            ("Normal rank purchase cannot admit a non-member", 75, 1100, 101, True),
            ("Purchase admission requires its explicit guild setting", 75, 1100, 101, True),
        ]
        for label, token, treasury, rank, admission in cases:
            marker = f'run_tx_expect_reject_matching "{label}"'
            block = marker + SOURCE.split(marker, 1)[1]
            block = re.split(r"\nrun_tx(?:_expect_reject_matching)? ", block, maxsplit=1)[0]
            for fault in ["none", "token", "treasury", "supply", "rank"] + (["membership"] if admission else []):
                with self.subTest(case=label, fault=fault):
                    expected_assertions = 5 if admission else 4
                    script = SETUP + ASSERT + "\n" + block + "\n"
                    script += '[ "$CALLS" -eq 1 ]\n'
                    script += f'[ "$((PASS_COUNT + FAIL_COUNT))" -eq {expected_assertions} ]\n'
                    script += f'[ "$FAIL_COUNT" -eq {0 if fault == "none" else 1} ]\n'
                    result = subprocess.run(
                        ["bash", "-c", script, "test", fault, str(token), str(treasury), str(rank)],
                        capture_output=True, text=True,
                    )
                    self.assertEqual(result.returncode, 0, result.stdout + result.stderr)


if __name__ == "__main__":
    unittest.main()
