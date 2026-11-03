import {parse,compileReact} from '@codedia/parser';
import {build} from 'vite';
import {readFile,writeFile} from 'node:fs/promises';
import path from 'node:path';
const root=path.dirname(new URL(import.meta.url).pathname.replace(/^\/([A-Za-z]:)/,'$1'));
const source=await readFile(path.join(root,'Benchmark.aui'),'utf8');
await writeFile(path.join(root,'generated.tsx'),compileReact(parse(source)));
await build({root,base:'./',resolve:{alias:{'@/components/ui':path.join(root,'adapter.ts')}},build:{outDir:process.env.BENCH_REPORT_OUT||path.join(root,'dist'),emptyOutDir:false,assetsInlineLimit:1000000}});
