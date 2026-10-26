"""Combat slot recovery must distinguish full arrays from failed state reads."""
import copy
import json
import os
from pathlib import Path
import re
import signal
import subprocess
import tempfile
import unittest


SOURCE = (Path(__file__).parent/'test_chain.sh').read_text()
HELPERS = '\n'.join(re.search(r'(?ms)^'+name+r'\(\) \{.*?^\}', SOURCE).group()
                    for name in ('first_free_slot', 'wait_for_free_slot'))


def location(kind='fleet', ambit='space', occupied=None):
    used = ['5-1', '', '5-3', ''] if occupied is None else occupied
    return {kind.title(): {'id': '9-1' if kind == 'fleet' else '2-1',
                           ambit+'Slots': str(len(used)), ambit: used}}


def invoke(document, *, kind='fleet', ambit='space', wait=False, rpc_code=0, later=None):
    with tempfile.TemporaryDirectory() as directory:
        root = Path(directory)
        (root/'response.json').write_text(json.dumps(document))
        (root/'later.json').write_text(json.dumps(later if later is not None else document))
        script = '''set -eu
fixture_dir=$1; kind=$2; ambit=$3; mode=$4; rpc_code=$5
GREEN=; YELLOW=; NC=
query() {
    echo "$*" >> "$fixture_dir/queries"
    if [ -f "$fixture_dir/sleeps" ]; then cat "$fixture_dir/later.json"; else cat "$fixture_dir/response.json"; fi
    return "$rpc_code"
}
sleep() { echo "$1" >> "$fixture_dir/sleeps"; }
'''+HELPERS+'''
if [ "$kind" = planet ]; then id=2-1; else id=9-1; fi
if [ "$mode" = wait ]; then
    result=$(wait_for_free_slot "$kind" "$id" "$ambit" 6) || exit 23
else
    result=$(first_free_slot "$kind" "$id" "$ambit") || exit 23
fi
printf '%s' "$result"
'''
        arguments = ['bash', '-c', script, 'slots', str(root), kind, ambit,
                     'wait' if wait else 'once', str(rpc_code)]
        # Bound the process group too: a count/array regression can leave jq
        # expanding a forged uint64 slot count inside a shell substitution.
        with subprocess.Popen(arguments, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                              text=True, start_new_session=True) as process:
            try:
                stdout, stderr = process.communicate(timeout=5)
            except subprocess.TimeoutExpired:
                os.killpg(process.pid, signal.SIGKILL)
                process.communicate()
                raise
            result = subprocess.CompletedProcess(arguments, process.returncode, stdout, stderr)
        queries = (root/'queries').read_text().splitlines() if (root/'queries').exists() else []
        sleeps = (root/'sleeps').read_text().splitlines() if (root/'sleeps').exists() else []
        return result, queries, sleeps


class CombatSlotTests(unittest.TestCase):
    def test_selects_first_actual_free_slot_for_each_location_and_ambit(self):
        for kind in ('fleet', 'planet'):
            for ambit in ('space', 'air', 'land', 'water'):
                for occupied, expected in ((['5-1', '', '5-3', ''], '1'),
                                           (['', '5-2'], '0'), (['5-1'], ''), ([], '')):
                    with self.subTest(kind=kind, ambit=ambit, occupied=occupied):
                        result, queries, sleeps = invoke(location(kind, ambit, occupied), kind=kind, ambit=ambit)
                        self.assertEqual(result.returncode, 0, result.stderr)
                        self.assertEqual(result.stdout, expected)
                        self.assertEqual(len(queries), 1)
                        self.assertEqual(sleeps, [])

    def test_absent_proto_defaults_are_zero_slots_but_null_is_not_an_array(self):
        result, _, _ = invoke({'Fleet': {'id': '9-1'}})
        self.assertEqual((result.returncode, result.stdout), (0, ''))
        for field in ('space', 'spaceSlots'):
            doc = {'Fleet': {'id': '9-1', field: None}}
            result, _, _ = invoke(doc)
            self.assertNotEqual(result.returncode, 0)

    def test_failed_rpc_cannot_advertise_a_free_slot_or_start_retrying(self):
        for wait in (False, True):
            for doc in (location(), location(occupied=['5-1']), {}):
                with self.subTest(wait=wait, document=doc):
                    result, queries, sleeps = invoke(doc, wait=wait, rpc_code=9)
                    self.assertEqual(result.returncode, 23)
                    self.assertEqual(result.stdout, '')
                    self.assertEqual(len(queries), 1)
                    self.assertEqual(sleeps, [])

    def test_missing_mismatched_or_malformed_occupancy_refuses_instead_of_appearing_full(self):
        malformed = [{}, {'Fleet': None}, {'Fleet': []}]
        for field, value in (('id', '9-2'), ('spaceSlots', 4), ('spaceSlots', '04'),
                             ('spaceSlots', '18446744073709551615'), ('spaceSlots', '3'),
                             ('space', []), ('space', ''), ('space', ['5-1', '', None, '']),
                             ('space', ['5-1', '', 'not-a-struct', ''])):
            doc = copy.deepcopy(location()); doc['Fleet'][field] = value
            malformed.append(doc)
        doc = location(); del doc['Fleet']['space']; malformed.append(doc)
        for doc in malformed:
            with self.subTest(document=doc):
                result, queries, sleeps = invoke(doc, wait=True)
                self.assertEqual(result.returncode, 23)
                self.assertEqual(result.stdout, '')
                self.assertEqual(len(queries), 1)
                self.assertEqual(sleeps, [])

    def test_waits_for_sweep_and_stops_at_timeout_when_full(self):
        full = location(occupied=['5-1', '5-2'])
        result, queries, sleeps = invoke(full, wait=True, later=location(occupied=['5-1', '']))
        self.assertEqual((result.returncode, result.stdout), (0, '1'), result.stderr)
        self.assertEqual(len(queries), 2)
        self.assertEqual(sleeps, ['3'])
        result, queries, sleeps = invoke(full, wait=True)
        self.assertEqual((result.returncode, result.stdout), (0, ''), result.stderr)
        self.assertEqual(len(queries), 2)
        self.assertEqual(sleeps, ['3', '3'])

    def test_invalid_location_kind_and_ambit_never_query(self):
        for kind, ambit in (('player', 'space'), ('fleet', 'none')):
            result, queries, sleeps = invoke(location(), kind=kind, ambit=ambit)
            self.assertEqual(result.returncode, 23)
            self.assertEqual((queries, sleeps), ([], []))


if __name__ == '__main__':
    unittest.main()
