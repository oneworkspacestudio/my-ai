const label=(key:string)=>({listId:'Task list',notebook_id:'Notebook',path:'File',source:'Drive',content:'Text',body:'Note text',data:'Event or contact',commit:'Saved Git revision',environment:'Destination',overwrite:'Replace existing file',requestKey:'Request identifier'} as Record<string,string>)[key]||key.replace(/[_-]/g,' ').replace(/([a-z])([A-Z])/g,'$1 $2').replace(/^./,s=>s.toUpperCase());

// These cards use recorded native results, never an assistant's claim of success.
export function workspaceDeployments(messages:any[],events:any[],apps:any[]):any[] {
 const jobs=new Map<string,any>();
 const webURL=(value:any)=>{try{const u=new URL(String(value));return ['http:','https:'].includes(u.protocol)&&!u.username&&!u.password?u:null}catch{return null}};
 const accept=(output:any,at:string)=>{
  if(output?.app!=='hosting'||!['deploy_project','deployment'].includes(output.operation))return;
  const result=output.result,job=result?.deployment||result;
  if(!job||typeof job.id!=='string'||typeof job.projectId!=='string'||typeof job.state!=='string')return;
  const old=jobs.get(job.id)||{};jobs.set(job.id,{...old,...job,checkedAt:at||old.checkedAt,sourceCommit:result?.sourceCommit||old.sourceCommit});
 };
 for(const m of messages){if(m.role!=='tool'||m.status==='imported'||!['use_workspace_app','workspace_hosting_deploy_project','workspace_hosting_deployment'].includes(m.toolName))continue;try{accept(JSON.parse(m.content),m.at)}catch{}}
 for(const event of events){
  if(event.method==='tool/result'&&!event.data?.failed)accept(event.data?.output,event.at);
  if(event.method==='deployment/progress'){
   const p=event.data;if(!p?.deployment||!p?.site)continue;
   const old=jobs.get(p.deployment)||{};jobs.set(p.deployment,{...old,id:p.deployment,projectId:p.site,state:p.state,url:p.url||old.url,checkedAt:p.checkedAt||event.at||old.checkedAt});
  }
 }
 const hostingURL=webURL(apps.find(a=>a.id==='hosting')?.url)?.href||'';
 return [...jobs.values()].sort((a,b)=>String(b.checkedAt||'').localeCompare(String(a.checkedAt||''))).slice(0,3).map(job=>{
  const url=webURL(job.url),hostOnly=!!url&&(url.hostname==='localhost'||url.hostname.endsWith('.localhost')||/^127(?:\.[0-9]{1,3}){3}$/.test(url.hostname)||url.hostname==='[::1]'),published=job.state==='published';
  const state=String(job.state).replace(/^./,(s:string)=>s.toUpperCase()),checked=job.checkedAt&&Number.isFinite(Date.parse(job.checkedAt))?' · '+new Date(job.checkedAt).toLocaleString():'';
  return {...job,title:job.config?.name||'My Hosting deployment',statusLabel:'Last confirmed: '+state+(job.environment?' · '+job.environment:'')+checked,hostOnly,canOpen:published&&!!url,url:url?.href||'',linkLabel:hostOnly?'Open server preview':'Open deployed site',detail:published?(hostOnly?'This preview opens on the hosting computer. Set a public site address in My Hosting to open it from other devices.':'My Hosting confirmed this deployment is published.'):(job.error||'Open My Hosting for current progress, or ask My AI to check this deployment.'),hostingURL,hasHosting:!!hostingURL};
 });
}

export function workspaceReview(params:any):string {
 const details=params?.details||{},body=details.body||{},lines=[`${params.app} · ${params.title}`];
 for(const key of ['project','repository','site','list'])if(details[key])lines.push(`${label(key)}: ${details[key]}`);
 for(const [key,value]of Object.entries(body)){
  if(key==='listId'&&details.list)continue;
  if(value===null||value===undefined)continue;
  const text=typeof value==='boolean'?(value?'Yes':'No'):typeof value==='object'?JSON.stringify(value,null,2):String(value);
  lines.push(`${label(key)}: ${text}`);
 }
 for(const [key,value]of Object.entries(details.parameters||{}))if(!(['project','site'].includes(key)&&details[key]))lines.push(`${label(key)}: ${value}`);
 return lines.join('\n\n');
}

export function workspaceWorkingLabel(events:any[],apps:any[]):string {
 const last=events.at(-1);
 if(last?.method==='deployment/progress')return 'Deployment · '+String(last.data?.state||'checking')+'…';
 if(last?.method==='tools/retrying')return 'Retrying an incomplete tool response…';
 if(last?.method!=='tool/start')return '';
 if(last.data?.name==='get_workspace_apps')return 'Checking your Workspace apps…';
 if(last.data?.name==='describe_workspace_app')return 'Reading the app’s available actions…';
 if(last.data?.name==='use_workspace_app')return `Using ${apps.find(a=>a.id===last.data.arguments?.app)?.name||'your Workspace app'}…`;
 return '';
}

export function workspaceActivity(event:any,apps:any[]):{method:string;content:string}|null {
 if(event.method==='deployment/progress')return {method:'My Hosting · '+String(event.data?.state||'checking'),content:event.data?.url||'Following the submitted deployment in My Hosting.'};
 if(event.method==='tools/retrying')return {method:'Retrying tool response',content:event.data?.notice||'No incomplete tool call was executed.'};
 const name=event.data?.name;
 if(!['get_workspace_apps','describe_workspace_app','use_workspace_app'].includes(name))return null;
 if(event.method==='tool/start')return {method:workspaceWorkingLabel([event],apps),content:name==='use_workspace_app'?'Checking this request with your app and account. Changes wait for your review.':'Reading the installed app’s registered capabilities.'};
 if(event.data?.failed)return {method:'App request could not finish',content:String(event.data.output?.error||'Open the app to check its current state.')};
 if(name!=='use_workspace_app')return {method:'App actions available',content:'My AI has the app’s current action definitions.'};
 const result=event.data?.output?.result;
 return {method:'Workspace app replied',content:result?.notice||'The app returned a result. My AI will check it before reporting the outcome.'};
}
