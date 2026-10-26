"""Price phase 15 from a pinned, restored one-vault generator fixture."""
import argparse
import json
import re
import sys

from struct_inventory import next_cursor

MAX = 2**64 - 1


def amount(value):
    if not isinstance(value, str) or len(value) > 20 or not re.fullmatch(r'0|[1-9][0-9]*', value) or int(value) > MAX:
        raise ValueError('noncanonical uint64 in generator fixture')
    return int(value)


def verify(document, generator, owner, vault, planet):
    unit = document['generator']['Struct']
    attrs = document['generator']['structAttributes']
    grid = document['generator']['gridAttributes']
    player = document['player']
    catalog = document['catalog']['StructType']
    if (unit.get('id'), unit.get('owner'), unit.get('type'), unit.get('locationId'), unit.get('locationType'), unit.get('operatingAmbit')) != (generator, owner, '20', planet, 'planet', 'land'):
        raise ValueError('generator identity or placement differs from fixture')
    if attrs.get('isBuilt') is not True or attrs.get('isOnline') is not True or attrs.get('isDestroyed', False) is not False:
        raise ValueError('generator is not built, living and online')
    health = amount(attrs['health'])
    if not 1 <= health <= 8 or player['Player']['id'] != owner or player['Player']['planetId'] != planet:
        raise ValueError('generator health or player home differs from fixture')
    if (catalog.get('id'), catalog.get('generatingRate'), catalog.get('maxHealth'), catalog.get('attackReduction')) != ('20', '2', '8', '1'):
        raise ValueError('field generator catalog differs from destruction fixture')
    rows = document['positions'].get('Infusion', [])
    if not isinstance(rows, list) or len(rows) > 1 or next_cursor(document['positions']):
        raise ValueError('phase 15 requires a complete zero-or-one vault position inventory')
    fuel = amount(grid.get('fuel', '0'))
    power, ready = 0, True
    if rows:
        row = rows[0]
        if (row.get('destinationId'), row.get('destinationType'), row.get('address'), row.get('ownerId') or row.get('playerId')) != (generator, 'struct', vault, owner):
            raise ValueError('generator position does not belong to the fixture vault')
        if amount(row.get('fuel', '0')) != fuel or amount(row.get('ratio', '0')) != 2 or row.get('commission') not in ('0', '0.000000000000000000'):
            raise ValueError('generator position fuel, ratio or commission differs from fixture')
        if amount(row.get('defusing', '0')) != 0:
            raise ValueError('generator position carries reactor accounting')
        disabled = row.get('generatorDisabled', False)
        if type(disabled) is not bool:
            raise ValueError('invalid generator disabled flag')
        power = amount(row.get('power', '0'))
        if power != (0 if disabled else fuel * 2):
            raise ValueError('generator power disagrees with its retained fuel')
        ready = not disabled
    elif fuel:
        raise ValueError('generator fuel has no fixture position')
    capacity = amount(player['gridAttributes'].get('capacity', '0'))
    if capacity < power or capacity + 2000000 > MAX or (fuel + 1000000) * 2 > MAX:
        raise ValueError('generator fixture has inconsistent or exhausted capacity')
    return {'ready': ready, 'fuelBefore': str(fuel), 'powerBefore': str(power),
            'capacityBefore': str(capacity), 'capacityAfterInfusion': str(capacity + 2000000),
            'capacityAfterDestruction': str(capacity - power), 'fuelAfterInfusion': str(fuel + 1000000)}


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('generator')
    parser.add_argument('owner')
    parser.add_argument('vault')
    parser.add_argument('planet')
    args = parser.parse_args()
    try:
        print(json.dumps(verify(json.load(sys.stdin), args.generator, args.owner, args.vault, args.planet)))
    except (ValueError, KeyError, TypeError) as error:
        parser.exit(1, f'Invalid generator lifecycle fixture: {error}\n')
