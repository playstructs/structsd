"""Verify two-ship batch atomicity and exact status events at delivered heights."""
import argparse
import copy
import json
import sys

from combat_fixture import fields, fixture
from economic_assertions import confirmed, records
from economic_assertions import integer

MISSING = '5-18446744073709551615'
STAGES = {'reject': ('ready', 'ready'), 'offline': ('ready', 'ready'),
          'first': ('activate', 'activate'), 'second': ('ready', 'activate')}


def before(document, mode, first, second, owner, fleet):
    if first == second or MISSING in (first, second):
        raise ValueError('batch requires two distinct live ships and one absent id')
    for key, sid, expected in zip(('first', 'second'), (first, second), STAGES[mode]):
        row = document[key]
        if fixture(row, sid, owner, '2', fleet, 'space') != expected:
            raise ValueError('batch role is not in its required state')
        status = integer(row['structAttributes']['status'], 2**64-1, 'status')
        if bool(status & 4) != (expected == 'ready'):
            raise ValueError('status bit disagrees with online flag')
    return {'verified': True}


def verify(document, mode, txhash):
    start = document['before']; a, b = (start[k]['Struct'] for k in ('first', 'second'))
    before(start, mode, a['id'], b['id'], a['owner'], a['locationId'])
    tx = document['tx']
    if mode == 'reject':
        fields(tx, {'code': 1, 'codespace': 'undefined'})
        height = confirmed({**tx, 'code': 0}, txhash)
        reason = f"struct ({MISSING}) not found"
        if not isinstance(tx.get('raw_log'), str) or reason not in tx['raw_log'] or 'panic' in tx['raw_log'].lower():
            raise ValueError('batch failed for the wrong reason')
    else:
        height = confirmed(tx, txhash)
    expected = copy.deepcopy(start)
    changed = ('first', 'second') if mode == 'offline' else (mode,) if mode in ('first', 'second') else ()
    statuses = {}
    for key in changed:
        attrs = expected[key]['structAttributes']
        status = int(attrs['status'])
        attrs['status'] = str(status & ~4 if mode == 'offline' else status | 4)
        attrs['isOnline'] = mode != 'offline'
        statuses['1-'+expected[key]['Struct']['id']] = attrs['status']
    current = copy.deepcopy(document['after'])
    for key in ('first', 'second'):
        # Proto JSON omits false; null or other malformed values are not defaults.
        expected[key]['structAttributes'].setdefault('isOnline', False)
        current[key]['structAttributes'].setdefault('isOnline', False)
    if current != expected:
        raise ValueError('batch changed an unexpected field or failed to update both ships')
    events = records(tx, 'EventStructAttribute', 'structAttributeRecord')
    if len(events) != len(statuses) or {r.get('attributeId'): r.get('value') for r in events} != statuses:
        raise ValueError('batch status events disagree with committed state')
    return {'verified': True, 'mode': mode, 'height': str(height), 'txhash': txhash,
            'first': a['id'], 'second': b['id']}


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command', choices=('before', 'verify'))
    parser.add_argument('mode', choices=tuple(STAGES))
    parser.add_argument('arguments', nargs='*')
    args = parser.parse_args()
    try:
        print(json.dumps(globals()[args.command](json.load(sys.stdin), args.mode, *args.arguments)))
    except (ValueError, KeyError, TypeError, AttributeError) as error:
        parser.exit(1, f'Invalid batch-deactivation evidence: {error}\n')
