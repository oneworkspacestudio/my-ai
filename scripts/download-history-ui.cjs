// Production UI with synthetic API only; never contacts owner Ollama/models.
const http=require('node:http'),fs=require('node:fs'),path=require('node:path'),assert=require('node:assert/strict');
const {chromium}=require('../web/node_modules/@playwright/test');
const dist=path.resolve(__dirname,'../internal/webui/dist');let jobs=['queued','downloading','failed','cancelled','completed'].map((state,i)=>({id:String(i),model:'fixture-'+state+':latest',state,status:state==='failed'?'Model requires a newer Ollama version':state,completed:30,total:100})),mutations=[];let role='owner',started=false,checks=0,submitted=0,browser;
const server=http.createServer(async(req,res)=>{const u=new URL(req.url,'http://fixture'),p=u.pathname;if(p.startsWith('/api/')){let body='';for await(const b of req)body+=b;let data={};if(p==='/api/bootstrap')data={oidc:false};else if(p==='/api/me')data={user:{id:'fixture-owner',name:'Fixture',role},csrf:'fixture-csrf'};else if(p==='/api/state')data={storageReady:true};else if(p==='/api/models')data={models:[{provider:'ollama',id:'fixture:latest',name:'Fixture model'}],status:{ollama:'Available'}};else if(p==='/api/models/downloads')data=jobs;else if(p.startsWith('/api/models/downloads/')){assert.equal(req.headers['x-csrf-token'],'fixture-csrf');const id=p.split('/')[4];mutations.push([req.method,id]);if(req.method==='DELETE')jobs=jobs.filter(j=>j.id!==id);else jobs=jobs.map(j=>j.id===id?{...j,state:'cancelled'}:j);data={ok:true};}else if(['/api/sessions','/api/projects','/api/models/downloads','/api/code-models'].includes(p))data=[];else if(p==='/api/ollama/upgrade'){if(role!=='owner'){res.writeHead(403);return res.end('{}')}if(req.method==='POST'){assert.equal(JSON.parse(body).confirm,true);assert.equal(req.headers['x-csrf-token'],'fixture-csrf');submitted++;started=true;checks=0;data={accepted:true};}else{const busy=started&&++checks<3;data={busy,current:started&&!busy?'0.40.13':'0.40.11',phase:busy?'Downloading verified installer':started?'Ollama upgraded to 0.40.13':'Ready to check',completed:0,total:0};}}res.writeHead(200,{'Content-Type':'application/json'});return res.end(JSON.stringify(data));}let file=p==='/'?path.join(dist,'index.html'):path.join(dist,p);if(!file.startsWith(dist)||!fs.existsSync(file)){res.writeHead(404);return res.end()};res.writeHead(200,{'Content-Type':file.endsWith('.js')?'application/javascript':file.endsWith('.css')?'text/css':'text/html'});res.end(fs.readFileSync(file))});
(async()=>{try{
 await new Promise(r=>server.listen(0,'127.0.0.1',r));browser=await chromium.launch({channel:'msedge',headless:true});
 const c=await browser.newContext(),page=await c.newPage(),errors=[];page.on('pageerror',e=>errors.push(e.message));
 await page.goto('http://127.0.0.1:'+server.address().port);await page.getByRole('button',{name:'Models',exact:true}).click();
 assert.equal(await page.getByRole('button',{name:'Cancel download',exact:true}).count(),2);
 assert.equal(await page.getByRole('button',{name:'Remove entry',exact:true}).count(),0);
 assert.equal(await page.getByRole('button',{name:'Upgrade Ollama',exact:true}).count(),0);
 for(const viewport of [{width:1440,height:1000},{width:390,height:844}]){
  await page.setViewportSize(viewport);
  assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth));
  if(process.env.MYAI_DOWNLOAD_EVIDENCE_DIR){fs.mkdirSync(process.env.MYAI_DOWNLOAD_EVIDENCE_DIR,{recursive:true});await page.screenshot({path:path.join(process.env.MYAI_DOWNLOAD_EVIDENCE_DIR,'models-'+viewport.width+'.png')});}
 }
 await page.getByText('Download history · 3',{exact:true}).click();
 assert.equal(await page.getByRole('button',{name:'Remove entry',exact:true}).count(),3);
 assert.equal(await page.locator('.card--download-row progress').count(),2);if(process.env.MYAI_DOWNLOAD_EVIDENCE_DIR)await page.screenshot({path:path.join(process.env.MYAI_DOWNLOAD_EVIDENCE_DIR,'models-history-mobile.png')});
 page.on('dialog',d=>d.accept());
 await page.getByRole('button',{name:'Remove entry',exact:true}).first().click();
 await page.getByText('Download history · 2',{exact:true}).waitFor();
 assert.deepEqual(mutations[0],['DELETE','2']);
 await page.getByRole('button',{name:'Cancel download',exact:true}).first().click();
 await page.getByText('Download history · 3',{exact:true}).waitFor();
 assert.deepEqual(mutations[1],['POST','0']);
 assert.equal(await page.getByRole('button',{name:'Cancel download',exact:true}).count(),1);
 await page.getByText('Add a local model',{exact:true}).click();
 await page.getByRole('textbox',{name:'Model tag, e.g. qwen3:0.6b'}).waitFor();
 await page.getByText('Server & maintenance',{exact:true}).click();
 await page.getByRole('button',{name:'Upgrade Ollama',exact:true}).waitFor();
 assert.ok(await page.evaluate(()=>document.querySelector('dialog').scrollWidth<=document.querySelector('dialog').clientWidth));
 await c.close();role='member';const m=await browser.newContext(),member=await m.newPage();
 await member.goto('http://127.0.0.1:'+server.address().port);await member.getByRole('button',{name:'Models',exact:true}).click();
 await member.getByText('Download history · 3',{exact:true}).click();
 for(const name of ['Remove entry','Cancel download','Upgrade Ollama'])assert.equal(await member.getByRole('button',{name,exact:true}).count(),0);
 assert.deepEqual(errors,[]);
 console.log('PASS model management UI: progressive disclosure, cancel/remove, member permissions, desktop/mobile layout');
}finally{if(browser)await browser.close();await new Promise(r=>server.close(r))}})().catch(e=>{console.error(e);process.exitCode=1});
