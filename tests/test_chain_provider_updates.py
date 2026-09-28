"""Exercise the actual advertised-provider update block with state/RPC faults."""
from pathlib import Path
import re
import subprocess
import unittest

SOURCE = (Path(__file__).parent / "test_chain.sh").read_text()
ASSERT = re.search(r"(?ms)^assert_eq\(\) \{.*?^\}", SOURCE).group()
BLOCK = SOURCE.split("# ─── provider-update-access-policy ───", 1)[1].split(
    "# ─── agreement-open: Player 2 opens an agreement ───", 1)[0]


class ProviderUpdatePhaseTests(unittest.TestCase):
    def test_each_update_and_unrelated_terms_are_checked(self):
        for stage in range(1, 7):
            for fault in ("none", "omitted_open", "dropped_update", "other_field", "missing_provider", "wrong_provider", "rpc_error", "false_policy"):
                with self.subTest(stage=stage, fault=fault):
                    # duration minimum is intentionally unchanged by this phase;
                    # retaining its old value is not a fault in that one case.
                    if stage == 5 and fault == "dropped_update":
                        continue
                    script = r'''
set -euo pipefail
GREEN='' RED='' NC='' PASS_COUNT=0 FAIL_COUNT=0 CALLS=0 PROVIDER_ID=11-1
stage="$1" fault="$2"
STATE='{"id":"11-1","owner":"1-1","substationId":"4-1","accessPolicy":"guildMarket","rate":{"denom":"ualpha","amount":"1"},"capacityMinimum":"100000","capacityMaximum":"20000000","durationMinimum":"1","durationMaximum":"100000"}'
query() {
    [ "$*" = 'query structs provider 11-1' ]
    if [ "$CALLS" -eq "$stage" ]; then
        case "$fault" in
            rpc_error) return 9 ;;
            missing_provider) echo '{}'; return ;;
            wrong_provider) echo "$STATE" | jq '{Provider: (.id = "11-99")}'; return ;;
            omitted_open)
                if [ "$CALLS" -eq 1 ]; then echo "$STATE" | jq '{Provider: del(.accessPolicy)}'; return; fi ;;
            false_policy) echo "$STATE" | jq '{Provider: (.accessPolicy = false)}'; return ;;
        esac
    fi
    echo "$STATE" | jq '{Provider: .}'
}
run_tx() {
    shift
    CALLS=$((CALLS + 1))
    [ "$1 $2" = 'tx structs' ]
    [ "$4 $6 $7" = '11-1 --from alice' ]
    case "$3" in
        provider-update-access-policy)
            FIELD=accessPolicy
            case "$5" in open-market) VALUE=openMarket ;; guild-market) VALUE=guildMarket ;; *) return 8 ;; esac ;;
        provider-update-capacity-minimum) FIELD=capacityMinimum VALUE="$5" ;;
        provider-update-capacity-maximum) FIELD=capacityMaximum VALUE="$5" ;;
        provider-update-duration-minimum) FIELD=durationMinimum VALUE="$5" ;;
        provider-update-duration-maximum) FIELD=durationMaximum VALUE="$5" ;;
        *) return 8 ;;
    esac
    if ! { [ "$CALLS" -eq "$stage" ] && [ "$fault" = dropped_update ]; }; then
        STATE=$(echo "$STATE" | jq --arg field "$FIELD" --arg value "$VALUE" '.[$field] = $value')
    fi
    if [ "$CALLS" -eq "$stage" ] && [ "$fault" = other_field ]; then
        STATE=$(echo "$STATE" | jq '.rate.amount = "2"')
    fi
}
'''
                    script += ASSERT + "\n" + BLOCK
                    script += '\n[ "$CALLS" -eq 6 ]\n[ "$PASS_COUNT" -eq 6 ]\n[ "$FAIL_COUNT" -eq 0 ]\n'
                    result = subprocess.run(["bash", "-c", script, "test", str(stage), fault],
                                            capture_output=True, text=True)
                    if fault in ("none", "omitted_open"):
                        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                    else:
                        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)


if __name__ == "__main__":
    unittest.main()
