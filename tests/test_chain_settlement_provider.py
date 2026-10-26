"""Settlement equations need the intended provider terms and distinct pools."""
import copy
import json
from pathlib import Path
import re
import subprocess
import unittest

SOURCE = (Path(__file__).parent / "test_chain.sh").read_text()
ASSERT = re.search(r"(?ms)^assert_eq\(\) \{.*?^\}", SOURCE).group()
BLOCK = SOURCE.split("# ─── settlement provider terms and pools ───", 1)[1].split(
    "# ─── settlement provider terms and pools complete ───", 1)[0]
PROVIDER = {"id": "10-1", "owner": "0-1", "substationId": "4-1", "rate": {"denom": "ualpha", "amount": "1"},
            "accessPolicy": "openMarket", "capacityMinimum": "1000", "capacityMaximum": "100000",
            "durationMinimum": "5", "durationMaximum": "100000", "providerCancellationPenalty": "0.500000000000000000",
            "consumerCancellationPenalty": "0.250000000000000000"}


class SettlementProviderTests(unittest.TestCase):
    def test_all_terms_and_pool_associations_are_checked(self):
        cases = [("valid", {"Provider": PROVIDER}, "none", True),
                 ("missing provider", {}, "none", False)]
        for field in PROVIDER:
            provider = copy.deepcopy(PROVIDER)
            provider[field] = {"amount": "2", "denom": "uother"} if field == "rate" else "wrong"
            cases.append((field, {"Provider": provider}, "none", False))
        provider = dict(PROVIDER)
        del provider["accessPolicy"]
        cases.append(("omitted open market", {"Provider": provider}, "none", True))
        cases.extend((fault, {"Provider": PROVIDER}, fault, False) for fault in
                     ("collateral_owner", "earnings_owner", "empty_address", "duplicate", "same_pool", "rpc"))
        for name, payload, fault, succeeds in cases:
            with self.subTest(name=name):
                script = r'''
set -euo pipefail
GREEN='' RED='' NC='' PASS_COUNT=0 FAIL_COUNT=0 SETTLE_PROV_ID=10-1 GUILD_ID=0-1 SUBSTATION_ID=4-1
payload="$1" fault="$2"
query() {
    case "$*" in
        'query structs provider 10-1') echo "$payload" ;;
        'query structs provider-collateral-address 10-1'|'query structs provider-earnings-address 10-1')
            if [ "$fault" = rpc ]; then return 9; fi
            OWNER=10-1
            if [ "$3" = provider-collateral-address ]; then
                ADDRESS=collateral
                if [ "$fault" = collateral_owner ]; then OWNER=10-2; fi
            else
                ADDRESS=earnings
                if [ "$fault" = earnings_owner ]; then OWNER=10-2; fi
                if [ "$fault" = same_pool ]; then ADDRESS=collateral; fi
            fi
            if [ "$fault" = empty_address ]; then ADDRESS=''; fi
            if [ "$fault" = duplicate ]; then
                echo '{"internalAddressAssociation":[{"address":"x","objectId":"10-1"},{"address":"y","objectId":"10-1"}]}'
            else
                printf '{"internalAddressAssociation":[{"address":"%s","objectId":"%s"}]}' "$ADDRESS" "$OWNER"
            fi ;;
        *) return 9 ;;
    esac
}
'''
                script += ASSERT + '\n' + BLOCK + '\n[ "$PASS_COUNT" -eq 2 ]\n[ "$FAIL_COUNT" -eq 0 ]\n'
                result = subprocess.run(["bash", "-c", script, "test", json.dumps(payload), fault], capture_output=True, text=True)
                if succeeds:
                    self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                else:
                    self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)


if __name__ == "__main__":
    unittest.main()
