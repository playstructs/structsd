"""Verify the actual phase-16 close block against cumulative window settlement."""
import json
from pathlib import Path
import re
import subprocess
import tempfile
import unittest

from agreement_terms import update_terms
from test_chain_agreement_terms import HASH, INITIAL, transcript

ROOT = Path(__file__).resolve().parent
SOURCE = (ROOT / 'test_chain.sh').read_text()
ASSERT = re.search(r'(?ms)^assert_eq\(\) \{.*?^\}', SOURCE).group()
TRACKER = re.search(r'(?ms)^record_agreement_terms\(\) \{.*?^\}', SOURCE).group()
BLOCK = SOURCE.split('# ─── agreement-close ───', 1)[1].split('# ─── agreement-close accounting complete ───', 1)[0]


class AgreementRefundTests(unittest.TestCase):
    def test_completed_provider_withdrawal_uses_exact_large_balances(self):
        block = 'ALICE_VAULT_BEFORE_WITHDRAW=' + SOURCE.split('ALICE_VAULT_BEFORE_WITHDRAW=', 1)[1].split('# ─── provider-delete ───', 1)[0]
        for value in (100, 2**64-1):
            for fault in ('none', 'short_credit', 'leftover_earnings'):
                script = r'''set -euo pipefail
GREEN='' RED='' NC='' PASS_COUNT=0 FAIL_COUNT=0
PLAYER_1_VAULT=vault PROV_EARN_ADDR=earnings PROVIDER_ID=10-1
VAULT="$1" PROV_EARN_BEFORE="$1" EARNINGS="$1" fault="$2"
get_balance() {
    [ "$2" = ualpha ] || return 9
    case "$1" in vault) echo "$VAULT" ;; earnings) echo "$EARNINGS" ;; *) return 9 ;; esac
}
run_tx() {
    shift
    [ "$*" = 'tx structs provider-withdraw-balance 10-1 --from alice' ] || return 9
    VAULT=$(python3 -c 'import sys; print(sum(map(int,sys.argv[1:])))' "$VAULT" "$EARNINGS")
    EARNINGS=0
    if [ "$fault" = short_credit ]; then VAULT=$(python3 -c 'import sys; print(int(sys.argv[1])-1)' "$VAULT"); fi
    if [ "$fault" = leftover_earnings ]; then EARNINGS=1; fi
}
'''
                script += ASSERT+'\n'+block+'\n[ "$FAIL_COUNT" -eq 0 ]\n'
                result = subprocess.run(['bash', '-c', script, 'test', str(value), fault], capture_output=True, text=True)
                if fault == 'none':
                    self.assertEqual(result.returncode, 0, result.stdout+result.stderr)
                else:
                    self.assertNotEqual(result.returncode, 0)

    def test_delivered_windows_and_all_three_final_ledgers(self):
        state = INITIAL
        for op, amount, tx in transcript()[:-1]:
            state = update_terms(state, op, amount, tx, HASH)
        for vault in (4000000, 2**64-1):
            for fault in ('none', 'vault', 'collateral', 'earnings', 'wrong_height', 'failed_tx',
                          'wrong_hash', 'penalty', 'wrong_rate'):
                with self.subTest(vault=vault, fault=fault), tempfile.TemporaryDirectory() as directory:
                    directory = Path(directory)
                    for name in ('agreement_terms.py', 'economic_assertions.py'):
                        (directory/name).symlink_to(ROOT/name)
                    tx = transcript()[-1][2]
                    if fault == 'wrong_height':
                        tx['height'] = '121'
                    if fault == 'failed_tx':
                        tx['code'] = 5
                    if fault == 'wrong_hash':
                        tx['txhash'] = 'B'*64
                    script = r'''set -euo pipefail
GREEN='' RED='' NC='' PASS_COUNT=0 FAIL_COUNT=0 CALLS=0
AGREE_TERMS_TEST_DIR="$1" P2_OPEN_VAULT_BEFORE="$2" fault="$3" AGREE_SERVICE_TERMS="$4" TX_PAYLOAD="$5"
PROVIDER_ID=10-1 AGREE_ID=11-1 PLAYER_2_ID=1-2 PLAYER_2_VAULT=vault PROV_OPEN_COLL_ADDR=pool
PROV_OPEN_COLL_BEFORE=17 PROV_OPEN_EARN_BEFORE=19 PROV_OPEN_EARN_ADDR=earnings
LAST_TX_OUTPUT='{"txhash":"''' + HASH + r'''"}'
_tx_json() { printf '%s' "$1"; }
query() {
    case "$*" in
        'query structs provider 10-1')
            PENALTY=0.000000000000000000 RATE=1
            if [ "$fault" = penalty ]; then PENALTY=0.5; fi
            if [ "$fault" = wrong_rate ]; then RATE=2; fi
            printf '{"Provider":{"rate":{"denom":"ualpha","amount":"%s"},"providerCancellationPenalty":"0","consumerCancellationPenalty":"%s"}}' "$RATE" "$PENALTY" ;;
        'query tx ''' + HASH + r'''') printf '%s' "$TX_PAYLOAD" ;;
        *) return 9 ;;
    esac
}
run_tx() {
    shift
    [ "$*" = 'tx structs agreement-close 11-1 --from player_2' ] || return 9
    CALLS=$((CALLS+1))
}
get_balance() {
    [ "$CALLS" -eq 1 ] || return 8
    [ "$2" = ualpha ] || return 9
    case "$1" in
        vault)
            delta=0; if [ "$fault" = vault ]; then delta=1; fi
            python3 -c 'import sys; print(int(sys.argv[1])-1350000+int(sys.argv[2]))' "$P2_OPEN_VAULT_BEFORE" "$delta" ;;
        pool) if [ "$fault" = collateral ]; then echo 75016; else echo 75017; fi ;;
        earnings) if [ "$fault" = earnings ]; then echo 1275020; else echo 1275019; fi ;;
        *) return 9 ;;
    esac
}
'''
                    script += ASSERT+'\n'+TRACKER+'\n'+BLOCK+'\n[ "$CALLS" -eq 1 ]\n[ "$PASS_COUNT" -eq 3 ]\n[ "$FAIL_COUNT" -eq 0 ]\n'
                    path = directory/'run.sh'
                    path.write_text(script)
                    result = subprocess.run(['bash', str(path), str(directory), str(vault), fault, json.dumps(state), json.dumps(tx)], capture_output=True, text=True)
                    if fault == 'none':
                        self.assertEqual(result.returncode, 0, result.stdout+result.stderr)
                    else:
                        self.assertNotEqual(result.returncode, 0, result.stdout+result.stderr)


if __name__ == '__main__':
    unittest.main()
