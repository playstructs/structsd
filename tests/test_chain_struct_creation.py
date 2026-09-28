"""Struct discovery must follow this transaction or this fleet, never a list page."""
import copy
import json
from pathlib import Path
import re
import subprocess
import unittest


SOURCE = Path(__file__).with_name("test_chain.sh").read_text()
HASH = "A" * 64


def helper(name):
    return re.search(r"(?ms)^" + name + r"\(\) \{.*?^\}", SOURCE).group()


def structure(index="9007199254740993", owner="1-3", kind="11"):
    return {"id": "5-" + index, "index": index, "owner": owner, "type": kind, "locationId": "9-3"}


def event(row):
    return {"type": "structs.structs.EventStruct", "attributes": [
        {"key": "structure", "value": json.dumps(row)}]}


def transaction(row=None):
    return {"txhash": HASH, "height": "123", "code": 0,
        "events": [event(row or structure())]}


def invoke(document, broadcast=None, status=0, body=None):
    script = '''set -euo pipefail
GREEN='' RED='' NC=''
PASS_COUNT=0 FAIL_COUNT=0
PLAYER_3_ID=1-3 CHARGE_BUILD=8
payload="$1" LAST_TX_OUTPUT="$2" query_status="$3" object="$4"
_tx_json() { printf '%s' "$1"; }
query() {
    case "$*" in
        'query tx AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA')
            printf '%s' "$payload"; return "$query_status";;
        'query structs struct 5-9007199254740993') printf '%s' "$object";;
        *) echo "unexpected query: $*" >&2; return 9;;
    esac
}
wait_for_charge() { :; }
run_tx() { :; }
'''
    script += helper("get_created_struct_id") + "\n" + helper("assert_new_struct") + "\n"
    script += body or 'get_created_struct_id 1-3 11\n'
    return subprocess.run(["bash", "-c", script, "test", json.dumps(document),
        json.dumps({"txhash": HASH} if broadcast is None else broadcast), str(status),
        json.dumps({"Struct": structure()})], capture_output=True, text=True)


