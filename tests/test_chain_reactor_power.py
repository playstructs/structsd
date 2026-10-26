"""Regression for flat backing across defusion and cancellation."""
import json
from pathlib import Path
import re
import subprocess
import unittest

from verify_reactor_defusion_cycle import verify_cycle


def fixture():
    def position(fuel, defusing):
        return {"Infusion": {
            "destinationId": "3-1", "address": "vault", "playerId": "1-2",
            "ownerId": "1-2", "fuel": str(fuel), "power": str(fuel),
            "ratio": "1", "defusing": str(defusing),
        }}
    return {
        "infused": position(2000000, 100),
        "defused": position(1500000, 500100),
        "cancelled": position(2000000, 100),
    }


class ReactorCycleTests(unittest.TestCase):
    def test_flat_power_and_existing_unbonding_are_preserved(self):
        verify_cycle(fixture(), "3-1", "vault", "1-2", 500000)

    def test_rejects_incomplete_accounting(self):
        for phase, field, value in (
            ("cancelled", "power", "1600000"),
            ("cancelled", "fuel", "1999999"),
            ("cancelled", "defusing", "0"),
            ("defused", "defusing", "0"),
            ("infused", "address", "signing-key"),
            ("cancelled", "ownerId", "0-1"),
            ("cancelled", "ratio", "0"),
            ("cancelled", "fuel", 2000000),
            ("cancelled", "fuel", str(2**64)),
        ):
            with self.subTest(phase=phase, field=field):
                document = fixture()
                document[phase]["Infusion"][field] = value
                with self.assertRaises(ValueError):
                    verify_cycle(document, "3-1", "vault", "1-2", 500000)

    def test_self_consistent_but_wrong_backing_is_rejected(self):
        for phase in ("defused", "cancelled"):
            document = fixture()
            document[phase]["Infusion"].update(fuel="1999999", power="1999999")
            with self.assertRaises(ValueError):
                verify_cycle(document, "3-1", "vault", "1-2", 500000)

    def test_cli_exit_status(self):
        path = Path(__file__).with_name("verify_reactor_defusion_cycle.py")
        for document, code in ((fixture(), 0), ({}, 1)):
            result = subprocess.run(["python3", str(path), "3-1", "vault", "1-2", "500000"],
                input=json.dumps(document), capture_output=True, text=True)
            self.assertEqual(result.returncode, code, result.stderr)
            self.assertEqual(result.stdout.strip(), "verified" if code == 0 else "")

    def test_live_phase_invokes_verifier_and_counts_failure(self):
        source = Path(__file__).with_name("test_chain.sh").read_text()
        assertion = re.search(r"(?ms)^assert_eq\(\) \{.*?^\}", source).group()
        proof = source.split("        if ! P2_CYCLE_PROOF=", 1)[1].split("        P2_CANCELLED_ENTRIES=", 1)[0]
        proof = "if ! P2_CYCLE_PROOF=" + proof
        for valid in (True, False):
            document = fixture()
            if not valid:
                document["cancelled"]["Infusion"].update(power="1375000")
            script = '''set -euo pipefail
GREEN='' RED='' NC=''
PASS_COUNT=0 FAIL_COUNT=0
REACTOR_ID=3-1 PLAYER_2_VAULT=vault PLAYER_2_ID=1-2
REACTOR_TEST_DIR="$1"
P2_INFUSION_INFUSED="$2" P2_INFUSION_DEFUSED="$3"
P2_INFUSION_CANCELLED="$4"
'''
            script += assertion + "\n" + proof + '\n[ "$FAIL_COUNT" -eq "$5" ]\n[ "$((PASS_COUNT + FAIL_COUNT))" -eq 1 ]\n'
            result = subprocess.run(["bash", "-c", script, "test", str(Path(__file__).parent),
                *(json.dumps(document[name]) for name in ("infused", "defused", "cancelled")),
                "0" if valid else "1"], capture_output=True, text=True)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)


if __name__ == "__main__":
    unittest.main()
