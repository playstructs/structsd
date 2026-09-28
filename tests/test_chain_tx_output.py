"""Exercise the actual shell helpers with fake CLI/query functions; no node."""
import json
import os
from pathlib import Path
import re
import subprocess
import unittest


SOURCE = (Path(__file__).parent / "test_chain.sh").read_text()
NAMES = [
    "_tx_json", "_tx_error_line", "_tx_delivered", "_tx_outcome",
    "_tx_outcome_known", "_check_tx_output", "run_tx_expect_fail",
    "run_tx_expect_fail_noauto", "run_tx_expect_permission_denied",
    "run_tx_expect_reject_matching", "run_tx_tolerate_reject",
]
FUNCTIONS = "\n".join(
    re.search(rf"(?ms)^{name}\(\) \{{.*?^\}}", SOURCE).group() for name in NAMES
)


def shell(body, reply=None, query=None):
    environment = dict(os.environ, CLI_REPLY=reply if isinstance(reply, str) else json.dumps(reply), QUERY_REPLY=json.dumps(query))
    script = """
set -euo pipefail
RED='' GREEN='' YELLOW='' BOLD='' NC=''
PARAMS_TX='' PARAMS_TX_NOFEE='' SLEEP=0 TX_CONFIRM_TIMEOUT=1
FAIL_COUNT=0 PASS_COUNT=0
info() { :; }
sleep() { :; }
structsd() { printf '%s\\n' "$CLI_REPLY"; }
query() { printf '%s\\n' "$QUERY_REPLY"; }
""" + FUNCTIONS + "\n" + body
    return subprocess.check_output(["bash", "-c", script], env=environment, text=True)


class TransactionOutputTests(unittest.TestCase):
    def test_only_matching_committed_result_proves_success(self):
        for query in (
            {"txhash": "ABC", "height": "9", "code": 0},
            {"txhash": "ABC", "height": "9"},
        ):
            self.assertEqual(shell('_tx_outcome "$CLI_REPLY"', {"code": 0, "txhash": "ABC"}, query), "ok|\n")
        for query in (None, {"txhash": "OTHER", "height": "9"}, {"txhash": "ABC", "height": "0"}, {"txhash": "ABC", "height": "9", "code": "bad"}):
            self.assertTrue(shell('_tx_outcome "$CLI_REPLY"', {"code": 0, "txhash": "ABC"}, query).startswith("unknown|"))

    def test_missing_hash_and_timeout_fail_every_expectation_helper(self):
        handlers = [
            'run_tx_expect_fail example', 'run_tx_expect_fail_noauto example',
            'run_tx_expect_permission_denied example',
            'run_tx_expect_reject_matching example ".*"',
            'run_tx_tolerate_reject example',
        ]
        for response in ({"code": 0}, {"code": 0, "txhash": "ABC"}, {"code": "bad"}, {}, "", "Error: dial tcp: connection refused"):
            for handler in handlers:
                with self.subTest(response=response, handler=handler):
                    output = shell(handler + '\nprintf "COUNTS:%s:%s\\n" "$FAIL_COUNT" "$PASS_COUNT"', response)
                    self.assertIn("COUNTS:1:0", output)
        output = shell('_check_tx_output "$CLI_REPLY"\nprintf "COUNTS:%s\\n" "$FAIL_COUNT"', {"code": 0})
        self.assertIn("COUNTS:1", output)

    def test_actual_rejection_still_counts_as_rejection(self):
        query = {"txhash": "ABC", "height": "9", "code": 5, "raw_log": "permission denied"}
        self.assertEqual(shell('_tx_outcome "$CLI_REPLY"', {"code": 0, "txhash": "ABC"}, query), "rejected|permission denied\n")
        self.assertEqual(shell('_tx_outcome "$CLI_REPLY"', {"code": 5, "raw_log": "permission denied"}), "rejected|permission denied\n")

    def test_permission_checks_require_permission_refusal_without_panic(self):
        for reason, expected in (
            ("calling player (1-2) has no guild_update_endpoint permission on guild (0-1)", "COUNTS:0:1"),
            ("permission denied", "COUNTS:0:1"),
            ("unauthorized", "COUNTS:0:1"),
            ("out of gas", "COUNTS:1:0"),
            ("object not found", "COUNTS:1:0"),
            ("parameter rank validation failed: zero", "COUNTS:1:0"),
            ("unknown flag: --permission", "COUNTS:1:0"),
            ("panic while checking permission denied", "COUNTS:1:0"),
        ):
            with self.subTest(reason=reason):
                output = shell(
                    'run_tx_expect_permission_denied example\nprintf "COUNTS:%s:%s\\n" "$FAIL_COUNT" "$PASS_COUNT"',
                    {"code": 5, "raw_log": reason},
                )
                self.assertIn(expected, output)


if __name__ == "__main__":
    unittest.main()
