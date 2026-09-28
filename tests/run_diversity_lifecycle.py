#!/usr/bin/env python3
"""Continue a copied local upgrade fixture with a nonzero diversity lifecycle.

Requires the alice/bob test keys created by prepare_upgrade_rehearsal.py. Never
connects to a remote endpoint or starts the second validator. Alice retains more
than two-thirds of local consensus power. Source state is copied while stopped.
"""
import argparse
import hashlib
import json
from pathlib import Path
import shutil
import subprocess
import tempfile

from live_reactor_diversity import verify_live
from run_export_restart import assert_no_nodes
from run_upgrade_rehearsal import Rehearsal
from upgrade_checkpoint import file_hash


def run(source, binary, parent):
    assert_no_nodes()
    original = json.loads((source / "manifest.json").read_text())
    if original.get("status") != "rehearsal checks passed; release approval remains outstanding":
        raise ValueError("requires a successfully verified local upgrade fixture")
    if Path(original["chain_home"]).resolve() != (source / "chain").resolve():
        raise ValueError("source fixture home mismatch")
    directory = Path(tempfile.mkdtemp(prefix="native-diversity-", dir=parent)).resolve()
    shutil.copytree(source / "chain", directory / "chain")
    candidate = directory / "structsd"
    shutil.copy2(binary, candidate)
    manifest = {"status": "prepared; not verified", "chain_home": str(directory / "chain"),
                "chain_id": original["chain_id"], "binary": str(candidate), "new_binary": str(candidate),
                "binary_sha256": file_hash(candidate), "new_binary_sha256": file_hash(candidate),
                "source": str(source), "source_manifest_sha256": file_hash(source / "manifest.json"),
                "script_sha256": {p.name: file_hash(p) for p in sorted(Path(__file__).parent.glob("*.py"))}}
    (directory / "manifest.json").write_text(json.dumps(manifest, indent=2) + "\n")
    print(f"Native diversity evidence: {directory}", flush=True)
    runner = Rehearsal(directory, candidate)
    try:
        node = runner.start([str(candidate), "start", "--home", str(runner.home),
                             "--p2p.seeds", "", "--p2p.persistent_peers", "", "--p2p.pex=false"], "node.log")
        initial = int(json.loads((source / "after.json").read_text())["initial_height"])
        runner.wait_height(candidate, node, initial + 1)
        # Generate a distinct consensus public key using the chain CLI. Its
        # private key stays in this local fixture home and is never output.
        secondary = directory / "second-validator"
        with (directory / "second-validator-init.log").open("xb") as output:
            subprocess.run([str(candidate), "init", "diversity-secondary", "--home", str(secondary)],
                           stdout=output, stderr=output, check=True, timeout=60)
        public = json.loads(subprocess.check_output([str(candidate), "comet", "show-validator", "--home", str(secondary)], text=True))
        validator_file = directory / "validator.json"
        validator_file.write_text(json.dumps({"pubkey": public, "amount": "1000000ualpha", "moniker": "diversity-secondary",
            "commission-rate": "0", "commission-max-rate": "1", "commission-max-change-rate": "1", "min-self-delegation": "1"}))
        def send(label, *args, signer="alice"):
            result = runner.tx(*args, signer=signer)
            runner.save(label + "-transaction.json", result)
            runner.wait_height(candidate, node, int(result["height"]) + 1)
            return result
        send("create-validator", "staking", "create-validator", str(validator_file), signer="bob")
        target = runner.cli(candidate, "keys", "show", "bob", "--bech", "val", "--keyring-backend", "test")["address"]
        primary = runner.cli(candidate, "keys", "show", "alice", "--keyring-backend", "test")["address"]
        owner = runner.cli(candidate, "query", "structs", "address", primary)["playerId"]
        queries = []
        def query(*args):
            result = runner.cli(candidate, *args)
            queries.append({"arguments": args, "response": result})
            return result
        send("infuse", "structs", "reactor-infuse", target, "1000000ualpha")
        infused = verify_live(query, owner)
        if int(infused["summary"]["appliedBonus"]) <= 0:
            raise ValueError("fixture did not establish nonzero diversity")
        result = send("defuse", "structs", "reactor-defuse", target, "500000ualpha")
        defused = verify_live(query, owner)
        send("cancel", "structs", "reactor-cancel-defusion", target, "500000ualpha", result["height"])
        cancelled = verify_live(query, owner)
        if any(infused[key] != cancelled[key] for key in ("summary", "capacity", "infusions", "owner")):
            raise ValueError("cancellation failed to restore the same portfolio and capacity")
        runner.stop(node)
        # Reopen the same native database, then verify persisted nonzero bonus.
        height = int(cancelled["height"])
        node = runner.start([str(candidate), "start", "--home", str(runner.home),
                             "--p2p.seeds", "", "--p2p.persistent_peers", "", "--p2p.pex=false"], "reopened-node.log")
        runner.wait_height(candidate, node, height + 3)
        reopened = verify_live(query, owner)
        if any(cancelled[key] != reopened[key] for key in ("summary", "capacity", "infusions", "owner")):
            raise ValueError("native database reopening changed diversity")
        runner.save("queries.json", queries)
        runner.save("verification.json", {"infused": infused, "defused": defused, "cancelled": cancelled,
            "reopened": reopened, "differences": [], "release_approved": False})
        runner.manifest["status"] = "nonzero diversity lifecycle and native reopen passed"
        runner.save("manifest.json", runner.manifest)
    except BaseException as error:
        runner.manifest.update(status="failed", error=str(error))
        runner.save("manifest.json", runner.manifest)
        raise
    finally:
        runner.cleanup()
    return directory


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("source", type=Path)
    parser.add_argument("--binary", type=Path, required=True)
    parser.add_argument("--output-root", type=Path, required=True)
    args = parser.parse_args()
    run(args.source.resolve(strict=True), args.binary.resolve(strict=True), args.output_root.resolve(strict=True))
