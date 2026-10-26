"""Closing one consumer must preserve the other consumer's record and allocation."""
import copy
import json
from pathlib import Path
import re
import subprocess
import unittest

SOURCE = (Path(__file__).parent / 'test_chain.sh').read_text()
HELPERS = '\n'.join(re.search(r'(?ms)^' + name + r'\(\) \{.*?^\}', SOURCE).group()
                    for name in ('assert_eq', 'assert_object_missing'))
BLOCK = 'SETTLE_B_JSON=' + SOURCE.split('SETTLE_B_JSON=', 1)[1].split('# The refund uses the delivered close block', 1)[0]
AGREEMENT = {'Agreement': {'id': '11-2', 'allocationId': '6-2', 'providerId': '10-1', 'owner': '1-3',
                           'capacity': '3000', 'startBlock': '100', 'endBlock': '220'}}
ALLOCATION = {'Allocation': {'id': '6-2', 'sourceObjectId': '10-1', 'destinationId': '1-3',
                             'controller': '1-3', 'creator': 'owner', 'index': '9007199254740993'},
              'gridAttributes': {'power': '3000'}}


class SettlementIsolationTests(unittest.TestCase):
    def test_actual_close_preserves_other_consumer(self):
        cases = [('none', copy.deepcopy(AGREEMENT), copy.deepcopy(ALLOCATION), '')]
        for field in AGREEMENT['Agreement']:
            changed = copy.deepcopy(AGREEMENT)
            changed['Agreement'][field] = 'corrupted'
            cases.append(('agreement_' + field, changed, ALLOCATION, ''))
        for field in ALLOCATION['Allocation']:
            changed = copy.deepcopy(ALLOCATION)
            changed['Allocation'][field] = 'corrupted'
            cases.append(('allocation_' + field, AGREEMENT, changed, ''))
        for grid in ({'power': '2999'}, None):
            changed = copy.deepcopy(ALLOCATION)
            changed['gridAttributes'] = grid
            cases.append(('grid', AGREEMENT, changed, ''))
        for fault in ('pre_agreement_rpc', 'pre_allocation_rpc', 'post_agreement_rpc', 'post_allocation_rpc'):
            cases.append((fault, AGREEMENT, ALLOCATION, fault))
        cases.extend([('missing_agreement', {}, ALLOCATION, ''), ('missing_allocation', AGREEMENT, {}, '')])
        for name, after_agreement, after_allocation, fault in cases:
            with self.subTest(name=name):
                script = r'''
set -euo pipefail
GREEN='' RED='' NC='' PASS_COUNT=0 FAIL_COUNT=0 PARAMS_QUERY='' CLOSED=0
SETTLE_AGREE_A=11-1 SETTLE_AGREE_B=11-2 SETTLE_ALLOC_A=6-1 SETTLE_ALLOC_B=6-2 PLAYER_2_VAULT=vault
BEFORE_AGREEMENT="$1" BEFORE_ALLOCATION="$2" AFTER_AGREEMENT="$3" AFTER_ALLOCATION="$4" fault="$5"
get_balance() { [ "$*" = 'vault ualpha' ]; echo 100; }
run_tx() { shift; [ "$*" = 'tx structs agreement-close 11-1 --from player_2' ]; CLOSED=1; echo CLOSE_SENT; }
structsd() {
    case "$*" in 'query structs agreement 11-1'|'query structs allocation 6-1') ;;
        *) return 9 ;; esac
    echo 'Error: object not found' >&2; return 1
}
query() {
    local phase=pre
    if [ "$CLOSED" -eq 1 ]; then phase=post; fi
    case "$*" in
        'query structs agreement 11-1') echo '{"Agreement":{"endBlock":"220"}}' ;;
        'query structs agreement 11-2')
            [ "$fault" != "${phase}_agreement_rpc" ] || return 1
            if [ "$CLOSED" -eq 0 ]; then echo "$BEFORE_AGREEMENT"; else echo "$AFTER_AGREEMENT"; fi ;;
        'query structs allocation 6-2')
            [ "$fault" != "${phase}_allocation_rpc" ] || return 1
            if [ "$CLOSED" -eq 0 ]; then echo "$BEFORE_ALLOCATION"; else echo "$AFTER_ALLOCATION"; fi ;;
        *) return 9 ;;
    esac
}
'''
                script += HELPERS + '\n' + BLOCK + '\n[ "$CLOSED" -eq 1 ]\n[ "$FAIL_COUNT" -eq 0 ]\n'
                result = subprocess.run(['bash', '-c', script, 'test', json.dumps(AGREEMENT), json.dumps(ALLOCATION),
                                         json.dumps(after_agreement), json.dumps(after_allocation), fault], capture_output=True, text=True)
                if name == 'none':
                    self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                else:
                    self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
                if fault.startswith('pre_'):
                    self.assertNotIn('CLOSE_SENT', result.stdout)


if __name__ == '__main__':
    unittest.main()