class StructCreationTests(unittest.TestCase):
    def test_exact_event_ids_and_unrelated_events(self):
        for index in ("1", "100", "9007199254740993", str(2**64 - 1)):
            document = transaction(structure(index))
            document["events"].extend([{"type": "message", "attributes": []},
                event(structure("9", owner="1-4")), event(structure("10", kind="2"))])
            result = invoke(document)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(result.stdout.strip(), "5-" + index)
        document = transaction()
        del document["code"]
        result = invoke(document)
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_refuses_unconfirmed_wrong_transaction_and_rpc_failure(self):
        for field, value in (("txhash", "B" * 64), ("height", "0"), ("height", None),
                ("height", "01"), ("height", str(2**63)), ("code", 1), ("code", None), ("code", False),
                ("events", []), ("events", None)):
            document = transaction()
            document[field] = value
            with self.subTest(field=field, value=value):
                self.assertNotEqual(invoke(document).returncode, 0)
        self.assertNotEqual(invoke(transaction(), broadcast={}).returncode, 0)
        result = invoke(transaction(), status=1)
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(result.stdout, "")

    def test_refuses_wrong_or_ambiguous_struct_records(self):
        for field, value in (("owner", "1-4"), ("type", "2"), ("type", 11),
                ("id", "5-9007199254740992"), ("index", "0"), ("index", "01"),
                ("index", 9007199254740993), ("index", str(2**64))):
            row = structure()
            row[field] = value
            self.assertNotEqual(invoke(transaction(row)).returncode, 0)
        for attributes in ([], [{"key": "structure", "value": "not-json"}],
                event(structure())["attributes"] * 2):
            document = transaction()
            document["events"][0]["attributes"] = attributes
            self.assertNotEqual(invoke(document).returncode, 0)
        document = transaction()
        document["events"] *= 2
        self.assertNotEqual(invoke(document).returncode, 0)

    def test_cruiser_preparation_is_required_and_defers_computation(self):
        line = next(line for line in SOURCE.splitlines() if line.startswith('combat_require_struct CRUISER_ID 3 11') and 'prepare' in line)
        for failed in (False, True):
            script = 'set -euo pipefail\nPLAYER_3_FLEET_ID=9-3\n'
            script += 'combat_require_struct() { echo "$*"; return '+('7' if failed else '0')+'; }\n'
            result = subprocess.run(['bash', '-c', script+line], capture_output=True, text=True)
            self.assertEqual(result.returncode == 0, not failed)
            self.assertEqual(result.stdout.strip(), 'CRUISER_ID 3 11 fleet 9-3 water P14 Cruiser  player_3 prepare')

    def test_build_callers_use_the_requested_owner_and_type(self):
        # Catch a copied owner's variable in every direct build call site.
        # AR's fighters, the old pool builder, raid ore roles and the twelve EB
        # roles use combat_require_struct. ExtendedBattleSetupTests checks the
        # complete EB owner/type/location roster and its prepare/ready passes.
        previous = None
        checked = 0
        for line in SOURCE.splitlines():
            build = re.search(r'tx structs struct-build-initiate ("[^"]+"|\S+) ("[^"]+"|\S+) ', line)
            if build:
                previous = build.groups()
            creation = re.search(r'\w+=\$\(get_created_struct_id ("[^"]+"|\S+) ("[^"]+"|\S+)\)', line)
            if creation:
                self.assertEqual(creation.groups(), previous, line)
                checked += 1
        self.assertEqual(checked, 8)
        self.assertNotIn("get_newest_struct_id", SOURCE)

    def test_required_rebuild_refuses_rejected_or_missing_creation_receipt(self):
        body = '''RESULT=5-previous
get_all_structs() { echo '{"Struct":[],"queryHeight":"123"}'; }
wait_for_free_slot() { echo 0; }
combat_fixture_state() {
    [ "$*" = '5-9007199254740993 1-3 11 2-3 land' ] || return 9
    echo ready
}
run_compute() { echo 'unexpected compute' >&2; return 9; }
''' + helper('combat_require_struct') + '''
combat_require_struct RESULT 3 11 planet 2-3 land Fixture || exit 1
printf 'RESULT:%s' "$RESULT"
'''
        for mode in ('success', 'rejected', 'missing_event', 'wrong_owner', 'rpc'):
            with self.subTest(mode=mode):
                document = transaction()
                if mode == 'rejected':
                    document['code'] = 5
                elif mode == 'missing_event':
                    document['events'] = []
                elif mode == 'wrong_owner':
                    document = transaction(structure(owner='1-4'))
                result = invoke(document, status=1 if mode == 'rpc' else 0, body=body)
                self.assertEqual(result.returncode == 0, mode == 'success', result.stdout + result.stderr)
                if mode == 'success':
                    self.assertIn('RESULT:5-9007199254740993', result.stdout)
                else:
                    self.assertNotIn('RESULT:', result.stdout)
                self.assertNotIn('unexpected compute', result.stderr)

    def test_command_ship_uses_fleet_association_and_verifies_object(self):
        row = structure(kind="1")
        fleet = {"Fleet": {"id": "9-3", "owner": "1-3", "commandStruct": row["id"]}}
        def call(fleet_document, struct_document, status=0):
            script = '''set -euo pipefail
fixture_fleet="$1" fixture_struct="$2" query_status="$3"
query() {
    case "$*" in
        'query structs fleet 9-3') printf '%s' "$fixture_fleet"; return "$query_status";;
        'query structs struct 5-9007199254740993') printf '%s' "$fixture_struct";;
        *) return 9;;
    esac
}
'''
            script += helper("get_command_ship_id") + '\nget_command_ship_id 1-3 9-3\n'
            return subprocess.run(["bash", "-c", script, "test", json.dumps(fleet_document),
                json.dumps(struct_document), str(status)], capture_output=True, text=True)
        result = call(fleet, {"Struct": row})
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout.strip(), row["id"])
        for field, value in (("id", "9-4"), ("owner", "1-4"), ("commandStruct", "")):
            wrong = copy.deepcopy(fleet)
            wrong["Fleet"][field] = value
            self.assertNotEqual(call(wrong, {"Struct": row}).returncode, 0)
        for field, value in (("id", "5-99"), ("owner", "1-4"), ("type", "2"), ("locationId", "9-4")):
            wrong = dict(row, **{field: value})
            self.assertNotEqual(call(fleet, {"Struct": wrong}).returncode, 0)
        self.assertNotEqual(call(fleet, {"Struct": row}, status=1).returncode, 0)


if __name__ == "__main__":
    unittest.main()
