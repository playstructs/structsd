import copy
import random
import unittest

from model import ALPHA, KW, MW, upkeep_price
from discount_delay_model import DiscountDelayGrid

WALLET = 10**20


class DiscountDelayRules(unittest.TestCase):
    def setUp(self):
        self.g = DiscountDelayGrid()
        self.s = self.g.add_station("A", 100*MW)
        self.day = self.g.p.blocks_per_day

    def test_same_block_drop_pay_restore_is_full_price_and_no_shedding(self):
        g = self.g
        g.add_station("B")
        edge = g.connect("A", "B", 60*MW)
        g.advance(self.day)
        g.root_change("A", KW)
        g.pay("A", 1, WALLET)
        g.root_change("A", 100*MW)
        g.cascade()
        self.assertEqual(self.s.burn, 401360000)
        self.assertIn(edge.index, self.s.outgoing)

    def test_exact_delay_boundary(self):
        g = self.g
        g.root_change("A", KW)
        g.advance(7*self.day-1)
        self.assertEqual(g.quote_supply("A"), 100*MW)
        g.advance(7*self.day)
        self.assertEqual(g.quote_supply("A"), KW)
        self.assertTrue(g.queue)  # Time did not service even the existing queue.

    def test_deep_late_cut_restarts_qualification(self):
        g = self.g
        g.root_change("A", 99*MW)
        g.advance(7*self.day-1)
        g.root_change("A", KW)
        g.advance(7*self.day)
        self.assertEqual(g.quote_supply("A"), 100*MW)
        g.advance(14*self.day-1)
        self.assertEqual(g.quote_supply("A"), KW)

    def test_broken_first_drop_timer_reproduces_late_cut_bypass(self):
        g = DiscountDelayGrid(restart_on_change=False)
        g.add_station("A", 100*MW)
        g.root_change("A", 99*MW)
        g.advance(7*self.day-1)
        g.root_change("A", KW)
        g.advance(7*self.day)
        self.assertEqual(g.quote_supply("A"), KW)

    def test_reconnect_renews_protection_even_after_discount_matures(self):
        g = self.g
        g.root_change("A", KW)
        g.advance(7*self.day)
        g.root_change("A", 100*MW)
        g.root_change("A", KW)
        self.assertEqual(g.quote_supply("A"), 100*MW)

    def test_unaccepted_input_has_no_effect(self):
        g = self.g
        g.add_station("source", 1000*MW)
        before = copy.deepcopy(g.discounts["A"])
        edge = g.connect("source", "A", 100*MW, accepted=False)
        g.resize(edge, 10*MW)
        g.destroy(edge)
        self.assertEqual(g.discounts["A"], before)

    def test_accepted_edge_mutations_update_guard(self):
        g = self.g
        g.add_station("source", 1000*MW)
        edge = g.connect("source", "A", 100*MW)
        self.assertEqual(g.quote_supply("A"), 200*MW)
        g.resize(edge, 50*MW)
        g.advance(7*self.day)
        self.assertEqual(g.quote_supply("A"), 150*MW)
        g.destroy(edge)
        self.assertEqual(g.quote_supply("A"), 150*MW)

    def test_small_fluctuations_delay_discount_but_not_upkeep(self):
        g = self.g
        g.root_change("A", KW)
        for day in range(1, 30):
            g.advance(day*self.day)
            g.root_change("A", KW+day%2)
            self.assertEqual(g.quote_supply("A"), 100*MW)
        g.pay("A", 1, WALLET)
        self.assertEqual(self.s.burn, 401360000)

    def test_failed_payment_keeps_pricing_and_health_state(self):
        g = self.g
        g.root_change("A", KW)
        before = copy.deepcopy(g.__dict__)
        with self.assertRaises(ValueError):
            g.pay("A", 1, 0)
        self.assertEqual(g.__dict__, before)

    def test_actual_low_supply_sheds_outgoing_allocations(self):
        g = self.g
        g.add_station("B")
        edge = g.connect("A", "B", 60*MW)
        g.root_change("A", KW)
        g.cascade()
        self.assertNotIn(edge.index, self.s.outgoing)
        g.advance(7*self.day)
        g.pay("A", 1, WALLET)
        g.root_change("A", 100*MW)
        self.assertEqual(self.s.burn, 9500)
        self.assertNotIn(edge.index, self.s.outgoing)

    def test_two_station_rotation_retains_service_and_cheap_repairs(self):
        g = self.g
        g.add_station("B", KW)
        active = "A"
        for day in range(1, 365):
            g.advance(day*self.day)
            if day%14 == 0:
                idle = "B" if active == "A" else "A"
                self.assertEqual(g.quote_supply(idle), KW)
                before = g.stations[idle].burn
                g.pay(idle, 1, WALLET)
                self.assertEqual(g.stations[idle].burn-before, 9500)
                g.root_change(active, KW)
                g.root_change(idle, 100*MW)
                active = idle  # Authorized player's connection follows supply.
                g.cascade()
            self.assertGreaterEqual(g.shared(active), 50*MW)
            self.assertEqual(sum(s.root for s in g.stations.values()), 100*MW+KW)

    def test_random_quotes_cover_current_and_protected_supply(self):
        rng = random.Random(928)
        g = self.g
        last_change = 0
        for _ in range(5000):
            old = g.supply("A")
            protected = g.quote_supply("A")
            if rng.randrange(2):
                g.advance(g.height+rng.randrange(0, 10*self.day))
            else:
                new = rng.choice([0, KW, MW, 100*MW, 1000*MW])
                g.root_change("A", new)
                if new != old:
                    last_change = g.height
                    self.assertGreaterEqual(g.quote_supply("A"), protected)
            self.assertGreaterEqual(g.quote_supply("A"), g.supply("A"))
            if g.height-last_change >= 7*self.day:
                self.assertEqual(g.quote_supply("A"), g.supply("A"))


if __name__ == "__main__":
    unittest.main()
