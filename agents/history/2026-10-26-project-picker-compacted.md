# Code project picker and General fallback

User requests a visible select/create project option in Code, automatic General
when none is selected, and push/release/install on both PCs. Applied repository
history and AUI. Source branch codex/my-ai-tools continues clean preview10
handoff at9b92627. Preserve all app/account/model state and normal protected
update behavior; never use live Accounts/Drive as functional fixtures.

## Implementation

Composer has a project button and New project at phone/desktop widths. Existing
name-only My Drive/private Git setup is reused. Choosing/creating keeps an unsent
draft; changing projects after a conversation starts opens a new conversation.
Server session save assigns General to unselected Code; old empty-project Code
sessions also recover on their next turn with revision/running rechecks.
General uses a stable48hex account-derived ID, broker authorization/storage/Git,
and serialized setup. Existing ready General is read without syncing unfinished
edits from another conversation. Failed setup preserves the draft and sends no
model request. Chat and explicit projects are unchanged; write permissions and
project turn exclusion remain enforced. General is private per account.

## Verification and delivery state

Go all-package tests including real bundled SearXNG/public search and vet pass.
New tests cover concurrent retries, stable IDs/restart, distinct accounts,
no resync on reuse, setup failure, explicit selection and Chat. AUI/tsc/Vite pass.
Scans:0 reachable/imported Go findings,1unused required-module advisory;
production npm0. No dependencies/search runtime changed.

Fresh synthetic Workspace92 + real Accounts/Drive/Git + real qwen3.5:4b Code
request sent with no project selected created General, wrote15,098byte Snake
HTML, completed and pushed. Remote bare Git HEAD and file bytes equal the
registry/Drive. Named project creation and switching retained the unsent draft
at effective434px phone viewport without horizontal overflow. Final UI polish
also updates the picker immediately after session save and fixes focus target;
final candidate reload checks passed. Final real qwen3:0.6b calculation turn
reused the same General after restart, retained Preview only, and returned42;
picker updated immediately after server assignment. Phone screenshot timed out;
desktop capture works. No live owner functional turn/cloud/full Codex shell,
clean-host/service/reboot certification.

Artifacts I:/Projects/tooling/my-ai-project-picker-20261026. Synthetic fixture
root C:/Projects/.cache/my-ai-20261024/p11; program payloads under artifacts;
recorded manager PID in fixture-process.json. Clean these only after final
checks. Test roots are private NTFS, do not delete global model libraries.

Preview11 publication/install pending. Authorized coordination peer confirms
no active apply/promotion, primary manager92/MyAI10/27healthy; secondary last
92/MyAI10/27healthy, Photos6. Suite checkout workspace-my-ai fast-forwarded
to8199f65. Reconcile current dynamic installed versions before guarded normal
MyAI-only applies; use released92 updater, no protection bypass. Promote public
channel after app publication and retain every unrelated release/withholding.
Do not repeat active applies during tunnel downtime. Need clean source freeze,
hash-matched immutable11/public asset verification, channel/pin promotion,
both-PC installs/audit, fixture cleanup and final handoff/docs push.

## Public and applying

Public11 source2c314f580075735751191a9f6b717e9e6381281c; clean EXE matches final
candidate24891dac5a39efd5396942606863562dcdc4d4b507722fcbf676207cfe704523.
All8asset digests/anonymous executable verified. Suite promotionf17c1ff pushed;
anonymous default raw channel verifies all11asset sizes/digests. Previous release
records/withholdings/other pins retained, future MyAI pin/default11, manager92.
Task fixture services/PID and both named C private roots guarded/stopped/removed.
Phone screenshot timeout remains; effective434px DOM/draft checks and desktop
picker screenshot passed. Final actual calculation turn reused General after
restart and retained Preview only; receipts final-ui-acceptance/native-acceptance.

BOTH NORMAL MYAI-ONLY UPDATES STARTED ONCE near20:47UTC26Oct. Primary guarded
immutable-public92 SDK helper --check-only passed exact10->11/public testedhash,
manager/all-other-contracts/packages/selection retained. Actual apply in CLI
session39500; primary-install11.log/primary-delivery11.json. Secondary browser
tab585734373 refreshed92 public channel; exactly1checkbox MyAI10->11,43MB,
normal backed-up Apply submitted once. Save baseline secondary-before-versions11.
Do not repeat either job during outages. Need observe both completions, primary
exactbinary/state/otherapps/27health/assets/models/latestrecovery, secondary
freshcheck11/unchangedotherrows/27health/assets, finaldocs/history push.

Primary completed: exact tested11EXE and unchanged manager92 hash
b1da5df7234ff9b1e743c6b40f0ac9c47964848d227eca66da201255bd64e889.
All other executable bytes/contracts/packages, configuration and decrypted
identities/credentials retained.28catalogapps/27running, served JS/CSS exact,
all current model names/sizes retained. Private bundled SearXNG/Python3.13.18
running loopback-only with anonymous404. Normal68.9MB/13file recovery verified;
kept update-20261026T204619-8bdaca69423a. Normal cleanup retired1older eligible
update copy (~175.1MB), no latest recovery/full-data removal. Primary job39500
finished0. Secondary original585734373 apply still in reconnect/startup downtime;
no replay. Need wait/verify secondary, then finaldocs/handoffs push.

## Both-PC delivery complete

Secondary normal only-MyAI43MB apply completed4m44s. All9stages complete and
27services healthy; new verified recovery update-20261026T204730-a528b2575c37
kept. Normal cleanup retired1older eligible copy(~149.9MB), preserving latest
and full-data backups. Fresh92 updater shows MyAI11current, all25otherdisplayed
groups exactly retained (Drive group5/Photos6, Tube5). Independent browser
health confirms27running/MyAI11, deployed JS index-CwhGQycp.js and CSS
index-C0x_bnkt.css match final content-addressed build names. No independent
remote disk/credential/JS-byte audit claimed. Original job submitted once;
no recovery/reapply used through4min tunnel outage.

Primary earlier audit remains valid: exacttested11/manager92/allotherappbytes,
ownerconfig/decryptedidentities retained;27running/assets exact/private search
healthy/all18models retained. Both latest recoveries kept. Authorized peer
notified of both completions; no active apply remains and coordination hold is
released. Public11 immutable, promotionf17c1ff, source2c314f5; no more binary or
channel mutation needed. Final acceptance docs/project-choice-acceptance.json.
Task test services and named private C roots already cleaned; browser viewport
reset and task tabs closed after evidence. Desktop picker proof and secondary
completion/installed screenshots remain outside Git under artifact root.

No required implementation/release/install work remains. Final handoff and
acceptance documentation commit does not move the immutable release tag or
replace public assets. Refresh MyAI to load the visible picker; sending Code
with no project selected uses the account's General automatically.
