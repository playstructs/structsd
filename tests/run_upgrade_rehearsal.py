#!/usr/bin/env python3
"""Run a local v0.21.0 -> v0.22.0 rehearsal from prepared offline genesis.

Starts nodes and broadcasts fixture/governance transactions. Requires the live
test authorization described in AGENTS.md. Checks below are rehearsal evidence,
not complete release approval; see the v0.22.0 test audit for remaining gates.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import signal
import socket
import subprocess
import sys
import time
from urllib.request import urlopen

from upgrade_inventory import inventory
from upgrade_checkpoint import capture_checkpoint, clone_checkpoint, file_hash, validate_replay
from verify_upgrade_bank import verify_bank
from verify_upgrade_vesting import verify_vesting
from verify_upgrade_grid import verify_grid
from verify_upgrade_stake_cap import verify_stake_cap
from verify_reactor_diversity import verify_diversity
from verify_upgrade_allocations import verify_allocations
from verify_upgrade_staking import verify
from verify_upgrade_infusion_backing import verify_infusion_backing
from seed_upgrade_vesting import proof_tool_input, schedule_duration, seed_vesting
from prepare_upgrade_rehearsal import fixture_arguments
from run_export_restart import assert_no_nodes
from seed_upgrade_agreements import seed_agreements
from verify_upgrade_agreements import verify_legacy_agreements


class Rehearsal:
    def __init__(self, directory, new_binary):
        self.directory = directory.resolve(strict=True)
        self.manifest = json.loads((self.directory / "manifest.json").read_text())
        self.home = self.directory / "chain"
        self.old = Path(self.manifest["binary"])
        self.new = new_binary.resolve(strict=True)
        self.processes = []
        self.logs = []
        self.name = "v0.22.0"

    def save(self, name, value):
        (self.directory / name).write_text(json.dumps(value, indent=2) + "\n")

    def approved_upgrade_height(self):
        proposal = json.loads((self.directory / "passed-proposal.json").read_text())
        if not isinstance(proposal, dict):
            raise ValueError("resume requires a proposal object")
        messages = proposal.get("messages", [])
        if proposal.get("status") not in ("PROPOSAL_STATUS_PASSED", 3) or not isinstance(messages, list) or len(messages) != 1:
            raise ValueError("resume requires one passed software upgrade proposal")
        message = messages[0]
        if not isinstance(message, dict):
            raise ValueError("resume requires a software upgrade plan")
        # The release CLI emits Any as type/value; newer protobuf JSON uses
        # @type with flattened fields. Accept either exact form, never a mix.
        if "@type" in message:
            if "type" in message or "value" in message:
                raise ValueError("resume proposal mixes message encodings")
            message_type, payload = message["@type"], message
        else:
            message_type, payload = message.get("type"), message.get("value")
        if not isinstance(payload, dict) or not isinstance(payload.get("plan"), dict):
            raise ValueError("resume requires a software upgrade plan")
        plan = payload["plan"]
        height = str(plan.get("height", ""))
        if message_type != "/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade" or plan.get("name") != self.name:
            raise ValueError("resume proposal does not name this upgrade")
        if not height.isascii() or not height.isdecimal() or height.startswith("0") or int(height) >= 2**63:
            raise ValueError("resume proposal has an invalid upgrade height")
        return int(height)

    def preflight(self, mark_started=False, resume_approved=False):
        if self.manifest.get("chain_home") != str(self.home):
            raise ValueError("manifest home does not belong to this rehearsal")
        if resume_approved:
            if self.manifest.get("status") != "rehearsal started; not yet verified":
                raise ValueError("resume requires an interrupted populated rehearsal")
            if any((self.directory / name).exists() for name in ("new-node.log", "before.json", "upgrade-checkpoint")):
                raise ValueError("resume must precede boundary export and candidate startup")
            self.approved_upgrade_height()
        elif self.manifest.get("status") != "offline genesis validated; no node started or transactions broadcast":
            raise ValueError("requires a fresh --bootstrap artifact; reruns must prepare another directory")
        for path, expected in (
            (self.old, self.manifest["binary_sha256"]),
            (self.home / "config/genesis.json", self.manifest["genesis_sha256"]),
            (self.directory / "test_chain_v0.21.0.sh", self.manifest["fixture_sha256"]),
            (self.directory / "prepare_upgrade_rehearsal.py", self.manifest["routing_helper_sha256"]),
            (self.directory / "bin/structsd", self.manifest["wrapper_sha256"]),
        ):
            if hashlib.sha256(path.read_bytes()).hexdigest() != expected:
                raise ValueError(f"prepared artifact changed: {path}")
        if not os.access(self.new, os.X_OK):
            raise ValueError("new binary is not executable")
        for port in (26656, 26657, 1317, 9090):
            with socket.socket() as connection:
                connection.settimeout(0.2)
                if connection.connect_ex(("127.0.0.1", port)) == 0:
                    raise ValueError(f"local port {port} is already occupied; no process was stopped")
        report = {"status": "preflight passed; no node started or transactions broadcast",
            "chain_home": str(self.home), "chain_id": self.manifest["chain_id"],
            "old_binary": str(self.old), "old_binary_sha256": self.manifest["binary_sha256"],
            "new_binary": str(self.new), "new_binary_sha256": hashlib.sha256(self.new.read_bytes()).hexdigest(),
            "ports_checked": [26656, 26657, 1317, 9090]}
        if mark_started:
            previous = self.manifest.get("new_binary_sha256")
            if previous and previous != report["new_binary_sha256"]:
                self.manifest.setdefault("previous_candidate_sha256", []).append(previous)
            self.manifest.update({"status": "rehearsal started; not yet verified", "new_binary": str(self.new),
                "new_binary_sha256": report["new_binary_sha256"]})
            self.save("manifest.json", self.manifest)
        return report

    def run_approved_upgrade(self):
        self.preflight(mark_started=True, resume_approved=True)
        upgrade_height = self.approved_upgrade_height()
        old = self.start([str(self.old), "start", "--home", str(self.home), "--p2p.laddr", "tcp://127.0.0.1:26656"], f"old-node-resumed-{time.time_ns()}.log")
        self.wait_height(self.old, old, 1)
        plan = self.cli(self.old, "query", "upgrade", "plan").get("plan", {})
        if plan.get("name") != self.name or int(plan.get("height", 0)) != upgrade_height:
            raise ValueError("live upgrade plan disagrees with the approved proposal")
        self.finish_upgrade(old, upgrade_height)

    def run_replay(self):
        height, before = validate_replay(self.directory, self.manifest, self.new)
        for port in (26656, 26657, 1317, 9090):
            with socket.socket() as connection:
                connection.settimeout(0.2)
                if connection.connect_ex(("127.0.0.1", port)) == 0:
                    raise ValueError(f"local port {port} is already occupied; no process was stopped")
        self.manifest["status"] = "checkpoint candidate started; not yet verified"
        self.save("manifest.json", self.manifest)
        self.run_candidate(height, before)

    def cli(self, binary, *arguments):
        routed = fixture_arguments(list(arguments), str(self.home), self.manifest["chain_id"])
        result = subprocess.run([str(binary), *routed, "--output", "json"],
            capture_output=True, text=True, timeout=60)
        if result.returncode:
            raise RuntimeError(f"{' '.join(arguments[:3])}: {result.stderr.strip()}")
        return json.loads(result.stdout)

    def start(self, arguments, log_name, environment=None):
        if len(arguments) > 1 and arguments[1] == "start":
            # Port-only preflight misses a rehearsal running on custom ports.
            # Check immediately before every node launch, including cold replay.
            assert_no_nodes()
        log = (self.directory / log_name).open("xb")
        self.logs.append(log)
        process = subprocess.Popen(arguments, stdout=log, stderr=log, env=environment, start_new_session=True)
        self.processes.append(process)
        return process

    def stop(self, process):
        if process.poll() is None:
            try:
                os.killpg(process.pid, signal.SIGTERM)
            except ProcessLookupError:
                process.wait(timeout=10)
                return
            try:
                process.wait(timeout=20)
            except subprocess.TimeoutExpired:
                try:
                    os.killpg(process.pid, signal.SIGKILL)
                except ProcessLookupError:
                    pass
                process.wait(timeout=10)

    def cleanup(self):
        for process in reversed(self.processes):
            self.stop(process)
        for log in self.logs:
            log.close()

    def height(self, binary):
        status = self.cli(binary, "status")
        if status["node_info"]["network"] != self.manifest["chain_id"]:
            raise ValueError("local RPC is serving a different chain")
        return int(status["sync_info"]["latest_block_height"])

    def wait_height(self, binary, process, height, timeout=180):
        deadline = time.monotonic() + timeout
        while time.monotonic() < deadline:
            if process.poll() is not None:
                raise RuntimeError("node exited before the required height; inspect its log")
            try:
                if self.height(binary) >= height:
                    return
            except (RuntimeError, ValueError, KeyError):
                pass
            time.sleep(1)
        raise RuntimeError(f"node did not reach height {height}")

    def tx(self, *arguments, signer="alice"):
        self.height(self.old)
        response = self.cli(self.old, "tx", *arguments, "--from", signer, "--keyring-backend", "test",
            "--chain-id", self.manifest["chain_id"], "--gas", "auto", "--gas-adjustment", "1.5", "--yes")
        if int(response.get("code", 0)) != 0 or not response.get("txhash"):
            raise RuntimeError(f"governance broadcast failed: {response}")
        deadline = time.monotonic() + 30
        while time.monotonic() < deadline:
            try:
                delivered = self.cli(self.old, "query", "tx", response["txhash"])
            except RuntimeError:
                time.sleep(1)
                continue
            if delivered.get("txhash") != response["txhash"] or int(delivered.get("height", 0)) <= 0:
                raise RuntimeError("transaction query did not confirm the requested transaction")
            if int(delivered.get("code", 0)) != 0:
                raise RuntimeError(f"governance transaction failed in block: {delivered}")
            return delivered
        raise RuntimeError("governance transaction delivery was not confirmed")

    def export(self, binary, height, filename):
        path = self.directory / filename
        # Never export against a running node's database. SDK exports the state
        # at this height and records initial_height=height+1 for continuation.
        result = subprocess.run([str(binary), "export", "--home", str(self.home), "--height", str(height),
            "--output-document", str(path)], capture_output=True, text=True, timeout=300)
        if result.returncode:
            raise RuntimeError(f"export failed: {result.stderr.strip()}")
        document = json.loads(path.read_text())
        if int(document["initial_height"]) != height + 1:
            raise ValueError("export did not preserve the requested boundary height")
        return document

    def run(self, population_timeout, vesting_proof_tool=None, vesting_only=False, live_agreements=False):
        if live_agreements and vesting_only:
            raise ValueError("live agreements require the full released gameplay fixture")
        vesting_tool = proof_tool_input(vesting_proof_tool) if vesting_proof_tool is not None else None
        if vesting_only and vesting_tool is None:
            raise ValueError("vesting-only population requires the legacy proof tool")
        vesting_seconds = schedule_duration(self.manifest.get("vesting_fixture_schedule_seconds", 86400)) if vesting_tool is not None else None
        self.preflight(mark_started=True)
        if vesting_tool is not None:
            self.manifest["vesting_fixture_proof_tool"] = vesting_tool
            self.manifest["population_mode"] = "vesting_only" if vesting_only else "release_fixture_and_vesting"
            self.save("manifest.json", self.manifest)
        old = self.start([str(self.old), "start", "--home", str(self.home), "--p2p.laddr", "tcp://127.0.0.1:26656"], "old-node.log")
        self.wait_height(self.old, old, 5)
        environment = dict(os.environ, PATH=str(self.directory / "bin") + os.pathsep + os.environ["PATH"])
        if not vesting_only:
            fixture = self.start(["bash", str(self.directory / "test_chain_v0.21.0.sh"), "--skip-mining"], "population.log", environment)
            if fixture.wait(timeout=population_timeout) != 0:
                raise RuntimeError("v0.21 fixture population failed; no upgrade was proposed")
        self.seed_and_upgrade(old, vesting_tool, vesting_seconds, live_agreements)

    def seed_and_upgrade(self, old, vesting_tool, vesting_seconds, live_agreements):
        """Continue an owned, populated old node; callers must verify its origin."""
        if live_agreements:
            seed_agreements(self)
        if vesting_tool is not None:
            seed_vesting(self, vesting_tool, schedule_seconds=vesting_seconds)
        upgrade_height = self.height(self.old) + 240
        proposal_tx = self.tx("upgrade", "software-upgrade", self.name, "--title", "v0.22.0 economy rehearsal",
            "--summary", "Rehearse vault and reactor economy migration", "--upgrade-height", str(upgrade_height),
            "--upgrade-info", "{}", "--no-validate", "--deposit", "10000000ualpha")
        self.save("proposal-transaction.json", proposal_tx)
        proposal_ids = [attribute["value"] for event in proposal_tx.get("events", []) if event["type"] == "submit_proposal"
            for attribute in event["attributes"] if attribute["key"] == "proposal_id"]
        if len(proposal_ids) != 1:
            raise ValueError("cannot identify exactly one proposal from the delivered transaction")
        proposal_id = proposal_ids[0]
        self.save("vote-transaction.json", self.tx("gov", "vote", proposal_id, "yes"))
        deadline = time.monotonic() + 200
        while time.monotonic() < deadline:
            proposal = self.cli(self.old, "query", "gov", "proposal", proposal_id)["proposal"]
            status = proposal["status"]
            if status in ("PROPOSAL_STATUS_PASSED", 3):
                self.save("passed-proposal.json", proposal)
                break
            if status in ("PROPOSAL_STATUS_REJECTED", "PROPOSAL_STATUS_FAILED", 4, 5):
                raise RuntimeError(f"upgrade proposal ended with {status}")
            time.sleep(2)
        else:
            raise RuntimeError("upgrade proposal never passed")
        self.finish_upgrade(old, upgrade_height)

    def finish_upgrade(self, old, upgrade_height):
        upgrade_info = self.home / "data/upgrade-info.json"
        deadline = time.monotonic() + 600
        plan = None
        while time.monotonic() < deadline:
            if upgrade_info.exists():
                try:
                    plan = json.loads(upgrade_info.read_text())
                    break
                except json.JSONDecodeError:
                    # The old process may still be writing the boundary file.
                    time.sleep(1)
                    continue
            if old.poll() is not None and not upgrade_info.exists():
                raise RuntimeError("old node exited without recording the upgrade boundary")
            time.sleep(1)
        if plan is None:
            raise RuntimeError("old binary did not reach the upgrade boundary")
        if plan["name"] != self.name or int(plan["height"]) != upgrade_height:
            raise ValueError("old node halted for an unexpected upgrade")
        self.stop(old)
        if file_hash(self.old) != self.manifest["binary_sha256"]:
            raise ValueError("old binary changed before boundary export")
        before = self.export(self.old, upgrade_height - 1, "before.json")
        report = inventory(before)
        self.save("before-inventory.json", report)
        if report["problems"]:
            raise ValueError("pre-upgrade inventory failed; inspect before-inventory.json")
        capture_checkpoint(self.directory, self.home, self.manifest, upgrade_height, self.processes)
        self.run_candidate(upgrade_height, before)

    def run_candidate(self, upgrade_height, before):
        if file_hash(self.new) != self.manifest["new_binary_sha256"]:
            raise ValueError("candidate changed before upgrade startup")
        new = self.start([str(self.new), "start", "--home", str(self.home), "--p2p.laddr", "tcp://127.0.0.1:26656"], "new-node.log")
        target = upgrade_height + 2
        self.wait_height(self.new, new, target, timeout=300)
        blocks = []
        for height in range(upgrade_height, target + 1):
            with urlopen(f"http://127.0.0.1:26657/block_results?height={height}", timeout=30) as response:
                result = json.load(response)
            if "error" in result or int(result["result"]["height"]) != height:
                raise ValueError("upgrade block-results response is invalid")
            blocks.append(result["result"])
        self.save("upgrade-block-results.json", blocks)
        if any(block.get("txs_results") for block in blocks):
            raise ValueError("unexpected transactions in the migration comparison window; captured evidence requires attribution")
        agreements = before["app_state"]["structs"].get("agreementList", [])
        quiet_agreements = agreements and all(int(row["endBlock"]) > target + 1 for row in agreements)
        provider_accounts = {}
        if quiet_agreements:
            for provider in before["app_state"]["structs"]["providerList"]:
                pair = {}
                for label in ("collateral", "earnings"):
                    response = self.cli(self.new, "query", "structs", "provider-" + label + "-address", provider["id"])
                    associations = response["internalAddressAssociation"]
                    if len(associations) != 1:
                        raise ValueError("expected one provider pool association")
                    pair[label] = associations[0]["address"]
                provider_accounts[provider["id"]] = pair
            self.save("provider-accounts.json", provider_accounts)
        self.stop(new)
        after = self.export(self.new, target, "after.json")
        vaults = {}
        for block in blocks:
            for event in block.get("finalize_block_events") or []:
                if event["type"] == "structs.structs.EventVaultAddress":
                    attributes = {item["key"]: item["value"] for item in event["attributes"]}
                    detail = json.loads(attributes["eventVaultAddressDetail"])
                    owner, address = detail["ownerId"], detail["vaultAddress"]
                    if owner in vaults and vaults[owner] != address:
                        raise ValueError("conflicting vault associations in upgrade events")
                    vaults[owner] = address
        self.save("vault-map.json", vaults)
        inputs = {name: hashlib.sha256((self.directory / name).read_bytes()).hexdigest()
            for name in ("before.json", "after.json", "upgrade-block-results.json", "vault-map.json")}
        if quiet_agreements:
            agreement_result = verify_legacy_agreements(before, after, blocks, provider_accounts)
            agreement_result["input_sha256"] = dict(inputs, **{"provider-accounts.json": file_hash(self.directory / "provider-accounts.json")})
            self.save("agreement-verification.json", agreement_result)
            if agreement_result["differences"]:
                raise ValueError("legacy agreement terms, load, checkpoint or pools changed during quiet upgrade")
        result = verify(before, after, vaults)
        result["input_sha256"] = inputs
        self.save("staking-verification.json", result)
        if not vaults or result["differences"]:
            raise ValueError("upgrade association/staking verification failed")
        bank_result = verify_bank(before, after, blocks, vaults)
        bank_result["input_sha256"] = inputs
        self.save("bank-verification.json", bank_result)
        if bank_result["balance_differences"] or bank_result["migration_differences"]:
            raise ValueError("bank event replay or vault routing failed")
        vesting_result = verify_vesting(before, after, blocks, vaults)
        vesting_result["input_sha256"] = inputs
        self.save("vesting-verification.json", vesting_result)
        if vesting_result["differences"]:
            raise ValueError("vesting account event replay failed")
        grid_result = verify_grid(before, after, blocks)
        grid_result["input_sha256"] = inputs
        self.save("grid-infusion-verification.json", grid_result)
        if grid_result["differences"]:
            raise ValueError("grid/infusion event replay failed")
        allocation_result = verify_allocations(before, after, blocks)
        allocation_result["input_sha256"] = inputs
        self.save("allocation-verification.json", allocation_result)
        if allocation_result["differences"]:
            raise ValueError("allocation event replay or legacy type normalization failed")
        backing_result = verify_infusion_backing(after, vaults)
        backing_result["input_sha256"] = inputs
        self.save("infusion-backing-verification.json", backing_result)
        if backing_result["differences"]:
            raise ValueError("infusion staking backing verification failed")
        cap_result = verify_stake_cap(before, after, blocks, upgrade_height)
        cap_result["input_sha256"] = inputs
        self.save("stake-cap-verification.json", cap_result)
        if cap_result["differences"]:
            raise ValueError("stake cap configuration/activation event verification failed")
        diversity_result = verify_diversity(before, after, blocks, upgrade_height=upgrade_height)
        diversity_result["input_sha256"] = inputs
        diversity_result["verifier_sha256"] = {name: file_hash(Path(__file__).with_name(name)) for name in (
            "verify_reactor_diversity.py", "verify_upgrade_bank.py", "verify_upgrade_staking.py")}
        self.save("diversity-verification.json", diversity_result)
        if diversity_result["differences"]:
            raise ValueError("diversity accounting or summary event replay failed")
        self.save("rehearsal-result.json", {"upgrade_height": upgrade_height, "comparison_height": target,
            "checks": "confirmed governance, binary transition, continued blocks, associations, staking comparison, all-denomination bank event replay/vault routing, vesting account replay, infusion/effective grid replay, allocation replacement/deletion replay, stake cap activation/configuration events and independent diversity accounting/summary replay",
            "release_approved": False,
            "remaining": "final release-wide acceptance: retained vesting self-stake, economic/queue invariants, production cardinality, chain-event completeness and full test audit; indexer work is separate"})
        self.manifest["status"] = "rehearsal checks passed; release approval remains outstanding"
        self.save("manifest.json", self.manifest)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("directory", type=Path, help="prepared rehearsal directory, or saved checkpoint with --replay-checkpoint")
    parser.add_argument("--new-binary", type=Path, default=Path(__file__).resolve().parent.parent / "build/structsd-v0.22.0")
    parser.add_argument("--population-timeout", type=int, default=7200)
    parser.add_argument("--preflight-only", action="store_true", help="check artifact hashes and local ports without changing the manifest or starting processes")
    parser.add_argument("--vesting-proof-tool", type=Path, help="seed all four SDK vesting types, an unassociated control and ordinary active/pending stake; executable legacyvestingproof tool")
    parser.add_argument("--vesting-only", action="store_true", help="use the focused vesting population instead of repeating the release gameplay fixture; requires --vesting-proof-tool")
    parser.add_argument("--live-agreements", action="store_true", help="after the full gameplay fixture, create three live legacy contracts sharing one provider pool")
    mode = parser.add_mutually_exclusive_group()
    mode.add_argument("--resume-approved-upgrade", action="store_true", help="resume a stopped populated rehearsal with a passed proposal, before boundary export")
    mode.add_argument("--replay-checkpoint", action="store_true", help="treat directory as a saved boundary checkpoint; clone it and test the candidate against its native database")
    args = parser.parse_args()
    if args.vesting_only and args.vesting_proof_tool is None:
        parser.error("--vesting-only requires --vesting-proof-tool")
    if args.live_agreements and (args.vesting_only or args.resume_approved_upgrade or args.replay_checkpoint):
        parser.error("--live-agreements requires fresh full gameplay population")
    if args.vesting_proof_tool is not None:
        if args.resume_approved_upgrade or args.replay_checkpoint:
            parser.error("vesting seeding requires a fresh rehearsal, not a resumed or replayed boundary")
        proof_tool_input(args.vesting_proof_tool)
    if args.replay_checkpoint:
        if args.preflight_only:
            parser.error("--replay-checkpoint creates a private clone and cannot be combined with --preflight-only")
        args.directory = clone_checkpoint(args.directory, args.directory.parent, args.new_binary)
        args.new_binary = args.directory / "structsd-candidate"
        print(f"Checkpoint replay artifact: {args.directory}", flush=True)
    rehearsal = Rehearsal(args.directory, args.new_binary)
    if args.preflight_only:
        print(json.dumps(rehearsal.preflight(resume_approved=args.resume_approved_upgrade), indent=2))
        return
    try:
        if args.replay_checkpoint:
            rehearsal.run_replay()
        elif args.resume_approved_upgrade:
            rehearsal.run_approved_upgrade()
        else:
            rehearsal.run(args.population_timeout, args.vesting_proof_tool, args.vesting_only, args.live_agreements)
    finally:
        rehearsal.cleanup()
    print(f"Rehearsal checks completed; release approval remains outstanding. Evidence: {rehearsal.directory}")


if __name__ == "__main__":
    main()
