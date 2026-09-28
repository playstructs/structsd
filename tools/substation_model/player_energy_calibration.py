"""Read branch definitions and calculate legal player loadouts and fee scenarios.

This is an offline calibration, not a chain-state query or production implementation.
"""
import argparse
import hashlib
import itertools
import json
from pathlib import Path
import re

from model import ALPHA, KW, Parameters, ceil_div, health_fraction
from experiments import emit_csv
from service_coverage_quote import ServiceCoverage

ROOT = Path(__file__).resolve().parents[2]
TYPE_SOURCE = ROOT/'x/structs/types/genesis_struct_type.go'
KEY_SOURCE = ROOT/'x/structs/types/keys.go'
AMBITS = ('water', 'land', 'air', 'space')
FLAGS = (2, 4, 8, 16)


def definitions():
    text = TYPE_SOURCE.read_text()
    rows = []
    for block in re.findall(r'structType\s*=\s*StructType\s*\{(.*?)\n\s*}', text, re.S):
        fields = dict(re.findall(r'^\s*(\w+)\s*:\s*([^,\n]+)', block, re.M))
        rows.append({"id": int(fields['Id']), "name": fields['Type'].strip('"'),
                     "category": fields['Category'].removeprefix('ObjectType_'),
                     "limit": int(fields['BuildLimit']), "build_mw": int(fields['BuildDraw']),
                     "passive_mw": int(fields['PassiveDraw']), "ambits": int(fields['PossibleAmbit']),
                     "generation_rate": int(fields['GeneratingRate'])})
    if len(rows) != 22:
        raise ValueError("type inventory changed: review calibration assumptions")
    keys = KEY_SOURCE.read_text()
    base = int(re.search(r'PlayerPassiveDraw\s*=\s*(\d+)', keys).group(1))
    slots = int(re.search(r'PlanetStartingSlots\s*=\s*(\d+)', keys).group(1))
    return rows, base, slots


def max_loadout(types, slots, required=frozenset()):
    """Exact dynamic program over four ambit occupancies and per-type limits.

    Each type is processed once, so its quantity limit applies globally across
    ambits. The command ship is a separate fleet field and excluded by the caller.
    """
    dp = {(0, 0, 0, 0): (0, ())}
    for t in types:
        choices = itertools.product(*(range(slots+1) if t['ambits'] & flag else [0] for flag in FLAGS))
        choices = [v for v in choices if sum(v) >= (1 if t['id'] in required else 0)
                   and (not t['limit'] or sum(v) <= t['limit'])]
        nxt = {}
        for used, (score, layout) in dp.items():
            for choice in choices:
                total = tuple(a+b for a, b in zip(used, choice))
                if max(total) > slots:
                    continue
                value = score+sum(choice)*t['passive_mw']
                if total not in nxt or value > nxt[total][0]:
                    addition = tuple((t['id'], ambit, n) for ambit, n in zip(AMBITS, choice) if n)
                    nxt[total] = value, layout+addition
        dp = nxt
    if not dp:
        raise ValueError("required loadout cannot fit")
    used, (power, layout) = max(dp.items(), key=lambda item: (item[1][0], sum(item[0])))
    return power, used, layout


def profiles(types, base, slots):
    by_id = {t['id']: t for t in types}
    fleet = [t for t in types if t['category'] == 'fleet' and t['id'] != 1]
    fp, fu, fl = max_loadout(fleet, slots)
    command = by_id[1]['passive_mw']
    rows = []

    def add(name, layout):
        draw = base+sum(by_id[i]['passive_mw']*n for i, ambit, n in layout)
        rows.append({"profile": name, "gross_kw": draw/KW, "gross_mw": draw,
                     "struct_count": sum(n for i, ambit, n in layout),
                     "layout": [{"type_id": i, "name": by_id[i]['name'], "ambit": ambit, "count": n}
                                for i, ambit, n in layout]})

    add('command_only', ((1, 'command', 1),))
    add('command_extractor_refinery', ((1, 'command', 1), (14, 'water', 1), (15, 'water', 1)))
    add('maximum_fleet_only', ((1, 'command', 1),)+fl)
    for name, excluded, all_required in [
            ('developed_without_generators', {20, 21, 22}, True),
            ('developed_with_field_generator', {21, 22}, True),
            ('developed_with_continental_plant', {22}, True),
            ('every_planet_type_plus_maximum_fleet', set(), True),
            ('absolute_slot_limited_draw_maximum', set(), False)]:
        planet = [t for t in types if t['category'] == 'planet' and t['id'] not in excluded]
        required = {t['id'] for t in planet} if all_required else set()
        pp, pu, pl = max_loadout(planet, slots, required)
        add(name, ((1, 'command', 1),)+fl+pl)
    planet = [t for t in types if t['category'] == 'planet']
    _, _, all_fleet = max_loadout(fleet, slots, {t['id'] for t in fleet})
    _, _, all_planet = max_loadout(planet, slots, {t['id'] for t in planet})
    add('every_type_present_fill_remaining_slots', ((1, 'command', 1),)+all_fleet+all_planet)
    return rows


