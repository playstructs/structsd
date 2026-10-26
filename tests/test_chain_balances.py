"""Exercise denomination-specific shell queries without a node."""
import json
from pathlib import Path
import re
import subprocess
import unittest

SOURCE = (Path(__file__).parent / "test_chain.sh").read_text()


class BalanceQueryTests(unittest.TestCase):
    def test_exact_zero_and_large_amounts_but_no_false_zero_on_errors(self):
        for name, field, invocation, command in (
                ("get_balance", "balance", "get_balance owner ualpha", "query bank balance owner ualpha"),
                ("guild_supply", "amount", "guild_supply ualpha", "query bank total-supply-of ualpha")):
            function = re.search(rf"(?ms)^{name}\(\) \{{.*?^\}}", SOURCE).group()
            for amount in ("0", "9007199254740993"):
                reply = json.dumps({field: {"denom": "ualpha", "amount": amount}})
                result = self.run_query(function, invocation, command, reply)
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertEqual(result.stdout.strip(), amount)
            for reply in ("", "not json", "{}", json.dumps({field: {"denom": "wrong", "amount": "0"}}),
                    json.dumps({field: {"denom": "ualpha", "amount": 0}})):
                result = self.run_query(function, invocation, command, reply)
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(result.stdout, "")
            result = self.run_query(function, invocation, command,
                json.dumps({field: {"denom": "ualpha", "amount": "0"}}), "1")
            self.assertNotEqual(result.returncode, 0, "query exit status must survive valid-looking output")
            self.assertEqual(result.stdout, "", "failed query output cannot satisfy a zero-balance assertion")

    def run_query(self, function, invocation, command, reply, status="0"):
        script = 'set -euo pipefail\nexpected="$1"; reply="$2"; status="$3"\nquery() { [ "$*" = "$expected" ] || return 99; printf "%s\\n" "$reply"; return "$status"; }\n'
        return subprocess.run(["bash", "-c", script + function + "\n" + invocation,
            "test", command, reply, status], capture_output=True, text=True)


if __name__ == "__main__":
    unittest.main()
