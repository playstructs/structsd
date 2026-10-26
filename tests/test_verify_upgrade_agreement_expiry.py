from copy import deepcopy
import unittest

from test_verify_upgrade_agreements import COLLATERAL, EARNINGS, event, fixture
from verify_upgrade_agreement_expiry import verify_legacy_expiry


def expiry_fixture():
    before,migrated,migration_blocks,accounts=fixture()
    after=deepcopy(migrated);after['initial_height']='41'
    state=after['app_state']['structs']
    blocks=[{'height':str(h),'finalize_block_events':[]} for h in range(11,41)]
    # At 30: checkpoint 27*4*3*.5=162, holdback floor(27*3*3*.5)=121.
    # At 40: checkpoint 10*1*3*.5=15, holdback floor(39*1*3*.5)=58.
    for index,total in ((1,283),(0,73)):
        row=state['agreementList'][index]
        events=blocks[int(row['endBlock'])-11]['finalize_block_events']
        events.extend([event('Delete','objectId',row['id']),event('Delete','objectId',row['allocationId'])])
        for kind,label,address in [('coin_spent','spender',COLLATERAL),('coin_received','receiver',EARNINGS)]:
            events.append({'type':kind,'attributes':[{'key':label,'value':address},{'key':'amount','value':str(total)+'ualpha'}]})
    state['agreementList']=[];state['allocationList']=[]
    state['gridList']=[{'attributeId':'14-10-1','value':'40'}]
    after['app_state']['bank']['balances'][0]['coins'][0]['amount']='18'
    after['app_state']['bank']['balances'][1]['coins'][0]['amount']='359'
    return [before,migrated,after,migration_blocks,blocks,accounts]


class LegacyAgreementExpiryTests(unittest.TestCase):
    def test_native_export_height_integers_and_canonical_strings(self):
        args=expiry_fixture()
        for document in args[:3]:document['initial_height']=int(document['initial_height'])
        self.assertEqual(verify_legacy_expiry(*args)['differences'],[])
        for value in (True,-1,2**64,41.0,'041'):
            args=expiry_fixture();args[2]['initial_height']=value
            with self.subTest(value=value),self.assertRaises(ValueError):verify_legacy_expiry(*args)

    def test_full_healthy_expiry_preserves_ordinary_rounding(self):
        result=verify_legacy_expiry(*expiry_fixture())
        self.assertEqual(result['differences'],[])
        self.assertEqual(result['provider_paid'],{'10-1':'356'})
        self.assertEqual(result['settled_agreements'],['11-1','11-2'])

    def test_missing_payments_and_records_cannot_pass(self):
        for fault in ('bank-event','bank-balance','collateral','missing-delete','retained-allocation','agreement','load','checkpoint'):
            args=expiry_fixture();state=args[2]['app_state']['structs']
            if fault=='bank-event':args[4][-1]['finalize_block_events'].pop()
            elif fault=='bank-balance':args[2]['app_state']['bank']['balances'][1]['coins'][0]['amount']='360'
            elif fault=='collateral':args[2]['app_state']['bank']['balances'][0]['coins'][0]['amount']='17'
            elif fault=='missing-delete':args[4][-1]['finalize_block_events'].pop(0)
            elif fault=='retained-allocation':state['allocationList']=[deepcopy(args[1]['app_state']['structs']['allocationList'][0])]
            elif fault=='agreement':state['agreementList']=[deepcopy(args[1]['app_state']['structs']['agreementList'][0])]
            elif fault=='load':state['gridList'].append({'attributeId':'3-10-1','value':'1'})
            else:state['gridList'][0]['value']='39'
            with self.subTest(fault=fault):
                if fault=='bank-event':
                    with self.assertRaisesRegex(ValueError,'within block'):verify_legacy_expiry(*args)
                else:self.assertTrue(verify_legacy_expiry(*args)['differences'])

    def test_mistimed_events_and_changed_scope_refuse(self):
        for fault in ('premature','duplicate-delete','incomplete','transaction','bad-migration','early-final','repricing','history'):
            args=expiry_fixture();events=args[4][-1]['finalize_block_events']
            if fault=='premature':args[4][-2]['finalize_block_events']=events;args[4][-1]['finalize_block_events']=[]
            elif fault=='duplicate-delete':events.append(deepcopy(events[0]))
            elif fault=='incomplete':args[4].pop(0)
            elif fault=='transaction':args[4][0]['txs_results']=[{'code':0}]
            elif fault=='bad-migration':args[1]['app_state']['structs']['agreementList'][0]['startBlock']='10'
            elif fault=='early-final':args[2]['initial_height']='40';args[4].pop()
            elif fault=='repricing':events.append(event('Agreement','agreement',{}))
            else:events.append(event('ServiceObligation','obligation',{}))
            with self.subTest(fault=fault),self.assertRaises(ValueError):verify_legacy_expiry(*args)

    def test_early_payment_cannot_hide_in_correct_final_totals(self):
        args=expiry_fixture();events=args[4][-1]['finalize_block_events']
        args[4][-2]['finalize_block_events']=events[-2:];del events[-2:]
        with self.assertRaisesRegex(ValueError,'within block'):verify_legacy_expiry(*args)


if __name__=='__main__':unittest.main()
