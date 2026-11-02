# Project build/deployment workflow20 — in progress

User reports a follow-up deploy request still returns capability disclaimers and
manual Git/Hosting instructions instead of building/saving/deploying a project.
Standing chat authorization covers push/release and primary install; secondary
still pending. Use synthetic NTFS accounts/data only, no owner functional fixtures.

Public baseline19 source21d7643 (release4eada57), primary110/MyAI19. Original dirty
I:/Projects/my-ai at3033f3d and earlier I WIPs preserved; final19 I worktree clean.
New authoritative source C:/Projects/.cache/my-ai-20261024/project20-src, branch
codex/project-workflows from21d7643, Git common I:/Projects/my-ai/.git. Artifacts
I:/Projects/tooling/my-ai-project-workflows-20261102. C native/temp roots use20
prefix; existing19 roots remain after policy rejected cleanup; do not bypass that
rejection via another shell/API. No new publication/install yet.

Cause:19 only promoted Chat to managed Code when the same turn had an attachment.
Short follow-ups had no writable project; no execution-completion guard rejected
model refusals. New current-user directive router prepares General for build/code
and deployment requests regardless of attachments, preserves explicit read-only
Code and Plain Chat. Source/history alone grants no mutation authority.

New app-owned static publication: prepare selected project, materialize a complete
legacy HTML artifact exactly when needed, checkpoint/push Git, create/bind a native
static Hosting site, submit exact commit, track real job/URL. Pure deployment skips
LLM/provider keys/context probes entirely. Build-and-deploy leaves file generation
to model but owns Hosting submission; named native mutations hidden from model for
that turn. Guard retries no-action refusals once, then fails explicitly. Complete
HTML returned as text can become an exact project file. Package.json projects need
an already configured native site; raw framework sources never silently published.

Account-private Publications mapping persists project/site/job/commit and pending
markers before external mutation. Existing canonical call/result history recovers
older site associations. A later uncertain submission blocks falling back to an
older receipt or duplicate mutation. Known declines clear pending. Native SSO,
reviews, project/identity/path guards and manager idempotency remain enforced.
Fresh Git checkpoint before cached-job reuse prevents ignoring changed source.

Complete HTML in older/current conversation text is captured as an immutable
account-private file reference, preserving saved body. Model replay replaces only
its exact digest-matched document span; file tools read bounded ranges. This is
not semantic RAG or arbitrary source-language parsing. Existing imports remain
untrusted and are not upgraded into native tool receipts. Attachment card UI reused.

App tests pass after these changes: authenticated deploy-it Chat→General exact
300KB HTML/Git/Hosting review/job flow; retry/source-change/decline/ambiguous-result,
canonical receipt migration, no model/cloud key dependency; refusal retry guard;
private inline-document reference accounting. Mock native bridge in these tests;
actual native acceptance still needed. UI build passed unchanged content-addressed
assets B0veHvrq/gfmkwG4Q. Versions bumped20. Full suite/security/native build next.
Native fresh Workspace110 setup planned from verified public cached setup/bundle,
separate profile/roots/ports. Never replace owner program/data as fixture.

## Native candidate acceptance

Public110 cached setup verified against its published SHA and installed normally,
offline/headless, five selected apps, dedicated NTFS private/files/profile roots
and task-owned program root. Correction: its embedded MyAI baseline is14, not18.
Development candidate20 and its checked contract replaced only those two fixture
payloads with catalogue digests updated. No owner fixture. Real normal MyAccounts,
Workspace, Git and MyAI OAuth sign-in passed. Empty Chat with no model/cloud key
and an uploaded HTML file prepared General, pushed Git, reviewed native create
and deployment, published through actual Hosting/Caddy. git show of the exact
pushed commit and actual served bytes equal upload; Chrome rendered/click worked.
A second deploy-only follow-up made no Hosting mutations. Native receipt/source
SHA is in docs/project-workflows20-acceptance.json. No model inference used.

Harness initially omitted Origin on Casdoor login, then hit Python's default100
header limit while Workspace cleared catalogue cookies; fixed only fixture client
headers/limit. Initial approval JSON used accept instead of decision and aborted
before submission. Found/fixed a real lifecycle edge: pending markers now persist
after successful review, immediately before native mutation, so stopped review
remains retryable. Test-only unsubmitted pending record cleared after proving no
mutation; candidate rebuilt. REST Git read via browser cookie was not supported;
verified actual Git commit bytes using the fixture's own native Git metadata/CLI.
Normal app APIs/security guards not weakened for any harness issue. Own runtime
stopped normally after proof; loopback Caddy preview9080 and five app ports tracked.
Future fixture runs should rebase Caddy to a separate port before startup. The
public20 normal14→20 managed upgrade and primary19→20 install remain pending.
