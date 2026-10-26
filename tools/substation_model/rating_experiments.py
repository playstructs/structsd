"""Reproducible adversarial rating experiments; standard library only."""
import argparse
import hashlib
import json
from pathlib import Path

from experiments import emit_csv, upkeep_search
from model import ALPHA, KW, MW, Parameters, upkeep_price
from rating_model import RatingGrid

WALLET = 10**25


def attacks(p):
    rows = []
    day = p.blocks_per_day
    for small in [1, KW, 100*KW, MW, 10*MW]:
        for age in [0, 7, 29, 31]:
            for days in [1, 3, 7]:
                for delay in [0, day//2, day-1, day, 7*day, 29*day]:
                    for split in [False, True]:
                        g = RatingGrid(p)
                        s = g.seed_station("A", 100*MW)
                        g.advance(age*day)
                        g.set_rating("A", small, WALLET)
                        g.pay("A", days, WALLET)
                        g.advance(g.height+delay)
                        steps = [small+(100*MW-small)*i//10 for i in range(1, 11)] if split else [100*MW]
                        for target in steps:
                            g.set_rating("A", target, WALLET)
                        # Rational lower bound, independently expressed via the
                        # full whole-day price (exact at this 100 MW target).
                        duration = max(day, days*day-delay)
                        lower_numerator = upkeep_price(100*MW, 1, p)*duration
                        assert s.burn*day >= lower_numerator
                        assert s.last_upkeep == age*day
                        rows.append({"small_rating_kw": small/KW, "age_days": age,
                                     "purchased_days": days, "delay_blocks": delay,
                                     "split_expansion": split, "burn_ualpha": s.burn,
                                     "lower_bound_alpha": lower_numerator/day/ALPHA,
                                     "output_kw": g.throughput("A")/KW,
                                     "health_reset_by_expansion": s.last_upkeep != age*day})
    return rows


def scenarios(p):
    rows = []

    def record(label, g):
        s = g.stations["A"]
        rows.append({"case": label, "burn_alpha": s.burn/ALPHA,
                     "output_mw": g.throughput("A")/MW,
                     "rating_mw": s.rating/MW,
                     "ceiling_mw": s.serviced_ceiling/MW,
                     "age_days": (g.height-s.last_upkeep)/p.blocks_per_day,
                     "prepaid_days_left": max(0, s.paid_until-g.height)/p.blocks_per_day})

    g = RatingGrid(p)
    g.seed_station("A", 100*MW)
    g.advance(29*p.blocks_per_day)
    g.root_change("A", KW)
    g.pay("A", 1, WALLET)
    g.root_change("A", 100*MW)
    record("disconnect_then_restore_supply", g)

    g = RatingGrid(p)
    g.seed_station("A", 100*MW)
    g.advance(29*p.blocks_per_day)
    g.pay("A", 1, WALLET, rating=KW)
    record("cheap_small_rating_before_expansion", g)
    g.set_rating("A", 100*MW, WALLET)
    record("cheap_small_rating_after_expansion", g)

    for delay in [0, 1, 6, 7, 29, 31]:
        g = RatingGrid(p)
        g.add_station("A", 100*MW)
        g.pay("A", 7, WALLET, rating=KW)
        g.advance(delay*p.blocks_per_day)
        g.set_rating("A", 100*MW, WALLET)
        record(f"seven_day_small_purchase_expand_day_{delay}", g)

    g = RatingGrid(p)
    g.add_station("A", 100*MW)
    g.pay("A", 6, WALLET, rating=KW)
    g.pay("A", 1, WALLET, rating=100*MW)
    record("renew_and_expand_with_six_days_carried", g)

    g = RatingGrid(p)
    g.seed_station("A", 100*MW)
    g.pay("A", 6, WALLET)
    g.pay("A", 1, WALLET, rating=KW)
    g.set_rating("A", 100*MW, WALLET)
    record("forfeit_large_prepaid_then_restore", g)

    g = RatingGrid(p)
    g.add_station("A", 100*MW)
    record("replacement_before_commissioning", g)
    g.pay("A", 1, WALLET, rating=100*MW)
    record("replacement_after_commissioning", g)
    return rows


def economics(p):
    rows = []
    for investment_bps in [0, 1000, 5000]:
        g = RatingGrid(p)
        s = g.seed_station("A", 153900*KW)
        raw = s.root
        wallet = 98908*ALPHA
        issued = invested = 0
        for day in range(366):
            g.advance(day*p.blocks_per_day)
            if day % 14 == 0:
                g.root_change("A", raw)
                # Rate changes occur atomically with the full-size renewal.
                wallet = g.pay("A", 1, wallet, rating=raw)
            if day == 365:
                assert wallet+s.burn+invested == 98908*ALPHA+issued
                rows.append({"investment_bps": investment_bps,
                             "day": day, "raw_mw": raw/MW,
                             "available_demand_ratio": g.throughput("A")/(16300*KW),
                             "upkeep_burn_alpha": s.burn/ALPHA,
                             "generator_burn_alpha": invested/ALPHA,
                             "liquid_alpha": wallet/ALPHA})
            if day < 365:
                income = 3000*ALPHA
                spend = income*investment_bps//10000
                issued += income
                invested += spend
                wallet += income-spend
                raw += spend*2
    return rows


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--out", type=Path,
                        default=Path("build/substation-model/substation-rating-results-2026-09-25"))
    parser.add_argument("--parameters", type=Path,
                        default=Path(__file__).with_name("parameters.json"))
    args = parser.parse_args()
    p = Parameters.read(args.parameters)
    search = upkeep_search(p)
    viable = [row for row in search if row["min_service_ratio"] >= 1]
    results = {"parameters": json.loads(args.parameters.read_text()),
               "source_sha256": {name: hashlib.sha256(Path(__file__).with_name(name).read_bytes()).hexdigest()
                                 for name in ["model.py", "rating_model.py", "experiments.py",
                                              "rating_experiments.py", "test_rating_model.py"]},
               "parameter_sha256": hashlib.sha256(args.parameters.read_bytes()).hexdigest(),
               "attacks": attacks(p), "scenarios": scenarios(p),
               "economics": economics(p), "upkeep_search": search,
               "cheapest_full_service": min(viable, key=lambda row: row["alpha_per_day"])}
    args.out.mkdir(parents=True, exist_ok=True)
    (args.out/"results.json").write_text(json.dumps(results, indent=2, sort_keys=True)+"\n")
    for key, value in results.items():
        if isinstance(value, list):
            emit_csv(args.out/f"{key}.csv", value)
    print(json.dumps({"rows": {k: len(v) for k, v in results.items() if isinstance(v, list)},
                      "scenarios": results["scenarios"], "economics": results["economics"],
                      "cheapest_full_service": results["cheapest_full_service"]}, indent=2))


if __name__ == "__main__":
    main()
