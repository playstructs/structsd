"""Ordinary rate-1, zero-penalty accounting from confirmed public terms.

Capacity rescaling leaves rounding dust in the pool until provider deletion.
This off-chain transcript is not an on-chain billing history.
"""
import argparse
import json
import re
import sys

from economic_assertions import field, integer

U64 = 2**64-1
U256 = 2**256-1


def identity(value, prefix):
    if not isinstance(value, str) or re.fullmatch(prefix + r'-[0-9]+', value) is None:
        raise ValueError('invalid fixture object identity')
    return value


def update_terms(previous, operation, amount, tx, txhash):
    state = dict(previous)
    identity(state['provider_id'], '10')
    identity(state['consumer_owner'], '[01]')
    for key in ('consumer_vault', 'collateral_address'):
        if not isinstance(state[key], str) or not state[key]:
            raise ValueError('funding addresses are required')
    if state['consumer_vault'] == state['collateral_address']:
        raise ValueError('consumer and collateral accounts must differ')
    if (not isinstance(txhash, str) or re.fullmatch('[0-9A-Fa-f]{64}', txhash) is None
            or not isinstance(tx.get('txhash'), str) or tx['txhash'].upper() != txhash.upper()
            or type(tx.get('code')) not in (str, int) or tx['code'] not in (0, '0')):
        raise ValueError('transaction must be confirmed, successful and hash-matched')
    height = integer(tx.get('height'), 2**63-1, 'height')
    delta = integer(amount, U64, 'change')
    capacity = integer(state['capacity'], U64, 'capacity')
    if height == 0 or capacity == 0 or state.get('closed', False):
        raise ValueError('invalid live fixture')
    deposit = 0
    if operation == 'open':
        if delta == 0 or 'agreement_id' in state:
            raise ValueError('opening requires a new positive-duration fixture')
        start, end, principal, gross, credit = height, height+delta, delta*capacity, 0, 0
        deposit = principal
    else:
        identity(state['agreement_id'], '11')
        identity(state['allocation_id'], '6')
        start, end, last = (integer(state[key], U64, key) for key in ('start', 'end', 'last_height'))
        principal, gross, credit = (integer(state[key], U256, key) for key in ('principal', 'gross', 'rounding_dust'))
        if not start <= last <= height <= end or start >= end:
            raise ValueError('change lies outside the current service window')
        if principal != gross + credit + (end-start)*capacity:
            raise ValueError('prior transcript does not conserve principal')
        if operation in ('increase', 'decrease'):
            next_capacity = capacity+delta if operation == 'increase' else capacity-delta
            if delta == 0 or not 0 < next_capacity <= U64:
                raise ValueError('invalid capacity change')
            remaining = (end-height)*capacity
            duration, remainder = divmod(remaining, next_capacity)
            if duration == 0:
                raise ValueError('capacity change buys no whole block')
            gross += (height-start)*capacity
            credit += remainder
            start, end, capacity = height, height+duration, next_capacity
        elif operation == 'extend':
            if delta == 0 or height >= end:
                raise ValueError('extension requires a live positive duration')
            deposit = delta*capacity
            end += delta
            principal += deposit
        elif operation == 'close':
            if delta != 0:
                raise ValueError('close accepts no term change')
            gross += (height-start)*capacity
            state.update(closed=True, provider_expected=str(gross), consumer_expected=str(principal-gross-credit))
        else:
            raise ValueError('unknown lifecycle operation')
    if end > U64 or principal > U256 or gross+credit > principal:
        raise ValueError('service arithmetic exceeds its range')
    events = tx['events']
    if operation == 'close':
        deleted = [json.loads(field(event, 'objectId')) for event in events if event['type'] == 'structs.structs.EventDelete']
        if deleted.count(state['agreement_id']) != 1:
            raise ValueError('close must delete the intended agreement exactly once')
    else:
        records = [json.loads(field(event, 'agreement')) for event in events if event['type'] == 'structs.structs.EventAgreement']
        if len(records) != 1:
            raise ValueError('transaction requires one agreement replacement')
        row = records[0]
        agreement = identity(row['id'], '11')
        allocation = identity(row['allocationId'], '6')
        expected = {'id': state.get('agreement_id', agreement), 'allocationId': state.get('allocation_id', allocation),
                    'providerId': state['provider_id'], 'owner': state['consumer_owner'],
                    'capacity': str(capacity), 'startBlock': str(start), 'endBlock': str(end)}
        if any(row.get(key) != value for key, value in expected.items()):
            raise ValueError('delivered agreement differs from independently priced terms')
        state.update(agreement_id=agreement, allocation_id=allocation)
    if deposit:
        movements = [event for event in events if event['type'] == 'transfer'
                     and field(event, 'sender') == state['consumer_vault']
                     and field(event, 'recipient') == state['collateral_address']]
        if len(movements) != 1 or field(movements[0], 'amount') != f'{deposit}ualpha':
            raise ValueError('transaction must transfer the exact added collateral')
    state.update(capacity=str(capacity), start=str(start), end=str(end), last_height=str(height),
                 principal=str(principal), gross=str(gross), rounding_dust=str(credit))
    return state


def final_balances(state, vault, collateral, earnings):
    if state.get('closed') is not True:
        raise ValueError('final balances require a delivered close')
    principal, provider, consumer = (integer(state[key], U256, key) for key in ('principal', 'provider_expected', 'consumer_expected'))
    vault, collateral, earnings = (integer(value, U256, 'balance') for value in (vault, collateral, earnings))
    dust = integer(state['rounding_dust'], U256, 'rounding_dust')
    if provider+consumer+dust != principal or vault < principal or earnings+provider > U256:
        raise ValueError('final accounting does not conserve the funded fixture')
    return {'vault': str(vault-principal+consumer), 'collateral': str(collateral+dust), 'earnings': str(earnings+provider)}


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest='command', required=True)
    step = commands.add_parser('step')
    for name in ('operation', 'amount', 'state', 'txhash'):
        step.add_argument(name)
    balances = commands.add_parser('balances')
    for name in ('state', 'vault', 'collateral', 'earnings'):
        balances.add_argument(name)
    args = parser.parse_args()
    try:
        state = json.loads(args.state)
        if args.command == 'step':
            result = update_terms(state, args.operation, args.amount, json.load(sys.stdin), args.txhash)
        else:
            result = final_balances(state, args.vault, args.collateral, args.earnings)
        print(json.dumps(result))
    except (ValueError, KeyError, TypeError) as error:
        print(f'Invalid agreement lifecycle evidence: {error}', file=sys.stderr)
        sys.exit(1)
