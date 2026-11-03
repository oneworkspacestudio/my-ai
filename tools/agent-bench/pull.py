import json, urllib.request, sys, time
base = 'http://127.0.0.1:11434'
for model in sys.argv[1:]:
    print('Pulling ' + model, flush=True)
    try:
        req = urllib.request.Request(base+'/api/pull', json.dumps({'model':model,'stream':True}).encode(), {'Content-Type':'application/json'})
        last = ''; started=time.monotonic(); reported=started
        with urllib.request.urlopen(req, timeout=600) as r:
            for line in r:
                event = json.loads(line)
                if event.get('error'): raise RuntimeError(event['error'])
                status=event.get('status','')
                if time.monotonic()-started>1800:raise TimeoutError('1800 second download budget; partial bytes retained for resume')
                if status != last:
                    print(model+': '+status, flush=True); last=status
                if time.monotonic()-reported>30:
                    print(model+': '+str(event.get('completed',0))+'/'+str(event.get('total',0))+' bytes',flush=True);reported=time.monotonic()
        print('Ready '+model, flush=True)
    except Exception as e: print('Unavailable '+model+': '+str(e), flush=True)
