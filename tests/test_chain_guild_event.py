"""Guild discovery binds a successful delivery to its creation and state events."""
import copy
import json
from pathlib import Path
import re
import subprocess
import unittest

SOURCE = (Path(__file__).parent / 'test_chain.sh').read_text()
HELPER = re.search(r'(?ms)^get_created_guild_id\(\) \{.*?^\}', SOURCE).group()
HASH = 'A' * 64
ID = '0-9007199254740993'
GUILD = {'id': ID, 'owner': '1-1', 'primaryReactorId': '3-1', 'endpoint': 'endpoint', 'entrySubstationId': '4-1'}

def event(kind, key, value):
    return {'type': 'structs.structs.' + kind, 'attributes': [{'key': key, 'value': json.dumps(value)}]}

RESPONSE = {'code': 0, 'txhash': HASH, 'events': [event('EventGuildBankAddress', 'eventGuildBankAddressDetail', {'guildId': ID}), event('EventGuild', 'guild', GUILD)]}

RESPONSE['events'] += [
    {'type': 'burn', 'attributes': [{'key': 'burner', 'value': 'module'}, {'key': 'amount', 'value': '1000000ualpha'}]},
    {'type': 'coinbase', 'attributes': [{'key': 'minter', 'value': 'module'}, {'key': 'amount', 'value': '1000000uguild.' + ID}]},
]


class GuildEventTests(unittest.TestCase):
    def invoke(self, response, status=0, station='4-1', broadcast=None):
        script = '''set -euo pipefail
payload="$1" status="$2" station="$3" LAST_TX_OUTPUT="$4"
_tx_json() { printf '%s' "$1"; }
query() { [ "$*" = 'query tx ''' + HASH + '''' ]; [ "$status" -eq 0 ] || return "$status"; printf '%s' "$payload"; }
'''
        script += HELPER + '\nget_created_guild_id 1-1 3-1 endpoint "$station"\n'
        return subprocess.run(['bash', '-c', script, 'test', json.dumps(response), str(status), station,
                               json.dumps({'txhash': HASH} if broadcast is None else broadcast)], capture_output=True, text=True)

    def test_matching_creation_allows_repeated_guild_snapshots(self):
        for station in ('4-1', ''):
            for code in (0, '0'):
                response = copy.deepcopy(RESPONSE)
                guild = dict(GUILD)
                if not station:
                    del guild['entrySubstationId']
                response['events'][1] = event('EventGuild', 'guild', guild)
                response['events'] += [event('EventGuild', 'guild', dict(guild, memberCount='1')),
                                       event('EventGuild', 'guild', {'id': '0-9', 'owner': '1-9'})]
                response['code'] = code
                response['txhash'] = HASH.lower()
                result = self.invoke(response, station=station)
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertEqual(result.stdout.strip(), ID)

    def test_failed_mismatched_and_ambiguous_creation_is_rejected(self):
        cases = [{}, dict(RESPONSE, code=1), dict(RESPONSE, code=False), dict(RESPONSE, txhash='B'*64), dict(RESPONSE, events=[])]
        for field in ('owner', 'primaryReactorId', 'endpoint', 'entrySubstationId', 'id'):
            response = copy.deepcopy(RESPONSE)
            response['events'][1] = event('EventGuild', 'guild', dict(GUILD, **{field: 'wrong'}))
            cases.append(response)
        for bad_id in (None, 1, '', '6-1'):
            response = copy.deepcopy(RESPONSE)
            response['events'][0] = event('EventGuildBankAddress', 'eventGuildBankAddressDetail', {'guildId': bad_id})
            cases.append(response)
        response = copy.deepcopy(RESPONSE)
        response['events'].append(response['events'][0])
        cases.append(response)
        response = copy.deepcopy(RESPONSE)
        response['events'].append(event('EventGuild', 'guild', dict(GUILD, owner='1-9')))
        cases.append(response)
        response = copy.deepcopy(RESPONSE)
        duplicate = copy.deepcopy(response['events'][1])
        duplicate['attributes'] *= 2
        response['events'].append(duplicate)
        cases.append(response)
        for index in (0, 1):
            for attributes in ([], [{'key': 'wrong', 'value': '{}'}], [{'key': 'guild' if index else 'eventGuildBankAddressDetail', 'value': 'broken'}]):
                response = copy.deepcopy(RESPONSE)
                response['events'][index]['attributes'] = attributes
                cases.append(response)
        for event_type in ('burn', 'coinbase'):
            for fault in ('missing', 'duplicate', 'amount', 'duplicate_amount'):
                response = copy.deepcopy(RESPONSE)
                native = next(e for e in response['events'] if e['type'] == event_type)
                if fault == 'missing':
                    response['events'].remove(native)
                elif fault == 'duplicate':
                    response['events'].append(copy.deepcopy(native))
                elif fault == 'amount':
                    native['attributes'][1]['value'] = '999999ualpha'
                else:
                    native['attributes'].append(copy.deepcopy(native['attributes'][1]))
                cases.append(response)
        for response in cases:
            result = self.invoke(response)
            self.assertNotEqual(result.returncode, 0, result.stdout)
            self.assertEqual(result.stdout, '')
        for station in (None, False):
            response = copy.deepcopy(RESPONSE)
            response['events'][1] = event('EventGuild', 'guild', dict(GUILD, entrySubstationId=station))
            self.assertNotEqual(self.invoke(response, station='').returncode, 0)
        self.assertNotEqual(self.invoke(RESPONSE, status=1).returncode, 0)
        self.assertNotEqual(self.invoke(RESPONSE, broadcast={}).returncode, 0)


if __name__ == '__main__':
    unittest.main()
