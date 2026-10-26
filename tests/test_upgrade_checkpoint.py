"""Checkpoint controls use temporary files and mocked processes, never nodes."""
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import Mock, patch

from run_upgrade_rehearsal import Rehearsal
from upgrade_checkpoint import capture_checkpoint, clone_checkpoint, file_hash, tree_hashes, validate_replay


def fixture(parent):
    directory = parent / 'original'
    home = directory / 'chain'
    (home / 'data/application.db').mkdir(parents=True)
    (home / 'config').mkdir()
    (home / 'data/application.db/000001.ldb').write_bytes(b'old native database')
    (home / 'data/cs.wal').write_bytes(b'consensus WAL at boundary')
    (home / 'data/priv_validator_state.json').write_text('{"height":"10","round":0,"step":2}')
    (home / 'config/genesis.json').write_text('{"chain_id":"test"}')
    (home / 'data/upgrade-info.json').write_text('{"name":"v0.22.0","height":10}')
    before = {'initial_height': '10', 'app_state': {'marker': 'old state'}}
    (directory / 'before.json').write_text(json.dumps(before))
    (directory / 'before-inventory.json').write_text('{"problems":[]}')
    proposal = {'status': 'PROPOSAL_STATUS_PASSED', 'messages': [
        {'@type': '/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade', 'plan': {'name': 'v0.22.0', 'height': '10'}}]}
    (directory / 'passed-proposal.json').write_text(json.dumps(proposal))
    for name in ('proposal-transaction.json', 'vote-transaction.json'):
        (directory / name).write_text('{"code":0,"height":"5","txhash":"test"}')
    old, new = parent / 'old', parent / 'new'
    old.write_bytes(b'old executable')
    new.write_bytes(b'candidate executable')
    old.chmod(0o700)
    new.chmod(0o700)
    manifest = {'chain_id': 'test', 'chain_home': str(home), 'binary': str(old), 'binary_sha256': file_hash(old)}
    (directory / 'manifest.json').write_text(json.dumps(manifest))
    return directory, home, manifest, new, before


