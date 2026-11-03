import {createServer} from 'node:http';
import {readFile,mkdir,writeFile} from 'node:fs/promises';
import {resolve,extname} from 'node:path';
import assert from 'node:assert/strict';
import {chromium} from 'playwright';
const dist=resolve('../internal/webui/dist'),output=process.env.MYAI_UI_EVIDENCE;
const code='const greeting = "Hello <world> 🌍";\n  console.log(greeting);\n';
const markdown='# Formatting check\n\nA **bold** word, *emphasis*, ~~removed~~ and `inline code`.\n\n## Lists\n\n1. First\n   - Nested\n2. Second\n\n> A useful quote.\n\n- [x] Complete\n- [ ] Pending\n\n| Name | Value |\n| --- | ---: |\n| Alpha | 42 |\n\n```javascript\n'+code+'```\n\nInline math: \\(x^2 + y^2 = z^2\\).\n\n\\[\nE = mc^2\n\\]\n\n[Source](https://example.com) and [unsafe](javascript:alert(1)).\n\n![Private image](https://tracker.invalid/pixel.png)\n\n<script>window.injected = true</script>\n\n```unknown-language\nliteral \\(not math\\)\n```\n';
const initial={id:'a'.repeat(48),schema:1,title:'Formatting fixture',provider:'ollama',model:'fixture:small',mode:'chat',policy:'read-only',revision:2,archived:false,pinned:false,messages:[{id:'u1',role:'user',content:'Please format this.'},{id:'a1',role:'assistant',content:markdown},{id:'a2',role:'assistant',content:'A separate response block.'}],events:[]};
let sessions=new Map([[initial.id,structuredClone(initial)]]),branchBody,turns=0;
const server=createServer(async(req,res)=>{try{
 const url=new URL(req.url,'http://localhost'),path=url.pathname;
 res.setHeader('X-Workspace-Account','fixture-user');
 const json=v=>{res.setHeader('Content-Type','application/json');res.end(JSON.stringify(v))};
 let body='';for await(const c of req)body+=c;
 const input=body?JSON.parse(body):{};
 if(path==='/api/bootstrap')return json({});
 if(path==='/api/me')return json({user:{id:'fixture-user',name:'Test User',role:'owner'},csrf:'fixture-csrf'});
 if(path==='/api/state')return json({storageReady:true});
 if(path==='/api/projects')return json([]);
 if(path==='/api/models')return json({models:[{provider:'ollama',id:'fixture:small',name:'Fixture model'}],status:{ollama:'Ready'}});
 if(path==='/api/models/downloads')return json([]);
 if(path==='/api/tools')return json({tools:[],skills:[{id:'project',name:'Project editing',description:'Small, focused workflow'}]});
 if(path==='/api/workspace')return json({apps:[]});
 if(path==='/api/model-benchmark')return json({});
 if(path==='/api/sessions'){
  if(req.method==='POST'){const old=sessions.get(input.id);const saved={...input,revision:(old?.revision||0)+1,messages:old?.messages||input.messages,events:old?.events||input.events};sessions.set(saved.id,saved);return json(saved)}
  return json([...sessions.values()]);
 }
 const match=/^\/api\/sessions\/([^/]+)(?:\/(.+))?$/.exec(path);
 if(match){const s=sessions.get(match[1]);if(!s){res.statusCode=404;return json({error:'Not found'})}
  if(match[2]==='branch'){branchBody=input;assert.equal(req.headers['x-csrf-token'],'fixture-csrf');const index=s.messages.findIndex(m=>m.id===input.message),clone={...structuredClone(s),id:'b'.repeat(48),title:'Branch fixture',revision:1,messages:s.messages.slice(0,index+(input.before?0:1))};sessions.set(clone.id,clone);return json(clone)}
  if(match[2]==='turn'){turns++;res.setHeader('Content-Type','application/x-ndjson');const send=(type,data)=>res.write(JSON.stringify({type,data})+'\n');const u={id:'u'+turns+Date.now(),role:'user',content:input.text};s.messages.push(u);send('session',s);send('delta','First streamed paragraph.');await new Promise(r=>setTimeout(r,100));const m={id:'stream-one-'+turns,role:'assistant',content:'First streamed paragraph.'};s.messages.push(m);send('message',m);send('delta','Second streamed paragraph.');await new Promise(r=>setTimeout(r,100));s.messages.push({id:'stream-two-'+turns,role:'assistant',content:'Second streamed paragraph.'});send('done',s);return res.end()}
  return json(s);
 }
 const target=resolve(dist,'.'+(path==='/'?'/index.html':path));if(!target.startsWith(dist)){res.statusCode=403;return res.end()}
 const types={'.js':'text/javascript','.css':'text/css','.html':'text/html','.svg':'image/svg+xml','.woff2':'font/woff2','.woff':'font/woff','.ttf':'font/ttf'};
 res.setHeader('Content-Type',types[extname(target)]||'application/octet-stream');res.end(await readFile(target));
 }catch(e){res.statusCode=500;res.end(JSON.stringify({error:e.message}))}});
