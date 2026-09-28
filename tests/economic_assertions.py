"""Strict integer parsing for off-chain economic test evidence."""
import json
import re


def integer(value, maximum, field):
    if not isinstance(value, str) or re.fullmatch(r"0|[1-9][0-9]*", value) is None:
        raise ValueError(f"{field} must be a canonical decimal string")
    if len(value) > len(str(maximum)) or int(value) > maximum:
        raise ValueError(f"{field} exceeds its range")
    return int(value)


def field(event, name):
    values = [item['value'] for item in event['attributes'] if item['key'] == name]
    if len(values) != 1 or not isinstance(values[0], str):
        raise ValueError(f'event requires one {name}')
    return values[0]


def confirmed(tx, txhash):
    if (not isinstance(tx, dict) or not isinstance(txhash, str) or re.fullmatch('[0-9A-Fa-f]{64}', txhash) is None
            or not isinstance(tx.get('txhash'), str) or tx['txhash'].upper() != txhash.upper()
            or type(tx.get('code')) not in (str, int) or tx['code'] not in (0, '0')):
        raise ValueError('transaction must be confirmed, successful and hash-matched')
    height = integer(tx.get('height'), 2**63-1, 'height')
    if height == 0 or not isinstance(tx.get('events'), list):
        raise ValueError('invalid delivered transaction')
    for event in tx['events']:
        if (not isinstance(event, dict) or not isinstance(event.get('type'), str)
                or not isinstance(event.get('attributes'), list)
                or any(not isinstance(item, dict) or not isinstance(item.get('key'), str)
                       or not isinstance(item.get('value'), str) for item in event['attributes'])):
            raise ValueError('invalid transaction event')
    return height


def records(tx, event_type, name):
    rows = [json.loads(field(event, name)) for event in tx['events']
            if event['type'] == 'structs.structs.' + event_type]
    shape = str if name == 'objectId' else dict
    if any(not isinstance(row, shape) for row in rows):
        raise ValueError('invalid typed event record')
    return rows


