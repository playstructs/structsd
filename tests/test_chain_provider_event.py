"""Provider discovery must match this creation's owner, station and delivery."""
import copy
import json
from pathlib import Path
import re
import subprocess
import unittest

SOURCE = (Path(__file__).parent / "test_chain.sh").read_text()
HELPER = re.search(r"(?ms)^get_created_provider_id\(\) \{.*?^\}", SOURCE).group()
HASH = "A" * 64
RECORD = {"id": "10-9007199254740993", "owner": "0-1", "substationId": "4-1"}
EVENT = {"type": "structs.structs.EventProvider", "attributes": [{"key": "provider", "value": json.dumps(RECORD)}]}
RESPONSE = {"code": 0, "txhash": HASH, "events": [EVENT]}


class ProviderEventTests(unittest.TestCase):
    def invoke(self, response, status=0, broadcast=None, command="get_created_provider_id 0-1 4-1"):
        script = '''set -euo pipefail
payload="$1" status="$2" LAST_TX_OUTPUT="$3" GUILD_ID=0-1 PLAYER_1_ID=1-1 SUBSTATION_ID=4-1
_tx_json() { printf '%s' "$1"; }
query() {
    [ "$*" = 'query tx ''' + HASH + '''' ] || return 9
    printf '%s' "$payload"
    return "$status"
}
'''
        script += HELPER + '\n' + command + '\n'
        return subprocess.run(["bash", "-c", script, "test", response, str(status),
                               json.dumps(broadcast if broadcast is not None else {"txhash": HASH})],
                              capture_output=True, text=True)

    def test_actual_creation_sites_preserve_exact_event_id(self):
        assignments = re.findall(r"(?m)^(?:PROVIDER_ID|SETTLE_PROV_ID)=\$\(get_created_provider_id .+$", SOURCE)
        self.assertEqual(len(assignments), 3)
        for assignment, owner in zip(assignments, ("0-1", "1-1", "0-1")):
            for code in (0, "0"):
                response = copy.deepcopy(RESPONSE)
                response["code"] = code
                response["txhash"] = HASH.lower()
                response["events"][0]["attributes"][0]["value"] = json.dumps(dict(RECORD, owner=owner))
                response["events"].append({"type": "transfer", "attributes": []})
                variable = assignment.split("=", 1)[0]
                result = self.invoke(json.dumps(response), command=assignment + '\nprintf "%s\\n" "${' + variable + '}"')
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertEqual(result.stdout.strip(), RECORD["id"])

    def test_ambiguous_or_wrong_creation_has_no_id(self):
        cases = ["", "invalid", "{}"]
        for field, value in (("code", 3), ("code", False), ("code", None), ("txhash", "B" * 64),
                             ("events", []), ("events", [EVENT, EVENT])):
            response = copy.deepcopy(RESPONSE)
            response[field] = value
            cases.append(json.dumps(response))
        for field, value in (("owner", "1-1"), ("substationId", "4-2"), ("id", "11-1"), ("id", "")):
            response = copy.deepcopy(RESPONSE)
            response["events"][0]["attributes"][0]["value"] = json.dumps(dict(RECORD, **{field: value}))
            cases.append(json.dumps(response))
        for attrs in ([], EVENT["attributes"] * 2, [{"key": "provider", "value": "broken"}]):
            response = copy.deepcopy(RESPONSE)
            response["events"][0]["attributes"] = attrs
            cases.append(json.dumps(response))
        for payload in cases:
            result = self.invoke(payload)
            self.assertNotEqual(result.returncode, 0)
            self.assertEqual(result.stdout, "")
        for status, broadcast in ((1, {"txhash": HASH}), (0, {}), (0, {"txhash": "short"})):
            result = self.invoke(json.dumps(RESPONSE), status, broadcast)
            self.assertNotEqual(result.returncode, 0)
            self.assertEqual(result.stdout, "")


if __name__ == "__main__":
    unittest.main()
