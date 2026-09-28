"""Verify simultaneous ordinary-stake refunds after a paused vesting rehearsal.

SDK 0.53 completes all mature entries for a delegator/validator together. Bank
movements are per entry; complete_unbonding reports their per-validator sum.
This verifies public effects for one vault, not arbitrary staking mutations.
"""
from collections import Counter
import re

from continue_upgrade_vesting import time_ns
from verify_upgrade_bank import amount, attributes, coins


def verify_maturity_batch(previous, current, results, pending, address, pool):
    height = int(current['height'])
    if height <= 1 or int(previous['height']) != height - 1 or int(results['height']) != height:
        raise ValueError('maturity evidence requires adjacent matching heights')
    if not previous.get('chain_id') or previous['chain_id'] != current.get('chain_id'):
        raise ValueError('maturity headers belong to different chains')
    if not address or not pool or address == pool or not isinstance(pending, dict) or not pending:
        raise ValueError('distinct refund accounts and complete pending entries are required')
    if results.get('txs_results'):
        raise ValueError('maturity block must be transaction-free')
    old_time, new_time = time_ns(previous['time']), time_ns(current['time'])
    by_validator, by_entry = Counter(), Counter()
    for identity, entry in pending.items():
        if not isinstance(identity, str) or not re.fullmatch(r'[1-9][0-9]*', identity) or entry.get('unbonding_id') != identity:
            raise ValueError('invalid pending entry identity')
        if entry.get('unbonding_on_hold_ref_count', '0') != '0':
            raise ValueError('held entry cannot be priced as a refund')
        validator = entry.get('validator_address')
        if not isinstance(validator, str) or not validator:
            raise ValueError('pending validator is required')
        if not old_time < time_ns(entry['completion_time']) <= new_time:
            raise ValueError('maturity headers must straddle every completion time')
        balance = amount(entry['balance'])
        if balance == 0:
            raise ValueError('fixture requires positive unbonding balances')
        by_validator[validator] += balance
        by_entry[balance] += 1
    events = results['finalize_block_events']
    if not isinstance(events, list):
        raise ValueError('maturity events must be present')
    completions, credits, debits = Counter(), Counter(), Counter()
    for event in events:
        kind = event['type']
        if kind not in ('complete_unbonding', 'coin_received', 'coin_spent'):
            continue
        data = attributes(event)
        matches = ((kind == 'complete_unbonding' and data.get('delegator') == address)
            or (kind == 'coin_received' and data.get('receiver') == address)
            or (kind == 'coin_spent' and data.get('spender') == pool))
        if not matches:
            continue
        if data.get('mode') != 'EndBlock':
            raise ValueError('refund effects must be EndBlock events')
        value = coins(data['amount'])
        if set(value) != {'ualpha'} or value['ualpha'] == 0:
            raise ValueError('refund must be a positive ualpha amount')
        if kind == 'complete_unbonding':
            completions[data['validator'], value['ualpha']] += 1
        elif kind == 'coin_received':
            credits[value['ualpha']] += 1
        else:
            debits[value['ualpha']] += 1
    if completions != Counter({(validator, balance): 1 for validator, balance in by_validator.items()}) or credits != by_entry or debits != by_entry:
        raise ValueError('refund completions, vault credits or pool debits differ from pending entries')
    return {'height': str(height), 'unbonding_ids': sorted(pending), 'refunded_ualpha': sum(by_validator.values()),
        'vault': address, 'pool': pool, 'completion_events': len(by_validator), 'entry_refunds': sum(by_entry.values())}
