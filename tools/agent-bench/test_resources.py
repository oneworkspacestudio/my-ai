import unittest
from resources import GiB,reserve,fit,pressure,workspace_health

class GuardTests(unittest.TestCase):
    def test_native_manager_health_checks_all_running_apps(self):
        self.assertEqual(workspace_health({'status':'ok','apps':{'accounts':'running','myai':'running'}}),(True,2))
        self.assertEqual(workspace_health({'status':'ok','apps':{'accounts':'running','myai':'failed'}}),(False,1))
        self.assertFalse(workspace_health({'status':'ok','apps':{}})[0])
    def host(self):return {'ramTotal':32*GiB,'ramAvailable':24*GiB,'cpuPercent':12,'gpus':[{'total':12*GiB,'available':10*GiB,'utilization':0}],'healthOK':True,'healthMS':15}
    def test_running_apps_reduce_eligible_model_budget(self):
        s=self.host();p=reserve(s);self.assertEqual(p['ram'],8*GiB)
        self.assertEqual(fit({'size':6*GiB},s,p)[0],'gpu')
        s['ramAvailable']=12*GiB
        self.assertIsNone(fit({'size':6*GiB},s,p)[0])
    def test_no_gpu_telemetry_forces_cpu_and_keeps_memory_reserve(self):
        s=self.host();s['gpus']=[];self.assertEqual(fit({'size':2*GiB},s,reserve(s))[0],'cpu')
        s['ramAvailable']=9*GiB;self.assertIsNone(fit({'size':2*GiB},s,reserve(s))[0])
    def test_memory_and_health_pressure_stop_inference(self):
        s=self.host();p=reserve(s);s['ramAvailable']=7*GiB;self.assertIn('RAM',pressure(s,p))
        s=self.host();s['healthOK']=False;self.assertIn('health',pressure(s,p))
        s=self.host();s['gpus'][0]['available']=GiB;self.assertIn('VRAM',pressure(s,p))
    def test_unknown_memory_or_busy_server_does_not_load_model(self):
        s=self.host();s['ramAvailable']=None;self.assertIsNone(fit({'size':GiB},s,reserve(s))[0])
        s=self.host();s['cpuPercent']=80;self.assertIsNone(fit({'size':GiB},s,reserve(s))[0])
    def test_explicit_reserve_cannot_reduce_automatic_reserve(self):
        s=self.host();self.assertEqual(reserve(s,1)['ram'],8*GiB);self.assertEqual(reserve(s,12)['ram'],12*GiB)
    def test_lost_telemetry_stops_a_previously_safe_run(self):
        s=self.host();p=reserve(s);s['gpus']=[];self.assertIn('GPU',pressure(s,p))
        s=self.host();s['cpuPercent']=None;self.assertIn('CPU',pressure(s,p))
if __name__=='__main__':unittest.main()
