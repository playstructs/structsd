#!/usr/bin/env python3
"""Recover fixture grid associations without treating list order as identity."""
import argparse
import json
import re
import subprocess
import sys

from struct_inventory import next_cursor, positive_integer


def object_id(value, prefixes, label):
    if not isinstance(value, str) or not re.fullmatch(r"[0-9]+-[1-9][0-9]*", value):
        raise ValueError(f"invalid {label}")
    prefix, index = value.split("-")
    if prefix not in prefixes:
        raise ValueError(f"invalid {label} prefix")
    positive_integer(index, 2**64 - 1, label)
    return value


def committed_height(query, height=None):
    if height is None:
        status = query("status")
        if not isinstance(status, dict) or not isinstance(status.get("sync_info"), dict):
            raise ValueError("invalid node status")
        height = status["sync_info"].get("latest_block_height")
    positive_integer(height, 2**63 - 1, "grid recovery height")
    return height


def feeder(query, source, destination, controller, height=None, page_size=100, max_pages=10000):
    object_id(source, ("0", "1"), "fixture source")
    # An empty destination recovers an interrupted create-before-connect fixture.
    if destination != "":
        object_id(destination, ("4",), "fixture substation")
    object_id(controller, ("0", "1"), "fixture controller")
    if type(page_size) is not int or not 1 <= page_size <= 100:
        raise ValueError("invalid allocation page size")
    if type(max_pages) is not int or not 1 <= max_pages <= 10000:
        raise ValueError("invalid allocation page budget")
    height = committed_height(query, height)
    cursor, seen_cursors, seen_ids, matches = "", set(), set(), []
    for _ in range(max_pages):
        args = ["query", "structs", "allocation-all-by-source", source,
            "--page-limit", str(page_size), "--height", height]
        if cursor:
            args.extend(("--page-key", cursor))
        document = query(*args)
        following = next_cursor(document)
        if set(document) - {"Allocation", "pagination"}:
            raise ValueError("unexpected allocation response fields")
        rows = document.get("Allocation")
        if rows is None:
            rows = []
        if not isinstance(rows, list) or len(rows) > page_size:
            raise ValueError("invalid allocation page")
        for row in rows:
            if not isinstance(row, dict):
                raise ValueError("invalid allocation record")
            positive_integer(row.get("index"), 2**64 - 1, "allocation index")
            if row.get("id") != "6-" + row["index"] or row["id"] in seen_ids:
                raise ValueError("invalid or duplicate allocation identity")
            seen_ids.add(row["id"])
            if row.get("sourceObjectId") != source:
                raise ValueError("allocation query returned a different source")
            object_id(row.get("controller"), ("0", "1"), "allocation controller")
            target = row.get("destinationId", "")
            if target != "":
                object_id(target, ("0", "1", "3", "4"), "allocation destination")
            if row.get("type") not in ("automated", "dynamic", "static", "providerAgreement"):
                raise ValueError("invalid allocation type")
            if target == destination and row["controller"] == controller and row["type"] == "dynamic":
                matches.append(row["id"])
        if not following:
            if len(matches) > 1:
                raise ValueError("ambiguous fixture feeder; multiple allocations match its associations")
            return matches[0] if matches else ""
        if following in seen_cursors:
            raise ValueError("allocation pagination cursor cycle")
        seen_cursors.add(following)
        cursor = following
    raise ValueError("allocation recovery exceeded its page budget")


def recover_grid(query, player, page_size=100, max_pages=10000):
    object_id(player, ("1",), "fixture player")
    height = committed_height(query)
    def record(kind, identifier):
        document = query("query", "structs", kind.lower(), identifier, "--height", height)
        row = document.get(kind) if isinstance(document, dict) else None
        if not isinstance(row, dict) or row.get("id") != identifier:
            raise ValueError(f"grid recovery returned the wrong {kind}")
        return row
    owner = record("Player", player)
    guild_id = object_id(owner.get("guildId"), ("0",), "fixture guild")
    # Alice's actual connection remains stable while guild settings temporarily
    # point new members at a throwaway substation.
    station_id = object_id(owner.get("substationId"), ("4",), "fixture substation")
    guild = record("Guild", guild_id)
    station = record("Substation", station_id)
    if station.get("owner") != player:
        raise ValueError("main fixture substation belongs to a different owner")
    reactor_id = object_id(guild.get("primaryReactorId"), ("3",), "fixture reactor")
    reactor = record("Reactor", reactor_id)
    validator = reactor.get("validator")
    if not isinstance(validator, str) or not validator:
        raise ValueError("reactor has no validator association")
    document = query("query", "staking", "validator", validator, "--height", height)
    row = document.get("validator") if isinstance(document, dict) else None
    if not isinstance(row, dict) or row.get("operator_address") != validator:
        raise ValueError("reactor validator association did not resolve")
    allocation = feeder(query, player, station_id, player, height, page_size, max_pages)
    if not allocation:
        raise ValueError("main fixture feeder is missing")
    return {"queryHeight": height, "playerId": player, "guildId": guild_id,
        "substationId": station_id, "reactorId": reactor_id,
        "validatorAddress": validator, "allocationId": allocation}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("mode", choices=("grid", "feeder"))
    parser.add_argument("--player")
    parser.add_argument("--source")
    parser.add_argument("--destination")
    parser.add_argument("--controller")
    parser.add_argument("--height")
    parser.add_argument("--page-size", type=int, default=100)
    parser.add_argument("--max-pages", type=int, default=10000)
    # Split at -- so all following flags belong to the actual chain CLI.
    arguments = sys.argv[1:]
    boundary = arguments.index("--") if "--" in arguments else len(arguments)
    args = parser.parse_args(arguments[:boundary])
    command = arguments[boundary + 1:]
    try:
        if not command:
            raise ValueError("grid recovery requires a CLI command")
        def query(*options):
            response = subprocess.run([*command, *options], check=True, capture_output=True, text=True, timeout=30)
            return json.loads(response.stdout)
        if args.mode == "grid":
            if args.height is not None:
                raise ValueError("grid recovery selects its own committed height")
            print(json.dumps(recover_grid(query, args.player, args.page_size, args.max_pages)))
        else:
            print(feeder(query, args.source, args.destination, args.controller, args.height, args.page_size, args.max_pages))
    except (ValueError, TypeError, KeyError, OSError, subprocess.SubprocessError) as error:
        parser.exit(1, f"Fixture grid recovery failed: {error}\n")


if __name__ == "__main__":
    main()
