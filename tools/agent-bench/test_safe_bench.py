"""Exercise stop/preflight decisions without loading a model or owner files."""
import io,json,tempfile,unittest
from pathlib import Path
from unittest.mock import patch
import safe_bench as runner
from resources import GiB

class Worker:
    def __init__(self,*a,**kw):self.returncode=None;self.pid=123;self.stopped=False
    def poll(self):return self.returncode
    def terminate(self):self.stopped=True;self.returncode=-15
    def kill(self):self.terminate()
    def wait(self,**kw):return self.returncode

class SupervisorTests(unittest.TestCase):
    def test_atomic_status_save_retries_a_windows_reader_lock(self):
        with tempfile.TemporaryDirectory() as directory:
            path=Path(directory)/'status.json';path.write_text('{}')
            replace=Path.replace;calls=[]
            def blocked_once(source,target):
                calls.append(1)
                if len(calls)==1:raise PermissionError('Synthetic Windows reader lock')
                return replace(source,target)
            with patch.object(Path,'replace',blocked_once),patch.object(runner.time,'sleep'):
                runner.save(path,{'status':'completed'})
            self.assertEqual(json.loads(path.read_text()),{'status':'completed'});self.assertEqual(len(calls),2)
    def run_guard(self,low=False):
        class Probe:
            def __init__(self,*a):self.count=0
            def sample(self):
                self.count+=1
                return dict(ramTotal=32*GiB,ramAvailable=(7 if low else 24)*GiB,cpuPercent=10,gpus=[],healthOK=self.count<=3,healthMS=20)
        def urlopen(req,**kw):
            return io.BytesIO(json.dumps({'models':[{'name':'small:latest','size':GiB}]} if isinstance(req,str) else {}).encode())
        with tempfile.TemporaryDirectory() as directory:
            with patch('sys.argv',['safe_bench','--out',directory,'--models','small:latest']),patch.object(runner,'Probe',Probe),patch.object(runner.urllib.request,'urlopen',side_effect=urlopen),patch.object(runner.time,'sleep'),patch.object(runner.shutil,'disk_usage',return_value=type('Disk',(),{'free':10*GiB})()),patch.object(runner.subprocess,'Popen',side_effect=Worker) as popen,patch.object(runner.os,'setpriority',create=True):
                runner.main()
                return json.loads((Path(directory)/'resource-results.json').read_text()),popen
    def test_low_headroom_never_starts_inference(self):
        result,popen=self.run_guard(low=True)
        popen.assert_not_called();self.assertEqual(result['models'][0]['state'],'skipped');self.assertEqual(result['recommended'],'')
    def test_service_health_failure_stops_worker_and_prevents_recommendation(self):
        result,popen=self.run_guard()
        self.assertEqual(popen.call_count,1);model=result['models'][0]
        self.assertEqual(model['state'],'stopped');self.assertFalse(model['resourceSafe']);self.assertIn('health',model['reason']);self.assertEqual(result['recommended'],'')
        self.assertEqual(model['monitorSamples'],2)
if __name__=='__main__':unittest.main()
