"""The shared settlement substation must have actual, uint64-safe headroom."""
import json
from pathlib import Path
import re
import shutil
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).parent
SOURCE = (ROOT / "test_chain.sh").read_text()
ASSERT = re.search(r"(?ms)^assert_eq\(\) \{.*?^\}", SOURCE).group()
BLOCK = SOURCE.split("# ─── settlement fixture headroom ───", 1)[1].split(
    "# ─── settlement fixture headroom complete ───", 1)[0]


class SettlementHeadroomTests(unittest.TestCase):
    def test_resumed_settlement_initializes_its_own_helper_path(self):
        # Resume skips phase 15, so its directory variable cannot be a hidden
        # prerequisite. Execute the actual phase-16 startup and headroom block
        # from a script path with spaces, outside the repository working dir.
        startup = SOURCE.split('section "PHASE 16: Provider & Agreement System"', 1)[1].split('# provider-create', 1)[0]
        for prior in ('unset GENERATOR_TEST_DIR', 'GENERATOR_TEST_DIR=/stale/generator/phase'):
            with self.subTest(prior=prior), tempfile.TemporaryDirectory(prefix='settlement resume ') as temporary:
                directory = Path(temporary)
                shutil.copyfile(ROOT / 'player_allocatable_capacity.py', directory / 'player_allocatable_capacity.py')
                script = directory / 'test_chain.sh'
                script.write_text('''set -euo pipefail
GREEN='' RED='' NC='' PASS_COUNT=0 FAIL_COUNT=0
PLAYER_1_ID=1-1 PLAYER_2_ID=1-2 PLAYER_3_ID=1-3
SUBSTATION_ID=4-1 SETTLE_CAP_A=2000 SETTLE_CAP_B=3000
get_vault_address() { printf 'vault-%s' "$1"; }
info() { :; }
query() {
  [ "$*" = 'query structs substation 4-1' ] || return 9
  echo '{"Substation":{"id":"4-1"},"gridAttributes":{"capacity":"10000","load":"4000"}}'
}
'''+prior+'\n'+startup+'\n'+ASSERT+'\n'+BLOCK+'''
[ "$SETTLE_SUB_FREE" = 5000 ]
[ "$SETTLE_SUB_HAS_ROOM" = true ]
[ "$FAIL_COUNT" = 0 ]
echo RESUMED
''')
                result = subprocess.run(['bash', str(script)], cwd='/', capture_output=True, text=True)
                self.assertEqual(result.returncode, 0, result.stdout+result.stderr)
                self.assertIn('RESUMED', result.stdout)

    def invoke(self, payload, status=0):
        script = '''set -euo pipefail
GREEN='' RED='' NC='' PASS_COUNT=0 FAIL_COUNT=0
GENERATOR_TEST_DIR="$1" payload="$2" status="$3" SUBSTATION_ID=4-1 SETTLE_CAP_A=2000 SETTLE_CAP_B=3000
info() { :; }
query() { [ "$*" = 'query structs substation 4-1' ]; printf '%s' "$payload"; return "$status"; }
'''
        script += ASSERT + '\n' + BLOCK + '\nprintf "RESULT:%s:%s:%s\\n" "$SETTLE_SUB_FREE" "$SETTLE_SUB_HAS_ROOM" "$FAIL_COUNT"\n'
        return subprocess.run(["bash", "-c", script, "test", str(ROOT.resolve()), json.dumps(payload), str(status)], capture_output=True, text=True)

    def test_threshold_and_brownout_use_exact_arithmetic(self):
        for capacity, load in ((5000, 0), (5000, 1), (10000, 5000), (100, 200),
                               (2**64-1, 0), (2**64-1, 2**64-5001), (2**53+5000, 2**53+1)):
            with self.subTest(capacity=capacity, load=load):
                result = self.invoke({"Substation": {"id": "4-1"}, "gridAttributes": {
                    "capacity": str(capacity), "load": str(load), "structsLoad": "999999"}})
                free = max(capacity*9//10-load, 0)
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertIn(f'RESULT:{free}:{str(free >= 5000).lower()}:{int(free < 5000)}', result.stdout)

    def test_invalid_or_unavailable_grid_fails(self):
        for payload in ({}, {"Substation": {"id": "4-2"}, "gridAttributes": {}},
                        {"Substation": {"id": "4-1"}},
                        {"Substation": {"id": "4-1"}, "gridAttributes": {"capacity": str(2**64)}},
                        {"Substation": {"id": "4-1"}, "gridAttributes": {"load": False}}):
            result = self.invoke(payload)
            self.assertNotEqual(result.returncode, 0)
            self.assertNotIn('RESULT:', result.stdout)
        result = self.invoke({"Substation": {"id": "4-1"}, "gridAttributes": {}}, status=1)
        self.assertNotEqual(result.returncode, 0)


if __name__ == "__main__":
    unittest.main()
