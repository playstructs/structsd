"""Guild earnings withdrawal must respect rejection and vault isolation."""
from pathlib import Path
import re
import subprocess
import unittest

SOURCE = (Path(__file__).parent / 'test_chain.sh').read_text()
HELPER = re.search(r'(?ms)^assert_eq\(\) \{.*?^\}', SOURCE).group()
BLOCK = 'SETTLE_TREASURY_BEFORE_WITHDRAW=' + SOURCE.split('SETTLE_TREASURY_BEFORE_WITHDRAW=', 1)[1].split('fi # settlement agreement B opened', 1)[0]


class GuildWithdrawalTests(unittest.TestCase):
    def test_rejection_and_success_preserve_other_balances(self):
        for balance in (100, 2**63+1, 2**64-1):
            for fault in ('none', 'reject_earnings', 'reject_collateral', 'reject_guild', 'reject_owner', 'reject_member',
                          'success_earnings', 'success_collateral', 'success_guild', 'success_owner', 'success_member', 'snapshot_rpc'):
                with self.subTest(balance=balance, fault=fault):
                    script = r'''
set -euo pipefail
GREEN='' RED='' NC='' PASS_COUNT=0 FAIL_COUNT=0 CALLS=0
SETTLE_GUILD_VAULT=guild SETTLE_OWNER_VAULT=owner PLAYER_2_VAULT=member
SETTLE_EARN_ADDR=earnings SETTLE_COLL_ADDR=collateral SETTLE_PROV_ID=10-1
SETTLE_EARN_END="$1" SETTLE_POOL_END=0 GUILD="$1" EARNINGS="$1" OWNER=23 MEMBER=29 COLLATERAL=0 fault="$2"
get_balance() {
    [ "$2" = ualpha ]
    if [ "$CALLS" -eq 0 ] && [ "$fault" = snapshot_rpc ] && [ "$1" = member ]; then return 1; fi
    case "$1" in
        guild) echo "$GUILD" ;; owner) echo "$OWNER" ;; member) echo "$MEMBER" ;;
        earnings) echo "$EARNINGS" ;; collateral) echo "$COLLATERAL" ;; *) return 9 ;;
    esac
}
run_tx_expect_permission_denied() {
    shift
    [ "$*" = 'tx structs provider-withdraw-balance 10-1 --from player_2' ]
    CALLS=$((CALLS+1))
    echo DENIAL_SENT
    case "$fault" in
        reject_earnings) EARNINGS=$(python3 -c 'import sys; print(int(sys.argv[1])-1)' "$EARNINGS") ;;
        reject_collateral) COLLATERAL=1 ;;
        reject_guild) GUILD=$(python3 -c 'import sys; print(int(sys.argv[1])+1)' "$GUILD") ;;
        reject_owner) OWNER=24 ;;
        reject_member) MEMBER=30 ;;
    esac
}
run_tx() {
    shift
    [ "$*" = 'tx structs provider-withdraw-balance 10-1 --from alice' ]
    CALLS=$((CALLS+1))
    # Restore a rejected-attempt fault before the later successful operation:
    # only the immediate rejection checks can detect that transient mutation.
    case "$fault" in reject_*) GUILD="$SETTLE_EARN_END" EARNINGS="$SETTLE_EARN_END" OWNER=23 MEMBER=29 COLLATERAL=0 ;; esac
    GUILD=$(python3 -c 'import sys; print(sum(map(int,sys.argv[1:])))' "$GUILD" "$EARNINGS")
    EARNINGS=0
    case "$fault" in
        success_earnings) EARNINGS=1 ;;
        success_collateral) COLLATERAL=1 ;;
        success_guild) GUILD=$(python3 -c 'import sys; print(int(sys.argv[1])-1)' "$GUILD") ;;
        success_owner) OWNER=24 ;;
        success_member) MEMBER=30 ;;
    esac
}
'''
                    script += HELPER + '\n' + BLOCK + '\n[ "$CALLS" -eq 2 ]\n[ "$FAIL_COUNT" -eq 0 ]\n'
                    result = subprocess.run(['bash', '-c', script, 'test', str(balance), fault], capture_output=True, text=True)
                    if fault == 'none':
                        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                    else:
                        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
                    if fault == 'snapshot_rpc':
                        self.assertNotIn('DENIAL_SENT', result.stdout)


if __name__ == '__main__':
    unittest.main()
