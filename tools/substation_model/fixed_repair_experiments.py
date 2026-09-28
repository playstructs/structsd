"""Fixed repair price sensitivity, failure scenarios and reproducible outputs."""
import argparse
from dataclasses import replace
import hashlib
import json
from pathlib import Path

from experiments import emit_csv
from fixed_repair_model import FixedRepairGrid, first_offline_block
from model import ALPHA, KW, MW, Parameters, project

PRICES = [1, 10, 50, 100, 500]


def cadence(p):
    rows = []
    demand = 16300*KW
    for efficiency in [7000, 10000]:
        q = replace(p, durable_bps=efficiency)
        for supply_kw in [16300, 23286, 24450, 29600, 32600, 48900, 81500, 153900, 1630000]:
            supply = supply_kw*KW
            interval = first_offline_block(supply, demand, q)
            for price in PRICES:
                rows.append({"baseline_bps": efficiency, "supply_kw": supply_kw,
                             "demand_kw": demand/KW, "repair_alpha": price,
                             "service_possible": bool(interval),
                             "latest_repair_days": interval/q.blocks_per_day,
                             "long_run_alpha_per_day": price*q.blocks_per_day/interval if interval else None,
                             "fraction_of_3000_issuance": price*q.blocks_per_day/interval/3000 if interval else None})
    return rows


