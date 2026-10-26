const phases:Record<string,string>={planning:'Planning the search…',reading:'Reading sources…',checking:'Checking sources…',writing:'Writing an answer…',verifying:'Checking the answer…'};

export function researchWorkingLabel(events:any[]):string {
 const last=events.at(-1);
 if(last?.method==='research/progress')return phases[last.data?.phase]||'Researching…';
 if(last?.method==='tool/start'&&last.data?.name==='web_search')return 'Searching the web…';
 if(last?.method==='tool/start'&&last.data?.name==='read_webpage')return 'Reading a source…';
 return events.some(v=>v.method==='tool/result'&&v.data?.name==='web_search')?'Writing an answer…':'Working…';
}

export function researchActivity(v:any):{method:string;content:string} {
 if(v.method==='research/progress')return {method:phases[v.data?.phase]||'Researching…',content:({planning:'Choosing useful queries for your question.',reading:'Reading the source material.',checking:'Checking relevance, dates and supporting evidence.',writing:'Preparing an answer from the sources.',verifying:'Checking factual claims and source links.'} as Record<string,string>)[v.data?.phase]||''};
 if(v.data?.name==='web_search')return {method:v.method==='tool/start'?'Searching the web':v.data?.failed?'Search could not finish':'Web search complete',content:v.method==='tool/start'?'Looking for relevant information.':v.data?.failed?String(v.data?.output?.error||'Please try again.'):'Found '+(v.data?.output?.sources?.length||0)+' sources; reading available pages.'};
 if(v.data?.name==='read_webpage')return {method:v.method==='tool/start'?'Reading a source':v.data?.failed?'Source unavailable':'Source read',content:v.data?.failed?String(v.data?.output?.error||'Trying other sources.'):v.method==='tool/start'?'Opening a public source.':'Source material is available for the answer.'};
 return {method:v.method,content:JSON.stringify(v.data,null,2).slice(0,6000)};
}
