"""Independent integer model for native vesting-vault balance observations.

Preserve each source's rounding and delegated-vesting offset. This does not
prove source authenticity, staking backing or successful transaction execution.
"""
import re

from verify_upgrade_bank import amount


PRECISION = 10**18


def round_even(numerator, denominator):
    whole, remainder = divmod(numerator, denominator)
    if remainder * 2 > denominator or (remainder * 2 == denominator and whole % 2):
        whole += 1
    return whole


def sdk_fraction(numerator, denominator):
    # LegacyDec.Quo divides at 36 places before rounding to 18.
    return round_even(numerator * PRECISION**2 // denominator, PRECISION)


def coin_map(rows):
    if rows is None:
        rows = []
    if not isinstance(rows, list):
        raise ValueError("invalid vesting coin list")
    result = {}
    for row in rows:
        if not isinstance(row, dict):
            raise ValueError("invalid vesting coin entry")
        denom = row.get("denom")
        if not isinstance(denom, str) or not re.fullmatch(r"[a-zA-Z][a-zA-Z0-9/:._-]{2,127}", denom) or denom in result:
            raise ValueError("invalid or duplicate vesting denomination")
        value = amount(row.get("amount"))
        if value == 0:
            raise ValueError("vesting coin entries must be positive")
        result[denom] = value
    return result


def timestamp(value):
    if not isinstance(value, str) or not re.fullmatch(r"0|-?[1-9][0-9]{0,18}", value):
        raise ValueError("invalid vesting timestamp")
    result = int(value)
    if not -(2**63) <= result < 2**63:
        raise ValueError("vesting timestamp exceeds int64")
    return result


def add(target, coins):
    for denom, value in coins.items():
        total = target.get(denom, 0) + value
        if total >= 2**256:
            raise ValueError("vesting aggregate exceeds SDK coin range")
        if total:
            target[denom] = total


def schedule_balances(schedule, at):
    if type(at) is not int or not -(2**63) <= at < 2**63:
        raise ValueError("invalid observation time")
    kind = schedule.get("kind")
    if type(kind) is not int or kind not in (1, 2, 3, 4):
        raise ValueError("unknown vesting schedule kind")
    start, end = (timestamp(schedule.get(field, "0")) for field in ("start_time", "end_time"))
    if kind in (2, 4) and start != 0 or kind == 4 and end != 0:
        raise ValueError("unexpected vesting time field")
    if kind in (1, 3) and start >= end:
        raise ValueError("vesting interval is empty or reversed")
    original = coin_map(schedule.get("original_vesting"))
    delegated_vesting = coin_map(schedule.get("delegated_vesting"))
    delegated_free = coin_map(schedule.get("delegated_free"))
    if any(value > original.get(denom, 0) for denom, value in delegated_vesting.items()):
        raise ValueError("delegated vesting exceeds original schedule")
    periods = schedule.get("periods", [])
    if periods is None:
        periods = []
    if not isinstance(periods, list) or kind != 3 and periods:
        raise ValueError("unexpected vesting periods")
    vested = {}
    if kind == 1:
        if at >= end:
            vested = dict(original)
        elif at > start:
            scalar = sdk_fraction(at - start, end - start)
            vested = {denom: round_even(value * scalar, PRECISION) for denom, value in original.items()}
    elif kind == 2:
        if at >= end:
            vested = dict(original)
    elif kind == 3:
        cumulative, all_coins = start, {}
        for period in periods:
            length = timestamp(period.get("length"))
            if length < 0:
                raise ValueError("negative vesting period")
            cumulative += length
            if cumulative >= 2**63:
                raise ValueError("periodic vesting time overflows")
            coins = coin_map(period.get("amount"))
            add(all_coins, coins)
            # SDK treats time == start as unvested, even for a zero-length
            # first period. Later observations consume completed periods.
            if at > start and at >= cumulative:
                add(vested, coins)
        if cumulative != end or all_coins != original:
            raise ValueError("periodic installments do not equal schedule totals")
    vesting = {denom: value - vested.get(denom, 0) for denom, value in original.items()}
    locked = {denom: max(value - delegated_vesting.get(denom, 0), 0) for denom, value in vesting.items()}
    return {name: {denom: value for denom, value in coins.items() if value}
        for name, coins in (("original", original), ("vested", vested), ("vesting", vesting),
            ("locked", locked), ("delegated_vesting", delegated_vesting), ("delegated_free", delegated_free))}


def vault_balances(vault, at):
    schedules = vault.get("schedules")
    if not isinstance(schedules, list) or not schedules:
        raise ValueError("vesting vault requires schedules")
    total = {name: {} for name in ("original", "vested", "vesting", "locked", "delegated_vesting", "delegated_free")}
    for schedule in schedules:
        for name, values in schedule_balances(schedule, at).items():
            add(total[name], values)
    add(total["delegated_free"], coin_map(vault.get("liquid_delegated")))
    return total
