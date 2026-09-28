"""Independent pool conservation and half-holdback floors for live phase 16b opens."""
import argparse
import json
import sys

from economic_assertions import integer


def opening_snapshot(height, collateral, earnings, claims):
    height = integer(height, 2**63-1, 'height')
    collateral = integer(collateral, 2**256-1, 'collateral')
    earnings = integer(earnings, 2**256-1, 'earnings')
    if height == 0 or not isinstance(claims, list) or not 1 <= len(claims) <= 2:
        raise ValueError('snapshot requires one or two live fixture contracts')
    principal, minimum = 0, 0
    for row in claims:
        start = integer(row['start'], 2**63-1, 'start')
        duration = integer(row['duration'], 2**64-1, 'duration')
        capacity = integer(row['capacity'], 2**64-1, 'capacity')
        if not 0 < start <= height < start+duration <= 2**64-1 or capacity == 0:
            raise ValueError('snapshot must be within each live service window')
        deposit = duration*capacity
        principal += deposit
        # Healthy service permits at most this half-rate payout by this height.
        # A lagging worker leaves more collateral, which is legitimate.
        minimum += deposit - (height-start)*capacity//2
    if collateral+earnings != principal:
        raise ValueError('deposits do not reconcile across the two pinned pools')
    if collateral < minimum:
        raise ValueError('collateral is below the live contracts remaining obligations')
    return {'principal': str(principal), 'minimum_collateral': str(minimum), 'verified': True}


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('height', 'collateral', 'earnings', 'claims'):
        parser.add_argument(name)
    args = parser.parse_args()
    try:
        print(json.dumps(opening_snapshot(args.height, args.collateral, args.earnings, json.loads(args.claims))))
    except (ValueError, KeyError, TypeError) as error:
        print(f'Invalid settlement opening snapshot: {error}', file=sys.stderr)
        sys.exit(1)