class UpgradeCheckpointTests(unittest.TestCase):
    def test_legacy_proposal_envelope_matches_actual_release_cli(self):
        for fault in ('none', 'mixed', 'wrong_type', 'bad_height', 'null_value'):
            with self.subTest(fault=fault), tempfile.TemporaryDirectory() as temporary:
                directory, _, _, new, _ = fixture(Path(temporary).resolve())
                path = directory / 'passed-proposal.json'
                proposal = json.loads(path.read_text())
                modern = proposal['messages'][0]
                kind = modern.pop('@type')
                message = {'type': kind, 'value': modern}
                if fault == 'mixed': message['@type'] = kind
                elif fault == 'wrong_type': message['type'] = '/cosmos.upgrade.v1beta1.MsgCancelUpgrade'
                elif fault == 'bad_height': message['value']['plan']['height'] = '010'
                elif fault == 'null_value': message['value'] = None
                proposal['messages'] = [message]
                path.write_text(json.dumps(proposal))
                runner = Rehearsal(directory, new)
                if fault == 'none':
                    self.assertEqual(runner.approved_upgrade_height(), 10)
                else:
                    with self.assertRaises(ValueError):
                        runner.approved_upgrade_height()

    def test_start_cannot_overwrite_previous_process_evidence(self):
        with tempfile.TemporaryDirectory() as temporary:
            directory, _, _, new, _ = fixture(Path(temporary).resolve())
            log = directory / 'old-node.log'
            log.write_bytes(b'previous failed attempt')
            runner = Rehearsal(directory, new)
            with patch('run_upgrade_rehearsal.subprocess.Popen') as start:
                with self.assertRaises(FileExistsError):
                    runner.start(['never-start'], 'old-node.log')
                start.assert_not_called()
            self.assertEqual(log.read_bytes(), b'previous failed attempt')

    def test_boundary_capture_precedes_candidate_start_and_follows_node_stop(self):
        with tempfile.TemporaryDirectory() as temporary:
            directory, _, manifest, new, before = fixture(Path(temporary).resolve())
            runner = Rehearsal(directory, new)
            old = Mock()
            old.poll.return_value = None
            runner.processes = [old]
            order = []
            def stop(process):
                self.assertIs(process, old)
                old.poll.return_value = 0
                order.append('stopped')
            def export(binary, height, name):
                self.assertEqual(order, ['stopped'])
                self.assertEqual((binary, height, name), (runner.old, 9, 'before.json'))
                order.append('exported')
                return before
            def candidate(height, state):
                self.assertEqual(order, ['stopped', 'exported'])
                metadata = json.loads((directory / 'upgrade-checkpoint/checkpoint.json').read_text())
                self.assertEqual(metadata['upgrade_height'], height)
                self.assertEqual(metadata['old_binary_sha256'], manifest['binary_sha256'])
                self.assertEqual(state, before)
                order.append('candidate')
            runner.stop, runner.export, runner.run_candidate = stop, export, candidate
            with patch('run_upgrade_rehearsal.inventory', return_value={'problems': []}):
                runner.finish_upgrade(old, 10)
            self.assertEqual(order, ['stopped', 'exported', 'candidate'])

    def test_approved_resume_requires_matching_live_plan(self):
        for fault in ('none', 'wrong_name', 'wrong_height', 'missing_plan'):
            with self.subTest(fault=fault), tempfile.TemporaryDirectory() as temporary:
                directory, _, _, new, _ = fixture(Path(temporary).resolve())
                runner = Rehearsal(directory, new)
                runner.preflight, runner.wait_height, runner.finish_upgrade = Mock(), Mock(), Mock()
                old = Mock()
                runner.start = Mock(return_value=old)
                plan = {'name': 'v0.22.0', 'height': '10'}
                if fault == 'wrong_name':
                    plan['name'] = 'different'
                elif fault == 'wrong_height':
                    plan['height'] = '11'
                elif fault == 'missing_plan':
                    plan = {}
                runner.cli = Mock(return_value={'plan': plan})
                if fault == 'none':
                    runner.run_approved_upgrade()
                    runner.finish_upgrade.assert_called_once_with(old, 10)
                else:
                    with self.assertRaisesRegex(ValueError, 'live upgrade plan'):
                        runner.run_approved_upgrade()
                    runner.finish_upgrade.assert_not_called()
                runner.cli.assert_called_once_with(runner.old, 'query', 'upgrade', 'plan')

    def test_approved_resume_denies_malformed_or_wrong_proposals(self):
        for fault in ('not_passed', 'empty_messages', 'two_messages', 'wrong_type', 'wrong_name',
                      'zero', 'negative', 'leading_zero', 'unicode_digits', 'overflow', 'bool', 'null_plan'):
            with self.subTest(fault=fault), tempfile.TemporaryDirectory() as temporary:
                directory, _, _, new, _ = fixture(Path(temporary).resolve())
                path = directory / 'passed-proposal.json'
                proposal = json.loads(path.read_text())
                message = proposal['messages'][0]
                if fault == 'not_passed': proposal['status'] = 'PROPOSAL_STATUS_VOTING_PERIOD'
                elif fault == 'empty_messages': proposal['messages'] = []
                elif fault == 'two_messages': proposal['messages'].append(message)
                elif fault == 'wrong_type': message['@type'] = '/cosmos.upgrade.v1beta1.MsgCancelUpgrade'
                elif fault == 'wrong_name': message['plan']['name'] = 'v0.23.0'
                elif fault == 'null_plan': message['plan'] = None
                else:
                    message['plan']['height'] = {'zero':'0', 'negative':'-1', 'leading_zero':'010',
                        'unicode_digits':'１０', 'overflow':str(2**63), 'bool':True}[fault]
                path.write_text(json.dumps(proposal))
                runner = Rehearsal(directory, new)
                with self.assertRaises(ValueError):
                    runner.approved_upgrade_height()

    def test_capture_and_clone_preserve_native_state_and_freeze_candidate(self):
        with tempfile.TemporaryDirectory() as temporary:
            parent = Path(temporary).resolve()
            directory, home, manifest, new, before = fixture(parent)
            stopped = Mock()
            stopped.poll.return_value = 0
            checkpoint = capture_checkpoint(directory, home, manifest, 10, [stopped])
            original_hashes = tree_hashes(checkpoint)
            (home / 'data/application.db/000001.ldb').write_bytes(b'original upgraded after capture')
            replay = clone_checkpoint(checkpoint, parent, new)
            metadata = json.loads((replay / 'manifest.json').read_text())
            candidate = replay / 'structsd-candidate'
            new.write_bytes(b'a later workspace build')
            self.assertEqual(candidate.read_bytes(), b'candidate executable')
            self.assertEqual((replay / 'chain/data/application.db/000001.ldb').read_bytes(), b'old native database')
            self.assertEqual((replay / 'chain/data/cs.wal').read_bytes(), b'consensus WAL at boundary')
            self.assertEqual((replay / 'chain/data/priv_validator_state.json').read_bytes(),
                             (checkpoint / 'home/data/priv_validator_state.json').read_bytes())
            self.assertEqual(validate_replay(replay, metadata, candidate), (10, before))
            self.assertEqual(tree_hashes(checkpoint), original_hashes)
            with patch('run_upgrade_rehearsal.socket.socket') as socket:
                socket.return_value.__enter__.return_value.connect_ex.return_value = 1
                runner = Rehearsal(replay, candidate)
                runner.run_candidate = Mock()
                runner.run_replay()
                runner.run_candidate.assert_called_once_with(10, before)
                with self.assertRaisesRegex(ValueError, 'already started'):
                    runner.run_replay()

    def test_capture_refuses_live_incomplete_or_wrong_boundary(self):
        for fault in ('live', 'no_process', 'height', 'plan', 'symlink', 'evidence_symlink', 'missing_evidence'):
            with self.subTest(fault=fault), tempfile.TemporaryDirectory() as temporary:
                directory, home, manifest, _, _ = fixture(Path(temporary).resolve())
                process = Mock()
                process.poll.return_value = None if fault == 'live' else 0
                if fault == 'height':
                    (directory / 'before.json').write_text('{"initial_height":"9"}')
                if fault == 'plan':
                    (home / 'data/upgrade-info.json').write_text('{"name":"different","height":10}')
                if fault == 'symlink':
                    (home / 'data/unrelated').symlink_to(directory / 'before.json')
                if fault in ('evidence_symlink', 'missing_evidence'):
                    (directory / 'vote-transaction.json').unlink()
                    if fault == 'evidence_symlink':
                        (directory / 'vote-transaction.json').symlink_to(directory / 'proposal-transaction.json')
                with self.assertRaises(ValueError):
                    capture_checkpoint(directory, home, manifest, 10, [] if fault == 'no_process' else [process])
                self.assertFalse((directory / 'upgrade-checkpoint').exists())

    def test_changed_checkpoint_or_replay_cannot_start_candidate(self):
        for fault in ('source_database', 'source_extra', 'source_symlink', 'replay_database',
                      'replay_signing_state', 'replay_extra', 'candidate', 'candidate_mode', 'height'):
            with self.subTest(fault=fault), tempfile.TemporaryDirectory() as temporary:
                parent = Path(temporary).resolve()
                directory, home, manifest, new, _ = fixture(parent)
                stopped = Mock()
                stopped.poll.return_value = 0
                checkpoint = capture_checkpoint(directory, home, manifest, 10, [stopped])
                if fault.startswith('source_'):
                    if fault == 'source_database':
                        (checkpoint / 'home/data/application.db/000001.ldb').write_bytes(b'changed')
                    elif fault == 'source_extra':
                        (checkpoint / 'home/extra').write_bytes(b'new file')
                    else:
                        (checkpoint / 'home/extra').symlink_to(new)
                    with self.assertRaises(ValueError):
                        clone_checkpoint(checkpoint, parent, new)
                    self.assertEqual(list(parent.glob('structs-upgrade-replay-*')), [])
                    continue
                replay = clone_checkpoint(checkpoint, parent, new)
                candidate = replay / 'structsd-candidate'
                metadata = json.loads((replay / 'manifest.json').read_text())
                if fault == 'replay_database':
                    (replay / 'chain/data/application.db/000001.ldb').write_bytes(b'changed')
                elif fault == 'replay_signing_state':
                    (replay / 'chain/data/priv_validator_state.json').write_text('{}')
                elif fault == 'replay_extra':
                    (replay / 'checkpoint.json').write_text('{}')
                elif fault == 'candidate':
                    candidate.write_bytes(b'changed candidate')
                elif fault == 'candidate_mode':
                    candidate.chmod(0o600)
                else:
                    metadata['upgrade_height'] = 11
                    (replay / 'manifest.json').write_text(json.dumps(metadata))
                runner = Rehearsal(replay, candidate)
                runner.run_candidate = Mock()
                with self.assertRaises(ValueError):
                    runner.run_replay()
                runner.run_candidate.assert_not_called()

    def test_occupied_port_preserves_pristine_replay(self):
        with tempfile.TemporaryDirectory() as temporary:
            parent = Path(temporary).resolve()
            directory, home, manifest, new, _ = fixture(parent)
            stopped = Mock()
            stopped.poll.return_value = 0
            checkpoint = capture_checkpoint(directory, home, manifest, 10, [stopped])
            replay = clone_checkpoint(checkpoint, parent, new)
            original = tree_hashes(replay)
            runner = Rehearsal(replay, replay / 'structsd-candidate')
            runner.run_candidate = Mock()
            with patch('run_upgrade_rehearsal.socket.socket') as socket:
                socket.return_value.__enter__.return_value.connect_ex.return_value = 0
                with self.assertRaisesRegex(ValueError, 'occupied'):
                    runner.run_replay()
            self.assertEqual(tree_hashes(replay), original)
            runner.run_candidate.assert_not_called()


if __name__ == '__main__':
    unittest.main()
