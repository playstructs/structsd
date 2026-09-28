"""Adversarial checks for the actual phase-15 EndBlock cleanup gate."""
import copy
import json
from pathlib import Path
import re
import subprocess
import unittest

from generator_destruction import verify
from test_chain_generator_attack import HASH, delivered

ROOT = Path(__file__).parent
SOURCE = (ROOT / 'test_chain.sh').read_text()


def event(kind, name, value, mode='EndBlock'):
    return {'type': 'structs.structs.' + kind, 'attributes': [
        {'key': name, 'value': json.dumps(value)}, {'key': 'mode', 'value': mode}]}


def fixture():
    return {'receipt': delivered(), 'block': {'height': '42', 'finalize_block_events': [
        event('EventDelete', 'objectId', '5-2-vault4'),
        *[event('EventGrid', 'gridRecord', {'attributeId': prefix + '-5-2', 'value': '0'})
          for prefix in ('1', '2', '3', '5')],
        event('EventGrid', 'gridRecord', {'attributeId': '2-1-4', 'value': '4800000'})]}}


def check(document):
    return verify(document, HASH, '5-1', '5-2', '1-4', 'vault4', '4800000')


class GeneratorDestructionTests(unittest.TestCase):
    def test_requires_lethal_receipt_and_endblock_effects(self):
        for height in ('42', 42):
            doc = fixture();doc['block']['height'] = height
            self.assertEqual(check(doc), '42')
        doc = fixture()
        doc['block']['finalize_block_events'] += [
            event('EventDelete', 'objectId', '5-99', 'BeginBlock'),
            event('EventGrid', 'gridRecord', {'attributeId': '2-1-7', 'value': '7'}, 'BeginBlock')]
        self.assertEqual(check(doc), '42')
        for tx in ({}, delivered('2'), dict(delivered(), code=1), dict(delivered(), txhash='B'*64)):
            with self.subTest(tx=tx), self.assertRaises((ValueError, KeyError)):
                check(dict(fixture(), receipt=tx))
        for block in ({}, {'height': '41', 'finalize_block_events': fixture()['block']['finalize_block_events']},
                      {'height': True, 'finalize_block_events': []}, {'height': '42', 'finalize_block_events': None}):
            with self.subTest(block=block), self.assertRaises(ValueError):
                check(dict(fixture(), block=block))

    def test_deletion_and_every_grid_effect_are_mandatory_and_exact(self):
        for index in range(6):
            for fault in ('missing', 'duplicate', 'wrong-value', 'wrong-mode', 'duplicate-mode', 'missing-mode', 'duplicate-field'):
                doc = fixture();events = doc['block']['finalize_block_events'];row = events[index]
                if fault == 'missing': events.pop(index)
                elif fault == 'duplicate': events.append(copy.deepcopy(row))
                elif fault == 'wrong-mode': row['attributes'][1]['value'] = 'BeginBlock'
                elif fault == 'duplicate-mode': row['attributes'].append(copy.deepcopy(row['attributes'][1]))
                elif fault == 'missing-mode': row['attributes'].pop(1)
                elif fault == 'duplicate-field': row['attributes'].append(copy.deepcopy(row['attributes'][0]))
                elif index == 0: row['attributes'][0]['value'] = json.dumps('5-2-other-vault')
                else:
                    value = json.loads(row['attributes'][0]['value']);value['value'] = str(int(value['value'])+1)
                    row['attributes'][0]['value'] = json.dumps(value)
                with self.subTest(index=index, fault=fault), self.assertRaises(ValueError): check(doc)
        for height in (True, False, 42.0, None, 0, -1, 2**63, '', '+42', '042', '42.0', str(2**63)):
            doc = fixture();doc['block']['height'] = height
            with self.subTest(height=height), self.assertRaises(ValueError): check(doc)
        for extra in (event('EventInfusion', 'infusion', {'destinationId': '5-2'}),
                      event('EventDelete', 'objectId', '5-2-other-vault'),
                      event('EventDelete', 'objectId', '5-2-vault4', 'BeginBlock')):
            doc = fixture();doc['block']['finalize_block_events'].append(extra)
            with self.assertRaises(ValueError): check(doc)

    def test_shell_fetches_exact_lethal_height_and_refuses_partial_evidence(self):
        function = re.search(r'(?ms)^generator_destruction_height\(\) \{.*?^\}', SOURCE).group()
        for fault in ('none', 'cli-height', 'receipt-rpc', 'block-rpc', 'wrong-height', 'missing-delete', 'failed-tx'):
            doc = fixture()
            if fault == 'cli-height': doc['block']['height'] = 42
            if fault == 'wrong-height': doc['block']['height'] = '41'
            if fault == 'missing-delete': doc['block']['finalize_block_events'].pop(0)
            if fault == 'failed-tx': doc['receipt']['code'] = 1
            script = '''set -euo pipefail
fault="$1" RECEIPT_RESPONSE="$2" BLOCK_RESPONSE="$3" GENERATOR_TEST_DIR="$4" HASH="$5"
DESTROYER_STRUCT_ID=5-1 GENERATOR_STRUCT_ID=5-2 PLAYER_4_ID=1-4 PLAYER_4_VAULT=vault4 P4_CAP_EXPECTED_DESTROYED=4800000
LAST_TX_OUTPUT=$(printf '{"txhash":"%s"}' "$HASH")
_tx_json() { printf '%s' "$1"; }
query() {
 case "$*" in
 "query tx $HASH") [ "$fault" != receipt-rpc ]; printf '%s' "$RECEIPT_RESPONSE";;
 'query block-results 42') [ "$fault" != block-rpc ]; printf '%s' "$BLOCK_RESPONSE";;
 *) return 9;;
 esac
}
'''
            # Explicit returns matter: the function is called through || by the shell gate.
            script = script.replace('[ "$fault" != receipt-rpc ];', '[ "$fault" != receipt-rpc ] || return 7;')
            script = script.replace('[ "$fault" != block-rpc ];', '[ "$fault" != block-rpc ] || return 8;')
            result = subprocess.run(['bash', '-c', script+function+'\ngenerator_destruction_height\n', 'test',
                fault, json.dumps(doc['receipt']), json.dumps(doc['block']), str(ROOT.resolve()), HASH], capture_output=True, text=True)
            self.assertEqual(result.returncode == 0, fault in ('none', 'cli-height'), result.stdout+result.stderr)
            if fault in ('none', 'cli-height'): self.assertEqual(result.stdout.strip(), '42')
        self.assertIn('GEN_DESTROY_HEIGHT=$(generator_destruction_height)', SOURCE)
        self.assertIn('P4_JSON=$(query query structs player "${PLAYER_4_ID}" --height "${GEN_DESTROY_HEIGHT}")', SOURCE)

    def test_shell_refuses_destruction_refunds_or_a_second_debit(self):
        assertions = '\n'.join(line for line in SOURCE.splitlines()
                               if line.startswith('assert_eq "Generator destruction preserves'))
        self.assertEqual(len(assertions.splitlines()), 2)
        assertion = re.search(r'(?ms)^assert_eq\(\) \{.*?^\}', SOURCE).group()
        for delta in (-1, 0, 1):
            for account in ('vault', 'key'):
                script = '''set -euo pipefail
GREEN='' RED='' NC='' PASS_COUNT=0 FAIL_COUNT=0
GEN_DESTROY_HEIGHT=42 PLAYER_4_VAULT=vault PLAYER_4_ADDRESS=key
P4_VAULT_EXPECTED=18446744073709551617 P4_SIGNING_BEFORE_GENERATOR=5000000
VALUE="$1" ACCOUNT="$2"
get_balance_at_height() {
 [ "$2:$3" = ualpha:42 ] || return 9
 if [ "$1" = "$ACCOUNT" ]; then echo "$VALUE";
 elif [ "$1" = vault ]; then echo "$P4_VAULT_EXPECTED";
 elif [ "$1" = key ]; then echo "$P4_SIGNING_BEFORE_GENERATOR";
 else return 8; fi
}
'''
                before = 18446744073709551617 if account == 'vault' else 5000000
                result = subprocess.run(['bash', '-c', script+assertion+'\n'+assertions+'\n[ "$FAIL_COUNT" -eq 0 ]',
                    'test', str(before+delta), account], capture_output=True, text=True)
                self.assertEqual(result.returncode == 0, delta == 0, result.stdout+result.stderr)


if __name__ == '__main__':
    unittest.main()
