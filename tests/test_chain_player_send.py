"""Exercise the actual PlayerSend phase with offline balance changes."""
from pathlib import Path
import re
import subprocess
import unittest

SOURCE = (Path(__file__).parent / "test_chain.sh").read_text()
ASSERT = re.search(r"(?ms)^assert_eq\(\) \{.*?^\}", SOURCE).group()
PHASE = SOURCE.split("# ─── player-send: fund a signing key", 1)[1].split("# Use a fresh player:", 1)[0]
PHASE = PHASE.split("\n", 1)[1]


class PlayerSendPhaseTests(unittest.TestCase):
    def test_empty_recipient_and_missing_bank_changes(self):
        for fault, expected_failures in (("none", 0), ("vault_debit", 1), ("recipient_credit", 1)):
            script = r'''
set -euo pipefail
GREEN='' RED='' NC=''
PASS_COUNT=0 FAIL_COUNT=0 CALLS=0
PLAYER_2_ID=1-2 PLAYER_2_ADDRESS=key2 PLAYER_3_ADDRESS=key3
VAULT=100000 KEY2=0 KEY3=0
fault="$1"
get_vault_address() { [ "$1" = 1-2 ]; echo vault2; }
fund_vault_alpha() { [ "$*" = '1-2 100000' ]; }
get_balance() {
    [ "$2" = ualpha ]
    case "$1" in
        vault2) echo "$VAULT" ;;
        key2) echo "$KEY2" ;;
        key3) echo "$KEY3" ;;
        *) return 9 ;;
    esac
}
run_tx() {
    shift
    CALLS=$((CALLS + 1))
    if [ "$CALLS" -eq 1 ]; then
        [ "$*" = 'tx structs player-send vault2 key2 100000ualpha --from player_2' ]
        if [ "$fault" != vault_debit ]; then VAULT=0; fi
        KEY2=100000
    else
        [ "$CALLS" -eq 2 ]
        [ "$*" = 'tx structs player-send key2 key3 100000ualpha --from player_2' ]
        KEY2=0
        if [ "$fault" != recipient_credit ]; then KEY3=100000; fi
    fi
}
'''
            script += ASSERT + "\n" + PHASE
            script += '\n[ "$CALLS" -eq 2 ]\n[ "$FAIL_COUNT" -eq "$2" ]\n[ "$((PASS_COUNT + FAIL_COUNT))" -eq 4 ]\n'
            result = subprocess.run(["bash", "-c", script, "test", fault, str(expected_failures)], capture_output=True, text=True)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)


if __name__ == "__main__":
    unittest.main()
