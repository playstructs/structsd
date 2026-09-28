#!/usr/bin/env python3
"""Add SDK share strings exactly for the live-chain migration assertions."""
import sys
from verify_upgrade_staking import shares


def sum_shares(values):
    if not values:
        raise ValueError("at least one share amount is required")
    total = sum(shares(value) for value in values)
    whole, fraction = divmod(total, 10**18)
    return f"{whole}.{fraction:018d}"


if __name__ == "__main__":
    try:
        print(sum_shares(sys.argv[1:]))
    except ValueError as error:
        raise SystemExit(str(error)) from error
