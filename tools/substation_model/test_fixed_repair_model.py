import copy
from dataclasses import replace
import random
import unittest

from model import ALPHA, KW, MW, Parameters, project
from fixed_repair_model import FixedRepairGrid, first_offline_block


class FixedRepairRules(unittest.TestCase):
    def setUp(self):
        self.g = FixedRepairGrid()
        self.s = self.g.seed_station("A", 100*MW)
        self.day = self.g.p.blocks_per_day

    def test_disconnect_and_expansion_do_not_change_repair_price(self):
        for root in [0, 1, KW, MW, 100*MW, 10**18]:
            g = FixedRepairGrid()
            s = g.seed_station("A", root)
            g.advance(29*self.day)
            wallet = g.repair("A", 100*ALPHA)
            g.root_change("A", 100*MW)
            self.assertEqual(wallet, 90*ALPHA)
            self.assertEqual(s.burn, 10*ALPHA)
            self.assertEqual(g.throughput("A"), 70*MW)

    def test_no_bonus_expiry_or_future_credit(self):
        g = self.g
        g.advance(self.day)
        g.repair("A", 100*ALPHA)
        self.assertEqual(g.throughput("A"), 70*MW)
        self.assertEqual(self.s.paid_until, 0)
        g.advance(8*self.day)
        self.assertEqual(g.throughput("A"), project(100*MW, 7*self.day, 7000, g.p))
        self.assertFalse(g.queue)
        with self.assertRaises(ValueError):
            g.pay("A", 7, 100*ALPHA)

    def test_payment_failures_do_not_mutate(self):
        g = self.g
        before = copy.deepcopy(g.__dict__)
        with self.assertRaises(ValueError):
            g.repair("A", 100*ALPHA)  # same block
        self.assertEqual(before, g.__dict__)
        g.advance(1)
        before = copy.deepcopy(g.__dict__)
        with self.assertRaises(ValueError):
            g.repair("A", 1)
        self.assertEqual(before, g.__dict__)
        self.s.collateral = 0
        with self.assertRaises(ValueError):
            g.repair("A", 100*ALPHA)
        self.assertEqual(g.throughput("A"), 0)

    def test_creation_funds_collateral_and_first_repair_without_mint(self):
        g = FixedRepairGrid()
        g.advance(50*self.day)
        wallet = g.create("new", 100*MW, 100*ALPHA)
        self.assertEqual(wallet, 89*ALPHA)
        self.assertEqual(g.bootstrap_minted, 0)
        self.assertEqual(g.collateral_funded, ALPHA//2)
        self.assertEqual(g.creation_burned, ALPHA//2)
        self.assertEqual(g.throughput("new"), 70*MW)
        before = copy.deepcopy(g.__dict__)
        with self.assertRaises(ValueError):
            g.create("poor", MW, 0)
        self.assertEqual(before, g.__dict__)

    def test_exact_outage_boundary_oracle(self):
        rng = random.Random(926)
        p = Parameters()
        for _ in range(2000):
            supply = rng.randrange(1, 10**15)
            demand = rng.randrange(1, supply+1)
            first = first_offline_block(supply, demand, p)
            self.assertLess(project(supply, first, 7000, p), demand)
            if first:
                self.assertGreaterEqual(project(supply, first-1, 7000, p), demand)

    def test_exact_headroom_requires_every_block_repair(self):
        self.assertEqual(first_offline_block(100*MW, 70*MW, self.g.p), 1)
        self.assertEqual(first_offline_block(100*MW, 71*MW, self.g.p), 0)

    def test_daily_repair_does_not_pay_future_days(self):
        for day in range(1, 8):
            self.g.advance(day*self.day)
            self.g.repair("A", 100*ALPHA)
        self.assertEqual(self.s.burn, 70*ALPHA)
        self.g.advance(37*self.day)
        self.assertEqual(self.g.throughput("A"), 0)
        self.assertFalse(self.g.queue)

    def test_neglected_relay_still_needs_player_enforcement(self):
        g = self.g
        g.seed_station("B")
        e = g.connect("A", "B", 60*MW, automated=True)
        g.advance(31*self.day)
        g.repair("B", 100*ALPHA)
        self.assertEqual(g.throughput("A"), 0)
        self.assertEqual(g.throughput("B"), 42*MW)
        self.assertFalse(g.queue)
        g.audit("A")
        self.assertEqual(e.power, 0)
        self.assertEqual(g.throughput("B"), 0)

    def test_repair_does_not_restore_destroyed_contracts(self):
        g = self.g
        g.seed_station("B")
        e = g.connect("A", "B", 60*MW)
        g.advance(31*self.day)
        g.audit("A")
        g.repair("A", 100*ALPHA)
        self.assertNotIn(e.index, self.s.outgoing)
        self.assertEqual(g.throughput("A"), 70*MW)

    def test_parallel_partition_cannot_improve_aggregate_output(self):
        rng = random.Random(927)
        for _ in range(1000):
            parts = [rng.randrange(1, 100*MW) for _ in range(rng.randrange(2, 20))]
            age = rng.randrange(0, 31*self.day)
            self.assertLessEqual(sum(project(x, age, 7000, self.g.p) for x in parts),
                                 project(sum(parts), age, 7000, self.g.p))
            self.assertGreater(len(parts)*self.g.repair_price, self.g.repair_price)

    def test_hundred_percent_is_only_a_sensitivity_parameter(self):
        g = FixedRepairGrid(replace(Parameters(), durable_bps=10000))
        g.seed_station("A", 100*MW)
        self.assertEqual(g.throughput("A"), 100*MW)

    def test_lossy_feedback_loop_drains_after_external_supply_is_removed(self):
        g = self.g
        g.seed_station("B")
        ab = g.connect("A", "B", g.safe("A"), automated=True)
        ba = g.connect("B", "A", g.safe("B"), automated=True)
        for _ in range(1000):
            before = (ab.power, ba.power)
            g.resize(ab, g.safe("A"), admission=True)
            g.resize(ba, g.safe("B"), admission=True)
            if before == (ab.power, ba.power):
                break
        else:
            self.fail("loop failed to converge")
        self.assertLessEqual(g.shared("A")+g.shared("B"), 100*MW)
        g.root_change("A", 0)
        g.drain(budget=4)
        self.assertEqual((ab.power, ba.power), (0, 0))
        self.assertEqual(g.shared("A")+g.shared("B"), 0)


if __name__ == "__main__":
    unittest.main()
