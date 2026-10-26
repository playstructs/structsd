"""Price a quiet legacy-agreement upgrade window from its pre-upgrade contracts.

This fixture oracle requires surviving live contracts to remain unmodified and
unpaid during the comparison. It does not accept outages or later transactions
as if they were baseline initialization, or certify later service delivery.
"""
import hashlib
import json
import re

from agreement_terms import field
from generator_lifecycle import amount as height_value
from verify_upgrade_bank import balances, coins
from verify_upgrade_vesting import source_address_bytes

SCALE = 10**18

def same(left, right):
    return json.dumps(left, sort_keys=True) == json.dumps(right, sort_keys=True)


def export_height(value):
    # SDK's outer GenesisDoc uses encoding/json integers; protobuf row and
    # RPC heights use canonical decimal strings. Booleans are not integers.
    if type(value) is int and 0 <= value < 2**64:
        return value
    return height_value(value)


def rows(items, key):
    if not isinstance(items, list):
        raise ValueError('complete record inventory is required')
    result = {}
    for row in items:
        if not isinstance(row, dict) or not isinstance(row.get(key), str) or not row[key] or row[key] in result:
            raise ValueError('invalid or duplicate record identity')
        result[row[key]] = row
    return result


def atomics(value):
    if not isinstance(value, str) or not re.fullmatch(r'(?:0(?:\.[0-9]{1,18})?|1(?:\.0{1,18})?)', value):
        raise ValueError('invalid penalty fraction')
    whole, _, fraction = value.partition('.')
    return int(whole)*SCALE + int(fraction.ljust(18, '0'))


def quiet_window(before, after, blocks):
    start, end = export_height(before['initial_height']), export_height(after['initial_height'])
    if start == 0 or end <= start or [height_value(b['height']) for b in blocks] != list(range(start, end)) or any(b.get('txs_results') for b in blocks):
        raise ValueError('requires a complete transaction-free comparison window')
    if any(not isinstance(b['finalize_block_events'], list) for b in blocks):
        raise ValueError('complete block events are required')
    return start, end


def verify_legacy_agreements(before, after, blocks, provider_accounts):
    height, end = quiet_window(before, after, blocks)
    old, new = before['app_state']['structs'], after['app_state']['structs']
    if old.get('service') is not None or new.get('service') is not None:
        raise ValueError('historical billing is outside the release')
    providers, agreements = rows(old['providerList'], 'id'), rows(old['agreementList'], 'id')
    if not agreements or set(provider_accounts) != set(providers):
        raise ValueError('legacy contracts and complete provider pool associations are required')
    addresses = []
    for provider, accounts in provider_accounts.items():
        for label, prefix in [('collateral', 'structs/Provider/Collateral/'), ('earnings', 'structs/Provider/Earnings/')]:
            address = accounts[label]
            if source_address_bytes(address) != hashlib.sha256((prefix+provider).encode()).digest()[:20]:
                raise ValueError('provider account is not its canonical SDK module address')
            addresses.append(address)
    if len(set(addresses)) != len(addresses):
        raise ValueError('provider pool addresses must be distinct')
    allocations = rows(old['allocationList'], 'id')
    final_allocations = rows(new['allocationList'], 'id')
    grid = rows(old['gridList'], 'attributeId')
    final_grid = rows(new['gridList'], 'attributeId')
    loads = dict.fromkeys(providers, 0)
    held = dict.fromkeys(providers, 0)
    differences = []
    for key, row in agreements.items():
        provider = providers[row['providerId']]
        start, stop, capacity = (height_value(row[n]) for n in ('startBlock', 'endBlock', 'capacity'))
        if not start < height < end <= stop or capacity == 0:
            raise ValueError('quiet migration requires preexisting agreements with no expiry in the window')
        allocation = allocations.get(row['allocationId'])
        power = '5-'+row['allocationId']
        if (not allocation or allocation.get('type') != 'providerAgreement' or allocation.get('sourceObjectId') != provider['substationId']
                or grid.get(power, {}).get('value') != str(capacity)):
            raise ValueError('legacy allocation does not match its contract')
        if not same(allocation, final_allocations.get(row['allocationId'])) or not same(grid.get(power), final_grid.get(power)):
            differences.append('live allocation or power changed: '+key)
        rate = provider['rate']['amount']
        if not isinstance(rate, str) or not re.fullmatch(r'[1-9][0-9]*', rate) or int(rate) >= 2**256:
            raise ValueError('invalid original service rate')
        penalty = atomics(provider['providerCancellationPenalty'])
        atomics(provider['consumerCancellationPenalty'])
        loads[provider['id']] += capacity
        held[provider['id']] += (stop-height)*capacity*int(rate)+(height-start)*capacity*int(rate)*penalty//SCALE
    if not same(rows(new['providerList'], 'id'), providers): differences.append('provider terms changed')
    if not same(rows(new['agreementList'], 'id'), agreements): differences.append('agreement terms changed')
    before_bank, after_bank = balances(before), balances(after)
    for key, provider in providers.items():
        pair = provider_accounts[key]
        if before_bank.get((pair['collateral'], provider['rate']['denom']), 0) < held[key]:
            raise ValueError('legacy pool cannot cover consumer refunds')
        checkpoint = height_value(grid.get('14-'+key, {}).get('value', '0'))
        if checkpoint > height or grid.get('3-'+key, {}).get('value', '0') != str(loads[key]):
            raise ValueError('legacy provider checkpoint/load is invalid')
        for attr in ('14-', '3-'):
            if not same(grid.get(attr+key), final_grid.get(attr+key)):
                differences.append('provider checkpoint/load changed: '+key)
        for address in pair.values():
            if {d:v for (a,d),v in before_bank.items() if a==address} != {d:v for (a,d),v in after_bank.items() if a==address}:
                differences.append('provider pool balance changed: '+key)
    for block in blocks:
        for event in block['finalize_block_events']:
            kind = event['type']
            if kind.startswith('structs.structs.EventService'):
                raise ValueError('historical financial event is outside the release')
            if kind in ('coin_spent', 'coin_received') and field(event, 'spender' if kind=='coin_spent' else 'receiver') in addresses:
                if any(coins(field(event, 'amount')).values()):
                    raise ValueError('quiet migration must not move provider funds')
            if kind == 'structs.structs.EventAgreement':
                row = json.loads(field(event, 'agreement'))
                if not same(agreements.get(row['id']), row): raise ValueError('unexpected agreement replacement')
            if kind == 'structs.structs.EventDelete' and json.loads(field(event, 'objectId')) in set(agreements)|set(allocations)|set(providers):
                raise ValueError('quiet migration deleted a live financial object')
    return {'differences': differences, 'live_agreements': sorted(agreements),
            'scope': 'quiet upgrade before any expiry; unchanged contracts, allocation power, provider clocks/load and pool balances; no historical billing state'}