await new Promise(r=>server.listen(0,'127.0.0.1',r));
const origin='http://127.0.0.1:'+server.address().port,browser=await chromium.launch({channel:'chrome',headless:true});
try{
 const context=await browser.newContext({viewport:{width:1280,height:900},permissions:['clipboard-read','clipboard-write']});
 const page=await context.newPage(),errors=[],outgoing=[];page.on('pageerror',e=>errors.push(e.message));page.on('request',r=>{if(!r.url().startsWith(origin))outgoing.push(r.url())});
 await page.goto(origin);await page.getByRole('button',{name:'Formatting fixture',exact:true}).click();
 const assistant=page.locator('[data-message-id="a1"]');
 await assistant.getByRole('heading',{name:'Formatting check'}).waitFor();
 assert.equal(await assistant.locator('.hljs-keyword').count()>0,true);
 assert.equal(await assistant.locator('.katex').count(),2);
 assert.equal(await assistant.locator('table').count(),1);
 assert.equal(await assistant.locator('blockquote').innerText(),'A useful quote.');
 assert.equal(await assistant.locator('script').count(),0);assert.equal(await page.evaluate(()=>window.injected),undefined);
 assert.deepEqual(outgoing,[]);assert.equal(await assistant.locator('a[href^="javascript:"]').count(),0);
 const codeBlock=assistant.locator('.code-block').first();
 await codeBlock.getByRole('button',{name:'Copy code',exact:true}).click();assert.equal(await page.evaluate(()=>navigator.clipboard.readText().then(s=>s.replaceAll('\r\n','\n'))),code);
 await codeBlock.getByRole('button',{name:'Wrap',exact:true}).click();assert.equal(await codeBlock.locator('pre').evaluate(e=>getComputedStyle(e).whiteSpace),'pre-wrap');
 await codeBlock.getByRole('button',{name:'Unwrap',exact:true}).click();assert.equal(await codeBlock.locator('pre').evaluate(e=>getComputedStyle(e).whiteSpace),'pre');
 const dl=page.waitForEvent('download');await codeBlock.getByRole('button',{name:'Download code',exact:true}).click();const download=await dl;assert.equal(download.suggestedFilename(),'code.js');assert.equal(await readFile(await download.path(),'utf8'),code);
 const card=page.locator('.card--assistant').filter({has:assistant});
 await card.getByRole('button',{name:'Copy Markdown',exact:true}).click();assert.equal(await page.evaluate(()=>navigator.clipboard.readText().then(s=>s.replaceAll('\r\n','\n'))),markdown);
 await card.getByRole('button',{name:'Copy text',exact:true}).click();const copied=await page.evaluate(()=>navigator.clipboard.readText().then(s=>s.replaceAll('\r\n','\n')));assert(copied.includes('Formatting check'));assert(!copied.includes('Copy code'));assert(!copied.includes('Copy Markdown'));
 await page.locator('.stack--conversation').evaluate(e=>e.scrollTop=0);
 if(output){await mkdir(output,{recursive:true});await page.screenshot({path:resolve(output,'chat-desktop.png')})}
 await page.setViewportSize({width:390,height:844});
 assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),true);
 assert.equal(await codeBlock.locator('pre').evaluate(e=>getComputedStyle(e).whiteSpace),'pre');
 if(output)await page.screenshot({path:resolve(output,'chat-mobile.png')});
 await codeBlock.scrollIntoViewIfNeeded();if(output)await page.screenshot({path:resolve(output,'chat-mobile-code.png')});
 await page.setViewportSize({width:1280,height:900});
 await page.getByRole('button',{name:'Edit in new branch',exact:true}).first().click();
 await page.waitForFunction(()=>document.querySelector('textarea')?.value==='Please format this.');assert.equal(branchBody.message,'u1');assert.equal(branchBody.before,true);assert.equal(branchBody.revision,2);
 const prompt=page.getByRole('textbox',{name:'Ask anything, explore an idea…'});assert.equal(await prompt.inputValue(),'Please format this.');assert.equal(initial.messages.length,3);assert.equal(turns,0);
 await prompt.fill('Continue');await prompt.press('Enter');await page.getByText('Second streamed paragraph.',{exact:true}).waitFor();
 assert.equal(await page.locator('.card--assistant').filter({hasText:'First streamed paragraph.'}).count(),1);assert.equal(await page.locator('.card--assistant').filter({hasText:'Second streamed paragraph.'}).count(),1);
 assert.equal(turns,1);assert.deepEqual(errors,[]);assert.deepEqual(outgoing,[]);
 console.log(JSON.stringify({passed:true,checks:['Markdown/GFM','math','highlighting','exact code copy/download','message copy','safe HTML/URLs','no automatic image fetch','desktop/mobile containment','edit branch','Enter to send','separate streamed messages']}));
}finally{await browser.close();server.closeAllConnections();server.close()}
