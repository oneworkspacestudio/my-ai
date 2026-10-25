"""Emit the checked Workspace managed-update contract for a prepared release.

Run after the native executable, portable ZIP and package/ tree are built.
This script has no publishing or credential access.
"""
from pathlib import Path
import argparse
import hashlib
import json
import zipfile

parser = argparse.ArgumentParser()
parser.add_argument('release', type=Path)
parser.add_argument('--version', required=True)
args = parser.parse_args()
root = Path(__file__).resolve().parents[1]
out = args.release.resolve()
version = args.version
assert json.loads((out / 'package/compatibility.json').read_text(encoding='utf-8'))['version'] == version
tag = 'v' + version
repo = 'oneworkspacestudio/my-ai'
asset = 'my-ai-' + version + '-windows-amd64.zip'
app = json.loads((root / 'integration/workspace-app.json').read_text(encoding='utf-8'))
app.update(version=version, source='https://github.com/' + repo + '/releases/tag/' + tag)
mapping = {'myai': ('MyAI.exe', 'MyAI.exe'), 'myai-contract': ('docs/MyAI/compatibility.json', 'compatibility.json')}
for name in ['LICENSE', 'NOTICE', 'THIRD-PARTY-LICENSES.txt']:
    mapping['myai-' + name] = ('licenses/MyAI-' + name, name)
for name in ['SETUP.md', 'SUPPORT.md', 'ADR-001-architecture.md', 'SECURITY-CHECKS.md']:
    mapping['myai-doc-' + name] = ('docs/MyAI/' + name, 'docs/' + name)
packages = {}
with zipfile.ZipFile(out / asset) as archive:
    assert len(archive.namelist()) == len(set(archive.namelist()))
    for key, (target, member) in mapping.items():
        payload = (out / 'package' / member).read_bytes()
        archive_member = 'MyAI/' + member
        assert archive.read(archive_member) == payload, archive_member
        packages[key] = {'file': target, 'size': len(payload), 'sha256': hashlib.sha256(payload).hexdigest(), 'asset': asset, 'member': archive_member}
manifest = {'schema': 2, 'integration': 3, 'repository': repo, 'tag': tag, 'minimumManager': '0.3.0-preview.87', 'apps': [app], 'packages': packages}
(out / 'workspace-update.json').write_text(json.dumps(manifest, indent=2) + '\n', encoding='utf-8')
print('Verified and emitted managed-update metadata for all9 My AI payloads')
