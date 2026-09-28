"""Preparation attacks must really destroy a weak fighter without harming the commander."""
import copy
import json
import unittest

from retire_fighter import after, attack, before, catalog
from test_chain_evasion import HASH, receipt, structure


def fixture(ah=6,th=1):
    start={'attacker':structure('5-1','1','space',ah,'1-6','9-6'),
           'target':structure('5-2','3','space',th,'1-3','9-3')}
    tx=receipt('EV4',False,ah,th)
    row=json.loads(tx['events'][0]['attributes'][0]['value'])
    row['eventAttackShotDetail'][0]['targetStructTypeId']='3'
    tx['events'][0]['attributes'][0]['value']=json.dumps(row)
    end=copy.deepcopy(start)
    end['target']['structAttributes'].update(health='0',isDestroyed=True,isOnline=False)
    return start,tx,end


class RetireFighterTests(unittest.TestCase):
    def test_exact_nonlethal_commander_and_destroyed_fighter_outcome(self):
        for ah in (1,6):
            for th in (1,2):
                start,tx,end=fixture(ah,th)
                outcome=attack({'before':start,'tx':tx},HASH)
                self.assertEqual(outcome['commanderHealth'],str(ah))
                self.assertEqual(after({'before':start,'after':end,'outcome':outcome}),{'verified':True})
                end['target']['structAttributes']['health']='1'
                with self.assertRaises(ValueError):after({'before':start,'after':end,'outcome':outcome})

    def test_preparation_refuses_healthy_defended_wrong_or_unavailable_fighters(self):
        for change in (lambda d:d['target']['structAttributes'].update(health='3'),
            lambda d:d['target'].update(structDefenders=['5-3']),
            lambda d:d['target']['Struct'].update(owner='1-6'),
            lambda d:d['target']['Struct'].update(type='2'),
            lambda d:d['attacker']['structAttributes'].update(isOnline=False)):
            start,_,_=fixture();change(start)
            with self.assertRaises(ValueError):before(start,'5-1','5-2','1-6','1-3','9-6','9-3')

    def test_retirement_catalog_and_receipt_must_guarantee_and_confirm_the_kill(self):
        doc={'attacker':{'StructType':{'id':'1','primaryWeaponControl':'guided','primaryWeaponCharge':'3',
             'primaryWeaponAmbits':'32','primaryWeaponTargets':'1','primaryWeaponShots':'1','primaryWeaponDamage':'2',
             'primaryWeaponShotSuccessRateNumerator':'1','primaryWeaponShotSuccessRateDenominator':'1'}},
             'target':{'StructType':{'id':'3','maxHealth':'3'}}}
        self.assertEqual(catalog(doc),{'verified':True})
        for role,key,value in [('attacker','primaryWeaponDamage','1'),('attacker','primaryWeaponShotSuccessRateDenominator','3'),
                ('target','guidedDefensiveSuccessRateNumerator','1'),('target','postDestructionDamage','1')]:
            bad=copy.deepcopy(doc);bad[role]['StructType'][key]=value
            with self.assertRaises(ValueError):catalog(bad)
        for key,value in [('damage','1'),('targetCountered',True),('targetHealthAfter','1'),
                          ('targetStructId','5-99'),('eventAttackDefenderCounterDetail',[{}])]:
            start,tx,_=fixture();row=json.loads(tx['events'][0]['attributes'][0]['value'])
            row['eventAttackShotDetail'][0][key]=value;tx['events'][0]['attributes'][0]['value']=json.dumps(row)
            with self.assertRaises(ValueError):attack({'before':start,'tx':tx},HASH)
        start,tx,_=fixture();tx['code']=1
        with self.assertRaises(ValueError):attack({'before':start,'tx':tx},HASH)


if __name__=='__main__':unittest.main()
