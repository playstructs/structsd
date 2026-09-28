"""Paid guild setup must verify the founder debit and initial treasury assets."""
from pathlib import Path
import re
import subprocess
import unittest

SOURCE = (Path(__file__).parent / 'test_chain.sh').read_text()
HELPERS = '\n'.join(re.search(r'(?ms)^' + name + r'\(\) \{.*?^\}', SOURCE).group()
                    for name in ('assert_eq', 'create_paid_guild'))


class PaidGuildTests(unittest.TestCase):
    def test_creation_accounting_and_failure_propagation(self):
        for balance in (2000000, 2**63+1, 2**64-1):
            for fault in ('none', 'debit', 'token', 'collateral', 'guild_alpha', 'unprovisioned', 'owner',
                          'funding', 'snapshot', 'tx_error', 'underfunded', 'wrong_pool_owner', 'duplicate_pool', 'wrong_vault_owner', 'wrong_vault_address', 'string_provisioned', 'wrong_guild_id', 'vault_rpc', 'guild_rpc', 'founder_rpc', 'wrong_founder'):
                for conditional in (False, True):
                    with self.subTest(balance=balance, fault=fault, conditional=conditional):
                        script = r'''
set -euo pipefail
GREEN='' RED='' NC='' PASS_COUNT=0 FAIL_COUNT=0 CREATED=0
VAULT="$1" fault="$2" TOKEN=1000000 COLLATERAL=1000000 GUILD_ALPHA=0 PROVISIONED=true OWNER=1-1
fund_vault_alpha() { [ "$*" = '1-1 2000000' ]; [ "$fault" != funding ]; }
get_created_guild_id() { [ "$*" = '1-1 3-1 endpoint 4-1' ]; echo 0-1; }
get_vault_address() { case "$1" in 1-1) echo founder ;; 0-1) echo guild ;; *) return 9 ;; esac; }
get_balance() {
    case "$*" in
        'founder ualpha')
            if [ "$CREATED" -eq 0 ]; then
                [ "$fault" != snapshot ] || return 1
                if [ "$fault" = underfunded ]; then echo 1999999; return; fi
            fi
            echo "$VAULT" ;;
        'guild uguild.0-1') echo "$TOKEN" ;;
        'collateral ualpha') echo "$COLLATERAL" ;;
        'guild ualpha') echo "$GUILD_ALPHA" ;;
        *) return 9 ;;
    esac
}
run_tx() {
    shift
    [ "$*" = 'tx structs guild-create 3-1 endpoint 4-1 --from alice' ]
    echo CREATE_SENT
    [ "$fault" != tx_error ] || return 1
    CREATED=1
    VAULT=$(python3 -c 'import sys; print(int(sys.argv[1])-2000000)' "$VAULT")
    case "$fault" in
        debit) VAULT=$(python3 -c 'import sys; print(int(sys.argv[1])+1)' "$VAULT") ;;
        token) TOKEN=999999 ;;
        collateral) COLLATERAL=999999 ;;
        guild_alpha) GUILD_ALPHA=1 ;;
        unprovisioned) PROVISIONED=false ;;
        owner) OWNER=1-2 ;;
    esac
}
query() {
    [ "$CREATED" -eq 1 ] || return 9
    case "$fault:$*" in
        'founder_rpc:query structs player 1-1') echo '{"Player":{"id":"1-1","guildId":"0-1"}}'; return 1 ;;
        'wrong_founder:query structs player 1-1') echo '{"Player":{"id":"1-2","guildId":"0-1"}}'; return ;;
        'wrong_pool_owner:query structs guild-bank-collateral-address 0-1') echo '{"internalAddressAssociation":[{"objectId":"0-2","address":"collateral"}]}'; return ;;
        'duplicate_pool:query structs guild-bank-collateral-address 0-1') echo '{"internalAddressAssociation":[{"objectId":"0-1","address":"collateral"},{"objectId":"0-1","address":"collateral"}]}'; return ;;
        'wrong_vault_owner:query structs vault 0-1') echo '{"ownerId":"0-2","address":"guild","provisioned":true}'; return ;;
        'wrong_vault_address:query structs vault 0-1') echo '{"ownerId":"0-1","address":"wrong","provisioned":true}'; return ;;
        'string_provisioned:query structs vault 0-1') echo '{"ownerId":"0-1","address":"guild","provisioned":"true"}'; return ;;
        'wrong_guild_id:query structs guild 0-1') echo '{"Guild":{"id":"0-2","owner":"1-1"}}'; return ;;
        'vault_rpc:query structs vault 0-1') echo '{"ownerId":"0-1","address":"guild","provisioned":true}'; return 1 ;;
        'guild_rpc:query structs guild 0-1') echo '{"Guild":{"id":"0-1","owner":"1-1"}}'; return 1 ;;
    esac
    case "$*" in
        'query structs player 1-1') echo '{"Player":{"id":"1-1","guildId":"0-1"}}' ;;
        'query structs guild-bank-collateral-address 0-1') echo '{"internalAddressAssociation":[{"objectId":"0-1","address":"collateral"}]}' ;;
        'query structs vault 0-1') printf '{"ownerId":"0-1","address":"guild","provisioned":%s}' "$PROVISIONED" ;;
        'query structs guild 0-1') printf '{"Guild":{"id":"0-1","owner":"%s"}}' "$OWNER" ;;
        *) return 9 ;;
    esac
}
'''
                        script += HELPERS + '\n'
                        command = 'create_paid_guild alice 1-1 3-1 endpoint 4-1'
                        script += ('if ' + command + '; then :; else exit "$?"; fi\n') if conditional else command + '\n'
                        script += '[ "$FAIL_COUNT" -eq 0 ]\n[ "$CREATED" -eq 1 ]\n'
                        result = subprocess.run(['bash', '-c', script, 'test', str(balance), fault], capture_output=True, text=True)
                        if fault == 'none':
                            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                        else:
                            self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
                        if fault in ('funding', 'snapshot', 'underfunded'):
                            self.assertNotIn('CREATE_SENT', result.stdout)


if __name__ == '__main__':
    unittest.main()
