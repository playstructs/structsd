"""Vault funding must calculate exact shortfalls and fail closed in conditionals."""
from pathlib import Path
import re
import subprocess
import unittest

SOURCE = (Path(__file__).parent / 'test_chain.sh').read_text()
HELPER = re.search(r'(?ms)^fund_vault_alpha\(\) \{.*?^\}', SOURCE).group()
B_BLOCK = 'SETTLE_COLL_B=' + SOURCE.split('SETTLE_COLL_B=', 1)[1].split('run_tx "Player 3 opening settlement agreement B', 1)[0]


class VaultFundingTests(unittest.TestCase):
    def invoke(self, balance, minimum, fault='', conditional=False, command='fund_vault_alpha 1-3 "$MINIMUM"', faucet_state='bob'):
        script = r'''
set -euo pipefail
BALANCE="$1" MINIMUM="$2" fault="$3" BOB_ADDRESS="$4" PLAYER_3_ID=1-3 SETTLE_DUR_B=120 SETTLE_CAP_B=3000 PARAMS_KEYS=fixture-key-flags
if [ "$BOB_ADDRESS" = unset ]; then unset BOB_ADDRESS; fi
structsd() {
    [ "$*" = 'fixture-key-flags keys show bob' ] || return 9
    case "$fault" in
        key_rpc) return 1 ;;
        key_empty) echo '{"address":""}' ;;
        key_missing) echo '{}' ;;
        key_number) echo '{"address":17}' ;;
        *) echo '{"address":"bob"}' ;;
    esac
}
get_vault_address() { [ "$1" = 1-3 ]; [ "$fault" != vault_rpc ] || return 1; echo vault; }
get_balance() { [ "$*" = 'vault ualpha' ]; [ "$fault" != balance_rpc ] || return 1; echo "$BALANCE"; }
run_tx() {
    shift
    [ "$1 $2 $3 $4" = 'tx bank send bob' ] || return 9
    [ "$5" = vault ] && [ "$7 $8" = '--from bob' ] || return 9
    echo "SENT:$6"
    [ "$fault" != tx_error ]
}
'''
        script += HELPER + '\n'
        if conditional:
            script += 'if ' + command + '; then exit 0; else exit "$?"; fi\n'
        else:
            script += command + '\n'
        return subprocess.run(['bash', '-c', script, 'test', str(balance), str(minimum), fault, faucet_state], capture_output=True, text=True)

    def test_early_and_resumed_funding_resolves_its_own_faucet(self):
        for faucet_state in ('unset', 'stale-address'):
            for conditional in (False, True):
                result = self.invoke(0, 2000000, conditional=conditional, faucet_state=faucet_state)
                self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                self.assertEqual(result.stdout, 'SENT:2000000ualpha\n')

    def test_funded_vault_does_not_need_a_faucet_lookup(self):
        result = self.invoke(2000000, 2000000, fault='key_rpc', faucet_state='unset')
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertEqual(result.stdout, '')

    def test_exact_shortfalls_in_direct_and_conditional_calls(self):
        for balance, minimum in ((0, 0), (0, 460000), (459999, 460000), (460000, 460000),
                                 (2**63+1, 460000), (2**64-1, 2**64+7)):
            for conditional in (False, True):
                result = self.invoke(balance, minimum, conditional=conditional)
                self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                expected = f'SENT:{minimum-balance}ualpha\n' if minimum > balance else ''
                self.assertEqual(result.stdout, expected)

    def test_invalid_inputs_and_failures_propagate(self):
        for conditional in (False, True):
            for fault in ('vault_rpc', 'balance_rpc', 'tx_error', 'key_rpc', 'key_empty', 'key_missing', 'key_number'):
                result = self.invoke(0, 100, fault, conditional)
                self.assertNotEqual(result.returncode, 0)
                if fault != 'tx_error':
                    self.assertEqual(result.stdout, '')
            for bad in ('', '-1', '00', '1.5', 'false', ' 1'):
                for balance, minimum in ((bad, 100), (0, bad)):
                    result = self.invoke(balance, minimum, conditional=conditional)
                    self.assertNotEqual(result.returncode, 0)
                    self.assertEqual(result.stdout, '')

    def test_actual_agreement_b_funding_preserves_reserve(self):
        for balance in (0, 459999, 460000, 2**64-1):
            result = self.invoke(balance, 0, command=B_BLOCK)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
            self.assertEqual(result.stdout, f'SENT:{460000-balance}ualpha\n' if balance < 460000 else '')


if __name__ == '__main__':
    unittest.main()
