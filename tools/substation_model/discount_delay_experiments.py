"""Adversarial delay tests, including standby rotation and deferred cascade."""
import argparse
import hashlib
import json
from pathlib import Path

from discount_delay_model import DiscountDelayGrid
from experiments import emit_csv
from model import ALPHA, KW, MW, Parameters, upkeep_price, project

WALLET = 10**25


def single_station(p):
    rows = []
    for delay in [1, 7, 14, 30]:
        for low in [KW, 10*MW, 32_600*KW, 50*MW]:
            for age in [0, 7, 14]:
                for wait in [0, delay*p.blocks_per_day-1, delay*p.blocks_per_day]:
                    g = DiscountDelayGrid(p, delay)
                    s = g.add_station("A", 100*MW)
                    g.advance(age*p.blocks_per_day)
                    g.root_change("A", low)
                    g.advance(g.height+wait)
                    minimum = g.throughput("A")
                    quote = g.quote_supply("A")
                    g.pay("A", 1, WALLET)
                    g.root_change("A", 100*MW)
                    rows.append({"delay_days": delay, "low_supply_kw": low/KW,
                                 "initial_age_days": age, "wait_blocks": wait,
                                 "quoted_kw": quote/KW, "burn_alpha": s.burn/ALPHA,
                                 "minimum_low_period_output_kw": minimum/KW,
                                 "still_serves_16_3mw": minimum >= 16300*KW,
                                 "restored_output_kw": g.throughput("A")/KW,
                                 "discount_fraction": 1-s.burn/upkeep_price(100*MW, 1, p)})
    return rows


def outgoing(p):
    rows = []
    for guard in [False, True]:
        for wait in [0, 7]:
            g = DiscountDelayGrid(p)
            s = g.add_station("A", 100*MW)
            g.add_station("B")
            edge = g.connect("A", "B", 60*MW)
            g.advance(p.blocks_per_day)
            g.root_change("A", KW)
            if wait:
                g.cascade()  # Actually exposed to low supply between blocks.
                g.advance(g.height+wait*p.blocks_per_day)
            if not guard:
                # Counterfactual old supply quote, bypass only the proposed guard.
                g.discounts["A"].protected_supply = KW
            g.pay("A", 1, WALLET)
            g.root_change("A", 100*MW)
            g.cascade()
            rows.append({"guard": guard, "wait_days": wait,
                         "burn_alpha": s.burn/ALPHA,
                         "outgoing_survives": edge.index in s.outgoing})
    return rows


def rotations(p):
    rows = []
    for delay in [1, 7, 14, 21, 30]:
        for count in [1, 2, 3, 4, 5]:
            for interval in [7, 14, 21]:
                for demand in [16300*KW, 50*MW]:
                    g = DiscountDelayGrid(p, delay)
                    for i in range(count):
                        g.add_station(str(i), 100*MW if i == 0 else KW)
                    active = 0
                    payments = []
                    minimum = g.shared("0")
                    # Compare the same number of completed service intervals.
                    intervals = 26
                    for cycle in range(1, intervals+1):
                        height = cycle*interval*p.blocks_per_day
                        g.advance(height-1)
                        minimum = min(minimum, g.shared(str(active)))
                        g.advance(height)
                        target = (active+1)%count
                        if target == active:
                            g.root_change(str(active), KW)
                        s = g.stations[str(target)]
                        before = s.burn
                        g.pay(str(target), 1, WALLET)
                        payments.append(s.burn-before)
                        g.root_change(str(active), KW)
                        g.root_change(str(target), 100*MW)
                        active = target  # Authorized player's connection follows input.
                        g.cascade()
                        minimum = min(minimum, g.shared(str(active)))
                        assert sum(s.root for s in g.stations.values()) == 100*MW+(count-1)*KW
                    steady = payments[2*count:]
                    rows.append({"delay_days": delay, "stations": count,
                                 "switch_interval_days": interval, "demand_mw": demand/MW,
                                 "idle_days_before_reuse": (count-1)*interval,
                                 "minimum_output_mw": minimum/MW,
                                 "full_service_throughout": minimum >= demand,
                                 "payments": len(payments), "burn_alpha": sum(payments)/ALPHA,
                                 "steady_average_payment_alpha": sum(steady)/len(steady)/ALPHA,
                                 "steady_full_discount": all(x == 9500 for x in steady),
                                 "extra_idle_input_kw": count-1,
                                 "normal_same_cadence_burn_alpha": intervals*upkeep_price(100*MW, 1, p)/ALPHA})
    return rows


