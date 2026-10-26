"""Verify the complete receipt for phase 15's one-Alpha generator top-up."""
import argparse
import json
import sys

from agreement_terms import field, identity
from generator_lifecycle import amount
from economic_assertions import confirmed, records


def verify(tx, txhash, generator, owner, vault, primary, signer, module, fuel, capacity):
    height = confirmed(tx, txhash)
    identity(generator, '5')
    identity(owner, '1')
    if any(not isinstance(address, str) or not address for address in (vault, primary, signer, module)):
        raise ValueError('fixture addresses are required')
    if vault in (primary, signer, module) or module in (primary, signer):
        raise ValueError('vault, module and player keys must be distinct')
    if amount(fuel) < 1000000 or amount(fuel) * 2 > 2**64 - 1 or amount(capacity) < amount(fuel) * 2:
        raise ValueError('invalid expected generator fuel or capacity')
    expected_message = {'@type': '/structs.structs.MsgStructGeneratorInfuse', 'creator': signer,
                        'structId': generator, 'infuseAmount': '1000000ualpha'}
    if tx['tx']['body']['messages'] != [expected_message]:
        raise ValueError('receipt is not the requested single generator infusion')
    for event in tx['events']:
        if event['type'] in ('structs.structs.EventAlphaInfuse', 'structs.structs.EventInfusion', 'structs.structs.EventGrid'):
            if field(event, 'msg_index') != '0':
                raise ValueError('generator event is not attributed to its message')
    expected_bank = [
        ('coin_spent', {'spender': vault}),
        ('coin_received', {'receiver': module}),
        ('transfer', {'sender': vault, 'recipient': module}),
        ('coin_spent', {'spender': module}),
        ('burn', {'burner': module}),
    ]
    bank = [event for event in tx['events']
            if event['type'] in ('coin_spent', 'coin_received', 'transfer', 'burn', 'coinbase', 'mint')]
    if len(bank) != len(expected_bank):
        raise ValueError('generator infusion must transfer and burn exactly once')
    for event, (kind, attrs) in zip(bank, expected_bank):
        expected = dict(attrs, amount='1000000ualpha', msg_index='0')
        if event['type'] != kind or any(field(event, key) != value for key, value in expected.items()):
            raise ValueError('generator bank event sequence differs from vault burn')
    details = records(tx, 'EventAlphaInfuse', 'eventAlphaInfuseDetail')
    if details != [{'playerId': owner, 'primaryAddress': primary, 'amount': '1000000', 'vaultAddress': vault}]:
        raise ValueError('alpha infusion event has incorrect player, primary or vault')
    rows = records(tx, 'EventInfusion', 'infusion')
    if len(rows) != 1:
        raise ValueError('one generator infusion position event is required')
    row = rows[0]
    expected = {'destinationType': 'struct', 'destinationId': generator, 'address': vault,
                'fuel': fuel, 'power': str(int(fuel) * 2), 'ratio': '2'}
    if any(row.get(key) != value for key, value in expected.items()) or (row.get('ownerId') or row.get('playerId')) != owner:
        raise ValueError('infusion position event differs from the expected generator output')
    if (row.get('commission') not in ('0', '0.000000000000000000') or row.get('defusing', '0') != '0'
            or row.get('generatorDisabled', False) is not False):
        raise ValueError('generator infusion carries reactor accounting or disabled power')
    grids = records(tx, 'EventGrid', 'gridRecord')
    for key, value in (('1-' + generator, fuel), ('2-' + owner, capacity)):
        if [row for row in grids if row.get('attributeId') == key] != [{'attributeId': key, 'value': value}]:
            raise ValueError('generator fuel or player capacity grid event differs from expected output')
    return str(height)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('txhash', 'generator', 'owner', 'vault', 'primary', 'signer', 'module', 'fuel', 'capacity'):
        parser.add_argument(name)
    args = parser.parse_args()
    try:
        print(verify(json.load(sys.stdin), **vars(args)))
    except (ValueError, KeyError, TypeError) as error:
        parser.exit(1, f'Invalid generator infusion: {error}\n')
