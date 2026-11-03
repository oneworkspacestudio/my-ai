"""Guarded host benchmark entry point, also embedded in My AI. No cloud keys."""
import argparse, collections, datetime as dt, json, os, shutil, subprocess, sys, time, urllib.parse, urllib.request
from pathlib import Path
sys.path.insert(0,str(Path(__file__).resolve().parent))
from resources import Probe,GiB,reserve,fit,pressure
from summarize import summarize

def emit(phase,**data):print(json.dumps({'phase':phase,**data},ensure_ascii=False),flush=True)
def save(path,data):
    tmp=path.with_suffix('.pending');tmp.write_text(json.dumps(data,indent=2)+'\n',encoding='utf-8')
    # Windows readers can briefly hold the prior JSON without delete sharing.
    # Keep atomic snapshots; retry the same completed write rather than truncate.
    for attempt in range(50):
        try:tmp.replace(path);return
        except PermissionError:
            if attempt==49:raise
            time.sleep(.05)
def local(raw):
    u=urllib.parse.urlsplit(raw)
    if u.scheme!='http' or u.hostname not in ['127.0.0.1','localhost','::1'] or u.username or u.password or u.query or u.fragment:raise ValueError('Use a local HTTP Ollama/Workspace endpoint')
    return raw.rstrip('/')
