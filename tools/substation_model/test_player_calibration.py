import copy
from collections import Counter
import random
import unittest

from model import ALPHA, KW, MW, upkeep_price
from player_energy_calibration import AMBITS, FLAGS, definitions, profiles, max_loadout
from service_coverage_quote import ServiceCoverage

WALLET = 10**25


class PlayerCalibration(unittest.TestCase):
    def test_calculated_profiles_obey_slot_and_type_limits(self):
        types, base, slots = definitions()
        by_id = {t['id']: t for t in types}
        for row in profiles(types, base, slots):
            used, quantities = Counter(), Counter()
            for entry in row['layout']:
                t, n = by_id[entry['type_id']], entry['count']
                quantities[t['id']] += n
                if t['id'] != 1:
                    flag = FLAGS[AMBITS.index(entry['ambit'])]
                    self.assertTrue(t['ambits'] & flag)
                    used[t['category'], entry['ambit']] += n
            self.assertTrue(all(n <= slots for n in used.values()))
            self.assertTrue(all(not by_id[i]['limit'] or n <= by_id[i]['limit'] for i, n in quantities.items()))
            self.assertEqual(row['gross_mw'], base+sum(by_id[i]['passive_mw']*n for i, n in quantities.items()))
            self.assertEqual(quantities[1], 1)

    def test_current_definition_golden_totals(self):
        types, base, slots = definitions()
        result = {r['profile']: r['gross_mw'] for r in profiles(types, base, slots)}
        self.assertEqual(result['developed_with_field_generator'], 8215000)
        self.assertEqual(result['every_planet_type_plus_maximum_fleet'], 116715000)
        self.assertEqual(result['absolute_slot_limited_draw_maximum'], 117215000)
        self.assertTrue(all(t['build_mw'] == t['passive_mw'] for t in types))

    def test_optimizer_handles_competing_ambits_and_global_type_limit(self):
        types = [dict(id=1, ambits=6, limit=1, passive_mw=10),
                 dict(id=2, ambits=2, limit=0, passive_mw=9)]
        power, used, layout = max_loadout(types, 1)
        self.assertEqual(power, 19)
        self.assertEqual(sum(n for i,a,n in layout if i == 1), 1)


class CoverageArithmetic(unittest.TestCase):
    def test_empty_repair_then_reconnect_charges_normal_total(self):
        c = ServiceCoverage()
        c.repair(0, 1, 0, 0, WALLET)
        c.add_service(0, 100, 0, WALLET)
        self.assertEqual(c.burned, 6*ALPHA)

    def test_shrink_repair_restore_charges_capacity_difference(self):
        c = ServiceCoverage(covered_players=100, covered_power=60*MW)
        c.repair(0, 1, 100, KW, WALLET)
        c.add_service(0, 100, 60*MW, WALLET)
        self.assertEqual(c.burned, 255360000)

    def test_both_components_and_seven_days_are_covered(self):
        c = ServiceCoverage()
        c.repair(0, 7, 0, 0, WALLET)
        c.add_service(0, 100, 60*MW, WALLET)
        self.assertEqual(c.burned, 7*255360000)

    def test_leaving_and_returning_without_cheap_repair_is_free(self):
        c = ServiceCoverage()
        c.repair(0, 1, 100, 60*MW, WALLET)
        before = c.burned
        c.add_service(5, 0, 0, WALLET)
        c.add_service(10, 100, 60*MW, WALLET)
        self.assertEqual(c.burned, before)
        self.assertEqual(c.repaired_at, 0)

    def test_expired_prepaid_time_does_not_make_new_health_coverage_free(self):
        c = ServiceCoverage()
        c.repair(0, 1, 0, 0, WALLET)
        c.add_service(10*c.p.blocks_per_day, 100, 0, WALLET)
        self.assertEqual(c.burned, 6*ALPHA)
        self.assertEqual(c.repaired_at, 0)

    def test_atomic_renewal_prices_remaining_prepaid_growth(self):
        c = ServiceCoverage()
        c.repair(0, 6, 0, 0, WALLET)
        c.repair(0, 1, 100, 60*MW, WALLET)
        self.assertEqual(c.burned, 7*255360000)

    def test_failed_quote_application_is_atomic(self):
        c = ServiceCoverage()
        before = copy.deepcopy(c)
        with self.assertRaises(ValueError):
            c.add_service(0, 100, MW, 0)
        self.assertEqual(c, before)
        with self.assertRaises(ValueError):
            c.repair(0, 1, 100, MW, 0)
        self.assertEqual(c, before)

    def test_random_split_growth_never_cheaper_than_full_purchase(self):
        rng = random.Random(930)
        for _ in range(1000):
            c = ServiceCoverage()
            n, power, days = rng.randrange(1, 1000), rng.randrange(1, 100*MW), rng.randrange(1, 8)
            c.repair(0, days, 0, 0, WALLET)
            for step in range(1, 11):
                c.add_service(0, n*step//10, power*step//10, WALLET)
            floor = (ALPHA+n*50000)*days+upkeep_price(power, days, c.p)
            self.assertGreaterEqual(c.burned, floor)


if __name__ == '__main__':
    unittest.main()
