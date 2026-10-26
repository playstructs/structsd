"""Independent window pricing and confirmed transaction evidence for phase 16."""
from copy import deepcopy
import json
from pathlib import Path
import re
import subprocess
import tempfile
import unittest

from agreement_terms import update_terms, final_balances

ROOT = Path(__file__).resolve().parent
SOURCE = (ROOT / 'test_chain.sh').read_text()
HELPER = re.search(r'(?ms)^record_agreement_terms\(\) \{.*?^\}', SOURCE).group()
HASH = 'A'*64
INITIAL = {'provider_id': '10-1', 'consumer_owner': '1-2', 'capacity': '50000',
           'consumer_vault': 'vault', 'collateral_address': 'pool'}


def transfer(amount):
    return {'type': 'transfer', 'attributes': [
        {'key': 'sender', 'value': 'vault'}, {'key': 'recipient', 'value': 'pool'},
        {'key': 'amount', 'value': str(amount)+'ualpha'}]}


def transaction(height, start=None, end=None, capacity=None, deposit=0):
    events = []
    if start is None:
        events = [{'type': 'structs.structs.EventDelete', 'attributes': [{'key': 'objectId', 'value': json.dumps('11-1')}]},
                  {'type': 'structs.structs.EventDelete', 'attributes': [{'key': 'objectId', 'value': json.dumps('6-1')}]}]
    else:
        row = {'id': '11-1', 'allocationId': '6-1', 'providerId': '10-1', 'owner': '1-2',
               'startBlock': str(start), 'endBlock': str(end), 'capacity': str(capacity)}
        events.append({'type': 'structs.structs.EventAgreement', 'attributes': [{'key': 'agreement', 'value': json.dumps(row)}]})
    if deposit:
        events.append(transfer(deposit))
    return {'code': 0, 'txhash': HASH, 'height': str(height), 'events': events}


def transcript(base=100):
    return [('open', '40', transaction(base, base, base+40, 50000, 2000000)),
            ('increase', '25000', transaction(base+5, base+5, base+28, 75000)),
            ('decrease', '10000', transaction(base+10, base+10, base+30, 65000)),
            ('extend', '20', transaction(base+12, base+10, base+50, 65000, 1300000)),
            ('close', '0', transaction(base+20))]


class ServiceTermsTests(unittest.TestCase):
    def test_independent_golden_windows_credit_and_final_balances(self):
        for base in (100, 2**53+1, 2**63-200):
            state = INITIAL
            for op, amount, tx in transcript(base):
                state = update_terms(state, op, amount, tx, HASH)
            self.assertEqual(state['rounding_dust'], '75000')
            self.assertEqual(state['principal'], '3300000')
            self.assertEqual(state['provider_expected'], '1275000')
            self.assertEqual(state['consumer_expected'], '1950000')
            self.assertEqual(final_balances(state, str(2**64), '17', '19'),
                             {'vault': str(2**64-1350000), 'collateral': '75017', 'earnings': '1275019'})
            with self.assertRaises(ValueError):
                update_terms(state, 'close', '0', transcript(base)[-1][2], HASH)

    def test_unknown_wrong_terms_and_funding_are_rejected(self):
        good = transcript()[0][2]
        invalid = [{}, dict(good, code=False), dict(good, code=2), dict(good, txhash='B'*64),
                   dict(good, height=str(2**63)), dict(good, events=[])]
        for key, value in [('providerId', '10-2'), ('owner', '1-3'), ('startBlock', '101'),
                           ('endBlock', '141'), ('capacity', '50001'), ('allocationId', '10-1')]:
            bad = deepcopy(good)
            row = json.loads(bad['events'][0]['attributes'][0]['value'])
            row[key] = value
            bad['events'][0]['attributes'][0]['value'] = json.dumps(row)
            invalid.append(bad)
        for index, value in [(0, 'other'), (1, 'other'), (2, '1999999ualpha'), (2, '2000000other')]:
            bad = deepcopy(good)
            bad['events'][1]['attributes'][index]['value'] = value
            invalid.append(bad)
        for duplicate in (0, 1):
            bad = deepcopy(good)
            bad['events'].append(deepcopy(bad['events'][duplicate]))
            invalid.append(bad)
        bad = deepcopy(good)
        bad['events'][1]['attributes'].append({'key': 'amount', 'value': '2000000ualpha'})
        invalid.append(bad)
        for tx in invalid:
            with self.subTest(tx=tx):
                with self.assertRaises((ValueError, KeyError, TypeError)):
                    update_terms(INITIAL, 'open', '40', tx, HASH)

    def test_boundary_changes_and_close_identity_are_rejected(self):
        state = update_terms(INITIAL, *transcript()[0], HASH)
        for op, amount, tx in [('increase', '0', transcript()[1][2]),
                               ('decrease', '50000', transcript()[2][2]),
                               ('increase', str(2**64), transcript()[1][2]),
                               ('extend', '0', transcript()[3][2]),
                               ('extend', '20', transaction(140, 100, 160, 50000, 1000000)),
                               ('close', '0', transaction(99)), ('close', '0', transaction(141)),
                               ('unknown', '0', transaction(120))]:
            with self.subTest(op=op, amount=amount):
                with self.assertRaises(ValueError):
                    update_terms(state, op, amount, tx, HASH)
        for bad_id in ('11-2', '6-1'):
            tx = transaction(120)
            tx['events'][0]['attributes'][0]['value'] = json.dumps(bad_id)
            with self.assertRaises(ValueError):
                update_terms(state, 'close', '0', tx, HASH)
        tx = transaction(120)
        tx['events'].append(deepcopy(tx['events'][0]))
        with self.assertRaises(ValueError):
            update_terms(state, 'close', '0', tx, HASH)
        with self.assertRaises(ValueError):
            final_balances(state, '4000000', '0', '0')

    def test_actual_shell_tracker_and_failure_in_conditional_callers(self):
        for conditional in (False, True):
            for fault in ('none', 'rpc', 'wrong_funding'):
                with tempfile.TemporaryDirectory() as directory:
                    directory = Path(directory)
                    for name in ('agreement_terms.py', 'economic_assertions.py'):
                        (directory/name).symlink_to(ROOT/name)
                    tx = transcript()[0][2]
                    if fault == 'wrong_funding':
                        tx['events'][1]['attributes'][2]['value'] = '1ualpha'
                    script = '''set -euo pipefail
state="$1" payload="$2" fault="$3"
LAST_TX_OUTPUT='{"txhash":"''' + HASH + '''"}'
_tx_json() { printf '%s' "$1"; }
query() { [ "$*" = 'query tx ''' + HASH + '''' ] || return 9; printf '%s' "$payload"; [ "$fault" != rpc ]; }
'''
                    script += HELPER + '\n'
                    call = 'record_agreement_terms open 40 "$state"'
                    script += ('if output=$(' + call + '); then printf "%s\\n" "$output"; else exit 7; fi\n') if conditional else call+'\n'
                    path = directory/'run.sh'
                    path.write_text(script)
                    result = subprocess.run(['bash', str(path), json.dumps(INITIAL), json.dumps(tx), fault], capture_output=True, text=True)
                    if fault == 'none':
                        self.assertEqual(result.returncode, 0, result.stderr)
                        self.assertEqual(json.loads(result.stdout)['principal'], '2000000')
                    else:
                        self.assertNotEqual(result.returncode, 0)
                        self.assertEqual(result.stdout, '')


if __name__ == '__main__':
    unittest.main()
