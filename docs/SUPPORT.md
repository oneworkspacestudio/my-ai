# Capability and acceptance matrix

| Capability | Preview status |
|---|---|
| Original Codex-inspired layout, rail, project/session sidebar, composer, effort popover, dark/light and mobile drawer | Implemented; native desktop/tablet/phone browser checks |
| Provider discovery and real streamed chat | Ollama demonstrated; OpenAI Responses and DeepSeek adapters tested with protocol fixtures; live API keys required for provider acceptance |
| Time, search and basic tools | Real clock/timezone, safe arithmetic, live public web search/page reading, private conversation/attachment lookup and own managed project reads; bounded provider function loops, explicit-command fallbacks, persisted activity and clickable sources |
| Bundled MCP | Native stdio public read-only tool server, automatically configured in isolated Codex sessions; no account/filesystem/shell access through MCP |
| Per-session provider/model, per-response attribution, explicit cloud indicator | Implemented; native Ollama persistence demonstrated |
| MyAccounts PKCE and stable issuer+subject | Existing apphost contract reused; real fresh-install OIDC/PKCE acceptance passed |
| My Drive authoritative storage | Default-deny Quantum source, isolated physical user scopes, guard before access; real same-user Quantum download, second-browser reopen and cross-user denial passed |
| Model picker and server downloads | Local first; collapsed/filterable cloud lists. Persistent shared server queue, byte progress, browser-independent execution, restart resume, owner cancel/retry, truthful upstream errors and real Ollama integration tests |
| Codex app-server | Real stdio handshake/model metadata demonstrated on0.157.1; thread/turn/resume/events/approvals/interrupt integrated; thread profile and MCP transport checked; legacy Windows unelevated backend refuses restricted profiles; actual edit/test turn requires live key and explicit MXC/elevated sandbox setup/acceptance |
| Native project editing | Provider-native bounded file tools; real local Ollama edit/commit/push demonstrated. OpenAI/DeepSeek transport fixtures pass; live cloud turns need configured keys. No shell/test runner. |
| Projects and Git | Name-only creation, private per-account My Drive/Git, initial push, automatic completed-turn commit/push, safe incoming fast-forward, retry and no-force conflict refusal; actual native Git/OIDC/Drive/cross-user checks |
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
