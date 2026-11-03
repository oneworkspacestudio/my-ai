# Small local model task benchmark

**Delivered:** eight models/192 samples, private native My Git source and real
My Hosting public report. Recommendation Qwen3.5:9b Q4_K_M,24/24 overall,
10/10 deployment checks;6.55GB download. Public URL and final delivery below.

User asks to find the best small LLM for file edits, task execution and native
app hosting, create/run a repeatable benchmark and deploy the results in My
Hosting. This is a benchmark/report artifact, not a new managed app or My AI
executable release. Preserve current installed MyAI20 and owner preferences.

Source worktree: C:/Projects/.cache/my-ai-20261024/agent-bench-src, branch
codex/agent-bench from public main0752b8e. Original I dirty checkouts preserved.
Tools under tools/agent-bench; artifacts I:/Projects/tooling/
my-ai-agent-bench-20261103. No subagents or owner app data used as fixtures.

Bench1.1.0: Python standard library, real local Ollama inference,12 scenarios,
seeds17/29, temperature0.2,thinkingfalse,num_ctx8192,num_predict768,16 rounds,
32calls/180seconds per sample. Exact artifact digests/size/quantization, full
synthetic transcripts and inference timings recorded. Fresh in-memory world per
sample; all file/Git/Hosting tools simulated with native-shaped names/contracts.
No actual shell execution, cloud keys, sign-in/review/project bootstrap scored.
Static publication requires exact saved Git commit, actual simulated job polling
to published, correct URL. Check exact file state, edit conflicts, attachment
copy, failed-job honesty, read-only/negative scope, malicious file instructions,
time/math and archived context distractors. Safety counts attempted forbidden
mutation, path escape and duplicate deployment, even when blocked.

Pilot1.0 (results/) discarded from comparison after tight10-round budget and
ISO-only date grader found unfair. Corrected16 rounds and equivalent date
formats, restarted all models in final-results/. A later Z-suffix compatibility
fix affects only Python3.10 unit fixtures; measured run timestamp uses +00:00.
Seven grading regression checks pass. Model first-load often exceeds90s warmup;
recorded as warmupError and subsequent measured samples continue. Median elapsed
includes failed attempts, not a tokens/sec ranking. Do not mix pilot and final.

Report UI authored in AUI, compiled with existing pinned web dependencies via a
local node_modules junction to project20-src/web. React adapter, sorting by
success/size/time, failures filter,24-sample pagination, expandable evidence,
downloadable JSON and runnable source. No CDN runtime dependencies. Build+TS check
passed on pilot data for layout only; final packaging refuses partial model runs.
Production output is a self-contained index.html plus summary/samples/sourceZIP
and checksums. No native credentials or private fixture paths enter public files.

Final measured results (24 samples each; deployment subset10):

| Exact local model tag | Overall | Deployment | Disk GB |
| --- | ---: | ---: | ---: |
| qwen3.5:9b |24|10|6.55|
| maternion/spark-x2.5-heretic:4b |21|7|2.60|
| qwen3.5:4b |19|5|3.39|
| maternion/minicpm5:2b |16|2|2.68|
| eriktonon/bonsai-2-uncensored:4b |14|0|1.58|
| ministral-3:3b |12|0|2.95|
| qwen3.5:0.8b |8|0|1.04|
| granite4:3b |8|0|2.10|

MiniCPM22/24 guardrail passes from two blocked duplicate submissions; all other
candidates24/24. Several Ministral attempts returned provider HTTP500 and are
counted as observed failures, not interpreted as raw-model research scores.
All eight completed; no unavailable candidates remain. Registry downloads
initially stalled; task-owned clients stopped/resumed without deleting partial
bytes. Later checksum verification needed a longer stream read deadline.
Inference remained sequential. RTX3060 has12GiB,32GB system RAM. An observed
9B8192-context allocation was5,862,210,928 bytes VRAM, not a peak measurement.
No default model preferences changed. New9B/Granite/Ministral installed into the
existing local shared Ollama model store for benchmarking; no second-PC model
installation or larger-context/multi-GPU acceptance is claimed.
Running Ollama0.40.13 actually uses H:/Model per its startup configuration, while
current shell OLLAMA_MODELS says I:/Ollama/models. Earlier20 audit compared the
shell-configured root, not this running service root. No owner model deletion,
movement or preference changes; new benchmark downloads/partial resumes only.

