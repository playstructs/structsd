"""Rotation must merge one complete delegation set and clear source accounts."""
import json
from pathlib import Path
import re
import subprocess
import unittest

SOURCE = (Path(__file__).parent / 'test_chain.sh').read_text()
HELPER = re.search(r'(?ms)^get_fixture_delegation_shares\(\) \{.*?^\}', SOURCE).group()
ASSERT = re.search(r'(?ms)^assert_eq\(\) \{.*?^\}', SOURCE).group()


def response(amount='2500000.000000000000000000'):
    return {'pagination': {'next_key': None, 'total': '0'}, 'delegation_responses': [
        {'delegation': {'delegator_address': 'vault', 'validator_address': 'validator',
                        'shares': amount}}]}


class RotationStakingTests(unittest.TestCase):
    def test_native_empty_defaults_require_complete_successful_response(self):
        for payload in ({'pagination': {}}, {'pagination': {}, 'delegation_responses': None}):
            for conditional in (False, True):
                result = self.invoke(payload, 'empty', conditional=conditional)
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertEqual(result.stdout.strip(), '0.000000000000000000')
        for payload in ({}, {'pagination': {}, 'delegation_responses': False},
                {'pagination': {}, 'delegationResponses': []},
                {'pagination': {'nextKey': 'YQ=='}},
                {'pagination': {'next_key': '', 'nextKey': ''}}):
            for conditional in (False, True):
                result = self.invoke(payload, 'empty', conditional=conditional)
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(result.stdout, '')

    def test_rotation_key_lookup_uses_json_and_rejects_missing_addresses(self):
        assignment = re.search(r'^ROTATION_ADDRESS=.*$', SOURCE, re.M).group()
        # PARAMS_KEYS already includes JSON output. The SDK refuses address-only
        # switches in the same invocation, including the later proof check.
        invocations = re.findall(r'structsd \$\{PARAMS_KEYS\} keys show[^|\n]*', SOURCE)
        self.assertTrue(invocations)
        for invocation in invocations:
            self.assertNotRegex(invocation, r'(?:\s-a|\s--address)(?:\s|\)|$)')
        for payload, succeeds in (({'address': 'rotation'}, True), ({}, False),
                ({'address': ''}, False), ({'address': None}, False), ({'address': 7}, False)):
            script = '''set -euo pipefail
PARAMS_KEYS='--keyring-backend test --output json'
ROTATION_KEY=rotation PAYLOAD="$1"
structsd() { [ "$*" = '--keyring-backend test --output json keys show rotation' ] || return 9; echo "$PAYLOAD"; }
'''
            result = subprocess.run(['bash', '-c', script + assignment + '\nprintf "%s" "$ROTATION_ADDRESS"',
                'test', json.dumps(payload)], capture_output=True, text=True)
            self.assertEqual(result.returncode == 0, succeeds, result.stdout + result.stderr)
            if succeeds:
                self.assertEqual(result.stdout, 'rotation')

    def invoke(self, payload, presence='present', status=0, conditional=False):
        script = '''set -euo pipefail
PAYLOAD="$1" STATUS="$2" PRESENCE="$3"
query() { [ "$*" = 'query staking delegations vault --page-limit 2' ] || return 9; echo "$PAYLOAD"; return "$STATUS"; }
'''
        call = 'get_fixture_delegation_shares vault validator "$PRESENCE"'
        if conditional:
            call = 'if ' + call + '; then exit 0; else exit "$?"; fi'
        return subprocess.run(['bash', '-c', script + HELPER + '\n' + call,
                               'test', json.dumps(payload), str(status), presence],
                              capture_output=True, text=True)

    def test_exact_shares_and_explicit_empty_set(self):
        for amount in ('0.000000000000000001', '2500000.000000000000000000',
                       '18446744073709551617.123456789012345678'):
            for conditional in (False, True):
                result = self.invoke(response(amount), conditional=conditional)
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertEqual(result.stdout.strip(), amount)
        empty = response()
        empty['delegation_responses'] = []
        result = self.invoke(empty, 'empty')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout.strip(), '0.000000000000000000')
        self.assertNotEqual(self.invoke(empty).returncode, 0)
        self.assertNotEqual(self.invoke(response(), 'empty').returncode, 0)

    def test_malformed_ambiguous_paginated_and_failed_queries_refuse(self):
        cases = [{}, {'pagination': {}}, {'pagination': {}, 'delegation_responses': None}]
        for field in ('delegator_address', 'validator_address'):
            item = response()
            item['delegation_responses'][0]['delegation'][field] = 'wrong'
            cases.append(item)
        for amount in ('0.000000000000000000', '1', 1, None, '-1.000000000000000000',
                       '01.000000000000000000', '1.00000000000000000x'):
            cases.append(response(amount))
        for next_key in ('another-page', False, 1):
            item = response()
            item['pagination']['next_key'] = next_key
            cases.append(item)
        duplicate = response()
        duplicate['delegation_responses'] *= 2
        cases.append(duplicate)
        for conditional in (False, True):
            for payload in cases:
                result = self.invoke(payload, conditional=conditional)
                self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
                self.assertEqual(result.stdout, '')
            for presence in ('present', 'empty'):
                payload = response()
                if presence == 'empty':
                    payload['delegation_responses'] = []
                result = self.invoke(payload, presence, status=1, conditional=conditional)
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(result.stdout, '')

    def test_each_actual_source_removal_assertion_detects_retained_stake(self):
        blocks = re.findall(r'assert_eq "(?:Rotation removes old primary delegation|'
                            r'Restoration leaves original primary without delegated stake|'
                            r'Revoked secondary retains no delegated stake)"[^\n]*\n[^\n]*', SOURCE)
        self.assertEqual(len(blocks), 3)
        for block in blocks:
            address = 'secondary' if 'ROTATION_SECOND_ADDRESS' in block else 'primary'
            for retained in (False, True):
                payload = response()
                payload['delegation_responses'][0]['delegation']['delegator_address'] = address
                if not retained:
                    payload['delegation_responses'] = []
                script = '''set -euo pipefail
GREEN='' RED='' NC='' PASS_COUNT=0 FAIL_COUNT=0
ROTATION_ADDRESS=primary ROTATION_SECOND_ADDRESS=secondary ROTATION_VALIDATOR=validator
PAYLOAD="$1" EXPECTED="$2"
query() { [ "$*" = "query staking delegations $EXPECTED --page-limit 2" ] || return 9; echo "$PAYLOAD"; }
'''
                result = subprocess.run(['bash', '-c', script + HELPER + '\n' + ASSERT + '\n' +
                                         block + '\n[ "$FAIL_COUNT" -eq 0 ]',
                                         'test', json.dumps(payload), address], capture_output=True, text=True)
                self.assertEqual(result.returncode == 0, not retained, result.stdout + result.stderr)


if __name__ == '__main__':
    unittest.main()
