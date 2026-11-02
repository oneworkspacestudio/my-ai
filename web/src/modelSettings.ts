export const modelSettingFields=[
 {key:'contextTokens',label:'Context size (tokens)',min:512,max:1048576,step:1,basic:true,help:'Chat context override. Blank uses this model and available server memory. Also applies to Code unless its separate override is set.'},
 {key:'codeContextTokens',label:'Code context size (tokens)',min:512,max:1048576,step:1,basic:true,help:'Code aims for 65,536 tokens automatically when the model and memory allow it. Set an exact size here, or leave blank.'},
 {key:'maxOutputTokens',label:'Maximum reply (tokens)',min:16,max:32768,step:1,basic:true,help:'Automatic reserves reply space inside the selected context. Keep a custom limit below both context sizes.'},
 {key:'temperature',label:'Temperature',min:0,max:2,step:0.05,basic:true,help:'Lower values produce more focused answers. Zero is allowed.'},
 {key:'batchSize',label:'Prompt batch size',min:1,max:2048,step:1,basic:false,help:'Lower this if loading a prompt uses too much memory; it may be slower.'},
 {key:'topP',label:'Top P',min:0,max:1,step:0.05,basic:false,help:'Limits sampling to the most likely part of the probability distribution.'},
 {key:'topK',label:'Top K',min:0,max:200,step:1,basic:false,help:'Limits candidate tokens. Zero disables the limit.'},
 {key:'repeatPenalty',label:'Repetition penalty',min:0,max:2,step:0.05,basic:false,help:'Values above 1 discourage repeated text.'},
 {key:'seed',label:'Random seed',min:0,max:2147483647,step:1,basic:false,help:'Set a number for repeatable sampling. Leave blank for the model default.'},
].map(f=>({...f,name:'llm_'+f.key}));
export function modelSettingsFields(s:Record<string,any>={}){
 const fields:Record<string,string>={llm_chatMode:s.chatMode||'auto',llm_keepAlive:s.keepAlive||''};
 for(const f of modelSettingFields)fields[f.name]=s[f.key]==null?'':String(s[f.key]);return fields;
}
export function parseModelSettings(fields:Record<string,any>){
 const s:Record<string,any>={};
 for(const f of modelSettingFields){const raw=String(fields[f.name]??'').trim();if(!raw)continue;const n=Number(raw);if(!Number.isFinite(n)||n<f.min||n>f.max||(f.step===1&&!Number.isInteger(n)))throw Error(f.label+' must be '+(f.step===1?'a whole number ':'')+'between '+f.min+' and '+f.max);s[f.key]=n;}
 if(s.contextTokens!=null&&s.maxOutputTokens!=null&&s.maxOutputTokens>=s.contextTokens)throw Error('Reply limit must be smaller than context size to leave room for your prompt');
 if(s.codeContextTokens!=null&&s.maxOutputTokens!=null&&s.maxOutputTokens>=s.codeContextTokens)throw Error('Reply limit must be smaller than Code context size');
 if(fields.llm_chatMode==='plain')s.chatMode='plain';if(fields.llm_keepAlive)s.keepAlive=fields.llm_keepAlive;return s;
}
