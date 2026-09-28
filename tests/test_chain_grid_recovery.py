"""Resume follows complete scoped grid associations and refuses ambiguity."""
import base64
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import tempfile
import unittest

from fixture_grid_recovery import feeder, recover_grid


def allocation(index, destination="4-2", controller="1-7"):
    return {"id": f"6-{index}", "index": str(index), "sourceObjectId": "1-7",
        "destinationId": destination, "controller": controller, "type": "dynamic"}


def page(rows=None, next_key=None):
    return {"Allocation": [] if rows is None else rows, "pagination": {"next_key": next_key}}


def cursor(n):
    return base64.b64encode(str(n).encode()).decode()


def grid_records():
    return {"player": {"Player": {"id": "1-7", "guildId": "0-22", "substationId": "4-11"}},
        "guild": {"Guild": {"id": "0-22", "entrySubstationId": "4-99", "primaryReactorId": "3-18"}},
        "substation": {"Substation": {"id": "4-11", "owner": "1-7"}},
        "reactor": {"Reactor": {"id": "3-18", "validator": "validator-match"}},
        "validator": {"validator": {"operator_address": "validator-match"}},
        "allocation-all-by-source": page([allocation(12, "4-11")])}


class GridRecoveryTests(unittest.TestCase):
    def test_complete_inventory_matches_associations_instead_of_newest_id(self):
        rows = sorted([allocation(n, "4-11" if n == 99 else "4-2") for n in range(1, 202)],
            key=lambda row: row["id"])
        calls = []
        def query(*args):
            calls.append(args)
            if args == ("status",):
                return {"sync_info": {"latest_block_height": "123"}}
            offset = (len(calls) - 2) * 100
            expected = ["query", "structs", "allocation-all-by-source", "1-7", "--page-limit", "100", "--height", "123"]
            if offset:
                expected += ["--page-key", cursor(offset)]
            self.assertEqual(args, tuple(expected))
            return page(rows[offset:offset + 100], cursor(offset + 100) if offset + 100 < len(rows) else None)
        self.assertEqual(feeder(query, "1-7", "4-11", "1-7"), "6-99")
        self.assertEqual(len(calls), 4)

    def test_ambiguity_on_a_later_page_is_failure(self):
        for following_key in ("next_key", "nextKey"):
            documents = iter([{"Allocation": [allocation(1, "4-11")], "pagination": {following_key: cursor(1)}},
                page([allocation(2, "4-11")])])
            with self.assertRaisesRegex(ValueError, "ambiguous fixture feeder"):
                feeder(lambda *args: next(documents), "1-7", "4-11", "1-7", "123", page_size=1)
        # Same destination with a different controller is a distinct allocation.
        self.assertEqual(feeder(lambda *args: page([allocation(1, "4-11"), allocation(2, "4-11", "0-1")]),
            "1-7", "4-11", "1-7", "123"), "6-1")

    def test_missing_is_empty_only_after_a_valid_complete_inventory(self):
        for document in (page(), {"pagination": {}}, {"Allocation": None, "pagination": {}}):
            self.assertEqual(feeder(lambda *args: document, "1-7", "4-11", "1-7", "123"), "")
        invalid = [None, {}, [], page(False), page([None]), page([{}]),
            {"allocation": [], "pagination": {}}, page([dict(allocation(1), index="01")]),
            page([dict(allocation(1), index=str(2**64))]), page([dict(allocation(1), sourceObjectId="1-8")]),
            page([dict(allocation(1), destinationId=None)]), page([dict(allocation(1), controller="1-0")]),
            page([dict(allocation(1), type="unknown")]), page([allocation(1), allocation(1)]),
            page([], "not base64"), {"Allocation": [], "pagination": {"nextKey": "", "next_key": ""}}]
        for document in invalid:
            with self.subTest(document=document), self.assertRaises((ValueError, TypeError)):
                feeder(lambda *args: document, "1-7", "4-11", "1-7", "123")

    def test_query_failures_cycles_and_budgets_do_not_return_partial_matches(self):
        first = page([allocation(1, "4-11")], cursor(1))
        def failed(*args):
            if "--page-key" in args:
                raise RuntimeError("RPC failed after a matching first page")
            return first
        with self.assertRaises(RuntimeError):
            feeder(failed, "1-7", "4-11", "1-7", "123", page_size=1)
        with self.assertRaisesRegex(ValueError, "budget"):
            feeder(lambda *args: first, "1-7", "4-11", "1-7", "123", page_size=1, max_pages=1)
        with self.assertRaisesRegex(ValueError, "cycle"):
            feeder(lambda *args: page([], cursor(1)), "1-7", "4-11", "1-7", "123", page_size=1)
        for height in ("0", "01", str(2**63), None):
            with self.assertRaises(ValueError):
                feeder(lambda *args: {}, "1-7", "4-11", "1-7", height)

    def test_grid_uses_live_associations_at_one_height(self):
        records = grid_records()
        calls = []
        def query(*args):
            calls.append(args)
            if args == ("status",):
                return {"sync_info": {"latest_block_height": "123"}}
            self.assertEqual(args[-2:], ("--height", "123"))
            return records[args[2]]
        result = recover_grid(query, "1-7")
        self.assertEqual(result, {"queryHeight": "123", "playerId": "1-7", "guildId": "0-22",
            "substationId": "4-11", "reactorId": "3-18", "validatorAddress": "validator-match", "allocationId": "6-12"})
        self.assertEqual(calls, [("status",),
            ("query", "structs", "player", "1-7", "--height", "123"),
            ("query", "structs", "guild", "0-22", "--height", "123"),
            ("query", "structs", "substation", "4-11", "--height", "123"),
            ("query", "structs", "reactor", "3-18", "--height", "123"),
            ("query", "staking", "validator", "validator-match", "--height", "123"),
            ("query", "structs", "allocation-all-by-source", "1-7", "--page-limit", "100", "--height", "123")])

    def test_wrong_objects_owners_and_missing_backing_refuse_recovery(self):
        for kind, root, field, value in (("player", "Player", "id", "1-8"),
                ("player", "Player", "guildId", "0-0"), ("player", "Player", "substationId", "4-0"),
                ("guild", "Guild", "id", "0-23"), ("guild", "Guild", "primaryReactorId", "3-0"),
                ("substation", "Substation", "owner", "1-8"), ("substation", "Substation", "id", "4-12"),
                ("reactor", "Reactor", "id", "3-19"), ("reactor", "Reactor", "validator", ""),
                ("validator", "validator", "operator_address", "wrong-validator")):
            records = grid_records()
            records[kind][root][field] = value
            with self.subTest(kind=kind, field=field), self.assertRaises(ValueError):
                recover_grid(lambda *args: {"sync_info": {"latest_block_height": "123"}} if args == ("status",) else records[args[2]], "1-7")
        records = grid_records()
        records["allocation-all-by-source"] = page()
        with self.assertRaisesRegex(ValueError, "missing"):
            recover_grid(lambda *args: {"sync_info": {"latest_block_height": "123"}} if args == ("status",) else records[args[2]], "1-7")

    def test_actual_shell_helpers_preserve_cli_flags_and_refuse_failed_json(self):
        source = Path(__file__).with_name("test_chain.sh").read_text()
        helpers = "\n".join(re.search(r"(?ms)^" + name + r"\(\) \{.*?^\}", source).group()
            for name in ("get_fixture_grid", "get_fixture_feeder"))
        with tempfile.TemporaryDirectory(prefix="fixture grid ") as directory:
            root = Path(directory)
            for name in ("fixture_grid_recovery.py", "struct_inventory.py"):
                shutil.copyfile(Path(__file__).with_name(name), root / name)
            (root / "records.json").write_text(json.dumps(grid_records()))
            fake = root / "structsd"
            fake.write_text('''#!/usr/bin/env python3
import json,os,sys
from pathlib import Path
args=sys.argv[1:]
assert args[:4]==['--output','json','--node','http://fixture.invalid']
args=args[4:]
if args==['status']:
 print(json.dumps({'sync_info':{'latest_block_height':'123'}}))
else:
 assert args[-2:]==['--height','123']
 records=json.loads(Path(__file__).with_name('records.json').read_text())
 print(json.dumps(records[args[2]]))
 if os.environ.get('FAIL_QUERY')=='1':sys.exit(1)
''')
            fake.chmod(0o755)
            script = root / "helpers.sh"
            script.write_text('set -euo pipefail\nPARAMS_QUERY="--output json --node http://fixture.invalid"\n' + helpers + '\nget_fixture_grid 1-7\nget_fixture_feeder 1-7 4-11 1-7 123\nget_fixture_feeder 1-7 4-11 1-7\n')
            environment = dict(os.environ, PATH=str(root) + os.pathsep + os.environ["PATH"])
            result = subprocess.run(["bash", str(script)], env=environment, capture_output=True, text=True)
            self.assertEqual(result.returncode, 0, result.stderr)
            lines = result.stdout.splitlines()
            self.assertEqual(json.loads(lines[0])["allocationId"], "6-12")
            self.assertEqual(lines[1:], ["6-12", "6-12"])
            result = subprocess.run(["bash", str(script)], env=dict(environment, FAIL_QUERY="1"), capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertEqual(result.stdout, "")
        self.assertNotIn("get_latest_allocation_for_source", source)
        self.assertNotIn(".Allocation[-1]", source)


if __name__ == "__main__":
    unittest.main()
