"""The fixture must not write after an uncertain or nonempty inventory."""
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import tempfile
import unittest

from fresh_chain_inventory import ENTITIES, inventory

ROOT = Path(__file__).resolve().parent


class FreshInventoryTests(unittest.TestCase):
    def query(self, responses, calls):
        def read(*args):
            calls.append(args)
            if args == ("status",):
                return {"sync_info": {"latest_block_height": "123"}}
            entity = next(name for name in ENTITIES if args[2] == name.lower() + "-all")
            self.assertEqual(args, ("query", "structs", entity.lower() + "-all",
                "--page-limit", "2" if entity == "Player" else "1", "--height", "123"))
            response = responses.get(entity, {"pagination": {}})
            if isinstance(response, Exception):
                raise response
            return response
        return read

    def test_empty_and_initial_player_use_seven_bounded_pinned_queries(self):
        for rows in (None, [], [{"id": "1-1", "index": "1"}]):
            calls = []
            result = inventory(self.query({"Player": {"Player": rows, "pagination": {}}}, calls))
            self.assertEqual(result["counts"]["Player"], len(rows or []))
            self.assertEqual(result["height"], "123")
            self.assertEqual(len(calls), 7)

    def test_each_excess_inventory_or_rpc_failure_refuses(self):
        for entity in ENTITIES:
            for response in ({entity: [{"id": "1-1", "index": "1"}] * (2 if entity == "Player" else 1),
                              "pagination": {}}, subprocess.CalledProcessError(1, "query")):
                with self.subTest(entity=entity, response=response), self.assertRaises((ValueError, subprocess.CalledProcessError)):
                    inventory(self.query({entity: response}, []))

    def test_malformed_or_partial_envelopes_never_prove_empty(self):
        malformed = [None, [], {}, {"pagination": None}, {"pagination": {}, "player": []},
            {"Player": False, "pagination": {}}, {"Player": [None], "pagination": {}},
            {"Player": [{}], "pagination": {}},
            {"Player": [{"id": "1-1", "index": "2"}], "pagination": {}},
            {"Player": [{"id": "1-1", "index": 1}], "pagination": {}},
            {"pagination": {"next_key": "YQ=="}}, {"pagination": {"next_key": False}},
            {"pagination": {"next_key": "", "nextKey": ""}}]
        for page in malformed:
            with self.subTest(page=page), self.assertRaises(ValueError):
                inventory(self.query({"Player": page}, []))
        for status in (None, {}, {"sync_info": None}, *({"sync_info": {"latest_block_height": h}}
                for h in (None, False, 0, "0", "01", "-1", str(2**63)))):
            with self.subTest(status=status), self.assertRaises(ValueError):
                inventory(lambda *args: status)

    def test_actual_shell_guard_refuses_failures_in_direct_and_conditional_calls(self):
        helper = re.search(r'(?ms)^assert_fresh_chain\(\) \{.*?^\}', (ROOT / "test_chain.sh").read_text()).group()
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory)
            for name in ("fresh_chain_inventory.py", "struct_inventory.py"):
                shutil.copyfile(ROOT / name, path / name)
            (path / "guard.sh").write_text(helper)
            cli = path / "structsd"
            cli.write_text('''#!/usr/bin/env python3
import json, os, sys
fault = os.environ['FAULT']
if sys.argv[1:] == ['status']:
    print(json.dumps({'sync_info': {'latest_block_height': '123'}}))
elif fault == 'rpc':
    print('{"pagination":{}}')
    sys.exit(1)
elif fault == 'malformed':
    print('{}')
else:
    print('{"pagination":{}}')
''')
            cli.chmod(0o755)
            for conditional in (False, True):
                for fault in ("none", "rpc", "malformed"):
                    script = 'set -euo pipefail\nPARAMS_QUERY=""\ninfo() { echo "$*"; }\nsource "$1"\n'
                    script += ('if assert_fresh_chain; then echo authorized; else exit 7; fi\n'
                        if conditional else 'assert_fresh_chain\necho authorized\n')
                    result = subprocess.run(["bash", "-c", script, "fixture", str(path / "guard.sh")],
                        env=dict(os.environ, PATH=str(path) + os.pathsep + os.environ["PATH"], FAULT=fault),
                        text=True, capture_output=True)
                    with self.subTest(conditional=conditional, fault=fault):
                        self.assertEqual(result.returncode == 0, fault == "none", result.stderr)
                        self.assertEqual("authorized" in result.stdout, fault == "none")


if __name__ == "__main__":
    unittest.main()
