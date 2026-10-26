"""Verify EndBlock cleanup for phase 15's single-position generator fixture."""
import argparse
import json
import sys

from agreement_terms import field, identity
from generator_attack import verify as attack
from generator_lifecycle import amount
from economic_assertions import records


def verify(document, txhash, attacker, generator, owner, vault, capacity):
    identity(owner, '1')
    amount(capacity)
    if not isinstance(vault, str) or not vault:
        raise ValueError('generator vault is required')
    height = attack(document['receipt'], txhash, attacker, generator, '1')['height']
    block = document['block']
    if not isinstance(block, dict) or not isinstance(block.get('finalize_block_events'), list):
        raise ValueError('cleanup must use the confirmed lethal block results')
    # SDK CLI uses encoding/json (number); CometBFT RPC uses its string encoding.
    # Do not let bool, floats, signed/zero-padded strings or overflow coerce to it.
    block_height = block.get('height')
    if type(block_height) is int:
        if not 0 < block_height < 2**63:
            raise ValueError('invalid block-results height')
        block_height = str(block_height)
    elif type(block_height) is not str or not 0 < amount(block_height) < 2**63:
        raise ValueError('invalid block-results height')
    if block_height != height:
        raise ValueError('cleanup must use the confirmed lethal block results')
    events = block['finalize_block_events']
    for event in events:
        if (not isinstance(event, dict) or not isinstance(event.get('type'), str)
                or not isinstance(event.get('attributes'), list)
                or any(not isinstance(item, dict) or not isinstance(item.get('key'), str)
                       or not isinstance(item.get('value'), str) for item in event['attributes'])):
            raise ValueError('invalid block event')
        if event['type'] in ('structs.structs.EventDelete', 'structs.structs.EventGrid', 'structs.structs.EventInfusion'):
            # Only this generator's effects below require EndBlock attribution;
            # unrelated BeginBlock sweeps may share the block legitimately.
            field(event, 'mode')
    end = {'events': [event for event in events if any(item['key'] == 'mode' and item['value'] == 'EndBlock'
                                                    for item in event['attributes'])]}
    deletes = records(end, 'EventDelete', 'objectId')
    all_deletes = records({'events': events}, 'EventDelete', 'objectId')
    wanted = [generator + '-' + vault]
    if ([key for key in deletes if key.startswith(generator + '-')] != wanted
            or [key for key in all_deletes if key.startswith(generator + '-')] != wanted):
        raise ValueError('generator position must be deleted exactly once in EndBlock')
    all_rows = records({'events': events}, 'EventInfusion', 'infusion')
    if any(row.get('destinationId') == generator for row in all_rows):
        raise ValueError('destroyed generator may not publish a surviving infusion')
    expected = {prefix + '-' + generator: '0' for prefix in ('1', '2', '3', '5')}
    expected['2-' + owner] = capacity
    grid = records(end, 'EventGrid', 'gridRecord')
    for key, value in expected.items():
        if [row for row in grid if row.get('attributeId') == key] != [{'attributeId': key, 'value': value}]:
            raise ValueError('generator cleanup must emit its exact zeroes and remaining player capacity')
    return height


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('txhash', 'attacker', 'generator', 'owner', 'vault', 'capacity'):
        parser.add_argument(name)
    args = parser.parse_args()
    try:
        print(verify(json.load(sys.stdin), **vars(args)))
    except (ValueError, KeyError, TypeError) as error:
        parser.exit(1, f'Invalid generator destruction: {error}\n')
