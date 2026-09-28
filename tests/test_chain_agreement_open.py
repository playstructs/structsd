"""The open phase validates the exact confirmed deposit and installed load."""
import json
from pathlib import Path
import re
import subprocess
import tempfile
import unittest

from test_chain_agreement_terms import HASH, transcript

ROOT = Path(__file__).resolve().parent
SOURCE = (ROOT/'test_chain.sh').read_text()
HELPERS = '\n'.join(re.search(r'(?ms)^'+name+r'\(\) \{.*?^\}', SOURCE).group()
                    for name in ('assert_eq', 'record_agreement_terms'))
BLOCK = SOURCE.split('# ─── agreement-open collateral accounting ───', 1)[1].split('# ─── agreement-open collateral accounting complete ───', 1)[0]


class AgreementOpenPhaseTests(unittest.TestCase):
    def test_exact_confirmed_collateral_and_load(self):
        for fault in ('none', 'omitted_zero', 'missing_deposit', 'wrong_sender', 'wrong_recipient',
                      'undercredit', 'overcredit', 'wrong_load', 'prior_load', 'missing_grid', 'false_load', 'rpc_failure'):
            with self.subTest(fault=fault), tempfile.TemporaryDirectory() as directory:
                directory = Path(directory)
                for name in ('agreement_terms.py', 'economic_assertions.py'):
                    (directory/name).symlink_to(ROOT/name)
                tx = transcript()[0][2]
                if fault == 'missing_deposit':
                    tx['events'].pop()
                else:
                    for tag, index, value in [('wrong_sender', 0, 'other'), ('wrong_recipient', 1, 'other'),
                                               ('undercredit', 2, '1999999ualpha'), ('overcredit', 2, '2000001ualpha')]:
                        if fault == tag:
                            tx['events'][1]['attributes'][index]['value'] = value
                script = r'''set -euo pipefail
GREEN='' RED='' NC='' PASS_COUNT=0 FAIL_COUNT=0 CALLS=0
PROVIDER_ID=10-1 PLAYER_2_ID=1-2 PLAYER_2_VAULT=vault AGREE_DURATION=40 AGREE_CAPACITY=50000
VAULT=4000000 POOL=17 LOAD=0 fault="$1" TX_PAYLOAD="$2"
LAST_TX_OUTPUT='{"txhash":"''' + HASH + r'''"}'
_tx_json() { printf '%s' "$1"; }
if [ "$fault" = prior_load ]; then LOAD=1; fi
query() {
    case "$*" in
        'query structs provider-collateral-address 10-1') echo '{"internalAddressAssociation":[{"address":"pool"}]}' ;;
        'query structs provider-earnings-address 10-1') echo '{"internalAddressAssociation":[{"address":"earnings","objectId":"10-1"}]}' ;;
        'query tx ''' + HASH + r'''') printf '%s' "$TX_PAYLOAD" ;;
        'query structs provider 10-1')
            case "$fault" in
                rpc_failure) return 9 ;;
                missing_grid) echo '{}'; return ;;
                false_load) echo '{"gridAttributes":{"load":false}}'; return ;;
                omitted_zero) if [ "$LOAD" = 0 ]; then echo '{"gridAttributes":{}}'; return; fi ;;
            esac
            printf '{"gridAttributes":{"load":"%s"}}' "$LOAD" ;;
        *) return 8 ;;
    esac
}
get_balance() {
    [ "$2" = ualpha ] || return 9
    case "$1" in vault) echo "$VAULT" ;; pool) echo "$POOL" ;; earnings) echo 19 ;; *) return 8 ;; esac
}
run_tx() {
    shift
    [ "$*" = 'tx structs agreement-open 10-1 40 50000 --from player_2' ] || return 9
    CALLS=$((CALLS+1))
    LOAD=50000
    if [ "$fault" = wrong_load ]; then LOAD=50001; fi
}
'''
                script += HELPERS+'\n'+BLOCK+'\n[ "$CALLS" -eq 1 ]\n[ "$PASS_COUNT" -eq 3 ]\n[ "$FAIL_COUNT" -eq 0 ]\n'
                path = directory/'run.sh'
                path.write_text(script)
                result = subprocess.run(['bash', str(path), fault, json.dumps(tx)], capture_output=True, text=True)
                if fault in ('none', 'omitted_zero'):
                    self.assertEqual(result.returncode, 0, result.stdout+result.stderr)
                else:
                    self.assertNotEqual(result.returncode, 0, result.stdout+result.stderr)


if __name__ == '__main__':
    unittest.main()
