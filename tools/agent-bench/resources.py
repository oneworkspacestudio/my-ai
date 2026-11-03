"""Read-only, portable resource telemetry; no model or system configuration edits."""
import ctypes, json, os, platform, shutil, subprocess, time, urllib.request
from pathlib import Path
GiB=1024**3

def command(args):
    opts={'creationflags':0x08000000} if os.name=='nt' else {}
    return subprocess.check_output(args,timeout=3,text=True,stderr=subprocess.DEVNULL,**opts)

def workspace_health(data):
    apps=data.get('apps')
    if isinstance(apps,dict) and apps:
        return data.get('status')=='ok' and all(v=='running' for v in apps.values()),sum(v=='running' for v in apps.values())
    return data.get('ok') is True,data.get('running')

class Probe:
    def __init__(self,health=''):
        self.previous=None;self.health=health;self.gpuCache=[];self.gpuAt=0
    def sample(self):
        total=available=None;times=None
        if os.name=='nt':
            class Memory(ctypes.Structure):
                _fields_=[('length',ctypes.c_ulong),('load',ctypes.c_ulong)]+[(k,ctypes.c_ulonglong) for k in ['total','available','totalPage','availablePage','totalVirtual','availableVirtual','extended']]
            m=Memory();m.length=ctypes.sizeof(m)
            if ctypes.windll.kernel32.GlobalMemoryStatusEx(ctypes.byref(m)):total,available=m.total,m.available
            idle,kernel,user=ctypes.c_ulonglong(),ctypes.c_ulonglong(),ctypes.c_ulonglong()
            if ctypes.windll.kernel32.GetSystemTimes(ctypes.byref(idle),ctypes.byref(kernel),ctypes.byref(user)):times=(idle.value,kernel.value+user.value)
        elif Path('/proc/meminfo').exists():
            mem={l.split(':')[0]:int(l.split()[1])*1024 for l in Path('/proc/meminfo').read_text().splitlines()}
            total,available=mem.get('MemTotal'),mem.get('MemAvailable')
            vals=[int(v) for v in Path('/proc/stat').read_text().splitlines()[0].split()[1:]];times=(vals[3]+vals[4],sum(vals[:8]))
        elif platform.system()=='Darwin':
            total=int(command(['/usr/sbin/sysctl','-n','hw.memsize']))
            lines=command(['/usr/bin/vm_stat']).splitlines();page=int(lines[0].split('page size of ')[1].split()[0]);counts={l.split(':')[0]:int(l.split(':')[1].strip().strip('.')) for l in lines[1:] if ':' in l}
            available=page*sum(counts.get(k,0) for k in ['Pages free','Pages inactive','Pages speculative'])
        cpu=None
        if times and self.previous and times[1]>self.previous[1]:cpu=max(0,min(100,100*(1-(times[0]-self.previous[0])/(times[1]-self.previous[1]))))
        self.previous=times
        if platform.system()=='Darwin':
            try:cpu=min(100,sum(float(v) for v in command(['/bin/ps','-A','-o','%cpu=']).split())/max(1,os.cpu_count() or 1))
            except (OSError,ValueError,subprocess.SubprocessError):pass
        if time.monotonic()-self.gpuAt>3:
            self.gpuAt=time.monotonic();self.gpuCache=[]
            candidates=[Path(os.environ.get('SystemRoot','C:/Windows'))/'System32/nvidia-smi.exe',Path(os.environ.get('ProgramFiles','C:/Program Files'))/'NVIDIA Corporation/NVSMI/nvidia-smi.exe'] if os.name=='nt' else [Path('/usr/bin/nvidia-smi'),Path('/usr/local/bin/nvidia-smi')]
            exe=next((p for p in candidates if p.is_file()),None)
            if exe:
                try:
                    for row in command([str(exe),'--query-gpu=memory.total,memory.free,utilization.gpu','--format=csv,noheader,nounits']).splitlines():
                        v=[float(n.strip()) for n in row.split(',')]
                        if len(v)==3:self.gpuCache.append({'total':int(v[0]*1048576),'available':int(v[1]*1048576),'utilization':v[2]})
                except (OSError,ValueError,subprocess.SubprocessError):pass
        healthOK=None;healthMS=None;services=None
        if self.health:
            begin=time.monotonic()
            try:
                with urllib.request.urlopen(self.health,timeout=2) as r:
                    data=json.load(r);healthOK,services=workspace_health(data);healthOK=r.status==200 and healthOK
                healthMS=round((time.monotonic()-begin)*1000)
            except Exception:healthOK=False;healthMS=2000
        return {'at':time.time(),'ramTotal':total,'ramAvailable':available,'cpuPercent':cpu,'gpus':self.gpuCache,'healthOK':healthOK,'healthMS':healthMS,'runningServices':services}

def reserve(sample,requested=0):
    total=sample.get('ramTotal') or 0
    return {'ram':max(4*GiB,int(total*.25),int(requested*GiB)), 'gpu':[max(GiB,int(g['total']*.20)) for g in sample.get('gpus',[])], 'threads':max(1,min(4,(os.cpu_count() or 2)//2))}

def fit(model,snapshot,policy):
    if not snapshot.get('ramTotal') or snapshot.get('ramAvailable') is None:return None,'Memory telemetry unavailable; no load test was started'
    if snapshot.get('cpuPercent') is None:return None,'CPU telemetry unavailable; no load test was started'
    size=model.get('size') or 0
    if size<=0:return None,'Model size unavailable'
    if snapshot['ramAvailable']<policy['ram']+int(size*1.25)+512*1048576:return None,'Not enough free RAM after reserving memory for Workspace and Windows'
    if snapshot.get('cpuPercent') is not None and snapshot['cpuPercent']>65:return None,'The server is busy; rerun when CPU demand is lower'
    if snapshot.get('healthOK') is False:return None,'Workspace is not healthy; no model was loaded'
    gpus=snapshot.get('gpus',[])
    if len(gpus)==1 and gpus[0]['available']-policy['gpu'][0]>=int(size*1.05)+256*1048576:
        if gpus[0]['utilization']>40:return None,'The GPU is busy with other work; rerun later'
        return 'gpu','Fits the current RAM/VRAM reserve estimate; live monitoring still applies'
    if snapshot['ramAvailable']>=policy['ram']+int(size*1.6)+512*1048576:return 'cpu','CPU-only test; GPU telemetry or headroom is insufficient'
    return None,'Not enough safe GPU or CPU headroom with the Workspace reserve'

def pressure(s,policy):
    if s.get('ramAvailable') is None:return 'Memory telemetry became unavailable'
    if s['ramAvailable']<policy['ram']:return 'RAM fell below the Workspace reserve'
    if s.get('healthOK') is False:return 'Workspace health check failed'
    if s.get('healthMS') and s['healthMS']>1500:return 'Workspace became slow to respond'
    if s.get('cpuPercent') is not None and s['cpuPercent']>90:return 'CPU pressure exceeded90%'
    if s.get('cpuPercent') is None:return 'CPU telemetry became unavailable'
    if len(s.get('gpus',[]))!=len(policy['gpu']):return 'GPU telemetry changed or became unavailable'
    for i,g in enumerate(s.get('gpus',[])):
        if i<len(policy['gpu']) and g['available']<policy['gpu'][i]:return 'VRAM fell below the Workspace reserve'
    return ''
