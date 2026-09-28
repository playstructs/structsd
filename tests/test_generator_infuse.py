"""Adversarial receipt checks for the native generator top-up fixture."""
import copy
import json
import unittest

from generator_infuse import verify

HASH = 'A' * 64


def event(kind, **attrs):
    return {'type': kind, 'attributes': [{'key': key, 'value': value} for key, value in attrs.items()]}


def typed(kind, name, row):
    return event('structs.structs.' + kind, **{name: json.dumps(row), 'msg_index': '0'})


def receipt(fuel=1000000, capacity=6800000):
    return {'txhash': HASH, 'code': 0, 'height': '101',
            'tx': {'body': {'messages': [{'@type': '/structs.structs.MsgStructGeneratorInfuse',
                'creator': 'key4', 'structId': '5-4', 'infuseAmount': '1000000ualpha'}]}},
            'events': [
                event('coin_spent', spender='vault4', amount='1000000ualpha', msg_index='0'),
                event('coin_received', receiver='module', amount='1000000ualpha', msg_index='0'),
                event('transfer', sender='vault4', recipient='module', amount='1000000ualpha', msg_index='0'),
                event('coin_spent', spender='module', amount='1000000ualpha', msg_index='0'),
                event('burn', burner='module', amount='1000000ualpha', msg_index='0'),
                typed('EventAlphaInfuse', 'eventAlphaInfuseDetail', {'playerId': '1-4', 'primaryAddress': 'primary4',
                    'vaultAddress': 'vault4', 'amount': '1000000'}),
                typed('EventInfusion', 'infusion', {'destinationType': 'struct', 'destinationId': '5-4',
                    'address': 'vault4', 'playerId': '1-4', 'fuel': str(fuel), 'power': str(fuel * 2),
                    'ratio': '2', 'commission': '0.000000000000000000', 'defusing': '0',
                    'generatorDisabled': False}),
                typed('EventGrid', 'gridRecord', {'attributeId': '1-5-4', 'value': str(fuel)}),
                typed('EventGrid', 'gridRecord', {'attributeId': '2-1-4', 'value': str(capacity)}),
            ]}


def check(tx, fuel=1000000, capacity=6800000):
    return verify(tx, HASH, '5-4', '1-4', 'vault4', 'primary4', 'key4', 'module', str(fuel), str(capacity))


class GeneratorInfuseTests(unittest.TestCase):
    def test_fresh_and_funded_with_distinct_signer_and_primary(self):
        for old in (0, 53, 1000000):
            fuel, capacity = old + 1000000, 6800000 + old * 2
            self.assertEqual(check(receipt(fuel, capacity), fuel, capacity), '101')
        tx = receipt()
        row = json.loads(tx['events'][6]['attributes'][0]['value'])
        row['ownerId'], row['playerId'] = '1-4', ''
        tx['events'][6] = typed('EventInfusion', 'infusion', row)
        self.assertEqual(check(tx), '101')

    def test_missing_duplicate_reordered_or_changed_bank_events(self):
        for index in range(5):
            for mutation in ('missing', 'duplicate', 'reorder', 'attribute', 'duplicate-attribute'):
                with self.subTest(index=index, mutation=mutation):
                    tx = receipt()
                    if mutation == 'missing':
                        del tx['events'][index]
                    elif mutation == 'duplicate':
                        tx['events'].insert(index, copy.deepcopy(tx['events'][index]))
                    elif mutation == 'reorder':
                        tx['events'][index], tx['events'][index + 1] = tx['events'][index + 1], tx['events'][index]
                        # Swapping the burn with a non-bank event does not change bank order.
                        if index == 4:
                            tx['events'][3], tx['events'][5] = tx['events'][5], tx['events'][3]
                    elif mutation == 'attribute':
                        tx['events'][index]['attributes'][0]['value'] = 'wrong'
                    else:
                        tx['events'][index]['attributes'].append(copy.deepcopy(tx['events'][index]['attributes'][0]))
                    with self.assertRaises(ValueError):
                        check(tx)
        for kind in ('mint', 'coinbase', 'coin_received'):
            tx = receipt()
            tx['events'].append(event(kind, amount='1ualpha'))
            with self.assertRaises(ValueError):
                check(tx)

    def test_typed_events_require_exact_identity_and_accounting(self):
        for index in range(5, 9):
            original = receipt()['events'][index]
            row = json.loads(original['attributes'][0]['value'])
            for key in row:
                with self.subTest(index=index, key=key):
                    tx = receipt()
                    altered = dict(row, **{key: 'wrong'})
                    tx['events'][index]['attributes'][0]['value'] = json.dumps(altered)
                    with self.assertRaises(ValueError):
                        check(tx)
            for duplicate in (False, True):
                tx = receipt()
                if duplicate:
                    tx['events'].append(copy.deepcopy(original))
                else:
                    del tx['events'][index]
                with self.assertRaises(ValueError):
                    check(tx)

    def test_amounts_and_message_indices_are_not_optional(self):
        for index in range(9):
            keys = ('amount', 'msg_index') if index < 5 else ('msg_index',)
            for key in keys:
                for replacement in ('missing', '1', '1000001ualpha', '1000000other'):
                    with self.subTest(index=index, key=key, replacement=replacement):
                        tx = receipt()
                        attrs = tx['events'][index]['attributes']
                        if replacement == 'missing':
                            tx['events'][index]['attributes'] = [a for a in attrs if a['key'] != key]
                        else:
                            next(a for a in attrs if a['key'] == key)['value'] = replacement
                        with self.assertRaises(ValueError):
                            check(tx)

    def test_receipt_and_message_must_match_requested_transaction(self):
        for key, value in (('code', 1), ('code', False), ('height', '0'), ('height', '01'), ('height', str(2**63)), ('txhash', 'B' * 64)):
            tx = receipt()
            tx[key] = value
            with self.assertRaises(ValueError):
                check(tx)
        for key in ('@type', 'creator', 'structId', 'infuseAmount'):
            tx = receipt()
            tx['tx']['body']['messages'][0][key] = 'wrong'
            with self.assertRaises(ValueError):
                check(tx)
        tx = receipt()
        tx['tx']['body']['messages'] *= 2
        with self.assertRaises(ValueError):
            check(tx)


if __name__ == '__main__':
    unittest.main()
