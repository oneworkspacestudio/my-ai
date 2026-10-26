# My AI Workspace app tools (in progress)

User requests My AI discover and use installed One Workspace apps, specifically
My Hosting for deployment, Git for source and Drive for storage. Started from
clean bd8ded7 after preview11 delivery (both machines on manager92/MyAI11).
Use the existing repository instructions, native account boundaries and AUI.

Uncommitted implementation: three discover/describe/use tools in the native
chat/Code loop, authenticated loopback manager bridge, ephemeral current-browser
capability (never serialized), conversation/account-scoped action reviews,
Tools & skills app discovery cards. Private app context persists as a boolean
so a later model turn cannot send app history to web tools. Public research
uses only the existing separately filtered public input; Workspace requests are
excluded. Public stdio MCP remains the four public tools, with no private app
authority. Legacy full Codex transport is not yet integrated.

Matching suite worktree workspace-my-ai has a registered native action gateway.
It reuses only the current browser's existing native app cookies, verifies real
issuer/subject, obtains native CSRF and preserves app permissions. No session
minting, recovery-key adoption or arbitrary URL proxy. Managed Git/Drive storage
and immutable already-pushed Git source handoff to Hosting reuse existing project
boundaries. Hosting receipts reconcile uncertain responses; queued work is never
reported as published. Native app roles still apply after My AI review.

Initial protocol tests passed (account separation, review replay/cancel, no public
MCP authority); AUI/tsc/Vite passed before latest backend privacy changes. More
native API adapters and tests are being added. Real native fixture and model
acceptance, full regression/vet, final documentation, release/install have NOT
been performed for this feature. Current app version still11. Do not publish
over immutable11 or claim general native acceptance from mocked protocol tests.
Artifacts: I:/Projects/tooling/my-ai-workspace-apps-20261026. checks.py runs
targeted or --all Go checks; synthetic private fixtures must stay under named
C:/Projects/.cache/my-ai-20261024, never live owner Accounts/Drive. No new fixture
has yet been started. Continue implementation/verification before publication.

## Current implementation and acceptance progress

Suite worktree fast-forwarded to0926501 (Tasks4 promotion); preserve its pins and
all parallel releases. Current feature remains entirely uncommitted/unpublished;
versions still11/92, candidates use dev.workspace-apps names.

Structured app calls now take top-level app/operation and parameters/query/body
objects, with complete callExamples and bodySchema returned by describe. Read
operations accept an empty JSON object and send no HTTP body. Describe is required
before using that app in each turn. Manager validates native body shapes before
review (including actual Tasks priority/status limits), resolves task list IDs,
and native apps still validate permissions and business rules. A decline blocks
all later app changes in that turn and ends the action loop. Identical errors stop
after3 attempts. Installed app summaries are supplied to the main model each turn.
Native Ollama app workflows use16k context instead of host default65k; this reduced
observed VRAM from5.65GB to3.65GB. Source-side private context now persists for app
requests and all project/conversation/attachment reads, including imported history;
session settings cannot reset it. Public research retains its filtered isolation.

UI now has plain-language app activity/reviews, submitting buttons are disabled,
and MyAI's own built-in tools are labelled accurately. AUI/tsc/Vite latest pass;
JS index-fEuav_bV.js, CSS index-C0x_bnkt.css (not final release assets).
OpenAI Docs official function-calling page fetched; dynamic native body uses
explicit strict:false, fixed public tools stay strict. No cloud keys/tests used.

Real fresh native fixture exists at C:/Projects/.cache/my-ai-20261024/p12;
programs under artifact root/fixture-programs. Portal localhost39600, Caddy site
port39680. Accounts/Drive/Git/Hosting/Tasks/Notes/MyAI all run. Native account
sign-in/SSO and app discovery verified in Chrome. Native Hosting imported its
verified official cached Caddy2.11.7 ZIP through normal UI; ready on isolated port.
No owner Accounts/Drive used. Three qwen3.5:4b task attempts exposed: initial
escaped-JSON contract loop, empty GET-body rejection, then skipped discovery and
invalid priority5. None created a task; third was declined and stopped. Its old
candidate re-prompted after decline, motivating the new hard decline guard. Those
issues have source fixes/tests but a successful new real-model task is still due.

Latest targeted app tests (including discovery requirement, same-user review,
decline/replay, structured arguments/privacy/public-MCP isolation) pass. Earlier
full MyAI tests with real bundled SearXNG passed before the most recent receipt/
preflight additions; rerun final checks. Suite targeted tests pass, including native
Tube SQLite mapping, immutable Git archive/credential refusal, body validation and
deployment receipts. Receipts now use server-derived per-user/conversation/turn
action IDs, source pending pointers, timestamped reconciliation and aliases; a
later explicit turn can redeploy the same commit. Unknown outcomes never trigger
duplicate source uploads; native definite validation/auth rejections can retry in
a later turn. Native Hosting1 ZIP API itself has no idempotency header support.

