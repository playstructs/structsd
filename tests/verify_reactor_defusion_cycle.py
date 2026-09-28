#!/usr/bin/env python3
"""Check a healthy reactor's live defuse/cancel fixture using exact integers."""
import json
import re
import sys


def uint(value, label, maximum=2**64 - 1):
    if not isinstance(value, str) or not re.fullmatch(r"0|[1-9][0-9]*", value):
        raise ValueError(f"{label} must be a canonical integer string")
    number = int(value)
    if number > maximum:
        raise ValueError(f"{label} is out of range")
    return number


def verify_cycle(document, reactor, vault, player, amount):
    if type(amount) is not int or not 0 < amount < 2**64:
        raise ValueError("invalid cycle amount")
    positions = []
    for phase in ("infused", "defused", "cancelled"):
        row = document[phase]["Infusion"]
        if (row.get("destinationId"), row.get("address"), row.get("playerId")) != (reactor, vault, player):
            raise ValueError(f"{phase} infusion identity mismatch")
        if row.get("ownerId", "") not in ("", player):
            raise ValueError(f"{phase} asset owner mismatch")
        fuel = uint(row.get("fuel", "0"), f"{phase} fuel")
        if uint(row.get("ratio", "0"), f"{phase} ratio") != 1:
            raise ValueError(f"{phase} fixture reactor is not producing power")
        if uint(row.get("power", "0"), f"{phase} power") != fuel:
            raise ValueError(f"{phase} power does not match flat backing")
        defusing = uint(row.get("defusing", "0"), f"{phase} defusing")
        positions.append((fuel, defusing))
    before, withdrawn, restored = positions
    if before[0] <= amount:
        raise ValueError("fixture must retain fuel during defusion")
    if withdrawn != (before[0] - amount, before[1] + amount):
        raise ValueError("defusion did not move the requested backing into unbonding")
    if restored != before:
        raise ValueError("cancellation did not restore the original backing")


if __name__ == "__main__":
    try:
        verify_cycle(json.load(sys.stdin), sys.argv[1], sys.argv[2], sys.argv[3], int(sys.argv[4]))
    except (ValueError, KeyError, TypeError, IndexError) as error:
        raise SystemExit(str(error)) from error
    print("verified")
