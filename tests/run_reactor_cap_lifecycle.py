#!/usr/bin/env python3
"""Exercise active stake caps, failed guild migration and downtime on a local node.

Uses a fresh, frozen fixture. Only this fixture lowers cap activation to two
validators and the downtime window to 1,000 blocks. Two additional validators
remain offline; the local signer retains more than two-thirds of consensus power.
"""
import argparse
import json
import os
from pathlib import Path
import socket
import subprocess
import sys
import time

from live_reactor_diversity import verify_live
from run_export_restart import assert_no_nodes, export, save
from run_full_lifecycle import prepare
from run_upgrade_rehearsal import Rehearsal
from upgrade_checkpoint import file_hash


def prepare_cap(binary, parent):
    directory = prepare(Path(__file__).resolve().parent.parent, binary, parent, True, prefix="native-cap-")
    genesis_path = directory / "chain/config/genesis.json"
    genesis = json.loads(genesis_path.read_text())
    genesis["app_state"]["structs"]["params"]["reactorStakeCapMinValidators"] = "2"
    genesis["app_state"]["slashing"]["params"]["signed_blocks_window"] = "1000"
    save(genesis_path, genesis)
    manifest = json.loads((directory / "manifest.json").read_text())
    manifest.update(kind="native cap, migration rollback and downtime", binary=str(directory / "structsd"),
        chain_home=str(directory / "chain"), status="prepared", fixture_overrides={"reactorStakeCapMinValidators": 2, "signed_blocks_window": 1000})
    manifest.pop("flags")
    manifest["configuration"]["chain/config/genesis.json"] = file_hash(genesis_path)
    save(directory / "manifest.json", manifest)
    with (directory / "cap-genesis-validation.log").open("xb") as log:
        subprocess.run([str(directory / "structsd"), "genesis", "validate", "--home", str(directory / "chain")],
                       stdout=log, stderr=log, check=True, timeout=60)
    return directory