def partition(p):
    rows = []
    total, demand = 100*MW, 50*MW
    for n in [1, 2, 5, 10, 50, 100, 1000]:
        interval = first_offline_block(total//n, demand//n, p)
        for price in PRICES:
            rows.append({"parallel_stations": n, "repair_alpha": price,
                         "raw_kw_per_station": total/n/KW,
                         "repair_interval_days": interval/p.blocks_per_day,
                         "network_alpha_per_day": n*price*p.blocks_per_day/interval,
                         "capacity_lost_if_one_station_fails_pct": 100/n})
    return rows


def creation(p):
    rows = []
    founding = ALPHA//2
    for price in PRICES:
        for replacements in [1, 10, 100]:
            # Existing collateral assumed fully recoverable: favorable lower bound
            # for attackers. No refund/deletion mechanic is implemented here.
            rows.append({"repair_alpha": price, "health_resets": replacements,
                         "normal_repairs_burn_alpha": replacements*price,
                         "unsafe_replacement_burn_alpha": replacements*founding/ALPHA,
                         "commissioned_replacement_burn_alpha": replacements*(price+founding/ALPHA),
                         "collateral_needed_per_live_station_alpha": p.initial_collateral_ualpha/ALPHA})
    return rows


def treasury(p):
    rows = []
    ticks, horizon = 96, 90
    demand, essential = 16300*KW, 3500*KW
    for price in PRICES:
        for income_bps in [0, 1, 10, 25, 50, 100]:
            for fallback in [False, True]:
                for prefunded in [False, True]:
                    g = FixedRepairGrid(p, price*ALPHA)
                    s = g.seed_station("A", 29600*KW)
                    wallet = price*ALPHA if prefunded else 0
                    initial = wallet
                    repairs = online = earning = 0
                    credited = remainder = 0
                    first_failure = None
                    for tick in range(horizon*ticks):
                        g.advance(tick*p.blocks_per_day//ticks)
                        next_height = (tick+1)*p.blocks_per_day//ticks
                        next_output = project(s.root, next_height-s.last_upkeep,
                                              p.durable_bps, p)
                        if next_output < demand and wallet >= g.repair_price:
                            wallet = g.repair("A", wallet)
                            repairs += 1
                        available = g.throughput("A")
                        full = available >= demand
                        earns = full or (fallback and available >= essential)
                        online += full
                        earning += earns
                        if not full and first_failure is None:
                            first_failure = g.height/p.blocks_per_day
                        if earns:
                            # Carry sub-ualpha remainder, avoiding artificial income loss.
                            remainder += 3000*ALPHA*income_bps
                            coins, remainder = divmod(remainder, 10000*ticks)
                            wallet += coins
                            credited += coins
                    assert wallet+s.burn == initial+credited
                    rows.append({"repair_alpha": price, "treasury_income_bps": income_bps,
                                 "essential_fallback": fallback, "one_repair_prefunded": prefunded,
                                 "repairs": repairs, "full_online_fraction": online/(horizon*ticks),
                                 "earning_fraction": earning/(horizon*ticks),
                                 "first_failure_day": first_failure,
                                 "final_wallet_alpha": wallet/ALPHA,
                                 "final_output_kw": g.throughput("A")/KW})
    return rows


def glut(p):
    rows = []
    for price in PRICES:
        for investment_bps in [0, 1000, 5000]:
            g = FixedRepairGrid(p, price*ALPHA)
            s = g.seed_station("A", 153900*KW)
            wallet = 98908*ALPHA
            issued = invested = 0
            for day in range(366):
                g.advance(day*p.blocks_per_day)
                # Keep the previous comparison's 14-day repair cadence; extra
                # supply can now be connected immediately without any surcharge.
                if day and day % 14 == 0:
                    wallet = g.repair("A", wallet)
                if day % 7 == 0 or day == 365:
                    rows.append({"repair_alpha": price, "investment_bps": investment_bps,
                                 "day": day, "raw_mw": s.root/MW,
                                 "available_demand_ratio": g.throughput("A")/(16300*KW),
                                 "repair_burn_alpha": s.burn/ALPHA,
                                 "generator_burn_alpha": invested/ALPHA,
                                 "liquid_alpha": wallet/ALPHA})
                assert wallet+s.burn+invested == 98908*ALPHA+issued
                if day < 365:
                    income = 3000*ALPHA
                    spend = income*investment_bps//10000
                    issued += income
                    invested += spend
                    wallet += income-spend
                    g.root_change("A", s.root+spend*2)
    return rows


def routing(p):
    rows = []
    for depth in [1, 2, 3, 5, 10]:
        g = FixedRepairGrid(p)
        for i in range(depth):
            g.seed_station(str(i), 100*MW if i == 0 else 0)
            if i:
                # Full safe commitment, held until an actual audit/mutation.
                g.connect(str(i-1), str(i), g.safe(str(i-1)), automated=True)
        for age in [0, 7, 15, 30]:
            g.advance(age*p.blocks_per_day)
            rows.append({"stations_in_series": depth, "age_days": age,
                         "leaf_output_mw": g.throughput(str(depth-1))/MW,
                         "total_synchronized_repair_alpha": depth*g.repair_price/ALPHA,
                         "automatic_queue_length": len(g.queue)})
    return rows


def enforcement(p):
    g = FixedRepairGrid(p)
    g.seed_station("relay", 100*MW)
    g.seed_station("leaf")
    e = g.connect("relay", "leaf", 60*MW, automated=True)
    g.advance(31*p.blocks_per_day)
    g.repair("leaf", 100*ALPHA)
    before = {"relay_mw": g.throughput("relay")/MW,
              "leaf_mw": g.throughput("leaf")/MW, "queue": len(g.queue)}
    reward = g.audit("relay", 9)
    return {"before_audit": before, "after_audit_leaf_mw": g.throughput("leaf")/MW,
            "after_audit_edge_mw": e.power/MW, "reward_alpha": reward["reward_ualpha"]/ALPHA}


def calibration(p):
    rows = []
    for count in [1, 2, 10, 50, 100, 1000]:
        for days in [7, 14, 28]:
            for target_bps in [200, 1000]:
                rows.append({"active_stations": count, "interval_days": days,
                             "target_issuance_bps": target_bps,
                             "required_fee_alpha": 3000*target_bps/10000*days/count})
    return rows


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--out", type=Path,
                        default=Path("build/substation-model/substation-fixed-repair-results-2026-09-25"))
    parser.add_argument("--parameters", type=Path,
                        default=Path(__file__).with_name("parameters.json"))
    args = parser.parse_args()
    p = Parameters.read(args.parameters)
    results = {"base_parameters": json.loads(args.parameters.read_text()),
               "alternative": {"repair_prices_alpha": PRICES, "default_repair_alpha": 10,
                               "founding_burn_alpha": 0.5, "temporary_bonus": False,
                               "baseline_bps": p.durable_bps, "rating": False,
                               "normal_creation_includes_first_repair_charge": True,
                               "all_numbers_are_candidate_inputs": True},
               "source_sha256": {name: hashlib.sha256(Path(__file__).with_name(name).read_bytes()).hexdigest()
                                 for name in ["model.py", "experiments.py", "fixed_repair_model.py",
                                              "fixed_repair_experiments.py", "test_fixed_repair_model.py"]},
               "parameter_sha256": hashlib.sha256(args.parameters.read_bytes()).hexdigest(),
               "cadence": cadence(p), "partition": partition(p), "creation": creation(p),
               "treasury": treasury(p), "glut": glut(p), "routing": routing(p),
               "enforcement": enforcement(p), "calibration": calibration(p)}
    args.out.mkdir(parents=True, exist_ok=True)
    (args.out/"results.json").write_text(json.dumps(results, indent=2, sort_keys=True)+"\n")
    for key, value in results.items():
        if isinstance(value, list):
            emit_csv(args.out/f"{key}.csv", value)
    print(json.dumps({"rows": {k: len(v) for k, v in results.items() if isinstance(v, list)},
                      "enforcement": results["enforcement"]}, indent=2))


if __name__ == "__main__":
    main()
