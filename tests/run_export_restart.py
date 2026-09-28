#!/usr/bin/env python3
"""Rehearse both genesis export modes from a stopped local chain, sequentially.

Creates isolated homes and retains exports, block events, logs and source hashes.
Does not copy source databases, start the source node, or send transactions.
Requires the same local-chain authorization as test_chain.sh.
"""
import argparse
from datetime import datetime, timezone
import json
import os
from pathlib import Path
import re
import shutil
import signal
import socket
import subprocess
import tempfile
import time
from urllib.request import urlopen

from upgrade_checkpoint import file_hash
from verify_export_restart import verify_restart
from verify_reactor_diversity import verify_diversity


def assert_no_nodes():
    # macOS truncates the separate comm column even when args is complete.
    # Match the executable in args so custom-port nodes are still detected.
    processes = subprocess.run(["ps", "-axww", "-o", "pid=,args="], text=True, capture_output=True, check=True).stdout
    for row in processes.splitlines():
        parts = row.split(None, 1)
        if len(parts) == 2 and re.search(r"(?:^|/)structsd[^\s/]*\s+start(?:\s|$)", parts[1]):
            raise ValueError(f"native Structs node PID {parts[0]} is still active; serial rehearsal required")
    for port in (26656, 26657, 27656, 27657, 29656, 29657):
        with socket.socket() as connection:
            connection.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
            try:
                connection.bind(("127.0.0.1", port))
            except OSError as error:
                raise ValueError(f"local rehearsal port {port} is occupied; no process was stopped") from error


def save(path, value):
    path.write_text(json.dumps(value, indent=2) + "\n")


def stop(node):
    if node is not None and node.poll() is None:
        os.killpg(node.pid, signal.SIGTERM)
        try:
            node.wait(timeout=30)
        except subprocess.TimeoutExpired:
            os.killpg(node.pid, signal.SIGKILL)
            node.wait(timeout=10)
            raise RuntimeError("restart node needed SIGKILL; rehearsal incomplete")


def export(binary, home, target, *, zero=False, height=None):
    command = [str(binary), "export", "--home", str(home), "--output-document", str(target)]
    if zero:
        command.append("--for-zero-height")
    if height is not None:
        command.extend(["--height", str(height)])
    # Explicit-height loads can log to stdout; keep that stream out of the JSON.
    # Claim the output path exclusively before allowing the CLI to replace it.
    target.touch(exist_ok=False)
    with target.with_suffix(".stdout.log").open("xb") as output, target.with_suffix(".log").open("xb") as errors:
        subprocess.run(command, stdout=output, stderr=errors, check=True, timeout=180)
    return json.loads(target.read_text())


def rpc(path, chain_id=None):
    with urlopen("http://127.0.0.1:29657/" + path, timeout=20) as response:
        result = json.load(response)["result"]
    if chain_id is not None and result["node_info"]["network"] != chain_id:
        raise ValueError("restart RPC is serving a different chain")
    return result


