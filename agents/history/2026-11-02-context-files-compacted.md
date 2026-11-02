# Dynamic context and attachment files — in progress

User asks editable and model/hardware-aware context, larger Code context, files
read only as needed, and how other assistants handle attachments. Standing chat
preference from preview18 work authorizes release and primary install by default;
second PC remains pending unless separately resumed. Use only isolated synthetic
test data, never owner chats/Drive as functional fixtures. The user-authorized
conversation investigation was read-only: browser and its own export established
208KB input, zero tools/jobs, and model treated a fragment87% through input as
the question. Do not store that private transcript/export or file body in Git.

## Source/worktrees

Original I:/Projects/my-ai is dirty/uncommitted18-era work at3033f3d; preserved.
Public origin/main is58905d9, immutable preview18 source34c809b. Created isolated
I:/Projects/my-ai-context branch codex/context-files from public main. Task source
then copied (16 changed/new files only) into a public-main NTFS worktree:
C:/Projects/.cache/my-ai-20261024/context19-src, branch codex/context-files-ntfs.
This C worktree is now authoritative; I context worktree contains an earlier WIP,
do not overwrite C with it. Later preserve final C changes to I before cleanup.
Do not stage the unborn aggregate workspace or original dirty MyAI checkout.

Task artifacts I:/Projects/tooling/my-ai-context-files-20261102. Initial Go check
against I caches stalled and its verified own Go PID was stopped; offline npm
also failed ENOTCACHED. Copied only required pinned apphost module/cache into the
C Go cache; allowed ordinary checksum-verified proxy download of missing pinned
x/crypto. No dependency versions changed. checks.py now uses C source, C toolchain
and C caches; task temp context19-tmp. Existing C hosting-fix-src node_modules was
copied into new source, old source/cache not modified. No new dependencies.

## Current implementation

New context_budget.go and platform memory probes: Ollama /api/show native model
limit/KV metadata, tags weights, ps loaded allocations; Windows memory API /Linux
meminfo and optional bounded absolute-path existing nvidia-smi. No downloaded
executables, model changes or shell tools exposed to LLM. Conservative fp16 KV
estimate/headroom, unknown metadata fallback8K; automatic Chat target32K, Code64K,
clamped by model/memory estimate. GPU allocations never summed across cards.
Manual shared-context override retained, separate codeContextTokens added; user
overrides capped to reported model max, and memory estimate uncertainty surfaced.
App ceiling1M, output reserve separate, research context/reply stays bounded.
Snapshot once per turn; authenticated GET /api/model-context previews estimates.
Context selection recorded as a safe Activity event. Conservative byte upper-bound
preflight rejects oversize prompts before provider truncation; it may reject
prompts some tokenizers could fit. No automatic model/hardware fit guarantee.

Attachments: immutable private text + bounded metadata sidecar (ID/name/size/SHA).
New Message.attachments references; original file content no longer appended to
user text. Metadata replay only across all providers; old inline dumps omitted
from model input without changing saved transcript; old files must be reattached
to recover exact references. Imports strip attachment references. Read by bounded
Unicode offsets/pagination; literal search returns bounded excerpts; exact copy
to selected writable project supports full1MB upload without LLM rewriting.
Checks account/private root, safe paths, source digest, native project identity
and existing write boundaries; rejects changed source and read-only scopes.
An attached explicit host/build/edit request from Chat enters managed General
Code project with writes allowed by the request; existing read-only Code retained;
Plain Chat returns explicit tools-disabled error instead. Uploads validated before
project setup. Attachment conversations suppress public web research. Native app
reviews still required. No duplicate catalogue/schema stuffing in initial prompt.

AUI adds file cards and separate Code context input plus estimate summary. main.tsx
loads budget separately from profile, handles backend Chat-to-Code session event,
and renders attachment links. Existing basic/advanced settings UI retained.

## Verification

