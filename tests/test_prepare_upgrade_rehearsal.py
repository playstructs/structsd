import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

from prepare_upgrade_rehearsal import bootstrap, fixture_arguments, isolated_arguments, prepare


class UpgradePreparationTests(unittest.TestCase):
    def test_network_commands_pin_endpoint_and_transaction_chain_id(self):
        home = "/tmp/isolated home"
        for arguments in (
            ["tx", "bank", "send", "alice", "bob", "1ualpha"],
            ["--home", "tx", "--keyring-dir", "/old", "--gas", "auto",
             "--gas-adjustment", "1.5", "--output", "json", "--yes=true",
             "tx", "structs", "allocation-create", "1-1", "100"],
            ["--chain-id=wrong", "tx", "bank", "send", "--chain-id", "also-wrong"],
            ["--node", "https://unrelated.invalid", "tx", "bank", "send", "--node=also-wrong"],
        ):
            self.assertEqual(fixture_arguments(arguments, home, "rehearsal"),
                isolated_arguments(arguments, home) + ["--chain-id", "rehearsal", "--node", "tcp://127.0.0.1:26657"])
        for arguments in (["query", "tx", "ABC"],
                ["--home", "tx", "status"], ["query", "bank", "balances", "address"],
                ["--node=wrong", "query", "structs", "player-all", "--node", "also-wrong"]):
            self.assertEqual(fixture_arguments(arguments, home, "rehearsal"),
                isolated_arguments(arguments, home) + ["--node", "tcp://127.0.0.1:26657"])
        for arguments in (["keys", "add", "tx"], ["keys", "show", "query"], ["export"]):
            self.assertEqual(fixture_arguments(arguments, home, "rehearsal"),
                isolated_arguments(arguments, home))
        for arguments in (["--gas"], ["--unknown", "tx"], ["--", "tx"]):
            with self.assertRaises(ValueError):
                fixture_arguments(arguments, home, "rehearsal")

    def test_routes_both_flag_forms_without_splitting_arguments(self):
        home = "/tmp/chain home with $ and ` characters"
        routed = isolated_arguments([
            "--home", "~/.structs", "tx", "bank", "send", "alice", "bob",
            "1ualpha", "--keyring-dir=/someone/elses/keyring", "--home=/other",
        ], home)
        self.assertEqual(routed, [
            "--home", home, "tx", "bank", "send", "alice", "bob", "1ualpha",
            "--keyring-dir=" + home, "--home=" + home, "--home", home,
        ])
        for arguments in (["--home"], ["--keyring-dir", "--home=x"], ["--", "--home=x"]):
            with self.assertRaises(ValueError):
                isolated_arguments(arguments, home)

    def test_prepares_exact_release_fixture_and_wrapper_without_running_binary(self):
        repository = Path(__file__).resolve().parent.parent
        with tempfile.TemporaryDirectory(prefix="upgrade preparation ") as parent:
            binary = Path(parent) / "fake binary"
            marker = Path(parent) / "invocations.json"
            binary.write_text(
                f"#!{sys.executable}\nimport json, sys\nfrom pathlib import Path\n"
                f"Path({str(marker)!r}).write_text(json.dumps(sys.argv[1:]))\n"
            )
            binary.chmod(0o700)
            directory = prepare(repository, Path(parent), binary)
            self.assertFalse(marker.exists(), "preparation must not invoke the binary")
            self.assertFalse((directory / "chain").exists())
            manifest = json.loads((directory / "manifest.json").read_text())
            expected = subprocess.check_output([
                "git", "-C", str(repository), "show",
                manifest["baseline_commit"] + ":tests/test_chain.sh",
            ])
            self.assertEqual(expected, (directory / "test_chain_v0.21.0.sh").read_bytes())
            subprocess.run([str(directory / "bin/structsd"), "query", "tx", "hash", "--home", "/original/home"], check=True)
            self.assertEqual(json.loads(marker.read_text()), [
                "query", "tx", "hash", "--home", str(directory / "chain"), "--home", str(directory / "chain"),
                "--node", "tcp://127.0.0.1:26657",
            ])
            subprocess.run([str(directory / "bin/structsd"), "tx", "bank", "send", "--chain-id", "wrong"], check=True)
            self.assertEqual(json.loads(marker.read_text())[-6:], [
                "--home", str(directory / "chain"), "--chain-id", "structs-upgrade-v022",
                "--node", "tcp://127.0.0.1:26657",
            ])
            marker.unlink()
            home = directory / "chain"
            home.mkdir()
            sentinel = home / "existing-state"
            sentinel.write_text("retain")
            with self.assertRaises(FileExistsError):
                bootstrap(directory)
            self.assertEqual(sentinel.read_text(), "retain")
            self.assertFalse(marker.exists(), "existing homes must be rejected before invoking the binary")
            sentinel.unlink()
            home.rmdir()
            binary.write_text(binary.read_text() + "# changed since preparation\n")
            with self.assertRaisesRegex(ValueError, "binary changed"):
                bootstrap(directory)
            self.assertFalse(home.exists())
            self.assertFalse(marker.exists())


if __name__ == "__main__":
    unittest.main()
