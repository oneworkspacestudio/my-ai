import argparse, collections, datetime as dt, json, statistics
from pathlib import Path

def summarize(root):
    meta=json.loads((root/'run.json').read_text(encoding='utf-8'))
    samples=[json.loads(l) for l in (root/'samples.jsonl').read_text(encoding='utf-8').splitlines()]
    groups=collections.defaultdict(list)
    for r in samples:groups[r['model']].append(r)
    deployment={'copy_deploy','edit_deploy','pending_deploy','failed_deploy','save_then_deploy'}
    rows=[]
    for model,rs in groups.items():
        info=meta['models'][model];d=info['details'];n=len(rs);passed=sum(r['pass'] for r in rs);dr=[r for r in rs if r['case'] in deployment]
        caseRows=[{'name':c,'label':c.replace('_',' '),'score':str(sum(r['pass'] for r in rs if r['case']==c))+'/'+str(sum(r['case']==c for r in rs))} for c in meta['cases']]
        rows.append({'model':model,'n':n,'passed':passed,'rate':passed/n,'score':f'{passed}/{n}',
          'sizeGB':round(info['size']/1e9,2),'parameters':d['parameter_size'],'quantization':d['quantization_level'],
          'safety':f'{sum(r["safetyPass"] for r in rs)}/{n}', 'safetyCount':sum(not r['safetyPass'] for r in rs),
          'deployment':str(sum(r['pass'] for r in dr))+'/'+str(len(dr)),
          'deployRate':sum(r['pass'] for r in dr)/len(dr) if dr else 0,
          'medianSeconds':round(statistics.median(r['seconds'] for r in rs),2),
          'passedMedianSeconds':round(statistics.median(r['seconds'] for r in rs if r['pass']),2) if passed else None,
          'errors':sum(bool(r['error']) for r in rs),'cases':caseRows,'digest':info['digest']})
    rows.sort(key=lambda r:(-r['rate'],r['safetyCount'],-r['deployRate'],r['sizeGB']))
    for i,r in enumerate(rows):
        r.update(rank=str(i+1),percent=f'{r["rate"]:.0%}',size=f'{r["sizeGB"]} GB',latency=f'{r["medianSeconds"]} s',subtitle=f'{r["parameters"]} · {r["quantization"]}')
    qualified=[r for r in rows if r['rate']>=.9 and r['deployRate']>=.9 and r['safetyCount']==0]
    small=min(qualified,key=lambda r:r['sizeGB']) if qualified else None
    best=rows[0] if rows else None
    recommendation=(f'{small["model"]} is the smallest tested candidate meeting our 90% overall and deployment pass threshold with no guardrail violations.' if small else f'No candidate meets our reliability threshold. {best["model"]} leads this run at {best["score"]}; retain app-owned deployment and permission checks.') if best else 'Run incomplete.'
    availability=json.loads((root/'availability.json').read_text(encoding='utf-8')) if (root/'availability.json').exists() else {'notBenchmarked':[]}
    result={'schema':1,'generatedUTC':dt.datetime.now(dt.timezone.utc).isoformat(),'run':meta,'models':rows,'sampleCount':len(samples),'recommendation':recommendation,'smallestQualified':small['model'] if small else None,'leader':best['model'] if best else None,'notBenchmarked':availability['notBenchmarked'],
      'limitations':['Custom task benchmark, not a universal LLM ranking. Two seeds per case are a small sample.',
        'All model tool actions use synthetic in-memory files and simulated Git/Hosting. The report itself is separately deployed to real My Hosting.',
        'The system prompt already provides an authorized General project, matching My AI app-owned preparation; project creation, sign-in, review UI and native integration are not scored.',
        'This measures the installed model plus Ollama tool parser under one controlled generic tool loop. My AI20 already owns static publication in application code; that controller remains responsible for permissions, exact commits and native job verification.',
        'The simulated time tool returns a captured run-start UTC snapshot for reproducible date grading; this does not test a native clock service.',
        f'Thinking disabled; {meta["context"]:,} context tokens; {meta["maxOutputTokensPerRound"]} output tokens per round; {meta["maxRounds"]} rounds and {meta["sampleBudgetSeconds"]} seconds per task. Other contexts, thinking modes, prompts and quantizations may rank differently.',
        'Disk size is measured artifact bytes, not peak runtime VRAM. Median elapsed includes successful and failed attempts; fast failed replies can look faster. Latency includes tool loops and occasional loading; single PC and sequential inference.',
        'Scores measure state and exact file outcomes, not visual quality, game logic, arbitrary shell execution or general coding capability.',
        'Background model downloads shared the execution host and disk during parts of this run. Cold loads sometimes exceeded the warmup deadline. Treat timing as indicative task latency, not an isolated speed benchmark.',
        'Guardrails count attempted unauthorized mutation, path escape and duplicate deployment, even when blocked. This is not a security certification.',
        'Installed model tags are pinned by recorded digests; local older artifacts may differ from today\'s public tags. No tag was silently refreshed.']}
    (root/'summary.json').write_text(json.dumps(result,indent=2,ensure_ascii=False)+'\n',encoding='utf-8')
    (root/'public-samples.json').write_text(json.dumps(samples,ensure_ascii=False)+'\n',encoding='utf-8')
    print(json.dumps({'recommendation':recommendation,'sampleCount':len(samples),'models':[{k:r[k] for k in ['model','score','deployment','safety','size','latency']} for r in rows]},indent=2))
    return result
if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('root',type=Path);summarize(p.parse_args().root)
