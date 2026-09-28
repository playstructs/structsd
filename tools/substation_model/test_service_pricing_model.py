import copy
from fractions import Fraction
import random
import unittest

from model import ALPHA, KW, MW
from service_pricing_model import ServicePricingGrid

WALLET = 10**25


class ServicePricingRules(unittest.TestCase):
    def setUp(self):
        self.g = ServicePricingGrid()
        self.s = self.g.add_station("A", 100*MW)
        self.g.add_station("B")
        self.day = self.g.p.blocks_per_day
        for i in range(100):
            self.g.move_player(str(i), "A")

    def test_input_cut_does_not_change_customer_or_outgoing_bill(self):
        g = self.g
        g.connect("A", "B", 60*MW)
        quote = g.quote("A")
        self.assertEqual(quote, 255360000)
        for root in [0, KW, MW, 100*MW, 1000*MW]:
            g.root_change("A", root)
            self.assertEqual(g.quote("A"), quote)

    def test_customer_migration_lowers_price_and_restoration_is_free(self):
        g = self.g
        g.advance(29*self.day)
        before = g.quote("A")
        for i in range(100):
            g.move_player(str(i), "B")
        g.pay("A", 1, WALLET)
        for i in range(100):
            g.move_player(str(i), "A")
        self.assertEqual(before, 6*ALPHA)
        self.assertEqual(self.s.burn, ALPHA)
        self.assertEqual(g.player_capacity("0"), MW)
        self.assertEqual(g.connection_counts, {"A": 100, "B": 0})

    def test_disconnect_does_not_require_a_second_station(self):
        g = self.g
        g.advance(self.day)
        for i in range(100):
            g.move_player(str(i), None)
        g.pay("A", 1, WALLET)
        for i in range(100):
            g.move_player(str(i), "A")
        self.assertEqual(self.s.burn, ALPHA)
        self.assertEqual(g.connection_counts["A"], 100)

    def test_dynamic_shrink_repair_restore_preserves_id_and_downstream_edge(self):
        g = self.g
        g.add_station("C")
        upstream = g.connect("A", "B", 60*MW)
        downstream = g.connect("B", "C", 30*MW, kind="static")
        g.advance(self.day)
        g.resize_dynamic(upstream, KW)
        g.pay("A", 1, WALLET)
        g.resize_dynamic(upstream, 60*MW)
        g.cascade()
        self.assertEqual(self.s.burn, 6009500)
        self.assertIn(upstream.index, self.s.outgoing)
        self.assertIn(downstream.index, g.stations["B"].outgoing)

    def test_cascade_between_shrink_and_restore_destroys_downstream_contract(self):
        g = self.g
        g.add_station("C")
        upstream = g.connect("A", "B", 60*MW)
        downstream = g.connect("B", "C", 30*MW, kind="static")
        g.resize_dynamic(upstream, KW)
        g.cascade()
        g.pay("A", 1, WALLET)
        g.resize_dynamic(upstream, 60*MW)
        self.assertNotIn(downstream.index, g.stations["B"].outgoing)

    def test_static_provider_and_auto_are_not_directly_resizable(self):
        for kind in ["static", "provider", "automated"]:
            g = ServicePricingGrid()
            g.add_station("A", 100*MW)
            g.add_station("B")
            edge = g.connect("A", "B", MW, kind=kind)
            with self.assertRaises(ValueError):
                g.resize_dynamic(edge, KW)
            self.assertEqual(edge.power, MW)

    def test_growth_still_must_fit_durable_budget(self):
        edge = self.g.connect("A", "B", MW)
        with self.assertRaises(ValueError):
            self.g.resize_dynamic(edge, 100*MW)
        self.assertEqual(edge.power, MW)

    def test_connection_counts_are_unique_and_permission_failure_is_atomic(self):
        g = self.g
        g.move_player("0", "A")
        self.assertEqual(g.connection_counts["A"], 100)
        before = copy.deepcopy(g.__dict__)
        with self.assertRaises(ValueError):
            g.move_player("0", "B", authorized=False)
        self.assertEqual(g.__dict__, before)

    def test_price_rounding_against_fraction_oracle(self):
        rng = random.Random(929)
        for _ in range(1000):
            g = ServicePricingGrid()
            g.add_station("A", 10**18)
            g.add_station("B")
            power = rng.randrange(1, 1000*MW)
            g.connect("A", "B", power)
            n, days = rng.randrange(20), rng.randrange(1, 8)
            for i in range(n):
                g.move_player(str(i), "A")
            kw = Fraction(power, KW)
            tariff = (min(kw, 100)*9500+min(max(0, kw-100), 9900)*5900
                      +max(0, kw-10000)*3800)*days
            expected = (ALPHA+n*50000)*days-(-tariff.numerator//tariff.denominator)
            self.assertEqual(g.quote("A", days), expected)

    def test_billing_sums_outgoing_before_tiering(self):
        one = self.g.connect("A", "B", 60*MW)
        quote = self.g.quote("A")
        self.g.destroy(one)
        for _ in range(60):
            self.g.connect("A", "B", MW)
        self.assertEqual(self.g.quote("A"), quote)

    def test_failed_payments_do_not_mutate(self):
        g = self.g
        before = copy.deepcopy(g.__dict__)
        for days, wallet in [(1, 0), (8, WALLET), (0, WALLET)]:
            with self.assertRaises(ValueError):
                g.pay("A", days, wallet)
            self.assertEqual(g.__dict__, before)

    def test_zero_base_empty_station_can_repair_for_free(self):
        g = ServicePricingGrid(base_fee=0)
        s = g.add_station("A", 100*MW)
        g.advance(29*self.day)
        g.pay("A", 1, 0)
        self.assertEqual(s.burn, 0)
        self.assertEqual(g.throughput("A"), 100*MW)

    def test_seven_days_can_be_bought_before_customers_arrive(self):
        g = ServicePricingGrid()
        s = g.add_station("A", 100*MW)
        g.pay("A", 7, WALLET)
        for i in range(100):
            g.move_player(str(i), "A")
        self.assertEqual(s.burn, 7*ALPHA)
        self.assertEqual(g.quote("A", 7), 42*ALPHA)

    def test_time_does_not_schedule_shutdown_or_remove_customers(self):
        self.g.advance(31*self.day)
        self.assertEqual(self.g.throughput("A"), 0)
        self.assertEqual(self.g.connection_counts["A"], 100)
        self.assertFalse(self.g.queue)


if __name__ == "__main__":
    unittest.main()
