"""Off-chain rate-1, half-holdback oracle for live withdrawal and deletion."""
import argparse
import copy
import json
import re
import sys

from agreement_terms import U256, field, identity, update_terms
from economic_assertions import confirmed, integer, records


def validate_state(state):
    identity(state['provider_id'], '10')
    identity(state['provider_owner'], '0')
    identity(state['source_id'], '4')
    addresses = [state[key] for key in ('owner_vault', 'personal_vault', 'collateral_address', 'earnings_address')]
    if any(not isinstance(v, str) or not v for v in addresses) or len(set(addresses)) != 4:
        raise ValueError('fixture requires four distinct owner/provider addresses')
    for key in ('initial_owner_balance', 'initial_personal_balance'):
        integer(state[key], U256, key)
    if not isinstance(state['agreements'], list) or len(state['agreements']) > 2:
        raise ValueError('fixture requires at most two agreements')


def transfers(tx, sources):
    result = {}
    for event in tx['events']:
        if event['type'] != 'transfer' or field(event, 'sender') not in sources:
            continue
        value = field(event, 'amount')
        if not value.endswith('ualpha'):
            raise ValueError('unexpected fixture asset')
        amount = integer(value[:-6], U256, 'transfer')
        if amount == 0:
            raise ValueError('empty transfer')
        key = field(event, 'sender'), field(event, 'recipient')
        result[key] = result.get(key, 0)+amount
    return result


def record_step(previous, operation, argument, tx, txhash):
    state = copy.deepcopy(previous)
    validate_state(state)
    height = confirmed(tx, txhash)
    if 'deleted_height' in state:
        raise ValueError('provider already deleted')
    agreements = state['agreements']
    if any(not int(a['start']) <= height < int(a['end']) for a in agreements):
        raise ValueError('operation must precede expiry')
    checkpoint = integer(state.get('checkpoint', str(height)), 2**63-1, 'checkpoint')
    if checkpoint > height:
        raise ValueError('checkpoint cannot go backward')
    # Aggregate load, then truncate once per checkpoint, exactly as before v0.22.
    earned = (height-checkpoint)*sum(int(a['capacity']) for a in agreements)//2
    accrued = integer(state.get('accrued', '0'), U256, 'accrued')+earned
    collateral, earnings, owner = (state[k] for k in ('collateral_address', 'earnings_address', 'owner_vault'))
    expected = {(collateral, earnings): earned} if earned else {}
    if operation == 'open':
        if 'withdrawal' in state or len(agreements) == 2:
            raise ValueError('open belongs before withdrawal')
        base = dict(argument, provider_id=state['provider_id'], collateral_address=collateral)
        agreement = update_terms(base, 'open', argument['duration'], tx, txhash)
        initial = integer(argument['initial_balance'], U256, 'initial_balance')
        if initial < int(agreement['principal']):
            raise ValueError('consumer fixture was not funded')
        if agreement['consumer_vault'] in [state[k] for k in ('owner_vault','personal_vault','collateral_address','earnings_address')]:
            raise ValueError('consumer vault must be independent')
        for old in agreements:
            if any(old[k] == agreement[k] for k in ('agreement_id','allocation_id','consumer_owner','consumer_vault')):
                raise ValueError('consumer fixtures must be distinct')
        agreement['initial_balance'] = str(initial)
        agreements.append(agreement)
    elif operation == 'withdraw':
        if len(agreements) != 2 or 'withdrawal' in state or accrued <= 0:
            raise ValueError('withdrawal requires two live agreements and positive earnings')
        expected[earnings, owner] = accrued
        state['withdrawal'] = {'height': str(height), 'amount': str(accrued)}
        accrued = 0
    elif operation == 'delete':
        if len(agreements) != 2 or 'withdrawal' not in state:
            raise ValueError('delete must follow withdrawal')
        deleted = records(tx, 'EventDelete', 'objectId')
        for wanted in [state['provider_id']]+[a[k] for a in agreements for k in ('agreement_id','allocation_id')]:
            if deleted.count(wanted) != 1:
                raise ValueError('delete must remove provider, agreements and allocations exactly once')
        total = 0
        for agreement in agreements:
            elapsed = height-int(agreement['start'])
            capacity = int(agreement['capacity'])
            refund = (int(agreement['end'])-height)*capacity + elapsed*capacity//2
            agreement['refund'] = str(refund)
            total += int(agreement['principal'])-refund
            if refund:
                expected[collateral, agreement['consumer_vault']] = refund
        remaining = total-int(state['withdrawal']['amount'])
        if remaining < accrued:
            raise ValueError('provider earnings exceed settled entitlement')
        # Final drain includes checkpoint earnings plus the pool's rounding dust.
        if remaining:
            expected[collateral, earnings] = remaining-accrued+earned
            expected[earnings, owner] = remaining
        state.update(deleted_height=str(height), provider_total=str(total))
        accrued = 0
    else:
        raise ValueError('unknown provider operation')
    expected = {key: amount for key, amount in expected.items() if amount}
    if transfers(tx, {collateral, earnings}) != expected:
        raise ValueError('provider transfers differ from exact checkpoint/refund/drain amounts')
    state.update(checkpoint=str(height), accrued=str(accrued))
    return state


def verify_final(state, balances):
    validate_state(state)
    integer(state['deleted_height'], 2**63-1, 'deleted_height')
    total = integer(state['provider_total'], U256, 'provider_total')
    expected = {state['owner_vault']: int(state['initial_owner_balance'])+total,
                state['personal_vault']: int(state['initial_personal_balance']),
                state['collateral_address']: 0, state['earnings_address']: 0}
    for agreement in state['agreements']:
        expected[agreement['consumer_vault']] = int(agreement['initial_balance'])-int(agreement['principal'])+int(agreement['refund'])
    if set(balances) != set(expected) or any(integer(balances[k], U256, 'balance') != value for k,value in expected.items()):
        raise ValueError('final vault/pool balances do not conserve the fixture')
    return {'verified': True, 'provider_total': str(total)}


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command', choices=('open','withdraw','delete','verify'))
    parser.add_argument('state')
    parser.add_argument('argument', nargs='?', default='{}')
    parser.add_argument('txhash', nargs='?')
    args = parser.parse_args()
    try:
        state, payload = json.loads(args.state), json.load(sys.stdin)
        result = verify_final(state, payload) if args.command == 'verify' else record_step(state, args.command, json.loads(args.argument), payload, args.txhash)
        print(json.dumps(result))
    except (ValueError, KeyError, TypeError) as error:
        print(f'Invalid provider lifecycle accounting: {error}', file=sys.stderr)
        sys.exit(1)
