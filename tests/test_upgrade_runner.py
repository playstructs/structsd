"""Runner control-flow tests use mocks exclusively; never start a process/node."""
from pathlib import Path
import hashlib
import json
import signal
import subprocess
import tempfile
import unittest
from unittest.mock import Mock, call, patch

from run_upgrade_rehearsal import Rehearsal


def rehearsal():
    value = object.__new__(Rehearsal)
    value.old, value.new = Path("/fake/old"), Path("/fake/new")
    value.directory, value.home = Path("/fake/rehearsal"), Path("/fake/rehearsal/chain")
    value.manifest = {"chain_id": "test"}
    return value


class UpgradeRunnerTests(unittest.TestCase):
    @patch("run_upgrade_rehearsal.subprocess.Popen")
    @patch("run_upgrade_rehearsal.assert_no_nodes")
    def test_active_node_refuses_start_before_opening_logs(self, guard, popen):
        runner = rehearsal()
        guard.side_effect = ValueError("native Structs node PID 123 is still active")
        with self.assertRaisesRegex(ValueError, "PID 123"):
            runner.start([str(runner.new), "start", "--home", str(runner.home)], "node.log")
        guard.assert_called_once_with()
        popen.assert_not_called()

    @patch("run_upgrade_rehearsal.subprocess.Popen")
    @patch("run_upgrade_rehearsal.assert_no_nodes")
    def test_population_script_can_run_alongside_its_owned_node(self, guard, popen):
        runner = rehearsal()
        with tempfile.TemporaryDirectory() as temporary:
            runner.directory = Path(temporary)
            runner.logs, runner.processes = [], []
            try:
                runner.start(["bash", "fixture.sh"], "population.log")
                guard.assert_not_called()
                popen.assert_called_once()
                self.assertEqual([popen.return_value], runner.processes)
            finally:
                for log in runner.logs:
                    log.close()

    @patch("run_upgrade_rehearsal.subprocess.run")
    def test_cli_pins_local_node_despite_an_explicit_other_endpoint(self, run):
        run.return_value = subprocess.CompletedProcess([], 0, '{}', '')
        runner = rehearsal()
        runner.cli(runner.new, "query", "bank", "balances", "address", "--node", "https://unrelated.invalid")
        argv = run.call_args.args[0]
        self.assertEqual(argv[-4:], ["--node", "tcp://127.0.0.1:26657", "--output", "json"])
        runner.cli(runner.old, "tx", "gov", "vote", "1", "yes", "--chain-id=wrong")
        self.assertEqual(run.call_args.args[0][-6:], ["--chain-id", "test", "--node", "tcp://127.0.0.1:26657", "--output", "json"])

    def test_wrong_local_chain_prevents_governance_broadcast(self):
        runner = rehearsal()
        runner.cli = Mock(return_value={"node_info": {"network": "different"}, "sync_info": {"latest_block_height": "42"}})
        with self.assertRaisesRegex(ValueError, "different chain"):
            runner.tx("gov", "vote", "1", "yes")
        runner.cli.assert_called_once_with(runner.old, "status")
        runner.cli.return_value["node_info"]["network"] = "test"
        self.assertEqual(runner.height(runner.old), 42)

    @patch("run_upgrade_rehearsal.socket.socket")
    def test_preflight_is_read_only_and_verifies_routing_artifacts(self, socket):
        socket.return_value.__enter__.return_value.connect_ex.return_value = 1
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary).resolve()
            home = directory / "chain"
            (home / "config").mkdir(parents=True)
            (directory / "bin").mkdir()
            files = {
                "binary_sha256": directory / "old",
                "genesis_sha256": home / "config/genesis.json",
                "fixture_sha256": directory / "test_chain_v0.21.0.sh",
                "routing_helper_sha256": directory / "prepare_upgrade_rehearsal.py",
                "wrapper_sha256": directory / "bin/structsd",
            }
            manifest = {"binary": str(files["binary_sha256"]), "chain_home": str(home), "chain_id": "test",
                "status": "offline genesis validated; no node started or transactions broadcast"}
            for key, path in files.items():
                path.write_text("fixture: " + key)
                manifest[key] = hashlib.sha256(path.read_bytes()).hexdigest()
            new = directory / "new"
            new.write_text("not invoked")
            new.chmod(0o700)
            manifest_path = directory / "manifest.json"
            manifest_path.write_text(json.dumps(manifest))
            before = manifest_path.read_bytes()
            runner = Rehearsal(directory, new)
            runner.start = Mock()
            report = runner.preflight()
            self.assertEqual(report["new_binary_sha256"], hashlib.sha256(new.read_bytes()).hexdigest())
            self.assertEqual(manifest_path.read_bytes(), before)
            self.assertEqual(runner.manifest, manifest)
            runner.start.assert_not_called()
            for key in ("routing_helper_sha256", "wrapper_sha256"):
                path = files[key]
                original = path.read_bytes()
                path.write_bytes(original + b"tampered")
                with self.assertRaisesRegex(ValueError, "artifact changed"):
                    runner.preflight()
                self.assertEqual(manifest_path.read_bytes(), before)
                path.write_bytes(original)
            socket.return_value.__enter__.return_value.connect_ex.return_value = 0
            with self.assertRaisesRegex(ValueError, "already occupied"):
                runner.preflight(mark_started=True)
            self.assertEqual(manifest_path.read_bytes(), before)
            socket.return_value.__enter__.return_value.connect_ex.return_value = 1
            runner.preflight(mark_started=True)
            self.assertEqual(json.loads(manifest_path.read_text())["status"], "rehearsal started; not yet verified")
            runner.start.assert_not_called()
            passed = directory / 'passed-proposal.json'
            passed.write_text(json.dumps({'status': 'PROPOSAL_STATUS_PASSED', 'messages': [
                {'@type': '/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade', 'plan': {'name': 'v0.22.0', 'height': '10'}}]}))
            before_resume = manifest_path.read_bytes()
            runner.preflight(resume_approved=True)
            self.assertEqual(manifest_path.read_bytes(), before_resume)
            for name in ('new-node.log', 'before.json', 'upgrade-checkpoint'):
                marker = directory / name
                marker.touch()
                with self.assertRaisesRegex(ValueError, 'precede boundary'):
                    runner.preflight(resume_approved=True)
                marker.unlink()
            passed.unlink()
            with self.assertRaises(FileNotFoundError):
                runner.preflight(resume_approved=True)
            runner.start.assert_not_called()

    def test_population_failure_prevents_governance_and_binary_swap(self):
        runner = rehearsal()
        runner.preflight, runner.wait_height, runner.tx = Mock(), Mock(), Mock()
        old, fixture = Mock(), Mock()
        fixture.wait.return_value = 1
        runner.start = Mock(side_effect=[old, fixture])
        with self.assertRaisesRegex(RuntimeError, "population failed"):
            runner.run(1)
        self.assertEqual(runner.start.call_count, 2)
        runner.tx.assert_not_called()

    def test_governance_requires_matching_committed_success(self):
        for delivered in (
            {"txhash": "OTHER", "height": "3", "code": 0},
            {"txhash": "ABC", "height": "0", "code": 0},
            {"txhash": "ABC", "height": "3", "code": 5},
        ):
            runner = rehearsal()
            runner.height = Mock(return_value=2)
            runner.cli = Mock(side_effect=[{"txhash": "ABC", "code": 0}, delivered])
            with self.assertRaises(RuntimeError):
                runner.tx("gov", "vote", "1", "yes")
        runner = rehearsal()
        runner.height = Mock(return_value=2)
        runner.cli = Mock(return_value={"code": 0})
        with self.assertRaisesRegex(RuntimeError, "broadcast failed"):
            runner.tx("gov", "vote", "1", "yes")
        self.assertEqual(runner.cli.call_count, 1)
        delivered = {"txhash": "ABC", "height": "3"}
        runner.cli = Mock(side_effect=[{"txhash": "ABC", "code": 0}, delivered])
        self.assertEqual(runner.tx("gov", "vote", "1", "yes"), delivered)

    @patch("run_upgrade_rehearsal.os.killpg")
    def test_stop_only_signals_its_tracked_process_group(self, killpg):
        process = Mock(pid=12345)
        process.poll.return_value = None
        process.wait.side_effect = [subprocess.TimeoutExpired("node", 20), 0]
        rehearsal().stop(process)
        self.assertEqual(killpg.call_args_list, [call(12345, signal.SIGTERM), call(12345, signal.SIGKILL)])
        killpg.reset_mock()
        process.poll.return_value = 0
        rehearsal().stop(process)
        killpg.assert_not_called()


if __name__ == "__main__":
    unittest.main()
