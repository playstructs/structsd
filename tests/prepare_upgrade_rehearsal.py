#!/usr/bin/env python3
"""Prepare an isolated v0.21 fixture, optionally with genesis; never start a node."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile


def isolated_arguments(arguments, chain_home):
    """Override both Cobra flag spellings, preserving all other argv exactly."""
    result = []
    index = 0
    while index < len(arguments):
        argument = arguments[index]
        if argument == "--":
            raise ValueError("fixture wrapper does not accept an option terminator")
        if argument in ("--home", "--keyring-dir"):
            if index + 1 >= len(arguments) or arguments[index + 1].startswith("--"):
                raise ValueError(f"missing value for {argument}")
            result.extend((argument, chain_home))
            index += 2
            continue
        if argument.startswith(("--home=", "--keyring-dir=")):
            result.append(argument.split("=", 1)[0] + "=" + chain_home)
        else:
            result.append(argument)
        index += 1
    # Also cover commands with no explicit home, such as help/status lookups.
    result.extend(("--home", chain_home))
    return result


def fixture_arguments(arguments, chain_home, chain_id):
    """Pin transaction signing to this chain, including legacy CLI paths.

    The release fixture puts global flags before the command. Do not mistake a
    flag value or `query tx` for the top-level transaction command. Reject an
    unknown leading flag rather than guessing whether it consumes a value.
    """
    index = 0
    value_flags = {"--home", "--keyring-dir", "--keyring-backend", "--gas",
        "--gas-adjustment", "--output", "--chain-id", "--node", "--fees",
        "--gas-prices", "--from", "--broadcast-mode"}
    while index < len(arguments) and arguments[index].startswith("-"):
        flag = arguments[index]
        if "=" in flag or flag in ("--yes", "-y"):
            index += 1
        elif flag in value_flags:
            if index + 1 >= len(arguments):
                raise ValueError(f"missing value for {flag}")
            index += 2
        else:
            raise ValueError(f"unsupported leading fixture flag: {flag}")
    result = isolated_arguments(arguments, chain_home)
    if index < len(arguments):
        command = arguments[index]
        if command == "tx":
            result.extend(("--chain-id", chain_id))
        # Home isolation does not guarantee endpoint isolation: environment or
        # CLI initialization can override client.toml. Pin every network path.
        if command in ("tx", "query", "status"):
            result.extend(("--node", "tcp://127.0.0.1:26657"))
    return result


def prepare(repository, output_parent, binary):
    repository, binary = repository.resolve(), binary.resolve(strict=True)
    if not binary.is_file() or not os.access(binary, os.X_OK):
        raise ValueError("v0.21.0 binary must be executable")
    commit = subprocess.check_output(
        ["git", "-C", str(repository), "rev-parse", "--verify", "v0.21.0^{commit}"], text=True
    ).strip()
    fixture = subprocess.check_output(
        ["git", "-C", str(repository), "show", f"{commit}:tests/test_chain.sh"]
    )
    directory = Path(tempfile.mkdtemp(prefix="structs-upgrade-", dir=output_parent)).resolve()
    (directory / "bin").mkdir()
    (directory / "test_chain_v0.21.0.sh").write_bytes(fixture)
    # Retain the routing implementation in the artifact, not an import into the
    # mutable workspace. Python literals preserve paths with spaces/metacharacters.
    (directory / "prepare_upgrade_rehearsal.py").write_bytes(Path(__file__).read_bytes())
    wrapper = (
        f"#!{sys.executable}\n"
        "import os, sys\n"
        "from pathlib import Path\n"
        "sys.path.insert(0, str(Path(__file__).resolve().parent.parent))\n"
        "from prepare_upgrade_rehearsal import fixture_arguments\n"
        f"binary = {str(binary)!r}\n"
        f"home = {str(directory / 'chain')!r}\n"
        "try:\n"
        "    arguments = fixture_arguments(sys.argv[1:], home, 'structs-upgrade-v022')\n"
        "except ValueError as error:\n"
        "    sys.exit(str(error))\n"
        "os.execv(binary, [binary, *arguments])\n"
    )
    (directory / "bin" / "structsd").write_text(wrapper)
    (directory / "bin" / "structsd").chmod(0o700)
    manifest = {
        "baseline_ref": "v0.21.0", "baseline_commit": commit,
        "fixture_sha256": hashlib.sha256(fixture).hexdigest(),
        "routing_helper_sha256": hashlib.sha256((directory / "prepare_upgrade_rehearsal.py").read_bytes()).hexdigest(),
        "wrapper_sha256": hashlib.sha256((directory / "bin/structsd").read_bytes()).hexdigest(),
        "binary": str(binary), "binary_sha256": hashlib.sha256(binary.read_bytes()).hexdigest(),
        "chain_home": str(directory / "chain"),
        "status": "prepared only; no chain initialized or started",
        "remaining": "runner bootstrap, fixture coverage, governance upgrade, assertions and live approval",
    }
    (directory / "manifest.json").write_text(json.dumps(manifest, indent=2) + "\n")
    return directory


def bootstrap(directory, chain_id="structs-upgrade-v022"):
    """Create and validate fresh offline genesis. Refuse an existing chain home."""
    directory = directory.resolve(strict=True)
    manifest_path = directory / "manifest.json"
    manifest = json.loads(manifest_path.read_text())
    binary = Path(manifest["binary"])
    home = directory / "chain"
    if manifest["chain_home"] != str(home):
        raise ValueError("manifest chain home does not belong to this rehearsal")
    if hashlib.sha256(binary.read_bytes()).hexdigest() != manifest["binary_sha256"]:
        raise ValueError("old binary changed since rehearsal preparation")
    # mkdir without exist_ok reserves the home and refuses accidental reruns,
    # including partially initialized homes. Failed artifacts remain inspectable.
    home.mkdir(mode=0o700)
    manifest["status"] = "offline bootstrap started; genesis not yet validated; no node started"
    manifest_path.write_text(json.dumps(manifest, indent=2) + "\n")
    log_path = directory / "bootstrap.log"

    def run(*arguments):
        with log_path.open("ab") as log:
            result = subprocess.run(
                [str(binary), *arguments, "--home", str(home)],
                stdout=log, stderr=log, timeout=60,
            )
        if result.returncode:
            raise ValueError(f"offline {' '.join(arguments[:2])} failed; inspect {log_path}")

    def config(path, key, value):
        contents, count = re.subn(
            rf"(?m)^{re.escape(key)}\s*=.*$", f"{key} = {value}", path.read_text()
        )
        if count != 1:
            raise ValueError(f"expected one {key} setting in {path}; found {count}")
        path.write_text(contents)

    run("init", "upgrade-rehearsal", "--chain-id", chain_id)
    for name in ("alice", "bob"):
        run("keys", "add", name, "--keyring-backend", "test", "--no-backup")
        run("genesis", "add-genesis-account", name, "4000000000000000ualpha", "--keyring-backend", "test")
    genesis_path = home / "config/genesis.json"
    genesis = json.loads(genesis_path.read_text())
    state = genesis["app_state"]
    state["staking"]["params"]["bond_denom"] = "ualpha"
    state["gov"]["params"].update({
        "voting_period": "120s", "max_deposit_period": "120s",
        "expedited_voting_period": "60s",
        "min_deposit": [{"denom": "ualpha", "amount": "10000000"}],
        "expedited_min_deposit": [{"denom": "ualpha", "amount": "50000000"}],
    })
    if "crisis" in state:
        state["crisis"]["constant_fee"]["denom"] = "ualpha"
    state["slashing"]["params"]["signed_blocks_window"] = "35000"
    state["slashing"]["params"]["min_signed_per_window"] = "0.05"
    # Match v0.21.0 config.yml's dev-chain settings. Production defaults make
    # the unchanged release fixture wait 432,000 blocks before its first guild.
    state["structs"]["params"].update({
        "guildCharterDifficultyRange": "2", "guildCharterReactorAge": "5",
    })
    genesis_path.write_text(json.dumps(genesis, indent=2) + "\n")
    run("genesis", "gentx", "alice", "100000000ualpha", "--chain-id", chain_id, "--keyring-backend", "test")
    run("genesis", "collect-gentxs")
    run("genesis", "validate")
    config(home / "config/client.toml", "chain-id", json.dumps(chain_id))
    config(home / "config/client.toml", "keyring-backend", '"test"')
    config(home / "config/app.toml", "minimum-gas-prices", '"0ualpha"')
    config(home / "config/config.toml", "timeout_commit", '"1s"')
    config(home / "config/config.toml", "timeout_propose", '"1s"')
    manifest.update({
        "chain_id": chain_id,
        "genesis_sha256": hashlib.sha256(genesis_path.read_bytes()).hexdigest(),
        "status": "offline genesis validated; no node started or transactions broadcast",
        "remaining": "fixture population and coverage, governance upgrade, assertions and live approval",
    })
    manifest_path.write_text(json.dumps(manifest, indent=2) + "\n")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output-parent", type=Path, required=True, help="existing directory for a new unique rehearsal directory")
    parser.add_argument("--old-binary", type=Path, help="defaults to build/structsd-v0.21.0")
    parser.add_argument("--bootstrap", action="store_true", help="also create test keys and validate offline genesis; does not start a node")
    args = parser.parse_args()
    repository = Path(__file__).resolve().parent.parent
    try:
        directory = prepare(repository, args.output_parent, args.old_binary or repository / "build/structsd-v0.21.0")
        if args.bootstrap:
            bootstrap(directory)
    except (OSError, ValueError, subprocess.SubprocessError) as error:
        parser.exit(1, f"Preparation failed: {error}\n")
    print(directory)


if __name__ == "__main__":
    main()
