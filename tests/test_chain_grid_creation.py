"""Grid fixtures must use their own delivered events, including feeder identity."""
import copy
import json
from pathlib import Path
import re
import subprocess
import unittest


SOURCE = Path(__file__).with_name("test_chain.sh").read_text()
HELPER = re.search(r"(?ms)^get_created_grid_id\(\) \{.*?^\}", SOURCE).group()
HASH = "A" * 64


def event(name, key, row):
    return {"type": "structs.structs.Event" + name,
        "attributes": [{"key": key, "value": json.dumps(row)}]}


def allocation(index="9007199254740993", destination=""):
    return {"id": "6-" + index, "index": index, "type": "dynamic",
        "sourceObjectId": "1-1", "controller": "0-2", "destinationId": destination}


def transaction(kind="allocation", index="9007199254740993"):
    events = [event("Allocation", "allocation", allocation(index))]
    if kind == "substation":
        events = [event("Allocation", "allocation", allocation(destination="4-" + index)),
            event("Substation", "substation", {"id": "4-" + index, "owner": "0-2"})]
    return {"txhash": HASH, "height": "123", "code": 0, "events": events}


def invoke(document, kind="allocation", status=0, broadcast=None):
    script = '''set -euo pipefail
LAST_TX_OUTPUT="$1" response="$2" query_status="$3"
_tx_json() { printf '%s' "$1"; }
query() {
    [ "$*" = 'query tx AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA' ] || return 9
    printf '%s' "$response"; return "$query_status"
}
''' + HELPER + '\nget_created_grid_id "$4" "$5" "$6"\n'
    arguments = ("1-1", "0-2") if kind == "allocation" else ("0-2", "6-9007199254740993")
    return subprocess.run(["bash", "-c", script, "test",
        json.dumps({"txhash": HASH} if broadcast is None else broadcast), json.dumps(document),
        str(status), kind, *arguments], capture_output=True, text=True)


class GridCreationTests(unittest.TestCase):
    def test_exact_ids_for_player_source_and_guild_ownership(self):
        for kind, prefix in (("allocation", "6-"), ("substation", "4-")):
            for index in ("1", "101", "9007199254740993", str(2**64 - 1)):
                document = transaction(kind, index)
                document["events"].append({"type": "message", "attributes": []})
                result = invoke(document, kind)
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertEqual(result.stdout.strip(), prefix + index)
            document = transaction(kind)
            del document["code"]
            self.assertEqual(invoke(document, kind).returncode, 0)

    def test_rejects_wrong_confirmation_or_query_failure(self):
        for kind in ("allocation", "substation"):
            for field, value in (("txhash", "B" * 64), ("height", "0"), ("height", "01"),
                    ("height", str(2**63)), ("height", None), ("code", 1), ("code", False),
                    ("code", None), ("events", []), ("events", None)):
                document = transaction(kind)
                document[field] = value
                with self.subTest(kind=kind, field=field, value=value):
                    self.assertNotEqual(invoke(document, kind).returncode, 0)
            self.assertNotEqual(invoke(transaction(kind), kind, broadcast={}).returncode, 0)
            result = invoke(transaction(kind), kind, status=1)
            self.assertNotEqual(result.returncode, 0)
            self.assertEqual(result.stdout, "")
        self.assertNotEqual(invoke(transaction(), "unknown").returncode, 0)

    def test_allocation_requires_identity_controller_source_and_disconnected_state(self):
        for field, value in (("id", "6-1"), ("index", "01"), ("index", "0"),
                ("index", str(2**64)), ("index", 9007199254740993), ("type", "automated"),
                ("sourceObjectId", "1-2"), ("controller", "1-1"),
                ("destinationId", "4-1"), ("destinationId", None)):
            row = allocation()
            row[field] = value
            document = transaction()
            document["events"] = [event("Allocation", "allocation", row)]
            self.assertNotEqual(invoke(document).returncode, 0, (field, value))
        row = allocation()
        del row["destinationId"]
        document["events"] = [event("Allocation", "allocation", row)]
        self.assertEqual(invoke(document).returncode, 0)

    def test_substation_requires_own_feeder_event(self):
        for event_index, field, value in ((1, "owner", "1-1"), (1, "id", "4-0"),
                (1, "id", "4-01"), (1, "id", "4-" + str(2**64)),
                (0, "destinationId", "4-1"), (0, "id", "6-1"), (0, "index", "1")):
            document = transaction("substation")
            attribute = document["events"][event_index]["attributes"][0]
            row = json.loads(attribute["value"])
            row[field] = value
            attribute["value"] = json.dumps(row)
            self.assertNotEqual(invoke(document, "substation").returncode, 0)
        document = transaction("substation")
        document["events"].pop(0)
        self.assertNotEqual(invoke(document, "substation").returncode, 0)

    def test_malformed_or_ambiguous_events_are_never_discovery(self):
        for kind in ("allocation", "substation"):
            for position in range(len(transaction(kind)["events"])):
                for mutation in ("duplicate event", "duplicate attribute", "malformed json", "no attribute"):
                    document = transaction(kind)
                    record = document["events"][position]
                    if mutation == "duplicate event":
                        document["events"].append(copy.deepcopy(record))
                    elif mutation == "duplicate attribute":
                        record["attributes"] *= 2
                    elif mutation == "malformed json":
                        record["attributes"][0]["value"] = "not-json"
                    else:
                        record["attributes"] = []
                    self.assertNotEqual(invoke(document, kind).returncode, 0, mutation)

    def test_all_creation_call_sites_use_requested_ownership(self):
        # Every creation lookup must match the transaction arguments above it.
        previous = None
        controller = None
        checked = 0
        for line in SOURCE.splitlines():
            match = re.search(r'tx structs (allocation|substation)-create ("[^\"]+"|\S+) ("[^\"]+"|\S+)', line)
            if match:
                previous = match.groups()
                controller = '"${PLAYER_1_ID}"'  # The sole omitted controller is Alice's initial allocation.
            control = re.search(r'--controller ("[^\"]+"|\S+)', line)
            if control:
                controller = control.group(1)
            lookup = re.search(r'get_created_grid_id (allocation|substation) ("[^\"]+"|\S+) ("[^\"]+"|\S+)\)', line)
            if lookup:
                kind, first, second = lookup.groups()
                self.assertEqual((kind, first), previous[:2], line)
                if kind == "substation":
                    self.assertEqual(second, previous[2], line)
                else:
                    self.assertEqual(second, controller, line)
                checked += 1
        self.assertEqual(checked, 11)
        self.assertNotIn("get_newest_substation_id", SOURCE)
        self.assertNotIn("get_allocation_count", SOURCE)


if __name__ == "__main__":
    unittest.main()
