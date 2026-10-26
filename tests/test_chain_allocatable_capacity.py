"""Catch allocation-vs-building headroom confusion in the real Phase 15 block."""
import json
from pathlib import Path
import re
import subprocess
import unittest

ROOT = Path(__file__).parent
SOURCE = (ROOT / "test_chain.sh").read_text()
ASSERT = re.search(r"(?ms)^assert_eq\(\) \{.*?^\}", SOURCE).group()
MARKER = 'P4_ALLOCATABLE=$(echo'
BLOCK = MARKER + SOURCE.split(MARKER, 1)[1].split('# ─── Now destroy the generator', 1)[0]


class AllocatableCapacityTests(unittest.TestCase):
    def test_actual_phase_uses_allocation_load_and_exact_integers(self):
        for capacity, load, structs_load, expected in (
            (100, 90, 0, 10), (100, 100, 0, 0), (100, 0, 150, 100),
            (5, 10, 0, 0), (2**64 - 1, 0, 0, 2**64 - 1),
            (2**53 + 1, 2**53, 0, 1),
        ):
            with self.subTest(capacity=capacity, load=load, structs_load=structs_load):
                payload = {"gridAttributes": {"capacity": str(capacity), "load": str(load), "structsLoad": str(structs_load), "connectionCapacity": "999999"}}
                script = '''set -euo pipefail
GREEN='' RED='' NC='' PASS_COUNT=0 FAIL_COUNT=0
GENERATOR_TEST_DIR="$1" P4_JSON="$2"
info() { :; }
'''
                script += ASSERT + "\n" + BLOCK
                script += '\n[ "$P4_AVAIL_CAP" = "$3" ]\n[ "$FAIL_COUNT" -eq "$4" ]\n'
                result = subprocess.run(["bash", "-c", script, "test", str(ROOT.resolve()), json.dumps(payload), str(expected), "0" if expected else "1"], capture_output=True, text=True)
                self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_lifecycle_sizes_use_free_capacity_without_integer_overflow(self):
        block = 'P5_LIFECYCLE_CAPACITY=$(echo' + SOURCE.split('P5_LIFECYCLE_CAPACITY=$(echo', 1)[1].split('# Create with half capacity', 1)[0]
        for capacity, load in ((100, 60), (4, 0), (2**53 + 5, 1), (2**64 - 1, 0)):
            with self.subTest(capacity=capacity, load=load):
                payload = {"gridAttributes": {"capacity": str(capacity), "load": str(load),
                                               "structsLoad": "999", "connectionCapacity": "888"}}
                available = capacity - load
                script = 'set -euo pipefail\nALLOCATION_TEST_DIR="$1" P5_JSON="$2" PLAYER_5_ID=1-5\ninfo() { :; }\n'
                script += block
                script += '\n[ "$P5_CAP" = "$3" ] && [ "$HALF_CAP" = "$4" ] && [ "$QUARTER_CAP" = "$5" ] && [ "$THREE_QUARTER_CAP" = "$6" ]\n'
                result = subprocess.run(["bash", "-c", script, "test", str(ROOT.resolve()), json.dumps(payload),
                                         str(available), str(available // 2), str(available // 4), str(available * 3 // 4)],
                                        capture_output=True, text=True)
                self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_lifecycle_refuses_insufficient_primary_headroom(self):
        for capacity, load in ((0, 0), (100, 97), (100, 100), (100, 101)):
            result = subprocess.run(["python3", str(ROOT / "player_allocatable_capacity.py"), "--lifecycle"],
                                    input=json.dumps({"gridAttributes": {"capacity": str(capacity), "load": str(load)}}),
                                    capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertEqual(result.stdout, "")

    def test_malformed_queries_cannot_report_headroom(self):
        for payload in ({}, {"gridAttributes": None}, {"gridAttributes": {"capacity": True}},
                        {"gridAttributes": {"load": "-1"}}, {"gridAttributes": {"capacity": str(2**64)}}):
            result = subprocess.run(["python3", str(ROOT / "player_allocatable_capacity.py")], input=json.dumps(payload), capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertEqual(result.stdout, "")


if __name__ == "__main__":
    unittest.main()
