#!/usr/bin/env python3
"""Run test_chain.sh on a fresh isolated chain and retain frozen evidence.

Local nodes must already be stopped. Never stops unrelated processes. The mining
and no-mining modes each create a separate genesis and execute extended battles.
"""
import argparse
import json
import os
from pathlib import Path
import re
import shutil
import socket
import subprocess
import sys
import tempfile
import time
from urllib.request import urlopen

from run_export_restart import assert_no_nodes, export, save, stop
from upgrade_checkpoint import file_hash


def prepare(repository, binary, parent, skip_mining, *, prefix="full-lifecycle-"):
    directory = Path(tempfile.mkdtemp(prefix=prefix, dir=parent)).resolve()
    source = directory / "source"
    names = subprocess.check_output(["git", "-C", str(repository), "ls-files", "-co", "--exclude-standard", "-z"]).decode().split("\0")
    hashes = {}
    for name in sorted(set(names) - {""}):
        original = repository / name
        if not original.is_file():
            continue
        target = source / name
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(original, target)
        hashes[name] = file_hash(target)
    save(directory / "source-inputs.json", hashes)
    candidate = directory / "structsd"
    shutil.copy2(binary, candidate)
    home = directory / "chain"
    chain_id = "structs-local-" + directory.name
    def cli(*args):
        with (directory / "bootstrap.log").open("ab") as log:
            subprocess.run([str(candidate), *args, "--home", str(home)], stdout=log, stderr=log, check=True, timeout=90)
    cli("init", "full-lifecycle", "--chain-id", chain_id)
    for key in ("alice", "bob"):
        cli("keys", "add", key, "--keyring-backend", "test", "--no-backup")
        cli("genesis", "add-genesis-account", key, "4000000000000000ualpha", "--keyring-backend", "test")
    path = home / "config/genesis.json"
    genesis = json.loads(path.read_text())
    state = genesis["app_state"]
    state["staking"]["params"]["bond_denom"] = "ualpha"
    state["gov"]["params"].update(voting_period="120s", max_deposit_period="120s", expedited_voting_period="60s",
        min_deposit=[dict(denom="ualpha", amount="10000000")], expedited_min_deposit=[dict(denom="ualpha", amount="50000000")])
    state["slashing"]["params"].update(signed_blocks_window="35000", min_signed_per_window="0.05")
    save(path, genesis)
    cli("genesis", "gentx", "alice", "100000000ualpha", "--chain-id", chain_id, "--keyring-backend", "test")
    cli("genesis", "collect-gentxs")
    cli("genesis", "validate")
    def config(filename, key, value):
        path = home / "config" / filename
        contents, count = re.subn(rf"(?m)^{re.escape(key)}\s*=.*$", f"{key} = {value}", path.read_text())
        if count != 1:
            raise ValueError(f"expected one {key} in {filename}, found {count}")
        path.write_text(contents)
    config("client.toml", "chain-id", json.dumps(chain_id))
    config("client.toml", "keyring-backend", '"test"')
    config("client.toml", "node", '"tcp://127.0.0.1:26657"')
    config("app.toml", "minimum-gas-prices", '"0ualpha"')
    config("config.toml", "timeout_commit", '"100ms"')
    config("config.toml", "timeout_propose", '"100ms"')
    config("config.toml", "pprof_laddr", '""')
    wrapper = directory / "bin/structsd"
    wrapper.parent.mkdir()
    wrapper.write_text(f"#!{sys.executable}\nimport os,sys\nsys.path.insert(0,{str(source / 'tests')!r})\n"
        "from prepare_upgrade_rehearsal import fixture_arguments\n"
        f"binary={str(candidate)!r}\narguments=fixture_arguments(sys.argv[1:],{str(home)!r},{chain_id!r})\n"
        "os.execv(binary,[binary,*arguments])\n")
    wrapper.chmod(0o700)
    flags = ["--extended-battle", "--log-battle"]
    if skip_mining:
        flags.insert(0, "--skip-mining")
    save(directory / "manifest.json", {"chain_id": chain_id, "home": str(home), "binary_sha256": file_hash(candidate),
        "source_manifest_sha256": file_hash(directory / "source-inputs.json"), "flags": flags,
        "environment": {"STRUCTS_TEST_SETTLE_SECONDS": "0.2", "STRUCTS_TEST_LONG_SETTLE_SECONDS": "1.5", "STRUCTS_TEST_DURATION_MULTIPLIER": "10"},
        "configuration": {str(p.relative_to(directory)): file_hash(p) for p in [wrapper, *(home / "config").glob("*.toml"), home / "config/genesis.json"]},
        "release_approved": False})
    return directory