def run(directory):
    assert_no_nodes()
    with socket.socket() as listener:
        listener.bind(("127.0.0.1", 9090))
    manifest = json.loads((directory / "manifest.json").read_text())
    binary = directory / "structsd"
    if manifest.get("status") != "prepared":
        raise ValueError("refusing to overwrite an attempted run")
    if file_hash(binary) != manifest["binary_sha256"] or file_hash(directory / "source-inputs.json") != manifest["source_manifest_sha256"]:
        raise ValueError("frozen binary or source manifest changed")
    for name, digest in json.loads((directory / "source-inputs.json").read_text()).items():
        if file_hash(directory / "source" / name) != digest:
            raise ValueError("frozen source changed: " + name)
    for name, digest in manifest["configuration"].items():
        if file_hash(directory / name) != digest:
            raise ValueError("frozen configuration changed: " + name)
    runner = Rehearsal(directory, binary)
    queries, receipts, evidence = [], [], {}
    def phase(value, **extra):
        runner.manifest.update(status=value, runner_pid=os.getpid(), **extra)
        runner.save("manifest.json", runner.manifest)
        print(value, flush=True)
    def query(*args):
        result = runner.cli(binary, *args)
        queries.append({"arguments": args, "response": result})
        return result
    def send(label, *args, signer="alice", expected_code=0):
        # Fixed gas forces rejection checks through signed DeliverTx rather
        # than treating a simulation error as a committed transaction result.
        response = runner.cli(binary, "tx", *args, "--from", signer, "--keyring-backend", "test",
                              "--gas", "20000000", "--fees", "1ualpha", "--yes")
        if int(response.get("code", 0)) != 0 or not response.get("txhash"):
            raise ValueError(f"{label}: broadcast rejected: {response}")
        deadline = time.monotonic() + 30
        while time.monotonic() < deadline:
            try:
                delivered = runner.cli(binary, "query", "tx", response["txhash"])
            except RuntimeError:
                time.sleep(.3)
                continue
            receipts.append({"label": label, "receipt": delivered})
            runner.save("transactions.json", receipts)
            if delivered.get("txhash") != response["txhash"] or int(delivered.get("height", 0)) <= 0:
                raise ValueError(label + ": transaction identity/height mismatch")
            if int(delivered.get("code", 0)) != expected_code:
                raise ValueError(f"{label}: unexpected result: {delivered}")
            if expected_code:
                if delivered.get("codespace") != "structs" or "reactor stake cap exceeded" not in delivered.get("raw_log", ""):
                    raise ValueError(label + ": wrong rejection reason")
                if any(event["type"].startswith("structs.structs.Event") or event["type"] in ("delegate", "redelegate", "unbond", "cancel_unbonding_delegation") for event in delivered.get("events", [])):
                    raise ValueError(label + ": rejected transaction published success events")
            runner.wait_height(binary, node, int(delivered["height"]) + 1)
            return delivered
        raise TimeoutError(label + ": no committed receipt")
    def start(log):
        node = runner.start([str(binary), "start", "--home", str(runner.home), "--pruning", "nothing",
            "--p2p.seeds", "", "--p2p.persistent_peers", "", "--p2p.pex=false",
            "--p2p.laddr", "tcp://127.0.0.1:26656", "--rpc.laddr", "tcp://127.0.0.1:26657",
            "--api.enable=false", "--grpc.enable=true", "--grpc.address", "127.0.0.1:9090", "--grpc-web.enable=false"], log)
        phase("node started", node_pid=node.pid)
        return node
    try:
        node = start("node.log")
        runner.wait_height(binary, node, 3)
        alice = query("keys", "show", "alice", "--keyring-backend", "test")["address"]
        bob = query("keys", "show", "bob", "--keyring-backend", "test")["address"]
        primary = query("keys", "show", "alice", "--bech", "val", "--keyring-backend", "test")["address"]
        secondary = query("keys", "show", "bob", "--bech", "val", "--keyring-backend", "test")["address"]
        owner = query("query", "structs", "address", alice)["playerId"]
        vault = query("query", "structs", "vault", owner)["address"]
        primary_id = next(row["id"] for row in query("query", "structs", "reactor-all")["Reactor"] if row["validator"] == primary)
        initial_cap = query("query", "structs", "reactor-stake-share", primary_id)
        if initial_cap.get("capActive", False):
            raise ValueError("single-validator genesis must not activate this fixture's cap")
        send("fund-vault", "bank", "send", bob, vault, "20000000ualpha", signer="bob")
        secondary_home = directory / "secondary-validator"
        with (directory / "secondary-init.log").open("xb") as log:
            subprocess.run([str(binary), "init", "offline-secondary", "--home", str(secondary_home)], stdout=log, stderr=log, check=True, timeout=60)
        pubkey = json.loads(subprocess.check_output([str(binary), "comet", "show-validator", "--home", str(secondary_home)], text=True))
        validator_file = directory / "validator.json"
        save(validator_file, {"pubkey": pubkey, "amount": "10000000ualpha", "moniker": "offline-secondary", "commission-rate": "0", "commission-max-rate": "1", "commission-max-change-rate": "1", "min-self-delegation": "1"})
        send("second-validator", "staking", "create-validator", str(validator_file), signer="bob")
        secondary_id = next(row["id"] for row in query("query", "structs", "reactor-all")["Reactor"] if row["validator"] == secondary)
        cap = query("query", "structs", "reactor-stake-share", primary_id)
        if cap.get("capActive") is not True or cap["externalDepositHeadroom"] != "0" or cap["bondedRedelegationHeadroom"] != "0":
            raise ValueError("cap did not activate over the concentrated primary")
        with (directory / "dave-key.log").open("xb") as log:
            subprocess.run([str(binary), "keys", "add", "dave", "--keyring-backend", "test", "--no-backup", "--home", str(runner.home)], stdout=log, stderr=log, check=True, timeout=60)
        dave = query("keys", "show", "dave", "--keyring-backend", "test")["address"]
        third = query("keys", "show", "dave", "--bech", "val", "--keyring-backend", "test")["address"]
        send("fund-third-validator", "bank", "send", bob, dave, "25000000ualpha", signer="bob")
        third_home = directory / "third-validator"
        with (directory / "third-init.log").open("xb") as log:
            subprocess.run([str(binary), "init", "offline-third", "--home", str(third_home)], stdout=log, stderr=log, check=True, timeout=60)
        pubkey = json.loads(subprocess.check_output([str(binary), "comet", "show-validator", "--home", str(third_home)], text=True))
        third_file = directory / "third-validator.json"
        save(third_file, {"pubkey": pubkey, "amount": "10000000ualpha", "moniker": "offline-third", "commission-rate": "0", "commission-max-rate": "1", "commission-max-change-rate": "1", "min-self-delegation": "1"})
        send("third-validator", "staking", "create-validator", str(third_file), signer="dave")
        evidence.update(initial_cap=initial_cap, active_cap=cap)
        send("diversify-vault", "structs", "reactor-infuse", secondary, "2000000ualpha")
        send("diversify-third", "structs", "reactor-infuse", third, "2000000ualpha")
        evidence["diversified"] = verify_live(query, owner)
        if int(evidence["diversified"]["summary"]["appliedBonus"]) <= 0:
            raise ValueError("fixture requires a positive diversity bonus")
        phase("checking cap rejections")
        def vault_state():
            return {"balance": query("query", "bank", "balances", vault),
                "delegations": query("query", "staking", "delegations", vault),
                "diversity": query("query", "structs", "owner-reactor-diversity", owner)}
        before = vault_state()
        send("native-deposit-rejected", "staking", "delegate", primary, "1ualpha", expected_code=1613)
        send("vault-deposit-rejected", "structs", "reactor-infuse", primary, "1ualpha", expected_code=1613)
        if before != vault_state():
            raise ValueError("rejected deposits changed vault backing or diversity")
        unbond = send("outflow-allowed", "staking", "unbond", primary, "1ualpha")
        pending = query("query", "staking", "unbonding-delegations", alice)
        send("cancellation-rejected", "staking", "cancel-unbond", primary, "1ualpha", unbond["height"], expected_code=1613)
        if pending != query("query", "staking", "unbonding-delegations", alice):
            raise ValueError("rejected cancellation changed pending stake")
        send("create-guild", "structs", "guild-create", primary_id, "cap-fixture.local", "")
        guild = query("query", "structs", "player", owner)["Player"]["guildId"]
        send("set-join-minimum", "structs", "guild-update-join-infusion-minimum", guild, "1000000")
        with (directory / "carol-key.log").open("xb") as log:
            subprocess.run([str(binary), "keys", "add", "carol", "--keyring-backend", "test", "--no-backup", "--home", str(runner.home)], stdout=log, stderr=log, check=True, timeout=60)
        carol = query("keys", "show", "carol", "--keyring-backend", "test")["address"]
        send("fund-joiner", "bank", "send", bob, carol, "5000000ualpha", signer="bob")
        send("joiner-stake", "staking", "delegate", secondary, "1000000ualpha", signer="carol")
        joiner = query("query", "structs", "address", carol)["playerId"]
        def join_state():
            return {"player": query("query", "structs", "player", joiner)["Player"],
                "guild": query("query", "structs", "guild", guild)["Guild"],
                "delegations": query("query", "staking", "delegations", carol),
                "infusion": query("query", "structs", "infusion", secondary_id, carol),
                "diversity": query("query", "structs", "owner-reactor-diversity", joiner)}
        before = join_state()
        if before["player"].get("guildId"):
            raise ValueError("joiner was already a member")
        send("guild-migration-rejected", "structs", "guild-membership-join", guild, secondary_id + "-" + carol, signer="carol", expected_code=1613)
        if before != join_state():
            raise ValueError("failed guild migration changed membership, backing or power")
        evidence["failed_join_unchanged"] = before
        # The same player can use the separately enabled no-migration route.
        send("enable-purchase-admission", "structs", "guild-update-rank-purchase", guild, "0", "0", "100", "true")
        send("purchase-admission", "structs", "guild-membership-join-by-rank-purchase", guild, "100", "0", signer="carol")
        admitted = join_state()
        if admitted["player"].get("guildId") != guild or admitted["player"].get("guildRank") != "100":
            raise ValueError("purchase admission did not join at the bought rank")
        if any(admitted[name] != before[name] for name in ("delegations", "infusion", "diversity")):
            raise ValueError("purchase admission moved stake or changed reactor power")
        evidence["purchase_admission"] = admitted
        phase("waiting for native downtime jail")
        # Offline proposers add consensus-round delays, so 1,000 blocks can
        # take several minutes despite a 100ms commit timeout. Bound the wait
        # without confusing that expected cadence with a missing jail hook.
        deadline = time.monotonic() + 600
        last_healthy = runner.height(binary)
        if query("query", "staking", "validator", secondary, "--height", str(last_healthy))["validator"].get("jailed"):
            raise ValueError("fixture reached downtime before admission checks finished")
        while True:
            observed = runner.height(binary)
            validator = query("query", "staking", "validator", secondary, "--height", str(observed))["validator"]
            if validator.get("jailed") is True:
                break
            if node.poll() is not None or time.monotonic() >= deadline:
                raise TimeoutError("offline validator was not jailed")
            last_healthy = observed
            time.sleep(1)
        first_jail = next(height for height in range(last_healthy + 1, observed + 1)
            if query("query", "staking", "validator", secondary, "--height", str(height))["validator"].get("jailed") is True)
        if query("query", "staking", "validator", third, "--height", str(first_jail))["validator"].get("jailed"):
            raise ValueError("fixture failed to isolate one-validator bonus loss")
        evidence["one_jailed"] = verify_live(query, owner, height=str(first_jail))
        if not 0 < int(evidence["one_jailed"]["summary"]["appliedBonus"]) < int(evidence["diversified"]["summary"]["appliedBonus"]):
            raise ValueError("losing one of three reactors did not reduce the bonus")
        while query("query", "staking", "validator", third)["validator"].get("jailed") is not True:
            if node.poll() is not None or time.monotonic() >= deadline:
                raise TimeoutError("third validator did not enter downtime jail")
            time.sleep(1)
        runner.wait_height(binary, node, runner.height(binary) + 2)
        evidence["after_jail"] = verify_live(query, owner)
        if int(evidence["after_jail"]["summary"]["appliedBonus"]) != 0 or int(evidence["after_jail"]["capacity"]) >= int(evidence["diversified"]["capacity"]):
            raise ValueError("jail did not remove secondary power and diversity")
        sticky = query("query", "structs", "reactor-stake-share", primary_id)
        if sticky.get("capActive") is not True:
            raise ValueError("cap deactivated after validator count fell")
        evidence["sticky_cap"] = sticky
        height = runner.height(binary)
        runner.stop(node)
        phase("reopening database")
        node = start("reopened-node.log")
        runner.wait_height(binary, node, height + 3)
        evidence["reopened"] = verify_live(query, owner)
        for name in ("summary", "capacity", "owner", "infusions"):
            if evidence["reopened"][name] != evidence["after_jail"][name]:
                raise ValueError("database reopen changed " + name)
        if query("query", "structs", "reactor-stake-share", primary_id).get("capActive") is not True:
            raise ValueError("database reopen lost cap activation")
        runner.stop(node)
        export(binary, runner.home, directory / "after-export.json")
        evidence.update(differences=[], release_approved=False)
        runner.save("verification.json", evidence)
        phase("native cap, failed migration, downtime and reopen passed")
    except BaseException as error:
        phase("failed", error=str(error))
        raise
    finally:
        runner.save("queries.json", queries)
        runner.cleanup()


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", type=Path)
    parser.add_argument("--output-root", type=Path)
    parser.add_argument("--prepared", type=Path, help=argparse.SUPPRESS)
    parser.add_argument("--prepare-only", action="store_true")
    args = parser.parse_args()
    if args.prepared:
        run(args.prepared.resolve(strict=True))
    else:
        if args.binary is None or args.output_root is None:
            parser.error("--binary and --output-root are required")
        directory = prepare_cap(args.binary.resolve(strict=True), args.output_root.resolve(strict=True))
        print(f"Native cap evidence: {directory}", flush=True)
        if not args.prepare_only:
            os.execv(sys.executable, [sys.executable, str(directory / "source/tests/run_reactor_cap_lifecycle.py"), "--prepared", str(directory)])
