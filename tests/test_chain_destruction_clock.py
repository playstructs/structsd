"""Destruction checks survive sweep timing and reject missing/failed evidence."""
import json
from pathlib import Path
import re
import subprocess
import unittest

SOURCE = Path(__file__).with_name('test_chain.sh').read_text()
HASH = 'A' * 64

class DestructionClockTests(unittest.TestCase):
    def test_receipt_height_survives_sweep_and_rejects_invalid_evidence(self):
        functions = '\n'.join(re.search(r'(?ms)^' + name + r'\(\) \{.*?^\}', SOURCE).group()
            for name in ('combat_state_at_height', 'struct_state_after_last_tx'))
        for fault in ('none', 'tx_rpc', 'state_rpc', 'failed_tx', 'wrong_hash', 'zero_height',
                      'missing_state', 'wrong_id', 'malformed_flag'):
            receipt = {'txhash': HASH, 'height': '42', 'code': 0}
            state = {'Struct': {'id': '5-2'}, 'structAttributes': {'isDestroyed': True}}
            if fault == 'failed_tx': receipt['code'] = 1
            if fault == 'wrong_hash': receipt['txhash'] = 'B' * 64
            if fault == 'zero_height': receipt['height'] = '0'
            if fault == 'missing_state': state = {}
            if fault == 'wrong_id': state['Struct']['id'] = '5-3'
            if fault == 'malformed_flag': state['structAttributes']['isDestroyed'] = 'true'
            script = '''set -euo pipefail
fault="$1" fixture_receipt="$2" fixture_state="$3" hash="$4"
LAST_TX_OUTPUT="{\\"txhash\\":\\"$hash\\"}"
_tx_json() { echo "$1"; }
query() {
 if [ "$*" = "query tx $hash" ]; then
  [ "$fault" != tx_rpc ] || return 1
  echo "$fixture_receipt"
 elif [ "$*" = 'query structs struct 5-2 --height 42' ]; then
  [ "$fault" != state_rpc ] || return 1
  echo "$fixture_state"
 else
  echo 'Latest object is already swept: wrong query' >&2
  return 9
 fi
}
'''
            result = subprocess.run(['bash','-c',script+functions+'\nstruct_state_after_last_tx 5-2',
                'test',fault,json.dumps(receipt),json.dumps(state),HASH],capture_output=True,text=True)
            with self.subTest(fault=fault):
                self.assertEqual(result.returncode == 0, fault == 'none',result.stdout+result.stderr)
                if fault == 'none': self.assertTrue(json.loads(result.stdout)['structAttributes']['isDestroyed'])
