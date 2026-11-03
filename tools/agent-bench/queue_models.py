"""Wait for explicitly requested downloads and benchmark available models serially."""
import argparse, json, subprocess, sys, time, urllib.request
from pathlib import Path

p=argparse.ArgumentParser();p.add_argument('--out',type=Path,required=True);p.add_argument('--wait-seconds',type=int,default=900);p.add_argument('models',nargs='+');a=p.parse_args()
pending=list(a.models);deadline=time.monotonic()+a.wait_seconds
while pending and time.monotonic()<deadline:
    try:
        with urllib.request.urlopen('http://127.0.0.1:11434/api/tags',timeout=20) as r:installed={m['name'] for m in json.load(r)['models']}
    except Exception as e:
        print('Model availability check: '+str(e),flush=True);time.sleep(10);continue
    ready=next((m for m in pending if m in installed),None)
    if ready:
        print('Benchmarking downloaded model '+ready,flush=True)
        subprocess.run([sys.executable,str(Path(__file__).with_name('bench.py')),'--out',str(a.out),'--models',ready],check=True)
        pending.remove(ready)
    else:time.sleep(10)
status={'notBenchmarked':[{'model':m,'reason':'Official registry download did not finish within the additional '+str(a.wait_seconds)+' second queue window; no estimated score.'} for m in pending]}
(a.out/'availability.json').write_text(json.dumps(status,indent=2)+'\n',encoding='utf-8')
print(json.dumps(status),flush=True)
