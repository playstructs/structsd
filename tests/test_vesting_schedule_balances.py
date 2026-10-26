"""Independent expectations for actual unlock and pending-refund observations."""
import copy
import unittest

from vesting_schedule_balances import schedule_balances, vault_balances


def coins(value, denom="ualpha"):
    return [{"denom": denom, "amount": str(value)}] if value else []


def schedule(kind, original=10, delegated=0, start=0, end=100):
    return {"kind": kind, "original_vesting": coins(original), "delegated_vesting": coins(delegated),
        "delegated_free": [], "start_time": str(start), "end_time": str(end), "periods": []}


class ScheduleBalanceTests(unittest.TestCase):
    def test_continuous_rounding_start_and_end_boundaries(self):
        row = schedule(1, original=5, delegated=1, start=100, end=110)
        for at, vested, locked in ((99, 0, 4), (100, 0, 4), (101, 0, 4),
                (103, 2, 2), (105, 2, 2), (107, 4, 0), (109, 4, 0), (110, 5, 0), (111, 5, 0)):
            with self.subTest(at=at):
                result = schedule_balances(row, at)
                self.assertEqual(result["vested"].get("ualpha", 0), vested)
                self.assertEqual(result["locked"].get("ualpha", 0), locked)

    def test_periodic_unequal_installments_and_zero_length_sdk_boundary(self):
        row = schedule(3, original=10, delegated=2, start=100, end=120)
        row["periods"] = [{"length": "0", "amount": coins(1)},
            {"length": "5", "amount": coins(3)}, {"length": "15", "amount": coins(6)}]
        for at, vested, locked in ((100, 0, 8), (101, 1, 7), (104, 1, 7), (105, 4, 4), (119, 4, 4), (120, 10, 0)):
            with self.subTest(at=at):
                result = schedule_balances(row, at)
                self.assertEqual(result["vested"].get("ualpha", 0), vested)
                self.assertEqual(result["locked"].get("ualpha", 0), locked)

    def test_delayed_and_permanent_locks(self):
        delayed, permanent = schedule(2, delegated=4), schedule(4, delegated=3, end=0)
        for at in (99, 100, 101, 2**62):
            self.assertEqual(schedule_balances(delayed, at)["locked"], {"ualpha": 6} if at < 100 else {})
            self.assertEqual(schedule_balances(permanent, at)["locked"], {"ualpha": 7})

    def test_vault_rounds_sources_separately_and_cannot_cross_offset_locks(self):
        # Two halves each round to even zero; merging their amounts would vest 1.
        vault = {"schedules": [schedule(1, original=1, start=100, end=102) for _ in range(2)],
            "liquid_delegated": coins(11)}
        self.assertEqual(vault_balances(vault, 101)["locked"], {"ualpha": 2})
        vault["schedules"] = [schedule(2, original=10, delegated=10, end=100),
            schedule(4, original=10, delegated=0, end=0)]
        result = vault_balances(vault, 101)
        self.assertEqual(result["locked"], {"ualpha": 10}, "expired-source delegated vesting cannot offset another source")
        self.assertEqual(result["delegated_free"], {"ualpha": 11}, "ordinary stake is not an extra vesting offset")

    def test_multi_denomination_counters_and_nonmutation(self):
        row = schedule(2, original=10, delegated=3)
        row["original_vesting"] += coins(9, "urocks")
        row["delegated_vesting"] += coins(8, "urocks")
        row["delegated_free"] = coins(13, "uother")
        original = copy.deepcopy(row)
        result = schedule_balances(row, 99)
        self.assertEqual(result["locked"], {"ualpha": 7, "urocks": 1})
        self.assertEqual(result["delegated_free"], {"uother": 13})
        self.assertEqual(row, original)

    def test_malformed_schedule_and_overflow_refuse(self):
        for field, value in (("kind", True), ("kind", 9), ("kind", "1"), ("start_time", "1"),
                ("end_time", "9223372036854775808"), ("original_vesting", coins(2**256)),
                ("delegated_vesting", coins(11)), ("periods", False), ("periods", ""),
                ("original_vesting", coins(10) * 2), ("original_vesting", [None])):
            with self.subTest(field=field):
                row = schedule(2)
                row[field] = value
                with self.assertRaises(ValueError):
                    schedule_balances(row, 1)
        row = schedule(3, start=100, end=110)
        for periods in ([], [{"length": "-1", "amount": coins(10)}],
                [{"length": "10", "amount": coins(9)}], [{"length": "11", "amount": coins(10)}]):
            row["periods"] = periods
            with self.assertRaises(ValueError):
                schedule_balances(row, 101)
        with self.assertRaisesRegex(ValueError, "aggregate"):
            vault_balances({"schedules": [schedule(4, original=2**255, end=0)] * 2}, 1)


if __name__ == "__main__":
    unittest.main()
