import React,{useEffect,useRef,useState} from 'react';
import {AIService} from './generated/AIService';
import {ServiceLogin} from './generated/ServiceLogin';
import {Fields} from './components/ui';
const h=React.createElement;
export function ServiceApp(){
 const base=new URL('.',location.href).pathname.replace(/\/$/,'');
 const [auth,sa]=useState<any>(null),[cfg,sc]=useState<any>({}),[state,ss]=useState<any>({}),[models,sm]=useState<any[]>([]),[clients,scl]=useState<any>({}),[fields,sf]=useState<any>({}),[message,smsg]=useState(''),[busy,sb]=useState(false),[newKey,sk]=useState('');
 const item=useRef<any>(null),authRef=useRef<any>(null);authRef.current=auth;
 async function api(path:string,opt:any={}){const a=authRef.current;const r=await fetch(base+path,{...opt,credentials:'same-origin',headers:{'Content-Type':'application/json','X-CSRF-Token':a?.csrf||'','X-Workspace-Expected-Account':a?.user?.id||''}});const b=await r.json();if(!r.ok){if(r.status===401){sa(null);sk('')}throw Error(b.error||'Request failed')};if(a&&r.headers.get('X-Workspace-Account')&&r.headers.get('X-Workspace-Account')!==a.user.id){sa(null);sk('');throw Error('Account changed. Reconnect to continue')};return b}
 async function refresh(){ss(await api('/api/service'));try{sm((await api('/api/service/models')).models||[])}catch(e:any){smsg(e.message)};if(authRef.current?.user.role==='owner')scl(await api('/api/service/clients'))}
 useEffect(()=>{api('/api/bootstrap').then(sc).then(()=>api('/api/me').then(sa).catch(()=>{})).catch(e=>smsg(e.message))},[]);
 useEffect(()=>{if(!auth)return;void refresh().catch(e=>smsg(e.message));const timer=setInterval(()=>api('/api/service').then(ss).catch(e=>smsg(e.message)),2000);return()=>clearInterval(timer)},[auth]);
 useEffect(()=>{if(state.settings?.model)sf((v:any)=>({...v,model:v.model||state.settings.model}))},[state.settings?.model]);
 const action=async(name:string)=>{if(busy)return;sb(true);smsg('');try{
   if(name==='oidc'){location.href=base+'/auth/login';return}
   if(name==='login'){const setup=location.hash.startsWith('#setup/')?location.hash.slice(7):'';if(setup){const r=await fetch(base+'/api/setup',{method:'POST',credentials:'same-origin',headers:{'Content-Type':'application/json','X-Setup-Token':setup,'Origin':location.origin},body:JSON.stringify({email:fields.email,password:fields.password})});const b=await r.json();if(!r.ok)throw Error(b.error||'Setup failed');history.replaceState(null,'',location.pathname)}else{await api('/api/login',{method:'POST',body:JSON.stringify({email:fields.email,password:fields.password})})};sa(await api('/api/me'));sf({});return}
   if(name==='logout'){await api('/auth/logout',{method:'POST',body:'{}'});sk('');sa(null);location.href=base+'/';return}
   if(name==='enable'||name==='disable'){await api('/api/service/settings',{method:'POST',body:JSON.stringify({model:fields.model||state.settings?.model||'',enabled:name==='enable'})});smsg(name==='enable'?'Service enabled.':'Service stopped. Releasing model memory…')}
   if(name==='createClient'){const c=await api('/api/service/clients',{method:'POST',body:JSON.stringify({id:fields.clientID||'',remote:fields.remote==='remote'})});sk(c.key)}
   if(name==='revoke')await api('/api/service/clients/'+encodeURIComponent(item.current.id),{method:'DELETE'});
   if(name==='copyKey'){await navigator.clipboard.writeText(newKey);smsg('Credential copied')}
   if(name==='dismissKey')sk('');
   await refresh();
 }catch(e:any){smsg(e.message)}finally{sb(false)}};
 const workspaceURL=cfg.workspace||'/',myaiURL=workspaceURL.replace(/\/$/,'')+'/myai/';
 const data=auth?{owner:auth.user.role==='owner',message,busy,workspaceURL,myaiURL,phase:state.phase||'Connecting…',modelLabel:state.settings?.model||'No active model selected',dependency:state.dependency||'Requires host Ollama',queueLabel:(state.queued||0)+' waiting',activeLabel:state.busy?'Processing '+(state.activeApp||'a request'):'Ready for requests',finishedLabel:(state.completed||0)+' completed · '+(state.failed||0)+' failed or cancelled',queuePolicy:state.queuePolicy||'',modelOptions:[{value:'',label:'Choose an installed model'},...models.map(m=>({value:m.name,label:m.name}))],managedLabel:clients.myaiManaged?'My AI · connected automatically by Workspace':'My AI connects automatically in a managed Workspace installation',clientScopes:[{value:'local',label:'Local server app only'},{value:'remote',label:'Remote server app (URL and token)'}],clients:(clients.clients||[]).map((id:string)=>({id,label:id+((clients.remoteClients||[]).includes(id)?' · remote server app':' · local only')})),newKey}:{oidc:cfg.oidc,error:message};
 return h(Fields.Provider,{value:{fields,setField:(k:string,v:string)=>sf((f:any)=>({...f,[k]:v})),item,action}},h(auth?AIService:ServiceLogin,{data,onAction:action}));
}
