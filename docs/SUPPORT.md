# Capability and acceptance matrix

| Capability | Preview status |
|---|---|
| Original Codex-inspired layout, rail, project/session sidebar, composer, effort popover, dark/light and mobile drawer | Implemented; native desktop/tablet/phone browser checks |
| Provider discovery and real streamed chat | Ollama demonstrated; OpenAI Responses and DeepSeek adapters tested with protocol fixtures; live API keys required for provider acceptance |
| Time, search and basic tools | Real clock/timezone, safe arithmetic, included SearXNG web search/page reading, automatic conversational search without a mode selector, natural-language web opt-out, private conversation/attachment lookup and own managed project reads; bounded provider function loops, explicit-command fallbacks, persisted activity and clickable sources |
| Bundled MCP | Native stdio public read-only tool server, automatically configured in isolated Codex sessions; no account/filesystem/shell access through MCP |
| Local model settings | Per-account/model context, output, sampling, batch and memory controls; low-memory preset, Plain chat and defaults reset. Synthetic persistence/wire/isolation and phone/desktop production UI tests pass. |
| Automatic Workspace app sign-in | Cold native sessions use normal MyAccounts SSO and exact request resume; read-only identity verification, separate app-change review, cancellation and mismatch denial tested with synthetic fixtures. |
| Per-session provider/model, per-response attribution, explicit cloud indicator | Implemented; native Ollama persistence demonstrated |
| MyAccounts PKCE and stable issuer+subject | Existing apphost contract reused; real fresh-install OIDC/PKCE acceptance passed |
| My Drive authoritative storage | Default-deny Quantum source, isolated physical user scopes, guard before access; real same-user Quantum download, second-browser reopen and cross-user denial passed |
| Model picker and server downloads | Local first; collapsed/filterable cloud lists. Persistent shared server queue, byte progress, browser-independent execution, restart resume, owner cancel/retry, truthful upstream errors and real Ollama integration tests |
| Codex app-server | Real stdio handshake/model metadata demonstrated on0.157.1; thread/turn/resume/events/approvals/interrupt integrated; thread profile and MCP transport checked; legacy Windows unelevated backend refuses restricted profiles; actual edit/test turn requires live key and explicit MXC/elevated sandbox setup/acceptance |
| Native project editing | Provider-native bounded file tools; real local Ollama edit/commit/push demonstrated. OpenAI/DeepSeek transport fixtures pass; live cloud turns need configured keys. No shell/test runner. |
| Projects and Git | Name-only creation, private per-account My Drive/Git, initial push, automatic completed-turn commit/push, safe incoming fast-forward, retry and no-force conflict refusal; actual native Git/OIDC/Drive/cross-user checks |
| Code project choice | Visible composer picker and New project on desktop/phone; unsent drafts retained; automatic private General project when none selected, stable across tabs/restarts; existing empty-project Code sessions recover on their next turn |
| Sessions | Search titles, pin/rename/archive/restore, revision conflict checks, JSON/Markdown export, JSON import, transcript fork, retry, Stop |
| Attachments | UTF-8 text/code up to1MB; stored in private Drive scope; image/document/audio inference unavailable |
| Markdown/code/tables | Safe React Markdown and GFM, raw HTML disabled; syntax colouring unavailable |
| Credentials | Per-user Windows DPAPI; no credential projection to Drive, exports or frontend settings responses |
| Recovery | Atomic current/previous version, interrupted response retention, inherited encrypted backups; automatic corrupted-file restoration and offline queue remain unavailable |
| Packaging | Embedded frontend and one native MyAI.exe; Ollama and optional legacy Codex remain separate processes |
| Advanced features | Shared model entitlements, billing budgets, automatic GPU sizing/catalog, image/voice services, multi-model comparisons, remembered approval rules and arbitrary custom endpoints unavailable |
| Deployment certification | Unsigned evaluation build; no clean-host, Windows-service, reboot or production certification |

Unavailable features are not presented as working. Do not equate a handshake with a successful model turn. Supplied reference screenshots were used; no protected desktop internals or live Codex UI were inspected. Existing live owner MyAccounts/Drive are never functional test fixtures.

Managed preview5 project checks include real native services, real local LLM file
edits and remote Git commits, phone/desktop layout, My Drive download, ordinary
member creation, physical/Git cross-user denial, incoming Git changes, competing
change preservation, retry/idempotence, credential exclusion and temporary-token
recovery. Current receipts identify the exact tested candidates. No live cloud,
full Codex command, clean-host or reboot test is implied.

## Local model loading errors

