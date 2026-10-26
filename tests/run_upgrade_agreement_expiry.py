#!/usr/bin/env python3
"""Verify legacy expiry after the combined upgrade and vesting continuation.

Reopens only the stopped original private node to retrieve historical public
block results, stops it, then exports retained heights. Never sends transactions.
The original upgrade proof remains separate from the later quiet-window proof.
"""
import argparse
import json
from pathlib import Path
from urllib.request import urlopen

from generator_lifecycle import amount
from run_export_restart import assert_no_nodes, export
from run_upgrade_rehearsal import Rehearsal
from upgrade_checkpoint import file_hash
from verify_upgrade_agreement_expiry import verify_legacy_expiry
from verify_upgrade_agreements import rows, same
from verify_upgrade_bank import balances


PASSED = "rehearsal checks passed; release approval remains outstanding"


def read(path):
    return json.loads(path.read_text())


def expiry_window(seed, initial, final):
    if seed.get("status") != "three live legacy agreements confirmed; upgrade verification pending":
        raise ValueError("requires confirmed live legacy contracts")
    rows = seed.get("agreements", [])
    if len(rows) != 3 or len({row["id"] for row in rows}) != 3:
        raise ValueError("requires three distinct seeded contracts")
    if initial.get("label") != "initial-sent" or final.get("label") != "timed-unlocks-complete":
        raise ValueError("requires the continuation's exact observation boundaries")
    # Exports are AFTER the named block, whereas initial_height is the next one.
    # The first two exports bracket one quiet block to establish unchanged terms.
    before = amount(initial["height"]) + 1
    migrated = before + 1
    ends = [amount(row["endBlock"]) for row in rows]
    after = max(ends)
    if min(ends) <= migrated + 1 or amount(final["height"]) <= after:
        raise ValueError("contract expiries do not fit between continuation transfers")
    return {"before": before, "migrated": migrated, "after": after}


def observation(directory, label):
    matches = list(directory.glob("[0-9][0-9]-" + label + "-queries.json"))
    if len(matches) != 1:
        raise ValueError("requires exactly one observation: " + label)
    return matches[0]


def unchanged_service_baseline(original, current, seed, providers):
    old, new = original["app_state"]["structs"], current["app_state"]["structs"]
    agreements = rows(old["agreementList"], "id")
    if set(agreements) != {row["id"] for row in seed["agreements"]}:
        raise ValueError("upgrade contracts differ from the seeded contracts")
    for name in ("agreementList", "providerList"):
        if not same(rows(old[name], "id"), rows(new[name], "id")):
            raise ValueError("service terms changed before the quiet baseline")
    for prefix in ("3-", "14-"):
        keys = {prefix + key for key in providers}
        values = [{key: row for key, row in rows(state["gridList"], "attributeId").items() if key in keys}
                  for state in (old, new)]
        if not same(*values):
            raise ValueError("provider load/checkpoint changed before the quiet baseline")
    addresses = {address for pair in providers.values() for address in pair.values()}
    funds = [{key: value for key, value in balances(document).items() if key[0] in addresses}
             for document in (original, current)]
    if funds[0] != funds[1]:
        raise ValueError("provider funds moved before the quiet baseline")


def run(directory, binary, continuation_name="vesting-continuation"):
    runner = Rehearsal(directory, binary)
    continuation = runner.directory / continuation_name
    if continuation.parent != runner.directory or not continuation.name.startswith("vesting-continuation"):
        raise ValueError("continuation must be an immediate evidence directory")
    output = runner.directory / "agreement-expiry"
    if output.exists() or runner.home.is_symlink():
        raise ValueError("requires original private home and unused output directory")
    if runner.manifest.get("chain_home") != str(runner.home) or runner.manifest.get("status") != PASSED:
        raise ValueError("original upgrade must pass")
    if file_hash(runner.new) != runner.manifest.get("new_binary_sha256"):
        raise ValueError("candidate binary changed")
    if read(continuation / "result.json").get("status") != "native continuation checks passed":
        raise ValueError("native continuation must finish before historical exports")
    original_proof = read(runner.directory / "agreement-verification.json")
    proof_names = ("before.json", "after.json", "upgrade-block-results.json", "vault-map.json", "provider-accounts.json")
    if original_proof.get("differences") != [] or original_proof.get("input_sha256") != {
            name: file_hash(runner.directory / name) for name in proof_names}:
        raise ValueError("original agreement migration must pass")
    initial = observation(continuation, "initial-sent")
    final = observation(continuation, "timed-unlocks-complete")
    seed_path = runner.directory / "agreement-seed.json"
    heights = expiry_window(read(seed_path), read(initial), read(final))
    providers = read(runner.directory / "provider-accounts.json")
    inputs = [runner.directory / name for name in (
        "manifest.json", "agreement-verification.json", "before.json", "after.json",
        "upgrade-block-results.json", "provider-accounts.json")]
    inputs += [continuation / "result.json", initial, final, seed_path, runner.new]
    inputs += sorted(Path(__file__).parent.glob("*.py"))
    assert_no_nodes()
    output.mkdir(mode=0o700)

    def save(name, value):
        with (output / name).open("x") as stream:
            json.dump(value, stream, indent=2)
            stream.write("\n")

    save("preflight.json", {"heights": heights, "input_sha256": {str(p): file_hash(p) for p in inputs}})
    try:
        node = runner.start([str(runner.new), "start", "--home", str(runner.home),
                             "--p2p.laddr", "tcp://127.0.0.1:26656"], "agreement-expiry/node.log")
        runner.wait_height(runner.new, node, heights["after"], timeout=60)
        status = runner.cli(runner.new, "status")
        if status["node_info"]["network"] != runner.manifest["chain_id"]:
            raise ValueError("RPC belongs to a different chain")
        blocks = []
        for height in range(heights["before"] + 1, heights["after"] + 1):
            with urlopen("http://127.0.0.1:26657/block_results?height=" + str(height), timeout=20) as response:
                envelope = json.load(response)
            # Retain the full response, including failures or unexpected txs.
            save("block-" + str(height) + ".json", envelope)
            block = envelope["result"]
            if amount(block["height"]) != height or block.get("txs_results"):
                raise ValueError("expiry window is incomplete or contains transactions")
            blocks.append(block)
        runner.stop(node)
        assert_no_nodes()
        exports = {name: export(runner.new, runner.home, output / (name + ".json"), height=height)
                   for name, height in heights.items()}
        for name, height in heights.items():
            if int(exports[name]["initial_height"]) != height + 1:
                raise ValueError("historical export returned a different height")
        unchanged_service_baseline(read(runner.directory / "after.json"), exports["before"], read(seed_path), providers)
        save("blocks.json", blocks)
        # The first block is the consecutive quiet baseline; the rest include
        # every expiry. Do not erase transaction results or unrelated events.
        result = verify_legacy_expiry(exports["before"], exports["migrated"], exports["after"],
                                      blocks[:1], blocks[1:], providers)
        result["input_sha256"] = {p.name: file_hash(p) for p in sorted(output.glob("*.json"))}
        result["release_approved"] = False
        save("verification.json", result)
        if result["differences"]:
            raise ValueError("legacy expiry verification failed")
        print(json.dumps(result, indent=2), flush=True)
        return result
    except BaseException as error:
        save("failure.json", {"error": str(error), "type": type(error).__name__, "release_approved": False})
        raise
    finally:
        runner.cleanup()


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("directory", type=Path)
    parser.add_argument("--new-binary", type=Path, required=True)
    parser.add_argument("--continuation-name", default="vesting-continuation")
    args = parser.parse_args()
    run(args.directory, args.new_binary, args.continuation_name)
