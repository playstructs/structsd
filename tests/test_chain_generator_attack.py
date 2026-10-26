"""Exercise confirmed generator hits and the script's historical state query."""
import copy
import json
from pathlib import Path
import re
import subprocess
import unittest

from generator_attack import verify

ROOT = Path(__file__).parent
SOURCE = (ROOT / 'test_chain.sh').read_text()
HASH = 'A' * 64


def delivered(health='1'):
    shot = {'targetStructId': '5-2', 'targetStructTypeId': '20', 'targetHealthBefore': health,
            'targetHealthAfter': str(int(health) - 1), 'targetDestroyed': health == '1',
            'damageDealt': '2', 'damageReduction': '1', 'damage': '1', 'blocked': False, 'evaded': False}
    attack = {'attackerStructId': '5-1', 'attackerStructTypeId': '9', 'weaponSystem': 'primaryWeapon',
              'eventAttackShotDetail': [shot]}
    return {'txhash': HASH, 'height': '42', 'code': 0, 'events': [{'type': 'structs.structs.EventAttack',
        'attributes': [{'key': 'eventAttackDetail', 'value': json.dumps(attack)}]}]}


class GeneratorAttackTests(unittest.TestCase):
    def test_every_armoured_hit_and_killing_blow(self):
        for health in range(1, 9):
            self.assertEqual(verify(delivered(str(health)), HASH, '5-1', '5-2', str(health)),
                {'height': '42', 'health': str(health - 1), 'destroyed': health == 1})

    def test_receipts_cannot_invent_destruction_or_skip_damage(self):
        for field, value in [('targetHealthBefore', '2'), ('targetHealthAfter', '1'), ('targetDestroyed', False),
                ('targetDestroyed', 1), ('targetStructId', '5-3'), ('targetStructTypeId', '19'),
                ('damageDealt', '1'), ('damageReduction', '0'), ('damage', '2'), ('blocked', True), ('evaded', True)]:
            tx = delivered()
            attack = json.loads(tx['events'][0]['attributes'][0]['value'])
            attack['eventAttackShotDetail'][0][field] = value
            tx['events'][0]['attributes'][0]['value'] = json.dumps(attack)
            with self.subTest(field=field), self.assertRaises(ValueError):
                verify(tx, HASH, '5-1', '5-2', '1')
        for tx in ({}, dict(delivered(), code=1), dict(delivered(), height='0'), dict(delivered(), txhash='B'*64),
                dict(delivered(), events=[]), dict(delivered(), events=delivered()['events']*2)):
            with self.subTest(tx=tx), self.assertRaises(ValueError):
                verify(tx, HASH, '5-1', '5-2', '1')

    def test_shell_uses_confirmed_height_and_refuses_unavailable_or_inconsistent_state(self):
        function = re.search(r'(?ms)^generator_state_after_attack\(\) \{.*?^\}', SOURCE).group()
        for health in ('1', '8'):
            base = {'Struct': {'id': '5-2'}, 'structAttributes': {}, 'gridAttributes': {}}
            if health == '1':
                base['structAttributes']['isDestroyed'] = True
            else:
                base['structAttributes']['health'] = '7'
            for fault in ('none', 'tx_rpc', 'state_rpc', 'wrong_id', 'missing', 'attributes_missing',
                    'grid_missing', 'health_null', 'health_wrong', 'flag_wrong', 'failed_tx'):
                state = copy.deepcopy(base)
                tx = delivered(health)
                if fault == 'wrong_id': state['Struct']['id'] = '5-3'
                if fault == 'missing': state = {}
                if fault == 'attributes_missing': del state['structAttributes']
                if fault == 'grid_missing': del state['gridAttributes']
                if fault == 'health_null': state['structAttributes']['health'] = None
                if fault == 'health_wrong': state['structAttributes']['health'] = '4'
                if fault == 'flag_wrong': state['structAttributes']['isDestroyed'] = health != '1'
                if fault == 'failed_tx': tx['code'] = 1
                script = '''set -euo pipefail
GENERATOR_TEST_DIR="$1" fault="$2" fixture_receipt="$3" fixture_state="$4" hash="$5" health="$6"
DESTROYER_STRUCT_ID=5-1 GENERATOR_STRUCT_ID=5-2
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
    # The latest object has already been swept: never query it as evidence.
    echo 'unexpected or latest-state query' >&2
    return 9
  fi
}
'''
                result = subprocess.run(['bash', '-c', script + function + '\ngenerator_state_after_attack "$health"',
                    'test', str(ROOT.resolve()), fault, json.dumps(tx), json.dumps(state), HASH, health],
                    capture_output=True, text=True)
                with self.subTest(health=health, fault=fault):
                    self.assertEqual(result.returncode == 0, fault == 'none', result.stdout + result.stderr)
                    if fault == 'none': self.assertEqual(json.loads(result.stdout), state)

    def test_destruction_loop_preserves_last_confirmed_state(self):
        block = SOURCE.split('# Record generator health before attacks', 1)[1].split('# ─── Verify Player 4', 1)[0]
        self.assertIn('GEN_JSON=$(generator_state_after_attack "${GEN_HP_BEFORE}")', block)
        self.assertIn('GEN_JSON=$(generator_state_after_attack "${GEN_HP_NOW}")', block)
        self.assertNotIn('query query structs struct', block.split('run_tx "Attack round 1', 1)[1])


if __name__ == '__main__':
    unittest.main()
