"""Lazy withdrawal/deletion evidence and exact post-deletion accounting."""
from copy import deepcopy
import json
from pathlib import Path
import re
import subprocess
import tempfile
import unittest

from provider_lifecycle_accounting import record_step, verify_final

ROOT = Path(__file__).resolve().parent
SOURCE = (ROOT / 'test_chain.sh').read_text()
TRACKER = re.search(r'(?ms)^record_provider_lifecycle_step\(\) \{.*?^\}', SOURCE).group()
HASH = 'A'*64


def event(kind, name, value):
    return {'type': 'structs.structs.' + kind, 'attributes': [{'key': name, 'value': json.dumps(value)}]}


def transfer(sender, recipient, amount):
    return {'type': 'transfer', 'attributes': [{'key': key, 'value': value} for key, value in
            [('sender', sender), ('recipient', recipient), ('amount', str(amount)+'ualpha')]]}


def receipt(height, events):
    return {'code': 0, 'height': str(height), 'txhash': HASH, 'events': events}


def fixture(base=1, initial_balance=10000, capacities=(50, 70), duration=40):
    initial = {'provider_id': '10-1', 'provider_owner': '0-1', 'source_id': '4-1', 'owner_vault': 'guild',
               'personal_vault': 'personal', 'collateral_address': 'collateral', 'earnings_address': 'earnings',
               'initial_owner_balance': str(initial_balance), 'initial_personal_balance': str(initial_balance), 'agreements': []}
    steps = []
    for i, capacity in enumerate(capacities):
        start, principal = base+i, capacity*duration
        agreement, allocation, owner, vault = f'11-{i+1}', f'6-{i+1}', f'1-{i+2}', f'consumer{i}'
        argument = {'consumer_owner': owner, 'consumer_vault': vault, 'capacity': str(capacity),
                    'duration': str(duration), 'initial_balance': str(initial_balance)}
        terms = {'id': agreement, 'allocationId': allocation, 'providerId': '10-1', 'owner': owner,
                 'capacity': str(capacity), 'startBlock': str(start), 'endBlock': str(start+duration)}
        events = [event('EventAgreement', 'agreement', terms), transfer(vault, 'collateral', principal)]
        if i:
            events.append(transfer('collateral', 'earnings', capacities[0]//2))
        steps.append({'operation': 'open', 'argument': argument, 'tx': receipt(start, events)})
    checkpoint_earned = 8*sum(capacities)//2
    earned_before = capacities[0]//2+checkpoint_earned
    steps.append({'operation': 'withdraw', 'argument': {}, 'tx': receipt(base+9, [
        transfer('collateral', 'earnings', checkpoint_earned), transfer('earnings', 'guild', earned_before)])})
    deleted = [event('EventDelete', 'objectId', key) for key in ('10-1', '11-1', '6-1', '11-2', '6-2')]
    balances, provider_total = {'personal': str(initial_balance), 'collateral': '0', 'earnings': '0'}, 0
    for i, capacity in enumerate(capacities):
        elapsed = 11-i
        refund = (duration-elapsed)*capacity+elapsed*capacity//2
        earned = duration*capacity-refund
        deleted.append(transfer('collateral', f'consumer{i}', refund))
        balances[f'consumer{i}'] = str(initial_balance-earned)
        provider_total += earned
    remaining = provider_total-earned_before
    deleted.extend([transfer('collateral', 'earnings', remaining), transfer('earnings', 'guild', remaining)])
    steps.append({'operation': 'delete', 'argument': {}, 'tx': receipt(base+11, deleted)})
    balances['guild'] = str(initial_balance+provider_total)
    return {'initial': initial, 'steps': steps, 'balances': balances}


def replay(f, stop=4):
    state = f['initial']
    for step in f['steps'][:stop]:
        state = record_step(state, step['operation'], step['argument'], step['tx'], HASH)
    return state


def replace_record(tx, kind, key, value, index=0):
    match = [e for e in tx['events'] if e['type'] == 'structs.structs.'+kind][index]
    row = json.loads(match['attributes'][0]['value'])
    row[key] = value
    match['attributes'][0]['value'] = json.dumps(row)


class ProviderLifecycleTests(unittest.TestCase):
    def test_exact_money_with_large_heights_balances_and_rounding(self):
        for base, balance in ((1, 10000), (2**53+1, 2**128), (2**63-100, 2**128)):
            for capacities, total in (((50, 70), '625'), ((51, 71), '636')):
                with self.subTest(base=base, capacities=capacities):
                    f = fixture(base, balance, capacities)
                    self.assertEqual(verify_final(replay(f), f['balances']), {'verified': True, 'provider_total': total})
                    self.assertEqual(f['initial']['agreements'], [])

    def test_bad_confirmation_or_event_shapes_fail_closed(self):
        f = fixture()
        tx = f['steps'][0]['tx']
        bad = [None, [], {}, dict(tx, code=False), dict(tx, code=1), dict(tx, txhash='B'*64),
               dict(tx, height='01'), dict(tx, height='0'), dict(tx, height=str(2**63)),
               dict(tx, events=None), dict(tx, events=[None]), dict(tx, events=[{'type': 'transfer', 'attributes': [None]}])]
        for payload in bad:
            with self.subTest(payload=payload), self.assertRaises((ValueError, KeyError, TypeError)):
                record_step(f['initial'], 'open', f['steps'][0]['argument'], payload, HASH)

    def test_open_requires_exact_terms_and_funding(self):
        f = fixture()
        for key, value in [('capacity', '51'), ('owner', '1-9'), ('startBlock', '0'), ('endBlock', '40'), ('providerId', '10-9')]:
            tx = deepcopy(f['steps'][0]['tx'])
            replace_record(tx, 'EventAgreement', key, value)
            with self.subTest(key=key), self.assertRaises(ValueError):
                record_step(f['initial'], 'open', f['steps'][0]['argument'], tx, HASH)
        for mode in ('missing', 'duplicate', 'bad_amount'):
            tx = deepcopy(f['steps'][0]['tx'])
            if mode == 'missing': tx['events'].pop()
            elif mode == 'duplicate': tx['events'].append(deepcopy(tx['events'][-1]))
            else: tx['events'][1]['attributes'][2]['value'] = '1999ualpha'
            with self.subTest(mode=mode), self.assertRaises(ValueError):
                record_step(f['initial'], 'open', f['steps'][0]['argument'], tx, HASH)

    def test_each_checkpoint_refund_and_drain_requires_exact_transfers(self):
        f = fixture()
        for step_index in (1, 2, 3):
            step = f['steps'][step_index]
            prior = replay(f, step_index)
            for index, e in enumerate(step['tx']['events']):
                for fault in ('missing', 'duplicate', 'wrong_recipient', 'one_coin'):
                    if e['type'] != 'transfer' and fault in ('wrong_recipient', 'one_coin'): continue
                    tx = deepcopy(step['tx'])
                    if fault == 'missing': tx['events'].pop(index)
                    elif fault == 'duplicate': tx['events'].append(deepcopy(e))
                    elif fault == 'wrong_recipient': tx['events'][index]['attributes'][1]['value'] = 'personal'
                    else: tx['events'][index]['attributes'][2]['value'] = str(int(e['attributes'][2]['value'][:-6])+1)+'ualpha'
                    with self.subTest(step=step_index, index=index, fault=fault), self.assertRaises(ValueError):
                        record_step(prior, step['operation'], step['argument'], tx, HASH)

    def test_fixture_order_and_expiry_are_checked(self):
        f = fixture()
        for count, op, index in [(0, 'withdraw', 2), (1, 'withdraw', 2), (2, 'delete', 3), (3, 'withdraw', 2), (4, 'delete', 3)]:
            with self.subTest(count=count, op=op), self.assertRaises(ValueError):
                record_step(replay(f, count), op, {}, f['steps'][index]['tx'], HASH)
        for height in (1, 2, 41, 42):
            tx = dict(f['steps'][2]['tx'], height=str(height))
            with self.subTest(height=height), self.assertRaises(ValueError):
                record_step(replay(f, 2), 'withdraw', {}, tx, HASH)

    def test_final_one_coin_errors_in_every_account_refuse(self):
        f = fixture()
        state = replay(f)
        for address in f['balances']:
            balances = dict(f['balances'])
            balances[address] = str(int(balances[address])+1)
            with self.subTest(address=address), self.assertRaises(ValueError):
                verify_final(state, balances)
        with self.assertRaises(ValueError):
            verify_final(state, dict(f['balances'], extra='0'))

    def test_actual_shell_tracker_propagates_failure_in_conditional_callers(self):
        f = fixture()
        for conditional in (False, True):
            for fault in ('none', 'rpc', 'failed_tx', 'bad_hash', 'missing_event'):
                with self.subTest(conditional=conditional, fault=fault), tempfile.TemporaryDirectory() as directory:
                    directory = Path(directory)
                    for name in ('provider_lifecycle_accounting.py', 'agreement_terms.py', 'economic_assertions.py'):
                        (directory/name).symlink_to(ROOT/name)
                    tx = deepcopy(f['steps'][0]['tx'])
                    if fault == 'failed_tx':
                        tx['code'] = 5
                    if fault == 'missing_event':
                        tx['events'].pop()
                    script = '''set -euo pipefail
state="$1" argument="$2" payload="$3" fault="$4"
LAST_TX_OUTPUT='{"txhash":"''' + HASH + '''"}'
if [ "$fault" = bad_hash ]; then LAST_TX_OUTPUT='{"txhash":"bad"}'; fi
_tx_json() { printf '%s' "$1"; }
query() { [ "$*" = 'query tx ''' + HASH + '''' ] || return 9; printf '%s' "$payload"; [ "$fault" != rpc ]; }
'''+TRACKER+'\n'
                    call = 'record_provider_lifecycle_step open "$argument" "$state"'
                    script += ('if output=$('+call+'); then printf "%s" "$output"; else exit 7; fi\n') if conditional else call+'\n'
                    path = directory/'run.sh'
                    path.write_text(script)
                    result = subprocess.run(['bash', str(path), json.dumps(f['initial']), json.dumps(f['steps'][0]['argument']), json.dumps(tx), fault], capture_output=True, text=True)
                    if fault == 'none':
                        self.assertEqual(result.returncode, 0, result.stderr)
                        self.assertEqual(json.loads(result.stdout)['agreements'][0]['principal'], '2000')
                    else:
                        self.assertNotEqual(result.returncode, 0)
                        self.assertEqual(result.stdout, '')

    def test_actual_phase_checks_live_receipts_and_every_balance(self):
        phase = SOURCE.split('# ─── pending provider lifecycle ───', 1)[1].split('# ─── pending provider lifecycle complete ───', 1)[0]
        for fault in ('none', 'headroom', 'receipt_rpc', 'missing_refund', 'block_timeout', 'wrong_balance'):
            with self.subTest(fault=fault), tempfile.TemporaryDirectory() as directory:
                directory = Path(directory)
                f = fixture(initial_balance=10000000, capacities=(2000, 3000), duration=1000)
                if fault == 'missing_refund':
                    f['steps'][3]['tx']['events'].pop(5)
                (directory/'fixture.json').write_text(json.dumps(f))
                for name in ('provider_lifecycle_accounting.py', 'agreement_terms.py', 'economic_assertions.py', 'player_allocatable_capacity.py'):
                    (directory/name).symlink_to(ROOT/name)
                # Every broadcast/query is replaced. Execute the actual phase
                # and receipt helper, never source or run the live-chain script.
                script = '''set -euo pipefail
cd "$(dirname "$0")"
fault="$1"; step=0; printf 0 > step
GENERATOR_TEST_DIR="$PWD"
SUBSTATION_ID=4-1 GUILD_ID=0-1 PLAYER_1_ID=1-1 PLAYER_2_ID=1-2 PLAYER_3_ID=1-3
PLAYER_2_VAULT=consumer0 PLAYER_3_VAULT=consumer1
section() { :; }
sleep() { :; }
assert_eq() { [ "$2" = "$3" ]; }
get_created_provider_id() { [ "$*" = '0-1 4-1' ]; echo 10-1; }
get_vault_address() { case "$1" in 0-1) echo guild;; 1-1) echo personal;; *) return 9;; esac; }
fund_vault_alpha() { case "$*" in '1-2 2000000'|'1-3 3000000') :;; *) return 9;; esac; }
_tx_json() { printf '%s' "$1"; }
run_tx() {
  shift
  case "$*" in
    'tx structs provider-create 4-1 1ualpha open-market 0.5 0.25 1000 100000 5 100000 --owner-id 0-1 --from alice') [ "$step" = 0 ];;
    'tx structs agreement-open 10-1 1000 2000 --from player_2') [ "$step" = 0 ]; step=1;;
    'tx structs agreement-open 10-1 1000 3000 --from player_3') [ "$step" = 1 ]; step=2;;
    'tx structs provider-withdraw-balance 10-1 --from alice') [ "$step" = 2 ]; step=3;;
    'tx structs provider-delete 10-1 --from alice') [ "$step" = 3 ]; step=4;;
    *) return 9;;
  esac
  printf '%s' "$step" > step
  echo "tx:$step" >> trace
  LAST_TX_OUTPUT='{"txhash":"''' + HASH + '''"}'
}
get_balance() {
  [ "$2" = ualpha ] || return 9
  if [ "$(cat step)" = 4 ]; then
    jq -er --arg address "$1" --arg fault "$fault" '.balances[$address] | if $fault == "wrong_balance" and $address == "personal" then "9999999" else . end' fixture.json
  else
    case "$1" in
      earnings) if [ "$(cat step)" = 2 ]; then echo 21000; else echo 0; fi;;
      collateral) echo 0;;
      guild|personal|consumer0|consumer1) echo 10000000;;
      *) return 9;;
    esac
  fi
}
query() {
  case "$*" in
    'query structs substation 4-1')
      jq -cn --arg fault "$fault" '{Substation:{id:"4-1"},gridAttributes:{capacity:(if $fault == "headroom" then "5555" else "5556" end),load:"0"}}';;
    'query structs provider 10-1')
      echo '{"Provider":{"id":"10-1","owner":"0-1","substationId":"4-1","rate":{"denom":"ualpha","amount":"1"},"providerCancellationPenalty":"0.500000000000000000","consumerCancellationPenalty":"0.250000000000000000"}}';;
    'query structs provider-collateral-address 10-1') echo '{"internalAddressAssociation":[{"objectId":"10-1","address":"collateral"}]}';;
    'query structs provider-earnings-address 10-1') echo '{"internalAddressAssociation":[{"objectId":"10-1","address":"earnings"}]}';;
    'query tx ''' + HASH + '''')
      [ "$fault" != receipt_rpc ] || return 1
      jq --argjson step "$(cat step)" '.steps[$step-1].tx' fixture.json;;
    *) return 9;;
  esac
}
assert_object_missing() {
  [ "$(cat step)" = 4 ] || return 9
  case "$2:$3" in provider:10-1|agreement:11-1|agreement:11-2|allocation:6-1|allocation:6-2) :;; *) return 9;; esac
}
wait_for_block() {
  [ "$*" = 3 ] || return 9
  [ "$fault" != block_timeout ] || return 1
  echo 'wait:3' >> trace
}
'''+TRACKER+'\n'+phase+'\necho completed\n'
                path = directory/'run.sh'
                path.write_text(script)
                result = subprocess.run(['bash', str(path), fault], capture_output=True, text=True)
                if fault == 'none':
                    self.assertEqual(result.returncode, 0, result.stderr)
                    self.assertIn('completed', result.stdout)
                    self.assertEqual((directory/'trace').read_text().splitlines(),
                                     ['tx:0', 'tx:1', 'tx:2', 'wait:3', 'tx:3', 'tx:4'])
                else:
                    self.assertNotEqual(result.returncode, 0, result.stderr)
                    self.assertNotIn('completed', result.stdout)


if __name__ == '__main__':
    unittest.main()
