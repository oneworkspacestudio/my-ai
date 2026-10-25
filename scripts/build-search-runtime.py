"""Build the pinned, separately licensed private SearXNG Windows runtime.

Inputs live outside the repository: verified upstream ZIP, PyPI wheel lock,
and CPython3.13.18 built from its verified official source using MSBuild.
No installer-time network, pip, Docker, WSL, global Python or public instance.
"""
from pathlib import Path
import argparse
import hashlib
import json
import zipfile
import difflib
import io

parser = argparse.ArgumentParser()
parser.add_argument('inputs', type=Path)
args = parser.parse_args()
inputs = args.inputs.resolve()
repo = Path(__file__).resolve().parents[1]
out = repo / 'internal/searchruntime'
sha = lambda b: hashlib.sha256(b).hexdigest()
lock = json.loads((inputs / 'wheels-lock.json').read_text(encoding='utf-8'))
files = {}
def add(name, data):
    assert name and ':' not in name and '..' not in name.split('/')
    assert name.lower() not in {n.lower() for n in files}, name
    files[name] = data

# The official3.13.17 package supplies only the redistribution licences and
# Microsoft VC runtime. Every Python executable/module and stdlib is replaced
# by the patched3.13.18 source build, not the superseded interpreter.
with zipfile.ZipFile(inputs / 'python-3.13.17-embed-amd64.zip') as z:
    assert sha((inputs / 'python-3.13.17-embed-amd64.zip').read_bytes()) == '9e35319bffedab8b36ad44d8f377cb4f1733bcec257a0b6d04d94f01486ddb38'
    for name in z.namelist():
        if name.startswith('vcruntime') and name.endswith('.dll'):
            add(name, z.read(name))
        # CPython's current source licence is included below.
cpython = inputs / 'Python-3.13.18'
assert sha((inputs / 'Python-3.13.18.tar.xz').read_bytes()) == 'abb0102e6d4ae027509bcb82327d9dd85515b44a3ba8ad538cfc5e9847029f68'
add('licences/Python-LICENSE.txt', (cpython / 'LICENSE').read_bytes())
for name, rel in {'OpenSSL-LICENSE.txt':'openssl-bin-3.5.9/amd64/LICENSE.txt','libffi-LICENSE.txt':'libffi-3.4.4/LICENSE','bzip2-LICENSE.txt':'bzip2-1.0.8/LICENSE','zlib-LICENSE.txt':'zlib-1.3.1/LICENSE'}.items():
    add('licences/'+name,(cpython/'externals'/rel).read_bytes())
for p in sorted((cpython / 'PCbuild/amd64').iterdir()):
    if p.name.startswith('_test') or p.name.startswith('_ctypes_test') or p.name=='pyshellext.dll': continue
    if p.suffix in ['.pyd', '.dll'] or p.name == 'python.exe':
        if p.name.startswith('vcruntime') and p.name in files: del files[p.name]
        add(p.name, p.read_bytes())
assert 'python.exe' in files and 'python313.dll' in files
stdlib = inputs / 'python313-patched.zip'
with zipfile.ZipFile(stdlib, 'w', zipfile.ZIP_DEFLATED, compresslevel=6) as z:
    for p in sorted((cpython / 'Lib').rglob('*.py')):
        rel = p.relative_to(cpython / 'Lib')
        if any(v in ['test', 'tests', '__pycache__', 'idlelib', 'tkinter', 'turtledemo', 'ensurepip'] for v in rel.parts): continue
        info=zipfile.ZipInfo(str(rel).replace('\\', '/'),date_time=(2026,10,25,0,0,0));info.compress_type=zipfile.ZIP_DEFLATED;z.writestr(info,p.read_bytes())
add('python313.zip', stdlib.read_bytes())
add('python313._pth', b'python313.zip\npackages.zip\n.\nLib/site-packages\nimport site\n')
pure = {}; dependency_patches=[]
def resource_import(text):
    lines=text.splitlines(True)
    at=next(i for i,line in enumerate(lines) if (line.startswith('import ') or line.startswith('from ')) and not line.startswith('from __future__'))
    lines.insert(at,'from importlib.resources import files as _files\n')
    return ''.join(lines)
