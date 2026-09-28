"""Battle evidence must distinguish attacks, rejected deliveries and query failures."""
import copy
import json
from pathlib import Path
import re
import subprocess
import tempfile
import unittest

from battle_log import capture
from test_chain_pdc_killing_blow import delivered, HASH

ROOT = Path(__file__).parent.resolve()
SOURCE = (ROOT / 'test_chain.sh').read_text()


class BattleLogTests(unittest.TestCase):
    def test_only_successful_exact_typed_events_are_attacks(self):
        tx = delivered()
        before = copy.deepcopy(tx)
        self.assertEqual(capture(tx, HASH, 'shot', 'now')['type'], 'attack')
        self.assertEqual(tx, before)
        rejected = dict(tx, code=7, events=[], raw_log='out_of_range')
        row = capture(rejected, HASH, 'shot', 'now')
        self.assertEqual((row['type'], row['code'], row['raw_log']), ('attack_rejected', 7, 'out_of_range'))
        for bad in ({}, dict(tx, code=None), dict(tx, code=True), dict(tx, code='00'),
                    dict(tx, code=-1), dict(tx, code=2**32), dict(tx, height='0'),
                    dict(tx, txhash='B'*64), dict(tx, events=[]), dict(tx, events=tx['events']*2),
                    dict(tx, events=[{'type': 'someOtherEventAttack', 'attributes': []}]),
                    dict(tx, code=1), dict(tx, events=[{'type': 'structs.structs.EventAttack', 'attributes': []}])):
            with self.subTest(receipt=bad), self.assertRaises(ValueError):
                capture(bad, HASH, 'shot', 'now')

    def test_actual_shell_preserves_attempts_without_inflating_event_count(self):
        functions = '\n'.join(re.search(r'(?ms)^' + name + r'\(\) \{.*?^\}', SOURCE).group()
                              for name in ('_tx_json', '_log_battle_event'))
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for module in ROOT.glob('*.py'):
                (root / module.name).symlink_to(module)
            script = root / 'log.sh'
            script.write_text('''set -euo pipefail
BATTLE_LOG_FILE="$1" mode="$2" receipt="$3" output="$4"
CYAN='' RED='' NC='' BATTLE_LOG_COUNT=0 FAIL_COUNT=0
query() {
  [ "$*" = 'query tx ''' + HASH + '''' ] || return 9
  printf '%s' "$receipt"
  [ "$mode" != rpc ]
}
''' + functions + '''
_log_battle_event "$output" 'a test attack'
printf 'COUNTS=%s,%s\\n' "$BATTLE_LOG_COUNT" "$FAIL_COUNT"
''')
            for mode in ('success', 'rejected', 'rpc', 'hash', 'malformed', 'missing_event', 'wrong_hash'):
                tx = delivered()
                output = 'CLI notice\n' + json.dumps({'txhash': HASH}, indent=2)
                if mode == 'rejected':
                    tx.update(code=7, events=[], raw_log='out_of_range')
                elif mode == 'missing_event':
                    tx['events'] = []
                elif mode == 'wrong_hash':
                    tx['txhash'] = 'B'*64
                elif mode == 'hash':
                    output = 'broadcast failed'
                receipt = 'not JSON' if mode == 'malformed' else json.dumps(tx)
                log = root / (mode + '.jsonl')
                result = subprocess.run(['bash', str(script), str(log), mode, receipt, output], capture_output=True, text=True)
                with self.subTest(mode=mode):
                    self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                    rows = [json.loads(line) for line in log.read_text().splitlines()]
                    self.assertEqual(len(rows), 1)
                    expected = 'attack' if mode == 'success' else 'attack_rejected' if mode == 'rejected' else 'attack_unverified'
                    self.assertEqual(rows[0]['type'], expected)
                    self.assertIn('COUNTS=' + ('1,0' if mode == 'success' else '0,0' if mode == 'rejected' else '0,1'), result.stdout)
                    if expected == 'attack_unverified':
                        self.assertEqual(rows[0]['tx_output'], output)
                        self.assertNotIn('battle event #', result.stdout)


if __name__ == '__main__':
    unittest.main()