def run_mode(binary, source, directory, mode, blocks):
    assert_no_nodes()
    directory.mkdir(mode=0o700)
    original = export(binary, source, directory / "source-export.json", zero=mode == "zero")
    home = directory / "chain"
    chain_id = original["chain_id"] + "-restart-" + mode
    with (directory / "init.log").open("xb") as log:
        subprocess.run([str(binary), "init", "export-restart", "--chain-id", chain_id, "--home", str(home)],
                       stdout=log, stderr=log, check=True, timeout=60)
    key = source / "config/priv_validator_key.json"
    validator_key = json.loads(key.read_text())["pub_key"]
    if not any(row["pub_key"] == validator_key for row in original["consensus"]["validators"]):
        raise ValueError("source local validator key is absent from the exported validator set")
    shutil.copy2(key, home / "config/priv_validator_key.json")
    original["chain_id"] = chain_id
    original["genesis_time"] = datetime.now(timezone.utc).isoformat().replace("+00:00", "Z")
    save(home / "config/genesis.json", original)
    save(directory / "before.json", original)
    config = home / "config/config.toml"
    text = config.read_text()
    text, changes = re.subn(r'^timeout_commit = ".*"$', 'timeout_commit = "250ms"', text, flags=re.M)
    if changes != 1:
        raise ValueError("could not configure the isolated block interval")
    text = re.sub(r'^pprof_laddr = ".*"$', 'pprof_laddr = ""', text, flags=re.M)
    config.write_text(text)
    with (directory / "validate.log").open("xb") as log:
        subprocess.run([str(binary), "genesis", "validate-genesis", "--home", str(home)], stdout=log, stderr=log, check=True, timeout=60)
    first = max(1, int(original["initial_height"]))
    target = first + blocks - 1
    command = [str(binary), "start", "--home", str(home), "--pruning", "nothing",
               "--minimum-gas-prices", "0ualpha", "--rpc.laddr", "tcp://127.0.0.1:29657",
               "--p2p.laddr", "tcp://127.0.0.1:29656", "--p2p.seeds", "", "--p2p.persistent_peers", "",
               "--p2p.pex=false", "--api.enable=false", "--grpc.enable=false", "--grpc-web.enable=false"]
    node = None
    try:
        with (directory / "node.log").open("xb") as log:
            node = subprocess.Popen(command, stdout=log, stderr=log, start_new_session=True)
        save(directory / "run-status.json", {"phase": "running", "node_pid": node.pid, "chain_id": chain_id, "target_height": target})
        deadline = time.monotonic() + 180
        while True:
            if node.poll() is not None:
                raise RuntimeError(f"restart node exited {node.returncode}")
            try:
                status = rpc("status", chain_id)
                if int(status["sync_info"]["latest_block_height"]) >= target:
                    break
            except OSError:
                pass
            if time.monotonic() >= deadline:
                raise TimeoutError("restart did not commit the required blocks")
            time.sleep(0.25)
        events = [rpc(f"block_results?height={height}") for height in range(first, target + 1)]
        save(directory / "blocks.json", events)
        save(directory / "last-observed-status.json", status)
    finally:
        stop(node)
    # Export the exact observed boundary even if additional empty blocks landed
    # while fetching RPC events. No implicit latest-height comparison is used.
    after = export(binary, home, directory / "after.json", height=target)
    result = verify_restart(original, after, events)
    result["diversity"] = verify_diversity(original, after, events, restart=True)
    result["differences"].extend(result["diversity"]["differences"])
    result["mode"] = mode
    result["input_sha256"] = {name: file_hash(directory / name) for name in ("source-export.json", "before.json", "after.json", "blocks.json")}
    save(directory / "verification.json", result)
    if result["differences"]:
        raise ValueError(f"{mode} restart state/event comparison failed; inspect verification.json")
    save(directory / "run-status.json", {"phase": "completed", "chain_id": chain_id, "verified_height": target, "release_approved": False})
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("source_home", type=Path)
    parser.add_argument("--binary", type=Path, required=True)
    parser.add_argument("--output-root", type=Path, required=True)
    parser.add_argument("--blocks", type=int, default=3)
    parser.add_argument("--preflight-only", action="store_true")
    args = parser.parse_args()
    source, binary, output = (path.resolve(strict=True) for path in (args.source_home, args.binary, args.output_root))
    if not source.is_dir() or not binary.is_file() or not os.access(binary, os.X_OK) or not output.is_dir() or args.blocks < 3:
        parser.error("requires a local home, executable binary, existing output directory and at least three blocks")
    for name in ("config/genesis.json", "config/priv_validator_key.json"):
        if not (source / name).is_file() or (source / name).is_symlink():
            parser.error("source requires ordinary genesis and validator-key files")
    assert_no_nodes()
    if args.preflight_only:
        print("Offline inputs and serial-run guard passed; no node started or source exported.")
        return
    directory = Path(tempfile.mkdtemp(prefix="native-export-restart-", dir=output))
    candidate = directory / "structsd"
    shutil.copy2(binary, candidate)
    manifest = {"source_home": str(source), "binary_sha256": file_hash(candidate), "blocks_per_mode": args.blocks,
                "script_sha256": {name: file_hash(Path(__file__).with_name(name)) for name in (
                    "run_export_restart.py", "verify_export_restart.py", "verify_reactor_diversity.py",
                    "verify_upgrade_staking.py", "verify_upgrade_bank.py", "upgrade_checkpoint.py", "upgrade_inventory.py")},
                "phase": "running", "release_approved": False}
    save(directory / "manifest.json", manifest)
    print(f"Native export/restart evidence: {directory}", flush=True)
    try:
        results = [run_mode(candidate, source, directory / mode, mode, args.blocks) for mode in ("preserved", "zero")]
        manifest.update(phase="completed", results=results)
    except BaseException as error:
        manifest.update(phase="failed", error=str(error))
        raise
    finally:
        save(directory / "manifest.json", manifest)


def terminated(*_):
    raise SystemExit("terminated")


if __name__ == "__main__":
    signal.signal(signal.SIGTERM, terminated)
    main()
