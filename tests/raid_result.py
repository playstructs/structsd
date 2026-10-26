"""Verify Phase 10's raid receipt and state at explicit committed heights."""
import argparse
import json
import re
import sys

from economic_assertions import confirmed, records
from economic_assertions import integer

U64 = 2**64 - 1


def same(actual, expected, label):
    if type(actual) is not type(expected) or actual != expected:
        raise ValueError(f'{label} differs from the raid fixture')


def number(row, key):
    return integer(row.get(key, '0'), U64, key)


def state(document):
    for name, prefix in (('victim_id', '1'), ('raider_id', '1'), ('planet_id', '2'),
                         ('home_id', '2'), ('fleet_id', '9')):
        if not isinstance(document[name], str) or not re.fullmatch(prefix + r'-[1-9][0-9]*', document[name]):
            raise ValueError('invalid raid identity')
    if document['victim_id'] == document['raider_id'] or document['planet_id'] == document['home_id']:
        raise ValueError('raid must involve distinct players and planets')
    height = integer(document['height'], 2**63 - 1, 'height')
    if height == 0 or type(document['require_ore']) is not bool:
        raise ValueError('invalid raid height or ore requirement')
    victim, raider = document['victim']['Player'], document['raider']['Player']
    planet, fleet = document['planet']['Planet'], document['fleet']['Fleet']
    for actual, expected, label in (
        (victim['id'], document['victim_id'], 'victim'),
        (raider['id'], document['raider_id'], 'raider'),
        (planet['id'], document['planet_id'], 'planet'),
        (fleet['id'], document['fleet_id'], 'fleet'),
        (planet['owner'], document['victim_id'], 'planet owner'),
        (fleet['owner'], document['raider_id'], 'fleet owner'),
        (victim['planetId'], document['planet_id'], 'active victim planet'),
        (raider['planetId'], document['home_id'], 'raider home'),
        (raider['fleetId'], document['fleet_id'], 'raider fleet'),
        (planet.get('status', 'active'), 'active', 'planet status'),
        (fleet['locationType'], 'planet', 'fleet location type'),
    ):
        same(actual, expected, label)
    for player in (victim, raider):
        if not isinstance(player['primaryAddress'], str) or not player['primaryAddress']:
            raise ValueError('player primary address is missing')
    if victim['primaryAddress'] == raider['primaryAddress']:
        raise ValueError('player addresses must differ')
    # Missing protobuf zero values are valid; a missing response/map is not.
    for response, key in (('victim', 'gridAttributes'), ('raider', 'gridAttributes'),
                          ('planet', 'gridAttributes'), ('planet', 'planetAttributes')):
        if not isinstance(document[response][key], dict):
            raise ValueError('missing raid attributes')
    attributes = document['planet']['planetAttributes']
    return {'height': height, 'victim': victim, 'raider': raider, 'planet': planet, 'fleet': fleet,
            'victim_ore': number(document['victim']['gridAttributes'], 'ore'),
            'raider_ore': number(document['raider']['gridAttributes'], 'ore'),
            'buried_ore': number(document['planet']['gridAttributes'], 'ore'),
            'attributes': {key: number(attributes, key) for key in (
                'blockStartRaid', 'blockRaiderArrived', 'blockStartOreMine', 'blockStartOreRefine',
                'oreMiningActiveQuantity', 'oreRefiningActiveQuantity', 'planetaryShield')}}


def before(document):
    row = state(document)
    same(row['fleet']['locationId'], document['planet_id'], 'raiding location')
    same(row['fleet'].get('status', 'onStation'), 'away', 'raiding fleet status')
    for key in ('locationListForward', 'locationListBackward'):
        same(row['fleet'].get(key, ''), '', 'sole visiting fleet')
    for key in ('locationListStart', 'locationListLast'):
        same(row['planet'].get(key, ''), document['fleet_id'], 'raid queue')
    same(number(row['planet'], 'locationListCount'), 1, 'raid queue count')
    attributes = row['attributes']
    if not 0 < attributes['blockRaiderArrived'] <= attributes['blockStartRaid'] <= row['height']:
        raise ValueError('raid clocks must be anchored before the observation')
    if document['require_ore'] and row['victim_ore'] == 0:
        raise ValueError('mining-enabled fixture must have ore to seize')
    for quantity, clock in (('oreMiningActiveQuantity', 'blockStartOreMine'),
                            ('oreRefiningActiveQuantity', 'blockStartOreRefine')):
        if document['require_ore'] and attributes[quantity] == 0:
            raise ValueError('required ore system is inactive')
        if attributes[clock] > row['height'] or (attributes[quantity] and attributes[clock] == 0):
            raise ValueError('invalid ore clock')
    if row['raider_ore'] + row['victim_ore'] > U64:
        raise ValueError('raid fixture would overflow stored ore')
    return row