for wheel in lock['wheels']:
    p = inputs / 'wheels' / wheel['file']
    assert sha(p.read_bytes()) == wheel['sha256'], p.name
    with zipfile.ZipFile(p) as z:
        native=any(n.endswith(('.pyd','.dll')) for n in z.namelist())
        for name in z.namelist():
            if name.endswith('/'): continue
            assert '.data/' not in name, name
            if native and name.endswith(('.h','.pxd','.pyx','.c','.cpp')): continue
            b=z.read(name)
            if name in ['babel/core.py','babel/localedata.py','pytz/__init__.py']:
                old=b.decode('utf-8');new=resource_import(old)
                if name=='babel/core.py':
                    new=new.replace("filename = os.path.join(dirname, 'global.dat')", "filename = _files('babel').joinpath('global.dat')").replace('os.path.isfile(filename)','filename.is_file()').replace("open(filename, 'rb')", "filename.open('rb')")
                if name=='babel/localedata.py':
                    new=new.replace('os.path.exists(resolve_locale_filename(name))',"_files('babel').joinpath('locale-data', str(os.path.basename(name)) + '.dat').is_file()")
                    new=new.replace('os.listdir(_dirname)',"[p.name for p in _files('babel').joinpath('locale-data').iterdir()]")
                    new=new.replace('filename = resolve_locale_filename(name)',"filename = _files('babel').joinpath('locale-data', str(os.path.basename(name)) + '.dat')").replace("open(filename, 'rb')", "filename.open('rb')")
                if name=='pytz/__init__.py':
                    a=new.index('def open_resource(');bpos=new.index('def resource_exists(',a)
                    new=new[:a]+"""def open_resource(name):
    parts = name.lstrip('/').split('/')
    if any(part in ('..', '') or chr(92) in part for part in parts):
        raise ValueError('Invalid timezone resource')
    return _files('pytz').joinpath('zoneinfo', *parts).open('rb')


"""+new[bpos:]
                dependency_patches.extend(difflib.unified_diff(old.splitlines(True),new.splitlines(True),fromfile='a/'+name,tofile='b/'+name));b=new.encode('utf-8')
            if native:add('Lib/site-packages/' + name,b)
            else:
                assert name not in pure;pure[name]=b
pure_zip=io.BytesIO()
with zipfile.ZipFile(pure_zip,'w',zipfile.ZIP_DEFLATED,compresslevel=6) as z:
    directories=set()
    for name in pure:
        parts=name.split('/')[:-1]
        for i in range(1,len(parts)+1):directories.add('/'.join(parts[:i])+'/')
    for name in sorted(directories):
        info=zipfile.ZipInfo(name,date_time=(2026,10,25,0,0,0));info.external_attr=0o40755<<16;z.writestr(info,b'')
    for name,b in sorted(pure.items()):
        info=zipfile.ZipInfo(name,date_time=(2026,10,25,0,0,0));info.compress_type=zipfile.ZIP_DEFLATED;z.writestr(info,b)
add('packages.zip',pure_zip.getvalue())
add('licences/dependency-resource-access.patch',''.join(dependency_patches).encode('utf-8'))
upstream = inputs / 'searxng-upstream.zip'
assert sha(upstream.read_bytes()) == '030e9e28e6cfa5083dd168771c60bbc109500f5b5f68653a0ebb0117f3364590'
patches = []
with zipfile.ZipFile(upstream) as z:
    prefix = z.namelist()[0]
    for name in z.namelist():
        rel = name[len(prefix):]
        if name.endswith('/') or not rel.startswith('searx/'): continue
        b = z.read(name)
        if rel == 'searx/valkeydb.py':
            old = b.decode('utf-8')
            new = old.replace('import pwd', 'try:\n    import pwd\nexcept ImportError:\n    pwd = None  # My AI Windows port: Unix account DB is unavailable.')
            new = new.replace('_pw = pwd.getpwuid(os.getuid())\n        logger.exception("[%s (%s)] can\'t connect valkey DB ...", _pw.pw_name, _pw.pw_uid)', 'logger.exception("Cannot connect to Valkey")')
            b = new.encode('utf-8')
            patches.extend(difflib.unified_diff(old.splitlines(True), new.splitlines(True), fromfile='a/'+rel, tofile='b/'+rel))
        add('Lib/site-packages/' + rel, b)
    add('licences/SearXNG-AGPL-3.0.txt', z.read(prefix + 'LICENSE'))