Native intended output already created through normal owner UI: private My Git
repository my-ai-agent-benchmark; My Hosting static project linked to main, site
slug my-ai-agent-benchmark, output'.',no install/build scripts. Source approval
for our own report only. Final live URL:
https://my-ai-agent-benchmark-sites.m8e.co.uk/ . Native Git/UI delivery below.
Do not use existing owner games/chats/sites as test fixtures.

Browser Chrome3, handle benchBrowser. Own final benchLive tab585734866 is the
deliverable; other own Git/Hosting/config/preview tabs are closed at completion.
User tabs untouched. Cua use only, fresh documentation after compaction. Local
Python preview39740 first served report/dist, then final published directory;
stopped after final verification. No viewport override was applied.
Partial model downloads retained for resume; do not delete owner model state.
No GitHub push/release/suite update requested by this benchmark task; source is
delivered into the explicitly requested native My Git/Hosting output project.

## Final verification and native delivery

Seven grading tests pass; full192-record receipt replay validates exact system/
user prompts, simulated tool receipts, recorded grades and unique complete sample
sets. Arithmetic grading accepts formatted4,014 and excludes40142; replay proved
all earlier grades unchanged. No silent prompt-cropping claim: maximum observed
prompt_eval_count6643, under8192. Hardware detection now observes actual memory
and degrades gracefully without NVIDIA tooling; current32GB matches original
metadata. Production AUI/TS/Vite and self-contained browser rendering pass.
Sorting by size/success and failure pagination(70 failed samples,24/24/22 pages)
verified. Desktop browser only; no physical-phone or full-context-window tests.

Frozen published/ files include standalone index, summary, full synthetic traces,
receipt-replay proof, runnable sourceZIP(18files,no runtime caches),README,
dependency MIT notices and SHA256 manifest. Root index313865 bytes. No CDN font/
runtime dependencies or native account credentials/private paths in publication.
Raw pilot results/ remain local and are excluded from final-results/ and output.

Gitea normal UI accepts only five uploads per commit. First stale one-hour-old
form refreshed sign-in and made no commit; fresh form succeeded, then supporting
files committed separately. Final native main e374190a093a8b8b1f3f94453f172b7268252f7d;
report/source commit9dbe3a58a8a1daa653b734b8bf4134619fdebb8f. No repeated ambiguous
Git mutation: inspected actual head/file list before proceeding.

Native Hosting2 project built exact final commit. First publication35569605...
failed its public HTTPS gate because a public resolver still returned the old
wildcard address. Real DNS subsequently agreed across local,Cloudflare andGoogle.
Normal **Retry same source & configuration** then published in4seconds, job
2f7605b935afaf040b65759e55c25d68,source archive SHA256
b97660115d9c02029b53408432f2ec30ac60bd571926d0c5105fbab49600b971.
No DNS/security setting, public-health check, Hosting source or binary was changed.
Read-only Hosting repository/history consulted; its unrelated preview3 preserved.

verify_live.py in task artifacts independently fetched all8 public files, HTTP200,
exact local byte hashes and expected X-MyHosting-Deploy header. Receipt:
live-delivery.json. Actual public dashboard rendered192samples/8models and live
sorting worked. Native full-page screenshot timed out; ordinary viewport capture
saved benchmark-live.png with hardware/counts/recommendation/winning row visible.
The public report tab remains open. No MyAI executable/suite release, GitHub
Actions, remote model defaults, unrelated owner project or conversation changed.

Future reruns: use a fresh result directory for changed settings/models/code.
Keep model inference serial and distinguish mocked tools from native integration.
Prior20 recursive cleanup was policy-blocked; do not bypass that restriction for
those named retained roots. This task made no recursive deletion attempts.
