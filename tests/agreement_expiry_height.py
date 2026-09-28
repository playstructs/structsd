"""Validate an agreement query and derive a safe post-expiry block wait target."""
import argparse
import json
import re
import sys


def expiry_height(query, agreement_id):
    agreement = query["Agreement"]
    if not isinstance(agreement, dict) or agreement.get("id") != agreement_id:
        raise ValueError("query must identify the requested agreement")
    end = agreement.get("endBlock")
    if not isinstance(end, str) or re.fullmatch(r"[1-9][0-9]*", end) is None:
        raise ValueError("endBlock must be a positive canonical decimal string")
    height = int(end)
    if height > 2**63 - 3:
        raise ValueError("post-expiry target exceeds signed block-height range")
    return {"endBlock": end, "waitHeight": str(height + 2)}


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("agreement_id")
    args = parser.parse_args()
    print(json.dumps(expiry_height(json.load(sys.stdin), args.agreement_id)))
