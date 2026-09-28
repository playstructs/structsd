#!/usr/bin/env python3
"""Read a complete, height-pinned struct inventory for lifecycle recovery."""
import argparse
import base64
import binascii
import json
import re
import subprocess
import sys


def positive_integer(value, maximum, label):
    if not isinstance(value, str) or not re.fullmatch(r"[1-9][0-9]*", value) or len(value) > 20:
        raise ValueError(f"invalid {label}")
    number = int(value)
    if number > maximum:
        raise ValueError(f"{label} is out of range")
    return number


def next_cursor(document):
    if not isinstance(document, dict) or not isinstance(document.get("pagination"), dict):
        raise ValueError("struct inventory requires pagination metadata")
    pagination = document["pagination"]
    if "next_key" in pagination and "nextKey" in pagination:
        raise ValueError("ambiguous struct pagination cursor")
    cursor = pagination.get("next_key", pagination.get("nextKey"))
    if cursor is None or cursor == "":
        return ""
    if not isinstance(cursor, str):
        raise ValueError("invalid struct pagination cursor")
    try:
        decoded = base64.b64decode(cursor, validate=True)
    except (ValueError, binascii.Error) as error:
        raise ValueError("invalid struct pagination cursor") from error
    if not decoded or base64.b64encode(decoded).decode("ascii") != cursor:
        raise ValueError("noncanonical struct pagination cursor")
    return cursor


def records(document):
    if set(document) - {"Struct", "pagination", "queryHeight"}:
        raise ValueError("unexpected struct inventory response fields")
    rows = document.get("Struct", [])
    if rows is None:
        rows = []
    if not isinstance(rows, list):
        raise ValueError("invalid struct inventory records")
    for row in rows:
        if not isinstance(row, dict):
            raise ValueError("invalid struct inventory record")
        positive_integer(row.get("index"), 2**64 - 1, "struct index")
        positive_integer(row.get("type"), 2**64 - 1, "struct type")
        if row.get("id") != "5-" + row["index"]:
            raise ValueError("struct identity disagrees with its index")
        if not isinstance(row.get("owner"), str) or not re.fullmatch(r"[01]-[1-9][0-9]*", row["owner"]):
            raise ValueError("invalid struct owner")
    return rows


def collect(query, page_size=100, max_pages=10000):
    if type(page_size) is not int or not 1 <= page_size <= 100:
        raise ValueError("invalid struct inventory page size")
    if type(max_pages) is not int or not 1 <= max_pages <= 10000:
        raise ValueError("invalid struct inventory page budget")
    status = query("status")
    if not isinstance(status, dict) or not isinstance(status.get("sync_info"), dict):
        raise ValueError("invalid node status")
    height = status["sync_info"].get("latest_block_height")
    positive_integer(height, 2**63 - 1, "inventory block height")
    result, seen_ids, seen_cursors = [], set(), set()
    cursor = ""
    for _ in range(max_pages):
        arguments = ["query", "structs", "struct-all", "--page-limit", str(page_size), "--height", height]
        if cursor:
            arguments.extend(("--page-key", cursor))
        page = query(*arguments)
        following = next_cursor(page)
        rows = records(page)
        if len(rows) > page_size:
            raise ValueError("struct query exceeded the requested page size")
        for row in rows:
            if row["id"] in seen_ids:
                raise ValueError("duplicate struct in paginated inventory")
            seen_ids.add(row["id"])
            result.append(row)
        if not following:
            return {"Struct": result, "pagination": {}, "queryHeight": height}
        if following in seen_cursors:
            raise ValueError("struct pagination cursor cycle")
        seen_cursors.add(following)
        cursor = following
    raise ValueError("struct inventory exceeded its page budget")


def select(document, owner, type_id, nth="1", newest=False):
    if next_cursor(document):
        raise ValueError("cannot select a struct from an incomplete inventory")
    positive_integer(document.get("queryHeight"), 2**63 - 1, "inventory block height")
    positive_integer(type_id, 2**64 - 1, "requested struct type")
    offset = positive_integer(nth, 2**64 - 1, "requested struct position") - 1
    if not isinstance(owner, str) or not re.fullmatch(r"[01]-[1-9][0-9]*", owner):
        raise ValueError("invalid requested struct owner")
    rows = records(document)
    if len({row["id"] for row in rows}) != len(rows):
        raise ValueError("duplicate struct in inventory")
    candidates = sorted((row for row in rows if row["owner"] == owner and row["type"] == type_id),
        key=lambda row: int(row["index"]), reverse=newest)
    return candidates[offset]["id"] if offset < len(candidates) else ""


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest="mode", required=True)
    collector = commands.add_parser("collect")
    collector.add_argument("--page-size", type=int, default=100)
    collector.add_argument("--max-pages", type=int, default=10000)
    collector.add_argument("command", nargs=argparse.REMAINDER)
    selector = commands.add_parser("select")
    selector.add_argument("owner")
    selector.add_argument("type_id")
    selector.add_argument("nth", nargs="?", default="1")
    selector.add_argument("--newest", action="store_true")
    args = parser.parse_args()
    try:
        if args.mode == "select":
            print(select(json.load(sys.stdin), args.owner, args.type_id, args.nth, args.newest))
        else:
            command = args.command[1:] if args.command[:1] == ["--"] else args.command
            if not command:
                raise ValueError("struct inventory requires a CLI command")
            def query(*arguments):
                response = subprocess.run([*command, *arguments], capture_output=True, text=True, check=True, timeout=30)
                return json.loads(response.stdout)
            print(json.dumps(collect(query, args.page_size, args.max_pages), separators=(",", ":")))
    except (ValueError, KeyError, TypeError, OSError, subprocess.SubprocessError) as error:
        parser.exit(1, f"Struct inventory failed: {error}\n")


if __name__ == "__main__":
    main()