App suite passed, full ./... with real bundled SearXNG passed, go vet passed.
Added tests for memory/model/manual clamping; actual generation wire budgets and
oversize rejection; metadata-only all-provider replay; Unicode pagination/search;
account/path/source-digest boundaries; exact300KB copy; authenticated HTTP turn
from Chat attachment to managed General Code, two actual mocked model calls,
bounded prompt, exact copied file and native Git sync invocation. Last app suite
after HTTP regression passed. Full Go + vet should rerun once final edits freeze.
No real cloud requests, owner inference/deployment, or physical-device acceptance.

Production AUI/tsc/Vite build passes. Synthetic UI server ui-fixture.cjs on
127.0.0.1:39720 (process receipt in artifacts). Cua browser3 contextUiTab585734819
rendered context/model limits and fields correctly; save test then lost debugger
before dispatch. Later completed successful Save of distinct Chat32K/Code96K, reload/reopen persistence, defaults Save/reset and another reload. Serializer/reset/invalid/zero unit checks pass. Saved desktop and390px screenshots show final controls/sticky Save; viewport reset. Read-only
DOM confirmed current default estimates and separate fields. Do not claim full
browser acceptance. Final production JS index-B0veHvrq, CSS index-gfmkwG4Q.
Need finish UI/serialization tests, context effective-value summary, docs and final
source review. Helper Node unit tests can validate serializer without browser.

## Publication/install pending

No preview19 version bump, source commit/push, tag, package, release/channel or
installation yet. Primary manager109/MyAI18 last read, update transaction complete.
Another task is publishing suite110 Git/Hosting (tooling/hosting-git-20261102);
coordinate before any maintenance, retain its pins. Use normal app-only backed-up
updater after public19/channel promotion, preserve configuration/credentials/models
and other apps. Prior18 source handoff has valid controller/install procedure and
pending old recovery cleanup caveat. Do not reapply an active update or bypass it.
Suite releases require signed Android entry point; this app-only release needs no
manager release. Actions disabled; never replace public18 assets.

Documented sources fetched: OpenAI file-inputs and tools-file-search (supports both
direct context inputs and retrieval), Claude Files API (storage/file IDs do not
remove context/token costs), Gemini file-search (chunks/index/retrieve), Ollama
context-length (64K for coding when memory allows), /api/ps and show-model-details.
Do not claim exact unpublished ChatGPT internals or that all uploads use retrieval.

## Final validation and source freeze

Final backend ./... with bundled SearXNG and vet pass after HTTP integration test,
independent Code profile persistence, paginated project reads and targeted1MiB
edits. Exact300KB Unicode copy, tail inspection/edit, denied writes/digest changes
pass. Final production AUI/TypeScript/Vite build and npm production audit pass.
Numeric settings Save/reset acceptance used only synthetic loopback API fixture.
No actual inference/model loading or owner deployment test; memory fitting remains
an estimate. Uploaded data stays account-private (UI corrected earlier My Drive
wording). Docs/MODEL-SETTINGS.md records limits, migration and official references.
Source versions bumped to19; native build/security check in progress. Generated
line-ending-only noise is excluded from staging. Primary now suite110/MyAI18,
transaction complete; peer is updating secondary. No primary maintenance submitted.

Final release gate found reachable GO-2026-6860 in pre-existing x/net0.61; upgraded
only x/net to0.62, checksum verified. Full tests/SearXNG/vet rerun pass; production
npm audit0; govulncheck0 reachable/package vulnerabilities (one unused required
module advisory). Native19 executable built with icon/trimpath. Its fresh isolated
NTFS standalone setup/login/profile/file roundtrip and18-to19 same-data restart
upgrade passed, including old profile and legacy attachment retention. Actual
managed suite fresh setup was not rerun; existing nine-payload contract unchanged.
Sanitized acceptance is docs/context-files19-acceptance.json. Candidate native SHA
cfa599fe9acdaf068144cc7fc169feb5de3655f1df3ce8d15c46043180923669.
