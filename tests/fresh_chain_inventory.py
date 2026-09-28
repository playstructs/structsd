#!/usr/bin/env python3
"""Prove the lifecycle fixture's empty-state precondition with bounded queries."""
import argparse
import json
import subprocess

from struct_inventory import next_cursor, positive_integer


ENTITIES = ("Struct", "Planet", "Player", "Guild", "Substation", "Allocation")


def inventory(query):
    status = query("status")
    if not isinstance(status, dict) or not isinstance(status.get("sync_info"), dict):
        raise ValueError("invalid node status")
    height = status["sync_info"].get("latest_block_height")
    positive_integer(height, 2**63 - 1, "inventory height")
    counts = {}
    for entity in ENTITIES:
        limit = 2 if entity == "Player" else 1
        page = query("query", "structs", entity.lower() + "-all",
            "--page-limit", str(limit), "--height", height)
        following = next_cursor(page)
        if set(page) - {entity, "pagination"}:
            raise ValueError(f"unexpected {entity} response fields")
        rows = page.get(entity)
        if rows is None:
            rows = []
        if not isinstance(rows, list) or len(rows) > limit:
            raise ValueError(f"invalid {entity} inventory")
        if following or len(rows) >= limit:
            raise ValueError(f"chain is not fresh: {entity} inventory exceeds fixture allowance")
        # Only the one permitted initial player can reach this branch with a row.
        for row in rows:
            if not isinstance(row, dict):
                raise ValueError("invalid initial player")
            positive_integer(row.get("index"), 2**64 - 1, "player index")
            if row.get("id") != "1-" + row["index"]:
                raise ValueError("initial player identity disagrees with its index")
        counts[entity] = len(rows)
    return {"height": height, "counts": counts}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command", nargs=argparse.REMAINDER)
    args = parser.parse_args()
    try:
        command = args.command[1:] if args.command[:1] == ["--"] else args.command
        if not command:
            raise ValueError("a CLI command is required")
        def query(*arguments):
            response = subprocess.run([*command, *arguments], capture_output=True,
                text=True, check=True, timeout=30)
            return json.loads(response.stdout)
        print(json.dumps(inventory(query), separators=(",", ":")))
    except (ValueError, KeyError, TypeError, OSError, subprocess.SubprocessError) as error:
        parser.exit(1, f"Fresh-chain check refused: {error}\n")


if __name__ == "__main__":
    main()