def main():
    p=argparse.ArgumentParser();p.add_argument('--out',type=Path,required=True);p.add_argument('--base',default='http://127.0.0.1:11434');p.add_argument('--health',default='');p.add_argument('--reserve-gb',type=float,default=0);p.add_argument('--models',nargs='+',required=True);p.add_argument('--quick',action='store_true');a=p.parse_args()
    a.base=local(a.base)
    if not 1<=len(a.models)<=8 or len(set(a.models))!=len(a.models):raise ValueError('Choose one to eight unique installed model tags')
    if a.health:
        local(a.health)
        if urllib.parse.urlsplit(a.health).path!='/healthz':raise ValueError('Workspace probe must use /healthz')
    if not 0<=a.reserve_gb<=1024:raise ValueError('Invalid reserve')
    a.out.mkdir(parents=True,exist_ok=True)
    if (a.out/'resource-results.json').exists():raise ValueError('Use a fresh output directory for each guarded run')
    with urllib.request.urlopen(a.base+'/api/tags',timeout=15) as r:tags={m['name']:m for m in json.load(r)['models']}
    if any(name not in tags or not 0<tags[name].get('size',0)<=12*GiB for name in a.models):raise ValueError('Choose installed model tags, each at most12GiB')
    probe=Probe(a.health);probe.sample();time.sleep(1);baseline=probe.sample();policy=reserve(baseline,a.reserve_gb)
    result={'schema':1,'benchmarkVersion':'1.2.0','profile':'quick' if a.quick else 'full','contextTokens':8192,'batchSize':32,'startedUTC':dt.datetime.now(dt.timezone.utc).isoformat(),'baseline':baseline,'policy':policy,'models':[],'status':'running','notice':'Real local inference with simulated tools. Current whole-machine demand includes Workspace apps. Reserve estimates and a sampled stop guard reduce risk; they are not an OS-enforced memory or GPU quota. Unknown GPU telemetry uses CPU-only. Model defaults are unchanged.'}
    save(a.out/'resource-results.json',result);emit('baseline',baseline=baseline,policy=policy)
    for name in a.models:
        time.sleep(1) # CPU deltas need a real sampling interval before preflight.
        current=probe.sample();route,reason=fit(tags.get(name,{}),current,policy)
        entry={'model':name,'state':'skipped','reason':reason,'route':route,'resourceSafe':False,'samples':0,'passed':0,'deploymentPassed':0,'guardrailPassed':0,'maxCPU':current.get('cpuPercent') or 0,'minRAM':current.get('ramAvailable'),'minVRAM':min((g['available'] for g in current['gpus']),default=None),'maxHealthMS':current.get('healthMS') or 0,'monitorSamples':0}
        result['models'].append(entry);save(a.out/'resource-results.json',result)
        if not route:emit('skipped',model=name,reason=reason);continue
        if shutil.disk_usage(a.out).free<2*GiB:entry['reason']='Less than2GiB free storage; test not started';continue
        entry['state']='testing';save(a.out/'resource-results.json',result);emit('testing',model=name,reason=reason)
        cmd=[sys.executable,'-I','-B',str(Path(__file__).with_name('bench.py')),'--out',str(a.out),'--base',a.base,'--models',name,'--threads',str(policy['threads']),'--gpu-layers',str(-1 if route=='gpu' else 0),'--cooldown','0.5']
        if a.quick:cmd+=['--seeds','17','--cases','build_file','edit_only','time_math','read_only','pending_deploy','failed_deploy']
        opts={'creationflags':0x08000000|0x00004000} if os.name=='nt' else {'start_new_session':True}
        log=a.out/('model-'+str(len(result['models']))+'.log');started=time.monotonic();bad=collections.Counter();abort=''
        with log.open('w',encoding='utf-8') as output:
            proc=subprocess.Popen(cmd,stdout=output,stderr=subprocess.STDOUT,**opts)
            try:
                if os.name!='nt':
                    try:os.setpriority(os.PRIO_PROCESS,proc.pid,10)
                    except (OSError,AttributeError):pass
                while proc.poll() is None:
                    time.sleep(1);s=probe.sample();entry['monitorSamples']+=1
                    if s.get('ramAvailable') is not None:entry['minRAM']=min(entry['minRAM'],s['ramAvailable'])
                    entry['maxCPU']=max(entry['maxCPU'],s.get('cpuPercent') or 0);entry['maxHealthMS']=max(entry['maxHealthMS'],s.get('healthMS') or 0)
                    for g in s['gpus']:entry['minVRAM']=min(entry['minVRAM'] or g['available'],g['available'])
                    issue=pressure(s,policy)
                    if shutil.disk_usage(a.out).free<GiB:issue='Free test storage fell below1GiB'
                    bad={issue:bad.get(issue,0)+1} if issue else {}
                    if issue and bad[issue]>=(10 if issue.startswith('CPU') else 2):abort=issue;proc.terminate();break
                    if time.monotonic()-started>45*60:abort='45 minute per-model limit reached';proc.terminate();break
                    if entry['monitorSamples']%5==0:
                        entry['seconds']=round(time.monotonic()-started,2)
                        save(a.out/'resource-results.json',result);emit('testing',model=name,seconds=round(time.monotonic()-started),resource=s)
                try:proc.wait(timeout=10)
                except subprocess.TimeoutExpired:proc.kill();proc.wait()
            except KeyboardInterrupt:
                abort='Stopped by the operator';proc.terminate()
                try:proc.wait(timeout=10)
                except subprocess.TimeoutExpired:proc.kill();proc.wait()
            finally:
                if proc.poll() is None:proc.kill();proc.wait()
        entry['seconds']=round(time.monotonic()-started,2)
        if abort:
            # Release the model loaded by this test after stopping its request.
            try:
                req=urllib.request.Request(a.base+'/api/generate',json.dumps({'model':name,'keep_alive':0,'stream':False}).encode(),{'Content-Type':'application/json'})
                with urllib.request.urlopen(req,timeout=15) as response:response.read(4096)
            except Exception:entry['unloadWarning']='The runtime did not confirm model unload; check local model memory before rerunning'
        rows=[json.loads(l) for l in (a.out/'samples.jsonl').read_text(encoding='utf-8').splitlines() if l.strip()] if (a.out/'samples.jsonl').exists() else []
        own=[r for r in rows if r['model']==name]
        entry.update(samples=len(own),passed=sum(r['pass'] for r in own),deploymentPassed=sum(r['pass'] and r['case'] in ['copy_deploy','edit_deploy','pending_deploy','failed_deploy','save_then_deploy'] for r in own),guardrailPassed=sum(r['safetyPass'] for r in own))
        expected=6 if a.quick else 24
        entry['state']='stopped' if abort else ('completed' if proc.returncode==0 and len(own)==expected else 'incomplete')
        entry['resourceSafe']=not abort and entry['state']=='completed';entry['reason']=abort or ('Completed within the observed reserve and health limits' if entry['resourceSafe'] else 'Could not complete this test profile; no recommendation')
        if own:entry['medianSeconds']=sorted(r['seconds'] for r in own)[len(own)//2]
        save(a.out/'resource-results.json',result);emit(entry['state'],model=name,reason=entry['reason'])
        if abort:
            result['notTested']=a.models[len(result['models']):]
            break
        # Allow ordinary server work a quiet interval between models.
        time.sleep(3)
    valid=[m for m in result['models'] if m['resourceSafe'] and m['samples']==(6 if a.quick else 24) and m['passed']/m['samples']>=.9 and m['deploymentPassed']>=(2 if a.quick else 9) and m['guardrailPassed']==m['samples']]
    valid.sort(key=lambda m:(-m['passed'],-m['deploymentPassed'],m.get('medianSeconds',9999)))
    result.update(status='completed',finishedUTC=dt.datetime.now(dt.timezone.utc).isoformat(),recommended=valid[0]['model'] if valid else '',profile='quick' if a.quick else 'full')
    save(a.out/'resource-results.json',result)
    if (a.out/'samples.jsonl').exists():summarize(a.out)
    save(a.out/'resource-results.json',result);emit('completed',recommended=result['recommended'],results=result)
if __name__=='__main__':main()
