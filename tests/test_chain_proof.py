"""Exercise proof parsing without generating, importing or printing a key."""
from pathlib import Path
import re
import subprocess
import unittest

SOURCE = (Path(__file__).parent / "test_chain.sh").read_text()
FUNCTION = re.search(r"(?ms)^parse_address_proof\(\) \{.*?^\}", SOURCE).group()
PROOF = "ADDRESS=structs1abc123\nPROOF_PUBKEY=" + "a" * 66 + "\nPROOF_SIGNATURE=" + "b" * 128 + "\nPRIVKEY=" + "c" * 64


class ProofParserTests(unittest.TestCase):
    def test_complete_and_malformed_data_without_output(self):
        for value, expected in ((PROOF, 0), ("build chatter\n" + PROOF, 0),
                (PROOF.rsplit("\n", 1)[0], 1), ("ADDRESS=\n" + PROOF, 1),
                (PROOF + "\nADDRESS=structs1duplicate", 1),
                (PROOF.replace("structs1abc123", "$(printf executed)"), 1)):
            result = subprocess.run(["bash", "-c", FUNCTION + '\nparse_address_proof "$1"', "test", value], capture_output=True)
            self.assertEqual(result.returncode, expected)
            self.assertEqual(result.stdout, b"")
            self.assertEqual(result.stderr, b"")

    def test_missing_fields_do_not_reuse_previous_proof(self):
        script = FUNCTION + '\nparse_address_proof "$1" || exit 9\nparse_address_proof "ADDRESS=structs1new"'
        result = subprocess.run(["bash", "-c", script, "test", PROOF], capture_output=True)
        self.assertEqual(result.returncode, 1)


if __name__ == "__main__":
    unittest.main()
