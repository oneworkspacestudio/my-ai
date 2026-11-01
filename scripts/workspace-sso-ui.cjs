// Isolated production-asset reproduction; no owner app, account or model is used.
const http=require('node:http'),fs=require('node:fs'),path=require('node:path'),assert=require('node:assert/strict');
const {chromium}=require('../web/node_modules/@playwright/test');
const dist=path.resolve(__dirname,'../internal/webui/dist');
let browser,origin,session,stream,connected=false,frameVisits=0,connectApprovals=0,writes=0,oauthState='',oauthCode='',failIdentity=process.env.SSO_CASE==='identity';
const user={id:'fixture-ai-account',name:'Fixture',role:'member'};
const send=(type,data)=>stream?.write(JSON.stringify({type,data})+'\n');
const json=(res,data,status=200)=>{res.writeHead(status,{'Content-Type':'application/json'});res.end(JSON.stringify(data))};
const redirect=(res,to)=>{res.writeHead(303,{Location:to});res.end()};
const hasCookie=(req,name,value)=>(req.headers.cookie||'').split(';').some(v=>v.trim()===name+'='+value);
const server=http.createServer(async(req,res)=>{
 const u=new URL(req.url,'http://fixture'),p=u.pathname;let body='';for await(const b of req)body+=b;
 if(p==='/_workspace/open/hosting'){frameVisits++;assert.ok(hasCookie(req,'workspace_session','fixture-portal'));return redirect(res,'/hosting/')}
 if(p==='/hosting/'){
  if(!hasCookie(req,'myhosting_session','fixture-hosting'))return redirect(res,'/hosting/auth/login');
  connected=!failIdentity;res.writeHead(200,{'Content-Type':'text/html'});return res.end('<!doctype html><title>Fixture Hosting</title>');
 }
 if(p==='/hosting/auth/login'){oauthState='state-'+Math.random();return redirect(res,'/_fixture/accounts/authorize?state='+oauthState)}
 if(p==='/_fixture/accounts/authorize'){
  assert.ok(hasCookie(req,'workspace_session','fixture-portal'));assert.equal(u.searchParams.get('state'),oauthState);
  oauthCode='one-time-code';return redirect(res,'/hosting/auth/callback?state='+oauthState+'&code='+oauthCode);
 }
 if(p==='/hosting/auth/callback'){if(process.env.SSO_CASE==='stop'){res.writeHead(200,{'Content-Type':'text/html'});return res.end('<title>Fixture sign-in challenge</title>')}
  assert.equal(u.searchParams.get('state'),oauthState);assert.equal(u.searchParams.get('code'),oauthCode);oauthCode='';
  res.setHeader('Set-Cookie','myhosting_session=fixture-hosting; Path=/hosting/; HttpOnly; SameSite=Lax');return redirect(res,'/hosting/');
 }
 if(p==='/hosting/api/me'){
  if(!hasCookie(req,'myhosting_session','fixture-hosting'))return json(res,{error:'sign in'},401);
  connected=!failIdentity;return json(res,{user:{id:'different-native-id',issuer:origin,subject:failIdentity?'other-account':'fixture-subject'}});
 }
 if(p.startsWith('/myai/api/')){res.setHeader('X-Workspace-Account',user.id);
  const api=p.slice('/myai'.length);
  if(api==='/api/bootstrap')return json(res,{oidc:false});
  if(api==='/api/me'){res.setHeader('Set-Cookie','workspace_session=fixture-portal; Path=/; HttpOnly; SameSite=Lax');return json(res,{user,csrf:'fixture-csrf'})}
  if(api==='/api/state')return json(res,{storageReady:true});
  if(api==='/api/models')return json(res,{models:[{provider:'ollama',id:'fixture:latest',name:'Fixture model'}],status:{ollama:'Available'}});
  if(api==='/api/workspace')return json(res,{apps:[{id:'hosting',name:'My Hosting',description:'Fixture hosting',url:origin+'/_workspace/open/hosting',probe:'/hosting/api/me',state:'running',connected,nativeAccess:true,operations:[{id:'sites',reviewRequired:false},{id:'create_site',reviewRequired:true}]}]});
  if(api==='/api/workspace/connection'){
   assert.equal(req.headers['x-csrf-token'],'fixture-csrf');assert.equal(req.headers['x-workspace-expected-account'],user.id);assert.equal(JSON.parse(body).app,'hosting');
   if(failIdentity&&frameVisits>0)return json(res,{error:'The app is signed in to another account'},409);return json(res,{connected,connectionRequired:!connected});
  }
  if(['/api/projects','/api/models/downloads','/api/code-models'].includes(api))return json(res,[]);
  if(api==='/api/tools')return json(res,{tools:[],commands:[]});
  if(api==='/api/sessions'){
   if(req.method==='POST'){session={...JSON.parse(body),revision:1};return json(res,session)}
   return json(res,session?[session]:[]);
  }
  if(api.endsWith('/turn')){
   res.writeHead(200,{'Content-Type':'application/x-ndjson','Cache-Control':'no-store'});stream=res;
   send('session',session);
   send('approval',{id:'connect-fixture',method:'workspace/connect',params:{title:'Connect My Hosting',app:'My Hosting',appId:'hosting',url:origin+'/_workspace/open/hosting'}});
   return;
  }
  if(api.endsWith('/approval')){
   const input=JSON.parse(body);assert.equal(req.headers['x-csrf-token'],'fixture-csrf');
   if(input.id==='connect-fixture'){
    assert.ok(connected,'Connection must be verified before resuming');assert.equal(input.decision,'accept');connectApprovals++;
    send('approval',{id:'mutation-fixture',method:'workspace/action-review',params:{title:'Create a site',app:'My Hosting',details:{name:'Synthetic site'}}});
   }else{
    assert.equal(input.id,'mutation-fixture');assert.equal(input.decision,'accept');writes++;
    const content='Created the synthetic site after your review.';send('delta',content);
    session={...session,messages:[{id:'reply',role:'assistant',content}]};send('done',session);stream.end();stream=null;
   }
   return json(res,{accepted:true});
  }
  return json(res,{});
 }
 if(p.startsWith('/myai/')){
  const rel=p.slice('/myai/'.length),file=rel?path.join(dist,rel):path.join(dist,'index.html');
  if(!file.startsWith(dist)||!fs.existsSync(file)){res.writeHead(404);return res.end()}
  res.writeHead(200,{'Content-Type':file.endsWith('.js')?'application/javascript':file.endsWith('.css')?'text/css':'text/html'});return res.end(fs.readFileSync(file));
 }
 res.writeHead(404);res.end();
});
(async()=>{try{
 await new Promise(r=>server.listen(0,'127.0.0.1',r));origin='http://127.0.0.1:'+server.address().port;
 browser=await chromium.launch({channel:'msedge',headless:true});const page=await browser.newPage(),errors=[];page.on('pageerror',e=>errors.push(e.message));
 await page.goto(origin+'/myai/');await page.getByRole('textbox',{name:/Ask anything/}).fill('Publish my project to My Hosting');
 await page.getByRole('button',{name:'Send message',exact:true}).click();
 if(process.env.EXPECT_CONNECTION_REPRO){
  await page.getByRole('link',{name:'Open My Hosting',exact:true}).waitFor({timeout:5000});assert.equal(frameVisits,0);assert.equal(connectApprovals,0);assert.equal(writes,0);
  console.log('REPRODUCED: cold Hosting session requires manual connection; API warming never starts SSO');return;
 }
 if(process.env.SSO_CASE==='stop'){
  await page.locator('iframe[data-workspace-connection]').waitFor({state:'attached'});
  await page.getByRole('button',{name:'Stop',exact:true}).click();
  await page.locator('iframe[data-workspace-connection]').waitFor({state:'detached'});await page.waitForTimeout(400);
  assert.equal(connectApprovals,0);assert.equal(writes,0);console.log('PASS: stopping cancels sign-in, removes frame and never resumes stale approvals');return;
 }
 if(process.env.SSO_CASE==='identity'){
  await page.getByRole('link',{name:'Resume MyAccounts sign-in',exact:true}).waitFor({timeout:15000});
  assert.equal(connectApprovals,0);assert.equal(writes,0);assert.equal(await page.locator('iframe[data-workspace-connection]').count(),0);
  console.log('PASS: mismatched native identity is denied; no automatic approval or mutation');return;
 }
 await page.getByRole('button',{name:'Approve once',exact:true}).waitFor({timeout:15000});
 assert.equal(connectApprovals,1);assert.equal(frameVisits,1);assert.equal(writes,0,'Sign-in must not approve app mutations');
 assert.equal(await page.getByRole('link',{name:'Open My Hosting',exact:true}).count(),0);
 await page.getByRole('button',{name:'Approve once',exact:true}).click();await page.getByText('Created the synthetic site after your review.',{exact:true}).waitFor();
 assert.equal(writes,1);assert.equal(await page.locator('iframe[data-workspace-connection]').count(),0);assert.deepEqual(errors,[]);
 console.log('PASS: cold session signs in through native SSO and resumes once; app changes still require explicit review');
}finally{if(browser)await browser.close();if(stream)stream.end();await new Promise(r=>server.close(r))}})().catch(e=>{console.error(e);process.exitCode=1});
