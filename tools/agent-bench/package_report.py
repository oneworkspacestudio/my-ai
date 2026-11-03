import argparse, hashlib, json, re, shutil, subprocess, sys, zipfile
from pathlib import Path
from summarize import summarize

def write_sources(here,output):
    with zipfile.ZipFile(output/'benchmark-source.zip','w',zipfile.ZIP_DEFLATED) as z:
        for source in here.rglob('*'):
            rel=source.relative_to(here)
            if source.is_file() and not any(x in rel.parts for x in ['node_modules','__pycache__','dist']) and source.name not in ['generated.tsx','summary.json','samples.json']:
                z.write(source,rel.as_posix())

def write_manifest(output):
    checks={f.name:hashlib.sha256(f.read_bytes()).hexdigest() for f in output.iterdir() if f.is_file() and f.name!='checksums.json'}
    (output/'checksums.json').write_text(json.dumps(checks,indent=2)+'\n',encoding='utf-8')

def main():
    p=argparse.ArgumentParser();p.add_argument('results',type=Path);p.add_argument('output',type=Path);a=p.parse_args()
    here=Path(__file__).resolve().parent;report=here/'report'
    subprocess.run([sys.executable,str(here/'verify_results.py'),str(a.results)],check=True)
    summary=summarize(a.results)
    expected=len(summary['run']['seeds'])*len(summary['run']['cases'])
    if any(row['n']!=expected for row in summary['models']):raise SystemExit('Refusing to package an incomplete model run')
    shutil.copyfile(a.results/'summary.json',report/'summary.json')
    samples=json.loads((a.results/'public-samples.json').read_text(encoding='utf-8'))
    briefKeys=['model','case','seed','pass','seconds','toolCalls','safetyPass','violations','error','final']
    (report/'samples.json').write_text(json.dumps([{k:r[k] for k in briefKeys} for r in samples],ensure_ascii=False),encoding='utf-8')
    a.output.mkdir(parents=True,exist_ok=True)
    subprocess.run(['node',str(report/'build.mjs')],cwd=report,check=True)
    dist=report/'dist';html=(dist/'index.html').read_text(encoding='utf-8')
    def script(m):
        js=(dist/m.group(1)).read_text(encoding='utf-8')
        js=re.sub(r'</script',lambda close:'<\\/'+close.group(0)[2:],js,flags=re.I)
        return '<script type="module">'+js+'</script>'
    html=re.sub(r'<script type="module"[^>]*src="\./([^"]+)"[^>]*></script>',script,html)
    def style(m):return '<style>'+(dist/m.group(1)).read_text(encoding='utf-8')+'</style>'
    html=re.sub(r'<link rel="stylesheet"[^>]*href="\./([^"]+)"[^>]*>',style,html)
    if re.search(r'(?:src|href)="\./assets/',html):raise SystemExit('Unbundled asset remains')
    (a.output/'index.html').write_text(html,encoding='utf-8')
    shutil.copyfile(a.results/'summary.json',a.output/'summary.json');shutil.copyfile(a.results/'public-samples.json',a.output/'samples.json')
    shutil.copyfile(a.results/'verification.json',a.output/'verification.json')
    shutil.copyfile(here/'THIRD-PARTY-LICENSES.txt',a.output/'THIRD-PARTY-LICENSES.txt')
    write_sources(here,a.output)
    (a.output/'README.md').write_text('# My AI agent benchmark\n\nMeasured task results, runnable synthetic tool benchmark, and AUI report source.\n\nSee the hosted page or summary.json for methodology and limitations. Run instructions are in benchmark-source.zip.\n',encoding='utf-8')
    write_manifest(a.output)
    print('Packaged '+str(a.output))
if __name__=='__main__':main()