def capacity_targets(p):
    rows = []
    for target_kw in [8.215, 10, 116.715, 120]:
        target = round(target_kw*KW)
        for days in [0, 7, 14, 21]:
            n, d = health_fraction(days*p.blocks_per_day, p)
            raw = ceil_div(target*10000*d, p.durable_bps*n)
            rows.append({"usable_target_kw": target_kw, "days_since_repair": days,
                         "required_raw_input_kw_per_player": raw/KW,
                         "per_player_day_price_at_3800_ualpha_per_kw": raw/KW*.0038,
                         "per_player_day_price_at_5900_ualpha_per_kw": raw/KW*.0059})
    return rows


def affordability():
    rows = []
    for n in [1, 10, 100, 1000]:
        for price in [.01, .05, .06, .1]:
            for cadence in [7, 14, 28]:
                cost = 1+n*price
                per_person = cost/n/cadence
                for income in [1, 1/7, 1/30]:
                    rows.append({"players": n, "base_alpha": 1,
                                 "per_player_price_alpha": price,
                                 "repair_interval_days": cadence,
                                 "station_purchase_alpha": cost,
                                 "per_player_average_alpha_per_day": per_person,
                                 "assumed_player_alpha_income_per_day": income,
                                 "upkeep_fraction_of_assumed_income": per_person/income})
    return rows


def coverage_examples():
    rows = []
    wallet = 10**25
    for days in [1, 7]:
        for n, power in [(100, 0), (100, 60*1000*KW)]:
            c = ServiceCoverage()
            c.repair(0, days, 0, 0, wallet)
            empty = c.burned
            c.add_service(0, n, power, wallet)
            rows.append({"days": days, "players": n, "outgoing_kw": power/KW,
                         "empty_purchase_alpha": empty/ALPHA,
                         "added_service_charge_alpha": (c.burned-empty)/ALPHA,
                         "total_alpha": c.burned/ALPHA})
    return rows


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--out', type=Path, default=ROOT/'build/substation-model/substation-player-calibration-2026-09-25')
    args = parser.parse_args()
    types, base, slots = definitions()
    rows = profiles(types, base, slots)
    sources = [TYPE_SOURCE, KEY_SOURCE, Path(__file__), ROOT/'x/structs/keeper/struct_context.go',
               ROOT/'x/structs/keeper/fleet_cache.go', ROOT/'x/structs/keeper/player_cache.go',
               ROOT/'x/structs/keeper/msg_server_struct_build_complete.go',
               Path(__file__).with_name('service_coverage_quote.py'),
               Path(__file__).with_name('test_player_calibration.py'),
               Path(__file__).with_name('model.py'), Path(__file__).with_name('rating_model.py')]
    result = {"source_sha256": {str(f.relative_to(ROOT)): hashlib.sha256(f.read_bytes()).hexdigest() for f in sources},
              "definitions": types, "base_player_draw_mw": base, "slots_per_ambit_per_location": slots,
              "profiles": rows, "all_build_draws_equal_passive": all(t['build_mw'] == t['passive_mw'] for t in types),
              "capacity_targets": capacity_targets(Parameters()), "affordability": affordability(),
              "coverage_examples": coverage_examples(),
              "limits": "Branch default type definitions, default one fleet/planet slots; not live-state measurement. Income scenarios are assumptions."}
    args.out.mkdir(parents=True, exist_ok=True)
    (args.out/'results.json').write_text(json.dumps(result, indent=2, sort_keys=True)+'\n')
    for key in ['definitions', 'capacity_targets', 'affordability', 'coverage_examples']:
        emit_csv(args.out/f'{key}.csv', result[key])
    print(json.dumps({"profiles": [{k:v for k,v in row.items() if k != 'layout'} for row in rows],
                      "all_build_draws_equal_passive": result['all_build_draws_equal_passive']}, indent=2))


if __name__ == '__main__':
    main()
