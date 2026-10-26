"""Verify the script's actual rank-event decoder against public event shapes."""
import copy
import json
import os
from pathlib import Path
import re
import subprocess
import unittest

SOURCE = (Path(__file__).parent / "test_chain.sh").read_text()
FUNCTIONS = "\n".join(
    re.search(rf"(?ms)^{name}\(\) \{{.*?^\}}", SOURCE).group()
    for name in ("_tx_json", "assert_eq", "assert_rank_purchase_event")
)


def event(joined=False, price="10"):
    fields = dict(guildId="0-1", playerId="1-2", previousRank="101", rank="100",
                  price=price, denom="uguild.0-1", joined=joined)
    return {"type": "structs.structs.EventGuildRankPurchase", "attributes": [
        {"key": key, "value": json.dumps(value)} for key, value in fields.items()
    ]}


class RankEventTests(unittest.TestCase):
    def check(self, events, *, joined=False, price="10", valid=True, txhash="ABC"):
        env = dict(os.environ, REPLY=json.dumps({"events": events}),
                   LAST_TX_OUTPUT=json.dumps({"txhash": txhash}),
                   EXPECT_PRICE=price, EXPECT_JOINED=json.dumps(joined))
        script = r'''
set -euo pipefail
GREEN='' RED='' NC='' PASS_COUNT=0 FAIL_COUNT=0
query() { printf '%s\n' "$REPLY"; }
''' + FUNCTIONS + r'''
assert_rank_purchase_event 0-1 1-2 101 100 "$EXPECT_PRICE" "$EXPECT_JOINED"
printf 'COUNTS:%s:%s\n' "$PASS_COUNT" "$FAIL_COUNT"
'''
        result = subprocess.run(["bash", "-c", script], env=env, capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn("COUNTS:1:0" if valid else "COUNTS:0:1", result.stdout)

    def test_typed_values_defaults_and_large_integer_strings(self):
        self.check([{"type": "transfer", "attributes": []}, event()])
        self.check([event(True)], joined=True)
        omitted_false = event()
        omitted_false["attributes"] = [a for a in omitted_false["attributes"] if a["key"] != "joined"]
        self.check([omitted_false])
        for price in ("0", "9007199254740993", "18446744073709551615"):
            with self.subTest(price=price):
                self.check([event(price=price)], price=price)

    def test_wrong_fields_and_missing_or_duplicate_events_fail(self):
        for key, value in dict(guildId="0-2", playerId="1-3", previousRank="102",
                              rank="99", price="11", denom="ualpha", joined=True).items():
            with self.subTest(field=key):
                wrong = copy.deepcopy(event())
                for attribute in wrong["attributes"]:
                    if attribute["key"] == key:
                        attribute["value"] = json.dumps(value)
                self.check([wrong], valid=False)
        self.check([], valid=False)
        self.check([event(), event()], valid=False)
        self.check([event()], txhash="", valid=False)
        self.check([event()], joined=True, valid=False)

    def test_adjacent_large_prices_do_not_compare_equal(self):
        self.check([event(price="9007199254740992")], price="9007199254740993", valid=False)


if __name__ == "__main__":
    unittest.main()
