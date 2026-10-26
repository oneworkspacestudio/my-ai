const label=(key:string)=>({listId:'Task list',notebook_id:'Notebook',path:'File',source:'Drive',content:'Text',body:'Note text',data:'Event or contact',commit:'Saved Git revision',environment:'Destination',overwrite:'Replace existing file',requestKey:'Request identifier'} as Record<string,string>)[key]||key.replace(/[_-]/g,' ').replace(/([a-z])([A-Z])/g,'$1 $2').replace(/^./,s=>s.toUpperCase());

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
 if(last?.method!=='tool/start')return '';
 if(last.data?.name==='get_workspace_apps')return 'Checking your Workspace apps…';
 if(last.data?.name==='describe_workspace_app')return 'Reading the app’s available actions…';
 if(last.data?.name==='use_workspace_app')return `Using ${apps.find(a=>a.id===last.data.arguments?.app)?.name||'your Workspace app'}…`;
 return '';
}

export function workspaceActivity(event:any,apps:any[]):{method:string;content:string}|null {
 const name=event.data?.name;
 if(!['get_workspace_apps','describe_workspace_app','use_workspace_app'].includes(name))return null;
 if(event.method==='tool/start')return {method:workspaceWorkingLabel([event],apps),content:name==='use_workspace_app'?'Checking this request with your app and account. Changes wait for your review.':'Reading the installed app’s registered capabilities.'};
 if(event.data?.failed)return {method:'App request could not finish',content:String(event.data.output?.error||'Open the app to check its current state.')};
 if(name!=='use_workspace_app')return {method:'App actions available',content:'My AI has the app’s current action definitions.'};
 const result=event.data?.output?.result;
 return {method:'Workspace app replied',content:result?.notice||'The app returned a result. My AI will check it before reporting the outcome.'};
}
