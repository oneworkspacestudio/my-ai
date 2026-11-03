import React,{useState} from 'react';
import Markdown from 'react-markdown';
import gfm from 'remark-gfm';
import math from 'remark-math';
import katex from 'rehype-katex';
import highlight from 'rehype-highlight';
import {View as CodeActions} from '../generated/CodeActions';
import {normalizeMath} from '../markdown';
import {copyText,downloadText,codeFilename} from '../clipboard';
import 'katex/dist/katex.min.css';

const h=React.createElement;
function nodeText(node:any):string{return node?.type==='text'?node.value:(node?.children||[]).map(nodeText).join('')}
function CodeBlock({node,children}:any){
  const [wrapped,setWrapped]=useState(false),[copied,setCopied]=useState(false),[error,setError]=useState('');
  const code=node?.children?.find((n:any)=>n.tagName==='code'),language=(code?.properties?.className||[]).find((c:string)=>c.startsWith('language-'))?.slice(9)||'text';
  // HAST text includes the terminal newline represented by the Markdown fence.
  // Highlight spans never contribute labels or alter the copied source.
  const text=nodeText(code);
  const action=async(name:string)=>{setError('');try{
    if(name==='wrap')setWrapped(v=>!v);
    if(name==='download')downloadText(text,codeFilename(language));
    if(name==='copy'){await copyText(text);setCopied(true);setTimeout(()=>setCopied(false),1800)}
  }catch(e:any){setError(e.message||'Copy failed. Select the code to copy it.')}};
  return h('div',{className:'code-block'},
    h('div',{'data-copy-exclude':true},h(CodeActions,{data:{language,wrapped,wrapLabel:wrapped?'Unwrap':'Wrap',copyLabel:copied?'Copied':'Copy code',error},onAction:action})),
    h('pre',{className:wrapped?'code-pre code-pre--wrap':'code-pre',tabIndex:0,'aria-label':language+' code'},children));
}
const components:any={pre:CodeBlock,
  a:({node,...p}:any)=>h('a',{...p,target:'_blank',rel:'noopener noreferrer'}),
  table:({node,...p}:any)=>h('div',{className:'markdown-table',tabIndex:0,'aria-label':'Scrollable table'},h('table',p)),
  // Untrusted Markdown must not trigger a tracking request merely by rendering.
  img:({src,alt}:any)=>src?h('a',{href:src,target:'_blank',rel:'noopener noreferrer'},alt||'Open image'):h('span',{},alt||'Image')};
export function MarkdownView({content,messageId}:any){
  return h('div',{className:'markdown','data-message-id':messageId},h(Markdown,{remarkPlugins:[gfm,math],rehypePlugins:[[katex,{trust:false,throwOnError:false,maxExpand:1000,maxSize:20}],[highlight,{detect:false,ignoreMissing:true}]],components,skipHtml:false,children:normalizeMath(String(content||''))}));
}
