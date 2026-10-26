"""Whole-object cleanup checks must consume every page or fail explicitly."""
import json
from pathlib import Path
import re
import subprocess
import unittest

SOURCE = (Path(__file__).parent / "test_chain.sh").read_text()
HELPER = re.search(r"(?ms)^get_permission_by_object\(\) \{.*?^\}", SOURCE).group()


class PermissionPageTests(unittest.TestCase):
    def test_pages_are_complete_and_query_failures_are_not_empty_results(self):
        for fault in ("none", "camel", "ambiguous", "empty", "rpc", "later_rpc", "malformed", "missing_page", "wrong_object", "duplicate", "cycle", "bad_cursor", "false_rows", "false_cursor"):
            with self.subTest(fault=fault):
                script = r'''
set -euo pipefail
fault="$1"
query() {
    case "$*" in
        'query structs permission-by-object 6-1 --page-limit 100')
            case "$fault" in
                rpc) return 7 ;;
                ambiguous) echo '{"pagination":{"next_key":"MS0y","nextKey":""}}'; return ;;
                empty) echo '{"pagination":{}}'; return ;;
                malformed) echo 'broken'; return ;;
                missing_page) echo '{}'; return ;;
                wrong_object) echo '{"permissionRecords":[{"permissionId":"6-10@1-1","value":"4"}],"pagination":{}}'; return ;;
                false_rows) echo '{"permissionRecords":false,"pagination":{}}'; return ;;
                false_cursor) echo '{"pagination":{"next_key":false}}'; return ;;
                bad_cursor) echo '{"pagination":{"next_key":12}}'; return ;;
            esac
            echo '{"permissionRecords":[{"permissionId":"6-1@1-1","value":"4"}],"pagination":{"next_key":"MS0y"}}' ;;
        'query structs permission-by-object 6-1 --page-limit 100 --page-key MS0y')
            case "$fault" in
                later_rpc) return 7 ;;
                duplicate) echo '{"permissionRecords":[{"permissionId":"6-1@1-1","value":"4"}],"pagination":{}}'; return ;;
                cycle) echo '{"permissionRecords":[],"pagination":{"next_key":"MS0y"}}'; return ;;
            esac
            echo '{"permissionRecords":[{"permissionId":"6-1@1-2","value":"8"}],"pagination":{}}' ;;
        *) return 9 ;;
    esac
}
'''
                if fault == "camel":
                    script = script.replace("next_key", "nextKey")
                script += HELPER + '\nget_permission_by_object 6-1\n'
                result = subprocess.run(["bash", "-c", script, "test", fault], capture_output=True, text=True)
                if fault in ("none", "camel", "empty"):
                    self.assertEqual(result.returncode, 0, result.stderr)
                    rows = json.loads(result.stdout)["permissionRecords"]
                    self.assertEqual([row["permissionId"] for row in rows],
                                     ["6-1@1-1", "6-1@1-2"] if fault != "empty" else [])
                else:
                    self.assertNotEqual(result.returncode, 0)
                    self.assertEqual(result.stdout, "", "partial data must never look like a complete result")


if __name__ == "__main__":
    unittest.main()
