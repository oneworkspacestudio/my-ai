"""Native AI Service acceptance using synthetic NTFS data and an isolated Ollama protocol server."""
from pathlib import Path
import argparse,concurrent.futures,hashlib,json,os,secrets,socket,subprocess,threading,time
from http.server import BaseHTTPRequestHandler,ThreadingHTTPServer
import requests

def main():
 p=argparse.ArgumentParser();p.add_argument('--exe',required=True,type=Path);p.add_argument('--root',required=True,type=Path);p.add_argument('--out',required=True,type=Path);args=p.parse_args()
 assert not args.root.exists();args.root.mkdir(parents=True)
 state={'loaded':False,'active':0,'maxActive':0,'calls':0,'pins':0,'unloads':0};lock=threading.Lock()
 class Ollama(BaseHTTPRequestHandler):
  def log_message(self,*args):pass
  def respond(self,body):
   b=json.dumps(body).encode();self.send_response(200);self.send_header('Content-Type','application/json');self.send_header('Content-Length',str(len(b)));self.end_headers();self.wfile.write(b)
  def do_GET(self):
   if self.path=='/api/tags':self.respond({'models':[{'name':'fixture:9b','size':9000}]})
   elif self.path=='/api/ps':self.respond({'models':[{'name':'fixture:9b'}] if state['loaded'] else []})
   else:self.send_error(404)
  def do_POST(self):
   body=json.loads(self.rfile.read(int(self.headers['Content-Length'])))
   if self.path=='/api/generate':
    with lock:
     state['loaded']=body['keep_alive']==-1;state['pins' if state['loaded'] else 'unloads']+=1
    self.respond({'done':True});return
   assert self.path=='/api/chat' and body['model']=='fixture:9b' and body['keep_alive']==-1
   with lock:state['active']+=1;state['calls']+=1;state['maxActive']=max(state['maxActive'],state['active'])
   try:
    time.sleep(1.3)
    self.respond({'message':{'role':'assistant','content':'Synthetic reply'},'done':True})
   except (BrokenPipeError,ConnectionResetError,ConnectionAbortedError):pass
   finally:
    with lock:state['active']-=1
 backend=ThreadingHTTPServer(('127.0.0.1',0),Ollama);threading.Thread(target=backend.serve_forever,daemon=True).start()
 sock=socket.socket();sock.bind(('127.0.0.1',0));port=sock.getsockname()[1];sock.close();origin='http://127.0.0.1:'+str(port)
 password=secrets.token_urlsafe(28);email='service-owner@example.test';token=secrets.token_hex(32)
 env={k:v for k,v in os.environ.items() if not k.startswith(('WORKSPACE_','MYAI_','AISERVICE_','GH_','GITHUB_'))};env.update(WORKSPACE_BOOTSTRAP_PASSWORD=password,AISERVICE_OLLAMA_URL='http://127.0.0.1:'+str(backend.server_port),AISERVICE_MYAI_TOKEN=token)
 proc=None;checks=[]
 def start(bootstrap):
  nonlocal proc
  cmd=[str(args.exe),'--data',str(args.root/'Private'),'--listen','127.0.0.1:'+str(port),'--origin',origin,'--no-browser']
  if bootstrap:cmd+=['--bootstrap-email',email]
  with (args.root/'native.log').open('ab') as log:proc=subprocess.Popen(cmd,env=env,stdout=log,stderr=log,creationflags=subprocess.CREATE_NO_WINDOW)
  for _ in range(120):
   assert proc.poll() is None,'Native fixture exited'
   try:
    if requests.get(origin+'/healthz',timeout=.5).status_code==200:return
   except requests.RequestException:pass
   time.sleep(.1)
  raise RuntimeError('Startup timeout')
 def stop():
  nonlocal proc
  if proc and proc.poll() is None:proc.terminate();proc.wait(timeout=15)
  proc=None
 def login(mail):
  s=requests.Session();s.trust_env=False
  assert s.post(origin+'/api/login',json={'email':mail,'password':password},headers={'Origin':origin}).status_code==200
  me=s.get(origin+'/api/me').json();s.headers.update({'Origin':origin,'X-CSRF-Token':me['csrf'],'X-Workspace-Expected-Account':me['user']['id']});return s
 def wait_for(fn):
  for _ in range(120):
   if fn():return
   time.sleep(.1)
  raise AssertionError('Expected state was not reached')
 def app_request(key,app='myai',user='fixture-user'):
  return requests.post(origin+'/_internal/chat',headers={'Authorization':'Bearer '+key,'X-AI-App':app,'X-AI-User':user},json={'model':'fixture:9b','messages':[{'role':'user','content':'Synthetic request'}],'stream':True},timeout=30)
 try:
  start(True);owner=login(email)
  assert owner.get(origin+'/api/bootstrap').json()['id']=='aiservice'
  assert not owner.get(origin+'/api/service').json()['settings']['enabled'];assert state['pins']==0
  assert requests.get(origin+'/api/service').status_code==401
  assert app_request('invalid').status_code==401
  assert app_request(token).status_code==503
  assert owner.post(origin+'/api/service/settings',json={'enabled':True,'model':'fixture:9b'},headers={'X-CSRF-Token':'bad'}).status_code==403
  assert owner.post(origin+'/api/local-users',json={'email':'member@example.test','password':password}).status_code==201
  member=login('member@example.test');assert member.post(origin+'/api/service/settings',json={'enabled':True,'model':'fixture:9b'}).status_code==403
  assert member.post(origin+'/api/service/clients',json={'id':'other'}).status_code==403
  assert owner.post(origin+'/api/service/settings',json={'enabled':True,'model':'missing:9b'}).status_code==400
  assert owner.post(origin+'/api/service/settings',json={'enabled':True,'model':'fixture:9b'}).status_code==200
  wait_for(lambda:state['loaded']);assert state['pins']==1;checks+=['native fresh disabled','authentication, owner and CSRF boundaries','installed model selection and residency']
  with concurrent.futures.ThreadPoolExecutor(max_workers=3) as ex:
   jobs=[ex.submit(app_request,token,'myai',user) for user in ['alice','bob','carol']]
   wait_for(lambda:owner.get(origin+'/api/service').json()['queued']>=2)
   assert all(f.result().status_code==200 for f in jobs)
  assert state['maxActive']==1;checks.append('multiple delegated users share one serial queue')
  key=owner.post(origin+'/api/service/clients',json={'id':'other-app'}).json()['key']
  assert 'key' not in owner.get(origin+'/api/service/clients').text
  with concurrent.futures.ThreadPoolExecutor(max_workers=2) as ex:
   one=ex.submit(app_request,token);wait_for(lambda:state['active']==1)
   two=ex.submit(app_request,key,'other-app','member');wait_for(lambda:owner.get(origin+'/api/service').json()['queued']==1)
   assert owner.delete(origin+'/api/service/clients/other-app').status_code==200
   assert two.result().status_code==408;assert one.result().status_code==200
  assert app_request(key,'other-app').status_code==401;checks.append('per-app credentials, multiple apps and revocation')
  with concurrent.futures.ThreadPoolExecutor(max_workers=2) as ex:
   one=ex.submit(app_request,token);wait_for(lambda:state['active']==1)
   two=ex.submit(lambda:member.post(origin+'/api/chat',json={'messages':[]},timeout=30));wait_for(lambda:owner.get(origin+'/api/service').json()['queued']==1)
   assert member.post(origin+'/auth/logout',json={}).status_code==200
   assert two.result().status_code==408;assert one.result().status_code==200
  checks.append('exact native logout cancels queued browser inference')
  stop();start(False);owner=login(email);assert owner.get(origin+'/api/service').json()['settings']=={'enabled':True,'model':'fixture:9b'};checks.append('restart preserves owner model and activation')
  assert owner.post(origin+'/api/service/settings',json={'enabled':False,'model':'fixture:9b'}).status_code==200
  wait_for(lambda:not state['loaded']);assert app_request(token).status_code==503;checks.append('turn off unloads selected model and rejects inference')
  args.out.write_text(json.dumps({'schema':1,'passed':True,'binarySHA256':hashlib.sha256(args.exe.read_bytes()).hexdigest(),'checks':checks,'maxConcurrentModelRequests':state['maxActive'],'ownerUsedAsFixture':False,'realModelAccuracyTested':False},indent=2)+'\n')
  print('Native AI Service acceptance passed:',len(checks),'checks')
 finally:stop();backend.shutdown();backend.server_close()
if __name__=='__main__':main()