frozen = b'VERSION_STRING="2026.10.25+54ca1d9cf-myai"\nVERSION_TAG="2026.10.25"\nDOCKER_TAG="2026.10.25"\nGIT_URL="https://github.com/searxng/searxng"\nGIT_BRANCH="master"\n'
add('Lib/site-packages/searx/version_frozen.py', frozen)
for name in ['server.py', 'settings.yml']: add(name, (out / name).read_bytes())
provenance = {'schema':1, 'searxngCommit':'54ca1d9cfb08674f66261d7f351b063558dfcc62', 'searxngSourceSHA256':sha(upstream.read_bytes()), 'python':'3.13.18', 'pythonSourceSHA256':sha((inputs/'Python-3.13.18.tar.xz').read_bytes()), 'sourceBuild':True, 'wheels':lock['wheels']}
add('licences/runtime-provenance.json', json.dumps(provenance,indent=2).encode())
add('licences/searxng-windows.patch', ''.join(patches).encode())
add('licences/server.py', (out/'server.py').read_bytes())
add('licences/settings.yml', (out/'settings.yml').read_bytes())
manifest = {name:sha(b) for name,b in files.items()}
add('runtime-files.json', json.dumps(manifest,sort_keys=True).encode())
with zipfile.ZipFile(out/'payload.zip','w',zipfile.ZIP_DEFLATED,compresslevel=6) as z:
    for name,b in sorted(files.items()):
        info=zipfile.ZipInfo(name,date_time=(2026,10,25,0,0,0));info.compress_type=zipfile.ZIP_DEFLATED;info.external_attr=0o100644<<16;z.writestr(info,b)
(out/'provenance.json').write_text(json.dumps({**provenance,'payloadSHA256':sha((out/'payload.zip').read_bytes()),'files':len(files),'unpackedBytes':sum(map(len,files.values()))},indent=2)+'\n',encoding='utf-8')
(inputs/'searxng-windows.patch').write_text(''.join(patches),encoding='utf-8')
with zipfile.ZipFile(inputs/'my-ai-searxng-corresponding-source.zip','w',zipfile.ZIP_DEFLATED) as z:
    z.write(upstream,'searxng-upstream.zip')
    z.write(inputs/'Python-3.13.18.tar.xz','Python-3.13.18.tar.xz')
    z.writestr('BUILD.md', '''Build CPython3.13.18 from the included official source using PCbuild/pcbuild.proj and MSBuild Release x64, IncludeExternals=true, IncludeTests=false, IncludeTkinter=false. External source versions and acquisition scripts are in CPython PCbuild/get_externals.bat. Place build in Python-3.13.18 next to these sources. Use the verified embed redistribution seed and wheel digests in runtime-provenance.json; save wheels-lock.json with the wheels array, and run build-search-runtime.py against that directory from a My AI source checkout. The My AI release source contains the runtime Go bridge and this exact builder. This separate SearXNG application and server.py are AGPL-3.0-or-later; dependency licences remain in their distributions. No owner data is part of this source.
''')
    z.write(inputs/'searxng-windows.patch','searxng-windows.patch')
    for name in ['server.py','settings.yml']: z.write(out/name,name)
    z.write(Path(__file__),'build-search-runtime.py')
    z.writestr('dependency-resource-access.patch',''.join(dependency_patches))
    for wheel in lock['wheels']: z.write(inputs/'wheels'/wheel['file'],'wheels/'+wheel['file'])
    z.writestr('runtime-provenance.json',json.dumps(provenance,indent=2))
    z.writestr('version_frozen.py',frozen)
print(json.dumps({'files':len(files),'bytes':sum(map(len,files.values())),'payloadBytes':(out/'payload.zip').stat().st_size,'payloadSHA256':sha((out/'payload.zip').read_bytes())}))
