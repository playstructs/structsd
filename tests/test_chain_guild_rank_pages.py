"""Guild rank cleanup must read every register page before reporting no grants."""
import json
from pathlib import Path
import re
import subprocess
import unittest

SOURCE = (Path(__file__).parent / "test_chain.sh").read_text()
HELPER = re.search(r"(?ms)^get_guild_rank_permission_by_object\(\) \{.*?^\}", SOURCE).group()


class GuildRankPageTests(unittest.TestCase):
    def test_multiple_permission_bits_and_later_guilds_are_retained(self):
        first = [{"objectId": "10-1", "guildId": "0-1", "permissions": bit, "rank": "3"} for bit in ("4", "8")]
        last = {"objectId": "10-1", "guildId": "0-2", "permissions": "4", "rank": "1"}
        for fault in ("none", "camel", "ambiguous", "empty", "rpc", "later_rpc", "missing", "false_rows", "wrong_object", "duplicate", "cycle", "missing_rank"):
            with self.subTest(fault=fault):
                page1 = {"guildRankPermissionRecords": first, "pagination": {"next_key": "MC0y"}}
                page2 = {"guild_rank_permission_records": [dict(last)], "pagination": {}}
                if fault == "camel":
                    page1["pagination"] = {"nextKey": "MC0y"}
                elif fault == "ambiguous":
                    page1["pagination"]["nextKey"] = ""
                elif fault == "empty":
                    page1 = {"pagination": {}}
                elif fault == "missing":
                    page1 = {}
                elif fault == "false_rows":
                    page1["guildRankPermissionRecords"] = False
                elif fault == "wrong_object":
                    page2["guild_rank_permission_records"][0]["objectId"] = "10-10"
                elif fault == "duplicate":
                    page2["guild_rank_permission_records"] = [first[0]]
                elif fault == "cycle":
                    page2 = {"pagination": {"next_key": "MC0y"}}
                elif fault == "missing_rank":
                    del page2["guild_rank_permission_records"][0]["rank"]
                script = r'''
set -euo pipefail
fault="$1" first="$2" second="$3"
query() {
    case "$*" in
        'query structs guild-rank-permission-by-object 10-1 --page-limit 100')
            [ "$fault" != rpc ] || return 7
            printf '%s' "$first" ;;
        'query structs guild-rank-permission-by-object 10-1 --page-limit 100 --page-key MC0y')
            [ "$fault" != later_rpc ] || return 7
            printf '%s' "$second" ;;
        *) return 9 ;;
    esac
}
'''
                script += HELPER + '\nget_guild_rank_permission_by_object 10-1\n'
                result = subprocess.run(["bash", "-c", script, "test", fault, json.dumps(page1), json.dumps(page2)],
                                        capture_output=True, text=True)
                if fault in ("none", "camel", "empty"):
                    self.assertEqual(result.returncode, 0, result.stderr)
                    self.assertEqual(json.loads(result.stdout)["guildRankPermissionRecords"],
                                     first + [last] if fault != "empty" else [])
                else:
                    self.assertNotEqual(result.returncode, 0)
                    self.assertEqual(result.stdout, "")


if __name__ == "__main__":
    unittest.main()
