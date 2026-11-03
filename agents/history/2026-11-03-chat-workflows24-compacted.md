# Chat formatting and small-model workflows

Active user asks chat formatting/copy/code features and Codex-like tools, skills
and context tailored to small LLMs. Actual source is NTFS worktree
C:/Projects/.cache/my-ai-20261024/advisor21-src, branch codex/model-advisor,
public main e58d31f (MyAI23). Aggregate workspace is not the app Git root.
Prior benchmark/report work remains uncommitted in tools/agent-bench/{README.md,
package_report.py,report/Benchmark.aui,report/main.ts} and the advisor21 history.
Do not revert or include those pending report changes in this release.

Implemented: AUI message actions and code toolbar; safe GFM, KaTeX, highlighted
code, exact code downloads/copy, message text/Markdown copy, per-message downloads,
scoped responsive styles; new native authenticated conversation branching endpoint
preserves attachments/actual receipts while clearing opaque provider replay and
leaving original data untouched; edit prepares a draft, does not replay actions.
Scroll following pauses when browsing history; desktop Enter/Shift+Enter behavior.

Backend: six embedded focused workflows, metadata-first local tool disclosure;
attachment readers eagerly available for attached files, conversation readers
loaded through load_skill. Removed automatic Git/Hosting schema preload for every
local project; explicit/current app discovery remains. Native scope/reviews kept.
Large model-facing tool outputs reference full private receipts; paginated reader
and paginated conversation reader. Exact file pages and schemas exempt from
abbreviation. Context packing folds complete tool exchanges only, never user or
system instructions, schemas or unresolved calls. Auto reply reserve may shrink
to >=512 tokens; explicit settings preserved. Whole chat summarization not added.

Official Codex adapter exists but managed projectCode uses My AI's engine for all
providers; do not claim this release switched local/managed Code to app-server.
Documented OpenAI skill disclosure/app-server sources were actually opened.

Deps installed into independent chat24-deps under the named C cache. web/node_modules
junction moved from shared project20 deps to this new root, old deps unchanged.
remark-math6.0.0, rehype-katex7.0.1, rehype-highlight7.0.2; KaTeX override0.18.2
fixes advisory in transitive0.16. Production npm audit0 after override. Check actual
package-lock for final versions. AUI adapter imports generated CodeActions and
JumpToLatest views; edit AUI, not generated trees.

Full Go/SearXNG suite passed before final reply/scroll refinements. Focused native
request test passes for8K project context with synthetic provider, exact user
constraints preserved. Initial scoped tool JSON4195 bytes vs9886 before disclosure;
metadata/guidance1160 bytes (not total token count). Browser fixture passes desktop
and390px, code copy/download, GFM/math, safe HTML/URLs, no automatic image network,
branch draft and separate stream blocks. Windows clipboard normalizes LF to CRLF;
download compared exact UTF8. Synthetic screenshots manually inspected in tooling.

Artifacts I:/Projects/tooling/my-ai-chat-workflows-20261103. Current new changes
not yet committed, version still23 at this checkpoint; no new release/install.
Need final build/tests/vet, licence notices, native fresh+upgrade/branch acceptance,
version24 and release/channel packaging, then protected app-only installs within
standing delivery preference. Preserve other tasks' updated pins and transactions.
Primary read-only current state: Workspace114, MyAI20, transaction complete21:47
UTC; don't rely on old113 installer expectations. PC2 last task installed23,
benchmark was interrupted by separate MyTube maintenance; no completed PC2 winner.
No owner conversation/inference/deployment used as a fixture.

Final checks: production build/browser fixture/audit0 and full Go+SearXNG/vet pass. Native standalone fresh24 and23-to24 restart upgrade pass with login, branch route, unchanged original, CSRF, revision conflict and cross-account denial; model profiles preserved. Candidate SHA2560fba570a5defb1581277b5388b399128432e7b116f6c6502de21a06754938c7d. Exact8K request-construction regression passes; no real-model/cloud claim. Native test roots chat24-native and chat24-native-r2 are synthetic and their owned processes stopped.22 new/changed production dependency licence notices collected, missing monorepo notices fetched by npm gitHead from official source. Version24 set; publication pending. All private test logs remain outside Git.

Release-build correction: clean checkout initially exposed mixed embedded Python/skill EOLs and missing generated icon. Canonical LF attributes/build normalization added; build now derives default version from compatibility.json and uses the existing pinned rsrc generator. Native fresh/upgrade tests rerun against final SHA256f1395d02eec24382bd1c3cad2721cc47b3d006f5e0ee7da3ad1324ad44ffbedd, identical to independent clean build. This supersedes prior candidate SHA. Full tests/vet pass again; govulncheck0 reachable/package findings (1 unused required-module advisory). Native r3 processes stopped. PC2 another-task update was observed and allowed to finish; no MyAI24 apply yet.

Delivery completed 2026-11-03: source/tag c936d8674585982a4a4a3db3ccdb90ea501b9453,
public v0.1.0-preview.24; nine uploaded assets and anonymous checksums verified.
Workspace channel ab551a1 promoted24, preserving unrelated rows/pins/withholdings.
Actions disabled. Immutable public assets were not replaced. Primary updated20
to24 on manager114; PC2 updated23 to24 on manager113, each through one normal
protected MyAI-only apply. Both report healthy apps and completed recovery/startup
checks. Primary retains36 other app contracts, unselected payload bytes, identity,
credential contents, model preferences/inventory. Its schema3 recovery manifest,
12 files and2 directories independently verified. Audit helper initially treated
directories as files; corrected read-only audit passed without repeating apply.
PC2 native controller verified its recovery; remote backup bytes were not audited
independently. Both served tested JS/CSS byte hashes and bootstrap version24.
PC2 Voice6/Slicer7/manager113 retained. No owner conversation or inference used as
a fixture. Browser network inspection stopped; synthetic native processes stopped.

Earlier PC2 benchmark checkpoint is superseded only as to run state: before this
update its API reported completed, so no benchmark was interrupted. Results and
ranking were not evaluated by this chat-formatting task. Preserve pending advisor
report/benchmark edits, including bench.py, safe_bench.py and summarize.py that
appeared while delivering24. No real-model quality or full Codex parity claim.
See docs/chat-workflows24-delivery.json for concise installation evidence.
