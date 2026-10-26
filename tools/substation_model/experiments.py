"""Reproducible adversarial experiments, not a forecast or chain exploit test."""
from __future__ import annotations

import argparse
import csv
from dataclasses import replace
import hashlib
import json
import math
from pathlib import Path
import random
import statistics

from model import (ALPHA, KW, MW, BPS, Grid, Parameters, bounty,
                   calculate_difficulty, ceil_div, health_fraction, project, upkeep_price)


def cycle(supply: int, demand: int, interval: int, paid_days: int,
          p: Parameters) -> dict:
    span = interval*p.blocks_per_day
    steps = interval*96  # 15-minute nominal observation; exact endpoint below.
    levels = [project(supply, j*span//steps,
                      BPS if j*span//steps < paid_days*p.blocks_per_day else p.durable_bps, p)
              for j in range(steps)]
    last = project(supply, span-1,
                   BPS if span-1 < paid_days*p.blocks_per_day else p.durable_bps, p)
    return {"supply_kw": supply/KW, "demand_kw": demand/KW,
            "interval_days": interval, "paid_days": paid_days,
            "alpha_per_day": upkeep_price(supply, paid_days, p)/ALPHA/interval,
            "online_fraction": sum(x >= demand for x in levels)/len(levels),
            "min_service_ratio": last/demand,
            "average_capacity_coverage": sum(min(1, x/demand) for x in levels)/len(levels)}


def upkeep_search(p: Parameters) -> list[dict]:
    rows = []
    demand = 16_300*KW
    for ratio in [1, 1.1, 1.25, 1.43, 1.5, 2, 3, 5, 8, 12, 25]:
        supply = round(demand*ratio)
        for interval in range(1, 30):
            rows.append(cycle(supply, demand, interval, 1, p))
            if 7 <= interval:
                rows.append(cycle(supply, demand, interval, 7, p))
    return rows


def prepayment(p: Parameters) -> list[dict]:
    rows = []
    for low_kw in [1, 10, 100, 1000]:
        for policy in ["naive", "capacity_ceiling", "sticky_billable_ceiling"]:
            g = Grid(p)
            s = g.add_station("leaf", 100*MW)
            g.prepaid_policy = "naive" if policy == "naive" else "capacity_ceiling"
            g.advance(29*p.blocks_per_day)
            before = g.shared("leaf")
            # Controller temporarily lowers its own accepted input; no outgoing
            # contracts, so no collateral or provider settlement bypass assumed.
            g.root_change("leaf", low_kw*KW)
            billable = 100*MW if policy == "sticky_billable_ceiling" else None
            g.pay("leaf", 1, 100_000*ALPHA, billable=billable)
            g.root_change("leaf", 100*MW)
            g.drain()
            rows.append({"policy": policy, "low_supply_kw": low_kw,
                         "price_alpha": s.burn/ALPHA,
                         "honest_price_alpha": upkeep_price(100*MW, 1, p)/ALPHA,
                         "before_kw": before/KW, "after_kw": g.shared("leaf")/KW,
                         "saving_fraction": 1-s.burn/upkeep_price(100*MW, 1, p)})
    return rows


def oldest_attack(p: Parameters) -> list[dict]:
    rows = []
    for withdrawal in [1, 3, 5, 6, 10, 20, 50, 90]:
        for oldest_pct in [10, 50, 90, 99]:
            for fragments in [1, 100]:
                g = Grid(p)
                g.add_station("hub", 100*MW)
                g.add_station("older_customers")
                g.add_station("newer_customers")
                total = g.safe("hub")
                old = total*oldest_pct//100
                pieces = [old//fragments]*(fragments-1)
                pieces.append(old-sum(pieces))
                for piece in pieces:
                    g.connect("hub", "older_customers", piece)
                g.connect("hub", "newer_customers", total-old)
                g.root_change("hub", 100*MW*(100-withdrawal)//100)
                blocks = g.drain()
                remaining = g.load("hub")
                rows.append({"withdrawal_pct": withdrawal, "oldest_pct": oldest_pct,
                             "old_fragments": fragments,
                             "installed_before_kw": total/KW,
                             "installed_after_kw": remaining/KW,
                             "commitments_lost_pct": 100*(total-remaining)/total,
                             "older_group_remaining_pct": 100*g.supply("older_customers")/old,
                             "cascade_blocks": blocks})
    return rows


def ghost_power(p: Parameters) -> tuple[list[dict], dict]:
    g = Grid(p)
    g.add_station("neglected_relay", 100*MW)
    leaf = g.add_station("maintained_leaf")
    g.connect("neglected_relay", "maintained_leaf", g.safe("neglected_relay"), automated=True)
    rows = []
    for day in range(61):
        g.advance(day*p.blocks_per_day)
        if day % 7 == 0:
            g.pay("maintained_leaf", 7, 1_000_000*ALPHA)
        rows.append({"day": day, "relay_throughput_kw": g.throughput("neglected_relay")/KW,
                     "installed_claim_kw": g.supply("maintained_leaf")/KW,
                     "leaf_available_kw": g.shared("maintained_leaf")/KW,
                     "queued_work": len(g.queue), "leaf_burn_alpha": leaf.burn/ALPHA})
    before = g.shared("maintained_leaf")
    audit = g.audit("neglected_relay", achieved=5)
    blocks = g.drain()
    summary = {"before_audit_leaf_kw": before/KW, "after_audit_leaf_kw": g.shared("maintained_leaf")/KW,
               "successful_proof_assumed": True, "audit_reward_alpha": audit["reward_ualpha"]/ALPHA,
               "cascade_blocks": blocks, "automatic_age_queue_entries": rows[-1]["queued_work"]}
    return rows, summary


def loop_control(p: Parameters) -> dict:
    g = Grid(p)
    g.add_station("A", 100*MW)
    g.add_station("B")
    ab = g.connect("A", "B", g.safe("A"), automated=True)
    ba = g.connect("B", "A", g.safe("B"), automated=True)
    iterations = 0
    for _ in range(1000):
        before = (ab.power, ba.power)
        g.resize(ab, g.safe("A"), admission=True)
        g.resize(ba, g.safe("B"), admission=True)
        iterations += 1
        if before == (ab.power, ba.power):
            break
    useful = g.shared("A")+g.shared("B")
    settled_edges = (ab.power, ba.power)
    g.root_change("A", 0)
    blocks = g.drain(budget=4)
    return {"external_supply_kw": 100_000, "feedback_iterations": iterations,
            "settled_edge_kw": [x/KW for x in settled_edges],
            "useful_before_withdrawal_kw": useful/KW, "drain_blocks": blocks,
            "useful_after_withdrawal_kw": (g.shared("A")+g.shared("B"))/KW,
            "edges_after_mw": [ab.power, ba.power],
            "energy_amplification_found": useful > 100*MW}


def fragmentation(p: Parameters) -> list[dict]:
    rows = []
    for count in [1, 32, 256, 2048, 8192]:
        g = Grid(p)
        g.add_station("hub", 100*MW)
        g.add_station("leaf")
        size = g.safe("hub")//count
        for _ in range(count):
            g.connect("hub", "leaf", size)
        g.root_change("hub", 0)
        blocks = g.drain()
        rows.append({"allocation_count": count, "cascade_blocks": blocks,
                     "final_leaf_kw": g.shared("leaf")/KW,
                     "nominal_seconds": blocks*5.55})
    return rows


def treasury(p: Parameters) -> list[dict]:
    rows = []
    for initial in [0, 1, 50, 134, 1000]:
        for revenue_bps in [0, 10, 25, 50, 100, 500]:
            for mode in ["fixed_load", "shed_to_essential"]:
                g = Grid(p)
                g.add_station("guild", 29_600*KW)
                wallet = initial*ALPHA
                daily_income = 3000*ALPHA
                demand, essential = 16_300*KW, 3500*KW
                full_ticks, earning_ticks, purchases, first_failure = 0, 0, 0, None
                for day in range(90):
                    g.advance(day*p.blocks_per_day)
                    if day-g.stations["guild"].last_upkeep//p.blocks_per_day >= 7 or day == 0:
                        try:
                            wallet = g.pay("guild", 1, wallet)
                            purchases += 1
                        except ValueError:
                            pass
                    for tick in range(96):
                        g.advance(day*p.blocks_per_day + tick*p.blocks_per_day//96)
                        available = g.shared("guild")
                        fully_online = available >= demand
                        earning = fully_online or (mode == "shed_to_essential" and available >= essential)
                        full_ticks += fully_online
                        earning_ticks += earning
                        if not fully_online and first_failure is None:
                            first_failure = g.height/p.blocks_per_day
                        if earning:
                            wallet += daily_income*revenue_bps//(BPS*96)
                rows.append({"initial_alpha": initial, "treasury_income_bps": revenue_bps,
                             "load_policy": mode, "upkeep_purchases": purchases,
                             "full_online_fraction": full_ticks/(90*96),
                             "earning_fraction": earning_ticks/(90*96),
                             "first_blackout_day": first_failure,
                             "final_treasury_alpha": wallet/ALPHA,
                             "final_available_kw": g.shared("guild")/KW,
                             "permanent_lockup_in_closed_model": g.shared("guild") == 0 and wallet < upkeep_price(g.supply("guild"), 1, p)})
    return rows


def glut(p: Parameters) -> list[dict]:
    rows = []
    demand = 16_300*KW
    for generator_yield in [2, 5, 10]:
        for invest_bps in [0, 1000, 5000]:
            g = Grid(p)
            g.add_station("aggregate", 153_900*KW)
            raw = 153_900*KW
            cash = 98_908*ALPHA
            cumulative_issued = cumulative_generator_burn = 0
            for day in range(366):
                g.advance(day*p.blocks_per_day)
                if day % 14 == 0:
                    g.root_change("aggregate", raw)  # newly built supply waits for next paid cycle
                    cash = g.pay("aggregate", 1, cash)
                if day % 7 == 0 or day == 365:
                    rows.append({"generator_kw_per_alpha": generator_yield,
                                 "investment_bps_of_issuance": invest_bps, "day": day,
                                 "raw_supply_kw": raw/KW, "available_kw": g.shared("aggregate")/KW,
                                 "demand_kw": demand/KW, "supply_demand_ratio": g.shared("aggregate")/demand,
                                 "upkeep_burn_alpha": g.stations["aggregate"].burn/ALPHA,
                                 "generator_burn_alpha": cumulative_generator_burn/ALPHA,
                                 "issued_alpha": cumulative_issued/ALPHA,
                                 "liquid_alpha": cash/ALPHA})
                if day < 365:
                    issuance = 3000*ALPHA
                    investment = issuance*invest_bps//BPS
                    cash += issuance-investment
                    cumulative_issued += issuance
                    cumulative_generator_burn += investment
                    raw += investment*generator_yield  # mW per ualpha = kW per Alpha
    return rows


def supply_shock(p: Parameters) -> list[dict]:
    rows = []
    initial, giant, demand, essential = 153_900*KW, 141_500*KW, 16_300*KW, 3500*KW
    for replacement_days in [0, 7, 30]:
        remaining = initial-giant
        for paid in [False, True]:
            available = project(remaining, 0, BPS if paid else p.durable_bps, p)
            needed = max(0, ceil_div(demand*BPS, BPS if paid else p.durable_bps)-remaining)
            rows.append({"external_rescue_delay_days": replacement_days,
                         "upkeep_active": paid, "raw_remaining_kw": remaining/KW,
                         "available_kw": available/KW,
                         "demand_coverage": available/demand,
                         "equal_share_full_load_players_online_fraction": int(available >= demand),
                         "essential_load_can_fit": available >= essential,
                         "minimum_new_field_generator_burn_alpha": ceil_div(needed, 2)/ALPHA})
    return rows


def threshold_age(required: int, difficulty_range: int) -> int:
    lo, hi = 1, difficulty_range+1
    while lo < hi:
        mid = (lo+hi)//2
        if calculate_difficulty(mid, difficulty_range) <= required:
            hi = mid
        else:
            lo = mid+1
    return lo


def quantile(xs: list[float], q: float) -> float:
    return sorted(xs)[int(q*(len(xs)-1))]


def audit_races(p: Parameters, seed: int, trials: int, seconds: float) -> list[dict]:
    """Continuous Poisson approximation to uniform independent hashes.

    Rate is aggregate constant active hash rate. Sniper submits at first d<=5;
    no mempool delays, electricity prices or strategic opponents inferred.
    """
    rows = []
    for difficulty_range in [1387, 65380]:
        boundaries = [(threshold_age(d, difficulty_range)*seconds, d) for d in range(63, 4, -1)]
        free_time = threshold_age(5, difficulty_range)*seconds
        for rate in [1000, 2_000_000, 9_000_000]:
            for prepared_days in [0, 7, 29]:
                rng = random.Random(seed + difficulty_range*101 + rate + prepared_days*7919)
                times, payouts, winners = [], [], 0
                for _ in range(trials):
                    prepared = 0
                    if prepared_days:
                        hashes = prepared_days*p.blocks_per_day*seconds*rate
                        u = rng.random()
                        for d in range(1, 65):
                            if u < -math.expm1(-hashes/(16**d)):
                                prepared = d
                            else:
                                break
                    best_time = free_time
                    achieved = 5
                    if prepared >= 6:
                        best_time = threshold_age(prepared, difficulty_range)*seconds
                        achieved = prepared
                    hazard_left = -math.log(max(rng.random(), 1e-300))
                    for i, (start, required) in enumerate(boundaries):
                        end = boundaries[i+1][0] if i+1 < len(boundaries) else free_time
                        if start >= best_time:
                            break
                        end = min(end, best_time)
                        rate_success = rate/(16**required)
                        hazard = max(0, end-start)*rate_success
                        if hazard_left <= hazard:
                            best_time = start + hazard_left/rate_success
                            achieved = required
                            while achieved < 64 and rng.random() < 1/16:
                                achieved += 1
                            break
                        hazard_left -= hazard
                    winners += best_time < free_time
                    times.append(best_time/3600)
                    payouts.append((2500+min(4,max(0,achieved-5))*1875)/BPS)
                rows.append({"difficulty_range": difficulty_range, "hashes_per_second": rate,
                             "preparation_days": prepared_days, "trials": trials,
                             "p10_hours": quantile(times,.1), "median_hours": statistics.median(times),
                             "p90_hours": quantile(times,.9), "hasher_win_fraction": winners/trials,
                             "mean_reward_multiplier": statistics.mean(payouts),
                             "free_point_hours": free_time/3600})
    return rows


def reserve_drain(p: Parameters) -> list[dict]:
    rows = []
    for dust in [1, 1000, 10000, 50000]:
        for achieved in [5, 9]:
            for shed_bps in [100, 1000, 10000]:
                cfg = replace(p, dust_threshold_ualpha=dust)
                remaining, count, received = p.initial_collateral_ualpha, 0, 0
                while remaining and count < 100_000:
                    r = bounty(remaining, shed_bps, BPS, achieved, cfg)
                    received += r
                    remaining -= r
                    count += 1
                rows.append({"dust_ualpha": dust, "difficulty": achieved,
                             "shed_bps": shed_bps, "valid_audits_to_exhaust": count,
                             "total_reward_alpha": received/ALPHA,
                             "requires_new_real_imbalance_each_time": True})
    return rows


def emit_csv(path: Path, rows: list[dict]) -> None:
    with path.open("w", newline="") as f:
        writer = csv.DictWriter(f, fieldnames=list(rows[0]))
        writer.writeheader()
        writer.writerows(rows)


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--out", type=Path, default=Path("build/substation-model/substation-model-results-2026-09-25"))
    parser.add_argument("--parameters", type=Path, default=Path(__file__).with_name("parameters.json"))
    args = parser.parse_args()
    raw = json.loads(args.parameters.read_text())
    p = Parameters.read(args.parameters)
    ghost_rows, ghost_summary = ghost_power(p)
    results = {
        "parameters": raw,
        "source_sha256": {name: hashlib.sha256(Path(__file__).with_name(name).read_bytes()).hexdigest()
                          for name in ["model.py", "experiments.py", "parameters.json"]},
        "upkeep_search": upkeep_search(p), "prepayment": prepayment(p),
        "oldest_first": oldest_attack(p), "ghost_power": ghost_rows,
        "ghost_summary": ghost_summary, "loop_control": loop_control(p),
        "fragmentation": fragmentation(p), "treasury": treasury(p),
        "glut": glut(p), "supply_shock": supply_shock(p),
        "audit_races": audit_races(p, raw["seed"], raw["audit_trials"], raw["nominal_seconds_per_block"]),
        "reserve_drain": reserve_drain(p)}
    args.out.mkdir(parents=True, exist_ok=True)
    (args.out/"results.json").write_text(json.dumps(results, indent=2, sort_keys=True)+"\n")
    for key, values in results.items():
        if isinstance(values, list) and values:
            emit_csv(args.out/f"{key}.csv", values)
    print(json.dumps({"out": str(args.out), "scenario_rows": {
        k:len(v) for k,v in results.items() if isinstance(v,list)},
        "ghost_summary": ghost_summary, "loop_control": results["loop_control"]}, indent=2))


if __name__ == "__main__":
    main()
