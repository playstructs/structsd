"""Semantic checks and independent arithmetic oracles for the research model."""
import random
import unittest
from dataclasses import replace
from fractions import Fraction

from model import (ALPHA, KW, MW, BPS, Grid, Parameters, bounty,
                   calculate_difficulty, project, upkeep_price)
from experiments import cycle, loop_control, threshold_age


class Arithmetic(unittest.TestCase):
    def setUp(self):
        self.p = Parameters()

    def test_price_golden_vectors_and_marginal_brackets(self):
        for kw, expected in [(1, 9500), (100, 950000), (1000, 6260000),
                             (10000, 59360000), (29600, 133840000), (100000, 401360000)]:
            self.assertEqual(upkeep_price(kw*KW, 1, self.p), expected)
        self.assertEqual(upkeep_price(1, 1, self.p), 1)
        self.assertEqual(upkeep_price(1000*KW, 7, self.p), 7*6260000)

    def test_quadratic_health_fraction_oracle(self):
        rng = random.Random(824)
        for _ in range(500):
            age, power = rng.randrange(1_000_000), rng.randrange(1, 10**18)
            h = max(Fraction(0), 1-Fraction(age, self.p.wear_blocks)**2)
            expected = int(power*Fraction(7, 10)*h)
            self.assertEqual(project(power, age, 7000, self.p), expected)
        self.assertEqual(project(MW, self.p.wear_blocks, BPS, self.p), 0)

    def test_difficulty_matches_power_inequalities(self):
        for r in [1387, 65380, 28000]:
            previous = 64
            for age in sorted({1, 2, 10, 100, 1000, r//2, r, r+1}):
                d = calculate_difficulty(age, r)
                self.assertGreaterEqual(previous, d)
                previous = d
                if age > 1:
                    exponent = 64-d
                    self.assertLessEqual(r**exponent, age**63)
                    if exponent < 63:
                        self.assertGreater(r**(exponent+1), age**63)
            for d in [5, 8, 9]:
                onset = threshold_age(d, r)
                self.assertLessEqual(calculate_difficulty(onset, r), d)
                self.assertGreater(calculate_difficulty(onset-1, r), d)

    def test_dust_sweep_boundaries_and_no_unbacked_rewards(self):
        p = replace(self.p, dust_threshold_ualpha=100)
        self.assertEqual(bounty(200, 1, 1, 9, p), 200)
        self.assertEqual(bounty(202, 1, 1, 9, p), 101)
        self.assertEqual(bounty(1, 1, 10**12, 5, p), 1)
        self.assertEqual(bounty(0, 1, 1, 9, p), 0)
        rng = random.Random(822)
        for _ in range(1000):
            balance, load = rng.randrange(1, 10**10), rng.randrange(1, 10**12)
            reward = bounty(balance, rng.randrange(1, load+1), load, rng.randrange(1, 16), p)
            self.assertTrue(0 < reward <= balance)


class GridRules(unittest.TestCase):
    def setUp(self):
        self.g = Grid()
        self.g.add_station("A", 100*MW)
        self.g.add_station("B")

    def test_unaccepted_input_cannot_raise_output_or_bill(self):
        g = self.g
        g.connect("A", "B", 10*MW, accepted=False)
        self.assertEqual(g.supply("B"), 0)
        self.assertEqual(g.throughput("B"), 0)

    def test_paid_period_keeps_health_decay_and_excludes_boost_from_budget(self):
        g = self.g
        budget = g.safe("A")
        g.pay("A", 7, 10000*ALPHA)
        self.assertEqual(g.safe("A"), budget)
        before = g.throughput("A")
        g.advance(g.p.blocks_per_day)
        self.assertLess(g.throughput("A"), before)
        g.advance(7*g.p.blocks_per_day)
        self.assertEqual(g.throughput("A"), project(100*MW, g.height, 7000, g.p))

    def test_time_passage_never_schedules_enforcement(self):
        g = self.g
        e = g.connect("A", "B", 60*MW, automated=True)
        g.advance(31*g.p.blocks_per_day)
        self.assertEqual(g.throughput("A"), 0)
        self.assertEqual(e.power, 60*MW)
        self.assertFalse(g.queue)

    def test_zero_health_audit_corrects_claim_and_is_not_repeatable(self):
        g = self.g
        e = g.connect("A", "B", 60*MW, automated=True)
        g.advance(31*g.p.blocks_per_day)
        result = g.audit("A", 9)
        self.assertEqual(result["reward_ualpha"], 250000)
        self.assertEqual(e.power, 0)
        self.assertIn(e.index, g.stations["A"].outgoing)
        self.assertEqual(g.bootstrap_minted, g.bounties_paid+sum(s.collateral for s in g.stations.values()))
        with self.assertRaises(ValueError):
            g.audit("A")

    def test_automated_exclusivity_including_zero_power(self):
        g = self.g
        e = g.connect("A", "B", 10*MW, automated=True)
        g.resize(e, 0)
        with self.assertRaisesRegex(ValueError, "exclusive"):
            g.connect("A", "B", MW)
        g.destroy(e)
        g.connect("A", "B", MW)
        with self.assertRaisesRegex(ValueError, "exclusive"):
            g.connect("A", "B", MW, automated=True)

    def test_full_dust_claim_forces_zero_capacity(self):
        g = self.g
        g.stations["A"].collateral = 10000
        g.connect("A", "B", 60*MW)
        g.advance(31*g.p.blocks_per_day)
        result = g.audit("A")
        self.assertEqual(result["reward_ualpha"], 10000)
        self.assertEqual(g.stations["A"].collateral, 0)
        self.assertEqual(g.load("A"), 0)

    def test_numeric_age_and_bounded_cascade(self):
        g = self.g
        edges = [g.connect("A", "B", MW) for _ in range(12)]
        g.root_change("A", 0)
        used = g.cascade(budget=3)
        self.assertLessEqual(used, 3)
        destroyed = [x["edge"] for x in g.events if x["kind"] == "destroy"]
        self.assertEqual(destroyed, [edges[0].index, edges[1].index])
        self.assertTrue(g.queue)
        g.drain(budget=3)
        destroyed = [x["edge"] for x in g.events if x["kind"] == "destroy"]
        self.assertEqual(destroyed, list(range(1,13)))
        self.assertEqual(g.supply("B"), 0)

    def test_paid_capacity_ceiling_does_not_fix_whole_station_health_reset(self):
        g = self.g
        g.advance(29*g.p.blocks_per_day)
        before = g.throughput("A")
        g.root_change("A", KW)
        g.pay("A", 1, ALPHA)
        g.root_change("A", 100*MW)
        self.assertEqual(g.stations["A"].burn, 9500)
        self.assertGreater(g.throughput("A"), before*10)
        self.assertLess(g.throughput("A"), 100*MW)

    def test_lossy_healthy_cycle_does_not_create_unlimited_power(self):
        result = loop_control(self.g.p)
        self.assertFalse(result["energy_amplification_found"])
        self.assertEqual(result["edges_after_mw"], [0,0])
        self.assertEqual(result["useful_after_withdrawal_kw"], 0)

    def test_trimmed_supply_and_monthly_upkeep_does_not_guarantee_uptime(self):
        p = self.g.p
        row = cycle(ceil_div_local(16_300*KW*10,7), 16_300*KW, 29, 1, p)
        self.assertLess(row["online_fraction"], .04)
        self.assertLess(row["min_service_ratio"], .07)


def ceil_div_local(n, d):
    return -(-n//d)


if __name__ == "__main__":
    unittest.main()
