"""Adversarial transitions and independent rational-price checks."""
import copy
from fractions import Fraction
import random
import unittest

from model import ALPHA, KW, MW, Parameters, upkeep_price
from rating_model import RatingGrid, tariff_numerator

WALLET = 10**25


class RatingRules(unittest.TestCase):
    def setUp(self):
        self.g = RatingGrid()
        self.s = self.g.seed_station("A", 100*MW)
        self.day = self.g.p.blocks_per_day

    def test_disconnect_does_not_discount_bill(self):
        g = self.g
        g.advance(29*self.day)
        g.root_change("A", KW)
        g.pay("A", 1, WALLET)
        g.root_change("A", 100*MW)
        self.assertEqual(self.s.burn, 401360000)
        self.assertEqual(g.throughput("A"), 100*MW)

    def test_cheap_reset_requires_full_topup_before_output_returns(self):
        g = self.g
        g.advance(29*self.day)
        g.set_rating("A", KW, WALLET)
        g.pay("A", 1, WALLET)
        self.assertEqual(g.throughput("A"), KW)
        self.assertEqual(self.s.serviced_ceiling, KW)
        g.set_rating("A", 100*MW, WALLET)
        self.assertEqual(self.s.burn, 401360000)
        self.assertEqual(g.throughput("A"), 100*MW)

    def test_down_up_without_upkeep_keeps_old_age(self):
        g = self.g
        g.advance(29*self.day)
        before = g.throughput("A")
        for _ in range(50):
            g.set_rating("A", 0, WALLET)
            g.set_rating("A", 100*MW, WALLET)
        self.assertEqual(self.s.burn, 0)
        self.assertEqual(self.s.last_upkeep, 0)
        self.assertEqual(g.throughput("A"), before)

    def test_expansion_does_not_heal_existing_capacity(self):
        g = self.g
        g.advance(31*self.day)
        g.set_rating("A", 101*MW, WALLET)
        self.assertEqual(g.throughput("A"), 0)
        self.assertEqual(self.s.last_upkeep, 0)

    def test_prepaid_expansion_covers_all_remaining_blocks(self):
        g = RatingGrid()
        s = g.add_station("A", 100*MW)
        g.pay("A", 7, WALLET, rating=KW)
        g.advance(self.day//2)
        expected = Fraction((401360000-9500)*(7*self.day-self.day//2), self.day)
        quote = g.expansion_price("A", 100*MW)
        self.assertEqual(quote, -(-expected.numerator//expected.denominator))
        before = (s.last_upkeep, s.paid_until)
        g.set_rating("A", 100*MW, WALLET)
        self.assertEqual((s.last_upkeep, s.paid_until), before)

    def test_atomic_renewal_expansion_charges_carried_time(self):
        g = RatingGrid()
        s = g.add_station("A", 100*MW)
        g.pay("A", 6, WALLET, rating=KW)
        g.pay("A", 1, WALLET, rating=100*MW)
        self.assertEqual(s.burn, upkeep_price(100*MW, 7, g.p))

    def test_downsize_renewal_forfeits_large_coverage(self):
        g = self.g
        g.pay("A", 6, WALLET)
        g.pay("A", 1, WALLET, rating=KW)
        self.assertEqual(self.s.serviced_ceiling, KW)
        self.assertEqual(g.throughput("A"), KW)
        self.assertEqual(g.expansion_price("A", 100*MW), 7*(401360000-9500))

    def test_one_day_floor_survives_expiry_and_tiny_time_remainder(self):
        for age in [self.day-1, self.day, 20*self.day]:
            g = RatingGrid()
            g.add_station("A", 100*MW)
            g.pay("A", 1, WALLET, rating=KW)
            g.advance(age)
            self.assertEqual(g.expansion_price("A", 100*MW), 401360000-9500)

    def test_new_station_cannot_replace_expensive_upkeep_for_free(self):
        g = RatingGrid()
        s = g.add_station("new", 100*MW)
        self.assertEqual(g.throughput("new"), 0)
        g.pay("new", 1, WALLET, rating=100*MW)
        self.assertEqual(s.burn, 401360000)

    def test_failed_purchase_is_atomic(self):
        g = self.g
        before = copy.deepcopy(g.__dict__)
        for action in [lambda: g.pay("A", 1, 0),
                       lambda: g.pay("A", 8, WALLET),
                       lambda: g.pay("A", 1, WALLET, rating=0),
                       lambda: g.set_rating("A", 101*MW, 0),
                       lambda: g.set_rating("A", -1, WALLET)]:
            with self.assertRaises(ValueError):
                action()
            self.assertEqual(g.__dict__, before)

    def test_rating_limits_budget_and_reduction_cascades(self):
        g = self.g
        g.seed_station("B", 0, rating=100*MW)
        e = g.connect("A", "B", 60*MW, automated=True)
        g.set_rating("A", MW, WALLET)
        self.assertIn("A", g.queued)
        g.cascade()
        self.assertEqual(e.power, g.safe("A"))
        self.assertLess(e.power, MW)
        with self.assertRaises(ValueError):
            g.connect("A", "B", MW)

    def test_time_still_does_not_enqueue(self):
        g = self.g
        g.advance(31*self.day)
        self.assertEqual(g.throughput("A"), 0)
        self.assertFalse(g.queue)

    def test_rating_cannot_revive_zero_collateral(self):
        self.s.collateral = 0
        self.g.set_rating("A", 0, WALLET)
        with self.assertRaises(ValueError):
            self.g.set_rating("A", 100*MW, WALLET)
        with self.assertRaises(ValueError):
            self.g.pay("A", 1, WALLET, rating=100*MW)
        self.assertEqual(self.g.throughput("A"), 0)

    def test_partitioning_commissioned_stations_does_not_discount_tariff(self):
        rng = random.Random(923)
        for _ in range(1000):
            parts = [rng.randrange(1, 100*MW) for _ in range(rng.randrange(2, 20))]
            days = rng.randrange(1, 8)
            self.assertGreaterEqual(sum(upkeep_price(r, days, self.g.p) for r in parts),
                                    upkeep_price(sum(parts), days, self.g.p))

    def test_dust_steps_cannot_round_the_expansion_price_to_zero(self):
        g = RatingGrid()
        s = g.add_station("A")
        g.pay("A", 1, WALLET, rating=1)
        for rating in range(2, 102):
            self.assertGreater(g.expansion_price("A", rating), 0)
            g.set_rating("A", rating, WALLET)
        self.assertEqual(s.burn, 101)

    def test_split_expansion_never_beats_single_quote(self):
        rng = random.Random(922)
        for _ in range(1000):
            g = RatingGrid()
            g.add_station("A")
            start = rng.randrange(1, 10*MW)
            g.pay("A", 7, WALLET, rating=start)
            g.advance(rng.randrange(7*self.day))
            targets = sorted(rng.sample(range(start+1, 100*MW), 10))
            direct = g.expansion_price("A", targets[-1])
            burn = g.stations["A"].burn
            for rating in targets:
                g.set_rating("A", rating, WALLET)
            self.assertGreaterEqual(g.stations["A"].burn-burn, direct)

    def test_random_sequences_preserve_funding_and_entitlement(self):
        # 50,000 actions with an independent rational tariff oracle. From a
        # fresh station, lifetime spend must cover its currently purchased
        # ceiling for at least one day or its remaining prepaid duration.
        rng = random.Random(925)
        ratings = [0, 1, KW, 100*KW-1, 100*KW+1, MW, 10*MW, 100*MW, 10**18]
        p = Parameters()

        def tariff(r):
            kw = Fraction(r, KW)
            return (min(kw, 100)*9500 + min(max(kw-100, 0), 9900)*5900
                    + max(kw-10000, 0)*3800)

        for _ in range(1000):
            g = RatingGrid(p)
            s = g.add_station("A", 100*MW)
            wallet = WALLET
            for _ in range(50):
                op = rng.randrange(4)
                previous_age = s.last_upkeep
                try:
                    if op == 0:
                        wallet = g.set_rating("A", rng.choice(ratings), wallet)
                        self.assertEqual(s.last_upkeep, previous_age)
                    elif op == 1:
                        wallet = g.pay("A", rng.randrange(1, 8), wallet,
                                       rating=rng.choice(ratings))
                    elif op == 2:
                        g.advance(g.height+rng.randrange(0, 10*self.day))
                    else:
                        g.root_change("A", rng.choice(ratings))
                except ValueError:
                    pass
                minimum = tariff(s.serviced_ceiling)*max(Fraction(1),
                                    Fraction(s.paid_until-g.height, self.day))
                self.assertGreaterEqual(s.burn, minimum)
                self.assertEqual(s.burn+wallet, WALLET)
                self.assertLessEqual(s.rating, s.serviced_ceiling)
                self.assertLessEqual(g.throughput("A"), min(s.rating, g.supply("A")))
                self.assertEqual(Fraction(tariff_numerator(s.rating, p), KW), tariff(s.rating))


if __name__ == "__main__":
    unittest.main()
