import copy
import unittest

from test_verify_upgrade_grid import event
from verify_upgrade_allocations import verify_allocations


def fixture():
    rows = [{'id': '6-'+str(i), 'index': str(i), 'type': mode, 'sourceObjectId': '1-1',
             'destinationId': '4-1', 'creator': 'key', 'controller': '1-1'}
            for i, mode in enumerate([500, -1, 2147483647, 'dynamic', 'automated', 'providerAgreement', 'static'], 1)]
    before = {'initial_height': '7', 'app_state': {'structs': {'allocationList': rows}}}
    after = copy.deepcopy(before);after['initial_height'] = '9'
    for row in after['app_state']['structs']['allocationList'][:3]:row['type'] = 'static'
    blocks = [{'height': '7', 'finalize_block_events': [event('Allocation', 'allocation', r)
               for r in after['app_state']['structs']['allocationList'][:3]]},
              {'height': '8', 'finalize_block_events': []}]
    return before, after, blocks


class AllocationMigrationReplayTests(unittest.TestCase):
    def test_exact_normalization_and_declared_modes(self):
        result = verify_allocations(*fixture())
        self.assertEqual(result['differences'], [])
        self.assertEqual(result['legacy_allocation_ids'], ['6-1', '6-2', '6-3'])
        self.assertEqual(result['event_counts'], dict(replacements=3, deletions=0, normalized=3))

    def test_brownout_deletion_after_repair_replays_in_event_order(self):
        before, after, blocks = fixture()
        after['app_state']['structs']['allocationList'].pop(0)
        blocks[0]['finalize_block_events'].append(event('Delete', 'objectId', '6-1', 'EndBlock'))
        self.assertEqual(verify_allocations(before, after, blocks)['differences'], [])
        blocks[0]['finalize_block_events'].reverse()
        self.assertTrue(verify_allocations(before, after, blocks)['differences'])

    def test_missing_event_never_invents_normalization(self):
        for index in range(3):
            before, after, blocks = fixture();blocks[0]['finalize_block_events'].pop(index)
            self.assertTrue(verify_allocations(before, after, blocks)['differences'])
        before, after, blocks = fixture()
        after['app_state']['structs']['allocationList'].pop(0)
        blocks[0]['finalize_block_events'].pop(0)
        blocks[0]['finalize_block_events'].append(event('Delete', 'objectId', '6-1', 'EndBlock'))
        self.assertIn('missing legacy allocation normalization events', verify_allocations(before, after, blocks)['differences'])

    def test_repair_may_not_change_fields_or_emit_twice(self):
        for field, value in [('controller', '1-2'), ('creator', 'other'), ('destinationId', ''),
                             ('sourceObjectId', '1-2'), ('type', 'dynamic'), ('index', '2')]:
            before, after, blocks = fixture();row = copy.deepcopy(after['app_state']['structs']['allocationList'][0]);row[field] = value
            blocks[0]['finalize_block_events'][0] = event('Allocation', 'allocation', row)
            with self.subTest(field=field), self.assertRaises(ValueError):verify_allocations(before, after, blocks)
        before, after, blocks = fixture();blocks[0]['finalize_block_events'].append(copy.deepcopy(blocks[0]['finalize_block_events'][0]))
        with self.assertRaises(ValueError):verify_allocations(before, after, blocks)

    def test_declared_types_and_complete_projection_are_preserved(self):
        before, after, blocks = fixture();row = after['app_state']['structs']['allocationList'][3];row['type'] = 'static'
        self.assertTrue(verify_allocations(before, after, blocks)['differences'])
        blocks[0]['finalize_block_events'].append(event('Allocation', 'allocation', row))
        with self.assertRaises(ValueError):verify_allocations(before, after, blocks)
        before, after, blocks = fixture();after['app_state']['structs']['allocationList'][4]['controller'] = '1-2'
        self.assertTrue(verify_allocations(before, after, blocks)['differences'])

    def test_bad_inventory_encodings_boundaries_and_duplicate_attributes(self):
        for mode in (True, 4.0, '500', 'unknown', 2**31, -(2**31)-1):
            before, after, blocks = fixture();before['app_state']['structs']['allocationList'][0]['type'] = mode
            with self.subTest(mode=mode), self.assertRaises(ValueError):verify_allocations(before, after, blocks)
        for fault in ('missing-inventory', 'duplicate-id', 'missing-block', 'duplicate-block', 'transaction', 'missing-events', 'object-events', 'string-events', 'duplicate-attribute', 'phase', 'post-unknown'):
            before, after, blocks = fixture()
            if fault == 'missing-inventory':before['app_state']['structs'].pop('allocationList')
            elif fault == 'duplicate-id':before['app_state']['structs']['allocationList'] *= 2
            elif fault == 'missing-block':blocks.pop()
            elif fault == 'duplicate-block':blocks.append(copy.deepcopy(blocks[-1]))
            elif fault == 'transaction':blocks[0]['txs_results'] = [{'code': 0}]
            elif fault == 'missing-events':blocks[0].pop('finalize_block_events')
            elif fault == 'object-events':blocks[1]['finalize_block_events'] = {}
            elif fault == 'string-events':blocks[1]['finalize_block_events'] = ''
            elif fault == 'duplicate-attribute':blocks[0]['finalize_block_events'][0]['attributes'] *= 2
            elif fault == 'phase':blocks[0]['finalize_block_events'][0]['attributes'].append({'key': 'mode', 'value': 'wrong'})
            elif fault == 'post-unknown':after['app_state']['structs']['allocationList'][0]['type'] = 500
            with self.subTest(fault=fault), self.assertRaises((ValueError, KeyError)):verify_allocations(before, after, blocks)


if __name__ == '__main__':
    unittest.main()
