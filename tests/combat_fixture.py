"""Validate identity, lifecycle and resolved health for required combat fixtures."""
import argparse
import json
import re
import sys
from agreement_terms import identity


def fields(actual, expected):
    if not isinstance(actual, dict) or any(type(actual.get(k)) is not type(v) or actual[k] != v for k, v in expected.items()):
        raise ValueError('fixture identity or attack outcome differs')


def attributes(document, sid, owner, kind, location, ambit):
    identity(sid, '5')
    identity(owner, '1')
    prefix = location.split('-')[0]
    if prefix not in ('2', '9'):
        raise ValueError('fixture location must be a planet or fleet')
    identity(location, prefix)
    fields(document.get('Struct'), {'id': sid, 'index': sid.split('-')[1], 'owner': owner,
                                  'type': kind, 'locationId': location, 'operatingAmbit': ambit})
    attrs = document.get('structAttributes')
    if not isinstance(attrs, dict):
        raise ValueError('missing fixture attributes')
    for flag in ('isBuilt', 'isOnline', 'isDestroyed'):
        if type(attrs.get(flag, False)) is not bool:
            raise ValueError('malformed fixture flag')
    health = attrs.get('health', '0')
    if not isinstance(health, str) or not re.fullmatch('0|[1-9][0-9]*', health) or len(health) > 20 or int(health) > 2**64-1:
        raise ValueError('malformed fixture health')
    return attrs, health


def fixture(document, sid, owner, kind, location, ambit):
    attrs, health = attributes(document, sid, owner, kind, location, ambit)
    if attrs.get('isDestroyed', False):
        return 'destroyed'
    if not attrs.get('isBuilt', False):
        return 'build'
    if health == '0':
        raise ValueError('built fixture has no health without destruction')
    return 'ready' if attrs.get('isOnline', False) else 'activate'


def state(document, sid, owner, kind, location, ambit, health, destroyed):
    attrs, actual = attributes(document, sid, owner, kind, location, ambit)
    if actual != health or attrs.get('isDestroyed', False) != destroyed:
        raise ValueError('canonical attack state disagrees with receipt')
    if not destroyed and fixture(document, sid, owner, kind, location, ambit) != 'ready':
        raise ValueError('surviving fixture is not ready')
    return {'verified': True}


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command', choices=('fixture', 'state'))
    for name in ('sid', 'owner', 'kind', 'location', 'ambit'):
        parser.add_argument(name)
    parser.add_argument('health', nargs='?')
    parser.add_argument('destroyed', nargs='?', choices=('true', 'false'))
    args = vars(parser.parse_args())
    mode = args.pop('command')
    if mode == 'fixture':
        if args.pop('health') is not None or args.pop('destroyed') is not None:
            parser.error('fixture takes no expected health arguments')
    else:
        if args['health'] is None or args['destroyed'] is None:
            parser.error('state requires expected health and destruction')
        args['destroyed'] = args['destroyed'] == 'true'
    try:
        result = globals()[mode](json.load(sys.stdin), **args)
        print(result if isinstance(result, str) else json.dumps(result))
    except (ValueError, TypeError, KeyError, AttributeError) as error:
        parser.exit(1, f'Invalid combat fixture: {error}\n')