My AI distinguishes Ollama model-format/version failures, insufficient memory, missing models and oversized conversations from cloud account errors. Both HTTP and streamed failures use safe fixed messages; upstream error bodies are never displayed. Models lists show the installed Ollama runtime version when available. Download completion confirms stored files, not that every model fits or is supported by the installed runtime.

For newer community model formats, update Ollama on the server using its official installer, then retry the same downloaded model. Existing models stay in place. Bonsai2 testing targets official Ollama 0.40.11; a basic isolated CPU test also passed on 0.40.7. A successful pull on an older runtime does not establish inference compatibility.

Preview7 native checks: included patched Python3.13.18 and SearXNG return live documentation sources; anonymous runtime access is refused. Official Codex discovers all four public MCP tools in an isolated home without a provider key/global login. Chat and managed-project forced-search/privacy/failure regression tests pass. Discovery is not a full Codex model or shell turn.

# Model advisor: find a model for this machine

Open **My AI → Models → Test this PC**. The server owner starts or stops a test;
everyone signed in can see the last result. Download candidate models in Models
first. The test measures the machine hosting My AI, including its current
Workspace apps, rather than the phone or browser used to open it.

Leave Workspace and its normal apps running. Finish downloads, chats, backups
and updates first. The full test runs 12 synthetic file-editing, tool-use and
Git/Hosting workflows twice for each installed candidate, one model at a time.
The optional six-task scan is preliminary. Default selection includes up to eight
installed models, each at most 12 GiB; advanced users can name exact installed
tags. No default model or preference is changed automatically.

Testing continues on the server if you navigate away. Return to Models for live
progress and the result, or download its evidence ZIP. A server restart marks an
unfinished test interrupted; it does not automatically resume a load test.
Starting normal local My AI work or model maintenance stops the test first.

## Keeping room for Workspace

Before loading each model, the advisor measures current available memory and CPU,
optional NVIDIA VRAM and utilisation, and the local Workspace manager's health.
It reserves at least 4 GiB or 25% of system RAM, whichever is larger, plus at
least 1 GiB or 20% of each measured GPU's VRAM. An optional larger RAM reserve
can increase this margin. Existing app usage is already subtracted from the
available memory; the reserve is additional room for Windows and app activity.

Model bytes plus conservative overhead must fit beyond that reserve. A busy CPU,
unhealthy Workspace, or missing RAM/CPU telemetry prevents loading. Unknown GPU
telemetry or multiple GPUs uses a CPU-only test, subject to sufficient RAM. The
profile uses 8,192 context tokens, batch 32, at most four CPU threads (and at most
half the available logical CPUs), with short quiet intervals between rounds.

During inference, sampled monitoring stops testing if RAM/VRAM falls below the
reserve, telemetry disappears, Workspace fails or becomes slow, or CPU pressure
persists. The benchmark worker has lower process priority and its own time limit.
Only the test's inference is stopped and its model is asked to unload.

These are estimates and sampled protection, **not an operating-system memory or
GPU quota**. A brief resource spike can occur between samples. Run under a
representative normal workload and rerun after significant app, hardware, model
or context changes. Larger coding context windows and concurrent users need
additional memory and are not certified by an 8K test. The reported recommendation
applies to this tested profile, not every possible future server load.

## Understanding the result

A recommendation needs a complete run within the observed resource limits, at
least 90% task success and deployment success, and every guardrail check passed.
The UI reports minimum free RAM/VRAM, peak sampled CPU and slowest Workspace health
response. An incomplete, stopped or skipped model cannot qualify. No candidate
may qualify; that is a useful result rather than a reason to recommend an unsafe
or unreliable model.

Inference uses real installed model bytes. Tool actions use fresh synthetic
in-memory files and simulated Git/Hosting receipts. They do not deploy user apps,
edit personal files, execute arbitrary shell commands or use cloud credentials.
This small benchmark does not measure visual quality or arbitrary programming.
Model parser, quantization, prompt and hardware differences affect the ranking.

## Standalone runner

The benchmark source in `tools/agent-bench` also runs with Python 3.10+ and local
Ollama on Windows, Linux and macOS, without third-party Python dependencies:

```powershell
python safe_bench.py --out machine-test-001 --models qwen3.5:4b qwen3.5:9b
```

Use a fresh output directory each time. Optional `--health` accepts the local
Workspace `/healthz` URL, `--reserve-gb` increases the RAM reserve, and `--quick`
runs the preliminary scan. Endpoints must be local HTTP. GPU telemetry currently
supports NVIDIA; unsupported GPUs fall back to CPU-only. The integrated app
ships its verified Python runtime on supported Windows machines; users do not
install Python separately. Tests export only synthetic evidence and resource
metadata, kept in My AI's private app data.
