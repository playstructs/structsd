"""Replay allocation replacements/deletions and verify legacy mode normalization."""
import copy
import json

from agreement_terms import field, identity
from generator_lifecycle import amount

MODES = {0: 'static', 1: 'dynamic', 2: 'automated', 3: 'providerAgreement'}
FIELDS = {'id', 'type', 'sourceObjectId', 'index', 'destinationId', 'creator', 'controller'}


def allocation(row, legacy=False):
    if not isinstance(row, dict) or set(row) - FIELDS:
        raise ValueError('invalid allocation record')
    key = identity(row['id'], '6')
    index = row.get('index', '0')
    if key != '6-' + str(amount(index)):
        raise ValueError('allocation id/index mismatch')
    mode = row.get('type', 'static')
    if type(mode) is int and -(2**31) <= mode < 2**31:
        mode = MODES.get(mode, mode)
    elif not isinstance(mode, str) or mode not in MODES.values():
        raise ValueError('invalid allocation enum encoding')
    if not legacy and mode not in MODES.values():
        raise ValueError('post-upgrade allocation type is undeclared')
    result = dict(id=key, type=mode, index=index)
    for name in ('sourceObjectId', 'destinationId', 'creator', 'controller'):
        value = row.get(name, '')
        if not isinstance(value, str):
            raise ValueError('allocation string field is malformed')
        result[name] = value
    return result


def inventory(document, legacy=False):
    rows = document['app_state']['structs']['allocationList']
    if not isinstance(rows, list):
        raise ValueError('allocation inventory is required')
    result = {}
    for row in rows:
        row = allocation(row, legacy)
        if row['id'] in result:
            raise ValueError('duplicate allocation id')
        result[row['id']] = row
    return result


def verify_allocations(before, after, blocks):
    start, end = int(before['initial_height']), int(after['initial_height'])
    if start <= 0 or end <= start or [int(block['height']) for block in blocks] != list(range(start, end)):
        raise ValueError('allocation events must cover the complete exported boundary')
    initial = inventory(before, legacy=True)
    expected = inventory(after)
    projected = copy.deepcopy(initial)
    repairs = {key: dict(row, type='static') for key, row in initial.items() if row['type'] not in MODES.values()}
    repaired = set()
    counts = dict(replacements=0, deletions=0, normalized=0)
    for block in blocks:
        if block.get('txs_results'):
            raise ValueError('allocation migration replay requires a transaction-free window')
        if 'finalize_block_events' not in block:
            raise ValueError('block event list is required')
        events = block['finalize_block_events']
        if events is not None and not isinstance(events, list):
            raise ValueError('block events must be a list or null')
        for event in events or []:
            kind = event['type']
            if kind not in ('structs.structs.EventAllocation', 'structs.structs.EventDelete'):
                continue
            modes = [a['value'] for a in event['attributes'] if a['key'] == 'mode']
            if len(modes) > 1 or modes and modes[0] not in ('BeginBlock', 'EndBlock'):
                raise ValueError('invalid block event phase')
            if kind == 'structs.structs.EventAllocation':
                row = allocation(json.loads(field(event, 'allocation')))
                key = row['id']
                if key in repairs:
                    if key in repaired or row != repairs[key]:
                        raise ValueError('legacy type repair must replace exactly the type once')
                    repaired.add(key)
                    counts['normalized'] += 1
                elif key in initial and row['type'] != initial[key]['type']:
                    raise ValueError('declared allocation mode changed during upgrade')
                projected[key] = row
                counts['replacements'] += 1
            else:
                key = json.loads(field(event, 'objectId'))
                if not isinstance(key, str):
                    raise ValueError('invalid deletion identity')
                if not key.startswith('6-'):
                    continue
                identity(key, '6')
                if key not in projected:
                    raise ValueError('allocation deletion is missing its record or is duplicated')
                del projected[key]
                counts['deletions'] += 1
    differences = []
    if repaired != set(repairs):
        differences.append('missing legacy allocation normalization events')
    if projected != expected:
        differences.append('allocation event projection differs from exported state')
    for key, row in expected.items():
        if key in initial and key not in repairs and row['type'] != initial[key]['type']:
            differences.append('declared allocation type changed: ' + key)
    return {'differences': differences, 'event_counts': counts, 'legacy_allocation_ids': sorted(repairs)}
