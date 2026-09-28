"""Compute allocatable power, including the fixed 10% substation operating loss."""
import argparse
import json
import re
import sys


def allocatable_capacity(query):
    grid = query["gridAttributes"]
    if not isinstance(grid, dict):
        raise ValueError("gridAttributes must be an object")
    amounts = []
    for field in ("capacity", "load"):
        value = grid.get(field, "0")
        if not isinstance(value, str) or len(value) > 20 or re.fullmatch(r"0|[1-9][0-9]*", value) is None:
            raise ValueError(f"{field} must be a uint64 decimal string")
        amount = int(value)
        if amount > 2**64 - 1:
            raise ValueError(f"{field} exceeds uint64")
        amounts.append(amount)
    capacity = amounts[0]
    if isinstance(query.get("Substation"), dict):
        capacity = capacity * 9 // 10
    return max(capacity - amounts[1], 0)


def lifecycle_capacity(query):
    available = allocatable_capacity(query)
    if available < 4:
        raise ValueError("allocation lifecycle requires at least 4 units of free primary capacity")
    return {"allocatableCapacity": str(available), "half": str(available // 2),
            "quarter": str(available // 4), "threeQuarter": str(available * 3 // 4)}


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--lifecycle", action="store_true")
    parser.add_argument("--substation-id", help="require this substation and include its exact capacity/load")
    parser.add_argument("--required-capacity", type=int, default=1,
                        help="minimum free capacity for hasCapacity (default: 1)")
    args = parser.parse_args()
    if not 1 <= args.required_capacity < 2**64:
        parser.error("required capacity must be between 1 and uint64 maximum")
    query = json.load(sys.stdin)
    if args.substation_id:
        if not isinstance(query, dict) or not isinstance(query.get("Substation"), dict) or query["Substation"].get("id") != args.substation_id:
            raise ValueError("substation query does not identify the requested object")
    if args.lifecycle:
        result = lifecycle_capacity(query)
    else:
        available = allocatable_capacity(query)
        result = {"allocatableCapacity": str(available), "hasCapacity": available >= args.required_capacity}
    if args.substation_id:
        result.update({field: query["gridAttributes"].get(field, "0") for field in ("capacity", "load")})
    print(json.dumps(result))
