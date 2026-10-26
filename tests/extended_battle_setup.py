"""Size the adversary's feeder while preserving other allocations and build power."""
import argparse
import json
import re
import sys

from agreement_terms import identity


def uint(value):
    if not isinstance(value, str) or len(value) > 20 or not re.fullmatch(r"0|[1-9][0-9]*", value):
        raise ValueError("fixture amounts must be uint64 decimal strings")
    number = int(value)
    if number >= 2**64:
        raise ValueError("fixture amount exceeds uint64")
    return number


def feeder_plan(document, owner, controller, destination, reserve):
    identity(owner, "1")
    identity(controller, "1")
    identity(destination, "4")
    reserve = uint(reserve)
    if not reserve:
        raise ValueError("build reserve must be positive")
    player = document["player"]
    if player["Player"]["id"] != owner:
        raise ValueError("wrong fixture player")
    grid = player["gridAttributes"]
    capacity, load = (uint(grid.get(key, "0")) for key in ("capacity", "load"))
    existing, power, target = document["allocation"], 0, ""
    if existing is not None:
        row = existing["Allocation"]
        identity(row["id"], "6")
        if row.get("sourceObjectId") != owner or row.get("controller") != controller or row.get("type") != "dynamic":
            raise ValueError("wrong fixture feeder")
        target = row.get("destinationId", "")
        if target not in ("", destination):
            raise ValueError("fixture feeder points to another destination")
        power = uint(existing["gridAttributes"].get("power", "0"))
        if power > load:
            raise ValueError("feeder power exceeds the player's allocation load")
    # Resizing releases this feeder's own load. Every other allocation remains
    # reserved; connectionCapacity and structsLoad are not allocation headroom.
    desired = capacity - (load - power) - reserve
    if desired <= 0:
        raise ValueError("insufficient primary capacity for feeder and build reserve")
    return {"power": str(desired), "previousPower": str(power), "reserve": str(reserve),
            "destinationId": target, "needsUpdate": existing is not None and desired != power}


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ("owner", "controller", "destination", "reserve"):
        parser.add_argument(name)
    args = parser.parse_args()
    try:
        print(json.dumps(feeder_plan(json.load(sys.stdin), **vars(args))))
    except (ValueError, TypeError, KeyError, AttributeError) as error:
        parser.exit(1, f"Invalid extended battle setup: {error}\n")
