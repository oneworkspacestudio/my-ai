// Accept the math delimiters commonly returned by assistants without changing
// literal code fences or inline code. Dollar syntax is already handled by remark.
export function normalizeMath(source:string):string {
 let fence='',inline='',math='',out='';
 for(let i=0;i<source.length;){
  const lineStart=i===0||source[i-1]==='\n';
  if(lineStart&&!inline&&!math){
   const end=source.indexOf('\n',i),line=source.slice(i,end<0?undefined:end),m=/^ {0,3}(`{3,}|~{3,})(.*)$/.exec(line);
   if(m){if(!fence)fence=m[1];else if(m[1][0]===fence[0]&&m[1].length>=fence.length&&!m[2].trim())fence='';out+=line;i+=line.length;continue}
  }
  if(fence){out+=source[i++];continue}
  if(source[i]==='`'&&!math){let j=i;while(source[j]==='`')j++;const ticks=source.slice(i,j);if(!inline)inline=ticks;else if(inline===ticks)inline='';out+=ticks;i=j;continue}
  if(!inline&&source[i]==='\\'){
   const pair=source.slice(i,i+2);
   if(pair==='\\\\'){out+=pair;i+=2;continue}
   if(!math&&(pair==='\\('||pair==='\\[')){
    const closing=pair==='\\('? '\\)':'\\]';
    if(source.indexOf(closing,i+2)>=0){math=closing;out+=pair==='\\('? '$':'\n$$\n';i+=2;continue}
   }else if(math&&pair===math){out+=math==='\\)'?'$':'\n$$\n';math='';i+=2;continue}
  }
  out+=source[i++];
 }
 return out;
}
