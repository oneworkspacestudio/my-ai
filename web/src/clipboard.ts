// Clipboard writes are only invoked by explicit clicks. Preserve exact source text.
export async function copyText(text:string) {
  if(navigator.clipboard?.writeText){await navigator.clipboard.writeText(text);return}
  const focused=document.activeElement as HTMLElement|null,area=document.createElement('textarea');
  area.value=text;area.setAttribute('aria-label','Copy content');area.style.cssText='position:fixed;left:-10000px;top:0';
  document.body.append(area);area.select();
  try{if(!document.execCommand('copy'))throw Error('Copy unavailable. Select the text and copy it manually.')}
  finally{area.remove();focused?.focus()}
}
export function downloadText(text:string,name:string,type='text/plain;charset=utf-8'){
  const url=URL.createObjectURL(new Blob([text],{type})),a=document.createElement('a');
  a.href=url;a.download=name;a.click();setTimeout(()=>URL.revokeObjectURL(url),2000);
}
export function codeFilename(language:string){
  const extensions:Record<string,string>={javascript:'js',js:'js',typescript:'ts',ts:'ts',tsx:'tsx',jsx:'jsx',python:'py',py:'py',html:'html',xml:'xml',css:'css',json:'json',yaml:'yaml',yml:'yml',markdown:'md',md:'md',go:'go',rust:'rs',rs:'rs',sql:'sql',bash:'sh',sh:'sh',powershell:'ps1',c:'c',cpp:'cpp',java:'java',text:'txt'};
  return 'code.'+(extensions[language.toLowerCase()]||'txt');
}
