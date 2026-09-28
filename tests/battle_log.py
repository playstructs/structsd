"""Classify a delivered attack transaction without counting refusals as events."""
import argparse
import json
import sys

from economic_assertions import confirmed, records
from economic_assertions import integer


def capture(tx, txhash, description, timestamp):
    if not isinstance(tx, dict) or type(tx.get('code')) not in (int, str):
        raise ValueError('missing or malformed delivery code')
    code = integer(str(tx['code']), 2**32 - 1, 'code')
    # Reuse envelope validation for failed receipts too; never change the input.
    height = confirmed({**tx, 'code': 0}, txhash)
    details = records(tx, 'EventAttack', 'eventAttackDetail')
    events = [event for event in tx['events'] if event['type'] == 'structs.structs.EventAttack']
    if code == 0 and len(details) != 1:
        raise ValueError('successful attack must contain exactly one typed attack event')
    if code != 0 and events:
        raise ValueError('failed transaction cannot certify an attack event')
    result = {'type': 'attack' if code == 0 else 'attack_rejected', 'timestamp': timestamp,
              'description': description, 'txhash': tx['txhash'], 'height': str(height),
              'code': code, 'events': events}
    if code != 0:
        result['raw_log'] = tx.get('raw_log', '')
    return result


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('txhash')
    parser.add_argument('description')
    parser.add_argument('timestamp')
    args = parser.parse_args()
    try:
        print(json.dumps(capture(json.load(sys.stdin), args.txhash, args.description, args.timestamp)))
    except (ValueError, KeyError, TypeError) as error:
        parser.exit(1, f'Unverified attack log: {error}\n')