def jitter(p):
    rows = []
    for restart in [False, True]:
        g = DiscountDelayGrid(p, restart_on_change=restart)
        g.add_station("A", 100*MW)
        g.root_change("A", 99*MW)
        g.advance(7*p.blocks_per_day-1)
        g.root_change("A", KW)
        g.advance(7*p.blocks_per_day)
        rows.append({"case": "late_deep_cut", "restart_timer": restart,
                     "quoted_kw": g.quote_supply("A")/KW})
        g = DiscountDelayGrid(p, restart_on_change=restart)
        g.add_station("A", 100*MW)
        g.root_change("A", KW)
        for day in range(1, 30):
            g.advance(day*p.blocks_per_day)
            g.root_change("A", KW+day%2)
        rows.append({"case": "daily_one_mw_unit_jitter", "restart_timer": restart,
                     "quoted_kw": g.quote_supply("A")/KW})
    return rows


def economics(p):
    savings = upkeep_price(100*MW, 1, p)-upkeep_price(KW, 1, p)
    rows = []
    for delay in [1, 7, 14, 30]:
        for earnings in [0, 1, 10, 50, 100, 1000]:
            rows.append({"delay_days": delay, "lost_earnings_alpha_per_day": earnings,
                         "saved_per_purchase_alpha": savings/ALPHA,
                         "hypothetical_earnings_lost_alpha": delay*earnings,
                         "net_alpha_per_purchase": savings/ALPHA-delay*earnings,
                         "breakeven_lost_earnings_per_day": savings/ALPHA/delay})
    return rows


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--out", type=Path,
                        default=Path("build/substation-model/substation-discount-delay-results-2026-09-25"))
    parser.add_argument("--parameters", type=Path,
                        default=Path(__file__).with_name("parameters.json"))
    args = parser.parse_args()
    p = Parameters.read(args.parameters)
    result = {"base_parameters": json.loads(args.parameters.read_text()),
              "assumptions": {"default_delay_days": 7, "restart_on_every_accepted_supply_change": True,
                              "temporary_bonus_limited_to_paid_capacity": True,
                              "whole_station_health_reset": True,
                              "seeded_existing_stations": True,
                              "rotation_moves_authorized_player_with_supply_atomically": True,
                              "production_chain_execution_or_gas_measured": False},
              "source_sha256": {name: hashlib.sha256(Path(__file__).with_name(name).read_bytes()).hexdigest()
                                for name in ["model.py", "experiments.py", "discount_delay_model.py",
                                             "discount_delay_experiments.py", "test_discount_delay_model.py"]},
              "parameter_sha256": hashlib.sha256(args.parameters.read_bytes()).hexdigest(),
              "single_station": single_station(p), "outgoing": outgoing(p),
              "rotations": rotations(p), "jitter": jitter(p), "economics": economics(p)}
    args.out.mkdir(parents=True, exist_ok=True)
    (args.out/"results.json").write_text(json.dumps(result, indent=2, sort_keys=True)+"\n")
    for key, value in result.items():
        if isinstance(value, list):
            emit_csv(args.out/f"{key}.csv", value)
    selected = [x for x in result["rotations"] if x["delay_days"] in [7, 30]
                and x["switch_interval_days"] == 14 and x["demand_mw"] == 50
                and x["stations"] in [1, 2, 4]]
    print(json.dumps({"rows": {k: len(v) for k, v in result.items() if isinstance(v, list)},
                      "outgoing": result["outgoing"], "rotations": selected,
                      "jitter": result["jitter"]}, indent=2))


if __name__ == "__main__":
    main()
