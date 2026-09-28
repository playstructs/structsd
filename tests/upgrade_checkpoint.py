"""Preserve a stopped native upgrade boundary and clone it for candidate replay."""
import hashlib
import json
import os
from pathlib import Path
import shutil
import tempfile


def file_hash(path):
    digest = hashlib.sha256()
    with path.open('rb') as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b''):
            digest.update(chunk)
    return digest.hexdigest()


def tree_hashes(directory, exclude=()):
    result = {}
    for path in sorted(directory.rglob('*')):
        if path.is_symlink():
            raise ValueError(f'checkpoint cannot contain a symlink: {path}')
        relative = str(path.relative_to(directory))
        if path.is_file() and relative not in exclude:
            result[relative] = file_hash(path)
    return result


def capture_checkpoint(directory, home, manifest, height, processes):
    # The caller has also completed the offline export. Never copy a database
    # with an owned node/fixture still running, including on error paths.
    if not processes or any(process.poll() is None for process in processes):
        raise ValueError('checkpoint requires all tracked processes to be stopped')
    if home.is_symlink() or home.resolve() != (directory / 'chain').resolve():
        raise ValueError('checkpoint home is outside the rehearsal')
    before = json.loads((directory / 'before.json').read_text())
    plan = json.loads((home / 'data/upgrade-info.json').read_text())
    if int(before['initial_height']) != height or plan['name'] != 'v0.22.0' or int(plan['height']) != height:
        raise ValueError('checkpoint is not the expected pre-upgrade boundary')
    # Validate the entire source before copying; following symlinks could copy
    # a different home or unrelated local data into the snapshot.
    tree_hashes(home)
    sidecars = ('before.json', 'before-inventory.json', 'passed-proposal.json',
                'proposal-transaction.json', 'vote-transaction.json')
    for name in sidecars:
        if (directory / name).is_symlink() or not (directory / name).is_file():
            raise ValueError(f'checkpoint requires a regular evidence file: {name}')
    target = directory / 'upgrade-checkpoint'
    target.mkdir(mode=0o700)
    shutil.copytree(home, target / 'home')
    for name in sidecars:
        shutil.copy2(directory / name, target / name)
    metadata = {'version': 1, 'upgrade_height': height, 'chain_id': manifest['chain_id'],
        'old_binary': manifest['binary'], 'old_binary_sha256': manifest['binary_sha256'],
        'source_rehearsal': str(directory), 'files_sha256': tree_hashes(target)}
    # A partial copy has no completion manifest and cannot be replayed.
    (target / 'checkpoint.json').write_text(json.dumps(metadata, indent=2) + '\n')
    return target


def clone_checkpoint(checkpoint, parent, binary):
    checkpoint, binary = checkpoint.resolve(strict=True), binary.resolve(strict=True)
    metadata = json.loads((checkpoint / 'checkpoint.json').read_text())
    if type(metadata.get('version')) is not int or metadata['version'] != 1 or tree_hashes(checkpoint, ('checkpoint.json',)) != metadata.get('files_sha256'):
        raise ValueError('checkpoint files changed or version is unsupported')
    if not binary.is_file() or not os.access(binary, os.X_OK):
        raise ValueError('candidate binary is not executable')
    directory = Path(tempfile.mkdtemp(prefix='structs-upgrade-replay-', dir=parent)).resolve()
    shutil.copytree(checkpoint / 'home', directory / 'chain')
    for name in ('before.json', 'before-inventory.json', 'passed-proposal.json',
                 'proposal-transaction.json', 'vote-transaction.json'):
        shutil.copy2(checkpoint / name, directory / name)
    # Snapshot a candidate too: rebuilding a workspace executable must not
    # silently change which code a queued replay will run.
    candidate = directory / 'structsd-candidate'
    shutil.copy2(binary, candidate)
    copied = {('chain/' + name[5:] if name.startswith('home/') else name): digest
        for name, digest in metadata['files_sha256'].items()}
    manifest = {'status': 'checkpoint replay prepared; no candidate started',
        'chain_home': str(directory / 'chain'), 'chain_id': metadata['chain_id'],
        'binary': metadata['old_binary'], 'binary_sha256': metadata['old_binary_sha256'],
        'new_binary': str(candidate), 'new_binary_sha256': file_hash(candidate),
        'upgrade_height': metadata['upgrade_height'], 'checkpoint': str(checkpoint),
        'checkpoint_sha256': file_hash(checkpoint / 'checkpoint.json'),
        'checkpoint_files_sha256': copied}
    (directory / 'manifest.json').write_text(json.dumps(manifest, indent=2) + '\n')
    return directory


def validate_replay(directory, manifest, binary):
    if manifest.get('status') != 'checkpoint replay prepared; no candidate started':
        raise ValueError('checkpoint replay has already started')
    if manifest.get('chain_home') != str(directory / 'chain'):
        raise ValueError('checkpoint replay home mismatch')
    if not os.access(binary, os.X_OK) or str(binary) != manifest.get('new_binary') or file_hash(binary) != manifest.get('new_binary_sha256'):
        raise ValueError('checkpoint candidate changed')
    actual = tree_hashes(directory)
    expected = dict(manifest['checkpoint_files_sha256'])
    expected['structsd-candidate'] = manifest['new_binary_sha256']
    # The replay manifest is not part of the captured database; everything
    # else must exactly match, including WAL and validator signing state.
    actual.pop('manifest.json', None)
    if actual != expected:
        raise ValueError('checkpoint replay files changed')
    before = json.loads((directory / 'before.json').read_text())
    plan = json.loads((directory / 'chain/data/upgrade-info.json').read_text())
    height = manifest['upgrade_height']
    if type(height) is not int or not 0 < height < 2**63 or int(before['initial_height']) != height or plan['name'] != 'v0.22.0' or int(plan['height']) != height:
        raise ValueError('checkpoint replay boundary mismatch')
    return height, before
