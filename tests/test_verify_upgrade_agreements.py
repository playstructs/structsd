from copy import deepcopy
import json
import unittest

from verify_upgrade_agreements import verify_legacy_agreements

COLLATERAL = 'structs1jhfvj3tag7g5dscmlsmtrm55tv0h8gz3r89juy'
EARNINGS = 'structs1p8n9cf529c824dyuwefpk0cc49kfnu4q6gc0tn'


def event(kind, label, row):
    return {'type': 'structs.structs.Event'+kind, 'attributes': [{'key': label, 'value': json.dumps(row)}]}


def fixture():
    provider = {'id':'10-1', 'substationId':'4-1', 'rate':{'amount':'3','denom':'ualpha'}, 'providerCancellationPenalty':'0.5','consumerCancellationPenalty':'0.25'}
    agreements = [{'id':'11-'+str(i),'allocationId':'6-'+str(i),'providerId':'10-1','startBlock':str(start),'endBlock':str(end),'capacity':str(cap),'owner':'1-'+str(i),'creator':'key-'+str(i)}
                  for i,start,end,cap in [(1,1,40,1),(2,3,30,3)]]
    allocations = [{'id':r['allocationId'],'type':'providerAgreement','sourceObjectId':'4-1'} for r in agreements]
    grid = [{'attributeId':'5-'+r['allocationId'],'value':r['capacity']} for r in agreements]
    grid += [{'attributeId':'14-10-1','value':'3'},{'attributeId':'3-10-1','value':'4'}]
    bank = [{'address':COLLATERAL,'coins':[{'denom':'ualpha','amount':'374'},{'denom':'other','amount':'7'}]}, {'address':EARNINGS,'coins':[{'denom':'ualpha','amount':'3'}]}]
    before = {'initial_height':'10','app_state':{'structs':{'providerList':[provider],'agreementList':agreements,'allocationList':allocations,'gridList':grid},'bank':{'balances':bank}}}
    after = deepcopy(before);after['initial_height']='11'
    return [before,after,[{'height':'10','finalize_block_events':[]}],{'10-1':{'collateral':COLLATERAL,'earnings':EARNINGS}}]


class LegacyAgreementMigrationTests(unittest.TestCase):
    def test_native_export_height_integers_and_canonical_strings(self):
        args=fixture();args[0]['initial_height']=10;args[1]['initial_height']=11
        self.assertEqual(verify_legacy_agreements(*args)['differences'],[])
        for value in (True,False,-1,2**64,10.0,'010'):
            args=fixture();args[0]['initial_height']=value
            with self.subTest(value=value),self.assertRaises(ValueError):verify_legacy_agreements(*args)

    def test_contract_clocks_and_pools_remain_unchanged(self):
        result=verify_legacy_agreements(*fixture())
        self.assertEqual(result['differences'],[])
        self.assertEqual(result['live_agreements'],['11-1','11-2'])
        for field,value in [('startBlock','10'),('endBlock','41'),('capacity','2'),('owner','1-9')]:
            args=fixture();args[1]['app_state']['structs']['agreementList'][0][field]=value
            with self.subTest(field=field):self.assertTrue(verify_legacy_agreements(*args)['differences'])
        for target in ('collateral','earnings','other'):
            args=fixture();bank=args[1]['app_state']['bank']['balances'];row=bank[1 if target=='earnings' else 0]['coins'][1 if target=='other' else 0];row['amount']=str(int(row['amount'])-1)
            with self.subTest(target=target):self.assertTrue(verify_legacy_agreements(*args)['differences'])
        for index in (0,2,3):
            args=fixture();args[1]['app_state']['structs']['gridList'][index]['value']='9'
            with self.subTest(index=index):self.assertTrue(verify_legacy_agreements(*args)['differences'])

    def test_early_movements_deletion_history_and_changed_events_refuse(self):
        for e in (event('Delete','objectId','11-1'), event('Delete','objectId','6-1'), event('ServiceObligation','obligation',{}),
                  event('Agreement','agreement',{'id':'11-1'}),
                  {'type':'coin_spent','attributes':[{'key':'spender','value':COLLATERAL},{'key':'amount','value':'1ualpha'}]}):
            args=fixture();args[2][0]['finalize_block_events'].append(e)
            with self.subTest(event=e),self.assertRaises(ValueError):verify_legacy_agreements(*args)

    def test_incomplete_inputs_accounts_allocation_and_expiry_cannot_pass(self):
        for fault in ('window','transaction','duplicate','mapping','swap','penalty','history','missing-allocation','power','expiry','short-pool','future-clock','load'):
            args=fixture();old=args[0]['app_state']['structs']
            if fault=='window':args[2]=[]
            elif fault=='transaction':args[2][0]['txs_results']=[{'code':0}]
            elif fault=='duplicate':old['agreementList'] *= 2
            elif fault=='mapping':args[3]={}
            elif fault=='swap':args[3]['10-1']['collateral']=EARNINGS
            elif fault=='penalty':old['providerList'][0]['providerCancellationPenalty']='0.5000000000000000001'
            elif fault=='history':args[1]['app_state']['structs']['service']={}
            elif fault=='missing-allocation':old['allocationList'].pop(0)
            elif fault=='power':old['gridList'][0]['value']='2'
            elif fault=='expiry':old['agreementList'][0]['endBlock']='10'
            elif fault=='short-pool':args[0]['app_state']['bank']['balances'][0]['coins'][0]['amount']='313'
            elif fault=='future-clock':old['gridList'][2]['value']='12'
            elif fault=='load':old['gridList'][3]['value']='5'
            with self.subTest(fault=fault),self.assertRaises(ValueError):verify_legacy_agreements(*args)


if __name__=='__main__':unittest.main()
