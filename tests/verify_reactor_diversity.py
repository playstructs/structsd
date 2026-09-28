"""Independently price exported portfolios and replay public diversity summaries.

This checks current derived accounting. Native eligibility/backing and pending
repair liveness remain separate checks; a consistent stale infusion is still stale.
"""
from collections import defaultdict
from fractions import Fraction
import json
import re

from verify_upgrade_bank import amount
from verify_upgrade_staking import shares

MAX_CAPACITY = 2**64 - 1
CHANGE = "structs.structs.EventOwnerReactorDiversity"
RESET = "structs.structs.EventOwnerReactorDiversityReset"


def integer(value):
    if not isinstance(value, str) or not re.fullmatch(r"0|[1-9][0-9]*", value):
        raise ValueError("invalid canonical diversity integer")
    return int(value)


def owner_id(value):
    if not isinstance(value, str) or not re.fullmatch(r"[01]-[1-9][0-9]*", value):
        raise ValueError("invalid diversity owner ID")
    return value


def owner_share(row):
    power, commission = amount(row["power"]), shares(row["commission"])
    if power > MAX_CAPACITY or commission > 10**18:
        raise ValueError("invalid infusion power or commission")
    # LegacyDec.RoundInt uses ties-to-even, not half-up. Fraction/round is
    # exact here, independent of the Go multiplication and rounding helpers.
    return power - round(Fraction(power * commission, 10**18))


def formula_bonus(total, squared, maximum, target):
    if not total:
        return 0
    effective_extra = Fraction(total**2, squared) - 1
    return int(Fraction(total * maximum, 10000) * min(effective_extra, target) / target)


def projection(document):
    state = document["app_state"]["structs"]
    params = state["params"]
    maximum = amount(params["diversityMaxBonusBps"])
    target = amount(params["diversityTargetExtraReactors"])
    if maximum > 10000 or not 0 < target <= MAX_CAPACITY:
        raise ValueError("invalid diversity parameters")
    owners = set()
    for field in ("playerList", "guildList"):
        for row in state[field]:
            owner = owner_id(row["id"])
            if owner in owners or owner.startswith("1-") != (field == "playerList"):
                raise ValueError("duplicate or mistyped diversity owner")
            owners.add(owner)
    base, positions, seen = defaultdict(int), defaultdict(lambda: defaultdict(int)), set()
    for row in state["infusionList"]:
        owner = owner_id(row.get("ownerId") or row.get("playerId"))
        if owner not in owners or row.get("playerId", "") not in ("", owner):
            raise ValueError("invalid infusion owner")
        if owner.startswith("0-") and row.get("playerId", ""):
            raise ValueError("guild infusion carries a player ID")
        destination, kind = row["destinationId"], row["destinationType"]
        prefix = {"reactor": "3-", "struct": "5-"}.get(kind)
        if prefix is None or not re.fullmatch(prefix + r"[1-9][0-9]*", destination):
            raise ValueError("invalid infusion destination")
        key = destination, row["address"]
        if key in seen:
            raise ValueError("duplicate infusion")
        seen.add(key)
        power = owner_share(row)
        base[owner] += power
        if kind == "reactor" and power:
            positions[owner][destination] += power
    summaries, capacities = {}, {}
    for owner in owners:
        powers = list(positions[owner].values())
        total, squared = sum(powers), sum(power**2 for power in powers)
        bonus = 0
        if total:
            bonus = formula_bonus(total, squared, maximum, target)
            bonus = min(bonus, max(0, MAX_CAPACITY - base[owner]))
            summaries[owner] = {"ownerId": owner, "power": str(total),
                                "squaredPower": str(squared), "appliedBonus": str(bonus)}
        capacities[owner] = min(MAX_CAPACITY, base[owner] + bonus)
    grid = {}
    for row in state["gridList"]:
        key, value = row["attributeId"], amount(row["value"])
        if key in grid or value > MAX_CAPACITY:
            raise ValueError("duplicate or invalid grid record")
        grid[key] = value
    differences = [f"owner primary capacity differs from infusion/diversity accounting: {owner}"
                   for owner in sorted(owners) if grid.get("2-" + owner, 0) != capacities[owner]]
    return summaries, differences


def summary(row):
    owner = owner_id(row["ownerId"])
    total, squared, bonus = (integer(row[name]) for name in ("power", "squaredPower", "appliedBonus"))
    if bonus > MAX_CAPACITY or (total == 0 and (squared or bonus)) or (total and not total <= squared <= total**2):
        raise ValueError("inconsistent diversity event totals")
    return owner, {"ownerId": owner, "power": str(total), "squaredPower": str(squared), "appliedBonus": str(bonus)}


def verify_diversity(before, after, blocks, *, restart=False, upgrade_height=None):
    start, end = max(1, int(before["initial_height"])), int(after["initial_height"])
    if end <= start or [int(block["height"]) for block in blocks] != list(range(start, end)):
        raise ValueError("diversity events must cover every exported boundary height exactly once")
    if not restart and (upgrade_height is None or not start <= upgrade_height < end):
        raise ValueError("upgrade height is required within the comparison window")
    expected, differences = projection(after)
    replay, resets, changes = {}, [], 0
    for block in blocks:
        height = int(block["height"])
        if block.get("txs_results"):
            raise ValueError("diversity replay requires a transaction-free window")
        for event in block["finalize_block_events"] or []:
            if event["type"] not in (CHANGE, RESET):
                continue
            data, seen = {}, set()
            for attribute in event["attributes"]:
                key, value = attribute["key"], attribute["value"]
                if key in seen:
                    raise ValueError("duplicate diversity event attribute")
                seen.add(key)
                if key == "mode":
                    if value not in ("BeginBlock", "EndBlock"):
                        raise ValueError("unknown diversity block event mode")
                else:
                    data[key] = json.loads(value)
            if event["type"] == RESET:
                if data:
                    raise ValueError("unexpected diversity reset payload")
                resets.append(height)
                replay.clear()
            else:
                if restart and not resets:
                    differences.append("diversity summary preceded restart reset")
                if not restart and height < upgrade_height:
                    differences.append("diversity summary preceded upgrade")
                if set(data) != {"diversity"}:
                    raise ValueError("unexpected diversity event payload")
                owner, row = summary(data["diversity"])
                if row["power"] == "0":
                    replay.pop(owner, None)
                else:
                    replay[owner] = row
                changes += 1
    if resets != ([start] if restart else []):
        differences.append("expected one first-block diversity reset on restart and none on upgrade")
    for owner in sorted(replay.keys() | expected.keys()):
        if replay.get(owner) != expected.get(owner):
            differences.append(f"diversity summary event replay differs from infusion accounting: {owner}")
    return {"differences": differences, "owner_portfolios": len(expected), "summary_events": changes,
            "reset_heights": resets, "scope": "exact commission-rounded owner/reactor aggregation, diversity formula and uint64 clipping, primary owner grid capacity, final summary event replay and restart reset; not native backing, intermediate event accuracy or repair liveness"}
