"""Check ordinary lazy settlement through healthy expiry after a quiet upgrade.

The off-chain oracle prices aggregate checkpoints and each contract's holdback.
No transactions, capacity changes, outages or provider deletion are permitted.
"""
import json

from agreement_terms import field
from generator_lifecycle import amount as height_value
from verify_upgrade_agreements import SCALE, atomics, quiet_window, rows, same, verify_legacy_agreements
from verify_upgrade_bank import balances, coins


def verify_legacy_expiry(before, migrated, after, migration_blocks, blocks, provider_accounts):
    baseline = verify_legacy_agreements(before, migrated, migration_blocks, provider_accounts)
    if baseline['differences']:
        raise ValueError('quiet migration has not passed')
    start, end = quiet_window(migrated, after, blocks)
    old, new = migrated['app_state']['structs'], after['app_state']['structs']
    if new.get('service') is not None:
        raise ValueError('historical billing is outside the release')
    agreements = rows(old['agreementList'], 'id')
    providers = rows(old['providerList'], 'id')
    grid = rows(old['gridList'], 'attributeId')
    final_grid = rows(new['gridList'], 'attributeId')
    checkpoints = {k: height_value(grid.get('14-'+k, {}).get('value', '0')) for k in providers}
    loads = {k: sum(int(a['capacity']) for a in agreements.values() if a['providerId']==k) for k in providers}
    expected_by_height = {}
    deletes_by_height = {}
    totals = dict.fromkeys(providers, 0)
    for expiry in sorted({int(a['endBlock']) for a in agreements.values()}):
        if not start <= expiry < end:
            raise ValueError('window must include every agreement expiry')
        due = [a for a in agreements.values() if int(a['endBlock'])==expiry]
        expected = {}
        for key in sorted({a['providerId'] for a in due}):
            provider = providers[key]
            rate, penalty = int(provider['rate']['amount']), atomics(provider['providerCancellationPenalty'])
            payment = (expiry-checkpoints[key])*rate*loads[key]*(SCALE-penalty)//SCALE
            for a in due:
                if a['providerId']==key:
                    payment += (expiry-int(a['startBlock']))*int(a['capacity'])*rate*penalty//SCALE
                    loads[key] -= int(a['capacity'])
            checkpoints[key] = expiry
            totals[key] += payment
            if payment:
                pair = provider_accounts[key]
                expected['coin_spent', pair['collateral'], provider['rate']['denom']] = payment
                expected['coin_received', pair['earnings'], provider['rate']['denom']] = payment
        expected_by_height[expiry] = expected
        deletes_by_height[expiry] = {a[k] for a in due for k in ('id', 'allocationId')}
    addresses = {v for pair in provider_accounts.values() for v in pair.values()}
    tracked = set(agreements)|{a['allocationId'] for a in agreements.values()}|set(providers)
    differences = []
    for block in blocks:
        height = height_value(block['height'])
        movements, deleted = {}, set()
        for event in block['finalize_block_events']:
            kind = event['type']
            if kind.startswith('structs.structs.EventService') or kind=='structs.structs.EventAgreement':
                raise ValueError('financial history or repricing is outside this fixture')
            if kind == 'structs.structs.EventDelete':
                key = json.loads(field(event, 'objectId'))
                if key in tracked:
                    if key in deleted or key not in deletes_by_height.get(height, set()):
                        raise ValueError('duplicate or mistimed financial object deletion')
                    deleted.add(key)
            if kind in ('coin_spent', 'coin_received'):
                address = field(event, 'spender' if kind=='coin_spent' else 'receiver')
                if address in addresses:
                    for denom, quantity in coins(field(event, 'amount')).items():
                        if quantity:
                            key = kind, address, denom
                            movements[key] = movements.get(key, 0)+quantity
        if movements != expected_by_height.get(height, {}):
            raise ValueError('checkpoint/holdback bank events differ within block')
        if deleted != deletes_by_height.get(height, set()):
            differences.append('missing agreement/allocation expiry event at '+str(height))
    if new['agreementList']: differences.append('agreements survived expiry')
    expected_allocations = {k:v for k,v in rows(old['allocationList'], 'id').items() if k not in tracked}
    if not same(rows(new['allocationList'], 'id'), expected_allocations): differences.append('allocation expiry differs')
    if not same(rows(new['providerList'], 'id'), providers): differences.append('provider terms changed')
    old_bank, new_bank = balances(migrated), balances(after)
    for key, pair in provider_accounts.items():
        denom, total = providers[key]['rate']['denom'], totals[key]
        if old_bank.get((pair['collateral'], denom), 0) < total:
            raise ValueError('fixture has insufficient collateral for ordinary expiry')
        for label, sign in [('collateral', -1), ('earnings', 1)]:
            address = pair[label]
            expected = {d:v for (a,d),v in old_bank.items() if a==address}
            expected[denom] = expected.get(denom, 0)+sign*total
            if {d:v for d,v in expected.items() if v} != {d:v for (a,d),v in new_bank.items() if a==address and v}:
                differences.append('provider pool balances differ: '+key+'/'+label)
        if final_grid.get('3-'+key, {}).get('value', '0') != str(loads[key]) or final_grid.get('14-'+key, {}).get('value', '0') != str(checkpoints[key]):
            differences.append('final provider checkpoint/load differs: '+key)
    return {'differences': differences, 'settled_agreements': sorted(agreements),
            'provider_paid': {k:str(v) for k,v in totals.items()},
            'scope': 'quiet healthy expiry; aggregate checkpoint and per-contract holdback rounding, exact pool balances and per-block bank/deletion events; residual dust stays in collateral'}
