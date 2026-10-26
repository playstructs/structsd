#!/usr/bin/env python3
"""Verify a healthy fixture owner's diversity at one committed query height.

The test harness reads every infusion page, then selects this owner. This is
off-chain acceptance tooling, not a proposed consensus-time scan.
"""
import argparse
import json
import subprocess
import sys

from fixture_grid_recovery import committed_height
from struct_inventory import next_cursor
from verify_reactor_diversity import formula_bonus, integer, owner_id, projection, summary


def verify_live(query, owner, height=None, page_size=100, max_pages=10000):
    owner_id(owner)
    if type(page_size) is not int or not 1 <= page_size <= 100 or type(max_pages) is not int or not 1 <= max_pages <= 10000:
        raise ValueError("invalid infusion page budget")
    height = committed_height(query, height)
    def at(*args):
        return query("query", "structs", *args, "--height", height)
    kind = "player" if owner.startswith("1-") else "guild"
    entity = at(kind, owner)
    if entity[kind.capitalize()]["id"] != owner:
        raise ValueError("owner query returned another object")
    params = at("params")["params"]
    result = at("owner-reactor-diversity", owner)
    if any(result.get(flag) is not True for flag in ("initialized", "livePortfolioComplete", "infusionInventoryComplete")):
        raise ValueError("healthy fixture requires initialized and complete diversity inventories")
    rows, seen, cursors, cursor = [], set(), set(), ""
    for _ in range(max_pages):
        args = ["infusion-all", "--page-limit", str(page_size)]
        if cursor:
            args += ["--page-key", cursor]
        page = at(*args)
        following = next_cursor(page)
        values = page.get("Infusion", [])
        if values is None:
            values = []
        if not isinstance(values, list) or len(values) > page_size:
            raise ValueError("invalid infusion page")
        for row in values:
            key = row["destinationId"], row["address"]
            if key in seen:
                raise ValueError("duplicate infusion across pages")
            seen.add(key)
            if (row.get("ownerId") or row.get("playerId")) == owner:
                # Query protobuf JSON omits zero uint64 fields. Export JSON
                # includes them; normalize only this RPC boundary before the
                # strict export oracle, preserving the captured response.
                rows.append(dict(row, power=row.get("power", "0")))
        if not following:
            break
        if following in cursors:
            raise ValueError("infusion pagination cursor cycle")
        cursors.add(following)
        cursor = following
    else:
        raise ValueError("infusion inventory exceeded its page budget")
    capacity = entity["gridAttributes"].get("capacity", "0")
    document = {"app_state": {"structs": {"params": params,
        "playerList": [{"id": owner}] if kind == "player" else [],
        "guildList": [{"id": owner}] if kind == "guild" else [],
        "infusionList": rows, "gridList": [{"attributeId": "2-" + owner, "value": capacity}]}}}
    expected, differences = projection(document)
    if differences:
        raise ValueError("; ".join(differences))
    wanted = expected.get(owner, {"ownerId": owner, "power": "0", "squaredPower": "0", "appliedBonus": "0"})
    stored = dict(result["stored"])
    stored.setdefault("appliedBonus", "0")
    _, stored = summary(stored)
    if stored != wanted:
        raise ValueError("owner diversity query differs from complete infusion accounting")
    maximum, target = (integer(params[name]) for name in ("diversityMaxBonusBps", "diversityTargetExtraReactors"))
    if integer(result.get("maxBonusBps", "0")) != maximum or integer(result["targetExtraReactors"]) != target:
        raise ValueError("diversity query parameters differ from module parameters")
    live = formula_bonus(integer(wanted["power"]), integer(wanted["squaredPower"]), maximum, target)
    if integer(result["liveBonus"]) != live or integer(result.get("usableBonus", "0")) != integer(wanted["appliedBonus"]):
        raise ValueError("healthy fixture live bonus differs from independently priced portfolio")
    if any(integer(result.get(field, "0")) != integer(capacity) for field in ("storedCapacity", "usablePrimaryCapacity")):
        raise ValueError("healthy fixture diversity capacity disagrees with owner grid")
    return {"status": "verified", "height": height, "owner": owner,
            "infusions": len(rows), "summary": wanted, "capacity": capacity}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("owner")
    parser.add_argument("--height")
    arguments = sys.argv[1:]
    boundary = arguments.index("--") if "--" in arguments else len(arguments)
    args = parser.parse_args(arguments[:boundary])
    command = arguments[boundary + 1:]
    try:
        if not command:
            raise ValueError("live diversity verification requires a CLI command")
        def query(*options):
            result = subprocess.run([*command, *options], check=True, capture_output=True, text=True, timeout=30)
            return json.loads(result.stdout)
        print(json.dumps(verify_live(query, args.owner, args.height)))
    except (ValueError, KeyError, TypeError, OSError, subprocess.SubprocessError) as error:
        parser.exit(1, f"Live diversity verification failed: {error}\n")


if __name__ == "__main__":
    main()
