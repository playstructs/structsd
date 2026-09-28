"""Actual resize/extension call sites retain exact terms, funding and grid checks."""
from copy import deepcopy
import json
from pathlib import Path
import re
import subprocess
import tempfile
import unittest

from agreement_terms import update_terms
from test_chain_agreement_terms import HASH, INITIAL, transcript

ROOT = Path(__file__).resolve().parent
SOURCE = (ROOT/'test_chain.sh').read_text()
HELPERS = '\n'.join(re.search(r'(?ms)^'+name+r'\(\) \{.*?^\}', SOURCE).group()
                    for name in ('assert_eq', 'jqr', 'record_agreement_terms'))
BLOCK = SOURCE.split('# ─── agreement-capacity-increase ───', 1)[1].split('# ─── agreement-close ───', 1)[0]


class AgreementResizeTests(unittest.TestCase):
    def test_exact_capacity_power_load_duration_and_deposit(self):
        for base in (100, 2**53+1, 2**63-200):
            for stage, fault in ((1, 'none'), (1, 'capacity'), (2, 'capacity'), (1, 'power'),
                                 (2, 'power'), (1, 'load'), (2, 'load'), (3, 'duration'), (2, 'rpc'),
                                 (3, 'missing_deposit'), (3, 'undercredit'), (3, 'overcredit'),
                                 (3, 'capacity'), (3, 'power'), (3, 'load')):
                with self.subTest(base=base, stage=stage, fault=fault), tempfile.TemporaryDirectory() as directory:
                    directory = Path(directory)
                    for name in ('agreement_terms.py', 'economic_assertions.py'):
                        (directory/name).symlink_to(ROOT/name)
                    steps = transcript(base)
                    state = update_terms(INITIAL, *steps[0], HASH)
                    for index, (_, _, tx) in enumerate(steps[1:4], 1):
                        tx = deepcopy(tx)
                        if index == 3:
                            if fault == 'missing_deposit':
                                tx['events'].pop()
                            if fault in ('undercredit', 'overcredit'):
                                tx['events'][1]['attributes'][2]['value'] = ('1299999' if fault == 'undercredit' else '1300001')+'ualpha'
                        (directory/f'{index}.json').write_text(json.dumps(tx))
                    script = r'''set -euo pipefail
GREEN='' RED='' NC='' PASS_COUNT=0 FAIL_COUNT=0 CALLS=0
AGREE_ID=11-1 AGREE_ALLOC_ID=6-1 PROVIDER_ID=10-1 AGREE_CAPACITY=50000 AGREE_OPEN_COLLATERAL=2000000
AGREE_SERVICE_TERMS="$1" root="$2" stage="$3" fault="$4" BASE="$5"
CAPACITY=50000 POWER=50000 LOAD=50000 END=0
LAST_TX_OUTPUT='{"txhash":"''' + HASH + r'''"}'
_tx_json() { printf '%s' "$1"; }
info() { :; }
query() {
    if [ "$fault" = rpc ] && [ "$CALLS" -eq "$stage" ]; then return 9; fi
    case "$*" in
        'query tx ''' + HASH + r'''') cat "$root/$CALLS.json" ;;
        'query structs agreement 11-1') printf '{"Agreement":{"capacity":"%s","endBlock":"%s"}}' "$CAPACITY" "$END" ;;
        'query structs allocation 6-1') printf '{"gridAttributes":{"power":"%s"}}' "$POWER" ;;
        'query structs provider 10-1') printf '{"gridAttributes":{"load":"%s"}}' "$LOAD" ;;
        *) return 8 ;;
    esac
}
run_tx() {
    shift
    CALLS=$((CALLS+1))
    case "$CALLS" in
        1) [ "$*" = 'tx structs agreement-capacity-increase 11-1 25000 --from player_2' ]; CAPACITY=75000 POWER=75000 LOAD=75000; ADD=28 ;;
        2) [ "$*" = 'tx structs agreement-capacity-decrease 11-1 10000 --from player_2' ]; CAPACITY=65000 POWER=65000 LOAD=65000; ADD=30 ;;
        3) [ "$*" = 'tx structs agreement-duration-increase 11-1 20 --from player_2' ]; ADD=50 ;;
    esac
    END=$(python3 -c 'import sys; print(sum(map(int,sys.argv[1:])))' "$BASE" "$ADD")
    if [ "$CALLS" -eq "$stage" ]; then
        case "$fault" in
            capacity) CAPACITY=$((CAPACITY+1)) ;;
            power) POWER=$((POWER+1)) ;;
            load) LOAD=$((LOAD+1)) ;;
            duration) END=$(python3 -c 'import sys; print(int(sys.argv[1])-1)' "$END") ;;
        esac
    fi
}
'''
                    script += HELPERS+'\n'+BLOCK+'\n[ "$CALLS" -eq 3 ]\n[ "$PASS_COUNT" -eq 11 ]\n[ "$FAIL_COUNT" -eq 0 ]\n'
                    path = directory/'run.sh'
                    path.write_text(script)
                    result = subprocess.run(['bash', str(path), json.dumps(state), str(directory), str(stage), fault, str(base)], capture_output=True, text=True)
                    if fault == 'none':
                        self.assertEqual(result.returncode, 0, result.stdout+result.stderr)
                    else:
                        self.assertNotEqual(result.returncode, 0, result.stdout+result.stderr)


if __name__ == '__main__':
    unittest.main()