def run(directory):
    assert_no_nodes()
    manifest = json.loads((directory / "manifest.json").read_text())
    binary, home = directory / "structsd", directory / "chain"
    if (directory / "run-status.json").exists():
        raise ValueError("refusing to overwrite existing run evidence")
    if file_hash(binary) != manifest["binary_sha256"] or file_hash(directory / "source-inputs.json") != manifest["source_manifest_sha256"]:
        raise ValueError("binary/source manifest hash mismatch")
    for name, digest in json.loads((directory / "source-inputs.json").read_text()).items():
        if file_hash(directory / "source" / name) != digest:
            raise ValueError("source hash mismatch: " + name)
    for name, digest in manifest["configuration"].items():
        if file_hash(directory / name) != digest:
            raise ValueError("configuration hash mismatch: " + name)
    node = child = None
    status = {"phase": "starting", "runner_pid": os.getpid(), "started_at": time.time(), "release_approved": False}
    def update(**values):
        status.update(values)
        save(directory / "run-status.json", status)
    def height():
        with urlopen("http://127.0.0.1:26657/status", timeout=5) as response:
            data = json.load(response)["result"]
        if data["node_info"]["network"] != manifest["chain_id"]:
            raise ValueError("wrong local chain")
        return int(data["sync_info"]["latest_block_height"])
    # SDK startup registers the simulation query only when API or gRPC is
    # enabled, even though the CLI reaches it through Comet RPC.
    def start(label):
        nonlocal node
        assert_no_nodes()
        with socket.socket() as listener:
            listener.bind(("127.0.0.1", 9090))
        with (directory / (label + "-node.log")).open("xb") as log:
            node = subprocess.Popen([str(binary), "start", "--home", str(home), "--pruning", "nothing",
                "--rpc.laddr", "tcp://127.0.0.1:26657", "--p2p.laddr", "tcp://127.0.0.1:26656",
                "--p2p.seeds", "", "--p2p.persistent_peers", "", "--p2p.pex=false",
                "--api.enable=false", "--grpc.enable=true", "--grpc.address", "127.0.0.1:9090", "--grpc-web.enable=false"], stdout=log, stderr=log, start_new_session=True)
        update(node_pid=node.pid)
        deadline = time.monotonic() + 90
        while time.monotonic() < deadline:
            if node.poll() is not None:
                raise RuntimeError(f"node exited {node.returncode}")
            try:
                if height() > 0:
                    return
            except OSError:
                pass
            time.sleep(.3)
        raise TimeoutError("node readiness")
    try:
        start("lifecycle")
        environment = dict(os.environ, **manifest["environment"])
        environment.update(PATH=str(directory / "bin") + os.pathsep + environment["PATH"], PYTHONDONTWRITEBYTECODE="1")
        with (directory / "lifecycle.log").open("xb") as log:
            child = subprocess.Popen(["bash", str(directory / "source/tests/test_chain.sh"), *manifest["flags"]],
                cwd=directory / "source", env=environment, stdout=log, stderr=log, start_new_session=True)
            update(phase="lifecycle", script_pid=child.pid)
            print(json.dumps(status), flush=True)
            code = child.wait()
        update(script_exit_code=code, last_height=height())
        stop(node)
        exported = export(binary, home, directory / "lifecycle-export.json")
        update(export_height=int(exported["initial_height"]))
        if code:
            raise RuntimeError(f"test_chain.sh exited {code}")
        update(phase="database-continuation")
        start("continuation")
        initial = height()
        deadline = time.monotonic() + 60
        while height() < initial + 3:
            if node.poll() is not None or time.monotonic() > deadline:
                raise RuntimeError("database continuation failed")
            time.sleep(.3)
        update(continuation_height=height())
        stop(node)
        exported = export(binary, home, directory / "continuation-export.json")
        update(phase="completed", continuation_export_height=int(exported["initial_height"]), finished_at=time.time())
        print(json.dumps(status), flush=True)
    except BaseException as error:
        update(phase="failed", error=str(error), finished_at=time.time())
        raise
    finally:
        try:
            stop(child)
        finally:
            stop(node)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", type=Path)
    parser.add_argument("--output-root", type=Path)
    parser.add_argument("--skip-mining", action="store_true")
    parser.add_argument("--prepared", type=Path, help=argparse.SUPPRESS)
    args = parser.parse_args()
    if args.prepared:
        run(args.prepared.resolve(strict=True))
    else:
        if args.binary is None or args.output_root is None:
            parser.error("--binary and --output-root are required")
        directory = prepare(Path(__file__).resolve().parent.parent, args.binary.resolve(strict=True), args.output_root.resolve(strict=True), args.skip_mining)
        print(f"Full lifecycle evidence: {directory}", flush=True)
        # Execute the frozen runner and helper imports, even if the workspace is
        # edited while a long proof-of-work run is in progress.
        os.execv(sys.executable, [sys.executable, str(directory / "source/tests/run_full_lifecycle.py"), "--prepared", str(directory)])
