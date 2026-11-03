import React from 'react';
import {createRoot} from 'react-dom/client';
import {Benchmark} from './generated';
import raw from './summary.json';
import samples from './samples.json';
import './style.css';
let order='score',failures=false,page=0;
const h=React.createElement;
const sources=[{title:'Ollama: native tool-calling loop',url:'https://docs.ollama.com/capabilities/tool-calling'},
 {title:'Qwen3.5 official model card',url:'https://huggingface.co/Qwen/Qwen3.5-4B'},
 {title:'Ministral 3 3B official model card',url:'https://huggingface.co/mistralai/Ministral-3-3B-Instruct-2512'},
 {title:'IBM Granite 4 official model card',url:'https://huggingface.co/ibm-granite/granite-4.0-micro'}];
const root=createRoot(document.getElementById('root')!);
function render(){
 const models=[...raw.models].sort((a,b)=>order==='size'?a.sizeGB-b.sizeGB:order==='speed'?a.medianSeconds-b.medianSeconds:b.rate-a.rate||a.sizeGB-b.sizeGB);
 const filtered=samples.filter(s=>!failures||!s.pass);
 const shown=filtered.slice(page*24,(page+1)*24).map((s,i)=>({title:`${s.pass?'PASS':'FAIL'} · ${s.model} · ${s.case.replaceAll('_',' ')} · seed ${s.seed}`,status:`${s.seconds}s · ${s.toolCalls} tool calls · ${s.safetyPass?'No guardrail violations':s.violations.join(', ')}${s.error?' · '+s.error:''}`,finalText:s.final||'(No final answer within budget.)',traceLink:'samples.json'}));
 const gpu=raw.run.gpu.split(',');
 const hardwareLabel=gpu[0].replace('NVIDIA GeForce ','')+(gpu[1]?` · ${Math.round(parseFloat(gpu[1])/1024)} GB`:'');
 const configurationLabel=`${raw.run.cases.length} scenarios × ${raw.run.seeds.length} seeds per model. Identical prompts and tool contracts for every candidate. Temperature ${raw.run.temperature}; thinking ${raw.run.thinking?'on':'off'}; ${raw.run.context.toLocaleString()} context tokens; ${raw.run.maxOutputTokensPerRound} output tokens per round; up to ${raw.run.maxRounds} rounds and ${raw.run.sampleBudgetSeconds} seconds per task. Sequential model inference.`;
 const data={...raw,models,samples:shown,date:raw.run.startedUTC.slice(0,10),modelCount:raw.models.length,ollama:raw.run.ollama.version,hardwareLabel,ramLabel:raw.run.ramGB?raw.run.ramGB+' GB':'Unknown',seedLabel:raw.run.seeds.join(', '),configurationLabel,noPrevious:page===0,noNext:(page+1)*24>=filtered.length,sampleFilter:`Showing ${filtered.length?page*24+1:0}–${page*24+shown.length} of ${filtered.length} ${failures?'failed':'total'} samples. Expand a row to read the final answer; open JSON for every tool call.`,sources};
 root.render(h(Benchmark,{data,onAction:(action:string)=>{if(['score','size','speed'].includes(action))order=action;else if(action==='next')page++;else if(action==='previous')page--;else{failures=action==='failures';page=0}render();}}));
}
render();
