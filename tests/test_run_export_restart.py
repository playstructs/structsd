import subprocess
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

from run_export_restart import assert_no_nodes, run_mode, export as export_state


class ExportRestartGuardTests(unittest.TestCase):
    @patch("run_export_restart.subprocess.run")
    def test_height_export_keeps_stdout_logs_out_of_json(self, run):
        with tempfile.TemporaryDirectory() as temporary:
            target = Path(temporary) / "after.json"

            def command(arguments, **kwargs):
                self.assertEqual(arguments[-2:], ["--height", "42"])
                document = Path(arguments[arguments.index("--output-document") + 1])
                kwargs["stdout"].write(b"INFO loaded height 42\n")
                document.write_text('{"initial_height":43}')

            run.side_effect = command
            self.assertEqual(export_state(Path("/fake/structsd"), Path("/fake/home"), target, height=42), {"initial_height": 43})
            self.assertEqual(target.with_suffix(".stdout.log").read_text(), "INFO loaded height 42\n")
            with self.assertRaises(FileExistsError):
                export_state(Path("/fake/structsd"), Path("/fake/home"), target, height=42)
            self.assertEqual(run.call_count, 1)

    @patch("run_export_restart.assert_no_nodes")
    @patch("run_export_restart.export")
    @patch("run_export_restart.subprocess.run")
    @patch("run_export_restart.subprocess.Popen", side_effect=RuntimeError("stopped before node startup"))
    def test_sdk_app_genesis_validator_envelope(self, popen, run, export, guard):
        # SDK v0.53 AppGenesis nests validators and params under consensus.
        public_key = {"type": "tendermint/PubKeyEd25519", "value": "fixture-public-key"}
        export.return_value = {"chain_id": "local", "initial_height": "42", "app_state": {},
                               "consensus": {"validators": [{"pub_key": public_key, "power": "1"}], "params": {}}}
        with tempfile.TemporaryDirectory() as temporary:
            source = Path(temporary) / "source"
            (source / "config").mkdir(parents=True)
            (source / "config/priv_validator_key.json").write_text(json.dumps({"pub_key": public_key}))
            (source / "data").mkdir()
            (source / "data/priv_validator_state.json").write_text('{"height":"41"}')
            directory = Path(temporary) / "preserved"

            def init(arguments, **_):
                if arguments[1] == "init":
                    config = directory / "chain/config"
                    config.mkdir(parents=True)
                    (config / "config.toml").write_text('timeout_commit = "1s"\npprof_laddr = "localhost:6060"\n')

            run.side_effect = init
            with self.assertRaisesRegex(RuntimeError, "stopped before node startup"):
                run_mode(Path("/fake/structsd"), source, directory, "preserved", 3)
            genesis = json.loads((directory / "chain/config/genesis.json").read_text())
            self.assertEqual(genesis["chain_id"], "local-restart-preserved")
            self.assertEqual(genesis["consensus"]["validators"][0]["pub_key"], public_key)
            self.assertEqual(genesis["initial_height"], "42")
            self.assertFalse((directory / "chain/data/priv_validator_state.json").exists())
            popen.assert_called_once()

    @patch("run_export_restart.socket.socket")
    @patch("run_export_restart.subprocess.run")
    def test_any_structsd_node_blocks_even_on_custom_ports(self, run, socket):
        for command in ("/long/repository/path/structsd start --home /tmp/local --rpc.laddr tcp://127.0.0.1:45678",
                        "/path with spaces/structsd-v0.22.0 start --home /tmp/local", "structsd start"):
            with self.subTest(command=command):
                run.return_value = subprocess.CompletedProcess([], 0, stdout="123 " + command + "\n")
                with self.assertRaisesRegex(ValueError, "PID 123 is still active"):
                    assert_no_nodes()
        socket.assert_not_called()

    @patch("run_export_restart.socket.socket")
    @patch("run_export_restart.subprocess.run")
    def test_non_node_commands_do_not_block(self, run, socket):
        run.return_value = subprocess.CompletedProcess([], 0, stdout="123 /bin/structsd query structs player 1-1\n124 python3 tests/run_export_restart.py /tmp/home\n")
        assert_no_nodes()
        self.assertEqual(6, socket.return_value.__enter__.return_value.bind.call_count)

    @patch("run_export_restart.socket.socket")
    @patch("run_export_restart.subprocess.run")
    def test_occupied_port_refuses_without_stopping_processes(self, run, socket):
        run.return_value = subprocess.CompletedProcess([], 0, stdout="")
        socket.return_value.__enter__.return_value.bind.side_effect = OSError("occupied")
        with self.assertRaisesRegex(ValueError, "port 26656 is occupied"):
            assert_no_nodes()
        self.assertEqual(1, run.call_count)


if __name__ == "__main__":
    unittest.main()
