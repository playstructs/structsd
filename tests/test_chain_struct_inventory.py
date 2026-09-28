"""Complete inventory and exact selection controls for resumed/combat fixtures."""
import base64
import copy
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import tempfile
import unittest

from struct_inventory import collect, select


def row(index, owner="1-2", kind="9"):
    return {"id": f"5-{index}", "index": str(index), "type": kind, "owner": owner}


def cursor(offset):
    return base64.b64encode(str(offset).encode()).decode()


def page(rows=None, following=None):
    return {"Struct": [] if rows is None else rows, "pagination": {"next_key": following}}


class StructInventoryTests(unittest.TestCase):
    def test_all_pages_share_one_height_despite_newer_blocks(self):
        rows = sorted([row(n) for n in range(1, 202)], key=lambda item: item["id"])
        calls = []
        def query(*args):
            calls.append(args)
            if args == ("status",):
                return {"sync_info": {"latest_block_height": "123"}}
            expected = ["query", "structs", "struct-all", "--page-limit", "100", "--height", "123"]
            offset = (len(calls) - 2) * 100
            if offset:
                expected.extend(("--page-key", cursor(offset)))
            self.assertEqual(args, tuple(expected))
            return page(rows[offset:offset + 100], cursor(offset + 100) if offset + 100 < len(rows) else None)
        result = collect(query)
        self.assertEqual(result, {"Struct": rows, "pagination": {}, "queryHeight": "123"})
        self.assertEqual(len(calls), 4)
        self.assertEqual(select(result, "1-2", "9", newest=True), "5-201")

    def test_empty_native_inventory_and_cursor_encodings(self):
        for response in ({"pagination": {}}, {"Struct": None, "pagination": {}}, page()):
            result = collect(lambda *args: {"sync_info": {"latest_block_height": "1"}} if args == ("status",) else response)
            self.assertEqual(result["Struct"], [])
            self.assertEqual(select(result, "1-2", "9"), "")
        pages = iter([{"Struct": [row(1)], "pagination": {"nextKey": cursor(1)}}, page([row(2)])])
        result = collect(lambda *args: {"sync_info": {"latest_block_height": "123"}} if args == ("status",) else next(pages), page_size=1)
        self.assertEqual(len(result["Struct"]), 2)

    def test_refuses_bad_pages_cycles_duplicates_and_incomplete_budget(self):
        malformed = [None, [], {}, {"Struct": [], "pagination": None},
            {"struct": [row(1)], "pagination": {}}, page(False), page([None]), page([{}]),
            page([dict(row(1), id="5-2")]), page([dict(row(1), index=1)]),
            page([dict(row(1), type="01")]), page([dict(row(1), owner="")]),
            page([dict(row(1), index=str(2**64))]), page([row(1)], "not base64"),
            page([row(1)], False), {"Struct": [], "pagination": {"next_key": "", "nextKey": ""}}]
        for response in malformed:
            with self.subTest(response=response):
                with self.assertRaises(ValueError):
                    collect(lambda *args: {"sync_info": {"latest_block_height": "123"}} if args == ("status",) else response)
        for pages in ([page([row(1)], cursor(1)), page([row(1)])],
                [page([row(1)], cursor(1)), page([row(2)], cursor(1))],
                [page([row(1)], cursor(1)), page([row(2)], cursor(2))]):
            responses = iter(pages)
            with self.assertRaises(ValueError):
                collect(lambda *args: {"sync_info": {"latest_block_height": "123"}} if args == ("status",) else next(responses), max_pages=2)
        with self.assertRaises(ValueError):
            collect(lambda *args: {"sync_info": {"latest_block_height": "123"}} if args == ("status",) else page([row(1), row(2)]), page_size=1)

    def test_status_and_query_failures_do_not_return_partial_inventory(self):
        for height in (None, False, 0, "0", "01", "-1", "1.0", str(2**63)):
            with self.assertRaises(ValueError):
                collect(lambda *args: {"sync_info": {"latest_block_height": height}})
        visited = []
        def failing_query(*args):
            visited.append(args)
            if args == ("status",):
                return {"sync_info": {"latest_block_height": "123"}}
            if "--page-key" not in args:
                return page([row(1)], cursor(1))
            raise subprocess.CalledProcessError(1, args)
        with self.assertRaises(subprocess.CalledProcessError):
            collect(failing_query)
        self.assertEqual(len(visited), 3)

    def test_selection_uses_exact_indices_and_refuses_partial_or_corrupt_input(self):
        indices = [2**64 - 1, 9007199254740993, 2, 9007199254740992]
        document = {"Struct": [row(n) for n in indices] + [row(1, owner="0-1")],
            "pagination": {}, "queryHeight": "123"}
        self.assertEqual(select(document, "1-2", "9"), "5-2")
        self.assertEqual(select(document, "1-2", "9", "2"), "5-9007199254740992")
        self.assertEqual(select(document, "1-2", "9", "2", newest=True), "5-9007199254740993")
        self.assertEqual(select(document, "1-2", "9", newest=True), f"5-{2**64 - 1}")
        self.assertEqual(select(document, "0-1", "9"), "5-1")
        self.assertEqual(select(document, "1-2", "9", "5"), "")
        for mutate in (lambda x: x.pop("queryHeight"),
                lambda x: x.update(pagination={"next_key": cursor(1)}),
                lambda x: x["Struct"].append(x["Struct"][0]),
                lambda x: x["Struct"][0].update(index=2**64 - 1)):
            invalid = copy.deepcopy(document)
            mutate(invalid)
            with self.assertRaises(ValueError):
                select(invalid, "1-2", "9")
        for nth in ("0", "01", "-1", str(2**64)):
            with self.assertRaises(ValueError):
                select(document, "1-2", "9", nth)

    def test_actual_shell_helpers_with_cli_pagination_and_failed_second_page(self):
        source = Path(__file__).with_name("test_chain.sh").read_text()
        names = ("get_all_structs", "find_struct_by_owner_type", "find_last_struct_by_owner_type")
        helpers = "\n".join(re.search(r"(?ms)^" + name + r"\(\) \{.*?^\}", source).group() for name in names)
        with tempfile.TemporaryDirectory(prefix="structs inventory ") as temporary:
            root = Path(temporary)
            shutil.copy2(Path(__file__).with_name("struct_inventory.py"), root / "struct_inventory.py")
            values = sorted([row(n) for n in range(1, 202)], key=lambda item: item["id"])
            (root / "rows.json").write_text(json.dumps(values))
            cli = root / "structsd"
            cli.write_text(f'''#!{sys.executable}
import base64, json, os, sys
from pathlib import Path
args = sys.argv[1:]
assert args[:4] == ['--home', 'fixture-home', '--output', 'json'], args
args = args[4:]
with open(os.environ['STRUCTS_TEST_INVENTORY_CALLS'], 'a') as log:
    log.write(json.dumps(args) + '\\n')
if args == ['status']:
    print(json.dumps({{'sync_info': {{'latest_block_height': '123'}}}}))
else:
    assert args[:7] == ['query', 'structs', 'struct-all', '--page-limit', '100', '--height', '123'], args
    offset = int(base64.b64decode(args[8])) if len(args) == 9 and args[7] == '--page-key' else 0
    rows = json.loads(Path(os.environ['STRUCTS_TEST_INVENTORY_FILE']).read_text())
    end = offset + 100
    key = base64.b64encode(str(end).encode()).decode() if end < len(rows) else None
    print(json.dumps({{'Struct': rows[offset:end], 'pagination': {{'next_key': key}}}}))
    if offset and os.environ.get('STRUCTS_TEST_INVENTORY_FAIL') == 'yes':
        sys.exit(1)
''')
            cli.chmod(0o700)
            script = root / "fixture.sh"
            script.write_text('set -euo pipefail\nPARAMS_QUERY="--home fixture-home --output json"\n' + helpers + '''
snapshot=$(get_all_structs)
oldest=$(find_struct_by_owner_type 1-2 9 1 "$snapshot")
newest=$(find_last_struct_by_owner_type 1-2 9 1 "$snapshot")
[ "$oldest" = '5-1' ]
[ "$newest" = '5-201' ]
printf '%s' "$snapshot"
''')
            environment = dict(os.environ, PATH=str(root) + os.pathsep + os.environ["PATH"],
                STRUCTS_TEST_INVENTORY_FILE=str(root / "rows.json"),
                STRUCTS_TEST_INVENTORY_CALLS=str(root / "calls.jsonl"))
            result = subprocess.run(["bash", str(script)], env=environment, capture_output=True, text=True)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(len(json.loads(result.stdout)["Struct"]), 201)
            self.assertEqual(len((root / "calls.jsonl").read_text().splitlines()), 4)
            environment["STRUCTS_TEST_INVENTORY_FAIL"] = "yes"
            result = subprocess.run(["bash", str(script)], env=environment, capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertEqual(result.stdout, "")


if __name__ == "__main__":
    unittest.main()
