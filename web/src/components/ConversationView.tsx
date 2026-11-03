import React,{useEffect,useRef,useState} from 'react';
import {View as JumpToLatest} from '../generated/JumpToLatest';
const h=React.createElement;
export function ConversationView({children,conversationId}:any){
 const ref=useRef<HTMLDivElement>(null),follow=useRef(true),[away,setAway]=useState(false);
 const latest=()=>{const el=ref.current;if(el){follow.current=true;el.scrollTop=el.scrollHeight;setAway(false)}};
 useEffect(()=>{latest()},[conversationId]);
 useEffect(()=>{const el=ref.current;if(!el)return;let frame=0;
  const onChange=()=>{cancelAnimationFrame(frame);frame=requestAnimationFrame(()=>{if(follow.current)latest()})};
  const composer=el.parentElement?.querySelector('.stack--composer-wrap');const size=new ResizeObserver(()=>{if(composer)el.parentElement?.style.setProperty('--composer-height',composer.getBoundingClientRect().height+'px')});if(composer)size.observe(composer);
  const observer=new MutationObserver(onChange);observer.observe(el,{subtree:true,childList:true,characterData:true});
  return()=>{observer.disconnect();size.disconnect();cancelAnimationFrame(frame)};
 },[]);
 return h(React.Fragment,{},h('div',{ref,className:'stack stack--conversation',onScroll:()=>{const el=ref.current!;follow.current=el.scrollHeight-el.scrollTop-el.clientHeight<90;setAway(!follow.current)}},children),away?h(JumpToLatest,{data:{},onAction:latest}):null);
}
