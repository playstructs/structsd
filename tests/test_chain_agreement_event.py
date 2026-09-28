"""Agreement discovery must identify the confirmed open, not an unrelated row."""
import copy
import json
from pathlib import Path
import re
import subprocess
import unittest

SOURCE = (Path(__file__).parent / "test_chain.sh").read_text()
HELPER = "\n".join(re.search(r"(?ms)^" + name + r"\(\) \{.*?^\}", SOURCE).group()
                   for name in ("get_opened_agreement_record", "get_opened_agreement_id"))
HASH = "A" * 64
RECORD = {"id": "11-9007199254740993", "providerId": "10-1", "owner": "1-2", "capacity": "50000", "allocationId": "6-9007199254740993"}
EVENT = {"type": "structs.structs.EventAgreement", "attributes": [{"key": "agreement", "value": json.dumps(RECORD)}]}
RESPONSE = {"code": 0, "txhash": HASH, "events": [EVENT]}


class AgreementEventTests(unittest.TestCase):
    def invoke(self, response, status=0, broadcast=None, command="get_opened_agreement_id 10-1 1-2 50000"):
        script = '''set -euo pipefail
payload="$1" status="$2" LAST_TX_OUTPUT="$3" GENERATOR_TEST_DIR="$4"
SETTLE_PROV_ID=10-1 PLAYER_2_ID=1-2 PLAYER_3_ID=1-3 GUILD_ID=0-1
SETTLE_CAP_A=2000 SETTLE_CAP_B=3000 SETTLE_CAP_C=2000 SETTLE_CAP_D=1000
_tx_json() { printf '%s' "$1"; }
query() {
    [ "$*" = 'query tx ''' + HASH + '''' ] || return 9
    printf '%s' "$payload"
    return "$status"
}
'''
        script += HELPER + '\n' + command + '\n'
        return subprocess.run(["bash", "-c", script, "test", response, str(status),
                               json.dumps(broadcast if broadcast is not None else {"txhash": HASH}), str(Path(__file__).parent.resolve())],
                              capture_output=True, text=True)

    def test_confirmed_matching_event_preserves_exact_id(self):
        for code in (0, "0"):
            response = copy.deepcopy(RESPONSE)
            response["code"] = code
            response["txhash"] = HASH.lower()
            response["events"].insert(0, {"type": "transfer", "attributes": []})
            result = self.invoke(json.dumps(response))
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(result.stdout.strip(), RECORD["id"])

    def test_actual_settlement_calls_select_the_intended_consumer(self):
        for label, owner, capacity in (("A", "1-2", "2000"), ("B", "1-3", "3000"),
                                        ("C", "1-2", "2000"), ("D", "0-1", "1000")):
            assignment = re.search(r"(?m)^SETTLE_AGREE_" + label + r"=.+$", SOURCE).group()
            command = assignment + '\nprintf "%s\\n" "${SETTLE_AGREE_' + label + '}"'
            if label in ("A", "B", "C", "D"):
                record_assignment = re.search(r"(?m)^SETTLE_" + label + r"_RECORD=.+$", SOURCE).group()
                command = record_assignment + "\n" + command
            for correct_owner in (True, False):
                with self.subTest(label=label, correct_owner=correct_owner):
                    response = copy.deepcopy(RESPONSE)
                    record = dict(RECORD, owner=owner if correct_owner else "1-999", capacity=capacity)
                    response["events"][0]["attributes"][0]["value"] = json.dumps(record)
                    result = self.invoke(json.dumps(response), command=command)
                    if correct_owner:
                        self.assertEqual(result.returncode, 0, result.stderr)
                        self.assertEqual(result.stdout.strip(), RECORD["id"])
                    else:
                        self.assertNotEqual(result.returncode, 0)
                        self.assertEqual(result.stdout, "")

    def test_expiry_targets_need_only_the_retained_creation_event(self):
        for label, owner, capacity in (("C", "1-2", "2000"), ("D", "0-1", "1000")):
            variables = [f"SETTLE_{label}_RECORD", f"SETTLE_AGREE_{label}", f"SETTLE_ALLOC_{label}"]
            if label == "D":
                variables.append("SETTLE_D_JSON")
            variables += [f"SETTLE_{label}_EXPIRY", f"SETTLE_{label}_END", f"SETTLE_{label}_WAIT"]
            command = "\n".join(re.search(r"(?m)^" + variable + r"=.+$", SOURCE).group() for variable in variables)
            command += '\nprintf "%s\\n" "${SETTLE_' + label + '_WAIT}"'
            for end in (None, "100", str(2**53+3)):
                response = copy.deepcopy(RESPONSE)
                record = dict(RECORD, owner=owner, capacity=capacity)
                if end is not None:
                    record["endBlock"] = end
                response["events"][0]["attributes"][0]["value"] = json.dumps(record)
                result = self.invoke(json.dumps(response), command=command)
                if end is None:
                    self.assertNotEqual(result.returncode, 0)
                    self.assertEqual(result.stdout, "")
                else:
                    self.assertEqual(result.returncode, 0, result.stderr)
                    self.assertEqual(result.stdout.strip(), str(int(end)+2))

    def test_settlement_allocation_ids_are_required_and_preserve_large_ids(self):
        for label in ("A", "B", "C", "D"):
            assignment = re.search(r"(?m)^SETTLE_ALLOC_" + label + r"=.+$", SOURCE).group()
            for allocation in (None, False, 6, "", "10-1", "6-1", "6-9007199254740993"):
                record = dict(RECORD, allocationId=allocation)
                command = "SETTLE_" + label + "_RECORD=" + "'" + json.dumps(record) + "'\n" + assignment
                command += '\nprintf "%s\\n" "${SETTLE_ALLOC_' + label + '}"'
                result = self.invoke(json.dumps(RESPONSE), command=command)
                if allocation in ("6-1", "6-9007199254740993"):
                    self.assertEqual(result.returncode, 0, result.stderr)
                    self.assertEqual(result.stdout.strip(), allocation)
                else:
                    self.assertNotEqual(result.returncode, 0)
                    self.assertEqual(result.stdout, "")

    def test_failed_ambiguous_or_mismatched_delivery_has_no_id(self):
        cases = ["", "not-json", "{}"]
        for field, value in (("code", 1), ("code", False), ("code", None), ("txhash", "B" * 64),
                             ("events", []), ("events", [EVENT, EVENT])):
            response = copy.deepcopy(RESPONSE)
            response[field] = value
            cases.append(json.dumps(response))
        for field, value in (("providerId", "10-2"), ("owner", "1-3"), ("capacity", "50001"),
                             ("capacity", 50000), ("id", "6-1"), ("id", "")):
            response = copy.deepcopy(RESPONSE)
            record = dict(RECORD, **{field: value})
            response["events"][0]["attributes"][0]["value"] = json.dumps(record)
            cases.append(json.dumps(response))
        for attrs in ([], EVENT["attributes"] * 2, [{"key": "agreement", "value": "broken"}]):
            response = copy.deepcopy(RESPONSE)
            response["events"][0]["attributes"] = attrs
            cases.append(json.dumps(response))
        for payload in cases:
            with self.subTest(payload=payload):
                result = self.invoke(payload)
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(result.stdout, "")
        for status, broadcast in ((1, {"txhash": HASH}), (0, {}), (0, {"txhash": "short"})):
            result = self.invoke(json.dumps(RESPONSE), status, broadcast)
            self.assertNotEqual(result.returncode, 0)
            self.assertEqual(result.stdout, "")


if __name__ == "__main__":
    unittest.main()