def receipt(document, tx, txhash):
    row = before(document)
    height = confirmed(tx, txhash)
    if height <= row['height']:
        raise ValueError('raid receipt must follow its baseline')
    thefts = records(tx, 'EventOreTheft', 'eventOreTheftDetail')
    raids = records(tx, 'EventRaid', 'eventRaidDetail')
    hashes = records(tx, 'EventHashSuccess', 'eventHashSuccessDetail')
    if (len(thefts), len(raids), len(hashes)) != (1, 1, 1):
        raise ValueError('one theft, successful raid and proof event required')
    for actual, expected in ((thefts[0], {
        'victimPlayerId': document['victim_id'], 'victimPrimaryAddress': row['victim']['primaryAddress'],
        'thiefPlayerId': document['raider_id'], 'thiefPrimaryAddress': row['raider']['primaryAddress'],
    }), (raids[0], {'fleetId': document['fleet_id'], 'planetId': document['planet_id'], 'status': 'raidSuccessful'}),
        (hashes[0], {'callerAddress': row['raider']['primaryAddress'], 'category': 'raid',
                     'objectId': document['fleet_id'], 'planetId': document['planet_id']})):
        for key, value in expected.items():
            same(actual.get(key), value, key)
    same(number(thefts[0], 'amount'), row['victim_ore'], 'theft amount')
    # Typed SDK events use the original proto field name, not query JSON's camel case.
    same(number(raids[0], 'seized_ore'), row['victim_ore'], 'raid amount')
    if not 1 <= number(hashes[0], 'difficulty') <= 64:
        raise ValueError('invalid raid proof difficulty')
    return height


def after(original, tx, txhash, current):
    height = receipt(original, tx, txhash)
    old, new = before(original), state(current)
    same(new['height'], height, 'post-raid query height')
    for key in ('victim_id', 'raider_id', 'planet_id', 'home_id', 'fleet_id', 'require_ore'):
        same(current[key], original[key], key)
    for player in ('victim', 'raider'):
        same(new[player]['primaryAddress'], old[player]['primaryAddress'], player + ' address')
    same(new['victim_ore'], 0, 'victim ore')
    same(new['raider_ore'], old['raider_ore'] + old['victim_ore'], 'raider ore')
    same(new['buried_ore'], old['buried_ore'], 'unmined planet ore')
    same(new['fleet']['locationId'], original['home_id'], 'fleet recall')
    same(new['fleet'].get('status', 'onStation'), 'onStation', 'fleet recalled status')
    for key in ('locationListForward', 'locationListBackward'):
        same(new['fleet'].get(key, ''), '', 'recalled fleet queue links')
    for key in ('locationListStart', 'locationListLast'):
        same(new['planet'].get(key, ''), '', 'cleared planet queue')
    same(number(new['planet'], 'locationListCount'), 0, 'cleared queue count')
    for key in ('blockStartRaid', 'blockRaiderArrived'):
        same(new['attributes'][key], 0, key)
    for key in ('oreMiningActiveQuantity', 'oreRefiningActiveQuantity', 'planetaryShield'):
        same(new['attributes'][key], old['attributes'][key], key)
    arrival = old['attributes']['blockRaiderArrived']
    for quantity, clock in (('oreMiningActiveQuantity', 'blockStartOreMine'),
                            ('oreRefiningActiveQuantity', 'blockStartOreRefine')):
        previous = old['attributes'][clock]
        expected = previous + height - max(previous, arrival) if old['attributes'][quantity] else previous
        same(new['attributes'][clock], expected, clock + ' exact pause shift')
    return {'height': str(height), 'seized_ore': str(old['victim_ore'])}


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('operation', choices=('before', 'receipt', 'after'))
    parser.add_argument('--before')
    parser.add_argument('--receipt')
    parser.add_argument('--txhash')
    args = parser.parse_args()
    try:
        incoming = json.load(sys.stdin)
        if args.operation == 'before':
            before(incoming)
            print(json.dumps(incoming))
        else:
            with open(args.before) as stream:
                original = json.load(stream)
            if args.operation == 'receipt':
                print(receipt(original, incoming, args.txhash))
            else:
                with open(args.receipt) as stream:
                    tx = json.load(stream)
                print(json.dumps(after(original, tx, args.txhash, incoming)))
    except (ValueError, KeyError, TypeError, OSError, AttributeError) as error:
        parser.exit(1, f'Invalid raid result: {error}\n')
