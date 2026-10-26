"""Separate customer-count and outgoing-capacity pricing experiments."""
import argparse
import hashlib
import json
from pathlib import Path

from experiments import emit_csv
from model import ALPHA, KW, MW, Parameters
from service_pricing_model import ServicePricingGrid

WALLET = 10**25


def setup(p, count, outgoing, base=ALPHA, player=ALPHA//20):
    g = ServicePricingGrid(p, base, player)
    s = g.add_station("A", 100*MW)
    g.add_station("B")
    for i in range(count):
        g.move_player(str(i), "A")
    edge = g.connect("A", "B", outgoing) if outgoing else None
    g.advance(7*p.blocks_per_day)
    return g, s, edge


def attacks(p):
    rows = []
    for base in [0, ALPHA, 10*ALPHA]:
        for per_player in [ALPHA//100, ALPHA//20, ALPHA]:
            for count in [1, 10, 100, 1000]:
                for outgoing in [0, MW, 60*MW]:
                    for days in [1, 7]:
                        for strategy in ["cut_input", "disconnect_players", "shrink_dynamic", "both"]:
                            g, s, edge = setup(p, count, outgoing, base, per_player)
                            normal = g.quote("A", days)
                            if strategy == "cut_input":
                                g.root_change("A", KW)
                            if strategy in ["disconnect_players", "both"]:
                                for i in range(count):
                                    g.move_player(str(i), None)
                            if edge and strategy in ["shrink_dynamic", "both"]:
                                g.resize_dynamic(edge, KW)
                            g.pay("A", days, WALLET)
                            if strategy == "cut_input":
                                g.root_change("A", 100*MW)
                            if strategy in ["disconnect_players", "both"]:
                                for i in range(count):
                                    g.move_player(str(i), "A")
                            if edge and strategy in ["shrink_dynamic", "both"]:
                                g.resize_dynamic(edge, outgoing)
                            g.cascade()
                            assert g.connection_counts["A"] == count
                            assert g.load("A") == outgoing
                            assert edge is None or edge.index in s.outgoing
                            rows.append({"base_alpha": base/ALPHA, "per_player_alpha": per_player/ALPHA,
                                         "players": count, "outgoing_mw": outgoing/MW,
                                         "days": days, "strategy": strategy,
                                         "normal_alpha": normal/ALPHA, "paid_alpha": s.burn/ALPHA,
                                         "saving_alpha": (normal-s.burn)/ALPHA,
                                         "saving_fraction": (normal-s.burn)/normal,
                                         "final_player_capacity_kw": g.player_capacity("0")/KW,
                                         "connection_mutations": 2*count if strategy in ["disconnect_players", "both"] else 0,
                                         "dynamic_mutations": 2 if edge and strategy in ["shrink_dynamic", "both"] else 0})
    return rows


def pricing(p):
    rows = []
    for count in [0, 1, 10, 100, 1000]:
        for outgoing in [0, KW, MW, 10*MW, 60*MW]:
            g, _, _ = setup(p, count, outgoing)
            for input_power in [KW, MW, 100*MW, 1000*MW]:
                g.root_change("A", input_power)
                rows.append({"players": count, "outgoing_mw": outgoing/MW,
                             "input_mw": input_power/MW, "price_alpha": g.quote("A")/ALPHA})
    return rows


def partitions(p):
    rows = []
    # Same 100 customers and 100 MW total supply divided in parallel. No outgoing
    # allocations in this comparison, so power-per-player stays 1 MW raw.
    for count in [1, 2, 5, 10, 20, 100]:
        g = ServicePricingGrid(p)
        for i in range(count):
            g.add_station(str(i), 100*MW//count)
        for i in range(100):
            g.move_player(str(i), str(i%count))
        total = sum(g.quote(str(i)) for i in range(count))
        rows.append({"stations": count, "total_players": 100,
                     "total_quote_alpha": total/ALPHA,
                     "player_component_alpha": 5,
                     "base_component_alpha": count})
    return rows


def power_per_player(p):
    rows = []
    for count in [1, 10, 100]:
        for raw in [MW, 10*MW, 100*MW, 1000*MW]:
            g = ServicePricingGrid(p)
            g.add_station("A", raw)
            for i in range(count):
                g.move_player(str(i), "A")
            rows.append({"players": count, "raw_mw": raw/MW,
                         "fresh_unpaid_kw_per_player": g.player_capacity("0")/KW,
                         "quote_alpha": g.quote("A")/ALPHA})
    return rows


def outgoing_order(p):
    rows = []
    for immediate_cascade in [False, True]:
        # Admit the downstream contract while both stations are still fresh.
        g = ServicePricingGrid(p)
        s = g.add_station("A", 100*MW)
        g.add_station("B")
        g.add_station("C")
        for i in range(100):
            g.move_player(str(i), "A")
        edge = g.connect("A", "B", 60*MW)
        downstream = g.connect("B", "C", 30*MW, kind="static")
        g.advance(p.blocks_per_day)
        g.resize_dynamic(edge, KW)
        if immediate_cascade:
            g.cascade()
        g.pay("A", 1, WALLET)
        g.resize_dynamic(edge, 60*MW)
        g.cascade()
        rows.append({"cascade_before_restore": immediate_cascade,
                     "burn_alpha": s.burn/ALPHA,
                     "upstream_id_preserved": edge.index in s.outgoing,
                     "downstream_contract_survives": downstream.index in g.stations["B"].outgoing})
    return rows


def retention_fees(p):
    # Diagnostic only, not a proposed policy: a one-day reconnection charge does
    # not cover a week of service purchased while empty.
    rows = []
    for n in [1, 100, 1000]:
        for days in [1, 7]:
            g, _, _ = setup(p, n, 0)
            regular = g.quote("A", days)
            empty_purchase = g.base_fee*days
            reconnect = n*g.player_fee
            rows.append({"players": n, "purchased_days": days,
                         "normal_alpha": regular/ALPHA,
                         "empty_purchase_plus_one_day_reconnect_fee_alpha": (empty_purchase+reconnect)/ALPHA,
                         "remaining_saving_alpha": (regular-empty_purchase-reconnect)/ALPHA})
    return rows


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--out", type=Path,
                        default=Path("build/substation-model/substation-service-pricing-results-2026-09-25"))
    parser.add_argument("--parameters", type=Path,
                        default=Path(__file__).with_name("parameters.json"))
    args = parser.parse_args()
    p = Parameters.read(args.parameters)
    result = {"base_parameters": json.loads(args.parameters.read_text()),
              "service_pricing": {"base_ualpha_per_day": ALPHA, "per_player_ualpha_per_day": ALPHA//20,
                                  "outgoing_tariff": "existing candidate marginal tiers, applied to aggregate outgoing power",
                                  "prices_not_approved": True, "original_health_and_prepaid_duration_retained": True,
                                  "no_live_chain_execution": True},
              "source_sha256": {name: hashlib.sha256(Path(__file__).with_name(name).read_bytes()).hexdigest()
                                for name in ["model.py", "experiments.py", "service_pricing_model.py",
                                             "service_pricing_experiments.py", "test_service_pricing_model.py"]},
              "parameter_sha256": hashlib.sha256(args.parameters.read_bytes()).hexdigest(),
              "attacks": attacks(p), "pricing": pricing(p), "partitions": partitions(p),
              "power_per_player": power_per_player(p), "outgoing_order": outgoing_order(p),
              "reconnection_fee_counterfactual": retention_fees(p)}
    args.out.mkdir(parents=True, exist_ok=True)
    (args.out/"results.json").write_text(json.dumps(result, indent=2, sort_keys=True)+"\n")
    for key, value in result.items():
        if isinstance(value, list):
            emit_csv(args.out/f"{key}.csv", value)
    selected = [x for x in result["attacks"] if x["base_alpha"] == 1 and x["per_player_alpha"] == .05
                and x["players"] == 100 and x["days"] == 1 and x["outgoing_mw"] in [0, 60]]
    print(json.dumps({"rows": {k: len(v) for k, v in result.items() if isinstance(v, list)},
                      "selected": selected, "outgoing_order": result["outgoing_order"]}, indent=2))


if __name__ == "__main__":
    main()
