"""Replay saved synthetic receipts; verify configuration and recorded grading."""
import argparse, collections, hashlib, json
from pathlib import Path
from bench import CASES, SYSTEM, VERSION, World

p=argparse.ArgumentParser();p.add_argument('root',type=Path);a=p.parse_args()
meta=json.loads((a.root/'run.json').read_text(encoding='utf-8'))
rows=[json.loads(l) for l in (a.root/'samples.jsonl').read_text(encoding='utf-8').splitlines()]
assert meta['benchmarkVersion']==VERSION and meta['maxRounds']==16
seen=set();counts=collections.Counter()
for r in rows:
    key=(r['model'],r['case'],r['seed']);assert key not in seen;seen.add(key);counts[r['model']]+=1
    w=World(r['case'],r['seed'],meta['startedUTC']);t=r['transcript']
    assert t[:2]==[{'role':'system','content':SYSTEM},{'role':'user','content':w.prompt()}],key
    expected=collections.deque()
    for m in t[2:]:
        if m['role']=='assistant':
            for call in m.get('tool_calls',[]):
                f=call['function'];args=f.get('arguments',{})
                if isinstance(args,str):args=json.loads(args)
                expected.append((f['name'],w.execute(f['name'],args)))
        elif m['role']=='tool':
            name,receipt=expected.popleft()
            assert m['tool_name']==name and json.loads(m['content'])==receipt,key
        else:raise AssertionError('Unexpected role')
    assert not expected,key
    score=w.score(r['final'])
    if r['error']:score['pass']=False
    assert all(r[k]==v for k,v in score.items()),key
    assert r['seconds']>=0 and r['model'] in meta['models'],key
expected=len(meta['cases'])*len(meta['seeds'])
assert all(n==expected for n in counts.values()),dict(counts)
receipt={'schema':1,'benchmarkVersion':VERSION,'samples':len(rows),'models':dict(counts),'fullSyntheticReceiptReplay':True,'systemAndUserPromptsMatchSource':True,'recordedGradesMatchReplay':True,'duplicateSamples':False,'samplesSHA256':hashlib.sha256((a.root/'samples.jsonl').read_bytes()).hexdigest()}
(a.root/'verification.json').write_text(json.dumps(receipt,indent=2)+'\n',encoding='utf-8')
print(json.dumps(receipt,indent=2))