Full suite tests originally failed on11 existing Mail/updater fixtures. An untouched
0926501 archive reproduced every failure (suite-baseline.log). Updated test-only
fixtures to current shared-Mail role probe and manager91/92 metadata recovery,
added old-controller rejection coverage and assertions preserving current libraries
and the previous full-data copy. Full suite passed after those test repairs, before
latest connector additions. No production recovery or Mail behavior weakened.

Artifacts include checks.py, build.py, start-suite.py, restart-fixture.py,
baseline-check.py, polish-ui.py. Track current fixture PID in fixture-process.json
(restarts change it); do not replay start-suite.py. Latest build.py is running;
after it completes restart only the named fixture with restart-fixture.py, reload
all test app tabs to cache native sessions, then repeat the real task, Notes/Drive
write, Code/Git saved source and Hosting deployment/status/actual-page checks.
Need further final Go/vet/AUI/scans, source review, compatibility/version bumps,
release/install if continuing prior delivery scope, and guarded fixture cleanup.
Do not claim native app actions succeeded yet or publish from this partial state.

## Native acceptance update

The latest qwen3.5:4b fixture completed a reviewed Tasks creation, independently
visible in native Tasks Inbox, and a reviewed Notes creation, independently
visible in native Notes. Exactly one of each synthetic acceptance item exists.
Full MyAI Go tests including bundled search and the suite's full Go tests passed.
Drive revealed an actual trailing-slash registry route bug: connector constructed
/drive//api/users and treated native307 as a connection issue. Normalize route
prefixes for both native calls and connection probes; added a direct HTTP regression
for routes with and without trailing slash and already-prefixed/native paths.
Rebuilding candidates/targeted suite tests for this correction. Native Drive and
Git-to-Hosting deployment acceptance still unfinished; no publication/install yet.


## Completed native acceptance and release preparation

Real qwen3.5:4b Code created General automatically, wrote index.html and pushed
it to private native Git. Registry HEAD and remote Git file bytes match Drive.
My AI then reviewed/created a static Hosting site, reviewed the exact pushed
revision, submitted it and polled its actual job to published. The returned local
preview page was opened and rendered the expected heading. Native Tasks, Notes
and Drive mutations are independently visible in their apps. Acceptance receipt
is docs/workspace-apps-acceptance.json in MyAI and artifact native-acceptance.json.
All tests used only the named synthetic fixture; no live owner app fixtures.

Installed app names now route from the actual catalogue and preload their native
action descriptions, covering future apps without a fixed one-scenario list.
A small local model still made invalid/irrelevant calls before recovering; do not
claim perfect model instruction-following or certify every registered app. Native
body validation, permissions and reviews remain authoritative. Mixed public web
research/private app saves should use separate turns; legacy full Codex and cloud
providers are not live-certified. Mail is read-only, unsupported apps open-only.

Final full Go tests in both repos passed, including bundled SearXNG runtime16.1s.
Both go vet ./... passed; AUI/TypeScript/Vite passed; npm production audit0. Go
vulnerability scan found no reachable vulnerable call paths (module advisory
GO-2026-5932 is reported for the existing crypto dependency; do not call it zero
advisories). Cleanup's bytes-freed assertion now uses actual stale-stage bytes:
hard-linked backup removal can correctly free0. Production updater unchanged.

Final versions prepared: MyAI12 requires manager93 in compatibility and managed
metadata. Candidate binaries built. Final fixture restart/version/persistence
check, clean source commits, immutable packaging, publication/channel promotion,
both protected owner installs and fixture cleanup remain. Preserve current main
b76ccf3 Maps5/Tasks4 promotion and unrelated pins. Authorized coordination peer
reports no overlapping update or manager93 reservation; secondary needs a fresh
transaction check before applying.


## Final privacy correction / delivery versions

Final review caught raw prior-user fallback in public research: a short follow-up
could refer to a dynamically recognised app request not matched by fixed filters.
When private Workspace context exists, the researcher now receives no raw message
history; only its separately verified public ResearchContext may resolve a follow-up.
A regression ensures a private app request followed by "tell me more" never starts
public research. Full app tests including bundled runtime13.6s and both vets passed.

App12 source e96a8fb and immutable public assets were already published, but never
promoted or installed. Explicitly withhold12; do not alter/delete its tag or assets.
Corrected app13 requires manager93. Manager93 remains unpublished/uncommitted.
Core native acceptance is unchanged; final13 candidate health and persisted data
verified after synthetic restart. Phone viewport override did not change the Chrome
extension viewport (still1794px), so mobile-specific rendering is NOT certified.
Desktop app cards render correctly. Continue app13 packaging/publication, suite93
source/package, channel generation (exclude12), protected installs and cleanup.
