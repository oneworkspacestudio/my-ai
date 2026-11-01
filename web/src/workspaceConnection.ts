type API=(path:string,options?:any)=>Promise<any>;
type Params={url?:string;appId?:string;app?:string};
const pending=new Map<string,Promise<void>>();

// Use the browser's normal native OIDC flow. Never mint credentials or approve a mutation.
export async function connectWorkspaceApp(params:Params,account:string,api:API,signal:AbortSignal) {
 const url=new URL(params.url||'',location.href),match=url.pathname.match(/^\/_workspace\/open\/([a-z0-9-]+)$/);
 if(url.origin!==location.origin||url.username||url.password||url.search||url.hash||!match||(params.appId&&params.appId!==match[1]))throw Error('The app sign-in target is invalid');
 const id=match[1],key=account+'|'+id;
 const existing=pending.get(key);if(existing)return existing;
 const job=(async()=>{
  const controller=new AbortController(),abort=()=>controller.abort();
  signal.addEventListener('abort',abort,{once:true});if(signal.aborted)controller.abort();
  const timeout=setTimeout(()=>controller.abort(),20000);let frame:HTMLIFrameElement|null=null;
  try{
   const catalog=await api('/api/workspace',{signal:controller.signal}),app=(catalog.apps||[]).find((a:any)=>a.id===id);
   if(!app?.nativeAccess||app.url!==url.href)throw Error('This app does not have an installed Workspace connection');
   const check=async()=>{
    const result=await api('/api/workspace/connection',{method:'POST',body:JSON.stringify({app:id}),signal:controller.signal});
    if(result.connected===true)return true;
    if(result.connectionRequired!==true)throw Error(result.error||'Workspace could not verify the app connection');
    return false;
   };
   if(await check())return;
   frame=document.createElement('iframe');frame.hidden=true;frame.title='Workspace sign-in';
   frame.dataset.workspaceConnection=id;frame.referrerPolicy='no-referrer';
   frame.setAttribute('sandbox','allow-scripts allow-same-origin allow-forms');frame.src=url.href;
   document.body.appendChild(frame);
   while(!controller.signal.aborted){
    await new Promise<void>((resolve,reject)=>{
     const stopped=()=>{clearTimeout(timer);reject(new DOMException('Aborted','AbortError'))};
     const timer=setTimeout(()=>{controller.signal.removeEventListener('abort',stopped);resolve()},300);
     controller.signal.addEventListener('abort',stopped,{once:true});
     if(controller.signal.aborted){controller.signal.removeEventListener('abort',stopped);stopped()}
    });
    if(await check())return;
   }
   throw new DOMException('Aborted','AbortError');
  }catch(error){
   if(signal.aborted)throw new DOMException('Aborted','AbortError');
   if(controller.signal.aborted)throw Error('Automatic MyAccounts sign-in did not complete. Refresh your Workspace sign-in, then retry. No app changes were submitted.');
   throw error;
  }finally{clearTimeout(timeout);signal.removeEventListener('abort',abort);controller.abort();frame?.remove()}
 })();
 pending.set(key,job);try{await job}finally{if(pending.get(key)===job)pending.delete(key)}
}
