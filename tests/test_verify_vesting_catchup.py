from copy import deepcopy
import unittest

from test_continue_upgrade_vesting import fixture, utc
from verify_vesting_catchup import verify_maturity_batch


def event(kind, **fields):
    return {'type': kind, 'attributes': [{'key': k, 'value': v} for k, v in dict(fields, mode='EndBlock').items()]}


def evidence():
    pending = fixture()['pending']
    previous = {'height': '79', 'chain_id': 'test', 'time': utc(1100)}
    current = {'height': '80', 'chain_id': 'test', 'time': utc(1300)}
    events = []
    for _ in pending:
        events.extend([event('coin_spent', spender='pool', amount='100000ualpha'),
                       event('coin_received', receiver='vault', amount='100000ualpha')])
    events.append(event('complete_unbonding', delegator='vault', validator='validator', amount='200000ualpha'))
    return [previous, current, {'height': '80', 'finalize_block_events': events}, pending, 'vault', 'pool']


class VestingCatchupTests(unittest.TestCase):
    def test_same_validator_entries_have_individual_bank_refunds_and_one_completion(self):
        result = verify_maturity_batch(*evidence())
        self.assertEqual(result['unbonding_ids'], ['1', '2'])
        self.assertEqual(result['refunded_ualpha'], 200000)
        self.assertEqual(result['completion_events'], 1)
        self.assertEqual(result['entry_refunds'], 2)

    def test_distinct_validators_require_distinct_completion_sums(self):
        args = evidence();args[3]['2']['validator_address'] = 'second'
        args[2]['finalize_block_events'][-1:] = [event('complete_unbonding', delegator='vault', validator=v, amount='100000ualpha') for v in ('validator', 'second')]
        self.assertEqual(verify_maturity_batch(*args)['completion_events'], 2)
        args[2]['finalize_block_events'].pop()
        with self.assertRaises(ValueError):verify_maturity_batch(*args)

    def test_missing_duplicate_misrouted_and_repriced_effects_refuse(self):
        for i in range(5):
            for fault in ('missing', 'duplicate', 'amount', 'mode', 'subject', 'duplicate-field'):
                args = evidence();events = args[2]['finalize_block_events']
                if fault == 'missing':events.pop(i)
                elif fault == 'duplicate':events.append(deepcopy(events[i]))
                elif fault == 'duplicate-field':events[i]['attributes'].append(deepcopy(events[i]['attributes'][0]))
                else:
                    target = {'amount': 'amount', 'mode': 'mode', 'subject': 'delegator' if i == 4 else 'spender' if i % 2 == 0 else 'receiver'}[fault]
                    for attr in events[i]['attributes']:
                        if attr['key'] == target:attr['value'] = '200000ualpha' if fault == 'amount' and i < 4 else 'wrong'
                with self.subTest(i=i, fault=fault), self.assertRaises(ValueError):verify_maturity_batch(*args)

    def test_pooled_bank_event_cannot_hide_missing_individual_refund(self):
        args = evidence();args[2]['finalize_block_events'][:4] = [event('coin_spent', spender='pool', amount='200000ualpha'),event('coin_received', receiver='vault', amount='200000ualpha')]
        with self.assertRaises(ValueError):verify_maturity_batch(*args)

    def test_time_boundary_identity_hold_and_transaction_controls(self):
        for fault in ('early', 'already-mature', 'height', 'chain', 'transaction', 'held', 'id', 'empty', 'zero', 'events'):
            args = evidence()
            if fault == 'early':args[1]['time'] = utc(1200)
            elif fault == 'already-mature':args[0]['time'] = utc(1200)
            elif fault == 'height':args[2]['height'] = '81'
            elif fault == 'chain':args[1]['chain_id'] = 'other'
            elif fault == 'transaction':args[2]['txs_results'] = [{'code': 0}]
            elif fault == 'held':args[3]['1']['unbonding_on_hold_ref_count'] = '1'
            elif fault == 'id':args[3]['1']['unbonding_id'] = '2'
            elif fault == 'empty':args[3] = {}
            elif fault == 'zero':args[3]['1']['balance'] = '0'
            elif fault == 'events':args[2]['finalize_block_events'] = None
            with self.subTest(fault=fault), self.assertRaises(ValueError):verify_maturity_batch(*args)


if __name__ == '__main__':
    unittest.main()
