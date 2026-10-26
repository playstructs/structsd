"""Read exact creation heights from a complete vault unbonding query."""
import argparse
import json
import re
import sys


def height(value):
    if not isinstance(value, str) or len(value) > 19 or re.fullmatch(r"0|[1-9][0-9]*", value) is None:
        raise ValueError("creation height must be a canonical int64 string")
    result = int(value)
    if result > 2**63 - 1:
        raise ValueError("creation height exceeds int64")
    return result


def creation_heights(response, delegator, validator):
    if not isinstance(response, dict) or not isinstance(response.get("pagination"), dict):
        raise ValueError("unbonding response must contain pagination metadata")
    page = response["pagination"]
    if "next_key" in page and "nextKey" in page:
        raise ValueError("ambiguous unbonding cursor")
    cursor = page.get("next_key", page.get("nextKey"))
    if cursor is not None and cursor != "":
        raise ValueError("unbonding query is incomplete or has an invalid cursor")
    rows = response.get("unbonding_responses")
    if rows is None:
        rows = []  # Native SDK JSON represents an empty repeated field as null.
    if not isinstance(rows, list):
        raise ValueError("unbonding records must be an array or null")
    selected, seen = [], set()
    for row in rows:
        if not isinstance(row, dict) or row.get("delegator_address") != delegator:
            raise ValueError("unbonding record has the wrong delegator")
        target = row.get("validator_address")
        if not isinstance(target, str) or not target or target in seen:
            raise ValueError("missing or duplicate unbonding validator")
        seen.add(target)
        entries = row.get("entries")
        if not isinstance(entries, list):
            raise ValueError("unbonding record must contain an entry array")
        for entry in entries:
            if not isinstance(entry, dict):
                raise ValueError("invalid unbonding entry")
            value = height(entry.get("creation_height"))
            if target == validator:
                selected.append(value)
    return selected


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("delegator")
    parser.add_argument("validator")
    parser.add_argument("--creation-height", help="count entries at this height instead of finding the latest")
    args = parser.parse_args()
    values = creation_heights(json.load(sys.stdin), args.delegator, args.validator)
    if args.creation_height is not None:
        print(values.count(height(args.creation_height)))
    else:
        print(max(values) if values else "")
