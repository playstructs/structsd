"""Run the actual generator funding/assertion block against an offline ledger."""
from pathlib import Path
import json
import re
import subprocess
import unittest
from tempfile import TemporaryDirectory

from test_generator_infuse import HASH, receipt
from test_generator_lifecycle import fixture

SOURCE = (Path(__file__).parent / "test_chain.sh").read_text()
ASSERT = re.search(r"(?ms)^assert_eq\(\) \{.*?^\}", SOURCE).group()
MARKER = "# Generator fuel is burned from the player vault, independently of key funds."
BLOCK = MARKER + SOURCE.split(MARKER, 1)[1].split("# Generator remains online", 1)[0]


class GeneratorVaultTests(unittest.TestCase):
    def test_capacity_and_destruction_assertions_reject_one_unit_errors(self):
        labels = ("Player 4 receives exact generator capacity", "Generator destroyed flag is set",
                  "Player 4 capacity removes all generator power including resumed fuel")
        blocks = '\n'.join(re.search(r'^assert_eq "' + label + r'".*$', SOURCE, re.M).group() for label in labels)
        for capacity, restored, destroyed, failures in ((6800000, 4800000, 'true', 0),
                (6799999, 4800000, 'true', 1), (6800001, 4800000, 'true', 1),
                (6800000, 4799999, 'true', 1), (6800000, 4800001, 'true', 1),
                (6800000, 4800000, 'false', 1), (6800000, 4800000, '', 1)):
            script = '''set -euo pipefail
GREEN='' RED='' NC='' PASS_COUNT=0 FAIL_COUNT=0
P4_CAP_BEFORE=4800000 P4_CAP_EXPECTED_INFUSED=6800000 P4_CAP_EXPECTED_DESTROYED=4800000
P4_CAP_AFTER_GEN="$1" P4_CAP_AFTER_DESTROY="$2" GEN_DESTROYED="$3"
'''
            result = subprocess.run(['bash', '-c', script + ASSERT + '\n' + blocks + '\n[ "$FAIL_COUNT" -eq "$4" ]',
                'test', str(capacity), str(restored), destroyed, str(failures)], capture_output=True, text=True)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_resumed_fuel_must_be_removed_in_addition_to_the_new_top_up(self):
        label = 'Player 4 capacity removes all generator power including resumed fuel'
        check = re.search(r'^assert_eq "' + label + r'".*$', SOURCE, re.M).group()
        for actual, failures in ((4800000, 0), (6800000, 1), (4799999, 1), (4800001, 1)):
            script = '''set -euo pipefail
GREEN='' RED='' NC='' PASS_COUNT=0 FAIL_COUNT=0
# One Alpha was already burned before this resume; the next top-up adds another.
P4_CAP_BEFORE=6800000 P4_CAP_EXPECTED_DESTROYED=4800000 P4_CAP_AFTER_DESTROY="$1"
'''
            result = subprocess.run(['bash', '-c', script + ASSERT + '\n' + check + '\n[ "$FAIL_COUNT" -eq "$2" ]',
                                    'test', str(actual), str(failures)], capture_output=True, text=True)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_exact_debit_key_preservation_and_fuel_credit(self):
        snapshot = re.search(r"(?ms)^generator_fixture_snapshot\(\) \{.*?^\}", SOURCE).group()
        for initial in (0, 53, 1000000):
            for fault in ("none", "vault", "key", "fuel", "receipt", "module", "history", "old-height"):
                with self.subTest(initial=initial, fault=fault), TemporaryDirectory() as directory:
                    path = Path(directory)
                    before = fixture(initial)
                    before['player']['Player']['primaryAddress'] = 'primary4'
                    after = fixture(initial + 1000000)
                    if fault == 'fuel':
                        after['generator']['gridAttributes']['fuel'] = str(initial + 999999)
                    tx = receipt(initial + 1000000, 6800000 + initial * 2)
                    if fault == 'receipt':
                        del tx['events'][4]  # Balances/fuel alone cannot prove that coins were burned.
                    if fault == 'old-height':
                        tx['height'] = '100'
                    for key, value in before.items():
                        (path / (key + '.json')).write_text(json.dumps(value))
                    (path / 'after.json').write_text(json.dumps(after['generator']))
                    (path / 'receipt.json').write_text(json.dumps(tx))
                    # Bank amounts may exceed Bash's signed 64-bit arithmetic.
                    balance = 2**64 + 1000000
                    script = r'''set -euo pipefail
GREEN='' RED='' NC='' PASS_COUNT=0 FAIL_COUNT=0 CALLS=0
PLAYER_4_ID=1-4 PLAYER_4_ADDRESS=key4 GENERATOR_STRUCT_ID=5-4 PLAYER_4_PLANET_ID=2-4
fault="$1" FIXTURES="$2" GENERATOR_TEST_DIR="$3" HASH="$4" BEFORE="$5" AFTER="$6"
info() { :; }
get_vault_address() { [ "$1" = 1-4 ]; echo vault4; }
fund_vault_alpha() { [ "$*" = '1-4 1000000' ]; }
get_balance_at_height() {
    [ "$2" = ualpha ]
    case "$1:$3" in
        vault4:100) echo "$BEFORE";;
        key4:100) echo 73;;
        vault4:101) if [ "$fault" = vault ]; then echo 1; else echo "$AFTER"; fi;;
        key4:101) if [ "$fault" = key ]; then echo 72; else echo 73; fi;;
        *) return 9;;
    esac
}
jqr() { echo "$1" | jq -r "$2"; }
_tx_json() { echo "$1"; }
query() {
    case "$*" in
    'query structs block-height') echo '{"blockHeight":"100"}';;
    'query structs struct 5-4 --height 100') cat "$FIXTURES/generator.json";;
    'query structs player 1-4 --height 100') cat "$FIXTURES/player.json";;
    'query structs infusion-all-by-destination 5-4 --page-limit 2 --height 100') cat "$FIXTURES/positions.json";;
    'query structs struct-type 20 --height 100') cat "$FIXTURES/catalog.json";;
    'query auth module-account structs --height 100')
        [ "$fault" != module ] || return 8
        echo '{"account":{"type":"/cosmos.auth.v1beta1.ModuleAccount","value":{"name":"structs","address":"module"}}}';;
    "query tx $HASH") [ "$CALLS" = 1 ]; cat "$FIXTURES/receipt.json";;
    'query structs struct 5-4 --height 101')
        [ "$fault" != history ] || return 8
        cat "$FIXTURES/after.json";;
    *) return 9;;
    esac
}
run_tx() {
    shift
    [ "$*" = 'tx structs struct-generator-infuse 5-4 1000000ualpha --from player_4' ]
    CALLS=$((CALLS + 1))
    LAST_TX_OUTPUT=$(printf '{"txhash":"%s"}' "$HASH")
}
'''
                    script += ASSERT + "\n" + snapshot + "\n" + BLOCK
                    script += '\n[ "$CALLS" -eq 1 ]\n[ "$((PASS_COUNT + FAIL_COUNT))" -eq 3 ]\n'
                    script += '[ "$FAIL_COUNT" -eq "$7" ]\n'
                    result = subprocess.run(
                        ["bash", "-c", script, "test", fault, directory, str(Path(__file__).parent.resolve()),
                         HASH, str(balance), str(balance - 1000000), "1" if fault in ('vault', 'key', 'fuel') else "0"],
                        capture_output=True, text=True,
                    )
                    self.assertEqual(result.returncode == 0, fault in ('none', 'vault', 'key', 'fuel'), result.stdout + result.stderr)


if __name__ == "__main__":
    unittest.main()
